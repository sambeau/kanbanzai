package git

import "time"

// GitAnchor represents a file path that a knowledge entry is anchored to.
// When the anchored file changes, the knowledge entry may be stale.
type GitAnchor struct {
	// Path is the file path relative to the repository root.
	// For example: "internal/api/handler.go"
	Path string
}

// StalenessInfo contains staleness detection results for a knowledge entry.
type StalenessInfo struct {
	// IsStale is true if any anchored file was modified after last_confirmed.
	IsStale bool

	// StaleReason provides a human-readable explanation of why the entry is stale.
	// Empty if not stale.
	StaleReason string

	// StaleFiles lists the files that caused staleness.
	// Empty if not stale.
	StaleFiles []StaleFile

	// LastConfirmed is the timestamp when the entry was last confirmed.
	// This is copied from the knowledge entry for context.
	LastConfirmed time.Time
}

// StaleFile contains information about a file that caused staleness.
type StaleFile struct {
	// Path is the file path relative to the repository root.
	Path string

	// ModifiedAt is when the file was last modified (commit timestamp).
	ModifiedAt time.Time

	// Commit is the SHA of the commit that last modified the file.
	Commit string
}
