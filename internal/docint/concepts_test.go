package docint

import (
	"testing"

	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// Task 1 — ConceptIntroEntry YAML unmarshaling
// ---------------------------------------------------------------------------

func TestConceptIntroEntry_UnmarshalYAML_PlainString(t *testing.T) {
	input := `- foo-concept`
	var entries []ConceptIntroEntry
	if err := yaml.Unmarshal([]byte(input), &entries); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Name != "foo-concept" {
		t.Errorf("Name = %q, want %q", entries[0].Name, "foo-concept")
	}
	if len(entries[0].Aliases) != 0 {
		t.Errorf("expected no aliases, got %v", entries[0].Aliases)
	}
}

func TestConceptIntroEntry_UnmarshalYAML_ObjectForm(t *testing.T) {
	input := `
- name: workflow-stage
  aliases:
    - stage
    - lifecycle-stage
`
	var entries []ConceptIntroEntry
	if err := yaml.Unmarshal([]byte(input), &entries); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Name != "workflow-stage" {
		t.Errorf("Name = %q, want %q", e.Name, "workflow-stage")
	}
	if len(e.Aliases) != 2 || e.Aliases[0] != "stage" || e.Aliases[1] != "lifecycle-stage" {
		t.Errorf("Aliases = %v, want [stage lifecycle-stage]", e.Aliases)
	}
}

func TestConceptIntroEntry_UnmarshalYAML_MixedList(t *testing.T) {
	input := `
- plain-concept
- name: rich-concept
  aliases: [alias-a, alias-b]
`
	var entries []ConceptIntroEntry
	if err := yaml.Unmarshal([]byte(input), &entries); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Name != "plain-concept" || len(entries[0].Aliases) != 0 {
		t.Errorf("entry[0] = %+v, want {Name:plain-concept Aliases:[]}", entries[0])
	}
	if entries[1].Name != "rich-concept" || len(entries[1].Aliases) != 2 {
		t.Errorf("entry[1] = %+v, want {Name:rich-concept Aliases:[alias-a alias-b]}", entries[1])
	}
}

func TestConceptIntroEntry_UnmarshalYAML_Backward(t *testing.T) {
	// Old-style plain string entries must still parse correctly after the type change.
	input := `
- alpha
- beta
- gamma
`
	var entries []ConceptIntroEntry
	if err := yaml.Unmarshal([]byte(input), &entries); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	names := []string{"alpha", "beta", "gamma"}
	for i, want := range names {
		if entries[i].Name != want {
			t.Errorf("entries[%d].Name = %q, want %q", i, entries[i].Name, want)
		}
		if len(entries[i].Aliases) != 0 {
			t.Errorf("entries[%d].Aliases = %v, want empty", i, entries[i].Aliases)
		}
	}
}

// ---------------------------------------------------------------------------
// Task 2 — UpdateConceptRegistry alias storage
// ---------------------------------------------------------------------------

func TestUpdateConceptRegistry_Aliases_Stored(t *testing.T) {
	registry := &ConceptRegistry{}
	classifications := []Classification{
		{
			SectionPath: "1",
			ConceptsIntro: []ConceptIntroEntry{
				{Name: "workflow-stage", Aliases: []string{"stage", "lifecycle-stage"}},
			},
		},
	}
	UpdateConceptRegistry(registry, "doc1", classifications)

	c := FindConcept(registry, "workflow-stage")
	if c == nil {
		t.Fatal("concept not found")
	}
	if len(c.Aliases) != 2 {
		t.Fatalf("expected 2 aliases, got %d: %v", len(c.Aliases), c.Aliases)
	}
	if !stringSliceContains(c.Aliases, "stage") {
		t.Errorf("alias 'stage' not stored; aliases = %v", c.Aliases)
	}
	if !stringSliceContains(c.Aliases, "lifecycle-stage") {
		t.Errorf("alias 'lifecycle-stage' not stored; aliases = %v", c.Aliases)
	}
}

func TestUpdateConceptRegistry_Aliases_Deduplicated(t *testing.T) {
	registry := &ConceptRegistry{}
	classifications := []Classification{
		{
			SectionPath: "1",
			ConceptsIntro: []ConceptIntroEntry{
				{Name: "concept-a", Aliases: []string{"alias-x", "alias-x", "alias-x"}},
			},
		},
	}
	UpdateConceptRegistry(registry, "doc1", classifications)

	c := FindConcept(registry, "concept-a")
	if c == nil {
		t.Fatal("concept not found")
	}
	if len(c.Aliases) != 1 {
		t.Errorf("expected 1 deduplicated alias, got %d: %v", len(c.Aliases), c.Aliases)
	}
}

func TestUpdateConceptRegistry_Aliases_Accumulated(t *testing.T) {
	registry := &ConceptRegistry{}

	// First call: introduces concept with alias-one
	UpdateConceptRegistry(registry, "doc1", []Classification{
		{
			SectionPath: "1",
			ConceptsIntro: []ConceptIntroEntry{
				{Name: "my-concept", Aliases: []string{"alias-one"}},
			},
		},
	})

	// Second call: same concept from another document, adds alias-two
	UpdateConceptRegistry(registry, "doc2", []Classification{
		{
			SectionPath: "2",
			ConceptsIntro: []ConceptIntroEntry{
				{Name: "my-concept", Aliases: []string{"alias-two"}},
			},
		},
	})

	c := FindConcept(registry, "my-concept")
	if c == nil {
		t.Fatal("concept not found")
	}
	if len(c.Aliases) != 2 {
		t.Errorf("expected 2 accumulated aliases, got %d: %v", len(c.Aliases), c.Aliases)
	}
	if !stringSliceContains(c.Aliases, "alias-one") || !stringSliceContains(c.Aliases, "alias-two") {
		t.Errorf("unexpected aliases: %v", c.Aliases)
	}
}

func TestUpdateConceptRegistry_Aliases_ExcludeCanonical(t *testing.T) {
	registry := &ConceptRegistry{}
	classifications := []Classification{
		{
			SectionPath: "1",
			ConceptsIntro: []ConceptIntroEntry{
				// "my-concept" is both the canonical name and listed as an alias — should be dropped
				{Name: "my-concept", Aliases: []string{"my-concept", "My Concept", "real-alias"}},
			},
		},
	}
	UpdateConceptRegistry(registry, "doc1", classifications)

	c := FindConcept(registry, "my-concept")
	if c == nil {
		t.Fatal("concept not found")
	}
	// "my-concept" and "My Concept" both normalise to "my-concept" — excluded
	// "real-alias" remains
	if len(c.Aliases) != 1 || c.Aliases[0] != "real-alias" {
		t.Errorf("expected [real-alias], got %v", c.Aliases)
	}
}

func TestUpdateConceptRegistry_PlainString_NoAliases(t *testing.T) {
	registry := &ConceptRegistry{}
	classifications := []Classification{
		{
			SectionPath: "1",
			ConceptsIntro: []ConceptIntroEntry{
				{Name: "simple-concept"},
			},
		},
	}
	UpdateConceptRegistry(registry, "doc1", classifications)

	c := FindConcept(registry, "simple-concept")
	if c == nil {
		t.Fatal("concept not found")
	}
	if len(c.Aliases) != 0 {
		t.Errorf("expected no aliases, got %v", c.Aliases)
	}
}

// ---------------------------------------------------------------------------
// Task 3 — FindConcept alias resolution
// ---------------------------------------------------------------------------

func TestFindConcept_AliasResolution(t *testing.T) {
	registry := &ConceptRegistry{
		Concepts: []Concept{
			{Name: "workflow-stage", Aliases: []string{"stage", "lifecycle-stage"}},
		},
	}
	c := FindConcept(registry, "lifecycle-stage")
	if c == nil {
		t.Fatal("expected concept via alias, got nil")
	}
	if c.Name != "workflow-stage" {
		t.Errorf("Name = %q, want %q", c.Name, "workflow-stage")
	}
}

func TestFindConcept_AliasResolution_CaseInsensitive(t *testing.T) {
	registry := &ConceptRegistry{
		Concepts: []Concept{
			{Name: "kanban-board", Aliases: []string{"kanban"}},
		},
	}
	c := FindConcept(registry, "KANBAN")
	if c == nil {
		t.Fatal("expected case-insensitive alias match, got nil")
	}
	if c.Name != "kanban-board" {
		t.Errorf("Name = %q, want %q", c.Name, "kanban-board")
	}
}

func TestFindConcept_CanonicalPriority(t *testing.T) {
	// "stage" is also an alias on another concept, but it IS the canonical name of this one.
	registry := &ConceptRegistry{
		Concepts: []Concept{
			{Name: "stage", Aliases: []string{}},
			{Name: "workflow-stage", Aliases: []string{"stage"}},
		},
	}
	c := FindConcept(registry, "stage")
	if c == nil {
		t.Fatal("expected a result, got nil")
	}
	// Canonical "stage" must win over the alias on "workflow-stage"
	if c.Name != "stage" {
		t.Errorf("canonical should win; Name = %q, want %q", c.Name, "stage")
	}
}

func TestFindConcept_NoMatch_ReturnsNil(t *testing.T) {
	registry := &ConceptRegistry{
		Concepts: []Concept{
			{Name: "alpha", Aliases: []string{"a"}},
		},
	}
	if c := FindConcept(registry, "beta"); c != nil {
		t.Errorf("expected nil, got %+v", c)
	}
}

func TestFindConcept_MutatesInPlace(t *testing.T) {
	registry := &ConceptRegistry{
		Concepts: []Concept{
			{Name: "my-concept", Aliases: []string{"mc"}},
		},
	}
	c := FindConcept(registry, "mc") // alias lookup
	if c == nil {
		t.Fatal("expected concept via alias, got nil")
	}
	// Mutate via the returned pointer — change should be visible in the registry.
	c.UsedIn = append(c.UsedIn, "doc1#1")
	if len(registry.Concepts[0].UsedIn) != 1 || registry.Concepts[0].UsedIn[0] != "doc1#1" {
		t.Errorf("mutation not visible through registry; UsedIn = %v", registry.Concepts[0].UsedIn)
	}
}
