# User Guide: Development Plan

| Document | User Guide Development Plan |
|----------|----------------------------|
| Status   | Draft |
| Feature  | FEAT-01KP8-T4HQMEA3 |
| Created  | 2026-04-15 |
| Spec     | `work/spec/user-guide.md` (FEAT-01KP8T4HQMEA3/specification-user-guide) |
| Design   | `work/design/public-release-documentation.md` §6 |
| Pipeline | `work/design/documentation-pipeline.md` |

---

## Scope

This plan implements the requirements defined in `work/spec/user-guide.md`. It covers the production of the User Guide (`docs/user-guide.md`) through the five-stage editorial pipeline defined in `work/design/documentation-pipeline.md`.

The plan produces one document: `docs/user-guide.md`. It does not produce any other documents in the P22 documentation set — those are covered by their own features and plans.

**Production methodology:** The `doc-publishing` stage binding orchestrates five sequential sub-agent stages: Write → Edit → Check → Style → Copyedit. Each stage has a dedicated role, skill, and non-overlapping editorial scope. The User Guide uses advisory checkpoints after Edit and Copyedit (not hard gates).

**Stage binding:** `doc-publishing` (pipeline-coordinator orchestration, sequential topology).

---

## Task Breakdown

### Task 1: Write — Produce First Draft

- **Description:** Produce a structured first draft of the User Guide following the inverted pyramid, using the design brief (§6) as the content specification. The draft must contain all 13 sections defined in FR-001 through FR-013, open with a purpose and audience statement (FR-015), and include placeholder links to all 7 cross-referenced documents (FR-017, FR-018).
- **Deliverable:** `docs/user-guide.md` — complete first draft with all sections, cross-references, and structural skeleton.
- **Depends on:** None.
- **Effort:** Large — this is the primary content creation task.
- **Spec requirements:** FR-001 through FR-018 (content, structure, and cross-references). NFR-001, NFR-002 (audience, tone).
- **Role:** `documenter`
- **Skill:** `write-docs`
- **Inputs:** Design brief (`work/design/public-release-documentation.md` §6), specification (`work/spec/user-guide.md`), current codebase for factual grounding.

### Task 2: Edit — Structural Editing

- **Description:** Verify and improve the document's structure, scannability, tone gradient, and audience fit. Check inverted-pyramid compliance at document and section level. Verify heading skeleton is descriptive (FR-016). Verify the exclusion requirements (FR-019 through FR-022) — flag any content that belongs in other documents. Produce a changelog of structural changes.
- **Deliverable:** Revised `docs/user-guide.md` with structural improvements applied. Editorial changelog documenting all changes.
- **Depends on:** Task 1 (Write).
- **Effort:** Medium.
- **Spec requirements:** FR-014 (inverted pyramid), FR-015 (purpose/audience opener), FR-016 (descriptive headings), FR-019 through FR-022 (exclusions). NFR-003 (scannability), NFR-007 (reading time).
- **Role:** `doc-editor`
- **Skill:** `edit-docs`
- **Advisory checkpoint:** After this stage, present the structural edit to the human for optional review before proceeding to Check.

### Task 3: Check — Fact Verification

- **Description:** Verify every factual claim in the document against the current Kanbanzai implementation. Test any code examples or command references. Flag hallucinations, inaccurate descriptions, outdated information, and vague claims that lack specificity. Produce a structured findings report classified by severity.
- **Deliverable:** Check stage findings report (structured list of verified claims, flagged issues, and corrections applied). Updated `docs/user-guide.md` with factual corrections.
- **Depends on:** Task 2 (Edit).
- **Effort:** Medium — requires cross-referencing claims against the codebase.
- **Spec requirements:** NFR-005 (factual accuracy against implementation). AC-019 (zero unresolved error/hallucination findings).
- **Role:** `doc-checker`
- **Skill:** `check-docs`
- **Inputs:** Current codebase, MCP tool definitions, entity schemas, configuration files.

### Task 4: Style — AI Artifact Removal

- **Description:** Hunt and eliminate AI writing patterns: banned vocabulary, inflated adjectives, faux-insider phrases, staccato rhetoric, hedging, tricolon overuse, rigid paragraph formulas, robotic transitions, elegant variation, and copula avoidance. Rewrite affected sentences while preserving meaning. Produce a findings report and changelog.
- **Deliverable:** Style stage findings report (patterns found and resolved). Updated `docs/user-guide.md` with AI artifacts removed.
- **Depends on:** Task 3 (Check).
- **Effort:** Medium.
- **Spec requirements:** NFR-002 (plain, direct prose), NFR-006 (no AI writing artifacts). AC-020 (zero unresolved style findings).
- **Role:** `doc-stylist`
- **Skill:** `style-docs`
- **Inputs:** `refs/humanising-ai-prose.md` (primary reference for banned patterns).

### Task 5: Copyedit — Sentence-Level Polish

- **Description:** Polish sentences for clarity: correct passive voice, fix smothered verbs, control sentence length, enforce punctuation rules, check capitalisation, verify parallel structure, ensure consistency, and test reading rhythm. Produce a changelog of copyediting changes.
- **Deliverable:** Final `docs/user-guide.md` ready for publication. Copyedit changelog documenting all sentence-level changes.
- **Depends on:** Task 4 (Style).
- **Effort:** Small to medium.
- **Spec requirements:** NFR-002 (active voice, present tense), NFR-006 (no AI artifacts at sentence level). AC-018 (all 5 pipeline stages completed with changelogs).
- **Role:** `doc-copyeditor`
- **Skill:** `copyedit-docs`
- **Inputs:** `refs/punctuation-guide.md`, `refs/technical-writing-guide.md` (sentence-level sections).
- **Advisory checkpoint:** After this stage, present the final document to the human for optional review before marking the feature complete.

---

## Dependency Graph

```
Task 1: Write       (no dependencies)
Task 2: Edit      → depends on Task 1
Task 3: Check     → depends on Task 2
Task 4: Style     → depends on Task 3
Task 5: Copyedit  → depends on Task 4
```

Parallel groups: None — the editorial pipeline is strictly sequential by design. Each stage trusts that the previous stage has done its job. There is no parallelism within a single document's pipeline.

Critical path: Task 1 → Task 2 → Task 3 → Task 4 → Task 5 (all tasks are on the critical path).

---

## Risk Assessment

### Risk: First Draft Exceeds Reading Time Budget

- **Probability:** Medium — the design brief specifies 13 sections, which could expand beyond the NFR-007 target of 15 minutes.
- **Impact:** Low — the Edit stage (Task 2) is designed to catch and correct structural bloat.
- **Mitigation:** The Write stage skill enforces orientation depth (one to three paragraphs per section). The Edit stage checks scannability and reading time.
- **Affected tasks:** Task 1 (Write), Task 2 (Edit).

### Risk: Factual Claims Outdated by Concurrent Development

- **Probability:** Low — the design document's Dependencies section notes the implementation should be stable during document production.
- **Impact:** Medium — incorrect claims undermine credibility and require rework.
- **Mitigation:** The Check stage (Task 3) verifies all claims against the current codebase. If the implementation changes after Check, a re-check can be run on the affected sections.
- **Affected tasks:** Task 3 (Check).

### Risk: AI Writing Patterns Persist After Style Stage

- **Probability:** Low — the Style skill has a comprehensive banned-word list and pattern detection procedure.
- **Impact:** Low — residual patterns are cosmetic, not factual errors.
- **Mitigation:** The Copyedit stage provides a final pass that catches remaining awkward phrasing. The advisory checkpoint after Copyedit allows human review.
- **Affected tasks:** Task 4 (Style), Task 5 (Copyedit).

### Risk: Links to Unproduced Documents Break Reader Flow

- **Probability:** Medium — the User Guide is produced first; the 7 documents it links to do not yet exist.
- **Impact:** Low — broken links are expected at this stage and resolve as later documents are produced. FR-018 explicitly permits this.
- **Mitigation:** Links use planned file paths from the design inventory (§4.1). A final link-check pass should run after all P22 documents are complete.
- **Affected tasks:** Task 1 (Write), Task 3 (Check).

---

## Verification Approach

| Acceptance Criterion | Verification Method | Producing Task |
|---------------------|---------------------|----------------|
| AC-001 (FR-001): "What is Kanbanzai" section | Inspection during Edit | Task 2 |
| AC-002 (FR-002): Collaboration model section | Inspection during Edit | Task 2 |
| AC-003 (FR-003): Stage-gate workflow summary | Inspection during Edit | Task 2 |
| AC-004 (FR-004): Document types mapped to stages | Inspection during Edit | Task 2 |
| AC-005 (FR-005): Approval gates and stage return | Inspection during Edit | Task 2 |
| AC-006 (FR-006): Bugs/incidents ≤ 3 paragraphs | Inspection during Edit | Task 2 |
| AC-007 (FR-007): Orchestration capabilities named | Inspection during Check | Task 3 |
| AC-008 (FR-008): Knowledge persistence and compounding | Inspection during Check | Task 3 |
| AC-009 (FR-009): Retrospective three-step workflow | Inspection during Check | Task 3 |
| AC-010 (FR-010): Concurrency mechanisms named | Inspection during Check | Task 3 |
| AC-011 (FR-011): MCP server transport and tool organisation | Inspection during Check | Task 3 |
| AC-012 (FR-012): `.kbz/`, Git-native, committed vs derived | Inspection during Check | Task 3 |
| AC-013 (FR-013): ≥ 3 reader-goal-to-document mappings | Inspection during Edit | Task 2 |
| AC-014 (FR-014, 015, 016): Structure and headings | Inspection during Edit | Task 2 |
| AC-015 (FR-017): Links to all 7 documents | Automated grep after Write | Task 1 |
| AC-016 (FR-019–022): No out-of-scope content | Inspection during Edit | Task 2 |
| AC-017 (NFR-001, 002): Voice, tense, no marketing | Inspection during Style and Copyedit | Task 4, Task 5 |
| AC-018 (NFR-004): All 5 pipeline stages completed | Inspection of changelogs after Copyedit | Task 5 |
| AC-019 (NFR-005): Zero error/hallucination findings | Check stage report | Task 3 |
| AC-020 (NFR-006): Zero unresolved style findings | Style stage report | Task 4 |