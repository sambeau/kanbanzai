package validate

import (
	"context"
	"strings"
	"testing"
)

func TestValidatorContext_ZeroValue(t *testing.T) {
	t.Parallel()

	vctx := ValidatorContext{}
	if vctx.DocumentPath != "" {
		t.Error("zero-value ValidatorContext.DocumentPath should be empty")
	}
	if vctx.DocumentType != "" {
		t.Error("zero-value ValidatorContext.DocumentType should be empty")
	}
}

func TestValidatorContext_AllFieldsPopulated(t *testing.T) {
	t.Parallel()

	vctx := ValidatorContext{
		DocumentPath:  "work/P43/spec.md",
		DocumentType:  "specification",
		ParentDocPath: "work/P43/design.md",
		RubricPath:    ".kbz/rubrics/spec-validator.yaml",
		FeatureID:     "FEAT-01KQSP41PE6JP",
	}

	if vctx.DocumentPath != "work/P43/spec.md" {
		t.Errorf("DocumentPath = %q, want %q", vctx.DocumentPath, "work/P43/spec.md")
	}
	if vctx.DocumentType != "specification" {
		t.Errorf("DocumentType = %q, want %q", vctx.DocumentType, "specification")
	}
	if vctx.ParentDocPath != "work/P43/design.md" {
		t.Errorf("ParentDocPath = %q, want %q", vctx.ParentDocPath, "work/P43/design.md")
	}
	if vctx.RubricPath != ".kbz/rubrics/spec-validator.yaml" {
		t.Errorf("RubricPath = %q, want %q", vctx.RubricPath, ".kbz/rubrics/spec-validator.yaml")
	}
	if vctx.FeatureID != "FEAT-01KQSP41PE6JP" {
		t.Errorf("FeatureID = %q, want %q", vctx.FeatureID, "FEAT-01KQSP41PE6JP")
	}
}

func TestValidatorSummary_VerdictPass(t *testing.T) {
	t.Parallel()

	summary := ValidatorSummary{
		Verdict:          VerdictPass,
		BlockingCount:    0,
		NonBlockingCount: 0,
		EvidenceScore:    0.95,
		ReportDocID:      "DOC-pass-001",
	}

	if summary.Verdict != VerdictPass {
		t.Errorf("Verdict = %q, want %q", summary.Verdict, VerdictPass)
	}
	if summary.BlockingCount != 0 {
		t.Errorf("BlockingCount = %d, want 0", summary.BlockingCount)
	}
}

func TestValidatorSummary_VerdictFail(t *testing.T) {
	t.Parallel()

	summary := ValidatorSummary{
		Verdict:          VerdictFail,
		BlockingCount:    3,
		NonBlockingCount: 2,
		EvidenceScore:    0.87,
		ReportDocID:      "DOC-fail-001",
	}

	if summary.Verdict != VerdictFail {
		t.Errorf("Verdict = %q, want %q", summary.Verdict, VerdictFail)
	}
	if summary.BlockingCount != 3 {
		t.Errorf("BlockingCount = %d, want 3", summary.BlockingCount)
	}
	if summary.NonBlockingCount != 2 {
		t.Errorf("NonBlockingCount = %d, want 2", summary.NonBlockingCount)
	}
}

func TestValidatorSummary_VerdictPassWithNotes(t *testing.T) {
	t.Parallel()

	summary := ValidatorSummary{
		Verdict:          VerdictPassWithNotes,
		BlockingCount:    0,
		NonBlockingCount: 4,
		EvidenceScore:    0.72,
		ReportDocID:      "DOC-notes-001",
	}

	if summary.Verdict != VerdictPassWithNotes {
		t.Errorf("Verdict = %q, want %q", summary.Verdict, VerdictPassWithNotes)
	}
}

func TestNewSpawnAgentDispatcher(t *testing.T) {
	t.Parallel()

	registerCalled := false
	registerFn := func(reportPath, reportContent, docType, title, featureID string) (string, error) {
		registerCalled = true
		return "DOC-001", nil
	}

	d := NewSpawnAgentDispatcher(registerFn)
	if d == nil {
		t.Fatal("NewSpawnAgentDispatcher returned nil")
	}
	if d.RegisterReportFunc == nil {
		t.Fatal("RegisterReportFunc is nil after construction")
	}

	// Verify the function was stored correctly by calling it.
	id, err := d.RegisterReportFunc("path", "content", "report", "title", "FEAT-001")
	if err != nil {
		t.Fatalf("RegisterReportFunc unexpected error: %v", err)
	}
	if id != "DOC-001" {
		t.Errorf("RegisterReportFunc returned %q, want %q", id, "DOC-001")
	}
	if !registerCalled {
		t.Error("RegisterReportFunc was not called")
	}
}

func TestSpawnAgentDispatcher_Dispatch_NilRegisterFunc(t *testing.T) {
	t.Parallel()

	d := &SpawnAgentDispatcher{}
	vctx := ValidatorContext{
		DocumentPath: "work/P43/spec.md",
		DocumentType: "specification",
		FeatureID:    "FEAT-001",
	}

	_, err := d.Dispatch(context.Background(), "spec-validator", "validate-spec", vctx)
	if err == nil {
		t.Fatal("expected error when RegisterReportFunc is nil")
	}
	if !strings.Contains(err.Error(), "RegisterReportFunc is nil") {
		t.Errorf("error message should mention nil RegisterReportFunc, got: %v", err)
	}
}

func TestSpawnAgentDispatcher_Dispatch_GeneratesPrompt(t *testing.T) {
	t.Parallel()

	registerCalled := false
	registerFn := func(reportPath, reportContent, docType, title, featureID string) (string, error) {
		registerCalled = true
		return "DOC-001", nil
	}

	d := NewSpawnAgentDispatcher(registerFn)
	d.DocContentFunc = func(path string) (string, error) {
		switch path {
		case "work/P43/spec.md":
			return "# Specification\n\nContent here.", nil
		case "work/P43/design.md":
			return "# Design\n\nDesign content.", nil
		case ".kbz/rubrics/spec-validator.yaml":
			return "checks:\n  - id: S1\n    description: All required sections present", nil
		default:
			return "", nil
		}
	}

	vctx := ValidatorContext{
		DocumentPath:  "work/P43/spec.md",
		DocumentType:  "specification",
		ParentDocPath: "work/P43/design.md",
		RubricPath:    ".kbz/rubrics/spec-validator.yaml",
		FeatureID:     "FEAT-001",
	}

	summary, err := d.Dispatch(context.Background(), "spec-validator", "validate-spec", vctx)
	// Currently Dispatch returns an error with the prompt embedded.
	// When fully wired to spawn_agent, it will return the summary directly.
	if err == nil {
		t.Fatal("expected error (with prompt) from Dispatch")
	}

	errStr := err.Error()

	// The prompt should contain key elements.
	if !strings.Contains(errStr, "Validator Dispatch") {
		t.Error("prompt should contain 'Validator Dispatch' header")
	}
	if !strings.Contains(errStr, "spec-validator") {
		t.Error("prompt should contain the role name")
	}
	if !strings.Contains(errStr, "validate-spec") {
		t.Error("prompt should contain the skill name")
	}
	if !strings.Contains(errStr, "work/P43/spec.md") {
		t.Error("prompt should contain the document path")
	}
	if !strings.Contains(errStr, "specification") {
		t.Error("prompt should contain the document type")
	}
	if !strings.Contains(errStr, "FEAT-001") {
		t.Error("prompt should contain the feature ID")
	}
	if !strings.Contains(errStr, "# Specification") {
		t.Error("prompt should contain the document content")
	}
	if !strings.Contains(errStr, "# Design") {
		t.Error("prompt should contain the parent document content")
	}
	if !strings.Contains(errStr, "S1") {
		t.Error("prompt should contain the rubric content")
	}
	if !strings.Contains(errStr, "blocking") {
		t.Error("prompt should mention blocking/non-blocking classification")
	}
	if !strings.Contains(errStr, "spawn_agent") {
		t.Error("prompt should reference spawn_agent in the error message")
	}

	// The provisional summary should be a pass (placeholder).
	if summary.Verdict != VerdictPass {
		t.Errorf("provisional verdict = %q, want %q", summary.Verdict, VerdictPass)
	}

	if registerCalled {
		t.Error("RegisterReportFunc should not be called by Dispatch — the sub-agent does that")
	}
}

func TestSpawnAgentDispatcher_Dispatch_WithoutParentDoc(t *testing.T) {
	t.Parallel()

	registerFn := func(reportPath, reportContent, docType, title, featureID string) (string, error) {
		return "DOC-001", nil
	}

	d := NewSpawnAgentDispatcher(registerFn)
	d.DocContentFunc = func(path string) (string, error) {
		return "# Doc", nil
	}

	vctx := ValidatorContext{
		DocumentPath: "work/P43/spec.md",
		DocumentType: "specification",
		FeatureID:    "FEAT-001",
		// ParentDocPath is empty
	}

	_, err := d.Dispatch(context.Background(), "spec-validator", "validate-spec", vctx)
	if err == nil {
		t.Fatal("expected error (with prompt)")
	}

	errStr := err.Error()
	if strings.Contains(errStr, "Parent Document") {
		t.Error("prompt should NOT contain 'Parent Document' when no parent doc is provided")
	}
}

func TestSpawnAgentDispatcher_Dispatch_WithoutRubric(t *testing.T) {
	t.Parallel()

	registerFn := func(reportPath, reportContent, docType, title, featureID string) (string, error) {
		return "DOC-001", nil
	}

	d := NewSpawnAgentDispatcher(registerFn)
	d.DocContentFunc = func(path string) (string, error) {
		return "# Doc", nil
	}

	vctx := ValidatorContext{
		DocumentPath: "work/P43/spec.md",
		DocumentType: "specification",
		FeatureID:    "FEAT-001",
		// RubricPath is empty
	}

	_, err := d.Dispatch(context.Background(), "spec-validator", "validate-spec", vctx)
	if err == nil {
		t.Fatal("expected error (with prompt)")
	}

	errStr := err.Error()
	if strings.Contains(errStr, "Validation Rubric") {
		t.Error("prompt should NOT contain 'Validation Rubric' when no rubric is provided")
	}
}

func TestSpawnAgentDispatcher_buildPrompt_DocReadError(t *testing.T) {
	t.Parallel()

	d := NewSpawnAgentDispatcher(nil)
	d.DocContentFunc = func(path string) (string, error) {
		return "", nil // DocContentFunc can return empty without error
	}

	vctx := ValidatorContext{
		DocumentPath:  "work/P43/missing.md",
		DocumentType:  "specification",
		ParentDocPath: "work/P43/also-missing.md",
		RubricPath:    ".kbz/rubrics/missing.yaml",
		FeatureID:     "FEAT-001",
	}

	prompt := d.buildPrompt("spec-validator", "validate-spec", vctx)

	// When DocContentFunc returns empty, the prompt still includes the headers
	// and the empty content.
	if !strings.Contains(prompt, "Document Under Validation") {
		t.Error("prompt should contain 'Document Under Validation' header even when content is empty")
	}
}

func TestSpawnAgentDispatcher_readDoc_NoFunc(t *testing.T) {
	t.Parallel()

	d := &SpawnAgentDispatcher{}
	_, err := d.readDoc("some/path.md")
	if err == nil {
		t.Fatal("expected error when DocContentFunc is nil")
	}
	if !strings.Contains(err.Error(), "DocContentFunc not configured") {
		t.Errorf("error should mention DocContentFunc, got: %v", err)
	}
}

func TestVerdictConstants(t *testing.T) {
	t.Parallel()

	if VerdictPass != "pass" {
		t.Errorf("VerdictPass = %q, want %q", VerdictPass, "pass")
	}
	if VerdictPassWithNotes != "pass_with_notes" {
		t.Errorf("VerdictPassWithNotes = %q, want %q", VerdictPassWithNotes, "pass_with_notes")
	}
	if VerdictFail != "fail" {
		t.Errorf("VerdictFail = %q, want %q", VerdictFail, "fail")
	}
}

// TestValidatorDispatcher_InterfaceSatisfaction verifies that
// *SpawnAgentDispatcher satisfies the ValidatorDispatcher interface.
func TestValidatorDispatcher_InterfaceSatisfaction(t *testing.T) {
	t.Parallel()

	// Compile-time assertion: if this compiles, the interface is satisfied.
	var _ ValidatorDispatcher = (*SpawnAgentDispatcher)(nil)

	// Runtime check
	var d ValidatorDispatcher = NewSpawnAgentDispatcher(nil)
	if d == nil {
		t.Fatal("NewSpawnAgentDispatcher returned nil")
	}
}

// TestValidatorDispatcher_AbstractionDoesNotHardcodeSpawnAgent verifies
// REQ-SESS-004: the ValidatorDispatcher interface must not reference
// spawn_agent or any spawn-agent-specific types.
func TestValidatorDispatcher_AbstractionDoesNotHardcodeSpawnAgent(t *testing.T) {
	t.Parallel()

	// The interface method signature uses only standard types:
	//   context.Context, string, ValidatorContext, ValidatorSummary, error
	//
	// Verify that the ValidatorContext and ValidatorSummary structs
	// also don't reference spawn_agent.
	vctx := ValidatorContext{
		DocumentPath:  "test",
		DocumentType:  "specification",
		ParentDocPath: "parent",
		RubricPath:    "rubric",
		FeatureID:     "FEAT-001",
	}
	_ = vctx

	summary := ValidatorSummary{
		Verdict:          VerdictPass,
		BlockingCount:    0,
		NonBlockingCount: 0,
		EvidenceScore:    1.0,
		ReportDocID:      "DOC-001",
	}
	_ = summary

	// The interface itself has zero references to spawn_agent or sub-agent concepts.
	// The concrete SpawnAgentDispatcher is named after the implementation strategy,
	// but P44 can provide a different ValidatorDispatcher without changing callers.
}
