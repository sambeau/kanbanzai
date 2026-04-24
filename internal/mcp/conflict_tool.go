package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/sambeau/kanbanzai/internal/service"
)

// ConflictTool returns the 2.0 conflict consolidated tool.
// It wraps the existing ConflictService in the action-dispatch pattern.
// The single "check" action passes task_ids or feature_ids through to ConflictService.
// The two modes are mutually exclusive.
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
			"Before dispatching tasks in parallel, check whether they risk conflicting on "+
				"shared files, dependencies, or architectural boundaries. Returns per-pair risk "+
				"assessment and recommendation (safe_to_parallelise, serialise, or checkpoint_required). "+
				"Use INSTEAD OF manually inspecting file lists to decide parallelism. "+
				"For actual merge conflict detection on branches, use branch(action: \"status\") instead. "+
				"Supports two modes: task_ids (two or more TASK-... IDs) or feature_ids (two or more "+
				"FEAT-... IDs). Modes are mutually exclusive. Actions: check.",
		),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("Action: check"),
		),
		mcp.WithArray("task_ids",
			mcp.WithStringItems(),
			mcp.Description("Two or more task IDs to check for conflict risk (mutually exclusive with feature_ids)"),
		),
		mcp.WithArray("feature_ids",
			mcp.WithStringItems(),
			mcp.Description("Two or more feature IDs to check for conflict risk (mutually exclusive with task_ids)"),
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
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
		args := req.GetArguments()

		// extractStringArgs extracts an optional string array from the arguments map.
		// Returns (nil, true) when the key is absent (allowed), (nil, false) when present but wrong type.
		extractStringArgs := func(key string) ([]string, bool) {
			raw, ok := args[key]
			if !ok {
				return nil, true
			}
			slice, ok := raw.([]interface{})
			if !ok {
				return nil, false
			}
			result := make([]string, 0, len(slice))
			for _, v := range slice {
				s, ok := v.(string)
				if !ok {
					return nil, false
				}
				result = append(result, s)
			}
			return result, true
		}

		taskIDs, ok := extractStringArgs("task_ids")
		if !ok {
			return inlineErr("invalid_parameter", "task_ids must be an array of strings")
		}

		featureIDs, ok := extractStringArgs("feature_ids")
		if !ok {
			return inlineErr("invalid_parameter", "feature_ids must be an array of strings")
		}

		if len(taskIDs) > 0 && len(featureIDs) > 0 {
			return inlineErr("mutually_exclusive", "task_ids and feature_ids are mutually exclusive")
		}

		if len(featureIDs) > 0 {
			result, err := conflictSvc.CheckFeatures(featureIDs)
			if err != nil {
				return nil, err
			}
			return buildFeatureConflictJSON(result), nil
		}

		// Task mode (existing path).
		checkResult, err := conflictSvc.Check(service.ConflictCheckInput{TaskIDs: taskIDs})
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

		pairs := make([]pairJSON, len(checkResult.Pairs))
		for i, p := range checkResult.Pairs {
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
			"task_ids":     checkResult.TaskIDs,
			"overall_risk": checkResult.OverallRisk,
			"pairs":        pairs,
		}, nil
	}
}

// buildFeatureConflictJSON serialises a FeatureConflictResult to the wire format.
func buildFeatureConflictJSON(result service.FeatureConflictResult) map[string]any {
	pairs := make([]map[string]any, len(result.Pairs))
	for i, p := range result.Pairs {
		pairs[i] = map[string]any{
			"feature_a": p.FeatureA,
			"feature_b": p.FeatureB,
			"risk":      p.Risk,
			"dimensions": map[string]any{
				"file_overlap": map[string]any{
					"risk":          p.Dimensions.FileOverlap.Risk,
					"shared_files":  p.Dimensions.FileOverlap.SharedFiles,
					"git_conflicts": p.Dimensions.FileOverlap.GitConflicts,
				},
				"dependency_order": map[string]any{
					"risk":   p.Dimensions.DependencyOrder.Risk,
					"detail": p.Dimensions.DependencyOrder.Detail,
				},
				"boundary_crossing": map[string]any{
					"risk":   p.Dimensions.BoundaryCrossing.Risk,
					"detail": p.Dimensions.BoundaryCrossing.Detail,
				},
			},
			"recommendation": p.Recommendation,
		}
	}

	features := make([]map[string]any, len(result.Features))
	for i, f := range result.Features {
		fi := map[string]any{
			"feature_id":    f.FeatureID,
			"files_planned": f.FilesPlanned,
			"no_file_data":  f.NoFileData,
		}
		if f.DriftDays != nil {
			fi["drift_days"] = *f.DriftDays
		}
		features[i] = fi
	}

	return map[string]any{
		"feature_ids":  result.FeatureIDs,
		"overall_risk": result.OverallRisk,
		"pairs":        pairs,
		"features":     features,
	}
}
