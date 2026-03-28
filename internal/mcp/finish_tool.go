// Package mcp finish_tool.go — completion tool for Kanbanzai 2.0 (Track E).
//
// finish(task_id, summary) completes a task in a single call:
//   - Accepts tasks in "ready" or "active" status (lenient lifecycle).
//   - If the task is in "ready" status, auto-transitions through "active" first.
//   - Sets completed timestamp and completion_summary.
//   - Optionally contributes inline knowledge entries.
//   - Reports dependency unblocking, knowledge outcomes, and worktree creation
//     as side_effects in the response.
//
// Batch mode: finish(tasks: [{task_id, summary, ...}, ...]) processes each item
// independently with partial-failure semantics.
package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"kanbanzai/internal/id"
	"kanbanzai/internal/service"
)

// nudgeNoRetroSignals is shown when a feature completes with no retro signals recorded for any task.
const nudgeNoRetroSignals = "No retrospective signals were recorded for any task in this feature. " +
	"If you observed workflow friction, tool gaps, spec ambiguity, or things that worked well, " +
	"call finish again on any completed task with the retrospective parameter, " +
	`or use knowledge(action: contribute) with tags: ["retrospective"].`

// nudgeNoKnowledge is shown when a task completes with a summary but no knowledge or retro.
const nudgeNoKnowledge = "Consider including knowledge entries (reusable facts learned during " +
	"this task) or retrospective signals (process observations) in your finish call. " +
	"These improve context assembly for future tasks."

// FinishTools returns the `finish` MCP tool registered in the core group.
func FinishTools(entitySvc *service.EntityService, dispatchSvc *service.DispatchService) []server.ServerTool {
	return []server.ServerTool{finishTool(entitySvc, dispatchSvc)}
}

func finishTool(entitySvc *service.EntityService, dispatchSvc *service.DispatchService) server.ServerTool {
	tool := mcp.NewTool("finish",
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithTitleAnnotation("Task Completion"),
		mcp.WithDescription(
			"Record task completion with optional knowledge contribution and retrospective signals. "+
				"Transitions the task to done (default) or needs-review. "+
				"Include the retrospective parameter to record observations about workflow friction, "+
				"tool gaps, spec ambiguity, or things that worked well — these feed the retro tool "+
				"for future synthesis. "+
				"Accepts tasks in ready or active status. "+
				"Supports batch completion via the tasks array.",
		),
		mcp.WithString("task_id", mcp.Description("Task ID to complete (single-item mode)")),
		mcp.WithString("summary", mcp.Description("Brief description of what was accomplished")),
		mcp.WithString("to_status", mcp.Description("Target status: done (default) or needs-review")),
		mcp.WithArray("files_modified", mcp.WithStringItems(), mcp.Description("Files created or modified")),
		mcp.WithString("verification", mcp.Description("Testing or verification performed")),
		mcp.WithArray("knowledge", mcp.Description(
			"Inline knowledge entries to contribute. Each entry: "+
				"{topic, content, scope} required; {tags, tier} optional. "+
				"Duplicates are rejected per-entry without blocking completion.",
		)),
		mcp.WithArray("retrospective", mcp.Description(
			"Retrospective signals to record at task completion. Each entry: "+
				"{category, observation, severity} required; {suggestion, related_decision} optional. "+
				"Valid categories: workflow-friction, tool-gap, tool-friction, spec-ambiguity, "+
				"context-gap, decomposition-issue, design-gap, worked-well. "+
				"Valid severities: minor, moderate, significant. "+
				"Invalid signals are rejected per-entry without blocking completion.",
		)),
		mcp.WithArray("tasks", mcp.Description(
			"Batch mode: array of task completion objects. "+
				"Each item contains task_id, summary, and optional fields. "+
				"Items are processed independently; a failure on one does not affect others.",
		)),
	)

	handler := WithSideEffects(func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		args, _ := req.Params.Arguments.(map[string]any)

		// Signal mutation so side_effects: [] is always present in the response
		// even when no cascades occur (spec §8.4: "The field is never omitted").
		SignalMutation(ctx)

		// Batch mode: tasks array provided.
		if IsBatchInput(args, "tasks") {
			items, _ := args["tasks"].([]any)
			return ExecuteBatch(ctx, items, func(ctx context.Context, item any) (string, any, error) {
				input := parseFinishItem(item)
				input.Batch = true
				result, err := finishOne(ctx, input, entitySvc, dispatchSvc)
				return input.TaskID, result, err
			})
		}

		// Single mode.
		input := parseFinishArgs(args)
		return finishOne(ctx, input, entitySvc, dispatchSvc)
	})

	return server.ServerTool{Tool: tool, Handler: handler}
}

// ─── Input types ─────────────────────────────────────────────────────────────

// finishInput holds the parameters for a single task completion.
type finishInput struct {
	TaskID        string
	Summary       string
	ToStatus      string
	FilesModified []string
	Verification  string
	Knowledge     []service.KnowledgeEntryInput
	RetroSignals  []service.RetroSignalInput
	Batch         bool // true when called from the batch path
}

// parseFinishArgs parses finishInput from the top-level MCP arguments map.
func parseFinishArgs(args map[string]any) finishInput {
	if args == nil {
		return finishInput{}
	}
	return finishInput{
		TaskID:        finishStringArg(args, "task_id"),
		Summary:       finishStringArg(args, "summary"),
		ToStatus:      finishStringArg(args, "to_status"),
		FilesModified: finishStringSliceArg(args, "files_modified"),
		Verification:  finishStringArg(args, "verification"),
		Knowledge:     parseFinishKnowledge(args),
		RetroSignals:  parseFinishRetro(args),
	}
}

// parseFinishItem parses finishInput from a single batch item (a map).
// If item is not a map, only the TaskID is populated (produces a validation error in finishOne).
func parseFinishItem(item any) finishInput {
	m, ok := item.(map[string]any)
	if !ok {
		return finishInput{}
	}
	return finishInput{
		TaskID:        finishStringArg(m, "task_id"),
		Summary:       finishStringArg(m, "summary"),
		ToStatus:      finishStringArg(m, "to_status"),
		FilesModified: finishStringSliceArg(m, "files_modified"),
		Verification:  finishStringArg(m, "verification"),
		Knowledge:     parseFinishKnowledge(m),
		RetroSignals:  parseFinishRetro(m),
	}
}

// ─── Core completion logic ────────────────────────────────────────────────────

// finishOne performs a single task completion. It is called by both the
// single-item handler and the batch handler.
//
// Side effects pushed onto ctx:
//   - worktree_created  — if a worktree was created during the ready→active auto-transition
//   - task_unblocked    — for each task unblocked by the completion
//   - knowledge_contributed — for each accepted knowledge entry
//   - knowledge_rejected   — for each rejected knowledge entry
func finishOne(
	ctx context.Context,
	input finishInput,
	entitySvc *service.EntityService,
	dispatchSvc *service.DispatchService,
) (any, error) {
	if strings.TrimSpace(input.TaskID) == "" {
		return nil, fmt.Errorf("task_id is required")
	}
	if strings.TrimSpace(input.Summary) == "" {
		return nil, fmt.Errorf("summary is required")
	}

	// Load the task to inspect its current status.
	task, err := entitySvc.Get("task", input.TaskID, "")
	if err != nil {
		return nil, fmt.Errorf("task %s not found", input.TaskID)
	}

	status, _ := task.State["status"].(string)

	switch status {
	case "ready":
		// Lenient lifecycle: auto-transition ready → active before completing.
		// This fires the worktree creation hook (if configured) and the
		// dependency unblocking hook.
		activeResult, err := entitySvc.UpdateStatus(service.UpdateStatusInput{
			Type:   "task",
			ID:     task.ID,
			Slug:   task.Slug,
			Status: "active",
		})
		if err != nil {
			return nil, fmt.Errorf("auto-transition to active: %w", err)
		}
		// Report worktree creation as a side effect if the hook fired.
		if wt := activeResult.WorktreeHookResult; wt != nil && wt.Created {
			PushSideEffect(ctx, SideEffect{
				Type:       SideEffectWorktreeCreated,
				EntityID:   task.ID,
				EntityType: "task",
				Extra: map[string]string{
					"worktree_id": wt.WorktreeID,
					"branch":      wt.Branch,
					"path":        wt.Path,
				},
			})
		}

	case "active":
		// Already active — proceed directly to completion.

	default:
		return nil, fmt.Errorf(
			"task %s is in status %q, expected \"ready\" or \"active\"",
			task.ID, status,
		)
	}

	// Complete the task (active → done or needs-review).
	result, err := dispatchSvc.CompleteTask(service.CompleteInput{
		TaskID:                input.TaskID,
		Summary:               input.Summary,
		ToStatus:              input.ToStatus,
		FilesModified:         input.FilesModified,
		VerificationPerformed: input.Verification,
		KnowledgeEntries:      input.Knowledge,
		RetroSignals:          input.RetroSignals,
	})
	if err != nil {
		return nil, err
	}

	// Push task_unblocked side effects for any tasks freed by this completion.
	for _, u := range result.UnblockedTasks {
		PushSideEffect(ctx, SideEffect{
			Type:       SideEffectTaskUnblocked,
			EntityID:   u.TaskID,
			EntityType: "task",
			FromStatus: u.PreviousStatus,
			ToStatus:   u.Status,
			Trigger:    fmt.Sprintf("All dependencies of %s are now in terminal state", u.TaskID),
		})
	}

	// Push knowledge side effects.
	for _, a := range result.KnowledgeContributions.Accepted {
		PushSideEffect(ctx, SideEffect{
			Type:     SideEffectKnowledgeContributed,
			EntityID: a.EntryID,
			Extra:    map[string]string{"topic": a.Topic},
		})
	}
	for _, r := range result.KnowledgeContributions.Rejected {
		PushSideEffect(ctx, SideEffect{
			Type:  SideEffectKnowledgeRejected,
			Extra: map[string]string{"topic": r.Topic, "reason": r.Reason},
		})
	}

	// Push retrospective signal side effects.
	for _, a := range result.RetroContributions.Accepted {
		PushSideEffect(ctx, SideEffect{
			Type:     SideEffectRetroSignalContributed,
			EntityID: a.EntryID,
			Extra:    map[string]string{"topic": a.Topic, "category": a.Category},
		})
	}

	// Build the response matching spec §12.6 and P5 §6.2.
	type acceptedKE struct {
		EntryID string `json:"entry_id"`
		Topic   string `json:"topic"`
	}
	type rejectedKE struct {
		Topic  string `json:"topic"`
		Reason string `json:"reason"`
	}

	accepted := make([]acceptedKE, len(result.KnowledgeContributions.Accepted))
	for i, a := range result.KnowledgeContributions.Accepted {
		accepted[i] = acceptedKE{EntryID: a.EntryID, Topic: a.Topic}
	}
	rejected := make([]rejectedKE, len(result.KnowledgeContributions.Rejected))
	for i, r := range result.KnowledgeContributions.Rejected {
		rejected[i] = rejectedKE{Topic: r.Topic, Reason: r.Reason}
	}

	// Enrich task output with display_id and entity_ref.
	if tid, ok := result.Task["id"].(string); ok {
		displayID := id.FormatFullDisplay(tid)
		result.Task["display_id"] = displayID
		slug, _ := result.Task["slug"].(string)
		label, _ := result.Task["label"].(string)
		result.Task["entity_ref"] = id.FormatEntityRef(displayID, slug, label)
	}

	resp := map[string]any{
		"task": result.Task,
		"knowledge": map[string]any{
			"accepted":        accepted,
			"rejected":        rejected,
			"total_attempted": result.KnowledgeContributions.TotalAttempted,
			"total_accepted":  result.KnowledgeContributions.TotalAccepted,
		},
	}

	// Include the retrospective section only when signals were submitted,
	// preserving the pre-P5 response shape when the parameter is absent (P5-1.1).
	if result.RetroContributions.TotalAttempted > 0 {
		type acceptedRetro struct {
			EntryID  string `json:"entry_id"`
			Topic    string `json:"topic"`
			Category string `json:"category"`
		}
		type rejectedRetro struct {
			Category    string `json:"category"`
			Observation string `json:"observation"`
			Reason      string `json:"reason"`
		}
		retroAccepted := make([]acceptedRetro, len(result.RetroContributions.Accepted))
		for i, a := range result.RetroContributions.Accepted {
			retroAccepted[i] = acceptedRetro{EntryID: a.EntryID, Topic: a.Topic, Category: a.Category}
		}
		retroRejected := make([]rejectedRetro, len(result.RetroContributions.Rejected))
		for i, r := range result.RetroContributions.Rejected {
			retroRejected[i] = rejectedRetro{Category: r.Category, Observation: r.Observation, Reason: r.Reason}
		}
		resp["retrospective"] = map[string]any{
			"accepted":        retroAccepted,
			"rejected":        retroRejected,
			"total_attempted": result.RetroContributions.TotalAttempted,
			"total_accepted":  result.RetroContributions.TotalAccepted,
		}
	}

	// Nudge logic — evaluates feature completion status and provides contextual hints.
	// Nudge 1 takes priority over Nudge 2 when both conditions are met.
	if !input.Batch {
		nudge1Fired := false

		// Nudge 1: feature just completed with no retro signals anywhere in the feature.
		parentFeatureID, _ := result.Task["parent_feature"].(string)
		if parentFeatureID != "" {
			siblings, err := entitySvc.ListEntitiesFiltered(service.ListFilteredInput{
				Type:   "task",
				Parent: parentFeatureID,
			})
			if err == nil {
				allTerminal := true
				siblingIDs := make([]string, 0, len(siblings))
				for _, s := range siblings {
					siblingIDs = append(siblingIDs, s.ID)
					st, _ := s.State["status"].(string)
					if !isFinishTerminal(st) {
						allTerminal = false
						break
					}
				}
				if allTerminal && !dispatchSvc.AnyTaskHasRetroSignals(siblingIDs) {
					resp["nudge"] = nudgeNoRetroSignals
					nudge1Fired = true
				}
			}
		}

		// Nudge 2: this call had no knowledge or retro, and summary was provided.
		if !nudge1Fired && input.Summary != "" &&
			len(input.Knowledge) == 0 && len(input.RetroSignals) == 0 {
			resp["nudge"] = nudgeNoKnowledge
		}
	}

	return resp, nil
}

// isFinishTerminal returns true for task statuses that count as complete.
func isFinishTerminal(status string) bool {
	switch status {
	case "done", "needs-review", "not-planned", "duplicate", "cancelled":
		return true
	}
	return false
}

// ─── Argument parsing helpers ────────────────────────────────────────────────

// finishStringArg extracts and trims a string field from an args map.
func finishStringArg(args map[string]any, key string) string {
	s, _ := args[key].(string)
	return strings.TrimSpace(s)
}

// finishStringSliceArg extracts a []string from an args map. Returns nil when absent.
func finishStringSliceArg(args map[string]any, key string) []string {
	v, ok := args[key]
	if !ok {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// parseFinishRetro extracts retrospective signals from an args map.
// Returns nil when the "retrospective" key is absent or empty.
func parseFinishRetro(args map[string]any) []service.RetroSignalInput {
	raw, ok := args["retrospective"]
	if !ok {
		return nil
	}
	arr, ok := raw.([]any)
	if !ok || len(arr) == 0 {
		return nil
	}
	signals := make([]service.RetroSignalInput, 0, len(arr))
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		signals = append(signals, service.RetroSignalInput{
			Category:        finishStringArg(m, "category"),
			Observation:     finishStringArg(m, "observation"),
			Severity:        finishStringArg(m, "severity"),
			Suggestion:      finishStringArg(m, "suggestion"),
			RelatedDecision: finishStringArg(m, "related_decision"),
		})
	}
	return signals
}

// parseFinishKnowledge extracts inline knowledge entries from an args map.
// Each entry must have topic, content, and scope; tags and tier are optional.
func parseFinishKnowledge(args map[string]any) []service.KnowledgeEntryInput {
	raw, ok := args["knowledge"]
	if !ok {
		return nil
	}
	arr, ok := raw.([]any)
	if !ok || len(arr) == 0 {
		return nil
	}

	entries := make([]service.KnowledgeEntryInput, 0, len(arr))
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		ke := service.KnowledgeEntryInput{
			Topic:   finishStringArg(m, "topic"),
			Content: finishStringArg(m, "content"),
			Scope:   finishStringArg(m, "scope"),
		}
		if tierVal, ok := m["tier"]; ok {
			if tierF, ok := tierVal.(float64); ok {
				ke.Tier = int(tierF)
			}
		}
		if tagsVal, ok := m["tags"]; ok {
			if tagsArr, ok := tagsVal.([]any); ok {
				for _, t := range tagsArr {
					if s, ok := t.(string); ok {
						ke.Tags = append(ke.Tags, s)
					}
				}
			}
		}
		entries = append(entries, ke)
	}
	return entries
}
