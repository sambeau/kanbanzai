# Dev-Plan: P50 Review Remediation

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-05                     |
| Status | draft |
| Author | architect                      |

## Overview

This dev-plan converts the reviewed P50 batch conformance review into an implementation plan for getting P50 to `done` without carrying the review findings forward.

Source review:
`work/P50-retro-may-2026/P50-report-batch-conformance-review.md`
(DOC-`P50-retro-may-2026/report-p50-report-batch-conformance-review`).

The review verdict is `fail`: F1 is approved, while F2, F3, and F4 require remediation. F5 (Document Path Tool) is still in `developing` and must either be completed and reviewed or explicitly deferred before P50 can close cleanly.

## Scope

This remediation plan covers fixes required by the approved P50 feature specifications and the P50 batch conformance review:

- `work/P50-retro-may-2026/P50-spec-decompose-proposal-quality.md`
- `work/P50-retro-may-2026/P50-spec-commit-discipline-prompts.md`
- `work/P50-retro-may-2026/P50-spec-merge-discipline.md`
- `work/P50-retro-may-2026/P50-spec-doc-path-tool.md` for the unresolved F5 completion decision
- `work/P50-retro-may-2026/P50-report-batch-conformance-review.md`

In scope:

- Fix every blocking finding BF-1 through BF-10 from the batch conformance review.
- Complete or explicitly defer F5 so the P50 scope is unambiguous.
- Add enough targeted tests to prove each blocking finding is resolved.
- Re-run feature review for F2, F3, F4, and F5 if completed.
- Merge only after every reviewed feature has a registered approval report and no active blocking findings.

Out of scope:

- Non-blocking findings NBF-1 through NBF-21, except where they are required to make a blocking fix safe.
- Broad refactors of MCP response serialization, git utilities, or decompose grouping heuristics.
- Fixing pre-existing unrelated repository test failures.

## Task Breakdown

### R1: Normalize P50 closure scope and workflow state

- **Deliverable:** Confirm the final P50 feature list and record the closure decision for F5 Document Path Tool: either complete/review it as part of P50 or explicitly defer it to a future batch with entity status and documentation updated consistently.
- **Findings addressed:** Feature Census gap (F5 still `developing`)
- **Depends on:** nothing
- **Effort:** 1
- **Parallelisable:** yes
- **Verification:** `status(id: "FEAT-01KQTNYN00HZA")` shows either `reviewing` with terminal tasks and approved documents, or a documented terminal/deferred state.

### R2: Implement F2 paired-task opt-out

- **Deliverable:** Add optional `paired_test_tasks` input support to `decompose propose`, defaulting to `true`. When explicitly false, proposal generation preserves one-task-per-AC behaviour.
- **Findings addressed:** BF-1
- **Depends on:** nothing
- **Effort:** 2
- **Parallelisable:** yes
- **Verification:** Unit test proves default 3 ACs → 6 tasks and `paired_test_tasks: false` → 3 tasks.

### R3: Complete F2 quality test task and close F2 re-review gap

- **Deliverable:** Complete TASK-01KQTX5DX2C1T and ensure all F2 tests from the original dev-plan are present: refuse-to-propose, paired output, depends_on correctness, opt-out flag, complete-task dependency graph, and test-only AC detection.
- **Findings addressed:** BF-1, F2 queued-task prerequisite gap
- **Depends on:** R2
- **Effort:** 2
- **Parallelisable:** no
- **Verification:** `status(id: "FEAT-01KQTNYN00M4P")` shows all tasks terminal; targeted decompose tests pass.

### R4: Wire F3 `state_modified` into production mutation handlers

- **Deliverable:** Call `SignalStateModified` in the actual state-mutating code paths for high-mutation tools: `entity` create/update/transition, `doc` register/approve/supersede and any implemented mutation actions that write state, `knowledge` contribute/confirm/retire, and `finish`.
- **Findings addressed:** BF-2
- **Depends on:** nothing
- **Effort:** 2
- **Parallelisable:** yes
- **Verification:** Grep shows production call sites outside tests; handler-level tests assert `state_modified: true` in real responses for each high-mutation tool family and absent/false for read-only tools.

### R5: Repair F3 skill dual-write mirrors

- **Deliverable:** Mirror the `state_modified` rule from `.agents/skills/kanbanzai-agents/SKILL.md` into `internal/kbzinit/skills/agents/SKILL.md`, and mirror the strengthened getting-started git status checklist into `internal/kbzinit/skills/getting-started/SKILL.md`.
- **Findings addressed:** BF-3, BF-4
- **Depends on:** nothing
- **Effort:** 1
- **Parallelisable:** yes
- **Verification:** Diff inspection confirms `.agents/skills/kanbanzai-*` changes and `internal/kbzinit/skills/*` mirrors match for the changed sections.

### R6: Enforce F4 lifecycle transitions in production code

- **Deliverable:** Wire `FeatureValidTransitions` / `IsValidFeatureTransition` into the entity transition validation path. Add the missing `needs-rework → developing` transition so rework loops remain possible.
- **Findings addressed:** BF-5, BF-9
- **Depends on:** nothing
- **Effort:** 2
- **Parallelisable:** yes
- **Verification:** Tests prove invalid skips are rejected, valid review/rework transitions are accepted, and `needs-rework → developing` works.

### R7: Align `executeMerge` with the `merging` stage

- **Deliverable:** Change `executeMerge` so a successful merge does not auto-transition directly to `done`. It should leave the feature in or advance it to `merging` as appropriate, then require the verifying step before `done`.
- **Findings addressed:** BF-6
- **Depends on:** R6
- **Effort:** 2
- **Parallelisable:** no
- **Verification:** Integration test calls `executeMerge` and asserts it does not bypass `merging`/`verifying`.

### R8: Restore F4 worktree ancestry verification

- **Deliverable:** Ensure the main implementation verifies `git merge-base --is-ancestor <branch> <main>` before marking a worktree `merged`. If the branch is not an ancestor, keep the worktree active and keep the feature out of `done`.
- **Findings addressed:** BF-7
- **Depends on:** R7
- **Effort:** 1
- **Parallelisable:** no
- **Verification:** Tests cover branch-is-ancestor and branch-not-ancestor outcomes.

### R9: Implement or complete F4 verifying-stage build/test gate

- **Deliverable:** Add the verifying-stage execution path that runs `go build ./...` and `go test ./...`, transitions to `done` only on success, and transitions to `needs-rework` with captured output on failure. Pre-existing unrelated failures must be handled via the existing waiver/known-failure mechanism rather than silently ignored.
- **Findings addressed:** BF-6, BF-10
- **Depends on:** R6, R7, R8
- **Effort:** 3
- **Parallelisable:** no
- **Verification:** Tests cover build failure, test failure, and success paths. The test that calls `executeMerge` crosses the production boundary into verifying behaviour.

### R10: Add F4 merge prompt to `kanbanzai-agents` and mirror it

- **Deliverable:** Add the same five-step post-review merge prompt from `orchestrate-review` into `.agents/skills/kanbanzai-agents/SKILL.md`, and mirror it into `internal/kbzinit/skills/agents/SKILL.md` in the same change.
- **Findings addressed:** BF-8
- **Depends on:** R6
- **Effort:** 1
- **Parallelisable:** yes after R6 defines the final lifecycle vocabulary
- **Verification:** Inspection confirms both files contain the merge prompt and use identical stage vocabulary.

### R11: Run P50 remediation verification and re-review

- **Deliverable:** Run targeted tests for F2/F3/F4, rerun the relevant review checks, and register re-review reports showing every blocking finding resolved.
- **Findings addressed:** BF-1 through BF-10
- **Depends on:** R1, R3, R4, R5, R9, R10
- **Effort:** 2
- **Parallelisable:** no
- **Verification:** F2, F3, F4 (and F5 if included) have approved review reports with no blocking findings.

### R12: Execute final merge and P50 closure checklist

- **Deliverable:** Merge approved feature branches, verify worktree ancestry, run final build/test verification or documented waived baseline, and transition P50/B49 scope to done only after all included features are terminal and reviewed.
- **Findings addressed:** aggregate completion gap
- **Depends on:** R11
- **Effort:** 1
- **Parallelisable:** no
- **Verification:** `status(id: "P50-retro-may-2026")` shows no incomplete P50 features, no orphaned reviewing warnings for included features, and no active worktrees for merged features.

## Dependency Graph

```text
R1 ───────────────────────────────────────────────┐
                                                   │
R2 ── R3 ─────────────────────────────────────────┤
                                                   │
R4 ───────────────────────────────────────────────┤
                                                   │
R5 ───────────────────────────────────────────────┤
                                                   ├── R11 ── R12
R6 ── R7 ── R8 ── R9 ─────────────────────────────┤
│                                                  │
└── R10 ──────────────────────────────────────────┘
```

Parallelisable groups:

- **Group A:** R1, R2, R4, R5, R6 can start independently.
- **Group B:** R3 follows R2; R7 follows R6; R10 follows R6.
- **Critical path:** R6 → R7 → R8 → R9 → R11 → R12.

## Interface Contracts

- **`decompose propose` input:** Optional `paired_test_tasks` boolean. Omitted means `true` to preserve the new default paired-task behaviour. Explicit `false` restores one-task-per-AC output.
- **`state_modified` response field:** Top-level additive boolean. Mutation handlers must call `SignalStateModified`; read-only handlers must not.
- **Feature lifecycle validation:** All feature status changes must pass the central feature transition validation. Valid remediation loop includes `needs-rework → developing`.
- **Merge execution:** `merge(action: "execute")` must not mark a feature `done` directly. Done requires verified merge ancestry and successful verifying-stage build/test outcome.
- **Dual-write documents:** Any change under `.agents/skills/kanbanzai-*` must be mirrored under `internal/kbzinit/skills/*` in the same logical change.

## Risk Assessment

### Risk: Dirty working tree hides remediation scope

- **Likelihood:** high
- **Impact:** high
- **Mitigation:** Before each remediation task, run `git status --short` and isolate commits by feature/finding. Do not mix review-report, dev-plan, and implementation changes in the same commit unless they are the same logical unit.

### Risk: F4 lifecycle changes conflict with existing transition assumptions

- **Likelihood:** medium
- **Impact:** high
- **Mitigation:** Wire the central transition validator before changing merge behaviour, then update tests that rely on old direct-to-done paths. Keep the rework loop explicit with `needs-rework → developing`.

### Risk: Existing repository test failures obscure remediation verification

- **Likelihood:** high
- **Impact:** medium
- **Mitigation:** Run targeted package tests for changed areas first. For `go test ./...`, record unrelated known failures separately; do not treat new P50 failures as waived.

### Risk: Dual-write drift recurs

- **Likelihood:** medium
- **Impact:** medium
- **Mitigation:** Add review checklist items for every skill-file task: source skill, kbzinit mirror, and diff inspection in the same commit.

### Risk: F5 remains ambiguous and blocks P50 closure

- **Likelihood:** medium
- **Impact:** high
- **Mitigation:** Resolve F5 first via R1. P50 cannot close cleanly while an in-scope feature remains `developing` without an explicit deferral decision.

## Verification Approach

| Verification | Method | Produced by |
|--------------|--------|-------------|
| Dev-plan structure | Run `.kbz/skills/write-dev-plan/scripts/validate-dev-plan-structure.sh work/P50-retro-may-2026/P50-dev-plan-review-remediation.md` | This plan |
| F2 opt-out | Unit tests for default paired output and `paired_test_tasks: false` | R2, R3 |
| F2 task terminality | `status(id: "FEAT-01KQTNYN00M4P")` shows no queued tasks | R3 |
| F3 state_modified | Handler-level tests assert true for mutations and absent/false for reads | R4 |
| F3 dual-write | Diff inspection of `.agents/skills/` vs `internal/kbzinit/skills/` changed sections | R5 |
| F4 lifecycle enforcement | Tests for valid/invalid transitions and rework loop | R6 |
| F4 merge behaviour | Integration test calls `executeMerge` and confirms no direct-to-done bypass | R7, R9 |
| F4 worktree ancestry | Tests for ancestor and non-ancestor worktree states | R8 |
| F4 verifying gate | Tests for build failure, test failure, and success-to-done | R9 |
| P50 completion | Re-review reports approved; `status` shows included features terminal and no orphaned review warnings | R11, R12 |

## Traceability Matrix

| Review Finding / Requirement | Remediation Task(s) |
|------------------------------|---------------------|
| F5 still developing / closure ambiguity | R1 |
| BF-1: F2 missing `paired=false` opt-out | R2, R3 |
| BF-2: F3 `SignalStateModified` never called | R4 |
| BF-3: F3 `kanbanzai-agents` mirror missing | R5 |
| BF-4: F3 getting-started mirror missing | R5 |
| BF-5: F4 transition state machine unenforced | R6 |
| BF-6: F4 `executeMerge` bypasses stages | R7, R9 |
| BF-7: F4 worktree ancestry verification missing on main | R8 |
| BF-8: F4 agents skill missing merge prompt | R10 |
| BF-9: F4 missing `needs-rework → developing` | R6 |
| BF-10: F4 `executeMerge` auto-advance untested | R7, R9 |
| Final P50 clean closure | R11, R12 |
