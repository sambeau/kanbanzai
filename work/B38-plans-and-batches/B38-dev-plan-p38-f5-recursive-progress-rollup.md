| Field  | Value                                                            |
|--------|------------------------------------------------------------------|
| Date   | 2026-04-28T01:48:14Z                                            |
| Status | Draft                                                            |
| Author | architect                                                        |

# P38-F5: Recursive Progress Rollup — Implementation Plan

## Scope

This plan implements the requirements defined in
`work/P38-plans-and-batches/P38-spec-p38-f5-recursive-progress-rollup.md`
(REQ-001 through REQ-014, REQ-NF-001 through REQ-NF-003). It covers the
rename of `ComputePlanRollup` to `ComputeBatchRollup`, the new recursive
`ComputePlanRollup`, integration with the MCP estimate and status tools, and
comprehensive unit tests.

This plan does **not** cover:
- Status dashboard rendering of recursive plan progress (P38-F6)
- Progress display for project overview (P38-F6)
- Caching of rollup results
- Rollup for the project singleton

## Task Breakdown

### Task 1: Rename ComputePlanRollup to ComputeBatchRollup with BatchRollup struct

- **Description:** Rename the existing `ComputePlanRollup` function to
  `ComputeBatchRollup` (logic unchanged). Introduce a `BatchRollup` struct
  with the same fields as the current `PlanRollup`. Update internal callers
  in `estimate_tool.go` and `status_tool.go` where they currently call
  `ComputePlanRollup` for plan entities — these callers will be fully
  reconciled in Task 3 after the new `ComputePlanRollup` exists.
- **Deliverable:** `internal/service/estimation.go` — renamed function,
  `BatchRollup` struct. `internal/mcp/estimate_tool.go` — temporary
  compilation fix. `internal/mcp/status_tool.go` — temporary compilation fix.
- **Depends on:** P38-F3 (batch entity must exist in the model and entity
  inference). None within this feature.
- **Effort:** small
- **Spec requirement:** REQ-001, REQ-002, REQ-003

### Task 2: Add child-count fields to PlanRollup, implement recursive ComputePlanRollup

- **Description:** Add `ChildBatchCount` and `ChildPlanCount` fields to the
  `PlanRollup` struct. Implement a new `ComputePlanRollup` that recursively
  aggregates:
  - Direct child batches via `ComputeBatchRollup`
  - Direct child plans via recursive `ComputePlanRollup`
  - Returns zero-value rollup for a plan with no children
  - Uses `EntityService.List("batch")` filtered by `parent == planID` and
    `EntityService.ListPlans(PlanFilters{})` filtered by `parent == planID`
    (or `List("plan")` if plans have a `parent` field in state).
- **Deliverable:** `internal/service/estimation.go` — updated `PlanRollup`
  struct, new `ComputePlanRollup` function.
- **Depends on:** Task 1 (needs `ComputeBatchRollup` to exist), P38-F2 (plan
  entity with `parent` field for tree discovery).
- **Effort:** medium
- **Spec requirement:** REQ-004, REQ-005, REQ-006, REQ-007, REQ-008, REQ-009,
  REQ-010, REQ-NF-001, REQ-NF-002, REQ-NF-003

### Task 3: Wire estimate and status tools for batch and plan dispatch

- **Description:** Update `estimateQueryAction` in `estimate_tool.go` to:
  - Add a `"batch"` case that calls `ComputeBatchRollup`
  - Update the `"plan"` case to use the new recursive `ComputePlanRollup`
  - Update the rollup response shape for plans to include `child_batch_count`
    and `child_plan_count`
  - Update `entityInferType` and `entityKindFromType` if batch IDs use a new
    prefix (e.g. `B1-...`)
  - Update `status_tool.go` plan dashboard to call the correct rollup function
    and include child count fields in `planDashboard`
- **Deliverable:** `internal/mcp/estimate_tool.go`, `internal/mcp/entity_tool.go`
  (entityInferType update), `internal/mcp/status_tool.go`.
- **Depends on:** Task 1, Task 2
- **Effort:** medium
- **Spec requirement:** REQ-011, REQ-012, REQ-013

### Task 4: Unit tests for ComputeBatchRollup

- **Description:** Write unit tests for `ComputeBatchRollup` covering:
  - Basic aggregation: batch with features/tasks, correct totals
  - Not-planned and duplicate task exclusion
  - Standalone batch produces identical results to legacy plan rollup
  - Empty batch (no features) returns zero-value rollup
- **Deliverable:** `internal/service/estimation_rollup_test.go` — new test
  functions.
- **Depends on:** Task 1
- **Effort:** medium
- **Spec requirement:** AC-001, AC-002, AC-003

### Task 5: Unit tests for recursive ComputePlanRollup

- **Description:** Write unit tests for the new `ComputePlanRollup` covering:
  - Plan with two child batches → correct aggregation
  - Plan with child plan (deep tree) → correct deep aggregation
  - Plan with mixed children (batches + plans) → combined totals
  - Three-level tree: P1 → P2 → B1 → correct full-depth aggregation
  - Plan with no children → zero-value rollup
  - Sibling plan does not include nephew batch in rollup
  - Verify `ComputeFeatureRollup` behaviour is unchanged
- **Deliverable:** `internal/service/estimation_rollup_test.go` — new test
  functions.
- **Depends on:** Task 2
- **Effort:** medium
- **Spec requirement:** AC-004 through AC-013

## Dependency Graph

```
Task 1 (rename → ComputeBatchRollup + BatchRollup)
  ├──> Task 2 (recursive ComputePlanRollup)
  │     ├──> Task 3 (wire estimate + status tools)
  │     └──> Task 5 (unit tests: recursive plan rollup)
  └──> Task 4 (unit tests: batch rollup)
```

```
Parallel groups: [Task 2, Task 4]  (both depend only on Task 1)
                 [Task 3, Task 5]  (both depend only on Task 2)
Critical path: Task 1 → Task 2 → Task 3
```

## Risk Assessment

### Risk: P38-F3 (batch entity) not yet complete
- **Probability:** medium
- **Impact:** high — `ComputeBatchRollup` needs `s.List("batch")` and batch
  entity type inference to work.
- **Mitigation:** Task 1 can implement `ComputeBatchRollup` against the
  existing `List("feature")` API with a `parent == batchID` filter — the
  batch entity's existence is only needed for `entityInferType`. If batch
  IDs use a `B1-...` prefix, add that to `entityInferType` in Task 3. If
  P38-F3 isn't merged yet, Task 3 can be deferred.
- **Affected tasks:** Task 1, Task 3

### Risk: Plan entities may not use a `parent` field in state
- **Probability:** low
- **Impact:** medium — recursive `ComputePlanRollup` needs to discover child
  plans via a parent reference.
- **Mitigation:** The P38-F2 spec defines a `parent` field on plan entities.
  If the field name differs, adjust the filter in Task 2. The
  `ListPlans(PlanFilters{})` API already exists and can be filtered
  in-memory.
- **Affected tasks:** Task 2

### Risk: Interface contract break for existing callers
- **Probability:** low
- **Impact:** medium — `ComputePlanRollup` is called in `estimate_tool.go`,
  `status_tool.go`, and tests.
- **Mitigation:** The new `ComputePlanRollup` retains the same signature
  `(planID string) (PlanRollup, error)`. The `PlanRollup` struct is additive
  (new fields appended). Existing callers that don't use the new fields are
  unaffected. Tests in `estimation_rollup_test.go` may need updating to
  account for the renamed function.
- **Affected tasks:** Task 2, Task 3

## Verification Approach

| Acceptance Criterion | Verification Method | Producing Task |
|---------------------|--------------------| ---------------|
| AC-001: Batch rollup basic aggregation | Unit test | Task 4 |
| AC-002: Batch rollup excludes not-planned/duplicate | Unit test | Task 4 |
| AC-003: Standalone batch matches legacy plan rollup | Unit test | Task 4 |
| AC-004: Plan with two child batches | Unit test | Task 5 |
| AC-005: Plan with child plan (deep tree) | Unit test | Task 5 |
| AC-006: Plan with mixed children | Unit test | Task 5 |
| AC-007: Three-level tree | Unit test | Task 5 |
| AC-008: Empty plan returns zero-value rollup | Unit test | Task 5 |
| AC-009: Sibling plan isolation | Unit test | Task 5 |
| AC-010: Both methods exported on EntityService | Code review | Task 2 |
| AC-011: Estimate dispatch per entity type | Unit test (or integration) | Task 3 |
| AC-012: Status dashboard renders correct progress | Manual inspection | Task 3 |
| AC-013: ComputeFeatureRollup unchanged | Run existing tests | Task 5 |