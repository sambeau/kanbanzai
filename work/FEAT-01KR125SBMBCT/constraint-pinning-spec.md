| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-07                     |
| Author | spec-author                     |
| Status | draft                          |
| Feature| FEAT-01KR125SBMBCT              |
| Design | P55-orchestrator-context-hygiene/design-p55-design-orchestrator-context-hygiene |

# Specification: Constraint Pinning in Pipeline

## Overview

This feature adds an orchestrator role reminder to every `next` and `handoff` response. The reminder leverages the recency peak of the U-shaped attention curve (Liu et al. 2024) to maintain role awareness across long sessions. Without pinning, the orchestrator's identity constraint — stated once at session start — falls into the attention valley by mid-session and is overridden by recency-weighted skill instructions. The P50 incident demonstrated this failure mode directly.

### Problem

The orchestrator's role constraint ("coordinate, don't implement") is loaded once at session start from `orchestrator.yaml`. By mid-session, accumulated implementation context pushes this constraint into the attention valley. The orchestrator then defaults to general-purpose problem-solving behaviour — reading code, self-reviewing, and forgetting close-out steps.

### Design References

This specification implements Decision 4 and Component 4 from the approved design `P55-orchestrator-context-hygiene/design-p55-design-orchestrator-context-hygiene`:

- **Decision 4:** Implement constraint pinning in next/handoff responses
- **Component 4:** Constraint Pinning in handoff/next Responses

The reminder text is specified in the design:

> **Role reminder:** You are the orchestrator — coordinate, dispatch, verify. Do not investigate implementation code. Sub-agents handle all code reading and writing via `handoff`.

### Related Specifications

- **FEAT-01KR12539CXH6 (Orchestrator role hardening)** — Adds the anti-pattern and hard constraint that this feature reinforces via pinning. F2 depends on F1 for the anti-pattern definition.
- **P52-fast-track-orchestration** — Fast-track profile that also benefits from constraint pinning.

## Scope

### In Scope

- Adding a role reminder string to the assembled context output of the `next` tool when the role is `orchestrator`
- Adding the same role reminder string to the assembled context output of the `handoff` tool when the role is `orchestrator`
- The reminder SHALL appear in every response — no first-response-only logic
- The reminder SHALL be injected at the context assembly layer, not in individual tool handlers, so both `next` and `handoff` benefit from a single implementation
- The reminder SHALL be included in the `constraints` field of the assembled context output

### Out of Scope

- Adding role reminders for any role other than `orchestrator`
- Modifying the reminder text per-feature or per-task (static text only)
- Runtime enforcement of the reminder content (procedural, not tool-blocking)
- Any changes to the `develop` tool (dispatch mode) — constraint pinning applies to `next` and `handoff` only
- Token budget adjustment for the reminder (the 15–20 token cost is absorbed within existing budget)

## Functional Requirements

### REQ-001: Role Reminder Constant

A constant string SHALL be defined in the context assembly package with the exact text specified in the design:

> You are the orchestrator — coordinate, dispatch, verify. Do not investigate implementation code. Sub-agents handle all code reading and writing via `handoff`.

The constant SHALL be named `OrchestratorRoleReminder` and SHALL be defined in `internal/context/assemble.go`.

### REQ-002: Injection in next Claim Mode

When the `next` tool is called in claim mode (`id` parameter provided) and the assembled context role is `orchestrator`, the role reminder SHALL appear in the `constraints` field of the response.

### REQ-003: Injection in handoff

When the `handoff` tool is called and the assembled context role is `orchestrator`, the role reminder SHALL appear in the `constraints` field of the response.

### REQ-004: Every Response

The reminder SHALL appear in every `next` and `handoff` response where the role is `orchestrator`. It SHALL NOT be limited to the first response in a session. There is no session-tracking logic — the reminder is stateless and unconditional for the orchestrator role.

### REQ-005: Role-Specific Only

The reminder SHALL NOT appear in responses for any role other than `orchestrator`. Roles `implementer`, `implementer-go`, `reviewer`, `reviewer-conformance`, `reviewer-quality`, `reviewer-security`, `reviewer-testing`, `architect`, `spec-author`, `researcher`, `documenter`, `verifier`, and any future role SHALL NOT receive the orchestrator role reminder.

### REQ-006: Constraints Field Format

The reminder SHALL be added to the existing `constraints` field as a new entry. It SHALL NOT replace or remove any existing constraints. The entry SHALL include a `type` field set to `"role_reminder"` to distinguish it from other constraint types.

### REQ-007: Backwards Compatibility

The addition of the role reminder SHALL NOT change the structure, field names, or field ordering of the `next` or `handoff` response. Existing consumers of these responses SHALL continue to function without modification.

### REQ-008: Unit Test Coverage

Unit tests SHALL verify:
- The reminder appears in `next` claim-mode output when role is `orchestrator`
- The reminder appears in `handoff` output when role is `orchestrator`
- The reminder does NOT appear when role is `implementer` or any non-orchestrator role
- The reminder appears in every response, not just the first

## Non-Functional Requirements

### NFR-001: Token Budget

The reminder SHALL be 15–25 tokens in length. The existing byte budget and token budget logic in `internal/context/assemble.go` SHALL NOT be modified to accommodate the reminder — it fits within the existing budget.

### NFR-002: Single Implementation Point

The reminder injection logic SHALL reside in a single location (the context assembly layer) so that both `next` and `handoff` benefit without duplicated code. Tool handlers SHALL NOT contain role-reminder logic.

### NFR-003: No Stateful Tracking

The implementation SHALL NOT track sessions, invocation counts, or whether the reminder has been shown before. It is stateless and unconditional.

## Acceptance Criteria

- [ ] **AC-001:** `OrchestratorRoleReminder` constant is defined in `internal/context/assemble.go` with the exact text from the design
- [ ] **AC-002:** The role reminder appears in the `constraints` field when `next` is called in claim mode with role `orchestrator`
- [ ] **AC-003:** The role reminder appears in the `constraints` field when `handoff` is called with role `orchestrator`
- [ ] **AC-004:** The role reminder appears in every `next` claim-mode response for the orchestrator role — tested with three consecutive calls
- [ ] **AC-005:** The role reminder appears in every `handoff` response for the orchestrator role — tested with three consecutive calls
- [ ] **AC-006:** The role reminder does NOT appear in `next` claim-mode output when role is `implementer`
- [ ] **AC-007:** The role reminder does NOT appear in `handoff` output when role is `implementer`
- [ ] **AC-008:** The role reminder does NOT appear in responses for `reviewer`, `architect`, `spec-author`, or `verifier` roles
- [ ] **AC-009:** The constraints entry has `type: "role_reminder"`
- [ ] **AC-010:** Existing constraints in the response are preserved alongside the role reminder
- [ ] **AC-011:** Existing `next` and `handoff` consumers continue to function without modification (backwards compatibility)
- [ ] **AC-012:** `go test ./...` passes, including new unit tests for the role reminder

## Verification Plan

| Requirement | Verification Method | Acceptance Criterion |
|-------------|-------------------|---------------------|
| REQ-001 | Code review of `internal/context/assemble.go` for constant definition | AC-001 |
| REQ-002 | Test `next` claim mode with orchestrator role, assert constraints contain reminder | AC-002 |
| REQ-003 | Test `handoff` with orchestrator role, assert constraints contain reminder | AC-003 |
| REQ-004 | Three consecutive `next` calls, assert reminder in all three | AC-004 |
| REQ-004 | Three consecutive `handoff` calls, assert reminder in all three | AC-005 |
| REQ-005 | Test with `implementer` role, assert reminder absent | AC-006, AC-007 |
| REQ-005 | Test with `reviewer`, `architect`, `spec-author`, `verifier` roles | AC-008 |
| REQ-006 | Assert constraints entry type field equals "role_reminder" | AC-009 |
| REQ-006 | Assert existing constraints remain present | AC-010 |
| REQ-007 | Existing integration tests pass without modification | AC-011 |
| REQ-008 | `go test ./...` passes with new unit tests | AC-012 |
