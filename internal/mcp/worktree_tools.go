package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"kanbanzai/internal/config"
	"kanbanzai/internal/git"
	"kanbanzai/internal/service"
	"kanbanzai/internal/worktree"
)

// WorktreeTools returns all worktree MCP tool definitions with their handlers.
func WorktreeTools(store *worktree.Store, entitySvc *service.EntityService, gitOps *worktree.Git) []server.ServerTool {
	return []server.ServerTool{
		worktreeCreateTool(store, entitySvc, gitOps),
		worktreeListTool(store),
		worktreeGetTool(store),
		worktreeRemoveTool(store, gitOps),
	}
}

// BranchTools returns all branch MCP tool definitions with their handlers.
func BranchTools(store *worktree.Store, repoPath string, thresholds git.BranchThresholds) []server.ServerTool {
	return []server.ServerTool{
		branchStatusTool(store, repoPath, thresholds),
	}
}

func worktreeCreateTool(store *worktree.Store, entitySvc *service.EntityService, gitOps *worktree.Git) server.ServerTool {
	tool := mcp.NewTool("worktree_create",
		mcp.WithDescription("Create a new Git worktree for a feature or bug entity. The worktree provides an isolated workspace for development."),
		mcp.WithString("entity_id", mcp.Description("Entity ID (FEAT-... or BUG-...)"), mcp.Required()),
		mcp.WithString("branch_name", mcp.Description("Custom branch name (auto-generated if omitted)")),
		mcp.WithString("created_by", mcp.Description("Who created the worktree. Auto-resolved from .kbz/local.yaml or git config if not provided.")),
		mcp.WithString("slug", mcp.Description("Human-readable slug for branch naming (extracted from entity if omitted)")),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		entityID, err := request.RequireString("entity_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Validate entity exists and is a valid type for worktrees
		entityType := entityTypeFromID(entityID)
		if entityType == "" {
			return mcp.NewToolResultError("INVALID_ENTITY_TYPE: entity ID must start with FEAT- or BUG-"), nil
		}

		entity, err := entitySvc.Get(entityType, entityID, "")
		if err != nil {
			if errors.Is(err, service.ErrNotFound) || strings.Contains(err.Error(), "no "+entityType+" entity found") {
				return mcp.NewToolResultError(fmt.Sprintf("ENTITY_NOT_FOUND: %s", entityID)), nil
			}
			return mcp.NewToolResultErrorFromErr("get entity failed", err), nil
		}

		// Check if worktree already exists for this entity
		existing, err := store.GetByEntityID(entityID)
		if err == nil && existing.ID != "" {
			return mcp.NewToolResultError(fmt.Sprintf("WORKTREE_EXISTS: worktree %s already exists for entity %s", existing.ID, entityID)), nil
		}

		// Resolve identity
		createdByRaw := request.GetString("created_by", "")
		createdBy, err := config.ResolveIdentity(createdByRaw)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Determine slug for naming
		slug := request.GetString("slug", "")
		if slug == "" {
			if s, ok := entity.State["slug"].(string); ok {
				slug = s
			}
		}

		// Generate or use provided branch name
		branchName := request.GetString("branch_name", "")
		if branchName == "" {
			branchName = worktree.GenerateBranchName(entityID, slug)
		}

		// Generate worktree path
		wtPath := worktree.GenerateWorktreePath(entityID, slug)

		// Create the git worktree with a new branch
		if err := gitOps.CreateWorktreeNewBranch(wtPath, branchName, ""); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("GIT_ERROR: %v", err)), nil
		}

		// Create the worktree record
		record := worktree.Record{
			EntityID:  entityID,
			Branch:    branchName,
			Path:      wtPath,
			Status:    worktree.StatusActive,
			Created:   time.Now().UTC(),
			CreatedBy: createdBy,
		}

		created, err := store.Create(record)
		if err != nil {
			// Try to clean up the git worktree if record creation fails
			_ = gitOps.RemoveWorktree(wtPath, true)
			return mcp.NewToolResultErrorFromErr("create worktree record failed", err), nil
		}

		resp := map[string]any{
			"success":  true,
			"worktree": worktreeRecordToMap(created),
		}
		return worktreeMapJSON(resp)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func worktreeListTool(store *worktree.Store) server.ServerTool {
	tool := mcp.NewTool("worktree_list",
		mcp.WithDescription("List all worktrees with optional filtering by status or entity."),
		mcp.WithString("status", mcp.Description("Filter by status: active, merged, abandoned, or all (default: all)")),
		mcp.WithString("entity_id", mcp.Description("Filter by entity ID")),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		statusFilter := request.GetString("status", "all")
		entityIDFilter := request.GetString("entity_id", "")

		records, err := store.List()
		if err != nil {
			return mcp.NewToolResultErrorFromErr("list worktrees failed", err), nil
		}

		// Apply filters
		var filtered []worktree.Record
		for _, r := range records {
			if entityIDFilter != "" && r.EntityID != entityIDFilter {
				continue
			}
			if statusFilter != "all" && string(r.Status) != statusFilter {
				continue
			}
			filtered = append(filtered, r)
		}

		worktrees := make([]map[string]any, 0, len(filtered))
		for _, r := range filtered {
			worktrees = append(worktrees, worktreeRecordToMap(r))
		}

		resp := map[string]any{
			"success":   true,
			"count":     len(worktrees),
			"worktrees": worktrees,
		}
		return worktreeMapJSON(resp)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func worktreeGetTool(store *worktree.Store) server.ServerTool {
	tool := mcp.NewTool("worktree_get",
		mcp.WithDescription("Get the worktree record for a specific entity."),
		mcp.WithString("entity_id", mcp.Description("Entity ID (FEAT-... or BUG-...)"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		entityID, err := request.RequireString("entity_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		record, err := store.GetByEntityID(entityID)
		if err != nil {
			if errors.Is(err, worktree.ErrNotFound) {
				return mcp.NewToolResultError(fmt.Sprintf("NO_WORKTREE: no worktree found for entity %s", entityID)), nil
			}
			return mcp.NewToolResultErrorFromErr("get worktree failed", err), nil
		}

		resp := map[string]any{
			"success":  true,
			"worktree": worktreeRecordToMap(record),
		}
		return worktreeMapJSON(resp)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func worktreeRemoveTool(store *worktree.Store, gitOps *worktree.Git) server.ServerTool {
	tool := mcp.NewTool("worktree_remove",
		mcp.WithDescription("Remove a worktree for an entity. By default, fails if there are uncommitted changes."),
		mcp.WithString("entity_id", mcp.Description("Entity ID (FEAT-... or BUG-...)"), mcp.Required()),
		mcp.WithBoolean("force", mcp.Description("If true, remove even with uncommitted changes (default: false)")),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		entityID, err := request.RequireString("entity_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		force := request.GetBool("force", false)

		record, err := store.GetByEntityID(entityID)
		if err != nil {
			if errors.Is(err, worktree.ErrNotFound) {
				return mcp.NewToolResultError(fmt.Sprintf("NO_WORKTREE: no worktree found for entity %s", entityID)), nil
			}
			return mcp.NewToolResultErrorFromErr("get worktree failed", err), nil
		}

		// Remove the git worktree
		if err := gitOps.RemoveWorktree(record.Path, force); err != nil {
			errStr := err.Error()
			if !force && (containsIgnoreCase(errStr, "uncommitted") || containsIgnoreCase(errStr, "untracked") || containsIgnoreCase(errStr, "changes")) {
				return mcp.NewToolResultError("UNCOMMITTED_CHANGES: worktree has uncommitted changes, use force=true to remove anyway"), nil
			}
			return mcp.NewToolResultError(fmt.Sprintf("GIT_ERROR: %v", err)), nil
		}

		// Delete the worktree record
		if err := store.Delete(record.ID); err != nil {
			return mcp.NewToolResultErrorFromErr("delete worktree record failed", err), nil
		}

		resp := map[string]any{
			"success": true,
			"removed": map[string]any{
				"id":   record.ID,
				"path": record.Path,
			},
		}
		return worktreeMapJSON(resp)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func branchStatusTool(store *worktree.Store, repoPath string, thresholds git.BranchThresholds) server.ServerTool {
	tool := mcp.NewTool("branch_status",
		mcp.WithDescription("Get branch health metrics for an entity's worktree branch. Reports staleness, drift from main, and merge conflicts."),
		mcp.WithString("entity_id", mcp.Description("Entity ID (FEAT-... or BUG-...)"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		entityID, err := request.RequireString("entity_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		record, err := store.GetByEntityID(entityID)
		if err != nil {
			if errors.Is(err, worktree.ErrNotFound) {
				return mcp.NewToolResultError(fmt.Sprintf("NO_WORKTREE: no worktree found for entity %s", entityID)), nil
			}
			return mcp.NewToolResultErrorFromErr("get worktree failed", err), nil
		}

		status, err := git.EvaluateBranchStatus(repoPath, record.Branch, thresholds)
		if err != nil {
			if errors.Is(err, git.ErrBranchNotFound) {
				return mcp.NewToolResultError(fmt.Sprintf("BRANCH_NOT_FOUND: branch %s does not exist", record.Branch)), nil
			}
			return mcp.NewToolResultErrorFromErr("evaluate branch status failed", err), nil
		}

		resp := map[string]any{
			"success": true,
			"branch":  record.Branch,
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
		return worktreeMapJSON(resp)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

// worktreeRecordToMap converts a worktree record to a map for JSON serialization.
func worktreeRecordToMap(r worktree.Record) map[string]any {
	m := map[string]any{
		"id":         r.ID,
		"entity_id":  r.EntityID,
		"branch":     r.Branch,
		"path":       r.Path,
		"status":     string(r.Status),
		"created":    r.Created.Format(time.RFC3339),
		"created_by": r.CreatedBy,
	}
	if r.MergedAt != nil {
		m["merged_at"] = r.MergedAt.Format(time.RFC3339)
	}
	if r.CleanupAfter != nil {
		m["cleanup_after"] = r.CleanupAfter.Format(time.RFC3339)
	}
	return m
}

// worktreeMapJSON marshals a map to JSON and returns it as a tool result.
func worktreeMapJSON(v map[string]any) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("marshal result: " + err.Error()), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

// entityTypeFromID extracts the entity type from an entity ID.
// Returns empty string if the ID format is not recognized.
func entityTypeFromID(id string) string {
	if len(id) < 4 {
		return ""
	}
	switch {
	case len(id) >= 5 && id[:5] == "FEAT-":
		return "feature"
	case len(id) >= 4 && id[:4] == "BUG-":
		return "bug"
	default:
		return ""
	}
}

// containsIgnoreCase checks if s contains substr, case-insensitively.
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
