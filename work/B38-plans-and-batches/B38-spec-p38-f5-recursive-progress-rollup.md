# P38-F5: Recursive Progress Rollup — Specification

| Field   | Value                                                                    |
|---------|--------------------------------------------------------------------------|
| Date    | 2026-04-28T01:19:36Z                                                     |
| Status | approved |
| Author  | spec-author                                                              |
| Feature | FEAT-01KQ7YQKMCM6T                                                       |
| Design  | `work/design/meta-planning-plans-and-batches.md` — §5                    |

---

## Overview

This specification defines recursive progress rollup for the plan and batch entity types,
as described in the P38 design document
`work/design/meta-planning-plans-and-batches.md` (§5 "Progress rollup").

The current `ComputePlanRollup` function aggregates progress by summing task estimates
across all features in a plan. After the entity rename (P38-F3), the batch entity inherits
this logic unchanged. A new `ComputeBatchRollup` is introduced (identical to the current
`ComputePlanRollup`), and `ComputePlanRollup` is reimplemented to aggregate progress
recursively across child batches and child plans.

---

## Scope

**In scope:**

- Rename the current `ComputePlanRollup` to `ComputeBatchRollup` with identical logic
- Implement a new `ComputePlanRollup` that recursively aggregates:
  - Direct child batch progress via `ComputeBatchRollup`
  - Direct child plan progress via recursive `ComputePlanRollup`
- Define rollup struct types for batch-level rollup (`BatchRollup`) and plan-level
  rollup (`PlanRollup`)
- Ensure rollup excludes not-planned and duplicate tasks (same as current behaviour)
- Update the estimation service to handle both entity types
- Update the MCP estimate tool (`estimate_tool.go`) to use the correct rollup function
  for each entity type
- Handle standalone batches (no parent plan) — they roll up identically to current plans
- Unit tests for batch rollup and recursive plan rollup

**Explicitly excluded:**

- Status dashboard rendering of recursive plan progress (P38-F6)
- Progress display for project overview (P38-F6)
- Caching of rollup results (performance optimisation, out of scope)
- Rollup for the project singleton (the project is not an entity with a lifecycle)

---

## Functional Requirements

### Batch Rollup

- **REQ-001:** The system MUST provide a `ComputeBatchRollup(batchID)` function whose
  behaviour is identical to the current `ComputePlanRollup(planID)`. It computes:
  - Task estimates summed across all features belonging to the batch
  - Progress summed across all done tasks in those features
  - Feature count, estimated feature count, and excluded task count
  - Not-planned and duplicate tasks excluded from totals

- **REQ-002:** The `BatchRollup` struct MUST have the same fields as the current
  `PlanRollup` struct. The field names use "batch" terminology where appropriate but the
  data shape is identical.

- **REQ-003:** For a standalone batch (no parent plan), `ComputeBatchRollup` MUST produce
  the same results that `ComputePlanRollup` would have produced for the same entity
  before the rename.

### Plan Rollup (Recursive)

- **REQ-004:** The system MUST provide a `ComputePlanRollup(planID)` function that
  recursively aggregates progress across the plan's descendants.

- **REQ-005:** For a plan with direct child batches, `ComputePlanRollup` MUST compute:
  - Total estimates = sum of all child batch totals (via `ComputeBatchRollup`)
  - Total progress = sum of all child batch progress values
  - Batch count, feature count, task counts aggregated from children

- **REQ-006:** For a plan with direct child plans, `ComputePlanRollup` MUST recursively
  aggregate:
  - Total estimates = sum of child plan totals (via recursive `ComputePlanRollup`)
  - Total progress = sum of child plan progress values

- **REQ-007:** For a plan with both child batches and child plans, `ComputePlanRollup`
  MUST aggregate both: the plan's total is the sum of all child batch totals plus all
  child plan totals.

- **REQ-008:** Recursion MUST be breadth-agnostic. A plan can have any combination of
  child plans and child batches at any depth, and the rollup correctly aggregates all
  descendant progress.

- **REQ-009:** A plan with no children (no child batches and no child plans) MUST return
  a zero-value `PlanRollup` with all counts and totals at zero.

- **REQ-010:** Recursive rollup MUST NOT double-count progress. Each descendant is
  counted exactly once at its immediate parent level, and the parent aggregates the
  child-level totals.

### Service Integration

- **REQ-011:** The estimation service (`internal/service/estimation.go`) MUST export both
  `ComputeBatchRollup` and `ComputePlanRollup` as distinct public methods on
  `EntityService`.

- **REQ-012:** The MCP estimate tool MUST dispatch to `ComputeBatchRollup` when the
  queried entity is a batch, and to `ComputePlanRollup` when the queried entity is a
  plan.

- **REQ-013:** The MCP status tool MUST use `ComputeBatchRollup` for batch dashboards
  and `ComputePlanRollup` for plan dashboards when computing progress statistics.

- **REQ-014:** The existing `ComputeFeatureRollup` function MUST remain unchanged.
  Feature-level rollup is not affected by this specification.

---

## Non-Functional Requirements

- **REQ-NF-001:** Recursive rollup MUST NOT cause infinite loops. Cycle detection at the
  entity level (P38-F2 REQ-015) prevents cycles in the plan tree; the rollup function
  MAY assume the tree is acyclic.

- **REQ-NF-002:** For a plan with N total descendants (batches + plans), rollup
  computation is O(N) in the number of entity lookups. Each descendant is visited
  exactly once.

- **REQ-NF-003:** Rollup computation MUST NOT modify any entity state. It is a read-only
  aggregation.

---

## Constraints

- `ComputeBatchRollup` is a rename-plus-relocate of the current `ComputePlanRollup`. The
  underlying algorithm (task aggregation, exclusion of not-planned/duplicate tasks) is
  NOT changed.
- `ComputePlanRollup` becomes recursive but the interface signature MUST remain
  compatible with existing callers. The return type (`PlanRollup`) may gain additional
  fields for child plan/batch counts but existing fields are preserved.
- Recursive rollup MUST NOT eagerly load all entities into memory. It processes children
  level by level, loading only the direct children at each step.
- The `PlanRollup` struct is additive — existing fields are retained; new fields (child
  plan count, child batch count) are added.

---

## Acceptance Criteria

**AC-001.** `ComputeBatchRollup` for a batch with two features, each having
  one task estimated at 3 points, one task done: returns `TaskTotal=6`, `Progress=3`,
  `FeatureCount=2`.

**AC-002.** `ComputeBatchRollup` excludes not-planned and duplicate tasks
  from all counts. A batch with one active task (3 points) and one not-planned task (5
  points) returns `TaskTotal=3`, `ExcludedTaskCount=1`.

**AC-003.** A standalone batch produces identical results to what
  `ComputePlanRollup` would have produced before the rename.

**AC-004.** Plan `P1` with child batch `B1` (total 8, progress 5) and child
  batch `B2` (total 5, progress 0): `ComputePlanRollup(P1)` returns `Total=13`,
  `Progress=5`.

**AC-005.** Plan `P1` with child plan `P2` (which has child batch `B3` with
  total 3, progress 3): `ComputePlanRollup(P1)` returns `Total=3`, `Progress=3`.

**AC-006.** Plan `P1` with child batch `B1` (total 5, progress 5) and child
  plan `P2` (which has child batch `B2` with total 3, progress 0):
  `ComputePlanRollup(P1)` returns `Total=8`, `Progress=5`.

**AC-007.** Three-level tree: P1 → P2 → B1 (total 3, progress 3).
  `ComputePlanRollup(P1)` returns `Total=3`, `Progress=3`.

**AC-008.** Plan with no children returns zero-value rollup (all counts and
  totals are zero).

**AC-009.** In the tree P1 → [B1(total 3), B2(total 3)], P2 → [B3(total 3)],
  `ComputePlanRollup(P1)` returns `Total=6` (B1 + B2, not B3 which belongs to P2).

**AC-010.** `estimation.ComputeBatchRollup` and
  `estimation.ComputePlanRollup` are both accessible methods on `EntityService`.

**AC-011.** `estimate(action: "query", entity_id: "B1-test")` uses
  `ComputeBatchRollup`. `estimate(action: "query", entity_id: "P1-test")` uses
  `ComputePlanRollup`.

**AC-012.** `status(id: "B1-test")` renders batch progress correctly.
  `status(id: "P1-test")` renders plan progress with recursive aggregation.

**AC-013.** `ComputeFeatureRollup` behaviour is unchanged by this feature.

---

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Test | Automated test: create batch with features/tasks, assert `ComputeBatchRollup` totals |
| AC-002 | Test | Automated test: batch with mixed task statuses, assert exclusions correct |
| AC-003 | Test | Automated test: standalone batch rollup equals legacy plan rollup output |
| AC-004 | Test | Automated test: plan with two child batches, assert aggregated totals |
| AC-005 | Test | Automated test: plan with child plan (which has child batch), assert deep aggregation |
| AC-006 | Test | Automated test: plan with mixed children (batches + plans), assert combined totals |
| AC-007 | Test | Automated test: three-level plan→plan→batch tree, assert full-depth aggregation |
| AC-008 | Test | Automated test: empty plan returns zero-value rollup |
| AC-009 | Test | Automated test: sibling plan does not include nephew batch in rollup |
| AC-010 | Inspection | Code review: verify both methods are exported on EntityService |
| AC-011 | Test | Automated test: MCP estimate query dispatches to correct rollup function per entity type |
| AC-012 | Test | Automated test: status dashboard renders correct progress for batch and plan |
| AC-013 | Test | Run existing `ComputeFeatureRollup` tests, assert all pass unchanged |

---

## Dependencies and Assumptions

- **P38-F3 (Batch Entity Rename):** `ComputeBatchRollup` replaces the current
  `ComputePlanRollup`. The batch entity must exist before batch rollup can be
  implemented.
- **P38-F2 (Plan Entity):** The plan entity's `parent` field enables the plan tree that
  recursive rollup traverses. `ListPlans(parentID)` is used to discover child plans.
- **Existing estimation service (`internal/service/estimation.go`):** The current
  `ComputePlanRollup` is renamed to `ComputeBatchRollup`. The new `ComputePlanRollup`
  is a fresh implementation.
- **Cycle prevention:** The plan entity's cycle detection (P38-F2 REQ-015) guarantees
  the plan tree is acyclic. The rollup function does not need its own cycle detection.
- **Not-planned and duplicate exclusion:** These task statuses are excluded from rollup
  exactly as they are today. No change to exclusion logic.
