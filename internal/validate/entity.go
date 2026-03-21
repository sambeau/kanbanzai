package validate

import (
	"fmt"
	"regexp"
	"strings"

	"kanbanzai/internal/id"
	"kanbanzai/internal/model"
)

// ValidationError describes a single field-level validation failure.
type ValidationError struct {
	EntityType string
	EntityID   string // best-effort, may be empty if ID is missing
	Field      string
	Message    string
}

// Error implements the error interface.
func (e ValidationError) Error() string {
	if e.EntityID != "" {
		return fmt.Sprintf("%s %s: %s: %s", e.EntityType, e.EntityID, e.Field, e.Message)
	}
	return fmt.Sprintf("%s: %s: %s", e.EntityType, e.Field, e.Message)
}

var validBugSeverities = map[string]struct{}{
	string(model.BugSeverityLow):      {},
	string(model.BugSeverityMedium):   {},
	string(model.BugSeverityHigh):     {},
	string(model.BugSeverityCritical): {},
}

var validBugPriorities = map[string]struct{}{
	string(model.BugPriorityLow):      {},
	string(model.BugPriorityMedium):   {},
	string(model.BugPriorityHigh):     {},
	string(model.BugPriorityCritical): {},
}

var validBugTypes = map[string]struct{}{
	string(model.BugTypeImplementationDefect): {},
	string(model.BugTypeSpecificationDefect):  {},
	string(model.BugTypeDesignProblem):        {},
}

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// ValidateBugSeverity returns an error if value is not a valid bug severity.
func ValidateBugSeverity(value string) error {
	if _, ok := validBugSeverities[value]; !ok {
		return fmt.Errorf("invalid bug severity %q: must be one of low, medium, high, critical", value)
	}
	return nil
}

// ValidateBugPriority returns an error if value is not a valid bug priority.
func ValidateBugPriority(value string) error {
	if _, ok := validBugPriorities[value]; !ok {
		return fmt.Errorf("invalid bug priority %q: must be one of low, medium, high, critical", value)
	}
	return nil
}

// ValidateBugType returns an error if value is not a valid bug type.
func ValidateBugType(value string) error {
	if _, ok := validBugTypes[value]; !ok {
		return fmt.Errorf("invalid bug type %q: must be one of implementation-defect, specification-defect, design-problem", value)
	}
	return nil
}

// ValidateSlug returns an error if slug is not a non-empty lowercase kebab-case value.
func ValidateSlug(slug string) error {
	trimmed := strings.TrimSpace(slug)
	if trimmed == "" {
		return fmt.Errorf("invalid slug %q: must not be empty", slug)
	}
	if trimmed != slug {
		return fmt.Errorf("invalid slug %q: must not contain leading or trailing whitespace", slug)
	}
	if strings.ContainsAny(slug, `/\`) {
		return fmt.Errorf("invalid slug %q: must not contain path separators", slug)
	}
	if !slugPattern.MatchString(slug) {
		return fmt.Errorf("invalid slug %q: must be lowercase kebab-case using only letters, numbers, and hyphens", slug)
	}
	return nil
}

var requiredFields = map[EntityKind][]string{
	EntityEpic:     {"id", "slug", "title", "status", "summary", "created", "created_by"},
	EntityFeature:  {"id", "slug", "epic", "status", "summary", "created", "created_by"},
	EntityTask:     {"id", "parent_feature", "slug", "summary", "status"},
	EntityBug:      {"id", "slug", "title", "status", "severity", "priority", "type", "reported_by", "reported", "observed", "expected"},
	EntityDecision: {"id", "slug", "summary", "rationale", "decided_by", "date", "status"},
}

// ValidateRecord checks an entity record's fields for correctness.
// It validates:
//   - required fields are present and non-empty
//   - slug format is valid
//   - enum fields contain valid values
//   - status is a known lifecycle state
//   - ID format is valid for the entity type
func ValidateRecord(entityType string, fields map[string]any) []ValidationError {
	kind := EntityKind(entityType)

	required, ok := requiredFields[kind]
	if !ok {
		return []ValidationError{{
			EntityType: entityType,
			Field:      "type",
			Message:    fmt.Sprintf("unknown entity type %q", entityType),
		}}
	}

	entityID, _ := stringField(fields, "id")

	var errs []ValidationError

	for _, field := range required {
		val, ok := fields[field]
		if !ok {
			errs = append(errs, ValidationError{
				EntityType: entityType,
				EntityID:   entityID,
				Field:      field,
				Message:    "required field is missing",
			})
			continue
		}
		if isEmpty(val) {
			errs = append(errs, ValidationError{
				EntityType: entityType,
				EntityID:   entityID,
				Field:      field,
				Message:    "required field is empty",
			})
		}
	}

	if slug, ok := stringField(fields, "slug"); ok && slug != "" {
		if err := ValidateSlug(slug); err != nil {
			errs = append(errs, ValidationError{
				EntityType: entityType,
				EntityID:   entityID,
				Field:      "slug",
				Message:    err.Error(),
			})
		}
	}

	if entityID != "" {
		allocator := id.NewAllocator()
		if err := allocator.Validate(model.EntityKind(kind), entityID); err != nil {
			errs = append(errs, ValidationError{
				EntityType: entityType,
				EntityID:   entityID,
				Field:      "id",
				Message:    err.Error(),
			})
		}
	}

	if status, ok := stringField(fields, "status"); ok && status != "" {
		if !IsKnownState(kind, status) {
			errs = append(errs, ValidationError{
				EntityType: entityType,
				EntityID:   entityID,
				Field:      "status",
				Message:    fmt.Sprintf("unknown %s status %q", entityType, status),
			})
		}
	}

	if kind == EntityBug {
		if severity, ok := stringField(fields, "severity"); ok && severity != "" {
			if err := ValidateBugSeverity(severity); err != nil {
				errs = append(errs, ValidationError{
					EntityType: entityType,
					EntityID:   entityID,
					Field:      "severity",
					Message:    err.Error(),
				})
			}
		}
		if priority, ok := stringField(fields, "priority"); ok && priority != "" {
			if err := ValidateBugPriority(priority); err != nil {
				errs = append(errs, ValidationError{
					EntityType: entityType,
					EntityID:   entityID,
					Field:      "priority",
					Message:    err.Error(),
				})
			}
		}
		if bugType, ok := stringField(fields, "type"); ok && bugType != "" {
			if err := ValidateBugType(bugType); err != nil {
				errs = append(errs, ValidationError{
					EntityType: entityType,
					EntityID:   entityID,
					Field:      "type",
					Message:    err.Error(),
				})
			}
		}
	}

	return errs
}

// ValidateEntityExists checks that a referenced entity exists.
// entityType is the type of the referenced entity (e.g., "epic").
// id is the ID to check.
// exists is a function that returns true if the entity exists.
func ValidateEntityExists(entityType, id string, exists func(entityType, id string) bool) *ValidationError {
	if exists(entityType, id) {
		return nil
	}
	return &ValidationError{
		EntityType: entityType,
		EntityID:   id,
		Field:      "id",
		Message:    fmt.Sprintf("%s %q does not exist", entityType, id),
	}
}

// stringField extracts a string value from a map field. It returns the value
// and true if the field exists and is a string.
func stringField(fields map[string]any, key string) (string, bool) {
	val, ok := fields[key]
	if !ok {
		return "", false
	}
	s, ok := val.(string)
	return s, ok
}

// isEmpty reports whether a field value is considered empty. A nil value, an
// empty string, or a zero-value time are all considered empty.
func isEmpty(val any) bool {
	if val == nil {
		return true
	}
	if s, ok := val.(string); ok && s == "" {
		return true
	}
	return false
}
