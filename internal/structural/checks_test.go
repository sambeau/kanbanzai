package structural

import (
	"testing"

	"github.com/sambeau/kanbanzai/internal/docint"
)

func TestCheckRequiredSections_AllPresent(t *testing.T) {
	t.Parallel()

	sections := []docint.Section{
		{Level: 1, Title: "My Design", Path: "1"},
		{Level: 2, Title: "Overview", Path: "2"},
		{Level: 2, Title: "Design", Path: "3"},
	}

	result := CheckRequiredSections(sections, "design", "DOC-001", "gate-1")
	if !result.Passed {
		t.Errorf("Passed = false, want true; Details = %v", result.Details)
	}
	if result.CheckType != "required_sections" {
		t.Errorf("CheckType = %q, want required_sections", result.CheckType)
	}
}

func TestCheckRequiredSections_MissingSections(t *testing.T) {
	t.Parallel()

	sections := []docint.Section{
		{Level: 1, Title: "My Design", Path: "1"},
		{Level: 2, Title: "Overview", Path: "2"},
	}

	result := CheckRequiredSections(sections, "design", "DOC-001", "gate-1")
	if result.Passed {
		t.Error("Passed = true, want false")
	}
	if len(result.Details) == 0 {
		t.Error("Details should be non-empty when check fails")
	}
}

func TestCheckRequiredSections_UnknownDocType(t *testing.T) {
	t.Parallel()

	result := CheckRequiredSections(nil, "research", "DOC-001", "gate-1")
	if !result.Passed {
		t.Error("Passed = false for unknown type, want true")
	}
}

func TestCheckRequiredSections_ExcludesH1(t *testing.T) {
	t.Parallel()

	sections := []docint.Section{
		{Level: 1, Title: "Overview and Design", Path: "1"},
		{Level: 2, Title: "Design", Path: "2"},
	}

	result := CheckRequiredSections(sections, "design", "DOC-001", "gate-1")
	if result.Passed {
		t.Error("Passed = true, but H1 should not satisfy section requirements")
	}
}

func TestCheckRequiredSections_CaseInsensitive(t *testing.T) {
	t.Parallel()

	sections := []docint.Section{
		{Level: 1, Title: "My Design", Path: "1"},
		{Level: 2, Title: "OVERVIEW", Path: "2"},
		{Level: 2, Title: "Design Details", Path: "3"},
	}

	result := CheckRequiredSections(sections, "design", "DOC-001", "gate-1")
	if !result.Passed {
		t.Errorf("Passed = false, want true (case-insensitive); Details = %v", result.Details)
	}
}

func TestCheckRequiredSections_Specification(t *testing.T) {
	t.Parallel()

	sections := []docint.Section{
		{Level: 1, Title: "Spec", Path: "1"},
		{Level: 2, Title: "Overview", Path: "2"},
		{Level: 2, Title: "Scope", Path: "3"},
		{Level: 2, Title: "Functional Requirements", Path: "4"},
		{Level: 2, Title: "Acceptance Criteria", Path: "5"},
	}

	result := CheckRequiredSections(sections, "specification", "DOC-002", "gate-1")
	if !result.Passed {
		t.Errorf("Passed = false, want true; Details = %v", result.Details)
	}
}

func TestCheckCrossReference_Found(t *testing.T) {
	t.Parallel()

	extractResult := docint.ExtractResult{
		CrossDocLinks: []docint.CrossDocLink{
			{TargetPath: "work/design/foo.md", LinkText: "Foo Design"},
		},
	}

	result := CheckCrossReference(extractResult, []string{"work/design/foo.md"}, nil, "DOC-002", "gate-1")
	if !result.Passed {
		t.Errorf("Passed = false, want true; Details = %v", result.Details)
	}
}

func TestCheckCrossReference_FoundViaEntityRef(t *testing.T) {
	t.Parallel()

	extractResult := docint.ExtractResult{
		EntityRefs: []docint.EntityRef{
			{EntityID: "DOC-abc123", EntityType: "document"},
		},
	}

	result := CheckCrossReference(extractResult, nil, []string{"DOC-abc123"}, "DOC-002", "gate-1")
	if !result.Passed {
		t.Errorf("Passed = false, want true; Details = %v", result.Details)
	}
}

func TestCheckCrossReference_NotFound(t *testing.T) {
	t.Parallel()

	extractResult := docint.ExtractResult{
		CrossDocLinks: []docint.CrossDocLink{
			{TargetPath: "work/design/other.md"},
		},
	}

	result := CheckCrossReference(extractResult, []string{"work/design/foo.md"}, nil, "DOC-002", "gate-1")
	if result.Passed {
		t.Error("Passed = true, want false")
	}
	if len(result.Details) == 0 {
		t.Error("Details should be non-empty")
	}
}

func TestCheckCrossReference_NoneProvided(t *testing.T) {
	t.Parallel()

	extractResult := docint.ExtractResult{}
	result := CheckCrossReference(extractResult, nil, nil, "DOC-002", "gate-1")
	if result.Passed {
		t.Error("Passed = true with no references and no approved docs, want false")
	}
}

func TestCheckAcceptanceCriteria_FoundInConventionalRoles(t *testing.T) {
	t.Parallel()

	sections := []docint.Section{
		{Level: 1, Title: "Spec", Path: "1"},
		{Level: 2, Title: "Acceptance Criteria", Path: "1.1"},
	}
	roles := []docint.ConventionalRole{
		{SectionPath: "1.1", Role: "requirement", Confidence: "high"},
	}

	result := CheckAcceptanceCriteria(sections, roles, "DOC-003", "gate-1")
	if !result.Passed {
		t.Errorf("Passed = false, want true; Details = %v", result.Details)
	}
}

func TestCheckAcceptanceCriteria_FoundViaHeadingFallback(t *testing.T) {
	t.Parallel()

	sections := []docint.Section{
		{Level: 1, Title: "Spec", Path: "1"},
		{Level: 2, Title: "Acceptance Criteria", Path: "1.1"},
	}

	result := CheckAcceptanceCriteria(sections, nil, "DOC-003", "gate-1")
	if !result.Passed {
		t.Errorf("Passed = false, want true via heading fallback; Details = %v", result.Details)
	}
}

func TestCheckAcceptanceCriteria_NotFound(t *testing.T) {
	t.Parallel()

	sections := []docint.Section{
		{Level: 1, Title: "Spec", Path: "1"},
		{Level: 2, Title: "Requirements", Path: "1.1"},
	}
	roles := []docint.ConventionalRole{
		{SectionPath: "1.1", Role: "requirement", Confidence: "high"},
	}

	result := CheckAcceptanceCriteria(sections, roles, "DOC-003", "gate-1")
	if result.Passed {
		t.Error("Passed = true, want false (requirements != acceptance criteria)")
	}
	if len(result.Details) == 0 {
		t.Error("Details should be non-empty")
	}
}
