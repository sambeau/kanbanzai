package kbzinit

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// FindGitRoot walks up from dir until it finds a .git directory.
// Returns the directory containing .git, or an error if not found.
func FindGitRoot(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}

	current := abs
	for {
		if _, err := os.Stat(filepath.Join(current, ".git")); err == nil {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			// reached filesystem root
			return "", fmt.Errorf("not a git repository")
		}
		current = parent
	}
}

// HasCommits returns true if the git repository at gitRoot has at least one commit.
func HasCommits(gitRoot string) (bool, error) {
	cmd := exec.Command("git", "-C", gitRoot, "rev-parse", "HEAD")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		// rev-parse HEAD fails when there are no commits
		return false, nil
	}
	return true, nil
}
