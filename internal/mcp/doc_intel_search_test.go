package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sambeau/kanbanzai/internal/service"
)

// ─── doc_intel search tests ───────────────────────────────────────────────────

type docIntelSearchEnv struct {
	intelSvc *service.IntelligenceService
	repoRoot string
}

func setupDocIntelSearch(t *testing.T) *docIntelSearchEnv {
	t.Helper()
	repoRoot := t.TempDir()
	indexRoot := filepath.Join(t.TempDir(), "index")
	return &docIntelSearchEnv{
		intelSvc: service.NewIntelligenceService(indexRoot, repoRoot),
		repoRoot: repoRoot,
	}
}

func writeDocIntelFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return name
}

func callDocIntel(t *testing.T, env *docIntelSearchEnv, args map[string]any) map[string]any {
	t.Helper()
	tool := docIntelTool(env.intelSvc, nil, nil)
	req := makeRequest(args)
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("doc_intel handler error: %v", err)
	}
	text := extractText(t, result)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("unmarshal response: %v\nraw: %s", err, text)
	}
	return parsed
}

const searchTestContent = `# Authentication System

This document describes the authentication system for FEAT-001.

## Overview

The authentication system handles user login and session management.

## Requirements

The system must support token-based authentication.
Users should be authenticated before accessing resources.
`

func TestDocIntelSearch_BasicQuery(t *testing.T) {
	env := setupDocIntelSearch(t)
	docPath := writeDocIntelFile(t, env.repoRoot, "docs/auth.md", searchTestContent)

	if _, err := env.intelSvc.IngestDocument("auth-doc", docPath); err != nil {
		t.Fatalf("IngestDocument: %v", err)
	}

	resp := callDocIntel(t, env, map[string]any{
		"action": "search",
		"query":  "authentication",
	})

	results, ok := resp["results"].([]any)
	if !ok {
		t.Fatalf("expected results array, got: %T %v", resp["results"], resp["results"])
	}
	if len(results) == 0 {
		t.Error("expected at least one result for 'authentication' query")
	}
	if _, ok := resp["total_matches"]; !ok {
		t.Error("response missing total_matches field")
	}
	if _, ok := resp["returned"]; !ok {
		t.Error("response missing returned field")
	}
	if _, ok := resp["query"]; !ok {
		t.Error("response missing query field")
	}
}

func TestDocIntelSearch_EmptyQuery(t *testing.T) {
	env := setupDocIntelSearch(t)

	resp := callDocIntel(t, env, map[string]any{
		"action": "search",
		"query":  "",
	})

	// Should return an error response (inline error)
	if _, ok := resp["error"]; !ok {
		t.Errorf("expected error for empty query, got: %v", resp)
	}
}

func TestDocIntelSearch_LimitClamped(t *testing.T) {
	env := setupDocIntelSearch(t)
	docPath := writeDocIntelFile(t, env.repoRoot, "docs/auth.md", searchTestContent)

	if _, err := env.intelSvc.IngestDocument("auth-doc", docPath); err != nil {
		t.Fatalf("IngestDocument: %v", err)
	}

	// limit=100 should be clamped to 50; no error
	resp := callDocIntel(t, env, map[string]any{
		"action": "search",
		"query":  "authentication",
		"limit":  float64(100),
	})

	// Should return results without error
	if _, ok := resp["error"]; ok {
		t.Errorf("expected no error, got: %v", resp)
	}
	results, _ := resp["results"].([]any)
	if len(results) > 50 {
		t.Errorf("returned %d results, expected at most 50 (clamped)", len(results))
	}
}

func TestDocIntelSearch_EmptyResults(t *testing.T) {
	env := setupDocIntelSearch(t)
	docPath := writeDocIntelFile(t, env.repoRoot, "docs/auth.md", searchTestContent)

	if _, err := env.intelSvc.IngestDocument("auth-doc", docPath); err != nil {
		t.Fatalf("IngestDocument: %v", err)
	}

	resp := callDocIntel(t, env, map[string]any{
		"action": "search",
		"query":  "xyzzy_nonexistent_zzzq",
	})

	// Should return empty results array, not an error
	if _, ok := resp["error"]; ok {
		t.Errorf("unexpected error for zero-match query: %v", resp)
	}
	results, ok := resp["results"].([]any)
	if !ok {
		t.Fatalf("expected results array, got: %T", resp["results"])
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestDocIntelSearch_DocTypeFilter(t *testing.T) {
	env := setupDocIntelSearch(t)

	// Document with type: specification in bullet front matter
	specContent := "# Auth Spec\n\n- Type: specification\n- Status: draft\n\n## Requirements\n\nThe system must validate tokens.\n"
	specPath := writeDocIntelFile(t, env.repoRoot, "docs/auth-spec.md", specContent)
	if _, err := env.intelSvc.IngestDocument("auth-spec", specPath); err != nil {
		t.Fatalf("IngestDocument spec: %v", err)
	}

	// Document without a type field
	designContent := "# Auth Design\n\n## Overview\n\nThe system should validate tokens using JWT.\n"
	designPath := writeDocIntelFile(t, env.repoRoot, "docs/auth-design.md", designContent)
	if _, err := env.intelSvc.IngestDocument("auth-design", designPath); err != nil {
		t.Fatalf("IngestDocument design: %v", err)
	}

	// Search with doc_type filter — should return only auth-spec results
	resp := callDocIntel(t, env, map[string]any{
		"action":   "search",
		"query":    "validate tokens",
		"doc_type": "specification",
	})

	if _, ok := resp["error"]; ok {
		t.Fatalf("unexpected error: %v", resp)
	}

	results, ok := resp["results"].([]any)
	if !ok {
		t.Fatalf("expected results array, got: %T", resp["results"])
	}

	// Every returned result must come from the specification document
	for _, r := range results {
		rm, ok := r.(map[string]any)
		if !ok {
			continue
		}
		docID, _ := rm["document_id"].(string)
		if docID != "auth-spec" {
			t.Errorf("expected only auth-spec results with doc_type filter, got document_id = %q", docID)
		}
	}
}
