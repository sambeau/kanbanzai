package context

import (
	"path/filepath"
	"testing"
)

func TestResolveToolHint(t *testing.T) {
	// Set up roles: grandparent -> parent -> child, plus unrelated.
	tmpDir := t.TempDir()
	newRoot := filepath.Join(tmpDir, "roles")
	legacyRoot := filepath.Join(tmpDir, "context", "roles")

	writeRoleFile(t, newRoot, "grandparent", validRoleYAML("grandparent"))
	writeRoleFile(t, newRoot, "parent", validRoleWithInherits("parent", "grandparent"))
	writeRoleFile(t, newRoot, "child", validRoleWithInherits("child", "parent"))
	writeRoleFile(t, newRoot, "unrelated", validRoleYAML("unrelated"))

	store := NewRoleStore(newRoot, legacyRoot)

	tests := []struct {
		name   string
		hints  map[string]string
		roleID string
		want   string
	}{
		{
			name:   "exact match",
			hints:  map[string]string{"child": "child-hint"},
			roleID: "child",
			want:   "child-hint",
		},
		{
			name:   "inherited match from parent",
			hints:  map[string]string{"parent": "parent-hint"},
			roleID: "child",
			want:   "parent-hint",
		},
		{
			name:   "inherited match from grandparent",
			hints:  map[string]string{"grandparent": "gp-hint"},
			roleID: "child",
			want:   "gp-hint",
		},
		{
			name:   "exact takes priority over inherited",
			hints:  map[string]string{"child": "child-hint", "parent": "parent-hint"},
			roleID: "child",
			want:   "child-hint",
		},
		{
			name:   "no match for unrelated role",
			hints:  map[string]string{"child": "child-hint"},
			roleID: "unrelated",
			want:   "",
		},
		{
			name:   "empty hints map",
			hints:  map[string]string{},
			roleID: "child",
			want:   "",
		},
		{
			name:   "nil hints map",
			hints:  nil,
			roleID: "child",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveToolHint(tt.hints, tt.roleID, store)
			if got != tt.want {
				t.Errorf("ResolveToolHint() = %q, want %q", got, tt.want)
			}
		})
	}
}
