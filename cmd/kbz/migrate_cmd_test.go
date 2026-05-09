package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunMigrate_NoArgs_PrintsUsage(t *testing.T) {
	deps, output := testDependencies()

	if err := runMigrate(nil, deps); err != nil {
		t.Fatalf("runMigrate(nil) error = %v", err)
	}

	stdout := output.String()
	if !strings.Contains(stdout, "kanbanzai migrate") {
		t.Fatalf("stdout missing migrate header:\n%s", stdout)
	}
}

func TestRunMigrate_Help_PrintsUsage(t *testing.T) {
	deps, output := testDependencies()

	if err := runMigrate([]string{"--help"}, deps); err != nil {
		t.Fatalf("runMigrate(--help) error = %v", err)
	}

	stdout := output.String()
	if !strings.Contains(stdout, "kanbanzai migrate") {
		t.Fatalf("stdout missing migrate header:\n%s", stdout)
	}
}

func TestRunMigrateStageBindings_Help_PrintsUsage(t *testing.T) {
	deps, output := testDependencies()

	if err := runMigrateStageBindings([]string{"--help"}, deps); err != nil {
		t.Fatalf("runMigrateStageBindings(--help) error = %v", err)
	}

	stdout := output.String()
	if !strings.Contains(stdout, "schema_version: 2") {
		t.Fatalf("stdout missing schema_version details:\n%s", stdout)
	}
}

func TestRunMigrateStageBindings_MissingFile_ReturnsError(t *testing.T) {
	// Use a temp directory with no .kbz/ to trigger the missing-file error.
	dir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer os.Chdir(origDir)
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	deps, _ := testDependencies()
	err = runMigrateStageBindings(nil, deps)
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
	if !strings.Contains(err.Error(), "no stage-bindings.yaml found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunMigrateStageBindings_AlreadyHasSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	kbzDir := filepath.Join(dir, ".kbz")
	if err := os.MkdirAll(kbzDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	yaml := "schema_version: 2\nstage_bindings:\n  designing:\n    description: test\n"
	if err := os.WriteFile(filepath.Join(kbzDir, "stage-bindings.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer os.Chdir(origDir)
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	deps, output := testDependencies()
	if err := runMigrateStageBindings(nil, deps); err != nil {
		t.Fatalf("runMigrateStageBindings error = %v", err)
	}

	stdout := output.String()
	if !strings.Contains(stdout, "already has schema_version") {
		t.Fatalf("expected 'already has schema_version', got:\n%s", stdout)
	}
}

func TestRunMigrateStageBindings_AddsSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	kbzDir := filepath.Join(dir, ".kbz")
	if err := os.MkdirAll(kbzDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	yaml := "stage_bindings:\n  designing:\n    description: test\n"
	if err := os.WriteFile(filepath.Join(kbzDir, "stage-bindings.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer os.Chdir(origDir)
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	deps, output := testDependencies()
	if err := runMigrateStageBindings(nil, deps); err != nil {
		t.Fatalf("runMigrateStageBindings error = %v", err)
	}

	stdout := output.String()
	if !strings.Contains(stdout, "Added schema_version: 2") {
		t.Fatalf("expected 'Added schema_version: 2', got:\n%s", stdout)
	}

	// Verify file now has schema_version.
	data, err := os.ReadFile(filepath.Join(kbzDir, "stage-bindings.yaml"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "schema_version: 2") {
		t.Fatalf("file missing schema_version after migrate:\n%s", content)
	}
	if !strings.Contains(content, "stage_bindings:") {
		t.Fatal("file missing stage_bindings after migrate")
	}
	// Verify schema_version comes before stage_bindings.
	svIdx := strings.Index(content, "schema_version: 2")
	sbIdx := strings.Index(content, "stage_bindings:")
	if svIdx < 0 || sbIdx < 0 || svIdx >= sbIdx {
		t.Fatalf("schema_version must appear before stage_bindings:\n%s", content)
	}

	// Run again — should be idempotent.
	deps2, output2 := testDependencies()
	if err := runMigrateStageBindings(nil, deps2); err != nil {
		t.Fatalf("second runMigrateStageBindings error = %v", err)
	}
	if !strings.Contains(output2.String(), "already has schema_version") {
		t.Fatalf("second run should report already migrated, got:\n%s", output2.String())
	}
}

func TestHasSchemaVersion(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want bool
	}{
		{"has schema_version", "schema_version: 2\nstage_bindings:\n  foo: bar\n", true},
		{"no schema_version", "stage_bindings:\n  foo: bar\n", false},
		{"schema_version after stage_bindings", "stage_bindings:\n  foo: bar\nschema_version: 2\n", false},
		{"empty file", "", false},
		{"comment before schema_version", "# comment\nschema_version: 2\nstage_bindings:\n  foo: bar\n", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasSchemaVersion([]byte(tt.yaml))
			if got != tt.want {
				t.Errorf("hasSchemaVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInsertSchemaVersion(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			"inserts before stage_bindings",
			"stage_bindings:\n  designing:\n    description: test\n",
			"schema_version: 2\nstage_bindings:\n  designing:\n    description: test\n",
		},
		{
			"preserves comments",
			"# managed: true\nstage_bindings:\n  foo: bar\n",
			"# managed: true\nschema_version: 2\nstage_bindings:\n  foo: bar\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(insertSchemaVersion([]byte(tt.in)))
			if got != tt.want {
				t.Errorf("insertSchemaVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestRunMigrateStageBindings_AC005_AddsSchemaVersionPreservesContent tests AC-005 (REQ-003):
// Given a v1 file without schema_version, when kbz migrate stage-bindings runs,
// then the file gains schema_version: 2 and all other content is preserved.
func TestRunMigrateStageBindings_AC005_AddsSchemaVersionPreservesContent(t *testing.T) {
	dir := t.TempDir()
	kbzDir := filepath.Join(dir, ".kbz")
	if err := os.MkdirAll(kbzDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Use a v1 fixture with multiple stages to verify content preservation.
	v1YAML := "# comment line\nstage_bindings:\n  designing:\n    description: design stage\n    orchestration: single-agent\n    roles: [designer]\n    skills: [design-skill]\n  reviewing:\n    description: review stage\n    orchestration: single-agent\n    roles: [reviewer]\n    skills: [code-review]\n    human_gate: true\n"
	bindingPath := filepath.Join(kbzDir, "stage-bindings.yaml")
	if err := os.WriteFile(bindingPath, []byte(v1YAML), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer os.Chdir(origDir)
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	deps, output := testDependencies()
	if err := runMigrateStageBindings(nil, deps); err != nil {
		t.Fatalf("runMigrateStageBindings error = %v", err)
	}

	stdout := output.String()
	if !strings.Contains(stdout, "Added schema_version: 2") {
		t.Fatalf("expected 'Added schema_version: 2', got:\n%s", stdout)
	}

	// Verify file content.
	data, err := os.ReadFile(bindingPath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	content := string(data)

	// AC-005: schema_version must be present at top level.
	if !strings.Contains(content, "schema_version: 2") {
		t.Fatalf("file missing schema_version after migrate:\n%s", content)
	}

	// AC-005: schema_version must appear before stage_bindings.
	svIdx := strings.Index(content, "schema_version: 2")
	sbIdx := strings.Index(content, "stage_bindings:")
	if svIdx < 0 || sbIdx < 0 || svIdx >= sbIdx {
		t.Fatalf("schema_version must appear before stage_bindings:\n%s", content)
	}

	// AC-005: All other content must be preserved (comment, all stages, all fields).
	for _, want := range []string{
		"# comment line",
		"designing:",
		"description: design stage",
		"reviewing:",
		"description: review stage",
		"human_gate: true",
		"code-review",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("file missing expected content %q after migrate:\n%s", want, content)
		}
	}

	// AC-005: The original v1 content should not have been removed.
	if !strings.Contains(content, "stage_bindings:") {
		t.Fatal("file missing stage_bindings after migrate")
	}
}

// TestRunMigrateStageBindings_AC006_Idempotent tests AC-006 (REQ-003):
// Given a file with schema_version: 2, when kbz migrate stage-bindings runs
// again, then the file is unchanged.
func TestRunMigrateStageBindings_AC006_Idempotent(t *testing.T) {
	dir := t.TempDir()
	kbzDir := filepath.Join(dir, ".kbz")
	if err := os.MkdirAll(kbzDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// File already has schema_version: 2.
	v2YAML := "schema_version: 2\nstage_bindings:\n  designing:\n    description: design stage\n    orchestration: single-agent\n    roles: [designer]\n    skills: [design-skill]\n"
	bindingPath := filepath.Join(kbzDir, "stage-bindings.yaml")
	if err := os.WriteFile(bindingPath, []byte(v2YAML), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer os.Chdir(origDir)
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	// First run: should report already migrated.
	deps1, output1 := testDependencies()
	if err := runMigrateStageBindings(nil, deps1); err != nil {
		t.Fatalf("first runMigrateStageBindings error = %v", err)
	}
	if !strings.Contains(output1.String(), "already has schema_version") {
		t.Fatalf("expected 'already has schema_version', got:\n%s", output1.String())
	}

	// Read file after first run.
	data1, err := os.ReadFile(bindingPath)
	if err != nil {
		t.Fatalf("read after first run: %v", err)
	}

	// Second run: should also report already migrated.
	deps2, output2 := testDependencies()
	if err := runMigrateStageBindings(nil, deps2); err != nil {
		t.Fatalf("second runMigrateStageBindings error = %v", err)
	}
	if !strings.Contains(output2.String(), "already has schema_version") {
		t.Fatalf("expected 'already has schema_version' on second run, got:\n%s", output2.String())
	}

	// AC-006: File must be byte-for-byte identical after the second run.
	data2, err := os.ReadFile(bindingPath)
	if err != nil {
		t.Fatalf("read after second run: %v", err)
	}
	if string(data1) != string(data2) {
		t.Errorf("file changed between first and second idempotent runs:\nbefore: %q\nafter:  %q", string(data1), string(data2))
	}

	// AC-006: Three runs should also be idempotent.
	deps3, output3 := testDependencies()
	if err := runMigrateStageBindings(nil, deps3); err != nil {
		t.Fatalf("third runMigrateStageBindings error = %v", err)
	}
	if !strings.Contains(output3.String(), "already has schema_version") {
		t.Fatalf("expected 'already has schema_version' on third run, got:\n%s", output3.String())
	}
	data3, err := os.ReadFile(bindingPath)
	if err != nil {
		t.Fatalf("read after third run: %v", err)
	}
	if string(data1) != string(data3) {
		t.Errorf("file changed between first and third idempotent runs")
	}
}

func TestRunMigrate_UnknownSubcommand(t *testing.T) {
	deps, _ := testDependencies()
	err := runMigrate([]string{"unknown"}, deps)
	if err == nil {
		t.Fatal("expected error for unknown subcommand")
	}
	if !strings.Contains(err.Error(), "unknown migrate subcommand") {
		t.Fatalf("unexpected error: %v", err)
	}
}
