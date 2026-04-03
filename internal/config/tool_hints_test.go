package config

import (
	"testing"
)

func TestMergeToolHints(t *testing.T) {
	tests := []struct {
		name    string
		project map[string]string
		local   map[string]string
		want    map[string]string
	}{
		{
			name:    "both nil",
			project: nil,
			local:   nil,
			want:    nil,
		},
		{
			name:    "project only",
			project: map[string]string{"architect": "use entity tool"},
			local:   nil,
			want:    map[string]string{"architect": "use entity tool"},
		},
		{
			name:    "local only",
			project: nil,
			local:   map[string]string{"reviewer": "be strict"},
			want:    map[string]string{"reviewer": "be strict"},
		},
		{
			name:    "disjoint keys",
			project: map[string]string{"architect": "hint-a"},
			local:   map[string]string{"reviewer": "hint-b"},
			want:    map[string]string{"architect": "hint-a", "reviewer": "hint-b"},
		},
		{
			name:    "local wins on same key",
			project: map[string]string{"architect": "project-hint"},
			local:   map[string]string{"architect": "local-hint"},
			want:    map[string]string{"architect": "local-hint"},
		},
		{
			name:    "empty project non-empty local",
			project: map[string]string{},
			local:   map[string]string{"reviewer": "hint"},
			want:    map[string]string{"reviewer": "hint"},
		},
		{
			name:    "non-empty project empty local",
			project: map[string]string{"architect": "hint"},
			local:   map[string]string{},
			want:    map[string]string{"architect": "hint"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeToolHints(tt.project, tt.local)
			if tt.want == nil {
				if got != nil {
					t.Fatalf("expected nil, got %v", got)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("expected %d entries, got %d: %v", len(tt.want), len(got), got)
			}
			for k, wantV := range tt.want {
				if gotV, ok := got[k]; !ok {
					t.Errorf("missing key %q", k)
				} else if gotV != wantV {
					t.Errorf("key %q: got %q, want %q", k, gotV, wantV)
				}
			}
		})
	}
}
