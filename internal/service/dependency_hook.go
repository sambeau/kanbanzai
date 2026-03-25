package service

import (
	"log"
	"strings"

	"kanbanzai/internal/validate"
)

// DependencyUnblockingHook implements StatusTransitionHook to automatically
// promote tasks from queued/blocked to ready when all of their dependencies
// reach a terminal state (done, not-planned, or duplicate).
//
// Hook failures never propagate — any error is logged as a warning and the
// hook returns nil so the original transition is unaffected.
type DependencyUnblockingHook struct {
	entitySvc *EntityService
}

// NewDependencyUnblockingHook creates a hook that evaluates dependent tasks
// for automatic unblocking after a task reaches a terminal state.
func NewDependencyUnblockingHook(entitySvc *EntityService) *DependencyUnblockingHook {
	return &DependencyUnblockingHook{entitySvc: entitySvc}
}

// OnStatusTransition checks whether the completed task unblocks any
// dependent tasks. It only fires for task entities transitioning to a
// terminal state (done, not-planned, duplicate).
func (h *DependencyUnblockingHook) OnStatusTransition(entityType, entityID, slug, fromStatus, toStatus string, state map[string]any) (result *WorktreeResult) {
	// Recover from any panic so the original transition is never affected.
	defer func() {
		if r := recover(); r != nil {
			log.Printf("dependency hook: recovered from panic for %s %s: %v", entityType, entityID, r)
			result = nil
		}
	}()

	if !strings.EqualFold(entityType, "task") {
		return nil
	}
	if !isTerminalTaskStatus(toStatus) {
		return nil
	}

	unblocked, err := h.evaluateDependents(entityID)
	if err != nil {
		log.Printf("dependency hook: error evaluating dependents of %s: %v", entityID, err)
		return nil
	}
	if len(unblocked) == 0 {
		return nil
	}

	return &WorktreeResult{
		UnblockedTasks: unblocked,
	}
}

// evaluateDependents finds all tasks that depend on completedTaskID,
// checks whether all of their dependencies are now terminal, and promotes
// eligible tasks from queued/blocked to ready via a direct store write
// (bypassing UpdateStatus to avoid the dependency gate re-check and
// recursive hook invocations).
func (h *DependencyUnblockingHook) evaluateDependents(completedTaskID string) ([]UnblockedTask, error) {
	allTasks, err := h.entitySvc.List("task")
	if err != nil {
		return nil, err
	}

	// Index all task statuses for fast lookup during dependency checks.
	taskStatuses := make(map[string]string, len(allTasks))
	for _, t := range allTasks {
		taskStatuses[t.ID] = stringFromState(t.State, "status")
	}

	var unblocked []UnblockedTask

	for _, task := range allTasks {
		deps := stringSliceFromState(task.State, "depends_on")
		if len(deps) == 0 {
			continue
		}

		// Only consider tasks that list the just-completed task as a dependency.
		if !containsString(deps, completedTaskID) {
			continue
		}

		status := stringFromState(task.State, "status")
		if status != "queued" && status != "blocked" {
			continue
		}

		// Check whether ALL dependencies are now in a terminal state.
		if !allDepsTerminal(deps, taskStatuses) {
			continue
		}

		// Promote to ready via direct store write (system-initiated transition).
		promoted, err := h.promoteToReady(task.ID, task.Slug)
		if err != nil {
			log.Printf("dependency hook: failed to promote task %s to ready: %v", task.ID, err)
			continue
		}
		unblocked = append(unblocked, promoted)
	}

	return unblocked, nil
}

// promoteToReady loads a task record from the store, sets its status to
// "ready", and writes it back. This bypasses UpdateStatus intentionally —
// it is a system-initiated transition that should not re-trigger hooks or
// re-validate the dependency gate (we already confirmed deps are terminal).
func (h *DependencyUnblockingHook) promoteToReady(taskID, taskSlug string) (UnblockedTask, error) {
	store := h.entitySvc.Store()
	rec, err := store.Load("task", taskID, taskSlug)
	if err != nil {
		return UnblockedTask{}, err
	}

	rec.Fields["status"] = "ready"
	if _, err := store.Write(rec); err != nil {
		return UnblockedTask{}, err
	}

	slug, _ := rec.Fields["slug"].(string)
	log.Printf("dependency hook: promoted task %s (%s) to ready — all dependencies terminal", taskID, slug)

	return UnblockedTask{
		TaskID: taskID,
		Slug:   slug,
		Status: "ready",
	}, nil
}

// isTerminalTaskStatus returns true if the status is a terminal task state
// that satisfies dependency checks.
func isTerminalTaskStatus(status string) bool {
	return validate.IsTaskDependencySatisfied(status)
}

// allDepsTerminal returns true if every dependency ID maps to a terminal
// task status in the provided status index.
func allDepsTerminal(deps []string, taskStatuses map[string]string) bool {
	for _, depID := range deps {
		status, ok := taskStatuses[depID]
		if !ok {
			return false
		}
		if !isTerminalTaskStatus(status) {
			return false
		}
	}
	return true
}

// containsString checks whether slice contains s.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
