# Batch Conformance Review: B41-fix-doc-ownership-lifecycle

## Scope
- **Batch:** B41-fix-doc-ownership-lifecycle (Fix Document Ownership and Lifecycle)
- **Parent Plan:** P40-retro-batch-april-2026
- **Features:** 4 total (3 delivered, 1 verification-only, 0 incomplete)
- **Review date:** 2026-05-01
- **Reviewer:** reviewer-conformance

## Feature Census

| Feature | Status | Spec Approved | Dev-Plan | Tasks | Notes |
|---------|--------|---------------|----------|-------|-------|
| FEAT-01KQG3AWTYSEQ (B41-F1) — Auto-Infer Document Owner from Path Context | done | yes | approved | 2/2 done | Code merged to main (`b5dbe05f`). Owner inference + conflict warning + PROJECT/ fallback implemented. |
| FEAT-01KQG3AWX916X (B41-F2) — Preserve Approval Status on Minor Doc Edits | done | yes | approved | 1/1 done | Code in unmerged branch (`feature/FEAT-01KQG3AWX916X-preserve-approval-minor-edits`, 1 commit ahead of main). Not yet on main. |
| FEAT-01KQG3AWZ3HEZ (B41-F3) — Add bypassable Field to Merge Gate Results | done | yes | approved | 1/1 done | **Verification-only.** Bypassable field already implemented in P31 (`6a5181bf`). Task confirmed existing behaviour. No new code was written for this feature. |
| FEAT-01KQG3AX1AD0K (B41-F4) — Add Verification Parameter to Entity Update | done | yes | approved | 2/2 done | Code in unmerged branch (`feature/FEAT-01KQG3AX1AD0K-entity-update-verification`, 1 commit ahead of main). Not yet on main. |

## Conformance Gaps

| # | Feature | Type | Description | Severity |
|---|---------|------|-------------|----------|
| CG-1 | FEAT-01KQG3AWX916X (B41-F2) | merge-status | Implementation exists in an unmerged branch. Code is NOT on `main`. No PR has been created. | **blocking** |
| CG-2 | FEAT-01KQG3AX1AD0K (B41-F4) | merge-status | Implementation exists in an unmerged branch. Code is NOT on `main`. No PR has been created. | **blocking** |
| CG-3 | FEAT-01KQG3AWZ3HEZ (B41-F3) | scope-delivery | Feature was verification-only — no new code was produced. The bypassable field was already implemented in a prior plan (P31). This is a legitimate outcome (the dev-plan confirms it), but the feature effectively validated existing behaviour rather than delivering new work. | non-blocking |
| CG-4 | B41 (all features) | worktree-cleanup | All 4 feature worktrees are still `active` despite branches being merged (F1, F3) or commits existing (F2, F4). Health check flags `worktree_branch_merged` for F1 and F3. Cleanup is overdue. | non-blocking |
| CG-5 | B41-fix-doc-ownership-lifecycle | document-currency | Batch is `done` but not listed in AGENTS.md Scope Guard. This is a pre-existing issue affecting all completed plans (systemic, not batch-specific). | non-blocking |

## Spec Conformance Verification

### REQ-001 (Owner inference from path) — **PASS**
- Code present on main: `extractPlanOrBatchSlug()` in `internal/service/documents.go` (L1118-L1133) extracts batch/plan slug from path.
- If entity exists, it's used as owner instead of `PROJECT/`.
- Tested by `TestSubmitDocument_InferOwnerFromBatchPath` and `TestSubmitDocument_ExplicitOwnerTakesPrecedence`.

### REQ-002 (Conflict warning) — **PASS**
- Code present on main: path conflict warning emitted when existing registration differs (L299-L312 of `documents.go`).
- Tested by `TestSubmitDocument_ExplicitOwnerNoConflictWarning`.

### REQ-003 (Fallback to PROJECT/) — **PASS**
- Code present on main: when no entity is resolved, `owner` remains empty and defaults to `PROJECT/`.
- Tested by `TestSubmitDocument_NoEntityHook_NoInference` and `TestSubmitDocument_NonPlanEntityNotInferred`.

### REQ-004 (Preserve approval on formatting-only changes) — **PENDING** (not on main)
- Implementation exists in unmerged branch `feature/FEAT-01KQG3AWX916X-preserve-approval-minor-edits`.
- Commit `2b5fe15a` adds scope-detection logic + `storage.CanonicalContentHash` + 431-line refresh test file.
- Cannot verify on main. **Blocking until merged.**

### REQ-005 (Warn on substantive changes) — **PENDING** (not on main)
- Same branch as REQ-004. Cannot verify on main. **Blocking until merged.**

### REQ-006 (bypassable field on merge gate results) — **PASS** (verification-only)
- `Bypassable bool` on `GateResult` struct (`internal/merge/gate.go` L36-43).
- `ReviewReportExistsGate` returns `Bypassable: false` (`internal/merge/gates.go` L433).
- All other gates return `Bypassable: true`.
- `NonBypassableBlockingFailures()` helper exists (`internal/merge/checker.go` L95-105).
- Tested by `TestCheckGates_ReviewingFeature_NoReport_Blocked` and `TestCheckGates_ExistingGates_AreBypassable`.
- Already on main from P31.

### REQ-007 (Verification params on entity update) — **PENDING** (not on main)
- Implementation exists in unmerged branch `feature/FEAT-01KQG3AX1AD0K-entity-update-verification`.
- Commit `d43ed30a` adds `verification` and `verification_status` params to entity update handler + 113-line test file.
- Cannot verify on main. **Blocking until merged.**

### REQ-008 (Error for unsupported entity types) — **PENDING** (not on main)
- Same branch as REQ-007. Cannot verify on main. **Blocking until merged.**

## Documentation Currency

- **AGENTS.md Scope Guard:** B41 is not listed. This is a pre-existing systemic issue (all completed plans are absent).
- **Workflow skills:** Stale tool references exist across `.agents/skills/` files (pre-existing, not caused by this batch).
- **Knowledge entries:** No knowledge entries were contributed during this batch's features (0 signals in retro synthesis).
- The batch's new capabilities (owner inference, change-scope detection, merge gate bypassable field, entity verification params) don't change existing skills or procedures in a way that requires documentation updates beyond the scope guard.

## Worktree Audit

| Feature | Worktree ID | Branch | Status | On Main? |
|---------|-------------|--------|--------|----------|
| B41-F1 | WT-01KQG7CZB6APK | `feature/FEAT-01KQG3AWTYSEQ-auto-infer-doc-owner` | active | ✅ Merged |
| B41-F2 | WT-01KQG830W6MNJ | `feature/FEAT-01KQG3AWX916X-preserve-approval-minor-edits` | active | ❌ Not merged |
| B41-F3 | WT-01KQG7CZBFQ62 | `feature/FEAT-01KQG3AWZ3HEZ-merge-gate-bypassable` | active | ✅ Merged (via P31) |
| B41-F4 | WT-01KQG7CZB6JTS | `feature/FEAT-01KQG3AX1AD0K-entity-update-verification` | active | ❌ Not merged |

All worktrees should be `merged` or `abandoned`. F1 and F3 worktrees are stale (branch already on main but worktree still active — flagged by health check). F2 and F4 worktrees contain unmerged code.

## Health Check Findings

- **feature_child_consistency:** FEAT-01KQG3AWZ3HEZ has 1 child task in terminal state but feature was `developing` — resolved by the override to done.
- **estimation_coverage:** FEAT-01KQG3AWZ3HEZ has 1 task with no estimates — pre-existing, non-blocking.
- **worktree_branch_merged:** B41-F1 and B41-F3 worktrees are active but branches are already merged — cleanup needed.
- No errors directly related to B41 entities (the batch-level parent plan P40 has an unknown status `"idea"`, which is a separate issue).

## Retrospective Summary

No retrospective signals were recorded for B41 specifically. The batch was completed in a single session with direct orchestration. The project-level retro synthesis surface 25 signals across 11 themes, dominated by tool-friction (stale binary, worktree file writing, heredoc escaping) and workflow-friction (commit discipline, worktree usage). None of these are unique to B41.

## Batch Verdict

### FAIL — 2 blocking conformance gaps

**CG-1** and **CG-2** are blocking: B41-F2 (Preserve Approval Status) and B41-F4 (Entity Update Verification) have been implemented and tested on feature branches, but **neither branch has been merged to `main`**. No pull requests exist. The code exists only in unmerged local branches. Until these are merged, the features are not delivered to the mainline.

**CG-3** (F3 verification-only) is non-blocking but worth documenting: the bypassable field feature was correctly identified as already implemented by P31, and the task verified this rather than producing new code. The dev-plan and spec handled this appropriately.

## Required Actions

1. **Merge B41-F2:** Create PR and merge `feature/FEAT-01KQG3AWX916X-preserve-approval-minor-edits` into `main`.
2. **Merge B41-F4:** Create PR and merge `feature/FEAT-01KQG3AX1AD0K-entity-update-verification` into `main`.
3. **Clean up worktrees:** Run `cleanup(action: "execute")` or `worktree(action: "remove")` for F1 and F3 worktrees (already on main).

After actions 1 and 2 are complete, re-run this conformance review for pass.

## Evidence
- Batch entity: `entity(action: "get", id: "B41-fix-doc-ownership-lifecycle")` → status: done
- Feature list: `entity(action: "list", type: "feature", parent: "B41-fix-doc-ownership-lifecycle")` → 4 features, all tasks done
- Spec approval: `doc(action: "get", id: "B41-fix-doc-ownership-lifecycle/spec-p40-spec-b41-doc-ownership-lifecycle")` → approved
- Design approval: `doc(action: "get", id: "P40-retro-batch-april-2026/design-p40-design-retro-batch-improvements")` → approved
- Dev-plans: All 4 dev-plans approved
- Health check: `health()` → no B41-specific errors
- Branch status: `git branch --merged main` confirms F1, F3 on main; F2, F4 unmerged
- Retro synthesis: `retro(action: "synthesise", scope: "B41-fix-doc-ownership-lifecycle")` → 0 signals
