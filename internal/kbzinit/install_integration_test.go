package kbzinit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// =============================================================================
// Integration tests for compareManaged-driven install behaviour (AC-003–AC-005)
// =============================================================================

// TestIntegration_NewerMarker_NoOp (AC-003):
// Pre-write an AGENTS.md with a managed marker at version v999, call
// writeAgentsMD, and assert the file is unchanged (no overwrite) and no
// "Updated" message is printed.
func TestIntegration_NewerMarker_NoOp(t *testing.T) {
	dir := t.TempDir()

	// Pre-write AGENTS.md with a newer managed version than binary.
	newerContent := "<!-- kanbanzai-managed: v999 -->\n\n# Newer AGENTS.md\nPreserve this.\n"
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte(newerContent), 0o644); err != nil {
		t.Fatal(err)
	}

	init, stdout := newTestInit(dir, "")
	if err := init.writeAgentsMD(dir); err != nil {
		t.Fatalf("writeAgentsMD: %v", err)
	}

	// File content must be unchanged.
	data, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	if string(data) != newerContent {
		t.Error("AGENTS.md was modified despite newer managed version")
	}

	// No "Updated" printed.
	if strings.Contains(stdout.String(), "Updated") {
		t.Errorf("expected no 'Updated' output, got: %s", stdout.String())
	}
}

// TestIntegration_UnparseableVersion_WarnSkip (AC-004):
// Pre-write a role file with an unparseable version, call installRoles,
// assert the file is unchanged and a warning is printed.
func TestIntegration_UnparseableVersion_WarnSkip(t *testing.T) {
	dir := t.TempDir()

	// Create the .kbz/roles/ dir and pre-write architect.yaml with garbage version.
	rolesDir := filepath.Join(dir, ".kbz", "roles")
	if err := os.MkdirAll(rolesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Also pre-create base.yaml so installRoles doesn't try to write it.
	if err := os.WriteFile(filepath.Join(rolesDir, "base.yaml"), []byte("base: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	garbageContent := "version: \"garbage\"\n# role content\n"
	architectPath := filepath.Join(rolesDir, "architect.yaml")
	if err := os.WriteFile(architectPath, []byte(garbageContent), 0o644); err != nil {
		t.Fatal(err)
	}

	init, stdout := newTestInit(dir, "")
	// installRoles takes baseDir (project root), not kbzDir.
	if err := init.installRoles(dir); err != nil {
		t.Fatalf("installRoles: %v", err)
	}

	// File content must be unchanged.
	data, err := os.ReadFile(architectPath)
	if err != nil {
		t.Fatalf("read architect.yaml: %v", err)
	}
	if string(data) != garbageContent {
		t.Error("architect.yaml was modified despite unparseable version")
	}

	// Warning must be printed.
	if !strings.Contains(stdout.String(), "Warning") {
		t.Errorf("expected warning in output, got: %s", stdout.String())
	}
}

// TestIntegration_StageBindings_NoDoubleWrite (AC-005):
// Call installStageBindings twice. The second call must not print "Updated
// .kbz/stage-bindings.yaml" (validates REQ-NF-002: Semver VersionKind fixes
// the always-rewrite defect).
func TestIntegration_StageBindings_NoDoubleWrite(t *testing.T) {
	dir := t.TempDir()

	// First install: should create.
	init1, stdout1 := newTestInit(dir, "")
	if err := init1.installStageBindings(dir); err != nil {
		t.Fatalf("first installStageBindings: %v", err)
	}
	if !strings.Contains(stdout1.String(), "Created") {
		t.Errorf("first install should create, got: %s", stdout1.String())
	}

	// Second install: should be NoOp (same version).
	init2, stdout2 := newTestInit(dir, "")
	if err := init2.installStageBindings(dir); err != nil {
		t.Fatalf("second installStageBindings: %v", err)
	}
	if strings.Contains(stdout2.String(), "Updated") || strings.Contains(stdout2.String(), "Created") {
		t.Errorf("second install should be no-op, got: %s", stdout2.String())
	}
}
