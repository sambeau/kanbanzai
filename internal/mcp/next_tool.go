// Package mcp next_tool.go — work queue and claim tool for Kanbanzai 2.0 (Track F).
//
// next() without an id returns the ready queue (same as work_queue in 1.0),
// including write-through promotion of eligible queued → ready tasks.
// Side effects report each task promoted in this call.
//
// next(id) claims the identified task (or the top ready task in a feature/plan)
// by transitioning it ready → active, recording dispatch metadata, and
// returning fully assembled structured context:
//
//   - spec_sections    — document intelligence sections referencing the parent feature
//   - acceptance_criteria — testable criteria extracted from spec sections
//   - knowledge        — knowledge entries scoped to the role or project
//   - files_context    — files from task's files_planned
//   - constraints      — role profile conventions
//   - byte_usage / byte_budget / trimmed
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

const nextDefaultBudget = 30720

// NextTools returns the `next` MCP tool registered in the core group.
func NextTools(
	entitySvc *service.EntityService,
	dispatchSvc *service.DispatchService,
	profileStore *kbzctx.ProfileStore,
	knowledgeSvc *service.KnowledgeService,
	intelligenceSvc *service.IntelligenceService,
) []server.ServerTool {
	return []server.ServerTool{nextTool(entitySvc, dispatchSvc, profileStore, knowledgeSvc, intelligenceSvc)}
}

func nextTool(
	entitySvc *service.EntityService,
	dispatchSvc *service.DispatchService,
	profileStore *kbzctx.ProfileStore,
	knowledgeSvc *service.KnowledgeService,
	intelligenceSvc *service.IntelligenceService,
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
	)

	handler := WithSideEffects(func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		args, _ := req.Params.Arguments.(map[string]any)
		rawID, _ := args["id"].(string)
		rawRole, _ := args["role"].(string)
		id := strings.TrimSpace(rawID)
		role := strings.TrimSpace(rawRole)

		if id == "" {
			return nextQueueMode(ctx, role, entitySvc)
		}
		return nextClaimMode(ctx, id, role, entitySvc, dispatchSvc, profileStore, knowledgeSvc, intelligenceSvc)
	})

	return server.ServerTool{Tool: tool, Handler: handler}
}

// ─── Queue inspection mode ────────────────────────────────────────────────────

// nextQueueMode returns the ready queue with write-through promotion.
func nextQueueMode(ctx context.Context, role string, entitySvc *service.EntityService) (any, error) {
	result, err := entitySvc.WorkQueue(service.WorkQueueInput{Role: role})
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

	// Assemble structured context.
	nctx := nextAssembleContext(task.State, parentFeature, role, profileStore, knowledgeSvc, intelligenceSvc)

	return map[string]any{
		"task":    taskOut,
		"context": nctx,
	}, nil
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

// ─── Context assembly ─────────────────────────────────────────────────────────

// nextCtxData is the assembled context for the claim response.
type nextCtxData struct {
	specSections       []nextSpecSection
	acceptanceCriteria []string
	knowledge          []nextKnowledgeEntry
	filesContext       []nextFileEntry
	constraints        []string
	roleProfile        string
	byteUsage          int
	byteBudget         int
	trimmed            []nextTrimmedEntry
}

type nextSpecSection struct {
	document string
	section  string
	content  string
}

type nextKnowledgeEntry struct {
	topic      string
	content    string
	scope      string
	confidence float64
	tier       int
}

type nextFileEntry struct {
	path string
	note string
}

type nextTrimmedEntry struct {
	entryType string
	topic     string
	sizeBytes int
}

// nextAssembleContext gathers spec sections, acceptance criteria, knowledge,
// file context, and profile conventions. All sources are best-effort: errors
// produce empty results rather than failures.
func nextAssembleContext(
	taskState map[string]any,
	parentFeature, role string,
	profileStore *kbzctx.ProfileStore,
	knowledgeSvc *service.KnowledgeService,
	intelligenceSvc *service.IntelligenceService,
) map[string]any {
	var nctx nextCtxData
	nctx.byteBudget = nextDefaultBudget

	// Role profile conventions.
	if profileStore != nil && role != "" {
		if profile, err := kbzctx.ResolveProfile(profileStore, role); err == nil {
			nctx.roleProfile = role
			nctx.constraints = append(nctx.constraints, profile.Conventions...)
		}
	}

	// Spec/design sections from document intelligence.
	if intelligenceSvc != nil && parentFeature != "" {
		if matches, err := intelligenceSvc.TraceEntity(parentFeature); err == nil {
			for _, match := range matches {
				_, content, err := intelligenceSvc.GetSection(match.DocumentID, match.SectionPath)
				if err != nil || len(content) == 0 {
					continue
				}
				title := match.SectionTitle
				if title == "" {
					title = match.SectionPath
				}
				nctx.specSections = append(nctx.specSections, nextSpecSection{
					document: match.DocumentID,
					section:  title,
					content:  string(content),
				})
			}
		}
	}

	// Extract acceptance criteria from spec sections.
	nctx.acceptanceCriteria = nextExtractCriteria(nctx.specSections)

	// Knowledge entries (Tier 2 + Tier 3), scoped to role or project.
	if knowledgeSvc != nil {
		nctx.knowledge = nextLoadKnowledge(knowledgeSvc, role)
	}

	// File context from task's files_planned.
	switch fp := taskState["files_planned"].(type) {
	case []any:
		for _, item := range fp {
			if p, ok := item.(string); ok && p != "" {
				nctx.filesContext = append(nctx.filesContext, nextFileEntry{path: p})
			}
		}
	case []string:
		for _, p := range fp {
			if p != "" {
				nctx.filesContext = append(nctx.filesContext, nextFileEntry{path: p})
			}
		}
	}

	// Byte usage and trim if over budget.
	nctx.byteUsage = nextByteCount(nctx)
	if nctx.byteUsage > nextDefaultBudget {
		nctx = nextTrimContext(nctx)
	}

	// Serialise to response map.
	specSections := make([]map[string]any, len(nctx.specSections))
	for i, s := range nctx.specSections {
		specSections[i] = map[string]any{
			"document": s.document,
			"section":  s.section,
			"content":  s.content,
		}
	}

	knowledgeOut := make([]map[string]any, len(nctx.knowledge))
	for i, ke := range nctx.knowledge {
		knowledgeOut[i] = map[string]any{
			"topic":      ke.topic,
			"content":    ke.content,
			"scope":      ke.scope,
			"confidence": ke.confidence,
		}
	}

	filesContextOut := make([]map[string]any, len(nctx.filesContext))
	for i, f := range nctx.filesContext {
		fe := map[string]any{"path": f.path}
		if f.note != "" {
			fe["note"] = f.note
		}
		filesContextOut[i] = fe
	}

	trimmedOut := make([]map[string]any, len(nctx.trimmed))
	for i, te := range nctx.trimmed {
		trimmedOut[i] = map[string]any{
			"type":       te.entryType,
			"topic":      te.topic,
			"size_bytes": te.sizeBytes,
		}
	}

	out := map[string]any{
		"spec_sections":       specSections,
		"acceptance_criteria": nctx.acceptanceCriteria,
		"knowledge":           knowledgeOut,
		"files_context":       filesContextOut,
		"constraints":         nctx.constraints,
		"byte_usage":          nctx.byteUsage,
		"byte_budget":         nctx.byteBudget,
		"trimmed":             trimmedOut,
	}
	if nctx.roleProfile != "" {
		out["role_profile"] = nctx.roleProfile
	}
	return out
}

// nextExtractCriteria extracts testable acceptance criteria from spec sections.
//
// Heuristic rules:
//  1. From sections whose title contains "acceptance", "criteria", or "requirement":
//     include all non-empty bullet/numbered list items.
//  2. From all other sections:
//     include bullet/numbered list items whose text contains "MUST", "SHALL",
//     "MUST NOT", or "SHALL NOT".
func nextExtractCriteria(sections []nextSpecSection) []string {
	var criteria []string
	seen := make(map[string]bool)

	addCriterion := func(s string) {
		s = strings.TrimSpace(s)
		if s != "" && !seen[s] {
			seen[s] = true
			criteria = append(criteria, s)
		}
	}

	for _, s := range sections {
		titleLower := strings.ToLower(s.section)
		isAcceptanceSection := strings.Contains(titleLower, "acceptance") ||
			strings.Contains(titleLower, "criteria") ||
			strings.Contains(titleLower, "requirement")

		for _, line := range strings.Split(s.content, "\n") {
			// Strip list marker to get the bare text.
			trimmed := strings.TrimSpace(line)
			text := trimmed
			for _, marker := range []string{"- ", "* ", "+ ", "• "} {
				if strings.HasPrefix(text, marker) {
					text = strings.TrimSpace(text[len(marker):])
					break
				}
			}
			// Numbered list: "1. ", "2. ", etc.
			if len(text) >= 3 && text[0] >= '0' && text[0] <= '9' {
				if idx := strings.Index(text, ". "); idx > 0 && idx < 4 {
					text = strings.TrimSpace(text[idx+2:])
				}
			}

			if text == "" || text == trimmed {
				// No list marker was stripped — not a list item; skip.
				continue
			}

			if isAcceptanceSection {
				addCriterion(text)
			} else {
				upper := strings.ToUpper(text)
				if strings.Contains(upper, " MUST ") || strings.HasSuffix(upper, " MUST") ||
					strings.Contains(upper, " SHALL ") || strings.HasSuffix(upper, " SHALL") ||
					strings.Contains(upper, " MUST NOT ") || strings.Contains(upper, " SHALL NOT ") {
					addCriterion(text)
				}
			}
		}
	}
	return criteria
}

// nextLoadKnowledge loads knowledge entries scoped to the role or project.
// Returns entries sorted by confidence descending (highest first).
func nextLoadKnowledge(svc *service.KnowledgeService, role string) []nextKnowledgeEntry {
	var entries []nextKnowledgeEntry

	for _, tc := range []struct {
		tier    int
		minConf float64
	}{
		{2, 0.3},
		{3, 0.5},
	} {
		recs, err := svc.List(service.KnowledgeFilters{
			Tier:          tc.tier,
			MinConfidence: tc.minConf,
		})
		if err != nil {
			continue
		}
		for _, rec := range recs {
			scope, _ := rec.Fields["scope"].(string)
			if scope != "project" && scope != role {
				continue
			}
			topic, _ := rec.Fields["topic"].(string)
			content, _ := rec.Fields["content"].(string)
			conf := nextFieldFloat(rec.Fields, "confidence")
			tier := nextFieldInt(rec.Fields, "tier")
			entries = append(entries, nextKnowledgeEntry{
				topic:      topic,
				content:    content,
				scope:      scope,
				confidence: conf,
				tier:       tier,
			})
		}
	}

	// Highest confidence first.
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].confidence > entries[j].confidence
	})
	return entries
}

// nextByteCount estimates the byte size of assembled context.
func nextByteCount(nctx nextCtxData) int {
	total := 0
	for _, s := range nctx.specSections {
		total += len(s.content) + len(s.document) + len(s.section) + 40
	}
	for _, ke := range nctx.knowledge {
		total += len(ke.content) + len(ke.topic) + 30
	}
	for _, c := range nctx.constraints {
		total += len(c) + 3
	}
	for _, f := range nctx.filesContext {
		total += len(f.path) + 20
	}
	for _, cr := range nctx.acceptanceCriteria {
		total += len(cr) + 3
	}
	return total
}

// nextTrimContext removes items to stay within the byte budget.
// Trim order: T3 lowest-confidence first, then T2, then spec sections from end.
func nextTrimContext(nctx nextCtxData) nextCtxData {
	var t3, t2 []nextKnowledgeEntry
	for _, ke := range nctx.knowledge {
		if ke.tier == 3 {
			t3 = append(t3, ke)
		} else {
			t2 = append(t2, ke)
		}
	}
	// Sort ascending so we remove lowest-confidence entries first.
	sort.SliceStable(t3, func(i, j int) bool { return t3[i].confidence < t3[j].confidence })
	sort.SliceStable(t2, func(i, j int) bool { return t2[i].confidence < t2[j].confidence })

	current := nextByteCount(nctx)

	for len(t3) > 0 && current > nctx.byteBudget {
		cut := t3[0]
		t3 = t3[1:]
		sz := len(cut.content) + len(cut.topic) + 30
		current -= sz
		nctx.trimmed = append(nctx.trimmed, nextTrimmedEntry{
			entryType: "knowledge",
			topic:     cut.topic,
			sizeBytes: sz,
		})
	}

	for len(t2) > 0 && current > nctx.byteBudget {
		cut := t2[0]
		t2 = t2[1:]
		sz := len(cut.content) + len(cut.topic) + 30
		current -= sz
		nctx.trimmed = append(nctx.trimmed, nextTrimmedEntry{
			entryType: "knowledge",
			topic:     cut.topic,
			sizeBytes: sz,
		})
	}

	for len(nctx.specSections) > 0 && current > nctx.byteBudget {
		cut := nctx.specSections[len(nctx.specSections)-1]
		nctx.specSections = nctx.specSections[:len(nctx.specSections)-1]
		sz := len(cut.content) + len(cut.document) + len(cut.section) + 40
		current -= sz
		nctx.trimmed = append(nctx.trimmed, nextTrimmedEntry{
			entryType: "spec",
			topic:     cut.section,
			sizeBytes: sz,
		})
	}

	// Rebuild knowledge list: T2 then T3, both descending by confidence.
	sort.SliceStable(t3, func(i, j int) bool { return t3[i].confidence > t3[j].confidence })
	sort.SliceStable(t2, func(i, j int) bool { return t2[i].confidence > t2[j].confidence })
	nctx.knowledge = append(t2, t3...)
	nctx.byteUsage = current
	return nctx
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

// nextFieldFloat reads a float64 value from a fields map.
func nextFieldFloat(fields map[string]any, key string) float64 {
	v, ok := fields[key]
	if !ok {
		return 0
	}
	switch typed := v.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	}
	return 0
}

// nextFieldInt reads an int value from a fields map.
func nextFieldInt(fields map[string]any, key string) int {
	v, ok := fields[key]
	if !ok {
		return 0
	}
	switch typed := v.(type) {
	case int:
		return typed
	case float64:
		return int(typed)
	case int64:
		return int(typed)
	}
	return 0
}
