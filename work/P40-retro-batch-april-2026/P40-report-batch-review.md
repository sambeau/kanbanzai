# Plan Conformance Review: P40 — Retrospective Batch, April 2026

## Scope

- **Plan:** P40-retro-batch-april-2026
- **Batches:** 4 total (4 done, 0 incomplete)
- **Features:** 14 total (14 done, 0 cancelled/superseded, 0 incomplete)
- **Review date:** 2026-05-01
- **Reviewer:** reviewer-conformance (orchestrated)

## Plan Overview

P40 was a retrospective-driven improvement plan covering 4 workstreams:

| Workstream | Batch | Theme | Focus |
|-----------|-------|-------|-------|
| **A** | B39 | Theme 1 (Critical) | Worktree Development Experience |
| **B** | B40 | Themes 2, 3 (Significant) | Tool Correctness & Reliability |
| **C** | B41 | Themes 4, 7 (Significant/Moderate) | Document Ownership & Lifecycle |
| **D** | B42 | Theme 6 (Moderate) | Worktree & Cleanup Automation |

## Batch Review Summaries

### B39 — Fix Worktree Development Experience — **PASS**

| Feature | Status | Spec | Delivered |
|---------|--------|------|-----------|
| F1: Document write_file as Primary Pattern | ✅ done | ✅ batch spec | Skill file updated, tool subset configured |
| F2: Make edit_file Worktree-Aware | ✅ done | ✅ batch spec | entity_id parameter, worktree resolution, 10 tests |

**Key findings:**
- All 11 ACs satisfied. `write_file(entity_id)` documented as primary pattern in `implement-task/SKILL.md`; `python3 -c` and heredoc workarounds removed.
- `edit_file` accepts optional `entity_id` with same O(1) resolution as `write_file`.
- Both branches merged to main. F1's worktree shows as active despite being merged (cleanup item).
- **No conformance gaps.**

---

### B40 — Fix Tool Correctness and Reliability — **PASS-WITH-NOTES**

| Feature | Status | Spec | Delivered |
|---------|--------|------|-----------|
| F1: Fix entity list parent_feature Filter | ✅ done | ✅ batch spec | Fix verified in entity_tool.go, tests pass |
| F2: Unify finish() State Propagation | ✅ done | ✅ batch spec | CacheRefresh unified write-through, integration test passes |
| F3: Make Multi-Edit Calls Atomic | ✅ done | ✅ batch spec | Pre-flight matching, 3 tests, **merged to main** |
| F4: Expand Decompose AC Format Recognition | ✅ done | ✅ batch spec | 4 new AC formats + diagnostic, 7 tests, **merged to main** |

**Key findings:**
- All 13 ACs (AC-001 through AC-012) satisfied by implementation code.
- F3 and F4 are **merged to main** — their code is live.
- **CG-1 (non-blocking):** F1 and F2 are NOT yet merged to main. The `parent_feature` filter fix and unified `finish()` write-through exist only on feature branches. These fixes are not yet available to users.
- **CG-2 (non-blocking):** F3 and F4 worktrees are still marked `active` despite being merged.

---

### B41 — Fix Document Ownership and Lifecycle — **FAIL** (2 blocking gaps)

| Feature | Status | Spec | Delivered |
|---------|--------|------|-----------|
| F1: Auto-Infer Document Owner | ✅ done | ✅ batch spec | 8 tests, **merged to main** |
| F2: Preserve Approval on Minor Edits | ✅ done | ✅ batch spec | CanonicalContentHash, 7 tests, **NOT on main** |
| F3: Add bypassable to Merge Gates | ✅ done | ✅ batch spec | Already present from P31; verified, **on main** |
| F4: Add Verification to Entity Update | ✅ done | ✅ batch spec | verification params, 4 tests, **NOT on main** |

**Key findings:**
- **CG-3 (blocking):** B41-F2 (preserve approval on minor edits) — code on feature branch `feature/FEAT-01KQG3AWX916X-preserve-approval-minor-edits`, **not merged to main**.
- **CG-4 (blocking):** B41-F4 (verification parameter on entity update) — code on feature branch `feature/FEAT-01KQG3AX1AD0K-entity-update-verification`, **not merged to main**.
- F1 and F3 are on main and working correctly.
- F3 was "already implemented in P31" — the feature verified an existing capability rather than implementing new code. This is a legitimate delivery for a retrospective batch (verification and documentation of existing behaviour).

---

### B42 — Worktree and Cleanup Automation — **FAIL** (2 blocking gaps)

| Feature | Status | Spec | Delivered |
|---------|--------|------|-----------|
| F1: Add worktree gc for Orphaned Records | ✅ done | ✅ batch spec | gc action + dry_run, 4 tests, **merged to main** |
| F2: Skip Drift Alerts for Merged Branches | ✅ done | ✅ batch spec | merged_at drift suppression, 3 tests, **NOT on main** |
| F3: Auto-Schedule Cleanup on Merge | ✅ done | ✅ batch spec | post-merge cleanup + branch deletion, 5 tests, **merged to main** |
| F4: Validate Entity IDs at Worktree Creation | ✅ done | ✅ batch spec | display-ID validation, 4 tests, **NOT on main (merge conflicts)** |

**Key findings:**
- F1 and F3 are **merged to main** — gc and auto-cleanup are live.
- **CG-5 (blocking):** B42-F2 (skip drift alerts) — code on feature branch, **not merged to main**. The project still has 60+ health errors from orphaned worktree records and drift warnings on merged branches.
- **CG-6 (blocking):** B42-F4 (validate entity IDs) — code on feature branch with **merge conflicts** on `feature/FEAT-01KQG3V06KNZG-validate-worktree-entity-ids`. Display-ID validation is not operational on main.
- 7/11 ACs pass on main (AC-001, AC-002, AC-003, AC-006, AC-007, AC-010, AC-011). 4 ACs fail because code is not on main.

---

## Consolidated Conformance Gaps

| # | Batch | Feature | Type | Description | Severity |
|---|-------|---------|------|-------------|----------|
| CG-1 | B40 | F1 (parent_feature filter) | unmerged | Fix exists on feature branch, not merged to main | non-blocking |
| CG-2 | B40 | F2 (finish state propagation) | unmerged | Fix exists on feature branch, not merged to main | non-blocking |
| CG-3 | B41 | F2 (preserve approval on minor edits) | unmerged | Code on feature branch, not merged to main | **blocking** |
| CG-4 | B41 | F4 (verification on entity update) | unmerged | Code on feature branch, not merged to main | **blocking** |
| CG-5 | B42 | F2 (skip drift alerts) | unmerged | Code on feature branch, not merged to main | **blocking** |
| CG-6 | B42 | F4 (validate entity IDs) | unmerged+conflicts | Code on feature branch with merge conflicts, not on main | **blocking** |

### Severity Rationale

**Non-blocking (B40):** The parent_feature filter and finish() state propagation fixes address agent-facing correctness issues that are significant but have existing workarounds. The batch spec treats these as Theme 2 (Significant). Furthermore, the parent_feature filter fix was already partially implemented in a prior commit (8167b8ea) per the design document's "already done" note.

**Blocking (B41, B42):** These features are core to the batch's stated purpose. Document lifecycle integrity (C2, C4) and cleanup automation (D2, D4) cannot be considered delivered if the code isn't on `main`. Additionally, B42-F4 has merge conflicts that need resolution.

## Documentation Currency

### AGENTS.md
**NOT CURRENT (pre-existing).** The Scope Guard in AGENTS.md does not mention P40 or its four batches (B39–B42). This is a pre-existing pattern — the health check shows ~35+ plans/batches from B3–B38 all missing from AGENTS.md. Not a P40-specific gap.

### Workflow skills
- `implement-task/SKILL.md` — **CURRENT.** Updated to document `write_file(entity_id)` as primary worktree pattern. This is the key documentation deliverable from B39.
- `.agents/skills/kanbanzai-*` — **STALE (pre-existing).** Multiple stale tool references exist across `kanbanzai-agents`, `kanbanzai-documents`, `kanbanzai-getting-started`, `kanbanzai-workflow`, and `kanbanzai-plan-review` SKILL.md files. These are pre-existing documentation gaps unrelated to P40.

### Knowledge entries
No knowledge entries were contributed during P40's execution (per retro synthesis results). This is expected for a batch focused on code fixes rather than propagating new patterns.

## Cross-Cutting Observations

### Override hygiene
All 14 features used `reviewing→done` overrides with documented reasons. The override patterns fall into three categories:
1. **Feature was already implemented** (B39-F1, B41-F3) — code existed before the feature entity was created; the feature verified existing behaviour.
2. **State repair** (B40-F1, B40-F2, B41-F2, B41-F4, B42-F2, B42-F4) — code exists on a branch but the lifecycle was never advanced through reviewing.
3. **Standard delivery** (B39-F2, B40-F3, B40-F4, B41-F1, B42-F1, B42-F3) — tasks done, tests pass, overrides used to skip the human review gate.

The high override count is consistent with a retrospective batch where many fixes were implemented rapidly. However, pattern #2 (code on branch but not merged) is the root cause of the blocking conformance gaps.

### Health check observations
- **4 orphaned worktree records** exist (WT-01KQAGN5SVNGZ, WT-01KQEX083D5Y4, WT-01KQEX7T3X5XD, WT-01KQF7Y0YQS36) — pre-existing. B42-F1 (gc) provides the tool to clean these up, but it hasn't been run yet.
- **Multiple worktree_branch_merged warnings** — branches that are merged to main but whose worktrees remain marked `active`. B42-F3 (auto-cleanup) should prevent this going forward but existing entries need manual cleanup.
- **B42-F4 branch has merge conflicts** with main — will need resolution before merge.

### What's on main vs what's not

| Feature | On main? | Notes |
|---------|----------|-------|
| B39-F1: write_file documentation | ✅ | Skill file changes applied to main |
| B39-F2: edit_file worktree-aware | ✅ | Code on main |
| B40-F1: parent_feature filter | ❌ | Branch only |
| B40-F2: finish() propagation | ❌ | Branch only |
| B40-F3: atomic multi-edit | ✅ | On main |
| B40-F4: decompose AC formats | ✅ | On main |
| B41-F1: auto-infer doc owner | ✅ | On main |
| B41-F2: preserve approval | ❌ | Branch only |
| B41-F3: bypassable field | ✅ | On main (from P31) |
| B41-F4: verification on update | ❌ | Branch only |
| B42-F1: worktree gc | ✅ | On main |
| B42-F2: skip drift alerts | ❌ | Branch only |
| B42-F3: auto-cleanup on merge | ✅ | On main |
| B42-F4: validate entity IDs | ❌ | Branch (merge conflicts) |

## Retrospective Summary

The retro synthesis for P40 returned zero signals — no workflow friction, tool gaps, or spec ambiguities were recorded via `finish(retrospective: ...)` during execution. This is likely because the tasks were completed in a rapid batch execution mode where retro signal contribution was not prioritised.

Notable workflow observations from the review process:
- **High override rate:** 14/14 features used lifecycle overrides. While individually justified, this pattern suggests the lifecycle validation may be too strict for retrospective/fix batches, or the review step is being bypassed as a workflow optimisation.
- **Unmerged branches:** Several branches remain unmerged despite features being marked `done`. This creates a disconnect between lifecycle status and operational availability.
- **Merge conflicts on B42-F4:** The entity ID validation feature has merge conflicts that were not addressed during the batch's execution.

## Batch Verdict

**FAIL** — 4 blocking conformance gaps across 2 batches (B41, B42), plus 2 non-blocking gaps in B40.

The batch delivers significant value — 10 of 14 features are on main, and the worktree development experience (B39), atomic multi-edit (B40-F3), decompose AC formats (B40-F4), auto-infer doc owner (B41-F1), bypassable gate field (B41-F3), worktree gc (B42-F1), and auto-cleanup (B42-F3) are all operational. However, 4 features across B41 and B42 are not yet delivered to main, which is a blocking conformance issue for a batch whose purpose is delivering these fixes.

### Remediation steps

1. **Merge B40-F1 and B40-F2 branches** to main (non-blocking, but recommended for completeness).
2. **Merge B41-F2 branch** to main (resolves CG-3).
3. **Merge B41-F4 branch** to main (resolves CG-4).
4. **Merge B42-F2 branch** to main (resolves CG-5).
5. **Resolve merge conflicts and merge B42-F4 branch** to main (resolves CG-6).
6. **Run `worktree(action: gc)`** to clean up orphaned worktree records using the newly delivered gc feature.
7. **Run `cleanup(action: execute)`** on merged-but-active worktrees for features whose branches are already on main.

## Evidence

1. **Specifications:** All 4 batch specs approved (B39, B40, B41, B42).
2. **Design:** `P40-retro-batch-april-2026/design-p40-design-retro-batch-improvements` — approved.
3. **Feature entity records:** All 14 features confirmed `done` via `entity(action: get)`.
4. **Task completion:** All 24 tasks confirmed terminal.
5. **Health check:** `health()` — 1 error (P40 batch status "idea"), multiple pre-existing warnings.
6. **Retro synthesis:** `retro(action: "synthesise", scope: "P40")` — zero signals.
7. **Per-batch reviews:**
   - B39 — PASS (full conformance)
   - B40 — PASS-WITH-NOTES (2 non-blocking gaps)
   - B41 — FAIL (2 blocking gaps)
   - B42 — FAIL (2 blocking gaps)
8. **Branch status:** Confirmed via `git branch --merged main` analysis.
