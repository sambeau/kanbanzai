package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sambeau/kanbanzai/internal/core"
)

// runMigrate implements the `kbz migrate` command group.
func runMigrate(args []string, deps dependencies) error {
	if len(args) == 0 || wantsHelp(args) {
		fmt.Fprint(deps.stdout, migrateUsageText)
		return nil
	}

	switch args[0] {
	case "stage-bindings":
		return runMigrateStageBindings(args[1:], deps)
	default:
		return fmt.Errorf("unknown migrate subcommand %q\n\n%s", args[0], migrateUsageText)
	}
}

// runMigrateStageBindings implements `kbz migrate stage-bindings`.
// It adds the `schema_version: 2` key to .kbz/stage-bindings.yaml
// idempotently without altering other content.
func runMigrateStageBindings(args []string, deps dependencies) error {
	if len(args) > 0 && wantsHelp(args) {
		fmt.Fprint(deps.stdout, migrateStageBindingsUsageText)
		return nil
	}

	// Determine the path to .kbz/stage-bindings.yaml.
	// core.RootPath() returns ".kbz" — resolve relative to CWD for CLI usage.
	kbzDir := core.RootPath()
	bindingPath := filepath.Join(kbzDir, "stage-bindings.yaml")

	data, err := os.ReadFile(bindingPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf(
				"no stage-bindings.yaml found at %s — run 'kanbanzai init' first",
				bindingPath,
			)
		}
		return fmt.Errorf("reading %s: %w", bindingPath, err)
	}

	// Check if schema_version is already present.
	if hasSchemaVersion(data) {
		fmt.Fprintln(deps.stdout, "stage-bindings.yaml already has schema_version — nothing to migrate")
		return nil
	}

	// Insert schema_version: 2 before the stage_bindings: line.
	migrated := insertSchemaVersion(data)

	if err := os.WriteFile(bindingPath, migrated, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", bindingPath, err)
	}

	fmt.Fprintln(deps.stdout, "Added schema_version: 2 to .kbz/stage-bindings.yaml")
	return nil
}

// hasSchemaVersion checks whether the YAML content already contains
// a top-level `schema_version:` key. It scans only until it reaches
// `stage_bindings:` or end of file.
func hasSchemaVersion(data []byte) bool {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		// Stop scanning once we hit stage_bindings — schema_version
		// must appear before it at the top level.
		if strings.HasPrefix(trimmed, "stage_bindings:") {
			return false
		}
		if strings.HasPrefix(trimmed, "schema_version:") {
			return true
		}
	}
	return false
}

// insertSchemaVersion inserts `schema_version: 2` on its own line
// immediately before the `stage_bindings:` line. The rest of the file
// content is preserved byte-for-byte.
func insertSchemaVersion(data []byte) []byte {
	var buf bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(data))
	inserted := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Insert schema_version before the first stage_bindings: line.
		if !inserted && strings.HasPrefix(trimmed, "stage_bindings:") {
			buf.WriteString("schema_version: 2\n")
			inserted = true
		}
		buf.WriteString(line)
		buf.WriteByte('\n')
	}

	// If we never found stage_bindings: (malformed file), still write it
	// back unchanged — caller should have caught this earlier.
	return buf.Bytes()
}

const migrateUsageText = `kanbanzai migrate <subcommand>

Migrate configuration and state files to the current schema version.

Subcommands:
  stage-bindings   Add schema_version: 2 to .kbz/stage-bindings.yaml

Use "kbz migrate <subcommand> --help" for more information about a subcommand.
`

const migrateStageBindingsUsageText = `kanbanzai migrate stage-bindings

Add a schema_version: 2 key to .kbz/stage-bindings.yaml.

This command is idempotent: if schema_version is already present, it does
nothing. The rest of the file content is preserved unchanged.
`
