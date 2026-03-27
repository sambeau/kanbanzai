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

	"kanbanzai/internal/config"
	kbzctx "kanbanzai/internal/context"
	"kanbanzai/internal/model"
	"kanbanzai/internal/service"
)

// NextTools returns the `next` MCP tool registered in the core group.
func NextTools(
	entitySvc *service.EntityService,
	dispatchSvc *service.DispatchService,
	profileStore *kbzctx.ProfileStore,
	knowledgeSvc *service.KnowledgeService,
	intelligenceSvc *service.IntelligenceService,
	docRecordSvc *service.DocumentService,
) []server.ServerTool {
	return []server.ServerTool{nextTool(entitySvc, dispatchSvc, profileStore, knowledgeSvc, intelligenceSvc, docRecordSvc)}
}

func nextTool(
	entitySvc *service.EntityService,
	dispatchSvc *service.DispatchService,
	profileStore *kbzctx.ProfileStore,
	knowledgeSvc *service.KnowledgeService,
	intelligenceSvc *service.IntelligenceService,
	docRecordSvc *service.DocumentService,
) server.ServerTool {
	tool := mcp.NewTool("next",
		mcp.WithDescription(
			"Claim work and get full context. "+
				"Call without id to inspect the ready queue (promotes eligible queued tasks). "+
				"Call with a task ID (TASK-...) to claim that specific task. "+
				"Call with a feature ID (FEAT-...) to claim the top ready task in the feature. "+
				"Call with a plan ID (e.g. P1-...) to claim the top ready task across the plan. "+
				"Claiming transitions the task from ready to active and returns structured context: "+
				"spec sections, acceptance criteria, knowledge entries, file context, and role conventions. "+
				"Returns an error if the task is not in ready status (already active returns the existing dispatch metadata). "+
				"Replaces the 1.0 pattern of work_queue → dispatch_task → context_assemble.",
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
		return nextClaimMode(ctx, id, role, entitySvc, dispatchSvc, profileStore, knowledgeSvc, intelligenceSvc, docRecordSvc)
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
		return nil, fmt.Errorf("work queue: %w", err)
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

	return map[string]any{
		"queue":          queueItems,
		"promoted_count": result.PromotedCount,
		"total_queued":   result.TotalQueued,
	}, nil
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
) (any, error) {
	// Resolve the input ID to a specific task ID.
	taskID, err := nextResolveTaskID(id, entitySvc)
	if err != nil {
		return nil, err
	}

	// Load the task to check its current status.
	task, err := entitySvc.Get("task", taskID, "")
	if err != nil {
		return nil, fmt.Errorf("task %s not found", taskID)
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
			"task %s is already claimed (dispatched to %q at %s by %s)",
			taskID, dispTo, claimedAt, dispBy,
		)
	default:
		return nil, fmt.Errorf(
			"task %s is in status %q, expected \"ready\"",
			taskID, status,
		)
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
		return nil, fmt.Errorf("claim task: %w", err)
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
		return nil, fmt.Errorf("reload task after claim: %w", err)
	}

	// Build parent feature info.
	parentFeature, _ := task.State["parent_feature"].(string)
	var parentFeatureInfo map[string]any
	if parentFeature != "" {
		if feat, ferr := entitySvc.Get("feature", parentFeature, ""); ferr == nil {
			parentFeatureInfo = map[string]any{
				"id":      feat.ID,
				"slug":    feat.Slug,
				"plan_id": nextStateStr(feat.State, "parent"),
			}
		}
	}

	// Build task summary.
	taskOut := map[string]any{
		"id":             task.ID,
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
		profileStore:    profileStore,
		knowledgeSvc:    knowledgeSvc,
		intelligenceSvc: intelligenceSvc,
		docRecordSvc:    docRecordSvc,
		entitySvc:       entitySvc,
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
			return "", fmt.Errorf("find ready task in feature %s: %w", id, err)
		}
		if task == nil {
			return "", fmt.Errorf("no ready tasks in feature %s", id)
		}
		return task.ID, nil

	case "plan":
		tasks, err := entitySvc.CrossEntityQuery(id)
		if err != nil {
			return "", fmt.Errorf("query tasks for plan %s: %w", id, err)
		}
		var ready []service.ListResult
		for _, t := range tasks {
			if nextStateStr(t.State, "status") == "ready" {
				ready = append(ready, t)
			}
		}
		if len(ready) == 0 {
			return "", fmt.Errorf("no ready tasks in plan %s", id)
		}
		nextSortByQueueOrder(ready)
		return ready[0].ID, nil

	default:
		return "", fmt.Errorf("entity %s not found or has unrecognised ID format", id)
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
