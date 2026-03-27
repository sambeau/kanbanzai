package mcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"kanbanzai/internal/service"
	"kanbanzai/internal/storage"
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

// callNext invokes the next tool and returns the raw text response.
func callNext(
	t *testing.T,
	entitySvc *service.EntityService,
	dispatchSvc *service.DispatchService,
	args map[string]any,
) string {
	t.Helper()
	tool := nextTool(entitySvc, dispatchSvc, nil, nil, nil)
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

// ─── Claim mode — task ID ────────────────────────────────────────────────────

func TestNext_ClaimByTaskID(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupNextTest(t)

	planID := createNextTestPlan(t, entitySvc, "next-c1")
	featID := createNextTestFeature(t, entitySvc, planID, "feat-c1")
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
	taskID, taskSlug := createNextTestTask(t, entitySvc, featID, "task-c3")
	setNextTaskReady(t, entitySvc, taskID, taskSlug)

	// Claim once.
	callNext(t, entitySvc, dispatchSvc, map[string]any{"id": taskID})

	// Claim again — should return an error.
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
}

func TestNext_ClaimByTaskID_NotReady(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupNextTest(t)

	planID := createNextTestPlan(t, entitySvc, "next-c4")
	featID := createNextTestFeature(t, entitySvc, planID, "feat-c4")
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

// ─── Claim mode — plan ID ────────────────────────────────────────────────────

func TestNext_ClaimByPlanID(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupNextTest(t)

	planID := createNextTestPlan(t, entitySvc, "next-p1")
	featID := createNextTestFeature(t, entitySvc, planID, "feat-p1")
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

// ─── Context assembly ────────────────────────────────────────────────────────

func TestNext_ContextAssembly_FilesContext(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupNextTest(t)

	planID := createNextTestPlan(t, entitySvc, "next-ctx1")
	featID := createNextTestFeature(t, entitySvc, planID, "feat-ctx1")
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
	// Unit test for the criteria extraction heuristic.
	sections := []nextSpecSection{
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

	criteria := nextExtractCriteria(sections)

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
	if budget != float64(nextDefaultBudget) {
		t.Errorf("byte_budget = %v, want %v", budget, nextDefaultBudget)
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
	nctx := nextCtxData{
		byteBudget: 100, // very small budget
		knowledge: []nextKnowledgeEntry{
			{topic: "high-conf", content: "a very long content string that takes up space in the byte budget calculation", confidence: 0.9, tier: 3},
			{topic: "low-conf", content: "another long content string that also takes up space in the calculation here", confidence: 0.1, tier: 3},
		},
	}
	nctx.byteUsage = nextByteCount(nctx)

	if nctx.byteUsage <= 100 {
		// Content is too small to trigger trim — adjust budget.
		nctx.byteBudget = 10
	}

	trimmed := nextTrimContext(nctx)

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
