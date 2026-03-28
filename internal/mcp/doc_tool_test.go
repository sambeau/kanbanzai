package mcp

import (
	"context"
	"encoding/json"
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
	return &docToolEnv{
		docSvc:   service.NewDocumentService(stateRoot, repoRoot),
		intelSvc: service.NewIntelligenceService(indexRoot, repoRoot),
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
	tool := docTool(env.docSvc)
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
	if doc["type"] != "specification" {
		t.Errorf("type = %q, want specification", doc["type"])
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
