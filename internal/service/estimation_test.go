package service

import (
	"strings"
	"testing"
)

func TestIsValidEstimate(t *testing.T) {
	t.Parallel()

	validValues := []float64{0, 0.5, 1, 2, 3, 5, 8, 13, 20, 40, 100}
	for _, v := range validValues {
		v := v
		t.Run("valid", func(t *testing.T) {
			t.Parallel()
			if !IsValidEstimate(v) {
				t.Errorf("IsValidEstimate(%v) = false, want true", v)
			}
		})
	}

	invalidValues := []float64{-1, 0.1, 0.25, 4, 6, 7, 9, 10, 11, 12, 14, 15, 21, 50, 99, 101, 200}
	for _, v := range invalidValues {
		v := v
		t.Run("invalid", func(t *testing.T) {
			t.Parallel()
			if IsValidEstimate(v) {
				t.Errorf("IsValidEstimate(%v) = true, want false", v)
			}
		})
	}
}

func TestIsValidEstimate_AllScaleValues(t *testing.T) {
	t.Parallel()

	for _, v := range EstimationScale {
		if !IsValidEstimate(v) {
			t.Errorf("IsValidEstimate(%v) = false for value in EstimationScale", v)
		}
	}
}

func TestIsValidEstimate_ZeroIsValid(t *testing.T) {
	t.Parallel()

	if !IsValidEstimate(0) {
		t.Error("IsValidEstimate(0) = false, want true (0 is a valid estimate)")
	}
}

func TestIsValidEstimate_HalfPointIsValid(t *testing.T) {
	t.Parallel()

	if !IsValidEstimate(0.5) {
		t.Error("IsValidEstimate(0.5) = false, want true")
	}
}

func TestValidateEstimate_Valid(t *testing.T) {
	t.Parallel()

	for _, v := range EstimationScale {
		v := v
		t.Run("valid", func(t *testing.T) {
			t.Parallel()
			if err := ValidateEstimate(v); err != nil {
				t.Errorf("ValidateEstimate(%v) returned error: %v", v, err)
			}
		})
	}
}

func TestValidateEstimate_Invalid(t *testing.T) {
	t.Parallel()

	invalidCases := []struct {
		estimate float64
		wantFrag string // substring expected in error message
	}{
		{-1, "invalid estimate"},
		{4, "invalid estimate"},
		{6, "invalid estimate"},
		{7, "must be one of"},
		{99, "must be one of"},
		{0.25, "invalid estimate"},
	}

	for _, tc := range invalidCases {
		tc := tc
		t.Run("invalid", func(t *testing.T) {
			t.Parallel()
			err := ValidateEstimate(tc.estimate)
			if err == nil {
				t.Fatalf("ValidateEstimate(%v) = nil, want error", tc.estimate)
			}
			if !strings.Contains(err.Error(), tc.wantFrag) {
				t.Errorf("ValidateEstimate(%v) error = %q, want it to contain %q", tc.estimate, err.Error(), tc.wantFrag)
			}
		})
	}
}

func TestValidateEstimate_ErrorFormat(t *testing.T) {
	t.Parallel()

	err := ValidateEstimate(7)
	if err == nil {
		t.Fatal("ValidateEstimate(7) = nil, want error")
	}

	msg := err.Error()
	if !strings.Contains(msg, "invalid estimate") {
		t.Errorf("error message %q missing 'invalid estimate'", msg)
	}
	if !strings.Contains(msg, "7.0") {
		t.Errorf("error message %q missing the estimate value '7.0'", msg)
	}
	if !strings.Contains(msg, "must be one of") {
		t.Errorf("error message %q missing 'must be one of'", msg)
	}
}

func TestSoftLimitWarning_NoWarning(t *testing.T) {
	t.Parallel()

	cases := []struct {
		entityType string
		estimate   float64
	}{
		{"task", 0},
		{"task", 1},
		{"task", 5},
		{"task", 13},
		{"bug", 13},
		{"feature", 20},
		{"feature", 100},
		{"epic", 40},
		{"epic", 100},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.entityType, func(t *testing.T) {
			t.Parallel()
			got := SoftLimitWarning(tc.entityType, tc.estimate)
			if got != "" {
				t.Errorf("SoftLimitWarning(%q, %v) = %q, want empty", tc.entityType, tc.estimate, got)
			}
		})
	}
}

func TestSoftLimitWarning_ReturnsWarning(t *testing.T) {
	t.Parallel()

	cases := []struct {
		entityType string
		estimate   float64
	}{
		{"task", 20},
		{"task", 40},
		{"task", 100},
		{"bug", 20},
		{"bug", 40},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.entityType, func(t *testing.T) {
			t.Parallel()
			got := SoftLimitWarning(tc.entityType, tc.estimate)
			if got == "" {
				t.Errorf("SoftLimitWarning(%q, %v) = empty, want non-empty warning", tc.entityType, tc.estimate)
			}
			if !strings.Contains(got, "exceeds the soft limit") {
				t.Errorf("SoftLimitWarning(%q, %v) = %q, want it to mention 'exceeds the soft limit'", tc.entityType, tc.estimate, got)
			}
			if !strings.Contains(got, "consider decomposing") {
				t.Errorf("SoftLimitWarning(%q, %v) = %q, want it to mention 'consider decomposing'", tc.entityType, tc.estimate, got)
			}
		})
	}
}

func TestSoftLimitWarning_UnknownEntityType(t *testing.T) {
	t.Parallel()

	got := SoftLimitWarning("unknown-type", 1000)
	if got != "" {
		t.Errorf("SoftLimitWarning(%q, 1000) = %q, want empty for unknown type", "unknown-type", got)
	}
}

func TestSoftLimitWarning_CaseInsensitive(t *testing.T) {
	t.Parallel()

	// Should match regardless of case
	lower := SoftLimitWarning("task", 20)
	upper := SoftLimitWarning("TASK", 20)
	mixed := SoftLimitWarning("Task", 20)

	if lower == "" {
		t.Error("SoftLimitWarning(\"task\", 20) = empty, want warning")
	}
	if upper == "" {
		t.Error("SoftLimitWarning(\"TASK\", 20) = empty, want warning")
	}
	if mixed == "" {
		t.Error("SoftLimitWarning(\"Task\", 20) = empty, want warning")
	}
}

func TestSoftLimitWarning_TaskLimit13(t *testing.T) {
	t.Parallel()

	// 13 is the limit, so it should NOT warn
	at := SoftLimitWarning("task", 13)
	if at != "" {
		t.Errorf("SoftLimitWarning(\"task\", 13) = %q, want empty (13 is the limit, not exceeding)", at)
	}

	// 20 is above limit, should warn
	above := SoftLimitWarning("task", 20)
	if above == "" {
		t.Error("SoftLimitWarning(\"task\", 20) = empty, want warning")
	}
}

func TestGetScaleEntries(t *testing.T) {
	t.Parallel()

	entries := GetScaleEntries()

	// Should have the same count as EstimationScale
	if len(entries) != len(EstimationScale) {
		t.Fatalf("GetScaleEntries() len = %d, want %d", len(entries), len(EstimationScale))
	}

	// Each entry should match the corresponding scale value
	for i, entry := range entries {
		wantPoints := EstimationScale[i]
		if entry.Points != wantPoints {
			t.Errorf("entries[%d].Points = %v, want %v", i, entry.Points, wantPoints)
		}
		wantMeaning := EstimationScaleMeanings[wantPoints]
		if entry.Meaning != wantMeaning {
			t.Errorf("entries[%d].Meaning = %q, want %q", i, entry.Meaning, wantMeaning)
		}
	}
}

func TestGetScaleEntries_AllHaveMeanings(t *testing.T) {
	t.Parallel()

	entries := GetScaleEntries()
	for _, entry := range entries {
		if entry.Meaning == "" {
			t.Errorf("scale entry for points=%v has empty meaning", entry.Points)
		}
	}
}

func TestGetScaleEntries_OrderedAscending(t *testing.T) {
	t.Parallel()

	entries := GetScaleEntries()
	for i := 1; i < len(entries); i++ {
		if entries[i].Points <= entries[i-1].Points {
			t.Errorf("entries[%d].Points = %v <= entries[%d].Points = %v: not ascending",
				i, entries[i].Points, i-1, entries[i-1].Points)
		}
	}
}

func TestGetScaleEntries_ContainsZero(t *testing.T) {
	t.Parallel()

	entries := GetScaleEntries()
	if len(entries) == 0 {
		t.Fatal("GetScaleEntries() returned empty slice")
	}
	if entries[0].Points != 0 {
		t.Errorf("entries[0].Points = %v, want 0", entries[0].Points)
	}
}

func TestGetScaleEntries_ContainsHundred(t *testing.T) {
	t.Parallel()

	entries := GetScaleEntries()
	last := entries[len(entries)-1]
	if last.Points != 100 {
		t.Errorf("last entry Points = %v, want 100", last.Points)
	}
}

func TestGetEstimateFromFields_Nil(t *testing.T) {
	t.Parallel()

	fields := map[string]any{}
	got := GetEstimateFromFields(fields)
	if got != nil {
		t.Errorf("GetEstimateFromFields(empty) = %v, want nil", got)
	}
}

func TestGetEstimateFromFields_NilValue(t *testing.T) {
	t.Parallel()

	fields := map[string]any{"estimate": nil}
	got := GetEstimateFromFields(fields)
	if got != nil {
		t.Errorf("GetEstimateFromFields({estimate: nil}) = %v, want nil", got)
	}
}

func TestGetEstimateFromFields_Float64(t *testing.T) {
	t.Parallel()

	fields := map[string]any{"estimate": float64(5)}
	got := GetEstimateFromFields(fields)
	if got == nil {
		t.Fatal("GetEstimateFromFields returned nil, want *float64")
	}
	if *got != 5 {
		t.Errorf("*GetEstimateFromFields = %v, want 5", *got)
	}
}

func TestGetEstimateFromFields_Int(t *testing.T) {
	t.Parallel()

	// YAML round-trips integer estimates (0, 1, 2...) as int
	fields := map[string]any{"estimate": int(8)}
	got := GetEstimateFromFields(fields)
	if got == nil {
		t.Fatal("GetEstimateFromFields returned nil for int estimate, want *float64")
	}
	if *got != 8 {
		t.Errorf("*GetEstimateFromFields = %v, want 8", *got)
	}
}

func TestGetEstimateFromFields_StringRoundTrip(t *testing.T) {
	t.Parallel()

	// YAML round-trips numbers as quoted strings (because needsQuotes=true for numerics)
	cases := []struct {
		raw  string
		want float64
	}{
		{"0.5", 0.5},
		{"1", 1},
		{"13", 13},
		{"0", 0},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.raw, func(t *testing.T) {
			t.Parallel()
			fields := map[string]any{"estimate": tc.raw}
			got := GetEstimateFromFields(fields)
			if got == nil {
				t.Fatalf("GetEstimateFromFields({estimate: %q}) = nil, want *float64", tc.raw)
			}
			if *got != tc.want {
				t.Errorf("*GetEstimateFromFields = %v, want %v", *got, tc.want)
			}
		})
	}
}

func TestGetEstimateFromFields_InvalidString(t *testing.T) {
	t.Parallel()

	fields := map[string]any{"estimate": "not-a-number"}
	got := GetEstimateFromFields(fields)
	if got != nil {
		t.Errorf("GetEstimateFromFields({estimate: 'not-a-number'}) = %v, want nil", got)
	}
}
