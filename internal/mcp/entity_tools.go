package mcp

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"kanbanzai/internal/service"
)

// EntityTools returns all entity-related MCP tool definitions with their handlers.
func EntityTools(svc *service.EntityService) []server.ServerTool {
	return []server.ServerTool{
		createEpicTool(svc),
		createFeatureTool(svc),
		createTaskTool(svc),
		createBugTool(svc),
		recordDecisionTool(svc),
		getEntityTool(svc),
		listEntitiesTool(svc),
		updateStatusTool(svc),
		updateEntityTool(svc),
		validateCandidateTool(svc),
		healthCheckTool(svc),
		rebuildCacheTool(svc),
	}
}

func createEpicTool(svc *service.EntityService) server.ServerTool {
	tool := mcp.NewTool("create_epic",
		mcp.WithDescription("Create a new epic entity"),
		mcp.WithString("slug", mcp.Description("URL-friendly identifier for the epic"), mcp.Required()),
		mcp.WithString("title", mcp.Description("Title of the epic"), mcp.Required()),
		mcp.WithString("summary", mcp.Description("Brief summary of the epic"), mcp.Required()),
		mcp.WithString("created_by", mcp.Description("Who created the epic"), mcp.Required()),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
		result, err := svc.CreateEpic(service.CreateEpicInput{
			Slug:      slug,
			Title:     title,
			Summary:   summary,
			CreatedBy: createdBy,
		})
		if err != nil {
			return mcp.NewToolResultErrorFromErr("create epic failed", err), nil
		}
		return jsonResult(result)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func createFeatureTool(svc *service.EntityService) server.ServerTool {
	tool := mcp.NewTool("create_feature",
		mcp.WithDescription("Create a new feature entity"),
		mcp.WithString("slug", mcp.Description("URL-friendly identifier for the feature"), mcp.Required()),
		mcp.WithString("epic", mcp.Description("Parent epic ID"), mcp.Required()),
		mcp.WithString("summary", mcp.Description("Brief summary of the feature"), mcp.Required()),
		mcp.WithString("created_by", mcp.Description("Who created the feature"), mcp.Required()),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slug, err := request.RequireString("slug")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		epic, err := request.RequireString("epic")
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
		result, err := svc.CreateFeature(service.CreateFeatureInput{
			Slug:      slug,
			Epic:      epic,
			Summary:   summary,
			CreatedBy: createdBy,
		})
		if err != nil {
			return mcp.NewToolResultErrorFromErr("create feature failed", err), nil
		}
		return jsonResult(result)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func createTaskTool(svc *service.EntityService) server.ServerTool {
	tool := mcp.NewTool("create_task",
		mcp.WithDescription("Create a new task entity"),
		mcp.WithString("parent_feature", mcp.Description("Parent feature ID"), mcp.Required()),
		mcp.WithString("slug", mcp.Description("URL-friendly identifier for the task"), mcp.Required()),
		mcp.WithString("summary", mcp.Description("Brief summary of the task"), mcp.Required()),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		parentFeature, err := request.RequireString("parent_feature")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		slug, err := request.RequireString("slug")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		summary, err := request.RequireString("summary")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		result, err := svc.CreateTask(service.CreateTaskInput{
			ParentFeature: parentFeature,
			Slug:          slug,
			Summary:       summary,
		})
		if err != nil {
			return mcp.NewToolResultErrorFromErr("create task failed", err), nil
		}
		return jsonResult(result)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func createBugTool(svc *service.EntityService) server.ServerTool {
	tool := mcp.NewTool("create_bug",
		mcp.WithDescription("Create a new bug entity"),
		mcp.WithString("slug", mcp.Description("URL-friendly identifier for the bug"), mcp.Required()),
		mcp.WithString("title", mcp.Description("Title of the bug"), mcp.Required()),
		mcp.WithString("reported_by", mcp.Description("Who reported the bug"), mcp.Required()),
		mcp.WithString("observed", mcp.Description("Observed behavior"), mcp.Required()),
		mcp.WithString("expected", mcp.Description("Expected behavior"), mcp.Required()),
		mcp.WithString("severity",
			mcp.Description("Bug severity level"),
			mcp.Required(),
			mcp.Enum("low", "medium", "high", "critical"),
		),
		mcp.WithString("priority",
			mcp.Description("Bug priority level"),
			mcp.Required(),
			mcp.Enum("low", "medium", "high", "critical"),
		),
		mcp.WithString("type",
			mcp.Description("Bug type classification"),
			mcp.Required(),
			mcp.Enum("implementation-defect", "specification-defect", "design-problem"),
		),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slug, err := request.RequireString("slug")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		title, err := request.RequireString("title")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		reportedBy, err := request.RequireString("reported_by")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		observed, err := request.RequireString("observed")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		expected, err := request.RequireString("expected")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		severity, err := request.RequireString("severity")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		priority, err := request.RequireString("priority")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		bugType, err := request.RequireString("type")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		result, err := svc.CreateBug(service.CreateBugInput{
			Slug:       slug,
			Title:      title,
			ReportedBy: reportedBy,
			Observed:   observed,
			Expected:   expected,
			Severity:   severity,
			Priority:   priority,
			Type:       bugType,
		})
		if err != nil {
			return mcp.NewToolResultErrorFromErr("create bug failed", err), nil
		}
		return jsonResult(result)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func recordDecisionTool(svc *service.EntityService) server.ServerTool {
	tool := mcp.NewTool("record_decision",
		mcp.WithDescription("Record a new decision entity"),
		mcp.WithString("slug", mcp.Description("URL-friendly identifier for the decision"), mcp.Required()),
		mcp.WithString("summary", mcp.Description("Brief summary of the decision"), mcp.Required()),
		mcp.WithString("rationale", mcp.Description("Rationale behind the decision"), mcp.Required()),
		mcp.WithString("decided_by", mcp.Description("Who made the decision"), mcp.Required()),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slug, err := request.RequireString("slug")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		summary, err := request.RequireString("summary")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		rationale, err := request.RequireString("rationale")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		decidedBy, err := request.RequireString("decided_by")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		result, err := svc.CreateDecision(service.CreateDecisionInput{
			Slug:      slug,
			Summary:   summary,
			Rationale: rationale,
			DecidedBy: decidedBy,
		})
		if err != nil {
			return mcp.NewToolResultErrorFromErr("record decision failed", err), nil
		}
		return jsonResult(result)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func getEntityTool(svc *service.EntityService) server.ServerTool {
	tool := mcp.NewTool("get_entity",
		mcp.WithDescription("Get a specific entity by type, ID, and slug"),
		mcp.WithString("entity_type",
			mcp.Description("Type of entity to retrieve"),
			mcp.Required(),
			mcp.Enum("epic", "feature", "task", "bug", "decision"),
		),
		mcp.WithString("id", mcp.Description("Entity ID or unambiguous prefix"), mcp.Required()),
		mcp.WithString("slug", mcp.Description("Entity slug (optional, resolved from ID prefix if omitted)")),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		entityType, err := request.RequireString("entity_type")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		id, err := request.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		slug := request.GetString("slug", "")
		result, err := svc.Get(entityType, id, slug)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("get entity failed", err), nil
		}
		return jsonResult(result)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func listEntitiesTool(svc *service.EntityService) server.ServerTool {
	tool := mcp.NewTool("list_entities",
		mcp.WithDescription("List all entities of a given type"),
		mcp.WithString("entity_type",
			mcp.Description("Type of entities to list"),
			mcp.Required(),
			mcp.Enum("epic", "feature", "task", "bug", "decision"),
		),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		entityType, err := request.RequireString("entity_type")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		results, err := svc.List(entityType)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("list entities failed", err), nil
		}
		return jsonResult(results)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func updateStatusTool(svc *service.EntityService) server.ServerTool {
	tool := mcp.NewTool("update_status",
		mcp.WithDescription("Update the lifecycle status of an entity"),
		mcp.WithString("entity_type",
			mcp.Description("Type of entity to update"),
			mcp.Required(),
			mcp.Enum("epic", "feature", "task", "bug", "decision"),
		),
		mcp.WithString("id", mcp.Description("Entity ID or unambiguous prefix"), mcp.Required()),
		mcp.WithString("slug", mcp.Description("Entity slug (optional, resolved from ID prefix if omitted)")),
		mcp.WithString("status", mcp.Description("New lifecycle status"), mcp.Required()),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		entityType, err := request.RequireString("entity_type")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		id, err := request.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		slug := request.GetString("slug", "")
		status, err := request.RequireString("status")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		result, err := svc.UpdateStatus(service.UpdateStatusInput{
			Type:   entityType,
			ID:     id,
			Slug:   slug,
			Status: status,
		})
		if err != nil {
			return mcp.NewToolResultErrorFromErr("update status failed", err), nil
		}
		return jsonResult(result)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func updateEntityTool(svc *service.EntityService) server.ServerTool {
	tool := mcp.NewTool("update_entity",
		mcp.WithDescription("Update fields of an existing entity. Cannot change id or status."),
		mcp.WithString("entity_type",
			mcp.Description("Type of entity to update"),
			mcp.Required(),
			mcp.Enum("epic", "feature", "task", "bug", "decision"),
		),
		mcp.WithString("id", mcp.Description("Entity ID or unambiguous prefix"), mcp.Required()),
		mcp.WithString("slug", mcp.Description("Entity slug (optional, resolved from ID prefix if omitted)")),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		entityType, err := request.RequireString("entity_type")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		id, err := request.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		slug := request.GetString("slug", "")
		args := request.GetArguments()
		fields := make(map[string]string, len(args))
		for k, v := range args {
			if k == "entity_type" || k == "id" || k == "slug" {
				continue
			}
			if s, ok := v.(string); ok {
				fields[k] = s
			}
		}
		result, err := svc.UpdateEntity(service.UpdateEntityInput{
			Type:   entityType,
			ID:     id,
			Slug:   slug,
			Fields: fields,
		})
		if err != nil {
			return mcp.NewToolResultErrorFromErr("update entity failed", err), nil
		}
		return jsonResult(result)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func validateCandidateTool(svc *service.EntityService) server.ServerTool {
	tool := mcp.NewTool("validate_candidate",
		mcp.WithDescription("Validate candidate entity data without persisting it"),
		mcp.WithString("entity_type",
			mcp.Description("Type of entity to validate"),
			mcp.Required(),
			mcp.Enum("epic", "feature", "task", "bug", "decision"),
		),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		entityType, err := request.RequireString("entity_type")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		args := request.GetArguments()
		fields := make(map[string]any, len(args))
		for k, v := range args {
			if k == "entity_type" {
				continue
			}
			fields[k] = v
		}
		errs := svc.ValidateCandidate(entityType, fields)
		return jsonResult(errs)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func healthCheckTool(svc *service.EntityService) server.ServerTool {
	tool := mcp.NewTool("health_check",
		mcp.WithDescription("Run a comprehensive health check across all entities"),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		report, err := svc.HealthCheck()
		if err != nil {
			return mcp.NewToolResultErrorFromErr("health check failed", err), nil
		}
		return jsonResult(report)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func rebuildCacheTool(svc *service.EntityService) server.ServerTool {
	tool := mcp.NewTool("rebuild_cache",
		mcp.WithDescription("Rebuild the local derived cache from canonical entity files. The cache accelerates queries but is not required for correctness."),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		count, err := svc.RebuildCache()
		if err != nil {
			return mcp.NewToolResultErrorFromErr("rebuild cache failed", err), nil
		}
		return jsonResult(map[string]any{
			"status":          "ok",
			"entities_cached": count,
		})
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func jsonResult(v any) (*mcp.CallToolResult, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("failed to marshal result", err), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}
