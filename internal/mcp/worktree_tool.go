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

	"github.com/sambeau/kanbanzai/internal/config"
	"github.com/sambeau/kanbanzai/internal/service"
	"github.com/sambeau/kanbanzai/internal/worktree"
)

// WorktreeTool returns the 2.0 consolidated worktree tool.
// It consolidates worktree_create, worktree_get, worktree_list, and worktree_remove
// into a single tool with an action parameter (spec §19.1).
func WorktreeTool(store *worktree.Store, entitySvc *service.EntityService, gitOps *worktree.Git) []server.ServerTool {
	return []server.ServerTool{worktreeTool(store, entitySvc, gitOps)}
}

func worktreeTool(store *worktree.Store, entitySvc *service.EntityService, gitOps *worktree.Git) server.ServerTool {
	tool := mcp.NewTool("worktree",
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithTitleAnnotation("Git Worktree Manager"),
		mcp.WithDescription(
			"Manage Git worktrees for feature and bug entities. "+
				"Consolidates worktree_create, worktree_get, worktree_list, and worktree_remove. "+
				"Actions: create, get, list, remove.",
		),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("Action: create, get, list, remove"),
		),
		mcp.WithString("entity_id",
			mcp.Description("Entity ID (FEAT-... or BUG-...) — required for create, get, remove; optional filter for list"),
		),
		mcp.WithString("branch_name",
			mcp.Description("Custom branch name (auto-generated if omitted) — create only"),
		),
		mcp.WithString("created_by",
			mcp.Description("Who created the worktree. Auto-resolved if omitted — create only"),
		),
		mcp.WithString("slug",
			mcp.Description("Human-readable slug for branch naming (extracted from entity if omitted) — create only"),
		),
		mcp.WithString("status",
			mcp.Description("Filter by status: active, merged, abandoned, or all (default: all) — list only"),
		),
		mcp.WithBoolean("force",
			mcp.Description("Remove even with uncommitted changes (default: false) — remove only"),
		),
	)

	handler := WithSideEffects(func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		return DispatchAction(ctx, req, map[string]ActionHandler{
			"create": worktreeCreateAction(store, entitySvc, gitOps),
			"get":    worktreeGetAction(store),
			"list":   worktreeListAction(store),
			"remove": worktreeRemoveAction(store, gitOps),
		})
	})

	return server.ServerTool{Tool: tool, Handler: handler}
}

// ─── create ──────────────────────────────────────────────────────────────────

func worktreeCreateAction(store *worktree.Store, entitySvc *service.EntityService, gitOps *worktree.Git) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		SignalMutation(ctx)

		entityID, err := req.RequireString("entity_id")
		if err != nil {
			return inlineErr("missing_parameter", "entity_id is required for create action")
		}

		// Validate entity type (worktrees are only for features and bugs).
		entityType := entityTypeFromID(entityID)
		if entityType == "" {
			return inlineErr("invalid_entity_type", "entity ID must start with FEAT- or BUG-")
		}

		entity, err := entitySvc.Get(entityType, entityID, "")
		if err != nil {
			if errors.Is(err, service.ErrNotFound) {
				return inlineErr("entity_not_found", fmt.Sprintf("entity %s not found", entityID))
			}
			return nil, fmt.Errorf("get entity: %w", err)
		}

		// Check if a worktree already exists for this entity.
		existing, existErr := store.GetByEntityID(entityID)
		if existErr == nil && existing.ID != "" {
			return inlineErr("worktree_exists",
				fmt.Sprintf("worktree %s already exists for entity %s", existing.ID, entityID))
		}

		// Resolve identity.
		createdByRaw := req.GetString("created_by", "")
		createdBy, err := config.ResolveIdentity(createdByRaw)
		if err != nil {
			return nil, err
		}

		// Determine slug for naming.
		slug := req.GetString("slug", "")
		if slug == "" {
			if s, ok := entity.State["slug"].(string); ok {
				slug = s
			}
		}

		// Generate or use provided branch name.
		branchName := req.GetString("branch_name", "")
		if branchName == "" {
			branchName = worktree.GenerateBranchName(entityID, slug)
		}

		// Generate worktree path.
		wtPath := worktree.GenerateWorktreePath(entityID, slug)

		// Create the git worktree with a new branch.
		if err := gitOps.CreateWorktreeNewBranch(wtPath, branchName, ""); err != nil {
			return inlineErr("git_error", fmt.Sprintf("git worktree create failed: %v", err))
		}

		// Create the worktree record.
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
			// Best-effort cleanup of the git worktree.
			_ = gitOps.RemoveWorktree(wtPath, true)
			return nil, fmt.Errorf("create worktree record: %w", err)
		}

		return map[string]any{
			"worktree": worktreeRecordToMap(created),
		}, nil
	}
}

// ─── get ──────────────────────────────────────────────────────────────────────

func worktreeGetAction(store *worktree.Store) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		entityID, err := req.RequireString("entity_id")
		if err != nil {
			return inlineErr("missing_parameter", "entity_id is required for get action")
		}

		record, err := store.GetByEntityID(entityID)
		if err != nil {
			if errors.Is(err, worktree.ErrNotFound) {
				return inlineErr("no_worktree",
					fmt.Sprintf("no worktree found for entity %s", entityID))
			}
			return nil, fmt.Errorf("get worktree: %w", err)
		}

		return map[string]any{
			"worktree": worktreeRecordToMap(record),
		}, nil
	}
}

// ─── list ─────────────────────────────────────────────────────────────────────

func worktreeListAction(store *worktree.Store) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		statusFilter := req.GetString("status", "all")
		entityIDFilter := req.GetString("entity_id", "")

		records, err := store.List()
		if err != nil {
			return nil, fmt.Errorf("list worktrees: %w", err)
		}

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

		return map[string]any{
			"count":     len(worktrees),
			"worktrees": worktrees,
		}, nil
	}
}

// ─── remove ───────────────────────────────────────────────────────────────────

func worktreeRemoveAction(store *worktree.Store, gitOps *worktree.Git) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		SignalMutation(ctx)

		entityID, err := req.RequireString("entity_id")
		if err != nil {
			return inlineErr("missing_parameter", "entity_id is required for remove action")
		}
		force := req.GetBool("force", false)

		record, err := store.GetByEntityID(entityID)
		if err != nil {
			if errors.Is(err, worktree.ErrNotFound) {
				return inlineErr("no_worktree",
					fmt.Sprintf("no worktree found for entity %s", entityID))
			}
			return nil, fmt.Errorf("get worktree: %w", err)
		}

		// Remove the git worktree.
		if err := gitOps.RemoveWorktree(record.Path, force); err != nil {
			errStr := err.Error()
			if !force && (containsIgnoreCase(errStr, "uncommitted") ||
				containsIgnoreCase(errStr, "untracked") ||
				containsIgnoreCase(errStr, "changes")) {
				return inlineErr("uncommitted_changes",
					"worktree has uncommitted changes; use force=true to remove anyway")
			}
			return inlineErr("git_error", fmt.Sprintf("git worktree remove failed: %v", err))
		}

		// Delete the worktree record.
		if err := store.Delete(record.ID); err != nil {
			return nil, fmt.Errorf("delete worktree record: %w", err)
		}

		return map[string]any{
			"removed": map[string]any{
				"id":   record.ID,
				"path": record.Path,
			},
		}, nil
	}
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

// containsIgnoreCase checks if s contains substr, case-insensitively.
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
