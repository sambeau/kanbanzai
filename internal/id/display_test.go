package id

import "testing"

func TestNormalizeID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		// Split form → canonical unsplit form
		{"FEAT-01KMR-X1SEQV49", "FEAT-01KMRX1SEQV49"},
		{"TASK-01J3K-ZZZBB4KF", "TASK-01J3KZZZBB4KF"},
		{"BUG-01J4A-R7WHN4F2", "BUG-01J4AR7WHN4F2"},
		{"DEC-01J3K-ABCDE7MX", "DEC-01J3KABCDE7MX"},
		{"DOC-01J3K-7MXP3RT5", "DOC-01J3K7MXP3RT5"},
		{"INC-01J3K-7MXP3RT5", "INC-01J3K7MXP3RT5"},

		// Already canonical — pass through unchanged
		{"FEAT-01KMRX1SEQV49", "FEAT-01KMRX1SEQV49"},
		{"TASK-01J3KZZZBB4KF", "TASK-01J3KZZZBB4KF"},

		// Plan IDs — pass through unchanged (hyphens are structural)
		{"P7-developer-experience", "P7-developer-experience"},
		{"P1-my-plan", "P1-my-plan"},

		// Empty and whitespace
		{"", ""},
		{"  FEAT-01KMR-X1SEQV49  ", "FEAT-01KMRX1SEQV49"},

		// Lowercase split form — normalized via uppercase
		{"feat-01kmr-x1seqv49", "FEAT-01KMRX1SEQV49"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got := NormalizeID(tt.input)
			if got != tt.want {
				t.Fatalf("NormalizeID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatEntityRef(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		displayID string
		slug      string
		label     string
		want      string
	}{
		{
			name:      "with slug no label",
			displayID: "FEAT-01KMR-X1SEQV49",
			slug:      "policy-docs",
			want:      "FEAT-01KMR-X1SEQV49 (policy-docs)",
		},
		{
			name:      "with slug and label",
			displayID: "FEAT-01KMR-X1SEQV49",
			slug:      "policy-docs",
			label:     "G",
			want:      "FEAT-01KMR-X1SEQV49 (G policy-docs)",
		},
		{
			name:      "no slug",
			displayID: "FEAT-01KMR-X1SEQV49",
			slug:      "",
			want:      "FEAT-01KMR-X1SEQV49",
		},
		{
			name:      "empty label",
			displayID: "TASK-01J3K-ZZZBB4KF",
			slug:      "my-task",
			label:     "",
			want:      "TASK-01J3K-ZZZBB4KF (my-task)",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := FormatEntityRef(tt.displayID, tt.slug, tt.label)
			if got != tt.want {
				t.Fatalf("FormatEntityRef(%q, %q, %q) = %q, want %q",
					tt.displayID, tt.slug, tt.label, got, tt.want)
			}
		})
	}
}

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
