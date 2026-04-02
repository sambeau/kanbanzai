package mcp

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/sambeau/kanbanzai/internal/config"
	"github.com/sambeau/kanbanzai/internal/git"
	"github.com/sambeau/kanbanzai/internal/github"
	"github.com/sambeau/kanbanzai/internal/merge"
	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/service"
	"github.com/sambeau/kanbanzai/internal/worktree"
)

// mergeCommitFunc is the function called after merge execute to commit
// the worktree state update. Package-level variable for test injection.
// Production value delegates to git.CommitStateWithMessage (FR-A09).
var mergeCommitFunc = func(repoRoot, message string) (bool, error) {
	return git.CommitStateWithMessage(repoRoot, message)
}

// MergeTool returns the 2.0 consolidated merge tool.
// It consolidates merge_readiness_check and merge_execute (spec §19.2).
func MergeTool(
	worktreeStore *worktree.Store,
	entitySvc *service.EntityService,
	repoPath string,
	thresholds git.BranchThresholds,
	localConfig *config.LocalConfig,
) []server.ServerTool {
	return []server.ServerTool{mergeTool(worktreeStore, entitySvc, repoPath, thresholds, localConfig)}
}

func mergeTool(
	worktreeStore *worktree.Store,
	entitySvc *service.EntityService,
	repoPath string,
	thresholds git.BranchThresholds,
	localConfig *config.LocalConfig,
) server.ServerTool {
	tool := mcp.NewTool("merge",
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithTitleAnnotation("Merge Gate & Execution"),
		mcp.WithDescription(
			"Use when a feature or bug is ready to be merged into the main branch. "+
				"Call check first to evaluate merge gates (CI status, review approvals, branch health, "+
				"task completion), then call execute to perform the merge. Do NOT merge directly via git — "+
				"merge enforces Kanbanzai workflow gates and records the merge in entity state. "+
				"Call AFTER all tasks are complete and pr(action: \"create\") has opened a pull request. "+
				"Actions: check (evaluate merge gates), execute (merge after gate verification). "+
				"For both actions: entity_id (FEAT-... or BUG-...) is required. "+
				"execute supports override + override_reason to bypass blocking gates when justified.",
		),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("Action: check, execute"),
		),
		mcp.WithString("entity_id",
			mcp.Required(),
			mcp.Description("Entity ID (FEAT-... or BUG-...)"),
		),
		// execute-only parameters
		mcp.WithBoolean("override",
			mcp.Description("Override blocking gates (execute only, default: false)"),
		),
		mcp.WithString("override_reason",
			mcp.Description("Required explanation when override is true (execute only)"),
		),
		mcp.WithString("merge_strategy",
			mcp.Description("Merge strategy: squash, merge, or rebase (execute only, default: squash)"),
		),
		mcp.WithBoolean("delete_branch",
			mcp.Description("Delete branch after merge (execute only, default: true)"),
		),
	)

	handler := WithSideEffects(func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		return DispatchAction(ctx, req, map[string]ActionHandler{
			"check":   mergeCheckAction(worktreeStore, entitySvc, repoPath, thresholds, localConfig),
			"execute": mergeExecuteAction(worktreeStore, entitySvc, repoPath, thresholds, localConfig),
		})
	})

	return server.ServerTool{Tool: tool, Handler: handler}
}

// ─── check ───────────────────────────────────────────────────────────────────

func mergeCheckAction(
	worktreeStore *worktree.Store,
	entitySvc *service.EntityService,
	repoPath string,
	thresholds git.BranchThresholds,
	localConfig *config.LocalConfig,
) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		entityID, err := req.RequireString("entity_id")
		if err != nil {
			return nil, fmt.Errorf("Cannot check merge readiness: entity_id is missing.\n\nTo resolve:\n  Provide entity_id: merge(action: \"check\", entity_id: \"FEAT-...\")")
		}

		result, err := checkMergeReadiness(ctx, worktreeStore, entitySvc, repoPath, thresholds, localConfig, entityID)
		if err != nil {
			return nil, err
		}

		return result, nil
	}
}

// ─── execute ─────────────────────────────────────────────────────────────────

func mergeExecuteAction(
	worktreeStore *worktree.Store,
	entitySvc *service.EntityService,
	repoPath string,
	thresholds git.BranchThresholds,
	localConfig *config.LocalConfig,
) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		SignalMutation(ctx)

		entityID, err := req.RequireString("entity_id")
		if err != nil {
			return nil, fmt.Errorf("Cannot execute merge: entity_id is missing.\n\nTo resolve:\n  Provide entity_id: merge(action: \"execute\", entity_id: \"FEAT-...\")")
		}

		override := req.GetBool("override", false)
		overrideReason := req.GetString("override_reason", "")
		strategyStr := req.GetString("merge_strategy", "squash")
		deleteBranch := req.GetBool("delete_branch", true)

		if override && overrideReason == "" {
			return inlineErr("missing_parameter", "override_reason is required when override is true")
		}

		strategy, err := parseMergeStrategy(strategyStr)
		if err != nil {
			return inlineErr("invalid_parameter", err.Error())
		}

		result, err := executeMerge(worktreeStore, entitySvc, repoPath, thresholds, localConfig, entityID, override, overrideReason, strategy, deleteBranch)
		if err != nil {
			return nil, err
		}

		// Post-merge install: rebuild binary and write install record.
		cfg := config.LoadOrDefault()
		if effect := postMergeInstall(ctx, repoPath, cfg); effect != nil {
			PushSideEffect(ctx, *effect)
		}

		return result, nil
	}
}

// checkMergeReadiness performs a merge readiness check for an entity.
func checkMergeReadiness(
	ctx context.Context,
	worktreeStore *worktree.Store,
	entitySvc *service.EntityService,
	repoPath string,
	thresholds git.BranchThresholds,
	localConfig *config.LocalConfig,
	entityID string,
) (map[string]any, error) {
	// Get the worktree for this entity
	wt, err := worktreeStore.GetByEntityID(entityID)
	if err != nil {
		if errors.Is(err, worktree.ErrNotFound) {
			return nil, fmt.Errorf("Cannot check merge readiness for %s: no worktree exists for this entity.\n\nTo resolve:\n  Create a worktree first: worktree(action: \"create\", entity_id: \"%s\")", entityID, entityID)
		}
		return nil, err
	}

	// Get the entity
	entityType := entityTypeFromID(entityID)
	if entityType == "" {
		return nil, fmt.Errorf("Cannot check merge readiness: entity ID %q is not a feature or bug.\n\nTo resolve:\n  Provide an entity ID starting with FEAT- or BUG-", entityID)
	}
	entity, err := entitySvc.Get(entityType, entityID, "")
	if err != nil {
		return nil, fmt.Errorf("Cannot check merge readiness for %s: failed to retrieve entity: %w.\n\nTo resolve:\n  Verify the entity ID exists: entity(action: \"get\", id: \"%s\")", entityID, err, entityID)
	}

	// Get child tasks if this is a feature
	var tasks []map[string]any
	if entityType == string(model.EntityKindFeature) {
		taskResults, err := entitySvc.ListEntitiesFiltered(service.ListFilteredInput{
			Type:   string(model.EntityKindTask),
			Parent: entityID,
		})
		if err == nil {
			for _, t := range taskResults {
				tasks = append(tasks, t.State)
			}
		}
	}

	// Build gate context
	gateCtx := merge.GateContext{
		RepoPath:   repoPath,
		EntityID:   entityID,
		Branch:     wt.Branch,
		Entity:     entity.State,
		Tasks:      tasks,
		Thresholds: thresholds,
	}

	// Check all gates
	gateResult := merge.CheckGates(gateCtx)

	// Build response using FormatGateResults, which includes entity_id, branch,
	// overall_status, per-gate results, and a summary with total/passed/failed/warning counts.
	resp := merge.FormatGateResults(gateResult)

	// Check PR status if GitHub is configured
	if localConfig != nil && localConfig.GetGitHubToken() != "" {
		prStatus, err := getPRStatus(ctx, repoPath, wt.Branch, localConfig)
		if err == nil && prStatus != nil {
			resp["pr_status"] = prStatus
		}
	}

	return resp, nil
}

// executeMerge performs the actual merge operation.
func executeMerge(
	worktreeStore *worktree.Store,
	entitySvc *service.EntityService,
	repoPath string,
	thresholds git.BranchThresholds,
	localConfig *config.LocalConfig,
	entityID string,
	override bool,
	overrideReason string,
	strategy worktree.MergeStrategy,
	deleteBranch bool,
) (map[string]any, error) {
	// Get the worktree for this entity
	wt, err := worktreeStore.GetByEntityID(entityID)
	if err != nil {
		if errors.Is(err, worktree.ErrNotFound) {
			return nil, fmt.Errorf("Cannot execute merge for %s: no worktree exists for this entity.\n\nTo resolve:\n  Create a worktree first: worktree(action: \"create\", entity_id: \"%s\")", entityID, entityID)
		}
		return nil, err
	}

	// Get the entity
	entityType := entityTypeFromID(entityID)
	if entityType == "" {
		return nil, fmt.Errorf("Cannot execute merge: entity ID %q is not a feature or bug.\n\nTo resolve:\n  Provide an entity ID starting with FEAT- or BUG-", entityID)
	}
	entity, err := entitySvc.Get(entityType, entityID, "")
	if err != nil {
		return nil, fmt.Errorf("Cannot execute merge for %s: failed to retrieve entity: %w.\n\nTo resolve:\n  Verify the entity ID exists: entity(action: \"get\", id: \"%s\")", entityID, err, entityID)
	}

	// Get child tasks if this is a feature
	var tasks []map[string]any
	if entityType == string(model.EntityKindFeature) {
		taskResults, err := entitySvc.ListEntitiesFiltered(service.ListFilteredInput{
			Type:   string(model.EntityKindTask),
			Parent: entityID,
		})
		if err == nil {
			for _, t := range taskResults {
				tasks = append(tasks, t.State)
			}
		}
	}

	// Build gate context
	gateCtx := merge.GateContext{
		RepoPath:   repoPath,
		EntityID:   entityID,
		Branch:     wt.Branch,
		Entity:     entity.State,
		Tasks:      tasks,
		Thresholds: thresholds,
	}

	// Check all gates
	gateResult := merge.CheckGates(gateCtx)

	// Check if merge is allowed
	if gateResult.OverallStatus == merge.OverallStatusBlocked && !override {
		failures := merge.BlockingFailures(gateResult.Gates)
		var msgs []string
		for _, f := range failures {
			msgs = append(msgs, f.Message)
		}
		return nil, fmt.Errorf("Cannot merge %s: blocking merge gates failed: %v.\n\nTo resolve:\n  Fix the failing gates shown above, or use override: true with an override_reason to bypass", entityID, msgs)
	}

	// Determine default branch once and use it throughout.
	defaultBranch, err := git.GetDefaultBranch(repoPath)
	if err != nil {
		return nil, fmt.Errorf("Cannot merge %s: failed to determine default branch: %w.\n\nTo resolve:\n  Ensure the repository has a valid default branch (main or master)", entityID, err)
	}

	// Check for merge conflicts explicitly before touching the working tree.
	hasConflicts, err := git.HasMergeConflicts(repoPath, wt.Branch, defaultBranch)
	if err != nil {
		return nil, fmt.Errorf("Cannot merge %s: failed to check for merge conflicts: %w.\n\nTo resolve:\n  Ensure the branch %s and default branch are both valid refs", entityID, err, wt.Branch)
	}
	if hasConflicts {
		return nil, fmt.Errorf("Cannot merge %s: branch %s has merge conflicts with %s.\n\nTo resolve:\n  Rebase or merge %s into %s and resolve conflicts before retrying", entityID, wt.Branch, defaultBranch, defaultBranch, wt.Branch)
	}

	// Perform the merge
	gitOps := worktree.NewGit(repoPath)

	// Checkout the default base branch.
	if err := gitOps.CheckoutBranch(defaultBranch); err != nil {
		return nil, fmt.Errorf("Cannot merge %s: failed to checkout base branch %s: %w.\n\nTo resolve:\n  Ensure the working tree is clean and the branch %s exists locally", entityID, defaultBranch, err, defaultBranch)
	}

	// Build merge message
	entityTitle := mergeStringFromState(entity.State, "title")
	if entityTitle == "" {
		entityTitle = mergeStringFromState(entity.State, "summary")
	}
	mergeMessage := fmt.Sprintf("Merge %s: %s", entityID, entityTitle)
	if override {
		mergeMessage += fmt.Sprintf("\n\nOverride reason: %s", overrideReason)
	}

	// Execute merge
	mergeResult, err := gitOps.MergeBranch(wt.Branch, strategy, mergeMessage)
	if err != nil {
		return nil, fmt.Errorf("Cannot merge %s: merge of branch %s failed: %w.\n\nTo resolve:\n  Check for conflicts or uncommitted changes, then retry", entityID, wt.Branch, err)
	}

	mergedAt := time.Now().UTC()

	// Update worktree status
	cfg := config.LoadOrDefault()
	gracePeriodDays := cfg.Cleanup.GracePeriodDays
	if gracePeriodDays == 0 {
		gracePeriodDays = 7
	}

	var warnings []string
	wt.MarkMerged(mergedAt, gracePeriodDays)
	if _, updateErr := worktreeStore.Update(wt); updateErr != nil {
		// Don't fail — the merge already succeeded. Surface as a warning so the
		// caller knows the worktree record is stale (it won't appear in cleanup lists).
		warnings = append(warnings, fmt.Sprintf("failed to update worktree record after merge: %v", updateErr))
	}

	// Delete branch if requested
	if deleteBranch {
		_ = gitOps.DeleteBranch(wt.Branch, false)
		// Also delete remote branch if configured
		if cfg.Cleanup.AutoDeleteRemoteBranch {
			_ = gitOps.DeleteRemoteBranch("origin", wt.Branch)
		}
	}

	// Auto-commit the worktree record update after merge and branch cleanup (FR-A09).
	// Best-effort: commit failure is logged but does not prevent the merge result.
	mergeStateMsg := fmt.Sprintf("workflow(%s): mark worktree merged", entityID)
	if _, commitErr := mergeCommitFunc(repoPath, mergeStateMsg); commitErr != nil {
		log.Printf("[merge] WARNING: auto-commit after merge of %s failed: %v", entityID, commitErr)
	}

	// Record override events when blocking gates were bypassed.
	var overrideRecords []map[string]any
	if override {
		blockingFailures := merge.BlockingFailures(gateResult.Gates)
		overriddenBy, _ := config.ResolveIdentity("")
		if overriddenBy == "" {
			overriddenBy = "unknown"
		}
		overrideReq := merge.OverrideRequest{
			EntityID:     entityID,
			Reason:       overrideReason,
			OverriddenBy: overriddenBy,
		}
		ovrs := merge.CreateOverrides(overrideReq, blockingFailures, mergedAt)
		overrideRecords = merge.FormatOverrides(ovrs)
	}

	// Build response
	resp := map[string]any{
		"merged": map[string]any{
			"entity_id":    entityID,
			"branch":       wt.Branch,
			"merge_commit": mergeResult.MergeCommit,
			"merged_at":    mergedAt.Format(time.RFC3339),
		},
	}

	if wt.CleanupAfter != nil {
		resp["cleanup_scheduled"] = map[string]any{
			"cleanup_after": wt.CleanupAfter.Format(time.RFC3339),
		}
	}

	if len(overrideRecords) > 0 {
		resp["overrides"] = overrideRecords
	}

	if len(warnings) > 0 {
		resp["warnings"] = warnings
	}

	return resp, nil
}

// getPRStatus fetches PR status from GitHub for the given branch.
func getPRStatus(ctx context.Context, repoPath, branch string, localConfig *config.LocalConfig) (map[string]any, error) {
	token := localConfig.GetGitHubToken()
	if token == "" {
		return nil, fmt.Errorf("Cannot fetch PR status: no GitHub token configured.\n\nTo resolve:\n  Add a GitHub token to .kbz/local.yaml: github_token: \"ghp_...\"")
	}

	client := github.NewClient(token)

	// Detect repository info
	repoInfo, err := github.DetectRepo(repoPath, localConfig)
	if err != nil {
		return nil, err
	}

	// Get PR by branch
	pr, err := client.GetPRByBranch(ctx, repoInfo, branch)
	if err != nil {
		if errors.Is(err, github.ErrPRNotFound) {
			return nil, nil // No PR exists
		}
		return nil, err
	}

	result := map[string]any{
		"url":           pr.URL,
		"ci_status":     pr.CIStatus,
		"review_status": pr.ReviewStatus,
		"has_conflicts": pr.HasConflicts,
	}

	return result, nil
}

// parseMergeStrategy parses a merge strategy string.
func parseMergeStrategy(s string) (worktree.MergeStrategy, error) {
	switch s {
	case "squash", "":
		return worktree.MergeStrategySquash, nil
	case "merge":
		return worktree.MergeStrategyMerge, nil
	case "rebase":
		return worktree.MergeStrategyRebase, nil
	default:
		return "", fmt.Errorf("Cannot execute merge: invalid merge_strategy %q.\n\nTo resolve:\n  Use one of: squash, merge, or rebase", s)
	}
}

// mergeStringFromState extracts a string value from a state map.
func mergeStringFromState(state map[string]any, key string) string {
	if state == nil {
		return ""
	}
	v, ok := state[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Sprint(v)
	}
	return s
}
