# Implementation Plan: Agent Instructions Consolidation

| Field | Value |
|-------|-------|
| **Feature** | FEAT-01KN6-ZA4NYJ25 (agent-instructions-consolidation) |
| **Specification** | `work/spec/agent-instructions-consolidation.md` |
| **Design** | `work/design/agent-instructions-consolidation.md` |
| **Review Report** | `work/reports/agent-instructions-review.md` |
| **Status** | Draft |

## Scope

This plan implements the requirements defined in `work/spec/agent-instructions-consolidation.md`. It covers all 38 functional requirements (FR-001 through FR-038) and 6 non-functional requirements (NFR-001 through NFR-006) across 5 work areas. It does not cover the evaluation harness (R17, out of scope per spec), cross-platform bootstrap files beyond `CLAUDE.md`, or changes to MCP tool descriptions.

The feature is content-focused — the deliverables are edited skill files, edited instruction files, one new bootstrap file, and at most minor fixes to Go code if verification reveals issues. The plan decomposes into 14 tasks with explicit dependency ordering and parallelism opportunities.

---

## Task Breakdown

### Task 1: Migrate Code Review Content — review-code

- **Description:** Port the four edge case playbooks (missing spec, partial implementation, ambiguous conformance, missing context) from `kanbanzai-code-review` into `review-code`. Expand the existing terse STOP instructions into multi-step procedural guidance. Verify per-dimension evaluation questions exist in `reviewer-*.yaml` role files; port any missing questions.
- **Deliverable:** Modified `.kbz/skills/review-code/SKILL.md` with edge case subsections in the Procedure section. Possibly modified `reviewer-*.yaml` role files if evaluation questions are missing.
- **Depends on:** None.
- **Effort:** Medium.
- **Spec requirements:** FR-001 (edge case playbooks), FR-006 (per-dimension evaluation questions).
- **Source material:** `kanbanzai-code-review` lines referenced in review report Appendix B.1. Read the old skill fully before editing the new one.
- **Constraints:** `review-code` must remain under 500 lines (NFR-001). If additions push past 500 lines, move edge case detail into `references/edge-cases.md`. Migrated content must use `review-code` vocabulary — replace "issue" with "finding", etc. (NFR-003). Place new content within the correct attention-curve section (NFR-004).

### Task 2: Migrate Code Review Content — orchestrate-review

- **Description:** Port the remediation phase (task creation, conflict-check, re-review scoping, escalation cycle with iteration cap), review document creation (naming convention, doc registration), human checkpoint integration (3 trigger scenarios), and context budget strategy from `kanbanzai-code-review` into `orchestrate-review`.
- **Deliverable:** Modified `.kbz/skills/orchestrate-review/SKILL.md` with remediation steps, document creation step, checkpoint guidance, and context budget note. Possibly a new `references/context-budget.md` if inline addition exceeds the line budget.
- **Depends on:** None.
- **Effort:** Medium.
- **Spec requirements:** FR-002 (remediation phase), FR-003 (review document creation), FR-004 (human checkpoint integration), FR-005 (context budget strategy).
- **Source material:** `kanbanzai-code-review` lines referenced in review report Appendix B.1.
- **Constraints:** `orchestrate-review` must remain under 500 lines (NFR-001). Use `orchestrate-review` vocabulary terms (NFR-003). Place content in correct attention-curve sections (NFR-004).

### Task 3: Retire kanbanzai-code-review

- **Description:** Delete `.agents/skills/kanbanzai-code-review/` directory. Scan for stale references in active instruction and routing files and update them.
- **Deliverable:** Deleted directory. Updated references in any files that pointed to the old skill.
- **Depends on:** Task 1, Task 2.
- **Effort:** Small.
- **Spec requirements:** FR-007 (retire kanbanzai-code-review).
- **Verification:** Run `grep -r "kanbanzai-code-review" AGENTS.md .agents/skills/ .kbz/ refs/ .github/` and confirm no active instruction or routing file references remain. Research reports and review documents are excluded.

### Task 4: Migrate Plan Review Content — review-plan

- **Description:** Port criterion-by-criterion spec conformance (read acceptance criteria, verify against implementation code), cross-cutting checks (`go test -race`, `health()`, `git status`), retrospective contribution step, document registration step, spec conformance detail table in output format, and inputs/prerequisites list from `kanbanzai-plan-review` into `review-plan`.
- **Deliverable:** Modified `.kbz/skills/review-plan/SKILL.md` with new procedure steps, updated output format, and prerequisites section.
- **Depends on:** None.
- **Effort:** Medium.
- **Spec requirements:** FR-008 (spec conformance), FR-009 (cross-cutting checks), FR-010 (retrospective and doc registration), FR-011 (inputs/prerequisites).
- **Source material:** `kanbanzai-plan-review` lines referenced in review report Appendix B.2.
- **Constraints:** `review-plan` must remain under 500 lines (NFR-001). Use `review-plan` vocabulary (NFR-003). Attention-curve section ordering (NFR-004).

### Task 5: Retire kanbanzai-plan-review

- **Description:** Delete `.agents/skills/kanbanzai-plan-review/` directory. Scan and update stale references.
- **Deliverable:** Deleted directory. Updated references.
- **Depends on:** Task 4.
- **Effort:** Small.
- **Spec requirements:** FR-012 (retire kanbanzai-plan-review).
- **Verification:** Run `grep -r "kanbanzai-plan-review" AGENTS.md .agents/skills/ .kbz/ refs/ .github/` and confirm no active references remain.

### Task 6: Migrate Design Content — write-design

- **Description:** Port "Design with Ambition" stance, human/agent role contract, iterative process framing, 3-tier risk escalation protocol, six-quality evaluation lens (as a reference file), draft lifecycle management, design splitting guidance, gotchas (registration, content hash drift, `doc refresh`), and next-steps-after-design handoff from `kanbanzai-design` into `write-design`.
- **Deliverable:** Modified `.kbz/skills/write-design/SKILL.md` with new preamble, risk escalation, and operational guidance sections. New `.kbz/skills/write-design/references/design-quality.md` containing the six-quality lens ported from `kanbanzai-design/references/design-quality.md`.
- **Depends on:** None.
- **Effort:** Medium.
- **Spec requirements:** FR-013 (stance and philosophy), FR-014 (risk escalation), FR-015 (design quality lens reference), FR-016 (operational guidance).
- **Source material:** `kanbanzai-design` and its `references/design-quality.md`, referenced in review report Appendix B.3.
- **Constraints:** `write-design` must remain under 500 lines — use `references/` for overflow (NFR-001). Reference file must link directly from SKILL.md, one level deep (CONVENTIONS.md). Use `write-design` vocabulary (NFR-003). Attention-curve ordering (NFR-004).

### Task 7: Retire kanbanzai-design

- **Description:** Delete `.agents/skills/kanbanzai-design/` directory including its `references/` subdirectory. Scan and update stale references.
- **Deliverable:** Deleted directory. Updated references.
- **Depends on:** Task 6.
- **Effort:** Small.
- **Spec requirements:** FR-017 (retire kanbanzai-design).
- **Verification:** Run `grep -r "kanbanzai-design" AGENTS.md .agents/skills/ .kbz/ refs/ .github/` and confirm no active references remain.

### Task 8: Fix Store Discipline

- **Description:** Add explicit `.kbz/state/` store-commit discipline to four files: (1) `kanbanzai-agents` commit discipline section, (2) `kanbanzai-getting-started` pre-task checklist, (3) `AGENTS.md` Git Discipline section, (4) `.kbz/roles/base.yaml` as a new "Store Neglect" anti-pattern with Detect/BECAUSE/Resolve.
- **Deliverable:** Modified `.agents/skills/kanbanzai-agents/SKILL.md`, `.agents/skills/kanbanzai-getting-started/SKILL.md`, `AGENTS.md`, and `.kbz/roles/base.yaml`.
- **Depends on:** None. (Best executed before Tasks 9–10 to avoid merge friction on shared files, but no hard dependency.)
- **Effort:** Small.
- **Spec requirements:** FR-024 (kanbanzai-agents), FR-025 (kanbanzai-getting-started), FR-026 (AGENTS.md), FR-027 (base.yaml Store Neglect anti-pattern).
- **Constraints:** The base.yaml anti-pattern must follow the existing YAML format (`name`, `detect`, `because`, `resolve` fields). The BECAUSE clause must reference parallel-agent failure modes (NFR-005).

### Task 9: Upgrade System Skills — Batch 1 (getting-started, workflow, planning)

- **Description:** Upgrade three system skills to match structural conventions: add vocabulary section (5–15 terms), convert prose anti-patterns to Detect/BECAUSE/Resolve format (≥3 per skill), add evaluation criteria (3–5 gradable questions), add "Questions This Skill Answers" retrieval anchors (5–8 per skill). Tighten existing prose to stay within 350-line budget. Preserve Anthropic's native frontmatter format.
- **Deliverable:** Modified `.agents/skills/kanbanzai-getting-started/SKILL.md`, `.agents/skills/kanbanzai-workflow/SKILL.md`, `.agents/skills/kanbanzai-planning/SKILL.md`.
- **Depends on:** Task 8 (store discipline adds content to `kanbanzai-getting-started`; upgrade should build on that).
- **Effort:** Large.
- **Spec requirements:** FR-018 (vocabulary), FR-019 (anti-patterns), FR-020 (evaluation criteria), FR-021 (retrieval anchors), FR-022 (350-line budget), FR-023 (preserve frontmatter).
- **Constraints:** Do NOT add Kanbanzai-specific frontmatter fields (`description.expert`, `triggers`, `roles`, `stage`, `constraint_level`). Vocabulary terms must be specific to each skill's domain — minimal duplication across the five skills. BECAUSE clauses must explain consequence chains (NFR-005). Each skill ≤ 350 lines.

### Task 10: Upgrade System Skills — Batch 2 (agents, documents)

- **Description:** Same structural upgrade as Task 9 but for `kanbanzai-agents` and `kanbanzai-documents`. Add vocabulary, convert anti-patterns, add evaluation criteria, add retrieval anchors. Tighten prose to stay within 350 lines. Preserve native frontmatter.
- **Deliverable:** Modified `.agents/skills/kanbanzai-agents/SKILL.md`, `.agents/skills/kanbanzai-documents/SKILL.md`.
- **Depends on:** Task 8 (store discipline adds content to `kanbanzai-agents`; upgrade should build on that).
- **Effort:** Medium.
- **Spec requirements:** FR-018, FR-019, FR-020, FR-021, FR-022, FR-023 (same as Task 9, applied to 2 skills).
- **Constraints:** Same as Task 9.

### Task 11: Clean Up AGENTS.md

- **Description:** Apply six incremental fixes to `AGENTS.md`: (1) deduplicate pre-task checklist — remove system-level items already in `kanbanzai-getting-started`, (2) replace inline Document Reading Order with pointer to `refs/document-map.md`, (3) remove phase labels from Repository Structure annotations, (4) add 5-term mini-vocabulary near the top (stage binding, role, skill, lifecycle gate, context packet), (5) add pointer to `.kbz/skills/` and `.kbz/stage-bindings.yaml`, (6) restructure Decision-Making Rules to remove phase-numbered references.
- **Deliverable:** Modified `AGENTS.md`.
- **Depends on:** Task 8 (store discipline adds content to the Git Discipline section; this task should build on that, not conflict with it).
- **Effort:** Medium.
- **Spec requirements:** FR-028 (deduplicate checklist), FR-029 (replace document reading order), FR-030 (remove phase labels), FR-031 (mini-vocabulary), FR-032 (pointer to skills/bindings), FR-033 (restructure decisions).

### Task 12: Fix Discovery Paths and Stale Pointers

- **Description:** (1) Update `refs/document-map.md` to replace stale skill pointers: `kanbanzai-code-review` → `review-code`/`orchestrate-review`, `kanbanzai-plan-review` → `review-plan`, `kanbanzai-design` → `write-design`. Fix any other stale pointers. (2) Add stage bindings and `.kbz/skills/` discovery to `kanbanzai-getting-started`.
- **Deliverable:** Modified `refs/document-map.md`, modified `.agents/skills/kanbanzai-getting-started/SKILL.md`.
- **Depends on:** Task 3, Task 5, Task 7 (the retirements must be complete before pointers are updated, otherwise the old paths still exist and the pointer update is premature). Task 9 (getting-started upgrade should be complete so this task adds discovery on top of the structural upgrade, not alongside it).
- **Effort:** Small.
- **Spec requirements:** FR-034 (update document-map), FR-035 (getting-started discovery).
- **Verification:** `grep -r "kanbanzai-code-review\|kanbanzai-plan-review\|kanbanzai-design" refs/ .agents/skills/kanbanzai-getting-started/` returns no matches.

### Task 13: Create CLAUDE.md

- **Description:** Create a `CLAUDE.md` bootstrap file at the repository root for Claude Code users. Point to `AGENTS.md`, `.kbz/stage-bindings.yaml`. Include condensed role table, condensed skill table, and critical rules. Keep under 150 lines. Content mirrors `.github/copilot-instructions.md` but shorter since it's loaded every turn.
- **Deliverable:** New `CLAUDE.md` at repository root.
- **Depends on:** Task 11 (AGENTS.md cleanup should be complete so that `CLAUDE.md` points to the cleaned-up version, and the pointer content is consistent).
- **Effort:** Small.
- **Spec requirements:** FR-036 (create CLAUDE.md).
- **Constraints:** ≤ 150 lines. Must include all five required elements: AGENTS.md pointer, stage bindings pointer, condensed role table, condensed skill table, critical rules.

### Task 14: Verify Context Assembly

- **Description:** (1) Read `internal/context/assemble.go` and trace the ordering of assembled context packet elements. Document whether identity/constraints appear first, supporting material in the middle, and instructions/anchors last. If ordering is incorrect, fix it. (2) Trace the `handoff` tool's code path through context assembly and verify that effort budget metadata from `stage-bindings.yaml` is included in the assembled context packet. If missing, add it. Run `go test -race ./...` after any changes.
- **Deliverable:** Verification report (in commit message or PR description). Possibly modified `internal/context/assemble.go` with test coverage if fixes are needed.
- **Depends on:** None (can run at any time).
- **Effort:** Medium (investigation-heavy; small if no fix needed, medium if fixes required).
- **Spec requirements:** FR-037 (attention-curve ordering), FR-038 (effort budget inclusion), NFR-006 (no test disruption).

---

## Dependency Graph

```
Task 1  (review-code migration)         ─┐
Task 2  (orchestrate-review migration)   ─┤─→ Task 3  (retire kanbanzai-code-review) ──┐
                                          │                                              │
Task 4  (review-plan migration)          ─┤─→ Task 5  (retire kanbanzai-plan-review)  ──┤
                                          │                                              │
Task 6  (write-design migration)         ─┤─→ Task 7  (retire kanbanzai-design)       ──┤
                                          │                                              │
Task 8  (store discipline)               ─┤─→ Task 9  (upgrade batch 1)              ──┤
                                          │─→ Task 10 (upgrade batch 2)                 │
                                          │─→ Task 11 (AGENTS.md cleanup)   ──→ Task 13 │
                                          │                                  (CLAUDE.md) │
                                          │                                              │
Task 14 (verify context assembly)         │   (independent, any time)                    │
                                          │                                              │
                                          └───────────→ Task 12 (fix discovery & stale pointers)
                                                         (after Tasks 3,5,7,9)
```

**Parallel groups:**
- **Group A (migrations):** Tasks 1, 2, 4, 6 — all touch disjoint file sets, can run in parallel.
- **Group B (retirements):** Tasks 3, 5, 7 — can run in parallel after their respective migrations.
- **Group C (system upgrades):** Tasks 9, 10 — can run in parallel (disjoint skill files), but both depend on Task 8.
- **Group D (independent):** Task 14 — no dependencies, can run at any time.

**Critical path:** Task 8 → Task 9 → Task 12 → (done)

**Recommended execution order:**

1. **Wave 1 (parallel):** Tasks 1, 2, 4, 6, 8, 14 — migrations, store discipline, and context assembly verification all start simultaneously.
2. **Wave 2 (parallel):** Tasks 3, 5, 7, 9, 10, 11 — retirements (after migrations), system upgrades (after store discipline), and AGENTS.md cleanup (after store discipline).
3. **Wave 3 (parallel):** Tasks 12, 13 — discovery fixes (after retirements + getting-started upgrade) and CLAUDE.md (after AGENTS.md cleanup).

---

## Interface Contracts

This feature has no code interfaces in the traditional sense — it is content-focused. The "interfaces" are the structural contracts that migrated and upgraded content must satisfy:

1. **CONVENTIONS.md compliance:** All `.kbz/skills/` content follows the section ordering, vocabulary format, anti-pattern format, example format, evaluation criteria format, and line budget defined in `.kbz/skills/CONVENTIONS.md`.

2. **System skill frontmatter contract:** `.agents/skills/` system skills use Anthropic's native format (`name` and `description` only). Kanbanzai-specific frontmatter fields must not be added.

3. **Reference file linking:** All reference files are linked one level deep from SKILL.md. Never from another reference file.

4. **Vocabulary consistency:** Migrated content adopts the target skill's vocabulary terms. No synonym drift.

5. **Store discipline wording:** All four store discipline additions use consistent terminology ("versioned project state", "not ephemeral cache", "commit alongside code changes").

---

## Risk Assessment

### Risk: Content Loss During Migration

- **Probability:** Medium.
- **Impact:** Medium — lost procedural guidance may not be noticed until an agent encounters the edge case.
- **Mitigation:** Each migration task requires reading the old skill fully before editing the new one. The commit message for each migration must reference the review report's Appendix B checklist and note which items were ported vs. dropped. The verification plan's Step 1 (migration completeness audit) catches omissions.
- **Affected tasks:** Tasks 1, 2, 4, 6.

### Risk: System Skill Line Budget Exceeded

- **Probability:** Medium — `kanbanzai-agents` is already 330 lines and receives both store discipline content (Task 8) and structural upgrade content (Task 10).
- **Impact:** Low — the mitigation is straightforward (tighten prose).
- **Mitigation:** Task 8 (store discipline) executes before Task 10 (upgrade). Task 10 starts with a line count and budgets the available headroom before adding content. If 350 lines is insufficient, tighten existing prose first — do not drop structural content.
- **Affected tasks:** Tasks 9, 10.

### Risk: Discovery Regression After Retirement

- **Probability:** Low — the discovery path through `copilot-instructions.md` → stage bindings → `.kbz/skills/` is already functional.
- **Impact:** High — agents unable to find review or design guidance would produce lower-quality output.
- **Mitigation:** Task 12 explicitly updates all routing files. The verification plan's Step 4 (stale reference scan) catches missed pointers. Task 13 (`CLAUDE.md`) creates a new discovery entry point.
- **Affected tasks:** Tasks 3, 5, 7, 12.

### Risk: Context Assembly Already Correct (Wasted Effort)

- **Probability:** High — the assembly pipeline was designed with attention-curve awareness.
- **Impact:** None (positive outcome — verification confirms correctness with no rework).
- **Mitigation:** Task 14 is scoped as verification-first. If the ordering is correct and effort budgets are present, the task closes with a verification note and no code changes.
- **Affected tasks:** Task 14.

---

## Verification Approach

| Acceptance Criterion | Verification Method | Producing Task |
|---------------------|--------------------| ---------------|
| Three retired directories no longer exist | `ls` / `find` — confirm deletion | Tasks 3, 5, 7 |
| All unique content migrated per Appendix B checklists | Manual audit — checklist walkthrough per migration | Tasks 1, 2, 4, 6 |
| Five system skills have vocabulary, anti-patterns, eval criteria, anchors | `grep` for section headings + manual content review | Tasks 9, 10 |
| Store discipline in 4 files | `grep -r "kbz/state"` across the 4 files | Task 8 |
| AGENTS.md cleaned up (no duplication, no phase labels, has pointers) | Manual review of AGENTS.md | Task 11 |
| `refs/document-map.md` has no stale skill pointers | `grep` for retired skill names | Task 12 |
| `kanbanzai-getting-started` mentions stage bindings and `.kbz/skills/` | `grep` for `stage-bindings` and `.kbz/skills` | Task 12 |
| `CLAUDE.md` exists and is ≤ 150 lines | `wc -l CLAUDE.md` | Task 13 |
| Context assembly follows attention curve | Code trace documented in commit/PR | Task 14 |
| Effort budgets included in handoff packets | Code trace documented in commit/PR | Task 14 |
| All Go tests pass | `go test -race ./...` | Task 14 (after any code changes) |
| No `.kbz/skills/` file exceeds 500 lines | `wc -l .kbz/skills/*/SKILL.md` | Tasks 1, 2, 4, 6 |
| No system skill exceeds 350 lines | `wc -l .agents/skills/*/SKILL.md` | Tasks 9, 10 |

---

## Traceability Matrix

| Spec Requirement | Task(s) |
|-----------------|---------|
| FR-001 | Task 1 |
| FR-002 | Task 2 |
| FR-003 | Task 2 |
| FR-004 | Task 2 |
| FR-005 | Task 2 |
| FR-006 | Task 1 |
| FR-007 | Task 3 |
| FR-008 | Task 4 |
| FR-009 | Task 4 |
| FR-010 | Task 4 |
| FR-011 | Task 4 |
| FR-012 | Task 5 |
| FR-013 | Task 6 |
| FR-014 | Task 6 |
| FR-015 | Task 6 |
| FR-016 | Task 6 |
| FR-017 | Task 7 |
| FR-018 | Tasks 9, 10 |
| FR-019 | Tasks 9, 10 |
| FR-020 | Tasks 9, 10 |
| FR-021 | Tasks 9, 10 |
| FR-022 | Tasks 9, 10 |
| FR-023 | Tasks 9, 10 |
| FR-024 | Task 8 |
| FR-025 | Task 8 |
| FR-026 | Task 8 |
| FR-027 | Task 8 |
| FR-028 | Task 11 |
| FR-029 | Task 11 |
| FR-030 | Task 11 |
| FR-031 | Task 11 |
| FR-032 | Task 11 |
| FR-033 | Task 11 |
| FR-034 | Task 12 |
| FR-035 | Task 12 |
| FR-036 | Task 13 |
| FR-037 | Task 14 |
| FR-038 | Task 14 |
| NFR-001 | Tasks 1, 2, 4, 6, 9, 10 |
| NFR-002 | Tasks 1, 2, 4, 6 |
| NFR-003 | Tasks 1, 2, 4, 6 |
| NFR-004 | Tasks 1, 2, 4, 6 |
| NFR-005 | Tasks 8, 9, 10 |
| NFR-006 | Task 14 |