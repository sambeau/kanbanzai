package coordination

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

func TestNew_InvalidURL(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// A syntactically valid URL pointing to a non-routable address that
	// should time out quickly.
	db, err := New(ctx, "postgres://192.0.2.1:5432/testdb")
	if err == nil {
		if db != nil {
			db.Close()
		}
		t.Fatal("expected error for unreachable host, got nil")
	}
}

func TestNew_ValidURL(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("New(ctx, databaseURL): %v", err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		t.Fatalf("Ping(): %v", err)
	}
}

// dbTest is a helper that opens a DB or skips the test.
func dbTest(t *testing.T) *DB {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestMigrate_CreatesTables(t *testing.T) {
	db := dbTest(t)
	ctx := context.Background()

	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// Verify all three tables exist.
	tables := []string{"counters", "batch_feature_seqs", "allocations"}
	for _, table := range tables {
		var exists bool
		err := db.pool.QueryRow(ctx,
			`SELECT EXISTS (
				SELECT FROM information_schema.tables
				WHERE table_schema = 'public'
				  AND table_name   = $1
			)`,
			table,
		).Scan(&exists)
		if err != nil {
			t.Fatalf("checking table %q: %v", table, err)
		}
		if !exists {
			t.Errorf("table %q does not exist after Migrate", table)
		}
	}
}

func TestMigrate_Idempotent(t *testing.T) {
	db := dbTest(t)
	ctx := context.Background()

	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("Migrate (first): %v", err)
	}
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("Migrate (second): %v", err)
	}
}

func TestAllocateID_Format(t *testing.T) {
	db := dbTest(t)
	ctx := context.Background()

	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	id, err := db.AllocateID(ctx, "test-project", "feature", "FEAT-", "my-feature")
	if err != nil {
		t.Fatalf("AllocateID: %v", err)
	}
	if id != "FEAT-1-my-feature" {
		t.Errorf("expected FEAT-1-my-feature, got %q", id)
	}
}

func TestAllocateID_Idempotent(t *testing.T) {
	db := dbTest(t)
	ctx := context.Background()

	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	first, err := db.AllocateID(ctx, "test-project-2", "task", "TASK-", "my-task")
	if err != nil {
		t.Fatalf("AllocateID (first): %v", err)
	}
	second, err := db.AllocateID(ctx, "test-project-2", "task", "TASK-", "my-task")
	if err != nil {
		t.Fatalf("AllocateID (second): %v", err)
	}
	if first != second {
		t.Errorf("idempotent violation: first=%q second=%q", first, second)
	}

	// Verify counter stayed at 2 (idempotent call did not increment).
	var counter int
	if err := db.pool.QueryRow(ctx,
		`SELECT next_value FROM counters WHERE project_id = $1 AND entity_type = $2`,
		"test-project-2", "task",
	).Scan(&counter); err != nil {
		t.Fatalf("reading counter: %v", err)
	}
	if counter != 2 {
		t.Errorf("expected counter=2 (next_value after first alloc), got %d", counter)
	}
}

func TestAllocateID_IncrementsPerEntityType(t *testing.T) {
	db := dbTest(t)
	ctx := context.Background()

	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	a, err := db.AllocateID(ctx, "test-project-3", "feature", "FEAT-", "alpha")
	if err != nil {
		t.Fatalf("AllocateID alpha: %v", err)
	}
	b, err := db.AllocateID(ctx, "test-project-3", "feature", "FEAT-", "beta")
	if err != nil {
		t.Fatalf("AllocateID beta: %v", err)
	}
	if a != "FEAT-1-alpha" {
		t.Errorf("alpha: expected FEAT-1-alpha, got %q", a)
	}
	if b != "FEAT-2-beta" {
		t.Errorf("beta: expected FEAT-2-beta, got %q", b)
	}
}

func TestAllocateID_DifferentEntityTypes(t *testing.T) {
	db := dbTest(t)
	ctx := context.Background()

	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// Allocate for batch_B — should start at 1.
	batch1, err := db.AllocateID(ctx, t.Name(), "batch_B", "B", "first-batch")
	if err != nil {
		t.Fatalf("AllocateID batch_B: %v", err)
	}
	if batch1 != "B1-first-batch" {
		t.Errorf("batch_B: expected B1-first-batch, got %q", batch1)
	}

	// Allocate for plan_P — should start at 1 independently.
	plan1, err := db.AllocateID(ctx, t.Name(), "plan_P", "P", "first-plan")
	if err != nil {
		t.Fatalf("AllocateID plan_P: %v", err)
	}
	if plan1 != "P1-first-plan" {
		t.Errorf("plan_P: expected P1-first-plan, got %q", plan1)
	}

	// Second batch_B allocation should be B2.
	batch2, err := db.AllocateID(ctx, t.Name(), "batch_B", "B", "second-batch")
	if err != nil {
		t.Fatalf("AllocateID batch_B second: %v", err)
	}
	if batch2 != "B2-second-batch" {
		t.Errorf("batch_B second: expected B2-second-batch, got %q", batch2)
	}
}

func TestAllocateFeatureSeq(t *testing.T) {
	db := dbTest(t)
	ctx := context.Background()

	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	s1, err := db.AllocateFeatureSeq(ctx, t.Name()+"-b6ffcc4d", "batch-1")
	if err != nil {
		t.Fatalf("AllocateFeatureSeq (1): %v", err)
	}
	if s1 != 1 {
		t.Errorf("first seq: expected 1, got %d", s1)
	}

	s2, err := db.AllocateFeatureSeq(ctx, t.Name()+"-b6ffcc4d", "batch-1")
	if err != nil {
		t.Fatalf("AllocateFeatureSeq (2): %v", err)
	}
	if s2 != 2 {
		t.Errorf("second seq: expected 2, got %d", s2)
	}

	s3, err := db.AllocateFeatureSeq(ctx, t.Name()+"-b6ffcc4d", "batch-1")
	if err != nil {
		t.Fatalf("AllocateFeatureSeq (3): %v", err)
	}
	if s3 != 3 {
		t.Errorf("third seq: expected 3, got %d", s3)
	}
}

func TestAllocateFeatureSeq_IndependentBatches(t *testing.T) {
	db := dbTest(t)
	ctx := context.Background()

	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	a1, _ := db.AllocateFeatureSeq(ctx, t.Name()+"-5cfad766", "batch-a")
	a2, _ := db.AllocateFeatureSeq(ctx, t.Name()+"-5cfad766", "batch-a")
	b1, _ := db.AllocateFeatureSeq(ctx, t.Name()+"-5cfad766", "batch-b")

	if a1 != 1 {
		t.Errorf("batch-a first: expected 1, got %d", a1)
	}
	if a2 != 2 {
		t.Errorf("batch-a second: expected 2, got %d", a2)
	}
	if b1 != 1 {
		t.Errorf("batch-b first: expected 1, got %d", b1)
	}
}

func TestAllocateID_FunctionExists(t *testing.T) {
	db := dbTest(t)
	ctx := context.Background()

	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	var exists bool
	err := db.pool.QueryRow(ctx,
		`SELECT EXISTS (
			SELECT FROM pg_proc
			WHERE proname = 'allocate_id'
			  AND pronamespace = (SELECT oid FROM pg_namespace WHERE nspname = 'public')
		)`,
	).Scan(&exists)
	if err != nil {
		t.Fatalf("checking allocate_id function: %v", err)
	}
	if !exists {
		t.Error("allocate_id function does not exist after Migrate")
	}
}

func TestAllocateID_BugFormat(t *testing.T) {
	db := dbTest(t)
	ctx := context.Background()

	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	id, err := db.AllocateID(ctx, "bugfmt-3e6e4a7e", "bug", "BUG-", "bugfmt-91910ec6")
	if err != nil {
		t.Fatalf("AllocateID: %v", err)
	}
	if id != "BUG-1-bugfmt-91910ec6" {
		t.Errorf("expected BUG-1-bugfmt-91910ec6, got %q", id)
	}
}

func TestNew_UnreachableHost(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := New(ctx, "postgres://user:pass@192.0.2.1:5432/db?connect_timeout=2")
	if err == nil {
		if db != nil {
			db.Close()
		}
		t.Fatal("expected error for unreachable host, got nil")
	}
}

func TestAllocateID_Concurrent(t *testing.T) {
	db := dbTest(t)
	ctx := context.Background()

	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	const goroutines = 20
	results := make([]string, goroutines)
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			slug := fmt.Sprintf("concurrent-%d", idx)
			id, err := db.AllocateID(ctx, "test-project-5", "feature", "FEAT-", slug)
			if err != nil {
				t.Errorf("AllocateID goroutine %d: %v", idx, err)
				return
			}
			results[idx] = id
		}(i)
	}
	wg.Wait()

	// Verify all IDs are unique.
	seen := make(map[string]bool)
	for _, id := range results {
		if id == "" {
			t.Error("empty ID returned")
			continue
		}
		if seen[id] {
			t.Errorf("duplicate ID: %s", id)
		}
		seen[id] = true
	}
}

func TestSeedCounters_SeedsAboveExisting(t *testing.T) {
	db := dbTest(t)
	ctx := context.Background()

	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// First allocation without seeding should start at 1.
	id1, err := db.AllocateID(ctx, t.Name()+"-eddd9de7", "bug", "BUG-", "first")
	if err != nil {
		t.Fatalf("AllocateID: %v", err)
	}
	if id1 != "BUG-1-first" {
		t.Errorf("expected BUG-1-first, got %q", id1)
	}

	// Seed counters for "bug" to start at 50.
	if err := db.SeedCounters(ctx, t.Name()+"-eddd9de7", map[string]int{"bug": 50}); err != nil {
		t.Fatalf("SeedCounters: %v", err)
	}

	// Next allocation should be at least 50 (seed value).
	// Since counter was at 2 (from first alloc), seed bumps it to 50.
	id2, err := db.AllocateID(ctx, t.Name()+"-eddd9de7", "bug", "BUG-", "second")
	if err != nil {
		t.Fatalf("AllocateID after seed: %v", err)
	}
	if id2 != "BUG-50-second" {
		t.Errorf("expected BUG-50-second after seeding to 50, got %q", id2)
	}

	// Third allocation should continue from 51.
	id3, err := db.AllocateID(ctx, t.Name()+"-eddd9de7", "bug", "BUG-", "third")
	if err != nil {
		t.Fatalf("AllocateID third: %v", err)
	}
	if id3 != "BUG-51-third" {
		t.Errorf("expected BUG-51-third, got %q", id3)
	}
}

func TestSeedCounters_NoopWhenBelowCurrent(t *testing.T) {
	db := dbTest(t)
	ctx := context.Background()

	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// Allocate several IDs to advance counter past 5.
	for i := 0; i < 10; i++ {
		_, err := db.AllocateID(ctx, t.Name()+"-bbc5c8e5", "bug", "BUG-", fmt.Sprintf("pre%d", i))
		if err != nil {
			t.Fatalf("AllocateID pre%d: %v", i, err)
		}
	}

	// Seed with a value lower than current counter — should be a no-op.
	if err := db.SeedCounters(ctx, t.Name()+"-bbc5c8e5", map[string]int{"bug": 3}); err != nil {
		t.Fatalf("SeedCounters: %v", err)
	}

	// Next allocation should be 11 (continuing from 10), not 3.
	id, err := db.AllocateID(ctx, t.Name()+"-bbc5c8e5", "bug", "BUG-", "after-seed")
	if err != nil {
		t.Fatalf("AllocateID after low seed: %v", err)
	}
	if id != "BUG-11-after-seed" {
		t.Errorf("expected BUG-11-after-seed (seed to 3 should be no-op), got %q", id)
	}
}

func TestSeedCounters_MultipleEntityTypes(t *testing.T) {
	db := dbTest(t)
	ctx := context.Background()

	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// Seed both bug and plan counters.
	if err := db.SeedCounters(ctx, t.Name()+"-1ebf0e92", map[string]int{
		"bug":    10,
		"plan_P": 52,
	}); err != nil {
		t.Fatalf("SeedCounters: %v", err)
	}

	// Bug should start at 10.
	bugID, err := db.AllocateID(ctx, t.Name()+"-1ebf0e92", "bug", "BUG-", "test-bug")
	if err != nil {
		t.Fatalf("AllocateID bug: %v", err)
	}
	if bugID != "BUG-10-test-bug" {
		t.Errorf("bug: expected BUG-10-test-bug, got %q", bugID)
	}

	// Plan should start at 52.
	planID, err := db.AllocateID(ctx, t.Name()+"-1ebf0e92", "plan_P", "P", "test-plan")
	if err != nil {
		t.Fatalf("AllocateID plan: %v", err)
	}
	if planID != "P52-test-plan" {
		t.Errorf("plan: expected P52-test-plan, got %q", planID)
	}

	// A different entity type not in seed should still start at 1.
	featID, err := db.AllocateID(ctx, t.Name()+"-1ebf0e92", "feature", "FEAT-", "test-feat")
	if err != nil {
		t.Fatalf("AllocateID feature: %v", err)
	}
	if featID != "FEAT-1-test-feat" {
		t.Errorf("feature: expected FEAT-1-test-feat, got %q", featID)
	}
}
