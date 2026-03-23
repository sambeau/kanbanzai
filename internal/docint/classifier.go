package docint

import (
	"fmt"
)

// ValidateClassifications validates a classification submission against
// the document's current index. It checks:
// 1. The document content hash matches (document hasn't changed)
// 2. Every classified section exists in the index
// 3. All roles are from the taxonomy
// 4. All confidence levels are valid
// 5. Model name and version are provided
func ValidateClassifications(index *DocumentIndex, submission ClassificationSubmission) []error {
	var errs []error

	// Check content hash
	if submission.ContentHash != index.ContentHash {
		errs = append(errs, fmt.Errorf("content hash mismatch: document has changed since skeleton was retrieved (expected %s, got %s)", index.ContentHash, submission.ContentHash))
	}

	// Check model provenance
	if submission.ModelName == "" {
		errs = append(errs, fmt.Errorf("model_name is required"))
	}
	if submission.ModelVersion == "" {
		errs = append(errs, fmt.Errorf("model_version is required"))
	}

	// Build set of valid section paths from index
	validPaths := collectSectionPaths(index.Sections)

	// Validate each classification
	for i, c := range submission.Classifications {
		// Check section exists
		if _, ok := validPaths[c.SectionPath]; !ok {
			errs = append(errs, fmt.Errorf("classification[%d]: unknown section_path %q", i, c.SectionPath))
		}

		// Validate against taxonomy
		if err := ValidateClassification(c); err != nil {
			errs = append(errs, fmt.Errorf("classification[%d]: %w", i, err))
		}
	}

	return errs
}

// ApplyClassifications applies validated classifications to a document index.
// Classifications are immutable once applied — this overwrites any existing.
func ApplyClassifications(index *DocumentIndex, submission ClassificationSubmission) {
	index.Classifications = submission.Classifications
	index.Classified = true
	index.ClassifiedAt = &submission.ClassifiedAt
	index.ClassifiedBy = submission.ModelName
	index.ClassifierVersion = submission.ModelVersion
}

// collectSectionPaths returns all section paths in the tree.
func collectSectionPaths(sections []Section) map[string]struct{} {
	paths := make(map[string]struct{})
	var walk func([]Section)
	walk = func(ss []Section) {
		for _, s := range ss {
			paths[s.Path] = struct{}{}
			walk(s.Children)
		}
	}
	walk(sections)
	return paths
}
