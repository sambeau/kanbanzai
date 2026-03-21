package id

import "strings"

// FormatFullDisplay formats an ID in full display form with break hyphen.
// For TSID-based IDs: FEAT-01J3K-7MXP3RT5
// For Epic IDs: returned as-is (EPIC-MYSLUG)
func FormatFullDisplay(canonicalID string) string {
	prefix, tsid, ok := splitCanonicalID(canonicalID)
	if !ok || prefix == "EPIC" {
		return canonicalID
	}
	if len(tsid) <= 5 {
		return canonicalID
	}
	return prefix + "-" + tsid[:5] + "-" + tsid[5:]
}

// FormatShortDisplay formats an ID in short display form using the shortest
// unique prefix. The ids slice should contain all canonical IDs of the same
// entity type (used to compute uniqueness).
func FormatShortDisplay(canonicalID string, sameTypeIDs []string) string {
	prefix, tsid, ok := splitCanonicalID(canonicalID)
	if !ok || prefix == "EPIC" {
		return canonicalID
	}

	minLen := ShortestUniquePrefix(tsid, extractTSIDs(prefix, sameTypeIDs))
	if minLen <= 5 {
		// Show at least through the break point
		return prefix + "-" + tsid[:5]
	}
	return prefix + "-" + tsid[:5] + "-" + tsid[5:minLen]
}

// ShortestUniquePrefix returns the minimum prefix length of target that
// uniquely distinguishes it from all other TSIDs. Minimum return value is 5.
func ShortestUniquePrefix(target string, otherTSIDs []string) int {
	minLen := 5
	for _, other := range otherTSIDs {
		if other == target {
			continue
		}
		// Find the length where they first differ
		common := 0
		limit := len(target)
		if len(other) < limit {
			limit = len(other)
		}
		for common < limit && target[common] == other[common] {
			common++
		}
		needed := common + 1
		if needed > minLen {
			minLen = needed
		}
	}
	if minLen > len(target) {
		minLen = len(target)
	}
	return minLen
}

// StripBreakHyphens removes display break hyphens from an ID, returning
// the canonical form. It normalizes to uppercase.
func StripBreakHyphens(input string) string {
	upper := strings.ToUpper(strings.TrimSpace(input))

	prefix, rest, ok := splitCanonicalID(upper)
	if !ok {
		// Try to parse as a TSID-type ID with an extra hyphen
		// e.g., "FEAT-01J3K-7MXP3RT5" -> find the type prefix, then rejoin
		for _, p := range []string{"EPIC", "FEAT", "BUG", "DEC", "TASK", "DOC"} {
			pfx := p + "-"
			if strings.HasPrefix(upper, pfx) {
				after := upper[len(pfx):]
				// Remove all hyphens from the TSID portion
				cleaned := strings.ReplaceAll(after, "-", "")
				return p + "-" + cleaned
			}
		}
		return upper
	}

	if prefix == "EPIC" {
		return upper
	}

	// For TSID types, remove any hyphens within the TSID portion
	cleaned := strings.ReplaceAll(rest, "-", "")
	return prefix + "-" + cleaned
}

// splitCanonicalID splits a canonical ID into its type prefix and the rest.
// Returns ("FEAT", "01J3K7MXP3RT5", true) for "FEAT-01J3K7MXP3RT5".
func splitCanonicalID(id string) (prefix, rest string, ok bool) {
	idx := strings.Index(id, "-")
	if idx <= 0 || idx >= len(id)-1 {
		return "", "", false
	}
	return id[:idx], id[idx+1:], true
}

// extractTSIDs extracts the TSID portions from canonical IDs that match the given prefix.
func extractTSIDs(prefix string, canonicalIDs []string) []string {
	pfx := prefix + "-"
	var tsids []string
	for _, id := range canonicalIDs {
		if strings.HasPrefix(id, pfx) {
			tsids = append(tsids, id[len(pfx):])
		}
	}
	return tsids
}
