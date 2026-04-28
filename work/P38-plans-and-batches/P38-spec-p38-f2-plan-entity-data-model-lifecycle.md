# P38-F2: Plan Entity Data Model and Lifecycle — Specification

| Field   | Value                                                                    |
|---------|--------------------------------------------------------------------------|
| Date    | 2026-04-28T01:19:36Z                                                     |
| Status  | draft                                                                    |
| Author  | spec-author                                                              |
| Feature | FEAT-01KQ7YQKBDNAP                                                       |
| Design  | `work/design/meta-planning-plans-and-batches.md` — §2, §4                |

---

## Overview

This specification defines the new recursive plan entity type and its planning-oriented
lifecycle, as described in the P38 design document
`work/design/meta-planning-plans-and-batches.md` (§2 "The plan entity (new)" and §4
"The relationship model"). The plan entity represents strategic scope decomposition —
*what needs to be built* — distinct from the batch entity (the renamed current plan)
which represents operational work grouping.

The existing plan entity struct (`internal/model/entities.go` — `Plan`) is renamed to
`Batch` in a separate feature (P38-F3). This feature introduces a new `Plan` struct with
fields for recursive nesting (`parent`, `order`) and inter-plan dependencies
(`depends_on`), and a planning-oriented lifecycle (`idea` → `shaping` → `ready` →
`active` → `done`) distinct from the batch lifecycle.

---

## Scope

**In scope:**

- New `Plan` struct in `internal/model/entities.go` with all fields enumerated below
- New `PlanStatus` type and constants for the planning lifecycle
- Plan lifecycle validation: allowed transitions, entry state, terminal states
- Plan CRUD operations in the entity service: create, get, update, list, transition
- Plan prefix resolution via `plan_prefixes` config registry (from P38-F1)
- Parent-child nesting: a plan MAY reference a parent plan via the `parent` field
- Cycle detection: a plan MUST NOT be its own ancestor (direct or indirect)
- Entity tool (`entity_tool.go`) support for plan CRUD and lifecycle transitions
- Unit tests for plan lifecycle transitions, nesting, and cycle detection

**Explicitly excluded:**

- Batch entity rename and `B` prefix (P38-F3)
- Feature display ID updates from `P{n}-F{n}` to `B{n}-F{n}` (P38-F4)
- Recursive progress rollup for plans (P38-F5)
- Status dashboard rendering of plan trees (P38-F6)
- `depends_on` enforcement as lifecycle gates (deferred — D2 §Open Questions #3)
- Document gate inheritance through the plan tree (P38-F4)
- Migration of existing state files (P38-F8)

---

## Functional Requirements

### Data Model

- **REQ-001:** The system MUST define a new `Plan` struct in `internal/model/entities.go`
  with the following YAML-serialisable fields: `id`, `slug`, `name`, `status`, `summary`,
  `parent`, `design`, `depends_on`, `order`, `tags`, `created`, `created_by`, `updated`,
  `supersedes`, `superseded_by`.

- **REQ-002:** The `parent` field MUST be a string (optional) that holds the ID of the
  parent plan, or an empty string for top-level plans. It enables recursive plan nesting.

- **REQ-003:** The `depends_on` field MUST be a `[]string` (optional) holding plan IDs that
  this plan depends on. An empty list means no dependencies. Dependency enforcement as a
  lifecycle gate is deferred; the field is present in the data model to avoid a schema
  migration later.

- **REQ-004:** The `order` field MUST be an `int` (default: 0) used for sibling ordering
  within a parent plan. Lower numbers sort first.

- **REQ-005:** The plan entity MUST NOT include a `next_feature_seq` field (that field
  belongs to the batch entity only).

- **REQ-006:** Plan entities MUST reuse the existing plan ID format: `P{prefix}{n}-{slug}`
  (e.g. `P1-social-platform`). The ID format does not change from the current plan.

- **REQ-007:** Plan entities MUST be stored in `.kbz/state/plans/{id}.yaml`, the same
  directory currently used for plan entities.

### Lifecycle

- **REQ-008:** The plan entity MUST define a new `PlanStatus` type with the following
  lifecycle states: `idea`, `shaping`, `ready`, `active`, `done`, plus terminal states
  `superseded` and `cancelled`.

- **REQ-009:** The entry state for new plans MUST be `idea`. A newly created plan starts
  in `idea` unless an explicit status is provided and it is a valid entry state.

- **REQ-010:** The following transitions MUST be valid:
  - `idea` → `shaping`, `superseded`, `cancelled`
  - `shaping` → `ready`, `idea`, `superseded`, `cancelled`
  - `ready` → `active`, `shaping`, `superseded`, `cancelled`
  - `active` → `done`, `shaping`, `superseded`, `cancelled`
  - `done` → `superseded`, `cancelled`

- **REQ-011:** The terminal states `superseded` and `cancelled` MUST be reachable from any
  non-terminal state.

- **REQ-012:** Any transition not listed in REQ-010 MUST be rejected with an error
  describing the invalid source-to-target pair.

- **REQ-013:** The `done` status MUST support the existing auto-advance mechanism: when all
  child batches and child plans reach terminal status, the plan auto-advances to `done`
  (following the same pattern as the current plan auto-advance in
  `entityTransitionAction`).

### Nesting and Cyclic Dependency

- **REQ-014:** When a plan's `parent` field is set on create or update, the system MUST
  verify that the referenced parent plan exists and is a valid plan entity.

- **REQ-015:** The system MUST detect and reject cycles in the parent chain. A plan MUST NOT
  be a direct or indirect ancestor of itself. An attempt to set `parent` to a plan that
  would create a cycle MUST return an error.

- **REQ-016:** There is NO enforced depth limit on plan nesting. The cycle check is the only
  structural constraint.

### CRUD Operations

- **REQ-017:** The entity service MUST support `CreatePlan(slug, name, parent, ...)` —
  creating a new plan with entry status `idea`, a unique ID derived from the plan prefix
  registry and sequence counter.

- **REQ-018:** The entity service MUST support `GetPlan(id)` — retrieving a plan by its
  full plan ID.

- **REQ-019:** The entity service MUST support `UpdatePlan(id, ...)` — updating mutable
  fields (name, summary, design, parent, depends_on, order, tags). Status changes use the
  dedicated transition method.

- **REQ-020:** The entity service MUST support `UpdatePlanStatus(id, newStatus)` —
  transitioning a plan between lifecycle states, with validation per REQ-010 and REQ-012.

- **REQ-021:** The entity service MUST support `ListPlans(parentID)` — listing direct
  children of a given plan, or top-level plans when `parentID` is empty.

- **REQ-022:** The MCP entity tool MUST support `type: "plan"` for create, get, list,
  update, and transition actions, with the same call patterns as the current plan entity.

### Document Expectations (Guidance, Not Gates)

- **REQ-023:** The following document expectations MUST be documented (in the entity tool
  response or status dashboard) for each plan status but MUST NOT be enforced as gates:
  - `idea`: none required
  - `shaping`: design document (draft or approved) recommended
  - `ready`: approved design document recommended; at least one child batch or child plan
    recommended
  - `active`: same as `ready`
  - `done`: same as `ready`; retrospective optional

---

## Non-Functional Requirements

- **REQ-NF-001:** The existing `Plan` struct in `internal/model/entities.go` MUST continue
  to compile and function during the transition period (it becomes the `Batch` struct in
  P38-F3). The new plan struct MUST coexist without breaking existing batch (current plan)
  functionality.

- **REQ-NF-002:** Plan state files MUST use the same YAML schema conventions as current
  plan files: `omitempty` for optional fields, RFC 3339 timestamps, and UTF-8 encoding.

- **REQ-NF-003:** The plan lifecycle transition validation MUST reject invalid transitions
  before any state mutation occurs (atomic validation).

- **REQ-NF-004:** Cycle detection MUST complete in O(depth) time where depth is the length
  of the ancestor chain. For practical depths (0–10), it completes in microseconds.

---

## Constraints

- The plan entity ID format (`P{prefix}{n}-{slug}`) is NOT changed by this feature. It
  reuses the current format.
- Plan storage directory (`.kbz/state/plans/`) is NOT changed. The directory is shared
  with any existing plan entities during the transition period.
- The `depends_on` field is a data model placeholder. No validation beyond field type
  correctness is performed. Dependency-based lifecycle gating is deferred.
- Document expectations are guidance only. No gate enforcement is implemented for plan
  transitions.
- The existing `Plan` struct fields are NOT removed or renamed in this feature — that
  rename to `Batch` is P38-F3.
- Plan lifecycle states MUST NOT share constant values with batch lifecycle states. The
  Go types must be distinct to prevent accidental interchange.

---

## Acceptance Criteria

- **AC-001 (REQ-001):** The new `Plan` struct exists in `internal/model/entities.go` with
  all fields enumerated in REQ-001. The struct is YAML-serialisable.

- **AC-002 (REQ-002, REQ-004):** A plan can be created with `parent: "P1-parent"` and
  `order: 3`. Both fields are optional. A plan created without them has `parent: ""` and
  `order: 0`.

- **AC-003 (REQ-005):** The plan struct does NOT have a `next_feature_seq` field.
  Accessing that conceptual field on a plan entity produces a zero value or compile error.

- **AC-004 (REQ-006):** Creating a plan with slug `social-platform` produces an ID
  matching the pattern `P{prefix}{n}-social-platform` where the number is the next in
  sequence for the plan prefix.

- **AC-005 (REQ-007):** After creating a plan, a YAML file exists at
  `.kbz/state/plans/{planID}.yaml` containing all serialised fields.

- **AC-006 (REQ-008, REQ-009):** A newly created plan has status `idea`.
  `PlanStatusIdea`, `PlanStatusShaping`, `PlanStatusReady`, `PlanStatusActive`,
  `PlanStatusDone`, `PlanStatusSuperseded`, and `PlanStatusCancelled` are all defined as
  distinct constants.

- **AC-007 (REQ-010):** Transitioning `idea → shaping` succeeds. Transitioning
  `shaping → ready` succeeds. Transitioning `shaping → idea` (backward reshape)
  succeeds. Transitioning `ready → active` succeeds. Transitioning `active → done`
  succeeds. Transitioning `active → shaping` (backward scope change) succeeds.

- **AC-008 (REQ-011):** Transitioning from `idea`, `shaping`, `ready`, `active`, or
  `done` to `superseded` succeeds. Same for `cancelled`.

- **AC-009 (REQ-012):** Transitioning `idea → done` fails. Transitioning
  `done → shaping` fails. Transitioning `superseded → idea` fails. Each failure returns a
  descriptive error.

- **AC-010 (REQ-014):** Creating a plan with `parent: "P1-nonexistent"` fails with a
  descriptive error that the parent plan does not exist.

- **AC-011 (REQ-015):** Given plans P1 (parent of P2) and P2 (parent of P3), updating
  P1's parent to P3 fails with a cycle detection error. Direct self-reference (`parent`
  set to own ID) also fails.

- **AC-012 (REQ-016):** Creating a plan tree of depth 5 succeeds without depth-limit
  errors. All plans are retrievable and maintain correct parent relationships.

- **AC-013 (REQ-017):** Calling `entity(action: "create", type: "plan", name: "My Plan",
  slug: "my-plan")` via the MCP entity tool creates a plan with status `idea` and returns
  the plan ID and full details.

- **AC-014 (REQ-018, REQ-020):** Calling `entity(action: "get", id: "P1-my-plan")`
  returns the plan with all fields. Calling `entity(action: "transition", id:
  "P1-my-plan", status: "shaping")` advances the plan.

- **AC-015 (REQ-021):** Calling `entity(action: "list", type: "plan")` returns all
  top-level plans. Calling with a parent filter returns only direct children of that
  parent.

- **AC-016 (REQ-NF-001):** After introducing the new plan struct, existing batch (current
  plan) CRUD operations continue to work: creating a feature under a batch, transitioning
  batch status, etc.

- **AC-017 (REQ-NF-003):** An invalid plan transition (e.g. `done → shaping`) does not
  modify the plan's state on disk. The status remains unchanged after the failed attempt.

---

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Test | Automated test: assert Plan struct exists with all fields, marshal to YAML and unmarshal back |
| AC-002 | Test | Automated test: create plan with parent + order, create plan without them, verify defaults |
| AC-003 | Test | Compile-time check: accessing `next_feature_seq` on the new Plan struct fails compilation |
| AC-004 | Test | Automated test: create plan, verify ID format matches `P{n}-{slug}` pattern with correct sequence |
| AC-005 | Test | Automated test: create plan, assert `.kbz/state/plans/{id}.yaml` exists with correct content |
| AC-006 | Test | Automated test: assert all 7 status constants are defined and distinct; new plan has status `idea` |
| AC-007 | Test | Automated test: exercise all valid forward and backward transitions, assert success |
| AC-008 | Test | Automated test: transition from each non-terminal state to `superseded` and `cancelled`, assert success |
| AC-009 | Test | Automated test: attempt each invalid transition, assert error with descriptive message |
| AC-010 | Test | Automated test: create plan with nonexistent parent, assert error |
| AC-011 | Test | Automated test: create chain P1→P2→P3, attempt to set P1.parent = P3, assert cycle error |
| AC-012 | Test | Automated test: create 5-level nesting, assert all succeed, verify parent relationships |
| AC-013 | Test | Automated test: call MCP entity create for plan type, assert response contains plan ID and status `idea` |
| AC-014 | Test | Automated test: get and transition a plan via MCP entity tool, assert correct behaviour |
| AC-015 | Test | Automated test: list top-level plans, list children of a parent, assert correct filtering |
| AC-016 | Test | Automated test: create feature under existing batch after plan changes, assert success |
| AC-017 | Test | Automated test: attempt invalid transition, assert plan state unchanged on disk |

---

## Dependencies and Assumptions

- **P38-F1 (Config Schema and Project Singleton):** Plan prefix resolution depends on the
  `plan_prefixes` registry and independent plan sequence counter defined in F1.
- **P38-F3 (Batch Entity Rename):** The existing `Plan` struct is renamed to `Batch` in F3.
  This specification introduces the new `Plan` struct without removing the current one.
  F3 coordinates the rename and ensures both structs coexist correctly.
- **Existing plan lifecycle code** (`internal/model/entities.go`,
  `internal/validate/lifecycle.go`, `internal/mcp/entity_tool.go`) is referenced for
  patterns but the new plan lifecycle is an independent implementation, not a modification
  of existing plan lifecycle code (which becomes the batch lifecycle).
- **`depends_on` is deferred:** The field is included in the data model per the design
  document but no dependency-satisfaction logic or lifecycle gating is implemented now.
- **Cycle detection:** Implementation walks the parent chain upward. For large
  parent-child trees, this is O(depth). The design doesn't specify a maximum depth;
  practical usage is expected to be 2–3 levels.
