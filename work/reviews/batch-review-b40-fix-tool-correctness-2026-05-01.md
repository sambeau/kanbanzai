# Batch Conformance Review: B40-fix-tool-correctness

## Scope
- **Batch:** B40-fix-tool-correctness (Fix Tool Correctness and Reliability)
- **Parent Plan:** P40-retro-batch-april-2026
- **Features:** 4 total (2 done, 2 done-but-cache-stale-showing-as-reviewing)
- **Review date:** 2026-05-01 (re-review; prior review same date 09:56 UTC)
- **Reviewer:** reviewer-conformance
- **Review cycle:** 2 (re-review of prior findings)

## Re-Review Context

This is a re-review. The prior review (same date, 09:56 UTC) identified four conformance gaps:
CG-1 (blocking: F1/F2 branches not merged), CG-2 (non-blocking: F3/F4 worktree cleanup),
CG-3 (non-blocking: cache/index staleness), CG-4 (non-blocking: scope guard gap).

Per the review-plan skill prerequisite #3, this re-review only re-examines the findings from
the prior review rather than re-reviewing every feature from scratch.

## Feature Census

| Feature | ID | Status (entity get) | Status (entity list) | Branch Merged? | Notes |
|---------|-----|---------------------|----------------------|----------------|-------|
| B40-F1 — Fix entity list parent_feature Filter | FEAT-01KQG2Q5MDC6E | done ✅ | reviewing ⚠️ | **NO** (48 commits behind, 1 ahead) | Cache staleness: entity get shows done, entity list shows reviewing. See CG-1, CG-3. |
| B40-F2 — Unify finish() State Propagation | FEAT-01KQG2Q5Q1HX9 | done ✅ | reviewing ⚠️ | **NO** (44 commits behind, 2 ahead) | Cache staleness: entity get shows done, entity list shows reviewing. See CG-1, CG-3. |
| B40-F3 — Make Multi-Edit Calls Atomic | FEAT-01KQG2Q5RS9G9 | done ✅ | done ✅ | YES ✅ | Worktree still active. See CG-2, CG-3. |
| B40-F4 — Expand Decompose AC Format Recognition | FEAT-01KQG2Q5V4S6W | done ✅ | done ✅ | YES ✅ | Worktree still active. See CG-2, CG-3. |

### Spec and Design Status
- **Batch specification:** `B40-fix-tool-correctness/spec-p40-spec-b40-tool-correctness` — **approved** (by sambeau, 2026-04-30)
- **Design:** `P40-retro-batch-april-2026/design-p40-design-retro-batch-improvements` — **approved** (by sambeau, 2026-04-30)
- All four features used the batch-level spec with overrides for `specifying → dev-planning`. No per-feature spec documents required.

### Task Status Detail
- **B40-F1:** TASK-01KQG2WSTPF5K — done ✅ (1 task)
- **B40-F2:** TASK-01KQG2WSY70G5, TASK-01KQG2WT0N1TR — done ✅ (2 tasks)
- **B40-F3:** TASK-01KQG2WT3GC31, TASK-01KQG2WT61X74 — done ✅ (2 tasks; `feature_child_consistency` warning persists — cache staleness)
- **B40-F4:** TASK-01KQG2WT8EVFH, TASK-01KQG2WTB2NFH, TASK-01KQG2WTDZHQJ — done ✅ (3 tasks; `feature_child_consistency` warning persists — cache staleness)

## Prior Finding Re-Examination

### CG-1 (blocking): F1 and F2 Branches Not Merged to Main

**Prior state:** Branches `feature/FEAT-01KQG2Q5MDC6E-fix-parent-feature-filter` and
`feature/FEAT-01KQG2Q5Q1HX9-unify-finish-state-propagation` not merged to main.
43 and 39 commits behind respectively.

**Current state (re-checked 2026-05-01):**
- F1 branch: 48 commits behind main, 1 commit ahead. Still NOT in `git branch --merged main`.
- F2 branch: 44 commits behind main, 2 commits ahead. Still NOT in `git branch --merged main`.

**Change from prior review:** Drift has increased slightly (43→48, 39→44) — the branches
have continued to fall behind main, consistent with no merge activity.

**Verdict:** **Still blocking.** No change. Both branches remain unmerged. Implementation
code exists only on feature branches, not on main.

### CG-2 (non-blocking): F3 and F4 Worktrees Still Active

**Prior state:** Worktrees WT-01KQG5SWH8QNK (F3) and WT-01KQG5SWH919E (F4) listed as
`active` despite their branches being merged to main. `worktree_branch_merged` warnings
in health().

**Current state (re-checked 2026-05-01):** Both worktrees still exist with status `active`.
Both still report `worktree_branch_merged` warnings in health(). F1 and F2 worktrees
(WT-01KQG5QGBJMMB, WT-01KQG6J5VWSQ9) are also active but their branches are not merged
so those are expected.

**Verdict:** **Still present, still non-blocking.** No change.

### CG-3 (non-blocking): Cache/Index Staleness

**Prior state:** `feature_child_consistency` warnings for F3 and F4 — features are `done`
but entity list showed non-terminal child tasks. Entity get confirmed tasks ARE done.

**Current state (re-checked 2026-05-01):**
- F3 and F4: `feature_child_consistency` warnings persist in health().
- **New:** F1 and F2 now also exhibit cache inconsistency: `entity get` returns `status: done`
  with review-cycle overrides recorded, but `entity list` returns `status: reviewing`.
  This is the same cache staleness pattern now affecting two additional features.

**Verdict:** **Still present, still non-blocking. Slightly worse scope** — F1 and F2 are
now also affected by cache staleness (list vs. get mismatch). This is ironic given that
F1 was supposed to fix the entity list filter and F2 was supposed to fix state propagation
consistency.

### CG-4 (non-blocking): Scope Guard Gap

**Prior state:** B40-fix-tool-correctness not mentioned in AGENTS.md Scope Guard. Preexisting
gap — consistent with all other completed batches/plans.

**Current state (re-checked 2026-05-01):** AGENTS.md Scope Guard section (lines 198–220)
remains unchanged. B40 is still not mentioned. Multiple other completed plans (B1 through B42)
also not mentioned — this is a system-wide documentation gap, not specific to B40.

**Verdict:** **Still present, still non-blocking.** No change.

## Spec Conformance Verification (from Prior Review — Not Re-Examined)

The prior review confirmed all 13 acceptance criteria (AC-001 through AC-013) are satisfied
by the implementation code. Since no code changes have occurred since the prior review
(only unmerged branches and cache staleness), these findings remain valid:

- **AC-001 through AC-012:** ✅ Satisfied (confirmed in prior review via branch diffs and
  worktree code inspection)
- **AC-013 (latency):** ⚠️ Cannot verify without CI/benchmark data (same as prior review)

## Worktree Status Summary

| Worktree ID | Feature | Branch | Status | Branch Merged? | Action Needed |
|-------------|---------|--------|--------|----------------|---------------|
| WT-01KQG5QGBJMMB | F1 | feature/FEAT-01KQG2Q5MDC6E-fix-parent-feature-filter | active | NO | Merge branch, then cleanup |
| WT-01KQG6J5VWSQ9 | F2 | feature/FEAT-01KQG2Q5Q1HX9-unify-finish-state-propagation | active | NO | Merge branch, then cleanup |
| WT-01KQG5SWH8QNK | F3 | feature/FEAT-01KQG2Q5RS9G9-atomic-multi-edit | active | YES | Cleanup (branch already merged) |
| WT-01KQG5SWH919E | F4 | feature/FEAT-01KQG2Q5V4S6W-expand-decompose-ac-formats | active | YES | Cleanup (branch already merged) |

## Documentation Currency

- **AGENTS.md:** Unchanged. Scope Guard gap is preexisting and system-wide.
- **Workflow skills:** Unchanged. Pre-existing doc_currency warnings in workflow skills
  are not introduced by B40.
- **Knowledge entries:** 0 knowledge entries contributed specifically to B40 scope.
- **Retro signals:** The prior review noted 0 retrospective signals — this remains unchanged.

## Batch Verdict

### **FAIL** ❌

**Rationale:** All four features have implementation code that is correct per the prior
review's spec conformance verification. However, **CG-1 remains a blocking gap**: F1
and F2 branches have not been merged to main. Since the prior review on 2026-05-01,
the branches have drifted further behind main (43→48 and 39→44 commits behind) with
no merge activity.

The prior review issued a PASS-WITH-NOTES verdict with the expectation that CG-1 would
be resolved promptly. It has not been. A re-review that finds the blocking gap still
unresolved after a prior cycle must escalate to FAIL per review-plan conventions.

The three non-blocking findings (CG-2, CG-3, CG-4) also remain unresolved. CG-3 has
actually expanded in scope — F1 and F2 now exhibit the same cache staleness pattern.

**Required to pass:**
1. Merge `feature/FEAT-01KQG2Q5MDC6E-fix-parent-feature-filter` to main
2. Merge `feature/FEAT-01KQG2Q5Q1HX9-unify-finish-state-propagation` to main
3. (Non-blocking but recommended) Clean up F3 and F4 worktrees
4. (Non-blocking but recommended) Rebuild cache to resolve staleness

## Evidence
- Branch merge check: `git branch --merged main | grep FEAT-01KQG2Q5` — only F3 and F4
- Commit drift: `git rev-list --count feature/FEAT-01KQG2Q5MDC6E-*..main` → 48 and 44
- Entity get: Both F1 and F2 confirm `status: done` with override records
- Entity list: Both F1 and F2 return `status: reviewing` (cache staleness)
- Worktree: All 4 worktrees active; F3/F4 branches already merged
- Health: `feature_child_consistency` warnings for F3, F4; `worktree_branch_merged` for F3, F4
- AGENTS.md: Scope Guard unchanged (lines 198–220)
- Spec/Design: Both approved; no changes since prior review
