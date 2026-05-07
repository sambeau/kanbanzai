package service

import (
	"strings"
	"testing"
)

// ─── RenderRetroDesign ────────────────────────────────────────────────────────

func TestRenderRetroDesign_ContainsAllRequiredSections(t *testing.T) {
	t.Parallel()

	theme := RetroTheme{
		Rank:                      1,
		Category:                  "workflow-friction",
		Title:                     "Sub-agent handoff context is incomplete",
		SignalCount:               5,
		SeverityScore:             20,
		Signals:                   []string{"KE-001", "KE-002", "KE-003"},
		TopSuggestion:             "Add required context fields to the handoff assembly pipeline",
		RepresentativeObservation: "Sub-agents frequently re-read files because handoff prompts omit spec sections and role conventions.",
	}

	output := RenderRetroDesign(theme, "FEAT-TEST001")
	o := strings.ToLower(output)

	// Required Component 1a sections
	requiredSections := []string{
		"## overview",
		"## goals and non-goals",
		"## design",
		"## alternatives considered",
		"## dependencies",
	}
	for _, section := range requiredSections {
		if !strings.Contains(o, section) {
			t.Errorf("RenderRetroDesign output missing required section %q", section)
		}
	}

	// Must contain the feature ID
	if !strings.Contains(output, "FEAT-TEST001") {
		t.Errorf("RenderRetroDesign output missing feature ID")
	}

	// Must contain theme data
	if !strings.Contains(output, theme.Title) {
		t.Errorf("RenderRetroDesign output missing theme title")
	}
	if !strings.Contains(output, theme.RepresentativeObservation) {
		t.Errorf("RenderRetroDesign output missing representative observation")
	}
	if !strings.Contains(output, theme.TopSuggestion) {
		t.Errorf("RenderRetroDesign output missing top suggestion")
	}
}

func TestRenderRetroDesign_SectionsHaveNonEmptyContent(t *testing.T) {
	t.Parallel()

	theme := RetroTheme{
		Rank:                      2,
		Category:                  "tool-gap",
		Title:                     "Missing tool for batch operations",
		SignalCount:               3,
		SeverityScore:             9,
		Signals:                   []string{"KE-010", "KE-011"},
		TopSuggestion:             "Create a batch operation endpoint",
		RepresentativeObservation: "Users report that performing individual operations is slow.",
	}

	output := RenderRetroDesign(theme, "FEAT-TEST002")

	// Each H2 section should have content after it (not immediately followed by another H2 or EOF)
	sections := []string{"## Overview", "## Goals and Non-Goals", "## Design", "## Alternatives Considered", "## Dependencies"}
	for i, sec := range sections {
		idx := strings.Index(output, sec)
		if idx < 0 {
			t.Fatalf("Section %q not found", sec)
		}
		// Content starts after the heading line
		afterHeading := idx + len(sec)
		// Find next section or end
		nextIdx := len(output)
		if i+1 < len(sections) {
			n := strings.Index(output[idx+len(sec):], sections[i+1])
			if n >= 0 {
				nextIdx = idx + len(sec) + n
			}
		}
		sectionBody := strings.TrimSpace(output[afterHeading:nextIdx])
		if sectionBody == "" {
			t.Errorf("Section %q has no content", sec)
		}
	}
}

func TestRenderRetroDesign_EmptySuggestion(t *testing.T) {
	t.Parallel()

	theme := RetroTheme{
		Rank:                      1,
		Category:                  "spec-ambiguity",
		Title:                     "Unclear requirements",
		SignalCount:               2,
		SeverityScore:             6,
		Signals:                   []string{"KE-020"},
		TopSuggestion:             "",
		RepresentativeObservation: "Specifications lack concrete examples.",
	}

	output := RenderRetroDesign(theme, "FEAT-TEST003")
	if !strings.Contains(output, "## Design") {
		t.Errorf("RenderRetroDesign with empty suggestion missing Design section")
	}
	// Must not panic — just verify output is non-empty
	if len(output) == 0 {
		t.Errorf("RenderRetroDesign returned empty string for theme with empty suggestion")
	}
}

// ─── RenderRetroSpec ──────────────────────────────────────────────────────────

func TestRenderRetroSpec_ContainsAllRequiredSections(t *testing.T) {
	t.Parallel()

	theme := RetroTheme{
		Rank:                      1,
		Category:                  "workflow-friction",
		Title:                     "Sub-agent handoff context is incomplete",
		SignalCount:               5,
		SeverityScore:             20,
		Signals:                   []string{"KE-001", "KE-002", "KE-003"},
		TopSuggestion:             "Add required context fields to the handoff assembly pipeline",
		RepresentativeObservation: "Sub-agents frequently re-read files because handoff prompts omit spec sections and role conventions.",
	}

	output := RenderRetroSpec(theme, "FEAT-TEST001")
	o := strings.ToLower(output)

	// Required Component 2 sections
	requiredSections := []string{
		"## overview",
		"## scope",
		"## functional requirements",
		"## non-functional requirements",
		"## acceptance criteria",
	}
	for _, section := range requiredSections {
		if !strings.Contains(o, section) {
			t.Errorf("RenderRetroSpec output missing required section %q", section)
		}
	}

	// Must contain the feature ID
	if !strings.Contains(output, "FEAT-TEST001") {
		t.Errorf("RenderRetroSpec output missing feature ID")
	}
}

func TestRenderRetroSpec_DerivesContentFromTheme(t *testing.T) {
	t.Parallel()

	theme := RetroTheme{
		Rank:                      3,
		Category:                  "tool-gap",
		Title:                     "No batch verification tool",
		SignalCount:               4,
		SeverityScore:             12,
		Signals:                   []string{"KE-030", "KE-031"},
		TopSuggestion:             "Expose a batch verification endpoint in the MCP server",
		RepresentativeObservation: "Verifying features one at a time is error-prone and slow.",
	}

	output := RenderRetroSpec(theme, "FEAT-TEST002")

	// REQ-010: Overview from title and representative observation
	if !strings.Contains(output, theme.Title) {
		t.Errorf("RenderRetroSpec Overview missing theme title")
	}
	if !strings.Contains(output, theme.RepresentativeObservation) {
		t.Errorf("RenderRetroSpec Overview missing representative observation")
	}

	// REQ-010: Functional Requirements from top suggestion
	if !strings.Contains(output, "batch verification") {
		t.Errorf("RenderRetroSpec Functional Requirements missing suggestion-derived content")
	}
	if !strings.Contains(output, "MUST") {
		t.Errorf("RenderRetroSpec Functional Requirements missing requirement language (MUST)")
	}

	// REQ-010: Acceptance Criteria in given/when/then format
	if !strings.Contains(output, "**Given**") {
		t.Errorf("RenderRetroSpec Acceptance Criteria missing Given clause")
	}
	if !strings.Contains(output, "**when**") {
		t.Errorf("RenderRetroSpec Acceptance Criteria missing When clause")
	}
	if !strings.Contains(output, "**then**") {
		t.Errorf("RenderRetroSpec Acceptance Criteria missing Then clause")
	}
}

func TestRenderRetroSpec_SectionsHaveNonEmptyContent(t *testing.T) {
	t.Parallel()

	theme := RetroTheme{
		Rank:                      1,
		Category:                  "context-gap",
		Title:                     "Missing error context",
		SignalCount:               2,
		SeverityScore:             4,
		Signals:                   []string{"KE-040"},
		TopSuggestion:             "Include stack trace in error responses",
		RepresentativeObservation: "Error messages are too vague for debugging.",
	}

	output := RenderRetroSpec(theme, "FEAT-TEST003")

	sections := []string{"## Overview", "## Scope", "## Functional Requirements", "## Non-Functional Requirements", "## Acceptance Criteria"}
	for i, sec := range sections {
		idx := strings.Index(output, sec)
		if idx < 0 {
			t.Fatalf("Section %q not found", sec)
		}
		afterHeading := idx + len(sec)
		nextIdx := len(output)
		if i+1 < len(sections) {
			n := strings.Index(output[idx+len(sec):], sections[i+1])
			if n >= 0 {
				nextIdx = idx + len(sec) + n
			}
		}
		sectionBody := strings.TrimSpace(output[afterHeading:nextIdx])
		if sectionBody == "" {
			t.Errorf("Section %q has no content", sec)
		}
	}
}

func TestRenderRetroSpec_EmptySuggestion(t *testing.T) {
	t.Parallel()

	theme := RetroTheme{
		Rank:                      5,
		Category:                  "decomposition-issue",
		Title:                     "Tasks are too large",
		SignalCount:               1,
		SeverityScore:             3,
		Signals:                   []string{"KE-050"},
		TopSuggestion:             "",
		RepresentativeObservation: "Decomposed tasks often exceed reasonable size.",
	}

	output := RenderRetroSpec(theme, "FEAT-TEST004")

	// Must still contain all sections
	requiredSections := []string{
		"## Overview", "## Scope", "## Functional Requirements",
		"## Non-Functional Requirements", "## Acceptance Criteria",
	}
	for _, section := range requiredSections {
		if !strings.Contains(output, section) {
			t.Errorf("RenderRetroSpec with empty suggestion missing section %q", section)
		}
	}

	// Must still have Given/When/Then in acceptance criteria
	if !strings.Contains(output, "**Given**") {
		t.Errorf("RenderRetroSpec with empty suggestion missing Given clause")
	}

	if len(output) == 0 {
		t.Errorf("RenderRetroSpec returned empty string for theme with empty suggestion")
	}
}
