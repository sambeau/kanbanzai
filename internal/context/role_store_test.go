package context

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeRoleFile(t *testing.T, dir, id string, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, id+".yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func validRoleYAML(id string) string {
	return "id: " + id + "\n" +
		"identity: \"Senior software engineer\"\n" +
		"vocabulary:\n" +
		"  - code review\n" +
		"  - testing\n"
}

func validRoleWithInherits(id, inherits string) string {
	return "id: " + id + "\n" +
		"inherits: " + inherits + "\n" +
		"identity: \"Specialist engineer\"\n" +
		"vocabulary:\n" +
		"  - specialist term\n"
}

func validRoleWithTools(id string) string {
	return "id: " + id + "\n" +
		"identity: \"Engineer with tools\"\n" +
		"vocabulary:\n" +
		"  - tooling\n" +
		"tools:\n" +
		"  - entity\n" +
		"  - grep\n"
}

func TestRoleStoreLoad(t *testing.T) {
	tests := []struct {
		name        string
		setupNew    func(t *testing.T, dir string)
		setupLegacy func(t *testing.T, dir string)
		loadID      string
		wantErr     bool
		errContains string
		checkRole   func(t *testing.T, r *Role)
	}{
		{
			name: "load from new location",
			setupNew: func(t *testing.T, dir string) {
				writeRoleFile(t, dir, "base", validRoleYAML("base"))
			},
			loadID:  "base",
			wantErr: false,
			checkRole: func(t *testing.T, r *Role) {
				if r.ID != "base" {
					t.Errorf("expected id 'base', got %q", r.ID)
				}
				if r.Identity != "Senior software engineer" {
					t.Errorf("unexpected identity: %q", r.Identity)
				}
				if len(r.Vocabulary) != 2 {
					t.Errorf("expected 2 vocabulary entries, got %d", len(r.Vocabulary))
				}
			},
		},
		{
			name: "fallback to legacy location",
			setupLegacy: func(t *testing.T, dir string) {
				writeRoleFile(t, dir, "reviewer", validRoleYAML("reviewer"))
			},
			loadID:  "reviewer",
			wantErr: false,
			checkRole: func(t *testing.T, r *Role) {
				if r.ID != "reviewer" {
					t.Errorf("expected id 'reviewer', got %q", r.ID)
				}
			},
		},
		{
			name: "new location takes precedence over legacy",
			setupNew: func(t *testing.T, dir string) {
				writeRoleFile(t, dir, "base",
					"id: base\nidentity: \"New location engineer\"\nvocabulary:\n  - new\n")
			},
			setupLegacy: func(t *testing.T, dir string) {
				writeRoleFile(t, dir, "base",
					"id: base\nidentity: \"Legacy location engineer\"\nvocabulary:\n  - legacy\n")
			},
			loadID:  "base",
			wantErr: false,
			checkRole: func(t *testing.T, r *Role) {
				if r.Identity != "New location engineer" {
					t.Errorf("expected new location identity, got %q", r.Identity)
				}
			},
		},
		{
			name:        "role not found in either location",
			loadID:      "nonexistent",
			wantErr:     true,
			errContains: "not found",
		},
		{
			name:        "invalid id format rejected",
			loadID:      "BAD_ID",
			wantErr:     true,
			errContains: "invalid role id",
		},
		{
			name:        "single character id rejected",
			loadID:      "a",
			wantErr:     true,
			errContains: "invalid role id",
		},
		{
			name: "id-filename mismatch rejected",
			setupNew: func(t *testing.T, dir string) {
				writeRoleFile(t, dir, "wrong-name",
					"id: correct-name\nidentity: \"Engineer\"\nvocabulary:\n  - term\n")
			},
			loadID:      "wrong-name",
			wantErr:     true,
			errContains: "does not match filename",
		},
		{
			name: "unknown YAML field rejected (strict parsing)",
			setupNew: func(t *testing.T, dir string) {
				writeRoleFile(t, dir, "strict-test",
					"id: strict-test\nidentity: \"Engineer\"\nvocabulary:\n  - term\nextra_field: bad\n")
			},
			loadID:      "strict-test",
			wantErr:     true,
			errContains: "strict-test",
		},
		{
			name: "missing required fields reported",
			setupNew: func(t *testing.T, dir string) {
				writeRoleFile(t, dir, "incomplete", "id: incomplete\n")
			},
			loadID:      "incomplete",
			wantErr:     true,
			errContains: "identity",
		},
		{
			name: "role with inherits field loads",
			setupNew: func(t *testing.T, dir string) {
				writeRoleFile(t, dir, "child", validRoleWithInherits("child", "parent"))
			},
			loadID:  "child",
			wantErr: false,
			checkRole: func(t *testing.T, r *Role) {
				if r.Inherits != "parent" {
					t.Errorf("expected inherits 'parent', got %q", r.Inherits)
				}
			},
		},
		{
			name: "role with tools loads",
			setupNew: func(t *testing.T, dir string) {
				writeRoleFile(t, dir, "tooled", validRoleWithTools("tooled"))
			},
			loadID:  "tooled",
			wantErr: false,
			checkRole: func(t *testing.T, r *Role) {
				if len(r.Tools) != 2 {
					t.Errorf("expected 2 tools, got %d", len(r.Tools))
				}
			},
		},
		{
			name: "role with anti_patterns loads",
			setupNew: func(t *testing.T, dir string) {
				writeRoleFile(t, dir, "ap-role",
					"id: ap-role\nidentity: \"Engineer\"\nvocabulary:\n  - term\n"+
						"anti_patterns:\n"+
						"  - name: Bad Pattern\n"+
						"    detect: signal\n"+
						"    because: reason\n"+
						"    resolve: fix\n")
			},
			loadID:  "ap-role",
			wantErr: false,
			checkRole: func(t *testing.T, r *Role) {
				if len(r.AntiPatterns) != 1 {
					t.Fatalf("expected 1 anti-pattern, got %d", len(r.AntiPatterns))
				}
				if r.AntiPatterns[0].Name != "Bad Pattern" {
					t.Errorf("expected anti-pattern name 'Bad Pattern', got %q", r.AntiPatterns[0].Name)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			newRoot := filepath.Join(tmpDir, "roles")
			legacyRoot := filepath.Join(tmpDir, "context", "roles")

			if tt.setupNew != nil {
				tt.setupNew(t, newRoot)
			}
			if tt.setupLegacy != nil {
				tt.setupLegacy(t, legacyRoot)
			}

			store := NewRoleStore(newRoot, legacyRoot)
			role, err := store.Load(tt.loadID)

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
			if role == nil {
				t.Fatal("expected non-nil role")
			}
			if tt.checkRole != nil {
				tt.checkRole(t, role)
			}
		})
	}
}

func TestRoleStoreLoadAll(t *testing.T) {
	tests := []struct {
		name        string
		setupNew    func(t *testing.T, dir string)
		setupLegacy func(t *testing.T, dir string)
		wantCount   int
		wantIDs     []string
		wantErr     bool
		errContains string
	}{
		{
			name:      "no directories exist returns empty",
			wantCount: 0,
		},
		{
			name: "roles from new location only",
			setupNew: func(t *testing.T, dir string) {
				writeRoleFile(t, dir, "base", validRoleYAML("base"))
				writeRoleFile(t, dir, "reviewer", validRoleYAML("reviewer"))
			},
			wantCount: 2,
			wantIDs:   []string{"base", "reviewer"},
		},
		{
			name: "roles from legacy location only",
			setupLegacy: func(t *testing.T, dir string) {
				writeRoleFile(t, dir, "base", validRoleYAML("base"))
			},
			wantCount: 1,
			wantIDs:   []string{"base"},
		},
		{
			name: "new location takes precedence in LoadAll",
			setupNew: func(t *testing.T, dir string) {
				writeRoleFile(t, dir, "base",
					"id: base\nidentity: \"New engineer\"\nvocabulary:\n  - new\n")
			},
			setupLegacy: func(t *testing.T, dir string) {
				writeRoleFile(t, dir, "base",
					"id: base\nidentity: \"Legacy engineer\"\nvocabulary:\n  - legacy\n")
			},
			wantCount: 1,
			wantIDs:   []string{"base"},
		},
		{
			name: "roles from both locations merged (no overlap)",
			setupNew: func(t *testing.T, dir string) {
				writeRoleFile(t, dir, "base", validRoleYAML("base"))
			},
			setupLegacy: func(t *testing.T, dir string) {
				writeRoleFile(t, dir, "reviewer", validRoleYAML("reviewer"))
			},
			wantCount: 2,
			wantIDs:   []string{"base", "reviewer"},
		},
		{
			name: "invalid role in new location returns error",
			setupNew: func(t *testing.T, dir string) {
				writeRoleFile(t, dir, "bad", "id: bad\n")
			},
			wantErr:     true,
			errContains: "identity",
		},
		{
			name: "non-yaml files skipped",
			setupNew: func(t *testing.T, dir string) {
				writeRoleFile(t, dir, "base", validRoleYAML("base"))
				// Write a non-yaml file
				if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# roles"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantCount: 1,
			wantIDs:   []string{"base"},
		},
		{
			name: "subdirectories skipped",
			setupNew: func(t *testing.T, dir string) {
				writeRoleFile(t, dir, "base", validRoleYAML("base"))
				if err := os.MkdirAll(filepath.Join(dir, "subdir"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			wantCount: 1,
			wantIDs:   []string{"base"},
		},
		{
			// Regression test for the ProfileStore coexistence bug: the legacy
			// directory (.kbz/context/roles/) is shared with ProfileStore which
			// writes old-format YAML (description/conventions/architecture fields
			// not present in Role). When a new-location counterpart exists the
			// old-format file must be skipped without returning an error.
			name: "old-format legacy file superseded by new-location file is skipped without error",
			setupNew: func(t *testing.T, dir string) {
				writeRoleFile(t, dir, "base", validRoleYAML("base"))
			},
			setupLegacy: func(t *testing.T, dir string) {
				oldFormat := "id: base\ndescription: \"Project conventions\"\nconventions: []\narchitecture:\n  summary: \"arch\"\n"
				writeRoleFile(t, dir, "base", oldFormat)
			},
			wantCount: 1,
			wantIDs:   []string{"base"},
		},
		{
			// Old-format legacy file with no new-location counterpart (e.g. the
			// legacy "developer" role that has no .kbz/roles/ equivalent). It must
			// be silently skipped rather than crashing LoadAll.
			name: "old-format legacy file with no new-location counterpart is silently skipped",
			setupLegacy: func(t *testing.T, dir string) {
				oldFormat := "id: developer\ninherits: base\ndescription: \"Developer conventions\"\npackages:\n  - internal/\nconventions: []\n"
				writeRoleFile(t, dir, "developer", oldFormat)
			},
			wantCount: 0,
		},
		{
			// A new-format role file placed in the legacy directory (the intended
			// backward-compat path for roles not yet migrated to .kbz/roles/) must
			// still be loaded successfully.
			name: "new-format role in legacy directory only is loaded successfully",
			setupLegacy: func(t *testing.T, dir string) {
				writeRoleFile(t, dir, "custom", validRoleYAML("custom"))
			},
			wantCount: 1,
			wantIDs:   []string{"custom"},
		},
		{
			// Invalid new-format role in the legacy directory (missing required
			// fields) must be silently skipped in lenient mode, not hard-fail.
			name: "invalid new-format role in legacy directory is silently skipped",
			setupLegacy: func(t *testing.T, dir string) {
				// Valid YAML but missing identity and vocabulary.
				writeRoleFile(t, dir, "incomplete", "id: incomplete\n")
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			newRoot := filepath.Join(tmpDir, "roles")
			legacyRoot := filepath.Join(tmpDir, "context", "roles")

			if tt.setupNew != nil {
				tt.setupNew(t, newRoot)
			}
			if tt.setupLegacy != nil {
				tt.setupLegacy(t, legacyRoot)
			}

			store := NewRoleStore(newRoot, legacyRoot)
			roles, err := store.LoadAll()

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
			if len(roles) != tt.wantCount {
				t.Errorf("expected %d roles, got %d", tt.wantCount, len(roles))
			}

			if tt.wantIDs != nil {
				gotIDs := make(map[string]bool)
				for _, r := range roles {
					gotIDs[r.ID] = true
				}
				for _, wantID := range tt.wantIDs {
					if !gotIDs[wantID] {
						t.Errorf("expected role %q not found in results", wantID)
					}
				}
			}
		})
	}
}

func TestRoleStoreExists(t *testing.T) {
	tests := []struct {
		name        string
		setupNew    func(t *testing.T, dir string)
		setupLegacy func(t *testing.T, dir string)
		checkID     string
		want        bool
	}{
		{
			name: "exists in new location",
			setupNew: func(t *testing.T, dir string) {
				writeRoleFile(t, dir, "base", validRoleYAML("base"))
			},
			checkID: "base",
			want:    true,
		},
		{
			name: "exists in legacy location",
			setupLegacy: func(t *testing.T, dir string) {
				writeRoleFile(t, dir, "reviewer", validRoleYAML("reviewer"))
			},
			checkID: "reviewer",
			want:    true,
		},
		{
			name:    "does not exist in either location",
			checkID: "nonexistent",
			want:    false,
		},
		{
			name:    "invalid id returns false",
			checkID: "BAD",
			want:    false,
		},
		{
			name:    "empty id returns false",
			checkID: "",
			want:    false,
		},
		{
			name:    "single char id returns false",
			checkID: "x",
			want:    false,
		},
		{
			name: "exists in both locations returns true",
			setupNew: func(t *testing.T, dir string) {
				writeRoleFile(t, dir, "base", validRoleYAML("base"))
			},
			setupLegacy: func(t *testing.T, dir string) {
				writeRoleFile(t, dir, "base", validRoleYAML("base"))
			},
			checkID: "base",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			newRoot := filepath.Join(tmpDir, "roles")
			legacyRoot := filepath.Join(tmpDir, "context", "roles")

			if tt.setupNew != nil {
				tt.setupNew(t, newRoot)
			}
			if tt.setupLegacy != nil {
				tt.setupLegacy(t, legacyRoot)
			}

			store := NewRoleStore(newRoot, legacyRoot)
			got := store.Exists(tt.checkID)
			if got != tt.want {
				t.Errorf("Exists(%q) = %v, want %v", tt.checkID, got, tt.want)
			}
		})
	}
}

func TestRoleStoreLoadPrecedenceVerifiesContent(t *testing.T) {
	tmpDir := t.TempDir()
	newRoot := filepath.Join(tmpDir, "roles")
	legacyRoot := filepath.Join(tmpDir, "context", "roles")

	writeRoleFile(t, newRoot, "base",
		"id: base\nidentity: \"New location\"\nvocabulary:\n  - new-term\n")
	writeRoleFile(t, legacyRoot, "base",
		"id: base\nidentity: \"Legacy location\"\nvocabulary:\n  - legacy-term\n")

	store := NewRoleStore(newRoot, legacyRoot)

	// Single Load should prefer new location.
	role, err := store.Load("base")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if role.Identity != "New location" {
		t.Errorf("Load: expected 'New location', got %q", role.Identity)
	}
	if len(role.Vocabulary) != 1 || role.Vocabulary[0] != "new-term" {
		t.Errorf("Load: expected vocabulary [new-term], got %v", role.Vocabulary)
	}

	// LoadAll should also prefer new location.
	roles, err := store.LoadAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(roles) != 1 {
		t.Fatalf("expected 1 role (deduplicated), got %d", len(roles))
	}
	if roles[0].Identity != "New location" {
		t.Errorf("LoadAll: expected 'New location', got %q", roles[0].Identity)
	}
}
