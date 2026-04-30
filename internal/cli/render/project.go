package render

import (
	"fmt"
	"io"
)

// RenderProject writes a TTY-aware human-readable project overview view.
func (r *Renderer) RenderProject(w io.Writer, p *ProjectInput) error {
	tty := r.IsTTY()

	// Header.
	header := "Kanbanzai"
	if p.Name != "" {
		header += fmt.Sprintf("  %s", p.Name)
	}
	fmt.Fprintln(w, header)

	// Plans block.
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  Plans (%d)\n", len(p.Plans))
	for _, pl := range p.Plans {
		statusIcon := r.featureStatusIcon(pl.Status, tty)
		featDesc := fmt.Sprintf("%d features active", pl.FeaturesActive)
		if pl.FeaturesActive == 0 {
			featDesc = "0 features started"
		}
		fmt.Fprintf(w, "    %s  %s   %s\n", pl.DisplayID, statusIcon, featDesc)
	}

	// Health.
	fmt.Fprintln(w)
	r.writeHealth(w, p.Health, tty)

	// Attention.
	if len(p.Attention) > 0 {
		fmt.Fprintln(w)
		r.writeAttention(w, p.Attention, tty)
	}

	// Work queue.
	fmt.Fprintln(w)
	r.writeWorkQueue(w, p.WorkQueue, tty)

	return nil
}

func (r *Renderer) writeHealth(w io.Writer, h *StatusHealthSummary, tty bool) {
	if h == nil {
		h = &StatusHealthSummary{}
	}

	ok := Symbol("ok", tty)
	missing := Symbol("missing", tty)
	sep := Symbol("separator", tty)

	if h.Errors == 0 && h.Warnings == 0 {
		fmt.Fprintf(w, "  Health  %s no errors %s no warnings\n", ok, sep)
		return
	}

	var parts []string
	if h.Errors > 0 {
		parts = append(parts, Red(fmt.Sprintf("%s %d errors", missing, h.Errors), tty))
	} else {
		parts = append(parts, fmt.Sprintf("%s no errors", ok))
	}
	if h.Warnings > 0 {
		parts = append(parts, Yellow(fmt.Sprintf("%d warnings", h.Warnings), tty))
	} else {
		parts = append(parts, "no warnings")
	}

	fmt.Fprintf(w, "  Health  %s %s %s\n", parts[0], sep, parts[1])
}

func (r *Renderer) writeWorkQueue(w io.Writer, wq ProjectWorkQueue, tty bool) {
	sep := Symbol("separator", tty)
	fmt.Fprintf(w, "  Work queue  %d ready %s %d active\n", wq.Ready, sep, wq.Active)
}
