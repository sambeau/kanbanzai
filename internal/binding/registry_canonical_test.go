package binding

import (
	"strings"
	"testing"
)

// TestRegistryLoad_CanonicalStructure verifies that BindingRegistry.Load
// succeeds against a YAML file with the same stage keys as the canonical
// .kbz/stage-bindings.yaml (all 12 stages, including the three added in
// Phase 1: merging, verifying, retro-fixing).
//
// AC-001: server starts normally with a valid stage-bindings.yaml.
// AC-003: ValidateBindingFile succeeds against the canonical file.
func TestRegistryLoad_CanonicalStructure(t *testing.T) {
	canonicalYAML := `schema_version: 2
stage_bindings:
  designing:
    description: "Design stage"
    orchestration: single-agent
    roles: [architect]
    skills: [write-design]
    human_gate: false
  specifying:
    description: "Spec stage"
    orchestration: single-agent
    roles: [spec-author]
    skills: [write-spec]
    human_gate: true
  dev-planning:
    description: "Dev planning stage"
    orchestration: single-agent
    roles: [architect]
    skills: [write-dev-plan]
    human_gate: true
  developing:
    description: "Dev stage"
    orchestration: orchestrator-workers
    roles: [orchestrator]
    skills: [orchestrate-development]
    human_gate: false
    sub_agents:
      roles: [implementer]
      skills: [implement-task]
      topology: parallel
  reviewing:
    description: "Review stage"
    orchestration: orchestrator-workers
    roles: [orchestrator]
    skills: [orchestrate-review]
    human_gate: true
    sub_agents:
      roles: [reviewer-conformance]
      skills: [review-code]
      topology: parallel
  merging:
    description: "Merge stage"
    orchestration: single-agent
    roles: [orchestrator]
    skills: [orchestrate-review]
    human_gate: false
  verifying:
    description: "Verify stage"
    orchestration: single-agent
    roles: [orchestrator]
    skills: [orchestrate-review]
    human_gate: false
    sub_agents:
      roles: [verifier]
      skills: [verify-closeout]
      topology: single
  batch-reviewing:
    description: "Batch review stage"
    orchestration: single-agent
    roles: [reviewer-conformance]
    skills: [review-plan]
    human_gate: true
  researching:
    description: "Research stage"
    orchestration: single-agent
    roles: [researcher]
    skills: [write-research]
    human_gate: false
  documenting:
    description: "Document stage"
    orchestration: single-agent
    roles: [documenter]
    skills: [update-docs]
    human_gate: false
  doc-publishing:
    description: "Doc publishing stage"
    orchestration: pipeline-coordinator
    roles: [doc-pipeline-orchestrator]
    skills: [orchestrate-doc-pipeline]
    human_gate: false
    sub_agents:
      roles: [doc-editor]
      skills: [edit-docs]
      topology: sequential
  retro-fixing:
    description: "Retro fix stage"
    orchestration: single-agent
    roles: [orchestrator]
    skills: [orchestrate-development]
    human_gate: false
`
	path := writeRegistryYAML(t, canonicalYAML)
	reg := NewBindingRegistry(path, nil)
	if err := reg.Load(); err != nil {
		t.Fatalf("canonical structure YAML failed validation: %v", err)
	}

	// Verify all 12 stages are present.
	stages := reg.Stages()
	wantStages := []string{
		"batch-reviewing", "designing", "dev-planning", "developing",
		"doc-publishing", "documenting", "merging", "researching",
		"retro-fixing", "reviewing", "specifying", "verifying",
	}
	if len(stages) != len(wantStages) {
		t.Errorf("got %d stages, want %d: %v", len(stages), len(wantStages), stages)
	}
	for i, s := range stages {
		if s != wantStages[i] {
			t.Errorf("stages[%d] = %q, want %q", i, s, wantStages[i])
		}
	}
}

// TestRegistryLoad_InvalidStageName verifies that an invalid stage name
// produces a validation error.
//
// AC-004: plan-reviewing produces "invalid stage name" error.
func TestRegistryLoad_InvalidStageName(t *testing.T) {
	invalidYAML := `schema_version: 2
stage_bindings:
  plan-reviewing:
    description: "Stale stage"
    orchestration: single-agent
    roles: [architect]
    skills: [write-design]
`
	path := writeRegistryYAML(t, invalidYAML)
	reg := NewBindingRegistry(path, nil)
	err := reg.Load()
	if err == nil {
		t.Fatal("expected error for plan-reviewing stage")
	}
	if !strings.Contains(err.Error(), `invalid stage name "plan-reviewing"`) {
		t.Errorf("expected 'invalid stage name \"plan-reviewing\"' in error, got: %s", err)
	}
}

// TestRegistryLoad_ValidatesRetroFixing verifies that the retro-fixing
// binding with orchestration, roles, and skills passes validation.
//
// AC-008: ValidateBindingFile succeeds for retro-fixing with passthrough fields.
func TestRegistryLoad_ValidatesRetroFixing(t *testing.T) {
	retroFixingYAML := `schema_version: 2
stage_bindings:
  retro-fixing:
    description: "Implementing a fix for a retrospective theme"
    orchestration: single-agent
    roles: [orchestrator]
    skills: [orchestrate-development]
`
	path := writeRegistryYAML(t, retroFixingYAML)
	reg := NewBindingRegistry(path, nil)
	if err := reg.Load(); err != nil {
		t.Fatalf("retro-fixing binding failed validation: %v", err)
	}
}
