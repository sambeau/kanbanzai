package health

import (
	"fmt"
	"testing"
	"time"
)

func TestCheckStalledDispatches_Disabled(t *testing.T) {
	t.Parallel()

	tasks := []map[string]any{
		{
			"id":            "TASK-001",
			"status":        "active",
			"dispatched_at": time.Now().Add(-48 * time.Hour).Format(time.RFC3339),
		},
	}

	result := CheckStalledDispatches(tasks, nil, "", 0)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0 (disabled when stallThresholdDays=0)", len(result.Issues))
	}
}

func TestCheckStalledDispatches_NilInputs(t *testing.T) {
	t.Parallel()

	// Must not panic with nil tasks and nil worktree map.
	result := CheckStalledDispatches(nil, nil, "", 1)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
	}
}

func TestCheckStalledDispatches_EmptyTasks(t *testing.T) {
	t.Parallel()

	result := CheckStalledDispatches([]map[string]any{}, nil, "", 1)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
}

func TestCheckStalledDispatches_NonActiveTasks_Skipped(t *testing.T) {
	t.Parallel()

	old := time.Now().Add(-72 * time.Hour).Format(time.RFC3339)

	tasks := []map[string]any{
		{"id": "TASK-001", "status": "done", "dispatched_at": old},
		{"id": "TASK-002", "status": "blocked", "dispatched_at": old},
		{"id": "TASK-003", "status": "queued", "dispatched_at": old},
		{"id": "TASK-004", "status": "ready", "dispatched_at": old},
		{"id": "TASK-005", "status": "needs-review", "dispatched_at": old},
	}

	result := CheckStalledDispatches(tasks, nil, "", 1)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0 (non-active tasks should be skipped)", len(result.Issues))
	}
}

func TestCheckStalledDispatches_NoDispatchedAt_Skipped(t *testing.T) {
	t.Parallel()

	// Manually activated tasks have no dispatched_at — they should be skipped.
	tasks := []map[string]any{
		{"id": "TASK-001", "status": "active"},
		{"id": "TASK-002", "status": "active", "dispatched_at": ""},
	}

	result := CheckStalledDispatches(tasks, nil, "", 1)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0 (no dispatched_at should be skipped)", len(result.Issues))
	}
}

func TestCheckStalledDispatches_InvalidDispatchedAt_Skipped(t *testing.T) {
	t.Parallel()

	tasks := []map[string]any{
		{"id": "TASK-001", "status": "active", "dispatched_at": "not-a-timestamp"},
	}

	result := CheckStalledDispatches(tasks, nil, "", 1)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0 (invalid timestamp should be skipped)", len(result.Issues))
	}
}

func TestCheckStalledDispatches_TaskBelowThreshold_NoWarning(t *testing.T) {
	t.Parallel()

	// Dispatched 12 hours ago, threshold is 2 days — should not trigger.
	recent := time.Now().Add(-12 * time.Hour).Format(time.RFC3339)
	tasks := []map[string]any{
		{"id": "TASK-001", "status": "active", "dispatched_at": recent},
	}

	result := CheckStalledDispatches(tasks, nil, "", 2)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0 (task is below threshold)", len(result.Issues))
	}
}

func TestCheckStalledDispatches_TaskAboveThreshold_NoBranch_Warning(t *testing.T) {
	t.Parallel()

	// Dispatched 3 days ago, threshold is 1 day, no worktree branch.
	// checkGitActivitySince returns false for empty branch → warning.
	old := time.Now().Add(-72 * time.Hour).Format(time.RFC3339)
	tasks := []map[string]any{
		{
			"id":             "TASK-001",
			"status":         "active",
			"dispatched_at":  old,
			"parent_feature": "FEAT-001",
		},
	}

	result := CheckStalledDispatches(tasks, map[string]string{}, "", 1)

	if result.Status != SeverityWarning {
		t.Errorf("Status = %v, want %v", result.Status, SeverityWarning)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("len(Issues) = %d, want 1", len(result.Issues))
	}

	issue := result.Issues[0]
	if issue.EntityID != "TASK-001" {
		t.Errorf("EntityID = %q, want %q", issue.EntityID, "TASK-001")
	}

	wantMsg := "TASK-001 has been active for >24h with no recent commits — may need unclaim"
	if issue.Message != wantMsg {
		t.Errorf("Message = %q, want %q", issue.Message, wantMsg)
	}
}

func TestCheckStalledDispatches_MultipleStalled_MultipleWarnings(t *testing.T) {
	t.Parallel()

	old := time.Now().Add(-48 * time.Hour).Format(time.RFC3339)
	tasks := []map[string]any{
		{"id": "TASK-001", "status": "active", "dispatched_at": old, "parent_feature": "FEAT-001"},
		{"id": "TASK-002", "status": "active", "dispatched_at": old, "parent_feature": "FEAT-002"},
		{"id": "TASK-003", "status": "done", "dispatched_at": old},
	}

	result := CheckStalledDispatches(tasks, map[string]string{}, "", 1)

	if result.Status != SeverityWarning {
		t.Errorf("Status = %v, want %v", result.Status, SeverityWarning)
	}
	if len(result.Issues) != 2 {
		t.Errorf("len(Issues) = %d, want 2", len(result.Issues))
	}
}

func TestCheckStalledDispatches_ExactlyAtThreshold_NoWarning(t *testing.T) {
	t.Parallel()

	// Dispatched exactly at the threshold boundary (within a small margin).
	// The task has been active for slightly less than 24h.
	justUnder := time.Now().Add(-23*time.Hour - 59*time.Minute).Format(time.RFC3339)
	tasks := []map[string]any{
		{"id": "TASK-001", "status": "active", "dispatched_at": justUnder, "parent_feature": "FEAT-001"},
	}

	result := CheckStalledDispatches(tasks, map[string]string{}, "", 1)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v (task is just under threshold)", result.Status, SeverityOK)
	}
}

func TestCheckStalledDispatches_MessageFormat(t *testing.T) {
	t.Parallel()

	// Verify the exact message format matches the spec.
	old := time.Now().Add(-48 * time.Hour).Format(time.RFC3339)
	tasks := []map[string]any{
		{
			"id":             "TASK-XYZ",
			"status":         "active",
			"dispatched_at":  old,
			"dispatched_to":  "some-agent",
			"parent_feature": "FEAT-ABC",
		},
	}

	result := CheckStalledDispatches(tasks, map[string]string{}, "", 1)

	if len(result.Issues) != 1 {
		t.Fatalf("len(Issues) = %d, want 1", len(result.Issues))
	}

	want := fmt.Sprintf("%s has been active for >24h with no recent commits — may need unclaim", "TASK-XYZ")
	if result.Issues[0].Message != want {
		t.Errorf("Message = %q, want %q", result.Issues[0].Message, want)
	}
}
