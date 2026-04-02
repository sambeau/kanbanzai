package mcp

import (
	"strings"
	"testing"
	"time"

	"github.com/sambeau/kanbanzai/internal/service"
	"github.com/sambeau/kanbanzai/internal/storage"
)

// ─── Gate enforcement integration tests ──────────────────────────────────────
//
// These tests exercise the full entityTransitionAction MCP handler path for
// gate enforcement, covering the requirements from FR-001 through FR-026.
// They use the existing test helpers from entity_tool_test.go.

// ─── Single-step gate failure ─────────────────────────────────────────────────

// TestGate_SingleStep_DesigningToSpecifying_NoDoc verifies that a single-step
// transition from designing to specifying fails when no design document exists.
func TestGate_SingleStep_DesigningToSpecifying_NoDoc(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createEntityTestPlan(t, entitySvc, "gate-d2s-nodoc")
	featID := createEntityTestFeatureWithStatus(t, entitySvc, planID, "feat-gate-d2s-nodoc", "designing")

	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action": "transition",
		"id":     featID,
		"status": "specifying",
	})

	// Gate failed: expect error and gate_failed fields.
	errMsg, _ := result["error"].(string)
	if errMsg == "" {
		t.Fatalf("expected error field in gate failure response, got: %v", result)
	}
	if !strings.Contains(errMsg, featID) {
		t.Errorf("error message does not contain feature ID %q: %s", featID, errMsg)
	}
	if !strings.Contains(errMsg, "designing") {
		t.Errorf("error message does not contain from-status 'designing': %s", errMsg)
	}
	if !strings.Contains(errMsg, "specifying") {
		t.Errorf("error message does not contain to-status 'specifying': %s", errMsg)
	}
	if !strings.Contains(errMsg, "To resolve:") {
		t.Errorf("error message missing 'To resolve:' section: %s", errMsg)
	}

	gateFailed, ok := result["gate_failed"].(map[string]any)
	if !ok {
		t.Fatalf("expected gate_failed map in response, got: %v", result)
	}
	if gateFailed["from_status"] != "designing" {
		t.Errorf("gate_failed.from_status = %q, want %q", gateFailed["from_status"], "designing")
	}
	if gateFailed["to_status"] != "specifying" {
		t.Errorf("gate_failed.to_status = %q, want %q", gateFailed["to_status"], "specifying")
	}

	// Feature must remain in designing.
	assertFeatureStatus(t, entitySvc, featID, "designing")
}

// TestGate_SingleStep_DesigningToSpecifying_WithDoc verifies that a single-step
// transition from designing to specifying succeeds when a design doc is present.
func TestGate_SingleStep_DesigningToSpecifying_WithDoc(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createEntityTestPlan(t, entitySvc, "gate-d2s-doc")
	featID := createEntityTestFeatureWithStatus(t, entitySvc, planID, "feat-gate-d2s-doc", "designing")

	setupApprovedDoc(t, docSvc, repoRoot, "work/design/gate-d2s.md", "design", featID)

	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action": "transition",
		"id":     featID,
		"status": "specifying",
	})

	// Gate satisfied: expect entity response (no error field).
	if _, hasErr := result["error"]; hasErr {
		t.Fatalf("unexpected error in gate-satisfied transition: %v", result["error"])
	}
	entity, _ := result["entity"].(map[string]any)
	if entity == nil {
		t.Fatalf("expected entity in response, got: %v", result)
	}
	assertFeatureStatus(t, entitySvc, featID, "specifying")
}

// TestGate_SingleStep_DevelopingToReviewing_NonTerminalTask verifies the
// developing→reviewing gate fails when child tasks are not all terminal.
func TestGate_SingleStep_DevelopingToReviewing_NonTerminalTask(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createEntityTestPlan(t, entitySvc, "gate-dev2rev-nonterminal")
	featID := createEntityTestFeatureWithStatus(t, entitySvc, planID, "feat-dev2rev-nt", "developing")
	// Create a non-terminal task.
	taskID, _ := createEntityTestTask(t, entitySvc, featID, "task-active")
	// Advance the task to active.
	transitionEntityStatus(t, entitySvc, "task", taskID, "ready")
	transitionEntityStatus(t, entitySvc, "task", taskID, "active")

	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action": "transition",
		"id":     featID,
		"status": "reviewing",
	})

	errMsg, _ := result["error"].(string)
	if errMsg == "" {
		t.Fatalf("expected error field for non-terminal task gate, got: %v", result)
	}
	// Error should mention the non-terminal task.
	if !strings.Contains(errMsg, taskID) {
		t.Errorf("error message should mention non-terminal task %q: %s", taskID, errMsg)
	}
	assertFeatureStatus(t, entitySvc, featID, "developing")
}

// TestGate_SingleStep_DevelopingToReviewing_AllTerminal verifies the gate passes.
func TestGate_SingleStep_DevelopingToReviewing_AllTerminal(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createEntityTestPlan(t, entitySvc, "gate-dev2rev-terminal")
	featID := createEntityTestFeatureWithStatus(t, entitySvc, planID, "feat-dev2rev-term", "developing")
	// Terminal task.
	taskID, _ := createEntityTestTask(t, entitySvc, featID, "task-done")
	transitionEntityStatus(t, entitySvc, "task", taskID, "ready")
	transitionEntityStatus(t, entitySvc, "task", taskID, "active")
	transitionEntityStatus(t, entitySvc, "task", taskID, "needs-review")
	transitionEntityStatus(t, entitySvc, "task", taskID, "done")

	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action": "transition",
		"id":     featID,
		"status": "reviewing",
	})

	if _, hasErr := result["error"]; hasErr {
		t.Fatalf("unexpected error for all-terminal gate: %v", result["error"])
	}
	assertFeatureStatus(t, entitySvc, featID, "reviewing")
}

// TestGate_SingleStep_ReviewingToDone_NoReport verifies reviewing→done gate.
func TestGate_SingleStep_ReviewingToDone_NoReport(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createEntityTestPlan(t, entitySvc, "gate-rev2done-noreport")
	featID := createEntityTestFeatureWithStatus(t, entitySvc, planID, "feat-rev2done-nr", "reviewing")

	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action": "transition",
		"id":     featID,
		"status": "done",
	})

	errMsg, _ := result["error"].(string)
	if errMsg == "" {
		t.Fatalf("expected error for reviewing→done without report: %v", result)
	}
	assertFeatureStatus(t, entitySvc, featID, "reviewing")
}

// TestGate_SingleStep_ReviewingToDone_WithReport verifies gate passes with report.
func TestGate_SingleStep_ReviewingToDone_WithReport(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createEntityTestPlan(t, entitySvc, "gate-rev2done-report")
	featID := createEntityTestFeatureWithStatus(t, entitySvc, planID, "feat-rev2done-rep", "reviewing")
	// Register a report (not necessarily approved).
	submitAndApproveTestDoc(t, docSvc, repoRoot, "work/reports/review.md", "report", featID, false)

	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action": "transition",
		"id":     featID,
		"status": "done",
	})

	if _, hasErr := result["error"]; hasErr {
		t.Fatalf("unexpected error with report registered: %v", result["error"])
	}
	assertFeatureStatus(t, entitySvc, featID, "done")
}

// ─── Terminal state transitions (ungated) ────────────────────────────────────

// TestGate_TerminalTransition_SupsededNotGated verifies that → superseded
// transitions are never subject to gate enforcement (FR-002).
func TestGate_TerminalTransition_SupersededNotGated(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createEntityTestPlan(t, entitySvc, "gate-superseded")
	featID := createEntityTestFeatureWithStatus(t, entitySvc, planID, "feat-supersede", "developing")

	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action": "transition",
		"id":     featID,
		"status": "superseded",
	})

	if _, hasErr := result["error"]; hasErr {
		t.Fatalf("→superseded should be ungated: %v", result["error"])
	}
	assertFeatureStatus(t, entitySvc, featID, "superseded")
}

// TestGate_TerminalTransition_CancelledNotGated verifies that → cancelled
// transitions are never gated (FR-002).
func TestGate_TerminalTransition_CancelledNotGated(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createEntityTestPlan(t, entitySvc, "gate-cancelled")
	featID := createEntityTestFeatureWithStatus(t, entitySvc, planID, "feat-cancel", "specifying")

	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action": "transition",
		"id":     featID,
		"status": "cancelled",
	})

	if _, hasErr := result["error"]; hasErr {
		t.Fatalf("→cancelled should be ungated: %v", result["error"])
	}
	assertFeatureStatus(t, entitySvc, featID, "cancelled")
}

// ─── Ungated transitions ─────────────────────────────────────────────────────

// TestGate_UngatedTransition_ProposedToDesigning verifies proposed→designing
// requires no gate prerequisite (FR-003).
func TestGate_UngatedTransition_ProposedToDesigning(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createEntityTestPlan(t, entitySvc, "gate-p2d")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-gate-p2d")

	// No documents needed.
	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action": "transition",
		"id":     featID,
		"status": "designing",
	})

	if _, hasErr := result["error"]; hasErr {
		t.Fatalf("proposed→designing should be ungated: %v", result["error"])
	}
	assertFeatureStatus(t, entitySvc, featID, "designing")
}

// TestGate_UngatedTransition_ReviewingToNeedsRework verifies reviewing→needs-rework
// is ungated (FR-003).
func TestGate_UngatedTransition_ReviewingToNeedsRework(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createEntityTestPlan(t, entitySvc, "gate-rev2rework")
	featID := createEntityTestFeatureWithStatus(t, entitySvc, planID, "feat-rev2rework", "reviewing")

	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action": "transition",
		"id":     featID,
		"status": "needs-rework",
	})

	if _, hasErr := result["error"]; hasErr {
		t.Fatalf("reviewing→needs-rework should be ungated: %v", result["error"])
	}
	assertFeatureStatus(t, entitySvc, featID, "needs-rework")
}

// ─── Phase 1 transitions (not gated) ─────────────────────────────────────────

// TestGate_Phase1Transition_NotGated verifies that Phase 1 feature transitions
// are not subject to Phase 2 gate enforcement (NFR-002).
func TestGate_Phase1Transition_NotGated(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createEntityTestPlan(t, entitySvc, "gate-phase1")
	// Phase 1 features start in "draft" — write directly to the store since
	// CreateFeature always starts in "proposed" (Phase 2 entry state).
	featID := createPhase1Feature(t, entitySvc, planID, "feat-phase1")

	// Phase 1 transition: draft → in-review (no docs required).
	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action": "transition",
		"id":     featID,
		"status": "in-review",
	})

	if _, hasErr := result["error"]; hasErr {
		t.Fatalf("Phase 1 transition should not be gated: %v", result["error"])
	}
	assertFeatureStatus(t, entitySvc, featID, "in-review")
}

// ─── Non-feature entities (not gated) ────────────────────────────────────────

// TestGate_NonFeatureEntity_NotGated verifies that task transitions are not
// subject to feature gate enforcement (NFR-002 / backward compat).
func TestGate_NonFeatureEntity_NotGated(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createEntityTestPlan(t, entitySvc, "gate-nonfeat")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-gate-nonfeat")
	taskID, _ := createEntityTestTask(t, entitySvc, featID, "task-gate-nonfeat")

	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action": "transition",
		"id":     taskID,
		"status": "ready",
	})

	if _, hasErr := result["error"]; hasErr {
		t.Fatalf("task transition should not trigger feature gate: %v", result["error"])
	}
}

// ─── Override mechanism ───────────────────────────────────────────────────────

// TestGate_Override_BypassesGate verifies that override=true with a reason
// bypasses a failing gate and the transition proceeds (FR-011).
func TestGate_Override_BypassesGate(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createEntityTestPlan(t, entitySvc, "gate-override-bypass")
	featID := createEntityTestFeatureWithStatus(t, entitySvc, planID, "feat-override-bypass", "designing")

	// No design doc — gate would fail without override.
	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action":          "transition",
		"id":              featID,
		"status":          "specifying",
		"override":        true,
		"override_reason": "design exists in external system",
	})

	// Override bypasses gate: expect success.
	if errMsg, hasErr := result["error"].(string); hasErr && errMsg != "" {
		t.Fatalf("override should bypass gate, got error: %s", errMsg)
	}
	entity, _ := result["entity"].(map[string]any)
	if entity == nil {
		t.Fatalf("expected entity in override response, got: %v", result)
	}
	assertFeatureStatus(t, entitySvc, featID, "specifying")
}

// TestGate_Override_LogsOverrideRecord verifies that a gate bypass is
// persisted as an OverrideRecord on the feature entity (FR-014).
func TestGate_Override_LogsOverrideRecord(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createEntityTestPlan(t, entitySvc, "gate-override-log")
	featID := createEntityTestFeatureWithStatus(t, entitySvc, planID, "feat-override-log", "designing")

	reason := "approved in external system"
	callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action":          "transition",
		"id":              featID,
		"status":          "specifying",
		"override":        true,
		"override_reason": reason,
	})

	// Read back the feature and verify the override record.
	getResult, err := entitySvc.Get("feature", featID, "")
	if err != nil {
		t.Fatalf("Get feature after override: %v", err)
	}

	overridesRaw, ok := getResult.State["overrides"]
	if !ok {
		t.Fatal("expected 'overrides' field in persisted feature state after override")
	}
	overrides, ok := overridesRaw.([]any)
	if !ok || len(overrides) == 0 {
		t.Fatalf("expected non-empty overrides in state, got %T %v", overridesRaw, overridesRaw)
	}

	override, ok := overrides[0].(map[string]any)
	if !ok {
		t.Fatalf("override[0] should be map[string]any, got %T", overrides[0])
	}
	if override["from_status"] != "designing" {
		t.Errorf("override.from_status = %q, want %q", override["from_status"], "designing")
	}
	if override["to_status"] != "specifying" {
		t.Errorf("override.to_status = %q, want %q", override["to_status"], "specifying")
	}
	if override["reason"] != reason {
		t.Errorf("override.reason = %q, want %q", override["reason"], reason)
	}
	if override["timestamp"] == "" {
		t.Error("override.timestamp should be non-empty")
	}
}

// TestGate_Override_RequiresReason verifies that override=true without
// override_reason is rejected with a validation error (FR-012).
func TestGate_Override_RequiresReason(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createEntityTestPlan(t, entitySvc, "gate-override-noreason")
	featID := createEntityTestFeatureWithStatus(t, entitySvc, planID, "feat-override-noreason", "designing")

	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action":   "transition",
		"id":       featID,
		"status":   "specifying",
		"override": true,
		// override_reason intentionally omitted
	})

	errMsg, _ := result["error"].(string)
	if errMsg == "" {
		t.Fatal("expected error when override=true but override_reason is empty")
	}
	if !strings.Contains(strings.ToLower(errMsg), "override_reason") {
		t.Errorf("error message should mention override_reason requirement: %s", errMsg)
	}
	// Feature must remain unchanged.
	assertFeatureStatus(t, entitySvc, featID, "designing")
}

// TestGate_Override_EmptyReasonRejected verifies that override=true with an
// empty string override_reason is rejected (FR-012).
func TestGate_Override_EmptyReasonRejected(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createEntityTestPlan(t, entitySvc, "gate-override-emptyreason")
	featID := createEntityTestFeatureWithStatus(t, entitySvc, planID, "feat-override-empty", "designing")

	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action":          "transition",
		"id":              featID,
		"status":          "specifying",
		"override":        true,
		"override_reason": "",
	})

	errMsg, _ := result["error"].(string)
	if errMsg == "" {
		t.Fatal("expected error when override_reason is empty string")
	}
	assertFeatureStatus(t, entitySvc, featID, "designing")
}

// TestGate_Override_ReasonIgnoredWithoutOverrideFlag verifies that
// override_reason alone (without override=true) does not bypass gates (FR-013).
func TestGate_Override_ReasonIgnoredWithoutOverrideFlag(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createEntityTestPlan(t, entitySvc, "gate-override-noreason-flag")
	featID := createEntityTestFeatureWithStatus(t, entitySvc, planID, "feat-override-noflag", "designing")

	// override_reason provided but override=false (default).
	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action":          "transition",
		"id":              featID,
		"status":          "specifying",
		"override_reason": "this reason should be ignored without override=true",
	})

	// Gate should still fire and fail (no design doc).
	errMsg, _ := result["error"].(string)
	if errMsg == "" {
		t.Fatal("gate should still fail when override=false even with override_reason set")
	}
	if _, hasGateFailed := result["gate_failed"]; !hasGateFailed {
		t.Error("expected gate_failed field in response")
	}
	assertFeatureStatus(t, entitySvc, featID, "designing")
}

// ─── Advance with override ────────────────────────────────────────────────────

// TestGate_Advance_WithOverride verifies that advance+override bypasses all
// gate failures along the path and reports overridden gates (FR-016).
func TestGate_Advance_WithOverride(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createEntityTestPlan(t, entitySvc, "gate-adv-override")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-adv-override")

	// No documents, no tasks — all gates will fail without override.
	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action":          "transition",
		"id":              featID,
		"status":          "developing",
		"advance":         true,
		"override":        true,
		"override_reason": "fast-tracking for demo",
	})

	status, _ := result["status"].(string)
	if status != "developing" {
		t.Errorf("advance with override should reach developing, got status=%q; result=%v", status, result)
	}

	// overridden_gates should list each bypassed gate.
	overriddenGates, _ := result["overridden_gates"].([]any)
	if len(overriddenGates) == 0 {
		t.Error("expected overridden_gates to be non-empty in advance+override response")
	}
}

// TestGate_Advance_WithoutOverride_StopsAtFirstGate verifies that advance
// without override stops at the first failing gate (FR-001).
func TestGate_Advance_WithoutOverride_StopsAtFirstGate(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createEntityTestPlan(t, entitySvc, "gate-adv-stop")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-adv-stop2")

	// No documents — advance should enter designing (ungated) then stop.
	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action":  "transition",
		"id":      featID,
		"status":  "developing",
		"advance": true,
	})

	status, _ := result["status"].(string)
	if status != "designing" {
		t.Errorf("advance without docs should stop at designing (after ungated proposed→designing), got %q", status)
	}
	stoppedReason, _ := result["stopped_reason"].(string)
	if stoppedReason == "" {
		t.Error("expected non-empty stopped_reason")
	}
}

// ─── Consistency: single-step vs advance produce same gate evaluation ─────────

// TestGate_Consistency_SingleStepVsAdvance verifies that the same gate failure
// occurs whether invoked as single-step or advance (FR-001 acceptance criteria).
func TestGate_Consistency_SingleStepVsAdvance(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createEntityTestPlan(t, entitySvc, "gate-consistency")

	// Feature A: single-step transition designing→specifying (no doc).
	featA := createEntityTestFeatureWithStatus(t, entitySvc, planID, "feat-consist-a", "designing")
	singleResult := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action": "transition",
		"id":     featA,
		"status": "specifying",
	})

	// Feature B: advance from designing toward specifying (no doc).
	featB := createEntityTestFeatureWithStatus(t, entitySvc, planID, "feat-consist-b", "designing")
	advResult := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action":  "transition",
		"id":      featB,
		"status":  "specifying",
		"advance": true,
	})

	// Both should fail with a gate error.
	singleErr, _ := singleResult["error"].(string)
	if singleErr == "" {
		t.Fatal("single-step should fail gate")
	}
	advStoppedReason, _ := advResult["stopped_reason"].(string)
	if advStoppedReason == "" {
		t.Fatal("advance should report stopped_reason on gate failure")
	}

	// Both features remain in designing.
	assertFeatureStatus(t, entitySvc, featA, "designing")
	assertFeatureStatus(t, entitySvc, featB, "designing")
}

// ─── Advance override_reason validation ──────────────────────────────────────

// TestGate_Advance_OverrideRequiresReason verifies that advance=true combined
// with override=true but an empty override_reason is rejected with an error
// response — no panic, no nil error, and the feature remains unchanged (B-02).
func TestGate_Advance_OverrideRequiresReason(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createEntityTestPlan(t, entitySvc, "gate-adv-noreason")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-adv-noreason")

	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action":   "transition",
		"id":       featID,
		"status":   "developing",
		"advance":  true,
		"override": true,
		// override_reason intentionally omitted — must be rejected.
	})

	errMsg, _ := result["error"].(string)
	if errMsg == "" {
		t.Fatal("expected error when advance=true, override=true but override_reason is missing")
	}
	if !strings.Contains(strings.ToLower(errMsg), "override_reason") {
		t.Errorf("error message should mention override_reason requirement: %s", errMsg)
	}
	// Feature must remain at proposed — the advance was rejected before it started.
	assertFeatureStatus(t, entitySvc, featID, "proposed")
}

// ─── Test helpers ─────────────────────────────────────────────────────────────

// createEntityTestFeatureWithStatus creates a feature and transitions it to
// the given status via valid lifecycle transitions.
func createEntityTestFeatureWithStatus(t *testing.T, entitySvc *service.EntityService, planID, slug, status string) string {
	t.Helper()

	featID := createEntityTestFeature(t, entitySvc, planID, slug)

	forwardPath := []string{
		"proposed", "designing", "specifying", "dev-planning", "developing", "reviewing",
	}
	phase2Terminal := map[string]bool{
		"superseded": true,
		"cancelled":  true,
	}

	if status == "proposed" {
		return featID
	}

	if phase2Terminal[status] {
		transitionEntityStatus(t, entitySvc, "feature", featID, status)
		return featID
	}

	if status == "needs-rework" {
		for _, s := range forwardPath[1:] {
			transitionEntityStatus(t, entitySvc, "feature", featID, s)
			if s == "reviewing" {
				break
			}
		}
		transitionEntityStatus(t, entitySvc, "feature", featID, "needs-rework")
		return featID
	}

	for _, s := range forwardPath[1:] {
		transitionEntityStatus(t, entitySvc, "feature", featID, s)
		if s == status {
			return featID
		}
	}

	if status == "done" {
		for _, s := range forwardPath[1:] {
			transitionEntityStatus(t, entitySvc, "feature", featID, s)
		}
		transitionEntityStatus(t, entitySvc, "feature", featID, "done")
		return featID
	}

	t.Fatalf("createEntityTestFeatureWithStatus: cannot reach Phase 2 status %q", status)
	return featID
}

// createPhase1Feature writes a feature entity directly in "draft" (Phase 1 entry)
// state to the store, bypassing CreateFeature which always starts at "proposed".
// Uses a deterministic ULID-format ID so that ResolvePrefix can locate it.
func createPhase1Feature(t *testing.T, entitySvc *service.EntityService, planID, slug string) string {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	// FEAT- + 13 uppercase alphanumeric chars = valid ULID-format feature ID.
	featID := "FEAT-01GATEPHASE1"
	record := storage.EntityRecord{
		Type: "feature",
		ID:   featID,
		Slug: slug,
		Fields: map[string]any{
			"id":         featID,
			"slug":       slug,
			"parent":     planID,
			"status":     "draft",
			"summary":    "Phase 1 test feature",
			"created":    now,
			"created_by": "tester",
		},
	}
	if _, err := entitySvc.Store().Write(record); err != nil {
		t.Fatalf("createPhase1Feature(%s): %v", slug, err)
	}
	return featID
}

// transitionEntityStatus transitions an entity to a new status via UpdateStatus.
func transitionEntityStatus(t *testing.T, entitySvc *service.EntityService, entityType, entityID, newStatus string) {
	t.Helper()
	_, err := entitySvc.UpdateStatus(service.UpdateStatusInput{
		Type:   entityType,
		ID:     entityID,
		Status: newStatus,
	})
	if err != nil {
		t.Fatalf("transitionEntityStatus(%s %s → %s): %v", entityType, entityID, newStatus, err)
	}
}

// assertFeatureStatus reads a feature entity from disk and asserts its status.
func assertFeatureStatus(t *testing.T, entitySvc *service.EntityService, featID, wantStatus string) {
	t.Helper()
	got, err := entitySvc.Get("feature", featID, "")
	if err != nil {
		t.Fatalf("Get feature %s: %v", featID, err)
	}
	gotStatus, _ := got.State["status"].(string)
	if gotStatus != wantStatus {
		t.Errorf("feature %s status = %q, want %q", featID, gotStatus, wantStatus)
	}
}
