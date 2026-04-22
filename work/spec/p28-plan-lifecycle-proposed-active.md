| Field  | Value       |
|--------|-------------|
| Date   | 2026-04-22  |
| Status | Draft       |
| Author | spec-author |

# Specification: Plan Lifecycle Proposed-to-Active Transition

## Problem Statement

This specification covers Issue 1 of Sprint 2 of the P28 plan (doc-intel polish and workflow
reliability): adding a direct `proposed → active` transition to the plan state machine.

Currently, a plan at `proposed` whose features are already mid-lifecycle (past `designing`)
cannot transition directly to `active`. The only legal transitions from `proposed` are
`designing`, `superseded`, and `cancelled`. When agents resume cross-session work on such a
plan, they are forced to step through `designing` and apply a manual override to reach `active`.
This produces misleading override records in the audit trail — records that appear as legitimate
gate violations rather than the routine resume operations they are.

**Parent design document:**
`P28-doc-intel-polish-workflow-reliability/design-p28-doc-intel-polish-workflow-reliability`

**In scope:**

- Adding a `proposed → active` transition to the plan state machine, guarded by a precondition
  that verifies the plan has at least one feature in a post-designing lifecycle state.
- Returning a descriptive error when the precondition is not satisfied.
- Writing a system-generated override record to the entity's audit trail when the transition
  fires successfully.

**Out of scope:**

- Changes to transitions originating from any plan state other than `proposed`.
- Changes to the `designing → active` gate or its approved-design-document prerequisite.
- Skill file updates documenting the workaround for this issue (covered by FEAT-01KPVDDYN855F,
  Sprint 0).
- Changes to feature-level lifecycle transitions.
- UI or CLI presentation of the new transition.
- The `decompose apply` skeleton dev-plan change (Sprint 2, Issue 2).
- The `worktree` timeout fix (Sprint 2, Issue 3).

---

## Requirements

### Functional Requirements

**REQ-001 — Direct `proposed → active` transition**

The plan state machine MUST permit a direct transition from `proposed` to `active` as a
first-class transition. It MUST be reachable via `entity(action: "transition", status: "active")`
on a plan in `proposed` status without requiring the caller to supply an `override` flag.

**REQ-002 — In-flight features precondition**

The `proposed → active` transition MUST succeed only when the plan has at least one feature
whose lifecycle state is in the post-designing set. The qualifying feature lifecycle states are:
`specifying`, `dev-planning`, `developing`, `reviewing`, and `done`.

**REQ-003 — Rejection when precondition is not met**

When `proposed → active` is attempted on a plan that does not satisfy the precondition in
REQ-002 — because the plan has no features, or because all features are at `proposed` or
`designing` — the system MUST reject the transition and return an error. The error message
MUST be descriptive and MUST include a directive for the agent to use the `proposed → designing`
path instead.

**REQ-004 — System-generated override record on success**

When the `proposed → active` transition fires successfully, the system MUST append a
system-generated entry to the entity's override records. The entry text MUST match the
following pattern exactly:

> `proposed → active shortcut: N feature(s) in post-designing state at transition time`

where `N` is the integer count of qualifying features (those in the post-designing set as
defined in REQ-002) at the moment of the transition.

**REQ-005 — Existing `proposed → designing` transition unchanged**

The existing `proposed → designing` transition MUST continue to be accepted for all plans in
`proposed` status, regardless of whether any features are in-flight. No precondition is added
to this transition.

**REQ-006 — `designing → active` gate unaffected**

The gate check that applies when transitioning a plan from `designing` to `active` — which
requires an approved design document to be registered against the plan — MUST NOT be weakened,
removed, or bypassed by the introduction of the `proposed → active` shortcut.

### Non-Functional Requirements

**REQ-NF-001 — System-generated override record distinguishability**

The system-generated override record entry (REQ-004) MUST be programmatically distinguishable
from human-authored override reasons. It MUST be prefixed with the fixed string
`"proposed → active shortcut:"` so that tooling and audit queries can identify it without
parsing free-form text.

**REQ-NF-002 — Transition response latency**

The `proposed → active` transition — including the precondition check and override record
write — MUST complete within 2 seconds on a local development machine running SQLite, measured
from receipt of the MCP tool call to the tool response, under normal single-user load.

**REQ-NF-003 — Precondition check uses current state**

The precondition check (REQ-002) MUST read feature lifecycle states from the entity index at
the time of the tool call. It MUST NOT rely on cached or otherwise stale state. A feature
transitioned to a qualifying state moments before the plan shortcut call MUST be counted.

---

## Constraints

- The `proposed → superseded` and `proposed → cancelled` transitions MUST NOT be altered.
- The `proposed → designing` transition MUST NOT be altered (REQ-005).
- Plans with an empty feature list MUST NOT be eligible for the `proposed → active` shortcut.
- Plans where every feature is at `proposed` or `designing` MUST NOT be eligible for the
  shortcut. The §11.1 lifecycle rule — that plans must have a design phase — is only relaxed
  when in-flight work demonstrates the design phase has implicitly occurred.
- The system-generated override record MUST be written to the same override-record store used
  by human-authored `override_reason` entries, so that unified audit access is preserved.
- No breaking changes to the existing `entity(action: "transition")` API surface are permitted.
  Callers that do not pass `override: true` and have a plan with qualifying in-flight features
  MUST receive a success response; callers on ineligible plans MUST receive a clear error — not
  a silent no-op.

---

## Acceptance Criteria

**AC-001 (REQ-001):** Given a plan entity in `proposed` status with at least one feature in
`specifying`, when `entity(action: "transition", id: "<plan-id>", status: "active")` is called
without an `override` flag, then the call succeeds and the plan status becomes `active`.

**AC-002 (REQ-002):** Given a plan entity in `proposed` status with three features at
`specifying`, `developing`, and `done` respectively, when
`entity(action: "transition", id: "<plan-id>", status: "active")` is called, then the
transition succeeds (at least one qualifying feature satisfies the precondition).

**AC-003 (REQ-002):** Given a plan entity in `proposed` status with all features at `designing`,
when `entity(action: "transition", id: "<plan-id>", status: "active")` is called, then the
transition is rejected (no feature is in the post-designing set).

**AC-004 (REQ-003):** Given a plan entity in `proposed` status with no features, when
`entity(action: "transition", id: "<plan-id>", status: "active")` is called, then the
transition is rejected and the error message includes a directive to use `proposed → designing`
instead.

**AC-005 (REQ-003):** Given a plan entity in `proposed` status with all features at `proposed`,
when `entity(action: "transition", id: "<plan-id>", status: "active")` is called, then the
transition is rejected and the error message references the `proposed → designing` path.

**AC-006 (REQ-004):** Given a plan that has just transitioned from `proposed` to `active` via
the shortcut with two qualifying features, when `entity(action: "get", id: "<plan-id>")` is
called, then the response includes an override record whose text is
`"proposed → active shortcut: 2 feature(s) in post-designing state at transition time"`.

**AC-007 (REQ-004, REQ-NF-001):** Given the override record written by a successful shortcut
transition, when it is compared to a human-authored `override_reason` string, then the
system-generated entry is programmatically identifiable by the prefix
`"proposed → active shortcut:"` — a prefix that no human-authored reason is expected to
carry.

**AC-008 (REQ-005):** Given a plan entity in `proposed` status with no features in a
post-designing state, when `entity(action: "transition", id: "<plan-id>", status: "designing")`
is called, then the transition succeeds exactly as it did before this change (no regression to
the existing designing path).

**AC-009 (REQ-005, REQ-006):** Given a plan entity in `designing` status with no approved
design document registered, when `entity(action: "transition", id: "<plan-id>", status: "active")`
is called, then the transition is rejected by the existing design-document gate. The
`proposed → active` shortcut has no bearing on this gate.

**AC-010 (REQ-006):** Given a plan entity in `designing` status with an approved design
document registered, when `entity(action: "transition", id: "<plan-id>", status: "active")` is
called, then the transition succeeds via the existing gate without modification.

**AC-011 (REQ-NF-002):** Given a plan entity in `proposed` status with qualifying in-flight
features, when the `proposed → active` shortcut is executed, then the tool response is received
within 2 seconds on a local development machine under single-user load.

**AC-012 (REQ-NF-003):** Given a plan in `proposed` status, when a feature belonging to that
plan is transitioned to `specifying` and the plan shortcut is then immediately attempted, then
the newly-qualified feature is included in the precondition count N — no stale-state cache
suppresses it.

---

## Verification Plan

| Criterion | Method     | Description |
|-----------|------------|-------------|
| AC-001    | Test       | Unit test: call `proposed → active` on a plan with one `specifying` feature and no `override` flag; assert status becomes `active` and call succeeds. |
| AC-002    | Test       | Unit test: plan with features at `specifying`, `developing`, and `done`; assert shortcut transition succeeds. |
| AC-003    | Test       | Unit test: plan with all features at `designing`; assert transition is rejected with a non-nil error. |
| AC-004    | Test       | Unit test: plan with no features; assert rejection and confirm error text contains the `proposed → designing` directive. |
| AC-005    | Test       | Unit test: plan with all features at `proposed`; assert rejection and confirm error text references `proposed → designing`. |
| AC-006    | Test       | Integration test: execute shortcut on a plan with exactly two qualifying features, then fetch entity; assert override record text matches the prescribed pattern with N = 2. |
| AC-007    | Inspection | Review override record store and the code path that writes the system-generated entry; confirm the `"proposed → active shortcut:"` prefix is hardcoded and distinct from the `override_reason` field supplied by callers. |
| AC-008    | Test       | Regression test: call `proposed → designing` on a plan with no qualifying features; assert success and unchanged behaviour. |
| AC-009    | Test       | Regression test: call `designing → active` on a plan with no approved design document; assert gate rejection is unchanged. |
| AC-010    | Test       | Regression test: call `designing → active` on a plan with an approved design document; assert success and unchanged gate behaviour. |
| AC-011    | Demo       | Time the shortcut transition end-to-end under local SQLite load (single user); assert wall-clock response is under 2 seconds. |
| AC-012    | Test       | Integration test: transition a feature to `specifying`, immediately call the plan shortcut, assert the feature appears in N and the transition succeeds. |