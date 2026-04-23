package docint

import (
	"testing"
)

func TestAllRoles(t *testing.T) {
	roles := AllRoles()
	if len(roles) != 11 {
		t.Errorf("expected 11 roles, got %d", len(roles))
	}

	// Verify all expected roles are present.
	expected := map[FragmentRole]bool{
		RoleRequirement: true,
		RoleDecision:    true,
		RoleRationale:   true,
		RoleConstraint:  true,
		RoleAssumption:  true,
		RoleRisk:        true,
		RoleQuestion:    true,
		RoleDefinition:  true,
		RoleExample:     true,
		RoleAlternative: true,
		RoleNarrative:   true,
	}
	for _, r := range roles {
		if !expected[r] {
			t.Errorf("unexpected role in AllRoles: %q", r)
		}
		delete(expected, r)
	}
	for r := range expected {
		t.Errorf("missing role from AllRoles: %q", r)
	}
}

func TestValidRole(t *testing.T) {
	tests := []struct {
		role string
		want bool
	}{
		{"requirement", true},
		{"decision", true},
		{"rationale", true},
		{"constraint", true},
		{"assumption", true},
		{"risk", true},
		{"question", true},
		{"definition", true},
		{"example", true},
		{"alternative", true},
		{"narrative", true},
		{"unknown", false},
		{"", false},
		{"Requirement", false},
		{"DECISION", false},
	}
	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			got := ValidRole(tt.role)
			if got != tt.want {
				t.Errorf("ValidRole(%q) = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}

func TestValidConfidence(t *testing.T) {
	tests := []struct {
		conf string
		want bool
	}{
		{"high", true},
		{"medium", true},
		{"low", true},
		{"", false},
		{"High", false},
		{"unknown", false},
		{"very_high", false},
	}
	for _, tt := range tests {
		t.Run(tt.conf, func(t *testing.T) {
			got := ValidConfidence(tt.conf)
			if got != tt.want {
				t.Errorf("ValidConfidence(%q) = %v, want %v", tt.conf, got, tt.want)
			}
		})
	}
}

func TestValidateClassification(t *testing.T) {
	t.Run("valid classification", func(t *testing.T) {
		c := Classification{
			SectionPath: "1.2",
			Role:        "decision",
			Confidence:  "high",
		}
		if err := ValidateClassification(c); err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})

	t.Run("valid with optional fields", func(t *testing.T) {
		c := Classification{
			SectionPath:   "1.2.3",
			Role:          "requirement",
			Confidence:    "medium",
			Summary:       "Must support YAML output",
			ConceptsIntro: []ConceptIntroEntry{{Name: "yaml-output"}},
			ConceptsUsed:  []string{"serialisation"},
		}
		if err := ValidateClassification(c); err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})

	t.Run("missing section_path", func(t *testing.T) {
		c := Classification{
			Role:       "decision",
			Confidence: "high",
		}
		err := ValidateClassification(c)
		if err == nil {
			t.Fatal("expected error for missing section_path")
		}
		if got := err.Error(); got != "classification missing section_path" {
			t.Errorf("unexpected error message: %s", got)
		}
	})

	t.Run("invalid role", func(t *testing.T) {
		c := Classification{
			SectionPath: "1",
			Role:        "bogus",
			Confidence:  "high",
		}
		err := ValidateClassification(c)
		if err == nil {
			t.Fatal("expected error for invalid role")
		}
	})

	t.Run("invalid confidence", func(t *testing.T) {
		c := Classification{
			SectionPath: "1",
			Role:        "decision",
			Confidence:  "very_high",
		}
		err := ValidateClassification(c)
		if err == nil {
			t.Fatal("expected error for invalid confidence")
		}
	})
}

func TestMatchConventionalRole(t *testing.T) {
	tests := []struct {
		name      string
		heading   string
		wantRole  FragmentRole
		wantMatch bool
	}{
		{"exact singular", "Decision", RoleDecision, true},
		{"exact plural", "Decisions", RoleDecision, true},
		{"case insensitive", "REQUIREMENTS", RoleRequirement, true},
		{"open questions", "Open Questions", RoleQuestion, true},
		{"alternatives considered", "Alternatives Considered", RoleAlternative, true},
		{"acceptance criteria", "Acceptance Criteria", RoleRequirement, true},
		{"non-goals", "Non-Goals", RoleConstraint, true},
		{"glossary", "Glossary", RoleDefinition, true},
		{"summary", "Summary", RoleNarrative, true},
		{"overview", "Overview", RoleNarrative, true},
		{"background", "Background", RoleNarrative, true},
		{"scope", "Scope", RoleConstraint, true},
		{"with whitespace", "  Risks  ", RoleRisk, true},
		{"no match", "Foo bar", "", false},
		{"empty string", "", "", false},
		{"partial keyword in heading", "Project Assumptions", RoleAssumption, true},
		{"multiple keywords deterministic", "Risk Assumptions", RoleAssumption, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRole, gotMatch := MatchConventionalRole(tt.heading)
			if gotMatch != tt.wantMatch {
				t.Errorf("MatchConventionalRole(%q) matched=%v, want %v", tt.heading, gotMatch, tt.wantMatch)
			}
			if gotMatch && gotRole != tt.wantRole {
				t.Errorf("MatchConventionalRole(%q) role=%q, want %q", tt.heading, gotRole, tt.wantRole)
			}
		})
	}
}

// ─── SuggestClassifications tests ─────────────────────────────────────────────

func TestNormaliseHeading(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Overview", "overview"},
		{"Acceptance Criteria", "acceptance criteria"},
		{"  Risks  ", "risks"},
		{"Multiple   Spaces", "multiple spaces"},
		{"UPPERCASE", "uppercase"},
		{"Mixed\tTab", "mixed tab"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normaliseHeading(tt.input)
			if got != tt.want {
				t.Errorf("normaliseHeading(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMatchSuggestedRole(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		wantRole FragmentRole
		wantOK   bool
	}{
		// Static table entries
		{"acceptance criteria", "Acceptance Criteria", RoleRequirement, true},
		{"purpose", "Purpose", RoleRationale, true},
		{"motivation", "Motivation", RoleRationale, true},
		{"problem statement", "Problem Statement", RoleRationale, true},
		{"problem and motivation", "Problem and Motivation", RoleRationale, true},
		{"scope", "Scope", RoleConstraint, true},
		{"in scope", "In Scope", RoleConstraint, true},
		{"out of scope", "Out of Scope", RoleConstraint, true},
		{"deferred", "Deferred", RoleConstraint, true},
		{"excluded", "Excluded", RoleConstraint, true},
		{"non-goals", "Non-Goals", RoleConstraint, true},
		{"glossary", "Glossary", RoleDefinition, true},
		{"definitions", "Definitions", RoleDefinition, true},
		{"reference table", "Reference Table", RoleDefinition, true},
		{"definition", "Definition", RoleDefinition, true},
		{"example", "Example", RoleExample, true},
		{"sample", "Sample", RoleExample, true},
		{"alternatives considered", "Alternatives Considered", RoleAlternative, true},
		{"alternative", "Alternative", RoleAlternative, true},
		{"overview", "Overview", RoleNarrative, true},
		{"background", "Background", RoleNarrative, true},
		{"executive summary", "Executive Summary", RoleNarrative, true},
		{"decision", "Decision", RoleDecision, true},
		{"risk", "Risk", RoleRisk, true},
		{"risks", "Risks", RoleRisk, true},
		{"assumption", "Assumption", RoleAssumption, true},
		{"assumptions", "Assumptions", RoleAssumption, true},
		// Case insensitivity
		{"lower case", "acceptance criteria", RoleRequirement, true},
		{"upper case", "OVERVIEW", RoleNarrative, true},
		// Regex patterns
		{"AC-1 regex", "AC-1", RoleRequirement, true},
		{"AC-001 regex", "AC-001: User login", RoleRequirement, true},
		{"D1: regex", "D1: Use PostgreSQL", RoleDecision, true},
		{"D42: regex", "D42: Caching strategy", RoleDecision, true},
		// No match
		{"no match", "Introduction", "", false},
		{"empty", "", "", false},
		// "Goals and Non-Goals" starts with "Goals " (space), triggering the "goals"
		// prefix entry → RoleRequirement. This is a known approximation: prefix-based
		// classification is inherently coarse, and agents are expected to review and
		// refine suggested classifications before accepting them.
		{"goals-and-non-goals (prefix-match approximation)", "Goals and Non-Goals", RoleRequirement, true},
		// REQ-106 prefix-match entries
		{"goals prefix", "Goals Overview", RoleRequirement, true},
		{"requirements prefix", "Requirements Overview", RoleRequirement, true},
		{"summary prefix", "Summary of Changes", RoleNarrative, true},
		{"decisions prefix", "Decisions Log", RoleDecision, true},
		{"design prefix", "Design Decisions", RoleDecision, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRole, gotOK := matchSuggestedRole(tt.title)
			if gotOK != tt.wantOK {
				t.Errorf("matchSuggestedRole(%q) ok=%v, want %v", tt.title, gotOK, tt.wantOK)
			}
			if gotOK && gotRole != tt.wantRole {
				t.Errorf("matchSuggestedRole(%q) role=%q, want %q", tt.title, gotRole, tt.wantRole)
			}
		})
	}
}

func TestSuggestClassifications_Empty(t *testing.T) {
	// AC-004: no recognisable headings → empty (non-nil) slice
	index := &DocumentIndex{
		Sections: []Section{
			{Path: "1", Title: "Introduction", Level: 1},
			{Path: "2", Title: "Conclusion", Level: 1},
		},
	}
	result := SuggestClassifications(index)
	if result == nil {
		t.Error("SuggestClassifications returned nil, want empty slice")
	}
	if len(result) != 0 {
		t.Errorf("expected 0 suggestions, got %d: %v", len(result), result)
	}
}

func TestSuggestClassifications_AcceptanceCriteria(t *testing.T) {
	// AC-005: "Acceptance Criteria" section → requirement, high
	index := &DocumentIndex{
		Sections: []Section{
			{Path: "1", Title: "Introduction", Level: 1},
			{Path: "2", Title: "Acceptance Criteria", Level: 1},
			{Path: "3", Title: "Summary", Level: 1},
		},
	}
	result := SuggestClassifications(index)
	found := false
	for _, s := range result {
		if s.SectionPath == "2" {
			found = true
			if s.Role != "requirement" {
				t.Errorf("AC section role = %q, want %q", s.Role, "requirement")
			}
			if s.Confidence != "high" {
				t.Errorf("AC section confidence = %q, want %q", s.Confidence, "high")
			}
			break
		}
	}
	if !found {
		t.Error("expected suggestion for section 2 (Acceptance Criteria), not found")
	}
}

func TestSuggestClassifications_AlternativesConsidered(t *testing.T) {
	// AC-007: "Alternatives Considered" → alternative, high
	index := &DocumentIndex{
		Sections: []Section{
			{Path: "1", Title: "Design", Level: 1},
			{Path: "2", Title: "Alternatives Considered", Level: 1},
		},
	}
	result := SuggestClassifications(index)
	found := false
	for _, s := range result {
		if s.SectionPath == "2" {
			found = true
			if s.Role != "alternative" {
				t.Errorf("alt section role = %q, want %q", s.Role, "alternative")
			}
			if s.Confidence != "high" {
				t.Errorf("alt section confidence = %q, want %q", s.Confidence, "high")
			}
			break
		}
	}
	if !found {
		t.Error("expected suggestion for section 2 (Alternatives Considered), not found")
	}
}

func TestSuggestClassifications_FrontMatter(t *testing.T) {
	// AC-006: front-matter key-value section → narrative for first section
	index := &DocumentIndex{
		FrontMatter: &FrontMatter{Type: "design", Status: "draft"},
		Sections: []Section{
			{Path: "1", Title: "My Design Document", Level: 1},
			{Path: "2", Title: "Scope", Level: 1},
		},
	}
	result := SuggestClassifications(index)
	found := false
	for _, s := range result {
		if s.SectionPath == "1" && s.Role == "narrative" {
			found = true
			if s.Confidence != "high" {
				t.Errorf("front matter section confidence = %q, want %q", s.Confidence, "high")
			}
			break
		}
	}
	if !found {
		t.Error("expected narrative suggestion for first section with front matter, not found")
	}
}

func TestSuggestClassifications_RegexPatterns(t *testing.T) {
	// REQ-008 regex patterns: AC-\d+ → requirement, D\d+: → decision
	index := &DocumentIndex{
		Sections: []Section{
			{Path: "1", Title: "AC-001: User must log in", Level: 2},
			{Path: "2", Title: "D1: Use PostgreSQL", Level: 2},
			{Path: "3", Title: "AC-42", Level: 2},
		},
	}
	result := SuggestClassifications(index)
	roleFor := make(map[string]string)
	for _, s := range result {
		roleFor[s.SectionPath] = s.Role
	}
	if roleFor["1"] != "requirement" {
		t.Errorf("AC-001 section role = %q, want requirement", roleFor["1"])
	}
	if roleFor["2"] != "decision" {
		t.Errorf("D1: section role = %q, want decision", roleFor["2"])
	}
	if roleFor["3"] != "requirement" {
		t.Errorf("AC-42 section role = %q, want requirement", roleFor["3"])
	}
}

func TestSuggestClassifications_Children(t *testing.T) {
	// Section headings in children are also matched.
	index := &DocumentIndex{
		Sections: []Section{
			{
				Path:  "1",
				Title: "Context",
				Level: 1,
				Children: []Section{
					{Path: "1.1", Title: "Assumptions", Level: 2},
					{Path: "1.2", Title: "No match here", Level: 2},
				},
			},
		},
	}
	result := SuggestClassifications(index)
	found := false
	for _, s := range result {
		if s.SectionPath == "1.1" && s.Role == "assumption" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected assumption suggestion for child section 1.1")
	}
}

func TestSuggestClassifications_NoDuplicates(t *testing.T) {
	// Front matter + heading pattern on the same section should not produce duplicates.
	index := &DocumentIndex{
		FrontMatter: &FrontMatter{Type: "design"},
		Sections: []Section{
			{Path: "1", Title: "Overview", Level: 1}, // matches both front-matter and overview→narrative
			{Path: "2", Title: "Scope", Level: 1},
		},
	}
	result := SuggestClassifications(index)
	// Count how many times section "1" appears
	count := 0
	for _, s := range result {
		if s.SectionPath == "1" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("section 1 appears %d times in suggestions, want 1", count)
	}
}

func TestSuggestClassifications_AllConfidenceHigh(t *testing.T) {
	// AC-004: every entry must have confidence:"high"
	index := &DocumentIndex{
		Sections: []Section{
			{Path: "1", Title: "Assumptions", Level: 1},
			{Path: "2", Title: "Glossary", Level: 1},
			{Path: "3", Title: "Acceptance Criteria", Level: 1},
		},
	}
	result := SuggestClassifications(index)
	for _, s := range result {
		if s.Confidence != "high" {
			t.Errorf("entry %q has confidence=%q, want high", s.SectionPath, s.Confidence)
		}
	}
}

func TestNonGoalsMapsToConstraint(t *testing.T) {
	// Decision: Non-Goals is classified as RoleConstraint (scope exclusion),
	// not RoleRequirement. This test locks the decision to prevent accidental changes.
	role, ok := matchSuggestedRole("Non-Goals")
	if !ok {
		t.Fatal("matchSuggestedRole('Non-Goals') returned no match")
	}
	if role != RoleConstraint {
		t.Errorf("Non-Goals role = %q, want %q (constraint, not requirement)", role, RoleConstraint)
	}
}

func TestMatchSuggestedRole_REQ004_Patterns(t *testing.T) {
	cases := []struct {
		title    string
		wantRole FragmentRole
	}{
		{"Problem Statement", RoleRationale},
		{"Motivation", RoleRationale},
		{"Definitions", RoleDefinition},
		{"Executive Summary", RoleNarrative},
	}
	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			role, ok := matchSuggestedRole(c.title)
			if !ok {
				t.Fatalf("matchSuggestedRole(%q) returned no match, want %q", c.title, c.wantRole)
			}
			if role != c.wantRole {
				t.Errorf("matchSuggestedRole(%q) = %q, want %q", c.title, role, c.wantRole)
			}
		})
	}
}
