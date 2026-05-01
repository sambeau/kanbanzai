// Package status provides CLI renderers for the kbz status command.
package status

import (
	"fmt"
	"io"
	"sort"

	"github.com/sambeau/kanbanzai/internal/cli/render"
	"github.com/sambeau/kanbanzai/internal/service"
)

// PlainRenderer writes plain key:value status output to an io.Writer.
// Each scope type has its own Render* method that consumes the
// same input types as the human renderer.
type PlainRenderer struct{}

// RenderFeature writes a plain key:value block for a feature.
func (r *PlainRenderer) RenderFeature(w io.Writer, in *render.FeatureInput) error {
	design := docByType(in.Documents, "design")
	spec := docByType(in.Documents, "specification")
	devPlan := docByType(in.Documents, "dev-plan")

	return writePairs(w, []pair{
		{"scope", "feature"},
		{"id", in.ID},
		{"slug", in.Slug},
		{"status", in.Status},
		{"plan", val(in.PlanID)},
		{"doc.design", docPath(design)},
		{"doc.design.status", docStatus(design)},
		{"doc.spec", docPath(spec)},
		{"doc.spec.status", docStatus(spec)},
		{"doc.dev-plan", docPath(devPlan)},
		{"doc.dev-plan.status", docStatus(devPlan)},
		{"tasks.active", fmt.Sprintf("%d", in.TasksActive)},
		{"tasks.ready", fmt.Sprintf("%d", in.TasksReady)},
		{"tasks.done", fmt.Sprintf("%d", in.TasksDone)},
		{"tasks.total", fmt.Sprintf("%d", in.TasksTotal)},
		{"attention", attentionFirst(in.Attention)},
	})
}

// RenderPlan writes a plain key:value block for a plan.
func (r *PlainRenderer) RenderPlan(w io.Writer, in *render.PlanInput) error {
	active, done, total := 0, 0, len(in.Features)
	for _, f := range in.Features {
		switch f.Status {
		case "done", "closed", "merged":
			done++
		case "active", "developing", "reviewing":
			active++
		}
	}

	return writePairs(w, []pair{
		{"scope", "plan"},
		{"id", in.ID},
		{"slug", in.Slug},
		{"status", in.Status},
		{"features.active", fmt.Sprintf("%d", active)},
		{"features.done", fmt.Sprintf("%d", done)},
		{"features.total", fmt.Sprintf("%d", total)},
		{"attention", attentionFirst(in.Attention)},
	})
}

// RenderTask writes a plain key:value block for a task.
func (r *PlainRenderer) RenderTask(w io.Writer, id, slug, status, parentFeature string, attention []render.AttentionItem) error {
	return writePairs(w, []pair{
		{"scope", "task"},
		{"id", id},
		{"slug", slug},
		{"status", status},
		{"parent_feature", val(parentFeature)},
		{"attention", attentionFirst(attention)},
	})
}

// RenderBug writes a plain key:value block for a bug.
func (r *PlainRenderer) RenderBug(w io.Writer, id, slug, status, severity, parentFeature string, attention []render.AttentionItem) error {
	pf := parentFeature
	if pf == "" {
		pf = "missing"
	}
	return writePairs(w, []pair{
		{"scope", "bug"},
		{"id", id},
		{"slug", slug},
		{"status", status},
		{"severity", val(severity)},
		{"parent_feature", pf},
		{"attention", attentionFirst(attention)},
	})
}

// RenderDocument writes a plain key:value block for a document.
func (r *PlainRenderer) RenderDocument(w io.Writer, d *service.DocumentResult) error {
	registered := "true"
	if d.ID == "" {
		registered = "false"
	}
	return writePairs(w, []pair{
		{"scope", "document"},
		{"id", val(d.ID)},
		{"path", d.Path},
		{"type", d.Type},
		{"status", d.Status},
		{"registered", registered},
		{"owner", val(d.Owner)},
		{"attention", "none"},
	})
}

// RenderProject writes a plain key:value block for the project overview.
func (r *PlainRenderer) RenderProject(w io.Writer, in *render.ProjectInput) error {
	healthErrors := 0
	healthWarnings := 0
	if in.Health != nil {
		healthErrors = in.Health.Errors
		healthWarnings = in.Health.Warnings
	}

	featuresTotal := 0
	featuresDone := 0
	featuresActive := 0
	for _, p := range in.Plans {
		featuresTotal += p.FeaturesTotal
		featuresDone += p.FeaturesTotal - p.FeaturesActive // approximate
		featuresActive += p.FeaturesActive
	}

	return writePairs(w, []pair{
		{"scope", "project"},
		{"plans.total", fmt.Sprintf("%d", len(in.Plans))},
		{"features.active", fmt.Sprintf("%d", featuresActive)},
		{"features.done", fmt.Sprintf("%d", featuresDone)},
		{"features.total", fmt.Sprintf("%d", featuresTotal)},
		{"health.errors", fmt.Sprintf("%d", healthErrors)},
		{"health.warnings", fmt.Sprintf("%d", healthWarnings)},
		{"attention", attentionFirst(in.Attention)},
	})
}

// ─── helpers ──────────────────────────────────────────────────────────────────

type pair struct{ key, val string }

func writePairs(w io.Writer, pairs []pair) error {
	for _, p := range pairs {
		if _, err := fmt.Fprintf(w, "%s: %s\n", p.key, p.val); err != nil {
			return err
		}
	}
	return nil
}

func attentionFirst(items []render.AttentionItem) string {
	if len(items) == 0 {
		return "none"
	}
	sort.Slice(items, func(i, j int) bool {
		return severityRank(items[i].Severity) > severityRank(items[j].Severity)
	})
	return items[0].Message
}

func severityRank(s string) int {
	switch s {
	case "error":
		return 3
	case "warning":
		return 2
	case "info":
		return 1
	default:
		return 0
	}
}

func docByType(docs []render.DocInput, docType string) *render.DocInput {
	for i := range docs {
		if docs[i].Type == docType {
			return &docs[i]
		}
	}
	return nil
}

func val(s string) string {
	if s == "" {
		return "missing"
	}
	return s
}

func docPath(d *render.DocInput) string {
	if d == nil {
		return "missing"
	}
	return val(d.Path)
}

func docStatus(d *render.DocInput) string {
	if d == nil {
		return "missing"
	}
	return val(d.Status)
}
