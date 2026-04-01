package service

import (
	"strings"
	"testing"

	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/storage"
)

func newDispatchTestServices(t *testing.T) (*EntityService, *KnowledgeService, *DispatchService) {
	t.Helper()
	root := t.TempDir()
	entitySvc := NewEntityService(root)
	knowledgeSvc := NewKnowledgeService(root)
	dispatchSvc := NewDispatchService(entitySvc, knowledgeSvc)
	return entitySvc, knowledgeSvc, dispatchSvc
}

// writeDispatchTestPlan creates a Plan entity directly on disk for dispatch tests.
func writeDispatchTestPlan(t *testing.T, svc *EntityService, id string) {
	t.Helper()
	_, _, slug := model.ParsePlanID(id)
	fields := map[string]any{
		"id":         id,
		"slug":       slug,
		"title":      "Test Plan",
		"status":     "active",
		"summary":    "Test plan for dispatch unit tests",
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
		t.Fatalf("writeDispatchTestPlan(%s) error = %v", id, err)
	}
}

// createReadyTask creates a task in ready status for dispatch tests.
func createReadyTask(t *testing.T, entitySvc *EntityService, slug string) string {
	t.Helper()

	planID := "P1-dispatch-plan"
	writeDispatchTestPlan(t, entitySvc, planID)

	fResult, err := entitySvc.CreateFeature(CreateFeatureInput{
		Name: "test",
		Slug:      "feat-" + slug,
		Parent:    planID,
		Summary:   "Feature for " + slug,
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature: %v", err)
	}

	tResult, err := entitySvc.CreateTask(CreateTaskInput{
		Name: "test",
		ParentFeature: fResult.ID,
		Slug:          slug,
		Summary:       "Task " + slug,
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	// Promote to ready.
	_, err = entitySvc.UpdateStatus(UpdateStatusInput{
		Type:   "task",
		ID:     tResult.ID,
		Slug:   tResult.Slug,
		Status: string(model.TaskStatusReady),
	})
	if err != nil {
		t.Fatalf("UpdateStatus queued→ready: %v", err)
	}

	return tResult.ID
}

// TestDispatchTask_Success verifies §15.3: dispatch_task success — task transitions to active,
// all four dispatch fields set.
func TestDispatchTask_Success(t *testing.T) {
	entitySvc, _, dispatchSvc := newDispatchTestServices(t)

	taskID := createReadyTask(t, entitySvc, "dispatch-success")

	result, err := dispatchSvc.DispatchTask(DispatchInput{
		TaskID:       taskID,
		Role:         "backend",
		DispatchedBy: "orchestrator-session-abc",
	})
	if err != nil {
		t.Fatalf("DispatchTask: %v", err)
	}

	status, _ := result.Task["status"].(string)
	if status != string(model.TaskStatusActive) {
		t.Errorf("status: got %q, want active", status)
	}

	if result.Task["claimed_at"] == nil || result.Task["claimed_at"] == "" {
		t.Error("claimed_at not set after dispatch")
	}
	if dt, _ := result.Task["dispatched_to"].(string); dt != "backend" {
		t.Errorf("dispatched_to: got %q, want backend", dt)
	}
	if result.Task["dispatched_at"] == nil || result.Task["dispatched_at"] == "" {
		t.Error("dispatched_at not set after dispatch")
	}
	if db, _ := result.Task["dispatched_by"].(string); db != "orchestrator-session-abc" {
		t.Errorf("dispatched_by: got %q, want orchestrator-session-abc", db)
	}
}

// TestDispatchTask_AlreadyClaimed verifies §15.3: dispatch_task already-claimed returns error
// with dispatched_by and claimed_at in message; task state unchanged.
func TestDispatchTask_AlreadyClaimed(t *testing.T) {
	entitySvc, _, dispatchSvc := newDispatchTestServices(t)

	taskID := createReadyTask(t, entitySvc, "already-claimed")

	// First dispatch succeeds.
	_, err := dispatchSvc.DispatchTask(DispatchInput{
		TaskID:       taskID,
		Role:         "backend",
		DispatchedBy: "orchestrator-session-abc",
	})
	if err != nil {
		t.Fatalf("first DispatchTask: %v", err)
	}

	// Second dispatch on the same task should fail with "already claimed".
	_, err = dispatchSvc.DispatchTask(DispatchInput{
		TaskID:       taskID,
		Role:         "backend",
		DispatchedBy: "orchestrator-session-xyz",
	})
	if err == nil {
		t.Fatal("expected error for already-claimed task, got nil")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "already claimed") {
		t.Errorf("error should mention 'already claimed', got: %q", errMsg)
	}
	if !strings.Contains(errMsg, "orchestrator-session-abc") {
		t.Errorf("error should name original dispatcher, got: %q", errMsg)
	}
}

// TestDispatchTask_NonReadyStatus verifies §15.3: dispatch_task non-ready status returns
// clear error naming the actual status.
func TestDispatchTask_NonReadyStatus(t *testing.T) {
	entitySvc, _, dispatchSvc := newDispatchTestServices(t)

	planID := "P1-non-ready-plan"
	writeDispatchTestPlan(t, entitySvc, planID)

	fResult, err := entitySvc.CreateFeature(CreateFeatureInput{
		Name: "test",
		Slug:      "feat-non-ready",
		Parent:    planID,
		Summary:   "Feature for non-ready test",
		CreatedBy: "test",
	})
	if err != nil {
		t.Fatalf("CreateFeature: %v", err)
	}

	tResult, err := entitySvc.CreateTask(CreateTaskInput{
		Name: "test",
		ParentFeature: fResult.ID,
		Slug:          "non-ready-task",
		Summary:       "Task in queued state",
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	// Attempt to dispatch a queued task (not ready).
	_, err = dispatchSvc.DispatchTask(DispatchInput{
		TaskID:       tResult.ID,
		Role:         "backend",
		DispatchedBy: "orchestrator",
	})
	if err == nil {
		t.Fatal("expected error for non-ready task, got nil")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "queued") {
		t.Errorf("error should name the actual status 'queued', got: %q", errMsg)
	}
}

// TestDispatchTask_RequiredParams verifies validation of required parameters.
func TestDispatchTask_RequiredParams(t *testing.T) {
	_, _, dispatchSvc := newDispatchTestServices(t)

	tests := []struct {
		name    string
		input   DispatchInput
		wantErr string
	}{
		{
			name:    "missing task_id",
			input:   DispatchInput{TaskID: "", Role: "backend", DispatchedBy: "agent"},
			wantErr: "task_id",
		},
		{
			name:    "missing role",
			input:   DispatchInput{TaskID: "TASK-001", Role: "", DispatchedBy: "agent"},
			wantErr: "role",
		},
		{
			name:    "missing dispatched_by",
			input:   DispatchInput{TaskID: "TASK-001", Role: "backend", DispatchedBy: ""},
			wantErr: "dispatched_by",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := dispatchSvc.DispatchTask(tc.input)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("expected error to mention %q, got: %q", tc.wantErr, err.Error())
			}
		})
	}
}

// TestCompleteTask_ToDone verifies §15.3: complete_task to done —
// task status = done, completed set, completion_summary stored.
func TestCompleteTask_ToDone(t *testing.T) {
	entitySvc, _, dispatchSvc := newDispatchTestServices(t)

	taskID := createReadyTask(t, entitySvc, "complete-to-done")

	// Dispatch first to make it active.
	_, err := dispatchSvc.DispatchTask(DispatchInput{
		TaskID:       taskID,
		Role:         "backend",
		DispatchedBy: "orchestrator",
	})
	if err != nil {
		t.Fatalf("DispatchTask: %v", err)
	}

	// Complete the task.
	result, err := dispatchSvc.CompleteTask(CompleteInput{
		TaskID:  taskID,
		Summary: "Implemented the feature with full test coverage.",
	})
	if err != nil {
		t.Fatalf("CompleteTask: %v", err)
	}

	status, _ := result.Task["status"].(string)
	if status != string(model.TaskStatusDone) {
		t.Errorf("status: got %q, want done", status)
	}

	if result.Task["completed"] == nil || result.Task["completed"] == "" {
		t.Error("completed timestamp not set")
	}

	summary, _ := result.Task["completion_summary"].(string)
	if summary != "Implemented the feature with full test coverage." {
		t.Errorf("completion_summary: got %q", summary)
	}
}

// TestCompleteTask_ToNeedsReview verifies §15.3: complete_task to needs-review.
func TestCompleteTask_ToNeedsReview(t *testing.T) {
	entitySvc, _, dispatchSvc := newDispatchTestServices(t)

	taskID := createReadyTask(t, entitySvc, "complete-needs-review")

	_, err := dispatchSvc.DispatchTask(DispatchInput{
		TaskID:       taskID,
		Role:         "backend",
		DispatchedBy: "orchestrator",
	})
	if err != nil {
		t.Fatalf("DispatchTask: %v", err)
	}

	result, err := dispatchSvc.CompleteTask(CompleteInput{
		TaskID:   taskID,
		Summary:  "Work done; please review.",
		ToStatus: "needs-review",
	})
	if err != nil {
		t.Fatalf("CompleteTask: %v", err)
	}

	status, _ := result.Task["status"].(string)
	if status != string(model.TaskStatusNeedsReview) {
		t.Errorf("status: got %q, want needs-review", status)
	}
}

// TestCompleteTask_InvalidToStatus verifies that an invalid to_status is rejected.
func TestCompleteTask_InvalidToStatus(t *testing.T) {
	entitySvc, _, dispatchSvc := newDispatchTestServices(t)

	taskID := createReadyTask(t, entitySvc, "complete-invalid-status")

	_, err := dispatchSvc.DispatchTask(DispatchInput{
		TaskID:       taskID,
		Role:         "backend",
		DispatchedBy: "orchestrator",
	})
	if err != nil {
		t.Fatalf("DispatchTask: %v", err)
	}

	_, err = dispatchSvc.CompleteTask(CompleteInput{
		TaskID:   taskID,
		Summary:  "Done.",
		ToStatus: "cancelled", // invalid
	})
	if err == nil {
		t.Fatal("expected error for invalid to_status, got nil")
	}
}

// TestCompleteTask_NotActive verifies §15.3: complete_task on non-active task returns error.
func TestCompleteTask_NotActive(t *testing.T) {
	entitySvc, _, dispatchSvc := newDispatchTestServices(t)

	taskID := createReadyTask(t, entitySvc, "complete-not-active")

	// Try to complete a ready task (never dispatched).
	_, err := dispatchSvc.CompleteTask(CompleteInput{
		TaskID:  taskID,
		Summary: "Done.",
	})
	if err == nil {
		t.Fatal("expected error for non-active task, got nil")
	}
	if !strings.Contains(err.Error(), "active") {
		t.Errorf("error should mention 'active', got: %q", err.Error())
	}
}

// TestCompleteTask_KnowledgeBatch verifies §15.3: knowledge batch — valid entries
// contributed, duplicates rejected with reason, task completes regardless.
func TestCompleteTask_KnowledgeBatch(t *testing.T) {
	entitySvc, _, dispatchSvc := newDispatchTestServices(t)

	taskID := createReadyTask(t, entitySvc, "complete-knowledge-batch")

	_, err := dispatchSvc.DispatchTask(DispatchInput{
		TaskID:       taskID,
		Role:         "backend",
		DispatchedBy: "orchestrator",
	})
	if err != nil {
		t.Fatalf("DispatchTask: %v", err)
	}

	result, err := dispatchSvc.CompleteTask(CompleteInput{
		TaskID:  taskID,
		Summary: "Implemented feature.",
		KnowledgeEntries: []KnowledgeEntryInput{
			{
				Topic:   "jwt-rs256-key-rotation",
				Content: "RS256 key rotation can be handled via JWKS endpoint without service restart.",
				Scope:   "backend",
				Tier:    3,
			},
			{
				Topic:   "jwt-rs256-key-rotation", // duplicate topic — should be rejected
				Content: "Duplicate content about RS256 key rotation.",
				Scope:   "backend",
				Tier:    3,
			},
		},
	})
	if err != nil {
		t.Fatalf("CompleteTask: %v", err)
	}

	// Task should be done regardless of knowledge batch outcome.
	status, _ := result.Task["status"].(string)
	if status != string(model.TaskStatusDone) {
		t.Errorf("task status: got %q, want done", status)
	}

	if result.KnowledgeContributions.TotalAttempted != 2 {
		t.Errorf("total_attempted: got %d, want 2", result.KnowledgeContributions.TotalAttempted)
	}
	if result.KnowledgeContributions.TotalAccepted != 1 {
		t.Errorf("total_accepted: got %d, want 1", result.KnowledgeContributions.TotalAccepted)
	}
	if len(result.KnowledgeContributions.Accepted) != 1 {
		t.Errorf("accepted count: got %d, want 1", len(result.KnowledgeContributions.Accepted))
	}
	if len(result.KnowledgeContributions.Rejected) != 1 {
		t.Errorf("rejected count: got %d, want 1", len(result.KnowledgeContributions.Rejected))
	}
}

// TestCompleteTask_RequiredSummary verifies that summary is required.
func TestCompleteTask_RequiredSummary(t *testing.T) {
	_, _, dispatchSvc := newDispatchTestServices(t)

	_, err := dispatchSvc.CompleteTask(CompleteInput{
		TaskID:  "TASK-001",
		Summary: "",
	})
	if err == nil {
		t.Fatal("expected error for empty summary, got nil")
	}
	if !strings.Contains(err.Error(), "summary") {
		t.Errorf("error should mention 'summary', got: %q", err.Error())
	}
}

// TestDispatchCompleteLoop verifies the full dispatch → complete cycle
// (Verifies §15.3: CP12 — work_queue → dispatch_task → complete_task).
func TestDispatchCompleteLoop(t *testing.T) {
	entitySvc, _, dispatchSvc := newDispatchTestServices(t)

	taskID := createReadyTask(t, entitySvc, "full-loop")

	// 1. Dispatch.
	dispResult, err := dispatchSvc.DispatchTask(DispatchInput{
		TaskID:       taskID,
		Role:         "backend",
		DispatchedBy: "orchestrator",
	})
	if err != nil {
		t.Fatalf("DispatchTask: %v", err)
	}

	s, _ := dispResult.Task["status"].(string)
	if s != "active" {
		t.Errorf("after dispatch: status = %q, want active", s)
	}

	// 2. Complete.
	compResult, err := dispatchSvc.CompleteTask(CompleteInput{
		TaskID:                taskID,
		Summary:               "Full loop complete.",
		VerificationPerformed: "Unit tests pass.",
		FilesModified:         []string{"internal/foo/bar.go"},
	})
	if err != nil {
		t.Fatalf("CompleteTask: %v", err)
	}

	s, _ = compResult.Task["status"].(string)
	if s != "done" {
		t.Errorf("after complete: status = %q, want done", s)
	}

	// Verify task no longer dispatchable (it's done, not ready).
	_, err = dispatchSvc.DispatchTask(DispatchInput{
		TaskID:       taskID,
		Role:         "backend",
		DispatchedBy: "orchestrator",
	})
	if err == nil {
		t.Fatal("expected error dispatching a done task, got nil")
	}
}
