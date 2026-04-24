package merge

import (
	"testing"

	"github.com/sambeau/kanbanzai/internal/git"
)

func TestDefaultGates(t *testing.T) {
	gates := DefaultGates()

	if len(gates) != 8 {
		t.Errorf("DefaultGates: got %d gates, want 8", len(gates))
	}

	// Verify order and names
	expectedNames := []string{
		"review_report_exists",
		"entity_done",
		"tasks_complete",
		"verification_exists",
		"verification_passed",
		"no_conflicts",
		"health_check_clean",
		"branch_not_stale",
	}

	for i, name := range expectedNames {
		if i >= len(gates) {
			break
		}
		if gates[i].Name() != name {
			t.Errorf("DefaultGates[%d]: got %q, want %q", i, gates[i].Name(), name)
		}
	}
}

func TestCheckGates_AllPassing(t *testing.T) {
	ctx := GateContext{
		EntityID: "FEAT-001",
		Branch:   "feature/FEAT-001",
		RepoPath: "/repo",
		Entity: map[string]any{
			"status":              "done",
			"verification":        "All tests pass",
			"verification_status": "passed",
		},
		Tasks: []map[string]any{
			{"id": "TASK-001", "status": "done"},
		},
		ConflictChecker: func(repoPath, branch, base string) (bool, error) {
			return false, nil
		},
		BranchStatusChecker: func(repoPath, branch string, thresholds git.BranchThresholds) (git.BranchStatus, error) {
			return git.BranchStatus{}, nil
		},
		DefaultBranchDetector: func(repoPath string) (string, error) {
			return "main", nil
		},
	}

	result := CheckGates(ctx)

	if result.EntityID != "FEAT-001" {
		t.Errorf("EntityID: got %q, want %q", result.EntityID, "FEAT-001")
	}
	if result.Branch != "feature/FEAT-001" {
		t.Errorf("Branch: got %q, want %q", result.Branch, "feature/FEAT-001")
	}
	if result.OverallStatus != OverallStatusPassed {
		t.Errorf("OverallStatus: got %q, want %q", result.OverallStatus, OverallStatusPassed)
	}
	if len(result.Gates) != 8 {
		t.Errorf("Gates: got %d, want 8", len(result.Gates))
	}

	for _, g := range result.Gates {
		if g.Status != GateStatusPassed {
			t.Errorf("Gate %q: got status %q, want %q", g.Name, g.Status, GateStatusPassed)
		}
	}
}

func TestCheckGates_BlockingFailure(t *testing.T) {
	ctx := GateContext{
		EntityID: "FEAT-001",
		Branch:   "feature/FEAT-001",
		RepoPath: "/repo",
		Entity: map[string]any{
			"status":              "done",
			"verification":        "", // Missing verification
			"verification_status": "passed",
		},
		Tasks: []map[string]any{
			{"id": "TASK-001", "status": "done"},
		},
		ConflictChecker: func(repoPath, branch, base string) (bool, error) {
			return false, nil
		},
		BranchStatusChecker: func(repoPath, branch string, thresholds git.BranchThresholds) (git.BranchStatus, error) {
			return git.BranchStatus{}, nil
		},
		DefaultBranchDetector: func(repoPath string) (string, error) {
			return "main", nil
		},
	}

	result := CheckGates(ctx)

	if result.OverallStatus != OverallStatusBlocked {
		t.Errorf("OverallStatus: got %q, want %q", result.OverallStatus, OverallStatusBlocked)
	}

	// Find the verification_exists gate
	var found bool
	for _, g := range result.Gates {
		if g.Name == "verification_exists" {
			found = true
			if g.Status != GateStatusFailed {
				t.Errorf("verification_exists status: got %q, want %q", g.Status, GateStatusFailed)
			}
		}
	}
	if !found {
		t.Error("verification_exists gate not found in results")
	}
}

func TestCheckGates_WarningsOnly(t *testing.T) {
	ctx := GateContext{
		EntityID: "FEAT-001",
		Branch:   "feature/FEAT-001",
		RepoPath: "/repo",
		Entity: map[string]any{
			"status":              "done",
			"verification":        "Tests pass",
			"verification_status": "passed",
		},
		Tasks: []map[string]any{
			{"id": "TASK-001", "status": "done"},
		},
		ConflictChecker: func(repoPath, branch, base string) (bool, error) {
			return false, nil
		},
		BranchStatusChecker: func(repoPath, branch string, thresholds git.BranchThresholds) (git.BranchStatus, error) {
			return git.BranchStatus{
				Warnings: []string{"branch is stale"},
			}, nil
		},
		DefaultBranchDetector: func(repoPath string) (string, error) {
			return "main", nil
		},
	}

	result := CheckGates(ctx)

	if result.OverallStatus != OverallStatusWarnings {
		t.Errorf("OverallStatus: got %q, want %q", result.OverallStatus, OverallStatusWarnings)
	}
}

func TestCheckGatesWithList_CustomGates(t *testing.T) {
	ctx := GateContext{
		EntityID: "FEAT-001",
		Entity: map[string]any{
			"verification": "Done",
		},
	}

	// Only run verification_exists gate
	gates := []Gate{VerificationExistsGate{}}
	result := CheckGatesWithList(ctx, gates)

	if len(result.Gates) != 1 {
		t.Errorf("Gates: got %d, want 1", len(result.Gates))
	}
	if result.Gates[0].Name != "verification_exists" {
		t.Errorf("Gate name: got %q, want %q", result.Gates[0].Name, "verification_exists")
	}
	if result.OverallStatus != OverallStatusPassed {
		t.Errorf("OverallStatus: got %q, want %q", result.OverallStatus, OverallStatusPassed)
	}
}

func TestCheckGatesWithList_EmptyGates(t *testing.T) {
	ctx := GateContext{
		EntityID: "FEAT-001",
	}

	result := CheckGatesWithList(ctx, []Gate{})

	if len(result.Gates) != 0 {
		t.Errorf("Gates: got %d, want 0", len(result.Gates))
	}
	if result.OverallStatus != OverallStatusPassed {
		t.Errorf("OverallStatus: got %q, want %q", result.OverallStatus, OverallStatusPassed)
	}
}

func TestDetermineOverallStatus(t *testing.T) {
	tests := []struct {
		name    string
		results []GateResult
		want    string
	}{
		{
			name:    "empty results passes",
			results: []GateResult{},
			want:    OverallStatusPassed,
		},
		{
			name: "all passed",
			results: []GateResult{
				{Name: "g1", Status: GateStatusPassed, Severity: GateSeverityBlocking},
				{Name: "g2", Status: GateStatusPassed, Severity: GateSeverityWarning},
			},
			want: OverallStatusPassed,
		},
		{
			name: "blocking failure blocks",
			results: []GateResult{
				{Name: "g1", Status: GateStatusPassed, Severity: GateSeverityBlocking},
				{Name: "g2", Status: GateStatusFailed, Severity: GateSeverityBlocking},
			},
			want: OverallStatusBlocked,
		},
		{
			name: "warning failure is warning",
			results: []GateResult{
				{Name: "g1", Status: GateStatusPassed, Severity: GateSeverityBlocking},
				{Name: "g2", Status: GateStatusFailed, Severity: GateSeverityWarning},
			},
			want: OverallStatusWarnings,
		},
		{
			name: "warning status is warning",
			results: []GateResult{
				{Name: "g1", Status: GateStatusPassed, Severity: GateSeverityBlocking},
				{Name: "g2", Status: GateStatusWarning, Severity: GateSeverityWarning},
			},
			want: OverallStatusWarnings,
		},
		{
			name: "blocking beats warning",
			results: []GateResult{
				{Name: "g1", Status: GateStatusWarning, Severity: GateSeverityWarning},
				{Name: "g2", Status: GateStatusFailed, Severity: GateSeverityBlocking},
			},
			want: OverallStatusBlocked,
		},
		{
			name: "multiple blocking failures still blocked",
			results: []GateResult{
				{Name: "g1", Status: GateStatusFailed, Severity: GateSeverityBlocking},
				{Name: "g2", Status: GateStatusFailed, Severity: GateSeverityBlocking},
			},
			want: OverallStatusBlocked,
		},
		{
			name: "multiple warnings",
			results: []GateResult{
				{Name: "g1", Status: GateStatusWarning, Severity: GateSeverityWarning},
				{Name: "g2", Status: GateStatusWarning, Severity: GateSeverityWarning},
			},
			want: OverallStatusWarnings,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetermineOverallStatus(tt.results)
			if got != tt.want {
				t.Errorf("DetermineOverallStatus: got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCountByStatus(t *testing.T) {
	tests := []struct {
		name        string
		results     []GateResult
		wantPassed  int
		wantFailed  int
		wantWarning int
	}{
		{
			name:        "empty",
			results:     []GateResult{},
			wantPassed:  0,
			wantFailed:  0,
			wantWarning: 0,
		},
		{
			name: "all passed",
			results: []GateResult{
				{Status: GateStatusPassed},
				{Status: GateStatusPassed},
			},
			wantPassed:  2,
			wantFailed:  0,
			wantWarning: 0,
		},
		{
			name: "mixed",
			results: []GateResult{
				{Status: GateStatusPassed},
				{Status: GateStatusFailed},
				{Status: GateStatusWarning},
				{Status: GateStatusPassed},
			},
			wantPassed:  2,
			wantFailed:  1,
			wantWarning: 1,
		},
		{
			name: "all failed",
			results: []GateResult{
				{Status: GateStatusFailed},
				{Status: GateStatusFailed},
			},
			wantPassed:  0,
			wantFailed:  2,
			wantWarning: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed, failed, warning := CountByStatus(tt.results)
			if passed != tt.wantPassed {
				t.Errorf("passed: got %d, want %d", passed, tt.wantPassed)
			}
			if failed != tt.wantFailed {
				t.Errorf("failed: got %d, want %d", failed, tt.wantFailed)
			}
			if warning != tt.wantWarning {
				t.Errorf("warning: got %d, want %d", warning, tt.wantWarning)
			}
		})
	}
}

func TestBlockingFailures(t *testing.T) {
	tests := []struct {
		name    string
		results []GateResult
		want    int
	}{
		{
			name:    "empty",
			results: []GateResult{},
			want:    0,
		},
		{
			name: "no failures",
			results: []GateResult{
				{Name: "g1", Status: GateStatusPassed, Severity: GateSeverityBlocking},
				{Name: "g2", Status: GateStatusWarning, Severity: GateSeverityWarning},
			},
			want: 0,
		},
		{
			name: "warning failure not blocking",
			results: []GateResult{
				{Name: "g1", Status: GateStatusFailed, Severity: GateSeverityWarning},
			},
			want: 0,
		},
		{
			name: "blocking failure",
			results: []GateResult{
				{Name: "g1", Status: GateStatusFailed, Severity: GateSeverityBlocking},
			},
			want: 1,
		},
		{
			name: "multiple blocking failures",
			results: []GateResult{
				{Name: "g1", Status: GateStatusFailed, Severity: GateSeverityBlocking},
				{Name: "g2", Status: GateStatusPassed, Severity: GateSeverityBlocking},
				{Name: "g3", Status: GateStatusFailed, Severity: GateSeverityBlocking},
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BlockingFailures(tt.results)
			if len(got) != tt.want {
				t.Errorf("BlockingFailures: got %d, want %d", len(got), tt.want)
			}

			// Verify all returned are blocking failures
			for _, r := range got {
				if r.Severity != GateSeverityBlocking {
					t.Errorf("returned non-blocking result: %s", r.Name)
				}
				if r.Status != GateStatusFailed {
					t.Errorf("returned non-failed result: %s", r.Name)
				}
			}
		})
	}
}

func TestBlockingFailures_PreservesOrder(t *testing.T) {
	results := []GateResult{
		{Name: "g1", Status: GateStatusFailed, Severity: GateSeverityBlocking},
		{Name: "g2", Status: GateStatusPassed, Severity: GateSeverityBlocking},
		{Name: "g3", Status: GateStatusFailed, Severity: GateSeverityBlocking},
		{Name: "g4", Status: GateStatusFailed, Severity: GateSeverityWarning},
		{Name: "g5", Status: GateStatusFailed, Severity: GateSeverityBlocking},
	}

	got := BlockingFailures(results)

	if len(got) != 3 {
		t.Fatalf("expected 3 failures, got %d", len(got))
	}

	expected := []string{"g1", "g3", "g5"}
	for i, name := range expected {
		if got[i].Name != name {
			t.Errorf("failure[%d]: got %q, want %q", i, got[i].Name, name)
		}
	}
}

// ─── Integration: full gate chain with ReviewReportExistsGate ────────────────

func TestCheckGates_ReviewingFeature_NoReport_Blocked(t *testing.T) {
	t.Parallel()
	ctx := GateContext{
		EntityID: "FEAT-REVIEWING-001",
		Branch:   "feature/FEAT-REVIEWING-001",
		RepoPath: "/repo",
		Entity: map[string]any{
			"status": "reviewing",
		},
		DocSvc:          &stubDocService{docs: nil},
		ConflictChecker: func(_, _, _ string) (bool, error) { return false, nil },
		BranchStatusChecker: func(_ string, _ string, _ git.BranchThresholds) (git.BranchStatus, error) {
			return git.BranchStatus{}, nil
		},
		DefaultBranchDetector: func(_ string) (string, error) { return "main", nil },
	}

	result := CheckGates(ctx)

	if result.OverallStatus != OverallStatusBlocked {
		t.Errorf("overall_status: got %q, want %q", result.OverallStatus, OverallStatusBlocked)
	}

	// ReviewReportExistsGate must be blocked and non-bypassable.
	nonBypassable := NonBypassableBlockingFailures(result.Gates)
	if len(nonBypassable) == 0 {
		t.Fatal("expected at least one non-bypassable blocking failure")
	}
	found := false
	for _, g := range nonBypassable {
		if g.Name == "review_report_exists" {
			found = true
			break
		}
	}
	if !found {
		t.Error("review_report_exists gate not in NonBypassableBlockingFailures results")
	}
}

func TestCheckGates_ReviewingFeature_WithReport_ReviewGatePasses(t *testing.T) {
	t.Parallel()
	ctx := GateContext{
		EntityID: "FEAT-REVIEWING-002",
		Branch:   "feature/FEAT-REVIEWING-002",
		RepoPath: "/repo",
		Entity: map[string]any{
			"status":              "reviewing",
			"verification":        "Reviewed",
			"verification_status": "passed",
		},
		DocSvc: &stubDocService{docs: []DocRecord{
			{ID: "rpt-1", Type: "report", Owner: "FEAT-REVIEWING-002", Status: "draft"},
		}},
		ConflictChecker: func(_, _, _ string) (bool, error) { return false, nil },
		BranchStatusChecker: func(_ string, _ string, _ git.BranchThresholds) (git.BranchStatus, error) {
			return git.BranchStatus{}, nil
		},
		DefaultBranchDetector: func(_ string) (string, error) { return "main", nil },
	}

	result := CheckGates(ctx)

	// EntityDoneGate will still block (reviewing != done), but ReviewReportExistsGate must pass.
	for _, g := range result.Gates {
		if g.Name == "review_report_exists" {
			if g.Status != GateStatusPassed {
				t.Errorf("review_report_exists: got %v, want passed", g.Status)
			}
			return
		}
	}
	t.Error("review_report_exists gate not found in results")
}

func TestCheckGates_ExistingGates_AreBypassable(t *testing.T) {
	t.Parallel()
	// Feature in "done" state should have all gates pass except possibly
	// doc service gates, and all results should have Bypassable: true.
	ctx := GateContext{
		EntityID: "FEAT-DONE-001",
		Branch:   "feature/FEAT-DONE-001",
		RepoPath: "/repo",
		Entity: map[string]any{
			"status":              "done",
			"verification":        "All tests pass",
			"verification_status": "passed",
		},
		Tasks: []map[string]any{
			{"id": "TASK-001", "status": "done"},
		},
		ConflictChecker: func(_, _, _ string) (bool, error) { return false, nil },
		BranchStatusChecker: func(_ string, _ string, _ git.BranchThresholds) (git.BranchStatus, error) {
			return git.BranchStatus{}, nil
		},
		DefaultBranchDetector: func(_ string) (string, error) { return "main", nil },
	}

	result := CheckGates(ctx)

	for _, g := range result.Gates {
		if g.Name == "review_report_exists" {
			continue // This gate sets Bypassable: true for non-reviewing features too.
		}
		if !g.Bypassable {
			t.Errorf("gate %q: Bypassable should be true for existing gates (regression)", g.Name)
		}
	}
}

func TestCheckGates_NilDocSvc_ReviewingGateFailsOpen(t *testing.T) {
	t.Parallel()
	ctx := GateContext{
		EntityID: "FEAT-REVIEWING-003",
		Branch:   "feature/FEAT-REVIEWING-003",
		RepoPath: "/repo",
		Entity: map[string]any{
			"status": "reviewing",
		},
		DocSvc:          nil, // no doc service
		ConflictChecker: func(_, _, _ string) (bool, error) { return false, nil },
		BranchStatusChecker: func(_ string, _ string, _ git.BranchThresholds) (git.BranchStatus, error) {
			return git.BranchStatus{}, nil
		},
		DefaultBranchDetector: func(_ string) (string, error) { return "main", nil },
	}

	result := CheckGates(ctx)

	// review_report_exists must pass (fail-open) — should NOT be in non-bypassable list.
	nonBypassable := NonBypassableBlockingFailures(result.Gates)
	for _, g := range nonBypassable {
		if g.Name == "review_report_exists" {
			t.Error("review_report_exists must fail-open when DocSvc is nil, but it appears as non-bypassable blocking failure")
		}
	}

	// The gate result itself must be Pass.
	for _, g := range result.Gates {
		if g.Name == "review_report_exists" {
			if g.Status != GateStatusPassed {
				t.Errorf("review_report_exists: got %v, want passed (fail-open with nil DocSvc)", g.Status)
			}
		}
	}
}
