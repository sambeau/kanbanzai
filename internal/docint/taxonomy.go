package docint

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
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
// Non-Goals → RoleConstraint: exclusions are scope constraints, not requirements (see suggestedClassTable comment).
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

// SuggestedClassification is a heading-pattern derived classification hint (REQ-005).
type SuggestedClassification struct {
	SectionPath string `json:"section_path"`
	Title       string `json:"title"`
	Role        string `json:"role"`
	Confidence  string `json:"confidence"`
}

// suggestedClassTable maps normalised heading keywords to roles per REQ-007.
// Entries are matched by exact equality after normaliseHeading().
var suggestedClassTable = []struct {
	keyword string
	role    FragmentRole
}{
	// Longer phrases first to avoid shorter substring matches from shadowing them.
	{"alternatives considered", RoleAlternative},
	{"problem and motivation", RoleRationale},
	{"problem statement", RoleRationale},
	{"executive summary", RoleNarrative},
	{"reference table", RoleDefinition},
	{"acceptance criteria", RoleRequirement},
	{"out of scope", RoleConstraint},
	{"in scope", RoleConstraint},
	// Non-Goals maps to RoleConstraint (not RoleRequirement): non-goals are scope
	// constraints that explicitly exclude functionality, not positive requirements.
	// This conflicts with an ambiguous reading of REQ-004; we treat them as constraints.
	{"non-goals", RoleConstraint},
	{"goals", RoleRequirement},
	{"requirements", RoleRequirement},
	{"summary", RoleNarrative},
	{"decisions", RoleDecision},
	{"design", RoleDecision},
	{"alternative", RoleAlternative},
	{"assumption", RoleAssumption},
	{"assumptions", RoleAssumption},
	{"background", RoleNarrative},
	{"decision", RoleDecision},
	{"deferred", RoleConstraint},
	{"definition", RoleDefinition},
	{"definitions", RoleDefinition},
	{"excluded", RoleConstraint},
	{"example", RoleExample},
	{"glossary", RoleDefinition},
	{"motivation", RoleRationale},
	{"overview", RoleNarrative},
	{"purpose", RoleRationale},
	{"risk", RoleRisk},
	{"risks", RoleRisk},
	{"sample", RoleExample},
	{"scope", RoleConstraint},
}

// reACPattern matches heading titles that identify acceptance-criteria sections (REQ-007).
var reACPattern = regexp.MustCompile(`AC-\d+`)

// reDPattern matches heading titles that identify decision sections (REQ-007).
var reDPattern = regexp.MustCompile(`D\d+:`)

// normaliseHeading lowercases a heading title and collapses all whitespace to
// single spaces. Used for case-insensitive normalised-whitespace matching (REQ-008).
func normaliseHeading(title string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(title)), " "))
}

// matchSuggestedRole checks a heading title against the REQ-007 exact table and
// the REQ-106 prefix table. Returns the matched role and true if a match is found.
func matchSuggestedRole(title string) (FragmentRole, bool) {
	normalized := normaliseHeading(title)
	// Exact-match table (REQ-007).
	for _, entry := range suggestedClassTable {
		if normalized == entry.keyword {
			return entry.role, true
		}
	}
	// Regex patterns are applied to the original title (REQ-008).
	if reACPattern.MatchString(title) {
		return RoleRequirement, true
	}
	if reDPattern.MatchString(title) {
		return RoleDecision, true
	}
	// Prefix-match table (REQ-106).
	return matchSuggestedRolePrefix(title)
}

// SuggestClassifications returns heading-pattern derived classification hints for all
// sections in the document index per REQ-007. The result is always non-nil (REQ-006).
// The handler must never write these suggestions to the classification store (REQ-008).
func SuggestClassifications(index *DocumentIndex) []SuggestedClassification {
	seen := make(map[string]bool)
	var result []SuggestedClassification

	// Walk all sections recursively and apply heading-pattern matching.
	collectSuggestions(index.Sections, seen, &result)

	// Front matter special case: if the document has a key-value metadata table
	// in its first section, suggest narrative for that section (REQ-007).
	if index.FrontMatter != nil && len(index.Sections) > 0 {
		first := index.Sections[0]
		if !seen[first.Path] {
			seen[first.Path] = true
			result = append(result, SuggestedClassification{
				SectionPath: first.Path,
				Title:       first.Title,
				Role:        string(RoleNarrative),
				Confidence:  "high",
			})
		}
	}

	if result == nil {
		return []SuggestedClassification{}
	}
	return result
}

// collectSuggestions recursively walks a section tree and appends matched entries.
func collectSuggestions(sections []Section, seen map[string]bool, result *[]SuggestedClassification) {
	for _, s := range sections {
		if !seen[s.Path] {
			if role, ok := matchSuggestedRole(s.Title); ok {
				seen[s.Path] = true
				*result = append(*result, SuggestedClassification{
					SectionPath: s.Path,
					Title:       s.Title,
					Role:        string(role),
					Confidence:  "high",
				})
			}
		}
		if len(s.Children) > 0 {
			collectSuggestions(s.Children, seen, result)
		}
	}
}

// suggestedClassPrefixTable maps normalised heading prefixes to roles per REQ-106.
// Entries are matched by case-insensitive prefix comparison after normaliseHeading().
// Longer prefixes appear first to ensure more specific patterns win.
var suggestedClassPrefixTable = []struct {
	prefix string
	role   FragmentRole
}{
	{"requirements", RoleRequirement},
	{"definition", RoleDefinition},
	{"decisions", RoleDecision},
	{"overview", RoleNarrative},
	{"glossary", RoleDefinition},
	{"summary", RoleNarrative},
	{"design", RoleDecision},
	{"goals", RoleRequirement},
	{"risk", RoleRisk},
}

// matchSuggestedRolePrefix checks a heading title against the REQ-106 prefix table.
// Returns the matched role and true if a match is found.
func matchSuggestedRolePrefix(title string) (FragmentRole, bool) {
	normalized := normaliseHeading(title)
	for _, entry := range suggestedClassPrefixTable {
		if normalized == entry.prefix || strings.HasPrefix(normalized, entry.prefix+" ") || strings.HasPrefix(normalized, entry.prefix+"/") || strings.HasPrefix(normalized, entry.prefix+"-") {
			return entry.role, true
		}
	}
	return "", false
}

// stopWords is the fixed set of words excluded from concept derivation (REQ-104).
var stopWords = map[string]bool{
	"a": true, "an": true, "the": true,
	"of": true, "in": true, "on": true, "at": true, "to": true, "for": true,
	"with": true, "by": true, "from": true, "as": true,
	"and": true, "or": true, "but": true,
	"it": true, "its": true,
	"is": true, "are": true, "was": true, "were": true, "be": true, "been": true,
	"have": true, "has": true,
}

// titleSplitter splits on slash, hyphen, and whitespace for concept token extraction.
var titleSplitter = regexp.MustCompile(`[\s/\-]+`)

// deriveConcepts applies the lexical pipeline to a single title string and returns
// the resulting tokens (REQ-104, tasks 1-5).
func deriveConcepts(title string) []string {
	raw := titleSplitter.Split(title, -1)
	seen := make(map[string]bool)
	var out []string
	for _, tok := range raw {
		tok = strings.TrimSpace(tok)
		if len(tok) < 2 {
			continue
		}
		lower := strings.ToLower(tok)
		if stopWords[lower] {
			continue
		}
		runes := []rune(tok)
		titled := string(unicode.ToUpper(runes[0])) + string(runes[1:])
		if !seen[titled] {
			seen[titled] = true
			out = append(out, titled)
		}
	}
	return out
}

// ConceptSuggestion holds per-section concept name candidates (REQ-102).
type ConceptSuggestion struct {
	SectionPath       string   `json:"section_path"`
	SectionTitle      string   `json:"section_title"`
	SuggestedConcepts []string `json:"suggested_concepts"`
}

// SuggestConcepts derives concept candidates for each section in the document index
// using a lexical pass over section titles and their ancestor titles (REQ-101–104).
// The result is always non-nil (REQ-101). Sections yielding no tokens are omitted (REQ-103).
func SuggestConcepts(index *DocumentIndex) []ConceptSuggestion {
	var result []ConceptSuggestion
	collectConceptSuggestions(index.Sections, nil, &result)
	if result == nil {
		return []ConceptSuggestion{}
	}
	return result
}

// collectConceptSuggestions recursively walks sections, tracking ancestor titles.
func collectConceptSuggestions(sections []Section, ancestors []string, result *[]ConceptSuggestion) {
	for _, s := range sections {
		// Build combined title: ancestors + current title (REQ-104).
		// Use a safe copy to avoid sharing the backing array with the caller's slice.
		allTitles := make([]string, len(ancestors)+1)
		copy(allTitles, ancestors)
		allTitles[len(ancestors)] = s.Title

		// Only include this section if its own title yields at least one token (REQ-103).
		// Ancestor tokens enrich the concept list but do not determine inclusion.
		if ownTokens := deriveConcepts(s.Title); len(ownTokens) > 0 {
			seen := make(map[string]bool)
			var tokens []string
			for _, title := range allTitles {
				for _, tok := range deriveConcepts(title) {
					if !seen[tok] {
						seen[tok] = true
						tokens = append(tokens, tok)
					}
				}
			}
			*result = append(*result, ConceptSuggestion{
				SectionPath:       s.Path,
				SectionTitle:      s.Title,
				SuggestedConcepts: tokens,
			})
		}
		if len(s.Children) > 0 {
			collectConceptSuggestions(s.Children, allTitles, result)
		}
	}
}
