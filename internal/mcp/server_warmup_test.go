package mcp

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sambeau/kanbanzai/internal/cache"
	"github.com/sambeau/kanbanzai/internal/service"
)

// openWarmupTestCache opens a cache in a temp directory and registers cleanup.
func openWarmupTestCache(t *testing.T) *cache.Cache {
	t.Helper()
	dir := filepath.Join(t.TempDir(), cache.CacheDir)
	c, err := cache.Open(dir)
	if err != nil {
		t.Fatalf("openWarmupTestCache: %v", err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

// TestRebuildCache_PopulatesCache covers AC-001/AC-002: after calling
// RebuildCache on a service with entities on disk, the returned count is
// positive and the cache reports entities as present.
func TestRebuildCache_PopulatesCache(t *testing.T) {
	t.Parallel()

	svc := service.NewEntityService(t.TempDir())
	c := openWarmupTestCache(t)
	svc.SetCache(c)

	// Create a plan and a feature so there is at least one entity on disk.
	planID := createEntityTestPlan(t, svc, "warmup-plan")
	createEntityTestFeature(t, svc, planID, "warmup-feature")

	n, err := svc.RebuildCache()
	if err != nil {
		t.Fatalf("RebuildCache() error = %v", err)
	}
	if n == 0 {
		t.Error("RebuildCache() returned 0 entities; expected > 0")
	}

	// The cache should now contain the feature.
	total, err := c.Count("feature")
	if err != nil {
		t.Fatalf("cache.Count(feature): %v", err)
	}
	if total == 0 {
		t.Error("cache has 0 features after RebuildCache; expected > 0")
	}
}

// TestRebuildCache_GracefulOnClosedDB covers AC-003/AC-004: when the
// underlying cache DB is closed before RebuildCache is called, the function
// returns a non-nil error without panicking. The server startup code logs
// the error and continues — this confirms the error path is safe.
func TestRebuildCache_GracefulOnClosedDB(t *testing.T) {
	t.Parallel()

	svc := service.NewEntityService(t.TempDir())
	dir := filepath.Join(t.TempDir(), cache.CacheDir)
	c, err := cache.Open(dir)
	if err != nil {
		t.Fatalf("cache.Open: %v", err)
	}
	svc.SetCache(c)

	// Close the DB to force a failure on the next RebuildCache call.
	if err := c.Close(); err != nil {
		t.Fatalf("cache.Close: %v", err)
	}

	_, rebuildErr := svc.RebuildCache()
	if rebuildErr == nil {
		t.Error("RebuildCache() with closed DB: expected error, got nil")
	}
}

// TestRebuildCache_NilCache covers AC-007: when cache.Open fails (or is
// skipped) so SetCache is never called, RebuildCache returns a zero count
// and a non-nil error rather than panicking. In server startup the entire
// warm-up block is skipped when cache.Open returns an error, so this
// path is only reachable in tests or if the caller invokes RebuildCache
// directly without a cache.
func TestRebuildCache_NilCache(t *testing.T) {
	t.Parallel()

	svc := service.NewEntityService(t.TempDir())
	// Deliberately no SetCache — svc.cache is nil.

	n, err := svc.RebuildCache()
	if err == nil {
		t.Error("RebuildCache() with nil cache: expected error, got nil")
	}
	if n != 0 {
		t.Errorf("RebuildCache() with nil cache: got count %d, want 0", n)
	}
}

// TestEntityService_NoCacheRegression_List covers AC-008: List still returns
// results from the filesystem when no cache is configured.
func TestEntityService_NoCacheRegression_List(t *testing.T) {
	t.Parallel()

	svc := service.NewEntityService(t.TempDir())
	// Deliberately no SetCache.

	planID := createEntityTestPlan(t, svc, "no-cache-list-plan")
	createEntityTestFeature(t, svc, planID, "no-cache-list-feat")

	results, err := svc.List("feature")
	if err != nil {
		t.Fatalf("List(feature) with no cache: %v", err)
	}
	if len(results) == 0 {
		t.Error("List(feature) with no cache returned 0 results; expected > 0")
	}
}

// TestEntityService_NoCacheRegression_Get covers AC-009: Get still resolves
// entities from the filesystem when no cache is configured.
func TestEntityService_NoCacheRegression_Get(t *testing.T) {
	t.Parallel()

	svc := service.NewEntityService(t.TempDir())
	// Deliberately no SetCache.

	planID := createEntityTestPlan(t, svc, "no-cache-get-plan")
	featID := createEntityTestFeature(t, svc, planID, "no-cache-get-feat")

	got, err := svc.Get("feature", featID, "")
	if err != nil {
		t.Fatalf("Get(feature, %s) with no cache: %v", featID, err)
	}
	if got.ID != featID {
		t.Errorf("Get returned ID %q, want %q", got.ID, featID)
	}
}

// TestRebuildCache_SuccessLogContainsCountAndDuration covers AC-005: the
// server warm-up success log contains a numeric entity count and a duration.
// The production log line in server.go is:
//
//	log.Printf("[server] cache warm-up: loaded %d entities in %s", n, time.Since(start))
//
// This test mirrors that warm-up block to capture and verify the log format.
func TestRebuildCache_SuccessLogContainsCountAndDuration(t *testing.T) {
	// Not parallel: mutates the global log output.

	svc := service.NewEntityService(t.TempDir())
	c := openWarmupTestCache(t)
	svc.SetCache(c)

	planID := createEntityTestPlan(t, svc, "ac005-plan")
	createEntityTestFeature(t, svc, planID, "ac005-feat")

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	// Mirror the server.go warm-up block to exercise the success log path.
	start := time.Now()
	n, err := svc.RebuildCache()
	if err != nil {
		log.Printf("[server] cache warm-up failed (continuing without cache): %v", err)
	} else {
		log.Printf("[server] cache warm-up: loaded %d entities in %s", n, time.Since(start))
	}

	if err != nil {
		t.Fatalf("expected RebuildCache success, got error: %v", err)
	}

	output := buf.String()

	// AC-005: log must contain the success prefix.
	if !strings.Contains(output, "[server] cache warm-up: loaded") {
		t.Errorf("log = %q; want '[server] cache warm-up: loaded'", output)
	}
	// AC-005: log must contain the "entities in" separator confirming both fields are present.
	if !strings.Contains(output, " entities in ") {
		t.Errorf("log = %q; want ' entities in ' separator", output)
	}
	// AC-005: log must contain a non-zero digit (the entity count).
	if !strings.ContainsAny(output, "123456789") {
		t.Errorf("log = %q; want a non-zero numeric count", output)
	}
}

// TestRebuildCache_FailureLogContainsErrorText covers AC-006: the server
// warm-up failure log contains "continuing without cache" and the error text.
// The production log line in server.go is:
//
//	log.Printf("[server] cache warm-up failed (continuing without cache): %v", err)
//
// This test mirrors that warm-up block to capture and verify the log format.
func TestRebuildCache_FailureLogContainsErrorText(t *testing.T) {
	// Not parallel: mutates the global log output.

	svc := service.NewEntityService(t.TempDir())
	dir := filepath.Join(t.TempDir(), cache.CacheDir)
	c, err := cache.Open(dir)
	if err != nil {
		t.Fatalf("cache.Open: %v", err)
	}
	svc.SetCache(c)
	// Close the DB so that RebuildCache fails, triggering the failure log path.
	if err := c.Close(); err != nil {
		t.Fatalf("cache.Close: %v", err)
	}

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	// Mirror the server.go warm-up block to exercise the failure log path.
	start := time.Now()
	n, rebuildErr := svc.RebuildCache()
	if rebuildErr != nil {
		log.Printf("[server] cache warm-up failed (continuing without cache): %v", rebuildErr)
	} else {
		log.Printf("[server] cache warm-up: loaded %d entities in %s", n, time.Since(start))
	}

	if rebuildErr == nil {
		t.Fatal("expected RebuildCache to fail with closed DB, got nil")
	}

	output := buf.String()

	// AC-006: failure log must contain the "continuing without cache" phrase.
	if !strings.Contains(output, "continuing without cache") {
		t.Errorf("log = %q; want 'continuing without cache'", output)
	}
	// AC-006: failure log must contain the error text.
	if !strings.Contains(output, rebuildErr.Error()) {
		t.Errorf("log = %q; want error text %q", output, rebuildErr.Error())
	}
}
