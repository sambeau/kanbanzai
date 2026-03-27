package health

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"kanbanzai/internal/worktree"
)

// setupGitRepo creates a minimal git repo with a "main" branch and one commit.
func setupGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v\nOutput: %s", args, err, out)
		}
	}

	git("init")
	git("config", "user.email", "test@example.com")
	git("config", "user.name", "Test User")
	git("checkout", "-b", "main")

	readme := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readme, []byte("# Test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git("add", "README.md")
	git("commit", "-m", "Initial commit")

	return dir
}

func TestCheckWorktreeBranchMerged_EmptyRepoPath(t *testing.T) {
	t.Parallel()

	worktrees := []worktree.Record{
		{
			ID:       "WT-001",
			EntityID: "FEAT-001",
			Branch:   "feature/test",
			Status:   worktree.StatusActive,
			Created:  time.Now(),
		},
	}

	result := CheckWorktreeBranchMerged("", worktrees)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
	}
}

func TestCheckWorktreeBranchMerged_NoWorktrees(t *testing.T) {
	t.Parallel()

	repo := setupGitRepo(t)

	result := CheckWorktreeBranchMerged(repo, nil)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
	}
}

func TestCheckWorktreeBranchMerged_ActiveBranchNotMerged(t *testing.T) {
	t.Parallel()

	repo := setupGitRepo(t)

	// Create a feature branch with an extra commit not on main.
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v\nOutput: %s", args, err, out)
		}
	}

	git("checkout", "-b", "feature/work")
	f := filepath.Join(repo, "work.txt")
	if err := os.WriteFile(f, []byte("work\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git("add", "work.txt")
	git("commit", "-m", "Feature work")
	git("checkout", "main")

	worktrees := []worktree.Record{
		{
			ID:       "WT-001",
			EntityID: "FEAT-001",
			Branch:   "feature/work",
			Status:   worktree.StatusActive,
			Created:  time.Now(),
		},
	}

	result := CheckWorktreeBranchMerged(repo, worktrees)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
	}
}

func TestCheckWorktreeBranchMerged_ActiveBranchAlreadyMerged(t *testing.T) {
	t.Parallel()

	repo := setupGitRepo(t)

	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v\nOutput: %s", args, err, out)
		}
	}

	// Create a branch, add a commit, merge it into main.
	git("checkout", "-b", "feature/merged")
	f := filepath.Join(repo, "merged.txt")
	if err := os.WriteFile(f, []byte("done\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git("add", "merged.txt")
	git("commit", "-m", "Feature done")
	git("checkout", "main")
	git("merge", "--no-ff", "feature/merged", "-m", "Merge feature/merged")

	worktrees := []worktree.Record{
		{
			ID:       "WT-001",
			EntityID: "FEAT-001",
			Branch:   "feature/merged",
			Status:   worktree.StatusActive,
			Created:  time.Now(),
		},
	}

	result := CheckWorktreeBranchMerged(repo, worktrees)

	if result.Status != SeverityWarning {
		t.Errorf("Status = %v, want %v", result.Status, SeverityWarning)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("len(Issues) = %d, want 1", len(result.Issues))
	}

	issue := result.Issues[0]
	if issue.Severity != SeverityWarning {
		t.Errorf("Issue.Severity = %v, want %v", issue.Severity, SeverityWarning)
	}
	if issue.EntityID != "FEAT-001" {
		t.Errorf("Issue.EntityID = %q, want %q", issue.EntityID, "FEAT-001")
	}
	if got, want := issue.Message, "worktree for FEAT-001 is active but branch feature/merged is already merged into main"; got != want {
		t.Errorf("Issue.Message = %q, want %q", got, want)
	}
}

func TestCheckWorktreeBranchMerged_SkipsNonActiveWorktrees(t *testing.T) {
	t.Parallel()

	repo := setupGitRepo(t)

	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v\nOutput: %s", args, err, out)
		}
	}

	// Create a branch and merge it — but mark worktree as "merged", not "active".
	git("checkout", "-b", "feature/old")
	f := filepath.Join(repo, "old.txt")
	if err := os.WriteFile(f, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git("add", "old.txt")
	git("commit", "-m", "Old feature")
	git("checkout", "main")
	git("merge", "--no-ff", "feature/old", "-m", "Merge feature/old")

	worktrees := []worktree.Record{
		{
			ID:       "WT-001",
			EntityID: "FEAT-001",
			Branch:   "feature/old",
			Status:   worktree.StatusMerged,
			Created:  time.Now(),
		},
		{
			ID:       "WT-002",
			EntityID: "FEAT-002",
			Branch:   "feature/old",
			Status:   worktree.StatusAbandoned,
			Created:  time.Now(),
		},
	}

	result := CheckWorktreeBranchMerged(repo, worktrees)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
	}
}

func TestCheckWorktreeBranchMerged_SkipsBranchWithEmptyName(t *testing.T) {
	t.Parallel()

	repo := setupGitRepo(t)

	worktrees := []worktree.Record{
		{
			ID:       "WT-001",
			EntityID: "FEAT-001",
			Branch:   "",
			Status:   worktree.StatusActive,
			Created:  time.Now(),
		},
	}

	result := CheckWorktreeBranchMerged(repo, worktrees)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
	}
}

func TestCheckWorktreeBranchMerged_GracefulOnMissingBranch(t *testing.T) {
	t.Parallel()

	repo := setupGitRepo(t)

	worktrees := []worktree.Record{
		{
			ID:       "WT-001",
			EntityID: "FEAT-001",
			Branch:   "feature/nonexistent",
			Status:   worktree.StatusActive,
			Created:  time.Now(),
		},
	}

	result := CheckWorktreeBranchMerged(repo, worktrees)

	// Should skip gracefully, not error
	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
	}
}

func TestCheckWorktreeBranchMerged_GracefulOnNonGitDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir() // Not a git repository

	worktrees := []worktree.Record{
		{
			ID:       "WT-001",
			EntityID: "FEAT-001",
			Branch:   "feature/test",
			Status:   worktree.StatusActive,
			Created:  time.Now(),
		},
	}

	result := CheckWorktreeBranchMerged(dir, worktrees)

	// Should skip gracefully when not a git repo
	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
	}
}

func TestCheckWorktreeBranchMerged_MultipleBranches(t *testing.T) {
	t.Parallel()

	repo := setupGitRepo(t)

	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v\nOutput: %s", args, err, out)
		}
	}

	// Create and merge branch A.
	git("checkout", "-b", "feature/a")
	fa := filepath.Join(repo, "a.txt")
	if err := os.WriteFile(fa, []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git("add", "a.txt")
	git("commit", "-m", "Feature A")
	git("checkout", "main")
	git("merge", "--no-ff", "feature/a", "-m", "Merge A")

	// Create branch B with an extra commit not merged.
	git("checkout", "-b", "feature/b")
	fb := filepath.Join(repo, "b.txt")
	if err := os.WriteFile(fb, []byte("b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git("add", "b.txt")
	git("commit", "-m", "Feature B")
	git("checkout", "main")

	worktrees := []worktree.Record{
		{
			ID:       "WT-001",
			EntityID: "FEAT-001",
			Branch:   "feature/a",
			Status:   worktree.StatusActive,
			Created:  time.Now(),
		},
		{
			ID:       "WT-002",
			EntityID: "FEAT-002",
			Branch:   "feature/b",
			Status:   worktree.StatusActive,
			Created:  time.Now(),
		},
	}

	result := CheckWorktreeBranchMerged(repo, worktrees)

	if result.Status != SeverityWarning {
		t.Errorf("Status = %v, want %v", result.Status, SeverityWarning)
	}
	// Only feature/a should be flagged.
	if len(result.Issues) != 1 {
		t.Fatalf("len(Issues) = %d, want 1", len(result.Issues))
	}
	if result.Issues[0].EntityID != "FEAT-001" {
		t.Errorf("Issue.EntityID = %q, want %q", result.Issues[0].EntityID, "FEAT-001")
	}
}

func TestIsBranchAncestorOf_Merged(t *testing.T) {
	t.Parallel()

	repo := setupGitRepo(t)

	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v\nOutput: %s", args, err, out)
		}
	}

	git("checkout", "-b", "feature/x")
	f := filepath.Join(repo, "x.txt")
	if err := os.WriteFile(f, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git("add", "x.txt")
	git("commit", "-m", "X")
	git("checkout", "main")
	git("merge", "--no-ff", "feature/x", "-m", "Merge X")

	merged, err := isBranchAncestorOf(repo, "feature/x", "main")
	if err != nil {
		t.Fatalf("isBranchAncestorOf() error = %v", err)
	}
	if !merged {
		t.Error("isBranchAncestorOf() = false, want true")
	}
}

func TestIsBranchAncestorOf_NotMerged(t *testing.T) {
	t.Parallel()

	repo := setupGitRepo(t)

	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v\nOutput: %s", args, err, out)
		}
	}

	git("checkout", "-b", "feature/y")
	f := filepath.Join(repo, "y.txt")
	if err := os.WriteFile(f, []byte("y\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git("add", "y.txt")
	git("commit", "-m", "Y")
	git("checkout", "main")

	merged, err := isBranchAncestorOf(repo, "feature/y", "main")
	if err != nil {
		t.Fatalf("isBranchAncestorOf() error = %v", err)
	}
	if merged {
		t.Error("isBranchAncestorOf() = true, want false")
	}
}

func TestIsBranchAncestorOf_BranchNotFound(t *testing.T) {
	t.Parallel()

	repo := setupGitRepo(t)

	_, err := isBranchAncestorOf(repo, "feature/ghost", "main")
	if err == nil {
		t.Error("isBranchAncestorOf() expected error for missing branch, got nil")
	}
}

func TestIsBranchAncestorOf_NotARepo(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	_, err := isBranchAncestorOf(dir, "feature/x", "main")
	if err == nil {
		t.Error("isBranchAncestorOf() expected error for non-repo, got nil")
	}
}
