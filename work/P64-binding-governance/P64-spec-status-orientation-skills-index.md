# Specification: Inline skills index in status() orientation block

## Overview

`status()` currently points agents at `.agents/skills/kanbanzai-getting-started/SKILL.md` as a path without surfacing skill content. Agents (orchestrators and casual chat) do not proactively discover or follow skills.

## Scope

Single-file change to `internal/mcp/status_tool.go` in the `synthesiseProject` function.

## Functional Requirements

- [ ] FR-1: The orientation block in `status()` output SHALL include a list of available skills with name and one-line summary for each.
- [ ] FR-2: When no task is claimed, the orientation block SHALL suggest the `kanbanzai-documents` skill.
- [ ] FR-3: When an active feature exists, the orientation block SHALL suggest the `kanbanzai-workflow` skill.
- [ ] FR-4: Skills SHALL be read dynamically from `.agents/skills/` at render time (no static index to maintain).

## Non-Functional Requirements

- [ ] NFR-1: The orientation block SHALL NOT exceed a reasonable token budget (~500 tokens).

## Acceptance Criteria

- [ ] AC-1: `status()` output includes an inline skills list with name + summary.
- [ ] AC-2: `status()` output includes a context-aware skill suggestion.
- [ ] AC-3: Existing tests continue to pass.
