package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"kanbanzai/internal/service"
)

// EstimationTools returns all estimation MCP tools.
func EstimationTools(entitySvc *service.EntityService, knowledgeSvc *service.KnowledgeService) []server.ServerTool {
	return []server.ServerTool{
		estimateSetTool(entitySvc, knowledgeSvc),
		estimateQueryTool(entitySvc),
		estimateReferenceAddTool(knowledgeSvc),
		estimateReferenceRemoveTool(knowledgeSvc),
	}
}

// resolveEntityType tries each entity type in order and returns the first match.
func resolveEntityType(entitySvc *service.EntityService, entityID string) (string, service.GetResult, error) {
	for _, etype := range []string{"task", "feature", "epic", "bug", "plan"} {
		result, err := entitySvc.Get(etype, entityID, "")
		if err == nil {
			return etype, result, nil
		}
	}
	return "", service.GetResult{}, fmt.Errorf("entity not found: %s", entityID)
}

func estimateSetTool(entitySvc *service.EntityService, knowledgeSvc *service.KnowledgeService) server.ServerTool {
	tool := mcp.NewTool("estimate_set",
		mcp.WithDescription("Set a story point estimate on a task, feature, epic, bug, or plan. Uses the Modified Fibonacci scale: 0, 0.5, 1, 2, 3, 5, 8, 13, 20, 40, 100. Entity type is auto-detected from the ID."),
		mcp.WithString("entity_id", mcp.Description("Entity ID (e.g. T-01ABCDEFGHIJK, FEAT-01ABCDEFGHIJK, BUG-01ABCDEFGHIJK)"), mcp.Required()),
		mcp.WithNumber("estimate", mcp.Description("Story point estimate from the Modified Fibonacci scale: 0, 0.5, 1, 2, 3, 5, 8, 13, 20, 40, 100"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		entityID, err := request.RequireString("entity_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		estimate := request.GetFloat("estimate", 0)

		// Auto-detect entity type
		entityType, _, err := resolveEntityType(entitySvc, entityID)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("entity not found", err), nil
		}

		fields, warning, err := entitySvc.SetEstimate(entityType, entityID, estimate)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("set estimate failed", err), nil
		}

		// Get estimation references for context
		refs, err := knowledgeSvc.GetEstimationReferences()
		if err != nil {
			refs = nil // best-effort
		}
		refsJSON := make([]map[string]any, 0, len(refs))
		for _, r := range refs {
			refsJSON = append(refsJSON, r.Fields)
		}

		// Build scale
		scale := service.GetScaleEntries()
		scaleJSON := make([]map[string]any, len(scale))
		for i, e := range scale {
			scaleJSON[i] = map[string]any{
				"points":  e.Points,
				"meaning": e.Meaning,
			}
		}

		var warningVal any
		if warning != "" {
			warningVal = warning
		}

		resp := map[string]any{
			"entity_id":          fields["id"],
			"entity_type":        entityType,
			"estimate":           estimate,
			"soft_limit_warning": warningVal,
			"references":         refsJSON,
			"scale":              scaleJSON,
		}
		return estimationMapJSON(resp)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func estimateQueryTool(entitySvc *service.EntityService) server.ServerTool {
	tool := mcp.NewTool("estimate_query",
		mcp.WithDescription("Query the current estimate and rollup statistics for an entity. For features, includes a task-level rollup. For epics/plans, includes a feature-level rollup."),
		mcp.WithString("entity_id", mcp.Description("Entity ID to query"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		entityID, err := request.RequireString("entity_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Auto-detect entity type
		entityType, result, err := resolveEntityType(entitySvc, entityID)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("entity not found", err), nil
		}

		ownEstimate := service.GetEstimateFromFields(result.State)

		var estimateVal any
		if ownEstimate != nil {
			estimateVal = *ownEstimate
		}

		resp := map[string]any{
			"entity_id":   result.ID,
			"entity_type": entityType,
			"estimate":    estimateVal,
			"rollup":      nil,
		}

		switch entityType {
		case "feature":
			rollup, err := entitySvc.ComputeFeatureRollup(result.ID)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("compute feature rollup failed", err), nil
			}

			// Compute delta: TaskTotal - own estimate
			var delta any
			if rollup.TaskTotal != nil && ownEstimate != nil {
				d := *rollup.TaskTotal - *ownEstimate
				delta = d
			}

			var taskTotalVal any
			if rollup.TaskTotal != nil {
				taskTotalVal = *rollup.TaskTotal
			}

			resp["rollup"] = map[string]any{
				"task_total":           taskTotalVal,
				"progress":             rollup.Progress,
				"delta":                delta,
				"task_count":           rollup.TaskCount,
				"estimated_task_count": rollup.EstimatedTaskCount,
				"excluded_task_count":  rollup.ExcludedTaskCount,
			}

		case "epic", "plan":
			rollup, err := entitySvc.ComputeEpicRollup(result.ID)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("compute epic rollup failed", err), nil
			}

			// Compute delta: FeatureTotal - own estimate
			var delta any
			if rollup.FeatureTotal != nil && ownEstimate != nil {
				d := *rollup.FeatureTotal - *ownEstimate
				delta = d
			}

			var featureTotalVal any
			if rollup.FeatureTotal != nil {
				featureTotalVal = *rollup.FeatureTotal
			}

			resp["rollup"] = map[string]any{
				"feature_total":           featureTotalVal,
				"progress":                rollup.Progress,
				"delta":                   delta,
				"feature_count":           rollup.FeatureCount,
				"estimated_feature_count": rollup.EstimatedFeatureCount,
			}
		}

		return estimationMapJSON(resp)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func estimateReferenceAddTool(knowledgeSvc *service.KnowledgeService) server.ServerTool {
	tool := mcp.NewTool("estimate_reference_add",
		mcp.WithDescription("Add a calibration reference example for an entity to help with future estimation. References are stored as project-scoped knowledge entries tagged 'estimation-reference' with TTL exempt (ttl_days=0)."),
		mcp.WithString("entity_id", mcp.Description("Entity ID this reference anchors to (e.g. T-01ABCDEFGHIJK)"), mcp.Required()),
		mcp.WithString("content", mcp.Description("Description of the work and its actual complexity, to serve as an estimation calibration example"), mcp.Required()),
		mcp.WithString("created_by", mcp.Description("Identity of the contributor")),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		entityID, err := request.RequireString("entity_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		content, err := request.RequireString("content")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		createdBy := request.GetString("created_by", "")

		record, err := knowledgeSvc.AddEstimationReference(entityID, content, createdBy)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("add estimation reference failed", err), nil
		}

		topic, _ := record.Fields["topic"].(string)

		resp := map[string]any{
			"entry_id":  record.ID,
			"entity_id": entityID,
			"topic":     topic,
			"status":    "added",
		}
		return estimationMapJSON(resp)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func estimateReferenceRemoveTool(knowledgeSvc *service.KnowledgeService) server.ServerTool {
	tool := mcp.NewTool("estimate_reference_remove",
		mcp.WithDescription("Remove (retire) the estimation calibration reference for an entity."),
		mcp.WithString("entity_id", mcp.Description("Entity ID whose estimation reference should be removed"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		entityID, err := request.RequireString("entity_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		entryID, err := knowledgeSvc.RemoveEstimationReference(entityID)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("remove estimation reference failed", err), nil
		}

		resp := map[string]any{
			"entity_id": entityID,
			"entry_id":  entryID,
			"status":    "removed",
		}
		return estimationMapJSON(resp)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

// estimationMapJSON marshals a map to JSON and returns it as a tool result.
func estimationMapJSON(v map[string]any) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("marshal result: " + err.Error()), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}
