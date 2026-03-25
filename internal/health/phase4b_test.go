package health

import (
	"testing"
	"time"
)

func TestCheckUnlinkedResolvedIncidents_FlagsOldResolvedWithoutRCA(t *testing.T) {
	t.Parallel()

	now := time.Now()
	incidents := []map[string]any{
		{
			"id":          "INC-ABC123",
			"status":      "resolved",
			"resolved_at": now.Add(-10 * 24 * time.Hour).Format(time.RFC3339),
		},
	}

	result := CheckUnlinkedResolvedIncidents(incidents, 7, now)

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
	if issue.EntityID != "INC-ABC123" {
		t.Errorf("Issue.EntityID = %q, want %q", issue.EntityID, "INC-ABC123")
	}
}

func TestCheckUnlinkedResolvedIncidents_FlagsOldRootCauseIdentifiedWithoutRCA(t *testing.T) {
	t.Parallel()

	now := time.Now()
	incidents := []map[string]any{
		{
			"id":      "INC-DEF456",
			"status":  "root-cause-identified",
			"updated": now.Add(-10 * 24 * time.Hour).Format(time.RFC3339),
		},
	}

	result := CheckUnlinkedResolvedIncidents(incidents, 7, now)

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
	if issue.EntityID != "INC-DEF456" {
		t.Errorf("Issue.EntityID = %q, want %q", issue.EntityID, "INC-DEF456")
	}
}

func TestCheckUnlinkedResolvedIncidents_NoFlagBeforeThreshold(t *testing.T) {
	t.Parallel()

	now := time.Now()
	incidents := []map[string]any{
		{
			"id":          "INC-GHI789",
			"status":      "resolved",
			"resolved_at": now.Add(-3 * 24 * time.Hour).Format(time.RFC3339),
		},
	}

	result := CheckUnlinkedResolvedIncidents(incidents, 7, now)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
	}
}

func TestCheckUnlinkedResolvedIncidents_NoFlagWithLinkedRCA(t *testing.T) {
	t.Parallel()

	now := time.Now()
	incidents := []map[string]any{
		{
			"id":          "INC-JKL012",
			"status":      "resolved",
			"resolved_at": now.Add(-30 * 24 * time.Hour).Format(time.RFC3339),
			"linked_rca":  "DOC-ABC123",
		},
	}

	result := CheckUnlinkedResolvedIncidents(incidents, 7, now)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
	}
}

func TestCheckUnlinkedResolvedIncidents_DisabledWhenZero(t *testing.T) {
	t.Parallel()

	now := time.Now()
	incidents := []map[string]any{
		{
			"id":          "INC-MNO345",
			"status":      "resolved",
			"resolved_at": now.Add(-30 * 24 * time.Hour).Format(time.RFC3339),
		},
	}

	result := CheckUnlinkedResolvedIncidents(incidents, 0, now)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
	}
}

func TestCheckUnlinkedResolvedIncidents_SkipsNonResolvedStatuses(t *testing.T) {
	t.Parallel()

	now := time.Now()
	incidents := []map[string]any{
		{
			"id":      "INC-PQR678",
			"status":  "reported",
			"updated": now.Add(-30 * 24 * time.Hour).Format(time.RFC3339),
		},
		{
			"id":      "INC-STU901",
			"status":  "triaged",
			"updated": now.Add(-30 * 24 * time.Hour).Format(time.RFC3339),
		},
	}

	result := CheckUnlinkedResolvedIncidents(incidents, 7, now)

	if result.Status != SeverityOK {
		t.Errorf("Status = %v, want %v", result.Status, SeverityOK)
	}
	if len(result.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(result.Issues))
	}
}
