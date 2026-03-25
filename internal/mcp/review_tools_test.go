package mcp_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/mcptest"

	kbzmcp "kanbanzai/internal/mcp"
	"kanbanzai/internal/service"
)

// reviewEnv holds the test server and service references for review tests.
type reviewEnv struct {
	server     *mcptest.Server
	entitySvc  *service.EntityService
	entityRoot string
	repoRoot   string
	featureID  string
	taskID     string
}

// setupReviewTestServer creates a test MCP server with entity and review tools
// registered. It creates a plan, feature, and active task for review testing.
func setupReviewTestServer(t *testing.T) *reviewEnv {
	t.Helper()

	entityRoot := t.TempDir()
	repoRoot := t.TempDir()
	indexRoot := t.TempDir()

	entitySvc := service.NewEntityService(entityRoot)
	intelSvc := service.NewIntelligenceService(indexRoot, repoRoot)
	reviewSvc := service.NewReviewService(entitySvc, intelSvc, repoRoot)

	// Write a plan directly to disk, bypassing the prefix registry/allocator.
	planDir := filepath.Join(entityRoot, "plans")
	if err := os.MkdirAll(planDir, 0o755); err != nil {
		t.Fatalf("mkdir plans: %v", err)
	}
	planYAML := "id: P1-review-test\nslug: review-test\ntitle: Test Plan\nstatus: active\nsummary: Plan for review tests\ncreated: \"2026-03-19T12:00:00Z\"\ncreated_by: test\nupdated: \"2026-03-19T12:00:00Z\"\n"
	if err := os.WriteFile(filepath.Join(planDir, "P1-review-test.yaml"), []byte(planYAML), 0o644); err != nil {
		t.Fatalf("write plan file: %v", err)
	}

	tools := kbzmcp.EntityTools(entitySvc)
	tools = append(tools, kbzmcp.ReviewTools(reviewSvc)...)

	ts, err := mcptest.NewServer(t, tools...)
	if err != nil {
		t.Fatalf("start review test server: %v", err)
	}

	env := &reviewEnv{
		server:     ts,
		entitySvc:  entitySvc,
		entityRoot: entityRoot,
		repoRoot:   repoRoot,
	}

	// Create a feature via MCP tool.
	featResult := callReviewTool(t, env, "create_feature", map[string]any{
		"slug":       "review-feat",
		"parent":     "P1-review-test",
		"summary":    "Feature for review MCP tests",
		"created_by": "tester",
	})
	if featResult.IsError {
		t.Fatalf("create_feature returned error: %s", reviewResultText(t, featResult))
	}
	var featParsed map[string]any
	if err := json.Unmarshal([]byte(reviewResultText(t, featResult)), &featParsed); err != nil {
		t.Fatalf("parse create_feature result: %v", err)
	}
	featureID, ok := featParsed["ID"].(string)
	if !ok || featureID == "" {
		t.Fatal("create_feature returned empty ID")
	}
	env.featureID = featureID

	// Create a task via MCP tool.
	taskResult := callReviewTool(t, env, "create_task", map[string]any{
		"parent_feature": featureID,
		"slug":           "review-mcp-task",
		"summary":        "Implement authentication middleware",
	})
	if taskResult.IsError {
		t.Fatalf("create_task returned error: %s", reviewResultText(t, taskResult))
	}
	var taskParsed map[string]any
	if err := json.Unmarshal([]byte(reviewResultText(t, taskResult)), &taskParsed); err != nil {
		t.Fatalf("parse create_task result: %v", err)
	}
	taskID, ok := taskParsed["ID"].(string)
	if !ok || taskID == "" {
		t.Fatal("create_task returned empty ID")
	}
	env.taskID = taskID

	// Transition task: queued → ready → active.
	for _, status := range []string{"ready", "active"} {
		statusResult := callReviewTool(t, env, "update_status", map[string]any{
			"entity_type": "task",
			"id":          taskID,
			"status":      status,
		})
		if statusResult.IsError {
			t.Fatalf("update_status to %s returned error: %s", status, reviewResultText(t, statusResult))
		}
	}

	return env
}

func callReviewTool(t *testing.T, env *reviewEnv, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	ctx := context.Background()
	request := mcp.CallToolRequest{}
	request.Params.Name = name
	request.Params.Arguments = args
	result, err := env.server.Client().CallTool(ctx, request)
	if err != nil {
		t.Fatalf("CallTool(%s): %v", name, err)
	}
	return result
}

func reviewResultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if len(result.Content) == 0 {
		t.Fatal("result has no content")
	}
	tc, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	return tc.Text
}

func TestReviewTaskOutput_MCP_MissingFile_Fails(t *testing.T) {
	t.Parallel()

	env := setupReviewTestServer(t)
	defer env.server.Close()

	result := callReviewTool(t, env, "review_task_output", map[string]any{
		"task_id":      env.taskID,
		"output_files": []any{"nonexistent/file.go"},
	})
	if result.IsError {
		t.Fatalf("review_task_output returned transport error: %s", reviewResultText(t, result))
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(reviewResultText(t, result)), &parsed); err != nil {
		t.Fatalf("parse review_task_output result: %v\nraw: %s", err, reviewResultText(t, result))
	}

	status, _ := parsed["status"].(string)
	if status != "fail" {
		t.Errorf("status = %q, want %q", status, "fail")
	}

	blockingCount, _ := parsed["blocking_count"].(float64)
	if blockingCount < 1 {
		t.Errorf("blocking_count = %v, want >= 1", blockingCount)
	}

	// Verify the task transitioned to needs-rework.
	taskResult, err := env.entitySvc.Get("task", env.taskID, "")
	if err != nil {
		t.Fatalf("Get task: %v", err)
	}
	taskStatus, _ := taskResult.State["status"].(string)
	if taskStatus != "needs-rework" {
		t.Errorf("task status = %q, want %q", taskStatus, "needs-rework")
	}
	reworkReason, _ := taskResult.State["rework_reason"].(string)
	if reworkReason == "" {
		t.Error("rework_reason should be set on failing review")
	}
}

func TestReviewTaskOutput_MCP_Pass_TransitionsToNeedsReview(t *testing.T) {
	t.Parallel()

	env := setupReviewTestServer(t)
	defer env.server.Close()

	// Create a real file in the repo root so the file check passes.
	realFile := "internal/auth.go"
	fullPath := filepath.Join(env.repoRoot, realFile)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte("package auth"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	result := callReviewTool(t, env, "review_task_output", map[string]any{
		"task_id":        env.taskID,
		"output_files":   []any{realFile},
		"output_summary": "Implemented authentication middleware with request validation",
	})
	if result.IsError {
		t.Fatalf("review_task_output returned transport error: %s", reviewResultText(t, result))
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(reviewResultText(t, result)), &parsed); err != nil {
		t.Fatalf("parse review_task_output result: %v\nraw: %s", err, reviewResultText(t, result))
	}

	status, _ := parsed["status"].(string)
	if status == "fail" {
		t.Errorf("status = %q, want pass or pass_with_warnings", status)
	}

	blockingCount, _ := parsed["blocking_count"].(float64)
	if blockingCount > 0 {
		t.Errorf("blocking_count = %v, want 0", blockingCount)
	}

	// Verify the task transitioned to needs-review.
	taskResult, err := env.entitySvc.Get("task", env.taskID, "")
	if err != nil {
		t.Fatalf("Get task: %v", err)
	}
	taskStatus, _ := taskResult.State["status"].(string)
	if taskStatus != "needs-review" {
		t.Errorf("task status = %q, want %q", taskStatus, "needs-review")
	}
}

func TestReviewTaskOutput_MCP_NoSpecWarning(t *testing.T) {
	t.Parallel()

	env := setupReviewTestServer(t)
	defer env.server.Close()

	result := callReviewTool(t, env, "review_task_output", map[string]any{
		"task_id":        env.taskID,
		"output_summary": "Implemented authentication middleware",
	})
	if result.IsError {
		t.Fatalf("review_task_output returned transport error: %s", reviewResultText(t, result))
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(reviewResultText(t, result)), &parsed); err != nil {
		t.Fatalf("parse review_task_output result: %v\nraw: %s", err, reviewResultText(t, result))
	}

	// Feature has no spec linked, so we expect a no_spec warning.
	findings, _ := parsed["findings"].([]any)
	hasNoSpec := false
	for _, f := range findings {
		fm, _ := f.(map[string]any)
		if fm["type"] == "no_spec" {
			hasNoSpec = true
			if fm["severity"] != "warning" {
				t.Errorf("no_spec severity = %v, want warning", fm["severity"])
			}
		}
	}
	if !hasNoSpec {
		t.Error("expected no_spec warning finding when feature has no spec linked")
	}
}

func TestReviewTaskOutput_MCP_AlreadyDone_NoTransition(t *testing.T) {
	t.Parallel()

	env := setupReviewTestServer(t)
	defer env.server.Close()

	// Transition task to done: active → done.
	statusResult := callReviewTool(t, env, "update_status", map[string]any{
		"entity_type": "task",
		"id":          env.taskID,
		"status":      "done",
	})
	if statusResult.IsError {
		t.Fatalf("update_status to done returned error: %s", reviewResultText(t, statusResult))
	}

	// Run review on done task.
	result := callReviewTool(t, env, "review_task_output", map[string]any{
		"task_id":      env.taskID,
		"output_files": []any{"nonexistent/file.go"},
	})
	if result.IsError {
		t.Fatalf("review_task_output returned transport error: %s", reviewResultText(t, result))
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(reviewResultText(t, result)), &parsed); err != nil {
		t.Fatalf("parse review_task_output result: %v\nraw: %s", err, reviewResultText(t, result))
	}

	// Should still report findings.
	totalFindings, _ := parsed["total_findings"].(float64)
	if totalFindings == 0 {
		t.Error("expected findings on done task review, got 0")
	}

	// But task should still be done (no transition).
	taskResult, err := env.entitySvc.Get("task", env.taskID, "")
	if err != nil {
		t.Fatalf("Get task: %v", err)
	}
	taskStatus, _ := taskResult.State["status"].(string)
	if taskStatus != "done" {
		t.Errorf("task status = %q, want %q (no transition expected for done task)", taskStatus, "done")
	}
}
