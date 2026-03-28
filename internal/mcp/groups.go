// Package mcp defines the MCP server tool groups for Kanbanzai 2.0.
//
// This file defines the feature group framework: the mapping from group names to
// 2.0 tool names, group constants, and the server group resolver.
package mcp

import "kanbanzai/internal/config"

// GroupToolNames maps each feature group name to the 2.0 tool names registered in that group.
// This is the authoritative membership map — it is used by tests to verify tool registration
// and by the server to determine which tools to register when a group is enabled.
//
// During Track A, this map is fully populated with the 2.0 tool names even though the
// tool implementations do not yet exist. The infrastructure is ready before the tools are.
// As each 2.0 tool is implemented (Tracks B–J), its handler is wired into NewServer under
// the appropriate group conditional.
var GroupToolNames = map[string][]string{
	config.GroupCore: {
		"status",
		"next",
		"finish",
		"handoff",
		"entity",
		"doc",
		"health",
		"server_info",
	},
	config.GroupPlanning: {
		"decompose",
		"estimate",
		"conflict",
		"retro",
	},
	config.GroupKnowledge: {
		"knowledge",
		"profile",
	},
	config.GroupGit: {
		"worktree",
		"merge",
		"pr",
		"branch",
		"cleanup",
	},
	config.GroupDocuments: {
		"doc_intel",
	},
	config.GroupIncidents: {
		"incident",
	},
	config.GroupCheckpoints: {
		"checkpoint",
	},
}

// resolveServerGroups returns the effective group configuration for a server instance.
// It calls config.EffectiveGroups(). If the config cannot resolve groups (e.g. unknown
// preset), the full preset is used as a safe fallback.
func resolveServerGroups(cfg *config.Config) map[string]bool {
	groups, _, err := cfg.EffectiveGroups()
	if err != nil {
		// Unknown preset: fall back to full and degrade gracefully.
		return map[string]bool{
			config.GroupCore:        true,
			config.GroupPlanning:    true,
			config.GroupKnowledge:   true,
			config.GroupGit:         true,
			config.GroupDocuments:   true,
			config.GroupIncidents:   true,
			config.GroupCheckpoints: true,
		}
	}
	// Warnings from EffectiveGroups (unknown group names, core-disabled override) are
	// advisory and surfaced via the health tool.
	return groups
}
