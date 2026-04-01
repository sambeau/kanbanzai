# Specification: Review Skill Content

| Field | Value |
|-------|-------|
| **Feature** | FEAT-01KN588PGJNM0 (review-skill-content) |
| **Design** | `work/design/skills-system-redesign-v2.md` §3.2, §5.3, §5.4 |
| **Status** | Draft |

## Overview

This specification defines the required content for three SKILL.md files that form the review layer of the Kanbanzai 3.0 skill catalog: `review-code`, `orchestrate-review`, and `review-plan`. Each skill file follows the attention-curve-optimised SKILL.md format defined in the design. `review-code` is the sub-agent review skill used by ALL specialist reviewers — specialisation comes from the role's vocabulary, not from the skill. `orchestrate-review` is the coordinator skill that decomposes a feature into review units, dispatches specialist sub-agents in parallel, collates findings, and routes to remediation or approval. `review-plan` is the plan-level review skill that checks whether all features shipped, specs are approved, and documentation is current. Together, these three skills implement the composition model where the same `review-code` procedure combined with different `reviewer-*` roles produces different expertise lenses on the same code, coordinated by the `orchestrate-review` orchestrator.

## Scope

### In Scope

- Content requirements for 3 SKILL.md files: `review-code`, `orchestrate-review`, `review-plan`
- Required frontmatter fields and their values for each skill
- Attention-curve section ordering requirements for SKILL.md files
- Vocabulary payload content requirements per skill (task-specific terms that combine with role vocabulary)
- Anti-pattern content requirements per skill (task-specific anti-patterns that combine with role anti-patterns)
- Checklist requirements for workflow-critical review skills
- Procedure structure and required steps for each skill
- Output format requirements for structured review deliverables
- Example content requirements (BAD/GOOD pairs with WHY explanations)
- Evaluation criteria requirements (gradable questions with weights)
- Retrieval anchor requirements ("Questions This Skill Answers" section)
- The composition model: `review-code` (one skill) + different `reviewer-*` roles = different review lenses
- Adaptive composition constraints for `orchestrate-review`
- Constraint level declarations per skill

### Explicitly Excluded

- The SKILL.md schema definition and parsing logic (covered by the Skill System feature FEAT-01KN588PDBW85)
- Review role content (`reviewer`, `reviewer-*`) — covered by the Review Role Content specification
- Binding registry structure and enforcement — covered by FEAT-01KN588PDPE8V
- Context assembly pipeline — covered by FEAT-01KN588PE43M6
- Implementation skills (`implement-task`, `orchestrate-development`) — covered by FEAT-01KN588PG7HA3
- Document authoring skills (`write-spec`, `write-design`, etc.) — covered by FEAT-01KN588PFWADY
- Skill file directory layout, `references/` and `scripts/` subdirectory structure — covered by the Skill System feature
- Implementation details: file paths, parsing code, validation code, test fixtures
- Gate enforcement mechanisms — covered by the workflow and tooling design

## Functional Requirements

### FR-001: SKILL.md Attention-Curve Section Ordering

Every review skill SKILL.md file MUST follow the attention-curve-optimised section ordering defined in the design. The body sections MUST appear in this order: Vocabulary → Anti-Patterns → Checklist (if applicable) → Procedure → Output Format → Examples → Evaluation Criteria → Questions This Skill Answers. Vocabulary MUST appear first (highest-attention position). Evaluation criteria and retrieval anchors MUST appear last (high-attention end-of-context position). Procedure MUST appear in the middle zone.

**Acceptance criteria:**
- Each of the 3 SKILL.md files has body sections in the prescribed order
- No skill file places Procedure before Vocabulary or Anti-Patterns
- No skill file places Evaluation Criteria or Questions before Examples
- The section ordering matches the attention curve: high → medium-high → medium → rising → high

### FR-002: SKILL.md Frontmatter Requirements

Every review skill SKILL.md file MUST contain YAML frontmatter with the following fields: `name`, `description` (with both `expert` and `natural` sub-fields), `triggers` (list of natural-language trigger phrases), `roles` (list of compatible role IDs), `stage` (workflow stage), and `constraint_level` (one of: low, medium, high). The `expert` description MUST activate deep domain knowledge on direct invocation. The `natural` description MUST trigger on casual phrasing. Both descriptions MUST include what the skill does and when to use it.

**Acceptance criteria:**
- Each of the 3 SKILL.md files contains YAML frontmatter with all 6 required fields
- Every `description` field has both `expert` and `natural` sub-fields
- Every `triggers` field is a non-empty list of natural-language phrases
- Every `roles` field is a non-empty list of valid role IDs
- Every `stage` field is a valid workflow stage name
- Every `constraint_level` field is one of: low, medium, high

### FR-003: review-code — Frontmatter Values

The `review-code` skill MUST have `name: review-code`, `stage: reviewing`, and `constraint_level: medium`. Its `roles` field MUST list all specialist reviewer roles: `reviewer`, `reviewer-conformance`, `reviewer-quality`, `reviewer-security`, `reviewer-testing`. Its `expert` description MUST reference multi-dimension code review, classified findings, and evidence-backed verdicts against acceptance criteria. Its `natural` description MUST describe reviewing code changes against a spec and producing a structured report. Its `triggers` MUST include phrases for reviewing code changes, evaluating implementation against spec, and checking code quality.

**Acceptance criteria:**
- The frontmatter contains `name: review-code`, `stage: reviewing`, `constraint_level: medium`
- The `roles` field lists all 5 reviewer role IDs
- The `expert` description mentions multi-dimension review, classified findings, and evidence-backed verdicts
- The `natural` description is understandable without domain expertise
- The `triggers` list contains at least 3 natural-language trigger phrases

### FR-004: review-code — Vocabulary Payload

The `review-code` skill MUST carry a task-specific vocabulary payload that combines with the role's vocabulary during context assembly. The vocabulary MUST include: finding classification (blocking, non-blocking), evidence-backed verdict, acceptance criteria traceability, per-dimension outcome (pass, pass_with_notes, concern, fail), review unit decomposition, structured review output, remediation recommendation, and spec conformance gap. These terms are specific to the review task methodology — they complement (not duplicate) the role's domain vocabulary.

**Acceptance criteria:**
- The Vocabulary section contains at least 8 task-specific review methodology terms
- The vocabulary includes the per-dimension outcome scale (pass, pass_with_notes, concern, fail)
- The vocabulary includes finding classification with its two values (blocking, non-blocking)
- No vocabulary term duplicates a term that belongs in a reviewer role's vocabulary (e.g., OWASP, cyclomatic complexity — those are role terms, not skill terms)
- Each term passes the 15-year practitioner test

### FR-005: review-code — Anti-Patterns

The `review-code` skill MUST carry at least 5 task-specific anti-patterns: "Rubber-Stamp Review" (referencing MAST FM-3.1), "Severity Inflation," "Dimension Bleed," "Prose Commentary," and "Missing Spec Anchor." Each anti-pattern MUST have a heading with the name, and sub-entries for Detect, BECAUSE, and Resolve. The "Rubber-Stamp Review" anti-pattern MUST reference FM-3.1 as the primary quality failure in multi-agent systems and MUST identify LLM sycophancy as the root cause. The "Missing Spec Anchor" anti-pattern MUST require every blocking finding to cite a specific spec requirement, with the rationale that without a spec anchor the finding is an opinion rather than a conformance gap.

**Acceptance criteria:**
- The Anti-Patterns section contains at least 5 named anti-patterns
- Every anti-pattern has Detect, BECAUSE, and Resolve sub-entries
- "Rubber-Stamp Review" references MAST FM-3.1 and LLM sycophancy
- "Severity Inflation" includes a detection threshold (e.g., >40% blocking findings)
- "Dimension Bleed" describes cross-dimension verdict contamination
- "Prose Commentary" detects qualitative prose replacing structured findings
- "Missing Spec Anchor" requires spec requirement citations for blocking findings
- Anti-patterns are task-specific (review methodology failures), not domain-specific (those belong in roles)

### FR-006: review-code — Checklist

The `review-code` skill MUST include a Checklist section because the skill has `constraint_level: medium`. The checklist MUST be a copyable list of items that agents reproduce in their response and check off as they progress. The checklist MUST include at minimum: reading the spec section(s), reading all files in the file list, confirming the review profile and required dimensions, evaluating each required dimension independently, classifying all findings, verifying every blocking finding has a spec anchor, and producing structured output in the required format.

**Acceptance criteria:**
- A Checklist section is present between Anti-Patterns and Procedure
- The checklist is formatted as a copyable checkbox list (e.g., `- [ ] item`)
- The checklist contains at least 7 items covering the full review workflow
- The checklist includes an item for reading the spec
- The checklist includes an item for reading the implementation files
- The checklist includes an item for independent per-dimension evaluation
- The checklist includes an item for verifying spec anchors on blocking findings
- The checklist includes an item for producing the structured output

### FR-007: review-code — Procedure

The `review-code` skill MUST include a Procedure section with at least 3 numbered steps: (1) Orient from inputs, (2) Evaluate each dimension independently, and (3) Validate and iterate. Step 1 MUST require reading the spec section(s), reading all files in the file list, noting the review profile, and STOPPING if any input is missing or the spec is ambiguous (referencing the uncertainty protocol from design §8.3). Step 2 MUST require working through each dimension's evaluation questions and recording a per-dimension outcome, with an explicit instruction that one dimension's result MUST NOT affect another's assessment. Step 3 MUST include a validate → fix → repeat loop: validate findings against classification criteria, check that every blocking finding cites a specific spec requirement, check that no dimension verdict is influenced by another, fix any issues found, and re-validate until all findings pass. The procedure MUST NOT proceed past validation failure without correction.

**Acceptance criteria:**
- The Procedure section contains at least 3 numbered steps
- Step 1 includes an explicit STOP instruction for missing inputs or ambiguous specs
- Step 1 references the uncertainty protocol (do not infer intent)
- Step 2 explicitly requires independent per-dimension evaluation
- Step 3 includes a validate → fix → re-validate iteration loop
- The procedure does not allow proceeding past validation failure
- The procedure uses numbered steps with IF/THEN conditions (structured to survive attention degradation)

### FR-008: review-code — Output Format

The `review-code` skill MUST define a structured output format for review deliverables. The format MUST include: review unit identification, overall verdict, and per-dimension outcomes. Each dimension outcome MUST include the dimension name, a verdict (pass, pass_with_notes, concern, or fail), and evidence citations. Findings MUST be classified as blocking or non-blocking. Blocking findings MUST include a spec requirement reference (the spec anchor). The format MUST be machine-parseable — a machine MUST be able to extract all findings from the output without ambiguity.

**Acceptance criteria:**
- An Output Format section is present in the skill
- The format includes a field for review unit identification
- The format includes an overall verdict field
- The format includes per-dimension outcomes with dimension name, verdict, and evidence
- The format distinguishes blocking from non-blocking findings
- Blocking findings include a spec requirement reference
- The format uses a consistent structure that could be parsed programmatically

### FR-009: review-code — Examples

The `review-code` skill MUST include at least 3 examples in BAD/GOOD pairs with WHY explanations. The examples MUST include: (1) a BAD rubber-stamp example with prose and no evidence, (2) a GOOD evidence-backed structured review with per-dimension verdicts and findings, and (3) a GOOD clearance example with zero findings but substantive per-dimension evidence. Each example MUST be followed by a WHY explanation stating what makes it bad or good. The GOOD clearance example MUST demonstrate that "approved" is not a rubber stamp when backed by per-dimension evidence — the reviewer demonstrably examined the code. The best example MUST appear last to exploit recency bias.

**Acceptance criteria:**
- The Examples section contains at least 3 examples
- One example is labelled BAD and demonstrates a rubber-stamp with prose and no structured findings
- One example is labelled GOOD and demonstrates evidence-backed review with per-dimension verdicts, specific evidence citations, and classified findings
- One example is labelled GOOD and demonstrates legitimate clearance (zero findings, substantive evidence per dimension)
- Every example is followed by a WHY explanation
- The final example in the section is a GOOD example (recency bias exploitation)

### FR-010: review-code — Evaluation Criteria

The `review-code` skill MUST include an Evaluation Criteria section containing gradable questions with weights. The criteria MUST be separated from the procedure — they are for evaluating the skill's OUTPUT, not for the agent to self-evaluate during execution. The criteria MUST include at minimum: (1) does every dimension have an explicit outcome (weight: required), (2) does every blocking finding cite a specific spec requirement (weight: required), (3) can a machine extract all findings without ambiguity (weight: high), (4) are dimensions evaluated independently with no bleed (weight: high), (5) does the output distinguish blocking from non-blocking findings (weight: required), and (6) is every "approved" verdict backed by per-dimension evidence (weight: high). The gradable question format and weight scale MUST support use by an LLM-as-judge automated evaluation pass.

**Acceptance criteria:**
- An Evaluation Criteria section is present after Examples
- The section contains at least 6 gradable questions
- At least 3 criteria are weighted as "required"
- Criteria are phrased as yes/no questions that can be graded by an external evaluator
- Criteria do not duplicate procedural instructions (they evaluate output, not process)
- The weight scale is consistent across all criteria (e.g., required, high, medium)

### FR-011: review-code — Retrieval Anchors

The `review-code` skill MUST include a "Questions This Skill Answers" section as the final section in the body. This section MUST contain at least 6 questions that serve as retrieval anchors for skill selection. The questions MUST cover: how to review code against a spec, what dimensions to evaluate, how to classify findings, what output format to use, when to escalate to a human checkpoint, and what a well-evidenced "approved" verdict looks like.

**Acceptance criteria:**
- A "Questions This Skill Answers" section is present as the last body section
- The section contains at least 6 questions
- The questions cover the key decision points an agent faces during code review
- The questions use natural language that an orchestrator or system would use when selecting a skill

### FR-012: orchestrate-review — Frontmatter and Purpose

The `orchestrate-review` skill MUST have `name: orchestrate-review`, `stage: reviewing`, and `roles: [orchestrator]`. It MUST be the coordinator skill for the review stage. Its procedure MUST cover: decomposing a feature into review units, dispatching specialist sub-agents (each receiving the `review-code` skill with a different reviewer role), collating findings from all sub-agents, deduplicating overlapping findings, and routing to either remediation (if blocking findings exist) or approval (if no blocking findings remain). The skill MUST carry vocabulary for orchestration-specific concerns (review unit decomposition, finding collation, verdict aggregation, remediation routing, dispatch protocol) and anti-patterns for orchestration failures.

**Acceptance criteria:**
- The frontmatter contains `name: orchestrate-review`, `stage: reviewing`, `roles: [orchestrator]`
- The procedure describes decomposition of a feature into review units
- The procedure describes dispatching specialist sub-agents with the `review-code` skill
- The procedure describes collating and deduplicating findings
- The procedure describes routing decisions (remediation vs approval)
- The skill has a Vocabulary section with orchestration-specific terms
- The skill has an Anti-Patterns section with orchestration-specific failure modes

### FR-013: orchestrate-review — Adaptive Composition

The `orchestrate-review` skill MUST implement adaptive composition for reviewer dispatch. The binding registry declares a maximum of 4 sub-agents, but the orchestrator MUST NOT always dispatch all 4. The skill MUST instruct the orchestrator to select reviewers based on the files changed: if no security-relevant code changed, the security reviewer MUST NOT be dispatched. Small features (≤10 files) MAY only need 1–2 reviewers. The skill MUST include guidance for when each specialist is warranted and when it can be omitted.

**Acceptance criteria:**
- The skill explicitly states that the maximum of 4 sub-agents is not a target — fewer may be dispatched
- The skill provides criteria for when to include or exclude each specialist reviewer type
- The skill references the change scope (files changed, code areas affected) as the selection input
- The skill describes the small-feature case (≤10 files, 1–2 reviewers may suffice)
- The adaptive composition guidance references the Captain Agent research finding (15–25% improvement over static teams)

### FR-014: orchestrate-review — Anti-Patterns

The `orchestrate-review` skill MUST carry at least 3 anti-patterns specific to review orchestration: "Result-without-evidence" (accepting sub-agent output without checking for evidence), "Over-decomposition" (splitting the review into too many review units, creating overhead), and at least one additional orchestration-specific failure mode. Each anti-pattern MUST have Detect, BECAUSE, and Resolve entries.

**Acceptance criteria:**
- The Anti-Patterns section contains at least 3 anti-patterns
- Every anti-pattern has Detect, BECAUSE, and Resolve entries
- "Result-without-evidence" detects accepting sub-agent review output that lacks per-dimension evidence or spec citations
- "Over-decomposition" detects splitting a feature into more review units than the code warrants
- No anti-pattern duplicates one that belongs in a reviewer role or in the `review-code` skill

### FR-015: review-plan — Frontmatter and Purpose

The `review-plan` skill MUST have `name: review-plan`, `stage: plan-reviewing`, and `roles: [reviewer-conformance]`. It MUST handle plan-level review: checking that all features within the plan have shipped, that all specifications are approved, and that documentation is current. The skill MUST be conformance-focused — it verifies completeness and approval status, not code quality or security. The skill MUST follow the attention-curve SKILL.md format with all required sections.

**Acceptance criteria:**
- The frontmatter contains `name: review-plan`, `stage: plan-reviewing`, `roles: [reviewer-conformance]`
- The skill's procedure checks feature completion status across the plan
- The skill's procedure checks specification approval status
- The skill's procedure checks documentation currency
- The skill follows the attention-curve section ordering (Vocabulary → Anti-Patterns → Procedure → Output Format → Examples → Evaluation Criteria → Questions)
- The skill is conformance-focused (no code quality or security review content)

## Non-Functional Requirements

### NFR-001: Novelty Test Compliance

Every paragraph of content in every review skill file MUST pass the novelty test (design §8.1). General explanations of what code review is, how structured output works, or what acceptance criteria are MUST NOT appear. Content MUST be limited to Kanbanzai-specific review methodology, project-specific vocabulary, and project-specific anti-patterns with concrete detection signals and resolution steps.

**Acceptance criteria:**
- No skill file contains explanations of general code review concepts
- No skill file explains what structured output or machine-parseable formats are in general terms
- Every content element is specific to the Kanbanzai review methodology or carries project-specific guidance

### NFR-002: SKILL.md Body Under 500 Lines

Every review skill SKILL.md file body MUST be under 500 lines. Extended anti-pattern documentation, additional examples, and detailed evaluation rubrics MUST be placed in the skill's `references/` directory if they would cause the body to exceed this limit. Reference files MUST be linked directly from SKILL.md (one level deep — no reference-to-reference chains).

**Acceptance criteria:**
- Each of the 3 SKILL.md files is under 500 lines
- If any skill requires overflow content, it is placed in `references/` and linked from SKILL.md
- No reference file links to another reference file

### NFR-003: Composition Model Integrity

The `review-code` skill MUST NOT contain domain-specific vocabulary or anti-patterns that belong in reviewer roles. The skill provides the review procedure and methodology — the role provides the expertise lens. This separation MUST be maintained so that the same `review-code` skill combined with different `reviewer-*` roles produces meaningfully different review outputs. If a term is specific to security (e.g., OWASP), quality (e.g., cyclomatic complexity), or testing (e.g., boundary value analysis), it belongs in the role, not the skill.

**Acceptance criteria:**
- The `review-code` vocabulary contains only review methodology terms, not domain-specific terms
- The `review-code` anti-patterns address review process failures, not domain-specific failures
- Running `review-code` with `reviewer-security` produces security-focused output while running it with `reviewer-testing` produces testing-focused output — verified by the vocabulary difference in the assembled context

### NFR-004: Tone and Explanatory Style

Anti-pattern BECAUSE clauses and all instructional content MUST use an explanatory tone. Content MUST explain WHY a pattern is harmful in a way that enables the model to generalise to adjacent cases. Bare imperatives without rationale MUST NOT appear. The tone follows design principle DP-4.

**Acceptance criteria:**
- Every BECAUSE clause provides a substantive explanation, not just a restatement of the Detect signal
- Every Resolve clause provides a concrete corrective action
- No instructional content consists of unexplained imperatives

### NFR-005: Uncertainty Protocol Inclusion

Every review skill that produces work output MUST include an explicit uncertainty instruction positioned early in the procedure (high-attention zone). The instruction MUST direct agents to STOP and report ambiguity rather than infer intent when the specification is ambiguous, incomplete, or contradictory. This follows design §8.3.

**Acceptance criteria:**
- The `review-code` procedure includes a STOP instruction for missing or ambiguous inputs in Step 1
- The `orchestrate-review` procedure includes a STOP instruction for missing prerequisites
- The `review-plan` procedure includes a STOP instruction for incomplete or contradictory plan state
- Each STOP instruction explicitly grants permission to report uncertainty rather than guess

### NFR-006: Constraint Level Alignment

Each skill's procedure style MUST match its declared `constraint_level`. Skills with `constraint_level: medium` MUST use templates with bounded flexibility — a preferred pattern exists but variation is acceptable within bounds. Skills with `constraint_level: low` MUST provide exact sequences. Skills with `constraint_level: high` MUST provide principles and vocabulary without rigid steps.

**Acceptance criteria:**
- `review-code` (medium) uses a structured procedure with numbered steps and a checklist, but allows reviewer judgement within dimensions
- `orchestrate-review`'s constraint level matches its procedure style
- `review-plan`'s constraint level matches its procedure style
- No skill has a mismatch between its declared constraint level and its procedural rigidity

### NFR-007: Terminology Consistency

Each skill's vocabulary payload defines the canonical terms for the review methodology. Within the skill's own content — anti-patterns, procedure, examples, evaluation criteria — those canonical terms MUST be used exclusively. If the vocabulary says "finding," the procedure MUST NOT use "issue" or "problem." If the vocabulary says "per-dimension outcome," the examples MUST NOT use "per-dimension result" or "per-dimension score."

**Acceptance criteria:**
- Within each skill file, terms used in prose match the vocabulary entries
- No synonyms are used for terms that appear in the vocabulary list
- The examples use the same terminology as the vocabulary and procedure

## Acceptance Criteria

The acceptance criteria for each requirement are listed inline with each FR and NFR above. The following are aggregate acceptance criteria for the specification as a whole:

1. **Completeness:** All 3 SKILL.md files are authored with all required frontmatter fields and body sections in the correct attention-curve order.
2. **Composition model verification:** Running `review-code` with `reviewer-security` produces security-focused review output, while running it with `reviewer-quality` produces quality-focused review output. The difference comes entirely from the role vocabulary, not from the skill content. Verified by running representative review tasks with different role-skill combinations and comparing the vocabulary and focus areas in the output.
3. **Orchestration workflow:** The `orchestrate-review` skill, when used with the `orchestrator` role, successfully decomposes a feature into review units, dispatches specialist sub-agents with appropriate roles, collates findings, and produces an aggregate verdict. Verified by running the orchestrated review workflow on a representative feature.
4. **Adaptive dispatch:** The `orchestrate-review` skill dispatches fewer than 4 reviewers when the change scope does not warrant all specialists. Verified by running orchestrated review on a small feature with no security-relevant changes and confirming the security reviewer is not dispatched.
5. **Anti-pattern effectiveness:** When the review skill anti-patterns are loaded into context, the model avoids the named failure modes. Verified by running review tasks designed to trigger each anti-pattern (e.g., a task that invites a rubber-stamp response) and confirming the model self-corrects or avoids the pattern.
6. **Structured output parsability:** The output produced by `review-code` follows the defined output format and can be parsed by a machine to extract all findings, their classifications, their spec anchors, and per-dimension verdicts without ambiguity.
7. **Plan review coverage:** The `review-plan` skill checks all required plan-level conformance criteria: feature completion, spec approval status, and documentation currency.

## Dependencies and Assumptions

### Dependencies

- **Skill System feature (FEAT-01KN588PDBW85):** Defines the SKILL.md format, frontmatter schema, directory layout (`references/`, `scripts/`), and the attention-curve section ordering. This specification assumes the skill system supports all frontmatter fields referenced here (`name`, `description`, `triggers`, `roles`, `stage`, `constraint_level`).
- **Review Role Content (FEAT-01KN588PFG6GY):** Defines the 5 reviewer roles (`reviewer`, `reviewer-conformance`, `reviewer-quality`, `reviewer-security`, `reviewer-testing`) that compose with the `review-code` skill. The composition model depends on both the roles defined there and the skills defined here.
- **Base and Authoring Role Content (FEAT-01KN588PF5P5Y):** Defines the `orchestrator` role that composes with the `orchestrate-review` skill.
- **Binding Registry feature (FEAT-01KN588PDPE8V):** Defines the stage-to-skill mappings and declares `max_agents: 4` for the reviewing stage. This specification declares adaptive composition within that maximum, but the binding registry enforces the ceiling.
- **Context Assembly Pipeline (FEAT-01KN588PE43M6):** Defines how skill vocabulary and anti-patterns are merged with role vocabulary and anti-patterns during context assembly, and how the assembled context is ordered by the attention curve. The composition model (same skill + different roles = different expertise lenses) depends on this merge behaviour.

### Assumptions

1. The skill system supports the SKILL.md format with YAML frontmatter and Markdown body sections as described in design §3.2.
2. Context assembly merges skill vocabulary with role vocabulary and skill anti-patterns with role anti-patterns additively — the assembled context for a sub-agent contains both the skill's methodology terms and the role's domain terms.
3. The `constraint_level` field is supported by the skill system and influences how the system treats the skill's procedure (exact enforcement for low, template enforcement for medium, guidance-only for high).
4. The `roles` field in skill frontmatter declares compatible roles. The binding registry and context assembly pipeline validate that role-skill combinations are sensible.
5. The `orchestrate-review` skill's adaptive composition guidance is advisory — the orchestrator agent makes the dispatch decision based on the guidance, but the system does not programmatically enforce the selection criteria.
6. The `review-plan` skill is a restructure of the existing plan-review SKILL from the current system, adapted to follow the new attention-curve format. Existing plan-review functionality is preserved.
7. The `review-code` skill's examples are representative templates — actual review output will vary based on the code under review and the role-specific vocabulary. The examples demonstrate the required structure and evidence standards, not the domain-specific content.
8. The evaluation criteria in `review-code` are designed to support automated LLM-as-judge evaluation passes. The criteria format (gradable questions with weights) is compatible with a single LLM call producing scores from 0.0–1.0 and a pass-fail grade.