package mcp

import (
	"context"
	"encoding/json"
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
		Slug:      "e2e-feature",
		Parent:    planID,
		Summary:   "End-to-end integration test feature",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature: %v", err)
	}

	taskResult, err := entitySvc.CreateTask(service.CreateTaskInput{
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

	nextTools := NextTools(entitySvc, dispatchSvc, profileStore, knowledgeSvc, intelligenceSvc, nil)
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

	handoffTools := HandoffTools(entitySvc, profileStore, knowledgeSvc, intelligenceSvc, nil, nil)
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
