// Package git provides Git operations for anchoring and staleness detection.
package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ErrBranchNotFound is returned when the specified branch does not exist.
var ErrBranchNotFound = fmt.Errorf("branch not found")

// BranchMetrics contains health metrics for a Git branch.
type BranchMetrics struct {
	// Branch is the name of the branch being measured.
	Branch string

	// BranchAgeDays is the number of days since the branch was created.
	BranchAgeDays int

	// CommitsBehindMain is the number of commits on main not in this branch.
	CommitsBehindMain int

	// CommitsAheadOfMain is the number of commits on this branch not in main.
	CommitsAheadOfMain int

	// LastCommitAt is the time of the most recent commit on the branch.
	LastCommitAt time.Time

	// LastCommitAgeDays is the number of days since the last commit.
	LastCommitAgeDays int

	// HasConflicts indicates whether the branch has merge conflicts with main.
	HasConflicts bool
}

// BranchStatus combines metrics with threshold evaluation.
type BranchStatus struct {
	// Metrics contains the computed branch metrics.
	Metrics BranchMetrics

	// Warnings contains warning-level issues.
	Warnings []string

	// Errors contains error-level issues.
	Errors []string
}

// BranchThresholds configures staleness and drift thresholds for branch evaluation.
type BranchThresholds struct {
	// StaleAfterDays is the number of days after which a branch with no commits is considered stale.
	StaleAfterDays int

	// DriftWarningCommits is the number of commits behind main that triggers a warning.
	DriftWarningCommits int

	// DriftErrorCommits is the number of commits behind main that triggers an error.
	DriftErrorCommits int
}

// DefaultBranchThresholds returns the default thresholds from the spec.
func DefaultBranchThresholds() BranchThresholds {
	return BranchThresholds{
		StaleAfterDays:      14,
		DriftWarningCommits: 50,
		DriftErrorCommits:   100,
	}
}

// GetBranchCreationTime returns when a branch was created (time of first commit on branch).
// This finds the first commit that is reachable from branch but not from the base branch.
// If the branch has no unique commits (is identical to base), returns the tip commit time.
func GetBranchCreationTime(repoPath, branch string) (time.Time, error) {
	// First check if the branch exists
	if err := checkBranchExists(repoPath, branch); err != nil {
		return time.Time{}, err
	}

	// Find the base branch (main or master)
	base, err := getDefaultBranch(repoPath)
	if err != nil {
		// If we can't find a default branch, use the branch's first commit
		return getFirstCommitTime(repoPath, branch)
	}

	// Get the first commit unique to this branch (oldest commit not on base)
	// git log base..branch --reverse --format=%ct | head -1
	cmd := exec.Command("git", "log", base+".."+branch, "--reverse", "--format=%ct")
	cmd.Dir = repoPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if strings.Contains(stderrStr, "not a git repository") {
			return time.Time{}, ErrNotARepository
		}
		return time.Time{}, fmt.Errorf("git log: %w", err)
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		// Branch has no unique commits; use the tip commit time
		_, commitTime, err := GetBranchLastCommit(repoPath, branch)
		return commitTime, err
	}

	// Take the first line (oldest unique commit)
	lines := strings.Split(output, "\n")
	timestamp, err := strconv.ParseInt(lines[0], 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse timestamp: %w", err)
	}

	return time.Unix(timestamp, 0).UTC(), nil
}

// GetBranchLastCommit returns the SHA and timestamp of the last commit on a branch.
func GetBranchLastCommit(repoPath, branch string) (sha string, at time.Time, err error) {
	// git log -1 --format=%H%n%ct <branch>
	cmd := exec.Command("git", "log", "-1", "--format=%H%n%ct", branch)
	cmd.Dir = repoPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if strings.Contains(stderrStr, "not a git repository") {
			return "", time.Time{}, ErrNotARepository
		}
		if strings.Contains(stderrStr, "unknown revision") || strings.Contains(stderrStr, "bad revision") {
			return "", time.Time{}, ErrBranchNotFound
		}
		return "", time.Time{}, fmt.Errorf("git log: %w", err)
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return "", time.Time{}, ErrBranchNotFound
	}

	lines := strings.Split(output, "\n")
	if len(lines) < 2 {
		return "", time.Time{}, fmt.Errorf("git log: unexpected output format: %q", output)
	}

	sha = lines[0]
	timestamp, err := strconv.ParseInt(lines[1], 10, 64)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("parse timestamp: %w", err)
	}

	return sha, time.Unix(timestamp, 0).UTC(), nil
}

// GetCommitsBehindAhead returns how many commits branch is behind/ahead of base.
// behind = commits in base but not in branch
// ahead = commits in branch but not in base
func GetCommitsBehindAhead(repoPath, branch, base string) (behind, ahead int, err error) {
	// git rev-list --left-right --count base...branch
	// Output: <behind>\t<ahead>
	cmd := exec.Command("git", "rev-list", "--left-right", "--count", base+"..."+branch)
	cmd.Dir = repoPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if strings.Contains(stderrStr, "not a git repository") {
			return 0, 0, ErrNotARepository
		}
		if strings.Contains(stderrStr, "unknown revision") || strings.Contains(stderrStr, "bad revision") {
			return 0, 0, ErrBranchNotFound
		}
		return 0, 0, fmt.Errorf("git rev-list: %w", err)
	}

	output := strings.TrimSpace(stdout.String())
	parts := strings.Fields(output)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("git rev-list: unexpected output format: %q", output)
	}

	behind, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("parse behind count: %w", err)
	}

	ahead, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("parse ahead count: %w", err)
	}

	return behind, ahead, nil
}

// HasMergeConflicts checks if merging branch into base would have conflicts.
// This does NOT perform a merge - it uses git merge-tree for a dry-run simulation.
func HasMergeConflicts(repoPath, branch, base string) (bool, error) {
	// Check that both branches exist
	if err := checkBranchExists(repoPath, branch); err != nil {
		return false, err
	}
	if err := checkBranchExists(repoPath, base); err != nil {
		return false, err
	}

	// First get the merge base
	mergeBaseCmd := exec.Command("git", "merge-base", base, branch)
	mergeBaseCmd.Dir = repoPath

	var mbStdout, mbStderr bytes.Buffer
	mergeBaseCmd.Stdout = &mbStdout
	mergeBaseCmd.Stderr = &mbStderr

	if err := mergeBaseCmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(mbStderr.String())
		if strings.Contains(stderrStr, "not a git repository") {
			return false, ErrNotARepository
		}
		if strings.Contains(stderrStr, "unknown revision") || strings.Contains(stderrStr, "bad revision") {
			return false, ErrBranchNotFound
		}
		// No merge base found means the branches have no common history
		// This would definitely cause merge issues, so treat as conflicts
		return true, nil
	}

	mergeBase := strings.TrimSpace(mbStdout.String())
	if mergeBase == "" {
		return true, nil
	}

	// Use git merge-tree to simulate the merge
	// git merge-tree <merge-base> <base> <branch>
	// If output contains conflict markers, there are conflicts
	cmd := exec.Command("git", "merge-tree", mergeBase, base, branch)
	cmd.Dir = repoPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if strings.Contains(stderrStr, "not a git repository") {
			return false, ErrNotARepository
		}
		return false, fmt.Errorf("git merge-tree: %w", err)
	}

	output := stdout.String()

	// Check for conflict markers in the output
	// merge-tree output contains lines starting with certain prefixes for conflicts
	// or we can look for conflict markers like <<<<<<<, =======, >>>>>>>
	hasConflicts := strings.Contains(output, "<<<<<<<") ||
		strings.Contains(output, "=======") ||
		strings.Contains(output, ">>>>>>>")

	return hasConflicts, nil
}

// ComputeBranchMetrics computes all metrics for a branch.
// The base branch defaults to "main" or "master" (whichever exists).
func ComputeBranchMetrics(repoPath, branch string) (BranchMetrics, error) {
	return ComputeBranchMetricsWithBase(repoPath, branch, "")
}

// ComputeBranchMetricsWithBase computes all metrics for a branch against a specified base.
// If base is empty, it defaults to "main" or "master" (whichever exists).
func ComputeBranchMetricsWithBase(repoPath, branch, base string) (BranchMetrics, error) {
	metrics := BranchMetrics{
		Branch: branch,
	}

	// Determine base branch if not specified
	if base == "" {
		var err error
		base, err = getDefaultBranch(repoPath)
		if err != nil {
			return metrics, fmt.Errorf("determine default branch: %w", err)
		}
	}

	// Get last commit info
	_, lastCommitAt, err := GetBranchLastCommit(repoPath, branch)
	if err != nil {
		return metrics, fmt.Errorf("get last commit: %w", err)
	}
	metrics.LastCommitAt = lastCommitAt
	metrics.LastCommitAgeDays = daysSince(lastCommitAt)

	// Get branch creation time
	createdAt, err := GetBranchCreationTime(repoPath, branch)
	if err != nil {
		return metrics, fmt.Errorf("get branch creation time: %w", err)
	}
	metrics.BranchAgeDays = daysSince(createdAt)

	// Get commits behind/ahead
	behind, ahead, err := GetCommitsBehindAhead(repoPath, branch, base)
	if err != nil {
		return metrics, fmt.Errorf("get commits behind/ahead: %w", err)
	}
	metrics.CommitsBehindMain = behind
	metrics.CommitsAheadOfMain = ahead

	// Check for merge conflicts
	hasConflicts, err := HasMergeConflicts(repoPath, branch, base)
	if err != nil {
		return metrics, fmt.Errorf("check merge conflicts: %w", err)
	}
	metrics.HasConflicts = hasConflicts

	return metrics, nil
}

// EvaluateBranchStatus computes metrics and evaluates against thresholds.
func EvaluateBranchStatus(repoPath, branch string, thresholds BranchThresholds) (BranchStatus, error) {
	return EvaluateBranchStatusWithBase(repoPath, branch, "", thresholds)
}

// EvaluateBranchStatusWithBase computes metrics and evaluates against thresholds.
// If base is empty, it defaults to "main" or "master" (whichever exists).
func EvaluateBranchStatusWithBase(repoPath, branch, base string, thresholds BranchThresholds) (BranchStatus, error) {
	status := BranchStatus{}

	metrics, err := ComputeBranchMetricsWithBase(repoPath, branch, base)
	if err != nil {
		return status, err
	}
	status.Metrics = metrics

	// Apply threshold evaluation rules from spec:
	// - No commits in X days → warning
	if metrics.LastCommitAgeDays >= thresholds.StaleAfterDays {
		status.Warnings = append(status.Warnings,
			fmt.Sprintf("branch is stale: no commits in %d days (threshold: %d days)",
				metrics.LastCommitAgeDays, thresholds.StaleAfterDays))
	}

	// - Behind main by 100+ commits → error
	// - Behind main by 50+ commits → warning
	// (error check first so we don't add both)
	if metrics.CommitsBehindMain >= thresholds.DriftErrorCommits {
		status.Errors = append(status.Errors,
			fmt.Sprintf("branch has critical drift: %d commits behind main (threshold: %d)",
				metrics.CommitsBehindMain, thresholds.DriftErrorCommits))
	} else if metrics.CommitsBehindMain >= thresholds.DriftWarningCommits {
		status.Warnings = append(status.Warnings,
			fmt.Sprintf("branch is drifting: %d commits behind main (threshold: %d)",
				metrics.CommitsBehindMain, thresholds.DriftWarningCommits))
	}

	// - Has merge conflicts → error
	if metrics.HasConflicts {
		status.Errors = append(status.Errors, "branch has merge conflicts with main")
	}

	return status, nil
}

// Helper functions

// checkBranchExists verifies that a branch exists in the repository.
func checkBranchExists(repoPath, branch string) error {
	cmd := exec.Command("git", "rev-parse", "--verify", branch)
	cmd.Dir = repoPath

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if strings.Contains(stderrStr, "not a git repository") {
			return ErrNotARepository
		}
		return ErrBranchNotFound
	}

	return nil
}

// getDefaultBranch returns the default branch name (main or master).
func getDefaultBranch(repoPath string) (string, error) {
	// Try "main" first
	if err := checkBranchExists(repoPath, "main"); err == nil {
		return "main", nil
	}

	// Try "master"
	if err := checkBranchExists(repoPath, "master"); err == nil {
		return "master", nil
	}

	return "", fmt.Errorf("no default branch found (tried main, master)")
}

// getFirstCommitTime returns the time of the first commit in the repository.
func getFirstCommitTime(repoPath, branch string) (time.Time, error) {
	// git log --reverse --format=%ct <branch> | head -1
	cmd := exec.Command("git", "log", "--reverse", "--format=%ct", branch)
	cmd.Dir = repoPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if strings.Contains(stderrStr, "not a git repository") {
			return time.Time{}, ErrNotARepository
		}
		return time.Time{}, fmt.Errorf("git log: %w", err)
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return time.Time{}, fmt.Errorf("branch has no commits")
	}

	lines := strings.Split(output, "\n")
	timestamp, err := strconv.ParseInt(lines[0], 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse timestamp: %w", err)
	}

	return time.Unix(timestamp, 0).UTC(), nil
}

// daysSince returns the number of days between t and now.
func daysSince(t time.Time) int {
	return int(time.Since(t).Hours() / 24)
}
