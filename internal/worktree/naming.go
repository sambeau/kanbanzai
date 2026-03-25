package worktree

import (
	"regexp"
	"strings"
)

// slugRegexp matches non-alphanumeric characters for slug normalization.
var slugRegexp = regexp.MustCompile(`[^a-z0-9]+`)

// GenerateBranchName creates a branch name from an entity ID and optional slug.
// For feature entities (FEAT-*), it returns "feature/FEAT-01JX-slug".
// For bug entities (BUG-*), it returns "bugfix/BUG-01JX-slug".
// The slug is normalized to lowercase and non-alphanumeric characters are replaced with hyphens.
func GenerateBranchName(entityID string, slug string) string {
	prefix := branchPrefix(entityID)
	name := entityID
	if slug != "" {
		name = entityID + "-" + normalizeSlug(slug)
	}
	return prefix + "/" + name
}

// GenerateWorktreePath creates the worktree path under .worktrees/
// Format: .worktrees/FEAT-01JX-slug or .worktrees/BUG-01JX-slug
// The slug is normalized to lowercase and non-alphanumeric characters are replaced with hyphens.
func GenerateWorktreePath(entityID string, slug string) string {
	name := entityID
	if slug != "" {
		name = entityID + "-" + normalizeSlug(slug)
	}
	return ".worktrees/" + name
}

// branchPrefix returns the branch prefix based on the entity type.
func branchPrefix(entityID string) string {
	upper := strings.ToUpper(entityID)
	if strings.HasPrefix(upper, "BUG-") {
		return "bugfix"
	}
	// Default to feature for FEAT- and any other entity types
	return "feature"
}

// normalizeSlug converts a human-readable slug to a branch-safe format.
// It lowercases the input and replaces non-alphanumeric characters with hyphens.
// Leading/trailing hyphens are removed, and multiple consecutive hyphens are collapsed.
func normalizeSlug(slug string) string {
	// Lowercase
	s := strings.ToLower(slug)

	// Replace non-alphanumeric with hyphens
	s = slugRegexp.ReplaceAllString(s, "-")

	// Trim leading/trailing hyphens
	s = strings.Trim(s, "-")

	// Collapse multiple consecutive hyphens
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}

	return s
}
