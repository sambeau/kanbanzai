package service

import (
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/sambeau/kanbanzai/internal/cache"
)

// attachCache opens a fresh SQLite cache in a temp dir, attaches it to svc,
// and registers a cleanup to close it at the end of the test.
func attachCache(t *testing.T, svc *EntityService) *cache.Cache {
	t.Helper()
	c, err := cache.Open(filepath.Join(t.TempDir(), "cache"))
	if err != nil {
		t.Fatalf("attachCache: open cache: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	svc.SetCache(c)
	return c
}

// ---- Get() fast path tests --------------------------------------------------

// TestGet_CacheFastPath verifies that when the cache is warm for the entity type,
// Get() resolves the slug from the cache and returns the correct result.
func TestGet_CacheFastPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")
	attachCache(t, svc) // cache attached before CreateFeature so auto-upsert fires

	planID := "P1-parent"
	writeTestPlan(t, svc, planID)

	created, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "cache fast path",
		Parent:    planID,
		Summary:   "Verify cache fast path resolves slug",
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}

	if !svc.cache.IsWarm("feature") {
		t.Fatal("cache.IsWarm(\"feature\") = false, want true after CreateFeature")
	}

	// Call Get with empty slug — fast path should resolve slug from cache.
	got, err := svc.Get("feature", created.ID, "")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("Get() ID = %q, want %q", got.ID, created.ID)
	}
	if got.Slug != created.Slug {
		t.Errorf("Get() Slug = %q, want %q", got.Slug, created.Slug)
	}
	if got.Type != created.Type {
		t.Errorf("Get() Type = %q, want %q", got.Type, created.Type)
	}
}

// TestGet_CacheMiss_FallsBack verifies that when the cache is warm but has no
// row for the requested ID, Get() falls back to the ResolvePrefix filesystem path.
func TestGet_CacheMiss_FallsBack(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")
	attachCache(t, svc)

	planID := "P1-parent"
	writeTestPlan(t, svc, planID)

	created, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "cache miss feature",
		Parent:    planID,
		Summary:   "Fallback on cache miss",
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}

	// Cache is warm for "feature" but we delete this specific entity's row.
	if err := svc.cache.Delete("feature", created.ID); err != nil {
		t.Fatalf("cache.Delete() error = %v", err)
	}

	// LookupByID will miss; Get should fall through to ResolvePrefix.
	got, err := svc.Get("feature", created.ID, "")
	if err != nil {
		t.Fatalf("Get() after cache miss error = %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("Get() ID = %q, want %q", got.ID, created.ID)
	}
	if got.Slug != created.Slug {
		t.Errorf("Get() Slug = %q, want %q", got.Slug, created.Slug)
	}
}

// TestGet_StaleCache_FallsBack verifies that when LookupByID returns a hit but
// store.Load fails (e.g., the cached file path no longer exists), Get() falls
// back to ResolvePrefix rather than returning a corrupt result. When the entity
// also doesn't exist on disk, ErrNotFound is returned.
func TestGet_StaleCache_FallsBack(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")
	attachCache(t, svc)

	planID := "P1-parent"
	writeTestPlan(t, svc, planID)

	// Create a real feature so the "feature" type becomes warm in the cache.
	_, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "real feature",
		Parent:    planID,
		Summary:   "Warms the feature type in cache",
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}

	// Inject a fabricated cache row for a non-existent entity, pointing to a
	// file that doesn't exist on disk — simulating a stale cache entry.
	if err := svc.cache.Upsert(cache.EntityRow{
		EntityType: "feature",
		ID:         "FEAT-01FAKEDEADBEEF",
		Slug:       "nonexistent",
		FilePath:   "/nonexistent/path.yaml",
		FieldsJSON: `{}`,
	}); err != nil {
		t.Fatalf("cache.Upsert() stale row error = %v", err)
	}

	// LookupByID will find the stale row; store.Load will fail (file missing);
	// Get should fall back to ResolvePrefix, which also fails (entity doesn't
	// exist on disk) — expect a non-nil error rather than a corrupt result.
	_, err = svc.Get("feature", "FEAT-01FAKEDEADBEEF", "")
	if err == nil {
		t.Fatal("Get() with stale cache entry returned nil error, want an error")
	}
}

// TestGet_NilCache_UsesFilesystem verifies that Get() works correctly when no
// cache is configured, using the standard ResolvePrefix filesystem path.
func TestGet_NilCache_UsesFilesystem(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")
	// svc.cache == nil by default

	planID := "P1-parent"
	writeTestPlan(t, svc, planID)

	created, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "nil cache feature",
		Parent:    planID,
		Summary:   "Nil cache uses filesystem",
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}

	got, err := svc.Get("feature", created.ID, "")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("Get() ID = %q, want %q", got.ID, created.ID)
	}
}

// TestGet_ColdType_FallsBack verifies that when the cache is set but IsWarm
// returns false for the entity type, Get() falls back to the filesystem path.
func TestGet_ColdType_FallsBack(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	// Create feature BEFORE attaching cache so the auto-upsert doesn't fire.
	planID := "P1-parent"
	writeTestPlan(t, svc, planID)

	created, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "cold type feature",
		Parent:    planID,
		Summary:   "Cold type falls back to filesystem",
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}

	// Attach cache AFTER creation — "feature" type is cold.
	c := attachCache(t, svc)
	// Warm the cache for a different type so we can confirm the cold-type branch.
	_ = c.Upsert(cache.EntityRow{
		EntityType: "bug",
		ID:         "BUG-01AAAAAAAAAAAA",
		Slug:       "unrelated",
		FieldsJSON: `{}`,
	})

	if svc.cache.IsWarm("feature") {
		t.Fatal("cache.IsWarm(\"feature\") = true, want false (entity created before cache was attached)")
	}

	// Get with cold "feature" type should fall through to ResolvePrefix.
	got, err := svc.Get("feature", created.ID, "")
	if err != nil {
		t.Fatalf("Get() with cold type error = %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("Get() ID = %q, want %q", got.ID, created.ID)
	}
}

// ---- List() fast path tests -------------------------------------------------

// TestList_CacheFastPath verifies that when the cache is warm, List() returns
// results from the cache (correct count and fields).
func TestList_CacheFastPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")
	attachCache(t, svc)

	planID := "P1-parent"
	writeTestPlan(t, svc, planID)

	slugs := []string{"alpha", "beta", "gamma"}
	for _, s := range slugs {
		_, err := svc.CreateFeature(CreateFeatureInput{
			Name:      "test",
			Slug:      s,
			Parent:    planID,
			Summary:   "Cache list test: " + s,
			CreatedBy: "test",
		})
		if err != nil {
			t.Fatalf("CreateFeature(%s) error = %v", s, err)
		}
	}

	results, err := svc.List("feature")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(results) != 3 {
		t.Errorf("List() returned %d results, want 3", len(results))
	}
	for _, r := range results {
		if r.ID == "" {
			t.Error("List() result has empty ID")
		}
		if r.Slug == "" {
			t.Error("List() result has empty Slug")
		}
		if r.Type != "feature" {
			t.Errorf("List() result Type = %q, want %q", r.Type, "feature")
		}
	}
}

// TestList_EmptyOnWarmCache verifies that when the cache is warm for a type but
// has zero rows, List() returns an empty slice without falling back to the filesystem.
func TestList_EmptyOnWarmCache(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")
	attachCache(t, svc)

	planID := "P1-parent"
	writeTestPlan(t, svc, planID)

	// Create a feature → cache is warm for "feature" with one row.
	created, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "to be evicted",
		Parent:    planID,
		Summary:   "Will be removed from cache",
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}

	// Delete from cache: warm map stays true, but 0 rows for "feature".
	if err := svc.cache.Delete("feature", created.ID); err != nil {
		t.Fatalf("cache.Delete() error = %v", err)
	}

	results, err := svc.List("feature")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	// Cache is warm with 0 rows — must return empty, not fall back to filesystem.
	if len(results) != 0 {
		t.Errorf("List() returned %d results on warm empty cache, want 0 (no filesystem fallback)", len(results))
	}
}

// TestList_NilCache_UsesFilesystem verifies that List() works correctly when no
// cache is configured, scanning the filesystem via filepath.Glob.
func TestList_NilCache_UsesFilesystem(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")
	// svc.cache == nil

	planID := "P1-parent"
	writeTestPlan(t, svc, planID)

	for _, slug := range []string{"one", "two"} {
		_, err := svc.CreateFeature(CreateFeatureInput{
			Name:      "test",
			Slug:      slug,
			Parent:    planID,
			Summary:   "Filesystem list: " + slug,
			CreatedBy: "test",
		})
		if err != nil {
			t.Fatalf("CreateFeature(%s) error = %v", slug, err)
		}
	}

	results, err := svc.List("feature")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(results) != 2 {
		t.Errorf("List() returned %d results, want 2", len(results))
	}
}

// TestList_ColdType_FallsBack verifies that List() falls back to filepath.Glob
// when the cache is set but IsWarm returns false for the entity type.
func TestList_ColdType_FallsBack(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	// Create features BEFORE attaching cache so the type stays cold.
	planID := "P1-parent"
	writeTestPlan(t, svc, planID)

	for _, slug := range []string{"first", "second"} {
		_, err := svc.CreateFeature(CreateFeatureInput{
			Name:      "test",
			Slug:      slug,
			Parent:    planID,
			Summary:   "Cold-type fallback: " + slug,
			CreatedBy: "test",
		})
		if err != nil {
			t.Fatalf("CreateFeature(%s) error = %v", slug, err)
		}
	}

	// Attach cache after creation — "feature" is cold.
	attachCache(t, svc)

	results, err := svc.List("feature")
	if err != nil {
		t.Fatalf("List() with cold type error = %v", err)
	}
	if len(results) != 2 {
		t.Errorf("List() returned %d results, want 2", len(results))
	}
}

// TestList_CorruptFieldsJSON_ReturnsError verifies that List() returns an error
// when a cache row has invalid (non-parseable) fields_json, rather than silently
// omitting the row or returning partial results.
func TestList_CorruptFieldsJSON_ReturnsError(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")
	attachCache(t, svc)

	planID := "P1-parent"
	writeTestPlan(t, svc, planID)

	created, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "corrupt json feature",
		Parent:    planID,
		Summary:   "Corrupt JSON test",
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}

	// Overwrite the cache row with invalid JSON in fields_json.
	if err := svc.cache.Upsert(cache.EntityRow{
		EntityType: "feature",
		ID:         created.ID,
		Slug:       created.Slug,
		FilePath:   created.Path,
		FieldsJSON: "not-json",
	}); err != nil {
		t.Fatalf("cache.Upsert() corrupt row error = %v", err)
	}

	_, err = svc.List("feature")
	if err == nil {
		t.Fatal("List() with corrupt fields_json returned nil error, want error")
	}
}

// TestList_ListByTypeError_FallsBack verifies that when ListByType returns an
// error (e.g., the DB is closed), List() falls back to the filesystem path and
// still returns the correct results.
func TestList_ListByTypeError_FallsBack(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")
	c := attachCache(t, svc)

	planID := "P1-parent"
	writeTestPlan(t, svc, planID)

	created, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "list fallback feature",
		Parent:    planID,
		Summary:   "ListByType error fallback",
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}

	// Close the underlying DB — IsWarm still returns true (in-memory map),
	// but ListByType will fail, triggering the filesystem fallback.
	_ = c.Close()

	results, err := svc.List("feature")
	if err != nil {
		t.Fatalf("List() after ListByType error = %v, want filesystem fallback", err)
	}
	if len(results) != 1 {
		t.Fatalf("List() returned %d results, want 1", len(results))
	}
	if results[0].ID != created.ID {
		t.Errorf("List() result ID = %q, want %q", results[0].ID, created.ID)
	}
}

// ---- IsWarm integration test ------------------------------------------------

// TestIsWarm_SetByRebuildCache verifies that calling EntityService.RebuildCache()
// marks the relevant entity types as warm in the in-process cache.
func TestIsWarm_SetByRebuildCache(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")

	planID := "P1-parent"
	writeTestPlan(t, svc, planID)

	// Create a feature and a task without any cache attached.
	feat, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "rebuild test",
		Parent:    planID,
		Summary:   "IsWarm via RebuildCache",
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}
	_, err = svc.CreateTask(CreateTaskInput{
		Name:          "test",
		ParentFeature: feat.ID,
		Slug:          "rebuild task",
		Summary:       "Task for rebuild test",
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	// Now attach cache and rebuild.
	attachCache(t, svc)

	n, err := svc.RebuildCache()
	if err != nil {
		t.Fatalf("RebuildCache() error = %v", err)
	}
	if n == 0 {
		t.Error("RebuildCache() returned 0, want > 0")
	}
	if !svc.cache.IsWarm("task") {
		t.Error("cache.IsWarm(\"task\") = false, want true after RebuildCache with tasks")
	}
	if !svc.cache.IsWarm("feature") {
		t.Error("cache.IsWarm(\"feature\") = false, want true after RebuildCache with features")
	}
}

// ---- Result equivalence tests -----------------------------------------------

// sortListResults sorts by ID so cache and filesystem results can be compared
// regardless of return order.
func sortListResults(results []ListResult) {
	sort.Slice(results, func(i, j int) bool {
		return results[i].ID < results[j].ID
	})
}

// TestList_CacheAndFilesystem_ResultEquivalent verifies that List() returns the
// same IDs, slugs, types, and paths whether the cache path or filesystem path
// is used.
func TestList_CacheAndFilesystem_ResultEquivalent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")
	attachCache(t, svc)

	planID := "P1-parent"
	writeTestPlan(t, svc, planID)

	for _, slug := range []string{"equiv-one", "equiv-two", "equiv-three"} {
		_, err := svc.CreateFeature(CreateFeatureInput{
			Name:      "test",
			Slug:      slug,
			Parent:    planID,
			Summary:   "Equivalence test: " + slug,
			CreatedBy: "test",
		})
		if err != nil {
			t.Fatalf("CreateFeature(%s) error = %v", slug, err)
		}
	}

	// List via warm cache.
	cacheResults, err := svc.List("feature")
	if err != nil {
		t.Fatalf("List() via cache error = %v", err)
	}

	// Bypass cache by setting it to nil.
	svc.cache = nil
	fsResults, err := svc.List("feature")
	if err != nil {
		t.Fatalf("List() via filesystem error = %v", err)
	}

	if len(cacheResults) != len(fsResults) {
		t.Fatalf("result count mismatch: cache=%d, filesystem=%d", len(cacheResults), len(fsResults))
	}

	sortListResults(cacheResults)
	sortListResults(fsResults)

	for i := range cacheResults {
		cr, fr := cacheResults[i], fsResults[i]
		if cr.ID != fr.ID {
			t.Errorf("result[%d].ID: cache=%q, fs=%q", i, cr.ID, fr.ID)
		}
		if cr.Slug != fr.Slug {
			t.Errorf("result[%d].Slug: cache=%q, fs=%q", i, cr.Slug, fr.Slug)
		}
		if cr.Type != fr.Type {
			t.Errorf("result[%d].Type: cache=%q, fs=%q", i, cr.Type, fr.Type)
		}
		if cr.Path != fr.Path {
			t.Errorf("result[%d].Path: cache=%q, fs=%q", i, cr.Path, fr.Path)
		}
		// Compare a representative sample of known-string State fields.
		for _, key := range []string{"id", "slug", "status", "summary"} {
			if cr.State[key] != fr.State[key] {
				t.Errorf("result[%d].State[%q]: cache=%v, fs=%v", i, key, cr.State[key], fr.State[key])
			}
		}
	}
}

// TestGet_CacheAndFilesystem_ResultEquivalent verifies that Get() returns the
// same result whether the cache fast path or the filesystem path is used.
func TestGet_CacheAndFilesystem_ResultEquivalent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-03-19T12:00:00Z")
	attachCache(t, svc)

	planID := "P1-parent"
	writeTestPlan(t, svc, planID)

	created, err := svc.CreateFeature(CreateFeatureInput{
		Name:      "test",
		Slug:      "get equivalence",
		Parent:    planID,
		Summary:   "Get equivalence test",
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature() error = %v", err)
	}

	// Get via warm cache (empty slug triggers fast path).
	cacheResult, err := svc.Get("feature", created.ID, "")
	if err != nil {
		t.Fatalf("Get() via cache error = %v", err)
	}

	// Get via filesystem (slug provided, no prefix resolution needed — no cache used).
	svc.cache = nil
	fsResult, err := svc.Get("feature", created.ID, created.Slug)
	if err != nil {
		t.Fatalf("Get() via filesystem error = %v", err)
	}

	if cacheResult.ID != fsResult.ID {
		t.Errorf("ID: cache=%q, fs=%q", cacheResult.ID, fsResult.ID)
	}
	if cacheResult.Slug != fsResult.Slug {
		t.Errorf("Slug: cache=%q, fs=%q", cacheResult.Slug, fsResult.Slug)
	}
	if cacheResult.Type != fsResult.Type {
		t.Errorf("Type: cache=%q, fs=%q", cacheResult.Type, fsResult.Type)
	}
	if cacheResult.Path != fsResult.Path {
		t.Errorf("Path: cache=%q, fs=%q", cacheResult.Path, fsResult.Path)
	}
	if !reflect.DeepEqual(cacheResult.State, fsResult.State) {
		t.Errorf("State mismatch:\ncache: %#v\nfs:    %#v", cacheResult.State, fsResult.State)
	}
}
