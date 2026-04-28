# P38-F3: Batch Entity Rename and B-Prefix IDs — Specification

| Field   | Value                                                                    |
|---------|--------------------------------------------------------------------------|
| Date    | 2026-04-28T01:19:36Z                                                     |
| Status | approved |
| Author  | spec-author                                                              |
| Feature | FEAT-01KQ7YQKEEHFY                                                       |
| Design  | `work/design/meta-planning-plans-and-batches.md` — §3, D4               |

---

## Overview

This specification defines the rename of the existing plan entity to **batch** throughout
the Kanbanzai codebase, as described in the P38 design document
`work/design/meta-planning-plans-and-batches.md` (§3 "The batch entity" and D4 "The `B`
prefix for batch IDs"). The current `Plan` struct, `PlanStatus` type, and all plan-related
identifiers in the model, service layer, state store, and MCP tools are renamed to batch
equivalents.

Batches retain all current plan functionality — they group features for execution, hold
documents, coordinate AI agent work, and use the existing plan lifecycle unchanged. The
primary visible change is the `B` prefix replacing `P` for batch-scoped display IDs
(e.g. `B24-F3` instead of `P24-F3`).

A backward-compatibility period is defined during which legacy `P{n}` references are
resolved to batch IDs, ensuring existing integrations and stored references continue to
function.

---

## Scope

**In scope:**

- Rename `Plan` struct to `Batch` in `internal/model/entities.go`
- Rename `PlanStatus` and its constants (`PlanStatusProposed` etc.) to `BatchStatus` and
  `BatchStatusProposed` etc.
- Rename all service methods: `CreatePlan` → `CreateBatch`, `GetPlan` → `GetBatch`,
  `UpdatePlanStatus` → `UpdateBatchStatus`, etc.
- Update the entity service to recognise `type: "batch"` alongside `type: "plan"` for
  backward compatibility
- Introduce `B{n}-{slug}` as the canonical ID format for new batches
- Legacy resolution: `P{n}-{slug}` references resolve to the corresponding `B{n}-{slug}`
  batch during the transition period
- Update the batch ID prefix from `P` to `B` in the batch prefix registry (P38-F1)
- Update state store: `.kbz/state/plans/` → `.kbz/state/batches/` directory (new batches;
  migration of existing files is P38-F8)
- Update all MCP tools that reference "plan" for batch-related operations (entity tool,
  status tool, decompose, etc.)
- Update exported types in `kbzschema/` package

**Explicitly excluded:**

- Migration of existing plan state files from `P{n}` to `B{n}` names (P38-F8)
- Migration of existing plan worktree folders from `P{n}` to `B{n}` (P38-F1 work tree
  migration feature)
- Updating feature display IDs from `P{n}-F{n}` to `B{n}-F{n}` (P38-F4)
- Document reference updates in existing document records (P38-F8)
- Removing or deprecating the `prefixes` field in config (P38-F8 migration)
- Status dashboard plan-tree rendering (P38-F6)

---

## Functional Requirements

### Struct and Type Renames

- **REQ-001:** The existing `Plan` struct in `internal/model/entities.go` MUST be renamed
  to `Batch`. All YAML tags remain unchanged.

- **REQ-002:** The existing `PlanStatus` type and all its constants MUST be renamed:
  - `PlanStatus` → `BatchStatus`
  - `PlanStatusProposed` → `BatchStatusProposed`
  - `PlanStatusDesigning` → `BatchStatusDesigning`
  - `PlanStatusActive` → `BatchStatusActive`
  - `PlanStatusReviewing` → `BatchStatusReviewing`
  - `PlanStatusDone` → `BatchStatusDone`
  - `PlanStatusSuperseded` → `BatchStatusSuperseded`
  - `PlanStatusCancelled` → `BatchStatusCancelled`

- **REQ-003:** The `Batch` struct MUST retain all existing fields of the current `Plan`
  struct: `id`, `slug`, `name`, `status`, `summary`, `design`, `tags`,
  `next_feature_seq`, `created`, `created_by`, `updated`, `supersedes`, `superseded_by`,
  plus the new `parent` field (string, optional) referencing a parent plan ID.

- **REQ-004:** The batch lifecycle remains unchanged from the current plan lifecycle:
  `proposed → designing → active → reviewing → done`, plus terminal states `superseded`
  and `cancelled`, with all existing transitions, gates, overrides, and the
  `proposed → active` shortcut.

### ID Format and Prefix

- **REQ-005:** New batches MUST use the ID format `B{prefix}{n}-{slug}` (e.g.
  `B24-auth-system`). The prefix letter is determined by the `batch_prefixes` registry.

- **REQ-006:** The batch sequence counter MUST derive its next number from the
  `batch_prefixes` registry independently of the plan counter (per P38-F1 REQ-007).

- **REQ-007:** Batch-scoped feature display IDs MUST use the `B{n}-F{n}` format (e.g.
  `B24-F3`). This replaces the current `P{n}-F{n}` format.

### Backward Compatibility

- **REQ-008:** The entity tool MUST accept `type: "plan"` as a synonym for
  `type: "batch"` during the transition period. Operations on `type: "plan"` MUST
  delegate to batch operations.

- **REQ-009:** When a legacy plan ID in `P{n}-{slug}` format is provided as input (e.g.
  entity transition, status query), the system MUST resolve it to the corresponding batch
  ID `B{n}-{slug}` and proceed transparently.

- **REQ-010:** Resolution of legacy `P{n}` references to batch IDs MUST NOT modify the
  stored ID. The batch's stored ID remains `B{n}-{slug}`; the legacy format is resolved
  at lookup time only.

- **REQ-011:** The backward-compatibility period MUST log a deprecation notice (at INFO
  level) when a legacy `P{n}` reference is resolved. This provides visibility into which
  callers still use the old format.

- **REQ-012:** Feature entities that reference a batch via their `parent` field MUST
  accept both the new `B{n}-{slug}` and legacy `P{n}-{slug}` format during the transition
  period. The canonical stored value is `B{n}-{slug}`.

### Service and Tool Updates

- **REQ-013:** All service methods referencing "plan" for batch operations MUST be renamed
  to use "batch": `CreatePlan` → `CreateBatch`, `GetPlan` → `GetBatch`,
  `UpdatePlan` → `UpdateBatch`, `UpdatePlanStatus` → `UpdateBatchStatus`,
  `ListPlans` → `ListBatches`, `ResolvePlanByNumber` → `ResolveBatchByNumber`.

- **REQ-014:** The `entity_tool.go` MCP tool MUST support `type: "batch"` as the canonical
  entity type for create, get, list, update, and transition operations on batches.
  `type: "plan"` is accepted as a deprecated synonym.

- **REQ-015:** The `status_tool.go` MCP tool MUST use "batch" terminology in all
  dashboard output where it currently renders plan-level information.

- **REQ-016:** The `kbzschema` exported types package MUST alias `Plan` to `Batch` and
  `PlanStatus*` constants to `BatchStatus*` equivalents, with the original names
  deprecated.

---

## Non-Functional Requirements

- **REQ-NF-001:** The rename MUST be a compile-time refactor. After the rename, no
  references to `model.Plan`, `model.PlanStatus`, or `model.PlanStatusProposed` (and
  siblings) remain in non-deprecated code paths.

- **REQ-NF-002:** The backward-compatibility `type: "plan"` synonym MUST NOT introduce a
  new code path or branch. It MUST delegate to the same batch handler via a simple
  dispatch table entry.

- **REQ-NF-003:** Legacy `P{n}` resolution MUST add no more than one additional state
  store lookup compared to a direct `B{n}` lookup.

- **REQ-NF-004:** All batch-related user-facing messages (error strings, help text,
  dashboard labels) MUST use the term "batch" consistently during the transition period.

---

## Constraints

- The existing `Plan` struct is NOT deleted in this feature. It is renamed to `Batch`
  in-place. The new `Plan` struct (from P38-F2) is a separate type.
- The batch lifecycle is NOT changed. All current lifecycle transitions, gates, and
  shortcuts remain identical.
- The `prefixes` field in config is NOT deprecated or removed by this feature (P38-F8).
- Existing plan state files in `.kbz/state/plans/` are NOT migrated to
  `.kbz/state/batches/` by this feature (P38-F8). New batches are written to
  `.kbz/state/batches/`.
- The backward-compatibility period has no defined end date in this specification. Removal
  of `type: "plan"` synonym and legacy `P{n}` resolution is a future feature.

---

## Acceptance Criteria

**AC-001.** The `Batch` struct and `BatchStatus` type/constants
  exist in `internal/model/entities.go`. A project-wide search for `model.Plan` (the old
  struct name) returns zero non-deprecated results.

**AC-002.** The `Batch` struct has all fields currently on the `Plan` struct,
  plus `parent`. A batch can be serialised to YAML and deserialised back without data
  loss.

**AC-003.** Transitioning a batch `proposed → designing → active → reviewing
  → done` succeeds. The shortcut `proposed → active` succeeds. Transitioning
  `active → cancelled` succeeds. All gates and overrides function as before.

**AC-004.** Creating a batch with slug `auth-system` produces an ID matching
  `B{n}-auth-system` where `n` is the next number for the `B` prefix.

**AC-005.** After creating batch `B1-foo` and plan `P1-bar`, creating another
  batch produces `B2-{slug}` (does NOT collide with plan numbering).

**AC-006.** A feature created under batch `B24-auth-system` has a display ID
  of `B24-F{n}`, not `P24-F{n}`.

**AC-007.** `entity(action: "create", type: "plan", name: "Test", slug:
  "test")` creates a batch entity successfully. The response uses "batch" terminology.

**AC-008.** Given a batch stored as `B5-my-batch`, calling
  `entity(action: "get", id: "P5-my-batch")` returns the batch. Calling
  `entity(action: "transition", id: "P5-my-batch", status: "active")` transitions it.

**AC-009.** After resolving `P5-my-batch` → `B5-my-batch`, the batch record's
  stored ID remains `B5-my-batch`. The legacy format is not persisted.

**AC-010.** When `P{n}` resolution occurs, a log line at INFO level is
  emitted containing the legacy reference and the resolved batch ID.

**AC-011.** Creating a feature with `parent: "B5-my-batch"` succeeds.
  Creating a feature with `parent: "P5-my-batch"` also succeeds and stores
  `B5-my-batch` as the canonical parent.

**AC-012.** All service method names use "Batch" terminology. Calling
  `CreateBatch`, `GetBatch`, `UpdateBatchStatus`, `ListBatches` compiles and functions.

**AC-013.** `entity(action: "get", type: "batch", id: "B1-test")` returns the
  batch. `entity(action: "list", type: "batch")` returns all batches.

**AC-014.** The status dashboard for a batch renders with "Batch" heading
  labels, not "Plan".

**AC-015.** `kbzschema.Batch` is an exported type identical to the previous
  `kbzschema.Plan`. `kbzschema.Plan` is an alias for `kbzschema.Batch` with a `Deprecated`
  comment.

**AC-016.** The project compiles without errors. All tests that referenced
  `Plan` are updated to reference `Batch`.

---

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Test + Inspection | Grep for `model.Plan` in non-deprecated code; assert zero results. Assert `Batch` struct and `BatchStatus` constants exist. |
| AC-002 | Test | Automated test: marshal/unmarshal a `Batch` struct, verify all fields round-trip. |
| AC-003 | Test | Automated test: exercise batch lifecycle transitions, verify gates still function. |
| AC-004 | Test | Automated test: create batch, assert ID format `B{n}-slug`. |
| AC-005 | Test | Automated test: create interleaved plans and batches, verify independent numbering. |
| AC-006 | Test | Automated test: create feature under batch, verify display ID uses `B` prefix. |
| AC-007 | Test | Automated test: call entity create with `type: "plan"`, assert batch created. |
| AC-008 | Test | Automated test: resolve legacy `P{n}` → `B{n}` for get and transition. |
| AC-009 | Test | Automated test: after legacy resolution, assert stored ID is canonical. |
| AC-010 | Test | Automated test: trigger legacy resolution, assert INFO log line emitted. |
| AC-011 | Test | Automated test: create feature with both `B{n}` and `P{n}` parent references. |
| AC-012 | Inspection | Review service method names; verify all use "Batch". |
| AC-013 | Test | Automated test: call entity tool with `type: "batch"` for get and list. |
| AC-014 | Test | Automated test: query status for a batch, verify "Batch" labels in output. |
| AC-015 | Inspection | Review `kbzschema` exports; verify alias exists with deprecation comment. |
| AC-016 | Test | Run `go build ./...` and `go test ./...`; assert zero failures. |

---

## Dependencies and Assumptions

- **P38-F1 (Config Schema):** The `batch_prefixes` registry and independent batch sequence
  counter must be available before batch creation can use the `B` prefix.
- **P38-F2 (Plan Entity):** The new `Plan` struct must coexist in the model package so
  that `Batch` and `Plan` are distinct types. This feature renames the old `Plan` to
  `Batch` without conflicting with the new `Plan`.
- **P38-F8 (State File Migration):** Migration of existing `P{n}` state files to `B{n}`
  is deferred to F8. This feature only handles new batch creation and runtime resolution.
- **Backward compatibility window:** The duration of the `type: "plan"` synonym and
  legacy `P{n}` resolution period is undefined. A future feature will remove these.
- **Go module compatibility:** The rename is a breaking change for external consumers of
  the `kbzschema` package. Deprecated aliases are provided to ease the transition.
