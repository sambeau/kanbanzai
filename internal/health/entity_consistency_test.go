package health

import (
	"strings"
	"testing"
)

func TestCheckFeatureChildConsistency_NoFeatures(t *testing.T) {
	t.Parallel()

	result := CheckFeatureChildConsistency(nil, nil)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
	}
}

func TestCheckFeatureChildConsistency_FeatureWithNoChildren(t *testing.T) {
	t.Parallel()

	features := []map[string]any{
		{"id": "FEAT-001", "status": "done"},
	}

	result := CheckFeatureChildConsistency(features, nil)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
	}
}

func TestCheckFeatureChildConsistency_TerminalFeatureWithNonTerminalChildren(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		featureStatus string
	}{
		{"done", "done"},
		{"superseded", "superseded"},
		{"cancelled", "cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			features := []map[string]any{
				{"id": "FEAT-001", "status": tt.featureStatus},
			}
			tasks := []map[string]any{
				{"id": "TASK-001", "parent_feature": "FEAT-001", "status": "active"},
				{"id": "TASK-002", "parent_feature": "FEAT-001", "status": "done"},
				{"id": "TASK-003", "parent_feature": "FEAT-001", "status": "queued"},
			}

			result := CheckFeatureChildConsistency(features, tasks)

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
			if !strings.Contains(issue.Message, "2 non-terminal child task(s)") {
				t.Errorf("Issue.Message = %q, want to contain %q", issue.Message, "2 non-terminal child task(s)")
			}
			if !strings.Contains(issue.Message, tt.featureStatus) {
				t.Errorf("Issue.Message = %q, want to contain status %q", issue.Message, tt.featureStatus)
			}
		})
	}
}

func TestCheckFeatureChildConsistency_TerminalFeatureAllChildrenTerminal(t *testing.T) {
	t.Parallel()

	features := []map[string]any{
		{"id": "FEAT-001", "status": "done"},
	}
	tasks := []map[string]any{
		{"id": "TASK-001", "parent_feature": "FEAT-001", "status": "done"},
		{"id": "TASK-002", "parent_feature": "FEAT-001", "status": "not-planned"},
		{"id": "TASK-003", "parent_feature": "FEAT-001", "status": "duplicate"},
	}

	result := CheckFeatureChildConsistency(features, tasks)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
	}
}

func TestCheckFeatureChildConsistency_EarlyFeatureAllChildrenTerminal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		featureStatus string
	}{
		{"proposed", "proposed"},
		{"designing", "designing"},
		{"specifying", "specifying"},
		{"dev-planning", "dev-planning"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			features := []map[string]any{
				{"id": "FEAT-001", "status": tt.featureStatus},
			}
			tasks := []map[string]any{
				{"id": "TASK-001", "parent_feature": "FEAT-001", "status": "done"},
				{"id": "TASK-002", "parent_feature": "FEAT-001", "status": "not-planned"},
			}

			result := CheckFeatureChildConsistency(features, tasks)

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
			if !strings.Contains(issue.Message, "all 2 child task(s) in terminal state") {
				t.Errorf("Issue.Message = %q, want to contain %q", issue.Message, "all 2 child task(s) in terminal state")
			}
			if !strings.Contains(issue.Message, tt.featureStatus) {
				t.Errorf("Issue.Message = %q, want to contain status %q", issue.Message, tt.featureStatus)
			}
		})
	}
}

func TestCheckFeatureChildConsistency_EarlyFeatureMixedChildren(t *testing.T) {
	t.Parallel()

	features := []map[string]any{
		{"id": "FEAT-001", "status": "proposed"},
	}
	tasks := []map[string]any{
		{"id": "TASK-001", "parent_feature": "FEAT-001", "status": "done"},
		{"id": "TASK-002", "parent_feature": "FEAT-001", "status": "queued"},
	}

	result := CheckFeatureChildConsistency(features, tasks)

	// Not all children are terminal, so no warning for early state
	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
	}
}

func TestCheckFeatureChildConsistency_DevelopingFeatureNotFlagged(t *testing.T) {
	t.Parallel()

	features := []map[string]any{
		{"id": "FEAT-001", "status": "developing"},
	}
	tasks := []map[string]any{
		{"id": "TASK-001", "parent_feature": "FEAT-001", "status": "active"},
		{"id": "TASK-002", "parent_feature": "FEAT-001", "status": "done"},
	}

	result := CheckFeatureChildConsistency(features, tasks)

	// "developing" is neither terminal/done nor early, so no warnings
	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
	}
}

func TestCheckFeatureChildConsistency_MultipleFeatures(t *testing.T) {
	t.Parallel()

	features := []map[string]any{
		{"id": "FEAT-001", "status": "done"},
		{"id": "FEAT-002", "status": "proposed"},
		{"id": "FEAT-003", "status": "developing"},
	}
	tasks := []map[string]any{
		// FEAT-001 (done) with non-terminal child → warning
		{"id": "TASK-001", "parent_feature": "FEAT-001", "status": "active"},
		// FEAT-002 (proposed) with all terminal children → warning
		{"id": "TASK-002", "parent_feature": "FEAT-002", "status": "done"},
		{"id": "TASK-003", "parent_feature": "FEAT-002", "status": "not-planned"},
		// FEAT-003 (developing) with mixed children → no warning
		{"id": "TASK-004", "parent_feature": "FEAT-003", "status": "active"},
		{"id": "TASK-005", "parent_feature": "FEAT-003", "status": "done"},
	}

	result := CheckFeatureChildConsistency(features, tasks)

	if result.Status != SeverityWarning {
		t.Errorf("Status = %v, want %v", result.Status, SeverityWarning)
	}
	if len(result.Issues) != 2 {
		t.Fatalf("len(Issues) = %d, want 2", len(result.Issues))
	}

	// Verify we got warnings for FEAT-001 and FEAT-002
	entityIDs := map[string]bool{}
	for _, issue := range result.Issues {
		entityIDs[issue.EntityID] = true
	}
	if !entityIDs["FEAT-001"] {
		t.Error("expected warning for FEAT-001")
	}
	if !entityIDs["FEAT-002"] {
		t.Error("expected warning for FEAT-002")
	}
}

func TestCheckFeatureChildConsistency_TasksWithNoParentSkipped(t *testing.T) {
	t.Parallel()

	features := []map[string]any{
		{"id": "FEAT-001", "status": "done"},
	}
	tasks := []map[string]any{
		// Task with no parent_feature should be ignored
		{"id": "TASK-001", "status": "active"},
		// Task with empty parent_feature should be ignored
		{"id": "TASK-002", "parent_feature": "", "status": "queued"},
	}

	result := CheckFeatureChildConsistency(features, tasks)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
	}
}

func TestCheckFeatureChildConsistency_FeatureWithEmptyID(t *testing.T) {
	t.Parallel()

	features := []map[string]any{
		{"status": "done"},
		{"id": "", "status": "done"},
	}
	tasks := []map[string]any{
		{"id": "TASK-001", "parent_feature": "FEAT-001", "status": "active"},
	}

	result := CheckFeatureChildConsistency(features, tasks)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
	}
}

func TestCheckFeatureChildConsistency_AllTaskTerminalStatuses(t *testing.T) {
	t.Parallel()

	// Verify that all three task terminal statuses are recognized.
	features := []map[string]any{
		{"id": "FEAT-001", "status": "cancelled"},
	}
	tasks := []map[string]any{
		{"id": "TASK-001", "parent_feature": "FEAT-001", "status": "done"},
		{"id": "TASK-002", "parent_feature": "FEAT-001", "status": "not-planned"},
		{"id": "TASK-003", "parent_feature": "FEAT-001", "status": "duplicate"},
	}

	result := CheckFeatureChildConsistency(features, tasks)

	// All children are terminal, feature is terminal → no warning
	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
	}
}

func TestCheckFeatureChildConsistency_DevelopingFeatureAllChildrenTerminal(t *testing.T) {
	t.Parallel()

	// "developing" is not an early state so should not be flagged even
	// when all children are terminal.
	features := []map[string]any{
		{"id": "FEAT-001", "status": "developing"},
	}
	tasks := []map[string]any{
		{"id": "TASK-001", "parent_feature": "FEAT-001", "status": "done"},
	}

	result := CheckFeatureChildConsistency(features, tasks)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
	}
}

func TestCheckFeatureChildConsistency_ReviewingFeatureNoFalseWarnings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		featureStatus string
	}{
		{"reviewing", "reviewing"},
		{"needs-rework", "needs-rework"},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_with_non_terminal_children", func(t *testing.T) {
			t.Parallel()

			// Feature in reviewing/needs-rework with non-terminal children is normal
			// (review states are not terminal/done, so no warning).
			features := []map[string]any{
				{"id": "FEAT-001", "status": tt.featureStatus},
			}
			tasks := []map[string]any{
				{"id": "TASK-001", "parent_feature": "FEAT-001", "status": "active"},
				{"id": "TASK-002", "parent_feature": "FEAT-001", "status": "done"},
			}

			result := CheckFeatureChildConsistency(features, tasks)

			if result.Status != SeverityOK {
				t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
			}
			if len(result.Issues) != 0 {
				t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
			}
		})

		t.Run(tt.name+"_with_all_terminal_children", func(t *testing.T) {
			t.Parallel()

			// Feature in reviewing/needs-rework with all children terminal is not
			// an early-state warning — these are active review states, not early states.
			features := []map[string]any{
				{"id": "FEAT-001", "status": tt.featureStatus},
			}
			tasks := []map[string]any{
				{"id": "TASK-001", "parent_feature": "FEAT-001", "status": "done"},
				{"id": "TASK-002", "parent_feature": "FEAT-001", "status": "not-planned"},
			}

			result := CheckFeatureChildConsistency(features, tasks)

			if result.Status != SeverityOK {
				t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
			}
			if len(result.Issues) != 0 {
				t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
			}
		})
	}
}
