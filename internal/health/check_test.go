package health

import (
	"testing"
	"time"

	"kanbanzai/internal/git"
	"kanbanzai/internal/worktree"
)

func TestDefaultCheckOptions(t *testing.T) {
	t.Parallel()

	opts := DefaultCheckOptions()

	if opts.IncludeOK {
		t.Error("DefaultCheckOptions().IncludeOK = true, want false")
	}
	if opts.SkipBranchCheck {
		t.Error("DefaultCheckOptions().SkipBranchCheck = true, want false")
	}
	if opts.SkipStalenessCheck {
		t.Error("DefaultCheckOptions().SkipStalenessCheck = true, want false")
	}

	// Check that thresholds are set to defaults
	defaults := git.DefaultBranchThresholds()
	if opts.BranchThresholds != defaults {
		t.Errorf("BranchThresholds = %+v, want %+v", opts.BranchThresholds, defaults)
	}
}

func TestRunHealthCheck_EmptyInputs(t *testing.T) {
	t.Parallel()

	opts := DefaultCheckOptions()
	opts.SkipBranchCheck = true
	opts.SkipStalenessCheck = true

	result := RunHealthCheck(nil, nil, time.Now(), opts)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}

	// Without IncludeOK, categories with no issues should not be included
	if len(result.Categories) != 0 {
		t.Errorf("len(Categories) = %d, want 0", len(result.Categories))
	}
}

func TestRunHealthCheck_IncludeOK(t *testing.T) {
	t.Parallel()

	opts := DefaultCheckOptions()
	opts.IncludeOK = true
	opts.SkipBranchCheck = true
	opts.SkipStalenessCheck = true

	result := RunHealthCheck(nil, nil, time.Now(), opts)

	// With IncludeOK=true, all categories should be present
	expectedCategories := []string{"worktree", "knowledge_ttl", "knowledge_conflicts", "cleanup"}
	for _, name := range expectedCategories {
		if _, ok := result.Categories[name]; !ok {
			t.Errorf("Categories[%q] missing with IncludeOK=true", name)
		}
	}
}

func TestRunHealthCheck_WithDisputedEntry(t *testing.T) {
	t.Parallel()

	entries := []map[string]any{
		{
			"id":     "KE-12345",
			"status": "disputed",
		},
	}

	opts := DefaultCheckOptions()
	opts.SkipBranchCheck = true
	opts.SkipStalenessCheck = true

	result := RunHealthCheck(nil, entries, time.Now(), opts)

	if result.Status != SeverityError {
		t.Errorf("Status = %v, want %v", result.Status, SeverityError)
	}

	conflicts, ok := result.Categories["knowledge_conflicts"]
	if !ok {
		t.Fatal("knowledge_conflicts category missing")
	}
	if conflicts.Status != SeverityError {
		t.Errorf("knowledge_conflicts.Status = %v, want %v", conflicts.Status, SeverityError)
	}
}

func TestRunHealthCheck_WithMissingWorktree(t *testing.T) {
	t.Parallel()

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

	opts := DefaultCheckOptions()
	opts.SkipBranchCheck = true
	opts.SkipStalenessCheck = true

	result := RunHealthCheck(worktrees, nil, time.Now(), opts)

	if result.Status != SeverityError {
		t.Errorf("Status = %v, want %v", result.Status, SeverityError)
	}

	wtResult, ok := result.Categories["worktree"]
	if !ok {
		t.Fatal("worktree category missing")
	}
	if wtResult.Status != SeverityError {
		t.Errorf("worktree.Status = %v, want %v", wtResult.Status, SeverityError)
	}
}

func TestRunHealthCheck_WithCleanupOverdue(t *testing.T) {
	t.Parallel()

	now := time.Now()
	cleanupAfter := now.Add(-48 * time.Hour)

	worktrees := []worktree.Record{
		{
			ID:           "WT-12345",
			EntityID:     "FEAT-001",
			Branch:       "feature/test",
			Path:         t.TempDir(), // Exists
			Status:       worktree.StatusMerged,
			Created:      now.AddDate(0, 0, -30),
			CleanupAfter: &cleanupAfter,
		},
	}

	opts := DefaultCheckOptions()
	opts.SkipBranchCheck = true
	opts.SkipStalenessCheck = true

	result := RunHealthCheck(worktrees, nil, now, opts)

	if result.Status != SeverityWarning {
		t.Errorf("Status = %v, want %v", result.Status, SeverityWarning)
	}

	cleanup, ok := result.Categories["cleanup"]
	if !ok {
		t.Fatal("cleanup category missing")
	}
	if cleanup.Status != SeverityWarning {
		t.Errorf("cleanup.Status = %v, want %v", cleanup.Status, SeverityWarning)
	}
}

func TestRunHealthCheck_MixedSeverities(t *testing.T) {
	t.Parallel()

	now := time.Now()
	cleanupAfter := now.Add(-48 * time.Hour)

	worktrees := []worktree.Record{
		{
			ID:       "WT-001",
			EntityID: "FEAT-001",
			Branch:   "feature/test",
			Path:     "/nonexistent/path", // Error: missing
			Status:   worktree.StatusActive,
			Created:  now.AddDate(0, 0, -30),
		},
		{
			ID:           "WT-002",
			EntityID:     "FEAT-002",
			Branch:       "feature/other",
			Path:         t.TempDir(),
			Status:       worktree.StatusMerged,
			CleanupAfter: &cleanupAfter, // Warning: overdue cleanup
		},
	}

	opts := DefaultCheckOptions()
	opts.SkipBranchCheck = true
	opts.SkipStalenessCheck = true

	result := RunHealthCheck(worktrees, nil, now, opts)

	// Overall status should be error (worst of error and warning)
	if result.Status != SeverityError {
		t.Errorf("Status = %v, want %v", result.Status, SeverityError)
	}
}

func TestRunHealthCheck_SkipBranchCheck(t *testing.T) {
	t.Parallel()

	worktrees := []worktree.Record{
		{
			ID:       "WT-12345",
			EntityID: "FEAT-001",
			Branch:   "feature/nonexistent-branch",
			Path:     t.TempDir(),
			Status:   worktree.StatusActive,
			Created:  time.Now(),
		},
	}

	opts := DefaultCheckOptions()
	opts.SkipBranchCheck = true
	opts.SkipStalenessCheck = true

	result := RunHealthCheck(worktrees, nil, time.Now(), opts)

	// Should not have branch category when skipped
	if _, ok := result.Categories["branch"]; ok {
		t.Error("branch category should not be present when SkipBranchCheck=true")
	}
}

func TestRunHealthCheck_SkipStalenessCheck(t *testing.T) {
	t.Parallel()

	entries := []map[string]any{
		{
			"id":     "KE-12345",
			"status": "confirmed",
			"git_anchors": []string{
				"internal/nonexistent.go",
			},
		},
	}

	opts := DefaultCheckOptions()
	opts.SkipBranchCheck = true
	opts.SkipStalenessCheck = true

	result := RunHealthCheck(nil, entries, time.Now(), opts)

	// Should not have knowledge_staleness category when skipped
	if _, ok := result.Categories["knowledge_staleness"]; ok {
		t.Error("knowledge_staleness category should not be present when SkipStalenessCheck=true")
	}
}

func TestDetermineOverallStatus_Empty(t *testing.T) {
	t.Parallel()

	status := DetermineOverallStatus(nil)
	if status != SeverityOK {
		t.Errorf("DetermineOverallStatus(nil) = %v, want %v", status, SeverityOK)
	}

	status = DetermineOverallStatus(map[string]CategoryResult{})
	if status != SeverityOK {
		t.Errorf("DetermineOverallStatus({}) = %v, want %v", status, SeverityOK)
	}
}

func TestDetermineOverallStatus_AllOK(t *testing.T) {
	t.Parallel()

	categories := map[string]CategoryResult{
		"worktree": {Status: SeverityOK},
		"branch":   {Status: SeverityOK},
		"cleanup":  {Status: SeverityOK},
	}

	status := DetermineOverallStatus(categories)
	if status != SeverityOK {
		t.Errorf("DetermineOverallStatus = %v, want %v", status, SeverityOK)
	}
}

func TestDetermineOverallStatus_OneWarning(t *testing.T) {
	t.Parallel()

	categories := map[string]CategoryResult{
		"worktree": {Status: SeverityOK},
		"branch":   {Status: SeverityWarning},
		"cleanup":  {Status: SeverityOK},
	}

	status := DetermineOverallStatus(categories)
	if status != SeverityWarning {
		t.Errorf("DetermineOverallStatus = %v, want %v", status, SeverityWarning)
	}
}

func TestDetermineOverallStatus_OneError(t *testing.T) {
	t.Parallel()

	categories := map[string]CategoryResult{
		"worktree": {Status: SeverityOK},
		"branch":   {Status: SeverityError},
		"cleanup":  {Status: SeverityOK},
	}

	status := DetermineOverallStatus(categories)
	if status != SeverityError {
		t.Errorf("DetermineOverallStatus = %v, want %v", status, SeverityError)
	}
}

func TestDetermineOverallStatus_MixedSeverities(t *testing.T) {
	t.Parallel()

	categories := map[string]CategoryResult{
		"worktree":            {Status: SeverityOK},
		"branch":              {Status: SeverityWarning},
		"knowledge_conflicts": {Status: SeverityError},
		"cleanup":             {Status: SeverityWarning},
	}

	status := DetermineOverallStatus(categories)
	if status != SeverityError {
		t.Errorf("DetermineOverallStatus = %v, want %v", status, SeverityError)
	}
}

func TestDetermineOverallStatus_AllErrors(t *testing.T) {
	t.Parallel()

	categories := map[string]CategoryResult{
		"worktree": {Status: SeverityError},
		"branch":   {Status: SeverityError},
		"cleanup":  {Status: SeverityError},
	}

	status := DetermineOverallStatus(categories)
	if status != SeverityError {
		t.Errorf("DetermineOverallStatus = %v, want %v", status, SeverityError)
	}
}

func TestDetermineOverallStatus_AllWarnings(t *testing.T) {
	t.Parallel()

	categories := map[string]CategoryResult{
		"worktree": {Status: SeverityWarning},
		"branch":   {Status: SeverityWarning},
		"cleanup":  {Status: SeverityWarning},
	}

	status := DetermineOverallStatus(categories)
	if status != SeverityWarning {
		t.Errorf("DetermineOverallStatus = %v, want %v", status, SeverityWarning)
	}
}
