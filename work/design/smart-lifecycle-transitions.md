# Smart Lifecycle Transitions

- Status: design note
- Date: 2026-03-27T23:15:40Z
- Plan: P6-workflow-quality-and-review
- Feature: A (Lifecycle & Entity Foundations)
- Motivation: `work/reports/kanbanzai-2.0-workflow-retrospective.md` — R1 (4/6 agents), R9 (2/6 agents)
- Related:
  - `work/design/code-review-workflow.md` §4 (adds `reviewing` and `needs-rework` states)
  - `internal/validate/lifecycle.go` (current state machine)
  - `internal/model/entities.go` (Feature struct with document reference fields)

---

## 1. Problem

The Phase 2 feature lifecycle requires sequential transitions through document-driven stages:

    proposed → designing → specifying → dev-planning → developing → done

Each transition requires a separate `update_status` call. When a feature's upstream documents already exist and are approved — because the feature shares a plan-level spec, or because all documents were written before the feature entity was created — agents must make 5 ceremony calls to reach `developing`. Four of six agents in the 2.0 retrospective identified this as the top friction point.

The code review workflow design (§4) will extend this to 7 states by adding `reviewing` and `needs-rework`. Without fixing the ceremony problem first, the review workflow makes it worse.

Additionally, when a transition is rejected, the error message says only `"invalid feature transition "proposed" → "done""` without indicating what the valid next states are. Agents resort to trial-and-error.

---

## 2. Design Principles

1. **The state machine stays sequential.** The allowed single-step transitions do not change. Skip-gates are a higher-level convenience, not a state machine redesign.

2. **Every intermediate state is still visited.** An advance from `proposed` to `developing` records transitions through each intermediate state. The audit trail is preserved. No states are "invisible."

3. **Skipping is opt-in at the call site.** A normal `update_status` call continues to enforce single-step transitions. The advance behaviour requires an explicit parameter.

4. **Prerequisites are document-based.** Each stage gate maps to a document type. A stage can be skipped when the corresponding document exists and is approved.

5. **The review gate is never skippable.** When the review states land in Phase 2, the `developing → reviewing` transition must always be explicit. "Done" always means "reviewed."

---

## 3. Stage Gate Prerequisites

Each document-driven stage has a prerequisite. The prerequisite is "satisfied" when an approved document of the corresponding type exists, owned by the feature or its parent plan.

| Stage | Entered via | Prerequisite to skip | Document type | Checked on |
|-------|-------------|---------------------|---------------|------------|
| `designing` | `proposed → designing` | Approved design document | `design` | Feature `design` field, or any approved design doc owned by feature or parent plan |
| `specifying` | `designing → specifying` | Approved specification document | `specification` | Feature `spec` field, or any approved spec doc owned by feature or parent plan |
| `dev-planning` | `specifying → dev-planning` | Approved dev-plan document | `dev-plan` | Feature `dev_plan` field, or any approved dev-plan doc owned by feature or parent plan |
| `developing` | `dev-planning → developing` | At least one child task exists | N/A | Feature has ≥1 child task entity |
| `reviewing` | `developing → reviewing` | **Never skippable** | — | — |

### 3.1 Document lookup order

For each stage gate, the system checks:

1. **Feature's own document field** — e.g., `feature.Design` references a document record ID. If that record exists and has `status: approved`, the gate is satisfied.
2. **Documents owned by the feature** — query `doc_record_list(owner=featureID, type=X, status=approved)`. If any result exists, the gate is satisfied.
3. **Documents owned by the parent plan** — query `doc_record_list(owner=feature.Parent, type=X, status=approved)`. If any result exists, the gate is satisfied.

This handles the common case where multiple features share a plan-level specification.

### 3.2 The developing gate

The `dev-planning → developing` gate is different: it checks for child tasks, not documents. A feature with zero tasks cannot advance to `developing` via skip-gates because there is nothing to implement. This gate is satisfied when the feature has at least one child task in any status.

This gate is optional for single-step transitions — an agent can still manually transition `dev-planning → developing` without tasks. The gate only applies during multi-step advance, where skipping `dev-planning` implies the planning work is done.

### 3.3 The review gate

The `developing → reviewing` transition is never skippable, regardless of what documents exist. This preserves the invariant from the code review workflow design (§4.4, §16.1): "done" means "reviewed."

An advance call targeting `done` will stop at `reviewing` and return a message indicating that review is required.

---

## 4. The `advance` Operation

### 4.1 Behaviour

A new parameter `advance: true` (or equivalent) on the status update operation enables multi-step advancement. When set:

1. The system determines the shortest path from the current status to the target status through the sequential lifecycle.
2. For each intermediate state on that path, it checks whether the stage gate prerequisite is satisfied.
3. If all intermediate prerequisites are met, it transitions through each state in order, updating the entity's `updated` timestamp at each step.
4. If a prerequisite is *not* met at some intermediate state, it stops at that state and returns a result indicating where it stopped and why.
5. The target status itself has no prerequisite check — only the states being *skipped through* are checked.

### 4.2 Interface

```
update_status(
  entity_type: "feature",
  id: "FEAT-01KMRJ81DZ3X2",
  status: "developing",
  advance: true          // new parameter
)
```

Response on success (skipped 3 intermediate states):

```
{
  "status": "developing",
  "advanced_through": ["designing", "specifying", "dev-planning"],
  "message": "Advanced from proposed to developing (skipped 3 stages with satisfied prerequisites)"
}
```

Response on partial advance (stopped at specifying because no approved spec exists):

```
{
  "status": "specifying",
  "advanced_through": ["designing"],
  "stopped_reason": "No approved specification document found for feature or parent plan",
  "message": "Advanced from proposed to specifying (1 of 3 requested stages). Stopped: no approved specification document."
}
```

### 4.3 What advance does NOT do

- It does not create documents. If a prerequisite is unmet, it stops.
- It does not create tasks. If the developing gate requires tasks, it stops at `dev-planning`.
- It does not skip the review gate. An advance targeting `done` stops at `reviewing`.
- It does not work backward. Advance only moves forward through the lifecycle.
- It does not apply to non-feature entities. Plan, task, bug, and decision lifecycles are unaffected.

---

## 5. Valid Transitions in Error Messages

Independently of the advance feature, all lifecycle transition error messages will include the valid next states from the current state.

### 5.1 Current behaviour

```
"invalid feature transition \"proposed\" → \"done\""
```

### 5.2 New behaviour

```
"invalid feature transition \"proposed\" → \"done\"; valid transitions from \"proposed\": designing, specifying, superseded, cancelled"
```

### 5.3 Implementation

`ValidateTransition` in `internal/validate/lifecycle.go` already has access to `allowedTransitions[kind][from]`. The error message change extracts the keys from that map and appends them to the error string. This is a one-line change to the error formatting.

A new exported function is also added:

```
// ValidNextStates returns the states reachable from the given state via a single transition.
func ValidNextStates(kind EntityKind, from string) []string
```

This is useful both for the error message and for any future UI that wants to show available actions.

---

## 6. Interaction with the Code Review Workflow

The code review design (§4.2) adds these transitions:

    developing → reviewing
    reviewing  → done, needs-rework
    needs-rework → developing, reviewing

The advance operation composes cleanly with these additions:

- `advance(target="done")` from any early state will advance through document-gated stages, pass through `developing`, and **stop at `reviewing`** because the review gate is never skippable.
- `advance(target="reviewing")` from any early state will advance through document-gated stages to `developing`, then transition to `reviewing` (this final step is a normal single-step transition, not a skip).
- From `needs-rework`, `advance(target="done")` stops at `reviewing` — rework must be reviewed.

The advance logic does not need to know about review-specific states. It follows the sequential lifecycle and respects the "review is never skippable" rule, which is implemented by simply not defining a prerequisite for the `reviewing` stage.

---

## 7. Scope

### In scope

- `advance` parameter on feature status updates
- Document-based prerequisite checking for stage gates
- `ValidNextStates` function and improved error messages
- Tests for all advance paths, partial advances, and gate failures

### Out of scope

- Advance for non-feature entities (plans, tasks, bugs)
- Automatic document creation
- Automatic task creation
- Changes to the allowed single-step transitions
- Structured advance audit trail beyond the entity's `updated` timestamp (can be added later if needed)

---

## 8. Design Decisions

### 8.1 Advance is opt-in, not default

Changing the default behaviour of `update_status` would break existing agents and test expectations. The `advance` parameter is additive. Agents that don't use it see no change.

### 8.2 Partial advance stops rather than failing

When a prerequisite is unmet, the advance stops at the furthest reachable state rather than rolling back and returning an error. This is more useful: the agent gets progress rather than nothing, and the response tells them exactly what's missing.

### 8.3 Parent plan documents satisfy feature gates

Features commonly share plan-level specifications. Requiring every feature to have its own spec document would be more ceremony, not less. Checking the parent plan's documents is the pragmatic choice.

### 8.4 The developing gate checks for tasks only during advance

A normal `dev-planning → developing` transition does not require tasks to exist — an agent might transition first and create tasks next. But during advance, skipping `dev-planning` implies that planning work is already done, which means tasks should already exist. This distinction prevents advance from silently skipping a stage where real work is missing.