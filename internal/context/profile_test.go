package context

import (
	"os"
	"path/filepath"
	"testing"
)

func writeProfileFile(t *testing.T, dir, filename, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0o644); err != nil {
		t.Fatalf("write profile file: %v", err)
	}
}

func TestProfileStore_Load_valid(t *testing.T) {
	dir := t.TempDir()
	writeProfileFile(t, dir, "base.yaml", `
id: base
description: "Project-wide conventions for all agents"
conventions:
  - "Error handling: wrap errors with fmt.Errorf and %w"
  - "Tests: table-driven, use t.TempDir() for filesystem tests"
`)

	store := NewProfileStore(dir)
	p, err := store.Load("base")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.ID != "base" {
		t.Errorf("ID: got %q, want %q", p.ID, "base")
	}
	if p.Description != "Project-wide conventions for all agents" {
		t.Errorf("Description: got %q", p.Description)
	}
	convs, ok := p.Conventions.([]interface{})
	if !ok || len(convs) != 2 {
		t.Errorf("Conventions: got %v (type %T), want []interface{} with 2 items", p.Conventions, p.Conventions)
	}
	if p.Packages != nil {
		t.Errorf("Packages: expected nil, got %v", p.Packages)
	}
	if p.Inherits != "" {
		t.Errorf("Inherits: expected empty, got %q", p.Inherits)
	}
}

func TestProfileStore_Load_withInherits(t *testing.T) {
	dir := t.TempDir()
	writeProfileFile(t, dir, "developer.yaml", `
id: developer
inherits: base
description: "General development conventions"
packages:
  - internal/
  - cmd/
conventions:
  - "Use go fmt before committing"
`)

	store := NewProfileStore(dir)
	p, err := store.Load("developer")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.Inherits != "base" {
		t.Errorf("Inherits: got %q, want %q", p.Inherits, "base")
	}
	if len(p.Packages) != 2 {
		t.Errorf("Packages: got %d items, want 2", len(p.Packages))
	}
}

func TestProfileStore_Load_withArchitecture(t *testing.T) {
	dir := t.TempDir()
	writeProfileFile(t, dir, "backend.yaml", `
id: backend
description: "Backend service conventions"
architecture:
  summary: "Layered service architecture"
  key_interfaces:
    - EntityService
    - DocumentService
`)

	store := NewProfileStore(dir)
	p, err := store.Load("backend")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.Architecture == nil {
		t.Fatal("Architecture: expected non-nil")
	}
	if p.Architecture.Summary != "Layered service architecture" {
		t.Errorf("Architecture.Summary: got %q", p.Architecture.Summary)
	}
	if len(p.Architecture.KeyInterfaces) != 2 {
		t.Errorf("Architecture.KeyInterfaces: got %d items, want 2", len(p.Architecture.KeyInterfaces))
	}
}

func TestProfileStore_Load_notFound(t *testing.T) {
	dir := t.TempDir()
	store := NewProfileStore(dir)

	_, err := store.Load("missing")
	if err == nil {
		t.Fatal("expected error for missing profile, got nil")
	}
}

func TestProfileStore_Load_missingDescription(t *testing.T) {
	dir := t.TempDir()
	writeProfileFile(t, dir, "nodesc.yaml", `
id: nodesc
packages:
  - internal/
`)

	store := NewProfileStore(dir)
	_, err := store.Load("nodesc")
	if err == nil {
		t.Fatal("expected error for missing description, got nil")
	}
}

func TestProfileStore_Load_missingID(t *testing.T) {
	dir := t.TempDir()
	writeProfileFile(t, dir, "noid.yaml", `
description: "A profile without an id field"
`)

	store := NewProfileStore(dir)
	_, err := store.Load("noid")
	if err == nil {
		t.Fatal("expected error for missing id, got nil")
	}
}

func TestProfileStore_Load_idMismatch(t *testing.T) {
	dir := t.TempDir()
	// File is named "other.yaml" but id field says "wrong"
	writeProfileFile(t, dir, "other.yaml", `
id: wrong
description: "ID does not match filename"
`)

	store := NewProfileStore(dir)
	_, err := store.Load("other")
	if err == nil {
		t.Fatal("expected error for id/filename mismatch, got nil")
	}
}

func TestProfileStore_Load_invalidYAML(t *testing.T) {
	dir := t.TempDir()
	writeProfileFile(t, dir, "broken.yaml", `
id: broken
description: [this is not valid: yaml: structure
  - oops
`)

	store := NewProfileStore(dir)
	_, err := store.Load("broken")
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestProfileStore_LoadAll_empty(t *testing.T) {
	dir := t.TempDir()
	store := NewProfileStore(dir)

	profiles, err := store.LoadAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(profiles) != 0 {
		t.Errorf("expected 0 profiles, got %d", len(profiles))
	}
}

func TestProfileStore_LoadAll_directoryNotExist(t *testing.T) {
	store := NewProfileStore("/nonexistent/path/to/roles")

	profiles, err := store.LoadAll()
	if err != nil {
		t.Fatalf("expected no error for missing directory, got: %v", err)
	}
	if profiles != nil {
		t.Errorf("expected nil slice, got %v", profiles)
	}
}

func TestProfileStore_LoadAll_multipleProfiles(t *testing.T) {
	dir := t.TempDir()
	writeProfileFile(t, dir, "base.yaml", `
id: base
description: "Base profile"
conventions:
  - "Wrap errors"
`)
	writeProfileFile(t, dir, "developer.yaml", `
id: developer
inherits: base
description: "Developer profile"
packages:
  - internal/
`)

	// Non-YAML file should be ignored.
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# ignore me"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}

	store := NewProfileStore(dir)
	profiles, err := store.LoadAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(profiles) != 2 {
		t.Errorf("expected 2 profiles, got %d", len(profiles))
	}
}

func TestProfileID_validation(t *testing.T) {
	cases := []struct {
		id    string
		valid bool
	}{
		{"a", false},         // 1-char rejected: minimum is 2 chars
		{"ab", true},         // 2-char alphanumeric
		{"abc", true},        // 3-char alphanumeric
		{"my-role", true},    // hyphens in middle
		{"dev-ops-2", true},  // trailing digit
		{"a1", true},         // digit suffix
		{"base", true},       // normal word
		{"-bad", false},      // leading hyphen
		{"bad-", false},      // trailing hyphen
		{"Bad", false},       // uppercase
		{"", false},          // empty
		{"has space", false}, // space
		{"has_under", false}, // underscore
	}

	for _, tc := range cases {
		matched := idRegexp.MatchString(tc.id)
		if matched != tc.valid {
			t.Errorf("id %q: got valid=%v, want valid=%v", tc.id, matched, tc.valid)
		}
	}
}
