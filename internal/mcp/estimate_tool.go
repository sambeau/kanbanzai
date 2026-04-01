package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/sambeau/kanbanzai/internal/config"
	"github.com/sambeau/kanbanzai/internal/service"
)

// resolveEntityType infers the entity type from an ID prefix and loads the entity.
// Returns the entity type string, the loaded entity result, and any error.
func resolveEntityType(entitySvc *service.EntityService, entityID string) (string, service.GetResult, error) {
	entityType, ok := entityInferType(entityID)
	if !ok {
		return "", service.GetResult{}, fmt.Errorf("Cannot resolve entity %q: unrecognised ID prefix.\n\nTo resolve:\n  Provide an ID with a valid prefix (e.g. FEAT-xxx, TASK-xxx, BUG-xxx)", entityID)
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
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithTitleAnnotation("Story Point Estimates"),
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

		// Batch mode: entities array provided.
		if IsBatchInput(args, "entities") {
			items, _ := args["entities"].([]any)
			return ExecuteBatch(ctx, items, func(ctx context.Context, item any) (string, any, error) {
				m, ok := item.(map[string]any)
				if !ok {
					return "", nil, fmt.Errorf("Cannot set estimate in batch: each item must be an object with entity_id and points.\n\nTo resolve:\n  Provide entities as [{\"entity_id\": \"...\", \"points\": N}, ...]")
				}
				entityID, _ := m["entity_id"].(string)
				if entityID == "" {
					return "", nil, fmt.Errorf("Cannot set estimate in batch: entity_id is missing from item.\n\nTo resolve:\n  Include entity_id in each batch item: {\"entity_id\": \"TASK-xxx\", \"points\": N}")
				}
				points, err := estimateParsePoints(m["points"])
				if err != nil {
					return entityID, nil, err
				}
				result, err := estimateSetOne(entitySvc, knowledgeSvc, entityID, points)
				return entityID, result, err
			})
		}

		// Single mode: entity_id + points are required.
		entityID, err := req.RequireString("entity_id")
		if err != nil {
			return nil, fmt.Errorf("Cannot set estimate: entity_id is missing.\n\nTo resolve:\n  Provide entity_id: estimate(action: \"set\", entity_id: \"...\", points: N)")
		}
		if _, ok := args["points"]; !ok {
			return nil, fmt.Errorf("Cannot set estimate: points is missing.\n\nTo resolve:\n  Provide points from the Modified Fibonacci scale: estimate(action: \"set\", entity_id: \"...\", points: N)")
		}
		points := req.GetFloat("points", 0)

		return estimateSetOne(entitySvc, knowledgeSvc, entityID, points)
	}
}

// estimateParsePoints extracts a float64 point value from a raw interface value.
// Accepts float64, int, and int64 (as produced by JSON unmarshalling and MCP).
func estimateParsePoints(v any) (float64, error) {
	switch p := v.(type) {
	case float64:
		return p, nil
	case int:
		return float64(p), nil
	case int64:
		return float64(p), nil
	default:
		return 0, fmt.Errorf("Cannot set estimate: points value is not a number.\n\nTo resolve:\n  Provide a numeric value from the Modified Fibonacci scale: 0, 0.5, 1, 2, 3, 5, 8, 13, 20, 40, 100")
	}
}

func estimateSetOne(
	entitySvc *service.EntityService,
	knowledgeSvc *service.KnowledgeService,
	entityID string,
	points float64,
) (map[string]any, error) {
	entityType, _, err := resolveEntityType(entitySvc, entityID)
	if err != nil {
		return nil, fmt.Errorf("Cannot set estimate for %s: entity not found.\n\nTo resolve:\n  Verify the entity ID exists with entity(action: \"get\", id: \"%s\")", entityID, entityID)
	}

	fields, warning, err := entitySvc.SetEstimate(entityType, entityID, points)
	if err != nil {
		return nil, fmt.Errorf("Cannot set estimate for %s: %w.\n\nTo resolve:\n  Check that the points value is on the Modified Fibonacci scale: 0, 0.5, 1, 2, 3, 5, 8, 13, 20, 40, 100", entityID, err)
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
			return nil, fmt.Errorf("Cannot query estimate: entity_id is missing.\n\nTo resolve:\n  Provide entity_id: estimate(action: \"query\", entity_id: \"...\")")
		}

		entityType, result, err := resolveEntityType(entitySvc, entityID)
		if err != nil {
			return nil, fmt.Errorf("Cannot query estimate for %s: entity not found.\n\nTo resolve:\n  Verify the entity ID exists with entity(action: \"get\", id: \"%s\")", entityID, entityID)
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
				return nil, fmt.Errorf("Cannot query estimate for feature %s: rollup computation failed: %w.\n\nTo resolve:\n  Check that the feature's child tasks are in a valid state", result.ID, err)
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
				return nil, fmt.Errorf("Cannot query estimate for %s %s: rollup computation failed: %w.\n\nTo resolve:\n  Check that the child features are in a valid state", entityType, result.ID, err)
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
			return nil, fmt.Errorf("Cannot add estimation reference: entity_id is missing.\n\nTo resolve:\n  Provide entity_id: estimate(action: \"add_reference\", entity_id: \"...\", content: \"...\")")
		}
		content, err := req.RequireString("content")
		if err != nil {
			return nil, fmt.Errorf("Cannot add estimation reference for %s: content is missing.\n\nTo resolve:\n  Provide content describing the work and its complexity: estimate(action: \"add_reference\", entity_id: \"%s\", content: \"...\")", entityID, entityID)
		}

		createdByRaw := req.GetString("created_by", "")
		createdBy, err := config.ResolveIdentity(createdByRaw)
		if err != nil {
			return nil, err
		}

		record, err := knowledgeSvc.AddEstimationReference(entityID, content, createdBy)
		if err != nil {
			return nil, fmt.Errorf("Cannot add estimation reference for %s: %w.\n\nTo resolve:\n  Verify the entity exists and the content is non-empty", entityID, err)
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
			return nil, fmt.Errorf("Cannot remove estimation reference: entity_id is missing.\n\nTo resolve:\n  Provide entity_id: estimate(action: \"remove_reference\", entity_id: \"...\")")
		}

		entryID, err := knowledgeSvc.RemoveEstimationReference(entityID)
		if err != nil {
			return nil, fmt.Errorf("Cannot remove estimation reference for %s: %w.\n\nTo resolve:\n  Verify a calibration reference exists for this entity with estimate(action: \"query\", entity_id: \"%s\")", entityID, err, entityID)
		}

		return map[string]any{
			"entity_id": entityID,
			"entry_id":  entryID,
			"status":    "removed",
		}, nil
	}
}
