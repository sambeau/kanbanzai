package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/sambeau/kanbanzai/internal/service"
)

func callDevelopToolJSON(t *testing.T, entitySvc *service.EntityService, conflictSvc *service.ConflictService, args map[string]any) map[string]any {
	t.Helper()
	tools := DevelopTool(entitySvc, conflictSvc)
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	req := makeRequest(args)
	result, err := tools[0].Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("develop handler error: %v", err)
	}
	text := extractText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse develop result: %v\nraw: %s", err, text)
	}
	return parsed
}

func advanceFeature(t *testing.T, entitySvc *service.EntityService, docSvc *service.DocumentService, repoRoot, featID, target string) {
	t.Helper()
	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action":  "transition",
		"id":      featID,
		"status":  target,
		"advance": true,
	})
	status, _ := result["status"].(string)
	if status != target {
		t.Logf("advanceFeature %s stopped at %q (target %s): reason=%v", featID, status, target, result["stopped_reason"])
	}
}

func setupFeatureDocs(t *testing.T, docSvc *service.DocumentService, repoRoot, featID, slug string) {
	t.Helper()
	setupApprovedDoc(t, docSvc, repoRoot, "work/"+slug+"/"+slug+"-design.md", "design", featID)
	setupApprovedDoc(t, docSvc, repoRoot, "work/"+slug+"/"+slug+"-spec.md", "specification", featID)
	setupApprovedDoc(t, docSvc, repoRoot, "work/"+slug+"/"+slug+"-plan.md", "dev-plan", featID)
}

func setTaskDependsOn(t *testing.T, entitySvc *service.EntityService, taskID string, deps []string) {
	t.Helper()
	_, err := entitySvc.UpdateEntity(service.UpdateEntityInput{
		Type: "task",
		ID:   taskID,
		ListFields: map[string][]string{
			"depends_on": deps,
		},
	})
	if err != nil {
		t.Fatalf("set depends_on: %v", err)
	}
}

func transitionTask(t *testing.T, entitySvc *service.EntityService, taskID, status string) {
	t.Helper()
	callEntityToolJSON(t, entitySvc, map[string]any{
		"action": "transition",
		"id":     taskID,
		"status": status,
	})
}

func TestCloseOut_BlocksWithoutReviewReport(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createEntityTestPlan(t, entitySvc, "co-norpt")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-co-norpt")

	setupFeatureDocs(t, docSvc, repoRoot, featID, "co-norpt")

	// Complete a task: queued -> ready -> active -> done
	taskID, _ := createEntityTestTask(t, entitySvc, featID, "task-co-norpt")
	transitionTask(t, entitySvc, taskID, "ready")
	transitionTask(t, entitySvc, taskID, "active")
	transitionTask(t, entitySvc, taskID, "done")

	advanceFeature(t, entitySvc, docSvc, repoRoot, featID, "reviewing")

	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action": "close-out",
		"id":     featID,
	})

	stoppedAt, _ := result["stopped_at"].(string)
	if stoppedAt != "reviewing" {
		t.Logf("result: %+v", result)
		t.Errorf("stopped_at = %q, want reviewing", stoppedAt)
	}

	na, hasNA := result["next_action"]
	if !hasNA {
		t.Fatal("expected next_action field when blocked by missing review report")
	}
	naMap, ok := na.(map[string]any)
	if !ok {
		t.Fatalf("next_action is not a map: %T", na)
	}
	if naMap["tool"] != "doc" {
		t.Errorf("next_action.tool = %v, want doc", naMap["tool"])
	}
	if naMap["action"] != "register" {
		t.Errorf("next_action.action = %v, want register", naMap["action"])
	}
}

func TestCloseOut_RejectsNonReviewingFeature(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createEntityTestPlan(t, entitySvc, "co-nonr")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-co-nonr")

	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action": "close-out",
		"id":     featID,
	})

	errStr, _ := result["error"].(string)
	if !strings.Contains(errStr, "not reviewing") {
		t.Errorf("expected error about not reviewing, got: %q", errStr)
	}
}

func TestCloseOut_SucceedsWithReviewReport(t *testing.T) {
	t.Parallel()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	planID := createEntityTestPlan(t, entitySvc, "co-ok")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-co-ok")

	setupFeatureDocs(t, docSvc, repoRoot, featID, "co-ok")

	taskID, _ := createEntityTestTask(t, entitySvc, featID, "task-co-ok")
	transitionTask(t, entitySvc, taskID, "ready")
	transitionTask(t, entitySvc, taskID, "active")
	transitionTask(t, entitySvc, taskID, "done")

	advanceFeature(t, entitySvc, docSvc, repoRoot, featID, "reviewing")

	setupApprovedDoc(t, docSvc, repoRoot, "work/reports/co-ok-review.md", "report", featID)

	result := callEntityToolWithDocSvcJSON(t, entitySvc, docSvc, map[string]any{
		"action": "close-out",
		"id":     featID,
	})

	status, _ := result["status"].(string)
	if status != "done" {
		t.Logf("result: %+v", result)
		t.Errorf("status = %q, want done", status)
	}
}

func TestDispatch_FiltersUnsatisfiedDependencies(t *testing.T) {
	t.Parallel()
	entitySvc := setupEntityToolTest(t)
	repoRoot := t.TempDir()

	planID := createEntityTestPlan(t, entitySvc, "disp-dep")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-disp-dep")

	taskA, _ := createEntityTestTask(t, entitySvc, featID, "task-disp-a")
	taskB, _ := createEntityTestTask(t, entitySvc, featID, "task-disp-b")
	setTaskDependsOn(t, entitySvc, taskB, []string{taskA})

	docSvc := service.NewDocumentService(t.TempDir(), repoRoot)
	setupFeatureDocs(t, docSvc, repoRoot, featID, "disp-dep")
	advanceFeature(t, entitySvc, docSvc, repoRoot, featID, "developing")

	transitionTask(t, entitySvc, taskA, "ready")
	transitionTask(t, entitySvc, taskB, "ready")

	result := callDevelopToolJSON(t, entitySvc, nil, map[string]any{
		"action":     "dispatch",
		"feature_id": featID,
	})

	dispatched, _ := result["dispatched"].([]any)
	if len(dispatched) != 1 {
		t.Fatalf("expected 1 dispatched task, got %d: %v", len(dispatched), dispatched)
	}
	d := dispatched[0].(map[string]any)
	if d["task_id"] != taskA {
		t.Errorf("dispatched task_id = %v, want %s", d["task_id"], taskA)
	}

	blocked, _ := result["blocked"].([]any)
	foundB := false
	for _, b := range blocked {
		bm := b.(map[string]any)
		if bm["task_id"] == taskB {
			foundB = true
			if reason, _ := bm["reason"].(string); !strings.Contains(reason, "depends_on not satisfied") {
				t.Errorf("blocked task B reason = %q, want 'depends_on not satisfied'", reason)
			}
		}
	}
	if !foundB {
		t.Errorf("task B should be in blocked list, got: %v", blocked)
	}
}

func TestDispatch_HandlesNilConflictSvc(t *testing.T) {
	t.Parallel()
	entitySvc := setupEntityToolTest(t)
	repoRoot := t.TempDir()

	planID := createEntityTestPlan(t, entitySvc, "disp-nil")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-disp-nil")

	taskA, _ := createEntityTestTask(t, entitySvc, featID, "task-disp-nil-a")
	taskB, _ := createEntityTestTask(t, entitySvc, featID, "task-disp-nil-b")

	docSvc := service.NewDocumentService(t.TempDir(), repoRoot)
	setupFeatureDocs(t, docSvc, repoRoot, featID, "disp-nil")
	advanceFeature(t, entitySvc, docSvc, repoRoot, featID, "developing")

	transitionTask(t, entitySvc, taskA, "ready")
	transitionTask(t, entitySvc, taskB, "ready")

	result := callDevelopToolJSON(t, entitySvc, nil, map[string]any{
		"action":     "dispatch",
		"feature_id": featID,
	})

	dispatched, _ := result["dispatched"].([]any)
	if len(dispatched) != 2 {
		t.Errorf("expected 2 dispatched tasks (nil conflictSvc), got %d", len(dispatched))
	}

	for _, d := range dispatched {
		dm := d.(map[string]any)
		if prompt, _ := dm["handoff_prompt"].(string); prompt == "" {
			t.Errorf("task %v missing handoff_prompt", dm["task_id"])
		}
		if _, hasHint := dm["handoff_hint"]; hasHint {
			t.Error("dispatched task has handoff_hint, should have handoff_prompt")
		}
	}
}

func TestDispatch_EmptyQueueWhenNoReadyTasks(t *testing.T) {
	t.Parallel()
	entitySvc := setupEntityToolTest(t)
	repoRoot := t.TempDir()

	planID := createEntityTestPlan(t, entitySvc, "disp-empty")
	featID := createEntityTestFeature(t, entitySvc, planID, "feat-disp-empty")

	createEntityTestTask(t, entitySvc, featID, "task-disp-empty-a")
	createEntityTestTask(t, entitySvc, featID, "task-disp-empty-b")

	docSvc := service.NewDocumentService(t.TempDir(), repoRoot)
	setupFeatureDocs(t, docSvc, repoRoot, featID, "disp-empty")
	advanceFeature(t, entitySvc, docSvc, repoRoot, featID, "developing")

	result := callDevelopToolJSON(t, entitySvc, nil, map[string]any{
		"action":     "dispatch",
		"feature_id": featID,
	})

	emptyQueue, _ := result["empty_queue"].(bool)
	if !emptyQueue {
		t.Error("expected empty_queue=true when no ready tasks")
	}

	dispatched, _ := result["dispatched"].([]any)
	if len(dispatched) != 0 {
		t.Errorf("expected 0 dispatched, got %d", len(dispatched))
	}
}
