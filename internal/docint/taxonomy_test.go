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
