package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kanbanzai/internal/model"
	"kanbanzai/internal/storage"
)

const testPlanIDReview = "P1-review-plan"

// setupReviewTest creates entity and intelligence services with a feature and
// an active task under it. Returns the review service, the task ID, and the
// task slug. If specDocID is non-empty, it is linked to the feature.
func setupReviewTest(t *testing.T, specDocID string) (*ReviewService, *EntityService, string, string) {
	t.Helper()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	indexRoot := t.TempDir()

	entitySvc := NewEntityService(stateRoot)
	intelSvc := NewIntelligenceService(indexRoot, repoRoot)

	writeReviewTestPlan(t, entitySvc, testPlanIDReview)

	featResult, err := entitySvc.CreateFeature(CreateFeatureInput{
		Slug:      "review-feature",
		Parent:    testPlanIDReview,
		Summary:   "Feature for review tests",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("create feature: %v", err)
	}
	featureID := featResult.ID

	if specDocID != "" {
		_, err = entitySvc.UpdateEntity(UpdateEntityInput{
			Type:   "feature",
			ID:     featureID,
			Slug:   "review-feature",
			Fields: map[string]string{"spec": specDocID},
		})
		if err != nil {
			t.Fatalf("link spec to feature: %v", err)
		}
	}

	taskResult, err := entitySvc.CreateTask(CreateTaskInput{
		ParentFeature: featureID,
		Slug:          "review-task",
		Summary:       "Implement authentication middleware with JWT tokens",
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	taskID := taskResult.ID

	// Transition task to active: queued → ready → active.
	_, err = entitySvc.UpdateStatus(UpdateStatusInput{
		Type: "task", ID: taskID, Slug: "review-task", Status: "ready",
	})
	if err != nil {
		t.Fatalf("transition to ready: %v", err)
	}
	_, err = entitySvc.UpdateStatus(UpdateStatusInput{
		Type: "task", ID: taskID, Slug: "review-task", Status: "active",
	})
	if err != nil {
		t.Fatalf("transition to active: %v", err)
	}

	reviewSvc := NewReviewService(entitySvc, intelSvc, repoRoot)
	return reviewSvc, entitySvc, taskID, "review-task"
}

func writeReviewTestPlan(t *testing.T, svc *EntityService, id string) {
	t.Helper()
	_, _, slug := model.ParsePlanID(id)
	fields := map[string]any{
		"id":         id,
		"slug":       slug,
		"title":      "Test Plan",
		"status":     "active",
		"summary":    "Test plan for review tests",
		"created":    "2026-03-19T12:00:00Z",
		"created_by": "test",
		"updated":    "2026-03-19T12:00:00Z",
	}
	_, err := svc.store.Write(storage.EntityRecord{
		Type:   string(model.EntityKindPlan),
		ID:     id,
		Slug:   slug,
		Fields: fields,
	})
	if err != nil {
		t.Fatalf("writeReviewTestPlan(%s) error = %v", id, err)
	}
}

// --- §16.3 AC 1: fail when output_files contains a file that does not exist ---

func TestReviewTaskOutput_MissingFile_Fails(t *testing.T) {
	t.Parallel()

	reviewSvc, _, taskID, _ := setupReviewTest(t, "")

	result, err := reviewSvc.ReviewTaskOutput(ReviewInput{
		TaskID:      taskID,
		OutputFiles: []string{"nonexistent/file.go", "also/missing.go"},
	})
	if err != nil {
		t.Fatalf("ReviewTaskOutput() error = %v", err)
	}

	if result.Status != "fail" {
		t.Errorf("Status = %q, want %q", result.Status, "fail")
	}
	if result.BlockingCount < 2 {
		t.Errorf("BlockingCount = %d, want >= 2", result.BlockingCount)
	}

	// All missing file findings should be errors.
	for _, f := range result.Findings {
		if f.Type == "missing_file" {
			if f.Severity != "error" {
				t.Errorf("missing_file finding severity = %q, want %q", f.Severity, "error")
			}
		}
	}
}

// --- §16.3 AC 2: transitions to needs-rework and sets rework_reason on fail ---

func TestReviewTaskOutput_FailTransitionsToNeedsRework(t *testing.T) {
	t.Parallel()

	reviewSvc, entitySvc, taskID, _ := setupReviewTest(t, "")

	result, err := reviewSvc.ReviewTaskOutput(ReviewInput{
		TaskID:      taskID,
		OutputFiles: []string{"does/not/exist.go"},
	})
	if err != nil {
		t.Fatalf("ReviewTaskOutput() error = %v", err)
	}

	if result.Status != "fail" {
		t.Fatalf("Status = %q, want %q", result.Status, "fail")
	}

	// Verify the task transitioned to needs-rework.
	taskResult, err := entitySvc.Get("task", taskID, "")
	if err != nil {
		t.Fatalf("Get task error = %v", err)
	}

	status, _ := taskResult.State["status"].(string)
	if status != "needs-rework" {
		t.Errorf("task status = %q, want %q", status, "needs-rework")
	}

	// Verify rework_reason is set.
	reworkReason, _ := taskResult.State["rework_reason"].(string)
	if reworkReason == "" {
		t.Error("rework_reason is empty, want non-empty")
	}
}

// --- §16.3 AC 3: transitions to needs-review on pass ---

func TestReviewTaskOutput_PassTransitionsToNeedsReview(t *testing.T) {
	t.Parallel()

	reviewSvc, entitySvc, taskID, _ := setupReviewTest(t, "DOC-fake-spec")

	// Create the output file so it exists.
	fullPath := filepath.Join(reviewSvc.repoRoot, "internal/auth.go")
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte("package auth"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	result, err := reviewSvc.ReviewTaskOutput(ReviewInput{
		TaskID:        taskID,
		OutputFiles:   []string{"internal/auth.go"},
		OutputSummary: "Implemented authentication middleware with JWT tokens and validation",
	})
	if err != nil {
		t.Fatalf("ReviewTaskOutput() error = %v", err)
	}

	// With a spec linked but no sections matching (spec_gap is a warning), this
	// should be pass_with_warnings at worst, not fail.
	if result.Status == "fail" {
		t.Fatalf("Status = %q, want pass or pass_with_warnings", result.Status)
	}

	// Verify the task transitioned to needs-review.
	taskResult, err := entitySvc.Get("task", taskID, "")
	if err != nil {
		t.Fatalf("Get task error = %v", err)
	}

	status, _ := taskResult.State["status"].(string)
	if status != "needs-review" {
		t.Errorf("task status = %q, want %q", status, "needs-review")
	}
}

// --- §16.3 AC 4: includes spec-level findings when feature has a linked spec ---

func TestReviewTaskOutput_SpecLevelFindings_WithSpec(t *testing.T) {
	t.Parallel()

	// Link a spec document to the feature. Since the spec doesn't reference
	// the task, we expect a spec_gap warning finding.
	reviewSvc, _, taskID, _ := setupReviewTest(t, "DOC-test-spec")

	result, err := reviewSvc.ReviewTaskOutput(ReviewInput{
		TaskID:        taskID,
		OutputSummary: "Implemented authentication middleware with JWT tokens",
	})
	if err != nil {
		t.Fatalf("ReviewTaskOutput() error = %v", err)
	}

	// Should include spec-level findings.
	hasSpecFinding := false
	for _, f := range result.Findings {
		if f.Type == "spec_gap" {
			hasSpecFinding = true
			if f.Severity != "warning" {
				t.Errorf("spec_gap finding severity = %q, want %q", f.Severity, "warning")
			}
		}
	}
	if !hasSpecFinding {
		t.Error("expected spec_gap finding when feature has spec but no sections reference the task")
	}

	// Spec-level findings must never be blocking (spec §17.4).
	if result.Status == "fail" {
		foundNonSpecError := false
		for _, f := range result.Findings {
			if f.Severity == "error" && f.Type != "spec_gap" {
				foundNonSpecError = true
			}
		}
		if !foundNonSpecError {
			t.Error("review failed solely on spec-level findings; spec findings must be warnings only")
		}
	}
}

// --- §16.3 AC 5: adds no_spec warning when no spec registered ---

func TestReviewTaskOutput_NoSpec_Warning(t *testing.T) {
	t.Parallel()

	// No spec document linked to the feature.
	reviewSvc, _, taskID, _ := setupReviewTest(t, "")

	result, err := reviewSvc.ReviewTaskOutput(ReviewInput{
		TaskID:        taskID,
		OutputSummary: "Implemented authentication middleware with JWT tokens",
	})
	if err != nil {
		t.Fatalf("ReviewTaskOutput() error = %v", err)
	}

	hasNoSpec := false
	for _, f := range result.Findings {
		if f.Type == "no_spec" {
			hasNoSpec = true
			if f.Severity != "warning" {
				t.Errorf("no_spec finding severity = %q, want %q", f.Severity, "warning")
			}
		}
	}
	if !hasNoSpec {
		t.Error("expected no_spec warning finding when no spec registered on feature")
	}

	// Should be pass_with_warnings (no_spec is a warning, not an error).
	if result.Status != "pass_with_warnings" {
		t.Errorf("Status = %q, want %q", result.Status, "pass_with_warnings")
	}
}

// --- §16.3 AC 6: rework_reason cleared on needs-rework → active ---

func TestReviewTaskOutput_ReworkReasonClearedOnActive(t *testing.T) {
	t.Parallel()

	reviewSvc, entitySvc, taskID, taskSlug := setupReviewTest(t, "")

	// Trigger a failing review to set rework_reason.
	_, err := reviewSvc.ReviewTaskOutput(ReviewInput{
		TaskID:      taskID,
		OutputFiles: []string{"missing/file.go"},
	})
	if err != nil {
		t.Fatalf("ReviewTaskOutput() error = %v", err)
	}

	// Verify rework_reason is set.
	taskResult, err := entitySvc.Get("task", taskID, "")
	if err != nil {
		t.Fatalf("Get task error = %v", err)
	}
	reworkReason, _ := taskResult.State["rework_reason"].(string)
	if reworkReason == "" {
		t.Fatal("rework_reason not set after failing review")
	}

	// Transition needs-rework → active.
	_, err = entitySvc.UpdateStatus(UpdateStatusInput{
		Type: "task", ID: taskID, Slug: taskSlug, Status: "active",
	})
	if err != nil {
		t.Fatalf("transition to active: %v", err)
	}

	// Verify rework_reason is cleared.
	taskResult, err = entitySvc.Get("task", taskID, "")
	if err != nil {
		t.Fatalf("Get task after reactivation: %v", err)
	}
	reworkReason, _ = taskResult.State["rework_reason"].(string)
	if reworkReason != "" {
		t.Errorf("rework_reason = %q after needs-rework → active, want empty", reworkReason)
	}
}

// --- §16.3 AC 7: no transition on needs-review or done ---

func TestReviewTaskOutput_AlreadyNeedsReview_NoTransition(t *testing.T) {
	t.Parallel()

	reviewSvc, entitySvc, taskID, taskSlug := setupReviewTest(t, "")

	// Transition to needs-review first.
	_, err := entitySvc.UpdateStatus(UpdateStatusInput{
		Type: "task", ID: taskID, Slug: taskSlug, Status: "needs-review",
	})
	if err != nil {
		t.Fatalf("transition to needs-review: %v", err)
	}

	// Run review with a missing file (would normally be a fail + transition).
	result, err := reviewSvc.ReviewTaskOutput(ReviewInput{
		TaskID:      taskID,
		OutputFiles: []string{"nonexistent.go"},
	})
	if err != nil {
		t.Fatalf("ReviewTaskOutput() error = %v", err)
	}

	// Should still report findings.
	if result.TotalFindings == 0 {
		t.Error("expected findings, got 0")
	}

	// But task should still be in needs-review (no transition).
	taskResult, err := entitySvc.Get("task", taskID, "")
	if err != nil {
		t.Fatalf("Get task: %v", err)
	}
	status, _ := taskResult.State["status"].(string)
	if status != "needs-review" {
		t.Errorf("task status = %q, want %q (no transition expected)", status, "needs-review")
	}
}

func TestReviewTaskOutput_AlreadyDone_NoTransition(t *testing.T) {
	t.Parallel()

	reviewSvc, entitySvc, taskID, taskSlug := setupReviewTest(t, "")

	// Transition to done: active → done.
	_, err := entitySvc.UpdateStatus(UpdateStatusInput{
		Type: "task", ID: taskID, Slug: taskSlug, Status: "done",
	})
	if err != nil {
		t.Fatalf("transition to done: %v", err)
	}

	result, err := reviewSvc.ReviewTaskOutput(ReviewInput{
		TaskID:      taskID,
		OutputFiles: []string{"nonexistent.go"},
	})
	if err != nil {
		t.Fatalf("ReviewTaskOutput() error = %v", err)
	}

	if result.TotalFindings == 0 {
		t.Error("expected findings, got 0")
	}

	// Task should still be done (no transition).
	taskResult, err := entitySvc.Get("task", taskID, "")
	if err != nil {
		t.Fatalf("Get task: %v", err)
	}
	status, _ := taskResult.State["status"].(string)
	if status != "done" {
		t.Errorf("task status = %q, want %q (no transition expected)", status, "done")
	}
}

// --- §16.3 AC 8: round-trip serialisation with rework_reason ---

func TestReviewTaskOutput_RoundTrip_ReworkReason(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	entitySvc := NewEntityService(stateRoot)

	writeReviewTestPlan(t, entitySvc, testPlanIDReview)

	featResult, err := entitySvc.CreateFeature(CreateFeatureInput{
		Slug:      "rt-feature",
		Parent:    testPlanIDReview,
		Summary:   "Feature for round-trip test",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("create feature: %v", err)
	}

	taskResult, err := entitySvc.CreateTask(CreateTaskInput{
		ParentFeature: featResult.ID,
		Slug:          "rt-task",
		Summary:       "Task for round-trip test",
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	taskID := taskResult.ID

	// Set rework_reason via UpdateEntity.
	_, err = entitySvc.UpdateEntity(UpdateEntityInput{
		Type:   "task",
		ID:     taskID,
		Slug:   "rt-task",
		Fields: map[string]string{"rework_reason": "output file missing: internal/auth.go"},
	})
	if err != nil {
		t.Fatalf("set rework_reason: %v", err)
	}

	// Read back the task YAML file.
	taskDir := filepath.Join(stateRoot, "tasks")
	entries, err := os.ReadDir(taskDir)
	if err != nil {
		t.Fatalf("read task dir: %v", err)
	}

	var taskFile string
	for _, e := range entries {
		if !e.IsDir() {
			taskFile = filepath.Join(taskDir, e.Name())
			break
		}
	}
	if taskFile == "" {
		t.Fatal("no task file found")
	}

	firstWrite, err := os.ReadFile(taskFile)
	if err != nil {
		t.Fatalf("read first write: %v", err)
	}

	// Load and re-save (trigger a round-trip through the store).
	loaded, err := entitySvc.Get("task", taskID, "")
	if err != nil {
		t.Fatalf("Get task: %v", err)
	}

	// Verify rework_reason survived the load.
	rr, _ := loaded.State["rework_reason"].(string)
	if rr != "output file missing: internal/auth.go" {
		t.Errorf("rework_reason after load = %q, want %q", rr, "output file missing: internal/auth.go")
	}

	// Write back with a trivial field update to force a re-write.
	_, err = entitySvc.UpdateEntity(UpdateEntityInput{
		Type:   "task",
		ID:     taskID,
		Slug:   "rt-task",
		Fields: map[string]string{"rework_reason": "output file missing: internal/auth.go"},
	})
	if err != nil {
		t.Fatalf("re-write task: %v", err)
	}

	secondWrite, err := os.ReadFile(taskFile)
	if err != nil {
		t.Fatalf("read second write: %v", err)
	}

	if string(firstWrite) != string(secondWrite) {
		t.Errorf("round-trip mismatch:\n--- first ---\n%s\n--- second ---\n%s", firstWrite, secondWrite)
	}
}

// --- Additional service tests (C.11) ---

func TestReviewTaskOutput_VerificationMet_Pass(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	indexRoot := t.TempDir()

	entitySvc := NewEntityService(stateRoot)
	intelSvc := NewIntelligenceService(indexRoot, repoRoot)

	writeReviewTestPlan(t, entitySvc, testPlanIDReview)

	featResult, err := entitySvc.CreateFeature(CreateFeatureInput{
		Slug:      "ver-feature",
		Parent:    testPlanIDReview,
		Summary:   "Feature for verification test",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("create feature: %v", err)
	}

	taskResult, err := entitySvc.CreateTask(CreateTaskInput{
		ParentFeature: featResult.ID,
		Slug:          "ver-task",
		Summary:       "Implement JWT authentication",
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	taskID := taskResult.ID

	// Set verification via UpdateEntity (not part of CreateTaskInput).
	_, err = entitySvc.UpdateEntity(UpdateEntityInput{
		Type:   "task",
		ID:     taskID,
		Slug:   "ver-task",
		Fields: map[string]string{"verification": "JWT tokens are validated; expired tokens are rejected; tests pass"},
	})
	if err != nil {
		t.Fatalf("set verification: %v", err)
	}

	// Transition to active.
	_, err = entitySvc.UpdateStatus(UpdateStatusInput{
		Type: "task", ID: taskID, Slug: "ver-task", Status: "ready",
	})
	if err != nil {
		t.Fatalf("transition to ready: %v", err)
	}
	_, err = entitySvc.UpdateStatus(UpdateStatusInput{
		Type: "task", ID: taskID, Slug: "ver-task", Status: "active",
	})
	if err != nil {
		t.Fatalf("transition to active: %v", err)
	}

	reviewSvc := NewReviewService(entitySvc, intelSvc, repoRoot)

	result, err := reviewSvc.ReviewTaskOutput(ReviewInput{
		TaskID:        taskID,
		OutputSummary: "Implemented JWT tokens validation and expired token rejection with full tests passing",
	})
	if err != nil {
		t.Fatalf("ReviewTaskOutput() error = %v", err)
	}

	// No blocking findings — verification keywords should match.
	if result.BlockingCount > 0 {
		t.Errorf("BlockingCount = %d, want 0", result.BlockingCount)
	}

	// Should not be fail.
	if result.Status == "fail" {
		t.Errorf("Status = %q, want pass or pass_with_warnings", result.Status)
	}
}

func TestReviewTaskOutput_ExistingFilePasses(t *testing.T) {
	t.Parallel()

	reviewSvc, _, taskID, _ := setupReviewTest(t, "")

	// Create a real file in the repo root.
	fullPath := filepath.Join(reviewSvc.repoRoot, "internal/handler.go")
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte("package internal"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	result, err := reviewSvc.ReviewTaskOutput(ReviewInput{
		TaskID:        taskID,
		OutputFiles:   []string{"internal/handler.go"},
		OutputSummary: "Implemented authentication middleware with JWT tokens",
	})
	if err != nil {
		t.Fatalf("ReviewTaskOutput() error = %v", err)
	}

	// Should have no missing_file findings.
	for _, f := range result.Findings {
		if f.Type == "missing_file" {
			t.Errorf("unexpected missing_file finding: %s", f.Detail)
		}
	}
}

func TestReviewTaskOutput_SpecGap_NeverBlocking(t *testing.T) {
	t.Parallel()

	// Feature has a spec but doc_trace won't find the task → spec_gap warning.
	reviewSvc, _, taskID, _ := setupReviewTest(t, "DOC-some-spec")

	result, err := reviewSvc.ReviewTaskOutput(ReviewInput{
		TaskID:        taskID,
		OutputSummary: "Implemented authentication middleware with JWT tokens",
	})
	if err != nil {
		t.Fatalf("ReviewTaskOutput() error = %v", err)
	}

	// spec_gap findings must be warnings, never errors.
	for _, f := range result.Findings {
		if f.Type == "spec_gap" && f.Severity == "error" {
			t.Errorf("spec_gap finding has severity=error; must be warning per spec §17.4")
		}
	}

	// The review must not fail solely on spec heuristics.
	if result.Status == "fail" {
		for _, f := range result.Findings {
			if f.Severity == "error" && f.Type != "spec_gap" {
				return // There's a real error, fail is justified.
			}
		}
		t.Error("review failed solely on spec-level findings; spec findings must be warnings only")
	}
}

// --- Invalid status tests ---

func TestReviewTaskOutput_InvalidStatus_RejectsQueued(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	repoRoot := t.TempDir()
	indexRoot := t.TempDir()

	entitySvc := NewEntityService(stateRoot)
	intelSvc := NewIntelligenceService(indexRoot, repoRoot)

	writeReviewTestPlan(t, entitySvc, testPlanIDReview)

	featResult, err := entitySvc.CreateFeature(CreateFeatureInput{
		Slug:      "q-feature",
		Parent:    testPlanIDReview,
		Summary:   "Feature for queued test",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("create feature: %v", err)
	}

	taskResult, err := entitySvc.CreateTask(CreateTaskInput{
		ParentFeature: featResult.ID,
		Slug:          "queued-task",
		Summary:       "A task that stays queued",
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	reviewSvc := NewReviewService(entitySvc, intelSvc, repoRoot)

	_, err = reviewSvc.ReviewTaskOutput(ReviewInput{
		TaskID: taskResult.ID,
	})
	if err == nil {
		t.Error("expected error for queued task, got nil")
	}
}

func TestReviewTaskOutput_EmptyTaskID(t *testing.T) {
	t.Parallel()

	reviewSvc, _, _, _ := setupReviewTest(t, "")

	_, err := reviewSvc.ReviewTaskOutput(ReviewInput{
		TaskID: "",
	})
	if err == nil {
		t.Error("expected error for empty task_id, got nil")
	}
}

// --- CLI routing tests (C.10) ---

func TestExtractKeywords(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  int // minimum number of keywords expected
	}{
		{"simple phrase", "implement JWT authentication", 2},
		{"stop words filtered", "the quick and the lazy dog", 2},
		{"short words filtered", "a b c go do it", 0},
		{"empty string", "", 0},
		{"hyphenated words kept", "two-factor auth", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keywords := extractKeywords(tt.input)
			if len(keywords) < tt.want {
				t.Errorf("extractKeywords(%q) = %d keywords %v, want >= %d", tt.input, len(keywords), keywords, tt.want)
			}
		})
	}
}

func TestSummarizeBlockingFindings(t *testing.T) {
	t.Parallel()

	findings := []ReviewFinding{
		{Severity: "error", Type: "missing_file", Detail: "file A missing"},
		{Severity: "warning", Type: "no_spec", Detail: "no spec"},
		{Severity: "error", Type: "missing_file", Detail: "file B missing"},
	}

	reason := summarizeBlockingFindings(findings)
	if reason == "" {
		t.Error("expected non-empty reason")
	}
	// Should only include error findings.
	if !reviewContains(reason, "file A missing") || !reviewContains(reason, "file B missing") {
		t.Errorf("reason = %q, expected both blocking details", reason)
	}
	if reviewContains(reason, "no spec") {
		t.Errorf("reason = %q, should not include warning-level findings", reason)
	}
}

func TestSummarizeBlockingFindings_NoErrors(t *testing.T) {
	t.Parallel()

	findings := []ReviewFinding{
		{Severity: "warning", Type: "no_spec", Detail: "no spec"},
	}

	reason := summarizeBlockingFindings(findings)
	if reason != "review failed" {
		t.Errorf("reason = %q, want %q", reason, "review failed")
	}
}

// reviewContains checks if s contains substr. Named to avoid conflict with
// the contains helper in decompose_test.go (same package).
func reviewContains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// --- Active → NeedsRework lifecycle transition test ---

func TestLifecycle_ActiveToNeedsRework(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	entitySvc := NewEntityService(stateRoot)

	writeReviewTestPlan(t, entitySvc, testPlanIDReview)

	featResult, err := entitySvc.CreateFeature(CreateFeatureInput{
		Slug:      "lc-feature",
		Parent:    testPlanIDReview,
		Summary:   "Feature for lifecycle test",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("create feature: %v", err)
	}

	taskResult, err := entitySvc.CreateTask(CreateTaskInput{
		ParentFeature: featResult.ID,
		Slug:          "lc-task",
		Summary:       "Lifecycle test task",
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	taskID := taskResult.ID

	// queued → ready → active → needs-rework should be valid.
	for _, status := range []string{"ready", "active", "needs-rework"} {
		_, err = entitySvc.UpdateStatus(UpdateStatusInput{
			Type: "task", ID: taskID, Slug: "lc-task", Status: status,
		})
		if err != nil {
			t.Fatalf("transition to %s: %v", status, err)
		}
	}

	taskResult2, err := entitySvc.Get("task", taskID, "")
	if err != nil {
		t.Fatalf("Get task: %v", err)
	}
	status, _ := taskResult2.State["status"].(string)
	if status != "needs-rework" {
		t.Errorf("task status = %q, want %q", status, "needs-rework")
	}
}
