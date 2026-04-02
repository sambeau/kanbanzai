// Package git commit.go — git commit helpers for Kanbanzai state persistence.
//
// CommitStateIfDirty is called by the handoff tool before assembling a
// sub-agent prompt. It protects workflow state written by orchestrator MCP
// tool calls from being destroyed by sub-agent git operations (stash,
// checkout, reset). See spec work/spec/sub-agent-state-isolation.md.
//
// CommitStateWithMessage and CommitStateAndPaths are the parameterised
// variants used by auto-commit call sites throughout the MCP tool handlers
// (FEAT-01KN73BFK4M4Z, Pillar A).
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
// changes and creates a commit with the fixed pre-dispatch message.
// Returns (true, nil) if a commit was created, (false, nil) if the working
// tree was clean (nothing to commit), and (false, err) if staging or
// committing failed.
//
// This function delegates to CommitStateWithMessage with the fixed
// stateCommitMessage constant, preserving backward compatibility for the
// handoff call site (FR-A05).
//
// repoRoot is the absolute path to the repository root (the directory
// containing .git/). The caller is responsible for determining this path.
func CommitStateIfDirty(repoRoot string) (committed bool, err error) {
	return CommitStateWithMessage(repoRoot, stateCommitMessage)
}

// CommitStateWithMessage stages all files under .kbz/state/ that have
// uncommitted changes and creates a commit with the caller-supplied message.
// Returns (true, nil) if a commit was created, (false, nil) if the working
// tree was clean (nothing to commit), and (false, err) if staging or
// committing failed.
//
// The commit includes only files under .kbz/state/. Files outside that
// directory are not staged or committed (FR-A14).
//
// No empty commits are created (FR-A04).
//
// repoRoot is the absolute path to the repository root (the directory
// containing .git/). The caller is responsible for determining this path.
func CommitStateWithMessage(repoRoot, message string) (bool, error) {
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
	if _, addErr := runGitCmd(repoRoot, "add", "--", stateDir); addErr != nil {
		return false, fmt.Errorf("git add: %w", addErr)
	}

	// Create the commit with the caller-supplied message.
	if _, commitErr := runGitCmd(repoRoot, "commit", "-m", message); commitErr != nil {
		return false, fmt.Errorf("git commit: %w", commitErr)
	}

	return true, nil
}

// CommitStateAndPaths stages all files under .kbz/state/ plus each path in
// extraPaths, and creates a single git commit with the caller-supplied
// message. Returns (true, nil) if a commit was created, (false, nil) if
// there are no dirty files to commit, and (false, err) on failure.
//
// Only paths explicitly listed in extraPaths are staged in addition to
// .kbz/state/. Globs, directory scans, and pattern matching are not used
// for extra paths (FR-A03).
//
// No empty commits are created (FR-A04).
//
// repoRoot is the absolute path to the repository root (the directory
// containing .git/). The caller is responsible for determining this path.
func CommitStateAndPaths(repoRoot, message string, extraPaths ...string) (bool, error) {
	// Check for uncommitted changes under .kbz/state/ and each extra path.
	// Build the status args: "status --porcelain -- .kbz/state/ <extraPaths...>"
	statusArgs := make([]string, 0, 3+len(extraPaths))
	statusArgs = append(statusArgs, "status", "--porcelain", "--", stateDir)
	statusArgs = append(statusArgs, extraPaths...)

	statusOut, statusErr := runGitCmd(repoRoot, statusArgs...)
	if statusErr != nil {
		return false, fmt.Errorf("git status: %w", statusErr)
	}
	if strings.TrimSpace(statusOut) == "" {
		// Nothing to commit — all paths are clean.
		return false, nil
	}

	// Stage .kbz/state/ only if it has dirty files. This avoids a git error
	// when the state dir is clean and only extraPaths are dirty.
	stateStatusOut, _ := runGitCmd(repoRoot, "status", "--porcelain", "--", stateDir)
	if strings.TrimSpace(stateStatusOut) != "" {
		if _, addErr := runGitCmd(repoRoot, "add", "--", stateDir); addErr != nil {
			return false, fmt.Errorf("git add state: %w", addErr)
		}
	}

	// Stage each extra path individually (one git add per path, per FR-A03).
	for _, p := range extraPaths {
		if _, addErr := runGitCmd(repoRoot, "add", "--", p); addErr != nil {
			return false, fmt.Errorf("git add %s: %w", p, addErr)
		}
	}

	// Create the commit with the caller-supplied message.
	if _, commitErr := runGitCmd(repoRoot, "commit", "-m", message); commitErr != nil {
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
