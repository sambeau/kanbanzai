package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"kanbanzai/internal/git"
	"kanbanzai/internal/service"
	"kanbanzai/internal/worktree"
)

// worktreeBranchLookup adapts a worktree.Store into the service.BranchLookup interface.
type worktreeBranchLookup struct {
	store *worktree.Store
}

func newWorktreeBranchLookup(store *worktree.Store, repoRoot string) *worktreeBranchLookup {
	return &worktreeBranchLookup{store: store}
}

func (w *worktreeBranchLookup) GetBranchForEntity(entityID string) (string, error) {
	rec, err := w.store.GetByEntityID(entityID)
	if err != nil {
		return "", err
	}
	return rec.Branch, nil
}

func (w *worktreeBranchLookup) GetFilesOnBranch(repoRoot, branch string) ([]string, error) {
	return git.GetFilesChangedOnBranch(repoRoot, branch)
}

// ConflictTools returns the conflict_domain_check MCP tool.
func ConflictTools(conflictSvc *service.ConflictService) []server.ServerTool {
	return []server.ServerTool{
		conflictDomainCheckTool(conflictSvc),
	}
}

func conflictDomainCheckTool(conflictSvc *service.ConflictService) server.ServerTool {
	tool := mcp.NewTool("conflict_domain_check",
		mcp.WithDescription("Analyse conflict risk between two or more tasks that might run in parallel. Checks file overlap (planned and git-history), dependency ordering, and architectural boundary crossing. Returns per-pair risk assessment and recommendation (safe_to_parallelise, serialise, or checkpoint_required)."),
		mcp.WithArray("task_ids",
			mcp.Description("Two or more task IDs to check for conflict risk"),
			mcp.Required(),
		),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract task_ids from the request
		args, ok := request.Params.Arguments.(map[string]any)
		if !ok {
			return mcp.NewToolResultError("task_ids is required"), nil
		}
		taskIDsRaw, ok := args["task_ids"]
		if !ok {
			return mcp.NewToolResultError("task_ids is required"), nil
		}
		taskIDSlice, ok := taskIDsRaw.([]interface{})
		if !ok {
			return mcp.NewToolResultError("task_ids must be an array of strings"), nil
		}
		taskIDs := make([]string, 0, len(taskIDSlice))
		for _, v := range taskIDSlice {
			s, ok := v.(string)
			if !ok {
				return mcp.NewToolResultError("task_ids must be an array of strings"), nil
			}
			taskIDs = append(taskIDs, s)
		}

		result, err := conflictSvc.Check(service.ConflictCheckInput{TaskIDs: taskIDs})
		if err != nil {
			return mcp.NewToolResultErrorFromErr("conflict_domain_check failed", err), nil
		}

		// Build JSON response matching spec §9.2 output format
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

		resp := map[string]any{
			"task_ids":     result.TaskIDs,
			"overall_risk": result.OverallRisk,
			"pairs":        pairs,
		}

		data, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshal result: %s", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}
