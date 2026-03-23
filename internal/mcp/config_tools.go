package mcp

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"kanbanzai/internal/config"
)

// ConfigTools returns all configuration-related MCP tool definitions with their handlers.
func ConfigTools() []server.ServerTool {
	return []server.ServerTool{
		getProjectConfigTool(),
		getPrefixRegistryTool(),
		addPrefixTool(),
		retirePrefixTool(),
	}
}

func getProjectConfigTool() server.ServerTool {
	tool := mcp.NewTool("get_project_config",
		mcp.WithDescription("Get the project configuration including the prefix registry, version, and other settings. Returns the contents of .kbz/config.yaml."),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cfg, err := config.Load()
		if err != nil {
			// Return defaults with an indication that no config exists
			defaultCfg := config.DefaultConfig()
			response := map[string]any{
				"success":        true,
				"using_defaults": true,
				"message":        "No config file found, returning defaults",
				"config": map[string]any{
					"version":  defaultCfg.Version,
					"prefixes": prefixesToMaps(defaultCfg.Prefixes),
				},
			}
			return configResultJSON(response)
		}

		response := map[string]any{
			"success": true,
			"config": map[string]any{
				"version":  cfg.Version,
				"prefixes": prefixesToMaps(cfg.Prefixes),
			},
		}

		return configResultJSON(response)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func getPrefixRegistryTool() server.ServerTool {
	tool := mcp.NewTool("get_prefix_registry",
		mcp.WithDescription("Get the Plan ID prefix registry. Lists all declared prefixes with their labels and retired status. Use this to find valid prefixes for creating new Plans."),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		cfg := config.LoadOrDefault()

		activePrefixes := cfg.ActivePrefixes()
		var retiredPrefixes []config.PrefixEntry
		for _, p := range cfg.Prefixes {
			if p.Retired {
				retiredPrefixes = append(retiredPrefixes, p)
			}
		}

		response := map[string]any{
			"success":          true,
			"active_prefixes":  prefixesToMaps(activePrefixes),
			"retired_prefixes": prefixesToMaps(retiredPrefixes),
			"total_count":      len(cfg.Prefixes),
			"active_count":     len(activePrefixes),
		}

		return configResultJSON(response)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func addPrefixTool() server.ServerTool {
	tool := mcp.NewTool("add_prefix",
		mcp.WithDescription("Add a new prefix to the Plan ID prefix registry. The prefix must be a single non-digit Unicode character."),
		mcp.WithString("prefix", mcp.Description("Single non-digit character for the prefix"), mcp.Required()),
		mcp.WithString("label", mcp.Description("Human-readable label describing the prefix purpose"), mcp.Required()),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		prefix, err := request.RequireString("prefix")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		label, err := request.RequireString("label")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Validate prefix format
		if err := config.ValidatePrefix(prefix); err != nil {
			return mcp.NewToolResultError("invalid prefix: " + err.Error()), nil
		}

		// Load or create config
		cfg, err := config.Load()
		if err != nil {
			// Config doesn't exist, create a new one
			defaultCfg := config.DefaultConfig()
			cfg = &defaultCfg
		}

		// Check if prefix already exists
		if cfg.IsValidPrefix(prefix) {
			return mcp.NewToolResultError("prefix already exists: " + prefix), nil
		}

		// Add the prefix
		if err := cfg.AddPrefix(prefix, label); err != nil {
			return mcp.NewToolResultError("add prefix failed: " + err.Error()), nil
		}

		// Save the config
		if err := cfg.Save(); err != nil {
			return mcp.NewToolResultError("save config failed: " + err.Error()), nil
		}

		response := map[string]any{
			"success": true,
			"message": "Prefix added successfully",
			"prefix": map[string]any{
				"prefix":  prefix,
				"label":   label,
				"retired": false,
			},
		}

		return configResultJSON(response)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func retirePrefixTool() server.ServerTool {
	tool := mcp.NewTool("retire_prefix",
		mcp.WithDescription("Mark a prefix as retired. Retired prefixes cannot be used for new Plans but remain valid for existing Plans. At least one active prefix must remain."),
		mcp.WithString("prefix", mcp.Description("The prefix to retire"), mcp.Required()),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		prefix, err := request.RequireString("prefix")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		cfg, err := config.Load()
		if err != nil {
			return mcp.NewToolResultError("load config failed: " + err.Error()), nil
		}

		if err := cfg.RetirePrefix(prefix); err != nil {
			return mcp.NewToolResultError("retire prefix failed: " + err.Error()), nil
		}

		// Save the config
		if err := cfg.Save(); err != nil {
			return mcp.NewToolResultError("save config failed: " + err.Error()), nil
		}

		response := map[string]any{
			"success": true,
			"message": "Prefix retired successfully",
			"prefix":  prefix,
		}

		return configResultJSON(response)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

// prefixesToMaps converts a slice of PrefixEntry to a slice of maps for JSON output.
func prefixesToMaps(prefixes []config.PrefixEntry) []map[string]any {
	result := make([]map[string]any, 0, len(prefixes))
	for _, p := range prefixes {
		m := map[string]any{
			"prefix": p.Prefix,
			"label":  p.Label,
		}
		if p.Retired {
			m["retired"] = true
		}
		result = append(result, m)
	}
	return result
}

// configResultJSON marshals the response map to JSON and returns it as a tool result.
func configResultJSON(response map[string]any) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("marshal result: " + err.Error()), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}
