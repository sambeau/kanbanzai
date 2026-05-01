# Batch Conformance Review: B42-worktree-cleanup-automation

## Scope
- **Batch:** B42-worktree-cleanup-automation
- **Parent Plan:** P40-retro-batch-april-2026
- **Features:** 4 total (4 done, 0 cancelled/superseded, 0 incomplete)
- **Review date:** 2026-05-01
- **Reviewer:** reviewer-conformance

## Feature Census

| Feature | Status | Spec Approved | Dev-Plan | Branch Merged to Main | Notes |
|---------|--------|---------------|----------|----------------------|-------|
| B42-F1: FEAT-01KQG3TZR2JDP — Add worktree gc for Orphaned Records | done | yes (batch spec) | yes | ✅ Yes | 2 tasks done. gc action + dry_run on main. 4 tests pass. |
| B42-F2: FEAT-01KQG3TZZD0ZG — Skip Drift Alerts for Merged Branches | done | yes (batch spec) | yes | ❌ **No** — branch not merged to main | 1 task done. `filterDrift` + `MergedAt` check exists on feature branch only. |
| B42-F3: FEAT-01KQG3V03PAM4 — Auto-Schedule Cleanup on Merge | done | yes (batch spec) | yes | ✅ Yes | 1 task done. Post-merge cleanup and branch deletion on main. 5 tests pass. |
| B42-F4: FEAT-01KQG3V06KNZG — Validate Entity IDs at Worktree Creation | done | yes (batch spec) | yes | ❌ **No** — branch **has merge conflicts** with main | 1 task done. `isDisplayEntityID` validation exists on feature branch only. |

## Conformance Gaps

| # | Feature | Type | Description | Severity |
|---|---------|------|-------------|----------|
| **CG-1** | FEAT-01KQG3TZZD0ZG | delivery | Feature is marked `done` but its branch `feature/FEAT-01KQG3TZZD0ZG-skip-drift-merged-branches` is **NOT merged to main**. The drift suppression code (filterDrift + MergedAt check in branch_tool.go) exists only on the feature branch. The `main` branch's `branchStatusAction` does not suppress drift alerts for merged worktrees. **REQ-004/AC-004 are not satisfied on main.** | **blocking** |
| **CG-2** | FEAT-01KQG3V06KNZG | delivery | Feature is marked `done` but its branch `feature/FEAT-01KQG3V06KNZG-validate-worktree-entity-ids` is **NOT merged to main** and **has merge conflicts with main**. The display-ID validation (isDisplayEntityID + displayToCanonical in worktree_tool.go) exists only on the feature branch. The `main` branch's `worktreeCreateAction` does not reject display-format entity IDs. **REQ-008/AC-008 is not satisfied on main.** | **blocking** |
| **CG-3** | FEAT-01KQG3V03PAM4 | evidence | Override rationale states "5 tests pass". On main, only 4 tests exist for post-merge cleanup (AC-006, AC-007, AC-011, and SetMergedAt — all in `merge_tool_cleanup_test.go`). The 5th test (DeleteBranchFalse_PreservesBranch) exists, making 5 total tests. **Resolved — no gap.** | non-blocking |
| **CG-4** | FEAT-01KQG3TZR2JDP | evidence | Override rationale states "both tasks complete, gc action with dry_run, 4 tests pass". Verified: 4 gc tests exist on main (AC-001, AC-002, AC-003, AC-010). **Resolved — no gap.** | non-blocking |

### Detailed Findings

#### CG-1: B42-F2 drift suppression not on main

The feature branch `feature/FEAT-01KQG3TZZD0ZG-skip-drift-merged-branches` contains:
- `branchStatusAction`: checks `record.MergedAt != nil` and calls `filterDrift()` to suppress drift warnings/errors
- `filterDrift()` helper function
- Tests for AC-004 (merged branch → no drift alerts) and AC-005 (unmerged branch → alerts emitted)

The `main` branch's `branch_tool.go` does NOT contain any of these changes. The `branch(action: "status")` tool on main will report drift warnings/errors for merged branches, violating REQ-004.

#### CG-2: B42-F4 ID validation not on main, merge conflicts

The feature branch `feature/FEAT-01KQG3V06KNZG-validate-worktree-entity-ids` contains:
- `isDisplayEntityID()` — detects display-format IDs (≥2 hyphens)
- `displayToCanonical()` — converts display IDs to canonical form
- Validation in `worktreeCreateAction` that rejects display-format IDs with suggestion error
- Tests for AC-008 (display ID rejected) and AC-009 (canonical ID accepted)

The `main` branch's `worktreeCreateAction` does NOT contain any of these changes. Furthermore, the health check reports `branch has merge conflicts with main`, meaning a simple merge will not work — the conflicts must be resolved first.

#### CG-3 and CG-4: Override rationales

Both overrides are truthful about their state. However, they bypassed the `reviewing` gate — meaning no formal review document was created for either feature. This is a pattern concern but not a blocking issue at batch-review level since the batch is reviewed here.

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

**Summary:** 6/11 ACs pass on main. 4 ACs (AC-004, AC-005, AC-008, AC-009) are not on main. AC-010 passes via code inspection.

## Documentation Currency

- **AGENTS.md**: Contains stale references (as reported by health check). The `main` tool reference is outdated. B42 does not change AGENTS.md — this is a pre-existing issue.
- **Workflow skills**: Multiple stale tool references reported across kanbanzai-agents, kanbanzai-documents, kanbanzai-getting-started, and kanbanzai-workflow skills. These pre-date B42.
- **AGENTS.md Scope Guard**: B42-worktree-cleanup-automation is not listed in the scope guard. This is a pre-existing pattern (many done batches are unlisted).
- **Knowledge entries**: 0 contributed during this batch's features. No knowledge confirmation needed.

## Cross-Cutting Checks (health())

### Orphaned worktrees
4 orphaned worktree records exist (WT-01KQAGN5SVNGZ, WT-01KQEX083D5Y4, WT-01KQEX7T3X5XD, WT-01KQF7Y0YQS36). The `worktree(action: gc)` feature (B42-F1) — which IS on main — is intended to handle these. However, it has not been run yet, so these remain. **Post-batch action:** run `worktree(action: gc)` to clean up.

### B42-related health warnings
- `FEAT-01KQG3TZR2JDP` (B42-F1): "active but branch already merged" — should have been marked merged/abandoned post-merge
- `FEAT-01KQG3V03PAM4` (B42-F3): "active but branch already merged" — same issue
- `FEAT-01KQG3V06KNZG` (B42-F4): "branch has merge conflicts with main" — **merge conflicts need resolution**
- Both B42-F1 and B42-F3 have merged branches but their worktree records still show `active` status. The post-merge cleanup feature (B42-F3) should have transitioned them to `merged`. This suggests the auto-advance did not fully update worktree status for these features, or the worktree records are stale.

### Cleanup overdue
4 worktrees are overdue for cleanup (by 0 days). Post-merge auto-cleanup (B42-F3) should reduce this going forward.

## Retrospective Summary

No retrospective signals were recorded during the batch's implementation. The retro synthesis returned 0 signals — this is itself a notable finding. For a batch that overrode all four features through the `reviewing→done` gate without creating individual review documents, the lack of retrospective observations means lessons (both positive and negative) are not captured. The override practice was widely used across this batch and should be reflected in retrospectives.

## Batch Verdict

**FAIL** — 2 blocking conformance gaps exist.

### Blocking reasons
1. **B42-F2** (`FEAT-01KQG3TZZD0ZG`): The drift suppression code is not on `main`. The feature is marked `done` but REQ-004 and REQ-005 are not operational in the production code.
2. **B42-F4** (`FEAT-01KQG3V06KNZG`): The entity ID validation code is not on `main` AND has merge conflicts. REQ-008 and REQ-009 are not operational in the production code.

### Remediation
To achieve conformance pass:
1. Merge B42-F2's branch (`feature/FEAT-01KQG3TZZD0ZG-skip-drift-merged-branches`) to main after resolving any base-branch divergence.
2. Resolve merge conflicts on B42-F4's branch (`feature/FEAT-01KQG3V06KNZG-validate-worktree-entity-ids`) and merge to main.
3. Run `worktree(action: gc)` to clean up the 4 orphaned worktree records.
4. Update the B42-F1 and B42-F3 worktree records from `active` to `merged` (they were already merged to main).

## Evidence

- **Batch entity:** `entity(action: "get", id: "B42-worktree-cleanup-automation")` — status: done
- **Feature list:** `entity(action: "list", type: "feature", parent: "B42-worktree-cleanup-automation")` — 4 features, all done
- **Health check:** `health()` — see cross-cutting checks above
- **Retro synthesis:** `retro(action: "synthesise", scope: "B42-worktree-cleanup-automation")` — 0 signals
- **Branch merge status:** `git branch --merged main` — B42-F1 and B42-F3 branches merged; B42-F2 and B42-F4 are NOT
- **Main branch_tool.go:** confirmed no drift suppression (verified via git show)
- **Main worktree_tool.go:** confirmed no display-ID validation (verified via git show)
