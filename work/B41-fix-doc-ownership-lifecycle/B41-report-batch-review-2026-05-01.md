# Batch Conformance Review: B41-fix-doc-ownership-lifecycle

## Scope
- **Batch:** B41-fix-doc-ownership-lifecycle (Fix Document Ownership and Lifecycle)
- **Parent Plan:** P40-retro-batch-april-2026
- **Features:** 4 total (4 done, 0 incomplete)
- **Review date:** 2026-05-01 (original), re-reviewed 2026-05-01
- **Reviewer:** reviewer-conformance
- **Re-review note:** This is an updated report. The prior review found 2 blocking gaps (CG-1, CG-2). The feature lifecycle states have advanced to `done` but the code still is not on `main`.

## Feature Census

| Feature | Status | Spec Approved | Dev-Plan | Tasks | Merge Status |
|---------|--------|---------------|----------|-------|--------------|
| FEAT-01KQG3AWTYSEQ (B41-F1) — Auto-Infer Document Owner from Path Context | done | yes | approved | 2/2 done | ✅ Merged to main (`b5dbe05f`) |
| FEAT-01KQG3AWX916X (B41-F2) — Preserve Approval Status on Minor Doc Edits | done | yes | approved | 1/1 done | ❌ NOT merged to main (`2b5fe15a` on `feature/FEAT-01KQG3AWX916X-preserve-approval-minor-edits`) |
| FEAT-01KQG3AWZ3HEZ (B41-F3) — Add bypassable Field to Merge Gate Results | done | yes | approved | 1/1 done | ✅ Merged to main (already existed from P31, `6a5181bf`) |
| FEAT-01KQG3AX1AD0K (B41-F4) — Add Verification Parameter to Entity Update | done | yes | approved | 2/2 done | ❌ NOT merged to main (`d43ed30a` on `feature/FEAT-01KQG3AX1AD0K-entity-update-verification`) |

### Changes since prior review
- B41-F2: `reviewing` → `done` (override: single task, CanonicalContentHash for formatting-only detection, 7 tests pass)
- B41-F4: `reviewing` → `done` (override: both tasks done, verification params on entity update, 4 tests pass)
- Neither branch has been merged. No PRs exist.

## Conformance Gaps

| # | Feature | Type | Description | Severity |
|---|---------|------|-------------|----------|
| CG-1 | FEAT-01KQG3AWX916X (B41-F2) | merge-status | Implementation exists in an unmerged branch. Code is NOT on `main`. No PR has been created. Feature entity is `done` but code is not delivered to the mainline. | **blocking** |
| CG-2 | FEAT-01KQG3AX1AD0K (B41-F4) | merge-status | Implementation exists in an unmerged branch. Code is NOT on `main`. No PR has been created. Feature entity is `done` but code is not delivered to the mainline. | **blocking** |
| CG-3 | FEAT-01KQG3AWZ3HEZ (B41-F3) | scope-delivery | Feature was verification-only — no new code was produced. The bypassable field was already implemented in P31. This is a legitimate outcome (the dev-plan confirms existing behaviour). | non-blocking |
| CG-4 | B41 (all features) | worktree-cleanup | All 4 feature worktrees are still active on disk. F1 (`WT-01KQG7CZB6APK`) and F3 (`WT-01KQG7CZBFQ62`) branches are already on main — worktrees should be removed. F2 (`WT-01KQG830W6MNJ`) and F4 (`WT-01KQG7CZB6JTS`) contain unmerged code. Health check flags `worktree_branch_merged` for F1 and F3. | non-blocking |
| CG-5 | B41-fix-doc-ownership-lifecycle | document-currency | Batch is `done` but not listed in AGENTS.md Scope Guard. This is a pre-existing systemic issue affecting all completed plans (not batch-specific). | non-blocking |

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

All four worktrees are still active on disk. F1 and F3 worktrees are stale (branch already on main but worktree still active — flagged by health check). F2 and F4 worktrees contain unmerged code.

## Health Check Findings (B41-specific)

- **worktree_branch_merged:** B41-F1 and B41-F3 worktrees are active but branches are already merged — cleanup needed.
- **feature_child_consistency:** FEAT-01KQG3AWZ3HEZ has all 1 child task(s) in terminal state but feature is `developing` — this appears to be a stale cache since the entity shows `done`.
- **estimation_coverage:** FEAT-01KQG3AWZ3HEZ has 1 task with no estimates — pre-existing, non-blocking.
- **doc_currency for B41:** Plan `B41-fix-doc-ownership-lifecycle` is `done` but not mentioned in AGENTS.md Scope Guard.
- **gate_overrides for B41-F2 and B41-F4:** Both features used overrides from `reviewing→done`. While the implementation work was complete, the merge-to-main step was skipped.

## Retrospective Summary

No retrospective signals were recorded for B41 specifically. The batch was completed in a single session with direct orchestration. The project-level retro synthesis surfaces 25 signals across 11 themes, dominated by tool-friction (stale binary, worktree file writing, heredoc escaping) and workflow-friction (commit discipline, worktree usage). None of these are unique to B41.

## Batch Verdict

### FAIL — 2 blocking conformance gaps (unchanged from prior review)

**CG-1** and **CG-2** remain blocking: B41-F2 (Preserve Approval Status on Minor Doc Edits) and B41-F4 (Add Verification Parameter to Entity Update) have been implemented and tested on feature branches, and their feature entities have advanced to `done`, but **neither branch has been merged to `main`**. No pull requests exist. The code exists only in unmerged local branches. Until these are merged, the features are not delivered to the mainline.

The lifecycle state advancement from `reviewing` to `done` for both features does not change the conformance outcome — the code is still not on `main`, which is the defining criterion for a merge-status conformance gap.

## Required Actions

1. **Merge B41-F2:** Create PR and merge `feature/FEAT-01KQG3AWX916X-preserve-approval-minor-edits` into `main`.
2. **Merge B41-F4:** Create PR and merge `feature/FEAT-01KQG3AX1AD0K-entity-update-verification` into `main`.
3. **Clean up worktrees:** Run `cleanup(action: "execute")` or `worktree(action: "remove")` for F1 and F3 worktrees (already on main). After F2 and F4 are merged, clean up those worktrees too.

After actions 1 and 2 are complete, re-run this conformance review for pass.

## Evidence
- Batch entity: `entity(action: "get", id: "B41-fix-doc-ownership-lifecycle")` → status: done
- Feature list: `entity(action: "get")` for all 4 features → all done
- Spec approval: `doc(action: "get", id: "B41-fix-doc-ownership-lifecycle/spec-p40-spec-b41-doc-ownership-lifecycle")` → approved
- Design approval: `doc(action: "get", id: "P40-retro-batch-april-2026/design-p40-design-retro-batch-improvements")` → approved
- Dev-plans: All 4 dev-plans approved
- Health check: `health()` → B41-F1/F3 worktree cleanup warnings; B41 Scope Guard doc-currency gap
- Branch status: `git branch --merged main` confirms F1, F3 on main; F2, F4 unmerged (1 commit ahead each)
- Git worktrees: All 4 still active on disk — `git worktree list` confirms
- Retro synthesis: `retro(action: "synthesise", scope: "B41-fix-doc-ownership-lifecycle")` → 0 signals
