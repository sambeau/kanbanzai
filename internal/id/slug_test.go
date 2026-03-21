package id

import "testing"

func TestValidateEpicSlug_Valid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"IDS", "IDS"},
		{"ids", "IDS"}, // lowercase normalized
		{"PHASE1", "PHASE1"},
		{"MY-EPIC", "MY-EPIC"},
		{"my-epic", "MY-EPIC"}, // lowercase with hyphen
		{"AB", "AB"},           // minimum length
		{"ABCDEFGHIJKLMNOPQRST", "ABCDEFGHIJKLMNOPQRST"}, // 20 chars, max length
		{"A1", "A1"},   // mix of letters and digits
		{"123", "123"}, // all digits
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got, err := ValidateEpicSlug(tt.input)
			if err != nil {
				t.Fatalf("ValidateEpicSlug(%q) error = %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("ValidateEpicSlug(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateEpicSlug_Invalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{"too short", "A"},
		{"too long", "ABCDEFGHIJKLMNOPQRSTU"}, // 21 chars
		{"contains space", "MY EPIC"},
		{"contains underscore", "MY_EPIC"},
		{"leading hyphen", "-EPIC"},
		{"trailing hyphen", "EPIC-"},
		{"consecutive hyphens", "EPIC--NAME"},
		{"empty", ""},
		{"only hyphen", "-"},
		{"contains lowercase special", "epic!"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := ValidateEpicSlug(tt.input)
			if err == nil {
				t.Fatalf("ValidateEpicSlug(%q) error = nil, want non-nil", tt.input)
			}
		})
	}
}
