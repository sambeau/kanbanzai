# Design: Worktree File-Writing Tooling

**Feature:** FEAT-01KR003J3ATM2 — Worktree file-writing tooling
**Tier:** retro_fix
**Retro Signals:** KE-01KN5CXMBWSXE, KE-01KPW39640YG9, KE-01KQ7TKTJ7YVB, KE-01KPPYEPA1XHZ

## Overview

Four separate retro signals describe the same root problem: writing files inside worktrees is unnecessarily hard. Agents default to terminal-based workarounds (python3 -c, heredoc) when `kanbanzai_edit_file` and `write_file` MCP tools already accept `entity_id` for worktree targeting.

## Design

### Change

Update `.kbz/skills/implement-task/SKILL.md` and `.github/copilot-instructions.md` to prefer `kanbanzai_edit_file` / `write_file` with `entity_id` as the primary method for worktree file writing.

### Locations

1. `.kbz/skills/implement-task/SKILL.md` — add to the "File Operations" or equivalent section
2. `.github/copilot-instructions.md` — add a note in the tool usage section

### What to change

- Document that `kanbanzai_edit_file(entity_id: "FEAT-...", mode: "edit", ...)` is the primary way to write files in a worktree
- Document that `write_file(entity_id: "FEAT-...", path: "...", content: "...")` is the primary way to create new files in a worktree
- Deprecate terminal-based workarounds (python3 -c, heredoc) for worktree file writing

## Goals and Non-Goals

### Goals

- Implement the change described in the Design section
- Keep scope minimal — no unrelated refactoring

### Non-Goals

- Does not change the underlying tool behavior (tools already support entity_id)
- Does not modify terminal tool behavior

## Alternatives Considered

### Do nothing

Leave the friction in place. Rejected: the retro signals show this wastes agent cycles and causes state drift.

### Automated enforcement

Add tooling to automatically enforce the desired behavior. Rejected: over-engineered for the current scale; documentation is sufficient.

## Dependencies

None. Pure documentation change.
