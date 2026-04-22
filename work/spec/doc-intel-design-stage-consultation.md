# Specification: Design-Stage Corpus Consultation

| Field         | Value                                                                                |
|---------------|--------------------------------------------------------------------------------------|
| Feature       | FEAT-01KPTHB5Z1H7T                                                                   |
| Design source | `work/design/doc-intel-adoption-design.md` §3 (Fix 1)                               |
| Status        | draft                                                                                |
| Author        | Spec-author task                                                                     |

## Related Work

### Prior designs and specifications consulted

- [Design: Document Intelligence Adoption and Integration](../design/doc-intel-adoption-design.md) —
  this specification directly implements Fix 1 (§3) of that design. All requirements are derived
  from §§3.2–3.6.

### Decisions that constrain this specification

- The reviewer-conformance blocking check targets `.kbz/roles/reviewer-conformance.yaml` (not a
  `SKILL.md`), because that is the only conformance artefact that exists at the time of this
  specification (confirmed by filesystem check). If a `reviewer-conformance/SKILL.md` is created
  before implementation, the implementing agent MUST consult the orchestrator to confirm the target.

- Phase 0 includes a pre-discovery classification check (§3.3 of the design) because an
  unclassified corpus produces false negatives in concept search; the check must precede corpus
  discovery, not follow it.

### How this specification extends prior work

This is a new specification for a new feature. It applies no design pattern not already established
in the write-design and write-spec skill files; it extends those files with additional phases and
checklist items.

---

## Overview

This specification covers FEAT-01KPTHB5Z1H7T: mandatory corpus consultation during the design
stage. It formalises changes to four workflow artefacts — the `write-design` skill, the
`write-spec` skill, the specification prompt template, and the `reviewer-conformance` role — that
together form a complete enforcement chain. Architects are required to search the corpus before
writing any design content and to document what they find in a Required Related Work section;
specification authors are required to verify that Related Work section exists before drafting
requirements; and conformance reviewers are required to reject designs that omit or hollow out the
section. The feature contains no Go code changes.

---

## Scope

### In scope

- Adding Phase 0: Corpus Discovery to `.kbz/skills/write-design/SKILL.md`
- Adding a cross-reference check to `.kbz/skills/write-spec/SKILL.md`
- Adding a Required Related Work section to `work/templates/specification-prompt-template.md`
- Adding a blocking Related Work check to `.kbz/roles/reviewer-conformance.yaml`

### Out of scope

- Corpus onboarding and completeness checks (Fix 2 of the design)
- Classification triggers and batch classification (Fix 3)
- Knowledge retrieval at implementation time (Fix 4)
- Access instrumentation (Fix 5)
- Plan close-out knowledge curation (Fix 6)
- Changes to Go server code or MCP tool action surfaces
- Embedding-based semantic search or new `doc_intel` tool actions
- Automated or pipeline-driven classification

---

## Functional Requirements

### Group A — `write-design/SKILL.md`: Phase 0 Corpus Discovery

**FR-001:** `write-design/SKILL.md` MUST add a "Phase 0: Corpus Discovery" section as its first
phase, placed before all existing phases.

**FR-002:** Phase 0 MUST be explicitly marked as non-optional within the skill text.

**FR-003:** Phase 0 MUST begin with a pre-discovery classification check: before any corpus
discovery calls are made, the agent MUST call `doc_intel(action: "pending")` and classify any
relevant unclassified documents it finds.

**FR-004:** Phase 0, Step 1 MUST specify concept search using both of the following calls for each
primary concept in the feature:
- `doc_intel(action: "search", query: "<concept>")`
- `doc_intel(action: "find", concept: "<concept>")`

**FR-005:** Phase 0, Step 1 MUST include a fallback instruction: if concept search returns no
results, the agent MUST fall back to FTS search and grep because the corpus may be unclassified.

**FR-006:** Phase 0, Step 2 MUST specify entity search: for each known related feature, the agent
MUST call `doc_intel(action: "find", entity_id: "<FEAT-xxx>")`.

**FR-007:** Phase 0, Step 3 MUST specify decision extraction: for each related document found, the
agent MUST call `doc_intel(action: "find", role: "decision", scope: "<DOC-xxx>")`.

**FR-008:** Phase 0, Step 4 MUST require the agent to produce a synthesis of: (a) related
documents, (b) relevant decisions that constrain the design, and (c) open questions raised by prior
work.

**FR-009:** Phase 0, Step 5 MUST require that the Related Work section of the design document is
written from the synthesis produced in Step 4, and MUST require this to happen before any other
section of the design document is written.

**FR-010:** Phase 0 MUST include a BECAUSE clause that explains the rationale for mandatory corpus
consultation before design begins.

**FR-011:** The `write-design/SKILL.md` checklist MUST include the following three new items,
with this precise wording:
1. `[ ] Conducted corpus discovery (concept search, entity search, decision extraction)`
2. `[ ] Wrote Related Work section before writing any design content`
3. `[ ] Cross-referenced at least one prior decision that constrains this design, OR attested that corpus search found no related work`

### Group B — `work/templates/specification-prompt-template.md`: Required Related Work Section

**FR-012:** The specification prompt template MUST add "Related Work" to its list of required
sections.

**FR-013:** The template MUST define exactly two valid forms for the Related Work section:

- **Option A — Related work found:** MUST contain all three of the following sub-elements:
  1. A list of prior designs and specifications consulted, each with a description of how it
     relates to the current design.
  2. A list of decisions that constrain the current design, each citing source document and
     section.
  3. A narrative explaining how the current design extends or diverges from prior work.

- **Option B — No related work found:** MUST contain all three of the following sub-elements:
  1. The concepts that were searched.
  2. The entity IDs that were searched (where applicable).
  3. An explicit attestation that no directly related prior work was found in the classified
     corpus.

**FR-014:** The template MUST explicitly state that an empty or absent Related Work section is not
a valid answer.

### Group C — `.kbz/roles/reviewer-conformance.yaml`: Blocking Related Work Check

**FR-015:** `.kbz/roles/reviewer-conformance.yaml` MUST add a blocking check for Related Work
section quality to its check procedure.

**FR-016:** The blocking check MUST declare REJECTED as the reviewer verdict when it is triggered.

**FR-017:** The blocking check MUST specify the following two conditions as grounds for REJECTED:
1. The Related Work section is absent from the design document.
2. The Related Work section contains placeholder text (e.g., "TBD", "N/A") without supporting
   evidence.

**FR-018:** The blocking check MUST enumerate three check items that must all pass before a design
can be approved:
1. The Related Work section is present in the design document.
2. The section contains either substantive cross-references to prior work OR an explicit
   "no related work found" attestation with search evidence.
3. If related documents exist in the corpus that clearly relate to this design, the design engages
   with them; it MUST NOT ignore them silently.

**FR-019:** The blocking check MUST include a BECAUSE clause explaining why the Related Work
section is the primary enforcement mechanism for design continuity at scale, and why an
unenforced Related Work requirement is worse than no requirement.

### Group D — `write-spec/SKILL.md`: Cross-Reference Check

**FR-020:** `write-spec/SKILL.md` MUST add a cross-reference check as a required step before any
specification content is written.

**FR-021:** The cross-reference check MUST contain the following four steps, in order:
1. Verify that the design document for this feature has a substantive Related Work section. If it
   does not, the agent MUST STOP and flag the incomplete design to the orchestrator.
2. Search for specifications of features identified in the Related Work section:
   `doc_intel(action: "find", role: "requirement", scope: "<DOC-related-spec>")`.
3. Identify requirements in adjacent specifications that the current specification must be
   consistent with.
4. Note deliberate divergences: if this specification takes a different approach than an adjacent
   specification, document the reason.

**FR-022:** The cross-reference check MUST include a BECAUSE clause explaining why
cross-referencing adjacent specifications prevents inconsistent behaviour across features.

---

## Non-Functional Requirements

**NFR-001:** All changes MUST be confined to the four specified artefacts:
`.kbz/skills/write-design/SKILL.md`, `.kbz/skills/write-spec/SKILL.md`,
`work/templates/specification-prompt-template.md`, and `.kbz/roles/reviewer-conformance.yaml`.
No Go source files are modified by this feature.

**NFR-002:** Changes to skill files and the role file MUST integrate with their existing
structure and formatting conventions without restructuring or removing any existing content.

**NFR-003:** All BECAUSE clauses required by FR-010, FR-019, and FR-022 MUST be retained in the
modified artefacts so the enforcement logic is self-documenting.

**NFR-004:** The five steps of Phase 0 (pre-discovery check, concept search, entity search,
decision extraction, synthesis, write Related Work section) MUST appear in the prescribed
order and MUST be numbered sequentially.

---

## Acceptance Criteria

- [ ] **FR-001, FR-002:** `write-design/SKILL.md` contains a "Phase 0: Corpus Discovery" section
  that precedes all other phases and is marked as non-optional.
- [ ] **FR-003:** Phase 0 contains a pre-discovery classification check that calls
  `doc_intel(action: "pending")` before any concept or entity search calls.
- [ ] **FR-004, FR-005:** Phase 0 Step 1 specifies both `doc_intel(action: "search")` and
  `doc_intel(action: "find", concept:)` for each primary concept, plus a fallback to FTS search
  and grep when concept search returns no results.
- [ ] **FR-006:** Phase 0 Step 2 specifies `doc_intel(action: "find", entity_id:)` for known
  related features.
- [ ] **FR-007:** Phase 0 Step 3 specifies `doc_intel(action: "find", role: "decision", scope:)`
  for each related document found.
- [ ] **FR-008:** Phase 0 Step 4 requires a synthesis of related documents, constrained decisions,
  and open questions.
- [ ] **FR-009:** Phase 0 Step 5 requires the Related Work section to be written before any other
  section of the design document.
- [ ] **FR-010:** Phase 0 contains a BECAUSE clause.
- [ ] **FR-011:** The `write-design/SKILL.md` checklist contains the three new corpus discovery
  check items with the prescribed wording.
- [ ] **FR-012:** `work/templates/specification-prompt-template.md` lists "Related Work" as a
  required section.
- [ ] **FR-013:** The template defines both Option A and Option B with their required sub-elements,
  and makes clear that both are valid forms.
- [ ] **FR-014:** The template explicitly states that an empty or absent Related Work section is
  not valid.
- [ ] **FR-015, FR-016:** `.kbz/roles/reviewer-conformance.yaml` contains a blocking check that
  declares REJECTED when triggered.
- [ ] **FR-017:** The blocking check fires when the Related Work section is absent or contains
  unsubstantiated placeholder text.
- [ ] **FR-018:** The blocking check lists three check items covering: section presence,
  substantive content or attestation, and engagement with corpus-discoverable related documents.
- [ ] **FR-019:** The blocking check contains a BECAUSE clause.
- [ ] **FR-020, FR-021:** `write-spec/SKILL.md` contains a cross-reference check as a required
  step before specification content is written, with the four prescribed steps in order.
- [ ] **FR-022:** The cross-reference check contains a BECAUSE clause.
- [ ] **NFR-001:** No Go source files are modified by this feature.
- [ ] **NFR-002:** All four modified artefacts retain their existing structural conventions; no
  existing content is removed or reordered.

---

## Dependencies and Assumptions

**DEP-001:** This specification depends on the approval of `work/design/doc-intel-adoption-design.md`.
The design is in `approved` status as of 2026-04-23.

**DEP-002:** The `doc_intel` tool actions used in Phase 0 — `search`, `find` (with `concept`,
`entity_id`, and `role`/`scope` parameters), and `pending` — MUST already exist in the tool
surface. This feature introduces no new tool actions.

**DEP-003:** The reviewer-conformance blocking check targets `.kbz/roles/reviewer-conformance.yaml`
because that is the only reviewer-conformance artefact that exists at specification time. If a
`reviewer-conformance/SKILL.md` is created before implementation, the implementing agent MUST
consult the orchestrator before choosing a target file.

**DEP-004:** This feature does not depend on corpus completeness (Fix 2) or classification
triggers (Fix 3) being implemented first. The fallback instruction in FR-005 makes Phase 0
tolerant of an unclassified or incomplete corpus.

**ASM-001:** It is assumed that agents executing the `write-design` skill read Phase 0 in its
entirety and do not skip it because it appears before existing content.

**ASM-002:** It is assumed that the template's explicit "not a valid answer" language for an empty
Related Work section is sufficient to prevent agents from leaving it blank, without requiring a
programmatic enforcement gate.
