package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/sambeau/kanbanzai/internal/config"
	"github.com/sambeau/kanbanzai/internal/id"
	"github.com/sambeau/kanbanzai/internal/service"
)

// IncidentTool returns the 2.0 consolidated incident tool.
// It consolidates incident_create, incident_update, incident_list, and incident_link_bug
// into a single tool with an action parameter (spec §21.1).
func IncidentTool(svc *service.EntityService) []server.ServerTool {
	return []server.ServerTool{incidentTool(svc)}
}

func incidentTool(svc *service.EntityService) server.ServerTool {
	tool := mcp.NewTool("incident",
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithTitleAnnotation("Incident Tracker"),
		mcp.WithDescription(
			"Track production incidents through their lifecycle from detection to resolution. "+
				"Use when a user-facing issue needs structured tracking beyond a single bug report — "+
				"incidents can link multiple bugs and affected features. "+
				"Do NOT use for code defects that aren't user-facing — use entity(type: bug) for those. "+
				"Actions: create (slug, title, severity, summary, reported_by required), "+
				"update (incident_id required), list (optional status_filter, severity_filter), "+
				"link_bug (incident_id and bug_id required).",
		),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("Action: create, update, list, link_bug"),
		),
		// create parameters
		mcp.WithString("slug",
			mcp.Description("URL-friendly identifier for the incident (create only)"),
		),
		mcp.WithString("title",
			mcp.Description("Title of the incident (create only)"),
		),
		mcp.WithString("severity",
			mcp.Description("Incident severity: critical, high, medium, or low (create required; update optional)"),
		),
		mcp.WithString("summary",
			mcp.Description("Brief summary of the incident (create required; update optional)"),
		),
		mcp.WithString("reported_by",
			mcp.Description("Who reported the incident. Auto-resolved from .kbz/local.yaml or git config if not provided (create only)."),
		),
		mcp.WithString("detected_at",
			mcp.Description("When the incident was detected (ISO 8601). Defaults to now if not provided (create only)."),
		),
		// update parameters
		mcp.WithString("incident_id",
			mcp.Description("Incident ID (full or prefix) — required for update and link_bug"),
		),
		mcp.WithString("status",
			mcp.Description("New lifecycle status (update only)"),
		),
		mcp.WithString("triaged_at",
			mcp.Description("When the incident was triaged (ISO 8601) (update only)"),
		),
		mcp.WithString("mitigated_at",
			mcp.Description("When the incident was mitigated (ISO 8601) (update only)"),
		),
		mcp.WithString("resolved_at",
			mcp.Description("When the incident was resolved (ISO 8601) (update only)"),
		),
		mcp.WithArray("affected_features",
			mcp.Description("List of affected feature IDs — replaces existing list (update only)"),
		),
		// list parameters
		mcp.WithString("status_filter",
			mcp.Description("Filter by status: reported, triaged, investigating, resolved, closed (list only)"),
		),
		mcp.WithString("severity_filter",
			mcp.Description("Filter by severity: critical, high, medium, low (list only)"),
		),
		// link_bug parameters
		mcp.WithString("bug_id",
			mcp.Description("Bug ID to link to the incident (link_bug only)"),
		),
	)

	handler := WithSideEffects(func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		return DispatchAction(ctx, req, map[string]ActionHandler{
			"create":   incidentCreateAction(svc),
			"update":   incidentUpdateAction(svc),
			"list":     incidentListAction(svc),
			"link_bug": incidentLinkBugAction(svc),
		})
	})

	return server.ServerTool{Tool: tool, Handler: handler}
}

// ─── create ──────────────────────────────────────────────────────────────────

func incidentCreateAction(svc *service.EntityService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		SignalMutation(ctx)

		slug, err := req.RequireString("slug")
		if err != nil {
			return nil, fmt.Errorf("Cannot create incident: slug is missing.\n\nTo resolve:\n  Provide slug: incident(action: \"create\", slug: \"my-incident\", ...)")
		}
		title, err := req.RequireString("title")
		if err != nil {
			return nil, fmt.Errorf("Cannot create incident: title is missing.\n\nTo resolve:\n  Provide title: incident(action: \"create\", title: \"...\", ...)")
		}
		severity, err := req.RequireString("severity")
		if err != nil {
			return nil, fmt.Errorf("Cannot create incident: severity is missing.\n\nTo resolve:\n  Provide severity (critical, high, medium, or low): incident(action: \"create\", severity: \"high\", ...)")
		}
		summary, err := req.RequireString("summary")
		if err != nil {
			return nil, fmt.Errorf("Cannot create incident: summary is missing.\n\nTo resolve:\n  Provide summary: incident(action: \"create\", summary: \"...\", ...)")
		}
		reportedByRaw, err := req.RequireString("reported_by")
		if err != nil {
			return nil, fmt.Errorf("Cannot create incident: reported_by is missing.\n\nTo resolve:\n  Provide reported_by: incident(action: \"create\", reported_by: \"...\", ...)")
		}
		reportedBy, err := config.ResolveIdentity(reportedByRaw)
		if err != nil {
			return nil, err
		}
		detectedAt := req.GetString("detected_at", "")

		result, err := svc.CreateIncident(service.CreateIncidentInput{
			Slug:       slug,
			Title:      title,
			Severity:   severity,
			Summary:    summary,
			ReportedBy: reportedBy,
			DetectedAt: detectedAt,
		})
		if err != nil {
			return nil, fmt.Errorf("Cannot create incident: %w.\n\nTo resolve:\n  Check that slug, title, severity, and summary are valid and try again", err)
		}

		return createResultWithDisplay(result), nil
	}
}

// ─── update ───────────────────────────────────────────────────────────────────

func incidentUpdateAction(svc *service.EntityService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		SignalMutation(ctx)

		incidentID, err := req.RequireString("incident_id")
		if err != nil {
			return nil, fmt.Errorf("Cannot update incident: incident_id is missing.\n\nTo resolve:\n  Provide incident_id: incident(action: \"update\", incident_id: \"INC-...\", ...)")
		}

		input := service.UpdateIncidentInput{
			ID:          incidentID,
			Status:      req.GetString("status", ""),
			Severity:    req.GetString("severity", ""),
			Summary:     req.GetString("summary", ""),
			TriagedAt:   req.GetString("triaged_at", ""),
			MitigatedAt: req.GetString("mitigated_at", ""),
			ResolvedAt:  req.GetString("resolved_at", ""),
		}

		// Parse affected_features array if provided.
		args := req.GetArguments()
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
			return nil, fmt.Errorf("Cannot update incident %s: %w.\n\nTo resolve:\n  Verify the incident ID exists using incident(action: \"list\") and check the field values", incidentID, err)
		}

		return result, nil
	}
}

// ─── list ─────────────────────────────────────────────────────────────────────

func incidentListAction(svc *service.EntityService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		statusFilter := req.GetString("status_filter", "")
		severityFilter := req.GetString("severity_filter", "")

		results, err := svc.ListIncidents(statusFilter, severityFilter)
		if err != nil {
			return nil, fmt.Errorf("Cannot list incidents: %w.\n\nTo resolve:\n  Check that status_filter and severity_filter values are valid, or omit them to list all", err)
		}

		// Re-encode results as a generic value for the side-effect wrapper.
		data, err := json.Marshal(results)
		if err != nil {
			return nil, fmt.Errorf("Cannot list incidents: result serialization failed: %w.\n\nTo resolve:\n  Retry the request", err)
		}
		var out any
		if err := json.Unmarshal(data, &out); err != nil {
			return nil, fmt.Errorf("Cannot list incidents: result processing failed: %w.\n\nTo resolve:\n  Retry the request", err)
		}
		return out, nil
	}
}

// ─── link_bug ────────────────────────────────────────────────────────────────

// createResultWithDisplay converts a service.CreateResult to a map with a
// formatted display ID for the MCP response.
func createResultWithDisplay(r service.CreateResult) map[string]any {
	return map[string]any{
		"Type":      r.Type,
		"ID":        r.ID,
		"DisplayID": id.FormatFullDisplay(r.ID),
		"Slug":      r.Slug,
		"Path":      r.Path,
		"State":     r.State,
	}
}

func incidentLinkBugAction(svc *service.EntityService) ActionHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
		SignalMutation(ctx)

		incidentID, err := req.RequireString("incident_id")
		if err != nil {
			return nil, fmt.Errorf("Cannot link bug to incident: incident_id is missing.\n\nTo resolve:\n  Provide incident_id: incident(action: \"link_bug\", incident_id: \"INC-...\", bug_id: \"BUG-...\")")
		}
		bugID, err := req.RequireString("bug_id")
		if err != nil {
			return nil, fmt.Errorf("Cannot link bug to incident: bug_id is missing.\n\nTo resolve:\n  Provide bug_id: incident(action: \"link_bug\", incident_id: \"INC-...\", bug_id: \"BUG-...\")")
		}

		result, err := svc.LinkBug(service.LinkBugInput{
			IncidentID: incidentID,
			BugID:      bugID,
		})
		if err != nil {
			return nil, fmt.Errorf("Cannot link bug %s to incident %s: %w.\n\nTo resolve:\n  Verify both IDs exist using incident(action: \"list\") and entity(action: \"get\", id: \"BUG-...\")", bugID, incidentID, err)
		}

		return result, nil
	}
}
