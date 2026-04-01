package skill

import (
	"fmt"
	"strings"
)

// ValidationMessage is either an error or a warning from section validation.
type ValidationMessage struct {
	Level   string // "error" or "warning"
	Message string
}

// canonicalSectionOrder is the attention-curve section ordering (FR-008).
var canonicalSectionOrder = []string{
	"Vocabulary",
	"Anti-Patterns",
	"Checklist",
	"Procedure",
	"Output Format",
	"Examples",
	"Evaluation Criteria",
	"Questions This Skill Answers",
}

// requiredSections must appear in every SKILL.md (FR-009).
var requiredSections = map[string]bool{
	"Vocabulary":                   true,
	"Anti-Patterns":                true,
	"Procedure":                    true,
	"Output Format":                true,
	"Evaluation Criteria":          true,
	"Questions This Skill Answers": true,
}

// canonicalIndex returns the position of a heading in canonicalSectionOrder,
// or -1 if the heading is not a known section.
func canonicalIndex(heading string) int {
	for i, h := range canonicalSectionOrder {
		if h == heading {
			return i
		}
	}
	return -1
}

// parseSections extracts ## sections from raw Markdown body.
// Returns sections in document order. Sub-headings (###, ####) are included
// in parent section content, not treated as separate sections.
func parseSections(bodyRaw string) []BodySection {
	var sections []BodySection
	var current *BodySection

	lines := strings.Split(bodyRaw, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "## ") && !strings.HasPrefix(line, "### ") {
			// Flush previous section.
			if current != nil {
				current.Content = strings.TrimRight(current.Content, " \t\n")
				sections = append(sections, *current)
			}
			heading := strings.TrimPrefix(line, "## ")
			current = &BodySection{Heading: heading}
			continue
		}

		if current != nil {
			current.Content += line + "\n"
		}
	}

	// Flush final section.
	if current != nil {
		current.Content = strings.TrimRight(current.Content, " \t\n")
		sections = append(sections, *current)
	}

	return sections
}

// validateSections checks section presence, ordering, and content rules.
// constraintLevel is needed for the checklist conditional requirement (FR-012).
func validateSections(sections []BodySection, constraintLevel string) []ValidationMessage {
	var msgs []ValidationMessage

	present := make(map[string]bool, len(sections))
	for _, s := range sections {
		present[s.Heading] = true
	}

	// FR-009: required sections must be present.
	for heading := range requiredSections {
		if !present[heading] {
			msgs = append(msgs, ValidationMessage{
				Level:   "error",
				Message: fmt.Sprintf("missing required section %q", heading),
			})
		}
	}

	// FR-012: Checklist is required when constraintLevel is "low" or "medium".
	if (constraintLevel == "low" || constraintLevel == "medium") && !present["Checklist"] {
		msgs = append(msgs, ValidationMessage{
			Level:   "error",
			Message: fmt.Sprintf("missing required section %q (required for constraint_level %q)", "Checklist", constraintLevel),
		})
	}

	// FR-008: check relative ordering of known sections.
	var lastIdx int
	var lastName string
	first := true
	for _, s := range sections {
		idx := canonicalIndex(s.Heading)
		if idx < 0 {
			continue // unknown section, skip ordering check
		}
		if !first && idx < lastIdx {
			msgs = append(msgs, ValidationMessage{
				Level:   "error",
				Message: fmt.Sprintf("section %q appears after %q but should appear before it", s.Heading, lastName),
			})
		}
		if first || idx > lastIdx {
			lastIdx = idx
			lastName = s.Heading
			first = false
		}
	}

	// Warn on unknown section headings.
	for _, s := range sections {
		if canonicalIndex(s.Heading) < 0 {
			msgs = append(msgs, ValidationMessage{
				Level:   "warning",
				Message: fmt.Sprintf("unknown section %q", s.Heading),
			})
		}
	}

	// FR-010: Vocabulary must have non-empty content.
	for _, s := range sections {
		if s.Heading == "Vocabulary" && strings.TrimSpace(s.Content) == "" {
			msgs = append(msgs, ValidationMessage{
				Level:   "error",
				Message: "section \"Vocabulary\" must have non-empty content",
			})
		}
	}

	// FR-013: Evaluation Criteria must have non-empty content.
	for _, s := range sections {
		if s.Heading == "Evaluation Criteria" && strings.TrimSpace(s.Content) == "" {
			msgs = append(msgs, ValidationMessage{
				Level:   "error",
				Message: "section \"Evaluation Criteria\" must have non-empty content",
			})
		}
	}

	return msgs
}
