# Specification: Mandatory Stage Gate Enforcement (Kanbanzai 3.0)

**Feature:** FEAT-01KN5-8J24S2XW (mandatory-stage-gates)
**Design reference:** `work/design/kanbanzai-3.0-workflow-and-tooling-v2.md` §3, §6.1–6.3
**Status:** Draft

---

## Overview

Stage gates become mandatory for all feature lifecycle transitions, not just during multi-step `advance` operations. Every transition through the `entity(action: "transition")` tool — whether single-step or part of an advance sequence — MUST evaluate the prerequisite defined for that transition before allowing the state change. Two new gates are added for the review lifecycle (`developing → reviewing` and `reviewing → done`), and gates are added for rework transitions. An override mechanism allows agents to bypass gates with a recorded reason, and the `health` tool flags features that advanced via override. All gate failures produce actionable error messages that identify what failed, why, and what specific tool calls to make to resolve the issue.

---

## Scope

### In scope

- Mandatory gate evaluation on all feature lifecycle transitions (single-step and advance)
- The complete gate prerequisite table from the design (§3.3), covering all nine gated transitions
- New gates: `developing → reviewing` (task completeness), `reviewing → done` (review report), `needs-rework → developing` (rework task exists), `needs-rework → reviewing` (rework tasks complete)
- Override mechanism: `override` and `override_reason` parameters on `entity(action: "transition")`
- Override logging: persisted on the feature entity with the reason
- Override health flagging: `health` tool reports features that used gate overrides as attention items
- All gates default to `agent` override policy for 3.0
- Actionable error messages for every gate failure, following the design template (§6.2)
- Transitions to terminal states (`superseded`, `cancelled`) remain ungated

### Explicitly excluded

- Reading gate definitions from the binding registry (feature: binding-registry-gate-integration, §3.5)
- Override policy tiers from binding registry; the `checkpoint` override policy behaviour (feature: binding-registry-gate-integration, §3.5)
- Review cycle counter and iteration cap (feature: review-rework-loop, §4)
- Document structural checks at gates (feature: document-structural-checks, §10.4)
- Lifecycle state validation in `handoff` / `next` (feature: stage-aware-context-assembly, §7.2)
- Broad error message audit across all non-transition tool handlers (feature: tool-description-audit)
- Plan lifecycle gate enforcement (plans have a separate, simpler lifecycle not covered by these gates)
- Phase 1 (legacy) feature lifecycle transitions (gates apply only to Phase 2 document-driven lifecycle states)

---

## Functional Requirements

### Gate enforcement on all transitions

**FR-001:** The `entity(action: "transition")` tool MUST evaluate the gate prerequisite for a feature transition BEFORE persisting the state change, regardless of whether the `advance` parameter is set. A single-step transition (e.g., `entity(action: "transition", id: "FEAT-001", status: "dev-planning")`) MUST be subject to the same gate as the equivalent step within an `advance` sequence.

**Acceptance criteria:**
- Calling `entity(action: "transition", id: "FEAT-001", status: "dev-planning")` from `specifying` without an approved specification returns an error and the feature remains in `specifying`
- Calling `entity(action: "transition", id: "FEAT-001", status: "dev-planning", advance: true)` from `specifying` without an approved specification stops before `dev-planning` with the same gate failure reason
- Calling `entity(action: "transition", id: "FEAT-001", status: "dev-planning")` from `specifying` with an approved specification succeeds and the feature moves to `dev-planning`
- Both single-step and advance paths reference the same gate evaluation logic

---

**FR-002:** Transitions to terminal states (`superseded`, `cancelled`) MUST NOT be gated. These transitions MUST remain allowed from any non-terminal feature state without prerequisite checks.

**Acceptance criteria:**
- Calling `entity(action: "transition", id: "FEAT-001", status: "cancelled")` from any non-terminal state succeeds without gate evaluation
- Calling `entity(action: "transition", id: "FEAT-001", status: "superseded")` from any non-terminal state succeeds without gate evaluation

---

**FR-003:** Transitions with no prerequisites defined in the gate table (specifically `proposed → designing` and `reviewing → needs-rework`) MUST proceed without gate evaluation. These are ungated by design.

**Acceptance criteria:**
- Calling `entity(action: "transition", id: "FEAT-001", status: "designing")` from `proposed` succeeds with no document or task checks
- Calling `entity(action: "transition", id: "FEAT-001", status: "needs-rework")` from `reviewing` succeeds with no prerequisite checks

---

### Complete gate prerequisite table

**FR-004:** The `designing → specifying` transition MUST require an approved design document owned by the feature or its parent plan. The gate MUST check the following, in order: (1) the feature's own `design` field reference, (2) approved design documents owned by the feature, (3) approved design documents owned by the parent plan. If any check finds an approved design document, the gate is satisfied.

**Acceptance criteria:**
- A feature with an approved design document referenced in its `design` field passes the gate
- A feature without a `design` field reference but with an approved design document owned by it passes the gate
- A feature with no approved design document owned by it but with an approved design document owned by its parent plan passes the gate
- A feature with no approved design document at any level fails the gate
- A feature with a draft (unapproved) design document fails the gate

---

**FR-005:** The `specifying → dev-planning` transition MUST require an approved specification document owned by the feature or its parent plan. The gate MUST follow the same three-level lookup as FR-004: feature field reference, feature-owned documents, parent-plan-owned documents.

**Acceptance criteria:**
- A feature with an approved specification document passes the gate
- A feature with only a draft specification document fails the gate
- A feature with no specification document but an approved specification on its parent plan passes the gate

---

**FR-006:** The `dev-planning → developing` transition MUST require BOTH: (a) an approved dev-plan document owned by the feature or its parent plan (same three-level lookup as FR-004), AND (b) at least one child task exists for the feature.

**Acceptance criteria:**
- A feature with an approved dev-plan and at least one child task passes the gate
- A feature with an approved dev-plan but no child tasks fails the gate with a message about missing tasks
- A feature with child tasks but no approved dev-plan fails the gate with a message about the missing dev-plan
- A feature with neither an approved dev-plan nor child tasks fails the gate (the error message reports the first unmet prerequisite)

---

**FR-007:** The `developing → reviewing` transition MUST require that all child tasks of the feature are in a terminal state. Terminal task states are: `done`, `not-planned`, and `duplicate`.

**Acceptance criteria:**
- A feature whose child tasks are all in `done` passes the gate
- A feature with one child task in `active` and one in `done` fails the gate
- A feature with child tasks in a mix of `done`, `not-planned`, and `duplicate` passes the gate
- A feature with a child task in `queued` or `ready` fails the gate
- The error message identifies the non-terminal tasks by ID and their current status

---

**FR-008:** The `reviewing → done` transition MUST require BOTH: (a) a review report document is registered and owned by the feature (type `report`, any status — the report must exist, not necessarily be approved), AND (b) no unresolved blocking findings exist. For 3.0, the blocking-findings check is satisfied if a review report document exists; structured finding tracking is deferred to the review-rework-loop feature.

**Acceptance criteria:**
- A feature with a registered review report document passes the gate
- A feature with no review report document fails the gate
- The error message instructs the agent how to register a review report

---

**FR-009:** The `needs-rework → developing` transition MUST require that at least one rework task exists for the feature. A rework task is any child task of the feature that was created after the feature entered `needs-rework` status, OR any child task that is not in a terminal state (i.e., there is active work to do).

**Acceptance criteria:**
- A feature in `needs-rework` with at least one non-terminal child task passes the gate
- A feature in `needs-rework` where all child tasks are in terminal states fails the gate
- The error message instructs the agent to create a rework task

---

**FR-010:** The `needs-rework → reviewing` transition MUST require that all child tasks of the feature are in a terminal state (same check as FR-007). This prevents returning to review with incomplete rework.

**Acceptance criteria:**
- A feature in `needs-rework` with all child tasks in terminal states passes the gate
- A feature in `needs-rework` with any child task not in a terminal state fails the gate
- The error message identifies the non-terminal tasks by ID and status

---

### Override mechanism

**FR-011:** The `entity(action: "transition")` tool MUST accept two new optional parameters: `override` (boolean, default `false`) and `override_reason` (string). When `override` is `true` and `override_reason` is a non-empty string, a gate failure MUST be bypassed and the transition MUST proceed.

**Acceptance criteria:**
- `entity(action: "transition", id: "FEAT-001", status: "dev-planning", override: true, override_reason: "Spec exists in external system")` succeeds even when the specification gate is not satisfied
- The feature's status changes to `dev-planning` after the override
- The override parameters are accepted on single-step transitions

---

**FR-012:** When `override` is `true` but `override_reason` is empty or not provided, the tool MUST reject the transition with an error requiring a reason. The gate MUST NOT be bypassed without a reason.

**Acceptance criteria:**
- `entity(action: "transition", id: "FEAT-001", status: "dev-planning", override: true)` returns an error stating that `override_reason` is required
- `entity(action: "transition", id: "FEAT-001", status: "dev-planning", override: true, override_reason: "")` returns an error stating that `override_reason` is required
- The feature's status remains unchanged after the rejected call

---

**FR-013:** When `override` is `false` or not provided, the `override_reason` parameter MUST be ignored. Normal gate evaluation MUST apply.

**Acceptance criteria:**
- `entity(action: "transition", id: "FEAT-001", status: "dev-planning", override_reason: "some reason")` without `override: true` evaluates the gate normally and fails if prerequisites are not met

---

**FR-014:** Override transitions MUST be logged on the feature entity. The feature's persisted state MUST record the override, including: the transition that was overridden (from-status → to-status), the reason provided, and a timestamp.

**Acceptance criteria:**
- After an override transition, `entity(action: "get", id: "FEAT-001")` returns state that includes the override record
- The override record contains the from-status, to-status, reason, and timestamp
- Multiple overrides on the same feature accumulate (all are recorded, not just the latest)

---

**FR-015:** The `health` tool MUST flag features that have been advanced via override as attention items (warnings, not errors). The warning MUST identify the feature, the overridden transition, and the reason.

**Acceptance criteria:**
- Running `health` after an override transition includes a warning for the overridden feature
- The warning message contains the feature ID, the transition that was overridden, and the override reason
- A feature with no overrides does not produce an override-related warning
- A feature with multiple overrides produces one warning per override

---

**FR-016:** Override MUST work with the `advance` parameter. When `advance: true` and `override: true`, every gate in the advance path that fails MUST be bypassed using the single provided `override_reason`. Each bypassed gate MUST be individually logged.

**Acceptance criteria:**
- `entity(action: "transition", id: "FEAT-001", status: "developing", advance: true, override: true, override_reason: "Fast-tracking for demo")` advances through all intermediate states, bypassing any failing gates
- Each bypassed gate is recorded as a separate override entry on the feature
- The response indicates which gates were overridden during the advance

---

**FR-017:** For 3.0, all gates MUST default to the `agent` override policy. This means any caller can override any gate by providing `override: true` with a reason. The `checkpoint` policy (which would create a human checkpoint and block) is not implemented in this feature.

**Acceptance criteria:**
- Every gate in the prerequisite table can be overridden with `override: true` and a reason
- No gate creates a human checkpoint on override

---

### Actionable error messages

**FR-018:** Every gate failure MUST return an error message following the design template (§6.2):

```
Cannot transition {feature_id} from "{from_status}" to "{to_status}": {reason}.

To resolve:
  1. {recovery_step_1}
  2. {recovery_step_2}
```

The error MUST include: (1) the feature ID, (2) the current status, (3) the target status, (4) a specific explanation of what prerequisite was not met, and (5) at least one recovery step formatted as a tool call the agent can execute.

**Acceptance criteria:**
- Every gate failure message contains the feature ID
- Every gate failure message contains both the from-status and to-status
- Every gate failure message contains a "To resolve:" section with at least one recovery step
- Recovery steps are formatted as tool call examples (e.g., `doc(action: "list", owner: "FEAT-001", pending: true)`)

---

**FR-019:** The `designing → specifying` gate failure (no approved design document) MUST include recovery steps that guide the agent to: (1) check for pending documents, (2) approve an existing document, or (3) register a new design document.

**Acceptance criteria:**
- The error message includes a `doc(action: "list", ...)` call to check pending documents
- The error message includes a `doc(action: "approve", ...)` call pattern
- The error message includes a `doc(action: "register", ...)` call pattern with `type: "design"`

---

**FR-020:** The `specifying → dev-planning` gate failure (no approved specification) MUST include recovery steps analogous to FR-019 but referencing `type: "specification"`.

**Acceptance criteria:**
- The error message references `type: "specification"` in recovery steps
- The recovery steps include list, approve, and register patterns

---

**FR-021:** The `dev-planning → developing` gate failure MUST include recovery steps appropriate to the specific sub-prerequisite that failed. If the dev-plan document is missing, recovery steps guide toward document registration/approval. If child tasks are missing, the recovery step guides toward task creation or decomposition.

**Acceptance criteria:**
- When the dev-plan document is missing, recovery steps reference `doc(action: ...)` calls for the dev-plan
- When child tasks are missing, the recovery step references `decompose(action: "propose", feature_id: "FEAT-...")` or `entity(action: "create", type: "task", ...)`

---

**FR-022:** The `developing → reviewing` gate failure (non-terminal child tasks) MUST identify each non-terminal task by ID and status in the error message, and provide recovery steps to complete or cancel outstanding tasks.

**Acceptance criteria:**
- The error message lists each non-terminal child task with its ID and current status
- Recovery steps include `finish(task_id: "TASK-...")` and `entity(action: "transition", id: "TASK-...", status: "not-planned")` patterns

---

**FR-023:** The `reviewing → done` gate failure (no review report) MUST include recovery steps guiding the agent to register a review report document.

**Acceptance criteria:**
- The error message includes a `doc(action: "register", type: "report", owner: "FEAT-...")` call pattern
- The error message explains that a review report must be registered before closing the feature

---

**FR-024:** The `needs-rework → developing` gate failure (no rework tasks) MUST include a recovery step guiding the agent to create a rework task.

**Acceptance criteria:**
- The error message includes an `entity(action: "create", type: "task", parent_feature: "FEAT-...", ...)` call pattern
- The error message explains that at least one rework task must exist

---

**FR-025:** The `needs-rework → reviewing` gate failure (non-terminal tasks) MUST follow the same pattern as FR-022, identifying outstanding tasks and providing completion/cancellation recovery steps.

**Acceptance criteria:**
- The error message lists each non-terminal child task with its ID and current status
- Recovery steps are consistent with those in FR-022

---

**FR-026:** When a transition is rejected by a gate, the tool MUST return the error as a structured response (not an MCP protocol error). The response MUST include an `"error"` field with the actionable message and a `"gate_failed"` field identifying the transition. The feature's status MUST remain unchanged.

**Acceptance criteria:**
- A gate rejection returns a JSON response with `"error"` and `"gate_failed"` keys
- The `"gate_failed"` field contains `from_status` and `to_status`
- The feature entity's status is unchanged after the rejection (verified via `entity(action: "get")`)
- The MCP call itself succeeds (HTTP 200 / no protocol error); the gate failure is in the response body

---

---

## Non-Functional Requirements

**NFR-001:** Gate evaluation MUST add no more than 100ms of latency to a transition call under normal operating conditions (local filesystem state, fewer than 1000 entities). Gate checks involve reading document and task state from the local filesystem; no network calls are required.

**Acceptance criteria:**
- Benchmark tests for gate evaluation on a feature with 10 child tasks complete within 100ms

---

**NFR-002:** Gate enforcement MUST be backward-compatible with Phase 1 (legacy) feature lifecycle transitions. Phase 1 transitions (e.g., `draft → in-review → approved → in-progress`) MUST NOT be subject to Phase 2 gate prerequisites. Gate checks MUST only apply to Phase 2 document-driven lifecycle transitions.

**Acceptance criteria:**
- A Phase 1 feature transitioning from `draft` to `in-review` is not subject to any document gate
- A Phase 2 feature transitioning from `designing` to `specifying` is subject to the design document gate

---

**NFR-003:** The override mechanism MUST NOT introduce a security boundary. For 3.0, override is a trust-based mechanism — any caller that can call `entity(action: "transition")` can also override. Access control is deferred to the `checkpoint` policy tier in a future feature.

---

**NFR-004:** Error messages MUST be deterministic. The same gate failure under the same conditions MUST produce the same error message (modulo entity IDs and timestamps). This supports testing and agent reliability.

**Acceptance criteria:**
- Calling the same failing transition twice with the same state produces identical error messages

---

**NFR-005:** The existing `advance` parameter behaviour MUST be preserved. The `advance` mode MUST continue to walk through multiple states toward a target, with the change that gates now fire on every step (not just intermediate steps in the advance path). The advance mode MUST still stop at `reviewing` as a mandatory halt state.

**Acceptance criteria:**
- `advance: true` with a target of `done` still halts at `reviewing`
- `advance: true` checks gates at every intermediate step (existing behaviour, now also enforced on single-step)

---

---

## Acceptance Criteria

The following are aggregate acceptance criteria for the specification as a whole:

1. **No bypass path exists:** There is no combination of `entity(action: "transition")` parameters (excluding `override: true`) that allows a feature to transition past a gated boundary without satisfying the prerequisite. This MUST be verified for every gated transition in the prerequisite table.

2. **Override always requires a reason:** Every override produces a persisted record with a non-empty reason string. There is no way to override without providing a reason.

3. **Health surfaces all overrides:** Running `health` after any override transition includes a warning for every overridden gate. No override is silently accepted.

4. **Error messages are actionable:** Every gate failure message contains (a) what failed, (b) why, and (c) at least one tool call the agent can execute to resolve the issue. No gate failure produces a bare "invalid transition" message.

5. **Backward compatibility:** Phase 1 features, non-feature entities (tasks, bugs, plans, decisions, incidents), and terminal-state transitions are unaffected by gate enforcement.

6. **Advance mode consistency:** The `advance` mode produces identical gate evaluation results as the equivalent sequence of single-step transitions.

---

## Dependencies and Assumptions

1. **Document service:** The existing `DocumentService` with its `ListDocuments` and `GetDocument` methods is available and correctly returns document records filtered by owner, type, and status. No changes to the document service are required.

2. **Entity service:** The existing `EntityService` with its `List`, `Get`, and `UpdateStatus` methods is available. The `UpdateStatus` method validates transitions against the lifecycle state machine in `internal/validate/lifecycle.go`.

3. **Existing gate logic:** The existing `CheckFeatureGate` function in `internal/service/prereq.go` provides the foundation for document and task gates. It currently handles `designing`, `specifying`, `dev-planning`, and `developing` gates. New gates for `reviewing → done`, `needs-rework → developing`, and `needs-rework → reviewing` must be added.

4. **Feature entity model:** The `model.Feature` struct can be extended to store override records (a list of override entries with transition, reason, and timestamp).

5. **Health check infrastructure:** The existing `CheckHealth` function in `internal/validate/health.go` supports the `ValidationWarning` type used for non-blocking health issues. Override warnings will use this existing mechanism.

6. **Task terminal states:** Terminal task states are defined in `internal/validate/lifecycle.go` as `done`, `not-planned`, and `duplicate`. This definition is authoritative and must be used by the task-completeness gates (FR-007, FR-010).

7. **Phase 2 lifecycle:** Gate enforcement targets only the Phase 2 document-driven lifecycle path: `proposed → designing → specifying → dev-planning → developing → reviewing → done`, plus the `needs-rework` loop. The allowed transitions map in `internal/validate/lifecycle.go` defines the valid transitions.

8. **No binding registry dependency:** For 3.0, gate prerequisites are hardcoded in the service layer. The binding registry integration (which would allow gate definitions to be read from `stage-bindings.yaml`) is a separate feature and not a prerequisite for this work.