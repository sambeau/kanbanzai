package service

import (
	"testing"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// FR-003: Get increments recent_use_count and sets last_accessed_at
// ─────────────────────────────────────────────────────────────────────────────

func TestKnowledgeService_Get_IncrementsRecentUseCount(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	rec, _, err := svc.Contribute(ContributeInput{
		Topic:   "get-access-test",
		Content: "Bloom filters provide probabilistic membership testing with configurable false positive rates.",
		Scope:   "project",
	})
	if err != nil {
		t.Fatalf("Contribute: %v", err)
	}

	// Call Get — should trigger background increment
	_, err = svc.Get(rec.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	// Wait for background goroutine to finish
	svc.Close()

	// Reload and verify
	loaded, err := svc.store.Load(rec.ID)
	if err != nil {
		t.Fatalf("Load after Get: %v", err)
	}

	count := knowledgeFieldInt(loaded.Fields, "recent_use_count")
	if count != 1 {
		t.Errorf("recent_use_count = %d, want 1", count)
	}
	lastAccessed, _ := loaded.Fields["last_accessed_at"].(string)
	if lastAccessed == "" {
		t.Error("last_accessed_at should be set after Get")
	}
}

func TestKnowledgeService_Get_IncrementsMultipleTimes(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	rec, _, err := svc.Contribute(ContributeInput{
		Topic:   "get-multi-access",
		Content: "Vector clocks track causality in distributed systems without requiring synchronized clocks.",
		Scope:   "project",
	})
	if err != nil {
		t.Fatalf("Contribute: %v", err)
	}

	for i := 0; i < 3; i++ {
		// Close after each to ensure sequential updates (goroutines don't race)
		_, err = svc.Get(rec.ID)
		if err != nil {
			t.Fatalf("Get iteration %d: %v", i, err)
		}
		svc.Close()
		// Re-create service to reset the wait group state
		// (Close drains the WaitGroup but doesn't destroy it)
	}

	loaded, err := svc.store.Load(rec.ID)
	if err != nil {
		t.Fatalf("Load after Gets: %v", err)
	}
	count := knowledgeFieldInt(loaded.Fields, "recent_use_count")
	if count != 3 {
		t.Errorf("recent_use_count = %d, want 3", count)
	}
}

func TestKnowledgeService_Get_ErrorDoesNotPreventReturn(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	// Get a non-existent entry — should return error but not panic
	_, err := svc.Get("KE-nonexistent")
	if err == nil {
		t.Fatal("Get(nonexistent) should return error")
	}
	// No panic, and no goroutine is spawned (Get returns before touching)
}

// ─────────────────────────────────────────────────────────────────────────────
// FR-004: List increments recent_use_count for every returned entry
// ─────────────────────────────────────────────────────────────────────────────

func TestKnowledgeService_List_IncrementsAllReturnedEntries(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	ids := make([]string, 3)
	listContents := []string{
		"Immutable data structures eliminate entire classes of concurrency bugs in practice.",
		"Backpressure mechanisms prevent memory exhaustion in streaming data pipelines.",
		"Blue-green deployments enable zero-downtime releases with instant rollback capability.",
	}
	for i, topic := range []string{"list-access-a", "list-access-b", "list-access-c"} {
		rec, _, err := svc.Contribute(ContributeInput{
			Topic:   topic,
			Content: listContents[i],
			Scope:   "project",
		})
		if err != nil {
			t.Fatalf("Contribute %s: %v", topic, err)
		}
		ids[i] = rec.ID
	}

	_, err := svc.List(KnowledgeFilters{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	// Wait for all background goroutines
	svc.Close()

	for _, id := range ids {
		loaded, err := svc.store.Load(id)
		if err != nil {
			t.Fatalf("Load %s: %v", id, err)
		}
		count := knowledgeFieldInt(loaded.Fields, "recent_use_count")
		if count != 1 {
			t.Errorf("entry %s recent_use_count = %d, want 1", id, count)
		}
	}
}

func TestKnowledgeService_List_EmptyResultNoGoroutine(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	// List on empty store should succeed without spawning goroutines
	records, err := svc.List(KnowledgeFilters{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected empty result, got %d", len(records))
	}
	// No goroutines spawned, Close() should return immediately
	svc.Close()
}

// ─────────────────────────────────────────────────────────────────────────────
// FR-005: 30-day rolling window (lazy decay)
// ─────────────────────────────────────────────────────────────────────────────

func TestKnowledgeService_AccessDecay_Within30Days(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	rec, _, err := svc.Contribute(ContributeInput{
		Topic:   "decay-within-window",
		Content: "Exponential backoff with jitter reduces thundering herd problems in distributed retry logic.",
		Scope:   "project",
	})
	if err != nil {
		t.Fatalf("Contribute: %v", err)
	}

	// Simulate a previous access 10 days ago (within 30-day window)
	tenDaysAgo := svc.now().Add(-10 * 24 * time.Hour)
	rec.Fields["recent_use_count"] = 5
	rec.Fields["last_accessed_at"] = tenDaysAgo.Format(time.RFC3339)
	if _, err := svc.store.Write(rec); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Access now — count should increment (10 days is within 30-day window)
	_, err = svc.Get(rec.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	svc.Close()

	loaded, err := svc.store.Load(rec.ID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	count := knowledgeFieldInt(loaded.Fields, "recent_use_count")
	if count != 6 {
		t.Errorf("recent_use_count = %d, want 6 (5 + 1 within window)", count)
	}
}

func TestKnowledgeService_AccessDecay_OlderThan30Days(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	rec, _, err := svc.Contribute(ContributeInput{
		Topic:   "decay-outside-window",
		Content: "Write-ahead logging ensures durability in database systems before committing transactions.",
		Scope:   "project",
	})
	if err != nil {
		t.Fatalf("Contribute: %v", err)
	}

	// Simulate a previous access 45 days ago (outside 30-day window)
	oldAccess := svc.now().Add(-45 * 24 * time.Hour)
	rec.Fields["recent_use_count"] = 10
	rec.Fields["last_accessed_at"] = oldAccess.Format(time.RFC3339)
	if _, err := svc.store.Write(rec); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Access now — count should reset to 1 (old count decayed)
	_, err = svc.Get(rec.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	svc.Close()

	loaded, err := svc.store.Load(rec.ID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	count := knowledgeFieldInt(loaded.Fields, "recent_use_count")
	if count != 1 {
		t.Errorf("recent_use_count = %d, want 1 (decayed to 0 then +1)", count)
	}
}

func TestKnowledgeService_AccessDecay_NoLastAccessed(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	rec, _, err := svc.Contribute(ContributeInput{
		Topic:   "decay-no-last-accessed",
		Content: "Saga pattern manages distributed transactions through compensating actions in microservices.",
		Scope:   "project",
	})
	if err != nil {
		t.Fatalf("Contribute: %v", err)
	}

	// Access with no last_accessed_at set — should set count to 1
	_, err = svc.Get(rec.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	svc.Close()

	loaded, err := svc.store.Load(rec.ID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	count := knowledgeFieldInt(loaded.Fields, "recent_use_count")
	if count != 1 {
		t.Errorf("recent_use_count = %d, want 1", count)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// FR-006: Sort by recent_use_count descending
// ─────────────────────────────────────────────────────────────────────────────

func TestKnowledgeService_List_SortRecent(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	// Create three entries with different recent_use_count values
	topicsAndCounts := []struct {
		topic   string
		count   int
		content string
	}{
		{"sort-recent-low", 1, "Dependency injection decouples components and improves testability in Go."},
		{"sort-recent-high", 10, "Merkle trees enable efficient verification of large data structures cryptographically."},
		{"sort-recent-mid", 5, "Circuit breakers prevent cascading failures in microservice architectures effectively."},
	}

	for _, tc := range topicsAndCounts {
		rec, _, err := svc.Contribute(ContributeInput{
			Topic:   tc.topic,
			Content: tc.content,
			Scope:   "project",
		})
		if err != nil {
			t.Fatalf("Contribute %s: %v", tc.topic, err)
		}
		rec.Fields["recent_use_count"] = tc.count
		if _, err := svc.store.Write(rec); err != nil {
			t.Fatalf("Write %s: %v", tc.topic, err)
		}
	}

	records, err := svc.List(KnowledgeFilters{Sort: "recent"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	svc.Close()

	if len(records) != 3 {
		t.Fatalf("List returned %d records, want 3", len(records))
	}

	// Verify descending order
	for i := 1; i < len(records); i++ {
		prev := knowledgeFieldInt(records[i-1].Fields, "recent_use_count")
		curr := knowledgeFieldInt(records[i].Fields, "recent_use_count")
		if prev < curr {
			t.Errorf("records[%d].recent_use_count=%d < records[%d].recent_use_count=%d, want descending",
				i-1, prev, i, curr)
		}
	}

	// Verify the highest-count entry is first
	firstCount := knowledgeFieldInt(records[0].Fields, "recent_use_count")
	if firstCount != 10 {
		t.Errorf("first record recent_use_count = %d, want 10 (highest)", firstCount)
	}
}

func TestKnowledgeService_List_SortRecent_NoSortByDefault(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	uniqueContents := map[string]string{
		"no-sort-a": "The quick brown fox jumps over the lazy dog near the river.",
		"no-sort-b": "Structured logging improves observability in distributed systems significantly.",
	}
	for _, topic := range []string{"no-sort-a", "no-sort-b"} {
		_, _, err := svc.Contribute(ContributeInput{
			Topic:   topic,
			Content: uniqueContents[topic],
			Scope:   "project",
		})
		if err != nil {
			t.Fatalf("Contribute %s: %v", topic, err)
		}
	}

	// Without Sort: "recent", the result should still be returned (any order)
	records, err := svc.List(KnowledgeFilters{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	svc.Close()

	if len(records) != 2 {
		t.Errorf("List returned %d records, want 2", len(records))
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// FR-001, FR-002: Fields default to zero when absent
// ─────────────────────────────────────────────────────────────────────────────

func TestKnowledgeService_NewEntry_HasZeroRecentUseCount(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	rec, _, err := svc.Contribute(ContributeInput{
		Topic:   "new-entry-zero-count",
		Content: "Raft consensus algorithm ensures linearizable reads and writes across cluster replicas.",
		Scope:   "project",
	})
	if err != nil {
		t.Fatalf("Contribute: %v", err)
	}

	count := knowledgeFieldInt(rec.Fields, "recent_use_count")
	if count != 0 {
		t.Errorf("new entry recent_use_count = %d, want 0", count)
	}
	if _, hasLastAccessed := rec.Fields["last_accessed_at"]; hasLastAccessed {
		t.Error("new entry should not have last_accessed_at set")
	}
}

func TestKnowledgeService_LoadLegacyEntry_TreatsAbsentFieldsAsZero(t *testing.T) {
	t.Parallel()
	svc := newTestKnowledgeService(t)

	// Write an entry without recent_use_count / last_accessed_at (simulating legacy data)
	rec, _, err := svc.Contribute(ContributeInput{
		Topic:   "legacy-entry",
		Content: "Consistent hashing minimizes key remapping when nodes join or leave a distributed cache.",
		Scope:   "project",
	})
	if err != nil {
		t.Fatalf("Contribute: %v", err)
	}
	// Remove the fields to simulate legacy data
	delete(rec.Fields, "recent_use_count")
	delete(rec.Fields, "last_accessed_at")
	if _, err := svc.store.Write(rec); err != nil {
		t.Fatalf("Write legacy: %v", err)
	}

	// Load via Get — should succeed and treat absent fields as zero
	loaded, err := svc.Get(rec.ID)
	if err != nil {
		t.Fatalf("Get legacy entry: %v", err)
	}
	svc.Close()

	// recent_use_count absent → treated as 0 before increment → now 1
	// Verify last_accessed_at is now set
	reloaded, err := svc.store.Load(loaded.ID)
	if err != nil {
		t.Fatalf("Reload: %v", err)
	}
	count := knowledgeFieldInt(reloaded.Fields, "recent_use_count")
	if count != 1 {
		t.Errorf("recent_use_count = %d, want 1 (absent treated as 0 then +1)", count)
	}
}
