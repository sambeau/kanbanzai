package mcp

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"kanbanzai/internal/config"
	"kanbanzai/internal/service"
)

// IncidentTools returns all incident-related MCP tool definitions with their handlers.
func IncidentTools(svc *service.EntityService) []server.ServerTool {
	return []server.ServerTool{
		incidentCreateTool(svc),
		incidentUpdateTool(svc),
		incidentListTool(svc),
		incidentLinkBugTool(svc),
	}
}

func incidentCreateTool(svc *service.EntityService) server.ServerTool {
	tool := mcp.NewTool("incident_create",
		mcp.WithDescription("Create a new incident entity in reported status"),
		mcp.WithString("slug", mcp.Description("URL-friendly identifier for the incident"), mcp.Required()),
		mcp.WithString("title", mcp.Description("Title of the incident"), mcp.Required()),
		mcp.WithString("severity", mcp.Description("Incident severity: critical, high, medium, or low"), mcp.Required()),
		mcp.WithString("summary", mcp.Description("Brief summary of the incident"), mcp.Required()),
		mcp.WithString("reported_by", mcp.Description("Who reported the incident. Auto-resolved from .kbz/local.yaml or git config if not provided."), mcp.Required()),
		mcp.WithString("detected_at", mcp.Description("When the incident was detected (ISO 8601). Defaults to now if not provided.")),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		slug, err := request.RequireString("slug")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		title, err := request.RequireString("title")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		severity, err := request.RequireString("severity")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		summary, err := request.RequireString("summary")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		reportedByRaw, err := request.RequireString("reported_by")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		reportedBy, err := config.ResolveIdentity(reportedByRaw)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		detectedAt := request.GetString("detected_at", "")

		result, err := svc.CreateIncident(service.CreateIncidentInput{
			Slug:       slug,
			Title:      title,
			Severity:   severity,
			Summary:    summary,
			ReportedBy: reportedBy,
			DetectedAt: detectedAt,
		})
		if err != nil {
			return mcp.NewToolResultErrorFromErr("create incident failed", err), nil
		}
		return jsonResult(createResultWithDisplay(result))
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func incidentUpdateTool(svc *service.EntityService) server.ServerTool {
	tool := mcp.NewTool("incident_update",
		mcp.WithDescription("Update an existing incident. Can change status (with lifecycle validation), severity, summary, timestamps, and affected features."),
		mcp.WithString("incident_id", mcp.Description("Incident ID (full or prefix)"), mcp.Required()),
		mcp.WithString("status", mcp.Description("New lifecycle status")),
		mcp.WithString("severity", mcp.Description("New severity: critical, high, medium, or low")),
		mcp.WithString("summary", mcp.Description("Updated summary")),
		mcp.WithString("triaged_at", mcp.Description("When the incident was triaged (ISO 8601)")),
		mcp.WithString("mitigated_at", mcp.Description("When the incident was mitigated (ISO 8601)")),
		mcp.WithString("resolved_at", mcp.Description("When the incident was resolved (ISO 8601)")),
		mcp.WithArray("affected_features", mcp.Description("List of affected feature IDs (replaces existing list)")),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		incidentID, err := request.RequireString("incident_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		input := service.UpdateIncidentInput{
			ID:          incidentID,
			Status:      request.GetString("status", ""),
			Severity:    request.GetString("severity", ""),
			Summary:     request.GetString("summary", ""),
			TriagedAt:   request.GetString("triaged_at", ""),
			MitigatedAt: request.GetString("mitigated_at", ""),
			ResolvedAt:  request.GetString("resolved_at", ""),
		}

		// Parse affected_features array if provided
		args := request.GetArguments()
		if featuresRaw, ok := args["affected_features"]; ok {
			if featuresArr, ok := featuresRaw.([]any); ok {
				features := make([]string, 0, len(featuresArr))
				for _, f := range featuresArr {
					if s, ok := f.(string); ok {
						features = append(features, s)
					}
				}
				input.AffectedFeatures = features
			}
		}

		result, err := svc.UpdateIncident(input)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("update incident failed", err), nil
		}

		return jsonResult(result)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func incidentListTool(svc *service.EntityService) server.ServerTool {
	tool := mcp.NewTool("incident_list",
		mcp.WithDescription("List incidents with optional status and severity filters"),
		mcp.WithString("status", mcp.Description("Filter by status (e.g. reported, triaged, investigating, resolved, closed)")),
		mcp.WithString("severity", mcp.Description("Filter by severity (critical, high, medium, low)")),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		statusFilter := request.GetString("status", "")
		severityFilter := request.GetString("severity", "")

		results, err := svc.ListIncidents(statusFilter, severityFilter)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("list incidents failed", err), nil
		}

		data, err := json.Marshal(results)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to marshal result", err), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func incidentLinkBugTool(svc *service.EntityService) server.ServerTool {
	tool := mcp.NewTool("incident_link_bug",
		mcp.WithDescription("Link a bug to an incident. Adds the bug to the incident's linked_bugs list. Idempotent — linking the same bug twice has no effect."),
		mcp.WithString("incident_id", mcp.Description("Incident ID (full or prefix)"), mcp.Required()),
		mcp.WithString("bug_id", mcp.Description("Bug ID to link"), mcp.Required()),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		incidentID, err := request.RequireString("incident_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		bugID, err := request.RequireString("bug_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		result, err := svc.LinkBug(service.LinkBugInput{
			IncidentID: incidentID,
			BugID:      bugID,
		})
		if err != nil {
			return mcp.NewToolResultErrorFromErr("link bug failed", err), nil
		}

		return jsonResult(result)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}
