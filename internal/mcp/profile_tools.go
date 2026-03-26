package mcp

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	kbzctx "kanbanzai/internal/context"
)

// ProfileTools returns all context profile MCP tool definitions with their handlers.
// These are read-only tools; profiles are created and edited on the filesystem directly.
func ProfileTools(store *kbzctx.ProfileStore) []server.ServerTool {
	return []server.ServerTool{
		profileGetTool(store),
		profileListTool(store),
	}
}

func profileGetTool(store *kbzctx.ProfileStore) server.ServerTool {
	tool := mcp.NewTool("profile_get",
		mcp.WithDescription("Get a context profile by ID. By default returns the fully resolved profile with inheritance applied. Set resolved=false to return the raw profile as defined in its file."),
		mcp.WithString("id", mcp.Description("Profile ID (filename without .yaml extension)"), mcp.Required()),
		mcp.WithBoolean("resolved", mcp.Description("Whether to apply inheritance resolution (default: true)")),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		resolved := request.GetBool("resolved", true)

		if resolved {
			rp, err := kbzctx.ResolveProfile(store, id)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("profile_get failed", err), nil
			}

			profileMap := map[string]any{
				"id":       rp.ID,
				"resolved": true,
			}
			if rp.Description != "" {
				profileMap["description"] = rp.Description
			}
			if rp.Packages != nil {
				profileMap["packages"] = rp.Packages
			}
			if rp.Conventions != nil {
				profileMap["conventions"] = rp.Conventions
			}
			if rp.Architecture != nil {
				profileMap["architecture"] = architectureToMap(rp.Architecture)
			}

			response := map[string]any{
				"success": true,
				"profile": profileMap,
			}
			return profileMapJSON(response)
		}

		// resolved=false: return raw profile as stored on disk.
		p, err := store.Load(id)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("profile_get failed", err), nil
		}

		profileMap := map[string]any{
			"id":       p.ID,
			"resolved": false,
		}
		if p.Inherits != "" {
			profileMap["inherits"] = p.Inherits
		}
		if p.Description != "" {
			profileMap["description"] = p.Description
		}
		if p.Packages != nil {
			profileMap["packages"] = p.Packages
		}
		if p.Conventions != nil {
			profileMap["conventions"] = p.Conventions
		}
		if p.Architecture != nil {
			profileMap["architecture"] = architectureToMap(p.Architecture)
		}

		response := map[string]any{
			"success": true,
			"profile": profileMap,
		}
		return profileMapJSON(response)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func profileListTool(store *kbzctx.ProfileStore) server.ServerTool {
	tool := mcp.NewTool("profile_list",
		mcp.WithDescription("List all context profiles with their ID, parent (inherits), and description."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		profiles, err := store.LoadAll()
		if err != nil {
			return mcp.NewToolResultErrorFromErr("profile_list failed", err), nil
		}

		items := make([]map[string]any, 0, len(profiles))
		for _, p := range profiles {
			item := map[string]any{
				"id":          p.ID,
				"inherits":    p.Inherits,
				"description": p.Description,
			}
			items = append(items, item)
		}

		response := map[string]any{
			"success":  true,
			"count":    len(profiles),
			"profiles": items,
		}
		return profileMapJSON(response)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

// architectureToMap converts an Architecture struct to a map for JSON output.
func architectureToMap(a *kbzctx.Architecture) map[string]any {
	if a == nil {
		return nil
	}
	m := map[string]any{}
	if a.Summary != "" {
		m["summary"] = a.Summary
	}
	if a.KeyInterfaces != nil {
		m["key_interfaces"] = a.KeyInterfaces
	}
	return m
}

// profileMapJSON marshals a map to JSON and returns it as a tool result.
func profileMapJSON(v map[string]any) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("marshal result: " + err.Error()), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}
