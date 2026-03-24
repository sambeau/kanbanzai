package knowledge

import (
	"testing"
)

func TestScanEntityRefs(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		wantSpans []string
	}{
		{
			name:      "empty string",
			text:      "",
			wantSpans: nil,
		},
		{
			name:      "no entity references",
			text:      "Just some plain text with no references at all",
			wantSpans: nil,
		},
		{
			name:      "single FEAT reference",
			text:      "See FEAT-ABC123 for details",
			wantSpans: []string{"FEAT-ABC123"},
		},
		{
			name:      "single TASK reference",
			text:      "Implements TASK-XYZ789",
			wantSpans: []string{"TASK-XYZ789"},
		},
		{
			name:      "single BUG reference",
			text:      "Fixed in BUG-DEF456",
			wantSpans: []string{"BUG-DEF456"},
		},
		{
			name:      "single DEC reference",
			text:      "Per DEC-GHI012",
			wantSpans: []string{"DEC-GHI012"},
		},
		{
			name:      "single KE reference",
			text:      "Knowledge entry KE-01J3K7MXP3RT5",
			wantSpans: []string{"KE-01J3K7MXP3RT5"},
		},
		{
			name:      "plan ID reference",
			text:      "Part of P2-basic-ui",
			wantSpans: []string{"P2-basic-ui"},
		},
		{
			name:      "multiple references",
			text:      "FEAT-A1 depends on TASK-B2 per DEC-C3",
			wantSpans: []string{"FEAT-A1", "TASK-B2", "DEC-C3"},
		},
		{
			name:      "duplicate references deduped",
			text:      "FEAT-ABC in one place, FEAT-ABC in another",
			wantSpans: []string{"FEAT-ABC"},
		},
		{
			name:      "adjacent references",
			text:      "FEAT-ABC TASK-DEF",
			wantSpans: []string{"FEAT-ABC", "TASK-DEF"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ScanEntityRefs(tt.text)

			if len(got) != len(tt.wantSpans) {
				t.Fatalf("got %d refs, want %d; refs = %v", len(got), len(tt.wantSpans), got)
			}

			for i, ref := range got {
				if ref.Span != tt.wantSpans[i] {
					t.Errorf("ref[%d].Span = %q, want %q", i, ref.Span, tt.wantSpans[i])
				}
				// Verify Span matches the source text at the reported offsets.
				if ref.End > len(tt.text) || ref.Start > ref.End {
					t.Errorf("ref[%d] has invalid offsets Start=%d End=%d (text len %d)",
						i, ref.Start, ref.End, len(tt.text))
				} else if tt.text[ref.Start:ref.End] != ref.Span {
					t.Errorf("ref[%d] offsets [%d:%d] = %q, want %q",
						i, ref.Start, ref.End, tt.text[ref.Start:ref.End], ref.Span)
				}
			}
		})
	}
}

func TestEntityTypeFromID(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"FEAT-ABC", "feature"},
		{"TASK-XYZ", "task"},
		{"BUG-DEF", "bug"},
		{"DEC-GHI", "decision"},
		{"KE-01J3K7MXP3RT5", "knowledge_entry"},
		{"P2-basic-ui", "plan"},
		{"unknown", "plan"},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			got := EntityTypeFromID(tt.id)
			if got != tt.want {
				t.Errorf("EntityTypeFromID(%q) = %q, want %q", tt.id, got, tt.want)
			}
		})
	}
}
