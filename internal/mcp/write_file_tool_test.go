package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/sambeau/kanbanzai/internal/service"
	"github.com/sambeau/kanbanzai/internal/worktree"
)

// invokeWriteFile calls the write_file tool handler with the given args and
// returns the parsed JSON response as a map.
func invokeWriteFile(t *testing.T, repoRoot string, store *worktree.Store, args map[string]any) map[string]any {
	t.Helper()

	tool := writeFileTool(repoRoot, store, nil)
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args

	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("write_file handler returned unexpected error: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("write_file handler returned empty content")
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

// assertErrorCode checks that resp contains an "error" object with the given code.
func assertWriteFileErrorCode(t *testing.T, resp map[string]any, wantCode string) {
	t.Helper()

	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error in response, got: %v", resp)
	}
	if code := errObj["code"]; code != wantCode {
		t.Errorf("error.code = %q, want %q", code, wantCode)
	}
}

// ─── AC-01: Basic write to repo root ─────────────────────────────────────────

// TestWriteFile_BasicWrite verifies AC-01:
// Writing a file relative to repoRoot creates the file with the correct content,
// and the response contains the absolute path and byte count.
func TestWriteFile_BasicWrite(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	store := worktree.NewStore(t.TempDir())

	content := "hello world"
	resp := invokeWriteFile(t, repoRoot, store, map[string]any{
		"path":    "hello.txt",
		"content": content,
	})

	// Response must not contain an error.
	if _, hasErr := resp["error"]; hasErr {
		t.Fatalf("unexpected error in response: %v", resp["error"])
	}

	// Response must contain "path" and "bytes".
	gotPath, ok := resp["path"].(string)
	if !ok {
		t.Fatalf("expected string path in response, got: %v", resp["path"])
	}
	gotBytes, ok := resp["bytes"].(float64)
	if !ok {
		t.Fatalf("expected numeric bytes in response, got: %v (%T)", resp["bytes"], resp["bytes"])
	}

	// Path must be within repoRoot.
	wantPath := filepath.Join(repoRoot, "hello.txt")
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}

	// Byte count must equal len(content).
	if int(gotBytes) != len(content) {
		t.Errorf("bytes = %d, want %d", int(gotBytes), len(content))
	}

	// File must exist on disk with exact content.
	data, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != content {
		t.Errorf("on-disk content = %q, want %q", string(data), content)
	}
}

// ─── AC-02: Write to worktree ─────────────────────────────────────────────────

// TestWriteFile_WriteToWorktree verifies AC-02:
// When entity_id is provided and an active worktree record exists, the file is
// written into the worktree directory, not the repo root.
func TestWriteFile_WriteToWorktree(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	wtDir := t.TempDir()
	store := worktree.NewStore(t.TempDir())

	entityID := "FEAT-01AAAAAAAAAAAAA"
	_, err := store.Create(worktree.Record{
		EntityID:  entityID,
		Branch:    "feature/test",
		Path:      wtDir,
		Status:    worktree.StatusActive,
		Created:   time.Now().UTC(),
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("store.Create: %v", err)
	}

	content := "worktree content"
	resp := invokeWriteFile(t, repoRoot, store, map[string]any{
		"entity_id": entityID,
		"path":      "output.txt",
		"content":   content,
	})

	if _, hasErr := resp["error"]; hasErr {
		t.Fatalf("unexpected error: %v", resp["error"])
	}

	// File must exist inside wtDir, not repoRoot.
	expectedPath := filepath.Join(wtDir, "output.txt")
	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("file not found in worktree dir: %v", err)
	}
	if string(data) != content {
		t.Errorf("content = %q, want %q", string(data), content)
	}

	// File must NOT exist in repoRoot.
	repoRootPath := filepath.Join(repoRoot, "output.txt")
	if _, err := os.Stat(repoRootPath); err == nil {
		t.Errorf("file unexpectedly written to repoRoot: %s", repoRootPath)
	}
}

// ─── AC-03: Directory auto-creation ──────────────────────────────────────────

// TestWriteFile_DirectoryAutoCreation verifies AC-03:
// Writing to a path with non-existent parent directories creates them automatically.
func TestWriteFile_DirectoryAutoCreation(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	store := worktree.NewStore(t.TempDir())

	resp := invokeWriteFile(t, repoRoot, store, map[string]any{
		"path":    "subdir/nested/deep/file.txt",
		"content": "nested content",
	})

	if _, hasErr := resp["error"]; hasErr {
		t.Fatalf("unexpected error: %v", resp["error"])
	}

	targetPath := filepath.Join(repoRoot, "subdir", "nested", "deep", "file.txt")
	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("file not found after auto-creation: %v", err)
	}
	if string(data) != "nested content" {
		t.Errorf("content = %q, want %q", string(data), "nested content")
	}
}

// ─── AC-05: Path traversal rejected ──────────────────────────────────────────

// TestWriteFile_PathTraversalRejected verifies AC-05:
// A path that escapes the root via ".." components returns a path_traversal error.
func TestWriteFile_PathTraversalRejected(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	store := worktree.NewStore(t.TempDir())

	resp := invokeWriteFile(t, repoRoot, store, map[string]any{
		"path":    "../../etc/passwd",
		"content": "malicious",
	})

	assertWriteFileErrorCode(t, resp, "path_traversal")

	// File must not have been created.
	if _, err := os.Stat("/etc/passwd_kbz_test"); err == nil {
		t.Error("traversal file should not exist")
	}
}

// ─── AC-06: Missing entity_id worktree ───────────────────────────────────────

// TestWriteFile_MissingWorktree verifies AC-06:
// When entity_id is set but no active worktree exists, worktree_not_found is returned.
func TestWriteFile_MissingWorktree(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	store := worktree.NewStore(t.TempDir())

	resp := invokeWriteFile(t, repoRoot, store, map[string]any{
		"entity_id": "FEAT-missing",
		"path":      "test.txt",
		"content":   "content",
	})

	assertWriteFileErrorCode(t, resp, "worktree_not_found")
}

// ─── AC-07: Empty path rejected ──────────────────────────────────────────────

// TestWriteFile_EmptyPathRejected verifies AC-07:
// An empty path returns a missing_parameter error.
func TestWriteFile_EmptyPathRejected(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	store := worktree.NewStore(t.TempDir())

	resp := invokeWriteFile(t, repoRoot, store, map[string]any{
		"path":    "",
		"content": "some content",
	})

	assertWriteFileErrorCode(t, resp, "missing_parameter")
}

// ─── AC-08: Missing content rejected ─────────────────────────────────────────

// TestWriteFile_MissingContentRejected verifies AC-08:
// Omitting the content key entirely returns a missing_parameter error.
func TestWriteFile_MissingContentRejected(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	store := worktree.NewStore(t.TempDir())

	// content key is intentionally absent from args.
	resp := invokeWriteFile(t, repoRoot, store, map[string]any{
		"path": "test.txt",
	})

	assertWriteFileErrorCode(t, resp, "missing_parameter")
}

// ─── Mutation warning tests (FR-206 through FR-210) ─────────────────────────

// createTestBug writes a minimal bug YAML file in the entity service root.
func createTestBug(t *testing.T, svc *service.EntityService, bugID, slug, status string) {
	t.Helper()

	bugsDir := filepath.Join(svc.Root(), "bugs")
	if err := os.MkdirAll(bugsDir, 0o755); err != nil {
		t.Fatalf("create bugs dir: %v", err)
	}

	yaml := fmt.Sprintf(`id: %s
slug: %s
name: Test Bug
status: %s
severity: medium
priority: medium
type: implementation-defect
reported_by: tester
reported: "2026-01-01T00:00:00Z"
observed: observed
expected: expected
`, bugID, slug, status)

	path := filepath.Join(bugsDir, bugID+"-"+slug+".yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatalf("write bug yaml: %v", err)
	}
}

// invokeWriteFileWithSvc is like invokeWriteFile but passes an entity service.
func invokeWriteFileWithSvc(t *testing.T, repoRoot string, store *worktree.Store, entitySvc *service.EntityService, args map[string]any) map[string]any {
	t.Helper()

	tool := writeFileTool(repoRoot, store, entitySvc)
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args

	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("write_file handler returned unexpected error: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("write_file handler returned empty content")
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

// TestWriteFile_NoWarningWhenEntityIDProvided verifies FR-210:
// No warning when entity_id is provided.
func TestWriteFile_NoWarningWhenEntityIDProvided(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	stateRoot := t.TempDir()

	entitySvc := service.NewEntityService(stateRoot)
	createTestBug(t, entitySvc, "BUG-01AAAAAAAAAAAAA", "test-bug", "in-progress")

	wtStore := worktree.NewStore(t.TempDir())
	wtDir := t.TempDir()
	_, err := wtStore.Create(worktree.Record{
		EntityID:  "BUG-01AAAAAAAAAAAAA",
		Branch:    "bug/test-bug",
		Path:      wtDir,
		Status:    worktree.StatusActive,
		Created:   time.Now().UTC(),
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("create worktree: %v", err)
	}

	resp := invokeWriteFileWithSvc(t, repoRoot, wtStore, entitySvc, map[string]any{
		"entity_id": "BUG-01AAAAAAAAAAAAA",
		"path":      "test.txt",
		"content":   "content",
	})

	if _, hasErr := resp["error"]; hasErr {
		t.Fatalf("unexpected error: %v", resp["error"])
	}
	if w, ok := resp["warning"].(string); ok && w != "" {
		t.Errorf("unexpected warning with entity_id: %q", w)
	}
}

// TestWriteFile_WarningWhenActiveBugWorktree verifies FR-206 and FR-207:
// Warning fires when writing without entity_id and an in-progress bug
// has an active worktree.
func TestWriteFile_WarningWhenActiveBugWorktree(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	stateRoot := t.TempDir()

	entitySvc := service.NewEntityService(stateRoot)
	bugID := "BUG-01BBBBBBBBBBBBB"
	createTestBug(t, entitySvc, bugID, "test-bug", "in-progress")

	wtStore := worktree.NewStore(t.TempDir())
	wtDir := t.TempDir()
	_, err := wtStore.Create(worktree.Record{
		EntityID:  bugID,
		Branch:    "bug/test-bug",
		Path:      wtDir,
		Status:    worktree.StatusActive,
		Created:   time.Now().UTC(),
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("create worktree: %v", err)
	}

	resp := invokeWriteFileWithSvc(t, repoRoot, wtStore, entitySvc, map[string]any{
		"path":    "test.txt",
		"content": "content",
	})

	if _, hasErr := resp["error"]; hasErr {
		t.Fatalf("unexpected error: %v", resp["error"])
	}

	warning, ok := resp["warning"].(string)
	if !ok || warning == "" {
		t.Fatal("expected warning but none was returned")
	}

	if !strings.Contains(warning, bugID) {
		t.Errorf("warning should contain bug ID %q, got: %q", bugID, warning)
	}
	if !strings.Contains(warning, "in-progress") {
		t.Errorf("warning should mention in-progress, got: %q", warning)
	}
	if !strings.Contains(warning, wtDir) {
		t.Errorf("warning should contain worktree path %q, got: %q", wtDir, warning)
	}

	// File should still have been written (non-blocking).
	targetPath := filepath.Join(repoRoot, "test.txt")
	if _, err := os.Stat(targetPath); err != nil {
		t.Errorf("file was not written (warning should be non-blocking): %v", err)
	}
}

// TestWriteFile_NoWarningWithoutInProgressBugs verifies:
// No warning when no bugs are in-progress.
func TestWriteFile_NoWarningWithoutInProgressBugs(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	stateRoot := t.TempDir()

	entitySvc := service.NewEntityService(stateRoot)
	createTestBug(t, entitySvc, "BUG-01CCCCCCCCCCCCC", "other-bug", "reported")

	wtStore := worktree.NewStore(t.TempDir())

	resp := invokeWriteFileWithSvc(t, repoRoot, wtStore, entitySvc, map[string]any{
		"path":    "test.txt",
		"content": "content",
	})

	if _, hasErr := resp["error"]; hasErr {
		t.Fatalf("unexpected error: %v", resp["error"])
	}
	if w, ok := resp["warning"].(string); ok && w != "" {
		t.Errorf("unexpected warning when no in-progress bugs: %q", w)
	}
}

// TestWriteFile_MultipleBugsWarningSuffix verifies FR-209:
// Multiple bugs get a count suffix.
func TestWriteFile_MultipleBugsWarningSuffix(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	stateRoot := t.TempDir()

	entitySvc := service.NewEntityService(stateRoot)

	wtStore := worktree.NewStore(t.TempDir())
	wtDir := t.TempDir()

	bugs := []string{"BUG-01DDDDDDDDDDDDD", "BUG-01EEEEEEEEEEEEE"}
	for i, bugID := range bugs {
		createTestBug(t, entitySvc, bugID, fmt.Sprintf("bug-%d", i), "in-progress")
		_, err := wtStore.Create(worktree.Record{
			EntityID:  bugID,
			Branch:    fmt.Sprintf("bug/bug-%d", i),
			Path:      wtDir,
			Status:    worktree.StatusActive,
			Created:   time.Now().UTC(),
			CreatedBy: "tester",
		})
		if err != nil {
			t.Fatalf("create worktree: %v", err)
		}
	}

	resp := invokeWriteFileWithSvc(t, repoRoot, wtStore, entitySvc, map[string]any{
		"path":    "test.txt",
		"content": "content",
	})

	if _, hasErr := resp["error"]; hasErr {
		t.Fatalf("unexpected error: %v", resp["error"])
	}

	warning, ok := resp["warning"].(string)
	if !ok || warning == "" {
		t.Fatal("expected warning but none was returned")
	}

	if !strings.Contains(warning, "and 1 other active bug worktree") {
		t.Errorf("warning should include count suffix for multiple bugs, got: %q", warning)
	}
}

// ─── AC-09: Go source content byte-fidelity ──────────────────────────────────

// TestWriteFile_ContentByteFidelity verifies AC-09:
// Content containing backticks, single quotes, and double quotes is written
// byte-for-byte without any escaping or transformation.
func TestWriteFile_ContentByteFidelity(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	store := worktree.NewStore(t.TempDir())

	// Mix of Go-relevant special characters: backticks, single/double quotes.
	content := "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(`hello 'world' \"foo\"`)\n}\n"

	resp := invokeWriteFile(t, repoRoot, store, map[string]any{
		"path":    "main.go",
		"content": content,
	})

	if _, hasErr := resp["error"]; hasErr {
		t.Fatalf("unexpected error: %v", resp["error"])
	}

	targetPath := filepath.Join(repoRoot, "main.go")
	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	if string(data) != content {
		t.Errorf("on-disk content does not match byte-for-byte\ngot:  %q\nwant: %q", string(data), content)
	}
}

// ─── AC-10: Permission bits ───────────────────────────────────────────────────

// TestWriteFile_PermissionBits verifies AC-10:
// Written files have mode 0o644.
func TestWriteFile_PermissionBits(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	store := worktree.NewStore(t.TempDir())

	resp := invokeWriteFile(t, repoRoot, store, map[string]any{
		"path":    "perm_test.txt",
		"content": "checking permissions",
	})

	if _, hasErr := resp["error"]; hasErr {
		t.Fatalf("unexpected error: %v", resp["error"])
	}

	targetPath := filepath.Join(repoRoot, "perm_test.txt")
	info, err := os.Stat(targetPath)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	const wantMode = os.FileMode(0o644)
	gotMode := info.Mode().Perm()
	if gotMode != wantMode {
		t.Errorf("file mode = %04o, want %04o", gotMode, wantMode)
	}
}
