package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"kanbanzai/internal/service"
)

// ConflictTool returns the 2.0 conflict consolidated tool.
// It wraps the existing ConflictService in the action-dispatch pattern.
// The single "check" action passes task_ids through to ConflictService.Check.
func ConflictTool(conflictSvc *service.ConflictService) []server.ServerTool {
	return []server.ServerTool{conflictTool(conflictSvc)}
}

func conflictTool(conflictSvc *service.ConflictService) server.ServerTool {
	tool := mcp.NewTool("conflict",
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithTitleAnnotation("Conflict Risk Analysis"),
		mcp.WithDescription(
			"Analyse conflict risk between two or more tasks that might run in parallel. "+
				"Checks file overlap (planned and git-history), dependency ordering, and "+
				"architectural boundary crossing. Returns per-pair risk assessment and "+
				"recommendation (safe_to_parallelise, serialise, or checkpoint_required). "+
				"Actions: check.",
		),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("Action: check"),
		),
		mcp.WithArray("task_ids",
			mcp.Required(),
			mcp.Description("Two or more task IDs to check for conflict risk"),
		),
	)

	handler := WithSideEffects(func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		return DispatchAction(ctx, req, map[string]ActionHandler{
			"check": conflictCheckAction(conflictSvc),
		})
	})

	return server.ServerTool{Tool: tool, Handler: handler}
}

// ─── check ───────────────────────────────────────────────────────────────────

func conflictCheckAction(conflictSvc *service.ConflictService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		// Extract task_ids array.
		args := req.GetArguments()
		taskIDsRaw, ok := args["task_ids"]
		if !ok {
			return inlineErr("missing_parameter", "task_ids is required")
		}
		taskIDSlice, ok := taskIDsRaw.([]interface{})
		if !ok {
			return inlineErr("invalid_parameter", "task_ids must be an array of strings")
		}
		taskIDs := make([]string, 0, len(taskIDSlice))
		for _, v := range taskIDSlice {
			s, ok := v.(string)
			if !ok {
				return inlineErr("invalid_parameter", "task_ids must be an array of strings")
			}
			taskIDs = append(taskIDs, s)
		}

		result, err := conflictSvc.Check(service.ConflictCheckInput{TaskIDs: taskIDs})
		if err != nil {
			return nil, err
		}

		// Build response matching the 1.0 output format.
		type fileOverlapJSON struct {
			Risk         string   `json:"risk"`
			SharedFiles  []string `json:"shared_files"`
			GitConflicts []string `json:"git_conflicts"`
		}
		type depOrderJSON struct {
			Risk   string `json:"risk"`
			Detail string `json:"detail"`
		}
		type boundaryJSON struct {
			Risk   string `json:"risk"`
			Detail string `json:"detail"`
		}
		type dimensionsJSON struct {
			FileOverlap      fileOverlapJSON `json:"file_overlap"`
			DependencyOrder  depOrderJSON    `json:"dependency_order"`
			BoundaryCrossing boundaryJSON    `json:"boundary_crossing"`
		}
		type pairJSON struct {
			TaskA          string         `json:"task_a"`
			TaskB          string         `json:"task_b"`
			Risk           string         `json:"risk"`
			Dimensions     dimensionsJSON `json:"dimensions"`
			Recommendation string         `json:"recommendation"`
		}

		pairs := make([]pairJSON, len(result.Pairs))
		for i, p := range result.Pairs {
			pairs[i] = pairJSON{
				TaskA: p.TaskA,
				TaskB: p.TaskB,
				Risk:  p.Risk,
				Dimensions: dimensionsJSON{
					FileOverlap: fileOverlapJSON{
						Risk:         p.Dimensions.FileOverlap.Risk,
						SharedFiles:  p.Dimensions.FileOverlap.SharedFiles,
						GitConflicts: p.Dimensions.FileOverlap.GitConflicts,
					},
					DependencyOrder: depOrderJSON{
						Risk:   p.Dimensions.DependencyOrder.Risk,
						Detail: p.Dimensions.DependencyOrder.Detail,
					},
					BoundaryCrossing: boundaryJSON{
						Risk:   p.Dimensions.BoundaryCrossing.Risk,
						Detail: p.Dimensions.BoundaryCrossing.Detail,
					},
				},
				Recommendation: p.Recommendation,
			}
		}

		return map[string]any{
			"task_ids":     result.TaskIDs,
			"overall_risk": result.OverallRisk,
			"pairs":        pairs,
		}, nil
	}
}
