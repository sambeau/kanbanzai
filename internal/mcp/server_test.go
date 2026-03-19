package mcp_test

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/mcptest"

	"kanbanzai/internal/document"
	kbzmcp "kanbanzai/internal/mcp"
	"kanbanzai/internal/service"
)

func setupTestServer(t *testing.T) *mcptest.Server {
	t.Helper()
	entityRoot := t.TempDir()
	docsRoot := t.TempDir()
	entitySvc := service.NewEntityService(entityRoot)
	docSvc := document.NewDocService(docsRoot)

	tools := append(kbzmcp.EntityTools(entitySvc), kbzmcp.DocumentTools(docSvc)...)
	ts, err := mcptest.NewServer(t, tools...)
	if err != nil {
		t.Fatalf("start test server: %v", err)
	}
	return ts
}

func callTool(t *testing.T, ts *mcptest.Server, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	ctx := context.Background()
	request := mcp.CallToolRequest{}
	request.Params.Name = name
	request.Params.Arguments = args
	result, err := ts.Client().CallTool(ctx, request)
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
	ts := setupTestServer(t)
	defer ts.Close()

	ctx := context.Background()
	listResult, err := ts.Client().ListTools(ctx, mcp.ListToolsRequest{})
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
	expectedDocTools := []string{
		"scaffold_document",
		"submit_document",
		"update_document_body",
		"approve_document",
		"retrieve_document",
		"list_documents",
		"validate_document",
	}

	expectedAll := append(expectedEntityTools, expectedDocTools...)
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
	ts := setupTestServer(t)
	defer ts.Close()

	result := callTool(t, ts, "create_epic", map[string]any{
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

	if !strings.HasPrefix(id, "E-") {
		t.Errorf("expected ID to start with E-, got %q", id)
	}
}

func TestServer_CreateAndGetEpic(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	createResult := callTool(t, ts, "create_epic", map[string]any{
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

	getResult := callTool(t, ts, "get_entity", map[string]any{
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

	state, ok := got["State"].(map[string]any)
	if !ok {
		t.Fatalf("expected State to be a map, got %T", got["State"])
	}
	if state["title"] != "Roundtrip Epic" {
		t.Errorf("state title = %v, want %q", state["title"], "Roundtrip Epic")
	}
}

func TestServer_UpdateStatus(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	createResult := callTool(t, ts, "create_epic", map[string]any{
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

	updateResult := callTool(t, ts, "update_status", map[string]any{
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

	state, ok := updated["State"].(map[string]any)
	if !ok {
		t.Fatalf("expected State to be a map, got %T", updated["State"])
	}
	if state["status"] != "approved" {
		t.Errorf("status = %v, want %q", state["status"], "approved")
	}
}

func TestServer_UpdateStatus_InvalidTransition(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	createResult := callTool(t, ts, "create_epic", map[string]any{
		"slug":       "invalid-transition-epic",
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
	updateResult := callTool(t, ts, "update_status", map[string]any{
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
	ts := setupTestServer(t)
	defer ts.Close()

	result := callTool(t, ts, "health_check", map[string]any{})

	if result.IsError {
		t.Fatalf("health_check returned error: %s", resultText(t, result))
	}

	text := resultText(t, result)
	if text == "" {
		t.Fatal("health_check returned empty result")
	}
}

func TestServer_DocumentLifecycle(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	// Step 1: Scaffold
	scaffoldResult := callTool(t, ts, "scaffold_document", map[string]any{
		"doc_type": "proposal",
		"title":    "Lifecycle Test Proposal",
	})

	if scaffoldResult.IsError {
		t.Fatalf("scaffold_document returned error: %s", resultText(t, scaffoldResult))
	}

	scaffoldText := resultText(t, scaffoldResult)
	if !strings.Contains(scaffoldText, "# Lifecycle Test Proposal") {
		t.Errorf("scaffold missing title heading, got:\n%s", scaffoldText)
	}

	// Step 2: Submit
	body := "# Lifecycle Test Proposal\n\n## Summary\n\nTest summary.\n\n## Problem\n\nTest problem.\n\n## Proposal\n\nTest proposal.\n"
	submitResult := callTool(t, ts, "submit_document", map[string]any{
		"doc_type":   "proposal",
		"title":      "Lifecycle Test Proposal",
		"body":       body,
		"created_by": "tester",
	})

	if submitResult.IsError {
		t.Fatalf("submit_document returned error: %s", resultText(t, submitResult))
	}

	var submitted map[string]any
	if err := json.Unmarshal([]byte(resultText(t, submitResult)), &submitted); err != nil {
		t.Fatalf("failed to parse submit result: %v", err)
	}

	docID := submitted["ID"].(string)

	// Step 3: Update body (normalise)
	updatedBody := "# Lifecycle Test Proposal\n\n## Summary\n\nNormalised summary.\n\n## Problem\n\nNormalised problem.\n\n## Proposal\n\nNormalised proposal.\n"
	updateResult := callTool(t, ts, "update_document_body", map[string]any{
		"doc_type": "proposal",
		"id":       docID,
		"body":     updatedBody,
	})

	if updateResult.IsError {
		t.Fatalf("update_document_body returned error: %s", resultText(t, updateResult))
	}

	// Step 4: Approve
	approveResult := callTool(t, ts, "approve_document", map[string]any{
		"doc_type":    "proposal",
		"id":          docID,
		"approved_by": "reviewer",
	})

	if approveResult.IsError {
		t.Fatalf("approve_document returned error: %s", resultText(t, approveResult))
	}

	// Step 5: Retrieve and verify verbatim round-trip
	retrieveResult := callTool(t, ts, "retrieve_document", map[string]any{
		"doc_type": "proposal",
		"id":       docID,
	})

	if retrieveResult.IsError {
		t.Fatalf("retrieve_document returned error: %s", resultText(t, retrieveResult))
	}

	retrievedBody := resultText(t, retrieveResult)
	if retrievedBody != updatedBody {
		t.Errorf("verbatim round-trip failed\nwant:\n%s\ngot:\n%s", updatedBody, retrievedBody)
	}
}

func TestServer_ListDocuments(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	body := "# List Test\n\n## Summary\n\nTest.\n\n## Problem\n\nTest.\n\n## Proposal\n\nTest.\n"
	submitResult := callTool(t, ts, "submit_document", map[string]any{
		"doc_type":   "proposal",
		"title":      "List Test Proposal",
		"body":       body,
		"created_by": "tester",
	})

	if submitResult.IsError {
		t.Fatalf("submit_document returned error: %s", resultText(t, submitResult))
	}

	var submitted map[string]any
	if err := json.Unmarshal([]byte(resultText(t, submitResult)), &submitted); err != nil {
		t.Fatalf("failed to parse submit result: %v", err)
	}
	docID := submitted["ID"].(string)

	listResult := callTool(t, ts, "list_documents", map[string]any{})

	if listResult.IsError {
		t.Fatalf("list_documents returned error: %s", resultText(t, listResult))
	}

	text := resultText(t, listResult)
	var docs []map[string]any
	if err := json.Unmarshal([]byte(text), &docs); err != nil {
		t.Fatalf("failed to parse list result: %v\nraw: %s", err, text)
	}

	if len(docs) == 0 {
		t.Fatal("list_documents returned no documents")
	}

	found := false
	for _, doc := range docs {
		if doc["ID"] == docID {
			found = true
			if doc["Title"] != "List Test Proposal" {
				t.Errorf("document title = %v, want %q", doc["Title"], "List Test Proposal")
			}
			break
		}
	}
	if !found {
		t.Errorf("submitted document %s not found in list results", docID)
	}
}

func TestServer_ScaffoldDocument(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	result := callTool(t, ts, "scaffold_document", map[string]any{
		"doc_type": "proposal",
		"title":    "My Test Proposal",
	})

	if result.IsError {
		t.Fatalf("scaffold_document returned error: %s", resultText(t, result))
	}

	text := resultText(t, result)

	// Proposal template requires: Summary, Problem, Proposal
	requiredSections := []string{
		"# My Test Proposal",
		"## Summary",
		"## Problem",
		"## Proposal",
	}

	for _, section := range requiredSections {
		if !strings.Contains(text, section) {
			t.Errorf("scaffold missing required section %q\ngot:\n%s", section, text)
		}
	}
}
