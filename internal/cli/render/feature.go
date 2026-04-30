package render

import (
	"fmt"
	"io"
)

// RenderFeature writes a TTY-aware human-readable feature detail view.
func (r *Renderer) RenderFeature(w io.Writer, f *FeatureInput) error {
	tty := r.IsTTY()

	// Header line.
	header := fmt.Sprintf("Feature  %s · %s", f.DisplayID, f.Slug)
	if f.Status != "" {
		header += fmt.Sprintf("  %s", r.statusBadge(f.Status, tty))
	}
	fmt.Fprintln(w, header)

	// Plan reference line.
	if f.PlanID != "" {
		planLine := fmt.Sprintf("  Plan:  %s", f.PlanID)
		if f.PlanName != "" {
			planLine += fmt.Sprintf(" · %s", f.PlanName)
		}
		fmt.Fprintln(w, planLine)
	}

	// Documents block.
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Documents")
	r.writeDocRows(w, f.Documents, tty)

	// Tasks summary.
	fmt.Fprintln(w)
	active := Symbol("active", tty)
	sep := Symbol("separator", tty)
	fmt.Fprintf(w, "  Tasks  %s %d active %s %d ready %s %d done  (%d total)\n",
		active, f.TasksActive, sep, f.TasksReady, sep, f.TasksDone, f.TasksTotal)

	// Attention block.
	if len(f.Attention) > 0 {
		fmt.Fprintln(w)
		r.writeAttention(w, f.Attention, tty)
	}

	return nil
}

func (r *Renderer) writeDocRows(w io.Writer, docs []DocInput, tty bool) {
	// Map the three standard doc types.
	docMap := map[string]DocInput{
		"design":   {Type: "design", Path: "missing", Status: "missing"},
		"spec":     {Type: "spec", Path: "missing", Status: "missing"},
		"dev-plan": {Type: "dev-plan", Path: "missing", Status: "missing"},
	}
	for _, d := range docs {
		// Normalise "specification" → "spec" for matching.
		t := d.Type
		if t == "specification" {
			t = "spec"
		}
		if _, ok := docMap[t]; ok {
			docMap[t] = d
		}
	}

	var rows [][]string
	order := []struct {
		key, label string
	}{
		{"design", "Design:"},
		{"spec", "Spec:"},
		{"dev-plan", "Dev plan:"},
	}

	okMark := Symbol("ok", tty)
	missingMark := Symbol("missing", tty)

	hasAll := true
	for _, o := range order {
		d := docMap[o.key]
		mark := okMark
		status := d.Status
		if d.Path == "missing" {
			mark = missingMark
			status = "missing"
			hasAll = false
		}
		rows = append(rows, []string{o.label, mark, d.Path, status})
	}

	for _, line := range AlignDocuments(rows) {
		fmt.Fprintln(w, line)
	}

	if !hasAll {
		fmt.Fprintln(w)
	}
}

func (r *Renderer) writeAttention(w io.Writer, items []AttentionItem, tty bool) {
	warn := Symbol("warn", tty)
	for _, a := range items {
		fmt.Fprintf(w, "  %s  %s\n", warn, a.Message)
	}
}

// statusBadge returns a coloured status badge string.
func (r *Renderer) statusBadge(status string, tty bool) string {
	switch status {
	case "done", "closed", "merged":
		return Green(status, tty)
	case "developing", "active", "reviewing":
		return Yellow(status, tty)
	case "blocked", "stalled":
		return Red(status, tty)
	default:
		return status
	}
}
