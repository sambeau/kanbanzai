package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"kanbanzai/internal/service"
)

// ReviewTools returns the MCP tools for worker review.
func ReviewTools(svc *service.ReviewService) []server.ServerTool {
	return []server.ServerTool{
		reviewTaskOutputTool(svc),
	}
}

func reviewTaskOutputTool(svc *service.ReviewService) server.ServerTool {
	tool := mcp.NewTool("review_task_output",
		mcp.WithDescription(
			"Run a first-pass review of a completed task's output against its verification criteria "+
				"and parent feature spec. Returns findings with severity (error/warning) and triggers "+
				"state transitions: fail → needs-rework, pass → needs-review. "+
				"Tasks already in needs-review or done are reviewed without state changes.",
		),
		mcp.WithString("task_id",
			mcp.Description("TASK ID of the completed or active task"),
			mcp.Required(),
		),
		mcp.WithObject("output_files",
			mcp.Description("Paths of files produced or modified by this task (array of strings)"),
		),
		mcp.WithString("output_summary",
			mcp.Description("Agent's description of what was done"),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		taskID, err := request.RequireString("task_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		input := service.ReviewInput{
			TaskID:        taskID,
			OutputSummary: request.GetString("output_summary", ""),
		}

		// Parse output_files from the object argument.
		if raw, ok := request.GetArguments()["output_files"]; ok && raw != nil {
			if arr, ok := raw.([]any); ok {
				for _, item := range arr {
					if s, ok := item.(string); ok {
						input.OutputFiles = append(input.OutputFiles, s)
					}
				}
			}
		}

		result, err := svc.ReviewTaskOutput(input)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("review_task_output failed", err), nil
		}

		return jsonResult(result)
	}

	return server.ServerTool{Tool: tool, Handler: handler}
}
