package render

import (
	"strings"
	"testing"
)

func TestAlignDocuments(t *testing.T) {
	t.Parallel()

	t.Run("empty rows returns nil", func(t *testing.T) {
		got := AlignDocuments(nil)
		if got != nil {
			t.Errorf("AlignDocuments(nil) = %v, want nil", got)
		}
		got = AlignDocuments([][]string{})
		if got != nil {
			t.Errorf("AlignDocuments([][]) = %v, want nil", got)
		}
	})

	t.Run("aligns path column to max width", func(t *testing.T) {
		rows := [][]string{
			{"Design:", "✓", "work/design/foo.md", "approved"},
			{"Spec:", "✓", "work/spec/short.md", "approved"},
			{"Dev plan:", "✗", "missing", "missing"},
		}
		got := AlignDocuments(rows)
		if len(got) != 3 {
			t.Fatalf("expected 3 rows, got %d", len(got))
		}

		// Now both label and path columns are padded.
		// Max label: "Dev plan:" (9 chars), max path: "work/design/foo.md" (20 chars).
		// The status text should start at the same column in all rows.
		// Check that "approved" appears at same column position for rows 0 and 1.
		var approvedPositions []int
		for _, line := range got {
			if pos := strings.Index(line, "  approved"); pos >= 0 {
				approvedPositions = append(approvedPositions, pos)
			}
		}
		if len(approvedPositions) == 2 {
			if approvedPositions[0] != approvedPositions[1] {
				t.Errorf("approved status not aligned: positions %d vs %d:\n%s",
					approvedPositions[0], approvedPositions[1], strings.Join(got, "\n"))
			}
		}
		// Verify the third row has "missing" as status at the same column.
		if len(approvedPositions) == 2 {
			if !strings.Contains(got[2], "missing             missing") {
				t.Errorf("row 2 (missing path + missing status) not padded correctly:\n%s", got[2])
			}
		}
	})

	t.Run("short rows passed through", func(t *testing.T) {
		rows := [][]string{
			{"just", "two"},
		}
		got := AlignDocuments(rows)
		if len(got) != 1 {
			t.Fatalf("expected 1 row, got %d", len(got))
		}
		if got[0] != "just two" {
			t.Errorf("expected 'just two', got %q", got[0])
		}
	})
}
