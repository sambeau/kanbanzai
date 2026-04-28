# P38-F4: Feature Display IDs and Document Inheritance — Dev Plan

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-04-28T01:45:03Z           |
| Status | approved |
| Author | architect                      |

---

## Overview

This dev-plan covers the two changes specified for FEAT-01KQ7YQKHK2GV: switching
feature display IDs from the plan-scoped `P{n}-F{n}` to the batch-scoped `B{n}-F{n}`,
and extending the document gate lookup chain from three levels to four to support
plan-level document inheritance through the batch→plan parent hierarchy. Both changes
follow from the P38 batch entity rename (P38-F3) and the plan entity data model
(P38-F2).

## Scope

This plan implements the requirements defined in
`work/P38-plans-and-batches/P38-spec-p38-f4-feature-display-ids-doc-inheritance.md`
(Feature FEAT-01KQ7YQKHK2GV). It covers two changes:

1. **Feature display IDs** (REQ-001–REQ-006): Switch from `P{n}-F{n}` to `B{n}-F{n}`,
   deriving the batch number from the parent batch entity's ID prefix.
2. **Document gate inheritance** (REQ-007–REQ-013): Extend the document gate lookup
   from three levels to four — feature → batch → grandparent plan.

This plan does **not** cover:

- Migration of existing feature records from `P{n}-F{n}` to `B{n}-F{n}` (P38-F8)
- Backward-compatible resolution of legacy `P{n}-F{n}` display IDs
- Worktree or document filename renaming
- Denormalisation of display IDs into state files

---

## Task Breakdown

### Task 1: Feature Display IDs with B-Prefix

- **Description:** Update feature display ID computation to use the batch prefix (`B`)
  instead of the plan prefix (`P`). Features created under a batch produce display IDs
  in `B{n}-F{m}` format using the batch's `next_feature_seq` counter.
- **Deliverable:** Patched `internal/service/entities.go` (`CreateFeature`),
  `internal/service/plans.go` (`AllocateFeatureDisplayIDInPlan` → renamed), and
  `internal/service/display_id_test.go` (new test cases for B-prefix format).
- **Depends on:** P38-F3 (batch entity, `B{n}-{slug}` IDs, `next_feature_seq` on batch).
- **Effort:** medium
- **Spec requirement:** REQ-001, REQ-002, REQ-003, REQ-004, REQ-005, REQ-006
- **Acceptance criteria:** AC-001, AC-002, AC-003, AC-004, AC-005, AC-006

### Task 2: Four-Level Document Gate Inheritance

- **Description:** Extend the document gate evaluator (`eval_documents.go`) from
  three to four lookup levels. After checking feature-owned and parent batch documents,
  resolve the batch's `parent` field to find the grandparent plan and include
  plan-owned documents in the gate check. Add the necessary entity lookup capability
  to the gate package's `EntityService` interface.
- **Deliverable:** Patched `internal/gate/eval_documents.go`, `internal/gate/evaluator.go`
  (interface extension), `internal/mcp/entity_tool.go` (`buildGateEvalContext` update),
  and `internal/gate/eval_documents_test.go` (new test cases for level 4).
- **Depends on:** P38-F3 (batch entity with `parent` field referencing plan ID).
- **Effort:** medium
- **Spec requirement:** REQ-007, REQ-008, REQ-009, REQ-010, REQ-011, REQ-012,
  REQ-NF-001, REQ-NF-002
- **Acceptance criteria:** AC-007, AC-008, AC-009, AC-010, AC-011, AC-012, AC-013

### Task 3: Integration Verification

- **Description:** Write integration tests verifying the two features work together
  end-to-end. Test feature creation under batches with and without parent plans,
  verify MCP tool responses show `B{n}-F{n}`, and exercise the full four-level
  gate lookup in realistic scenarios.
- **Deliverable:** New integration tests in `internal/gate/integration_test.go`
  and `internal/service/display_id_test.go` (extensions), plus verification that
  all 13 acceptance criteria pass.
- **Depends on:** Task 1, Task 2
- **Effort:** small
- **Spec requirement:** All REQs (cross-cutting verification)
- **Acceptance criteria:** AC-001 through AC-013 (verification pass)

---

## Dependency Graph

```
Task 1 (Feature Display IDs)     — no dependencies on Task 2
Task 2 (Four-Level Gate)         — no dependencies on Task 1
Task 3 (Integration Verification) → depends on Task 1, Task 2

Parallel groups: [Task 1, Task 2]
Critical path: Task 1 → Task 3 (or Task 2 → Task 3, both equal length)
```

Both Task 1 and Task 2 are independent — display ID formatting and gate evaluation
touch different packages (`internal/service/` vs `internal/gate/`). They can be
implemented in parallel by separate sub-agents. Task 3 integrates and verifies
both together.

---

## Interface Contracts

### Batch entity shape (from P38-F3)

Task 1 and Task 2 both consume the batch entity. After P38-F3, the batch entity
exposes:

- `id`: string in `B{n}-{slug}` format (e.g. `B24-auth-system`)
- `parent`: optional string referencing a plan ID (e.g. `P1-platform`)
- `next_feature_seq`: int counter for feature display ID allocation
- `Model.ParseBatchID(id)` → `(prefix, number, slug)` for extracting the batch number

### Gate EntityService interface extension

Task 2 extends `gate.EntityService` with a `GetEntity` method:

```go
type EntityService interface {
    List(entityType string) ([]EntityResult, error)
    GetEntity(entityType string, id string) (*EntityResult, error) // new
}
```

The `gateEntityAdapter` in `internal/mcp/entity_tool.go` delegates to the concrete
`service.EntityService`. The concrete service already has entity lookup methods
(`Get`, `GetPlan`, `GetBatch`); the adapter maps `GetEntity("batch", id)` to the
appropriate call.

### buildGateEvalContext — no signature change

The `buildGateEvalContext` function signature remains unchanged. The four-level
lookup is handled internally by `evalOneDocument` using the `EntitySvc` available
on the context. No new fields are added to `PrereqEvalContext`.

---

## Traceability Matrix

| Task | REQs Covered | ACs Covered |
|------|-------------|-------------|
| Task 1: Feature Display IDs | REQ-001, REQ-002, REQ-003, REQ-004, REQ-005, REQ-006 | AC-001, AC-002, AC-003, AC-004, AC-005, AC-006 |
| Task 2: Four-Level Gate | REQ-007, REQ-008, REQ-009, REQ-010, REQ-011, REQ-012, REQ-NF-001, REQ-NF-002 | AC-007, AC-008, AC-009, AC-010, AC-011, AC-012, AC-013 |
| Task 3: Integration | All REQs (verification) | AC-001–AC-013 (pass confirmation) |

---

## Risk Assessment

### Risk: P38-F3 batch entity not fully implemented

- **Probability:** medium
- **Impact:** high — both Task 1 and Task 2 depend on the batch entity existing
  with `B{n}-{slug}` IDs, `next_feature_seq`, and a `parent` field.
- **Mitigation:** Verify P38-F3 is complete before starting. If P38-F3 is in
  progress, coordinate on the interface contract (batch struct shape, ID format).
- **Affected tasks:** Task 1, Task 2

### Risk: EntityService interface extension causes ripple in adapters

- **Probability:** low
- **Impact:** medium — extending `EntityService` in the gate package requires
  updating `gateEntityAdapter` in `entity_tool.go` and all mock implementations
  in tests.
- **Mitigation:** Add a minimal `GetEntity` method. The adapter already wraps
  the concrete service; adding one delegation method is low-risk.
- **Affected tasks:** Task 2

### Risk: Feature display ID computation for features with plan parent (not batch)

- **Probability:** low
- **Impact:** medium — after P38-F3, all features should have batch parents.
  But during transition, some features may still reference plan IDs.
- **Mitigation:** REQ-005 and AC-005 already scope this to new features only.
  The display ID computation reads the parent's type to decide prefix.
- **Affected tasks:** Task 1

### Risk: Gate lookup for deleted/missing plan

- **Probability:** low
- **Impact:** low — a batch with a dangling `parent` reference is an edge case.
- **Mitigation:** REQ-NF-002 requires graceful skip. Implementation logs a
  warning and returns batch-level result. Covered by AC-013.
- **Affected tasks:** Task 2

---

## Verification Approach

| Acceptance Criterion | Verification Method | Producing Task |
|---------------------|--------------------|----------------|
| AC-001: Feature under batch produces `B24-F1` | Unit test | Task 1 |
| AC-002: `next_feature_seq` increments correctly | Unit test | Task 1 |
| AC-003: `FormatFullDisplay` uses `B{n}-F{n}` | Unit test | Task 1 |
| AC-004: MCP `entity get` shows `B{n}-F{n}` | Unit test | Task 1 |
| AC-005: Existing features retain current display ID | Unit test | Task 1 |
| AC-006: Standalone batch feature uses `B` prefix | Unit test | Task 1 |
| AC-007: Plan→batch→feature spec gate resolution | Unit test | Task 2 |
| AC-008: Four-level lookup for design, spec, dev-plan gates | Unit test | Task 2 |
| AC-009: Standalone batch gates at batch level, no error | Unit test | Task 2 |
| AC-010: Plan design doc satisfies grandchild feature gate | Unit test | Task 2 |
| AC-011: Nearest level precedence (feature > batch > plan) | Unit test | Task 2 |
| AC-012: Max three state store lookups (code review) | Inspection | Task 2 |
| AC-013: Dangling parent reference gates correctly | Unit test | Task 2 |
| All ACs: End-to-end workflow | Integration test | Task 3 |
