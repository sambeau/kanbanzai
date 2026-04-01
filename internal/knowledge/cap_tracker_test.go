package knowledge

import (
	"path/filepath"
	"testing"
)

func TestCapTracker_ThreeConsecutiveHits(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	tracker := NewCapTracker(dir)

	for i := 0; i < 3; i++ {
		if err := tracker.RecordAssembly("backend", true); err != nil {
			t.Fatalf("RecordAssembly(%d): %v", i, err)
		}
	}

	scopes := tracker.ScopesNeedingCompaction()
	if len(scopes) != 1 {
		t.Fatalf("expected 1 scope, got %d", len(scopes))
	}
	if scopes[0].Scope != "backend" {
		t.Errorf("expected scope %q, got %q", "backend", scopes[0].Scope)
	}
	if scopes[0].ConsecutiveHits != 3 {
		t.Errorf("expected 3 consecutive hits, got %d", scopes[0].ConsecutiveHits)
	}
}

func TestCapTracker_ResetOnBelowCap(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	tracker := NewCapTracker(dir)

	_ = tracker.RecordAssembly("backend", true)
	_ = tracker.RecordAssembly("backend", true)
	_ = tracker.RecordAssembly("backend", false)

	scopes := tracker.ScopesNeedingCompaction()
	if len(scopes) != 0 {
		t.Fatalf("expected 0 scopes needing compaction, got %d", len(scopes))
	}
}

func TestCapTracker_IndependentScopes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	tracker := NewCapTracker(dir)

	for i := 0; i < 3; i++ {
		_ = tracker.RecordAssembly("backend", true)
	}
	_ = tracker.RecordAssembly("frontend", true)

	scopes := tracker.ScopesNeedingCompaction()
	if len(scopes) != 1 {
		t.Fatalf("expected 1 scope needing compaction, got %d", len(scopes))
	}
	if scopes[0].Scope != "backend" {
		t.Errorf("expected scope %q, got %q", "backend", scopes[0].Scope)
	}
}

func TestCapTracker_PersistenceAcrossInstances(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	tracker1 := NewCapTracker(dir)

	for i := 0; i < 3; i++ {
		if err := tracker1.RecordAssembly("backend", true); err != nil {
			t.Fatalf("RecordAssembly(%d): %v", i, err)
		}
	}

	tracker2 := NewCapTracker(dir)

	scopes := tracker2.ScopesNeedingCompaction()
	if len(scopes) != 1 {
		t.Fatalf("expected 1 scope from new instance, got %d", len(scopes))
	}
	if scopes[0].ConsecutiveHits != 3 {
		t.Errorf("expected 3 consecutive hits after reload, got %d", scopes[0].ConsecutiveHits)
	}
}

func TestCapTracker_EmptyScope(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	tracker := NewCapTracker(dir)

	if err := tracker.RecordAssembly("", true); err != nil {
		t.Fatalf("RecordAssembly with empty scope: %v", err)
	}

	scopes := tracker.ScopesNeedingCompaction()
	if len(scopes) != 0 {
		t.Fatalf("expected 0 scopes, got %d", len(scopes))
	}
}

func TestCapTracker_MissingFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	tracker := NewCapTracker(filepath.Join(dir, "nonexistent-subdir"))

	scopes := tracker.ScopesNeedingCompaction()
	if len(scopes) != 0 {
		t.Fatalf("expected 0 scopes for fresh tracker, got %d", len(scopes))
	}
}

func TestCapTracker_ThresholdExactlyThree(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	tracker := NewCapTracker(dir)

	_ = tracker.RecordAssembly("backend", true)
	_ = tracker.RecordAssembly("backend", true)

	scopes := tracker.ScopesNeedingCompaction()
	if len(scopes) != 0 {
		t.Fatalf("expected 0 scopes after 2 hits, got %d", len(scopes))
	}

	_ = tracker.RecordAssembly("backend", true)

	scopes = tracker.ScopesNeedingCompaction()
	if len(scopes) != 1 {
		t.Fatalf("expected 1 scope after 3rd hit, got %d", len(scopes))
	}
	if scopes[0].ConsecutiveHits != 3 {
		t.Errorf("expected exactly 3 consecutive hits, got %d", scopes[0].ConsecutiveHits)
	}
}
