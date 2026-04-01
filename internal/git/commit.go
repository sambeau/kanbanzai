// Package git commit.go — git commit helpers for Kanbanzai state persistence.
//
// CommitStateIfDirty is called by the handoff tool before assembling a
// sub-agent prompt. It protects workflow state written by orchestrator MCP
// tool calls from being destroyed by sub-agent git operations (stash,
// checkout, reset). See spec work/spec/sub-agent-state-isolation.md.
package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// stateCommitMessage is the fixed commit message used for all pre-dispatch
// state commits (REQ-04 of sub-agent-state-isolation spec).
const stateCommitMessage = "chore(kbz): persist workflow state before sub-agent dispatch"

// stateDir is the directory whose changes are staged and committed.
const stateDir = ".kbz/state/"

// CommitStateIfDirty stages all files under .kbz/state/ that have uncommitted
// changes and creates a commit. Returns (true, nil) if a commit was created,
// (false, nil) if the working tree was clean (nothing to commit), and
// (false, err) if staging or committing failed.
//
// The commit includes only files under .kbz/state/. Files outside that
// directory are not staged or committed (REQ-03).
//
// No empty commits are created (REQ-05).
//
// repoRoot is the absolute path to the repository root (the directory
// containing .git/). The caller is responsible for determining this path.
func CommitStateIfDirty(repoRoot string) (committed bool, err error) {
	// Check for uncommitted changes under .kbz/state/.
	// git status --porcelain outputs one line per changed file; empty means clean.
	statusOut, statusErr := runGitCmd(repoRoot, "status", "--porcelain", "--", stateDir)
	if statusErr != nil {
		return false, fmt.Errorf("git status: %w", statusErr)
	}
	if strings.TrimSpace(statusOut) == "" {
		// Nothing to commit — working tree is clean for .kbz/state/.
		return false, nil
	}

	// Stage only files under .kbz/state/.
	// Using "git add -- .kbz/state/" avoids accidentally staging anything else.
	if _, addErr := runGitCmd(repoRoot, "add", "--", stateDir); addErr != nil {
		return false, fmt.Errorf("git add: %w", addErr)
	}

	// Create the commit. The message is fixed per spec REQ-04.
	if _, commitErr := runGitCmd(repoRoot, "commit", "-m", stateCommitMessage); commitErr != nil {
		return false, fmt.Errorf("git commit: %w", commitErr)
	}

	return true, nil
}

// runGitCmd runs a git command in repoRoot, returning stdout on success or
// an error that includes stderr output on failure.
func runGitCmd(repoRoot string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if runErr := cmd.Run(); runErr != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if stderrStr != "" {
			return "", fmt.Errorf("%s", stderrStr)
		}
		return "", runErr
	}

	return stdout.String(), nil
}
