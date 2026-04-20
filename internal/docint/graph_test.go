package docint

import "testing"

// ---------------------------------------------------------------------------
// Task 4 — BuildGraphEdges with extended ConceptsIntro
// ---------------------------------------------------------------------------

func TestBuildGraphEdges_ExtendedConceptsIntro(t *testing.T) {
	index := &DocumentIndex{
		DocumentID: "doc1",
		Classifications: []Classification{
			{
				SectionPath: "1",
				ConceptsIntro: []ConceptIntroEntry{
					{Name: "plain-concept"},
					{Name: "rich-concept", Aliases: []string{"rc", "rich"}},
				},
			},
		},
	}

	edges := BuildGraphEdges(index)

	// Expect two INTRODUCES edges — one per concepts_intro entry.
	var introduces []GraphEdge
	for _, e := range edges {
		if e.EdgeType == "INTRODUCES" {
			introduces = append(introduces, e)
		}
	}

	if len(introduces) != 2 {
		t.Fatalf("expected 2 INTRODUCES edges, got %d: %v", len(introduces), introduces)
	}

	wantTargets := map[string]bool{
		"plain-concept": false,
		"rich-concept":  false,
	}
	for _, e := range introduces {
		if _, ok := wantTargets[e.To]; !ok {
			t.Errorf("unexpected INTRODUCES target %q", e.To)
		}
		wantTargets[e.To] = true
		if e.From != "doc1#1" {
			t.Errorf("From = %q, want %q", e.From, "doc1#1")
		}
		if e.FromType != "fragment" {
			t.Errorf("FromType = %q, want %q", e.FromType, "fragment")
		}
		if e.ToType != "concept" {
			t.Errorf("ToType = %q, want %q", e.ToType, "concept")
		}
	}
	for name, found := range wantTargets {
		if !found {
			t.Errorf("INTRODUCES edge to %q not found", name)
		}
	}
}
