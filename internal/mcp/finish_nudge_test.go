package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"kanbanzai/internal/service"
)

// ─── Nudge test helpers ───────────────────────────────────────────────────────

// setupNudgeScenario creates a plan → feature → task chain and returns the
// feature ID, task ID, and task slug. Unlike setupFinishScenario, it exposes
// the feature ID for nudge tests that need to check feature-level completion.
func setupNudgeScenario(t *testing.T, entitySvc *service.EntityService, suffix string) (featID, taskID, taskSlug string) {
	t.Helper()
	planID := createFinishTestPlan(t, entitySvc, "nudge-plan-"+suffix)
	featID = createFinishTestFeature(t, entitySvc, planID, "nudge-feat-"+suffix)
	taskID, taskSlug = createFinishTestTask(t, entitySvc, featID, "nudge-task-"+suffix)
	return featID, taskID, taskSlug
}

// callFinishNudge is a convenience wrapper that completes a task and returns
// the parsed JSON response.
func callFinishNudge(t *testing.T, entitySvc *service.EntityService, dispatchSvc *service.DispatchService, args map[string]any) map[string]any {
	t.Helper()
	tool := finishTool(entitySvc, dispatchSvc)
	req := makeRequest(args)
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("finish handler error: %v", err)
	}
	text := extractText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse finish result: %v\nraw: %s", err, text)
	}
	return parsed
}

// ─── Nudge 1 tests ────────────────────────────────────────────────────────────

// TestNudge1_FiredOnFeatureCompletionWithNoRetro verifies that nudge1
// (nudgeNoRetroSignals) is present when a feature's only task is completed
// with no retrospective signals recorded for any task.
func TestNudge1_FiredOnFeatureCompletionWithNoRetro(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupFinishTest(t)
	_, taskID, taskSlug := setupNudgeScenario(t, entitySvc, "n1-fired")
	advanceToActive(t, entitySvc, taskID, taskSlug)

	resp := callFinishNudge(t, entitySvc, dispatchSvc, map[string]any{
		"task_id": taskID,
		"summary": "Implemented the feature",
	})

	nudge, ok := resp["nudge"].(string)
	if !ok || nudge == "" {
		t.Fatalf("expected nudge1 in response, got: %v", resp["nudge"])
	}
	if nudge != nudgeNoRetroSignals {
		t.Errorf("nudge = %q, want nudgeNoRetroSignals", nudge)
	}
}

// TestNudge1_SuppressedWhenRetroSignalsExist verifies that nudge1 is absent
// when the completed task has a retrospective signal recorded.
func TestNudge1_SuppressedWhenRetroSignalsExist(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupFinishTest(t)
	_, taskID, taskSlug := setupNudgeScenario(t, entitySvc, "n1-retro")
	advanceToActive(t, entitySvc, taskID, taskSlug)

	resp := callFinishNudge(t, entitySvc, dispatchSvc, map[string]any{
		"task_id": taskID,
		"summary": "Implemented with retro",
		"retrospective": []any{
			map[string]any{
				"category":    "worked-well",
				"observation": "Pair programming helped",
				"severity":    "minor",
			},
		},
	})

	if nudge, exists := resp["nudge"]; exists && nudge != nil {
		t.Errorf("expected no nudge when retro signals exist, got nudge = %v", nudge)
	}
}

// TestNudge1_SuppressedWhenFeatureNotComplete verifies that nudge1 is absent
// when the feature still has non-terminal tasks after this completion.
func TestNudge1_SuppressedWhenFeatureNotComplete(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupFinishTest(t)
	planID := createFinishTestPlan(t, entitySvc, "nudge-plan-n1-incomplete")
	featID := createFinishTestFeature(t, entitySvc, planID, "nudge-feat-n1-incomplete")

	// Two tasks in the feature: complete task1 only.
	taskID1, taskSlug1 := createFinishTestTask(t, entitySvc, featID, "n1-incomplete-t1")
	_, _ = createFinishTestTask(t, entitySvc, featID, "n1-incomplete-t2") // stays queued

	advanceToActive(t, entitySvc, taskID1, taskSlug1)

	resp := callFinishNudge(t, entitySvc, dispatchSvc, map[string]any{
		"task_id": taskID1,
		"summary": "Task 1 done",
	})

	if nudge, exists := resp["nudge"]; exists && nudge == nudgeNoRetroSignals {
		t.Errorf("expected nudge1 absent when feature not complete, got nudge = %v", nudge)
	}
}

// ─── Nudge 2 tests ────────────────────────────────────────────────────────────

// TestNudge2_FiredWhenSummaryPresentNoKnowledgeNoRetro verifies that nudge2
// (nudgeNoKnowledge) is present when a task is completed with a summary but
// no knowledge entries or retrospective signals. Uses a 2-task feature so
// nudge1 does not fire.
func TestNudge2_FiredWhenSummaryPresentNoKnowledgeNoRetro(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupFinishTest(t)
	planID := createFinishTestPlan(t, entitySvc, "nudge-plan-n2-fired")
	featID := createFinishTestFeature(t, entitySvc, planID, "nudge-feat-n2-fired")

	// Two tasks so the feature stays incomplete after task1 → nudge1 won't fire.
	taskID1, taskSlug1 := createFinishTestTask(t, entitySvc, featID, "n2-fired-t1")
	_, _ = createFinishTestTask(t, entitySvc, featID, "n2-fired-t2") // stays queued

	advanceToActive(t, entitySvc, taskID1, taskSlug1)

	resp := callFinishNudge(t, entitySvc, dispatchSvc, map[string]any{
		"task_id": taskID1,
		"summary": "Task done with no knowledge",
	})

	nudge, ok := resp["nudge"].(string)
	if !ok || nudge == "" {
		t.Fatalf("expected nudge2 in response, got: %v", resp["nudge"])
	}
	if nudge != nudgeNoKnowledge {
		t.Errorf("nudge = %q, want nudgeNoKnowledge", nudge)
	}
}

// TestNudge2_SuppressedWhenKnowledgeProvided verifies that nudge2 is absent
// when the task completion includes knowledge entries.
func TestNudge2_SuppressedWhenKnowledgeProvided(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupFinishTest(t)
	planID := createFinishTestPlan(t, entitySvc, "nudge-plan-n2-ke")
	featID := createFinishTestFeature(t, entitySvc, planID, "nudge-feat-n2-ke")

	taskID1, taskSlug1 := createFinishTestTask(t, entitySvc, featID, "n2-ke-t1")
	_, _ = createFinishTestTask(t, entitySvc, featID, "n2-ke-t2")

	advanceToActive(t, entitySvc, taskID1, taskSlug1)

	resp := callFinishNudge(t, entitySvc, dispatchSvc, map[string]any{
		"task_id": taskID1,
		"summary": "Task done with knowledge",
		"knowledge": []any{
			map[string]any{
				"topic":   "test-topic",
				"content": "Some useful fact",
				"scope":   "project",
			},
		},
	})

	if nudge, exists := resp["nudge"]; exists && nudge != nil {
		t.Errorf("expected no nudge when knowledge provided, got nudge = %v", nudge)
	}
}

// TestNudge2_SuppressedWhenRetroProvided verifies that nudge2 is absent when
// the task completion includes retrospective signals.
func TestNudge2_SuppressedWhenRetroProvided(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupFinishTest(t)
	planID := createFinishTestPlan(t, entitySvc, "nudge-plan-n2-retro")
	featID := createFinishTestFeature(t, entitySvc, planID, "nudge-feat-n2-retro")

	taskID1, taskSlug1 := createFinishTestTask(t, entitySvc, featID, "n2-retro-t1")
	_, _ = createFinishTestTask(t, entitySvc, featID, "n2-retro-t2")

	advanceToActive(t, entitySvc, taskID1, taskSlug1)

	resp := callFinishNudge(t, entitySvc, dispatchSvc, map[string]any{
		"task_id": taskID1,
		"summary": "Task done with retro",
		"retrospective": []any{
			map[string]any{
				"category":    "tool-gap",
				"observation": "Missing tool X",
				"severity":    "moderate",
			},
		},
	})

	if nudge, exists := resp["nudge"]; exists && nudge != nil {
		t.Errorf("expected no nudge when retro provided, got nudge = %v", nudge)
	}
}

// TestNudge2_SuppressedWhenSummaryEmpty verifies that when no summary is
// provided the call returns an error and no nudge key is present.
// (summary is required; nudge2 guard requires non-empty summary.)
func TestNudge2_SuppressedWhenSummaryEmpty(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupFinishTest(t)
	_, taskID, taskSlug := setupNudgeScenario(t, entitySvc, "n2-no-summary")
	advanceToActive(t, entitySvc, taskID, taskSlug)

	// Omit summary — the call should return an error response, not a nudge.
	resp := callFinishNudge(t, entitySvc, dispatchSvc, map[string]any{
		"task_id": taskID,
	})

	// The response should contain an error, not a nudge.
	if _, hasError := resp["error"]; !hasError {
		t.Errorf("expected error response for missing summary, got: %v", resp)
	}
	if nudge, exists := resp["nudge"]; exists && nudge != nil {
		t.Errorf("expected no nudge in error response, got nudge = %v", nudge)
	}
}

// ─── Batch mode tests ─────────────────────────────────────────────────────────

// TestNudge_AbsentInBatchMode verifies that no nudge key appears in batch
// completion results, even when a feature's only task is completed with no retro.
func TestNudge_AbsentInBatchMode(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupFinishTest(t)
	_, taskID, taskSlug := setupNudgeScenario(t, entitySvc, "n-batch")
	advanceToActive(t, entitySvc, taskID, taskSlug)

	resp := callFinishNudge(t, entitySvc, dispatchSvc, map[string]any{
		"tasks": []any{
			map[string]any{
				"task_id": taskID,
				"summary": "Batch completed",
			},
		},
	})

	// Batch response has "results" array.
	results, ok := resp["results"].([]any)
	if !ok || len(results) != 1 {
		t.Fatalf("expected 1 batch result, got: %v", resp["results"])
	}

	item := results[0].(map[string]any)
	if item["status"] != "ok" {
		t.Fatalf("batch item status = %q, want \"ok\"; error: %v", item["status"], item["error"])
	}

	data, ok := item["data"].(map[string]any)
	if !ok {
		t.Fatalf("batch item data missing or not a map: %v", item["data"])
	}

	if nudge, exists := data["nudge"]; exists && nudge != nil {
		t.Errorf("expected no nudge in batch mode result, got nudge = %v", nudge)
	}
}

// ─── Priority tests ───────────────────────────────────────────────────────────

// TestNudge_Nudge1TakesPriorityOverNudge2 verifies that when both nudge
// conditions are met (feature complete with no retro AND summary with no
// knowledge/retro), only nudge1 (nudgeNoRetroSignals) is returned.
func TestNudge_Nudge1TakesPriorityOverNudge2(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupFinishTest(t)
	// Single-task feature: completing it makes the feature fully terminal.
	_, taskID, taskSlug := setupNudgeScenario(t, entitySvc, "n-priority")
	advanceToActive(t, entitySvc, taskID, taskSlug)

	// Both nudge conditions would apply: feature complete + no retro (nudge1),
	// and summary present + no knowledge/retro (nudge2). Nudge1 must win.
	resp := callFinishNudge(t, entitySvc, dispatchSvc, map[string]any{
		"task_id": taskID,
		"summary": "Feature complete, no knowledge, no retro",
	})

	nudge, ok := resp["nudge"].(string)
	if !ok || nudge == "" {
		t.Fatalf("expected a nudge in response, got: %v", resp["nudge"])
	}
	if nudge != nudgeNoRetroSignals {
		t.Errorf("nudge = %q, want nudgeNoRetroSignals (nudge1 priority over nudge2)", nudge)
	}
}
