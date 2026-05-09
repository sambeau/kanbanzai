# Batch Conformance Review: B61 — Instruction Corpus Cleanup

## Scope
- **Batch:** B61-tidy-contradictions-stray-files-corpus-size
- **Feature reviewed:** FEAT-01KR3MES5ZJ11 (B61-F1) — Instruction corpus cleanup
- **Features in batch:** 1 total (1 in developing, 0 done, 0 cancelled)
- **Tasks:** 6 total (6 done, 0 active)
- **Review date:** 2026-05-08
- **Reviewer:** reviewer-conformance

## Feature Census

| Feature | Status | Spec | Dev-Plan | Tasks Done | Notes |
|---------|--------|------|----------|------------|-------|
| FEAT-01KR3MES5ZJ11 | developing | approved | approved | 6/6 | All tasks done; lifecycle not yet advanced past developing |

## Acceptance Criteria Verification

Each AC was verified against the feature branch
(`feature/FEAT-01KR3MES5ZJ11-instruction-corpus-cleanup`, worktree `WT-01KR3Q3NBMGCS`).

| AC | Requirement | Verdict | Evidence |
|----|-------------|---------|---------|
| AC-001 | `.kbz/skills/SKILL.md` absent; README present | ✅ PASS | `SKILL.md` deleted on feature branch (git says "exists on disk, but not in feature branch"). `README.md` present with table of all skill subdirectories. No `name:`, `triggers:`, or `## Procedure` markers — not executable as a skill. |
| AC-002 | `.agents/skills/SKILL.md` absent; README present | ✅ PASS | Same pattern — `SKILL.md` deleted on feature branch; `README.md` present with table of all agent skill subdirectories. No skill front-matter markers. |
| AC-003 | No top-level file instructs `git stash` | ✅ PASS | `.github/copilot-instructions.md:132`: "do not use `git stash`". `CLAUDE.md:54`: "do not use `git stash`". `AGENTS.md:138,140,195`: "do not stash". All three files consistent. |
| AC-004 | `orchestrate-development/SKILL.md` < 500 lines | ✅ PASS | 451 lines (down from 520). Procedure (Steps 1–7), Checklist, and Output Format sections all retained. |
| AC-005 | `kanbanzai-agents/SKILL.md` < 500 lines | ✅ PASS | 482 lines (down from 501). Dispatch, Commit, Finish, and Knowledge-contribution sections all retained. Embedded copy at `internal/kbzinit/skills/agents/SKILL.md` updated to 458 lines. |
| AC-006 | Examples > 60 lines moved to reference files | ✅ PASS | 11 skill files had `## Examples` sections relocated to `references/examples-*.md`. Source skills link to reference files with relative paths. Reference files preserve example content (heading normalisation only). |
| AC-007 | Duplicate anti-pattern prose replaced by cross-references | ✅ PASS | `orchestrate-development/SKILL.md:92`: cross-reference to `orchestrator` role. `review-code/SKILL.md:57-59`: blockquote reference to `reviewer` role. No verbatim or near-verbatim anti-pattern prose remains in skill files that duplicates role YAML. |
| AC-008 | `grep`/`search_graph` absent from base; needing roles list explicitly | ✅ PASS | `base.yaml` tools: `[entity, doc]` only — no `grep` or `search_graph`. All 15 roles that need code-search tools list them explicitly. `reviewer-security` inherits from `reviewer` (which lists both). Four roles that don't need them (`orchestrator`, `spec-validator`, `doc-copyeditor`, `doc-stylist`) confirmed against their skill procedures. |
| AC-009 | Corpus ≤ 6,500 lines or exception | ⚠️ EXCEPTION | 9,571 lines (target: 6,500). 17.7% reduction (2,053 lines saved). Exception documented in corpus report with rationale: further reduction requires content-authoring work (procedure condensation), which was out of scope for this mechanical-cleanup feature. |
| AC-010 | B59 hard workflow invariants discoverable | ✅ PASS | All 5 invariants (handoff-only dispatch, registered-entity requirement, commit-orphaned-state, no-shell-reads-of-state, artefact-gate-enforcement) traced to specific line numbers in post-cleanup skill files. |
| AC-011 | No stage-bound role lost required tools | ✅ PASS | T4 audit verified each role against its skill's tool requirements. `reviewer-security` inherits both tools from `reviewer`. `orchestrator` delegates via `handoff`/`spawn_agent` and does not require code-search tools. |

## Conformance Gaps

| # | Type | Description | Severity |
|---|------|-------------|----------|
| CG-1 | lifecycle-state | Feature `FEAT-01KR3MES5ZJ11` is in `developing` status despite all 6 tasks being `done`. The feature has not advanced through `reviewing` → `done`. Health check confirms: "feature FEAT-01KR3MES5ZJ11 has all 6 child task(s) in terminal state but feature is developing". | **blocking** |
| CG-2 | merge-status | Feature branch `feature/FEAT-01KR3MES5ZJ11-instruction-corpus-cleanup` is 87 commits behind main and has not been merged. All changes exist only on the feature branch — the corpus cleanup is not live on main. | **blocking** |
| CG-3 | spec-exception | AC-009 corpus size target (≤6,500 lines) was not met. Actual: 9,571 lines (1,577 lines saved from `.kbz/skills/`, 476 from `.agents/skills/`). Exception is documented and justified — the scope was mechanical cleanup only (deletion, de-duplication, relocation). Procedure condensation for remaining large skills requires content-authoring work out of scope for B61. | **non-blocking** |

## Documentation Currency

- **Spec:** `FEAT-01KR3MES5ZJ11/spec-b61-f1-spec-instruction-corpus-cleanup` — **approved** (2026-05-08, by sambeau)
- **Dev-plan:** `FEAT-01KR3MES5ZJ11/dev-plan-b61-f1-dev-plan` — **approved** (2026-05-08, by sambeau)
- **Corpus report:** `work/B61-tidy-contradictions-stray-files-corpus-size/B61-F1-corpus-report.md` — present on feature branch, not yet registered as a document record
- **Dual-write:** `internal/kbzinit/skills/agents/SKILL.md` updated (458 lines), diff shows examples change applied. No `stash` references in consumer copy (expected — project-specific git discipline rules live in project-local files only).
- **AGENTS.md scope guard:** B61 is not yet listed (expected — batch is still active and not yet merged).

## Health Check Findings

Relevant to this batch:
- ⚠️ `feature_child_consistency`: FEAT-01KR3MES5ZJ11 has all 6 child tasks in terminal state but feature is `developing` (matches CG-1)
- ⚠️ `branch`: WT-01KR3Q3NBMGCS branch is 87 commits behind main (matches CG-2)
- ⚠️ `estimation_coverage`: FEAT-01KR3MES5ZJ11 has 6 tasks with no estimates

No errors or warnings specific to B61 entities beyond the above.

## Retrospective Summary

Two signals surfaced from the B61 scope, both positive: (1) systematic comparison of skill anti-pattern sections against role YAML was straightforward — the Detect/Because/Resolve structure made near-verbatim comparison fast and unambiguous; (2) reconstructing the exact before-baseline from `git show` at the parent commit gave precise per-file line counts, making the before/after table fully verifiable rather than relying on the approximate P59 audit figure.

## Batch Verdict

**PASS WITH BLOCKING GAPS**

The implementation on the feature branch satisfies all 11 acceptance criteria (AC-009 with documented exception). The changes are mechanical, reviewable, and preserve all hard workflow invariants. The corpus report is thorough and evidence-backed.

However, two blocking gaps prevent this from being a clean pass:

1. **CG-1 (lifecycle):** The feature must be advanced from `developing` → `reviewing` → `done` before the batch can be considered deliverable.
2. **CG-2 (merge):** The feature branch must be merged to main (or at minimum rebased) before the cleanup changes are live in the project.

These are lifecycle and merge gaps — the implementation quality is sound.

## Evidence

- Feature entity: `entity(action: "get", id: "FEAT-01KR3MES5ZJ11")` → status: developing, 6/6 tasks done
- Task list: `entity(action: "list", type: "task", parent_feature: "FEAT-01KR3MES5ZJ11")` → 6 tasks, all done
- Health check: `health()` — feature_child_consistency warning, branch drift warning
- Spec document: `doc(action: "get", id: "FEAT-01KR3MES5ZJ11/spec-b61-f1-spec-instruction-corpus-cleanup")` → approved
- Dev-plan document: `doc(action: "get", id: "FEAT-01KR3MES5ZJ11/dev-plan-b61-f1-dev-plan")` → approved
- Corpus report: `work/B61-tidy-contradictions-stray-files-corpus-size/B61-F1-corpus-report.md` → present on feature branch
- Retro synthesis: `retro(action: "synthesise", scope: "B61-tidy-contradictions-stray-files-corpus-size")` → 2 signals (both worked-well)
- File verification: All ACs verified via `git show` against feature branch HEAD
