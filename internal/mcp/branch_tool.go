package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"kanbanzai/internal/git"
	"kanbanzai/internal/worktree"
)

// BranchTool returns the 2.0 consolidated branch tool.
// It consolidates branch_status into a single tool with an action parameter (spec §19.4).
func BranchTool(store *worktree.Store, repoPath string, thresholds git.BranchThresholds) []server.ServerTool {
	return []server.ServerTool{branchTool(store, repoPath, thresholds)}
}

func branchTool(store *worktree.Store, repoPath string, thresholds git.BranchThresholds) server.ServerTool {
	tool := mcp.NewTool("branch",
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithTitleAnnotation("Branch Health Monitor"),
		mcp.WithDescription(
			"Get branch health metrics for an entity's worktree branch. "+
				"Reports staleness, drift from main, and merge conflicts. "+
				"Consolidates branch_status. "+
				"Actions: status.",
		),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("Action: status"),
		),
		mcp.WithString("entity_id",
			mcp.Required(),
			mcp.Description("Entity ID (FEAT-... or BUG-...)"),
		),
	)

	// Branch status is read-only; no WithSideEffects wrapper needed (spec §8.5).
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		action, _ := args["action"].(string)
		if action == "" {
			return ActionError("missing_parameter", "action is required; valid actions: status", nil), nil
		}
		if action != "status" {
			return ActionError("unknown_action",
				fmt.Sprintf("unknown action %q; valid actions: status", action), nil), nil
		}

		return branchStatusAction(store, repoPath, thresholds, req)
	}

	return server.ServerTool{Tool: tool, Handler: handler}
}

// ─── status ───────────────────────────────────────────────────────────────────

func branchStatusAction(
	store *worktree.Store,
	repoPath string,
	thresholds git.BranchThresholds,
	req mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	entityID := req.GetString("entity_id", "")
	if entityID == "" {
		return ActionError("missing_parameter", "entity_id is required for status action", nil), nil
	}

	record, err := store.GetByEntityID(entityID)
	if err != nil {
		if isWorktreeNotFound(err) {
			return ActionError("no_worktree",
				fmt.Sprintf("no worktree found for entity %s", entityID), nil), nil
		}
		return ActionError("internal_error", fmt.Sprintf("get worktree: %v", err), nil), nil
	}

	status, err := git.EvaluateBranchStatus(repoPath, record.Branch, thresholds)
	if err != nil {
		if isBranchNotFound(err) {
			return ActionError("branch_not_found",
				fmt.Sprintf("branch %s does not exist", record.Branch), nil), nil
		}
		return ActionError("internal_error", fmt.Sprintf("evaluate branch status: %v", err), nil), nil
	}

	resp := map[string]any{
		"branch": record.Branch,
		"metrics": map[string]any{
			"branch_age_days":       status.Metrics.BranchAgeDays,
			"commits_behind_main":   status.Metrics.CommitsBehindMain,
			"commits_ahead_of_main": status.Metrics.CommitsAheadOfMain,
			"last_commit_at":        status.Metrics.LastCommitAt.Format(time.RFC3339),
			"last_commit_age_days":  status.Metrics.LastCommitAgeDays,
			"has_conflicts":         status.Metrics.HasConflicts,
		},
		"warnings": status.Warnings,
		"errors":   status.Errors,
	}

	return branchToolMapJSON(resp)
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// isWorktreeNotFound returns true if the error indicates a missing worktree record.
func isWorktreeNotFound(err error) bool {
	if err == nil {
		return false
	}
	return err == worktree.ErrNotFound
}

// isBranchNotFound returns true if the error indicates a missing git branch.
func isBranchNotFound(err error) bool {
	if err == nil {
		return false
	}
	return err == git.ErrBranchNotFound
}

// branchToolMapJSON marshals a map to JSON and returns it as a tool result.
func branchToolMapJSON(v map[string]any) (*mcp.CallToolResult, error) {
	return worktreeMapJSON(v)
}
