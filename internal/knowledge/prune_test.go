package knowledge

import (
	"testing"
	"time"
)

func TestPruneExpiredEntries_Basic(t *testing.T) {
	config := DefaultTTLConfig()
	now := time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)
	expiredDate := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	notExpiredDate := now.Add(30 * 24 * time.Hour)
	oldCreated := expiredDate.Add(-60 * 24 * time.Hour)

	entries := []map[string]any{
		// Should be pruned: tier 3, expired, low use count
		{
			"id":             "KE-001",
			"topic":          "expired-tier3",
			"tier":           3,
			"use_count":      1,
			"ttl_expires_at": expiredDate.Format(time.RFC3339),
			"created":        oldCreated.Format(time.RFC3339),
		},
		// Should NOT be pruned: tier 3, expired, high use count
		{
			"id":             "KE-002",
			"topic":          "valid-tier3",
			"tier":           3,
			"use_count":      5,
			"ttl_expires_at": expiredDate.Format(time.RFC3339),
			"created":        oldCreated.Format(time.RFC3339),
		},
		// Should NOT be pruned: tier 3, not expired
		{
			"id":             "KE-003",
			"topic":          "fresh-tier3",
			"tier":           3,
			"use_count":      0,
			"ttl_expires_at": notExpiredDate.Format(time.RFC3339),
			"created":        oldCreated.Format(time.RFC3339),
		},
		// Should be pruned: tier 2, expired, low confidence
		{
			"id":             "KE-004",
			"topic":          "expired-tier2",
			"tier":           2,
			"confidence":     0.3,
			"ttl_expires_at": expiredDate.Format(time.RFC3339),
			"created":        oldCreated.Format(time.RFC3339),
		},
		// Should NOT be pruned: tier 2, expired, high confidence
		{
			"id":             "KE-005",
			"topic":          "valid-tier2",
			"tier":           2,
			"confidence":     0.8,
			"ttl_expires_at": expiredDate.Format(time.RFC3339),
			"created":        oldCreated.Format(time.RFC3339),
		},
	}

	results := PruneExpiredEntries(entries, now, config, PruneOptions{})

	if len(results) != 2 {
		t.Errorf("expected 2 entries to prune, got %d", len(results))
		for _, r := range results {
			t.Logf("pruned: %s (tier %d) - %s", r.EntryID, r.Tier, r.Reason)
		}
		return
	}

	// Check that the right entries were identified
	prunedIDs := make(map[string]bool)
	for _, r := range results {
		prunedIDs[r.EntryID] = true
	}

	if !prunedIDs["KE-001"] {
		t.Error("KE-001 should be pruned")
	}
	if !prunedIDs["KE-004"] {
		t.Error("KE-004 should be pruned")
	}
}

func TestPruneExpiredEntries_DryRun(t *testing.T) {
	config := DefaultTTLConfig()
	now := time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)
	expiredDate := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	oldCreated := expiredDate.Add(-60 * 24 * time.Hour)

	entries := []map[string]any{
		{
			"id":             "KE-001",
			"topic":          "test-entry",
			"tier":           3,
			"use_count":      0,
			"ttl_expires_at": expiredDate.Format(time.RFC3339),
			"created":        oldCreated.Format(time.RFC3339),
		},
	}

	// Dry-run should still return results
	results := PruneExpiredEntries(entries, now, config, PruneOptions{DryRun: true})

	if len(results) != 1 {
		t.Errorf("dry-run should return results, got %d", len(results))
	}

	// The original entry should not be modified (this is just a check that
	// PruneExpiredEntries doesn't mutate - actual retirement is done by caller)
	if entries[0]["status"] == "retired" {
		t.Error("dry-run should not modify entries")
	}
}

func TestPruneExpiredEntries_TierFilter(t *testing.T) {
	config := DefaultTTLConfig()
	now := time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)
	expiredDate := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	oldCreated := expiredDate.Add(-60 * 24 * time.Hour)

	entries := []map[string]any{
		{
			"id":             "KE-001",
			"topic":          "tier3-entry",
			"tier":           3,
			"use_count":      0,
			"ttl_expires_at": expiredDate.Format(time.RFC3339),
			"created":        oldCreated.Format(time.RFC3339),
		},
		{
			"id":             "KE-002",
			"topic":          "tier2-entry",
			"tier":           2,
			"confidence":     0.2,
			"ttl_expires_at": expiredDate.Format(time.RFC3339),
			"created":        oldCreated.Format(time.RFC3339),
		},
	}

	// Filter to tier 3 only
	results := PruneExpiredEntries(entries, now, config, PruneOptions{Tier: 3})

	if len(results) != 1 {
		t.Errorf("expected 1 tier-3 entry, got %d", len(results))
	}

	if len(results) > 0 && results[0].Tier != 3 {
		t.Errorf("expected tier 3, got tier %d", results[0].Tier)
	}

	// Filter to tier 2 only
	results = PruneExpiredEntries(entries, now, config, PruneOptions{Tier: 2})

	if len(results) != 1 {
		t.Errorf("expected 1 tier-2 entry, got %d", len(results))
	}

	if len(results) > 0 && results[0].Tier != 2 {
		t.Errorf("expected tier 2, got tier %d", results[0].Tier)
	}
}

func TestPruneExpiredEntries_Tier1NeverPruned(t *testing.T) {
	config := DefaultTTLConfig()
	now := time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)
	expiredDate := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	oldCreated := expiredDate.Add(-365 * 24 * time.Hour)

	entries := []map[string]any{
		{
			"id":             "KE-001",
			"topic":          "tier1-entry",
			"tier":           1,
			"use_count":      0,
			"confidence":     0.1,
			"ttl_expires_at": expiredDate.Format(time.RFC3339),
			"created":        oldCreated.Format(time.RFC3339),
		},
	}

	results := PruneExpiredEntries(entries, now, config, PruneOptions{})

	if len(results) != 0 {
		t.Errorf("tier 1 should never be pruned, got %d results", len(results))
	}
}

func TestPruneExpiredEntries_GracePeriodExemption(t *testing.T) {
	config := DefaultTTLConfig()
	now := time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)

	// Entry created 3 days ago (within 7-day grace period)
	recentCreated := now.Add(-3 * 24 * time.Hour)
	// TTL already expired (set to yesterday)
	expiredDate := now.Add(-1 * 24 * time.Hour)

	entries := []map[string]any{
		{
			"id":             "KE-001",
			"topic":          "new-entry",
			"tier":           3,
			"use_count":      0,
			"ttl_expires_at": expiredDate.Format(time.RFC3339),
			"created":        recentCreated.Format(time.RFC3339),
		},
	}

	results := PruneExpiredEntries(entries, now, config, PruneOptions{})

	if len(results) != 0 {
		t.Errorf("entry within grace period should be exempt, got %d results", len(results))
	}
}

func TestPruneExpiredEntries_SkipsNilEntries(t *testing.T) {
	config := DefaultTTLConfig()
	now := time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)
	expiredDate := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	oldCreated := expiredDate.Add(-60 * 24 * time.Hour)

	entries := []map[string]any{
		nil,
		{
			"id":             "KE-001",
			"topic":          "valid-entry",
			"tier":           3,
			"use_count":      0,
			"ttl_expires_at": expiredDate.Format(time.RFC3339),
			"created":        oldCreated.Format(time.RFC3339),
		},
		nil,
	}

	// Should not panic
	results := PruneExpiredEntries(entries, now, config, PruneOptions{})

	if len(results) != 1 {
		t.Errorf("expected 1 result (skipping nils), got %d", len(results))
	}
}

func TestPruneExpiredEntries_SkipsRetired(t *testing.T) {
	config := DefaultTTLConfig()
	now := time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)
	expiredDate := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	oldCreated := expiredDate.Add(-60 * 24 * time.Hour)

	entries := []map[string]any{
		{
			"id":             "KE-001",
			"topic":          "already-retired",
			"tier":           3,
			"status":         "retired",
			"use_count":      0,
			"ttl_expires_at": expiredDate.Format(time.RFC3339),
			"created":        oldCreated.Format(time.RFC3339),
		},
	}

	results := PruneExpiredEntries(entries, now, config, PruneOptions{})

	if len(results) != 0 {
		t.Errorf("already retired entries should not be pruned again, got %d results", len(results))
	}
}

func TestPruneExpiredEntries_EmptyInput(t *testing.T) {
	config := DefaultTTLConfig()
	now := time.Now()

	results := PruneExpiredEntries(nil, now, config, PruneOptions{})
	if len(results) != 0 {
		t.Errorf("nil input should return empty results, got %d", len(results))
	}

	results = PruneExpiredEntries([]map[string]any{}, now, config, PruneOptions{})
	if len(results) != 0 {
		t.Errorf("empty input should return empty results, got %d", len(results))
	}
}

func TestPruneResult_Fields(t *testing.T) {
	config := DefaultTTLConfig()
	now := time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)
	expiredDate := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	oldCreated := expiredDate.Add(-60 * 24 * time.Hour)

	entries := []map[string]any{
		{
			"id":             "KE-TEST123",
			"topic":          "my-test-topic",
			"tier":           3,
			"use_count":      1,
			"ttl_expires_at": expiredDate.Format(time.RFC3339),
			"created":        oldCreated.Format(time.RFC3339),
		},
	}

	results := PruneExpiredEntries(entries, now, config, PruneOptions{})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.EntryID != "KE-TEST123" {
		t.Errorf("EntryID = %s, want KE-TEST123", r.EntryID)
	}
	if r.Topic != "my-test-topic" {
		t.Errorf("Topic = %s, want my-test-topic", r.Topic)
	}
	if r.Tier != 3 {
		t.Errorf("Tier = %d, want 3", r.Tier)
	}
	if r.Reason == "" {
		t.Error("Reason should not be empty")
	}
}

func TestComputeStats(t *testing.T) {
	results := []PruneResult{
		{EntryID: "KE-001", Tier: 3, Reason: "expired"},
		{EntryID: "KE-002", Tier: 3, Reason: "expired"},
		{EntryID: "KE-003", Tier: 2, Reason: "expired"},
		{EntryID: "KE-004", Tier: 3, Reason: "expired"},
	}

	stats := ComputeStats(results, 10)

	if stats.TotalChecked != 10 {
		t.Errorf("TotalChecked = %d, want 10", stats.TotalChecked)
	}
	if stats.TotalPruned != 4 {
		t.Errorf("TotalPruned = %d, want 4", stats.TotalPruned)
	}
	if stats.Tier2Pruned != 1 {
		t.Errorf("Tier2Pruned = %d, want 1", stats.Tier2Pruned)
	}
	if stats.Tier3Pruned != 3 {
		t.Errorf("Tier3Pruned = %d, want 3", stats.Tier3Pruned)
	}
}

func TestComputeStats_Empty(t *testing.T) {
	stats := ComputeStats(nil, 5)

	if stats.TotalChecked != 5 {
		t.Errorf("TotalChecked = %d, want 5", stats.TotalChecked)
	}
	if stats.TotalPruned != 0 {
		t.Errorf("TotalPruned = %d, want 0", stats.TotalPruned)
	}
	if stats.Tier2Pruned != 0 {
		t.Errorf("Tier2Pruned = %d, want 0", stats.Tier2Pruned)
	}
	if stats.Tier3Pruned != 0 {
		t.Errorf("Tier3Pruned = %d, want 0", stats.Tier3Pruned)
	}
}

func TestExtractFields(t *testing.T) {
	fields := map[string]any{
		"id":   "KE-001",
		"tier": 3,
	}

	result := ExtractFields(fields)
	if result["id"] != "KE-001" {
		t.Error("ExtractFields should return the same map")
	}
}

func TestPruneExpiredEntries_CustomConfig(t *testing.T) {
	// Custom config with shorter TTLs
	config := TTLConfig{
		Tier3Days:       7,
		Tier2Days:       30,
		GracePeriodDays: 2,
	}
	now := time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)

	// Entry created 10 days ago, last used 10 days ago
	// With 7-day TTL for tier 3, this should be expired
	oldDate := now.Add(-10 * 24 * time.Hour)

	entries := []map[string]any{
		{
			"id":        "KE-001",
			"topic":     "test-entry",
			"tier":      3,
			"use_count": 0,
			"last_used": oldDate.Format(time.RFC3339),
			"created":   oldDate.Format(time.RFC3339),
			"ttl_days":  7,
		},
	}

	results := PruneExpiredEntries(entries, now, config, PruneOptions{})

	if len(results) != 1 {
		t.Errorf("expected 1 result with custom config, got %d", len(results))
	}
}
