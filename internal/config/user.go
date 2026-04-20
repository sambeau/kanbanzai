package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/sambeau/kanbanzai/internal/core"
)

// LocalConfigFile is the name of the user-local configuration file.
const LocalConfigFile = "local.yaml"

// GitHubConfig holds GitHub-related settings for the local environment.
type GitHubConfig struct {
	// Token is the GitHub personal access token or fine-grained token.
	Token string `yaml:"token,omitempty"`
	// Owner is an optional override for the repository owner (org or user).
	Owner string `yaml:"owner,omitempty"`
	// Repo is an optional override for the repository name.
	Repo string `yaml:"repo,omitempty"`
}

// CodebaseMemoryConfig holds codebase-memory-mcp settings for the local environment.
type CodebaseMemoryConfig struct {
	// GraphProject is the default project name for graph-based code navigation tools.
	// Used as the graph_project value when creating worktrees, so it need not be
	// provided explicitly on every worktree(action: "create") call.
	// Set once per machine to the project name shown in codebase-memory-mcp
	// (e.g. "Users-alice-Dev-myrepo").
	GraphProject string `yaml:"graph_project,omitempty"`
}

// LocalConfig is the schema for .kbz/local.yaml.
type LocalConfig struct {
	User struct {
		Name string `yaml:"name"`
	} `yaml:"user"`
	// GitHub holds GitHub-related settings.
	GitHub GitHubConfig `yaml:"github,omitempty"`
	// ToolHints maps role IDs to opaque tool guidance strings. Local hints
	// override project-level hints on a per-key basis.
	ToolHints map[string]string `yaml:"tool_hints,omitempty"`
	// CodebaseMemory holds codebase-memory-mcp settings for this machine.
	CodebaseMemory CodebaseMemoryConfig `yaml:"codebase_memory,omitempty"`
}

// GetGitHubToken returns the configured GitHub token, or empty string if not set.
func (lc *LocalConfig) GetGitHubToken() string {
	return lc.GitHub.Token
}

// GetGitHubOwner returns the configured GitHub owner override, or empty string if not set.
func (lc *LocalConfig) GetGitHubOwner() string {
	return lc.GitHub.Owner
}

// GetGitHubRepo returns the configured GitHub repo override, or empty string if not set.
func (lc *LocalConfig) GetGitHubRepo() string {
	return lc.GitHub.Repo
}

// LoadLocalConfig loads the local configuration from the default location.
func LoadLocalConfig() (*LocalConfig, error) {
	return LoadLocalConfigFrom(filepath.Join(core.RootPath(), LocalConfigFile))
}

// LoadLocalConfigFrom loads the local configuration from the specified path.
func LoadLocalConfigFrom(path string) (*LocalConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read local config: %w", err)
	}

	var lc LocalConfig
	if err := yaml.Unmarshal(data, &lc); err != nil {
		return nil, fmt.Errorf("parse local config: %w", err)
	}

	return &lc, nil
}

// ResolveIdentity returns the user identity to use for created_by/approved_by fields.
// Resolution order: explicit → .kbz/local.yaml → git config user.name → error.
func ResolveIdentity(explicit string) (string, error) {
	return resolveIdentity(explicit, filepath.Join(core.RootPath(), LocalConfigFile))
}

// resolveIdentity is the testable core of ResolveIdentity.
// It accepts the local config path so tests can point at a temp directory.
func resolveIdentity(explicit, localConfigPath string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return strings.TrimSpace(explicit), nil
	}

	// Try .kbz/local.yaml
	if data, err := os.ReadFile(localConfigPath); err == nil {
		var lc LocalConfig
		if err := yaml.Unmarshal(data, &lc); err == nil && strings.TrimSpace(lc.User.Name) != "" {
			return strings.TrimSpace(lc.User.Name), nil
		}
	}

	// Try git config user.name
	out, err := exec.Command("git", "config", "user.name").Output()
	if err == nil {
		name := strings.TrimSpace(string(out))
		if name != "" {
			return name, nil
		}
	}

	return "", fmt.Errorf("cannot resolve user identity: provide created_by explicitly, or set user.name in .kbz/local.yaml, or configure git user.name")
}

// ResolveGraphProject returns the default codebase-memory-mcp project name from local config.
// Returns empty string if local.yaml is absent or has no codebase_memory.graph_project set.
// Resolution is best-effort: errors are silently ignored so worktree creation never fails
// solely because the local config is missing.
func ResolveGraphProject() string {
	return resolveGraphProject(filepath.Join(core.RootPath(), LocalConfigFile))
}

// resolveGraphProject is the testable core of ResolveGraphProject.
func resolveGraphProject(localConfigPath string) string {
	data, err := os.ReadFile(localConfigPath)
	if err != nil {
		return ""
	}
	var lc LocalConfig
	if err := yaml.Unmarshal(data, &lc); err != nil {
		return ""
	}
	return strings.TrimSpace(lc.CodebaseMemory.GraphProject)
}
