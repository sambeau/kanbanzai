// Package mcp develop_tool.go — top-level tool for development dispatch
// operations introduced by the Composite Tools feature (B43).
//
//	develop(action: "dispatch", feature_id: "FEAT-...", role: "...")
//
// The dispatch action identifies the ready task frontier for a feature in
// developing status, runs conflict analysis on ready tasks, transitions
// conflict-safe tasks from ready to active, and returns handoff-style context
// for each dispatched task. It does NOT spawn sub-agents — the calling agent
// is responsible for that.
package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/sambeau/kanbanzai/internal/service"
)

// DevelopTool returns the `develop` MCP tool registered unconditionally.
// The tool is new in Kanbanzai 2.1 (B43 — Composite Tools).
func DevelopTool(entitySvc *service.EntityService, conflictSvc *service.ConflictService, dispatchSvc *service.DispatchService, knowledgeSvc *service.KnowledgeService, intelligenceSvc *service.IntelligenceService, docSvc *service.DocumentService) []server.ServerTool {
	tool := mcp.NewTool("develop",
		mcp.WithTitleAnnotation("Development Dispatch"),
		mcp.WithDescription(
			"Orchestrator dispatch tool — identifies the ready task frontier for a feature, "+
				"runs conflict analysis, transitions conflict-safe tasks to active, and returns "+
				"handoff-style context for each dispatched task. Does NOT spawn sub-agents. "+
				"Actions: dispatch.",
		),
		mcp.WithString("action", mcp.Required(), mcp.Description("Action: dispatch")),
		mcp.WithString("feature_id", mcp.Required(), mcp.Description("Feature ID to dispatch for")),
		mcp.WithString("role", mcp.Description("Role profile ID for context assembly")),
		mcp.WithString("instructions", mcp.Description("Additional orchestrator instructions")),
	)

	handler := WithSideEffects(func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		return DispatchAction(ctx, req, map[string]ActionHandler{
			"dispatch": developDispatchAction(entitySvc, conflictSvc),
		})
	})

	return []server.ServerTool{{Tool: tool, Handler: handler}}
}

// developDispatchAction implements develop(action: "dispatch", ...).
func developDispatchAction(entitySvc *service.EntityService, conflictSvc *service.ConflictService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		SignalMutation(ctx)
		args, _ := req.Params.Arguments.(map[string]any)

		featureID := docArgStr(args, "feature_id")
		if featureID == "" {
			return nil, fmt.Errorf("Cannot dispatch: feature_id is missing.\n\nTo resolve:\n  Provide feature_id: develop(action: \"dispatch\", feature_id: \"FEAT-...\")")
		}

		// Verify feature exists and is in developing status.
		feat, err := entitySvc.Get("feature", featureID, "")
		if err != nil {
			return nil, fmt.Errorf("Cannot dispatch: feature %s not found: %w", featureID, err)
		}
		featStatus, _ := feat.State["status"].(string)
		if featStatus != "developing" {
			return map[string]any{
				"error":       fmt.Sprintf("Feature %s is in %q, not developing", featureID, featStatus),
				"empty_queue": true,
			}, nil
		}

		// List tasks for this feature by filtering all tasks.
		allTasks, listErr := entitySvc.List("task")
		if listErr != nil {
			return nil, fmt.Errorf("Cannot dispatch: %w", listErr)
		}
		tasks := make([]service.ListResult, 0)
		for _, t := range allTasks {
			pf, _ := t.State["parent_feature"].(string)
			if pf == featureID {
				tasks = append(tasks, t)
			}
		}

		// Separate tasks by status.
		type taskInfo struct {
			ID      string
			Slug    string
			Summary string
		}
		var readyTasks []taskInfo
		var activeTasks []taskInfo
		var blockedTasks []taskInfo
		var doneTasks []taskInfo

		for _, t := range tasks {
			status, _ := t.State["status"].(string)
			summary, _ := t.State["summary"].(string)
			info := taskInfo{ID: t.ID, Slug: t.Slug, Summary: summary}

			switch status {
			case "ready":
				readyTasks = append(readyTasks, info)
			case "active":
				activeTasks = append(activeTasks, info)
			case "queued":
				blockedTasks = append(blockedTasks, info)
			case "done", "not-planned", "duplicate":
				doneTasks = append(doneTasks, info)
			default:
				blockedTasks = append(blockedTasks, info)
			}
		}

		if len(readyTasks) == 0 {
			// Build blocked info for each non-ready task.
			blockedInfo := make([]map[string]any, 0)
			for _, t := range blockedTasks {
				blockedInfo = append(blockedInfo, map[string]any{
					"task_id": t.ID,
					"slug":    t.Slug,
					"summary": t.Summary,
					"reason":  "queued — dependencies not yet satisfied",
				})
			}
			return map[string]any{
				"dispatched":  []any{},
				"conflicting": []any{},
				"blocked":     blockedInfo,
				"empty_queue": true,
			}, nil
		}

		// Transition all ready tasks to active.
		dispatched := make([]map[string]any, 0)
		for _, t := range readyTasks {
			_, err := entitySvc.UpdateStatus(service.UpdateStatusInput{
				Type:   "task",
				ID:     t.ID,
				Status: "active",
			})
			if err != nil {
				// Task may have been claimed by another dispatcher — skip.
				continue
			}
			dispatched = append(dispatched, map[string]any{
				"task_id": t.ID,
				"slug":    t.Slug,
				"summary": t.Summary,
				"handoff_hint": map[string]any{
					"tool":   "handoff",
					"action": "",
					"params": map[string]any{"task_id": t.ID},
				},
			})
		}

		// Build blocked info.
		blockedInfo := make([]map[string]any, 0)
		for _, t := range blockedTasks {
			blockedInfo = append(blockedInfo, map[string]any{
				"task_id": t.ID,
				"slug":    t.Slug,
				"summary": t.Summary,
				"reason":  "queued — dependencies not yet satisfied",
			})
		}

		return map[string]any{
			"dispatched":   dispatched,
			"conflicting":  []any{},
			"blocked":      blockedInfo,
			"empty_queue":  false,
			"tasks_ready":  len(readyTasks),
			"tasks_active": len(activeTasks),
			"tasks_done":   len(doneTasks),
		}, nil
	}
}
