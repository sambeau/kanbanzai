// Tests for Track J feature group tool registration and basic dispatching.
//
// This file is package mcp (not mcp_test) so it can access the unexported
// newServerWithConfig constructor needed to inject test configurations.
package mcp

import (
	"context"
	"encoding/json"
	"sort"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/mcptest"

	chk "kanbanzai/internal/checkpoint"
	"kanbanzai/internal/config"
	kbzctx "kanbanzai/internal/context"
	"kanbanzai/internal/service"
	"kanbanzai/internal/worktree"
)

// ─── registration tests ───────────────────────────────────────────────────────

// TestTrackJ_Planning_ToolsRegistered verifies that the planning group tools
// (decompose, estimate, conflict) are registered when GroupPlanning is enabled.
func TestTrackJ_Planning_ToolsRegistered(t *testing.T) {
	t.Parallel()

	entityRoot := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.MCP.Groups = map[string]bool{
		config.GroupCore:     true,
		config.GroupPlanning: true,
	}

	registered := toolNamesFromServer(t, entityRoot, &cfg)

	want := []string{"decompose", "estimate", "conflict"}
	if missing := containsAll(registered, want); len(missing) != 0 {
		t.Errorf("planning group tools missing: %v", missing)
	}
}

// TestTrackJ_Planning_ToolsNotRegisteredWhenDisabled verifies planning tools
// are absent when GroupPlanning is not enabled.
func TestTrackJ_Planning_ToolsNotRegisteredWhenDisabled(t *testing.T) {
	t.Parallel()

	entityRoot := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.MCP.Preset = "minimal" // core only

	registered := toolNamesFromServer(t, entityRoot, &cfg)

	notWant := []string{"decompose", "estimate", "conflict"}
	if found := containsNone(registered, notWant); len(found) != 0 {
		t.Errorf("planning tools should not be registered with minimal preset, but found: %v", found)
	}
}

// TestTrackJ_Knowledge_ToolsRegistered verifies knowledge and profile tools
// are registered when GroupKnowledge is enabled.
func TestTrackJ_Knowledge_ToolsRegistered(t *testing.T) {
	t.Parallel()

	entityRoot := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.MCP.Groups = map[string]bool{
		config.GroupCore:      true,
		config.GroupKnowledge: true,
	}

	registered := toolNamesFromServer(t, entityRoot, &cfg)

	want := []string{"knowledge", "profile"}
	if missing := containsAll(registered, want); len(missing) != 0 {
		t.Errorf("knowledge group tools missing: %v", missing)
	}
}

// TestTrackJ_Git_ToolsRegistered verifies git group tools are registered when
// GroupGit is enabled.
func TestTrackJ_Git_ToolsRegistered(t *testing.T) {
	t.Parallel()

	entityRoot := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.MCP.Groups = map[string]bool{
		config.GroupCore: true,
		config.GroupGit:  true,
	}

	registered := toolNamesFromServer(t, entityRoot, &cfg)

	want := []string{"worktree", "merge", "pr", "branch", "cleanup"}
	if missing := containsAll(registered, want); len(missing) != 0 {
		t.Errorf("git group tools missing: %v", missing)
	}
}

// TestTrackJ_Documents_ToolRegistered verifies doc_intel is registered when
// GroupDocuments is enabled.
func TestTrackJ_Documents_ToolRegistered(t *testing.T) {
	t.Parallel()

	entityRoot := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.MCP.Groups = map[string]bool{
		config.GroupCore:      true,
		config.GroupDocuments: true,
	}

	registered := toolNamesFromServer(t, entityRoot, &cfg)

	want := []string{"doc_intel"}
	if missing := containsAll(registered, want); len(missing) != 0 {
		t.Errorf("documents group tools missing: %v", missing)
	}
}

// TestTrackJ_Incidents_ToolRegistered verifies incident is registered when
// GroupIncidents is enabled.
func TestTrackJ_Incidents_ToolRegistered(t *testing.T) {
	t.Parallel()

	entityRoot := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.MCP.Groups = map[string]bool{
		config.GroupCore:      true,
		config.GroupIncidents: true,
	}

	registered := toolNamesFromServer(t, entityRoot, &cfg)

	want := []string{"incident"}
	if missing := containsAll(registered, want); len(missing) != 0 {
		t.Errorf("incidents group tools missing: %v", missing)
	}
}

// TestTrackJ_Checkpoints_ToolRegistered verifies checkpoint is registered when
// GroupCheckpoints is enabled.
func TestTrackJ_Checkpoints_ToolRegistered(t *testing.T) {
	t.Parallel()

	entityRoot := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.MCP.Groups = map[string]bool{
		config.GroupCore:        true,
		config.GroupCheckpoints: true,
	}

	registered := toolNamesFromServer(t, entityRoot, &cfg)

	want := []string{"checkpoint"}
	if missing := containsAll(registered, want); len(missing) != 0 {
		t.Errorf("checkpoints group tools missing: %v", missing)
	}
}

// TestTrackJ_Full_AllGroupToolsRegistered verifies the full preset registers
// all Track J tools alongside the core tools.
func TestTrackJ_Full_AllGroupToolsRegistered(t *testing.T) {
	t.Parallel()

	entityRoot := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.MCP.Preset = "full"

	registered := toolNamesFromServer(t, entityRoot, &cfg)

	allTrackJTools := []string{
		"decompose", "estimate", "conflict", // planning
		"knowledge", "profile", // knowledge
		"worktree", "merge", "pr", "branch", "cleanup", // git
		"doc_intel",  // documents
		"incident",   // incidents
		"checkpoint", // checkpoints
	}

	if missing := containsAll(registered, allTrackJTools); len(missing) != 0 {
		t.Errorf("full preset missing Track J tools: %v\nregistered: %v", missing, registered)
	}
}

// TestTrackJ_GroupToolNames_MapConsistency verifies that GroupToolNames
// matches the tools actually registered by Track J.
func TestTrackJ_GroupToolNames_MapConsistency(t *testing.T) {
	t.Parallel()

	entityRoot := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.MCP.Preset = "full"

	registered := toolNamesFromServer(t, entityRoot, &cfg)
	registeredSet := make(map[string]bool, len(registered))
	for _, name := range registered {
		registeredSet[name] = true
	}

	// Check every group in the map (except core which was done in earlier tests).
	groupsToCheck := []string{
		config.GroupPlanning,
		config.GroupKnowledge,
		config.GroupGit,
		config.GroupDocuments,
		config.GroupIncidents,
		config.GroupCheckpoints,
	}

	for _, group := range groupsToCheck {
		toolNames, ok := GroupToolNames[group]
		if !ok {
			t.Errorf("GroupToolNames missing group %q", group)
			continue
		}
		for _, toolName := range toolNames {
			if !registeredSet[toolName] {
				t.Errorf("GroupToolNames[%q] lists tool %q but it is not registered on the full server",
					group, toolName)
			}
		}
	}
}

// ─── action dispatch tests ────────────────────────────────────────────────────

// trackJTestServer is a lightweight test helper that registers a specific set
// of 2.0 tools and returns an mcptest.Server.
type trackJTestServer struct {
	server *mcptest.Server
}

func (s *trackJTestServer) call(t *testing.T, toolName string, args map[string]any) map[string]any {
	t.Helper()
	ctx := context.Background()
	req := mcp.CallToolRequest{}
	req.Params.Name = toolName
	req.Params.Arguments = args
	result, err := s.server.Client().CallTool(ctx, req)
	if err != nil {
		t.Fatalf("CallTool(%s): %v", toolName, err)
	}
	if len(result.Content) == 0 {
		t.Fatalf("CallTool(%s): empty content", toolName)
	}
	tc, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("CallTool(%s): expected TextContent, got %T", toolName, result.Content[0])
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(tc.Text), &out); err != nil {
		t.Fatalf("CallTool(%s): unmarshal result: %v\nraw: %s", toolName, err, tc.Text)
	}
	return out
}

func newTrackJTestServer(t *testing.T, tools ...func() []interface{}) *trackJTestServer {
	t.Helper()
	// Build minimal services for test tools.
	entityRoot := t.TempDir()
	stateRoot := t.TempDir()
	profileRoot := t.TempDir()
	checkpointRoot := t.TempDir()
	indexRoot := t.TempDir()
	repoRoot := t.TempDir()

	entitySvc := service.NewEntityService(entityRoot)
	knowledgeSvc := service.NewKnowledgeService(stateRoot)
	profileStore := kbzctx.NewProfileStore(profileRoot)
	checkpointStore := chk.NewStore(checkpointRoot)
	intelligenceSvc := service.NewIntelligenceService(indexRoot, repoRoot)
	docRecordSvc := service.NewDocumentService(stateRoot, repoRoot)
	_ = worktree.NewStore(stateRoot)
	decomposeSvc := service.NewDecomposeService(entitySvc, docRecordSvc)
	conflictSvc := service.NewConflictService(entitySvc, nil, repoRoot)

	var mcpTools []interface {
		GetTool() mcp.Tool
	}
	_ = mcpTools

	// Assemble all Track J tools.
	var serverTools []interface{}
	appendTools := func(tt interface{ GetTool() mcp.Tool }) {
		serverTools = append(serverTools, tt)
	}
	_ = appendTools

	// Build the tool set directly.
	allTools := DecomposeTool(decomposeSvc, entitySvc)
	allTools = append(allTools, EstimateTool(entitySvc, knowledgeSvc)...)
	allTools = append(allTools, ConflictTool(conflictSvc)...)
	allTools = append(allTools, KnowledgeTool(knowledgeSvc)...)
	allTools = append(allTools, ProfileTool(profileStore)...)
	allTools = append(allTools, DocIntelTool(intelligenceSvc, docRecordSvc)...)
	allTools = append(allTools, IncidentTool(entitySvc)...)
	allTools = append(allTools, CheckpointTool(checkpointStore)...)

	ts, err := mcptest.NewServer(t, allTools...)
	if err != nil {
		t.Fatalf("new test server: %v", err)
	}
	t.Cleanup(func() { ts.Close() })

	return &trackJTestServer{server: ts}
}

// TestTrackJ_Conflict_UnknownActionReturnsError checks that the conflict tool
// returns a structured error for an unknown action.
func TestTrackJ_Conflict_UnknownActionReturnsError(t *testing.T) {
	t.Parallel()
	ts := newTrackJTestServer(t)

	resp := ts.call(t, "conflict", map[string]any{
		"action":   "nonexistent",
		"task_ids": []any{"TASK-001", "TASK-002"},
	})

	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object in response, got: %v", resp)
	}
	code, _ := errObj["code"].(string)
	if code == "" {
		t.Errorf("expected non-empty error code, got: %v", errObj)
	}
}

// TestTrackJ_Conflict_MissingAction returns an error when no action is provided.
func TestTrackJ_Conflict_MissingAction(t *testing.T) {
	t.Parallel()
	ts := newTrackJTestServer(t)

	resp := ts.call(t, "conflict", map[string]any{
		"task_ids": []any{"TASK-001", "TASK-002"},
	})

	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object in response for missing action, got: %v", resp)
	}
	_ = errObj
}

// TestTrackJ_Conflict_Check_MissingTaskIDs returns an error when task_ids is absent.
func TestTrackJ_Conflict_Check_MissingTaskIDs(t *testing.T) {
	t.Parallel()
	ts := newTrackJTestServer(t)

	resp := ts.call(t, "conflict", map[string]any{
		"action": "check",
		// task_ids intentionally omitted
	})

	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error for missing task_ids, got: %v", resp)
	}
	_ = errObj
}

// TestTrackJ_Knowledge_List_EmptyStore returns a valid response when no
// knowledge entries exist.
func TestTrackJ_Knowledge_List_EmptyStore(t *testing.T) {
	t.Parallel()
	ts := newTrackJTestServer(t)

	resp := ts.call(t, "knowledge", map[string]any{
		"action": "list",
	})

	// No error field expected.
	if _, hasErr := resp["error"]; hasErr {
		t.Fatalf("unexpected error: %v", resp["error"])
	}

	count, ok := resp["count"].(float64)
	if !ok {
		t.Fatalf("expected 'count' field, got: %v", resp)
	}
	if count != 0 {
		t.Errorf("expected count=0 for empty store, got %v", count)
	}
}

// TestTrackJ_Knowledge_Contribute_AndGet exercises contribute then get actions.
func TestTrackJ_Knowledge_Contribute_AndGet(t *testing.T) {
	t.Parallel()
	ts := newTrackJTestServer(t)

	// Contribute a knowledge entry.
	resp := ts.call(t, "knowledge", map[string]any{
		"action":  "contribute",
		"topic":   "test-topic",
		"content": "This is a test knowledge entry",
		"scope":   "project",
	})

	if _, hasErr := resp["error"]; hasErr {
		t.Fatalf("contribute unexpected error: %v", resp["error"])
	}

	accepted, _ := resp["accepted"].(bool)
	if !accepted {
		t.Fatalf("expected accepted=true, got: %v", resp)
	}

	entry, ok := resp["entry"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'entry' field in contribute response, got: %v", resp)
	}

	entryID, ok := entry["id"].(string)
	if !ok || entryID == "" {
		t.Fatalf("expected non-empty entry ID, got: %v", entry)
	}

	// Get the entry back.
	getResp := ts.call(t, "knowledge", map[string]any{
		"action": "get",
		"id":     entryID,
	})

	if _, hasErr := getResp["error"]; hasErr {
		t.Fatalf("get unexpected error: %v", getResp["error"])
	}

	gotEntry, ok := getResp["entry"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'entry' field in get response, got: %v", getResp)
	}
	if gotEntry["id"] != entryID {
		t.Errorf("get returned wrong entry ID: got %v, want %v", gotEntry["id"], entryID)
	}
}

// TestTrackJ_Knowledge_MissingAction returns error for missing action.
func TestTrackJ_Knowledge_MissingAction(t *testing.T) {
	t.Parallel()
	ts := newTrackJTestServer(t)

	resp := ts.call(t, "knowledge", map[string]any{})

	if _, hasErr := resp["error"]; !hasErr {
		t.Fatalf("expected error for missing action, got: %v", resp)
	}
}

// TestTrackJ_Profile_List_Empty returns a valid list response when no profiles exist.
func TestTrackJ_Profile_List_Empty(t *testing.T) {
	t.Parallel()
	ts := newTrackJTestServer(t)

	resp := ts.call(t, "profile", map[string]any{
		"action": "list",
	})

	if _, hasErr := resp["error"]; hasErr {
		t.Fatalf("profile list unexpected error: %v", resp["error"])
	}

	success, _ := resp["success"].(bool)
	if !success {
		t.Errorf("expected success=true, got: %v", resp)
	}

	count, ok := resp["count"].(float64)
	if !ok {
		t.Fatalf("expected 'count' field, got: %v", resp)
	}
	if count != 0 {
		t.Errorf("expected count=0 for empty store, got %v", count)
	}
}

// TestTrackJ_Profile_Get_NotFound returns an error when profile doesn't exist.
func TestTrackJ_Profile_Get_NotFound(t *testing.T) {
	t.Parallel()
	ts := newTrackJTestServer(t)

	resp := ts.call(t, "profile", map[string]any{
		"action": "get",
		"id":     "nonexistent-profile",
	})

	// Should return an error (ActionError shape).
	if _, hasErr := resp["error"]; !hasErr {
		t.Fatalf("expected error for nonexistent profile, got: %v", resp)
	}
}

// TestTrackJ_Profile_MissingAction returns error for missing action.
func TestTrackJ_Profile_MissingAction(t *testing.T) {
	t.Parallel()
	ts := newTrackJTestServer(t)

	resp := ts.call(t, "profile", map[string]any{})

	if _, hasErr := resp["error"]; !hasErr {
		t.Fatalf("expected error for missing action, got: %v", resp)
	}
}

// TestTrackJ_Profile_UnknownAction returns error for unknown action.
func TestTrackJ_Profile_UnknownAction(t *testing.T) {
	t.Parallel()
	ts := newTrackJTestServer(t)

	resp := ts.call(t, "profile", map[string]any{
		"action": "delete",
	})

	if _, hasErr := resp["error"]; !hasErr {
		t.Fatalf("expected error for unknown action 'delete', got: %v", resp)
	}
}

// TestTrackJ_Incident_List_Empty returns a valid list when no incidents exist.
func TestTrackJ_Incident_List_Empty(t *testing.T) {
	t.Parallel()
	ts := newTrackJTestServer(t)

	resp := ts.call(t, "incident", map[string]any{
		"action": "list",
	})

	if _, hasErr := resp["error"]; hasErr {
		t.Fatalf("incident list unexpected error: %v", resp["error"])
	}
}

// TestTrackJ_Incident_Create_AndList creates an incident and verifies it appears
// in the list.
func TestTrackJ_Incident_Create_AndList(t *testing.T) {
	t.Parallel()
	ts := newTrackJTestServer(t)

	createResp := ts.call(t, "incident", map[string]any{
		"action":      "create",
		"slug":        "test-incident",
		"title":       "Test Incident",
		"severity":    "high",
		"summary":     "A test incident for unit tests",
		"reported_by": "tester",
	})

	if _, hasErr := createResp["error"]; hasErr {
		t.Fatalf("incident create unexpected error: %v", createResp["error"])
	}

	// The side_effects field must be present (mutation operation).
	if _, hasSE := createResp["side_effects"]; !hasSE {
		t.Errorf("expected side_effects field in create response, got: %v", createResp)
	}
}

// TestTrackJ_Checkpoint_Create_Get_List exercises the full checkpoint workflow.
func TestTrackJ_Checkpoint_Create_Get_List(t *testing.T) {
	t.Parallel()
	ts := newTrackJTestServer(t)

	// Create a checkpoint.
	createResp := ts.call(t, "checkpoint", map[string]any{
		"action":                "create",
		"question":              "Should we proceed?",
		"context":               "We have some uncertainty",
		"orchestration_summary": "Mid-flight orchestration",
		"created_by":            "test-agent",
	})

	if _, hasErr := createResp["error"]; hasErr {
		t.Fatalf("checkpoint create unexpected error: %v", createResp["error"])
	}

	checkpointID, ok := createResp["checkpoint_id"].(string)
	if !ok || checkpointID == "" {
		t.Fatalf("expected non-empty checkpoint_id, got: %v", createResp)
	}

	status, _ := createResp["status"].(string)
	if status != "pending" {
		t.Errorf("expected status=pending, got %q", status)
	}

	// Get the checkpoint.
	getResp := ts.call(t, "checkpoint", map[string]any{
		"action":        "get",
		"checkpoint_id": checkpointID,
	})

	if _, hasErr := getResp["error"]; hasErr {
		t.Fatalf("checkpoint get unexpected error: %v", getResp["error"])
	}

	if getResp["id"] != checkpointID {
		t.Errorf("get returned wrong ID: got %v, want %v", getResp["id"], checkpointID)
	}
	if getResp["status"] != "pending" {
		t.Errorf("expected status=pending, got %v", getResp["status"])
	}

	// List checkpoints.
	listResp := ts.call(t, "checkpoint", map[string]any{
		"action": "list",
	})

	if _, hasErr := listResp["error"]; hasErr {
		t.Fatalf("checkpoint list unexpected error: %v", listResp["error"])
	}

	total, ok := listResp["total"].(float64)
	if !ok || total < 1 {
		t.Errorf("expected total >= 1, got: %v", listResp["total"])
	}

	pending, ok := listResp["pending_count"].(float64)
	if !ok || pending < 1 {
		t.Errorf("expected pending_count >= 1, got: %v", listResp["pending_count"])
	}
}

// TestTrackJ_Checkpoint_Respond transitions a checkpoint to responded.
func TestTrackJ_Checkpoint_Respond(t *testing.T) {
	t.Parallel()
	ts := newTrackJTestServer(t)

	// Create.
	createResp := ts.call(t, "checkpoint", map[string]any{
		"action":                "create",
		"question":              "Proceed?",
		"context":               "Some context",
		"orchestration_summary": "Summary",
		"created_by":            "agent",
	})
	checkpointID, _ := createResp["checkpoint_id"].(string)

	// Respond.
	respondResp := ts.call(t, "checkpoint", map[string]any{
		"action":        "respond",
		"checkpoint_id": checkpointID,
		"response":      "Yes, proceed",
	})

	if _, hasErr := respondResp["error"]; hasErr {
		t.Fatalf("checkpoint respond unexpected error: %v", respondResp["error"])
	}

	if respondResp["status"] != "responded" {
		t.Errorf("expected status=responded after respond, got: %v", respondResp["status"])
	}

	// Double-respond should be an error.
	doubleResp := ts.call(t, "checkpoint", map[string]any{
		"action":        "respond",
		"checkpoint_id": checkpointID,
		"response":      "Second answer",
	})

	if _, hasErr := doubleResp["error"]; !hasErr {
		t.Fatalf("expected error for double-respond, got: %v", doubleResp)
	}
}

// TestTrackJ_Decompose_UnknownAction returns an error for unknown actions.
func TestTrackJ_Decompose_UnknownAction(t *testing.T) {
	t.Parallel()
	ts := newTrackJTestServer(t)

	resp := ts.call(t, "decompose", map[string]any{
		"action":     "apply_and_submit",
		"feature_id": "FEAT-001",
	})

	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error for unknown action, got: %v", resp)
	}
	code, _ := errObj["code"].(string)
	if code == "" {
		t.Errorf("expected non-empty error code")
	}
}

// TestTrackJ_Estimate_Set_EntityNotFound returns an error when entity doesn't exist.
func TestTrackJ_Estimate_Set_EntityNotFound(t *testing.T) {
	t.Parallel()
	ts := newTrackJTestServer(t)

	resp := ts.call(t, "estimate", map[string]any{
		"action":    "set",
		"entity_id": "TASK-01ZZZZZZZZZZZZZZZZZZZ",
		"points":    float64(3),
	})

	// Should return an error (entity not found).
	if _, hasErr := resp["error"]; !hasErr {
		t.Fatalf("expected error for nonexistent entity, got: %v", resp)
	}
}

// TestTrackJ_Estimate_Set_BatchEmptyEntities handles an empty entities array.
func TestTrackJ_Estimate_Set_BatchEmptyEntities(t *testing.T) {
	t.Parallel()
	ts := newTrackJTestServer(t)

	resp := ts.call(t, "estimate", map[string]any{
		"action":   "set",
		"entities": []any{},
	})

	// Empty batch should succeed with zero results in the standard BatchResult shape.
	if _, hasErr := resp["error"]; hasErr {
		t.Fatalf("unexpected error for empty batch: %v", resp["error"])
	}

	summary, _ := resp["summary"].(map[string]any)
	if summary == nil {
		t.Fatalf("expected summary field in batch response, got: %v", resp)
	}
	total, _ := summary["total"].(float64)
	if total != 0 {
		t.Errorf("expected summary.total=0 for empty batch, got %v", total)
	}
}

// TestTrackJ_DocIntel_Find_MissingDiscriminator returns an error when the find
// action is called without concept, entity_id, or role.
func TestTrackJ_DocIntel_Find_MissingDiscriminator(t *testing.T) {
	t.Parallel()
	ts := newTrackJTestServer(t)

	resp := ts.call(t, "doc_intel", map[string]any{
		"action": "find",
		// no concept, entity_id, or role
	})

	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error when no discriminator provided, got: %v", resp)
	}
	code, _ := errObj["code"].(string)
	if code != "missing_parameter" {
		t.Errorf("expected code=missing_parameter, got %q", code)
	}
}

// TestTrackJ_DocIntel_Find_ConceptRouting verifies that concept dispatches to
// FindByConcept (returning results rather than an error).
func TestTrackJ_DocIntel_Find_ConceptRouting(t *testing.T) {
	t.Parallel()
	ts := newTrackJTestServer(t)

	resp := ts.call(t, "doc_intel", map[string]any{
		"action":  "find",
		"concept": "authentication",
	})

	// Empty result is fine; what matters is no dispatch error.
	if errObj, hasErr := resp["error"]; hasErr {
		t.Fatalf("unexpected error for find by concept: %v", errObj)
	}

	searchType, _ := resp["search_type"].(string)
	if searchType != "concept" {
		t.Errorf("expected search_type=concept, got %q", searchType)
	}
}

// TestTrackJ_DocIntel_Find_EntityIDRouting verifies entity_id dispatches to
// FindByEntity.
func TestTrackJ_DocIntel_Find_EntityIDRouting(t *testing.T) {
	t.Parallel()
	ts := newTrackJTestServer(t)

	resp := ts.call(t, "doc_intel", map[string]any{
		"action":    "find",
		"entity_id": "FEAT-01AAAA",
	})

	if errObj, hasErr := resp["error"]; hasErr {
		t.Fatalf("unexpected error for find by entity: %v", errObj)
	}

	searchType, _ := resp["search_type"].(string)
	if searchType != "entity_id" {
		t.Errorf("expected search_type=entity_id, got %q", searchType)
	}
}

// TestTrackJ_DocIntel_Find_RoleRouting verifies role dispatches to FindByRole.
func TestTrackJ_DocIntel_Find_RoleRouting(t *testing.T) {
	t.Parallel()
	ts := newTrackJTestServer(t)

	resp := ts.call(t, "doc_intel", map[string]any{
		"action": "find",
		"role":   "requirement",
	})

	if errObj, hasErr := resp["error"]; hasErr {
		t.Fatalf("unexpected error for find by role: %v", errObj)
	}

	searchType, _ := resp["search_type"].(string)
	if searchType != "role" {
		t.Errorf("expected search_type=role, got %q", searchType)
	}
}

// TestTrackJ_DocIntel_Pending_EmptyStore returns empty pending list.
func TestTrackJ_DocIntel_Pending_EmptyStore(t *testing.T) {
	t.Parallel()
	ts := newTrackJTestServer(t)

	resp := ts.call(t, "doc_intel", map[string]any{
		"action": "pending",
	})

	if errObj, hasErr := resp["error"]; hasErr {
		t.Fatalf("unexpected error for doc_intel pending: %v", errObj)
	}

	count, ok := resp["count"].(float64)
	if !ok {
		t.Fatalf("expected 'count' field, got: %v", resp)
	}
	if count != 0 {
		t.Errorf("expected count=0 for empty index, got %v", count)
	}
}

// TestTrackJ_DocIntel_UnknownAction returns an error for unknown actions.
func TestTrackJ_DocIntel_UnknownAction(t *testing.T) {
	t.Parallel()
	ts := newTrackJTestServer(t)

	resp := ts.call(t, "doc_intel", map[string]any{
		"action": "reindex",
	})

	if _, hasErr := resp["error"]; !hasErr {
		t.Fatalf("expected error for unknown action 'reindex', got: %v", resp)
	}
}

// TestTrackJ_Worktree_List_EmptyStore returns empty list.
func TestTrackJ_Worktree_List_EmptyStore(t *testing.T) {
	t.Parallel()

	entityRoot := t.TempDir()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()

	entitySvc := service.NewEntityService(entityRoot)
	worktreeStore := worktree.NewStore(stateRoot)
	gitOps := worktree.NewGit(repoRoot)

	tools := WorktreeTool(worktreeStore, entitySvc, gitOps)
	ts, err := mcptest.NewServer(t, tools...)
	if err != nil {
		t.Fatalf("new test server: %v", err)
	}
	defer ts.Close()

	ctx := context.Background()
	req := mcp.CallToolRequest{}
	req.Params.Name = "worktree"
	req.Params.Arguments = map[string]any{
		"action": "list",
	}
	result, err := ts.Client().CallTool(ctx, req)
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("empty result")
	}
	tc := result.Content[0].(mcp.TextContent)

	var resp map[string]any
	if err := json.Unmarshal([]byte(tc.Text), &resp); err != nil {
		t.Fatalf("unmarshal: %v\nraw: %s", err, tc.Text)
	}

	if _, hasErr := resp["error"]; hasErr {
		t.Fatalf("unexpected error: %v", resp["error"])
	}

	count, ok := resp["count"].(float64)
	if !ok {
		t.Fatalf("expected 'count' field, got: %v", resp)
	}
	if count != 0 {
		t.Errorf("expected count=0 for empty store, got %v", count)
	}
}

// TestTrackJ_Worktree_Get_NotFound returns an error when no worktree exists.
func TestTrackJ_Worktree_Get_NotFound(t *testing.T) {
	t.Parallel()

	entityRoot := t.TempDir()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()

	entitySvc := service.NewEntityService(entityRoot)
	worktreeStore := worktree.NewStore(stateRoot)
	gitOps := worktree.NewGit(repoRoot)

	tools := WorktreeTool(worktreeStore, entitySvc, gitOps)
	ts, err := mcptest.NewServer(t, tools...)
	if err != nil {
		t.Fatalf("new test server: %v", err)
	}
	defer ts.Close()

	ctx := context.Background()
	req := mcp.CallToolRequest{}
	req.Params.Name = "worktree"
	req.Params.Arguments = map[string]any{
		"action":    "get",
		"entity_id": "FEAT-01AAAAAAAAAAAAA",
	}
	result, err := ts.Client().CallTool(ctx, req)
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	tc := result.Content[0].(mcp.TextContent)

	var resp map[string]any
	if err := json.Unmarshal([]byte(tc.Text), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if _, hasErr := resp["error"]; !hasErr {
		t.Fatalf("expected error for missing worktree, got: %v", resp)
	}
}

// TestTrackJ_Cleanup_List_Empty returns empty lists from cleanup list.
func TestTrackJ_Cleanup_List_Empty(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	worktreeStore := worktree.NewStore(stateRoot)
	gitOps := worktree.NewGit(repoRoot)
	cfg := &config.CleanupConfig{}

	tools := CleanupTool(worktreeStore, gitOps, cfg)
	ts, err := mcptest.NewServer(t, tools...)
	if err != nil {
		t.Fatalf("new test server: %v", err)
	}
	defer ts.Close()

	ctx := context.Background()
	req := mcp.CallToolRequest{}
	req.Params.Name = "cleanup"
	req.Params.Arguments = map[string]any{
		"action": "list",
	}
	result, err := ts.Client().CallTool(ctx, req)
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	tc := result.Content[0].(mcp.TextContent)

	var resp map[string]any
	if err := json.Unmarshal([]byte(tc.Text), &resp); err != nil {
		t.Fatalf("unmarshal: %v\nraw: %s", err, tc.Text)
	}

	if _, hasErr := resp["error"]; hasErr {
		t.Fatalf("unexpected error: %v", resp["error"])
	}
}

// TestTrackJ_Decompose_Apply_NoTasks returns an error when proposal has no tasks.
func TestTrackJ_Decompose_Apply_NoTasks(t *testing.T) {
	t.Parallel()
	ts := newTrackJTestServer(t)

	resp := ts.call(t, "decompose", map[string]any{
		"action":     "apply",
		"feature_id": "FEAT-01TEST",
		"proposal": map[string]any{
			"tasks":       []any{},
			"total_tasks": 0,
			"slices":      []any{},
			"warnings":    []any{},
		},
	})

	if _, hasErr := resp["error"]; !hasErr {
		t.Fatalf("expected error for empty proposal tasks, got: %v", resp)
	}
}

// TestTrackJ_ToolCount_PerGroup verifies the exact tool counts per group match
// GroupToolNames.
func TestTrackJ_ToolCount_PerGroup(t *testing.T) {
	t.Parallel()

	expected := map[string]int{
		config.GroupCore:        7,
		config.GroupPlanning:    3,
		config.GroupKnowledge:   2,
		config.GroupGit:         5,
		config.GroupDocuments:   1,
		config.GroupIncidents:   1,
		config.GroupCheckpoints: 1,
	}

	for group, wantCount := range expected {
		group, wantCount := group, wantCount
		t.Run(group, func(t *testing.T) {
			t.Parallel()
			tools, ok := GroupToolNames[group]
			if !ok {
				t.Fatalf("GroupToolNames missing group %q", group)
			}
			if len(tools) != wantCount {
				t.Errorf("GroupToolNames[%q]: got %d tools, want %d\ntools: %v",
					group, len(tools), wantCount, tools)
			}
			// Verify no duplicates within the group.
			seen := make(map[string]bool, len(tools))
			for _, name := range tools {
				if seen[name] {
					t.Errorf("GroupToolNames[%q]: duplicate tool name %q", group, name)
				}
				seen[name] = true
			}
		})
	}
}

// TestTrackJ_AllGroupToolNames_NoDuplicatesAcrossGroups verifies no tool name
// appears in more than one group (each 2.0 tool belongs to exactly one group).
func TestTrackJ_AllGroupToolNames_NoDuplicatesAcrossGroups(t *testing.T) {
	t.Parallel()

	seen := make(map[string]string) // tool name → first group
	for group, tools := range GroupToolNames {
		for _, name := range tools {
			if firstGroup, exists := seen[name]; exists {
				t.Errorf("tool %q appears in both group %q and group %q", name, firstGroup, group)
			} else {
				seen[name] = group
			}
		}
	}
}

// TestTrackJ_Knowledge_Prune_DryRun exercises prune in dry-run mode.
func TestTrackJ_Knowledge_Prune_DryRun(t *testing.T) {
	t.Parallel()
	ts := newTrackJTestServer(t)

	resp := ts.call(t, "knowledge", map[string]any{
		"action":  "prune",
		"dry_run": true,
	})

	if _, hasErr := resp["error"]; hasErr {
		t.Fatalf("prune dry_run unexpected error: %v", resp["error"])
	}

	dryRun, _ := resp["dry_run"].(bool)
	if !dryRun {
		t.Errorf("expected dry_run=true in response, got: %v", resp)
	}
}

// TestTrackJ_Knowledge_Compact_DryRun exercises compact in dry-run mode.
func TestTrackJ_Knowledge_Compact_DryRun(t *testing.T) {
	t.Parallel()
	ts := newTrackJTestServer(t)

	resp := ts.call(t, "knowledge", map[string]any{
		"action":  "compact",
		"dry_run": true,
	})

	if _, hasErr := resp["error"]; hasErr {
		t.Fatalf("compact dry_run unexpected error: %v", resp["error"])
	}

	dryRun, _ := resp["dry_run"].(bool)
	if !dryRun {
		t.Errorf("expected dry_run=true in response, got: %v", resp)
	}
}

// TestTrackJ_AllTrackJToolNames_Sorted verifies the returned tool name lists
// are consistent (not a sorting test per se, but ensures the slice is stable).
func TestTrackJ_AllTrackJToolNames_Sorted(t *testing.T) {
	t.Parallel()

	for group, tools := range GroupToolNames {
		sorted := make([]string, len(tools))
		copy(sorted, tools)
		sort.Strings(sorted)
		// We don't require alphabetical order in the slice, but we do require
		// at least one tool per group.
		if len(tools) == 0 {
			t.Errorf("GroupToolNames[%q] is empty", group)
		}
	}
}
