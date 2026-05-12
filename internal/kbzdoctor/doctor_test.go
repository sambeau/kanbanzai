package kbzdoctor

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctor_NoKbzDir(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	d := New(&stdout, &stderr)

	results, err := d.Run(dir)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Missing {
		t.Error("expected .kbz/ to be reported as missing")
	}
	if ExitCode(results) != 1 {
		t.Error("expected exit code 1")
	}
}

func TestDoctor_AllPresent(t *testing.T) {
	dir := t.TempDir()
	createMinimalInstall(t, dir)

	var stdout, stderr bytes.Buffer
	d := New(&stdout, &stderr)

	results, err := d.Run(dir)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if ExitCode(results) != 0 {
		t.Errorf("expected exit code 0, got %d", ExitCode(results))
		for _, r := range results {
			if r.Missing || r.Warning != "" {
				t.Logf("  %s: missing=%v warning=%q", r.Path, r.Missing, r.Warning)
			}
		}
	}
}

func TestDoctor_MissingRequiredFile(t *testing.T) {
	dir := t.TempDir()
	createMinimalInstall(t, dir)

	// Delete AGENTS.md to trigger missing.
	os.Remove(filepath.Join(dir, "AGENTS.md"))

	var stdout, stderr bytes.Buffer
	d := New(&stdout, &stderr)

	results, err := d.Run(dir)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if ExitCode(results) != 1 {
		t.Error("expected exit code 1 for missing AGENTS.md")
	}

	found := false
	for _, r := range results {
		if strings.HasSuffix(r.Path, "AGENTS.md") && r.Missing {
			found = true
		}
	}
	if !found {
		t.Error("expected AGENTS.md to be reported as missing")
	}
}

func TestDoctor_UnmanagedFile(t *testing.T) {
	dir := t.TempDir()
	createMinimalInstall(t, dir)

	// Write an unmanaged AGENTS.md (no marker).
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# Custom\n"), 0o644)

	var stdout, stderr bytes.Buffer
	d := New(&stdout, &stderr)

	results, err := d.Run(dir)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	found := false
	for _, r := range results {
		if strings.HasSuffix(r.Path, "AGENTS.md") && r.Warning != "" {
			found = true
		}
	}
	if !found {
		t.Error("expected warning for unmanaged AGENTS.md")
	}
}

func TestDoctor_GhostFile(t *testing.T) {
	dir := t.TempDir()
	createMinimalInstall(t, dir)

	// Create a ghost skill file.
	ghostPath := filepath.Join(dir, ".kbz", "skills", "legacy", "SKILL.md")
	os.MkdirAll(filepath.Dir(ghostPath), 0o755)
	os.WriteFile(ghostPath, []byte("# Legacy skill\n"), 0o644)

	var stdout, stderr bytes.Buffer
	d := New(&stdout, &stderr)

	results, err := d.Run(dir)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// Ghost file should produce a warning but not change exit code.
	if ExitCode(results) != 0 {
		t.Error("expected exit code 0 (ghost files are warnings only)")
	}

	found := false
	for _, r := range results {
		if strings.Contains(r.Path, "legacy/SKILL.md") && strings.Contains(r.Warning, "ghost") {
			found = true
		}
	}
	if !found {
		t.Error("expected ghost file warning for legacy/SKILL.md")
	}
}

func TestRunTestSuite_EmptyPath(t *testing.T) {
	// Empty path should return empty result
	result := RunTestSuite("")
	if result.TotalPackages != 0 {
		t.Errorf("TotalPackages: got %d, want 0", result.TotalPackages)
	}
	if result.HasFailure {
		t.Error("HasFailure: got true, want false")
	}
}

func TestRunTestSuite_NoGoModDir(t *testing.T) {
	// A temp dir without go.mod should return empty result
	dir := t.TempDir()
	result := RunTestSuite(dir)
	if result.TotalPackages != 0 {
		t.Errorf("TotalPackages: got %d, want 0", result.TotalPackages)
	}
	if result.HasFailure {
		t.Error("HasFailure: got true, want false")
	}
}

func TestRunTestSuite_Mocked(t *testing.T) {
	// Save and restore the command runner
	origCmd := runTestSuiteCmd
	t.Cleanup(func() { runTestSuiteCmd = origCmd })

	// Set up a mock that returns passing output
	runTestSuiteCmd = func(repoRoot string) TestSuiteResult {
		return TestSuiteResult{
			TotalPackages:   10,
			PassingPackages: 10,
			FailingPackages: nil,
			FailingTests:    nil,
			HasFailure:      false,
		}
	}

	result := RunTestSuite(".")
	if result.TotalPackages != 10 {
		t.Errorf("TotalPackages: got %d, want 10", result.TotalPackages)
	}
	if result.HasFailure {
		t.Error("HasFailure: got true, want false")
	}
}

func TestRunTestSuite_MockedFailing(t *testing.T) {
	origCmd := runTestSuiteCmd
	t.Cleanup(func() { runTestSuiteCmd = origCmd })

	runTestSuiteCmd = func(repoRoot string) TestSuiteResult {
		return TestSuiteResult{
			TotalPackages:   10,
			PassingPackages: 8,
			FailingPackages: []string{"./internal/foo"},
			FailingTests:    []string{"TestFoo", "TestBar"},
			HasFailure:      true,
		}
	}

	result := RunTestSuite(".")
	if !result.HasFailure {
		t.Error("HasFailure: got false, want true")
	}
	if len(result.FailingTests) != 2 {
		t.Errorf("len(FailingTests): got %d, want 2", len(result.FailingTests))
	}
	if len(result.FailingPackages) != 1 {
		t.Errorf("len(FailingPackages): got %d, want 1", len(result.FailingPackages))
	}
}

func TestFormatTestSuiteResult(t *testing.T) {
	tests := []struct {
		name   string
		result TestSuiteResult
		want   string
	}{
		{
			name: "no failures",
			result: TestSuiteResult{
				TotalPackages:   10,
				PassingPackages: 10,
				FailingPackages: nil,
				FailingTests:    nil,
			},
			want: "10 package(s) tested, 10 passed, 0 failed",
		},
		{
			name: "with failures and test names",
			result: TestSuiteResult{
				TotalPackages:   10,
				PassingPackages: 8,
				FailingPackages: []string{"./internal/foo"},
				FailingTests:    []string{"TestFoo", "TestBar"},
			},
			want: "10 package(s) tested, 8 passed, 1 failed; failing tests: TestFoo, TestBar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTestSuiteResult(tt.result)
			if got != tt.want {
				t.Errorf("formatTestSuiteResult: got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPrintResults(t *testing.T) {
	var stdout bytes.Buffer
	d := New(&stdout, &bytes.Buffer{})

	results := []CheckResult{
		{Path: "AGENTS.md", Ok: true},
		{Path: "missing.md", Missing: true, Warning: "missing"},
		{Path: "ghost.md", Warning: "ghost file"},
	}

	d.PrintResults(results)
	out := stdout.String()

	if !strings.Contains(out, "ERROR: missing.md") {
		t.Error("expected ERROR for missing.md")
	}
	if !strings.Contains(out, "WARN:  ghost.md") {
		t.Error("expected WARN for ghost.md")
	}
}

// TestDoctor_OlderMarkerVersion verifies that when a managed file has an
// older version marker, the doctor detects it and produces a warning.
// Note: the current doctor checks for marker presence but not marker version.
// This test documents the expected behavior once version checking is implemented.
func TestDoctor_OlderMarkerVersion(t *testing.T) {
	dir := t.TempDir()
	createMinimalInstall(t, dir)

	// Replace AGENTS.md with an older version marker (v1 vs v3 in fixture).
	oldContent := "<!-- kanbanzai-managed: v1 -->\n# Old version\n"
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte(oldContent), 0o644)

	var stdout, stderr bytes.Buffer
	d := New(&stdout, &stderr)

	results, err := d.Run(dir)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Currently the doctor only checks marker presence, not version.
	// The marker IS present, so the check passes.
	// Once version checking is implemented, this should produce a warning.
	if ExitCode(results) != 0 {
		t.Error("expected exit code 0 (version check not yet implemented)")
	}

	// Verify AGENTS.md is marked OK (marker present).
	found := false
	for _, r := range results {
		if strings.HasSuffix(r.Path, "AGENTS.md") && r.Ok {
			found = true
		}
	}
	if !found {
		t.Error("expected AGENTS.md to pass (marker is present)")
	}
}

// createMinimalInstall creates a minimal valid install in dir for testing.
func createMinimalInstall(t *testing.T, dir string) {
	t.Helper()

	kbzDir := filepath.Join(dir, ".kbz")
	os.MkdirAll(kbzDir, 0o755)
	os.MkdirAll(filepath.Join(dir, ".github"), 0o755)
	os.MkdirAll(filepath.Join(dir, ".zed"), 0o755)
	os.MkdirAll(filepath.Join(dir, ".agents", "skills"), 0o755)
	os.MkdirAll(filepath.Join(kbzDir, "skills"), 0o755)
	os.MkdirAll(filepath.Join(kbzDir, "roles"), 0o755)

	managed := "<!-- kanbanzai-managed: v3 -->\n# Content\n"
	os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte(managed), 0o644)
	os.WriteFile(filepath.Join(dir, ".github", "copilot-instructions.md"), []byte(managed), 0o644)

	mcpJSON := `{"kanbanzai-managed": true}`
	os.WriteFile(filepath.Join(dir, ".mcp.json"), []byte(mcpJSON), 0o644)
	os.WriteFile(filepath.Join(dir, ".zed", "settings.json"), []byte(mcpJSON), 0o644)

	configYAML := "version: \"2\"\nname: test\n"
	os.WriteFile(filepath.Join(kbzDir, "config.yaml"), []byte(configYAML), 0o644)

	bindings := "# kanbanzai-managed: true\n# kanbanzai-version: dev\n"
	os.WriteFile(filepath.Join(kbzDir, "stage-bindings.yaml"), []byte(bindings), 0o644)

	os.WriteFile(filepath.Join(kbzDir, ".init-complete"), []byte{}, 0o644)
}
