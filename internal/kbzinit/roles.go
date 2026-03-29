package kbzinit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// baseYAMLContent is the scaffold role file for the project team to own.
// It has no managed marker and is never overwritten by kbz init.
const baseYAMLContent = `id: base
description: "Project-wide conventions for all agents"
# Add your project's global conventions here.
# All other roles inherit from base unless they declare their own inherits field.
conventions: []
# architecture:
#   summary: "One paragraph describing the overall project structure"
#   key_interfaces:
#     - "The most important files/packages and what they do"
`

// reviewerYAMLTemplate is the kanbanzai-managed reviewer role template.
// The version placeholder is replaced at install time with the binary version.
const reviewerYAMLTemplate = `id: reviewer
inherits: base
description: "Context profile for code review agents. Provides review dimensions, structured output format, and quality gate criteria."
metadata:
  kanbanzai-managed: "true"
  version: "VERSION_PLACEHOLDER"
conventions:
  review_approach:
    - "Review is structured, not conversational. Produce findings, not commentary."
    - "Every finding has a dimension, severity, location, and description."
    - "Blocking findings must cite the specific requirement or convention violated."
    - "Non-blocking findings are suggestions, not demands."
    - "When uncertain whether something is a defect, classify as concern, not fail."
  output_format:
    - "Use the structured review output format from the kanbanzai-review skill."
    - "Report per-dimension outcomes: pass, pass_with_notes, concern, fail, not_applicable."
    - "Report overall verdict: approved, approved_with_followups, changes_required, blocked."
    - "List blocking findings separately from non-blocking notes."
  dimensions:
    - "Specification conformance: does the implementation match the approved spec?"
    - "Implementation quality: is the code correct, idiomatic, and maintainable?"
    - "Test adequacy: are tests appropriate, sufficient, and well-structured?"
    - "Documentation currency: is documentation accurate and up to date?"
    - "Workflow integrity: is the workflow state consistent with the work?"
`

const yamlManagedMarker = `kanbanzai-managed: "true"`

// installRoles installs base.yaml and reviewer.yaml into <kbzDir>/context/roles/.
// base.yaml is only created if absent (never overwritten).
// reviewer.yaml uses version-aware managed logic.
func (i *Initializer) installRoles(kbzDir string) error {
	rolesDir := filepath.Join(kbzDir, "context", "roles")
	if err := os.MkdirAll(rolesDir, 0o755); err != nil {
		return fmt.Errorf("create roles dir: %w", err)
	}

	if err := i.writeBaseRole(rolesDir); err != nil {
		return err
	}
	return i.writeReviewerRole(rolesDir)
}

// writeBaseRole writes base.yaml only if it does not already exist.
func (i *Initializer) writeBaseRole(rolesDir string) error {
	destPath := filepath.Join(rolesDir, "base.yaml")
	if _, err := os.Stat(destPath); err == nil {
		// Already exists — leave it alone, no message.
		return nil
	}
	if err := os.WriteFile(destPath, []byte(baseYAMLContent), 0o644); err != nil {
		return fmt.Errorf("write base.yaml: %w", err)
	}
	fmt.Fprintln(i.stdout, "Created .kbz/context/roles/base.yaml")
	return nil
}

// writeReviewerRole writes reviewer.yaml using version-aware managed logic.
func (i *Initializer) writeReviewerRole(rolesDir string) error {
	destPath := filepath.Join(rolesDir, "reviewer.yaml")
	content := strings.ReplaceAll(reviewerYAMLTemplate, "VERSION_PLACEHOLDER", i.version)

	existing, readErr := os.ReadFile(destPath)
	if readErr != nil {
		if !os.IsNotExist(readErr) {
			return fmt.Errorf("read reviewer.yaml: %w", readErr)
		}
		// Create new.
		if err := os.WriteFile(destPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write reviewer.yaml: %w", err)
		}
		fmt.Fprintln(i.stdout, "Created .kbz/context/roles/reviewer.yaml")
		return nil
	}

	// File exists — check managed marker.
	if !strings.Contains(string(existing), yamlManagedMarker) {
		fmt.Fprintf(i.stdout, "Warning: .kbz/context/roles/reviewer.yaml exists but is not managed by kanbanzai (no managed marker). Skipping.\n")
		return nil
	}

	// Extract existing version.
	existingVersion := extractYAMLVersion(existing)
	if existingVersion == i.version {
		// At current version — no-op.
		return nil
	}

	// Older managed version — overwrite.
	if err := os.WriteFile(destPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("update reviewer.yaml: %w", err)
	}
	fmt.Fprintln(i.stdout, "Updated .kbz/context/roles/reviewer.yaml")
	return nil
}

// updateManagedRoles updates reviewer.yaml if it is managed and at an older version.
// It never touches base.yaml.
func (i *Initializer) updateManagedRoles(kbzDir string) error {
	rolesDir := filepath.Join(kbzDir, "context", "roles")
	// Only try if reviewer.yaml exists.
	destPath := filepath.Join(rolesDir, "reviewer.yaml")
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		return nil
	}
	return i.writeReviewerRole(rolesDir)
}

// extractYAMLVersion extracts the version value from a reviewer.yaml file.
// Looks for a line like:   version: "1.0.0"
func extractYAMLVersion(data []byte) string {
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "version:") {
			v := strings.TrimPrefix(trimmed, "version:")
			v = strings.TrimSpace(v)
			v = strings.Trim(v, `"`)
			return v
		}
	}
	return ""
}
