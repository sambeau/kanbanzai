// Package mcp next_tool.go — work queue and claim tool for Kanbanzai 2.0 (Track F).
//
// next() without an id returns the ready queue (same as work_queue in 1.0),
// including write-through promotion of eligible queued → ready tasks.
// Side effects report each task promoted in this call.
//
// next(id) claims the identified task (or the top ready task in a feature/plan)
// by transitioning it ready → active, recording dispatch metadata, and
// returning fully assembled structured context (see assembly.go for the
// shared context assembly pipeline).
//
// Unlike handoff (which returns rendered Markdown), next returns structured
// data so the caller can render or process it however they need.
package mcp

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/sambeau/kanbanzai/internal/config"
	kbzctx "github.com/sambeau/kanbanzai/internal/context"
	idpkg "github.com/sambeau/kanbanzai/internal/id"
	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/service"
	"github.com/sambeau/kanbanzai/internal/stage"
	"github.com/sambeau/kanbanzai/internal/worktree"
)

// NextTools returns the `next` MCP tool registered in the core group.
func NextTools(
	entitySvc *service.EntityService,
	dispatchSvc *service.DispatchService,
	profileStore *kbzctx.ProfileStore,
	knowledgeSvc *service.KnowledgeService,
	intelligenceSvc *service.IntelligenceService,
	docRecordSvc *service.DocumentService,
	mergedToolHints map[string]string,
	roleStore *kbzctx.RoleStore,
	worktreeStore *worktree.Store,
) []server.ServerTool {
	return []server.ServerTool{nextTool(entitySvc, dispatchSvc, profileStore, knowledgeSvc, intelligenceSvc, docRecordSvc, mergedToolHints, roleStore, worktreeStore)}
}

func nextTool(
	entitySvc *service.EntityService,
	dispatchSvc *service.DispatchService,
	profileStore *kbzctx.ProfileStore,
	knowledgeSvc *service.KnowledgeService,
	intelligenceSvc *service.IntelligenceService,
	docRecordSvc *service.DocumentService,
	mergedToolHints map[string]string,
	roleStore *kbzctx.RoleStore,
	worktreeStore *worktree.Store,
) server.ServerTool {
	tool := mcp.NewTool("next",
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithTitleAnnotation("Work Queue & Dispatch"),
		mcp.WithDescription(
			"Start here when beginning work — the primary way to find and claim tasks. "+
				"Call without id to inspect the work queue (all ready tasks sorted by priority). "+
				"Call with a task, feature, or plan ID to claim the next ready task and receive "+
				"assembled context (spec sections, knowledge entries, file paths, role conventions). "+
				"Use INSTEAD OF manually querying entities with entity(action: \"list\") and assembling "+
				"context yourself. Call BEFORE handoff when delegating to sub-agents, or before starting "+
				"work directly. For dashboard views and progress metrics, use status instead. "+
				"When id is provided, the task transitions ready → active (claim). "+
				"When id is omitted, no state changes occur (read-only queue inspection).",
		),
		mcp.WithString("id", mcp.Description(
			"Task ID (TASK-... or T-...), Feature ID (FEAT-...), or Plan ID to claim. "+
				"Omit to inspect the ready queue.",
		)),
		mcp.WithString("role", mcp.Description(
			"Role profile ID for context assembly (e.g. backend, frontend). "+
				"In queue mode, filters results to tasks whose parent feature matches the role.",
		)),
		mcp.WithBoolean("conflict_check", mcp.Description(
			"When true in queue mode, annotate each ready task with conflict risk "+
				"against currently active tasks. Matches Phase 4b work_queue --conflict-check behaviour.",
		)),
	)

	handler := WithSideEffects(func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		args, _ := req.Params.Arguments.(map[string]any)
		rawID, _ := args["id"].(string)
		rawRole, _ := args["role"].(string)
		conflictCheck, _ := args["conflict_check"].(bool)
		id := strings.TrimSpace(rawID)
		role := strings.TrimSpace(rawRole)

		if id == "" {
			return nextQueueMode(ctx, role, conflictCheck, entitySvc)
		}
		return nextClaimMode(ctx, id, role, entitySvc, dispatchSvc, profileStore, knowledgeSvc, intelligenceSvc, docRecordSvc, mergedToolHints, roleStore, worktreeStore)
	})

	return server.ServerTool{Tool: tool, Handler: handler}
}

// ─── Queue inspection mode ────────────────────────────────────────────────────

// nextQueueMode returns the ready queue with write-through promotion.
func nextQueueMode(ctx context.Context, role string, conflictCheck bool, entitySvc *service.EntityService) (any, error) {
	result, err := entitySvc.WorkQueue(service.WorkQueueInput{
		Role:          role,
		ConflictCheck: conflictCheck,
	})
	if err != nil {
		return nil, fmt.Errorf("Cannot load work queue: %w.\n\nTo resolve:\n  Check project health with status() and verify .kbz/state/ is intact", err)
	}

	// Report each task promoted in this call as a side effect.
	for _, taskID := range result.PromotedTaskIDs {
		PushSideEffect(ctx, SideEffect{
			Type:       SideEffectTaskUnblocked,
			EntityID:   taskID,
			EntityType: "task",
			FromStatus: "queued",
			ToStatus:   "ready",
			Trigger:    "All dependencies resolved",
		})
	}

	// Build queue item list.
	queueItems := make([]map[string]any, 0, len(result.Queue))
	for _, item := range result.Queue {
		qitem := map[string]any{
			"task_id":        item.TaskID,
			"display_id":     idpkg.FormatFullDisplay(item.TaskID),
			"slug":           item.Slug,
			"summary":        item.Summary,
			"parent_feature": item.ParentFeature,
			"feature_slug":   item.FeatureSlug,
			"age_days":       item.AgeDays,
			"estimate":       nil,
		}
		if item.Estimate != nil {
			qitem["estimate"] = *item.Estimate
		}
		// Conflict annotations (only present when conflict_check=true).
		if item.ConflictRisk != "" {
			qitem["conflict_risk"] = item.ConflictRisk
		}
		if len(item.ConflictWith) > 0 {
			qitem["conflict_with"] = item.ConflictWith
		}
		queueItems = append(queueItems, qitem)
	}

	response := map[string]any{
		"queue":          queueItems,
		"promoted_count": result.PromotedCount,
		"total_queued":   result.TotalQueued,
	}

	// Include orientation breadcrumb when the queue is empty — the agent is
	// likely orienting rather than mid-flow. Omit when work is available.
	if len(queueItems) == 0 {
		response["orientation"] = map[string]any{
			"message":     "This is a kanbanzai-managed project. For workflow guidance, read .agents/skills/kanbanzai-getting-started/SKILL.md",
			"skills_path": ".agents/skills/",
		}
	}

	return response, nil
}

// ─── Claim mode ───────────────────────────────────────────────────────────────

// nextClaimMode claims a task and returns full structured context.
func nextClaimMode(
	ctx context.Context,
	id, role string,
	entitySvc *service.EntityService,
	dispatchSvc *service.DispatchService,
	profileStore *kbzctx.ProfileStore,
	knowledgeSvc *service.KnowledgeService,
	intelligenceSvc *service.IntelligenceService,
	docRecordSvc *service.DocumentService,
	mergedToolHints map[string]string,
	roleStore *kbzctx.RoleStore,
	worktreeStore *worktree.Store,
) (any, error) {
	// Resolve the input ID to a specific task ID.
	taskID, err := nextResolveTaskID(id, entitySvc)
	if err != nil {
		return nil, err
	}

	// Load the task to check its current status.
	task, err := entitySvc.Get("task", taskID, "")
	if err != nil {
		return nil, fmt.Errorf("Cannot claim task %s: task not found.\n\nTo resolve:\n  Verify the task ID with entity(action: \"list\", type: \"task\") or inspect the queue with next()", taskID)
	}

	status, _ := task.State["status"].(string)
	switch status {
	case "ready":
		// Proceed to claim.
	case "active":
		// Already claimed — return error with existing dispatch metadata.
		dispTo, _ := task.State["dispatched_to"].(string)
		claimedAt, _ := task.State["claimed_at"].(string)
		dispBy, _ := task.State["dispatched_by"].(string)
		return nil, fmt.Errorf(
			"Cannot claim task %s: already dispatched to %q at %s by %s.\n\nTo resolve:\n  Use handoff(task_id: %q) to generate a prompt for this active task, or pick another task from next()",
			taskID, dispTo, claimedAt, dispBy, taskID,
		)
	default:
		return nil, fmt.Errorf(
			"Cannot claim task %s: status is %q, but only \"ready\" tasks can be claimed.\n\nTo resolve:\n  Check task details with status(id: %q) and ensure prerequisites are met",
			taskID, status, taskID,
		)
	}

	// Stage-aware lifecycle validation (FR-002).
	parentFeatureForValidation, _ := task.State["parent_feature"].(string)
	featureStage, valErr := ValidateFeatureStage(parentFeatureForValidation, entitySvc)
	if valErr != nil {
		return nil, fmt.Errorf("Cannot claim task %s: %v", taskID, valErr)
	}

	// Determine dispatched_to and dispatched_by.
	dispatchedTo := role
	if dispatchedTo == "" {
		dispatchedTo = "unspecified"
	}
	callerIdentity, _ := config.ResolveIdentity("")
	if callerIdentity == "" {
		callerIdentity = "mcp-session"
	}

	// Claim the task: ready → active, set dispatch metadata.
	_, err = dispatchSvc.DispatchTask(service.DispatchInput{
		TaskID:       taskID,
		Role:         dispatchedTo,
		DispatchedBy: callerIdentity,
	})
	if err != nil {
		return nil, fmt.Errorf("Cannot claim task %s: dispatch failed: %w.\n\nTo resolve:\n  Check task status with status(id: %q) and retry", taskID, err, taskID)
	}

	// Report status transition as a side effect.
	PushSideEffect(ctx, SideEffect{
		Type:       SideEffectStatusTransition,
		EntityID:   taskID,
		EntityType: "task",
		FromStatus: "ready",
		ToStatus:   "active",
		Trigger:    "Claimed via next",
	})

	// Reload the task to get updated dispatch fields.
	task, err = entitySvc.Get("task", taskID, "")
	if err != nil {
		return nil, fmt.Errorf("Cannot reload task %s after claim: %w.\n\nTo resolve:\n  The task was claimed successfully — use entity(action: \"get\", id: %q) to retrieve it", taskID, err, taskID)
	}

	// Build parent feature info.
	parentFeature, _ := task.State["parent_feature"].(string)
	var parentFeatureInfo map[string]any
	if parentFeature != "" {
		if feat, ferr := entitySvc.Get("feature", parentFeature, ""); ferr == nil {
			parentFeatureInfo = map[string]any{
				"id":         feat.ID,
				"display_id": idpkg.FormatFullDisplay(feat.ID),
				"slug":       feat.Slug,
				"plan_id":    nextStateStr(feat.State, "parent"),
			}
		}
	}

	// Build task summary.
	taskOut := map[string]any{
		"id":             task.ID,
		"display_id":     idpkg.FormatFullDisplay(task.ID),
		"slug":           task.Slug,
		"summary":        nextStateStr(task.State, "summary"),
		"status":         nextStateStr(task.State, "status"),
		"parent_feature": parentFeatureInfo,
	}
	if est := service.GetEstimateFromFields(task.State); est != nil {
		taskOut["estimate"] = *est
	}

	// Assemble structured context using the shared pipeline (assembly.go).
	actx := assembleContext(asmInput{
		taskState:       task.State,
		parentFeature:   parentFeature,
		role:            role,
		featureStage:    featureStage,
		profileStore:    profileStore,
		knowledgeSvc:    knowledgeSvc,
		intelligenceSvc: intelligenceSvc,
		docRecordSvc:    docRecordSvc,
		entitySvc:       entitySvc,
		mergedToolHints: mergedToolHints,
		roleStore:       roleStore,
		worktreeStore:   worktreeStore,
	})

	return map[string]any{
		"task":    taskOut,
		"context": nextContextToMap(actx),
	}, nil
}

// nextContextToMap converts an assembledContext to the structured map returned
// by the next tool's claim mode response.
func nextContextToMap(actx assembledContext) map[string]any {
	specSections := make([]map[string]any, len(actx.specSections))
	for i, s := range actx.specSections {
		specSections[i] = map[string]any{
			"document": s.document,
			"section":  s.section,
			"content":  s.content,
		}
	}

	knowledgeOut := make([]map[string]any, len(actx.knowledge))
	for i, ke := range actx.knowledge {
		knowledgeOut[i] = map[string]any{
			"topic":      ke.topic,
			"content":    ke.content,
			"scope":      ke.scope,
			"confidence": ke.confidence,
		}
	}

	filesContextOut := make([]map[string]any, len(actx.filesContext))
	for i, f := range actx.filesContext {
		fe := map[string]any{"path": f.path}
		if f.note != "" {
			fe["note"] = f.note
		}
		filesContextOut[i] = fe
	}

	trimmedOut := make([]map[string]any, len(actx.trimmed))
	for i, te := range actx.trimmed {
		trimmedOut[i] = map[string]any{
			"type":       te.entryType,
			"topic":      te.topic,
			"size_bytes": te.sizeBytes,
		}
	}

	out := map[string]any{
		"spec_sections":       specSections,
		"acceptance_criteria": actx.acceptanceCriteria,
		"knowledge":           knowledgeOut,
		"files_context":       filesContextOut,
		"constraints":         actx.constraints,
		"byte_usage":          actx.byteUsage,
		"byte_budget":         actx.byteBudget,
		"trimmed":             trimmedOut,
	}
	if actx.roleProfile != "" {
		out["role_profile"] = actx.roleProfile
	}
	if actx.specFallbackPath != "" {
		out["spec_fallback_path"] = actx.specFallbackPath
	}
	if len(actx.experimentNudge) > 0 {
		nudges := make([]map[string]any, len(actx.experimentNudge))
		for i, n := range actx.experimentNudge {
			nudges[i] = map[string]any{
				"decision_id": n.decisionID,
				"summary":     n.summary,
			}
		}
		out["active_experiments"] = nudges
	}
	// Stage-aware fields (FR-013).
	if actx.stageAware {
		out["stage_aware"] = true
		out["feature_stage"] = actx.featureStage
		if cfg, ok := stage.ForStage(actx.featureStage); ok {
			out["orchestration_pattern"] = string(cfg.Orchestration)
			out["effort_budget"] = map[string]any{
				"stage":   actx.featureStage,
				"text":    cfg.EffortBudget.Text,
				"warning": cfg.EffortBudget.Warning,
			}
			out["tool_subset"] = map[string]any{
				"primary":  cfg.PrimaryTools,
				"excluded": cfg.ExcludedTools,
			}
			if cfg.OutputConvention {
				out["output_convention"] = "Sub-agents write outputs to documents and task records. Read their status via entity(action: \"get\") and doc(action: \"get\"). Do not retain sub-agent conversation output in your context — use references (document IDs, task IDs, status summaries) instead of contents."
			}
		}
	} else {
		out["stage_aware"] = false
	}
	if actx.reviewRubricText != "" {
		out["review_rubric"] = actx.reviewRubricText
	}
	if actx.testExpectText != "" {
		out["test_expectations"] = actx.testExpectText
	}
	if actx.implGuidanceText != "" {
		out["impl_guidance"] = actx.implGuidanceText
	}
	if actx.planGuidanceText != "" {
		out["plan_guidance"] = actx.planGuidanceText
	}
	// Tool hint — omitted when no hint resolves (FR-015, FR-016).
	if actx.toolHint != "" {
		out["tool_hint"] = actx.toolHint
	}
	// Graph project — always present (empty string when no worktree).
	out["graph_project"] = actx.graphProject
	return out
}

// ─── ID resolution ────────────────────────────────────────────────────────────

// nextResolveTaskID maps a task/feature/plan ID to a specific task ID to claim.
func nextResolveTaskID(id string, entitySvc *service.EntityService) (string, error) {
	switch nextInferEntityType(id) {
	case "task":
		return id, nil

	case "feature":
		task, err := nextFindTopReadyTask(id, entitySvc)
		if err != nil {
			return "", fmt.Errorf("Cannot find ready task in feature %s: %w.\n\nTo resolve:\n  Check feature status with status(id: %q)", id, err, id)
		}
		if task == nil {
			return "", fmt.Errorf("Cannot claim from feature %s: no tasks in ready status.\n\nTo resolve:\n  Check feature progress with status(id: %q) — tasks may be queued, active, or done", id, id)
		}
		return task.ID, nil

	case "plan":
		tasks, err := entitySvc.CrossEntityQuery(id)
		if err != nil {
			return "", fmt.Errorf("Cannot query tasks for plan %s: %w.\n\nTo resolve:\n  Check plan status with status(id: %q)", id, err, id)
		}
		var ready []service.ListResult
		for _, t := range tasks {
			if nextStateStr(t.State, "status") == "ready" {
				ready = append(ready, t)
			}
		}
		if len(ready) == 0 {
			return "", fmt.Errorf("Cannot claim from plan %s: no tasks in ready status.\n\nTo resolve:\n  Check plan progress with status(id: %q) — tasks may be queued, active, or done", id, id)
		}
		nextSortByQueueOrder(ready)
		return ready[0].ID, nil

	default:
		return "", fmt.Errorf("Cannot resolve ID %q: unrecognised ID format.\n\nTo resolve:\n  Use a prefixed ID: TASK-..., FEAT-..., or a plan ID (e.g. P1-slug)", id)
	}
}

// nextFindTopReadyTask returns the highest-priority ready task in a feature.
// Returns nil (with no error) when the feature has no ready tasks.
func nextFindTopReadyTask(featureID string, entitySvc *service.EntityService) (*service.ListResult, error) {
	allTasks, err := entitySvc.List("task")
	if err != nil {
		return nil, err
	}
	var ready []service.ListResult
	for _, t := range allTasks {
		if nextStateStr(t.State, "parent_feature") == featureID &&
			nextStateStr(t.State, "status") == "ready" {
			ready = append(ready, t)
		}
	}
	if len(ready) == 0 {
		return nil, nil
	}
	nextSortByQueueOrder(ready)
	return &ready[0], nil
}

// nextSortByQueueOrder sorts tasks by: estimate ASC (nil last), age DESC, ID ASC.
func nextSortByQueueOrder(tasks []service.ListResult) {
	now := time.Now()
	sort.SliceStable(tasks, func(i, j int) bool {
		ai := service.GetEstimateFromFields(tasks[i].State)
		bj := service.GetEstimateFromFields(tasks[j].State)

		if ai == nil && bj != nil {
			return false // nil estimate sorts last
		}
		if ai != nil && bj == nil {
			return true
		}
		if ai != nil && bj != nil && *ai != *bj {
			return *ai < *bj
		}

		// Age descending: older tasks have higher priority.
		ciStr := nextStateStr(tasks[i].State, "created")
		cjStr := nextStateStr(tasks[j].State, "created")
		ci, _ := time.Parse(time.RFC3339, ciStr)
		cj, _ := time.Parse(time.RFC3339, cjStr)
		ageI := int(now.Sub(ci).Hours() / 24)
		ageJ := int(now.Sub(cj).Hours() / 24)
		if ageI != ageJ {
			return ageI > ageJ
		}

		return tasks[i].ID < tasks[j].ID
	})
}

// nextInferEntityType returns the entity type implied by an ID string.
// Returns "task", "feature", "plan", or "" for unrecognised formats.
func nextInferEntityType(id string) string {
	upper := strings.ToUpper(id)
	switch {
	case strings.HasPrefix(upper, "TASK-"), strings.HasPrefix(upper, "T-"):
		return "task"
	case strings.HasPrefix(upper, "FEAT-"):
		return "feature"
	case model.IsPlanID(id):
		return "plan"
	default:
		return ""
	}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// nextStateStr reads a string value from a state map. Returns "" if absent.
func nextStateStr(state map[string]any, key string) string {
	if state == nil {
		return ""
	}
	s, _ := state[key].(string)
	return s
}
