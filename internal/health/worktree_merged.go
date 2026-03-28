package health

import (
	"bytes"
	"fmt"
	"os/exec"

	"github.com/sambeau/kanbanzai/internal/git"
	"github.com/sambeau/kanbanzai/internal/worktree"
)

// CheckWorktreeBranchMerged checks for active worktrees whose branch is already
// merged into main. A branch is considered merged if its HEAD is an ancestor of
// the default branch HEAD.
//
// This is a best-effort check — if git operations fail (e.g., branch doesn't
// exist locally, not in a git repo), the check is skipped gracefully rather
// than erroring.
//
// Detects:
//   - Warning: Active worktree whose branch is already merged into main
func CheckWorktreeBranchMerged(repoPath string, worktrees []worktree.Record) CategoryResult {
	result := NewCategoryResult()

	if repoPath == "" {
		return result
	}

	// Find the default branch (main or master).
	defaultBranch, err := git.GetDefaultBranch(repoPath)
	if err != nil {
		return result // can't determine default branch; skip gracefully
	}

	for _, wt := range worktrees {
		if wt.Status != worktree.StatusActive {
			continue
		}
		if wt.Branch == "" {
			continue
		}

		merged, err := isBranchAncestorOf(repoPath, wt.Branch, defaultBranch)
		if err != nil {
			continue // best-effort: skip on any error
		}

		if merged {
			result.AddIssue(Issue{
				Severity: SeverityWarning,
				EntityID: wt.EntityID,
				Message: fmt.Sprintf(
					"worktree for %s is active but branch %s is already merged into main",
					wt.EntityID, wt.Branch,
				),
			})
		}
	}

	return result
}

// isBranchAncestorOf checks whether branch is an ancestor of base using
// git merge-base --is-ancestor. Returns true when branch's HEAD is reachable
// from base's HEAD (i.e., the branch has been fully merged into base).
//
// Exit code 0 → ancestor (merged). Exit code 1 → not an ancestor.
// Any other error is returned to the caller.
func isBranchAncestorOf(repoPath, branch, base string) (bool, error) {
	cmd := exec.Command("git", "merge-base", "--is-ancestor", branch, base)
	cmd.Dir = repoPath

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		return true, nil // exit 0: branch is ancestor of base
	}

	// Exit code 1 means "not an ancestor" — not an error condition.
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		return false, nil
	}

	// Any other failure (not a repo, branch missing, etc.).
	return false, err
}
