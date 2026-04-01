package skill

import (
	"strings"
	"testing"
)

func allRequiredSections() []BodySection {
	return []BodySection{
		{Heading: "Vocabulary", Content: "- **Term**: Definition"},
		{Heading: "Anti-Patterns", Content: "- Don't do X"},
		{Heading: "Procedure", Content: "1. Step one"},
		{Heading: "Output Format", Content: "Return markdown"},
		{Heading: "Evaluation Criteria", Content: "- Criterion A"},
		{Heading: "Questions This Skill Answers", Content: "- How to X?"},
	}
}

func allRequiredSectionsMarkdown() string {
	return `## Vocabulary

- **Term**: Definition

## Anti-Patterns

- Don't do X

## Procedure

1. Step one

## Output Format

Return markdown

## Evaluation Criteria

- Criterion A

## Questions This Skill Answers

- How to X?
`
}

func TestParseSections(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantLen   int
		wantFirst string
		wantLast  string
	}{
		{
			name:      "single section",
			input:     "## Vocabulary\n\n- **Term**: Definition\n",
			wantLen:   1,
			wantFirst: "Vocabulary",
			wantLast:  "Vocabulary",
		},
		{
			name:      "multiple sections",
			input:     "## Vocabulary\n\nContent A\n\n## Procedure\n\nContent B\n",
			wantLen:   2,
			wantFirst: "Vocabulary",
			wantLast:  "Procedure",
		},
		{
			name:    "empty input",
			input:   "",
			wantLen: 0,
		},
		{
			name:    "no sections - just text",
			input:   "Some preamble text\nwithout any headings\n",
			wantLen: 0,
		},
		{
			name:      "all required sections",
			input:     allRequiredSectionsMarkdown(),
			wantLen:   6,
			wantFirst: "Vocabulary",
			wantLast:  "Questions This Skill Answers",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sections := parseSections(tc.input)
			if len(sections) != tc.wantLen {
				t.Fatalf("got %d sections, want %d", len(sections), tc.wantLen)
			}
			if tc.wantLen > 0 {
				if sections[0].Heading != tc.wantFirst {
					t.Errorf("first heading = %q, want %q", sections[0].Heading, tc.wantFirst)
				}
				if sections[len(sections)-1].Heading != tc.wantLast {
					t.Errorf("last heading = %q, want %q", sections[len(sections)-1].Heading, tc.wantLast)
				}
			}
		})
	}
}

func TestParseSections_SubHeadings(t *testing.T) {
	input := "## Vocabulary\n\n- **Term**: Definition\n\n### Sub-heading\n\nSub content\n\n#### Deep heading\n\nDeep content\n\n## Procedure\n\n1. Step\n"

	sections := parseSections(input)
	if len(sections) != 2 {
		t.Fatalf("got %d sections, want 2", len(sections))
	}

	if sections[0].Heading != "Vocabulary" {
		t.Errorf("first heading = %q, want %q", sections[0].Heading, "Vocabulary")
	}
	if !strings.Contains(sections[0].Content, "### Sub-heading") {
		t.Error("### sub-heading should be included in parent section content")
	}
	if !strings.Contains(sections[0].Content, "#### Deep heading") {
		t.Error("#### deep heading should be included in parent section content")
	}
	if !strings.Contains(sections[0].Content, "Sub content") {
		t.Error("sub-heading content should be included in parent section")
	}
	if !strings.Contains(sections[0].Content, "Deep content") {
		t.Error("deep heading content should be included in parent section")
	}

	if sections[1].Heading != "Procedure" {
		t.Errorf("second heading = %q, want %q", sections[1].Heading, "Procedure")
	}
}

func TestParseSections_ContentTrimming(t *testing.T) {
	input := "## Vocabulary\n\nSome content\n\n\n\n"
	sections := parseSections(input)
	if len(sections) != 1 {
		t.Fatalf("got %d sections, want 1", len(sections))
	}
	if strings.HasSuffix(sections[0].Content, "\n") {
		t.Error("trailing whitespace should be trimmed from section content")
	}
}

func TestValidateSections(t *testing.T) {
	tests := []struct {
		name            string
		sections        []BodySection
		constraintLevel string
		wantErrors      int
		wantWarnings    int
		wantSubstr      string // substring expected in at least one message
	}{
		{
			name:            "all required sections in correct order",
			sections:        allRequiredSections(),
			constraintLevel: "high",
			wantErrors:      0,
			wantWarnings:    0,
		},
		{
			name: "missing required section - Vocabulary",
			sections: []BodySection{
				{Heading: "Anti-Patterns", Content: "content"},
				{Heading: "Procedure", Content: "content"},
				{Heading: "Output Format", Content: "content"},
				{Heading: "Evaluation Criteria", Content: "content"},
				{Heading: "Questions This Skill Answers", Content: "content"},
			},
			constraintLevel: "high",
			wantErrors:      1,
			wantWarnings:    0,
			wantSubstr:      "Vocabulary",
		},
		{
			name: "missing required section - Procedure",
			sections: []BodySection{
				{Heading: "Vocabulary", Content: "content"},
				{Heading: "Anti-Patterns", Content: "content"},
				{Heading: "Output Format", Content: "content"},
				{Heading: "Evaluation Criteria", Content: "content"},
				{Heading: "Questions This Skill Answers", Content: "content"},
			},
			constraintLevel: "high",
			wantErrors:      1,
			wantWarnings:    0,
			wantSubstr:      "Procedure",
		},
		{
			name: "out-of-order sections",
			sections: []BodySection{
				{Heading: "Procedure", Content: "content"},
				{Heading: "Vocabulary", Content: "content"},
				{Heading: "Anti-Patterns", Content: "content"},
				{Heading: "Output Format", Content: "content"},
				{Heading: "Evaluation Criteria", Content: "content"},
				{Heading: "Questions This Skill Answers", Content: "content"},
			},
			constraintLevel: "high",
			wantErrors:      2,
			wantWarnings:    0,
			wantSubstr:      "appears after",
		},
		{
			name: "unknown heading produces warning",
			sections: append(allRequiredSections(), BodySection{
				Heading: "Custom Section",
				Content: "some content",
			}),
			constraintLevel: "high",
			wantErrors:      0,
			wantWarnings:    1,
			wantSubstr:      "unknown section",
		},
		{
			name:            "checklist required for low constraint_level - missing",
			sections:        allRequiredSections(),
			constraintLevel: "low",
			wantErrors:      1,
			wantWarnings:    0,
			wantSubstr:      "Checklist",
		},
		{
			name:            "checklist required for medium constraint_level - missing",
			sections:        allRequiredSections(),
			constraintLevel: "medium",
			wantErrors:      1,
			wantWarnings:    0,
			wantSubstr:      "Checklist",
		},
		{
			name:            "checklist optional for high constraint_level - no error",
			sections:        allRequiredSections(),
			constraintLevel: "high",
			wantErrors:      0,
			wantWarnings:    0,
		},
		{
			name: "checklist present for low constraint_level - no error",
			sections: func() []BodySection {
				s := allRequiredSections()
				// Insert Checklist after Anti-Patterns (canonical position).
				result := make([]BodySection, 0, len(s)+1)
				for _, sec := range s {
					result = append(result, sec)
					if sec.Heading == "Anti-Patterns" {
						result = append(result, BodySection{Heading: "Checklist", Content: "- [ ] Check"})
					}
				}
				return result
			}(),
			constraintLevel: "low",
			wantErrors:      0,
			wantWarnings:    0,
		},
		{
			name: "empty vocabulary body",
			sections: func() []BodySection {
				s := allRequiredSections()
				s[0].Content = "   \n\t\n  "
				return s
			}(),
			constraintLevel: "high",
			wantErrors:      1,
			wantWarnings:    0,
			wantSubstr:      "Vocabulary",
		},
		{
			name: "empty evaluation criteria body",
			sections: func() []BodySection {
				s := allRequiredSections()
				for i := range s {
					if s[i].Heading == "Evaluation Criteria" {
						s[i].Content = ""
					}
				}
				return s
			}(),
			constraintLevel: "high",
			wantErrors:      1,
			wantWarnings:    0,
			wantSubstr:      "Evaluation Criteria",
		},
		{
			name: "multiple errors accumulated",
			sections: []BodySection{
				{Heading: "Procedure", Content: "content"},
				{Heading: "Vocabulary", Content: ""}, // out of order + empty
			},
			constraintLevel: "low",
			// Missing: Anti-Patterns, Output Format, Evaluation Criteria, Questions This Skill Answers = 4
			// Missing Checklist for low = 1
			// Out of order Vocabulary after Procedure = 1
			// Empty Vocabulary = 1
			// Missing Evaluation Criteria counted in required, plus empty content won't fire (section absent)
			wantErrors:   7,
			wantWarnings: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msgs := validateSections(tc.sections, tc.constraintLevel)

			var errors, warnings int
			for _, m := range msgs {
				switch m.Level {
				case "error":
					errors++
				case "warning":
					warnings++
				}
			}

			if errors != tc.wantErrors {
				t.Errorf("got %d errors, want %d", errors, tc.wantErrors)
				for _, m := range msgs {
					t.Errorf("  %s: %s", m.Level, m.Message)
				}
			}
			if warnings != tc.wantWarnings {
				t.Errorf("got %d warnings, want %d", warnings, tc.wantWarnings)
				for _, m := range msgs {
					t.Errorf("  %s: %s", m.Level, m.Message)
				}
			}

			if tc.wantSubstr != "" {
				found := false
				for _, m := range msgs {
					if strings.Contains(m.Message, tc.wantSubstr) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected a message containing %q, got:", tc.wantSubstr)
					for _, m := range msgs {
						t.Errorf("  %s: %s", m.Level, m.Message)
					}
				}
			}
		})
	}
}

func TestValidateSections_OrderingDetail(t *testing.T) {
	sections := []BodySection{
		{Heading: "Vocabulary", Content: "content"},
		{Heading: "Anti-Patterns", Content: "content"},
		{Heading: "Evaluation Criteria", Content: "content"},
		{Heading: "Procedure", Content: "content"}, // out of order: should be before Evaluation Criteria
		{Heading: "Output Format", Content: "content"},
		{Heading: "Questions This Skill Answers", Content: "content"},
	}

	msgs := validateSections(sections, "high")

	found := false
	for _, m := range msgs {
		if m.Level == "error" && strings.Contains(m.Message, "Procedure") && strings.Contains(m.Message, "Evaluation Criteria") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected ordering error mentioning Procedure and Evaluation Criteria")
		for _, m := range msgs {
			t.Errorf("  %s: %s", m.Level, m.Message)
		}
	}
}

func TestValidateSections_UnknownSectionsNotErrors(t *testing.T) {
	sections := append(allRequiredSections(),
		BodySection{Heading: "My Custom Section", Content: "stuff"},
		BodySection{Heading: "Another Custom", Content: "more stuff"},
	)

	msgs := validateSections(sections, "high")

	for _, m := range msgs {
		if m.Level == "error" {
			t.Errorf("unexpected error: %s", m.Message)
		}
	}

	warnings := 0
	for _, m := range msgs {
		if m.Level == "warning" {
			warnings++
		}
	}
	if warnings != 2 {
		t.Errorf("got %d warnings, want 2", warnings)
	}
}

func TestParseSections_RoundTrip(t *testing.T) {
	input := allRequiredSectionsMarkdown()
	sections := parseSections(input)
	msgs := validateSections(sections, "high")

	for _, m := range msgs {
		if m.Level == "error" {
			t.Errorf("unexpected error from valid input: %s", m.Message)
		}
	}
}
