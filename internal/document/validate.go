package document

import (
	"fmt"
	"strings"
)

// ValidationError represents a document validation failure.
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return e.Message
}

// ValidateDocument validates a document's metadata and structure against its template.
// Returns a list of validation errors (empty if valid).
func ValidateDocument(doc Document) []ValidationError {
	var errs []ValidationError

	// Validate type
	if !ValidDocType(string(doc.Meta.Type)) {
		errs = append(errs, ValidationError{
			Field:   "type",
			Message: fmt.Sprintf("unrecognised document type: %s", doc.Meta.Type),
		})
		return errs // can't validate further without a valid type
	}

	// Validate required frontmatter fields
	if strings.TrimSpace(doc.Meta.ID) == "" {
		errs = append(errs, ValidationError{Field: "id", Message: "required"})
	}
	if strings.TrimSpace(doc.Meta.Title) == "" {
		errs = append(errs, ValidationError{Field: "title", Message: "required"})
	}
	if strings.TrimSpace(string(doc.Meta.Status)) == "" {
		errs = append(errs, ValidationError{Field: "status", Message: "required"})
	} else if !validDocStatus(string(doc.Meta.Status)) {
		errs = append(errs, ValidationError{
			Field:   "status",
			Message: fmt.Sprintf("invalid status: %s", doc.Meta.Status),
		})
	}
	if strings.TrimSpace(doc.Meta.CreatedBy) == "" {
		errs = append(errs, ValidationError{Field: "created_by", Message: "required"})
	}
	if doc.Meta.Created.IsZero() {
		errs = append(errs, ValidationError{Field: "created", Message: "required"})
	}
	if doc.Meta.Updated.IsZero() {
		errs = append(errs, ValidationError{Field: "updated", Message: "required"})
	}

	// Validate feature reference for types that require it
	if requiresFeatureRef(doc.Meta.Type) && strings.TrimSpace(doc.Meta.Feature) == "" {
		errs = append(errs, ValidationError{
			Field:   "feature",
			Message: fmt.Sprintf("feature reference required for %s documents", doc.Meta.Type),
		})
	}

	// Validate approved documents have approval metadata
	if doc.Meta.Status == DocStatusApproved {
		if strings.TrimSpace(doc.Meta.ApprovedBy) == "" {
			errs = append(errs, ValidationError{Field: "approved_by", Message: "required for approved documents"})
		}
		if doc.Meta.ApprovedAt == nil {
			errs = append(errs, ValidationError{Field: "approved_at", Message: "required for approved documents"})
		}
	}

	// Validate body is not empty
	if strings.TrimSpace(doc.Body) == "" {
		errs = append(errs, ValidationError{Field: "body", Message: "document body is required"})
	}

	// Validate required sections from template
	sectionErrs := validateRequiredSections(doc)
	errs = append(errs, sectionErrs...)

	return errs
}

// validateRequiredSections checks that the document body contains all required
// sections defined by its template.
func validateRequiredSections(doc Document) []ValidationError {
	tmpl, err := GetTemplate(doc.Meta.Type)
	if err != nil {
		return nil // already validated type above
	}

	headings := extractHeadings(doc.Body)
	headingSet := make(map[string]bool, len(headings))
	for _, h := range headings {
		headingSet[strings.ToLower(h)] = true
	}

	var errs []ValidationError
	for _, required := range tmpl.RequiredSections {
		if !headingSet[strings.ToLower(required)] {
			errs = append(errs, ValidationError{
				Field:   "body",
				Message: fmt.Sprintf("missing required section: %s", required),
			})
		}
	}

	return errs
}

// extractHeadings extracts markdown headings from document body.
// Returns heading text without the leading # characters.
func extractHeadings(body string) []string {
	var headings []string
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			// Strip leading # characters and spaces
			heading := strings.TrimLeft(trimmed, "#")
			heading = strings.TrimSpace(heading)
			if heading != "" {
				headings = append(headings, heading)
			}
		}
	}
	return headings
}

// validDocStatus returns true if the given string is a valid document status.
func validDocStatus(s string) bool {
	switch DocStatus(s) {
	case DocStatusDraft, DocStatusSubmitted, DocStatusNormalised, DocStatusApproved:
		return true
	}
	return false
}

// requiresFeatureRef returns true if the document type requires a feature reference.
// Design, specification, and implementation-plan documents must link to a feature.
func requiresFeatureRef(dt DocType) bool {
	switch dt {
	case DocTypeDesign, DocTypeSpecification, DocTypeImplementationPlan:
		return true
	}
	return false
}

// ValidDocTransition returns true if the status transition is valid.
// Document lifecycle: draft → submitted → normalised → approved
func ValidDocTransition(from, to DocStatus) bool {
	switch from {
	case DocStatusDraft:
		return to == DocStatusSubmitted
	case DocStatusSubmitted:
		return to == DocStatusNormalised
	case DocStatusNormalised:
		return to == DocStatusApproved
	case DocStatusApproved:
		return false // approved is terminal
	}
	return false
}
