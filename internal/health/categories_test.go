package health

import (
	"testing"
	"time"

	"github.com/sambeau/kanbanzai/internal/git"
	"github.com/sambeau/kanbanzai/internal/worktree"
)

func TestCheckWorktree_EmptyInput(t *testing.T) {
	t.Parallel()

	result := CheckWorktree("", nil)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
	}
}

func TestCheckWorktree_MissingPath(t *testing.T) {
	t.Parallel()

	// Create a worktree record pointing to a non-existent path
	worktrees := []worktree.Record{
		{
			ID:       "WT-12345",
			EntityID: "FEAT-001",
			Branch:   "feature/test",
			Path:     "/nonexistent/path/to/worktree",
			Status:   worktree.StatusActive,
			Created:  time.Now(),
		},
	}

	result := CheckWorktree("", worktrees)

	if result.Status != SeverityError {
		t.Errorf("Status = %v, want %v", result.Status, SeverityError)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("len(Issues) = %d, want 1", len(result.Issues))
	}

	issue := result.Issues[0]
	if issue.Severity != SeverityError {
		t.Errorf("Issue.Severity = %v, want %v", issue.Severity, SeverityError)
	}
	if issue.EntityID != "WT-12345" {
		t.Errorf("Issue.EntityID = %q, want %q", issue.EntityID, "WT-12345")
	}
}

func TestCheckWorktree_MergedWorktreeIgnored(t *testing.T) {
	t.Parallel()

	// Merged worktrees should not be checked for path existence
	worktrees := []worktree.Record{
		{
			ID:       "WT-12345",
			EntityID: "FEAT-001",
			Branch:   "feature/test",
			Path:     "/nonexistent/path",
			Status:   worktree.StatusMerged,
			Created:  time.Now(),
		},
	}

	result := CheckWorktree("", worktrees)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
	}
}

func TestCheckWorktree_AbandonedWorktreeIgnored(t *testing.T) {
	t.Parallel()

	// Abandoned worktrees should not be checked for path existence
	worktrees := []worktree.Record{
		{
			ID:       "WT-12345",
			EntityID: "FEAT-001",
			Branch:   "feature/test",
			Path:     "/nonexistent/path",
			Status:   worktree.StatusAbandoned,
			Created:  time.Now(),
		},
	}

	result := CheckWorktree("", worktrees)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
	}
}

func TestCheckWorktree_ExistingPath(t *testing.T) {
	t.Parallel()

	// Use the temp dir as a path that exists
	tempDir := t.TempDir()

	worktrees := []worktree.Record{
		{
			ID:       "WT-12345",
			EntityID: "FEAT-001",
			Branch:   "feature/test",
			Path:     tempDir,
			Status:   worktree.StatusActive,
			Created:  time.Now(),
		},
	}

	result := CheckWorktree("", worktrees)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
	}
}

func TestCheckBranch_EmptyInput(t *testing.T) {
	t.Parallel()

	thresholds := git.DefaultBranchThresholds()
	result := CheckBranch("", nil, thresholds)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
	}
}

func TestCheckBranch_SkipsMergedWorktrees(t *testing.T) {
	t.Parallel()

	worktrees := []worktree.Record{
		{
			ID:       "WT-12345",
			EntityID: "FEAT-001",
			Branch:   "feature/test",
			Path:     "/some/path",
			Status:   worktree.StatusMerged,
			Created:  time.Now(),
		},
	}

	thresholds := git.DefaultBranchThresholds()
	result := CheckBranch("", worktrees, thresholds)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
	}
}

func TestCheckBranch_SkipsEmptyBranch(t *testing.T) {
	t.Parallel()

	worktrees := []worktree.Record{
		{
			ID:       "WT-12345",
			EntityID: "FEAT-001",
			Branch:   "", // Empty branch
			Path:     "/some/path",
			Status:   worktree.StatusActive,
			Created:  time.Now(),
		},
	}

	thresholds := git.DefaultBranchThresholds()
	result := CheckBranch("", worktrees, thresholds)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
}

func TestCheckKnowledgeStaleness_EmptyInput(t *testing.T) {
	t.Parallel()

	result := CheckKnowledgeStaleness("", nil)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
	}
}

func TestCheckKnowledgeStaleness_SkipsRetiredEntries(t *testing.T) {
	t.Parallel()

	entries := []map[string]any{
		{
			"id":     "KE-12345",
			"status": "retired",
			"git_anchors": []string{
				"internal/nonexistent.go",
			},
		},
	}

	result := CheckKnowledgeStaleness("", entries)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
	}
}

func TestCheckKnowledgeStaleness_SkipsEntriesWithoutID(t *testing.T) {
	t.Parallel()

	entries := []map[string]any{
		{
			"status": "confirmed",
			"git_anchors": []string{
				"internal/nonexistent.go",
			},
		},
	}

	result := CheckKnowledgeStaleness("", entries)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
}

func TestCheckKnowledgeStaleness_NoAnchorsOK(t *testing.T) {
	t.Parallel()

	entries := []map[string]any{
		{
			"id":     "KE-12345",
			"status": "confirmed",
			// No git_anchors
		},
	}

	result := CheckKnowledgeStaleness("", entries)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
}

func TestCheckKnowledgeTTL_EmptyInput(t *testing.T) {
	t.Parallel()

	result := CheckKnowledgeTTL(nil, time.Now())

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
	}
}

func TestCheckKnowledgeTTL_SkipsRetiredEntries(t *testing.T) {
	t.Parallel()

	now := time.Now()
	// Entry with expired TTL but retired
	entries := []map[string]any{
		{
			"id":             "KE-12345",
			"status":         "retired",
			"tier":           3,
			"use_count":      0,
			"last_used":      now.AddDate(0, 0, -60).Format(time.RFC3339),
			"ttl_days":       30,
			"ttl_expires_at": now.AddDate(0, 0, -30).Format(time.RFC3339),
		},
	}

	result := CheckKnowledgeTTL(entries, now)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
	}
}

func TestCheckKnowledgeTTL_SkipsEntriesWithoutID(t *testing.T) {
	t.Parallel()

	now := time.Now()
	entries := []map[string]any{
		{
			"status":         "confirmed",
			"tier":           3,
			"ttl_expires_at": now.Add(-time.Hour).Format(time.RFC3339),
		},
	}

	result := CheckKnowledgeTTL(entries, now)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
}

func TestCheckKnowledgeTTL_ExpiredAndPruneEligible(t *testing.T) {
	t.Parallel()

	now := time.Now()
	created := now.AddDate(0, 0, -60) // Created 60 days ago (past grace period)
	lastUsed := now.AddDate(0, 0, -45)

	entries := []map[string]any{
		{
			"id":             "KE-12345",
			"status":         "confirmed",
			"tier":           3,
			"use_count":      1, // Less than 3, so eligible for pruning
			"created":        created.Format(time.RFC3339),
			"last_used":      lastUsed.Format(time.RFC3339),
			"ttl_days":       30,
			"ttl_expires_at": now.AddDate(0, 0, -15).Format(time.RFC3339), // Expired 15 days ago
		},
	}

	result := CheckKnowledgeTTL(entries, now)

	if result.Status != SeverityError {
		t.Errorf("Status = %v, want %v", result.Status, SeverityError)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("len(Issues) = %d, want 1", len(result.Issues))
	}

	issue := result.Issues[0]
	if issue.Severity != SeverityError {
		t.Errorf("Issue.Severity = %v, want %v", issue.Severity, SeverityError)
	}
	if issue.EntryID != "KE-12345" {
		t.Errorf("Issue.EntryID = %q, want %q", issue.EntryID, "KE-12345")
	}
}

func TestCheckKnowledgeTTL_ExpiresSoon(t *testing.T) {
	t.Parallel()

	now := time.Now()
	created := now.AddDate(0, 0, -30) // Created 30 days ago
	lastUsed := now.AddDate(0, 0, -10)

	entries := []map[string]any{
		{
			"id":             "KE-12345",
			"status":         "confirmed",
			"tier":           3,
			"use_count":      5, // High use count, won't be pruned
			"created":        created.Format(time.RFC3339),
			"last_used":      lastUsed.Format(time.RFC3339),
			"ttl_days":       30,
			"ttl_expires_at": now.Add(3 * 24 * time.Hour).Format(time.RFC3339), // Expires in 3 days
		},
	}

	result := CheckKnowledgeTTL(entries, now)

	if result.Status != SeverityWarning {
		t.Errorf("Status = %v, want %v", result.Status, SeverityWarning)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("len(Issues) = %d, want 1", len(result.Issues))
	}

	issue := result.Issues[0]
	if issue.Severity != SeverityWarning {
		t.Errorf("Issue.Severity = %v, want %v", issue.Severity, SeverityWarning)
	}
	if issue.EntryID != "KE-12345" {
		t.Errorf("Issue.EntryID = %q, want %q", issue.EntryID, "KE-12345")
	}
}

func TestCheckKnowledgeTTL_FarFromExpiry(t *testing.T) {
	t.Parallel()

	now := time.Now()
	entries := []map[string]any{
		{
			"id":             "KE-12345",
			"status":         "confirmed",
			"tier":           3,
			"ttl_expires_at": now.Add(30 * 24 * time.Hour).Format(time.RFC3339), // Expires in 30 days
		},
	}

	result := CheckKnowledgeTTL(entries, now)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
	}
}

func TestCheckKnowledgeTTL_NoTTLConfigured(t *testing.T) {
	t.Parallel()

	now := time.Now()
	entries := []map[string]any{
		{
			"id":     "KE-12345",
			"status": "confirmed",
			"tier":   1, // Tier 1 has no TTL
		},
	}

	result := CheckKnowledgeTTL(entries, now)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
}

func TestCheckKnowledgeConflicts_EmptyInput(t *testing.T) {
	t.Parallel()

	result := CheckKnowledgeConflicts(nil)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
	}
}

func TestCheckKnowledgeConflicts_DisputedEntry(t *testing.T) {
	t.Parallel()

	entries := []map[string]any{
		{
			"id":     "KE-12345",
			"status": "disputed",
		},
	}

	result := CheckKnowledgeConflicts(entries)

	if result.Status != SeverityError {
		t.Errorf("Status = %v, want %v", result.Status, SeverityError)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("len(Issues) = %d, want 1", len(result.Issues))
	}

	issue := result.Issues[0]
	if issue.Severity != SeverityError {
		t.Errorf("Issue.Severity = %v, want %v", issue.Severity, SeverityError)
	}
	if issue.EntryID != "KE-12345" {
		t.Errorf("Issue.EntryID = %q, want %q", issue.EntryID, "KE-12345")
	}
}

func TestCheckKnowledgeConflicts_MultipleDisputed(t *testing.T) {
	t.Parallel()

	entries := []map[string]any{
		{"id": "KE-001", "status": "disputed"},
		{"id": "KE-002", "status": "confirmed"},
		{"id": "KE-003", "status": "disputed"},
	}

	result := CheckKnowledgeConflicts(entries)

	if result.Status != SeverityError {
		t.Errorf("Status = %v, want %v", result.Status, SeverityError)
	}
	if len(result.Issues) != 2 {
		t.Fatalf("len(Issues) = %d, want 2", len(result.Issues))
	}
}

func TestCheckKnowledgeConflicts_NoDisputed(t *testing.T) {
	t.Parallel()

	entries := []map[string]any{
		{"id": "KE-001", "status": "confirmed"},
		{"id": "KE-002", "status": "contributed"},
		{"id": "KE-003", "status": "retired"},
	}

	result := CheckKnowledgeConflicts(entries)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
	}
}

func TestCheckKnowledgeConflicts_SkipsEntriesWithoutID(t *testing.T) {
	t.Parallel()

	entries := []map[string]any{
		{"status": "disputed"}, // No ID - should be skipped
	}

	result := CheckKnowledgeConflicts(entries)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
}

func TestCheckCleanup_EmptyInput(t *testing.T) {
	t.Parallel()

	result := CheckCleanup(nil, time.Now())

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
	}
}

func TestCheckCleanup_SkipsActiveWorktrees(t *testing.T) {
	t.Parallel()

	now := time.Now()
	cleanupAfter := now.Add(-24 * time.Hour) // Past cleanup time

	worktrees := []worktree.Record{
		{
			ID:           "WT-12345",
			Status:       worktree.StatusActive, // Active, not merged
			CleanupAfter: &cleanupAfter,
		},
	}

	result := CheckCleanup(worktrees, now)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
}

func TestCheckCleanup_MergedPastCleanupAfter(t *testing.T) {
	t.Parallel()

	now := time.Now()
	cleanupAfter := now.Add(-48 * time.Hour) // 2 days past cleanup time

	worktrees := []worktree.Record{
		{
			ID:           "WT-12345",
			EntityID:     "FEAT-001",
			Branch:       "feature/test",
			Path:         "/some/path",
			Status:       worktree.StatusMerged,
			Created:      now.AddDate(0, 0, -30),
			CleanupAfter: &cleanupAfter,
		},
	}

	result := CheckCleanup(worktrees, now)

	if result.Status != SeverityWarning {
		t.Errorf("Status = %v, want %v", result.Status, SeverityWarning)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("len(Issues) = %d, want 1", len(result.Issues))
	}

	issue := result.Issues[0]
	if issue.Severity != SeverityWarning {
		t.Errorf("Issue.Severity = %v, want %v", issue.Severity, SeverityWarning)
	}
	if issue.EntityID != "WT-12345" {
		t.Errorf("Issue.EntityID = %q, want %q", issue.EntityID, "WT-12345")
	}
}

func TestCheckCleanup_MergedNotYetPastCleanupAfter(t *testing.T) {
	t.Parallel()

	now := time.Now()
	cleanupAfter := now.Add(24 * time.Hour) // Future cleanup time

	worktrees := []worktree.Record{
		{
			ID:           "WT-12345",
			Status:       worktree.StatusMerged,
			CleanupAfter: &cleanupAfter,
		},
	}

	result := CheckCleanup(worktrees, now)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
	}
}

func TestCheckCleanup_MergedNoCleanupAfter(t *testing.T) {
	t.Parallel()

	worktrees := []worktree.Record{
		{
			ID:           "WT-12345",
			Status:       worktree.StatusMerged,
			CleanupAfter: nil, // No cleanup time set
		},
	}

	result := CheckCleanup(worktrees, time.Now())

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
}

func TestCheckCleanup_MultipleOverdue(t *testing.T) {
	t.Parallel()

	now := time.Now()
	cleanupAfter1 := now.Add(-24 * time.Hour)
	cleanupAfter2 := now.Add(-72 * time.Hour)

	worktrees := []worktree.Record{
		{
			ID:           "WT-001",
			Status:       worktree.StatusMerged,
			CleanupAfter: &cleanupAfter1,
		},
		{
			ID:           "WT-002",
			Status:       worktree.StatusMerged,
			CleanupAfter: &cleanupAfter2,
		},
	}

	result := CheckCleanup(worktrees, now)

	if result.Status != SeverityWarning {
		t.Errorf("Status = %v, want %v", result.Status, SeverityWarning)
	}
	if len(result.Issues) != 2 {
		t.Errorf("len(Issues) = %d, want 2", len(result.Issues))
	}
}

func TestGetEntryID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		fields map[string]any
		want   string
	}{
		{
			name:   "nil_fields",
			fields: nil,
			want:   "",
		},
		{
			name:   "empty_fields",
			fields: map[string]any{},
			want:   "",
		},
		{
			name: "has_id",
			fields: map[string]any{
				"id": "KE-12345",
			},
			want: "KE-12345",
		},
		{
			name: "id_not_string",
			fields: map[string]any{
				"id": 12345,
			},
			want: "",
		},
		{
			name: "id_empty_string",
			fields: map[string]any{
				"id": "",
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getEntryID(tt.fields)
			if got != tt.want {
				t.Errorf("getEntryID() = %q, want %q", got, tt.want)
			}
		})
	}
}
