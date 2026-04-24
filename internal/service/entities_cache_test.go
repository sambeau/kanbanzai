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

// TestUpdateEntity_CacheReflectsNewSlug verifies FR-003: after UpdateEntity
// renames the slug, cache.LookupByID returns the new slug, not the old one.
// The cache row is upserted in-place (keyed on entity_type+id), so no stale
// row with the old slug is left behind.
func TestUpdateEntity_CacheReflectsNewSlug(t *testing.T) {
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

	planID := "P1-slug-rename"
	writeTestPlan(t, svc, planID)

	feat, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "original-slug",
		Parent:    planID,
		Summary:   "Feature for slug rename cache test",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}

	// Sanity check: the original slug is in the cache after creation.
	gotSlug, _, found := c.LookupByID(feat.Type, feat.ID)
	if !found {
		t.Fatal("cache.LookupByID() found = false before rename; want true")
	}
	if gotSlug != "original-slug" {
		t.Fatalf("slug before rename = %q, want %q", gotSlug, "original-slug")
	}

	// Rename the slug via UpdateEntity.
	updated, err := svc.UpdateEntity(UpdateEntityInput{
		Type:   feat.Type,
		ID:     feat.ID,
		Slug:   feat.Slug,
		Fields: map[string]string{"slug": "renamed-slug"},
	})
	if err != nil {
		t.Fatalf("UpdateEntity() error = %v", err)
	}
	if updated.Slug != "renamed-slug" {
		t.Fatalf("UpdateEntity() returned slug = %q, want %q", updated.Slug, "renamed-slug")
	}

	// FR-003: the cache row must now carry the new slug.
	gotSlug, _, found = c.LookupByID(feat.Type, feat.ID)
	if !found {
		t.Fatal("cache.LookupByID() found = false after rename; want true")
	}
	if gotSlug != "renamed-slug" {
		t.Fatalf("cached slug after rename = %q, want %q", gotSlug, "renamed-slug")
	}
}

// TestUpdateEntity_NilCache_NoPanic verifies FR-004: UpdateEntity completes
// normally when no cache is configured (s.cache == nil).
func TestUpdateEntity_NilCache_NoPanic(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")
	// No SetCache call — svc.cache remains nil.

	planID := "P1-update-nil-cache"
	writeTestPlan(t, svc, planID)

	feat, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "nil-cache-update-feat",
		Parent:    planID,
		Summary:   "Feature for UpdateEntity nil cache test",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}

	// UpdateEntity must not panic with a nil cache.
	updated, err := svc.UpdateEntity(UpdateEntityInput{
		Type:   feat.Type,
		ID:     feat.ID,
		Slug:   feat.Slug,
		Fields: map[string]string{"summary": "Updated summary"},
	})
	if err != nil {
		t.Fatalf("UpdateEntity() error = %v", err)
	}
	if got := updated.State["summary"]; got != "Updated summary" {
		t.Fatalf("summary after UpdateEntity = %v, want %q", got, "Updated summary")
	}
}

// TestGet_NilCache_NoPanic verifies FR-004: Get completes normally when no
// cache is configured (s.cache == nil).
func TestGet_NilCache_NoPanic(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")
	// No SetCache call.

	planID := "P1-get-nil-cache"
	writeTestPlan(t, svc, planID)

	feat, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "nil-cache-get-feat",
		Parent:    planID,
		Summary:   "Feature for Get nil cache test",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}

	got, err := svc.Get(feat.Type, feat.ID, feat.Slug)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != feat.ID {
		t.Fatalf("Get() id = %q, want %q", got.ID, feat.ID)
	}
}

// TestList_NilCache_NoPanic verifies FR-004: List completes normally when no
// cache is configured (s.cache == nil).
func TestList_NilCache_NoPanic(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")
	// No SetCache call.

	planID := "P1-list-nil-cache"
	writeTestPlan(t, svc, planID)

	_, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "nil-cache-list-feat",
		Parent:    planID,
		Summary:   "Feature for List nil cache test",
		CreatedBy: "sam",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}

	results, err := svc.List("feature")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(results) == 0 {
		t.Fatal("List() returned no results; want at least one")
	}
}

// TestCacheDelete_EvictsRow verifies FR-001/FR-002: cache.Delete is the
// correct eviction API. After inserting a row and calling Delete, LookupByID
// returns found=false.
func TestCacheDelete_EvictsRow(t *testing.T) {
	t.Parallel()

	cacheDir := t.TempDir()
	c, err := cache.Open(cacheDir)
	if err != nil {
		t.Fatalf("cache.Open() error = %v", err)
	}
	defer c.Close()

	row := cache.EntityRow{
		EntityType: "task",
		ID:         "TASK-01TESTEVICT001",
		Slug:       "evict-me",
		Status:     "queued",
		Title:      "Evict Me",
		FilePath:   "/fake/path.yaml",
		FieldsJSON: `{"status":"queued"}`,
	}

	if err := c.Upsert(row); err != nil {
		t.Fatalf("cache.Upsert() error = %v", err)
	}

	// Confirm the row is present before deletion.
	_, _, found := c.LookupByID(row.EntityType, row.ID)
	if !found {
		t.Fatal("cache.LookupByID() found = false before Delete; want true")
	}

	// Delete must evict the row.
	if err := c.Delete(row.EntityType, row.ID); err != nil {
		t.Fatalf("cache.Delete() error = %v", err)
	}

	_, _, found = c.LookupByID(row.EntityType, row.ID)
	if found {
		t.Fatal("cache.LookupByID() found = true after Delete; want false")
	}
}

// TestCacheDelete_NonExistentRow verifies FR-001/FR-002: calling cache.Delete
// on a row that does not exist returns nil, not an error.
func TestCacheDelete_NonExistentRow(t *testing.T) {
	t.Parallel()

	cacheDir := t.TempDir()
	c, err := cache.Open(cacheDir)
	if err != nil {
		t.Fatalf("cache.Open() error = %v", err)
	}
	defer c.Close()

	// Delete an ID that was never inserted.
	if err := c.Delete("task", "TASK-01DOESNOTEXIST"); err != nil {
		t.Fatalf("cache.Delete() on non-existent row error = %v; want nil", err)
	}
}
