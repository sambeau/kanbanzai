package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

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
