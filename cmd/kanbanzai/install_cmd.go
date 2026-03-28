package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"kanbanzai/internal/config"
	"kanbanzai/internal/install"
)

func runInstallRecord(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing install-record subcommand\n\n%s", installRecordUsageText)
	}

	switch args[0] {
	case "write":
		return runInstallRecordWrite(args[1:], deps)
	default:
		return fmt.Errorf("unknown install-record subcommand %q\n\n%s", args[0], installRecordUsageText)
	}
}

func runInstallRecordWrite(args []string, deps dependencies) error {
	flags, err := parseFlags(args)
	if err != nil {
		return err
	}

	by := flags["by"]
	if by == "" {
		by = "manual"
	}

	// Verify we're in a kanbanzai-initialised directory.
	_, err = config.Load()
	if err != nil {
		return fmt.Errorf("not a kanbanzai-initialised directory (no .kbz/config.yaml): %w", err)
	}

	// Get the current git SHA.
	gitCmd := exec.Command("git", "rev-parse", "HEAD")
	shaBytes, err := gitCmd.Output()
	if err != nil {
		return fmt.Errorf("get git SHA: %w", err)
	}
	gitSHA := strings.TrimSpace(string(shaBytes))

	// Get the binary path, resolving symlinks.
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}
	binaryPath, err := filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("resolve symlinks: %w", err)
	}

	// Write the install record. Root is "." since we verified .kbz/config.yaml exists.
	if err := install.WriteRecord(".", gitSHA, binaryPath, by); err != nil {
		return fmt.Errorf("write install record: %w", err)
	}

	fmt.Fprintf(deps.stdout, "Install record written\n")
	fmt.Fprintf(deps.stdout, "  git_sha: %s\n", gitSHA[:7])
	fmt.Fprintf(deps.stdout, "  binary:  %s\n", binaryPath)
	fmt.Fprintf(deps.stdout, "  by:      %s\n", by)

	return nil
}

const installRecordUsageText = `kanbanzai install-record <subcommand> [flags]

Manage binary install records.

Subcommands:
  write   Write an install record for the current binary

Flags (write):
  --by <source>   Who/what triggered the install (default: "manual")

Examples:
  kbz install-record write
  kbz install-record write --by makefile
  kbz install-record write --by "post-merge"
`
