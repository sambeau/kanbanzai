// Package worktree provides tracking and management of Git worktrees
// for feature and bug development in isolated branches.
package worktree

import "time"

// Status is the lifecycle state of a worktree.
type Status string

const (
	// StatusActive indicates the worktree is in active development.
	StatusActive Status = "active"
	// StatusMerged indicates the worktree's branch has been merged.
	StatusMerged Status = "merged"
	// StatusAbandoned indicates the worktree was abandoned without merging.
	StatusAbandoned Status = "abandoned"
)

// ValidStatus returns true if s is a valid worktree status.
func ValidStatus(s Status) bool {
	switch s {
	case StatusActive, StatusMerged, StatusAbandoned:
		return true
	default:
		return false
	}
}

// Record is the storage representation of a worktree tracking record.
type Record struct {
	ID           string     // Worktree ID (ULID-based, prefix WT-)
	EntityID     string     // Associated feature or bug entity ID
	Branch       string     // Git branch name
	Path         string     // Filesystem path relative to repo root
	Status       Status     // Lifecycle status
	Created      time.Time  // When the worktree was created
	CreatedBy    string     // User who created the worktree
	MergedAt     *time.Time // Timestamp when merged (optional)
	CleanupAfter *time.Time // When to auto-delete (optional)
	FileHash     string     // SHA-256 hex digest of file contents at load time; used for optimistic locking
}

// Fields returns the record as a map for YAML serialization.
func (r Record) Fields() map[string]any {
	fields := map[string]any{
		"id":         r.ID,
		"entity_id":  r.EntityID,
		"branch":     r.Branch,
		"path":       r.Path,
		"status":     string(r.Status),
		"created":    r.Created.Format(time.RFC3339),
		"created_by": r.CreatedBy,
	}

	if r.MergedAt != nil {
		fields["merged_at"] = r.MergedAt.Format(time.RFC3339)
	}
	if r.CleanupAfter != nil {
		fields["cleanup_after"] = r.CleanupAfter.Format(time.RFC3339)
	}

	return fields
}

// FieldOrder returns the canonical field order for worktree records.
func FieldOrder() []string {
	return []string{
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
}
