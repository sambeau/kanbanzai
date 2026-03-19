package validate

import (
	"strings"
	"testing"
)

func TestValidateBugSeverity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "low is valid", value: "low", wantErr: false},
		{name: "medium is valid", value: "medium", wantErr: false},
		{name: "high is valid", value: "high", wantErr: false},
		{name: "critical is valid", value: "critical", wantErr: false},
		{name: "extreme is invalid", value: "extreme", wantErr: true},
		{name: "empty is invalid", value: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateBugSeverity(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBugSeverity(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestValidateBugPriority(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "low is valid", value: "low", wantErr: false},
		{name: "medium is valid", value: "medium", wantErr: false},
		{name: "high is valid", value: "high", wantErr: false},
		{name: "critical is valid", value: "critical", wantErr: false},
		{name: "extreme is invalid", value: "extreme", wantErr: true},
		{name: "empty is invalid", value: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateBugPriority(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBugPriority(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestValidateBugType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "implementation-defect is valid", value: "implementation-defect", wantErr: false},
		{name: "specification-defect is valid", value: "specification-defect", wantErr: false},
		{name: "design-problem is valid", value: "design-problem", wantErr: false},
		{name: "typo is invalid", value: "typo", wantErr: true},
		{name: "empty is invalid", value: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateBugType(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBugType(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func validEpicFields() map[string]any {
	return map[string]any{
		"id":         "E-001",
		"slug":       "my-epic",
		"title":      "My Epic",
		"status":     "proposed",
		"summary":    "A summary of the epic",
		"created":    "2024-01-15",
		"created_by": "alice",
	}
}

func validFeatureFields() map[string]any {
	return map[string]any{
		"id":         "FEAT-001",
		"slug":       "my-feature",
		"epic":       "EPIC-001",
		"status":     "draft",
		"summary":    "A summary of the feature",
		"created":    "2024-01-15",
		"created_by": "alice",
	}
}

func validTaskFields() map[string]any {
	return map[string]any{
		"id":      "FEAT-001.1",
		"feature": "FEAT-001",
		"slug":    "my-task",
		"summary": "A summary of the task",
		"status":  "queued",
	}
}

func validBugFields() map[string]any {
	return map[string]any{
		"id":          "BUG-001",
		"slug":        "my-bug",
		"title":       "My Bug",
		"status":      "reported",
		"severity":    "high",
		"priority":    "medium",
		"type":        "implementation-defect",
		"reported_by": "bob",
		"reported":    "2024-01-15",
		"observed":    "The button doesn't work",
		"expected":    "The button should work",
	}
}

func validDecisionFields() map[string]any {
	return map[string]any{
		"id":         "DEC-001",
		"slug":       "my-decision",
		"summary":    "We decided something",
		"rationale":  "Because reasons",
		"decided_by": "alice",
		"date":       "2024-01-15",
		"status":     "proposed",
	}
}

func TestValidateRecord_ValidEpic(t *testing.T) {
	t.Parallel()

	errs := ValidateRecord("epic", validEpicFields())
	if len(errs) != 0 {
		t.Errorf("expected no errors for valid epic, got %v", errs)
	}
}

func TestValidateRecord_ValidBug(t *testing.T) {
	t.Parallel()

	errs := ValidateRecord("bug", validBugFields())
	if len(errs) != 0 {
		t.Errorf("expected no errors for valid bug, got %v", errs)
	}
}

func TestValidateRecord_ValidDecision(t *testing.T) {
	t.Parallel()

	errs := ValidateRecord("decision", validDecisionFields())
	if len(errs) != 0 {
		t.Errorf("expected no errors for valid decision, got %v", errs)
	}
}

func TestValidateSlug(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		slug    string
		wantErr bool
	}{
		{name: "valid kebab case", slug: "my-valid-slug", wantErr: false},
		{name: "valid single word", slug: "feature", wantErr: false},
		{name: "empty", slug: "", wantErr: true},
		{name: "uppercase", slug: "My-Slug", wantErr: true},
		{name: "underscore", slug: "my_slug", wantErr: true},
		{name: "space", slug: "my slug", wantErr: true},
		{name: "slash", slug: "my/slug", wantErr: true},
		{name: "backslash", slug: "my\\slug", wantErr: true},
		{name: "special chars", slug: "my.slug!", wantErr: true},
		{name: "leading hyphen", slug: "-my-slug", wantErr: true},
		{name: "trailing hyphen", slug: "my-slug-", wantErr: true},
		{name: "double hyphen", slug: "my--slug", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateSlug(tt.slug)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSlug(%q) error = %v, wantErr %v", tt.slug, err, tt.wantErr)
			}
		})
	}
}

func TestValidateRecord_MissingRequiredFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		entityType   string
		baseFields   func() map[string]any
		missingField string
	}{
		{name: "epic missing title", entityType: "epic", baseFields: validEpicFields, missingField: "title"},
		{name: "epic missing slug", entityType: "epic", baseFields: validEpicFields, missingField: "slug"},
		{name: "epic missing status", entityType: "epic", baseFields: validEpicFields, missingField: "status"},
		{name: "epic missing summary", entityType: "epic", baseFields: validEpicFields, missingField: "summary"},
		{name: "epic missing created", entityType: "epic", baseFields: validEpicFields, missingField: "created"},
		{name: "epic missing created_by", entityType: "epic", baseFields: validEpicFields, missingField: "created_by"},
		{name: "feature missing epic", entityType: "feature", baseFields: validFeatureFields, missingField: "epic"},
		{name: "feature missing slug", entityType: "feature", baseFields: validFeatureFields, missingField: "slug"},
		{name: "task missing feature", entityType: "task", baseFields: validTaskFields, missingField: "feature"},
		{name: "task missing slug", entityType: "task", baseFields: validTaskFields, missingField: "slug"},
		{name: "bug missing severity", entityType: "bug", baseFields: validBugFields, missingField: "severity"},
		{name: "bug missing priority", entityType: "bug", baseFields: validBugFields, missingField: "priority"},
		{name: "bug missing type", entityType: "bug", baseFields: validBugFields, missingField: "type"},
		{name: "bug missing reported_by", entityType: "bug", baseFields: validBugFields, missingField: "reported_by"},
		{name: "decision missing rationale", entityType: "decision", baseFields: validDecisionFields, missingField: "rationale"},
		{name: "decision missing decided_by", entityType: "decision", baseFields: validDecisionFields, missingField: "decided_by"},
		{name: "decision missing date", entityType: "decision", baseFields: validDecisionFields, missingField: "date"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fields := tt.baseFields()
			delete(fields, tt.missingField)

			errs := ValidateRecord(tt.entityType, fields)

			found := false
			for _, e := range errs {
				if e.Field == tt.missingField {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected error for missing field %q, got errors: %v", tt.missingField, errs)
			}
		})
	}
}

func TestValidateRecord_EmptyRequiredField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		entityType string
		baseFields func() map[string]any
		emptyField string
	}{
		{name: "epic empty title", entityType: "epic", baseFields: validEpicFields, emptyField: "title"},
		{name: "feature empty summary", entityType: "feature", baseFields: validFeatureFields, emptyField: "summary"},
		{name: "task empty slug", entityType: "task", baseFields: validTaskFields, emptyField: "slug"},
		{name: "bug empty observed", entityType: "bug", baseFields: validBugFields, emptyField: "observed"},
		{name: "decision empty rationale", entityType: "decision", baseFields: validDecisionFields, emptyField: "rationale"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fields := tt.baseFields()
			fields[tt.emptyField] = ""

			errs := ValidateRecord(tt.entityType, fields)

			found := false
			for _, e := range errs {
				if e.Field == tt.emptyField {
					found = true
					if !strings.Contains(e.Message, "empty") {
						t.Errorf("expected error message to mention 'empty' for field %q, got %q", tt.emptyField, e.Message)
					}
					break
				}
			}
			if !found {
				t.Errorf("expected error for empty field %q, got errors: %v", tt.emptyField, errs)
			}
		})
	}
}

func TestValidateRecord_UnknownEntityType(t *testing.T) {
	t.Parallel()

	errs := ValidateRecord("unknown", map[string]any{"id": "X-001"})
	if len(errs) != 1 {
		t.Fatalf("expected 1 error for unknown entity type, got %d: %v", len(errs), errs)
	}
	if errs[0].Field != "type" {
		t.Errorf("expected error on field 'type', got %q", errs[0].Field)
	}
	if !strings.Contains(errs[0].Message, "unknown") {
		t.Errorf("expected error message to mention 'unknown', got %q", errs[0].Message)
	}
}

func TestValidateRecord_InvalidStatus(t *testing.T) {
	t.Parallel()

	fields := validEpicFields()
	fields["status"] = "mystery"

	errs := ValidateRecord("epic", fields)

	found := false
	for _, e := range errs {
		if e.Field == "status" {
			found = true
			if !strings.Contains(e.Message, "mystery") {
				t.Errorf("expected status error to mention 'mystery', got %q", e.Message)
			}
			break
		}
	}
	if !found {
		t.Errorf("expected a status validation error, got errors: %v", errs)
	}
}

func TestValidateRecord_InvalidBugEnums(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		field    string
		badValue string
	}{
		{name: "bad severity", field: "severity", badValue: "extreme"},
		{name: "bad priority", field: "priority", badValue: "urgent"},
		{name: "bad type", field: "type", badValue: "typo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fields := validBugFields()
			fields[tt.field] = tt.badValue

			errs := ValidateRecord("bug", fields)

			found := false
			for _, e := range errs {
				if e.Field == tt.field {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected error for invalid %s %q, got errors: %v", tt.field, tt.badValue, errs)
			}
		})
	}
}

func TestValidateEntityExists_Exists(t *testing.T) {
	t.Parallel()

	result := ValidateEntityExists("epic", "EPIC-001", func(entityType, id string) bool {
		return true
	})
	if result != nil {
		t.Errorf("expected nil for existing entity, got %v", result)
	}
}

func TestValidateEntityExists_NotExists(t *testing.T) {
	t.Parallel()

	result := ValidateEntityExists("epic", "EPIC-999", func(entityType, id string) bool {
		return false
	})
	if result == nil {
		t.Fatal("expected non-nil ValidationError for missing entity")
	}
	if result.EntityType != "epic" {
		t.Errorf("expected EntityType 'epic', got %q", result.EntityType)
	}
	if result.EntityID != "EPIC-999" {
		t.Errorf("expected EntityID 'EPIC-999', got %q", result.EntityID)
	}
}

func TestValidationError_Error(t *testing.T) {
	t.Parallel()

	t.Run("with entity ID", func(t *testing.T) {
		t.Parallel()

		e := ValidationError{
			EntityType: "epic",
			EntityID:   "EPIC-001",
			Field:      "title",
			Message:    "required field is missing",
		}
		got := e.Error()
		want := "epic EPIC-001: title: required field is missing"
		if got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("without entity ID", func(t *testing.T) {
		t.Parallel()

		e := ValidationError{
			EntityType: "epic",
			EntityID:   "",
			Field:      "id",
			Message:    "required field is missing",
		}
		got := e.Error()
		want := "epic: id: required field is missing"
		if got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})
}
