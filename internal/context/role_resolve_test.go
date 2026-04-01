package context

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupRoleStore(t *testing.T, roles map[string]string) *RoleStore {
	t.Helper()
	tmpDir := t.TempDir()
	newRoot := filepath.Join(tmpDir, "roles")
	legacyRoot := filepath.Join(tmpDir, "context", "roles")

	if err := os.MkdirAll(newRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	for id, content := range roles {
		if err := os.WriteFile(filepath.Join(newRoot, id+".yaml"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return NewRoleStore(newRoot, legacyRoot)
}

func TestResolveRoleChain(t *testing.T) {
	tests := []struct {
		name        string
		roles       map[string]string
		resolveID   string
		wantChain   []string // expected IDs from root to leaf
		wantErr     bool
		errContains string
	}{
		{
			name: "single role (no inheritance)",
			roles: map[string]string{
				"base": "id: base\nidentity: \"Base engineer\"\nvocabulary:\n  - foundations\n",
			},
			resolveID: "base",
			wantChain: []string{"base"},
		},
		{
			name: "two-level chain",
			roles: map[string]string{
				"base":     "id: base\nidentity: \"Base engineer\"\nvocabulary:\n  - foundations\n",
				"reviewer": "id: reviewer\ninherits: base\nidentity: \"Code reviewer\"\nvocabulary:\n  - review patterns\n",
			},
			resolveID: "reviewer",
			wantChain: []string{"base", "reviewer"},
		},
		{
			name: "three-level chain",
			roles: map[string]string{
				"base":              "id: base\nidentity: \"Base engineer\"\nvocabulary:\n  - foundations\n",
				"reviewer":          "id: reviewer\ninherits: base\nidentity: \"Code reviewer\"\nvocabulary:\n  - review patterns\n",
				"reviewer-security": "id: reviewer-security\ninherits: reviewer\nidentity: \"Security reviewer\"\nvocabulary:\n  - OWASP\n",
			},
			resolveID: "reviewer-security",
			wantChain: []string{"base", "reviewer", "reviewer-security"},
		},
		{
			name: "direct cycle detected",
			roles: map[string]string{
				"role-a": "id: role-a\ninherits: role-b\nidentity: \"A\"\nvocabulary:\n  - term-a\n",
				"role-b": "id: role-b\ninherits: role-a\nidentity: \"B\"\nvocabulary:\n  - term-b\n",
			},
			resolveID:   "role-a",
			wantErr:     true,
			errContains: "cycle",
		},
		{
			name: "transitive cycle detected",
			roles: map[string]string{
				"role-a": "id: role-a\ninherits: role-b\nidentity: \"A\"\nvocabulary:\n  - a\n",
				"role-b": "id: role-b\ninherits: role-c\nidentity: \"B\"\nvocabulary:\n  - b\n",
				"role-c": "id: role-c\ninherits: role-a\nidentity: \"C\"\nvocabulary:\n  - c\n",
			},
			resolveID:   "role-a",
			wantErr:     true,
			errContains: "cycle",
		},
		{
			name: "missing parent role",
			roles: map[string]string{
				"child": "id: child\ninherits: nonexistent\nidentity: \"Child\"\nvocabulary:\n  - term\n",
			},
			resolveID:   "child",
			wantErr:     true,
			errContains: "not found",
		},
		{
			name:        "nonexistent role",
			roles:       map[string]string{},
			resolveID:   "missing",
			wantErr:     true,
			errContains: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := setupRoleStore(t, tt.roles)
			chain, err := ResolveRoleChain(store, tt.resolveID)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(chain) != len(tt.wantChain) {
				t.Fatalf("expected chain length %d, got %d", len(tt.wantChain), len(chain))
			}
			for i, wantID := range tt.wantChain {
				if chain[i].ID != wantID {
					t.Errorf("chain[%d].ID = %q, want %q", i, chain[i].ID, wantID)
				}
			}
		})
	}
}

func TestResolveRole(t *testing.T) {
	tests := []struct {
		name             string
		roles            map[string]string
		resolveID        string
		wantID           string
		wantIdentity     string
		wantVocabulary   []string
		wantAntiPatterns int
		wantTools        []string
		wantErr          bool
		errContains      string
	}{
		{
			name: "single role resolves to itself",
			roles: map[string]string{
				"base": "id: base\nidentity: \"Base engineer\"\nvocabulary:\n  - foundations\n  - testing\ntools:\n  - entity\n  - grep\n",
			},
			resolveID:      "base",
			wantID:         "base",
			wantIdentity:   "Base engineer",
			wantVocabulary: []string{"foundations", "testing"},
			wantTools:      []string{"entity", "grep"},
		},
		{
			name: "vocabulary concatenation (parent first, child appended)",
			roles: map[string]string{
				"base":     "id: base\nidentity: \"Base\"\nvocabulary:\n  - parent-a\n  - parent-b\n",
				"reviewer": "id: reviewer\ninherits: base\nidentity: \"Reviewer\"\nvocabulary:\n  - child-a\n  - child-b\n",
			},
			resolveID:      "reviewer",
			wantID:         "reviewer",
			wantIdentity:   "Reviewer",
			wantVocabulary: []string{"parent-a", "parent-b", "child-a", "child-b"},
		},
		{
			name: "anti_patterns concatenation (parent first, child appended)",
			roles: map[string]string{
				"base":  "id: base\nidentity: \"Base\"\nvocabulary:\n  - term\nanti_patterns:\n  - name: parent-ap\n    detect: pd\n    because: pb\n    resolve: pr\n",
				"child": "id: child\ninherits: base\nidentity: \"Child\"\nvocabulary:\n  - term\nanti_patterns:\n  - name: child-ap\n    detect: cd\n    because: cb\n    resolve: cr\n",
			},
			resolveID:        "child",
			wantID:           "child",
			wantIdentity:     "Child",
			wantAntiPatterns: 2,
		},
		{
			name: "tools union (no duplicates)",
			roles: map[string]string{
				"base":  "id: base\nidentity: \"Base\"\nvocabulary:\n  - term\ntools:\n  - entity\n  - knowledge\n",
				"child": "id: child\ninherits: base\nidentity: \"Child\"\nvocabulary:\n  - term\ntools:\n  - entity\n  - grep\n",
			},
			resolveID:    "child",
			wantID:       "child",
			wantIdentity: "Child",
			wantTools:    []string{"entity", "knowledge", "grep"},
		},
		{
			name: "identity always from leaf",
			roles: map[string]string{
				"base":  "id: base\nidentity: \"Parent identity\"\nvocabulary:\n  - term\n",
				"child": "id: child\ninherits: base\nidentity: \"Child identity\"\nvocabulary:\n  - term\n",
			},
			resolveID:    "child",
			wantID:       "child",
			wantIdentity: "Child identity",
		},
		{
			name: "three-level chain full merge",
			roles: map[string]string{
				"grandparent": "id: grandparent\nidentity: \"Grandparent\"\nvocabulary:\n  - gp-term\ntools:\n  - entity\nanti_patterns:\n  - name: gp-ap\n    detect: gd\n    because: gb\n    resolve: gr\n",
				"parent":      "id: parent\ninherits: grandparent\nidentity: \"Parent\"\nvocabulary:\n  - p-term\ntools:\n  - entity\n  - knowledge\nanti_patterns:\n  - name: p-ap\n    detect: pd\n    because: pb\n    resolve: pr\n",
				"child":       "id: child\ninherits: parent\nidentity: \"Child\"\nvocabulary:\n  - c-term\ntools:\n  - grep\n  - knowledge\nanti_patterns:\n  - name: c-ap\n    detect: cd\n    because: cb\n    resolve: cr\n",
			},
			resolveID:        "child",
			wantID:           "child",
			wantIdentity:     "Child",
			wantVocabulary:   []string{"gp-term", "p-term", "c-term"},
			wantAntiPatterns: 3,
			wantTools:        []string{"entity", "knowledge", "grep"},
		},
		{
			name: "parent with no tools, child with tools",
			roles: map[string]string{
				"base":  "id: base\nidentity: \"Base\"\nvocabulary:\n  - term\n",
				"child": "id: child\ninherits: base\nidentity: \"Child\"\nvocabulary:\n  - term\ntools:\n  - entity\n",
			},
			resolveID:    "child",
			wantID:       "child",
			wantIdentity: "Child",
			wantTools:    []string{"entity"},
		},
		{
			name: "parent with tools, child with no tools",
			roles: map[string]string{
				"base":  "id: base\nidentity: \"Base\"\nvocabulary:\n  - term\ntools:\n  - entity\n",
				"child": "id: child\ninherits: base\nidentity: \"Child\"\nvocabulary:\n  - term\n",
			},
			resolveID:    "child",
			wantID:       "child",
			wantIdentity: "Child",
			wantTools:    []string{"entity"},
		},
		{
			name: "parent with anti_patterns, child with none",
			roles: map[string]string{
				"base":  "id: base\nidentity: \"Base\"\nvocabulary:\n  - term\nanti_patterns:\n  - name: ap\n    detect: d\n    because: b\n    resolve: r\n",
				"child": "id: child\ninherits: base\nidentity: \"Child\"\nvocabulary:\n  - term\n",
			},
			resolveID:        "child",
			wantAntiPatterns: 1,
		},
		{
			name: "cycle produces error",
			roles: map[string]string{
				"role-a": "id: role-a\ninherits: role-b\nidentity: \"A\"\nvocabulary:\n  - a\n",
				"role-b": "id: role-b\ninherits: role-a\nidentity: \"B\"\nvocabulary:\n  - b\n",
			},
			resolveID:   "role-a",
			wantErr:     true,
			errContains: "cycle",
		},
		{
			name: "missing parent produces error",
			roles: map[string]string{
				"child": "id: child\ninherits: missing\nidentity: \"Child\"\nvocabulary:\n  - term\n",
			},
			resolveID:   "child",
			wantErr:     true,
			errContains: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := setupRoleStore(t, tt.roles)
			resolved, err := ResolveRole(store, tt.resolveID)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantID != "" && resolved.ID != tt.wantID {
				t.Errorf("ID = %q, want %q", resolved.ID, tt.wantID)
			}

			if tt.wantIdentity != "" && resolved.Identity != tt.wantIdentity {
				t.Errorf("Identity = %q, want %q", resolved.Identity, tt.wantIdentity)
			}

			if tt.wantVocabulary != nil {
				if len(resolved.Vocabulary) != len(tt.wantVocabulary) {
					t.Fatalf("Vocabulary length = %d, want %d; got %v",
						len(resolved.Vocabulary), len(tt.wantVocabulary), resolved.Vocabulary)
				}
				for i, want := range tt.wantVocabulary {
					if resolved.Vocabulary[i] != want {
						t.Errorf("Vocabulary[%d] = %q, want %q", i, resolved.Vocabulary[i], want)
					}
				}
			}

			if tt.wantAntiPatterns > 0 {
				if len(resolved.AntiPatterns) != tt.wantAntiPatterns {
					t.Errorf("AntiPatterns count = %d, want %d",
						len(resolved.AntiPatterns), tt.wantAntiPatterns)
				}
			}

			if tt.wantTools != nil {
				if len(resolved.Tools) != len(tt.wantTools) {
					t.Fatalf("Tools length = %d, want %d; got %v",
						len(resolved.Tools), len(tt.wantTools), resolved.Tools)
				}
				for i, want := range tt.wantTools {
					if resolved.Tools[i] != want {
						t.Errorf("Tools[%d] = %q, want %q", i, resolved.Tools[i], want)
					}
				}
			}
		})
	}
}

func TestResolveRoleAntiPatternOrder(t *testing.T) {
	roles := map[string]string{
		"base":  "id: base\nidentity: \"Base\"\nvocabulary:\n  - term\nanti_patterns:\n  - name: parent-first\n    detect: d\n    because: b\n    resolve: r\n  - name: parent-second\n    detect: d\n    because: b\n    resolve: r\n",
		"child": "id: child\ninherits: base\nidentity: \"Child\"\nvocabulary:\n  - term\nanti_patterns:\n  - name: child-first\n    detect: d\n    because: b\n    resolve: r\n",
	}

	store := setupRoleStore(t, roles)
	resolved, err := ResolveRole(store, "child")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedOrder := []string{"parent-first", "parent-second", "child-first"}
	if len(resolved.AntiPatterns) != len(expectedOrder) {
		t.Fatalf("expected %d anti-patterns, got %d", len(expectedOrder), len(resolved.AntiPatterns))
	}
	for i, want := range expectedOrder {
		if resolved.AntiPatterns[i].Name != want {
			t.Errorf("AntiPatterns[%d].Name = %q, want %q", i, resolved.AntiPatterns[i].Name, want)
		}
	}
}

func TestResolveRoleToolsUnionPreservesOrder(t *testing.T) {
	roles := map[string]string{
		"base":  "id: base\nidentity: \"Base\"\nvocabulary:\n  - term\ntools:\n  - alpha\n  - beta\n  - gamma\n",
		"child": "id: child\ninherits: base\nidentity: \"Child\"\nvocabulary:\n  - term\ntools:\n  - beta\n  - delta\n  - alpha\n",
	}

	store := setupRoleStore(t, roles)
	resolved, err := ResolveRole(store, "child")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expected: parent tools in order, then child additions in order (skipping dupes).
	expectedTools := []string{"alpha", "beta", "gamma", "delta"}
	if len(resolved.Tools) != len(expectedTools) {
		t.Fatalf("expected %d tools, got %d: %v", len(expectedTools), len(resolved.Tools), resolved.Tools)
	}
	for i, want := range expectedTools {
		if resolved.Tools[i] != want {
			t.Errorf("Tools[%d] = %q, want %q", i, resolved.Tools[i], want)
		}
	}
}

func TestMergeToolsUnion(t *testing.T) {
	tests := []struct {
		name      string
		base      []string
		additions []string
		want      []string
	}{
		{
			name:      "both empty",
			base:      nil,
			additions: nil,
			want:      []string{},
		},
		{
			name:      "base only",
			base:      []string{"a", "b"},
			additions: nil,
			want:      []string{"a", "b"},
		},
		{
			name:      "additions only",
			base:      nil,
			additions: []string{"a", "b"},
			want:      []string{"a", "b"},
		},
		{
			name:      "no overlap",
			base:      []string{"a", "b"},
			additions: []string{"c", "d"},
			want:      []string{"a", "b", "c", "d"},
		},
		{
			name:      "full overlap",
			base:      []string{"a", "b"},
			additions: []string{"b", "a"},
			want:      []string{"a", "b"},
		},
		{
			name:      "partial overlap preserves order",
			base:      []string{"a", "b", "c"},
			additions: []string{"b", "d", "a", "e"},
			want:      []string{"a", "b", "c", "d", "e"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeToolsUnion(tt.base, tt.additions)
			if len(got) != len(tt.want) {
				t.Fatalf("length = %d, want %d; got %v", len(got), len(tt.want), got)
			}
			for i, want := range tt.want {
				if got[i] != want {
					t.Errorf("result[%d] = %q, want %q", i, got[i], want)
				}
			}
		})
	}
}

func TestResolveRoleWithLegacyLocation(t *testing.T) {
	tmpDir := t.TempDir()
	newRoot := filepath.Join(tmpDir, "roles")
	legacyRoot := filepath.Join(tmpDir, "context", "roles")

	// Put the base role in legacy location.
	if err := os.MkdirAll(legacyRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyRoot, "base.yaml"),
		[]byte("id: base\nidentity: \"Legacy base\"\nvocabulary:\n  - legacy-term\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Put the child role in new location.
	if err := os.MkdirAll(newRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(newRoot, "child.yaml"),
		[]byte("id: child\ninherits: base\nidentity: \"New child\"\nvocabulary:\n  - new-term\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	store := NewRoleStore(newRoot, legacyRoot)
	resolved, err := ResolveRole(store, "child")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved.ID != "child" {
		t.Errorf("ID = %q, want 'child'", resolved.ID)
	}
	if resolved.Identity != "New child" {
		t.Errorf("Identity = %q, want 'New child'", resolved.Identity)
	}

	expectedVocab := []string{"legacy-term", "new-term"}
	if len(resolved.Vocabulary) != len(expectedVocab) {
		t.Fatalf("Vocabulary length = %d, want %d", len(resolved.Vocabulary), len(expectedVocab))
	}
	for i, want := range expectedVocab {
		if resolved.Vocabulary[i] != want {
			t.Errorf("Vocabulary[%d] = %q, want %q", i, resolved.Vocabulary[i], want)
		}
	}
}
