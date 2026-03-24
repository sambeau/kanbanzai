package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"kanbanzai/internal/core"
)

// LocalConfigFile is the name of the user-local configuration file.
const LocalConfigFile = "local.yaml"

// LocalConfig is the schema for .kbz/local.yaml.
type LocalConfig struct {
	User struct {
		Name string `yaml:"name"`
	} `yaml:"user"`
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
