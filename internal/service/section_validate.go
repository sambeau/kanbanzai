// Package service section_validate.go — document section validation for
// Kanbanzai workflow automation (FEAT-01KN73BFK4M4Z, Pillar D).
//
// ValidateSections checks a markdown file for the presence of required
// level-2 headings declared in stage-bindings.yaml. It is called by
// SubmitDocument (warnings, FR-D04) and ApproveDocument (hard error, FR-D05).
package service

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// SectionValidationResult holds the outcome of a section validation check.
type SectionValidationResult struct {
	// Found lists the required section names that were present in the file.
	Found []string
	// Missing lists the required section names that were absent.
	Missing []string
	// Valid is true when Missing is empty.
	Valid bool
}

// ValidateSections checks filePath for the presence of each name in
// requiredSections by scanning for level-2 markdown headings ("## ...").
// Matching is case-insensitive (FR-D03). Level-1 and level-3+ headings
// are ignored (FR-D03). If requiredSections is empty, validation passes
// unconditionally (FR-D06).
//
// Returns an error only when the file cannot be read; a document with
// missing sections returns a result with Valid == false, not an error.
func ValidateSections(filePath string, requiredSections []string) (SectionValidationResult, error) {
	if len(requiredSections) == 0 {
		return SectionValidationResult{Valid: true}, nil
	}

	f, err := os.Open(filePath)
	if err != nil {
		return SectionValidationResult{}, fmt.Errorf("open file for section validation: %w", err)
	}
	defer f.Close()

	// Build a set of required section names in lower-case for O(1) lookup.
	required := make(map[string]string, len(requiredSections)) // lower → original
	for _, s := range requiredSections {
		required[strings.ToLower(strings.TrimSpace(s))] = strings.TrimSpace(s)
	}

	// Scan the file line by line for level-2 headings ("## ...").
	// Lines starting with exactly "## " (two hashes and a space) are level-2.
	// "### " or "# " are other levels and must NOT match (FR-D03).
	found := make(map[string]bool, len(requiredSections))
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "## ") {
			continue
		}
		heading := strings.TrimSpace(strings.TrimPrefix(line, "## "))
		lowerHeading := strings.ToLower(heading)
		if _, ok := required[lowerHeading]; ok {
			found[lowerHeading] = true
		}
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return SectionValidationResult{}, fmt.Errorf("read file for section validation: %w", scanErr)
	}

	var foundList []string
	var missing []string
	for lower, original := range required {
		if found[lower] {
			foundList = append(foundList, original)
		} else {
			missing = append(missing, original)
		}
	}

	return SectionValidationResult{
		Found:   foundList,
		Missing: missing,
		Valid:   len(missing) == 0,
	}, nil
}
