package mcp_test

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/mcptest"

	chk "kanbanzai/internal/checkpoint"
	kbzctx "kanbanzai/internal/context"
	kbzmcp "kanbanzai/internal/mcp"
	"kanbanzai/internal/service"
)

type testEnv struct {
	server *mcptest.Server
}

func setupTestServer(t *testing.T) *testEnv {
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
		t.Fatalf("start test server: %v", err)
	}
	return &testEnv{server: ts}
}

func callTool(t *testing.T, ts *testEnv, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	ctx := context.Background()
	request := mcp.CallToolRequest{}
	request.Params.Name = name
	request.Params.Arguments = args
	result, err := ts.server.Client().CallTool(ctx, request)
	if err != nil {
		t.Fatalf("CallTool(%s): %v", name, err)
	}
	return result
}

func resultText(t *testing.T, result *mcp.CallToolResult) string {
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

func TestServer_ListTools(t *testing.T) {
	env := setupTestServer(t)
	defer env.server.Close()

	ctx := context.Background()
	listResult, err := env.server.Client().ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	expectedEntityTools := []string{
		"create_epic",
		"create_feature",
		"create_task",
		"create_bug",
		"record_decision",
		"get_entity",
		"list_entities",
		"update_status",
		"update_entity",
		"validate_candidate",
		"health_check",
		"rebuild_cache",
	}
	expectedPhase4aTools := []string{
		"estimate_set",
		"estimate_query",
		"estimate_reference_add",
		"estimate_reference_remove",
		"work_queue",
		"dependency_status",
		"dispatch_task",
		"complete_task",
		"human_checkpoint",
		"human_checkpoint_respond",
		"human_checkpoint_get",
		"human_checkpoint_list",
	}

	expectedAll := append(expectedEntityTools, expectedPhase4aTools...)
	sort.Strings(expectedAll)

	var gotNames []string
	for _, tool := range listResult.Tools {
		gotNames = append(gotNames, tool.Name)
	}
	sort.Strings(gotNames)

	if len(gotNames) != len(expectedAll) {
		t.Fatalf("expected %d tools, got %d\nexpected: %v\ngot: %v",
			len(expectedAll), len(gotNames), expectedAll, gotNames)
	}

	for i := range expectedAll {
		if gotNames[i] != expectedAll[i] {
			t.Errorf("tool[%d]: expected %q, got %q", i, expectedAll[i], gotNames[i])
		}
	}
}

func TestServer_CreateEpic(t *testing.T) {
	env := setupTestServer(t)
	defer env.server.Close()

	result := callTool(t, env, "create_epic", map[string]any{
		"slug":       "test-epic",
		"title":      "Test Epic",
		"summary":    "A test epic for integration testing",
		"created_by": "tester",
	})

	if result.IsError {
		t.Fatalf("create_epic returned error: %s", resultText(t, result))
	}

	text := resultText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("failed to parse result JSON: %v\nraw: %s", err, text)
	}

	id, ok := parsed["ID"].(string)
	if !ok || id == "" {
		t.Fatalf("expected non-empty ID in result, got: %v", parsed["ID"])
	}

	if !strings.HasPrefix(id, "EPIC-") {
		t.Errorf("expected ID to start with EPIC-, got %q", id)
	}

	displayID, ok := parsed["DisplayID"].(string)
	if !ok || displayID == "" {
		t.Fatalf("expected non-empty DisplayID in result, got: %v", parsed["DisplayID"])
	}
	// Epic IDs pass through FormatFullDisplay unchanged
	if displayID != id {
		t.Errorf("DisplayID = %q, want %q (epic IDs should be unchanged)", displayID, id)
	}
}

func TestServer_CreateAndGetEpic(t *testing.T) {
	env := setupTestServer(t)
	defer env.server.Close()

	createResult := callTool(t, env, "create_epic", map[string]any{
		"slug":       "roundtrip-epic",
		"title":      "Roundtrip Epic",
		"summary":    "Test create and get roundtrip",
		"created_by": "tester",
	})

	if createResult.IsError {
		t.Fatalf("create_epic returned error: %s", resultText(t, createResult))
	}

	var created map[string]any
	if err := json.Unmarshal([]byte(resultText(t, createResult)), &created); err != nil {
		t.Fatalf("failed to parse create result: %v", err)
	}

	epicID := created["ID"].(string)
	epicSlug := created["Slug"].(string)

	if _, ok := created["DisplayID"].(string); !ok {
		t.Errorf("create result missing DisplayID field")
	}

	getResult := callTool(t, env, "get_entity", map[string]any{
		"entity_type": "epic",
		"id":          epicID,
		"slug":        epicSlug,
	})

	if getResult.IsError {
		t.Fatalf("get_entity returned error: %s", resultText(t, getResult))
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(resultText(t, getResult)), &got); err != nil {
		t.Fatalf("failed to parse get result: %v", err)
	}

	if got["ID"] != epicID {
		t.Errorf("get ID = %v, want %v", got["ID"], epicID)
	}
	if got["Slug"] != epicSlug {
		t.Errorf("get Slug = %v, want %v", got["Slug"], epicSlug)
	}
	if _, ok := got["DisplayID"].(string); !ok {
		t.Errorf("get result missing DisplayID field")
	}

	state, ok := got["State"].(map[string]any)
	if !ok {
		t.Fatalf("expected State to be a map, got %T", got["State"])
	}
	if state["title"] != "Roundtrip Epic" {
		t.Errorf("state title = %v, want %q", state["title"], "Roundtrip Epic")
	}
}

func TestServer_UpdateStatus(t *testing.T) {
	env := setupTestServer(t)
	defer env.server.Close()

	createResult := callTool(t, env, "create_epic", map[string]any{
		"slug":       "status-epic",
		"title":      "Status Epic",
		"summary":    "Test status update",
		"created_by": "tester",
	})

	if createResult.IsError {
		t.Fatalf("create_epic returned error: %s", resultText(t, createResult))
	}

	var created map[string]any
	if err := json.Unmarshal([]byte(resultText(t, createResult)), &created); err != nil {
		t.Fatalf("failed to parse create result: %v", err)
	}

	epicID := created["ID"].(string)
	epicSlug := created["Slug"].(string)

	updateResult := callTool(t, env, "update_status", map[string]any{
		"entity_type": "epic",
		"id":          epicID,
		"slug":        epicSlug,
		"status":      "approved",
	})

	if updateResult.IsError {
		t.Fatalf("update_status returned error: %s", resultText(t, updateResult))
	}

	var updated map[string]any
	if err := json.Unmarshal([]byte(resultText(t, updateResult)), &updated); err != nil {
		t.Fatalf("failed to parse update result: %v", err)
	}

	if _, ok := updated["DisplayID"].(string); !ok {
		t.Errorf("update_status result missing DisplayID field")
	}

	state, ok := updated["State"].(map[string]any)
	if !ok {
		t.Fatalf("expected State to be a map, got %T", updated["State"])
	}
	if state["status"] != "approved" {
		t.Errorf("status = %v, want %q", state["status"], "approved")
	}
}

func TestServer_UpdateStatus_InvalidTransition(t *testing.T) {
	env := setupTestServer(t)
	defer env.server.Close()

	createResult := callTool(t, env, "create_epic", map[string]any{
		"slug":       "inv-trans-epic",
		"title":      "Invalid Transition Epic",
		"summary":    "Test invalid status transition",
		"created_by": "tester",
	})

	if createResult.IsError {
		t.Fatalf("create_epic returned error: %s", resultText(t, createResult))
	}

	var created map[string]any
	if err := json.Unmarshal([]byte(resultText(t, createResult)), &created); err != nil {
		t.Fatalf("failed to parse create result: %v", err)
	}

	epicID := created["ID"].(string)
	epicSlug := created["Slug"].(string)

	// proposed -> done is not a valid transition (must go proposed -> approved -> active -> done)
	updateResult := callTool(t, env, "update_status", map[string]any{
		"entity_type": "epic",
		"id":          epicID,
		"slug":        epicSlug,
		"status":      "done",
	})

	if !updateResult.IsError {
		t.Fatalf("expected update_status to return error for invalid transition, got success: %s",
			resultText(t, updateResult))
	}
}

func TestServer_HealthCheck(t *testing.T) {
	env := setupTestServer(t)
	defer env.server.Close()

	result := callTool(t, env, "health_check", map[string]any{})

	if result.IsError {
		t.Fatalf("health_check returned error: %s", resultText(t, result))
	}

	text := resultText(t, result)
	if text == "" {
		t.Fatal("health_check returned empty result")
	}
}

func TestServer_GetEntityWithoutSlug(t *testing.T) {
	env := setupTestServer(t)
	defer env.server.Close()

	// Create an epic first
	createResult := callTool(t, env, "create_epic", map[string]any{
		"slug":       "prefix-test-epic",
		"title":      "Prefix Test Epic",
		"summary":    "Test get without slug",
		"created_by": "tester",
	})

	if createResult.IsError {
		t.Fatalf("create_epic returned error: %s", resultText(t, createResult))
	}

	var created map[string]any
	if err := json.Unmarshal([]byte(resultText(t, createResult)), &created); err != nil {
		t.Fatalf("failed to parse create result: %v", err)
	}

	epicID := created["ID"].(string)
	epicSlug := created["Slug"].(string)

	// Get entity using only entity_type and id (no slug) — prefix resolution
	getResult := callTool(t, env, "get_entity", map[string]any{
		"entity_type": "epic",
		"id":          epicID,
	})

	if getResult.IsError {
		t.Fatalf("get_entity without slug returned error: %s", resultText(t, getResult))
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(resultText(t, getResult)), &got); err != nil {
		t.Fatalf("failed to parse get result: %v", err)
	}

	if got["ID"] != epicID {
		t.Errorf("get ID = %v, want %v", got["ID"], epicID)
	}
	if got["Slug"] != epicSlug {
		t.Errorf("get Slug = %v, want %v", got["Slug"], epicSlug)
	}

	// Also verify a prefix of the ID works
	prefix := epicID[:7]
	prefixResult := callTool(t, env, "get_entity", map[string]any{
		"entity_type": "epic",
		"id":          prefix,
	})

	if prefixResult.IsError {
		t.Fatalf("get_entity with prefix %q returned error: %s", prefix, resultText(t, prefixResult))
	}

	var gotPrefix map[string]any
	if err := json.Unmarshal([]byte(resultText(t, prefixResult)), &gotPrefix); err != nil {
		t.Fatalf("failed to parse prefix get result: %v", err)
	}

	if gotPrefix["ID"] != epicID {
		t.Errorf("prefix get ID = %v, want %v", gotPrefix["ID"], epicID)
	}
	if _, ok := gotPrefix["DisplayID"].(string); !ok {
		t.Errorf("prefix get result missing DisplayID field")
	}
}
