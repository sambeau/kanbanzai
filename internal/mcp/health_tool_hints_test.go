package mcp

import (
	"strings"
	"testing"
)

// TestToolHintsHealthChecker_WithHints verifies that configured hints appear
// in health output (AC-017).
func TestToolHintsHealthChecker_WithHints(t *testing.T) {
	checker := ToolHintsHealthChecker(map[string]string{
		"implementer-go": "Use search_graph for code navigation",
		"reviewer":       "Read the tracing skill before reviewing",
	})
	report, err := checker()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(report.Warnings) != 2 {
		t.Fatalf("expected 2 warnings, got %d", len(report.Warnings))
	}
	// Check that both roles appear.
	var foundImpl, foundRev bool
	for _, w := range report.Warnings {
		if w.EntityType != "tool_hints" {
			t.Errorf("expected entity type tool_hints, got %q", w.EntityType)
		}
		if strings.Contains(w.Message, "implementer-go") {
			foundImpl = true
		}
		if strings.Contains(w.Message, "reviewer") {
			foundRev = true
		}
	}
	if !foundImpl {
		t.Error("expected implementer-go hint in health output")
	}
	if !foundRev {
		t.Error("expected reviewer hint in health output")
	}
}

// TestToolHintsHealthChecker_NoHints verifies the empty case (AC-018).
func TestToolHintsHealthChecker_NoHints(t *testing.T) {
	checker := ToolHintsHealthChecker(nil)
	report, err := checker()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(report.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(report.Warnings))
	}
	w := report.Warnings[0]
	if w.EntityType != "tool_hints" {
		t.Errorf("expected entity type tool_hints, got %q", w.EntityType)
	}
	if !strings.Contains(w.Message, "No tool hints configured") {
		t.Errorf("expected 'No tool hints configured' message, got %q", w.Message)
	}
}

// TestToolHintsHealthChecker_LongHintTruncated verifies truncation of long hints.
func TestToolHintsHealthChecker_LongHintTruncated(t *testing.T) {
	longHint := strings.Repeat("x", 100)
	checker := ToolHintsHealthChecker(map[string]string{"test-role": longHint})
	report, err := checker()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(report.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(report.Warnings))
	}
	if !strings.HasSuffix(report.Warnings[0].Message, "...") {
		t.Errorf("expected truncated hint ending with ..., got %q", report.Warnings[0].Message)
	}
}
