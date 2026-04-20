package kbzinit

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// agentsMDVersion is the current schema version for both AGENTS.md and
// .github/copilot-instructions.md. Increment when the generated content
// changes in a way that warrants overwriting existing managed files.
const agentsMDVersion = 2

// agentsMDMarkerPrefix is the HTML comment marker written on line 1 of
// managed markdown files. It is invisible to agents reading the file as
// Markdown but machine-parseable by the version-aware conflict logic.
const agentsMDMarkerPrefix = "<!-- kanbanzai-managed: v"

// agentsMDMarkerSuffix closes the HTML comment marker.
const agentsMDMarkerSuffix = " -->"

// agentsMDContent is the generated content for AGENTS.md.
// Must start with the managed marker comment on line 1.
// Must not exceed 50 lines.
const agentsMDContent = `<!-- kanbanzai-managed: v2 -->

# Agent Instructions

This project uses **Kanbanzai** for workflow management via MCP tools. All
workflow state is managed through the kanbanzai MCP server.

## Before You Do Anything

1. Call ` + "`status`" + ` to see the current project state.
2. Call ` + "`next`" + ` to see the work queue.
3. Read ` + "`.agents/skills/kanbanzai-getting-started/SKILL.md`" + ` for full orientation.

## Rules

- **Use kanbanzai MCP tools** (` + "`status`, `next`, `entity`, `doc`, `finish`" + `) for all
  workflow operations. Do not create or modify entities or documents by writing
  files directly — this bypasses lifecycle enforcement and health checks.
- **Follow the stage gates**: Planning → Design → Features → Specification →
  Dev plan → Implementation. Skipping forward is not allowed. See
  ` + "`.agents/skills/kanbanzai-workflow/SKILL.md`" + `.
- **Human approval is required** at stage gates. When in doubt, stop and ask.

## Skills Reference

| Skill | When to read |
|---|---|
| ` + "`kanbanzai-getting-started`" + ` | Start of every session |
| ` + "`kanbanzai-workflow`" + ` | Before any stage transition or entity creation |
| ` + "`kanbanzai-design`" + ` | During design work |
| ` + "`kanbanzai-specification`" + ` | During specification work |
| ` + "`kanbanzai-documents`" + ` | When creating or registering any document |
| ` + "`kanbanzai-agents`" + ` | During implementation: task dispatch, commits, knowledge |
| ` + "`kanbanzai-planning`" + ` | During planning conversations |
| ` + "`kanbanzai-review`" + ` | When reviewing completed features |
| ` + "`kanbanzai-plan-review`" + ` | When reviewing plans |

## Optional: Code Graph Integration

If your project uses codebase-memory-mcp, set graph_project in
.kbz/local.yaml once per machine. The worktree tool uses it automatically
when creating worktrees.

    codebase_memory:
      graph_project: YOUR-GRAPH-PROJECT-NAME

Derive the name: take the repo absolute path, drop the leading slash, and
replace remaining slashes with hyphens. For example /Users/alice/Dev/myrepo
becomes Users-alice-Dev-myrepo.
`

// copilotInstructionsContent is the generated content for
// .github/copilot-instructions.md.
// Must start with the managed marker comment on line 1.
// Must not exceed 25 lines.
const copilotInstructionsContent = `<!-- kanbanzai-managed: v2 -->

# Copilot Instructions

This project uses **Kanbanzai** for workflow management. Read ` + "`AGENTS.md`" + ` in
the project root before doing any work — it contains the essential rules and
skill pointers for this project.

## Quick Reference

1. Call ` + "`status`" + ` to see the current project state.
2. Call ` + "`next`" + ` to see the work queue.
3. Read ` + "`.agents/skills/kanbanzai-getting-started/SKILL.md`" + ` at session start.
4. Use kanbanzai MCP tools for all workflow operations — do not write
   ` + "`work/`" + ` documents or ` + "`.kbz/state/`" + ` entities directly.
`

// writeAgentsMD writes AGENTS.md to baseDir applying version-aware conflict logic:
//   - No file → create.
//   - File exists, managed marker present, version < current → overwrite.
//   - File exists, managed marker present, version >= current → no-op.
//   - File exists, no managed marker → skip and print warning.
func (i *Initializer) writeAgentsMD(baseDir string) error {
	destPath := filepath.Join(baseDir, "AGENTS.md")
	return i.writeMarkdownConfig(destPath, "AGENTS.md", agentsMDContent,
		"add the kanbanzai workflow section manually. See docs/getting-started.md for the snippet.")
}

// writeCopilotInstructions writes .github/copilot-instructions.md to baseDir,
// creating the .github/ directory if it does not exist.
// Applies the same version-aware conflict logic as writeAgentsMD.
func (i *Initializer) writeCopilotInstructions(baseDir string) error {
	githubDir := filepath.Join(baseDir, ".github")
	if _, err := os.Stat(githubDir); os.IsNotExist(err) {
		if err := os.MkdirAll(githubDir, 0o755); err != nil {
			return fmt.Errorf("create .github/: %w", err)
		}
	}
	destPath := filepath.Join(githubDir, "copilot-instructions.md")
	return i.writeMarkdownConfig(destPath, ".github/copilot-instructions.md",
		copilotInstructionsContent,
		"add the kanbanzai instructions section manually. See docs/getting-started.md for the snippet.")
}

// writeMarkdownConfig applies version-aware create/update/skip logic to a
// Markdown config file using the HTML comment managed marker on line 1.
func (i *Initializer) writeMarkdownConfig(destPath, displayName, newContent, warningInstruction string) error {
	existing, readErr := os.ReadFile(destPath)
	if readErr != nil {
		if !os.IsNotExist(readErr) {
			return fmt.Errorf("read %s: %w", displayName, readErr)
		}
		// File does not exist — create it.
		if err := os.WriteFile(destPath, []byte(newContent), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", displayName, err)
		}
		fmt.Fprintf(i.stdout, "Created %s\n", displayName)
		return nil
	}

	// File exists — check the managed marker on line 1.
	existingVersion, managed, err := readMarkdownManagedVersion(existing)
	if err != nil {
		// Can't parse first line — treat as unmanaged.
		fmt.Fprintf(i.stdout, "Warning: %s exists but could not be read. To configure it for kanbanzai, %s\n",
			displayName, warningInstruction)
		return nil
	}
	if !managed {
		// No managed marker — user-owned file, leave it alone.
		fmt.Fprintf(i.stdout, "Warning: %s exists and is not managed by kanbanzai. To configure it for kanbanzai, %s\n",
			displayName, warningInstruction)
		return nil
	}
	if existingVersion >= agentsMDVersion {
		// Already at current version — no-op.
		return nil
	}

	// Older managed version — overwrite.
	if err := os.WriteFile(destPath, []byte(newContent), 0o644); err != nil {
		return fmt.Errorf("update %s: %w", displayName, err)
	}
	fmt.Fprintf(i.stdout, "Updated %s\n", displayName)
	return nil
}

// readMarkdownManagedVersion reads the first line of data and extracts the
// kanbanzai managed marker version. It returns (version, true, nil) when the
// marker is found and valid, (0, false, nil) when the first line contains no
// marker, and (0, false, err) only on genuine I/O or parse failures.
//
// The expected first-line format is:
//
//	<!-- kanbanzai-managed: vN -->
//
// where N is a positive integer.
func readMarkdownManagedVersion(data []byte) (version int, managed bool, err error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	if !scanner.Scan() {
		if scanErr := scanner.Err(); scanErr != nil {
			return 0, false, scanErr
		}
		// Empty file — no marker.
		return 0, false, nil
	}
	line := strings.TrimSpace(scanner.Text())
	if !strings.HasPrefix(line, agentsMDMarkerPrefix) {
		return 0, false, nil
	}
	// Strip prefix and suffix to extract the version number.
	inner := strings.TrimPrefix(line, agentsMDMarkerPrefix)
	inner = strings.TrimSuffix(inner, agentsMDMarkerSuffix)
	inner = strings.TrimSpace(inner)
	v, parseErr := strconv.Atoi(inner)
	if parseErr != nil {
		return 0, false, nil // malformed marker — treat as unmanaged
	}
	return v, true, nil
}
