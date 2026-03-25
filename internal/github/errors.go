// Package github provides GitHub API operations for PR management.
package github

import "errors"

// Common error types for GitHub operations.
var (
	// ErrNoToken is returned when no GitHub token is configured.
	ErrNoToken = errors.New("GitHub token not configured")

	// ErrRepoNotFound is returned when the repository is not found or not accessible.
	ErrRepoNotFound = errors.New("repository not found or not accessible")

	// ErrPRNotFound is returned when the pull request is not found.
	ErrPRNotFound = errors.New("pull request not found")

	// ErrRateLimited is returned when the GitHub API rate limit is exceeded.
	ErrRateLimited = errors.New("GitHub API rate limited")

	// ErrUnauthorized is returned when the GitHub token is invalid or lacks permissions.
	ErrUnauthorized = errors.New("GitHub token invalid or lacks permissions")

	// ErrBranchNotFound is returned when a branch is not found.
	ErrBranchNotFound = errors.New("branch not found")

	// ErrRemoteNotFound is returned when no git remote is configured.
	ErrRemoteNotFound = errors.New("git remote not found")

	// ErrNotGitHubRemote is returned when the remote URL is not a GitHub URL.
	ErrNotGitHubRemote = errors.New("remote is not a GitHub repository")
)
