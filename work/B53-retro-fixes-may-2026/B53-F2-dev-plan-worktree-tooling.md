# Dev-Plan: Worktree File-Writing Tooling

**Feature:** FEAT-01KR003J3ATM2
**Tier:** retro_fix
**Spec:** FEAT-01KR003J3ATM2/spec-b53-f2-spec-worktree-tooling

## Overview

Two-file change: update implement-task skill and copilot instructions to prefer kanbanzai_edit_file/write_file for worktree file operations.

## Task Breakdown

### T1: Update implement-task/SKILL.md

- **File:** `.kbz/skills/implement-task/SKILL.md`
- **Description:** Add guidance for worktree file writing using `kanbanzai_edit_file(entity_id: ..., mode: ..., ...)` and `write_file(entity_id: ..., path: ..., content: ...)`. Deprecate terminal-based workarounds.
- **Deliverable:** Updated skill file.

### T2: Update copilot-instructions.md

- **File:** `.github/copilot-instructions.md`
- **Description:** Add note that `kanbanzai_edit_file` and `write_file` accept `entity_id` for worktree targeting.
- **Deliverable:** Updated instructions file.

## Dependency Graph

```
T1 (no dependencies)
T2 (no dependencies)
```

T1 and T2 touch different files — dispatch in parallel.

## Interface Contracts

No interfaces changed. Pure documentation.

## Traceability Matrix

| Task | FR-001 | FR-002 | FR-003 | FR-004 | AC-001 | AC-002 | AC-003 | AC-004 |
|------|--------|--------|--------|--------|--------|--------|--------|--------|
| T1   | x      | x      | x      |        | x      | x      | x      |        |
| T2   |        |        |        | x      |        |        |        | x      |
