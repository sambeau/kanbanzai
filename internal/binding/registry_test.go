package binding

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const registryValidYAML = `stage_bindings:
  designing:
    description: "Design stage"
    orchestration: single-agent
    roles: [designer]
    skills: [design-skill]
    human_gate: false
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
  reviewing:
    description: "Review stage"
    orchestration: single-agent
    roles: [reviewer]
    skills: [code-review]
    human_gate: true
`

const registryInvalidStageYAML = `stage_bindings:
  totally-bogus:
    description: "Bad stage"
    orchestration: single-agent
    roles: [designer]
    skills: [design-skill]
`

const registryRoleFallbackYAML = `stage_bindings:
  designing:
    description: "Design stage"
    orchestration: single-agent
    roles: [designer-senior]
    skills: [design-skill]
    human_gate: false
`

func writeRegistryYAML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "stage-bindings.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}
	return path
}

func TestNewBindingRegistry(t *testing.T) {
	r := NewBindingRegistry("/some/path", nil)
	if r == nil {
		t.Fatal("expected non-nil registry")
	}
	if r.loaded {
		t.Error("new registry should not be loaded")
	}
}

func TestRegistryLoad(t *testing.T) {
	tests := []struct {
		name      string
		yaml      string
		wantErr   bool
		errSubstr string
	}{
		{
			name:    "valid file loads successfully",
			yaml:    registryValidYAML,
			wantErr: false,
		},
		{
			name:      "invalid stage name causes load error",
			yaml:      registryInvalidStageYAML,
			wantErr:   true,
			errSubstr: "invalid stage name",
		},
		{
			name: "invalid binding causes load error",
			yaml: `stage_bindings:
  designing:
    description: ""
    orchestration: single-agent
    roles: [designer]
    skills: [design-skill]
`,
			wantErr:   true,
			errSubstr: "description must not be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeRegistryYAML(t, tt.yaml)
			r := NewBindingRegistry(path, nil)
			err := r.Load()

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("expected error containing %q, got: %s", tt.errSubstr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestRegistryLoad_missing_file(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.yaml")
	r := NewBindingRegistry(path, nil)
	err := r.Load()
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "loading binding file") {
		t.Errorf("expected 'loading binding file' in error, got: %s", err)
	}
}

func TestRegistryLookup(t *testing.T) {
	path := writeRegistryYAML(t, registryValidYAML)
	r := NewBindingRegistry(path, nil)
	if err := r.Load(); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	tests := []struct {
		name      string
		stage     string
		wantErr   bool
		errSubstr string
	}{
		{
			name:    "existing stage returns binding",
			stage:   "designing",
			wantErr: false,
		},
		{
			name:    "another existing stage returns binding",
			stage:   "reviewing",
			wantErr: false,
		},
		{
			name:      "missing stage returns error",
			stage:     "nonexistent",
			wantErr:   true,
			errSubstr: `no binding for stage "nonexistent"`,
		},
		{
			name:      "empty string returns error",
			stage:     "",
			wantErr:   true,
			errSubstr: "must not be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sb, err := r.Lookup(tt.stage)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("expected error containing %q, got: %s", tt.errSubstr, err)
				}
				if sb != nil {
					t.Error("expected nil binding on error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sb == nil {
				t.Fatal("expected non-nil StageBinding")
			}
		})
	}
}

func TestRegistryLookup_returns_correct_binding(t *testing.T) {
	path := writeRegistryYAML(t, registryValidYAML)
	r := NewBindingRegistry(path, nil)
	if err := r.Load(); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	sb, err := r.Lookup("designing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sb.Description != "Design stage" {
		t.Errorf("description = %q, want %q", sb.Description, "Design stage")
	}
	if sb.Orchestration != "single-agent" {
		t.Errorf("orchestration = %q, want %q", sb.Orchestration, "single-agent")
	}
}

func TestRegistryLookup_before_load(t *testing.T) {
	r := NewBindingRegistry("/some/path", nil)
	_, err := r.Lookup("designing")
	if err == nil {
		t.Fatal("expected error for lookup before load")
	}
	if !strings.Contains(err.Error(), "registry not loaded") {
		t.Errorf("expected 'registry not loaded' in error, got: %s", err)
	}
}

func TestRegistryStages(t *testing.T) {
	path := writeRegistryYAML(t, registryValidYAML)
	r := NewBindingRegistry(path, nil)
	if err := r.Load(); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	stages := r.Stages()
	want := []string{"designing", "developing", "reviewing"}

	if len(stages) != len(want) {
		t.Fatalf("got %d stages, want %d: %v", len(stages), len(want), stages)
	}

	for i, s := range stages {
		if s != want[i] {
			t.Errorf("stages[%d] = %q, want %q", i, s, want[i])
		}
	}
}

func TestRegistryStages_sorted(t *testing.T) {
	path := writeRegistryYAML(t, registryValidYAML)
	r := NewBindingRegistry(path, nil)
	if err := r.Load(); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	stages := r.Stages()
	for i := 1; i < len(stages); i++ {
		if stages[i] < stages[i-1] {
			t.Errorf("stages not sorted: %v", stages)
			break
		}
	}
}

func TestRegistryStages_before_load(t *testing.T) {
	r := NewBindingRegistry("/some/path", nil)
	stages := r.Stages()
	if stages != nil {
		t.Errorf("expected nil stages before load, got %v", stages)
	}
}

func TestRegistryWarnings(t *testing.T) {
	knownRoles := map[string]bool{
		"designer": true,
	}
	rc := func(id string) bool { return knownRoles[id] }

	path := writeRegistryYAML(t, registryRoleFallbackYAML)
	r := NewBindingRegistry(path, rc)
	if err := r.Load(); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	warnings := r.Warnings()
	if len(warnings) == 0 {
		t.Fatal("expected warnings for role fallback")
	}

	found := false
	for _, w := range warnings {
		if strings.Contains(w, "designer-senior") && strings.Contains(w, "fallback") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected fallback warning for 'designer-senior', got: %v", warnings)
	}
}

func TestRegistryWarnings_no_role_checker(t *testing.T) {
	path := writeRegistryYAML(t, registryRoleFallbackYAML)
	r := NewBindingRegistry(path, nil)
	if err := r.Load(); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	warnings := r.Warnings()
	if len(warnings) != 0 {
		t.Errorf("expected no warnings with nil roleChecker, got: %v", warnings)
	}
}

func TestRegistryWarnings_before_load(t *testing.T) {
	r := NewBindingRegistry("/some/path", nil)
	warnings := r.Warnings()
	if warnings != nil {
		t.Errorf("expected nil warnings before load, got %v", warnings)
	}
}
