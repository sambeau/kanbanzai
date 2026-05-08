| Field  | Value |
|--------|-------|
| Date   | 2026-05-08T11:48:16Z |
| Status | Draft |
| Author | architect |
| Feature | FEAT-01KR3MDSZKAFG — High-violation MCP rule invariants |
| Batch | B59 — Enforce high-violation rules as MCP invariants |

## Overview

This plan implements the requirements defined in
`work/B59-enforce-high-violation-rules-mcp-invariants/B59-F1-spec-high-violation-mcp-rule-invariants.md`.
It covers six tasks (T1–T6). It does not cover the P44 dispatch-implementation work, P56 bug
lifecycle gate enforcement, the B58 constraint card renderer, or the B62 runtime wrapper discovery
surfaces — those boundaries are explicitly excluded by the spec.

The feature translates five frequently-violated workflow rules into stable MCP-layer contracts:

| Code | Rule |
|------|------|
| INV-001 | Handoff-only dispatch — no direct `spawn_agent` composition |
| INV-002 | Registered-entity requirement — `next`/`handoff` refuse unknown IDs |
| INV-003 | Commit-before-task — `next` task-claim refuses orphaned workflow state |
| INV-004 | No shell reads of `.kbz/state/` — mandatory warning on all context surfaces |
| INV-005 | Artefact gate enforcement — gates are mandatory, never advisory |

P59 owns the invariant catalog, prompt-layer text, tool-description text, and prose
de-duplication. P44 owns the `dispatch_task` implementation that will remove `spawn_agent`
from the orchestrator's tool list at runtime. P56 owns the bug gate checks that INV-005
points to for bugs.

**Dependency on P44:** INV-001's full effect (removing `spawn_agent` from the orchestrator's
tool list) cannot be verified until P44's `dispatch_task` path is available. T5 removes
`spawn_agent` from `orchestrator.yaml` and updates the prose, but AC-002 cannot be marked
complete until the P44 tool is live. The tasks that don't depend on P44 (T1–T4, T6) can be
completed and verified independently.

## Task Breakdown

### Task 1: Define the invariant catalog package

- **Description:** Create `internal/invariants/catalog.go` defining the five invariant codes
  as exported string constants (INV-001 through INV-005), a `RefusalResponse` struct with
  fields `Code`, `Operation`, `Reason`, and `NextAction`, and a `Format` function that
  serialises a `RefusalResponse` to the JSON shape `{"error":{"code":...,"operation":...,"reason":...,"next_action":...}}`.
  The catalog is the single authoritative source of invariant codes; all other packages import
  from here.
- **Deliverable:** `internal/invariants/catalog.go` + `internal/invariants/catalog_test.go`
  (round-trip JSON format tests, byte-length upper bound test ≤ 1,200 bytes).
- **Depends on:** None.
- **Effort:** 3 story points (small new package).
- **Spec requirements:** REQ-001, REQ-002, REQ-009, REQ-NF-001, REQ-NF-002.

### Task 2: Enforce registered-entity invariant in `next` and `handoff`

- **Description:** Modify `internal/mcp/next_tool.go` and `internal/mcp/handoff_tool.go` so
  that whenever an entity ID is not found in the store (task not found, feature not found,
  plan not found), the tool returns a structured refusal using `invariants.Format` with
  `Code: invariants.INV002`, the refused operation, a reason, and a next valid action
  (e.g. "Create the entity with entity(action: \"create\") or list with entity(action: \"list\")" ).
  The existing ad-hoc `not_found` error strings in `handoffErrorJSON` and
  `nextClaimMode`/`nextResolveTaskID` are replaced with the catalog response.
- **Deliverable:** Updated `next_tool.go` and `handoff_tool.go`; all `not_found` refusals for
  unregistered IDs emit the INV-002 code in the four-field structured shape.
- **Depends on:** Task 1.
- **Effort:** 2 story points (targeted error-path edits in two files).
- **Spec requirements:** REQ-005, REQ-011.

### Task 3: Enforce orphaned-workflow-state invariant in `next` task-claim

- **Description:** Add a pre-claim dirty-state check to `nextClaimMode` in
  `internal/mcp/next_tool.go`. Before transitioning a task from `ready` to `active`, call a
  new helper (in `internal/git/dirty.go`) that runs
  `git status --porcelain -- .kbz/state/ .kbz/index/ .kbz/context/` and returns the list of
  affected files. If the list is non-empty, return a structured refusal using
  `invariants.Format` with `Code: invariants.INV003`, the affected file list in the `Reason`
  field, and "Commit or stash the listed files, then retry next" as the next action. The check
  is skipped on reclaim (`isReclaim == true`) for parity with the existing reclaim path.
- **Deliverable:** New `internal/git/dirty.go` + `internal/git/dirty_test.go`; updated
  `next_tool.go` with the pre-claim check; the reclaim path is unaffected.
- **Depends on:** Task 1.
- **Effort:** 3 story points (new git helper + integration into claim path).
- **Spec requirements:** REQ-006, REQ-011.

### Task 4: Add shell-read warning to task-context assembly surfaces

- **Description:** Inject a `workflow_state_warning` field into the structured context map
  produced by `nextContextToMap` in `internal/mcp/assembly.go`. The warning text must state
  that `.kbz/state/`, `.kbz/index/`, and `.kbz/context/` must not be read via shell/terminal
  tools and must point to the MCP workflow tools instead. The same warning must appear in the
  rendered prompt produced by `kbzctx.RenderPrompt` (add a standard "Invariants" section
  containing INV-004 text). Update the `handoff` and `next` tool description strings to
  include a one-sentence reference to INV-004.
- **Deliverable:** Updated `assembly.go` (new field in context map); updated
  `internal/context/render.go` or equivalent render path (new invariants section); updated
  tool description strings in `next_tool.go` and `handoff_tool.go`.
- **Depends on:** Task 1.
- **Effort:** 2 story points (additive text injection, no structural changes).
- **Spec requirements:** REQ-007.

### Task 5: Prose de-duplication in orchestrator role and orchestration skill

- **Description:** Replace duplicated long-form invariant prose in `.kbz/roles/orchestrator.yaml`
  and `.kbz/skills/orchestrate-development/SKILL.md` with short cross-references to invariant
  codes. Specifically:
  (a) Remove `spawn_agent` from the `tools` list in `orchestrator.yaml` and add a note that
      handoff-only dispatch is enforced by INV-001 (with reference to `dispatch_task` once P44
      is available).
  (b) Replace the "Manual Prompt Composition" anti-pattern long-form prose in
      `orchestrate-development/SKILL.md` with a one-paragraph reference to INV-001.
  (c) Add cross-references to INV-004 in the "Phase 1: Read the Dev-Plan" constraint block
      in place of the currently duplicated shell-read warning prose.
  (d) Verify that no removed long-form copy was the *only* copy of a rule — each removed
      paragraph must be replaced by a pointer to the invariant code. Artefact gate text
      must be phrased as mandatory (INV-005), not advisory.
- **Deliverable:** Updated `orchestrator.yaml`; updated `orchestrate-development/SKILL.md`;
  all removed prose replaced by invariant-code cross-references.
- **Depends on:** Task 1.
- **Effort:** 2 story points (text-only edits to two YAML/Markdown files).
- **Spec requirements:** REQ-003, REQ-010, REQ-008.

### Task 6: Invariant boundary tests

- **Description:** Write integration tests in `internal/mcp/` covering each hard invariant at
  the tool boundary and the shell-read warning at the rendered-surface boundary:
  - `TestInvariant_INV002_Next_UnregisteredTask`: call `next` with a nonexistent task ID,
    assert the response JSON contains `"code":"INV-002"`, `"operation"`, `"reason"`, and
    `"next_action"`.
  - `TestInvariant_INV002_Handoff_UnregisteredTask`: same for `handoff`.
  - `TestInvariant_INV003_Next_OrphanedState`: stub `git status` to return a dirty `.kbz/`
    file list; assert `next` claim refusal with `"code":"INV-003"` and the file list in the
    response.
  - `TestInvariant_INV004_ContextWarning_Next`: call `next` in claim mode, assert the context
    map contains a `workflow_state_warning` field with INV-004 reference text.
  - `TestInvariant_INV004_ContextWarning_Handoff`: call `handoff`, assert rendered prompt
    contains the INV-004 invariant section.
  - `TestInvariant_RefusalSize`: assert each structured refusal response body is ≤ 1,200 bytes.
- **Deliverable:** New `internal/mcp/invariant_boundary_test.go` (or equivalently named
  file); all six test cases pass.
- **Depends on:** Tasks 2, 3, 4, 5.
- **Effort:** 5 story points (six test cases with non-trivial stubs for git and entity store).
- **Spec requirements:** REQ-012.

## Dependency Graph

```
T1: Invariant catalog           (no dependencies)
T2: Registered-entity check     → depends on T1
T3: Orphaned-state check        → depends on T1
T4: Shell-read warning          → depends on T1
T5: Prose de-duplication        → depends on T1
T6: Invariant boundary tests    → depends on T2, T3, T4, T5
```

Parallel groups after T1: [T2, T3, T4, T5] — all four can run concurrently.

Critical path: T1 → T2 → T6 (or any of T1 → T{2,3,4,5} → T6; all equal length).

T6 is the integration gate — it cannot start until all four parallel tasks complete.

**Note on P44 dependency:** INV-001 (handoff-only dispatch) is documented in T5's prose
changes. The corresponding tool-list change in `orchestrator.yaml` that removes `spawn_agent`
is also part of T5, but the full AC-002 test (asserting `spawn_agent` is absent from resolved
tools) is deferred to T6's test and is conditional on P44 availability. T6 should mark AC-002
tests as pending/skipped if `dispatch_task` is not yet registered, not fail the build.

## Interface Contracts

### `internal/invariants` package (new)

```go
// Invariant codes — stable, never renamed.
const (
    INV001 = "INV-001" // Handoff-only dispatch
    INV002 = "INV-002" // Registered-entity required
    INV003 = "INV-003" // Commit before task claim
    INV004 = "INV-004" // No shell reads of .kbz/state/
    INV005 = "INV-005" // Artefact gate enforcement (mandatory)
)

type RefusalResponse struct {
    Code       string // one of INV001–INV005
    Operation  string // the refused operation (e.g. "next task-claim")
    Reason     string // human-readable reason (≤ 400 bytes recommended)
    NextAction string // what the caller should do instead
}

// Format serialises r to the canonical JSON refusal shape.
// Total output is guaranteed to be ≤ 1,200 bytes when Reason ≤ 400 bytes.
func Format(r RefusalResponse) string
```

The `Format` output shape is:

```json
{
  "error": {
    "code": "INV-002",
    "operation": "next task-claim",
    "reason": "Task TASK-xxx is not registered in Kanbanzai workflow state.",
    "next_action": "Create the entity with entity(action: \"create\") or verify the ID."
  }
}
```

### `internal/git.CheckKbzDirty` (new function in `dirty.go`)

```go
// CheckKbzDirty returns the list of modified or untracked files under
// .kbz/state/, .kbz/index/, and .kbz/context/ in the given repo root.
// Returns (nil, nil) when the working tree is clean for those paths.
func CheckKbzDirty(repoRoot string) ([]string, error)
```

Called by `nextClaimMode` before task dispatch. The function must not be called on reclaim
paths (`isReclaim == true`). It is a package-level variable (`var checkKbzDirtyFunc`) in
`next_tool.go` so tests can inject a stub without a real git repo (same pattern as
`commitStateFunc` in `handoff_tool.go`).

### Context map field `workflow_state_warning` (additive)

`nextContextToMap` gains a new key:

```go
"workflow_state_warning": "INV-004: Do not read .kbz/state/, .kbz/index/, or .kbz/context/ via terminal or shell tools. Use MCP workflow tools (entity, doc, status, knowledge) instead."
```

This field is always present in the context map (not omitted when empty), consistent with
the `graph_project` field behaviour established by `TestNextContextToMap_GraphProjectEmpty`.

### Rendered prompt invariants section (additive)

`RenderPrompt` gains a `## Invariants` section injected before `## Instructions`:

```
## Invariants

The following workflow invariants apply to this task session:

- **INV-004** — Do not read `.kbz/state/`, `.kbz/index/`, or `.kbz/context/` directly
  via terminal or shell tools. Use MCP workflow tools (`entity`, `doc`, `status`,
  `knowledge`) instead.
```

The section is unconditionally injected (no feature flag). It is the last section before
instructions so that implementing agents see it without it burying the task summary.

### Ownership boundaries preserved

- `dispatch_task` wiring and prompt-assembly gate: **P44 owns** — T5 references it.
- Bug gate check logic: **P56 owns** — INV-005 catalog entry points to P56 without
  reimplementing it. T5's prose changes ensure gate text is mandatory, not advisory.

## Traceability Matrix

| Acceptance Criterion | Requirement(s) | Verification Method | Producing Task |
|----------------------|----------------|---------------------|----------------|
| AC-001: catalog has 5 invariants with stable codes | REQ-001, REQ-002 | Inspection / unit test | T1 |
| AC-002: spawn_agent absent from orchestrator; dispatch uses pipeline | REQ-003, REQ-004 | Test (pending P44) | T5, T6 |
| AC-003: next/handoff refuse unregistered ID with INV-002 | REQ-005, REQ-011 | Integration test | T2, T6 |
| AC-004: next refuses orphaned kbz state with INV-003 + file list | REQ-006, REQ-011 | Integration test | T3, T6 |
| AC-005: context surfaces include INV-004 warning | REQ-007 | Test (rendered surface) | T4, T6 |
| AC-006: feature/bug transitions without artefacts refuse | REQ-008 | Integration test (existing gates) | T5 (text alignment) |
| AC-007: handoff-only invariant has no override path | REQ-009 | Inspection / test | T1, T5 |
| AC-008: orchestrator.yaml and skill files de-duplicated | REQ-010 | Inspection | T5 |
| AC-009: all refusals include code, operation, reason, next_action | REQ-011 | Test | T1, T2, T3, T6 |
| AC-010: boundary tests for each invariant | REQ-012 | Test run | T6 |
| AC-011: refusal bodies ≤ 1,200 bytes | REQ-NF-002 | Test (byte measurement) | T1, T6 |
