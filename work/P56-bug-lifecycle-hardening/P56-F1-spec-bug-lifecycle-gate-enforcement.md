| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-07                     |
| Author | spec-author                     |
| Status | draft                          |
| Plan   | P56-bug-lifecycle-hardening    |
| Feature | FEAT-01KR12RE970R8             |

# Specification: Bug Lifecycle Gate Enforcement

## Related Work

- **P56-design-bug-lifecycle-hardening.md** (P56-bug-lifecycle-hardening/design-p56-design-bug-lifecycle-hardening) — Design document for the overall bug lifecycle hardening plan. This spec implements Components A, C, and D from that design.
- **internal/validate/lifecycle.go** — Existing lifecycle state machine. This spec extends it with bug stop-states and gate enforcement.
- **internal/service/prereq.go** — Existing feature gate enforcement (`CheckTransitionGate`). This spec adds a bug equivalent.
- **internal/mcp/entity_tool.go** — Existing `entityTransitionAction`. This spec wires bug gate checks into the transition handler.
- **P55-design-orchestrator-context-hygiene.md** — Component 5 (Fast-Track Review Dispatch) provides the orchestrator-side review mechanism that the `needs-review` stop-state enforces server-side.

**Constraining decisions:**
- P55 Decision 6: Dispatch review sub-agents in fast-track close-out. This spec makes that enforceable at the server level.
- P56 Decision 2: `needs-review` is a mandatory stop-state, not a gate-mode toggle.
- P56 Decision 8: The verifier is gate-dispatched, not orchestrator-dispatched.

## Overview

The bug lifecycle currently has no gate enforcement. Bugs can skip `needs-review`, have no review cycle tracking, and pass through `entity(action: "transition")` with only the basic lifecycle graph check. This specification adds mandatory stop-states, lifecycle gate enforcement, review cycle tracking, and renames the `verified` stage to `verifying` to match the feature lifecycle convention.

## Scope

**In scope:**
- Rename `BugStatusVerified` to `BugStatusVerifying` in the bug lifecycle model
- Add `bugStopStates` map with `needs-review` as a mandatory halt
- Add `CheckBugTransitionGate` with gates at all four transitions from `in-progress` onward
- Add `review_cycle` tracking for bugs (increment on rework, cap enforcement)
- Wire bug gate checks into `entityTransitionAction` for `entityType == "bug"`
- Feature lifecycle behaviour is unchanged

**Out of scope:**
- Auto-generated specs (F2)
- Worktree enforcement (F3)
- Close-out verifier dispatch (F4 — depends on P55 Component 7)
- Adding `advance` support for bugs

## Functional Requirements

### Pillar A — Lifecycle Stage Rename

**FR-001:** The `BugStatusVerified` constant in `internal/model/entities.go` MUST be renamed to `BugStatusVerifying`. The value MUST remain `"verifying"`.

**FR-001a:** All references to `BugStatusVerified` across the codebase MUST be updated to `BugStatusVerifying`.

**FR-001b:** The bug lifecycle state machine in `internal/validate/lifecycle.go` MUST be updated: the transition `needs-review → verified` becomes `needs-review → verifying`, and `verified → closed` becomes `verifying → closed`.

**FR-001c:** The `kbzschema/schema.go` bug schema MUST list `"verifying"` instead of `"verified"` in the status enum.

**Acceptance criteria:**
- `BugStatusVerified` does not appear in any Go source file
- `BugStatusVerifying` appears in `internal/model/entities.go` with value `"verifying"`
- `entity(action: "transition", id: "BUG-...", status: "verifying")` succeeds from `needs-review`
- `entity(action: "transition", id: "BUG-...", status: "verified")` returns an error (unknown state)
- All existing tests that reference `BugStatusVerified` are updated and pass

### Pillar B — Mandatory Stop-State

**FR-002:** A new `bugStopStates` map MUST be added to `internal/validate/lifecycle.go`, mirroring `advanceStopStates` for features. The map MUST include `BugStatusNeedsReview` as a mandatory halt.

**FR-003:** When a bug transitions into `needs-review`, the `entityTransitionAction` handler MUST record that the bug is in a stop-state. The bug cannot be auto-advanced out of `needs-review`.

**FR-004:** A health check MUST flag any bug that reached `closed` or `verifying` without having passed through `needs-review`. The health check severity is `warning`.

**Acceptance criteria:**
- `bugStopStates` contains `"needs-review"`
- A bug in `needs-review` cannot be transitioned to `verifying` without satisfying the review report gate (FR-008)
- Health check reports a warning for bugs that skipped `needs-review`

### Pillar C — Lifecycle Gate Enforcement

**FR-005:** A new function `CheckBugTransitionGate` MUST be added to `internal/service/prereq.go`, accepting `(from, to string, bug *model.Bug, docSvc *DocumentService, entitySvc *EntityService)` and returning a `GateResult`.

**FR-006:** The `in-progress → needs-review` gate MUST verify that the bug's worktree has at least one commit beyond the base branch. If no worktree exists or no commits are present, the gate MUST fail with the reason `"no fix commits found on worktree"`.

**FR-007:** The `needs-review → verifying` gate MUST verify that:
- At least one report document is registered and owned by the bug (Component C from the design)
- `go test ./...` passes on the worktree branch

If either check fails, the gate MUST fail with a specific reason identifying which check failed.

**FR-008:** The `needs-review → in-progress` gate MUST:
- Increment `review_cycle` on the bug entity
- If `review_cycle >= tier.MaxCycles`, block the transition and escalate to a human checkpoint
- If `review_cycle < tier.MaxCycles`, allow the transition (rework is permitted)

**FR-009:** The `verifying → closed` gate MUST dispatch a verifier sub-agent (defined in F4) and block the transition unless the verifier returns all-pass on all 8 DoD items. Until F4 is implemented, this gate MUST be a no-op placeholder that returns pass with the reason `"verifier not yet implemented — see F4"`.

**FR-010:** All bug gate checks MUST be wired into `entityTransitionAction` in `internal/mcp/entity_tool.go`. When `entityType == "bug"` and the transition is from `in-progress` or later, the handler MUST call `CheckBugTransitionGate` and reject the transition on gate failure.

**FR-011:** Gate failure responses for bugs MUST follow the same format as feature gate failures: an error message with `gate_failed` details including `from_status`, `to_status`, and `reason`.

**Acceptance criteria:**
- `CheckBugTransitionGate` exists and is called during bug transitions
- Transitioning `in-progress → needs-review` without commits on the worktree returns a gate failure
- Transitioning `needs-review → verifying` without a review report returns a gate failure
- Transitioning `needs-review → in-progress` at the review cycle cap returns a gate failure with a checkpoint
- Transitioning `verifying → closed` succeeds (placeholder pass until F4)
- Gate failure responses include `gate_failed` with `from_status`, `to_status`, and `reason`

### Pillar D — Review Cycle Tracking

**FR-012:** A `review_cycle` field MUST be added to the bug entity model in `internal/model/entities.go`. The field is an integer, defaulting to 0.

**FR-013:** Each `needs-review → in-progress` transition MUST increment `review_cycle` by 1. The increment is performed by `CheckBugTransitionGate` before evaluating the cap.

**FR-014:** When `review_cycle >= tier.MaxCycles`, the `needs-review → in-progress` gate MUST:
- Block the transition
- Set `blocked_reason` on the bug entity
- Create a human checkpoint with the question: `"Bug {id} has reached the review iteration cap ({cycle}/{max}). What should happen next?"`
- Return a gate failure with `ReviewCapReached: true`

**FR-015:** The `review_cycle` field MUST NOT be incremented by any other transition.

**Acceptance criteria:**
- New bugs have `review_cycle: 0`
- After one rework cycle, `review_cycle` is 1
- After reaching `MaxCycles` (2 for `bug_fix`), further rework transitions are blocked
- A human checkpoint is created when the cap is reached
- Transitions unrelated to rework do not change `review_cycle`

## Non-Functional Requirements

**NFR-001:** `CheckBugTransitionGate` MUST return in under 50ms for synchronous gates (FR-006, FR-007). The `verifying → closed` gate (FR-009) is exempt — it dispatches a sub-agent.

**NFR-002:** The bug gate system MUST NOT change the behaviour of feature transitions. Existing feature gate tests MUST continue to pass without modification.

**NFR-003:** The `BugStatusVerified` rename MUST be backward-compatible for existing bug state files. Bugs on disk with `status: verified` MUST be readable and their status MUST be interpreted as `verifying`.

## Acceptance Criteria (Cross-Cutting)

**AC-001:** A bug walks the full lifecycle `in-progress → needs-review → verifying → closed` with all gates passing.

**AC-002:** A bug skips `needs-review` (goes `in-progress → verifying`) — the health check flags it.

**AC-003:** A bug attempts `needs-review → verifying` without a review report — the gate blocks it with a specific reason.

**AC-004:** A bug reaches the review cycle cap — rework is blocked and a checkpoint created.

**AC-005:** A bug uses the old `status: verified` in its state file — the system reads it as `verifying`.

**AC-006:** Feature transitions are unaffected — existing feature gate tests pass without changes.

## Dependencies and Assumptions

**Dependencies:**
- F2 (Bug Spec and Document Infrastructure) — `CheckBugTransitionGate` references the spec document path defined in F2 for the `needs-review → verifying` gate's review report check.
- F4 (Bug Close-Out Verification) — The `verifying → closed` gate dispatches the verifier defined in F4. Until F4 is implemented, this gate is a pass-through placeholder.
- P55 Component 7 — The verifier role and `verify-closeout` skill are defined by P55.

**Assumptions:**
1. The `bug_fix` tier has `MaxCycles: 2` as defined in `DefaultFastTrackConfig()`.
2. The worktree auto-creation hook (`WorktreeTransitionHook.handleBugInProgress`) fires before `CheckBugTransitionGate` runs, so the worktree exists by the time the gate checks for commits.
3. Existing bugs on disk may have `status: verified` — these are interpreted as `verifying` per NFR-003.
