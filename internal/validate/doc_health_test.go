package validate

import (
	"fmt"
	"os"
	"testing"

	"github.com/sambeau/kanbanzai/internal/model"
)

func docInfo(id string, fields map[string]any) DocumentInfo {
	return DocumentInfo{ID: id, Fields: fields}
}

func validDocFields(id, path, status string) map[string]any {
	return map[string]any{
		"id":           id,
		"path":         path,
		"type":         "design",
		"title":        "Test Document",
		"status":       status,
		"content_hash": "abc123",
		"created":      "2026-01-01T00:00:00Z",
		"created_by":   "tester",
		"updated":      "2026-01-01T00:00:00Z",
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write test file %s: %v", path, err)
	}
}

// --- CheckDocumentHealth tests ---

func TestCheckDocumentHealth_AllValid(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	docPath := tmpDir + "/valid.md"
	writeTestFile(t, docPath, "valid doc")

	fields := validDocFields("FEAT-123/design-v1", docPath, "draft")
	docs := []DocumentInfo{docInfo("FEAT-123/design-v1", fields)}

	loadAll := func() ([]DocumentInfo, error) { return docs, nil }
	entityExists := func(_, _ string) bool { return true }
	checkHash := func(_, _ string) (bool, error) { return true, nil }

	report, err := CheckDocumentHealth(loadAll, entityExists, checkHash)
	if err != nil {
		t.Fatalf("CheckDocumentHealth returned error: %v", err)
	}
	if len(report.Errors) != 0 {
		t.Fatalf("expected 0 errors, got %d: %v", len(report.Errors), report.Errors)
	}
	if len(report.Warnings) != 0 {
		t.Fatalf("expected 0 warnings, got %d: %v", len(report.Warnings), report.Warnings)
	}
	if report.Summary.TotalEntities != 1 {
		t.Fatalf("expected TotalEntities=1, got %d", report.Summary.TotalEntities)
	}
	if report.Summary.EntitiesByType["document"] != 1 {
		t.Fatalf("expected 1 document, got %d", report.Summary.EntitiesByType["document"])
	}
}

func TestCheckDocumentHealth_MissingFile(t *testing.T) {
	t.Parallel()

	fields := validDocFields("FEAT-X/doc", "/tmp/nonexistent-kbz-test-path-12345/missing.md", "draft")
	docs := []DocumentInfo{docInfo("FEAT-X/doc", fields)}

	loadAll := func() ([]DocumentInfo, error) { return docs, nil }
	entityExists := func(_, _ string) bool { return true }
	checkHash := func(_, _ string) (bool, error) { return false, fmt.Errorf("file not found") }

	report, err := CheckDocumentHealth(loadAll, entityExists, checkHash)
	if err != nil {
		t.Fatalf("CheckDocumentHealth returned error: %v", err)
	}

	found := false
	for _, e := range report.Errors {
		if e.Field == "path" && e.EntityID == "FEAT-X/doc" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected a path error for FEAT-X/doc, got: %v", report.Errors)
	}
}

func TestCheckDocumentHealth_ContentHashDrift(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	docPath := tmpDir + "/drifted.md"
	writeTestFile(t, docPath, "old content")

	fields := validDocFields("FEAT-Y/doc", docPath, "draft")
	docs := []DocumentInfo{docInfo("FEAT-Y/doc", fields)}

	loadAll := func() ([]DocumentInfo, error) { return docs, nil }
	entityExists := func(_, _ string) bool { return true }
	checkHash := func(_, _ string) (bool, error) { return false, nil }

	report, err := CheckDocumentHealth(loadAll, entityExists, checkHash)
	if err != nil {
		t.Fatalf("CheckDocumentHealth returned error: %v", err)
	}

	if len(report.Warnings) == 0 {
		t.Fatal("expected warnings for content drift, got 0")
	}

	found := false
	for _, w := range report.Warnings {
		if w.Field == "content_hash" && w.EntityID == "FEAT-Y/doc" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected content_hash warning for FEAT-Y/doc, got: %v", report.Warnings)
	}
}

func TestCheckDocumentHealth_OrphanedDocument(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	docPath := tmpDir + "/orphan.md"
	writeTestFile(t, docPath, "orphaned")

	fields := validDocFields("FEAT-Z/doc", docPath, "draft")
	fields["owner"] = "FEAT-01NONEXISTENT"
	docs := []DocumentInfo{docInfo("FEAT-Z/doc", fields)}

	loadAll := func() ([]DocumentInfo, error) { return docs, nil }
	entityExists := func(_, _ string) bool { return false }
	checkHash := func(_, _ string) (bool, error) { return true, nil }

	report, err := CheckDocumentHealth(loadAll, entityExists, checkHash)
	if err != nil {
		t.Fatalf("CheckDocumentHealth returned error: %v", err)
	}

	found := false
	for _, e := range report.Errors {
		if e.Field == "owner" && e.EntityID == "FEAT-Z/doc" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected owner error for orphaned document, got: %v", report.Errors)
	}
}

func TestCheckDocumentHealth_ApprovedMissingApprovalFields(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	docPath := tmpDir + "/approved.md"
	writeTestFile(t, docPath, "approved doc")

	fields := validDocFields("FEAT-Q/doc", docPath, string(model.DocumentStatusApproved))
	docs := []DocumentInfo{docInfo("FEAT-Q/doc", fields)}

	loadAll := func() ([]DocumentInfo, error) { return docs, nil }
	entityExists := func(_, _ string) bool { return true }
	checkHash := func(_, _ string) (bool, error) { return true, nil }

	report, err := CheckDocumentHealth(loadAll, entityExists, checkHash)
	if err != nil {
		t.Fatalf("CheckDocumentHealth returned error: %v", err)
	}

	foundBy := false
	foundAt := false
	for _, e := range report.Errors {
		if e.EntityID == "FEAT-Q/doc" && e.Field == "approved_by" {
			foundBy = true
		}
		if e.EntityID == "FEAT-Q/doc" && e.Field == "approved_at" {
			foundAt = true
		}
	}
	if !foundBy {
		t.Error("expected approved_by error for approved document without approved_by")
	}
	if !foundAt {
		t.Error("expected approved_at error for approved document without approved_at")
	}
}

func TestCheckDocumentHealth_ApprovedWithApprovalFields(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	docPath := tmpDir + "/ok-approved.md"
	writeTestFile(t, docPath, "properly approved")

	fields := validDocFields("FEAT-R/doc", docPath, string(model.DocumentStatusApproved))
	fields["approved_by"] = "reviewer"
	fields["approved_at"] = "2026-02-01T00:00:00Z"
	docs := []DocumentInfo{docInfo("FEAT-R/doc", fields)}

	loadAll := func() ([]DocumentInfo, error) { return docs, nil }
	entityExists := func(_, _ string) bool { return true }
	checkHash := func(_, _ string) (bool, error) { return true, nil }

	report, err := CheckDocumentHealth(loadAll, entityExists, checkHash)
	if err != nil {
		t.Fatalf("CheckDocumentHealth returned error: %v", err)
	}

	for _, e := range report.Errors {
		if e.EntityID == "FEAT-R/doc" && (e.Field == "approved_by" || e.Field == "approved_at") {
			t.Errorf("unexpected approval error: %v", e)
		}
	}
}

func TestCheckDocumentHealth_LoadError(t *testing.T) {
	t.Parallel()

	loadAll := func() ([]DocumentInfo, error) {
		return nil, fmt.Errorf("disk failure")
	}
	entityExists := func(_, _ string) bool { return true }

	_, err := CheckDocumentHealth(loadAll, entityExists, nil)
	if err == nil {
		t.Fatal("expected error when loadAllDocs fails")
	}
}

func TestCheckDocumentHealth_NoOwner(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	docPath := tmpDir + "/no-owner.md"
	writeTestFile(t, docPath, "no owner")

	fields := validDocFields("standalone/doc", docPath, "draft")
	docs := []DocumentInfo{docInfo("standalone/doc", fields)}

	loadAll := func() ([]DocumentInfo, error) { return docs, nil }
	entityExists := func(_, _ string) bool { return false }
	checkHash := func(_, _ string) (bool, error) { return true, nil }

	report, err := CheckDocumentHealth(loadAll, entityExists, checkHash)
	if err != nil {
		t.Fatalf("CheckDocumentHealth returned error: %v", err)
	}

	for _, e := range report.Errors {
		if e.Field == "owner" {
			t.Errorf("should not report owner error when owner is empty: %v", e)
		}
	}
}

func TestCheckDocumentHealth_EmptyDocumentSet(t *testing.T) {
	t.Parallel()

	loadAll := func() ([]DocumentInfo, error) { return nil, nil }
	entityExists := func(_, _ string) bool { return true }

	report, err := CheckDocumentHealth(loadAll, entityExists, nil)
	if err != nil {
		t.Fatalf("CheckDocumentHealth returned error: %v", err)
	}
	if report.Summary.TotalEntities != 0 {
		t.Errorf("expected 0 entities, got %d", report.Summary.TotalEntities)
	}
}

// --- CheckPlanPrefixes tests ---

func TestCheckPlanPrefixes_AllValid(t *testing.T) {
	t.Parallel()

	plans := []EntityInfo{
		{Type: "plan", ID: "P1-basic-ui", Fields: map[string]any{"id": "P1-basic-ui"}},
		{Type: "plan", ID: "P2-backend", Fields: map[string]any{"id": "P2-backend"}},
	}
	loadAll := func() ([]EntityInfo, error) { return plans, nil }
	validPrefix := func(p string) bool { return p == "P" }

	report, err := CheckPlanPrefixes(loadAll, validPrefix)
	if err != nil {
		t.Fatalf("CheckPlanPrefixes returned error: %v", err)
	}
	if len(report.Errors) != 0 {
		t.Fatalf("expected 0 errors, got %d: %v", len(report.Errors), report.Errors)
	}
	if report.Summary.TotalEntities != 2 {
		t.Fatalf("expected 2 entities, got %d", report.Summary.TotalEntities)
	}
}

func TestCheckPlanPrefixes_UndeclaredPrefix(t *testing.T) {
	t.Parallel()

	plans := []EntityInfo{
		{Type: "plan", ID: "X1-experimental", Fields: map[string]any{"id": "X1-experimental"}},
	}
	loadAll := func() ([]EntityInfo, error) { return plans, nil }
	validPrefix := func(p string) bool { return p == "P" }

	report, err := CheckPlanPrefixes(loadAll, validPrefix)
	if err != nil {
		t.Fatalf("CheckPlanPrefixes returned error: %v", err)
	}
	if len(report.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(report.Errors), report.Errors)
	}
	if report.Errors[0].Field != "id" {
		t.Errorf("error field = %q, want 'id'", report.Errors[0].Field)
	}
}

func TestCheckPlanPrefixes_InvalidPlanID(t *testing.T) {
	t.Parallel()

	plans := []EntityInfo{
		{Type: "plan", ID: "not-a-plan-id", Fields: map[string]any{"id": "not-a-plan-id"}},
	}
	loadAll := func() ([]EntityInfo, error) { return plans, nil }
	validPrefix := func(_ string) bool { return true }

	report, err := CheckPlanPrefixes(loadAll, validPrefix)
	if err != nil {
		t.Fatalf("CheckPlanPrefixes returned error: %v", err)
	}
	if len(report.Errors) != 1 {
		t.Fatalf("expected 1 error for unparsable plan ID, got %d", len(report.Errors))
	}
}

func TestCheckPlanPrefixes_LoadError(t *testing.T) {
	t.Parallel()

	loadAll := func() ([]EntityInfo, error) {
		return nil, fmt.Errorf("failed")
	}
	_, err := CheckPlanPrefixes(loadAll, func(_ string) bool { return true })
	if err == nil {
		t.Fatal("expected error when loadAllPlans fails")
	}
}

func TestCheckPlanPrefixes_MultiplePrefixes(t *testing.T) {
	t.Parallel()

	plans := []EntityInfo{
		{Type: "plan", ID: "P1-alpha", Fields: map[string]any{"id": "P1-alpha"}},
		{Type: "plan", ID: "Q1-beta", Fields: map[string]any{"id": "Q1-beta"}},
		{Type: "plan", ID: "R1-gamma", Fields: map[string]any{"id": "R1-gamma"}},
	}
	loadAll := func() ([]EntityInfo, error) { return plans, nil }
	declared := map[string]bool{"P": true, "Q": true}
	validPrefix := func(p string) bool { return declared[p] }

	report, err := CheckPlanPrefixes(loadAll, validPrefix)
	if err != nil {
		t.Fatalf("CheckPlanPrefixes returned error: %v", err)
	}
	if len(report.Errors) != 1 {
		t.Fatalf("expected 1 error (R undeclared), got %d: %v", len(report.Errors), report.Errors)
	}
	if report.Errors[0].EntityID != "R1-gamma" {
		t.Errorf("error entity = %q, want R1-gamma", report.Errors[0].EntityID)
	}
}

// --- CheckFeatureParentRefs tests ---

func TestCheckFeatureParentRefs_ValidParent(t *testing.T) {
	t.Parallel()

	features := []EntityInfo{
		{Type: "feature", ID: "FEAT-01AAAA", Fields: map[string]any{
			"id":     "FEAT-01AAAA",
			"parent": "P1-test",
		}},
	}
	loadAll := func() ([]EntityInfo, error) { return features, nil }
	entityExists := func(typ, id string) bool {
		return typ == "batch" && id == "P1-test"
	}

	report, err := CheckFeatureParentRefs(loadAll, entityExists)
	if err != nil {
		t.Fatalf("CheckFeatureParentRefs returned error: %v", err)
	}
	if len(report.Errors) != 0 {
		t.Fatalf("expected 0 errors, got %d: %v", len(report.Errors), report.Errors)
	}
}

func TestCheckFeatureParentRefs_MissingParent(t *testing.T) {
	t.Parallel()

	features := []EntityInfo{
		{Type: "feature", ID: "FEAT-01BBBB", Fields: map[string]any{
			"id":     "FEAT-01BBBB",
			"parent": "P99-nonexistent",
		}},
	}
	loadAll := func() ([]EntityInfo, error) { return features, nil }
	entityExists := func(_, _ string) bool { return false }

	report, err := CheckFeatureParentRefs(loadAll, entityExists)
	if err != nil {
		t.Fatalf("CheckFeatureParentRefs returned error: %v", err)
	}
	if len(report.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(report.Errors), report.Errors)
	}
	if report.Errors[0].Field != "parent" {
		t.Errorf("error field = %q, want 'parent'", report.Errors[0].Field)
	}
}

func TestCheckFeatureParentRefs_NoParent(t *testing.T) {
	t.Parallel()

	features := []EntityInfo{
		{Type: "feature", ID: "FEAT-01CCCC", Fields: map[string]any{
			"id": "FEAT-01CCCC",
		}},
	}
	loadAll := func() ([]EntityInfo, error) { return features, nil }
	entityExists := func(_, _ string) bool { return false }

	report, err := CheckFeatureParentRefs(loadAll, entityExists)
	if err != nil {
		t.Fatalf("CheckFeatureParentRefs returned error: %v", err)
	}
	if len(report.Errors) != 0 {
		t.Fatalf("expected 0 errors for feature without parent, got %d", len(report.Errors))
	}
}

func TestCheckFeatureParentRefs_NonPlanParent(t *testing.T) {
	t.Parallel()

	features := []EntityInfo{
		{Type: "feature", ID: "FEAT-01DDDD", Fields: map[string]any{
			"id":     "FEAT-01DDDD",
			"parent": "PROJ-LEGACY",
		}},
	}
	loadAll := func() ([]EntityInfo, error) { return features, nil }
	entityExists := func(_, _ string) bool { return false }

	report, err := CheckFeatureParentRefs(loadAll, entityExists)
	if err != nil {
		t.Fatalf("CheckFeatureParentRefs returned error: %v", err)
	}
	if len(report.Errors) != 0 {
		t.Fatalf("expected 0 errors for non-plan parent, got %d: %v", len(report.Errors), report.Errors)
	}
}

func TestCheckFeatureParentRefs_LoadError(t *testing.T) {
	t.Parallel()

	loadAll := func() ([]EntityInfo, error) {
		return nil, fmt.Errorf("failed")
	}
	_, err := CheckFeatureParentRefs(loadAll, func(_, _ string) bool { return true })
	if err == nil {
		t.Fatal("expected error when loadAllFeatures fails")
	}
}

func TestCheckFeatureParentRefs_MultipleFeatures(t *testing.T) {
	t.Parallel()

	features := []EntityInfo{
		{Type: "feature", ID: "FEAT-01EEEE", Fields: map[string]any{
			"id":     "FEAT-01EEEE",
			"parent": "P1-good",
		}},
		{Type: "feature", ID: "FEAT-01FFFF", Fields: map[string]any{
			"id":     "FEAT-01FFFF",
			"parent": "P9-bad",
		}},
		{Type: "feature", ID: "FEAT-01GGGG", Fields: map[string]any{
			"id": "FEAT-01GGGG",
		}},
	}
	loadAll := func() ([]EntityInfo, error) { return features, nil }
	entityExists := func(typ, id string) bool {
		return typ == "batch" && id == "P1-good"
	}

	report, err := CheckFeatureParentRefs(loadAll, entityExists)
	if err != nil {
		t.Fatalf("CheckFeatureParentRefs returned error: %v", err)
	}
	if len(report.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(report.Errors), report.Errors)
	}
	if report.Errors[0].EntityID != "FEAT-01FFFF" {
		t.Errorf("error entity = %q, want FEAT-01FFFF", report.Errors[0].EntityID)
	}
}

// --- MergeReports tests ---

func TestMergeReports_Empty(t *testing.T) {
	t.Parallel()

	merged := MergeReports()
	if merged.Summary.TotalEntities != 0 {
		t.Errorf("expected 0 total, got %d", merged.Summary.TotalEntities)
	}
	if len(merged.Errors) != 0 {
		t.Errorf("expected 0 errors, got %d", len(merged.Errors))
	}
}

func TestMergeReports_CombinesReports(t *testing.T) {
	t.Parallel()

	r1 := &HealthReport{
		Errors: []ValidationError{
			{EntityType: "plan", EntityID: "P1-x", Field: "id", Message: "bad prefix"},
		},
		Warnings: []ValidationWarning{
			{EntityType: "document", EntityID: "d1", Field: "hash", Message: "drift"},
		},
		Summary: HealthSummary{
			TotalEntities:  3,
			ErrorCount:     1,
			WarningCount:   1,
			EntitiesByType: map[string]int{"plan": 2, "document": 1},
		},
	}
	r2 := &HealthReport{
		Errors: []ValidationError{
			{EntityType: "feature", EntityID: "F1", Field: "parent", Message: "missing"},
		},
		Summary: HealthSummary{
			TotalEntities:  2,
			ErrorCount:     1,
			EntitiesByType: map[string]int{"feature": 2},
		},
	}

	merged := MergeReports(r1, r2)

	if merged.Summary.TotalEntities != 5 {
		t.Errorf("expected 5 total, got %d", merged.Summary.TotalEntities)
	}
	if len(merged.Errors) != 2 {
		t.Errorf("expected 2 errors, got %d", len(merged.Errors))
	}
	if len(merged.Warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(merged.Warnings))
	}
	if merged.Summary.ErrorCount != 2 {
		t.Errorf("expected ErrorCount=2, got %d", merged.Summary.ErrorCount)
	}
	if merged.Summary.WarningCount != 1 {
		t.Errorf("expected WarningCount=1, got %d", merged.Summary.WarningCount)
	}
	if merged.Summary.EntitiesByType["plan"] != 2 {
		t.Errorf("expected 2 plans, got %d", merged.Summary.EntitiesByType["plan"])
	}
	if merged.Summary.EntitiesByType["feature"] != 2 {
		t.Errorf("expected 2 features, got %d", merged.Summary.EntitiesByType["feature"])
	}
	if merged.Summary.EntitiesByType["document"] != 1 {
		t.Errorf("expected 1 document, got %d", merged.Summary.EntitiesByType["document"])
	}
}

func TestMergeReports_NilReports(t *testing.T) {
	t.Parallel()

	r1 := &HealthReport{
		Errors: []ValidationError{
			{EntityType: "plan", EntityID: "P1-x", Field: "id", Message: "err"},
		},
		Summary: HealthSummary{
			TotalEntities:  1,
			EntitiesByType: map[string]int{"plan": 1},
		},
	}

	merged := MergeReports(nil, r1, nil)
	if merged.Summary.TotalEntities != 1 {
		t.Errorf("expected 1 total, got %d", merged.Summary.TotalEntities)
	}
	if len(merged.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(merged.Errors))
	}
}

// --- inferEntityType tests ---

func TestInferEntityType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		id   string
		want string
	}{
		{"P1-basic", string(model.EntityKindPlan)},
		{"FEAT-01AAAA", string(model.EntityKindFeature)},
		{"TASK-01AAAA", string(model.EntityKindTask)},
		{"BUG-01AAAA", string(model.EntityKindBug)},
		{"DEC-01AAAA", string(model.EntityKindDecision)},
		{"unknown-id", ""},
		{"", ""},
	}

	for _, tt := range tests {
		got := inferEntityType(tt.id)
		if got != tt.want {
			t.Errorf("inferEntityType(%q) = %q, want %q", tt.id, got, tt.want)
		}
	}
}
