# Implementation Plan: Composite Tools for Workflow Chaining

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-04                     |
| Status | approved |
| Author | sambeau                        |

## Overview

This implementation plan decomposes the Composite Tools for Workflow Chaining
specification into seven tasks across two implementation waves plus one
integration verification wave. Five composite MCP tool actions are implemented
on four tools: `doc(action: "publish")`, `entity(action: "bootstrap")`,
`entity(action: "close-out")`, `develop(action: "dispatch")`, and
`batch(action: "snapshot")`. A shared `next_action` response helper ensures
format consistency across all gate-failure responses.

## Scope

> This plan implements the requirements defined in
> `work/B43-composite-tools/B43-F1-spec-composite-tools.md`
> (FEAT-01KQJ7CJGQR7Y/spec-b43-f1-spec-composite-tools).

This plan covers the implementation of five composite MCP tool actions on
four tools (`doc`, `entity`, `develop`, `batch`). It covers tasks T1–T7
below: three tool-group implementation tasks, one new-tool scaffolding task,
one integration verification task, one structured response helper task,
and one test-suite preservation task.

This plan does **not** cover: changes to the underlying `service`, `gate`,
or `binding` packages; AI-driven server-side classification; client-side
workflow scripting; or removal of existing individual tool actions.

## Task Breakdown

### Task 1: Implement `doc(action: "publish")`

- **Description:** Add a `publish` action handler to the `doc` tool's
  `DispatchAction` map. The handler accepts `path`, `type`, `title`,
  `owner`, optional `classifications` array, `model_name`, and
  `model_version`. It calls the existing `docRegisterOne` path, then if
  classifications with populated `concepts_intro` are provided, calls
  classification logic internally and approves via `docApproveOne`. If
  classifications are omitted, returns the existing classification nudge
  with a structured `next_action`. On partial failure, returns
  `registration: "ok"` and `approval: "failed: <reason>"`.
- **Deliverable:** Modified `internal/mcp/doc_tool.go` with `publish` action
  handler, plus unit tests in `internal/mcp/doc_tool_test.go`.
- **Depends on:** None (beyond existing codebase)
- **Effort:** Medium (3 story points)
- **Spec requirements:** REQ-001, REQ-002, REQ-003, REQ-004

### Task 2: Implement `entity(action: "bootstrap")`

- **Description:** Add a `bootstrap` action handler to the `entity` tool's
  `DispatchAction` map. The handler accepts `feature_id` and optional
  `target`. It reuses `AdvanceFeatureStatus` internally to walk the forward
  path, but enriches the response: at each gate failure it returns a
  structured `next_action` object (`{tool, action, params, description}`)
  derived from the specific missing prerequisite. At human gates, it stops
  with `reason: "human_gate"`. On success, returns `advanced_through` list.
- **Deliverable:** Modified `internal/mcp/entity_tool.go` with `bootstrap`
  action handler, plus unit tests in `internal/mcp/entity_tool_test.go`.
- **Depends on:** None
- **Effort:** Large (5 story points)
- **Spec requirements:** REQ-005, REQ-006, REQ-007, REQ-008

### Task 3: Implement `entity(action: "close-out")`

- **Description:** Add a `close-out` action handler to the `entity` tool's
  `DispatchAction` map. The handler accepts `feature_id`, verifies all tasks
  are terminal, checks for an approved review report document, advances the
  feature to `done`, and triggers parent batch auto-advancement via
  `MaybeAutoAdvancePlan`. On any blocker, returns structured `next_action`.
  On success, enumerates all cascading transitions.
- **Deliverable:** Modified `internal/mcp/entity_tool.go` with `close-out`
  action handler, plus unit tests in `internal/mcp/entity_tool_test.go`.
- **Depends on:** None (can parallelise with Task 2, but shares
  `entity_tool.go` — see dependency graph for serialisation guidance)
- **Effort:** Medium (3 story points)
- **Spec requirements:** REQ-009, REQ-010, REQ-011, REQ-012

### Task 4: Scaffold `develop` tool and implement `dispatch` action

- **Description:** Create a new top-level `develop` tool
  (`internal/mcp/develop_tool.go`) following the existing consolidated-tool
  pattern (`server.ServerTool` with `DispatchAction` map). Register it in
  `internal/mcp/server.go`. Implement the `dispatch` action handler: accept
  `feature_id`, `role`, `instructions`; identify the ready task frontier;
  run conflict analysis via the existing conflict detection logic; transition
  conflict-safe tasks from `ready` to `active`; generate handoff prompts via
  the existing `handoff` pipeline. Return `dispatched`, `conflicting`,
  `blocked`, and `empty_queue`. Do NOT call `spawn_agent`.
- **Deliverable:** New `internal/mcp/develop_tool.go` and modified
  `internal/mcp/server.go`, plus unit tests in
  `internal/mcp/develop_tool_test.go`.
- **Depends on:** None
- **Effort:** Large (5 story points)
- **Spec requirements:** REQ-013, REQ-014, REQ-015, REQ-016

### Task 5: Implement `batch(action: "snapshot")`

- **Description:** Add a `snapshot` action to the batch infrastructure.
  The handler accepts `batch_id`, lists all features in the batch, evaluates
  blocking gates for each non-terminal feature, and returns a structured
  rollup: per-feature status, blocked state, blocking gate, missing
  prerequisite, and structured `next_action`. Features are sorted in
  dependency order (features that unblock others first).
- **Deliverable:** Modified batch tool handler (in batch infrastructure
  package), plus unit tests.
- **Depends on:** None
- **Effort:** Medium (3 story points)
- **Spec requirements:** REQ-017, REQ-018

### Task 6: Implement structured `next_action` response helper

- **Description:** Extract a shared helper for constructing structured
  `next_action` objects (`{tool, action, params, description}`). Gate
  failure responses from bootstrap, close-out, and snapshot all produce
  these objects — a shared helper ensures format consistency. The helper
  maps common gate failure types (missing document, missing approval,
  non-terminal tasks, human gate) to the correct tool/action/params tuple.
  This task also includes ensuring all composite responses comply with
  REQ-NF-003 (structured `next_action` schema).
- **Deliverable:** New or modified helper file in `internal/mcp/`, consumed
  by Tasks 2, 3, and 5.
- **Depends on:** None (provides shared dependency for Tasks 2, 3, 5)
- **Effort:** Small (2 story points)
- **Spec requirements:** REQ-006, REQ-011, REQ-018, REQ-NF-003

### Task 7: Integration verification and test-suite preservation

- **Description:** Write integration-level tests that exercise each
  composite action end-to-end: publish with and without classifications,
  bootstrap through multiple stages and gate failures, close-out with
  cascade, dispatch with mixed task states, snapshot with mixed feature
  states. Verify that `SideEffect` types from composite actions match those
  from equivalent sequences of individual tool calls (REQ-014 verify).
  Run the full existing test suites for `doc_tool_test.go` and
  `entity_tool_test.go` to confirm zero regressions (AC-015).
- **Deliverable:** Integration test file and verified passing existing
  test suites.
- **Depends on:** Tasks 1, 2, 3, 4, 5, 6 (all implementation tasks)
- **Effort:** Medium (3 story points)
- **Spec requirements:** REQ-019, REQ-020, REQ-021, REQ-022, REQ-NF-004,
  AC-014, AC-015

## Dependency Graph

```
Task 6 (shared next_action helper) — no dependencies
Task 1 (doc publish)               — no dependencies
Task 4 (develop dispatch)          — no dependencies
Task 5 (batch snapshot)            — no dependencies [uses Task 6 for next_action format]

Task 2 (entity bootstrap)          — depends on Task 6
Task 3 (entity close-out)          — depends on Task 6

Task 7 (integration verification)  — depends on Tasks 1, 2, 3, 4, 5, 6
```

**Parallel groups:**
- Wave 1: [Task 1, Task 4, Task 5, Task 6] — all independent
- Wave 2: [Task 2, Task 3] — depend on Task 6, can run in parallel with each other
  *Note:* Task 2 and Task 3 both modify `entity_tool.go` — if run in parallel, risk
  merge conflicts. Serialisation recommended: Task 2 → Task 3, or use separate branches
  with explicit merge coordination.
- Wave 3: [Task 7] — depends on all others

**Critical path:** Task 6 → Task 2 → Task 7 (2 + 5 + 3 = 10 story points)

## Risk Assessment

### Risk: Merge conflicts on `entity_tool.go` (Tasks 2 and 3)

- **Probability:** High — both tasks add new entries to the same
  `DispatchAction` map in the same file.
- **Impact:** Medium — resolution is straightforward (add both entries) but
  may cause rework if one task changes handler signatures.
- **Mitigation:** Serialise Tasks 2 and 3 on the same branch, or have the
  second task rebase onto the first. Alternatively, use the feature's
  worktree isolation to keep them separate.
- **Affected tasks:** Task 2, Task 3

### Risk: Conflict analysis logic not directly callable from `develop`

- **Probability:** Medium — the existing `conflict` tool currently operates
  as an MCP tool entry point; the internal logic may need extraction into a
  reusable function.
- **Impact:** Medium — would require refactoring the conflict tool to expose
  an internal function, adding scope to Task 4.
- **Mitigation:** Task 4 implementer should inspect the conflict tool's
  internal structure early and flag if extraction is needed. If extraction
  is non-trivial, split into a prerequisite refactor task.
- **Affected tasks:** Task 4

### Risk: `AdvanceFeatureStatus` response format insufficient for `bootstrap`

- **Probability:** Low — `AdvanceFeatureStatus` already stops at gate
  failures and reports `stopped_reason`. The `bootstrap` action enriches
  this with structured `next_action` derived from the same gate evaluation
  results.
- **Impact:** Medium — if the gate failure information is insufficient to
  construct a precise `next_action`, additional gate introspection may be
  needed.
- **Mitigation:** Task 2 implementer should verify that gate evaluation
  results carry enough detail (document type, document ID, task IDs) to
  construct precise `next_action` objects. The `next_action` helper (Task 6)
  should be designed to accept whatever granularity the gate results provide
  and degrade gracefully to a best-effort instruction.
- **Affected tasks:** Task 2, Task 6

### Risk: Handoff prompt generation not callable as a pure function

- **Probability:** Low — `handoff` is already an action handler in the
  consolidated tool pattern; the `dispatch` action calls it for each task.
- **Impact:** Low — if the handoff pipeline has side effects or requires
  specific request shapes, the `dispatch` handler can construct the
  appropriate internal request objects.
- **Mitigation:** Task 4 implementer should trace the handoff call path
  early to confirm it can be called programmatically from another handler.
- **Affected tasks:** Task 4

## Interface Contracts

Composites do not introduce new abstractions — they wrap existing functions.
The interface contracts are therefore the existing function signatures of the
wrapped service and gate logic:

| Composite | Wrapped functions | File |
|-----------|-------------------|------|
| `doc publish` | `docRegisterOne`, classification logic, `docApproveOne` | `internal/mcp/doc_tool.go` |
| `entity bootstrap` | `AdvanceFeatureStatus`, `GateRouter.CheckGate` | `service/advance.go`, `gate/` |
| `entity close-out` | `CountNonTerminalTasks`, `UpdateStatus`, `MaybeAutoAdvancePlan` | `service/entity_children.go` |
| `develop dispatch` | `handoff` pipeline, conflict analysis logic | `internal/mcp/handoff_tool.go`, conflict infrastructure |
| `batch snapshot` | `ListFeatures`, gate evaluation | batch infrastructure, `gate/` |

All composites use `WithSideEffects` middleware and produce `SideEffect*`
types matching individual tool calls. No function signature changes are
required on any wrapped function.

## Traceability Matrix

| Requirement | Task | Acceptance Criterion |
|------------|------|---------------------|
| REQ-001 | T1 | AC-001 |
| REQ-002 | T1 | AC-001 |
| REQ-003 | T1 | AC-002 |
| REQ-004 | T1 | AC-003 |
| REQ-005 | T2 | AC-004 |
| REQ-006 | T2, T6 | AC-005 |
| REQ-007 | T2 | AC-006 |
| REQ-008 | T2 | AC-004 |
| REQ-009 | T3 | AC-007 |
| REQ-010 | T3 | AC-008 |
| REQ-011 | T3, T6 | AC-009 |
| REQ-012 | T3 | AC-007 |
| REQ-013 | T4 | AC-010, AC-011 |
| REQ-014 | T4 | AC-010 |
| REQ-015 | T4 | AC-011 |
| REQ-016 | T4 | AC-012 |
| REQ-017 | T5 | AC-013 |
| REQ-018 | T5, T6 | AC-013 |
| REQ-019 | T7 | AC-014 |
| REQ-020 | T7 | AC-014 |
| REQ-021 | T7 | — (architectural) |
| REQ-022 | T7 | — (architectural) |
| REQ-NF-001 | T7 | — (measurement) |
| REQ-NF-002 | T7 | — (measurement) |
| REQ-NF-003 | T6 | — (schema compliance) |
| REQ-NF-004 | T7 | AC-015 |

## Verification Approach

| Acceptance Criterion | Verification Method | Producing Task |
|----------------------|--------------------|----------------|
| AC-001: publish with classifications → approved | Unit test | Task 1 |
| AC-002: publish without classifications → draft + next_action | Unit test | Task 1 |
| AC-003: publish with failed classification → partial success | Unit test | Task 1 |
| AC-004: bootstrap through multiple stages | Unit test | Task 2 |
| AC-005: bootstrap blocked by missing doc → next_action | Unit test | Task 2 |
| AC-006: bootstrap stops at human gate | Unit test | Task 2 |
| AC-007: close-out succeeds with cascade | Unit test | Task 3 |
| AC-008: close-out blocked by non-terminal task | Unit test | Task 3 |
| AC-009: close-out blocked by missing review report | Unit test | Task 3 |
| AC-010: dispatch transitions ready tasks and generates prompts | Unit test | Task 4 |
| AC-011: dispatch does not spawn agents | Code inspection | Task 4 |
| AC-012: dispatch reports mixed task states correctly | Unit test | Task 4 |
| AC-013: snapshot enumerates every feature with next_action | Unit test | Task 5 |
| AC-014: composite side effects match individual transitions | Integration test | Task 7 |
| AC-015: existing test suites pass unchanged | Regression test | Task 7 |
| REQ-NF-001: tool call reduction ≥50% | Manual measurement | Task 7 |
| REQ-NF-002: response time ≤2× individual operations | Manual measurement | Task 7 |
