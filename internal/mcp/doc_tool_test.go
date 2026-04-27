package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/kanbanzai/internal/service"
)

// ─── Test setup ───────────────────────────────────────────────────────────────

type docToolEnv struct {
	docSvc   *service.DocumentService
	intelSvc *service.IntelligenceService
	repoRoot string
}

func setupDocToolTest(t *testing.T) *docToolEnv {
	t.Helper()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	indexRoot := t.TempDir()
	intelSvc := service.NewIntelligenceService(indexRoot, repoRoot)
	t.Cleanup(func() { _ = intelSvc.Close() })
	return &docToolEnv{
		docSvc:   service.NewDocumentService(stateRoot, repoRoot),
		intelSvc: intelSvc,
		repoRoot: repoRoot,
	}
}

// writeDocFile creates a file at repoRoot/relPath with the given content.
func writeDocFile(t *testing.T, repoRoot, relPath, content string) {
	t.Helper()
	full := filepath.Join(repoRoot, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("writeDocFile mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("writeDocFile write: %v", err)
	}
}

// callDoc invokes the doc tool with the given args and returns the parsed JSON map.
func callDoc(t *testing.T, env *docToolEnv, args map[string]any) map[string]any {
	t.Helper()
	tool := docTool(env.docSvc, nil, nil)
	req := makeRequest(args)
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("doc handler error: %v", err)
	}
	text := extractText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("unmarshal response: %v\nraw: %s", err, text)
	}
	return parsed
}

// registerDoc is a helper that registers a document and returns its ID.
func registerDoc(t *testing.T, env *docToolEnv, relPath, docType, title string) string {
	t.Helper()
	writeDocFile(t, env.repoRoot, relPath, "# "+title+"\n\nContent.")
	resp := callDoc(t, env, map[string]any{
		"action": "register",
		"path":   relPath,
		"type":   docType,
		"title":  title,
	})
	doc, ok := resp["document"].(map[string]any)
	if !ok {
		t.Fatalf("register: expected document field, got: %v", resp)
	}
	id, _ := doc["id"].(string)
	if id == "" {
		t.Fatalf("register: document.id is empty; response: %v", resp)
	}
	return id
}

// mockDocHook is a test-local implementation of service.EntityLifecycleHook
// used in cascade tests. It stores entity type and current status so that
// DocumentService.ApproveDocument / SupersedeDocument can determine the
// correct cascade target without requiring a real EntityService.
type mockDocHook struct {
	entityType string
	status     string
}

func (m *mockDocHook) TransitionStatus(_ string, newStatus string) error {
	m.status = newStatus
	return nil
}

func (m *mockDocHook) SetDocumentRef(_, _, _ string) error { return nil }

func (m *mockDocHook) GetEntityStatus(_ string) (string, string, error) {
	return m.entityType, m.status, nil
}

// ─── register ─────────────────────────────────────────────────────────────────

func TestDocTool_Register_Single(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)
	writeDocFile(t, env.repoRoot, "work/spec/foo.md", "# Foo\n\nSpec content.")

	resp := callDoc(t, env, map[string]any{
		"action": "register",
		"path":   "work/spec/foo.md",
		"type":   "specification",
		"title":  "Foo Specification",
		"owner":  "FEAT-001",
	})

	doc, ok := resp["document"].(map[string]any)
	if !ok {
		t.Fatalf("expected document field, got: %v", resp)
	}
	if doc["status"] != "draft" {
		t.Errorf("status = %q, want draft", doc["status"])
	}
	if doc["type"] != "spec" {
		t.Errorf("type = %q, want spec", doc["type"])
	}
	if doc["title"] != "Foo Specification" {
		t.Errorf("title = %q, want Foo Specification", doc["title"])
	}
	if doc["owner"] != "FEAT-001" {
		t.Errorf("owner = %q, want FEAT-001", doc["owner"])
	}
	// side_effects must be present on mutation (spec §8.4)
	if _, hasSE := resp["side_effects"]; !hasSE {
		t.Error("expected side_effects field on mutation response")
	}
}

func TestDocTool_Register_MissingPath(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	resp := callDoc(t, env, map[string]any{
		"action": "register",
		"type":   "design",
		"title":  "No Path",
	})

	if _, hasErr := resp["error"]; !hasErr {
		t.Errorf("expected error response, got: %v", resp)
	}
}

func TestDocTool_Register_Batch(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	writeDocFile(t, env.repoRoot, "work/design/a.md", "# A\n")
	writeDocFile(t, env.repoRoot, "work/spec/b.md", "# B\n")

	resp := callDoc(t, env, map[string]any{
		"action": "register",
		"documents": []any{
			map[string]any{"path": "work/design/a.md", "type": "design", "title": "Doc A"},
			map[string]any{"path": "work/spec/b.md", "type": "specification", "title": "Doc B"},
		},
	})

	results, ok := resp["results"].([]any)
	if !ok {
		t.Fatalf("expected results array, got: %v", resp)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for i, r := range results {
		item, _ := r.(map[string]any)
		if item["status"] != "ok" {
			t.Errorf("result[%d] status = %q, want ok", i, item["status"])
		}
	}

	summary, _ := resp["summary"].(map[string]any)
	if summary["total"] != float64(2) {
		t.Errorf("summary.total = %v, want 2", summary["total"])
	}
	if summary["succeeded"] != float64(2) {
		t.Errorf("summary.succeeded = %v, want 2", summary["succeeded"])
	}
}

// ─── approve ──────────────────────────────────────────────────────────────────

func TestDocTool_Approve_Single(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	id := registerDoc(t, env, "work/design/mydesign.md", "design", "My Design")

	resp := callDoc(t, env, map[string]any{
		"action": "approve",
		"id":     id,
	})

	doc, ok := resp["document"].(map[string]any)
	if !ok {
		t.Fatalf("expected document field, got: %v", resp)
	}
	if doc["status"] != "approved" {
		t.Errorf("status = %q, want approved", doc["status"])
	}
	// side_effects must be present (mutation)
	if _, hasSE := resp["side_effects"]; !hasSE {
		t.Error("expected side_effects field on mutation response")
	}
}

func TestDocTool_Approve_MissingID(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	resp := callDoc(t, env, map[string]any{
		"action": "approve",
	})

	if _, hasErr := resp["error"]; !hasErr {
		t.Errorf("expected error response, got: %v", resp)
	}
}

func TestDocTool_Approve_Batch(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	id1 := registerDoc(t, env, "work/design/d1.md", "design", "Design 1")
	id2 := registerDoc(t, env, "work/spec/s1.md", "specification", "Spec 1")

	resp := callDoc(t, env, map[string]any{
		"action": "approve",
		"ids":    []any{id1, id2},
	})

	results, ok := resp["results"].([]any)
	if !ok {
		t.Fatalf("expected results array, got: %v", resp)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for i, r := range results {
		item, _ := r.(map[string]any)
		if item["status"] != "ok" {
			t.Errorf("result[%d] status = %q, want ok", i, item["status"])
		}
		data, _ := item["data"].(map[string]any)
		doc, _ := data["document"].(map[string]any)
		if doc["status"] != "approved" {
			t.Errorf("result[%d] document.status = %q, want approved", i, doc["status"])
		}
	}
}

func TestDocTool_Approve_ReportsEntityTransition(t *testing.T) {
	t.Parallel()

	env := setupDocToolTest(t)
	const featID = "FEAT-CASCADE"

	// Wire a mock hook: feature is in "specifying"; approving its spec
	// document should cascade it to "dev-planning" (spec §30.2 criterion 7).
	mock := &mockDocHook{entityType: "feature", status: "specifying"}
	env.docSvc.SetEntityHook(mock)

	// Register a specification document with an owner so the hook fires.
	writeDocFile(t, env.repoRoot, "work/spec/cascade-spec.md", "# Cascade Spec\n")
	reg := callDoc(t, env, map[string]any{
		"action": "register",
		"path":   "work/spec/cascade-spec.md",
		"type":   "specification",
		"title":  "Cascade Spec",
		"owner":  featID,
	})
	doc, _ := reg["document"].(map[string]any)
	id, _ := doc["id"].(string)
	if id == "" {
		t.Fatal("register: got empty document ID")
	}

	resp := callDoc(t, env, map[string]any{
		"action": "approve",
		"id":     id,
	})

	// The side_effects must contain a status_transition for the owning feature.
	sideEffects, _ := resp["side_effects"].([]any)
	found := false
	for _, se := range sideEffects {
		seMap, _ := se.(map[string]any)
		if seMap["type"] == "status_transition" && seMap["entity_id"] == featID {
			found = true
			if seMap["from_status"] == "" {
				t.Error("side effect missing from_status")
			}
			if seMap["to_status"] == "" {
				t.Error("side effect missing to_status")
			}
			if trigger, _ := seMap["trigger"].(string); !strings.Contains(trigger, id) {
				t.Errorf("trigger %q does not mention document ID %q", trigger, id)
			}
		}
	}
	if !found {
		t.Errorf("expected status_transition side effect for feature %s; side_effects: %v", featID, sideEffects)
	}
}

// TestDocTool_Approve_Batch_WithEntityTransition is a regression test for F2:
// batch doc approval where each document triggers an entity lifecycle transition.
// Before the fix, SignalMutation + ExecuteBatch produced duplicate "side_effects"
// keys in the JSON, causing parsers to see side_effects: [] (empty) instead of
// the real transitions.
func TestDocTool_Approve_Batch_WithEntityTransition(t *testing.T) {
	t.Parallel()

	env := setupDocToolTest(t)
	const featID = "FEAT-CASCADE-BATCH"

	// Wire a mock hook so both doc approvals trigger a feature status cascade.
	mock := &mockDocHook{entityType: "feature", status: "specifying"}
	env.docSvc.SetEntityHook(mock)

	// Register two spec documents with the same owning feature.
	writeDocFile(t, env.repoRoot, "work/spec/batch-spec-1.md", "# Batch Spec 1\n")
	writeDocFile(t, env.repoRoot, "work/spec/batch-spec-2.md", "# Batch Spec 2\n")

	reg1 := callDoc(t, env, map[string]any{
		"action": "register",
		"path":   "work/spec/batch-spec-1.md",
		"type":   "specification",
		"title":  "Batch Spec 1",
		"owner":  featID,
	})
	reg2 := callDoc(t, env, map[string]any{
		"action": "register",
		"path":   "work/spec/batch-spec-2.md",
		"type":   "specification",
		"title":  "Batch Spec 2",
		"owner":  featID,
	})
	id1, _ := reg1["document"].(map[string]any)["id"].(string)
	id2, _ := reg2["document"].(map[string]any)["id"].(string)
	if id1 == "" || id2 == "" {
		t.Fatalf("register: got empty document IDs; id1=%q id2=%q", id1, id2)
	}

	// Approve both in a single batch call.
	resp := callDoc(t, env, map[string]any{
		"action": "approve",
		"ids":    []any{id1, id2},
	})

	// The response must have the standard batch shape.
	results, ok := resp["results"].([]any)
	if !ok || len(results) != 2 {
		t.Fatalf("expected results array with 2 items, got: %v", resp)
	}
	for i, r := range results {
		item, _ := r.(map[string]any)
		if item["status"] != "ok" {
			t.Errorf("results[%d].status = %q, want ok", i, item["status"])
		}
	}

	// The top-level side_effects must be non-empty (contains status_transitions).
	// Before the F2 fix, duplicate JSON keys caused side_effects to appear as [].
	topSideEffects, _ := resp["side_effects"].([]any)
	if len(topSideEffects) == 0 {
		t.Errorf("top-level side_effects is empty — possible duplicate key bug (F2); full response: %v", resp)
	}

	// At least one top-level effect must be a status_transition for the feature.
	found := false
	for _, se := range topSideEffects {
		seMap, _ := se.(map[string]any)
		if seMap["type"] == "status_transition" && seMap["entity_id"] == featID {
			found = true
		}
	}
	if !found {
		t.Errorf("expected status_transition for %s in top-level side_effects; got: %v", featID, topSideEffects)
	}
}

// ─── get ──────────────────────────────────────────────────────────────────────

func TestDocTool_Get_ByID(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	id := registerDoc(t, env, "work/design/getme.md", "design", "Get Me")

	resp := callDoc(t, env, map[string]any{
		"action": "get",
		"id":     id,
	})

	doc, ok := resp["document"].(map[string]any)
	if !ok {
		t.Fatalf("expected document field, got: %v", resp)
	}
	if doc["id"] != id {
		t.Errorf("document.id = %q, want %q", doc["id"], id)
	}
	if doc["title"] != "Get Me" {
		t.Errorf("document.title = %q, want 'Get Me'", doc["title"])
	}
}

func TestDocTool_Get_ByPath(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	const relPath = "work/design/bypath.md"
	id := registerDoc(t, env, relPath, "design", "By Path")

	resp := callDoc(t, env, map[string]any{
		"action": "get",
		"path":   relPath,
	})

	doc, ok := resp["document"].(map[string]any)
	if !ok {
		t.Fatalf("expected document field, got: %v", resp)
	}
	if doc["id"] != id {
		t.Errorf("document.id = %q, want %q", doc["id"], id)
	}
}

func TestDocTool_Get_PathNotFound(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	resp := callDoc(t, env, map[string]any{
		"action": "get",
		"path":   "work/nonexistent.md",
	})

	if _, hasErr := resp["error"]; !hasErr {
		t.Errorf("expected error response, got: %v", resp)
	}
}

func TestDocTool_Get_NeitherIDNorPath(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	resp := callDoc(t, env, map[string]any{
		"action": "get",
	})

	if _, hasErr := resp["error"]; !hasErr {
		t.Errorf("expected error response, got: %v", resp)
	}
}

// ─── content ──────────────────────────────────────────────────────────────────

func TestDocTool_Content(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	const relPath = "work/design/content-test.md"
	const body = "# My Doc\n\nHello world content."
	writeDocFile(t, env.repoRoot, relPath, body)

	// Register directly (not via registerDoc, which would overwrite the file body).
	reg := callDoc(t, env, map[string]any{
		"action": "register",
		"path":   relPath,
		"type":   "design",
		"title":  "Content Test",
	})
	doc, _ := reg["document"].(map[string]any)
	id, _ := doc["id"].(string)
	if id == "" {
		t.Fatal("register: got empty document ID")
	}

	resp := callDoc(t, env, map[string]any{
		"action": "content",
		"id":     id,
	})

	content, _ := resp["content"].(string)
	if content != body {
		t.Errorf("content = %q, want %q", content, body)
	}
	if _, hasDrift := resp["drift"]; hasDrift {
		t.Error("unexpected drift flag on freshly registered document")
	}
}

func TestDocTool_Content_MissingID(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	resp := callDoc(t, env, map[string]any{
		"action": "content",
	})

	if _, hasErr := resp["error"]; !hasErr {
		t.Errorf("expected error response, got: %v", resp)
	}
}

// ─── list ─────────────────────────────────────────────────────────────────────

func TestDocTool_List_All(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	registerDoc(t, env, "work/design/d1.md", "design", "Design 1")
	registerDoc(t, env, "work/spec/s1.md", "specification", "Spec 1")

	resp := callDoc(t, env, map[string]any{
		"action": "list",
	})

	total, _ := resp["total"].(float64)
	if total < 2 {
		t.Errorf("total = %v, want >= 2", total)
	}
	docs, _ := resp["documents"].([]any)
	if len(docs) < 2 {
		t.Errorf("documents count = %d, want >= 2", len(docs))
	}
}

func TestDocTool_List_FilterByType(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	registerDoc(t, env, "work/design/d1.md", "design", "Design 1")
	registerDoc(t, env, "work/spec/s1.md", "specification", "Spec 1")

	resp := callDoc(t, env, map[string]any{
		"action": "list",
		"type":   "design",
	})

	docs, _ := resp["documents"].([]any)
	for i, d := range docs {
		dm, _ := d.(map[string]any)
		if dm["type"] != "design" {
			t.Errorf("documents[%d].type = %q, want design", i, dm["type"])
		}
	}
}

func TestDocTool_List_PendingShorthand(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	id1 := registerDoc(t, env, "work/design/pending.md", "design", "Pending Doc")

	// Approve id1 so it is not pending.
	callDoc(t, env, map[string]any{"action": "approve", "id": id1})

	// Register a second doc that stays draft.
	registerDoc(t, env, "work/spec/still-draft.md", "specification", "Still Draft")

	resp := callDoc(t, env, map[string]any{
		"action":  "list",
		"pending": true,
	})

	docs, _ := resp["documents"].([]any)
	for i, d := range docs {
		dm, _ := d.(map[string]any)
		if dm["status"] != "draft" {
			t.Errorf("documents[%d].status = %q, want draft", i, dm["status"])
		}
	}
	if len(docs) == 0 {
		t.Error("expected at least one pending (draft) document")
	}
}

// ─── gaps ─────────────────────────────────────────────────────────────────────

func TestDocTool_Gaps_AllMissing(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	resp := callDoc(t, env, map[string]any{
		"action":     "gaps",
		"feature_id": "FEAT-NOMATCH",
	})

	gaps, _ := resp["gaps"].([]any)
	// All three expected types (design, specification, dev-plan) are missing.
	if len(gaps) != 3 {
		t.Errorf("gaps count = %d, want 3", len(gaps))
	}
	for i, g := range gaps {
		gm, _ := g.(map[string]any)
		if gm["status"] != "missing" {
			t.Errorf("gaps[%d].status = %q, want missing", i, gm["status"])
		}
	}

	present, _ := resp["present"].([]any)
	if len(present) != 0 {
		t.Errorf("present count = %d, want 0", len(present))
	}
}

func TestDocTool_Gaps_DraftCountsAsGap(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	const featID = "FEAT-DRAFT"

	// Register a design doc owned by the feature but leave it as draft.
	writeDocFile(t, env.repoRoot, "work/design/draft-design.md", "# Draft\n")
	callDoc(t, env, map[string]any{
		"action": "register",
		"path":   "work/design/draft-design.md",
		"type":   "design",
		"title":  "Draft Design",
		"owner":  featID,
	})

	resp := callDoc(t, env, map[string]any{
		"action":     "gaps",
		"feature_id": featID,
	})

	gaps, _ := resp["gaps"].([]any)
	foundDesignGap := false
	for _, g := range gaps {
		gm, _ := g.(map[string]any)
		if gm["type"] == "design" && gm["status"] == "draft" {
			foundDesignGap = true
		}
	}
	if !foundDesignGap {
		t.Errorf("expected draft design to appear in gaps; gaps: %v", gaps)
	}

	present, _ := resp["present"].([]any)
	if len(present) != 0 {
		t.Errorf("present count = %d, want 0 (draft not approved)", len(present))
	}
}

func TestDocTool_Gaps_ApprovedCountsAsPresent(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	const featID = "FEAT-APPROVED"

	// Register and approve a design document.
	writeDocFile(t, env.repoRoot, "work/design/approved.md", "# Approved\n")
	r := callDoc(t, env, map[string]any{
		"action": "register",
		"path":   "work/design/approved.md",
		"type":   "design",
		"title":  "Approved Design",
		"owner":  featID,
	})
	docMap, _ := r["document"].(map[string]any)
	docID, _ := docMap["id"].(string)

	callDoc(t, env, map[string]any{"action": "approve", "id": docID})

	resp := callDoc(t, env, map[string]any{
		"action":     "gaps",
		"feature_id": featID,
	})

	present, _ := resp["present"].([]any)
	foundDesign := false
	for _, p := range present {
		pm, _ := p.(map[string]any)
		if pm["type"] == "design" && pm["status"] == "approved" {
			foundDesign = true
		}
	}
	if !foundDesign {
		t.Errorf("expected approved design in present list; present: %v", present)
	}
}

func TestDocTool_Gaps_MissingFeatureID(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	resp := callDoc(t, env, map[string]any{
		"action": "gaps",
	})

	if _, hasErr := resp["error"]; !hasErr {
		t.Errorf("expected error response, got: %v", resp)
	}
}

func TestDocTool_List_FilterByStatus(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	id1 := registerDoc(t, env, "work/design/approved-doc.md", "design", "Approved Doc")
	registerDoc(t, env, "work/spec/draft-doc.md", "specification", "Draft Doc")

	// Approve one document.
	callDoc(t, env, map[string]any{"action": "approve", "id": id1})

	resp := callDoc(t, env, map[string]any{
		"action": "list",
		"status": "approved",
	})

	docs, _ := resp["documents"].([]any)
	if len(docs) == 0 {
		t.Fatal("expected at least one approved document")
	}
	for i, d := range docs {
		dm, _ := d.(map[string]any)
		if dm["status"] != "approved" {
			t.Errorf("documents[%d].status = %q, want approved", i, dm["status"])
		}
	}
}

// ─── validate ─────────────────────────────────────────────────────────────────

func TestDocTool_Validate_Valid(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	id := registerDoc(t, env, "work/design/valid.md", "design", "Valid Doc")

	resp := callDoc(t, env, map[string]any{
		"action": "validate",
		"id":     id,
	})

	valid, _ := resp["valid"].(bool)
	if !valid {
		t.Errorf("valid = false, want true; issues: %v", resp["issues"])
	}
}

func TestDocTool_Validate_MissingID(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	resp := callDoc(t, env, map[string]any{
		"action": "validate",
	})

	if _, hasErr := resp["error"]; !hasErr {
		t.Errorf("expected error response, got: %v", resp)
	}
}

// ─── supersede ────────────────────────────────────────────────────────────────

func TestDocTool_Validate_WithIssues(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	// Register a document, then delete the file so validation will report drift/missing issues.
	const relPath = "work/design/deleted.md"
	id := registerDoc(t, env, relPath, "design", "Deleted Doc")

	// Delete the underlying file to cause validation issues.
	if err := os.Remove(filepath.Join(env.repoRoot, relPath)); err != nil {
		t.Fatalf("remove file: %v", err)
	}

	resp := callDoc(t, env, map[string]any{
		"action": "validate",
		"id":     id,
	})

	valid, _ := resp["valid"].(bool)
	if valid {
		t.Error("valid = true, want false for document with missing file")
	}
	issues, _ := resp["issues"].([]any)
	if len(issues) == 0 {
		t.Error("expected at least one validation issue for missing file")
	}
}

func TestDocTool_Supersede(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	oldID := registerDoc(t, env, "work/spec/old-spec.md", "specification", "Old Spec")
	callDoc(t, env, map[string]any{"action": "approve", "id": oldID})

	newID := registerDoc(t, env, "work/spec/new-spec.md", "specification", "New Spec")
	callDoc(t, env, map[string]any{"action": "approve", "id": newID})

	resp := callDoc(t, env, map[string]any{
		"action":        "supersede",
		"id":            oldID,
		"superseded_by": newID,
	})

	doc, ok := resp["document"].(map[string]any)
	if !ok {
		t.Fatalf("expected document field, got: %v", resp)
	}
	if doc["status"] != "superseded" {
		t.Errorf("status = %q, want superseded", doc["status"])
	}
	if _, hasSE := resp["side_effects"]; !hasSE {
		t.Error("expected side_effects field on mutation response")
	}
}

func TestDocTool_Supersede_ReportsEntityTransition(t *testing.T) {
	t.Parallel()

	env := setupDocToolTest(t)
	const featID = "FEAT-SUPERSEDE"

	// Wire a mock hook: feature is in "dev-planning"; superseding its spec
	// document should cascade it back to "specifying" (backward lifecycle transition).
	mock := &mockDocHook{entityType: "feature", status: "dev-planning"}
	env.docSvc.SetEntityHook(mock)

	// Register and approve the old spec, then register and approve a new spec.
	writeDocFile(t, env.repoRoot, "work/spec/old-spec.md", "# Old Spec\n")
	reg1 := callDoc(t, env, map[string]any{
		"action": "register",
		"path":   "work/spec/old-spec.md",
		"type":   "specification",
		"title":  "Old Spec",
		"owner":  featID,
	})
	oldDoc, _ := reg1["document"].(map[string]any)
	oldID, _ := oldDoc["id"].(string)
	if oldID == "" {
		t.Fatal("register old spec: got empty document ID")
	}
	callDoc(t, env, map[string]any{"action": "approve", "id": oldID})

	writeDocFile(t, env.repoRoot, "work/spec/new-spec.md", "# New Spec\n")
	reg2 := callDoc(t, env, map[string]any{
		"action": "register",
		"path":   "work/spec/new-spec.md",
		"type":   "specification",
		"title":  "New Spec",
		"owner":  featID,
	})
	newDoc, _ := reg2["document"].(map[string]any)
	newID, _ := newDoc["id"].(string)
	if newID == "" {
		t.Fatal("register new spec: got empty document ID")
	}
	callDoc(t, env, map[string]any{"action": "approve", "id": newID})

	resp := callDoc(t, env, map[string]any{
		"action":        "supersede",
		"id":            oldID,
		"superseded_by": newID,
	})

	// The side_effects must contain a status_transition for the owning feature.
	sideEffects, _ := resp["side_effects"].([]any)
	found := false
	for _, se := range sideEffects {
		seMap, _ := se.(map[string]any)
		if seMap["type"] == "status_transition" && seMap["entity_id"] == featID {
			found = true
			if seMap["from_status"] == "" {
				t.Error("side effect missing from_status")
			}
			if seMap["to_status"] == "" {
				t.Error("side effect missing to_status")
			}
			if trigger, _ := seMap["trigger"].(string); !strings.Contains(trigger, oldID) {
				t.Errorf("trigger %q does not mention superseded document ID %q", trigger, oldID)
			}
		}
	}
	if !found {
		t.Errorf("expected status_transition side effect for feature %s; side_effects: %v", featID, sideEffects)
	}
}

func TestDocTool_Supersede_MissingFields(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	// Missing superseded_by.
	resp := callDoc(t, env, map[string]any{
		"action": "supersede",
		"id":     "DOC-FAKE",
	})
	if _, hasErr := resp["error"]; !hasErr {
		t.Errorf("expected error for missing superseded_by, got: %v", resp)
	}

	// Missing id.
	resp2 := callDoc(t, env, map[string]any{
		"action":        "supersede",
		"superseded_by": "DOC-OTHER",
	})
	if _, hasErr := resp2["error"]; !hasErr {
		t.Errorf("expected error for missing id, got: %v", resp2)
	}
}

// ─── import ───────────────────────────────────────────────────────────────────

func TestDocTool_Import(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	// Create a directory with two markdown files.
	writeDocFile(t, env.repoRoot, "work/design/imp-a.md", "# Import A\n")
	writeDocFile(t, env.repoRoot, "work/design/imp-b.md", "# Import B\n")

	resp := callDoc(t, env, map[string]any{
		"action": "import",
		"path":   filepath.Join(env.repoRoot, "work/design"),
	})

	imported, _ := resp["imported"].(float64)
	if imported < 2 {
		t.Errorf("imported count = %v, want >= 2", imported)
	}
	if _, hasSE := resp["side_effects"]; !hasSE {
		t.Error("expected side_effects field on mutation response")
	}
}

func TestDocTool_Import_Idempotent(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	writeDocFile(t, env.repoRoot, "work/design/idem.md", "# Idempotent\n")

	// First import.
	callDoc(t, env, map[string]any{"action": "import", "path": filepath.Join(env.repoRoot, "work/design")})

	// Second import: already-imported files should be skipped.
	resp := callDoc(t, env, map[string]any{"action": "import", "path": filepath.Join(env.repoRoot, "work/design")})

	skipped, _ := resp["skipped"].([]any)
	if len(skipped) == 0 {
		t.Error("expected at least one skipped file on second import")
	}
}

func TestDocTool_Import_MissingPath(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	resp := callDoc(t, env, map[string]any{
		"action": "import",
	})

	if _, hasErr := resp["error"]; !hasErr {
		t.Errorf("expected error response, got: %v", resp)
	}
}

// ─── unknown action ───────────────────────────────────────────────────────────

func TestDocTool_UnknownAction(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	resp := callDoc(t, env, map[string]any{
		"action": "frobnicate",
	})

	if _, hasErr := resp["error"]; !hasErr {
		t.Fatalf("expected error response for unknown action, got: %v", resp)
	}
}

// ─── refresh ──────────────────────────────────────────────────────────────────

func TestDocRefresh_ChangedDoc(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	// Register a document then modify its underlying file.
	id := registerDoc(t, env, "work/design/refresh-me.md", "design", "Refresh Me")
	writeDocFile(t, env.repoRoot, "work/design/refresh-me.md", "# Refresh Me\n\nRevised content.\n")

	resp := callDoc(t, env, map[string]any{
		"action": "refresh",
		"id":     id,
	})

	if changed, _ := resp["changed"].(bool); !changed {
		t.Errorf("expected changed=true, got: %v", resp)
	}
	if status, _ := resp["status"].(string); status != "draft" {
		t.Errorf("status = %q, want draft", status)
	}
	if _, ok := resp["old_hash"]; !ok {
		t.Errorf("expected old_hash field in response")
	}
	if _, ok := resp["new_hash"]; !ok {
		t.Errorf("expected new_hash field in response")
	}
}

func TestDocRefresh_UnchangedDoc(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	id := registerDoc(t, env, "work/design/unchanged.md", "design", "Unchanged")

	resp := callDoc(t, env, map[string]any{
		"action": "refresh",
		"id":     id,
	})

	if changed, _ := resp["changed"].(bool); changed {
		t.Errorf("expected changed=false, got: %v", resp)
	}
}

// ─── chain ────────────────────────────────────────────────────────────────────

func TestDocChain_ReturnsChain(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	// Register and approve v1.
	v1ID := registerDoc(t, env, "work/spec/chain-v1.md", "specification", "Chain V1")
	callDoc(t, env, map[string]any{"action": "approve", "id": v1ID})

	// Register and approve v2.
	v2ID := registerDoc(t, env, "work/spec/chain-v2.md", "specification", "Chain V2")
	callDoc(t, env, map[string]any{"action": "approve", "id": v2ID})

	// Supersede v1 with v2.
	callDoc(t, env, map[string]any{
		"action":        "supersede",
		"id":            v1ID,
		"superseded_by": v2ID,
	})

	resp := callDoc(t, env, map[string]any{
		"action": "chain",
		"id":     v1ID,
	})

	length, _ := resp["length"].(float64)
	if int(length) != 2 {
		t.Errorf("length = %v, want 2", resp["length"])
	}
	chain, _ := resp["chain"].([]any)
	if len(chain) != 2 {
		t.Fatalf("chain slice length = %d, want 2", len(chain))
	}
	first, _ := chain[0].(map[string]any)
	if first["id"] != v1ID {
		t.Errorf("chain[0].id = %q, want %q", first["id"], v1ID)
	}
	second, _ := chain[1].(map[string]any)
	if second["id"] != v2ID {
		t.Errorf("chain[1].id = %q, want %q", second["id"], v2ID)
	}
}

func TestDocChain_EmptyID(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	resp := callDoc(t, env, map[string]any{
		"action": "chain",
	})

	if _, hasErr := resp["error"]; !hasErr {
		t.Errorf("expected error for missing id, got: %v", resp)
	}
}

// ─── doc audit tests (AC-16 through AC-20) ────────────────────────────────────

// AC-16: doc(action:"audit") returns unregistered files found under default
// document directories. When path is provided, only that directory is scanned.
func TestDocAudit_UnregisteredFiles(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	// Create two unregistered .md files in a spec directory.
	writeDocFile(t, env.repoRoot, "work/spec/feature-a.md", "# Feature A\n")
	writeDocFile(t, env.repoRoot, "work/spec/feature-b.md", "# Feature B\n")

	resp := callDoc(t, env, map[string]any{
		"action": "audit",
		"path":   filepath.Join(env.repoRoot, "work/spec"),
	})

	unregistered, ok := resp["unregistered"].([]any)
	if !ok {
		t.Fatalf("unregistered field missing or wrong type; response: %v", resp)
	}
	if len(unregistered) != 2 {
		t.Errorf("len(unregistered) = %d, want 2", len(unregistered))
	}

	// Each entry must have path and inferred_type fields.
	for i, entry := range unregistered {
		m, ok := entry.(map[string]any)
		if !ok {
			t.Fatalf("unregistered[%d] is not a map", i)
		}
		if _, hasPath := m["path"]; !hasPath {
			t.Errorf("unregistered[%d] missing path field", i)
		}
		if _, hasType := m["inferred_type"]; !hasType {
			t.Errorf("unregistered[%d] missing inferred_type field", i)
		}
	}
}

// AC-17: doc(action:"audit") returns missing records whose files no longer
// exist on disk.
func TestDocAudit_MissingRecords(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	// Register a doc whose file we will then delete.
	relPath := "work/spec/will-be-deleted.md"
	writeDocFile(t, env.repoRoot, relPath, "# Will Be Deleted\n")
	registerDoc(t, env, relPath, "specification", "Will Be Deleted")

	// Remove the file from disk.
	if err := os.Remove(filepath.Join(env.repoRoot, relPath)); err != nil {
		t.Fatalf("remove file: %v", err)
	}

	resp := callDoc(t, env, map[string]any{
		"action": "audit",
		"path":   filepath.Join(env.repoRoot, "work/spec"),
	})

	missing, ok := resp["missing"].([]any)
	if !ok {
		t.Fatalf("missing field missing or wrong type; response: %v", resp)
	}
	if len(missing) != 1 {
		t.Errorf("len(missing) = %d, want 1", len(missing))
	}
	m, ok := missing[0].(map[string]any)
	if !ok {
		t.Fatal("missing[0] is not a map")
	}
	if _, hasPath := m["path"]; !hasPath {
		t.Error("missing[0] missing path field")
	}
	if _, hasDocID := m["doc_id"]; !hasDocID {
		t.Error("missing[0] missing doc_id field")
	}
}

// AC-18: Each unregistered file entry includes an inferred_type based on its
// directory path.
func TestDocAudit_InferredType(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	writeDocFile(t, env.repoRoot, "work/spec/inferred.md", "# Inferred\n")

	resp := callDoc(t, env, map[string]any{
		"action": "audit",
		"path":   filepath.Join(env.repoRoot, "work/spec"),
	})

	unregistered, _ := resp["unregistered"].([]any)
	if len(unregistered) == 0 {
		t.Fatal("expected at least one unregistered file")
	}
	entry, _ := unregistered[0].(map[string]any)
	inferredType, _ := entry["inferred_type"].(string)
	if inferredType == "" {
		t.Errorf("inferred_type is empty for work/spec file; want non-empty (e.g. 'specification')")
	}
}

// AC-19: The path parameter scopes the scan to the specified directory;
// files outside that directory are not reported.
func TestDocAudit_PathScopesResults(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	// Create files in two different directories.
	writeDocFile(t, env.repoRoot, "work/spec/in-scope.md", "# In Scope\n")
	writeDocFile(t, env.repoRoot, "work/design/out-of-scope.md", "# Out Of Scope\n")

	// Audit only the spec directory.
	resp := callDoc(t, env, map[string]any{
		"action": "audit",
		"path":   filepath.Join(env.repoRoot, "work/spec"),
	})

	unregistered, _ := resp["unregistered"].([]any)
	if len(unregistered) != 1 {
		t.Errorf("len(unregistered) = %d, want 1 (only spec dir scanned)", len(unregistered))
	}
	if len(unregistered) == 1 {
		entry, _ := unregistered[0].(map[string]any)
		path, _ := entry["path"].(string)
		if strings.Contains(path, "design") {
			t.Errorf("audit returned a file from 'design' dir; want only 'spec' dir: %s", path)
		}
	}
}

// AC-20: Files that are already registered are counted in summary.registered
// but not individually listed when include_registered is false.
func TestDocAudit_RegisteredCountedNotListed(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	// Register one file, leave one unregistered.
	writeDocFile(t, env.repoRoot, "work/spec/registered.md", "# Registered\n")
	registerDoc(t, env, "work/spec/registered.md", "specification", "Registered")
	writeDocFile(t, env.repoRoot, "work/spec/unregistered.md", "# Unregistered\n")

	resp := callDoc(t, env, map[string]any{
		"action": "audit",
		"path":   filepath.Join(env.repoRoot, "work/spec"),
	})

	// summary.registered must be 1.
	summary, _ := resp["summary"].(map[string]any)
	if summary == nil {
		t.Fatal("summary field missing from audit response")
	}
	registered, _ := summary["registered"].(float64)
	if registered != 1 {
		t.Errorf("summary.registered = %v, want 1", registered)
	}

	// The top-level "registered" array must be absent when include_registered is false.
	if _, hasRegistered := resp["registered"]; hasRegistered {
		t.Error("'registered' array must be absent when include_registered is false/omitted")
	}

	// When include_registered is true, the array must be present.
	resp2 := callDoc(t, env, map[string]any{
		"action":             "audit",
		"path":               filepath.Join(env.repoRoot, "work/spec"),
		"include_registered": true,
	})
	if _, hasRegistered := resp2["registered"]; !hasRegistered {
		t.Error("'registered' array must be present when include_registered is true")
	}
}

// Invariant: summary.registered + summary.unregistered == summary.total_on_disk.
func TestDocAudit_SummaryInvariant(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	writeDocFile(t, env.repoRoot, "work/spec/reg.md", "# Registered\n")
	registerDoc(t, env, "work/spec/reg.md", "specification", "Registered")
	writeDocFile(t, env.repoRoot, "work/spec/unreg1.md", "# Unregistered 1\n")
	writeDocFile(t, env.repoRoot, "work/spec/unreg2.md", "# Unregistered 2\n")

	resp := callDoc(t, env, map[string]any{
		"action": "audit",
		"path":   filepath.Join(env.repoRoot, "work/spec"),
	})

	summary, _ := resp["summary"].(map[string]any)
	if summary == nil {
		t.Fatal("summary missing")
	}
	total, _ := summary["total_on_disk"].(float64)
	reg, _ := summary["registered"].(float64)
	unreg, _ := summary["unregistered"].(float64)
	if int(reg)+int(unreg) != int(total) {
		t.Errorf("invariant violated: registered(%v) + unregistered(%v) != total_on_disk(%v)", reg, unreg, total)
	}
	if total != 3 {
		t.Errorf("total_on_disk = %v, want 3", total)
	}
}

// ─── doc import dry-run tests (AC-21 through AC-25) ──────────────────────────

// AC-21 + AC-23: dry_run returns files that would be imported with inferred metadata.
func TestDocImport_DryRun_ReturnsWouldImport(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	writeDocFile(t, env.repoRoot, "work/spec/dry-a.md", "# Dry A\n")
	writeDocFile(t, env.repoRoot, "work/spec/dry-b.md", "# Dry B\n")

	resp := callDoc(t, env, map[string]any{
		"action":  "import",
		"path":    filepath.Join(env.repoRoot, "work/spec"),
		"dry_run": true,
	})

	wouldImport, ok := resp["would_import"].([]any)
	if !ok {
		t.Fatalf("would_import field missing or wrong type; response: %v", resp)
	}
	if len(wouldImport) != 2 {
		t.Errorf("len(would_import) = %d, want 2", len(wouldImport))
	}

	// AC-23: each entry must include type, title, owner.
	for i, entry := range wouldImport {
		m, ok := entry.(map[string]any)
		if !ok {
			t.Fatalf("would_import[%d] is not a map", i)
		}
		if _, hasPath := m["path"]; !hasPath {
			t.Errorf("would_import[%d] missing path", i)
		}
		if _, hasType := m["type"]; !hasType {
			t.Errorf("would_import[%d] missing type", i)
		}
		if _, hasTitle := m["title"]; !hasTitle {
			t.Errorf("would_import[%d] missing title", i)
		}
		if _, hasOwner := m["owner"]; !hasOwner {
			t.Errorf("would_import[%d] missing owner", i)
		}
	}
}

// AC-22: In dry-run mode, no document records are created in the store.
func TestDocImport_DryRun_NoStoreRecordsCreated(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	writeDocFile(t, env.repoRoot, "work/spec/no-store.md", "# No Store\n")

	callDoc(t, env, map[string]any{
		"action":  "import",
		"path":    filepath.Join(env.repoRoot, "work/spec"),
		"dry_run": true,
	})

	// Query the store — must be empty.
	resp := callDoc(t, env, map[string]any{
		"action": "list",
	})
	docs, _ := resp["documents"].([]any)
	if len(docs) != 0 {
		t.Errorf("store has %d records after dry-run import, want 0", len(docs))
	}
}

// AC-24: Files that would be skipped (already registered) are listed in
// would_skip with reason "already registered".
func TestDocImport_DryRun_AlreadyRegisteredSkipped(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	// Register one file, leave one unregistered.
	writeDocFile(t, env.repoRoot, "work/spec/already-reg.md", "# Already Registered\n")
	registerDoc(t, env, "work/spec/already-reg.md", "specification", "Already Registered")
	writeDocFile(t, env.repoRoot, "work/spec/not-reg.md", "# Not Registered\n")

	resp := callDoc(t, env, map[string]any{
		"action":  "import",
		"path":    filepath.Join(env.repoRoot, "work/spec"),
		"dry_run": true,
	})

	wouldSkip, ok := resp["would_skip"].([]any)
	if !ok {
		t.Fatalf("would_skip field missing or wrong type; response: %v", resp)
	}
	if len(wouldSkip) != 1 {
		t.Errorf("len(would_skip) = %d, want 1", len(wouldSkip))
	}
	entry, _ := wouldSkip[0].(map[string]any)
	reason, _ := entry["reason"].(string)
	if reason != "already registered" {
		t.Errorf("would_skip[0].reason = %q, want \"already registered\"", reason)
	}

	// The unregistered file should appear in would_import.
	wouldImport, _ := resp["would_import"].([]any)
	if len(wouldImport) != 1 {
		t.Errorf("len(would_import) = %d, want 1", len(wouldImport))
	}
}

// AC-25: When dry_run is false, behaviour is unchanged (live import runs).
func TestDocImport_DryRunFalse_LiveBehaviourUnchanged(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	writeDocFile(t, env.repoRoot, "work/spec/live.md", "# Live Import\n")

	resp := callDoc(t, env, map[string]any{
		"action":  "import",
		"path":    filepath.Join(env.repoRoot, "work/spec"),
		"dry_run": false,
	})

	// Live import response must have "imported" field, not "would_import".
	if _, hasImported := resp["imported"]; !hasImported {
		t.Errorf("expected 'imported' field for live import (dry_run=false), got: %v", resp)
	}
	if _, hasWouldImport := resp["would_import"]; hasWouldImport {
		t.Error("'would_import' must not be present for live import (dry_run=false)")
	}

	// Verify the record was actually created in the store.
	listResp := callDoc(t, env, map[string]any{"action": "list"})
	docs, _ := listResp["documents"].([]any)
	if len(docs) == 0 {
		t.Error("expected at least one document record after live import")
	}
}

// AC-25: When dry_run is absent, behaviour is unchanged (live import runs).
func TestDocImport_NoDryRun_LiveBehaviourUnchanged(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	writeDocFile(t, env.repoRoot, "work/spec/live2.md", "# Live Import 2\n")

	resp := callDoc(t, env, map[string]any{
		"action": "import",
		"path":   filepath.Join(env.repoRoot, "work/spec"),
		// dry_run absent
	})

	if _, hasImported := resp["imported"]; !hasImported {
		t.Errorf("expected 'imported' field when dry_run absent, got: %v", resp)
	}
}

// Dry-run summary counts must equal array lengths.
func TestDocImport_DryRun_SummaryCounts(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	writeDocFile(t, env.repoRoot, "work/spec/s1.md", "# S1\n")
	writeDocFile(t, env.repoRoot, "work/spec/s2.md", "# S2\n")
	registerDoc(t, env, "work/spec/s1.md", "specification", "S1")

	resp := callDoc(t, env, map[string]any{
		"action":  "import",
		"path":    filepath.Join(env.repoRoot, "work/spec"),
		"dry_run": true,
	})

	summary, _ := resp["summary"].(map[string]any)
	if summary == nil {
		t.Fatal("summary missing from dry-run response")
	}
	wouldImportCount, _ := summary["would_import"].(float64)
	wouldSkipCount, _ := summary["would_skip"].(float64)

	wouldImport, _ := resp["would_import"].([]any)
	wouldSkip, _ := resp["would_skip"].([]any)

	if int(wouldImportCount) != len(wouldImport) {
		t.Errorf("summary.would_import=%v != len(would_import)=%d", wouldImportCount, len(wouldImport))
	}
	if int(wouldSkipCount) != len(wouldSkip) {
		t.Errorf("summary.would_skip=%v != len(would_skip)=%d", wouldSkipCount, len(wouldSkip))
	}
}

// ─── Auto-commit injection tests (F03, AC-A12, AC-A13) ───────────────────────

// TestDocTool_Register_AutoCommit_MessageFormat verifies that docCommitPathsFunc
// is called with the correct "workflow(<id>): register <type>" format.
func TestDocTool_Register_AutoCommit_MessageFormat(t *testing.T) {
	// Not parallel: modifies package-level docCommitPathsFunc.
	env := setupDocToolTest(t)
	writeDocFile(t, env.repoRoot, "work/spec/msg-fmt.md", "# Spec\n\nContent.")

	var capturedMsg string
	savedFn := docCommitPathsFunc
	docCommitPathsFunc = func(repoRoot, message string, extraPaths ...string) (bool, error) {
		capturedMsg = message
		return false, nil
	}
	defer func() { docCommitPathsFunc = savedFn }()

	callDoc(t, env, map[string]any{
		"action": "register",
		"path":   "work/spec/msg-fmt.md",
		"type":   "specification",
		"title":  "Message Format Spec",
	})

	if !strings.HasPrefix(capturedMsg, "workflow(") {
		t.Errorf("commit message = %q; want prefix \"workflow(\"", capturedMsg)
	}
	if !strings.Contains(capturedMsg, "): register spec") {
		t.Errorf("commit message = %q; want it to contain \": register spec\"", capturedMsg)
	}
}

// TestDocTool_Register_AutoCommit_FailureDoesNotBlockResult verifies that a
// commit failure does not prevent the register result from being returned
// (best-effort semantics, AC-A13).
func TestDocTool_Register_AutoCommit_FailureDoesNotBlockResult(t *testing.T) {
	// Not parallel: modifies package-level docCommitPathsFunc.
	env := setupDocToolTest(t)
	writeDocFile(t, env.repoRoot, "work/spec/commit-fail.md", "# Spec\n\nContent.")

	savedFn := docCommitPathsFunc
	docCommitPathsFunc = func(repoRoot, message string, extraPaths ...string) (bool, error) {
		return false, fmt.Errorf("simulated git commit failure")
	}
	defer func() { docCommitPathsFunc = savedFn }()

	resp := callDoc(t, env, map[string]any{
		"action": "register",
		"path":   "work/spec/commit-fail.md",
		"type":   "specification",
		"title":  "Commit Fail Spec",
	})

	// Must not contain a top-level error — commit failure is non-blocking.
	if _, hasErr := resp["error"]; hasErr {
		t.Errorf("register returned error after commit failure; should proceed normally: %v", resp["error"])
	}
	doc, ok := resp["document"].(map[string]any)
	if !ok {
		t.Fatalf("expected document field in response, got: %v", resp)
	}
	if doc["status"] != "draft" {
		t.Errorf("document status = %q, want draft", doc["status"])
	}
}

// TestDocTool_Approve_AutoCommit_MessageFormat verifies that docCommitFunc is
// called with the correct "workflow(<id>): approve <type>" format.
func TestDocTool_Approve_AutoCommit_MessageFormat(t *testing.T) {
	// Not parallel: modifies package-level docCommitFunc.
	env := setupDocToolTest(t)

	docID := registerDoc(t, env, "work/design/approve-fmt.md", "design", "Approve Format Design")

	var capturedMsg string
	savedFn := docCommitFunc
	docCommitFunc = func(repoRoot, message string) (bool, error) {
		capturedMsg = message
		return false, nil
	}
	defer func() { docCommitFunc = savedFn }()

	callDoc(t, env, map[string]any{
		"action": "approve",
		"id":     docID,
	})

	if !strings.HasPrefix(capturedMsg, "workflow(") {
		t.Errorf("commit message = %q; want prefix \"workflow(\"", capturedMsg)
	}
	if !strings.Contains(capturedMsg, "): approve design") {
		t.Errorf("commit message = %q; want it to contain \": approve design\"", capturedMsg)
	}
}

// TestDocTool_Approve_AutoCommit_FailureDoesNotBlockResult verifies that a
// commit failure after approve does not prevent the result from being returned.
func TestDocTool_Approve_AutoCommit_FailureDoesNotBlockResult(t *testing.T) {
	// Not parallel: modifies package-level docCommitFunc.
	env := setupDocToolTest(t)

	docID := registerDoc(t, env, "work/design/approve-fail.md", "design", "Approve Fail Design")

	savedFn := docCommitFunc
	docCommitFunc = func(repoRoot, message string) (bool, error) {
		return false, fmt.Errorf("simulated git commit failure")
	}
	defer func() { docCommitFunc = savedFn }()

	resp := callDoc(t, env, map[string]any{
		"action": "approve",
		"id":     docID,
	})

	if _, hasErr := resp["error"]; hasErr {
		t.Errorf("approve returned error after commit failure; should proceed normally: %v", resp["error"])
	}
	doc, ok := resp["document"].(map[string]any)
	if !ok {
		t.Fatalf("expected document field in response, got: %v", resp)
	}
	if doc["status"] != "approved" {
		t.Errorf("document status = %q, want approved", doc["status"])
	}
}

// ─── gaps: plan-level document inheritance ────────────────────────────────────

// callDocWithEntitySvc invokes the doc tool with an entitySvc wired in (for
// inheritance tests) and returns the parsed JSON map.
func callDocWithEntitySvc(t *testing.T, env *docToolEnv, entitySvc *service.EntityService, args map[string]any) map[string]any {
	t.Helper()
	tool := docTool(env.docSvc, nil, entitySvc)
	req := makeRequest(args)
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("doc handler error: %v", err)
	}
	text := extractText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("unmarshal response: %v\nraw: %s", err, text)
	}
	return parsed
}

// setupPlanFeature creates a plan record and a feature under it, returning
// the plan ID and feature ID. The feature's State["parent"] will point to the plan.
// Uses createEntityTestPlan and createEntityTestFeature from entity_tool_test.go
// (same package), which handle the storage layer directly.
func setupPlanFeature(t *testing.T, entitySvc *service.EntityService) (planID, featureID string) {
	t.Helper()
	planID = createEntityTestPlan(t, entitySvc, "gap-test-plan")
	featureID = createEntityTestFeature(t, entitySvc, planID, "gap-test-feature")
	return planID, featureID
}

func TestDocTool_Gaps_PlanDocInherited(t *testing.T) {
	t.Parallel()

	env := setupDocToolTest(t)
	entitySvc := service.NewEntityService(t.TempDir())
	planID, featureID := setupPlanFeature(t, entitySvc)

	// Register and approve a specification doc owned by the plan.
	writeDocFile(t, env.repoRoot, "work/spec/plan-spec.md", "# Plan Spec\n\nContent.")
	regResp := callDoc(t, env, map[string]any{
		"action": "register",
		"path":   "work/spec/plan-spec.md",
		"type":   "specification",
		"title":  "Plan Specification",
		"owner":  planID,
	})
	specDoc, _ := regResp["document"].(map[string]any)
	specID, _ := specDoc["id"].(string)
	if specID == "" {
		t.Fatalf("register plan spec failed; response: %v", regResp)
	}
	callDoc(t, env, map[string]any{"action": "approve", "id": specID})

	// The feature has no spec of its own. Gaps action should inherit from plan.
	resp := callDocWithEntitySvc(t, env, entitySvc, map[string]any{
		"action":     "gaps",
		"feature_id": featureID,
	})

	present, _ := resp["present"].([]any)
	foundInherited := false
	for _, p := range present {
		pm, _ := p.(map[string]any)
		if pm["type"] == "spec" && pm["status"] == "approved" {
			inherited, _ := pm["inherited"].(bool)
			if inherited {
				foundInherited = true
			}
		}
	}
	if !foundInherited {
		t.Errorf("expected inherited specification in present list; present: %v, gaps: %v", present, resp["gaps"])
	}

	// The inherited spec must NOT appear in gaps.
	gaps, _ := resp["gaps"].([]any)
	for _, g := range gaps {
		gm, _ := g.(map[string]any)
		if gm["type"] == "spec" {
			t.Errorf("spec should not appear in gaps when inherited from plan; gaps: %v", gaps)
		}
	}
}

func TestDocTool_Gaps_DraftPlanDoc_NotInherited(t *testing.T) {
	t.Parallel()

	env := setupDocToolTest(t)
	entitySvc := service.NewEntityService(t.TempDir())
	planID, featureID := setupPlanFeature(t, entitySvc)

	// Register a design doc for the plan but leave it as DRAFT (not approved).
	writeDocFile(t, env.repoRoot, "work/design/plan-design.md", "# Plan Design\n\nContent.")
	regResp := callDoc(t, env, map[string]any{
		"action": "register",
		"path":   "work/design/plan-design.md",
		"type":   "design",
		"title":  "Plan Design (draft)",
		"owner":  planID,
	})
	designDoc, _ := regResp["document"].(map[string]any)
	designID, _ := designDoc["id"].(string)
	if designID == "" {
		t.Fatalf("register plan design failed; response: %v", regResp)
	}
	// Do NOT approve — leave as draft.

	resp := callDocWithEntitySvc(t, env, entitySvc, map[string]any{
		"action":     "gaps",
		"feature_id": featureID,
	})

	// Draft plan doc must NOT satisfy inheritance — design should still be a gap.
	gaps, _ := resp["gaps"].([]any)
	foundDesignGap := false
	for _, g := range gaps {
		gm, _ := g.(map[string]any)
		if gm["type"] == "design" {
			foundDesignGap = true
		}
	}
	if !foundDesignGap {
		t.Errorf("expected design to appear in gaps when plan doc is draft; gaps: %v", gaps)
	}

	// And it must NOT appear as inherited in present.
	present, _ := resp["present"].([]any)
	for _, p := range present {
		pm, _ := p.(map[string]any)
		if pm["type"] == "design" {
			t.Errorf("design should not appear in present when plan doc is draft; present: %v", present)
		}
	}
}

func TestDocTool_Gaps_FeatureDocTakesPrecedenceOverPlan(t *testing.T) {
	t.Parallel()

	env := setupDocToolTest(t)
	entitySvc := service.NewEntityService(t.TempDir())
	planID, featureID := setupPlanFeature(t, entitySvc)

	// Register and approve a design for the plan.
	writeDocFile(t, env.repoRoot, "work/design/plan-design.md", "# Plan Design\n")
	planRegResp := callDoc(t, env, map[string]any{
		"action": "register",
		"path":   "work/design/plan-design.md",
		"type":   "design",
		"title":  "Plan Design",
		"owner":  planID,
	})
	planDesignDoc, _ := planRegResp["document"].(map[string]any)
	planDesignID, _ := planDesignDoc["id"].(string)
	callDoc(t, env, map[string]any{"action": "approve", "id": planDesignID})

	// Register and approve a design for the feature itself.
	writeDocFile(t, env.repoRoot, "work/design/feat-design.md", "# Feature Design\n")
	featRegResp := callDoc(t, env, map[string]any{
		"action": "register",
		"path":   "work/design/feat-design.md",
		"type":   "design",
		"title":  "Feature Design",
		"owner":  featureID,
	})
	featDesignDoc, _ := featRegResp["document"].(map[string]any)
	featDesignID, _ := featDesignDoc["id"].(string)
	callDoc(t, env, map[string]any{"action": "approve", "id": featDesignID})

	resp := callDocWithEntitySvc(t, env, entitySvc, map[string]any{
		"action":     "gaps",
		"feature_id": featureID,
	})

	present, _ := resp["present"].([]any)
	var foundDesign map[string]any
	for _, p := range present {
		pm, _ := p.(map[string]any)
		if pm["type"] == "design" {
			foundDesign = pm
		}
	}
	if foundDesign == nil {
		t.Fatalf("expected design in present list; present: %v", present)
	}
	// Must use the feature's own doc, not the inherited one.
	if id, _ := foundDesign["id"].(string); id != featDesignID {
		t.Errorf("present design id = %q, want feature's own doc %q", id, featDesignID)
	}
	if inherited, _ := foundDesign["inherited"].(bool); inherited {
		t.Errorf("feature's own approved doc should not be marked inherited")
	}
}

func TestDocTool_Gaps_NoParentPlan_NoInheritance(t *testing.T) {
	t.Parallel()

	env := setupDocToolTest(t)
	entitySvc := service.NewEntityService(t.TempDir())

	// Feature with no parent plan — entity lookup will return an error or empty parent.
	const featureID = "FEAT-ORPHAN"

	resp := callDocWithEntitySvc(t, env, entitySvc, map[string]any{
		"action":     "gaps",
		"feature_id": featureID,
	})

	// All three types should be missing gaps with no inheritance.
	gaps, _ := resp["gaps"].([]any)
	if len(gaps) != 3 {
		t.Errorf("expected 3 gaps for orphan feature, got %d; gaps: %v", len(gaps), gaps)
	}
	present, _ := resp["present"].([]any)
	if len(present) != 0 {
		t.Errorf("expected 0 present for orphan feature, got %d; present: %v", len(present), present)
	}
}

// TestDocTool_Gaps_FeatureDraftDocBlocksPlanInheritance verifies AC-3 of the
// document-inheritance spec: a feature's own draft document takes precedence
// over the parent plan's approved document. The feature's draft appears in
// gaps; the plan's approved doc is NOT inherited.
func TestDocTool_Gaps_FeatureDraftDocBlocksPlanInheritance(t *testing.T) {
	t.Parallel()

	env := setupDocToolTest(t)
	entitySvc := service.NewEntityService(t.TempDir())
	planID, featureID := setupPlanFeature(t, entitySvc)

	// Register and approve a specification for the plan.
	writeDocFile(t, env.repoRoot, "work/spec/plan-spec.md", "# Plan Spec\n\nContent.")
	planRegResp := callDoc(t, env, map[string]any{
		"action": "register",
		"path":   "work/spec/plan-spec.md",
		"type":   "specification",
		"title":  "Plan Specification",
		"owner":  planID,
	})
	planSpecDoc, _ := planRegResp["document"].(map[string]any)
	planSpecID, _ := planSpecDoc["id"].(string)
	if planSpecID == "" {
		t.Fatalf("register plan spec failed; response: %v", planRegResp)
	}
	callDoc(t, env, map[string]any{"action": "approve", "id": planSpecID})

	// Register (but do NOT approve) a specification for the feature — it stays draft.
	writeDocFile(t, env.repoRoot, "work/spec/feat-spec.md", "# Feature Spec Draft\n\nContent.")
	featRegResp := callDoc(t, env, map[string]any{
		"action": "register",
		"path":   "work/spec/feat-spec.md",
		"type":   "specification",
		"title":  "Feature Specification Draft",
		"owner":  featureID,
	})
	featSpecDoc, _ := featRegResp["document"].(map[string]any)
	featSpecID, _ := featSpecDoc["id"].(string)
	if featSpecID == "" {
		t.Fatalf("register feature spec failed; response: %v", featRegResp)
	}
	// Intentionally not approving — feature spec remains draft.

	resp := callDocWithEntitySvc(t, env, entitySvc, map[string]any{
		"action":     "gaps",
		"feature_id": featureID,
	})

	// The feature's draft spec must appear in gaps (not present), with the
	// feature's own doc ID — the plan's approved spec must NOT be inherited.
	gaps, _ := resp["gaps"].([]any)
	foundSpecGap := false
	for _, g := range gaps {
		gm, _ := g.(map[string]any)
		if gm["type"] == "spec" {
			foundSpecGap = true
			// The gap entry must reference the feature's own draft doc.
			if id, _ := gm["id"].(string); id != featSpecID {
				t.Errorf("gap spec id = %q, want feature's own draft doc %q", id, featSpecID)
			}
		}
	}
	if !foundSpecGap {
		t.Errorf("expected feature's draft spec in gaps; gaps: %v, present: %v", gaps, resp["present"])
	}

	// The plan's approved spec must NOT appear as inherited in present.
	present, _ := resp["present"].([]any)
	for _, p := range present {
		pm, _ := p.(map[string]any)
		if pm["type"] == "spec" {
			inherited, _ := pm["inherited"].(bool)
			if inherited {
				t.Errorf("plan's approved spec should not be inherited when feature has its own draft; present: %v", present)
			}
		}
	}
}

func TestDocTool_Register_Single_ClassificationNudge(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)
	writeDocFile(t, env.repoRoot, "work/spec/nudge.md", "# Nudge\n\nSpec content.")

	resp := callDoc(t, env, map[string]any{
		"action": "register",
		"path":   "work/spec/nudge.md",
		"type":   "specification",
		"title":  "Nudge Specification",
	})

	nudge, ok := resp["classification_nudge"].(map[string]any)
	if !ok || nudge == nil {
		t.Fatalf("expected classification_nudge object field, got: %v", resp)
	}

	// Nudge must have message, content_hash, and outline keys.
	msg, _ := nudge["message"].(string)
	if msg == "" {
		t.Errorf("expected classification_nudge.message to be non-empty")
	}
	if _, ok := nudge["content_hash"]; !ok {
		t.Errorf("expected classification_nudge.content_hash to be present")
	}
	if _, ok := nudge["outline"]; !ok {
		t.Errorf("expected classification_nudge.outline to be present")
	}

	// Message must contain the actual document ID.
	doc, _ := resp["document"].(map[string]any)
	docID, _ := doc["id"].(string)
	if docID == "" {
		t.Fatal("expected document.id to be set")
	}
	if !strings.Contains(msg, docID) {
		t.Errorf("nudge.message %q does not contain document ID %q", msg, docID)
	}
}

func TestDocTool_Register_Batch_ClassificationNudge(t *testing.T) {
	t.Parallel()
	env := setupDocToolTest(t)

	writeDocFile(t, env.repoRoot, "work/design/a.md", "# A\n")
	writeDocFile(t, env.repoRoot, "work/spec/b.md", "# B\n")

	resp := callDoc(t, env, map[string]any{
		"action": "register",
		"documents": []any{
			map[string]any{"path": "work/design/a.md", "type": "design", "title": "Doc A"},
			map[string]any{"path": "work/spec/b.md", "type": "specification", "title": "Doc B"},
		},
	})

	results, ok := resp["results"].([]any)
	if !ok || len(results) != 2 {
		t.Fatalf("expected 2 results, got: %v", resp)
	}

	for i, r := range results {
		item, _ := r.(map[string]any)
		data, _ := item["data"].(map[string]any)
		if data == nil {
			t.Fatalf("result[%d] missing data field: %v", i, item)
		}

		nudge, ok := data["classification_nudge"].(map[string]any)
		if !ok || nudge == nil {
			t.Errorf("result[%d] missing classification_nudge object, got data: %v", i, data)
			continue
		}
		msg, _ := nudge["message"].(string)
		if msg == "" {
			t.Errorf("result[%d] classification_nudge.message is empty", i)
		}
		if _, ok := nudge["content_hash"]; !ok {
			t.Errorf("result[%d] classification_nudge missing content_hash", i)
		}
		if _, ok := nudge["outline"]; !ok {
			t.Errorf("result[%d] classification_nudge missing outline", i)
		}

		doc, _ := data["document"].(map[string]any)
		docID, _ := doc["id"].(string)
		if docID == "" {
			t.Errorf("result[%d] document.id is empty", i)
			continue
		}
		if !strings.Contains(msg, docID) {
			t.Errorf("result[%d] nudge.message %q does not contain document ID %q", i, msg, docID)
		}
	}
}
