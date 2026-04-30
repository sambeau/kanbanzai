package render

import (
	"fmt"
	"strings"
)

// AlignDocuments takes document rows (each of the form
// ["Design:", "✓", "work/design/foo.md", "approved"]) and returns
// a []string where the label, ✓ mark, path, and status columns are aligned
// across all rows.
//
// Row format: [label, present_mark, path, status]
// The present_mark column is constant width (1 for TTY, or the
// ASCII symbol width for non-TTY).
func AlignDocuments(rows [][]string) []string {
	if len(rows) == 0 {
		return nil
	}

	// Find max width of label (index 0) and path (index 2) columns.
	maxLabel := 0
	maxPath := 0
	for _, r := range rows {
		if len(r) >= 1 && len(r[0]) > maxLabel {
			maxLabel = len(r[0])
		}
		if len(r) >= 3 && len(r[2]) > maxPath {
			maxPath = len(r[2])
		}
	}

	out := make([]string, len(rows))
	for i, r := range rows {
		if len(r) < 4 {
			out[i] = strings.Join(r, " ")
			continue
		}
		label := r[0]
		mark := r[1]
		path := r[2]
		status := r[3]
		paddedLabel := fmt.Sprintf("%-*s", maxLabel, label)
		paddedPath := fmt.Sprintf("%-*s", maxPath, path)
		out[i] = fmt.Sprintf("    %s  %s  %s  %s", paddedLabel, mark, paddedPath, status)
	}
	return out
}
