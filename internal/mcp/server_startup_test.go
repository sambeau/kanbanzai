package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/kanbanzai/internal/config"
)

// Note: These tests use os.Chdir to set up a fake .kbz/ directory.
// They must NOT use t.Parallel() because os.Chdir affects the process globally.

func TestNewServer_BindingValidation_HardError(t *testing.T) {
	dir := setupFakeKBZ(t, `schema_version: 2
stage_bindings:
  plan-reviewing:
    description: "Stale stage that is not in validStages"
    orchestration: single-agent
    roles: [architect]
    skills: [write-design]
`)
	origDir := cd(t, dir)

	cfg := config.DefaultConfig()
	cfg.MCP.Preset = "minimal"

	_, err := newServerWithConfig("", &cfg)
	os.Chdir(origDir)

	if err == nil {
		t.Fatal("expected hard error for invalid stage-bindings.yaml, got nil")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "binding validation failed") {
		t.Errorf("error should indicate binding validation failure, got: %s", errStr)
	}
	if !strings.Contains(errStr, "kbz binding doctor") {
		t.Errorf("error should mention 'kbz binding doctor' as recovery path, got: %s", errStr)
	}
	if !strings.Contains(errStr, "kbz init --upgrade") {
		t.Errorf("error should mention 'kbz init --upgrade' for consumer files, got: %s", errStr)
	}
	if !strings.Contains(errStr, "plan-reviewing") {
		t.Errorf("error should name the invalid stage, got: %s", errStr)
	}
}

func TestNewServer_BindingLoad_SoftWarning(t *testing.T) {
	dir := t.TempDir()
	kbzDir := filepath.Join(dir, ".kbz")
	if err := os.MkdirAll(kbzDir, 0o755); err != nil {
		t.Fatal(err)
	}
	origDir := cd(t, dir)

	cfg := config.DefaultConfig()
	cfg.MCP.Preset = "minimal"

	srv, err := newServerWithConfig("", &cfg)
	os.Chdir(origDir)

	if err != nil {
		t.Fatalf("expected soft warning (no error) when binding file is missing, got: %v", err)
	}
	if srv == nil {
		t.Fatal("expected non-nil server even with missing binding file")
	}
}

func TestNewServer_BindingValid_Success(t *testing.T) {
	dir := setupFakeKBZ(t, `schema_version: 2
stage_bindings:
  designing:
    description: "Design stage"
    orchestration: single-agent
    roles: [architect]
    skills: [write-design]
    human_gate: false
`)
	// Create a roles directory with a dummy architect role so the
	// role checker doesn't produce warnings (but warnings aren't errors).
	rolesDir := filepath.Join(dir, ".kbz", "roles")
	if err := os.MkdirAll(rolesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	origDir := cd(t, dir)

	cfg := config.DefaultConfig()
	cfg.MCP.Preset = "minimal"

	srv, err := newServerWithConfig("", &cfg)
	os.Chdir(origDir)

	if err != nil {
		t.Fatalf("expected success with valid binding file, got: %v", err)
	}
	if srv == nil {
		t.Fatal("expected non-nil server with valid binding file")
	}
}

func TestNewServer_BindingValidation_ErrorFormatting(t *testing.T) {
	dir := setupFakeKBZ(t, `schema_version: 2
stage_bindings:
  designing:
    description: ""
    orchestration: bogus-orch
    roles: []
    skills: []
`)
	origDir := cd(t, dir)

	cfg := config.DefaultConfig()
	cfg.MCP.Preset = "minimal"

	_, err := newServerWithConfig("", &cfg)
	os.Chdir(origDir)

	if err == nil {
		t.Fatal("expected error for broken binding")
	}

	errStr := err.Error()

	wantSubstrs := []string{
		"description must not be empty",
		"invalid orchestration",
		"roles must not be empty",
		"skills must not be empty",
		"kbz binding doctor",
		"kbz init --upgrade",
	}
	for _, want := range wantSubstrs {
		if !strings.Contains(errStr, want) {
			t.Errorf("error should contain %q, got: %s", want, errStr)
		}
	}
}

// setupFakeKBZ creates a temp dir with .kbz/stage-bindings.yaml and returns the dir path.
func setupFakeKBZ(t *testing.T, yamlContent string) string {
	t.Helper()
	dir := t.TempDir()
	bindingDir := filepath.Join(dir, ".kbz")
	if err := os.MkdirAll(bindingDir, 0o755); err != nil {
		t.Fatal(err)
	}
	bindingPath := filepath.Join(bindingDir, "stage-bindings.yaml")
	if err := os.WriteFile(bindingPath, []byte(yamlContent), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// cd changes to dir and returns the previous working directory.
func cd(t *testing.T, dir string) string {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	return orig
}
