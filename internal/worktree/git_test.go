package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// resolvePath resolves symlinks in a path for consistent comparison.
// On macOS, /var is a symlink to /private/var, which can cause path mismatches.
func resolvePath(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		// If symlink resolution fails, return original path
		return path
	}
	return resolved
}

// setupGitRepo creates a temporary Git repository for testing.
// Returns the repo path and a cleanup function.
func setupGitRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	// Initialize Git repo
	if err := runGit(dir, "init"); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// Configure user for commits
	if err := runGit(dir, "config", "user.email", "test@example.com"); err != nil {
		t.Fatalf("git config user.email failed: %v", err)
	}
	if err := runGit(dir, "config", "user.name", "Test User"); err != nil {
		t.Fatalf("git config user.name failed: %v", err)
	}

	// Create initial commit (required for worktrees)
	readmePath := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Test Repo\n"), 0o644); err != nil {
		t.Fatalf("write README.md failed: %v", err)
	}
	if err := runGit(dir, "add", "README.md"); err != nil {
		t.Fatalf("git add failed: %v", err)
	}
	if err := runGit(dir, "commit", "-m", "Initial commit"); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	return dir
}

// runGit executes a git command in the given directory.
func runGit(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	return cmd.Run()
}

// skipIfNoGit skips the test if git is not available.
func skipIfNoGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH, skipping test")
	}
}

func TestGit_CreateWorktree(t *testing.T) {
	skipIfNoGit(t)
	t.Parallel()

	repoDir := setupGitRepo(t)
	git := NewGit(repoDir)

	// Create a branch first
	if err := git.CreateBranch("feature-branch", ""); err != nil {
		t.Fatalf("CreateBranch() error = %v", err)
	}

	// Create worktree
	wtPath := filepath.Join(repoDir, ".worktrees", "feature-1")
	if err := git.CreateWorktree(wtPath, "feature-branch"); err != nil {
		t.Fatalf("CreateWorktree() error = %v", err)
	}

	// Verify worktree exists
	if _, err := os.Stat(wtPath); os.IsNotExist(err) {
		t.Error("CreateWorktree() did not create worktree directory")
	}

	// Verify it's listed
	worktrees, err := git.ListWorktrees()
	if err != nil {
		t.Fatalf("ListWorktrees() error = %v", err)
	}

	// Resolve symlinks for comparison (macOS /var -> /private/var)
	resolvedWtPath := resolvePath(t, wtPath)
	found := false
	for _, wt := range worktrees {
		if resolvePath(t, wt) == resolvedWtPath {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("ListWorktrees() = %v, want to contain %q", worktrees, wtPath)
	}
}

func TestGit_CreateWorktreeNewBranch(t *testing.T) {
	skipIfNoGit(t)
	t.Parallel()

	repoDir := setupGitRepo(t)
	git := NewGit(repoDir)

	// Create worktree with new branch
	wtPath := filepath.Join(repoDir, ".worktrees", "new-feature")
	if err := git.CreateWorktreeNewBranch(wtPath, "new-feature-branch", ""); err != nil {
		t.Fatalf("CreateWorktreeNewBranch() error = %v", err)
	}

	// Verify worktree exists
	if _, err := os.Stat(wtPath); os.IsNotExist(err) {
		t.Error("CreateWorktreeNewBranch() did not create worktree directory")
	}

	// Verify branch was created
	if !git.BranchExists("new-feature-branch") {
		t.Error("CreateWorktreeNewBranch() did not create the branch")
	}
}

func TestGit_CreateWorktreeNewBranch_FromBase(t *testing.T) {
	skipIfNoGit(t)
	t.Parallel()

	repoDir := setupGitRepo(t)
	git := NewGit(repoDir)

	// Create a base branch with a commit
	if err := git.CreateBranch("develop", ""); err != nil {
		t.Fatalf("CreateBranch(develop) error = %v", err)
	}

	// Create worktree from develop branch
	wtPath := filepath.Join(repoDir, ".worktrees", "from-develop")
	if err := git.CreateWorktreeNewBranch(wtPath, "feature-from-develop", "develop"); err != nil {
		t.Fatalf("CreateWorktreeNewBranch() error = %v", err)
	}

	// Verify worktree exists
	if _, err := os.Stat(wtPath); os.IsNotExist(err) {
		t.Error("CreateWorktreeNewBranch() did not create worktree directory")
	}

	// Verify branch was created
	if !git.BranchExists("feature-from-develop") {
		t.Error("CreateWorktreeNewBranch() did not create the branch")
	}
}

func TestGit_RemoveWorktree(t *testing.T) {
	skipIfNoGit(t)
	t.Parallel()

	repoDir := setupGitRepo(t)
	git := NewGit(repoDir)

	// Create worktree with new branch
	wtPath := filepath.Join(repoDir, ".worktrees", "to-remove")
	if err := git.CreateWorktreeNewBranch(wtPath, "remove-branch", ""); err != nil {
		t.Fatalf("CreateWorktreeNewBranch() error = %v", err)
	}

	// Remove worktree
	if err := git.RemoveWorktree(wtPath, false); err != nil {
		t.Fatalf("RemoveWorktree() error = %v", err)
	}

	// Verify worktree is gone
	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Error("RemoveWorktree() did not remove worktree directory")
	}
}

func TestGit_RemoveWorktree_Force(t *testing.T) {
	skipIfNoGit(t)
	t.Parallel()

	repoDir := setupGitRepo(t)
	git := NewGit(repoDir)

	// Create worktree with new branch
	wtPath := filepath.Join(repoDir, ".worktrees", "to-force-remove")
	if err := git.CreateWorktreeNewBranch(wtPath, "force-remove-branch", ""); err != nil {
		t.Fatalf("CreateWorktreeNewBranch() error = %v", err)
	}

	// Create uncommitted changes in worktree
	testFile := filepath.Join(wtPath, "uncommitted.txt")
	if err := os.WriteFile(testFile, []byte("uncommitted changes"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Force remove worktree
	if err := git.RemoveWorktree(wtPath, true); err != nil {
		t.Fatalf("RemoveWorktree(force=true) error = %v", err)
	}

	// Verify worktree is gone
	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Error("RemoveWorktree(force=true) did not remove worktree directory")
	}
}

func TestGit_ListWorktrees(t *testing.T) {
	skipIfNoGit(t)
	t.Parallel()

	repoDir := setupGitRepo(t)
	git := NewGit(repoDir)

	// Initial list should have just the main worktree
	worktrees, err := git.ListWorktrees()
	if err != nil {
		t.Fatalf("ListWorktrees() error = %v", err)
	}
	if len(worktrees) != 1 {
		t.Errorf("ListWorktrees() = %d worktrees, want 1", len(worktrees))
	}

	// Create additional worktrees
	wt1Path := filepath.Join(repoDir, ".worktrees", "wt1")
	wt2Path := filepath.Join(repoDir, ".worktrees", "wt2")

	if err := git.CreateWorktreeNewBranch(wt1Path, "branch-1", ""); err != nil {
		t.Fatalf("CreateWorktreeNewBranch(wt1) error = %v", err)
	}
	if err := git.CreateWorktreeNewBranch(wt2Path, "branch-2", ""); err != nil {
		t.Fatalf("CreateWorktreeNewBranch(wt2) error = %v", err)
	}

	// List should now have 3 worktrees
	worktrees, err = git.ListWorktrees()
	if err != nil {
		t.Fatalf("ListWorktrees() error = %v", err)
	}
	if len(worktrees) != 3 {
		t.Errorf("ListWorktrees() = %d worktrees, want 3", len(worktrees))
	}

	// Verify our worktrees are in the list (resolve symlinks for comparison)
	resolvedWt1 := resolvePath(t, wt1Path)
	resolvedWt2 := resolvePath(t, wt2Path)
	foundWt1, foundWt2 := false, false
	for _, wt := range worktrees {
		resolved := resolvePath(t, wt)
		if resolved == resolvedWt1 {
			foundWt1 = true
		}
		if resolved == resolvedWt2 {
			foundWt2 = true
		}
	}
	if !foundWt1 {
		t.Errorf("ListWorktrees() missing %q", wt1Path)
	}
	if !foundWt2 {
		t.Errorf("ListWorktrees() missing %q", wt2Path)
	}
}

func TestGit_BranchExists(t *testing.T) {
	skipIfNoGit(t)
	t.Parallel()

	repoDir := setupGitRepo(t)
	git := NewGit(repoDir)

	// Check for non-existent branch
	if git.BranchExists("nonexistent-branch") {
		t.Error("BranchExists() = true for non-existent branch")
	}

	// Create a branch and verify it exists
	if err := git.CreateBranch("test-branch", ""); err != nil {
		t.Fatalf("CreateBranch() error = %v", err)
	}

	if !git.BranchExists("test-branch") {
		t.Error("BranchExists() = false for existing branch")
	}
}

func TestGit_CreateBranch(t *testing.T) {
	skipIfNoGit(t)
	t.Parallel()

	repoDir := setupGitRepo(t)
	git := NewGit(repoDir)

	// Create branch from HEAD
	if err := git.CreateBranch("new-branch", ""); err != nil {
		t.Fatalf("CreateBranch() error = %v", err)
	}

	if !git.BranchExists("new-branch") {
		t.Error("CreateBranch() did not create the branch")
	}
}

func TestGit_CreateBranch_FromRef(t *testing.T) {
	skipIfNoGit(t)
	t.Parallel()

	repoDir := setupGitRepo(t)
	git := NewGit(repoDir)

	// Create a base branch
	if err := git.CreateBranch("base-branch", ""); err != nil {
		t.Fatalf("CreateBranch(base-branch) error = %v", err)
	}

	// Create branch from base-branch
	if err := git.CreateBranch("derived-branch", "base-branch"); err != nil {
		t.Fatalf("CreateBranch(derived-branch) error = %v", err)
	}

	if !git.BranchExists("derived-branch") {
		t.Error("CreateBranch() from ref did not create the branch")
	}
}

func TestGit_CreateBranch_AlreadyExists(t *testing.T) {
	skipIfNoGit(t)
	t.Parallel()

	repoDir := setupGitRepo(t)
	git := NewGit(repoDir)

	// Create branch
	if err := git.CreateBranch("existing-branch", ""); err != nil {
		t.Fatalf("CreateBranch() error = %v", err)
	}

	// Try to create again - should fail
	if err := git.CreateBranch("existing-branch", ""); err == nil {
		t.Error("CreateBranch() for existing branch should fail")
	}
}

func TestGit_DeleteBranch(t *testing.T) {
	skipIfNoGit(t)
	t.Parallel()

	repoDir := setupGitRepo(t)
	git := NewGit(repoDir)

	// Create branch
	if err := git.CreateBranch("to-delete", ""); err != nil {
		t.Fatalf("CreateBranch() error = %v", err)
	}

	// Delete branch
	if err := git.DeleteBranch("to-delete", false); err != nil {
		t.Fatalf("DeleteBranch() error = %v", err)
	}

	if git.BranchExists("to-delete") {
		t.Error("DeleteBranch() did not delete the branch")
	}
}

func TestGit_DeleteBranch_Force(t *testing.T) {
	skipIfNoGit(t)
	t.Parallel()

	repoDir := setupGitRepo(t)
	git := NewGit(repoDir)

	// Create branch and add unmerged commit
	if err := git.CreateBranch("unmerged-branch", ""); err != nil {
		t.Fatalf("CreateBranch() error = %v", err)
	}

	// Checkout branch, add commit
	if err := runGit(repoDir, "checkout", "unmerged-branch"); err != nil {
		t.Fatalf("git checkout error = %v", err)
	}

	testFile := filepath.Join(repoDir, "unmerged.txt")
	if err := os.WriteFile(testFile, []byte("unmerged content"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := runGit(repoDir, "add", "unmerged.txt"); err != nil {
		t.Fatalf("git add error = %v", err)
	}
	if err := runGit(repoDir, "commit", "-m", "Unmerged commit"); err != nil {
		t.Fatalf("git commit error = %v", err)
	}

	// Checkout back to master/main
	// Try master first, fall back to main
	if err := runGit(repoDir, "checkout", "master"); err != nil {
		if err := runGit(repoDir, "checkout", "main"); err != nil {
			t.Fatalf("git checkout main/master error = %v", err)
		}
	}

	// Force delete the unmerged branch
	if err := git.DeleteBranch("unmerged-branch", true); err != nil {
		t.Fatalf("DeleteBranch(force=true) error = %v", err)
	}

	if git.BranchExists("unmerged-branch") {
		t.Error("DeleteBranch(force=true) did not delete the branch")
	}
}

func TestGit_DeleteRemoteBranch(t *testing.T) {
	skipIfNoGit(t)
	t.Parallel()

	// Create main repo
	repoDir := setupGitRepo(t)
	git := NewGit(repoDir)

	// Create a bare repo to act as "origin"
	bareDir := t.TempDir()
	if err := runGit(bareDir, "init", "--bare"); err != nil {
		t.Fatalf("git init --bare error = %v", err)
	}

	// Add the bare repo as origin
	if err := runGit(repoDir, "remote", "add", "origin", bareDir); err != nil {
		t.Fatalf("git remote add origin error = %v", err)
	}

	// Create a branch and push it
	if err := git.CreateBranch("remote-test-branch", ""); err != nil {
		t.Fatalf("CreateBranch() error = %v", err)
	}
	if err := runGit(repoDir, "push", "origin", "remote-test-branch"); err != nil {
		t.Fatalf("git push error = %v", err)
	}

	// Verify branch exists on remote
	output, err := exec.Command("git", "-C", bareDir, "branch").Output()
	if err != nil {
		t.Fatalf("git branch in bare repo error = %v", err)
	}
	if !strings.Contains(string(output), "remote-test-branch") {
		t.Fatal("branch not pushed to remote")
	}

	// Delete the remote branch
	if err := git.DeleteRemoteBranch("origin", "remote-test-branch"); err != nil {
		t.Fatalf("DeleteRemoteBranch() error = %v", err)
	}

	// Verify branch is gone from remote
	output, err = exec.Command("git", "-C", bareDir, "branch").Output()
	if err != nil {
		t.Fatalf("git branch in bare repo error = %v", err)
	}
	if strings.Contains(string(output), "remote-test-branch") {
		t.Error("DeleteRemoteBranch() did not delete the branch from remote")
	}
}

func TestGit_DeleteRemoteBranch_NonExistent(t *testing.T) {
	skipIfNoGit(t)
	t.Parallel()

	repoDir := setupGitRepo(t)
	git := NewGit(repoDir)

	// Create a bare repo to act as "origin"
	bareDir := t.TempDir()
	if err := runGit(bareDir, "init", "--bare"); err != nil {
		t.Fatalf("git init --bare error = %v", err)
	}

	// Add the bare repo as origin
	if err := runGit(repoDir, "remote", "add", "origin", bareDir); err != nil {
		t.Fatalf("git remote add origin error = %v", err)
	}

	// Try to delete non-existent remote branch - should fail
	err := git.DeleteRemoteBranch("origin", "nonexistent-branch")
	if err == nil {
		t.Error("DeleteRemoteBranch() for non-existent branch should fail")
	}
}

func TestGit_CurrentBranch(t *testing.T) {
	skipIfNoGit(t)
	t.Parallel()

	repoDir := setupGitRepo(t)
	git := NewGit(repoDir)

	// Get current branch (should be master or main)
	branch, err := git.CurrentBranch()
	if err != nil {
		t.Fatalf("CurrentBranch() error = %v", err)
	}

	// Default branch is typically master or main
	if branch != "master" && branch != "main" {
		t.Errorf("CurrentBranch() = %q, want 'master' or 'main'", branch)
	}

	// Create and checkout a new branch
	if err := git.CreateBranch("test-current", ""); err != nil {
		t.Fatalf("CreateBranch() error = %v", err)
	}
	if err := runGit(repoDir, "checkout", "test-current"); err != nil {
		t.Fatalf("git checkout error = %v", err)
	}

	branch, err = git.CurrentBranch()
	if err != nil {
		t.Fatalf("CurrentBranch() error = %v", err)
	}
	if branch != "test-current" {
		t.Errorf("CurrentBranch() = %q, want 'test-current'", branch)
	}
}

func TestGit_PruneWorktrees(t *testing.T) {
	skipIfNoGit(t)
	t.Parallel()

	repoDir := setupGitRepo(t)
	git := NewGit(repoDir)

	// Create worktree
	wtPath := filepath.Join(repoDir, ".worktrees", "to-prune")
	if err := git.CreateWorktreeNewBranch(wtPath, "prune-branch", ""); err != nil {
		t.Fatalf("CreateWorktreeNewBranch() error = %v", err)
	}

	// Manually remove the worktree directory (simulating stale entry)
	if err := os.RemoveAll(wtPath); err != nil {
		t.Fatalf("RemoveAll() error = %v", err)
	}

	// Prune should clean up the stale entry
	if err := git.PruneWorktrees(); err != nil {
		t.Fatalf("PruneWorktrees() error = %v", err)
	}

	// List should not include the pruned worktree
	worktrees, err := git.ListWorktrees()
	if err != nil {
		t.Fatalf("ListWorktrees() error = %v", err)
	}

	resolvedWtPath := resolvePath(t, wtPath)
	for _, wt := range worktrees {
		if resolvePath(t, wt) == resolvedWtPath {
			t.Errorf("ListWorktrees() still contains pruned worktree %q", wtPath)
		}
	}
}

func TestGit_EmptyRepoDir(t *testing.T) {
	skipIfNoGit(t)
	t.Parallel()

	repoDir := setupGitRepo(t)
	git := NewGit("") // Empty RepoDir

	// Commands should run in current directory
	// We need to change to the repo dir for this test
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	defer os.Chdir(oldDir)

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	// Should work with empty RepoDir
	branch, err := git.CurrentBranch()
	if err != nil {
		t.Fatalf("CurrentBranch() with empty RepoDir error = %v", err)
	}
	if branch != "master" && branch != "main" {
		t.Errorf("CurrentBranch() = %q, want 'master' or 'main'", branch)
	}
}

func TestGit_ErrorHandling(t *testing.T) {
	skipIfNoGit(t)
	t.Parallel()

	// Create a non-git directory
	tmpDir := t.TempDir()
	git := NewGit(tmpDir)

	// Operations should fail gracefully
	_, err := git.ListWorktrees()
	if err == nil {
		t.Error("ListWorktrees() in non-git dir should fail")
	}

	err = git.CreateWorktree("/invalid/path", "nonexistent")
	if err == nil {
		t.Error("CreateWorktree() with invalid branch should fail")
	}

	_, err = git.CurrentBranch()
	if err == nil {
		t.Error("CurrentBranch() in non-git dir should fail")
	}
}

func TestGit_ErrorMessages(t *testing.T) {
	skipIfNoGit(t)
	t.Parallel()

	repoDir := setupGitRepo(t)
	git := NewGit(repoDir)

	// Try to create worktree with non-existent branch
	wtPath := filepath.Join(repoDir, ".worktrees", "will-fail")
	err := git.CreateWorktree(wtPath, "nonexistent-branch")
	if err == nil {
		t.Fatal("CreateWorktree() with non-existent branch should fail")
	}

	// Error should contain helpful information
	errStr := err.Error()
	if !strings.Contains(errStr, "git worktree add") {
		t.Errorf("Error message should mention 'git worktree add', got: %s", errStr)
	}
}
