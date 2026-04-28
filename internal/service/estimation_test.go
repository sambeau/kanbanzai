package service

import "testing"

// TestGetEstimateFromFields covers all type branches: float64, int, string,
// nil value, missing key, and invalid string.
func TestGetEstimateFromFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		fields map[string]any
		want   *float64
	}{
		{
			name:   "float64 value",
			fields: map[string]any{"estimate": 5.0},
			want:   ptr(5.0),
		},
		{
			name:   "int value",
			fields: map[string]any{"estimate": 3},
			want:   ptr(3.0),
		},
		{
			name:   "string integer",
			fields: map[string]any{"estimate": "5"},
			want:   ptr(5.0),
		},
		{
			name:   "string float",
			fields: map[string]any{"estimate": "3.5"},
			want:   ptr(3.5),
		},
		{
			name:   "empty string",
			fields: map[string]any{"estimate": ""},
			want:   nil,
		},
		{
			name:   "invalid string",
			fields: map[string]any{"estimate": "not-a-number"},
			want:   nil,
		},
		{
			name:   "nil value",
			fields: map[string]any{"estimate": nil},
			want:   nil,
		},
		{
			name:   "missing key",
			fields: map[string]any{},
			want:   nil,
		},
		{
			name:   "zero float64",
			fields: map[string]any{"estimate": 0.0},
			want:   ptr(0.0),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := GetEstimateFromFields(tt.fields)
			if tt.want == nil {
				if got != nil {
					t.Errorf("GetEstimateFromFields() = %v, want nil", *got)
				}
			} else {
				if got == nil {
					t.Fatal("GetEstimateFromFields() = nil, want non-nil")
				}
				if *got != *tt.want {
					t.Errorf("GetEstimateFromFields() = %v, want %v", *got, *tt.want)
				}
			}
		})
	}
}

func ptr(f float64) *float64 {
	return &f
}
