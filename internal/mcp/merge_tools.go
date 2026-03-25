package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"kanbanzai/internal/config"
	"kanbanzai/internal/git"
	"kanbanzai/internal/github"
	"kanbanzai/internal/merge"
	"kanbanzai/internal/model"
	"kanbanzai/internal/service"
	"kanbanzai/internal/worktree"
)

// MergeTools returns all merge-related MCP tool definitions with their handlers.
func MergeTools(
	worktreeStore *worktree.Store,
	entitySvc *service.EntityService,
	repoPath string,
	thresholds git.BranchThresholds,
	localConfig *config.LocalConfig,
) []server.ServerTool {
	return []server.ServerTool{
		mergeReadinessCheckTool(worktreeStore, entitySvc, repoPath, thresholds, localConfig),
		mergeExecuteTool(worktreeStore, entitySvc, repoPath, thresholds, localConfig),
	}
}

func mergeReadinessCheckTool(
	worktreeStore *worktree.Store,
	entitySvc *service.EntityService,
	repoPath string,
	thresholds git.BranchThresholds,
	localConfig *config.LocalConfig,
) server.ServerTool {
	tool := mcp.NewTool("merge_readiness_check",
		mcp.WithDescription("Check if an entity (feature or bug) is ready to merge. Evaluates all merge gates and optionally checks PR status if GitHub is configured."),
		mcp.WithString("entity_id", mcp.Description("Entity ID (FEAT-... or BUG-...)"), mcp.Required()),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		entityID, err := request.RequireString("entity_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		result, err := checkMergeReadiness(worktreeStore, entitySvc, repoPath, thresholds, localConfig, entityID)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("merge readiness check failed", err), nil
		}

		return mergeMapJSON(result)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func mergeExecuteTool(
	worktreeStore *worktree.Store,
	entitySvc *service.EntityService,
	repoPath string,
	thresholds git.BranchThresholds,
	localConfig *config.LocalConfig,
) server.ServerTool {
	tool := mcp.NewTool("merge_execute",
		mcp.WithDescription("Execute a merge for an entity after verifying all gates pass. Use override with reason to bypass blocking gates."),
		mcp.WithString("entity_id", mcp.Description("Entity ID (FEAT-... or BUG-...)"), mcp.Required()),
		mcp.WithBoolean("override", mcp.Description("Override blocking gates (default: false)")),
		mcp.WithString("override_reason", mcp.Description("Required explanation when override is true")),
		mcp.WithString("merge_strategy", mcp.Description("Merge strategy: squash, merge, or rebase (default: squash)")),
		mcp.WithBoolean("delete_branch", mcp.Description("Delete branch after merge (default: true)")),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		entityID, err := request.RequireString("entity_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		override := request.GetBool("override", false)
		overrideReason := request.GetString("override_reason", "")
		strategyStr := request.GetString("merge_strategy", "squash")
		deleteBranch := request.GetBool("delete_branch", true)

		// Validate override reason
		if override && overrideReason == "" {
			return mcp.NewToolResultError("OVERRIDE_REASON_REQUIRED: override_reason is required when override is true"), nil
		}

		// Parse merge strategy
		strategy, err := parseMergeStrategy(strategyStr)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		result, err := executeMerge(worktreeStore, entitySvc, repoPath, thresholds, localConfig, entityID, override, overrideReason, strategy, deleteBranch)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("merge execute failed", err), nil
		}

		return mergeMapJSON(result)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

// checkMergeReadiness performs a merge readiness check for an entity.
func checkMergeReadiness(
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
			return nil, fmt.Errorf("NO_WORKTREE: no worktree found for entity %s", entityID)
		}
		return nil, err
	}

	// Get the entity
	entityType := entityTypeFromID(entityID)
	if entityType == "" {
		return nil, fmt.Errorf("invalid entity ID: must start with FEAT- or BUG-")
	}
	entity, err := entitySvc.Get(entityType, entityID, "")
	if err != nil {
		return nil, fmt.Errorf("get entity: %w", err)
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

	// Build response
	resp := map[string]any{
		"entity_id":      entityID,
		"branch":         wt.Branch,
		"overall_status": gateResult.OverallStatus,
	}

	// Convert gate results
	gates := make([]map[string]any, 0, len(gateResult.Gates))
	for _, g := range gateResult.Gates {
		gate := map[string]any{
			"name":     g.Name,
			"status":   string(g.Status),
			"severity": string(g.Severity),
		}
		if g.Message != "" {
			gate["message"] = g.Message
		}
		gates = append(gates, gate)
	}
	resp["gates"] = gates

	// Check PR status if GitHub is configured
	if localConfig != nil && localConfig.GetGitHubToken() != "" {
		prStatus, err := getPRStatus(repoPath, wt.Branch, localConfig)
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
			return nil, fmt.Errorf("NO_WORKTREE: no worktree found for entity %s", entityID)
		}
		return nil, err
	}

	// Get the entity
	entityType := entityTypeFromID(entityID)
	if entityType == "" {
		return nil, fmt.Errorf("invalid entity ID: must start with FEAT- or BUG-")
	}
	entity, err := entitySvc.Get(entityType, entityID, "")
	if err != nil {
		return nil, fmt.Errorf("get entity: %w", err)
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
		return nil, fmt.Errorf("GATES_FAILED: %v", msgs)
	}

	// Check for merge conflicts explicitly
	hasConflicts, err := git.HasMergeConflicts(repoPath, wt.Branch, "main")
	if err != nil {
		// Try master
		hasConflicts, err = git.HasMergeConflicts(repoPath, wt.Branch, "master")
	}
	if err == nil && hasConflicts {
		return nil, fmt.Errorf("MERGE_CONFLICT: branch %s has merge conflicts with main", wt.Branch)
	}

	// Perform the merge
	gitOps := worktree.NewGit(repoPath)

	// Checkout main/master
	if err := gitOps.CheckoutBranch("main"); err != nil {
		// Try master
		if err := gitOps.CheckoutBranch("master"); err != nil {
			return nil, fmt.Errorf("checkout base branch: %w", err)
		}
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
		return nil, fmt.Errorf("merge branch: %w", err)
	}

	mergedAt := time.Now().UTC()

	// Update worktree status
	cfg := config.LoadOrDefault()
	gracePeriodDays := cfg.Cleanup.GracePeriodDays
	if gracePeriodDays == 0 {
		gracePeriodDays = 7
	}

	wt.MarkMerged(mergedAt, gracePeriodDays)
	if _, err := worktreeStore.Update(wt); err != nil {
		// Log but don't fail - merge succeeded
		_ = err
	}

	// Delete branch if requested
	if deleteBranch {
		_ = gitOps.DeleteBranch(wt.Branch, false)
		// Also delete remote branch if configured
		if cfg.Cleanup.AutoDeleteRemoteBranch {
			_ = gitOps.DeleteRemoteBranch("origin", wt.Branch)
		}
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

	return resp, nil
}

// getPRStatus fetches PR status from GitHub for the given branch.
func getPRStatus(repoPath, branch string, localConfig *config.LocalConfig) (map[string]any, error) {
	token := localConfig.GetGitHubToken()
	if token == "" {
		return nil, fmt.Errorf("no GitHub token configured")
	}

	client := github.NewClient(token)

	// Detect repository info
	repoInfo, err := github.DetectRepo(repoPath, localConfig)
	if err != nil {
		return nil, err
	}

	// Get PR by branch
	pr, err := client.GetPRByBranch(repoInfo, branch)
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
		return "", fmt.Errorf("invalid merge_strategy: %s (must be squash, merge, or rebase)", s)
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

// mergeMapJSON marshals a map to JSON and returns it as a tool result.
func mergeMapJSON(v map[string]any) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("marshal result: " + err.Error()), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}
