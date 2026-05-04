package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/sambeau/kanbanzai/internal/hashvalidate"
	"github.com/sambeau/kanbanzai/internal/worktree"
)

// ReadFileTool returns the read_file tool that reads a file within
// a repo root or worktree, with worktree path resolution and traversal prevention.
// Supports optional hash-tagged output for use with edit_file's hash_validate mode.
func ReadFileTool(repoRoot string, worktreeStore *worktree.Store) []server.ServerTool {
	return []server.ServerTool{readFileTool(repoRoot, worktreeStore)}
}

func readFileTool(repoRoot string, worktreeStore *worktree.Store) server.ServerTool {
	tool := mcp.NewTool("read_file",
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithTitleAnnotation("Read File"),
		mcp.WithDescription(
			"Read the contents of a file within the repository root or a specific worktree. "+
				"Supports optional hash-tagged output for use with edit_file's hash_validate mode. "+
				"Provide entity_id to read from a worktree's working directory; "+
				"omit entity_id to read relative to the repository root.",
		),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("The full path of the file to read in the project."),
		),
		mcp.WithString("entity_id",
			mcp.Description("Entity ID (FEAT-... or BUG-...) to scope reads to the entity's active worktree. Omit to use the repository root."),
		),
		mcp.WithBoolean("hash_tag",
			mcp.Description("When true, each line is prefixed with its line number, a 2-char content hash, and a separator (format: {line}#{hash}| {content}). Line numbers are 1-based and left-padded to at least 4 chars. When false or absent, plain text is returned."),
		),
		mcp.WithNumber("start_line",
			mcp.Description("Optional line number to start reading on (1-based index)."),
		),
		mcp.WithNumber("end_line",
			mcp.Description("Optional line number to end reading on (1-based index, inclusive)."),
		),
	)

	handler := WithSideEffects(func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		path := req.GetString("path", "")
		if path == "" {
			return inlineErr("missing_parameter", "path must not be empty")
		}

		root := repoRoot
		entityID := req.GetString("entity_id", "")
		if entityID != "" {
			record, err := worktreeStore.GetByEntityID(entityID)
			if err != nil || record == nil {
				return inlineErr("worktree_not_found", "no active worktree for entity "+entityID)
			}
			root = record.Path
		}

		absRoot, err := filepath.Abs(root)
		if err != nil {
			return nil, fmt.Errorf("resolve root: %w", err)
		}

		var resolved string
		if filepath.IsAbs(path) {
			resolved = filepath.Clean(path)
		} else {
			resolved = filepath.Clean(filepath.Join(absRoot, path))
		}

		if resolved != absRoot && !strings.HasPrefix(resolved, absRoot+string(os.PathSeparator)) {
			return inlineErr("path_traversal", "path escapes root")
		}

		data, err := os.ReadFile(resolved)
		if err != nil {
			return nil, fmt.Errorf("read file: %w", err)
		}

		content := string(data)
		lines := strings.Split(content, "\n")

		// Apply line range if specified.
		startLine := 1
		endLine := len(lines)
		if req.Params.Arguments != nil {
			if args, ok := req.Params.Arguments.(map[string]any); ok {
				if sl, ok := args["start_line"].(float64); ok && sl > 0 {
					startLine = int(sl)
				}
				if el, ok := args["end_line"].(float64); ok && el > 0 {
					endLine = int(el)
				}
			}
		}
		if startLine < 1 {
			startLine = 1
		}
		if endLine > len(lines) {
			endLine = len(lines)
		}
		if startLine > endLine {
			return inlineErr("invalid_parameter", "start_line must be <= end_line")
		}

		hashTag := req.GetBool("hash_tag", false)
		if !hashTag {
			// Plain text output (no hash tags).
			selected := lines[startLine-1 : endLine]
			return map[string]any{
				"path":    resolved,
				"content": strings.Join(selected, "\n"),
			}, nil
		}

		// Hash-tagged output.
		// Build the output from selected lines, stripping a trailing empty
		// line if the file ends with \n.
		selected := lines[startLine-1 : endLine]
		if len(selected) > 0 && selected[len(selected)-1] == "" {
			selected = selected[:len(selected)-1]
		}
		var out strings.Builder
		for i, line := range selected {
			lineNum := startLine + i // 1-based absolute line number
			hash := hashvalidate.HashLine(line)
			fmt.Fprintf(&out, "%4d#%s| %s\n", lineNum, hash, line)
		}

		return map[string]any{
			"path":    resolved,
			"content": out.String(),
		}, nil
	})

	return server.ServerTool{Tool: tool, Handler: handler}
}
