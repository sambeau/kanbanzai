package worktree

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Git provides operations on Git worktrees and branches.
type Git struct {
	// RepoDir is the path to the main repository.
	// If empty, commands run in the current directory.
	RepoDir string
}

// NewGit creates a new Git instance for the given repository directory.
func NewGit(repoDir string) *Git {
	return &Git{RepoDir: repoDir}
}

// CreateWorktree creates a new worktree at the given path for the specified branch.
// The branch must already exist.
func (g *Git) CreateWorktree(path, branch string) error {
	args := []string{"worktree", "add", path, branch}
	if err := g.run(args...); err != nil {
		return fmt.Errorf("git worktree add: %w", err)
	}
	return nil
}

// CreateWorktreeNewBranch creates a new worktree at the given path with a new branch.
// The new branch is created from the specified base branch (or HEAD if base is empty).
func (g *Git) CreateWorktreeNewBranch(path, branch, base string) error {
	args := []string{"worktree", "add", "-b", branch, path}
	if base != "" {
		args = append(args, base)
	}
	if err := g.run(args...); err != nil {
		return fmt.Errorf("git worktree add -b: %w", err)
	}
	return nil
}

// RemoveWorktree removes the worktree at the given path.
// If force is true, the worktree is removed even if it has uncommitted changes.
func (g *Git) RemoveWorktree(path string, force bool) error {
	args := []string{"worktree", "remove", path}
	if force {
		args = append(args, "--force")
	}
	if err := g.run(args...); err != nil {
		return fmt.Errorf("git worktree remove: %w", err)
	}
	return nil
}

// ListWorktrees returns the paths of all worktrees in the repository.
func (g *Git) ListWorktrees() ([]string, error) {
	output, err := g.output("worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("git worktree list: %w", err)
	}

	var paths []string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "worktree ") {
			path := strings.TrimPrefix(line, "worktree ")
			paths = append(paths, path)
		}
	}

	return paths, nil
}

// BranchExists returns true if the branch exists in the repository.
func (g *Git) BranchExists(branch string) bool {
	err := g.run("rev-parse", "--verify", "--quiet", "refs/heads/"+branch)
	return err == nil
}

// CreateBranch creates a new branch from the specified base.
// If from is empty, the branch is created from HEAD.
func (g *Git) CreateBranch(branch, from string) error {
	args := []string{"branch", branch}
	if from != "" {
		args = append(args, from)
	}
	if err := g.run(args...); err != nil {
		return fmt.Errorf("git branch: %w", err)
	}
	return nil
}

// DeleteBranch deletes the specified branch.
// If force is true, the branch is deleted even if not fully merged.
func (g *Git) DeleteBranch(branch string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	if err := g.run("branch", flag, branch); err != nil {
		return fmt.Errorf("git branch %s: %w", flag, err)
	}
	return nil
}

// CurrentBranch returns the name of the currently checked out branch.
func (g *Git) CurrentBranch() (string, error) {
	output, err := g.output("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("git rev-parse: %w", err)
	}
	return strings.TrimSpace(output), nil
}

// PruneWorktrees removes stale worktree entries (worktrees that no longer exist on disk).
func (g *Git) PruneWorktrees() error {
	if err := g.run("worktree", "prune"); err != nil {
		return fmt.Errorf("git worktree prune: %w", err)
	}
	return nil
}

// run executes a git command and returns any error.
func (g *Git) run(args ...string) error {
	cmd := exec.Command("git", args...)
	if g.RepoDir != "" {
		cmd.Dir = g.RepoDir
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return fmt.Errorf("%s: %s", err, errMsg)
		}
		return err
	}
	return nil
}

// output executes a git command and returns its stdout.
func (g *Git) output(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if g.RepoDir != "" {
		cmd.Dir = g.RepoDir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return "", fmt.Errorf("%s: %s", err, errMsg)
		}
		return "", err
	}
	return stdout.String(), nil
}
