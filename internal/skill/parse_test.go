package skill

import (
	"fmt"
	"strings"
	"testing"
)

func validSKILLMD() string {
	return `---
name: my-skill
description:
  expert: Expert description
  natural: Natural description
triggers:
  - when asked about X
roles:
  - backend
stage: developing
constraint_level: medium
---
## Vocabulary

Some body content here.
`
}

func TestParseSKILLMD(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantErrs    int
		wantSubstr  string // substring expected in at least one error
		wantBody    string // expected body content (empty means don't check)
		wantLines   int    // expected line count (0 means don't check)
		wantName    string // expected frontmatter name (empty means don't check)
		wantNilData bool   // expect nil result
	}{
		{
			name:     "valid frontmatter and body",
			input:    validSKILLMD(),
			wantErrs: 0,
			wantBody: "## Vocabulary\n\nSome body content here.\n",
			wantName: "my-skill",
		},
		{
			name:       "missing opening delimiter",
			input:      "name: my-skill\n---\nbody\n",
			wantErrs:   1,
			wantSubstr: "missing opening frontmatter delimiter",
		},
		{
			name:       "missing closing delimiter",
			input:      "---\nname: my-skill\nbody\n",
			wantErrs:   1,
			wantSubstr: "missing closing frontmatter delimiter",
		},
		{
			name: "500-line file passes boundary",
			input: func() string {
				var b strings.Builder
				b.WriteString("---\nname: my-skill\n---\n")
				// 3 lines used so far, fill to 500
				for i := 0; i < 497; i++ {
					b.WriteString("line\n")
				}
				return b.String()
			}(),
			wantErrs:  0,
			wantLines: 500,
			wantName:  "my-skill",
		},
		{
			name: "501-line file returns error with line count",
			input: func() string {
				var b strings.Builder
				b.WriteString("---\nname: my-skill\n---\n")
				for i := 0; i < 498; i++ {
					b.WriteString("line\n")
				}
				return b.String()
			}(),
			wantErrs:   1,
			wantSubstr: "501",
			wantLines:  501,
			wantName:   "my-skill", // still parses despite error
		},
		{
			name:     "empty body after frontmatter is valid",
			input:    "---\nname: my-skill\n---\n",
			wantErrs: 0,
			wantBody: "",
			wantName: "my-skill",
		},
		{
			name:     "frontmatter only no trailing newline",
			input:    "---\nname: my-skill\n---",
			wantErrs: 0,
			wantBody: "",
			wantName: "my-skill",
		},
		{
			name: "unknown YAML field is rejected",
			input: `---
name: my-skill
unknown_field: oops
---
body
`,
			wantErrs:   1,
			wantSubstr: "unknown_field",
		},
		{
			name: "invalid YAML syntax returns error",
			input: `---
name: [invalid
---
body
`,
			wantErrs: 1,
		},
		{
			name: "body content is captured correctly",
			input: `---
name: my-skill
---
# Title

Paragraph one.

Paragraph two.
`,
			wantErrs: 0,
			wantBody: "# Title\n\nParagraph one.\n\nParagraph two.\n",
			wantName: "my-skill",
		},
		{
			name: "line count is accurate",
			input: `---
name: my-skill
---
line 4
line 5
`,
			wantErrs:  0,
			wantLines: 5,
		},
		{
			name:     "leading blank lines before opening delimiter",
			input:    "\n\n---\nname: my-skill\n---\nbody\n",
			wantErrs: 0,
			wantBody: "body\n",
			wantName: "my-skill",
		},
		{
			name:       "completely empty input",
			input:      "",
			wantErrs:   1,
			wantSubstr: "missing opening frontmatter delimiter",
		},
		{
			name:       "only whitespace input",
			input:      "   \n  \n",
			wantErrs:   1,
			wantSubstr: "missing opening frontmatter delimiter",
		},
		{
			name: "501-line file still parses frontmatter",
			input: func() string {
				var b strings.Builder
				b.WriteString("---\nname: parsed-despite-length\n---\n")
				for i := 0; i < 498; i++ {
					b.WriteString("x\n")
				}
				return b.String()
			}(),
			wantErrs: 1,
			wantName: "parsed-despite-length",
		},
		{
			name: "multiple errors accumulated",
			input: func() string {
				var b strings.Builder
				b.WriteString("---\nunknown: bad\n---\n")
				for i := 0; i < 498; i++ {
					b.WriteString("x\n")
				}
				return b.String()
			}(),
			wantErrs: 2, // line limit + unknown field
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, errs := parseSKILLMD([]byte(tc.input))

			if len(errs) != tc.wantErrs {
				t.Errorf("got %d errors, want %d", len(errs), tc.wantErrs)
				for i, err := range errs {
					t.Errorf("  error[%d]: %v", i, err)
				}
				return
			}

			if tc.wantSubstr != "" {
				found := false
				for _, err := range errs {
					if strings.Contains(err.Error(), tc.wantSubstr) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected an error containing %q, got:", tc.wantSubstr)
					for i, err := range errs {
						t.Errorf("  error[%d]: %v", i, err)
					}
				}
			}

			if result == nil {
				if !tc.wantNilData {
					// result can be nil when delimiters are missing
					return
				}
				t.Fatal("got nil result")
			}

			if tc.wantName != "" && result.Frontmatter.Name != tc.wantName {
				t.Errorf("frontmatter name = %q, want %q", result.Frontmatter.Name, tc.wantName)
			}

			if tc.wantBody != "" && result.BodyRaw != tc.wantBody {
				t.Errorf("body:\ngot:  %q\nwant: %q", result.BodyRaw, tc.wantBody)
			}
			// Also check empty body explicitly when set
			if tc.wantBody == "" && tc.wantName != "" && result.BodyRaw != "" {
				// Only check if we expected this to be empty (has a wantName indicating we're verifying the result)
				if tc.name == "empty body after frontmatter is valid" || tc.name == "frontmatter only no trailing newline" {
					t.Errorf("body: got %q, want empty", result.BodyRaw)
				}
			}

			if tc.wantLines != 0 && result.LineCount != tc.wantLines {
				t.Errorf("line count = %d, want %d", result.LineCount, tc.wantLines)
			}
		})
	}
}

func TestParseSKILLMD_MaxLinesConstant(t *testing.T) {
	if maxLines != 500 {
		t.Errorf("maxLines = %d, want 500 (FR-014)", maxLines)
	}
}

func TestParseSKILLMD_FullFrontmatterDecode(t *testing.T) {
	input := `---
name: my-skill
description:
  expert: Expert desc
  natural: Natural desc
triggers:
  - trigger one
  - trigger two
roles:
  - backend
  - frontend
stage: developing
constraint_level: high
---
Body here.
`
	result, errs := parseSKILLMD([]byte(input))
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	fm := result.Frontmatter
	if fm.Name != "my-skill" {
		t.Errorf("Name = %q, want %q", fm.Name, "my-skill")
	}
	if fm.Description.Expert != "Expert desc" {
		t.Errorf("Description.Expert = %q", fm.Description.Expert)
	}
	if fm.Description.Natural != "Natural desc" {
		t.Errorf("Description.Natural = %q", fm.Description.Natural)
	}
	if len(fm.Triggers) != 2 || fm.Triggers[0] != "trigger one" || fm.Triggers[1] != "trigger two" {
		t.Errorf("Triggers = %v", fm.Triggers)
	}
	if len(fm.Roles) != 2 || fm.Roles[0] != "backend" || fm.Roles[1] != "frontend" {
		t.Errorf("Roles = %v", fm.Roles)
	}
	if fm.Stage != "developing" {
		t.Errorf("Stage = %q", fm.Stage)
	}
	if fm.ConstraintLevel != "high" {
		t.Errorf("ConstraintLevel = %q", fm.ConstraintLevel)
	}
	if result.BodyRaw != "Body here.\n" {
		t.Errorf("BodyRaw = %q, want %q", result.BodyRaw, "Body here.\n")
	}
}

func TestParseSKILLMD_LineCountExact(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"single line no newline", "---", 1},
		{"three lines", "---\nname: x\n---", 3},
		{"trailing newline adds line", "---\nname: x\n---\n", 3},
		{"five lines", "a\nb\nc\nd\ne", 5},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, _ := parseSKILLMD([]byte(tc.input))
			if result == nil {
				t.Fatal("result is nil")
			}
			if result.LineCount != tc.want {
				t.Errorf("LineCount = %d, want %d (input: %q)", result.LineCount, tc.want, tc.input)
			}
		})
	}
}

func TestParseSKILLMD_ExactBoundary(t *testing.T) {
	// Build a file with exactly 500 lines
	var b strings.Builder
	b.WriteString("---\nname: boundary\n---\n")
	for i := 4; i <= 500; i++ {
		fmt.Fprintf(&b, "line %d\n", i)
	}
	input500 := b.String()

	// Build a file with exactly 501 lines
	b.Reset()
	b.WriteString("---\nname: boundary\n---\n")
	for i := 4; i <= 501; i++ {
		fmt.Fprintf(&b, "line %d\n", i)
	}
	input501 := b.String()

	result500, errs500 := parseSKILLMD([]byte(input500))
	if len(errs500) != 0 {
		t.Errorf("500 lines: unexpected errors: %v", errs500)
	}
	if result500.LineCount != 500 {
		t.Errorf("500 lines: LineCount = %d", result500.LineCount)
	}

	result501, errs501 := parseSKILLMD([]byte(input501))
	if len(errs501) != 1 {
		t.Errorf("501 lines: got %d errors, want 1", len(errs501))
	}
	if result501.LineCount != 501 {
		t.Errorf("501 lines: LineCount = %d", result501.LineCount)
	}
}
