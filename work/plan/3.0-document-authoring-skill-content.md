# Implementation Plan: Document Authoring Skill Content

| Field   | Value                                                              |
|---------|--------------------------------------------------------------------|
| Status  | Draft                                                              |
| Feature | FEAT-01KN5-88PFWADY (document-authoring-skill-content)             |
| Spec    | `work/spec/3.0-document-authoring-skill-content.md`                |
| Design  | `work/design/skills-system-redesign-v2.md` §5.1, §3.2, §8, §9    |

---

## 1. Overview

This plan decomposes the document authoring skill content specification into assignable tasks for AI agents. The specification requires authoring five SKILL.md files (`write-design`, `write-spec`, `write-dev-plan`, `write-research`, `update-docs`) that replace the current generic `document-creation` skill. Each skill must follow the attention-curve format, carry type-specific vocabulary and anti-patterns, define gate-checkable templates (where applicable), include BAD/GOOD examples, and have companion validation scripts.

**Scope boundaries (from specification):**

- IN: Content for five SKILL.md files, validation scripts for three gate-checkable skills, `references/` files where needed to stay under 500 lines.
- OUT: SKILL.md format schema, context assembly pipeline, binding registry schema, role content, gate enforcement mechanism, validation script implementation logic beyond interface.

**Key structural decision:** All five skills are independent of each other — they share format conventions but not content. A single conventions-agreement task runs first, then all five skills are authored in parallel. Validation scripts accompany their respective skills (not separated into a later task) because the script author needs the template definition in working memory.

---

## 2. Task Breakdown

### Task 1: Establish Shared Conventions and Reference Materials

**Objective:** Produce a conventions document that all five skill authors reference for consistent formatting decisions — frontmatter field patterns, vocabulary term format, anti-pattern structure, example layout, and evaluation criteria phrasing. This is NOT a new spec; it is a working reference that resolves ambiguity in the shared format so the five parallel authors produce consistent output.

**Specification references:** FR-004 (attention-curve ordering), FR-005 (vocabulary format), FR-006 (anti-pattern structure), FR-007 (example format), FR-008 (evaluation criteria format), FR-014 (frontmatter fields), NFR-001 (500-line limit)

**Input context:**
- `work/design/skills-system-redesign-v2.md` §3.2 — attention-curve SKILL.md format definition
- `work/design/skills-system-redesign-v2.md` §9 — skill quality gate checklist
- `work/spec/3.0-document-authoring-skill-content.md` §3 FR-004 through FR-008, FR-014
- `.agents/skills/kanbanzai-documents/SKILL.md` — existing document skill for tone reference

**Output artifacts:**
- `.kbz/skills/CONVENTIONS.md` — shared formatting reference covering:
  - Frontmatter YAML block template (all six required fields with placeholder values)
  - Vocabulary section format (term + one-line definition, 15–30 terms)
  - Anti-pattern format (Name / Detect / BECAUSE / Resolve — four fields per entry)
  - Example format (BAD with `WHY BAD`, GOOD with `WHY GOOD`, GOOD last)
  - Evaluation criteria format (gradable question + weight)
  - `## Questions This Skill Answers` format (5–10 natural-language queries)
  - Line-budget guidance: when to use `references/` to stay under 500 lines

**Dependencies:** None — this is the first task.

---

### Task 2: Author `write-design/SKILL.md`

**Objective:** Create the design document authoring skill. This skill guides an agent paired with the `architect` role through producing a design document during the `designing` stage. The output template has 4 required sections: Problem and Motivation, Design, Alternatives Considered, Decisions. Include a companion validation script.

**Specification references:** FR-001, FR-002 (stage: `designing`), FR-003 (role: `architect`), FR-004, FR-005 (architecture/design vocabulary), FR-006 (design-specific anti-patterns), FR-007, FR-008, FR-011 (4-section template), FR-012 (no predecessor cross-reference required), FR-013 (validation script), FR-014, FR-015, NFR-001, NFR-002, NFR-003, NFR-004

**Input context:**
- `.kbz/skills/CONVENTIONS.md` (from Task 1)
- `work/spec/3.0-document-authoring-skill-content.md` FR-011 — required sections: Problem and Motivation, Design, Alternatives Considered, Decisions
- `work/design/skills-system-redesign-v2.md` §5.1 — canonical template definitions
- `.agents/skills/kanbanzai-design/SKILL.md` — existing MCP-client-side design skill for content inspiration (not to be copied verbatim — different format and purpose)
- `work/design/skills-system-redesign-v2.md` §9.2 — quality gate checklist

**Output artifacts:**
- `.kbz/skills/write-design/SKILL.md` — complete skill file with frontmatter (`name: write-design`, `stage: designing`, `roles: [architect]`, `constraint_level: high`), all attention-curve sections
- `.kbz/skills/write-design/scripts/validate-design-structure.sh` — POSIX shell script that accepts a file path, checks for 4 required sections, exits 0/non-zero with human-readable output
- `.kbz/skills/write-design/references/` — any overflow content (only if SKILL.md exceeds 500 lines without it)

**Dependencies:** Task 1

**Validation script interface contract:**
```
#!/bin/sh
# Usage: validate-design-structure.sh <path-to-document>
# Exit 0: all required sections present
# Exit 1: one or more sections missing (names printed to stdout)
# Required sections: "Problem and Motivation", "Design", "Alternatives Considered", "Decisions"
# Dependencies: POSIX shell utilities only (grep, sed, awk)
# Runtime: < 5 seconds on files up to 2000 lines
```

---

### Task 3: Author `write-spec/SKILL.md`

**Objective:** Create the specification authoring skill. This skill guides an agent paired with the `spec-author` role through producing a specification during the `specifying` stage. The output template has 5 required sections: Problem Statement, Requirements, Constraints, Acceptance Criteria, Verification Plan. The Problem Statement must reference the parent design document. Include a companion validation script.

**Specification references:** FR-001, FR-002 (stage: `specifying`), FR-003 (role: `spec-author`), FR-004, FR-005 (requirements-engineering vocabulary), FR-006 (spec-specific anti-patterns — vague requirements), FR-007, FR-008, FR-009 (5-section template), FR-012 (cross-reference to design), FR-013 (validation script), FR-014, FR-015, NFR-001, NFR-002, NFR-003, NFR-004

**Input context:**
- `.kbz/skills/CONVENTIONS.md` (from Task 1)
- `work/spec/3.0-document-authoring-skill-content.md` FR-009, FR-012 — required sections and cross-reference requirement
- `work/design/skills-system-redesign-v2.md` §5.1 — canonical template definitions
- Example specifications in `work/spec/` — for vocabulary and anti-pattern derivation
- `work/design/skills-system-redesign-v2.md` §9.2 — quality gate checklist

**Output artifacts:**
- `.kbz/skills/write-spec/SKILL.md` — complete skill file with frontmatter (`name: write-spec`, `stage: specifying`, `roles: [spec-author]`, `constraint_level: high`), all attention-curve sections
- `.kbz/skills/write-spec/scripts/validate-spec-structure.sh` — POSIX shell script that checks for 5 required sections AND verifies the Problem Statement contains a design document reference
- `.kbz/skills/write-spec/references/` — overflow content if needed

**Dependencies:** Task 1

**Validation script interface contract:**
```
#!/bin/sh
# Usage: validate-spec-structure.sh <path-to-document>
# Exit 0: all required sections and cross-references present
# Exit 1: missing sections or cross-references (names printed to stdout)
# Required sections: "Problem Statement", "Requirements", "Constraints", "Acceptance Criteria", "Verification Plan"
# Cross-reference check: Problem Statement must contain a reference to a design document (path or document ID)
# Dependencies: POSIX shell utilities only
# Runtime: < 5 seconds
```

---

### Task 4: Author `write-dev-plan/SKILL.md`

**Objective:** Create the implementation plan authoring skill. This skill guides an agent paired with the `architect` role through producing a dev-plan during the `dev-planning` stage. The output template has 5 required sections: Scope, Task Breakdown, Dependency Graph, Risk Assessment, Verification Approach. The Scope section must reference the parent specification. Include a companion validation script.

**Specification references:** FR-001, FR-002 (stage: `dev-planning`), FR-003 (role: `architect`), FR-004, FR-005 (planning vocabulary), FR-006 (planning-specific anti-patterns), FR-007, FR-008, FR-010 (5-section template), FR-012 (cross-reference to specification), FR-013 (validation script), FR-014, FR-015, NFR-001, NFR-002, NFR-003, NFR-004

**Input context:**
- `.kbz/skills/CONVENTIONS.md` (from Task 1)
- `work/spec/3.0-document-authoring-skill-content.md` FR-010, FR-012 — required sections and cross-reference requirement
- `work/design/skills-system-redesign-v2.md` §5.1 — canonical template definitions
- `work/templates/implementation-plan-prompt-template.md` — existing plan template for vocabulary derivation
- Example plans in `work/plan/` — for anti-pattern and example derivation
- `work/design/skills-system-redesign-v2.md` §9.2 — quality gate checklist

**Output artifacts:**
- `.kbz/skills/write-dev-plan/SKILL.md` — complete skill file with frontmatter (`name: write-dev-plan`, `stage: dev-planning`, `roles: [architect]`, `constraint_level: high`), all attention-curve sections
- `.kbz/skills/write-dev-plan/scripts/validate-dev-plan-structure.sh` — POSIX shell script that checks for 5 required sections AND verifies the Scope contains a specification reference
- `.kbz/skills/write-dev-plan/references/` — overflow content if needed

**Dependencies:** Task 1

**Validation script interface contract:**
```
#!/bin/sh
# Usage: validate-dev-plan-structure.sh <path-to-document>
# Exit 0: all required sections and cross-references present
# Exit 1: missing sections or cross-references (names printed to stdout)
# Required sections: "Scope", "Task Breakdown", "Dependency Graph", "Risk Assessment", "Verification Approach"
# Cross-reference check: Scope must contain a reference to a specification (path or document ID)
# Dependencies: POSIX shell utilities only
# Runtime: < 5 seconds
```

---

### Task 5: Author `write-research/SKILL.md`

**Objective:** Create the research report authoring skill. This skill guides an agent paired with the `researcher` role through producing a research report during the `researching` stage. This skill does NOT have a gate-checkable template or validation script (per spec assumptions), but still requires full attention-curve sections including an output format section describing research report structure.

**Specification references:** FR-001, FR-002 (stage: `researching`), FR-003 (role: `researcher`), FR-004, FR-005 (research vocabulary — literature synthesis, evidence grading, methodology, finding), FR-006 (research-specific anti-patterns), FR-007, FR-008, FR-014, FR-015, NFR-001, NFR-002, NFR-003

**Input context:**
- `.kbz/skills/CONVENTIONS.md` (from Task 1)
- `work/spec/3.0-document-authoring-skill-content.md` FR-005 — research vocabulary examples
- `work/design/skills-system-redesign-v2.md` §5.1 — stage definitions
- `work/design/skills-system-redesign-v2.md` §9.2 — quality gate checklist

**Output artifacts:**
- `.kbz/skills/write-research/SKILL.md` — complete skill file with frontmatter (`name: write-research`, `stage: researching`, `roles: [researcher]`, `constraint_level: medium`), all attention-curve sections
- `.kbz/skills/write-research/references/` — overflow content if needed

**Dependencies:** Task 1

---

### Task 6: Author `update-docs/SKILL.md`

**Objective:** Create the documentation update skill. This skill guides an agent paired with the `documenter` role through updating documentation during the `documenting` stage. This skill does NOT have a gate-checkable template or validation script (per spec assumptions), but still requires full attention-curve sections including an output format section describing documentation update conventions.

**Specification references:** FR-001, FR-002 (stage: `documenting`), FR-003 (role: `documenter`), FR-004, FR-005 (documentation vocabulary — information architecture, content currency, cross-reference integrity), FR-006 (documentation-specific anti-patterns), FR-007, FR-008, FR-014, FR-015, NFR-001, NFR-002, NFR-003

**Input context:**
- `.kbz/skills/CONVENTIONS.md` (from Task 1)
- `work/spec/3.0-document-authoring-skill-content.md` FR-005 — documentation vocabulary examples
- `work/design/skills-system-redesign-v2.md` §5.1 — stage definitions
- `.github/copilot-instructions.md`, `AGENTS.md` — current documentation references for tone
- `work/design/skills-system-redesign-v2.md` §9.2 — quality gate checklist

**Output artifacts:**
- `.kbz/skills/update-docs/SKILL.md` — complete skill file with frontmatter (`name: update-docs`, `stage: documenting`, `roles: [documenter]`, `constraint_level: medium`), all attention-curve sections
- `.kbz/skills/update-docs/references/` — overflow content if needed

**Dependencies:** Task 1

---

## 3. Dependency Graph

```
Task 1: Establish Shared Conventions
  │
  ├──→ Task 2: write-design   ─┐
  ├──→ Task 3: write-spec      │
  ├──→ Task 4: write-dev-plan  ├──→ All complete
  ├──→ Task 5: write-research  │
  └──→ Task 6: update-docs    ─┘
```

**Parallelism:** Tasks 2–6 are fully independent and can execute in parallel once Task 1 completes. Maximum parallelism is 5 concurrent agents.

**Serial constraint:** Only Task 1 must complete before any other task begins. There are no inter-dependencies among Tasks 2–6.

---

## 4. Interface Contracts

### 4.1 SKILL.md Frontmatter Schema (shared across all tasks)

Every SKILL.md produced by Tasks 2–6 must use this frontmatter structure:

```yaml
---
name: "{skill-name}"
description:
  expert: "{technical description for direct invocation}"
  natural: "{plain-language description for trigger matching}"
triggers:
  - "{trigger phrase 1}"
  - "{trigger phrase 2}"
roles:
  - "{paired-role-id}"
stage: "{lifecycle-stage}"
constraint_level: "{low|medium|high}"
---
```

### 4.2 Attention-Curve Section Order (shared across all tasks)

Every SKILL.md must contain sections in exactly this order:
1. Frontmatter (YAML)
2. `## Vocabulary` — 15–30 terms, each as `**term** — definition`
3. `## Anti-Patterns` — 5–10 entries, each with Name / Detect / BECAUSE / Resolve
4. `## Checklist` — optional, include for `constraint_level: medium` or `low`
5. `## Procedure` — step-by-step authoring process
6. `## Output Format` — document template with required sections
7. `## Examples` — BAD (with `WHY BAD`) then GOOD (with `WHY GOOD`); GOOD last
8. `## Evaluation Criteria` — 4–8 gradable questions with weight (`required`/`high`/`medium`)
9. `## Questions This Skill Answers` — 5–10 retrieval-anchor questions

### 4.3 Validation Script Interface (shared across Tasks 2, 3, 4)

```
#!/bin/sh
# Arguments: $1 = path to document file
# Exit code: 0 = all checks pass, 1 = one or more checks fail
# Stdout: human-readable list of missing sections/references (on failure)
# Stdout: "All required sections present." (on success)
# Dependencies: POSIX shell only (sh, grep, sed, awk)
# Runtime: < 5 seconds on files up to 2000 lines
```

### 4.4 Vocabulary Non-Overlap Contract

Each skill's vocabulary must not duplicate terms from its paired role. The spec (NFR-002) assigns domain terms to roles and document-type terms to skills. Tasks 2–6 should prefer document-methodology terms (e.g., "design alternative", "traceability matrix") and leave domain terms (e.g., "goroutine", "interface contract") to roles.

---

## 5. Traceability Matrix

| Requirement | Task(s) | Notes |
|-------------|---------|-------|
| FR-001 | 2, 3, 4, 5, 6 | Each task creates one of the five skills |
| FR-002 | 2, 3, 4, 5, 6 | Each task sets the correct `stage` in frontmatter |
| FR-003 | 2, 3, 4, 5, 6 | Each task sets the correct `roles` in frontmatter |
| FR-004 | 1, 2, 3, 4, 5, 6 | Task 1 defines the order; Tasks 2–6 follow it |
| FR-005 | 2, 3, 4, 5, 6 | Each task writes type-specific vocabulary (15–30 terms) |
| FR-006 | 2, 3, 4, 5, 6 | Each task writes type-specific anti-patterns (5–10) |
| FR-007 | 2, 3, 4, 5, 6 | Each task writes BAD/GOOD examples |
| FR-008 | 2, 3, 4, 5, 6 | Each task writes evaluation criteria (4–8 questions) |
| FR-009 | 3 | write-spec template: 5 sections + design cross-ref |
| FR-010 | 4 | write-dev-plan template: 5 sections + spec cross-ref |
| FR-011 | 2 | write-design template: 4 sections |
| FR-012 | 2, 3, 4 | Task 2: no predecessor; Task 3: refs design; Task 4: refs spec |
| FR-013 | 2, 3, 4 | Each of these tasks produces a validation script |
| FR-014 | 1, 2, 3, 4, 5, 6 | Task 1 defines frontmatter template; Tasks 2–6 populate it |
| FR-015 | 2, 3, 4, 5, 6 | Each task writes 5–10 retrieval-anchor questions |
| NFR-001 | 1, 2, 3, 4, 5, 6 | Task 1 provides line-budget guidance; Tasks 2–6 enforce ≤500 lines |
| NFR-002 | 2, 3, 4, 5, 6 | Each task avoids vocabulary overlap with paired role |
| NFR-003 | 2, 3, 4, 5, 6 | Each task ensures one-level-deep reference linking |
| NFR-004 | 2, 3, 4 | Validation scripts: POSIX-only, <5 seconds |