package mcp

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"kanbanzai/internal/config"
	"kanbanzai/internal/service"
)

// KnowledgeTools returns all knowledge entry MCP tool definitions with their handlers.
func KnowledgeTools(svc *service.KnowledgeService) []server.ServerTool {
	return []server.ServerTool{
		knowledgeContributeTool(svc),
		knowledgeGetTool(svc),
		knowledgeListTool(svc),
		knowledgeUpdateTool(svc),
		knowledgeConfirmTool(svc),
		knowledgeFlagTool(svc),
		knowledgeRetireTool(svc),
		knowledgePromoteTool(svc),
		knowledgeContextReportTool(svc),
	}
}

func knowledgeContributeTool(svc *service.KnowledgeService) server.ServerTool {
	tool := mcp.NewTool("knowledge_contribute",
		mcp.WithDescription("Contribute a new knowledge entry to the shared knowledge base. Topics are normalised (lowercased, hyphenated). Duplicate detection rejects entries with an identical topic or similar content (Jaccard > 0.65) in the same scope."),
		mcp.WithString("topic", mcp.Description("Topic identifier for the knowledge entry (will be normalised)"), mcp.Required()),
		mcp.WithString("content", mcp.Description("Concise, actionable statement of the knowledge"), mcp.Required()),
		mcp.WithString("scope", mcp.Description("Scope of the entry: a profile name or \"project\""), mcp.Required()),
		mcp.WithNumber("tier", mcp.Description("Knowledge tier: 2 (project-level) or 3 (session-level, default)")),
		mcp.WithString("learned_from", mcp.Description("Optional provenance: Task ID or other reference")),
		mcp.WithString("created_by", mcp.Description("Identity of the contributor")),
		mcp.WithArray("tags", mcp.WithStringItems(), mcp.Description("Optional classification tags")),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		topic, err := request.RequireString("topic")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		content, err := request.RequireString("content")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		scope, err := request.RequireString("scope")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		tier := int(request.GetFloat("tier", 3))
		learnedFrom := request.GetString("learned_from", "")
		createdByRaw := request.GetString("created_by", "")
		createdBy, err := config.ResolveIdentity(createdByRaw)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		tags := request.GetStringSlice("tags", nil)

		input := service.ContributeInput{
			Topic:       topic,
			Content:     content,
			Scope:       scope,
			Tier:        tier,
			LearnedFrom: learnedFrom,
			CreatedBy:   createdBy,
			Tags:        tags,
		}

		record, duplicate, err := svc.Contribute(input)
		if err != nil {
			if duplicate != nil {
				resp := map[string]any{
					"success":   false,
					"duplicate": true,
					"message":   err.Error(),
					"existing":  duplicate.Fields,
				}
				return knowledgeMapJSON(resp)
			}
			return mcp.NewToolResultErrorFromErr("contribute knowledge entry failed", err), nil
		}

		resp := map[string]any{
			"success": true,
			"message": "Knowledge entry contributed successfully",
			"entry":   record.Fields,
		}
		return knowledgeMapJSON(resp)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func knowledgeGetTool(svc *service.KnowledgeService) server.ServerTool {
	tool := mcp.NewTool("knowledge_get",
		mcp.WithDescription("Get a knowledge entry by ID."),
		mcp.WithString("id", mcp.Description("Knowledge entry ID (KE-...)"), mcp.Required()),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		record, err := svc.Get(id)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("get knowledge entry failed", err), nil
		}

		resp := map[string]any{
			"success": true,
			"entry":   record.Fields,
		}
		return knowledgeMapJSON(resp)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func knowledgeListTool(svc *service.KnowledgeService) server.ServerTool {
	tool := mcp.NewTool("knowledge_list",
		mcp.WithDescription("List knowledge entries with optional filters. Retired entries are excluded by default."),
		mcp.WithNumber("tier", mcp.Description("Filter by tier: 2 or 3")),
		mcp.WithString("scope", mcp.Description("Filter by scope")),
		mcp.WithString("status", mcp.Description("Filter by status: contributed, confirmed, disputed, stale, retired")),
		mcp.WithString("topic", mcp.Description("Filter by exact normalised topic")),
		mcp.WithNumber("min_confidence", mcp.Description("Minimum confidence score (0.0–1.0)")),
		mcp.WithArray("tags", mcp.WithStringItems(), mcp.Description("Filter: entries must have all of these tags")),
		mcp.WithBoolean("include_retired", mcp.Description("Include retired entries (default: false)")),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		filters := service.KnowledgeFilters{
			Tier:           int(request.GetFloat("tier", 0)),
			Scope:          request.GetString("scope", ""),
			Status:         request.GetString("status", ""),
			Topic:          request.GetString("topic", ""),
			MinConfidence:  request.GetFloat("min_confidence", 0),
			Tags:           request.GetStringSlice("tags", nil),
			IncludeRetired: request.GetBool("include_retired", false),
		}

		records, err := svc.List(filters)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("list knowledge entries failed", err), nil
		}

		entries := make([]map[string]any, 0, len(records))
		for _, rec := range records {
			entries = append(entries, rec.Fields)
		}

		resp := map[string]any{
			"success": true,
			"count":   len(records),
			"entries": entries,
		}
		return knowledgeMapJSON(resp)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func knowledgeUpdateTool(svc *service.KnowledgeService) server.ServerTool {
	tool := mcp.NewTool("knowledge_update",
		mcp.WithDescription("Update the content of a knowledge entry. Resets use_count, miss_count, and confidence to defaults."),
		mcp.WithString("id", mcp.Description("Knowledge entry ID (KE-...)"), mcp.Required()),
		mcp.WithString("content", mcp.Description("New content for the entry"), mcp.Required()),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		content, err := request.RequireString("content")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		record, err := svc.Update(id, content)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("update knowledge entry failed", err), nil
		}

		resp := map[string]any{
			"success": true,
			"message": "Knowledge entry updated successfully",
			"entry":   record.Fields,
		}
		return knowledgeMapJSON(resp)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func knowledgeConfirmTool(svc *service.KnowledgeService) server.ServerTool {
	tool := mcp.NewTool("knowledge_confirm",
		mcp.WithDescription("Manually confirm a knowledge entry, transitioning it from contributed or disputed to confirmed status."),
		mcp.WithString("id", mcp.Description("Knowledge entry ID (KE-...)"), mcp.Required()),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		record, err := svc.Confirm(id)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("confirm knowledge entry failed", err), nil
		}

		resp := map[string]any{
			"success": true,
			"message": "Knowledge entry confirmed",
			"entry":   record.Fields,
		}
		return knowledgeMapJSON(resp)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func knowledgeFlagTool(svc *service.KnowledgeService) server.ServerTool {
	tool := mcp.NewTool("knowledge_flag",
		mcp.WithDescription("Flag a knowledge entry as incorrect or disputed. Increments miss_count and recomputes confidence. If miss_count reaches 2, the entry is automatically retired."),
		mcp.WithString("id", mcp.Description("Knowledge entry ID (KE-...)"), mcp.Required()),
		mcp.WithString("reason", mcp.Description("Reason for flagging the entry"), mcp.Required()),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		reason := request.GetString("reason", "")
		if reason == "" {
			return mcp.NewToolResultError("missing required parameter: reason"), nil
		}

		record, err := svc.Flag(id, reason)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("flag knowledge entry failed", err), nil
		}

		resp := map[string]any{
			"success": true,
			"message": "Knowledge entry flagged",
			"entry":   record.Fields,
		}
		return knowledgeMapJSON(resp)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func knowledgeRetireTool(svc *service.KnowledgeService) server.ServerTool {
	tool := mcp.NewTool("knowledge_retire",
		mcp.WithDescription("Manually retire a knowledge entry, marking it as no longer valid. Retired entries are excluded from listing by default."),
		mcp.WithString("id", mcp.Description("Knowledge entry ID (KE-...)"), mcp.Required()),
		mcp.WithString("reason", mcp.Description("Reason for retiring the entry"), mcp.Required()),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		reason := request.GetString("reason", "")
		if reason == "" {
			return mcp.NewToolResultError("missing required parameter: reason"), nil
		}

		record, err := svc.Retire(id, reason)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("retire knowledge entry failed", err), nil
		}

		resp := map[string]any{
			"success": true,
			"message": "Knowledge entry retired",
			"entry":   record.Fields,
		}
		return knowledgeMapJSON(resp)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func knowledgePromoteTool(svc *service.KnowledgeService) server.ServerTool {
	tool := mcp.NewTool("knowledge_promote",
		mcp.WithDescription("Promote a tier-3 knowledge entry to tier 2 in place, extending its TTL from 30 to 90 days."),
		mcp.WithString("id", mcp.Description("Knowledge entry ID (KE-...)"), mcp.Required()),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		record, err := svc.Promote(id)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("promote knowledge entry failed", err), nil
		}

		resp := map[string]any{
			"success": true,
			"message": "Knowledge entry promoted to tier 2",
			"entry":   record.Fields,
		}
		return knowledgeMapJSON(resp)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func knowledgeContextReportTool(svc *service.KnowledgeService) server.ServerTool {
	tool := mcp.NewTool("context_report",
		mcp.WithDescription("Report knowledge entry usage from a completed task. For each used entry: increments use_count and updates last_used; auto-confirms if use_count >= 3 and miss_count == 0. For each flagged entry: increments miss_count; auto-retires if miss_count >= 2."),
		mcp.WithString("task_id", mcp.Description("ID of the task that consumed the knowledge entries"), mcp.Required()),
		mcp.WithArray("used", mcp.WithStringItems(), mcp.Description("List of knowledge entry IDs that were used and found helpful"), mcp.Required()),
		mcp.WithString("flagged", mcp.Description("JSON array of flagged entries: [{\"entry_id\": \"KE-...\", \"reason\": \"...\"}]")),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		taskID, err := request.RequireString("task_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		used := request.GetStringSlice("used", nil)
		if len(used) == 0 {
			return mcp.NewToolResultError("missing required parameter: used (list of knowledge entry IDs)"), nil
		}

		flaggedRaw := request.GetString("flagged", "")
		flagged, err := service.ParseFlaggedEntries(flaggedRaw)
		if err != nil {
			return mcp.NewToolResultError("parse flagged entries: " + err.Error()), nil
		}

		if err := svc.ContextReport(taskID, used, flagged); err != nil {
			return mcp.NewToolResultErrorFromErr("context report failed", err), nil
		}

		resp := map[string]any{
			"success":       true,
			"task_id":       taskID,
			"used_count":    len(used),
			"flagged_count": len(flagged),
			"message":       "Context report processed successfully",
		}
		return knowledgeMapJSON(resp)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

// knowledgeMapJSON marshals a map to JSON and returns it as a tool result.
func knowledgeMapJSON(v map[string]any) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("marshal result: " + err.Error()), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}
