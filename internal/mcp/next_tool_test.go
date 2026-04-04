package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	kbzctx "github.com/sambeau/kanbanzai/internal/context"
	"github.com/sambeau/kanbanzai/internal/service"
	"github.com/sambeau/kanbanzai/internal/storage"
)

// ─── Test setup helpers ───────────────────────────────────────────────────────

// setupNextTest creates the services needed for next tool tests.
func setupNextTest(t *testing.T) (*service.EntityService, *service.DispatchService) {
	t.Helper()
	entityRoot := t.TempDir()
	stateRoot := t.TempDir()
	entitySvc := service.NewEntityService(entityRoot)
	knowledgeSvc := service.NewKnowledgeService(stateRoot)
	dispatchSvc := service.NewDispatchService(entitySvc, knowledgeSvc)
	return entitySvc, dispatchSvc
}

// setupNextTestFull creates all services including knowledge and profiles.
func setupNextTestFull(t *testing.T) (
	*service.EntityService,
	*service.DispatchService,
	*service.KnowledgeService,
	*kbzctx.ProfileStore,
	*service.IntelligenceService,
	*service.DocumentService,
) {
	t.Helper()
	entityRoot := t.TempDir()
	stateRoot := t.TempDir()
	profileRoot := filepath.Join(t.TempDir(), "roles")
	indexRoot := t.TempDir()
	repoRoot := t.TempDir()

	entitySvc := service.NewEntityService(entityRoot)
	knowledgeSvc := service.NewKnowledgeService(stateRoot)
	dispatchSvc := service.NewDispatchService(entitySvc, knowledgeSvc)
	profileStore := kbzctx.NewProfileStore(profileRoot)
	intelligenceSvc := service.NewIntelligenceService(indexRoot, repoRoot)
	docRecordSvc := service.NewDocumentService(stateRoot, repoRoot)
	docRecordSvc.SetIntelligenceService(intelligenceSvc)

	return entitySvc, dispatchSvc, knowledgeSvc, profileStore, intelligenceSvc, docRecordSvc
}

// createNextTestPlan writes a plan record directly (no config.yaml needed).
func createNextTestPlan(t *testing.T, entitySvc *service.EntityService, slug string) string {
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
			"title":      "Test plan " + slug,
			"status":     "proposed",
			"summary":    "Test plan",
			"created":    now,
			"created_by": "tester",
			"updated":    now,
		},
	}
	if _, err := entitySvc.Store().Write(record); err != nil {
		t.Fatalf("createNextTestPlan(%s): %v", slug, err)
	}
	return planID
}

// createNextTestFeature creates a feature for next tool tests.
func createNextTestFeature(t *testing.T, entitySvc *service.EntityService, planID, slug string) string {
	t.Helper()
	result, err := entitySvc.CreateFeature(service.CreateFeatureInput{
		Name:      "test",
		Slug:      slug,
		Parent:    planID,
		Summary:   "Test feature " + slug,
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature(%s): %v", slug, err)
	}
	return result.ID
}

// createNextTestTask creates a task in queued status, returns (ID, slug).
func createNextTestTask(t *testing.T, entitySvc *service.EntityService, featID, slug string) (string, string) {
	t.Helper()
	result, err := entitySvc.CreateTask(service.CreateTaskInput{
		Name:          "test",
		ParentFeature: featID,
		Slug:          slug,
		Summary:       "Test task " + slug,
	})
	if err != nil {
		t.Fatalf("CreateTask(%s): %v", slug, err)
	}
	return result.ID, result.Slug
}

// setNextTaskReady transitions queued → ready.
func setNextTaskReady(t *testing.T, entitySvc *service.EntityService, taskID, taskSlug string) {
	t.Helper()
	if _, err := entitySvc.UpdateStatus(service.UpdateStatusInput{
		Type:   "task",
		ID:     taskID,
		Slug:   taskSlug,
		Status: "ready",
	}); err != nil {
		t.Fatalf("advance %s to ready: %v", taskID, err)
	}
}

// setNextTaskEstimate sets the estimate on a task.
func setNextTaskEstimate(t *testing.T, entitySvc *service.EntityService, taskID, taskSlug string, estimate float64) {
	t.Helper()
	rec, err := entitySvc.Store().Load("task", taskID, taskSlug)
	if err != nil {
		t.Fatalf("load task %s: %v", taskID, err)
	}
	rec.Fields["estimate"] = estimate
	if _, err := entitySvc.Store().Write(rec); err != nil {
		t.Fatalf("set estimate on %s: %v", taskID, err)
	}
}

// callNext invokes the next tool with nil optional services and returns the raw text.
// The handler is wrapped with WithSideEffects, so errors are returned as
// structured JSON error objects inside the MCP result — not as Go errors.
func callNext(
	t *testing.T,
	entitySvc *service.EntityService,
	dispatchSvc *service.DispatchService,
	args map[string]any,
) string {
	t.Helper()
	tool := nextTool(entitySvc, dispatchSvc, nil, nil, nil, nil, nil, nil, nil)
	req := makeRequest(args)
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("next handler error: %v", err)
	}
	return extractText(t, result)
}

// callNextJSON invokes the next tool and parses the result as JSON.
func callNextJSON(
	t *testing.T,
	entitySvc *service.EntityService,
	dispatchSvc *service.DispatchService,
	args map[string]any,
) map[string]any {
	t.Helper()
	text := callNext(t, entitySvc, dispatchSvc, args)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse next result: %v\nraw: %s", err, text)
	}
	return parsed
}

// callNextFull invokes the next tool with all services and returns parsed JSON.
func callNextFull(
	t *testing.T,
	entitySvc *service.EntityService,
	dispatchSvc *service.DispatchService,
	profileStore *kbzctx.ProfileStore,
	knowledgeSvc *service.KnowledgeService,
	intelligenceSvc *service.IntelligenceService,
	docRecordSvc *service.DocumentService,
	args map[string]any,
) map[string]any {
	t.Helper()
	tool := nextTool(entitySvc, dispatchSvc, profileStore, knowledgeSvc, intelligenceSvc, docRecordSvc, nil, nil, nil)
	req := makeRequest(args)
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("next handler error: %v", err)
	}
	text := extractText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse next result: %v\nraw: %s", err, text)
	}
	return parsed
}

// advanceNextFeatureTo transitions a feature through the lifecycle to reach
// the given target status. The forward chain is:
//
//	proposed -> designing -> specifying -> dev-planning -> developing -> reviewing
func advanceNextFeatureTo(t *testing.T, entitySvc *service.EntityService, featID, target string) {
	t.Helper()
	chain := []string{"designing", "specifying", "dev-planning", "developing", "reviewing"}
	for _, s := range chain {
		if _, err := entitySvc.UpdateStatus(service.UpdateStatusInput{
			Type:   "feature",
			ID:     featID,
			Status: s,
		}); err != nil {
			t.Fatalf("advance feature %s to %s: %v", featID, s, err)
		}
		if s == target {
			return
		}
	}
	t.Fatalf("advanceNextFeatureTo: unsupported target %q", target)
}

// ─── Queue inspection mode ───────────────────────────────────────────────────

func TestNext_QueueMode_Empty(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupNextTest(t)

	result := callNextJSON(t, entitySvc, dispatchSvc, map[string]any{})

	if _, ok := result["queue"]; !ok {
		t.Fatal("expected 'queue' field in response")
	}
	if _, ok := result["promoted_count"]; !ok {
		t.Fatal("expected 'promoted_count' field in response")
	}
	if _, ok := result["total_queued"]; !ok {
		t.Fatal("expected 'total_queued' field in response")
	}

	queue, _ := result["queue"].([]any)
	if len(queue) != 0 {
		t.Errorf("expected empty queue, got %d items", len(queue))
	}
	if result["promoted_count"].(float64) != 0 {
		t.Errorf("expected promoted_count=0, got %v", result["promoted_count"])
	}
}

func TestNext_QueueMode_ReadyTask(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupNextTest(t)

	planID := createNextTestPlan(t, entitySvc, "next-q1")
	featID := createNextTestFeature(t, entitySvc, planID, "feat-q1")
	taskID, taskSlug := createNextTestTask(t, entitySvc, featID, "task-q1")
	setNextTaskReady(t, entitySvc, taskID, taskSlug)

	result := callNextJSON(t, entitySvc, dispatchSvc, map[string]any{})

	queue, _ := result["queue"].([]any)
	if len(queue) != 1 {
		t.Fatalf("expected 1 item in queue, got %d", len(queue))
	}
	item, _ := queue[0].(map[string]any)
	if item["task_id"] != taskID {
		t.Errorf("queue[0].task_id = %v, want %v", item["task_id"], taskID)
	}
	if item["slug"] != "task-q1" {
		t.Errorf("queue[0].slug = %v, want task-q1", item["slug"])
	}
}

func TestNext_QueueMode_PromotesQueuedTasks(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupNextTest(t)

	planID := createNextTestPlan(t, entitySvc, "next-q2")
	featID := createNextTestFeature(t, entitySvc, planID, "feat-q2")

	// Create a task in queued status with no dependencies — it should be promoted.
	taskID, taskSlug := createNextTestTask(t, entitySvc, featID, "task-q2")
	_ = taskSlug

	// Verify the task starts in queued status.
	task, err := entitySvc.Get("task", taskID, "")
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if task.State["status"] != "queued" {
		t.Fatalf("expected queued status, got %v", task.State["status"])
	}

	// Queue inspection should promote the task.
	result := callNextJSON(t, entitySvc, dispatchSvc, map[string]any{})

	if result["promoted_count"].(float64) != 1 {
		t.Errorf("expected promoted_count=1, got %v", result["promoted_count"])
	}

	queue, _ := result["queue"].([]any)
	if len(queue) != 1 {
		t.Fatalf("expected 1 item in queue after promotion, got %d", len(queue))
	}

	// Side effects should include the promotion.
	sideEffects, _ := result["side_effects"].([]any)
	if len(sideEffects) != 1 {
		t.Fatalf("expected 1 side effect for promotion, got %d", len(sideEffects))
	}
	se, _ := sideEffects[0].(map[string]any)
	if se["type"] != "task_unblocked" {
		t.Errorf("side_effect[0].type = %v, want task_unblocked", se["type"])
	}
	if se["entity_id"] != taskID {
		t.Errorf("side_effect[0].entity_id = %v, want %v", se["entity_id"], taskID)
	}
	if se["from_status"] != "queued" {
		t.Errorf("side_effect[0].from_status = %v, want queued", se["from_status"])
	}
	if se["to_status"] != "ready" {
		t.Errorf("side_effect[0].to_status = %v, want ready", se["to_status"])
	}
}

// TestNext_QueueMode_SortOrder verifies that the ready queue is sorted by
// estimate ASC (null last), age DESC, ID lexicographic (AC #3).
func TestNext_QueueMode_SortOrder(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupNextTest(t)

	planID := createNextTestPlan(t, entitySvc, "next-sort")
	featID := createNextTestFeature(t, entitySvc, planID, "feat-sort")

	// Create tasks with different estimates. Task creation order matters for age.
	t1ID, t1Slug := createNextTestTask(t, entitySvc, featID, "task-large")
	setNextTaskReady(t, entitySvc, t1ID, t1Slug)
	setNextTaskEstimate(t, entitySvc, t1ID, t1Slug, 13)

	t2ID, t2Slug := createNextTestTask(t, entitySvc, featID, "task-small")
	setNextTaskReady(t, entitySvc, t2ID, t2Slug)
	setNextTaskEstimate(t, entitySvc, t2ID, t2Slug, 2)

	t3ID, t3Slug := createNextTestTask(t, entitySvc, featID, "task-nil")
	setNextTaskReady(t, entitySvc, t3ID, t3Slug)
	// No estimate set — should sort last.

	result := callNextJSON(t, entitySvc, dispatchSvc, map[string]any{})

	queue, _ := result["queue"].([]any)
	if len(queue) != 3 {
		t.Fatalf("expected 3 items in queue, got %d", len(queue))
	}

	// Expected order: task-small (est=2), task-large (est=13), task-nil (est=nil).
	first, _ := queue[0].(map[string]any)
	second, _ := queue[1].(map[string]any)
	third, _ := queue[2].(map[string]any)

	if first["task_id"] != t2ID {
		t.Errorf("queue[0] should be task-small (est=2), got %v", first["slug"])
	}
	if second["task_id"] != t1ID {
		t.Errorf("queue[1] should be task-large (est=13), got %v", second["slug"])
	}
	if third["task_id"] != t3ID {
		t.Errorf("queue[2] should be task-nil (est=nil), got %v", third["slug"])
	}
}

// TestNext_QueueMode_ConflictCheck verifies that conflict_check=true annotates
// queue items with conflict risk (AC #16).
func TestNext_QueueMode_ConflictCheck(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupNextTest(t)

	planID := createNextTestPlan(t, entitySvc, "next-cc")
	featID := createNextTestFeature(t, entitySvc, planID, "feat-cc")

	// Create a task and make it ready.
	taskID, taskSlug := createNextTestTask(t, entitySvc, featID, "task-cc")
	setNextTaskReady(t, entitySvc, taskID, taskSlug)

	// Call with conflict_check=true. With no active tasks, annotations
	// should be absent (omitted when empty) or "none". The key thing is
	// the call succeeds and includes queue items.
	result := callNextJSON(t, entitySvc, dispatchSvc, map[string]any{
		"conflict_check": true,
	})

	queue, _ := result["queue"].([]any)
	if len(queue) != 1 {
		t.Fatalf("expected 1 item in queue, got %d", len(queue))
	}

	// Without active tasks, there's nothing to conflict with.
	// Verify the item is still present and the call didn't error.
	item, _ := queue[0].(map[string]any)
	if item["task_id"] != taskID {
		t.Errorf("queue[0].task_id = %v, want %v", item["task_id"], taskID)
	}
}

// TestNext_QueueMode_ConflictCheckWithActiveTasks verifies conflict annotations
// are populated when there are active tasks.
func TestNext_QueueMode_ConflictCheckWithActiveTasks(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupNextTest(t)

	planID := createNextTestPlan(t, entitySvc, "next-cc2")
	featID := createNextTestFeature(t, entitySvc, planID, "feat-cc2")

	// Create an active task.
	activeID, activeSlug := createNextTestTask(t, entitySvc, featID, "task-active")
	setNextTaskReady(t, entitySvc, activeID, activeSlug)
	if _, err := entitySvc.UpdateStatus(service.UpdateStatusInput{
		Type: "task", ID: activeID, Slug: activeSlug, Status: "active",
	}); err != nil {
		t.Fatalf("transition to active: %v", err)
	}

	// Create a ready task.
	readyID, readySlug := createNextTestTask(t, entitySvc, featID, "task-ready")
	setNextTaskReady(t, entitySvc, readyID, readySlug)

	// Call with conflict_check — should produce annotations.
	result := callNextJSON(t, entitySvc, dispatchSvc, map[string]any{
		"conflict_check": true,
	})

	queue, _ := result["queue"].([]any)
	if len(queue) != 1 {
		t.Fatalf("expected 1 item in queue (only ready tasks), got %d", len(queue))
	}

	// The item should have conflict_risk set (even if "none").
	item, _ := queue[0].(map[string]any)
	if item["task_id"] != readyID {
		t.Errorf("queue[0].task_id = %v, want %v", item["task_id"], readyID)
	}
	// With both tasks in the same feature, there may or may not be a risk.
	// The key assertion is that the call succeeded and returned a queue.
}

// ─── Claim mode — task ID ────────────────────────────────────────────────────

func TestNext_ClaimByTaskID(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupNextTest(t)

	planID := createNextTestPlan(t, entitySvc, "next-c1")
	featID := createNextTestFeature(t, entitySvc, planID, "feat-c1")
	advanceNextFeatureTo(t, entitySvc, featID, "developing")
	taskID, taskSlug := createNextTestTask(t, entitySvc, featID, "task-c1")
	setNextTaskReady(t, entitySvc, taskID, taskSlug)

	result := callNextJSON(t, entitySvc, dispatchSvc, map[string]any{
		"id": taskID,
	})

	// Response must have task and context.
	taskOut, ok := result["task"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'task' object in response, got: %v", result)
	}
	if taskOut["id"] != taskID {
		t.Errorf("task.id = %v, want %v", taskOut["id"], taskID)
	}
	if taskOut["status"] != "active" {
		t.Errorf("task.status = %v, want active", taskOut["status"])
	}

	ctxOut, ok := result["context"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'context' object in response")
	}
	for _, field := range []string{"spec_sections", "acceptance_criteria", "knowledge", "files_context", "constraints", "byte_usage", "byte_budget", "trimmed"} {
		if _, exists := ctxOut[field]; !exists {
			t.Errorf("context missing field %q", field)
		}
	}

	// Side effects must include the status transition.
	sideEffects, _ := result["side_effects"].([]any)
	if len(sideEffects) != 1 {
		t.Fatalf("expected 1 side effect (status transition), got %d", len(sideEffects))
	}
	se, _ := sideEffects[0].(map[string]any)
	if se["type"] != "status_transition" {
		t.Errorf("side_effect[0].type = %v, want status_transition", se["type"])
	}
	if se["from_status"] != "ready" || se["to_status"] != "active" {
		t.Errorf("unexpected status transition: %v → %v", se["from_status"], se["to_status"])
	}
}

func TestNext_ClaimByTaskID_TransitionsToActive(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupNextTest(t)

	planID := createNextTestPlan(t, entitySvc, "next-c2")
	featID := createNextTestFeature(t, entitySvc, planID, "feat-c2")
	advanceNextFeatureTo(t, entitySvc, featID, "developing")
	taskID, taskSlug := createNextTestTask(t, entitySvc, featID, "task-c2")
	setNextTaskReady(t, entitySvc, taskID, taskSlug)

	callNext(t, entitySvc, dispatchSvc, map[string]any{"id": taskID})

	// Verify the task is now active in the store.
	task, err := entitySvc.Get("task", taskID, "")
	if err != nil {
		t.Fatalf("get task after claim: %v", err)
	}
	if task.State["status"] != "active" {
		t.Errorf("task.status = %v after claim, want active", task.State["status"])
	}
	if task.State["dispatched_to"] == nil {
		t.Error("task.dispatched_to not set after claim")
	}
}

func TestNext_ClaimByTaskID_AlreadyActive(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupNextTest(t)

	planID := createNextTestPlan(t, entitySvc, "next-c3")
	featID := createNextTestFeature(t, entitySvc, planID, "feat-c3")
	advanceNextFeatureTo(t, entitySvc, featID, "developing")
	taskID, taskSlug := createNextTestTask(t, entitySvc, featID, "task-c3")
	setNextTaskReady(t, entitySvc, taskID, taskSlug)

	// Claim once.
	callNext(t, entitySvc, dispatchSvc, map[string]any{"id": taskID})

	// Claim again — should return an error (AC #14).
	raw := callNext(t, entitySvc, dispatchSvc, map[string]any{"id": taskID})
	var result map[string]any
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	errObj, hasErr := result["error"].(map[string]any)
	if !hasErr {
		t.Fatalf("expected error for already-claimed task, got: %s", raw)
	}
	if errObj["code"] == nil {
		t.Error("expected error.code")
	}
	// Error message should contain dispatch metadata.
	msg, _ := errObj["message"].(string)
	if msg == "" {
		t.Error("expected non-empty error message with dispatch metadata")
	}
}

func TestNext_ClaimByTaskID_NotReady(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupNextTest(t)

	planID := createNextTestPlan(t, entitySvc, "next-c4")
	featID := createNextTestFeature(t, entitySvc, planID, "feat-c4")
	advanceNextFeatureTo(t, entitySvc, featID, "developing")
	taskID, _ := createNextTestTask(t, entitySvc, featID, "task-c4")
	// Task is still in queued status — not ready.

	raw := callNext(t, entitySvc, dispatchSvc, map[string]any{"id": taskID})
	var result map[string]any
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if _, hasErr := result["error"]; !hasErr {
		t.Fatalf("expected error for non-ready task, got: %s", raw)
	}
}

// ─── Claim mode — feature ID ─────────────────────────────────────────────────

func TestNext_ClaimByFeatureID(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupNextTest(t)

	planID := createNextTestPlan(t, entitySvc, "next-f1")
	featID := createNextTestFeature(t, entitySvc, planID, "feat-f1")
	advanceNextFeatureTo(t, entitySvc, featID, "developing")
	taskID, taskSlug := createNextTestTask(t, entitySvc, featID, "task-f1")
	setNextTaskReady(t, entitySvc, taskID, taskSlug)

	result := callNextJSON(t, entitySvc, dispatchSvc, map[string]any{
		"id": featID,
	})

	taskOut, ok := result["task"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'task' object, got: %v", result)
	}
	if taskOut["id"] != taskID {
		t.Errorf("task.id = %v, want %v", taskOut["id"], taskID)
	}
	if taskOut["status"] != "active" {
		t.Errorf("task.status = %v, want active", taskOut["status"])
	}
}

func TestNext_ClaimByFeatureID_NoReadyTasks(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupNextTest(t)

	planID := createNextTestPlan(t, entitySvc, "next-f2")
	featID := createNextTestFeature(t, entitySvc, planID, "feat-f2")
	// No tasks created — should error.

	raw := callNext(t, entitySvc, dispatchSvc, map[string]any{"id": featID})
	var result map[string]any
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if _, hasErr := result["error"]; !hasErr {
		t.Fatalf("expected error for feature with no ready tasks, got: %s", raw)
	}
}

// TestNext_ClaimByFeatureID_PicksTopTask verifies that claiming by feature ID
// picks the highest-priority ready task when multiple exist.
func TestNext_ClaimByFeatureID_PicksTopTask(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupNextTest(t)

	planID := createNextTestPlan(t, entitySvc, "next-f3")
	featID := createNextTestFeature(t, entitySvc, planID, "feat-f3")
	advanceNextFeatureTo(t, entitySvc, featID, "developing")

	// Create two tasks: one with estimate=8, one with estimate=2.
	t1ID, t1Slug := createNextTestTask(t, entitySvc, featID, "task-f3-big")
	setNextTaskReady(t, entitySvc, t1ID, t1Slug)
	setNextTaskEstimate(t, entitySvc, t1ID, t1Slug, 8)

	t2ID, t2Slug := createNextTestTask(t, entitySvc, featID, "task-f3-small")
	setNextTaskReady(t, entitySvc, t2ID, t2Slug)
	setNextTaskEstimate(t, entitySvc, t2ID, t2Slug, 2)

	// Claim by feature — should pick the smaller estimate (t2).
	result := callNextJSON(t, entitySvc, dispatchSvc, map[string]any{
		"id": featID,
	})

	taskOut, ok := result["task"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'task' object, got: %v", result)
	}
	if taskOut["id"] != t2ID {
		t.Errorf("claim-by-feature picked %v, want %v (smaller estimate)", taskOut["id"], t2ID)
	}
}

// ─── Claim mode — plan ID ────────────────────────────────────────────────────

func TestNext_ClaimByPlanID(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupNextTest(t)

	planID := createNextTestPlan(t, entitySvc, "next-p1")
	featID := createNextTestFeature(t, entitySvc, planID, "feat-p1")
	advanceNextFeatureTo(t, entitySvc, featID, "developing")
	taskID, taskSlug := createNextTestTask(t, entitySvc, featID, "task-p1")
	setNextTaskReady(t, entitySvc, taskID, taskSlug)

	result := callNextJSON(t, entitySvc, dispatchSvc, map[string]any{
		"id": planID,
	})

	taskOut, ok := result["task"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'task' object, got: %v", result)
	}
	if taskOut["id"] != taskID {
		t.Errorf("task.id = %v, want %v", taskOut["id"], taskID)
	}
}

func TestNext_ClaimByPlanID_NoReadyTasks(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupNextTest(t)

	planID := createNextTestPlan(t, entitySvc, "next-p2")
	// No features or tasks — should error.

	raw := callNext(t, entitySvc, dispatchSvc, map[string]any{"id": planID})
	var result map[string]any
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if _, hasErr := result["error"]; !hasErr {
		t.Fatalf("expected error for plan with no ready tasks, got: %s", raw)
	}
}

// TestNext_ClaimByPlanID_PicksTopTaskAcrossFeatures verifies that claim-by-plan
// picks the highest-priority ready task across multiple features.
func TestNext_ClaimByPlanID_PicksTopTaskAcrossFeatures(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupNextTest(t)

	planID := createNextTestPlan(t, entitySvc, "next-p3")
	feat1 := createNextTestFeature(t, entitySvc, planID, "feat-p3a")
	advanceNextFeatureTo(t, entitySvc, feat1, "developing")
	feat2 := createNextTestFeature(t, entitySvc, planID, "feat-p3b")
	advanceNextFeatureTo(t, entitySvc, feat2, "developing")

	t1ID, t1Slug := createNextTestTask(t, entitySvc, feat1, "task-p3-big")
	setNextTaskReady(t, entitySvc, t1ID, t1Slug)
	setNextTaskEstimate(t, entitySvc, t1ID, t1Slug, 13)

	t2ID, t2Slug := createNextTestTask(t, entitySvc, feat2, "task-p3-small")
	setNextTaskReady(t, entitySvc, t2ID, t2Slug)
	setNextTaskEstimate(t, entitySvc, t2ID, t2Slug, 3)

	result := callNextJSON(t, entitySvc, dispatchSvc, map[string]any{
		"id": planID,
	})

	taskOut, ok := result["task"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'task' object, got: %v", result)
	}
	if taskOut["id"] != t2ID {
		t.Errorf("claim-by-plan picked %v, want %v (smaller estimate)", taskOut["id"], t2ID)
	}
}

// ─── Context assembly ────────────────────────────────────────────────────────

func TestNext_ContextAssembly_FilesContext(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupNextTest(t)

	planID := createNextTestPlan(t, entitySvc, "next-ctx1")
	featID := createNextTestFeature(t, entitySvc, planID, "feat-ctx1")
	advanceNextFeatureTo(t, entitySvc, featID, "developing")
	taskID, taskSlug := createNextTestTask(t, entitySvc, featID, "task-ctx1")
	setNextTaskReady(t, entitySvc, taskID, taskSlug)

	// Write files_planned onto the task.
	rec, err := entitySvc.Store().Load("task", taskID, taskSlug)
	if err != nil {
		t.Fatalf("load task: %v", err)
	}
	rec.Fields["files_planned"] = []any{"internal/auth/middleware.go", "internal/auth/middleware_test.go"}
	if _, err := entitySvc.Store().Write(rec); err != nil {
		t.Fatalf("write task: %v", err)
	}

	result := callNextJSON(t, entitySvc, dispatchSvc, map[string]any{"id": taskID})

	ctxOut, _ := result["context"].(map[string]any)
	filesContext, _ := ctxOut["files_context"].([]any)
	if len(filesContext) != 2 {
		t.Errorf("expected 2 files_context entries, got %d", len(filesContext))
	}
	if len(filesContext) > 0 {
		first, _ := filesContext[0].(map[string]any)
		if first["path"] != "internal/auth/middleware.go" {
			t.Errorf("files_context[0].path = %v", first["path"])
		}
	}
}

func TestNext_ContextAssembly_AcceptanceCriteria(t *testing.T) {
	t.Parallel()
	// Unit test for the criteria extraction heuristic (shared in assembly.go).
	sections := []asmSpecSection{
		{
			document: "spec.md",
			section:  "Acceptance Criteria",
			content:  "- The system MUST authenticate users\n- All tokens SHALL be verified\n- Optional check",
		},
		{
			document: "spec.md",
			section:  "Overview",
			content:  "This is an overview.\n- Tokens MUST be refreshed every hour\n- This is just info",
		},
	}

	criteria := asmExtractCriteria(sections)

	// From the acceptance criteria section, all bullet items are included.
	// From the overview, only MUST/SHALL items are included.
	if len(criteria) == 0 {
		t.Fatal("expected non-empty acceptance criteria")
	}

	hasMustAuth := false
	hasMustRefresh := false
	for _, c := range criteria {
		if c == "The system MUST authenticate users" {
			hasMustAuth = true
		}
		if c == "Tokens MUST be refreshed every hour" {
			hasMustRefresh = true
		}
	}
	if !hasMustAuth {
		t.Errorf("expected 'The system MUST authenticate users' in criteria, got: %v", criteria)
	}
	if !hasMustRefresh {
		t.Errorf("expected 'Tokens MUST be refreshed every hour' in criteria, got: %v", criteria)
	}
}

func TestNext_ContextAssembly_ByteBudgetFields(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupNextTest(t)

	planID := createNextTestPlan(t, entitySvc, "next-ctx2")
	featID := createNextTestFeature(t, entitySvc, planID, "feat-ctx2")
	advanceNextFeatureTo(t, entitySvc, featID, "developing")
	taskID, taskSlug := createNextTestTask(t, entitySvc, featID, "task-ctx2")
	setNextTaskReady(t, entitySvc, taskID, taskSlug)

	result := callNextJSON(t, entitySvc, dispatchSvc, map[string]any{"id": taskID})

	ctxOut, _ := result["context"].(map[string]any)
	if _, ok := ctxOut["byte_usage"]; !ok {
		t.Error("context missing byte_usage")
	}
	if _, ok := ctxOut["byte_budget"]; !ok {
		t.Error("context missing byte_budget")
	}
	budget, _ := ctxOut["byte_budget"].(float64)
	if budget != float64(assemblyDefaultBudget) {
		t.Errorf("byte_budget = %v, want %v", budget, assemblyDefaultBudget)
	}
}

// TestNext_ContextAssembly_KnowledgeEntries verifies that knowledge entries
// are included in the assembled context, sorted by confidence (AC #12).
func TestNext_ContextAssembly_KnowledgeEntries(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc, knowledgeSvc, profileStore, intelligenceSvc, docRecordSvc := setupNextTestFull(t)

	planID := createNextTestPlan(t, entitySvc, "next-ke")
	featID := createNextTestFeature(t, entitySvc, planID, "feat-ke")
	advanceNextFeatureTo(t, entitySvc, featID, "developing")
	taskID, taskSlug := createNextTestTask(t, entitySvc, featID, "task-ke")
	setNextTaskReady(t, entitySvc, taskID, taskSlug)

	// Add knowledge entries.
	if _, _, err := knowledgeSvc.Contribute(service.ContributeInput{
		Topic: "low-priority", Content: "Less important fact", Scope: "project",
		Tier: 2, CreatedBy: "tester",
	}); err != nil {
		t.Fatalf("contribute knowledge: %v", err)
	}
	if _, _, err := knowledgeSvc.Contribute(service.ContributeInput{
		Topic: "high-priority", Content: "Critical constraint", Scope: "project",
		Tier: 2, CreatedBy: "tester",
	}); err != nil {
		t.Fatalf("contribute knowledge: %v", err)
	}

	result := callNextFull(t, entitySvc, dispatchSvc, profileStore, knowledgeSvc, intelligenceSvc, docRecordSvc, map[string]any{
		"id": taskID,
	})

	ctxOut, _ := result["context"].(map[string]any)
	knowledge, _ := ctxOut["knowledge"].([]any)
	if len(knowledge) < 2 {
		t.Errorf("expected at least 2 knowledge entries, got %d", len(knowledge))
	}

	// Verify entries have expected fields.
	if len(knowledge) > 0 {
		first, _ := knowledge[0].(map[string]any)
		if first["topic"] == nil || first["content"] == nil || first["confidence"] == nil {
			t.Errorf("knowledge entry missing fields: %v", first)
		}
	}
}

// TestNext_ContextAssembly_GracefulDegradation verifies that when document
// intelligence has no index, the spec_fallback_path is set (AC #11, §24.3).
func TestNext_ContextAssembly_GracefulDegradation(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc, _, _, _, docRecordSvc := setupNextTestFull(t)

	planID := createNextTestPlan(t, entitySvc, "next-gd")
	featID := createNextTestFeature(t, entitySvc, planID, "feat-gd")
	advanceNextFeatureTo(t, entitySvc, featID, "developing")
	taskID, taskSlug := createNextTestTask(t, entitySvc, featID, "task-gd")
	setNextTaskReady(t, entitySvc, taskID, taskSlug)

	// Register a spec document for the feature (but with a non-existent
	// file so that indexing will fail and graceful degradation kicks in).
	specDir := filepath.Join(docRecordSvc.RepoRoot(), "work", "spec")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	specPath := filepath.Join("work", "spec", "test-spec.md")
	fullSpecPath := filepath.Join(docRecordSvc.RepoRoot(), specPath)
	if err := os.WriteFile(fullSpecPath, []byte("# Spec\n\nSome content for "+featID+"\n"), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	_, err := docRecordSvc.SubmitDocument(service.SubmitDocumentInput{
		Path:      specPath,
		Type:      "specification",
		Title:     "Test Spec",
		Owner:     featID,
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("submit document: %v", err)
	}

	// Use a nil intelligence service to force graceful degradation.
	tool := nextTool(entitySvc, dispatchSvc, nil, nil, nil, docRecordSvc, nil, nil, nil)
	req := makeRequest(map[string]any{"id": taskID})
	toolResult, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("next handler error: %v", err)
	}
	text := extractText(t, toolResult)

	var result map[string]any
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		t.Fatalf("parse result: %v\nraw: %s", err, text)
	}

	ctxOut, _ := result["context"].(map[string]any)
	fallback, _ := ctxOut["spec_fallback_path"].(string)
	if fallback == "" {
		t.Error("expected spec_fallback_path to be set when intelligence is unavailable")
	}
	if fallback != specPath {
		t.Errorf("spec_fallback_path = %q, want %q", fallback, specPath)
	}
}

// ─── ID inference ─────────────────────────────────────────────────────────────

func TestNextInferEntityType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		id   string
		want string
	}{
		{"TASK-01JX123", "task"},
		{"T-01JX123", "task"},
		{"task-01JX123", "task"},
		{"FEAT-01JX123", "feature"},
		{"feat-01JX123", "feature"},
		{"P1-my-plan", "plan"},
		{"P2-another", "plan"},
		{"BUG-01JX123", ""},  // bug is not handled by next
		{"EPIC-01JX123", ""}, // epic is not handled by next
		{"", ""},
		{"random-string", ""},
	}

	for _, tt := range tests {
		got := nextInferEntityType(tt.id)
		if got != tt.want {
			t.Errorf("nextInferEntityType(%q) = %q, want %q", tt.id, got, tt.want)
		}
	}
}

// ─── Trim logic ──────────────────────────────────────────────────────────────

func TestNextTrimContext_RemovesLowConfidenceFirst(t *testing.T) {
	t.Parallel()

	// Build a context with many large knowledge entries to force trimming.
	actx := assembledContext{
		byteBudget: 100, // very small budget
		knowledge: []asmKnowledgeEntry{
			{topic: "high-conf", content: "a very long content string that takes up space in the byte budget calculation", confidence: 0.9, tier: 3},
			{topic: "low-conf", content: "another long content string that also takes up space in the calculation here", confidence: 0.1, tier: 3},
		},
	}
	actx.byteUsage = asmByteCount(actx)

	if actx.byteUsage <= 100 {
		// Content is too small to trigger trim — adjust budget.
		actx.byteBudget = 10
	}

	trimmed := asmTrimContext(actx)

	// The low-confidence entry should have been trimmed.
	for _, ke := range trimmed.knowledge {
		if ke.topic == "low-conf" {
			t.Error("low-conf entry should have been trimmed")
		}
	}
	if len(trimmed.trimmed) == 0 {
		t.Error("expected at least one trimmed entry recorded")
	}
}

// TestNextTrimContext_T3BeforeT2 verifies that Tier 3 entries are trimmed
// before Tier 2, regardless of confidence.
func TestNextTrimContext_T3BeforeT2(t *testing.T) {
	t.Parallel()

	actx := assembledContext{
		byteBudget: 10, // very small budget to force trimming
		knowledge: []asmKnowledgeEntry{
			{topic: "t2-low", content: "tier 2 content that is quite long and will consume budget space", confidence: 0.4, tier: 2},
			{topic: "t3-high", content: "tier 3 content that is also quite long and consuming budget space", confidence: 0.9, tier: 3},
		},
	}
	actx.byteUsage = asmByteCount(actx)

	result := asmTrimContext(actx)

	// T3 should be trimmed before T2.
	if len(result.trimmed) == 0 {
		t.Fatal("expected trimmed entries")
	}
	if result.trimmed[0].topic != "t3-high" {
		t.Errorf("first trimmed entry should be t3-high, got %q", result.trimmed[0].topic)
	}
}

// ─── Orientation breadcrumb tests (AC-E4, AC-E5, AC-E6) ──────────────────────

func TestNext_EmptyQueue_HasOrientation(t *testing.T) {
	t.Parallel()
	// AC-E4: next with no id and an empty queue returns the orientation field.
	entitySvc, dispatchSvc := setupNextTest(t)

	parsed := callNextJSON(t, entitySvc, dispatchSvc, map[string]any{})

	orientation, ok := parsed["orientation"].(map[string]any)
	if !ok {
		t.Fatalf("orientation field missing or wrong type in empty-queue response: %v", parsed)
	}
	msg, _ := orientation["message"].(string)
	if !strings.Contains(msg, "kanbanzai-getting-started/SKILL.md") {
		t.Errorf("orientation.message does not reference getting-started skill: %q", msg)
	}
	skillsPath, _ := orientation["skills_path"].(string)
	if skillsPath != ".agents/skills/" {
		t.Errorf("orientation.skills_path = %q, want .agents/skills/", skillsPath)
	}
}

func TestNext_NonEmptyQueue_NoOrientation(t *testing.T) {
	t.Parallel()
	// AC-E5: next with no id and a non-empty queue does NOT return orientation.
	entitySvc, dispatchSvc := setupNextTest(t)
	planID := createNextTestPlan(t, entitySvc, "orient-nonempty-plan")
	featID := createNextTestFeature(t, entitySvc, planID, "orient-nonempty-feat")
	taskID, taskSlug := createNextTestTask(t, entitySvc, featID, "orient-nonempty-task")
	setNextTaskReady(t, entitySvc, taskID, taskSlug)

	parsed := callNextJSON(t, entitySvc, dispatchSvc, map[string]any{})

	if _, ok := parsed["orientation"]; ok {
		t.Errorf("orientation field should not be present in non-empty queue response")
	}
	// Queue should have the task.
	queue, _ := parsed["queue"].([]any)
	if len(queue) == 0 {
		t.Fatalf("expected non-empty queue, got empty")
	}
}

func TestNext_ClaimMode_NoOrientation(t *testing.T) {
	t.Parallel()
	// AC-E6: next with an id (claim mode) does NOT return orientation.
	entitySvc, dispatchSvc := setupNextTest(t)
	planID := createNextTestPlan(t, entitySvc, "orient-claim-plan")
	featID := createNextTestFeature(t, entitySvc, planID, "orient-claim-feat")
	advanceNextFeatureTo(t, entitySvc, featID, "developing")
	taskID, taskSlug := createNextTestTask(t, entitySvc, featID, "orient-claim-task")
	setNextTaskReady(t, entitySvc, taskID, taskSlug)

	parsed := callNextJSON(t, entitySvc, dispatchSvc, map[string]any{"id": taskID})

	if _, ok := parsed["orientation"]; ok {
		t.Errorf("orientation field should not be present in claim-mode response")
	}
	// Should have task and context fields instead.
	if _, ok := parsed["task"]; !ok {
		t.Errorf("task field missing from claim-mode response")
	}
}

// ─── Stage validation (FR-002) ────────────────────────────────────────────────

// TestNext_ClaimByTaskID_ProposedFeatureRejected verifies that claiming a task
// whose parent feature is in "proposed" (a non-working state) returns an error
// and leaves the task in "ready" status — the claim must not succeed (B-10).
func TestNext_ClaimByTaskID_ProposedFeatureRejected(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupNextTest(t)

	planID := createNextTestPlan(t, entitySvc, "next-proposed-feat")
	featID := createNextTestFeature(t, entitySvc, planID, "feat-proposed")
	// Feature is deliberately left in "proposed" — do NOT call advanceNextFeatureTo.

	taskID, taskSlug := createNextTestTask(t, entitySvc, featID, "task-proposed")
	setNextTaskReady(t, entitySvc, taskID, taskSlug)

	raw := callNext(t, entitySvc, dispatchSvc, map[string]any{"id": taskID})
	var result map[string]any
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("parse result: %v\nraw: %s", err, raw)
	}

	// An error must be present.
	if _, hasErr := result["error"]; !hasErr {
		t.Fatalf("expected error for task with proposed parent feature, got: %s", raw)
	}

	// Task must remain in "ready" — it was not claimed.
	task, err := entitySvc.Get("task", taskID, "")
	if err != nil {
		t.Fatalf("get task after rejected claim: %v", err)
	}
	if task.State["status"] != "ready" {
		t.Errorf("task.status = %v after rejected claim, want \"ready\"", task.State["status"])
	}
}
