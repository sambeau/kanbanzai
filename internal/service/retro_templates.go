// Package service retro_templates.go — auto-generated document templates for
// retro fix features. RenderRetroDesign and RenderRetroSpec produce complete
// markdown documents from RetroTheme data per spec REQ-008, REQ-009, and REQ-010.
//
// Component 1a (design): Overview, Goals and Non-Goals, Design, Alternatives
// Considered, Dependencies.
//
// Component 2 (spec): Overview, Scope, Functional Requirements, Non-Functional
// Requirements, Acceptance Criteria.
package service

import (
	"fmt"
	"strings"
)

// RenderRetroDesign produces a complete design markdown document for a retro fix
// feature. The document is derived from the theme data and includes all required
// Component 1a sections.
func RenderRetroDesign(theme RetroTheme, featureID string) string {
	var b strings.Builder

	// Title and metadata header
	fmt.Fprintf(&b, "# Design: %s\n\n", theme.Title)
	fmt.Fprintf(&b, "| Field  | Value                          |\n")
	fmt.Fprintf(&b, "|--------|--------------------------------|\n")
	fmt.Fprintf(&b, "| Feature | %s |\n", featureID)
	fmt.Fprintf(&b, "| Source | Retro theme #%d (%s) |\n", theme.Rank, theme.Category)
	fmt.Fprintf(&b, "| Signal Count | %d |\n", theme.SignalCount)
	fmt.Fprintf(&b, "| Severity Score | %d |\n\n", theme.SeverityScore)

	// Overview
	b.WriteString("## Overview\n\n")
	b.WriteString(theme.RepresentativeObservation)
	b.WriteString("\n\n")

	// Goals and Non-Goals
	b.WriteString("## Goals and Non-Goals\n\n")
	b.WriteString("### Goals\n\n")
	fmt.Fprintf(&b, "- Address the %q theme identified in the retrospective\n", theme.Title)
	if theme.TopSuggestion != "" {
		fmt.Fprintf(&b, "- %s\n", formatGoalFromSuggestion(theme.TopSuggestion))
	} else {
		fmt.Fprintf(&b, "- Resolve the signals associated with this theme\n")
	}
	b.WriteString("\n### Non-Goals\n\n")
	fmt.Fprintf(&b, "- Changing unrelated systems or workflows\n")
	fmt.Fprintf(&b, "- Modifying the retrospective synthesis engine itself\n\n")

	// Design
	b.WriteString("## Design\n\n")
	fmt.Fprintf(&b, "This fix addresses a retro theme ranked #%d in the %q category. ", theme.Rank, theme.Category)
	fmt.Fprintf(&b, "The theme aggregated %d signal(s) with a severity score of %d.\n\n", theme.SignalCount, theme.SeverityScore)
	fmt.Fprintf(&b, "**Key observation:** %s\n\n", theme.RepresentativeObservation)
	if theme.TopSuggestion != "" {
		fmt.Fprintf(&b, "**Proposed approach:** %s\n\n", theme.TopSuggestion)
	} else {
		fmt.Fprintf(&b, "**Proposed approach:** Investigate and resolve the underlying issue based on signal analysis.\n\n")
	}
	if len(theme.Signals) > 0 {
		fmt.Fprintf(&b, "**Source signals:** %s\n\n", strings.Join(theme.Signals, ", "))
	}

	// Alternatives Considered
	b.WriteString("## Alternatives Considered\n\n")
	fmt.Fprintf(&b, "- **Do nothing:** Accept the workflow friction as-is. Rejected because the theme has ")
	fmt.Fprintf(&b, "%d signal(s) with severity score %d, indicating meaningful impact.\n", theme.SignalCount, theme.SeverityScore)
	fmt.Fprintf(&b, "- **Manual process change:** Document a new convention without code changes. ")
	fmt.Fprintf(&b, "Rejected because automation provides stronger guarantees and reduces cognitive load.\n\n")

	// Dependencies
	b.WriteString("## Dependencies\n\n")
	fmt.Fprintf(&b, "- The `retro_fix` tier configuration in the fast-track config\n")
	fmt.Fprintf(&b, "- The signal entries referenced by this theme: %s\n", strings.Join(theme.Signals, ", "))
	fmt.Fprintf(&b, "- No external services or new infrastructure required\n")

	return b.String()
}

// RenderRetroSpec produces a complete specification markdown document for a retro
// fix feature. The document is derived from the theme data and includes all required
// Component 2 sections. Per REQ-010, the Overview is drawn from the theme title and
// representative observation, Functional Requirements from the top suggestion, and
// Acceptance Criteria from the suggestion expressed as given/when/then.
func RenderRetroSpec(theme RetroTheme, featureID string) string {
	var b strings.Builder

	// Title and metadata header
	fmt.Fprintf(&b, "# Specification: %s\n\n", theme.Title)
	fmt.Fprintf(&b, "| Field  | Value                          |\n")
	fmt.Fprintf(&b, "|--------|--------------------------------|\n")
	fmt.Fprintf(&b, "| Feature | %s |\n", featureID)
	fmt.Fprintf(&b, "| Source | Retro theme #%d (%s) |\n", theme.Rank, theme.Category)
	fmt.Fprintf(&b, "| Signal Count | %d |\n", theme.SignalCount)
	fmt.Fprintf(&b, "| Severity Score | %d |\n\n", theme.SeverityScore)

	// Overview — derived from theme title and representative observation (REQ-010)
	b.WriteString("## Overview\n\n")
	fmt.Fprintf(&b, "This specification addresses the %q retrospective theme. ", theme.Title)
	b.WriteString(theme.RepresentativeObservation)
	b.WriteString("\n\n")

	// Scope
	b.WriteString("## Scope\n\n")
	fmt.Fprintf(&b, "**In scope:** Changes required to address the %q theme and its %d underlying signal(s).\n\n", theme.Title, theme.SignalCount)
	fmt.Fprintf(&b, "**Out of scope:** Unrelated workflow changes, retro synthesis engine modifications, ")
	fmt.Fprintf(&b, "and changes to systems not implicated by the source signals.\n\n")

	// Functional Requirements — derived from top suggestion (REQ-010)
	b.WriteString("## Functional Requirements\n\n")
	reqs := buildFunctionalRequirements(theme)
	for i, req := range reqs {
		fmt.Fprintf(&b, "- **FR-%d:** %s\n", i+1, req)
	}
	b.WriteString("\n")

	// Non-Functional Requirements
	b.WriteString("## Non-Functional Requirements\n\n")
	fmt.Fprintf(&b, "- **NFR-1:** The fix must not regress existing tests.\n")
	fmt.Fprintf(&b, "- **NFR-2:** The fix must not introduce new security vulnerabilities.\n")
	fmt.Fprintf(&b, "- **NFR-3:** The fix must follow existing code conventions and patterns.\n\n")

	// Acceptance Criteria — derived from suggestion as given/when/then (REQ-010)
	b.WriteString("## Acceptance Criteria\n\n")
	acItems := buildAcceptanceCriteria(theme)
	for i, ac := range acItems {
		fmt.Fprintf(&b, "- **AC-%d:** %s\n", i+1, ac)
	}

	return b.String()
}

// formatGoalFromSuggestion converts a suggestion into a goal statement.
func formatGoalFromSuggestion(suggestion string) string {
	s := strings.TrimSpace(suggestion)
	// Remove trailing punctuation for cleaner integration
	s = strings.TrimRight(s, ".")
	if s == "" {
		return "Resolve the identified issue"
	}
	return s
}

// buildFunctionalRequirements derives functional requirements from the theme's top
// suggestion. Per REQ-010, the suggestion is converted to requirement language.
func buildFunctionalRequirements(theme RetroTheme) []string {
	if theme.TopSuggestion == "" {
		return []string{
			fmt.Sprintf("Address the %q theme identified by %d retrospective signal(s).", theme.Title, theme.SignalCount),
		}
	}

	suggestion := strings.TrimSpace(theme.TopSuggestion)
	suggestion = strings.TrimRight(suggestion, ".")

	// Convert suggestion to requirement language by adding MUST
	return []string{
		fmt.Sprintf("The system MUST %s.", suggestion),
	}
}

// buildAcceptanceCriteria builds given/when/then acceptance criteria from the
// theme's suggestion. Per REQ-010, the suggestion is expressed as given/when/then.
func buildAcceptanceCriteria(theme RetroTheme) []string {
	suggestion := ""
	if theme.TopSuggestion != "" {
		suggestion = strings.TrimSpace(theme.TopSuggestion)
		suggestion = strings.TrimRight(suggestion, ".")
	}

	base := []string{
		fmt.Sprintf(
			"**Given** the %q retrospective theme with %d signal(s), "+
				"**when** the fix is implemented, "+
				"**then** no new signals are generated for this theme category.",
			theme.Title, theme.SignalCount,
		),
	}

	if suggestion != "" {
		base = append(base, fmt.Sprintf(
			"**Given** the current workflow behaviour, "+
				"**when** the fix is applied, "+
				"**then** %s.",
			suggestion,
		))
	}

	return base
}
