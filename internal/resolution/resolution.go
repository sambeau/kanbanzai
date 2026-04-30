// Package resolution provides lexical target disambiguation for the kbz CLI.
//
// The Disambiguate function classifies a target string as a file path, entity ID,
// bare plan prefix, or none of the above using purely lexical rules (no I/O).
// This implements NFR-001: the disambiguation logic must execute without any
// file reads, state store access, or network calls before determining the
// resolution strategy.
package resolution

import (
	"regexp"
	"strings"
)

// ResolutionKind classifies what kind of target a string represents.
type ResolutionKind int

const (
	// ResolvePath indicates the target is a file path — it contains '/' or ends
	// with .md or .txt. The caller should perform file path resolution.
	ResolvePath ResolutionKind = iota

	// ResolveEntity indicates the target matches a known entity ID pattern.
	// The caller should perform entity ID lookups.
	ResolveEntity

	// ResolvePlanPrefix indicates the target is a bare plan prefix (e.g. "P1")
	// that should be expanded to a full plan ID by the caller.
	ResolvePlanPrefix

	// ResolveNone indicates the target does not match any known pattern.
	// The caller should probe entity then path, then error if both fail.
	ResolveNone
)

// String returns a human-readable representation of the resolution kind.
func (k ResolutionKind) String() string {
	switch k {
	case ResolvePath:
		return "ResolvePath"
	case ResolveEntity:
		return "ResolveEntity"
	case ResolvePlanPrefix:
		return "ResolvePlanPrefix"
	case ResolveNone:
		return "ResolveNone"
	default:
		return "unknown"
	}
}

// knownEntityPrefixes are the ID prefixes recognised by the entity system.
var knownEntityPrefixes = []string{
	"FEAT-",
	"TASK-",
	"T-",
	"BUG-",
	"INC-",
}

// barePlanPrefixRE matches a bare plan prefix: a single uppercase letter
// followed by one or more digits, with nothing else.
//
// Examples: "P1", "P42", "B7"
// Non-examples: "P1-my-plan" (has slug), "p1" (lowercase), "P" (no digits)
var barePlanPrefixRE = regexp.MustCompile(`^[A-Z][0-9]+$`)

// batchPrefixRE matches a batch/plan ID with slug: a single uppercase letter,
// digits, hyphen, then slug. E.g. "P1-my-plan", "B24-auth-system".
var batchPrefixRE = regexp.MustCompile(`^[A-Z][0-9]+-.+$`)

// entityDisplayFormatRE matches display-format entity IDs like "FEAT-042",
// "TASK-001", "BUG-007", "INC-003".
var entityDisplayFormatRE = regexp.MustCompile(`^(FEAT|TASK|BUG|INC)-[0-9]+$`)

// entityFullFormatRE matches full TSID13-based entity IDs.
// E.g. "FEAT-01KMKA278DFNV", "TASK-01KMKA278DFNV", etc.
var entityFullFormatRE = regexp.MustCompile(`^(FEAT|TASK|T|BUG|INC)-[A-Z0-9]{13,}$`)

// Disambiguate classifies target into one of four resolution kinds using
// purely lexical rules. It makes no I/O calls — no file reads, no state
// store access, no network calls (NFR-001).
//
// Rules applied in order (per design §5.1):
//  1. If target contains '/' or ends in .md or .txt → ResolvePath
//  2. If target matches a known entity ID pattern → ResolveEntity
//  3. If target matches a bare plan prefix pattern → ResolvePlanPrefix
//  4. Otherwise → ResolveNone
func Disambiguate(target string) ResolutionKind {
	if target == "" {
		return ResolveNone
	}

	// Rule 1: File path detection.
	if strings.Contains(target, "/") ||
		strings.HasSuffix(target, ".md") ||
		strings.HasSuffix(target, ".txt") {
		return ResolvePath
	}

	// Rule 2: Entity ID pattern detection.
	if isEntityID(target) {
		return ResolveEntity
	}

	// Rule 3: Bare plan prefix detection.
	if barePlanPrefixRE.MatchString(target) {
		return ResolvePlanPrefix
	}

	// Rule 4: Fallthrough — no pattern matched.
	return ResolveNone
}

// isEntityID reports whether target matches any known entity ID pattern.
func isEntityID(target string) bool {
	upper := strings.ToUpper(target)
	for _, prefix := range knownEntityPrefixes {
		if strings.HasPrefix(upper, prefix) {
			rest := target[len(prefix):]
			if len(rest) > 0 {
				return true
			}
		}
	}

	if entityDisplayFormatRE.MatchString(upper) {
		return true
	}

	if batchPrefixRE.MatchString(target) {
		return true
	}

	if entityFullFormatRE.MatchString(upper) {
		return true
	}

	return false
}
