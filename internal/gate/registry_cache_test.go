package gate

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

const minimalBindingYAML = `stage_bindings:
  designing:
    description: "Design phase"
    orchestration: single-agent
    roles: [architect]
    skills: [write-design]
    prerequisites:
      documents:
        - type: design
          status: approved
      override_policy: agent
  specifying:
    description: "Spec phase"
    orchestration: single-agent
    roles: [spec-author]
    skills: [write-spec]
    prerequisites:
      override_policy: checkpoint
`

func writeBindingFile(t *testing.T, dir string) string {
	t.Helper()
	p := filepath.Join(dir, "stage-bindings.yaml")
	if err := os.WriteFile(p, []byte(minimalBindingYAML), 0o644); err != nil {
		t.Fatalf("writing test binding file: %v", err)
	}
	return p
}

func TestGet_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := writeBindingFile(t, dir)

	cache := NewRegistryCache(path)
	bf, err := cache.Get()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bf == nil {
		t.Fatal("expected non-nil BindingFile")
	}
	if _, ok := bf.StageBindings["designing"]; !ok {
		t.Error("expected 'designing' stage in bindings")
	}
	if _, ok := bf.StageBindings["specifying"]; !ok {
		t.Error("expected 'specifying' stage in bindings")
	}
}

func TestGet_MissingFile(t *testing.T) {
	cache := NewRegistryCache(filepath.Join(t.TempDir(), "nonexistent.yaml"))
	bf, err := cache.Get()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bf != nil {
		t.Fatal("expected nil BindingFile for missing file")
	}
}

func TestGet_CacheHit(t *testing.T) {
	dir := t.TempDir()
	path := writeBindingFile(t, dir)

	cache := NewRegistryCache(path)

	bf1, err := cache.Get()
	if err != nil {
		t.Fatalf("first Get: %v", err)
	}
	if bf1 == nil {
		t.Fatal("first Get returned nil")
	}

	bf2, err := cache.Get()
	if err != nil {
		t.Fatalf("second Get: %v", err)
	}

	// Same pointer means the cache was reused, not re-parsed.
	if bf1 != bf2 {
		t.Error("expected second Get to return the same pointer (cache hit)")
	}
}

func TestGet_FileDeletedAfterLoad(t *testing.T) {
	dir := t.TempDir()
	path := writeBindingFile(t, dir)

	cache := NewRegistryCache(path)

	bf, err := cache.Get()
	if err != nil {
		t.Fatalf("initial Get: %v", err)
	}
	if bf == nil {
		t.Fatal("initial Get returned nil")
	}

	if err := os.Remove(path); err != nil {
		t.Fatalf("removing file: %v", err)
	}

	bf, err = cache.Get()
	if err != nil {
		t.Fatalf("Get after delete: %v", err)
	}
	if bf != nil {
		t.Error("expected nil after file deletion")
	}
}

func TestGet_MtimeChangeTriggersReload(t *testing.T) {
	dir := t.TempDir()
	path := writeBindingFile(t, dir)

	cache := NewRegistryCache(path)

	bf1, err := cache.Get()
	if err != nil {
		t.Fatalf("first Get: %v", err)
	}
	if bf1 == nil {
		t.Fatal("first Get returned nil")
	}

	// Advance the mtime by 2 seconds so the cache sees a change.
	now := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(path, now, now); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	bf2, err := cache.Get()
	if err != nil {
		t.Fatalf("second Get: %v", err)
	}
	if bf2 == nil {
		t.Fatal("second Get returned nil")
	}

	// Different pointer means the file was re-parsed.
	if bf1 == bf2 {
		t.Error("expected different pointer after mtime change (cache miss)")
	}
}

func TestGet_ConcurrentAccess(t *testing.T) {
	dir := t.TempDir()
	path := writeBindingFile(t, dir)

	cache := NewRegistryCache(path)

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)
	errs := make(chan error, goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			bf, err := cache.Get()
			if err != nil {
				errs <- err
				return
			}
			if bf == nil {
				errs <- nil // not an error per se, but unexpected
				return
			}
			if _, ok := bf.StageBindings["designing"]; !ok {
				errs <- os.ErrInvalid
			}
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Errorf("concurrent Get returned error: %v", err)
		}
	}
}

func TestLookupPrereqs_Found(t *testing.T) {
	dir := t.TempDir()
	path := writeBindingFile(t, dir)

	cache := NewRegistryCache(path)

	prereqs, ok := cache.LookupPrereqs("designing")
	if !ok {
		t.Fatal("expected ok=true for 'designing'")
	}
	if prereqs == nil {
		t.Fatal("expected non-nil prerequisites")
	}
	if len(prereqs.Documents) != 1 {
		t.Fatalf("expected 1 document prereq, got %d", len(prereqs.Documents))
	}
	if prereqs.Documents[0].Type != "design" {
		t.Errorf("expected document type 'design', got %q", prereqs.Documents[0].Type)
	}
}

func TestLookupPrereqs_UnknownStage(t *testing.T) {
	dir := t.TempDir()
	path := writeBindingFile(t, dir)

	cache := NewRegistryCache(path)

	prereqs, ok := cache.LookupPrereqs("nonexistent")
	if ok {
		t.Error("expected ok=false for unknown stage")
	}
	if prereqs != nil {
		t.Error("expected nil prerequisites for unknown stage")
	}
}

func TestLookupPrereqs_MissingFile(t *testing.T) {
	cache := NewRegistryCache(filepath.Join(t.TempDir(), "nope.yaml"))

	prereqs, ok := cache.LookupPrereqs("designing")
	if ok {
		t.Error("expected ok=false for missing file")
	}
	if prereqs != nil {
		t.Error("expected nil prerequisites for missing file")
	}
}

func TestLookupOverridePolicy_Found(t *testing.T) {
	dir := t.TempDir()
	path := writeBindingFile(t, dir)

	cache := NewRegistryCache(path)

	policy, ok := cache.LookupOverridePolicy("specifying")
	if !ok {
		t.Fatal("expected ok=true for 'specifying'")
	}
	if policy != "checkpoint" {
		t.Errorf("expected policy 'checkpoint', got %q", policy)
	}
}

func TestLookupOverridePolicy_DefaultForUnknownStage(t *testing.T) {
	dir := t.TempDir()
	path := writeBindingFile(t, dir)

	cache := NewRegistryCache(path)

	policy, ok := cache.LookupOverridePolicy("nonexistent")
	if ok {
		t.Error("expected ok=false for unknown stage")
	}
	if policy != "agent" {
		t.Errorf("expected default policy 'agent', got %q", policy)
	}
}

func TestLookupOverridePolicy_DefaultForMissingFile(t *testing.T) {
	cache := NewRegistryCache(filepath.Join(t.TempDir(), "nope.yaml"))

	policy, ok := cache.LookupOverridePolicy("designing")
	if ok {
		t.Error("expected ok=false for missing file")
	}
	if policy != "agent" {
		t.Errorf("expected default policy 'agent', got %q", policy)
	}
}
