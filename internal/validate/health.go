package validate

import (
	"fmt"
	"os"
	"strings"

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
			parent := toString(e.Fields["parent"])
			if parent != "" {
				if model.IsPlanID(parent) {
					checkRef("parent", string(EntityPlan), parent)
				} else {
					checkRef("parent", string(EntityEpic), parent)
				}
			}
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

// DocumentInfo holds the fields of a single document record for health checking.
type DocumentInfo struct {
	ID     string
	Fields map[string]any
}

// CheckDocumentHealth runs health checks specific to document records.
// loadAllDocs returns all document records.
// entityExists checks whether a specific entity of a given type and ID exists.
// checkContentHash verifies that the recorded hash matches the file on disk;
// it returns (matches, error). If the file doesn't exist the caller should
// return an error.
func CheckDocumentHealth(
	loadAllDocs func() ([]DocumentInfo, error),
	entityExists func(entityType, entityID string) bool,
	checkContentHash func(docPath, recordedHash string) (bool, error),
) (*HealthReport, error) {
	docs, err := loadAllDocs()
	if err != nil {
		return nil, fmt.Errorf("loading documents: %w", err)
	}

	report := &HealthReport{
		Summary: HealthSummary{
			EntitiesByType: make(map[string]int),
		},
	}

	for _, d := range docs {
		report.Summary.TotalEntities++
		report.Summary.EntitiesByType["document"]++

		docPath := toString(d.Fields["path"])
		status := toString(d.Fields["status"])
		owner := toString(d.Fields["owner"])

		// Check 1: Document file must exist.
		if docPath != "" {
			if _, statErr := os.Stat(docPath); statErr != nil {
				report.Errors = append(report.Errors, ValidationError{
					EntityType: "document",
					EntityID:   d.ID,
					Field:      "path",
					Message:    fmt.Sprintf("document file not found: %s", docPath),
				})
			}
		}

		// Check 2: Content hash must match file on disk.
		recordedHash := toString(d.Fields["content_hash"])
		if docPath != "" && recordedHash != "" && checkContentHash != nil {
			matches, hashErr := checkContentHash(docPath, recordedHash)
			if hashErr != nil {
				// File probably doesn't exist — already reported above.
			} else if !matches {
				report.Warnings = append(report.Warnings, ValidationWarning{
					EntityType: "document",
					EntityID:   d.ID,
					Field:      "content_hash",
					Message:    "content hash does not match file on disk (content drift)",
				})
			}
		}

		// Check 3: Orphaned document — owner entity must exist.
		if owner != "" {
			ownerType := inferEntityType(owner)
			if ownerType != "" && !entityExists(ownerType, owner) {
				report.Errors = append(report.Errors, ValidationError{
					EntityType: "document",
					EntityID:   d.ID,
					Field:      "owner",
					Message:    fmt.Sprintf("owner entity not found: %s %s", ownerType, owner),
				})
			}
		}

		// Check 4: Approved documents must have approved_by and approved_at.
		if status == string(model.DocumentStatusApproved) {
			if toString(d.Fields["approved_by"]) == "" {
				report.Errors = append(report.Errors, ValidationError{
					EntityType: "document",
					EntityID:   d.ID,
					Field:      "approved_by",
					Message:    "approved document missing approved_by",
				})
			}
			if toString(d.Fields["approved_at"]) == "" {
				report.Errors = append(report.Errors, ValidationError{
					EntityType: "document",
					EntityID:   d.ID,
					Field:      "approved_at",
					Message:    "approved document missing approved_at",
				})
			}
		}
	}

	report.Summary.ErrorCount = len(report.Errors)
	report.Summary.WarningCount = len(report.Warnings)

	return report, nil
}

// CheckPlanPrefixes validates that all Plan entities use prefixes declared in
// the prefix registry. validPrefix returns true if the prefix is declared.
func CheckPlanPrefixes(
	loadAllPlans func() ([]EntityInfo, error),
	validPrefix func(prefix string) bool,
) (*HealthReport, error) {
	plans, err := loadAllPlans()
	if err != nil {
		return nil, fmt.Errorf("loading plans: %w", err)
	}

	report := &HealthReport{
		Summary: HealthSummary{
			EntitiesByType: make(map[string]int),
		},
	}

	for _, p := range plans {
		report.Summary.TotalEntities++
		report.Summary.EntitiesByType["plan"]++

		prefix, _, _ := model.ParsePlanID(p.ID)
		if prefix == "" {
			report.Errors = append(report.Errors, ValidationError{
				EntityType: "plan",
				EntityID:   p.ID,
				Field:      "id",
				Message:    fmt.Sprintf("cannot parse Plan ID prefix from %q", p.ID),
			})
			continue
		}

		if !validPrefix(prefix) {
			report.Errors = append(report.Errors, ValidationError{
				EntityType: "plan",
				EntityID:   p.ID,
				Field:      "id",
				Message:    fmt.Sprintf("Plan uses undeclared prefix %q", prefix),
			})
		}
	}

	report.Summary.ErrorCount = len(report.Errors)
	report.Summary.WarningCount = len(report.Warnings)

	return report, nil
}

// CheckFeatureParentRefs validates that every feature's parent field references
// an existing Plan entity.
func CheckFeatureParentRefs(
	loadAllFeatures func() ([]EntityInfo, error),
	entityExists func(entityType, entityID string) bool,
) (*HealthReport, error) {
	features, err := loadAllFeatures()
	if err != nil {
		return nil, fmt.Errorf("loading features: %w", err)
	}

	report := &HealthReport{
		Summary: HealthSummary{
			EntitiesByType: make(map[string]int),
		},
	}

	for _, f := range features {
		report.Summary.TotalEntities++
		report.Summary.EntitiesByType["feature"]++

		parent := toString(f.Fields["parent"])
		if parent == "" {
			continue // legacy features may not have parent
		}

		if model.IsPlanID(parent) {
			if !entityExists(string(model.EntityKindPlan), parent) {
				report.Errors = append(report.Errors, ValidationError{
					EntityType: "feature",
					EntityID:   f.ID,
					Field:      "parent",
					Message:    fmt.Sprintf("references non-existent plan %q", parent),
				})
			}
		}
	}

	report.Summary.ErrorCount = len(report.Errors)
	report.Summary.WarningCount = len(report.Warnings)

	return report, nil
}

// MergeReports combines multiple health reports into one.
func MergeReports(reports ...*HealthReport) *HealthReport {
	merged := &HealthReport{
		Summary: HealthSummary{
			EntitiesByType: make(map[string]int),
		},
	}

	for _, r := range reports {
		if r == nil {
			continue
		}
		merged.Errors = append(merged.Errors, r.Errors...)
		merged.Warnings = append(merged.Warnings, r.Warnings...)
		merged.Summary.TotalEntities += r.Summary.TotalEntities
		for k, v := range r.Summary.EntitiesByType {
			merged.Summary.EntitiesByType[k] += v
		}
	}

	merged.Summary.ErrorCount = len(merged.Errors)
	merged.Summary.WarningCount = len(merged.Warnings)

	return merged
}

// inferEntityType guesses the entity type from an ID string.
func inferEntityType(id string) string {
	if model.IsPlanID(id) {
		return string(model.EntityKindPlan)
	}
	if strings.HasPrefix(id, "EPIC-") {
		return string(model.EntityKindEpic)
	}
	if strings.HasPrefix(id, "FEAT-") {
		return string(model.EntityKindFeature)
	}
	if strings.HasPrefix(id, "TASK-") {
		return string(model.EntityKindTask)
	}
	if strings.HasPrefix(id, "BUG-") {
		return string(model.EntityKindBug)
	}
	if strings.HasPrefix(id, "DEC-") {
		return string(model.EntityKindDecision)
	}
	return ""
}

// toString extracts a string from an any value, returning "" if nil or not a string.
func toString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
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
