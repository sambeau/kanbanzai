package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"kanbanzai/internal/cleanup"
	"kanbanzai/internal/config"
	"kanbanzai/internal/worktree"
)

// CleanupTool returns the 2.0 consolidated cleanup tool.
// It consolidates cleanup_list and cleanup_execute into a single tool (spec §19.5).
func CleanupTool(store *worktree.Store, git *worktree.Git, cfg *config.CleanupConfig) []server.ServerTool {
	return []server.ServerTool{cleanupTool(store, git, cfg)}
}

func cleanupTool(store *worktree.Store, git *worktree.Git, cfg *config.CleanupConfig) server.ServerTool {
	tool := mcp.NewTool("cleanup",
		mcp.WithDescription(
			"Manage worktree cleanup. Lists worktrees pending cleanup and executes cleanup operations. "+
				"Consolidates cleanup_list and cleanup_execute. "+
				"Actions: list (show worktrees pending cleanup), execute (remove worktree directories, "+
				"delete branches, and remove tracking records).",
		),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("Action: list, execute"),
		),
		// list parameters
		mcp.WithBoolean("include_pending",
			mcp.Description("Include items past grace period that are ready for cleanup (default: true) — list only"),
		),
		mcp.WithBoolean("include_scheduled",
			mcp.Description("Include items within grace period that are scheduled for future cleanup (default: true) — list only"),
		),
		// execute parameters
		mcp.WithString("worktree_id",
			mcp.Description("Specific worktree ID to clean up (e.g., WT-01JX...). If omitted, cleans all ready items — execute only"),
		),
		mcp.WithBoolean("dry_run",
			mcp.Description("Simulate cleanup without making changes (default: false) — execute only"),
		),
	)

	handler := WithSideEffects(func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		return DispatchAction(ctx, req, map[string]ActionHandler{
			"list":    cleanupListAction(store),
			"execute": cleanupExecuteAction(store, git, cfg),
		})
	})

	return server.ServerTool{Tool: tool, Handler: handler}
}

// ─── list ─────────────────────────────────────────────────────────────────────

func cleanupListAction(store *worktree.Store) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		includePending := req.GetBool("include_pending", true)
		includeScheduled := req.GetBool("include_scheduled", true)

		records, err := store.List()
		if err != nil {
			return nil, fmt.Errorf("list worktrees: %w", err)
		}

		now := time.Now()
		opts := cleanup.ListOptions{
			IncludePending:   includePending,
			IncludeScheduled: includeScheduled,
			IncludeAbandoned: includePending,
		}

		items := cleanup.ListCleanupItems(records, now, opts)

		var pendingItems []map[string]any
		var scheduledItems []map[string]any

		for _, item := range items {
			entry := map[string]any{
				"worktree_id": item.WorktreeID,
				"entity_id":   item.EntityID,
				"branch":      item.Branch,
				"path":        item.Path,
				"status":      item.Status,
			}
			if !item.MergedAt.IsZero() {
				entry["merged_at"] = item.MergedAt.Format(time.RFC3339)
			}
			if !item.CleanupAfter.IsZero() {
				entry["cleanup_after"] = item.CleanupAfter.Format(time.RFC3339)
			}

			if item.Status == "ready" || item.Status == "abandoned" {
				pendingItems = append(pendingItems, entry)
			} else if item.Status == "scheduled" {
				scheduledItems = append(scheduledItems, entry)
			}
		}

		return map[string]any{
			"pending_cleanup":   pendingItems,
			"scheduled_cleanup": scheduledItems,
		}, nil
	}
}

// ─── execute ─────────────────────────────────────────────────────────────────

func cleanupExecuteAction(store *worktree.Store, git *worktree.Git, cfg *config.CleanupConfig) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		worktreeID := req.GetString("worktree_id", "")
		dryRun := req.GetBool("dry_run", false)

		if !dryRun {
			SignalMutation(ctx)
		}

		opts := cleanup.CleanupOptions{
			DryRun:             dryRun,
			DeleteRemoteBranch: cfg.AutoDeleteRemoteBranch,
		}

		var results []cleanup.CleanupResult

		if worktreeID != "" {
			record, err := store.Get(worktreeID)
			if err != nil {
				return nil, fmt.Errorf("get worktree %s: %w", worktreeID, err)
			}

			now := time.Now()
			if !cleanup.IsReadyForCleanup(&record, now) && record.Status != worktree.StatusAbandoned {
				return inlineErr("not_ready",
					"worktree is not ready for cleanup (still within grace period)")
			}

			if record.Status == worktree.StatusAbandoned {
				opts.ForceRemove = true
			}

			result := cleanup.ExecuteCleanup(store, git, record, opts)
			results = append(results, result)
		} else {
			results = cleanup.ExecuteAllReady(store, git, opts)
		}

		var cleaned []map[string]any
		var errs []map[string]any

		for _, r := range results {
			entry := map[string]any{
				"worktree_id":           r.WorktreeID,
				"branch":                r.Branch,
				"path":                  r.Path,
				"remote_branch_deleted": r.RemoteBranchDeleted,
			}
			if r.Success {
				cleaned = append(cleaned, entry)
			} else {
				if r.Error != nil {
					entry["error"] = r.Error.Error()
				}
				errs = append(errs, entry)
			}
		}

		resp := map[string]any{
			"dry_run": dryRun,
			"cleaned": cleaned,
		}
		if len(errs) > 0 {
			resp["errors"] = errs
		}
		if dryRun && len(cleaned) > 0 {
			resp["message"] = "Dry run: no changes made. The listed items would be cleaned."
		}

		return resp, nil
	}
}
