package git

import (
	"os/exec"
	"strings"
	"testing"
	"time"
)

// setupBranchTestRepo creates a test repo with main branch and multiple commits.
// Returns the repo path. Use t.Cleanup to ensure cleanup.
func setupBranchTestRepo(t *testing.T) string {
	t.Helper()
	repo := setupTestRepo(t)

	// Rename default branch to "main" for consistency
	runGit(t, repo, "branch", "-M", "main")

	// Create initial commits on main
	createFile(t, repo, "README.md", "# Test Project")
	commitFile(t, repo, "README.md", "Initial commit")

	return repo
}

// createBranch creates a new branch from the current HEAD.
func createBranch(t *testing.T, repo, branchName string) {
	t.Helper()
	runGit(t, repo, "checkout", "-b", branchName)
}

// checkoutBranch switches to an existing branch.
func checkoutBranch(t *testing.T, repo, branchName string) {
	t.Helper()
	runGit(t, repo, "checkout", branchName)
}

// TestGetBranchLastCommit tests getting the last commit on a branch.
func TestGetBranchLastCommit_ValidBranch(t *testing.T) {
	t.Parallel()

	repo := setupBranchTestRepo(t)

	sha, at, err := GetBranchLastCommit(repo, "main")
	if err != nil {
		t.Fatalf("GetBranchLastCommit() error = %v", err)
	}

	if sha == "" {
		t.Error("GetBranchLastCommit() sha is empty")
	}
	if len(sha) < 40 {
		t.Errorf("GetBranchLastCommit() sha = %q, expected full SHA (40 chars)", sha)
	}

	// Commit should be recent
	if time.Since(at) > time.Minute {
		t.Errorf("GetBranchLastCommit() at = %v, expected recent commit", at)
	}
}

func TestGetBranchLastCommit_InvalidBranch(t *testing.T) {
	t.Parallel()

	repo := setupBranchTestRepo(t)

	_, _, err := GetBranchLastCommit(repo, "nonexistent-branch")
	if err != ErrBranchNotFound {
		t.Errorf("GetBranchLastCommit() error = %v, want ErrBranchNotFound", err)
	}
}

func TestGetBranchLastCommit_NotARepository(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	_, _, err := GetBranchLastCommit(dir, "main")
	if err != ErrNotARepository {
		t.Errorf("GetBranchLastCommit() error = %v, want ErrNotARepository", err)
	}
}

// TestGetBranchCreationTime tests getting branch creation time.
func TestGetBranchCreationTime_NewBranch(t *testing.T) {
	t.Parallel()

	repo := setupBranchTestRepo(t)

	// Create a feature branch
	createBranch(t, repo, "feature-1")

	// Add a commit to the branch
	createFile(t, repo, "feature.txt", "feature content")
	commitFile(t, repo, "feature.txt", "Feature commit")

	beforeTest := time.Now().Add(-time.Minute)

	createdAt, err := GetBranchCreationTime(repo, "feature-1")
	if err != nil {
		t.Fatalf("GetBranchCreationTime() error = %v", err)
	}

	if createdAt.Before(beforeTest) {
		t.Errorf("GetBranchCreationTime() = %v, expected after %v", createdAt, beforeTest)
	}
}

func TestGetBranchCreationTime_InvalidBranch(t *testing.T) {
	t.Parallel()

	repo := setupBranchTestRepo(t)

	_, err := GetBranchCreationTime(repo, "nonexistent-branch")
	if err != ErrBranchNotFound {
		t.Errorf("GetBranchCreationTime() error = %v, want ErrBranchNotFound", err)
	}
}

func TestGetBranchCreationTime_NotARepository(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	_, err := GetBranchCreationTime(dir, "main")
	if err != ErrNotARepository {
		t.Errorf("GetBranchCreationTime() error = %v, want ErrNotARepository", err)
	}
}

// TestGetCommitsBehindAhead tests counting commits behind/ahead.
func TestGetCommitsBehindAhead_SameBranch(t *testing.T) {
	t.Parallel()

	repo := setupBranchTestRepo(t)

	behind, ahead, err := GetCommitsBehindAhead(repo, "main", "main")
	if err != nil {
		t.Fatalf("GetCommitsBehindAhead() error = %v", err)
	}

	if behind != 0 || ahead != 0 {
		t.Errorf("GetCommitsBehindAhead() = (%d, %d), want (0, 0)", behind, ahead)
	}
}

func TestGetCommitsBehindAhead_BranchAhead(t *testing.T) {
	t.Parallel()

	repo := setupBranchTestRepo(t)

	// Create a branch and add commits
	createBranch(t, repo, "feature")
	createFile(t, repo, "feature1.txt", "content 1")
	commitFile(t, repo, "feature1.txt", "Feature commit 1")
	createFile(t, repo, "feature2.txt", "content 2")
	commitFile(t, repo, "feature2.txt", "Feature commit 2")

	behind, ahead, err := GetCommitsBehindAhead(repo, "feature", "main")
	if err != nil {
		t.Fatalf("GetCommitsBehindAhead() error = %v", err)
	}

	if behind != 0 {
		t.Errorf("GetCommitsBehindAhead() behind = %d, want 0", behind)
	}
	if ahead != 2 {
		t.Errorf("GetCommitsBehindAhead() ahead = %d, want 2", ahead)
	}
}

func TestGetCommitsBehindAhead_BranchBehind(t *testing.T) {
	t.Parallel()

	repo := setupBranchTestRepo(t)

	// Create a branch
	createBranch(t, repo, "feature")

	// Go back to main and add commits
	checkoutBranch(t, repo, "main")
	createFile(t, repo, "main1.txt", "content 1")
	commitFile(t, repo, "main1.txt", "Main commit 1")
	createFile(t, repo, "main2.txt", "content 2")
	commitFile(t, repo, "main2.txt", "Main commit 2")
	createFile(t, repo, "main3.txt", "content 3")
	commitFile(t, repo, "main3.txt", "Main commit 3")

	behind, ahead, err := GetCommitsBehindAhead(repo, "feature", "main")
	if err != nil {
		t.Fatalf("GetCommitsBehindAhead() error = %v", err)
	}

	if behind != 3 {
		t.Errorf("GetCommitsBehindAhead() behind = %d, want 3", behind)
	}
	if ahead != 0 {
		t.Errorf("GetCommitsBehindAhead() ahead = %d, want 0", ahead)
	}
}

func TestGetCommitsBehindAhead_BranchDiverged(t *testing.T) {
	t.Parallel()

	repo := setupBranchTestRepo(t)

	// Create a branch and add commits
	createBranch(t, repo, "feature")
	createFile(t, repo, "feature.txt", "feature content")
	commitFile(t, repo, "feature.txt", "Feature commit")

	// Go back to main and add different commits
	checkoutBranch(t, repo, "main")
	createFile(t, repo, "main.txt", "main content 1")
	commitFile(t, repo, "main.txt", "Main commit 1")
	createFile(t, repo, "main2.txt", "main content 2")
	commitFile(t, repo, "main2.txt", "Main commit 2")

	behind, ahead, err := GetCommitsBehindAhead(repo, "feature", "main")
	if err != nil {
		t.Fatalf("GetCommitsBehindAhead() error = %v", err)
	}

	if behind != 2 {
		t.Errorf("GetCommitsBehindAhead() behind = %d, want 2", behind)
	}
	if ahead != 1 {
		t.Errorf("GetCommitsBehindAhead() ahead = %d, want 1", ahead)
	}
}

func TestGetCommitsBehindAhead_InvalidBranch(t *testing.T) {
	t.Parallel()

	repo := setupBranchTestRepo(t)

	_, _, err := GetCommitsBehindAhead(repo, "nonexistent", "main")
	if err != ErrBranchNotFound {
		t.Errorf("GetCommitsBehindAhead() error = %v, want ErrBranchNotFound", err)
	}
}

func TestGetCommitsBehindAhead_InvalidBase(t *testing.T) {
	t.Parallel()

	repo := setupBranchTestRepo(t)

	_, _, err := GetCommitsBehindAhead(repo, "main", "nonexistent")
	if err != ErrBranchNotFound {
		t.Errorf("GetCommitsBehindAhead() error = %v, want ErrBranchNotFound", err)
	}
}

// TestHasMergeConflicts tests conflict detection.
func TestHasMergeConflicts_NoConflicts(t *testing.T) {
	t.Parallel()

	repo := setupBranchTestRepo(t)

	// Create a branch with changes to different files
	createBranch(t, repo, "feature")
	createFile(t, repo, "feature.txt", "feature content")
	commitFile(t, repo, "feature.txt", "Feature commit")

	// Add different changes to main
	checkoutBranch(t, repo, "main")
	createFile(t, repo, "main.txt", "main content")
	commitFile(t, repo, "main.txt", "Main commit")

	hasConflicts, err := HasMergeConflicts(repo, "feature", "main")
	if err != nil {
		t.Fatalf("HasMergeConflicts() error = %v", err)
	}

	if hasConflicts {
		t.Error("HasMergeConflicts() = true, want false (no conflicts)")
	}
}

func TestHasMergeConflicts_WithConflicts(t *testing.T) {
	t.Parallel()

	repo := setupBranchTestRepo(t)

	// Create a branch with changes to a file
	createBranch(t, repo, "feature")
	createFile(t, repo, "shared.txt", "feature version of content")
	commitFile(t, repo, "shared.txt", "Feature commit")

	// Add conflicting changes to main
	checkoutBranch(t, repo, "main")
	createFile(t, repo, "shared.txt", "main version of content")
	commitFile(t, repo, "shared.txt", "Main commit")

	hasConflicts, err := HasMergeConflicts(repo, "feature", "main")
	if err != nil {
		t.Fatalf("HasMergeConflicts() error = %v", err)
	}

	if !hasConflicts {
		t.Error("HasMergeConflicts() = false, want true (conflicting changes)")
	}
}

func TestHasMergeConflicts_SameBranch(t *testing.T) {
	t.Parallel()

	repo := setupBranchTestRepo(t)

	hasConflicts, err := HasMergeConflicts(repo, "main", "main")
	if err != nil {
		t.Fatalf("HasMergeConflicts() error = %v", err)
	}

	if hasConflicts {
		t.Error("HasMergeConflicts() = true, want false (same branch)")
	}
}

func TestHasMergeConflicts_InvalidBranch(t *testing.T) {
	t.Parallel()

	repo := setupBranchTestRepo(t)

	_, err := HasMergeConflicts(repo, "nonexistent", "main")
	if err != ErrBranchNotFound {
		t.Errorf("HasMergeConflicts() error = %v, want ErrBranchNotFound", err)
	}
}

// TestComputeBranchMetrics tests full metric computation.
func TestComputeBranchMetrics_FeatureBranch(t *testing.T) {
	t.Parallel()

	repo := setupBranchTestRepo(t)

	// Create a feature branch
	createBranch(t, repo, "feature")
	createFile(t, repo, "feature.txt", "content")
	commitFile(t, repo, "feature.txt", "Feature commit")

	// Add commits to main
	checkoutBranch(t, repo, "main")
	createFile(t, repo, "main.txt", "content")
	commitFile(t, repo, "main.txt", "Main commit")

	metrics, err := ComputeBranchMetrics(repo, "feature")
	if err != nil {
		t.Fatalf("ComputeBranchMetrics() error = %v", err)
	}

	if metrics.Branch != "feature" {
		t.Errorf("metrics.Branch = %q, want %q", metrics.Branch, "feature")
	}
	if metrics.CommitsBehindMain != 1 {
		t.Errorf("metrics.CommitsBehindMain = %d, want 1", metrics.CommitsBehindMain)
	}
	if metrics.CommitsAheadOfMain != 1 {
		t.Errorf("metrics.CommitsAheadOfMain = %d, want 1", metrics.CommitsAheadOfMain)
	}
	if metrics.HasConflicts {
		t.Error("metrics.HasConflicts = true, want false")
	}
	if metrics.LastCommitAgeDays < 0 {
		t.Errorf("metrics.LastCommitAgeDays = %d, want >= 0", metrics.LastCommitAgeDays)
	}
	if metrics.BranchAgeDays < 0 {
		t.Errorf("metrics.BranchAgeDays = %d, want >= 0", metrics.BranchAgeDays)
	}
}

func TestComputeBranchMetrics_InvalidBranch(t *testing.T) {
	t.Parallel()

	repo := setupBranchTestRepo(t)

	_, err := ComputeBranchMetrics(repo, "nonexistent")
	if err == nil {
		t.Error("ComputeBranchMetrics() expected error for nonexistent branch")
	}
}

func TestComputeBranchMetrics_WithBase(t *testing.T) {
	t.Parallel()

	repo := setupBranchTestRepo(t)

	// Create a develop branch
	createBranch(t, repo, "develop")
	createFile(t, repo, "dev.txt", "content")
	commitFile(t, repo, "dev.txt", "Dev commit")

	// Create a feature branch from develop
	createBranch(t, repo, "feature")
	createFile(t, repo, "feature.txt", "content")
	commitFile(t, repo, "feature.txt", "Feature commit")

	// Compute metrics against develop, not main
	metrics, err := ComputeBranchMetricsWithBase(repo, "feature", "develop")
	if err != nil {
		t.Fatalf("ComputeBranchMetricsWithBase() error = %v", err)
	}

	if metrics.CommitsBehindMain != 0 {
		t.Errorf("metrics.CommitsBehindMain = %d, want 0 (against develop)", metrics.CommitsBehindMain)
	}
	if metrics.CommitsAheadOfMain != 1 {
		t.Errorf("metrics.CommitsAheadOfMain = %d, want 1", metrics.CommitsAheadOfMain)
	}
}

// TestEvaluateBranchStatus tests threshold evaluation.
func TestEvaluateBranchStatus_HealthyBranch(t *testing.T) {
	t.Parallel()

	repo := setupBranchTestRepo(t)

	// Create a healthy branch (recent commits, not behind main)
	createBranch(t, repo, "feature")
	createFile(t, repo, "feature.txt", "content")
	commitFile(t, repo, "feature.txt", "Feature commit")

	thresholds := DefaultBranchThresholds()
	status, err := EvaluateBranchStatus(repo, "feature", thresholds)
	if err != nil {
		t.Fatalf("EvaluateBranchStatus() error = %v", err)
	}

	if len(status.Warnings) != 0 {
		t.Errorf("status.Warnings = %v, want empty", status.Warnings)
	}
	if len(status.Errors) != 0 {
		t.Errorf("status.Errors = %v, want empty", status.Errors)
	}
}

func TestEvaluateBranchStatus_DriftWarning(t *testing.T) {
	t.Parallel()

	repo := setupBranchTestRepo(t)

	// Create a branch
	createBranch(t, repo, "feature")

	// Add 55 commits to main (above warning threshold of 50)
	checkoutBranch(t, repo, "main")
	for i := 0; i < 55; i++ {
		createFile(t, repo, "file.txt", "content "+string(rune(i)))
		commitFile(t, repo, "file.txt", "Main commit")
	}

	thresholds := DefaultBranchThresholds()
	status, err := EvaluateBranchStatus(repo, "feature", thresholds)
	if err != nil {
		t.Fatalf("EvaluateBranchStatus() error = %v", err)
	}

	if len(status.Warnings) == 0 {
		t.Error("status.Warnings is empty, expected drift warning")
	}
	if len(status.Errors) != 0 {
		t.Errorf("status.Errors = %v, want empty (not at error threshold)", status.Errors)
	}
}

func TestEvaluateBranchStatus_DriftError(t *testing.T) {
	t.Parallel()

	repo := setupBranchTestRepo(t)

	// Create a branch
	createBranch(t, repo, "feature")

	// Add 105 commits to main (above error threshold of 100)
	checkoutBranch(t, repo, "main")
	for i := 0; i < 105; i++ {
		createFile(t, repo, "file.txt", "content "+string(rune(i)))
		commitFile(t, repo, "file.txt", "Main commit")
	}

	thresholds := DefaultBranchThresholds()
	status, err := EvaluateBranchStatus(repo, "feature", thresholds)
	if err != nil {
		t.Fatalf("EvaluateBranchStatus() error = %v", err)
	}

	// Should have error but not warning (error supersedes warning)
	if len(status.Errors) == 0 {
		t.Error("status.Errors is empty, expected drift error")
	}
	// Check that the drift error is present
	hasDriftError := false
	for _, e := range status.Errors {
		if strings.Contains(e, "critical drift") {
			hasDriftError = true
			break
		}
	}
	if !hasDriftError {
		t.Error("status.Errors does not contain critical drift error")
	}
}

func TestEvaluateBranchStatus_MergeConflicts(t *testing.T) {
	t.Parallel()

	repo := setupBranchTestRepo(t)

	// Create conflicting changes
	createBranch(t, repo, "feature")
	createFile(t, repo, "shared.txt", "feature version")
	commitFile(t, repo, "shared.txt", "Feature commit")

	checkoutBranch(t, repo, "main")
	createFile(t, repo, "shared.txt", "main version")
	commitFile(t, repo, "shared.txt", "Main commit")

	thresholds := DefaultBranchThresholds()
	status, err := EvaluateBranchStatus(repo, "feature", thresholds)
	if err != nil {
		t.Fatalf("EvaluateBranchStatus() error = %v", err)
	}

	if len(status.Errors) == 0 {
		t.Error("status.Errors is empty, expected merge conflict error")
	}
	hasConflictError := false
	for _, e := range status.Errors {
		if strings.Contains(e, "merge conflicts") {
			hasConflictError = true
			break
		}
	}
	if !hasConflictError {
		t.Error("status.Errors does not contain merge conflicts error")
	}
}

func TestEvaluateBranchStatus_CustomThresholds(t *testing.T) {
	t.Parallel()

	repo := setupBranchTestRepo(t)

	// Create a branch
	createBranch(t, repo, "feature")

	// Add 10 commits to main
	checkoutBranch(t, repo, "main")
	for i := 0; i < 10; i++ {
		createFile(t, repo, "file.txt", "content "+string(rune(i)))
		commitFile(t, repo, "file.txt", "Main commit")
	}

	// Use custom thresholds where 10 commits triggers a warning
	thresholds := BranchThresholds{
		StaleAfterDays:      1,
		DriftWarningCommits: 5,
		DriftErrorCommits:   15,
	}

	status, err := EvaluateBranchStatus(repo, "feature", thresholds)
	if err != nil {
		t.Fatalf("EvaluateBranchStatus() error = %v", err)
	}

	if len(status.Warnings) == 0 {
		t.Error("status.Warnings is empty, expected drift warning with custom threshold")
	}
}

func TestEvaluateBranchStatus_InvalidBranch(t *testing.T) {
	t.Parallel()

	repo := setupBranchTestRepo(t)

	thresholds := DefaultBranchThresholds()
	_, err := EvaluateBranchStatus(repo, "nonexistent", thresholds)
	if err == nil {
		t.Error("EvaluateBranchStatus() expected error for nonexistent branch")
	}
}

// TestDefaultBranchThresholds tests default threshold values.
func TestDefaultBranchThresholds(t *testing.T) {
	t.Parallel()

	thresholds := DefaultBranchThresholds()

	if thresholds.StaleAfterDays != 14 {
		t.Errorf("StaleAfterDays = %d, want 14", thresholds.StaleAfterDays)
	}
	if thresholds.DriftWarningCommits != 50 {
		t.Errorf("DriftWarningCommits = %d, want 50", thresholds.DriftWarningCommits)
	}
	if thresholds.DriftErrorCommits != 100 {
		t.Errorf("DriftErrorCommits = %d, want 100", thresholds.DriftErrorCommits)
	}
}

// TestMasterBranchFallback tests that master is used if main doesn't exist.
func TestMasterBranchFallback(t *testing.T) {
	t.Parallel()

	repo := setupTestRepo(t)

	// Don't rename to main - keep as master (or whatever default)
	// First ensure we have a master branch
	runGit(t, repo, "branch", "-M", "master")

	createFile(t, repo, "README.md", "# Test")
	commitFile(t, repo, "README.md", "Initial commit")

	// Create a feature branch
	createBranch(t, repo, "feature")
	createFile(t, repo, "feature.txt", "content")
	commitFile(t, repo, "feature.txt", "Feature commit")

	// Should compute metrics against master
	metrics, err := ComputeBranchMetrics(repo, "feature")
	if err != nil {
		t.Fatalf("ComputeBranchMetrics() error = %v", err)
	}

	// Should have 1 commit ahead of master
	if metrics.CommitsAheadOfMain != 1 {
		t.Errorf("metrics.CommitsAheadOfMain = %d, want 1", metrics.CommitsAheadOfMain)
	}
}

// TestBranchWithNoUniqueCommits tests a branch that's identical to main.
func TestBranchWithNoUniqueCommits(t *testing.T) {
	t.Parallel()

	repo := setupBranchTestRepo(t)

	// Create a branch at the same point as main (no unique commits)
	createBranch(t, repo, "feature")

	// Should still be able to compute metrics
	metrics, err := ComputeBranchMetrics(repo, "feature")
	if err != nil {
		t.Fatalf("ComputeBranchMetrics() error = %v", err)
	}

	if metrics.CommitsAheadOfMain != 0 {
		t.Errorf("metrics.CommitsAheadOfMain = %d, want 0", metrics.CommitsAheadOfMain)
	}
	if metrics.CommitsBehindMain != 0 {
		t.Errorf("metrics.CommitsBehindMain = %d, want 0", metrics.CommitsBehindMain)
	}
}

// TestNotARepository tests error handling for non-repository paths.
func TestComputeBranchMetrics_NotARepository(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	_, err := ComputeBranchMetrics(dir, "main")
	if err == nil {
		t.Error("ComputeBranchMetrics() expected error for non-repository")
	}
}

func TestEvaluateBranchStatus_NotARepository(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	thresholds := DefaultBranchThresholds()
	_, err := EvaluateBranchStatus(dir, "main", thresholds)
	if err == nil {
		t.Error("EvaluateBranchStatus() expected error for non-repository")
	}
}

// TestGetCurrentBranch is a helper test to verify git is working.
func TestGitAvailable(t *testing.T) {
	t.Parallel()

	cmd := exec.Command("git", "--version")
	if err := cmd.Run(); err != nil {
		t.Skip("git not available:", err)
	}
}
