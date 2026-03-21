package id

import "testing"

func TestFormatFullDisplay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"FEAT-01J3K7MXP3RT5", "FEAT-01J3K-7MXP3RT5"},
		{"BUG-01J4AR7WHN4F2", "BUG-01J4A-R7WHN4F2"},
		{"TASK-01J3KZZZBB4KF", "TASK-01J3K-ZZZBB4KF"},
		{"DEC-01J3KABCDE7MX", "DEC-01J3K-ABCDE7MX"},
		{"DOC-01J3K7MXP3RT5", "DOC-01J3K-7MXP3RT5"},
		{"EPIC-MYPROJECT", "EPIC-MYPROJECT"}, // epic unchanged
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := FormatFullDisplay(tt.input)
			if got != tt.want {
				t.Fatalf("FormatFullDisplay(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestStripBreakHyphens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"FEAT-01J3K-7MXP3RT5", "FEAT-01J3K7MXP3RT5"},
		{"feat-01j3k-7mxp3rt5", "FEAT-01J3K7MXP3RT5"}, // case normalized
		{"FEAT-01J3K7MXP3RT5", "FEAT-01J3K7MXP3RT5"},  // already canonical
		{"EPIC-MYPROJECT", "EPIC-MYPROJECT"},          // epic unchanged
		{"BUG-01J4A-R7WHN4F2", "BUG-01J4AR7WHN4F2"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := StripBreakHyphens(tt.input)
			if got != tt.want {
				t.Fatalf("StripBreakHyphens(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatShortDisplay(t *testing.T) {
	t.Parallel()

	ids := []string{
		"FEAT-01J3K7MXP3RT5",
		"FEAT-01J3KABCDE7MX",
		"FEAT-01J4ZR7WHN4F2",
	}

	// First two share prefix "01J3K" so need more than 5 chars
	got := FormatShortDisplay("FEAT-01J3K7MXP3RT5", ids)
	if got == "FEAT-01J3K" {
		// This would be ambiguous with the second ID
		t.Fatalf("FormatShortDisplay() = %q, should be longer to disambiguate", got)
	}

	// Third ID has unique prefix at 5 chars ("01J4Z" vs "01J3K")
	got = FormatShortDisplay("FEAT-01J4ZR7WHN4F2", ids)
	if got != "FEAT-01J4Z" {
		t.Fatalf("FormatShortDisplay() = %q, want %q", got, "FEAT-01J4Z")
	}
}

func TestShortestUniquePrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		target string
		others []string
		want   int
	}{
		{"no others", "01J3K7MXP3RT5", nil, 5},
		{"all unique at 5", "01J3K7MXP3RT5", []string{"01J4ZR7WHN4F2"}, 5},
		{"needs 6", "01J3K7MXP3RT5", []string{"01J3KABCDE7MX"}, 6},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ShortestUniquePrefix(tt.target, tt.others)
			if got != tt.want {
				t.Fatalf("ShortestUniquePrefix() = %d, want %d", got, tt.want)
			}
		})
	}
}
