package docint

import "testing"

func TestUpdateConceptRegistry_NewConcept(t *testing.T) {
	registry := &ConceptRegistry{}
	classifications := []Classification{
		{
			SectionPath:   "1",
			ConceptsIntro: []string{"Lifecycle States"},
		},
	}

	UpdateConceptRegistry(registry, "doc-a", classifications)

	if len(registry.Concepts) != 1 {
		t.Fatalf("expected 1 concept, got %d", len(registry.Concepts))
	}
	c := registry.Concepts[0]
	if c.Name != "lifecycle-states" {
		t.Errorf("concept name = %q, want %q", c.Name, "lifecycle-states")
	}
	if len(c.IntroducedIn) != 1 || c.IntroducedIn[0] != "doc-a#1" {
		t.Errorf("introduced_in = %v, want [doc-a#1]", c.IntroducedIn)
	}
}

func TestUpdateConceptRegistry_ExistingConcept(t *testing.T) {
	registry := &ConceptRegistry{
		Concepts: []Concept{
			{Name: "lifecycle-states", IntroducedIn: []string{"doc-a#1"}},
		},
	}
	classifications := []Classification{
		{
			SectionPath:  "2",
			ConceptsUsed: []string{"lifecycle-states"},
		},
	}

	UpdateConceptRegistry(registry, "doc-b", classifications)

	if len(registry.Concepts) != 1 {
		t.Fatalf("expected 1 concept, got %d", len(registry.Concepts))
	}
	c := registry.Concepts[0]
	if len(c.IntroducedIn) != 1 {
		t.Errorf("introduced_in should still have 1 entry, got %d", len(c.IntroducedIn))
	}
	if len(c.UsedIn) != 1 || c.UsedIn[0] != "doc-b#2" {
		t.Errorf("used_in = %v, want [doc-b#2]", c.UsedIn)
	}
}

func TestUpdateConceptRegistry_NoDuplicateRefs(t *testing.T) {
	registry := &ConceptRegistry{}
	classifications := []Classification{
		{
			SectionPath:   "1",
			ConceptsIntro: []string{"Entity Model"},
		},
	}

	// Add the same classification twice.
	UpdateConceptRegistry(registry, "doc-a", classifications)
	UpdateConceptRegistry(registry, "doc-a", classifications)

	if len(registry.Concepts) != 1 {
		t.Fatalf("expected 1 concept, got %d", len(registry.Concepts))
	}
	c := registry.Concepts[0]
	if len(c.IntroducedIn) != 1 {
		t.Errorf("introduced_in should have 1 entry (no duplicates), got %d: %v", len(c.IntroducedIn), c.IntroducedIn)
	}
}

func TestUpdateConceptRegistry_Normalization(t *testing.T) {
	registry := &ConceptRegistry{}

	// First: introduce "Lifecycle States" (space-separated, title case).
	UpdateConceptRegistry(registry, "doc-a", []Classification{
		{SectionPath: "1", ConceptsIntro: []string{"Lifecycle States"}},
	})

	// Second: use "lifecycle-states" (already normalized form).
	UpdateConceptRegistry(registry, "doc-b", []Classification{
		{SectionPath: "3", ConceptsUsed: []string{"lifecycle-states"}},
	})

	if len(registry.Concepts) != 1 {
		t.Fatalf("expected 1 concept after normalization merge, got %d", len(registry.Concepts))
	}
	c := registry.Concepts[0]
	if len(c.IntroducedIn) != 1 {
		t.Errorf("introduced_in count = %d, want 1", len(c.IntroducedIn))
	}
	if len(c.UsedIn) != 1 {
		t.Errorf("used_in count = %d, want 1", len(c.UsedIn))
	}
}

func TestRemoveDocumentFromRegistry(t *testing.T) {
	registry := &ConceptRegistry{
		Concepts: []Concept{
			{
				Name:         "lifecycle-states",
				IntroducedIn: []string{"doc-a#1"},
				UsedIn:       []string{"doc-a#2", "doc-b#1"},
			},
			{
				Name:         "entity-model",
				IntroducedIn: []string{"doc-a#3"},
			},
		},
	}

	RemoveDocumentFromRegistry(registry, "doc-a")

	// "entity-model" should be pruned entirely (only had doc-a references).
	// "lifecycle-states" should remain with only doc-b#1 in used_in.
	if len(registry.Concepts) != 1 {
		t.Fatalf("expected 1 concept after removal, got %d", len(registry.Concepts))
	}
	c := registry.Concepts[0]
	if c.Name != "lifecycle-states" {
		t.Errorf("remaining concept name = %q, want %q", c.Name, "lifecycle-states")
	}
	if len(c.IntroducedIn) != 0 {
		t.Errorf("introduced_in should be empty, got %v", c.IntroducedIn)
	}
	if len(c.UsedIn) != 1 || c.UsedIn[0] != "doc-b#1" {
		t.Errorf("used_in = %v, want [doc-b#1]", c.UsedIn)
	}
}

func TestRemoveDocumentFromRegistry_NoFalsePositive(t *testing.T) {
	// Ensure "doc-a" removal doesn't affect "doc-ab" references.
	registry := &ConceptRegistry{
		Concepts: []Concept{
			{
				Name:         "some-concept",
				IntroducedIn: []string{"doc-a#1", "doc-ab#1"},
			},
		},
	}

	RemoveDocumentFromRegistry(registry, "doc-a")

	if len(registry.Concepts) != 1 {
		t.Fatalf("expected 1 concept, got %d", len(registry.Concepts))
	}
	c := registry.Concepts[0]
	if len(c.IntroducedIn) != 1 || c.IntroducedIn[0] != "doc-ab#1" {
		t.Errorf("introduced_in = %v, want [doc-ab#1]", c.IntroducedIn)
	}
}

func TestNormalizeConcept(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Lifecycle States", "lifecycle-states"},
		{"lifecycle-states", "lifecycle-states"},
		{"  Entity Model  ", "entity-model"},
		{"UPPER_CASE", "upper-case"},
		{"mixed-Case_Name", "mixed-case-name"},
		{"already-normal", "already-normal"},
		{"  spaces  everywhere  ", "spaces-everywhere"},
	}

	for _, tt := range tests {
		got := NormalizeConcept(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeConcept(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFindConcept(t *testing.T) {
	registry := &ConceptRegistry{
		Concepts: []Concept{
			{Name: "lifecycle-states"},
			{Name: "entity-model"},
		},
	}

	// Exact match.
	c := FindConcept(registry, "lifecycle-states")
	if c == nil {
		t.Fatal("expected to find lifecycle-states")
	}
	if c.Name != "lifecycle-states" {
		t.Errorf("name = %q, want %q", c.Name, "lifecycle-states")
	}

	// Case-insensitive match via normalization.
	c = FindConcept(registry, "Entity Model")
	if c == nil {
		t.Fatal("expected to find entity-model via 'Entity Model'")
	}
	if c.Name != "entity-model" {
		t.Errorf("name = %q, want %q", c.Name, "entity-model")
	}

	// Not found.
	c = FindConcept(registry, "nonexistent")
	if c != nil {
		t.Errorf("expected nil for nonexistent concept, got %v", c)
	}
}

func TestFindConcept_EmptyRegistry(t *testing.T) {
	registry := &ConceptRegistry{}
	c := FindConcept(registry, "anything")
	if c != nil {
		t.Errorf("expected nil for empty registry, got %v", c)
	}
}

func TestFindConcept_MutatesInPlace(t *testing.T) {
	registry := &ConceptRegistry{
		Concepts: []Concept{
			{Name: "lifecycle-states"},
		},
	}

	c := FindConcept(registry, "lifecycle-states")
	if c == nil {
		t.Fatal("expected to find concept")
	}
	c.UsedIn = append(c.UsedIn, "doc-x#5")

	// The mutation should be visible through the registry.
	if len(registry.Concepts[0].UsedIn) != 1 {
		t.Errorf("expected mutation to be visible in registry, got used_in = %v", registry.Concepts[0].UsedIn)
	}
}
