package context

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/kanbanzai/internal/skill"
)

// TestStepResolveToolHint_ExactMatch verifies that an exact role ID match
// resolves the hint without walking inheritance (AC-010).
func TestStepResolveToolHint_ExactMatch(t *testing.T) {
	p := &Pipeline{
		MergedToolHints: map[string]string{
			"implementer-go": "Use search_graph for code navigation",
		},
	}
	state := &PipelineState{
		Role: &ResolvedRole{ID: "implementer-go"},
	}
	p.stepResolveToolHint(state)
	if state.ToolHint != "Use search_graph for code navigation" {
		t.Errorf("expected exact match hint, got %q", state.ToolHint)
	}
}

// TestStepResolveToolHint_Inherited verifies inheritance walking (AC-010).
func TestStepResolveToolHint_Inherited(t *testing.T) {
	dir := t.TempDir()
	toolHintWriteRole(t, dir, "implementer-go", "implementer")
	toolHintWriteRole(t, dir, "implementer", "")
	store := NewRoleStore(dir, "")

	p := &Pipeline{
		MergedToolHints:   map[string]string{"implementer": "Use grep for searching"},
		ToolHintRoleStore: store,
		Roles:             &RoleStoreAdapter{Store: store},
	}
	state := &PipelineState{
		Role: &ResolvedRole{ID: "implementer-go"},
	}
	p.stepResolveToolHint(state)
	if state.ToolHint != "Use grep for searching" {
		t.Errorf("expected inherited hint, got %q", state.ToolHint)
	}
}

// TestStepResolveToolHint_NoMatch verifies that unrelated roles get no hint (AC-011).
func TestStepResolveToolHint_NoMatch(t *testing.T) {
	dir := t.TempDir()
	toolHintWriteRole(t, dir, "implementer-go", "implementer")
	toolHintWriteRole(t, dir, "implementer", "")
	store := NewRoleStore(dir, "")

	p := &Pipeline{
		MergedToolHints:   map[string]string{"researcher": "Use get_architecture"},
		ToolHintRoleStore: store,
		Roles:             &RoleStoreAdapter{Store: store},
	}
	state := &PipelineState{
		Role: &ResolvedRole{ID: "implementer-go"},
	}
	p.stepResolveToolHint(state)
	if state.ToolHint != "" {
		t.Errorf("expected no hint, got %q", state.ToolHint)
	}
}

// TestStepResolveToolHint_NilHints verifies no panic on nil hints map.
func TestStepResolveToolHint_NilHints(t *testing.T) {
	p := &Pipeline{}
	state := &PipelineState{
		Role: &ResolvedRole{ID: "implementer-go"},
	}
	p.stepResolveToolHint(state)
	if state.ToolHint != "" {
		t.Errorf("expected no hint with nil map, got %q", state.ToolHint)
	}
}

// TestStepAssembleSections_AvailableTools verifies the ## Available Tools section
// is rendered when a tool hint is set (AC-010).
func TestStepAssembleSections_AvailableTools(t *testing.T) {
	p := &Pipeline{
		Roles:    &mockRoleResolver{},
		Skills:   &mockSkillResolver{},
		Bindings: &mockBindingResolver{},
	}
	state := &PipelineState{
		Input: PipelineInput{
			TaskState: map[string]any{"id": "TASK-123", "summary": "test task"},
		},
		Role:     &ResolvedRole{ID: "implementer-go", Identity: "Go implementer"},
		ToolHint: "Use search_graph and trace_call_path for code navigation.",
	}
	p.stepAssembleSections(state)

	var found bool
	for _, s := range state.Sections {
		if s.Label == "Available Tools" {
			found = true
			if s.Position != PositionAvailableTools {
				t.Errorf("expected position %d, got %d", PositionAvailableTools, s.Position)
			}
			if !strings.Contains(s.Content, "## Available Tools") {
				t.Errorf("expected ## Available Tools header, got %q", s.Content)
			}
			if !strings.Contains(s.Content, "search_graph") {
				t.Errorf("expected hint content, got %q", s.Content)
			}
		}
	}
	if !found {
		t.Error("Available Tools section not found in assembled sections")
	}
}

// TestStepAssembleSections_NoToolHint verifies no section when hint is empty (AC-011).
func TestStepAssembleSections_NoToolHint(t *testing.T) {
	p := &Pipeline{
		Roles:    &mockRoleResolver{},
		Skills:   &mockSkillResolver{},
		Bindings: &mockBindingResolver{},
	}
	state := &PipelineState{
		Input: PipelineInput{
			TaskState: map[string]any{"id": "TASK-123", "summary": "test task"},
		},
		Role: &ResolvedRole{ID: "implementer-go", Identity: "Go implementer"},
	}
	p.stepAssembleSections(state)

	for _, s := range state.Sections {
		if s.Label == "Available Tools" {
			t.Error("Available Tools section should not be present when hint is empty")
		}
	}
}

// TestStepAssembleSections_ToolHintsBeforeProcedure verifies section ordering (FR-011, FR-017).
func TestStepAssembleSections_ToolHintsBeforeProcedure(t *testing.T) {
	if PositionAvailableTools >= PositionProcedure {
		t.Errorf("PositionAvailableTools (%d) must be < PositionProcedure (%d)",
			PositionAvailableTools, PositionProcedure)
	}
}

// TestStepAssembleSections_ToolHintBeforeProcedure_Behavioral verifies that when
// both a tool hint and a skill with a Procedure section are present, the
// "Available Tools" section has a lower Position than "Procedure" in the
// assembled output (AC-016).
func TestStepAssembleSections_ToolHintBeforeProcedure_Behavioral(t *testing.T) {
	p := &Pipeline{
		Roles:    &mockRoleResolver{},
		Skills:   &mockSkillResolver{},
		Bindings: &mockBindingResolver{},
	}
	state := &PipelineState{
		Input: PipelineInput{
			TaskState: map[string]any{"id": "TASK-123", "summary": "test task"},
		},
		Role:     &ResolvedRole{ID: "implementer-go", Identity: "Go implementer"},
		ToolHint: "Use search_graph",
		Skill: &skill.Skill{
			Sections: []skill.BodySection{
				{Heading: "Procedure", Content: "Do the thing.\n"},
			},
		},
	}
	p.stepAssembleSections(state)

	toolsPos := -1
	procPos := -1
	for _, s := range state.Sections {
		if s.Label == "Available Tools" {
			toolsPos = s.Position
		}
		if s.Label == "Procedure" {
			procPos = s.Position
		}
	}
	if toolsPos == -1 {
		t.Fatal("Available Tools section not found in assembled sections")
	}
	if procPos == -1 {
		t.Fatal("Procedure section not found in assembled sections")
	}
	if toolsPos >= procPos {
		t.Errorf("Available Tools position (%d) must be less than Procedure position (%d)", toolsPos, procPos)
	}
}

// toolHintWriteRole creates a minimal role YAML file for testing.
func toolHintWriteRole(t *testing.T, dir, id, inherits string) {
	t.Helper()
	var sb strings.Builder
	sb.WriteString("id: " + id + "\n")
	if inherits != "" {
		sb.WriteString("inherits: " + inherits + "\n")
	}
	sb.WriteString("identity: test role for " + id + "\n")
	sb.WriteString("vocabulary:\n  - test term\n")
	path := filepath.Join(dir, id+".yaml")
	if err := os.WriteFile(path, []byte(sb.String()), 0644); err != nil {
		t.Fatalf("writing role %s: %v", id, err)
	}
}
