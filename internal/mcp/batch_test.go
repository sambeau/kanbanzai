package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
)

// ─── ExecuteBatch tests ───────────────────────────────────────────────────────

func TestExecuteBatch_SingleItem(t *testing.T) {
	t.Parallel()

	// Single-item batch should return a BatchResult with one result.
	items := []any{"TASK-001"}
	result, err := ExecuteBatch(context.Background(), items, func(ctx context.Context, item any) (string, any, error) {
		id, _ := item.(string)
		return id, map[string]string{"task": id, "status": "done"}, nil
	})
	if err != nil {
		t.Fatalf("ExecuteBatch error: %v", err)
	}

	br, ok := result.(*BatchResult)
	if !ok {
		t.Fatalf("result type = %T, want *BatchResult", result)
	}
	if len(br.Results) != 1 {
		t.Fatalf("len(Results) = %d, want 1", len(br.Results))
	}
	if br.Results[0].Status != "ok" {
		t.Errorf("Results[0].Status = %q, want ok", br.Results[0].Status)
	}
	if br.Results[0].ItemID != "TASK-001" {
		t.Errorf("Results[0].ItemID = %q, want TASK-001", br.Results[0].ItemID)
	}
	if br.Summary.Total != 1 {
		t.Errorf("Summary.Total = %d, want 1", br.Summary.Total)
	}
	if br.Summary.Succeeded != 1 {
		t.Errorf("Summary.Succeeded = %d, want 1", br.Summary.Succeeded)
	}
	if br.Summary.Failed != 0 {
		t.Errorf("Summary.Failed = %d, want 0", br.Summary.Failed)
	}
}

func TestExecuteBatch_MultipleItems(t *testing.T) {
	t.Parallel()

	// Verifies §30.3: batch call returns BatchResult with per-item results.
	items := []any{"TASK-001", "TASK-002", "TASK-003"}
	result, err := ExecuteBatch(context.Background(), items, func(ctx context.Context, item any) (string, any, error) {
		id, _ := item.(string)
		return id, map[string]string{"task": id, "status": "done"}, nil
	})
	if err != nil {
		t.Fatalf("ExecuteBatch error: %v", err)
	}

	br, ok := result.(*BatchResult)
	if !ok {
		t.Fatalf("result type = %T, want *BatchResult", result)
	}
	if len(br.Results) != 3 {
		t.Fatalf("len(Results) = %d, want 3", len(br.Results))
	}
	if br.Summary.Total != 3 {
		t.Errorf("Summary.Total = %d, want 3", br.Summary.Total)
	}
	if br.Summary.Succeeded != 3 {
		t.Errorf("Summary.Succeeded = %d, want 3", br.Summary.Succeeded)
	}
	if br.Summary.Failed != 0 {
		t.Errorf("Summary.Failed = %d, want 0", br.Summary.Failed)
	}

	// Items are returned in input order.
	for i, item := range items {
		if br.Results[i].ItemID != item {
			t.Errorf("Results[%d].ItemID = %q, want %q", i, br.Results[i].ItemID, item)
		}
	}
}

func TestExecuteBatch_PartialFailure(t *testing.T) {
	t.Parallel()

	// Verifies §30.3: one item fails, others succeed, summary counts are correct.
	items := []any{"TASK-001", "TASK-002", "TASK-003"}
	result, err := ExecuteBatch(context.Background(), items, func(ctx context.Context, item any) (string, any, error) {
		id, _ := item.(string)
		if id == "TASK-002" {
			return id, nil, fmt.Errorf("task TASK-002 not found")
		}
		return id, map[string]string{"task": id, "status": "done"}, nil
	})
	if err != nil {
		t.Fatalf("ExecuteBatch error: %v", err)
	}

	br, ok := result.(*BatchResult)
	if !ok {
		t.Fatalf("result type = %T, want *BatchResult", result)
	}

	if br.Summary.Total != 3 {
		t.Errorf("Summary.Total = %d, want 3", br.Summary.Total)
	}
	if br.Summary.Succeeded != 2 {
		t.Errorf("Summary.Succeeded = %d, want 2", br.Summary.Succeeded)
	}
	if br.Summary.Failed != 1 {
		t.Errorf("Summary.Failed = %d, want 1", br.Summary.Failed)
	}

	// TASK-001: ok
	if br.Results[0].Status != "ok" {
		t.Errorf("Results[0].Status = %q, want ok", br.Results[0].Status)
	}
	// TASK-002: error
	if br.Results[1].Status != "error" {
		t.Errorf("Results[1].Status = %q, want error", br.Results[1].Status)
	}
	if br.Results[1].Error == nil {
		t.Error("Results[1].Error = nil, want non-nil error detail")
	} else if br.Results[1].Error.Message == "" {
		t.Error("Results[1].Error.Message is empty, want error description")
	}
	// TASK-003: ok (processed despite TASK-002 failing)
	if br.Results[2].Status != "ok" {
		t.Errorf("Results[2].Status = %q, want ok (item after failure should still be processed)", br.Results[2].Status)
	}
}

func TestExecuteBatch_AllFail(t *testing.T) {
	t.Parallel()

	items := []any{"TASK-001", "TASK-002"}
	result, err := ExecuteBatch(context.Background(), items, func(ctx context.Context, item any) (string, any, error) {
		id, _ := item.(string)
		return id, nil, fmt.Errorf("service unavailable")
	})
	if err != nil {
		t.Fatalf("ExecuteBatch error: %v", err)
	}

	br, ok := result.(*BatchResult)
	if !ok {
		t.Fatalf("result type = %T, want *BatchResult", result)
	}
	if br.Summary.Failed != 2 {
		t.Errorf("Summary.Failed = %d, want 2", br.Summary.Failed)
	}
	if br.Summary.Succeeded != 0 {
		t.Errorf("Summary.Succeeded = %d, want 0", br.Summary.Succeeded)
	}
}

func TestExecuteBatch_LimitExceeded(t *testing.T) {
	t.Parallel()

	// Verifies §30.3: reject batches >100 items before any processing.
	items := make([]any, MaxBatchSize+1)
	for i := range items {
		items[i] = fmt.Sprintf("TASK-%03d", i)
	}

	handlerCalled := false
	_, err := ExecuteBatch(context.Background(), items, func(ctx context.Context, item any) (string, any, error) {
		handlerCalled = true
		return "", nil, nil
	})

	if err == nil {
		t.Fatal("ExecuteBatch: expected error for oversized batch, got nil")
	}
	if handlerCalled {
		t.Error("handler was called despite batch limit being exceeded (should reject before processing)")
	}
}

func TestExecuteBatch_ExactlyAtLimit(t *testing.T) {
	t.Parallel()

	// Exactly MaxBatchSize items should succeed (not exceed).
	items := make([]any, MaxBatchSize)
	for i := range items {
		items[i] = fmt.Sprintf("TASK-%03d", i)
	}

	result, err := ExecuteBatch(context.Background(), items, func(ctx context.Context, item any) (string, any, error) {
		id, _ := item.(string)
		return id, map[string]string{"ok": "true"}, nil
	})
	if err != nil {
		t.Fatalf("ExecuteBatch with exactly %d items returned error: %v", MaxBatchSize, err)
	}
	br, _ := result.(*BatchResult)
	if br.Summary.Total != MaxBatchSize {
		t.Errorf("Summary.Total = %d, want %d", br.Summary.Total, MaxBatchSize)
	}
}

func TestExecuteBatch_EmptyBatch(t *testing.T) {
	t.Parallel()

	result, err := ExecuteBatch(context.Background(), []any{}, func(ctx context.Context, item any) (string, any, error) {
		return "", nil, nil
	})
	if err != nil {
		t.Fatalf("ExecuteBatch empty batch error: %v", err)
	}
	br, ok := result.(*BatchResult)
	if !ok {
		t.Fatalf("result type = %T, want *BatchResult", result)
	}
	if br.Summary.Total != 0 {
		t.Errorf("Summary.Total = %d, want 0", br.Summary.Total)
	}
}

func TestExecuteBatch_SideEffectsAggregated(t *testing.T) {
	t.Parallel()

	// Verifies §30.3: aggregate side effects are union of per-item side effects.
	items := []any{"TASK-001", "TASK-002"}
	result, err := ExecuteBatch(context.Background(), items, func(ctx context.Context, item any) (string, any, error) {
		id, _ := item.(string)
		// Each item unblocks a downstream task.
		PushSideEffect(ctx, SideEffect{
			Type:       SideEffectTaskUnblocked,
			EntityID:   "DOWNSTREAM-" + id,
			EntityType: "task",
			ToStatus:   "ready",
		})
		return id, map[string]string{"task": id}, nil
	})
	if err != nil {
		t.Fatalf("ExecuteBatch error: %v", err)
	}

	br, ok := result.(*BatchResult)
	if !ok {
		t.Fatalf("result type = %T, want *BatchResult", result)
	}

	// Per-item side effects should be on each result.
	if len(br.Results[0].SideEffects) != 1 {
		t.Errorf("Results[0].SideEffects len = %d, want 1", len(br.Results[0].SideEffects))
	}
	if len(br.Results[1].SideEffects) != 1 {
		t.Errorf("Results[1].SideEffects len = %d, want 1", len(br.Results[1].SideEffects))
	}

	// Aggregate side effects should be the union of all per-item effects.
	if len(br.SideEffects) != 2 {
		t.Errorf("BatchResult.SideEffects len = %d, want 2 (union of per-item effects)", len(br.SideEffects))
	}
}

func TestExecuteBatch_SideEffectsAbsentWhenEmpty(t *testing.T) {
	t.Parallel()

	// When no side effects are produced, the fields should be absent from JSON.
	items := []any{"TASK-001"}
	result, err := ExecuteBatch(context.Background(), items, func(ctx context.Context, item any) (string, any, error) {
		id, _ := item.(string)
		return id, map[string]string{"task": id}, nil
	})
	if err != nil {
		t.Fatalf("ExecuteBatch error: %v", err)
	}

	b, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if _, ok := parsed["side_effects"]; ok {
		t.Error("side_effects present when no effects produced, want absent")
	}

	results, _ := parsed["results"].([]any)
	if len(results) != 1 {
		t.Fatalf("results len = %d, want 1", len(results))
	}
	if _, ok := results[0].(map[string]any)["side_effects"]; ok {
		t.Error("per-item side_effects present when no effects produced, want absent")
	}
}

func TestExecuteBatch_InputOrderPreserved(t *testing.T) {
	t.Parallel()

	// Results must be in the same order as the input items.
	items := []any{"TASK-C", "TASK-A", "TASK-B"}
	result, err := ExecuteBatch(context.Background(), items, func(ctx context.Context, item any) (string, any, error) {
		id, _ := item.(string)
		return id, nil, nil
	})
	if err != nil {
		t.Fatalf("ExecuteBatch error: %v", err)
	}

	br, _ := result.(*BatchResult)
	for i, expected := range []string{"TASK-C", "TASK-A", "TASK-B"} {
		if br.Results[i].ItemID != expected {
			t.Errorf("Results[%d].ItemID = %q, want %q (input order must be preserved)", i, br.Results[i].ItemID, expected)
		}
	}
}

// ─── ExecuteBatch tool-result error detection tests (AC-12 through AC-15) ─────

// AC-12: A handler returning (id, map{"error":"msg"}, nil) is counted as failed.
func TestExecuteBatch_ToolResultError_CountedAsFailed(t *testing.T) {
	t.Parallel()

	items := []any{"TASK-001"}
	result, err := ExecuteBatch(context.Background(), items, func(ctx context.Context, item any) (string, any, error) {
		id, _ := item.(string)
		return id, map[string]any{"error": "validation failed: missing required field"}, nil
	})
	if err != nil {
		t.Fatalf("ExecuteBatch error: %v", err)
	}

	br, ok := result.(*BatchResult)
	if !ok {
		t.Fatalf("result type = %T, want *BatchResult", result)
	}
	if br.Summary.Failed != 1 {
		t.Errorf("Summary.Failed = %d, want 1 (tool-result error must be counted as failed)", br.Summary.Failed)
	}
	if br.Summary.Succeeded != 0 {
		t.Errorf("Summary.Succeeded = %d, want 0", br.Summary.Succeeded)
	}
}

// AC-13: ItemResult for a tool-result error has Status "error" and Error contains
// the message from the tool-result payload.
func TestExecuteBatch_ToolResultError_ItemResultShape(t *testing.T) {
	t.Parallel()

	const errMsg = "entity not found: TASK-999"
	items := []any{"TASK-999"}
	result, err := ExecuteBatch(context.Background(), items, func(ctx context.Context, item any) (string, any, error) {
		id, _ := item.(string)
		return id, map[string]any{"error": errMsg}, nil
	})
	if err != nil {
		t.Fatalf("ExecuteBatch error: %v", err)
	}

	br, _ := result.(*BatchResult)
	if len(br.Results) != 1 {
		t.Fatalf("len(Results) = %d, want 1", len(br.Results))
	}
	r := br.Results[0]
	if r.Status != "error" {
		t.Errorf("Results[0].Status = %q, want \"error\"", r.Status)
	}
	if r.Error == nil {
		t.Fatal("Results[0].Error = nil, want non-nil ErrorDetail")
	}
	if r.Error.Message != errMsg {
		t.Errorf("Results[0].Error.Message = %q, want %q", r.Error.Message, errMsg)
	}
	if r.Data != nil {
		t.Errorf("Results[0].Data = %v, want nil (data must be absent for error items)", r.Data)
	}
}

// AC-14: A handler returning genuine success data (no "error" key) is still
// counted as succeeded with no change in result shape.
func TestExecuteBatch_GenuineSuccess_NotAffectedByToolErrorCheck(t *testing.T) {
	t.Parallel()

	items := []any{"TASK-001"}
	result, err := ExecuteBatch(context.Background(), items, func(ctx context.Context, item any) (string, any, error) {
		id, _ := item.(string)
		return id, map[string]any{"result": "ok", "status": "done"}, nil
	})
	if err != nil {
		t.Fatalf("ExecuteBatch error: %v", err)
	}

	br, _ := result.(*BatchResult)
	if br.Summary.Succeeded != 1 {
		t.Errorf("Summary.Succeeded = %d, want 1 (genuine success must remain succeeded)", br.Summary.Succeeded)
	}
	if br.Summary.Failed != 0 {
		t.Errorf("Summary.Failed = %d, want 0", br.Summary.Failed)
	}
	if br.Results[0].Status != "ok" {
		t.Errorf("Results[0].Status = %q, want \"ok\"", br.Results[0].Status)
	}
}

// AC-15: A handler returning a non-nil Go error is still counted as failed,
// regardless of the data value.
func TestExecuteBatch_GoError_CountedAsFailed(t *testing.T) {
	t.Parallel()

	items := []any{"TASK-001"}
	result, err := ExecuteBatch(context.Background(), items, func(ctx context.Context, item any) (string, any, error) {
		id, _ := item.(string)
		return id, nil, fmt.Errorf("service unavailable")
	})
	if err != nil {
		t.Fatalf("ExecuteBatch error: %v", err)
	}

	br, _ := result.(*BatchResult)
	if br.Summary.Failed != 1 {
		t.Errorf("Summary.Failed = %d, want 1 (Go error must be counted as failed)", br.Summary.Failed)
	}
	if br.Results[0].Status != "error" {
		t.Errorf("Results[0].Status = %q, want \"error\"", br.Results[0].Status)
	}
	if br.Results[0].Error == nil || br.Results[0].Error.Message != "service unavailable" {
		t.Errorf("Results[0].Error.Message = %q, want \"service unavailable\"", br.Results[0].Error.Message)
	}
}

// REQ-08: A batch with one tool-result-error item and one genuine success
// produces succeeded=1, failed=1 (mutually exclusive and exhaustive).
func TestExecuteBatch_MixedToolErrorAndSuccess_SummaryCorrect(t *testing.T) {
	t.Parallel()

	items := []any{"TASK-001", "TASK-002"}
	result, err := ExecuteBatch(context.Background(), items, func(ctx context.Context, item any) (string, any, error) {
		id, _ := item.(string)
		if id == "TASK-001" {
			return id, map[string]any{"error": "not found"}, nil
		}
		return id, map[string]any{"status": "done"}, nil
	})
	if err != nil {
		t.Fatalf("ExecuteBatch error: %v", err)
	}

	br, _ := result.(*BatchResult)
	if br.Summary.Total != 2 {
		t.Errorf("Summary.Total = %d, want 2", br.Summary.Total)
	}
	if br.Summary.Succeeded != 1 {
		t.Errorf("Summary.Succeeded = %d, want 1", br.Summary.Succeeded)
	}
	if br.Summary.Failed != 1 {
		t.Errorf("Summary.Failed = %d, want 1", br.Summary.Failed)
	}
	// TASK-001: error (tool-result error)
	if br.Results[0].Status != "error" {
		t.Errorf("Results[0].Status = %q, want \"error\"", br.Results[0].Status)
	}
	// TASK-002: ok (genuine success)
	if br.Results[1].Status != "ok" {
		t.Errorf("Results[1].Status = %q, want \"ok\"", br.Results[1].Status)
	}
}

// extractToolResultError: empty string error value is not treated as an error.
func TestExtractToolResultError_EmptyStringNotAnError(t *testing.T) {
	t.Parallel()

	items := []any{"TASK-001"}
	result, err := ExecuteBatch(context.Background(), items, func(ctx context.Context, item any) (string, any, error) {
		id, _ := item.(string)
		return id, map[string]any{"error": ""}, nil
	})
	if err != nil {
		t.Fatalf("ExecuteBatch error: %v", err)
	}

	br, _ := result.(*BatchResult)
	// Empty string in "error" key is not a tool-result error; should succeed.
	if br.Summary.Succeeded != 1 {
		t.Errorf("Summary.Succeeded = %d, want 1 (empty error string is not a tool-result error)", br.Summary.Succeeded)
	}
}

// ─── IsBatchInput tests ───────────────────────────────────────────────────────

func TestIsBatchInput_True(t *testing.T) {
	t.Parallel()

	args := map[string]any{
		"tasks": []any{"TASK-001", "TASK-002"},
	}
	if !IsBatchInput(args, "tasks") {
		t.Error("IsBatchInput = false, want true for non-nil array value")
	}
}

func TestIsBatchInput_False_MissingKey(t *testing.T) {
	t.Parallel()

	args := map[string]any{
		"task_id": "TASK-001",
	}
	if IsBatchInput(args, "tasks") {
		t.Error("IsBatchInput = true, want false for missing key")
	}
}

func TestIsBatchInput_False_NonArray(t *testing.T) {
	t.Parallel()

	args := map[string]any{
		"tasks": "TASK-001", // string, not array
	}
	if IsBatchInput(args, "tasks") {
		t.Error("IsBatchInput = true, want false for non-array value")
	}
}

func TestIsBatchInput_False_NilArgs(t *testing.T) {
	t.Parallel()

	if IsBatchInput(nil, "tasks") {
		t.Error("IsBatchInput = true, want false for nil args")
	}
}

func TestIsBatchInput_False_EmptyArray(t *testing.T) {
	t.Parallel()

	// An empty slice is still valid batch input (returns true; caller handles empty case).
	args := map[string]any{
		"tasks": []any{},
	}
	// Empty array is not nil — it's a valid (empty) batch.
	if !IsBatchInput(args, "tasks") {
		t.Error("IsBatchInput = false, want true for empty (non-nil) array")
	}
}

// ─── BatchResult JSON shape tests ─────────────────────────────────────────────

func TestBatchResult_JSONShape(t *testing.T) {
	t.Parallel()

	// Verifies the JSON output shape matches the spec.
	items := []any{"TASK-001", "TASK-002"}
	result, _ := ExecuteBatch(context.Background(), items, func(ctx context.Context, item any) (string, any, error) {
		id, _ := item.(string)
		return id, map[string]string{"completed": id}, nil
	})

	b, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	// Must have results, summary, no side_effects (none were produced).
	if _, ok := parsed["results"]; !ok {
		t.Error("results field missing from BatchResult JSON")
	}
	if _, ok := parsed["summary"]; !ok {
		t.Error("summary field missing from BatchResult JSON")
	}

	summary, _ := parsed["summary"].(map[string]any)
	if summary["total"] != float64(2) {
		t.Errorf("summary.total = %v, want 2", summary["total"])
	}
	if summary["succeeded"] != float64(2) {
		t.Errorf("summary.succeeded = %v, want 2", summary["succeeded"])
	}
	if summary["failed"] != float64(0) {
		t.Errorf("summary.failed = %v, want 0", summary["failed"])
	}

	results, _ := parsed["results"].([]any)
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	r0, _ := results[0].(map[string]any)
	if r0["item_id"] != "TASK-001" {
		t.Errorf("results[0].item_id = %v, want TASK-001", r0["item_id"])
	}
	if r0["status"] != "ok" {
		t.Errorf("results[0].status = %v, want ok", r0["status"])
	}
}
