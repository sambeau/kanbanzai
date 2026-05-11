package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/kanbanzai/internal/binding"
)

func TestRunBinding_MissingSubcommand_ReturnsUsageError(t *testing.T) {
	deps, _ := testDependencies()

	err := run([]string{"binding"}, deps)
	if err == nil {
		t.Fatal("run(binding) error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "missing binding subcommand") {
		t.Fatalf("error missing subcommand message: %v", err)
	}
}

func TestRunBinding_UnknownSubcommand_ReturnsUsageError(t *testing.T) {
	deps, _ := testDependencies()

	err := run([]string{"binding", "unknown"}, deps)
	if err == nil {
		t.Fatal("run(binding unknown) error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "unknown binding subcommand") {
		t.Fatalf("error unknown subcommand message: %v", err)
	}
}

func TestRunBindingDoctor_ValidBindingFile(t *testing.T) {
	dir := t.TempDir()
	createTestBindingEnv(t, dir, validBindingYAML)
	bindingFile := filepath.Join(dir, ".kbz", "stage-bindings.yaml")

	var stdout bytes.Buffer
	deps := dependencies{stdout: &stdout}

	err := runBindingDoctor([]string{"--file", bindingFile}, deps)
	if err != nil {
		t.Fatalf("runBindingDoctor error = %v\noutput:\n%s", err, stdout.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "valid") {
		t.Errorf("expected 'valid' in output, got:\n%s", out)
	}
}

func TestRunBindingDoctor_InvalidStageName(t *testing.T) {
	dir := t.TempDir()
	createTestBindingEnv(t, dir, invalidBindingYAML)
	bindingFile := filepath.Join(dir, ".kbz", "stage-bindings.yaml")

	var stdout bytes.Buffer
	deps := dependencies{stdout: &stdout}

	err := runBindingDoctor([]string{"--file", bindingFile}, deps)
	if err == nil {
		t.Fatal("runBindingDoctor with invalid file: error = nil, want non-nil")
	}

	out := stdout.String()
	if !strings.Contains(out, "invalid stage name") {
		t.Errorf("expected 'invalid stage name' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "cooking") {
		t.Errorf("expected 'cooking' in output, got:\n%s", out)
	}
}

func TestRunBindingDoctor_MissingRolesField(t *testing.T) {
	dir := t.TempDir()
	createTestBindingEnv(t, dir, missingRolesYAML)
	bindingFile := filepath.Join(dir, ".kbz", "stage-bindings.yaml")

	var stdout bytes.Buffer
	deps := dependencies{stdout: &stdout}

	err := runBindingDoctor([]string{"--file", bindingFile}, deps)
	if err == nil {
		t.Fatal("runBindingDoctor with missing roles: error = nil, want non-nil")
	}

	out := stdout.String()
	if !strings.Contains(out, "roles must not be empty") {
		t.Errorf("expected 'roles must not be empty' in output, got:\n%s", out)
	}
}

func TestRunBindingDoctor_MissingFile(t *testing.T) {
	bindingFile := filepath.Join(t.TempDir(), "nonexistent", "stage-bindings.yaml")

	var stdout bytes.Buffer
	deps := dependencies{stdout: &stdout}

	err := runBindingDoctor([]string{"--file", bindingFile}, deps)
	if err == nil {
		t.Fatal("runBindingDoctor with missing file: error = nil, want non-nil")
	}

	out := stdout.String()
	if !strings.Contains(err.Error(), "load failed") || !strings.Contains(out, "ERROR") {
		t.Errorf("expected load failure errors, got: %v\noutput:\n%s", err, out)
	}
}

func TestRunBindingDoctor_RoleFallbackWarning(t *testing.T) {
	dir := t.TempDir()
	createTestBindingEnv(t, dir, roleWithFallbackYAML)

	// Create the fallback role file (reviewer.yaml) but not reviewer-security.yaml.
	os.WriteFile(filepath.Join(dir, ".kbz", "roles", "reviewer.yaml"), []byte("name: reviewer\n"), 0o644)

	bindingFile := filepath.Join(dir, ".kbz", "stage-bindings.yaml")

	var stdout bytes.Buffer
	deps := dependencies{stdout: &stdout}

	err := runBindingDoctor([]string{"--file", bindingFile}, deps)
	if err != nil {
		t.Fatalf("runBindingDoctor with fallback role: error = %v\noutput:\n%s", err, stdout.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "resolved via fallback") {
		t.Errorf("expected fallback warning in output, got:\n%s", out)
	}
}

func TestRunBindingDoctor_NoRolesDir(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".kbz"), 0o755)
	os.WriteFile(filepath.Join(dir, ".kbz", "stage-bindings.yaml"), []byte(validBindingYAML), 0o644)
	bindingFile := filepath.Join(dir, ".kbz", "stage-bindings.yaml")

	var stdout bytes.Buffer
	deps := dependencies{stdout: &stdout}

	// Should still work — role checker just won't find roles and will report warnings.
	err := runBindingDoctor([]string{"--file", bindingFile}, deps)
	if err != nil {
		t.Fatalf("runBindingDoctor without roles dir: error = %v\noutput:\n%s", err, stdout.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "not found in stage") {
		t.Errorf("expected role-not-found warnings in output, got:\n%s", out)
	}
}

func TestBindingDoctorUsageText(t *testing.T) {
	if !strings.Contains(bindingDoctorUsageText, "kbz binding doctor") {
		t.Error("usage text should mention 'kbz binding doctor'")
	}
	if !strings.Contains(bindingDoctorUsageText, "Exit codes") {
		t.Error("usage text should mention exit codes")
	}
}

func TestPrintBindingDoctorResults_NoErrorsOrWarnings(t *testing.T) {
	var buf bytes.Buffer
	result := &binding.ValidationResult{}
	printBindingDoctorResults(&buf, result)

	out := buf.String()
	if !strings.Contains(out, "valid") {
		t.Errorf("expected 'valid' in output, got: %s", out)
	}
}

func TestPrintBindingDoctorResults_ErrorsOnly(t *testing.T) {
	var buf bytes.Buffer
	result := &binding.ValidationResult{
		Errors: []error{
			&testError{message: "stage \"frying\": roles must not be empty"},
			&testError{message: "stage \"cooking\": invalid stage name"},
		},
	}
	printBindingDoctorResults(&buf, result)

	out := buf.String()
	if !strings.Contains(out, "Validation errors: 2") {
		t.Errorf("expected error count, got: %s", out)
	}
	if !strings.Contains(out, "roles must not be empty") {
		t.Errorf("expected first error, got: %s", out)
	}
	if !strings.Contains(out, "invalid stage name") {
		t.Errorf("expected second error, got: %s", out)
	}
}

func TestPrintBindingDoctorResults_WarningsOnly(t *testing.T) {
	var buf bytes.Buffer
	result := &binding.ValidationResult{
		Warnings: []string{
			"role \"ghost\" not found in stage \"designing\"",
		},
	}
	printBindingDoctorResults(&buf, result)

	out := buf.String()
	if !strings.Contains(out, "Validation warnings: 1") {
		t.Errorf("expected warning count, got: %s", out)
	}
	if !strings.Contains(out, "passed validation") {
		t.Errorf("expected 'passed validation', got: %s", out)
	}
}

func TestPrintBindingDoctorResults_ErrorsAndWarnings(t *testing.T) {
	var buf bytes.Buffer
	result := &binding.ValidationResult{
		Errors: []error{
			&testError{message: "bad"},
		},
		Warnings: []string{
			"role not found",
		},
	}
	printBindingDoctorResults(&buf, result)

	out := buf.String()
	if !strings.Contains(out, "ERROR") {
		t.Errorf("expected ERROR in output, got: %s", out)
	}
	if !strings.Contains(out, "WARN") {
		t.Errorf("expected WARN in output, got: %s", out)
	}
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func createTestBindingEnv(t *testing.T, dir string, bindingYAML string) {
	t.Helper()

	kbzDir := filepath.Join(dir, ".kbz")
	rolesDir := filepath.Join(kbzDir, "roles")
	os.MkdirAll(rolesDir, 0o755)

	roleFiles := map[string]string{
		"architect":                 "name: architect\n",
		"spec-author":               "name: spec-author\n",
		"orchestrator":              "name: orchestrator\n",
		"researcher":                "name: researcher\n",
		"documenter":                "name: documenter\n",
		"reviewer-conformance":      "name: reviewer-conformance\n",
		"doc-pipeline-orchestrator": "name: doc-pipeline-orchestrator\n",
		"verifier":                  "name: verifier\n",
		"implementer":               "name: implementer\n",
	}
	for name, content := range roleFiles {
		os.WriteFile(filepath.Join(rolesDir, name+".yaml"), []byte(content), 0o644)
	}

	os.WriteFile(filepath.Join(kbzDir, "stage-bindings.yaml"), []byte(bindingYAML), 0o644)
}

// Valid binding YAML — only uses stages currently in the validStages allowlist.
const validBindingYAML = `schema_version: 2
stage_bindings:
  designing:
    description: "Creating or revising a design document"
    orchestration: single-agent
    roles: [architect]
    skills: [write-design]
  specifying:
    description: "Writing a formal specification"
    orchestration: single-agent
    roles: [spec-author]
    skills: [write-spec]
  dev-planning:
    description: "Breaking a spec into an implementation plan"
    orchestration: single-agent
    roles: [architect]
    skills: [write-dev-plan, decompose-feature]
  developing:
    description: "Implementing tasks"
    orchestration: orchestrator-workers
    roles: [orchestrator]
    skills: [orchestrate-development]
    sub_agents:
      roles: [implementer]
      skills: [implement-task]
      topology: parallel
  reviewing:
    description: "Evaluating implementation"
    orchestration: orchestrator-workers
    roles: [orchestrator]
    skills: [orchestrate-review]
    sub_agents:
      roles: [reviewer-conformance]
      skills: [review-code]
      topology: parallel
  researching:
    description: "Producing a research report"
    orchestration: single-agent
    roles: [researcher]
    skills: [write-research]
  documenting:
    description: "Updating project documentation"
    orchestration: single-agent
    roles: [documenter]
    skills: [update-docs]
`

// Invalid binding YAML — uses a bogus stage name "cooking".
const invalidBindingYAML = `schema_version: 2
stage_bindings:
  designing:
    description: "Creating or revising a design document"
    orchestration: single-agent
    roles: [architect]
    skills: [write-design]
  cooking:
    description: "Not a real stage"
    orchestration: single-agent
    roles: [chef]
    skills: [cooking-skill]
`

// Missing roles field — "developing" stage without roles.
const missingRolesYAML = `schema_version: 2
stage_bindings:
  designing:
    description: "Creating or revising a design document"
    orchestration: single-agent
    roles: [architect]
    skills: [write-design]
  developing:
    description: "Implementing tasks"
    orchestration: orchestrator-workers
    roles: []
    skills: [orchestrate-development]
    sub_agents:
      roles: [implementer]
      skills: [implement-task]
      topology: parallel
`

// Role with fallback — reviewer-security with only reviewer.yaml on disk.
const roleWithFallbackYAML = `schema_version: 2
stage_bindings:
  designing:
    description: "Creating or revising a design document"
    orchestration: single-agent
    roles: [architect]
    skills: [write-design]
  reviewing:
    description: "Evaluating implementation"
    orchestration: single-agent
    roles: [reviewer-security]
    skills: [review-code]
`
