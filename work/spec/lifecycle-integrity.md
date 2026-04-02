# Lifecycle Integrity and Proactive Status — Specification

> Specification for FEAT-01KN83QN0VAFG (lifecycle-integrity-proactive-status)
> Plan: P19-workflow-lifecycle-integrity
> Design: work/design/lifecycle-integrity.md

---

## Problem Statement

This specification implements the design described in
`work/design/lifecycle-integrity.md`.

Kanbanzai's entity lifecycle (Plan → Feature → Task) currently permits two
classes of inconsistency: a parent entity can be marked terminal while its
children are still active (premature closure), and all children can reach a
terminal state while the parent remains in an in-progress status (stranded
completion). Additionally, the `status` tool returns health anomalies only as
bare error/warning counts, requiring a separate `health` call to see actionable
detail; and `attention` items are plain strings that cannot be filtered, linked,
or consumed by external tooling.

This specification covers three coordinated pillars:

- **Pillar A** — Lifecycle gates that block premature closure of features and plans
- **Pillar B** — Auto-advance that moves features and plans forward when all children complete
- **Pillar C** — Structured attention items and proactive surface of health findings in `status`

**Explicitly out of scope:**
- Reopening closed features or plans automatically when bugs are filed
- Auto-advancing features past `reviewing` to `done`
- Task dependency propagation when a dependency is marked `not-planned`
- Background or event-driven auto-advance (all transitions remain synchronous)
- Any change to the `health` tool itself

---

## Requirements

### Functional Requirements

#### Pillar A — Lifecycle Gates

**REQ-001:** The `entity(action: transition)` handler MUST reject a transition
of a feature to `done`, `superseded`, or `cancelled` if one or more of that
feature's child tasks are in a non-terminal state. Non-terminal task states are:
`queued`, `ready`, `active`, `needs-rework`. Terminal task states are: `done`,
`not-planned`, `duplicate`.

**REQ-002:** The rejection response for REQ-001 MUST identify the number of
blocking non-terminal tasks. The feature status MUST remain unchanged after a
rejected transition.

**REQ-003:** The `entity(action: transition)` handler MUST reject a transition
of a plan to `done`, `superseded`, or `cancelled` if one or more of that plan's
child features are in a non-terminal state. Non-terminal feature states are all
states except `done`, `superseded`, and `cancelled`.

**REQ-004:** The rejection response for REQ-003 MUST identify the number of
blocking non-terminal features. The plan status MUST remain unchanged after a
rejected transition.

**REQ-005:** A feature or plan with no child entities MUST NOT be blocked by
the gate. An entity with no children passes the gate unconditionally.

**REQ-006:** The gate MUST be bypassable via `override: true` with a non-empty
`override_reason`. When bypassed, the override and reason MUST be recorded
permanently on the entity as an override record (consistent with the existing
override mechanism).

**REQ-007:** If the child-entity query fails due to a storage error, the
transition MUST be permitted (best-effort). The failure MUST be surfaced as a
warning in the transition response. The gate MUST NOT block legitimate lifecycle
advancement due to infrastructure errors.

#### Pillar B — Auto-Advance

**REQ-008:** When a task transition causes all tasks belonging to a feature to
be in terminal state (`done`, `not-planned`, or `duplicate`), AND the feature is
in `developing` or `needs-rework` status, AND at least one task is in `done`
status, the feature MUST be automatically transitioned to `reviewing` as a side
effect of the task transition.

**REQ-009:** The auto-advance defined in REQ-008 MUST NOT fire if all tasks are
`not-planned` or `duplicate` with none in `done`. A feature where all work was
descoped MUST NOT be advanced to `reviewing` automatically.

**REQ-010:** The auto-advance defined in REQ-008 MUST be triggered by all
operations that can move a task to a terminal state: `entity(action: transition)`
on a task, `finish`, and `complete_task`.

**REQ-011:** The feature auto-advance MUST be recorded as a side effect in the
response of the triggering operation, using the existing side-effect reporting
mechanism. The side effect MUST include the feature ID, the from-status
(`developing` or `needs-rework`), and the to-status (`reviewing`).

**REQ-012:** If the feature auto-advance transition fails (e.g. the feature is
in a state where `reviewing` is not a valid next transition), the original task
transition MUST still succeed. The auto-advance failure MUST be surfaced as a
warning in the side effects, not as an error on the primary operation.

**REQ-013:** When a feature transition causes all features belonging to a plan
to be in a finished state (`done`, `superseded`, or `cancelled`), AND the plan
is in `active` status, AND at least one feature is in `done` status, the plan
MUST be automatically transitioned to `done` as a side effect.

**REQ-014:** The plan auto-advance defined in REQ-013 MUST NOT fire if all
features are `superseded` or `cancelled` with none in `done`.

**REQ-015:** The plan auto-advance MUST be triggered by `entity(action:
transition)` on a feature. The side effect MUST be recorded in the response
using the same mechanism as REQ-011.

**REQ-016:** If the plan auto-advance fails, the triggering feature transition
MUST still succeed. The failure MUST be surfaced as a warning, not an error.

#### Pillar C — Proactive Status

**REQ-017:** The `attention` field in all `status` response objects
(`projectOverview`, `planDashboard`, `featureDetail`, `taskDetail`, `bugDetail`)
MUST be changed from `[]string` to `[]AttentionItem`. Each `AttentionItem` MUST
carry the following fields:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | yes | Machine-readable category from the type registry |
| `severity` | string | yes | One of: `error`, `warning`, `info` |
| `entity_id` | string | no | ID of the entity this item relates to |
| `display_id` | string | no | Human-readable ID (e.g. `FEAT-01KN8-3QN0VAFG`) |
| `message` | string | yes | Human-readable description |

**REQ-018:** All existing attention item content MUST be preserved verbatim in
the `message` field of the new `AttentionItem` structure. No existing attention
message text MUST be changed as part of this migration.

**REQ-019:** The following attention type values MUST be defined and used
consistently. No other type values are permitted by this specification (the
registry is extensible in future specifications):

| Type | Severity | Trigger condition |
|------|----------|-------------------|
| `ready_tasks` | `info` | One or more tasks in `ready` status |
| `stalled_task` | `warning` | Active task with no state update for >3 days |
| `stuck_task` | `warning` | Active task dispatched >24h with no git commits |
| `all_tasks_done` | `warning` | Feature in `developing`/`needs-rework` with all tasks terminal |
| `stale_reviewing` | `warning` | Feature in `reviewing` beyond configured threshold |
| `open_critical_bug` | `warning` | Open critical/high bug against an active feature |
| `plan_ready_to_close` | `info` | All plan features finished, plan not `done` |
| `feature_no_tasks` | `info` | Feature has no tasks |
| `missing_spec` | `warning` | Feature missing an approved specification document |
| `missing_dev_plan` | `warning` | Feature missing an approved dev-plan document |

**REQ-020:** The `health` field in `projectOverview` and `planDashboard` MUST
be retained as a compact summary (`{errors: N, warnings: N}`). Health check
findings that are actionable MUST additionally be surfaced as `AttentionItem`
entries in the `attention` array of the same response object.

**REQ-021:** A project-level `status` call (no `id` argument) MUST run the
entity health check and inject any findings as `AttentionItem` entries. Each
finding MUST include the entity ID (if available), severity mapped from the
health check severity, and a message derived from the health check issue message.

**REQ-022:** A feature in `reviewing` status whose `updated` timestamp predates
the current time by more than `lifecycle.stale_reviewing_days` (from
`config.yaml`) MUST emit a `stale_reviewing` attention item when that feature's
status is requested.

**REQ-023:** The default value of `lifecycle.stale_reviewing_days` MUST be `7`.
Setting the value to `0` MUST disable the stale reviewing check entirely. The
key MUST be optional in `config.yaml`; absence is treated as the default.

**REQ-024:** If a feature's `updated` field is absent or zero-valued (legacy
entities), the stale reviewing check MUST be skipped for that feature. The
zero-value MUST be treated as unknown age, not infinite age.

**REQ-025:** An open bug with `severity: critical` OR `priority: critical` OR
`priority: high`, whose `origin_feature` references a feature that is not in
`done`, `superseded`, or `cancelled` status, MUST emit an `open_critical_bug`
attention item on that feature's `featureDetail` status response.

**REQ-026:** Open bug status for REQ-025 is defined as: status NOT in `done`,
`not-planned`, `duplicate`, or `wont-fix`. Bugs in any other status are
considered open.

**REQ-027:** Bug warnings (REQ-025) MUST be attention items only. They MUST NOT
gate or block any feature lifecycle transition.

---

### Non-Functional Requirements

**REQ-NF-001:** The child-entity query introduced by the lifecycle gate (REQ-001,
REQ-003) MUST NOT increase the p99 latency of an `entity(transition)` call by
more than 50ms under a project with up to 500 task entities.

**REQ-NF-002:** The `attention` field schema change (REQ-017) MUST be
backward-compatible at the JSON level. Consumers reading only the `message`
field of each item MUST continue to function without modification.

**REQ-NF-003:** Auto-advance (REQ-008, REQ-013) MUST complete within the same
synchronous call as the triggering transition. No deferred or background
processing is introduced.

---

## Constraints

- The `reviewing` stage is a mandatory human gate. Auto-advance stops at
  `reviewing`. This specification does NOT permit auto-advance past `reviewing`
  to `done`.
- The override mechanism introduced in an earlier plan is preserved unchanged.
  This specification does not alter override semantics — it only adds a new
  gate that the override can bypass.
- The `health` tool itself is unchanged. This specification covers only the
  surfacing of health findings within `status` responses.
- Bug entities use `origin_feature` (not `parent_feature`). The bug warning
  check (REQ-025) MUST use `origin_feature`.
- This specification does not introduce any new entity fields. The
  `lifecycle.stale_reviewing_days` key is a new `config.yaml` entry only.
- All transitions remain synchronous. No background goroutines, file-system
  watchers, or polling loops are introduced.

---

## Acceptance Criteria

### Pillar A — Lifecycle Gates

**AC-001 (REQ-001, REQ-002):** Given a feature in `developing` status with one
task in `ready` status, when `entity(transition, status: done)` is called on
the feature, then the call MUST return an error, the error message MUST
reference the count of blocking tasks, and the feature status MUST remain
`developing`.

**AC-002 (REQ-001):** Given a feature in `developing` status with all tasks in
`done` status, when `entity(transition, status: done)` is called, then the
transition MUST succeed.

**AC-003 (REQ-001):** Given a feature in `developing` status with tasks in
`done` and `not-planned` and `duplicate` statuses (all terminal), when
`entity(transition, status: done)` is called, then the transition MUST succeed.

**AC-004 (REQ-001):** Given a feature in `developing` status with one task in
`ready` status, when `entity(transition, status: superseded)` is called, then
the call MUST return an error and the feature status MUST remain `developing`.

**AC-005 (REQ-001):** Given a feature in `developing` status with one task in
`ready` status, when `entity(transition, status: cancelled)` is called, then
the call MUST return an error and the feature status MUST remain `developing`.

**AC-006 (REQ-003, REQ-004):** Given a plan in `active` status with one feature
in `developing` status, when `entity(transition, status: done)` is called on
the plan, then the call MUST return an error, the error message MUST reference
the count of blocking features, and the plan status MUST remain `active`.

**AC-007 (REQ-005):** Given a feature with no child tasks, when
`entity(transition, status: done)` is called, then the transition MUST succeed.

**AC-008 (REQ-006):** Given a feature in `developing` status with one task in
`ready` status, when `entity(transition, status: done, override: true,
override_reason: "state repair")` is called, then the transition MUST succeed
and the override record MUST be stored on the feature entity.

**AC-009 (REQ-007):** Given a storage error occurs when querying child tasks,
when `entity(transition, status: done)` is called on the feature, then the
transition MUST succeed and the response MUST include a warning indicating that
the child-state check could not be completed.

### Pillar B — Auto-Advance

**AC-010 (REQ-008, REQ-011):** Given a feature in `developing` status with two
tasks, one already `done`, when the second task is transitioned to `done` via
`entity(transition)`, then the feature MUST be automatically transitioned to
`reviewing` and the response MUST include a side effect recording the feature
auto-advance from `developing` to `reviewing`.

**AC-011 (REQ-008):** Given a feature in `needs-rework` status with all tasks
in `done` status, when the last task is transitioned to `done`, then the feature
MUST be automatically transitioned to `reviewing`.

**AC-012 (REQ-009):** Given a feature in `developing` status with all tasks
transitioned to `not-planned` (none in `done`), when the last task is
transitioned to `not-planned`, then the feature MUST NOT be auto-advanced.

**AC-013 (REQ-010):** Given a feature in `developing` status with one remaining
non-terminal task, when `finish` is called for that task, then the feature MUST
be automatically transitioned to `reviewing` as a side effect.

**AC-014 (REQ-012):** Given a feature in `reviewing` status (not `developing`
or `needs-rework`) with all tasks terminal, when a task transition occurs, then
auto-advance MUST NOT fire (the feature is already past `developing`).

**AC-015 (REQ-013, REQ-015):** Given a plan in `active` status with two
features, one already `done`, when the second feature is transitioned to `done`,
then the plan MUST be automatically transitioned to `done` and the response MUST
include a side effect recording the plan auto-advance.

**AC-016 (REQ-014):** Given a plan in `active` status with all features in
`superseded` status (none in `done`), when the last feature is transitioned to
`superseded`, then the plan MUST NOT be auto-advanced.

**AC-017 (REQ-012, REQ-016):** Given a feature auto-advance fails (e.g.
`reviewing` is not a valid next state from the current feature status), when the
triggering task transition is called, then the task transition MUST succeed and
the response MUST include a warning side effect describing the auto-advance
failure.

### Pillar C — Proactive Status

**AC-018 (REQ-017, REQ-018):** Given a `status` call that previously returned
`attention: ["2 task(s) ready to claim"]`, after the schema migration the same
call MUST return `attention: [{type: "ready_tasks", severity: "info", message:
"2 task(s) ready to claim", ...}]`.

**AC-019 (REQ-019):** Given a feature in `developing` status with all tasks
terminal, when `status(id: featureID)` is called, then the attention array MUST
contain an item with `type: "all_tasks_done"` and `severity: "warning"`.

**AC-020 (REQ-022, REQ-023):** Given `lifecycle.stale_reviewing_days: 7` in
`config.yaml` and a feature that has been in `reviewing` status for 8 days,
when `status(id: featureID)` is called, then the attention array MUST contain
an item with `type: "stale_reviewing"` and `severity: "warning"`.

**AC-021 (REQ-022, REQ-023):** Given `lifecycle.stale_reviewing_days: 7` and a
feature that has been in `reviewing` for 6 days, when `status(id: featureID)`
is called, then the attention array MUST NOT contain a `stale_reviewing` item.

**AC-022 (REQ-023):** Given `lifecycle.stale_reviewing_days: 0`, when
`status(id: featureID)` is called for any feature in `reviewing` regardless of
age, then the attention array MUST NOT contain a `stale_reviewing` item.

**AC-023 (REQ-024):** Given a feature in `reviewing` whose `updated` field is
absent, when `status(id: featureID)` is called, then the attention array MUST
NOT contain a `stale_reviewing` item.

**AC-024 (REQ-025, REQ-026):** Given an open bug with `severity: critical` and
`origin_feature` referencing a feature in `developing` status, when
`status(id: featureID)` is called, then the attention array MUST contain an
item with `type: "open_critical_bug"` and `severity: "warning"`.

**AC-025 (REQ-025):** Given a bug with `priority: high` (but severity not
critical) and `origin_feature` referencing a feature in `developing` status,
when `status(id: featureID)` is called, then the attention array MUST contain
an item with `type: "open_critical_bug"`.

**AC-026 (REQ-025):** Given a bug with `severity: low` and `priority: low`
against a feature in `developing` status, when `status(id: featureID)` is
called, then the attention array MUST NOT contain an `open_critical_bug` item.

**AC-027 (REQ-025):** Given a bug with `severity: critical` against a feature
in `done` status, when `status(id: featureID)` is called, then the attention
array MUST NOT contain an `open_critical_bug` item (closed features are exempt).

**AC-028 (REQ-020, REQ-021):** Given the entity health check returns two
warnings for a specific feature ID, when `status` is called with no `id`
argument (project-level), then the attention array MUST contain two items
derived from those health findings, each with `entity_id` set to the relevant
feature ID.

**AC-029 (REQ-020):** Given the entity health check returns two warnings, when
`status` is called, then the `health` field MUST still show `{errors: 0,
warnings: 2}` in addition to the attention items derived from those findings.

**AC-030 (REQ-NF-002):** A JSON consumer that reads only the `message` field
from each element of the `attention` array MUST receive the same string content
as before the schema migration, with no change in behaviour.

---

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Automated test | `TestFeatureGate_BlockedByNonTerminalTask` — transition to `done` with a `ready` child task, assert error and unchanged status |
| AC-002 | Automated test | `TestFeatureGate_AllTasksDone` — transition to `done` with all tasks `done`, assert success |
| AC-003 | Automated test | `TestFeatureGate_MixedTerminalStatuses` — `done` + `not-planned` + `duplicate`, assert success |
| AC-004 | Automated test | `TestFeatureGate_BlockedOnSuperseded` — supersede with `ready` child, assert error |
| AC-005 | Automated test | `TestFeatureGate_BlockedOnCancelled` — cancel with `ready` child, assert error |
| AC-006 | Automated test | `TestPlanGate_BlockedByNonTerminalFeature` — plan `done` with `developing` child, assert error |
| AC-007 | Automated test | `TestFeatureGate_NoChildren` — transition with no tasks, assert success |
| AC-008 | Automated test | `TestFeatureGate_OverrideBypass` — override with reason, assert success and override recorded |
| AC-009 | Automated test | `TestFeatureGate_StorageErrorPermits` — mock storage error on task query, assert transition succeeds with warning |
| AC-010 | Automated test | `TestFeatureAutoAdvance_LastTaskDone` — complete last task, assert feature transitions to `reviewing` and side effect present |
| AC-011 | Automated test | `TestFeatureAutoAdvance_FromNeedsRework` — last task done on `needs-rework` feature, assert advance to `reviewing` |
| AC-012 | Automated test | `TestFeatureAutoAdvance_NoGuardIfAllNotPlanned` — all tasks `not-planned`, assert no auto-advance |
| AC-013 | Automated test | `TestFinishTriggerAutoAdvance` — `finish` on last task, assert feature side effect |
| AC-014 | Automated test | `TestFeatureAutoAdvance_DoesNotFireFromReviewing` — feature already in `reviewing`, task transition, assert no second advance |
| AC-015 | Automated test | `TestPlanAutoAdvance_LastFeatureDone` — complete last feature, assert plan advances and side effect present |
| AC-016 | Automated test | `TestPlanAutoAdvance_NoGuardIfAllSuperseded` — all features `superseded`, assert no auto-advance |
| AC-017 | Automated test | `TestFeatureAutoAdvance_FailureSurfacedAsWarning` — auto-advance to invalid state, assert task succeeds with warning |
| AC-018 | Automated test | `TestAttentionItem_SchemaPreservesMessage` — compare legacy string content with `message` field |
| AC-019 | Automated test | `TestStatusAttention_AllTasksDone` — feature with all terminal tasks, assert `all_tasks_done` attention item |
| AC-020 | Automated test | `TestStatusAttention_StaleReviewing_Over` — feature in reviewing 8 days, threshold 7, assert item present |
| AC-021 | Automated test | `TestStatusAttention_StaleReviewing_Under` — feature in reviewing 6 days, threshold 7, assert item absent |
| AC-022 | Automated test | `TestStatusAttention_StaleReviewing_Disabled` — threshold 0, assert no stale items regardless of age |
| AC-023 | Automated test | `TestStatusAttention_StaleReviewing_NoUpdated` — absent `updated` field, assert no stale item |
| AC-024 | Automated test | `TestStatusAttention_CriticalBug_SeverityCritical` — critical severity bug, assert `open_critical_bug` item |
| AC-025 | Automated test | `TestStatusAttention_CriticalBug_PriorityHigh` — high priority bug, assert item present |
| AC-026 | Automated test | `TestStatusAttention_CriticalBug_LowSeverity` — low/low bug, assert item absent |
| AC-027 | Automated test | `TestStatusAttention_CriticalBug_ClosedFeature` — bug on `done` feature, assert item absent |
| AC-028 | Automated test | `TestStatusAttention_HealthFindingsInjected` — mock health warnings, assert attention items present at project level |
| AC-029 | Automated test | `TestStatusAttention_HealthSummaryRetained` — health findings in attention, assert `health` counts field still present |
| AC-030 | Automated test | `TestAttentionItem_BackwardCompatibility` — read only `message` from `[]AttentionItem`, assert same string as legacy `[]string` |