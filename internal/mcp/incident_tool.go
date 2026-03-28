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
			"Manage incidents. Consolidates incident_create, incident_update, incident_list, "+
				"and incident_link_bug. "+
				"Actions: create (new incident in reported status), update (change status/severity/summary), "+
				"list (filter by status/severity), link_bug (associate a bug with an incident).",
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
			return nil, fmt.Errorf("slug is required for create action")
		}
		title, err := req.RequireString("title")
		if err != nil {
			return nil, fmt.Errorf("title is required for create action")
		}
		severity, err := req.RequireString("severity")
		if err != nil {
			return nil, fmt.Errorf("severity is required for create action")
		}
		summary, err := req.RequireString("summary")
		if err != nil {
			return nil, fmt.Errorf("summary is required for create action")
		}
		reportedByRaw, err := req.RequireString("reported_by")
		if err != nil {
			return nil, fmt.Errorf("reported_by is required for create action")
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
			return nil, fmt.Errorf("create incident: %w", err)
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
			return nil, fmt.Errorf("incident_id is required for update action")
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
			return nil, fmt.Errorf("update incident: %w", err)
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
			return nil, fmt.Errorf("list incidents: %w", err)
		}

		// Re-encode results as a generic value for the side-effect wrapper.
		data, err := json.Marshal(results)
		if err != nil {
			return nil, fmt.Errorf("marshal incident list: %w", err)
		}
		var out any
		if err := json.Unmarshal(data, &out); err != nil {
			return nil, fmt.Errorf("unmarshal incident list: %w", err)
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
			return nil, fmt.Errorf("incident_id is required for link_bug action")
		}
		bugID, err := req.RequireString("bug_id")
		if err != nil {
			return nil, fmt.Errorf("bug_id is required for link_bug action")
		}

		result, err := svc.LinkBug(service.LinkBugInput{
			IncidentID: incidentID,
			BugID:      bugID,
		})
		if err != nil {
			return nil, fmt.Errorf("link bug: %w", err)
		}

		return result, nil
	}
}
