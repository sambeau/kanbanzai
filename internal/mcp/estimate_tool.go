package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"kanbanzai/internal/config"
	"kanbanzai/internal/service"
)

// resolveEntityType infers the entity type from an ID prefix and loads the entity.
// Returns the entity type string, the loaded entity result, and any error.
func resolveEntityType(entitySvc *service.EntityService, entityID string) (string, service.GetResult, error) {
	entityType, ok := entityInferType(entityID)
	if !ok {
		return "", service.GetResult{}, fmt.Errorf("cannot infer entity type from ID %q", entityID)
	}
	result, err := entitySvc.Get(entityType, entityID, "")
	if err != nil {
		return "", service.GetResult{}, err
	}
	return entityType, result, nil
}

// EstimateTool returns the 2.0 consolidated estimate tool.
// It consolidates estimate_set, estimate_query, estimate_reference_add, and
// estimate_reference_remove into a single tool (spec §17.2).
func EstimateTool(entitySvc *service.EntityService, knowledgeSvc *service.KnowledgeService) []server.ServerTool {
	return []server.ServerTool{estimateTool(entitySvc, knowledgeSvc)}
}

func estimateTool(entitySvc *service.EntityService, knowledgeSvc *service.KnowledgeService) server.ServerTool {
	tool := mcp.NewTool("estimate",
		mcp.WithDescription(
			"Set and query story point estimates on entities. "+
				"Consolidates estimate_set, estimate_query, estimate_reference_add, and estimate_reference_remove. "+
				"Actions: set (single entity_id+points, or batch via entities array), query (rollup stats), "+
				"add_reference (add calibration example), remove_reference (remove calibration example). "+
				"Uses the Modified Fibonacci scale: 0, 0.5, 1, 2, 3, 5, 8, 13, 20, 40, 100.",
		),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("Action: set, query, add_reference, remove_reference"),
		),
		// Shared: entity_id used by set (single), query, add_reference, remove_reference.
		mcp.WithString("entity_id",
			mcp.Description("Entity ID — type auto-detected from prefix"),
		),
		// set — single mode
		mcp.WithNumber("points",
			mcp.Description("Story points from Modified Fibonacci scale: 0, 0.5, 1, 2, 3, 5, 8, 13, 20, 40, 100 (set action, single mode)"),
		),
		// set — batch mode
		mcp.WithArray("entities",
			mcp.Description("Batch set: array of {entity_id, points} objects (set action, batch mode)"),
		),
		// add_reference
		mcp.WithString("content",
			mcp.Description("Description of the work and its actual complexity (add_reference only)"),
		),
		mcp.WithString("created_by",
			mcp.Description("Identity of the contributor. Auto-resolved if omitted (add_reference only)."),
		),
	)

	handler := WithSideEffects(func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		return DispatchAction(ctx, req, map[string]ActionHandler{
			"set":              estimateSetAction(entitySvc, knowledgeSvc),
			"query":            estimateQueryAction(entitySvc),
			"add_reference":    estimateAddReferenceAction(knowledgeSvc),
			"remove_reference": estimateRemoveReferenceAction(knowledgeSvc),
		})
	})

	return server.ServerTool{Tool: tool, Handler: handler}
}

// ─── set ─────────────────────────────────────────────────────────────────────

func estimateSetAction(entitySvc *service.EntityService, knowledgeSvc *service.KnowledgeService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		SignalMutation(ctx)

		args := req.GetArguments()

		// Check for batch mode: entities array takes precedence over single mode.
		if entitiesRaw, ok := args["entities"]; ok && entitiesRaw != nil {
			return estimateSetBatch(entitySvc, knowledgeSvc, entitiesRaw)
		}

		// Single mode: entity_id + points are required.
		entityID, err := req.RequireString("entity_id")
		if err != nil {
			return nil, fmt.Errorf("entity_id is required for set (single mode)")
		}
		if _, ok := args["points"]; !ok {
			return nil, fmt.Errorf("points is required for set (single mode)")
		}
		points := req.GetFloat("points", 0)

		return estimateSetOne(entitySvc, knowledgeSvc, entityID, points)
	}
}

func estimateSetBatch(
	entitySvc *service.EntityService,
	knowledgeSvc *service.KnowledgeService,
	entitiesRaw any,
) (any, error) {
	entitiesSlice, ok := entitiesRaw.([]any)
	if !ok {
		return nil, fmt.Errorf("entities must be an array of {entity_id, points} objects")
	}

	type batchResult struct {
		EntityID string  `json:"entity_id"`
		Points   float64 `json:"points,omitempty"`
		Status   string  `json:"status"`
		Error    string  `json:"error,omitempty"`
	}

	results := make([]batchResult, 0, len(entitiesSlice))
	succeeded := 0
	failed := 0

	for _, rawItem := range entitiesSlice {
		item, ok := rawItem.(map[string]any)
		if !ok {
			failed++
			results = append(results, batchResult{Status: "error", Error: "item must be an object with entity_id and points"})
			continue
		}

		entityID, _ := item["entity_id"].(string)
		if entityID == "" {
			failed++
			results = append(results, batchResult{Status: "error", Error: "entity_id is required"})
			continue
		}

		var points float64
		switch v := item["points"].(type) {
		case float64:
			points = v
		case int:
			points = float64(v)
		case int64:
			points = float64(v)
		default:
			failed++
			results = append(results, batchResult{EntityID: entityID, Status: "error", Error: "points must be a number"})
			continue
		}

		_, err := estimateSetOne(entitySvc, knowledgeSvc, entityID, points)
		if err != nil {
			failed++
			results = append(results, batchResult{EntityID: entityID, Status: "error", Error: err.Error()})
			continue
		}

		succeeded++
		results = append(results, batchResult{EntityID: entityID, Points: points, Status: "set"})
	}

	return map[string]any{
		"results":   results,
		"total":     len(entitiesSlice),
		"succeeded": succeeded,
		"failed":    failed,
	}, nil
}

func estimateSetOne(
	entitySvc *service.EntityService,
	knowledgeSvc *service.KnowledgeService,
	entityID string,
	points float64,
) (map[string]any, error) {
	entityType, _, err := resolveEntityType(entitySvc, entityID)
	if err != nil {
		return nil, fmt.Errorf("entity not found: %s", entityID)
	}

	fields, warning, err := entitySvc.SetEstimate(entityType, entityID, points)
	if err != nil {
		return nil, fmt.Errorf("set estimate: %w", err)
	}

	// Gather calibration references for context (best-effort).
	refs, _ := knowledgeSvc.GetEstimationReferences()
	refsJSON := make([]map[string]any, 0, len(refs))
	for _, r := range refs {
		refsJSON = append(refsJSON, r.Fields)
	}

	// Modified Fibonacci scale labels.
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

	return map[string]any{
		"entity_id":          fields["id"],
		"entity_type":        entityType,
		"estimate":           points,
		"soft_limit_warning": warningVal,
		"references":         refsJSON,
		"scale":              scaleJSON,
	}, nil
}

// ─── query ───────────────────────────────────────────────────────────────────

func estimateQueryAction(entitySvc *service.EntityService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		// Read-only: no SignalMutation.
		entityID, err := req.RequireString("entity_id")
		if err != nil {
			return nil, fmt.Errorf("entity_id is required for query action")
		}

		entityType, result, err := resolveEntityType(entitySvc, entityID)
		if err != nil {
			return nil, fmt.Errorf("entity not found: %s", entityID)
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
				return nil, fmt.Errorf("compute feature rollup: %w", err)
			}

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
				return nil, fmt.Errorf("compute epic rollup: %w", err)
			}

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

		return resp, nil
	}
}

// ─── add_reference ───────────────────────────────────────────────────────────

func estimateAddReferenceAction(knowledgeSvc *service.KnowledgeService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		SignalMutation(ctx)

		entityID, err := req.RequireString("entity_id")
		if err != nil {
			return nil, fmt.Errorf("entity_id is required for add_reference action")
		}
		content, err := req.RequireString("content")
		if err != nil {
			return nil, fmt.Errorf("content is required for add_reference action")
		}

		createdByRaw := req.GetString("created_by", "")
		createdBy, err := config.ResolveIdentity(createdByRaw)
		if err != nil {
			return nil, err
		}

		record, err := knowledgeSvc.AddEstimationReference(entityID, content, createdBy)
		if err != nil {
			return nil, fmt.Errorf("add estimation reference: %w", err)
		}

		topic, _ := record.Fields["topic"].(string)

		return map[string]any{
			"entry_id":  record.ID,
			"entity_id": entityID,
			"topic":     topic,
			"status":    "added",
		}, nil
	}
}

// ─── remove_reference ────────────────────────────────────────────────────────

func estimateRemoveReferenceAction(knowledgeSvc *service.KnowledgeService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		SignalMutation(ctx)

		entityID, err := req.RequireString("entity_id")
		if err != nil {
			return nil, fmt.Errorf("entity_id is required for remove_reference action")
		}

		entryID, err := knowledgeSvc.RemoveEstimationReference(entityID)
		if err != nil {
			return nil, fmt.Errorf("remove estimation reference: %w", err)
		}

		return map[string]any{
			"entity_id": entityID,
			"entry_id":  entryID,
			"status":    "removed",
		}, nil
	}
}
