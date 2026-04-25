package mcp

// Tests added for AC-001 through AC-009 (next tool UX improvements).
// Tests for AC-006, AC-007, AC-008 already exist in next_tool_test.go.

import (
	"encoding/json"
	"testing"
)

// TestNextClaimMode_AlreadyActive_ReturnsContextPacket verifies that claiming
// an already-active task returns a full context packet with reclaimed: true
// (FR-001, FR-002 — AC-001).
func TestNextClaimMode_AlreadyActive_ReturnsContextPacket(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupNextTest(t)

	planID := createNextTestPlan(t, entitySvc, "ac001-plan")
	featID := createNextTestFeature(t, entitySvc, planID, "ac001-feat")
	advanceNextFeatureTo(t, entitySvc, featID, "developing")
	taskID, taskSlug := createNextTestTask(t, entitySvc, featID, "ac001-task")
	setNextTaskReady(t, entitySvc, taskID, taskSlug)

	// First claim: ready → active.
	callNext(t, entitySvc, dispatchSvc, map[string]any{"id": taskID})

	// Reclaim: task is already active.
	raw := callNext(t, entitySvc, dispatchSvc, map[string]any{"id": taskID})
	var result map[string]any
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("parse reclaim result: %v", err)
	}

	// Must not be an error response.
	if _, hasErr := result["error"]; hasErr {
		t.Fatalf("expected success on reclaim, got error: %s", raw)
	}
	// reclaimed: true must be present (FR-002).
	reclaimed, _ := result["reclaimed"].(bool)
	if !reclaimed {
		t.Errorf("expected reclaimed: true, got: %s", raw)
	}
	// Full context packet: task and context must both be present (FR-001).
	if result["task"] == nil {
		t.Error("expected task field in reclaim response")
	}
	if result["context"] == nil {
		t.Error("expected context field in reclaim response")
	}
}

// TestNextClaimMode_AlreadyActive_PreservesDispatchMeta verifies that the
// dispatch metadata set on first claim (dispatched_to, claimed_at) is
// unchanged after a reclaim — i.e. the task is not re-dispatched (AC-002).
func TestNextClaimMode_AlreadyActive_PreservesDispatchMeta(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupNextTest(t)

	planID := createNextTestPlan(t, entitySvc, "ac002-plan")
	featID := createNextTestFeature(t, entitySvc, planID, "ac002-feat")
	advanceNextFeatureTo(t, entitySvc, featID, "developing")
	taskID, taskSlug := createNextTestTask(t, entitySvc, featID, "ac002-task")
	setNextTaskReady(t, entitySvc, taskID, taskSlug)

	// First claim.
	callNext(t, entitySvc, dispatchSvc, map[string]any{"id": taskID})

	// Record dispatch metadata from the entity store after first claim.
	afterFirst, err := entitySvc.Get("task", taskID, taskSlug)
	if err != nil {
		t.Fatalf("get task after first claim: %v", err)
	}
	dispatchedTo, _ := afterFirst.State["dispatched_to"].(string)
	claimedAt, _ := afterFirst.State["claimed_at"].(string)
	if dispatchedTo == "" {
		t.Fatal("dispatched_to not set after first claim")
	}
	if claimedAt == "" {
		t.Fatal("claimed_at not set after first claim")
	}

	// Reclaim.
	raw := callNext(t, entitySvc, dispatchSvc, map[string]any{"id": taskID})
	var result map[string]any
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("parse reclaim result: %v", err)
	}
	if _, hasErr := result["error"]; hasErr {
		t.Fatalf("expected success on reclaim, got error: %s", raw)
	}

	// Dispatch metadata must be unchanged after reclaim.
	afterReclaim, err := entitySvc.Get("task", taskID, taskSlug)
	if err != nil {
		t.Fatalf("get task after reclaim: %v", err)
	}
	if afterReclaim.State["dispatched_to"] != dispatchedTo {
		t.Errorf("dispatched_to changed: was %q, now %q", dispatchedTo, afterReclaim.State["dispatched_to"])
	}
	if afterReclaim.State["claimed_at"] != claimedAt {
		t.Errorf("claimed_at changed: was %q, now %q", claimedAt, afterReclaim.State["claimed_at"])
	}
}

// TestNextClaimMode_AlreadyActive_NoHookRefired verifies that reclaiming an
// active task does not re-invoke the dispatch hook — proved by (a) the
// dispatched_at timestamp being unchanged, and (b) no side effects being emitted
// (AC-003).
func TestNextClaimMode_AlreadyActive_NoHookRefired(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupNextTest(t)

	planID := createNextTestPlan(t, entitySvc, "ac003-plan")
	featID := createNextTestFeature(t, entitySvc, planID, "ac003-feat")
	advanceNextFeatureTo(t, entitySvc, featID, "developing")
	taskID, taskSlug := createNextTestTask(t, entitySvc, featID, "ac003-task")
	setNextTaskReady(t, entitySvc, taskID, taskSlug)

	// First claim: dispatches the task and pushes a status_transition side effect.
	firstRaw := callNext(t, entitySvc, dispatchSvc, map[string]any{"id": taskID})
	var firstResult map[string]any
	if err := json.Unmarshal([]byte(firstRaw), &firstResult); err != nil {
		t.Fatalf("parse first claim result: %v", err)
	}
	firstSideEffects, _ := firstResult["side_effects"].([]any)
	if len(firstSideEffects) != 1 {
		t.Fatalf("expected 1 side effect on first claim, got %d", len(firstSideEffects))
	}

	// Record the dispatched_at timestamp to detect whether dispatch fires again.
	afterFirst, err := entitySvc.Get("task", taskID, taskSlug)
	if err != nil {
		t.Fatalf("get task after first claim: %v", err)
	}
	dispatchedAt, _ := afterFirst.State["dispatched_at"].(string)
	if dispatchedAt == "" {
		t.Fatal("dispatched_at not set after first claim")
	}

	// Reclaim: must NOT re-invoke dispatch.
	reclaimRaw := callNext(t, entitySvc, dispatchSvc, map[string]any{"id": taskID})
	var reclaimResult map[string]any
	if err := json.Unmarshal([]byte(reclaimRaw), &reclaimResult); err != nil {
		t.Fatalf("parse reclaim result: %v", err)
	}
	if _, hasErr := reclaimResult["error"]; hasErr {
		t.Fatalf("expected success on reclaim: %s", reclaimRaw)
	}

	// No side effects on reclaim (dispatch hook not re-fired).
	reclaimSideEffects, _ := reclaimResult["side_effects"].([]any)
	if len(reclaimSideEffects) != 0 {
		t.Errorf("expected 0 side effects on reclaim, got %d: %v", len(reclaimSideEffects), reclaimSideEffects)
	}

	// dispatched_at must be unchanged (proves DispatchTask was not called again).
	afterReclaim, err := entitySvc.Get("task", taskID, taskSlug)
	if err != nil {
		t.Fatalf("get task after reclaim: %v", err)
	}
	if afterReclaim.State["dispatched_at"] != dispatchedAt {
		t.Errorf("dispatched_at changed on reclaim: was %q, now %q — dispatch hook was re-fired",
			dispatchedAt, afterReclaim.State["dispatched_at"])
	}
}

// TestNextClaimMode_DoneTask_StillErrors verifies that a task in "done" status
// cannot be claimed and returns an error (AC-004).
func TestNextClaimMode_DoneTask_StillErrors(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupNextTest(t)

	planID := createNextTestPlan(t, entitySvc, "ac004-plan")
	featID := createNextTestFeature(t, entitySvc, planID, "ac004-feat")
	advanceNextFeatureTo(t, entitySvc, featID, "developing")
	taskID, taskSlug := createNextTestTask(t, entitySvc, featID, "ac004-task")

	// Force the task into "done" status by writing directly to storage.
	rec, err := entitySvc.Store().Load("task", taskID, taskSlug)
	if err != nil {
		t.Fatalf("load task record: %v", err)
	}
	rec.Fields["status"] = "done"
	if _, err := entitySvc.Store().Write(rec); err != nil {
		t.Fatalf("write task record with done status: %v", err)
	}

	raw := callNext(t, entitySvc, dispatchSvc, map[string]any{"id": taskID})
	var result map[string]any
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if _, hasErr := result["error"]; !hasErr {
		t.Fatalf("expected error for done task, got success: %s", raw)
	}
}

// TestNextClaimMode_QueuedTask_StillErrors verifies that a task in "queued"
// status (not yet promoted to ready) cannot be claimed and returns an error
// (AC-005).
func TestNextClaimMode_QueuedTask_StillErrors(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupNextTest(t)

	planID := createNextTestPlan(t, entitySvc, "ac005-plan")
	featID := createNextTestFeature(t, entitySvc, planID, "ac005-feat")
	advanceNextFeatureTo(t, entitySvc, featID, "developing")
	taskID, _ := createNextTestTask(t, entitySvc, featID, "ac005-task")
	// Task is in queued status — not yet promoted to ready.

	raw := callNext(t, entitySvc, dispatchSvc, map[string]any{"id": taskID})
	var result map[string]any
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if _, hasErr := result["error"]; !hasErr {
		t.Fatalf("expected error for queued task, got success: %s", raw)
	}
}

// TestNextClaimMode_FirstClaim_NoReclaimedField verifies that a fresh first
// claim of a ready task does NOT include the "reclaimed" key in the response
// (FR-002 — AC-009).
func TestNextClaimMode_FirstClaim_NoReclaimedField(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupNextTest(t)

	planID := createNextTestPlan(t, entitySvc, "ac009-plan")
	featID := createNextTestFeature(t, entitySvc, planID, "ac009-feat")
	advanceNextFeatureTo(t, entitySvc, featID, "developing")
	taskID, taskSlug := createNextTestTask(t, entitySvc, featID, "ac009-task")
	setNextTaskReady(t, entitySvc, taskID, taskSlug)

	// First claim.
	raw := callNext(t, entitySvc, dispatchSvc, map[string]any{"id": taskID})
	var result map[string]any
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("parse first claim result: %v", err)
	}
	if _, hasErr := result["error"]; hasErr {
		t.Fatalf("unexpected error on first claim: %s", raw)
	}

	// "reclaimed" must be absent on a first claim (FR-002).
	if _, ok := result["reclaimed"]; ok {
		t.Errorf("first claim must not include reclaimed key, got: %s", raw)
	}
	// Sanity: task and context must be present.
	if result["task"] == nil {
		t.Error("expected task field in first-claim response")
	}
	if result["context"] == nil {
		t.Error("expected context field in first-claim response")
	}
}
