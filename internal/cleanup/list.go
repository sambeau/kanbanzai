// Package cleanup provides post-merge cleanup scheduling and execution
// for worktrees and their associated branches.
package cleanup

import (
	"sort"
	"time"

	"github.com/sambeau/kanbanzai/internal/worktree"
)

// CleanupItem represents an item in the cleanup list.
type CleanupItem struct {
	WorktreeID   string
	EntityID     string
	Branch       string
	Path         string
	MergedAt     time.Time
	CleanupAfter time.Time
	Status       string // "ready" (past grace), "scheduled" (within grace), "abandoned"
}

// ListOptions configures which items to list.
type ListOptions struct {
	IncludePending   bool // Include items past grace period (ready for cleanup)
	IncludeScheduled bool // Include items within grace period
	IncludeAbandoned bool // Include abandoned worktrees
}

// ListCleanupItems returns items pending cleanup based on options.
// Items are sorted by cleanup time (earliest first), then by ID.
func ListCleanupItems(records []worktree.Record, now time.Time, opts ListOptions) []CleanupItem {
	var items []CleanupItem

	for _, r := range records {
		var item *CleanupItem

		switch r.Status {
		case worktree.StatusMerged:
			item = listMergedItem(r, now, opts)
		case worktree.StatusAbandoned:
			if opts.IncludeAbandoned {
				item = &CleanupItem{
					WorktreeID: r.ID,
					EntityID:   r.EntityID,
					Branch:     r.Branch,
					Path:       r.Path,
					Status:     "abandoned",
				}
			}
		default:
			// Active worktrees are not included in cleanup list
			continue
		}

		if item != nil {
			items = append(items, *item)
		}
	}

	// Sort by cleanup time (earliest first), then by ID for determinism
	sort.Slice(items, func(i, j int) bool {
		// Abandoned items (no cleanup time) sort first
		if items[i].CleanupAfter.IsZero() && !items[j].CleanupAfter.IsZero() {
			return true
		}
		if !items[i].CleanupAfter.IsZero() && items[j].CleanupAfter.IsZero() {
			return false
		}
		if items[i].CleanupAfter.Equal(items[j].CleanupAfter) {
			return items[i].WorktreeID < items[j].WorktreeID
		}
		return items[i].CleanupAfter.Before(items[j].CleanupAfter)
	})

	return items
}

// listMergedItem creates a CleanupItem for a merged worktree if it matches the options.
func listMergedItem(r worktree.Record, now time.Time, opts ListOptions) *CleanupItem {
	item := CleanupItem{
		WorktreeID: r.ID,
		EntityID:   r.EntityID,
		Branch:     r.Branch,
		Path:       r.Path,
	}

	if r.MergedAt != nil {
		item.MergedAt = *r.MergedAt
	}

	if r.CleanupAfter != nil {
		item.CleanupAfter = *r.CleanupAfter
	}

	// Determine if ready (past grace) or scheduled (within grace)
	if r.CleanupAfter != nil && !r.CleanupAfter.IsZero() {
		if now.After(*r.CleanupAfter) || now.Equal(*r.CleanupAfter) {
			item.Status = "ready"
			if opts.IncludePending {
				return &item
			}
		} else {
			item.Status = "scheduled"
			if opts.IncludeScheduled {
				return &item
			}
		}
	} else {
		// No cleanup_after set - treat as ready for cleanup
		item.Status = "ready"
		if opts.IncludePending {
			return &item
		}
	}

	return nil
}

// ListReadyItems is a convenience function that returns only items ready for cleanup.
func ListReadyItems(records []worktree.Record, now time.Time) []CleanupItem {
	return ListCleanupItems(records, now, ListOptions{
		IncludePending:   true,
		IncludeAbandoned: true,
	})
}
