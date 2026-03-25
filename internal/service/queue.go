package service

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"kanbanzai/internal/model"
	"kanbanzai/internal/validate"
)

// WorkQueueInput holds parameters for the work_queue operation.
type WorkQueueInput struct {
	Role          string // optional: filter by role profile
	ConflictCheck bool   // when true, annotate each ready task with conflict risk
}

// WorkQueueItem represents a task in the ready queue.
type WorkQueueItem struct {
	TaskID        string
	Slug          string
	Summary       string
	ParentFeature string
	FeatureSlug   string
	Estimate      *float64
	AgeDays       int
	Status        string
	ConflictRisk  string   // only set when ConflictCheck=true
	ConflictWith  []string // only set when ConflictCheck=true
}

// WorkQueueResult is the result of the work_queue operation.
type WorkQueueResult struct {
	Queue         []WorkQueueItem
	PromotedCount int
	TotalQueued   int
}

// WorkQueue promotes eligible queued tasks to ready and returns the ready queue.
// This is a write-through query: it modifies task state as a side effect.
func (s *EntityService) WorkQueue(input WorkQueueInput) (WorkQueueResult, error) {
	var result WorkQueueResult

	// Load all tasks
	allTasks, err := s.List(string(model.EntityKindTask))
	if err != nil {
		return result, fmt.Errorf("list tasks: %w", err)
	}

	// Build a map of task IDs to statuses for dependency checking
	taskStatuses := make(map[string]string, len(allTasks))
	for _, t := range allTasks {
		taskStatuses[t.ID] = stringFromState(t.State, "status")
	}

	// Count and attempt to promote queued tasks
	var stillQueued int
	for _, t := range allTasks {
		status := stringFromState(t.State, "status")
		if status != string(model.TaskStatusQueued) {
			continue
		}

		// Check dependencies
		dependsOn := stringSliceFromState(t.State, "depends_on")
		depStatuses := make(map[string]string, len(dependsOn))
		for _, depID := range dependsOn {
			if s, ok := taskStatuses[depID]; ok {
				depStatuses[depID] = s
			} else {
				depStatuses[depID] = "" // unknown
			}
		}

		if err := validate.ValidateTaskQueuedToReady(dependsOn, depStatuses); err != nil {
			stillQueued++
			continue
		}

		// Eligible for promotion — attempt transition
		_, transErr := s.UpdateStatus(UpdateStatusInput{
			Type:   "task",
			ID:     t.ID,
			Slug:   t.Slug,
			Status: string(model.TaskStatusReady),
		})
		if transErr != nil {
			stillQueued++
			continue
		}

		result.PromotedCount++
		// Update our local status map so later dependency checks reflect the promotion
		taskStatuses[t.ID] = string(model.TaskStatusReady)
	}
	result.TotalQueued = stillQueued

	// Reload tasks to get updated statuses
	allTasks, err = s.List(string(model.EntityKindTask))
	if err != nil {
		return result, fmt.Errorf("reload tasks: %w", err)
	}

	// Build feature slug map
	featureSlugs := make(map[string]string)
	allFeatures, _ := s.List(string(model.EntityKindFeature))
	for _, f := range allFeatures {
		featureSlugs[f.ID] = f.Slug
	}

	// Collect ready tasks
	now := time.Now()
	for _, t := range allTasks {
		status := stringFromState(t.State, "status")
		if status != string(model.TaskStatusReady) {
			continue
		}

		parentFeature := stringFromState(t.State, "parent_feature")

		// Role filter: if role specified, check parent feature's profile.
		// For Phase 4a, features don't have explicit role fields, so skip filter
		// (role filter is optional per §10.2 of implementation plan).
		_ = input.Role

		item := WorkQueueItem{
			TaskID:        t.ID,
			Slug:          t.Slug,
			Summary:       stringFromState(t.State, "summary"),
			ParentFeature: parentFeature,
			FeatureSlug:   featureSlugs[parentFeature],
			Status:        status,
		}

		// Estimate
		if est := GetEstimateFromFields(t.State); est != nil {
			item.Estimate = est
		}

		// Age in days (from created or started)
		createdStr := stringFromState(t.State, "created")
		if createdStr == "" {
			// Try started
			createdStr = stringFromState(t.State, "started")
		}
		if createdStr != "" {
			if created, err := time.Parse(time.RFC3339, createdStr); err == nil {
				item.AgeDays = int(now.Sub(created).Hours() / 24)
			}
		}

		result.Queue = append(result.Queue, item)
	}

	// Sort: estimate ASC (nil last), age DESC, task ID lexicographic
	sort.Slice(result.Queue, func(i, j int) bool {
		a, b := result.Queue[i], result.Queue[j]

		if a.Estimate == nil && b.Estimate != nil {
			return false // nil estimate sorts last
		}
		if a.Estimate != nil && b.Estimate == nil {
			return true // non-nil estimate sorts first
		}
		if a.Estimate != nil && b.Estimate != nil {
			if *a.Estimate != *b.Estimate {
				return *a.Estimate < *b.Estimate
			}
		}

		// Same estimate band: age DESC
		if a.AgeDays != b.AgeDays {
			return a.AgeDays > b.AgeDays
		}

		// Tie-break: task ID lexicographic
		return a.TaskID < b.TaskID
	})

	// Conflict check: annotate each ready task with conflict risk against active tasks
	if input.ConflictCheck && len(result.Queue) > 0 {
		// Collect active task IDs
		var activeTaskIDs []string
		for _, t := range allTasks {
			status := stringFromState(t.State, "status")
			if status == string(model.TaskStatusActive) {
				activeTaskIDs = append(activeTaskIDs, t.ID)
			}
		}

		if len(activeTaskIDs) > 0 {
			// We need a ConflictService — create one inline (no branch lookup for queue mode)
			conflictSvc := NewConflictService(s, nil, "")

			for i := range result.Queue {
				item := &result.Queue[i]
				checkIDs := append([]string{item.TaskID}, activeTaskIDs...)
				checkResult, err := conflictSvc.Check(ConflictCheckInput{TaskIDs: checkIDs})
				if err != nil {
					continue // best-effort
				}

				// Find max risk for this task against active tasks
				itemRisk := "none"
				var conflictWith []string
				for _, pair := range checkResult.Pairs {
					// Only look at pairs involving this task
					var otherID string
					if pair.TaskA == item.TaskID {
						otherID = pair.TaskB
					} else if pair.TaskB == item.TaskID {
						otherID = pair.TaskA
					} else {
						continue
					}
					if riskLevel(pair.Risk) > riskLevel(itemRisk) {
						itemRisk = pair.Risk
					}
					if pair.Risk != "none" {
						conflictWith = append(conflictWith, otherID)
					}
				}

				item.ConflictRisk = itemRisk
				item.ConflictWith = conflictWith
			}
		}
	}

	return result, nil
}

// DependencyStatusInput holds parameters for the dependency_status operation.
type DependencyStatusInput struct {
	TaskID string
}

// DependencyEntry represents a single dependency with its current status.
type DependencyEntry struct {
	TaskID        string
	Slug          string
	Status        string
	Blocking      bool
	TerminalState *string
}

// DependencyStatusResult is the result of dependency_status.
type DependencyStatusResult struct {
	TaskID         string
	Slug           string
	Status         string
	DependsOnCount int
	BlockingCount  int
	Dependencies   []DependencyEntry
}

// GetDependencyStatus returns the dependency picture for a task.
func (s *EntityService) GetDependencyStatus(taskID string) (DependencyStatusResult, error) {
	taskID = strings.TrimSpace(taskID)

	// Resolve and load the task
	result, err := s.Get("task", taskID, "")
	if err != nil {
		return DependencyStatusResult{}, fmt.Errorf("task not found: %w", err)
	}

	status := stringFromState(result.State, "status")
	dependsOn := stringSliceFromState(result.State, "depends_on")

	out := DependencyStatusResult{
		TaskID:         result.ID,
		Slug:           result.Slug,
		Status:         status,
		DependsOnCount: len(dependsOn),
	}

	for _, depID := range dependsOn {
		depResult, err := s.Get("task", depID, "")

		var depStatus string
		var depSlug string
		if err != nil {
			depStatus = "unknown"
			depSlug = depID
		} else {
			depStatus = stringFromState(depResult.State, "status")
			depSlug = depResult.Slug
		}

		blocking := !validate.IsTaskDependencySatisfied(depStatus)
		if blocking {
			out.BlockingCount++
		}

		entry := DependencyEntry{
			TaskID:   depID,
			Slug:     depSlug,
			Status:   depStatus,
			Blocking: blocking,
		}
		if !blocking {
			entry.TerminalState = &depStatus
		}

		out.Dependencies = append(out.Dependencies, entry)
	}

	return out, nil
}

// stringSliceFromState reads a []string from a state map.
func stringSliceFromState(state map[string]any, key string) []string {
	v, ok := state[key]
	if !ok || v == nil {
		return nil
	}
	switch typed := v.(type) {
	case []string:
		return typed
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}
