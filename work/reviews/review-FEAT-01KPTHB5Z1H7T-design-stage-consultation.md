# Review: Design-Stage Corpus Consultation
**Feature:** FEAT-01KPTHB5Z1H7T
**Branch:** feature/FEAT-01KPTHB5Z1H7T-design-stage-consultation
**Spec:** work/spec/doc-intel-design-stage-consultation.md
**Date:** 2026-04-22

---

```
Review Unit: design-stage-consultation
Files:
  - .kbz/roles/reviewer-conformance.yaml
  - .kbz/skills/write-design/SKILL.md
  - .kbz/skills/write-spec/SKILL.md
  - work/templates/specification-prompt-template.md
Spec: work/spec/doc-intel-design-stage-consultation.md
Reviewer Role: reviewer-conformance + reviewer-quality

Overall: approved_with_followups

Dimensions:
  spec_conformance: pass
    Evidence:
      - AC-1 (FR-001, FR-002): "Phase 0: Corpus Discovery" present at L100-152 in
        write-design/SKILL.md, preceding all other phases (Step 1 begins at L152).
        Text reads: "This phase is non-optional." — satisfies both requirements.
      - AC-2 (FR-003): Pre-Discovery Classification Check at L106-114 explicitly calls
        `doc_intel(action: "pending")` as step 1 before any concept or entity search.
        Section precedes Step 0.1: Concept Search (L114-122).
      - AC-3 (FR-004, FR-005): Step 0.1 (L114-122) calls both
        `doc_intel(action: "search", query: "<concept>")` and
        `doc_intel(action: "find", concept: "<concept>")`, then specifies a Fallback
        to FTS search with broader terms and `grep` when both return no results.
      - AC-4 (FR-006): Step 0.2 (L122-128) specifies
        `doc_intel(action: "find", entity_id: "<FEAT-xxx>")` for each known related feature.
      - AC-5 (FR-007): Step 0.3 (L128-135) specifies
        `doc_intel(action: "find", role: "decision", scope: "<DOC-xxx>")` for each
        related document found in prior steps.
      - AC-6 (FR-008): Step 0.4 (L135-143) requires synthesis of (a) Related documents,
        (b) Relevant decisions, (c) Open questions raised by prior work — all three
        elements explicitly present.
      - AC-7 (FR-009): Step 0.5 (L143-152) requires writing the Related Work section
        "BEFORE writing any other design section (Problem and Motivation, Design,
        Alternatives Considered, Decisions)."
      - AC-8 (FR-010): BECAUSE clause present in Phase 0 preamble: "Designing in
        isolation from prior work creates design debt, contradictions, and redundant
        decisions…"
      - AC-9 (FR-011): Checklist (L331-337) contains all three items with exact
        prescribed wording.
      - AC-10 (FR-012): specification-prompt-template.md lists "Related Work" as
        item #1 in the Required Sections list.
      - AC-11 (FR-013): Template defines Option A (three sub-elements: prior designs
        list, constraining decisions list, extension narrative) and Option B (three
        sub-elements: concepts searched, entity IDs searched, explicit attestation).
        Both forms presented as valid.
      - AC-12 (FR-014): Template states "An empty or absent Related Work section is
        NOT a valid answer."
      - AC-13 (FR-015, FR-016): reviewer-conformance.yaml adds anti-pattern entry
        "Missing or Placeholder Related Work Section" with `verdict: REJECTED`.
      - AC-14 (FR-017): `fires_when` in the blocking check lists exactly the two
        required conditions: section absent; section contains placeholder text
        without supporting evidence.
      - AC-15 (FR-018): `checks` in the blocking check lists all three required items:
        (1) section present, (2) substantive content or attestation with evidence,
        (3) engagement with corpus-discoverable related documents.
      - AC-16 (FR-019): `because` clause present in the blocking check entry, covering
        design continuity at scale and the "unenforced requirement is worse than no
        requirement" rationale.
      - AC-17 (FR-020, FR-021): write-spec/SKILL.md Cross-Reference Check (L82-93)
        marked "This step is required" and positioned before Step 1. Four prescribed
        steps in correct order: (1) verify Related Work section, STOP if absent;
        (2) search related specs with `doc_intel(action: "find", role: "requirement",
        scope:)`; (3) identify consistency constraints; (4) note deliberate divergences.
      - AC-18 (FR-022): BECAUSE clause present in Cross-Reference Check preamble.
      - AC-19 (NFR-001): All four changed artefacts are skill/role/template documents.
        No Go source files are present in the diff.
      - AC-20 (NFR-002): Compared worktree files against main-branch originals.
        write-design: original had Steps 1-5 (L98-138) and no Checklist section;
        worktree inserts Phase 0 before Step 1 and adds a new Checklist section —
        no existing content removed or reordered. write-spec: original had Steps 1-7
        (L80-130); worktree inserts Cross-Reference Check before Step 1 — all seven
        original steps preserved and shifted. reviewer-conformance.yaml: original had
        three anti-patterns; worktree adds a fourth — original entries unchanged.
        specification-prompt-template.md: Related Work added as item #1 in required
        sections; all pre-existing sections retained.

    Findings:
      - [non-blocking] spec-gap: NFR-004 states "The five steps of Phase 0" but
        enumerates six items (pre-discovery check, concept search, entity search,
        decision extraction, synthesis, write Related Work section). The implementation
        correctly resolves this as an unnumbered Pre-Discovery Classification Check
        plus five numbered steps (0.1-0.5), which is fully consistent with FR-003
        through FR-009. The inconsistency is in the spec wording, not the
        implementation. (spec: NFR-004)

  implementation_quality: pass_with_notes
    Evidence:
      - Phase 0 is coherently sequenced: pre-discovery check -> concept search ->
        entity search -> decision extraction -> synthesis -> write Related Work.
        Steps are numbered 0.1-0.5 and each is actionable with explicit tool calls.
        (write-design/SKILL.md L100-152)
      - Related Work section requirement in specification-prompt-template.md is
        unambiguous: two valid forms (Option A / Option B), each with enumerated
        sub-elements, and an explicit statement that empty/absent is not valid.
      - Blocking conformance check in reviewer-conformance.yaml is well-positioned
        as a new anti-pattern entry consistent with the existing three entries.
        Fields `verdict`, `fires_when`, `checks`, `because`, and `resolve` are
        self-contained and actionable.
      - write-spec Cross-Reference Check is clear and actionable; BECAUSE clause
        explains the rationale concisely; Step 1 includes the correct STOP
        instruction with escalation path to the orchestrator.
      - write-design checklist items use the exact prescribed wording and are
        correctly placed in the new Checklist section.

    Findings:
      - [non-blocking] Output Format inconsistency in write-design: The Output Format
        section (L190-243) states "The document then has exactly 4 required sections"
        and the embedded document template shows only Problem and Motivation, Design,
        Alternatives Considered, Decisions. Phase 0 Step 0.5 mandates writing a
        Related Work section before those four sections, but the Output Format template
        does not show Related Work as a required section of the design document, does
        not provide example content or a valid-forms description for it, and does not
        update the "exactly 4" count. An agent consulting the Output Format section as
        the authoritative design-document template could produce a design document
        without a Related Work section while believing they have correctly followed the
        skill. The enforcement chain relies entirely on Phase 0 being read and
        followed; the Output Format creates a contradictory signal.
        (location: write-design/SKILL.md L193-194, "exactly 4 required sections")

Finding Summary:
  Blocking: 0
  Non-blocking: 2
  Total: 2
```

---

## Acceptance Criteria Traceability Matrix

| AC | FR/NFR | Verdict | Evidence Location |
|----|--------|---------|-------------------|
| AC-1 | FR-001, FR-002 | pass | write-design/SKILL.md L100-105 |
| AC-2 | FR-003 | pass | write-design/SKILL.md L106-114 |
| AC-3 | FR-004, FR-005 | pass | write-design/SKILL.md L114-122 |
| AC-4 | FR-006 | pass | write-design/SKILL.md L122-128 |
| AC-5 | FR-007 | pass | write-design/SKILL.md L128-135 |
| AC-6 | FR-008 | pass | write-design/SKILL.md L135-143 |
| AC-7 | FR-009 | pass | write-design/SKILL.md L143-152 |
| AC-8 | FR-010 | pass | write-design/SKILL.md L103-105 |
| AC-9 | FR-011 | pass | write-design/SKILL.md L331-337 |
| AC-10 | FR-012 | pass | specification-prompt-template.md (item #1) |
| AC-11 | FR-013 | pass | specification-prompt-template.md (Option A, Option B) |
| AC-12 | FR-014 | pass | specification-prompt-template.md (explicit NOT valid statement) |
| AC-13 | FR-015, FR-016 | pass | reviewer-conformance.yaml (4th anti-pattern, `verdict: REJECTED`) |
| AC-14 | FR-017 | pass | reviewer-conformance.yaml (`fires_when`, 2 conditions) |
| AC-15 | FR-018 | pass | reviewer-conformance.yaml (`checks`, 3 items) |
| AC-16 | FR-019 | pass | reviewer-conformance.yaml (`because` clause) |
| AC-17 | FR-020, FR-021 | pass | write-spec/SKILL.md L82-93 |
| AC-18 | FR-022 | pass | write-spec/SKILL.md L82-86 (BECAUSE clause) |
| AC-19 | NFR-001 | pass | No Go files in diff |
| AC-20 | NFR-002 | pass | Diff comparison: original content preserved in all four files |

---

## Follow-Up Items

### FU-1 (non-blocking quality): Update Output Format in write-design SKILL.md

**File:** `.kbz/skills/write-design/SKILL.md`
**Location:** L193-194 ("The document then has exactly 4 required sections")

The Output Format section should be updated to:
1. Change "exactly 4 required sections" to "exactly 5 required sections"
2. Add Related Work as the first section in the embedded document template, with guidance
   matching the two valid forms (Option A / Option B) already defined in
   `specification-prompt-template.md`

Without this change the Output Format section provides a contradictory signal to agents
who reference it as the authoritative design-document template.

### FU-2 (spec-gap, for future spec maintenance): NFR-004 word count

**File:** `work/spec/doc-intel-design-stage-consultation.md`
**Location:** NFR-004

NFR-004 reads "The five steps of Phase 0" but the parenthetical enumeration that follows
lists six items. The implementation correctly handles this as an unnumbered pre-check plus
five numbered steps (0.1-0.5). The spec wording should be corrected in a future revision
to avoid ambiguity — either "a pre-discovery check followed by five numbered steps" or
relabelling the pre-check as Step 0.0.