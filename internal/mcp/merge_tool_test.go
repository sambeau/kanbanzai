package mcp

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sambeau/kanbanzai/internal/config"
	"github.com/sambeau/kanbanzai/internal/git"
	"github.com/sambeau/kanbanzai/internal/service"
	"github.com/sambeau/kanbanzai/internal/worktree"
)

// newEmptyWorktreeStore creates a Store backed by an empty temp directory,
// so GetByEntityID always returns worktree.ErrNotFound.
func newEmptyWorktreeStore(t *testing.T) *worktree.Store {
	t.Helper()
	return worktree.NewStore(t.TempDir())
}

// newBrokenWorktreeStore creates a Store whose worktrees directory is actually
// a regular file, causing List() (and thus GetByEntityID) to return a non-ErrNotFound error.
func newBrokenWorktreeStore(t *testing.T) *worktree.Store {
	t.Helper()
	root := t.TempDir()
	// Create a file where the worktrees directory should be.
	wtDir := filepath.Join(root, worktree.WorktreesDir)
	if err := os.WriteFile(wtDir, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("newBrokenWorktreeStore: %v", err)
	}
	return worktree.NewStore(root)
}

// ─── checkMergeReadiness ──────────────────────────────────────────────────────

func TestCheckMergeReadiness_NoWorktree_NotApplicable(t *testing.T) {
	t.Parallel()

	store := newEmptyWorktreeStore(t)
	entitySvc := service.NewEntityService(t.TempDir())
	thresholds := git.BranchThresholds{}

	result, err := checkMergeReadiness(
		context.Background(),
		store,
		entitySvc,
		nil, // docSvc
		t.TempDir(),
		thresholds,
		nil,
		"FEAT-001",
	)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected a result map, got nil")
	}
	if result["status"] != "not_applicable" {
		t.Errorf("status = %q, want not_applicable; full result: %v", result["status"], result)
	}
	if result["reason"] == "" {
		t.Error("expected non-empty reason field")
	}
	if result["recommendation"] == "" {
		t.Error("expected non-empty recommendation field")
	}
}

func TestCheckMergeReadiness_InvalidEntityID_ReturnsError(t *testing.T) {
	t.Parallel()

	store := newEmptyWorktreeStore(t)
	entitySvc := service.NewEntityService(t.TempDir())
	thresholds := git.BranchThresholds{}

	// A TASK- prefix is not a feature or bug — should error, not return not_applicable.
	_, err := checkMergeReadiness(
		context.Background(),
		store,
		entitySvc,
		nil, // docSvc
		t.TempDir(),
		thresholds,
		nil,
		"TASK-001",
	)

	if err == nil {
		t.Fatal("expected an error for invalid entity ID, got nil")
	}
}

func TestCheckMergeReadiness_InvalidEntityID_PlainString_ReturnsError(t *testing.T) {
	t.Parallel()

	store := newEmptyWorktreeStore(t)
	entitySvc := service.NewEntityService(t.TempDir())
	thresholds := git.BranchThresholds{}

	_, err := checkMergeReadiness(
		context.Background(),
		store,
		entitySvc,
		nil, // docSvc
		t.TempDir(),
		thresholds,
		nil,
		"not-an-entity",
	)

	if err == nil {
		t.Fatal("expected an error for invalid entity ID, got nil")
	}
}

func TestCheckMergeReadiness_StoreError_Propagated(t *testing.T) {
	t.Parallel()

	store := newBrokenWorktreeStore(t)
	entitySvc := service.NewEntityService(t.TempDir())
	thresholds := git.BranchThresholds{}

	_, err := checkMergeReadiness(
		context.Background(),
		store,
		entitySvc,
		nil, // docSvc
		t.TempDir(),
		thresholds,
		nil,
		"FEAT-001",
	)

	if err == nil {
		t.Fatal("expected a store error to propagate, got nil")
	}
}

// ─── executeMerge ─────────────────────────────────────────────────────────────

func TestExecuteMerge_NoWorktree_Skipped(t *testing.T) {
	t.Parallel()

	store := newEmptyWorktreeStore(t)
	entitySvc := service.NewEntityService(t.TempDir())
	thresholds := git.BranchThresholds{}

	result, err := executeMerge(
		store,
		entitySvc,
		nil, // docSvc
		t.TempDir(),
		thresholds,
		nil,
		"FEAT-001",
		false,
		"",
		worktree.MergeStrategySquash,
		true,
	)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected a result map, got nil")
	}
	if result["status"] != "skipped" {
		t.Errorf("status = %q, want skipped; full result: %v", result["status"], result)
	}
	if result["reason"] == "" {
		t.Error("expected non-empty reason field")
	}
}

func TestExecuteMerge_NoWorktree_BugEntity_Skipped(t *testing.T) {
	t.Parallel()

	store := newEmptyWorktreeStore(t)
	entitySvc := service.NewEntityService(t.TempDir())
	thresholds := git.BranchThresholds{}

	result, err := executeMerge(
		store,
		entitySvc,
		nil, // docSvc
		t.TempDir(),
		thresholds,
		nil,
		"BUG-001",
		false,
		"",
		worktree.MergeStrategySquash,
		true,
	)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result["status"] != "skipped" {
		t.Errorf("status = %q, want skipped", result["status"])
	}
}

func TestExecuteMerge_InvalidEntityID_ReturnsError(t *testing.T) {
	t.Parallel()

	store := newEmptyWorktreeStore(t)
	entitySvc := service.NewEntityService(t.TempDir())
	thresholds := git.BranchThresholds{}

	_, err := executeMerge(
		store,
		entitySvc,
		nil, // docSvc
		t.TempDir(),
		thresholds,
		nil,
		"TASK-001",
		false,
		"",
		worktree.MergeStrategySquash,
		true,
	)

	if err == nil {
		t.Fatal("expected an error for invalid entity ID, got nil")
	}
}

func TestExecuteMerge_StoreError_Propagated(t *testing.T) {
	t.Parallel()

	store := newBrokenWorktreeStore(t)
	entitySvc := service.NewEntityService(t.TempDir())
	thresholds := git.BranchThresholds{}

	_, err := executeMerge(
		store,
		entitySvc,
		nil, // docSvc
		t.TempDir(),
		thresholds,
		nil,
		"FEAT-001",
		false,
		"",
		worktree.MergeStrategySquash,
		true,
	)

	if err == nil {
		t.Fatal("expected a store error to propagate, got nil")
	}
}

// ─── merge tool via action handler ───────────────────────────────────────────

func TestMergeCheckAction_NoWorktree_NotApplicable(t *testing.T) {
	t.Parallel()

	store := newEmptyWorktreeStore(t)
	entitySvc := service.NewEntityService(t.TempDir())
	thresholds := git.BranchThresholds{}
	localConfig := (*config.LocalConfig)(nil)

	tool := mergeTool(store, entitySvc, nil, t.TempDir(), thresholds, localConfig)

	req := makeRequest(map[string]any{
		"action":    "check",
		"entity_id": "FEAT-NOWORKTREE",
	})
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	text := extractText(t, result)
	if text == "" {
		t.Fatal("expected non-empty result text")
	}
}

func TestMergeExecuteAction_NoWorktree_Skipped(t *testing.T) {
	t.Parallel()

	store := newEmptyWorktreeStore(t)
	entitySvc := service.NewEntityService(t.TempDir())
	thresholds := git.BranchThresholds{}
	localConfig := (*config.LocalConfig)(nil)

	// Override commit func to avoid git operations.
	old := mergeCommitFunc
	mergeCommitFunc = func(_, _ string) (bool, error) { return false, nil }
	t.Cleanup(func() { mergeCommitFunc = old })

	tool := mergeTool(store, entitySvc, nil, t.TempDir(), thresholds, localConfig)

	req := makeRequest(map[string]any{
		"action":    "execute",
		"entity_id": "FEAT-NOWORKTREE",
	})
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	text := extractText(t, result)
	if text == "" {
		t.Fatal("expected non-empty result text")
	}
}
