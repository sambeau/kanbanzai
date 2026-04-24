package service

import (
	"path/filepath"
	"testing"

	"github.com/sambeau/kanbanzai/internal/cache"
)

// TestUpdateStatus_CacheUpsertOnTerminalTransition verifies FR-006: after
// UpdateStatus transitions a task to the terminal "done" state, the cache row
// is upserted (not evicted) — LookupByID returns found=true with the updated
// status field.
func TestUpdateStatus_CacheUpsertOnTerminalTransition(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	// Open a real cache backed by a temp SQLite file.
	cacheDir := filepath.Join(root, cache.CacheDir)
	c, err := cache.Open(cacheDir)
	if err != nil {
		t.Fatalf("cache.Open() error = %v", err)
	}
	defer c.Close()
	svc.SetCache(c)

	// Set up a plan and feature as prerequisites.
	planID := "P1-cache-terminal"
	writeTestPlan(t, svc, planID)

	feat, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "cache-terminal-feat",
		Parent:    planID,
		Summary:   "Feature for cache terminal transition test",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}

	task, err := svc.CreateTask(CreateTaskInput{
		Name:          "test",
		ParentFeature: feat.ID,
		Slug:          "cache-terminal-task",
		Summary:       "Task for cache terminal transition test",
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	// Drive the task through its lifecycle to the terminal "done" state.
	transitions := []string{"ready", "active", "needs-review", "done"}
	current := task
	for _, next := range transitions {
		updated, err := svc.UpdateStatus(UpdateStatusInput{
			Type:   current.Type,
			ID:     current.ID,
			Slug:   current.Slug,
			Status: next,
		})
		if err != nil {
			t.Fatalf("UpdateStatus(-> %q) error = %v", next, err)
		}
		current = CreateResult(updated)
	}

	// The cache must still contain the entry — Delete must never be called.
	_, _, found := c.LookupByID(task.Type, task.ID)
	if !found {
		t.Fatal("cache.LookupByID() found = false after terminal transition; UpdateStatus must upsert, not evict")
	}

	// The cached fields must reflect the terminal status.
	fields, err := c.GetFields(task.Type, task.ID)
	if err != nil {
		t.Fatalf("cache.GetFields() error = %v", err)
	}
	if got := fields["status"]; got != "done" {
		t.Fatalf("cached status = %v, want %q", got, "done")
	}
}

// TestUpdateStatus_CacheUpsertOnNonTerminalTransition verifies that
// UpdateStatus upserts the cache row for non-terminal status transitions.
// After queued -> ready, LookupByID returns found=true and the status field
// in the cache matches the new status.
func TestUpdateStatus_CacheUpsertOnNonTerminalTransition(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	cacheDir := filepath.Join(root, cache.CacheDir)
	c, err := cache.Open(cacheDir)
	if err != nil {
		t.Fatalf("cache.Open() error = %v", err)
	}
	defer c.Close()
	svc.SetCache(c)

	planID := "P1-cache-nonterminal"
	writeTestPlan(t, svc, planID)

	feat, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "cache-nonterminal-feat",
		Parent:    planID,
		Summary:   "Feature for cache non-terminal transition test",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}

	task, err := svc.CreateTask(CreateTaskInput{
		Name:          "test",
		ParentFeature: feat.ID,
		Slug:          "cache-nonterminal-task",
		Summary:       "Task for cache non-terminal transition test",
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	// Transition to a non-terminal state.
	_, err = svc.UpdateStatus(UpdateStatusInput{
		Type:   task.Type,
		ID:     task.ID,
		Slug:   task.Slug,
		Status: "ready",
	})
	if err != nil {
		t.Fatalf("UpdateStatus(-> ready) error = %v", err)
	}

	_, _, found := c.LookupByID(task.Type, task.ID)
	if !found {
		t.Fatal("cache.LookupByID() found = false after non-terminal transition; want true")
	}

	fields, err := c.GetFields(task.Type, task.ID)
	if err != nil {
		t.Fatalf("cache.GetFields() error = %v", err)
	}
	if got := fields["status"]; got != "ready" {
		t.Fatalf("cached status = %v, want %q", got, "ready")
	}
}

// TestUpdateStatus_NilCache_NoPanic verifies that UpdateStatus completes
// normally when no cache is configured on the EntityService (s.cache == nil).
func TestUpdateStatus_NilCache_NoPanic(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")
	// Intentionally no SetCache call — svc.cache remains nil.

	planID := "P1-nil-cache"
	writeTestPlan(t, svc, planID)

	feat, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "nil-cache-feat",
		Parent:    planID,
		Summary:   "Feature for nil cache test",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}

	task, err := svc.CreateTask(CreateTaskInput{
		Name:          "test",
		ParentFeature: feat.ID,
		Slug:          "nil-cache-task",
		Summary:       "Task for nil cache test",
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	// UpdateStatus must not panic when the cache is nil.
	updated, err := svc.UpdateStatus(UpdateStatusInput{
		Type:   task.Type,
		ID:     task.ID,
		Slug:   task.Slug,
		Status: "ready",
	})
	if err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
	if got := updated.State["status"]; got != "ready" {
		t.Fatalf("status after UpdateStatus = %v, want %q", got, "ready")
	}
}
