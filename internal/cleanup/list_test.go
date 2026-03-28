package cleanup

import (
	"testing"
	"time"

	"github.com/sambeau/kanbanzai/internal/worktree"
)

func TestListCleanupItems_EmptyRecords(t *testing.T) {
	now := time.Now()
	opts := ListOptions{
		IncludePending:   true,
		IncludeScheduled: true,
		IncludeAbandoned: true,
	}

	items := ListCleanupItems(nil, now, opts)
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestListCleanupItems_ExcludesActive(t *testing.T) {
	now := time.Now()
	records := []worktree.Record{
		{
			ID:       "WT-001",
			EntityID: "FEAT-001",
			Branch:   "feature/test",
			Path:     ".worktrees/feature-test",
			Status:   worktree.StatusActive,
		},
	}

	opts := ListOptions{
		IncludePending:   true,
		IncludeScheduled: true,
		IncludeAbandoned: true,
	}

	items := ListCleanupItems(records, now, opts)
	if len(items) != 0 {
		t.Errorf("expected 0 items (active excluded), got %d", len(items))
	}
}

func TestListCleanupItems_IncludesPending(t *testing.T) {
	now := time.Now()
	mergedAt := now.Add(-10 * 24 * time.Hour)    // 10 days ago
	cleanupAfter := now.Add(-3 * 24 * time.Hour) // 3 days ago (past grace)

	records := []worktree.Record{
		{
			ID:           "WT-001",
			EntityID:     "FEAT-001",
			Branch:       "feature/test",
			Path:         ".worktrees/feature-test",
			Status:       worktree.StatusMerged,
			MergedAt:     &mergedAt,
			CleanupAfter: &cleanupAfter,
		},
	}

	// Only include pending (past grace period)
	opts := ListOptions{
		IncludePending: true,
	}

	items := ListCleanupItems(records, now, opts)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	if items[0].Status != "ready" {
		t.Errorf("expected status 'ready', got %q", items[0].Status)
	}
	if items[0].WorktreeID != "WT-001" {
		t.Errorf("expected WorktreeID 'WT-001', got %q", items[0].WorktreeID)
	}
}

func TestListCleanupItems_IncludesScheduled(t *testing.T) {
	now := time.Now()
	mergedAt := now.Add(-2 * 24 * time.Hour)    // 2 days ago
	cleanupAfter := now.Add(5 * 24 * time.Hour) // 5 days from now (within grace)

	records := []worktree.Record{
		{
			ID:           "WT-001",
			EntityID:     "FEAT-001",
			Branch:       "feature/test",
			Path:         ".worktrees/feature-test",
			Status:       worktree.StatusMerged,
			MergedAt:     &mergedAt,
			CleanupAfter: &cleanupAfter,
		},
	}

	// Only include scheduled (within grace period)
	opts := ListOptions{
		IncludeScheduled: true,
	}

	items := ListCleanupItems(records, now, opts)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	if items[0].Status != "scheduled" {
		t.Errorf("expected status 'scheduled', got %q", items[0].Status)
	}
}

func TestListCleanupItems_IncludesAbandoned(t *testing.T) {
	now := time.Now()

	records := []worktree.Record{
		{
			ID:       "WT-001",
			EntityID: "FEAT-001",
			Branch:   "feature/test",
			Path:     ".worktrees/feature-test",
			Status:   worktree.StatusAbandoned,
		},
	}

	// Only include abandoned
	opts := ListOptions{
		IncludeAbandoned: true,
	}

	items := ListCleanupItems(records, now, opts)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	if items[0].Status != "abandoned" {
		t.Errorf("expected status 'abandoned', got %q", items[0].Status)
	}
}

func TestListCleanupItems_ExcludesBasedOnOptions(t *testing.T) {
	now := time.Now()
	mergedAt := now.Add(-10 * 24 * time.Hour)
	cleanupAfter := now.Add(-3 * 24 * time.Hour) // Past grace

	records := []worktree.Record{
		{
			ID:           "WT-001",
			EntityID:     "FEAT-001",
			Branch:       "feature/ready",
			Path:         ".worktrees/feature-ready",
			Status:       worktree.StatusMerged,
			MergedAt:     &mergedAt,
			CleanupAfter: &cleanupAfter,
		},
		{
			ID:       "WT-002",
			EntityID: "FEAT-002",
			Branch:   "feature/abandoned",
			Path:     ".worktrees/feature-abandoned",
			Status:   worktree.StatusAbandoned,
		},
	}

	// Exclude pending (ready), only include abandoned
	opts := ListOptions{
		IncludePending:   false,
		IncludeAbandoned: true,
	}

	items := ListCleanupItems(records, now, opts)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	if items[0].WorktreeID != "WT-002" {
		t.Errorf("expected WorktreeID 'WT-002', got %q", items[0].WorktreeID)
	}
}

func TestListCleanupItems_Sorting(t *testing.T) {
	now := time.Now()

	// Create records with different cleanup times
	cleanup1 := now.Add(-1 * 24 * time.Hour) // 1 day ago
	cleanup2 := now.Add(-3 * 24 * time.Hour) // 3 days ago (earliest)
	cleanup3 := now.Add(-2 * 24 * time.Hour) // 2 days ago

	merged := now.Add(-10 * 24 * time.Hour)

	records := []worktree.Record{
		{
			ID:           "WT-003",
			EntityID:     "FEAT-003",
			Branch:       "feature/third",
			Path:         ".worktrees/feature-third",
			Status:       worktree.StatusMerged,
			MergedAt:     &merged,
			CleanupAfter: &cleanup1,
		},
		{
			ID:           "WT-001",
			EntityID:     "FEAT-001",
			Branch:       "feature/first",
			Path:         ".worktrees/feature-first",
			Status:       worktree.StatusMerged,
			MergedAt:     &merged,
			CleanupAfter: &cleanup2, // Earliest
		},
		{
			ID:           "WT-002",
			EntityID:     "FEAT-002",
			Branch:       "feature/second",
			Path:         ".worktrees/feature-second",
			Status:       worktree.StatusMerged,
			MergedAt:     &merged,
			CleanupAfter: &cleanup3,
		},
	}

	opts := ListOptions{
		IncludePending: true,
	}

	items := ListCleanupItems(records, now, opts)
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	// Should be sorted by cleanup time (earliest first)
	if items[0].WorktreeID != "WT-001" {
		t.Errorf("expected first item 'WT-001' (earliest), got %q", items[0].WorktreeID)
	}
	if items[1].WorktreeID != "WT-002" {
		t.Errorf("expected second item 'WT-002', got %q", items[1].WorktreeID)
	}
	if items[2].WorktreeID != "WT-003" {
		t.Errorf("expected third item 'WT-003' (latest), got %q", items[2].WorktreeID)
	}
}

func TestListCleanupItems_SortingAbandonedFirst(t *testing.T) {
	now := time.Now()
	cleanup := now.Add(-1 * 24 * time.Hour)
	merged := now.Add(-10 * 24 * time.Hour)

	records := []worktree.Record{
		{
			ID:           "WT-002",
			EntityID:     "FEAT-002",
			Branch:       "feature/merged",
			Path:         ".worktrees/feature-merged",
			Status:       worktree.StatusMerged,
			MergedAt:     &merged,
			CleanupAfter: &cleanup,
		},
		{
			ID:       "WT-001",
			EntityID: "FEAT-001",
			Branch:   "feature/abandoned",
			Path:     ".worktrees/feature-abandoned",
			Status:   worktree.StatusAbandoned,
			// No CleanupAfter - should sort first
		},
	}

	opts := ListOptions{
		IncludePending:   true,
		IncludeAbandoned: true,
	}

	items := ListCleanupItems(records, now, opts)
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	// Abandoned (no cleanup time) should sort first
	if items[0].WorktreeID != "WT-001" {
		t.Errorf("expected first item 'WT-001' (abandoned), got %q", items[0].WorktreeID)
	}
}

func TestListCleanupItems_SortingByIDWhenSameTime(t *testing.T) {
	now := time.Now()
	cleanup := now.Add(-1 * 24 * time.Hour)
	merged := now.Add(-10 * 24 * time.Hour)

	records := []worktree.Record{
		{
			ID:           "WT-003",
			EntityID:     "FEAT-003",
			Branch:       "feature/c",
			Path:         ".worktrees/feature-c",
			Status:       worktree.StatusMerged,
			MergedAt:     &merged,
			CleanupAfter: &cleanup,
		},
		{
			ID:           "WT-001",
			EntityID:     "FEAT-001",
			Branch:       "feature/a",
			Path:         ".worktrees/feature-a",
			Status:       worktree.StatusMerged,
			MergedAt:     &merged,
			CleanupAfter: &cleanup,
		},
		{
			ID:           "WT-002",
			EntityID:     "FEAT-002",
			Branch:       "feature/b",
			Path:         ".worktrees/feature-b",
			Status:       worktree.StatusMerged,
			MergedAt:     &merged,
			CleanupAfter: &cleanup,
		},
	}

	opts := ListOptions{
		IncludePending: true,
	}

	items := ListCleanupItems(records, now, opts)
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	// Same cleanup time - should be sorted by ID
	if items[0].WorktreeID != "WT-001" {
		t.Errorf("expected first item 'WT-001', got %q", items[0].WorktreeID)
	}
	if items[1].WorktreeID != "WT-002" {
		t.Errorf("expected second item 'WT-002', got %q", items[1].WorktreeID)
	}
	if items[2].WorktreeID != "WT-003" {
		t.Errorf("expected third item 'WT-003', got %q", items[2].WorktreeID)
	}
}

func TestListCleanupItems_CleanupAtExactTime(t *testing.T) {
	now := time.Now()
	cleanupAfter := now // Exactly now

	records := []worktree.Record{
		{
			ID:           "WT-001",
			EntityID:     "FEAT-001",
			Branch:       "feature/test",
			Path:         ".worktrees/feature-test",
			Status:       worktree.StatusMerged,
			MergedAt:     &cleanupAfter,
			CleanupAfter: &cleanupAfter,
		},
	}

	opts := ListOptions{
		IncludePending: true,
	}

	items := ListCleanupItems(records, now, opts)
	if len(items) != 1 {
		t.Fatalf("expected 1 item (cleanup at exact time), got %d", len(items))
	}

	if items[0].Status != "ready" {
		t.Errorf("expected status 'ready' at exact cleanup time, got %q", items[0].Status)
	}
}

func TestListCleanupItems_MergedWithoutCleanupAfter(t *testing.T) {
	now := time.Now()
	mergedAt := now.Add(-10 * 24 * time.Hour)

	records := []worktree.Record{
		{
			ID:           "WT-001",
			EntityID:     "FEAT-001",
			Branch:       "feature/test",
			Path:         ".worktrees/feature-test",
			Status:       worktree.StatusMerged,
			MergedAt:     &mergedAt,
			CleanupAfter: nil, // No cleanup scheduled
		},
	}

	opts := ListOptions{
		IncludePending: true,
	}

	items := ListCleanupItems(records, now, opts)
	if len(items) != 1 {
		t.Fatalf("expected 1 item (merged without cleanup_after treated as ready), got %d", len(items))
	}

	if items[0].Status != "ready" {
		t.Errorf("expected status 'ready', got %q", items[0].Status)
	}
}

func TestListCleanupItems_ItemFields(t *testing.T) {
	now := time.Now()
	mergedAt := now.Add(-10 * 24 * time.Hour)
	cleanupAfter := now.Add(-3 * 24 * time.Hour)

	records := []worktree.Record{
		{
			ID:           "WT-001",
			EntityID:     "FEAT-123",
			Branch:       "feature/test-branch",
			Path:         ".worktrees/feature-test-branch",
			Status:       worktree.StatusMerged,
			MergedAt:     &mergedAt,
			CleanupAfter: &cleanupAfter,
		},
	}

	opts := ListOptions{
		IncludePending: true,
	}

	items := ListCleanupItems(records, now, opts)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	item := items[0]
	if item.WorktreeID != "WT-001" {
		t.Errorf("WorktreeID = %q, want %q", item.WorktreeID, "WT-001")
	}
	if item.EntityID != "FEAT-123" {
		t.Errorf("EntityID = %q, want %q", item.EntityID, "FEAT-123")
	}
	if item.Branch != "feature/test-branch" {
		t.Errorf("Branch = %q, want %q", item.Branch, "feature/test-branch")
	}
	if item.Path != ".worktrees/feature-test-branch" {
		t.Errorf("Path = %q, want %q", item.Path, ".worktrees/feature-test-branch")
	}
	if !item.MergedAt.Equal(mergedAt) {
		t.Errorf("MergedAt = %v, want %v", item.MergedAt, mergedAt)
	}
	if !item.CleanupAfter.Equal(cleanupAfter) {
		t.Errorf("CleanupAfter = %v, want %v", item.CleanupAfter, cleanupAfter)
	}
}

func TestListReadyItems(t *testing.T) {
	now := time.Now()
	cleanupPast := now.Add(-3 * 24 * time.Hour)
	cleanupFuture := now.Add(5 * 24 * time.Hour)
	merged := now.Add(-10 * 24 * time.Hour)

	records := []worktree.Record{
		{
			ID:           "WT-001",
			EntityID:     "FEAT-001",
			Branch:       "feature/ready",
			Path:         ".worktrees/feature-ready",
			Status:       worktree.StatusMerged,
			MergedAt:     &merged,
			CleanupAfter: &cleanupPast, // Ready
		},
		{
			ID:           "WT-002",
			EntityID:     "FEAT-002",
			Branch:       "feature/scheduled",
			Path:         ".worktrees/feature-scheduled",
			Status:       worktree.StatusMerged,
			MergedAt:     &merged,
			CleanupAfter: &cleanupFuture, // Not ready (within grace)
		},
		{
			ID:       "WT-003",
			EntityID: "FEAT-003",
			Branch:   "feature/abandoned",
			Path:     ".worktrees/feature-abandoned",
			Status:   worktree.StatusAbandoned,
		},
	}

	items := ListReadyItems(records, now)

	// Should include ready (WT-001) and abandoned (WT-003), but not scheduled (WT-002)
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	// Check both expected items are present
	hasWT001 := false
	hasWT003 := false
	for _, item := range items {
		if item.WorktreeID == "WT-001" {
			hasWT001 = true
		}
		if item.WorktreeID == "WT-003" {
			hasWT003 = true
		}
	}

	if !hasWT001 {
		t.Error("expected WT-001 (ready) to be in results")
	}
	if !hasWT003 {
		t.Error("expected WT-003 (abandoned) to be in results")
	}
}

func TestListCleanupItems_MixedStatuses(t *testing.T) {
	now := time.Now()
	cleanupPast := now.Add(-3 * 24 * time.Hour)
	cleanupFuture := now.Add(5 * 24 * time.Hour)
	merged := now.Add(-10 * 24 * time.Hour)

	records := []worktree.Record{
		{
			ID:       "WT-001",
			EntityID: "FEAT-001",
			Branch:   "feature/active",
			Path:     ".worktrees/feature-active",
			Status:   worktree.StatusActive,
		},
		{
			ID:           "WT-002",
			EntityID:     "FEAT-002",
			Branch:       "feature/ready",
			Path:         ".worktrees/feature-ready",
			Status:       worktree.StatusMerged,
			MergedAt:     &merged,
			CleanupAfter: &cleanupPast,
		},
		{
			ID:           "WT-003",
			EntityID:     "FEAT-003",
			Branch:       "feature/scheduled",
			Path:         ".worktrees/feature-scheduled",
			Status:       worktree.StatusMerged,
			MergedAt:     &merged,
			CleanupAfter: &cleanupFuture,
		},
		{
			ID:       "WT-004",
			EntityID: "FEAT-004",
			Branch:   "feature/abandoned",
			Path:     ".worktrees/feature-abandoned",
			Status:   worktree.StatusAbandoned,
		},
	}

	// Include all types
	opts := ListOptions{
		IncludePending:   true,
		IncludeScheduled: true,
		IncludeAbandoned: true,
	}

	items := ListCleanupItems(records, now, opts)

	// Should include all except active
	if len(items) != 3 {
		t.Fatalf("expected 3 items (all except active), got %d", len(items))
	}

	// Check statuses
	statusMap := make(map[string]string)
	for _, item := range items {
		statusMap[item.WorktreeID] = item.Status
	}

	if statusMap["WT-002"] != "ready" {
		t.Errorf("WT-002 status = %q, want 'ready'", statusMap["WT-002"])
	}
	if statusMap["WT-003"] != "scheduled" {
		t.Errorf("WT-003 status = %q, want 'scheduled'", statusMap["WT-003"])
	}
	if statusMap["WT-004"] != "abandoned" {
		t.Errorf("WT-004 status = %q, want 'abandoned'", statusMap["WT-004"])
	}
}
