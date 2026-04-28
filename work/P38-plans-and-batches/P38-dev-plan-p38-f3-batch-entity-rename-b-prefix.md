# P38-F3: Batch Entity Rename and B-Prefix IDs — Dev Plan

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-04-28T01:44:45Z           |
| Status | Draft                          |
| Author | architect                      |

---

## Scope

This plan implements the requirements defined in
`work/P38-plans-and-batches/P38-spec-p38-f3-batch-entity-rename-b-prefix.md`
(FEAT-01KQ7YQKEEHFY/spec-p38-spec-p38-f3-batch-entity-rename-b-prefix).
It covers the rename of the existing `Plan` entity to `Batch` across the
model, service, MCP tool, kbzschema, and CLI layers, plus the introduction
of `B{n}-{slug}` IDs and backward-compatible resolution of legacy `P{n}`
references.

**In scope:**
- Model: `Plan` → `Batch`, `PlanStatus` → `BatchStatus`, `EntityPlan` → `EntityBatch`
- Service: all plan service methods renamed, state directory `plans/` → `batches/`
- MCP tools: `entity_tool.go` supports `type: "batch"` (canonical) with `type: "plan"` synonym
- MCP tools: `status_tool.go` renders "Batch" labels
- kbzschema: export `Batch`, alias `Plan`, update Reader
- CLI: update entity commands, move command, estimate command
- Legacy `P{n}` → `B{n}` resolution with INFO-level deprecation logging
- All tests updated to reference Batch

**Out of scope (deferred to P38-F8 and P38-F4):**
- Migration of existing `.kbz/state/plans/` files to `.kbz/state/batches/`
- Migration of worktree folders
- Feature display ID `B{n}-F{n}` updates (P38-F4 handles this)
- Document reference updates in existing document records
- Prefix deprecation/removal from config
- Status dashboard plan-tree rendering (P38-F6)

## Task Breakdown

### Task 1: Model and Validation Layer Rename

- **Description:** Rename `Plan` struct to `Batch`, `PlanStatus` to `BatchStatus`
  with all constants, add `IsBatchID`/`ParseBatchID` functions, add
  `EntityKindBatch` with "plan" as backward-compatible synonym, rename
  `EntityPlan` → `EntityBatch` in the validate package along with all
  lifecycle transition tables.
- **Deliverable:** Updated `internal/model/entities.go` with `Batch` struct and
  `BatchStatus` type; updated `internal/validate/lifecycle.go` and
  `internal/validate/entity.go` with `EntityBatch`; updated `internal/validate/health.go`.
- **Depends on:** None
- **Effort:** medium
- **Spec requirement:** REQ-001, REQ-002, REQ-003, REQ-004

### Task 2: Service Layer Rename

- **Description:** Rename all service methods (`CreatePlan` → `CreateBatch`,
  `GetPlan` → `GetBatch`, `UpdatePlan` → `UpdateBatch`,
  `UpdatePlanStatus` → `UpdateBatchStatus`, `ListPlans` → `ListBatches`,
  `AllocateFeatureDisplayIDInPlan` → `AllocateFeatureDisplayIDInBatch`).
  Rename input/output types (`CreatePlanInput` → `CreateBatchInput`,
  `UpdatePlanInput` → `UpdateBatchInput`, `PlanFilters` → `BatchFilters`).
  Update state directory from `"plans"` to `"batches"` in `loadPlan`/`listPlanIDs`/`writePlan`.
  Add legacy `P{n}` → `B{n}` resolution in `GetBatch` with INFO-level
  deprecation logging. Update feature display ID prefix from `P` to `B`.
  Update `entityDirectory`/`entityFileName` in storage for batch naming.
- **Deliverable:** Updated `internal/service/plans.go` (renamed to use batch
  terminology); updated `internal/service/entities.go` (`HealthCheck` and
  `Get` code paths); updated `internal/storage/entity_store.go`
  (`entityFileName` for `EntityKindBatch`).
- **Depends on:** Task 1
- **Effort:** large
- **Spec requirement:** REQ-005, REQ-006, REQ-007, REQ-008, REQ-009, REQ-010, REQ-011, REQ-012, REQ-013

### Task 3: MCP Tools and CLI Updates

- **Description:** Update `entity_tool.go` to accept `type: "batch"` as canonical
  with `type: "plan"` as deprecated synonym. Add batch variants to all
  entity action switch cases (create, get, list, update, transition).
  Update `status_tool.go` to use "Batch" terminology in dashboard labels,
  `idTypePlan` → `idTypeBatch`, and `inferIDType` to use `IsBatchID`.
  Update CLI commands (`entity_cmd.go`, `move_cmd.go`, `estimate_cmd.go`)
  to use "batch" terminology and support `type: "batch"`.
- **Deliverable:** Updated `internal/mcp/entity_tool.go` with batch type support
  and backward-compat plan synonym; updated `internal/mcp/status_tool.go`
  with batch dashboard labels; updated CLI command files.
- **Depends on:** Task 2
- **Effort:** medium
- **Spec requirement:** REQ-008, REQ-014, REQ-015

### Task 4: kbzschema Package Update

- **Description:** Export `Batch` struct and `BatchStatus*` constants from
  `kbzschema/types.go`. Deprecate `Plan` struct as an alias for `Batch`
  with a `Deprecated` comment; deprecate `PlanStatus*` constants as aliases
  for `BatchStatus*`. Update `Reader` methods (`GetPlan` → `GetBatch`,
  `ListPlans` → `ListBatches`). Update schema generation in
  `kbzschema/schema.go`. Update `_testexternal/main.go` to exercise new
  `Batch` types.
- **Deliverable:** Updated `kbzschema/types.go` with `Batch` struct and
  deprecated `Plan` alias; updated `kbzschema/reader.go` with batch
  methods; updated `kbzschema/schema.go`; updated
  `_testexternal/main.go`.
- **Depends on:** Task 1
- **Effort:** medium
- **Spec requirement:** REQ-016 (AC-015)

### Task 5: Test Updates and Compilation Verification

- **Description:** Update all test files to reference `Batch` instead of `Plan`.
  This includes: `internal/service/plans_test.go`,
  `internal/service/plans_shortcut_test.go`, `internal/service/display_id_test.go`,
  `internal/service/entities_test.go`, `internal/service/advance_test.go`,
  `internal/validate/lifecycle_test.go`, `internal/validate/health_test.go`,
  `internal/cache/cache_test.go`, `cmd/kanbanzai/main_test.go`,
  `cmd/kanbanzai/move_cmd_test.go`, `kbzschema/reader_test.go`.
  Verify `go build ./...` and `go test ./...` pass with zero failures.
  Verify AC-016 (external compilation via `_testexternal`).
- **Deliverable:** All test files updated; `go build ./...` succeeds;
  `go test ./...` succeeds; `_testexternal` compiles.
- **Depends on:** Task 2, Task 3, Task 4
- **Effort:** medium
- **Spec requirement:** REQ-NF-001, AC-001 through AC-016

## Dependency Graph

```
Task 1 (no dependencies)
Task 2 → depends on Task 1
Task 3 → depends on Task 2
Task 4 → depends on Task 1
Task 5 → depends on Task 2, Task 3, Task 4

Parallel groups: [Task 2, Task 4]  (both depend on Task 1)
                  [Task 3, Task 4]  (Task 3 needs Task 2, Task 4 only needs Task 1)

Critical path: Task 1 → Task 2 → Task 3 → Task 5
```

## Risk Assessment

### Risk: Test Breakage Cascade

- **Probability:** high
- **Impact:** medium — many test files reference `Plan` and a missed reference
  causes compilation failure.
- **Mitigation:** Systematic grep-driven update: `grep -rl "\.Plan\b\|PlanStatus\|EntityPlan\|model\.Plan"` 
  across the entire repository. Task 5 addresses all matches. Run `go build ./...` 
  after each task to catch missing references early.
- **Affected tasks:** Task 2, Task 3, Task 5

### Risk: Type Collision with New Plan Struct (P38-F2)

- **Probability:** medium
- **Impact:** high — P38-F2 introduces a new `Plan` struct. If F2 changes are
  not yet merged when this rename lands, the old `Plan` being renamed to
  `Batch` could conflict with F2's new `Plan`.
- **Mitigation:** This feature renames the old `Plan` in-place. The new `Plan`
  from F2 must use a distinct struct name or must be merged before this
  feature to avoid a compilation collision. Coordinate with P38-F2 state.
- **Affected tasks:** Task 1, Task 5

### Risk: Backward Compatibility Gap in Edge Cases

- **Probability:** low
- **Impact:** medium — legacy `P{n}` resolution might miss a code path (e.g.,
  `resolvePlanArg` in `move_cmd.go`, `statusTool` type inference) that
  continues to hard-code P-prefix checks.
- **Mitigation:** Exhaustive grep for `"plan"`, `"plans"`, `IsPlanID`, and
  `P{n}` patterns in non-test code after Tasks 1–4. The remaining
  references should only be: the backward-compat synonym, the deprecated
  kbzschema aliases, and the legacy resolution path itself.
- **Affected tasks:** Task 2, Task 3

### Risk: State Directory Dual-Read Requirement

- **Probability:** low
- **Impact:** low — new batches are written to `state/batches/`, but existing
  batches from before migration still live in `state/plans/`. If
  `ListBatches` only reads `state/batches/`, pre-existing batches are
  invisible.
- **Mitigation:** Per spec, new batches are written to `batches/` and existing
  file migration is P38-F8. During the transition, `ListBatches` should
  read both directories OR a note should warn that existing plans must be
  migrated via F8 before they appear in batch listings. The spec
  explicitly excludes migration in this feature.
- **Affected tasks:** Task 2

## Verification Approach

| Acceptance Criterion | Verification Method | Producing Task |
|---------------------|--------------------| ---------------|
| AC-001: `Batch` struct exists, no non-deprecated `model.Plan` references | Grep + compilation | Task 5 |
| AC-002: `Batch` serialisation round-trip | Unit test | Task 1 |
| AC-003: Batch lifecycle transitions function | Unit test (existing plans_test.go updated) | Task 5 |
| AC-004: Batch ID format `B{n}-slug` | Unit test | Task 2 |
| AC-005: Independent plan/batch numbering | Unit test (display_id_test.go) | Task 5 |
| AC-006: Feature display ID uses `B` prefix | Unit test | Task 2 |
| AC-007: `entity(type: "plan")` creates a batch | Integration test | Task 3 |
| AC-008: Legacy `P{n}` resolution for get/transition | Unit test | Task 2 |
| AC-009: Stored ID remains canonical after legacy resolution | Unit test | Task 2 |
| AC-010: INFO log emitted on legacy resolution | Unit test (log capture) | Task 2 |
| AC-011: Feature parent `P{n}` → `B{n}` canonical storage | Unit test | Task 2 |
| AC-012: Service methods use "Batch" terminology | Code inspection | Task 2 |
| AC-013: `entity(type: "batch")` get and list | Integration test | Task 3 |
| AC-014: Status dashboard renders "Batch" labels | Unit test (status tool) | Task 3 |
| AC-015: `kbzschema.Batch` exported, `Plan` deprecated alias | Code inspection | Task 4 |
| AC-016: `go build ./...` and `go test ./...` pass | Compilation + test run | Task 5 |
