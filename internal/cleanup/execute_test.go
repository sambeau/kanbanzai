package cleanup

import (
	"errors"
	"testing"
	"time"

	"kanbanzai/internal/worktree"
)

// mockGit provides a test double for worktree.Git operations.
type mockGit struct {
	removeWorktreeCalls []string
	removeWorktreeErr   error
	pruneWorktreesCalls int
	deleteBranchCalls   []deleteBranchCall
	deleteBranchErr     error
	deleteRemoteCalls   []deleteRemoteCall
	deleteRemoteErr     error
}

type deleteBranchCall struct {
	branch string
	force  bool
}

type deleteRemoteCall struct {
	remote string
	branch string
}

func (m *mockGit) RemoveWorktree(path string, force bool) error {
	m.removeWorktreeCalls = append(m.removeWorktreeCalls, path)
	return m.removeWorktreeErr
}

func (m *mockGit) PruneWorktrees() error {
	m.pruneWorktreesCalls++
	return nil
}

func (m *mockGit) DeleteBranch(branch string, force bool) error {
	m.deleteBranchCalls = append(m.deleteBranchCalls, deleteBranchCall{branch, force})
	return m.deleteBranchErr
}

func (m *mockGit) DeleteRemoteBranch(remote, branch string) error {
	m.deleteRemoteCalls = append(m.deleteRemoteCalls, deleteRemoteCall{remote, branch})
	return m.deleteRemoteErr
}

// gitAdapter wraps mockGit to work with ExecuteCleanup.
// Since ExecuteCleanup expects *worktree.Git, we'll need to use
// a test approach that doesn't require actual git operations.

func TestExecuteCleanup_DryRun(t *testing.T) {
	// Dry run should not actually do anything
	store := worktree.NewStore(t.TempDir())

	// Create a record to clean up
	record := worktree.Record{
		ID:        "WT-TEST001",
		EntityID:  "FEAT-001",
		Branch:    "feature/test",
		Path:      ".worktrees/feature-test",
		Status:    worktree.StatusMerged,
		Created:   time.Now().Add(-24 * time.Hour),
		CreatedBy: "test-user",
	}

	createdRecord, err := store.Create(record)
	if err != nil {
		t.Fatalf("Failed to create record: %v", err)
	}

	// Create a real Git instance pointing to temp dir
	git := worktree.NewGit(t.TempDir())

	result := ExecuteCleanup(store, git, createdRecord, CleanupOptions{
		DryRun:             true,
		DeleteRemoteBranch: true,
	})

	if !result.Success {
		t.Errorf("DryRun should succeed, got error: %v", result.Error)
	}
	if result.WorktreeID != "WT-TEST001" {
		t.Errorf("WorktreeID = %q, want %q", result.WorktreeID, "WT-TEST001")
	}
	if result.Branch != "feature/test" {
		t.Errorf("Branch = %q, want %q", result.Branch, "feature/test")
	}
	if !result.RemoteBranchDeleted {
		t.Error("DryRun with DeleteRemoteBranch should report RemoteBranchDeleted=true")
	}

	// Verify record still exists (dry run doesn't delete)
	_, err = store.Get("WT-TEST001")
	if err != nil {
		t.Errorf("Record should still exist after dry run: %v", err)
	}
}

func TestCleanupResult_Fields(t *testing.T) {
	result := CleanupResult{
		WorktreeID:          "WT-123",
		Branch:              "feature/test",
		Path:                ".worktrees/test",
		RemoteBranchDeleted: true,
		Success:             true,
		Error:               nil,
	}

	if result.WorktreeID != "WT-123" {
		t.Errorf("WorktreeID = %q, want %q", result.WorktreeID, "WT-123")
	}
	if result.Branch != "feature/test" {
		t.Errorf("Branch = %q, want %q", result.Branch, "feature/test")
	}
	if result.Path != ".worktrees/test" {
		t.Errorf("Path = %q, want %q", result.Path, ".worktrees/test")
	}
	if !result.RemoteBranchDeleted {
		t.Error("RemoteBranchDeleted should be true")
	}
	if !result.Success {
		t.Error("Success should be true")
	}
	if result.Error != nil {
		t.Errorf("Error should be nil, got %v", result.Error)
	}
}

func TestCleanupResult_WithError(t *testing.T) {
	testErr := errors.New("test error")
	result := CleanupResult{
		WorktreeID: "WT-456",
		Success:    false,
		Error:      testErr,
	}

	if result.Success {
		t.Error("Success should be false when there's an error")
	}
	if result.Error != testErr {
		t.Errorf("Error = %v, want %v", result.Error, testErr)
	}
}

func TestCleanupOptions_Defaults(t *testing.T) {
	opts := CleanupOptions{}

	if opts.DryRun {
		t.Error("Default DryRun should be false")
	}
	if opts.DeleteRemoteBranch {
		t.Error("Default DeleteRemoteBranch should be false")
	}
	if opts.ForceRemove {
		t.Error("Default ForceRemove should be false")
	}
}

func TestIsWorktreeNotFoundError(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect bool
	}{
		{
			name:   "nil error",
			err:    nil,
			expect: false,
		},
		{
			name:   "not a working tree",
			err:    errors.New("'/path/to/wt' is not a working tree"),
			expect: true,
		},
		{
			name:   "not a valid directory",
			err:    errors.New("not a valid directory"),
			expect: true,
		},
		{
			name:   "does not exist",
			err:    errors.New("worktree does not exist"),
			expect: true,
		},
		{
			name:   "generic error",
			err:    errors.New("permission denied"),
			expect: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isWorktreeNotFoundError(tc.err)
			if got != tc.expect {
				t.Errorf("isWorktreeNotFoundError(%v) = %v, want %v", tc.err, got, tc.expect)
			}
		})
	}
}

func TestIsBranchNotFoundError(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect bool
	}{
		{
			name:   "nil error",
			err:    nil,
			expect: false,
		},
		{
			name:   "branch not found",
			err:    errors.New("branch 'feature/test' not found"),
			expect: true,
		},
		{
			name:   "not found generic",
			err:    errors.New("error: branch not found."),
			expect: true,
		},
		{
			name:   "generic error",
			err:    errors.New("fatal: cannot lock ref"),
			expect: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isBranchNotFoundError(tc.err)
			if got != tc.expect {
				t.Errorf("isBranchNotFoundError(%v) = %v, want %v", tc.err, got, tc.expect)
			}
		})
	}
}

func TestIsRemoteBranchNotFoundError(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect bool
	}{
		{
			name:   "nil error",
			err:    nil,
			expect: false,
		},
		{
			name:   "remote ref does not exist",
			err:    errors.New("error: remote ref does not exist"),
			expect: true,
		},
		{
			name:   "unable to delete remote ref",
			err:    errors.New("error: unable to delete 'feature/test': remote ref does not exist"),
			expect: true,
		},
		{
			name:   "generic error",
			err:    errors.New("fatal: could not read from remote"),
			expect: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isRemoteBranchNotFoundError(tc.err)
			if got != tc.expect {
				t.Errorf("isRemoteBranchNotFoundError(%v) = %v, want %v", tc.err, got, tc.expect)
			}
		})
	}
}

func TestExecuteAllReady_EmptyStore(t *testing.T) {
	store := worktree.NewStore(t.TempDir())
	git := worktree.NewGit(t.TempDir())

	results := ExecuteAllReady(store, git, 7, CleanupOptions{DryRun: true})

	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty store, got %d", len(results))
	}
}

func TestExecuteAllReady_FiltersNotReady(t *testing.T) {
	store := worktree.NewStore(t.TempDir())
	git := worktree.NewGit(t.TempDir())

	now := time.Now()
	futureCleanup := now.Add(24 * time.Hour)

	// Create a record that's not ready (cleanup in the future)
	record := worktree.Record{
		ID:           "WT-NOTREADY",
		EntityID:     "FEAT-001",
		Branch:       "feature/not-ready",
		Path:         ".worktrees/not-ready",
		Status:       worktree.StatusMerged,
		Created:      now.Add(-48 * time.Hour),
		CreatedBy:    "test-user",
		CleanupAfter: &futureCleanup,
	}

	_, err := store.Create(record)
	if err != nil {
		t.Fatalf("Failed to create record: %v", err)
	}

	results := ExecuteAllReady(store, git, 7, CleanupOptions{DryRun: true})

	if len(results) != 0 {
		t.Errorf("Expected 0 results (record not ready), got %d", len(results))
	}
}

func TestExecuteAllReady_ProcessesReadyItems(t *testing.T) {
	store := worktree.NewStore(t.TempDir())
	git := worktree.NewGit(t.TempDir())

	now := time.Now()
	pastCleanup := now.Add(-24 * time.Hour)

	// Create a record that's ready (cleanup in the past)
	record := worktree.Record{
		ID:           "WT-READY",
		EntityID:     "FEAT-002",
		Branch:       "feature/ready",
		Path:         ".worktrees/ready",
		Status:       worktree.StatusMerged,
		Created:      now.Add(-72 * time.Hour),
		CreatedBy:    "test-user",
		CleanupAfter: &pastCleanup,
	}

	_, err := store.Create(record)
	if err != nil {
		t.Fatalf("Failed to create record: %v", err)
	}

	results := ExecuteAllReady(store, git, 7, CleanupOptions{DryRun: true})

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].WorktreeID != "WT-READY" {
		t.Errorf("WorktreeID = %q, want %q", results[0].WorktreeID, "WT-READY")
	}
	if !results[0].Success {
		t.Errorf("Expected success for dry run, got error: %v", results[0].Error)
	}
}

func TestExecuteAllReady_ProcessesAbandonedWithForce(t *testing.T) {
	store := worktree.NewStore(t.TempDir())
	git := worktree.NewGit(t.TempDir())

	now := time.Now()
	pastCleanup := now.Add(-1 * time.Hour)

	// Create an abandoned record
	record := worktree.Record{
		ID:           "WT-ABANDONED",
		EntityID:     "FEAT-003",
		Branch:       "feature/abandoned",
		Path:         ".worktrees/abandoned",
		Status:       worktree.StatusAbandoned,
		Created:      now.Add(-48 * time.Hour),
		CreatedBy:    "test-user",
		CleanupAfter: &pastCleanup,
	}

	_, err := store.Create(record)
	if err != nil {
		t.Fatalf("Failed to create record: %v", err)
	}

	// Run with DryRun to avoid actual git operations
	results := ExecuteAllReady(store, git, 7, CleanupOptions{DryRun: true})

	if len(results) != 1 {
		t.Fatalf("Expected 1 result for abandoned worktree, got %d", len(results))
	}

	if results[0].WorktreeID != "WT-ABANDONED" {
		t.Errorf("WorktreeID = %q, want %q", results[0].WorktreeID, "WT-ABANDONED")
	}
	if !results[0].Success {
		t.Errorf("Expected success for dry run, got error: %v", results[0].Error)
	}
}

func TestContains_Helper(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"hello world", "world", true},
		{"hello world", "foo", false},
		{"hello", "hello", true},
		{"", "", true},
		{"hello", "", true},
		{"", "hello", false},
		{"abc", "abcd", false},
	}

	for _, tc := range tests {
		t.Run(tc.s+"_"+tc.substr, func(t *testing.T) {
			got := contains(tc.s, tc.substr)
			if got != tc.want {
				t.Errorf("contains(%q, %q) = %v, want %v", tc.s, tc.substr, got, tc.want)
			}
		})
	}
}
