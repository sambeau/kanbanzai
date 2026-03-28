# Plan Review Lifecycle Specification

| Document | Plan Review Lifecycle Specification       |
|----------|-------------------------------------------|
| Status   | Draft                                     |
| Created  | 2026-03-28T16:10:00Z                      |
| Plan     | P10-review-and-doc-currency               |
| Feature  | FEAT-01KMTJH8YQ17P (plan-review-lifecycle)|
| Design   | work/plan/P10-review-and-doc-currency-plan.md §6 |

---

## 1. Purpose

This specification defines the acceptance criteria for adding a `reviewing` state to the Plan lifecycle state machine. Plans must pass through review before reaching `done`, mirroring the mandatory review gate that features already have.

---

## 2. Background

The current plan lifecycle is:

```
proposed → designing → active → done
```

With terminal states `superseded` and `cancelled` reachable from any non-terminal state.

The `active → done` shortcut means plans can be closed without any review gate. The post-P9 feedback analysis identified this as a structural gap — plan reviews happen outside the workflow system because nothing in the lifecycle models them.

The feature lifecycle already has a mandatory `reviewing` gate (`developing → reviewing → done`, with no `developing → done` shortcut). This specification applies the same pattern to plans.

---

## 3. New Plan Lifecycle

```
proposed → designing → active → reviewing → done
```

Terminal states remain `superseded` and `cancelled`, reachable from any non-terminal state (including `reviewing`).

The `active → done` transition is removed. Plans must pass through `reviewing`.

The `reviewing → active` transition is added as a rework path. If a plan review fails, the plan can return to `active` for remediation before being re-reviewed.

### 3.1 Complete Transition Table

| From | To | Notes |
|------|----|-------|
| `proposed` | `designing` | |
| `proposed` | `superseded` | Terminal |
| `proposed` | `cancelled` | Terminal |
| `designing` | `active` | |
| `designing` | `superseded` | Terminal |
| `designing` | `cancelled` | Terminal |
| `active` | `reviewing` | New — review gate |
| `active` | `superseded` | Terminal |
| `active` | `cancelled` | Terminal |
| `reviewing` | `done` | New — review passes |
| `reviewing` | `active` | New — review fails, rework needed |
| `reviewing` | `superseded` | Terminal |
| `reviewing` | `cancelled` | Terminal |
| `done` | `superseded` | Existing — plan replaced by successor |
| `done` | `cancelled` | Existing |

### 3.2 Removed Transitions

| From | To | Reason |
|------|----|--------|
| `active` | `done` | Must pass through `reviewing` |

---

## 4. Implementation

### 4.1 Model Change

Add `PlanStatusReviewing` to the plan status enum in `internal/model/entities.go`:

```go
PlanStatusReviewing PlanStatus = "reviewing"
```

### 4.2 State Machine Change

Update `allowedTransitions` in `internal/validate/lifecycle.go`:

1. Update the comment to reflect the new lifecycle: `proposed → designing → active → reviewing → done`.
2. In the `active` state's transition map, replace `done` with `reviewing`.
3. Add a new `reviewing` state entry with transitions to `done`, `active`, `superseded`, and `cancelled`.

### 4.3 Known States

The `reviewing` state must be registered as a known state for plans. Verify that `IsKnownState(EntityPlan, "reviewing")` returns `true` after the change. This should happen automatically if the state appears in `allowedTransitions`.

### 4.4 Terminal States

The `reviewing` state is NOT a terminal state. No change to `terminalStates` is needed. The terminal states for plans remain `superseded` and `cancelled`.

### 4.5 Existing Plans

Plans already in `done` status are unaffected. The state machine governs future transitions, not existing state. No migration is required.

---

## 5. Acceptance Criteria

| # | Criterion |
|---|-----------|
| B.1 | `PlanStatusReviewing` constant exists in `internal/model/entities.go` with value `"reviewing"` |
| B.2 | Plan state machine allows `active → reviewing` and `reviewing → done` |
| B.3 | Plan state machine does NOT allow `active → done` |
| B.4 | Plan state machine allows `reviewing → active` (rework path) |
| B.5 | Plan state machine allows `reviewing → superseded` and `reviewing → cancelled` |
| B.6 | Plans already in `done` state are unaffected (no migration required) |
| B.7 | `status(id: "<plan>")` correctly displays plans in `reviewing` state |
| B.8 | `entity(action: "transition", id: "<plan>", status: "reviewing")` works from `active` state |
| B.9 | `ValidateTransition(EntityPlan, "active", "done")` returns an error |
| B.10 | `ValidateTransition(EntityPlan, "active", "reviewing")` succeeds |
| B.11 | `ValidateTransition(EntityPlan, "reviewing", "done")` succeeds |
| B.12 | `ValidateTransition(EntityPlan, "reviewing", "active")` succeeds |
| B.13 | Error message for `active → done` includes `reviewing` in the list of valid transitions |
| B.14 | `go test -race ./...` passes |

---

## 6. Test Requirements

### 6.1 Unit Tests

Add or update tests in `internal/validate/lifecycle_test.go`:

- `TestPlanLifecycle_ReviewingTransitions` — table-driven test covering all valid transitions from `reviewing` (`done`, `active`, `superseded`, `cancelled`).
- `TestPlanLifecycle_ActiveCannotSkipReviewing` — verify `active → done` is rejected and the error message lists `reviewing` as a valid target.
- `TestPlanLifecycle_FullLifecyclePath` — verify the happy path `proposed → designing → active → reviewing → done` succeeds.

### 6.2 Existing Tests

Existing plan lifecycle tests must continue to pass. If any test asserts `active → done` as valid, update it to go through `reviewing`.