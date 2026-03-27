package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"kanbanzai/internal/service"
)

// writeDecisionEntityForAssembly writes a decision entity YAML file to the test root.
func writeDecisionEntityForAssembly(t *testing.T, root, id, slug, summary, status string, tags []string) {
	t.Helper()
	dir := filepath.Join(root, "decisions")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir decisions: %v", err)
	}
	tagsYAML := ""
	for _, tag := range tags {
		tagsYAML += fmt.Sprintf("\n  - %s", tag)
	}
	content := fmt.Sprintf(`id: %s
slug: %s
summary: %s
rationale: Test rationale
decided_by: test
date: "2026-03-01T00:00:00Z"
status: %s
tags:%s
`, id, slug, summary, status, tagsYAML)
	path := filepath.Join(dir, id+"-"+slug+".yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write decision entity: %v", err)
	}
}

// P5-3.8: When the project has 1+ decision entities tagged workflow-experiment
// with status accepted, context assembly appends experiment nudges.
func TestAssemblyExperimentNudge_Present(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	entitySvc := service.NewEntityService(root)

	writeDecisionEntityForAssembly(t, root, "DEC-0100000000001", "add-error-format",
		"Add error format to spec template", "accepted",
		[]string{"workflow-experiment", "retrospective"})

	nudges := asmLoadExperimentNudge(entitySvc)
	if len(nudges) != 1 {
		t.Fatalf("nudges len = %d, want 1", len(nudges))
	}
	if nudges[0].decisionID != "DEC-0100000000001" {
		t.Errorf("decisionID = %q, want %q", nudges[0].decisionID, "DEC-0100000000001")
	}
	if nudges[0].summary != "Add error format to spec template" {
		t.Errorf("summary = %q, want %q", nudges[0].summary, "Add error format to spec template")
	}
}

// P5-3.9: When no workflow-experiment decisions are in accepted status,
// no experiment nudge section is appended.
func TestAssemblyExperimentNudge_AbsentWhenNoneAccepted(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	entitySvc := service.NewEntityService(root)

	// Decision tagged workflow-experiment but in rejected status.
	writeDecisionEntityForAssembly(t, root, "DEC-0100000000002", "rejected-exp",
		"Rejected experiment", "rejected",
		[]string{"workflow-experiment"})

	nudges := asmLoadExperimentNudge(entitySvc)
	if len(nudges) != 0 {
		t.Errorf("nudges len = %d, want 0 when no accepted experiments", len(nudges))
	}
}

func TestAssemblyExperimentNudge_AbsentWhenNoDecisions(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	entitySvc := service.NewEntityService(root)

	nudges := asmLoadExperimentNudge(entitySvc)
	if nudges != nil {
		t.Errorf("nudges = %v, want nil when no decisions exist", nudges)
	}
}

func TestAssemblyExperimentNudge_ExcludesNonExperimentDecisions(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	entitySvc := service.NewEntityService(root)

	// Accepted decision but NOT tagged workflow-experiment.
	writeDecisionEntityForAssembly(t, root, "DEC-0100000000003", "regular-decision",
		"Regular accepted decision", "accepted",
		[]string{"some-other-tag"})

	nudges := asmLoadExperimentNudge(entitySvc)
	if len(nudges) != 0 {
		t.Errorf("nudges len = %d, want 0 for non-experiment decisions", len(nudges))
	}
}

func TestAssemblyExperimentNudge_MultipleExperiments(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	entitySvc := service.NewEntityService(root)

	writeDecisionEntityForAssembly(t, root, "DEC-0100000000004", "exp-alpha",
		"Experiment Alpha", "accepted",
		[]string{"workflow-experiment"})
	writeDecisionEntityForAssembly(t, root, "DEC-0100000000005", "exp-beta",
		"Experiment Beta", "accepted",
		[]string{"workflow-experiment"})

	nudges := asmLoadExperimentNudge(entitySvc)
	if len(nudges) != 2 {
		t.Fatalf("nudges len = %d, want 2", len(nudges))
	}
}

// P5-3.10: The experiment nudge does not count against knowledge entry budget.
// Verify that nudge bytes are counted for total usage but not trimmed.
func TestAssemblyExperimentNudge_NotTrimmed(t *testing.T) {
	t.Parallel()

	// Build an assembledContext with the budget nearly exceeded and a nudge.
	actx := assembledContext{
		byteBudget: 100,
		knowledge: []asmKnowledgeEntry{
			{topic: "topic-1", content: "some knowledge content here for testing", scope: "project", confidence: 0.5, tier: 3},
		},
		experimentNudge: []asmExperimentNudge{
			{decisionID: "DEC-0100000000001", summary: "Add error format to spec template"},
		},
	}

	actx.byteUsage = asmByteCount(actx)

	// Force trimming.
	if actx.byteUsage <= actx.byteBudget {
		// Inflate knowledge to exceed budget.
		actx.knowledge = append(actx.knowledge,
			asmKnowledgeEntry{topic: "topic-2", content: "extra padding content to exceed the byte budget limit", scope: "project", confidence: 0.3, tier: 3},
			asmKnowledgeEntry{topic: "topic-3", content: "even more extra content to really push over the limit", scope: "project", confidence: 0.2, tier: 3},
		)
		actx.byteUsage = asmByteCount(actx)
	}

	trimmed := asmTrimContext(actx)

	// The nudge should survive trimming.
	if len(trimmed.experimentNudge) != 1 {
		t.Errorf("experimentNudge len = %d, want 1 after trimming (nudge should not be trimmed)", len(trimmed.experimentNudge))
	}
	if trimmed.experimentNudge[0].decisionID != "DEC-0100000000001" {
		t.Errorf("nudge decisionID = %q, want %q", trimmed.experimentNudge[0].decisionID, "DEC-0100000000001")
	}
}

// P5-3.8: assembleContext includes nudge when entitySvc is provided and
// experiments exist.
func TestAssembleContext_IncludesExperimentNudge(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	entitySvc := service.NewEntityService(root)

	writeDecisionEntityForAssembly(t, root, "DEC-0100000000006", "asm-test",
		"Assembly test experiment", "accepted",
		[]string{"workflow-experiment"})

	actx := assembleContext(asmInput{
		entitySvc: entitySvc,
	})

	if len(actx.experimentNudge) != 1 {
		t.Fatalf("experimentNudge len = %d, want 1", len(actx.experimentNudge))
	}
	if actx.experimentNudge[0].decisionID != "DEC-0100000000006" {
		t.Errorf("decisionID = %q, want %q", actx.experimentNudge[0].decisionID, "DEC-0100000000006")
	}
}

// P5-3.9: assembleContext does not include nudge when entitySvc is nil.
func TestAssembleContext_NoNudgeWithoutEntitySvc(t *testing.T) {
	t.Parallel()

	actx := assembleContext(asmInput{})
	if len(actx.experimentNudge) != 0 {
		t.Errorf("experimentNudge len = %d, want 0 when entitySvc is nil", len(actx.experimentNudge))
	}
}

// hasTag helper tests.
func TestHasTag_FoundInAnySlice(t *testing.T) {
	t.Parallel()
	state := map[string]any{
		"tags": []any{"workflow-experiment", "retrospective"},
	}
	if !hasTag(state, "workflow-experiment") {
		t.Error("hasTag should find 'workflow-experiment' in []any tags")
	}
}

func TestHasTag_FoundInStringSlice(t *testing.T) {
	t.Parallel()
	state := map[string]any{
		"tags": []string{"workflow-experiment", "retrospective"},
	}
	if !hasTag(state, "workflow-experiment") {
		t.Error("hasTag should find 'workflow-experiment' in []string tags")
	}
}

func TestHasTag_NotFound(t *testing.T) {
	t.Parallel()
	state := map[string]any{
		"tags": []any{"some-tag", "other-tag"},
	}
	if hasTag(state, "workflow-experiment") {
		t.Error("hasTag should not find 'workflow-experiment'")
	}
}

func TestHasTag_NoTagsField(t *testing.T) {
	t.Parallel()
	state := map[string]any{}
	if hasTag(state, "workflow-experiment") {
		t.Error("hasTag should return false when no tags field exists")
	}
}

// nextContextToMap includes active_experiments when nudge present.
func TestNextContextToMap_WithExperiments(t *testing.T) {
	t.Parallel()
	actx := assembledContext{
		experimentNudge: []asmExperimentNudge{
			{decisionID: "DEC-0100000000001", summary: "Test experiment"},
		},
	}
	m := nextContextToMap(actx)
	exps, ok := m["active_experiments"].([]map[string]any)
	if !ok {
		t.Fatal("active_experiments should be present in context map")
	}
	if len(exps) != 1 {
		t.Fatalf("active_experiments len = %d, want 1", len(exps))
	}
	if exps[0]["decision_id"] != "DEC-0100000000001" {
		t.Errorf("decision_id = %v, want %q", exps[0]["decision_id"], "DEC-0100000000001")
	}
	if exps[0]["summary"] != "Test experiment" {
		t.Errorf("summary = %v, want %q", exps[0]["summary"], "Test experiment")
	}
}

// nextContextToMap omits active_experiments when no nudge present.
func TestNextContextToMap_WithoutExperiments(t *testing.T) {
	t.Parallel()
	actx := assembledContext{}
	m := nextContextToMap(actx)
	if _, ok := m["active_experiments"]; ok {
		t.Error("active_experiments should not be present when no nudges exist")
	}
}

// renderHandoffPrompt includes experiments section when nudge present.
func TestRenderHandoffPrompt_WithExperiments(t *testing.T) {
	t.Parallel()
	taskState := map[string]any{
		"id":      "TASK-001",
		"summary": "Test task",
	}
	actx := assembledContext{
		experimentNudge: []asmExperimentNudge{
			{decisionID: "DEC-0100000000001", summary: "Add error format to spec template"},
		},
	}
	prompt := renderHandoffPrompt(taskState, actx, "")
	if !containsStr(prompt, "Active Workflow Experiments") {
		t.Error("prompt should contain 'Active Workflow Experiments' section")
	}
	if !containsStr(prompt, "DEC-0100000000001") {
		t.Error("prompt should contain decision ID")
	}
	if !containsStr(prompt, "Add error format to spec template") {
		t.Error("prompt should contain experiment summary")
	}
}

// renderHandoffPrompt omits experiments section when no nudge.
func TestRenderHandoffPrompt_WithoutExperiments(t *testing.T) {
	t.Parallel()
	taskState := map[string]any{
		"id":      "TASK-001",
		"summary": "Test task",
	}
	actx := assembledContext{}
	prompt := renderHandoffPrompt(taskState, actx, "")
	if containsStr(prompt, "Active Workflow Experiments") {
		t.Error("prompt should not contain 'Active Workflow Experiments' when no nudges")
	}
}

// containsStr is a simple helper to avoid importing strings in this test file
// just for one function.
func containsStr(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
