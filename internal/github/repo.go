package github

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"kanbanzai/internal/config"
)

// RepoInfo contains the owner and repository name.
type RepoInfo struct {
	Owner string
	Repo  string
}

// String returns the owner/repo format.
func (r RepoInfo) String() string {
	return r.Owner + "/" + r.Repo
}

// IsEmpty returns true if both Owner and Repo are empty.
func (r RepoInfo) IsEmpty() bool {
	return r.Owner == "" && r.Repo == ""
}

// Regular expressions for parsing GitHub remote URLs.
var (
	// HTTPS format: https://github.com/owner/repo.git or https://github.com/owner/repo
	httpsRE = regexp.MustCompile(`^https://github\.com/([^/]+)/([^/]+?)(?:\.git)?$`)

	// SSH format: git@github.com:owner/repo.git or git@github.com:owner/repo
	sshRE = regexp.MustCompile(`^git@github\.com:([^/]+)/([^/]+?)(?:\.git)?$`)

	// SSH with ssh:// prefix: ssh://git@github.com/owner/repo.git
	sshPrefixRE = regexp.MustCompile(`^ssh://git@github\.com/([^/]+)/([^/]+?)(?:\.git)?$`)
)

// DetectRepo attempts to detect owner/repo from the Git remote.
// It first tries to parse the remote URL, then falls back to config if provided.
func DetectRepo(repoPath string, cfg *config.LocalConfig) (RepoInfo, error) {
	// Try to detect from git remote
	info, err := detectFromRemote(repoPath)
	if err == nil && !info.IsEmpty() {
		return info, nil
	}

	// Fall back to config if available
	if cfg != nil {
		owner := cfg.GetGitHubOwner()
		repo := cfg.GetGitHubRepo()
		if owner != "" && repo != "" {
			return RepoInfo{Owner: owner, Repo: repo}, nil
		}
	}

	// If we had a remote error, return it
	if err != nil {
		return RepoInfo{}, err
	}

	return RepoInfo{}, fmt.Errorf("could not detect repository: no git remote and no config override")
}

// detectFromRemote gets the origin remote URL and parses it.
func detectFromRemote(repoPath string) (RepoInfo, error) {
	url, err := getRemoteURL(repoPath, "origin")
	if err != nil {
		return RepoInfo{}, err
	}

	return ParseRemoteURL(url)
}

// getRemoteURL gets the URL for a named remote.
func getRemoteURL(repoPath, remoteName string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", remoteName)
	cmd.Dir = repoPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if strings.Contains(stderrStr, "No such remote") {
			return "", ErrRemoteNotFound
		}
		if strings.Contains(stderrStr, "not a git repository") {
			return "", fmt.Errorf("not a git repository: %s", repoPath)
		}
		return "", fmt.Errorf("git remote get-url: %s", stderrStr)
	}

	return strings.TrimSpace(stdout.String()), nil
}

// ParseRemoteURL parses a Git remote URL and extracts owner and repo.
// Supports HTTPS and SSH formats for GitHub.
func ParseRemoteURL(url string) (RepoInfo, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return RepoInfo{}, ErrRemoteNotFound
	}

	// Try HTTPS format
	if matches := httpsRE.FindStringSubmatch(url); matches != nil {
		return RepoInfo{Owner: matches[1], Repo: matches[2]}, nil
	}

	// Try SSH format (git@github.com:owner/repo)
	if matches := sshRE.FindStringSubmatch(url); matches != nil {
		return RepoInfo{Owner: matches[1], Repo: matches[2]}, nil
	}

	// Try SSH with prefix (ssh://git@github.com/owner/repo)
	if matches := sshPrefixRE.FindStringSubmatch(url); matches != nil {
		return RepoInfo{Owner: matches[1], Repo: matches[2]}, nil
	}

	// Check if it's a GitHub URL at all
	if strings.Contains(url, "github.com") {
		return RepoInfo{}, fmt.Errorf("unrecognized GitHub URL format: %s", url)
	}

	return RepoInfo{}, ErrNotGitHubRemote
}
