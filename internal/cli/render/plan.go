package render

import (
	"fmt"
	"io"
)

// RenderPlan writes a TTY-aware human-readable plan dashboard view.
func (r *Renderer) RenderPlan(w io.Writer, p *PlanInput) error {
	tty := r.IsTTY()

	// Header.
	header := fmt.Sprintf("Plan  %s · %s", p.DisplayID, p.Slug)
	if p.Status != "" {
		header += fmt.Sprintf("  %s", r.statusBadge(p.Status, tty))
	}
	fmt.Fprintln(w, header)

	// Features block.
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  Features (%d)\n", len(p.Features))
	for _, f := range p.Features {
		statusIcon := r.featureStatusIcon(f.Status, tty)
		fmt.Fprintf(w, "    %s  %s  %s\n", f.DisplayID, f.Slug, statusIcon)
	}

	// Tasks summary.
	fmt.Fprintln(w)
	active := Symbol("active", tty)
	sep := Symbol("separator", tty)
	fmt.Fprintf(w, "  Tasks  %s %d active %s %d ready %s %d done  (%d total)\n",
		active, p.TasksActive, sep, p.TasksReady, sep, p.TasksDone, p.TasksTotal)

	// Attention block.
	if len(p.Attention) > 0 {
		fmt.Fprintln(w)
		r.writeAttention(w, p.Attention, tty)
	}

	return nil
}

func (r *Renderer) featureStatusIcon(status string, tty bool) string {
	ok := Symbol("ok", tty)
	switch status {
	case "done", "closed", "merged", "superseded", "cancelled":
		return fmt.Sprintf("%s %s", ok, Green(status, tty))
	case "developing", "reviewing", "active":
		return fmt.Sprintf("%s %s", Symbol("active", tty), Yellow(status, tty))
	case "ready":
		return fmt.Sprintf("%s %s", Symbol("ready", tty), status)
	case "blocked", "stalled":
		return fmt.Sprintf("%s %s", Symbol("missing", tty), Red(status, tty))
	default:
		return fmt.Sprintf("%s %s", Symbol("ready", tty), status)
	}
}
