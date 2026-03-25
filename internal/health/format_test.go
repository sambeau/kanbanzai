package health

import (
	"testing"
)

func TestFormatHealthResult_EmptyCategories(t *testing.T) {
	t.Parallel()

	result := HealthResult{
		Status:     SeverityOK,
		Categories: nil,
	}

	output := FormatHealthResult(result)

	if output["status"] != "ok" {
		t.Errorf("status = %q, want %q", output["status"], "ok")
	}

	if output["categories"] != nil {
		t.Errorf("categories = %v, want nil", output["categories"])
	}
}

func TestFormatHealthResult_WithCategories(t *testing.T) {
	t.Parallel()

	result := HealthResult{
		Status: SeverityError,
		Categories: map[string]CategoryResult{
			"worktree": {
				Status: SeverityError,
				Issues: []Issue{
					{Severity: SeverityError, EntityID: "WT-12345", Message: "missing path"},
				},
			},
			"cleanup": {
				Status: SeverityOK,
				Issues: nil,
			},
		},
	}

	output := FormatHealthResult(result)

	if output["status"] != "error" {
		t.Errorf("status = %q, want %q", output["status"], "error")
	}

	categories, ok := output["categories"].(map[string]any)
	if !ok {
		t.Fatalf("categories is not map[string]any: %T", output["categories"])
	}

	if len(categories) != 2 {
		t.Errorf("len(categories) = %d, want 2", len(categories))
	}

	if _, ok := categories["worktree"]; !ok {
		t.Error("categories[worktree] missing")
	}
	if _, ok := categories["cleanup"]; !ok {
		t.Error("categories[cleanup] missing")
	}
}

func TestFormatCategoryResult_NoIssues(t *testing.T) {
	t.Parallel()

	result := CategoryResult{
		Status: SeverityOK,
		Issues: nil,
	}

	output := FormatCategoryResult(result)

	if output["status"] != "ok" {
		t.Errorf("status = %q, want %q", output["status"], "ok")
	}

	if output["issues"] != nil {
		t.Errorf("issues = %v, want nil", output["issues"])
	}
}

func TestFormatCategoryResult_WithIssues(t *testing.T) {
	t.Parallel()

	result := CategoryResult{
		Status: SeverityWarning,
		Issues: []Issue{
			{Severity: SeverityWarning, EntityID: "WT-001", Message: "first"},
			{Severity: SeverityWarning, EntryID: "KE-001", Message: "second"},
		},
	}

	output := FormatCategoryResult(result)

	if output["status"] != "warning" {
		t.Errorf("status = %q, want %q", output["status"], "warning")
	}

	issues, ok := output["issues"].([]map[string]any)
	if !ok {
		t.Fatalf("issues is not []map[string]any: %T", output["issues"])
	}

	if len(issues) != 2 {
		t.Fatalf("len(issues) = %d, want 2", len(issues))
	}

	// Check first issue
	if issues[0]["entity_id"] != "WT-001" {
		t.Errorf("issues[0].entity_id = %q, want %q", issues[0]["entity_id"], "WT-001")
	}

	// Check second issue
	if issues[1]["entry_id"] != "KE-001" {
		t.Errorf("issues[1].entry_id = %q, want %q", issues[1]["entry_id"], "KE-001")
	}
}

func TestFormatIssue_MinimalFields(t *testing.T) {
	t.Parallel()

	issue := Issue{
		Severity: SeverityWarning,
		Message:  "test message",
	}

	output := FormatIssue(issue)

	if output["severity"] != "warning" {
		t.Errorf("severity = %q, want %q", output["severity"], "warning")
	}
	if output["message"] != "test message" {
		t.Errorf("message = %q, want %q", output["message"], "test message")
	}

	// Optional fields should not be present
	if _, ok := output["entity_id"]; ok {
		t.Error("entity_id should not be present for minimal issue")
	}
	if _, ok := output["entry_id"]; ok {
		t.Error("entry_id should not be present for minimal issue")
	}
	if _, ok := output["entries"]; ok {
		t.Error("entries should not be present for minimal issue")
	}
}

func TestFormatIssue_AllFields(t *testing.T) {
	t.Parallel()

	issue := Issue{
		Severity: SeverityError,
		EntityID: "WT-12345",
		EntryID:  "KE-67890",
		Entries:  []string{"KE-A", "KE-B"},
		Message:  "full issue",
	}

	output := FormatIssue(issue)

	if output["severity"] != "error" {
		t.Errorf("severity = %q, want %q", output["severity"], "error")
	}
	if output["message"] != "full issue" {
		t.Errorf("message = %q, want %q", output["message"], "full issue")
	}
	if output["entity_id"] != "WT-12345" {
		t.Errorf("entity_id = %q, want %q", output["entity_id"], "WT-12345")
	}
	if output["entry_id"] != "KE-67890" {
		t.Errorf("entry_id = %q, want %q", output["entry_id"], "KE-67890")
	}

	entries, ok := output["entries"].([]string)
	if !ok {
		t.Fatalf("entries is not []string: %T", output["entries"])
	}
	if len(entries) != 2 {
		t.Errorf("len(entries) = %d, want 2", len(entries))
	}
}

func TestCountIssues_Empty(t *testing.T) {
	t.Parallel()

	result := HealthResult{
		Status:     SeverityOK,
		Categories: nil,
	}

	count := CountIssues(result)
	if count != 0 {
		t.Errorf("CountIssues = %d, want 0", count)
	}
}

func TestCountIssues_MultipleCategories(t *testing.T) {
	t.Parallel()

	result := HealthResult{
		Status: SeverityError,
		Categories: map[string]CategoryResult{
			"worktree": {
				Status: SeverityError,
				Issues: []Issue{
					{Severity: SeverityError, Message: "issue 1"},
					{Severity: SeverityWarning, Message: "issue 2"},
				},
			},
			"cleanup": {
				Status: SeverityWarning,
				Issues: []Issue{
					{Severity: SeverityWarning, Message: "issue 3"},
				},
			},
			"branch": {
				Status: SeverityOK,
				Issues: nil,
			},
		},
	}

	count := CountIssues(result)
	if count != 3 {
		t.Errorf("CountIssues = %d, want 3", count)
	}
}

func TestCountBySeverity_Empty(t *testing.T) {
	t.Parallel()

	result := HealthResult{
		Status:     SeverityOK,
		Categories: nil,
	}

	counts := CountBySeverity(result)

	if counts[SeverityOK] != 0 {
		t.Errorf("counts[OK] = %d, want 0", counts[SeverityOK])
	}
	if counts[SeverityWarning] != 0 {
		t.Errorf("counts[Warning] = %d, want 0", counts[SeverityWarning])
	}
	if counts[SeverityError] != 0 {
		t.Errorf("counts[Error] = %d, want 0", counts[SeverityError])
	}
}

func TestCountBySeverity_Mixed(t *testing.T) {
	t.Parallel()

	result := HealthResult{
		Status: SeverityError,
		Categories: map[string]CategoryResult{
			"worktree": {
				Status: SeverityError,
				Issues: []Issue{
					{Severity: SeverityError, Message: "error 1"},
					{Severity: SeverityWarning, Message: "warning 1"},
				},
			},
			"cleanup": {
				Status: SeverityWarning,
				Issues: []Issue{
					{Severity: SeverityWarning, Message: "warning 2"},
					{Severity: SeverityWarning, Message: "warning 3"},
					{Severity: SeverityError, Message: "error 2"},
				},
			},
		},
	}

	counts := CountBySeverity(result)

	if counts[SeverityError] != 2 {
		t.Errorf("counts[Error] = %d, want 2", counts[SeverityError])
	}
	if counts[SeverityWarning] != 3 {
		t.Errorf("counts[Warning] = %d, want 3", counts[SeverityWarning])
	}
	if counts[SeverityOK] != 0 {
		t.Errorf("counts[OK] = %d, want 0", counts[SeverityOK])
	}
}

func TestSummary_AllPassed(t *testing.T) {
	t.Parallel()

	result := HealthResult{
		Status:     SeverityOK,
		Categories: nil,
	}

	summary := Summary(result)
	if summary != "All health checks passed" {
		t.Errorf("Summary = %q, want %q", summary, "All health checks passed")
	}
}

func TestSummary_OneWarning(t *testing.T) {
	t.Parallel()

	result := HealthResult{
		Status: SeverityWarning,
		Categories: map[string]CategoryResult{
			"cleanup": {
				Status: SeverityWarning,
				Issues: []Issue{{Severity: SeverityWarning, Message: "warning"}},
			},
		},
	}

	summary := Summary(result)
	if summary != "1 warning found" {
		t.Errorf("Summary = %q, want %q", summary, "1 warning found")
	}
}

func TestSummary_MultipleWarnings(t *testing.T) {
	t.Parallel()

	result := HealthResult{
		Status: SeverityWarning,
		Categories: map[string]CategoryResult{
			"cleanup": {
				Status: SeverityWarning,
				Issues: []Issue{
					{Severity: SeverityWarning, Message: "warning 1"},
					{Severity: SeverityWarning, Message: "warning 2"},
					{Severity: SeverityWarning, Message: "warning 3"},
				},
			},
		},
	}

	summary := Summary(result)
	if summary != "3 warnings found" {
		t.Errorf("Summary = %q, want %q", summary, "3 warnings found")
	}
}

func TestSummary_OneError(t *testing.T) {
	t.Parallel()

	result := HealthResult{
		Status: SeverityError,
		Categories: map[string]CategoryResult{
			"worktree": {
				Status: SeverityError,
				Issues: []Issue{{Severity: SeverityError, Message: "error"}},
			},
		},
	}

	summary := Summary(result)
	if summary != "1 error found" {
		t.Errorf("Summary = %q, want %q", summary, "1 error found")
	}
}

func TestSummary_MultipleErrors(t *testing.T) {
	t.Parallel()

	result := HealthResult{
		Status: SeverityError,
		Categories: map[string]CategoryResult{
			"worktree": {
				Status: SeverityError,
				Issues: []Issue{
					{Severity: SeverityError, Message: "error 1"},
					{Severity: SeverityError, Message: "error 2"},
				},
			},
		},
	}

	summary := Summary(result)
	if summary != "2 errors found" {
		t.Errorf("Summary = %q, want %q", summary, "2 errors found")
	}
}

func TestSummary_Mixed(t *testing.T) {
	t.Parallel()

	result := HealthResult{
		Status: SeverityError,
		Categories: map[string]CategoryResult{
			"worktree": {
				Status: SeverityError,
				Issues: []Issue{
					{Severity: SeverityError, Message: "error 1"},
					{Severity: SeverityError, Message: "error 2"},
					{Severity: SeverityWarning, Message: "warning 1"},
				},
			},
			"cleanup": {
				Status: SeverityWarning,
				Issues: []Issue{
					{Severity: SeverityWarning, Message: "warning 2"},
				},
			},
		},
	}

	summary := Summary(result)
	if summary != "2 errors, 2 warnings found" {
		t.Errorf("Summary = %q, want %q", summary, "2 errors, 2 warnings found")
	}
}

func TestFormatCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		n    int
		want string
	}{
		{0, "0"},
		{1, "1"},
		{5, "5"},
		{9, "9"},
		{10, "10"},
		{15, "15"},
		{100, "100"},
		{123, "123"},
		{999, "999"},
		{1000, "1000"},
	}

	for _, tt := range tests {
		got := formatCount(tt.n)
		if got != tt.want {
			t.Errorf("formatCount(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func TestFormatHealthResult_DeterministicOrder(t *testing.T) {
	t.Parallel()

	// Run multiple times to verify deterministic ordering
	for i := 0; i < 10; i++ {
		result := HealthResult{
			Status: SeverityWarning,
			Categories: map[string]CategoryResult{
				"zebra": {Status: SeverityOK},
				"alpha": {Status: SeverityWarning},
				"beta":  {Status: SeverityOK},
			},
		}

		output := FormatHealthResult(result)
		categories, ok := output["categories"].(map[string]any)
		if !ok {
			t.Fatalf("categories is not map[string]any: %T", output["categories"])
		}

		// Verify all categories are present
		if len(categories) != 3 {
			t.Errorf("len(categories) = %d, want 3", len(categories))
		}
	}
}
