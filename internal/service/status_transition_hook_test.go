package service

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambeau/kanbanzai/internal/worktree"
)

// initTestGitRepo creates a minimal git repo at path with one commit on "main".
func initTestGitRepo(t *testing.T, path string) {
	t.Helper()
	for _, args := range [][]string{
		{"init"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Test User"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = path
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	readme := filepath.Join(path, "README.md")
	if err := os.WriteFile(readme, []byte("# test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	for _, args := range [][]string{
		{"add", "."},
		{"commit", "-m", "initial"},
		{"branch", "-M", "main"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = path
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

const testPlanID = "P1-hook-test"

// --- Tests using a mock StatusTransitionHook ---

// mockStatusTransitionHook records calls for assertion.
type mockStatusTransitionHook struct {
	calls []statusTransitionCall
	// result to return from OnStatusTransition
	result *WorktreeResult
}

type statusTransitionCall struct {
	entityType string
	entityID   string
	slug       string
	fromStatus string
	toStatus   string
}

func (m *mockStatusTransitionHook) OnStatusTransition(entityType, entityID, slug, fromStatus, toStatus string, state map[string]any) *WorktreeResult {
	m.calls = append(m.calls, statusTransitionCall{
		entityType: entityType,
		entityID:   entityID,
		slug:       slug,
		fromStatus: fromStatus,
		toStatus:   toStatus,
	})
	return m.result
}

func TestUpdateStatus_FiresHookOnTransition(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-06-01T12:00:00Z")
	mock := &mockStatusTransitionHook{}
	svc.SetStatusTransitionHook(mock)

	// Create a plan and feature so we can create a task under it
	writeTestPlan(t, svc, testPlanID)
	feat, err := svc.CreateFeature(CreateFeatureInput{
		Name: "test",
		Slug:      "test-feature",
		Parent:    testPlanID,
		Summary:   "A test feature",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature: %v", err)
	}

	// Create a task under the feature
	task, err := svc.CreateTask(CreateTaskInput{
		Name: "test",
		ParentFeature: feat.ID,
		Slug:          "test-task",
		Summary:       "A test task",
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	// Transition task: queued → ready
	_, err = svc.UpdateStatus(UpdateStatusInput{
		Type:   "task",
		ID:     task.ID,
		Slug:   task.Slug,
		Status: "ready",
	})
	if err != nil {
		t.Fatalf("UpdateStatus queued→ready: %v", err)
	}

	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 hook call, got %d", len(mock.calls))
	}

	call := mock.calls[0]
	if call.entityType != "task" {
		t.Errorf("hook entityType = %q, want %q", call.entityType, "task")
	}
	if call.entityID != task.ID {
		t.Errorf("hook entityID = %q, want %q", call.entityID, task.ID)
	}
	if call.fromStatus != "queued" {
		t.Errorf("hook fromStatus = %q, want %q", call.fromStatus, "queued")
	}
	if call.toStatus != "ready" {
		t.Errorf("hook toStatus = %q, want %q", call.toStatus, "ready")
	}
}

func TestUpdateStatus_HookResultAttachedToGetResult(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-06-01T12:00:00Z")
	mock := &mockStatusTransitionHook{
		result: &WorktreeResult{
			Created:    true,
			WorktreeID: "WT-TESTID",
			EntityID:   "FEAT-ABCDEF",
			Branch:     "feature/FEAT-ABCDEF-test",
			Path:       ".worktrees/FEAT-ABCDEF-test",
		},
	}
	svc.SetStatusTransitionHook(mock)

	writeTestPlan(t, svc, testPlanID)
	feat, err := svc.CreateFeature(CreateFeatureInput{
		Name: "test",
		Slug:      "hook-result-test",
		Parent:    testPlanID,
		Summary:   "Test hook result propagation",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature: %v", err)
	}

	task, err := svc.CreateTask(CreateTaskInput{
		Name: "test",
		ParentFeature: feat.ID,
		Slug:          "hook-result-task",
		Summary:       "Test task",
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	// queued → ready
	result, err := svc.UpdateStatus(UpdateStatusInput{
		Type:   "task",
		ID:     task.ID,
		Slug:   task.Slug,
		Status: "ready",
	})
	if err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	if result.WorktreeHookResult == nil {
		t.Fatal("WorktreeHookResult is nil, expected non-nil")
	}
	if !result.WorktreeHookResult.Created {
		t.Error("WorktreeHookResult.Created = false, want true")
	}
	if result.WorktreeHookResult.WorktreeID != "WT-TESTID" {
		t.Errorf("WorktreeHookResult.WorktreeID = %q, want %q", result.WorktreeHookResult.WorktreeID, "WT-TESTID")
	}
}

func TestUpdateStatus_NilHookResultWhenNoHookSet(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-06-01T12:00:00Z")
	// No hook set

	writeTestPlan(t, svc, testPlanID)
	feat, err := svc.CreateFeature(CreateFeatureInput{
		Name: "test",
		Slug:      "no-hook-test",
		Parent:    testPlanID,
		Summary:   "Test no hook",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature: %v", err)
	}

	task, err := svc.CreateTask(CreateTaskInput{
		Name: "test",
		ParentFeature: feat.ID,
		Slug:          "no-hook-task",
		Summary:       "Test task",
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	result, err := svc.UpdateStatus(UpdateStatusInput{
		Type:   "task",
		ID:     task.ID,
		Slug:   task.Slug,
		Status: "ready",
	})
	if err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	if result.WorktreeHookResult != nil {
		t.Errorf("WorktreeHookResult should be nil when no hook is set, got %+v", result.WorktreeHookResult)
	}
}

func TestUpdateStatus_HookNilResultDoesNotPanic(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-06-01T12:00:00Z")
	mock := &mockStatusTransitionHook{result: nil} // returns nil
	svc.SetStatusTransitionHook(mock)

	writeTestPlan(t, svc, testPlanID)
	feat, err := svc.CreateFeature(CreateFeatureInput{
		Name: "test",
		Slug:      "nil-result-test",
		Parent:    testPlanID,
		Summary:   "Test nil result from hook",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature: %v", err)
	}

	task, err := svc.CreateTask(CreateTaskInput{
		Name: "test",
		ParentFeature: feat.ID,
		Slug:          "nil-result-task",
		Summary:       "Test task",
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	result, err := svc.UpdateStatus(UpdateStatusInput{
		Type:   "task",
		ID:     task.ID,
		Slug:   task.Slug,
		Status: "ready",
	})
	if err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	if result.WorktreeHookResult != nil {
		t.Errorf("WorktreeHookResult should be nil when hook returns nil, got %+v", result.WorktreeHookResult)
	}

	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 hook call, got %d", len(mock.calls))
	}
}

func TestUpdateStatus_HookCalledForBugTransition(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-06-01T12:00:00Z")
	mock := &mockStatusTransitionHook{}
	svc.SetStatusTransitionHook(mock)

	bug, err := svc.CreateBug(CreateBugInput{
		Slug:       "test-bug",
		Name:      "A test bug",
		ReportedBy: "tester",
		Observed:   "Something broke",
		Expected:   "Should not break",
		Severity:   "medium",
		Priority:   "medium",
		Type:       "implementation-defect",
	})
	if err != nil {
		t.Fatalf("CreateBug: %v", err)
	}

	// reported → triaged
	_, err = svc.UpdateStatus(UpdateStatusInput{
		Type:   "bug",
		ID:     bug.ID,
		Slug:   bug.Slug,
		Status: "triaged",
	})
	if err != nil {
		t.Fatalf("UpdateStatus reported→triaged: %v", err)
	}

	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 hook call after triaged, got %d", len(mock.calls))
	}

	call := mock.calls[0]
	if call.entityType != "bug" {
		t.Errorf("hook entityType = %q, want %q", call.entityType, "bug")
	}
	if call.toStatus != "triaged" {
		t.Errorf("hook toStatus = %q, want %q", call.toStatus, "triaged")
	}
}

func TestUpdateStatus_HookNotCalledOnFailedTransition(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-06-01T12:00:00Z")
	mock := &mockStatusTransitionHook{}
	svc.SetStatusTransitionHook(mock)

	writeTestPlan(t, svc, testPlanID)
	feat, err := svc.CreateFeature(CreateFeatureInput{
		Name: "test",
		Slug:      "invalid-transition-test",
		Parent:    testPlanID,
		Summary:   "Test invalid transition",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature: %v", err)
	}

	task, err := svc.CreateTask(CreateTaskInput{
		Name: "test",
		ParentFeature: feat.ID,
		Slug:          "invalid-transition-task",
		Summary:       "Test task",
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	// Try invalid transition: queued → active (should fail, must go through ready first)
	_, err = svc.UpdateStatus(UpdateStatusInput{
		Type:   "task",
		ID:     task.ID,
		Slug:   task.Slug,
		Status: "active",
	})
	if err == nil {
		t.Fatal("expected error for invalid transition queued→active, got nil")
	}

	if len(mock.calls) != 0 {
		t.Errorf("hook should not be called on failed transition, got %d calls", len(mock.calls))
	}
}

// --- Tests for WorktreeTransitionHook logic ---

func TestWorktreeTransitionHook_OnStatusTransition_IgnoresIrrelevantTransitions(t *testing.T) {
	t.Parallel()

	// The hook should return nil for transitions that don't trigger worktree creation.
	hook := &WorktreeTransitionHook{} // nil deps are fine since we won't reach them

	cases := []struct {
		name       string
		entityType string
		toStatus   string
	}{
		{"task_to_ready", "task", "ready"},
		{"task_to_blocked", "task", "blocked"},
		{"task_to_done", "task", "done"},
		{"task_to_needs-review", "task", "needs-review"},
		{"bug_to_triaged", "bug", "triaged"},
		{"bug_to_reproduced", "bug", "reproduced"},
		{"bug_to_planned", "bug", "planned"},
		{"bug_to_needs-review", "bug", "needs-review"},
		{"bug_to_closed", "bug", "closed"},
		{"feature_to_developing", "feature", "developing"},
		{"decision_to_decided", "decision", "decided"},
		{"epic_to_active", "epic", "active"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := hook.OnStatusTransition(tc.entityType, "ID-123", "slug", "prev", tc.toStatus, nil)
			if result != nil {
				t.Errorf("expected nil result for %s→%s, got %+v", tc.entityType, tc.toStatus, result)
			}
		})
	}
}

func TestWorktreeTransitionHook_TaskToActive_NoParentFeature(t *testing.T) {
	t.Parallel()

	hook := &WorktreeTransitionHook{} // deps not needed for this path

	state := map[string]any{
		"id":     "TASK-ABC",
		"slug":   "test-task",
		"status": "active",
		// no parent_feature
	}

	result := hook.OnStatusTransition("task", "TASK-ABC", "test-task", "ready", "active", state)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Created {
		t.Error("should not have created a worktree")
	}
	if result.Warning == "" {
		t.Error("expected a warning about missing feature association")
	}
	if !strings.Contains(result.Warning, "not associated with a feature") {
		t.Errorf("warning should mention missing feature association, got %q", result.Warning)
	}
}

func TestWorktreeTransitionHook_TaskToActive_ParentFeatureNotFound(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-06-01T12:00:00Z")

	// Don't create the feature — it should fail to load
	hook := &WorktreeTransitionHook{entitySvc: svc}

	state := map[string]any{
		"id":             "TASK-ABC",
		"slug":           "test-task",
		"status":         "active",
		"parent_feature": "FEAT-NONEXISTENT",
	}

	result := hook.OnStatusTransition("task", "TASK-ABC", "test-task", "ready", "active", state)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Created {
		t.Error("should not have created a worktree")
	}
	if result.Warning == "" {
		t.Error("expected a warning about feature not found")
	}
	if !strings.Contains(result.Warning, "could not load parent feature") {
		t.Errorf("warning should mention feature load failure, got %q", result.Warning)
	}
}

func TestWorktreeTransitionHook_TaskToActive_CreatesWorktreeForFeature(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	initTestGitRepo(t, repoRoot)

	stateRoot := filepath.Join(repoRoot, ".kbz", "state")
	if err := os.MkdirAll(stateRoot, 0755); err != nil {
		t.Fatal(err)
	}

	svc := newTestEntityService(stateRoot, "2026-06-01T12:00:00Z")
	writeTestPlan(t, svc, testPlanID)

	feat, err := svc.CreateFeature(CreateFeatureInput{
		Name: "test",
		Slug:      "auto-worktree",
		Parent:    testPlanID,
		Summary:   "Feature for auto worktree",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature: %v", err)
	}

	wkStore := newWorktreeStore(t, stateRoot)
	wkGit := newWorktreeGit(t, repoRoot)

	hook := NewWorktreeTransitionHook(wkStore, wkGit, svc)

	state := map[string]any{
		"id":             "TASK-ABC",
		"slug":           "test-task",
		"status":         "active",
		"parent_feature": feat.ID,
	}

	result := hook.OnStatusTransition("task", "TASK-ABC", "test-task", "ready", "active", state)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.Created {
		t.Fatalf("expected worktree to be created, got warning: %q", result.Warning)
	}
	if result.WorktreeID == "" {
		t.Error("WorktreeID should not be empty")
	}
	if result.EntityID != feat.ID {
		t.Errorf("EntityID = %q, want %q", result.EntityID, feat.ID)
	}
	if !strings.HasPrefix(result.Branch, "feature/") {
		t.Errorf("Branch = %q, want prefix 'feature/'", result.Branch)
	}
	if !strings.Contains(result.Branch, feat.ID) {
		t.Errorf("Branch = %q, should contain feature ID %q", result.Branch, feat.ID)
	}
	if result.Path == "" {
		t.Error("Path should not be empty")
	}
	if result.Warning != "" {
		t.Errorf("unexpected warning: %q", result.Warning)
	}

	// Verify worktree record was persisted
	record, err := wkStore.GetByEntityID(feat.ID)
	if err != nil {
		t.Fatalf("worktree record not found for entity %s: %v", feat.ID, err)
	}
	if record.Branch != result.Branch {
		t.Errorf("persisted branch = %q, want %q", record.Branch, result.Branch)
	}
	if string(record.Status) != "active" {
		t.Errorf("persisted status = %q, want %q", record.Status, "active")
	}
}

func TestWorktreeTransitionHook_TaskToActive_IdempotentWhenWorktreeExists(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	initTestGitRepo(t, repoRoot)

	stateRoot := filepath.Join(repoRoot, ".kbz", "state")
	if err := os.MkdirAll(stateRoot, 0755); err != nil {
		t.Fatal(err)
	}

	svc := newTestEntityService(stateRoot, "2026-06-01T12:00:00Z")
	writeTestPlan(t, svc, testPlanID)

	feat, err := svc.CreateFeature(CreateFeatureInput{
		Name: "test",
		Slug:      "idempotent-wt",
		Parent:    testPlanID,
		Summary:   "Feature for idempotent test",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature: %v", err)
	}

	wkStore := newWorktreeStore(t, stateRoot)
	wkGit := newWorktreeGit(t, repoRoot)

	hook := NewWorktreeTransitionHook(wkStore, wkGit, svc)

	state := map[string]any{
		"id":             "TASK-001",
		"slug":           "first-task",
		"status":         "active",
		"parent_feature": feat.ID,
	}

	// First activation — should create worktree
	result1 := hook.OnStatusTransition("task", "TASK-001", "first-task", "ready", "active", state)
	if result1 == nil || !result1.Created {
		t.Fatalf("first call should create worktree, got %+v", result1)
	}

	// Second activation (e.g., a second task becoming active) — should find existing worktree
	state2 := map[string]any{
		"id":             "TASK-002",
		"slug":           "second-task",
		"status":         "active",
		"parent_feature": feat.ID,
	}

	result2 := hook.OnStatusTransition("task", "TASK-002", "second-task", "ready", "active", state2)
	if result2 == nil {
		t.Fatal("expected non-nil result on second call")
	}
	if result2.Created {
		t.Error("second call should not create a new worktree")
	}
	if !result2.AlreadyExists {
		t.Error("second call should indicate worktree already exists")
	}
	if result2.WorktreeID != result1.WorktreeID {
		t.Errorf("existing worktree ID = %q, want %q", result2.WorktreeID, result1.WorktreeID)
	}
}

func TestWorktreeTransitionHook_BugToInProgress_CreatesWorktreeForBug(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	initTestGitRepo(t, repoRoot)

	stateRoot := filepath.Join(repoRoot, ".kbz", "state")
	if err := os.MkdirAll(stateRoot, 0755); err != nil {
		t.Fatal(err)
	}

	svc := newTestEntityService(stateRoot, "2026-06-01T12:00:00Z")

	bug, err := svc.CreateBug(CreateBugInput{
		Slug:       "auto-wt-bug",
		Name:      "Bug for auto worktree",
		ReportedBy: "tester",
		Observed:   "Something broke",
		Expected:   "Should not break",
		Severity:   "medium",
		Priority:   "medium",
		Type:       "implementation-defect",
	})
	if err != nil {
		t.Fatalf("CreateBug: %v", err)
	}

	wkStore := newWorktreeStore(t, stateRoot)
	wkGit := newWorktreeGit(t, repoRoot)

	hook := NewWorktreeTransitionHook(wkStore, wkGit, svc)

	bugState := map[string]any{
		"id":     bug.ID,
		"slug":   bug.Slug,
		"status": "in-progress",
	}

	result := hook.OnStatusTransition("bug", bug.ID, bug.Slug, "planned", "in-progress", bugState)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.Created {
		t.Fatalf("expected worktree to be created for bug, got warning: %q", result.Warning)
	}
	if result.EntityID != bug.ID {
		t.Errorf("EntityID = %q, want %q", result.EntityID, bug.ID)
	}
	if !strings.HasPrefix(result.Branch, "bug/") {
		t.Errorf("Branch = %q, want prefix 'bug/'", result.Branch)
	}
	if !strings.Contains(result.Branch, bug.ID) {
		t.Errorf("Branch = %q, should contain bug ID %q", result.Branch, bug.ID)
	}

	// Verify worktree record was persisted
	record, err := wkStore.GetByEntityID(bug.ID)
	if err != nil {
		t.Fatalf("worktree record not found for bug %s: %v", bug.ID, err)
	}
	if record.Branch != result.Branch {
		t.Errorf("persisted branch = %q, want %q", record.Branch, result.Branch)
	}
}

func TestWorktreeTransitionHook_BugToInProgress_IdempotentWhenWorktreeExists(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	initTestGitRepo(t, repoRoot)

	stateRoot := filepath.Join(repoRoot, ".kbz", "state")
	if err := os.MkdirAll(stateRoot, 0755); err != nil {
		t.Fatal(err)
	}

	svc := newTestEntityService(stateRoot, "2026-06-01T12:00:00Z")

	bug, err := svc.CreateBug(CreateBugInput{
		Slug:       "idempotent-bug",
		Name:      "Bug for idempotent test",
		ReportedBy: "tester",
		Observed:   "Broken",
		Expected:   "Not broken",
		Severity:   "low",
		Priority:   "low",
		Type:       "implementation-defect",
	})
	if err != nil {
		t.Fatalf("CreateBug: %v", err)
	}

	wkStore := newWorktreeStore(t, stateRoot)
	wkGit := newWorktreeGit(t, repoRoot)

	hook := NewWorktreeTransitionHook(wkStore, wkGit, svc)

	bugState := map[string]any{
		"id":     bug.ID,
		"slug":   bug.Slug,
		"status": "in-progress",
	}

	result1 := hook.OnStatusTransition("bug", bug.ID, bug.Slug, "planned", "in-progress", bugState)
	if result1 == nil || !result1.Created {
		t.Fatalf("first call should create worktree, got %+v", result1)
	}

	// Simulate a rework loop: in-progress → needs-review → needs-rework → in-progress
	result2 := hook.OnStatusTransition("bug", bug.ID, bug.Slug, "needs-rework", "in-progress", bugState)
	if result2 == nil {
		t.Fatal("expected non-nil result on second call")
	}
	if result2.Created {
		t.Error("should not create a second worktree")
	}
	if !result2.AlreadyExists {
		t.Error("should indicate worktree already exists")
	}
}

func TestWorktreeTransitionHook_CaseInsensitiveEntityType(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	initTestGitRepo(t, repoRoot)

	stateRoot := filepath.Join(repoRoot, ".kbz", "state")
	if err := os.MkdirAll(stateRoot, 0755); err != nil {
		t.Fatal(err)
	}

	svc := newTestEntityService(stateRoot, "2026-06-01T12:00:00Z")

	bug, err := svc.CreateBug(CreateBugInput{
		Slug:       "case-test-bug",
		Name:      "Case test bug",
		ReportedBy: "tester",
		Observed:   "Broken",
		Expected:   "Works",
		Severity:   "low",
		Priority:   "low",
		Type:       "implementation-defect",
	})
	if err != nil {
		t.Fatalf("CreateBug: %v", err)
	}

	wkStore := newWorktreeStore(t, stateRoot)
	wkGit := newWorktreeGit(t, repoRoot)

	hook := NewWorktreeTransitionHook(wkStore, wkGit, svc)

	// Use uppercase "Bug" — should still trigger
	bugState := map[string]any{
		"id":     bug.ID,
		"slug":   bug.Slug,
		"status": "in-progress",
	}

	result := hook.OnStatusTransition("Bug", bug.ID, bug.Slug, "planned", "in-progress", bugState)
	if result == nil {
		t.Fatal("expected non-nil result with uppercase entity type")
	}
	if !result.Created {
		t.Fatalf("expected worktree creation with uppercase entity type, got warning: %q", result.Warning)
	}
}

// --- End-to-end test: UpdateStatus with real hook ---

func TestUpdateStatus_EndToEnd_TaskActivationCreatesWorktree(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	initTestGitRepo(t, repoRoot)

	stateRoot := filepath.Join(repoRoot, ".kbz", "state")
	if err := os.MkdirAll(stateRoot, 0755); err != nil {
		t.Fatal(err)
	}

	svc := newTestEntityService(stateRoot, "2026-06-01T12:00:00Z")

	wkStore := newWorktreeStore(t, stateRoot)
	wkGit := newWorktreeGit(t, repoRoot)

	svc.SetStatusTransitionHook(NewWorktreeTransitionHook(wkStore, wkGit, svc))

	// Create plan, feature and task
	writeTestPlan(t, svc, testPlanID)
	feat, err := svc.CreateFeature(CreateFeatureInput{
		Name: "test",
		Slug:      "e2e-auto-wt",
		Parent:    testPlanID,
		Summary:   "E2E auto worktree",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature: %v", err)
	}

	task, err := svc.CreateTask(CreateTaskInput{
		Name: "test",
		ParentFeature: feat.ID,
		Slug:          "e2e-task",
		Summary:       "E2E task",
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	// queued → ready (should not create worktree)
	result, err := svc.UpdateStatus(UpdateStatusInput{
		Type:   "task",
		ID:     task.ID,
		Slug:   task.Slug,
		Status: "ready",
	})
	if err != nil {
		t.Fatalf("UpdateStatus queued→ready: %v", err)
	}
	if result.WorktreeHookResult != nil && result.WorktreeHookResult.Created {
		t.Error("worktree should not be created on queued→ready")
	}

	// ready → active (should create worktree)
	result, err = svc.UpdateStatus(UpdateStatusInput{
		Type:   "task",
		ID:     task.ID,
		Slug:   task.Slug,
		Status: "active",
	})
	if err != nil {
		t.Fatalf("UpdateStatus ready→active: %v", err)
	}

	if result.WorktreeHookResult == nil {
		t.Fatal("WorktreeHookResult is nil after task→active")
	}
	if !result.WorktreeHookResult.Created {
		t.Fatalf("expected worktree creation on task→active, got warning: %q", result.WorktreeHookResult.Warning)
	}
	if result.WorktreeHookResult.EntityID != feat.ID {
		t.Errorf("worktree entity = %q, want parent feature %q", result.WorktreeHookResult.EntityID, feat.ID)
	}

	// Verify worktree exists in store
	record, err := wkStore.GetByEntityID(feat.ID)
	if err != nil {
		t.Fatalf("worktree not found in store: %v", err)
	}
	if string(record.Status) != "active" {
		t.Errorf("worktree status = %q, want 'active'", record.Status)
	}
}

func TestUpdateStatus_EndToEnd_BugInProgressCreatesWorktree(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	initTestGitRepo(t, repoRoot)

	stateRoot := filepath.Join(repoRoot, ".kbz", "state")
	if err := os.MkdirAll(stateRoot, 0755); err != nil {
		t.Fatal(err)
	}

	svc := newTestEntityService(stateRoot, "2026-06-01T12:00:00Z")

	wkStore := newWorktreeStore(t, stateRoot)
	wkGit := newWorktreeGit(t, repoRoot)

	svc.SetStatusTransitionHook(NewWorktreeTransitionHook(wkStore, wkGit, svc))

	bug, err := svc.CreateBug(CreateBugInput{
		Slug:       "e2e-bug",
		Name:      "E2E bug",
		ReportedBy: "tester",
		Observed:   "Broken",
		Expected:   "Works",
		Severity:   "medium",
		Priority:   "medium",
		Type:       "implementation-defect",
	})
	if err != nil {
		t.Fatalf("CreateBug: %v", err)
	}

	// Walk the bug through its lifecycle to in-progress
	transitions := []string{"triaged", "reproduced", "planned", "in-progress"}
	var lastResult GetResult

	for i, status := range transitions {
		result, err := svc.UpdateStatus(UpdateStatusInput{
			Type:   "bug",
			ID:     bug.ID,
			Slug:   bug.Slug,
			Status: status,
		})
		if err != nil {
			t.Fatalf("UpdateStatus to %s: %v", status, err)
		}

		if i < len(transitions)-1 {
			// Before in-progress, no worktree should be created
			if result.WorktreeHookResult != nil && result.WorktreeHookResult.Created {
				t.Errorf("worktree should not be created on transition to %s", status)
			}
		}

		lastResult = result
	}

	// The final transition to in-progress should have created the worktree
	if lastResult.WorktreeHookResult == nil {
		t.Fatal("WorktreeHookResult is nil after bug→in-progress")
	}
	if !lastResult.WorktreeHookResult.Created {
		t.Fatalf("expected worktree creation on bug→in-progress, got warning: %q", lastResult.WorktreeHookResult.Warning)
	}
	if lastResult.WorktreeHookResult.EntityID != bug.ID {
		t.Errorf("worktree entity = %q, want bug %q", lastResult.WorktreeHookResult.EntityID, bug.ID)
	}
	if !strings.HasPrefix(lastResult.WorktreeHookResult.Branch, "bug/") {
		t.Errorf("branch = %q, want 'bug/' prefix", lastResult.WorktreeHookResult.Branch)
	}

	// Verify worktree record in store
	record, err := wkStore.GetByEntityID(bug.ID)
	if err != nil {
		t.Fatalf("worktree not found in store for bug: %v", err)
	}
	if string(record.Status) != "active" {
		t.Errorf("worktree status = %q, want 'active'", record.Status)
	}
}

func TestUpdateStatus_EndToEnd_SecondTaskDoesNotDuplicateWorktree(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	initTestGitRepo(t, repoRoot)

	stateRoot := filepath.Join(repoRoot, ".kbz", "state")
	if err := os.MkdirAll(stateRoot, 0755); err != nil {
		t.Fatal(err)
	}

	svc := newTestEntityService(stateRoot, "2026-06-01T12:00:00Z")

	wkStore := newWorktreeStore(t, stateRoot)
	wkGit := newWorktreeGit(t, repoRoot)

	svc.SetStatusTransitionHook(NewWorktreeTransitionHook(wkStore, wkGit, svc))

	writeTestPlan(t, svc, testPlanID)
	feat, err := svc.CreateFeature(CreateFeatureInput{
		Name: "test",
		Slug:      "multi-task-wt",
		Parent:    testPlanID,
		Summary:   "Feature with multiple tasks",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature: %v", err)
	}

	// Create two tasks
	task1, err := svc.CreateTask(CreateTaskInput{
		Name: "test",
		ParentFeature: feat.ID,
		Slug:          "task-one",
		Summary:       "First task",
	})
	if err != nil {
		t.Fatalf("CreateTask 1: %v", err)
	}

	task2, err := svc.CreateTask(CreateTaskInput{
		Name: "test",
		ParentFeature: feat.ID,
		Slug:          "task-two",
		Summary:       "Second task",
	})
	if err != nil {
		t.Fatalf("CreateTask 2: %v", err)
	}

	// Activate task 1
	_, err = svc.UpdateStatus(UpdateStatusInput{
		Type: "task", ID: task1.ID, Slug: task1.Slug, Status: "ready",
	})
	if err != nil {
		t.Fatalf("task1 queued→ready: %v", err)
	}
	result1, err := svc.UpdateStatus(UpdateStatusInput{
		Type: "task", ID: task1.ID, Slug: task1.Slug, Status: "active",
	})
	if err != nil {
		t.Fatalf("task1 ready→active: %v", err)
	}

	if result1.WorktreeHookResult == nil || !result1.WorktreeHookResult.Created {
		t.Fatal("first task activation should create worktree")
	}

	firstWT := result1.WorktreeHookResult.WorktreeID

	// Activate task 2 — should not create a second worktree
	_, err = svc.UpdateStatus(UpdateStatusInput{
		Type: "task", ID: task2.ID, Slug: task2.Slug, Status: "ready",
	})
	if err != nil {
		t.Fatalf("task2 queued→ready: %v", err)
	}
	result2, err := svc.UpdateStatus(UpdateStatusInput{
		Type: "task", ID: task2.ID, Slug: task2.Slug, Status: "active",
	})
	if err != nil {
		t.Fatalf("task2 ready→active: %v", err)
	}

	if result2.WorktreeHookResult == nil {
		t.Fatal("expected WorktreeHookResult on second task activation")
	}
	if result2.WorktreeHookResult.Created {
		t.Error("second task activation should not create another worktree")
	}
	if !result2.WorktreeHookResult.AlreadyExists {
		t.Error("should indicate existing worktree")
	}
	if result2.WorktreeHookResult.WorktreeID != firstWT {
		t.Errorf("existing worktree ID = %q, want %q", result2.WorktreeHookResult.WorktreeID, firstWT)
	}
}

func TestUpdateStatus_EndToEnd_FeatureTransitionDoesNotTriggerWorktree(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := newTestEntityService(root, "2026-06-01T12:00:00Z")
	mock := &mockStatusTransitionHook{}
	svc.SetStatusTransitionHook(mock)

	writeTestPlan(t, svc, testPlanID)
	feat, err := svc.CreateFeature(CreateFeatureInput{
		Name: "test",
		Slug:      "feature-transition-test",
		Parent:    testPlanID,
		Summary:   "Feature transition test",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("CreateFeature: %v", err)
	}

	// A feature going to "developing" should fire the hook but the hook
	// should not create a worktree (only task→active and bug→in-progress do)
	_, err = svc.UpdateStatus(UpdateStatusInput{
		Type:   "feature",
		ID:     feat.ID,
		Slug:   feat.Slug,
		Status: "designing",
	})
	if err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	if len(mock.calls) != 1 {
		t.Fatalf("expected 1 hook call, got %d", len(mock.calls))
	}
	if mock.calls[0].entityType != "feature" {
		t.Errorf("hook entityType = %q, want %q", mock.calls[0].entityType, "feature")
	}
	if mock.calls[0].toStatus != "designing" {
		t.Errorf("hook toStatus = %q, want %q", mock.calls[0].toStatus, "designing")
	}
}

// --- Helpers that bridge to the worktree package ---

func newWorktreeStore(t *testing.T, stateRoot string) *worktree.Store {
	t.Helper()
	return worktree.NewStore(stateRoot)
}

func newWorktreeGit(t *testing.T, repoRoot string) *worktree.Git {
	t.Helper()
	return worktree.NewGit(repoRoot)
}
