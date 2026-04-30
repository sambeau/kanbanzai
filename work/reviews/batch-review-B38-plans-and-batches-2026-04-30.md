# Batch Conformance Review: B38-plans-and-batches

## Scope
- **Batch:** B38-plans-and-batches (Plans and Batches)
- **Features:** 10 total (6 done, 3 reviewing, 1 superseded)
- **Review date:** 2026-04-30
- **Reviewer:** reviewer-conformance

## Feature Census

| # | Feature | Display ID | Status | Spec | Dev-Plan | Tasks | Notes |
|---|---------|------------|--------|------|----------|-------|-------|
| 1 | FEAT-01KQ7YQKBDNAP | B38-F2 | done | approved | — | 6/6 done | Plan Entity Data Model |
| 2 | FEAT-01KQ7YQKEEHFY | B38-F3 | done | approved | — | 5/5 done | Batch Entity Rename |
| 3 | FEAT-01KQ7YQKHK2GV | B38-F4 | done | approved | — | 3/3 done | Feature Display IDs |
| 4 | FEAT-01KQ7YQKMCM6T | B38-F5 | done | approved | — | 5/5 done | Recursive Progress Rollup |
| 5 | FEAT-01KQ7YQKPT8HF | B38-F6 | done | approved | — | 7/7 done | MCP Tools & Dashboard |
| 6 | FEAT-01KQ7YQKWTBRP | B38-F7 | done | approved | — | 7/7 done | Documentation & Skills |
| 7 | FEAT-01KQ7JDT511BZ | B38-F9 | **reviewing** | approved | approved | 5/5 done | Work Tree Migration |
| 8 | FEAT-01KQ7YQK6DDDA | B38-F1 | **reviewing** | approved | — | 4/4 done | Config Schema — **merge conflicts** |
| 9 | FEAT-01KQ7YQKT04M7 | B38-F8 | **reviewing** | **draft** | — | **0 tasks** | State File Migration — **no implementation** |
| 10 | FEAT-01KQAGMVQABXH | B38-F10 | superseded | approved | approved | — | Full Combined Migration — superseded by F7+F8+F9 |

## Conformance Gaps

| # | Feature | Type | Description | Severity |
|---|---------|------|-------------|----------|
| CG-1 | FEAT-01KQ7YQKT04M7 (B38-F8) | spec-status | Spec `spec-b38-spec-p38-f8-combined-migration` is **draft**, not approved. The feature is in `reviewing` status but has never passed the spec-approval gate. | **blocking** |
| CG-2 | FEAT-01KQ7YQKT04M7 (B38-F8) | missing-tasks | Feature has **0 tasks**. It is in `reviewing` status, but the `reviewing` stage prerequisite requires `tasks.all_terminal: true`. With no tasks, there is nothing to verify as complete. | **blocking** |
| CG-3 | FEAT-01KQ7YQKT04M7 (B38-F8) | lifecycle-override | Feature was overridden from `designing` straight to `developing`, then from `specifying` to `dev-planning`, then from `dev-planning` to `developing` (twice). The spec was written retrospectively and never approved. Four gate overrides for a feature with zero tasks indicates a process gap. | **blocking** |
| CG-4 | FEAT-01KQ7YQK6DDDA (B38-F1) | branch-conflict | Worktree branch `feature/FEAT-01KQ7YQK6DDDA-config-schema-project-singleton` has **merge conflicts with main**. The branch is also 8 commits behind main. It cannot be merged until conflicts are resolved. | **blocking** |
| CG-5 | FEAT-01KQ7JDT511BZ (B38-F9) | stale-worktree | Two active worktrees exist for the same feature: `WT-01KQEX083D5Y4` (branch already merged to main per health check) and `WT-01KQEX7T3X5XD`. The first worktree's branch is already merged but the worktree was not cleaned up. The second worktree also has a stale branch display_id format (`FEAT-01KQ7-JDT511BZ` with a hyphen embedded in the entity ID). | non-blocking |
| CG-6 | FEAT-01KQ7JDT511BZ (B38-F9) | branch-staleness | Branch `feature/FEAT-01KQ7JDT511BZ-work-tree-migration` is **23 commits behind main**. While it has no conflicts, the drift is significant and the branch should be rebased before merge. | non-blocking |
| CG-7 | B38-plans-and-batches | prior-review | Prior review `P38-plans-and-batches/report-p38-report-review-f2-f7` (for F2-F7) is still in **draft** status. A batch-level review should either supersede or incorporate prior reviews. | non-blocking |

## Documentation Currency

- **AGENTS.md:** References stale tool names (`sh`, `graph_project`, `main`) — flagged by health check.
- **B38 scope guard:** B38 is not listed in AGENTS.md scope guard section — 27 prior batches are also missing from the scope guard.
- **Workflow skills:** Multiple stale tool references in `.agents/skills/kanbanzai-getting-started/SKILL.md`, `.agents/skills/kanbanzai-documents/SKILL.md`, `.agents/skills/kanbanzai-workflow/SKILL.md`, `.agents/skills/kanbanzai-agents/SKILL.md`, and `.agents/skills/kanbanzai-plan-review/SKILL.md`. These are **pre-existing** issues not introduced by B38 but are blocking for project-level documentation currency.
- **Knowledge entries:** No knowledge entries contributed under B38 scope.
- **B38 design doc:** Design `P38-plans-and-batches/design-meta-planning-plans-and-batches` is approved (registered under the legacy P38 ID).

## Cross-Cutting Checks

- **Health check:** 9 errors (1 branch conflict on FEAT-01KQ7YQK6DDDA, 8 TTL-expired knowledge entries). Warnings include stale worktrees, outdated doc references, and missing scope guard entries.
- **Orphaned worktrees:** 4 active worktrees for B38 features — 2 for F9 (one stale/merged), 1 for F1 (conflicted), 1 for F10 (superseded). All should be resolved before batch closure.
- **Cohort merge checkpoints:** Not applicable — B38 features were implemented sequentially or in ad-hoc parallel, not via formal cohorts.

## Retrospective Summary

No retrospective signals were recorded for B38-plans-and-batches scope (retro synthesis returned 0 signals). This is itself a signal: the batch's features completed with no recorded observations about workflow friction, tool gaps, or things that worked well. For a batch of this size and complexity (10 features spanning a foundational entity model change), the absence of recorded retro signals suggests the retrospective contribution step was skipped during individual feature completion.

## Batch Verdict

**fail** — 4 blocking conformance gaps (CG-1 through CG-4) prevent batch closure:

1. **B38-F8 (State File Migration)**: Spec is draft. Zero tasks. No implementation exists. This feature cannot be in `reviewing` — it hasn't been specified, planned, or developed.
2. **B38-F1 (Config Schema)**: Branch has merge conflicts with main. Cannot be merged until resolved.

### Resolution Path

1. **B38-F8**: Either (a) approve the spec, create tasks, implement, and review the feature; or (b) if the work is truly covered by B38-F9 (Work Tree Migration) + B38-F10 (superseded), then supersede this feature with a documented rationale.
2. **B38-F1**: Rebase the feature branch onto main and resolve merge conflicts, then re-verify.
3. **B38-F9**: Rebase onto main (23 commits behind), clean up the stale/merged worktree, then run the feature review stage.
4. After all 3 reviewing features reach `done`, re-run this batch review.

## Evidence

- Batch entity: `entity(action: "get", id: "B38-plans-and-batches", type: "batch")`
- Feature list: `entity(action: "list", type: "feature", parent: "B38-plans-and-batches")` → 10 features
- Health check: `health()` — merge conflict on FEAT-01KQ7YQK6DDDA, 8 TTL-expired knowledge entries
- Retro synthesis: `retro(action: "synthesise", scope: "B38-plans-and-batches")` → 0 signals
- Worktrees: `worktree(action: "list", status: "active")` → 4 active for B38 features
- Branch status: F1 has merge conflicts; F9 is 23 commits behind main
