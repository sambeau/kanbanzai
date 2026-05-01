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

// EditFileTool returns the edit_file tool that edits a file within
// a repo root or worktree, with worktree path resolution and traversal prevention.
func EditFileTool(repoRoot string, worktreeStore *worktree.Store) []server.ServerTool {
	return []server.ServerTool{editFileTool(repoRoot, worktreeStore)}
}

func editFileTool(repoRoot string, worktreeStore *worktree.Store) server.ServerTool {
	tool := mcp.NewTool("edit_file",
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithTitleAnnotation("Edit File"),
		mcp.WithDescription(
			"Create or update a single file in the repository root or a specific worktree. "+
				"Supports write mode (full file overwrite) and edit mode (granular find-and-replace). "+
				"Provide entity_id to write into a worktree's working directory; "+
				"omit entity_id to write relative to the repository root.",
		),
		mcp.WithString("display_description",
			mcp.Required(),
			mcp.Description("A one-line, user-friendly description of the edit."),
		),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("The full path of the file to create or modify in the project."),
		),
		mcp.WithString("mode",
			mcp.Required(),
			mcp.Description("The mode of operation: 'write' (full file overwrite) or 'edit' (granular find-and-replace)."),
		),
		mcp.WithString("content",
			mcp.Description("The complete content for the new file (required for 'write' mode)."),
		),
		mcp.WithArray("edits",
			mcp.Items(map[string]any{"type": "object"}),
			mcp.Description("List of edit operations to apply sequentially (required for 'edit' mode). Each edit finds old_text and replaces it with new_text."),
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

		mode := req.GetString("mode", "")
		if mode != "write" && mode != "edit" {
			return inlineErr("invalid_parameter", "mode must be 'write' or 'edit'")
		}

		root := repoRoot
		entityID := req.GetString("entity_id", "")
		if entityID != "" {
			record, err := worktreeStore.GetByEntityID(entityID)
			if err != nil || record == nil {
				return inlineErr("worktree_not_found", "no worktree found for entity "+entityID)
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

		switch mode {
		case "write":
			content := req.GetString("content", "")
			if content == "" {
				return inlineErr("missing_parameter", "content must not be empty for 'write' mode")
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

		case "edit":
			args, _ := req.Params.Arguments.(map[string]any)
			editsRaw, ok := args["edits"]
			if !ok {
				return inlineErr("missing_parameter", "edits must be provided for 'edit' mode")
			}

			edits, ok := editsRaw.([]any)
			if !ok {
				return inlineErr("invalid_parameter", "edits must be an array")
			}
			if len(edits) == 0 {
				return inlineErr("invalid_parameter", "edits must not be empty for 'edit' mode")
			}

			// Read the current file content.
			current, err := os.ReadFile(resolved)
			if err != nil {
				return nil, fmt.Errorf("read file for editing: %w", err)
			}

			display := req.GetString("display_description", "")

			// Apply edits sequentially.
			modified := string(current)
			for i, editRaw := range edits {
				editMap, ok := editRaw.(map[string]any)
				if !ok {
					return inlineErr("invalid_parameter", fmt.Sprintf("edits[%d] must be an object", i))
				}
				oldText, _ := editMap["old_text"].(string)
				newText, _ := editMap["new_text"].(string)
				if oldText == "" && newText == "" {
					continue
				}

				// Do fuzzy matching: try exact first, fall back to fuzzy.
				idx := strings.Index(modified, oldText)
				matchLen := len(oldText)
				if idx < 0 {
					// Fuzzy match: normalize whitespace and try again.
					idx, matchLen = fuzzyMatch(modified, oldText)
					if idx < 0 {
						return inlineErr("edit_failed",
							fmt.Sprintf("edit %d: could not find old_text in file (even with fuzzy matching)", i+1))
					}
				}

				// Preserve indentation: when oldText matches at the start
				// of an indented line and newText doesn't start with
				// whitespace, extend the match backward to include the
				// indentation so newText inherits it without doubling.
				lineIndent := lineIndentation(modified, idx)
				if lineIndent != "" && !startsWithWhitespace(newText) {
					idx -= len(lineIndent)
					matchLen += len(lineIndent)
					newText = lineIndent + newText
				}

				modified = modified[:idx] + newText + modified[idx+matchLen:]
			}

			// Write modifications back atomically.
			if err := fsutil.WriteFileAtomic(resolved, []byte(modified), 0o644); err != nil {
				return nil, fmt.Errorf("write edited file: %w", err)
			}

			return map[string]any{
				"path":          resolved,
				"display":       display,
				"edits_applied": len(edits),
			}, nil
		}

		return inlineErr("invalid_parameter", "unknown mode: "+mode)
	})

	return server.ServerTool{Tool: tool, Handler: handler}
}

// fuzzyMatch finds oldText in s with tolerance for whitespace differences.
// It normalizes both strings, finds the match in normalized space, maps the
// start and end positions back to the original string, and returns the start
// position and match length in the original string.
func fuzzyMatch(s, oldText string) (start, length int) {
	normalized := normalizeWhitespace(s)
	normalizedOld := normalizeWhitespace(oldText)
	nIdx := strings.Index(normalized, normalizedOld)
	if nIdx < 0 {
		return -1, 0
	}
	start = mapToOriginal(s, nIdx)
	end := mapToOriginal(s, nIdx+len(normalizedOld))
	return start, end - start
}

// mapToOriginal converts a position in the normalized string back to the
// corresponding position in the original string.
func mapToOriginal(original string, normalizedIdx int) int {
	origIdx := 0
	nIdx := 0
	inSpace := false
	for origIdx < len(original) && nIdx < normalizedIdx {
		r := original[origIdx]
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if !inSpace {
				nIdx++
				inSpace = true
			}
			origIdx++
		} else {
			nIdx++
			origIdx++
			inSpace = false
		}
	}
	return origIdx
}

// lineIndentation returns the leading whitespace (tabs/spaces) on the line
// containing position idx in s. Returns empty string if the line has no
// leading whitespace, or if idx is not at the first non-whitespace character
// of the line (i.e. there is non-whitespace content between line start and idx).
func lineIndentation(s string, idx int) string {
	// Find start of line (position after last newline before idx).
	lineStart := 0
	for i := idx - 1; i >= 0; i-- {
		if s[i] == '\n' {
			lineStart = i + 1
			break
		}
	}
	// Walk from lineStart to idx: everything must be whitespace.
	for i := lineStart; i < idx; i++ {
		if s[i] != ' ' && s[i] != '\t' {
			return "" // non-whitespace between line start and idx — not at line start
		}
	}
	// Everything from lineStart to idx is whitespace.
	return s[lineStart:idx]
}

// startsWithWhitespace reports whether s starts with a space or tab.
func startsWithWhitespace(s string) bool {
	return len(s) > 0 && (s[0] == ' ' || s[0] == '\t')
}

// normalizeWhitespace collapses consecutive whitespace characters (spaces, tabs,
// newlines) into a single space.
func normalizeWhitespace(s string) string {
	var b strings.Builder
	inSpace := false
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if !inSpace {
				b.WriteByte(' ')
				inSpace = true
			}
		} else {
			b.WriteRune(r)
			inSpace = false
		}
	}
	return b.String()
}
