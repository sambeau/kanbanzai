package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"kanbanzai/internal/service"
	"kanbanzai/internal/storage"
)

// ─── Test helpers ─────────────────────────────────────────────────────────────

// setupFinishTest creates the services needed for finish tool tests.
// Returns entitySvc and dispatchSvc backed by temp dirs, with no worktree
// hook configured (avoids git calls in tests).
func setupFinishTest(t *testing.T) (*service.EntityService, *service.DispatchService) {
	t.Helper()
	entityRoot := t.TempDir()
	stateRoot := t.TempDir()
	entitySvc := service.NewEntityService(entityRoot)
	knowledgeSvc := service.NewKnowledgeService(stateRoot)
	dispatchSvc := service.NewDispatchService(entitySvc, knowledgeSvc)
	return entitySvc, dispatchSvc
}

// createFinishTestPlan writes a plan record directly, bypassing config-dependent
// CreatePlan so tests work without a .kbz/config.yaml file.
func createFinishTestPlan(t *testing.T, entitySvc *service.EntityService, slug string) string {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	id := "P1-" + slug
	record := storage.EntityRecord{
		Type: "plan",
		ID:   id,
		Slug: slug,
		Fields: map[string]any{
			"id":         id,
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
		t.Fatalf("createFinishTestPlan(%s): %v", slug, err)
	}
	return id
}

// createFinishTestFeature creates a feature for finish tests.
func createFinishTestFeature(t *testing.T, entitySvc *service.EntityService, planID, slug string) string {
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

// createFinishTestTask creates a task in queued status and returns its (ID, slug).
func createFinishTestTask(t *testing.T, entitySvc *service.EntityService, featID, slug string) (string, string) {
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

// advanceToReady transitions a task from queued → ready.
func advanceToReady(t *testing.T, entitySvc *service.EntityService, taskID, taskSlug string) {
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

// advanceToActive transitions a task from queued → ready → active.
func advanceToActive(t *testing.T, entitySvc *service.EntityService, taskID, taskSlug string) {
	t.Helper()
	advanceToReady(t, entitySvc, taskID, taskSlug)
	if _, err := entitySvc.UpdateStatus(service.UpdateStatusInput{
		Type:   "task",
		ID:     taskID,
		Slug:   taskSlug,
		Status: "active",
	}); err != nil {
		t.Fatalf("advance %s to active: %v", taskID, err)
	}
}

// setFinishDependsOn writes a depends_on list onto a task record directly.
func setFinishDependsOn(t *testing.T, entitySvc *service.EntityService, taskID, taskSlug string, deps []string) {
	t.Helper()
	rec, err := entitySvc.Store().Load("task", taskID, taskSlug)
	if err != nil {
		t.Fatalf("load task %s for setDependsOn: %v", taskID, err)
	}
	depsAny := make([]any, len(deps))
	for i, d := range deps {
		depsAny[i] = d
	}
	rec.Fields["depends_on"] = depsAny
	if _, err := entitySvc.Store().Write(rec); err != nil {
		t.Fatalf("write depends_on for %s: %v", taskID, err)
	}
}

// callFinish invokes the finish tool directly and returns the raw text response.
func callFinish(t *testing.T, entitySvc *service.EntityService, dispatchSvc *service.DispatchService, args map[string]any) string {
	t.Helper()
	tool := finishTool(entitySvc, dispatchSvc)
	req := makeRequest(args)
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("finish handler error: %v", err)
	}
	return extractText(t, result)
}

// callFinishJSON invokes the finish tool and parses the result as JSON.
func callFinishJSON(t *testing.T, entitySvc *service.EntityService, dispatchSvc *service.DispatchService, args map[string]any) map[string]any {
	t.Helper()
	text := callFinish(t, entitySvc, dispatchSvc, args)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse finish result: %v\nraw: %s", err, text)
	}
	return parsed
}

// setupFinishScenario is a convenience helper that creates a plan → feature → task
// chain and returns the task ID and slug.
func setupFinishScenario(t *testing.T, entitySvc *service.EntityService, suffix string) (taskID, taskSlug string) {
	t.Helper()
	planID := createFinishTestPlan(t, entitySvc, "fin-plan-"+suffix)
	featID := createFinishTestFeature(t, entitySvc, planID, "fin-feat-"+suffix)
	taskID, taskSlug = createFinishTestTask(t, entitySvc, featID, "fin-task-"+suffix)
	return taskID, taskSlug
}

// ─── Single-item completion tests ─────────────────────────────────────────────

// TestFinish_ActiveTask completes a task that is already in active status.
// Acceptance criterion: finish(task_id, summary) completes a task in active status.
func TestFinish_ActiveTask(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupFinishTest(t)
	taskID, taskSlug := setupFinishScenario(t, entitySvc, "active")
	advanceToActive(t, entitySvc, taskID, taskSlug)

	resp := callFinishJSON(t, entitySvc, dispatchSvc, map[string]any{
		"task_id": taskID,
		"summary": "Completed the implementation",
	})

	// Task field must be present.
	taskData, ok := resp["task"].(map[string]any)
	if !ok {
		t.Fatalf("expected task in response, got: %v", resp)
	}

	// Status must be done (default).
	if got := taskData["status"]; got != "done" {
		t.Errorf("task status = %q, want \"done\"", got)
	}

	// Acceptance criterion: completion_summary and completed timestamp are set.
	if got := taskData["completion_summary"]; got != "Completed the implementation" {
		t.Errorf("completion_summary = %q, want \"Completed the implementation\"", got)
	}
	if completed, _ := taskData["completed"].(string); completed == "" {
		t.Error("completed timestamp is missing or empty")
	}
}

// TestFinish_DefaultsToDone verifies that omitting to_status results in "done".
// Acceptance criterion: finish defaults to to_status: "done".
func TestFinish_DefaultsToDone(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupFinishTest(t)
	taskID, taskSlug := setupFinishScenario(t, entitySvc, "default-done")
	advanceToActive(t, entitySvc, taskID, taskSlug)

	resp := callFinishJSON(t, entitySvc, dispatchSvc, map[string]any{
		"task_id": taskID,
		"summary": "Done",
	})

	taskData := resp["task"].(map[string]any)
	if got := taskData["status"]; got != "done" {
		t.Errorf("status = %q, want \"done\"", got)
	}
}

// TestFinish_ToNeedsReview completes a task targeting needs-review status.
// Acceptance criterion: finish with to_status: "needs-review" transitions to needs-review.
func TestFinish_ToNeedsReview(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupFinishTest(t)
	taskID, taskSlug := setupFinishScenario(t, entitySvc, "needs-review")
	advanceToActive(t, entitySvc, taskID, taskSlug)

	resp := callFinishJSON(t, entitySvc, dispatchSvc, map[string]any{
		"task_id":   taskID,
		"summary":   "Implementation complete, please review",
		"to_status": "needs-review",
	})

	taskData, ok := resp["task"].(map[string]any)
	if !ok {
		t.Fatalf("expected task in response, got: %v", resp)
	}
	if got := taskData["status"]; got != "needs-review" {
		t.Errorf("status = %q, want \"needs-review\"", got)
	}
}

// TestFinish_CompletionMetadata verifies both timestamp and summary are persisted.
// Acceptance criterion: completion_summary and completed timestamp are set on the task.
func TestFinish_CompletionMetadata(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupFinishTest(t)
	taskID, taskSlug := setupFinishScenario(t, entitySvc, "metadata")
	advanceToActive(t, entitySvc, taskID, taskSlug)

	const wantSummary = "Added JWT middleware with full test coverage"
	resp := callFinishJSON(t, entitySvc, dispatchSvc, map[string]any{
		"task_id": taskID,
		"summary": wantSummary,
	})

	taskData := resp["task"].(map[string]any)

	if got := taskData["completion_summary"]; got != wantSummary {
		t.Errorf("completion_summary = %q, want %q", got, wantSummary)
	}

	completed, _ := taskData["completed"].(string)
	if completed == "" {
		t.Fatal("completed timestamp is missing")
	}
	// Verify it's a valid RFC3339 timestamp.
	if _, err := time.Parse(time.RFC3339, completed); err != nil {
		t.Errorf("completed %q is not a valid RFC3339 timestamp: %v", completed, err)
	}
}

// ─── Lenient lifecycle tests ──────────────────────────────────────────────────

// TestFinish_ReadyTask verifies lenient lifecycle: a ready task is accepted and completed.
// Acceptance criterion: finish accepts tasks in ready status (lenient lifecycle).
func TestFinish_ReadyTask(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupFinishTest(t)
	taskID, taskSlug := setupFinishScenario(t, entitySvc, "ready")
	advanceToReady(t, entitySvc, taskID, taskSlug)

	resp := callFinishJSON(t, entitySvc, dispatchSvc, map[string]any{
		"task_id": taskID,
		"summary": "Completed without explicit dispatch",
	})

	taskData, ok := resp["task"].(map[string]any)
	if !ok {
		t.Fatalf("expected task in response, got: %v", resp)
	}
	if got := taskData["status"]; got != "done" {
		t.Errorf("status = %q, want \"done\"", got)
	}
	if taskData["completion_summary"] != "Completed without explicit dispatch" {
		t.Errorf("completion_summary not set, got: %v", taskData["completion_summary"])
	}
}

// TestFinish_ReadyTask_ToNeedsReview verifies lenient lifecycle + needs-review target.
func TestFinish_ReadyTask_ToNeedsReview(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupFinishTest(t)
	taskID, taskSlug := setupFinishScenario(t, entitySvc, "ready-nr")
	advanceToReady(t, entitySvc, taskID, taskSlug)

	resp := callFinishJSON(t, entitySvc, dispatchSvc, map[string]any{
		"task_id":   taskID,
		"summary":   "Done, please review",
		"to_status": "needs-review",
	})

	taskData := resp["task"].(map[string]any)
	if got := taskData["status"]; got != "needs-review" {
		t.Errorf("status = %q, want \"needs-review\"", got)
	}
}

// ─── Inline knowledge tests ───────────────────────────────────────────────────

// TestFinish_KnowledgeAccepted verifies inline knowledge entries are processed.
// Acceptance criterion: inline knowledge entries are processed through the contribution pipeline.
// Acceptance criterion: knowledge contributions are reported in the response.
func TestFinish_KnowledgeAccepted(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupFinishTest(t)
	taskID, taskSlug := setupFinishScenario(t, entitySvc, "ke-accept")
	advanceToActive(t, entitySvc, taskID, taskSlug)

	resp := callFinishJSON(t, entitySvc, dispatchSvc, map[string]any{
		"task_id": taskID,
		"summary": "Implemented feature",
		"knowledge": []any{
			map[string]any{
				"topic":   "billing-api-idempotency",
				"content": "The billing API requires idempotency keys on all POST requests",
				"scope":   "backend",
			},
		},
	})

	ke, ok := resp["knowledge"].(map[string]any)
	if !ok {
		t.Fatalf("expected knowledge in response, got: %v", resp)
	}

	accepted, ok := ke["accepted"].([]any)
	if !ok || len(accepted) != 1 {
		t.Fatalf("expected 1 accepted entry, got: %v", ke["accepted"])
	}

	entry := accepted[0].(map[string]any)
	if got := entry["topic"]; got != "billing-api-idempotency" {
		t.Errorf("accepted entry topic = %q, want \"billing-api-idempotency\"", got)
	}
	if entryID, _ := entry["entry_id"].(string); entryID == "" {
		t.Error("accepted entry missing entry_id")
	}

	if got := ke["total_attempted"]; got != float64(1) {
		t.Errorf("total_attempted = %v, want 1", got)
	}
	if got := ke["total_accepted"]; got != float64(1) {
		t.Errorf("total_accepted = %v, want 1", got)
	}

	// Knowledge contribution must also appear in side_effects.
	sideEffects, _ := resp["side_effects"].([]any)
	var foundKE bool
	for _, se := range sideEffects {
		effect := se.(map[string]any)
		if effect["type"] == "knowledge_contributed" {
			foundKE = true
		}
	}
	if !foundKE {
		t.Errorf("expected knowledge_contributed side effect, got side_effects: %v", sideEffects)
	}
}

// TestFinish_KnowledgeDuplicateRejected verifies that duplicate entries are rejected
// per-entry without blocking the overall task completion.
// Acceptance criterion: duplicate knowledge entries are rejected per-entry without blocking completion.
func TestFinish_KnowledgeDuplicateRejected(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupFinishTest(t)
	taskID, taskSlug := setupFinishScenario(t, entitySvc, "ke-dup")
	advanceToActive(t, entitySvc, taskID, taskSlug)

	// First completion: contributes the entry successfully.
	callFinishJSON(t, entitySvc, dispatchSvc, map[string]any{
		"task_id": taskID,
		"summary": "First completion",
		"knowledge": []any{
			map[string]any{
				"topic":   "retry-backoff",
				"content": "Retry delays follow exponential backoff with jitter",
				"scope":   "project",
			},
		},
	})

	// Second task: contributes the exact same topic → should be rejected.
	taskID2, taskSlug2 := setupFinishScenario(t, entitySvc, "ke-dup2")
	advanceToActive(t, entitySvc, taskID2, taskSlug2)

	resp := callFinishJSON(t, entitySvc, dispatchSvc, map[string]any{
		"task_id": taskID2,
		"summary": "Second completion — duplicate knowledge",
		"knowledge": []any{
			map[string]any{
				"topic":   "retry-backoff",
				"content": "Retry delays follow exponential backoff with jitter",
				"scope":   "project",
			},
		},
	})

	// Task must still be completed despite knowledge rejection.
	taskData, ok := resp["task"].(map[string]any)
	if !ok {
		t.Fatalf("expected task in response, got: %v", resp)
	}
	if got := taskData["status"]; got != "done" {
		t.Errorf("task status = %q, want \"done\" (completion should not be blocked by KE rejection)", got)
	}

	ke := resp["knowledge"].(map[string]any)
	rejected, ok := ke["rejected"].([]any)
	if !ok || len(rejected) == 0 {
		t.Fatalf("expected rejected entries, got: %v", ke["rejected"])
	}
	rejEntry := rejected[0].(map[string]any)
	if rejEntry["topic"] != "retry-backoff" {
		t.Errorf("rejected topic = %q, want \"retry-backoff\"", rejEntry["topic"])
	}
	if reason, _ := rejEntry["reason"].(string); reason == "" {
		t.Error("rejected entry missing reason")
	}

	// Rejection must appear in side_effects too.
	sideEffects, _ := resp["side_effects"].([]any)
	var foundRejected bool
	for _, se := range sideEffects {
		effect := se.(map[string]any)
		if effect["type"] == "knowledge_rejected" {
			foundRejected = true
		}
	}
	if !foundRejected {
		t.Errorf("expected knowledge_rejected side effect, got side_effects: %v", sideEffects)
	}
}

// TestFinish_MultipleKnowledgeEntries verifies partial acceptance across multiple entries.
func TestFinish_MultipleKnowledgeEntries(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupFinishTest(t)
	taskID, taskSlug := setupFinishScenario(t, entitySvc, "ke-multi")
	advanceToActive(t, entitySvc, taskID, taskSlug)

	resp := callFinishJSON(t, entitySvc, dispatchSvc, map[string]any{
		"task_id": taskID,
		"summary": "Multi-knowledge task",
		"knowledge": []any{
			map[string]any{
				"topic":   "jwt-verification",
				"content": "JWT tokens use RS256 signing",
				"scope":   "backend",
			},
			map[string]any{
				"topic":   "cache-invalidation",
				"content": "Cache is invalidated on write operations",
				"scope":   "backend",
			},
		},
	})

	ke := resp["knowledge"].(map[string]any)
	if got := ke["total_attempted"]; got != float64(2) {
		t.Errorf("total_attempted = %v, want 2", got)
	}
	if got := ke["total_accepted"]; got != float64(2) {
		t.Errorf("total_accepted = %v, want 2", got)
	}
}

// ─── Side-effect: dependency unblocking ──────────────────────────────────────

// TestFinish_UnblockedTasksInSideEffects verifies that tasks unblocked by completion
// appear in the side_effects array.
// Acceptance criterion: unblocked tasks are reported in side_effects.
func TestFinish_UnblockedTasksInSideEffects(t *testing.T) {
	t.Parallel()

	entityRoot := t.TempDir()
	stateRoot := t.TempDir()
	entitySvc := service.NewEntityService(entityRoot)
	knowledgeSvc := service.NewKnowledgeService(stateRoot)
	dispatchSvc := service.NewDispatchService(entitySvc, knowledgeSvc)

	// Wire the dependency unblocking hook.
	hook := service.NewDependencyUnblockingHook(entitySvc)
	entitySvc.SetStatusTransitionHook(hook)

	planID := createFinishTestPlan(t, entitySvc, "unblock")
	featID := createFinishTestFeature(t, entitySvc, planID, "unblock-feat")

	// Task A has no deps; task B depends on A.
	taskAID, taskASlug := createFinishTestTask(t, entitySvc, featID, "unblock-a")
	taskBID, taskBSlug := createFinishTestTask(t, entitySvc, featID, "unblock-b")

	setFinishDependsOn(t, entitySvc, taskBID, taskBSlug, []string{taskAID})

	// Advance A to active and complete it.
	advanceToActive(t, entitySvc, taskAID, taskASlug)

	resp := callFinishJSON(t, entitySvc, dispatchSvc, map[string]any{
		"task_id": taskAID,
		"summary": "A is done",
	})

	// Side effects must contain task_unblocked for task B.
	sideEffects, _ := resp["side_effects"].([]any)
	if len(sideEffects) == 0 {
		t.Fatalf("expected side_effects, got none; response: %v", resp)
	}

	var foundUnblocked bool
	for _, se := range sideEffects {
		effect, ok := se.(map[string]any)
		if !ok {
			continue
		}
		if effect["type"] == "task_unblocked" && effect["entity_id"] == taskBID {
			foundUnblocked = true
			if effect["entity_type"] != "task" {
				t.Errorf("task_unblocked entity_type = %q, want \"task\"", effect["entity_type"])
			}
			if effect["to_status"] != "ready" {
				t.Errorf("task_unblocked to_status = %q, want \"ready\"", effect["to_status"])
			}
		}
	}
	if !foundUnblocked {
		t.Errorf("expected task_unblocked side effect for %s, got: %v", taskBID, sideEffects)
	}
}

// TestFinish_NoSideEffectsWhenNoDependents verifies that finish always returns
// side_effects: [] (spec §8.4) even when no cascades occur (no downstream tasks,
// no knowledge contributions).
func TestFinish_NoSideEffectsWhenNoDependents(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupFinishTest(t)
	taskID, taskSlug := setupFinishScenario(t, entitySvc, "no-deps")
	advanceToActive(t, entitySvc, taskID, taskSlug)

	resp := callFinishJSON(t, entitySvc, dispatchSvc, map[string]any{
		"task_id": taskID,
		"summary": "Done, no dependents",
	})

	// finish is a mutation — side_effects must always be present, even when empty
	// (spec §8.4: "The field is never omitted").
	sideEffects, exists := resp["side_effects"]
	if !exists {
		t.Fatal("side_effects absent from finish response — spec §8.4 requires it for mutations")
	}
	arr, _ := sideEffects.([]any)
	for _, se := range arr {
		effect, _ := se.(map[string]any)
		if effect["type"] == "task_unblocked" {
			t.Errorf("unexpected task_unblocked side effect: %v", effect)
		}
	}
}

// ─── Error cases ──────────────────────────────────────────────────────────────

// TestFinish_TaskNotFound verifies the error response when the task does not exist.
func TestFinish_TaskNotFound(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupFinishTest(t)

	text := callFinish(t, entitySvc, dispatchSvc, map[string]any{
		"task_id": "TASK-01ZZZZZZZZZZZZ",
		"summary": "Done",
	})

	var resp map[string]any
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("parse response: %v\nraw: %s", err, text)
	}
	if _, hasError := resp["error"]; !hasError {
		t.Errorf("expected error field in response, got: %v", resp)
	}
}

// TestFinish_MissingSummary verifies that omitting summary returns an error.
func TestFinish_MissingSummary(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupFinishTest(t)
	taskID, taskSlug := setupFinishScenario(t, entitySvc, "no-summary")
	advanceToActive(t, entitySvc, taskID, taskSlug)

	text := callFinish(t, entitySvc, dispatchSvc, map[string]any{
		"task_id": taskID,
	})

	var resp map[string]any
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("parse response: %v\nraw: %s", err, text)
	}
	if _, hasError := resp["error"]; !hasError {
		t.Errorf("expected error for missing summary, got: %v", resp)
	}
	errDetail, _ := resp["error"].(map[string]any)
	if msg, _ := errDetail["message"].(string); !strings.Contains(msg, "summary") {
		t.Errorf("error message should mention \"summary\", got: %q", msg)
	}
}

// TestFinish_MissingTaskID verifies that omitting task_id returns an error.
func TestFinish_MissingTaskID(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupFinishTest(t)

	text := callFinish(t, entitySvc, dispatchSvc, map[string]any{
		"summary": "Done",
	})

	var resp map[string]any
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("parse response: %v\nraw: %s", err, text)
	}
	if _, hasError := resp["error"]; !hasError {
		t.Errorf("expected error for missing task_id, got: %v", resp)
	}
}

// TestFinish_TaskInTerminalStatus verifies that completing an already-done task returns an error.
func TestFinish_TaskInTerminalStatus(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupFinishTest(t)
	taskID, taskSlug := setupFinishScenario(t, entitySvc, "terminal")
	advanceToActive(t, entitySvc, taskID, taskSlug)

	// Complete the task once.
	callFinishJSON(t, entitySvc, dispatchSvc, map[string]any{
		"task_id": taskID,
		"summary": "First completion",
	})

	// Attempt to complete again — should fail.
	text := callFinish(t, entitySvc, dispatchSvc, map[string]any{
		"task_id": taskID,
		"summary": "Second completion attempt",
	})

	var resp map[string]any
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("parse response: %v\nraw: %s", err, text)
	}
	if _, hasError := resp["error"]; !hasError {
		t.Errorf("expected error for terminal-status task, got: %v", resp)
	}
}

// TestFinish_TaskInQueuedStatus verifies that a queued task (not yet ready) is rejected.
func TestFinish_TaskInQueuedStatus(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupFinishTest(t)
	taskID, _ := setupFinishScenario(t, entitySvc, "queued")
	// Task is in queued status — do not advance.

	text := callFinish(t, entitySvc, dispatchSvc, map[string]any{
		"task_id": taskID,
		"summary": "Done",
	})

	var resp map[string]any
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("parse response: %v\nraw: %s", err, text)
	}
	if _, hasError := resp["error"]; !hasError {
		t.Errorf("expected error for queued task, got: %v", resp)
	}
	errDetail, _ := resp["error"].(map[string]any)
	msg, _ := errDetail["message"].(string)
	if !strings.Contains(msg, "queued") {
		t.Errorf("error should mention current status, got: %q", msg)
	}
}

// ─── Batch tests ──────────────────────────────────────────────────────────────

// TestFinish_BatchCompletion verifies batch mode: tasks array processes each item.
// Acceptance criterion: batch finish processes each task independently (best-effort).
// Acceptance criterion: batch finish returns per-item results with aggregate side effects.
func TestFinish_BatchCompletion(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupFinishTest(t)

	// Create two tasks in active status.
	taskAID, taskASlug := setupFinishScenario(t, entitySvc, "batch-a")
	taskBID, taskBSlug := setupFinishScenario(t, entitySvc, "batch-b")
	advanceToActive(t, entitySvc, taskAID, taskASlug)
	advanceToActive(t, entitySvc, taskBID, taskBSlug)

	text := callFinish(t, entitySvc, dispatchSvc, map[string]any{
		"tasks": []any{
			map[string]any{"task_id": taskAID, "summary": "Task A done"},
			map[string]any{"task_id": taskBID, "summary": "Task B done"},
		},
	})

	var resp map[string]any
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("parse batch response: %v\nraw: %s", err, text)
	}

	// Batch response must have "results" and "summary".
	results, ok := resp["results"].([]any)
	if !ok || len(results) != 2 {
		t.Fatalf("expected 2 results, got: %v", resp["results"])
	}
	summary, ok := resp["summary"].(map[string]any)
	if !ok {
		t.Fatalf("expected summary in batch response, got: %v", resp)
	}

	if total := summary["total"]; total != float64(2) {
		t.Errorf("summary.total = %v, want 2", total)
	}
	if succeeded := summary["succeeded"]; succeeded != float64(2) {
		t.Errorf("summary.succeeded = %v, want 2", succeeded)
	}
	if failed := summary["failed"]; failed != float64(0) {
		t.Errorf("summary.failed = %v, want 0", failed)
	}

	// Each item result must have status "ok" and a task payload.
	for i, r := range results {
		item := r.(map[string]any)
		if item["status"] != "ok" {
			t.Errorf("results[%d].status = %q, want \"ok\"; error: %v", i, item["status"], item["error"])
		}
		data, ok := item["data"].(map[string]any)
		if !ok {
			t.Errorf("results[%d].data missing or not a map: %v", i, item["data"])
			continue
		}
		taskData, ok := data["task"].(map[string]any)
		if !ok {
			t.Errorf("results[%d].data.task missing: %v", i, data)
			continue
		}
		if taskData["status"] != "done" {
			t.Errorf("results[%d].data.task.status = %q, want \"done\"", i, taskData["status"])
		}
	}
}

// TestFinish_BatchPartialFailure verifies that a failure on one item does not block others.
// Acceptance criterion: batch finish processes each task independently (best-effort).
func TestFinish_BatchPartialFailure(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupFinishTest(t)

	// Create one valid task and one invalid task ID.
	taskAID, taskASlug := setupFinishScenario(t, entitySvc, "bf-a")
	advanceToActive(t, entitySvc, taskAID, taskASlug)

	text := callFinish(t, entitySvc, dispatchSvc, map[string]any{
		"tasks": []any{
			map[string]any{"task_id": taskAID, "summary": "Valid completion"},
			map[string]any{"task_id": "TASK-01ZZZZNONEXISTENT", "summary": "Invalid"},
		},
	})

	var resp map[string]any
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("parse batch response: %v\nraw: %s", err, text)
	}

	summary := resp["summary"].(map[string]any)
	if summary["total"] != float64(2) {
		t.Errorf("total = %v, want 2", summary["total"])
	}
	if summary["succeeded"] != float64(1) {
		t.Errorf("succeeded = %v, want 1", summary["succeeded"])
	}
	if summary["failed"] != float64(1) {
		t.Errorf("failed = %v, want 1", summary["failed"])
	}

	results := resp["results"].([]any)

	// First item should succeed.
	first := results[0].(map[string]any)
	if first["status"] != "ok" {
		t.Errorf("results[0].status = %q, want \"ok\"", first["status"])
	}

	// Second item should fail.
	second := results[1].(map[string]any)
	if second["status"] != "error" {
		t.Errorf("results[1].status = %q, want \"error\"", second["status"])
	}
}

// TestFinish_BatchWithReadyTasks verifies lenient lifecycle in batch mode.
func TestFinish_BatchWithReadyTasks(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupFinishTest(t)

	taskAID, taskASlug := setupFinishScenario(t, entitySvc, "bready-a")
	taskBID, taskBSlug := setupFinishScenario(t, entitySvc, "bready-b")
	// Advance both to ready only — no explicit dispatch.
	advanceToReady(t, entitySvc, taskAID, taskASlug)
	advanceToReady(t, entitySvc, taskBID, taskBSlug)

	text := callFinish(t, entitySvc, dispatchSvc, map[string]any{
		"tasks": []any{
			map[string]any{"task_id": taskAID, "summary": "A done (from ready)"},
			map[string]any{"task_id": taskBID, "summary": "B done (from ready)"},
		},
	})

	var resp map[string]any
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("parse batch response: %v\nraw: %s", err, text)
	}

	summary := resp["summary"].(map[string]any)
	if summary["succeeded"] != float64(2) {
		t.Errorf("succeeded = %v, want 2", summary["succeeded"])
	}
}

// TestFinish_BatchWithKnowledge verifies that batch items can carry inline knowledge.
func TestFinish_BatchWithKnowledge(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupFinishTest(t)

	taskID, taskSlug := setupFinishScenario(t, entitySvc, "bke")
	advanceToActive(t, entitySvc, taskID, taskSlug)

	text := callFinish(t, entitySvc, dispatchSvc, map[string]any{
		"tasks": []any{
			map[string]any{
				"task_id": taskID,
				"summary": "Done with knowledge",
				"knowledge": []any{
					map[string]any{
						"topic":   "batch-ke-topic",
						"content": "Batch knowledge entry content",
						"scope":   "project",
					},
				},
			},
		},
	})

	var resp map[string]any
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("parse batch response: %v\nraw: %s", err, text)
	}

	results := resp["results"].([]any)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	first := results[0].(map[string]any)
	if first["status"] != "ok" {
		t.Errorf("result status = %q, want \"ok\"; error: %v", first["status"], first["error"])
	}

	data := first["data"].(map[string]any)
	ke := data["knowledge"].(map[string]any)
	if ke["total_accepted"] != float64(1) {
		t.Errorf("total_accepted = %v, want 1", ke["total_accepted"])
	}
}

// TestFinish_BatchAggregateSideEffects verifies that the top-level side_effects
// in a BatchResult is the union of all per-item side effects.
// Acceptance criterion: batch finish returns per-item results with aggregate side effects.
func TestFinish_BatchAggregateSideEffects(t *testing.T) {
	t.Parallel()

	entityRoot := t.TempDir()
	stateRoot := t.TempDir()
	entitySvc := service.NewEntityService(entityRoot)
	knowledgeSvc := service.NewKnowledgeService(stateRoot)
	dispatchSvc := service.NewDispatchService(entitySvc, knowledgeSvc)

	// Wire dependency unblocking hook.
	hook := service.NewDependencyUnblockingHook(entitySvc)
	entitySvc.SetStatusTransitionHook(hook)

	planID := createFinishTestPlan(t, entitySvc, "agg-se")
	featID := createFinishTestFeature(t, entitySvc, planID, "agg-feat")

	// Task A blocks task C; task B blocks task D.
	taskAID, taskASlug := createFinishTestTask(t, entitySvc, featID, "agg-a")
	taskBID, taskBSlug := createFinishTestTask(t, entitySvc, featID, "agg-b")
	taskCID, taskCSlug := createFinishTestTask(t, entitySvc, featID, "agg-c")
	taskDID, taskDSlug := createFinishTestTask(t, entitySvc, featID, "agg-d")

	setFinishDependsOn(t, entitySvc, taskCID, taskCSlug, []string{taskAID})
	setFinishDependsOn(t, entitySvc, taskDID, taskDSlug, []string{taskBID})

	advanceToActive(t, entitySvc, taskAID, taskASlug)
	advanceToActive(t, entitySvc, taskBID, taskBSlug)

	text := callFinish(t, entitySvc, dispatchSvc, map[string]any{
		"tasks": []any{
			map[string]any{"task_id": taskAID, "summary": "A done"},
			map[string]any{"task_id": taskBID, "summary": "B done"},
		},
	})

	var resp map[string]any
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("parse batch response: %v\nraw: %s", err, text)
	}

	// Top-level side_effects must contain both unblocked tasks.
	topSideEffects, _ := resp["side_effects"].([]any)
	unblockedIDs := map[string]bool{}
	for _, se := range topSideEffects {
		effect := se.(map[string]any)
		if effect["type"] == "task_unblocked" {
			unblockedIDs[effect["entity_id"].(string)] = true
		}
	}
	if !unblockedIDs[taskCID] {
		t.Errorf("expected task_unblocked for %s in aggregate side_effects", taskCID)
	}
	if !unblockedIDs[taskDID] {
		t.Errorf("expected task_unblocked for %s in aggregate side_effects", taskDID)
	}
}
