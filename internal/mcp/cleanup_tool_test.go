package mcp

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/sambeau/kanbanzai/internal/config"
	"github.com/sambeau/kanbanzai/internal/worktree"
)

// setupGitRepoForCleanup creates a temporary git repository with an initial commit
// and a worktree, then marks the worktree as merged so it is eligible for cleanup.
func setupGitRepoForCleanup(t *testing.T, wtRelPath, branchName string) (repoDir string, wtAbsPath string) {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH, skipping test")
	}

	repoDir = t.TempDir()

	runGitCmd := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	runGitCmd("init")
	runGitCmd("config", "user.email", "test@test.com")
	runGitCmd("config", "user.name", "Test")

	readme := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(readme, []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	runGitCmd("add", "README.md")
	runGitCmd("commit", "-m", "init")

	wtAbsPath = filepath.Join(repoDir, wtRelPath)
	runGitCmd("worktree", "add", "-b", branchName, wtAbsPath)

	return repoDir, wtAbsPath
}

// TestCleanup_GraphProjectNote verifies AC-017:
// cleanup(action: execute) on a worktree with non-empty GraphProject → output
// contains a note referencing the project name and instructing delete_project.
func TestCleanup_GraphProjectNote(t *testing.T) {
	t.Parallel()

	repoDir, wtAbsPath := setupGitRepoForCleanup(t, "wt-cleanup-gp", "cleanup-branch")
	gitOps := worktree.NewGit(repoDir)
	store := worktree.NewStore(t.TempDir())
	cfg := &config.CleanupConfig{}

	// Create a merged worktree record with GraphProject set and cleanup_after in the past.
	mergedAt := time.Now().Add(-48 * time.Hour)
	cleanupAfter := time.Now().Add(-24 * time.Hour) // past grace period
	record := worktree.Record{
		EntityID:     "FEAT-01CCCCCCCCCCCCC",
		Branch:       "cleanup-branch",
		Path:         wtAbsPath,
		Status:       worktree.StatusMerged,
		Created:      time.Now().Add(-72 * time.Hour),
		CreatedBy:    "tester",
		MergedAt:     &mergedAt,
		CleanupAfter: &cleanupAfter,
		GraphProject: "kanbanzai-FEAT-XXX",
	}

	created, err := store.Create(record)
	if err != nil {
		t.Fatalf("store.Create: %v", err)
	}

	handler := cleanupExecuteAction(store, gitOps, cfg)
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"worktree_id": created.ID,
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("cleanupExecuteAction: %v", err)
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var resp map[string]any
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	cleaned, ok := resp["cleaned"].([]any)
	if !ok || len(cleaned) == 0 {
		t.Fatalf("expected non-empty cleaned array, got: %v", resp)
	}

	entry, ok := cleaned[0].(map[string]any)
	if !ok {
		t.Fatalf("expected map entry in cleaned, got: %T", cleaned[0])
	}

	note, ok := entry["graph_project_note"].(string)
	if !ok {
		t.Fatalf("expected graph_project_note in cleaned entry, got: %v", entry)
	}
	if !containsIgnoreCase(note, "kanbanzai-FEAT-XXX") {
		t.Errorf("graph_project_note should reference project name, got: %s", note)
	}
	if !containsIgnoreCase(note, "delete_project") {
		t.Errorf("graph_project_note should mention delete_project, got: %s", note)
	}
}

// TestCleanup_NoGraphProjectNote verifies the inverse of AC-017:
// cleanup on a worktree with empty GraphProject → no graph_project_note.
func TestCleanup_NoGraphProjectNote(t *testing.T) {
	t.Parallel()

	repoDir, wtAbsPath := setupGitRepoForCleanup(t, "wt-cleanup-no-gp", "cleanup-no-gp-branch")
	gitOps := worktree.NewGit(repoDir)
	store := worktree.NewStore(t.TempDir())
	cfg := &config.CleanupConfig{}

	mergedAt := time.Now().Add(-48 * time.Hour)
	cleanupAfter := time.Now().Add(-24 * time.Hour)
	record := worktree.Record{
		EntityID:     "FEAT-01DDDDDDDDDDDDD",
		Branch:       "cleanup-no-gp-branch",
		Path:         wtAbsPath,
		Status:       worktree.StatusMerged,
		Created:      time.Now().Add(-72 * time.Hour),
		CreatedBy:    "tester",
		MergedAt:     &mergedAt,
		CleanupAfter: &cleanupAfter,
		GraphProject: "", // empty — no note expected
	}

	created, err := store.Create(record)
	if err != nil {
		t.Fatalf("store.Create: %v", err)
	}

	handler := cleanupExecuteAction(store, gitOps, cfg)
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"worktree_id": created.ID,
	}

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("cleanupExecuteAction: %v", err)
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var resp map[string]any
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	cleaned, ok := resp["cleaned"].([]any)
	if !ok || len(cleaned) == 0 {
		t.Fatalf("expected non-empty cleaned array, got: %v", resp)
	}

	entry, ok := cleaned[0].(map[string]any)
	if !ok {
		t.Fatalf("expected map entry in cleaned, got: %T", cleaned[0])
	}

	if _, hasNote := entry["graph_project_note"]; hasNote {
		t.Errorf("unexpected graph_project_note for empty GraphProject: %v", entry)
	}
}

// AC-018 note: cleanup notes are purely additive string fields — they do not call
// codebase_memory_mcp. When the MCP is unavailable, GraphProject is empty and no
// note is emitted. All non-graph behaviour remains identical.
