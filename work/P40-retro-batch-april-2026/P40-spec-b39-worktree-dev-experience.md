# Specification: Fix Worktree Development Experience

| Field  | Value                         |
|--------|-------------------------------|
| Date   | 2026-04-30                    |
| Status | approved |
| Author | Spec Author                   |
| Batch  | B39-fix-worktree-dev-experience |
| Design | P40-retro-batch-april-2026/design-p40-design-retro-batch-improvements |

---

## Problem Statement

This specification implements Workstream A of the design described in
`work/P40-retro-batch-april-2026/P40-design-retro-batch-improvements.md`
(P40-retro-batch-april-2026/design-p40-design-retro-batch-improvements, approved).

Five separate retrospective reports (B38 implementation, retro synthesis, P38
implementation, B36 review fixes, and P37 kbz move) document that worktree file
editing is broken and undiscoverable. Agents developing in feature worktrees
cannot use `edit_file` — it silently writes to the main repo root. The
`write_file` MCP tool supports worktrees via its `entity_id` parameter, but this
capability is undocumented in the `implement-task` skill. The workaround patterns
(`python3 -c` with triple-escaping, heredocs in `sh`) are fragile and error-prone.

This specification addresses two changes from the design's Workstream A:

1. **A1 — Document `write_file(entity_id)` as primary worktree pattern:** Update
   the `implement-task` skill to make `write_file(entity_id: ...)` the
   recommended primary pattern and add it to the developing-stage tool subset.
2. **A2 — Make `edit_file` worktree-aware:** Add an optional `entity_id`
   parameter to `edit_file` so it writes to the correct worktree.

A3 (heredoc support in the terminal tool) is superseded by A1: once
`write_file(entity_id)` is the documented primary pattern, agents no longer
need to reach for heredocs as a worktree file-writing mechanism. Fixing a
fallback shell that should no longer be used adds maintenance cost with no
ongoing benefit.

**Scope inclusion:** The `implement-task/SKILL.md` skill file and the `edit_file`
MCP tool implementation (parameter addition, worktree path resolution, tests).

**Scope exclusion:** The `write_file` tool itself (it already works correctly).
Other skill files. The context assembly pipeline. The terminal tool's shell
configuration. The `edit_file` multi-edit atomicity fix (that is B3 in Workstream
B). The `edit_file` stale-buffer issue (that is a separate investigation).

---

## Requirements

### Functional Requirements

- **REQ-001:** The `implement-task` skill (`implement-task/SKILL.md`) MUST
  document `write_file(entity_id: ...)` as the primary recommended pattern for
  creating and modifying files in worktrees, replacing all references to
  `python3 -c` and heredoc workarounds as the recommended approach.

- **REQ-002:** The `implement-task` skill MUST remove or relegate to fallback
  status any guidance that recommends `python3 -c` or shell heredocs for
  worktree file creation when `write_file` is available.

- **REQ-003:** `write_file` MUST be added to the default tool subset for the
  `developing` stage in the stage bindings or role configuration, so sub-agents
  discover it without explicit instruction from the orchestrator.

- **REQ-004:** The `edit_file` tool MUST accept an optional `entity_id`
  parameter. When provided, the tool MUST resolve the entity's worktree path
  and apply all edits relative to that worktree root rather than the main
  repository root.

- **REQ-005:** When `entity_id` is omitted from `edit_file`, behaviour MUST be
  unchanged — writes go to the main repository root as they do today.

- **REQ-006:** When `entity_id` is provided to `edit_file` but no active
  worktree exists for that entity, the tool MUST return a clear error message
  stating that no worktree was found for the given entity.

- **REQ-007:** The `edit_file` worktree resolution logic MUST use the same
  worktree path resolution mechanism already implemented for `write_file`
  (the `entity_id` to worktree path lookup), not a separate implementation.

### Non-Functional Requirements

- **REQ-NF-001:** Adding `entity_id` to `edit_file` MUST NOT change the
  behaviour of any existing `edit_file` call that does not include the new
  parameter. Backward compatibility is required.

- **REQ-NF-002:** The `edit_file` worktree resolution MUST complete in O(1)
  time (a single cache or store lookup) — it MUST NOT introduce a filesystem
  scan or linear search.

- **REQ-NF-003:** The skill file edits for REQ-001 and REQ-002 MUST NOT change
  any procedure steps, anti-patterns, or vocabulary definitions in
  `implement-task/SKILL.md` other than the file-writing guidance.

---

## Constraints

- The `write_file` tool's `entity_id` parameter and worktree resolution are
  already implemented and tested (added in P21). This specification does NOT
  change `write_file` behaviour.
- The `edit_file` tool's existing parameter surface (path, edits, mode,
  display_description) is preserved. `entity_id` is additive.
- The `implement-task` skill's structure (sections, headings, checklist format)
  MUST be preserved. Only the content of file-writing guidance sections may
  change.
- All changes are to existing files; no new files are created except test files
  for REQ-004 through REQ-007.
- This specification does NOT cover the multi-edit atomicity fix, the
  stale-buffer issue, or any other `edit_file` reliability improvements
  (those are Workstream B).

---

## Acceptance Criteria

- **AC-001 (REQ-001):** Given an agent reading `implement-task/SKILL.md` for
  file-writing guidance, when they reach the section describing how to write
  files in worktrees, then the primary recommendation is
  `write_file(entity_id: "...", path: "...", content: "...")` with an
  explanation of the `entity_id` parameter.

- **AC-002 (REQ-002):** Given the `implement-task/SKILL.md` file, when
  searched for `python3 -c` or heredoc patterns (`<<`), then any remaining
  references are explicitly marked as fallback approaches, not primary
  recommendations.

- **AC-003 (REQ-003):** Given a sub-agent spawned for the `developing` stage,
  when the agent's available tool set is assembled, then `write_file` is
  included in the default tool subset.

- **AC-004 (REQ-004):** Given a feature with an active worktree at
  `.worktrees/FEAT-01EXAMPLE/path/to/file.go`, when `edit_file` is called with
  `entity_id: "FEAT-01EXAMPLE"` and `path: "path/to/file.go"`, then the edit
  is applied to `.worktrees/FEAT-01EXAMPLE/path/to/file.go`, not to the main
  repo root.

- **AC-005 (REQ-005):** Given an `edit_file` call with no `entity_id`
  parameter, when the edit is applied, then the file path is resolved relative
  to the main repository root, identical to current behaviour.

- **AC-006 (REQ-006):** Given an `edit_file` call with `entity_id:
  "FEAT-01NONEXISTENT"` where no worktree exists for that entity, when the
  tool executes, then it returns an error message containing "no worktree
  found" and the entity ID.

- **AC-007 (REQ-004):** Given a feature with an active worktree, when
  `edit_file` is called with `entity_id` and a multi-edit payload, then all
  edits are applied within the worktree directory.

- **AC-008 (REQ-007):** Given the `edit_file` tool's worktree resolution
  implementation, when the resolved path is traced, then it uses the same
  code path as `write_file` (the existing worktree record lookup), not a
  duplicated or divergent implementation.

- **AC-009 (REQ-NF-001):** Given the existing test suite for `edit_file`, when
  run against the modified implementation, then all existing tests pass
  without modification.

- **AC-010 (REQ-NF-002):** Given the `edit_file` tool with `entity_id`
  provided, when the worktree resolution is profiled, then it completes in a
  single cache or store lookup without scanning the filesystem or iterating
  over worktree records.

- **AC-011 (REQ-NF-003):** Given a diff of `implement-task/SKILL.md` before
  and after the changes, when the diff is reviewed, then no procedure steps,
  anti-pattern names, or vocabulary definitions are changed — only the
  file-writing guidance content is modified.

---

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Inspection | Review `implement-task/SKILL.md` to confirm `write_file(entity_id: ...)` is the primary recommendation in the worktree file-writing section |
| AC-002 | Inspection | Grep `implement-task/SKILL.md` for `python3 -c` and `<<`; confirm any remaining references are marked as fallback |
| AC-003 | Inspection | Verify `write_file` appears in the developing-stage tool subset in the stage bindings or role configuration |
| AC-004 | Test | Automated test: create a worktree, call `edit_file` with `entity_id`, verify the file was modified in the worktree path not the main repo |
| AC-005 | Test | Automated test: call `edit_file` without `entity_id`, verify behaviour is identical to current (writes to main repo root) |
| AC-006 | Test | Automated test: call `edit_file` with a non-existent `entity_id`, verify error message contains "no worktree found" |
| AC-007 | Test | Automated test: create a worktree, call `edit_file` with `entity_id` and a multi-edit payload, verify all edits applied in worktree |
| AC-008 | Inspection | Code review: trace the worktree resolution code path in `edit_file` and confirm it calls the same function as `write_file` |
| AC-009 | Test | Run existing `edit_file` test suite; all tests pass without modification |
| AC-010 | Inspection | Code review: confirm worktree resolution is a single lookup (no loops, no filesystem walks) |
| AC-011 | Inspection | Diff `implement-task/SKILL.md` before/after; confirm only file-writing guidance changed |
