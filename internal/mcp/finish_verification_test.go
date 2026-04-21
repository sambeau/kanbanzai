package mcp

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/sambeau/kanbanzai/internal/service"
)

// ─── Shared-feature helpers ───────────────────────────────────────────────────

// setupFinishSharedFeature creates a plan and a feature that can hold multiple tasks.
// Returns (planID, featID).
func setupFinishSharedFeature(t *testing.T, entitySvc *service.EntityService, suffix string) (string, string) {
	t.Helper()
	planID := createFinishTestPlan(t, entitySvc, "shared-"+suffix)
	featID := createFinishTestFeature(t, entitySvc, planID, "shared-feat-"+suffix)
	return planID, featID
}

// ─── Verification aggregation tests ──────────────────────────────────────────

// TestFinishOne_AggregatesOnLastTask verifies that verification aggregation fires
// only when the last sibling task is completed (all tasks terminal), not before.
func TestFinishOne_AggregatesOnLastTask(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupFinishTest(t)
	_, featID := setupFinishSharedFeature(t, entitySvc, "last-task")

	// Two tasks under the same feature.
	taskID1, taskSlug1 := createFinishTestTask(t, entitySvc, featID, "lt-task-1")
	taskID2, taskSlug2 := createFinishTestTask(t, entitySvc, featID, "lt-task-2")
	advanceToActive(t, entitySvc, taskID1, taskSlug1)
	advanceToActive(t, entitySvc, taskID2, taskSlug2)

	// Finish first task — second still active, so NOT all terminal.
	resp1 := callFinishJSON(t, entitySvc, dispatchSvc, map[string]any{
		"task_id":      taskID1,
		"summary":      "First task done",
		"verification": "unit tests passed",
	})
	if _, hasAgg := resp1["verification_aggregation"]; hasAgg {
		t.Error("verification_aggregation present after first task; expected absent (not all terminal)")
	}

	// Finish second (last) task — now all terminal -> aggregation fires.
	resp2 := callFinishJSON(t, entitySvc, dispatchSvc, map[string]any{
		"task_id":      taskID2,
		"summary":      "Second task done",
		"verification": "integration tests passed",
	})
	agg, hasAgg := resp2["verification_aggregation"]
	if !hasAgg {
		t.Fatal("verification_aggregation absent after last task finish; expected present")
	}
	aggMap, ok := agg.(map[string]any)
	if !ok {
		t.Fatalf("verification_aggregation is not a map: %T %v", agg, agg)
	}
	if aggMap["status"] != "passed" {
		t.Errorf("verification_aggregation.status = %v, want %q", aggMap["status"], "passed")
	}
	if aggMap["written"] != true {
		t.Errorf("verification_aggregation.written = %v, want true", aggMap["written"])
	}
}

// TestFinishBatch_DefersAggregation verifies that batch-finishing all remaining tasks
// triggers aggregation exactly once, in the result of the last processed task.
func TestFinishBatch_DefersAggregation(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupFinishTest(t)
	_, featID := setupFinishSharedFeature(t, entitySvc, "batch-agg")

	taskID1, taskSlug1 := createFinishTestTask(t, entitySvc, featID, "ba-task-1")
	taskID2, taskSlug2 := createFinishTestTask(t, entitySvc, featID, "ba-task-2")
	advanceToActive(t, entitySvc, taskID1, taskSlug1)
	advanceToActive(t, entitySvc, taskID2, taskSlug2)

	text := callFinish(t, entitySvc, dispatchSvc, map[string]any{
		"tasks": []any{
			map[string]any{"task_id": taskID1, "summary": "Task 1 done", "verification": "unit tests pass"},
			map[string]any{"task_id": taskID2, "summary": "Task 2 done", "verification": "integration tests pass"},
		},
	})
	var resp map[string]any
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("parse batch response: %v\nraw: %s", err, text)
	}

	results, _ := resp["results"].([]any)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d; resp: %v", len(results), resp)
	}

	// First item result: task2 still active when task1 ran -> no aggregation.
	first := results[0].(map[string]any)
	firstData, _ := first["data"].(map[string]any)
	if _, hasAgg := firstData["verification_aggregation"]; hasAgg {
		t.Error("first batch item has verification_aggregation; should be absent (not all terminal yet)")
	}

	// Second (last) item result: all tasks terminal -> aggregation fires.
	second := results[1].(map[string]any)
	secondData, _ := second["data"].(map[string]any)
	agg, hasAgg := secondData["verification_aggregation"]
	if !hasAgg {
		t.Fatal("last batch item missing verification_aggregation")
	}
	aggMap, ok := agg.(map[string]any)
	if !ok {
		t.Fatalf("verification_aggregation is not a map: %T %v", agg, agg)
	}
	if aggMap["status"] != "passed" {
		t.Errorf("verification_aggregation.status = %v, want %q", aggMap["status"], "passed")
	}
}

// TestFinishOne_AggregationWriteFailureDoesNotFailFinish verifies that a failure
// to write the feature entity during aggregation does not prevent task completion
// (best-effort semantics: task is still marked done, no top-level error).
func TestFinishOne_AggregationWriteFailureDoesNotFailFinish(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupFinishTest(t)
	_, featID := setupFinishSharedFeature(t, entitySvc, "write-fail")

	taskID, taskSlug := createFinishTestTask(t, entitySvc, featID, "wf-task")
	advanceToActive(t, entitySvc, taskID, taskSlug)

	// Delete the feature entity file to force UpdateEntity to fail during aggregation.
	feat, err := entitySvc.Get("feature", featID, "")
	if err != nil {
		t.Fatalf("get feature: %v", err)
	}
	if err := os.Remove(feat.Path); err != nil {
		t.Fatalf("remove feature entity: %v", err)
	}

	resp := callFinishJSON(t, entitySvc, dispatchSvc, map[string]any{
		"task_id":      taskID,
		"summary":      "Done despite write failure",
		"verification": "all tests pass",
	})

	// Task must still be done.
	taskData, ok := resp["task"].(map[string]any)
	if !ok {
		t.Fatalf("expected task in response, got: %v", resp)
	}
	if taskData["status"] != "done" {
		t.Errorf("task status = %q, want %q", taskData["status"], "done")
	}
	// No top-level error.
	if _, hasErr := resp["error"]; hasErr {
		t.Errorf("unexpected error in response: %v", resp["error"])
	}
	// verification_aggregation, if present, must report written=false.
	if agg, hasAgg := resp["verification_aggregation"]; hasAgg {
		aggMap, _ := agg.(map[string]any)
		if aggMap["written"] == true {
			t.Error("verification_aggregation.written should be false when feature entity write fails")
		}
	}
}

// TestFinishOne_ResponseContainsAggregationKey verifies that the MCP response
// includes the verification_aggregation key with the correct status when the
// finished task is the last terminal task in its feature.
func TestFinishOne_ResponseContainsAggregationKey(t *testing.T) {
	t.Parallel()
	entitySvc, dispatchSvc := setupFinishTest(t)
	_, featID := setupFinishSharedFeature(t, entitySvc, "resp-key")

	taskID, taskSlug := createFinishTestTask(t, entitySvc, featID, "rk-task")
	advanceToActive(t, entitySvc, taskID, taskSlug)

	resp := callFinishJSON(t, entitySvc, dispatchSvc, map[string]any{
		"task_id":      taskID,
		"summary":      "Implemented and tested",
		"verification": "go test ./... passed",
	})

	agg, hasAgg := resp["verification_aggregation"]
	if !hasAgg {
		t.Fatal("verification_aggregation key absent from response (expected on last-task finish)")
	}
	aggMap, ok := agg.(map[string]any)
	if !ok {
		t.Fatalf("verification_aggregation not a map: %T %v", agg, agg)
	}
	if aggMap["status"] != "passed" {
		t.Errorf("verification_aggregation.status = %v, want %q", aggMap["status"], "passed")
	}
	if aggMap["written"] != true {
		t.Errorf("verification_aggregation.written = %v, want true", aggMap["written"])
	}
}
