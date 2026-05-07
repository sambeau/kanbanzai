// Package mcp retro_tool.go — retrospective synthesis tool for Kanbanzai 2.0 (P5 Phase 2).
//
// retro(action?) synthesises accumulated retrospective signals from the knowledge base:
//   - synthesise (default): clusters signals by category and Jaccard similarity, ranks
//     themes by severity-weighted signal count, and returns a structured response.
//   - report: runs synthesis and additionally generates a markdown document, writes it
//     to the given output_path, and registers it as a document record.
//   - create_fix: synthesises signals and creates Feature entities to address themes.
//     In human-gated mode, a single theme is selected by index. In auto mode, themes
//     are selected by count and/or severity threshold, and features are auto-advanced
//     through the full lifecycle (design → spec → dev-plan → developing).
//
// The retro tool is a member of the planning feature group (spec §7.5).
package mcp

import (
	"context"
	"fmt"
	"math"
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
			"Before writing any retrospective or review document, call action: synthesise first "+
				"— it surfaces signals from across the project that may not be in your session context. "+
				"Do NOT write retrospective documents from memory alone; always synthesise first to avoid "+
				"missing signals. Use finish(retrospective: [...]) to record individual signals — use this "+
				"tool to analyse them. action: report requires output_path; generates a markdown file and "+
				"registers it as a document record. "+
				"Actions: synthesise (default — cluster and rank), report, create_fix.",
		),
		mcp.WithString("action",
			mcp.Description("Action: synthesise (default), report, or create_fix"),
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
		// create_fix parameters.
		mcp.WithString("mode",
			mcp.Description("create_fix mode: human-gated (default, select by theme_index) or auto (batch select by theme_count and/or severity_threshold)"),
		),
		mcp.WithNumber("theme_index",
			mcp.Description("0-based index into ranked themes for human-gated mode (create_fix)"),
		),
		mcp.WithNumber("theme_count",
			mcp.Description("Top N themes to select in auto mode (create_fix)"),
		),
		mcp.WithNumber("severity_threshold",
			mcp.Description("Minimum severity score for theme selection in auto mode (create_fix)"),
		),
		mcp.WithString("name",
			mcp.Description("Optional feature name (create_fix)"),
		),
		mcp.WithString("parent_plan",
			mcp.Description("Optional parent plan ID for created features (create_fix; auto-created if omitted)"),
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
		case "create_fix":
			SignalMutation(ctx)
			return retroCreateFixAction(ctx, args, retroSvc)
		default:
			return inlineErr("unknown_action", fmt.Sprintf(
				"unknown action %q; valid actions: synthesise, report, create_fix", action,
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

// retroCreateFixAction handles the create_fix action.
func retroCreateFixAction(_ context.Context, args map[string]any, svc *service.RetroService) (any, error) {
	// Auto-resolve the caller identity.
	createdBy, err := config.ResolveIdentity("")
	if err != nil {
		createdBy = "retro"
	}

	mode := retroArgStr(args, "mode")

	themeIndex := retroArgFloatAsInt(args, "theme_index")
	themeCount := retroArgFloatAsInt(args, "theme_count")
	severityThreshold := retroArgFloatAsInt(args, "severity_threshold")

	input := service.CreateFixInput{
		RetroSynthesisInput: service.RetroSynthesisInput{
			Scope:       retroArgStr(args, "scope"),
			Since:       retroArgStr(args, "since"),
			Until:       retroArgStr(args, "until"),
			MinSeverity: retroArgStr(args, "min_severity"),
		},
		Mode:              mode,
		ThemeIndex:        themeIndex,
		ThemeCount:        themeCount,
		SeverityThreshold: severityThreshold,
		Name:              retroArgStr(args, "name"),
		ParentPlan:        retroArgStr(args, "parent_plan"),
		CreatedBy:         createdBy,
	}
	result, err := svc.CreateFix(input)
	if err != nil {
		return inlineErr("create_fix_failed", err.Error())
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

// retroArgFloatAsInt extracts a numeric field from an args map and returns it
// as an int. MCP numbers are transmitted as float64. NaN and Inf are treated as 0.
func retroArgFloatAsInt(args map[string]any, key string) int {
	if args == nil {
		return 0
	}
	v, ok := args[key].(float64)
	if !ok {
		return 0
	}
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return int(v)
}
