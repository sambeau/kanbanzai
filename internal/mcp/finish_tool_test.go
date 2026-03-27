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

// mockWorktreeHook is a StatusTransitionHook that signals worktree creation
// whenever an entity transitions to "active" status. Used in tests that verify
// the worktree_created side effect flows through the finish tool's lenient
// lifecycle path (ready → active → done).
type mockWorktreeHook struct {
	worktreeID string
	branch     string
	path       string
}

func (h *mockWorktreeHook) OnStatusTransition(_, entityID, _, _, toStatus string, _ map[string]any) *service.WorktreeResult {
	if toStatus == "active" {
		return &service.WorktreeResult{
			Created:    true,
			WorktreeID: h.worktreeID,
			EntityID:   entityID,
			Branch:     h.branch,
			Path:       h.path,
		}
	}
	return nil
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

// TestFinish_ReadyTask_WorktreeCreated verifies that the worktree_created side
// effect is reported when a ready task auto-transitions through active and the
// configured worktree hook fires during that transition (spec §30.2:
// "Worktree auto-creation on entity transition is reported as a side effect").
//
// This tests the code path in finishOne case "ready": that checks
// activeResult.WorktreeHookResult and pushes SideEffectWorktreeCreated.
func TestFinish_ReadyTask_WorktreeCreated(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupFinishTest(t)

	// Wire a mock hook that signals worktree creation on any → active transition.
	entitySvc.SetStatusTransitionHook(&mockWorktreeHook{
		worktreeID: "WT-01TEST",
		branch:     "feature/test-branch",
		path:       "/tmp/test-worktree",
	})

	taskID, taskSlug := setupFinishScenario(t, entitySvc, "wt-created")
	advanceToReady(t, entitySvc, taskID, taskSlug)

	resp := callFinishJSON(t, entitySvc, dispatchSvc, map[string]any{
		"task_id": taskID,
		"summary": "Completed via lenient lifecycle with worktree hook",
	})

	// Task must complete successfully despite the hook.
	taskData, ok := resp["task"].(map[string]any)
	if !ok {
		t.Fatalf("expected task in response, got: %v", resp)
	}
	if got := taskData["status"]; got != "done" {
		t.Errorf("task status = %q, want \"done\"", got)
	}

	// The worktree_created side effect must be present — the lenient lifecycle
	// fires the ready → active transition, which triggers the hook.
	sideEffects, _ := resp["side_effects"].([]any)
	var foundWT bool
	for _, se := range sideEffects {
		effect, ok := se.(map[string]any)
		if !ok {
			continue
		}
		if effect["type"] == "worktree_created" {
			foundWT = true
			if effect["entity_id"] != taskID {
				t.Errorf("worktree_created entity_id = %q, want %q", effect["entity_id"], taskID)
			}
			if effect["entity_type"] != "task" {
				t.Errorf("worktree_created entity_type = %q, want \"task\"", effect["entity_type"])
			}
		}
	}
	if !foundWT {
		t.Errorf("expected worktree_created side effect; got side_effects: %v", sideEffects)
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

// ─── Retrospective signal tests ───────────────────────────────────────────────

// TestFinish_RetroNoParam verifies that omitting the retrospective parameter
// produces no regression in the response shape (P5-1.1).
func TestFinish_RetroNoParam(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupFinishTest(t)
	taskID, taskSlug := setupFinishScenario(t, entitySvc, "retro-noparam")
	advanceToActive(t, entitySvc, taskID, taskSlug)

	resp := callFinishJSON(t, entitySvc, dispatchSvc, map[string]any{
		"task_id": taskID,
		"summary": "Done",
	})

	// retrospective key must be absent when no signals are submitted.
	if _, ok := resp["retrospective"]; ok {
		t.Error("expected no \"retrospective\" key in response when parameter is absent")
	}
	// task and knowledge must still be present (no regression).
	if _, ok := resp["task"]; !ok {
		t.Error("expected \"task\" key in response")
	}
	if _, ok := resp["knowledge"]; !ok {
		t.Error("expected \"knowledge\" key in response")
	}
}

// TestFinish_RetroAccepted verifies that valid signals are stored as knowledge
// entries and appear in the response and side_effects (P5-1.2, P5-1.14, P5-1.15).
func TestFinish_RetroAccepted(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupFinishTest(t)
	taskID, taskSlug := setupFinishScenario(t, entitySvc, "retro-accept")
	advanceToActive(t, entitySvc, taskID, taskSlug)

	resp := callFinishJSON(t, entitySvc, dispatchSvc, map[string]any{
		"task_id": taskID,
		"summary": "Implemented feature",
		"retrospective": []any{
			map[string]any{
				"category":    "spec-ambiguity",
				"observation": "Spec did not define error format",
				"severity":    "moderate",
				"suggestion":  "Add error format section",
			},
		},
	})

	// Task must complete.
	taskData, ok := resp["task"].(map[string]any)
	if !ok {
		t.Fatalf("expected task in response, got: %v", resp)
	}
	if got := taskData["status"]; got != "done" {
		t.Errorf("task status = %q, want \"done\"", got)
	}

	// retrospective section must be present.
	retro, ok := resp["retrospective"].(map[string]any)
	if !ok {
		t.Fatalf("expected retrospective section in response, got: %v", resp)
	}
	if got := retro["total_attempted"]; got != float64(1) {
		t.Errorf("total_attempted = %v, want 1", got)
	}
	if got := retro["total_accepted"]; got != float64(1) {
		t.Errorf("total_accepted = %v, want 1", got)
	}
	accepted, ok := retro["accepted"].([]any)
	if !ok || len(accepted) != 1 {
		t.Fatalf("expected 1 accepted retro signal, got: %v", retro["accepted"])
	}
	entry := accepted[0].(map[string]any)
	if got := entry["category"]; got != "spec-ambiguity" {
		t.Errorf("accepted signal category = %q, want \"spec-ambiguity\"", got)
	}
	if entryID, _ := entry["entry_id"].(string); entryID == "" {
		t.Error("accepted signal missing entry_id")
	}
	if topic, _ := entry["topic"].(string); topic == "" {
		t.Error("accepted signal missing topic")
	}

	// retrospective_signal_contributed must appear in side_effects.
	sideEffects, _ := resp["side_effects"].([]any)
	var foundRetro bool
	for _, se := range sideEffects {
		effect := se.(map[string]any)
		if effect["type"] == "retrospective_signal_contributed" {
			foundRetro = true
		}
	}
	if !foundRetro {
		t.Errorf("expected retrospective_signal_contributed side effect, got: %v", sideEffects)
	}
}

// TestFinish_RetroStoredAsKnowledgeEntry verifies the storage convention:
// tier=3, scope=project, tags=["retrospective", category] (P5-1.3, P5-1.6).
func TestFinish_RetroStoredAsKnowledgeEntry(t *testing.T) {
	t.Parallel()
	entityRoot := t.TempDir()
	stateRoot := t.TempDir()
	entitySvc := service.NewEntityService(entityRoot)
	knowledgeSvc := service.NewKnowledgeService(stateRoot)
	dispatchSvc := service.NewDispatchService(entitySvc, knowledgeSvc)

	planID := createFinishTestPlan(t, entitySvc, "retro-ke-plan")
	featID := createFinishTestFeature(t, entitySvc, planID, "retro-ke-feat")
	taskID, taskSlug := createFinishTestTask(t, entitySvc, featID, "retro-ke-task")
	advanceToActive(t, entitySvc, taskID, taskSlug)

	callFinish(t, entitySvc, dispatchSvc, map[string]any{
		"task_id": taskID,
		"summary": "Done",
		"retrospective": []any{
			map[string]any{
				"category":    "context-gap",
				"observation": "Error convention was missing from context packet",
				"severity":    "minor",
			},
		},
	})

	// Retrieve stored entries tagged "retrospective".
	entries, err := knowledgeSvc.List(service.KnowledgeFilters{Tags: []string{"retrospective"}})
	if err != nil {
		t.Fatalf("list knowledge entries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 retrospective entry, got %d", len(entries))
	}
	rec := entries[0]
	fields := rec.Fields

	if tier, _ := fields["tier"].(int); tier != 3 {
		// tier may be stored as float64 after YAML round-trip
		if tierF, _ := fields["tier"].(float64); int(tierF) != 3 {
			t.Errorf("tier = %v, want 3", fields["tier"])
		}
	}
	if scope, _ := fields["scope"].(string); scope != "project" {
		t.Errorf("scope = %q, want \"project\"", scope)
	}
	if lf, _ := fields["learned_from"].(string); lf != taskID {
		t.Errorf("learned_from = %q, want %q", lf, taskID)
	}
	// Tags must include both "retrospective" and the category.
	var tags []string
	switch v := fields["tags"].(type) {
	case []string:
		tags = v
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				tags = append(tags, s)
			}
		}
	}
	hasRetroTag, hasCatTag := false, false
	for _, tag := range tags {
		if tag == "retrospective" {
			hasRetroTag = true
		}
		if tag == "context-gap" {
			hasCatTag = true
		}
	}
	if !hasRetroTag {
		t.Error("stored entry missing tag \"retrospective\"")
	}
	if !hasCatTag {
		t.Error("stored entry missing tag \"context-gap\"")
	}
}

// TestFinish_RetroTopicNaming verifies topic sequence for multiple signals from
// the same task (P5-1.4).
func TestFinish_RetroTopicNaming(t *testing.T) {
	t.Parallel()
	entityRoot := t.TempDir()
	stateRoot := t.TempDir()
	entitySvc := service.NewEntityService(entityRoot)
	knowledgeSvc := service.NewKnowledgeService(stateRoot)
	dispatchSvc := service.NewDispatchService(entitySvc, knowledgeSvc)

	planID := createFinishTestPlan(t, entitySvc, "retro-topic-plan")
	featID := createFinishTestFeature(t, entitySvc, planID, "retro-topic-feat")
	taskID, taskSlug := createFinishTestTask(t, entitySvc, featID, "retro-topic-task")
	advanceToActive(t, entitySvc, taskID, taskSlug)

	resp := callFinishJSON(t, entitySvc, dispatchSvc, map[string]any{
		"task_id": taskID,
		"summary": "Done",
		"retrospective": []any{
			map[string]any{
				"category":    "spec-ambiguity",
				"observation": "First observation",
				"severity":    "minor",
			},
			map[string]any{
				"category":    "context-gap",
				"observation": "Second observation",
				"severity":    "moderate",
			},
			map[string]any{
				"category":    "worked-well",
				"observation": "Third observation",
				"severity":    "minor",
			},
		},
	})

	retro := resp["retrospective"].(map[string]any)
	if got := retro["total_accepted"]; got != float64(3) {
		t.Fatalf("total_accepted = %v, want 3", got)
	}
	accepted := retro["accepted"].([]any)

	topics := make([]string, len(accepted))
	for i, a := range accepted {
		topics[i] = a.(map[string]any)["topic"].(string)
	}

	// First topic must be "retro-{taskID}".
	if want := "retro-" + taskID; topics[0] != want {
		t.Errorf("first topic = %q, want %q", topics[0], want)
	}
	// Second must be "retro-{taskID}-2".
	if want := "retro-" + taskID + "-2"; topics[1] != want {
		t.Errorf("second topic = %q, want %q", topics[1], want)
	}
	// Third must be "retro-{taskID}-3".
	if want := "retro-" + taskID + "-3"; topics[2] != want {
		t.Errorf("third topic = %q, want %q", topics[2], want)
	}
	// All topics must be unique.
	seen := make(map[string]bool)
	for _, topic := range topics {
		if seen[topic] {
			t.Errorf("duplicate topic: %q", topic)
		}
		seen[topic] = true
	}
}

// TestFinish_RetroInvalidCategoryNonBlocking verifies that an unknown category
// rejects only that signal and does not block task completion or other signals
// (P5-1.8, plus P5-1.2 non-blocking).
func TestFinish_RetroInvalidCategoryNonBlocking(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupFinishTest(t)
	taskID, taskSlug := setupFinishScenario(t, entitySvc, "retro-bad-cat")
	advanceToActive(t, entitySvc, taskID, taskSlug)

	resp := callFinishJSON(t, entitySvc, dispatchSvc, map[string]any{
		"task_id": taskID,
		"summary": "Done",
		"retrospective": []any{
			map[string]any{
				"category":    "not-a-real-category",
				"observation": "Something happened",
				"severity":    "minor",
			},
			map[string]any{
				"category":    "tool-gap",
				"observation": "No tool for X",
				"severity":    "moderate",
			},
		},
	})

	// Task must complete.
	taskData := resp["task"].(map[string]any)
	if got := taskData["status"]; got != "done" {
		t.Errorf("task status = %q, want \"done\"", got)
	}

	retro := resp["retrospective"].(map[string]any)
	if got := retro["total_attempted"]; got != float64(2) {
		t.Errorf("total_attempted = %v, want 2", got)
	}
	if got := retro["total_accepted"]; got != float64(1) {
		t.Errorf("total_accepted = %v, want 1 (valid signal accepted, bad one rejected)", got)
	}

	rejected := retro["rejected"].([]any)
	if len(rejected) != 1 {
		t.Fatalf("expected 1 rejected signal, got %d", len(rejected))
	}
	r := rejected[0].(map[string]any)
	if got := r["category"]; got != "not-a-real-category" {
		t.Errorf("rejected category = %q, want \"not-a-real-category\"", got)
	}
	if reason, _ := r["reason"].(string); reason == "" {
		t.Error("rejected signal missing reason")
	}
}

// TestFinish_RetroMissingFieldsNonBlocking verifies per-field validation
// rejects only the bad signal without blocking completion (P5-1.9–P5-1.12).
func TestFinish_RetroMissingFieldsNonBlocking(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		signal map[string]any
		errMsg string
	}{
		{
			name:   "missing observation",
			signal: map[string]any{"category": "tool-gap", "severity": "minor"},
			errMsg: "observation is required",
		},
		{
			name:   "missing severity",
			signal: map[string]any{"category": "tool-gap", "observation": "No tool"},
			errMsg: "severity is required",
		},
		{
			name:   "unknown severity",
			signal: map[string]any{"category": "tool-gap", "observation": "No tool", "severity": "extreme"},
			errMsg: "unknown severity",
		},
		{
			name:   "missing category",
			signal: map[string]any{"observation": "Something", "severity": "minor"},
			errMsg: "category is required",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			entitySvc, dispatchSvc := setupFinishTest(t)
			taskID, taskSlug := setupFinishScenario(t, entitySvc, "retro-val-"+tc.name[:4])
			advanceToActive(t, entitySvc, taskID, taskSlug)

			resp := callFinishJSON(t, entitySvc, dispatchSvc, map[string]any{
				"task_id":       taskID,
				"summary":       "Done",
				"retrospective": []any{tc.signal},
			})

			// Task must complete despite invalid signal.
			taskData := resp["task"].(map[string]any)
			if got := taskData["status"]; got != "done" {
				t.Errorf("task status = %q, want \"done\"", got)
			}
			retro := resp["retrospective"].(map[string]any)
			if got := retro["total_accepted"]; got != float64(0) {
				t.Errorf("total_accepted = %v, want 0", got)
			}
			rejected := retro["rejected"].([]any)
			if len(rejected) == 0 {
				t.Fatal("expected rejected entry, got none")
			}
			reason, _ := rejected[0].(map[string]any)["reason"].(string)
			if !strings.Contains(reason, tc.errMsg) {
				t.Errorf("reason = %q, want to contain %q", reason, tc.errMsg)
			}
		})
	}
}

// TestFinish_RetroOptionalSuggestion verifies that suggestion-less signals are
// accepted and stored without a Suggestion clause (P5-1.13).
func TestFinish_RetroOptionalSuggestion(t *testing.T) {
	t.Parallel()
	entityRoot := t.TempDir()
	stateRoot := t.TempDir()
	entitySvc := service.NewEntityService(entityRoot)
	knowledgeSvc := service.NewKnowledgeService(stateRoot)
	dispatchSvc := service.NewDispatchService(entitySvc, knowledgeSvc)

	planID := createFinishTestPlan(t, entitySvc, "retro-nosugg-plan")
	featID := createFinishTestFeature(t, entitySvc, planID, "retro-nosugg-feat")
	taskID, taskSlug := createFinishTestTask(t, entitySvc, featID, "retro-nosugg-task")
	advanceToActive(t, entitySvc, taskID, taskSlug)

	resp := callFinishJSON(t, entitySvc, dispatchSvc, map[string]any{
		"task_id": taskID,
		"summary": "Done",
		"retrospective": []any{
			map[string]any{
				"category":    "worked-well",
				"observation": "Vertical slice decomposition worked great",
				"severity":    "minor",
				// no suggestion
			},
		},
	})

	retro := resp["retrospective"].(map[string]any)
	if got := retro["total_accepted"]; got != float64(1) {
		t.Fatalf("total_accepted = %v, want 1", got)
	}

	// The stored content must not contain "Suggestion:".
	entries, err := knowledgeSvc.List(service.KnowledgeFilters{Tags: []string{"retrospective"}})
	if err != nil {
		t.Fatalf("list knowledge entries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	content, _ := entries[0].Fields["content"].(string)
	if strings.Contains(content, "Suggestion:") {
		t.Errorf("content should not contain \"Suggestion:\" when suggestion is absent, got: %q", content)
	}
}

// TestFinish_RetroOnlyStoredAfterTransition verifies that if task completion
// fails (task not in completable status), no signals are stored (P5-1.7).
func TestFinish_RetroOnlyStoredAfterTransition(t *testing.T) {
	t.Parallel()
	entityRoot := t.TempDir()
	stateRoot := t.TempDir()
	entitySvc := service.NewEntityService(entityRoot)
	knowledgeSvc := service.NewKnowledgeService(stateRoot)
	dispatchSvc := service.NewDispatchService(entitySvc, knowledgeSvc)

	planID := createFinishTestPlan(t, entitySvc, "retro-notrans-plan")
	featID := createFinishTestFeature(t, entitySvc, planID, "retro-notrans-feat")
	taskID, taskSlug := createFinishTestTask(t, entitySvc, featID, "retro-notrans-task")
	// Leave task in queued status — not completable.
	_ = taskSlug

	tool := finishTool(entitySvc, dispatchSvc)
	req := makeRequest(map[string]any{
		"task_id": taskID,
		"summary": "Attempting completion of queued task",
		"retrospective": []any{
			map[string]any{
				"category":    "spec-ambiguity",
				"observation": "Should not be stored",
				"severity":    "minor",
			},
		},
	})
	// Expect an error response (task is in queued status).
	result, _ := tool.Handler(context.Background(), req)
	text := extractText(t, result)
	if !strings.Contains(text, "queued") && !strings.Contains(text, "ready") && !strings.Contains(text, "active") {
		t.Logf("finish response: %s", text)
	}

	// No knowledge entries should have been stored.
	entries, err := knowledgeSvc.List(service.KnowledgeFilters{Tags: []string{"retrospective"}})
	if err != nil {
		t.Fatalf("list knowledge entries: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 retrospective entries after failed completion, got %d", len(entries))
	}
}

// TestFinish_BatchWithRetro verifies that batch mode supports per-task
// retrospective arrays (P5-1.16).
func TestFinish_BatchWithRetro(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupFinishTest(t)

	planID := createFinishTestPlan(t, entitySvc, "retro-batch-plan")
	featID := createFinishTestFeature(t, entitySvc, planID, "retro-batch-feat")
	taskAID, taskASlug := createFinishTestTask(t, entitySvc, featID, "retro-batch-a")
	taskBID, taskBSlug := createFinishTestTask(t, entitySvc, featID, "retro-batch-b")
	advanceToActive(t, entitySvc, taskAID, taskASlug)
	advanceToActive(t, entitySvc, taskBID, taskBSlug)

	text := callFinish(t, entitySvc, dispatchSvc, map[string]any{
		"tasks": []any{
			map[string]any{
				"task_id": taskAID,
				"summary": "A done",
				"retrospective": []any{
					map[string]any{
						"category":    "tool-friction",
						"observation": "Tool A was awkward",
						"severity":    "minor",
					},
				},
			},
			map[string]any{
				"task_id": taskBID,
				"summary": "B done",
				"retrospective": []any{
					map[string]any{
						"category":    "worked-well",
						"observation": "Tool B was great",
						"severity":    "minor",
					},
				},
			},
		},
	})

	var resp map[string]any
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("parse batch response: %v\nraw: %s", err, text)
	}

	results, _ := resp["results"].([]any)
	if len(results) != 2 {
		t.Fatalf("expected 2 batch results, got %d", len(results))
	}

	for i, r := range results {
		item := r.(map[string]any)
		data, ok := item["data"].(map[string]any)
		if !ok {
			t.Errorf("result[%d]: expected data in item, got: %v", i, item)
			continue
		}
		retro, ok := data["retrospective"].(map[string]any)
		if !ok {
			t.Errorf("result[%d]: expected retrospective section in data, got: %v", i, data)
			continue
		}
		if got := retro["total_accepted"]; got != float64(1) {
			t.Errorf("result[%d]: total_accepted = %v, want 1", i, got)
		}
	}
}

// TestFinish_BatchRetroIsolation verifies that signal failures in one batch
// item do not affect signal processing in other items (P5-1.17).
func TestFinish_BatchRetroIsolation(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupFinishTest(t)

	planID := createFinishTestPlan(t, entitySvc, "retro-iso-plan")
	featID := createFinishTestFeature(t, entitySvc, planID, "retro-iso-feat")
	taskAID, taskASlug := createFinishTestTask(t, entitySvc, featID, "retro-iso-a")
	taskBID, taskBSlug := createFinishTestTask(t, entitySvc, featID, "retro-iso-b")
	advanceToActive(t, entitySvc, taskAID, taskASlug)
	advanceToActive(t, entitySvc, taskBID, taskBSlug)

	text := callFinish(t, entitySvc, dispatchSvc, map[string]any{
		"tasks": []any{
			map[string]any{
				"task_id": taskAID,
				"summary": "A done",
				"retrospective": []any{
					map[string]any{
						"category":    "completely-invalid",
						"observation": "Bad signal",
						"severity":    "minor",
					},
				},
			},
			map[string]any{
				"task_id": taskBID,
				"summary": "B done",
				"retrospective": []any{
					map[string]any{
						"category":    "tool-gap",
						"observation": "Valid signal on task B",
						"severity":    "moderate",
					},
				},
			},
		},
	})

	var resp map[string]any
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("parse batch response: %v\nraw: %s", err, text)
	}

	results, _ := resp["results"].([]any)
	if len(results) != 2 {
		t.Fatalf("expected 2 batch results, got %d", len(results))
	}

	// Task A: signal rejected, but task itself completed.
	itemA := results[0].(map[string]any)
	dataA := itemA["data"].(map[string]any)
	taskA := dataA["task"].(map[string]any)
	if got := taskA["status"]; got != "done" {
		t.Errorf("task A status = %q, want \"done\"", got)
	}
	retroA := dataA["retrospective"].(map[string]any)
	if got := retroA["total_accepted"]; got != float64(0) {
		t.Errorf("task A: total_accepted = %v, want 0", got)
	}

	// Task B: signal accepted despite A's rejection.
	itemB := results[1].(map[string]any)
	dataB := itemB["data"].(map[string]any)
	retroB := dataB["retrospective"].(map[string]any)
	if got := retroB["total_accepted"]; got != float64(1) {
		t.Errorf("task B: total_accepted = %v, want 1 (should not be affected by task A's bad signal)", got)
	}
}
