package structural

import (
	"testing"
)

func TestRequiredSections_Design(t *testing.T) {
	t.Parallel()

	sections := RequiredSections("design")
	if len(sections) != 2 {
		t.Fatalf("len(sections) = %d, want 2", len(sections))
	}

	if sections[0].Label != "overview/purpose/summary" {
		t.Errorf("sections[0].Label = %q, want %q", sections[0].Label, "overview/purpose/summary")
	}
	if len(sections[0].Keywords) != 3 {
		t.Errorf("len(sections[0].Keywords) = %d, want 3", len(sections[0].Keywords))
	}

	if sections[1].Label != "design" {
		t.Errorf("sections[1].Label = %q, want %q", sections[1].Label, "design")
	}
	if len(sections[1].Keywords) != 1 {
		t.Errorf("len(sections[1].Keywords) = %d, want 1", len(sections[1].Keywords))
	}
}

func TestRequiredSections_Specification(t *testing.T) {
	t.Parallel()

	sections := RequiredSections("specification")
	if len(sections) != 4 {
		t.Fatalf("len(sections) = %d, want 4", len(sections))
	}

	labels := []string{"overview", "scope", "functional requirements", "acceptance criteria"}
	for i, want := range labels {
		if sections[i].Label != want {
			t.Errorf("sections[%d].Label = %q, want %q", i, sections[i].Label, want)
		}
	}
}

func TestRequiredSections_DevPlan(t *testing.T) {
	t.Parallel()

	sections := RequiredSections("dev-plan")
	if len(sections) != 2 {
		t.Fatalf("len(sections) = %d, want 2", len(sections))
	}

	if sections[0].Label != "overview" {
		t.Errorf("sections[0].Label = %q, want %q", sections[0].Label, "overview")
	}
	if sections[1].Label != "task" {
		t.Errorf("sections[1].Label = %q, want %q", sections[1].Label, "task")
	}
}

func TestRequiredSections_UnknownType(t *testing.T) {
	t.Parallel()

	for _, docType := range []string{"research", "report", "policy", "rca", "", "unknown"} {
		sections := RequiredSections(docType)
		if sections != nil {
			t.Errorf("RequiredSections(%q) = %v, want nil", docType, sections)
		}
	}
}

func TestRequiredSections_Keywords(t *testing.T) {
	t.Parallel()

	// Verify keyword contents for specification
	sections := RequiredSections("specification")
	tests := []struct {
		idx      int
		keywords []string
	}{
		{0, []string{"overview"}},
		{1, []string{"scope"}},
		{2, []string{"functional requirements"}},
		{3, []string{"acceptance criteria"}},
	}
	for _, tt := range tests {
		got := sections[tt.idx].Keywords
		if len(got) != len(tt.keywords) {
			t.Errorf("sections[%d].Keywords len = %d, want %d", tt.idx, len(got), len(tt.keywords))
			continue
		}
		for j, kw := range tt.keywords {
			if got[j] != kw {
				t.Errorf("sections[%d].Keywords[%d] = %q, want %q", tt.idx, j, got[j], kw)
			}
		}
	}
}

func TestRequiredSections_DevPlanKeywords(t *testing.T) {
	t.Parallel()

	sections := RequiredSections("dev-plan")
	if sections[0].Keywords[0] != "overview" {
		t.Errorf("dev-plan sections[0].Keywords[0] = %q, want %q", sections[0].Keywords[0], "overview")
	}
	if sections[1].Keywords[0] != "task" {
		t.Errorf("dev-plan sections[1].Keywords[0] = %q, want %q", sections[1].Keywords[0], "task")
	}
}
