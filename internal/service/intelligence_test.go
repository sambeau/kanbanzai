package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sambeau/kanbanzai/internal/docint"
)

func writeTestDoc(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return name
}

const testMarkdown = `# Design Document

- Type: design
- Status: draft

## Overview

This document describes FEAT-001 and its requirements.

## Requirements

The system must support TASK-042 completion.

### Sub-requirement

Additional details about FEAT-001 here.
`

const testMarkdown2 = `# Specification

- Type: specification
- Status: draft

## Scope

This spec covers FEAT-001 in detail.

## Constraints

Performance constraint for FEAT-001.
`

func TestIngestDocument(t *testing.T) {
	tmp := t.TempDir()
	indexRoot := filepath.Join(tmp, "index")
	docPath := writeTestDoc(t, tmp, "docs/design.md", testMarkdown)

	svc := NewIntelligenceService(indexRoot, tmp)
	t.Cleanup(func() { svc.Close() }) //nolint:errcheck

	index, err := svc.IngestDocument("test-doc", docPath)
	if err != nil {
		t.Fatalf("IngestDocument: %v", err)
	}

	if index.DocumentID != "test-doc" {
		t.Errorf("DocumentID = %q, want %q", index.DocumentID, "test-doc")
	}
	if index.DocumentPath != docPath {
		t.Errorf("DocumentPath = %q, want %q", index.DocumentPath, docPath)
	}
	if index.ContentHash == "" {
		t.Error("ContentHash is empty")
	}
	if len(index.Sections) == 0 {
		t.Fatal("Sections is empty")
	}

	// Verify sections were parsed (should have "Design Document" as root)
	if index.Sections[0].Title != "Design Document" {
		t.Errorf("first section title = %q, want %q", index.Sections[0].Title, "Design Document")
	}

	// Verify entity refs were extracted
	foundFEAT := false
	foundTASK := false
	for _, ref := range index.EntityRefs {
		if ref.EntityID == "FEAT-001" {
			foundFEAT = true
		}
		if ref.EntityID == "TASK-042" {
			foundTASK = true
		}
	}
	if !foundFEAT {
		t.Error("expected to find FEAT-001 in entity refs")
	}
	if !foundTASK {
		t.Error("expected to find TASK-042 in entity refs")
	}

	// Verify index was persisted
	store := docint.NewIndexStore(indexRoot)
	loaded, err := store.LoadDocumentIndex("test-doc")
	if err != nil {
		t.Fatalf("LoadDocumentIndex after ingest: %v", err)
	}
	if loaded.DocumentID != "test-doc" {
		t.Errorf("persisted DocumentID = %q, want %q", loaded.DocumentID, "test-doc")
	}

	// Verify graph was saved
	graph, err := store.LoadGraph()
	if err != nil {
		t.Fatalf("LoadGraph after ingest: %v", err)
	}
	if len(graph.Edges) == 0 {
		t.Error("graph edges empty after ingest")
	}
}

func TestClassifyDocument(t *testing.T) {
	tmp := t.TempDir()
	indexRoot := filepath.Join(tmp, "index")
	docPath := writeTestDoc(t, tmp, "docs/design.md", testMarkdown)

	svc := NewIntelligenceService(indexRoot, tmp)
	t.Cleanup(func() { svc.Close() }) //nolint:errcheck

	index, err := svc.IngestDocument("test-doc", docPath)
	if err != nil {
		t.Fatalf("IngestDocument: %v", err)
	}

	submission := docint.ClassificationSubmission{
		DocumentID:   "test-doc",
		ContentHash:  index.ContentHash,
		ModelName:    "test-model",
		ModelVersion: "1.0",
		ClassifiedAt: time.Now().UTC(),
		Classifications: []docint.Classification{
			{
				SectionPath:   "1",
				Role:          "narrative",
				Confidence:    "high",
				Summary:       "Design overview",
				ConceptsIntro: []docint.ConceptIntroEntry{{Name: "workflow-design"}},
			},
		},
	}

	if err := svc.ClassifyDocument(submission); err != nil {
		t.Fatalf("ClassifyDocument: %v", err)
	}

	// Verify classifications were stored
	store := docint.NewIndexStore(indexRoot)
	loaded, err := store.LoadDocumentIndex("test-doc")
	if err != nil {
		t.Fatalf("LoadDocumentIndex after classify: %v", err)
	}
	if !loaded.Classified {
		t.Error("expected Classified = true")
	}
	if loaded.ClassifiedBy != "test-model" {
		t.Errorf("ClassifiedBy = %q, want %q", loaded.ClassifiedBy, "test-model")
	}
	if loaded.ClassifiedAt == nil {
		t.Error("expected ClassifiedAt to be non-nil")
	} else if loaded.ClassifiedAt.IsZero() {
		t.Error("expected ClassifiedAt to be non-zero")
	}
	if len(loaded.Classifications) != 1 {
		t.Fatalf("Classifications count = %d, want 1", len(loaded.Classifications))
	}
	if loaded.Classifications[0].Role != "narrative" {
		t.Errorf("Classification role = %q, want %q", loaded.Classifications[0].Role, "narrative")
	}

	// Verify concept registry was updated
	registry, err := store.LoadConceptRegistry()
	if err != nil {
		t.Fatalf("LoadConceptRegistry: %v", err)
	}
	concept := docint.FindConcept(registry, "workflow-design")
	if concept == nil {
		t.Fatal("expected concept 'workflow-design' in registry")
	}
	if len(concept.IntroducedIn) == 0 {
		t.Error("expected IntroducedIn to have entries")
	}
}

func TestClassifyDocument_ValidationError(t *testing.T) {
	tmp := t.TempDir()
	indexRoot := filepath.Join(tmp, "index")
	docPath := writeTestDoc(t, tmp, "docs/design.md", testMarkdown)

	svc := NewIntelligenceService(indexRoot, tmp)
	t.Cleanup(func() { svc.Close() }) //nolint:errcheck

	index, err := svc.IngestDocument("test-doc", docPath)
	if err != nil {
		t.Fatalf("IngestDocument: %v", err)
	}
	_ = index

	submission := docint.ClassificationSubmission{
		DocumentID:   "test-doc",
		ContentHash:  "wrong-hash",
		ModelName:    "test-model",
		ModelVersion: "1.0",
		ClassifiedAt: time.Now().UTC(),
		Classifications: []docint.Classification{
			{SectionPath: "1", Role: "narrative", Confidence: "high"},
		},
	}

	err = svc.ClassifyDocument(submission)
	if err == nil {
		t.Fatal("expected error for mismatched content hash")
	}
}

func TestGetOutline(t *testing.T) {
	tmp := t.TempDir()
	indexRoot := filepath.Join(tmp, "index")
	docPath := writeTestDoc(t, tmp, "docs/design.md", testMarkdown)

	svc := NewIntelligenceService(indexRoot, tmp)
	t.Cleanup(func() { svc.Close() }) //nolint:errcheck
	if _, err := svc.IngestDocument("test-doc", docPath); err != nil {
		t.Fatalf("IngestDocument: %v", err)
	}

	sections, err := svc.GetOutline("test-doc")
	if err != nil {
		t.Fatalf("GetOutline: %v", err)
	}

	if len(sections) == 0 {
		t.Fatal("GetOutline returned empty sections")
	}
	if sections[0].Title != "Design Document" {
		t.Errorf("first section = %q, want %q", sections[0].Title, "Design Document")
	}
	// Should have children (Overview, Requirements)
	if len(sections[0].Children) < 2 {
		t.Errorf("expected at least 2 children, got %d", len(sections[0].Children))
	}
}

func TestGetSection(t *testing.T) {
	tmp := t.TempDir()
	indexRoot := filepath.Join(tmp, "index")
	docPath := writeTestDoc(t, tmp, "docs/design.md", testMarkdown)

	svc := NewIntelligenceService(indexRoot, tmp)
	t.Cleanup(func() { svc.Close() }) //nolint:errcheck
	if _, err := svc.IngestDocument("test-doc", docPath); err != nil {
		t.Fatalf("IngestDocument: %v", err)
	}

	section, content, err := svc.GetSection("test-doc", "1")
	if err != nil {
		t.Fatalf("GetSection: %v", err)
	}
	if section == nil {
		t.Fatal("GetSection returned nil section")
	}
	if section.Title != "Design Document" {
		t.Errorf("section title = %q, want %q", section.Title, "Design Document")
	}
	if len(content) == 0 {
		t.Error("section content is empty")
	}
}

func TestGetSection_NotFound(t *testing.T) {
	tmp := t.TempDir()
	indexRoot := filepath.Join(tmp, "index")
	docPath := writeTestDoc(t, tmp, "docs/design.md", testMarkdown)

	svc := NewIntelligenceService(indexRoot, tmp)
	t.Cleanup(func() { svc.Close() }) //nolint:errcheck
	if _, err := svc.IngestDocument("test-doc", docPath); err != nil {
		t.Fatalf("IngestDocument: %v", err)
	}

	_, _, err := svc.GetSection("test-doc", "99.99")
	if err == nil {
		t.Fatal("expected error for nonexistent section")
	}
}

func TestFindByEntity(t *testing.T) {
	tmp := t.TempDir()
	indexRoot := filepath.Join(tmp, "index")
	docPath1 := writeTestDoc(t, tmp, "docs/design.md", testMarkdown)
	docPath2 := writeTestDoc(t, tmp, "docs/spec.md", testMarkdown2)

	svc := NewIntelligenceService(indexRoot, tmp)
	t.Cleanup(func() { svc.Close() }) //nolint:errcheck
	if _, err := svc.IngestDocument("doc-design", docPath1); err != nil {
		t.Fatalf("IngestDocument doc-design: %v", err)
	}
	if _, err := svc.IngestDocument("doc-spec", docPath2); err != nil {
		t.Fatalf("IngestDocument doc-spec: %v", err)
	}

	matches, err := svc.FindByEntity("FEAT-001")
	if err != nil {
		t.Fatalf("FindByEntity: %v", err)
	}

	if len(matches) < 2 {
		t.Fatalf("expected at least 2 matches for FEAT-001 across docs, got %d", len(matches))
	}

	// Both documents should appear
	docIDs := map[string]bool{}
	for _, m := range matches {
		docIDs[m.DocumentID] = true
	}
	if !docIDs["doc-design"] {
		t.Error("expected doc-design in matches")
	}
	if !docIDs["doc-spec"] {
		t.Error("expected doc-spec in matches")
	}
}

func TestFindByEntity_NoMatch(t *testing.T) {
	tmp := t.TempDir()
	indexRoot := filepath.Join(tmp, "index")
	docPath := writeTestDoc(t, tmp, "docs/design.md", testMarkdown)

	svc := NewIntelligenceService(indexRoot, tmp)
	t.Cleanup(func() { svc.Close() }) //nolint:errcheck
	if _, err := svc.IngestDocument("test-doc", docPath); err != nil {
		t.Fatalf("IngestDocument: %v", err)
	}

	matches, err := svc.FindByEntity("FEAT-999")
	if err != nil {
		t.Fatalf("FindByEntity: %v", err)
	}
	if len(matches) != 0 {
		t.Errorf("expected 0 matches, got %d", len(matches))
	}
}

func TestGetPendingClassification(t *testing.T) {
	tmp := t.TempDir()
	indexRoot := filepath.Join(tmp, "index")
	docPath := writeTestDoc(t, tmp, "docs/design.md", testMarkdown)

	svc := NewIntelligenceService(indexRoot, tmp)
	t.Cleanup(func() { svc.Close() }) //nolint:errcheck

	// Ingest without classifying
	index, err := svc.IngestDocument("test-doc", docPath)
	if err != nil {
		t.Fatalf("IngestDocument: %v", err)
	}

	pending, err := svc.GetPendingClassification()
	if err != nil {
		t.Fatalf("GetPendingClassification: %v", err)
	}

	if len(pending) != 1 {
		t.Fatalf("expected 1 pending, got %d", len(pending))
	}
	if pending[0].ID != "test-doc" {
		t.Errorf("pending[0].ID = %q, want %q", pending[0].ID, "test-doc")
	}

	// Classify the document
	submission := docint.ClassificationSubmission{
		DocumentID:   "test-doc",
		ContentHash:  index.ContentHash,
		ModelName:    "test-model",
		ModelVersion: "1.0",
		ClassifiedAt: time.Now().UTC(),
		Classifications: []docint.Classification{
			{SectionPath: "1", Role: "narrative", Confidence: "high"},
		},
	}
	if err := svc.ClassifyDocument(submission); err != nil {
		t.Fatalf("ClassifyDocument: %v", err)
	}

	pending, err = svc.GetPendingClassification()
	if err != nil {
		t.Fatalf("GetPendingClassification after classify: %v", err)
	}
	if len(pending) != 0 {
		t.Errorf("expected 0 pending after classify, got %d", len(pending))
	}
}

func TestAnalyzeGaps(t *testing.T) {
	tmp := t.TempDir()
	indexRoot := filepath.Join(tmp, "index")
	stateRoot := filepath.Join(tmp, "state")

	svc := NewIntelligenceService(indexRoot, tmp)
	t.Cleanup(func() { svc.Close() }) //nolint:errcheck
	docSvc := NewDocumentService(stateRoot, tmp)

	// Create a document file that can be submitted
	writeTestDoc(t, tmp, "docs/design.md", testMarkdown)

	// Submit a design document owned by FEAT-001
	_, err := docSvc.SubmitDocument(SubmitDocumentInput{
		Path:      "docs/design.md",
		Type:      "design",
		Title:     "Design for FEAT-001",
		Owner:     "FEAT-001",
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("SubmitDocument: %v", err)
	}

	gaps, err := svc.AnalyzeGaps("FEAT-001", docSvc)
	if err != nil {
		t.Fatalf("AnalyzeGaps: %v", err)
	}

	// Should be missing specification and dev-plan
	if len(gaps) != 2 {
		t.Fatalf("expected 2 gaps, got %d: %v", len(gaps), gaps)
	}

	gapSet := map[string]bool{}
	for _, g := range gaps {
		gapSet[g] = true
	}
	if !gapSet["specification"] {
		t.Error("expected 'specification' in gaps")
	}
	if !gapSet["dev-plan"] {
		t.Error("expected 'dev-plan' in gaps")
	}
}

func TestAnalyzeGaps_AllPresent(t *testing.T) {
	tmp := t.TempDir()
	indexRoot := filepath.Join(tmp, "index")
	stateRoot := filepath.Join(tmp, "state")

	svc := NewIntelligenceService(indexRoot, tmp)
	t.Cleanup(func() { svc.Close() }) //nolint:errcheck
	docSvc := NewDocumentService(stateRoot, tmp)

	// Create document files
	writeTestDoc(t, tmp, "docs/design.md", "# Design\n")
	writeTestDoc(t, tmp, "docs/spec.md", "# Spec\n")
	writeTestDoc(t, tmp, "docs/plan.md", "# Plan\n")

	// Submit all three document types
	for _, tc := range []struct {
		path, typ, title string
	}{
		{"docs/design.md", "design", "Design"},
		{"docs/spec.md", "specification", "Spec"},
		{"docs/plan.md", "dev-plan", "Dev Plan"},
	} {
		_, err := docSvc.SubmitDocument(SubmitDocumentInput{
			Path:      tc.path,
			Type:      tc.typ,
			Title:     tc.title,
			Owner:     "FEAT-002",
			CreatedBy: "test",
		})
		if err != nil {
			t.Fatalf("SubmitDocument %s: %v", tc.typ, err)
		}
	}

	gaps, err := svc.AnalyzeGaps("FEAT-002", docSvc)
	if err != nil {
		t.Fatalf("AnalyzeGaps: %v", err)
	}
	if len(gaps) != 0 {
		t.Errorf("expected 0 gaps when all types present, got %d: %v", len(gaps), gaps)
	}
}

func TestFindByRole(t *testing.T) {
	tmp := t.TempDir()
	indexRoot := filepath.Join(tmp, "index")

	content := "# Doc\n\n## Requirements\n\nSome requirements here.\n"
	docPath := writeTestDoc(t, tmp, "docs/design.md", content)

	svc := NewIntelligenceService(indexRoot, tmp)
	t.Cleanup(func() { svc.Close() }) //nolint:errcheck
	if _, err := svc.IngestDocument("test-doc", docPath); err != nil {
		t.Fatalf("IngestDocument: %v", err)
	}

	// "Requirements" heading should be detected as conventional role
	matches, err := svc.FindByRole("requirement", "")
	if err != nil {
		t.Fatalf("FindByRole: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("expected at least 1 match for 'requirement' role")
	}
	if matches[0].DocumentID != "test-doc" {
		t.Errorf("match document = %q, want %q", matches[0].DocumentID, "test-doc")
	}
}

func TestFindByRole_WithScope(t *testing.T) {
	tmp := t.TempDir()
	indexRoot := filepath.Join(tmp, "index")

	content := "# Doc\n\n## Requirements\n\nSome requirements.\n"
	docPath1 := writeTestDoc(t, tmp, "docs/a.md", content)
	docPath2 := writeTestDoc(t, tmp, "docs/b.md", content)

	svc := NewIntelligenceService(indexRoot, tmp)
	t.Cleanup(func() { svc.Close() }) //nolint:errcheck
	if _, err := svc.IngestDocument("doc-a", docPath1); err != nil {
		t.Fatalf("IngestDocument doc-a: %v", err)
	}
	if _, err := svc.IngestDocument("doc-b", docPath2); err != nil {
		t.Fatalf("IngestDocument doc-b: %v", err)
	}

	// Without scope: both docs
	all, err := svc.FindByRole("requirement", "")
	if err != nil {
		t.Fatalf("FindByRole all: %v", err)
	}
	if len(all) < 2 {
		t.Fatalf("expected at least 2 matches without scope, got %d", len(all))
	}

	// With scope: only doc-a
	scoped, err := svc.FindByRole("requirement", "doc-a")
	if err != nil {
		t.Fatalf("FindByRole scoped: %v", err)
	}
	if len(scoped) != 1 {
		t.Fatalf("expected 1 match with scope=doc-a, got %d", len(scoped))
	}
	if scoped[0].DocumentID != "doc-a" {
		t.Errorf("scoped match = %q, want %q", scoped[0].DocumentID, "doc-a")
	}
}

func TestFindByRole_DeduplicatesWhenLayer2And3Agree(t *testing.T) {
	tmp := t.TempDir()
	indexRoot := filepath.Join(tmp, "index")

	content := "# Doc\n\n## Requirements\n\nSome requirements here.\n"
	docPath := writeTestDoc(t, tmp, "docs/design.md", content)

	svc := NewIntelligenceService(indexRoot, tmp)
	t.Cleanup(func() { svc.Close() }) //nolint:errcheck
	index, err := svc.IngestDocument("test-doc", docPath)
	if err != nil {
		t.Fatalf("IngestDocument: %v", err)
	}

	// At this point, Layer 2 has detected "Requirements" as a conventional role
	matches1, err := svc.FindByRole("requirement", "")
	if err != nil {
		t.Fatalf("FindByRole before classification: %v", err)
	}
	if len(matches1) != 1 {
		t.Fatalf("expected 1 match from Layer 2, got %d", len(matches1))
	}

	// Now classify with Layer 3, agreeing with Layer 2 (same section, same role)
	submission := docint.ClassificationSubmission{
		DocumentID:   "test-doc",
		ContentHash:  index.ContentHash,
		ModelName:    "test-model",
		ModelVersion: "1.0",
		ClassifiedAt: time.Now().UTC(),
		Classifications: []docint.Classification{
			{
				SectionPath: matches1[0].SectionPath,
				Role:        "requirement",
				Confidence:  "high",
				Summary:     "Requirements section",
			},
		},
	}
	if err := svc.ClassifyDocument(submission); err != nil {
		t.Fatalf("ClassifyDocument: %v", err)
	}

	// After classification, should still get only 1 match (deduplicated)
	matches2, err := svc.FindByRole("requirement", "")
	if err != nil {
		t.Fatalf("FindByRole after classification: %v", err)
	}
	if len(matches2) != 1 {
		t.Fatalf("expected 1 deduplicated match after classification, got %d", len(matches2))
	}
	// Layer 3 should take precedence (higher confidence, has summary)
	if matches2[0].Summary == "" {
		t.Error("expected Layer 3 match with summary, got Layer 2 match")
	}
}

func TestGetImpact(t *testing.T) {
	tmp := t.TempDir()
	indexRoot := filepath.Join(tmp, "index")
	docPath := writeTestDoc(t, tmp, "docs/design.md", testMarkdown)

	svc := NewIntelligenceService(indexRoot, tmp)
	t.Cleanup(func() { svc.Close() }) //nolint:errcheck
	if _, err := svc.IngestDocument("test-doc", docPath); err != nil {
		t.Fatalf("IngestDocument: %v", err)
	}

	// CONTAINS edges should point to sections like "test-doc#1"
	edges, err := svc.GetImpact("test-doc#1")
	if err != nil {
		t.Fatalf("GetImpact: %v", err)
	}
	if len(edges) == 0 {
		t.Error("expected at least one edge pointing to test-doc#1")
	}
}

func TestFindByConcept(t *testing.T) {
	tmp := t.TempDir()
	indexRoot := filepath.Join(tmp, "index")
	docPath := writeTestDoc(t, tmp, "docs/design.md", testMarkdown)

	svc := NewIntelligenceService(indexRoot, tmp)
	t.Cleanup(func() { svc.Close() }) //nolint:errcheck
	index, err := svc.IngestDocument("test-doc", docPath)
	if err != nil {
		t.Fatalf("IngestDocument: %v", err)
	}

	// Classify with concepts
	submission := docint.ClassificationSubmission{
		DocumentID:   "test-doc",
		ContentHash:  index.ContentHash,
		ModelName:    "test-model",
		ModelVersion: "1.0",
		ClassifiedAt: time.Now().UTC(),
		Classifications: []docint.Classification{
			{
				SectionPath:   "1",
				Role:          "narrative",
				Confidence:    "high",
				ConceptsIntro: []docint.ConceptIntroEntry{{Name: "lifecycle-management"}},
			},
			{
				SectionPath:  "1.1",
				Role:         "narrative",
				Confidence:   "medium",
				ConceptsUsed: []string{"lifecycle-management"},
			},
		},
	}
	if err := svc.ClassifyDocument(submission); err != nil {
		t.Fatalf("ClassifyDocument: %v", err)
	}

	matches, err := svc.FindByConcept("lifecycle-management")
	if err != nil {
		t.Fatalf("FindByConcept: %v", err)
	}

	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}

	hasIntro := false
	hasUses := false
	for _, m := range matches {
		if m.Relationship == "introduces" {
			hasIntro = true
		}
		if m.Relationship == "uses" {
			hasUses = true
		}
	}
	if !hasIntro {
		t.Error("expected an 'introduces' match")
	}
	if !hasUses {
		t.Error("expected a 'uses' match")
	}
}

func TestTraceEntity(t *testing.T) {
	tmp := t.TempDir()
	indexRoot := filepath.Join(tmp, "index")

	designContent := "# Design\n\n- Type: design\n\nFEAT-010 design here.\n"
	specContent := "# Spec\n\n- Type: specification\n\nFEAT-010 spec here.\n"
	planContent := "# Plan\n\n- Type: dev-plan\n\nFEAT-010 plan here.\n"

	designPath := writeTestDoc(t, tmp, "docs/design.md", designContent)
	specPath := writeTestDoc(t, tmp, "docs/spec.md", specContent)
	planPath := writeTestDoc(t, tmp, "docs/plan.md", planContent)

	svc := NewIntelligenceService(indexRoot, tmp)
	t.Cleanup(func() { svc.Close() }) //nolint:errcheck

	// Ingest in reverse order to test sorting
	if _, err := svc.IngestDocument("doc-plan", planPath); err != nil {
		t.Fatalf("IngestDocument plan: %v", err)
	}
	if _, err := svc.IngestDocument("doc-design", designPath); err != nil {
		t.Fatalf("IngestDocument design: %v", err)
	}
	if _, err := svc.IngestDocument("doc-spec", specPath); err != nil {
		t.Fatalf("IngestDocument spec: %v", err)
	}

	matches, err := svc.TraceEntity("FEAT-010")
	if err != nil {
		t.Fatalf("TraceEntity: %v", err)
	}

	if len(matches) != 3 {
		t.Fatalf("expected 3 matches, got %d", len(matches))
	}

	// Should be ordered: design, specification, dev-plan
	if matches[0].DocumentID != "doc-design" {
		t.Errorf("first match = %q, want doc-design", matches[0].DocumentID)
	}
	if matches[1].DocumentID != "doc-spec" {
		t.Errorf("second match = %q, want doc-spec", matches[1].DocumentID)
	}
	if matches[2].DocumentID != "doc-plan" {
		t.Errorf("third match = %q, want doc-plan", matches[2].DocumentID)
	}
}

func TestParseSectionRef(t *testing.T) {
	tests := []struct {
		input       string
		wantDocID   string
		wantSection string
	}{
		{"doc-id#1.2", "doc-id", "1.2"},
		{"PROJECT/design-workflow#3", "PROJECT/design-workflow", "3"},
		{"bare-id", "bare-id", ""},
	}

	for _, tc := range tests {
		docID, section := parseSectionRef(tc.input)
		if docID != tc.wantDocID {
			t.Errorf("parseSectionRef(%q) docID = %q, want %q", tc.input, docID, tc.wantDocID)
		}
		if section != tc.wantSection {
			t.Errorf("parseSectionRef(%q) section = %q, want %q", tc.input, section, tc.wantSection)
		}
	}
}

// ─── SQLite dual-write and query tests ────────────────────────────────────────

func TestIngestDocument_SQLite_FTSRowsCreated(t *testing.T) {
	tmp := t.TempDir()
	indexRoot := filepath.Join(tmp, "index")
	docPath := writeTestDoc(t, tmp, "docs/design.md", testMarkdown)

	svc := NewIntelligenceService(indexRoot, tmp)
	t.Cleanup(func() { svc.Close() }) //nolint:errcheck
	if _, err := svc.IngestDocument("test-doc", docPath); err != nil {
		t.Fatalf("IngestDocument: %v", err)
	}

	count, err := svc.indexStore.CountFTSSectionsForDoc("test-doc")
	if err != nil {
		t.Fatalf("count fts: %v", err)
	}
	if count == 0 {
		t.Error("expected FTS rows after IngestDocument, got 0")
	}
}

func TestIngestDocument_SQLite_ReIngestReplacesFTS(t *testing.T) {
	tmp := t.TempDir()
	indexRoot := filepath.Join(tmp, "index")
	docPath := writeTestDoc(t, tmp, "docs/design.md", testMarkdown)

	svc := NewIntelligenceService(indexRoot, tmp)
	t.Cleanup(func() { svc.Close() }) //nolint:errcheck
	if _, err := svc.IngestDocument("test-doc", docPath); err != nil {
		t.Fatalf("first IngestDocument: %v", err)
	}
	if _, err := svc.IngestDocument("test-doc", docPath); err != nil {
		t.Fatalf("second IngestDocument: %v", err)
	}

	count, err := svc.indexStore.CountFTSSectionsForDoc("test-doc")
	if err != nil {
		t.Fatalf("count fts: %v", err)
	}

	// Count should match section count, not be doubled
	index, err := svc.indexStore.LoadDocumentIndex("test-doc")
	if err != nil {
		t.Fatalf("load index: %v", err)
	}
	sectionCount := countSections(index.Sections)
	if count != sectionCount {
		t.Errorf("FTS row count = %d, want %d (section count)", count, sectionCount)
	}
}

func TestIngestDocument_SQLite_EdgeCount(t *testing.T) {
	tmp := t.TempDir()
	indexRoot := filepath.Join(tmp, "index")
	docPath := writeTestDoc(t, tmp, "docs/design.md", testMarkdown)

	svc := NewIntelligenceService(indexRoot, tmp)
	t.Cleanup(func() { svc.Close() }) //nolint:errcheck
	idx, err := svc.IngestDocument("test-doc", docPath)
	if err != nil {
		t.Fatalf("IngestDocument: %v", err)
	}

	sqliteCount, err := svc.indexStore.CountEdgesForDoc("test-doc")
	if err != nil {
		t.Fatalf("count edges: %v", err)
	}

	expectedEdges := docint.BuildGraphEdges(idx)
	if sqliteCount != len(expectedEdges) {
		t.Errorf("SQLite edge count = %d, want %d", sqliteCount, len(expectedEdges))
	}
}

func TestIngestDocument_SQLite_EntityRefCount(t *testing.T) {
	tmp := t.TempDir()
	indexRoot := filepath.Join(tmp, "index")
	docPath := writeTestDoc(t, tmp, "docs/design.md", testMarkdown)

	svc := NewIntelligenceService(indexRoot, tmp)
	t.Cleanup(func() { svc.Close() }) //nolint:errcheck
	idx, err := svc.IngestDocument("test-doc", docPath)
	if err != nil {
		t.Fatalf("IngestDocument: %v", err)
	}

	count, err := svc.indexStore.CountEntityRefsForDoc("test-doc")
	if err != nil {
		t.Fatalf("count entity_refs: %v", err)
	}
	if count != len(idx.EntityRefs) {
		t.Errorf("SQLite entity_ref count = %d, want %d", count, len(idx.EntityRefs))
	}
}

func TestFindByEntity_SQLite(t *testing.T) {
	tmp := t.TempDir()
	indexRoot := filepath.Join(tmp, "index")
	docPath1 := writeTestDoc(t, tmp, "docs/design.md", testMarkdown)
	docPath2 := writeTestDoc(t, tmp, "docs/spec.md", testMarkdown2)

	svc := NewIntelligenceService(indexRoot, tmp)
	t.Cleanup(func() { svc.Close() }) //nolint:errcheck
	if _, err := svc.IngestDocument("doc-design", docPath1); err != nil {
		t.Fatalf("IngestDocument doc-design: %v", err)
	}
	if _, err := svc.IngestDocument("doc-spec", docPath2); err != nil {
		t.Fatalf("IngestDocument doc-spec: %v", err)
	}

	matches, err := svc.FindByEntity("FEAT-001")
	if err != nil {
		t.Fatalf("FindByEntity: %v", err)
	}
	if len(matches) < 2 {
		t.Fatalf("expected at least 2 matches, got %d", len(matches))
	}
	docIDs := map[string]bool{}
	for _, m := range matches {
		docIDs[m.DocumentID] = true
	}
	if !docIDs["doc-design"] || !docIDs["doc-spec"] {
		t.Errorf("expected both docs in matches, got %v", docIDs)
	}
}

func TestGetImpact_SQLite(t *testing.T) {
	tmp := t.TempDir()
	indexRoot := filepath.Join(tmp, "index")
	docPath := writeTestDoc(t, tmp, "docs/design.md", testMarkdown)

	svc := NewIntelligenceService(indexRoot, tmp)
	t.Cleanup(func() { svc.Close() }) //nolint:errcheck
	if _, err := svc.IngestDocument("test-doc", docPath); err != nil {
		t.Fatalf("IngestDocument: %v", err)
	}

	edges, err := svc.GetImpact("test-doc#1")
	if err != nil {
		t.Fatalf("GetImpact: %v", err)
	}
	if len(edges) == 0 {
		t.Error("expected at least one edge pointing to test-doc#1")
	}
}


func TestRebuildIndex_RoundTrip(t *testing.T) {
	tmp := t.TempDir()
	indexRoot := filepath.Join(tmp, "index")
	docPath1 := writeTestDoc(t, tmp, "docs/design.md", testMarkdown)
	docPath2 := writeTestDoc(t, tmp, "docs/spec.md", testMarkdown2)

	svc := NewIntelligenceService(indexRoot, tmp)
	t.Cleanup(func() { svc.Close() }) //nolint:errcheck

	if _, err := svc.IngestDocument("test-doc-1", docPath1); err != nil {
		t.Fatalf("IngestDocument 1: %v", err)
	}
	if _, err := svc.IngestDocument("test-doc-2", docPath2); err != nil {
		t.Fatalf("IngestDocument 2: %v", err)
	}

	// Record pre-rebuild FTS row counts
	preCount1, err := svc.indexStore.CountFTSSectionsForDoc("test-doc-1")
	if err != nil {
		t.Fatalf("pre-rebuild count 1: %v", err)
	}
	preCount2, err := svc.indexStore.CountFTSSectionsForDoc("test-doc-2")
	if err != nil {
		t.Fatalf("pre-rebuild count 2: %v", err)
	}
	if preCount1 == 0 || preCount2 == 0 {
		t.Fatal("expected non-zero FTS rows before rebuild")
	}

	// Rebuild the index
	stats, err := svc.RebuildIndex()
	if err != nil {
		t.Fatalf("RebuildIndex: %v", err)
	}

	if stats.Documents != 2 {
		t.Errorf("stats.Documents = %d, want 2", stats.Documents)
	}
	if stats.Failed != 0 {
		t.Errorf("stats.Failed = %d, want 0", stats.Failed)
	}
	if stats.FTSSections == 0 {
		t.Error("expected FTSSections > 0 after rebuild")
	}

	// Verify FTS row counts are restored after rebuild
	postCount1, err := svc.indexStore.CountFTSSectionsForDoc("test-doc-1")
	if err != nil {
		t.Fatalf("post-rebuild count 1: %v", err)
	}
	postCount2, err := svc.indexStore.CountFTSSectionsForDoc("test-doc-2")
	if err != nil {
		t.Fatalf("post-rebuild count 2: %v", err)
	}

	if postCount1 != preCount1 {
		t.Errorf("test-doc-1 FTS row count after rebuild = %d, want %d", postCount1, preCount1)
	}
	if postCount2 != preCount2 {
		t.Errorf("test-doc-2 FTS row count after rebuild = %d, want %d", postCount2, preCount2)
	}
}

func TestClassifyDocument_SQLite_EdgesDualWrite(t *testing.T) {
	tmp := t.TempDir()
	indexRoot := filepath.Join(tmp, "index")
	docPath := writeTestDoc(t, tmp, "docs/design.md", testMarkdown)

	svc := NewIntelligenceService(indexRoot, tmp)
	t.Cleanup(func() { svc.Close() }) //nolint:errcheck

	idx, err := svc.IngestDocument("test-doc", docPath)
	if err != nil {
		t.Fatalf("IngestDocument: %v", err)
	}

	preEdgeCount, err := svc.indexStore.CountEdgesForDoc("test-doc")
	if err != nil {
		t.Fatalf("pre-classify edge count: %v", err)
	}

	// Apply a minimal valid classification
	submission := docint.ClassificationSubmission{
		DocumentID:      "test-doc",
		ContentHash:     idx.ContentHash,
		ModelName:       "test-model",
		ModelVersion:    "v1",
		Classifications: []docint.Classification{},
	}
	if err := svc.ClassifyDocument(submission); err != nil {
		t.Fatalf("ClassifyDocument: %v", err)
	}

	// Edge count in SQLite should remain consistent after classify
	postEdgeCount, err := svc.indexStore.CountEdgesForDoc("test-doc")
	if err != nil {
		t.Fatalf("post-classify edge count: %v", err)
	}

	if postEdgeCount != preEdgeCount {
		t.Errorf("edge count changed after classify: before=%d after=%d (expected equal)", preEdgeCount, postEdgeCount)
	}
}
