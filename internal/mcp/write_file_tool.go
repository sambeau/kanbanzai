package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/sambeau/kanbanzai/internal/fsutil"
	"github.com/sambeau/kanbanzai/internal/worktree"
)

// WriteFileTool returns the write_file tool that writes content to a file within
// a repo root or worktree, with full path resolution and traversal prevention.
func WriteFileTool(repoRoot string, worktreeStore *worktree.Store) []server.ServerTool {
	return []server.ServerTool{writeFileTool(repoRoot, worktreeStore)}
}

func writeFileTool(repoRoot string, worktreeStore *worktree.Store) server.ServerTool {
	tool := mcp.NewTool("write_file",
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithTitleAnnotation("Write File"),
		mcp.WithDescription(
			"Write content to a file within the repository root or a specific worktree. "+
				"Creates parent directories as needed. Uses atomic write to prevent corruption. "+
				"Provide entity_id to write into a worktree's working directory; "+
				"omit entity_id to write relative to the repository root.",
		),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("File path to write. Relative paths are resolved against the root (repo root or worktree path). Absolute paths must be within the root."),
		),
		mcp.WithString("content",
			mcp.Required(),
			mcp.Description("File content to write."),
		),
		mcp.WithString("entity_id",
			mcp.Description("Entity ID (FEAT-... or BUG-...) to scope writes to the entity's active worktree. Omit to use the repository root."),
		),
	)

	handler := WithSideEffects(func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		SignalMutation(ctx)

		path := req.GetString("path", "")
		if path == "" {
			return inlineErr("missing_parameter", "path must not be empty")
		}

		args, _ := req.Params.Arguments.(map[string]any)
		_, hasContent := args["content"]
		if !hasContent {
			return inlineErr("missing_parameter", "content must not be empty")
		}
		content := req.GetString("content", "")

		root := repoRoot
		entityID := req.GetString("entity_id", "")
		if entityID != "" {
			record, err := worktreeStore.GetByEntityID(entityID)
			if err != nil || record == nil {
				return inlineErr("worktree_not_found", "no active worktree for entity "+entityID)
			}
			root = record.Path
		}

		// Resolve to absolute root for reliable prefix comparison.
		absRoot, err := filepath.Abs(root)
		if err != nil {
			return nil, fmt.Errorf("resolve root: %w", err)
		}

		// Resolve path: join relative paths onto root; clean absolute paths in place.
		var resolved string
		if filepath.IsAbs(path) {
			resolved = filepath.Clean(path)
		} else {
			resolved = filepath.Clean(filepath.Join(absRoot, path))
		}

		// Traversal check: resolved must be within absRoot.
		if resolved != absRoot && !strings.HasPrefix(resolved, absRoot+string(os.PathSeparator)) {
			return inlineErr("path_traversal", "path escapes root")
		}

		// Create parent directories as needed.
		if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
			return nil, fmt.Errorf("create directories: %w", err)
		}

		// Write file atomically.
		if err := fsutil.WriteFileAtomic(resolved, []byte(content), 0o644); err != nil {
			return nil, fmt.Errorf("write file: %w", err)
		}

		return map[string]any{
			"path":  resolved,
			"bytes": len(content),
		}, nil
	})

	return server.ServerTool{Tool: tool, Handler: handler}
}
