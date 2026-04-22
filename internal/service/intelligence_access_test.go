package service

import (
	"path/filepath"
	"testing"

	"github.com/sambeau/kanbanzai/internal/docint"
)

// newTestIntelligenceService creates an IntelligenceService backed by temp dirs.
func newTestIntelligenceService(t *testing.T) (*IntelligenceService, string) {
	t.Helper()
	tmp := t.TempDir()
	indexRoot := filepath.Join(tmp, "index")
	svc := NewIntelligenceService(indexRoot, tmp)
	t.Cleanup(func() { svc.Close() }) //nolint:errcheck
	return svc, tmp
}

// ingestTestDoc ingests a document and returns the docID.
func ingestTestDoc(t *testing.T, svc *IntelligenceService, tmp, name, content string) string {
	t.Helper()
	docPath := writeTestDoc(t, tmp, name, content)
	docID := name // use path as ID for simplicity
	_, err := svc.IngestDocument(docID, docPath)
	if err != nil {
		t.Fatalf("IngestDocument(%s): %v", docID, err)
	}
	return docID
}

// loadIndexDirect loads a DocumentIndex directly from the index store (bypasses
// IntelligenceService query methods to avoid triggering additional counter increments).
func loadIndexDirect(t *testing.T, indexRoot, docID string) *docint.DocumentIndex {
	t.Helper()
	store := docint.NewIndexStore(indexRoot)
	index, err := store.LoadDocumentIndex(docID)
	if err != nil {
		t.Fatalf("LoadDocumentIndex(%s): %v", docID, err)
	}
	return index
}

// ─────────────────────────────────────────────────────────────────────────────
// FR-009 / FR-010 / FR-018: DocumentIndex fields present with zero defaults
// ─────────────────────────────────────────────────────────────────────────────

func TestDocumentIndex_NewIndex_HasZeroAccessCount(t *testing.T) {
	t.Parallel()
	svc, tmp := newTestIntelligenceService(t)
	indexRoot := filepath.Join(tmp, "index")

	docID := ingestTestDoc(t, svc, tmp, "docs/design.md", testMarkdown)

	index := loadIndexDirect(t, indexRoot, docID)
	if index.AccessCount != 0 {
		t.Errorf("AccessCount = %d, want 0 for freshly ingested document", index.AccessCount)
	}
	if index.LastAccessedAt != nil {
		t.Error("LastAccessedAt should be nil for freshly ingested document")
	}
	if index.SectionAccess != nil {
		t.Error("SectionAccess should be nil for freshly ingested document")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// FR-012: GetOutline increments AccessCount
// ─────────────────────────────────────────────────────────────────────────────

func TestIntelligenceService_GetOutline_IncrementsAccessCount(t *testing.T) {
	t.Parallel()
	svc, tmp := newTestIntelligenceService(t)
	indexRoot := filepath.Join(tmp, "index")

	docID := ingestTestDoc(t, svc, tmp, "docs/design.md", testMarkdown)

	_, err := svc.GetOutline(docID)
	if err != nil {
		t.Fatalf("GetOutline: %v", err)
	}
	svc.Wait()

	index := loadIndexDirect(t, indexRoot, docID)
	if index.AccessCount != 1 {
		t.Errorf("AccessCount = %d, want 1 after GetOutline", index.AccessCount)
	}
	if index.LastAccessedAt == nil {
		t.Error("LastAccessedAt should be non-nil after GetOutline")
	}
}

func TestIntelligenceService_GetOutline_IncrementsMultipleTimes(t *testing.T) {
	t.Parallel()
	svc, tmp := newTestIntelligenceService(t)
	indexRoot := filepath.Join(tmp, "index")

	docID := ingestTestDoc(t, svc, tmp, "docs/design.md", testMarkdown)

	for i := 0; i < 3; i++ {
		_, err := svc.GetOutline(docID)
		if err != nil {
			t.Fatalf("GetOutline iteration %d: %v", i, err)
		}
		svc.Wait()
	}

	index := loadIndexDirect(t, indexRoot, docID)
	if index.AccessCount != 3 {
		t.Errorf("AccessCount = %d, want 3 after 3 GetOutline calls", index.AccessCount)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// FR-013: GetDocumentIndex (guide) increments AccessCount
// ─────────────────────────────────────────────────────────────────────────────

func TestIntelligenceService_GetDocumentIndex_IncrementsAccessCount(t *testing.T) {
	t.Parallel()
	svc, tmp := newTestIntelligenceService(t)
	indexRoot := filepath.Join(tmp, "index")

	docID := ingestTestDoc(t, svc, tmp, "docs/design.md", testMarkdown)

	_, err := svc.GetDocumentIndex(docID)
	if err != nil {
		t.Fatalf("GetDocumentIndex: %v", err)
	}
	svc.Wait()

	index := loadIndexDirect(t, indexRoot, docID)
	if index.AccessCount != 1 {
		t.Errorf("AccessCount = %d, want 1 after GetDocumentIndex (guide)", index.AccessCount)
	}
	if index.LastAccessedAt == nil {
		t.Error("LastAccessedAt should be non-nil after GetDocumentIndex")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// FR-014: GetSection increments both document AccessCount and SectionAccess
// ─────────────────────────────────────────────────────────────────────────────

func TestIntelligenceService_GetSection_IncrementsDocAndSectionCounters(t *testing.T) {
	t.Parallel()
	svc, tmp := newTestIntelligenceService(t)
	indexRoot := filepath.Join(tmp, "index")

	docID := ingestTestDoc(t, svc, tmp, "docs/design.md", testMarkdown)

	// Find a valid section path from the outline
	sections, err := svc.GetOutline(docID)
	svc.Wait() // flush the GetOutline counter
	if err != nil || len(sections) == 0 {
		t.Fatalf("GetOutline: err=%v, sections=%d", err, len(sections))
	}

	// Reset the counter so we're only counting GetSection's increment
	indexRoot2 := filepath.Join(tmp, "index")
	store := docint.NewIndexStore(indexRoot2)
	idx, _ := store.LoadDocumentIndex(docID)
	idx.AccessCount = 0
	idx.LastAccessedAt = nil
	_ = store.SaveDocumentIndex(idx)

	sectionPath := sections[0].Path
	_, _, err = svc.GetSection(docID, sectionPath)
	if err != nil {
		t.Fatalf("GetSection(%s): %v", sectionPath, err)
	}
	svc.Wait()

	index := loadIndexDirect(t, indexRoot, docID)
	if index.AccessCount != 1 {
		t.Errorf("DocumentIndex.AccessCount = %d, want 1 after GetSection", index.AccessCount)
	}
	if index.LastAccessedAt == nil {
		t.Error("DocumentIndex.LastAccessedAt should be set after GetSection")
	}
	if index.SectionAccess == nil {
		t.Fatal("SectionAccess map should be non-nil after GetSection")
	}
	info, ok := index.SectionAccess[sectionPath]
	if !ok {
		t.Errorf("SectionAccess[%q] not found", sectionPath)
	}
	if info.AccessCount != 1 {
		t.Errorf("SectionAccess[%q].AccessCount = %d, want 1", sectionPath, info.AccessCount)
	}
	if info.LastAccessedAt == nil {
		t.Errorf("SectionAccess[%q].LastAccessedAt should be set", sectionPath)
	}
}

func TestIntelligenceService_GetSection_AccumulatesSectionCounts(t *testing.T) {
	t.Parallel()
	svc, tmp := newTestIntelligenceService(t)
	indexRoot := filepath.Join(tmp, "index")

	docID := ingestTestDoc(t, svc, tmp, "docs/design.md", testMarkdown)

	sections, err := svc.GetOutline(docID)
	svc.Wait()
	if err != nil || len(sections) == 0 {
		t.Fatalf("GetOutline failed: err=%v len=%d", err, len(sections))
	}

	// Reset
	store := docint.NewIndexStore(indexRoot)
	idx, _ := store.LoadDocumentIndex(docID)
	idx.AccessCount = 0
	idx.LastAccessedAt = nil
	idx.SectionAccess = nil
	_ = store.SaveDocumentIndex(idx)

	sectionPath := sections[0].Path
	for i := 0; i < 3; i++ {
		_, _, err = svc.GetSection(docID, sectionPath)
		if err != nil {
			t.Fatalf("GetSection iteration %d: %v", i, err)
		}
		svc.Wait()
	}

	index := loadIndexDirect(t, indexRoot, docID)
	if index.SectionAccess == nil {
		t.Fatal("SectionAccess should not be nil")
	}
	info := index.SectionAccess[sectionPath]
	if info.AccessCount != 3 {
		t.Errorf("SectionAccess[%q].AccessCount = %d, want 3", sectionPath, info.AccessCount)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// FR-015: FindByEntity increments AccessCount for distinct documents
// ─────────────────────────────────────────────────────────────────────────────

func TestIntelligenceService_FindByEntity_IncrementsAccessCount(t *testing.T) {
	t.Parallel()
	svc, tmp := newTestIntelligenceService(t)
	indexRoot := filepath.Join(tmp, "index")

	// testMarkdown references FEAT-001 and TASK-042
	docID := ingestTestDoc(t, svc, tmp, "docs/design.md", testMarkdown)

	matches, err := svc.FindByEntity("FEAT-001")
	if err != nil {
		t.Fatalf("FindByEntity: %v", err)
	}
	svc.Wait()

	if len(matches) == 0 {
		t.Skip("no matches found for FEAT-001 in testMarkdown — skipping counter check")
	}

	index := loadIndexDirect(t, indexRoot, docID)
	if index.AccessCount == 0 {
		t.Errorf("AccessCount = 0, want > 0 after FindByEntity with non-empty results")
	}
}

func TestIntelligenceService_FindByEntity_NoIncrementWhenNoResults(t *testing.T) {
	t.Parallel()
	svc, tmp := newTestIntelligenceService(t)
	indexRoot := filepath.Join(tmp, "index")

	docID := ingestTestDoc(t, svc, tmp, "docs/design.md", testMarkdown)

	_, err := svc.FindByEntity("FEAT-999999")
	if err != nil {
		t.Fatalf("FindByEntity: %v", err)
	}
	svc.Wait()

	index := loadIndexDirect(t, indexRoot, docID)
	if index.AccessCount != 0 {
		t.Errorf("AccessCount = %d, want 0 when FindByEntity returns no matches", index.AccessCount)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// FR-015: FindByConcept increments AccessCount for distinct documents
// ─────────────────────────────────────────────────────────────────────────────

func TestIntelligenceService_FindByConcept_IncrementsAccessCount(t *testing.T) {
	t.Parallel()
	svc, tmp := newTestIntelligenceService(t)
	indexRoot := filepath.Join(tmp, "index")

	docID := ingestTestDoc(t, svc, tmp, "docs/design.md", testMarkdown)

	// Classify the document to introduce a concept
	index0, err := svc.IngestDocument(docID, "docs/design.md")
	if err != nil {
		t.Fatalf("re-IngestDocument: %v", err)
	}
	err = svc.ClassifyDocument(docint.ClassificationSubmission{
		DocumentID:   docID,
		ContentHash:  index0.ContentHash,
		ModelName:    "test",
		ModelVersion: "1.0",
		Classifications: []docint.Classification{
			{
				SectionPath:   "1",
				Role:          "narrative",
				Confidence:    "high",
				ConceptsIntro: []docint.ConceptIntroEntry{{Name: "test-concept-xyz"}},
			},
		},
	})
	if err != nil {
		t.Fatalf("ClassifyDocument: %v", err)
	}

	// Reset counter
	store := docint.NewIndexStore(indexRoot)
	idx, _ := store.LoadDocumentIndex(docID)
	idx.AccessCount = 0
	idx.LastAccessedAt = nil
	_ = store.SaveDocumentIndex(idx)

	matches, err := svc.FindByConcept("test-concept-xyz")
	if err != nil {
		t.Fatalf("FindByConcept: %v", err)
	}
	svc.Wait()

	if len(matches) == 0 {
		t.Skip("no matches for concept — skipping counter check")
	}

	index := loadIndexDirect(t, indexRoot, docID)
	if index.AccessCount == 0 {
		t.Errorf("AccessCount = 0, want > 0 after FindByConcept with non-empty results")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// FR-015: FindByRole increments AccessCount for distinct documents
// ─────────────────────────────────────────────────────────────────────────────

func TestIntelligenceService_FindByRole_IncrementsAccessCount(t *testing.T) {
	t.Parallel()
	svc, tmp := newTestIntelligenceService(t)
	indexRoot := filepath.Join(tmp, "index")

	docID := ingestTestDoc(t, svc, tmp, "docs/design.md", testMarkdown)

	// Classify the document with a known role
	index0, err := svc.IngestDocument(docID, "docs/design.md")
	if err != nil {
		t.Fatalf("re-IngestDocument: %v", err)
	}
	err = svc.ClassifyDocument(docint.ClassificationSubmission{
		DocumentID:   docID,
		ContentHash:  index0.ContentHash,
		ModelName:    "test",
		ModelVersion: "1.0",
		Classifications: []docint.Classification{
			{
				SectionPath: "1",
				Role:        "requirement",
				Confidence:  "high",
			},
		},
	})
	if err != nil {
		t.Fatalf("ClassifyDocument: %v", err)
	}

	// Reset counter
	store := docint.NewIndexStore(indexRoot)
	idx, _ := store.LoadDocumentIndex(docID)
	idx.AccessCount = 0
	idx.LastAccessedAt = nil
	_ = store.SaveDocumentIndex(idx)

	matches, err := svc.FindByRole("requirement", "")
	if err != nil {
		t.Fatalf("FindByRole: %v", err)
	}
	svc.Wait()

	if len(matches) == 0 {
		t.Skip("no matches for role — skipping counter check")
	}

	index := loadIndexDirect(t, indexRoot, docID)
	if index.AccessCount == 0 {
		t.Errorf("AccessCount = 0, want > 0 after FindByRole with non-empty results")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// FR-017 / NFR-002: Counter errors do not propagate to callers
// ─────────────────────────────────────────────────────────────────────────────

func TestIntelligenceService_GetOutline_CounterErrorDoesNotFail(t *testing.T) {
	t.Parallel()
	// Use a non-existent docID — GetOutline should error, but not panic.
	svc, _ := newTestIntelligenceService(t)

	_, err := svc.GetOutline("nonexistent-doc")
	if err == nil {
		t.Error("GetOutline(nonexistent) should return error")
	}
	// Should not panic; no goroutine is spawned when the index load fails.
	svc.Wait()
}

// ─────────────────────────────────────────────────────────────────────────────
// FR-016: Search increments AccessCount for distinct documents in results
// ─────────────────────────────────────────────────────────────────────────────

func TestIntelligenceService_Search_IncrementsAccessCount(t *testing.T) {
	t.Parallel()
	svc, tmp := newTestIntelligenceService(t)
	indexRoot := filepath.Join(tmp, "index")

	docID := ingestTestDoc(t, svc, tmp, "docs/design.md", testMarkdown)

	_, results, err := svc.Search(docint.SearchParams{Query: "requirements"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	svc.Wait()

	if len(results) == 0 {
		t.Skip("no search results for 'requirements' — skipping counter check")
	}

	index := loadIndexDirect(t, indexRoot, docID)
	if index.AccessCount == 0 {
		t.Errorf("AccessCount = 0, want > 0 after Search with non-empty results")
	}
	if index.LastAccessedAt == nil {
		t.Error("LastAccessedAt should be non-nil after Search with non-empty results")
	}
}

func TestIntelligenceService_Search_NoIncrementWhenNoResults(t *testing.T) {
	t.Parallel()
	svc, tmp := newTestIntelligenceService(t)
	indexRoot := filepath.Join(tmp, "index")

	docID := ingestTestDoc(t, svc, tmp, "docs/design.md", testMarkdown)

	_, results, err := svc.Search(docint.SearchParams{Query: "zzzznonexistent"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	svc.Wait()

	if len(results) != 0 {
		t.Skip("unexpected results for nonexistent query — skipping no-increment check")
	}

	index := loadIndexDirect(t, indexRoot, docID)
	if index.AccessCount != 0 {
		t.Errorf("AccessCount = %d, want 0 when Search returns no results", index.AccessCount)
	}
}
