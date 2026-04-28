# P38-F5: Recursive Progress Rollup — Specification

| Field   | Value |
|---------|-------|
| Date    | 2026-04-27 |
| Status  | draft |
| Feature | FEAT-01KQ7YQKMCM6T |
| Design  | P38 Meta-Planning: Plans and Batches — §5 (`work/design/meta-planning-plans-and-batches.md`) |

---

## Related Work

**Design:** `work/design/meta-planning-plans-and-batches.md` — §5 "Progress rollup"
defines the semantic split between batch rollup (execution-level) and plan rollup
(strategic-level, recursive).

| Document | Relevance |
|----------|-----------|
| P38 Meta-Planning Design §3 | Batch entity: current plan entity renamed. Lifecycle and feature relationship unchanged. |
| P38 Meta-Planning Design §4 | Relationship model: plans are recursive; batches reference a parent plan via `parent` field; the plan tree is a DAG. |
| P38 F2 (plan entity) | Provides the plan data model and the no-cycle enforcement rule that guarantees the plan tree is a DAG. This spec depends on that guarantee for cycle safety. |
| P38 F3 (batch entity) | Provides the batch data model. `ComputeBatchRollup` reads child features from the batch store. |
| `internal/service/estimation.go` | Current `ComputeFeatureRollup` and `ComputePlanRollup` implementations — the baseline that `ComputeBatchRollup` replicates and `ComputePlanRollup` supersedes. |
| `internal/mcp/status_tool.go` | Status tool: `synthesiseProject`, `synthesisePlan` — currently build task/feature counts directly; must be extended to surface recursive progress for plan-scoped dashboards. |
| `internal/mcp/estimate_tool.go` | Estimate query tool: calls `ComputePlanRollup` for `type=plan`; must be updated to also handle `type=batch` via `ComputeBatchRollup`. |

---

## Overview

P38 splits the single "plan" entity into two: a recursive **plan** (strategic,
hierarchical) and a **batch** (execution unit, identical to the current plan).
This change requires the progress rollup system to track and expose progress at
both levels.

This specification covers three coordinated changes:

1. **`ComputeBatchRollup`** — a new function that replicates the existing
   `ComputePlanRollup` logic, operating on a batch and its child features and
   their tasks. The current `ComputePlanRollup` becomes `ComputeBatchRollup`
   without semantic change.

2. **`ComputePlanRollup`** (replacement) — a new recursive function that
   aggregates progress across a plan's child batches (via `ComputeBatchRollup`)
   and child plans (via recursive self-invocation). The function replaces the
   current `ComputePlanRollup` with new semantics.

3. **Status tool integration** — the `status` tool is extended so that a plan ID
   shows child plans and child batches with their aggregated progress, and the
   project overview includes top-level plans with their recursive progress
   percentage.

Neither the `ComputeFeatureRollup` function nor the feature-level progress logic
is changed by this feature. The feature-level rollup remains the leaf node of the
computation tree.

All rollup functions are read-only; they produce no side effects.

---

## Functional Requirements

### ComputeBatchRollup

**FR-001** — The system MUST expose a new method
`EntityService.ComputeBatchRollup(batchID string) (BatchRollup, error)` on the
`EntityService` type in `internal/service/estimation.go`.

**FR-002** — `ComputeBatchRollup` MUST implement logic identical to the current
`ComputePlanRollup` implementation: it iterates all features whose `parent` field
equals `batchID`, computes a `FeatureRollup` for each (via the existing
`ComputeFeatureRollup`), accumulates total and progress estimates, and returns
a `BatchRollup`.

**FR-003** — The `BatchRollup` struct MUST contain the following fields, matching
the semantics of the current `PlanRollup`:

| Field | Type | Meaning |
|-------|------|---------|
| `FeatureTotal` | `*float64` | Sum of all included feature effective estimates; `nil` when no feature carries an estimate |
| `Progress` | `float64` | Sum of done-task estimates across all features |
| `Delta` | `*float64` | `FeatureTotal - batch.estimate` (nil when either is absent) |
| `FeatureCount` | `int` | Number of non-excluded features |
| `EstimatedFeatureCount` | `int` | Number of features with a computable effective estimate |

**FR-004** — A feature's effective estimate within `ComputeBatchRollup` MUST be
computed identically to the current `ComputePlanRollup` rule: use
`FeatureRollup.TaskTotal` when non-nil, otherwise fall back to the feature's own
`estimate` field. When neither is available, the feature contributes to
`FeatureCount` but not to `EstimatedFeatureCount` or `FeatureTotal`.

**FR-005** — `ComputeBatchRollup` MUST exclude tasks with status `not-planned` or
`duplicate` from all totals (this exclusion is already enforced by
`ComputeFeatureRollup`; `ComputeBatchRollup` inherits it transitively).

**FR-006** — When `batchID` has no child features, `ComputeBatchRollup` MUST
return a zero-value `BatchRollup` with `FeatureTotal = nil` and `Progress = 0`.

### ComputePlanRollup (replacement)

**FR-007** — The existing `ComputePlanRollup(planID string) (PlanRollup, error)`
method MUST be replaced with a new implementation that performs a recursive
aggregation. The method signature and return type name (`PlanRollup`) MUST remain
unchanged so that existing call sites in `estimate_tool.go` require only
targeted updates, not structural rewrites.

**FR-008** — The `PlanRollup` struct MUST be updated to contain the following
fields:

| Field | Type | Meaning |
|-------|------|---------|
| `Total` | `*float64` | Recursive sum of all child batch and child plan totals; `nil` when no child carries an estimate |
| `Progress` | `float64` | Recursive sum of progress from child batches and child plans |
| `BatchCount` | `int` | Number of direct child batches |
| `PlanCount` | `int` | Number of direct child plans |

> Note: The current `PlanRollup` fields (`FeatureTotal`, `Delta`,
> `FeatureCount`, `EstimatedFeatureCount`) are feature-level concepts that do not
> apply to the recursive plan rollup. They MUST be removed or replaced; callers
> that reference them (e.g. `estimate_tool.go`) MUST be updated to use the new
> field names.

**FR-009** — `ComputePlanRollup` MUST iterate all **direct child batches** of
`planID` (batches whose `parent` field equals `planID`), call
`ComputeBatchRollup` for each, and accumulate their `FeatureTotal` (when non-nil)
and `Progress` into running totals.

**FR-010** — `ComputePlanRollup` MUST iterate all **direct child plans** of
`planID` (plans whose `parent` field equals `planID`), call `ComputePlanRollup`
recursively for each, and accumulate their `Total` (when non-nil) and `Progress`
into the same running totals as FR-009.

**FR-011** — When no child batch or child plan contributes an estimate,
`ComputePlanRollup` MUST return `Total = nil` and `Progress = 0`. An empty plan
(no children at all) MUST return the same zero values.

**FR-012** — When at least one child contributes an estimate, `ComputePlanRollup`
MUST set `Total` to the accumulated sum and `Progress` to the accumulated
progress. Children that do not contribute an estimate (all estimates absent) are
still counted in `BatchCount` or `PlanCount` but do not affect `Total`.

**FR-013** — `ComputePlanRollup` MUST NOT modify any entity state; it is a
read-only computation.

**FR-014** — `ComputePlanRollup` MUST rely on cycle-freedom guaranteed by F2's
no-cycle enforcement rule. It MUST NOT implement its own cycle-detection
algorithm. If a cycle is somehow present (i.e. F2's invariant is violated), the
recursive traversal MUST terminate via stack overflow or depth limit rather than
looping indefinitely. A depth guard of at least 50 levels is RECOMMENDED to
produce a recoverable error rather than a stack overflow in adversarial or corrupt
states.

### Progress percentage

**FR-015** — Progress percentage MUST be computed on demand by callers as
`Progress / Total * 100` when `Total` is non-nil and `Total > 0`. Neither
`BatchRollup` nor `PlanRollup` structs store a pre-computed percentage field.
This matches the existing pattern in `FeatureRollup`.

**FR-016** — Progress percentage values surfaced through MCP tool responses MUST
be expressed as a `float64` in the range `[0, 100]`, rounded to one decimal place
using standard rounding (round-half-up), matching the precision pattern of the
existing estimate query responses. A `Total` of zero or nil MUST produce a
displayed percentage of `null` or be omitted rather than `0` or `NaN`.

### Status tool — plan dashboard

**FR-017** — When `status` is called with a **plan ID**, the response scope MUST
be `"plan"` and the dashboard MUST include:

- A `plan` header (ID, slug, name, status) — unchanged structure.
- A `child_plans` array: each entry is a plan summary including ID, slug, name,
  status, and recursive progress (`progress`, `total`, `progress_pct`).
- A `child_batches` array: each entry is a batch summary including ID, slug,
  name, status, feature count, and batch-level progress (`progress`, `total`,
  `progress_pct`).
- An `aggregate` block: `{ "progress": float64, "total": float64 | null,
  "progress_pct": float64 | null }` for the plan as a whole.

**FR-018** — `child_plans` and `child_batches` MUST include only **direct**
children of the queried plan. Grandchildren appear within the recursive progress
totals but do not appear as separate list entries.

**FR-019** — When a plan has no child plans, `child_plans` MUST be an empty array.
When a plan has no child batches, `child_batches` MUST be an empty array.

### Status tool — batch dashboard

**FR-020** — When `status` is called with a **batch ID**, the response scope MUST
be `"batch"` (changed from `"plan"`). The batch dashboard MUST expose the same
fields as the current plan dashboard (features list, doc gaps, health, attention,
generated_at) with the scope label updated.

**FR-021** — The batch dashboard MUST additionally include a `progress` block:
`{ "progress": float64, "total": float64 | null, "progress_pct": float64 | null }`
derived from `ComputeBatchRollup`. This block MAY be omitted when `total` is
`nil` (no estimates set).

**FR-022** — The existing per-feature task counts (queued, ready, active, done,
total) in the batch dashboard MUST remain unchanged.

### Status tool — project overview

**FR-023** — The project overview (`status` with no ID) MUST include a `progress`
block on each plan summary entry:
`{ "progress": float64, "total": float64 | null, "progress_pct": float64 | null }`
derived from `ComputePlanRollup` for that plan's ID. Plans with no estimates MUST
still appear; their `progress_pct` MUST be `null`.

**FR-024** — The project overview MUST continue to display task counts
(ready/active/done/total) alongside the new progress block. The two are
complementary; neither replaces the other.

### Estimate tool

**FR-025** — When `estimate(action: "query", entity_id: "<batch-id>")` is called,
the response MUST include a `rollup` block populated via `ComputeBatchRollup`,
using the `BatchRollup` fields. The response shape MUST mirror the current plan
query response, substituting `feature_total` for the prior `feature_total` field
and preserving `progress`, `feature_count`, and `estimated_feature_count`.

**FR-026** — When `estimate(action: "query", entity_id: "<plan-id>")` is called,
the response MUST include a `rollup` block populated via the new recursive
`ComputePlanRollup`, exposing `total`, `progress`, `progress_pct`, `batch_count`,
and `plan_count`.

---

## Non-Functional Requirements

**NFR-001** — All rollup functions MUST be read-only. They MUST NOT write to the
entity store, the document store, the knowledge store, or any other persistent
state.

**NFR-002** — `ComputeBatchRollup` and the new `ComputePlanRollup` MUST perform
no more disk I/O passes than necessary. `ComputeBatchRollup` calls
`ComputeFeatureRollup` per feature, which issues one `List("task")` call each.
Callers that invoke rollup for many batches in a loop (e.g. plan dashboard) SHOULD
pre-load features and tasks and pass them down, or the implementation SHOULD
accept pre-loaded slices to avoid O(n²) list calls. A design that avoids redundant
`List` calls is REQUIRED for the status tool integration.

**NFR-003** — The depth guard introduced in FR-014 MUST return a descriptive error
(e.g. `"plan rollup depth limit exceeded at plan %s"`) rather than panicking.

**NFR-004** — Unit tests MUST cover: empty batch (FR-006), batch with features but
no estimates (FR-004 fallback), batch with all features done, empty plan with no
children (FR-011), plan with only child batches (FR-009), plan with only child
plans (FR-010), plan with mixed child batches and child plans (FR-009 + FR-010),
and progress percentage rounding (FR-016).

---

## Scope

**In scope:**

- New `BatchRollup` struct and `ComputeBatchRollup` method in
  `internal/service/estimation.go`.
- Replacement `PlanRollup` struct and `ComputePlanRollup` method in the same
  file.
- Updates to `internal/mcp/estimate_tool.go` to route batch IDs to
  `ComputeBatchRollup` and plan IDs to the new `ComputePlanRollup`.
- Updates to `internal/mcp/status_tool.go`: plan dashboard (`synthesisePlan`),
  batch dashboard (renamed from `synthesisePlan`), and project overview
  (`synthesiseProject`) to surface recursive progress.
- Unit tests for all rollup functions.

**Out of scope:**

- Migrating existing plan state files to batch state files (covered by F3/F7).
- Changing the `ComputeFeatureRollup` function or `FeatureRollup` struct.
- Adding progress percentages to the `estimate(action: "set")` response.
- Changes to the estimation scale, soft limits, or calibration reference system.
- Surfacing plan progress in the `next` work queue tool or any tool other than
  `estimate` and `status`.
- Cycle detection logic beyond the depth guard (FR-014 defers this to F2).

---

## Acceptance Criteria

**AC-001 (FR-006)** — Given a batch `B1-empty` with no child features, when
`ComputeBatchRollup("B1-empty")` is called, then `FeatureTotal` is `nil` and
`Progress` is `0.0`.

**AC-002 (FR-002, FR-004)** — Given a batch with two features, feature A having
three tasks (estimates 2, 3, 5 — all active) and feature B having no tasks but an
own estimate of 8, when `ComputeBatchRollup` is called, then `FeatureTotal` is
`10 + 8 = 18` (task total for A, own estimate for B) and `Progress` is `0`
(no done tasks).

**AC-003 (FR-002, FR-005)** — Given a batch with one feature whose tasks include
one done task (estimate 5), one active task (estimate 3), and one not-planned task
(estimate 8), when `ComputeBatchRollup` is called, then `FeatureTotal` is `8`
(5 + 3; not-planned excluded), `Progress` is `5`.

**AC-004 (FR-011)** — Given a plan `P1-empty` with no child plans and no child
batches, when `ComputePlanRollup("P1-empty")` is called, then `Total` is `nil`
and `Progress` is `0.0`.

**AC-005 (FR-009)** — Given a plan `P1-with-batches` with two direct child batches:
batch A (`FeatureTotal = 10, Progress = 4`) and batch B (`FeatureTotal = 6,
Progress = 6`), when `ComputePlanRollup("P1-with-batches")` is called, then
`Total` is `16`, `Progress` is `10`, and `BatchCount` is `2`.

**AC-006 (FR-010)** — Given a plan `P1-parent` with one child plan `P2-child`,
and `P2-child` has one batch with `FeatureTotal = 20, Progress = 10`, when
`ComputePlanRollup("P1-parent")` is called, then `Total` is `20`, `Progress` is
`10`, `PlanCount` is `1`, and `BatchCount` is `0`.

**AC-007 (FR-009, FR-010)** — Given a plan `P1-mixed` with one direct child batch
(`FeatureTotal = 8, Progress = 4`) and one direct child plan whose own rollup
returns `Total = 12, Progress = 6`, when `ComputePlanRollup("P1-mixed")` is
called, then `Total` is `20`, `Progress` is `10`.

**AC-008 (FR-016)** — Given `Progress = 1, Total = 3`, the computed progress
percentage is `33.3` (rounded to one decimal place). Given `Progress = 2,
Total = 3`, the percentage is `66.7`.

**AC-009 (FR-016)** — Given `Total = nil`, the displayed progress percentage is
`null` (or field omitted); no division is performed.

**AC-010 (FR-017)** — When `status(id: "P1-my-plan")` is called and `P1-my-plan`
has one child batch and one child plan, the response includes `"scope": "plan"`,
a non-empty `child_batches` array, a non-empty `child_plans` array, and an
`aggregate` block with a numeric `progress` value.

**AC-011 (FR-020)** — When `status(id: "B24-auth-system")` is called, the
response includes `"scope": "batch"`, a `features` array (unchanged from current
plan dashboard), and a `progress` block when at least one feature carries an
estimate.

**AC-012 (FR-023)** — When `status()` is called (no ID) and the project contains
plan `P1-platform` with a recursive total of `40` and recursive progress of `20`,
the `P1-platform` entry in `plans` includes `"progress_pct": 50.0`.

**AC-013 (FR-023)** — When `status()` is called and a plan has no estimates in its
subtree, its `plans` entry includes `"progress_pct": null` (or omits the field);
it does not include `0` or an error.

**AC-014 (FR-025)** — When `estimate(action: "query", entity_id: "B24-auth-system")`
is called, the response includes a `rollup` block with `feature_total`,
`progress`, `feature_count`, and `estimated_feature_count` fields derived from
`ComputeBatchRollup`.

**AC-015 (FR-026)** — When `estimate(action: "query", entity_id: "P1-platform")`
is called, the response includes a `rollup` block with `total`, `progress`,
`progress_pct`, `batch_count`, and `plan_count` fields derived from the new
`ComputePlanRollup`.

---

## Dependencies and Assumptions

**DEP-001** — This feature depends on **F2 (plan entity)** for the `parent` field
on plan records and for the no-cycle invariant (FR-014). `ComputePlanRollup`
assumes the plan tree is a DAG. If F2 is not yet implemented, the recursive plan
rollup cannot be safely invoked on real data.

**DEP-002** — This feature depends on **F3 (batch entity)** for the batch entity
data model and the `parent` field on batch records. `ComputeBatchRollup` loads
batch child features by matching `parent == batchID`. If F3 is not yet
implemented, `ComputeBatchRollup` will always return a zero rollup (no features
with a batch parent will exist in the store).

**DEP-003** — The existing `ComputeFeatureRollup` method and `FeatureRollup`
struct are unchanged prerequisites. `ComputeBatchRollup` delegates to
`ComputeFeatureRollup` exactly as the current `ComputePlanRollup` does.

**ASM-001** — The plan tree contains no cycles. This is guaranteed by F2's
no-cycle enforcement rule. The depth guard (FR-014) is a defensive measure for
corrupt states only; it is not the primary cycle-safety mechanism.

**ASM-002** — A batch's `parent` field contains a plan ID (or is empty). Batches
never nest inside other batches. `ComputePlanRollup` therefore never calls
`ComputeBatchRollup` on an entity that is itself a plan.

**ASM-003** — Progress percentage rounding rules match the pattern already used
throughout the estimate query tool (raw floats passed to callers; formatting
applied at the response layer). No new rounding library or utility is needed.

**ASM-004** — The `status` tool's project overview currently builds task counts by
iterating features directly. The addition of a `progress` block per plan summary
(FR-023) introduces one `ComputePlanRollup` call per plan. For projects with
up to ~100 plans this is acceptable; performance optimisation (e.g. batched
pre-loading) is deferred to a follow-on feature if profiling reveals a need.