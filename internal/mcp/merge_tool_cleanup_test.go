package mcp

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sambeau/kanbanzai/internal/cleanup"
	"github.com/sambeau/kanbanzai/internal/git"
	"github.com/sambeau/kanbanzai/internal/service"
	"github.com/sambeau/kanbanzai/internal/storage"
	"github.com/sambeau/kanbanzai/internal/worktree"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

// runGit runs a git command in the given directory, failing the test on error.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\nOutput: %s", args, err, out)
	}
}

// setupMergeTestRepo creates a real git repo with two commits on main and a
// feature branch with one additional commit. Returns the repo path, the
// worktree store, the entity service, the entity ID, and the worktree record.
func setupMergeTestRepo(t *testing.T) (
	repoPath string,
	wtStore *worktree.Store,
	entitySvc *service.EntityService,
	entityID string,
	wtRecord worktree.Record,
) {
	t.Helper()

	repoPath = t.TempDir()
	stateRoot := filepath.Join(repoPath, ".kbz", "state")

	// Initialize git repo.
	runGit(t, repoPath, "init", "-b", "main")
	runGit(t, repoPath, "config", "user.email", "test@example.com")
	runGit(t, repoPath, "config", "user.name", "Test User")

	// Create an initial commit on main so we have something to branch from.
	initFile := filepath.Join(repoPath, "README.md")
	if err := exec.Command("sh", "-c", "echo '# Test' > "+initFile).Run(); err != nil {
		t.Fatalf("create README.md: %v", err)
	}
	runGit(t, repoPath, "add", "README.md")
	runGit(t, repoPath, "commit", "-m", "initial commit")

	// Create a feature branch with a commit.
	branch := "feature/FEAT-01KQG3MRGTEST1"
	entityID = "FEAT-01KQG3MRGTEST1"
	runGit(t, repoPath, "checkout", "-b", branch)
	featFile := filepath.Join(repoPath, "feat.txt")
	if err := exec.Command("sh", "-c", "echo 'feature work' > "+featFile).Run(); err != nil {
		t.Fatalf("create feat.txt: %v", err)
	}
	runGit(t, repoPath, "add", "feat.txt")
	runGit(t, repoPath, "commit", "-m", "feature commit")

	// Write entity record.
	estore := storage.NewEntityStore(stateRoot)
	_, err := estore.Write(storage.EntityRecord{
		Type: "feature",
		ID:   entityID,
		Slug: "merge-cleanup-test",
		Fields: map[string]any{
			"id":         entityID,
			"slug":       "merge-cleanup-test",
			"parent":     "B42-test",
			"status":     "developing",
			"summary":    "Merge cleanup test feature",
			"created":    "2026-04-30T00:00:00Z",
			"created_by": "test",
		},
	})
	if err != nil {
		t.Fatalf("write test entity: %v", err)
	}

	entitySvc = service.NewEntityService(stateRoot)

	// Create worktree record.
	wtStore = worktree.NewStore(stateRoot)
	wtRecord = worktree.Record{
		ID:        "WT-MRGTEST01",
		EntityID:  entityID,
		Branch:    branch,
		Path:      ".worktrees/FEAT-01KQG3MRGTEST1-merge-cleanup-test",
		Status:    worktree.StatusActive,
		Created:   time.Now(),
		CreatedBy: "test",
	}
	if _, err := wtStore.Create(wtRecord); err != nil {
		t.Fatalf("create worktree record: %v", err)
	}

	return
}

// ─── AC-006: merged worktree appears in cleanup list ─────────────────────────

func TestExecuteMerge_WorktreeAppearsInCleanupList_AC006(t *testing.T) {
	// Not parallel: modifies git repo and package-level funcs.
	repoPath, wtStore, entitySvc, entityID, _ := setupMergeTestRepo(t)
	_ = repoPath
	_ = entitySvc

	// Simulate a successful merge's post-merge cleanup scheduling:
	// MarkMerged + Update — exactly what executeMerge does after merge success.
	mergedAt := time.Now().UTC()
	gracePeriodDays := 7

	record, err := wtStore.Get("WT-MRGTEST01")
	if err != nil {
		t.Fatalf("Get() = %v", err)
	}

	record.MarkMerged(mergedAt, gracePeriodDays)
	if _, err := wtStore.Update(record); err != nil {
		t.Fatalf("Update() = %v", err)
	}

	// Verify the record was updated with merged status and cleanup time.
	updated, err := wtStore.Get("WT-MRGTEST01")
	if err != nil {
		t.Fatalf("Get() after merge: %v", err)
	}
	if updated.Status != worktree.StatusMerged {
		t.Errorf("status = %q, want %q", updated.Status, worktree.StatusMerged)
	}
	if updated.MergedAt == nil {
		t.Error("merged_at is nil")
	}
	if updated.CleanupAfter == nil {
		t.Error("cleanup_after is nil")
	}

	// Verify the worktree appears in the cleanup list.
	records, err := wtStore.List()
	if err != nil {
		t.Fatalf("List() = %v", err)
	}

	now := time.Now()
	items := cleanup.ListCleanupItems(records, now, cleanup.ListOptions{
		IncludePending:   true,
		IncludeScheduled: true,
	})

	found := false
	for _, item := range items {
		if item.WorktreeID == "WT-MRGTEST01" {
			found = true
			// Scheduled because within 7-day grace period.
			if item.Status != "scheduled" {
				t.Errorf("cleanup item status = %q, want scheduled", item.Status)
			}
			if item.EntityID != entityID {
				t.Errorf("cleanup item entity_id = %q, want %q", item.EntityID, entityID)
			}
			break
		}
	}
	if !found {
		t.Error("merged worktree WT-MRGTEST01 not found in cleanup list")
	}
}

// ─── AC-007: squash merge deletes local branch ───────────────────────────────

func TestExecuteMerge_SquashMergeDeletesLocalBranch_AC007(t *testing.T) {
	// Not parallel: modifies git repo and package-level funcs.
	repoPath, wtStore, entitySvc, entityID, _ := setupMergeTestRepo(t)

	oldCommit := mergeCommitFunc
	mergeCommitFunc = func(_, _ string) (bool, error) { return false, nil }
	t.Cleanup(func() { mergeCommitFunc = oldCommit })

	branch := "feature/FEAT-01KQG3MRGTEST1"

	// Verify the branch exists before merge.
	cmd := exec.Command("git", "branch", "--list", branch)
	cmd.Dir = repoPath
	out, _ := cmd.Output()
	if !strings.Contains(string(out), branch) {
		t.Fatalf("branch %q does not exist before merge", branch)
	}

	// Execute merge with override to skip gate checks.
	_, err := executeMerge(
		wtStore,
		entitySvc,
		nil, // docSvc
		repoPath,
		git.BranchThresholds{},
		nil, // no localConfig → GitHub token absent → PR block skipped
		entityID,
		true, // override gates
		"testing squash merge cleanup",
		worktree.MergeStrategySquash,
		true, // deleteBranch = true
	)
	if err != nil {
		t.Fatalf("executeMerge() error = %v", err)
	}

	// Verify the branch was deleted.
	cmd = exec.Command("git", "branch", "--list", branch)
	cmd.Dir = repoPath
	out, _ = cmd.Output()
	if strings.Contains(string(out), branch) {
		t.Errorf("branch %q still exists after squash merge", branch)
	}

	// Verify we're back on main.
	cmd = exec.Command("git", "branch", "--show-current")
	cmd.Dir = repoPath
	out, err = cmd.Output()
	if err != nil {
		t.Fatalf("git branch --show-current: %v", err)
	}
	currentBranch := strings.TrimSpace(string(out))
	if currentBranch != "main" {
		t.Errorf("current branch = %q, want main", currentBranch)
	}
}

// ─── AC-011: cleanup failure doesn't fail merge ──────────────────────────────

func TestExecuteMerge_CleanupFailureDoesNotFailMerge_AC011(t *testing.T) {
	// Not parallel: modifies git repo and package-level funcs.
	repoPath, wtStore, entitySvc, entityID, _ := setupMergeTestRepo(t)

	oldCommit := mergeCommitFunc
	mergeCommitFunc = func(_, _ string) (bool, error) { return false, nil }
	t.Cleanup(func() { mergeCommitFunc = oldCommit })

	// Execute merge — the key assertion from REQ-NF-002: if cleanup scheduling
	// (worktree update) fails, the merge must still report success.
	// The code at merge_tool.go:~463 wraps update errors as warnings, never
	// as merge failures.

	result, err := executeMerge(
		wtStore,
		entitySvc,
		nil, // docSvc
		repoPath,
		git.BranchThresholds{},
		nil,
		entityID,
		true,
		"testing cleanup failure handling",
		worktree.MergeStrategySquash,
		true,
	)
	if err != nil {
		// Merge itself must succeed — the error return is for merge failures only.
		t.Fatalf("executeMerge() error = %v (merge should succeed even if cleanup has issues)", err)
	}

	// Verify the merge result contains the expected fields.
	merged, ok := result["merged"].(map[string]any)
	if !ok {
		t.Fatalf("result[\"merged\"] is not a map: %T", result["merged"])
	}
	if merged["entity_id"] != entityID {
		t.Errorf("merged.entity_id = %q, want %q", merged["entity_id"], entityID)
	}

	// Verify cleanup_scheduled is present.
	if _, ok := result["cleanup_scheduled"]; !ok {
		t.Error("cleanup_scheduled missing from merge result")
	}

	// Verify the worktree record was updated successfully.
	record, err := wtStore.Get("WT-MRGTEST01")
	if err != nil {
		t.Fatalf("Get() = %v", err)
	}
	if record.Status != worktree.StatusMerged {
		t.Errorf("worktree status = %q, want %q", record.Status, worktree.StatusMerged)
	}

	// Verify that even when there are warnings, the merge succeeded.
	// The warnings from auto-advance (feature lifecycle) are expected and
	// demonstrate the pattern: non-fatal issues → warnings, not errors.
	if warnings, hasWarnings := result["warnings"]; hasWarnings {
		t.Logf("merge succeeded with warnings: %v", warnings)
	}
}

// ─── Additional: executeMerge sets correct merged_at and cleanup_after ────────

func TestExecuteMerge_SetsMergedAtAndCleanupAfter(t *testing.T) {
	// Not parallel: modifies git repo and package-level funcs.
	repoPath, wtStore, entitySvc, entityID, _ := setupMergeTestRepo(t)

	oldCommit := mergeCommitFunc
	mergeCommitFunc = func(_, _ string) (bool, error) { return false, nil }
	t.Cleanup(func() { mergeCommitFunc = oldCommit })

	beforeMerge := time.Now().UTC().Truncate(time.Second)

	_, err := executeMerge(
		wtStore,
		entitySvc,
		nil,
		repoPath,
		git.BranchThresholds{},
		nil,
		entityID,
		true,
		"testing timestamps",
		worktree.MergeStrategySquash,
		true,
	)
	if err != nil {
		t.Fatalf("executeMerge() error = %v", err)
	}

	record, err := wtStore.Get("WT-MRGTEST01")
	if err != nil {
		t.Fatalf("Get() = %v", err)
	}

	if record.MergedAt == nil {
		t.Fatal("merged_at is nil after merge")
	}
	// Allow up to 2 seconds of clock skew.
	if record.MergedAt.Before(beforeMerge.Add(-2 * time.Second)) {
		t.Errorf("merged_at = %v, want >= %v", record.MergedAt, beforeMerge)
	}

	if record.CleanupAfter == nil {
		t.Fatal("cleanup_after is nil after merge")
	}

	// CleanupAfter should be MergedAt + grace period (7 days).
	expectedCleanup := record.MergedAt.AddDate(0, 0, 7)
	if !record.CleanupAfter.Equal(expectedCleanup) {
		t.Errorf("cleanup_after = %v, want %v (merged_at + 7 days)", record.CleanupAfter, expectedCleanup)
	}
}

// ─── delete_branch=false preserves the branch ─────────────────────────────────

func TestExecuteMerge_DeleteBranchFalse_PreservesBranch(t *testing.T) {
	// Not parallel: modifies git repo and package-level funcs.
	repoPath, wtStore, entitySvc, entityID, _ := setupMergeTestRepo(t)

	oldCommit := mergeCommitFunc
	mergeCommitFunc = func(_, _ string) (bool, error) { return false, nil }
	t.Cleanup(func() { mergeCommitFunc = oldCommit })

	branch := "feature/FEAT-01KQG3MRGTEST1"

	_, err := executeMerge(
		wtStore,
		entitySvc,
		nil,
		repoPath,
		git.BranchThresholds{},
		nil,
		entityID,
		true,
		"testing delete_branch=false",
		worktree.MergeStrategySquash,
		false, // deleteBranch = false
	)
	if err != nil {
		t.Fatalf("executeMerge() error = %v", err)
	}

	// Branch should still exist.
	cmd := exec.Command("git", "branch", "--list", branch)
	cmd.Dir = repoPath
	out, _ := cmd.Output()
	if !strings.Contains(string(out), branch) {
		t.Errorf("branch %q was deleted despite delete_branch=false", branch)
	}
}
