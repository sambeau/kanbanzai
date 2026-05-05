# Batch Conformance Review: P50 — Retrospective Fixes May 2026

## Scope

- **Plan:** P50-retro-may-2026 (Retrospective Fixes — May 2026)
- **Parent Batch:** B49-retro-fixes-may-2026 (informal — not registered as formal entity)
- **Features:** 5 total (1 still developing, 4 in reviewing)
- **Review date:** 2026-05-05
- **Reviewer:** orchestrator (4 specialist reviewers: conformance, quality, security, testing)

## Feature Census

| Feature | ID | Status | Spec Approved | Design Approved | Tasks | Notes |
|---------|-----|--------|---------------|-----------------|-------|-------|
| F1: Error Classification | FEAT-01KQTNYMZRT6V | `reviewing` | yes | yes | 4/4 done | All tasks complete. Verification passed. |
| F2: Decompose Proposal Quality | FEAT-01KQTNYN00M4P | `reviewing` | yes | yes | 4/5 done | **1 task queued** (TASK-01KQTX5DX2C1T). Override used to advance. |
| F3: Commit Discipline Prompts | FEAT-01KQTNYN01ZF8 | `reviewing` | yes | yes | 4/5 (1 not-planned) | Verification partial. |
| F4: Merge Discipline and DOD | FEAT-01KQTWFY52EG1 | `reviewing` | yes | yes | 4/4 done | Verification partial. Orphaned reviewing warning. |
| F5: Document Path Tool | FEAT-01KQTNYN00HZA | `developing` | — | — | — | **Not in reviewing — excluded from this review.** |

## Aggregate Verdict: **FAIL**

The P50 implementation has **10 blocking findings** across F2, F3, and F4. F3's `state_modified` pipeline is wired end-to-end but has zero production callers — the feature is dead code. F4's merge lifecycle (`merging` → `verifying` → `done`) is defined but not wired into the merge tool. F2 is missing the `paired=false` opt-out flag. Dual-write rules for skill file mirrors are broken across F3 and F4.

F1 (Error Classification) passes all review dimensions cleanly.

---

## Per-Dimension Verdicts

| Dimension | Verdict | Reviewer |
|-----------|---------|----------|
| spec_conformance | **fail** | reviewer-conformance |
| implementation_quality | **fail** | reviewer-quality |
| security | **pass** | reviewer-security |
| test_adequacy | **concern** | reviewer-testing |

---

## Blocking Findings

### BF-1: F2 — Missing `paired=false` opt-out flag (AC-004 / REQ-004)

- **Dimension:** spec_conformance
- **Spec anchor:** AC-004 — "when paired-test-task mode disabled, proposal contains one task per AC"
- **Location:** `internal/service/decompose.go` L14-17 (`DecomposeInput` has no `paired` field), `internal/mcp/decompose_tool.go` L75-94
- **Evidence:** Test at `decompose_test.go` L3387-3391 documents: "opt-out flag not yet implemented"
- **Impact:** Users cannot disable paired-task mode.

### BF-2: F3 — `SignalStateModified` never called from production code

- **Dimension:** implementation_quality, dead_code_detection
- **Spec anchor:** REQ-001 — state_modified must be true when .kbz/state/ is modified
- **Location:** `internal/mcp/sideeffect.go` L245 — zero non-test call sites
- **Evidence:** `SignalMutation` has 40+ call sites; `SignalStateModified` has zero. Pipeline fully wired but never triggered.
- **Impact:** `state_modified: true` never appears in any tool response. Feature is dead code.

### BF-3: F3 — Dual-write gap: `kanbanzai-agents` skill mirror

- **Dimension:** spec_conformance
- **Spec anchor:** AC-006 (REQ-006) — dual-write rule
- **Location:** `.agents/skills/kanbanzai-agents/SKILL.md` L54-57 (rule present); `internal/kbzinit/skills/agents/SKILL.md` L51-57 (rule absent)
- **Impact:** Consumer installs won't receive the state_modified commit discipline rule.

### BF-4: F3 — Dual-write gap: `kanbanzai-getting-started` skill mirror

- **Dimension:** spec_conformance
- **Spec anchor:** AC-006 (REQ-006) — dual-write rule
- **Location:** `.agents/skills/kanbanzai-getting-started/SKILL.md` (strengthened); `internal/kbzinit/skills/getting-started/SKILL.md` (not strengthened)
- **Impact:** Consumer installs won't receive the strengthened session-start git check.

### BF-5: F4 — `FeatureValidTransitions` never wired into enforcement

- **Dimension:** implementation_quality, dead_code_detection
- **Spec anchor:** REQ-001, REQ-002 — lifecycle transitions reviewing → merging → verifying → done
- **Location:** `internal/model/entities.go` L736-792 — zero non-test callers
- **Impact:** State machine defined but unenforced. Merging/verifying stages have no programmatic gate.

### BF-6: F4 — `executeMerge` bypasses `merging`/`verifying` stages

- **Dimension:** spec_conformance, implementation_quality
- **Spec anchor:** AC-001 through AC-005 — merging → verifying → done (or needs-rework on failure)
- **Location:** `internal/mcp/merge_tool.go` L507-530
- **Evidence:** After successful merge, transitions directly to `done`, skipping merging and verifying.
- **Impact:** Features marked done without post-merge verification — the exact gap P50 identified.

### BF-7: F4 — Worktree merge verification not on main

- **Dimension:** spec_conformance
- **Spec anchor:** AC-006 (REQ-006) — verify branch is ancestor before marking merged
- **Location:** `internal/mcp/merge_tool.go` L470-475 (unconditionally marks merged on main)
- **Evidence:** Worktree branch has correct `IsBranchAncestorOf` check; main does not.
- **Impact:** Worktrees marked merged even when branch not on main.

### BF-8: F4 — `kanbanzai-agents` skill missing merge prompt

- **Dimension:** spec_conformance
- **Spec anchor:** AC-008 (REQ-008) — agents skill must include merge prompt
- **Location:** `.agents/skills/kanbanzai-agents/SKILL.md` L106-120 (old 2-step guidance)
- **Impact:** Agents working without orchestrator won't know the merge lifecycle.

### BF-9: F4 — Missing `needs-rework` → `developing` transition

- **Dimension:** implementation_quality
- **Spec anchor:** REQ-001 — lifecycle integrity
- **Location:** `internal/model/entities.go` L781-785
- **Evidence:** `FeatureValidTransitions[FeatureStatusNeedsRework]` lacks `developing`. Existing tests expect this transition.
- **Impact:** Rework loops would break if state machine is enforced.

### BF-10: F4 — `executeMerge` auto-advance path untested

- **Dimension:** test_adequacy
- **Spec anchor:** AC-001 through AC-005 — merge lifecycle flow
- **Location:** `internal/mcp/merge_tool.go` L507-530 (untested); tests exercise lifecycle directly, never call `executeMerge`
- **Impact:** Mismatch between tested flow (reviewing→merging→verifying→done) and production flow (→done) invisible to tests.

---

## Non-Blocking Findings

| # | Feature | Dimension | Description | Location |
|---|---------|-----------|-------------|----------|
| NBF-1 | F1 | spec_conformance | ClassifyError uses keyword heuristics instead of sentinel errors. Refactoring could break classification. | `classify.go` L19-22 |
| NBF-2 | F2 | spec_conformance | generateProposal groups 2-4 ACs into single pair vs per-AC. Either document or add flag. | `decompose.go` L632-670 |
| NBF-3 | F2 | quality | Grouped path uses old `allCoversContainTest` vs new `isTestingConcern` in individual path. | `decompose.go` L666 vs L689 |
| NBF-4 | F2 | quality | `collapsePairedTasks` indexes map without ok check — zero value for missing deps. | `decompose.go` L1311 |
| NBF-5 | F2 | testing | No test for checkGaps with out-of-range Covers references. | `decompose_test.go` |
| NBF-6 | F2 | testing | ReviewProposal with nil proposal/nil Tasks untested. | `decompose_test.go` |
| NBF-7 | F2 | testing | Test task detection uses substring match on Summary vs slug-based. | `decompose_test.go` L380-386 |
| NBF-8 | F3 | quality | `buildResult(nil, nil, false, true)` yields awkward null+state_modified envelope. | `sideeffect.go` L341-397 |
| NBF-9 | F3 | quality | `buildResult` manages 4 orthogonal concerns — combinatorial risk. | `sideeffect.go` L315-397 |
| NBF-10 | F3 | quality | Manual string concat for JSON injection — fragile for dynamic fields. | `sideeffect.go` L376 |
| NBF-11 | F3 | testing | Table-driven mutation tests removed; real handler integration unclear. | `sideeffect_test.go` |
| NBF-12 | F4 | quality | `IsBranchAncestorOf` exported from health, only used by merge_tool. | `worktree_merged.go` L63 |
| NBF-13 | F4 | quality | MarkMerged modifies wt before Update — stale in-memory on failure. | `merge_tool.go` L492-496 |
| NBF-14 | F4 | testing | `mergeCommitFunc` package-level override — not goroutine-safe for parallel tests. | `merge_tool_test.go` L243-245 |
| NBF-15 | F4 | testing | No build-verification-failure path test in verifying status. | `merge_verify_done_test.go` |
| NBF-16 | F1 | testing | ClassifyError empty string → ErrorInternalError untested. | `classify_test.go` |
| NBF-17 | F1 | testing | ComputeTimePerStage median edge cases untested. | `metrics_test.go` L44-70 |
| NBF-18 | F1 | testing | Corrupted JSON in log file read path untested. | `hook_test.go` |
| NBF-19 | — | security | Git errors surfaced to MCP client without sanitization. Local-only, low risk. | `sideeffect.go` L263-267 |
| NBF-20 | — | security | buildErrorResult fallback includes raw err.Error(). | `sideeffect.go` L449 |
| NBF-21 | — | security | Branch names not validated before git command args. | `worktree_merged.go` L63 |

---

## Documentation Currency

- **AGENTS.md:** Current — no P50-specific changes needed
- **Workflow skills:**
  - `kanbanzai-agents`: Missing merge prompt (BF-8). state_modified rule missing from kbzinit mirror (BF-3).
  - `kanbanzai-getting-started`: Strengthened checklist missing from kbzinit mirror (BF-4).
  - `orchestrate-review`: Post-Review Merge present and dual-written correctly.
- **Knowledge entries:** Not assessed.

---

## Per-Feature Summary

### F1: Error Classification — ✅ APPROVED

All 7 ACs satisfied. `ClassifyError` + `Hook.Wrap` apply classification centrally. All 5 high-volume tools covered. 4 non-blocking test-coverage notes only.

### F2: Decompose Proposal Quality — ❌ REJECTED

1 blocking (AC-004: paired=false flag). Test task still queued. 4 non-blocking.

### F3: Commit Discipline Prompts — ❌ REJECTED

3 blocking: SignalStateModified dead code (BF-2), dual-write gaps (BF-3, BF-4). 4 non-blocking.

### F4: Merge Discipline and DOD — ❌ REJECTED

6 blocking: lifecycle not enforced (BF-5), merge bypasses stages (BF-6), worktree verification missing (BF-7), agents skill lacks merge prompt (BF-8), missing transition (BF-9), untested path (BF-10). 5 non-blocking.

---

## Remediation Plan

### Priority 1 — Critical

1. **F3 BF-2:** Wire `SignalStateModified` into all high-mutation tool handlers.
2. **F4 BF-6:** Update `executeMerge` to transition to `merging` not `done`. Add `verifying` with build/test.
3. **F4 BF-9:** Add `needs-rework` → `developing` transition.

### Priority 2 — High

4. **F4 BF-7:** Port ancestry verification from worktree to main.
5. **F4 BF-8:** Add merge prompt to kanbanzai-agents SKILL.md.
6. **F3 BF-3, BF-4:** Fix dual-write mirrors.

### Priority 3 — Medium

7. **F4 BF-10:** Add executeMerge integration test.
8. **F2 BF-1:** Implement paired=false flag.
9. **F2 task:** Complete queued decompose-quality-tests task.

### Priority 4 — Non-blocking

Address NBF-1 through NBF-21 as capacity permits.

---

## Review Panel

| Reviewer | Session | Verdict | Blocking | Non-Blocking |
|----------|---------|---------|----------|--------------|
| reviewer-conformance | 5c81570d | fail | 6 | 2 |
| reviewer-quality | ef2e308a | fail | 5 | 8 |
| reviewer-security | 58992d3b | pass | 0 | 3 |
| reviewer-testing | b0b6d758 | concern | 2 | 12 |

**Deduplicated totals:** 10 blocking, 21 non-blocking

---

## Evidence

- Feature statuses: `status(id:...)` for all 4 features
- Specifications: `doc(action: "content", id:...)` for all 4 specs
- Review sub-agents: 4 parallel specialist reviewers (sessions above)
- Code locations cited inline in each finding
