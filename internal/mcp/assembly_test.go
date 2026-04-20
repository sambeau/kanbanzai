package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sambeau/kanbanzai/internal/service"
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

// ─── asmExtractCriteria bold-identifier tests ─────────────────────────────────

// AC-01: **AC-NN.** lines in an acceptance-criteria section are extracted.
func TestAsmExtractCriteria_BoldAC_InACSection(t *testing.T) {
	t.Parallel()
	sections := []asmSpecSection{
		{
			document: "spec.md",
			section:  "Acceptance Criteria",
			content:  "**AC-01.** The system must do X.\n**AC-02.** The system must do Y.",
		},
	}
	got := asmExtractCriteria(sections)
	if len(got) != 2 {
		t.Fatalf("len(criteria) = %d, want 2; got: %v", len(got), got)
	}
	if got[0] != "AC-01: The system must do X." {
		t.Errorf("criteria[0] = %q, want %q", got[0], "AC-01: The system must do X.")
	}
	if got[1] != "AC-02: The system must do Y." {
		t.Errorf("criteria[1] = %q, want %q", got[1], "AC-02: The system must do Y.")
	}
}

// AC-02: **REQ-NN.** lines in a requirements section are extracted.
func TestAsmExtractCriteria_BoldREQ_InReqSection(t *testing.T) {
	t.Parallel()
	sections := []asmSpecSection{
		{
			document: "spec.md",
			section:  "Requirements",
			content:  "**REQ-01.** The service MUST respond within 100ms.\n**REQ-02.** The service SHALL log all errors.",
		},
	}
	got := asmExtractCriteria(sections)
	if len(got) != 2 {
		t.Fatalf("len(criteria) = %d, want 2; got: %v", len(got), got)
	}
	if got[0] != "REQ-01: The service MUST respond within 100ms." {
		t.Errorf("criteria[0] = %q, want %q", got[0], "REQ-01: The service MUST respond within 100ms.")
	}
}

// AC-02: **C-NN.** lines in a constraints section are extracted.
func TestAsmExtractCriteria_BoldC_InConstraintsSection(t *testing.T) {
	t.Parallel()
	sections := []asmSpecSection{
		{
			document: "spec.md",
			section:  "Constraints",
			content:  "**C-01.** No external network calls.\n**C-02.** Must be idempotent.",
		},
	}
	got := asmExtractCriteria(sections)
	if len(got) != 2 {
		t.Fatalf("len(criteria) = %d, want 2; got: %v", len(got), got)
	}
	if got[0] != "C-01: No external network calls." {
		t.Errorf("criteria[0] = %q, want %q", got[0], "C-01: No external network calls.")
	}
}

// AC-03: Extracted criterion text preserves the identifier prefix as "XX-NN: text".
func TestAsmExtractCriteria_BoldIdent_PreservesPrefix(t *testing.T) {
	t.Parallel()
	sections := []asmSpecSection{
		{
			document: "spec.md",
			section:  "Acceptance Criteria",
			content:  "**INV-03.** The invariant must hold at all times.",
		},
	}
	got := asmExtractCriteria(sections)
	if len(got) != 1 {
		t.Fatalf("len(criteria) = %d, want 1; got: %v", len(got), got)
	}
	want := "INV-03: The invariant must hold at all times."
	if got[0] != want {
		t.Errorf("criterion = %q, want %q", got[0], want)
	}
}

// AC-04: Bold-identifier lines outside acceptance/requirement/constraint sections
// are only extracted when their text contains an RFC 2119 keyword.
func TestAsmExtractCriteria_BoldIdent_OutsideACSection_NoKeyword(t *testing.T) {
	t.Parallel()
	sections := []asmSpecSection{
		{
			document: "spec.md",
			section:  "Background",
			content:  "**AC-01.** Some informational note without keywords.",
		},
	}
	got := asmExtractCriteria(sections)
	if len(got) != 0 {
		t.Errorf("criteria = %v, want empty (no RFC 2119 keyword outside AC section)", got)
	}
}

func TestAsmExtractCriteria_BoldIdent_OutsideACSection_WithKeyword(t *testing.T) {
	t.Parallel()
	sections := []asmSpecSection{
		{
			document: "spec.md",
			section:  "Implementation Notes",
			content:  "**AC-01.** The handler MUST validate the input before processing.",
		},
	}
	got := asmExtractCriteria(sections)
	if len(got) != 1 {
		t.Fatalf("len(criteria) = %d, want 1 (has MUST keyword); got: %v", len(got), got)
	}
	want := "AC-01: The handler MUST validate the input before processing."
	if got[0] != want {
		t.Errorf("criterion = %q, want %q", got[0], want)
	}
}

// AC-05 regression: Existing list-item extraction is unaffected.
func TestAsmExtractCriteria_ListItems_Unaffected(t *testing.T) {
	t.Parallel()
	sections := []asmSpecSection{
		{
			document: "spec.md",
			section:  "Acceptance Criteria",
			content:  "- The system stores data correctly.\n- The system retrieves data correctly.\n- Error cases return proper codes.",
		},
	}
	got := asmExtractCriteria(sections)
	if len(got) != 3 {
		t.Fatalf("len(criteria) = %d, want 3 (list items unaffected); got: %v", len(got), got)
	}
	if got[0] != "The system stores data correctly." {
		t.Errorf("criteria[0] = %q, want %q", got[0], "The system stores data correctly.")
	}
}

// AC-05 regression: Existing numbered-list extraction is unaffected.
func TestAsmExtractCriteria_NumberedList_Unaffected(t *testing.T) {
	t.Parallel()
	sections := []asmSpecSection{
		{
			document: "spec.md",
			section:  "Requirements",
			content:  "1. First requirement.\n2. Second requirement.\n3. Third requirement.",
		},
	}
	got := asmExtractCriteria(sections)
	if len(got) != 3 {
		t.Fatalf("len(criteria) = %d, want 3 (numbered list unaffected); got: %v", len(got), got)
	}
	if got[0] != "First requirement." {
		t.Errorf("criteria[0] = %q, want %q", got[0], "First requirement.")
	}
}

// REQ-12: No bold-idents, no list items → zero criteria.
func TestAsmExtractCriteria_NoBoldIdents_ZeroCriteria(t *testing.T) {
	t.Parallel()
	sections := []asmSpecSection{
		{
			document: "spec.md",
			section:  "Overview",
			content:  "This is a prose paragraph.\nIt has no list items or bold identifiers.\nSo no criteria are extracted.",
		},
	}
	got := asmExtractCriteria(sections)
	if len(got) != 0 {
		t.Errorf("criteria = %v, want empty (no list items or bold idents)", got)
	}
}

// REQ-03: Bold-identifier pattern is case-sensitive for the prefix (must be uppercase).
func TestAsmExtractCriteria_BoldIdent_LowercasePrefixNotMatched(t *testing.T) {
	t.Parallel()
	sections := []asmSpecSection{
		{
			document: "spec.md",
			section:  "Acceptance Criteria",
			content:  "**ac-01.** Lowercase prefix should not match.",
		},
	}
	got := asmExtractCriteria(sections)
	if len(got) != 0 {
		t.Errorf("criteria = %v, want empty (lowercase prefix must not match)", got)
	}
}

// REQ-06: Constraints section heading is treated as an acceptance section.
func TestAsmExtractCriteria_ConstraintsSection_ExtractsAllBoldIdents(t *testing.T) {
	t.Parallel()
	sections := []asmSpecSection{
		{
			document: "spec.md",
			section:  "Constraints and Invariants",
			content:  "**C-01.** No side effects.\n**INV-01.** State is always consistent.",
		},
	}
	got := asmExtractCriteria(sections)
	if len(got) != 2 {
		t.Fatalf("len(criteria) = %d, want 2 (constraints section); got: %v", len(got), got)
	}
}

// RFC 2119 keywords: SHOULD, SHOULD NOT, MAY, REQUIRED, RECOMMENDED, OPTIONAL.
func TestAsmExtractCriteria_BoldIdent_ShouldKeyword(t *testing.T) {
	t.Parallel()
	sections := []asmSpecSection{
		{
			document: "spec.md",
			section:  "Notes",
			content:  "**REC-01.** Implementations SHOULD prefer the batch path for performance.",
		},
	}
	got := asmExtractCriteria(sections)
	if len(got) != 1 {
		t.Fatalf("len(criteria) = %d, want 1 (SHOULD keyword); got: %v", len(got), got)
	}
}

func TestAsmExtractCriteria_BoldIdent_MayKeyword(t *testing.T) {
	t.Parallel()
	sections := []asmSpecSection{
		{
			document: "spec.md",
			section:  "Notes",
			content:  "**OPT-01.** Callers MAY pass an empty slice.",
		},
	}
	got := asmExtractCriteria(sections)
	if len(got) != 1 {
		t.Fatalf("len(criteria) = %d, want 1 (MAY keyword); got: %v", len(got), got)
	}
}

// TestAssembleContext_StageContentFilter verifies that file path extraction
// respects stage-aware IncludeFilePaths configuration (FR-005).
func TestAssembleContext_StageContentFilter(t *testing.T) {
	t.Parallel()

	taskState := map[string]any{
		"id":            "TASK-TEST",
		"files_planned": []any{"file1.go", "file2.go"},
	}

	tests := []struct {
		name         string
		featureStage string
		wantFiles    int
	}{
		{"designing stage excludes files", "designing", 0},
		{"specifying stage excludes files", "specifying", 0},
		{"developing stage includes files", "developing", 2},
		{"reviewing stage includes files", "reviewing", 2},
		{"no stage (backward compat) includes files", "", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actx := assembleContext(asmInput{
				taskState:    taskState,
				featureStage: tt.featureStage,
			})
			if got := len(actx.filesContext); got != tt.wantFiles {
				t.Errorf("stage %q: filesContext has %d entries, want %d", tt.featureStage, got, tt.wantFiles)
			}
		})
	}
}

// renderHandoffPrompt includes ## Available Tools section when tool hint is set (AC-012).
func TestRenderHandoffPrompt_WithToolHint(t *testing.T) {
	t.Parallel()
	taskState := map[string]any{
		"id":      "TASK-001",
		"summary": "Test task",
	}
	actx := assembledContext{
		toolHint: "Use search_graph and trace_call_path for code navigation.",
	}
	prompt := renderHandoffPrompt(taskState, actx, "some instructions")
	if !containsStr(prompt, "## Available Tools") {
		t.Error("prompt should contain '## Available Tools' section when tool hint is set")
	}
	if !containsStr(prompt, "search_graph") {
		t.Error("prompt should contain the tool hint content")
	}
	// Verify ordering: Available Tools before Additional Instructions (FR-013).
	toolsIdx := 0
	instrIdx := 0
	for i := 0; i+len("## Available Tools") <= len(prompt); i++ {
		if prompt[i:i+len("## Available Tools")] == "## Available Tools" {
			toolsIdx = i
			break
		}
	}
	for i := 0; i+len("### Additional Instructions") <= len(prompt); i++ {
		if prompt[i:i+len("### Additional Instructions")] == "### Additional Instructions" {
			instrIdx = i
			break
		}
	}
	if toolsIdx >= instrIdx {
		t.Errorf("## Available Tools (pos %d) must appear before ### Additional Instructions (pos %d)", toolsIdx, instrIdx)
	}
}

// renderHandoffPrompt omits ## Available Tools when no tool hint (AC-013).
func TestRenderHandoffPrompt_WithoutToolHint(t *testing.T) {
	t.Parallel()
	taskState := map[string]any{
		"id":      "TASK-001",
		"summary": "Test task",
	}
	actx := assembledContext{}
	prompt := renderHandoffPrompt(taskState, actx, "")
	if containsStr(prompt, "## Available Tools") {
		t.Error("prompt should not contain '## Available Tools' when no tool hint is set")
	}
}

// nextContextToMap includes tool_hint when set (AC-014).
func TestNextContextToMap_WithToolHint(t *testing.T) {
	t.Parallel()
	actx := assembledContext{
		toolHint: "Use search_graph for code navigation",
	}
	m := nextContextToMap(actx)
	hint, ok := m["tool_hint"].(string)
	if !ok {
		t.Fatal("tool_hint should be present in context map")
	}
	if hint != "Use search_graph for code navigation" {
		t.Errorf("tool_hint = %q, want %q", hint, "Use search_graph for code navigation")
	}
}

// nextContextToMap omits tool_hint when not set (AC-015).
func TestNextContextToMap_WithoutToolHint(t *testing.T) {
	t.Parallel()
	actx := assembledContext{}
	m := nextContextToMap(actx)
	if _, ok := m["tool_hint"]; ok {
		t.Error("tool_hint should not be present when no hint is set")
	}
}

// TestRenderHandoffPrompt_NoHintsIdenticalOutput verifies that a zero-value
// assembledContext and one with an explicitly empty toolHint produce
// byte-identical prompts, and neither contains "## Available Tools" (AC-019).
func TestRenderHandoffPrompt_NoHintsIdenticalOutput(t *testing.T) {
	t.Parallel()
	taskState := map[string]any{
		"id":      "TASK-001",
		"summary": "Test task",
	}

	zeroCtx := assembledContext{}
	emptyCtx := assembledContext{toolHint: ""}

	promptZero := renderHandoffPrompt(taskState, zeroCtx, "")
	promptEmpty := renderHandoffPrompt(taskState, emptyCtx, "")

	if promptZero != promptEmpty {
		t.Error("zero-value and explicitly-empty toolHint prompts must be byte-identical")
	}
	if containsStr(promptZero, "## Available Tools") {
		t.Error("zero-value prompt should not contain '## Available Tools'")
	}
	if containsStr(promptEmpty, "## Available Tools") {
		t.Error("empty toolHint prompt should not contain '## Available Tools'")
	}
}

// ─── Code Graph section tests ─────────────────────────────────────────────────

// TestRenderHandoffPrompt_CodeGraphSection_ProjectSet verifies AC-006: when
// GraphProject is set, the prompt contains ## Code Graph with the project name
// and four tool call examples.
func TestRenderHandoffPrompt_CodeGraphSection_ProjectSet(t *testing.T) {
	t.Parallel()
	taskState := map[string]any{
		"id":      "TASK-001",
		"summary": "Test task",
	}
	actx := assembledContext{
		graphProject: "kanbanzai-FEAT-XXX",
		worktreePath: "/tmp/wt/feat-xxx",
		hasWorktree:  true,
	}
	prompt := renderHandoffPrompt(taskState, actx, "")

	if !containsStr(prompt, "## Code Graph") {
		t.Error("prompt should contain '## Code Graph' section")
	}
	if !containsStr(prompt, "kanbanzai-FEAT-XXX") {
		t.Error("prompt should contain the project name")
	}
	// Four tool examples required by AC-006.
	for _, tool := range []string{"search_graph", "trace_call_path", "query_graph", "get_code_snippet"} {
		if !containsStr(prompt, tool) {
			t.Errorf("prompt should contain tool example for %s", tool)
		}
	}
	// Preference instruction.
	if !containsStr(prompt, "prefer graph tools over grep") {
		t.Error("prompt should contain graph tool preference instruction")
	}
	// Re-indexing instruction.
	if !containsStr(prompt, "index_repository") {
		t.Error("prompt should contain re-indexing instruction")
	}
	if !containsStr(prompt, "/tmp/wt/feat-xxx") {
		t.Error("prompt should contain worktree path in re-indexing instruction")
	}
}

// TestRenderHandoffPrompt_CodeGraphSection_ProjectEmpty verifies AC-007: when
// hasWorktree is true but GraphProject is empty, the prompt contains ## Code Graph
// with an index_repository instruction.
func TestRenderHandoffPrompt_CodeGraphSection_ProjectEmpty(t *testing.T) {
	t.Parallel()
	taskState := map[string]any{
		"id":      "TASK-001",
		"summary": "Test task",
	}
	actx := assembledContext{
		graphProject: "",
		worktreePath: "/tmp/wt/feat-yyy",
		hasWorktree:  true,
	}
	prompt := renderHandoffPrompt(taskState, actx, "")

	if !containsStr(prompt, "## Code Graph") {
		t.Error("prompt should contain '## Code Graph' section when worktree exists but project is empty")
	}
	if !containsStr(prompt, "index_repository") {
		t.Error("prompt should contain index_repository instruction")
	}
	if !containsStr(prompt, "/tmp/wt/feat-yyy") {
		t.Error("prompt should contain worktree path")
	}
	// Should NOT contain tool examples since project is empty.
	if containsStr(prompt, "search_graph") {
		t.Error("prompt should not contain tool examples when project is empty")
	}
}

// TestRenderHandoffPrompt_CodeGraphSection_NoWorktree verifies AC-008: when
// no worktree exists, the prompt must not contain ## Code Graph.
func TestRenderHandoffPrompt_CodeGraphSection_NoWorktree(t *testing.T) {
	t.Parallel()
	taskState := map[string]any{
		"id":      "TASK-001",
		"summary": "Test task",
	}
	actx := assembledContext{
		hasWorktree: false,
	}
	prompt := renderHandoffPrompt(taskState, actx, "")

	if containsStr(prompt, "## Code Graph") {
		t.Error("prompt should not contain '## Code Graph' when no worktree exists")
	}
}

// TestRenderHandoffPrompt_CodeGraphSection_AfterAvailableTools verifies AC-009:
// ## Code Graph appears after ## Available Tools when both are present.
func TestRenderHandoffPrompt_CodeGraphSection_AfterAvailableTools(t *testing.T) {
	t.Parallel()
	taskState := map[string]any{
		"id":      "TASK-001",
		"summary": "Test task",
	}
	actx := assembledContext{
		toolHint:     "Use search_graph for code navigation.",
		graphProject: "kanbanzai-FEAT-ZZZ",
		worktreePath: "/tmp/wt/feat-zzz",
		hasWorktree:  true,
	}
	prompt := renderHandoffPrompt(taskState, actx, "")

	toolsIdx := strings.Index(prompt, "## Available Tools")
	graphIdx := strings.Index(prompt, "## Code Graph")
	if toolsIdx < 0 {
		t.Fatal("prompt missing '## Available Tools'")
	}
	if graphIdx < 0 {
		t.Fatal("prompt missing '## Code Graph'")
	}
	if toolsIdx >= graphIdx {
		t.Errorf("## Available Tools (pos %d) must appear before ## Code Graph (pos %d)", toolsIdx, graphIdx)
	}
}

// TestRenderHandoffPrompt_CodeGraphSection_Under500Bytes verifies NFR-003:
// the ## Code Graph section must not exceed 500 bytes when GraphProject is set.
func TestRenderHandoffPrompt_CodeGraphSection_Under500Bytes(t *testing.T) {
	t.Parallel()
	taskState := map[string]any{
		"id":      "TASK-001",
		"summary": "Test task",
	}
	// Use a reasonably long project name and worktree path.
	actx := assembledContext{
		graphProject: "kanbanzai-FEAT-01JXABCDEF123456",
		worktreePath: "/Users/someone/Dev/kanbanzai/.kbz/worktrees/feat-some-feature-name",
		hasWorktree:  true,
	}
	prompt := renderHandoffPrompt(taskState, actx, "")

	// Extract just the ## Code Graph section.
	start := strings.Index(prompt, "## Code Graph")
	if start < 0 {
		t.Fatal("prompt missing '## Code Graph'")
	}
	// Find the next ## heading or end of string.
	rest := prompt[start+len("## Code Graph"):]
	end := strings.Index(rest, "\n## ")
	var section string
	if end < 0 {
		section = prompt[start:]
	} else {
		section = prompt[start : start+len("## Code Graph")+end]
	}

	if len(section) > 500 {
		t.Errorf("## Code Graph section is %d bytes, must be <= 500 bytes:\n%s", len(section), section)
	}
}

// TestNextContextToMap_GraphProject verifies AC-010: graph_project is present
// in next structured output when worktree has a GraphProject.
func TestNextContextToMap_GraphProject(t *testing.T) {
	t.Parallel()
	actx := assembledContext{
		graphProject: "kanbanzai-FEAT-XXX",
		hasWorktree:  true,
	}
	m := nextContextToMap(actx)
	gp, ok := m["graph_project"].(string)
	if !ok {
		t.Fatal("graph_project should be present in context map")
	}
	if gp != "kanbanzai-FEAT-XXX" {
		t.Errorf("graph_project = %q, want %q", gp, "kanbanzai-FEAT-XXX")
	}
}

// TestNextContextToMap_GraphProjectEmpty verifies AC-011: graph_project is
// empty string when no worktree exists.
func TestNextContextToMap_GraphProjectEmpty(t *testing.T) {
	t.Parallel()
	actx := assembledContext{}
	m := nextContextToMap(actx)
	gp, ok := m["graph_project"].(string)
	if !ok {
		t.Fatal("graph_project should be present in context map (as empty string)")
	}
	if gp != "" {
		t.Errorf("graph_project = %q, want empty string", gp)
	}
}

// TestWorkflow_NoGraphToolsAvailable verifies AC-018: when codebase_memory_mcp
// is unavailable (GraphProject empty, no worktree), handoff, next context, and
// status attention all produce no errors and identical non-graph behaviour.
// This is the verification plan entry for AC-018.
func TestWorkflow_NoGraphToolsAvailable(t *testing.T) {
	t.Parallel()

	taskState := map[string]any{
		"id":      "TASK-001",
		"summary": "Test task",
	}

	// Zero-value assembledContext simulates no worktree / no graph tools.
	actx := assembledContext{}

	// 1. Handoff: no ## Code Graph section, no errors.
	prompt := renderHandoffPrompt(taskState, actx, "some instructions")
	if containsStr(prompt, "## Code Graph") {
		t.Error("handoff prompt should not contain '## Code Graph' when no graph tools available")
	}
	if containsStr(prompt, "index_repository") {
		t.Error("handoff prompt should not reference index_repository when no worktree exists")
	}
	if !containsStr(prompt, "Test task") {
		t.Error("handoff prompt should still contain the task summary")
	}

	// 2. Next context: graph_project is empty string, no error fields.
	m := nextContextToMap(actx)
	gp, ok := m["graph_project"].(string)
	if !ok {
		t.Fatal("graph_project should be present in next context map")
	}
	if gp != "" {
		t.Errorf("graph_project = %q, want empty string", gp)
	}

	// 3. Status attention: no missing_graph_index item when no worktree.
	// Use the full generateFeatureAttention signature with zero/empty values
	// to simulate a feature with no worktree and no graph tools.
	items := generateFeatureAttention(
		nil,          // tasks
		nil,          // docs
		0,            // totalTasks
		"",           // featureID
		"",           // featureDisplayID
		"developing", // featureStatus
		time.Time{},  // featureUpdated
		false,        // inheritedHasSpec
		false,        // inheritedHasDevPlan
		14,           // staleReviewingDays
		nil,          // bugs
		false,        // hasActiveWorktree
		"",           // worktreeGraphProject
	)
	for _, item := range items {
		if item.Type == "missing_graph_index" {
			t.Error("status should not emit missing_graph_index when no worktree exists")
		}
	}
}

// ─── asmLoadDocumentPointers tests ───────────────────────────────────────────

// TestAsmLoadDocumentPointers_EmptySvc verifies that a nil intelligence service
// returns nil (graceful degradation).
func TestAsmLoadDocumentPointers_EmptySvc(t *testing.T) {
	t.Parallel()
	knowledge := []asmKnowledgeEntry{
		{topic: "some-topic", content: "some content", scope: "FEAT-01TESTPOINTER0001"},
	}
	got := asmLoadDocumentPointers(nil, knowledge)
	if got != nil {
		t.Errorf("asmLoadDocumentPointers(nil svc) = %v, want nil", got)
	}
}

// TestAsmLoadDocumentPointers_EmptyKnowledge verifies that empty knowledge
// entries returns nil.
func TestAsmLoadDocumentPointers_EmptyKnowledge(t *testing.T) {
	t.Parallel()
	repoRoot := t.TempDir()
	indexRoot := filepath.Join(t.TempDir(), "index")
	svc := service.NewIntelligenceService(indexRoot, repoRoot)

	got := asmLoadDocumentPointers(svc, nil)
	if got != nil {
		t.Errorf("asmLoadDocumentPointers(empty knowledge) = %v, want nil", got)
	}
	got = asmLoadDocumentPointers(svc, []asmKnowledgeEntry{})
	if got != nil {
		t.Errorf("asmLoadDocumentPointers(empty slice) = %v, want nil", got)
	}
}

// TestAsmLoadDocumentPointers_NonEntityScope verifies that entries with
// non-entity scopes (like "project" or a role name) produce no pointers.
func TestAsmLoadDocumentPointers_NonEntityScope(t *testing.T) {
	t.Parallel()
	repoRoot := t.TempDir()
	indexRoot := filepath.Join(t.TempDir(), "index")
	svc := service.NewIntelligenceService(indexRoot, repoRoot)

	knowledge := []asmKnowledgeEntry{
		{topic: "t1", content: "c1", scope: "project"},
		{topic: "t2", content: "c2", scope: "implementer-go"},
	}
	got := asmLoadDocumentPointers(svc, knowledge)
	if got != nil {
		t.Errorf("asmLoadDocumentPointers(non-entity scopes) = %v, want nil", got)
	}
}

// TestAsmIsEntityID verifies the entity ID detection helper.
func TestAsmIsEntityID(t *testing.T) {
	t.Parallel()
	cases := []struct {
		s    string
		want bool
	}{
		{"FEAT-01ABCDEF", true},
		{"TASK-01ABCDEF", true},
		{"BUG-01ABCDEF", true},
		{"project", false},
		{"implementer-go", false},
		{"", false},
		{"DEC-01ABCDEF", false},
	}
	for _, tc := range cases {
		got := asmIsEntityID(tc.s)
		if got != tc.want {
			t.Errorf("asmIsEntityID(%q) = %v, want %v", tc.s, got, tc.want)
		}
	}
}

// TestAsmLoadDocumentPointers_EntityScopeNoIndex verifies that an entity-scoped
// knowledge entry with no indexed documents produces no pointers (not an error).
func TestAsmLoadDocumentPointers_EntityScopeNoIndex(t *testing.T) {
	t.Parallel()
	repoRoot := t.TempDir()
	indexRoot := filepath.Join(t.TempDir(), "index")
	svc := service.NewIntelligenceService(indexRoot, repoRoot)

	knowledge := []asmKnowledgeEntry{
		{topic: "t1", content: "c1", scope: "TASK-01TESTNOINDEX00001"},
	}
	got := asmLoadDocumentPointers(svc, knowledge)
	// No index = no matches, so no pointers. Must not error.
	if len(got) != 0 {
		t.Errorf("asmLoadDocumentPointers with no index = %d pointers, want 0", len(got))
	}
}

// TestAsmLoadDocumentPointers_EntityScopeWithIndex verifies the happy path:
// a knowledge entry with an entity ID scope, where the entity is referenced
// in an indexed document, produces a document pointer.
func TestAsmLoadDocumentPointers_EntityScopeWithIndex(t *testing.T) {
	t.Parallel()
	repoRoot := t.TempDir()
	indexRoot := filepath.Join(t.TempDir(), "index")
	svc := service.NewIntelligenceService(indexRoot, repoRoot)

	entityID := "FEAT-ASMPTRENTITY001"

	// Write and ingest a document that references the entity.
	content := "# Design\n\nThis document covers " + entityID + " feature.\n"
	docPath := filepath.Join(repoRoot, "work", "doc.md")
	if err := os.MkdirAll(filepath.Dir(docPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(docPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write doc: %v", err)
	}
	if _, err := svc.IngestDocument("work/doc.md", "work/doc.md"); err != nil {
		t.Fatalf("IngestDocument: %v", err)
	}

	// Knowledge entry with scope = entity ID (so it is picked up).
	knowledge := []asmKnowledgeEntry{
		{topic: "feature-insight", content: "Some insight", scope: entityID},
	}

	got := asmLoadDocumentPointers(svc, knowledge)
	if len(got) == 0 {
		t.Error("expected at least one document pointer for entity-scoped knowledge with indexed doc, got 0")
	}
}

// TestAsmLoadDocumentPointers_LearnedFromEntityID verifies that a knowledge entry
// whose learnedFrom field is an entity ID (and scope is "project") still generates
// document pointers when the entity appears in indexed documents.
func TestAsmLoadDocumentPointers_LearnedFromEntityID(t *testing.T) {
	t.Parallel()
	repoRoot := t.TempDir()
	indexRoot := filepath.Join(t.TempDir(), "index")
	svc := service.NewIntelligenceService(indexRoot, repoRoot)

	entityID := "FEAT-ASMLEARNFROM0001"

	// Write and ingest a document that references the entity.
	content := "# Design\n\nThis document covers " + entityID + " feature.\n"
	docPath := filepath.Join(repoRoot, "work", "doc.md")
	if err := os.MkdirAll(filepath.Dir(docPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(docPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write doc: %v", err)
	}
	if _, err := svc.IngestDocument("work/doc.md", "work/doc.md"); err != nil {
		t.Fatalf("IngestDocument: %v", err)
	}

	// Knowledge entry with scope="project" but learnedFrom=entityID.
	knowledge := []asmKnowledgeEntry{
		{topic: "task-insight", content: "Learned from the feature task", scope: "project", learnedFrom: entityID},
	}

	got := asmLoadDocumentPointers(svc, knowledge)
	if len(got) == 0 {
		t.Error("expected at least one document pointer for learnedFrom-based entity ID, got 0")
	}
}
