// Package cleanup provides post-merge cleanup scheduling and execution
// for Git worktrees.
package cleanup

import (
	"time"

	"kanbanzai/internal/worktree"
)

// CleanupSchedule represents an item scheduled for cleanup.
type CleanupSchedule struct {
	WorktreeID   string
	EntityID     string
	MergedAt     time.Time
	CleanupAfter time.Time
	Status       string // "scheduled", "ready", "pending_abandoned"
}

// ScheduleCleanup sets cleanup_after on a worktree record after merge.
// It updates the record's MergedAt and CleanupAfter fields based on the
// grace period. The record's Status should already be set to StatusMerged
// before calling this function.
//
// For abandoned worktrees, pass gracePeriodDays = 0 for immediate cleanup.
func ScheduleCleanup(record *worktree.Record, mergedAt time.Time, gracePeriodDays int) {
	record.MergedAt = &mergedAt

	cleanupAfter := mergedAt.AddDate(0, 0, gracePeriodDays)
	record.CleanupAfter = &cleanupAfter
}

// ScheduleAbandonedCleanup schedules an abandoned worktree for immediate cleanup.
// Abandoned worktrees have no grace period.
func ScheduleAbandonedCleanup(record *worktree.Record, abandonedAt time.Time) {
	// Abandoned worktrees are cleaned up immediately (no grace period).
	record.CleanupAfter = &abandonedAt
}

// IsReadyForCleanup checks if a worktree is ready to be cleaned up.
// A worktree is ready if:
// - It has a CleanupAfter time set, AND
// - The current time is at or after CleanupAfter
func IsReadyForCleanup(record *worktree.Record, now time.Time) bool {
	if record.CleanupAfter == nil {
		return false
	}
	return !now.Before(*record.CleanupAfter)
}

// GetScheduleStatus returns the cleanup status for a worktree record.
// Returns:
// - "ready" if past grace period and ready for cleanup
// - "scheduled" if within grace period
// - "pending_abandoned" if abandoned and ready for cleanup
// - "" if not scheduled for cleanup
func GetScheduleStatus(record *worktree.Record, now time.Time) string {
	if record.CleanupAfter == nil {
		return ""
	}

	isReady := !now.Before(*record.CleanupAfter)

	if record.Status == worktree.StatusAbandoned {
		if isReady {
			return "pending_abandoned"
		}
		return "scheduled"
	}

	if record.Status == worktree.StatusMerged {
		if isReady {
			return "ready"
		}
		return "scheduled"
	}

	return ""
}

// ToSchedule converts a worktree record to a CleanupSchedule.
// Returns nil if the record is not scheduled for cleanup.
func ToSchedule(record *worktree.Record, now time.Time) *CleanupSchedule {
	status := GetScheduleStatus(record, now)
	if status == "" {
		return nil
	}

	schedule := &CleanupSchedule{
		WorktreeID:   record.ID,
		EntityID:     record.EntityID,
		CleanupAfter: *record.CleanupAfter,
		Status:       status,
	}

	if record.MergedAt != nil {
		schedule.MergedAt = *record.MergedAt
	}

	return schedule
}
