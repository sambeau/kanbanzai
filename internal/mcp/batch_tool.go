// Package mcp batch_tool.go — batch snapshot tool introduced by the
// Composite Tools feature (B43).
//
//	batch(action: "snapshot", batch_id: "B43-composite-tools")
//
// The snapshot action enumerates every feature in the batch with its current
// status, whether it is blocked, what gate is blocking it, and a structured
// next_action for each blocked feature.
package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/sambeau/kanbanzai/internal/service"
)

// BatchTool returns the `batch` MCP tool (B43 — Composite Tools).
// Currently exposes a single action: snapshot.
func BatchTool(entitySvc *service.EntityService) []server.ServerTool {
	tool := mcp.NewTool("batch",
		mcp.WithTitleAnnotation("Batch Operations"),
		mcp.WithDescription(
			"Batch-level operations for workflow management. "+
				"Actions: snapshot.",
		),
		mcp.WithString("action", mcp.Required(), mcp.Description("Action: snapshot")),
		mcp.WithString("batch_id", mcp.Description("Batch ID for snapshot action")),
	)

	handler := WithSideEffects(func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		return DispatchAction(ctx, req, map[string]ActionHandler{
			"snapshot": batchSnapshotAction(entitySvc),
		})
	})

	return []server.ServerTool{{Tool: tool, Handler: handler}}
}

// batchSnapshotAction implements batch(action: "snapshot", ...).
func batchSnapshotAction(entitySvc *service.EntityService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		args, _ := req.Params.Arguments.(map[string]any)

		batchID := docArgStr(args, "batch_id")
		if batchID == "" {
			return nil, fmt.Errorf("Cannot snapshot: batch_id is missing.\n\nTo resolve:\n  Provide batch_id: batch(action: \"snapshot\", batch_id: \"B43-composite-tools\")")
		}

		// Retrieve the batch entity.
		batch, err := entitySvc.Get("batch", batchID, "")
		if err != nil {
			return nil, fmt.Errorf("Cannot snapshot: batch %s not found: %w", batchID, err)
		}
		batchStatus, _ := batch.State["status"].(string)
		batchSlug, _ := batch.State["slug"].(string)

		// List features for this batch by filtering all features.
		allFeatures, listErr := entitySvc.List("feature")
		if listErr != nil {
			return nil, fmt.Errorf("Cannot snapshot: %w", listErr)
		}
		features := make([]service.ListResult, 0)
		for _, f := range allFeatures {
			parent, _ := f.State["parent"].(string)
			if parent == batchID {
				features = append(features, f)
			}
		}

		// Build per-feature status summary.
		featureInfos := make([]map[string]any, 0, len(features))
		blockedCount := 0
		readyCount := 0

		for _, f := range features {
			featStatus, _ := f.State["status"].(string)
			featSummary, _ := f.State["summary"].(string)

			info := map[string]any{
				"feature_id": f.ID,
				"slug":       f.Slug,
				"status":     featStatus,
				"summary":    featSummary,
			}

			// Determine if the feature is blocked and what the next action is.
			blocked, blockingGate, missing, nextAct := analyseFeatureBlocked(f.ID, featStatus, f.State)
			info["blocked"] = blocked
			if blocked {
				blockedCount++
				info["blocking_gate"] = blockingGate
				if missing != nil {
					info["missing"] = missing
				}
				if nextAct != nil {
					info["next_action"] = nextAct
				}
			} else if isFeatureStatusReady(featStatus) {
				readyCount++
			}

			featureInfos = append(featureInfos, info)
		}

		summary := fmt.Sprintf("%d of %d features blocked. %d features ready to advance.",
			blockedCount, len(features), readyCount)

		return map[string]any{
			"batch": map[string]any{
				"id":     batchID,
				"slug":   batchSlug,
				"status": batchStatus,
			},
			"features": featureInfos,
			"summary":  summary,
		}, nil
	}
}

// analyseFeatureBlocked determines whether a feature is blocked at its current
// stage and produces a structured next_action if so.
func analyseFeatureBlocked(featureID, status string, state map[string]any) (bool, string, map[string]any, *nextAction) {
	// Stage gate analysis — checks which gate a feature in a given status
	// needs to pass to advance.
	switch status {
	case "designing":
		return true, "designing→specifying",
			map[string]any{"type": "document", "document_type": "design", "status_needed": "approved"},
			ptrNextAction(nextActionForMissingDocument("design", featureID))
	case "specifying":
		return true, "specifying→dev-planning",
			map[string]any{"type": "document", "document_type": "specification", "status_needed": "approved"},
			ptrNextAction(nextActionForMissingDocument("specification", featureID))
	case "dev-planning":
		return true, "dev-planning→developing",
			map[string]any{"type": "document", "document_type": "dev-plan", "status_needed": "approved"},
			ptrNextAction(nextActionForMissingDocument("dev-plan", featureID))
	case "developing":
		// Blocked if tasks are not complete.
		return true, "developing→reviewing",
			map[string]any{"type": "tasks", "status_needed": "all terminal"},
			ptrNextAction(nextActionForNonTerminalTasks(featureID))
	case "reviewing":
		return true, "reviewing→done",
			map[string]any{"type": "document", "document_type": "report", "status_needed": "approved"},
			ptrNextAction(nextActionForMissingDocument("report", featureID))
	case "done", "cancelled", "superseded":
		return false, "", nil, nil
	default:
		return false, "", nil, nil
	}
}

// ptrNextAction returns a pointer to the given nextAction.
func ptrNextAction(na nextAction) *nextAction {
	return &na
}

// isFeatureStatusReady returns true if the feature is in a state where it could
// be advanced (all prerequisites met).
func isFeatureStatusReady(status string) bool {
	return status == "ready" || status == "active"
}
