package health

import (
	"testing"
)

func TestSeverityConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		severity Severity
		want     string
	}{
		{SeverityOK, "ok"},
		{SeverityWarning, "warning"},
		{SeverityError, "error"},
	}

	for _, tt := range tests {
		if string(tt.severity) != tt.want {
			t.Errorf("Severity %v = %q, want %q", tt.severity, string(tt.severity), tt.want)
		}
	}
}

func TestWorstSeverity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a    Severity
		b    Severity
		want Severity
	}{
		{"ok_ok", SeverityOK, SeverityOK, SeverityOK},
		{"ok_warning", SeverityOK, SeverityWarning, SeverityWarning},
		{"ok_error", SeverityOK, SeverityError, SeverityError},
		{"warning_ok", SeverityWarning, SeverityOK, SeverityWarning},
		{"warning_warning", SeverityWarning, SeverityWarning, SeverityWarning},
		{"warning_error", SeverityWarning, SeverityError, SeverityError},
		{"error_ok", SeverityError, SeverityOK, SeverityError},
		{"error_warning", SeverityError, SeverityWarning, SeverityError},
		{"error_error", SeverityError, SeverityError, SeverityError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WorstSeverity(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("WorstSeverity(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestNewCategoryResult(t *testing.T) {
	t.Parallel()

	result := NewCategoryResult()

	if result.Status != SeverityOK {
		t.Errorf("NewCategoryResult().Status = %v, want %v", result.Status, SeverityOK)
	}

	if result.Issues != nil {
		t.Errorf("NewCategoryResult().Issues = %v, want nil", result.Issues)
	}
}

func TestCategoryResult_AddIssue(t *testing.T) {
	t.Parallel()

	t.Run("adds_first_issue", func(t *testing.T) {
		result := NewCategoryResult()

		issue := Issue{
			Severity: SeverityWarning,
			Message:  "test warning",
		}
		result.AddIssue(issue)

		if len(result.Issues) != 1 {
			t.Fatalf("len(Issues) = %d, want 1", len(result.Issues))
		}

		if result.Issues[0].Message != "test warning" {
			t.Errorf("Issues[0].Message = %q, want %q", result.Issues[0].Message, "test warning")
		}

		if result.Status != SeverityWarning {
			t.Errorf("Status = %v, want %v", result.Status, SeverityWarning)
		}
	})

	t.Run("updates_status_to_worst", func(t *testing.T) {
		result := NewCategoryResult()

		// Add warning first
		result.AddIssue(Issue{Severity: SeverityWarning, Message: "warning"})
		if result.Status != SeverityWarning {
			t.Errorf("Status after warning = %v, want %v", result.Status, SeverityWarning)
		}

		// Add error - should upgrade
		result.AddIssue(Issue{Severity: SeverityError, Message: "error"})
		if result.Status != SeverityError {
			t.Errorf("Status after error = %v, want %v", result.Status, SeverityError)
		}

		// Add another warning - should stay error
		result.AddIssue(Issue{Severity: SeverityWarning, Message: "another warning"})
		if result.Status != SeverityError {
			t.Errorf("Status after second warning = %v, want %v", result.Status, SeverityError)
		}

		if len(result.Issues) != 3 {
			t.Errorf("len(Issues) = %d, want 3", len(result.Issues))
		}
	})

	t.Run("preserves_issue_fields", func(t *testing.T) {
		result := NewCategoryResult()

		issue := Issue{
			Severity: SeverityError,
			EntityID: "WT-12345",
			EntryID:  "KE-67890",
			Entries:  []string{"KE-A", "KE-B"},
			Message:  "full issue",
		}
		result.AddIssue(issue)

		got := result.Issues[0]
		if got.Severity != SeverityError {
			t.Errorf("Severity = %v, want %v", got.Severity, SeverityError)
		}
		if got.EntityID != "WT-12345" {
			t.Errorf("EntityID = %q, want %q", got.EntityID, "WT-12345")
		}
		if got.EntryID != "KE-67890" {
			t.Errorf("EntryID = %q, want %q", got.EntryID, "KE-67890")
		}
		if len(got.Entries) != 2 {
			t.Errorf("len(Entries) = %d, want 2", len(got.Entries))
		}
		if got.Message != "full issue" {
			t.Errorf("Message = %q, want %q", got.Message, "full issue")
		}
	})
}

func TestIssue_Fields(t *testing.T) {
	t.Parallel()

	// Test that Issue struct works correctly with all fields
	issue := Issue{
		Severity: SeverityWarning,
		EntityID: "FEAT-123",
		EntryID:  "",
		Entries:  nil,
		Message:  "test message",
	}

	if issue.Severity != SeverityWarning {
		t.Errorf("Severity = %v, want %v", issue.Severity, SeverityWarning)
	}
	if issue.EntityID != "FEAT-123" {
		t.Errorf("EntityID = %q, want %q", issue.EntityID, "FEAT-123")
	}
	if issue.EntryID != "" {
		t.Errorf("EntryID = %q, want empty", issue.EntryID)
	}
	if issue.Entries != nil {
		t.Errorf("Entries = %v, want nil", issue.Entries)
	}
	if issue.Message != "test message" {
		t.Errorf("Message = %q, want %q", issue.Message, "test message")
	}
}
