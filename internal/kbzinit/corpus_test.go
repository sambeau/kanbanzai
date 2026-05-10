package kbzinit

import (
	"bufio"
	"fmt"
	"strings"
	"testing"
)

// =============================================================================
// TestEmbeddedCorpus — structural assertions against the embedded corpus
// =============================================================================

// TestEmbeddedCorpus verifies four invariants against the live Manifest:
//   1. Every Manifest entry with a non-empty EmbedPath resolves in its FS.
//   2. Every embedded skill/role contains its marker line and parseable version.
//   3. AGENTS.md references no workflow skill absent from the Manifest.
//   4. stage-bindings.yaml references no role absent from the Manifest.
func TestEmbeddedCorpus(t *testing.T) {
	t.Run("embed_fs_coverage", testEmbedFSCoverage)
	t.Run("marker_and_version_parseable", testMarkerAndVersionParseable)
	t.Run("agentsmd_drift_check", testAgentsMDDriftCheck)
	t.Run("dangling_role_references", testDanglingRoleReferences)
}

// ---------------------------------------------------------------------------
// 1. Embed FS coverage
// ---------------------------------------------------------------------------

func testEmbedFSCoverage(t *testing.T) {
	for _, a := range Manifest {
		if a.EmbedPath == "" {
			continue // generated content (AGENTS.md, copilot-instructions.md)
		}
		if err := resolveEmbedded(a); err != nil {
			t.Errorf("Manifest entry %s (Kind=%s EmbedPath=%s): %v", a.Name, a.Kind, a.EmbedPath, err)
		}
	}
}

// resolveEmbedded reads the embedded content for an artifact from the
// appropriate embed.FS. Returns an error if the path cannot be resolved.
func resolveEmbedded(a Artifact) error {
	switch a.Kind {
	case WorkflowSkill:
		_, err := embeddedSkills.ReadFile(a.EmbedPath)
		return err
	case TaskSkill:
		_, err := embeddedTaskSkills.ReadFile(a.EmbedPath)
		return err
	case Role:
		_, err := embeddedRoles.ReadFile(a.EmbedPath)
		return err
	case StageBindings:
		_ = embeddedStageBindings // compile-time check
		return nil
	default:
		return nil
	}
}

// ---------------------------------------------------------------------------
// 2. Marker and version parseability
// ---------------------------------------------------------------------------

func testMarkerAndVersionParseable(t *testing.T) {
	for _, a := range Manifest {
		if a.EmbedPath == "" || a.Marker.Comment == "" {
			continue // generated content or no marker (base.yaml)
		}

		data, err := readEmbeddedBytes(a)
		if err != nil {
			t.Errorf("%s: cannot read embedded source: %v", a.Name, err)
			continue
		}

		// Find the marker line.
		line, found := findMarkerLine(data, a.Marker.Comment)
		if !found {
			t.Errorf("%s: embedded source missing marker line %q", a.Name, a.Marker.Comment)
			continue
		}

		// Extract and parse the version.
		raw := strings.TrimPrefix(line, a.Marker.Comment)
		raw = strings.TrimSuffix(strings.TrimSpace(raw), "-->")
		raw = strings.TrimSpace(raw)
		raw = strings.Trim(raw, `"`)

		// "dev" is the canonical build-time placeholder.
		if raw == "dev" || raw == "" {
			continue
		}

		switch a.Marker.VersionKind {
		case IntCounter:
			if _, err := parseIntVersion(raw); err != nil {
				t.Errorf("%s: unparseable IntCounter version %q from line %q", a.Name, raw, line)
			}
		case Semver:
			if _, err := parseSemverParts(raw); err != nil {
				t.Errorf("%s: unparseable Semver version %q from line %q", a.Name, raw, line)
			}
		}
	}
}

// readEmbeddedBytes reads the raw embedded bytes for an artifact.
func readEmbeddedBytes(a Artifact) ([]byte, error) {
	switch a.Kind {
	case WorkflowSkill:
		return embeddedSkills.ReadFile(a.EmbedPath)
	case TaskSkill:
		return embeddedTaskSkills.ReadFile(a.EmbedPath)
	case Role:
		return embeddedRoles.ReadFile(a.EmbedPath)
	case StageBindings:
		return embeddedStageBindings, nil
	default:
		return nil, nil
	}
}

// parseIntVersion parses a version string as an integer counter.
func parseIntVersion(s string) (int, error) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("not an integer: %q", s)
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

// ---------------------------------------------------------------------------
// 3. AGENTS.md / copilot-instructions.md drift check
// ---------------------------------------------------------------------------

func testAgentsMDDriftCheck(t *testing.T) {
	manifestNames := workflowSkillManifestNames()

	for _, entry := range []struct {
		name    string
		content string
	}{
		{"AGENTS.md", agentsMDContent},
		{"copilot-instructions.md", copilotInstructionsContent},
	} {
		refs := extractWorkflowSkillRefs(entry.content)
		for _, ref := range refs {
			if !manifestNames[ref] {
				t.Errorf("%s references workflow skill %q not found in Manifest", entry.name, ref)
			}
		}
	}
}

// workflowSkillManifestNames returns a set of WorkflowSkill artifact names.
func workflowSkillManifestNames() map[string]bool {
	names := make(map[string]bool)
	for _, a := range Manifest {
		if a.Kind == WorkflowSkill {
			names[a.Name] = true
		}
	}
	return names
}

// extractWorkflowSkillRefs parses a markdown content string and returns
// skill names found in backtick-quoted entries matching the kanbanzai-
// prefix (workflow skill naming convention).
func extractWorkflowSkillRefs(content string) []string {
	var refs []string
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		for {
			start := strings.Index(line, "`")
			if start == -1 {
				break
			}
			line = line[start+1:]
			end := strings.Index(line, "`")
			if end == -1 {
				break
			}
			name := line[:end]
			line = line[end+1:]
			if strings.HasPrefix(name, "kanbanzai-") {
				refs = append(refs, name)
			}
		}
	}
	return refs
}

// ---------------------------------------------------------------------------
// 4. Dangling role references from stage-bindings.yaml
// ---------------------------------------------------------------------------

func testDanglingRoleReferences(t *testing.T) {
	manifestRoles := roleManifestNames()
	refs := extractRoleRefs(string(embeddedStageBindings))

	for _, ref := range refs {
		if !manifestRoles[ref] {
			t.Errorf("stage-bindings.yaml references role %q not found in Manifest", ref)
		}
	}
}

// roleManifestNames returns a set of Role artifact names.
func roleManifestNames() map[string]bool {
	names := make(map[string]bool)
	for _, a := range Manifest {
		if a.Kind == Role {
			names[a.Name] = true
		}
	}
	return names
}

// extractRoleRefs parses stage-bindings.yaml content and returns role
// names found after `role:` keys. Only returns entries that look like
// single-word identifiers (no spaces, no quotes) to avoid picking up
// section headers or descriptions.
func extractRoleRefs(content string) []string {
	var refs []string
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "- ") {
			continue
		}
		role := strings.TrimPrefix(line, "- ")
		role = strings.TrimSpace(role)
		// Filter: single lowercase word possibly with hyphens.
		if role == "" || strings.Contains(role, " ") || strings.HasPrefix(role, "#") ||
			strings.HasPrefix(role, `"`) || strings.Contains(role, ".") {
			continue
		}
		refs = append(refs, role+".yaml")
	}
	return refs
}

// =============================================================================
// Negative fixture sub-tests (AC-006, AC-007, AC-008)
// =============================================================================

// TestEmbeddedCorpus_MissingMarker (AC-006):
// Synthetic fixture whose bytes lack the expected marker.
func TestEmbeddedCorpus_MissingMarker(t *testing.T) {
	a := Artifact{
		Name:   "test-fixture",
		Kind:   WorkflowSkill,
		Marker: MarkerSpec{Comment: "# kanbanzai-version:", VersionKind: Semver},
	}
	data := []byte("# No version marker here\nJust content.\n")
	_, found := findMarkerLine(data, a.Marker.Comment)
	if found {
		t.Error("expected marker to be missing from synthetic fixture")
	}
}

// TestEmbeddedCorpus_AgentsMDDrift (AC-007):
// Synthetic AGENTS.md referencing a skill not in the Manifest.
func TestEmbeddedCorpus_AgentsMDDrift(t *testing.T) {
	manifestNames := workflowSkillManifestNames()
	syntheticContent := "# AGENTS.md\n\n`kanbanzai-nonexistent` is listed but not shipped.\n"
	refs := extractWorkflowSkillRefs(syntheticContent)

	foundMissing := false
	for _, ref := range refs {
		if !manifestNames[ref] {
			foundMissing = true
		}
	}
	if !foundMissing {
		t.Error("expected kanbanzai-nonexistent to be missing from Manifest")
	}
}

// TestEmbeddedCorpus_DanglingRole (AC-008):
// Synthetic stage-bindings.yaml referencing a role not in the Manifest.
func TestEmbeddedCorpus_DanglingRole(t *testing.T) {
	manifestRoles := roleManifestNames()
	syntheticYAML := "roles:\n  - nonexistent-role\n  - architect\n"
	refs := extractRoleRefs(syntheticYAML)

	foundMissing := false
	for _, ref := range refs {
		if !manifestRoles[ref] {
			foundMissing = true
		}
	}
	if !foundMissing {
		t.Error("expected nonexistent-role.yaml to be missing from Manifest")
	}
}
