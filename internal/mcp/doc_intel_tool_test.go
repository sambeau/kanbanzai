package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sambeau/kanbanzai/internal/service"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

type docIntelFindEnv struct {
	intelSvc     *service.IntelligenceService
	knowledgeSvc *service.KnowledgeService
	stateRoot    string
}

func setupDocIntelFind(t *testing.T) *docIntelFindEnv {
	t.Helper()
	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	indexRoot := filepath.Join(t.TempDir(), "index")
	return &docIntelFindEnv{
		intelSvc:     service.NewIntelligenceService(indexRoot, repoRoot),
		knowledgeSvc: service.NewKnowledgeService(stateRoot),
		stateRoot:    stateRoot,
	}
}

func callDocIntelFind(t *testing.T, env *docIntelFindEnv, entityID string) map[string]any {
	t.Helper()
	tool := docIntelTool(env.intelSvc, nil, env.knowledgeSvc)
	req := makeRequest(map[string]any{
		"action":    "find",
		"entity_id": entityID,
	})
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	text := extractText(t, result)
	var out map[string]any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("unmarshal response: %v (text=%q)", err, text)
	}
	return out
}

func contributeKnowledge(t *testing.T, svc *service.KnowledgeService, topic, content, scope, learnedFrom string, tags []string) string {
	t.Helper()
	rec, _, err := svc.Contribute(service.ContributeInput{
		Topic:       topic,
		Content:     content,
		Scope:       scope,
		Tier:        3,
		LearnedFrom: learnedFrom,
		CreatedBy:   "tester",
		Tags:        tags,
	})
	if err != nil {
		t.Fatalf("contribute knowledge (%q): %v", topic, err)
	}
	return rec.ID
}

func writeKnowledgeFile(t *testing.T, stateRoot, id, topic, content, scope, learnedFrom, status string, tags []string) {
	t.Helper()
	dir := filepath.Join(stateRoot, "knowledge")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir knowledge: %v", err)
	}
	tagsYAML := ""
	for _, tag := range tags {
		tagsYAML += "\n  - " + tag
	}
	learnedFromYAML := ""
	if learnedFrom != "" {
		learnedFromYAML = "\nlearned_from: " + learnedFrom
	}
	yml := "id: " + id + "\n" +
		"topic: " + topic + "\n" +
		"content: " + content + "\n" +
		"scope: " + scope + "\n" +
		"status: " + status + "\n" +
		"tier: 3\n" +
		"confidence: 0.5\n" +
		"use_count: 0\n" +
		"miss_count: 0\n" +
		learnedFromYAML +
		"\ntags:" + tagsYAML + "\n"
	path := filepath.Join(dir, id+".yaml")
	if err := os.WriteFile(path, []byte(yml), 0o644); err != nil {
		t.Fatalf("write knowledge file: %v", err)
	}
}

// ─── tests ────────────────────────────────────────────────────────────────────

// TestDocIntelFind_EntityID_BackwardCompat verifies that existing response fields
// (search_type, entity_id, count, matches) are present and unchanged.
func TestDocIntelFind_EntityID_BackwardCompat(t *testing.T) {
	t.Parallel()
	env := setupDocIntelFind(t)

	out := callDocIntelFind(t, env, "FEAT-01TESTBACKCOMPAT001")

	if got, _ := out["search_type"].(string); got != "entity_id" {
		t.Errorf("search_type = %q, want %q", got, "entity_id")
	}
	if got, _ := out["entity_id"].(string); got != "FEAT-01TESTBACKCOMPAT001" {
		t.Errorf("entity_id = %q, want %q", got, "FEAT-01TESTBACKCOMPAT001")
	}
	if _, ok := out["count"]; !ok {
		t.Error("count field missing")
	}
	if _, ok := out["matches"]; !ok {
		t.Error("matches field missing")
	}
}

// TestDocIntelFind_EntityID_NoKnowledgeSvc verifies that a nil knowledgeSvc
// returns an empty related_knowledge array (graceful degradation).
func TestDocIntelFind_EntityID_NoKnowledgeSvc(t *testing.T) {
	t.Parallel()
	repoRoot := t.TempDir()
	indexRoot := filepath.Join(t.TempDir(), "index")
	intelSvc := service.NewIntelligenceService(indexRoot, repoRoot)

	tool := docIntelTool(intelSvc, nil, nil) // nil knowledgeSvc
	req := makeRequest(map[string]any{
		"action":    "find",
		"entity_id": "FEAT-01TESTNILSVC000001",
	})
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	text := extractText(t, result)
	var out map[string]any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	rk, ok := out["related_knowledge"]
	if !ok {
		t.Fatal("related_knowledge field missing")
	}
	rkSlice, ok := rk.([]any)
	if !ok {
		t.Fatalf("related_knowledge type = %T, want []any", rk)
	}
	if len(rkSlice) != 0 {
		t.Errorf("related_knowledge len = %d, want 0 (nil svc)", len(rkSlice))
	}
	if km, _ := out["knowledge_matches"].(float64); km != 0 {
		t.Errorf("knowledge_matches = %v, want 0", out["knowledge_matches"])
	}
}

// TestDocIntelFind_EntityID_RelatedKnowledge_LearnedFrom verifies that a knowledge
// entry with learned_from matching the queried entity appears in related_knowledge.
func TestDocIntelFind_EntityID_RelatedKnowledge_LearnedFrom(t *testing.T) {
	t.Parallel()
	env := setupDocIntelFind(t)

	entityID := "FEAT-01TESTLEARNFROM00001"
	contributeKnowledge(t, env.knowledgeSvc,
		"learned-from-feat", "Content from feature task", "project", entityID, nil)

	out := callDocIntelFind(t, env, entityID)

	rk, ok := out["related_knowledge"].([]any)
	if !ok {
		t.Fatalf("related_knowledge type = %T", out["related_knowledge"])
	}
	if len(rk) != 1 {
		t.Fatalf("related_knowledge len = %d, want 1", len(rk))
	}

	entry, _ := rk[0].(map[string]any)
	if entry["topic"] != "learned-from-feat" {
		t.Errorf("topic = %q, want %q", entry["topic"], "learned-from-feat")
	}
	if _, ok := entry["id"]; !ok {
		t.Error("entry missing 'id' field")
	}
	if _, ok := entry["content"]; !ok {
		t.Error("entry missing 'content' field")
	}
	if _, ok := entry["confidence"]; !ok {
		t.Error("entry missing 'confidence' field")
	}
	if _, ok := entry["status"]; !ok {
		t.Error("entry missing 'status' field")
	}
	if km, _ := out["knowledge_matches"].(float64); km != 1 {
		t.Errorf("knowledge_matches = %v, want 1", out["knowledge_matches"])
	}
}

// TestDocIntelFind_EntityID_RelatedKnowledge_Tags verifies matching by tag:
// both entity-ID tag and entity-type tag should match.
func TestDocIntelFind_EntityID_RelatedKnowledge_Tags(t *testing.T) {
	t.Parallel()
	env := setupDocIntelFind(t)

	entityID := "FEAT-01TESTTAGMATCH00001"
	// Match by entity ID tag.
	contributeKnowledge(t, env.knowledgeSvc,
		"tag-entity-id", "Tagged with entity ID", "project", "", []string{entityID})
	// Match by entity type tag.
	contributeKnowledge(t, env.knowledgeSvc,
		"tag-entity-type", "Tagged feature", "project", "", []string{"feature"})
	// No match.
	contributeKnowledge(t, env.knowledgeSvc,
		"tag-unrelated", "Unrelated entry", "project", "", []string{"unrelated"})

	out := callDocIntelFind(t, env, entityID)

	rk, ok := out["related_knowledge"].([]any)
	if !ok {
		t.Fatalf("related_knowledge type = %T", out["related_knowledge"])
	}
	if len(rk) != 2 {
		t.Errorf("related_knowledge len = %d, want 2 (entity-id tag + entity-type tag)", len(rk))
	}
	// Verify unrelated entry is absent.
	for _, item := range rk {
		e, _ := item.(map[string]any)
		if e["topic"] == "tag-unrelated" {
			t.Error("unrelated entry should not appear in related_knowledge")
		}
	}
}

// TestDocIntelFind_EntityID_RelatedKnowledge_Dedup verifies that an entry matching
// both learned_from and tags appears exactly once.
func TestDocIntelFind_EntityID_RelatedKnowledge_Dedup(t *testing.T) {
	t.Parallel()
	env := setupDocIntelFind(t)

	entityID := "FEAT-01TESTDEDUP000000001"
	contributeKnowledge(t, env.knowledgeSvc,
		"dedup-entry", "Matches both ways", "project", entityID, []string{entityID})

	out := callDocIntelFind(t, env, entityID)

	rk, ok := out["related_knowledge"].([]any)
	if !ok {
		t.Fatalf("related_knowledge type = %T", out["related_knowledge"])
	}
	if len(rk) != 1 {
		t.Errorf("related_knowledge len = %d, want 1 (deduplicated)", len(rk))
	}
}

// TestDocIntelFind_EntityID_RetiredExcluded verifies that retired knowledge entries
// do not appear in related_knowledge.
func TestDocIntelFind_EntityID_RetiredExcluded(t *testing.T) {
	t.Parallel()
	env := setupDocIntelFind(t)

	entityID := "FEAT-01TESTRETIRED000001"

	// Contribute then retire.
	id := contributeKnowledge(t, env.knowledgeSvc,
		"retired-entry", "Should be excluded", "project", entityID, nil)
	if _, err := env.knowledgeSvc.Retire(id, "test"); err != nil {
		t.Fatalf("retire: %v", err)
	}

	out := callDocIntelFind(t, env, entityID)

	rk, ok := out["related_knowledge"].([]any)
	if !ok {
		t.Fatalf("related_knowledge type = %T", out["related_knowledge"])
	}
	if len(rk) != 0 {
		t.Errorf("related_knowledge len = %d, want 0 (retired excluded)", len(rk))
	}
}

// TestDocIntelFind_EntityID_RelatedKnowledge_ScopeMatch verifies that a knowledge
// entry whose scope is a path prefix of a document referencing the entity appears
// in related_knowledge (FR-004 scope-based matching).
func TestDocIntelFind_EntityID_RelatedKnowledge_ScopeMatch(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	indexRoot := filepath.Join(t.TempDir(), "index")
	stateRoot := t.TempDir()

	intelSvc := service.NewIntelligenceService(indexRoot, repoRoot)
	knowledgeSvc := service.NewKnowledgeService(stateRoot)

	entityID := "FEAT-TESTSCOPEMATCH001"

	// Write and ingest a document that references the entity.
	docContent := "# Feature Design\n\nThis design covers " + entityID + " requirements.\n"
	docPath := filepath.Join(repoRoot, "work/design/feat.md")
	if err := os.MkdirAll(filepath.Dir(docPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(docPath, []byte(docContent), 0o644); err != nil {
		t.Fatalf("write doc: %v", err)
	}
	if _, err := intelSvc.IngestDocument("work/design/feat.md", "work/design/feat.md"); err != nil {
		t.Fatalf("IngestDocument: %v", err)
	}

	// Contribute a knowledge entry with scope = the document path (exact match is a prefix match).
	contributeKnowledge(t, knowledgeSvc, "scope-match-entry", "Design insight", "work/design/feat.md", "", nil)

	tool := docIntelTool(intelSvc, nil, knowledgeSvc)
	req := makeRequest(map[string]any{
		"action":    "find",
		"entity_id": entityID,
	})
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	text := extractText(t, result)
	var out map[string]any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("unmarshal: %v\nraw: %s", err, text)
	}

	rk, ok := out["related_knowledge"].([]any)
	if !ok {
		t.Fatalf("expected related_knowledge array, got %T", out["related_knowledge"])
	}
	if len(rk) == 0 {
		t.Error("expected at least one knowledge entry matched by scope prefix, got 0")
	}
	// Verify the matched entry is the one we contributed.
	found := false
	for _, e := range rk {
		if em, ok := e.(map[string]any); ok {
			if em["topic"] == "scope-match-entry" {
				found = true
				break
			}
		}
	}
	if !found {
		t.Errorf("expected 'scope-match-entry' in related_knowledge, got: %v", rk)
	}
}

// TestKnowledgeEntityType verifies the entity type derivation helper.
func TestKnowledgeEntityType(t *testing.T) {
	t.Parallel()
	cases := []struct {
		id   string
		want string
	}{
		{"FEAT-01ABCDEF", "feature"},
		{"TASK-01ABCDEF", "task"},
		{"BUG-01ABCDEF", "bug"},
		{"P1-some-plan", ""},
		{"", ""},
		{"UNKNOWN-123", ""},
	}
	for _, tc := range cases {
		got := knowledgeEntityType(tc.id)
		if got != tc.want {
			t.Errorf("knowledgeEntityType(%q) = %q, want %q", tc.id, got, tc.want)
		}
	}
}
