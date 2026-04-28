# Lifecycle Integrity and Proactive Status — Design

> Design for FEAT-01KN83QN0VAFG (lifecycle-integrity-proactive-status)
> Plan: P19-workflow-lifecycle-integrity

---

## Overview

This design addresses two classes of entity lifecycle inconsistency in Kanbanzai —
premature closure (a parent marked terminal while children are still active) and
stranded completion (all children terminal but parent never advanced) — and a
structural gap in the `status` tool that makes health anomalies invisible unless
a separate `health` call is made. Three coordinated pillars are proposed: lifecycle
gates that block invalid closures, auto-advance that moves parents forward when
children complete, and structured attention items that surface health findings
directly in `status` responses.

---

## Goals and Non-Goals

### Goals

- Prevent features and plans from being marked terminal while child entities are
  still in active states
- Automatically advance features from `developing`/`needs-rework` to `reviewing`
  when all tasks complete, eliminating manual last-mile advancement
- Automatically advance plans to `done` when all features finish
- Surface health check findings as actionable structured items within `status`,
  so a single `status` call gives a complete picture
- Make `attention` items machine-readable with a typed, linkable structure that
  supports external tooling (e.g. dashboards)
- Detect features stuck in `reviewing` for longer than a configurable threshold
- Warn when open critical/high severity bugs exist against active features

### Non-Goals

- Auto-advancing features past `reviewing` to `done` (review remains a mandatory
  human gate)
- Reopening closed features or plans when bugs are filed against them
- Blocking feature transitions based on open bugs (bug warnings are advisory only)
- Propagating `not-planned` state to downstream task dependencies
- Background or event-driven processing (all transitions remain synchronous)
- Changes to the `health` tool itself

---

## Problem and Motivation

Kanbanzai tracks workflow state as a hierarchy: Plans contain Features, Features
contain Tasks. The lifecycle of each entity is governed by explicit status transitions.
Two categories of inconsistency arise in practice and were observed directly during
the P5 cleanup session that motivated this design.

### 1. Premature closure

A feature or plan can be marked `done` while child entities are still in active,
non-terminal states. The transition is not blocked. The result is a parent entity
that claims completion while work is visibly outstanding. This was observed with
`FEAT-01KMRJ83GYAQ9`, which was marked `done` while three tasks remained `ready`.

The `CheckFeatureChildConsistency` health check detects this condition after the
fact, but only when `health` is called explicitly. Nothing prevents the inconsistent
state from being created.

### 2. Stranded completion

The inverse problem: all child tasks reach a terminal state but the parent feature
remains in `developing` or `needs-rework` indefinitely. The work is done; the entity
does not reflect it. This was observed with `FEAT-01KMRJ7Z2PVR9`, which stayed in
`proposed` while all five of its tasks were `done`. The same pattern applies at the
plan level when all features are finished but the plan is never closed.

`CheckFeatureChildConsistency` detects this too — but again, only on explicit request.

### 3. Headless observability gap

Kanbanzai is a headless system: nothing is visible unless explicitly requested. The
`status` tool provides a workflow dashboard; the `health` tool detects structural
anomalies. These are separate calls. A routine `status` call returns progress metrics
and pre-computed `attention` items, but `health` findings — stale reviewing, consistency
violations, open critical bugs — are reported only as bare error/warning counts in the
`health` field. The actionable detail stays hidden unless the operator remembers to call
`health` separately.

Additionally, `attention` items are currently plain strings. They cannot be filtered by
type, linked to a specific entity, or consumed by external tooling (e.g. a dashboard).

### What happens if nothing changes

- Features and plans continue to be closeable with outstanding children, requiring
  manual cleanup sessions to detect and repair the inconsistency.
- Completed work continues to sit unacknowledged in `developing`, requiring manual
  lifecycle advancement.
- The `health` tool remains a separate call that operators must remember to make;
  anomalies that warrant attention stay invisible to routine status checks.
- External tooling cannot build on `attention` data because it has no structure.

---

## Design

Three coordinated pillars address the three problem categories above.

### Pillar A: Lifecycle Gates

#### Current state

`entity(action: transition)` accepts any lifecycle transition that is valid according to
the state machine, with no check of child entity states. A feature can be transitioned
to `done` regardless of how many tasks are still `ready` or `active`.

#### Proposed change

Add a child-state precondition to terminal transitions for features and plans:

**Feature gate:** Before accepting `done`, `superseded`, or `cancelled` on a feature,
the transition handler queries all tasks with `parent_feature == featureID`. If any
task is in a non-terminal state (`queued`, `ready`, `active`, `needs-rework`), the
transition is rejected with an error that identifies the count of blocking tasks.

**Plan gate:** Before accepting `done`, `superseded`, or `cancelled` on a plan, the
transition handler queries all features with `parent == planID`. If any feature is in a
non-terminal state, the transition is rejected.

Terminal task states (pass the gate): `done`, `not-planned`, `duplicate`.
Terminal feature states (pass the plan gate): `done`, `superseded`, `cancelled`.

#### Override path

The existing `override: true` parameter on `entity(transition)` bypasses the gate when
accompanied by `override_reason`. The override and reason are recorded permanently on the
entity. This preserves the state-repair path used during the P5 cleanup session.

#### Failure mode

If the task or feature query fails (e.g. storage error), the transition is permitted
with a warning rather than blocked. A gate that fires on infrastructure errors would
prevent legitimate lifecycle advancement during outages. The failure is logged.

---

### Pillar B: Auto-Advance

#### Current state

Moving a feature from `developing` to `reviewing` is a manual operation. An agent or
human must observe that all tasks are terminal and call `entity(transition)` explicitly.
This step is frequently missed, leaving features stranded in `developing` or
`needs-rework` after implementation is complete.

#### Proposed change

**Feature auto-advance:** When a task transition causes all tasks on a feature to reach
terminal state, and the feature is in `developing` or `needs-rework`, the feature is
automatically transitioned to `reviewing`. This fires as a side effect of the task
transition — no separate call is needed.

**Guard condition:** At least one task must be in `done` state. If all tasks are
`not-planned` or `duplicate` (the feature was descoped, not completed), auto-advance
does not fire. This distinguishes completion from abandonment.

The auto-advance is recorded as a `task_unblocked`-style side effect in the transition
response so callers can observe it.

**Plan auto-advance:** When a feature transition causes all features on a plan to reach
a finished state (`done`, `superseded`, or `cancelled`), and the plan is `active`, the
plan is automatically transitioned to `done`. Same guard: at least one feature must be
`done`.

#### Trigger points

- Feature auto-advance: triggered by `entity(transition)` on a task, by `finish`, and
  by `complete_task`. Any operation that moves a task to a terminal state checks the
  parent feature.
- Plan auto-advance: triggered by `entity(transition)` on a feature. Any operation that
  moves a feature to a terminal state checks the parent plan.

#### Failure mode

If the auto-advance transition fails (e.g. the feature is in a state where `reviewing`
is not a valid transition), the original task transition still succeeds. Auto-advance
failure is surfaced as a warning in the side effects, not as an error on the primary
operation.

---

### Pillar C: Proactive Status

#### Current state

The `status` tool returns structured JSON with an `attention` field (`[]string`) and a
`health` field (`{errors: N, warnings: N}`). Attention items are pre-computed
human-readable strings. Health findings are count-only — the operator must call `health`
separately to see what the N warnings are.

#### C1: Structured attention items

Replace `[]string` with `[]AttentionItem` across all status response objects. Each item
carries:

```
type AttentionItem struct {
    Type      string  // machine-readable category (see registry below)
    Severity  string  // "error", "warning", "info"
    EntityID  string  // entity ID this item relates to (may be empty for project-level)
    DisplayID string  // human-readable ID (e.g. "FEAT-01KN8-3QN0VAFG")
    Message   string  // human-readable description (same content as current string)
}
```

Existing `attention` string content is preserved verbatim in the `Message` field.
Agents continue to read `Message`; external tools can additionally filter by `Type`,
`Severity`, and link via `EntityID`.

**Attention type registry (initial set):**

| Type | Severity | Trigger |
|------|----------|---------|
| `ready_tasks` | `info` | Tasks available to claim |
| `stalled_task` | `warning` | Active task with no recent commits |
| `stuck_task` | `warning` | Active task dispatched >24h with no git activity |
| `all_tasks_done` | `warning` | Feature in developing/needs-rework, all tasks terminal |
| `stale_reviewing` | `warning` | Feature in reviewing > threshold days |
| `open_critical_bug` | `warning` | Critical/high severity open bug on active feature |
| `plan_ready_to_close` | `info` | All features finished, plan not yet closed |
| `feature_no_tasks` | `info` | Feature has no tasks |
| `missing_spec` | `warning` | Feature missing specification document |
| `missing_dev_plan` | `warning` | Feature missing dev-plan document |

#### C2: Health findings in status

Surface health check findings as `AttentionItem` entries in the `attention` array.
The `health` field is retained as a compact summary (`{errors: N, warnings: N}`) for
quick triage, but the findings themselves appear as structured attention items so that
a single `status` call surfaces the complete picture.

Project-level `status` (no ID argument) runs the full health check and injects findings.
Scoped calls (plan, feature, task) run only the checks relevant to that scope.

#### C3: Stale reviewing detection

A feature that has been in `reviewing` status for longer than a configurable threshold
emits a `stale_reviewing` attention item. The threshold is set in `config.yaml` under
`lifecycle.stale_reviewing_days` (default: 7). Setting the threshold to 0 disables the
check entirely.

The stale reviewing check uses the entity's `updated` timestamp to measure time in
`reviewing`. If the `updated` field is absent (legacy entities), the check is skipped
for that entity — the zero-value is treated as unknown age, not infinite age.

#### C4: Bug warnings on active features

Open bugs (status not `done`, `not-planned`, `duplicate`, or `wont-fix`) with severity
`critical` or priority `critical` or `high` that reference an active feature (status not
`done`, `superseded`, `cancelled`) emit an `open_critical_bug` attention item on that
feature's status response.

Bug warnings are attention items, not gates. Whether an open bug blocks feature
advancement is a product decision that the system cannot make mechanically; severity and
priority are set by humans and may not reflect blocking intent.

---

## Alternatives Considered

### A. Enforce gate at the store layer, not the handler layer

Putting the child-state check inside the entity store would make it impossible to bypass
at any layer. Rejected because: the store is a generic persistence layer and should not
carry business logic; the override path (which is legitimate for state repair) would
become harder to implement cleanly; and the store layer has no access to cross-entity
queries without circular dependencies.

### B. Auto-advance to `done` rather than `reviewing`

Advancing directly to `done` when all tasks complete would skip the mandatory review
gate entirely. Rejected. The `reviewing` stage exists specifically because code, spec
conformance, security, and testing reviews must happen before work is considered
complete. Auto-advancing past it would make the review gate meaningless.

### C. Replace `health` field with attention items entirely

Removing the `{errors: N, warnings: N}` summary and replacing it with only the
`attention` array was considered. Rejected because the count summary serves a different
purpose — it lets an operator quickly assess overall project health without scanning the
full attention list. Both are useful; they coexist.

### D. Per-type attention item arrays instead of a single typed array

Having separate `warnings`, `errors`, `info` arrays (or per-type arrays like
`stale_features`, `missing_docs`) was considered. Rejected in favour of a single
`[]AttentionItem` with a `Type` field, because: a single array preserves ordering
(priority can be communicated through array position); it is simpler to consume; and
adding a new type does not change the response schema.

### E. Event-driven auto-advance via a background process

Using a background goroutine or file-system watcher to detect completion and trigger
auto-advance was considered. Rejected. Kanbanzai is a tool-invocation system, not a
daemon. Introducing a background process adds deployment complexity, race conditions,
and makes state changes non-deterministic from the caller's perspective. Triggering
within the task transition handler keeps the system synchronous and predictable.

---

## Decisions

### D1: Gates apply to all terminal transitions, not just `done`

The child-state gate fires on `done`, `superseded`, and `cancelled`. A feature should
not be superseded or cancelled while tasks are still active — those tasks are orphaned
with no parent doing work. Consistent application of the gate to all three terminal
transitions prevents this class of orphan.

### D2: Guard requires at least one `done` child for auto-advance

Auto-advance fires only when at least one child is `done`. If every child is
`not-planned` or `duplicate`, the feature was descoped, not completed. Advancing it to
`reviewing` in that case would be semantically wrong — there is nothing to review.
The correct action for a fully-descoped feature is manual cancellation.

### D3: Override path is preserved and logged

`override: true` with `override_reason` bypasses the gate. The bypass is recorded on
the entity. This is the legitimate path for state repair (as demonstrated during P5
cleanup), emergency closures, and situations where business context justifies overriding
the structural check. Removing the override path would make the system inflexible in
exactly the edge cases where flexibility is most needed.

### D4: `AttentionItem` uses a string `Type` field, not an enum

A string type registry (rather than a Go enum or iota) allows new attention item types
to be added without a schema migration. External tools can handle unknown types
gracefully by falling back to displaying `Message`. The initial registry is documented
in this design; it is expected to grow.

### D5: Stale reviewing threshold is configurable, default 7 days

Seven days reflects a typical review turnaround for a solo or small-team workflow.
Configuring it to 0 disables the check for projects where reviewing can legitimately
take longer (e.g. pending external stakeholder sign-off). The threshold lives in
`config.yaml` under `lifecycle.stale_reviewing_days`.

### D6: Bug warnings are attention items, not gates

Bugs are independent entities that can be filed at any time, including against closed
features. Making open critical bugs a hard gate on feature advancement would require the
system to determine whether a bug is actually blocking — a product judgement that
involves severity, workarounds, and release strategy. The system surfaces the information
as an attention item; the human decides whether to act.

---

## Dependencies

- **Entity service** (`internal/service/entities.go`) — the lifecycle gate and
  auto-advance logic is added to the feature and plan transition handlers in this
  service. No new external dependencies.
- **`finish` and `complete_task` tools** — must call the auto-advance check after
  writing task terminal state, consistent with how they currently fire side effects
  such as task unblocking.
- **`status` tool** (`internal/mcp/status_tool.go`) — the `attention` field type
  change and health finding injection are contained within this file and its
  synthesis functions.
- **`config.yaml` schema** — a new optional key `lifecycle.stale_reviewing_days`
  (integer, default 7) is added. The config package must be extended to read this
  field and supply the default.
- **Existing health check infrastructure** (`internal/health/`) — reused as-is for
  injecting findings into `status`. No changes to the health package are required.
- **`CheckFeatureChildConsistency`** (`internal/health/entity_consistency.go`) —
  the existing detection logic is complementary and remains in place. The new gate
  operates at write time (prevention); the health check operates at read time
  (detection). Both are needed.