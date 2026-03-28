package service

import (
	"strings"
	"testing"

	"github.com/sambeau/kanbanzai/internal/model"
)

func TestCreatePlan_Success(t *testing.T) {
	// CreatePlan uses LoadOrDefault(), so it works even without .kbz/config.yaml.
	// The default config includes prefix "P" for Plan.
	root := t.TempDir()
	svc := NewEntityService(root)

	result, err := svc.CreatePlan(CreatePlanInput{
		Prefix:    "P",
		Slug:      "test-plan",
		Title:     "Test Plan",
		Summary:   "A test plan for unit testing",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreatePlan() error = %v", err)
	}

	if result.Type != string(model.EntityKindPlan) {
		t.Errorf("Type = %q, want %q", result.Type, model.EntityKindPlan)
	}
	if !strings.HasPrefix(result.ID, "P") {
		t.Errorf("ID = %q, should start with P", result.ID)
	}
	if result.Slug != "test-plan" {
		t.Errorf("Slug = %q, want %q", result.Slug, "test-plan")
	}
	if result.State["status"] != string(model.PlanStatusProposed) {
		t.Errorf("status = %q, want %q", result.State["status"], model.PlanStatusProposed)
	}
}

func TestCreatePlan_WithTags(t *testing.T) {
	root := t.TempDir()
	svc := NewEntityService(root)

	result, err := svc.CreatePlan(CreatePlanInput{
		Prefix:    "P",
		Slug:      "tagged-plan",
		Title:     "Tagged Plan",
		Summary:   "A test plan with tags",
		CreatedBy: "tester",
		Tags:      []string{"test", "phase:2"},
	})
	if err != nil {
		t.Fatalf("CreatePlan() error = %v", err)
	}

	if result.Type != string(model.EntityKindPlan) {
		t.Errorf("Type = %q, want %q", result.Type, model.EntityKindPlan)
	}
	if !strings.HasPrefix(result.ID, "P") {
		t.Errorf("ID = %q, should start with P", result.ID)
	}
	if result.Slug != "tagged-plan" {
		t.Errorf("Slug = %q, want %q", result.Slug, "tagged-plan")
	}
	if result.State["status"] != string(model.PlanStatusProposed) {
		t.Errorf("status = %q, want %q", result.State["status"], model.PlanStatusProposed)
	}
}

func TestCreatePlan_UndeclaredPrefix(t *testing.T) {
	root := t.TempDir()
	svc := NewEntityService(root)

	// Default config only has prefix "P". Using "Z" should fail.
	_, err := svc.CreatePlan(CreatePlanInput{
		Prefix:    "Z",
		Slug:      "test-plan",
		Title:     "Test Plan",
		Summary:   "A test plan",
		CreatedBy: "tester",
	})
	if err == nil {
		t.Fatal("expected error for undeclared prefix, got nil")
	}
	if !strings.Contains(err.Error(), "undeclared prefix") {
		t.Errorf("error = %q, want to contain 'undeclared prefix'", err.Error())
	}
}

func TestCreatePlan_MissingRequiredFields(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := NewEntityService(root)

	testCases := []struct {
		name  string
		input CreatePlanInput
		want  string
	}{
		{
			name: "missing prefix",
			input: CreatePlanInput{
				Slug:      "test",
				Title:     "Test",
				Summary:   "Summary",
				CreatedBy: "tester",
			},
			want: "prefix is required",
		},
		{
			name: "missing slug",
			input: CreatePlanInput{
				Prefix:    "P",
				Title:     "Test",
				Summary:   "Summary",
				CreatedBy: "tester",
			},
			want: "slug is required",
		},
		{
			name: "missing title",
			input: CreatePlanInput{
				Prefix:    "P",
				Slug:      "test",
				Summary:   "Summary",
				CreatedBy: "tester",
			},
			want: "title is required",
		},
		{
			name: "missing summary",
			input: CreatePlanInput{
				Prefix:    "P",
				Slug:      "test",
				Title:     "Test",
				CreatedBy: "tester",
			},
			want: "summary is required",
		},
		{
			name: "missing created_by",
			input: CreatePlanInput{
				Prefix:  "P",
				Slug:    "test",
				Title:   "Test",
				Summary: "Summary",
			},
			want: "created_by is required",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := svc.CreatePlan(tc.input)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tc.want)
			}
		})
	}
}

func TestGetPlan_InvalidIDFormat(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := NewEntityService(root)

	_, err := svc.GetPlan("FEAT-123")
	if err == nil {
		t.Fatal("expected error for invalid Plan ID format")
	}
	if !strings.Contains(err.Error(), "invalid Plan ID format") {
		t.Errorf("error = %q, want to contain 'invalid Plan ID format'", err.Error())
	}
}

func TestIsPlanID(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		id   string
		want bool
	}{
		{"P1-basic", true},
		{"P12-multi-word-slug", true},
		{"X99-test", true},
		{"α1-unicode-prefix", true},
		{"P1-a", true},
		{"FEAT-123", false},    // Not a Plan ID
		{"TASK-abc", false},    // Not a Plan ID
		{"1P-invalid", false},  // Starts with digit
		{"P-no-number", false}, // No number
		{"P1", false},          // No slug (no hyphen after number)
		{"P1-", false},         // Empty slug
		{"", false},            // Empty
		{"P", false},           // Too short
		{"PP1-double", false},  // Double prefix char
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.id, func(t *testing.T) {
			t.Parallel()
			got := model.IsPlanID(tc.id)
			if got != tc.want {
				t.Errorf("IsPlanID(%q) = %v, want %v", tc.id, got, tc.want)
			}
		})
	}
}

func TestParsePlanID(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		id         string
		wantPrefix string
		wantNumber string
		wantSlug   string
	}{
		{"P1-basic", "P", "1", "basic"},
		{"P12-multi-word-slug", "P", "12", "multi-word-slug"},
		{"X99-test", "X", "99", "test"},
		{"P1-a", "P", "1", "a"},
		{"FEAT-123", "", "", ""}, // Invalid
		{"", "", "", ""},         // Empty
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.id, func(t *testing.T) {
			t.Parallel()
			prefix, number, slug := model.ParsePlanID(tc.id)
			if prefix != tc.wantPrefix {
				t.Errorf("prefix = %q, want %q", prefix, tc.wantPrefix)
			}
			if number != tc.wantNumber {
				t.Errorf("number = %q, want %q", number, tc.wantNumber)
			}
			if slug != tc.wantSlug {
				t.Errorf("slug = %q, want %q", slug, tc.wantSlug)
			}
		})
	}
}

func TestNormalizeTags(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		tags []string
		want []string
	}{
		{
			name: "lowercase conversion",
			tags: []string{"Phase:2", "URGENT", "Test"},
			want: []string{"phase:2", "urgent", "test"},
		},
		{
			name: "trim whitespace",
			tags: []string{"  spaced  ", "normal"},
			want: []string{"spaced", "normal"},
		},
		{
			name: "remove duplicates",
			tags: []string{"test", "Test", "TEST"},
			want: []string{"test"},
		},
		{
			name: "remove empty",
			tags: []string{"valid", "", "  ", "also-valid"},
			want: []string{"valid", "also-valid"},
		},
		{
			name: "nil input",
			tags: nil,
			want: nil,
		},
		{
			name: "empty input",
			tags: []string{},
			want: nil,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := normalizeTags(tc.tags)
			if len(got) != len(tc.want) {
				t.Fatalf("len = %d, want %d; got %v, want %v", len(got), len(tc.want), got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("tag[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestMatchesPlanFilters(t *testing.T) {
	t.Parallel()

	result := ListResult{
		ID: "P1-test",
		State: map[string]any{
			"status": "active",
			"tags":   []any{"phase:2", "urgent"},
		},
	}

	testCases := []struct {
		name    string
		filters PlanFilters
		want    bool
	}{
		{
			name:    "no filters",
			filters: PlanFilters{},
			want:    true,
		},
		{
			name:    "matching status",
			filters: PlanFilters{Status: "active"},
			want:    true,
		},
		{
			name:    "non-matching status",
			filters: PlanFilters{Status: "proposed"},
			want:    false,
		},
		{
			name:    "matching prefix",
			filters: PlanFilters{Prefix: "P"},
			want:    true,
		},
		{
			name:    "non-matching prefix",
			filters: PlanFilters{Prefix: "X"},
			want:    false,
		},
		{
			name:    "matching single tag",
			filters: PlanFilters{Tags: []string{"urgent"}},
			want:    true,
		},
		{
			name:    "matching multiple tags",
			filters: PlanFilters{Tags: []string{"phase:2", "urgent"}},
			want:    true,
		},
		{
			name:    "non-matching tag",
			filters: PlanFilters{Tags: []string{"missing"}},
			want:    false,
		},
		{
			name:    "partial tag match fails",
			filters: PlanFilters{Tags: []string{"urgent", "missing"}},
			want:    false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := matchesPlanFilters(result, tc.filters)
			if got != tc.want {
				t.Errorf("matchesPlanFilters() = %v, want %v", got, tc.want)
			}
		})
	}
}
