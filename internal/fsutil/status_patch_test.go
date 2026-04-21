package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeTestFile: %v", err)
	}
	return path
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("readTestFile: %v", err)
	}
	return string(data)
}

// AC-001: pipe-table row
func TestPatchStatusField_PipeTable(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := writeTestFile(t, dir, "doc.md", "# Title\n\n| Field | Value |\n| Status | Draft |\n| Author | Alice |\n")

	patched, err := PatchStatusField(path, "approved")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !patched {
		t.Fatal("want patched=true, got false")
	}
	got := readTestFile(t, path)
	want := "# Title\n\n| Field | Value |\n| Status | approved |\n| Author | Alice |\n"
	if got != want {
		t.Errorf("file content mismatch\ngot:  %q\nwant: %q", got, want)
	}
}

// AC-002: bullet-list item
func TestPatchStatusField_BulletList(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := writeTestFile(t, dir, "doc.md", "# Doc\n\n- Status: draft\n- Owner: Bob\n")

	patched, err := PatchStatusField(path, "approved")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !patched {
		t.Fatal("want patched=true, got false")
	}
	got := readTestFile(t, path)
	want := "# Doc\n\n- Status: approved\n- Owner: Bob\n"
	if got != want {
		t.Errorf("file content mismatch\ngot:  %q\nwant: %q", got, want)
	}
}

// AC-003: bare YAML key
func TestPatchStatusField_BareYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := writeTestFile(t, dir, "doc.md", "---\ntitle: My Doc\nstatus: draft\nauthor: Carol\n---\n")

	patched, err := PatchStatusField(path, "approved")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !patched {
		t.Fatal("want patched=true, got false")
	}
	got := readTestFile(t, path)
	want := "---\ntitle: My Doc\nstatus: approved\nauthor: Carol\n---\n"
	if got != want {
		t.Errorf("file content mismatch\ngot:  %q\nwant: %q", got, want)
	}
}

// AC-004: case-insensitive field name (| STATUS | Draft |)
func TestPatchStatusField_CaseInsensitive(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := writeTestFile(t, dir, "doc.md", "| STATUS | Draft |\n")

	patched, err := PatchStatusField(path, "approved")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !patched {
		t.Fatal("want patched=true, got false")
	}
	got := readTestFile(t, path)
	want := "| Status | approved |\n"
	if got != want {
		t.Errorf("file content mismatch\ngot:  %q\nwant: %q", got, want)
	}
}

// AC-005: no Status field → (false, nil)
func TestPatchStatusField_NoField(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := writeTestFile(t, dir, "doc.md", "# Title\n\nNo status here.\n")

	patched, err := PatchStatusField(path, "approved")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if patched {
		t.Fatal("want patched=false, got true")
	}
	// File should be unchanged
	got := readTestFile(t, path)
	want := "# Title\n\nNo status here.\n"
	if got != want {
		t.Errorf("file should be unchanged, got: %q", got)
	}
}

// AC-006: only the first Status line is replaced
func TestPatchStatusField_FirstMatchOnly(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := writeTestFile(t, dir, "doc.md", "- Status: draft\n- Status: pending\n")

	patched, err := PatchStatusField(path, "approved")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !patched {
		t.Fatal("want patched=true, got false")
	}
	got := readTestFile(t, path)
	want := "- Status: approved\n- Status: pending\n"
	if got != want {
		t.Errorf("file content mismatch\ngot:  %q\nwant: %q", got, want)
	}
}

// AC-007: file not found → (false, err)
func TestPatchStatusField_FileNotFound(t *testing.T) {
	t.Parallel()
	patched, err := PatchStatusField("/nonexistent/path/doc.md", "approved")
	if err == nil {
		t.Fatal("want error for missing file, got nil")
	}
	if patched {
		t.Fatal("want patched=false on error, got true")
	}
}
