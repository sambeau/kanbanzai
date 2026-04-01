# Specification: Review-Rework Loop Formalisation

| Field   | Value                                                              |
|---------|--------------------------------------------------------------------|
| Status  | Draft                                                              |
| Feature | FEAT-01KN5-8J2606B0 (review-rework-loop)                          |
| Design  | `work/design/kanbanzai-3.0-workflow-and-tooling-v2.md` §4          |

---

## 1. Overview

This specification formalises the interaction between the `reviewing` and `needs-rework` feature lifecycle states by adding a review cycle counter to the feature entity, enforcing an iteration cap that blocks infinite refinement loops, and signalling focused re-review context when a feature returns to `reviewing` after rework. Together these changes make review-rework iteration observable, bounded, and contextually efficient.

---

## 2. Scope

### 2.1 In Scope

- A `review_cycle` counter field on the feature entity model, incremented on each transition into `reviewing`.
- Persistence of `review_cycle` in the feature entity YAML record.
- Visibility of `review_cycle` in the `status` tool output for feature detail.
- A hardcoded iteration cap (`max_review_cycles = 3`) applied when a review verdict is "fail" at the cap.
- Blocking behaviour when the cap is reached: prevent auto-transition to `needs-rework`, set a blocked reason, and create a human checkpoint automatically.
- A `re_review` signal in context assembly output when `review_cycle ≥ 2`, including metadata about what to surface for focused re-review.

### 2.2 Out of Scope

- **Reviewing skill methodology.** The content and structure of the reviewing skill is owned by the skills redesign feature. This spec covers the counter and cap mechanics, not how reviews are conducted.
- **Stage gate enforcement on transitions.** The mandatory-stage-gates feature owns gate prerequisite evaluation. This spec adds the counter and cap, not the gate prerequisites themselves.
- **Binding registry runtime integration.** For 3.0, `max_review_cycles` is hardcoded to 3. Reading it from the binding registry at runtime is a separate feature.
- **Stage-aware context assembly pipeline changes.** The context assembly pipeline spec owns the assembly steps. This spec defines the re-review signal that the pipeline must honour, but does not specify the pipeline itself.
- **Review verdict recording.** How the review verdict is captured and stored is outside this spec. This spec acts on the verdict outcome.

---

## 3. Functional Requirements

### FR-001: Review Cycle Counter Field

The feature entity model MUST include a `review_cycle` field of integer type. The field MUST default to `0` for newly created features and for existing features that do not have the field set (backward compatibility).

**Acceptance criteria:**

- A newly created feature has `review_cycle` equal to `0` (or absent, treated as `0`).
- The field is serialised to YAML as `review_cycle` and deserialised back correctly.
- Existing feature YAML files without a `review_cycle` field load successfully with the value treated as `0`.

---

### FR-002: Counter Increment on Transition to Reviewing

The `review_cycle` counter MUST increment by exactly 1 each time a feature transitions INTO the `reviewing` status, regardless of the source state. The increment MUST occur as part of the transition — it MUST NOT be possible to enter `reviewing` without the counter incrementing.

| Scenario | Before | After |
|---|---|---|
| First entry to `reviewing` (from `developing`) | `review_cycle: 0` | `review_cycle: 1` |
| Return after rework (`needs-rework → developing → reviewing`) | `review_cycle: 1` | `review_cycle: 2` |
| Direct re-entry (`needs-rework → reviewing`) | `review_cycle: N` | `review_cycle: N+1` |

**Acceptance criteria:**

- Transitioning a feature from `developing` to `reviewing` when `review_cycle` is `0` results in `review_cycle` equal to `1`.
- Transitioning a feature from `developing` to `reviewing` when `review_cycle` is `1` results in `review_cycle` equal to `2`.
- Transitioning a feature from `needs-rework` directly to `reviewing` increments the counter by 1.
- The counter is persisted to the feature entity record after the transition.
- Transitions to any status other than `reviewing` do NOT modify the counter.

---

### FR-003: Counter Visible in Status Output

The `status` tool MUST include the `review_cycle` value in the feature detail response. The field MUST appear in the feature detail JSON output when the value is greater than `0`.

**Acceptance criteria:**

- Calling `status` with a feature ID where `review_cycle` is `0` omits the field (or includes it as `0`).
- Calling `status` with a feature ID where `review_cycle` is `2` returns a response containing `"review_cycle": 2` in the feature object.
- The field is present in the feature detail response, not only in nested structures.

---

### FR-004: Hardcoded Iteration Cap

The system MUST enforce an iteration cap of `max_review_cycles = 3`. This value MUST be a named constant in the codebase, not a magic number inlined in logic, to facilitate future binding registry integration.

**Acceptance criteria:**

- A named constant (e.g. `DefaultMaxReviewCycles = 3`) exists in the codebase.
- The cap check logic references this constant, not a literal `3`.

---

### FR-005: Cap-Reached Blocking Behaviour

When a feature's `review_cycle` equals `max_review_cycles` AND a review verdict of "fail" is recorded, the system MUST NOT auto-transition the feature to `needs-rework`. Instead, the following sequence MUST occur:

1. The review verdict is recorded normally (this is outside our scope — the verdict recording itself is unaffected).
2. The system detects that `review_cycle == max_review_cycles` and the verdict is "fail".
3. The feature MUST NOT transition to `needs-rework`.
4. The feature MUST enter a blocked state. The blocked reason MUST be set to a message that includes: the current cycle count, the cap value, and the three available human decisions. The message MUST match the pattern: `"Review iteration cap reached (N/N). Human decision required: accept with known issues, rework with revised scope, or cancel."`
5. A human checkpoint MUST be created automatically (see FR-006).

**Acceptance criteria:**

- A feature at `review_cycle: 3` (with `max_review_cycles = 3`) that receives a "fail" verdict does NOT transition to `needs-rework`.
- The feature's blocked reason contains "Review iteration cap reached (3/3)".
- The feature's blocked reason contains all three decision options: "accept with known issues", "rework with revised scope", "cancel".
- A feature at `review_cycle: 2` (below cap) that receives a "fail" verdict transitions to `needs-rework` normally.
- A feature at `review_cycle: 3` that receives a "pass" verdict transitions to `done` normally — the cap only applies to "fail" verdicts.

---

### FR-006: Automatic Checkpoint Creation at Cap

When the cap-reached blocking behaviour (FR-005) is triggered, the system MUST automatically create a human checkpoint via the checkpoint store. The checkpoint MUST include:

- **question:** A message identifying that the review iteration cap has been reached and requesting a human decision.
- **context:** The feature ID, current review cycle, the cap value, and a summary of the three options (accept with known issues, rework with revised scope, or cancel).
- **created_by:** The system identifier (e.g. `"system"` or the agent identity that triggered the review).

**Acceptance criteria:**

- When FR-005 triggers, exactly one checkpoint is created.
- The checkpoint's `question` field references the review iteration cap.
- The checkpoint's `context` field includes the feature ID and cycle count.
- The checkpoint status is `pending` immediately after creation.
- If the cap is NOT reached, no checkpoint is created for the review verdict.

---

### FR-007: Blocked State Representation

The feature entity MUST support a blocked state when the iteration cap is reached. This MUST be represented by one of the following approaches (implementation decides which):

- (a) A `blocked_reason` field on the feature entity that, when non-empty, indicates the feature is blocked. The feature remains in the `reviewing` status but is functionally blocked.
- (b) Transition to an existing or new blocked-like status with the reason stored as metadata.

Regardless of representation, the following MUST hold:

1. The `status` tool output MUST clearly indicate that the feature is blocked and include the blocked reason.
2. The feature MUST NOT accept a transition to `needs-rework` while blocked due to the iteration cap.
3. After a human responds to the checkpoint, the block can be resolved and the feature can transition according to the human's decision.

**Acceptance criteria:**

- A feature blocked by the iteration cap shows a blocked indicator and reason in `status` output.
- Attempting to transition a cap-blocked feature to `needs-rework` returns an error explaining that human decision is required.
- The blocked reason text matches the format specified in FR-005.

---

### FR-008: Focused Re-Review Signal

When a feature transitions to `reviewing` with `review_cycle ≥ 2`, the system MUST mark the feature as requiring a **focused re-review**. This signal MUST be available to the context assembly system (the `handoff` tool) so that it can adjust what content is surfaced.

The re-review signal MUST convey:

- That this is a re-review (cycle N), not a first review.
- That context assembly should include only rework tasks and their changes, not the full implementation.
- That context assembly should include the previous review findings that triggered the rework.
- That context assembly should include rework task descriptions showing what was supposed to change.

**Acceptance criteria:**

- A feature entering `reviewing` at `review_cycle: 1` is NOT marked as a re-review.
- A feature entering `reviewing` at `review_cycle: 2` IS marked as a re-review.
- The re-review signal includes the current cycle number.
- The `handoff` tool (or context assembly system) can read the re-review signal from the feature entity to adjust its output.

---

### FR-009: Counter Survives Non-Review Transitions

The `review_cycle` counter MUST NOT be reset or modified by any transition other than a transition INTO `reviewing` (which increments it per FR-002). Specifically:

- Transitioning from `reviewing` to `needs-rework` MUST NOT change the counter.
- Transitioning from `needs-rework` to `developing` MUST NOT change the counter.
- Transitioning from `reviewing` to `done` MUST NOT change the counter.
- Cancelling or superseding a feature MUST NOT change the counter.

**Acceptance criteria:**

- A feature at `review_cycle: 2` that transitions to `needs-rework` still has `review_cycle: 2`.
- A feature at `review_cycle: 2` in `needs-rework` that transitions to `developing` still has `review_cycle: 2`.
- A feature at `review_cycle: 1` that transitions to `done` still has `review_cycle: 1`.

---

### FR-010: Entity Tool Exposes Review Cycle

The `entity(action: "get")` response for a feature MUST include the `review_cycle` field so that tools and agents can read the current cycle count without going through the `status` tool.

**Acceptance criteria:**

- Calling `entity(action: "get", id: "<feature-id>")` on a feature with `review_cycle: 2` returns a response containing the `review_cycle` value.

---

## 4. Non-Functional Requirements

### NFR-001: Backward Compatibility

Existing feature entities created before this change MUST continue to load and function correctly. Missing `review_cycle` fields MUST be treated as `0`. No migration step is required.

**Acceptance criteria:**

- A feature YAML file without a `review_cycle` field loads without error.
- All existing tests pass without modification to the feature entity loading logic (beyond adding the new field with a zero default).

---

### NFR-002: Performance

The counter increment and cap check MUST add negligible overhead to the transition operation. The check is a simple integer comparison and MUST NOT involve additional file I/O, network calls, or expensive computation beyond what the transition already performs.

**Acceptance criteria:**

- The transition to `reviewing` completes within the same latency bounds as any other feature transition.

---

### NFR-003: Determinism

Given the same feature state and the same review verdict, the cap-reached behaviour MUST be deterministic. There MUST be no race conditions between the counter increment, the cap check, and the checkpoint creation — these operations MUST execute as a single logical unit.

**Acceptance criteria:**

- Two identical transitions on features with the same state produce the same outcome.
- The counter is incremented before the cap is checked (the check uses the post-increment value).

---

### NFR-004: Constant Extractability

The `max_review_cycles` value MUST be defined as a named constant so that a future binding registry integration can replace it with a runtime lookup without changing the cap-check logic structure.

**Acceptance criteria:**

- The constant is defined in a single location and referenced wherever the cap is checked.

---

## 5. Acceptance Criteria

This section consolidates the verification approach for each requirement.

| Requirement | Verification Method |
|---|---|
| FR-001 | Unit test: create feature, verify `review_cycle` is 0. Load legacy YAML without field, verify 0. |
| FR-002 | Unit test: transition feature to `reviewing` from multiple source states, verify counter increments by exactly 1 each time. Verify non-reviewing transitions leave counter unchanged. |
| FR-003 | Integration test: call `status` tool with feature at various cycle counts, verify JSON output contains correct `review_cycle` value. |
| FR-004 | Code inspection: verify named constant exists and is referenced by cap-check logic. |
| FR-005 | Unit test: feature at cap with "fail" verdict does not transition to `needs-rework`, blocked reason matches expected format. Feature below cap with "fail" verdict transitions normally. Feature at cap with "pass" verdict transitions to `done`. |
| FR-006 | Unit test: verify checkpoint is created when FR-005 triggers. Verify checkpoint fields contain feature ID, cycle count, and decision options. Verify no checkpoint created when below cap. |
| FR-007 | Integration test: `status` output shows blocked indicator and reason. Transition to `needs-rework` is rejected while blocked. |
| FR-008 | Unit test: feature at cycle ≥ 2 entering `reviewing` has re-review signal set. Feature at cycle 1 does not. Verify signal includes cycle number. |
| FR-009 | Unit test: verify counter is unchanged after transitions to `needs-rework`, `developing`, `done`, `cancelled`, `superseded`. |
| FR-010 | Integration test: `entity(action: "get")` response includes `review_cycle` field. |

---

## 6. Dependencies and Assumptions

### Dependencies

- **Feature entity model** (`internal/model/entities.go`): The `Feature` struct must be extended with the `review_cycle` field.
- **Feature entity storage** (`internal/storage/`): The YAML serialisation must handle the new field with backward-compatible zero default.
- **Lifecycle validation** (`internal/validate/lifecycle.go`): The transition logic must support the counter increment hook when the target status is `reviewing`.
- **Checkpoint store** (`internal/checkpoint/`): The existing checkpoint creation API is used to create the automatic checkpoint at cap. No changes to the checkpoint store are required.
- **Status tool** (`internal/mcp/status_tool.go`): The `featureDetail` and `featureInfo` response structs must include the `review_cycle` field.
- **Entity tool** (`internal/mcp/entity_tool.go`): The `get` action response must surface the `review_cycle` field.

### Assumptions

- The review verdict ("pass" or "fail") is available at the point where the system decides whether to transition to `needs-rework` or `done`. This spec does not define how the verdict is captured — it acts on the outcome.
- The `needs-rework` transition after a "fail" verdict is triggered by an agent or tool action, not by a fully automatic system process. The cap check intercepts this action.
- A hardcoded cap of 3 is acceptable for the 3.0 release. Binding registry integration will make this configurable in a future release.
- The checkpoint store is initialised and available in all execution contexts where a review verdict can trigger the cap behaviour.