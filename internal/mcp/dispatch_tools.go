package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	chk "kanbanzai/internal/checkpoint"
	kbzctx "kanbanzai/internal/context"
	"kanbanzai/internal/service"
)

// DispatchTools returns all dispatch/completion/checkpoint MCP tools.
func DispatchTools(
	dispatchSvc *service.DispatchService,
	checkpointStore *chk.Store,
	profileStore *kbzctx.ProfileStore,
	knowledgeSvc *service.KnowledgeService,
	entitySvc *service.EntityService,
	intelligenceSvc *service.IntelligenceService,
) []server.ServerTool {
	return []server.ServerTool{
		dispatchTaskTool(dispatchSvc, profileStore, knowledgeSvc, entitySvc, intelligenceSvc),
		completeTaskTool(dispatchSvc),
		humanCheckpointTool(checkpointStore),
		humanCheckpointRespondTool(checkpointStore),
		humanCheckpointGetTool(checkpointStore),
		humanCheckpointListTool(checkpointStore),
	}
}

func dispatchTaskTool(
	svc *service.DispatchService,
	profileStore *kbzctx.ProfileStore,
	knowledgeSvc *service.KnowledgeService,
	entitySvc *service.EntityService,
	intelligenceSvc *service.IntelligenceService,
) server.ServerTool {
	tool := mcp.NewTool("dispatch_task",
		mcp.WithDescription("Atomically claim a ready task and return its context packet. Transitions the task from ready to active, records dispatch metadata, and assembles the context packet for the executing agent."),
		mcp.WithString("task_id", mcp.Required(), mcp.Description("Task ID to dispatch (must be in ready status)")),
		mcp.WithString("role", mcp.Required(), mcp.Description("Role profile ID for the executing agent (e.g. backend, frontend)")),
		mcp.WithString("dispatched_by", mcp.Required(), mcp.Description("Identity string of the orchestrating agent")),
		mcp.WithString("orchestration_context", mcp.Description("Optional handoff note injected into the context packet (ephemeral, not persisted)")),
		mcp.WithNumber("max_bytes", mcp.Description("Byte budget for context assembly (default: 30720)")),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		taskID, err := request.RequireString("task_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		role, err := request.RequireString("role")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		dispatchedBy, err := request.RequireString("dispatched_by")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		orchCtx := request.GetString("orchestration_context", "")
		maxBytes := int(request.GetFloat("max_bytes", 0))

		// Claim the task (ready → active, set dispatch metadata).
		result, err := svc.DispatchTask(service.DispatchInput{
			TaskID:       taskID,
			Role:         role,
			DispatchedBy: dispatchedBy,
		})
		if err != nil {
			return mcp.NewToolResultErrorFromErr("dispatch_task failed", err), nil
		}

		// Assemble context packet now that the task is claimed.
		assemblyInput := kbzctx.AssemblyInput{
			Role:                 role,
			TaskID:               taskID,
			MaxBytes:             maxBytes,
			OrchestrationContext: orchCtx,
		}
		ctxResult, err := kbzctx.Assemble(assemblyInput, profileStore, knowledgeSvc, entitySvc, intelligenceSvc)
		if err != nil {
			// Task is claimed but context assembly failed — report with recovery hint.
			return mcp.NewToolResultError(fmt.Sprintf(
				"task %s is now active but context_assemble failed: %s — use context_assemble manually to retrieve the context packet",
				taskID, err.Error(),
			)), nil
		}

		type trimmedEntry struct {
			EntryID   string `json:"entry_id,omitempty"`
			Type      string `json:"type"`
			Topic     string `json:"topic,omitempty"`
			Tier      int    `json:"tier,omitempty"`
			SizeBytes int    `json:"size_bytes"`
		}
		trimmedItems := make([]trimmedEntry, len(ctxResult.TrimmedEntries))
		for i, te := range ctxResult.TrimmedEntries {
			trimmedItems[i] = trimmedEntry{
				EntryID:   te.EntryID,
				Type:      te.Type,
				Topic:     te.Topic,
				Tier:      te.Tier,
				SizeBytes: te.SizeBytes,
			}
		}

		resp := map[string]any{
			"task": result.Task,
			"context": map[string]any{
				"role":       ctxResult.Role,
				"byte_usage": ctxResult.ByteCount,
				"trimmed":    trimmedItems,
				"items":      ctxResult.Items,
			},
		}

		data, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshal result: %s", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func completeTaskTool(svc *service.DispatchService) server.ServerTool {
	tool := mcp.NewTool("complete_task",
		mcp.WithDescription("Close the dispatch loop for a completed task. Transitions the task to done (or needs-review), records completion metadata, and contributes knowledge entries to the knowledge base."),
		mcp.WithString("task_id", mcp.Required(), mcp.Description("Task ID to complete (must be in active status)")),
		mcp.WithString("summary", mcp.Required(), mcp.Description("Brief description of what was accomplished")),
		mcp.WithString("to_status", mcp.Description("Target status: done (default) or needs-review")),
		mcp.WithArray("files_modified", mcp.WithStringItems(), mcp.Description("Files created or modified")),
		mcp.WithString("verification_performed", mcp.Description("Testing or verification carried out")),
		mcp.WithString("blockers_encountered", mcp.Description("Obstacles noted for future work")),
		mcp.WithArray("knowledge_entries", mcp.Description("Knowledge entries to contribute (array of objects with topic, content, scope, tier, tags)")),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		taskID, err := request.RequireString("task_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		summary, err := request.RequireString("summary")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		toStatus := request.GetString("to_status", "done")
		filesModified := request.GetStringSlice("files_modified", nil)
		verificationPerformed := request.GetString("verification_performed", "")
		blockersEncountered := request.GetString("blockers_encountered", "")

		// Parse knowledge entries from the raw arguments array.
		var knowledgeEntries []service.KnowledgeEntryInput
		if args, ok := request.Params.Arguments.(map[string]any); ok {
			if rawEntries, ok := args["knowledge_entries"]; ok {
				if entriesData, err := json.Marshal(rawEntries); err == nil {
					var entries []map[string]any
					if json.Unmarshal(entriesData, &entries) == nil {
						for _, e := range entries {
							ke := service.KnowledgeEntryInput{
								Topic:   stringMapVal(e, "topic"),
								Content: stringMapVal(e, "content"),
								Scope:   stringMapVal(e, "scope"),
							}
							if tierVal, ok := e["tier"]; ok {
								if tierF, ok := tierVal.(float64); ok {
									ke.Tier = int(tierF)
								}
							}
							if tagsVal, ok := e["tags"]; ok {
								if tagsArr, ok := tagsVal.([]any); ok {
									for _, t := range tagsArr {
										if s, ok := t.(string); ok {
											ke.Tags = append(ke.Tags, s)
										}
									}
								}
							}
							knowledgeEntries = append(knowledgeEntries, ke)
						}
					}
				}
			}
		}

		result, err := svc.CompleteTask(service.CompleteInput{
			TaskID:                taskID,
			Summary:               summary,
			ToStatus:              toStatus,
			FilesModified:         filesModified,
			VerificationPerformed: verificationPerformed,
			BlockersEncountered:   blockersEncountered,
			KnowledgeEntries:      knowledgeEntries,
		})
		if err != nil {
			return mcp.NewToolResultErrorFromErr("complete_task failed", err), nil
		}

		type acceptedEntry struct {
			EntryID string `json:"entry_id"`
			Topic   string `json:"topic"`
		}
		type rejectedEntry struct {
			Topic  string `json:"topic"`
			Reason string `json:"reason"`
		}

		accepted := make([]acceptedEntry, len(result.KnowledgeContributions.Accepted))
		for i, a := range result.KnowledgeContributions.Accepted {
			accepted[i] = acceptedEntry{EntryID: a.EntryID, Topic: a.Topic}
		}
		rejected := make([]rejectedEntry, len(result.KnowledgeContributions.Rejected))
		for i, r := range result.KnowledgeContributions.Rejected {
			rejected[i] = rejectedEntry{Topic: r.Topic, Reason: r.Reason}
		}

		type unblockedEntry struct {
			TaskID string `json:"task_id"`
			Slug   string `json:"slug"`
			Status string `json:"status"`
		}
		unblockedTasks := make([]unblockedEntry, 0, len(result.UnblockedTasks))
		for _, u := range result.UnblockedTasks {
			unblockedTasks = append(unblockedTasks, unblockedEntry{
				TaskID: u.TaskID,
				Slug:   u.Slug,
				Status: u.Status,
			})
		}

		resp := map[string]any{
			"task": result.Task,
			"knowledge_contributions": map[string]any{
				"accepted":        accepted,
				"rejected":        rejected,
				"total_attempted": result.KnowledgeContributions.TotalAttempted,
				"total_accepted":  result.KnowledgeContributions.TotalAccepted,
			},
			"unblocked_tasks": unblockedTasks,
		}
		if blockersEncountered != "" {
			resp["blockers_noted"] = blockersEncountered
		}

		data, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshal result: %s", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func humanCheckpointTool(store *chk.Store) server.ServerTool {
	tool := mcp.NewTool("human_checkpoint",
		mcp.WithDescription("Record a structured decision point requiring human input. After calling this, stop dispatching new tasks until you poll human_checkpoint_get and receive status: responded."),
		mcp.WithString("question", mcp.Required(), mcp.Description("The decision or question requiring human input")),
		mcp.WithString("context", mcp.Required(), mcp.Description("Background information to help the human answer")),
		mcp.WithString("orchestration_summary", mcp.Required(), mcp.Description("Brief state of the orchestration session at checkpoint time")),
		mcp.WithString("created_by", mcp.Required(), mcp.Description("Identity of the orchestrating agent")),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		question, err := request.RequireString("question")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		contextStr, err := request.RequireString("context")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		orchSummary, err := request.RequireString("orchestration_summary")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		createdBy, err := request.RequireString("created_by")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		record := chk.Record{
			Question:             question,
			Context:              contextStr,
			OrchestrationSummary: orchSummary,
			Status:               chk.StatusPending,
			CreatedAt:            time.Now().UTC(),
			CreatedBy:            createdBy,
		}

		created, err := store.Create(record)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("human_checkpoint failed", err), nil
		}

		resp := map[string]any{
			"checkpoint_id": created.ID,
			"status":        string(created.Status),
			"created_at":    created.CreatedAt.Format(time.RFC3339),
			"message":       "Checkpoint recorded. Stop dispatching new tasks. Poll human_checkpoint_get with checkpoint_id until status is 'responded'.",
		}
		data, _ := json.MarshalIndent(resp, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func humanCheckpointRespondTool(store *chk.Store) server.ServerTool {
	tool := mcp.NewTool("human_checkpoint_respond",
		mcp.WithDescription("Record a human response to a pending checkpoint."),
		mcp.WithString("checkpoint_id", mcp.Required(), mcp.Description("CHK ID of the checkpoint to respond to")),
		mcp.WithString("response", mcp.Required(), mcp.Description("The human's answer or decision")),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		checkpointID, err := request.RequireString("checkpoint_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		response, err := request.RequireString("response")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		record, err := store.Get(checkpointID)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("checkpoint not found", err), nil
		}

		if record.Status == chk.StatusResponded {
			return mcp.NewToolResultError(fmt.Sprintf("checkpoint %s is already responded", checkpointID)), nil
		}

		now := time.Now().UTC()
		record.Status = chk.StatusResponded
		record.RespondedAt = &now
		record.Response = &response

		updated, err := store.Update(record)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("human_checkpoint_respond failed", err), nil
		}

		resp := map[string]any{
			"checkpoint_id": updated.ID,
			"status":        string(updated.Status),
			"responded_at":  updated.RespondedAt.Format(time.RFC3339),
		}
		data, _ := json.MarshalIndent(resp, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func humanCheckpointGetTool(store *chk.Store) server.ServerTool {
	tool := mcp.NewTool("human_checkpoint_get",
		mcp.WithDescription("Get the current state of a checkpoint. Poll this after calling human_checkpoint until status is 'responded'."),
		mcp.WithString("checkpoint_id", mcp.Required(), mcp.Description("CHK ID of the checkpoint to retrieve")),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		checkpointID, err := request.RequireString("checkpoint_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		record, err := store.Get(checkpointID)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("checkpoint not found", err), nil
		}

		resp := checkpointToMap(record)
		data, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshal result: %s", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func humanCheckpointListTool(store *chk.Store) server.ServerTool {
	tool := mcp.NewTool("human_checkpoint_list",
		mcp.WithDescription("List checkpoint records. Optionally filter by status. Returns total count and pending count."),
		mcp.WithString("status", mcp.Description("Optional status filter: pending or responded")),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		statusFilter := request.GetString("status", "")

		records, err := store.List(statusFilter)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("human_checkpoint_list failed", err), nil
		}

		// Count pending checkpoints (even when filter is applied, report overall).
		pendingCount := 0
		allRecords := records
		if statusFilter != "" {
			// Need all records to compute pending_count accurately.
			allRecords, _ = store.List("")
		}
		for _, r := range allRecords {
			if r.Status == chk.StatusPending {
				pendingCount++
			}
		}

		checkpoints := make([]map[string]any, len(records))
		for i, r := range records {
			checkpoints[i] = checkpointToMap(r)
		}

		resp := map[string]any{
			"checkpoints":   checkpoints,
			"total":         len(records),
			"pending_count": pendingCount,
		}
		data, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshal result: %s", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

// checkpointToMap converts a checkpoint Record to a response map.
func checkpointToMap(r chk.Record) map[string]any {
	m := map[string]any{
		"id":                    r.ID,
		"question":              r.Question,
		"context":               r.Context,
		"orchestration_summary": r.OrchestrationSummary,
		"status":                string(r.Status),
		"created_at":            r.CreatedAt.Format(time.RFC3339),
		"created_by":            r.CreatedBy,
		"responded_at":          nil,
		"response":              nil,
	}
	if r.RespondedAt != nil {
		m["responded_at"] = r.RespondedAt.Format(time.RFC3339)
	}
	if r.Response != nil {
		m["response"] = *r.Response
	}
	return m
}

// stringMapVal safely reads a string value from a map[string]any.
func stringMapVal(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
