package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sambeau/kanbanzai/internal/registry"
)

const docsUsageText = `Usage: kbz docs <subcommand> [options]

Subcommands:
  sync [--root <path>]   Update generated regions in CLAUDE.md,
                         .github/copilot-instructions.md, and README.md.
  check [--root <path>]  Report stale generated regions without writing.
                         Exits non-zero if any region is stale.

Options:
  --root <path>   Repository root. Defaults to current directory.
`

// docsTargetFiles lists the files (relative to root) managed by kbz docs.
// AGENTS.md is intentionally excluded (REQ-011).
var docsTargetFiles = []string{
	"CLAUDE.md",
	".github/copilot-instructions.md",
	"README.md",
}

func runDocs(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("missing docs subcommand\n\n%s", docsUsageText)
	}
	switch args[0] {
	case "sync":
		return runDocsSync(args[1:], deps)
	case "check":
		return runDocsCheck(args[1:], deps)
	default:
		return fmt.Errorf("unknown docs subcommand %q\n\n%s", args[0], docsUsageText)
	}
}

// parseDocsRoot parses the --root flag from args. Returns the root directory
// or an error for unrecognised flags or a missing flag value.
func parseDocsRoot(args []string) (string, error) {
	root := "."
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--root":
			if i+1 >= len(args) {
				return "", fmt.Errorf("--root requires a value")
			}
			i++
			root = args[i]
		default:
			return "", fmt.Errorf("unknown flag %q\n\n%s", args[i], docsUsageText)
		}
	}
	return root, nil
}

// buildRenderers extracts the registry model from root and returns a map of
// region name to rendered candidate content for all known regions.
func buildRenderers(root string) (map[string]string, error) {
	model, err := registry.Extract(root)
	if err != nil {
		return nil, fmt.Errorf("registry extract: %w", err)
	}
	return map[string]string{
		"roles-and-skills": registry.RolesAndSkillsContent(model),
		"role-index":       registry.RoleIndexContent(model),
	}, nil
}

func runDocsSync(args []string, deps dependencies) error {
	root, err := parseDocsRoot(args)
	if err != nil {
		return err
	}

	// REQ-011: AGENTS.md is intentionally excluded from generated output.
	fmt.Fprintln(deps.stdout, "AGENTS.md: skipped (hand-authored narrative, excluded by design)")

	renderers, err := buildRenderers(root)
	if err != nil {
		return err
	}

	for _, relPath := range docsTargetFiles {
		absPath := filepath.Join(root, filepath.FromSlash(relPath))
		data, err := os.ReadFile(absPath)
		if err != nil {
			return fmt.Errorf("%s: %w", relPath, err)
		}
		content := string(data)

		// Parse regions once to detect structural errors and discover present regions.
		regions, err := registry.ParseRegions(relPath, content)
		if err != nil {
			return err
		}
		present := make(map[string]bool, len(regions))
		for _, r := range regions {
			present[r.Name] = true
		}

		updatedCount := 0
		for regionName, candidate := range renderers {
			if !present[regionName] {
				continue
			}
			updated, err := registry.SyncRegion(relPath, content, regionName, candidate)
			if err != nil {
				return err
			}
			content = updated
			updatedCount++
		}

		if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("%s: write: %w", relPath, err)
		}
		fmt.Fprintf(deps.stdout, "%s: updated %d region(s)\n", relPath, updatedCount)
	}

	return nil
}

func runDocsCheck(args []string, deps dependencies) error {
	root, err := parseDocsRoot(args)
	if err != nil {
		return err
	}

	// REQ-011: AGENTS.md is intentionally excluded.
	fmt.Fprintln(deps.stdout, "AGENTS.md: skipped (hand-authored narrative, excluded by design)")

	renderers, err := buildRenderers(root)
	if err != nil {
		return err
	}

	var staleFiles []string
	for _, relPath := range docsTargetFiles {
		absPath := filepath.Join(root, filepath.FromSlash(relPath))
		data, err := os.ReadFile(absPath)
		if err != nil {
			return fmt.Errorf("%s: %w", relPath, err)
		}
		content := string(data)

		// Parse regions once to detect structural errors and discover present regions.
		regions, err := registry.ParseRegions(relPath, content)
		if err != nil {
			return err
		}
		present := make(map[string]bool, len(regions))
		for _, r := range regions {
			present[r.Name] = true
		}

		fileStale := false
		for regionName, candidate := range renderers {
			if !present[regionName] {
				continue
			}
			stale, _, _, err := registry.CheckRegion(relPath, content, regionName, candidate)
			if err != nil {
				return err
			}
			if stale {
				fmt.Fprintf(deps.stdout, "%s: stale region %q\n", relPath, regionName)
				fileStale = true
			}
		}

		if fileStale {
			staleFiles = append(staleFiles, relPath)
		} else {
			fmt.Fprintf(deps.stdout, "%s: OK\n", relPath)
		}
	}

	if len(staleFiles) > 0 {
		return fmt.Errorf("stale generated regions in: %s", strings.Join(staleFiles, ", "))
	}
	return nil
}
