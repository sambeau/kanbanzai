package docint

import (
	"testing"
	"time"
)

// testIndex builds a DocumentIndex with known sections for testing.
func testIndex() *DocumentIndex {
	return &DocumentIndex{
		DocumentID:   "doc-test",
		DocumentPath: "work/design/test.md",
		ContentHash:  "abc123",
		IndexedAt:    time.Now(),
		Sections: []Section{
			{
				Path:  "1",
				Level: 1,
				Title: "Overview",
				Children: []Section{
					{Path: "1.1", Level: 2, Title: "Background"},
					{Path: "1.2", Level: 2, Title: "Goals"},
				},
			},
			{
				Path:  "2",
				Level: 1,
				Title: "Requirements",
				Children: []Section{
					{Path: "2.1", Level: 2, Title: "Functional"},
				},
			},
		},
	}
}

func testSubmission() ClassificationSubmission {
	return ClassificationSubmission{
		DocumentID:   "doc-test",
		ContentHash:  "abc123",
		ModelName:    "gpt-4",
		ModelVersion: "2024-01-01",
		ClassifiedAt: time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC),
		Classifications: []Classification{
			{SectionPath: "1", Role: "narrative", Confidence: "high"},
			{SectionPath: "1.1", Role: "rationale", Confidence: "medium"},
			{SectionPath: "2", Role: "requirement", Confidence: "high"},
		},
	}
}

func TestValidateClassifications_Valid(t *testing.T) {
	index := testIndex()
	sub := testSubmission()

	errs := ValidateClassifications(index, sub)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %d:", len(errs))
		for _, e := range errs {
			t.Errorf("  %v", e)
		}
	}
}

func TestValidateClassifications_HashMismatch(t *testing.T) {
	index := testIndex()
	sub := testSubmission()
	sub.ContentHash = "wrong-hash"

	errs := ValidateClassifications(index, sub)
	if len(errs) == 0 {
		t.Fatal("expected error for hash mismatch, got none")
	}

	found := false
	for _, e := range errs {
		if contains(e.Error(), "content hash mismatch") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected content hash mismatch error, got: %v", errs)
	}
}

func TestValidateClassifications_MissingModel(t *testing.T) {
	index := testIndex()
	sub := testSubmission()
	sub.ModelName = ""

	errs := ValidateClassifications(index, sub)
	if len(errs) == 0 {
		t.Fatal("expected error for missing model_name, got none")
	}

	found := false
	for _, e := range errs {
		if contains(e.Error(), "model_name is required") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected model_name required error, got: %v", errs)
	}
}

func TestValidateClassifications_MissingModelVersion(t *testing.T) {
	index := testIndex()
	sub := testSubmission()
	sub.ModelVersion = ""

	errs := ValidateClassifications(index, sub)
	if len(errs) == 0 {
		t.Fatal("expected error for missing model_version, got none")
	}

	found := false
	for _, e := range errs {
		if contains(e.Error(), "model_version is required") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected model_version required error, got: %v", errs)
	}
}

func TestValidateClassifications_InvalidRole(t *testing.T) {
	index := testIndex()
	sub := testSubmission()
	sub.Classifications = []Classification{
		{SectionPath: "1", Role: "bogus-role", Confidence: "high"},
	}

	errs := ValidateClassifications(index, sub)
	if len(errs) == 0 {
		t.Fatal("expected error for invalid role, got none")
	}

	found := false
	for _, e := range errs {
		if contains(e.Error(), "invalid fragment role") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected invalid fragment role error, got: %v", errs)
	}
}

func TestValidateClassifications_InvalidConfidence(t *testing.T) {
	index := testIndex()
	sub := testSubmission()
	sub.Classifications = []Classification{
		{SectionPath: "1", Role: "narrative", Confidence: "maybe"},
	}

	errs := ValidateClassifications(index, sub)
	if len(errs) == 0 {
		t.Fatal("expected error for invalid confidence, got none")
	}

	found := false
	for _, e := range errs {
		if contains(e.Error(), "invalid confidence") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected invalid confidence error, got: %v", errs)
	}
}

func TestValidateClassifications_UnknownSection(t *testing.T) {
	index := testIndex()
	sub := testSubmission()
	sub.Classifications = []Classification{
		{SectionPath: "99.99", Role: "narrative", Confidence: "high"},
	}

	errs := ValidateClassifications(index, sub)
	if len(errs) == 0 {
		t.Fatal("expected error for unknown section, got none")
	}

	found := false
	for _, e := range errs {
		if contains(e.Error(), "unknown section_path") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected unknown section_path error, got: %v", errs)
	}
}

func TestValidateClassifications_MultipleErrors(t *testing.T) {
	index := testIndex()
	sub := ClassificationSubmission{
		DocumentID:   "doc-test",
		ContentHash:  "wrong-hash",
		ModelName:    "",
		ModelVersion: "",
		ClassifiedAt: time.Now(),
		Classifications: []Classification{
			{SectionPath: "99", Role: "bogus", Confidence: "maybe"},
		},
	}

	errs := ValidateClassifications(index, sub)
	// Expect at least: hash mismatch, missing model_name, missing model_version,
	// unknown section, invalid role (or invalid confidence — ValidateClassification
	// returns the first error it finds per classification).
	if len(errs) < 4 {
		t.Errorf("expected at least 4 errors, got %d:", len(errs))
		for _, e := range errs {
			t.Errorf("  %v", e)
		}
	}
}

func TestApplyClassifications(t *testing.T) {
	index := testIndex()
	sub := testSubmission()

	if index.Classified {
		t.Fatal("precondition: index should not be classified yet")
	}
	if index.ClassifiedAt != nil {
		t.Fatal("precondition: ClassifiedAt should be nil")
	}

	ApplyClassifications(index, sub)

	if !index.Classified {
		t.Error("expected Classified to be true")
	}
	if index.ClassifiedAt == nil {
		t.Fatal("expected ClassifiedAt to be set")
	}
	if !index.ClassifiedAt.Equal(sub.ClassifiedAt) {
		t.Errorf("ClassifiedAt = %v, want %v", *index.ClassifiedAt, sub.ClassifiedAt)
	}
	if index.ClassifiedBy != "gpt-4" {
		t.Errorf("ClassifiedBy = %q, want %q", index.ClassifiedBy, "gpt-4")
	}
	if index.ClassifierVersion != "2024-01-01" {
		t.Errorf("ClassifierVersion = %q, want %q", index.ClassifierVersion, "2024-01-01")
	}
	if len(index.Classifications) != 3 {
		t.Fatalf("expected 3 classifications, got %d", len(index.Classifications))
	}
	if index.Classifications[0].SectionPath != "1" {
		t.Errorf("first classification section_path = %q, want %q", index.Classifications[0].SectionPath, "1")
	}
}

func TestApplyClassifications_OverwritesExisting(t *testing.T) {
	index := testIndex()
	first := testSubmission()
	ApplyClassifications(index, first)

	second := ClassificationSubmission{
		DocumentID:   "doc-test",
		ContentHash:  "abc123",
		ModelName:    "claude-4",
		ModelVersion: "2025-06-01",
		ClassifiedAt: time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
		Classifications: []Classification{
			{SectionPath: "1", Role: "definition", Confidence: "low"},
		},
	}
	ApplyClassifications(index, second)

	if index.ClassifiedBy != "claude-4" {
		t.Errorf("ClassifiedBy = %q, want %q", index.ClassifiedBy, "claude-4")
	}
	if len(index.Classifications) != 1 {
		t.Errorf("expected 1 classification after overwrite, got %d", len(index.Classifications))
	}
}

func TestCollectSectionPaths(t *testing.T) {
	index := testIndex()
	paths := collectSectionPaths(index.Sections)

	expected := []string{"1", "1.1", "1.2", "2", "2.1"}
	for _, p := range expected {
		if _, ok := paths[p]; !ok {
			t.Errorf("expected path %q in collected paths", p)
		}
	}
	if len(paths) != len(expected) {
		t.Errorf("expected %d paths, got %d", len(expected), len(paths))
	}
}

func TestCollectSectionPaths_Empty(t *testing.T) {
	paths := collectSectionPaths(nil)
	if len(paths) != 0 {
		t.Errorf("expected 0 paths for nil sections, got %d", len(paths))
	}
}

// contains reports whether s contains substr.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
