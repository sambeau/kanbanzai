package health

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"kanbanzai/internal/git"
	"kanbanzai/internal/knowledge"
	"kanbanzai/internal/worktree"
)

// CheckWorktree checks worktree state consistency.
// Detects:
// - Error: Orphaned worktrees (path exists but no tracking record) - requires disk scan
// - Error: Missing worktrees (tracking record exists but path doesn't exist)
// - Warning: Active worktrees with no commits in 14+ days (requires branch status)
func CheckWorktree(repoPath string, worktrees []worktree.Record) CategoryResult {
	result := NewCategoryResult()

	for _, wt := range worktrees {
		// Check for missing worktrees (tracking record exists but path doesn't exist)
		if wt.Status == worktree.StatusActive {
			wtPath := wt.Path
			if !filepath.IsAbs(wtPath) && repoPath != "" {
				wtPath = filepath.Join(repoPath, wt.Path)
			}

			if _, err := os.Stat(wtPath); os.IsNotExist(err) {
				result.AddIssue(Issue{
					Severity: SeverityError,
					EntityID: wt.ID,
					Message:  fmt.Sprintf("worktree path does not exist: %s", wt.Path),
				})
			}
		}
	}

	return result
}

// CheckBranch checks branch health for each active worktree.
// Uses git.EvaluateBranchStatus for each worktree's branch.
// Detects:
// - Warning: Branch stale (no commits in X days)
// - Warning: Branch behind main by 50+ commits
// - Error: Branch behind main by 100+ commits
// - Error: Branch has merge conflicts
func CheckBranch(repoPath string, worktrees []worktree.Record, thresholds git.BranchThresholds) CategoryResult {
	result := NewCategoryResult()

	for _, wt := range worktrees {
		// Only check active worktrees
		if wt.Status != worktree.StatusActive {
			continue
		}

		// Skip if branch is empty
		if wt.Branch == "" {
			continue
		}

		status, err := git.EvaluateBranchStatus(repoPath, wt.Branch, thresholds)
		if err != nil {
			// Branch might not exist anymore - this is an error
			result.AddIssue(Issue{
				Severity: SeverityError,
				EntityID: wt.ID,
				Message:  fmt.Sprintf("failed to evaluate branch %s: %v", wt.Branch, err),
			})
			continue
		}

		// Add errors from branch status
		for _, errMsg := range status.Errors {
			result.AddIssue(Issue{
				Severity: SeverityError,
				EntityID: wt.ID,
				Message:  fmt.Sprintf("branch %s: %s", wt.Branch, errMsg),
			})
		}

		// Add warnings from branch status
		for _, warnMsg := range status.Warnings {
			result.AddIssue(Issue{
				Severity: SeverityWarning,
				EntityID: wt.ID,
				Message:  fmt.Sprintf("branch %s: %s", wt.Branch, warnMsg),
			})
		}
	}

	return result
}

// CheckKnowledgeStaleness checks for stale knowledge entries.
// Uses git.CheckStaleness for entries with git_anchors.
// Detects:
// - Warning: Entry has anchored files that were modified since last confirmation
func CheckKnowledgeStaleness(repoPath string, entries []map[string]any) CategoryResult {
	result := NewCategoryResult()

	for _, entry := range entries {
		// Extract ID
		id := getEntryID(entry)
		if id == "" {
			continue
		}

		// Skip retired entries
		status := knowledge.GetStatus(entry)
		if status == "retired" {
			continue
		}

		// Check staleness using git package
		stalenessInfo, err := git.CheckEntryStaleness(repoPath, entry)
		if err != nil {
			// Git errors are warnings, not errors - the entry might reference files
			// that don't exist anymore
			result.AddIssue(Issue{
				Severity: SeverityWarning,
				EntryID:  id,
				Message:  fmt.Sprintf("failed to check staleness: %v", err),
			})
			continue
		}

		if stalenessInfo.IsStale {
			result.AddIssue(Issue{
				Severity: SeverityWarning,
				EntryID:  id,
				Message:  stalenessInfo.StaleReason,
			})
		}
	}

	return result
}

// CheckKnowledgeTTL checks for entries approaching or past TTL expiry.
// Detects:
// - Warning: Entry TTL expires within 7 days
// - Error: Entry TTL expired and meets prune conditions
func CheckKnowledgeTTL(entries []map[string]any, now time.Time) CategoryResult {
	result := NewCategoryResult()
	config := knowledge.DefaultTTLConfig()
	warningThreshold := 7 * 24 * time.Hour // 7 days warning

	for _, entry := range entries {
		id := getEntryID(entry)
		if id == "" {
			continue
		}

		// Skip retired entries
		status := knowledge.GetStatus(entry)
		if status == "retired" {
			continue
		}

		// Check if entry should be pruned (TTL expired + meets conditions)
		pruneCondition := knowledge.CheckPruneCondition(entry, now, config)
		if pruneCondition.ShouldPrune {
			result.AddIssue(Issue{
				Severity: SeverityError,
				EntryID:  id,
				Message:  fmt.Sprintf("TTL expired and eligible for pruning: %s", pruneCondition.Reason),
			})
			continue
		}

		// Check if TTL expires soon
		ttlExpiresAt := knowledge.GetTTLExpiresAt(entry)
		if ttlExpiresAt.IsZero() {
			// Compute from last_used or created if not set
			lastUsed := knowledge.GetLastUsed(entry)
			if lastUsed.IsZero() {
				lastUsed = knowledge.GetCreatedAt(entry)
			}
			if lastUsed.IsZero() {
				continue
			}

			ttlDays := knowledge.GetTTLDays(entry)
			if ttlDays == 0 {
				tier := knowledge.GetTier(entry)
				ttlDays = knowledge.GetDefaultTTL(tier)
			}
			if ttlDays == 0 {
				continue
			}
			ttlExpiresAt = knowledge.ComputeTTLExpiry(lastUsed, ttlDays)
		}

		timeUntilExpiry := ttlExpiresAt.Sub(now)
		if timeUntilExpiry > 0 && timeUntilExpiry <= warningThreshold {
			daysUntil := int(timeUntilExpiry.Hours() / 24)
			result.AddIssue(Issue{
				Severity: SeverityWarning,
				EntryID:  id,
				Message:  fmt.Sprintf("TTL expires in %d days", daysUntil),
			})
		}
	}

	return result
}

// CheckKnowledgeConflicts checks for disputed entries requiring resolution.
// Detects:
// - Error: Entries with status "disputed"
func CheckKnowledgeConflicts(entries []map[string]any) CategoryResult {
	result := NewCategoryResult()

	for _, entry := range entries {
		id := getEntryID(entry)
		if id == "" {
			continue
		}

		status := knowledge.GetStatus(entry)
		if status == "disputed" {
			result.AddIssue(Issue{
				Severity: SeverityError,
				EntryID:  id,
				Message:  "entry is disputed and requires resolution",
			})
		}
	}

	return result
}

// CheckCleanup checks for items pending cleanup past grace period.
// Uses worktrees with status "merged" where now > cleanup_after.
// Detects:
// - Warning: Items past grace period but not yet cleaned
func CheckCleanup(worktrees []worktree.Record, now time.Time) CategoryResult {
	result := NewCategoryResult()

	for _, wt := range worktrees {
		// Only check merged worktrees
		if wt.Status != worktree.StatusMerged {
			continue
		}

		// Check if past cleanup_after
		if wt.CleanupAfter != nil && now.After(*wt.CleanupAfter) {
			daysPast := int(now.Sub(*wt.CleanupAfter).Hours() / 24)
			result.AddIssue(Issue{
				Severity: SeverityWarning,
				EntityID: wt.ID,
				Message:  fmt.Sprintf("worktree cleanup overdue by %d days", daysPast),
			})
		}
	}

	return result
}

// getEntryID extracts the ID from a knowledge entry fields map.
func getEntryID(fields map[string]any) string {
	if fields == nil {
		return ""
	}
	id, _ := fields["id"].(string)
	return id
}
