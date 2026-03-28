package mcp

import (
	"context"
	"errors"
	"fmt"

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

// PRTool returns the 2.0 consolidated pr tool.
// It consolidates pr_create, pr_status, and pr_update into a single tool (spec §19.3).
func PRTool(
	worktreeStore *worktree.Store,
	entitySvc *service.EntityService,
	repoPath string,
	thresholds git.BranchThresholds,
	localConfig *config.LocalConfig,
) []server.ServerTool {
	return []server.ServerTool{prTool(worktreeStore, entitySvc, repoPath, thresholds, localConfig)}
}

func prTool(
	worktreeStore *worktree.Store,
	entitySvc *service.EntityService,
	repoPath string,
	thresholds git.BranchThresholds,
	localConfig *config.LocalConfig,
) server.ServerTool {
	tool := mcp.NewTool("pr",
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithTitleAnnotation("Pull Request Manager"),
		mcp.WithDescription(
			"Create and manage GitHub pull requests for feature/bug entities. "+
				"Consolidates pr_create, pr_status, and pr_update. "+
				"Actions: create (open a new PR), status (get CI/review status), "+
				"update (refresh description and labels). "+
				"Requires GitHub token in .kbz/local.yaml.",
		),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("Action: create, status, update"),
		),
		mcp.WithString("entity_id",
			mcp.Required(),
			mcp.Description("Entity ID (FEAT-... or BUG-...)"),
		),
		// create-only
		mcp.WithBoolean("draft",
			mcp.Description("Create as draft PR (create only, default: false)"),
		),
	)

	handler := WithSideEffects(func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		return DispatchAction(ctx, req, map[string]ActionHandler{
			"create": prCreateAction(worktreeStore, entitySvc, repoPath, localConfig),
			"status": prStatusAction(worktreeStore, repoPath, localConfig),
			"update": prUpdateAction(worktreeStore, entitySvc, repoPath, thresholds, localConfig),
		})
	})

	return server.ServerTool{Tool: tool, Handler: handler}
}

// ─── create ──────────────────────────────────────────────────────────────────

func prCreateAction(
	worktreeStore *worktree.Store,
	entitySvc *service.EntityService,
	repoPath string,
	localConfig *config.LocalConfig,
) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		SignalMutation(ctx)

		entityID, err := req.RequireString("entity_id")
		if err != nil {
			return nil, fmt.Errorf("entity_id is required for create action")
		}
		draft := req.GetBool("draft", false)

		result, err := createPR(ctx, worktreeStore, entitySvc, repoPath, localConfig, entityID, draft)
		if err != nil {
			return nil, err
		}

		return result, nil
	}
}

// ─── status ──────────────────────────────────────────────────────────────────

func prStatusAction(
	worktreeStore *worktree.Store,
	repoPath string,
	localConfig *config.LocalConfig,
) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		entityID, err := req.RequireString("entity_id")
		if err != nil {
			return nil, fmt.Errorf("entity_id is required for status action")
		}

		result, err := getPRStatusForEntity(ctx, worktreeStore, repoPath, localConfig, entityID)
		if err != nil {
			return nil, err
		}

		return result, nil
	}
}

// ─── update ──────────────────────────────────────────────────────────────────

func prUpdateAction(
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
			return nil, fmt.Errorf("entity_id is required for update action")
		}

		result, err := updatePR(ctx, worktreeStore, entitySvc, repoPath, thresholds, localConfig, entityID)
		if err != nil {
			return nil, err
		}

		return result, nil
	}
}

// createPR creates a new pull request for an entity.
func createPR(
	ctx context.Context,
	worktreeStore *worktree.Store,
	entitySvc *service.EntityService,
	repoPath string,
	localConfig *config.LocalConfig,
	entityID string,
	draft bool,
) (map[string]any, error) {
	// Check GitHub configuration
	if localConfig == nil || localConfig.GetGitHubToken() == "" {
		return nil, fmt.Errorf("GITHUB_NOT_CONFIGURED: GitHub token not configured in .kbz/local.yaml")
	}

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

	client := github.NewClient(localConfig.GetGitHubToken())

	// Detect repository info
	repoInfo, err := github.DetectRepo(repoPath, localConfig)
	if err != nil {
		return nil, fmt.Errorf("detect repository: %w", err)
	}

	// Check if PR already exists
	existingPR, err := client.GetPRByBranch(ctx, repoInfo, wt.Branch)
	if err == nil && existingPR != nil {
		return nil, fmt.Errorf("PR_EXISTS: PR already exists for branch %s: %s", wt.Branch, existingPR.URL)
	}
	if err != nil && !errors.Is(err, github.ErrPRNotFound) {
		return nil, fmt.Errorf("check existing PR: %w", err)
	}

	// Build PR title and body
	entityTitle := prStringFromState(entity.State, "title")
	if entityTitle == "" {
		entityTitle = prStringFromState(entity.State, "summary")
	}
	prTitle := fmt.Sprintf("%s: %s", entityID, entityTitle)

	// Get tasks for description
	var tasks []github.TaskData
	if entityType == string(model.EntityKindFeature) {
		taskResults, err := entitySvc.ListEntitiesFiltered(service.ListFilteredInput{
			Type:   string(model.EntityKindTask),
			Parent: entityID,
		})
		if err == nil {
			for _, t := range taskResults {
				tasks = append(tasks, github.TaskData{
					ID:     prStringFromState(t.State, "id"),
					Title:  prStringFromState(t.State, "title"),
					Status: prStringFromState(t.State, "status"),
				})
			}
		}
	}

	descData := github.DescriptionData{
		EntityID:           entityID,
		EntityTitle:        entityTitle,
		EntityDescription:  prStringFromState(entity.State, "description"),
		EntityType:         entity.Type,
		Tasks:              tasks,
		Verification:       prStringFromState(entity.State, "verification"),
		VerificationStatus: prStringFromState(entity.State, "verification_status"),
		Created:            prStringFromState(entity.State, "created"),
		Branch:             wt.Branch,
	}
	prBody := github.GenerateDescription(descData)

	// Ensure standard labels exist
	var warnings []string
	if err := client.EnsureStandardLabels(ctx, repoInfo); err != nil {
		warnings = append(warnings, fmt.Sprintf("label setup: %v", err))
	}

	// Create the PR
	baseBranch, baseErr := git.GetDefaultBranch(repoPath)
	if baseErr != nil {
		baseBranch = "main" // fallback for repos where git detection might not work
	}
	pr, err := client.CreatePR(ctx, repoInfo, wt.Branch, baseBranch, prTitle, prBody, draft)
	if err != nil {
		return nil, fmt.Errorf("create PR: %w", err)
	}

	// Set initial labels
	labels := github.ComputeLabels(entityType, false, false, false)
	if len(labels) > 0 {
		if err := client.SetPRLabels(ctx, repoInfo, pr.Number, labels); err != nil {
			warnings = append(warnings, fmt.Sprintf("set labels: %v", err))
		}
	}

	resp := map[string]any{
		"pr": map[string]any{
			"url":    pr.URL,
			"number": pr.Number,
			"title":  pr.Title,
			"state":  pr.State,
			"draft":  pr.Draft,
		},
	}
	if len(warnings) > 0 {
		resp["warnings"] = warnings
	}
	return resp, nil
}

// updatePR updates an existing pull request's description and labels.
func updatePR(
	ctx context.Context,
	worktreeStore *worktree.Store,
	entitySvc *service.EntityService,
	repoPath string,
	thresholds git.BranchThresholds,
	localConfig *config.LocalConfig,
	entityID string,
) (map[string]any, error) {
	// Check GitHub configuration
	if localConfig == nil || localConfig.GetGitHubToken() == "" {
		return nil, fmt.Errorf("GITHUB_NOT_CONFIGURED: GitHub token not configured in .kbz/local.yaml")
	}

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

	client := github.NewClient(localConfig.GetGitHubToken())

	// Detect repository info
	repoInfo, err := github.DetectRepo(repoPath, localConfig)
	if err != nil {
		return nil, fmt.Errorf("detect repository: %w", err)
	}

	// Get existing PR
	pr, err := client.GetPRByBranch(ctx, repoInfo, wt.Branch)
	if err != nil {
		if errors.Is(err, github.ErrPRNotFound) {
			return nil, fmt.Errorf("NO_PR: no PR found for branch %s", wt.Branch)
		}
		return nil, fmt.Errorf("get PR: %w", err)
	}

	// Build updated title and body
	entityTitle := prStringFromState(entity.State, "title")
	if entityTitle == "" {
		entityTitle = prStringFromState(entity.State, "summary")
	}
	prTitle := fmt.Sprintf("%s: %s", entityID, entityTitle)

	// Get tasks for description
	var tasks []github.TaskData
	tasksComplete := true
	if entityType == string(model.EntityKindFeature) {
		taskResults, err := entitySvc.ListEntitiesFiltered(service.ListFilteredInput{
			Type:   string(model.EntityKindTask),
			Parent: entityID,
		})
		if err == nil {
			for _, t := range taskResults {
				status := prStringFromState(t.State, "status")
				tasks = append(tasks, github.TaskData{
					ID:     prStringFromState(t.State, "id"),
					Title:  prStringFromState(t.State, "title"),
					Status: status,
				})
				if status != "done" && status != "wont_do" {
					tasksComplete = false
				}
			}
		}
	} else {
		// For bugs, no tasks to check
		tasksComplete = true
	}

	descData := github.DescriptionData{
		EntityID:           entityID,
		EntityTitle:        entityTitle,
		EntityDescription:  prStringFromState(entity.State, "description"),
		EntityType:         entityType,
		Tasks:              tasks,
		Verification:       prStringFromState(entity.State, "verification"),
		VerificationStatus: prStringFromState(entity.State, "verification_status"),
		Created:            prStringFromState(entity.State, "created"),
		Branch:             wt.Branch,
	}
	prBody := github.GenerateDescription(descData)

	// Track changes made
	var changes []string

	// Update PR
	_, err = client.UpdatePR(ctx, repoInfo, pr.Number, prTitle, prBody)
	if err != nil {
		return nil, fmt.Errorf("update PR: %w", err)
	}
	changes = append(changes, "Updated description")

	// Check verification status
	verificationPassed := prStringFromState(entity.State, "verification_status") == "passed"

	// Check merge gates
	gatesPass := false
	gateCtx := merge.GateContext{
		RepoPath:   repoPath,
		EntityID:   entityID,
		Branch:     wt.Branch,
		Entity:     entity.State,
		Thresholds: thresholds,
	}
	if entityType == string(model.EntityKindFeature) {
		var taskStates []map[string]any
		for _, t := range tasks {
			taskStates = append(taskStates, map[string]any{
				"id":     t.ID,
				"title":  t.Title,
				"status": t.Status,
			})
		}
		gateCtx.Tasks = taskStates
	}
	gateResult := merge.CheckGates(gateCtx)
	gatesPass = gateResult.OverallStatus == merge.OverallStatusPassed

	// Update labels
	labels := github.ComputeLabels(entityType, tasksComplete, verificationPassed, gatesPass)
	if err := client.SetPRLabels(ctx, repoInfo, pr.Number, labels); err == nil {
		for _, label := range labels {
			changes = append(changes, fmt.Sprintf("Added label: %s", label))
		}
	}

	return map[string]any{
		"pr": map[string]any{
			"url":     pr.URL,
			"updated": true,
			"changes": changes,
		},
	}, nil
}

// getPRStatusForEntity gets PR status for an entity.
func getPRStatusForEntity(
	ctx context.Context,
	worktreeStore *worktree.Store,
	repoPath string,
	localConfig *config.LocalConfig,
	entityID string,
) (map[string]any, error) {
	// Check GitHub configuration
	if localConfig == nil || localConfig.GetGitHubToken() == "" {
		return nil, fmt.Errorf("GITHUB_NOT_CONFIGURED: GitHub token not configured in .kbz/local.yaml")
	}

	// Get the worktree for this entity
	wt, err := worktreeStore.GetByEntityID(entityID)
	if err != nil {
		if errors.Is(err, worktree.ErrNotFound) {
			return nil, fmt.Errorf("NO_WORKTREE: no worktree found for entity %s", entityID)
		}
		return nil, err
	}

	client := github.NewClient(localConfig.GetGitHubToken())

	// Detect repository info
	repoInfo, err := github.DetectRepo(repoPath, localConfig)
	if err != nil {
		return nil, fmt.Errorf("detect repository: %w", err)
	}

	// Get PR by branch
	pr, err := client.GetPRByBranch(ctx, repoInfo, wt.Branch)
	if err != nil {
		if errors.Is(err, github.ErrPRNotFound) {
			return nil, fmt.Errorf("NO_PR: no PR found for branch %s", wt.Branch)
		}
		return nil, fmt.Errorf("get PR: %w", err)
	}

	// Build reviews list
	reviews := make([]map[string]any, 0, len(pr.Reviews))
	for _, r := range pr.Reviews {
		reviews = append(reviews, map[string]any{
			"user":  r.User,
			"state": r.State,
		})
	}

	// Determine CI status
	ciStatus := pr.CIStatus
	if ciStatus == "" {
		ciStatus = "none"
	}

	// Determine review status
	reviewStatus := pr.ReviewStatus
	if reviewStatus == "" {
		reviewStatus = "none"
	}

	return map[string]any{
		"pr": map[string]any{
			"url":           pr.URL,
			"number":        pr.Number,
			"state":         pr.State,
			"draft":         pr.Draft,
			"ci_status":     ciStatus,
			"review_status": reviewStatus,
			"reviews":       reviews,
			"has_conflicts": pr.HasConflicts,
			"mergeable":     pr.Mergeable,
		},
	}, nil
}

// prStringFromState extracts a string value from a state map.
func prStringFromState(state map[string]any, key string) string {
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
