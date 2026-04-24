package mcp

import (
	"testing"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

// extractConceptsSuggested pulls the concepts_suggested array from a guide response.
func extractConceptsSuggested(t *testing.T, out map[string]any) []map[string]any {
	t.Helper()
	raw, ok := out["concepts_suggested"]
	if !ok {
		t.Fatal("guide response missing concepts_suggested field")
	}
	slice, ok := raw.([]interface{})
	if !ok {
		t.Fatalf("concepts_suggested is not an array: %T", raw)
	}
	result := make([]map[string]any, 0, len(slice))
	for _, item := range slice {
		m, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("concepts_suggested entry is not an object: %T", item)
		}
		result = append(result, m)
	}
	return result
}

// findConceptsByPath returns the concepts_suggested entry for a given section_path.
func findConceptsByPath(cs []map[string]any, path string) map[string]any {
	for _, entry := range cs {
		if entry["section_path"] == path {
			return entry
		}
	}
	return nil
}

// suggestedConceptsFor extracts the suggested_concepts string slice from a ConceptSuggestion entry.
func suggestedConceptsFor(t *testing.T, entry map[string]any) []string {
	t.Helper()
	raw, ok := entry["suggested_concepts"]
	if !ok {
		t.Fatal("entry missing suggested_concepts field")
	}
	slice, ok := raw.([]interface{})
	if !ok {
		t.Fatalf("suggested_concepts is not an array: %T", raw)
	}
	result := make([]string, 0, len(slice))
	for _, item := range slice {
		s, ok := item.(string)
		if !ok {
			t.Fatalf("suggested_concepts item is not a string: %T", item)
		}
		result = append(result, s)
	}
	return result
}

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// ─── AC-101: concepts_suggested field always present ─────────────────────────

func TestDocIntelGuide_ConceptsSuggested_AlwaysPresent(t *testing.T) {
	svc := setupGuideEnv(t, "cs-present-doc", "# Introduction\n\nContent.\n")
	out := callGuide(t, svc, "cs-present-doc")

	if _, ok := out["concepts_suggested"]; !ok {
		t.Fatal("guide response missing concepts_suggested field")
	}
	raw := out["concepts_suggested"]
	if raw == nil {
		t.Fatal("concepts_suggested is nil, want array")
	}
	if _, ok := raw.([]interface{}); !ok {
		t.Fatalf("concepts_suggested is not an array: %T", raw)
	}
}

// ─── AC-102: one entry per section, non-empty suggested_concepts ──────────────

func TestDocIntelGuide_ConceptsSuggested_OneEntryPerSection(t *testing.T) {
	markdown := "# Doc\n\n## Risk Assessment\n\nContent.\n"
	svc := setupGuideEnv(t, "cs-one-entry", markdown)
	out := callGuide(t, svc, "cs-one-entry")

	cs := extractConceptsSuggested(t, out)

	// Count entries for the "Risk Assessment" section
	count := 0
	var foundEntry map[string]any
	for _, entry := range cs {
		if entry["section_title"] == "Risk Assessment" {
			count++
			foundEntry = entry
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 concepts_suggested entry for 'Risk Assessment', got %d", count)
	}
	if foundEntry == nil {
		t.Fatal("no entry found for 'Risk Assessment'")
	}
	concepts := suggestedConceptsFor(t, foundEntry)
	if len(concepts) == 0 {
		t.Error("suggested_concepts for 'Risk Assessment' is empty, want non-empty")
	}
}

// ─── AC-103: ancestor titles contribute tokens ────────────────────────────────

func TestDocIntelGuide_ConceptsSuggested_AncestorTitles(t *testing.T) {
	markdown := "# Doc\n\n## Overview\n\nParent content.\n\n### Design / Architecture\n\nChild content.\n"
	svc := setupGuideEnv(t, "cs-ancestor", markdown)
	out := callGuide(t, svc, "cs-ancestor")

	cs := extractConceptsSuggested(t, out)

	// Find the child section entry
	var childEntry map[string]any
	for _, entry := range cs {
		if entry["section_title"] == "Design / Architecture" {
			childEntry = entry
			break
		}
	}
	if childEntry == nil {
		t.Fatal("no concepts_suggested entry for 'Design / Architecture'")
	}
	concepts := suggestedConceptsFor(t, childEntry)

	// Must contain tokens from ancestor "Overview" and own title "Design / Architecture"
	for _, want := range []string{"Overview", "Design", "Architecture"} {
		if !containsString(concepts, want) {
			t.Errorf("expected suggested_concepts to contain %q, got %v", want, concepts)
		}
	}

	// No duplicates
	seen := make(map[string]bool)
	for _, c := range concepts {
		if seen[c] {
			t.Errorf("duplicate token %q in suggested_concepts %v", c, concepts)
		}
		seen[c] = true
	}
}

// ─── AC-104: all-stop-word section omitted ────────────────────────────────────

func TestDocIntelGuide_ConceptsSuggested_AllStopWords_Omitted(t *testing.T) {
	// "In and Of" normalises entirely to stop words; the section must be absent.
	markdown := "# Doc\n\n## In and Of\n\nContent.\n"
	svc := setupGuideEnv(t, "cs-stopwords", markdown)
	out := callGuide(t, svc, "cs-stopwords")

	cs := extractConceptsSuggested(t, out)

	// The "In and Of" section must be absent from concepts_suggested.
	for _, entry := range cs {
		if entry["section_title"] == "In and Of" {
			t.Error("expected 'In and Of' to be absent from concepts_suggested (all tokens are stop words), but it was present")
		}
	}

	// Also verify no entry has an empty suggested_concepts list.
	for _, entry := range cs {
		concepts := suggestedConceptsFor(t, entry)
		if len(concepts) == 0 {
			t.Errorf("concepts_suggested entry %v has empty suggested_concepts, should be omitted", entry)
		}
	}
}

// ─── AC-105: "Problem and Motivation" → rationale ────────────────────────────

func TestDocIntelGuide_SuggestedClassifications_ProblemAndMotivation(t *testing.T) {
	markdown := "# Spec\n\n## Problem and Motivation\n\nWhy we are doing this.\n"
	svc := setupGuideEnv(t, "sc-pam", markdown)
	out := callGuide(t, svc, "sc-pam")

	sc := extractSuggestedClassifications(t, out)
	entry := findSuggestionByTitle(sc, "Problem and Motivation")
	if entry == nil {
		t.Fatal("no suggested_classifications entry for 'Problem and Motivation'")
	}
	if entry["role"] != "rationale" {
		t.Errorf("role = %q, want rationale", entry["role"])
	}
	if entry["confidence"] != "high" {
		t.Errorf("confidence = %q, want high", entry["confidence"])
	}
}

// ─── AC-106: "Decisions" → decision ──────────────────────────────────────────

func TestDocIntelGuide_SuggestedClassifications_Decisions(t *testing.T) {
	markdown := "# Design\n\n## Decisions\n\nWhat we decided.\n"
	svc := setupGuideEnv(t, "sc-decisions", markdown)
	out := callGuide(t, svc, "sc-decisions")

	sc := extractSuggestedClassifications(t, out)
	entry := findSuggestionByTitle(sc, "Decisions")
	if entry == nil {
		t.Fatal("no suggested_classifications entry for 'Decisions'")
	}
	if entry["role"] != "decision" {
		t.Errorf("role = %q, want decision", entry["role"])
	}
	if entry["confidence"] != "high" {
		t.Errorf("confidence = %q, want high", entry["confidence"])
	}
}

// ─── AC-107: "Design" → decision ─────────────────────────────────────────────

func TestDocIntelGuide_SuggestedClassifications_Design(t *testing.T) {
	markdown := "# Document\n\n## Design\n\nHow it works.\n"
	svc := setupGuideEnv(t, "sc-design", markdown)
	out := callGuide(t, svc, "sc-design")

	sc := extractSuggestedClassifications(t, out)
	entry := findSuggestionByTitle(sc, "Design")
	if entry == nil {
		t.Fatal("no suggested_classifications entry for 'Design'")
	}
	if entry["role"] != "decision" {
		t.Errorf("role = %q, want decision", entry["role"])
	}
	if entry["confidence"] != "high" {
		t.Errorf("confidence = %q, want high", entry["confidence"])
	}
}

// ─── AC-108: "Overview" and "Summary" → narrative ────────────────────────────

func TestDocIntelGuide_SuggestedClassifications_OverviewSummary(t *testing.T) {
	markdown := "# Doc\n\n## Overview\n\nHigh-level view.\n\n## Summary\n\nConclusion.\n"
	svc := setupGuideEnv(t, "sc-overview-summary", markdown)
	out := callGuide(t, svc, "sc-overview-summary")

	sc := extractSuggestedClassifications(t, out)
	for _, title := range []string{"Overview", "Summary"} {
		entry := findSuggestionByTitle(sc, title)
		if entry == nil {
			t.Errorf("no suggested_classifications entry for %q", title)
			continue
		}
		if entry["role"] != "narrative" {
			t.Errorf("%q: role = %q, want narrative", title, entry["role"])
		}
		if entry["confidence"] != "high" {
			t.Errorf("%q: confidence = %q, want high", title, entry["confidence"])
		}
	}
}

// ─── AC-109: "Requirements" and "Goals" → requirement ────────────────────────

func TestDocIntelGuide_SuggestedClassifications_RequirementsGoals(t *testing.T) {
	markdown := "# Spec\n\n## Requirements\n\nWhat must happen.\n\n## Goals\n\nWhat we aim for.\n"
	svc := setupGuideEnv(t, "sc-reqs-goals", markdown)
	out := callGuide(t, svc, "sc-reqs-goals")

	sc := extractSuggestedClassifications(t, out)
	for _, title := range []string{"Requirements", "Goals"} {
		entry := findSuggestionByTitle(sc, title)
		if entry == nil {
			t.Errorf("no suggested_classifications entry for %q", title)
			continue
		}
		if entry["role"] != "requirement" {
			t.Errorf("%q: role = %q, want requirement", title, entry["role"])
		}
		if entry["confidence"] != "high" {
			t.Errorf("%q: confidence = %q, want high", title, entry["confidence"])
		}
	}
}

// ─── AC-110: "Risk" and "Risks" → risk ───────────────────────────────────────

func TestDocIntelGuide_SuggestedClassifications_RiskRisks(t *testing.T) {
	markdown := "# Doc\n\n## Risk\n\nSomething bad.\n\n## Risks\n\nMore bad things.\n"
	svc := setupGuideEnv(t, "sc-risk-risks", markdown)
	out := callGuide(t, svc, "sc-risk-risks")

	sc := extractSuggestedClassifications(t, out)
	for _, title := range []string{"Risk", "Risks"} {
		entry := findSuggestionByTitle(sc, title)
		if entry == nil {
			t.Errorf("no suggested_classifications entry for %q", title)
			continue
		}
		if entry["role"] != "risk" {
			t.Errorf("%q: role = %q, want risk", title, entry["role"])
		}
		if entry["confidence"] != "high" {
			t.Errorf("%q: confidence = %q, want high", title, entry["confidence"])
		}
	}
}

// ─── AC-111: "Definition" and "Glossary" → definition ────────────────────────

func TestDocIntelGuide_SuggestedClassifications_DefinitionGlossary(t *testing.T) {
	markdown := "# Doc\n\n## Definition\n\nWhat words mean.\n\n## Glossary\n\nMore words.\n"
	svc := setupGuideEnv(t, "sc-def-glossary", markdown)
	out := callGuide(t, svc, "sc-def-glossary")

	sc := extractSuggestedClassifications(t, out)
	for _, title := range []string{"Definition", "Glossary"} {
		entry := findSuggestionByTitle(sc, title)
		if entry == nil {
			t.Errorf("no suggested_classifications entry for %q", title)
			continue
		}
		if entry["role"] != "definition" {
			t.Errorf("%q: role = %q, want definition", title, entry["role"])
		}
		if entry["confidence"] != "high" {
			t.Errorf("%q: confidence = %q, want high", title, entry["confidence"])
		}
	}
}

// ─── AC-112: P28 patterns unchanged ──────────────────────────────────────────

func TestDocIntelGuide_SuggestedClassifications_P28Regression(t *testing.T) {
	markdown := "# Spec\n\n## Acceptance Criteria\n\n- AC-1. It works.\n\n## Alternatives Considered\n\nOption A vs B.\n"
	svc := setupGuideEnv(t, "sc-p28-regression", markdown)
	out := callGuide(t, svc, "sc-p28-regression")

	sc := extractSuggestedClassifications(t, out)

	ac := findSuggestionByTitle(sc, "Acceptance Criteria")
	if ac == nil {
		t.Fatal("no suggested_classifications entry for 'Acceptance Criteria'")
	}
	if ac["role"] != "requirement" {
		t.Errorf("'Acceptance Criteria' role = %q, want requirement", ac["role"])
	}
	if ac["confidence"] != "high" {
		t.Errorf("'Acceptance Criteria' confidence = %q, want high", ac["confidence"])
	}

	alt := findSuggestionByTitle(sc, "Alternatives Considered")
	if alt == nil {
		t.Fatal("no suggested_classifications entry for 'Alternatives Considered'")
	}
	if alt["role"] != "alternative" {
		t.Errorf("'Alternatives Considered' role = %q, want alternative", alt["role"])
	}
	if alt["confidence"] != "high" {
		t.Errorf("'Alternatives Considered' confidence = %q, want high", alt["confidence"])
	}
}

// ─── AC-113: expanded entries have required fields ────────────────────────────

func TestDocIntelGuide_SuggestedClassifications_ExpandedFieldsPresent(t *testing.T) {
	markdown := "# Doc\n\n## Design\n\n## Goals\n\n## Risk\n\n## Decisions\n\n"
	svc := setupGuideEnv(t, "sc-expanded-fields", markdown)
	out := callGuide(t, svc, "sc-expanded-fields")

	sc := extractSuggestedClassifications(t, out)
	titles := map[string]bool{"Design": true, "Goals": true, "Risk": true, "Decisions": true}
	for _, entry := range sc {
		title, _ := entry["title"].(string)
		if !titles[title] {
			continue
		}
		for _, field := range []string{"section_path", "role", "confidence"} {
			if _, ok := entry[field]; !ok {
				t.Errorf("entry for %q missing field %q", title, field)
			}
		}
		if entry["confidence"] != "high" {
			t.Errorf("entry for %q: confidence = %q, want high", title, entry["confidence"])
		}
	}
}

// ─── AC-114: pre-existing fields preserved ───────────────────────────────────

func TestDocIntelGuide_AllOriginalFieldsPreserved(t *testing.T) {
	svc := setupGuideEnv(t, "cs-fields-doc", "# Doc\n\nSome text.\n")
	out := callGuide(t, svc, "cs-fields-doc")

	required := []string{
		"document_id", "document_path", "content_hash", "classified",
		"outline", "entity_refs", "extraction_hints", "taxonomy",
		"suggested_classifications", "concepts_suggested",
	}
	for _, field := range required {
		if _, ok := out[field]; !ok {
			t.Errorf("guide response missing required field %q", field)
		}
	}
}
