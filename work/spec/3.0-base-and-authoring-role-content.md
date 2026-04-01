# Specification: Base and Authoring Role Content

| Field | Value |
|-------|-------|
| **Feature** | FEAT-01KN588PF5P5Y (base-and-authoring-role-content) |
| **Design** | `work/design/skills-system-redesign-v2.md` §4.1, §4.2, §4.4 |
| **Status** | Draft |

## Overview

This specification defines the required content for eight role YAML files that form the base and authoring layers of the Kanbanzai 3.0 role taxonomy: `base`, `architect`, `spec-author`, `implementer` (abstract), `implementer-go`, `researcher`, `documenter`, and `orchestrator`. Each role file defines the agent's professional identity, domain vocabulary, named anti-patterns, and MCP tool subset. The `base` role provides project-wide identity and conventions inherited by every other role. Authoring roles provide domain-specific expertise for agents that produce documents and code. The `orchestrator` coordination role provides vocabulary and constraints for agents that dispatch and manage other agents.

## Scope

### In Scope

- Content requirements for 8 role YAML files: `base`, `architect`, `spec-author`, `implementer`, `implementer-go`, `researcher`, `documenter`, `orchestrator`
- Required fields, field constraints, and content guidelines for each role
- Inheritance relationships between roles
- Vocabulary payload content requirements per role
- Anti-pattern content requirements per role (name, detect, because, resolve)
- Tool subset declarations per role
- Stage binding associations per role
- Token budget constraint for the `base` role
- Hard constraints carried by the `orchestrator` role

### Explicitly Excluded

- The role YAML schema definition and parsing logic (covered by the Role System feature FEAT-01KN588PCVN4Y)
- Inheritance resolution mechanics (covered by the Role System feature)
- Context assembly pipeline (covered by FEAT-01KN588PE43M6)
- Review roles (`reviewer`, `reviewer-*`) — covered by the Review Role Content specification
- Skill file content — covered by separate skill content specifications
- Binding registry structure and enforcement — covered by FEAT-01KN588PDPE8V
- Implementation details: file paths, YAML serialisation, parsing code, validation code

## Functional Requirements

### FR-001: Common Role Schema

Every role file MUST contain an `id` field, an `identity` field, a `vocabulary` field, an `anti_patterns` field, and a `tools` field. Every role file except `base` MUST also contain an `inherits` field.

**Acceptance criteria:**
- Each of the 8 role files contains all required fields
- The `base` role file does not contain an `inherits` field
- The remaining 7 role files each contain an `inherits` field

### FR-002: Base Role — Project Identity

The `base` role MUST carry the project identity statement "Kanbanzai — Git-native workflow system for human-AI development" (or semantically equivalent wording). The `base` role MUST carry the following hard constraints: "Spec is law," "No scope creep," and "Deterministic YAML serialisation." The `base` role MUST carry commit conventions.

**Acceptance criteria:**
- The `base` role contains a project identity statement that identifies Kanbanzai and its purpose
- The `base` role contains at least the three named hard constraints
- The `base` role contains commit convention guidance

### FR-003: Base Role — Orientation Convention

The `base` role MUST carry the orientation convention: on session start, call `status` to see current project state, then call `next` to see the work queue. This convention MUST be inherited by every other role through the inheritance mechanism.

**Acceptance criteria:**
- The `base` role contains the orientation convention referencing the `status` and `next` tools in that order
- Roles that inherit from `base` receive this convention through inheritance resolution (verified by the Role System feature's inheritance tests)

### FR-004: Base Role — Project-Wide Anti-Patterns

The `base` role MUST carry at least two project-wide anti-patterns: "Flattery Prompting" and "Silent Scope Expansion." Each anti-pattern MUST have `name`, `detect`, `because`, and `resolve` fields. The "Flattery Prompting" anti-pattern MUST reference the PRISM research finding about superlatives degrading domain-specific output quality. The "Silent Scope Expansion" anti-pattern MUST reference the cost of undocumented design decisions during implementation.

**Acceptance criteria:**
- The `base` role contains an `anti_patterns` list with at least 2 entries
- One entry has `name: "Flattery Prompting"` with all four fields populated
- One entry has `name: "Silent Scope Expansion"` with all four fields populated
- The `detect` field for Flattery Prompting references superlatives or praise in prompts
- The `because` field for Flattery Prompting references activation of motivational/marketing text patterns

### FR-005: Base Role — Token Budget

The `base` role's total content MUST fit within approximately 200–300 tokens. This is always-loaded context (progressive disclosure Layer 1) and MUST NOT exceed this budget to preserve context window space for task-specific content.

**Acceptance criteria:**
- The `base` role content, when tokenised, falls within the 200–300 token range (±10% tolerance)
- No general-knowledge explanations are present in the base role (novelty test per design §8.1)

### FR-006: Identity Field Constraints

Every role's `identity` field MUST be a real job title under 50 tokens. The identity MUST NOT contain superlatives ("world-class," "expert," "the best"), flattery, or elaborate backstories. Competence is defined by the vocabulary and anti-patterns, not by adjectives in the identity.

**Acceptance criteria:**
- Each role's `identity` field is under 50 tokens
- No `identity` field contains the words "expert," "world-class," "the best," "excels," or equivalent superlatives
- Each `identity` field is recognisable as a real job title

### FR-007: Vocabulary Field Constraints

Every role's `vocabulary` field MUST be a non-empty list of domain-specific terms. Each term MUST pass the 15-year practitioner test: a senior expert with 15+ years of domain experience would use this exact term when talking with a peer. Vocabulary lists MUST contain between 5 and 30 terms.

**Acceptance criteria:**
- Each role's `vocabulary` field is a list with at least 5 and no more than 30 entries
- No vocabulary entry is a general-knowledge term that a junior developer would use (e.g., "variable," "function," "class")
- Each vocabulary entry is a domain-specific term appropriate to the role's identity

### FR-008: Anti-Pattern Field Structure

Every role's `anti_patterns` field MUST be a list of named anti-patterns. Each anti-pattern MUST have four fields: `name` (human-readable label), `detect` (observable signal that the anti-pattern is occurring), `because` (explanation of why this is harmful — the generalisation mechanism), and `resolve` (concrete corrective action). Anti-pattern lists MUST contain between 2 and 10 entries.

**Acceptance criteria:**
- Each role's `anti_patterns` field is a list with at least 2 and no more than 10 entries
- Every entry has all four fields: `name`, `detect`, `because`, `resolve`
- No `because` field is empty or contains only a restatement of the `detect` field
- Each `resolve` field contains a concrete action, not a vague instruction

### FR-009: Architect Role Content

The `architect` role MUST have `id: architect`, `inherits: base`, and `identity: "Senior software architect"`. Its vocabulary MUST include terms for system decomposition, vertical slice analysis, dependency graph analysis, coupling analysis, blast radius assessment, interface boundary design, and contract-first design. Its anti-patterns MUST include gold plating, premature abstraction, and accidental coupling. The role MUST be associated with the `designing` and `dev-planning` stages.

**Acceptance criteria:**
- The `architect` role contains the specified `id`, `inherits`, and `identity` values
- The vocabulary contains at least: "system decomposition," "vertical slice," "dependency graph," "coupling analysis," "blast radius assessment," "interface boundary"
- The anti-patterns contain entries named "Gold plating" (or equivalent), "Premature abstraction," and "Accidental coupling"
- The role is declared for use in stages `designing` and `dev-planning`

### FR-010: Spec-Author Role Content

The `spec-author` role MUST have `id: spec-author`, `inherits: base`, and `identity: "Senior requirements engineer"`. Its vocabulary MUST include terms for acceptance criteria (Given/When/Then), requirement traceability, testable assertion, boundary condition, and specification completeness. Its anti-patterns MUST include untestable requirement, implicit assumption, scope ambiguity, over-specification, and under-specification. The role MUST be associated with the `specifying` stage.

**Acceptance criteria:**
- The `spec-author` role contains the specified `id`, `inherits`, and `identity` values
- The vocabulary contains terms for acceptance criteria format, requirement traceability, and testable assertions
- The anti-patterns list contains at least 5 entries covering the named failure modes
- The role is declared for use in stage `specifying`

### FR-011: Implementer and Implementer-Go Role Content

The `implementer` role MUST be an abstract role with `id: implementer` and `inherits: base`. It MUST serve as the parent for project-specific implementation roles. The `implementer-go` role MUST have `id: implementer-go`, `inherits: implementer`, and `identity: "Senior Go engineer"`. The `implementer-go` vocabulary MUST include terms for goroutine leak, interface segregation, error wrapping (%w), table-driven test, struct embedding, functional option pattern, context propagation, and zero-value usability. Its anti-patterns MUST include god struct, interface pollution, init() coupling, naked goroutine, and error swallowing. The `implementer-go` role MUST be associated with the `developing` stage.

**Acceptance criteria:**
- The `implementer` role exists with `id: implementer` and `inherits: base`
- The `implementer-go` role exists with `inherits: implementer` (two levels of inheritance: implementer-go → implementer → base)
- The `implementer-go` vocabulary contains at least 8 Go-specific terms from the design
- The `implementer-go` anti-patterns contain at least 5 entries covering Go-specific failure modes
- The `implementer-go` role is declared for use in stage `developing`

### FR-012: Researcher Role Content

The `researcher` role MUST have `id: researcher`, `inherits: base`, and `identity: "Senior technical analyst"`. Its vocabulary MUST include terms for literature review, evidence synthesis, citation traceability, confidence assessment, and counter-evidence. Its anti-patterns MUST include cherry-picking, false equivalence, and unsupported generalisation. The role MUST be associated with the `researching` stage.

**Acceptance criteria:**
- The `researcher` role contains the specified `id`, `inherits`, and `identity` values
- The vocabulary contains at least 5 research-domain terms from the design
- The anti-patterns contain at least 3 entries covering research methodology failure modes
- The role is declared for use in stage `researching`

### FR-013: Documenter Role Content

The `documenter` role MUST have `id: documenter`, `inherits: base`, and `identity: "Senior technical writer"`. Its vocabulary MUST include terms for progressive disclosure, information architecture, cross-reference integrity, terminology consistency, and reading order optimisation. Its anti-patterns MUST include documentation-code divergence, outdated example, assumed knowledge, and documentation duplication. The role MUST be associated with the `documenting` stage.

**Acceptance criteria:**
- The `documenter` role contains the specified `id`, `inherits`, and `identity` values
- The vocabulary contains at least 5 technical writing terms from the design
- The anti-patterns contain at least 4 entries covering documentation failure modes
- The role is declared for use in stage `documenting`

### FR-014: Orchestrator Role Content

The `orchestrator` role MUST have `id: orchestrator`, `inherits: base`, and `identity: "Senior engineering manager coordinating an agent team"`. Its vocabulary MUST be organised into four categories: dispatch mechanics, workflow governance, quality assessment, and pattern matching. The dispatch mechanics vocabulary MUST include terms for task decomposition, handoff protocol, parallel dispatch, conflict detection, dependency ordering, and remediation routing. The workflow governance vocabulary MUST include terms for lifecycle gate, stage prerequisite, hard constraint (ℋ), and soft constraint (𝒮). The quality assessment vocabulary MUST include terms for decomposition quality, vertical slice completeness, and review verdict. The pattern matching vocabulary MUST include terms for sequential reasoning penalty, parallelisable task, and orchestrator-workers parallel.

**Acceptance criteria:**
- The `orchestrator` role contains the specified `id`, `inherits`, and `identity` values
- The vocabulary contains terms in all four named categories
- At least 10 vocabulary terms are present across the categories
- The vocabulary includes the symbolic notation for hard constraint (ℋ) and soft constraint (𝒮)

### FR-015: Orchestrator Anti-Patterns

The `orchestrator` role MUST carry at least 7 anti-patterns: over-decomposition, under-decomposition, context forwarding, result-without-evidence, reactive communication, premature delegation, and infinite refinement loop. The "Reactive communication" anti-pattern MUST reference the Masters et al. finding about proactive orchestrators. The "Premature delegation" anti-pattern MUST reference the Google Research finding about multi-agent coordination degrading sequential reasoning by 39–70%. The "Infinite refinement loop" anti-pattern MUST reference the `max_review_cycles` threshold and recommend escalation to a human checkpoint.

**Acceptance criteria:**
- The `orchestrator` role contains at least 7 anti-pattern entries
- Each anti-pattern has all four required fields (name, detect, because, resolve)
- The reactive communication entry references the comparative statistic (14.5× decomposition, 26× dependency tracking)
- The premature delegation entry references the 39–70% degradation finding
- The infinite refinement loop entry references `max_review_cycles` and human checkpoint escalation

### FR-016: Orchestrator Hard Constraints

The `orchestrator` role MUST carry three hard constraints as explicit, non-negotiable rules: (1) the 45% context utilisation threshold, (2) agent saturation at 4 for specialist panels, and (3) the cascade pattern. These MUST be treated as hard constraints by any agent assigned the orchestrator role — they are decision boundaries, not suggestions.

**Acceptance criteria:**
- The `orchestrator` role content contains the 45% threshold as an explicit constraint
- The `orchestrator` role content contains the saturation limit of 4 agents as an explicit constraint
- The `orchestrator` role content contains the cascade pattern as an explicit constraint
- These three items are marked or positioned as hard constraints, distinguishable from general vocabulary

### FR-017: Orchestrator Stage Associations

The `orchestrator` role MUST be associated with the `developing` stage (as the coordinator dispatching implementers) and the `reviewing` stage (as the coordinator dispatching reviewers). The role MUST NOT be associated with single-agent stages (designing, specifying, researching, documenting).

**Acceptance criteria:**
- The `orchestrator` role is declared for use in stages `developing` and `reviewing`
- The `orchestrator` role is not declared for stages `designing`, `specifying`, `researching`, or `documenting`

## Non-Functional Requirements

### NFR-001: Novelty Test Compliance

Every paragraph of content in every role file MUST pass the novelty test (design §8.1): it must contain information the model does not already know. General-knowledge explanations (what a state machine is, how Go interfaces work, what YAML is) MUST NOT appear in any role file.

**Acceptance criteria:**
- No role file contains explanations of general programming concepts, language features, or widely-known methodologies
- Every content element is specific to the Kanbanzai project or represents domain vocabulary that routes model attention

### NFR-002: Tone and Explanatory Style

Anti-pattern `because` clauses and any instructional content MUST use an explanatory tone rather than unexplained imperatives. Content MUST say "Do X because Y" rather than "ALWAYS do X" or "NEVER do Y" without explanation. This follows design principle DP-4.

**Acceptance criteria:**
- No anti-pattern entry lacks a substantive `because` clause
- No instructional content consists solely of bare imperatives without rationale

### NFR-003: Terminology Consistency

Each role's vocabulary payload defines the canonical terms for its domain. Within the role file's own content — anti-patterns, constraints, any prose — those canonical terms MUST be used exclusively. Synonyms MUST NOT be alternated (e.g., if the vocabulary says "finding," the anti-patterns must not say "issue" or "problem").

**Acceptance criteria:**
- Within each role file, terms used in prose match the vocabulary entries
- No synonyms are used for terms that appear in the vocabulary list

### NFR-004: Lean Content

Each role file MUST follow design principle DP-6 (lean by default). Vocabulary lists MUST contain 5–30 terms. Anti-pattern lists MUST contain 2–10 entries. Content MUST be concise — no role file (other than `base` which has the 200–300 token budget, and `orchestrator` which has extensive vocabulary) should exceed approximately 500 tokens.

**Acceptance criteria:**
- All vocabulary lists are within the 5–30 term range
- All anti-pattern lists are within the 2–10 entry range
- No role file contains redundant or inflated content

### NFR-005: No Implementation Leakage

Role files MUST NOT contain implementation details, code examples, technology choices (beyond what is inherent in the identity — e.g., `implementer-go` referencing Go is appropriate), or file path references. Role files define "who you are," not "how to do things."

**Acceptance criteria:**
- No role file contains code snippets, file paths, or references to specific implementation mechanisms
- No role file contains procedural instructions (procedures belong in skills)

## Acceptance Criteria

The acceptance criteria for each requirement are listed inline with each FR and NFR above. The following are aggregate acceptance criteria for the specification as a whole:

1. **Completeness:** All 8 role files are authored with all required fields populated.
2. **Inheritance integrity:** The inheritance chain is valid: `implementer-go` → `implementer` → `base`, all authoring roles → `base`, `orchestrator` → `base`.
3. **No duplication:** Content that belongs in `base` does not appear duplicated in child roles. Child roles carry only ADDITIONAL content.
4. **Vocabulary routing verification:** When a role's vocabulary is loaded into context, it activates domain-appropriate knowledge in the model — verified by running representative tasks with and without the role context and comparing output quality.
5. **Anti-pattern effectiveness:** When a role's anti-patterns are loaded into context, the model avoids the named failure modes — verified by running tasks designed to trigger each anti-pattern and confirming the model self-corrects or avoids the pattern.

## Dependencies and Assumptions

### Dependencies

- **Role System feature (FEAT-01KN588PCVN4Y):** Defines the YAML schema that these role files must conform to, including the inheritance resolution mechanism. This specification assumes the schema supports all fields referenced here (`id`, `inherits`, `identity`, `vocabulary`, `anti_patterns`, `tools`).
- **Binding Registry feature (FEAT-01KN588PDPE8V):** Defines the stage-to-role mappings. This specification declares which stages each role is associated with, but the binding registry enforces those associations.
- **Context Assembly Pipeline (FEAT-01KN588PE43M6):** Defines how role content is merged during inheritance resolution and how the assembled context is ordered. This specification assumes vocabulary and anti-patterns from parent and child roles are merged (not replaced) during assembly.

### Assumptions

1. The role YAML schema supports all field types referenced in this specification (string identity, list vocabulary, structured anti-pattern entries with four fields, list tools).
2. Inheritance resolution merges `vocabulary` and `anti_patterns` lists additively — a child role's entries are appended to the parent's, not substituted.
3. The `tools` field accepts a list of MCP tool names and is used to scope tool availability during context assembly.
4. The `base` role is the root of the inheritance hierarchy — it has no parent.
5. The `implementer` role can function as an abstract parent (providing shared implementer conventions) even though it does not carry a project-specific identity. Concrete implementer roles (e.g., `implementer-go`) inherit from it and provide the specific identity and vocabulary.
6. Token counts referenced in this specification (200–300 for `base`, 50 for identity) use the approximate tokenisation of modern LLMs (GPT-4/Claude class). Exact counts may vary by model.