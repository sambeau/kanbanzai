package service

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sambeau/kanbanzai/internal/docint"
)

// seedIndexWithAccessCount creates a document index in the given indexRoot with
// the provided AccessCount and (optionally) a LastAccessedAt timestamp.
func seedIndexWithAccessCount(t *testing.T, indexRoot, docID, docPath string, accessCount int, lastAccessed *time.Time) {
	t.Helper()
	store := docint.NewIndexStore(indexRoot)
	index := &docint.DocumentIndex{
		DocumentID:     docID,
		DocumentPath:   docPath,
		ContentHash:    "deadbeef",
		IndexedAt:      time.Now().UTC(),
		AccessCount:    accessCount,
		LastAccessedAt: lastAccessed,
	}
	if err := store.SaveDocumentIndex(index); err != nil {
		t.Fatalf("seedIndexWithAccessCount(%s): %v", docID, err)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// FR-019 / FR-020: AuditResult.MostAccessed population
// ─────────────────────────────────────────────────────────────────────────────

func TestAuditDocuments_MostAccessed_PopulatedFromIntelSvc(t *testing.T) {
	t.Parallel()

	docSvc, repoRoot := newAuditTestSetup(t)
	indexRoot := filepath.Join(t.TempDir(), "index")
	intelSvc := NewIntelligenceService(indexRoot, repoRoot)
	t.Cleanup(func() { intelSvc.Close() }) //nolint:errcheck

	now := time.Now().UTC()

	// Seed three document indexes with different access counts
	seedIndexWithAccessCount(t, indexRoot, "DOC-001", "work/spec/a.md", 5, &now)
	seedIndexWithAccessCount(t, indexRoot, "DOC-002", "work/spec/b.md", 10, &now)
	seedIndexWithAccessCount(t, indexRoot, "DOC-003", "work/spec/c.md", 1, &now)

	result, err := AuditDocuments(context.Background(), docSvc, repoRoot, nil, false, intelSvc)
	if err != nil {
		t.Fatalf("AuditDocuments: %v", err)
	}

	if len(result.MostAccessed) != 3 {
		t.Fatalf("MostAccessed len = %d, want 3", len(result.MostAccessed))
	}

	// Verify descending order
	if result.MostAccessed[0].AccessCount != 10 {
		t.Errorf("MostAccessed[0].AccessCount = %d, want 10", result.MostAccessed[0].AccessCount)
	}
	if result.MostAccessed[1].AccessCount != 5 {
		t.Errorf("MostAccessed[1].AccessCount = %d, want 5", result.MostAccessed[1].AccessCount)
	}
	if result.MostAccessed[2].AccessCount != 1 {
		t.Errorf("MostAccessed[2].AccessCount = %d, want 1", result.MostAccessed[2].AccessCount)
	}
}

func TestAuditDocuments_MostAccessed_ExcludesZeroAccessCount(t *testing.T) {
	t.Parallel()

	docSvc, repoRoot := newAuditTestSetup(t)
	indexRoot := filepath.Join(t.TempDir(), "index")
	intelSvc := NewIntelligenceService(indexRoot, repoRoot)
	t.Cleanup(func() { intelSvc.Close() }) //nolint:errcheck

	now := time.Now().UTC()

	// One with AccessCount > 0, one with AccessCount == 0
	seedIndexWithAccessCount(t, indexRoot, "DOC-A", "work/spec/a.md", 3, &now)
	seedIndexWithAccessCount(t, indexRoot, "DOC-B", "work/spec/b.md", 0, &now)

	result, err := AuditDocuments(context.Background(), docSvc, repoRoot, nil, false, intelSvc)
	if err != nil {
		t.Fatalf("AuditDocuments: %v", err)
	}

	if len(result.MostAccessed) != 1 {
		t.Fatalf("MostAccessed len = %d, want 1 (AccessCount==0 excluded)", len(result.MostAccessed))
	}
	if result.MostAccessed[0].DocID != "DOC-A" {
		t.Errorf("MostAccessed[0].DocID = %q, want DOC-A", result.MostAccessed[0].DocID)
	}
}

func TestAuditDocuments_MostAccessed_ExcludesNilLastAccessedAt(t *testing.T) {
	t.Parallel()

	docSvc, repoRoot := newAuditTestSetup(t)
	indexRoot := filepath.Join(t.TempDir(), "index")
	intelSvc := NewIntelligenceService(indexRoot, repoRoot)
	t.Cleanup(func() { intelSvc.Close() }) //nolint:errcheck

	now := time.Now().UTC()

	// One with LastAccessedAt set, one with nil
	seedIndexWithAccessCount(t, indexRoot, "DOC-X", "work/spec/x.md", 7, &now)
	seedIndexWithAccessCount(t, indexRoot, "DOC-Y", "work/spec/y.md", 3, nil)

	result, err := AuditDocuments(context.Background(), docSvc, repoRoot, nil, false, intelSvc)
	if err != nil {
		t.Fatalf("AuditDocuments: %v", err)
	}

	if len(result.MostAccessed) != 1 {
		t.Fatalf("MostAccessed len = %d, want 1 (nil LastAccessedAt excluded)", len(result.MostAccessed))
	}
	if result.MostAccessed[0].DocID != "DOC-X" {
		t.Errorf("MostAccessed[0].DocID = %q, want DOC-X", result.MostAccessed[0].DocID)
	}
}

func TestAuditDocuments_MostAccessed_CapAt10(t *testing.T) {
	t.Parallel()

	docSvc, repoRoot := newAuditTestSetup(t)
	indexRoot := filepath.Join(t.TempDir(), "index")
	intelSvc := NewIntelligenceService(indexRoot, repoRoot)
	t.Cleanup(func() { intelSvc.Close() }) //nolint:errcheck

	now := time.Now().UTC()

	// Seed 15 documents with varying access counts
	for i := 1; i <= 15; i++ {
		docID := "DOC-" + string(rune('A'+i-1))
		seedIndexWithAccessCount(t, indexRoot, docID, "work/spec/doc.md", i, &now)
	}

	result, err := AuditDocuments(context.Background(), docSvc, repoRoot, nil, false, intelSvc)
	if err != nil {
		t.Fatalf("AuditDocuments: %v", err)
	}

	if len(result.MostAccessed) != 10 {
		t.Errorf("MostAccessed len = %d, want 10 (capped at 10)", len(result.MostAccessed))
	}

	// The top entry should be the highest access count (15)
	if result.MostAccessed[0].AccessCount != 15 {
		t.Errorf("MostAccessed[0].AccessCount = %d, want 15", result.MostAccessed[0].AccessCount)
	}
}

func TestAuditDocuments_MostAccessed_EmptyWhenNoIntelSvc(t *testing.T) {
	t.Parallel()

	docSvc, repoRoot := newAuditTestSetup(t)

	// Call without intelligenceSvc — MostAccessed should be nil/empty
	result, err := AuditDocuments(context.Background(), docSvc, repoRoot, nil, false)
	if err != nil {
		t.Fatalf("AuditDocuments: %v", err)
	}

	if len(result.MostAccessed) != 0 {
		t.Errorf("MostAccessed len = %d, want 0 when no IntelligenceService provided", len(result.MostAccessed))
	}
}

func TestAuditDocuments_MostAccessed_EmptyWhenNoIndexes(t *testing.T) {
	t.Parallel()

	docSvc, repoRoot := newAuditTestSetup(t)
	indexRoot := filepath.Join(t.TempDir(), "index")
	intelSvc := NewIntelligenceService(indexRoot, repoRoot)
	t.Cleanup(func() { intelSvc.Close() }) //nolint:errcheck

	// No document indexes seeded
	result, err := AuditDocuments(context.Background(), docSvc, repoRoot, nil, false, intelSvc)
	if err != nil {
		t.Fatalf("AuditDocuments: %v", err)
	}

	if len(result.MostAccessed) != 0 {
		t.Errorf("MostAccessed len = %d, want 0 when no indexes exist", len(result.MostAccessed))
	}
}

func TestAuditDocuments_MostAccessed_ContainsCorrectFields(t *testing.T) {
	t.Parallel()

	docSvc, repoRoot := newAuditTestSetup(t)
	indexRoot := filepath.Join(t.TempDir(), "index")
	intelSvc := NewIntelligenceService(indexRoot, repoRoot)
	t.Cleanup(func() { intelSvc.Close() }) //nolint:errcheck

	now := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	seedIndexWithAccessCount(t, indexRoot, "DOC-FIELDS", "work/spec/fields.md", 42, &now)

	result, err := AuditDocuments(context.Background(), docSvc, repoRoot, nil, false, intelSvc)
	if err != nil {
		t.Fatalf("AuditDocuments: %v", err)
	}

	if len(result.MostAccessed) != 1 {
		t.Fatalf("MostAccessed len = %d, want 1", len(result.MostAccessed))
	}

	entry := result.MostAccessed[0]
	if entry.DocID != "DOC-FIELDS" {
		t.Errorf("DocID = %q, want DOC-FIELDS", entry.DocID)
	}
	if entry.Path != "work/spec/fields.md" {
		t.Errorf("Path = %q, want work/spec/fields.md", entry.Path)
	}
	if entry.AccessCount != 42 {
		t.Errorf("AccessCount = %d, want 42", entry.AccessCount)
	}
	if entry.LastAccessedAt == nil {
		t.Fatal("LastAccessedAt should not be nil")
	}
	if !entry.LastAccessedAt.Equal(now) {
		t.Errorf("LastAccessedAt = %v, want %v", entry.LastAccessedAt, now)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// FR-021: MCP audit renders MostAccessed as Markdown table
// ─────────────────────────────────────────────────────────────────────────────

func TestRenderMostAccessedTable_BasicOutput(t *testing.T) {
	t.Parallel()

	now := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	entries := []AccessedDocumentEntry{
		{DocID: "DOC-001", Path: "work/spec/a.md", AccessCount: 10, LastAccessedAt: &now},
		{DocID: "DOC-002", Path: "work/spec/b.md", AccessCount: 5, LastAccessedAt: &now},
	}

	table := RenderMostAccessedTable(entries)

	if !strings.Contains(table, "Most Accessed Documents") {
		t.Error("table should contain 'Most Accessed Documents' heading")
	}
	if !strings.Contains(table, "| Rank |") {
		t.Error("table should have Rank column header")
	}
	if !strings.Contains(table, "| Path |") {
		t.Error("table should have Path column header")
	}
	if !strings.Contains(table, "| Access Count |") {
		t.Error("table should have Access Count column header")
	}
	if !strings.Contains(table, "| Last Accessed |") {
		t.Error("table should have Last Accessed column header")
	}
	if !strings.Contains(table, "work/spec/a.md") {
		t.Error("table should contain first entry path")
	}
	if !strings.Contains(table, "work/spec/b.md") {
		t.Error("table should contain second entry path")
	}
	if !strings.Contains(table, "10") {
		t.Error("table should contain access count 10")
	}
	if !strings.Contains(table, "| 1 |") {
		t.Error("table should have rank 1")
	}
	if !strings.Contains(table, "| 2 |") {
		t.Error("table should have rank 2")
	}
}

func TestRenderMostAccessedTable_DateFormat(t *testing.T) {
	t.Parallel()

	ts := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	entries := []AccessedDocumentEntry{
		{DocID: "DOC-001", Path: "docs/spec.md", AccessCount: 7, LastAccessedAt: &ts},
	}

	table := RenderMostAccessedTable(entries)

	if !strings.Contains(table, "2024-03-15") {
		t.Errorf("table should contain formatted date 2024-03-15, got:\n%s", table)
	}
}

func TestRenderMostAccessedTable_NilLastAccessedAt(t *testing.T) {
	t.Parallel()

	entries := []AccessedDocumentEntry{
		{DocID: "DOC-001", Path: "docs/spec.md", AccessCount: 3, LastAccessedAt: nil},
	}

	table := RenderMostAccessedTable(entries)

	// Should render a dash or placeholder for nil LastAccessedAt
	if !strings.Contains(table, "—") && !strings.Contains(table, "-") {
		t.Errorf("table should contain placeholder for nil LastAccessedAt, got:\n%s", table)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// FR-020: MostAccessed ordering is strictly descending by AccessCount
// ─────────────────────────────────────────────────────────────────────────────

func TestAuditDocuments_MostAccessed_DescendingOrder(t *testing.T) {
	t.Parallel()

	docSvc, repoRoot := newAuditTestSetup(t)
	indexRoot := filepath.Join(t.TempDir(), "index")
	intelSvc := NewIntelligenceService(indexRoot, repoRoot)
	t.Cleanup(func() { intelSvc.Close() }) //nolint:errcheck

	now := time.Now().UTC()

	// Seed in non-sorted order
	counts := []int{3, 9, 1, 7, 5}
	for i, c := range counts {
		docID := "DOC-ORDER-" + string(rune('A'+i))
		seedIndexWithAccessCount(t, indexRoot, docID, "docs/x.md", c, &now)
	}

	result, err := AuditDocuments(context.Background(), docSvc, repoRoot, nil, false, intelSvc)
	if err != nil {
		t.Fatalf("AuditDocuments: %v", err)
	}

	if len(result.MostAccessed) != 5 {
		t.Fatalf("MostAccessed len = %d, want 5", len(result.MostAccessed))
	}

	for i := 1; i < len(result.MostAccessed); i++ {
		prev := result.MostAccessed[i-1].AccessCount
		curr := result.MostAccessed[i].AccessCount
		if prev < curr {
			t.Errorf("MostAccessed not descending: [%d]=%d < [%d]=%d",
				i-1, prev, i, curr)
		}
	}

	// First entry must be highest (9)
	if result.MostAccessed[0].AccessCount != 9 {
		t.Errorf("first entry AccessCount = %d, want 9", result.MostAccessed[0].AccessCount)
	}
}
