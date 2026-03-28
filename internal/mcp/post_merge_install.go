package mcp

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sambeau/kanbanzai/internal/config"
	"github.com/sambeau/kanbanzai/internal/install"
)

// postMergeInstall attempts to rebuild the kanbanzai binary and write an install
// record after a successful merge. It is a best-effort operation: merge success
// is never affected by build failures.
//
// Returns a SideEffect describing the outcome (install_complete or install_failed),
// or nil if the step was skipped entirely (opt-out or no cmd/kanbanzai/main.go).
func postMergeInstall(ctx context.Context, repoPath string, cfg *config.Config) *SideEffect {
	// Check opt-out: merge.post_merge_install == false.
	if cfg.Merge.PostMergeInstall != nil && !*cfg.Merge.PostMergeInstall {
		return nil
	}

	// Check if cmd/kanbanzai/main.go exists at the repo root.
	mainPath := filepath.Join(repoPath, "cmd", "kanbanzai", "main.go")
	if _, err := os.Stat(mainPath); err != nil {
		// File doesn't exist — skip silently.
		return nil
	}

	// Get the current git SHA from HEAD (we just merged, so HEAD is the merge commit).
	gitSHA, err := gitRevParseHead(repoPath)
	if err != nil {
		return installFailedEffect(fmt.Sprintf("get git SHA: %v", err))
	}

	// Get build time.
	buildTime := time.Now().UTC().Format("2006-01-02T15:04:05Z")

	// Check dirty status.
	dirty := "false"
	dirtyCmd := exec.Command("git", "diff", "--quiet")
	dirtyCmd.Dir = repoPath
	if err := dirtyCmd.Run(); err != nil {
		dirty = "true"
	}

	// Build ldflags matching the Makefile pattern.
	pkg := "github.com/sambeau/kanbanzai/internal/buildinfo"
	ldflags := fmt.Sprintf(
		"-X '%s.Version=dev' -X '%s.GitSHA=%s' -X '%s.BuildTime=%s' -X '%s.Dirty=%s'",
		pkg, pkg, gitSHA, pkg, buildTime, pkg, dirty,
	)

	// Run go install with ldflags.
	installCmd := exec.Command("go", "install", "-ldflags", ldflags, "./cmd/kanbanzai")
	installCmd.Dir = repoPath
	installOutput, err := installCmd.CombinedOutput()
	if err != nil {
		msg := fmt.Sprintf("go install failed: %v", err)
		if len(installOutput) > 0 {
			msg += ": " + strings.TrimSpace(string(installOutput))
		}
		return installFailedEffect(msg)
	}

	// Determine the installed binary path via GOBIN or GOPATH.
	binaryPath, err := resolveInstalledBinaryPath()
	if err != nil {
		return installFailedEffect(fmt.Sprintf("resolve binary path: %v", err))
	}

	// Write the install record.
	if err := install.WriteRecord(repoPath, gitSHA, binaryPath, "post-merge"); err != nil {
		return installFailedEffect(fmt.Sprintf("write install record: %v", err))
	}

	gitSHAShort := deriveGitSHAShort(gitSHA)

	return &SideEffect{
		Type:    "install_complete",
		Trigger: "post-merge binary rebuild",
		Extra: map[string]string{
			"git_sha":     gitSHAShort,
			"binary_path": binaryPath,
			"message":     fmt.Sprintf("Binary rebuilt at %s from %s. Please restart the MCP server to use the updated binary.", binaryPath, gitSHAShort),
		},
	}
}

// gitRevParseHead runs git rev-parse HEAD in the given directory.
func gitRevParseHead(repoPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// resolveInstalledBinaryPath determines where `go install` placed the binary.
func resolveInstalledBinaryPath() (string, error) {
	// Check GOBIN first.
	gobin := os.Getenv("GOBIN")
	if gobin != "" {
		return filepath.Join(gobin, "kanbanzai"), nil
	}

	// Fall back to GOPATH/bin.
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		// Default GOPATH is ~/go.
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("determine home directory: %w", err)
		}
		gopath = filepath.Join(home, "go")
	}

	return filepath.Join(gopath, "bin", "kanbanzai"), nil
}

// installFailedEffect returns a side effect for a failed install attempt.
func installFailedEffect(message string) *SideEffect {
	return &SideEffect{
		Type:    "install_failed",
		Trigger: "post-merge binary rebuild",
		Extra: map[string]string{
			"message": message,
		},
	}
}
