| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-06T20:47:43+01:00      |
| Status | Draft                          |
| Author | sambeau                        |

## Problem Statement

This specification implements the design described in
`work/P52-fast-track-orchestration/P52-design-fast-track-orchestration.md` (PROJECT/design-p52-design-fast-track-orchestration).

The design defines a **fast-track behavioral profile** for the `orchestrate-development`
skill — a lightweight alternative procedure branch that activates when the feature's tier
indicates fast-track (e.g. `retro_fix`). The profile replaces the full 6-phase procedure
with a 3-phase flow: Session Start Audit → Dispatch → Close-Out, governed by an explicit
no-stop contract and fast-track-specific anti-patterns.

**Scope:** This specification covers the fast-track profile section to be added to the
`orchestrate-development` SKILL.md file. It is purely a documentation change — no code
changes to the MCP server or underlying tools.

**Out of scope:** Integration with P44's `dispatch_task` mechanism, automated token-count
triggers (P44 Phase 3b), dirty-work attribution (P53), and the `finish` summary limit
documentation. These are deferred to their respective plans.

## Requirements

### Functional Requirements

- **REQ-001:** The `orchestrate-development` SKILL.md must gain a `## Fast-Track Profile`
  section that activates when the feature's tier is `retro_fix` or the batch is explicitly
  marked fast-track.

- **REQ-002:** The Fast-Track Profile section must contain a preamble declaring the
  **No-Stop Contract**: the orchestrator will NOT stop for confirmation at any point.
  The only valid stop conditions are: all work done, build failure requiring code changes
  beyond the orchestrator's scope, missing dependency blocking all remaining tasks, or
  spec ambiguity that cannot be resolved without human input.

- **REQ-003:** The Fast-Track Profile must define a **Phase 0: Session Start Audit** that
  replaces the full procedure's Phase 0 (Cohort Setup). The audit must:
  (a) Call `status()` to build authoritative entity state — identifying terminal, in-flight,
      ready, and stuck tasks.
  (b) Cross-reference task state against code existence (ghost-work detection).
  (c) Check review-readiness drift before dispatch — surface blocker if a feature in
      `reviewing` via override still has non-terminal tasks.
  (d) Check dirty working-tree state before implementation, classifying changes as
      current-scope, prior work, or workflow/index metadata.
  (e) Identify ghost work: for each ready task, check whether the described change
      already exists. If yes, mark `not-planned` — do not claim or implement.

- **REQ-004:** The Fast-Track Profile must define a **Phase 1: Dispatch** that replaces
  the full procedure's Phase 1–2 ceremony. It must:
  (a) From the session-start audit, identify the ready frontier — all tasks in `ready`
      status with satisfied dependencies.
  (b) Dispatch all ready frontier tasks in parallel using
      `handoff(task_id: "TASK-xxx", role: "implementer-go")`.
  (c) Do NOT stop after dispatching. If nothing to dispatch because tasks are still active,
      poll `status()` once per minute until something completes.
  (d) When a task completes, immediately dispatch any newly-unblocked tasks.

- **REQ-005:** The Fast-Track Profile must define a **Phase 2: Close-Out** that replaces
  the full procedure's Phase 4–6. It must:
  (a) When all tasks are terminal, verify each feature can advance.
  (b) Transition features through to `done` or `reviewing` as appropriate.
  (c) Report completion: list features and their final status.
  (d) Include a procedural compaction trigger: if at ~60%+ estimated context utilisation
      and work remains for another feature, produce a compaction artefact using the
      U-shaped template and instruct the human to start a fresh session.

- **REQ-006:** The Fast-Track Profile must include a **Rules** section with these rules:
  - NO stops at batch boundaries, milestone completions, or wave dispatches
  - NO status tables until all work is done
  - NO ghost work — audit before claiming
  - NO user summaries — trust `status()` output only
  - ALWAYS use `handoff(task_id, role: "implementer-go")` for sub-agent dispatch

- **REQ-007:** The Fast-Track Profile must include fast-track-specific anti-patterns:
  **Implicit Gate**, **Ghost Work Discovery**, **State Ambiguity Drift**, and
  **Milestone Pause** — each with detection criteria, rationale, and resolution steps.

- **REQ-008:** The Fast-Track Profile must include integration notes documenting:
  (a) That P51's handoff pipeline changes are not required but the dispatch phase
      explicitly uses `role: "implementer-go"` regardless of P51's state.
  (b) That P44's `dispatch_task` will eventually replace the `handoff` + `spawn_agent`
      dispatch mechanism in Phase 1, but the behavioral rules remain unchanged.
  (c) That the procedural compaction trigger (Phase 2) is a stopgap; P44 Phase 3b
      will replace manual estimation with automated token-count-based triggers.

- **REQ-009:** The Fast-Track Profile section must be additive — it must NOT modify the
  existing 6-phase procedure or anti-patterns of the `orchestrate-development` skill.
  The profile is an alternate branch activated by feature tier, not a rewrite.

### Non-Functional Requirements

- **REQ-NF-001:** The Fast-Track Profile section must be placed after the Procedure
  section and before the Output Format section in the SKILL.md file, to maintain the
  document's logical flow (vocabulary → anti-patterns → checklist → procedure →
  fast-track profile → output format → examples).

## Constraints

- This is a documentation-only change to the `.kbz/skills/orchestrate-development/SKILL.md`
  file. No Go code, no MCP tool changes, no infrastructure changes.
- The corresponding canonical source file at
  `internal/kbzinit/skills/task-execution/orchestrate-development/SKILL.md` must receive
  the identical change to keep the generated installation in sync.
- The section must be additive — the existing 6-phase procedure, anti-patterns, checklist,
  vocabulary, and examples must remain unchanged.
- The profile must reference the `orchestrate-development` vocabulary and anti-patterns
  section by section ID, avoiding duplication of content.
- The profile must not introduce any dependencies on tools or features not available in
  the current server version.
- The no-stop contract preamble must appear at the top of the Fast-Track Profile section
  before any phase descriptions, to ensure it is read before any procedural step.

## Acceptance Criteria

- **AC-001 (REQ-001):** Given the `orchestrate-development/SKILL.md` file, when the file
  is read, then it contains a `## Fast-Track Profile` section immediately after the
  `## Procedure` section.

- **AC-002 (REQ-002):** Given the Fast-Track Profile section, when read from top to
  bottom, then the first substantive content after the heading is the No-Stop Contract
  preamble, containing: "You are in fast-track mode. You will NOT stop for confirmation
  at any point." and the four valid stop conditions.

- **AC-003 (REQ-003):** Given the Fast-Track Profile's Phase 0, when read, then all
  five sub-steps (a–e) of the Session Start Audit are present and described: status()
  call, ghost-work cross-reference, review-readiness drift check, dirty working-tree
  classification, and per-task ghost-work detection before claiming.

- **AC-004 (REQ-004):** Given the Fast-Track Profile's Phase 1, when read, then all
  four sub-steps (a–d) of the Dispatch phase are present: ready frontier identification,
  parallel handoff dispatch, non-stop-after-dispatch rule, and immediate dispatch of
  newly-unblocked tasks.

- **AC-005 (REQ-005):** Given the Fast-Track Profile's Phase 2, when read, then all
  four sub-steps (a–d) of the Close-Out phase are present: terminality verification,
  feature transition, completion reporting, and procedural compaction trigger.

- **AC-006 (REQ-006):** Given the Rules subsection of the Fast-Track Profile, when read,
  then all five rules are present: no stops at batch boundaries, no status tables until
  done, no ghost work, no user summaries, and always use handoff with implementer-go role.

- **AC-007 (REQ-007):** Given the anti-patterns section of the Fast-Track Profile, when
  read, then all four fast-track-specific anti-patterns are present (Implicit Gate, Ghost
  Work Discovery, State Ambiguity Drift, Milestone Pause), each with a Detect, BECAUSE,
  and Resolve clause.

- **AC-008 (REQ-008):** Given the Fast-Track Profile section, when read, then integration
  notes for P51 (role override regardless of P51 state), P44 dispatch_task future
  replacement, and P44 Phase 3b automated triggers are present in text.

- **AC-009 (REQ-009):** Given the existing sections of the `orchestrate-development`
  SKILL.md file, when the Fast-Track Profile section is added, then all pre-existing
  content (vocabulary, anti-patterns, checklist, procedure, output format, examples)
  remains unchanged — only the new section is added.

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Inspection | Read SKILL.md and verify `## Fast-Track Profile` heading exists after `## Procedure` section |
| AC-002 | Inspection | Verify no-stop contract preamble is the first content after the heading |
| AC-003 | Inspection | Verify all 5 session-start audit sub-steps are documented |
| AC-004 | Inspection | Verify all 4 dispatch sub-steps are documented |
| AC-005 | Inspection | Verify all 4 close-out sub-steps are documented |
| AC-006 | Inspection | Verify all 5 rules are listed in the Rules subsection |
| AC-007 | Inspection | Verify 4 anti-patterns with Detect/BECAUSE/Resolve clauses |
| AC-008 | Inspection | Verify P51, P44 dispatch_task, and P44 Phase 3b integration notes |
| AC-009 | Inspection | Diff the file — verify only the new section was added, no existing content modified |
