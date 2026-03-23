package docint

import (
	"testing"
	"time"
)

func graphTestIndex() *DocumentIndex {
	return &DocumentIndex{
		DocumentID:   "work/design/test.md",
		DocumentPath: "work/design/test.md",
		ContentHash:  "abc123",
		IndexedAt:    time.Now(),
		Sections: []Section{
			{
				Path:  "1",
				Level: 1,
				Title: "Overview",
				Children: []Section{
					{Path: "1.1", Level: 2, Title: "Background"},
				},
			},
			{
				Path:  "2",
				Level: 1,
				Title: "Requirements",
			},
		},
		EntityRefs: []EntityRef{
			{EntityID: "FEAT-001", EntityType: "feature", SectionPath: "1"},
			{EntityID: "TASK-002", EntityType: "task", SectionPath: "1.1"},
		},
		CrossDocLinks: []CrossDocLink{
			{TargetPath: "work/spec/phase-1.md", LinkText: "Phase 1 Spec", SectionPath: "2"},
		},
		Classifications: []Classification{
			{
				SectionPath:   "1",
				Role:          "narrative",
				Confidence:    "high",
				ConceptsIntro: []string{"Lifecycle States"},
			},
			{
				SectionPath:  "1.1",
				Role:         "rationale",
				Confidence:   "medium",
				ConceptsUsed: []string{"Lifecycle States", "Entity Model"},
			},
		},
	}
}

func TestBuildGraphEdges_ContainsEdges(t *testing.T) {
	index := graphTestIndex()
	edges := BuildGraphEdges(index)

	containsEdges := filterEdges(edges, "CONTAINS")
	if len(containsEdges) != 3 {
		t.Fatalf("expected 3 CONTAINS edges, got %d", len(containsEdges))
	}

	// doc → 1
	assertEdge(t, containsEdges[0], "work/design/test.md", "document", "work/design/test.md#1", "section")
	// 1 → 1.1
	assertEdge(t, containsEdges[1], "work/design/test.md#1", "section", "work/design/test.md#1.1", "section")
	// doc → 2
	assertEdge(t, containsEdges[2], "work/design/test.md", "document", "work/design/test.md#2", "section")
}

func TestBuildGraphEdges_ReferencesEdges(t *testing.T) {
	index := graphTestIndex()
	edges := BuildGraphEdges(index)

	refEdges := filterEdges(edges, "REFERENCES")
	if len(refEdges) != 2 {
		t.Fatalf("expected 2 REFERENCES edges, got %d", len(refEdges))
	}

	assertEdge(t, refEdges[0], "work/design/test.md#1", "section", "FEAT-001", "entity_ref")
	assertEdge(t, refEdges[1], "work/design/test.md#1.1", "section", "TASK-002", "entity_ref")
}

func TestBuildGraphEdges_LinksToEdges(t *testing.T) {
	index := graphTestIndex()
	edges := BuildGraphEdges(index)

	linkEdges := filterEdges(edges, "LINKS_TO")
	if len(linkEdges) != 1 {
		t.Fatalf("expected 1 LINKS_TO edge, got %d", len(linkEdges))
	}

	assertEdge(t, linkEdges[0], "work/design/test.md#2", "section", "work/spec/phase-1.md", "document")
}

func TestBuildGraphEdges_ConceptEdges(t *testing.T) {
	index := graphTestIndex()
	edges := BuildGraphEdges(index)

	introEdges := filterEdges(edges, "INTRODUCES")
	if len(introEdges) != 1 {
		t.Fatalf("expected 1 INTRODUCES edge, got %d", len(introEdges))
	}
	assertEdge(t, introEdges[0], "work/design/test.md#1", "fragment", "lifecycle-states", "concept")

	usesEdges := filterEdges(edges, "USES")
	if len(usesEdges) != 2 {
		t.Fatalf("expected 2 USES edges, got %d", len(usesEdges))
	}
	assertEdge(t, usesEdges[0], "work/design/test.md#1.1", "fragment", "lifecycle-states", "concept")
	assertEdge(t, usesEdges[1], "work/design/test.md#1.1", "fragment", "entity-model", "concept")
}

func TestMergeGraphEdges(t *testing.T) {
	existing := []GraphEdge{
		{From: "doc-a", FromType: "document", To: "doc-a#1", ToType: "section", EdgeType: "CONTAINS"},
		{From: "doc-b", FromType: "document", To: "doc-b#1", ToType: "section", EdgeType: "CONTAINS"},
		{From: "doc-a#1", FromType: "section", To: "FEAT-001", ToType: "entity_ref", EdgeType: "REFERENCES"},
	}

	newEdges := []GraphEdge{
		{From: "doc-a", FromType: "document", To: "doc-a#1", ToType: "section", EdgeType: "CONTAINS"},
		{From: "doc-a", FromType: "document", To: "doc-a#2", ToType: "section", EdgeType: "CONTAINS"},
	}

	merged := MergeGraphEdges(existing, "doc-a", newEdges)

	// Should have: 1 kept from doc-b + 2 new from doc-a = 3
	if len(merged) != 3 {
		t.Fatalf("expected 3 merged edges, got %d", len(merged))
	}

	// First edge should be the kept one from doc-b
	if merged[0].From != "doc-b" {
		t.Errorf("first merged edge From = %q, want %q", merged[0].From, "doc-b")
	}

	// Last two should be the new doc-a edges
	if merged[1].To != "doc-a#1" || merged[2].To != "doc-a#2" {
		t.Errorf("new edges not appended correctly: got To=%q, To=%q", merged[1].To, merged[2].To)
	}
}

func filterEdges(edges []GraphEdge, edgeType string) []GraphEdge {
	var result []GraphEdge
	for _, e := range edges {
		if e.EdgeType == edgeType {
			result = append(result, e)
		}
	}
	return result
}

func assertEdge(t *testing.T, edge GraphEdge, from, fromType, to, toType string) {
	t.Helper()
	if edge.From != from {
		t.Errorf("edge From = %q, want %q", edge.From, from)
	}
	if edge.FromType != fromType {
		t.Errorf("edge FromType = %q, want %q", edge.FromType, fromType)
	}
	if edge.To != to {
		t.Errorf("edge To = %q, want %q", edge.To, to)
	}
	if edge.ToType != toType {
		t.Errorf("edge ToType = %q, want %q", edge.ToType, toType)
	}
}
