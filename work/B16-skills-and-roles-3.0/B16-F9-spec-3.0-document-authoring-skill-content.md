# Specification: Document Authoring Skill Content

| Field   | Value                                                              |
|---------|--------------------------------------------------------------------|
| Status  | Draft                                                              |
| Feature | FEAT-01KN5-88PFWADY (document-authoring-skill-content)             |
| Design  | `work/design/skills-system-redesign-v2.md` §5.1, §3.2, §8, §9    |

---

## 1. Overview

This specification defines the content requirements for five document authoring SKILL.md files — `write-design`, `write-spec`, `write-dev-plan`, `write-research`, and `update-docs` — that replace the current generic `document-creation` SKILL. Each skill follows the attention-curve SKILL.md format defined in §3.2 of the design, is bound to a specific feature lifecycle stage and paired role, carries document-type-specific vocabulary and anti-patterns, defines a gate-checkable output template with required sections, and includes BAD vs GOOD examples with evaluation criteria phrased as gradable questions. Companion validation scripts in each skill's `scripts/` directory enable programmatic structural verification at stage transitions.

---

## 2. Scope

### 2.1 In Scope

- Content requirements for five SKILL.md files: `write-design`, `write-spec`, `write-dev-plan`, `write-research`, `update-docs`.
- Stage alignment, paired role binding, and frontmatter field values for each skill.
- Required vocabulary payload, anti-pattern set, output format template, examples, and evaluation criteria for each skill.
- Gate-checkable document templates: required sections per document type and cross-reference requirements.
- Validation scripts in `scripts/` for structural verification of each skill's output.
- Replacement of the current generic `.skills/document-creation.md`.

### 2.2 Explicitly Excluded

- The SKILL.md file format schema itself (that is a structural concern defined in the skill system spec and design §3.2).
- Context assembly — how skills are combined with roles and injected into agent prompts (covered by the context assembly pipeline spec).
- The binding registry schema or stage-bindings.yaml structure (covered by the binding registry spec).
- Role content — the vocabulary and anti-patterns carried by paired roles such as `architect` or `spec-author` (covered by the role content spec).
- Gate enforcement mechanism — how `entity(action: "transition")` checks prerequisites (covered by the workflow spec).
- Skill development process and quality gate procedure (design §9 — a process concern, not a content spec).
- Implementation of validation scripts (this spec defines their interface, not their code).

---

## 3. Functional Requirements

### FR-001: Five Authoring Skills Replace Generic Document-Creation

The system MUST provide exactly five document authoring skills (`write-design`, `write-spec`, `write-dev-plan`, `write-research`, `update-docs`), each stored at `.kbz/skills/{skill-name}/SKILL.md`. The current `.skills/document-creation.md` MUST NOT be used as the authoring skill for any document type once these five skills are available.

**Acceptance criteria:**
- Each of the five SKILL.md files exists at its specified path under `.kbz/skills/`
- No stage binding references `document-creation` as a skill; all document-producing stages reference one of the five type-specific skills
- The `document-creation` skill is not listed in any skill's `roles` field or any binding's `skill` field

---

### FR-002: Stage Alignment

Each authoring skill MUST declare exactly one `stage` in its frontmatter that matches the feature lifecycle stage where that document type is produced. The required bindings are:

| Skill | Stage |
|-------|-------|
| `write-design` | `designing` |
| `write-spec` | `specifying` |
| `write-dev-plan` | `dev-planning` |
| `write-research` | `researching` |
| `update-docs` | `documenting` |

**Acceptance criteria:**
- Each skill's frontmatter `stage` field matches the value in the table above
- No two authoring skills declare the same stage (each stage has exactly one authoring skill)
- The stage values correspond to valid feature lifecycle states in the existing state machine

---

### FR-003: Paired Role Declaration

Each authoring skill MUST declare its compatible roles in the frontmatter `roles` field. The required pairings are:

| Skill | Paired Role(s) |
|-------|----------------|
| `write-design` | `architect` |
| `write-spec` | `spec-author` |
| `write-dev-plan` | `architect` |
| `write-research` | `researcher` |
| `update-docs` | `documenter` |

**Acceptance criteria:**
- Each skill's `roles` field contains exactly the role(s) listed in the table above
- Every role listed in a skill's `roles` field corresponds to a role file that exists (or will exist) at `.kbz/roles/{role-id}.yaml`

---

### FR-004: Attention-Curve Section Ordering

Each authoring skill's SKILL.md MUST follow the attention-curve section ordering defined in the design (§3.2). The sections MUST appear in this order:

1. Frontmatter (YAML)
2. `## Vocabulary`
3. `## Anti-Patterns`
4. `## Checklist` (optional — required for medium/low constraint_level skills)
5. `## Procedure`
6. `## Output Format`
7. `## Examples`
8. `## Evaluation Criteria`
9. `## Questions This Skill Answers`

**Acceptance criteria:**
- Every authoring SKILL.md contains sections 1–2, 3, 5, 6, 7, 8, and 9 in the specified order
- No section appears out of the defined order
- Optional sections, if present, appear at their defined position

---

### FR-005: Vocabulary Payload Requirements

Each authoring skill MUST carry a vocabulary section containing 15–30 domain-specific terms relevant to its document type. Terms MUST pass the 15-year practitioner test: a senior expert with 15+ years of domain experience would use the exact term when speaking with a peer. The vocabulary MUST be specific to the document type, not generic document-writing vocabulary.

**Acceptance criteria:**
- Each skill's `## Vocabulary` section contains between 15 and 30 terms (inclusive)
- The vocabulary for `write-spec` includes requirements-engineering terms (e.g., "testable assertion", "acceptance criterion", "traceability matrix", "constraint")
- The vocabulary for `write-design` includes architecture/design terms (e.g., "design alternative", "architectural decision record", "trade-off analysis")
- The vocabulary for `write-dev-plan` includes planning terms (e.g., "dependency graph", "vertical slice", "task decomposition", "effort estimate")
- The vocabulary for `write-research` includes research terms (e.g., "literature synthesis", "evidence grading", "methodology", "finding")
- The vocabulary for `update-docs` includes documentation terms (e.g., "information architecture", "content currency", "cross-reference integrity")
- No two skills share more than 5 vocabulary terms

---

### FR-006: Anti-Pattern Requirements

Each authoring skill MUST carry 5–10 named anti-patterns specific to common authoring failures for its document type. Each anti-pattern MUST include: a name, a `Detect` signal, a `BECAUSE` clause explaining why it is harmful, and a `Resolve` step.

**Acceptance criteria:**
- Each skill's `## Anti-Patterns` section contains between 5 and 10 anti-patterns (inclusive)
- Every anti-pattern has all four fields: name, Detect, BECAUSE, Resolve
- Anti-patterns are specific to the document type (e.g., `write-spec` has anti-patterns for vague requirements; `write-design` has anti-patterns for missing alternatives)
- No anti-pattern's BECAUSE clause is empty or restates the detection signal

---

### FR-007: BAD vs GOOD Examples

Each authoring skill MUST include at least two examples in its `## Examples` section: at minimum one BAD example and one GOOD example. Each example MUST include a `WHY BAD` or `WHY GOOD` explanation. The best (GOOD) example MUST appear last in the section to exploit recency bias.

**Acceptance criteria:**
- Each skill has at least one BAD and one GOOD example
- Every BAD example has a `WHY BAD` explanation
- Every GOOD example has a `WHY GOOD` explanation
- The final example in each skill's Examples section is a GOOD example
- Examples are representative of the specific document type, not generic

---

### FR-008: Evaluation Criteria as Gradable Questions

Each authoring skill MUST include an `## Evaluation Criteria` section containing 4–8 gradable questions about the skill's output quality. Each question MUST have a weight designation (`required`, `high`, or `medium`). The criteria MUST be evaluable by an LLM-as-judge pass producing scores from 0.0–1.0.

**Acceptance criteria:**
- Each skill has between 4 and 8 evaluation criteria (inclusive)
- Each criterion is phrased as a yes/no or gradable question
- Each criterion has a weight of `required`, `high`, or `medium`
- At least one criterion per skill has weight `required`
- The criteria are specific to the document type, not generic writing quality

---

### FR-009: Gate-Checkable Specification Template

The `write-spec` skill's output format MUST define a template with exactly 5 required sections: Problem Statement, Requirements, Constraints, Acceptance Criteria, Verification Plan. The template MUST require that the Problem Statement references the parent design document.

**Acceptance criteria:**
- The `write-spec` `## Output Format` section lists exactly these 5 required sections by name
- The output format states that Problem Statement must reference the parent design document
- The required sections match the canonical definition in the design (§5.1) and in `stage-bindings.yaml`'s `document_template.required_sections` for the `specifying` stage

---

### FR-010: Gate-Checkable Dev-Plan Template

The `write-dev-plan` skill's output format MUST define a template with exactly 5 required sections: Scope, Task Breakdown, Dependency Graph, Risk Assessment, Verification Approach. The template MUST require that the Scope section references the parent specification.

**Acceptance criteria:**
- The `write-dev-plan` `## Output Format` section lists exactly these 5 required sections by name
- The output format states that Scope must reference the parent specification
- The required sections match the canonical definition in the design (§5.1) and in `stage-bindings.yaml`'s `document_template.required_sections` for the `dev-planning` stage

---

### FR-011: Gate-Checkable Design Document Template

The `write-design` skill's output format MUST define a template with exactly 4 required sections: Problem and Motivation, Design, Alternatives Considered, Decisions.

**Acceptance criteria:**
- The `write-design` `## Output Format` section lists exactly these 4 required sections by name
- The required sections match the canonical definition in the design (§5.1) and in `stage-bindings.yaml`'s `document_template.required_sections` for the `designing` stage

---

### FR-012: Cross-Reference Requirements

Each document template that has a predecessor document type MUST declare a cross-reference requirement in its output format. Specifically: the specification template MUST require a reference to the parent design document, and the dev-plan template MUST require a reference to the parent specification.

**Acceptance criteria:**
- The `write-spec` output format includes a cross-reference requirement to the parent design document
- The `write-dev-plan` output format includes a cross-reference requirement to the parent specification
- The `write-design` output format does NOT require a cross-reference to a predecessor (it is the first document in the chain)
- Cross-reference requirements are stated as verifiable conditions (e.g., "the Problem Statement section MUST contain a reference to the design document by path or document ID")

---

### FR-013: Validation Scripts

Each authoring skill that defines a gate-checkable template (`write-design`, `write-spec`, `write-dev-plan`) MUST have a validation script at `.kbz/skills/{skill-name}/scripts/validate-{type}-structure.sh`. The script MUST check for the presence of all required sections and cross-references defined in that skill's output format. The script MUST exit with code 0 on success and non-zero on failure, and MUST output a human-readable description of any missing sections or references.

**Acceptance criteria:**
- `write-design` has a script at `.kbz/skills/write-design/scripts/validate-design-structure.sh`
- `write-spec` has a script at `.kbz/skills/write-spec/scripts/validate-spec-structure.sh`
- `write-dev-plan` has a script at `.kbz/skills/write-dev-plan/scripts/validate-dev-plan-structure.sh`
- Each script accepts a file path as its argument
- Each script exits 0 when all required sections and cross-references are present
- Each script exits non-zero and prints the names of missing sections when any are absent
- Script output is suitable for inclusion in an agent's context window (human-readable, concise)

---

### FR-014: Frontmatter Completeness

Each authoring skill's SKILL.md MUST include a YAML frontmatter block with all required fields: `name`, `description` (with `expert` and `natural` sub-fields), `triggers` (list of at least 2 trigger phrases), `roles`, `stage`, and `constraint_level`.

**Acceptance criteria:**
- Each of the five skills has a frontmatter block containing all six required fields
- The `description.expert` field is a technical description suitable for direct invocation
- The `description.natural` field is a plain-language description suitable for casual trigger matching
- The `triggers` field contains at least 2 trigger phrases per skill
- The `constraint_level` field is one of `low`, `medium`, or `high`

---

### FR-015: Questions This Skill Answers

Each authoring skill MUST include a `## Questions This Skill Answers` section as the final section, containing 5–10 questions that serve as retrieval anchors. The questions MUST be specific to the document type and represent queries an agent or orchestrator would use to select this skill.

**Acceptance criteria:**
- Each skill has a `## Questions This Skill Answers` section as its last section
- Each section contains between 5 and 10 questions (inclusive)
- Questions are phrased as natural-language queries an agent would ask
- Questions are specific to the document type (e.g., `write-spec` questions reference specifications, not generic documents)

---

## 4. Non-Functional Requirements

**NFR-001:** Each SKILL.md file MUST be under 500 lines. Detailed anti-pattern documentation, extended examples, and evaluation rubrics that would push a skill over this limit MUST be placed in `references/` files within the skill's directory.

**Acceptance criteria:**
- `wc -l` on each SKILL.md returns a value ≤ 500
- Any `references/` files are linked directly from the SKILL.md (one level deep — no reference-to-reference chains)

---

**NFR-002:** Vocabulary terms across an authoring skill and its paired role MUST compose additively without duplication. Where the same term appears in both the role vocabulary and the skill vocabulary, it MUST appear in only one (preferring the role for domain terms and the skill for document-type terms).

**Acceptance criteria:**
- The intersection of vocabulary terms between each skill and its paired role contains zero entries
- Combined vocabulary (role + skill) stays within the 15–30 term guideline per unit

---

**NFR-003:** All reference files within a skill directory MUST be linked directly from the SKILL.md. No reference file may link to another reference file (one-level-deep constraint).

**Acceptance criteria:**
- Every file in a skill's `references/` directory is referenced by a link or citation in the parent SKILL.md
- No file in `references/` contains links to other files in `references/`

---

**NFR-004:** Validation scripts MUST complete within 5 seconds on a standard development machine and MUST have no dependencies beyond POSIX shell utilities (sh, grep, sed, awk).

**Acceptance criteria:**
- Each validation script runs to completion in under 5 seconds on a file of up to 2000 lines
- Scripts use only POSIX-standard utilities (no Python, Node, or other runtime dependencies)

---

## 5. Acceptance Criteria

The requirements above include inline acceptance criteria. The following are system-level acceptance criteria for the feature as a whole:

1. **Completeness:** All five SKILL.md files exist at their specified paths, each containing all required sections in attention-curve order.
2. **Template enforcement:** The three gate-checkable templates (`write-design`, `write-spec`, `write-dev-plan`) define required sections that match the canonical definitions in the design (§5.1) and are consistent with `stage-bindings.yaml`.
3. **Cross-reference chain:** The specification template requires a reference to the design document, and the dev-plan template requires a reference to the specification, forming a verifiable document chain.
4. **Validation coverage:** Each gate-checkable template has a companion validation script that correctly identifies missing sections and cross-references.
5. **Replacement verification:** No workflow path references the old `document-creation` skill after these five skills are available; the old skill is superseded.
6. **Quality gate:** Each skill passes the skill quality gate checklist defined in the design (§9.2): attention-curve ordering, vocabulary count 15–30, anti-patterns 5–10 with BECAUSE clauses, examples with WHY explanations, evaluation criteria as gradable questions.

---

## 6. Dependencies and Assumptions

### Dependencies

- **Skill system schema:** The SKILL.md file format (frontmatter fields, section structure) must be defined before authoring content. This is specified by the skill system feature.
- **Role content:** The paired roles (`architect`, `spec-author`, `researcher`, `documenter`) must have their vocabulary and anti-patterns defined so that skill content complements rather than duplicates role content.
- **Binding registry:** The `stage-bindings.yaml` file must define `document_template` structures for `designing`, `specifying`, and `dev-planning` stages. The required sections in the binding registry and in each skill's output format must be identical.
- **Feature lifecycle states:** The stages referenced by each skill (`designing`, `specifying`, `dev-planning`, `researching`, `documenting`) must be valid states in the feature lifecycle state machine (assumed unchanged from current system).

### Assumptions

- The attention-curve SKILL.md format (§3.2 of the design) is stable and will not change materially before these skills are authored.
- The existing document types (`design`, `specification`, `dev-plan`, `research`, `report`) and their registration via the `doc` tool are unchanged.
- Validation scripts receive a single file path argument and check only structural presence of sections, not semantic content quality.
- The `update-docs` and `write-research` skills do not require gate-checkable templates or validation scripts because their document types do not serve as prerequisites for stage transitions in the core lifecycle.