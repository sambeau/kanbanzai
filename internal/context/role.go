package context

import (
	"fmt"
	"strings"
)

// AntiPattern represents a single anti-pattern entry in a role definition.
type AntiPattern struct {
	Name    string `yaml:"name"    json:"name"`
	Detect  string `yaml:"detect"  json:"detect"`
	Because string `yaml:"because" json:"because"`
	Resolve string `yaml:"resolve" json:"resolve"`
}

// Role is a role definition as loaded from a YAML file.
// Strict parsing: unknown fields are rejected (NFR-002).
type Role struct {
	ID           string        `yaml:"id"`
	Inherits     string        `yaml:"inherits,omitempty"`
	Identity     string        `yaml:"identity"`
	Vocabulary   []string      `yaml:"vocabulary"`
	AntiPatterns []AntiPattern `yaml:"anti_patterns,omitempty"`
	Tools        []string      `yaml:"tools,omitempty"`
}

// ResolvedRole is the result of walking the inheritance chain.
type ResolvedRole struct {
	ID           string
	Identity     string        // always from leaf role (FR-010)
	Vocabulary   []string      // parent ++ child concatenation (FR-010)
	AntiPatterns []AntiPattern // parent ++ child concatenation (FR-010)
	Tools        []string      // union, no duplicates (FR-010)
}

// validateRole checks all field-level invariants and accumulates errors.
// Returns nil if valid, or an error containing all validation failures.
// expectedID is the ID derived from the filename (without .yaml extension).
func validateRole(r *Role, expectedID string) error {
	var errs []string

	// FR-002: id is required.
	if r.ID == "" {
		errs = append(errs, "missing required field 'id'")
	} else {
		// FR-003: id must match the idRegexp format.
		if !idRegexp.MatchString(r.ID) {
			errs = append(errs, fmt.Sprintf("invalid id %q: must be lowercase alphanumeric and hyphens, 2-30 chars", r.ID))
		}
		// FR-001: id must match filename.
		if r.ID != expectedID {
			errs = append(errs, fmt.Sprintf("id %q does not match filename %q", r.ID, expectedID))
		}
	}

	// FR-004: identity is required, non-empty, under 50 tokens.
	if r.Identity == "" {
		errs = append(errs, "missing required field 'identity'")
	} else {
		tokenCount := len(strings.Fields(r.Identity))
		if tokenCount > 50 {
			errs = append(errs, fmt.Sprintf("identity exceeds 50-token limit (%d tokens)", tokenCount))
		}
	}

	// FR-005: vocabulary is required and must be non-empty.
	if len(r.Vocabulary) == 0 {
		errs = append(errs, "missing required field 'vocabulary': must be a non-empty list")
	}

	// FR-006: each anti-pattern must have all four fields non-empty.
	for i, ap := range r.AntiPatterns {
		if ap.Name == "" {
			errs = append(errs, fmt.Sprintf("anti_patterns[%d]: missing required field 'name'", i))
		}
		if ap.Detect == "" {
			errs = append(errs, fmt.Sprintf("anti_patterns[%d]: missing required field 'detect'", i))
		}
		if ap.Because == "" {
			errs = append(errs, fmt.Sprintf("anti_patterns[%d]: missing required field 'because'", i))
		}
		if ap.Resolve == "" {
			errs = append(errs, fmt.Sprintf("anti_patterns[%d]: missing required field 'resolve'", i))
		}
	}

	// FR-007: tools list must not contain duplicates.
	if len(r.Tools) > 0 {
		seen := make(map[string]bool, len(r.Tools))
		for _, t := range r.Tools {
			if seen[t] {
				errs = append(errs, fmt.Sprintf("tools: duplicate entry %q", t))
			}
			seen[t] = true
		}
	}

	if len(errs) == 0 {
		return nil
	}

	prefix := fmt.Sprintf("role %q", expectedID)
	if r.ID != "" && r.ID != expectedID {
		prefix = fmt.Sprintf("role file %q (id: %q)", expectedID, r.ID)
	}
	return fmt.Errorf("%s: %s", prefix, strings.Join(errs, "; "))
}
