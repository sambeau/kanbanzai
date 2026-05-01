package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/sambeau/kanbanzai/internal/worktree"
)

// invokeEditFile calls the edit_file tool handler with the given args and
// returns the parsed JSON response as a map.
func invokeEditFile(t *testing.T, repoRoot string, store *worktree.Store, args map[string]any) map[string]any {
	t.Helper()

	tool := editFileTool(repoRoot, store)
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args

	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("edit_file handler returned unexpected error: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("edit_file handler returned empty content")
	}

	tc, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected mcp.TextContent, got %T", result.Content[0])
	}

	var resp map[string]any
	if err := json.Unmarshal([]byte(tc.Text), &resp); err != nil {
		t.Fatalf("unmarshal response: %v\nraw: %s", err, tc.Text)
	}
	return resp
}

// assertEditFileErrorCode checks that resp contains an "error" object with the given code.
func assertEditFileErrorCode(t *testing.T, resp map[string]any, wantCode string) {
	t.Helper()

	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error in response, got: %v", resp)
	}
	if code := errObj["code"]; code != wantCode {
		t.Errorf("error.code = %q, want %q", code, wantCode)
	}
}

// ─── AC-004: entity_id with active worktree → edit applied in worktree ────────

// TestEditFile_EntityIDWithWorktree verifies AC-004:
// When entity_id is provided and an active worktree exists, edits are applied
// inside the worktree directory, not the main repo root.
func TestEditFile_EntityIDWithWorktree(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	wtDir := t.TempDir()
	store := worktree.NewStore(t.TempDir())

	// Create a file in the worktree to edit.
	origContent := "line one\nline two\nline three\n"
	wtFile := filepath.Join(wtDir, "target.txt")
	if err := os.MkdirAll(filepath.Dir(wtFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(wtFile, []byte(origContent), 0o644); err != nil {
		t.Fatal(err)
	}

	entityID := "FEAT-01KQG1AAAAAAA"
	_, err := store.Create(worktree.Record{
		EntityID:  entityID,
		Branch:    "feature/test-ac004",
		Path:      wtDir,
		Status:    worktree.StatusActive,
		Created:   time.Now().UTC(),
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("store.Create: %v", err)
	}

	resp := invokeEditFile(t, repoRoot, store, map[string]any{
		"entity_id":           entityID,
		"path":                "target.txt",
		"mode":                "edit",
		"display_description": "Test edit in worktree",
		"edits": []any{
			map[string]any{
				"old_text": "line two",
				"new_text": "line two modified",
			},
		},
	})

	if _, hasErr := resp["error"]; hasErr {
		t.Fatalf("unexpected error: %v", resp["error"])
	}

	// Verify file in worktree was modified.
	data, err := os.ReadFile(wtFile)
	if err != nil {
		t.Fatalf("ReadFile worktree: %v", err)
	}
	wantContent := "line one\nline two modified\nline three\n"
	if string(data) != wantContent {
		t.Errorf("worktree content = %q, want %q", string(data), wantContent)
	}

	// Verify no file was created in the repo root.
	repoFile := filepath.Join(repoRoot, "target.txt")
	if _, err := os.Stat(repoFile); err == nil {
		t.Errorf("file unexpectedly created in repo root: %s", repoFile)
	}
}

// ─── AC-005: entity_id omitted → writes to main repo root ─────────────────────

// TestEditFile_NoEntityID_BackwardCompat verifies AC-005:
// When entity_id is omitted, edits are applied relative to the main repository
// root, preserving backward compatibility.
func TestEditFile_NoEntityID_BackwardCompat(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	store := worktree.NewStore(t.TempDir())

	// Create a file in the repo root to edit.
	origContent := "hello world\nfoo bar\n"
	repoFile := filepath.Join(repoRoot, "test.txt")
	if err := os.WriteFile(repoFile, []byte(origContent), 0o644); err != nil {
		t.Fatal(err)
	}

	resp := invokeEditFile(t, repoRoot, store, map[string]any{
		"path":                "test.txt",
		"mode":                "edit",
		"display_description": "Test backward compat edit",
		"edits": []any{
			map[string]any{
				"old_text": "foo bar",
				"new_text": "baz qux",
			},
		},
	})

	if _, hasErr := resp["error"]; hasErr {
		t.Fatalf("unexpected error: %v", resp["error"])
	}

	data, err := os.ReadFile(repoFile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	wantContent := "hello world\nbaz qux\n"
	if string(data) != wantContent {
		t.Errorf("content = %q, want %q", string(data), wantContent)
	}
}

// ─── AC-006: entity_id for non-existent entity → "no worktree found" ──────────

// TestEditFile_NonExistentEntity verifies AC-006:
// When entity_id is provided but no worktree exists for that entity, the tool
// returns an error containing "no worktree found" and the entity ID.
func TestEditFile_NonExistentEntity(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	store := worktree.NewStore(t.TempDir())

	resp := invokeEditFile(t, repoRoot, store, map[string]any{
		"entity_id":           "FEAT-01NONEXISTENT",
		"path":                "test.txt",
		"mode":                "write",
		"display_description": "Should fail",
		"content":             "some content",
	})

	assertEditFileErrorCode(t, resp, "worktree_not_found")

	errObj := resp["error"].(map[string]any)
	msg, _ := errObj["message"].(string)
	if msg == "" {
		t.Fatal("error message is empty")
	}
	if !stringsContains(msg, "no worktree found") {
		t.Errorf("error message %q does not contain 'no worktree found'", msg)
	}
	if !stringsContains(msg, "FEAT-01NONEXISTENT") {
		t.Errorf("error message %q does not contain entity ID", msg)
	}
}

// ─── AC-007: entity_id with multi-edit payload → all edits applied in worktree ─

// TestEditFile_MultiEditInWorktree verifies AC-007:
// When entity_id is provided with a multi-edit payload, all edits are applied
// sequentially within the worktree directory.
func TestEditFile_MultiEditInWorktree(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	wtDir := t.TempDir()
	store := worktree.NewStore(t.TempDir())

	origContent := "package main\n\nfunc main() {\n\t// TODO: implement\n\tfmt.Println(\"hello\")\n}\n"
	wtFile := filepath.Join(wtDir, "main.go")
	if err := os.MkdirAll(filepath.Dir(wtFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(wtFile, []byte(origContent), 0o644); err != nil {
		t.Fatal(err)
	}

	entityID := "FEAT-01KQG1BBBBBBB"
	_, err := store.Create(worktree.Record{
		EntityID:  entityID,
		Branch:    "feature/test-ac007",
		Path:      wtDir,
		Status:    worktree.StatusActive,
		Created:   time.Now().UTC(),
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("store.Create: %v", err)
	}

	resp := invokeEditFile(t, repoRoot, store, map[string]any{
		"entity_id":           entityID,
		"path":                "main.go",
		"mode":                "edit",
		"display_description": "Multi-edit in worktree",
		"edits": []any{
			map[string]any{
				"old_text": "// TODO: implement",
				"new_text": "// Implementation complete",
			},
			map[string]any{
				"old_text": "\"hello\"",
				"new_text": "\"hello, world!\"",
			},
		},
	})

	if _, hasErr := resp["error"]; hasErr {
		t.Fatalf("unexpected error: %v", resp["error"])
	}

	data, err := os.ReadFile(wtFile)
	if err != nil {
		t.Fatalf("ReadFile worktree: %v", err)
	}
	wantContent := "package main\n\nfunc main() {\n\t// Implementation complete\n\tfmt.Println(\"hello, world!\")\n}\n"
	if string(data) != wantContent {
		t.Errorf("worktree content = %q, want %q", string(data), wantContent)
	}

	// edits_applied should be 2.
	editsApplied, _ := resp["edits_applied"].(float64)
	if int(editsApplied) != 2 {
		t.Errorf("edits_applied = %v, want 2", editsApplied)
	}
}

// ─── AC-009: existing test suite passes ──────────────────────────────────────
// This test verifies that the edit_file tool handler works correctly in write mode.
// It indirectly contributes to AC-009 by ensuring the handler doesn't break core functionality.

// TestEditFile_WriteMode verifies that write mode works correctly.
func TestEditFile_WriteMode(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	store := worktree.NewStore(t.TempDir())

	content := "package main\n\nfunc main() {}\n"
	resp := invokeEditFile(t, repoRoot, store, map[string]any{
		"path":                "main.go",
		"mode":                "write",
		"display_description": "Write new file",
		"content":             content,
	})

	if _, hasErr := resp["error"]; hasErr {
		t.Fatalf("unexpected error: %v", resp["error"])
	}

	gotPath, _ := resp["path"].(string)
	wantPath := filepath.Join(repoRoot, "main.go")
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}

	gotBytes, _ := resp["bytes"].(float64)
	if int(gotBytes) != len(content) {
		t.Errorf("bytes = %d, want %d", int(gotBytes), len(content))
	}

	data, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != content {
		t.Errorf("content = %q, want %q", string(data), content)
	}
}

// TestEditFile_EmptyPathRejected verifies that empty path returns error.
func TestEditFile_EmptyPathRejected(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	store := worktree.NewStore(t.TempDir())

	resp := invokeEditFile(t, repoRoot, store, map[string]any{
		"path":                "",
		"mode":                "write",
		"display_description": "Empty path",
		"content":             "content",
	})

	assertEditFileErrorCode(t, resp, "missing_parameter")
}

// TestEditFile_InvalidModeRejected verifies that invalid mode returns error.
func TestEditFile_InvalidModeRejected(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	store := worktree.NewStore(t.TempDir())

	// Need a file to exist for edit mode, but first test mode validation.
	resp := invokeEditFile(t, repoRoot, store, map[string]any{
		"path":                "test.txt",
		"mode":                "invalid",
		"display_description": "Invalid mode",
	})

	assertEditFileErrorCode(t, resp, "invalid_parameter")
}

// TestEditFile_PathTraversalRejected verifies path traversal is blocked.
func TestEditFile_PathTraversalRejected(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	store := worktree.NewStore(t.TempDir())

	resp := invokeEditFile(t, repoRoot, store, map[string]any{
		"path":                "../../etc/passwd",
		"mode":                "write",
		"display_description": "Traversal attempt",
		"content":             "malicious",
	})

	assertEditFileErrorCode(t, resp, "path_traversal")
}

// TestEditFile_WriteModeInWorktree verifies write mode works in worktrees.
func TestEditFile_WriteModeInWorktree(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	wtDir := t.TempDir()
	store := worktree.NewStore(t.TempDir())

	entityID := "FEAT-01KQG1CCCCCCC"
	_, err := store.Create(worktree.Record{
		EntityID:  entityID,
		Branch:    "feature/test-write-wt",
		Path:      wtDir,
		Status:    worktree.StatusActive,
		Created:   time.Now().UTC(),
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("store.Create: %v", err)
	}

	content := "new file in worktree"
	resp := invokeEditFile(t, repoRoot, store, map[string]any{
		"entity_id":           entityID,
		"path":                "newfile.txt",
		"mode":                "write",
		"display_description": "Write new file in worktree",
		"content":             content,
	})

	if _, hasErr := resp["error"]; hasErr {
		t.Fatalf("unexpected error: %v", resp["error"])
	}

	wtFile := filepath.Join(wtDir, "newfile.txt")
	data, err := os.ReadFile(wtFile)
	if err != nil {
		t.Fatalf("ReadFile worktree: %v", err)
	}
	if string(data) != content {
		t.Errorf("content = %q, want %q", string(data), content)
	}

	// Not in repo root.
	repoFile := filepath.Join(repoRoot, "newfile.txt")
	if _, err := os.Stat(repoFile); err == nil {
		t.Errorf("file unexpectedly in repo root: %s", repoFile)
	}
}

// TestEditFile_FuzzyMatching verifies that fuzzy whitespace matching works.
func TestEditFile_FuzzyMatching(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	store := worktree.NewStore(t.TempDir())

	// File has multi-line content with extra spaces.
	origContent := "func hello() {\n    fmt.Println(\"hi\")\n}\n"
	repoFile := filepath.Join(repoRoot, "hello.go")
	if err := os.WriteFile(repoFile, []byte(origContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Search text has different whitespace than the file.
	resp := invokeEditFile(t, repoRoot, store, map[string]any{
		"path":                "hello.go",
		"mode":                "edit",
		"display_description": "Fuzzy match test",
		"edits": []any{
			map[string]any{
				"old_text": "func hello() {\nfmt.Println(\"hi\")\n}",
				"new_text": "func hello() {\n\tfmt.Println(\"hello\")\n}",
			},
		},
	})

	if _, hasErr := resp["error"]; hasErr {
		t.Fatalf("unexpected error: %v", resp["error"])
	}

	data, err := os.ReadFile(repoFile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	wantContent := "func hello() {\n\tfmt.Println(\"hello\")\n}\n"
	if string(data) != wantContent {
		t.Errorf("content = %q, want %q", string(data), wantContent)
	}
}

// ─── Tab indentation preservation tests ──────────────────────────────────────

// TestEditFile_TabIndentedLineReplacementPreservesTabs verifies AC-001:
// When a tab-indented line in a Go file is replaced via edit mode, the
// replacement text inherits the tab indentation level of the matched line.
func TestEditFile_TabIndentedLineReplacementPreservesTabs(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	store := worktree.NewStore(t.TempDir())

	origContent := "package main\n\nfunc main() {\n\tx := 1\n\ty := 2\n\tz := 3\n}\n"
	repoFile := filepath.Join(repoRoot, "main.go")
	if err := os.WriteFile(repoFile, []byte(origContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Replace one tab-indented line. newText has no leading whitespace.
	resp := invokeEditFile(t, repoRoot, store, map[string]any{
		"path":                "main.go",
		"mode":                "edit",
		"display_description": "Replace tab-indented line",
		"edits": []any{
			map[string]any{
				"old_text": "y := 2",
				"new_text": "y := 42",
			},
		},
	})

	if _, hasErr := resp["error"]; hasErr {
		t.Fatalf("unexpected error: %v", resp["error"])
	}

	data, err := os.ReadFile(repoFile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	// The replacement should be tab-indented, same level as surrounding lines.
	wantContent := "package main\n\nfunc main() {\n\tx := 1\n\ty := 42\n\tz := 3\n}\n"
	if string(data) != wantContent {
		t.Errorf("content =\n%q\nwant =\n%q", string(data), wantContent)
	}

	// Verify surrounding lines retain tab indentation.
	lines := strings.SplitN(string(data), "\n", -1)
	for _, line := range lines {
		if strings.HasPrefix(line, "x :=") || strings.HasPrefix(line, "z :=") {
			if !strings.HasPrefix(line, "\t") {
				t.Errorf("surrounding line lost tab indentation: %q", line)
			}
		}
	}
}

// TestEditFile_StructFieldReplacementDoesntCorruptNeighbors verifies AC-002:
// When a struct field in a tab-indented Go struct is replaced, other fields
// retain their tab indentation.
func TestEditFile_StructFieldReplacementDoesntCorruptNeighbors(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	store := worktree.NewStore(t.TempDir())

	origContent := "package main\n\ntype Config struct {\n\tName    string\n\tAge     int\n\tEnabled bool\n}\n"
	repoFile := filepath.Join(repoRoot, "types.go")
	if err := os.WriteFile(repoFile, []byte(origContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Replace "int" with "uint" — newText has no leading whitespace.
	resp := invokeEditFile(t, repoRoot, store, map[string]any{
		"path":                "types.go",
		"mode":                "edit",
		"display_description": "Replace struct field type",
		"edits": []any{
			map[string]any{
				"old_text": "Age     int",
				"new_text": "Age     uint",
			},
		},
	})

	if _, hasErr := resp["error"]; hasErr {
		t.Fatalf("unexpected error: %v", resp["error"])
	}

	data, err := os.ReadFile(repoFile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	// Verify no field has extra tab indentation (double-tab).
	if strings.Contains(string(data), "\t\t") {
		t.Errorf("content contains double-tab indentation: %q", string(data))
	}

	// Verify exact content to catch any whitespace corruption.
	wantContent := "package main\n\ntype Config struct {\n\tName    string\n\tAge     uint\n\tEnabled bool\n}\n"
	if string(data) != wantContent {
		t.Errorf("content =\n%q\nwant =\n%q", string(data), wantContent)
	}
}

// TestEditFile_SpaceIndentedFilesUnchanged verifies AC-003:
// Space-indented files are unaffected by the fix — existing behaviour preserved.
func TestEditFile_SpaceIndentedFilesUnchanged(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	store := worktree.NewStore(t.TempDir())

	// Space-indented YAML file.
	origContent := "root:\n  key: old-value\n  nested:\n    child: foo\n"
	repoFile := filepath.Join(repoRoot, "config.yaml")
	if err := os.WriteFile(repoFile, []byte(origContent), 0o644); err != nil {
		t.Fatal(err)
	}

	resp := invokeEditFile(t, repoRoot, store, map[string]any{
		"path":                "config.yaml",
		"mode":                "edit",
		"display_description": "Replace YAML value",
		"edits": []any{
			map[string]any{
				"old_text": "key: old-value",
				"new_text": "key: new-value",
			},
		},
	})

	if _, hasErr := resp["error"]; hasErr {
		t.Fatalf("unexpected error: %v", resp["error"])
	}

	data, err := os.ReadFile(repoFile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	wantContent := "root:\n  key: new-value\n  nested:\n    child: foo\n"
	if string(data) != wantContent {
		t.Errorf("content =\n%q\nwant =\n%q", string(data), wantContent)
	}
}

// TestEditFile_FuzzyMatchPreservesTabIndentation verifies that fuzzy matching
// with tab-indented files preserves indentation (critical bug scenario).
func TestEditFile_FuzzyMatchPreservesTabIndentation(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	store := worktree.NewStore(t.TempDir())

	// Tab-indented Go file simulating the bug scenario: a string slice with
	// tab-indented entries.
	origContent := "var tools = []string{\n\t\"read\",\n\t\"write\",\n\t\"list\",\n}\n"
	repoFile := filepath.Join(repoRoot, "tools.go")
	if err := os.WriteFile(repoFile, []byte(origContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Replace a tab-indented entry — oldText has no tabs (fuzzy matching
	// normalizes them away) and newText has no tabs either.
	resp := invokeEditFile(t, repoRoot, store, map[string]any{
		"path":                "tools.go",
		"mode":                "edit",
		"display_description": "Replace tab-indented slice entry",
		"edits": []any{
			map[string]any{
				"old_text": "\"list\"",
				"new_text": "\"download\"",
			},
		},
	})

	if _, hasErr := resp["error"]; hasErr {
		t.Fatalf("unexpected error: %v", resp["error"])
	}

	data, err := os.ReadFile(repoFile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	wantContent := "var tools = []string{\n\t\"read\",\n\t\"write\",\n\t\"download\",\n}\n"
	if string(data) != wantContent {
		t.Errorf("content =\n%q\nwant =\n%q", string(data), wantContent)
	}
}

// stringsContains reports whether s contains substr.
func stringsContains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
