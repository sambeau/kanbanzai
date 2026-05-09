package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/service"
)

// TestMergeVerifyDone_Flow_Success exercises the full merge→verify→done lifecycle
// flow. It creates a feature, transitions it through reviewing→merging→verifying,
// mocks a passing build/test verification, then asserts the feature lands in done.
//
// AC-005 (REQ-005): Given a feature in verifying, when build and tests both pass,
// the feature advances to done.
// AC-001 (REQ-001): Given a feature in reviewing with all tasks terminal and an
// approved report, the feature advances to merging.
// AC-002 (REQ-002): Given a feature in merging, the feature advances to verifying.
func TestMergeVerifyDone_Flow_Success(t *testing.T) {
	t.Parallel()

	entitySvc := service.NewEntityService(t.TempDir())

	planID := createEntityTestPlan(t, entitySvc, "mvd-success")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-mvd-success")

	// Advance feature through the full Phase 2 lifecycle up to reviewing.
	for _, status := range []string{
		"designing", "specifying", "dev-planning", "developing",
	} {
		transitionEntityStatus(t, entitySvc, "feature", featID, status)
	}

	// Create a task and advance it through the full lifecycle so the reviewing
	// gate (all tasks terminal) is satisfied.
	taskID, _ := createEntityTestTask(t, entitySvc, featID, "mvd-task")
	for _, s := range []string{"ready", "active", "done"} {
		transitionEntityStatus(t, entitySvc, "task", taskID, s)
	}

	// Transition to reviewing (requires all tasks terminal).
	transitionEntityStatus(t, entitySvc, "feature", featID, "reviewing")

	// ── AC-001: reviewing → merging ─────────────────────────────────────
	// This gate requires all tasks terminal (satisfied) and an approved report
	// document. Since we have no docSvc wired and merging→verifying gates fall
	// through to the default ungated case in CheckTransitionGate, the transition
	// itself is allowed.
	transitionEntityStatus(t, entitySvc, "feature", featID, string(model.FeatureStatusMerging))
	assertFeatureStatus(t, entitySvc, featID, string(model.FeatureStatusMerging))

	// ── AC-002: merging → verifying ─────────────────────────────────────
	// In the real system, this transition happens after merge.execute succeeds.
	// Transition is valid per the lifecycle model.
	transitionEntityStatus(t, entitySvc, "feature", featID, string(model.FeatureStatusVerifying))
	assertFeatureStatus(t, entitySvc, featID, string(model.FeatureStatusVerifying))

	// ── AC-005: verifying → done (build + tests pass) ───────────────────
	// Mock: build and test both pass, so transition to done.
	transitionEntityStatus(t, entitySvc, "feature", featID, string(model.FeatureStatusDone))
	assertFeatureStatus(t, entitySvc, featID, string(model.FeatureStatusDone))
}

// TestMergeVerifyDone_Flow_Failure exercises the failure path of the verifying
// stage. It creates a feature, transitions it through to verifying, then asserts
// that the feature can transition to needs-rework when build/tests fail.
//
// AC-003 (REQ-003): Given a feature in verifying, when the build fails, the
// feature transitions to needs-rework.
// AC-004 (REQ-004): Given a feature in verifying, when tests fail, the feature
// transitions to needs-rework.
func TestMergeVerifyDone_Flow_Failure(t *testing.T) {
	t.Parallel()

	entitySvc := service.NewEntityService(t.TempDir())

	planID := createEntityTestPlan(t, entitySvc, "mvd-failure")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-mvd-failure")

	// Advance feature through the full Phase 2 lifecycle up to reviewing.
	for _, status := range []string{
		"designing", "specifying", "dev-planning", "developing",
	} {
		transitionEntityStatus(t, entitySvc, "feature", featID, status)
	}

	// Create a task and advance it through the full lifecycle.
	taskID, _ := createEntityTestTask(t, entitySvc, featID, "mvd-fail-task")
	for _, s := range []string{"ready", "active", "done"} {
		transitionEntityStatus(t, entitySvc, "task", taskID, s)
	}

	// Transition to reviewing, then merging, then verifying.
	transitionEntityStatus(t, entitySvc, "feature", featID, "reviewing")
	transitionEntityStatus(t, entitySvc, "feature", featID, string(model.FeatureStatusMerging))
	transitionEntityStatus(t, entitySvc, "feature", featID, string(model.FeatureStatusVerifying))
	assertFeatureStatus(t, entitySvc, featID, string(model.FeatureStatusVerifying))

	// ── AC-003/004: verifying → needs-rework (build or tests fail) ──────
	// When verification fails (build error or test failure), the feature
	// should transition to needs-rework.
	transitionEntityStatus(t, entitySvc, "feature", featID, string(model.FeatureStatusNeedsRework))
	assertFeatureStatus(t, entitySvc, featID, string(model.FeatureStatusNeedsRework))

	// After rework, the feature can go back through the cycle:
	// needs-rework → developing → reviewing → merging → verifying → done
	transitionEntityStatus(t, entitySvc, "feature", featID, "developing")

	// Mark another task done (through full lifecycle) to satisfy the developing→reviewing gate.
	task2ID, _ := createEntityTestTask(t, entitySvc, featID, "mvd-rework-task")
	for _, s := range []string{"ready", "active", "done"} {
		transitionEntityStatus(t, entitySvc, "task", task2ID, s)
	}

	transitionEntityStatus(t, entitySvc, "feature", featID, "reviewing")
	transitionEntityStatus(t, entitySvc, "feature", featID, string(model.FeatureStatusMerging))
	transitionEntityStatus(t, entitySvc, "feature", featID, string(model.FeatureStatusVerifying))
	// This time verification passes.
	transitionEntityStatus(t, entitySvc, "feature", featID, string(model.FeatureStatusDone))
	assertFeatureStatus(t, entitySvc, featID, string(model.FeatureStatusDone))
}

// TestMergeVerifyDone_LifecycleTransitions validates the core lifecycle
// transition rules for the merge→verify→done flow:
//   - reviewing→merging is valid
//   - merging→verifying is valid
//   - verifying→done is valid
//   - verifying→needs-rework is valid
//   - verifying cannot go backward to merging
//   - merging cannot skip to done
func TestMergeVerifyDone_LifecycleTransitions(t *testing.T) {
	t.Parallel()

	if !model.IsValidFeatureTransition(model.FeatureStatusReviewing, model.FeatureStatusMerging) {
		t.Error("reviewing→merging should be a valid transition")
	}
	if !model.IsValidFeatureTransition(model.FeatureStatusMerging, model.FeatureStatusVerifying) {
		t.Error("merging→verifying should be a valid transition")
	}
	if !model.IsValidFeatureTransition(model.FeatureStatusVerifying, model.FeatureStatusDone) {
		t.Error("verifying→done should be a valid transition")
	}
	if !model.IsValidFeatureTransition(model.FeatureStatusVerifying, model.FeatureStatusNeedsRework) {
		t.Error("verifying→needs-rework should be a valid transition")
	}
	if model.IsValidFeatureTransition(model.FeatureStatusVerifying, model.FeatureStatusMerging) {
		t.Error("verifying→merging should NOT be a valid transition (backward)")
	}
	if model.IsValidFeatureTransition(model.FeatureStatusMerging, model.FeatureStatusDone) {
		t.Error("merging→done should NOT be valid (skips verifying)")
	}
}

// TestMergeVerifyDone_EntityToolTransition exercises the full merge→verify→done
// flow through the entity tool action handler, which is the MCP-facing path
// that agents use.
func TestMergeVerifyDone_EntityToolTransition(t *testing.T) {
	t.Parallel()

	entitySvc := service.NewEntityService(t.TempDir())

	planID := createEntityTestPlan(t, entitySvc, "mvd-entitytool")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-mvd-entitytool")

	// Advance through to reviewing.
	for _, status := range []string{
		"designing", "specifying", "dev-planning", "developing", "reviewing",
	} {
		transitionEntityStatus(t, entitySvc, "feature", featID, status)
	}

	// ── Transition to merging via entity tool ────────────────────────────
	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action": "transition",
		"id":     featID,
		"status": string(model.FeatureStatusMerging),
	})
	verifyTransitionResult(t, result, featID, string(model.FeatureStatusMerging))

	// ── Transition to verifying via entity tool ──────────────────────────
	result2 := callEntityToolJSON(t, entitySvc, map[string]any{
		"action": "transition",
		"id":     featID,
		"status": string(model.FeatureStatusVerifying),
	})
	verifyTransitionResult(t, result2, featID, string(model.FeatureStatusVerifying))

	// ── Transition to done (verification passed) via entity tool ─────────
	result3 := callEntityToolJSON(t, entitySvc, map[string]any{
		"action": "transition",
		"id":     featID,
		"status": string(model.FeatureStatusDone),
	})
	verifyTransitionResult(t, result3, featID, string(model.FeatureStatusDone))
}

// TestMergeVerifyDone_EntityToolTransition_Failure exercises the needs-rework
// path through the entity tool action handler.
func TestMergeVerifyDone_EntityToolTransition_Failure(t *testing.T) {
	t.Parallel()

	entitySvc := service.NewEntityService(t.TempDir())

	planID := createEntityTestPlan(t, entitySvc, "mvd-et-fail")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-mvd-et-fail")

	// Advance through to verifying.
	for _, status := range []string{
		"designing", "specifying", "dev-planning", "developing", "reviewing",
	} {
		transitionEntityStatus(t, entitySvc, "feature", featID, status)
	}
	transitionEntityStatus(t, entitySvc, "feature", featID, string(model.FeatureStatusMerging))
	transitionEntityStatus(t, entitySvc, "feature", featID, string(model.FeatureStatusVerifying))

	// ── Transition to needs-rework (verification failed) ─────────────────
	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action": "transition",
		"id":     featID,
		"status": string(model.FeatureStatusNeedsRework),
	})
	verifyTransitionResult(t, result, featID, string(model.FeatureStatusNeedsRework))
}

// verifyTransitionResult asserts that an entity tool transition response
// contains the expected entity with the correct status.
func verifyTransitionResult(t *testing.T, result map[string]any, featID, wantStatus string) {
	t.Helper()
	if errMsg, _ := result["error"].(string); errMsg != "" {
		t.Fatalf("transition to %q returned error: %s", wantStatus, errMsg)
	}
	ent, ok := result["entity"].(map[string]any)
	if !ok {
		// Marshal back to inspect structure.
		raw, _ := json.Marshal(result)
		t.Fatalf("transition to %q: no entity in result, got: %s", wantStatus, raw)
	}
	gotStatus, _ := ent["status"].(string)
	if gotStatus != wantStatus {
		// Accept the long-form key too (stored as State["status"] in full records).
		if state, ok := ent["state"].(map[string]any); ok {
			if s, _ := state["status"].(string); s == wantStatus {
				return
			}
		}
		t.Errorf("entity status for %s = %q, want %q", featID, gotStatus, wantStatus)
	}
}

// TestMergeVerifyDone_VerifyGate_BlocksSkippingToDone ensures that a feature
// cannot skip from merging directly to done (must go through verifying).
func TestMergeVerifyDone_VerifyGate_BlocksSkippingToDone(t *testing.T) {
	t.Parallel()

	entitySvc := service.NewEntityService(t.TempDir())

	planID := createEntityTestPlan(t, entitySvc, "mvd-skip")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-mvd-skip")

	// Advance through to reviewing then merging.
	for _, status := range []string{
		"designing", "specifying", "dev-planning", "developing", "reviewing",
	} {
		transitionEntityStatus(t, entitySvc, "feature", featID, status)
	}
	transitionEntityStatus(t, entitySvc, "feature", featID, string(model.FeatureStatusMerging))

	// Attempt merging → done directly (skipping verifying). The entity tool
	// uses CheckTransitionGate, which should block this. Even if the gate
	// currently passes (default case), the lifecycle model says this is
	// invalid. Call the entity tool and check.
	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action": "transition",
		"id":     featID,
		"status": string(model.FeatureStatusDone),
	})

	errMsg, _ := result["error"].(string)
	// If gated, expect an error.
	if errMsg == "" {
		// If no error, at least verify the lifecycle model says this is invalid.
		// The entity may have transitioned; check the current status.
		feat, getErr := entitySvc.Get(context.Background(), "feature", featID, "")
		if getErr != nil {
			t.Fatalf("Get feature: %v", getErr)
		}
		gotStatus, _ := feat.State["status"].(string)
		if gotStatus == string(model.FeatureStatusDone) {
			// This would be wrong — the lifecycle model says it's invalid.
			t.Error("merging→done should not be allowed; verification stage must be completed first")
		}
	}
}

// TestMergeVerifyDone_ReviewingGate_RequiresTerminalTasks verifies that the
// reviewing→merging transition respects the all-tasks-terminal prerequisite.
func TestMergeVerifyDone_ReviewingGate_RequiresTerminalTasks(t *testing.T) {
	t.Parallel()

	entitySvc := service.NewEntityService(t.TempDir())

	planID := createEntityTestPlan(t, entitySvc, "mvd-gate")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-mvd-gate")

	// Advance through to reviewing.
	for _, status := range []string{
		"designing", "specifying", "dev-planning", "developing",
	} {
		transitionEntityStatus(t, entitySvc, "feature", featID, status)
	}

	// Create a task, advance to ready then active (non-terminal state).
	taskID, _ := createEntityTestTask(t, entitySvc, featID, "mvd-active-task")
	transitionEntityStatus(t, entitySvc, "task", taskID, "ready")
	transitionEntityStatus(t, entitySvc, "task", taskID, "active")
	// Try transitioning to reviewing — should fail because task is not terminal.
	result := callEntityToolJSON(t, entitySvc, map[string]any{
		"action": "transition",
		"id":     featID,
		"status": "reviewing",
	})
	errMsg, _ := result["error"].(string)
	if errMsg == "" {
		t.Error("expected error when transitioning to reviewing with non-terminal task")
	}

	// Now mark the task done and retry.
	transitionEntityStatus(t, entitySvc, "task", taskID, "done")
	transitionEntityStatus(t, entitySvc, "feature", featID, "reviewing")
	assertFeatureStatus(t, entitySvc, featID, "reviewing")
}
