package merge

import (
	"testing"
	"time"
)

func TestFormatGateResults(t *testing.T) {
	result := GateCheckResult{
		EntityID:      "FEAT-001",
		Branch:        "feature/FEAT-001",
		OverallStatus: OverallStatusWarnings,
		Gates: []GateResult{
			{Name: "tasks_complete", Status: GateStatusPassed, Severity: GateSeverityBlocking},
			{Name: "verification_exists", Status: GateStatusPassed, Severity: GateSeverityBlocking},
			{Name: "branch_not_stale", Status: GateStatusWarning, Severity: GateSeverityWarning, Message: "branch is stale"},
		},
	}

	output := FormatGateResults(result)

	if output["entity_id"] != "FEAT-001" {
		t.Errorf("entity_id: got %q, want %q", output["entity_id"], "FEAT-001")
	}
	if output["branch"] != "feature/FEAT-001" {
		t.Errorf("branch: got %q, want %q", output["branch"], "feature/FEAT-001")
	}
	if output["overall_status"] != OverallStatusWarnings {
		t.Errorf("overall_status: got %q, want %q", output["overall_status"], OverallStatusWarnings)
	}

	gates, ok := output["gates"].([]map[string]any)
	if !ok {
		t.Fatalf("gates is not []map[string]any")
	}
	if len(gates) != 3 {
		t.Fatalf("expected 3 gates, got %d", len(gates))
	}

	// Check first gate
	if gates[0]["name"] != "tasks_complete" {
		t.Errorf("gates[0].name: got %q, want %q", gates[0]["name"], "tasks_complete")
	}
	if gates[0]["status"] != "passed" {
		t.Errorf("gates[0].status: got %q, want %q", gates[0]["status"], "passed")
	}
	if gates[0]["severity"] != "blocking" {
		t.Errorf("gates[0].severity: got %q, want %q", gates[0]["severity"], "blocking")
	}
	if _, hasMsg := gates[0]["message"]; hasMsg {
		t.Error("gates[0] should not have message when passed")
	}

	// Check gate with message
	if gates[2]["message"] != "branch is stale" {
		t.Errorf("gates[2].message: got %q, want %q", gates[2]["message"], "branch is stale")
	}

	// Check summary
	summary, ok := output["summary"].(map[string]any)
	if !ok {
		t.Fatalf("summary is not map[string]any")
	}
	if summary["total"] != 3 {
		t.Errorf("summary.total: got %v, want 3", summary["total"])
	}
	if summary["passed"] != 2 {
		t.Errorf("summary.passed: got %v, want 2", summary["passed"])
	}
	if summary["failed"] != 0 {
		t.Errorf("summary.failed: got %v, want 0", summary["failed"])
	}
	if summary["warning"] != 1 {
		t.Errorf("summary.warning: got %v, want 1", summary["warning"])
	}
}

func TestFormatGateResults_EmptyGates(t *testing.T) {
	result := GateCheckResult{
		EntityID:      "FEAT-001",
		Branch:        "",
		OverallStatus: OverallStatusPassed,
		Gates:         []GateResult{},
	}

	output := FormatGateResults(result)

	gates, ok := output["gates"].([]map[string]any)
	if !ok {
		t.Fatalf("gates is not []map[string]any")
	}
	if len(gates) != 0 {
		t.Errorf("expected 0 gates, got %d", len(gates))
	}

	summary := output["summary"].(map[string]any)
	if summary["total"] != 0 {
		t.Errorf("summary.total: got %v, want 0", summary["total"])
	}
}

func TestFormatGateResultsCompact(t *testing.T) {
	result := GateCheckResult{
		EntityID:      "FEAT-001",
		Branch:        "feature/FEAT-001",
		OverallStatus: OverallStatusBlocked,
		Gates: []GateResult{
			{Name: "tasks_complete", Status: GateStatusPassed, Severity: GateSeverityBlocking},
			{Name: "verification_exists", Status: GateStatusFailed, Severity: GateSeverityBlocking, Message: "verification field is empty"},
			{Name: "branch_not_stale", Status: GateStatusWarning, Severity: GateSeverityWarning, Message: "branch is stale"},
		},
	}

	output := FormatGateResultsCompact(result)

	if output["entity_id"] != "FEAT-001" {
		t.Errorf("entity_id: got %q, want %q", output["entity_id"], "FEAT-001")
	}
	if output["status"] != OverallStatusBlocked {
		t.Errorf("status: got %q, want %q", output["status"], OverallStatusBlocked)
	}

	// Should not have branch in compact format
	if _, hasBranch := output["branch"]; hasBranch {
		t.Error("compact format should not include branch")
	}

	issues, ok := output["issues"].([]map[string]any)
	if !ok {
		t.Fatalf("issues is not []map[string]any")
	}
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues (failed + warning), got %d", len(issues))
	}

	// Check failed gate
	if issues[0]["gate"] != "verification_exists" {
		t.Errorf("issues[0].gate: got %q, want %q", issues[0]["gate"], "verification_exists")
	}
	if issues[0]["status"] != "failed" {
		t.Errorf("issues[0].status: got %q, want %q", issues[0]["status"], "failed")
	}
	if issues[0]["severity"] != "blocking" {
		t.Errorf("issues[0].severity: got %q, want %q", issues[0]["severity"], "blocking")
	}
	if issues[0]["message"] != "verification field is empty" {
		t.Errorf("issues[0].message: got %q, want %q", issues[0]["message"], "verification field is empty")
	}

	// Check warning gate
	if issues[1]["gate"] != "branch_not_stale" {
		t.Errorf("issues[1].gate: got %q, want %q", issues[1]["gate"], "branch_not_stale")
	}
}

func TestFormatGateResultsCompact_NoIssues(t *testing.T) {
	result := GateCheckResult{
		EntityID:      "FEAT-001",
		OverallStatus: OverallStatusPassed,
		Gates: []GateResult{
			{Name: "tasks_complete", Status: GateStatusPassed, Severity: GateSeverityBlocking},
		},
	}

	output := FormatGateResultsCompact(result)

	if _, hasIssues := output["issues"]; hasIssues {
		t.Error("compact format should not include issues when all passed")
	}
}

func TestFormatOverrides(t *testing.T) {
	overrides := []Override{
		{
			Gate:         "tasks_complete",
			Reason:       "Emergency deployment",
			OverriddenBy: "admin",
			OverriddenAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			Gate:         "no_conflicts",
			Reason:       "Conflicts resolved manually",
			OverriddenBy: "alice",
			OverriddenAt: time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC),
		},
	}

	output := FormatOverrides(overrides)

	if len(output) != 2 {
		t.Fatalf("expected 2 overrides, got %d", len(output))
	}

	// Check first override
	if output[0]["gate"] != "tasks_complete" {
		t.Errorf("output[0].gate: got %q, want %q", output[0]["gate"], "tasks_complete")
	}
	if output[0]["reason"] != "Emergency deployment" {
		t.Errorf("output[0].reason: got %q, want %q", output[0]["reason"], "Emergency deployment")
	}
	if output[0]["overridden_by"] != "admin" {
		t.Errorf("output[0].overridden_by: got %q, want %q", output[0]["overridden_by"], "admin")
	}

	at, ok := output[0]["overridden_at"].(time.Time)
	if !ok {
		t.Fatalf("output[0].overridden_at is not time.Time")
	}
	expectedTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	if !at.Equal(expectedTime) {
		t.Errorf("output[0].overridden_at: got %v, want %v", at, expectedTime)
	}

	// Check second override
	if output[1]["gate"] != "no_conflicts" {
		t.Errorf("output[1].gate: got %q, want %q", output[1]["gate"], "no_conflicts")
	}
}

func TestFormatOverrides_Empty(t *testing.T) {
	output := FormatOverrides([]Override{})
	if output != nil {
		t.Errorf("expected nil for empty overrides, got %v", output)
	}

	output = FormatOverrides(nil)
	if output != nil {
		t.Errorf("expected nil for nil overrides, got %v", output)
	}
}

func TestFormatGateResults_AllStatuses(t *testing.T) {
	result := GateCheckResult{
		EntityID:      "FEAT-001",
		Branch:        "feature/test",
		OverallStatus: OverallStatusBlocked,
		Gates: []GateResult{
			{Name: "g1", Status: GateStatusPassed, Severity: GateSeverityBlocking},
			{Name: "g2", Status: GateStatusFailed, Severity: GateSeverityBlocking, Message: "failed"},
			{Name: "g3", Status: GateStatusWarning, Severity: GateSeverityWarning, Message: "warning"},
		},
	}

	output := FormatGateResults(result)
	summary := output["summary"].(map[string]any)

	if summary["passed"] != 1 {
		t.Errorf("summary.passed: got %v, want 1", summary["passed"])
	}
	if summary["failed"] != 1 {
		t.Errorf("summary.failed: got %v, want 1", summary["failed"])
	}
	if summary["warning"] != 1 {
		t.Errorf("summary.warning: got %v, want 1", summary["warning"])
	}
}
