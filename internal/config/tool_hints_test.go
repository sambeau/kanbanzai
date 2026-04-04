package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigParse_NoToolHints(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	yamlContent := `version: "2"
prefixes:
  - prefix: P
    name: Plan
`
	if err := os.WriteFile(cfgPath, []byte(yamlContent), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	loaded, err := LoadFrom(cfgPath)
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}

	if loaded.ToolHints != nil {
		t.Errorf("expected ToolHints to be nil, got %v", loaded.ToolHints)
	}
}

func TestConfigParse_WithToolHints(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	yamlContent := `version: "2"
prefixes:
  - prefix: P
    name: Plan
tool_hints:
  implementer-go: "Use search_graph"
`
	if err := os.WriteFile(cfgPath, []byte(yamlContent), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	loaded, err := LoadFrom(cfgPath)
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}

	if loaded.ToolHints == nil {
		t.Fatal("expected ToolHints to be non-nil")
	}
	if got := loaded.ToolHints["implementer-go"]; got != "Use search_graph" {
		t.Errorf("ToolHints[\"implementer-go\"] = %q, want %q", got, "Use search_graph")
	}
}

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
