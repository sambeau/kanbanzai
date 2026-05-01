# Dev-Plan: Make edit_file Worktree-Aware (B39-F2)

| Field  | Value                         |
|--------|-------------------------------|
| Date   | 2026-04-30                    |
| Status | approved |
| Author | Architect                     |
| Feature | FEAT-01KQG1XWE9ABP           |
| Spec   | B39-fix-worktree-dev-experience/spec-p40-spec-b39-worktree-dev-experience |

---

## Overview

Add an optional `entity_id` parameter to the `edit_file` MCP tool. When
provided, resolve the entity's worktree path using the same mechanism already
implemented for `write_file` (P21) and apply all edits relative to the worktree
root. When omitted, behaviour is unchanged. This is a backward-compatible
parameter addition with no structural changes to the tool.

---

## Task Breakdown

### T1 — Add entity_id parameter to edit_file tool schema

**Deliverable:** Updated `edit_file` tool registration with the new `entity_id`
parameter accepted in the MCP tool schema.

**Scope:**
- Add `entity_id` as an optional string parameter to the `edit_file` tool's
  input schema definition.
- The parameter must be optional (not required) to preserve backward
  compatibility (REQ-NF-001).

**Dependencies:** None. First task — defines the interface.

**Verification:** Inspect the tool schema to confirm `entity_id` is present as
an optional parameter.

**Estimated effort:** 0.5 (one parameter addition in tool registration)

### T2 — Implement worktree path resolution in edit_file handler

**Deliverable:** Worktree path resolution logic in the `edit_file` handler that
mirrors the existing `write_file` implementation.

**Scope:**
- When `entity_id` is provided, resolve the worktree path using the same
  function that `write_file` uses (REQ-007, AC-008).
- When `entity_id` is omitted, resolve paths relative to the main repo root as
  currently (REQ-005, AC-005).
- When `entity_id` is provided but no worktree exists, return a clear error
  containing "no worktree found" and the entity ID (REQ-006, AC-006).
- The resolution must be O(1) — a single cache/store lookup, no filesystem
  scans (REQ-NF-002, AC-010).

**Dependencies:** T1 (needs the parameter schema defined)

**Verification:** Code review: trace the resolution code path and confirm it
calls the same function as `write_file` (AC-008); confirm single-lookup pattern
(AC-010).

**Estimated effort:** 2 (reuse existing worktree lookup, add error branch)

### T3 — Write tests for edit_file worktree-aware behaviour

**Deliverable:** Test file covering all new `entity_id` scenarios.

**Scope:**
- Test: `entity_id` provided with active worktree — edit applied in worktree,
  not main repo (AC-004).
- Test: `entity_id` omitted — behaviour identical to current, writes to main
  repo root (AC-005).
- Test: `entity_id` for non-existent entity — error with "no worktree found"
  (AC-006).
- Test: `entity_id` with multi-edit payload — all edits applied in worktree
  (AC-007).
- Test: existing `edit_file` test suite passes without modification (AC-009).

**Dependencies:** T2 (needs the implementation to test against)

**Verification:** `go test ./...` passes all new and existing tests.

**Estimated effort:** 2 (five test cases, worktree fixture setup)

---

## Dependency Graph

```
T1 (schema)
  └── T2 (resolution)
        └── T3 (tests)
```

Sequential chain: schema → implementation → tests. Each task builds on the
previous.

---

## Interface Contracts

| Boundary | Contract |
|----------|----------|
| T1 → T2 | `entity_id` parameter is available in the `edit_file` handler's input struct as an optional `*string` field. T2 reads it; if nil, uses main repo root; if set, resolves to worktree path. |
| T2 → T3 | `edit_file` handler correctly resolves paths for all three cases (entity_id set + worktree exists, entity_id nil, entity_id set + no worktree). T3 tests against these observable behaviours. |

---

## Traceability Matrix

| Requirement | Task(s) |
|-------------|---------|
| REQ-004 | T1, T2 |
| REQ-005 | T2 |
| REQ-006 | T2 |
| REQ-007 | T2 |
| REQ-NF-001 | T1, T3 |
| REQ-NF-002 | T2 |
| AC-004 | T3 |
| AC-005 | T3 |
| AC-006 | T3 |
| AC-007 | T3 |
| AC-008 | T2 |
| AC-009 | T3 |
| AC-010 | T2 |
