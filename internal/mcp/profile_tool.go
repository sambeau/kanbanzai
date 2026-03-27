package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	kbzctx "kanbanzai/internal/context"
)

// ProfileTool returns the 2.0 profile consolidated tool.
// It consolidates profile_list and profile_get into a single tool (spec §18.2).
func ProfileTool(store *kbzctx.ProfileStore) []server.ServerTool {
	return []server.ServerTool{profileTool(store)}
}

func profileTool(store *kbzctx.ProfileStore) server.ServerTool {
	tool := mcp.NewTool("profile",
		mcp.WithDescription(
			"List and retrieve context role profiles. "+
				"Consolidates profile_list and profile_get. "+
				"Actions: list (list all profiles), get (get a profile by ID, resolved or raw).",
		),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("Action: list, get"),
		),
		mcp.WithString("id",
			mcp.Description("Profile ID (filename without .yaml extension) — required for get"),
		),
		mcp.WithBoolean("resolved",
			mcp.Description("Whether to apply inheritance resolution (default: true) — get only"),
		),
	)

	// Profiles are read-only; no WithSideEffects wrapper needed (spec §8.5).
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		action, _ := args["action"].(string)
		if action == "" {
			return ActionError("missing_parameter", "action is required; valid actions: get, list", nil), nil
		}

		switch action {
		case "list":
			return profileListAction(store)
		case "get":
			return profileGetAction(store, req)
		default:
			return ActionError("unknown_action", fmt.Sprintf("unknown action %q; valid actions: get, list", action), nil), nil
		}
	}

	return server.ServerTool{Tool: tool, Handler: handler}
}

// ─── list ─────────────────────────────────────────────────────────────────────

func profileListAction(store *kbzctx.ProfileStore) (*mcp.CallToolResult, error) {
	profiles, err := store.LoadAll()
	if err != nil {
		return ActionError("list_failed", "profile list failed: "+err.Error(), nil), nil
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

	return profileToolMapJSON(map[string]any{
		"success":  true,
		"count":    len(profiles),
		"profiles": items,
	})
}

// ─── get ──────────────────────────────────────────────────────────────────────

func profileGetAction(store *kbzctx.ProfileStore, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetString("id", "")
	if id == "" {
		return ActionError("missing_parameter", "id is required for get action", nil), nil
	}

	resolved := req.GetBool("resolved", true)

	if resolved {
		rp, err := kbzctx.ResolveProfile(store, id)
		if err != nil {
			return ActionError("get_failed", "profile_get failed: "+err.Error(), nil), nil
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

		return profileToolMapJSON(map[string]any{
			"success": true,
			"profile": profileMap,
		})
	}

	// resolved=false: return raw profile as stored on disk.
	p, err := store.Load(id)
	if err != nil {
		return ActionError("get_failed", "profile_get failed: "+err.Error(), nil), nil
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

	return profileToolMapJSON(map[string]any{
		"success": true,
		"profile": profileMap,
	})
}

// profileToolMapJSON marshals a map to JSON and returns it as a tool result.
func profileToolMapJSON(v map[string]any) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("marshal result: " + err.Error()), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

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
