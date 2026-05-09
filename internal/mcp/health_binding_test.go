package mcp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBindingLoadableHealthChecker_ValidBinding(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "stage-bindings.yaml")
	if err := os.WriteFile(path, []byte("stage_bindings: {}\n"), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	checker := BindingLoadableHealthChecker(path)
	report, err := checker()
	if err != nil {
		t.Fatalf("checker error: %v", err)
	}
	if len(report.Warnings) != 0 {
		t.Errorf("expected no warnings for valid binding, got: %v", report.Warnings)
	}
	if report.Summary.WarningCount != 0 {
		t.Errorf("WarningCount = %d, want 0", report.Summary.WarningCount)
	}
}

func TestBindingLoadableHealthChecker_MalformedBinding(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "stage-bindings.yaml")
	if err := os.WriteFile(path, []byte("not: valid: yaml: [{\n"), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	checker := BindingLoadableHealthChecker(path)
	report, err := checker()
	if err != nil {
		t.Fatalf("checker error: %v", err)
	}
	if len(report.Warnings) == 0 {
		t.Fatal("expected warnings for malformed binding, got none")
	}
	if report.Warnings[0].EntityType != "binding_loadable" {
		t.Errorf("warning EntityType = %q, want %q", report.Warnings[0].EntityType, "binding_loadable")
	}
	if report.Summary.WarningCount != 1 {
		t.Errorf("WarningCount = %d, want 1", report.Summary.WarningCount)
	}
}

func TestBindingLoadableHealthChecker_MissingFile(t *testing.T) {
	t.Parallel()

	checker := BindingLoadableHealthChecker("/nonexistent/path/stage-bindings.yaml")
	report, err := checker()
	if err != nil {
		t.Fatalf("checker error: %v", err)
	}
	if len(report.Warnings) == 0 {
		t.Fatal("expected warnings for missing file, got none")
	}
	if report.Warnings[0].EntityType != "binding_loadable" {
		t.Errorf("warning EntityType = %q, want %q", report.Warnings[0].EntityType, "binding_loadable")
	}
}

func TestBindingLoadableHealthChecker_EmptyPath(t *testing.T) {
	t.Parallel()

	checker := BindingLoadableHealthChecker("")
	report, err := checker()
	if err != nil {
		t.Fatalf("checker error: %v", err)
	}
	if len(report.Warnings) == 0 {
		t.Fatal("expected warning for empty path, got none")
	}
	if report.Warnings[0].EntityType != "binding_loadable" {
		t.Errorf("warning EntityType = %q, want %q", report.Warnings[0].EntityType, "binding_loadable")
	}
}
