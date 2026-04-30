package render

import (
	"strings"
	"testing"
)

func TestRenderFeature(t *testing.T) {
	t.Parallel()

	t.Run("full feature with all documents TTY", func(t *testing.T) {
		r := NewRenderer(StaticTTY{Value: true})
		f := &FeatureInput{
			DisplayID:   "F-042",
			ID:          "FEAT-01KQ2VHKJB5V8",
			Slug:        "my-feature",
			Status:      "developing",
			PlanID:      "P1-my-plan",
			PlanName:    "main-plan",
			TasksActive: 1,
			TasksReady:  3,
			TasksDone:   7,
			TasksTotal:  11,
			Documents: []DocInput{
				{Type: "design", Path: "work/design/my-feature.md", Status: "approved"},
				{Type: "specification", Path: "work/spec/my-feature-spec.md", Status: "approved"},
				{Type: "dev-plan", Path: "work/plan/my-feature-plan.md", Status: "approved"},
			},
		}

		var buf strings.Builder
		if err := r.RenderFeature(&buf, f); err != nil {
			t.Fatalf("RenderFeature: %v", err)
		}
		out := buf.String()

		// Header.
		if !strings.Contains(out, "F-042") {
			t.Errorf("missing display ID in output:\n%s", out)
		}
		if !strings.Contains(out, "my-feature") {
			t.Errorf("missing slug in output:\n%s", out)
		}
		// Plan line.
		if !strings.Contains(out, "P1-my-plan") {
			t.Errorf("missing plan ID in output:\n%s", out)
		}
		if !strings.Contains(out, "main-plan") {
			t.Errorf("missing plan name in output:\n%s", out)
		}
		// Documents.
		if !strings.Contains(out, "work/design/my-feature.md") {
			t.Errorf("missing design doc in output:\n%s", out)
		}
		if !strings.Contains(out, "work/spec/my-feature-spec.md") {
			t.Errorf("missing spec doc in output:\n%s", out)
		}
		if !strings.Contains(out, "work/plan/my-feature-plan.md") {
			t.Errorf("missing dev-plan doc in output:\n%s", out)
		}
		// TTY symbols.
		if !strings.Contains(out, "✓") {
			t.Errorf("missing TTY ok symbol:\n%s", out)
		}
		// Tasks.
		if !strings.Contains(out, "11 total") {
			t.Errorf("missing task total:\n%s", out)
		}
	})

	t.Run("full feature with all documents ASCII", func(t *testing.T) {
		r := NewRenderer(StaticTTY{Value: false})
		f := &FeatureInput{
			DisplayID:   "F-042",
			ID:          "FEAT-01KQ2VHKJB5V8",
			Slug:        "my-feature",
			Status:      "developing",
			PlanID:      "P1-my-plan",
			TasksActive: 1,
			TasksReady:  3,
			TasksDone:   7,
			TasksTotal:  11,
			Documents: []DocInput{
				{Type: "design", Path: "work/design/my-feature.md", Status: "approved"},
				{Type: "specification", Path: "work/spec/my-feature-spec.md", Status: "approved"},
				{Type: "dev-plan", Path: "work/plan/my-feature-plan.md", Status: "approved"},
			},
		}

		var buf strings.Builder
		if err := r.RenderFeature(&buf, f); err != nil {
			t.Fatalf("RenderFeature: %v", err)
		}
		out := buf.String()

		if !strings.Contains(out, "[ok]") {
			t.Errorf("missing ASCII ok symbol:\n%s", out)
		}
		if strings.Contains(out, "✓") {
			t.Errorf("should not contain TTY unicode in ASCII mode:\n%s", out)
		}
	})

	t.Run("no plan omits Plan line", func(t *testing.T) {
		r := NewRenderer(StaticTTY{Value: true})
		f := &FeatureInput{
			DisplayID:   "F-001",
			Slug:        "orphan",
			Status:      "ready",
			TasksActive: 0,
			TasksReady:  0,
			TasksDone:   0,
			TasksTotal:  0,
		}

		var buf strings.Builder
		if err := r.RenderFeature(&buf, f); err != nil {
			t.Fatalf("RenderFeature: %v", err)
		}
		out := buf.String()

		if strings.Contains(out, "Plan:") {
			t.Errorf("should not contain Plan line when no plan:\n%s", out)
		}
	})

	t.Run("missing docs show all three with missing marks", func(t *testing.T) {
		r := NewRenderer(StaticTTY{Value: true})
		f := &FeatureInput{
			DisplayID:   "F-002",
			Slug:        "docless",
			Status:      "specifying",
			PlanID:      "P1-plan",
			TasksActive: 0,
			TasksReady:  0,
			TasksDone:   0,
			TasksTotal:  0,
		}

		var buf strings.Builder
		if err := r.RenderFeature(&buf, f); err != nil {
			t.Fatalf("RenderFeature: %v", err)
		}
		out := buf.String()

		if !strings.Contains(out, "Design:") {
			t.Errorf("missing Design label:\n%s", out)
		}
		if !strings.Contains(out, "Spec:") {
			t.Errorf("missing Spec label:\n%s", out)
		}
		if !strings.Contains(out, "Dev plan:") {
			t.Errorf("missing Dev plan label:\n%s", out)
		}
		// All should show ✗ missing.
		missingCount := strings.Count(out, "✗")
		if missingCount < 3 {
			t.Errorf("expected at least 3 ✗ symbols for missing docs, got %d:\n%s", missingCount, out)
		}
	})

	t.Run("no dev-plan shows missing mark", func(t *testing.T) {
		r := NewRenderer(StaticTTY{Value: true})
		f := &FeatureInput{
			DisplayID:   "F-003",
			Slug:        "no-plan-doc",
			Status:      "designing",
			TasksActive: 0,
			TasksReady:  0,
			TasksDone:   0,
			TasksTotal:  0,
			Documents: []DocInput{
				{Type: "design", Path: "work/design/foo.md", Status: "approved"},
				{Type: "specification", Path: "work/spec/foo-spec.md", Status: "approved"},
			},
		}

		var buf strings.Builder
		if err := r.RenderFeature(&buf, f); err != nil {
			t.Fatalf("RenderFeature: %v", err)
		}
		out := buf.String()

		// The dev-plan row should show ✗ missing.
		lines := strings.Split(out, "\n")
		foundDevPlanMissing := false
		for _, l := range lines {
			if strings.Contains(l, "Dev plan:") && strings.Contains(l, "✗") && strings.Contains(l, "missing") {
				foundDevPlanMissing = true
				break
			}
		}
		if !foundDevPlanMissing {
			t.Errorf("dev-plan row should show missing mark:\n%s", out)
		}
	})

	t.Run("zero tasks", func(t *testing.T) {
		r := NewRenderer(StaticTTY{Value: true})
		f := &FeatureInput{
			DisplayID:   "F-004",
			Slug:        "no-tasks",
			Status:      "ready",
			TasksActive: 0,
			TasksReady:  0,
			TasksDone:   0,
			TasksTotal:  0,
		}

		var buf strings.Builder
		if err := r.RenderFeature(&buf, f); err != nil {
			t.Fatalf("RenderFeature: %v", err)
		}
		out := buf.String()

		if !strings.Contains(out, "0 active") {
			t.Errorf("expected 0 active:\n%s", out)
		}
		if !strings.Contains(out, "0 total") {
			t.Errorf("expected 0 total:\n%s", out)
		}
	})

	t.Run("no attention omits attention block", func(t *testing.T) {
		r := NewRenderer(StaticTTY{Value: true})
		f := &FeatureInput{
			DisplayID:   "F-005",
			Slug:        "quiet",
			Status:      "done",
			TasksDone:   1,
			TasksTotal:  1,
		}

		var buf strings.Builder
		if err := r.RenderFeature(&buf, f); err != nil {
			t.Fatalf("RenderFeature: %v", err)
		}
		out := buf.String()

		if strings.Contains(out, "⚠") {
			t.Errorf("should not contain warning symbol when no attention:\n%s", out)
		}
	})

	t.Run("with attention items", func(t *testing.T) {
		r := NewRenderer(StaticTTY{Value: true})
		f := &FeatureInput{
			DisplayID:   "F-006",
			Slug:        "needs-attention",
			Status:      "developing",
			TasksActive: 1,
			TasksTotal:  1,
			Attention: []AttentionItem{
				{Type: "missing_doc", Severity: "warning", Message: "No dev-plan document — agents cannot begin planning"},
			},
		}

		var buf strings.Builder
		if err := r.RenderFeature(&buf, f); err != nil {
			t.Fatalf("RenderFeature: %v", err)
		}
		out := buf.String()

		if !strings.Contains(out, "⚠") {
			t.Errorf("missing warning symbol in attention block:\n%s", out)
		}
		if !strings.Contains(out, "No dev-plan document") {
			t.Errorf("missing attention message:\n%s", out)
		}
	})

	t.Run("specification normalised to spec", func(t *testing.T) {
		r := NewRenderer(StaticTTY{Value: true})
		f := &FeatureInput{
			DisplayID:   "F-007",
			Slug:        "spec-test",
			Status:      "specifying",
			TasksTotal:  0,
			Documents: []DocInput{
				{Type: "specification", Path: "work/spec/my-spec.md", Status: "draft"},
			},
		}

		var buf strings.Builder
		if err := r.RenderFeature(&buf, f); err != nil {
			t.Fatalf("RenderFeature: %v", err)
		}
		out := buf.String()

		if !strings.Contains(out, "my-spec.md") {
			t.Errorf("specification type should be recognised as spec, output:\n%s", out)
		}
	})
}
