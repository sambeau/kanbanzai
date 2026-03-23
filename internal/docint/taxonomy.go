package docint

import (
	"fmt"
	"strings"
)

// FragmentRole defines the valid roles for document fragment classification.
type FragmentRole string

const (
	RoleRequirement FragmentRole = "requirement"
	RoleDecision    FragmentRole = "decision"
	RoleRationale   FragmentRole = "rationale"
	RoleConstraint  FragmentRole = "constraint"
	RoleAssumption  FragmentRole = "assumption"
	RoleRisk        FragmentRole = "risk"
	RoleQuestion    FragmentRole = "question"
	RoleDefinition  FragmentRole = "definition"
	RoleExample     FragmentRole = "example"
	RoleAlternative FragmentRole = "alternative"
	RoleNarrative   FragmentRole = "narrative"
)

// AllRoles returns all valid fragment roles.
func AllRoles() []FragmentRole {
	return []FragmentRole{
		RoleRequirement, RoleDecision, RoleRationale, RoleConstraint,
		RoleAssumption, RoleRisk, RoleQuestion, RoleDefinition,
		RoleExample, RoleAlternative, RoleNarrative,
	}
}

// ValidRole returns true if the role string is in the taxonomy.
func ValidRole(role string) bool {
	for _, r := range AllRoles() {
		if string(r) == role {
			return true
		}
	}
	return false
}

// ValidConfidence returns true if the confidence level is valid.
func ValidConfidence(conf string) bool {
	switch conf {
	case "high", "medium", "low":
		return true
	}
	return false
}

// ValidateClassification validates a single classification against the taxonomy.
func ValidateClassification(c Classification) error {
	if c.SectionPath == "" {
		return fmt.Errorf("classification missing section_path")
	}
	if !ValidRole(c.Role) {
		return fmt.Errorf("invalid fragment role: %q (valid: %v)", c.Role, AllRoles())
	}
	if !ValidConfidence(c.Confidence) {
		return fmt.Errorf("invalid confidence: %q (valid: high, medium, low)", c.Confidence)
	}
	return nil
}

// conventionalRoleKeywords maps heading keywords to fragment roles.
// Used by Layer 2 pattern-based extraction.
var conventionalRoleKeywords = map[string]FragmentRole{
	"decision":                RoleDecision,
	"decisions":               RoleDecision,
	"rationale":               RoleRationale,
	"requirement":             RoleRequirement,
	"requirements":            RoleRequirement,
	"constraint":              RoleConstraint,
	"constraints":             RoleConstraint,
	"assumption":              RoleAssumption,
	"assumptions":             RoleAssumption,
	"risk":                    RoleRisk,
	"risks":                   RoleRisk,
	"question":                RoleQuestion,
	"questions":               RoleQuestion,
	"open question":           RoleQuestion,
	"open questions":          RoleQuestion,
	"definition":              RoleDefinition,
	"definitions":             RoleDefinition,
	"glossary":                RoleDefinition,
	"example":                 RoleExample,
	"examples":                RoleExample,
	"alternative":             RoleAlternative,
	"alternatives":            RoleAlternative,
	"alternatives considered": RoleAlternative,
	"acceptance criteria":     RoleRequirement,
	"non-goals":               RoleConstraint,
	"non-goal":                RoleConstraint,
	"scope":                   RoleConstraint,
	"summary":                 RoleNarrative,
	"overview":                RoleNarrative,
	"purpose":                 RoleNarrative,
	"background":              RoleNarrative,
	"context":                 RoleNarrative,
}

// conventionalRoleKeywordsOrdered is a deterministic ordering of keywords for substring matching.
// Longer keywords appear first to ensure more specific patterns match before shorter ones.
var conventionalRoleKeywordsOrdered = []struct {
	keyword string
	role    FragmentRole
}{
	{"alternatives considered", RoleAlternative},
	{"acceptance criteria", RoleRequirement},
	{"open questions", RoleQuestion},
	{"open question", RoleQuestion},
	{"alternatives", RoleAlternative},
	{"alternative", RoleAlternative},
	{"assumptions", RoleAssumption},
	{"assumption", RoleAssumption},
	{"background", RoleNarrative},
	{"constraints", RoleConstraint},
	{"constraint", RoleConstraint},
	{"decisions", RoleDecision},
	{"decision", RoleDecision},
	{"definitions", RoleDefinition},
	{"definition", RoleDefinition},
	{"examples", RoleExample},
	{"example", RoleExample},
	{"glossary", RoleDefinition},
	{"non-goals", RoleConstraint},
	{"non-goal", RoleConstraint},
	{"overview", RoleNarrative},
	{"purpose", RoleNarrative},
	{"questions", RoleQuestion},
	{"question", RoleQuestion},
	{"rationale", RoleRationale},
	{"requirements", RoleRequirement},
	{"requirement", RoleRequirement},
	{"context", RoleNarrative},
	{"risks", RoleRisk},
	{"risk", RoleRisk},
	{"scope", RoleConstraint},
	{"summary", RoleNarrative},
}

// MatchConventionalRole checks if a heading title matches a conventional role keyword.
// Returns the role and true if matched, or empty and false if not.
func MatchConventionalRole(headingTitle string) (FragmentRole, bool) {
	lower := strings.ToLower(strings.TrimSpace(headingTitle))

	// Check exact match first
	if role, ok := conventionalRoleKeywords[lower]; ok {
		return role, true
	}

	// Check if heading contains a keyword (ordered, deterministic)
	for _, kw := range conventionalRoleKeywordsOrdered {
		if strings.Contains(lower, kw.keyword) {
			return kw.role, true
		}
	}

	return "", false
}
