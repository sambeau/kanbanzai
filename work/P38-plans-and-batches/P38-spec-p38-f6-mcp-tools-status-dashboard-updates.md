# P38-F6: MCP Tools and Status Dashboard Updates — Specification

| Field   | Value                                                                    |
|---------|--------------------------------------------------------------------------|
| Date    | 2026-04-28T01:19:36Z                                                     |
| Status  | draft                                                                    |
| Author  | spec-author                                                              |
| Feature | FEAT-01KQ7YQKPT8HF                                                       |
| Design  | `work/design/meta-planning-plans-and-batches.md` — §2, §3, §4, §5       |

---

## Overview

This specification defines the MCP tool and status dashboard changes required to support
the new plan entity, the renamed batch entity, and recursive progress rollup. It
implements the user-facing surface of the P38 design document
`work/design/meta-planning-plans-and-batches.md`.

Every MCP tool that currently operates on plans must be updated to distinguish between
plan entities and batch entities, use correct terminology, and surface the new plan tree
and batch structure in status dashboards.

---

## Scope

**In scope:**

- Entity tool: `type: "plan"` CRUD with new plan lifecycle, `type: "batch"` CRUD
- Status tool: plan dashboard (child plans, child batches, recursive progress), batch
  dashboard (renamed current plan dashboard), project overview with top-level plans
- Decompose tool: accept batch IDs where plan IDs were previously accepted
- Next tool: accept batch IDs, surface batch terminology in context packets
- Estimate tool: dispatch to correct rollup function per entity type
- Knowledge tool: batch-scoped knowledge entries
- Status dashboard: render `B{n}-F{n}` feature display IDs, batch/plan terminology
- All error messages and user-facing strings updated for plan/batch terminology

**Explicitly excluded:**

- New MCP tool implementations beyond the scope of plan/batch support
- Retroactive update of existing entity records (P38-F8)
- Document record migration (P38-F8)
- Any changes to MCP tool schemas beyond entity type support

---

## Functional Requirements

### Entity Tool

- **REQ-001:** The entity tool MUST accept `type: "plan"` for create, get, list, update,
  and transition operations on plan entities. Plan operations use the plan lifecycle
  states defined in P38-F2.

- **REQ-002:** The entity tool MUST accept `type: "batch"` for create, get, list, update,
  and transition operations on batch entities. `type: "plan"` (lowercase "p") is
  accepted as a deprecated synonym for `type: "batch"` during the transition period
  (per P38-F3 REQ-008).

- **REQ-003:** `entity(action: "create", type: "plan", name: "...", slug: "...")` MUST
  create a plan with status `idea` and return the plan ID and full entity details.

- **REQ-004:** `entity(action: "transition", id: "P1-...", status: "shaping")` MUST
  transition a plan through its lifecycle, validating against the plan lifecycle
  transitions defined in P38-F2.

- **REQ-005:** `entity(action: "list", type: "plan")` MUST return all plans. An optional
  `parent` parameter filters to direct children of the specified plan ID.

- **REQ-006:** `entity(action: "list", type: "batch")` MUST return all batches. An
  optional `parent` parameter filters to batches under the specified plan ID.

- **REQ-007:** All entity tool responses that reference a batch entity MUST use "batch"
  terminology in display fields, messages, and commit messages.

### Status Dashboard

- **REQ-008:** `status` with no ID (project overview) MUST render:
  - Top-level plans with their statuses and recursive progress percentages
  - Standalone batches (batches with no parent plan)
  - Summary counts: total plans, total batches, total features, total tasks
  - Attention items scoped to the project level

- **REQ-009:** `status(id: "P{n}-{slug}")` (plan dashboard) MUST render:
  - The plan's name, status, summary, and recursive progress
  - Direct child plans with their statuses and recursive progress
  - Direct child batches with their statuses and progress
  - The plan's document references (design document path/status)
  - Attention items: child plans in `idea` status, blocked batches, overdue items

- **REQ-010:** `status(id: "B{n}-{slug}")` (batch dashboard) MUST render the same
  information currently shown for plans (before P38), but with "batch" terminology:
  - Batch name, status, summary, progress
  - Features belonging to the batch with their statuses
  - Tasks ready/active/done counts per feature
  - Document gaps
  - Attention items

- **REQ-011:** The status dashboard MUST use the `B{n}-F{n}` feature display ID format
  for features belonging to batches (per P38-F4 REQ-004).

- **REQ-012:** The status dashboard MUST compute plan progress using
  `ComputePlanRollup` (recursive) and batch progress using `ComputeBatchRollup`
  (per P38-F5).

- **REQ-013:** Recursive progress for plans MUST be rendered in the plan dashboard as
  a progress bar or percentage with a child-entity summary (e.g. "3 batches, 2 plans
  — 65% complete").

### Other MCP Tools

- **REQ-014:** The decompose tool MUST accept a batch ID as the `feature_id` parent.
  Decompose continues to operate on features owned by batches (same as it previously
  operated on features owned by plans).

- **REQ-015:** The next tool MUST accept batch IDs in addition to feature/task IDs.
  Context packets for tasks under batch-owned features MUST use "batch" terminology
  in the assembled context description.

- **REQ-016:** The estimate tool MUST dispatch `estimate(action: "query", entity_id:
  "B{n}-...")` to `ComputeBatchRollup` and `estimate(action: "query", entity_id:
  "P{n}-...")` to `ComputePlanRollup`.

- **REQ-017:** The knowledge tool MUST scope batch-level knowledge entries to batch IDs.
  `knowledge(action: "list", scope: "B{n}-...")` returns knowledge entries tagged with
  that batch.

- **REQ-018:** The doc tool MUST accept a batch ID as the `owner` parameter for document
  registration, listing, and gaps analysis.

### Terminology Consistency

- **REQ-019:** All MCP tool response messages, error strings, and commit messages MUST
  use "batch" when referring to the batch entity and "plan" when referring to the plan
  entity. No tool response may use "plan" to mean "batch" after this feature is
  complete (except in the deprecated `type: "plan"` synonym).

- **REQ-020:** The existing `SideEffectPlanAutoAdvanced` side effect type for
  auto-advancing plans MUST be renamed to `SideEffectBatchAutoAdvanced` for batch
  auto-advance, and a new `SideEffectPlanAutoAdvanced` for plan auto-advance (when all
  child batches and plans reach terminal state).

---

## Non-Functional Requirements

- **REQ-NF-001:** Status dashboard rendering MUST complete in time proportional to the
  number of direct children of the queried entity. Recursive progress values are taken
  from the pre-computed rollup; the dashboard does not recompute aggregate progress
  during rendering.

- **REQ-NF-002:** All tool responses MUST be backward-compatible with existing MCP
  clients. New fields are additive; existing field names and types are preserved where
  possible.

- **REQ-NF-003:** Entity tool dispatch for `type: "plan"` (deprecated synonym for batch)
  MUST NOT add measurable overhead compared to `type: "batch"`. The synonym is resolved
  at dispatch time in a single map lookup.

---

## Constraints

- The status dashboard structure (JSON shape) is extended but not redesigned. Existing
  fields remain at their current paths.
- Tool schemas (input parameters, output shapes) are versioned via the MCP protocol.
  Breaking changes to schemas are coordinated with MCP protocol version bumps.
- The `type: "plan"` synonym for batch is temporary. It is removed in a future cleanup
  feature.
- Plan dashboard rendering MUST NOT eagerly load the entire plan tree. Only direct
  children are loaded; recursive progress values come from pre-computed rollup.

---

## Acceptance Criteria

- **AC-001 (REQ-001):** `entity(action: "create", type: "plan", name: "My Plan", slug:
  "my-plan")` returns a plan ID and status `idea`.

- **AC-002 (REQ-002):** `entity(action: "create", type: "batch", name: "My Batch", slug:
  "my-batch")` creates a batch. `entity(action: "create", type: "plan", name: "Old",
  slug: "old")` also creates a batch (deprecated synonym).

- **AC-003 (REQ-003):** Plan create responds with `{"id": "P{n}-my-plan", "status":
  "idea", ...}`.

- **AC-004 (REQ-004):** `entity(action: "transition", id: "P1-...", status: "shaping")`
  succeeds. `entity(action: "transition", id: "P1-...", status: "done")` from `idea`
  fails with a descriptive error.

- **AC-005 (REQ-005, REQ-006):** `entity(action: "list", type: "plan")` returns plans.
  `entity(action: "list", type: "plan", parent: "P1")` returns only P1's child plans.
  `entity(action: "list", type: "batch", parent: "P1")` returns P1's child batches.

- **AC-006 (REQ-007):** Entity responses for batch operations use "batch" in all display
  fields. Commit messages say "create batch" not "create plan".

- **AC-007 (REQ-008):** `status` (no ID) renders top-level plans and standalone batches
  with summary counts.

- **AC-008 (REQ-009):** `status(id: "P1-platform")` renders child plans, child batches,
  recursive progress percentage, and document references.

- **AC-009 (REQ-010):** `status(id: "B24-auth")` renders features, task counts, document
  gaps, and attention items — equivalent to pre-P38 plan dashboard.

- **AC-010 (REQ-011):** Feature display IDs in status output use `B{n}-F{n}` format.

- **AC-011 (REQ-012):** Plan status dashboard shows recursive progress from
  `ComputePlanRollup`. Batch status shows progress from `ComputeBatchRollup`.

- **AC-012 (REQ-013):** Plan dashboard includes a child-entity summary string like
  "3 batches, 2 plans — 65% complete".

- **AC-013 (REQ-014):** `decompose(action: "propose", feature_id: "<FEAT-id>")` where
  the feature's parent is a batch succeeds and produces tasks.

- **AC-014 (REQ-015):** `next(id: "<TASK-id>")` for a task under a batch-owned feature
  uses "batch" terminology in the context packet.

- **AC-015 (REQ-016):** Estimate query dispatches correctly per entity type.

- **AC-016 (REQ-017):** Knowledge entries scoped to a batch ID are queryable.

- **AC-017 (REQ-018):** `doc(action: "register", owner: "B24-auth", ...)` registers a
  document under a batch.

- **AC-018 (REQ-019):** A project-wide grep for "plan" in MCP tool response strings
  returns zero false references to the batch entity.

- **AC-019 (REQ-020):** Plan auto-advance side effects use the correct side effect type.
  Batch auto-advance uses `SideEffectBatchAutoAdvanced`.

- **AC-020 (REQ-NF-002):** An existing MCP client calling `status(id: "<batch-id>")`
  receives a response with existing fields at their expected paths.

---

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Test | Automated: create plan via entity tool, verify response |
| AC-002 | Test | Automated: create batch via both type strings, verify both work |
| AC-003 | Test | Automated: verify create response shape for plan |
| AC-004 | Test | Automated: exercise valid and invalid plan transitions |
| AC-005 | Test | Automated: list plans and batches with and without parent filter |
| AC-006 | Inspection | Code review: verify batch commit messages and display fields |
| AC-007 | Test | Automated: status() output structure and content |
| AC-008 | Test | Automated: status(P1-...) output includes recursive progress |
| AC-009 | Test | Automated: status(B24-...) output matches pre-P38 plan dashboard shape |
| AC-010 | Test | Automated: grep status output for feature display ID format |
| AC-011 | Test | Automated: verify rollup function dispatch in status rendering |
| AC-012 | Test | Automated: verify child-entity summary string in plan dashboard |
| AC-013 | Test | Automated: decompose under batch-owned feature |
| AC-014 | Test | Automated: next() context packet terminology |
| AC-015 | Test | Automated: estimate query dispatch per entity type |
| AC-016 | Test | Automated: knowledge entry scope round-trip |
| AC-017 | Test | Automated: doc registration under batch owner |
| AC-018 | Inspection | Grep MCP tool files for incorrect "plan" usage |
| AC-019 | Test | Automated: verify auto-advance side effect types |
| AC-020 | Test | Automated: existing client compatibility test |

---

## Dependencies and Assumptions

- **P38-F1 (Config Schema):** Plan and batch prefix registries enable ID resolution in
  the entity tool.
- **P38-F2 (Plan Entity):** Plan lifecycle defines valid transitions enforced by the
  entity tool.
- **P38-F3 (Batch Entity Rename):** Batch entity must exist before status dashboards
  can render batch information.
- **P38-F4 (Display IDs):** Feature display ID format (`B{n}-F{n}`) is used in status
  output.
- **P38-F5 (Recursive Rollup):** `ComputePlanRollup` and `ComputeBatchRollup` provide
  progress data for status dashboards and estimate queries.
- **Existing MCP tool code** (`internal/mcp/*.go`): All tools are extended, not
  rewritten. New plan/batch handling is added alongside existing logic.
- **MCP protocol version:** No breaking schema changes are introduced. New fields are
  additive.
