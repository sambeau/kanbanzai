package service

import (
	"fmt"
	"log"
	"strings"
	"time"

	"kanbanzai/internal/config"
	"kanbanzai/internal/worktree"
)

// UnblockedTask records a task that was automatically promoted to ready
// because all of its dependencies reached a terminal state.
type UnblockedTask struct {
	TaskID string
	Slug   string
	Status string // always "ready"
}

// WorktreeResult holds the outcome of an automatic worktree creation attempt
// triggered by a status transition. It is informational — worktree creation
// failures never block the transition itself.
type WorktreeResult struct {
	// Created is true when a new worktree was successfully created.
	Created bool
	// WorktreeID is the ID of the worktree (set only when Created is true).
	WorktreeID string
	// EntityID is the entity the worktree is associated with (feature or bug).
	EntityID string
	// Branch is the git branch name (set only when Created is true).
	Branch string
	// Path is the worktree filesystem path (set only when Created is true).
	Path string
	// Warning is a human-readable message when creation was skipped or failed.
	Warning string
	// AlreadyExists is true when a worktree already existed for the entity.
	AlreadyExists bool
	// UnblockedTasks lists tasks that were promoted to ready because the
	// completed task was their last unsatisfied dependency.
	UnblockedTasks []UnblockedTask
}

// StatusTransitionHook is called by EntityService after a successful status
// transition. Implementations may trigger side effects (e.g. automatic
// worktree creation) but MUST NOT return errors that block the transition —
// the transition has already been persisted when the hook fires.
type StatusTransitionHook interface {
	// OnStatusTransition is called after a status transition has been
	// persisted. It receives the entity type, resolved ID, slug, the
	// previous status, the new status, and the full entity state.
	// The returned WorktreeResult is informational and may be nil.
	OnStatusTransition(entityType, entityID, slug, fromStatus, toStatus string, state map[string]any) *WorktreeResult
}

// CompositeTransitionHook chains multiple StatusTransitionHook implementations,
// merging their results into a single WorktreeResult.
type CompositeTransitionHook struct {
	hooks []StatusTransitionHook
}

// NewCompositeTransitionHook creates a hook that delegates to all provided
// hooks in order and merges their results.
func NewCompositeTransitionHook(hooks ...StatusTransitionHook) *CompositeTransitionHook {
	return &CompositeTransitionHook{hooks: hooks}
}

// OnStatusTransition delegates to each hook in order and merges results.
func (c *CompositeTransitionHook) OnStatusTransition(entityType, entityID, slug, fromStatus, toStatus string, state map[string]any) *WorktreeResult {
	var result *WorktreeResult
	for _, h := range c.hooks {
		r := h.OnStatusTransition(entityType, entityID, slug, fromStatus, toStatus, state)
		if r == nil {
			continue
		}
		if result == nil {
			result = r
			continue
		}
		// Merge: worktree fields from the first hook that sets them
		if r.Created && !result.Created {
			result.Created = true
			result.WorktreeID = r.WorktreeID
			result.EntityID = r.EntityID
			result.Branch = r.Branch
			result.Path = r.Path
		}
		if r.AlreadyExists && !result.AlreadyExists {
			result.AlreadyExists = true
			if result.WorktreeID == "" {
				result.WorktreeID = r.WorktreeID
			}
		}
		if r.Warning != "" && result.Warning == "" {
			result.Warning = r.Warning
		}
		result.UnblockedTasks = append(result.UnblockedTasks, r.UnblockedTasks...)
	}
	return result
}

// WorktreeTransitionHook implements StatusTransitionHook to automatically
// create a worktree when:
//   - A task transitions to "active" (worktree is created for its parent feature)
//   - A bug transitions to "in-progress" (worktree is created for the bug itself)
//
// Per spec §6.5: worktree creation failure does not block the task transition.
type WorktreeTransitionHook struct {
	store     *worktree.Store
	gitOps    *worktree.Git
	entitySvc *EntityService
}

// NewWorktreeTransitionHook creates a hook that bridges status transitions to
// automatic worktree creation.
func NewWorktreeTransitionHook(store *worktree.Store, gitOps *worktree.Git, entitySvc *EntityService) *WorktreeTransitionHook {
	return &WorktreeTransitionHook{
		store:     store,
		gitOps:    gitOps,
		entitySvc: entitySvc,
	}
}

// OnStatusTransition checks if the transition should trigger automatic
// worktree creation and attempts it if so.
func (h *WorktreeTransitionHook) OnStatusTransition(entityType, entityID, slug, fromStatus, toStatus string, state map[string]any) *WorktreeResult {
	switch {
	case strings.EqualFold(entityType, "task") && toStatus == "active":
		return h.handleTaskActivation(entityID, state)
	case strings.EqualFold(entityType, "bug") && toStatus == "in-progress":
		return h.handleBugInProgress(entityID, state)
	default:
		return nil
	}
}

// handleTaskActivation creates a worktree for the task's parent feature
// when the task transitions to "active".
func (h *WorktreeTransitionHook) handleTaskActivation(taskID string, taskState map[string]any) *WorktreeResult {
	// Look up the parent feature
	parentFeature, _ := taskState["parent_feature"].(string)
	if parentFeature == "" {
		return &WorktreeResult{
			Warning: fmt.Sprintf("task %s is not associated with a feature; skipping automatic worktree creation", taskID),
		}
	}

	// Load the parent feature to get its slug for branch naming
	feature, err := h.entitySvc.Get("feature", parentFeature, "")
	if err != nil {
		return &WorktreeResult{
			Warning: fmt.Sprintf("could not load parent feature %s for task %s: %v", parentFeature, taskID, err),
		}
	}

	featureSlug, _ := feature.State["slug"].(string)
	return h.ensureWorktree(parentFeature, featureSlug)
}

// handleBugInProgress creates a worktree for the bug itself when it
// transitions to "in-progress".
func (h *WorktreeTransitionHook) handleBugInProgress(bugID string, bugState map[string]any) *WorktreeResult {
	bugSlug, _ := bugState["slug"].(string)
	return h.ensureWorktree(bugID, bugSlug)
}

// ensureWorktree checks if a worktree already exists for the given entity,
// and creates one if not. This is the shared core logic extracted from the
// worktree_create MCP tool handler.
func (h *WorktreeTransitionHook) ensureWorktree(entityID, slug string) *WorktreeResult {
	// Idempotency: if a worktree already exists, succeed silently
	existing, err := h.store.GetByEntityID(entityID)
	if err == nil && existing.ID != "" {
		log.Printf("status hook: worktree %s already exists for %s; skipping creation", existing.ID, entityID)
		return &WorktreeResult{
			AlreadyExists: true,
			WorktreeID:    existing.ID,
			EntityID:      entityID,
			Branch:        existing.Branch,
			Path:          existing.Path,
		}
	}

	// Resolve identity for created_by (best-effort, same as worktree_create)
	createdBy, err := config.ResolveIdentity("")
	if err != nil {
		return &WorktreeResult{
			EntityID: entityID,
			Warning:  fmt.Sprintf("automatic worktree creation skipped for %s: cannot resolve identity: %v", entityID, err),
		}
	}

	// Generate branch name and worktree path
	branchName := worktree.GenerateBranchName(entityID, slug)
	wtPath := worktree.GenerateWorktreePath(entityID, slug)

	// Create git worktree with a new branch from HEAD
	if err := h.gitOps.CreateWorktreeNewBranch(wtPath, branchName, ""); err != nil {
		log.Printf("status hook: git worktree creation failed for %s: %v", entityID, err)
		return &WorktreeResult{
			EntityID: entityID,
			Warning:  fmt.Sprintf("automatic worktree creation failed for %s: %v", entityID, err),
		}
	}

	// Create the worktree tracking record
	record := worktree.Record{
		EntityID:  entityID,
		Branch:    branchName,
		Path:      wtPath,
		Status:    worktree.StatusActive,
		Created:   time.Now().UTC(),
		CreatedBy: createdBy,
	}

	created, err := h.store.Create(record)
	if err != nil {
		// Compensating cleanup: remove the git worktree if record creation fails
		_ = h.gitOps.RemoveWorktree(wtPath, true)
		log.Printf("status hook: worktree record creation failed for %s: %v", entityID, err)
		return &WorktreeResult{
			EntityID: entityID,
			Warning:  fmt.Sprintf("automatic worktree creation failed for %s: record creation error: %v", entityID, err),
		}
	}

	log.Printf("status hook: automatically created worktree %s (branch %s) for %s", created.ID, branchName, entityID)
	return &WorktreeResult{
		Created:    true,
		WorktreeID: created.ID,
		EntityID:   entityID,
		Branch:     created.Branch,
		Path:       created.Path,
	}
}
