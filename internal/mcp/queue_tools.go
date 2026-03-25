package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"kanbanzai/internal/service"
)

// QueueTools returns the work_queue and dependency_status MCP tools.
func QueueTools(entitySvc *service.EntityService) []server.ServerTool {
	return []server.ServerTool{
		workQueueTool(entitySvc),
		dependencyStatusTool(entitySvc),
	}
}

func workQueueTool(entitySvc *service.EntityService) server.ServerTool {
	tool := mcp.NewTool("work_queue",
		mcp.WithDescription("Return the current ready task queue, promoting eligible queued tasks first. This is a write-through query: it promotes queued tasks whose dependencies are all in terminal states (done, not-planned, or duplicate) to ready status as a side effect. Returns all ready tasks sorted by estimate (ascending, null last), then age (descending), then task ID."),
		mcp.WithString("role", mcp.Description("Optional: filter results to tasks whose parent feature matches this role profile")),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		role := request.GetString("role", "")

		result, err := entitySvc.WorkQueue(service.WorkQueueInput{Role: role})
		if err != nil {
			return mcp.NewToolResultErrorFromErr("work_queue failed", err), nil
		}

		type queueItem struct {
			TaskID        string   `json:"task_id"`
			Slug          string   `json:"slug"`
			Summary       string   `json:"summary"`
			ParentFeature string   `json:"parent_feature"`
			FeatureSlug   string   `json:"feature_slug,omitempty"`
			Estimate      *float64 `json:"estimate"`
			AgeDays       int      `json:"age_days"`
			Status        string   `json:"status"`
		}

		items := make([]queueItem, len(result.Queue))
		for i, item := range result.Queue {
			items[i] = queueItem{
				TaskID:        item.TaskID,
				Slug:          item.Slug,
				Summary:       item.Summary,
				ParentFeature: item.ParentFeature,
				FeatureSlug:   item.FeatureSlug,
				Estimate:      item.Estimate,
				AgeDays:       item.AgeDays,
				Status:        item.Status,
			}
		}

		resp := map[string]any{
			"queue":          items,
			"promoted_count": result.PromotedCount,
			"total_queued":   result.TotalQueued,
		}

		data, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshal result: %s", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func dependencyStatusTool(entitySvc *service.EntityService) server.ServerTool {
	tool := mcp.NewTool("dependency_status",
		mcp.WithDescription("Show the dependency picture for a given task: each dependency, its current status, and whether it is blocking (not yet terminal) or resolved."),
		mcp.WithString("task_id", mcp.Description("Task ID to check dependencies for"), mcp.Required()),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		taskID, err := request.RequireString("task_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		result, err := entitySvc.GetDependencyStatus(taskID)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("dependency_status failed", err), nil
		}

		type depEntry struct {
			TaskID        string  `json:"task_id"`
			Slug          string  `json:"slug"`
			Status        string  `json:"status"`
			Blocking      bool    `json:"blocking"`
			TerminalState *string `json:"terminal_state"`
		}

		deps := make([]depEntry, len(result.Dependencies))
		for i, d := range result.Dependencies {
			deps[i] = depEntry{
				TaskID:        d.TaskID,
				Slug:          d.Slug,
				Status:        d.Status,
				Blocking:      d.Blocking,
				TerminalState: d.TerminalState,
			}
		}

		resp := map[string]any{
			"task_id":          result.TaskID,
			"slug":             result.Slug,
			"status":           result.Status,
			"depends_on_count": result.DependsOnCount,
			"blocking_count":   result.BlockingCount,
			"dependencies":     deps,
		}

		data, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshal result: %s", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}
