package knowledge

import (
	"regexp"
	"sort"
	"strings"
)

// entityPatterns are the compiled regex patterns for finding entity ID references in text.
// Order matters: more specific patterns (fixed prefixes) are listed before the generic plan pattern.
var entityPatterns = []*regexp.Regexp{
	regexp.MustCompile(`FEAT-[A-Z0-9]+`),
	regexp.MustCompile(`TASK-[A-Z0-9]+`),
	regexp.MustCompile(`BUG-[A-Z0-9]+`),
	regexp.MustCompile(`DEC-[A-Z0-9]+`),
	regexp.MustCompile(`KE-[A-Z0-9]+`),
	regexp.MustCompile(`[A-Z][0-9]+-[a-z0-9-]+`),
}

// EntityRef is a reference found in text.
type EntityRef struct {
	Span  string
	Start int
	End   int
}

// ScanEntityRefs finds all entity ID patterns in the given text.
// Returns unique spans (by text content) sorted by first occurrence position.
func ScanEntityRefs(text string) []EntityRef {
	type rawMatch struct {
		span  string
		start int
		end   int
	}

	var all []rawMatch
	for _, pat := range entityPatterns {
		for _, loc := range pat.FindAllStringIndex(text, -1) {
			all = append(all, rawMatch{
				span:  text[loc[0]:loc[1]],
				start: loc[0],
				end:   loc[1],
			})
		}
	}

	// Sort by start position so we process left-to-right.
	sort.Slice(all, func(i, j int) bool {
		return all[i].start < all[j].start
	})

	seen := make(map[string]struct{})
	var refs []EntityRef
	for _, m := range all {
		if _, ok := seen[m.span]; ok {
			continue
		}
		seen[m.span] = struct{}{}
		refs = append(refs, EntityRef{
			Span:  m.span,
			Start: m.start,
			End:   m.end,
		})
	}

	return refs
}

// EntityTypeFromID returns a simple entity type name for a given entity ID based on its prefix.
func EntityTypeFromID(id string) string {
	switch {
	case strings.HasPrefix(id, "FEAT-"):
		return "feature"
	case strings.HasPrefix(id, "TASK-"):
		return "task"
	case strings.HasPrefix(id, "BUG-"):
		return "bug"
	case strings.HasPrefix(id, "DEC-"):
		return "decision"
	case strings.HasPrefix(id, "KE-"):
		return "knowledge_entry"
	default:
		return "plan"
	}
}
