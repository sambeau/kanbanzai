# Design: Inline skills index in status() orientation block

## Overview

Modify `synthesiseProject` in `internal/mcp/status_tool.go` to surface skill summaries inline in the orientation block of `status()` output.

## Design

Replace the current static reference to `.agents/skills/kanbanzai-getting-started/SKILL.md` with a dynamically-generated skills index that includes:
1. A list of available skills (name + one-line summary)
2. A context-aware suggestion: if no claimed task → suggest `kanbanzai-documents`; if active feature → suggest `kanbanzai-workflow`

Skills are read from the `.agents/skills/` directory at render time. The change is a single-file edit to `internal/mcp/status_tool.go` with corresponding test additions.

## Alternatives Considered

- **Separate MCP tool**: Add a `list_skills` tool. Rejected — adds friction; the orientation block is the natural discovery point.
- **Static index file**: Maintain a hand-written index. Rejected — goes stale; dynamic discovery is more reliable.

## Dependencies

None. Self-contained.
