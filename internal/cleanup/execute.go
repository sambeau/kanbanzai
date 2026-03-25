// Package cleanup provides post-merge cleanup operations for worktrees.
package cleanup

import (
	"strings"
	"time"

	"kanbanzai/internal/worktree"
)

// CleanupResult contains the result of cleaning up a worktree.
type CleanupResult struct {
	WorktreeID          string
	Branch              string
	Path                string
	RemoteBranchDeleted bool
	Success             bool
	Error               error
}

// CleanupOptions configures cleanup execution.
type CleanupOptions struct {
	DryRun             bool
	DeleteRemoteBranch bool // From config: auto_delete_remote_branch
	ForceRemove        bool // Force removal even if worktree has uncommitted changes
}

// ExecuteCleanup cleans up a single worktree.
// Steps:
// 1. Remove Git worktree: git worktree remove {path}
// 2. Delete local branch: git branch -d {branch}
// 3. Delete remote branch (if configured): git push origin --delete {branch}
// 4. Delete worktree tracking record
func ExecuteCleanup(store *worktree.Store, git *worktree.Git, record worktree.Record, opts CleanupOptions) CleanupResult {
	result := CleanupResult{
		WorktreeID: record.ID,
		Branch:     record.Branch,
		Path:       record.Path,
		Success:    false,
	}

	if opts.DryRun {
		result.Success = true
		result.RemoteBranchDeleted = opts.DeleteRemoteBranch
		return result
	}

	// Step 1: Remove Git worktree
	if err := git.RemoveWorktree(record.Path, opts.ForceRemove); err != nil {
		// If the worktree doesn't exist on disk, continue with cleanup
		// This handles cases where the directory was manually deleted
		if !isWorktreeNotFoundError(err) {
			result.Error = err
			return result
		}
	}

	// Prune stale worktree entries
	_ = git.PruneWorktrees()

	// Step 2: Delete local branch
	// Use force=false for merged branches, they should be safe to delete
	// For abandoned branches, the caller should set ForceRemove
	if err := git.DeleteBranch(record.Branch, opts.ForceRemove); err != nil {
		// If the branch doesn't exist, continue
		if !isBranchNotFoundError(err) {
			result.Error = err
			return result
		}
	}

	// Step 3: Delete remote branch (if configured)
	if opts.DeleteRemoteBranch {
		if err := git.DeleteRemoteBranch("origin", record.Branch); err != nil {
			// If the remote branch doesn't exist, continue
			if !isRemoteBranchNotFoundError(err) {
				// Log error but don't fail the cleanup
				// Remote branch deletion is best-effort
				result.RemoteBranchDeleted = false
			}
		} else {
			result.RemoteBranchDeleted = true
		}
	}

	// Step 4: Delete worktree tracking record
	if err := store.Delete(record.ID); err != nil {
		result.Error = err
		return result
	}

	result.Success = true
	return result
}

// ExecuteAllReady cleans up all items ready for cleanup.
func ExecuteAllReady(store *worktree.Store, git *worktree.Git, opts CleanupOptions) []CleanupResult {
	records, err := store.List()
	if err != nil {
		return []CleanupResult{{
			Success: false,
			Error:   err,
		}}
	}

	now := time.Now()
	var results []CleanupResult

	for _, record := range records {
		ready := IsReadyForCleanup(&record, now)
		if !ready && record.Status == worktree.StatusAbandoned {
			// Abandoned worktrees without CleanupAfter are always ready
			ready = true
		}
		if !ready {
			continue
		}

		// Abandoned worktrees can be force-removed
		cleanupOpts := opts
		if record.Status == worktree.StatusAbandoned {
			cleanupOpts.ForceRemove = true
		}

		result := ExecuteCleanup(store, git, record, cleanupOpts)
		results = append(results, result)
	}

	return results
}

// isWorktreeNotFoundError checks if the error indicates the worktree doesn't exist.
func isWorktreeNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// Git returns various messages for missing worktrees
	return strings.Contains(errStr, "is not a working tree") ||
		strings.Contains(errStr, "not a valid directory") ||
		strings.Contains(errStr, "does not exist")
}

// isBranchNotFoundError checks if the error indicates the branch doesn't exist.
func isBranchNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "not found") ||
		strings.Contains(errStr, "branch") && strings.Contains(errStr, "not found")
}

// isRemoteBranchNotFoundError checks if the error indicates the remote branch doesn't exist.
func isRemoteBranchNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "remote ref does not exist") ||
		strings.Contains(errStr, "unable to delete") && strings.Contains(errStr, "remote ref")
}
