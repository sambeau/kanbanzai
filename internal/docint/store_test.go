package docint

import (
	"testing"
	"time"
)

func TestIndexStore_SaveAndLoad(t *testing.T) {
	store := NewIndexStore(t.TempDir())

	now := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
	original := &DocumentIndex{
		DocumentID:   "work/design/test.md",
		DocumentPath: "work/design/test.md",
		ContentHash:  "abc123",
		IndexedAt:    now,
		Sections: []Section{
			{Path: "1", Level: 1, Title: "Overview"},
		},
	}

	if err := store.SaveDocumentIndex(original); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := store.LoadDocumentIndex("work/design/test.md")
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if loaded.DocumentID != original.DocumentID {
		t.Errorf("DocumentID = %q, want %q", loaded.DocumentID, original.DocumentID)
	}
	if loaded.ContentHash != original.ContentHash {
		t.Errorf("ContentHash = %q, want %q", loaded.ContentHash, original.ContentHash)
	}
	if len(loaded.Sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(loaded.Sections))
	}
	if loaded.Sections[0].Title != "Overview" {
		t.Errorf("section title = %q, want %q", loaded.Sections[0].Title, "Overview")
	}
}

func TestIndexStore_SaveAndLoad_PreservesClassifications(t *testing.T) {
	store := NewIndexStore(t.TempDir())

	classifiedAt := time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC)
	original := &DocumentIndex{
		DocumentID:        "work/design/classified.md",
		DocumentPath:      "work/design/classified.md",
		ContentHash:       "def456",
		IndexedAt:         time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC),
		Classified:        true,
		ClassifiedAt:      &classifiedAt,
		ClassifiedBy:      "gpt-4",
		ClassifierVersion: "2024-01-01",
		Classifications: []Classification{
			{
				SectionPath:   "1",
				Role:          "narrative",
				Confidence:    "high",
				ConceptsIntro: []ConceptIntroEntry{{Name: "lifecycle-states"}},
			},
			{
				SectionPath:  "2",
				Role:         "requirement",
				Confidence:   "medium",
				ConceptsUsed: []string{"lifecycle-states", "entity-model"},
			},
		},
	}

	if err := store.SaveDocumentIndex(original); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := store.LoadDocumentIndex("work/design/classified.md")
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if !loaded.Classified {
		t.Error("expected Classified to be true")
	}
	if loaded.ClassifiedBy != "gpt-4" {
		t.Errorf("ClassifiedBy = %q, want %q", loaded.ClassifiedBy, "gpt-4")
	}
	if len(loaded.Classifications) != 2 {
		t.Fatalf("expected 2 classifications, got %d", len(loaded.Classifications))
	}
	if len(loaded.Classifications[0].ConceptsIntro) != 1 {
		t.Errorf("first classification concepts_intro count = %d, want 1", len(loaded.Classifications[0].ConceptsIntro))
	}
	if len(loaded.Classifications[1].ConceptsUsed) != 2 {
		t.Errorf("second classification concepts_used count = %d, want 2", len(loaded.Classifications[1].ConceptsUsed))
	}
}

func TestIndexStore_ListDocumentIndexes(t *testing.T) {
	store := NewIndexStore(t.TempDir())

	// Save two indexes
	for _, id := range []string{"work/design/a.md", "work/design/b.md"} {
		idx := &DocumentIndex{
			DocumentID:   id,
			DocumentPath: id,
			ContentHash:  "hash",
			IndexedAt:    time.Now(),
		}
		if err := store.SaveDocumentIndex(idx); err != nil {
			t.Fatalf("save %s: %v", id, err)
		}
	}

	ids, err := store.ListDocumentIndexes()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 indexes, got %d", len(ids))
	}

	found := map[string]bool{}
	for _, id := range ids {
		found[id] = true
	}
	if !found["work/design/a.md"] || !found["work/design/b.md"] {
		t.Errorf("listed IDs = %v, want work/design/a.md and work/design/b.md", ids)
	}
}

func TestIndexStore_ListDocumentIndexes_Empty(t *testing.T) {
	store := NewIndexStore(t.TempDir())

	ids, err := store.ListDocumentIndexes()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("expected 0 indexes, got %d", len(ids))
	}
}

func TestIndexStore_GraphRoundTrip(t *testing.T) {
	store := NewIndexStore(t.TempDir())

	now := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
	original := &DocumentGraph{
		UpdatedAt: now,
		Edges: []GraphEdge{
			{From: "doc-a", FromType: "document", To: "doc-a#1", ToType: "section", EdgeType: "CONTAINS"},
			{From: "doc-a#1", FromType: "section", To: "FEAT-001", ToType: "entity_ref", EdgeType: "REFERENCES"},
		},
	}

	if err := store.SaveGraph(original); err != nil {
		t.Fatalf("save graph: %v", err)
	}

	loaded, err := store.LoadGraph()
	if err != nil {
		t.Fatalf("load graph: %v", err)
	}

	if len(loaded.Edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(loaded.Edges))
	}
	if loaded.Edges[0].From != "doc-a" {
		t.Errorf("edge[0] From = %q, want %q", loaded.Edges[0].From, "doc-a")
	}
	if loaded.Edges[0].EdgeType != "CONTAINS" {
		t.Errorf("edge[0] type = %q, want %q", loaded.Edges[0].EdgeType, "CONTAINS")
	}
	if loaded.Edges[1].EdgeType != "REFERENCES" {
		t.Errorf("edge[1] type = %q, want %q", loaded.Edges[1].EdgeType, "REFERENCES")
	}
}

func TestIndexStore_ConceptRegistryRoundTrip(t *testing.T) {
	store := NewIndexStore(t.TempDir())

	now := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
	original := &ConceptRegistry{
		UpdatedAt: now,
		Concepts: []Concept{
			{
				Name:         "lifecycle-states",
				IntroducedIn: []string{"doc-a#1"},
				UsedIn:       []string{"doc-b#2"},
			},
			{
				Name:         "entity-model",
				Aliases:      []string{"data model"},
				IntroducedIn: []string{"doc-c#1"},
			},
		},
	}

	if err := store.SaveConceptRegistry(original); err != nil {
		t.Fatalf("save registry: %v", err)
	}

	loaded, err := store.LoadConceptRegistry()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}

	if len(loaded.Concepts) != 2 {
		t.Fatalf("expected 2 concepts, got %d", len(loaded.Concepts))
	}
	c := loaded.Concepts[0]
	if c.Name != "lifecycle-states" {
		t.Errorf("concept[0] name = %q, want %q", c.Name, "lifecycle-states")
	}
	if len(c.IntroducedIn) != 1 || c.IntroducedIn[0] != "doc-a#1" {
		t.Errorf("concept[0] introduced_in = %v, want [doc-a#1]", c.IntroducedIn)
	}
	if len(c.UsedIn) != 1 || c.UsedIn[0] != "doc-b#2" {
		t.Errorf("concept[0] used_in = %v, want [doc-b#2]", c.UsedIn)
	}
	if loaded.Concepts[1].Name != "entity-model" {
		t.Errorf("concept[1] name = %q, want %q", loaded.Concepts[1].Name, "entity-model")
	}
	if len(loaded.Concepts[1].Aliases) != 1 || loaded.Concepts[1].Aliases[0] != "data model" {
		t.Errorf("concept[1] aliases = %v, want [data model]", loaded.Concepts[1].Aliases)
	}
}

func TestIndexStore_LoadMissing(t *testing.T) {
	store := NewIndexStore(t.TempDir())

	// LoadGraph should return empty graph for non-existent file
	graph, err := store.LoadGraph()
	if err != nil {
		t.Fatalf("LoadGraph error: %v", err)
	}
	if len(graph.Edges) != 0 {
		t.Errorf("expected empty edges, got %d", len(graph.Edges))
	}

	// LoadConceptRegistry should return empty registry for non-existent file
	registry, err := store.LoadConceptRegistry()
	if err != nil {
		t.Fatalf("LoadConceptRegistry error: %v", err)
	}
	if len(registry.Concepts) != 0 {
		t.Errorf("expected empty concepts, got %d", len(registry.Concepts))
	}

	// LoadDocumentIndex should return error for non-existent file
	_, err = store.LoadDocumentIndex("nonexistent")
	if err == nil {
		t.Error("expected error loading nonexistent index, got nil")
	}
}

func TestIndexStore_DocumentIndexExists(t *testing.T) {
	store := NewIndexStore(t.TempDir())

	if store.DocumentIndexExists("work/design/test.md") {
		t.Error("expected DocumentIndexExists to return false before save")
	}

	idx := &DocumentIndex{
		DocumentID:   "work/design/test.md",
		DocumentPath: "work/design/test.md",
		ContentHash:  "hash",
		IndexedAt:    time.Now(),
	}
	if err := store.SaveDocumentIndex(idx); err != nil {
		t.Fatalf("save: %v", err)
	}

	if !store.DocumentIndexExists("work/design/test.md") {
		t.Error("expected DocumentIndexExists to return true after save")
	}

	if store.DocumentIndexExists("work/design/other.md") {
		t.Error("expected DocumentIndexExists to return false for different doc")
	}
}

func TestIndexStore_OverwriteDocumentIndex(t *testing.T) {
	store := NewIndexStore(t.TempDir())

	original := &DocumentIndex{
		DocumentID:   "work/design/test.md",
		DocumentPath: "work/design/test.md",
		ContentHash:  "hash-v1",
		IndexedAt:    time.Now(),
		Sections: []Section{
			{Path: "1", Level: 1, Title: "Original"},
		},
	}
	if err := store.SaveDocumentIndex(original); err != nil {
		t.Fatalf("save original: %v", err)
	}

	updated := &DocumentIndex{
		DocumentID:   "work/design/test.md",
		DocumentPath: "work/design/test.md",
		ContentHash:  "hash-v2",
		IndexedAt:    time.Now(),
		Sections: []Section{
			{Path: "1", Level: 1, Title: "Updated"},
			{Path: "2", Level: 1, Title: "New Section"},
		},
	}
	if err := store.SaveDocumentIndex(updated); err != nil {
		t.Fatalf("save updated: %v", err)
	}

	loaded, err := store.LoadDocumentIndex("work/design/test.md")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.ContentHash != "hash-v2" {
		t.Errorf("ContentHash = %q, want %q", loaded.ContentHash, "hash-v2")
	}
	if len(loaded.Sections) != 2 {
		t.Fatalf("expected 2 sections after overwrite, got %d", len(loaded.Sections))
	}
	if loaded.Sections[0].Title != "Updated" {
		t.Errorf("section[0] title = %q, want %q", loaded.Sections[0].Title, "Updated")
	}
}
