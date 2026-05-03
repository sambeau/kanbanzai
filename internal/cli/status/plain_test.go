package status

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sambeau/kanbanzai/internal/cli/render"
	"github.com/sambeau/kanbanzai/internal/service"
)

// ─── FR-3 Feature plain ───────────────────────────────────────────────────────

func TestPlainRenderer_RenderFeature_Full(t *testing.T) {
	r := &PlainRenderer{}
	in := &render.FeatureInput{
		ID:          "FEAT-042",
		Slug:        "my-feature",
		Status:      "developing",
		PlanID:      "P1-my-plan",
		TasksActive: 1,
		TasksReady:  3,
		TasksDone:   7,
		TasksTotal:  11,
		Documents: []render.DocInput{
			{ID: "DOC-0019", Type: "design", Path: "work/design/my-feature.md", Status: "approved"},
			{ID: "DOC-0023", Type: "specification", Path: "work/spec/my-feature-spec.md", Status: "approved"},
		},
		Attention: []render.AttentionItem{
			{Severity: "warning", Message: "No dev-plan document registered"},
		},
	}

	var buf bytes.Buffer
	if err := r.RenderFeature(&buf, in); err != nil {
		t.Fatalf("RenderFeature error: %v", err)
	}

	out := buf.String()

	// FR-3: first key MUST be scope
	if !strings.HasPrefix(out, "scope: feature") {
		t.Errorf("output does not start with 'scope: feature':\n%s", out)
	}

	assertPlainKey(t, out, "id", "FEAT-042")
	assertPlainKey(t, out, "slug", "my-feature")
	assertPlainKey(t, out, "status", "developing")
	assertPlainKey(t, out, "plan", "P1-my-plan")
	assertPlainKey(t, out, "doc.design", "work/design/my-feature.md")
	assertPlainKey(t, out, "doc.design.status", "approved")
	assertPlainKey(t, out, "doc.spec", "work/spec/my-feature-spec.md")
	assertPlainKey(t, out, "doc.spec.status", "approved")
	assertPlainKey(t, out, "doc.dev-plan", "missing")
	assertPlainKey(t, out, "doc.dev-plan.status", "missing")
	assertPlainKey(t, out, "tasks.active", "1")
	assertPlainKey(t, out, "tasks.ready", "3")
	assertPlainKey(t, out, "tasks.done", "7")
	assertPlainKey(t, out, "tasks.total", "11")
	assertPlainContains(t, out, "attention", "No dev-plan document")
}

// ─── FR-3.1 Feature with no plan ──────────────────────────────────────────────

func TestPlainRenderer_RenderFeature_NoPlan(t *testing.T) {
	r := &PlainRenderer{}
	in := &render.FeatureInput{
		ID:     "FEAT-099",
		Slug:   "no-plan",
		Status: "designing",
		PlanID: "",
	}

	var buf bytes.Buffer
	if err := r.RenderFeature(&buf, in); err != nil {
		t.Fatalf("RenderFeature error: %v", err)
	}

	assertPlainKey(t, buf.String(), "plan", "missing")
}

// ─── FR-3.4 Empty attention → "none" ──────────────────────────────────────────

func TestPlainRenderer_RenderFeature_EmptyAttention(t *testing.T) {
	r := &PlainRenderer{}
	in := &render.FeatureInput{
		ID:     "FEAT-001",
		Slug:   "clean",
		Status: "done",
	}

	var buf bytes.Buffer
	if err := r.RenderFeature(&buf, in); err != nil {
		t.Fatalf("RenderFeature error: %v", err)
	}

	assertPlainKey(t, buf.String(), "attention", "none")
}

// ─── FR-4 Plan plain ──────────────────────────────────────────────────────────

func TestPlainRenderer_RenderPlan(t *testing.T) {
	r := &PlainRenderer{}
	in := &render.PlanInput{
		ID:     "P1-my-plan",
		Slug:   "my-plan",
		Status: "active",
		Features: []render.PlanFeatureInput{
			{Status: "developing"},
			{Status: "reviewing"},
			{Status: "done"},
			{Status: "done"},
			{Status: "done"},
		},
	}

	var buf bytes.Buffer
	if err := r.RenderPlan(&buf, in); err != nil {
		t.Fatalf("RenderPlan error: %v", err)
	}

	out := buf.String()

	if !strings.HasPrefix(out, "scope: plan") {
		t.Errorf("output does not start with 'scope: plan':\n%s", out)
	}
	assertPlainKey(t, out, "features.active", "2")
	assertPlainKey(t, out, "features.done", "3")
	assertPlainKey(t, out, "features.total", "5")
}

// ─── FR-5 Task plain ──────────────────────────────────────────────────────────

func TestPlainRenderer_RenderTask(t *testing.T) {
	r := &PlainRenderer{}
	var buf bytes.Buffer
	if err := r.RenderTask(&buf, "TASK-0099", "implement-output-flag", "active", "FEAT-042", nil); err != nil {
		t.Fatalf("RenderTask error: %v", err)
	}

	out := buf.String()
	if !strings.HasPrefix(out, "scope: task") {
		t.Errorf("output does not start with 'scope: task':\n%s", out)
	}
	assertPlainKey(t, out, "id", "TASK-0099")
	assertPlainKey(t, out, "slug", "implement-output-flag")
	assertPlainKey(t, out, "status", "active")
	assertPlainKey(t, out, "parent_feature", "FEAT-042")
	assertPlainKey(t, out, "attention", "none")
}

// ─── FR-5 Bug plain ───────────────────────────────────────────────────────────

func TestPlainRenderer_RenderBug(t *testing.T) {
	r := &PlainRenderer{}
	var buf bytes.Buffer
	if err := r.RenderBug(&buf, "BUG-0017", "crash-on-empty-project", "active", "high", "", nil); err != nil {
		t.Fatalf("RenderBug error: %v", err)
	}

	out := buf.String()
	if !strings.HasPrefix(out, "scope: bug") {
		t.Errorf("output does not start with 'scope: bug':\n%s", out)
	}
	assertPlainKey(t, out, "parent_feature", "missing")
}

func TestPlainRenderer_RenderBug_WithParentFeature(t *testing.T) {
	r := &PlainRenderer{}
	var buf bytes.Buffer
	if err := r.RenderBug(&buf, "BUG-0017", "crash", "active", "high", "FEAT-042", nil); err != nil {
		t.Fatalf("RenderBug error: %v", err)
	}

	assertPlainKey(t, buf.String(), "parent_feature", "FEAT-042")
}

// ─── FR-6 Document plain ──────────────────────────────────────────────────────

func TestPlainRenderer_RenderDocument_Registered(t *testing.T) {
	r := &PlainRenderer{}
	d := &service.DocumentResult{
		ID:     "DOC-0023",
		Path:   "work/spec/my-feature-spec.md",
		Type:   "specification",
		Status: "approved",
		Owner:  "FEAT-042",
	}

	var buf bytes.Buffer
	if err := r.RenderDocument(&buf, d); err != nil {
		t.Fatalf("RenderDocument error: %v", err)
	}

	out := buf.String()
	if !strings.HasPrefix(out, "scope: document") {
		t.Errorf("output does not start with 'scope: document':\n%s", out)
	}
	assertPlainKey(t, out, "id", "DOC-0023")
	assertPlainKey(t, out, "path", "work/spec/my-feature-spec.md")
	assertPlainKey(t, out, "type", "specification")
	assertPlainKey(t, out, "status", "approved")
	assertPlainKey(t, out, "registered", "true")
	assertPlainKey(t, out, "owner", "FEAT-042")
}

// ─── FR-6.2 Unregistered document → registered: false ─────────────────────────

func TestPlainRenderer_RenderDocument_Unregistered(t *testing.T) {
	r := &PlainRenderer{}
	d := &service.DocumentResult{
		Path: "work/spec/unregistered-spec.md",
	}

	var buf bytes.Buffer
	if err := r.RenderDocument(&buf, d); err != nil {
		t.Fatalf("RenderDocument error: %v", err)
	}

	out := buf.String()
	assertPlainKey(t, out, "registered", "false")
	assertPlainKey(t, out, "id", "missing")
}

// ─── FR-7 Project overview plain ──────────────────────────────────────────────

func TestPlainRenderer_RenderProject(t *testing.T) {
	r := &PlainRenderer{}
	in := &render.ProjectInput{
		Plans: []render.ProjectPlanInput{
			{DisplayID: "P1", Status: "active", FeaturesActive: 2, FeaturesTotal: 5},
			{DisplayID: "P2", Status: "done", FeaturesActive: 0, FeaturesTotal: 3},
		},
		Health: &render.StatusHealthSummary{Errors: 1, Warnings: 2},
	}

	var buf bytes.Buffer
	if err := r.RenderProject(&buf, in); err != nil {
		t.Fatalf("RenderProject error: %v", err)
	}

	out := buf.String()
	if !strings.HasPrefix(out, "scope: project") {
		t.Errorf("output does not start with 'scope: project':\n%s", out)
	}
	assertPlainKey(t, out, "plans.total", "2")
	// features.done is 5 - 2 + 3 - 0 = 6
	assertPlainKey(t, out, "features.done", "6")
	assertPlainKey(t, out, "features.total", "8")
	assertPlainKey(t, out, "health.errors", "1")
	assertPlainKey(t, out, "health.warnings", "2")
}

// ─── FR-7.1 health.errors for CI gate ─────────────────────────────────────────

func TestPlainRenderer_RenderProject_HealthGate(t *testing.T) {
	r := &PlainRenderer{}

	// Healthy project
	in := &render.ProjectInput{
		Plans:  []render.ProjectPlanInput{},
		Health: &render.StatusHealthSummary{Errors: 0, Warnings: 0},
	}

	var buf bytes.Buffer
	if err := r.RenderProject(&buf, in); err != nil {
		t.Fatalf("RenderProject error: %v", err)
	}

	assertPlainKey(t, buf.String(), "health.errors", "0")

	// Unhealthy project
	in2 := &render.ProjectInput{
		Plans:  []render.ProjectPlanInput{},
		Health: &render.StatusHealthSummary{Errors: 3, Warnings: 1},
	}

	buf.Reset()
	if err := r.RenderProject(&buf, in2); err != nil {
		t.Fatalf("RenderProject error: %v", err)
	}

	assertPlainKey(t, buf.String(), "health.errors", "3")
}

// ─── NFR-1.5 / AC-11: Plain key contract test ─────────────────────────────────

func TestPlainSchemaContract_RequiredKeys(t *testing.T) {
	r := &PlainRenderer{}

	// Feature keys (FR-3)
	t.Run("feature", func(t *testing.T) {
		var buf bytes.Buffer
		r.RenderFeature(&buf, &render.FeatureInput{
			ID: "x", Slug: "x", Status: "x",
		})
		out := buf.String()
		for _, key := range []string{
			"scope:", "id:", "slug:", "status:", "plan:",
			"doc.design:", "doc.design.status:", "doc.spec:", "doc.spec.status:",
			"doc.dev-plan:", "doc.dev-plan.status:",
			"tasks.active:", "tasks.ready:", "tasks.done:", "tasks.total:",
			"attention:",
		} {
			if !strings.Contains(out, key+" ") && !strings.Contains(out, key+"\n") {
				t.Errorf("missing required plain key: %s", key)
			}
		}
	})

	// Plan keys (FR-4)
	t.Run("plan", func(t *testing.T) {
		var buf bytes.Buffer
		r.RenderPlan(&buf, &render.PlanInput{
			ID: "x", Slug: "x", Status: "x",
		})
		out := buf.String()
		for _, key := range []string{
			"scope:", "id:", "slug:", "status:",
			"features.active:", "features.done:", "features.total:",
			"attention:",
		} {
			if !strings.Contains(out, key+" ") && !strings.Contains(out, key+"\n") {
				t.Errorf("missing required plain key: %s", key)
			}
		}
	})

	// Task keys (FR-5)
	t.Run("task", func(t *testing.T) {
		var buf bytes.Buffer
		r.RenderTask(&buf, "x", "x", "x", "x", nil)
		out := buf.String()
		for _, key := range []string{
			"scope:", "id:", "slug:", "status:", "parent_feature:", "attention:",
		} {
			if !strings.Contains(out, key+" ") && !strings.Contains(out, key+"\n") {
				t.Errorf("missing required plain key: %s", key)
			}
		}
	})

	// Bug keys (FR-5)
	t.Run("bug", func(t *testing.T) {
		var buf bytes.Buffer
		r.RenderBug(&buf, "x", "x", "x", "x", "x", nil)
		out := buf.String()
		for _, key := range []string{
			"scope:", "id:", "slug:", "status:", "severity:", "parent_feature:", "attention:",
		} {
			if !strings.Contains(out, key+" ") && !strings.Contains(out, key+"\n") {
				t.Errorf("missing required plain key: %s", key)
			}
		}
	})

	// Document keys (FR-6)
	t.Run("document", func(t *testing.T) {
		var buf bytes.Buffer
		r.RenderDocument(&buf, &service.DocumentResult{
			ID: "x", Path: "x", Type: "x", Status: "x", Owner: "x",
		})
		out := buf.String()
		for _, key := range []string{
			"scope:", "id:", "path:", "type:", "status:", "registered:", "owner:", "attention:",
		} {
			if !strings.Contains(out, key+" ") && !strings.Contains(out, key+"\n") {
				t.Errorf("missing required plain key: %s", key)
			}
		}
	})

	// Project keys (FR-7)
	t.Run("project", func(t *testing.T) {
		var buf bytes.Buffer
		r.RenderProject(&buf, &render.ProjectInput{
			Plans:  []render.ProjectPlanInput{},
			Health: &render.StatusHealthSummary{},
		})
		out := buf.String()
		for _, key := range []string{
			"scope:", "plans.total:", "features.active:", "features.done:", "features.total:",
			"health.errors:", "health.warnings:", "attention:",
		} {
			if !strings.Contains(out, key+" ") && !strings.Contains(out, key+"\n") {
				t.Errorf("missing required plain key: %s", key)
			}
		}
	})
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func assertPlainKey(t *testing.T, out, key, want string) {
	t.Helper()
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, key+": ") || strings.HasPrefix(trimmed, key+":\t") {
			val := strings.TrimPrefix(trimmed, key+": ")
			val = strings.TrimPrefix(val, key+":\t")
			if val != want {
				t.Errorf("key %q = %q, want %q", key, val, want)
			}
			return
		}
	}
	t.Errorf("key %q not found in output:\n%s", key, out)
}

func assertPlainContains(t *testing.T, out, key, want string) {
	t.Helper()
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, key+": ") {
			if !strings.Contains(trimmed, want) {
				t.Errorf("key %q contains %q, want substring %q", key, trimmed, want)
			}
			return
		}
	}
	t.Errorf("key %q not found in output:\n%s", key, out)
}
