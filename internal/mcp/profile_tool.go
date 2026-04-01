package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	kbzctx "github.com/sambeau/kanbanzai/internal/context"
)

// ProfileTool returns the 2.0 profile consolidated tool.
// It consolidates profile_list and profile_get into a single tool (spec §18.2).
// Updated for 3.0 to use RoleStore with new role schema fields.
func ProfileTool(roleStore *kbzctx.RoleStore) []server.ServerTool {
	return []server.ServerTool{profileTool(roleStore)}
}

func profileTool(roleStore *kbzctx.RoleStore) server.ServerTool {
	tool := mcp.NewTool("profile",
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
		mcp.WithTitleAnnotation("Context Role Profiles"),
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
			return roleListAction(roleStore)
		case "get":
			return roleGetAction(roleStore, req)
		default:
			return ActionError("unknown_action", fmt.Sprintf("unknown action %q; valid actions: get, list", action), nil), nil
		}
	}

	return server.ServerTool{Tool: tool, Handler: handler}
}

// ─── list ─────────────────────────────────────────────────────────────────────

func roleListAction(store *kbzctx.RoleStore) (*mcp.CallToolResult, error) {
	roles, err := store.LoadAll()
	if err != nil {
		return ActionError("list_failed", "profile list failed: "+err.Error(), nil), nil
	}

	items := make([]map[string]any, 0, len(roles))
	for _, r := range roles {
		item := map[string]any{
			"id":       r.ID,
			"identity": r.Identity,
		}
		if r.Inherits != "" {
			item["inherits"] = r.Inherits
		}
		items = append(items, item)
	}

	return profileToolMapJSON(map[string]any{
		"success":  true,
		"count":    len(roles),
		"profiles": items,
	})
}

// ─── get ──────────────────────────────────────────────────────────────────────

func roleGetAction(store *kbzctx.RoleStore, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetString("id", "")
	if id == "" {
		return ActionError("missing_parameter", "id is required for get action", nil), nil
	}

	resolved := req.GetBool("resolved", true)

	if resolved {
		rr, err := kbzctx.ResolveRole(store, id)
		if err != nil {
			return ActionError("get_failed", "profile_get failed: "+err.Error(), nil), nil
		}

		profileMap := map[string]any{
			"id":       rr.ID,
			"resolved": true,
			"identity": rr.Identity,
		}
		if len(rr.Vocabulary) > 0 {
			profileMap["vocabulary"] = rr.Vocabulary
		}
		if len(rr.AntiPatterns) > 0 {
			profileMap["anti_patterns"] = antiPatternsToSlice(rr.AntiPatterns)
		}
		if len(rr.Tools) > 0 {
			profileMap["tools"] = rr.Tools
		}

		return profileToolMapJSON(map[string]any{
			"success": true,
			"profile": profileMap,
		})
	}

	// resolved=false: return raw role as stored on disk.
	r, err := store.Load(id)
	if err != nil {
		return ActionError("get_failed", "profile_get failed: "+err.Error(), nil), nil
	}

	profileMap := map[string]any{
		"id":       r.ID,
		"resolved": false,
		"identity": r.Identity,
	}
	if r.Inherits != "" {
		profileMap["inherits"] = r.Inherits
	}
	if len(r.Vocabulary) > 0 {
		profileMap["vocabulary"] = r.Vocabulary
	}
	if len(r.AntiPatterns) > 0 {
		profileMap["anti_patterns"] = antiPatternsToSlice(r.AntiPatterns)
	}
	if len(r.Tools) > 0 {
		profileMap["tools"] = r.Tools
	}

	return profileToolMapJSON(map[string]any{
		"success": true,
		"profile": profileMap,
	})
}

// antiPatternsToSlice converts a slice of AntiPattern structs to a slice of maps
// for JSON serialisation in tool responses.
func antiPatternsToSlice(aps []kbzctx.AntiPattern) []map[string]any {
	result := make([]map[string]any, len(aps))
	for i, ap := range aps {
		result[i] = map[string]any{
			"name":    ap.Name,
			"detect":  ap.Detect,
			"because": ap.Because,
			"resolve": ap.Resolve,
		}
	}
	return result
}

// profileToolMapJSON marshals a map to JSON and returns it as a tool result.
func profileToolMapJSON(v map[string]any) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("marshal result: " + err.Error()), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}
