package health

import (
	"fmt"

	"github.com/sambeau/kanbanzai/internal/worktree"
)

// CheckBugWorktree checks that every in-progress bug has an active worktree.
// Detects:
//   - Warning: Bug with status "in-progress" but no active worktree record
func CheckBugWorktree(bugs []map[string]any, worktrees []worktree.Record) CategoryResult {
	result := NewCategoryResult()

	// Build a set of entity IDs that have active worktrees.
	activeEntityIDs := make(map[string]struct{}, len(worktrees))
	for _, wt := range worktrees {
		if wt.Status == worktree.StatusActive && wt.EntityID != "" {
			activeEntityIDs[wt.EntityID] = struct{}{}
		}
	}

	for _, bug := range bugs {
		bugID, _ := bug["id"].(string)
		if bugID == "" {
			continue
		}

		status, _ := bug["status"].(string)
		if status != "in-progress" {
			continue
		}

		if _, ok := activeEntityIDs[bugID]; !ok {
			result.AddIssue(Issue{
				Severity: SeverityWarning,
				EntityID: bugID,
				Message:  fmt.Sprintf("bug %s is in-progress but has no active worktree — changes may not be isolated", bugID),
			})
		}
	}

	return result
}
