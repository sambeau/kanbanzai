package health

import (
	"testing"

	"github.com/sambeau/kanbanzai/internal/worktree"
)

func TestCheckBugWorktree_NoBugs(t *testing.T) {
	t.Parallel()

	result := CheckBugWorktree(nil, nil)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
	}
}

func TestCheckBugWorktree_InProgressBugWithWorktree(t *testing.T) {
	t.Parallel()

	bugs := []map[string]any{
		{
			"id":     "BUG-001",
			"status": "in-progress",
		},
	}

	worktrees := []worktree.Record{
		{
			ID:       "WT-001",
			EntityID: "BUG-001",
			Branch:   "bug/test",
			Path:     "/tmp/bug-worktree",
			Status:   worktree.StatusActive,
		},
	}

	result := CheckBugWorktree(bugs, worktrees)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
}

func TestCheckBugWorktree_InProgressBugWithoutWorktree(t *testing.T) {
	t.Parallel()

	bugs := []map[string]any{
		{
			"id":     "BUG-001",
			"status": "in-progress",
		},
	}

	result := CheckBugWorktree(bugs, nil)

	if result.Status != SeverityWarning {
		t.Errorf("Status = %v, want %v", result.Status, SeverityWarning)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("len(Issues) = %d, want 1", len(result.Issues))
	}

	issue := result.Issues[0]
	if issue.EntityID != "BUG-001" {
		t.Errorf("EntityID = %q, want %q", issue.EntityID, "BUG-001")
	}
	if issue.Severity != SeverityWarning {
		t.Errorf("Severity = %v, want %v", issue.Severity, SeverityWarning)
	}
}

func TestCheckBugWorktree_BugInOtherStatus(t *testing.T) {
	t.Parallel()

	bugs := []map[string]any{
		{"id": "BUG-001", "status": "reported"},
		{"id": "BUG-002", "status": "triaged"},
		{"id": "BUG-003", "status": "needs-review"},
		{"id": "BUG-004", "status": "closed"},
	}

	result := CheckBugWorktree(bugs, nil)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
	}
}

func TestCheckBugWorktree_MixedStatuses(t *testing.T) {
	t.Parallel()

	bugs := []map[string]any{
		{"id": "BUG-001", "status": "in-progress"}, // no worktree → warning
		{"id": "BUG-002", "status": "reported"},     // different status → no warning
		{"id": "BUG-003", "status": "in-progress"}, // has worktree → no warning
	}

	worktrees := []worktree.Record{
		{
			ID:       "WT-003",
			EntityID: "BUG-003",
			Branch:   "bug/bug-003",
			Path:     "/tmp/bug-003",
			Status:   worktree.StatusActive,
		},
	}

	result := CheckBugWorktree(bugs, worktrees)

	if result.Status != SeverityWarning {
		t.Errorf("Status = %v, want %v", result.Status, SeverityWarning)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("len(Issues) = %d, want 1", len(result.Issues))
	}
	if result.Issues[0].EntityID != "BUG-001" {
		t.Errorf("EntityID = %q, want %q", result.Issues[0].EntityID, "BUG-001")
	}
}

func TestCheckBugWorktree_WorktreeNotActive(t *testing.T) {
	t.Parallel()

	bugs := []map[string]any{
		{"id": "BUG-001", "status": "in-progress"},
	}

	// Worktree exists but is merged, not active
	worktrees := []worktree.Record{
		{
			ID:       "WT-001",
			EntityID: "BUG-001",
			Branch:   "bug/test",
			Path:     "/tmp/bug-worktree",
			Status:   worktree.StatusMerged,
		},
	}

	result := CheckBugWorktree(bugs, worktrees)

	if result.Status != SeverityWarning {
		t.Errorf("Status = %v, want %v (merged worktree does not count as active)", result.Status, SeverityWarning)
	}
}

func TestCheckBugWorktree_MessageFormat(t *testing.T) {
	t.Parallel()

	bugs := []map[string]any{
		{"id": "BUG-01KR197N74YTY", "status": "in-progress"},
	}

	result := CheckBugWorktree(bugs, nil)

	if len(result.Issues) != 1 {
		t.Fatalf("len(Issues) = %d, want 1", len(result.Issues))
	}

	expectedMsg := "bug BUG-01KR197N74YTY is in-progress but has no active worktree — changes may not be isolated"
	if result.Issues[0].Message != expectedMsg {
		t.Errorf("Message = %q, want %q", result.Issues[0].Message, expectedMsg)
	}
}
