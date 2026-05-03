package kbzinit

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

//go:embed stage-bindings.yaml
var embeddedStageBindings []byte

const stageBindingsVersion = 1

const stageBindingsManagedMarker = "# kanbanzai-managed:"
const stageBindingsVersionMarker = "# kanbanzai-version:"

// installStageBindings installs the canonical stage-bindings.yaml into
// <kbzDir>/stage-bindings.yaml.
// It applies version-aware create/update/skip logic using YAML comment
// markers on lines 1-2 of the file.
func (i *Initializer) installStageBindings(kbzDir string) error {
	destPath := filepath.Join(kbzDir, "stage-bindings.yaml")

	// Prepare the content with the proper version injected.
	content := transformStageBindingsContent(embeddedStageBindings, i.version)

	// Ensure the .kbz/ directory exists (e.g., --update-skills on a project
	// that has .mcp.json but no .kbz/ yet).
	if err := os.MkdirAll(kbzDir, 0o755); err != nil {
		return fmt.Errorf("create .kbz/: %w", err)
	}

	existing, readErr := os.ReadFile(destPath)
	if readErr != nil {
		if !os.IsNotExist(readErr) {
			return fmt.Errorf("read existing stage-bindings.yaml: %w", readErr)
		}
		// File does not exist — create it.
		if err := os.WriteFile(destPath, content, 0o644); err != nil {
			return fmt.Errorf("write stage-bindings.yaml: %w", err)
		}
		fmt.Fprintln(i.stdout, "Created .kbz/stage-bindings.yaml")
		return nil
	}

	// File exists — check for the managed marker on line 1.
	if !hasLinePrefix(existing, stageBindingsManagedMarker) {
		fmt.Fprintf(i.stdout, "Warning: .kbz/stage-bindings.yaml exists but is not managed by kanbanzai (no '%s' marker). Skipping.\n", stageBindingsManagedMarker)
		return nil
	}

	// Has managed marker — check version.
	existingVersion := extractStageBindingsVersion(existing)
	if existingVersion == i.version && i.version != "dev" {
		// Already at current version — no-op.
		return nil
	}

	// Older managed version, or dev build — overwrite.
	if err := os.WriteFile(destPath, content, 0o644); err != nil {
		return fmt.Errorf("update stage-bindings.yaml: %w", err)
	}
	fmt.Fprintln(i.stdout, "Updated .kbz/stage-bindings.yaml")
	return nil
}

// transformStageBindingsContent replaces the version placeholder in the
// embedded stage-bindings.yaml with the actual binary version.
func transformStageBindingsContent(src []byte, version string) []byte {
	var buf bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(src))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, stageBindingsManagedMarker) {
			buf.WriteString("# kanbanzai-managed: true\n")
			continue
		}
		if strings.HasPrefix(line, stageBindingsVersionMarker) {
			buf.WriteString("# kanbanzai-version: " + version + "\n")
			continue
		}
		buf.WriteString(line)
		buf.WriteByte('\n')
	}
	return buf.Bytes()
}

// hasLinePrefix reports whether data contains a line that starts with prefix.
func hasLinePrefix(data []byte, prefix string) bool {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), prefix) {
			return true
		}
	}
	return false
}

// extractStageBindingsVersion returns the version from the first line starting
// with "# kanbanzai-version:". Returns empty string if not found.
func extractStageBindingsVersion(data []byte) string {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, stageBindingsVersionMarker) {
			v := strings.TrimPrefix(line, stageBindingsVersionMarker)
			v = strings.TrimSpace(v)
			// Validate it's an integer.
			if _, err := strconv.Atoi(v); err == nil {
				return v
			}
			return ""
		}
	}
	return ""
}
