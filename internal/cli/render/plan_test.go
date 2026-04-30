package render

import (
	"strings"
	"testing"
)

func TestRenderPlan(t *testing.T) {
	t.Parallel()

	t.Run("full plan TTY", func(t *testing.T) {
		r := NewRenderer(StaticTTY{Value: true})
		p := &PlanInput{
			DisplayID:   "P1-main-plan",
			ID:          "P1-main-plan",
			Slug:        "main-plan",
			Status:      "developing",
			TasksActive: 4,
			TasksReady:  6,
			TasksDone:   31,
			TasksTotal:  51,
			Features: []PlanFeatureInput{
				{DisplayID: "F-039", Slug: "spec-doc-gaps", Status: "done", HasDevPlan: true},
				{DisplayID: "F-040", Slug: "lifecycle-gate-hardening", Status: "done", HasDevPlan: true},
				{DisplayID: "F-041", Slug: "binary-rename", Status: "developing", HasDevPlan: true},
				{DisplayID: "F-042", Slug: "my-feature", Status: "developing", HasDevPlan: false},
				{DisplayID: "F-043", Slug: "cli-status-command", Status: "ready", HasDevPlan: true},
			},
			Attention: []AttentionItem{
				{Type: "missing_doc", Severity: "warning", EntityID: "FEAT-042", DisplayID: "F-042", Message: "F-042 my-feature: no dev-plan document"},
			},
		}

		var buf strings.Builder
		if err := r.RenderPlan(&buf, p); err != nil {
			t.Fatalf("RenderPlan: %v", err)
		}
		out := buf.String()

		// Header.
		if !strings.Contains(out, "P1-main-plan") {
			t.Errorf("missing plan display ID:\n%s", out)
		}
		if !strings.Contains(out, "main-plan") {
			t.Errorf("missing plan slug:\n%s", out)
		}
		// Features count.
		if !strings.Contains(out, "Features (5)") {
			t.Errorf("missing feature count:\n%s", out)
		}
		// Feature rows.
		for _, want := range []string{"spec-doc-gaps", "lifecycle-gate-hardening", "binary-rename", "my-feature", "cli-status-command"} {
			if !strings.Contains(out, want) {
				t.Errorf("missing feature %q:\n%s", want, out)
			}
		}
		// Tasks.
		if !strings.Contains(out, "51 total") {
			t.Errorf("missing total tasks:\n%s", out)
		}
		// Attention.
		if !strings.Contains(out, "⚠") {
			t.Errorf("missing warning symbol:\n%s", out)
		}
		if !strings.Contains(out, "no dev-plan document") {
			t.Errorf("missing attention message:\n%s", out)
		}
	})

	t.Run("full plan ASCII", func(t *testing.T) {
		r := NewRenderer(StaticTTY{Value: false})
		p := &PlanInput{
			DisplayID:   "P1-plan",
			Slug:        "my-plan",
			Status:      "active",
			TasksActive: 1,
			TasksDone:   2,
			TasksTotal:  3,
			Features: []PlanFeatureInput{
				{DisplayID: "F-001", Slug: "feat", Status: "done"},
			},
		}

		var buf strings.Builder
		if err := r.RenderPlan(&buf, p); err != nil {
			t.Fatalf("RenderPlan: %v", err)
		}
		out := buf.String()

		if !strings.Contains(out, "[*]") {
			t.Errorf("missing ASCII active symbol:\n%s", out)
		}
		if strings.Contains(out, "●") {
			t.Errorf("should not contain TTY unicode in ASCII mode:\n%s", out)
		}
	})

	t.Run("no features", func(t *testing.T) {
		r := NewRenderer(StaticTTY{Value: true})
		p := &PlanInput{
			DisplayID:   "P1-empty",
			Slug:        "empty",
			Status:      "ready",
			TasksActive: 0,
			TasksReady:  0,
			TasksDone:   0,
			TasksTotal:  0,
		}

		var buf strings.Builder
		if err := r.RenderPlan(&buf, p); err != nil {
			t.Fatalf("RenderPlan: %v", err)
		}
		out := buf.String()

		if !strings.Contains(out, "Features (0)") {
			t.Errorf("expected Features (0):\n%s", out)
		}
		if !strings.Contains(out, "0 total") {
			t.Errorf("expected 0 total:\n%s", out)
		}
	})

	t.Run("no attention omits block", func(t *testing.T) {
		r := NewRenderer(StaticTTY{Value: true})
		p := &PlanInput{
			DisplayID:   "P1-clean",
			Slug:        "clean",
			Status:      "done",
			TasksDone:   5,
			TasksTotal:  5,
			Features: []PlanFeatureInput{
				{DisplayID: "F-001", Slug: "feat", Status: "done"},
			},
		}

		var buf strings.Builder
		if err := r.RenderPlan(&buf, p); err != nil {
			t.Fatalf("RenderPlan: %v", err)
		}
		out := buf.String()

		if strings.Contains(out, "⚠") {
			t.Errorf("should not contain warning with no attention:\n%s", out)
		}
	})

	t.Run("feature status icons", func(t *testing.T) {
		r := NewRenderer(StaticTTY{Value: true})
		tests := []struct {
			status string
			want   string
		}{
			{"done", "✓"},
			{"developing", "●"},
			{"ready", "○"},
			{"reviewing", "●"},
			{"blocked", "✗"},
			{"unknown", "○"}, // default is ready icon
		}

		for _, tt := range tests {
			p := &PlanInput{
				DisplayID:   "P1-test",
				Slug:        "test",
				Status:      "active",
				TasksTotal:  0,
				Features: []PlanFeatureInput{
					{DisplayID: "F-001", Slug: "test", Status: tt.status},
				},
			}

			var buf strings.Builder
			if err := r.RenderPlan(&buf, p); err != nil {
				t.Fatalf("RenderPlan for status %q: %v", tt.status, err)
			}
			out := buf.String()

			if !strings.Contains(out, tt.want) {
				t.Errorf("status %q: expected icon %q in output:\n%s", tt.status, tt.want, out)
			}
		}
	})
}
