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

// setupGuideEnv creates a temp IntelligenceService and ingests the given markdown.
func setupGuideEnv(t *testing.T, docID, markdown string) *service.IntelligenceService {
	t.Helper()
	tmp := t.TempDir()
	indexRoot := filepath.Join(tmp, "index")
	docPath := filepath.Join(tmp, "doc.md")
	if err := os.WriteFile(docPath, []byte(markdown), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := service.NewIntelligenceService(indexRoot, tmp)
	t.Cleanup(func() { svc.Close() }) //nolint:errcheck
	if _, err := svc.IngestDocument(docID, "doc.md"); err != nil {
		t.Fatalf("IngestDocument: %v", err)
	}
	return svc
}

// callGuide invokes the guide action and returns the unmarshalled response.
func callGuide(t *testing.T, svc *service.IntelligenceService, docID string) map[string]any {
	t.Helper()
	tool := docIntelTool(svc, nil, nil)
	req := makeRequest(map[string]any{"action": "guide", "id": docID})
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("guide handler error: %v", err)
	}
	text := extractText(t, result)
	var out map[string]any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("unmarshal guide response: %v (text=%q)", err, text)
	}
	return out
}

// callPending invokes the pending action and returns the unmarshalled response.
func callPending(t *testing.T, svc *service.IntelligenceService) map[string]any {
	t.Helper()
	tool := docIntelTool(svc, nil, nil)
	req := makeRequest(map[string]any{"action": "pending"})
	result, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("pending handler error: %v", err)
	}
	text := extractText(t, result)
	var out map[string]any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("unmarshal pending response: %v (text=%q)", err, text)
	}
	return out
}

// ─── AC-001: section_count in pending response ────────────────────────────────

// TestDocIntelPending_SectionCount verifies every entry in the pending response
// has a section_count field equal to the document's Layer 1 section count (AC-001).
func TestDocIntelPending_SectionCount(t *testing.T) {
	markdown := `# Design Document

- Status: draft

## Overview

Content here.

## Requirements

More content.
`
	svc := setupGuideEnv(t, "pending-doc", markdown)
	out := callPending(t, svc)

	docs, ok := out["documents"].([]interface{})
	if !ok {
		t.Fatalf("pending response missing documents array: %v", out)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 pending doc, got %d", len(docs))
	}
	entry, ok := docs[0].(map[string]any)
	if !ok {
		t.Fatalf("pending entry is not map: %T", docs[0])
	}
	if entry["id"] != "pending-doc" {
		t.Errorf("entry id = %v, want pending-doc", entry["id"])
	}
	// section_count should be an integer > 0
	countRaw, exists := entry["section_count"]
	if !exists {
		t.Error("pending entry missing section_count field")
	}
	// JSON numbers unmarshal as float64
	count, ok := countRaw.(float64)
	if !ok {
		t.Fatalf("section_count is %T (%v), want number", countRaw, countRaw)
	}
	if count <= 0 {
		t.Errorf("section_count = %v, want > 0", count)
	}
}

// TestDocIntelPending_SectionCount_Zero verifies section_count is 0 for a
// document with no sections (AC-001).
func TestDocIntelPending_SectionCount_Zero(t *testing.T) {
	// A document with no headings produces no sections in the Layer 1 index.
	markdown := "Just some text with no headings.\n"
	svc := setupGuideEnv(t, "no-sections-doc", markdown)
	out := callPending(t, svc)

	docs, ok := out["documents"].([]interface{})
	if !ok {
		t.Fatalf("pending response missing documents array: %v", out)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 pending doc, got %d", len(docs))
	}
	entry := docs[0].(map[string]any)
	count, ok := entry["section_count"].(float64)
	if !ok {
		t.Fatalf("section_count missing or wrong type: %v", entry["section_count"])
	}
	if count != 0 {
		t.Errorf("section_count = %v, want 0", count)
	}
}

// ─── AC-002: taxonomy in guide response ───────────────────────────────────────

// TestDocIntelGuide_Taxonomy verifies the guide response includes a taxonomy
// block with roles and confidence arrays (AC-002).
func TestDocIntelGuide_Taxonomy(t *testing.T) {
	svc := setupGuideEnv(t, "tax-doc", "# Doc\n\nContent.\n")
	out := callGuide(t, svc, "tax-doc")

	taxRaw, ok := out["taxonomy"]
	if !ok {
		t.Fatal("guide response missing taxonomy field")
	}
	tax, ok := taxRaw.(map[string]any)
	if !ok {
		t.Fatalf("taxonomy is %T, want map", taxRaw)
	}

	// Check roles array
	rolesRaw, ok := tax["roles"]
	if !ok {
		t.Fatal("taxonomy missing roles field")
	}
	roles, ok := rolesRaw.([]interface{})
	if !ok {
		t.Fatalf("taxonomy.roles is %T, want array", rolesRaw)
	}
	if len(roles) == 0 {
		t.Error("taxonomy.roles is empty, want non-empty")
	}
	// Verify "question" appears in roles (AC-003 / REQ-004)
	foundQuestion := false
	for _, r := range roles {
		if r == "question" {
			foundQuestion = true
			break
		}
	}
	if !foundQuestion {
		t.Error("taxonomy.roles does not contain 'question'")
	}

	// Check confidence array equals ["high","medium","low"]
	confRaw, ok := tax["confidence"]
	if !ok {
		t.Fatal("taxonomy missing confidence field")
	}
	conf, ok := confRaw.([]interface{})
	if !ok {
		t.Fatalf("taxonomy.confidence is %T, want array", confRaw)
	}
	want := []string{"high", "medium", "low"}
	if len(conf) != len(want) {
		t.Fatalf("taxonomy.confidence len=%d, want %d", len(conf), len(want))
	}
	for i, v := range want {
		if conf[i] != v {
			t.Errorf("taxonomy.confidence[%d] = %v, want %q", i, conf[i], v)
		}
	}
}

// ─── AC-004: suggested_classifications in guide response ──────────────────────

// TestDocIntelGuide_SuggestedClassifications_Empty verifies that the
// suggested_classifications array is present even when empty (AC-004).
func TestDocIntelGuide_SuggestedClassifications_Empty(t *testing.T) {
	// No recognisable headings
	svc := setupGuideEnv(t, "sc-empty-doc", "# Introduction\n\nContent.\n")
	out := callGuide(t, svc, "sc-empty-doc")

	scRaw, ok := out["suggested_classifications"]
	if !ok {
		t.Fatal("guide response missing suggested_classifications field")
	}
	sc, ok := scRaw.([]interface{})
	if !ok {
		t.Fatalf("suggested_classifications is %T, want array", scRaw)
	}
	// May be empty or contain "Introduction" - just verify it's an array
	_ = sc
}

// ─── AC-005: Acceptance Criteria section → requirement ────────────────────────

// TestDocIntelGuide_SuggestedClassifications_AcceptanceCriteria verifies that a
// section titled "Acceptance Criteria" produces a requirement suggestion (AC-005).
func TestDocIntelGuide_SuggestedClassifications_AcceptanceCriteria(t *testing.T) {
	markdown := `# Spec

## Overview

Some content.

## Acceptance Criteria

- AC-1. Something must happen.
`
	svc := setupGuideEnv(t, "sc-ac-doc", markdown)
	out := callGuide(t, svc, "sc-ac-doc")

	sc := extractSuggestedClassifications(t, out)
	entry := findSuggestionByTitle(sc, "Acceptance Criteria")
	if entry == nil {
		t.Fatal("no suggestion for 'Acceptance Criteria' section")
	}
	if entry["role"] != "requirement" {
		t.Errorf("role = %v, want requirement", entry["role"])
	}
	if entry["confidence"] != "high" {
		t.Errorf("confidence = %v, want high", entry["confidence"])
	}
}

// ─── AC-006: front-matter first section → narrative ───────────────────────────

// TestDocIntelGuide_SuggestedClassifications_FrontMatter verifies that the first
// section of a document with front-matter gets a narrative suggestion (AC-006).
func TestDocIntelGuide_SuggestedClassifications_FrontMatter(t *testing.T) {
	// The parser extracts front matter from bullet-list after the first heading.
	markdown := `# Feature Spec

- Type: specification
- Status: draft

---

## Goals

What we want to achieve.
`
	svc := setupGuideEnv(t, "sc-fm-doc", markdown)
	out := callGuide(t, svc, "sc-fm-doc")

	sc := extractSuggestedClassifications(t, out)
	// The first section path should have a narrative suggestion
	if len(sc) == 0 {
		t.Fatal("expected at least one suggestion for front-matter document")
	}
	found := false
	for _, entry := range sc {
		if entry["role"] == "narrative" && entry["confidence"] == "high" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("no narrative/high suggestion found in %v", sc)
	}
}

// ─── AC-007: Alternatives Considered → alternative ────────────────────────────

// TestDocIntelGuide_SuggestedClassifications_AlternativesConsidered verifies
// that a section titled "Alternatives Considered" produces an alternative
// suggestion (AC-007).
func TestDocIntelGuide_SuggestedClassifications_AlternativesConsidered(t *testing.T) {
	markdown := `# Design

## Alternatives Considered

Option A vs Option B.
`
	svc := setupGuideEnv(t, "sc-alt-doc", markdown)
	out := callGuide(t, svc, "sc-alt-doc")

	sc := extractSuggestedClassifications(t, out)
	entry := findSuggestionByTitle(sc, "Alternatives Considered")
	if entry == nil {
		t.Fatal("no suggestion for 'Alternatives Considered' section")
	}
	if entry["role"] != "alternative" {
		t.Errorf("role = %v, want alternative", entry["role"])
	}
	if entry["confidence"] != "high" {
		t.Errorf("confidence = %v, want high", entry["confidence"])
	}
}

// ─── AC-008: guide does not auto-classify ─────────────────────────────────────

// TestDocIntelGuide_NoAutoClassify verifies that calling guide (which generates
// suggestions) does not write any classifications to the index (AC-008).
func TestDocIntelGuide_NoAutoClassify(t *testing.T) {
	tmp := t.TempDir()
	indexRoot := filepath.Join(tmp, "index")
	docPath := filepath.Join(tmp, "doc.md")
	markdown := `# Design

## Acceptance Criteria

The thing must work.
`
	if err := os.WriteFile(docPath, []byte(markdown), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := service.NewIntelligenceService(indexRoot, tmp)
	t.Cleanup(func() { svc.Close() }) //nolint:errcheck

	index0, err := svc.IngestDocument("ac008-doc", "doc.md")
	if err != nil {
		t.Fatalf("IngestDocument: %v", err)
	}
	if index0.Classified {
		t.Fatal("document should not be classified after ingest")
	}

	// Call guide (generates suggestions).
	_ = callGuide(t, svc, "ac008-doc")

	// Load the index again and confirm classification state is unchanged.
	index1, err := svc.GetDocumentIndex("ac008-doc")
	if err != nil {
		t.Fatalf("GetDocumentIndex after guide: %v", err)
	}
	if index1.Classified {
		t.Error("index.Classified became true after guide call — guide must not auto-classify")
	}
	if len(index1.Classifications) != 0 {
		t.Errorf("index has %d classifications after guide, want 0", len(index1.Classifications))
	}
}

// ─── AC-009: pre-existing guide fields preserved ──────────────────────────────

// TestDocIntelGuide_ExistingFieldsPreserved verifies that all pre-existing guide
// response fields are still present after the enrichment changes (AC-009).
func TestDocIntelGuide_ExistingFieldsPreserved(t *testing.T) {
	svc := setupGuideEnv(t, "ac009-doc", "# Doc\n\nSome text.\n")
	out := callGuide(t, svc, "ac009-doc")

	required := []string{"document_id", "document_path", "content_hash", "classified", "outline", "entity_refs", "extraction_hints"}
	for _, field := range required {
		if _, ok := out[field]; !ok {
			t.Errorf("guide response missing required field %q", field)
		}
	}
}

// ─── Benchmarks (AC-010, AC-011) ──────────────────────────────────────────────

// BenchmarkDocIntelPending_50Docs benchmarks the pending action with 50 docs (AC-010).
func BenchmarkDocIntelPending_50Docs(b *testing.B) {
	tmp := b.TempDir()
	indexRoot := filepath.Join(tmp, "index")
	svc := service.NewIntelligenceService(indexRoot, tmp)
	b.Cleanup(func() { svc.Close() }) //nolint:errcheck

	// Seed 50 documents with a few sections each.
	for i := 0; i < 50; i++ {
		docID := "bench-doc-" + itoa(i)
		content := "# Doc " + itoa(i) + "\n\n## Overview\n\nContent.\n\n## Requirements\n\nMore.\n"
		docPath := filepath.Join(tmp, docID+".md")
		if err := os.WriteFile(docPath, []byte(content), 0o644); err != nil {
			b.Fatal(err)
		}
		if _, err := svc.IngestDocument(docID, docID+".md"); err != nil {
			b.Fatal(err)
		}
	}

	tool := docIntelTool(svc, nil, nil)
	req := makeRequest(map[string]any{"action": "pending"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := tool.Handler(context.Background(), req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDocIntelGuide_200Sections benchmarks the guide action with a large doc (AC-011).
func BenchmarkDocIntelGuide_200Sections(b *testing.B) {
	tmp := b.TempDir()
	indexRoot := filepath.Join(tmp, "index")
	svc := service.NewIntelligenceService(indexRoot, tmp)
	b.Cleanup(func() { svc.Close() }) //nolint:errcheck

	// Build a document with ~200 sections.
	var mdBuilder []byte
	mdBuilder = append(mdBuilder, []byte("# Large Document\n\n")...)
	for i := 0; i < 200; i++ {
		line := "## Section " + itoa(i) + "\n\nContent for section " + itoa(i) + ".\n\n"
		mdBuilder = append(mdBuilder, []byte(line)...)
	}
	docPath := filepath.Join(tmp, "large.md")
	if err := os.WriteFile(docPath, mdBuilder, 0o644); err != nil {
		b.Fatal(err)
	}
	if _, err := svc.IngestDocument("large-doc", "large.md"); err != nil {
		b.Fatal(err)
	}

	tool := docIntelTool(svc, nil, nil)
	req := makeRequest(map[string]any{"action": "guide", "id": "large-doc"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := tool.Handler(context.Background(), req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ─── local helpers ────────────────────────────────────────────────────────────

func extractSuggestedClassifications(t *testing.T, out map[string]any) []map[string]any {
	t.Helper()
	scRaw, ok := out["suggested_classifications"]
	if !ok {
		t.Fatal("guide response missing suggested_classifications field")
	}
	rawSlice, ok := scRaw.([]interface{})
	if !ok {
		t.Fatalf("suggested_classifications is %T, want array", scRaw)
	}
	result := make([]map[string]any, 0, len(rawSlice))
	for _, item := range rawSlice {
		m, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("suggestion entry is %T, want map", item)
		}
		result = append(result, m)
	}
	return result
}

func findSuggestionByTitle(sc []map[string]any, title string) map[string]any {
	for _, entry := range sc {
		if entry["title"] == title {
			return entry
		}
	}
	return nil
}

// itoa is a minimal int-to-string helper for benchmark setup.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 10)
	for n > 0 {
		buf = append(buf, byte('0'+n%10))
		n /= 10
	}
	// reverse
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
