| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-06T20:47:43+01:00      |
| Status | Draft                          |
| Author | sambeau                        |

## Scope

This plan implements the requirements defined in
`work/P52-fast-track-orchestration/P52-spec-fast-track-profile.md`
(P52-fast-track-orchestration/spec-p52-spec-fast-track-profile).

It covers the addition of the Fast-Track Profile section to the `orchestrate-development`
skill file. This is a documentation-only change — no Go code or MCP tool changes.

The plan covers one task: editing both the local skill file and the canonical source copy to
add the new section. It does NOT cover P44 integration (dispatch_task), P53 (dirty-work
attribution), or the `finish` summary limit documentation — these are deferred to their
respective plans.

## Task Breakdown

### Task 1: Add Fast-Track Profile section to orchestrate-development SKILL.md

- **Description:** Add the `## Fast-Track Profile` section to `.kbz/skills/orchestrate-development/SKILL.md`
  after the `## Procedure` section. The section must contain: the No-Stop Contract preamble,
  Phase 0 (Session Start Audit) with 5 sub-steps, Phase 1 (Dispatch) with 4 sub-steps,
  Phase 2 (Close-Out) with 4 sub-steps, the Rules subsection with 5 rules, and the
  fast-track-specific anti-patterns (Implicit Gate, Ghost Work Discovery, State Ambiguity
  Drift, Milestone Pause). Also includes P51/P44 integration notes.
- **Deliverable:** Updated `.kbz/skills/orchestrate-development/SKILL.md` with the new section.
- **Depends on:** None
- **Effort:** Medium
- **Spec requirement:** REQ-001 through REQ-009, REQ-NF-001

### Task 2: Mirror change to canonical source file

- **Description:** Apply the identical change to `internal/kbzinit/skills/task-execution/orchestrate-development/SKILL.md`
  so that generated installations receive the same fast-track profile.
- **Deliverable:** Updated `internal/kbzinit/skills/task-execution/orchestrate-development/SKILL.md`.
- **Depends on:** Task 1 (must have the exact content to mirror)
- **Effort:** Small
- **Spec requirement:** Constraints section (canonical source must stay in sync)

## Dependency Graph

  Task 1 (no dependencies)
  Task 2 → depends on Task 1

  Parallel groups: [Task 1], [Task 2]
  Critical path: Task 1 → Task 2

## Risk Assessment

### Risk: Incorrect section placement

- **Probability:** Low
- **Impact:** Medium — wrong placement would require a follow-up edit to move the section.
- **Mitigation:** Verify section placement against the output format spec (after Procedure,
  before Output Format) before committing.
- **Affected tasks:** Task 1

### Risk: Content drift between local and canonical copies

- **Probability:** Low
- **Impact:** Medium — if the copies diverge, consumer installations get a different skill
  than the local development copy.
- **Mitigation:** Apply the edit to the canonical file immediately after Task 1, using
  the same content. Verify with diff.
- **Affected tasks:** Task 2

### Risk: No-stop contract is not strong enough to prevent implicit gates

- **Probability:** Medium
- **Impact:** High — the root problem the design aims to solve would persist.
- **Mitigation:** The design identifies this risk and adds tracking: orchestrator
  self-reports implicit gates during close-out. If gates persist after 5 fast-track runs,
  the anti-patterns and contract language can be strengthened.
- **Affected tasks:** Task 1 (contract language)

## Verification Approach

| Acceptance Criterion | Verification Method | Produced By |
|---------------------|-------------------|-------------|
| AC-001 — Section heading exists | Inspection | Task 1 |
| AC-002 — No-stop contract preamble | Inspection | Task 1 |
| AC-003 — Phase 0 with 5 sub-steps | Inspection | Task 1 |
| AC-004 — Phase 1 with 4 sub-steps | Inspection | Task 1 |
| AC-005 — Phase 2 with 4 sub-steps | Inspection | Task 1 |
| AC-006 — Rules subsection with 5 rules | Inspection | Task 1 |
| AC-007 — 4 anti-patterns with Detect/BECAUSE/Resolve | Inspection | Task 1 |
| AC-008 — P51/P44 integration notes | Inspection | Task 1 |
| AC-009 — Existing content unchanged | Inspection (diff) | Tasks 1, 2 |
| Canonical source matches local | Inspection (diff) | Task 2 |
