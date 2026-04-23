package mcp

import (
	"strings"
	"testing"
	"time"

	"github.com/sambeau/kanbanzai/internal/service"
	"github.com/sambeau/kanbanzai/internal/storage"
)

// ─── Local helpers ────────────────────────────────────────────────────────────

// createActivePlan writes a plan record in "active" status directly, bypassing the plan lifecycle.
func createActivePlan(t *testing.T, entitySvc *service.EntityService, slug string) string {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	planID := "P1-" + slug
	record := storage.EntityRecord{
		Type: "plan",
		ID:   planID,
		Slug: slug,
		Fields: map[string]any{
			"id":         planID,
			"slug":       slug,
			"name":       "Test plan " + slug,
			"status":     "active",
			"summary":    "Test plan",
			"created":    now,
			"created_by": "tester",
			"updated":    now,
		},
	}
	if _, err := entitySvc.Store().Write(record); err != nil {
		t.Fatalf("createActivePlan(%s): %v", slug, err)
	}
	return planID
}

// assertPlanStatus reads a plan entity and asserts its status field.
func assertPlanStatus(t *testing.T, entitySvc *service.EntityService, planID, wantStatus string) {
	t.Helper()
	got, err := entitySvc.GetPlan(planID)
	if err != nil {
		t.Fatalf("GetPlan %s: %v", planID, err)
	}
	gotStatus, _ := got.State["status"].(string)
	if gotStatus != wantStatus {
		t.Errorf("plan %s status = %q, want %q", planID, gotStatus, wantStatus)
	}
}

// ─── Pillar A — Lifecycle Gate tests ─────────────────────────────────────────

// TestFeatureGate_BlockedByNonTerminalTask (AC-001) verifies that a terminal
// feature transition is blocked when at least one child task is non-terminal.
func TestFeatureGate_BlockedByNonTerminalTask(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createEntityTestPlan(t, entitySvc, "gate-blocked-by-task")
	featID := createEntityTestFeatureWithStatus(t, entitySvc, planID, "feat-blocked-by-task", "developing")
	taskID, _ := createEntityTestTask(t, entitySvc, featID, "blocked-task")
	transitionEntityStatus(t, entitySvc, "task", taskID, "ready")

	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action": "transition",
		"id":     featID,
		"status": "done",
	})

	errMsg, _ := result["error"].(string)
	if errMsg == "" {
		t.Fatalf("expected error field in gate failure response, got: %v", result)
	}
	if !strings.Contains(errMsg, "1 non-terminal task(s)") {
		t.Errorf("error message should contain '1 non-terminal task(s)', got: %s", errMsg)
	}

	gateFailed, ok := result["gate_failed"].(map[string]any)
	if !ok {
		t.Fatalf("expected gate_failed map in response, got: %v", result)
	}
	if gateFailed["from_status"] != "developing" {
		t.Errorf("gate_failed.from_status = %q, want %q", gateFailed["from_status"], "developing")
	}
	if gateFailed["to_status"] != "done" {
		t.Errorf("gate_failed.to_status = %q, want %q", gateFailed["to_status"], "done")
	}

	assertFeatureStatus(t, entitySvc, featID, "developing")
}

// TestFeatureGate_AllTasksDone (AC-002) verifies that a terminal feature
// transition succeeds when all child tasks are in terminal states.
func TestFeatureGate_AllTasksDone(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createEntityTestPlan(t, entitySvc, "gate-all-tasks-done")
	featID := createEntityTestFeatureWithStatus(t, entitySvc, planID, "feat-all-tasks-done", "developing")
	taskID, _ := createEntityTestTask(t, entitySvc, featID, "task-to-done")
	transitionEntityStatus(t, entitySvc, "task", taskID, "ready")
	transitionEntityStatus(t, entitySvc, "task", taskID, "active")
	transitionEntityStatus(t, entitySvc, "task", taskID, "needs-review")
	transitionEntityStatus(t, entitySvc, "task", taskID, "done")

	// Use "superseded" to avoid triggering the Phase 2 document gate.
	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action": "transition",
		"id":     featID,
		"status": "superseded",
	})

	if _, hasErr := result["error"]; hasErr {
		t.Fatalf("unexpected error when all tasks done: %v", result["error"])
	}
	assertFeatureStatus(t, entitySvc, featID, "superseded")
}

// TestFeatureGate_MixedTerminalStatuses (AC-003) verifies that a terminal
// transition succeeds when tasks are a mix of "done" and "not-planned".
func TestFeatureGate_MixedTerminalStatuses(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createEntityTestPlan(t, entitySvc, "gate-mixed-terminal")
	featID := createEntityTestFeatureWithStatus(t, entitySvc, planID, "feat-mixed-terminal", "developing")

	task1ID, _ := createEntityTestTask(t, entitySvc, featID, "task-mixed-done")
	transitionEntityStatus(t, entitySvc, "task", task1ID, "ready")
	transitionEntityStatus(t, entitySvc, "task", task1ID, "active")
	transitionEntityStatus(t, entitySvc, "task", task1ID, "needs-review")
	transitionEntityStatus(t, entitySvc, "task", task1ID, "done")

	task2ID, _ := createEntityTestTask(t, entitySvc, featID, "task-mixed-notplanned")
	transitionEntityStatus(t, entitySvc, "task", task2ID, "not-planned")

	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action": "transition",
		"id":     featID,
		"status": "superseded",
	})

	if _, hasErr := result["error"]; hasErr {
		t.Fatalf("unexpected error with mixed terminal tasks: %v", result["error"])
	}
	assertFeatureStatus(t, entitySvc, featID, "superseded")
}

// TestFeatureGate_BlockedOnSuperseded (AC-004) verifies the gate fires for
// the "superseded" target status when tasks are non-terminal.
func TestFeatureGate_BlockedOnSuperseded(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createEntityTestPlan(t, entitySvc, "gate-blocked-superseded")
	featID := createEntityTestFeatureWithStatus(t, entitySvc, planID, "feat-blocked-superseded", "developing")
	taskID, _ := createEntityTestTask(t, entitySvc, featID, "task-ready-s")
	transitionEntityStatus(t, entitySvc, "task", taskID, "ready")

	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action": "transition",
		"id":     featID,
		"status": "superseded",
	})

	errMsg, _ := result["error"].(string)
	if errMsg == "" {
		t.Fatalf("expected error for superseded with non-terminal task, got: %v", result)
	}
	if !strings.Contains(errMsg, "1 non-terminal task(s)") {
		t.Errorf("error should contain '1 non-terminal task(s)', got: %s", errMsg)
	}
	assertFeatureStatus(t, entitySvc, featID, "developing")
}

// TestFeatureGate_BlockedOnCancelled (AC-005) verifies the gate fires for
// the "cancelled" target status when tasks are non-terminal.
func TestFeatureGate_BlockedOnCancelled(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createEntityTestPlan(t, entitySvc, "gate-blocked-cancelled")
	featID := createEntityTestFeatureWithStatus(t, entitySvc, planID, "feat-blocked-cancelled", "developing")
	taskID, _ := createEntityTestTask(t, entitySvc, featID, "task-ready-c")
	transitionEntityStatus(t, entitySvc, "task", taskID, "ready")

	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action": "transition",
		"id":     featID,
		"status": "cancelled",
	})

	errMsg, _ := result["error"].(string)
	if errMsg == "" {
		t.Fatalf("expected error for cancelled with non-terminal task, got: %v", result)
	}
	if !strings.Contains(errMsg, "1 non-terminal task(s)") {
		t.Errorf("error should contain '1 non-terminal task(s)', got: %s", errMsg)
	}
	assertFeatureStatus(t, entitySvc, featID, "developing")
}

// TestPlanGate_BlockedByNonTerminalFeature (AC-006) verifies that a terminal
// plan transition is blocked when at least one child feature is non-terminal.
func TestPlanGate_BlockedByNonTerminalFeature(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createActivePlan(t, entitySvc, "gate-plan-blocked")
	// Feature in "developing" (non-terminal) parented to this plan.
	_ = createEntityTestFeatureWithStatus(t, entitySvc, planID, "feat-plan-blocked", "developing")

	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action": "transition",
		"id":     planID,
		"status": "done",
	})

	errMsg, _ := result["error"].(string)
	if errMsg == "" {
		t.Fatalf("expected error field in plan gate failure response, got: %v", result)
	}
	if !strings.Contains(errMsg, "1 non-terminal feature(s)") {
		t.Errorf("error message should contain '1 non-terminal feature(s)', got: %s", errMsg)
	}

	assertPlanStatus(t, entitySvc, planID, "active")
}

// TestFeatureGate_NoChildren (AC-007) verifies that a terminal feature
// transition succeeds when there are no child tasks at all.
func TestFeatureGate_NoChildren(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createEntityTestPlan(t, entitySvc, "gate-no-children")
	featID := createEntityTestFeatureWithStatus(t, entitySvc, planID, "feat-no-children", "developing")

	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action": "transition",
		"id":     featID,
		"status": "superseded",
	})

	if _, hasErr := result["error"]; hasErr {
		t.Fatalf("unexpected error with no child tasks: %v", result["error"])
	}
	assertFeatureStatus(t, entitySvc, featID, "superseded")
}

// ─── Pillar B — Auto-advance tests (entity transition path) ──────────────────

// findSideEffect returns the first side effect in the response with the given type,
// or nil if not found.
func findSideEffect(result map[string]any, sideEffectType string) map[string]any {
	sideEffects, _ := result["side_effects"].([]any)
	for _, se := range sideEffects {
		seMap, _ := se.(map[string]any)
		if seMap["type"] == sideEffectType {
			return seMap
		}
	}
	return nil
}

// TestFeatureAutoAdvance_LastTaskDone (AC-010) verifies that completing the
// last non-terminal task auto-advances the feature from developing to reviewing.
func TestFeatureAutoAdvance_LastTaskDone(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createEntityTestPlan(t, entitySvc, "auto-advance-last-task")
	featID := createEntityTestFeatureWithStatus(t, entitySvc, planID, "feat-auto-advance-last", "developing")

	// Task 1: advance to done via service layer.
	task1ID, _ := createEntityTestTask(t, entitySvc, featID, "task-aa-done-1")
	transitionEntityStatus(t, entitySvc, "task", task1ID, "ready")
	transitionEntityStatus(t, entitySvc, "task", task1ID, "active")
	transitionEntityStatus(t, entitySvc, "task", task1ID, "needs-review")
	transitionEntityStatus(t, entitySvc, "task", task1ID, "done")

	// Task 2: advance to needs-review via service layer.
	task2ID, _ := createEntityTestTask(t, entitySvc, featID, "task-aa-done-2")
	transitionEntityStatus(t, entitySvc, "task", task2ID, "ready")
	transitionEntityStatus(t, entitySvc, "task", task2ID, "active")
	transitionEntityStatus(t, entitySvc, "task", task2ID, "needs-review")

	// Completing task2 via the MCP tool should trigger the auto-advance.
	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action": "transition",
		"id":     task2ID,
		"status": "done",
	})

	if _, hasErr := result["error"]; hasErr {
		t.Fatalf("unexpected error transitioning task2 to done: %v", result["error"])
	}

	se := findSideEffect(result, "feature_auto_advanced")
	if se == nil {
		t.Fatalf("expected feature_auto_advanced side effect, got side_effects: %v", result["side_effects"])
	}
	if se["entity_id"] != featID {
		t.Errorf("feature_auto_advanced entity_id = %q, want %q", se["entity_id"], featID)
	}
	if se["from_status"] != "developing" {
		t.Errorf("feature_auto_advanced from_status = %q, want %q", se["from_status"], "developing")
	}
	if se["to_status"] != "reviewing" {
		t.Errorf("feature_auto_advanced to_status = %q, want %q", se["to_status"], "reviewing")
	}

	assertFeatureStatus(t, entitySvc, featID, "reviewing")
}

// TestFeatureAutoAdvance_FromNeedsRework (AC-011) verifies that auto-advance
// fires when a feature is in needs-rework and its last task completes.
func TestFeatureAutoAdvance_FromNeedsRework(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createEntityTestPlan(t, entitySvc, "auto-advance-rework")
	featID := createEntityTestFeatureWithStatus(t, entitySvc, planID, "feat-auto-advance-rework", "needs-rework")

	taskID, _ := createEntityTestTask(t, entitySvc, featID, "task-rework-done")
	transitionEntityStatus(t, entitySvc, "task", taskID, "ready")
	transitionEntityStatus(t, entitySvc, "task", taskID, "active")
	transitionEntityStatus(t, entitySvc, "task", taskID, "needs-review")

	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action": "transition",
		"id":     taskID,
		"status": "done",
	})

	if _, hasErr := result["error"]; hasErr {
		t.Fatalf("unexpected error: %v", result["error"])
	}

	se := findSideEffect(result, "feature_auto_advanced")
	if se == nil {
		t.Fatalf("expected feature_auto_advanced side effect, got side_effects: %v", result["side_effects"])
	}
	if se["from_status"] != "needs-rework" {
		t.Errorf("feature_auto_advanced from_status = %q, want %q", se["from_status"], "needs-rework")
	}
	if se["to_status"] != "reviewing" {
		t.Errorf("feature_auto_advanced to_status = %q, want %q", se["to_status"], "reviewing")
	}

	assertFeatureStatus(t, entitySvc, featID, "reviewing")
}

// TestFeatureAutoAdvance_NoGuardIfAllNotPlanned (AC-012) verifies that
// auto-advance does NOT fire when all tasks are not-planned (none done).
func TestFeatureAutoAdvance_NoGuardIfAllNotPlanned(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createEntityTestPlan(t, entitySvc, "auto-advance-not-planned")
	featID := createEntityTestFeatureWithStatus(t, entitySvc, planID, "feat-auto-advance-np", "developing")

	taskID, _ := createEntityTestTask(t, entitySvc, featID, "task-not-planned")

	// Transition task directly from queued to not-planned.
	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action": "transition",
		"id":     taskID,
		"status": "not-planned",
	})

	if _, hasErr := result["error"]; hasErr {
		t.Fatalf("unexpected error: %v", result["error"])
	}

	if se := findSideEffect(result, "feature_auto_advanced"); se != nil {
		t.Errorf("unexpected feature_auto_advanced side effect when all tasks not-planned: %v", se)
	}

	assertFeatureStatus(t, entitySvc, featID, "developing")
}

// TestFeatureAutoAdvance_DoesNotFireFromReviewing (AC-014) verifies that
// auto-advance does NOT fire when the feature is already in reviewing.
func TestFeatureAutoAdvance_DoesNotFireFromReviewing(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createEntityTestPlan(t, entitySvc, "auto-advance-reviewing")
	featID := createEntityTestFeatureWithStatus(t, entitySvc, planID, "feat-already-reviewing", "reviewing")

	taskID, _ := createEntityTestTask(t, entitySvc, featID, "task-reviewing-done")
	transitionEntityStatus(t, entitySvc, "task", taskID, "ready")
	transitionEntityStatus(t, entitySvc, "task", taskID, "active")
	transitionEntityStatus(t, entitySvc, "task", taskID, "needs-review")

	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action": "transition",
		"id":     taskID,
		"status": "done",
	})

	if _, hasErr := result["error"]; hasErr {
		t.Fatalf("unexpected error: %v", result["error"])
	}

	if se := findSideEffect(result, "feature_auto_advanced"); se != nil {
		t.Errorf("unexpected feature_auto_advanced side effect for feature already in reviewing: %v", se)
	}

	assertFeatureStatus(t, entitySvc, featID, "reviewing")
}

// TestPlanAutoAdvance_LastFeatureDone (AC-015) verifies that completing the
// last non-terminal feature auto-advances the plan from active to done.
func TestPlanAutoAdvance_LastFeatureDone(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createActivePlan(t, entitySvc, "plan-auto-advance-done")

	// Feature 1: advance to done via service (walk to reviewing, then done).
	// Avoid createEntityTestFeatureWithStatus(..., "done") which has a double-loop
	// that tries to re-walk an already-at-reviewing feature from the start.
	feat1ID := createEntityTestFeatureWithStatus(t, entitySvc, planID, "feat-plan-aa-done", "reviewing")
	transitionEntityStatus(t, entitySvc, "feature", feat1ID, "done")

	// Feature 2: in developing with no tasks — superseding it triggers plan advance.
	feat2ID := createEntityTestFeatureWithStatus(t, entitySvc, planID, "feat-plan-aa-dev", "developing")

	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action": "transition",
		"id":     feat2ID,
		"status": "superseded",
	})

	if _, hasErr := result["error"]; hasErr {
		t.Fatalf("unexpected error superseding feat2: %v", result["error"])
	}

	se := findSideEffect(result, "plan_auto_advanced")
	if se == nil {
		t.Fatalf("expected plan_auto_advanced side effect, got side_effects: %v", result["side_effects"])
	}
	if se["entity_id"] != planID {
		t.Errorf("plan_auto_advanced entity_id = %q, want %q", se["entity_id"], planID)
	}
	if se["to_status"] != "done" {
		t.Errorf("plan_auto_advanced to_status = %q, want %q", se["to_status"], "done")
	}

	assertPlanStatus(t, entitySvc, planID, "done")
}

// TestPlanAutoAdvance_NoGuardIfAllSuperseded (AC-016) verifies that plan
// auto-advance does NOT fire when all features are superseded (none done).
func TestPlanAutoAdvance_NoGuardIfAllSuperseded(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createActivePlan(t, entitySvc, "plan-no-guard-superseded")

	// Feature 1: developing with no tasks — supersede via entity tool first.
	feat1ID := createEntityTestFeatureWithStatus(t, entitySvc, planID, "feat-supersede-ng-1", "developing")
	resp1 := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action": "transition",
		"id":     feat1ID,
		"status": "superseded",
	})
	if _, hasErr := resp1["error"]; hasErr {
		t.Fatalf("unexpected error superseding feat1: %v", resp1["error"])
	}

	// Feature 2: developing with no tasks.
	feat2ID := createEntityTestFeatureWithStatus(t, entitySvc, planID, "feat-supersede-ng-2", "developing")

	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action": "transition",
		"id":     feat2ID,
		"status": "superseded",
	})

	if _, hasErr := result["error"]; hasErr {
		t.Fatalf("unexpected error superseding feat2: %v", result["error"])
	}

	if se := findSideEffect(result, "plan_auto_advanced"); se != nil {
		t.Errorf("unexpected plan_auto_advanced side effect when all features superseded: %v", se)
	}

	assertPlanStatus(t, entitySvc, planID, "active")
}

// TestFeatureAutoAdvance_FailureSurfacedAsWarning (AC-017) verifies that a
// failed auto-advance attempt is surfaced as a side effect warning without
// blocking the primary task transition.
func TestFeatureAutoAdvance_FailureSurfacedAsWarning(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createEntityTestPlan(t, entitySvc, "auto-advance-failure")
	featID := createEntityTestFeatureWithStatus(t, entitySvc, planID, "feat-aa-failure", "developing")
	taskID, taskSlug := createEntityTestTask(t, entitySvc, featID, "task-aa-failure")

	// Overwrite the task's parent_feature to a non-existent feature ID.
	rec, err := entitySvc.Store().Load("task", taskID, taskSlug)
	if err != nil {
		t.Fatalf("load task %s: %v", taskID, err)
	}
	rec.Fields["parent_feature"] = "FEAT-DOESNOTEXIST"
	if _, err := entitySvc.Store().Write(rec); err != nil {
		t.Fatalf("write task with broken parent_feature: %v", err)
	}

	// Advance the task to needs-review (parent_feature check is not part of
	// lifecycle validation, so these transitions succeed).
	transitionEntityStatus(t, entitySvc, "task", taskID, "ready")
	transitionEntityStatus(t, entitySvc, "task", taskID, "active")
	transitionEntityStatus(t, entitySvc, "task", taskID, "needs-review")

	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action": "transition",
		"id":     taskID,
		"status": "done",
	})

	// The task transition itself must succeed (no top-level error).
	if _, hasErr := result["error"]; hasErr {
		t.Fatalf("task transition should succeed even when auto-advance fails, got error: %v", result["error"])
	}

	// The auto-advance failure must be surfaced as a side effect warning.
	se := findSideEffect(result, "feature_auto_advanced")
	if se == nil {
		t.Fatalf("expected feature_auto_advanced side effect for failure warning, got side_effects: %v", result["side_effects"])
	}
	trigger, _ := se["trigger"].(string)
	if !strings.Contains(trigger, "auto-advance failed") {
		t.Errorf("feature_auto_advanced trigger should contain 'auto-advance failed', got: %q", trigger)
	}
}
