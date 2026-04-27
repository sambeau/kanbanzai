package service

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// planFolderRe matches a work/P{n}-{slug} directory (case-insensitive on plan ID).
// Group 1: plan ID (e.g. "P37" or "p37").
var planFolderRe = regexp.MustCompile(`(?i)^work/(p\d+)-`)

// planFilenamePrefixRe matches a filename that begins with a plan ID prefix P{n}-.
// Group 1: plan ID (e.g. "P37" or "p37").
var planFilenamePrefixRe = regexp.MustCompile(`(?i)^(p\d+)-`)

// featurePfxRe matches the optional F{n}- infix in plan filenames (case-insensitive).
var featurePfxRe = regexp.MustCompile(`(?i)^f\d+-`)

// slugRe matches a valid slug component: at least one [a-z0-9] character with optional
// hyphens interspersed, and not starting or ending with a hyphen.
var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$|^[a-z0-9]$`)

// sortedDocTypes holds the recognised file-prefix types ordered by descending length so
// that longer names (e.g. "dev-plan") are matched before shorter prefixes.
var sortedDocTypes = func() []string {
	types := []string{
		"design", "spec", "dev-plan", "review", "report", "research", "retro", "proposal",
		"policy", "rca",
	}
	sort.Slice(types, func(i, j int) bool { return len(types[i]) > len(types[j]) })
	return types
}()

// validateDocumentFilename checks the file at path has a canonical filename.
// Returns nil if valid, an error with the specific expected pattern if not.
// path is relative to repo root.
func validateDocumentFilename(path string) error {
	path = filepath.ToSlash(path)

	// work/templates/ files are exempt from all validation (REQ-009).
	if strings.HasPrefix(path, "work/templates/") {
		return nil
	}
	// docs/ files have no filename rules (REQ-010).
	if strings.HasPrefix(path, "docs/") {
		return nil
	}

	dir := filepath.ToSlash(filepath.Dir(path))
	filename := filepath.Base(path)

	// work/_project/ — filename must be {type}[-{slug}].{ext}
	if dir == "work/_project" {
		if !matchesProjectFilename(filename) {
			return fmt.Errorf(
				"expected filename to match {type}[-{slug}].{ext} (e.g. spec-{slug}.md or design.md), got %q",
				filename,
			)
		}
		return nil
	}

	// work/P{n}-{slug}/ — extract plan ID from folder, validate filename prefix.
	if m := planFolderRe.FindStringSubmatch(dir); m != nil {
		planID := strings.ToUpper(m[1]) // e.g. "P37"
		if !matchesPlanFilename(filename, planID) {
			return fmt.Errorf(
				"expected filename to match %s-{type}[-{slug}].{ext} or %s-F{n}-{type}[-{slug}].{ext} (e.g. %s-spec-{slug}.md), got %q",
				planID, planID, planID, filename,
			)
		}
		return nil
	}

	// No filename rules for other paths.
	return nil
}

// validateDocumentFolder checks the file is in the folder that corresponds to the plan
// ID (or absence thereof) in its filename.
// Returns nil if valid, an error with the specific expected directory if not.
// path is relative to repo root.
func validateDocumentFolder(path string) error {
	path = filepath.ToSlash(path)

	// work/templates/ files are exempt from all validation (REQ-009).
	if strings.HasPrefix(path, "work/templates/") {
		return nil
	}
	// docs/ files are exempt from folder validation (REQ-010).
	if strings.HasPrefix(path, "docs/") {
		return nil
	}

	dir := filepath.ToSlash(filepath.Dir(path))
	filename := filepath.Base(path)

	// Filename starts with P{n}- → must be in work/P{n}-{slug}/.
	if m := planFilenamePrefixRe.FindStringSubmatch(filename); m != nil {
		planID := strings.ToUpper(m[1]) // e.g. "P37"
		expectedPfxLower := strings.ToLower("work/" + planID + "-")
		if !strings.HasPrefix(strings.ToLower(dir), expectedPfxLower) {
			return fmt.Errorf(
				"expected file to be in work/%s-{slug}/ (filename prefix %s- requires a matching plan folder), found in %s/",
				planID, planID, dir,
			)
		}
		return nil
	}

	// Filename starts with a recognised type → must be in work/_project/.
	if startsWithDocType(filename) {
		if dir != "work/_project" {
			return fmt.Errorf(
				"expected file to be in work/_project/ (type-only filename prefix requires _project folder), found in %s/",
				dir,
			)
		}
		return nil
	}

	// No folder rule applies.
	return nil
}

// matchesProjectFilename returns true if filename matches {type}[-{slug}].{ext}.
func matchesProjectFilename(filename string) bool {
	typeStr, rest := extractTypePrefix(filename)
	if typeStr == "" {
		return false
	}
	return matchesFilenameRemainder(rest)
}

// matchesPlanFilename returns true if filename matches
// {PlanID}-{type}[-{slug}].{ext} or {PlanID}-F{n}-{type}[-{slug}].{ext}
// (case-insensitive on PlanID).
func matchesPlanFilename(filename, planID string) bool {
	lower := strings.ToLower(filename)
	prefix := strings.ToLower(planID) + "-"
	if !strings.HasPrefix(lower, prefix) {
		return false
	}
	rest := filename[len(prefix):]

	// Optionally strip F{n}- infix.
	if featurePfxRe.MatchString(rest) {
		loc := featurePfxRe.FindStringIndex(rest)
		rest = rest[loc[1]:]
	}

	// Remaining part must be {type}[-{slug}].{ext}.
	typeStr, remainder := extractTypePrefix(rest)
	if typeStr == "" {
		return false
	}
	return matchesFilenameRemainder(remainder)
}

// matchesFilenameRemainder returns true if s matches "[-{slug}].{ext}":
// either ".{ext}" or "-{slug}.{ext}" where slug is [a-z0-9][a-z0-9-]*.
func matchesFilenameRemainder(s string) bool {
	if s == "" {
		return false
	}
	if s[0] == '.' {
		return len(s) > 1 // must have a non-empty extension
	}
	if s[0] == '-' {
		inner := s[1:] // slug.ext
		dotIdx := strings.LastIndex(inner, ".")
		if dotIdx < 0 || dotIdx == 0 {
			return false
		}
		slug := inner[:dotIdx]
		ext := inner[dotIdx:]
		return len(ext) > 1 && slugRe.MatchString(slug)
	}
	return false
}

// extractTypePrefix returns the recognised type string if filename starts with one,
// plus the remainder of the string after the type.
// Returns ("", "") if no type prefix matches.
func extractTypePrefix(filename string) (string, string) {
	lower := strings.ToLower(filename)
	for _, t := range sortedDocTypes {
		if strings.HasPrefix(lower, t) {
			rest := filename[len(t):]
			// Must be followed by '-', '.', or end of string.
			if rest == "" || rest[0] == '-' || rest[0] == '.' {
				return t, rest
			}
		}
	}
	return "", ""
}

// startsWithDocType returns true if filename starts with a recognised doc type prefix.
func startsWithDocType(filename string) bool {
	t, _ := extractTypePrefix(filename)
	return t != ""
}
