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

	"kanbanzai/internal/service"
)

// FinishTools returns the `finish` MCP tool registered in the core group.
func FinishTools(entitySvc *service.EntityService, dispatchSvc *service.DispatchService) []server.ServerTool {
	return []server.ServerTool{finishTool(entitySvc, dispatchSvc)}
}

func finishTool(entitySvc *service.EntityService, dispatchSvc *service.DispatchService) server.ServerTool {
	tool := mcp.NewTool("finish",
		mcp.WithDescription(
			"Record task completion. Transitions the task to done (default) or needs-review, "+
				"sets completion metadata, and optionally contributes inline knowledge entries. "+
				"Accepts tasks in ready or active status — no prior next/dispatch call required. "+
				"Unblocked tasks and knowledge outcomes are reported in side_effects. "+
				"Supports batch completion via the tasks array parameter.",
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

	// Build the response matching spec §12.6.
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

	return map[string]any{
		"task": result.Task,
		"knowledge": map[string]any{
			"accepted":        accepted,
			"rejected":        rejected,
			"total_attempted": result.KnowledgeContributions.TotalAttempted,
			"total_accepted":  result.KnowledgeContributions.TotalAccepted,
		},
	}, nil
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
