package cleanup

import (
	"testing"
	"time"

	"kanbanzai/internal/worktree"
)

func TestScheduleCleanup(t *testing.T) {
	baseTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name            string
		gracePeriodDays int
		wantCleanupAt   time.Time
	}{
		{
			name:            "7 day grace period",
			gracePeriodDays: 7,
			wantCleanupAt:   baseTime.AddDate(0, 0, 7),
		},
		{
			name:            "0 day grace period (immediate)",
			gracePeriodDays: 0,
			wantCleanupAt:   baseTime,
		},
		{
			name:            "14 day grace period",
			gracePeriodDays: 14,
			wantCleanupAt:   baseTime.AddDate(0, 0, 14),
		},
		{
			name:            "1 day grace period",
			gracePeriodDays: 1,
			wantCleanupAt:   baseTime.AddDate(0, 0, 1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := &worktree.Record{
				ID:       "WT-TEST123",
				EntityID: "FEAT-001",
				Branch:   "feat/test",
				Path:     ".worktrees/feat-test",
				Status:   worktree.StatusMerged,
			}

			ScheduleCleanup(record, baseTime, tt.gracePeriodDays)

			if record.MergedAt == nil {
				t.Fatal("MergedAt should be set")
			}
			if !record.MergedAt.Equal(baseTime) {
				t.Errorf("MergedAt = %v, want %v", *record.MergedAt, baseTime)
			}

			if record.CleanupAfter == nil {
				t.Fatal("CleanupAfter should be set")
			}
			if !record.CleanupAfter.Equal(tt.wantCleanupAt) {
				t.Errorf("CleanupAfter = %v, want %v", *record.CleanupAfter, tt.wantCleanupAt)
			}
		})
	}
}

func TestScheduleAbandonedCleanup(t *testing.T) {
	abandonedAt := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	record := &worktree.Record{
		ID:       "WT-TEST123",
		EntityID: "FEAT-001",
		Branch:   "feat/test",
		Path:     ".worktrees/feat-test",
		Status:   worktree.StatusAbandoned,
	}

	ScheduleAbandonedCleanup(record, abandonedAt)

	if record.CleanupAfter == nil {
		t.Fatal("CleanupAfter should be set")
	}
	if !record.CleanupAfter.Equal(abandonedAt) {
		t.Errorf("CleanupAfter = %v, want %v (immediate cleanup)", *record.CleanupAfter, abandonedAt)
	}
}

func TestIsReadyForCleanup(t *testing.T) {
	now := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	pastTime := now.Add(-24 * time.Hour)
	futureTime := now.Add(24 * time.Hour)

	tests := []struct {
		name         string
		cleanupAfter *time.Time
		now          time.Time
		want         bool
	}{
		{
			name:         "cleanup time in the past",
			cleanupAfter: &pastTime,
			now:          now,
			want:         true,
		},
		{
			name:         "cleanup time is now",
			cleanupAfter: &now,
			now:          now,
			want:         true,
		},
		{
			name:         "cleanup time in the future",
			cleanupAfter: &futureTime,
			now:          now,
			want:         false,
		},
		{
			name:         "no cleanup time set",
			cleanupAfter: nil,
			now:          now,
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := &worktree.Record{
				ID:           "WT-TEST123",
				CleanupAfter: tt.cleanupAfter,
			}

			got := IsReadyForCleanup(record, tt.now)
			if got != tt.want {
				t.Errorf("IsReadyForCleanup() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetScheduleStatus(t *testing.T) {
	now := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	pastTime := now.Add(-24 * time.Hour)
	futureTime := now.Add(24 * time.Hour)

	tests := []struct {
		name         string
		status       worktree.Status
		cleanupAfter *time.Time
		want         string
	}{
		{
			name:         "merged and ready for cleanup",
			status:       worktree.StatusMerged,
			cleanupAfter: &pastTime,
			want:         "ready",
		},
		{
			name:         "merged and scheduled (within grace period)",
			status:       worktree.StatusMerged,
			cleanupAfter: &futureTime,
			want:         "scheduled",
		},
		{
			name:         "abandoned and ready for cleanup",
			status:       worktree.StatusAbandoned,
			cleanupAfter: &pastTime,
			want:         "pending_abandoned",
		},
		{
			name:         "abandoned and scheduled",
			status:       worktree.StatusAbandoned,
			cleanupAfter: &futureTime,
			want:         "scheduled",
		},
		{
			name:         "merged but no cleanup time",
			status:       worktree.StatusMerged,
			cleanupAfter: nil,
			want:         "",
		},
		{
			name:         "active worktree",
			status:       worktree.StatusActive,
			cleanupAfter: &pastTime,
			want:         "",
		},
		{
			name:         "merged cleanup time exactly now",
			status:       worktree.StatusMerged,
			cleanupAfter: &now,
			want:         "ready",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := &worktree.Record{
				ID:           "WT-TEST123",
				Status:       tt.status,
				CleanupAfter: tt.cleanupAfter,
			}

			got := GetScheduleStatus(record, now)
			if got != tt.want {
				t.Errorf("GetScheduleStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToSchedule(t *testing.T) {
	now := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	mergedAt := now.Add(-48 * time.Hour)
	cleanupAfter := now.Add(-24 * time.Hour)

	t.Run("creates schedule for ready merged worktree", func(t *testing.T) {
		record := &worktree.Record{
			ID:           "WT-TEST123",
			EntityID:     "FEAT-001",
			Status:       worktree.StatusMerged,
			MergedAt:     &mergedAt,
			CleanupAfter: &cleanupAfter,
		}

		schedule := ToSchedule(record, now)

		if schedule == nil {
			t.Fatal("expected schedule to be created")
		}
		if schedule.WorktreeID != "WT-TEST123" {
			t.Errorf("WorktreeID = %q, want %q", schedule.WorktreeID, "WT-TEST123")
		}
		if schedule.EntityID != "FEAT-001" {
			t.Errorf("EntityID = %q, want %q", schedule.EntityID, "FEAT-001")
		}
		if !schedule.MergedAt.Equal(mergedAt) {
			t.Errorf("MergedAt = %v, want %v", schedule.MergedAt, mergedAt)
		}
		if !schedule.CleanupAfter.Equal(cleanupAfter) {
			t.Errorf("CleanupAfter = %v, want %v", schedule.CleanupAfter, cleanupAfter)
		}
		if schedule.Status != "ready" {
			t.Errorf("Status = %q, want %q", schedule.Status, "ready")
		}
	})

	t.Run("creates schedule for abandoned worktree", func(t *testing.T) {
		abandonedTime := now.Add(-1 * time.Hour)
		record := &worktree.Record{
			ID:           "WT-TEST456",
			EntityID:     "BUG-002",
			Status:       worktree.StatusAbandoned,
			CleanupAfter: &abandonedTime,
		}

		schedule := ToSchedule(record, now)

		if schedule == nil {
			t.Fatal("expected schedule to be created")
		}
		if schedule.Status != "pending_abandoned" {
			t.Errorf("Status = %q, want %q", schedule.Status, "pending_abandoned")
		}
		if schedule.MergedAt.IsZero() == false {
			t.Errorf("MergedAt should be zero for abandoned worktree, got %v", schedule.MergedAt)
		}
	})

	t.Run("returns nil for active worktree", func(t *testing.T) {
		record := &worktree.Record{
			ID:       "WT-TEST789",
			EntityID: "FEAT-003",
			Status:   worktree.StatusActive,
		}

		schedule := ToSchedule(record, now)

		if schedule != nil {
			t.Error("expected nil schedule for active worktree")
		}
	})

	t.Run("returns nil for worktree without cleanup time", func(t *testing.T) {
		record := &worktree.Record{
			ID:       "WT-TEST101",
			EntityID: "FEAT-004",
			Status:   worktree.StatusMerged,
		}

		schedule := ToSchedule(record, now)

		if schedule != nil {
			t.Error("expected nil schedule when no cleanup time set")
		}
	})

	t.Run("creates schedule for scheduled worktree within grace period", func(t *testing.T) {
		futureCleanup := now.Add(24 * time.Hour)
		record := &worktree.Record{
			ID:           "WT-SCHEDULED",
			EntityID:     "FEAT-005",
			Status:       worktree.StatusMerged,
			MergedAt:     &now,
			CleanupAfter: &futureCleanup,
		}

		schedule := ToSchedule(record, now)

		if schedule == nil {
			t.Fatal("expected schedule to be created")
		}
		if schedule.Status != "scheduled" {
			t.Errorf("Status = %q, want %q", schedule.Status, "scheduled")
		}
	})
}

func TestScheduleCleanup_IntegrationWithMarkMerged(t *testing.T) {
	// Verify that ScheduleCleanup produces compatible results with Record.MarkMerged
	mergedAt := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	gracePeriodDays := 7

	// Using ScheduleCleanup
	record1 := &worktree.Record{
		ID:       "WT-TEST1",
		EntityID: "FEAT-001",
		Branch:   "feat/test",
		Path:     ".worktrees/feat-test",
		Status:   worktree.StatusMerged,
	}
	ScheduleCleanup(record1, mergedAt, gracePeriodDays)

	// Using MarkMerged
	record2 := &worktree.Record{
		ID:       "WT-TEST2",
		EntityID: "FEAT-002",
		Branch:   "feat/test2",
		Path:     ".worktrees/feat-test2",
		Status:   worktree.StatusActive,
	}
	record2.MarkMerged(mergedAt, gracePeriodDays)

	// Both should have the same cleanup time behavior
	if !record1.CleanupAfter.Equal(*record2.CleanupAfter) {
		t.Errorf("CleanupAfter mismatch: ScheduleCleanup=%v, MarkMerged=%v",
			*record1.CleanupAfter, *record2.CleanupAfter)
	}
}
