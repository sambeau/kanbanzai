package validate

import (
	"fmt"

	"kanbanzai/internal/id"
	"kanbanzai/internal/model"
)

// ValidationWarning is a non-blocking issue found during health checks.
type ValidationWarning struct {
	EntityType string
	EntityID   string
	Field      string
	Message    string
}

// Error formats the warning as a human-readable string.
func (w ValidationWarning) Error() string {
	if w.EntityID != "" {
		return fmt.Sprintf("warning: %s %s: %s: %s", w.EntityType, w.EntityID, w.Field, w.Message)
	}
	return fmt.Sprintf("warning: %s: %s: %s", w.EntityType, w.Field, w.Message)
}

// HealthSummary provides aggregate counts from a health check.
type HealthSummary struct {
	TotalEntities  int
	ErrorCount     int
	WarningCount   int
	EntitiesByType map[string]int
}

// HealthReport summarizes the results of a project-wide health check.
type HealthReport struct {
	Errors   []ValidationError
	Warnings []ValidationWarning
	Summary  HealthSummary
}

// EntityInfo holds the type, ID, and fields of a single entity for health checking.
type EntityInfo struct {
	Type   string
	ID     string
	Fields map[string]any
}

// entityKey identifies an entity by type and ID for internal lookups.
type entityKey struct {
	typ string
	id  string
}

// CheckHealth runs a comprehensive health check across all entities.
// loadAll returns all entities across all types.
// entityExists checks whether a specific entity of a given type and ID exists.
func CheckHealth(loadAll func() ([]EntityInfo, error), entityExists func(entityType, id string) bool) (*HealthReport, error) {
	allocator := id.NewAllocator()
	entities, err := loadAll()
	if err != nil {
		return nil, fmt.Errorf("loading entities: %w", err)
	}

	report := &HealthReport{
		Summary: HealthSummary{
			EntitiesByType: make(map[string]int),
		},
	}

	// Build a lookup of all entity fields for supersession consistency checks.
	fieldsByKey := make(map[entityKey]map[string]any, len(entities))
	for _, e := range entities {
		fieldsByKey[entityKey{e.Type, e.ID}] = e.Fields
	}

	for _, e := range entities {
		report.Summary.TotalEntities++
		report.Summary.EntitiesByType[e.Type]++

		// Field validation.
		errs := ValidateRecord(e.Type, e.Fields)
		report.Errors = append(report.Errors, errs...)

		if err := allocator.Validate(model.EntityKind(e.Type), e.ID); err != nil {
			report.Errors = append(report.Errors, ValidationError{
				EntityType: e.Type,
				EntityID:   e.ID,
				Field:      "id",
				Message:    err.Error(),
			})
		}

		// Cross-reference checks.
		checkRef := func(field, targetType, targetID string) {
			if targetID == "" {
				return
			}
			if !entityExists(targetType, targetID) {
				report.Errors = append(report.Errors, ValidationError{
					EntityType: e.Type,
					EntityID:   e.ID,
					Field:      field,
					Message:    fmt.Sprintf("references non-existent %s %q", targetType, targetID),
				})
			}
		}

		checkRefSlice := func(field, targetType string, ids []string) {
			for _, id := range ids {
				if id == "" {
					continue
				}
				if !entityExists(targetType, id) {
					report.Errors = append(report.Errors, ValidationError{
						EntityType: e.Type,
						EntityID:   e.ID,
						Field:      field,
						Message:    fmt.Sprintf("references non-existent %s %q", targetType, id),
					})
				}
			}
		}

		switch e.Type {
		case string(EntityEpic):
			checkRefSlice("features", string(EntityFeature), toStringSlice(e.Fields["features"]))

		case string(EntityFeature):
			checkRef("epic", string(EntityEpic), toString(e.Fields["epic"]))
			checkRef("supersedes", string(EntityFeature), toString(e.Fields["supersedes"]))
			checkRef("superseded_by", string(EntityFeature), toString(e.Fields["superseded_by"]))
			checkRefSlice("tasks", string(EntityTask), toStringSlice(e.Fields["tasks"]))
			checkRefSlice("decisions", string(EntityDecision), toStringSlice(e.Fields["decisions"]))

		case string(EntityTask):
			checkRef("parent_feature", string(EntityFeature), toString(e.Fields["parent_feature"]))
			checkRefSlice("depends_on", string(EntityTask), toStringSlice(e.Fields["depends_on"]))

		case string(EntityBug):
			checkRef("origin_feature", string(EntityFeature), toString(e.Fields["origin_feature"]))
			checkRef("origin_task", string(EntityTask), toString(e.Fields["origin_task"]))
			checkRef("duplicate_of", string(EntityBug), toString(e.Fields["duplicate_of"]))

		case string(EntityDecision):
			checkRef("supersedes", string(EntityDecision), toString(e.Fields["supersedes"]))
			checkRef("superseded_by", string(EntityDecision), toString(e.Fields["superseded_by"]))
		}

		// Supersession mutual consistency checks.
		checkSupersessionConsistency(e, fieldsByKey, entityExists, report)
	}

	report.Summary.ErrorCount = len(report.Errors)
	report.Summary.WarningCount = len(report.Warnings)

	return report, nil
}

// checkSupersessionConsistency warns if supersedes/superseded_by fields are not
// mutually consistent between two entities. Only applies to feature and decision.
func checkSupersessionConsistency(
	e EntityInfo,
	fieldsByKey map[entityKey]map[string]any,
	entityExists func(string, string) bool,
	report *HealthReport,
) {
	if e.Type != string(EntityFeature) && e.Type != string(EntityDecision) {
		return
	}

	supersedes := toString(e.Fields["supersedes"])
	if supersedes != "" && entityExists(e.Type, supersedes) {
		otherFields := fieldsByKey[entityKey{e.Type, supersedes}]
		if otherFields != nil {
			otherSupersededBy := toString(otherFields["superseded_by"])
			if otherSupersededBy != e.ID {
				report.Warnings = append(report.Warnings, ValidationWarning{
					EntityType: e.Type,
					EntityID:   e.ID,
					Field:      "supersedes",
					Message: fmt.Sprintf(
						"%s %s supersedes %s, but %s does not have superseded_by = %s",
						e.Type, e.ID, supersedes, supersedes, e.ID,
					),
				})
			}
		}
	}

	supersededBy := toString(e.Fields["superseded_by"])
	if supersededBy != "" && entityExists(e.Type, supersededBy) {
		otherFields := fieldsByKey[entityKey{e.Type, supersededBy}]
		if otherFields != nil {
			otherSupersedes := toString(otherFields["supersedes"])
			if otherSupersedes != e.ID {
				report.Warnings = append(report.Warnings, ValidationWarning{
					EntityType: e.Type,
					EntityID:   e.ID,
					Field:      "superseded_by",
					Message: fmt.Sprintf(
						"%s %s is superseded by %s, but %s does not have supersedes = %s",
						e.Type, e.ID, supersededBy, supersededBy, e.ID,
					),
				})
			}
		}
	}
}

// toString extracts a string from an any value, returning "" if nil or not a string.
func toString(v any) string {
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

// toStringSlice extracts a []string from an any value, returning nil if not convertible.
func toStringSlice(v any) []string {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case []string:
		return val
	case []any:
		result := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	default:
		return nil
	}
}
