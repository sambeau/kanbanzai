package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	chk "github.com/sambeau/kanbanzai/internal/checkpoint"
)

// CheckpointTool returns the 2.0 consolidated checkpoint tool.
// It consolidates human_checkpoint, human_checkpoint_get, human_checkpoint_respond,
// and human_checkpoint_list into a single tool (spec §22.1).
func CheckpointTool(store *chk.Store) []server.ServerTool {
	return []server.ServerTool{checkpointTool(store)}
}

func checkpointTool(store *chk.Store) server.ServerTool {
	tool := mcp.NewTool("checkpoint",
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithTitleAnnotation("Human Decision Checkpoints"),
		mcp.WithDescription(
			"Pause automated orchestration when a decision requires human input. "+
				"Use the create → get (poll) → respond workflow: after create, "+
				"stop dispatching new tasks and poll with get until status changes to 'responded'. "+
				"Do NOT use for information-only messages — checkpoints block work until a human responds. "+
				"Actions: create (question, context, orchestration_summary, created_by required), "+
				"get (checkpoint_id required), respond (checkpoint_id and response required), "+
				"list (optional status filter: pending or responded).",
		),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("Action: create, get, respond, list"),
		),
		// create parameters
		mcp.WithString("question",
			mcp.Description("The decision or question requiring human input (create only)"),
		),
		mcp.WithString("context",
			mcp.Description("Background information to help the human answer (create only)"),
		),
		mcp.WithString("orchestration_summary",
			mcp.Description("Brief state of the orchestration session at checkpoint time (create only)"),
		),
		mcp.WithString("created_by",
			mcp.Description("Identity of the orchestrating agent (create only)"),
		),
		// get / respond parameters
		mcp.WithString("checkpoint_id",
			mcp.Description("CHK ID of the checkpoint — required for get and respond"),
		),
		// respond parameters
		mcp.WithString("response",
			mcp.Description("The human's answer or decision (respond only)"),
		),
		// list parameters
		mcp.WithString("status",
			mcp.Description("Optional status filter: pending or responded (list only)"),
		),
	)

	handler := WithSideEffects(func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		return DispatchAction(ctx, req, map[string]ActionHandler{
			"create":  checkpointCreateAction(store),
			"get":     checkpointGetAction(store),
			"respond": checkpointRespondAction(store),
			"list":    checkpointListAction(store),
		})
	})

	return server.ServerTool{Tool: tool, Handler: handler}
}

// ─── create ──────────────────────────────────────────────────────────────────

func checkpointCreateAction(store *chk.Store) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		SignalMutation(ctx)

		question, err := req.RequireString("question")
		if err != nil {
			return inlineErr("missing_parameter", "Cannot create checkpoint: question is missing.\n\nTo resolve:\n  Provide question: checkpoint(action: \"create\", question: \"...\", context: \"...\", orchestration_summary: \"...\", created_by: \"...\")")
		}
		contextStr, err := req.RequireString("context")
		if err != nil {
			return inlineErr("missing_parameter", "Cannot create checkpoint: context is missing.\n\nTo resolve:\n  Provide context: checkpoint(action: \"create\", question: \"...\", context: \"...\", orchestration_summary: \"...\", created_by: \"...\")")
		}
		orchSummary, err := req.RequireString("orchestration_summary")
		if err != nil {
			return inlineErr("missing_parameter", "Cannot create checkpoint: orchestration_summary is missing.\n\nTo resolve:\n  Provide orchestration_summary: checkpoint(action: \"create\", question: \"...\", context: \"...\", orchestration_summary: \"...\", created_by: \"...\")")
		}
		createdBy, err := req.RequireString("created_by")
		if err != nil {
			return inlineErr("missing_parameter", "Cannot create checkpoint: created_by is missing.\n\nTo resolve:\n  Provide created_by: checkpoint(action: \"create\", question: \"...\", context: \"...\", orchestration_summary: \"...\", created_by: \"agent-name\")")
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
			return nil, fmt.Errorf("Cannot create checkpoint: %w.\n\nTo resolve:\n  Check that the checkpoint store is accessible and retry", err)
		}

		return map[string]any{
			"checkpoint_id": created.ID,
			"status":        string(created.Status),
			"created_at":    created.CreatedAt.Format(time.RFC3339),
			"message":       "Checkpoint recorded. Stop dispatching new tasks. Poll checkpoint with action=get until status is 'responded'.",
		}, nil
	}
}

// ─── get ──────────────────────────────────────────────────────────────────────

func checkpointGetAction(store *chk.Store) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		checkpointID, err := req.RequireString("checkpoint_id")
		if err != nil {
			return inlineErr("missing_parameter", "Cannot get checkpoint: checkpoint_id is missing.\n\nTo resolve:\n  Provide checkpoint_id: checkpoint(action: \"get\", checkpoint_id: \"CHK-...\")")
		}

		record, err := store.Get(checkpointID)
		if err != nil {
			return nil, fmt.Errorf("Cannot get checkpoint %s: %w.\n\nTo resolve:\n  Verify the checkpoint ID using checkpoint(action: \"list\")", checkpointID, err)
		}

		return checkpointRecordToMap(record), nil
	}
}

// ─── respond ──────────────────────────────────────────────────────────────────

func checkpointRespondAction(store *chk.Store) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		SignalMutation(ctx)

		checkpointID, err := req.RequireString("checkpoint_id")
		if err != nil {
			return inlineErr("missing_parameter", "Cannot respond to checkpoint: checkpoint_id is missing.\n\nTo resolve:\n  Provide checkpoint_id: checkpoint(action: \"respond\", checkpoint_id: \"CHK-...\", response: \"...\")")
		}
		response, err := req.RequireString("response")
		if err != nil {
			return inlineErr("missing_parameter", "Cannot respond to checkpoint: response is missing.\n\nTo resolve:\n  Provide response: checkpoint(action: \"respond\", checkpoint_id: \"CHK-...\", response: \"...\")")
		}

		record, err := store.Get(checkpointID)
		if err != nil {
			return nil, fmt.Errorf("Cannot respond to checkpoint %s: lookup failed: %w.\n\nTo resolve:\n  Verify the checkpoint ID using checkpoint(action: \"list\")", checkpointID, err)
		}

		if record.Status == chk.StatusResponded {
			return inlineErr("already_responded",
				fmt.Sprintf("Cannot respond to checkpoint %s: checkpoint already has a response.\n\nTo resolve:\n  Use checkpoint(action: \"get\", checkpoint_id: \"%s\") to view the existing response", checkpointID, checkpointID))
		}

		now := time.Now().UTC()
		record.Status = chk.StatusResponded
		record.RespondedAt = &now
		record.Response = &response

		updated, err := store.Update(record)
		if err != nil {
			return nil, fmt.Errorf("Cannot respond to checkpoint %s: update failed: %w.\n\nTo resolve:\n  Verify the checkpoint status and retry", checkpointID, err)
		}

		resp := map[string]any{
			"checkpoint_id": updated.ID,
			"status":        string(updated.Status),
		}
		if updated.RespondedAt != nil {
			resp["responded_at"] = updated.RespondedAt.Format(time.RFC3339)
		}

		return resp, nil
	}
}

// ─── list ─────────────────────────────────────────────────────────────────────

func checkpointListAction(store *chk.Store) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		statusFilter := req.GetString("status", "")

		records, err := store.List(statusFilter)
		if err != nil {
			return nil, fmt.Errorf("Cannot list checkpoints: %w.\n\nTo resolve:\n  Check the status filter value (valid: pending, responded) or omit it to list all", err)
		}

		// Count pending checkpoints across all records for accurate reporting.
		pendingCount := 0
		allRecords := records
		if statusFilter != "" {
			allRecords, _ = store.List("")
		}
		for _, r := range allRecords {
			if r.Status == chk.StatusPending {
				pendingCount++
			}
		}

		checkpoints := make([]map[string]any, len(records))
		for i, r := range records {
			checkpoints[i] = checkpointRecordToMap(r)
		}

		return map[string]any{
			"checkpoints":   checkpoints,
			"total":         len(records),
			"pending_count": pendingCount,
		}, nil
	}
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// checkpointRecordToMap converts a checkpoint Record to a response map.
func checkpointRecordToMap(r chk.Record) map[string]any {
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
