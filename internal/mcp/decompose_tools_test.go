package mcp_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/mcptest"

	kbzmcp "kanbanzai/internal/mcp"
	"kanbanzai/internal/service"
)

// decomposeEnv holds the test server and service references for decompose tests.
type decomposeEnv struct {
	server     *mcptest.Server
	entitySvc  *service.EntityService
	docSvc     *service.DocumentService
	entityRoot string
	repoRoot   string
	featureID  string
	specDocID  string
}

// setupDecomposeTestServer creates a test MCP server with entity, doc record,
// and decompose tools registered. It creates a plan on disk (bypassing the
// prefix registry), a feature via the MCP create_feature tool, a spec document
// via doc_record_submit, and links the spec to the feature via update_entity.
func setupDecomposeTestServer(t *testing.T, specContent string) *decomposeEnv {
	t.Helper()

	entityRoot := t.TempDir()
	repoRoot := t.TempDir()

	entitySvc := service.NewEntityService(entityRoot)
	docSvc := service.NewDocumentService(entityRoot, repoRoot)
	decomposeSvc := service.NewDecomposeService(entitySvc, docSvc)

	// Write a plan directly to disk, bypassing the prefix registry/allocator.
	planDir := filepath.Join(entityRoot, "plans")
	if err := os.MkdirAll(planDir, 0o755); err != nil {
		t.Fatalf("mkdir plans: %v", err)
	}
	planYAML := "id: P1-decompose-test\nslug: decompose-test\ntitle: Test Plan\nstatus: active\nsummary: Plan for decompose tests\ncreated: \"2026-03-19T12:00:00Z\"\ncreated_by: test\nupdated: \"2026-03-19T12:00:00Z\"\n"
	if err := os.WriteFile(filepath.Join(planDir, "P1-decompose-test.yaml"), []byte(planYAML), 0o644); err != nil {
		t.Fatalf("write plan file: %v", err)
	}

	tools := kbzmcp.EntityTools(entitySvc)
	tools = append(tools, kbzmcp.DocRecordTools(docSvc)...)
	tools = append(tools, kbzmcp.DecomposeTools(decomposeSvc)...)

	ts, err := mcptest.NewServer(t, tools...)
	if err != nil {
		t.Fatalf("start decompose test server: %v", err)
	}

	env := &decomposeEnv{
		server:     ts,
		entitySvc:  entitySvc,
		docSvc:     docSvc,
		entityRoot: entityRoot,
		repoRoot:   repoRoot,
	}

	// Create a feature via MCP tool.
	featResult := callDecomposeTool(t, env, "create_feature", map[string]any{
		"slug":       "decompose-feat",
		"parent":     "P1-decompose-test",
		"summary":    "Feature for decompose tests",
		"created_by": "tester",
	})
	if featResult.IsError {
		t.Fatalf("create_feature returned error: %s", decomposeResultText(t, featResult))
	}
	var featParsed map[string]any
	if err := json.Unmarshal([]byte(decomposeResultText(t, featResult)), &featParsed); err != nil {
		t.Fatalf("parse create_feature result: %v", err)
	}
	featureID, ok := featParsed["ID"].(string)
	if !ok || featureID == "" {
		t.Fatal("create_feature returned empty ID")
	}
	env.featureID = featureID

	if specContent != "" {
		// Write the spec document file to repoRoot.
		specPath := "work/spec/decompose-test-spec.md"
		fullPath := filepath.Join(repoRoot, specPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("mkdir for spec: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(specContent), 0o644); err != nil {
			t.Fatalf("write spec file: %v", err)
		}

		// Submit spec document via MCP tool.
		docResult := callDecomposeTool(t, env, "doc_record_submit", map[string]any{
			"path":       specPath,
			"type":       "specification",
			"title":      "Test Specification",
			"owner":      featureID,
			"created_by": "tester",
		})
		if docResult.IsError {
			t.Fatalf("doc_record_submit returned error: %s", decomposeResultText(t, docResult))
		}
		var docParsed map[string]any
		if err := json.Unmarshal([]byte(decomposeResultText(t, docResult)), &docParsed); err != nil {
			t.Fatalf("parse doc_record_submit result: %v", err)
		}
		docMap, ok := docParsed["document"].(map[string]any)
		if !ok {
			t.Fatal("doc_record_submit did not return document map")
		}
		specDocID, ok := docMap["id"].(string)
		if !ok || specDocID == "" {
			t.Fatal("doc_record_submit returned empty document ID")
		}
		env.specDocID = specDocID

		// Link spec document to feature via update_entity.
		updateResult := callDecomposeTool(t, env, "update_entity", map[string]any{
			"entity_type": "feature",
			"id":          featureID,
			"spec":        specDocID,
		})
		if updateResult.IsError {
			t.Fatalf("update_entity returned error: %s", decomposeResultText(t, updateResult))
		}
	}

	return env
}

// callDecomposeTool calls a tool on the decompose test server.
func callDecomposeTool(t *testing.T, env *decomposeEnv, name string, args map[string]any) *mcp.CallToolResult {
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

// decomposeResultText extracts text content from a tool result.
func decomposeResultText(t *testing.T, result *mcp.CallToolResult) string {
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

func TestDecomposeFeature_MCP_NoSpec(t *testing.T) {
	t.Parallel()

	// Set up a feature with no spec linked (empty content skips doc creation).
	env := setupDecomposeTestServer(t, "")
	defer env.server.Close()

	result := callDecomposeTool(t, env, "decompose_feature", map[string]any{
		"feature_id": env.featureID,
	})

	if !result.IsError {
		t.Fatalf("decompose_feature should return error when no spec linked, got success: %s",
			decomposeResultText(t, result))
	}

	text := decomposeResultText(t, result)
	if !strings.Contains(text, "no linked specification document") {
		t.Errorf("error text = %q, want it to contain %q", text, "no linked specification document")
	}
}

func TestDecomposeFeature_MCP_ProposalProduced(t *testing.T) {
	t.Parallel()

	specContent := `# Feature Spec

## Authentication

### Acceptance Criteria

- [ ] Users can log in with email and password
- [ ] Users can reset their password via email
- [ ] Sessions expire after 24 hours of inactivity

## Authorization

- [ ] Role-based access control is enforced on all API endpoints
- [ ] Admin users can manage other users
`
	env := setupDecomposeTestServer(t, specContent)
	defer env.server.Close()

	result := callDecomposeTool(t, env, "decompose_feature", map[string]any{
		"feature_id": env.featureID,
	})
	if result.IsError {
		t.Fatalf("decompose_feature returned error: %s", decomposeResultText(t, result))
	}

	text := decomposeResultText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse decompose_feature result: %v\nraw: %s", err, text)
	}

	if parsed["feature_id"] == "" {
		t.Error("result.feature_id is empty")
	}
	if parsed["spec_document_id"] == "" {
		t.Error("result.spec_document_id is empty")
	}

	proposal, ok := parsed["proposal"].(map[string]any)
	if !ok {
		t.Fatalf("expected proposal map, got %T", parsed["proposal"])
	}

	totalTasks, _ := proposal["total_tasks"].(float64)
	if totalTasks < 5 {
		t.Errorf("total_tasks = %v, want at least 5 (one per acceptance criterion)", totalTasks)
	}

	tasks, ok := proposal["tasks"].([]any)
	if !ok {
		t.Fatalf("expected tasks array, got %T", proposal["tasks"])
	}
	for i, raw := range tasks {
		task, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("task[%d] is not a map", i)
		}
		if task["slug"] == nil || task["slug"] == "" {
			t.Errorf("task[%d].slug is empty", i)
		}
		if task["summary"] == nil || task["summary"] == "" {
			t.Errorf("task[%d].summary is empty", i)
		}
		if task["rationale"] == nil || task["rationale"] == "" {
			t.Errorf("task[%d].rationale is empty", i)
		}
	}

	slices, ok := proposal["slices"].([]any)
	if !ok || len(slices) == 0 {
		t.Error("expected non-empty slices")
	}
	foundAuth := false
	foundAuthz := false
	for _, s := range slices {
		if s == "Authentication" {
			foundAuth = true
		}
		if s == "Authorization" {
			foundAuthz = true
		}
	}
	if !foundAuth {
		t.Errorf("slices = %v, want it to contain %q", slices, "Authentication")
	}
	if !foundAuthz {
		t.Errorf("slices = %v, want it to contain %q", slices, "Authorization")
	}
}

func TestDecomposeReview_MCP_Pass(t *testing.T) {
	t.Parallel()

	specContent := `# Spec
- [ ] Users can log in
- [ ] Users can log out
`
	env := setupDecomposeTestServer(t, specContent)
	defer env.server.Close()

	proposal := map[string]any{
		"tasks": []any{
			map[string]any{
				"slug":      "login",
				"summary":   "Implement user login with email and password",
				"rationale": "Covers acceptance criterion: users can log in",
			},
			map[string]any{
				"slug":      "logout",
				"summary":   "Implement user logout and session cleanup",
				"rationale": "Covers acceptance criterion: users can log out",
			},
		},
		"total_tasks": 2,
	}

	result := callDecomposeTool(t, env, "decompose_review", map[string]any{
		"feature_id": env.featureID,
		"proposal":   proposal,
	})
	if result.IsError {
		t.Fatalf("decompose_review returned error: %s", decomposeResultText(t, result))
	}

	text := decomposeResultText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse decompose_review result: %v\nraw: %s", err, text)
	}

	if parsed["status"] != "pass" {
		t.Errorf("status = %v, want %q", parsed["status"], "pass")
	}
	blockingCount, _ := parsed["blocking_count"].(float64)
	if blockingCount != 0 {
		t.Errorf("blocking_count = %v, want 0", blockingCount)
	}
}

func TestDecomposeReview_MCP_Gap(t *testing.T) {
	t.Parallel()

	specContent := `# Spec
- [ ] Users can log in with email and password
- [ ] Users can reset their password
- [ ] Sessions expire after inactivity
`
	env := setupDecomposeTestServer(t, specContent)
	defer env.server.Close()

	// Proposal only covers login — missing password reset and session expiry.
	proposal := map[string]any{
		"tasks": []any{
			map[string]any{
				"slug":      "login",
				"summary":   "Implement user login with email and password",
				"rationale": "Covers: users can log in with email and password",
			},
		},
		"total_tasks": 1,
	}

	result := callDecomposeTool(t, env, "decompose_review", map[string]any{
		"feature_id": env.featureID,
		"proposal":   proposal,
	})
	if result.IsError {
		t.Fatalf("decompose_review returned error: %s", decomposeResultText(t, result))
	}

	text := decomposeResultText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse decompose_review result: %v\nraw: %s", err, text)
	}

	if parsed["status"] != "fail" {
		t.Errorf("status = %v, want %q (gaps are blocking)", parsed["status"], "fail")
	}

	findings, ok := parsed["findings"].([]any)
	if !ok {
		t.Fatalf("expected findings array, got %T", parsed["findings"])
	}

	gapCount := 0
	for _, raw := range findings {
		f, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if f["type"] == "gap" {
			gapCount++
		}
	}
	if gapCount < 2 {
		t.Errorf("gap findings = %d, want at least 2 (password reset, session expiry)", gapCount)
	}

	blockingCount, _ := parsed["blocking_count"].(float64)
	if blockingCount < 2 {
		t.Errorf("blocking_count = %v, want at least 2", blockingCount)
	}
}

func TestDecomposeReview_MCP_Oversized(t *testing.T) {
	t.Parallel()

	specContent := `# Spec
- [ ] Feature is implemented
`
	env := setupDecomposeTestServer(t, specContent)
	defer env.server.Close()

	proposal := map[string]any{
		"tasks": []any{
			map[string]any{
				"slug":      "big-task",
				"summary":   "Implement the entire feature in one monolithic task that is implemented",
				"estimate":  13.0,
				"rationale": "Covers: feature is implemented",
			},
		},
		"total_tasks": 1,
	}

	result := callDecomposeTool(t, env, "decompose_review", map[string]any{
		"feature_id": env.featureID,
		"proposal":   proposal,
	})
	if result.IsError {
		t.Fatalf("decompose_review returned error: %s", decomposeResultText(t, result))
	}

	text := decomposeResultText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse decompose_review result: %v\nraw: %s", err, text)
	}

	findings, ok := parsed["findings"].([]any)
	if !ok {
		t.Fatalf("expected findings array, got %T", parsed["findings"])
	}

	oversizedCount := 0
	for _, raw := range findings {
		f, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if f["type"] == "oversized" {
			oversizedCount++
			if f["task_slug"] != "big-task" {
				t.Errorf("oversized finding task_slug = %v, want %q", f["task_slug"], "big-task")
			}
		}
	}
	if oversizedCount != 1 {
		t.Errorf("oversized findings = %d, want 1", oversizedCount)
	}
}

func TestDecomposeReview_MCP_Cycle(t *testing.T) {
	t.Parallel()

	specContent := `# Spec
- [ ] Task alpha is done
- [ ] Task beta is done
- [ ] Task gamma is done
`
	env := setupDecomposeTestServer(t, specContent)
	defer env.server.Close()

	// Create a dependency cycle: alpha → beta → gamma → alpha.
	proposal := map[string]any{
		"tasks": []any{
			map[string]any{
				"slug":       "alpha",
				"summary":    "Task alpha is done and depends on gamma",
				"depends_on": []any{"gamma"},
				"rationale":  "Covers: task alpha is done",
			},
			map[string]any{
				"slug":       "beta",
				"summary":    "Task beta is done and depends on alpha",
				"depends_on": []any{"alpha"},
				"rationale":  "Covers: task beta is done",
			},
			map[string]any{
				"slug":       "gamma",
				"summary":    "Task gamma is done and depends on beta",
				"depends_on": []any{"beta"},
				"rationale":  "Covers: task gamma is done",
			},
		},
		"total_tasks": 3,
	}

	result := callDecomposeTool(t, env, "decompose_review", map[string]any{
		"feature_id": env.featureID,
		"proposal":   proposal,
	})
	if result.IsError {
		t.Fatalf("decompose_review returned error: %s", decomposeResultText(t, result))
	}

	text := decomposeResultText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("parse decompose_review result: %v\nraw: %s", err, text)
	}

	if parsed["status"] != "fail" {
		t.Errorf("status = %v, want %q (cycles are blocking)", parsed["status"], "fail")
	}

	findings, ok := parsed["findings"].([]any)
	if !ok {
		t.Fatalf("expected findings array, got %T", parsed["findings"])
	}

	cycleCount := 0
	for _, raw := range findings {
		f, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if f["type"] == "cycle" {
			cycleCount++
		}
	}
	if cycleCount == 0 {
		t.Error("expected at least one cycle finding, got none")
	}
}
