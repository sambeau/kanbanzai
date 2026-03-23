package mcp

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"kanbanzai/internal/id"
	"kanbanzai/internal/service"
)

// PlanTools returns all Plan-related MCP tool definitions with their handlers.
func PlanTools(svc *service.EntityService) []server.ServerTool {
	return []server.ServerTool{
		createPlanTool(svc),
		getPlanTool(svc),
		listPlansTool(svc),
		updatePlanStatusTool(svc),
		updatePlanTool(svc),
	}
}

func createPlanTool(svc *service.EntityService) server.ServerTool {
	tool := mcp.NewTool("create_plan",
		mcp.WithDescription("Create a new Plan entity. Plans coordinate bodies of work and organise Features. The prefix must be declared in .kbz/config.yaml."),
		mcp.WithString("prefix", mcp.Description("Single-character prefix for the Plan ID (must be declared in prefix registry)"), mcp.Required()),
		mcp.WithString("slug", mcp.Description("URL-friendly identifier for the Plan (appended after prefix and number)"), mcp.Required()),
		mcp.WithString("title", mcp.Description("Human-readable title of the Plan"), mcp.Required()),
		mcp.WithString("summary", mcp.Description("Brief description of the Plan's purpose and scope"), mcp.Required()),
		mcp.WithString("created_by", mcp.Description("Who created the Plan"), mcp.Required()),
		mcp.WithArray("tags", mcp.Description("Optional freeform tags for organisation (e.g., phase:2, priority:high)")),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		prefix, err := request.RequireString("prefix")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		slug, err := request.RequireString("slug")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		title, err := request.RequireString("title")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		summary, err := request.RequireString("summary")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		createdBy, err := request.RequireString("created_by")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		tags := getStringArray(request, "tags")

		result, err := svc.CreatePlan(service.CreatePlanInput{
			Prefix:    prefix,
			Slug:      slug,
			Title:     title,
			Summary:   summary,
			CreatedBy: createdBy,
			Tags:      tags,
		})
		if err != nil {
			return mcp.NewToolResultErrorFromErr("create plan failed", err), nil
		}
		return jsonResult(createResultWithDisplay(result))
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func getPlanTool(svc *service.EntityService) server.ServerTool {
	tool := mcp.NewTool("get_plan",
		mcp.WithDescription("Get a Plan by its ID. Plan IDs have the format {prefix}{number}-{slug}, e.g., P1-basic-ui."),
		mcp.WithString("id", mcp.Description("Plan ID (e.g., P1-basic-ui)"), mcp.Required()),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		result, err := svc.GetPlan(id)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("get plan failed", err), nil
		}
		return jsonResult(listResultWithDisplay(result))
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func listPlansTool(svc *service.EntityService) server.ServerTool {
	tool := mcp.NewTool("list_plans",
		mcp.WithDescription("List all Plans with optional filtering by status, prefix, or tags."),
		mcp.WithString("status", mcp.Description("Filter by status (proposed, designing, active, done, superseded, cancelled)")),
		mcp.WithString("prefix", mcp.Description("Filter by Plan prefix (single character)")),
		mcp.WithArray("tags", mcp.Description("Filter by tags (Plans must have all specified tags)")),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var filters service.PlanFilters

		filters.Status = request.GetString("status", "")
		filters.Prefix = request.GetString("prefix", "")
		filters.Tags = getStringArray(request, "tags")

		results, err := svc.ListPlans(filters)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("list plans failed", err), nil
		}
		return jsonResult(listResultsWithDisplay(results))
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func updatePlanStatusTool(svc *service.EntityService) server.ServerTool {
	tool := mcp.NewTool("update_plan_status",
		mcp.WithDescription("Transition a Plan to a new lifecycle status. Valid transitions: proposed→designing, designing→active, active→done. Any non-terminal state can transition to superseded or cancelled."),
		mcp.WithString("id", mcp.Description("Plan ID"), mcp.Required()),
		mcp.WithString("slug", mcp.Description("Plan slug"), mcp.Required()),
		mcp.WithString("status", mcp.Description("New status (proposed, designing, active, done, superseded, cancelled)"), mcp.Required()),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		slug, err := request.RequireString("slug")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		status, err := request.RequireString("status")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		result, err := svc.UpdatePlanStatus(id, slug, status)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("update plan status failed", err), nil
		}
		return jsonResult(listResultWithDisplay(result))
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func updatePlanTool(svc *service.EntityService) server.ServerTool {
	tool := mcp.NewTool("update_plan",
		mcp.WithDescription("Update mutable fields on a Plan (title, summary, design reference, tags)."),
		mcp.WithString("id", mcp.Description("Plan ID"), mcp.Required()),
		mcp.WithString("slug", mcp.Description("Plan slug"), mcp.Required()),
		mcp.WithString("title", mcp.Description("New title (optional)")),
		mcp.WithString("summary", mcp.Description("New summary (optional)")),
		mcp.WithString("design", mcp.Description("Reference to design document record (optional, empty string to clear)")),
		mcp.WithArray("tags", mcp.Description("New tags (replaces existing tags)")),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		slug, err := request.RequireString("slug")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		input := service.UpdatePlanInput{
			ID:   id,
			Slug: slug,
		}

		if hasArgument(request, "title") {
			title := request.GetString("title", "")
			input.Title = &title
		}
		if hasArgument(request, "summary") {
			summary := request.GetString("summary", "")
			input.Summary = &summary
		}
		if hasArgument(request, "design") {
			design := request.GetString("design", "")
			input.Design = &design
		}
		if hasArgument(request, "tags") {
			input.Tags = getStringArray(request, "tags")
		}

		result, err := svc.UpdatePlan(input)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("update plan failed", err), nil
		}
		return jsonResult(listResultWithDisplay(result))
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

// listResultWithDisplay converts a ListResult to a display map.
func listResultWithDisplay(r service.ListResult) map[string]any {
	return map[string]any{
		"Type":      r.Type,
		"ID":        r.ID,
		"DisplayID": id.FormatFullDisplay(r.ID),
		"Slug":      r.Slug,
		"Path":      r.Path,
		"State":     r.State,
	}
}

func planListResultsWithDisplay(results []service.ListResult) map[string]any {
	out := make([]map[string]any, len(results))
	for i, r := range results {
		out[i] = listResultWithDisplay(r)
	}
	return map[string]any{
		"success": true,
		"count":   len(results),
		"plans":   out,
	}
}

// Helper to create JSON result from any value
func jsonResultAny(v any) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("marshal result: " + err.Error()), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

// getStringArray extracts a string array from request arguments.
func getStringArray(request mcp.CallToolRequest, key string) []string {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return nil
	}
	raw, ok := args[key]
	if !ok {
		return nil
	}
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	var result []string
	for _, item := range arr {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

// hasArgument checks if an argument key exists in the request.
func hasArgument(request mcp.CallToolRequest, key string) bool {
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return false
	}
	_, exists := args[key]
	return exists
}
