package mcp

import (
	"context"
	"encoding/json"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"kanbanzai/internal/cleanup"
	"kanbanzai/internal/config"
	"kanbanzai/internal/worktree"
)

// CleanupTools returns all cleanup-related MCP tool definitions with their handlers.
func CleanupTools(store *worktree.Store, git *worktree.Git, cfg *config.CleanupConfig) []server.ServerTool {
	return []server.ServerTool{
		cleanupListTool(store),
		cleanupExecuteTool(store, git, cfg),
	}
}

func cleanupListTool(store *worktree.Store) server.ServerTool {
	tool := mcp.NewTool("cleanup_list",
		mcp.WithDescription("List worktrees pending cleanup. Shows merged and abandoned worktrees that are either ready for cleanup (past grace period) or scheduled (within grace period)."),
		mcp.WithBoolean("include_pending", mcp.Description("Include items past grace period that are ready for cleanup (default: true)")),
		mcp.WithBoolean("include_scheduled", mcp.Description("Include items within grace period that are scheduled for future cleanup (default: true)")),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		includePending := request.GetBool("include_pending", true)
		includeScheduled := request.GetBool("include_scheduled", true)

		records, err := store.List()
		if err != nil {
			return mcp.NewToolResultErrorFromErr("list worktrees failed", err), nil
		}

		now := time.Now()
		opts := cleanup.ListOptions{
			IncludePending:   includePending,
			IncludeScheduled: includeScheduled,
			IncludeAbandoned: includePending, // Abandoned items are also "pending"
		}

		items := cleanup.ListCleanupItems(records, now, opts)

		// Separate into pending and scheduled
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

		resp := map[string]any{
			"success":           true,
			"pending_cleanup":   pendingItems,
			"scheduled_cleanup": scheduledItems,
		}

		return cleanupMapJSON(resp)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func cleanupExecuteTool(store *worktree.Store, git *worktree.Git, cfg *config.CleanupConfig) server.ServerTool {
	tool := mcp.NewTool("cleanup_execute",
		mcp.WithDescription("Execute cleanup on worktrees. Removes worktree directories, deletes local branches, optionally deletes remote branches, and removes tracking records."),
		mcp.WithString("worktree_id", mcp.Description("Specific worktree ID to clean up (e.g., WT-01JX...). If omitted, cleans all ready items.")),
		mcp.WithBoolean("dry_run", mcp.Description("If true, simulates cleanup without making changes (default: false)")),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		worktreeID := request.GetString("worktree_id", "")
		dryRun := request.GetBool("dry_run", false)

		opts := cleanup.CleanupOptions{
			DryRun:             dryRun,
			DeleteRemoteBranch: cfg.AutoDeleteRemoteBranch,
		}

		var results []cleanup.CleanupResult

		if worktreeID != "" {
			// Clean specific worktree
			record, err := store.Get(worktreeID)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("get worktree failed", err), nil
			}

			// Check if ready for cleanup
			now := time.Now()
			if !cleanup.IsReadyForCleanup(&record, now) && record.Status != worktree.StatusAbandoned {
				return mcp.NewToolResultError("worktree is not ready for cleanup (still within grace period)"), nil
			}

			// Force remove for abandoned worktrees
			if record.Status == worktree.StatusAbandoned {
				opts.ForceRemove = true
			}

			result := cleanup.ExecuteCleanup(store, git, record, opts)
			results = append(results, result)
		} else {
			// Clean all ready items
			results = cleanup.ExecuteAllReady(store, git, opts)
		}

		// Build response
		var cleaned []map[string]any
		var errors []map[string]any

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
				errors = append(errors, entry)
			}
		}

		resp := map[string]any{
			"success": len(errors) == 0,
			"dry_run": dryRun,
			"cleaned": cleaned,
		}

		if len(errors) > 0 {
			resp["errors"] = errors
		}

		if dryRun && len(cleaned) > 0 {
			resp["message"] = "Dry run: no changes made. The listed items would be cleaned."
		}

		return cleanupMapJSON(resp)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

// cleanupMapJSON marshals a map to JSON and returns it as a tool result.
func cleanupMapJSON(v map[string]any) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("marshal result: " + err.Error()), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}
