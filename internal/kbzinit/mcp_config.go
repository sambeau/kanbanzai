package kbzinit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// mcpVersion is the current schema version for both .mcp.json and .zed/settings.json.
const mcpVersion = 1

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

// zedConfig is the structure written to .zed/settings.json
type zedConfig struct {
	Managed        managedBlock                `json:"_managed"`
	ContextServers map[string]zedContextServer `json:"context_servers"`
}

type zedContextServer struct {
	Command zedCommand `json:"command"`
}

type zedCommand struct {
	Path string   `json:"path"`
	Args []string `json:"args"`
}

// writeMCPConfig writes .mcp.json to baseDir applying version-aware conflict logic.
// If an unmanaged file exists, it skips and warns. If a managed older version exists,
// it overwrites. If at current version, it no-ops.
func (i *Initializer) writeMCPConfig(baseDir string) error {
	destPath := filepath.Join(baseDir, ".mcp.json")

	content := mcpConfig{
		Managed: managedBlock{Tool: "kanbanzai", Version: mcpVersion},
		MCPServers: map[string]mcpServer{
			"kanbanzai": {Command: "kanbanzai", Args: []string{"serve"}},
		},
	}

	data, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal .mcp.json: %w", err)
	}
	data = append(data, '\n')

	return i.writeJSONConfig(destPath, ".mcp.json", data, "add the kanbanzai server entry to it manually. See docs/getting-started.md for the snippet")
}

// writeZedConfig writes .zed/settings.json to baseDir, applying the same
// version-aware conflict logic as writeMCPConfig.
//
// When createIfAbsent is true (new projects), the .zed/ directory is created
// if it does not already exist. When false (existing projects), a missing .zed/
// directory is treated as a signal that the project does not use Zed, and the
// function returns without writing anything.
func (i *Initializer) writeZedConfig(baseDir string, createIfAbsent bool) error {
	zedDir := filepath.Join(baseDir, ".zed")
	if _, err := os.Stat(zedDir); os.IsNotExist(err) {
		if !createIfAbsent {
			// No .zed/ directory and we're on an existing project — silently skip.
			return nil
		}
		// New project: create .zed/ so settings.json can be written.
		if err := os.MkdirAll(zedDir, 0o755); err != nil {
			return fmt.Errorf("create .zed/: %w", err)
		}
	}

	destPath := filepath.Join(zedDir, "settings.json")

	content := zedConfig{
		Managed: managedBlock{Tool: "kanbanzai", Version: mcpVersion},
		ContextServers: map[string]zedContextServer{
			"kanbanzai": {
				Command: zedCommand{Path: "kanbanzai", Args: []string{"serve"}},
			},
		},
	}

	data, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal .zed/settings.json: %w", err)
	}
	data = append(data, '\n')

	return i.writeJSONConfig(destPath, ".zed/settings.json", data, "add the kanbanzai context server entry manually. See docs/getting-started.md for the snippet")
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
