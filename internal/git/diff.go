package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// GetFilesChangedOnBranch returns the list of files changed on a branch since it diverged
// from the default branch (main or master). Returns ErrBranchNotFound if the branch
// does not exist. Returns an empty list (not an error) if the branch has no unique commits.
func GetFilesChangedOnBranch(repoPath, branch string) ([]string, error) {
	if err := checkBranchExists(repoPath, branch); err != nil {
		return nil, err
	}

	base, err := GetDefaultBranch(repoPath)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command("git", "diff", "--name-only", base+".."+branch)
	cmd.Dir = repoPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if strings.Contains(stderrStr, "not a git repository") {
			return nil, ErrNotARepository
		}
		return nil, fmt.Errorf("git diff: %w", err)
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return []string{}, nil
	}

	lines := strings.Split(output, "\n")
	files := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}

	return files, nil
}
