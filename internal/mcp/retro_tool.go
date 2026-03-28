// Package mcp retro_tool.go — retrospective synthesis tool for Kanbanzai 2.0 (P5 Phase 2).
//
// retro(action?) synthesises accumulated retrospective signals from the knowledge base:
//   - synthesise (default): clusters signals by category and Jaccard similarity, ranks
//     themes by severity-weighted signal count, and returns a structured response.
//   - report: runs synthesis and additionally generates a markdown document, writes it
//     to the given output_path, and registers it as a document record.
//
// The retro tool is a member of the planning feature group (spec §7.5).
package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/sambeau/kanbanzai/internal/config"
	"github.com/sambeau/kanbanzai/internal/service"
)

// RetroTool returns the retro consolidated tool registered in the planning feature group.
func RetroTool(retroSvc *service.RetroService) []server.ServerTool {
	return []server.ServerTool{retroTool(retroSvc)}
}

func retroTool(retroSvc *service.RetroService) server.ServerTool {
	tool := mcp.NewTool("retro",
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithTitleAnnotation("Retrospective Synthesis"),
		mcp.WithDescription(
			"Synthesise retrospective signals into themed clusters. "+
				"Before writing any retrospective or review document, call action: synthesise first "+
				"— it surfaces signals from across the project that may not be in your session context. "+
				"Actions: synthesise (read signals, cluster, rank), report (generate and register a "+
				"markdown report document).",
		),
		mcp.WithString("action",
			mcp.Description("Action: synthesise (default) or report"),
		),
		mcp.WithString("scope",
			mcp.Description("Plan ID, Feature ID, or \"project\" (default: \"project\")"),
		),
		mcp.WithString("since",
			mcp.Description("ISO 8601 timestamp; only include signals created after this time"),
		),
		mcp.WithString("until",
			mcp.Description("ISO 8601 timestamp; only include signals created before this time"),
		),
		mcp.WithString("min_severity",
			mcp.Description("Minimum severity to include: minor (default), moderate, or significant"),
		),
		mcp.WithString("output_path",
			mcp.Description("Repository-relative path for the generated report file (required for report action)"),
		),
		mcp.WithString("title",
			mcp.Description("Title for the document record (report action; defaults to \"Retrospective: {scope} {date}\")"),
		),
	)

	handler := WithSideEffects(func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		args, _ := req.Params.Arguments.(map[string]any)
		action := retroArgStr(args, "action")
		if action == "" {
			action = "synthesise"
		}

		switch action {
		case "synthesise", "synthesize":
			return retroSynthesiseAction(ctx, args, retroSvc)
		case "report":
			SignalMutation(ctx)
			return retroReportAction(ctx, args, retroSvc)
		default:
			return inlineErr("unknown_action", fmt.Sprintf(
				"unknown action %q; valid actions: synthesise, report", action,
			))
		}
	})

	return server.ServerTool{Tool: tool, Handler: handler}
}

// retroSynthesiseAction handles the synthesise action.
func retroSynthesiseAction(_ context.Context, args map[string]any, svc *service.RetroService) (any, error) {
	input := service.RetroSynthesisInput{
		Scope:       retroArgStr(args, "scope"),
		Since:       retroArgStr(args, "since"),
		Until:       retroArgStr(args, "until"),
		MinSeverity: retroArgStr(args, "min_severity"),
	}
	result, err := svc.Synthesise(input)
	if err != nil {
		return inlineErr("synthesis_failed", err.Error())
	}
	return result, nil
}

// retroReportAction handles the report action.
func retroReportAction(_ context.Context, args map[string]any, svc *service.RetroService) (any, error) {
	outputPath := retroArgStr(args, "output_path")
	if outputPath == "" {
		return inlineErr("missing_parameter", "output_path is required for report action")
	}

	// Auto-resolve the caller identity for document registration.
	createdBy, err := config.ResolveIdentity("")
	if err != nil {
		createdBy = "retro"
	}

	input := service.RetroReportInput{
		RetroSynthesisInput: service.RetroSynthesisInput{
			Scope:       retroArgStr(args, "scope"),
			Since:       retroArgStr(args, "since"),
			Until:       retroArgStr(args, "until"),
			MinSeverity: retroArgStr(args, "min_severity"),
		},
		OutputPath: outputPath,
		Title:      retroArgStr(args, "title"),
		CreatedBy:  createdBy,
	}
	result, err := svc.Report(input)
	if err != nil {
		return inlineErr("report_failed", err.Error())
	}
	return result, nil
}

// retroArgStr extracts and trims a string field from an args map.
func retroArgStr(args map[string]any, key string) string {
	if args == nil {
		return ""
	}
	s, _ := args[key].(string)
	return strings.TrimSpace(s)
}
