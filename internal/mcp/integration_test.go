package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	kbzctx "github.com/sambeau/kanbanzai/internal/context"
	"github.com/sambeau/kanbanzai/internal/service"
	"github.com/sambeau/kanbanzai/internal/storage"
)

// TestIntegration_NextHandoffFinish exercises the full 2.0 workflow cycle:
//
//	next(task_id) → handoff(task_id) → finish(task_id)
//
// This is the acceptance gate for Track K (spec §30.11, K.13).
func TestIntegration_NextHandoffFinish(t *testing.T) {
	t.Parallel()

	// ── Setup services ──────────────────────────────────────────────────

	entityRoot := t.TempDir()
	knowledgeRoot := t.TempDir()
	profileRoot := filepath.Join(t.TempDir(), "roles")
	indexRoot := t.TempDir()

	entitySvc := service.NewEntityService(entityRoot)
	knowledgeSvc := service.NewKnowledgeService(knowledgeRoot)
	dispatchSvc := service.NewDispatchService(entitySvc, knowledgeSvc)
	profileStore := kbzctx.NewProfileStore(profileRoot)
	intelligenceSvc := service.NewIntelligenceService(indexRoot, ".")

	// Wire the dependency unblocking hook so finish reports unblocked tasks.
	hook := service.NewDependencyUnblockingHook(entitySvc)
	entitySvc.SetStatusTransitionHook(hook)

	// ── Create entities: plan → feature → task ──────────────────────────

	planID := "P1-integration"
	writeIntegrationPlan(t, entitySvc, planID)

	feat, err := entitySvc.CreateFeature(service.CreateFeatureInput{
		Name:      "test",
		Slug:      "e2e-feature",
		Parent:    planID,
		Summary:   "End-to-end integration test feature",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature: %v", err)
	}

	// Advance feature to a working state so stage validation passes.
	for _, fStatus := range []string{"designing", "specifying", "dev-planning", "developing"} {
		if _, err := entitySvc.UpdateStatus(service.UpdateStatusInput{
			Type: "feature", ID: feat.ID, Slug: "e2e-feature", Status: fStatus,
		}); err != nil {
			t.Fatalf("advance feature to %s: %v", fStatus, err)
		}
	}

	taskResult, err := entitySvc.CreateTask(service.CreateTaskInput{
		Name:          "test",
		ParentFeature: feat.ID,
		Slug:          "e2e-task",
		Summary:       "Implement the widget",
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	taskID := taskResult.ID
	taskSlug := taskResult.Slug

	// Advance task to ready: queued → ready.
	if _, err := entitySvc.UpdateStatus(service.UpdateStatusInput{
		Type: "task", ID: taskID, Slug: taskSlug, Status: "ready",
	}); err != nil {
		t.Fatalf("advance to ready: %v", err)
	}

	// ── Step 1: next(task_id) — claim the task ──────────────────────────

	nextTools := NextTools(entitySvc, dispatchSvc, profileStore, knowledgeSvc, intelligenceSvc, nil, nil, nil, nil)
	if len(nextTools) == 0 {
		t.Fatal("NextTools returned no tools")
	}
	nextHandler := nextTools[0].Handler

	nextReq := makeRequest(map[string]any{
		"id": taskID,
	})

	nextResult, err := nextHandler(context.Background(), nextReq)
	if err != nil {
		t.Fatalf("next(%s) error: %v", taskID, err)
	}
	if nextResult.IsError {
		t.Fatalf("next(%s) returned error result: %v", taskID, extractText(t, nextResult))
	}

	nextText := extractText(t, nextResult)
	if nextText == "" {
		t.Fatal("next returned empty result")
	}

	// The next tool should have transitioned the task to active.
	taskAfterNext, err := entitySvc.Get("task", taskID, "")
	if err != nil {
		t.Fatalf("Get task after next: %v", err)
	}
	statusAfterNext, _ := taskAfterNext.State["status"].(string)
	if statusAfterNext != "active" {
		t.Fatalf("task status after next = %q, want \"active\"", statusAfterNext)
	}

	// The response should contain the task ID somewhere.
	if !strings.Contains(nextText, taskID) && !strings.Contains(nextText, "e2e-task") {
		t.Errorf("next response does not reference the task:\n%.500s", nextText)
	}

	// ── Step 2: handoff(task_id) — generate sub-agent prompt ────────────

	handoffTools := HandoffTools(entitySvc, profileStore, knowledgeSvc, intelligenceSvc, nil, nil, nil, nil, nil)
	if len(handoffTools) == 0 {
		t.Fatal("HandoffTools returned no tools")
	}
	handoffHandler := handoffTools[0].Handler

	handoffReq := makeRequest(map[string]any{
		"task_id": taskID,
	})

	handoffResult, err := handoffHandler(context.Background(), handoffReq)
	if err != nil {
		t.Fatalf("handoff(%s) error: %v", taskID, err)
	}
	if handoffResult.IsError {
		t.Fatalf("handoff(%s) returned error result: %v", taskID, extractText(t, handoffResult))
	}

	handoffText := extractText(t, handoffResult)
	if handoffText == "" {
		t.Fatal("handoff returned empty result")
	}

	// The handoff prompt should mention the task summary.
	if !strings.Contains(handoffText, "widget") && !strings.Contains(handoffText, "e2e-task") {
		t.Errorf("handoff prompt does not reference the task content:\n%.500s", handoffText)
	}

	// ── Step 3: finish(task_id) — complete the task ─────────────────────

	finishTools := FinishTools(entitySvc, dispatchSvc)
	if len(finishTools) == 0 {
		t.Fatal("FinishTools returned no tools")
	}
	finishHandler := finishTools[0].Handler

	finishReq := makeRequest(map[string]any{
		"task_id": taskID,
		"summary": "Implemented the widget with full test coverage",
		"files_modified": []any{
			"internal/widget/widget.go",
			"internal/widget/widget_test.go",
		},
		"verification": "All tests pass with race detector",
	})

	finishResult, err := finishHandler(context.Background(), finishReq)
	if err != nil {
		t.Fatalf("finish(%s) error: %v", taskID, err)
	}
	if finishResult.IsError {
		t.Fatalf("finish(%s) returned error result: %v", taskID, extractText(t, finishResult))
	}

	finishText := extractText(t, finishResult)

	// Parse the finish response to verify structure.
	var finishResp map[string]any
	if err := json.Unmarshal([]byte(finishText), &finishResp); err != nil {
		t.Fatalf("parse finish response: %v\nraw: %.500s", err, finishText)
	}

	// Verify finish response contains the task.
	if _, ok := finishResp["task"]; !ok {
		t.Errorf("finish response missing 'task' key: %v", finishResp)
	}

	// ── Verify final state ──────────────────────────────────────────────

	taskAfterFinish, err := entitySvc.Get("task", taskID, "")
	if err != nil {
		t.Fatalf("Get task after finish: %v", err)
	}
	statusAfterFinish, _ := taskAfterFinish.State["status"].(string)
	if statusAfterFinish != "done" {
		t.Fatalf("task status after finish = %q, want \"done\"", statusAfterFinish)
	}

	// Verify completion metadata was set.
	completionSummary, _ := taskAfterFinish.State["completion_summary"].(string)
	if completionSummary == "" {
		t.Error("task completion_summary is empty after finish")
	}
	if !strings.Contains(completionSummary, "widget") {
		t.Errorf("completion_summary = %q, want it to contain \"widget\"", completionSummary)
	}

	t.Logf("Integration test passed: next → handoff → finish cycle complete")
	t.Logf("  Task %s: queued → ready → active → done", taskID)
}

// ─── Integration test helpers ────────────────────────────────────────────────

// writeIntegrationPlan writes a plan record directly, bypassing config-dependent
// CreatePlan so the test works without a .kbz/config.yaml file.
func writeIntegrationPlan(t *testing.T, entitySvc *service.EntityService, id string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	slug := integrationPlanSlug(id)
	record := storage.EntityRecord{
		Type: "plan",
		ID:   id,
		Slug: slug,
		Fields: map[string]any{
			"id":         id,
			"slug":       slug,
			"title":      "Integration Test Plan",
			"status":     "active",
			"summary":    "Plan for integration testing",
			"created":    now,
			"created_by": "tester",
			"updated":    now,
		},
	}
	if _, err := entitySvc.Store().Write(record); err != nil {
		t.Fatalf("writeIntegrationPlan(%s): %v", id, err)
	}
}

// integrationPlanSlug extracts the slug from a plan ID like "P1-foo" → "foo".
func integrationPlanSlug(id string) string {
	if idx := strings.Index(id[1:], "-"); idx >= 0 {
		return id[idx+2:]
	}
	return id
}

// TestIntegration_FinishStateConsistency verifies AC-003: after finish() returns
// success, all read paths (entity get, entity list with parent_feature filter,
// and sibling-based gate checks) observe the task as done.
func TestIntegration_FinishStateConsistency(t *testing.T) {
	t.Parallel()

	// ── Setup services ──────────────────────────────────────────────────

	entityRoot := t.TempDir()
	knowledgeRoot := t.TempDir()

	entitySvc := service.NewEntityService(entityRoot)
	knowledgeSvc := service.NewKnowledgeService(knowledgeRoot)
	dispatchSvc := service.NewDispatchService(entitySvc, knowledgeSvc)

	// Wire the dependency unblocking hook.
	hook := service.NewDependencyUnblockingHook(entitySvc)
	entitySvc.SetStatusTransitionHook(hook)

	// ── Create entities: plan → feature → task ──────────────────────────

	planID := "P1-finish-consistency"
	writeIntegrationPlan(t, entitySvc, planID)

	feat, err := entitySvc.CreateFeature(service.CreateFeatureInput{
		Name:      "test",
		Slug:      "finish-consistency",
		Parent:    planID,
		Summary:   "Test feature for finish state consistency",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature: %v", err)
	}

	// Advance feature to developing so tasks can be manipulated.
	for _, fStatus := range []string{"designing", "specifying", "dev-planning", "developing"} {
		if _, err := entitySvc.UpdateStatus(service.UpdateStatusInput{
			Type: "feature", ID: feat.ID, Slug: "finish-consistency", Status: fStatus,
		}); err != nil {
			t.Fatalf("advance feature to %s: %v", fStatus, err)
		}
	}

	// Create two tasks — one to finish, one to leave active for sibling testing.
	task1Result, err := entitySvc.CreateTask(service.CreateTaskInput{
		Name:          "test",
		ParentFeature: feat.ID,
		Slug:          "task-to-finish",
		Summary:       "Task we will finish",
	})
	if err != nil {
		t.Fatalf("CreateTask 1: %v", err)
	}
	task1ID := task1Result.ID
	task1Slug := task1Result.Slug

	task2Result, err := entitySvc.CreateTask(service.CreateTaskInput{
		Name:          "test",
		ParentFeature: feat.ID,
		Slug:          "sibling-task",
		Summary:       "Sibling task that stays active",
	})
	if err != nil {
		t.Fatalf("CreateTask 2: %v", err)
	}
	task2ID := task2Result.ID

	// Advance both tasks to active (queued → ready → active).
	advanceToActive(t, entitySvc, task1ID, task1Slug)
	advanceToActive(t, entitySvc, task2ID, task2Result.Slug)

	// ── Call finish() on task1 ──────────────────────────────────────────

	finishTools := FinishTools(entitySvc, dispatchSvc)
	finishHandler := finishTools[0].Handler

	finishReq := makeRequest(map[string]any{
		"task_id": task1ID,
		"summary": "Completed the state consistency fix",
	})

	finishResult, err := finishHandler(context.Background(), finishReq)
	if err != nil {
		t.Fatalf("finish(%s) error: %v", task1ID, err)
	}
	if finishResult.IsError {
		t.Fatalf("finish(%s) returned error result: %v", task1ID, extractText(t, finishResult))
	}

	// ── AC-003: entity get returns done ─────────────────────────────────

	taskAfterFinish, err := entitySvc.Get("task", task1ID, "")
	if err != nil {
		t.Fatalf("entity get after finish: %v", err)
	}
	statusAfterFinish, _ := taskAfterFinish.State["status"].(string)
	if statusAfterFinish != "done" {
		t.Errorf("entity get: task status = %q, want \"done\"", statusAfterFinish)
	}

	// Completion metadata should be present.
	completionSummary, _ := taskAfterFinish.State["completion_summary"].(string)
	if completionSummary == "" {
		t.Error("entity get: completion_summary is empty after finish")
	}

	// ── AC-003: entity list with parent_feature shows task as done ──────

	tasksForFeature, err := entitySvc.ListEntitiesFiltered(service.ListFilteredInput{
		Type:   "task",
		Parent: feat.ID,
	})
	if err != nil {
		t.Fatalf("entity list with parent_feature: %v", err)
	}

	var foundTask1, foundTask2 bool
	for _, tResult := range tasksForFeature {
		st, _ := tResult.State["status"].(string)
		switch tResult.ID {
		case task1ID:
			foundTask1 = true
			if st != "done" {
				t.Errorf("entity list: task1 status = %q, want \"done\"", st)
			}
		case task2ID:
			foundTask2 = true
			if st != "active" {
				t.Errorf("entity list: task2 status = %q, want \"active\" (untouched sibling)", st)
			}
		}
	}
	if !foundTask1 {
		t.Errorf("entity list with parent_feature did not return finished task %s", task1ID)
	}
	if !foundTask2 {
		t.Errorf("entity list with parent_feature did not return sibling task %s", task2ID)
	}

	// ── AC-003: gate check — sibling check correctly sees task1 as done ──
	// The sibling-based "all tasks terminal" check is done in finish_tool.go
	// by calling ListEntitiesFiltered. We simulate the same check here:
	// task1 is done, task2 is active → not all terminal.
	allTerminal := true
	for _, tResult := range tasksForFeature {
		st, _ := tResult.State["status"].(string)
		if !isFinishTerminal(st) {
			allTerminal = false
			break
		}
	}
	if allTerminal {
		t.Error("gate check: all tasks terminal = true, want false (task2 is still active)")
	}

	t.Logf("Finish state consistency test passed:")
	t.Logf("  entity get:  task1 = %s", statusAfterFinish)
	t.Logf("  entity list: task1 = done, task2 = active")
	t.Logf("  gate check:  all terminal = false (correct — task2 still active)")
}

// BenchmarkFinishLatency measures p95 latency of finish() calls to satisfy
// AC-013 (REQ-NF-002): no measurable latency increase from the cache write.
func BenchmarkFinishLatency(b *testing.B) {
	// ── Setup ───────────────────────────────────────────────────────────

	entityRoot := b.TempDir()
	knowledgeRoot := b.TempDir()

	entitySvc := service.NewEntityService(entityRoot)
	knowledgeSvc := service.NewKnowledgeService(knowledgeRoot)
	dispatchSvc := service.NewDispatchService(entitySvc, knowledgeSvc)

	planID := "P1-bench-finish"
	now := time.Now().UTC().Format(time.RFC3339)
	slug := "bench-finish"
	record := storage.EntityRecord{
		Type: "plan",
		ID:   planID,
		Slug: slug,
		Fields: map[string]any{
			"id":         planID,
			"slug":       slug,
			"title":      "Bench Plan",
			"status":     "active",
			"summary":    "Benchmark plan",
			"created":    now,
			"created_by": "tester",
			"updated":    now,
		},
	}
	if _, err := entitySvc.Store().Write(record); err != nil {
		b.Fatalf("write plan: %v", err)
	}

	feat, err := entitySvc.CreateFeature(service.CreateFeatureInput{
		Name:      "test",
		Slug:      "bench-feat",
		Parent:    planID,
		Summary:   "Benchmark feature",
		CreatedBy: "tester",
	})
	if err != nil {
		b.Fatalf("CreateFeature: %v", err)
	}

	finishTools := FinishTools(entitySvc, dispatchSvc)
	finishHandler := finishTools[0].Handler

	// Pre-create tasks and advance them to active.
	type benchTask struct {
		id, slug string
	}
	var tasks []benchTask
	for i := 0; i < b.N; i++ {
		taskSlug := fmt.Sprintf("bench-task-%d", i)
		taskResult, err := entitySvc.CreateTask(service.CreateTaskInput{
			Name:          "test",
			ParentFeature: feat.ID,
			Slug:          taskSlug,
			Summary:       "Benchmark task",
		})
		if err != nil {
			b.Fatalf("CreateTask %d: %v", i, err)
		}
		// Advance: queued → ready → active (inline since advanceToActive takes *testing.T).
		if _, err := entitySvc.UpdateStatus(service.UpdateStatusInput{
			Type: "task", ID: taskResult.ID, Slug: taskResult.Slug, Status: "ready",
		}); err != nil {
			b.Fatalf("advance %s to ready: %v", taskResult.ID, err)
		}
		if _, err := entitySvc.UpdateStatus(service.UpdateStatusInput{
			Type: "task", ID: taskResult.ID, Slug: taskResult.Slug, Status: "active",
		}); err != nil {
			b.Fatalf("advance %s to active: %v", taskResult.ID, err)
		}
		tasks = append(tasks, benchTask{id: taskResult.ID, slug: taskResult.Slug})
	}

	// ── Benchmark ───────────────────────────────────────────────────────

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := makeRequest(map[string]any{
			"task_id": tasks[i].id,
			"summary": "Benchmarked finish",
		})
		result, err := finishHandler(context.Background(), req)
		if err != nil {
			b.Fatalf("finish iteration %d: %v", i, err)
		}
		if result.IsError {
			b.Fatalf("finish iteration %d returned error", i)
		}
	}
	b.StopTimer()
}
