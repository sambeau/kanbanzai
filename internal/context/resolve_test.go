package context

import (
	"strings"
	"testing"
)

// writeResolveProfile is a helper to write a profile YAML file in the given dir.
func writeResolveProfile(t *testing.T, dir, id, content string) {
	t.Helper()
	writeProfileFile(t, dir, id+".yaml", content)
}

func TestResolveChain_singleProfile(t *testing.T) {
	dir := t.TempDir()
	writeResolveProfile(t, dir, "solo", `
id: solo
description: "A standalone profile"
conventions:
  - "Use gofmt"
`)

	store := NewProfileStore(dir)
	chain, err := ResolveChain(store, "solo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chain) != 1 {
		t.Fatalf("expected chain length 1, got %d", len(chain))
	}
	if chain[0].ID != "solo" {
		t.Errorf("chain[0].ID: got %q, want %q", chain[0].ID, "solo")
	}
}

func TestResolveChain_twoLevels(t *testing.T) {
	dir := t.TempDir()
	writeResolveProfile(t, dir, "base", `
id: base
description: "Base profile"
conventions:
  - "Wrap errors"
`)
	writeResolveProfile(t, dir, "developer", `
id: developer
inherits: base
description: "Developer profile"
packages:
  - internal/
`)

	store := NewProfileStore(dir)
	chain, err := ResolveChain(store, "developer")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chain) != 2 {
		t.Fatalf("expected chain length 2, got %d", len(chain))
	}
	// Root ancestor is first, leaf is last.
	if chain[0].ID != "base" {
		t.Errorf("chain[0].ID: got %q, want %q", chain[0].ID, "base")
	}
	if chain[1].ID != "developer" {
		t.Errorf("chain[1].ID: got %q, want %q", chain[1].ID, "developer")
	}
}

func TestResolveChain_deepChain(t *testing.T) {
	dir := t.TempDir()
	writeResolveProfile(t, dir, "root", `
id: root
description: "Root profile"
`)
	writeResolveProfile(t, dir, "mid", `
id: mid
inherits: root
description: "Middle profile"
`)
	writeResolveProfile(t, dir, "leaf", `
id: leaf
inherits: mid
description: "Leaf profile"
`)

	store := NewProfileStore(dir)
	chain, err := ResolveChain(store, "leaf")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chain) != 3 {
		t.Fatalf("expected chain length 3, got %d", len(chain))
	}
	wantOrder := []string{"root", "mid", "leaf"}
	for i, want := range wantOrder {
		if chain[i].ID != want {
			t.Errorf("chain[%d].ID: got %q, want %q", i, chain[i].ID, want)
		}
	}
}

func TestResolveChain_cycleDetection(t *testing.T) {
	dir := t.TempDir()
	// aa → bb → aa (cycle)
	writeResolveProfile(t, dir, "aa", `
id: aa
inherits: bb
description: "Profile AA"
`)
	writeResolveProfile(t, dir, "bb", `
id: bb
inherits: aa
description: "Profile BB"
`)

	store := NewProfileStore(dir)
	_, err := ResolveChain(store, "aa")
	if err == nil {
		t.Fatal("expected cycle error, got nil")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Errorf("error should mention cycle, got: %v", err)
	}
}

func TestResolveChain_selfCycle(t *testing.T) {
	dir := t.TempDir()
	writeResolveProfile(t, dir, "self", `
id: self
inherits: self
description: "Self-referential profile"
`)

	store := NewProfileStore(dir)
	_, err := ResolveChain(store, "self")
	if err == nil {
		t.Fatal("expected cycle error for self-reference, got nil")
	}
}

func TestResolveChain_missingReference(t *testing.T) {
	dir := t.TempDir()
	writeResolveProfile(t, dir, "child", `
id: child
inherits: nonexistent
description: "Child pointing to missing parent"
`)

	store := NewProfileStore(dir)
	_, err := ResolveChain(store, "child")
	if err == nil {
		t.Fatal("expected error for missing parent reference, got nil")
	}
}

func TestResolveProfile_simpleResolution(t *testing.T) {
	dir := t.TempDir()
	writeResolveProfile(t, dir, "base", `
id: base
description: "Base profile"
conventions:
  - "Wrap errors"
  - "Use table-driven tests"
`)
	writeResolveProfile(t, dir, "developer", `
id: developer
inherits: base
description: "Developer profile"
packages:
  - internal/
  - cmd/
`)

	store := NewProfileStore(dir)
	rp, err := ResolveProfile(store, "developer")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rp.ID != "developer" {
		t.Errorf("ID: got %q, want %q", rp.ID, "developer")
	}
	// description comes from child (leaf wins)
	if rp.Description != "Developer profile" {
		t.Errorf("Description: got %q, want %q", rp.Description, "Developer profile")
	}
	// packages defined only by child
	if len(rp.Packages) != 2 {
		t.Errorf("Packages: got %d items, want 2", len(rp.Packages))
	}
	// conventions inherited from parent (child didn't set them)
	if len(rp.Conventions) != 2 {
		t.Errorf("Conventions: got %d items, want 2 (inherited from base)", len(rp.Conventions))
	}
}

func TestResolveProfile_leafReplacesListNotConcat(t *testing.T) {
	dir := t.TempDir()
	writeResolveProfile(t, dir, "base", `
id: base
description: "Base profile"
conventions:
  - "A"
  - "B"
`)
	writeResolveProfile(t, dir, "developer", `
id: developer
inherits: base
description: "Developer profile"
conventions:
  - "C"
`)

	store := NewProfileStore(dir)
	rp, err := ResolveProfile(store, "developer")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Child's conventions replace parent's entirely — should be ["C"], not ["A","B","C"].
	if len(rp.Conventions) != 1 {
		t.Errorf("Conventions: got %d items, want 1 (leaf replaces, not concatenates)", len(rp.Conventions))
	}
	if len(rp.Conventions) > 0 && rp.Conventions[0] != "C" {
		t.Errorf("Conventions[0]: got %q, want %q", rp.Conventions[0], "C")
	}
}

func TestResolveProfile_absentFieldInherited(t *testing.T) {
	dir := t.TempDir()
	writeResolveProfile(t, dir, "base", `
id: base
description: "Base profile"
packages:
  - internal/
  - cmd/
conventions:
  - "Wrap errors"
`)
	// child does NOT set packages or conventions
	writeResolveProfile(t, dir, "child", `
id: child
inherits: base
description: "Child profile with no packages or conventions"
`)

	store := NewProfileStore(dir)
	rp, err := ResolveProfile(store, "child")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// packages and conventions should be inherited from base
	if len(rp.Packages) != 2 {
		t.Errorf("Packages: got %d items, want 2 (inherited from base)", len(rp.Packages))
	}
	if len(rp.Conventions) != 1 {
		t.Errorf("Conventions: got %d items, want 1 (inherited from base)", len(rp.Conventions))
	}
}

func TestResolveProfile_deepChainResolution(t *testing.T) {
	dir := t.TempDir()
	writeResolveProfile(t, dir, "root", `
id: root
description: "Root profile"
conventions:
  - "Root convention"
packages:
  - root/
`)
	writeResolveProfile(t, dir, "mid", `
id: mid
inherits: root
description: "Mid profile"
packages:
  - mid/
`)
	writeResolveProfile(t, dir, "leaf", `
id: leaf
inherits: mid
description: "Leaf profile"
conventions:
  - "Leaf convention"
`)

	store := NewProfileStore(dir)
	rp, err := ResolveProfile(store, "leaf")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rp.ID != "leaf" {
		t.Errorf("ID: got %q, want %q", rp.ID, "leaf")
	}
	// description from leaf
	if rp.Description != "Leaf profile" {
		t.Errorf("Description: got %q", rp.Description)
	}
	// packages: mid overrode root's, leaf didn't set → mid's packages inherited
	if len(rp.Packages) != 1 || rp.Packages[0] != "mid/" {
		t.Errorf("Packages: got %v, want [mid/]", rp.Packages)
	}
	// conventions: leaf overrode root's
	if len(rp.Conventions) != 1 || rp.Conventions[0] != "Leaf convention" {
		t.Errorf("Conventions: got %v, want [Leaf convention]", rp.Conventions)
	}
}

func TestResolveProfile_architectureInheritance(t *testing.T) {
	dir := t.TempDir()
	writeResolveProfile(t, dir, "base", `
id: base
description: "Base profile"
architecture:
  summary: "Base architecture"
  key_interfaces:
    - BaseInterface
`)
	// child does NOT set architecture — should inherit from base
	writeResolveProfile(t, dir, "child", `
id: child
inherits: base
description: "Child profile"
`)

	store := NewProfileStore(dir)
	rp, err := ResolveProfile(store, "child")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rp.Architecture == nil {
		t.Fatal("Architecture: expected non-nil (inherited from base)")
	}
	if rp.Architecture.Summary != "Base architecture" {
		t.Errorf("Architecture.Summary: got %q", rp.Architecture.Summary)
	}
}

func TestResolveProfile_architectureReplacedByChild(t *testing.T) {
	dir := t.TempDir()
	writeResolveProfile(t, dir, "base", `
id: base
description: "Base profile"
architecture:
  summary: "Base architecture"
  key_interfaces:
    - BaseInterface
`)
	writeResolveProfile(t, dir, "child", `
id: child
inherits: base
description: "Child profile"
architecture:
  summary: "Child architecture"
`)

	store := NewProfileStore(dir)
	rp, err := ResolveProfile(store, "child")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rp.Architecture == nil {
		t.Fatal("Architecture: expected non-nil")
	}
	// child's architecture replaces parent's entirely — no key merging
	if rp.Architecture.Summary != "Child architecture" {
		t.Errorf("Architecture.Summary: got %q, want %q", rp.Architecture.Summary, "Child architecture")
	}
	// key_interfaces not set by child — child's map replaced parent's entirely, so nil
	if rp.Architecture.KeyInterfaces != nil {
		t.Errorf("Architecture.KeyInterfaces: expected nil (child map replaced parent map entirely), got %v", rp.Architecture.KeyInterfaces)
	}
}

func TestResolveProfile_missingProfile(t *testing.T) {
	dir := t.TempDir()
	store := NewProfileStore(dir)

	_, err := ResolveProfile(store, "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing profile, got nil")
	}
}

func TestResolveProfile_idNotInherited(t *testing.T) {
	dir := t.TempDir()
	writeResolveProfile(t, dir, "parent", `
id: parent
description: "Parent profile"
`)
	writeResolveProfile(t, dir, "child", `
id: child
inherits: parent
description: "Child profile"
`)

	store := NewProfileStore(dir)
	rp, err := ResolveProfile(store, "child")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// id must be from the leaf, not the parent
	if rp.ID != "child" {
		t.Errorf("ID: got %q, want %q (id is never inherited)", rp.ID, "child")
	}
}
