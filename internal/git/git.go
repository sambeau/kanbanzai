// Package git provides Git operations for anchoring and staleness detection.
package git

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ErrNotARepository is returned when the path is not a Git repository.
var ErrNotARepository = errors.New("not a git repository")

// ErrFileNotFound is returned when the file does not exist or has no commits.
var ErrFileNotFound = errors.New("file not found or has no commits")

// GetFileLastModified returns the commit that last modified a file and its timestamp.
// The repoPath should be the root directory of the Git repository.
// The filePath should be relative to the repository root.
func GetFileLastModified(repoPath, filePath string) (commit string, modifiedAt time.Time, err error) {
	// git log -1 --format=%H%n%ct -- <file>
	// %H = full commit hash
	// %ct = committer date, UNIX timestamp
	cmd := exec.Command("git", "log", "-1", "--format=%H%n%ct", "--", filePath)
	cmd.Dir = repoPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if strings.Contains(stderrStr, "not a git repository") {
			return "", time.Time{}, ErrNotARepository
		}
		if stderrStr != "" {
			return "", time.Time{}, fmt.Errorf("git log: %s", stderrStr)
		}
		return "", time.Time{}, fmt.Errorf("git log: %w", err)
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return "", time.Time{}, ErrFileNotFound
	}

	lines := strings.Split(output, "\n")
	if len(lines) < 2 {
		return "", time.Time{}, fmt.Errorf("git log: unexpected output format: %q", output)
	}

	commit = lines[0]
	timestamp, err := strconv.ParseInt(lines[1], 10, 64)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("git log: parse timestamp: %w", err)
	}

	modifiedAt = time.Unix(timestamp, 0).UTC()
	return commit, modifiedAt, nil
}

// GetCommitTimestamp returns the timestamp of a commit.
// The commitSHA can be a full or partial SHA, branch name, or tag.
func GetCommitTimestamp(repoPath, commitSHA string) (time.Time, error) {
	// git show -s --format=%ct <commit>
	// %ct = committer date, UNIX timestamp
	cmd := exec.Command("git", "show", "-s", "--format=%ct", commitSHA)
	cmd.Dir = repoPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if strings.Contains(stderrStr, "not a git repository") {
			return time.Time{}, ErrNotARepository
		}
		if strings.Contains(stderrStr, "unknown revision") || strings.Contains(stderrStr, "bad revision") {
			return time.Time{}, fmt.Errorf("commit not found: %s", commitSHA)
		}
		if stderrStr != "" {
			return time.Time{}, fmt.Errorf("git show: %s", stderrStr)
		}
		return time.Time{}, fmt.Errorf("git show: %w", err)
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return time.Time{}, fmt.Errorf("commit not found: %s", commitSHA)
	}

	timestamp, err := strconv.ParseInt(output, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("git show: parse timestamp: %w", err)
	}

	return time.Unix(timestamp, 0).UTC(), nil
}

// IsFileModifiedSince checks if a file was modified after a given timestamp.
// Returns true if the file's last modification is after the since timestamp.
func IsFileModifiedSince(repoPath, filePath string, since time.Time) (bool, error) {
	_, modifiedAt, err := GetFileLastModified(repoPath, filePath)
	if err != nil {
		return false, err
	}

	return modifiedAt.After(since), nil
}

// GitMove moves src to dst using git mv, preserving Git history.
// Both src and dst must be paths relative to repoRoot.
func GitMove(repoRoot, src, dst string) error {
	_, err := runGitCmd(repoRoot, "mv", src, dst)
	return err
}
