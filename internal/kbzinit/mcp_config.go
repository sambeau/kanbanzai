package kbzinit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// mcpVersion is the current schema version for both .mcp.json and .zed/settings.json.
const mcpVersion = 2

// managedBlock is the _managed metadata written to both MCP config files.
type managedBlock struct {
	Tool    string `json:"tool"`
	Version int    `json:"version"`
}

// mcpConfig is the structure written to .mcp.json
type mcpConfig struct {
	Managed    managedBlock         `json:"_managed"`
	MCPServers map[string]mcpServer `json:"mcpServers"`
}

type mcpServer struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// zedConfig is the structure written to .zed/settings.json.
// It does not include a _managed block — Zed validates settings.json against
// its own schema and rejects unknown top-level properties.
type zedConfig struct {
	ContextServers map[string]zedContextServer `json:"context_servers"`
	Agent          *zedAgentConfig             `json:"agent,omitempty"`
}

type zedContextServer struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

type zedAgentConfig struct {
	ToolPermissions *zedToolPermissions `json:"tool_permissions,omitempty"`
}

type zedToolPermissions struct {
	Tools map[string]zedToolRule `json:"tools,omitempty"`
}

type zedToolRule struct {
	Default string `json:"default"`
}

// kanbanzaiToolPermissions defines per-tool defaults for all 22 kanbanzai MCP tools.
// All workflow tools are pre-approved (allow) to eliminate friction for normal agent
// operations. Tools with external visibility or irreversible filesystem effects
// (merge, pr, cleanup) require explicit confirmation.
var kanbanzaiToolPermissions = map[string]zedToolRule{
	// Workflow operations — auto-approve.
	"mcp:kanbanzai:status":      {Default: "allow"},
	"mcp:kanbanzai:next":        {Default: "allow"},
	"mcp:kanbanzai:entity":      {Default: "allow"},
	"mcp:kanbanzai:doc":         {Default: "allow"},
	"mcp:kanbanzai:finish":      {Default: "allow"},
	"mcp:kanbanzai:knowledge":   {Default: "allow"},
	"mcp:kanbanzai:health":      {Default: "allow"},
	"mcp:kanbanzai:server_info": {Default: "allow"},
	"mcp:kanbanzai:profile":     {Default: "allow"},
	"mcp:kanbanzai:handoff":     {Default: "allow"},
	"mcp:kanbanzai:retro":       {Default: "allow"},
	"mcp:kanbanzai:estimate":    {Default: "allow"},
	"mcp:kanbanzai:decompose":   {Default: "allow"},
	"mcp:kanbanzai:doc_intel":   {Default: "allow"},
	"mcp:kanbanzai:incident":    {Default: "allow"},
	"mcp:kanbanzai:checkpoint":  {Default: "allow"},
	"mcp:kanbanzai:branch":      {Default: "allow"},
	"mcp:kanbanzai:conflict":    {Default: "allow"},
	"mcp:kanbanzai:worktree":    {Default: "allow"},
	// External visibility or irreversible filesystem effects — require confirmation.
	"mcp:kanbanzai:merge":   {Default: "confirm"},
	"mcp:kanbanzai:pr":      {Default: "confirm"},
	"mcp:kanbanzai:cleanup": {Default: "confirm"},
}

// writeMCPConfig writes .mcp.json to baseDir applying version-aware conflict logic.
// If an unmanaged file exists, it skips and warns. If a managed older version exists,
// it overwrites. If at current version, it no-ops.
func (i *Initializer) writeMCPConfig(baseDir string) error {
	destPath := filepath.Join(baseDir, ".mcp.json")

	content := mcpConfig{
		Managed: managedBlock{Tool: "kanbanzai", Version: mcpVersion},
		MCPServers: map[string]mcpServer{
			"kanbanzai": {Command: "kbz", Args: []string{"serve"}},
		},
	}

	data, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal .mcp.json: %w", err)
	}
	data = append(data, '\n')

	return i.writeJSONConfig(destPath, ".mcp.json", data, "add the kanbanzai server entry to it manually. See docs/getting-started.md for the snippet")
}

// writeZedConfig writes .zed/settings.json to baseDir.
//
// When createIfAbsent is true (first-time init), the .zed/ directory is created
// if it does not already exist. When false (re-running on an already-initialised
// project), a missing .zed/ directory signals the project does not use Zed and
// the function returns without writing anything.
//
// Unlike .mcp.json, the Zed settings file does not include a _managed block —
// Zed validates settings.json against its own schema and rejects unknown
// top-level properties. Managed state is inferred from the file content instead:
//   - context_servers.kanbanzai present → already configured, no-op
//   - _managed.tool == "kanbanzai" present → old format written by an earlier
//     version of kanbanzai; rewrite without the _managed block (migration)
//   - neither present → user's own settings file, warn and skip
func (i *Initializer) writeZedConfig(baseDir string, createIfAbsent bool) error {
	zedDir := filepath.Join(baseDir, ".zed")
	if _, err := os.Stat(zedDir); os.IsNotExist(err) {
		if !createIfAbsent {
			return nil
		}
		if err := os.MkdirAll(zedDir, 0o755); err != nil {
			return fmt.Errorf("create .zed/: %w", err)
		}
	}

	destPath := filepath.Join(zedDir, "settings.json")

	content := zedConfig{
		ContextServers: map[string]zedContextServer{
			"kanbanzai": {Command: "kbz", Args: []string{"serve"}},
		},
		Agent: &zedAgentConfig{
			ToolPermissions: &zedToolPermissions{
				Tools: kanbanzaiToolPermissions,
			},
		},
	}
	data, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal .zed/settings.json: %w", err)
	}
	data = append(data, '\n')

	existing, readErr := os.ReadFile(destPath)
	if readErr != nil {
		if !os.IsNotExist(readErr) {
			return fmt.Errorf("read .zed/settings.json: %w", readErr)
		}
		// File does not exist — create it.
		if err := os.WriteFile(destPath, data, 0o644); err != nil {
			return fmt.Errorf("write .zed/settings.json: %w", err)
		}
		fmt.Fprintln(i.stdout, "Created .zed/settings.json")
		return nil
	}

	// File exists — parse it.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(existing, &raw); err != nil {
		fmt.Fprintf(i.stdout, "Warning: .zed/settings.json exists but could not be parsed. To enable the MCP server in Zed, add the kanbanzai context server entry manually. See docs/getting-started.md for the snippet.\n")
		return nil
	}

	// Migration: older kanbanzai versions wrote a _managed block that Zed's schema
	// rejects. If we find our marker, rewrite the file without it.
	if managedRaw, ok := raw["_managed"]; ok {
		var managed managedBlock
		if err := json.Unmarshal(managedRaw, &managed); err == nil && managed.Tool == "kanbanzai" {
			if err := os.WriteFile(destPath, data, 0o644); err != nil {
				return fmt.Errorf("update .zed/settings.json: %w", err)
			}
			fmt.Fprintln(i.stdout, "Updated .zed/settings.json")
			return nil
		}
	}

	// Check whether context_servers.kanbanzai is already present.
	if csRaw, ok := raw["context_servers"]; ok {
		var cs map[string]json.RawMessage
		if err := json.Unmarshal(csRaw, &cs); err == nil {
			if _, ok := cs["kanbanzai"]; ok {
				// Server is registered. Check whether tool_permissions are already
				// present. If not (e.g. a file written by an older kanbanzai that
				// predates this feature), and the file has no user-added "agent"
				// block, migrate it to the full config.
				if _, hasAgent := raw["agent"]; !hasAgent {
					if err := os.WriteFile(destPath, data, 0o644); err != nil {
						return fmt.Errorf("update .zed/settings.json: %w", err)
					}
					fmt.Fprintln(i.stdout, "Updated .zed/settings.json")
					return nil
				}
				// Agent block already present (user-customised) — no-op.
				return nil
			}
		}
	}

	// File exists but has no kanbanzai entry — this is the user's own Zed settings.
	fmt.Fprintf(i.stdout, "Warning: .zed/settings.json exists and does not include a kanbanzai server entry. To enable the MCP server in Zed, add the kanbanzai context server entry manually. See docs/getting-started.md for the snippet.\n")
	return nil
}

// detectStaleMCPConfigs inspects managed .mcp.json and .zed/settings.json for a stale
// "command": "kanbanzai" value (from before the binary rename to "kbz") and prints
// a human-readable warning. It is called by both kbz init and kbz init --update-skills
// so that projects with stale configs are never silently left unreported.
func (i *Initializer) detectStaleMCPConfigs(baseDir string) {
	staleMsg := "Warning: %s references \"kanbanzai\" which is no longer installed.\n  Run: kbz init to update editor configuration.\n"

	// Check .mcp.json
	mcpPath := filepath.Join(baseDir, ".mcp.json")
	if data, err := os.ReadFile(mcpPath); err == nil {
		var raw map[string]json.RawMessage
		if json.Unmarshal(data, &raw) == nil {
			if managedRaw, ok := raw["_managed"]; ok {
				var managed managedBlock
				if json.Unmarshal(managedRaw, &managed) == nil && managed.Tool == "kanbanzai" {
					if serversRaw, ok := raw["mcpServers"]; ok {
						var servers map[string]mcpServer
						if json.Unmarshal(serversRaw, &servers) == nil {
							if srv, ok := servers["kanbanzai"]; ok && srv.Command == "kanbanzai" {
								fmt.Fprintf(i.stdout, staleMsg, ".mcp.json")
							}
						}
					}
				}
			}
		}
	}

	// Check .zed/settings.json
	zedPath := filepath.Join(baseDir, ".zed", "settings.json")
	if data, err := os.ReadFile(zedPath); err == nil {
		var raw map[string]json.RawMessage
		if json.Unmarshal(data, &raw) == nil {
			if csRaw, ok := raw["context_servers"]; ok {
				var cs map[string]zedContextServer
				if json.Unmarshal(csRaw, &cs) == nil {
					if srv, ok := cs["kanbanzai"]; ok && srv.Command == "kanbanzai" {
						fmt.Fprintf(i.stdout, staleMsg, ".zed/settings.json")
					}
				}
			}
		}
	}
}

// writeJSONConfig applies version-aware create/update/skip logic to a JSON config file.
// It reads any existing file, checks the _managed marker, and writes, skips, or warns accordingly.
func (i *Initializer) writeJSONConfig(destPath, displayName string, newContent []byte, warningInstruction string) error {
	existing, readErr := os.ReadFile(destPath)
	if readErr != nil {
		if !os.IsNotExist(readErr) {
			return fmt.Errorf("read %s: %w", displayName, readErr)
		}
		// File does not exist — create it.
		if err := os.WriteFile(destPath, newContent, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", displayName, err)
		}
		fmt.Fprintf(i.stdout, "Created %s\n", displayName)
		return nil
	}

	// File exists — parse it to check managed marker.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(existing, &raw); err != nil {
		// Can't parse — treat as unmanaged.
		fmt.Fprintf(i.stdout, "Warning: %s exists but could not be parsed. To enable automatic MCP connection, %s\n", displayName, warningInstruction)
		return nil
	}

	managedRaw, ok := raw["_managed"]
	if !ok {
		// No _managed key — unmanaged file, skip with warning.
		fmt.Fprintf(i.stdout, "Warning: %s exists and is not managed by kanbanzai. To enable automatic MCP connection, %s\n", displayName, warningInstruction)
		return nil
	}

	var managed managedBlock
	if err := json.Unmarshal(managedRaw, &managed); err != nil || managed.Tool != "kanbanzai" {
		// Managed by someone else — skip with warning.
		fmt.Fprintf(i.stdout, "Warning: %s exists and is not managed by kanbanzai. To enable automatic MCP connection, %s\n", displayName, warningInstruction)
		return nil
	}

	if managed.Version >= mcpVersion {
		// At current version — no-op.
		return nil
	}

	// Older managed version — overwrite.
	if err := os.WriteFile(destPath, newContent, 0o644); err != nil {
		return fmt.Errorf("update %s: %w", displayName, err)
	}
	fmt.Fprintf(i.stdout, "Updated %s\n", displayName)
	return nil
}
