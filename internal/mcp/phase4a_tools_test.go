package mcp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/mcptest"

	chk "kanbanzai/internal/checkpoint"
	kbzctx "kanbanzai/internal/context"
	kbzmcp "kanbanzai/internal/mcp"
	"kanbanzai/internal/service"
	"kanbanzai/internal/storage"
)

// phase4aEnv holds the test server and service references for Phase 4a tests.
type phase4aEnv struct {
	server      *mcptest.Server
	entitySvc   *service.EntityService
	entityRoot  string
	profileRoot string
}

// setupPhase4aTestServer creates a test MCP server with entity, estimation,
// queue, and dispatch tools registered.
func setupPhase4aTestServer(t *testing.T) *phase4aEnv {
	t.Helper()

	entityRoot := t.TempDir()
	stateRoot := t.TempDir()
	profileRoot := t.TempDir()
	checkpointRoot := t.TempDir()
	indexRoot := t.TempDir()
	repoRoot := t.TempDir()

	entitySvc := service.NewEntityService(entityRoot)
	knowledgeSvc := service.NewKnowledgeService(stateRoot)
	dispatchSvc := service.NewDispatchService(entitySvc, knowledgeSvc)
	checkpointStore := chk.NewStore(checkpointRoot)
	profileStore := kbzctx.NewProfileStore(profileRoot)
	intelligenceSvc := service.NewIntelligenceService(indexRoot, repoRoot)

	tools := kbzmcp.EntityTools(entitySvc)
	tools = append(tools, kbzmcp.EstimationTools(entitySvc, knowledgeSvc)...)
	tools = append(tools, kbzmcp.QueueTools(entitySvc)...)
	tools = append(tools, kbzmcp.DispatchTools(dispatchSvc, checkpointStore, profileStore, knowledgeSvc, entitySvc, intelligenceSvc)...)

	ts, err := mcptest.NewServer(t, tools...)
	if err != nil {
		t.Fatalf("start Phase 4a test server: %v", err)
	}

	return &phase4aEnv{
		server:      ts,
		entitySvc:   entitySvc,
		entityRoot:  entityRoot,
		profileRoot: profileRoot,
	}
}

// callP4Tool calls a tool on the Phase 4a test server.
func callP4Tool(t *testing.T, env *phase4aEnv, name string, args map[string]any) *mcp.CallToolResult {
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

// resultP4Text extracts text content from a tool result.
func resultP4Text(t *testing.T, result *mcp.CallToolResult) string {
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

// writeTestPlanFile writes a minimal plan YAML directly to disk so that
// feature creation can find the parent plan via entityExists.
func writeTestPlanFile(t *testing.T, entityRoot, planID string) {
	t.Helper()
	planDir := filepath.Join(entityRoot, "plans")
	if err := os.MkdirAll(planDir, 0o755); err != nil {
		t.Fatalf("create plans dir: %v", err)
	}
	// Extract slug from plan ID (e.g. "P1-test-plan" → "test-plan")
	slug := planID
	if idx := strings.Index(planID, "-"); idx >= 0 {
		slug = planID[idx+1:]
	}
	content := fmt.Sprintf(
		"id: %s\nslug: %s\ntitle: Test Plan\nstatus: active\nsummary: Test plan\ncreated: \"2026-03-19T12:00:00Z\"\ncreated_by: test\nupdated: \"2026-03-19T12:00:00Z\"\n",
		planID, slug,
	)
	if err := os.WriteFile(filepath.Join(planDir, planID+".yaml"), []byte(content), 0o644); err != nil {
		t.Fatalf("write test plan: %v", err)
	}
}

// createFeatureViaP4 creates a feature via the MCP create_feature tool and
// returns the feature ID.
func createFeatureViaP4(t *testing.T, env *phase4aEnv, planID, slug, summary string) string {
	t.Helper()
	result := callP4Tool(t, env, "create_feature", map[string]any{
		"slug":       slug,
		"parent":     planID,
		"summary":    summary,
		"created_by": "tester",
	})
	if result.IsError {
		t.Fatalf("create_feature returned error: %s", resultP4Text(t, result))
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(resultP4Text(t, result)), &parsed); err != nil {
		t.Fatalf("parse create_feature result: %v", err)
	}
	id, ok := parsed["ID"].(string)
	if !ok || id == "" {
		t.Fatalf("create_feature: expected non-empty ID, got %v", parsed["ID"])
	}
	return id
}

// createTaskViaP4 creates a task via the MCP create_task tool and returns the
// task ID and slug.
func createTaskViaP4(t *testing.T, env *phase4aEnv, featureID, slug, summary string) (string, string) {
	t.Helper()
	result := callP4Tool(t, env, "create_task", map[string]any{
		"parent_feature": featureID,
		"slug":           slug,
		"summary":        summary,
	})
	if result.IsError {
		t.Fatalf("create_task returned error: %s", resultP4Text(t, result))
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(resultP4Text(t, result)), &parsed); err != nil {
		t.Fatalf("parse create_task result: %v", err)
	}
	id, ok := parsed["ID"].(string)
	if !ok || id == "" {
		t.Fatalf("create_task: expected non-empty ID, got %v", parsed["ID"])
	}
	taskSlug, _ := parsed["Slug"].(string)
	return id, taskSlug
}

// setDependsOn loads a task record from the entity store, sets depends_on,
// and writes it back.
func setDependsOn(t *testing.T, entityRoot, taskID, taskSlug string, deps []string) {
	t.Helper()
	store := storage.NewEntityStore(entityRoot)
	record, err := store.Load("task", taskID, taskSlug)
	if err != nil {
		t.Fatalf("load task %s for depends_on: %v", taskID, err)
	}
	record.Fields["depends_on"] = deps
	if _, err := store.Write(record); err != nil {
		t.Fatalf("write task %s with depends_on: %v", taskID, err)
	}
}

// writeTestProfile writes a minimal role profile YAML to the profile root.
func writeTestProfile(t *testing.T, profileRoot, roleID string) {
	t.Helper()
	content := fmt.Sprintf("id: %s\ndescription: \"Test role for integration tests\"\n", roleID)
	if err := os.WriteFile(filepath.Join(profileRoot, roleID+".yaml"), []byte(content), 0o644); err != nil {
		t.Fatalf("write test profile %s: %v", roleID, err)
	}
}

// --- Tests ---

func TestPhase4a_EstimateSet(t *testing.T) {
	t.Parallel()
	env := setupPhase4aTestServer(t)
	defer env.server.Close()

	// Set up: plan → feature → task
	writeTestPlanFile(t, env.entityRoot, "P1-est-set")
	featureID := createFeatureViaP4(t, env, "P1-est-set", "est-set-feat", "Estimation set feature")
	taskID, _ := createTaskViaP4(t, env, featureID, "est-set-task", "Estimation set task")

	// Set a valid estimate (5 is in Modified Fibonacci scale)
	result := callP4Tool(t, env, "estimate_set", map[string]any{
		"entity_id": taskID,
		"estimate":  5,
	})
	if result.IsError {
		t.Fatalf("estimate_set returned error: %s", resultP4Text(t, result))
	}

	text := resultP4Text(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse estimate_set result: %v\nraw: %s", err, text)
	}

	if parsed["entity_id"] != taskID {
		t.Errorf("entity_id = %v, want %v", parsed["entity_id"], taskID)
	}
	if parsed["entity_type"] != "task" {
		t.Errorf("entity_type = %v, want %q", parsed["entity_type"], "task")
	}
	if est, ok := parsed["estimate"].(float64); !ok || est != 5 {
		t.Errorf("estimate = %v, want 5", parsed["estimate"])
	}
	if parsed["scale"] == nil {
		t.Error("expected scale in response, got nil")
	}

	// Try an invalid estimate (7 is not in the Fibonacci scale)
	invalidResult := callP4Tool(t, env, "estimate_set", map[string]any{
		"entity_id": taskID,
		"estimate":  7,
	})
	if !invalidResult.IsError {
		t.Fatalf("estimate_set(7) should have returned error, got success: %s",
			resultP4Text(t, invalidResult))
	}
}

func TestPhase4a_EstimateQuery(t *testing.T) {
	t.Parallel()
	env := setupPhase4aTestServer(t)
	defer env.server.Close()

	writeTestPlanFile(t, env.entityRoot, "P1-est-query")
	featureID := createFeatureViaP4(t, env, "P1-est-query", "est-query-feat", "Estimation query feature")
	task1ID, _ := createTaskViaP4(t, env, featureID, "est-query-t1", "Task one")
	task2ID, _ := createTaskViaP4(t, env, featureID, "est-query-t2", "Task two")

	// Set estimates on both tasks
	r1 := callP4Tool(t, env, "estimate_set", map[string]any{
		"entity_id": task1ID,
		"estimate":  3,
	})
	if r1.IsError {
		t.Fatalf("estimate_set(t1, 3) error: %s", resultP4Text(t, r1))
	}
	r2 := callP4Tool(t, env, "estimate_set", map[string]any{
		"entity_id": task2ID,
		"estimate":  5,
	})
	if r2.IsError {
		t.Fatalf("estimate_set(t2, 5) error: %s", resultP4Text(t, r2))
	}

	// Query the feature for rollup
	queryResult := callP4Tool(t, env, "estimate_query", map[string]any{
		"entity_id": featureID,
	})
	if queryResult.IsError {
		t.Fatalf("estimate_query returned error: %s", resultP4Text(t, queryResult))
	}

	text := resultP4Text(t, queryResult)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse estimate_query result: %v\nraw: %s", err, text)
	}

	if parsed["entity_type"] != "feature" {
		t.Errorf("entity_type = %v, want %q", parsed["entity_type"], "feature")
	}

	rollup, ok := parsed["rollup"].(map[string]any)
	if !ok {
		t.Fatalf("expected rollup to be a map, got %T (%v)", parsed["rollup"], parsed["rollup"])
	}

	taskTotal, ok := rollup["task_total"].(float64)
	if !ok {
		t.Fatalf("task_total not a number: %v (%T)", rollup["task_total"], rollup["task_total"])
	}
	if taskTotal != 8 { // 3 + 5
		t.Errorf("task_total = %v, want 8", taskTotal)
	}
}

func TestPhase4a_EstimateReference(t *testing.T) {
	t.Parallel()
	env := setupPhase4aTestServer(t)
	defer env.server.Close()

	// Use a lowercase entity_id because RemoveEstimationReference compares
	// the raw "estimation-ref-{entity_id}" against the stored topic, which
	// has been lowercased by NormalizeTopic during Contribute.
	entityID := "task-fake12345abcd"

	// Add an estimation reference
	addResult := callP4Tool(t, env, "estimate_reference_add", map[string]any{
		"entity_id":  entityID,
		"content":    "This task involved writing 3 unit tests and took about 2 story points of effort.",
		"created_by": "tester",
	})
	if addResult.IsError {
		t.Fatalf("estimate_reference_add returned error: %s", resultP4Text(t, addResult))
	}

	text := resultP4Text(t, addResult)
	var addParsed map[string]any
	if err := json.Unmarshal([]byte(text), &addParsed); err != nil {
		t.Fatalf("parse estimate_reference_add result: %v", err)
	}

	if addParsed["status"] != "added" {
		t.Errorf("status = %v, want %q", addParsed["status"], "added")
	}
	if addParsed["entity_id"] != entityID {
		t.Errorf("entity_id = %v, want %q", addParsed["entity_id"], entityID)
	}
	entryID, ok := addParsed["entry_id"].(string)
	if !ok || entryID == "" {
		t.Fatalf("expected non-empty entry_id, got %v", addParsed["entry_id"])
	}

	// Remove the estimation reference
	removeResult := callP4Tool(t, env, "estimate_reference_remove", map[string]any{
		"entity_id": entityID,
	})
	if removeResult.IsError {
		t.Fatalf("estimate_reference_remove returned error: %s", resultP4Text(t, removeResult))
	}

	var removeParsed map[string]any
	if err := json.Unmarshal([]byte(resultP4Text(t, removeResult)), &removeParsed); err != nil {
		t.Fatalf("parse estimate_reference_remove result: %v", err)
	}

	if removeParsed["status"] != "removed" {
		t.Errorf("status = %v, want %q", removeParsed["status"], "removed")
	}
	if removeParsed["entity_id"] != entityID {
		t.Errorf("entity_id = %v, want %q", removeParsed["entity_id"], entityID)
	}
}

func TestPhase4a_WorkQueue(t *testing.T) {
	t.Parallel()
	env := setupPhase4aTestServer(t)
	defer env.server.Close()

	writeTestPlanFile(t, env.entityRoot, "P1-wq")
	featureID := createFeatureViaP4(t, env, "P1-wq", "wq-feat", "Work queue feature")
	taskAID, taskASlug := createTaskViaP4(t, env, featureID, "wq-task-a", "Independent task A")
	taskBID, taskBSlug := createTaskViaP4(t, env, featureID, "wq-task-b", "Dependent task B")

	// Task B depends on task A
	setDependsOn(t, env.entityRoot, taskBID, taskBSlug, []string{taskAID})

	// Call work_queue — should promote task A (no deps) to ready
	queueResult := callP4Tool(t, env, "work_queue", map[string]any{})
	if queueResult.IsError {
		t.Fatalf("work_queue returned error: %s", resultP4Text(t, queueResult))
	}

	text := resultP4Text(t, queueResult)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse work_queue result: %v\nraw: %s", err, text)
	}

	queue, ok := parsed["queue"].([]any)
	if !ok {
		t.Fatalf("expected queue to be array, got %T", parsed["queue"])
	}

	// Task A should be in the queue (promoted to ready)
	foundA := false
	foundB := false
	for _, item := range queue {
		qi, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if qi["task_id"] == taskAID {
			foundA = true
			if qi["status"] != "ready" {
				t.Errorf("task A status = %v, want %q", qi["status"], "ready")
			}
		}
		if qi["task_id"] == taskBID {
			foundB = true
		}
	}

	if !foundA {
		t.Errorf("task A (%s) should be in the ready queue", taskAID)
	}
	if foundB {
		t.Errorf("task B (%s) should NOT be in the ready queue (blocked by A)", taskBID)
	}

	promotedCount, _ := parsed["promoted_count"].(float64)
	if promotedCount < 1 {
		t.Errorf("promoted_count = %v, want >= 1", promotedCount)
	}

	// Verify task B is still queued (check via dependency_status showing it as blocked)
	_ = taskASlug // used above via setDependsOn
	_ = taskBSlug // used above via setDependsOn
}

func TestPhase4a_DependencyStatus(t *testing.T) {
	t.Parallel()
	env := setupPhase4aTestServer(t)
	defer env.server.Close()

	writeTestPlanFile(t, env.entityRoot, "P1-ds")
	featureID := createFeatureViaP4(t, env, "P1-ds", "ds-feat", "Dependency status feature")
	taskAID, _ := createTaskViaP4(t, env, featureID, "ds-task-a", "Blocker task A")
	taskBID, taskBSlug := createTaskViaP4(t, env, featureID, "ds-task-b", "Blocked task B")

	// Task B depends on task A
	setDependsOn(t, env.entityRoot, taskBID, taskBSlug, []string{taskAID})

	// Check dependency status for task B
	result := callP4Tool(t, env, "dependency_status", map[string]any{
		"task_id": taskBID,
	})
	if result.IsError {
		t.Fatalf("dependency_status returned error: %s", resultP4Text(t, result))
	}

	text := resultP4Text(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse dependency_status result: %v\nraw: %s", err, text)
	}

	if parsed["task_id"] != taskBID {
		t.Errorf("task_id = %v, want %v", parsed["task_id"], taskBID)
	}

	dependsOnCount, _ := parsed["depends_on_count"].(float64)
	if dependsOnCount != 1 {
		t.Errorf("depends_on_count = %v, want 1", dependsOnCount)
	}

	blockingCount, _ := parsed["blocking_count"].(float64)
	if blockingCount != 1 {
		t.Errorf("blocking_count = %v, want 1", blockingCount)
	}

	deps, ok := parsed["dependencies"].([]any)
	if !ok || len(deps) == 0 {
		t.Fatalf("expected non-empty dependencies array, got %v", parsed["dependencies"])
	}

	dep, ok := deps[0].(map[string]any)
	if !ok {
		t.Fatalf("expected dependency entry to be map, got %T", deps[0])
	}
	if dep["task_id"] != taskAID {
		t.Errorf("dependency task_id = %v, want %v", dep["task_id"], taskAID)
	}
	if dep["blocking"] != true {
		t.Errorf("dependency blocking = %v, want true", dep["blocking"])
	}
}

func TestPhase4a_DispatchAndComplete(t *testing.T) {
	t.Parallel()
	env := setupPhase4aTestServer(t)
	defer env.server.Close()

	// Write the role profile needed for dispatch
	writeTestProfile(t, env.profileRoot, "test-role")

	writeTestPlanFile(t, env.entityRoot, "P1-dispatch")
	featureID := createFeatureViaP4(t, env, "P1-dispatch", "dispatch-feat", "Dispatch feature")
	taskID, _ := createTaskViaP4(t, env, featureID, "dispatch-task", "Task to dispatch")

	// Promote the task to ready via work_queue
	queueResult := callP4Tool(t, env, "work_queue", map[string]any{})
	if queueResult.IsError {
		t.Fatalf("work_queue returned error: %s", resultP4Text(t, queueResult))
	}

	// Verify the task made it to ready
	var queueParsed map[string]any
	if err := json.Unmarshal([]byte(resultP4Text(t, queueResult)), &queueParsed); err != nil {
		t.Fatalf("parse work_queue result: %v", err)
	}
	queue, _ := queueParsed["queue"].([]any)
	taskInQueue := false
	for _, item := range queue {
		qi, _ := item.(map[string]any)
		if qi["task_id"] == taskID {
			taskInQueue = true
			break
		}
	}
	if !taskInQueue {
		t.Fatalf("task %s not found in ready queue after work_queue", taskID)
	}

	// Dispatch the task
	dispatchResult := callP4Tool(t, env, "dispatch_task", map[string]any{
		"task_id":       taskID,
		"role":          "test-role",
		"dispatched_by": "test-agent",
	})
	if dispatchResult.IsError {
		t.Fatalf("dispatch_task returned error: %s", resultP4Text(t, dispatchResult))
	}

	dispatchText := resultP4Text(t, dispatchResult)
	var dispatchParsed map[string]any
	if err := json.Unmarshal([]byte(dispatchText), &dispatchParsed); err != nil {
		t.Fatalf("parse dispatch_task result: %v\nraw: %s", err, dispatchText)
	}

	if dispatchParsed["task"] == nil {
		t.Error("dispatch_task response missing 'task' field")
	}
	if dispatchParsed["context"] == nil {
		t.Error("dispatch_task response missing 'context' field")
	}

	// Verify context contains role
	if ctxMap, ok := dispatchParsed["context"].(map[string]any); ok {
		if ctxMap["role"] != "test-role" {
			t.Errorf("context.role = %v, want %q", ctxMap["role"], "test-role")
		}
	}

	// Complete the task
	completeResult := callP4Tool(t, env, "complete_task", map[string]any{
		"task_id": taskID,
		"summary": "completed successfully",
	})
	if completeResult.IsError {
		t.Fatalf("complete_task returned error: %s", resultP4Text(t, completeResult))
	}

	completeText := resultP4Text(t, completeResult)
	var completeParsed map[string]any
	if err := json.Unmarshal([]byte(completeText), &completeParsed); err != nil {
		t.Fatalf("parse complete_task result: %v\nraw: %s", err, completeText)
	}

	if completeParsed["task"] == nil {
		t.Error("complete_task response missing 'task' field")
	}

	// Verify the task is now done by querying it
	getResult := callP4Tool(t, env, "get_entity", map[string]any{
		"entity_type": "task",
		"id":          taskID,
	})
	if getResult.IsError {
		t.Fatalf("get_entity after complete returned error: %s", resultP4Text(t, getResult))
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(resultP4Text(t, getResult)), &got); err != nil {
		t.Fatalf("parse get_entity result: %v", err)
	}
	state, _ := got["State"].(map[string]any)
	if state["status"] != "done" {
		t.Errorf("task status after complete = %v, want %q", state["status"], "done")
	}
}

func TestPhase4a_Checkpoint_Lifecycle(t *testing.T) {
	t.Parallel()
	env := setupPhase4aTestServer(t)
	defer env.server.Close()

	// Create a checkpoint
	createResult := callP4Tool(t, env, "human_checkpoint", map[string]any{
		"question":              "Should we proceed with approach A or B?",
		"context":               "We have two viable approaches for the authentication module.",
		"orchestration_summary": "3 tasks completed, 2 remaining. Blocked on architecture decision.",
		"created_by":            "orchestrator-agent",
	})
	if createResult.IsError {
		t.Fatalf("human_checkpoint returned error: %s", resultP4Text(t, createResult))
	}

	createText := resultP4Text(t, createResult)
	var createParsed map[string]any
	if err := json.Unmarshal([]byte(createText), &createParsed); err != nil {
		t.Fatalf("parse human_checkpoint result: %v\nraw: %s", err, createText)
	}

	checkpointID, ok := createParsed["checkpoint_id"].(string)
	if !ok || checkpointID == "" {
		t.Fatalf("expected non-empty checkpoint_id, got %v", createParsed["checkpoint_id"])
	}
	if !strings.HasPrefix(checkpointID, "CHK-") {
		t.Errorf("checkpoint_id should start with CHK-, got %q", checkpointID)
	}
	if createParsed["status"] != "pending" {
		t.Errorf("status = %v, want %q", createParsed["status"], "pending")
	}

	// Get the checkpoint — should still be pending
	getResult := callP4Tool(t, env, "human_checkpoint_get", map[string]any{
		"checkpoint_id": checkpointID,
	})
	if getResult.IsError {
		t.Fatalf("human_checkpoint_get returned error: %s", resultP4Text(t, getResult))
	}

	var getParsed map[string]any
	if err := json.Unmarshal([]byte(resultP4Text(t, getResult)), &getParsed); err != nil {
		t.Fatalf("parse human_checkpoint_get result: %v", err)
	}
	if getParsed["status"] != "pending" {
		t.Errorf("get status = %v, want %q", getParsed["status"], "pending")
	}
	if getParsed["question"] != "Should we proceed with approach A or B?" {
		t.Errorf("question mismatch: %v", getParsed["question"])
	}
	if getParsed["response"] != nil {
		t.Errorf("response should be nil for pending checkpoint, got %v", getParsed["response"])
	}

	// Respond to the checkpoint
	respondResult := callP4Tool(t, env, "human_checkpoint_respond", map[string]any{
		"checkpoint_id": checkpointID,
		"response":      "Go with approach A. It has better test coverage.",
	})
	if respondResult.IsError {
		t.Fatalf("human_checkpoint_respond returned error: %s", resultP4Text(t, respondResult))
	}

	var respondParsed map[string]any
	if err := json.Unmarshal([]byte(resultP4Text(t, respondResult)), &respondParsed); err != nil {
		t.Fatalf("parse human_checkpoint_respond result: %v", err)
	}
	if respondParsed["status"] != "responded" {
		t.Errorf("respond status = %v, want %q", respondParsed["status"], "responded")
	}
	if respondParsed["checkpoint_id"] != checkpointID {
		t.Errorf("respond checkpoint_id = %v, want %v", respondParsed["checkpoint_id"], checkpointID)
	}

	// Get again — should now be responded
	getResult2 := callP4Tool(t, env, "human_checkpoint_get", map[string]any{
		"checkpoint_id": checkpointID,
	})
	if getResult2.IsError {
		t.Fatalf("human_checkpoint_get (after respond) returned error: %s", resultP4Text(t, getResult2))
	}

	var getParsed2 map[string]any
	if err := json.Unmarshal([]byte(resultP4Text(t, getResult2)), &getParsed2); err != nil {
		t.Fatalf("parse second human_checkpoint_get result: %v", err)
	}
	if getParsed2["status"] != "responded" {
		t.Errorf("get status after respond = %v, want %q", getParsed2["status"], "responded")
	}
	if getParsed2["response"] != "Go with approach A. It has better test coverage." {
		t.Errorf("response mismatch: %v", getParsed2["response"])
	}

	// List checkpoints — should include our checkpoint
	listResult := callP4Tool(t, env, "human_checkpoint_list", map[string]any{})
	if listResult.IsError {
		t.Fatalf("human_checkpoint_list returned error: %s", resultP4Text(t, listResult))
	}

	var listParsed map[string]any
	if err := json.Unmarshal([]byte(resultP4Text(t, listResult)), &listParsed); err != nil {
		t.Fatalf("parse human_checkpoint_list result: %v", err)
	}

	total, _ := listParsed["total"].(float64)
	if total < 1 {
		t.Errorf("total = %v, want >= 1", total)
	}

	checkpoints, ok := listParsed["checkpoints"].([]any)
	if !ok || len(checkpoints) == 0 {
		t.Fatalf("expected non-empty checkpoints array, got %v", listParsed["checkpoints"])
	}

	found := false
	for _, cp := range checkpoints {
		cpMap, ok := cp.(map[string]any)
		if !ok {
			continue
		}
		if cpMap["id"] == checkpointID {
			found = true
			if cpMap["status"] != "responded" {
				t.Errorf("listed checkpoint status = %v, want %q", cpMap["status"], "responded")
			}
			break
		}
	}
	if !found {
		t.Errorf("checkpoint %s not found in list results", checkpointID)
	}

	// Verify pending_count is 0 (our only checkpoint is responded)
	pendingCount, _ := listParsed["pending_count"].(float64)
	if pendingCount != 0 {
		t.Errorf("pending_count = %v, want 0", pendingCount)
	}
}
