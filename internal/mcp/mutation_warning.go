package mcp

import (
	"fmt"

	"github.com/sambeau/kanbanzai/internal/id"
	"github.com/sambeau/kanbanzai/internal/service"
	"github.com/sambeau/kanbanzai/internal/worktree"
)

// activeBugWorktreeWarning checks whether the caller is mutating the repo root
// (entityID empty) while one or more in-progress bugs have active worktrees.
// Returns a warning string ready for inclusion in the tool response, or empty
// string if there is nothing to warn about.
//
// FR-206 through FR-210: non-blocking informational warning.
func activeBugWorktreeWarning(entitySvc *service.EntityService, worktreeStore *worktree.Store, entityID string) string {
	if entityID != "" || entitySvc == nil || worktreeStore == nil {
		return ""
	}

	// List bugs with status "in-progress".
	bugs, err := entitySvc.ListEntitiesFiltered(service.ListFilteredInput{
		Type:   "bug",
		Status: "in-progress",
	})
	if err != nil || len(bugs) == 0 {
		return ""
	}

	// Collect bugs that have active worktrees.
	type bugInfo struct {
		displayID string
		canonical string
		path      string
	}
	var active []bugInfo
	for _, bug := range bugs {
		record, err := worktreeStore.GetByEntityID(bug.ID)
		if err != nil || record == nil || record.Status != worktree.StatusActive {
			continue
		}
		active = append(active, bugInfo{
			displayID: id.FormatFullDisplay(bug.ID),
			canonical: bug.ID,
			path:      record.Path,
		})
	}

	if len(active) == 0 {
		return ""
	}

	first := active[0]
	warning := fmt.Sprintf(
		"warning: bug %s is in-progress with an active worktree at %s. Consider scoping your edit with entity_id: %q to isolate changes.",
		first.displayID, first.path, first.canonical,
	)

	if len(active) > 1 {
		rest := len(active) - 1
		warning += fmt.Sprintf(" (and %d other active bug worktree%s)", rest, pluralS(rest))
	}

	return warning
}

func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
