package service

import (
	"fmt"
	"os"
	"testing"
)

const testPlanIDPromote = "P1-promote-test"

// setupPromoteTest creates an EntityService with a plan and feature.
// Returns the service and feature ID.
func setupPromoteTest(t *testing.T) (*EntityService, string) {
	t.Helper()
	root := t.TempDir()
	svc := newTestEntityService(root, "2026-06-01T12:00:00Z")
	writeTestPlan(t, svc, testPlanIDPromote)
	feat, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "promote-feat",
		Parent:    testPlanIDPromote,
		Summary:   "Feature for PromoteQueuedTasks tests",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature: %v", err)
	}
	return svc, feat.ID
}

// TestPromoteQueuedTasks_NoDeps verifies AC-001: two queued tasks with no
// dependencies are both transitioned to ready.
func TestPromoteQueuedTasks_NoDeps(t *testing.T) {
	t.Parallel()
	svc, featID := setupPromoteTest(t)

	taskAID, taskASlug := createTestTask(t, svc, featID, "task-a", "Task A")
	taskBID, taskBSlug := createTestTask(t, svc, featID, "task-b", "Task B")

	if err := svc.PromoteQueuedTasks(featID); err != nil {
		t.Fatalf("PromoteQueuedTasks() error = %v", err)
	}

	for _, tc := range []struct{ id, slug string }{{taskAID, taskASlug}, {taskBID, taskBSlug}} {
		res, err := svc.Get("task", tc.id, tc.slug)
		if err != nil {
			t.Fatalf("Get(%s): %v", tc.id, err)
		}
		if got := res.State["status"]; got != "ready" {
			t.Errorf("task %s status = %v, want ready", tc.id, got)
		}
	}
}

// TestPromoteQueuedTasks_AllDoneDeps verifies AC-002: a queued task whose
// entire depends_on list is in a terminal (done) state is promoted to ready.
func TestPromoteQueuedTasks_AllDoneDeps(t *testing.T) {
	t.Parallel()
	svc, featID := setupPromoteTest(t)

	depID, depSlug := createTestTask(t, svc, featID, "dep-task", "Dependency")
	blockedID, blockedSlug := createTestTask(t, svc, featID, "blocked-task", "Blocked")

	setTestDependsOn(t, svc, blockedID, blockedSlug, []string{depID})
	advanceTaskTo(t, svc, depID, depSlug, "done")

	if err := svc.PromoteQueuedTasks(featID); err != nil {
		t.Fatalf("PromoteQueuedTasks() error = %v", err)
	}

	res, err := svc.Get("task", blockedID, blockedSlug)
	if err != nil {
		t.Fatalf("Get blocked task: %v", err)
	}
	if got := res.State["status"]; got != "ready" {
		t.Errorf("blocked task status = %v, want ready", got)
	}
}

// TestPromoteQueuedTasks_BlockedByQueued verifies AC-003: a queued task whose
// depends_on contains a task that is still queued is left in queued.
func TestPromoteQueuedTasks_BlockedByQueued(t *testing.T) {
	t.Parallel()
	svc, featID := setupPromoteTest(t)

	depID, depSlug := createTestTask(t, svc, featID, "dep-task", "Dependency")
	blockedID, blockedSlug := createTestTask(t, svc, featID, "blocked-task", "Blocked")

	setTestDependsOn(t, svc, blockedID, blockedSlug, []string{depID})
	// dep remains queued

	if err := svc.PromoteQueuedTasks(featID); err != nil {
		t.Fatalf("PromoteQueuedTasks() error = %v", err)
	}

	// dep has no deps → gets promoted to ready
	depRes, err := svc.Get("task", depID, depSlug)
	if err != nil {
		t.Fatalf("Get dep task: %v", err)
	}
	if got := depRes.State["status"]; got != "ready" {
		t.Errorf("dep task status = %v, want ready", got)
	}

	// blocked task still has a non-terminal dep (dep is now ready, not done)
	blockedRes, err := svc.Get("task", blockedID, blockedSlug)
	if err != nil {
		t.Fatalf("Get blocked task: %v", err)
	}
	if got := blockedRes.State["status"]; got != "queued" {
		t.Errorf("blocked task status = %v, want queued", got)
	}
}

// TestPromoteQueuedTasks_AlreadyReady verifies AC-004: a task already in
// ready status is not modified.
func TestPromoteQueuedTasks_AlreadyReady(t *testing.T) {
	t.Parallel()
	svc, featID := setupPromoteTest(t)

	taskID, taskSlug := createTestTask(t, svc, featID, "task-a", "Task A")
	// Manually advance to ready before calling PromoteQueuedTasks
	advanceTaskTo(t, svc, taskID, taskSlug, "ready")

	if err := svc.PromoteQueuedTasks(featID); err != nil {
		t.Fatalf("PromoteQueuedTasks() error = %v", err)
	}

	res, err := svc.Get("task", taskID, taskSlug)
	if err != nil {
		t.Fatalf("Get task: %v", err)
	}
	if got := res.State["status"]; got != "ready" {
		t.Errorf("task status = %v, want ready (should be unchanged)", got)
	}
}

// TestPromoteQueuedTasks_Idempotent verifies AC-007: calling PromoteQueuedTasks
// a second time after all tasks are already ready is a no-op and returns no error.
func TestPromoteQueuedTasks_Idempotent(t *testing.T) {
	t.Parallel()
	svc, featID := setupPromoteTest(t)

	taskAID, taskASlug := createTestTask(t, svc, featID, "task-a", "Task A")
	taskBID, taskBSlug := createTestTask(t, svc, featID, "task-b", "Task B")

	// First call: promotes both
	if err := svc.PromoteQueuedTasks(featID); err != nil {
		t.Fatalf("PromoteQueuedTasks() first call error = %v", err)
	}

	// Verify both are ready
	for _, tc := range []struct{ id, slug string }{{taskAID, taskASlug}, {taskBID, taskBSlug}} {
		res, err := svc.Get("task", tc.id, tc.slug)
		if err != nil {
			t.Fatalf("Get(%s): %v", tc.id, err)
		}
		if got := res.State["status"]; got != "ready" {
			t.Errorf("task %s after first call: status = %v, want ready", tc.id, got)
		}
	}

	// Second call: no-op
	if err := svc.PromoteQueuedTasks(featID); err != nil {
		t.Fatalf("PromoteQueuedTasks() second call error = %v", err)
	}

	// Status unchanged
	for _, tc := range []struct{ id, slug string }{{taskAID, taskASlug}, {taskBID, taskBSlug}} {
		res, err := svc.Get("task", tc.id, tc.slug)
		if err != nil {
			t.Fatalf("Get(%s): %v", tc.id, err)
		}
		if got := res.State["status"]; got != "ready" {
			t.Errorf("task %s after second call: status = %v, want ready", tc.id, got)
		}
	}
}

// TestPromoteQueuedTasks_OnlyOwnFeature verifies that tasks belonging to a
// different feature are not touched.
func TestPromoteQueuedTasks_OnlyOwnFeature(t *testing.T) {
	t.Parallel()
	svc, featID := setupPromoteTest(t)

	// Create a second feature under the same plan
	otherFeat, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "other",
		Slug:      "other-feat",
		Parent:    testPlanIDPromote,
		Summary:   "Other feature",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature other: %v", err)
	}

	ownTaskID, ownTaskSlug := createTestTask(t, svc, featID, "own-task", "Own task")
	otherTaskID, otherTaskSlug := createTestTask(t, svc, otherFeat.ID, "other-task", "Other task")

	if err := svc.PromoteQueuedTasks(featID); err != nil {
		t.Fatalf("PromoteQueuedTasks() error = %v", err)
	}

	ownRes, err := svc.Get("task", ownTaskID, ownTaskSlug)
	if err != nil {
		t.Fatalf("Get own task: %v", err)
	}
	if got := ownRes.State["status"]; got != "ready" {
		t.Errorf("own task status = %v, want ready", got)
	}

	otherRes, err := svc.Get("task", otherTaskID, otherTaskSlug)
	if err != nil {
		t.Fatalf("Get other task: %v", err)
	}
	if got := otherRes.State["status"]; got != "queued" {
		t.Errorf("other feature task status = %v, want queued (untouched)", got)
	}
}

// TestPromoteQueuedTasks_FailureIsolation verifies AC-006: if UpdateStatus fails
// for one task, the error is logged and the loop continues to promote other tasks.
func TestPromoteQueuedTasks_FailureIsolation(t *testing.T) {
	t.Parallel()
	svc, featID := setupPromoteTest(t)

	taskAID, taskASlug := createTestTask(t, svc, featID, "task-a", "Task A")
	taskBResult, err := svc.CreateTask(CreateTaskInput{
		Name:          "test",
		ParentFeature: featID,
		Slug:          "task-b",
		Summary:       "Task B",
	})
	if err != nil {
		t.Fatalf("CreateTask B: %v", err)
	}
	taskBID := taskBResult.ID
	taskBSlug := taskBResult.Slug

	// Corrupt task B's file by writing a YAML whose id field mismatches the
	// filename. store.Write validates id consistency, so UpdateStatus will fail
	// for this task while leaving task A unaffected.
	corruptContent := fmt.Sprintf(`id: TASK-CORRUPT-ID
slug: %s
parent_feature: %s
name: Task B
status: queued
summary: Task B
type: task
`, taskBSlug, featID)
	if err := os.WriteFile(taskBResult.Path, []byte(corruptContent), 0o644); err != nil {
		t.Fatalf("write corrupt file: %v", err)
	}

	if err := svc.PromoteQueuedTasks(featID); err != nil {
		t.Fatalf("PromoteQueuedTasks() must not return error even when a task fails: %v", err)
	}

	// Task A should be promoted
	resA, err := svc.Get("task", taskAID, taskASlug)
	if err != nil {
		t.Fatalf("Get task A: %v", err)
	}
	if got := resA.State["status"]; got != "ready" {
		t.Errorf("task A status = %v, want ready", got)
	}

	// Task B should still be queued (UpdateStatus failed, loop continued)
	resB, err := svc.Get("task", taskBID, taskBSlug)
	if err != nil {
		t.Fatalf("Get task B: %v", err)
	}
	if got := resB.State["status"]; got != "queued" {
		t.Errorf("task B status = %v, want queued (update should have failed)", got)
	}
}
