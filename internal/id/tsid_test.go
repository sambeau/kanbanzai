package id

import (
	"testing"
	"time"
)

func TestGenerateTSID13_Length(t *testing.T) {
	t.Parallel()
	tsid, err := GenerateTSID13()
	if err != nil {
		t.Fatalf("GenerateTSID13() error = %v", err)
	}
	if len(tsid) != 13 {
		t.Fatalf("GenerateTSID13() length = %d, want 13", len(tsid))
	}
}

func TestGenerateTSID13_Alphabet(t *testing.T) {
	t.Parallel()
	tsid, err := GenerateTSID13()
	if err != nil {
		t.Fatalf("GenerateTSID13() error = %v", err)
	}
	for i, c := range tsid {
		if c > 255 || crockfordDecode[c] < 0 {
			t.Fatalf("invalid character %q at position %d", c, i)
		}
	}
}

func TestGenerateTSID13_Uppercase(t *testing.T) {
	t.Parallel()
	tsid, err := GenerateTSID13()
	if err != nil {
		t.Fatalf("GenerateTSID13() error = %v", err)
	}
	for i, c := range tsid {
		if c >= 'a' && c <= 'z' {
			t.Fatalf("character %q at position %d is lowercase", c, i)
		}
	}
}

func TestGenerateTSID13_TimeSortable(t *testing.T) {
	// Not parallel: mutates package-level tsidNow.
	// Use a controlled clock to guarantee different timestamps
	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	original := tsidNow
	defer func() { tsidNow = original }()

	tsidNow = func() time.Time { return baseTime }
	id1, err := GenerateTSID13()
	if err != nil {
		t.Fatalf("first GenerateTSID13() error = %v", err)
	}

	tsidNow = func() time.Time { return baseTime.Add(10 * time.Millisecond) }
	id2, err := GenerateTSID13()
	if err != nil {
		t.Fatalf("second GenerateTSID13() error = %v", err)
	}

	if id1 >= id2 {
		t.Fatalf("IDs not time-sorted: %q >= %q", id1, id2)
	}
}

func TestGenerateTSID13_Uniqueness(t *testing.T) {
	// Not parallel: mutates package-level tsidNow.
	// Use an incrementing clock so each call gets a distinct millisecond,
	// avoiding birthday-bound collisions within the 15-bit random space.
	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	callCount := 0

	original := tsidNow
	defer func() { tsidNow = original }()

	tsidNow = func() time.Time {
		t := baseTime.Add(time.Duration(callCount) * time.Millisecond)
		callCount++
		return t
	}

	seen := make(map[string]struct{}, 10000)
	for i := 0; i < 10000; i++ {
		tsid, err := GenerateTSID13()
		if err != nil {
			t.Fatalf("GenerateTSID13() error at iteration %d: %v", i, err)
		}
		if _, exists := seen[tsid]; exists {
			t.Fatalf("duplicate TSID at iteration %d: %q", i, tsid)
		}
		seen[tsid] = struct{}{}
	}
}

func TestValidateTSID13_Valid(t *testing.T) {
	t.Parallel()
	if err := ValidateTSID13("01J3K7MXP3RT5"); err != nil {
		t.Fatalf("ValidateTSID13() error = %v", err)
	}
}

func TestValidateTSID13_WrongLength(t *testing.T) {
	t.Parallel()
	if err := ValidateTSID13("01J3K7MXP3RT"); err == nil {
		t.Fatal("ValidateTSID13() error = nil, want non-nil for 12 chars")
	}
}

func TestValidateTSID13_InvalidChar(t *testing.T) {
	t.Parallel()
	// 'U' is not in Crockford alphabet
	if err := ValidateTSID13("01J3K7MXP3RTU"); err == nil {
		t.Fatal("ValidateTSID13() error = nil, want non-nil for invalid char U")
	}
}

func TestNormalizeTSID_CrockfordSubstitution(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"O to 0", "01J3K7MXP3RTO", "01J3K7MXP3RT0"},
		{"l to 1", "01J3K7MXP3RTl", "01J3K7MXP3RT1"},
		{"i to 1", "01J3K7MXP3RTi", "01J3K7MXP3RT1"},
		{"L to 1 with lowercase", "01j3k7mxp3rtL", "01J3K7MXP3RT1"},
		{"no substitution needed", "01J3K7MXP3RT5", "01J3K7MXP3RT5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := NormalizeTSID(tt.input)
			if err != nil {
				t.Fatalf("NormalizeTSID(%q) error = %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("NormalizeTSID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeTSID_Lowercase(t *testing.T) {
	t.Parallel()
	got, err := NormalizeTSID("01j3k7mxp3rt5")
	if err != nil {
		t.Fatalf("NormalizeTSID() error = %v", err)
	}
	if got != "01J3K7MXP3RT5" {
		t.Fatalf("NormalizeTSID() = %q, want %q", got, "01J3K7MXP3RT5")
	}
}
