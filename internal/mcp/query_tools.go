package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"kanbanzai/internal/service"
)

// QueryTools returns MCP tools for rich queries across entities and documents.
func QueryTools(entitySvc *service.EntityService, docSvc *service.DocumentService) []server.ServerTool {
	return []server.ServerTool{
		listTagsTool(entitySvc),
		listByTagTool(entitySvc),
		queryPlanTasksTool(entitySvc),
		docSupersessionChainTool(docSvc),
	}
}

func listTagsTool(svc *service.EntityService) server.ServerTool {
	tool := mcp.NewTool("list_tags",
		mcp.WithDescription("List all unique tags across all entity types, sorted alphabetically."),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		tags, err := svc.ListAllTags()
		if err != nil {
			return mcp.NewToolResultErrorFromErr("list tags failed", err), nil
		}
		return jsonResult(map[string]any{
			"success": true,
			"count":   len(tags),
			"tags":    tags,
		})
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func listByTagTool(svc *service.EntityService) server.ServerTool {
	tool := mcp.NewTool("list_by_tag",
		mcp.WithDescription("List all entities across all types that have the given tag."),
		mcp.WithString("tag", mcp.Description("Tag to search for"), mcp.Required()),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		tag, err := request.RequireString("tag")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		results, err := svc.ListEntitiesByTag(tag)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("list by tag failed", err), nil
		}
		return jsonResult(map[string]any{
			"success":  true,
			"tag":      tag,
			"count":    len(results),
			"entities": listResultsWithDisplay(results),
		})
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func queryPlanTasksTool(svc *service.EntityService) server.ServerTool {
	tool := mcp.NewTool("query_plan_tasks",
		mcp.WithDescription("Find all tasks belonging to features under a given Plan. Useful for getting a complete task breakdown for a Plan."),
		mcp.WithString("plan_id", mcp.Description("Plan ID (e.g., P1-basic-ui)"), mcp.Required()),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		planID, err := request.RequireString("plan_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		results, err := svc.CrossEntityQuery(planID)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("query plan tasks failed", err), nil
		}
		return jsonResult(map[string]any{
			"success": true,
			"plan_id": planID,
			"count":   len(results),
			"tasks":   listResultsWithDisplay(results),
		})
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func docSupersessionChainTool(docSvc *service.DocumentService) server.ServerTool {
	tool := mcp.NewTool("doc_supersession_chain",
		mcp.WithDescription("Follow supersedes/superseded_by links to build the full version chain for a document. Returns documents ordered from oldest to newest."),
		mcp.WithString("id", mcp.Description("Document record ID to start from"), mcp.Required()),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		chain, err := docSvc.SupersessionChain(id)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("supersession chain failed", err), nil
		}

		docs := make([]map[string]any, 0, len(chain))
		for _, r := range chain {
			doc := map[string]any{
				"id":           r.ID,
				"path":         r.Path,
				"type":         r.Type,
				"title":        r.Title,
				"status":       r.Status,
				"owner":        r.Owner,
				"content_hash": r.ContentHash,
				"created":      r.Created,
				"updated":      r.Updated,
			}
			if r.Supersedes != "" {
				doc["supersedes"] = r.Supersedes
			}
			if r.SupersededBy != "" {
				doc["superseded_by"] = r.SupersededBy
			}
			docs = append(docs, doc)
		}

		return jsonResult(map[string]any{
			"success":      true,
			"start_id":     id,
			"chain_length": len(chain),
			"chain":        docs,
		})
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}
