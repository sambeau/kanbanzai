package service

import (
	"testing"

	"kanbanzai/internal/model"
)

const testPlanIDQueue = "P1-queue-test"

// setupQueueTest creates an EntityService with a plan, a feature, and two tasks.
// It returns the service, the feature ID, and the two task IDs (taskA, taskB).
// Both tasks have files_planned set to share "internal/shared/handler.go".
func setupQueueTest(t *testing.T) (*EntityService, string, string, string) {
	t.Helper()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	writeTestPlan(t, svc, testPlanIDQueue)

	feat, err := svc.CreateFeature(CreateFeatureInput{
		Slug:      "queue-feat",
		Parent:    testPlanIDQueue,
		Summary:   "Feature for queue conflict tests",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature: %v", err)
	}

	taskA, err := svc.CreateTask(CreateTaskInput{
		ParentFeature: feat.ID,
		Slug:          "task-alpha",
		Summary:       "First task for conflict testing",
	})
	if err != nil {
		t.Fatalf("CreateTask A: %v", err)
	}

	taskB, err := svc.CreateTask(CreateTaskInput{
		ParentFeature: feat.ID,
		Slug:          "task-beta",
		Summary:       "Second task for conflict testing",
	})
	if err != nil {
		t.Fatalf("CreateTask B: %v", err)
	}

	// Write files_planned on both tasks by reloading, patching, and re-writing via the store.
	setFilesPlanned(t, svc, taskA.ID, taskA.Slug, []string{
		"internal/shared/handler.go",
		"internal/alpha/alpha.go",
	})
	setFilesPlanned(t, svc, taskB.ID, taskB.Slug, []string{
		"internal/shared/handler.go",
		"internal/beta/beta.go",
	})

	return svc, feat.ID, taskA.ID, taskB.ID
}

// setFilesPlanned loads a task record, adds files_planned, and writes it back.
func setFilesPlanned(t *testing.T, svc *EntityService, taskID, slug string, files []string) {
	t.Helper()

	record, err := svc.Store().Load("task", taskID, slug)
	if err != nil {
		t.Fatalf("Store().Load(%s): %v", taskID, err)
	}

	// Store files as []any so the canonical YAML serialiser handles them correctly.
	anyFiles := make([]any, len(files))
	for i, f := range files {
		anyFiles[i] = f
	}
	record.Fields["files_planned"] = anyFiles

	if _, err := svc.Store().Write(record); err != nil {
		t.Fatalf("Store().Write(%s) with files_planned: %v", taskID, err)
	}
}

// transitionTask is a small helper that transitions a task to the given status.
func transitionTask(t *testing.T, svc *EntityService, taskID, slug, status string) {
	t.Helper()
	_, err := svc.UpdateStatus(UpdateStatusInput{
		Type:   "task",
		ID:     taskID,
		Slug:   slug,
		Status: status,
	})
	if err != nil {
		t.Fatalf("UpdateStatus(%s → %s): %v", taskID, status, err)
	}
}

// TestWorkQueue_ConflictCheckAnnotated verifies that when ConflictCheck is true,
// ready tasks are annotated with ConflictRisk and ConflictWith based on
// files_planned overlap with active tasks (spec §16.4).
func TestWorkQueue_ConflictCheckAnnotated(t *testing.T) {
	t.Parallel()

	svc, _, taskAID, taskBID := setupQueueTest(t)

	// Transition task A: queued → ready → active
	transitionTask(t, svc, taskAID, "task-alpha", string(model.TaskStatusReady))
	transitionTask(t, svc, taskAID, "task-alpha", string(model.TaskStatusActive))

	// Transition task B: queued → ready
	transitionTask(t, svc, taskBID, "task-beta", string(model.TaskStatusReady))

	result, err := svc.WorkQueue(WorkQueueInput{ConflictCheck: true})
	if err != nil {
		t.Fatalf("WorkQueue(ConflictCheck=true): %v", err)
	}

	// The queue should contain task B (ready). Task A is active, not in queue.
	if len(result.Queue) == 0 {
		t.Fatal("expected at least one item in the queue, got 0")
	}

	var found *WorkQueueItem
	for i := range result.Queue {
		if result.Queue[i].TaskID == taskBID {
			found = &result.Queue[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("task B (%s) not found in queue; queue has %d items", taskBID, len(result.Queue))
	}

	// ConflictRisk should be set (non-empty) because tasks share files_planned.
	if found.ConflictRisk == "" {
		t.Error("ConflictRisk is empty; expected non-empty risk annotation when files overlap with active task")
	}
	if found.ConflictRisk == "none" {
		t.Error("ConflictRisk is \"none\"; expected higher risk due to shared files_planned with active task")
	}

	// ConflictWith should contain task A's ID.
	if len(found.ConflictWith) == 0 {
		t.Fatalf("ConflictWith is empty; expected it to contain %s", taskAID)
	}
	hasTaskA := false
	for _, cw := range found.ConflictWith {
		if cw == taskAID {
			hasTaskA = true
			break
		}
	}
	if !hasTaskA {
		t.Errorf("ConflictWith = %v; expected it to contain %s", found.ConflictWith, taskAID)
	}
}

// TestWorkQueue_NoConflictCheckUnchanged verifies that when ConflictCheck is false,
// ready tasks have no conflict annotation — ConflictRisk is "" and ConflictWith is
// nil/empty. This preserves Phase 4a behaviour (spec §16.4).
func TestWorkQueue_NoConflictCheckUnchanged(t *testing.T) {
	t.Parallel()

	svc, _, taskAID, taskBID := setupQueueTest(t)

	// Transition task A: queued → ready → active
	transitionTask(t, svc, taskAID, "task-alpha", string(model.TaskStatusReady))
	transitionTask(t, svc, taskAID, "task-alpha", string(model.TaskStatusActive))

	// Transition task B: queued → ready
	transitionTask(t, svc, taskBID, "task-beta", string(model.TaskStatusReady))

	result, err := svc.WorkQueue(WorkQueueInput{ConflictCheck: false})
	if err != nil {
		t.Fatalf("WorkQueue(ConflictCheck=false): %v", err)
	}

	if len(result.Queue) == 0 {
		t.Fatal("expected at least one item in the queue, got 0")
	}

	var found *WorkQueueItem
	for i := range result.Queue {
		if result.Queue[i].TaskID == taskBID {
			found = &result.Queue[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("task B (%s) not found in queue", taskBID)
	}

	if found.ConflictRisk != "" {
		t.Errorf("ConflictRisk = %q; expected empty string when ConflictCheck is false", found.ConflictRisk)
	}
	if len(found.ConflictWith) != 0 {
		t.Errorf("ConflictWith = %v; expected nil/empty when ConflictCheck is false", found.ConflictWith)
	}
}

// TestWorkQueue_ConflictCheckNoActiveTasksReturnsNone verifies that when
// ConflictCheck is true but there are no active tasks, ready tasks get
// no conflict annotation (the check short-circuits).
func TestWorkQueue_ConflictCheckNoActiveTasksReturnsNone(t *testing.T) {
	t.Parallel()

	svc, _, _, taskBID := setupQueueTest(t)

	// Only transition task B to ready. Task A stays queued (not active).
	transitionTask(t, svc, taskBID, "task-beta", string(model.TaskStatusReady))

	result, err := svc.WorkQueue(WorkQueueInput{ConflictCheck: true})
	if err != nil {
		t.Fatalf("WorkQueue: %v", err)
	}

	// Task A should have been promoted to ready by the queue (no dependencies).
	// But neither task is active, so conflict check should produce no annotations.
	for _, item := range result.Queue {
		if item.ConflictRisk != "" {
			t.Errorf("task %s: ConflictRisk = %q; expected empty when no active tasks", item.TaskID, item.ConflictRisk)
		}
		if len(item.ConflictWith) != 0 {
			t.Errorf("task %s: ConflictWith = %v; expected empty when no active tasks", item.TaskID, item.ConflictWith)
		}
	}
}

// TestWorkQueue_ConflictCheckNoFileOverlap verifies that when ConflictCheck is
// true and active/ready tasks exist but share no files_planned, the conflict
// annotation reflects no file-overlap risk and ConflictWith is empty.
// Tasks are placed under separate features with fully disjoint keywords to
// avoid the boundary-crossing heuristic contributing any risk.
func TestWorkQueue_ConflictCheckNoFileOverlap(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	writeTestPlan(t, svc, "P1-no-overlap")

	// Two features with completely different slugs/summaries so the boundary
	// crossing heuristic finds fewer than 3 shared keywords.
	featX, err := svc.CreateFeature(CreateFeatureInput{
		Slug:      "payments-gateway",
		Parent:    "P1-no-overlap",
		Summary:   "Stripe integration for checkout",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature X: %v", err)
	}

	featY, err := svc.CreateFeature(CreateFeatureInput{
		Slug:      "telemetry-pipeline",
		Parent:    "P1-no-overlap",
		Summary:   "OpenTelemetry collector setup",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature Y: %v", err)
	}

	taskA, err := svc.CreateTask(CreateTaskInput{
		ParentFeature: featX.ID,
		Slug:          "charge-endpoint",
		Summary:       "Implement Stripe charge endpoint",
	})
	if err != nil {
		t.Fatalf("CreateTask A: %v", err)
	}

	taskB, err := svc.CreateTask(CreateTaskInput{
		ParentFeature: featY.ID,
		Slug:          "span-exporter",
		Summary:       "Configure OTLP span exporter",
	})
	if err != nil {
		t.Fatalf("CreateTask B: %v", err)
	}

	// Completely disjoint files.
	setFilesPlanned(t, svc, taskA.ID, taskA.Slug, []string{"cmd/server/main.go"})
	setFilesPlanned(t, svc, taskB.ID, taskB.Slug, []string{"pkg/telemetry/exporter.go"})

	transitionTask(t, svc, taskA.ID, taskA.Slug, string(model.TaskStatusReady))
	transitionTask(t, svc, taskA.ID, taskA.Slug, string(model.TaskStatusActive))
	transitionTask(t, svc, taskB.ID, taskB.Slug, string(model.TaskStatusReady))

	result, err := svc.WorkQueue(WorkQueueInput{ConflictCheck: true})
	if err != nil {
		t.Fatalf("WorkQueue: %v", err)
	}

	var found *WorkQueueItem
	for i := range result.Queue {
		if result.Queue[i].TaskID == taskB.ID {
			found = &result.Queue[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("task B (%s) not found in queue", taskB.ID)
	}

	// No file overlap and disjoint keywords — ConflictWith should be empty.
	if len(found.ConflictWith) != 0 {
		t.Errorf("ConflictWith = %v; expected empty for disjoint files and unrelated tasks", found.ConflictWith)
	}
}

// TestWorkQueue_PromotionStillWorksWithConflictCheck verifies that the
// write-through promotion behaviour (queued → ready) still functions
// when conflict_check is enabled — it doesn't interfere with normal queue logic.
func TestWorkQueue_PromotionStillWorksWithConflictCheck(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	writeTestPlan(t, svc, "P1-promote")

	feat, err := svc.CreateFeature(CreateFeatureInput{
		Slug:      "promote-feat",
		Parent:    "P1-promote",
		Summary:   "Feature for promotion test",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature: %v", err)
	}

	// Create a single task — it starts as queued with no dependencies.
	task, err := svc.CreateTask(CreateTaskInput{
		ParentFeature: feat.ID,
		Slug:          "promotable",
		Summary:       "Task that should be auto-promoted to ready",
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	// Verify it starts queued.
	got, err := svc.Get("task", task.ID, task.Slug)
	if err != nil {
		t.Fatalf("Get task: %v", err)
	}
	if status := stringFromState(got.State, "status"); status != "queued" {
		t.Fatalf("task initial status = %q, want queued", status)
	}

	result, err := svc.WorkQueue(WorkQueueInput{ConflictCheck: true})
	if err != nil {
		t.Fatalf("WorkQueue: %v", err)
	}

	if result.PromotedCount != 1 {
		t.Errorf("PromotedCount = %d, want 1", result.PromotedCount)
	}

	// The promoted task should now appear in the queue as ready.
	if len(result.Queue) == 0 {
		t.Fatal("expected promoted task in queue, got empty queue")
	}

	var found bool
	for _, item := range result.Queue {
		if item.TaskID == task.ID {
			found = true
			if item.Status != string(model.TaskStatusReady) {
				t.Errorf("promoted task status = %q, want ready", item.Status)
			}
			break
		}
	}
	if !found {
		t.Errorf("promoted task %s not found in queue", task.ID)
	}
}

// writeDispatchFieldsForActiveTask is a small helper to set the dispatch
// fields required on an active task (claimed_at, dispatched_to, etc.)
// so that the transition from ready → active succeeds cleanly.
// In practice, dispatch_task does this, but for queue tests we use the
// raw UpdateStatus path which the state machine allows without dispatch fields.
// This helper exists in case future validation tightens.

// TestWorkQueue_ConflictCheckMultipleActiveTasks verifies that conflict
// annotations work correctly when there are multiple active tasks.
func TestWorkQueue_ConflictCheckMultipleActiveTasks(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	writeTestPlan(t, svc, "P1-multi-active")

	feat, err := svc.CreateFeature(CreateFeatureInput{
		Slug:      "multi-active-feat",
		Parent:    "P1-multi-active",
		Summary:   "Feature for multi-active conflict test",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature: %v", err)
	}

	taskA, err := svc.CreateTask(CreateTaskInput{
		ParentFeature: feat.ID,
		Slug:          "active-one",
		Summary:       "First active task",
	})
	if err != nil {
		t.Fatalf("CreateTask A: %v", err)
	}

	taskB, err := svc.CreateTask(CreateTaskInput{
		ParentFeature: feat.ID,
		Slug:          "active-two",
		Summary:       "Second active task",
	})
	if err != nil {
		t.Fatalf("CreateTask B: %v", err)
	}

	taskC, err := svc.CreateTask(CreateTaskInput{
		ParentFeature: feat.ID,
		Slug:          "ready-one",
		Summary:       "Ready task to check conflicts",
	})
	if err != nil {
		t.Fatalf("CreateTask C: %v", err)
	}

	// Task A shares files with C, task B shares different files with C.
	setFilesPlanned(t, svc, taskA.ID, taskA.Slug, []string{"internal/shared/handler.go"})
	setFilesPlanned(t, svc, taskB.ID, taskB.Slug, []string{"internal/shared/router.go"})
	setFilesPlanned(t, svc, taskC.ID, taskC.Slug, []string{
		"internal/shared/handler.go",
		"internal/shared/router.go",
	})

	// Make A and B active.
	transitionTask(t, svc, taskA.ID, taskA.Slug, string(model.TaskStatusReady))
	transitionTask(t, svc, taskA.ID, taskA.Slug, string(model.TaskStatusActive))
	transitionTask(t, svc, taskB.ID, taskB.Slug, string(model.TaskStatusReady))
	transitionTask(t, svc, taskB.ID, taskB.Slug, string(model.TaskStatusActive))

	// Make C ready.
	transitionTask(t, svc, taskC.ID, taskC.Slug, string(model.TaskStatusReady))

	result, err := svc.WorkQueue(WorkQueueInput{ConflictCheck: true})
	if err != nil {
		t.Fatalf("WorkQueue: %v", err)
	}

	var found *WorkQueueItem
	for i := range result.Queue {
		if result.Queue[i].TaskID == taskC.ID {
			found = &result.Queue[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("task C (%s) not found in queue", taskC.ID)
	}

	if found.ConflictRisk == "" || found.ConflictRisk == "none" {
		t.Errorf("ConflictRisk = %q; expected non-none risk with file overlap against two active tasks", found.ConflictRisk)
	}

	// ConflictWith should mention both active tasks.
	if len(found.ConflictWith) < 2 {
		t.Errorf("ConflictWith = %v; expected both %s and %s", found.ConflictWith, taskA.ID, taskB.ID)
	}
}
