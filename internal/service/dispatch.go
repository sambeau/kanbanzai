package service

import (
	"fmt"
	"strings"
	"time"

	"kanbanzai/internal/model"
	"kanbanzai/internal/validate"
)

// DispatchInput holds parameters for dispatch_task.
type DispatchInput struct {
	TaskID       string
	Role         string
	DispatchedBy string
}

// DispatchResult holds the result of dispatch_task.
type DispatchResult struct {
	Task map[string]any
}

// KnowledgeEntryInput is a single knowledge entry to contribute during complete_task.
type KnowledgeEntryInput struct {
	Topic   string
	Content string
	Scope   string
	Tier    int
	Tags    []string
}

// CompleteInput holds parameters for complete_task.
type CompleteInput struct {
	TaskID                string
	Summary               string
	ToStatus              string // "done" or "needs-review"; default "done"
	FilesModified         []string
	VerificationPerformed string
	BlockersEncountered   string
	KnowledgeEntries      []KnowledgeEntryInput
}

// KnowledgeContributionResult is the result of knowledge entry contributions.
type KnowledgeContributionResult struct {
	Accepted []struct {
		EntryID string
		Topic   string
	}
	Rejected []struct {
		Topic  string
		Reason string
	}
	TotalAttempted int
	TotalAccepted  int
}

// CompleteResult holds the result of complete_task.
type CompleteResult struct {
	Task                   map[string]any
	KnowledgeContributions KnowledgeContributionResult
}

// DispatchService handles dispatch and completion operations.
type DispatchService struct {
	entitySvc    *EntityService
	knowledgeSvc *KnowledgeService
	now          func() time.Time
}

// NewDispatchService creates a DispatchService.
func NewDispatchService(
	entitySvc *EntityService,
	knowledgeSvc *KnowledgeService,
) *DispatchService {
	return &DispatchService{
		entitySvc:    entitySvc,
		knowledgeSvc: knowledgeSvc,
		now:          func() time.Time { return time.Now().UTC() },
	}
}

// DispatchTask atomically claims a ready task and returns its updated state.
// Context assembly (context_assemble) is the responsibility of the caller.
func (s *DispatchService) DispatchTask(input DispatchInput) (DispatchResult, error) {
	taskID := strings.TrimSpace(input.TaskID)
	role := strings.TrimSpace(input.Role)
	dispatchedBy := strings.TrimSpace(input.DispatchedBy)

	if taskID == "" {
		return DispatchResult{}, fmt.Errorf("task_id is required")
	}
	if role == "" {
		return DispatchResult{}, fmt.Errorf("role is required")
	}
	if dispatchedBy == "" {
		return DispatchResult{}, fmt.Errorf("dispatched_by is required")
	}

	// Load the task.
	task, err := s.entitySvc.Get("task", taskID, "")
	if err != nil {
		return DispatchResult{}, fmt.Errorf("task not found: %w", err)
	}

	// Verify task status is ready.
	status := stringFromState(task.State, "status")
	if status != string(model.TaskStatusReady) {
		if status == string(model.TaskStatusActive) {
			dispBy := stringFromState(task.State, "dispatched_by")
			claimedAt := stringFromState(task.State, "claimed_at")
			return DispatchResult{}, fmt.Errorf(
				"task %s is already claimed — dispatched by %s at %s",
				task.ID, dispBy, claimedAt,
			)
		}
		return DispatchResult{}, fmt.Errorf(
			"task %s cannot be dispatched: status is %q (must be ready)",
			task.ID, status,
		)
	}

	// Belt-and-suspenders dependency check.
	dependsOn := stringSliceFromState(task.State, "depends_on")
	if len(dependsOn) > 0 {
		depStatuses := make(map[string]string, len(dependsOn))
		for _, depID := range dependsOn {
			dep, err := s.entitySvc.Get("task", depID, "")
			if err == nil {
				depStatuses[depID] = stringFromState(dep.State, "status")
			}
		}
		if err := validate.ValidateTaskQueuedToReady(dependsOn, depStatuses); err != nil {
			return DispatchResult{}, fmt.Errorf("dependency check failed: %w", err)
		}
	}

	// Transition task ready → active.
	now := s.now()
	_, err = s.entitySvc.UpdateStatus(UpdateStatusInput{
		Type:   "task",
		ID:     task.ID,
		Slug:   task.Slug,
		Status: string(model.TaskStatusActive),
	})
	if err != nil {
		return DispatchResult{}, fmt.Errorf("transition task to active: %w", err)
	}

	// Set dispatch metadata fields via store directly (status already changed above).
	taskRecord, err := s.entitySvc.Store().Load("task", task.ID, task.Slug)
	if err != nil {
		return DispatchResult{}, fmt.Errorf("reload task after transition: %w", err)
	}

	taskRecord.Fields["claimed_at"] = now.Format(time.RFC3339)
	taskRecord.Fields["dispatched_to"] = role
	taskRecord.Fields["dispatched_at"] = now.Format(time.RFC3339)
	taskRecord.Fields["dispatched_by"] = dispatchedBy

	if _, err := s.entitySvc.Store().Write(taskRecord); err != nil {
		// Task is already active — don't rollback, just report.
		return DispatchResult{}, fmt.Errorf("write dispatch fields for task %s (task is now active): %w", task.ID, err)
	}

	// Reload final task state.
	finalTask, err := s.entitySvc.Get("task", task.ID, "")
	if err != nil {
		return DispatchResult{}, fmt.Errorf("reload task: %w", err)
	}

	return DispatchResult{
		Task: finalTask.State,
	}, nil
}

// CompleteTask closes the dispatch loop for a completed task.
func (s *DispatchService) CompleteTask(input CompleteInput) (CompleteResult, error) {
	taskID := strings.TrimSpace(input.TaskID)

	if taskID == "" {
		return CompleteResult{}, fmt.Errorf("task_id is required")
	}
	if strings.TrimSpace(input.Summary) == "" {
		return CompleteResult{}, fmt.Errorf("summary is required")
	}

	// Determine target status.
	toStatus := strings.TrimSpace(input.ToStatus)
	if toStatus == "" {
		toStatus = string(model.TaskStatusDone)
	}
	if toStatus != string(model.TaskStatusDone) && toStatus != string(model.TaskStatusNeedsReview) {
		return CompleteResult{}, fmt.Errorf("to_status must be %q or %q", model.TaskStatusDone, model.TaskStatusNeedsReview)
	}

	// Load task.
	task, err := s.entitySvc.Get("task", taskID, "")
	if err != nil {
		return CompleteResult{}, fmt.Errorf("task not found: %w", err)
	}

	status := stringFromState(task.State, "status")
	if status != string(model.TaskStatusActive) {
		return CompleteResult{}, fmt.Errorf(
			"task %s cannot be completed: status is %q (must be active)",
			task.ID, status,
		)
	}

	// Transition: active → done (directly) or active → needs-review.
	_, err = s.entitySvc.UpdateStatus(UpdateStatusInput{
		Type:   "task",
		ID:     task.ID,
		Slug:   task.Slug,
		Status: toStatus,
	})
	if err != nil {
		return CompleteResult{}, fmt.Errorf("transition task to %s: %w", toStatus, err)
	}

	// Set completion metadata fields via store directly.
	now := s.now()
	taskRecord, err := s.entitySvc.Store().Load("task", task.ID, task.Slug)
	if err != nil {
		return CompleteResult{}, fmt.Errorf("reload task after transition: %w", err)
	}

	taskRecord.Fields["completed"] = now.Format(time.RFC3339)
	taskRecord.Fields["completion_summary"] = input.Summary
	if len(input.FilesModified) > 0 {
		taskRecord.Fields["files_planned"] = input.FilesModified
	}
	if input.VerificationPerformed != "" {
		taskRecord.Fields["verification"] = input.VerificationPerformed
	}

	if _, err := s.entitySvc.Store().Write(taskRecord); err != nil {
		return CompleteResult{}, fmt.Errorf("write completion fields: %w", err)
	}

	// Process knowledge entries (best-effort).
	var kResult KnowledgeContributionResult
	for _, entry := range input.KnowledgeEntries {
		kResult.TotalAttempted++

		tier := entry.Tier
		if tier != 2 && tier != 3 {
			tier = 3
		}

		rec, _, err := s.knowledgeSvc.Contribute(ContributeInput{
			Topic:       entry.Topic,
			Content:     entry.Content,
			Scope:       entry.Scope,
			Tier:        tier,
			LearnedFrom: task.ID,
			Tags:        entry.Tags,
		})

		if err != nil {
			kResult.Rejected = append(kResult.Rejected, struct {
				Topic  string
				Reason string
			}{Topic: entry.Topic, Reason: err.Error()})
		} else {
			kResult.Accepted = append(kResult.Accepted, struct {
				EntryID string
				Topic   string
			}{EntryID: rec.ID, Topic: entry.Topic})
			kResult.TotalAccepted++
		}
	}

	// Reload final task state.
	finalTask, err := s.entitySvc.Get("task", task.ID, "")
	if err != nil {
		return CompleteResult{}, fmt.Errorf("reload task: %w", err)
	}

	return CompleteResult{
		Task:                   finalTask.State,
		KnowledgeContributions: kResult,
	}, nil
}
