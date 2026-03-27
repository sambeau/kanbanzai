package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// ─── SideEffectCollector tests ────────────────────────────────────────────────

func TestSideEffectCollector_PushAndDrain(t *testing.T) {
	t.Parallel()

	c := &SideEffectCollector{}
	c.Push(SideEffect{Type: SideEffectTaskUnblocked, EntityID: "TASK-001"})
	c.Push(SideEffect{Type: SideEffectWorktreeCreated, EntityID: "FEAT-001"})

	effects := c.Drain()
	if len(effects) != 2 {
		t.Fatalf("Drain() len = %d, want 2", len(effects))
	}
	if effects[0].EntityID != "TASK-001" {
		t.Errorf("effects[0].EntityID = %q, want TASK-001", effects[0].EntityID)
	}
	if effects[1].EntityID != "FEAT-001" {
		t.Errorf("effects[1].EntityID = %q, want FEAT-001", effects[1].EntityID)
	}
}

func TestSideEffectCollector_EmptyAfterDrain(t *testing.T) {
	t.Parallel()

	c := &SideEffectCollector{}
	c.Push(SideEffect{Type: SideEffectTaskUnblocked, EntityID: "TASK-001"})

	_ = c.Drain()

	// Second drain should return empty.
	second := c.Drain()
	if len(second) != 0 {
		t.Errorf("second Drain() len = %d, want 0 (collector should be empty after drain)", len(second))
	}
}

func TestSideEffectCollector_DrainEmpty(t *testing.T) {
	t.Parallel()

	c := &SideEffectCollector{}
	effects := c.Drain()
	// Should return nil or empty, never panic.
	if len(effects) != 0 {
		t.Errorf("Drain() on empty collector = %v, want empty", effects)
	}
}

func TestSideEffectCollector_Len(t *testing.T) {
	t.Parallel()

	c := &SideEffectCollector{}
	if c.Len() != 0 {
		t.Errorf("Len() = %d, want 0 on empty collector", c.Len())
	}
	c.Push(SideEffect{Type: SideEffectTaskUnblocked})
	c.Push(SideEffect{Type: SideEffectTaskUnblocked})
	if c.Len() != 2 {
		t.Errorf("Len() = %d, want 2", c.Len())
	}
	c.Drain()
	if c.Len() != 0 {
		t.Errorf("Len() after Drain() = %d, want 0", c.Len())
	}
}

func TestSideEffectCollector_ConcurrentPush(t *testing.T) {
	t.Parallel()

	// Verify the collector is goroutine-safe.
	c := &SideEffectCollector{}
	done := make(chan struct{})
	const n = 100

	for i := 0; i < n; i++ {
		go func() {
			c.Push(SideEffect{Type: SideEffectTaskUnblocked, EntityID: "TASK-001"})
			done <- struct{}{}
		}()
	}
	for i := 0; i < n; i++ {
		<-done
	}

	effects := c.Drain()
	if len(effects) != n {
		t.Errorf("Drain() len = %d after %d concurrent pushes, want %d", len(effects), n, n)
	}
}

// ─── Context helper tests ─────────────────────────────────────────────────────

func TestContextWithCollector_RoundTrip(t *testing.T) {
	t.Parallel()

	c := &SideEffectCollector{}
	ctx := ContextWithCollector(context.Background(), c)

	got := CollectorFromContext(ctx)
	if got != c {
		t.Error("CollectorFromContext: got different collector than the one stored")
	}
}

func TestCollectorFromContext_Missing(t *testing.T) {
	t.Parallel()

	// No collector in context → should return nil, not panic.
	got := CollectorFromContext(context.Background())
	if got != nil {
		t.Errorf("CollectorFromContext(empty ctx) = %v, want nil", got)
	}
}

func TestPushSideEffect_WithCollector(t *testing.T) {
	t.Parallel()

	c := &SideEffectCollector{}
	ctx := ContextWithCollector(context.Background(), c)

	PushSideEffect(ctx, SideEffect{Type: SideEffectTaskUnblocked, EntityID: "TASK-001"})

	effects := c.Drain()
	if len(effects) != 1 {
		t.Fatalf("len(effects) = %d, want 1", len(effects))
	}
	if effects[0].EntityID != "TASK-001" {
		t.Errorf("EntityID = %q, want TASK-001", effects[0].EntityID)
	}
}

func TestPushSideEffect_NoCollector_IsNoop(t *testing.T) {
	t.Parallel()

	// Should not panic when there is no collector in the context.
	ctx := context.Background()
	PushSideEffect(ctx, SideEffect{Type: SideEffectTaskUnblocked, EntityID: "TASK-001"})
	// If we reach here without panic, the test passes.
}

// ─── WithSideEffects middleware tests ─────────────────────────────────────────

func TestWithSideEffects_NoSideEffects(t *testing.T) {
	t.Parallel()

	// A handler that produces no side effects should return its result directly.
	inner := func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		return map[string]string{"status": "ok"}, nil
	}
	handler := WithSideEffects(inner)

	result, err := handler(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	text := extractText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("failed to parse result: %v\nraw: %s", err, text)
	}
	if _, hasSideEffects := parsed["side_effects"]; hasSideEffects {
		t.Error("side_effects present in result, want absent when no side effects")
	}
	if parsed["status"] != "ok" {
		t.Errorf("status = %v, want ok", parsed["status"])
	}
}

func TestWithSideEffects_WithSideEffects(t *testing.T) {
	t.Parallel()

	inner := func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		PushSideEffect(ctx, SideEffect{
			Type:       SideEffectTaskUnblocked,
			EntityID:   "TASK-002",
			EntityType: "task",
			ToStatus:   "ready",
		})
		return map[string]string{"completed": "TASK-001"}, nil
	}
	handler := WithSideEffects(inner)

	result, err := handler(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	text := extractText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("failed to parse result: %v\nraw: %s", err, text)
	}

	sideEffects, ok := parsed["side_effects"].([]any)
	if !ok {
		t.Fatalf("side_effects not present or wrong type in result: %v", parsed)
	}
	if len(sideEffects) != 1 {
		t.Fatalf("len(side_effects) = %d, want 1", len(sideEffects))
	}

	se, _ := sideEffects[0].(map[string]any)
	if se["type"] != string(SideEffectTaskUnblocked) {
		t.Errorf("side_effects[0].type = %v, want task_unblocked", se["type"])
	}
	if se["entity_id"] != "TASK-002" {
		t.Errorf("side_effects[0].entity_id = %v, want TASK-002", se["entity_id"])
	}
}

func TestWithSideEffects_MultipleSideEffects(t *testing.T) {
	t.Parallel()

	inner := func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		PushSideEffect(ctx, SideEffect{Type: SideEffectTaskUnblocked, EntityID: "TASK-002"})
		PushSideEffect(ctx, SideEffect{Type: SideEffectTaskUnblocked, EntityID: "TASK-003"})
		PushSideEffect(ctx, SideEffect{Type: SideEffectWorktreeCreated, EntityID: "FEAT-001"})
		return map[string]string{"ok": "true"}, nil
	}
	handler := WithSideEffects(inner)

	result, err := handler(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	text := extractText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("failed to parse result: %v\nraw: %s", err, text)
	}

	sideEffects, _ := parsed["side_effects"].([]any)
	if len(sideEffects) != 3 {
		t.Errorf("len(side_effects) = %d, want 3", len(sideEffects))
	}
}

func TestWithSideEffects_CollectorIsolatedPerRequest(t *testing.T) {
	t.Parallel()

	// Each invocation gets its own fresh collector.
	callCount := 0
	inner := func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		callCount++
		if callCount == 1 {
			PushSideEffect(ctx, SideEffect{Type: SideEffectTaskUnblocked, EntityID: "TASK-001"})
		}
		// Second call pushes nothing.
		return map[string]string{"call": "ok"}, nil
	}
	handler := WithSideEffects(inner)

	// First call — has a side effect.
	r1, _ := handler(context.Background(), mcp.CallToolRequest{})
	text1 := extractText(t, r1)
	var parsed1 map[string]any
	_ = json.Unmarshal([]byte(text1), &parsed1)
	if _, ok := parsed1["side_effects"]; !ok {
		t.Error("first call: expected side_effects, got none")
	}

	// Second call — no side effects (fresh collector).
	r2, _ := handler(context.Background(), mcp.CallToolRequest{})
	text2 := extractText(t, r2)
	var parsed2 map[string]any
	_ = json.Unmarshal([]byte(text2), &parsed2)
	if _, ok := parsed2["side_effects"]; ok {
		t.Error("second call: expected no side_effects, but found them (collector leaked between calls)")
	}
}

// ─── DispatchAction tests ─────────────────────────────────────────────────────

func makeRequest(args map[string]any) mcp.CallToolRequest {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	return req
}

func TestDispatchAction_RoutesToHandler(t *testing.T) {
	t.Parallel()

	called := ""
	handlers := map[string]ActionHandler{
		"create": func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
			called = "create"
			return map[string]string{"action": "create"}, nil
		},
		"get": func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
			called = "get"
			return map[string]string{"action": "get"}, nil
		},
	}

	result, err := DispatchAction(context.Background(), makeRequest(map[string]any{"action": "create"}), handlers)
	if err != nil {
		t.Fatalf("DispatchAction error: %v", err)
	}
	if called != "create" {
		t.Errorf("called = %q, want create", called)
	}
	m, _ := result.(map[string]string)
	if m["action"] != "create" {
		t.Errorf("result action = %q, want create", m["action"])
	}
}

func TestDispatchAction_UnknownAction_ReturnsError(t *testing.T) {
	t.Parallel()

	handlers := map[string]ActionHandler{
		"create": func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
			return nil, nil
		},
		"get": func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
			return nil, nil
		},
	}

	_, err := DispatchAction(context.Background(), makeRequest(map[string]any{"action": "bogus"}), handlers)
	if err == nil {
		t.Fatal("DispatchAction returned nil error for unknown action, want error")
	}

	uae, ok := err.(*UnknownActionError)
	if !ok {
		t.Fatalf("error type = %T, want *UnknownActionError", err)
	}
	if uae.Action != "bogus" {
		t.Errorf("UnknownActionError.Action = %q, want bogus", uae.Action)
	}
	if !strings.Contains(err.Error(), "create") || !strings.Contains(err.Error(), "get") {
		t.Errorf("error message %q should list valid actions (create, get)", err.Error())
	}
}

func TestDispatchAction_MissingAction_ReturnsError(t *testing.T) {
	t.Parallel()

	handlers := map[string]ActionHandler{
		"create": func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
			return nil, nil
		},
	}

	_, err := DispatchAction(context.Background(), makeRequest(map[string]any{}), handlers)
	if err == nil {
		t.Fatal("DispatchAction returned nil error for missing action, want error")
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Errorf("error %q should mention missing parameter", err.Error())
	}
}

func TestDispatchAction_IrrelevantParametersIgnored(t *testing.T) {
	// Verifies §7.3 + §30.2: parameters not needed by the current action
	// are silently ignored, not rejected. This is the key UX guarantee that
	// lets agents pass a superset of parameters without worrying about which
	// are relevant to each action.
	t.Parallel()

	handlers := map[string]ActionHandler{
		"get": func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
			args, _ := req.Params.Arguments.(map[string]any)
			id, _ := args["id"].(string)
			return map[string]string{"id": id}, nil
		},
	}

	// "get" only reads "id"; all other parameters are irrelevant and must
	// not cause an error.
	result, err := DispatchAction(context.Background(), makeRequest(map[string]any{
		"action":         "get",
		"id":             "TASK-001",
		"parent_feature": "FEAT-001",  // irrelevant to get
		"summary":        "some task", // irrelevant to get
		"type":           "task",      // irrelevant to get
		"slug":           "my-task",   // irrelevant to get
	}), handlers)
	if err != nil {
		t.Fatalf("unexpected error for irrelevant parameters: %v (spec §7.3: irrelevant params must be ignored)", err)
	}
	m, ok := result.(map[string]string)
	if !ok {
		t.Fatalf("result type = %T, want map[string]string", result)
	}
	if m["id"] != "TASK-001" {
		t.Errorf("id = %q, want TASK-001", m["id"])
	}
}

func TestDispatchAction_ValidActionsSorted(t *testing.T) {
	t.Parallel()

	// Valid actions in error message should be sorted for deterministic output.
	handlers := map[string]ActionHandler{
		"update":     func(ctx context.Context, req mcp.CallToolRequest) (any, error) { return nil, nil },
		"create":     func(ctx context.Context, req mcp.CallToolRequest) (any, error) { return nil, nil },
		"transition": func(ctx context.Context, req mcp.CallToolRequest) (any, error) { return nil, nil },
		"get":        func(ctx context.Context, req mcp.CallToolRequest) (any, error) { return nil, nil },
	}

	_, err := DispatchAction(context.Background(), makeRequest(map[string]any{"action": "bogus"}), handlers)
	if err == nil {
		t.Fatal("expected error")
	}

	msg := err.Error()
	// Verify "create" appears before "get" which appears before "transition" which appears before "update"
	posCreate := strings.Index(msg, "create")
	posGet := strings.Index(msg, "get")
	posTrans := strings.Index(msg, "transition")
	posUpdate := strings.Index(msg, "update")

	if posCreate < 0 || posGet < 0 || posTrans < 0 || posUpdate < 0 {
		t.Fatalf("not all action names appear in error: %q", msg)
	}
	if !(posCreate < posGet && posGet < posTrans && posTrans < posUpdate) {
		t.Errorf("valid actions not sorted in error message: %q", msg)
	}
}

// ─── ActionError / error response shape tests ─────────────────────────────────

func TestActionError_Shape(t *testing.T) {
	t.Parallel()

	result := ActionError("not_found", "Entity TASK-999 not found", nil)

	text := extractText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse error result: %v\nraw: %s", err, text)
	}

	errField, ok := parsed["error"].(map[string]any)
	if !ok {
		t.Fatalf("error field missing or wrong type: %v", parsed)
	}
	if errField["code"] != "not_found" {
		t.Errorf("error.code = %v, want not_found", errField["code"])
	}
	if !strings.Contains(errField["message"].(string), "TASK-999") {
		t.Errorf("error.message = %q, want mention of TASK-999", errField["message"])
	}

	// side_effects should be absent on a plain error.
	if _, ok := parsed["side_effects"]; ok {
		t.Error("side_effects present in error response, want absent")
	}
}

func TestActionError_WithDetails(t *testing.T) {
	t.Parallel()

	result := ActionError("invalid_transition", "Cannot transition to that status", map[string]any{
		"current_status":    "done",
		"valid_transitions": []string{"superseded"},
	})

	text := extractText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse error result: %v\nraw: %s", err, text)
	}

	errField, _ := parsed["error"].(map[string]any)
	details, ok := errField["details"].(map[string]any)
	if !ok {
		t.Fatalf("error.details missing or wrong type: %v", errField)
	}
	if details["current_status"] != "done" {
		t.Errorf("details.current_status = %v, want done", details["current_status"])
	}
}

func TestBuildResult_SideEffectsInjectedIntoObject(t *testing.T) {
	t.Parallel()

	effects := []SideEffect{
		{Type: SideEffectTaskUnblocked, EntityID: "TASK-002", EntityType: "task", ToStatus: "ready"},
	}

	result := buildResult(map[string]string{"task": "TASK-001", "status": "done"}, effects, false)
	text := extractText(t, result)

	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse result: %v\nraw: %s", err, text)
	}

	if parsed["task"] != "TASK-001" {
		t.Errorf("task = %v, want TASK-001", parsed["task"])
	}
	if parsed["status"] != "done" {
		t.Errorf("status = %v, want done", parsed["status"])
	}
	sideEffects, _ := parsed["side_effects"].([]any)
	if len(sideEffects) != 1 {
		t.Errorf("len(side_effects) = %d, want 1", len(sideEffects))
	}
}

func TestBuildResult_NilResultNilEffects(t *testing.T) {
	t.Parallel()

	result := buildResult(nil, nil, false)
	text := extractText(t, result)
	if text != "{}" {
		t.Errorf("buildResult(nil, nil) = %q, want {}", text)
	}
}

func TestBuildResult_NonObjectWrappedInEnvelope(t *testing.T) {
	t.Parallel()

	// An array result should be wrapped in an envelope when side effects are present.
	effects := []SideEffect{{Type: SideEffectTaskUnblocked, EntityID: "TASK-001"}}
	result := buildResult([]string{"item1", "item2"}, effects, false)
	text := extractText(t, result)

	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse result: %v\nraw: %s", err, text)
	}

	if _, ok := parsed["data"]; !ok {
		t.Error("data field missing from envelope wrapping non-object result")
	}
	if _, ok := parsed["side_effects"]; !ok {
		t.Error("side_effects field missing from envelope")
	}
}

// ─── Mutation with no side effects — side_effects: [] always present ─────────

func TestBuildResult_MutationNoSideEffects(t *testing.T) {
	// Verifies §8.4: mutations always include side_effects: [] even with no cascades.
	t.Parallel()

	result := buildResult(map[string]string{"entity": "TASK-001"}, nil, true)
	text := extractText(t, result)

	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse result: %v\nraw: %s", err, text)
	}

	sideEffects, ok := parsed["side_effects"]
	if !ok {
		t.Fatal("side_effects missing from mutation response, want []")
	}
	arr, ok := sideEffects.([]any)
	if !ok {
		t.Fatalf("side_effects = %T, want []any", sideEffects)
	}
	if len(arr) != 0 {
		t.Errorf("len(side_effects) = %d, want 0", len(arr))
	}
}

func TestWithSideEffects_MutationNoSideEffectsField(t *testing.T) {
	// Verifies §8.4 + §30.2: mutation responses include side_effects: [] even
	// when no cascades occurred. The handler calls SignalMutation to mark itself.
	t.Parallel()

	inner := func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		SignalMutation(ctx) // marks this as a mutation
		// No PushSideEffect calls — no cascades occurred.
		return map[string]string{"entity": "TASK-001", "status": "done"}, nil
	}
	handler := WithSideEffects(inner)

	result, err := handler(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	text := extractText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse result: %v\nraw: %s", err, text)
	}

	sideEffects, ok := parsed["side_effects"]
	if !ok {
		t.Fatal("side_effects missing from mutation response, want []")
	}
	arr, ok := sideEffects.([]any)
	if !ok {
		t.Fatalf("side_effects = %T (%v), want []any", sideEffects, sideEffects)
	}
	if len(arr) != 0 {
		t.Errorf("len(side_effects) = %d, want 0", len(arr))
	}
}

// ─── Read-only operation — no side_effects field ──────────────────────────────

func TestWithSideEffects_ReadOnlyHandler_NoSideEffectsField(t *testing.T) {
	// Verifies §30.2: read-only operations do not include side_effects field.
	t.Parallel()

	inner := func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		// No PushSideEffect calls — this is a read-only handler.
		return map[string]any{"count": 5, "items": []string{"a", "b"}}, nil
	}
	handler := WithSideEffects(inner)

	result, err := handler(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	text := extractText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse result: %v\nraw: %s", err, text)
	}

	if _, ok := parsed["side_effects"]; ok {
		t.Error("side_effects present in read-only operation result, want absent")
	}
}

// ─── Helper ───────────────────────────────────────────────────────────────────

func extractText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if result == nil {
		t.Fatal("result is nil")
	}
	if len(result.Content) == 0 {
		t.Fatal("result has no content")
	}
	tc, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	return tc.Text
}
