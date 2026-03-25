package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"kanbanzai/internal/service"
)

// DecomposeTools returns the MCP tools for feature decomposition and review.
func DecomposeTools(svc *service.DecomposeService) []server.ServerTool {
	return []server.ServerTool{
		decomposeFeatureTool(svc),
		decomposeReviewTool(svc),
	}
}

func decomposeFeatureTool(svc *service.DecomposeService) server.ServerTool {
	tool := mcp.NewTool("decompose_feature",
		mcp.WithDescription(
			"Propose a task decomposition for a feature based on its linked specification document. "+
				"Applies embedded decomposition guidance (vertical slices, size limits, explicit dependencies). "+
				"Returns a proposal preview — does NOT write any tasks.",
		),
		mcp.WithString("feature_id",
			mcp.Description("FEAT ID of the feature to decompose"),
			mcp.Required(),
		),
		mcp.WithString("context",
			mcp.Description("Additional guidance for the decomposition (passed as orchestration context)"),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		featureID, err := request.RequireString("feature_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		input := service.DecomposeInput{
			FeatureID: featureID,
			Context:   request.GetString("context", ""),
		}

		result, err := svc.DecomposeFeature(input)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("decompose_feature failed", err), nil
		}

		return jsonResult(result)
	}

	return server.ServerTool{Tool: tool, Handler: handler}
}

func decomposeReviewTool(svc *service.DecomposeService) server.ServerTool {
	tool := mcp.NewTool("decompose_review",
		mcp.WithDescription(
			"Review a decomposition proposal against a feature's specification. "+
				"Checks for uncovered acceptance criteria, oversized tasks, dependency cycles, "+
				"and ambiguous summaries. Returns structured findings with pass/fail/warn status.",
		),
		mcp.WithString("feature_id",
			mcp.Description("FEAT ID of the feature"),
			mcp.Required(),
		),
		mcp.WithObject("proposal",
			mcp.Description("The proposal object from decompose_feature output"),
			mcp.Required(),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		featureID, err := request.RequireString("feature_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Extract proposal from arguments.
		args := request.GetArguments()
		proposalRaw, ok := args["proposal"]
		if !ok {
			return mcp.NewToolResultError("proposal is required"), nil
		}

		proposal, err := parseProposal(proposalRaw)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid proposal", err), nil
		}

		input := service.DecomposeReviewInput{
			FeatureID: featureID,
			Proposal:  proposal,
		}

		result, err := svc.ReviewProposal(input)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("decompose_review failed", err), nil
		}

		return jsonResult(result)
	}

	return server.ServerTool{Tool: tool, Handler: handler}
}

// parseProposal converts the raw proposal argument (map or JSON) into a
// service.Proposal. The MCP transport may deliver this as a map[string]any
// or as a pre-parsed structure.
func parseProposal(raw any) (service.Proposal, error) {
	// Marshal to JSON then unmarshal into the typed struct. This handles
	// both map[string]any and already-typed inputs uniformly.
	data, err := json.Marshal(raw)
	if err != nil {
		return service.Proposal{}, fmt.Errorf("marshal proposal: %w", err)
	}

	var proposal service.Proposal
	if err := json.Unmarshal(data, &proposal); err != nil {
		return service.Proposal{}, fmt.Errorf("unmarshal proposal: %w", err)
	}

	return proposal, nil
}
