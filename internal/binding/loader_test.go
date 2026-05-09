package binding

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeYAML(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "stage-bindings.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}
	return path
}

const validMinimalYAML = `schema_version: 2
stage_bindings:
  designing:
    description: "Design stage"
    orchestration: single-agent
    roles: [designer]
    skills: [design-skill]
    human_gate: false
`

const validMultiStageYAML = `schema_version: 2
stage_bindings:
  designing:
    description: "Design stage"
    orchestration: single-agent
    roles: [designer]
    skills: [design-skill]
    human_gate: false
  reviewing:
    description: "Review stage"
    orchestration: single-agent
    roles: [reviewer]
    skills: [code-review]
    human_gate: true
  developing:
    description: "Dev stage"
    orchestration: orchestrator-workers
    roles: [lead]
    skills: [implementation]
    human_gate: false
    sub_agents:
      roles: [backend]
      skills: [coding]
      topology: parallel
`

// v1MinimalYAML is a v1 fixture without schema_version, used for legacy decode tests.
const v1MinimalYAML = `stage_bindings:
  designing:
    description: "Design stage"
    orchestration: single-agent
    roles: [designer]
    skills: [design-skill]
    human_gate: false
`

func TestLoadBindingFile(t *testing.T) {
	tests := []struct {
		name       string
		yaml       string // YAML content; empty string means use a missing file path
		wantOK     bool
		wantErrors []string // substrings expected in error messages
		wantStages []string // expected stage keys when wantOK is true
	}{
		{
			name:       "valid minimal file",
			yaml:       validMinimalYAML,
			wantOK:     true,
			wantStages: []string{"designing"},
		},
		{
			name:       "valid file with multiple stages",
			yaml:       validMultiStageYAML,
			wantOK:     true,
			wantStages: []string{"designing", "reviewing", "developing"},
		},
		{
			name:       "missing file returns error",
			yaml:       "", // signals: don't write a file
			wantErrors: []string{"reading binding file"},
		},
		{
			name: "missing stage_bindings key",
			yaml: `schema_version: 2
something_else:
  foo: bar
`,
			wantErrors: []string{"missing required 'stage_bindings' key"},
		},
		{
			name: "stage_bindings is a sequence not mapping",
			yaml: `schema_version: 2
stage_bindings:
  - designing
  - reviewing
`,
			wantErrors: []string{"stage_bindings must be a mapping"},
		},
		{
			name: "duplicate stage key detected",
			yaml: `schema_version: 2
stage_bindings:
  designing:
    description: "First"
    orchestration: single-agent
    roles: [designer]
    skills: [design-skill]
  designing:
    description: "Duplicate"
    orchestration: single-agent
    roles: [designer]
    skills: [design-skill]
`,
			wantErrors: []string{"duplicate stage key: designing"},
		},
		{
			name: "unknown top-level key rejected",
			yaml: `schema_version: 2
stage_bindings:
  designing:
    description: "Design stage"
    orchestration: single-agent
    roles: [designer]
    skills: [design-skill]
extra_key: true
`,
			wantErrors: []string{"unknown top-level key"},
		},
		{
			name: "empty stage_bindings mapping loads with zero bindings",
			yaml: `schema_version: 2
stage_bindings: {}
`,
			wantOK:     true,
			wantStages: nil,
		},
		{
			name: "unknown field within a binding entry rejected (AC-004)",
			yaml: `schema_version: 2
stage_bindings:
  designing:
    description: "Design stage"
    orchestration: single-agent
    roles: [designer]
    skills: [design-skill]
    totally_bogus_field: true
`,
			wantErrors: []string{"decoding binding file"},
		},
		// REQ-002 tests
		{
			name: "AC-002: valid v2 file decodes successfully",
			yaml: `schema_version: 2
stage_bindings:
  designing:
    description: "Design stage"
    orchestration: single-agent
    roles: [designer]
    skills: [design-skill]
    human_gate: false
`,
			wantOK:     true,
			wantStages: []string{"designing"},
		},
		{
			name: "AC-003: unsupported schema_version 99 returns structured error",
			yaml: `schema_version: 99
stage_bindings:
  designing:
    description: "Design stage"
    orchestration: single-agent
    roles: [designer]
    skills: [design-skill]
`,
			wantErrors: []string{"unsupported schema_version", "99", "2"},
		},
		{
			name: "schema_version with future version 3 is unsupported",
			yaml: `schema_version: 3
stage_bindings:
  designing:
    description: "Design stage"
    orchestration: single-agent
    roles: [designer]
    skills: [design-skill]
`,
			wantErrors: []string{"unsupported schema_version", "3", "2"},
		},
		{
			name: "schema_version present but not integer",
			yaml: `schema_version: "two"
stage_bindings:
  designing:
    description: "Design stage"
    orchestration: single-agent
    roles: [designer]
    skills: [design-skill]
`,
			wantErrors: []string{"schema_version must be an integer"},
		},
		{
			name:       "file without schema_version still loads (backward compat)",
			yaml: `stage_bindings:
  designing:
    description: "Design stage"
    orchestration: single-agent
    roles: [designer]
    skills: [design-skill]
    human_gate: false
`,
			wantOK:     true,
			wantStages: []string{"designing"},
		},
		{
			name:       "schema_version 0 loads as backward compat",
			yaml: `schema_version: 0
stage_bindings:
  designing:
    description: "Design stage"
    orchestration: single-agent
    roles: [designer]
    skills: [design-skill]
    human_gate: false
`,
			wantOK:     true,
			wantStages: []string{"designing"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var path string
			if tt.yaml == "" {
				// Use a path that does not exist.
				path = filepath.Join(t.TempDir(), "nonexistent.yaml")
			} else {
				path = writeYAML(t, t.TempDir(), tt.yaml)
			}

			bf, errs := LoadBindingFile(path)

			if tt.wantOK {
				if len(errs) != 0 {
					t.Errorf("expected no errors, got %d:", len(errs))
					for _, e := range errs {
						t.Errorf("  %s", e)
					}
					return
				}
				if bf == nil {
					t.Fatal("expected non-nil BindingFile")
				}
				if len(bf.StageBindings) != len(tt.wantStages) {
					t.Errorf("expected %d stages, got %d", len(tt.wantStages), len(bf.StageBindings))
				}
				for _, stage := range tt.wantStages {
					if _, ok := bf.StageBindings[stage]; !ok {
						t.Errorf("expected stage %q in bindings", stage)
					}
				}
				return
			}

			// Expect errors.
			if len(errs) == 0 {
				t.Fatal("expected errors, got none")
			}
			if bf != nil {
				t.Error("expected nil BindingFile on error")
			}

			errStrings := make([]string, len(errs))
			for i, e := range errs {
				errStrings[i] = e.Error()
			}
			joined := strings.Join(errStrings, "\n")

			for _, want := range tt.wantErrors {
				if !strings.Contains(joined, want) {
					t.Errorf("expected error containing %q, got:\n%s", want, joined)
				}
			}
		})
	}
}

func TestLoadBindingFile_valid_bindings_have_correct_fields(t *testing.T) {
	yaml := `schema_version: 2
stage_bindings:
  designing:
    description: "Design stage"
    orchestration: single-agent
    roles: [designer, lead]
    skills: [design-skill]
    human_gate: true
    document_type: design
    notes: "some notes"
`
	path := writeYAML(t, t.TempDir(), yaml)
	bf, errs := LoadBindingFile(path)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	sb := bf.StageBindings["designing"]
	if sb == nil {
		t.Fatal("expected 'designing' stage binding")
	}
	if sb.Description != "Design stage" {
		t.Errorf("description = %q, want %q", sb.Description, "Design stage")
	}
	if sb.Orchestration != "single-agent" {
		t.Errorf("orchestration = %q, want %q", sb.Orchestration, "single-agent")
	}
	if len(sb.Roles) != 2 || sb.Roles[0] != "designer" || sb.Roles[1] != "lead" {
		t.Errorf("roles = %v, want [designer lead]", sb.Roles)
	}
	if !sb.HumanGate {
		t.Error("human_gate = false, want true")
	}
	if sb.DocumentType == nil || *sb.DocumentType != "design" {
		t.Errorf("document_type = %v, want \"design\"", sb.DocumentType)
	}
	if sb.Notes != "some notes" {
		t.Errorf("notes = %q, want %q", sb.Notes, "some notes")
	}
	if bf.SchemaVersion != 2 {
		t.Errorf("SchemaVersion = %d, want 2", bf.SchemaVersion)
	}
}

func TestErrUnsupportedSchemaVersion(t *testing.T) {
	// AC-003: verify that the returned error wraps ErrUnsupportedSchemaVersion
	// so callers can use errors.Is to detect unsupported version errors.
	yaml := `schema_version: 99
stage_bindings:
  designing:
    description: "Design stage"
    orchestration: single-agent
    roles: [designer]
    skills: [design-skill]
`
	path := writeYAML(t, t.TempDir(), yaml)
	_, errs := LoadBindingFile(path)
	if len(errs) != 1 {
		t.Fatalf("expected exactly 1 error, got %d", len(errs))
	}
	if !errors.Is(errs[0], ErrUnsupportedSchemaVersion) {
		t.Errorf("error does not wrap ErrUnsupportedSchemaVersion: %v", errs[0])
	}
	if !strings.Contains(errs[0].Error(), "unsupported schema_version") {
		t.Errorf("error does not contain 'unsupported schema_version': %v", errs[0])
	}
	if !strings.Contains(errs[0].Error(), "99") {
		t.Errorf("error does not contain version '99': %v", errs[0])
	}
}

// TestLoadBindingFile_AC001_SchemaVersionPresent tests AC-001 (REQ-001):
// Given stage-bindings.yaml, when inspected via LoadBindingFile, then
// schema_version: 2 is present at the top level.
func TestLoadBindingFile_AC001_SchemaVersionPresent(t *testing.T) {
	yaml := `schema_version: 2
stage_bindings:
  designing:
    description: "Design stage"
    orchestration: single-agent
    roles: [designer]
    skills: [design-skill]
    human_gate: false
`
	path := writeYAML(t, t.TempDir(), yaml)
	bf, errs := LoadBindingFile(path)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if bf.SchemaVersion != 2 {
		t.Errorf("SchemaVersion = %d, want 2", bf.SchemaVersion)
	}
}

// TestDecodeBindingFileLegacy tests AC-009 (REQ-005): older binaries without
// schema_version support must refuse v2 files with a clear message.
func TestDecodeBindingFileLegacy(t *testing.T) {
	tests := []struct {
		name       string
		yaml       string
		wantOK     bool
		wantErrors []string
	}{
		{
			name:   "v1 file (no schema_version) loads fine",
			yaml:   v1MinimalYAML,
			wantOK: true,
		},
		{
			name:       "v2 file rejected with clear version-mismatch error",
			yaml:       validMinimalYAML,
			wantErrors: []string{"binding version mismatch", "schema_version 2", "upgrade"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bf, err := DecodeBindingFileLegacy([]byte(tt.yaml))

			if tt.wantOK {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
					return
				}
				if bf == nil {
					t.Fatal("expected non-nil BindingFile")
				}
				return
			}

			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if bf != nil {
				t.Error("expected nil BindingFile on error")
			}

			errStr := err.Error()
			for _, want := range tt.wantErrors {
				if !strings.Contains(errStr, want) {
					t.Errorf("expected error containing %q, got: %s", want, errStr)
				}
			}
		})
	}
}

// TestDecodeBindingFileLegacy_errorsIs tests that DecodeBindingFileLegacy errors
// wrap ErrBindingVersionMismatch for programmatic detection.
func TestDecodeBindingFileLegacy_errorsIs(t *testing.T) {
	_, err := DecodeBindingFileLegacy([]byte(validMinimalYAML))
	if err == nil {
		t.Fatal("expected error for v2 file, got nil")
	}
	if !errors.Is(err, ErrBindingVersionMismatch) {
		t.Errorf("error does not wrap ErrBindingVersionMismatch: %v", err)
	}
}

// TestDecodeBindingFileLegacy_v1WithUnknownField ensures the legacy decoder
// still rejects unknown fields via strict YAML decoding, preserving the
// pre-REQ-002 typo detection behavior.
func TestDecodeBindingFileLegacy_v1WithUnknownField(t *testing.T) {
	yaml := `stage_bindings:
  designing:
    description: "Design stage"
    orchestration: single-agent
    roles: [designer]
    skills: [design-skill]
    totally_bogus: true
`
	_, err := DecodeBindingFileLegacy([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for unknown field, got nil")
	}
	if !strings.Contains(err.Error(), "decoding binding file") {
		t.Errorf("expected 'decoding binding file' in error, got: %s", err)
	}
}
