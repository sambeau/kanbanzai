| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-07                     |
| Author | spec-author                     |
| Status | approved |
| Plan   | P56-bug-lifecycle-hardening    |
| Feature | FEAT-01KR12RE98N93             |

# Specification: Bug Close-Out Verification

## Related Work

- **P56-design-bug-lifecycle-hardening.md** (P56-bug-lifecycle-hardening/design-p56-design-bug-lifecycle-hardening) — Design document. This spec implements Component G.
- **P55-design-orchestrator-context-hygiene.md** (P55-orchestrator-context-hygiene/design-p55-design-orchestrator-context-hygiene) — Component 7 defines the `verifier` role and `verify-closeout` skill that this spec depends on.
- **F1 — Bug Lifecycle Gate Enforcement** — The `verifying → closed` gate (FR-009) dispatches the verifier defined in this spec.
- **P56 Definition of Done** — The 8-item checklist is derived directly from the DoD in the P56 design.

**Constraining decisions:**
- P56 Decision 7: Close-out verification is delegated to a clean-context sub-agent.
- P56 Decision 8: The verifier is gate-dispatched, not orchestrator-dispatched.
- P55 Component 7: Defines the `verifier` role and `verify-closeout` skill that this spec adopts.

## Overview

A bug cannot be `closed` until it is verified as meeting all eight conditions in the Definition of Done. This specification defines a gate-dispatched verifier sub-agent that runs at the `verifying → closed` transition, checks each DoD item independently with concrete verification actions, and returns a structured pass/fail report. The gate blocks the transition unless all items pass.

This feature depends on P55 Component 7 for the `verifier` role and `verify-closeout` skill. The bug verifier uses the same role and skill with an adapted 8-item checklist.

## Scope

**In scope:**
- Gate-dispatched verifier sub-agent at `verifying → closed` transition
- Bug-adapted 8-item DoD checklist
- Structured pass/fail report with evidence per item
- Integration with F1's `CheckBugTransitionGate` for the `verifying → closed` gate

**Out of scope:**
- Defining the `verifier` role (P55 Component 7)
- Defining the `verify-closeout` skill (P55 Component 7)
- Feature close-out verification (P55 scope)
- The review report gate (F1, `needs-review → verifying`)
- Worktree cleanup enforcement (F3 provides health warnings; the verifier confirms cleanup)

## Functional Requirements

### Pillar A — Verifier Dispatch

**FR-401:** The `verifying → closed` gate in `CheckBugTransitionGate` (F1, FR-009) MUST dispatch a verifier sub-agent when a bug attempts to transition from `verifying` to `closed`.

**FR-402:** The dispatch MUST use the `verifier` role (defined by P55 Component 7) and the `verify-closeout` skill (defined by P55 Component 7).

**FR-403:** The verifier MUST receive:
- The bug ID
- The bug's slug
- The Definition of Done checklist (8 items, see FR-404)
- The current entity state (status, verification field, worktree path if any)

**FR-404:** The verifier MUST execute the following 8-item checklist. Each item has a concrete verification action and produces a pass/fail verdict with evidence.

| # | DoD Item | Verification Action |
|---|---|---|
| 1 | Fix verified | Read the bug entity. Confirm `verification` field is populated and non-empty (minimum 10 characters). |
| 2 | Changes committed | Run `git status --porcelain`. Confirm output is empty (no uncommitted changes). |
| 3 | Temp files removed | Run `git ls-files --others --exclude-standard`. Confirm no untracked files outside `work/` and `docs/`. |
| 4 | Tests pass | Run `go test ./...` on the worktree branch (or main if no worktree). Confirm exit code is 0. |
| 5 | Code reviewed | Call `doc(action: "list", owner: "<bug-id>", type: "report")`. Confirm at least one report document exists with status `approved` or `draft`. |
| 6 | Full lifecycle | Read the bug entity. Confirm current status is `verifying` and the bug reached it via `needs-review` (no skipped stages). |
| 7 | Landed on main | If a worktree exists, run `git merge-base --is-ancestor <branch> main`. Confirm exit code is 0. If no worktree (direct-to-main), confirm `git branch --contains HEAD` includes `main`. |
| 8 | Worktree cleaned up | Run `git worktree list`. Confirm no entry exists for this bug. Run `git branch | grep <bug-id>`. Confirm no output. |

**FR-405:** The verifier MUST return a structured JSON report:

```json
{
  "bug_id": "BUG-...",
  "checked_at": "<RFC 3339 timestamp>",
  "verdict": "pass" | "fail",
  "items": [
    {
      "dod_item": 1,
      "description": "Fix verified against expected behaviour",
      "verdict": "pass" | "fail",
      "evidence": "<output of verification action or failure reason>"
    },
    ...
  ]
}
```

**FR-406:** The verifier MUST run each check independently. It MUST NOT trust entity state claims — it re-runs commands (e.g., `go test ./...`, `git status --porcelain`) even when the entity state suggests they should pass.

**FR-407:** The verifier MUST NOT modify any state. It is read-only: it checks and reports.

**FR-408:** If the verifier cannot execute a verification action (e.g., `go` binary not found, git repository corrupted), it MUST report that item as `fail` with the error as evidence. It MUST NOT skip the item.

**Acceptance criteria:**
- The `verifying → closed` gate dispatches a verifier sub-agent
- The verifier receives the bug ID, slug, and 8-item checklist
- Each checklist item produces a pass/fail verdict with evidence
- The report is valid JSON matching the schema in FR-405
- A bug with all 8 items passing can transition to `closed`
- A bug with any failing item is blocked at `verifying` with the failure details

### Pillar B — Gate Integration

**FR-409:** The `verifying → closed` gate in `CheckBugTransitionGate` MUST:
- Spawn the verifier sub-agent (FR-401)
- Wait for the verifier to complete (synchronous dispatch with timeout)
- Parse the structured report
- If `verdict == "pass"`: allow the transition, return a satisfied `GateResult`
- If `verdict == "fail"`: block the transition, return an unsatisfied `GateResult` with the failure items listed in the reason

**FR-410:** The gate failure reason MUST list each failing DoD item by number and description. Example: `"close-out verification failed: DoD items 2 (changes committed), 4 (tests pass) — see verifier report for details"`.

**FR-411:** The verifier report MUST be registered as a document of type `report`, owned by the bug, at `work/reviews/verify-<bug-id>-<slug>.md`. The report content is the structured JSON from FR-405 formatted as a Markdown code block with a brief header.

**FR-412:** If the verifier sub-agent times out (does not complete within the dispatch timeout), the gate MUST fail with the reason `"verifier sub-agent timed out"`. The bug stays in `verifying`.

**Acceptance criteria:**
- A passing verifier report allows `verifying → closed`
- A failing verifier report blocks `verifying → closed` with itemised failures
- The verifier report is registered as a document owned by the bug
- A timeout produces a clear gate failure reason

### Pillar C — P55 Dependency Management

**FR-413:** Until P55 Component 7 is implemented (the `verifier` role and `verify-closeout` skill exist), the `verifying → closed` gate MUST be a pass-through placeholder as specified in F1 FR-009. The placeholder MUST log `"verifier not yet implemented — see F4 and P55 Component 7"` at INFO level.

**FR-414:** When P55 Component 7 is available (detected by the presence of the `verifier` role file at `.kbz/roles/verifier.yaml`), the gate MUST switch from placeholder to full dispatch automatically. No code change required — the gate checks for the role file at runtime.

**Acceptance criteria:**
- Before P55 Component 7 exists: `verifying → closed` succeeds as placeholder
- After P55 Component 7 exists: `verifying → closed` dispatches the verifier
- The transition between placeholder and full dispatch requires no code change

## Non-Functional Requirements

**NFR-401:** The verifier sub-agent MUST complete within 120 seconds (dispatch timeout). Typical bug fixes (1–3 files) should verify in under 30 seconds.

**NFR-402:** The verifier report document MUST be under 5KB (the JSON report plus Markdown wrapper).

**NFR-403:** The verifier MUST NOT leave side effects: no file writes, no state mutations, no git operations beyond read-only queries.

## Acceptance Criteria (Cross-Cutting)

**AC-401:** End-to-end: a bug with all DoD items satisfied transitions `verifying → closed` via the verifier. The verifier report is registered as a document.

**AC-402:** End-to-end: a bug with failing tests (DoD item 4) is blocked at `verifying`. The gate failure reason lists item 4. The bug stays in `verifying`.

**AC-403:** End-to-end: a bug with uncommitted changes (DoD item 2) and an orphaned worktree (DoD item 8) is blocked with both items listed in the failure reason.

**AC-404:** Placeholder mode: before P55 Component 7 exists, `verifying → closed` succeeds with a log message.

## Dependencies and Assumptions

**Dependencies:**
- **P55 Component 7** — Defines the `verifier` role (`.kbz/roles/verifier.yaml`) and `verify-closeout` skill (`.kbz/skills/verify-closeout/SKILL.md`). This spec cannot be fully implemented until P55 delivers these artefacts.
- **F1 (Bug Lifecycle Gate Enforcement)** — The `verifying → closed` gate in `CheckBugTransitionGate` is the integration point. FR-009 in F1 references the verifier dispatch defined here.
- **`spawn_agent` tool** — Used to dispatch the verifier sub-agent. Must support role-based dispatch with the `verifier` role.

**Assumptions:**
1. P55 Component 7 will produce a `verifier.yaml` role file and `verify-closeout/SKILL.md` skill file at predictable paths.
2. The `spawn_agent` tool can dispatch sub-agents with a specified role and skill.
3. The verifier sub-agent has access to git commands and the `go` binary in its execution environment.
4. The structured JSON report format in FR-405 is parseable by the gate code.
