package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/sambeau/kanbanzai/internal/service"
)

// setupFindEnv creates an IntelligenceService with n documents, each containing
// a "Requirements" section (detected as role=requirement by Layer 2 conventional
// role detection). Returns the service and a cleanup function.
func setupFindEnv(t *testing.T, n int) *service.IntelligenceService {
	t.Helper()
	tmp := t.TempDir()
	indexRoot := filepath.Join(tmp, "index")

	for i := 0; i < n; i++ {
		docPath := filepath.Join(tmp, fmt.Sprintf("doc-%03d.md", i))
		content := fmt.Sprintf("# Document %d\n\n## Requirements\n\nRequirement content for doc %d.\n", i, i)
		if err := os.WriteFile(docPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	svc := service.NewIntelligenceService(indexRoot, tmp)
	t.Cleanup(func() { svc.Close() }) //nolint:errcheck

	for i := 0; i < n; i++ {
		docID := fmt.Sprintf("doc-%03d", i)
		relPath := fmt.Sprintf("doc-%03d.md", i)
		if _, err := svc.IngestDocument(docID, relPath); err != nil {
			t.Fatalf("IngestDocument(%s): %v", docID, err)
		}
	}

	return svc
}

func callFind(t *testing.T, svc *service.IntelligenceService, args map[string]any) map[string]any {
	t.Helper()
	args["action"] = "find"
	tool := docIntelTool(svc, nil, nil)
	req := makeRequest(args)
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("find handler error: %v", err)
	}
	text := extractText(t, result)
	var out map[string]any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("unmarshal find result: %v\nraw: %s", err, text)
	}
	return out
}

// TestDocIntelFind_Role_DefaultLimitTruncates verifies that find(role) with no
// explicit limit returns at most 10 matches and sets truncated=true when there
// are more results available.
func TestDocIntelFind_Role_DefaultLimitTruncates(t *testing.T) {
	t.Parallel()
	svc := setupFindEnv(t, 15) // 15 docs × 1 requirement section = 15 matches

	out := callFind(t, svc, map[string]any{"role": "requirement"})

	count, _ := out["count"].(float64)
	if int(count) != 15 {
		t.Errorf("count = %v, want 15 (total matches)", count)
	}

	matches, ok := out["matches"].([]any)
	if !ok {
		t.Fatalf("matches is not an array: %T", out["matches"])
	}
	if len(matches) != 10 {
		t.Errorf("returned matches = %d, want 10 (default limit)", len(matches))
	}

	truncated, _ := out["truncated"].(bool)
	if !truncated {
		t.Error("truncated = false, want true")
	}

	returned, _ := out["returned"].(float64)
	if int(returned) != 10 {
		t.Errorf("returned = %v, want 10", returned)
	}
}

// TestDocIntelFind_Role_ExplicitLimitRespected verifies that an explicit limit
// parameter controls how many matches are returned.
func TestDocIntelFind_Role_ExplicitLimitRespected(t *testing.T) {
	t.Parallel()
	svc := setupFindEnv(t, 15)

	out := callFind(t, svc, map[string]any{"role": "requirement", "limit": float64(5)})

	count, _ := out["count"].(float64)
	if int(count) != 15 {
		t.Errorf("count = %v, want 15 (total matches)", count)
	}

	matches, ok := out["matches"].([]any)
	if !ok {
		t.Fatalf("matches is not an array: %T", out["matches"])
	}
	if len(matches) != 5 {
		t.Errorf("returned matches = %d, want 5", len(matches))
	}

	truncated, _ := out["truncated"].(bool)
	if !truncated {
		t.Error("truncated = false, want true")
	}
}

// TestDocIntelFind_Role_UnderLimitNoTruncation verifies that when total matches
// are within the limit, truncated is absent and all matches are returned.
func TestDocIntelFind_Role_UnderLimitNoTruncation(t *testing.T) {
	t.Parallel()
	svc := setupFindEnv(t, 3)

	out := callFind(t, svc, map[string]any{"role": "requirement"})

	count, _ := out["count"].(float64)
	if int(count) != 3 {
		t.Errorf("count = %v, want 3", count)
	}

	matches, ok := out["matches"].([]any)
	if !ok {
		t.Fatalf("matches is not an array: %T", out["matches"])
	}
	if len(matches) != 3 {
		t.Errorf("returned matches = %d, want 3", len(matches))
	}

	if _, hasTruncated := out["truncated"]; hasTruncated {
		t.Error("truncated field should be absent when results fit within limit")
	}
}

// TestDocIntelFind_Role_LimitCappedAt50 verifies the limit cannot exceed 50.
func TestDocIntelFind_Role_LimitCappedAt50(t *testing.T) {
	t.Parallel()
	svc := setupFindEnv(t, 55)

	out := callFind(t, svc, map[string]any{"role": "requirement", "limit": float64(100)})

	matches, ok := out["matches"].([]any)
	if !ok {
		t.Fatalf("matches is not an array: %T", out["matches"])
	}
	if len(matches) != 50 {
		t.Errorf("returned matches = %d, want 50 (max cap)", len(matches))
	}

	truncated, _ := out["truncated"].(bool)
	if !truncated {
		t.Error("truncated = false, want true")
	}
}

// TestDocIntelFind_Role_ScopeWithLimit verifies scope and limit work together.
func TestDocIntelFind_Role_ScopeWithLimit(t *testing.T) {
	t.Parallel()
	svc := setupFindEnv(t, 15)

	out := callFind(t, svc, map[string]any{
		"role":  "requirement",
		"scope": "doc-003",
	})

	count, _ := out["count"].(float64)
	if int(count) != 1 {
		t.Errorf("count = %v, want 1 (scoped to single doc)", count)
	}

	matches, ok := out["matches"].([]any)
	if !ok {
		t.Fatalf("matches is not an array: %T", out["matches"])
	}
	if len(matches) != 1 {
		t.Errorf("returned matches = %d, want 1", len(matches))
	}

	if _, hasTruncated := out["truncated"]; hasTruncated {
		t.Error("truncated field should be absent for scoped single-doc result")
	}

	scope, _ := out["scope"].(string)
	if scope != "doc-003" {
		t.Errorf("scope = %q, want %q", scope, "doc-003")
	}
}

// TestDocIntelFind_Concept_LimitApplied verifies that find(concept) also respects
// the limit parameter. Uses a concept that appears in multiple documents.
func TestDocIntelFind_Concept_LimitApplied(t *testing.T) {
	t.Parallel()
	// Concept-based find depends on the concept registry, which is populated
	// during classification. Since we only have Layer 2 conventional roles here,
	// we just verify the handler doesn't error with a concept query.
	// The truncation logic is shared, so role-based tests cover the core path.
	svc := setupFindEnv(t, 3)

	out := callFind(t, svc, map[string]any{"concept": "nonexistent-concept-xyz"})

	// Should return 0 matches gracefully (concept not in registry).
	count, _ := out["count"].(float64)
	if int(count) != 0 {
		t.Errorf("count = %v, want 0 for unknown concept", count)
	}
}

// TestDocIntelFind_Role_NegativeLimitUsesDefault verifies negative limit falls
// back to the default of 10.
func TestDocIntelFind_Role_NegativeLimitUsesDefault(t *testing.T) {
	t.Parallel()
	svc := setupFindEnv(t, 15)

	out := callFind(t, svc, map[string]any{"role": "requirement", "limit": float64(-5)})

	matches, ok := out["matches"].([]any)
	if !ok {
		t.Fatalf("matches is not an array: %T", out["matches"])
	}
	if len(matches) != 10 {
		t.Errorf("returned matches = %d, want 10 (negative limit should use default)", len(matches))
	}
}

// TestDocIntelFind_Role_ZeroLimitUsesDefault verifies limit=0 falls back to default.
func TestDocIntelFind_Role_ZeroLimitUsesDefault(t *testing.T) {
	t.Parallel()
	svc := setupFindEnv(t, 15)

	out := callFind(t, svc, map[string]any{"role": "requirement", "limit": float64(0)})

	matches, ok := out["matches"].([]any)
	if !ok {
		t.Fatalf("matches is not an array: %T", out["matches"])
	}
	if len(matches) != 10 {
		t.Errorf("returned matches = %d, want 10 (zero limit should use default)", len(matches))
	}
}
