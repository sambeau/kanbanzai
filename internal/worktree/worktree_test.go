package worktree

import (
	"testing"
	"time"
)

func TestValidStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status Status
		want   bool
	}{
		{"active is valid", StatusActive, true},
		{"merged is valid", StatusMerged, true},
		{"abandoned is valid", StatusAbandoned, true},
		{"empty is invalid", Status(""), false},
		{"unknown is invalid", Status("unknown"), false},
		{"pending is invalid", Status("pending"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ValidStatus(tt.status); got != tt.want {
				t.Errorf("ValidStatus(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestRecord_Fields(t *testing.T) {
	t.Parallel()

	created := time.Date(2025, 1, 27, 10, 0, 0, 0, time.UTC)
	record := Record{
		ID:        "WT-01JX123456789",
		EntityID:  "FEAT-01JX987654321",
		Branch:    "feature/FEAT-01JX987654321-user-profiles",
		Path:      ".worktrees/FEAT-01JX987654321-user-profiles",
		Status:    StatusActive,
		Created:   created,
		CreatedBy: "sambeau",
	}

	fields := record.Fields()

	// Check required fields
	if fields["id"] != "WT-01JX123456789" {
		t.Errorf("fields[id] = %v, want %v", fields["id"], "WT-01JX123456789")
	}
	if fields["entity_id"] != "FEAT-01JX987654321" {
		t.Errorf("fields[entity_id] = %v, want %v", fields["entity_id"], "FEAT-01JX987654321")
	}
	if fields["branch"] != "feature/FEAT-01JX987654321-user-profiles" {
		t.Errorf("fields[branch] = %v, want %v", fields["branch"], "feature/FEAT-01JX987654321-user-profiles")
	}
	if fields["path"] != ".worktrees/FEAT-01JX987654321-user-profiles" {
		t.Errorf("fields[path] = %v, want %v", fields["path"], ".worktrees/FEAT-01JX987654321-user-profiles")
	}
	if fields["status"] != "active" {
		t.Errorf("fields[status] = %v, want %v", fields["status"], "active")
	}
	if fields["created"] != "2025-01-27T10:00:00Z" {
		t.Errorf("fields[created] = %v, want %v", fields["created"], "2025-01-27T10:00:00Z")
	}
	if fields["created_by"] != "sambeau" {
		t.Errorf("fields[created_by] = %v, want %v", fields["created_by"], "sambeau")
	}

	// Optional fields should not be present when nil
	if _, ok := fields["merged_at"]; ok {
		t.Error("fields[merged_at] should not be present when MergedAt is nil")
	}
	if _, ok := fields["cleanup_after"]; ok {
		t.Error("fields[cleanup_after] should not be present when CleanupAfter is nil")
	}
}

func TestRecord_Fields_WithOptionalFields(t *testing.T) {
	t.Parallel()

	created := time.Date(2025, 1, 27, 10, 0, 0, 0, time.UTC)
	mergedAt := time.Date(2025, 1, 28, 15, 30, 0, 0, time.UTC)
	cleanupAfter := time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC)

	record := Record{
		ID:           "WT-01JX123456789",
		EntityID:     "FEAT-01JX987654321",
		Branch:       "feature/FEAT-01JX987654321-user-profiles",
		Path:         ".worktrees/FEAT-01JX987654321-user-profiles",
		Status:       StatusMerged,
		Created:      created,
		CreatedBy:    "sambeau",
		MergedAt:     &mergedAt,
		CleanupAfter: &cleanupAfter,
	}

	fields := record.Fields()

	if fields["merged_at"] != "2025-01-28T15:30:00Z" {
		t.Errorf("fields[merged_at] = %v, want %v", fields["merged_at"], "2025-01-28T15:30:00Z")
	}
	if fields["cleanup_after"] != "2025-02-28T00:00:00Z" {
		t.Errorf("fields[cleanup_after] = %v, want %v", fields["cleanup_after"], "2025-02-28T00:00:00Z")
	}
}

func TestFieldOrder(t *testing.T) {
	t.Parallel()

	order := FieldOrder()

	// Check expected order
	expected := []string{
		"id",
		"entity_id",
		"branch",
		"path",
		"status",
		"created",
		"created_by",
		"merged_at",
		"cleanup_after",
	}

	if len(order) != len(expected) {
		t.Fatalf("FieldOrder() returned %d fields, want %d", len(order), len(expected))
	}

	for i, field := range expected {
		if order[i] != field {
			t.Errorf("FieldOrder()[%d] = %q, want %q", i, order[i], field)
		}
	}
}

func TestFieldOrder_ContainsAllRecordFields(t *testing.T) {
	t.Parallel()

	// Create a record with all fields populated
	created := time.Date(2025, 1, 27, 10, 0, 0, 0, time.UTC)
	mergedAt := time.Date(2025, 1, 28, 15, 30, 0, 0, time.UTC)
	cleanupAfter := time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC)

	record := Record{
		ID:           "WT-01JX123456789",
		EntityID:     "FEAT-01JX987654321",
		Branch:       "feature/FEAT-01JX987654321-user-profiles",
		Path:         ".worktrees/FEAT-01JX987654321-user-profiles",
		Status:       StatusMerged,
		Created:      created,
		CreatedBy:    "sambeau",
		MergedAt:     &mergedAt,
		CleanupAfter: &cleanupAfter,
	}

	fields := record.Fields()
	order := FieldOrder()

	// Every field in Fields() should be in FieldOrder()
	orderSet := make(map[string]bool)
	for _, f := range order {
		orderSet[f] = true
	}

	for key := range fields {
		if !orderSet[key] {
			t.Errorf("Field %q from Record.Fields() is not in FieldOrder()", key)
		}
	}
}

func TestRecord_MarkMerged(t *testing.T) {
	t.Parallel()

	created := time.Date(2025, 1, 27, 10, 0, 0, 0, time.UTC)
	mergedAt := time.Date(2025, 1, 28, 15, 30, 0, 0, time.UTC)

	tests := []struct {
		name            string
		gracePeriodDays int
		wantCleanupAt   time.Time
	}{
		{
			name:            "7 day grace period",
			gracePeriodDays: 7,
			wantCleanupAt:   mergedAt.AddDate(0, 0, 7),
		},
		{
			name:            "0 day grace period",
			gracePeriodDays: 0,
			wantCleanupAt:   mergedAt,
		},
		{
			name:            "14 day grace period",
			gracePeriodDays: 14,
			wantCleanupAt:   mergedAt.AddDate(0, 0, 14),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			record := Record{
				ID:        "WT-01JX123456789",
				EntityID:  "FEAT-01JX987654321",
				Branch:    "feature/test",
				Path:      ".worktrees/test",
				Status:    StatusActive,
				Created:   created,
				CreatedBy: "test-user",
			}

			record.MarkMerged(mergedAt, tt.gracePeriodDays)

			if record.Status != StatusMerged {
				t.Errorf("Status = %v, want %v", record.Status, StatusMerged)
			}
			if record.MergedAt == nil {
				t.Fatal("MergedAt should not be nil")
			}
			if !record.MergedAt.Equal(mergedAt) {
				t.Errorf("MergedAt = %v, want %v", *record.MergedAt, mergedAt)
			}
			if record.CleanupAfter == nil {
				t.Fatal("CleanupAfter should not be nil")
			}
			if !record.CleanupAfter.Equal(tt.wantCleanupAt) {
				t.Errorf("CleanupAfter = %v, want %v", *record.CleanupAfter, tt.wantCleanupAt)
			}
		})
	}
}

func TestRecord_MarkAbandoned(t *testing.T) {
	t.Parallel()

	created := time.Date(2025, 1, 27, 10, 0, 0, 0, time.UTC)

	record := Record{
		ID:        "WT-01JX123456789",
		EntityID:  "FEAT-01JX987654321",
		Branch:    "feature/test",
		Path:      ".worktrees/test",
		Status:    StatusActive,
		Created:   created,
		CreatedBy: "test-user",
	}

	record.MarkAbandoned()

	if record.Status != StatusAbandoned {
		t.Errorf("Status = %v, want %v", record.Status, StatusAbandoned)
	}
	if record.MergedAt != nil {
		t.Errorf("MergedAt should be nil for abandoned worktree, got %v", *record.MergedAt)
	}
	if record.CleanupAfter != nil {
		t.Errorf("CleanupAfter should be nil for abandoned worktree, got %v", *record.CleanupAfter)
	}
}

func TestRecord_MarkAbandoned_ClearsPreviousMergeInfo(t *testing.T) {
	t.Parallel()

	created := time.Date(2025, 1, 27, 10, 0, 0, 0, time.UTC)
	mergedAt := time.Date(2025, 1, 28, 15, 30, 0, 0, time.UTC)
	cleanupAfter := time.Date(2025, 2, 4, 15, 30, 0, 0, time.UTC)

	record := Record{
		ID:           "WT-01JX123456789",
		EntityID:     "FEAT-01JX987654321",
		Branch:       "feature/test",
		Path:         ".worktrees/test",
		Status:       StatusMerged,
		Created:      created,
		CreatedBy:    "test-user",
		MergedAt:     &mergedAt,
		CleanupAfter: &cleanupAfter,
	}

	record.MarkAbandoned()

	if record.Status != StatusAbandoned {
		t.Errorf("Status = %v, want %v", record.Status, StatusAbandoned)
	}
	if record.MergedAt != nil {
		t.Error("MergedAt should be cleared when marking as abandoned")
	}
	if record.CleanupAfter != nil {
		t.Error("CleanupAfter should be cleared when marking as abandoned")
	}
}
