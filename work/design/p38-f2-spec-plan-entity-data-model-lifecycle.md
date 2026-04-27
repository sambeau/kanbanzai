# P38-F2: Plan Entity Data Model and Lifecycle — Specification

| Field   | Value |
|---------|-------|
| Date    | 2026-04-27 |
| Status  | draft |
| Feature | FEAT-01KQ7YQKBDNAP |
| Design  | P38 Meta-Planning: Plans and Batches — §2, D1, D2 |

---

## Related Work

- **Design:** `work/design/meta-planning-plans-and-batches.md` — §2 (plan entity), §4 (relationship model), D1, D2
- **Current model:** `internal/model/entities.go` — `Plan` struct, `PlanStatus` constants
- **Current service:** `internal/service/plans.go` — CRUD and lifecycle operations
- **Current lifecycle:** `internal/validate/lifecycle.go` — `allowedTransitions`, `entryStates`, `terminalStates`
- **P38-F3:** Renames the current plan entity to "batch" — this feature creates the new plan entity that takes the `plan` kind

---

## Overview

This feature introduces the new **plan** entity type: a recursive, human-managed unit of strategic planning. Plans represent *what needs to be built* — a direction, a system decomposition, or a themed group of work. They sit above batches in the entity hierarchy and can nest arbitrarily to model multi-level planning trees.

The new plan entity **reuses the current plan ID format** (`P{prefix}{n}-{slug}`) and storage directory (`.kbz/state/plans/`). It adds three new fields (`parent`, `depends_on`, `order`) and replaces the existing plan lifecycle with a planning-oriented state machine (`idea → shaping → ready → active → done`).

This feature covers:
1. The YAML state file schema (all fields, types, constraints)
2. The planning lifecycle state machine (5 active states + 2 terminal)
3. All valid lifecycle transitions including backward transitions
4. Nesting rules: parent validation, cycle detection
5. CRUD operations exposed via the `entity` tool with `type: plan`

---

## Functional Requirements

### FR-001 — Plan state file schema

Each plan entity is persisted as a YAML file at `.kbz/state/plans/{id}.yaml`. The canonical field set is:

```yaml
id: P1-social-platform           # string, required, immutable after creation
slug: social-platform            # string, required, URL-safe lowercase, immutable after creation
name: "Social Media Platform"    # string, required, human-readable display name
status: idea                     # string, required, one of the lifecycle states defined in FR-003
summary: "..."                   # string, required, brief description
parent: ""                       # string, optional, parent plan ID (empty = top-level plan)
design: "DOC-xxx"                # string, optional, document record ID for the plan's design doc
depends_on: []                   # []string, optional, plan IDs this plan depends on (deferred — not enforced)
order: 0                         # integer, optional, sibling ordering within parent (lower = higher priority)
tags: []                         # []string, optional, freeform lowercase tags
created: "2026-04-27T18:00:00Z"  # RFC3339, required, set at creation, immutable
created_by: "..."                # string, required, set at creation, immutable
updated: "2026-04-27T18:00:00Z"  # RFC3339, required, updated on every write
supersedes: ""                   # string, optional, ID of plan this supersedes
superseded_by: ""                # string, optional, ID of plan that supersedes this one
```

### FR-002 — Field types and constraints

| Field | Type | Required | Constraints |
|-------|------|----------|-------------|
| `id` | string | yes | Must match `P{prefix}{n}-{slug}` format. Immutable after creation. |
| `slug` | string | yes | Lowercase, URL-safe (`[a-z0-9-]+`). Derived from input at creation. Immutable. |
| `name` | string | yes | Non-empty after trimming whitespace. |
| `status` | string | yes | One of the recognised lifecycle states (FR-003). |
| `summary` | string | yes | Non-empty after trimming whitespace. |
| `parent` | string | no | If set, must be a valid plan ID in the `P{prefix}{n}-{slug}` format. Must reference an existing plan. Must not equal the plan's own ID (FR-009). Must not create a cycle (FR-010). |
| `design` | string | no | If set, should reference a valid document record ID. No referential integrity enforced. |
| `depends_on` | []string | no | List of plan IDs. Present in the data model but **not enforced** in this feature (deferred). Stored as-is. |
| `order` | integer | no | Defaults to `0`. Used for sibling display ordering (lower = higher priority). No uniqueness constraint. |
| `tags` | []string | no | Lowercase, deduplicated. Empty list omitted from file. |
| `created` | RFC3339 | yes | Set at creation. Immutable. |
| `created_by` | string | yes | Set at creation. Immutable. |
| `updated` | RFC3339 | yes | Updated on every write operation. |
| `supersedes` | string | no | ID of the plan this plan supersedes. |
| `superseded_by` | string | no | ID of the plan that supersedes this plan. Set during the `superseded` transition. |

### FR-003 — Planning lifecycle state machine

The plan entity uses a planning-oriented lifecycle distinct from the batch lifecycle. There are five active states and two terminal states:

| Status | Meaning |
|--------|---------|
| `idea` | Vague direction or aspiration. May have only a summary. Not yet decomposed. |
| `shaping` | Actively being refined. Design work in progress. May be partially decomposed. |
| `ready` | Fully shaped. Design approved. Decomposed into batches or child plans. |
| `active` | Work is in progress on one or more child batches or child plans. |
| `done` | All child batches and child plans are complete. |
| `superseded` | Terminal. Replaced by another plan. |
| `cancelled` | Terminal. Abandoned; will not be executed. |

### FR-004 — Entry state

Every newly created plan must have its initial status set to `idea`. Any create request that attempts to set a different initial status must be rejected with an error.

### FR-005 — Allowed lifecycle transitions

The following transitions are valid. All others must be rejected by the service layer.

| From | To | Notes |
|------|----|-------|
| `idea` | `shaping` | Forward: begin shaping |
| `idea` | `superseded` | Terminal |
| `idea` | `cancelled` | Terminal |
| `shaping` | `ready` | Forward: shaping complete |
| `shaping` | `idea` | Backward: return to unshapen idea |
| `shaping` | `superseded` | Terminal |
| `shaping` | `cancelled` | Terminal |
| `ready` | `active` | Forward: work has begun |
| `ready` | `shaping` | Backward: design revised |
| `ready` | `superseded` | Terminal |
| `ready` | `cancelled` | Terminal |
| `active` | `done` | Forward: all child work complete |
| `active` | `shaping` | Backward: scope changed |
| `active` | `superseded` | Terminal |
| `active` | `cancelled` | Terminal |
| `done` | `superseded` | Terminal |
| `done` | `cancelled` | Terminal |

Self-transitions (from = to) are always invalid.

### FR-006 — Terminal state enforcement

Once a plan reaches `superseded` or `cancelled`, no further transitions are permitted. Any attempt to transition from a terminal state must be rejected.

### FR-007 — Plan create operation

The `entity` tool must accept `action: create, type: plan` with the following input fields:

| Field | Required | Notes |
|-------|----------|-------|
| `prefix` | yes | Single character. Must be declared in the plan prefix registry. |
| `slug` | yes | URL-safe lowercase slug. |
| `name` | yes | Human-readable display name. |
| `summary` | yes | Brief description. |
| `created_by` | yes | Identity of the creator. |
| `parent` | no | Parent plan ID. Validated per FR-009 and FR-010 if provided. |
| `tags` | no | Freeform tags. |
| `order` | no | Defaults to 0. |

The created plan is assigned the next available sequence number for the given prefix, and its status is set to `idea`.

### FR-008 — Plan get, list, update, and transition operations

- **Get:** `entity(action: get, id: P{n}-{slug})` returns the plan's full state.
- **List:** `entity(action: list, type: plan)` returns all plan state files. Supports filtering by `status`, `parent`, and `tags`.
  - The `parent` filter, when set to a plan ID, returns only direct child plans whose `parent` field matches that ID.
  - The `parent` filter set to an empty string or `""` returns only top-level plans (no parent).
- **Update:** `entity(action: update, id: ...)` allows updating `name`, `summary`, `design`, `tags`, and `order`. The `parent` field may be updated subject to FR-009 and FR-010.
- **Transition:** `entity(action: transition, id: ..., status: ...)` changes the lifecycle status per FR-005.

### FR-009 — Parent field validation

When a plan's `parent` field is set (at creation or update):

1. The value must match the plan ID format (`P{prefix}{n}-{slug}`).
2. The referenced parent plan must exist in storage.
3. The value must not equal the plan's own ID (no self-reference).

### FR-010 — Cycle detection

When a plan's `parent` field is set (at creation or update), the system must verify that the target parent is not a descendant of the plan being modified. Specifically:

1. Walk the parent chain of the proposed parent plan upward.
2. If the plan's own ID appears anywhere in that ancestor chain, the operation must be rejected with an error indicating a cycle would be created.
3. This check must work at any depth (not just one level).

A plan must never be its own ancestor at any depth.

### FR-011 — `order` field defaults

If `order` is not provided at creation, it defaults to `0`. Zero is a valid order value, not a sentinel. Multiple sibling plans may share the same `order` value; no uniqueness is enforced. Order is used only for display sorting.

### FR-012 — `depends_on` field: present but not enforced

The `depends_on` field is part of the data model and must be stored and returned correctly. However, in this feature:

- It is **not validated** (referenced IDs need not exist).
- It is **not enforced** as a lifecycle gate (plan transitions are not blocked by dependency status).
- It is **not used** by any query or rollup logic.

This field is included now to ensure the data model is migration-free when dependency enforcement is implemented in a future feature.

### FR-013 — Nesting rules

- A plan with no `parent` is a top-level plan.
- A plan with a `parent` set to another plan ID is a child plan.
- A plan can contain child plans, child batches, or both.
- There is no enforced depth limit for plan nesting.
- Batches parent to plans, not to other batches.

### FR-014 — Storage location

Plan state files are stored at `.kbz/state/plans/{id}.yaml`, where `{id}` is the full plan ID (e.g. `P1-social-platform.yaml`). This reuses the current plan storage directory.

---

## Non-Functional Requirements

### NFR-001 — Lifecycle validation parity

Plan lifecycle validation must use the same `validate.ValidateTransition` mechanism used by all other entity types. The new `idea → shaping → ready → active → done` state machine must be registered in `internal/validate/lifecycle.go` alongside existing entity kinds.

### NFR-002 — Entry state enforcement

`validate.ValidateInitialState` must enforce `idea` as the entry state for `EntityKindPlan`, replacing the current `proposed` entry state.

### NFR-003 — Error messages

Transition rejection errors must name the current and target state and list valid next states, consistent with the error format used by other entity transitions.

### NFR-004 — No enforced document gates for plans

Plan lifecycle transitions do not require approved documents as prerequisites. Plans are human-managed; over-gating them defeats the purpose of supporting gradual refinement. Document expectations (FR-003 table) are guidance only.

### NFR-005 — Backward compatibility

The plan ID format (`P{prefix}{n}-{slug}`) and storage path (`.kbz/state/plans/`) are unchanged. Existing plan state files written before this feature must remain readable.

---

## Scope

**In scope:**
- New `PlanStatus` constants: `idea`, `shaping`, `ready`, `active`, `done`, `superseded`, `cancelled`
- Updated `model.Plan` struct with `Parent`, `DependsOn`, and `Order` fields
- New lifecycle state machine registered in `internal/validate/lifecycle.go`
- Entry state changed from `proposed` to `idea`
- `parent` field validation (format check, existence check, self-reference check, cycle detection)
- `depends_on` field stored and returned; not enforced
- `order` field stored and returned; defaults to 0
- `CreatePlan`, `GetPlan`, `ListPlans`, `UpdatePlan`, `UpdatePlanStatus` service functions updated
- `ListPlans` supports `parent` filter
- `entity` tool `type: plan` routes correctly for all CRUD and transition actions

**Out of scope:**
- Renaming the current plan entity to "batch" (P38-F3)
- `depends_on` enforcement or dependency gates (deferred)
- Automatic plan status advancement based on child batch completion (separate feature)
- Progress rollup across recursive plan trees (separate feature)
- Status dashboard rendering of plan trees (separate feature)
- Document gate inheritance (batch inheriting from parent plan) (separate feature)
- `plan_prefixes` / `batch_prefixes` registry separation (P38-F3)

---

## Acceptance Criteria

### AC-001 — Create plan with entry state

Given a valid create request with `type: plan`, when no `status` is provided, the created plan has `status: idea`. Any attempt to create a plan with a different initial status is rejected.

### AC-002 — Forward transitions

A plan in `idea` can transition to `shaping`. A plan in `shaping` can transition to `ready`. A plan in `ready` can transition to `active`. A plan in `active` can transition to `done`.

### AC-003 — Backward transitions

A plan in `shaping` can transition back to `idea`. A plan in `ready` can transition back to `shaping`. A plan in `active` can transition back to `shaping`.

### AC-004 — Terminal transitions

A plan in any non-terminal state can transition to `superseded` or `cancelled`. A plan in `superseded` or `cancelled` cannot transition to any state.

### AC-005 — Invalid transitions rejected

A plan in `idea` cannot transition to `ready`, `active`, or `done`. A plan in `done` cannot transition to any active state. Invalid transitions return an error naming the current state, target state, and valid alternatives.

### AC-006 — Parent field: valid reference accepted

A plan created with a `parent` field pointing to an existing plan ID succeeds. The state file contains the `parent` field with the given value.

### AC-007 — Parent field: non-existent reference rejected

A plan created with a `parent` field pointing to a plan ID that does not exist is rejected with an error.

### AC-008 — Parent field: self-reference rejected

A plan whose `parent` field is set to its own ID is rejected with an error.

### AC-009 — Cycle detection: direct cycle rejected

Given plan A with parent B, an attempt to set B's parent to A is rejected with an error indicating a cycle would be created.

### AC-010 — Cycle detection: indirect cycle rejected

Given A → B → C (A is parent of B, B is parent of C), an attempt to set A's parent to C is rejected.

### AC-011 — `order` field defaults to 0

A plan created without an `order` field has `order: 0` in its state file. A plan created with `order: 5` has `order: 5`.

### AC-012 — `depends_on` stored and returned

A plan created with `depends_on: [P2-other-plan]` stores that value and returns it in get/list responses. No validation of the referenced IDs is performed.

### AC-013 — List with parent filter

`entity(action: list, type: plan, parent: P1-social-platform)` returns only plans whose `parent` field equals `P1-social-platform`. Plans with a different parent or no parent are excluded.

### AC-014 — List top-level plans

`entity(action: list, type: plan, parent: "")` returns only plans with no `parent` field set (top-level plans).

### AC-015 — State file schema conformance

A plan state file written by this feature contains exactly the fields defined in FR-001. No legacy fields from the old plan lifecycle (`proposed`, `designing`, `reviewing`) appear in new files.

---

## Dependencies and Assumptions

### Dependencies

- **P38-F3 (Batch entity)** — F3 renames the current plan entity to batch, freeing the `plan` kind for this new entity. F2 and F3 must be implemented and merged together or in the correct order to avoid a period where `EntityKindPlan` is ambiguous.
- **`internal/validate/lifecycle.go`** — the new plan state machine is registered here alongside existing entity lifecycles.
- **`internal/model/entities.go`** — `Plan` struct gains `Parent`, `DependsOn`, and `Order` fields; `PlanStatus` constants are replaced.
- **`internal/service/plans.go`** — `CreatePlan`, `UpdatePlanStatus`, `ListPlans`, `UpdatePlan` are updated; cycle detection logic is added.
- **Prefix registry** — plan prefixes continue to use the existing `prefixes` registry until P38-F3 separates `plan_prefixes` from `batch_prefixes`.

### Assumptions

- The plan ID format (`P{prefix}{n}-{slug}`) and storage path (`.kbz/state/plans/`) are stable and do not change in this feature.
- Existing plan state files (from before P38) will be treated as "batch" entities after P38-F3 runs migration. This feature does not need to handle migration of old files.
- Document gate enforcement is not required for plan transitions. Plan lifecycle is human-managed and ungated by design (D2, Open Question 2 in design doc).
- Cycle detection traverses the plan parent chain at operation time by reading state files. No pre-built ancestry index is required for MVP; practical plan trees will be shallow (2–3 levels).
- The `depends_on` field is included in the data model now solely to avoid a future migration. No behaviour change is expected from storing it.