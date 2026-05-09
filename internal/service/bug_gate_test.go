package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sambeau/kanbanzai/internal/model"
)

// ─── T1: Review Cycle Cap ──────────────────────────────────────────────────

func TestCheckBugTransitionGate_NeedsReviewToNeedsRework_CapNotReached(t *testing.T) {
	t.Parallel()

	bug := &model.Bug{
		ID:          "BUG-01TEST0001",
		Slug:        "test-bug",
		Status:      model.BugStatusNeedsReview,
		Tier:        "bug_fix",
		ReviewCycle: 0,
	}

	result := CheckBugTransitionGate(
		string(model.BugStatusNeedsReview),
		string(model.BugStatusNeedsRework),
		bug, nil, nil,
	)

	if !result.Satisfied {
		t.Errorf("expected gate satisfied at cycle 0, got: %s", result.Reason)
	}
}

func TestCheckBugTransitionGate_NeedsReviewToNeedsRework_CapReached(t *testing.T) {
	t.Parallel()

	bug := &model.Bug{
		ID:          "BUG-01TEST0002",
		Slug:        "test-bug",
		Status:      model.BugStatusNeedsReview,
		Tier:        "bug_fix",
		ReviewCycle: 2, // at cap (DefaultBugMaxReviewCycles = 2)
	}

	result := CheckBugTransitionGate(
		string(model.BugStatusNeedsReview),
		string(model.BugStatusNeedsRework),
		bug, nil, nil,
	)

	if result.Satisfied {
		t.Error("expected gate unsatisfied at cap")
	}
	if !result.ReviewCapReached {
		t.Error("expected ReviewCapReached=true")
	}
}

func TestCheckBugTransitionGate_NeedsReviewToNeedsRework_FeatureEquivalent(t *testing.T) {
	t.Parallel()

	bug := &model.Bug{
		ID:          "BUG-01TEST0003",
		Slug:        "test-bug",
		Status:      model.BugStatusNeedsReview,
		Tier:        "feature_equivalent",
		ReviewCycle: 3, // below cap for feature_equivalent (4)
	}

	result := CheckBugTransitionGate(
		string(model.BugStatusNeedsReview),
		string(model.BugStatusNeedsRework),
		bug, nil, nil,
	)

	if !result.Satisfied {
		t.Errorf("expected gate satisfied at cycle 3 for feature_equivalent, got: %s", result.Reason)
	}
}

// ─── T1: Placeholder — Verifier Not Yet Implemented ────────────────────────

func TestCheckBugTransitionGate_VerifiedToClosed_Placeholder(t *testing.T) {
	t.Parallel()

	bug := &model.Bug{
		ID:     "BUG-01TEST0004",
		Slug:   "test-bug",
		Status: model.BugStatusVerifying,
	}

	result := CheckBugTransitionGate(
		string(model.BugStatusVerifying),
		string(model.BugStatusClosed),
		bug, nil, nil,
	)

	if !result.Satisfied {
		t.Errorf("expected placeholder gate to be satisfied, got: %s", result.Reason)
	}
	if result.Reason != "verifier not yet implemented — see F4 and P55 Component 7" {
		t.Errorf("expected placeholder reason, got: %s", result.Reason)
	}
}

// ─── T1: Auto-Detection — Verifier Role File Exists ────────────────────────

func TestCheckBugTransitionGate_VerifiedToClosed_VerifierExists(t *testing.T) {
	// NOTE: cannot run in parallel because it calls os.Chdir.

	tmpDir := t.TempDir()
	kbzRolesDir := filepath.Join(tmpDir, ".kbz", "roles")
	if err := os.MkdirAll(kbzRolesDir, 0o755); err != nil {
		t.Fatalf("mkdir .kbz/roles: %v", err)
	}
	if err := os.WriteFile(filepath.Join(kbzRolesDir, "verifier.yaml"), []byte("id: verifier\n"), 0o644); err != nil {
		t.Fatalf("write verifier.yaml: %v", err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(origDir)

	bug := &model.Bug{
		ID:     "BUG-01TEST0005",
		Slug:   "test-bug",
		Status: model.BugStatusVerifying,
	}

	result := CheckBugTransitionGate(
		string(model.BugStatusVerifying),
		string(model.BugStatusClosed),
		bug, nil, nil,
	)

	if result.Satisfied {
		t.Error("expected gate to require verifier dispatch (not satisfied)")
	}
	if !result.NeedsVerifier {
		t.Error("expected NeedsVerifier=true when verifier role file exists")
	}
	if result.VerifierPrompt == "" {
		t.Error("expected non-empty VerifierPrompt")
	}
	if !strContains(result.VerifierPrompt, "verifier") {
		t.Error("prompt should mention verifier role")
	}
	if !strContains(result.VerifierPrompt, "verify-closeout") {
		t.Error("prompt should mention verify-closeout skill")
	}
	if !strContains(result.VerifierPrompt, bug.ID) {
		t.Error("prompt should contain bug ID")
	}
}

// ─── T1: Verifier Timeout (FR-412) ─────────────────────────────────────────

func TestCheckBugTransitionGate_VerifiedToClosed_VerifierTimeout(t *testing.T) {
	// NOTE: cannot run in parallel because it calls os.Chdir.

	tmpDir := t.TempDir()
	kbzRolesDir := filepath.Join(tmpDir, ".kbz", "roles")
	if err := os.MkdirAll(kbzRolesDir, 0o755); err != nil {
		t.Fatalf("mkdir .kbz/roles: %v", err)
	}
	if err := os.WriteFile(filepath.Join(kbzRolesDir, "verifier.yaml"), []byte("id: verifier\n"), 0o644); err != nil {
		t.Fatalf("write verifier.yaml: %v", err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(origDir)

	bug := &model.Bug{
		ID:     "BUG-01TEST0401",
		Slug:   "timeout-bug",
		Status: model.BugStatusVerifying,
	}

	// Clean up any previous dispatch record for this bug.
	ResetVerifierDispatchTime(bug.ID)

	// First call: should dispatch the verifier.
	result1 := CheckBugTransitionGate(
		string(model.BugStatusVerifying),
		string(model.BugStatusClosed),
		bug, nil, nil,
	)
	if result1.Satisfied {
		t.Error("expected gate to require verifier dispatch (not satisfied)")
	}
	if !result1.NeedsVerifier {
		t.Error("expected NeedsVerifier=true on first dispatch")
	}
	if result1.VerifierTimedOut {
		t.Error("expected VerifierTimedOut=false on first dispatch")
	}

	// Manipulate the dispatch time to simulate a timeout: set it in the past.
	verifierDispatchTimes.Store(bug.ID, time.Now().Add(-VerifierTimeout-time.Second))

	// Second call: should detect timeout.
	result2 := CheckBugTransitionGate(
		string(model.BugStatusVerifying),
		string(model.BugStatusClosed),
		bug, nil, nil,
	)
	if result2.Satisfied {
		t.Error("expected timeout gate to be unsatisfied")
	}
	if !result2.VerifierTimedOut {
		t.Error("expected VerifierTimedOut=true after timeout")
	}
	if result2.NeedsVerifier {
		t.Error("expected NeedsVerifier=false when timed out")
	}
	if !strContains(result2.Reason, "timed out") {
		t.Errorf("expected reason to contain 'timed out', got: %s", result2.Reason)
	}
}

func TestCheckBugTransitionGate_VerifiedToClosed_VerifierStillRunning(t *testing.T) {
	// NOTE: cannot run in parallel because it calls os.Chdir.

	tmpDir := t.TempDir()
	kbzRolesDir := filepath.Join(tmpDir, ".kbz", "roles")
	if err := os.MkdirAll(kbzRolesDir, 0o755); err != nil {
		t.Fatalf("mkdir .kbz/roles: %v", err)
	}
	if err := os.WriteFile(filepath.Join(kbzRolesDir, "verifier.yaml"), []byte("id: verifier\n"), 0o644); err != nil {
		t.Fatalf("write verifier.yaml: %v", err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(origDir)

	bug := &model.Bug{
		ID:     "BUG-01TEST0402",
		Slug:   "running-bug",
		Status: model.BugStatusVerifying,
	}

	// Clean up and set a recent dispatch time (not timed out).
	ResetVerifierDispatchTime(bug.ID)
	verifierDispatchTimes.Store(bug.ID, time.Now())

	// Should return NeedsVerifier=true (still running), not timed out.
	result := CheckBugTransitionGate(
		string(model.BugStatusVerifying),
		string(model.BugStatusClosed),
		bug, nil, nil,
	)
	if result.Satisfied {
		t.Error("expected gate to be unsatisfied while verifier is running")
	}
	if !result.NeedsVerifier {
		t.Error("expected NeedsVerifier=true while verifier is still running")
	}
	if result.VerifierTimedOut {
		t.Error("expected VerifierTimedOut=false while verifier is still running")
	}
}

// ─── T2: Verifier Prompt Content ───────────────────────────────────────────

func TestBuildVerifierPrompt_ContainsAllChecklistItems(t *testing.T) {
	t.Parallel()

	bug := &model.Bug{
		ID:   "BUG-01TEST0006",
		Slug: "prompt-test-bug",
	}

	prompt := buildVerifierPrompt(bug)

	required := []string{
		"Fix verified",
		"Changes committed",
		"Temp files removed",
		"Tests pass",
		"Code reviewed",
		"Full lifecycle",
		"Landed on main",
		"Worktree cleaned up",
	}
	for _, item := range required {
		if !strContains(prompt, item) {
			t.Errorf("prompt missing checklist item: %s", item)
		}
	}

	if !strContains(prompt, "bug_id") {
		t.Error("prompt missing JSON output schema")
	}
	if !strContains(prompt, "verdict") {
		t.Error("prompt missing verdict field")
	}
	if !strContains(prompt, "doc(action: \"register\"") {
		t.Error("prompt missing doc registration instruction")
	}
	if !strContains(prompt, "work/reviews/verify-") {
		t.Error("prompt missing report path")
	}
}

// ─── T3: Parse Verifier Report ─────────────────────────────────────────────

func TestParseVerifierReport_Pass(t *testing.T) {
	t.Parallel()

	raw := `{
  "bug_id": "BUG-01TEST0007",
  "checked_at": "2026-05-07T12:00:00Z",
  "verdict": "pass",
  "items": [
    {"dod_item": 1, "description": "Fix verified", "verdict": "pass", "evidence": "ok"},
    {"dod_item": 2, "description": "Changes committed", "verdict": "pass", "evidence": "ok"},
    {"dod_item": 3, "description": "Temp files removed", "verdict": "pass", "evidence": "ok"},
    {"dod_item": 4, "description": "Tests pass", "verdict": "pass", "evidence": "ok"},
    {"dod_item": 5, "description": "Code reviewed", "verdict": "pass", "evidence": "ok"},
    {"dod_item": 6, "description": "Full lifecycle", "verdict": "pass", "evidence": "ok"},
    {"dod_item": 7, "description": "Landed on main", "verdict": "pass", "evidence": "ok"},
    {"dod_item": 8, "description": "Worktree cleaned up", "verdict": "pass", "evidence": "ok"}
  ]
}`

	report, err := ParseVerifierReport(raw)
	if err != nil {
		t.Fatalf("ParseVerifierReport: %v", err)
	}
	if report.Verdict != "pass" {
		t.Errorf("verdict = %q, want pass", report.Verdict)
	}
	if len(report.Items) != 8 {
		t.Errorf("got %d items, want 8", len(report.Items))
	}
}

func TestParseVerifierReport_Fail(t *testing.T) {
	t.Parallel()

	raw := `{
  "bug_id": "BUG-01TEST0008",
  "checked_at": "2026-05-07T12:00:00Z",
  "verdict": "fail",
  "items": [
    {"dod_item": 1, "description": "Fix verified", "verdict": "pass", "evidence": "ok"},
    {"dod_item": 2, "description": "Changes committed", "verdict": "pass", "evidence": "ok"},
    {"dod_item": 3, "description": "Temp files removed", "verdict": "pass", "evidence": "ok"},
    {"dod_item": 4, "description": "Tests pass", "verdict": "fail", "evidence": "go test failed"},
    {"dod_item": 5, "description": "Code reviewed", "verdict": "pass", "evidence": "ok"},
    {"dod_item": 6, "description": "Full lifecycle", "verdict": "pass", "evidence": "ok"},
    {"dod_item": 7, "description": "Landed on main", "verdict": "pass", "evidence": "ok"},
    {"dod_item": 8, "description": "Worktree cleaned up", "verdict": "fail", "evidence": "branch exists"}
  ]
}`

	report, err := ParseVerifierReport(raw)
	if err != nil {
		t.Fatalf("ParseVerifierReport: %v", err)
	}
	if report.Verdict != "fail" {
		t.Errorf("verdict = %q, want fail", report.Verdict)
	}
}

func TestParseVerifierReport_MarkdownWrapped(t *testing.T) {
	t.Parallel()

	raw := "```json\n{\"bug_id\": \"BUG-01TEST0009\", \"checked_at\": \"2026-05-07T12:00:00Z\", \"verdict\": \"pass\", \"items\": []}\n```"

	report, err := ParseVerifierReport(raw)
	if err != nil {
		t.Fatalf("ParseVerifierReport with markdown: %v", err)
	}
	if report.BugID != "BUG-01TEST0009" {
		t.Errorf("bug_id = %q, want BUG-01TEST0009", report.BugID)
	}
}

func TestParseVerifierReport_InvalidJSON(t *testing.T) {
	t.Parallel()

	_, err := ParseVerifierReport("not json at all")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// ─── T3: Apply Verifier Report ─────────────────────────────────────────────

func TestApplyVerifierReport_AllPass(t *testing.T) {
	t.Parallel()

	report := &VerifierReport{
		BugID:   "BUG-01TEST0010",
		Verdict: "pass",
	}

	result := ApplyVerifierReport(report)
	if !result.Satisfied {
		t.Errorf("expected satisfied for all-pass, got: %s", result.Reason)
	}
}

func TestApplyVerifierReport_SomeFail(t *testing.T) {
	t.Parallel()

	report := &VerifierReport{
		BugID:   "BUG-01TEST0011",
		Verdict: "fail",
		Items: []VerifierReportItem{
			{DoDItem: 2, Description: "Changes committed", Verdict: "fail", Evidence: "unstaged changes"},
			{DoDItem: 4, Description: "Tests pass", Verdict: "fail", Evidence: "1 test failure"},
			{DoDItem: 1, Description: "Fix verified", Verdict: "pass", Evidence: "ok"},
		},
	}

	result := ApplyVerifierReport(report)
	if result.Satisfied {
		t.Error("expected unsatisfied for failing report")
	}
	if !strContains(result.Reason, "2 (Changes committed)") {
		t.Errorf("reason missing item 2: %s", result.Reason)
	}
	if !strContains(result.Reason, "4 (Tests pass)") {
		t.Errorf("reason missing item 4: %s", result.Reason)
	}
	if strContains(result.Reason, "1 (Fix verified)") {
		t.Error("reason should not list passing items")
	}
}

// ─── T1: Ungated Transitions ───────────────────────────────────────────────

func TestCheckBugTransitionGate_UngatedTransitions(t *testing.T) {
	t.Parallel()

	tests := []struct{ from, to string }{
		{"reported", "triaged"},
		{"triaged", "reproduced"},
		{"triaged", "cannot-reproduce"},
		{"triaged", "not-planned"},
		{"reproduced", "planned"},
		{"planned", "in-progress"},
		{"in-progress", "needs-review"},
		{"needs-review", "verified"},
		{"needs-rework", "in-progress"},
	}

	for _, tt := range tests {
		bug := &model.Bug{
			ID:     "BUG-01TEST0099",
			Slug:   "test-bug",
			Status: model.BugStatus(tt.from),
		}
		result := CheckBugTransitionGate(tt.from, tt.to, bug, nil, nil)
		if !result.Satisfied {
			t.Errorf("%s→%s: expected ungated, got: %s", tt.from, tt.to, result.Reason)
		}
	}
}

// ─── T3: Verifier Report Round-Trip ────────────────────────────────────────

func TestVerifierReport_RoundTrip(t *testing.T) {
	t.Parallel()

	original := VerifierReport{
		BugID:     "BUG-01TEST0012",
		CheckedAt: "2026-05-07T12:00:00Z",
		Verdict:   "pass",
		Items: []VerifierReportItem{
			{DoDItem: 1, Description: "Fix verified", Verdict: "pass", Evidence: "ok"},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	parsed, err := ParseVerifierReport(string(data))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if parsed.BugID != original.BugID {
		t.Errorf("BugID: got %q, want %q", parsed.BugID, original.BugID)
	}
	if parsed.Verdict != original.Verdict {
		t.Errorf("Verdict: got %q, want %q", parsed.Verdict, original.Verdict)
	}
	if len(parsed.Items) != len(original.Items) {
		t.Errorf("Items: got %d, want %d", len(parsed.Items), len(original.Items))
	}
}

func strContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
