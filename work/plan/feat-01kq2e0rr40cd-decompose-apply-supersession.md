# Implementation Plan: Decompose Apply Supersession

| Field   | Value                                                                              |
|---------|------------------------------------------------------------------------------------|
| Date    | 2026-04-25                                                                         |
| Status | approved |
| Feature | FEAT-01KQ2E0RR40CD (decompose-apply-supersession)                                  |
| Spec    | `work/spec/feat-01kq2e0rr40cd-decompose-apply-supersession.md`                     |
| Plan    | P34-agent-workflow-ergonomics                                                      |

---

## Overview

This plan implements the requirements defined in
`work/spec/feat-01kq2e0rr40cd-decompose-apply-supersession.md`. It covers the
supersession pass inserted at the start of `decomposeApply` in
`internal/mcp/decompose_tool.go`, the `superseded_count` and `warning` response
fields, and the unit tests that verify idempotency and the protected-status
partition.

It does **not** cover:
- Changes to `decompose(action: propose)` or `decompose(action: review)`.
- Changes to Pass 1 (task creation) or Pass 2 (dependency wiring) beyond the
  insertion point.
- Dashboard filtering or display of `not-planned` tasks.

---

## Task Breakdown

### Task 1: Implement the supersession pass and response fields

- **Description:** Insert a supersession pass at the start of `decomposeApply`
  (before Pass 1). List all tasks for the feature, partition them into
  supersedable (`queued`) and protected (all other statuses), transition
  supersedable tasks to `not-planned` via `entitySvc.UpdateStatus`, and collect
  counts. Add `superseded_count` (always present) and `warning` (when any
  `active` or `needs-rework` tasks are preserved) to the response map.
- **Deliverable:** Modified `internal/mcp/decompose_tool.go` with a
  `supersessionPass` helper function called at the start of `decomposeApply`.
- **Depends on:** None.
- **Effort:** Small.
- **Spec requirements:** FR-001, FR-002, FR-003, FR-004, FR-005, FR-006,
  FR-007, FR-008, FR-009, FR-010, NFR-001, NFR-002.

### Task 2: Unit tests

- **Description:** Add unit tests covering the supersession pass: all-queued
  supersession, zero-tasks no-op, mixed protected/supersedable partition, active
  task warning, ready task protection, idempotency over three apply calls.
- **Deliverable:** New or extended test cases in
  `internal/mcp/decompose_tool_test.go` (or `internal/service/decompose_test.go`
  if the supersession helper is factored to the service layer).
- **Depends on:** Task 1.
- **Effort:** Small.
- **Spec requirements:** AC-001 through AC-008.

---

## Dependency Graph

```
Task 1 (supersession pass + response fields)  ──→  Task 2 (tests)
```

Sequential. Task 2 cannot be written until the `decomposeApply` handler shape
is settled (pass structure, response keys). No parallelism available.

**Critical path:** Task 1 → Task 2.

---

## Risk Assessment

### Risk: `UpdateStatus` rejects `queued → not-planned`

- **Probability:** Low.
- **Impact:** High — the supersession pass cannot proceed.
- **Mitigation:** Before writing, verify that the task lifecycle state machine
  in `internal/validate/` permits `queued → not-planned`. If the transition is
  not currently allowed, it must be added as part of this task. Check
  `validate.NextStates(EntityTask, "queued")` before implementing.
- **Affected tasks:** Task 1.

### Risk: Entity list performance during supersession pass

- **Probability:** Low (P29 cache is in place).
- **Impact:** Low — degradation is O(n) without cache, not incorrect.
- **Mitigation:** Use the same `entitySvc.ListTasksForFeature`-equivalent that
  `decomposeApply` already calls, so no new list path is introduced. The P29
  cache applies uniformly.
- **Affected tasks:** Task 1.

### Risk: Supersession pass commits intermediate state if Pass 1 fails

- **Probability:** Low.
- **Impact:** Medium — superseded tasks become `not-planned` but the new task
  set is not created. ASM-003 in the spec acknowledges this; callers must retry.
- **Mitigation:** Document the behaviour in a code comment. Do not attempt
  rollback — the spec explicitly accepts this outcome (NFR-001).
- **Affected tasks:** Task 1, Task 2 (test must assert the non-rollback
  behaviour is observable and not a bug).

---

## Traceability Matrix

| Acceptance Criterion | Verification Method | Producing Task |
|----------------------|--------------------:|----------------|
| AC-001: 5 queued tasks → superseded_count=5 | Unit test | Task 2 |
| AC-002: No existing tasks → superseded_count=0, no warning | Unit test | Task 2 |
| AC-003: 2 done + 3 queued → done preserved, 3 superseded | Unit test | Task 2 |
| AC-004: 1 active + 3 queued → 3 superseded, warning present | Unit test | Task 2 |
| AC-005: 1 ready + 0 queued → ready preserved, superseded_count=0 | Unit test | Task 2 |
| AC-006: 1 active → Pass 1 not blocked, warning present | Unit test | Task 2 |
| AC-007: 3× apply → exactly one queued set after final call | Unit test | Task 2 |
| AC-008: 2 needs-rework → warning references 2 in-progress | Unit test | Task 2 |

**Build verification** (run after both tasks complete):

```
go build ./...
go test ./internal/mcp/...
go test -race ./internal/mcp/...
go vet ./...
```

---

## Interface Contracts

`decomposeApply` currently returns a `map[string]any` response. The supersession
pass adds two new top-level keys to that map:

| Key | Type | Always present |
|-----|------|----------------|
| `superseded_count` | `int` | Yes — 0 when nothing was superseded |
| `warning` | `string` | No — omitted when no in-progress tasks detected |

No existing response keys are removed or renamed. Callers that do not read the
new keys continue to work correctly.

The supersession helper function signature (unexported, may be inlined):

```go
// supersessionResult holds the outcome of the pre-apply supersession pass.
type supersessionResult struct {
    supersededCount    int
    inProgressWarning  string // empty when no in-progress tasks
}

func runSupersessionPass(ctx context.Context, featureID string, entitySvc *service.EntityService) supersessionResult
```

`runSupersessionPass` is called at the start of `decomposeApply`, before Pass 1
creates any new tasks. Its results are merged into the final response map after
Pass 2 completes.