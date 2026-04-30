package render

import (
	"strings"
	"testing"
)

func TestRenderProject(t *testing.T) {
	t.Parallel()

	t.Run("full project TTY", func(t *testing.T) {
		r := NewRenderer(StaticTTY{Value: true})
		p := &ProjectInput{
			Name: "my-project",
			Plans: []ProjectPlanInput{
				{DisplayID: "P1-main-plan", Status: "developing", FeaturesActive: 4, FeaturesTotal: 5},
				{DisplayID: "P2-infrastructure", Status: "ready", FeaturesActive: 0, FeaturesTotal: 3},
			},
			Health: &StatusHealthSummary{Errors: 0, Warnings: 2},
			Attention: []AttentionItem{
				{Type: "missing_doc", Severity: "warning", EntityID: "FEAT-042", DisplayID: "F-042", Message: "F-042 my-feature: no dev-plan document"},
				{Type: "doc_draft", Severity: "warning", EntityID: "DOC-0031", DisplayID: "DOC-0031", Message: "DOC-0031 work/design/new-draft.md: draft — not yet approved"},
			},
			WorkQueue: ProjectWorkQueue{Ready: 6, Active: 2},
		}

		var buf strings.Builder
		if err := r.RenderProject(&buf, p); err != nil {
			t.Fatalf("RenderProject: %v", err)
		}
		out := buf.String()

		// Header.
		if !strings.Contains(out, "Kanbanzai") {
			t.Errorf("missing Kanbanzai header:\n%s", out)
		}
		if !strings.Contains(out, "my-project") {
			t.Errorf("missing project name:\n%s", out)
		}
		// Plans.
		if !strings.Contains(out, "Plans (2)") {
			t.Errorf("missing plans count:\n%s", out)
		}
		if !strings.Contains(out, "P1-main-plan") {
			t.Errorf("missing first plan:\n%s", out)
		}
		if !strings.Contains(out, "P2-infrastructure") {
			t.Errorf("missing second plan:\n%s", out)
		}
		if !strings.Contains(out, "4 features active") {
			t.Errorf("missing features active for plan 1:\n%s", out)
		}
		if !strings.Contains(out, "0 features started") {
			t.Errorf("missing 0 features started for plan 2:\n%s", out)
		}
		// Health.
		if !strings.Contains(out, "✓") {
			t.Errorf("missing ok symbol in health:\n%s", out)
		}
		if !strings.Contains(out, "2 warnings") {
			t.Errorf("missing warning count:\n%s", out)
		}
		// Attention.
		if !strings.Contains(out, "⚠") {
			t.Errorf("missing warning symbol:\n%s", out)
		}
		// Work queue.
		if !strings.Contains(out, "6 ready") {
			t.Errorf("missing ready count:\n%s", out)
		}
		if !strings.Contains(out, "2 active") {
			t.Errorf("missing active count:\n%s", out)
		}
	})

	t.Run("full project ASCII", func(t *testing.T) {
		r := NewRenderer(StaticTTY{Value: false})
		p := &ProjectInput{
			Name: "test",
			Plans: []ProjectPlanInput{
				{DisplayID: "P1-plan", Status: "done", FeaturesActive: 0, FeaturesTotal: 3},
			},
			Health:    &StatusHealthSummary{Errors: 0, Warnings: 0},
			WorkQueue: ProjectWorkQueue{Ready: 1, Active: 0},
		}

		var buf strings.Builder
		if err := r.RenderProject(&buf, p); err != nil {
			t.Fatalf("RenderProject: %v", err)
		}
		out := buf.String()

		if !strings.Contains(out, "[ok]") {
			t.Errorf("missing ASCII ok symbol:\n%s", out)
		}
		if strings.Contains(out, "✓") {
			t.Errorf("should not contain TTY unicode in ASCII:\n%s", out)
		}
		if !strings.Contains(out, "ready") {
			t.Errorf("missing ready text in work queue:\n%s", out)
		}
	})

	t.Run("no attention omits block", func(t *testing.T) {
		r := NewRenderer(StaticTTY{Value: true})
		p := &ProjectInput{
			Name: "quiet",
			Plans: []ProjectPlanInput{
				{DisplayID: "P1-plan", Status: "done", FeaturesActive: 0, FeaturesTotal: 1},
			},
			Health:    &StatusHealthSummary{Errors: 0, Warnings: 0},
			WorkQueue: ProjectWorkQueue{Ready: 0, Active: 0},
		}

		var buf strings.Builder
		if err := r.RenderProject(&buf, p); err != nil {
			t.Fatalf("RenderProject: %v", err)
		}
		out := buf.String()

		if strings.Contains(out, "⚠") {
			t.Errorf("should not contain warning with no attention:\n%s", out)
		}
	})

	t.Run("no plans", func(t *testing.T) {
		r := NewRenderer(StaticTTY{Value: true})
		p := &ProjectInput{
			Name:      "empty",
			Health:    &StatusHealthSummary{Errors: 0, Warnings: 0},
			WorkQueue: ProjectWorkQueue{Ready: 0, Active: 0},
		}

		var buf strings.Builder
		if err := r.RenderProject(&buf, p); err != nil {
			t.Fatalf("RenderProject: %v", err)
		}
		out := buf.String()

		if !strings.Contains(out, "Plans (0)") {
			t.Errorf("expected Plans (0):\n%s", out)
		}
		// Health and work queue still shown.
		if !strings.Contains(out, "Health") {
			t.Errorf("missing Health section:\n%s", out)
		}
		if !strings.Contains(out, "Work queue") {
			t.Errorf("missing Work queue section:\n%s", out)
		}
	})

	t.Run("health errors", func(t *testing.T) {
		r := NewRenderer(StaticTTY{Value: true})
		p := &ProjectInput{
			Name:      "broken",
			Health:    &StatusHealthSummary{Errors: 3, Warnings: 1},
			WorkQueue: ProjectWorkQueue{Ready: 0, Active: 0},
		}

		var buf strings.Builder
		if err := r.RenderProject(&buf, p); err != nil {
			t.Fatalf("RenderProject: %v", err)
		}
		out := buf.String()

		// Should show ✗ for errors.
		if !strings.Contains(out, "✗") {
			t.Errorf("missing error symbol:\n%s", out)
		}
		if !strings.Contains(out, "3 errors") {
			t.Errorf("missing error count:\n%s", out)
		}
		if !strings.Contains(out, "1 warnings") {
			t.Errorf("missing warning count:\n%s", out)
		}
	})

	t.Run("health warnings only", func(t *testing.T) {
		r := NewRenderer(StaticTTY{Value: true})
		p := &ProjectInput{
			Name:      "warny",
			Health:    &StatusHealthSummary{Errors: 0, Warnings: 5},
			WorkQueue: ProjectWorkQueue{Ready: 0, Active: 0},
		}

		var buf strings.Builder
		if err := r.RenderProject(&buf, p); err != nil {
			t.Fatalf("RenderProject: %v", err)
		}
		out := buf.String()

		if !strings.Contains(out, "no errors") {
			t.Errorf("missing no errors:\n%s", out)
		}
		if !strings.Contains(out, "5 warnings") {
			t.Errorf("missing warning count:\n%s", out)
		}
	})

	t.Run("nil health defaults to zero", func(t *testing.T) {
		r := NewRenderer(StaticTTY{Value: true})
		p := &ProjectInput{
			Name:      "nilhealth",
			Health:    nil,
			WorkQueue: ProjectWorkQueue{Ready: 0, Active: 0},
		}

		var buf strings.Builder
		if err := r.RenderProject(&buf, p); err != nil {
			t.Fatalf("RenderProject: %v", err)
		}
		out := buf.String()

		if !strings.Contains(out, "no errors") {
			t.Errorf("nil health should default to no errors:\n%s", out)
		}
		if !strings.Contains(out, "no warnings") {
			t.Errorf("nil health should default to no warnings:\n%s", out)
		}
	})
}
