# Specification: Next Tool UX Improvements

| Field   | Value                                         |
|---------|-----------------------------------------------|
| Date    | 2026-04-25                                    |
| Status  | Draft                                         |
| Author  | Spec Author                                   |
| Feature | FEAT-01KQ2E0RHSNP1                            |
| Plan    | P34-agent-workflow-ergonomics                 |

---

## 1. Related Work

**Design:** `work/design/design-p34-agent-workflow-ergonomics.md`
(`P34-agent-workflow-ergonomics/design-design-p34-agent-workflow-ergonomics`) —
§3 · Idempotent task claim in `next()` and §4 · Worktree path in context packet.

**Prior decisions and designs consulted:**

| Document | Relevance to this specification |
|----------|---------------------------------|
| P29 — State Store Read Path Performance | Reduced `next()` timeout frequency by eliminating O(n) file scans. This specification addresses the remaining UX problem when a timeout does still occur: the tool returns an unrecoverable error rather than the context packet the agent needs. |
| P21 — Codebase Memory Integration | Added `worktree.Path` to the worktree record and `actx.worktreePath` to context assembly. The worktree path is already computed in `assembly.go` but never written to the `next()` response. This specification covers that one-field omission. |

No prior specification was found for idempotent task claiming or worktree path
surfacing in the context packet. The design's Related Work section attests:
"No directly related prior work was found on idempotent task claiming."

---

## 2. Overview

This specification covers two small, independently motivated improvements to
`next_tool.go`. First, when `next(id: TASK-...)` is called for a task already in
`active` status, the tool returns the assembled context packet (with a `reclaimed:
true` flag) instead of an error — recovering gracefully from timeout-induced
double-claims without requiring any manual state repair. Second, the context packet
response is extended with a `worktree_path` field containing the filesystem path of
the active worktree for the parent feature, eliminating a redundant
`worktree(action: get)` round-trip for information the server has already retrieved.

---

## 3. Scope

**In scope:**

- Changing the `active` status branch in `nextClaimMode` to return the context
  packet with `reclaimed: true` instead of returning an error.
- Adding a conditional `worktree_path` field to the `nextContextToMap` response
  when an active worktree exists for the parent feature.
- Preserving existing dispatch metadata (`claimed_at`, `dispatched_to`,
  `dispatched_by`) unchanged on a reclaim.

**Out of scope:**

- Changes to error behaviour for tasks in any status other than `active` (tasks in
  `queued`, `done`, `not-planned`, `cancelled`, `needs-review`, or `needs-rework`
  continue to return errors unchanged).
- Any change to how the initial (first) claim of a task works.
- Re-firing the `OnStatusTransition` hook or updating `claimed_at` on a reclaim.
- Adding `worktree_path` to any MCP tool response other than `next()`.
- Automatic crash recovery or stale-active-task reversion (addressed separately
  in P13).

---

## 4. Functional Requirements

### Idempotent task claim

**FR-001** — When `next(id: TASK-...)` is called for a task whose current status
is `active`, the tool MUST return the assembled context packet instead of an error.

**FR-002** — The context packet returned for an already-active task MUST include
a boolean field `reclaimed` set to `true`. A task claimed for the first time MUST
NOT include a `reclaimed` field (or MUST set it to `false`).

**FR-003** — When returning the context packet for an already-active task, the
tool MUST NOT overwrite or modify the task's `claimed_at`, `dispatched_to`, or
`dispatched_by` metadata fields. The values written during the original claim MUST
be surfaced unchanged in the reclaim response.

**FR-004** — When returning the context packet for an already-active task, the
tool MUST NOT re-fire the `OnStatusTransition` hook or trigger any other lifecycle
side effect.

**FR-005** — The error behaviour for tasks in all statuses other than `active` and
`ready` MUST remain unchanged. Calling `next(id: TASK-...)` for a task in `queued`,
`done`, `not-planned`, `cancelled`, `needs-review`, or `needs-rework` status MUST
continue to return an error.

### Worktree path in context packet

**FR-006** — When `next(id: TASK-...)` is called (for either a first claim or a
reclaim) and an active worktree record exists for the task's parent feature, the
context packet MUST include a `worktree_path` field whose value is the `Path` field
from the worktree record.

**FR-007** — When no active worktree record exists for the task's parent feature,
the `worktree_path` field MUST be omitted from the context packet entirely. The
field MUST NOT be present as a null value or empty string.

**FR-008** — The `worktree_path` value MUST be sourced from the worktree record's
`Path` field (i.e. `actx.worktreePath`, already computed in `assembleContext`).
No additional worktree lookup is required.

---

## 5. Non-Functional Requirements

**NFR-001** — The reclaim response MUST be backward-compatible: callers that do
not inspect the `reclaimed` field MUST continue to behave correctly, since both
a first-claim and a reclaim return the same context packet shape.

**NFR-002** — The `worktree_path` field's omission-when-absent behaviour MUST be
consistent with how other optional fields (`tool_hint`, `graph_project`, and
similar) are handled in the same `nextContextToMap` function: absent means the key
is not present in the response map, not that it is present with a zero value.

**NFR-003** — The idempotent claim path MUST NOT introduce a second call to the
task storage layer beyond the one already made to check the task's current status.
The task record loaded for the error-path check MUST be reused for context
assembly.

---

## 6. Acceptance Criteria

**AC-001 (FR-001, FR-002)** — Given a task in `active` status, when
`next(id: <task-id>)` is called, then the tool returns a success response
containing the full context packet and `reclaimed: true`, with no error.

**AC-002 (FR-003)** — Given a task first claimed at time T with `dispatched_to`
set to agent A, when `next(id: <task-id>)` is called again, then the response
contains `claimed_at = T` and `dispatched_to = A` (original metadata preserved).

**AC-003 (FR-004)** — Given a task in `active` status, when
`next(id: <task-id>)` is called, then no lifecycle side effect is triggered and
the task's status remains `active`.

**AC-004 (FR-005)** — Given a task in `done` status, when
`next(id: <task-id>)` is called, then the tool returns an error (unchanged
behaviour).

**AC-005 (FR-005)** — Given a task in `queued` status, when
`next(id: <task-id>)` is called, then the tool returns an error (unchanged
behaviour).

**AC-006 (FR-006)** — Given a task in `ready` status whose parent feature has
an active worktree at path `/worktrees/feat-foo`, when `next(id: <task-id>)` is
called, then the response includes `worktree_path: "/worktrees/feat-foo"`.

**AC-007 (FR-007)** — Given a task in `ready` status whose parent feature has
no active worktree, when `next(id: <task-id>)` is called, then the response map
does not contain a `worktree_path` key.

**AC-008 (FR-006, FR-001)** — Given a task in `active` status whose parent
feature has an active worktree, when `next(id: <task-id>)` is called, then the
response includes both `reclaimed: true` and `worktree_path` set to the worktree's
path.

**AC-009 (FR-002)** — Given a task in `ready` status claimed for the first time,
when `next(id: <task-id>)` is called, then the response does not contain a
`reclaimed` field (or it is `false`).

---

## 7. Dependencies and Assumptions

**DEP-001** — The worktree record's `Path` field must be populated. This field was
added in P21 as part of `write_file` entity_id support; it is a hard prerequisite
for FR-006 through FR-008.

**DEP-002** — `assembleContext` must continue to set `actx.worktreePath = wt.Path`
when a worktree record is found. This specification does not change that logic; it
only adds the field to `nextContextToMap`.

**DEP-003** — The `active` status check in `nextClaimMode` must have access to the
already-loaded task record so that `assembleContext` can be called without an
additional storage read.

**ASM-001** — A task in `active` status is a valid, well-formed task record that
can be passed to `assembleContext` without modification. The context assembly
function does not require the task to be in `ready` status.

**ASM-002** — Callers that receive `reclaimed: true` in the response will treat it
identically to a first-claim response: they proceed with the task work. The flag is
informational only and does not change the agent's subsequent behaviour.

**ASM-003** — A task has at most one parent feature, and that feature has at most
one active worktree record. No disambiguation is required when looking up
`worktree_path`.