# Specification: Composite Tools for Workflow Chaining

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-04                     |
| Status | approved |
| Author | sambeau                        |

> This specification implements the design described in
> `work/B43-composite-tools/B43-F1-design-composite-tools.md` (FEAT-01KQJ7CJGQR7Y/design-b43-f1-design-composite-tools).

## Overview

This specification defines testable requirements for five **composite MCP tool actions** —
new actions on existing consolidated MCP tools (plus one new top-level `develop` tool) that
collapse multi-step workflow sequences into single server-side operations. It implements the
design described in `work/B43-composite-tools/B43-F1-design-composite-tools.md`
(FEAT-01KQJ7CJGQR7Y/design-b43-f1-design-composite-tools).

Each composite is deterministic, synchronous, and stateless: it orchestrates existing service
and gate logic without introducing AI calls, background execution, or new state machines.

The five composite actions are:
- `doc(action: "publish")` — register + classify + approve
- `entity(action: "bootstrap")` — lifecycle setup through developing
- `entity(action: "close-out")` — reviewing → done with cascade
- `develop(action: "dispatch")` — one-cycle orchestrator dispatch
- `batch(action: "snapshot")` — prescriptive status rollup

## Scope

**In scope:**
- Five new composite actions on the `doc`, `entity`, `develop`, and `batch` MCP tools
- Structured `next_action` response objects for every gate failure and incomplete workflow
- Deterministic orchestration of existing service, gate, and lifecycle functions
- Full test coverage for each composite action including error paths

**Out of scope:**
- Changes to the underlying service, gate, or lifecycle packages — composites wrap existing logic
- AI-driven classification inside the MCP server — classification remains a chat-agent responsibility
- Client-side workflow scripts or stateful session management — composites are stateless request-response calls
- Removal or deprecation of individual tool actions — existing actions remain available
- Cross-batch or cross-project workflow coordination

## Functional Requirements

- **REQ-001:** The `doc` tool SHALL support a `publish` action that accepts a document path,
  type, title, owner, and optional classifications array.
- **REQ-002:** When classifications with at least one populated `concepts_intro` are provided,
  `doc(action: "publish")` SHALL register the document, apply the classifications, and approve
  the document in a single call, returning the approved document record.
- **REQ-003:** When classifications are omitted or have no populated `concepts_intro`,
  `doc(action: "publish")` SHALL register the document and return a classification nudge
  with a structured `next_action` indicating `doc(action: "approve")` as the next step,
  without auto-approving.
- **REQ-004:** If registration succeeds but classification or approval fails during
  `doc(action: "publish")`, the document SHALL remain in `draft` status and the response
  SHALL report `registration: "ok"` and `approval: "failed: <reason>"`.
- **REQ-005:** The `entity` tool SHALL support a `bootstrap` action that accepts a
  `feature_id` and optional `target` status, and walks the feature's forward path from its
  current status toward the target.
- **REQ-006:** When `entity(action: "bootstrap")` encounters a failing gate, it SHALL stop
  and return a response containing `stopped_at`, `reason`, and a structured `next_action`
  object with fields `tool`, `action`, `params`, and `description`.
- **REQ-007:** When `entity(action: "bootstrap")` encounters a stage with `human_gate: true`,
  it SHALL stop and return `reason: "human_gate"` with a descriptive message, and SHALL NOT
  auto-advance past the human gate under any circumstances.
- **REQ-008:** When `entity(action: "bootstrap")` reaches the target status, it SHALL return
  the new status and a list of stages advanced through.
- **REQ-009:** The `entity` tool SHALL support a `close-out` action that accepts a
  `feature_id` and advances it from `reviewing` to `done`, then checks the parent batch for
  cascading auto-advancement.
- **REQ-010:** `entity(action: "close-out")` SHALL verify all tasks are in terminal states
  before advancing the feature, and SHALL return a structured `next_action` if any task
  is not terminal.
- **REQ-011:** `entity(action: "close-out")` SHALL check for an approved review report
  document before advancing the feature, and SHALL return a structured `next_action`
  indicating the missing document if absent.
- **REQ-012:** When `entity(action: "close-out")` succeeds, the response SHALL enumerate
  every entity affected by the cascading transitions (feature, parent batch, any auto-advanced
  entities).
- **REQ-013:** A new top-level `develop` tool SHALL exist with a `dispatch` action that
  accepts a `feature_id`, optional `role`, and optional `instructions`.
- **REQ-014:** `develop(action: "dispatch")` SHALL identify the ready task frontier (tasks
  with status `ready` and all `depends_on` satisfied), run conflict analysis on those tasks
  using the existing conflict detection logic, and transition conflict-safe tasks from
  `ready` to `active`.
- **REQ-015:** `develop(action: "dispatch")` SHALL generate a handoff prompt for each
  dispatched task using the existing handoff pipeline, and SHALL NOT call `spawn_agent`
  itself.
- **REQ-016:** `develop(action: "dispatch")` SHALL return `dispatched` (array of dispatched
  tasks with handoff prompts), `conflicting` (array of ready-but-conflicting tasks with
  explanation), `blocked` (array of non-ready tasks with blocking reasons), and
  `empty_queue` (boolean) fields.
- **REQ-017:** The `batch` tool SHALL support a `snapshot` action that accepts a `batch_id`
  and returns a prescriptive status rollup.
- **REQ-018:** `batch(action: "snapshot")` SHALL enumerate every feature in the batch with
  its current status, whether it is blocked, what gate is blocking it, what specific
  prerequisite is missing, and a structured `next_action` for each blocked feature.
- **REQ-019:** All composite actions SHALL reuse existing service, gate, and lifecycle logic
  without modification — composites are wrappers over existing functions, not new
  abstractions.
- **REQ-020:** All composite actions SHALL use the existing `WithSideEffects` middleware and
  push the same `SideEffect*` types as individual tool calls.
- **REQ-021:** Every composite action SHALL be synchronous — each call completes entirely
  before returning a response, with no background execution or event-driven continuation.
- **REQ-022:** Each internal step within a composite action SHALL be an independent atomic
  operation — if step N fails, steps 1 through N-1 remain persisted and the response
  SHALL report which steps succeeded and which failed.

## Non-Functional Requirements

- **REQ-NF-001:** Each composite action, when it replaces N sequential tool calls, SHALL
  reduce the total tool calls needed for the workflow by at least 50% (e.g., the 3-step
  document publish workflow SHALL complete in 1 call plus at most 1 approval call when no
  classifications are provided).
- **REQ-NF-002:** Composite action response times SHALL be no more than 2× the sum of the
  response times of the individual operations they compose, measured under identical system
  load.
- **REQ-NF-003:** All composite action responses SHALL include a `next_action` field (or
  explicit `null`) whenever the workflow is incomplete, following the schema:
  `{tool, action, params, description}`.
- **REQ-NF-004:** Existing individual tool actions (`doc register`, `doc approve`,
  `entity transition`, etc.) SHALL continue to function identically after composite actions
  are added — no breaking changes to existing tool signatures or behaviour.

## Constraints

- **Backward compatibility:** No existing tool action signature or behaviour may change.
  Composite actions are additive entries in existing `DispatchAction` maps.
- **No new state machines:** Composite actions reuse the existing lifecycle state machines
  in `service/advance.go`, gate evaluators in `gate/`, and transition logic — no new
  state-tracking infrastructure.
- **No AI/LLM calls:** All composite logic is deterministic orchestration of existing
  service functions. No API keys, model selection, or LLM invocation occurs inside the
  MCP server.
- **Human gates are hard stops:** No composite action may auto-pass a stage with
  `human_gate: true` in the stage bindings.
- **Design constraints:** Implementation must follow the `DispatchAction` pattern used
  by all consolidated tools (per `internal/mcp/sideeffect.go`), with one new top-level tool
  (`develop`) for the dispatch action. Design principles DP-9 (constraint levels), DP-5
  (composition), and DP-10 (only add what the model doesn't know) from the Skills System
  Redesign constrain all composite behaviour.

## Acceptance Criteria

- **AC-001 (REQ-001, REQ-002):** Given a valid document path, type, title, and owner with
  at least one classification containing a populated `concepts_intro`, when
  `doc(action: "publish")` is called, then the document is registered and approved within
  a single response, and the returned document record has `status: "approved"`.
- **AC-002 (REQ-003):** Given a valid document registration without classifications, when
  `doc(action: "publish")` is called, then the document is registered in `draft` status and
  the response includes `next_action: {tool: "doc", action: "approve", ...}` without
  auto-approving.
- **AC-003 (REQ-004):** Given a document registration that succeeds but classification
  fails (e.g., invalid classification schema), when `doc(action: "publish")` is called,
  then the response contains `registration: "ok"` and `approval: "failed: <reason>"` and
  the document remains in `draft` status.
- **AC-004 (REQ-005, REQ-008):** Given a feature in `specifying` with an approved spec and
  no blocking prerequisites, when `entity(action: "bootstrap", feature_id: "<id>")` is
  called, then the feature advances through `dev-planning` to `developing` and the response
  includes `advanced_through: ["specifying", "dev-planning", "developing"]`.
- **AC-005 (REQ-006):** Given a feature in `specifying` with a missing specification
  document, when `entity(action: "bootstrap")` is called, then the response contains
  `stopped_at: "specifying"`, a reason indicating the missing document, and
  `next_action: {tool: "doc", action: "register", params: {type: "specification"}, ...}`.
- **AC-006 (REQ-007):** Given a feature configured with `human_gate: true` on the
  `specifying` stage, when `entity(action: "bootstrap")` reaches that stage, then the
  response contains `stopped_at: "specifying"`, `reason: "human_gate"`, and the feature
  is not auto-advanced past the human gate.
- **AC-007 (REQ-009, REQ-010):** Given a feature in `reviewing` with all tasks terminal,
  when `entity(action: "close-out")` is called, then the feature transitions to `done`
  and the response enumerates the feature and any auto-advanced parent batch.
- **AC-008 (REQ-010):** Given a feature in `reviewing` with at least one non-terminal task,
  when `entity(action: "close-out")` is called, then the response contains a structured
  `next_action` indicating which tasks are not terminal and the feature remains in
  `reviewing`.
- **AC-009 (REQ-011):** Given a feature in `reviewing` with all tasks terminal but no
  approved review report, when `entity(action: "close-out")` is called, then the response
  contains a structured `next_action` indicating the missing review report document.
- **AC-010 (REQ-013, REQ-014):** Given a feature in `developing` with at least one task
  in `ready` status and no conflicts, when `develop(action: "dispatch")` is called, then
  the ready tasks transition to `active`, handoff prompts are generated for each, and the
  response includes a `dispatched` array with `{task_id, handoff_prompt}` for each.
- **AC-011 (REQ-015):** Given a successful `develop(action: "dispatch")` call, the response
  SHALL NOT include any sub-agent spawn results — only handoff prompts for the caller to
  use with `spawn_agent`.
- **AC-012 (REQ-016):** Given a feature in `developing` with a mix of ready, conflicting,
  and blocked tasks, when `develop(action: "dispatch")` is called, then the response
  contains `dispatched`, `conflicting`, `blocked`, and `empty_queue` fields correctly
  reflecting each task's state.
- **AC-013 (REQ-017, REQ-018):** Given a batch containing features in mixed lifecycle
  states, when `batch(action: "snapshot")` is called, then the response enumerates every
  feature with its status, blocked state, blocking gate, missing prerequisite, and
  structured `next_action`.
- **AC-014 (REQ-019):** Given a composite action that advances a feature (e.g., bootstrap
  or close-out), the resulting side effects SHALL be identical in type and content to
  those produced by the equivalent sequence of individual `entity(action: "transition")`
  calls.
- **AC-015 (REQ-NF-004):** Given the addition of composite action handlers to the `doc`
  and `entity` tools, existing actions (`register`, `approve`, `transition`, etc.) SHALL
  pass all existing tests without modification.

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Test | Automated Go test: call `doc(action: "publish")` with valid classifications, assert document status is `approved` |
| AC-002 | Test | Automated Go test: call `doc(action: "publish")` without classifications, assert draft status and `next_action` structure |
| AC-003 | Test | Automated Go test: inject classification failure via mock, assert partial-success response format |
| AC-004 | Test | Automated Go test: create feature in `specifying` with approved documents, call bootstrap, assert final status and `advanced_through` |
| AC-005 | Test | Automated Go test: create feature in `specifying` without spec doc, call bootstrap, assert `stopped_at` and `next_action` |
| AC-006 | Test | Automated Go test: configure human gate on stage, call bootstrap, assert stop at human gate |
| AC-007 | Test | Automated Go test: create feature in `reviewing` with all tasks done, call close-out, assert `done` status and cascade |
| AC-008 | Test | Automated Go test: create feature with non-terminal task, call close-out, assert blocked with `next_action` |
| AC-009 | Test | Automated Go test: create feature with tasks done but no review report, call close-out, assert `next_action` for missing report |
| AC-010 | Test | Automated Go test: create feature with ready tasks, call dispatch, assert tasks are `active` and handoff prompts present |
| AC-011 | Inspection | Code review: verify `develop(action: "dispatch")` handler does not import or call any spawn-agent infrastructure |
| AC-012 | Test | Automated Go test: create mixed task states, call dispatch, assert all four response fields correctly populated |
| AC-013 | Test | Automated Go test: create batch with mixed feature states, call snapshot, assert per-feature status and `next_action` |
| AC-014 | Test | Automated Go test: call bootstrap, capture side effects, compare to side effects from equivalent individual `transition` calls |
| AC-015 | Test | Run existing `doc_tool_test.go` and `entity_tool_test.go` suites — all must pass with zero changes to test code |
