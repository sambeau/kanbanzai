# Specification: Fix Tool Correctness and Reliability

| Field  | Value                         |
|--------|-------------------------------|
| Date   | 2026-04-30                    |
| Status | Draft                         |
| Author | Spec Author                   |
| Batch  | B40-fix-tool-correctness      |
| Design | P40-retro-batch-april-2026/design-p40-design-retro-batch-improvements |

---

## Problem Statement

This specification implements Workstream B of the design described in
`work/P40-retro-batch-april-2026/P40-design-retro-batch-improvements.md`
(P40-retro-batch-april-2026/design-p40-design-retro-batch-improvements, approved).

Four retrospective reports (B36 review fixes, B38 batch review, P37 kbz move,
and the branch audit) document multiple correctness bugs in the MCP tool surface.
These are not discoverability or workflow issues — they are cases where tools
return incorrect results, silently corrupt data, or fail to propagate state
between the entity store and the cache layer. Each bug independently erodes
trust in workflow automation and forces agents into manual workarounds.

This specification addresses four fixes from the design's Workstream B:

1. **B1 — Fix `entity list parent_feature` filter:** The `parent_feature` filter
   on `entity(action: list, type: task)` returns all 614 tasks regardless of the
   filter value. It must return only tasks owned by the specified feature.
2. **B2 — Unify `finish()` state propagation:** `finish()` returns success but
   subsequent gate checks and list queries observe the task as `active`, not
   `done`. The write-through path for the entity record and the cache/index must
   be unified.
3. **B3 — Make multi-edit calls atomic:** When a multi-edit `edit_file` call
   contains multiple `old_text`/`new_text` pairs, non-matching edits are silently
   skipped while matching ones are applied. Multi-edit calls must be all-or-nothing.
4. **B4 — Expand decompose AC format recognition:** `decompose(action: propose)`
   only recognises three narrow AC formats. It must accept heading-based ACs,
   bold-with-parenthetical references, Given/When/Then blocks, and numbered lists
   under an "Acceptance Criteria" heading, and emit helpful diagnostics on failure.

**Scope inclusion:** The `entity` tool's list handler, the `finish` tool's
write-through logic, the `edit_file` tool's multi-edit application, and the
`decompose` tool's AC format parser.

**Scope exclusion:** The `edit_file` worktree-awareness parameter (that is B39-F2).
The terminal tool shell configuration. Document ownership inference (Workstream C).
Worktree cleanup automation (Workstream D). The `entity list` tool for any entity
type other than `task`. The `decompose` tool's proposal generation logic beyond
AC format recognition.

---

## Requirements

### Functional Requirements

- **REQ-001:** `entity(action: list, type: task, parent_feature: "<FEAT-ID>")`
  MUST return only tasks whose `parent_feature` field exactly matches the
  provided feature ID. It MUST NOT return tasks belonging to other features or
  tasks with no parent feature.

- **REQ-002:** When `finish(task_id: "TASK-...", status: "done")` returns
  success, all subsequent reads — including `entity(action: get)`,
  `entity(action: list)`, feature transition gate checks, and `status()`
  dashboards — MUST observe the task as `done` within the same request context.

- **REQ-003:** The `finish()` tool MUST write the task's terminal status to the
  entity YAML store and the SQLite cache/index synchronously before returning
  the response. The write order MUST be: entity record first, then cache
  invalidation/update.

- **REQ-004:** When `edit_file` receives a multi-edit call (multiple
  `old_text`/`new_text` pairs in the `edits` array), the tool MUST pre-flight
  all `old_text` patterns against the target file before applying any edits.

- **REQ-005:** If any `old_text` pattern in a multi-edit call fails to match,
  `edit_file` MUST apply zero edits and return an error listing which pattern(s)
  failed to match. The target file MUST be unchanged from its pre-call state.

- **REQ-006:** If all `old_text` patterns in a multi-edit call match, `edit_file`
  MUST apply all edits sequentially to the target file. The resulting file MUST
  reflect all requested changes.

- **REQ-007:** `decompose(action: propose)` MUST recognise acceptance criteria
  in each of the following formats when parsing a specification document:
  - Heading-based: `### AC-NNN` or `### AC-NNN: description`
  - Bold with parenthetical reference: `**AC-NNN (REQ-NNN):** description`
  - Bold with period: `**AC-NN.** description` (existing, must be preserved)
  - Checklist: `- [ ] **AC-NNN:** description` (existing, must be preserved)
  - Given/When/Then blocks: lines beginning with `**Given**`, `**When**`,
    `**Then**` under an acceptance criteria section
  - Numbered items under an "Acceptance Criteria" heading

- **REQ-008:** When `decompose(action: propose)` finds no recognisable acceptance
  criteria in any supported format, it MUST emit a diagnostic error that includes:
  - The list of sections found in the document
  - The closest matching patterns that were not recognised (with line numbers)
  - An example of the expected format

- **REQ-009:** Single-edit `edit_file` calls (one `old_text`/`new_text` pair)
  MUST behave identically to current behaviour — no change in matching or
  application logic for single-edit calls.

### Non-Functional Requirements

- **REQ-NF-001:** Fixing the `parent_feature` filter MUST NOT change the
  behaviour of `entity(action: list, type: task)` when called without the
  `parent_feature` parameter (i.e., unfiltered listing must still work).

- **REQ-NF-002:** The unified `finish()` write-through MUST NOT introduce a
  measurable latency increase — the additional cache write must complete in
  the same transaction scope as the existing entity record write.

- **REQ-NF-003:** Multi-edit pre-flighting MUST NOT read the target file more
  than once — all `old_text` patterns must be matched against a single read of
  the file content.

---

## Constraints

- The `entity` tool's existing parameter surface is preserved. No parameters are
  removed or renamed.
- The `finish` tool's existing parameter surface and response format are preserved.
  The change is internal to the write path.
- The `edit_file` tool's existing single-edit behaviour is preserved exactly.
  The change is limited to multi-edit calls.
- The `decompose` tool's existing proposal structure (task names, descriptions,
  dependency inference) is preserved. Only the AC format parser is extended.
- This specification does NOT cover worktree-awareness of `edit_file` (that is
  B39-F2, which may interact with B3 but is specified separately).
- All existing tests for the affected tools must continue to pass.

---

## Acceptance Criteria

- **AC-001 (REQ-001):** Given a feature `FEAT-01EXAMPLE` with 3 tasks and 611
  other tasks in the project, when `entity(action: list, type: task, parent_feature:
  "FEAT-01EXAMPLE")` is called, then exactly 3 tasks are returned, each with
  `parent_feature` equal to `FEAT-01EXAMPLE`.

- **AC-002 (REQ-001):** Given a feature ID that has no tasks, when
  `entity(action: list, type: task, parent_feature: "FEAT-01NO-TASKS")` is
  called, then an empty result set is returned, not all tasks.

- **AC-003 (REQ-002, REQ-003):** Given a task in `active` status, when
  `finish(task_id: "TASK-...", status: "done")` is called and returns success,
  then a subsequent `entity(action: get, id: "TASK-...")` returns `status: "done"`,
  and a subsequent `entity(action: list, type: task, parent_feature: "...")`
  shows the task as `done`, and a subsequent feature transition gate check does
  not block on that task.

- **AC-004 (REQ-004, REQ-005):** Given a file with content `line1\nline2\nline3`
  and a multi-edit call with two `old_text` patterns: `line1` (matches) and
  `nonexistent` (does not match), when the call executes, then the file is
  unchanged (still `line1\nline2\nline3`) and the error response contains
  `nonexistent` in the failed-pattern list.

- **AC-005 (REQ-004, REQ-006):** Given a file with content `line1\nline2\nline3`
  and a multi-edit call with two `old_text` patterns: `line1` and `line3`, both
  matching, when the call executes, then both edits are applied and the file
  reflects both changes.

- **AC-006 (REQ-007):** Given a specification document using heading-based ACs
  (`### AC-001: Given a user, when they log in, then they see the dashboard`),
  when `decompose(action: propose, feature_id: "...")` is called, then a task
  proposal is generated from the heading-based ACs.

- **AC-007 (REQ-007):** Given a specification document using bold-with-parenthetical
  ACs (`**AC-001 (REQ-001):** Given a user, when they log in, then they see
  the dashboard`), when `decompose(action: propose, feature_id: "...")` is
  called, then a task proposal is generated from these ACs.

- **AC-008 (REQ-007):** Given a specification document using Given/When/Then
  blocks under an acceptance criteria heading, when
  `decompose(action: propose, feature_id: "...")` is called, then a task
  proposal is generated from the GWT blocks.

- **AC-009 (REQ-008):** Given a specification document with acceptance criteria
  in an unrecognised format, when `decompose(action: propose, feature_id: "...")`
  is called, then the error response lists the sections found, shows the closest
  unrecognised patterns with line numbers, and provides an example of the
  expected format.

- **AC-010 (REQ-007):** Given a specification document using the existing
  supported formats (bold-with-period and checklist), when
  `decompose(action: propose, feature_id: "...")` is called, then the tool
  continues to generate correct task proposals — existing format support is
  preserved.

- **AC-011 (REQ-009, REQ-NF-003):** Given an `edit_file` single-edit call
  (one `old_text`/`new_text` pair), when the call executes, then behaviour is
  identical to the current implementation — no regression in single-edit
  matching or application.

- **AC-012 (REQ-NF-001):** Given an `entity(action: list, type: task)` call
  with no `parent_feature` filter, when the call executes, then all tasks are
  returned — the unfiltered listing path is unchanged.

- **AC-013 (REQ-NF-002):** Given a `finish()` call on a task, when the latency
  is measured against the pre-fix implementation, then the additional cache
  write does not increase p95 latency by more than 5ms.

---

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Test | Automated test: create feature with 3 tasks, call list with parent_feature filter, assert exactly 3 results with correct parent_feature |
| AC-002 | Test | Automated test: call list with parent_feature for a feature ID that has no tasks, assert empty result set |
| AC-003 | Test | Integration test: finish a task, then immediately get/list/check-gate, assert all three observe done status |
| AC-004 | Test | Automated test: create file, call multi-edit with one matching and one non-matching pattern, assert file unchanged and error names the failed pattern |
| AC-005 | Test | Automated test: create file, call multi-edit with two matching patterns, assert both edits applied |
| AC-006 | Test | Automated test: create spec with heading-based ACs, call decompose propose, assert task proposal generated |
| AC-007 | Test | Automated test: create spec with bold-with-parenthetical ACs, call decompose propose, assert task proposal generated |
| AC-008 | Test | Automated test: create spec with Given/When/Then blocks, call decompose propose, assert task proposal generated |
| AC-009 | Test | Automated test: create spec with unrecognised AC format, call decompose propose, assert diagnostic error with sections, candidates, and format example |
| AC-010 | Test | Run existing decompose tests with bold-period and checklist formats; all pass |
| AC-011 | Test | Run existing edit_file single-edit tests; all pass without modification |
| AC-012 | Test | Call entity list without parent_feature; assert all tasks returned |
| AC-013 | Test | Benchmark finish() latency before and after; assert p95 increase ≤ 5ms |
