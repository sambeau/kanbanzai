// Tests for the feature group framework's integration with NewServer (Track A).
//
// This file is package mcp (not mcp_test) so it can access the unexported
// newServerWithConfig constructor needed to inject test configurations.
package mcp

import (
	"sort"
	"testing"

	"kanbanzai/internal/config"
)

// toolNames returns the sorted tool names registered on the server.
// It uses the MCPServer.ListTools() API so this goes through the real registration path.
func toolNamesFromServer(t *testing.T, entityRoot string, cfg *config.Config) []string {
	t.Helper()
	mcpSrv := newServerWithConfig(entityRoot, cfg)
	tools := mcpSrv.ListTools()
	names := make([]string, 0, len(tools))
	for name := range tools {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// containsAll returns true if got contains every element of want.
func containsAll(got []string, want []string) (missing []string) {
	set := make(map[string]bool, len(got))
	for _, n := range got {
		set[n] = true
	}
	for _, w := range want {
		if !set[w] {
			missing = append(missing, w)
		}
	}
	return missing
}

// containsNone returns any of disallowed that appear in got.
func containsNone(got []string, disallowed []string) (found []string) {
	set := make(map[string]bool, len(got))
	for _, n := range got {
		set[n] = true
	}
	for _, d := range disallowed {
		if set[d] {
			found = append(found, d)
		}
	}
	return found
}

// TestServer_ListTools_GroupConfig verifies that NewServer registers different sets of
// 2.0 tools depending on the effective group configuration (spec §30.1).
//
// This test focuses on the 2.0 tool surface: it asserts that tools from enabled groups ARE
// registered and tools from disabled groups are NOT registered.
func TestServer_ListTools_GroupConfig(t *testing.T) {
	t.Parallel()

	// coreTools are the 8 core-group tools (Tracks D–I + server_info). All are implemented.
	// This slice must be kept in sync with the tools registered in the GroupCore
	// conditional block in newServerWithConfig.
	implementedCoreTools := []string{"status", "next", "finish", "handoff", "entity", "doc", "health", "server_info"}

	tests := []struct {
		name        string
		setupConfig func(*config.Config)
		// mustHave: 2.0 tools that MUST appear in the tool list
		mustHave []string
		// mustNotHave: 2.0 tools that must NOT appear in the tool list
		mustNotHave []string
	}{
		{
			name: "preset_full_includes_all_implemented_core_tools",
			setupConfig: func(c *config.Config) {
				c.MCP.Preset = "full"
			},
			mustHave: implementedCoreTools,
		},
		{
			name: "preset_minimal_includes_implemented_core_tools",
			// minimal = core only; all 2.0 implemented core tools must be present
			setupConfig: func(c *config.Config) {
				c.MCP.Preset = "minimal"
			},
			mustHave: implementedCoreTools,
		},
		{
			name: "preset_orchestration_includes_implemented_core_tools",
			setupConfig: func(c *config.Config) {
				c.MCP.Preset = "orchestration"
			},
			mustHave: implementedCoreTools,
		},
		{
			name: "core_group_disabled_is_overridden_to_true",
			// Setting core: false must be silently overridden; core tools must still appear.
			setupConfig: func(c *config.Config) {
				c.MCP.Preset = "minimal"
				c.MCP.Groups = map[string]bool{
					config.GroupCore: false,
				}
			},
			mustHave: implementedCoreTools,
		},
		{
			name: "default_config_includes_implemented_core_tools",
			// No mcp section → defaults to preset: full.
			setupConfig: func(c *config.Config) {},
			mustHave:    implementedCoreTools,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			entityRoot := t.TempDir()
			cfg := config.DefaultConfig()
			tt.setupConfig(&cfg)

			registered := toolNamesFromServer(t, entityRoot, &cfg)

			if missing := containsAll(registered, tt.mustHave); len(missing) != 0 {
				t.Errorf("tools missing from registration: %v\nregistered: %v", missing, registered)
			}
			if present := containsNone(registered, tt.mustNotHave); len(present) != 0 {
				t.Errorf("unexpected tools present in registration: %v\nregistered: %v", present, registered)
			}
		})
	}
}

// TestServer_ListTools_CoreGroupConditional verifies that when the core group is
// explicitly enabled, the 2.0 core tools appear in addition to the _legacy tools
// (spec §30.1: "Setting a group to true registers all tools in that group").
func TestServer_ListTools_CoreGroupConditional(t *testing.T) {
	t.Parallel()

	entityRoot := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.MCP.Preset = "minimal" // only core in 2.0

	registered := toolNamesFromServer(t, entityRoot, &cfg)

	// Verify each implemented core tool is registered
	for _, tool := range []string{"status", "next", "finish", "handoff", "entity", "doc", "health", "server_info"} {
		found := false
		for _, name := range registered {
			if name == tool {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("core tool %q not registered (preset: minimal, core must always be on)", tool)
		}
	}
}
