package binding

import (
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

const validMinimalYAML = `stage_bindings:
  designing:
    description: "Design stage"
    orchestration: single-agent
    roles: [designer]
    skills: [design-skill]
    human_gate: false
`

const validMultiStageYAML = `stage_bindings:
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
			yaml: `something_else:
  foo: bar
`,
			wantErrors: []string{"missing required 'stage_bindings' key"},
		},
		{
			name: "stage_bindings is a sequence not mapping",
			yaml: `stage_bindings:
  - designing
  - reviewing
`,
			wantErrors: []string{"stage_bindings must be a mapping"},
		},
		{
			name: "duplicate stage key detected",
			yaml: `stage_bindings:
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
			yaml: `stage_bindings:
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
			yaml: `stage_bindings: {}
`,
			wantOK:     true,
			wantStages: nil,
		},
		{
			name: "unknown field within a binding entry rejected",
			yaml: `stage_bindings:
  designing:
    description: "Design stage"
    orchestration: single-agent
    roles: [designer]
    skills: [design-skill]
    totally_bogus_field: true
`,
			wantErrors: []string{"decoding binding file"},
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
	yaml := `stage_bindings:
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
}
