# Batch Conformance Review: B38-plans-and-batches

## Scope
- **Batch:** B38-plans-and-batches (Plans and Batches)
- **Features:** 10 total (6 done, 3 reviewing, 1 superseded)
- **Review date:** 2026-04-30 (revised 13:25 BST)
- **Reviewer:** reviewer-conformance
- **Review cycle:** 2 (re-verification of prior draft findings)

## Feature Census

| # | Feature | Display ID | Status | Spec | Dev-Plan | Tasks | Notes |
|---|---------|------------|--------|------|----------|-------|-------|
| 1 | FEAT-01KQ7YQKBDNAP | B38-F2 | done | approved | — | 6/6 done | Plan Entity Data Model |
| 2 | FEAT-01KQ7YQKEEHFY | B38-F3 | done | approved | — | 5/5 done | Batch Entity Rename |
| 3 | FEAT-01KQ7YQKHK2GV | B38-F4 | done | approved | — | 3/3 done | Feature Display IDs |
| 4 | FEAT-01KQ7YQKMCM6T | B38-F5 | done | approved | — | 5/5 done | Recursive Progress Rollup |
| 5 | FEAT-01KQ7YQKPT8HF | B38-F6 | done | approved | — | 7/7 done | MCP Tools & Dashboard |
| 6 | FEAT-01KQ7YQKWTBRP | B38-F7 | done | approved | — | 7/7 done | Documentation & Skills |
| 7 | FEAT-01KQ7JDT511BZ | B38-F9 | **reviewing** | approved | approved | 5/5 done | Work Tree Migration — branch **merged to main** ✓ |
| 8 | FEAT-01KQ7YQK6DDDA | B38-F1 | **reviewing** | approved | — | 4/4 done | Config Schema — **merge conflicts with main** |
| 9 | FEAT-01KQ7YQKT04M7 | B38-F8 | **reviewing** | **draft** | — | **0 tasks** | State File Migration — spec unapproved, no implementation |
| 10 | FEAT-01KQAGMVQABXH | B38-F10 | superseded | approved | approved | — | Full Combined Migration — superseded by F8+F9 |

## Conformance Gaps

| # | Feature | Type | Description | Severity | Status |
|---|---------|------|-------------|----------|--------|
| CG-1 | FEAT-01KQ7YQKT04M7 (B38-F8) | spec-status | Spec `spec-b38-spec-p38-f8-combined-migration` is **draft**, not approved. Feature is in `reviewing` but never passed the spec-approval gate. | **blocking** | unresolved |
| CG-2 | FEAT-01KQ7YQKT04M7 (B38-F8) | missing-tasks | Feature has **0 tasks**. `reviewing` prerequisite requires `tasks.all_terminal: true`. With no tasks there is nothing to verify as complete. | **blocking** | unresolved |
| CG-3 | FEAT-01KQ7YQKT04M7 (B38-F8) | lifecycle-integrity | Feature was overridden from `designing → developing`, `specifying → dev-planning`, and `dev-planning → developing` (twice). Spec written retrospectively and never approved. Four gate overrides with zero tasks indicates no implementation was delivered. | **blocking** | unresolved |
| CG-4 | FEAT-01KQ7YQK6DDDA (B38-F1) | branch-conflict | Worktree branch `feature/FEAT-01KQ7YQK6DDDA-config-schema-project-singleton` has **merge conflicts with main**. Cannot be merged until resolved. | **blocking** | unresolved |
| CG-5 | FEAT-01KQ7JDT511BZ (B38-F9) | stale-worktree | Two active worktrees exist: `WT-01KQEX083D5Y4` (branch already merged) and `WT-01KQEX7T3X5XD`. Neither has been cleaned up. | non-blocking | unresolved |
| CG-6 | B38-plans-and-batches | prior-review-draft | Prior internal review `P38-plans-and-batches/report-p38-report-review-f2-f7` (covering F2–F7) remains in draft status. | non-blocking | unresolved |

> **Change since cycle 1:** CG-6 from the prior draft (B38-F9 branch 23 commits behind main) is **resolved** — `feature/FEAT-01KQ7JDT511BZ-work-tree-migration` has been merged to main (confirmed via health check warning `worktree_branch_merged`). The `kbz migrate` command is now on main. B38-F9 feature status still needs to advance `reviewing → done`.

## Documentation Currency

- **AGENTS.md:** Stale tool references (`sh`, `graph_project`, `main`) — pre-existing, not introduced by B38.
- **Scope guard:** B38 not listed in AGENTS.md scope guard. 27+ prior batches also absent — pre-existing systemic gap.
- **Workflow skills:** Multiple stale tool references across `.agents/skills/` — pre-existing.
- **Knowledge entries:** 1 retrospective signal recorded for B38 period (tool friction writing Go files from worktree sub-agents). No formal knowledge entries contributed under B38 scope.

## Retrospective Summary

One workflow signal was captured from the B38 implementation period: writing Go source files from worktree sub-agents is significantly complicated by shell/Python/Go triple-escaping; the `write_file` MCP tool should be the primary recommendation in `implement-task SKILL.md`. No other signals were recorded. For a batch spanning 10 features and a foundational entity model change, the low signal count suggests retrospective contribution was not consistently observed during feature completion.

## Batch Verdict

**fail** — 4 blocking conformance gaps (CG-1 through CG-4) prevent batch closure.

### Summary of blocking issues

1. **B38-F8 (State File Migration):** Spec is draft; 0 tasks; no implementation evidence. The feature appears to have been staged into `reviewing` via lifecycle overrides without any actual work being done. Resolution options: (a) supersede this feature — document that state-file migration was subsumed by B38-F9 (Work Tree Migration), which already handles the migration — or (b) approve the spec, create tasks, implement, and review.

2. **B38-F1 (Config Schema):** Branch has merge conflicts with main. All 4 tasks are done and spec is approved; the only blocking issue is the unresolved merge conflict. Resolution: rebase the branch onto main, resolve conflicts, re-verify tests pass.

### Resolution path to `done`

1. **B38-F8:** Supersede with documented rationale that state-file migration work was completed as part of B38-F9, OR approve spec, create tasks, implement, and review.
2. **B38-F1:** Rebase `feature/FEAT-01KQ7YQK6DDDA-config-schema-project-singleton` onto main, resolve conflicts, verify tests, advance to `done`.
3. **B38-F9:** Advance feature lifecycle `reviewing → done` (branch is already on main). Clean up the two stale worktrees.
4. **Batch:** Once all 3 features reach a terminal state, re-run this batch review. The 6 `done` features and the 1 `superseded` feature are conformance-clean.

## Evidence

- Batch entity: `entity(action: "get", id: "B38-plans-and-batches")`
- Feature list: `entity(action: "list", type: "feature", parent: "B38-plans-and-batches")` → 10 features
- Spec statuses: F9 approved, F1 approved, F8 draft
- Task verification: F9 5/5 done; F1 4/4 done; F8 0 tasks
- Health check: merge conflict on FEAT-01KQ7YQK6DDDA (error); branch-merged warning on FEAT-01KQ7JDT511BZ; 2 stale worktrees; 9 TTL-expired knowledge entries
- Retro synthesis (project scope, since 2026-04-27): 1 signal (tool-friction, worktree file writing)
- Worktrees: `worktree(action: "list", status: "active")` → 4 active (F1, F9 ×2, F10-superseded)
