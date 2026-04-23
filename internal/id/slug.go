package id

import (
	"fmt"
	"strings"
)

// ValidateEpicSlug validates and normalizes an epic slug per ID spec §8.
// Returns the normalized (uppercased) slug and nil on success, or an error
// explaining which rule was violated.
func ValidateEpicSlug(slug string) (string, error) {
	if strings.ContainsRune(slug, ' ') {
		return "", fmt.Errorf("invalid epic slug %q: must not contain spaces", slug)
	}

	normalized := strings.ToUpper(strings.TrimSpace(slug))

	if len(normalized) < 2 {
		return "", fmt.Errorf("invalid epic slug %q: must be at least 2 characters", slug)
	}
	if len(normalized) > 20 {
		return "", fmt.Errorf("invalid epic slug %q: must be at most 20 characters", slug)
	}

	for i, c := range normalized {
		if !((c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-') {
			return "", fmt.Errorf("invalid epic slug %q: invalid character %q at position %d (only A-Z, 0-9, and hyphens allowed)", slug, c, i)
		}
	}

	if strings.HasPrefix(normalized, "-") {
		return "", fmt.Errorf("invalid epic slug %q: must not start with a hyphen", slug)
	}
	if strings.HasSuffix(normalized, "-") {
		return "", fmt.Errorf("invalid epic slug %q: must not end with a hyphen", slug)
	}
	if strings.Contains(normalized, "--") {
		return "", fmt.Errorf("invalid epic slug %q: must not contain consecutive hyphens", slug)
	}

	return normalized, nil
}
