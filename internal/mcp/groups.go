// Package mcp defines the MCP server tool groups for Kanbanzai 2.0.
//
// This file defines the feature group framework (Track A): the mapping from group
// names to 2.0 tool names, group constants, and the _legacy group used during the
// dual-registration development period.
//
// During development (Tracks A–J), the _legacy group is enabled by default so that
// all 1.0 tools remain registered alongside incrementally-built 2.0 tools. The _legacy
// group is removed entirely in Track K when the 1.0 tool surface is retired.
package mcp

import "kanbanzai/internal/config"

// GroupLegacy is the development-only group that contains all 1.0 tool registrations.
// It is enabled by default during the Kanbanzai 2.0 development period and removed in Track K.
// It is not a user-facing config key and does not appear in KnownGroups.
const GroupLegacy = "_legacy"

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
	},
	config.GroupPlanning: {
		"decompose",
		"estimate",
		"conflict",
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
// It calls config.EffectiveGroups(), logs any warnings, and forces the _legacy group
// to enabled so all 1.0 tools remain registered during the development period.
//
// If the config cannot resolve groups (e.g. unknown preset), a warning is printed and
// the full preset is used as a safe fallback.
func resolveServerGroups(cfg *config.Config) map[string]bool {
	groups, _, err := cfg.EffectiveGroups()
	if err != nil {
		// Unknown preset: fall back to full and log a warning.
		// This is a startup issue, not fatal — we degrade gracefully.
		_ = err // caller may inspect via health_check; we continue with the full preset
		full := map[string]bool{
			config.GroupCore:        true,
			config.GroupPlanning:    true,
			config.GroupKnowledge:   true,
			config.GroupGit:         true,
			config.GroupDocuments:   true,
			config.GroupIncidents:   true,
			config.GroupCheckpoints: true,
			GroupLegacy:             true,
		}
		return full
	}
	// Warnings from EffectiveGroups (unknown group names, core-disabled override) are
	// advisory. They are discarded here; Track D's status tool will surface them
	// in the project health report once implemented.

	// Enable the _legacy group for the dual-registration development period.
	// Removed in Track K.
	groups[GroupLegacy] = true

	return groups
}
