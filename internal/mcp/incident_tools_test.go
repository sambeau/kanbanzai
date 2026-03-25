package mcp_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/mcptest"

	kbzmcp "kanbanzai/internal/mcp"
	"kanbanzai/internal/service"
)

// incidentEnv holds the test server and service references for incident tests.
type incidentEnv struct {
	server     *mcptest.Server
	entitySvc  *service.EntityService
	entityRoot string
}

// setupIncidentTestServer creates a test MCP server with entity and incident
// tools registered.
func setupIncidentTestServer(t *testing.T) *incidentEnv {
	t.Helper()

	entityRoot := t.TempDir()
	entitySvc := service.NewEntityService(entityRoot)

	tools := kbzmcp.EntityTools(entitySvc)
	tools = append(tools, kbzmcp.IncidentTools(entitySvc)...)

	ts, err := mcptest.NewServer(t, tools...)
	if err != nil {
		t.Fatalf("start incident test server: %v", err)
	}

	return &incidentEnv{
		server:     ts,
		entitySvc:  entitySvc,
		entityRoot: entityRoot,
	}
}

// callIncTool calls a tool on the incident test server.
func callIncTool(t *testing.T, env *incidentEnv, name string, args map[string]any) *mcp.CallToolResult {
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

// resultIncText extracts text content from a tool result.
func resultIncText(t *testing.T, result *mcp.CallToolResult) string {
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

// createIncidentViaTools creates an incident via the MCP tool and returns
// the parsed response map.
func createIncidentViaTools(t *testing.T, env *incidentEnv, slug, title, severity, summary string) map[string]any {
	t.Helper()
	result := callIncTool(t, env, "incident_create", map[string]any{
		"slug":        slug,
		"title":       title,
		"severity":    severity,
		"summary":     summary,
		"reported_by": "tester",
	})
	if result.IsError {
		t.Fatalf("incident_create returned error: %s", resultIncText(t, result))
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(resultIncText(t, result)), &parsed); err != nil {
		t.Fatalf("parse incident_create result: %v\nraw: %s", err, resultIncText(t, result))
	}
	return parsed
}

func TestIncidentCreate(t *testing.T) {
	t.Parallel()
	env := setupIncidentTestServer(t)
	defer env.server.Close()

	result := callIncTool(t, env, "incident_create", map[string]any{
		"slug":        "api-outage",
		"title":       "API Gateway Down",
		"severity":    "high",
		"summary":     "API gateway returning 503 for all requests",
		"reported_by": "oncall-eng",
	})
	if result.IsError {
		t.Fatalf("incident_create returned error: %s", resultIncText(t, result))
	}

	text := resultIncText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse incident_create result: %v\nraw: %s", err, text)
	}

	id, ok := parsed["ID"].(string)
	if !ok || id == "" {
		t.Fatalf("expected non-empty ID, got %v", parsed["ID"])
	}
	if !strings.HasPrefix(id, "INC-") {
		t.Errorf("expected ID to start with INC-, got %q", id)
	}

	state, ok := parsed["State"].(map[string]any)
	if !ok {
		t.Fatalf("expected State map, got %T", parsed["State"])
	}
	if state["status"] != "reported" {
		t.Errorf("status = %v, want %q", state["status"], "reported")
	}
	if state["severity"] != "high" {
		t.Errorf("severity = %v, want %q", state["severity"], "high")
	}
}

func TestIncidentCreate_InvalidSeverity(t *testing.T) {
	t.Parallel()
	env := setupIncidentTestServer(t)
	defer env.server.Close()

	result := callIncTool(t, env, "incident_create", map[string]any{
		"slug":        "bad-sev",
		"title":       "Bad severity incident",
		"severity":    "extreme",
		"summary":     "Testing invalid severity value",
		"reported_by": "tester",
	})

	if !result.IsError {
		t.Fatalf("incident_create(extreme) should have returned error, got success: %s",
			resultIncText(t, result))
	}

	text := resultIncText(t, result)
	if !strings.Contains(text, "invalid incident severity") {
		t.Errorf("error text = %q, want it to contain %q", text, "invalid incident severity")
	}
}

func TestIncidentUpdateStatus(t *testing.T) {
	t.Parallel()
	env := setupIncidentTestServer(t)
	defer env.server.Close()

	created := createIncidentViaTools(t, env, "update-test", "Update Test", "medium", "Testing status update")
	incidentID := created["ID"].(string)

	result := callIncTool(t, env, "incident_update", map[string]any{
		"incident_id": incidentID,
		"status":      "triaged",
	})
	if result.IsError {
		t.Fatalf("incident_update returned error: %s", resultIncText(t, result))
	}

	text := resultIncText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse incident_update result: %v\nraw: %s", err, text)
	}

	state, ok := parsed["State"].(map[string]any)
	if !ok {
		t.Fatalf("expected State map, got %T", parsed["State"])
	}
	if state["status"] != "triaged" {
		t.Errorf("status = %v, want %q", state["status"], "triaged")
	}
}

func TestIncidentUpdateStatus_Invalid(t *testing.T) {
	t.Parallel()
	env := setupIncidentTestServer(t)
	defer env.server.Close()

	created := createIncidentViaTools(t, env, "bad-transition", "Bad Transition", "low", "Testing invalid transition")
	incidentID := created["ID"].(string)

	// reported → investigating is invalid (must go through triaged first)
	result := callIncTool(t, env, "incident_update", map[string]any{
		"incident_id": incidentID,
		"status":      "investigating",
	})

	if !result.IsError {
		t.Fatalf("incident_update(investigating) should have returned error, got success: %s",
			resultIncText(t, result))
	}
}

func TestIncidentList(t *testing.T) {
	t.Parallel()
	env := setupIncidentTestServer(t)
	defer env.server.Close()

	// Create two incidents
	createIncidentViaTools(t, env, "list-one", "List One", "high", "First incident")
	created2 := createIncidentViaTools(t, env, "list-two", "List Two", "medium", "Second incident")

	// Transition the second one to triaged
	inc2ID := created2["ID"].(string)
	transResult := callIncTool(t, env, "incident_update", map[string]any{
		"incident_id": inc2ID,
		"status":      "triaged",
	})
	if transResult.IsError {
		t.Fatalf("incident_update returned error: %s", resultIncText(t, transResult))
	}

	// List all — should get 2
	allResult := callIncTool(t, env, "incident_list", map[string]any{})
	if allResult.IsError {
		t.Fatalf("incident_list returned error: %s", resultIncText(t, allResult))
	}
	var allParsed []map[string]any
	if err := json.Unmarshal([]byte(resultIncText(t, allResult)), &allParsed); err != nil {
		t.Fatalf("parse incident_list result: %v", err)
	}
	if len(allParsed) != 2 {
		t.Errorf("incident_list (all) count = %d, want 2", len(allParsed))
	}

	// List with status filter "reported" — should get 1
	filteredResult := callIncTool(t, env, "incident_list", map[string]any{
		"status": "reported",
	})
	if filteredResult.IsError {
		t.Fatalf("incident_list(reported) returned error: %s", resultIncText(t, filteredResult))
	}
	var filteredParsed []map[string]any
	if err := json.Unmarshal([]byte(resultIncText(t, filteredResult)), &filteredParsed); err != nil {
		t.Fatalf("parse incident_list(reported) result: %v", err)
	}
	if len(filteredParsed) != 1 {
		t.Errorf("incident_list (reported) count = %d, want 1", len(filteredParsed))
	}
}

func TestIncidentLinkBug(t *testing.T) {
	t.Parallel()
	env := setupIncidentTestServer(t)
	defer env.server.Close()

	// Create an incident
	created := createIncidentViaTools(t, env, "link-test", "Link Test", "high", "Testing bug linking")
	incidentID := created["ID"].(string)

	// Create a bug via the entity tools
	bugResult := callIncTool(t, env, "create_bug", map[string]any{
		"slug":        "linked-bug",
		"title":       "A Bug To Link",
		"reported_by": "tester",
		"observed":    "System crashes",
		"expected":    "System works",
		"severity":    "high",
		"priority":    "high",
		"type":        "implementation-defect",
	})
	if bugResult.IsError {
		t.Fatalf("create_bug returned error: %s", resultIncText(t, bugResult))
	}
	var bugParsed map[string]any
	if err := json.Unmarshal([]byte(resultIncText(t, bugResult)), &bugParsed); err != nil {
		t.Fatalf("parse create_bug result: %v", err)
	}
	bugID := bugParsed["ID"].(string)

	// Link the bug to the incident
	linkResult := callIncTool(t, env, "incident_link_bug", map[string]any{
		"incident_id": incidentID,
		"bug_id":      bugID,
	})
	if linkResult.IsError {
		t.Fatalf("incident_link_bug returned error: %s", resultIncText(t, linkResult))
	}

	var linkParsed map[string]any
	if err := json.Unmarshal([]byte(resultIncText(t, linkResult)), &linkParsed); err != nil {
		t.Fatalf("parse incident_link_bug result: %v", err)
	}

	state, ok := linkParsed["State"].(map[string]any)
	if !ok {
		t.Fatalf("expected State map, got %T", linkParsed["State"])
	}

	linkedBugs, ok := state["linked_bugs"].([]any)
	if !ok {
		t.Fatalf("expected linked_bugs array, got %T", state["linked_bugs"])
	}
	if len(linkedBugs) != 1 {
		t.Fatalf("len(linked_bugs) = %d, want 1", len(linkedBugs))
	}
	if linkedBugs[0] != bugID {
		t.Errorf("linked_bugs[0] = %v, want %q", linkedBugs[0], bugID)
	}
}

func TestIncidentLinkBug_Idempotent(t *testing.T) {
	t.Parallel()
	env := setupIncidentTestServer(t)
	defer env.server.Close()

	// Create an incident
	created := createIncidentViaTools(t, env, "idem-test", "Idempotent Test", "medium", "Testing idempotent linking")
	incidentID := created["ID"].(string)

	// Create a bug
	bugResult := callIncTool(t, env, "create_bug", map[string]any{
		"slug":        "idem-bug",
		"title":       "Idempotent Bug",
		"reported_by": "tester",
		"observed":    "Broken",
		"expected":    "Working",
		"severity":    "medium",
		"priority":    "medium",
		"type":        "implementation-defect",
	})
	if bugResult.IsError {
		t.Fatalf("create_bug returned error: %s", resultIncText(t, bugResult))
	}
	var bugParsed map[string]any
	if err := json.Unmarshal([]byte(resultIncText(t, bugResult)), &bugParsed); err != nil {
		t.Fatalf("parse create_bug result: %v", err)
	}
	bugID := bugParsed["ID"].(string)

	// Link the same bug twice
	for i := 0; i < 2; i++ {
		linkResult := callIncTool(t, env, "incident_link_bug", map[string]any{
			"incident_id": incidentID,
			"bug_id":      bugID,
		})
		if linkResult.IsError {
			t.Fatalf("incident_link_bug (call %d) returned error: %s", i+1, resultIncText(t, linkResult))
		}
	}

	// Verify only one entry in linked_bugs
	getResult := callIncTool(t, env, "incident_link_bug", map[string]any{
		"incident_id": incidentID,
		"bug_id":      bugID,
	})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(resultIncText(t, getResult)), &parsed); err != nil {
		t.Fatalf("parse final link result: %v", err)
	}

	state, ok := parsed["State"].(map[string]any)
	if !ok {
		t.Fatalf("expected State map, got %T", parsed["State"])
	}
	linkedBugs, ok := state["linked_bugs"].([]any)
	if !ok {
		t.Fatalf("expected linked_bugs array, got %T", state["linked_bugs"])
	}
	if len(linkedBugs) != 1 {
		t.Errorf("len(linked_bugs) = %d after linking same bug 3 times, want 1", len(linkedBugs))
	}
}
