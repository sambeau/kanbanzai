package validate

import (
	"errors"
	"strings"
	"testing"
)

// existsSet builds an entityExists function from a set of type+id pairs.
func existsSet(entries ...EntityInfo) func(string, string) bool {
	m := make(map[[2]string]bool, len(entries))
	for _, e := range entries {
		m[[2]string{e.Type, e.ID}] = true
	}
	return func(entityType, id string) bool {
		return m[[2]string{entityType, id}]
	}
}

func hasErrorWithField(errs []ValidationError, field string) bool {
	for _, e := range errs {
		if e.Field == field {
			return true
		}
	}
	return false
}

func hasErrorMatching(errs []ValidationError, field, substr string) bool {
	for _, e := range errs {
		if e.Field == field && strings.Contains(e.Message, substr) {
			return true
		}
	}
	return false
}

func hasWarningMatching(warnings []ValidationWarning, field, substr string) bool {
	for _, w := range warnings {
		if w.Field == field && strings.Contains(w.Message, substr) {
			return true
		}
	}
	return false
}

func epicFields(id, slug string) map[string]any {
	f := validEpicFields()
	f["id"] = id
	f["slug"] = slug
	return f
}

func featureFields(id, slug, epic string) map[string]any {
	f := validFeatureFields()
	f["id"] = id
	f["slug"] = slug
	f["epic"] = epic
	return f
}

func taskFields(id, feature, slug string) map[string]any {
	f := validTaskFields()
	f["id"] = id
	f["feature"] = feature
	f["slug"] = slug
	return f
}

func bugFields(id, slug string) map[string]any {
	f := validBugFields()
	f["id"] = id
	f["slug"] = slug
	return f
}

func decisionFields(id, slug string) map[string]any {
	f := validDecisionFields()
	f["id"] = id
	f["slug"] = slug
	return f
}

func TestCheckHealth_AllValid(t *testing.T) {
	t.Parallel()

	epic := EntityInfo{Type: string(EntityEpic), ID: "E-001", Fields: epicFields("E-001", "test")}
	feat := EntityInfo{Type: string(EntityFeature), ID: "FEAT-001", Fields: featureFields("FEAT-001", "test-feat", "E-001")}

	entities := []EntityInfo{epic, feat}
	loadAll := func() ([]EntityInfo, error) { return entities, nil }
	exists := existsSet(entities...)

	report, err := CheckHealth(loadAll, exists)
	if err != nil {
		t.Fatalf("CheckHealth returned error: %v", err)
	}
	if len(report.Errors) != 0 {
		t.Fatalf("expected 0 errors, got %d: %v", len(report.Errors), report.Errors)
	}
	if len(report.Warnings) != 0 {
		t.Fatalf("expected 0 warnings, got %d: %v", len(report.Warnings), report.Warnings)
	}
	if report.Summary.TotalEntities != 2 {
		t.Fatalf("expected TotalEntities=2, got %d", report.Summary.TotalEntities)
	}
	if report.Summary.ErrorCount != 0 {
		t.Fatalf("expected ErrorCount=0, got %d", report.Summary.ErrorCount)
	}
	if report.Summary.WarningCount != 0 {
		t.Fatalf("expected WarningCount=0, got %d", report.Summary.WarningCount)
	}
	if report.Summary.EntitiesByType[string(EntityEpic)] != 1 {
		t.Fatalf("expected 1 epic, got %d", report.Summary.EntitiesByType[string(EntityEpic)])
	}
	if report.Summary.EntitiesByType[string(EntityFeature)] != 1 {
		t.Fatalf("expected 1 feature, got %d", report.Summary.EntitiesByType[string(EntityFeature)])
	}
}

func TestCheckHealth_BrokenReference(t *testing.T) {
	t.Parallel()

	feat := EntityInfo{
		Type:   string(EntityFeature),
		ID:     "FEAT-001",
		Fields: featureFields("FEAT-001", "test-feat", "E-999"),
	}

	loadAll := func() ([]EntityInfo, error) { return []EntityInfo{feat}, nil }
	exists := func(entityType, id string) bool { return false }

	report, err := CheckHealth(loadAll, exists)
	if err != nil {
		t.Fatalf("CheckHealth returned error: %v", err)
	}
	if !hasErrorMatching(report.Errors, "epic", "non-existent") {
		t.Fatalf("expected error on field 'epic' about non-existent entity, got errors: %v", report.Errors)
	}
}

func TestCheckHealth_TaskBrokenFeatureRef(t *testing.T) {
	t.Parallel()

	task := EntityInfo{
		Type:   string(EntityTask),
		ID:     "FEAT-001.1",
		Fields: taskFields("FEAT-001.1", "FEAT-999", "do-thing"),
	}

	loadAll := func() ([]EntityInfo, error) { return []EntityInfo{task}, nil }
	exists := func(entityType, id string) bool { return false }

	report, err := CheckHealth(loadAll, exists)
	if err != nil {
		t.Fatalf("CheckHealth returned error: %v", err)
	}
	if !hasErrorMatching(report.Errors, "feature", "non-existent") {
		t.Fatalf("expected error on field 'feature' about non-existent entity, got errors: %v", report.Errors)
	}
}

func TestCheckHealth_MalformedEntityID(t *testing.T) {
	t.Parallel()

	feature := EntityInfo{
		Type:   string(EntityFeature),
		ID:     "E-001",
		Fields: featureFields("E-001", "test-feat", "E-123"),
	}

	loadAll := func() ([]EntityInfo, error) { return []EntityInfo{feature}, nil }
	exists := func(entityType, id string) bool {
		return entityType == string(EntityEpic) && id == "E-123"
	}

	report, err := CheckHealth(loadAll, exists)
	if err != nil {
		t.Fatalf("CheckHealth returned error: %v", err)
	}
	if !hasErrorMatching(report.Errors, "id", "expected prefix FEAT") {
		t.Fatalf("expected malformed ID error, got errors: %v", report.Errors)
	}
}

func TestCheckHealth_BugBrokenDuplicateRef(t *testing.T) {
	t.Parallel()

	fields := bugFields("BUG-001", "broken-dup")
	fields["duplicate_of"] = "BUG-999"

	bug := EntityInfo{Type: string(EntityBug), ID: "BUG-001", Fields: fields}

	loadAll := func() ([]EntityInfo, error) { return []EntityInfo{bug}, nil }
	exists := func(entityType, id string) bool {
		// The bug itself exists, but BUG-999 does not.
		return entityType == string(EntityBug) && id == "BUG-001"
	}

	report, err := CheckHealth(loadAll, exists)
	if err != nil {
		t.Fatalf("CheckHealth returned error: %v", err)
	}
	if !hasErrorMatching(report.Errors, "duplicate_of", "non-existent") {
		t.Fatalf("expected error on field 'duplicate_of' about non-existent entity, got errors: %v", report.Errors)
	}
}

func TestCheckHealth_SupersessionConsistencyWarning(t *testing.T) {
	t.Parallel()

	fields1 := featureFields("FEAT-001", "feat-one", "E-001")
	fields1["supersedes"] = "FEAT-002"

	fields2 := featureFields("FEAT-002", "feat-two", "E-001")
	// FEAT-002 does NOT have superseded_by=FEAT-001

	feat1 := EntityInfo{Type: string(EntityFeature), ID: "FEAT-001", Fields: fields1}
	feat2 := EntityInfo{Type: string(EntityFeature), ID: "FEAT-002", Fields: fields2}

	entities := []EntityInfo{feat1, feat2}
	loadAll := func() ([]EntityInfo, error) { return entities, nil }
	baseExists := existsSet(entities...)
	exists := func(entityType, id string) bool {
		if entityType == string(EntityEpic) && id == "E-001" {
			return true
		}
		return baseExists(entityType, id)
	}

	report, err := CheckHealth(loadAll, exists)
	if err != nil {
		t.Fatalf("CheckHealth returned error: %v", err)
	}
	if !hasWarningMatching(report.Warnings, "supersedes", "does not have superseded_by") {
		t.Fatalf("expected warning about inconsistent supersession, got warnings: %v", report.Warnings)
	}
}

func TestCheckHealth_SupersessionConsistent(t *testing.T) {
	t.Parallel()

	fields1 := featureFields("FEAT-001", "feat-one", "E-001")
	fields1["supersedes"] = "FEAT-002"

	fields2 := featureFields("FEAT-002", "feat-two", "E-001")
	fields2["superseded_by"] = "FEAT-001"

	feat1 := EntityInfo{Type: string(EntityFeature), ID: "FEAT-001", Fields: fields1}
	feat2 := EntityInfo{Type: string(EntityFeature), ID: "FEAT-002", Fields: fields2}

	entities := []EntityInfo{feat1, feat2}
	loadAll := func() ([]EntityInfo, error) { return entities, nil }
	baseExists := existsSet(entities...)
	exists := func(entityType, id string) bool {
		if entityType == string(EntityEpic) && id == "E-001" {
			return true
		}
		return baseExists(entityType, id)
	}

	report, err := CheckHealth(loadAll, exists)
	if err != nil {
		t.Fatalf("CheckHealth returned error: %v", err)
	}
	if len(report.Warnings) != 0 {
		t.Fatalf("expected 0 warnings, got %d: %v", len(report.Warnings), report.Warnings)
	}
}

func TestCheckHealth_DecisionSupersessionWarning(t *testing.T) {
	t.Parallel()

	fields1 := decisionFields("DEC-001", "dec-one")
	fields1["superseded_by"] = "DEC-002"

	fields2 := decisionFields("DEC-002", "dec-two")
	// DEC-002 does NOT have supersedes=DEC-001

	dec1 := EntityInfo{Type: string(EntityDecision), ID: "DEC-001", Fields: fields1}
	dec2 := EntityInfo{Type: string(EntityDecision), ID: "DEC-002", Fields: fields2}

	entities := []EntityInfo{dec1, dec2}
	loadAll := func() ([]EntityInfo, error) { return entities, nil }
	exists := existsSet(entities...)

	report, err := CheckHealth(loadAll, exists)
	if err != nil {
		t.Fatalf("CheckHealth returned error: %v", err)
	}
	if !hasWarningMatching(report.Warnings, "superseded_by", "does not have supersedes") {
		t.Fatalf("expected warning about inconsistent supersession, got warnings: %v", report.Warnings)
	}
}

func TestCheckHealth_InvalidRecord(t *testing.T) {
	t.Parallel()

	// Epic missing required "title" field.
	fields := epicFields("E-001", "test")
	delete(fields, "title")

	epic := EntityInfo{Type: string(EntityEpic), ID: "E-001", Fields: fields}

	loadAll := func() ([]EntityInfo, error) { return []EntityInfo{epic}, nil }
	exists := existsSet(epic)

	report, err := CheckHealth(loadAll, exists)
	if err != nil {
		t.Fatalf("CheckHealth returned error: %v", err)
	}
	if !hasErrorWithField(report.Errors, "title") {
		t.Fatalf("expected validation error on field 'title', got errors: %v", report.Errors)
	}
}

func TestCheckHealth_LoadError(t *testing.T) {
	t.Parallel()

	loadErr := errors.New("disk on fire")
	loadAll := func() ([]EntityInfo, error) { return nil, loadErr }
	exists := func(string, string) bool { return false }

	report, err := CheckHealth(loadAll, exists)
	if report != nil {
		t.Fatalf("expected nil report on error, got %v", report)
	}
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, loadErr) {
		t.Fatalf("expected wrapped loadErr, got %v", err)
	}
}

func TestCheckHealth_EmptyProject(t *testing.T) {
	t.Parallel()

	loadAll := func() ([]EntityInfo, error) { return []EntityInfo{}, nil }
	exists := func(string, string) bool { return false }

	report, err := CheckHealth(loadAll, exists)
	if err != nil {
		t.Fatalf("CheckHealth returned error: %v", err)
	}
	if len(report.Errors) != 0 {
		t.Fatalf("expected 0 errors, got %d", len(report.Errors))
	}
	if len(report.Warnings) != 0 {
		t.Fatalf("expected 0 warnings, got %d", len(report.Warnings))
	}
	if report.Summary.TotalEntities != 0 {
		t.Fatalf("expected TotalEntities=0, got %d", report.Summary.TotalEntities)
	}
	if report.Summary.ErrorCount != 0 {
		t.Fatalf("expected ErrorCount=0, got %d", report.Summary.ErrorCount)
	}
	if report.Summary.WarningCount != 0 {
		t.Fatalf("expected WarningCount=0, got %d", report.Summary.WarningCount)
	}
	if len(report.Summary.EntitiesByType) != 0 {
		t.Fatalf("expected empty EntitiesByType, got %v", report.Summary.EntitiesByType)
	}
}

func TestCheckHealth_SummaryCountsCorrect(t *testing.T) {
	t.Parallel()

	// Valid epic.
	epic := EntityInfo{Type: string(EntityEpic), ID: "E-001", Fields: epicFields("E-001", "test")}

	// Feature referencing non-existent epic → produces a reference error.
	feat := EntityInfo{
		Type:   string(EntityFeature),
		ID:     "FEAT-001",
		Fields: featureFields("FEAT-001", "test-feat", "E-999"),
	}

	// Task with missing required field "feature" → produces a validation error.
	brokenTaskFields := map[string]any{
		"id":      "FEAT-001.1",
		"slug":    "broken-task",
		"summary": "S",
		"status":  "queued",
	}
	task := EntityInfo{Type: string(EntityTask), ID: "FEAT-001.1", Fields: brokenTaskFields}

	// Two features with inconsistent supersession → produces a warning.
	fields1 := featureFields("FEAT-002", "feat-two", "E-001")
	fields1["supersedes"] = "FEAT-003"
	fields2 := featureFields("FEAT-003", "feat-three", "E-001")
	feat2 := EntityInfo{Type: string(EntityFeature), ID: "FEAT-002", Fields: fields1}
	feat3 := EntityInfo{Type: string(EntityFeature), ID: "FEAT-003", Fields: fields2}

	entities := []EntityInfo{epic, feat, task, feat2, feat3}
	loadAll := func() ([]EntityInfo, error) { return entities, nil }
	baseExists := existsSet(entities...)
	exists := func(entityType, id string) bool {
		if entityType == string(EntityEpic) && id == "E-001" {
			return true
		}
		return baseExists(entityType, id)
	}

	report, err := CheckHealth(loadAll, exists)
	if err != nil {
		t.Fatalf("CheckHealth returned error: %v", err)
	}

	if report.Summary.TotalEntities != 5 {
		t.Fatalf("expected TotalEntities=5, got %d", report.Summary.TotalEntities)
	}
	if report.Summary.ErrorCount != len(report.Errors) {
		t.Fatalf("ErrorCount=%d does not match len(Errors)=%d", report.Summary.ErrorCount, len(report.Errors))
	}
	if report.Summary.WarningCount != len(report.Warnings) {
		t.Fatalf("WarningCount=%d does not match len(Warnings)=%d", report.Summary.WarningCount, len(report.Warnings))
	}
	if report.Summary.ErrorCount == 0 {
		t.Fatal("expected at least one error")
	}
	if report.Summary.WarningCount == 0 {
		t.Fatal("expected at least one warning")
	}
	if report.Summary.EntitiesByType[string(EntityEpic)] != 1 {
		t.Fatalf("expected 1 epic, got %d", report.Summary.EntitiesByType[string(EntityEpic)])
	}
	if report.Summary.EntitiesByType[string(EntityFeature)] != 3 {
		t.Fatalf("expected 3 features, got %d", report.Summary.EntitiesByType[string(EntityFeature)])
	}
	if report.Summary.EntitiesByType[string(EntityTask)] != 1 {
		t.Fatalf("expected 1 task, got %d", report.Summary.EntitiesByType[string(EntityTask)])
	}
}

func TestCheckHealth_EpicBrokenFeatureRef(t *testing.T) {
	t.Parallel()

	fields := epicFields("E-001", "test")
	fields["features"] = []string{"FEAT-999"}

	epic := EntityInfo{Type: string(EntityEpic), ID: "E-001", Fields: fields}

	loadAll := func() ([]EntityInfo, error) { return []EntityInfo{epic}, nil }
	exists := func(entityType, id string) bool {
		return entityType == string(EntityEpic) && id == "E-001"
	}

	report, err := CheckHealth(loadAll, exists)
	if err != nil {
		t.Fatalf("CheckHealth returned error: %v", err)
	}
	if !hasErrorMatching(report.Errors, "features", "non-existent") {
		t.Fatalf("expected error on field 'features' about non-existent entity, got errors: %v", report.Errors)
	}
}

func TestCheckHealth_TaskBrokenDependency(t *testing.T) {
	t.Parallel()

	fields := taskFields("FEAT-001.1", "FEAT-001", "do-thing")
	fields["depends_on"] = []string{"FEAT-001.99"}

	task := EntityInfo{Type: string(EntityTask), ID: "FEAT-001.1", Fields: fields}

	loadAll := func() ([]EntityInfo, error) { return []EntityInfo{task}, nil }
	exists := func(entityType, id string) bool {
		// Task itself and its feature exist, but the dependency does not.
		return (entityType == string(EntityTask) && id == "FEAT-001.1") ||
			(entityType == string(EntityFeature) && id == "FEAT-001")
	}

	report, err := CheckHealth(loadAll, exists)
	if err != nil {
		t.Fatalf("CheckHealth returned error: %v", err)
	}
	if !hasErrorMatching(report.Errors, "depends_on", "non-existent") {
		t.Fatalf("expected error on field 'depends_on' about non-existent entity, got errors: %v", report.Errors)
	}
}

func TestValidationWarning_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		warning ValidationWarning
		want    string
	}{
		{
			name: "with entity ID",
			warning: ValidationWarning{
				EntityType: "feature",
				EntityID:   "FEAT-001",
				Field:      "supersedes",
				Message:    "inconsistent supersession",
			},
			want: "warning: feature FEAT-001: supersedes: inconsistent supersession",
		},
		{
			name: "without entity ID",
			warning: ValidationWarning{
				EntityType: "feature",
				Field:      "supersedes",
				Message:    "inconsistent supersession",
			},
			want: "warning: feature: supersedes: inconsistent supersession",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.warning.Error()
			if got != tt.want {
				t.Fatalf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}
