# Batch Conformance Review: B42-worktree-cleanup-automation

## Scope
- **Batch:** B42-worktree-cleanup-automation
- **Parent Plan:** P40-retro-batch-april-2026
- **Features:** 4 total (2 done, 2 reviewing, 0 cancelled/superseded, 0 incomplete)
- **Original review date:** 2026-05-01
- **Re-review date:** 2026-05-01
- **Reviewer:** reviewer-conformance
- **Review cycle:** Re-review (prior review findings re-examined)

## Feature Census

| Feature | Status | Spec Approved | Dev-Plan | Branch Merged to Main | Notes |
|---------|--------|---------------|----------|----------------------|-------|
| B42-F1: FEAT-01KQG3TZR2JDP — Add worktree gc for Orphaned Records | reviewing | yes (batch spec) | yes | ✅ Yes | 2 tasks done. gc action + dry_run on main. 4 tests pass. |
| B42-F2: FEAT-01KQG3TZZD0ZG — Skip Drift Alerts for Merged Branches | done | yes (batch spec) | yes | ❌ **No** — branch not merged to main | 1 task done. `filterDrift` + `MergedAt` check exists on feature branch only. **Unchanged since original review.** |
| B42-F3: FEAT-01KQG3V03PAM4 — Auto-Schedule Cleanup on Merge | reviewing | yes (batch spec) | yes | ✅ Yes | 1 task done. Post-merge cleanup and branch deletion on main. 5 tests pass. |
| B42-F4: FEAT-01KQG3V06KNZG — Validate Entity IDs at Worktree Creation | done | yes (batch spec) | yes | ❌ **No** — branch **has merge conflicts** with main | 1 task done. `isDisplayEntityID` validation exists on feature branch only. **Unchanged since original review.** |

## Re-Review Summary

This is a re-review triggered by prior findings CG-1 (blocking) and CG-2 (blocking). Per the review-plan skill prerequisite #3, only findings from the prior review were re-examined.

### CG-1 Re-Examination: B42-F2 drift suppression not on main

**Original finding:** Feature `FEAT-01KQG3TZZD0ZG` is marked `done` but branch `feature/FEAT-01KQG3TZZD0ZG-skip-drift-merged-branches` is NOT merged to main. REQ-004 and REQ-005 are not satisfied on main.

**Re-verification:** `git branch --merged main | grep "FEAT-01KQG3TZZD0ZG"` returned no match (exit code 1). The branch is still not merged to main.

**Status:** ⛔ **BLOCKING — unchanged.** The drift suppression code (`filterDrift` + `MergedAt` check in `branch_tool.go`) exists only on the feature branch. The `main` branch's `branchStatusAction` does not suppress drift alerts for merged worktrees.

### CG-2 Re-Examination: B42-F4 ID validation not on main, merge conflicts

**Original finding:** Feature `FEAT-01KQG3V06KNZG` is marked `done` but branch `feature/FEAT-01KQG3V06KNZG-validate-worktree-entity-ids` is NOT merged to main AND has merge conflicts with main. REQ-008 and REQ-009 are not satisfied on main.

**Re-verification:** Dry-run merge `git merge --no-commit --no-ff feature/FEAT-01KQG3V06KNZG-validate-worktree-entity-ids` failed with:
```
CONFLICT (content): Merge conflict in internal/mcp/worktree_tool_test.go
Automatic merge failed; fix conflicts and then commit the result.
```
Merge was aborted after verification. The health check also confirms: `branch feature/FEAT-01KQG3V06KNZG-validate-worktree-entity-ids: branch has merge conflicts with main`.

**Status:** ⛔ **BLOCKING — unchanged.** The entity ID validation code (`isDisplayEntityID` + `displayToCanonical` in `worktree_tool.go`) exists only on the feature branch. The merge conflict must be resolved before the branch can be merged.

### CG-3 and CG-4: Override rationales (previously resolved)

Both were resolved in the original review and remain non-blocking. No re-examination needed.

## Conformance Gaps

| # | Feature | Type | Description | Severity |
|---|---------|------|-------------|----------|
| **CG-1** | FEAT-01KQG3TZZD0ZG | delivery | Feature is marked `done` but its branch `feature/FEAT-01KQG3TZZD0ZG-skip-drift-merged-branches` is **NOT merged to main**. The drift suppression code (filterDrift + MergedAt check in branch_tool.go) exists only on the feature branch. The `main` branch's `branchStatusAction` does not suppress drift alerts for merged worktrees. **REQ-004/AC-004 and REQ-005/AC-005 are not satisfied on main.** | **blocking** |
| **CG-2** | FEAT-01KQG3V06KNZG | delivery | Feature is marked `done` but its branch `feature/FEAT-01KQG3V06KNZG-validate-worktree-entity-ids` is **NOT merged to main** and **has merge conflicts with main**. The display-ID validation (isDisplayEntityID + displayToCanonical in worktree_tool.go) exists only on the feature branch. The `main` branch's `worktreeCreateAction` does not reject display-format entity IDs. **REQ-008/AC-008 and REQ-009/AC-009 are not satisfied on main.** | **blocking** |

## Spec Conformance Matrix

| AC | Requirement | Feature | Status on `main` | Evidence |
|----|-------------|---------|-----------------|----------|
| AC-001 | REQ-001/REQ-002: dry_run lists orphans | B42-F1 | ✅ Pass | `TestWorktreeGc_DryRunListsOrphaned` |
| AC-002 | REQ-003: gc removes orphaned state files | B42-F1 | ✅ Pass | `TestWorktreeGc_RemovesOrphanedStateFiles` |
| AC-003 | REQ-001: existing dir not removed | B42-F1 | ✅ Pass | `TestWorktreeGc_DoesNotRemoveExistingDirectories` |
| AC-004 | REQ-004: merged branch → no drift alert | B42-F2 | ❌ **Fail** — not on main | Code only on feature branch |
| AC-005 | REQ-005: unmerged branch → drift alert | B42-F2 | ❌ **Fail** — not on main | Code only on feature branch |
| AC-006 | REQ-006: merge schedules cleanup | B42-F3 | ✅ Pass | `TestExecuteMerge_WorktreeAppearsInCleanupList_AC006` |
| AC-007 | REQ-007: squash merge deletes branch | B42-F3 | ✅ Pass | `TestExecuteMerge_SquashMergeDeletesLocalBranch_AC007` |
| AC-008 | REQ-008: display ID rejected at create | B42-F4 | ❌ **Fail** — not on main, merge conflicts | Code only on feature branch |
| AC-009 | REQ-009: canonical ID accepted | B42-F4 | ❌ **Fail** — not on main | Code only on feature branch |
| AC-010 | REQ-NF-001: no git during gc detection | B42-F1 | ✅ Pass | `TestWorktreeGc_NoGitInvocation` + code review |
| AC-011 | REQ-NF-002: cleanup failure → merge success | B42-F3 | ✅ Pass | `TestExecuteMerge_CleanupFailureDoesNotFailMerge_AC011` |

**Summary:** 6/11 ACs pass on main. 4 ACs (AC-004, AC-005, AC-008, AC-009) are not on main. No change from original review.

## Documentation Currency

- **AGENTS.md**: Contains stale references (pre-existing, not caused by B42). B42 is not listed in the scope guard — consistent with many other done batches.
- **Workflow skills**: Multiple stale tool references (pre-existing, not caused by B42).
- **Knowledge entries**: 0 contributed during this batch's features. No change.

## Cross-Cutting Checks (health())

- **B42-F4 merge conflict:** `branch feature/FEAT-01KQG3V06KNZG-validate-worktree-entity-ids: branch has merge conflicts with main` — still present.
- **B42-F1 and B42-F3 worktree records:** Both still show active but their branches are merged — should be transitioned to `merged`.
- **Orphaned worktrees:** 4 orphaned worktree records remain (WT-01KQAGN5SVNGZ, WT-01KQEX083D5Y4, WT-01KQEX7T3X5XD, WT-01KQF7Y0YQS36). The gc feature (B42-F1) is on main but has not been run.
- **P40 status:** `unknown batch status "idea"` — the parent plan has an unrecognised lifecycle status. This is pre-existing and not caused by B42.

## Retrospective Summary

No retrospective signals were recorded during the batch's implementation or since the original review. The retro synthesis returned 0 signals. For a batch that overrode all four features through the `reviewing→done` gate without creating individual review documents, the lack of retrospective observations means lessons (both positive and negative) are not captured.

## Batch Verdict

**FAIL** — 2 blocking conformance gaps persist from the original review. No changes since the original review on 2026-05-01.

### Blocking reasons (unchanged)

1. **B42-F2** (`FEAT-01KQG3TZZD0ZG`): The drift suppression code is not on `main`. The feature is marked `done` but REQ-004 and REQ-005 are not operational in the production code.
2. **B42-F4** (`FEAT-01KQG3V06KNZG`): The entity ID validation code is not on `main` AND has merge conflicts. REQ-008 and REQ-009 are not operational in the production code.

### Remediation (unchanged)

To achieve conformance pass:
1. Merge B42-F2's branch (`feature/FEAT-01KQG3TZZD0ZG-skip-drift-merged-branches`) to main after resolving any base-branch divergence.
2. Resolve merge conflicts on B42-F4's branch (`feature/FEAT-01KQG3V06KNZG-validate-worktree-entity-ids`) and merge to main.
3. Run `worktree(action: gc)` to clean up the 4 orphaned worktree records.
4. Update the B42-F1 and B42-F3 worktree records from `active` to `merged` (they were already merged to main).

## Evidence

- **Prior review:** `work/reviews/batch-review-B42-worktree-cleanup-automation-2026-05-01.md` (original)
- **Batch entity:** `status(id: "B42-worktree-cleanup-automation")` — status: done, 4 features
- **CG-1 re-verification:** `git branch --merged main | grep "FEAT-01KQG3TZZD0ZG"` — no match (exit 1)
- **CG-2 re-verification:** `git merge --no-commit --no-ff feature/FEAT-01KQG3V06KNZG-validate-worktree-entity-ids` — CONFLICT in `internal/mcp/worktree_tool_test.go`; merge aborted
- **Health check:** `health()` — confirms CG-2 merge conflict, orphaned worktrees, and stale worktree records
- **Retro synthesis:** `retro(action: "synthesise", scope: "B42-worktree-cleanup-automation")` — 0 signals
