package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/sambeau/kanbanzai/internal/git"
	"github.com/sambeau/kanbanzai/internal/service"
)

// decomposeCommitFunc is the function called after decompose apply to commit
// the created task files atomically. Package-level variable for test injection.
// Production value delegates to git.CommitStateWithMessage (FR-A08).
var decomposeCommitFunc = func(repoRoot, message string) (bool, error) {
	return git.CommitStateWithMessage(repoRoot, message)
}

// DecomposeTool returns the 2.0 decompose consolidated tool.
// It wraps DecomposeService (propose/review/slice) and EntityService (apply).
func DecomposeTool(decomposeSvc *service.DecomposeService, entitySvc *service.EntityService) []server.ServerTool {
	return []server.ServerTool{decomposeTool(decomposeSvc, entitySvc)}
}

func decomposeTool(decomposeSvc *service.DecomposeService, entitySvc *service.EntityService) server.ServerTool {
	tool := mcp.NewTool("decompose",
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithTitleAnnotation("Feature Decomposition"),
		mcp.WithDescription(
			"Use when a feature needs to be broken into implementation tasks — the standard workflow "+
				"for feature decomposition. Follow the propose → review → apply sequence: propose generates "+
				"a task breakdown from the feature's specification, review validates it, apply creates the tasks. "+
				"Do NOT manually create tasks with entity(action: \"create\") when a structured decomposition is "+
				"needed — decompose produces dependency-aware, spec-traced task proposals. "+
				"Call AFTER the feature has an approved specification. "+
				"For all actions: feature_id is required. For review and apply: the proposal object from the "+
				"previous step is required. The slice action provides vertical slice analysis without creating tasks.",
		),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("Action: propose, review, apply, slice"),
		),
		mcp.WithString("feature_id",
			mcp.Description("FEAT ID of the feature (required for propose, review, apply, slice)"),
		),
		mcp.WithString("context",
			mcp.Description("Additional guidance for the decomposition (propose only)"),
		),
		mcp.WithObject("proposal",
			mcp.Description("The proposal object from propose output (required for review and apply)"),
		),
	)

	handler := WithSideEffects(func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		return DispatchAction(ctx, req, map[string]ActionHandler{
			"propose": decomposePropose(decomposeSvc),
			"review":  decomposeReview(decomposeSvc),
			"apply":   decomposeApply(entitySvc),
			"slice":   decomposeSlice(decomposeSvc),
		})
	})

	return server.ServerTool{Tool: tool, Handler: handler}
}

// ─── propose ─────────────────────────────────────────────────────────────────

func decomposePropose(svc *service.DecomposeService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		featureID, err := req.RequireString("feature_id")
		if err != nil {
			return inlineErr("missing_parameter", "Cannot propose task decomposition: feature_id is missing.\n\nTo resolve:\n  Provide the feature ID: decompose(action: \"propose\", feature_id: \"FEAT-...\")")
		}

		input := service.DecomposeInput{
			FeatureID: featureID,
			Context:   req.GetString("context", ""),
		}

		result, err := svc.DecomposeFeature(input)
		if err != nil {
			return nil, fmt.Errorf("Cannot propose task decomposition for feature %s: %w.\n\nTo resolve:\n  Ensure the feature exists and has an approved specification: doc(action: \"list\", owner: \"%s\")", featureID, err, featureID)
		}

		return result, nil
	}
}

// ─── review ──────────────────────────────────────────────────────────────────

func decomposeReview(svc *service.DecomposeService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		featureID, err := req.RequireString("feature_id")
		if err != nil {
			return inlineErr("missing_parameter", "Cannot review decomposition proposal: feature_id is missing.\n\nTo resolve:\n  Provide the feature ID: decompose(action: \"review\", feature_id: \"FEAT-...\", proposal: {...})")
		}

		args := req.GetArguments()
		proposalRaw, ok := args["proposal"]
		if !ok {
			return inlineErr("missing_parameter", "Cannot review decomposition proposal: proposal is missing.\n\nTo resolve:\n  Run decompose(action: \"propose\", feature_id: \"FEAT-...\") first and pass the returned proposal object unmodified")
		}

		proposal, err := parseProposal(proposalRaw)
		if err != nil {
			return inlineErr("invalid_parameter", err.Error())
		}

		result, err := svc.ReviewProposal(service.DecomposeReviewInput{
			FeatureID: featureID,
			Proposal:  proposal,
		})
		if err != nil {
			return nil, fmt.Errorf("Cannot review decomposition proposal for feature %s: %w.\n\nTo resolve:\n  Verify the proposal was generated by decompose(action: \"propose\", feature_id: \"%s\") and passed unmodified", featureID, err, featureID)
		}

		return result, nil
	}
}

// ─── apply ───────────────────────────────────────────────────────────────────

func decomposeApply(entitySvc *service.EntityService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		SignalMutation(ctx)

		featureID, err := req.RequireString("feature_id")
		if err != nil {
			return inlineErr("missing_parameter", "Cannot apply decomposition proposal: feature_id is missing.\n\nTo resolve:\n  Provide the feature ID: decompose(action: \"apply\", feature_id: \"FEAT-...\", proposal: {...})")
		}

		args := req.GetArguments()
		proposalRaw, ok := args["proposal"]
		if !ok {
			return inlineErr("missing_parameter", "Cannot apply decomposition proposal: proposal is missing.\n\nTo resolve:\n  Run decompose(action: \"propose\", feature_id: \"FEAT-...\") first and pass the returned proposal object unmodified")
		}

		proposal, err := parseProposal(proposalRaw)
		if err != nil {
			return inlineErr("invalid_parameter", err.Error())
		}

		if len(proposal.Tasks) == 0 {
			return inlineErr("invalid_parameter", "Cannot apply decomposition proposal: proposal contains no tasks.\n\nTo resolve:\n  Re-run decompose(action: \"propose\", feature_id: \"FEAT-...\") to generate a proposal with tasks")
		}

		// Pass 1: create all tasks; build slug→ID map for dependency resolution.
		type createdTask struct {
			ID     string
			Slug   string
			Status string
			DepsOn []string // raw slug-based depends_on from the proposal
		}

		slugToID := make(map[string]string, len(proposal.Tasks))
		created := make([]createdTask, 0, len(proposal.Tasks))

		for _, pt := range proposal.Tasks {
			result, err := entitySvc.CreateTask(service.CreateTaskInput{
				ParentFeature: featureID,
				Slug:          pt.Slug,
				Name:          pt.Name,
				Summary:       pt.Summary,
			})
			if err != nil {
				return nil, fmt.Errorf("Cannot create task %q for feature %s: %w.\n\nTo resolve:\n  Check that the task slug is unique within the feature and the feature accepts new tasks: entity(action: \"get\", id: \"%s\")", pt.Slug, featureID, err, featureID)
			}

			slugToID[pt.Slug] = result.ID
			status, _ := result.State["status"].(string)
			created = append(created, createdTask{
				ID:     result.ID,
				Slug:   pt.Slug,
				Status: status,
				DepsOn: pt.DependsOn,
			})
		}

		// Pass 2: resolve slug-based depends_on to task IDs and persist.
		type taskOut struct {
			ID        string   `json:"id"`
			Slug      string   `json:"slug"`
			Status    string   `json:"status"`
			DependsOn []string `json:"depends_on,omitempty"`
		}

		tasksOut := make([]taskOut, 0, len(created))

		for _, ct := range created {
			if len(ct.DepsOn) == 0 {
				tasksOut = append(tasksOut, taskOut{
					ID:     ct.ID,
					Slug:   ct.Slug,
					Status: ct.Status,
				})
				continue
			}

			// Resolve slug→ID; skip any unresolvable slugs (best-effort).
			// Two cases are mutually exclusive:
			//   1. The dep is a slug from the current proposal → resolve to new task ID.
			//   2. The dep is already a full task ID (cross-feature dep) → keep as-is.
			// The else-if ensures the same value is never appended twice.
			resolvedIDs := make([]string, 0, len(ct.DepsOn))
			for _, depSlug := range ct.DepsOn {
				if id, ok := slugToID[depSlug]; ok {
					resolvedIDs = append(resolvedIDs, id)
				} else if len(depSlug) > 5 && (depSlug[:5] == "TASK-" || depSlug[:2] == "T-") {
					// Cross-feature dep: depSlug is already a resolved task ID.
					resolvedIDs = append(resolvedIDs, depSlug)
				}
			}

			if len(resolvedIDs) > 0 {
				// Write depends_on directly to the entity store record.
				store := entitySvc.Store()
				rec, loadErr := store.Load("task", ct.ID, ct.Slug)
				if loadErr == nil {
					rec.Fields["depends_on"] = resolvedIDs
					_, _ = store.Write(rec) // best-effort; task already created
				}
			}

			tasksOut = append(tasksOut, taskOut{
				ID:        ct.ID,
				Slug:      ct.Slug,
				Status:    ct.Status,
				DependsOn: resolvedIDs,
			})
		}

		// Auto-commit all task files after both passes complete (FR-A08).
		// Best-effort: commit failure is logged but does not block the result.
		commitMsg := fmt.Sprintf("workflow(%s): decompose into %d tasks", featureID, len(tasksOut))
		if _, commitErr := decomposeCommitFunc(".", commitMsg); commitErr != nil {
			log.Printf("[decompose] WARNING: auto-commit after apply for %s failed: %v", featureID, commitErr)
		}

		return map[string]any{
			"feature_id":    featureID,
			"tasks_created": tasksOut,
			"total_created": len(tasksOut),
		}, nil
	}
}

// ─── slice ────────────────────────────────────────────────────────────────────

func decomposeSlice(svc *service.DecomposeService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		featureID, err := req.RequireString("feature_id")
		if err != nil {
			return inlineErr("missing_parameter", "Cannot perform slice analysis: feature_id is missing.\n\nTo resolve:\n  Provide the feature ID: decompose(action: \"slice\", feature_id: \"FEAT-...\")")
		}

		result, err := svc.SliceAnalysis(service.SliceAnalysisInput{FeatureID: featureID})
		if err != nil {
			return nil, fmt.Errorf("Cannot perform slice analysis for feature %s: %w.\n\nTo resolve:\n  Ensure the feature exists and has an approved specification: doc(action: \"list\", owner: \"%s\")", featureID, err, featureID)
		}

		return result, nil
	}
}

func parseProposal(raw any) (service.Proposal, error) {
	data, err := json.Marshal(raw)
	if err != nil {
		return service.Proposal{}, fmt.Errorf("Cannot process proposal data: the proposal object could not be serialized: %w.\n\nTo resolve:\n  Re-run decompose(action: \"propose\") to generate a fresh proposal", err)
	}
	var proposal service.Proposal
	if err := json.Unmarshal(data, &proposal); err != nil {
		return service.Proposal{}, fmt.Errorf("Cannot parse proposal data: the proposal object is malformed or corrupt: %w.\n\nTo resolve:\n  Re-run decompose(action: \"propose\") to generate a fresh proposal and pass it unmodified", err)
	}
	return proposal, nil
}
