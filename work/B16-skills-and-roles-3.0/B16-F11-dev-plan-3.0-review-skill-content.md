# Implementation Plan: Review Skill Content

| Document | Review Skill Content — Implementation Plan |
|----------|-------------------------------------------|
| Feature  | FEAT-01KN588PGJNM0 (review-skill-content) |
| Status   | Draft |
| Spec     | `work/spec/3.0-review-skill-content.md` |
| Design   | `work/design/skills-system-redesign-v2.md` §3.2, §5.3, §5.4 |

---

## 1. Overview

This plan decomposes the authoring of 3 SKILL.md files into 3 assignable tasks. The skills
form the review layer of the Kanbanzai 3.0 skill catalog:

- **Sub-agent review skill:** `review-code/SKILL.md` — the core review procedure used by ALL specialist reviewers. Specialisation comes from the role's vocabulary, not from the skill. This is the most complex skill in the catalog.
- **Coordinator skill:** `orchestrate-review/SKILL.md` — decomposes a feature into review units, dispatches specialist sub-agents in parallel, collates findings, routes to remediation or approval.
- **Plan-level review skill:** `review-plan/SKILL.md` — checks feature delivery, spec approval, and documentation currency at the plan level.

All 3 skills follow the attention-curve-optimised SKILL.md format: YAML frontmatter
followed by body sections in the order Vocabulary → Anti-Patterns → Checklist (if
applicable) → Procedure → Output Format → Examples → Evaluation Criteria → Questions
This Skill Answers. Every skill body must stay under 500 lines.

The composition model is central: the same `review-code` procedure combined with different
`reviewer-*` roles produces different expertise lenses on the same code. The skill carries
review methodology vocabulary and anti-patterns. The role carries domain-specific vocabulary
and anti-patterns. Context assembly merges both additively.

**Scope boundaries (from spec):**
- This plan covers CONTENT authoring only — SKILL.md files with frontmatter and body sections
- SKILL.md schema definition, parsing logic, and directory layout are out of scope
- Review roles (`reviewer`, `reviewer-*`) are covered by the Review Role Content plan
- Base and authoring roles are covered by a separate plan
- Gate enforcement mechanisms and binding registry structure are out of scope
- Implementation skills and document authoring skills are covered by separate plans

---

## 2. Task Breakdown

### Task 1: Author `review-code/SKILL.md`

**Objective:** Create the core review skill used by all specialist reviewers. This is the
most detailed skill in the catalog — it carries the review methodology vocabulary,
5 named anti-patterns with research citations, a copyable checklist, a 3-step procedure
with validation loops, a structured output format, 3 BAD/GOOD examples with WHY
explanations, gradable evaluation criteria, and retrieval anchors. The skill must be
domain-agnostic — all domain-specific terms belong in the reviewer roles, not here.

**Specification references:** FR-001, FR-002, FR-003, FR-004, FR-005, FR-006, FR-007,
FR-008, FR-009, FR-010, FR-011, NFR-001, NFR-002, NFR-003, NFR-004, NFR-005, NFR-006,
NFR-007

**Input context:**
- Spec §FR-003 — frontmatter values (name, stage, constraint_level, roles list, descriptions, triggers)
- Spec §FR-004 — vocabulary payload (8+ review methodology terms)
- Spec §FR-005 — anti-patterns (5 named patterns with MAST FM-3.1 citation)
- Spec §FR-006 — checklist (7+ items for medium constraint_level)
- Spec §FR-007 — procedure (3 steps: orient, evaluate, validate)
- Spec §FR-008 — output format (structured, machine-parseable)
- Spec §FR-009 — examples (3 BAD/GOOD pairs with WHY)
- Spec §FR-010 — evaluation criteria (6+ gradable questions with weights)
- Spec §FR-011 — retrieval anchors (6+ questions)
- Design §3.2 — full `review-code` SKILL.md example (the canonical example in the design
  document, including frontmatter, all body sections, and design decision rationale)
- Design §5.3 — review skill descriptions and composition model
- Design §5.4 — skill composition during review (4-specialist dispatch example)
- Design §8.1 — novelty test (content must not explain general concepts)
- Design §8.3 — uncertainty protocol (STOP instruction for missing/ambiguous inputs)
- Design DP-4 — anti-patterns with BECAUSE clauses
- Design DP-9 — constraint level alignment (medium = template with bounded flexibility)
- The completed reviewer role files (from the Review Role Content plan) — to verify
  vocabulary separation: skill terms are methodology terms, not domain terms

**Output artifacts:**
- `.kbz/skills/review-code/SKILL.md`
- `.kbz/skills/review-code/references/` directory (if overflow content is needed to stay under 500 lines)

**Dependencies:** None — this task can begin immediately. The design document §3.2 contains
the canonical example with enough detail to author the skill without waiting for the
reviewer role files. Vocabulary separation can be verified after completion if the role
files are not yet available.

**Content guidance:**

*Frontmatter:*
- `name: review-code`
- `description.expert`: Reference multi-dimension code review, classified findings,
  evidence-backed verdicts against acceptance criteria
- `description.natural`: Describe reviewing code changes against a spec and producing a
  structured report
- `triggers`: At least 3 phrases — review code changes, evaluate implementation against
  spec, check code quality
- `roles: [reviewer, reviewer-conformance, reviewer-quality, reviewer-security, reviewer-testing]`
- `stage: reviewing`
- `constraint_level: medium`

*Body sections (in attention-curve order):*

1. **Vocabulary** — At least 8 review methodology terms: finding classification (blocking,
   non-blocking), evidence-backed verdict, acceptance criteria traceability, per-dimension
   outcome (pass, pass_with_notes, concern, fail), review unit decomposition, structured
   review output, remediation recommendation, spec conformance gap. No domain-specific
   terms (OWASP, cyclomatic complexity, boundary value analysis — those belong in roles).

2. **Anti-Patterns** — At least 5 entries, each with Detect/BECAUSE/Resolve:
   - "Rubber-Stamp Review (MAST FM-3.1)" — BECAUSE: LLM sycophancy makes approval the
     path of least resistance; FM-3.1 is the #1 quality failure in multi-agent systems
   - "Severity Inflation" — detect: >40% of findings classified as blocking
   - "Dimension Bleed" — detect: one dimension influencing another's verdict
   - "Prose Commentary" — detect: qualitative prose replacing structured findings
   - "Missing Spec Anchor" — detect: blocking finding without a spec requirement citation;
     BECAUSE: without a spec anchor the finding is an opinion, not a conformance gap

3. **Checklist** — At least 7 copyable checkbox items (`- [ ] item`) covering the full
   review workflow: read spec, read files, confirm review profile and dimensions, evaluate
   each required dimension independently, classify all findings, verify spec anchors on
   blocking findings, produce structured output

4. **Procedure** — 3 numbered steps:
   - Step 1 (Orient): Read spec, read files, note review profile. IF input missing → STOP.
     IF spec ambiguous → STOP. Reference uncertainty protocol (design §8.3).
   - Step 2 (Evaluate): Work through each dimension independently. Record per-dimension
     outcome. One dimension's result MUST NOT affect another's assessment.
   - Step 3 (Validate): Validate → fix → re-validate loop. Check spec anchors on blocking
     findings. Check no dimension bleed. Do not proceed past validation failure.

5. **Output Format** — Structured template with: review unit identification, overall
   verdict, per-dimension outcomes (dimension name, verdict, evidence citations), findings
   classified as blocking or non-blocking, spec requirement references for blocking findings.
   Machine-parseable — a machine must be able to extract all findings without ambiguity.

6. **Examples** — At least 3 examples:
   - BAD: Rubber-stamp with prose and no evidence (demonstrates FM-3.1 failure)
   - GOOD: Evidence-backed structured review with per-dimension verdicts, specific
     evidence citations, and classified findings
   - GOOD (last — recency bias): Legitimate clearance — zero findings but substantive
     per-dimension evidence. Demonstrates that "approved" is not a rubber stamp when backed
     by evidence. Best example placed last.

7. **Evaluation Criteria** — At least 6 gradable questions with weights (required/high/medium):
   - Every dimension has explicit outcome? (required)
   - Every blocking finding cites a spec requirement? (required)
   - Machine can extract all findings without ambiguity? (high)
   - Dimensions evaluated independently, no bleed? (high)
   - Output distinguishes blocking from non-blocking? (required)
   - Every "approved" verdict backed by per-dimension evidence? (high)

8. **Questions This Skill Answers** — At least 6 retrieval anchor questions covering:
   how to review code against spec, what dimensions to evaluate, how to classify findings,
   what output format to use, when to escalate to human checkpoint, what a well-evidenced
   "approved" verdict looks like

*Constraints:*
- Body must stay under 500 lines. If overflow, place extended content in `references/`
  and link from SKILL.md (one level deep — no reference-to-reference chains).
- No domain-specific vocabulary or anti-patterns (those belong in reviewer roles).
- Terminology must be consistent: if the vocabulary says "finding," never use "issue"
  or "problem" elsewhere in the skill.

---

### Task 2: Author `orchestrate-review/SKILL.md`

**Objective:** Create the coordinator skill for the review stage. This skill guides the
orchestrator through decomposing a feature into review units, dispatching specialist
sub-agents (each receiving the `review-code` skill with a different reviewer role),
collating findings from all sub-agents, deduplicating overlapping findings, and routing
to remediation or approval. The skill must implement adaptive composition — dispatching
1–4 reviewers based on change scope, not always the maximum.

**Specification references:** FR-001, FR-002, FR-012, FR-013, FR-014, NFR-001, NFR-002,
NFR-003, NFR-004, NFR-005, NFR-006, NFR-007

**Input context:**
- Spec §FR-012 — frontmatter, purpose, and procedure requirements (decompose, dispatch,
  collate, deduplicate, route)
- Spec §FR-013 — adaptive composition (1–4 reviewers based on change scope, Captain Agent
  research citation, ≤10 files guidance)
- Spec §FR-014 — anti-patterns (3+ orchestration-specific failure modes)
- Design §5.3 — `orchestrate-review` description
- Design §5.4 — skill composition during review (4-specialist dispatch example, adaptive
  composition guidance, Captain Agent research — 15–25% improvement over static teams)
- Design §4.4 — orchestrator role definition (the paired role for this skill)
- Design §3.3 — stage bindings for `reviewing` (orchestrator-workers topology, max 4
  sub-agents)
- Design §8.3 — uncertainty protocol (STOP instruction for missing prerequisites)
- The `review-code` SKILL.md from Task 1 — to understand the output format that
  sub-agents produce (the orchestrator collates this output)

**Output artifacts:**
- `.kbz/skills/orchestrate-review/SKILL.md`
- `.kbz/skills/orchestrate-review/references/` directory (if overflow content is needed)

**Dependencies:** None — this task can begin in parallel with Tasks 1 and 3. The spec and
design contain enough detail about the `review-code` output format to author the collation
procedure without waiting for Task 1. However, if Tasks 2 and 3 start after Task 1, the
completed `review-code` SKILL.md can serve as additional input context.

**Content guidance:**

*Frontmatter:*
- `name: orchestrate-review`
- `description.expert`: Reference review orchestration, adaptive specialist dispatch,
  finding collation, verdict aggregation across review dimensions
- `description.natural`: Describe coordinating a team of code reviewers, collecting their
  findings, and deciding whether the code is ready to ship
- `triggers`: At least 3 phrases — orchestrate code review, coordinate review team,
  run review for feature
- `roles: [orchestrator]`
- `stage: reviewing`
- `constraint_level`: Select based on procedure style — likely `medium` (structured
  dispatch procedure with bounded flexibility on reviewer selection)

*Body sections (in attention-curve order):*

1. **Vocabulary** — Orchestration-specific terms: review unit decomposition, finding
   collation, verdict aggregation, remediation routing, dispatch protocol, adaptive
   composition, review cycle count, specialist selection criteria, deduplication pass

2. **Anti-Patterns** — At least 3 entries, each with Detect/BECAUSE/Resolve:
   - "Result-without-evidence" — detect: accepting sub-agent review output that lacks
     per-dimension evidence or spec citations. Resolve: reject output and re-dispatch
     with explicit evidence requirements
   - "Over-decomposition" — detect: splitting a feature into more review units than the
     code warrants. BECAUSE: each review unit adds dispatch overhead and increases
     context budget consumption
   - At least one additional orchestration failure mode (e.g., "Static team dispatch" —
     always dispatching all 4 specialists regardless of change scope)

3. **Procedure** — Must cover:
   - Decompose feature into review units (group related files)
   - Select specialist reviewers adaptively based on files changed:
     - Maximum 4 sub-agents (binding registry ceiling), but this is NOT a target
     - Small features (≤10 files) may need only 1–2 reviewers
     - If no security-relevant code changed, do not dispatch security reviewer
     - Reference Captain Agent research (15–25% improvement over static teams)
   - Dispatch sub-agents: each receives `review-code` skill + a different reviewer role
   - Collate findings from all sub-agents
   - Deduplicate overlapping findings (same location, same issue, different reviewers)
   - Route: if blocking findings exist → remediation; if none → approval
   - Include STOP instruction for missing prerequisites (uncertainty protocol)

4. **Output Format** — Aggregate review report structure: per-reviewer summaries, collated
   finding list (deduplicated), aggregate verdict, remediation plan (if blocking findings)

5. **Examples** — At least 1 BAD/GOOD pair showing orchestration decisions

6. **Evaluation Criteria** — Gradable questions covering: adaptive selection rationale,
   finding deduplication, evidence validation, routing decision correctness

7. **Questions This Skill Answers** — Retrieval anchors covering: how to coordinate
   a review, when to dispatch fewer reviewers, how to handle conflicting findings

*Constraints:*
- Body must stay under 500 lines.
- No review methodology vocabulary (that belongs in `review-code`).
- No domain-specific vocabulary (that belongs in reviewer roles).
- Terminology must be consistent with the vocabulary section.

---

### Task 3: Author `review-plan/SKILL.md`

**Objective:** Create the plan-level review skill that checks whether all features within
a plan have shipped, all specifications are approved, and documentation is current. This
is a conformance-focused skill — it verifies completeness and approval status, not code
quality or security. The skill is a restructure of the existing plan-review functionality,
adapted to the new attention-curve SKILL.md format.

**Specification references:** FR-001, FR-002, FR-015, NFR-001, NFR-002, NFR-003, NFR-004,
NFR-005, NFR-006, NFR-007

**Input context:**
- Spec §FR-015 — frontmatter, purpose, and requirements (feature completion, spec approval,
  documentation currency, conformance focus)
- Design §5.3 — `review-plan` description
- Design §3.3 — stage bindings for `plan-reviewing` (single-agent orchestration)
- Design §8.3 — uncertainty protocol (STOP instruction for incomplete/contradictory plan state)
- Existing plan-review SKILL from the current system (`.agents/skills/kanbanzai-plan-review/SKILL.md`)
  — for reference on existing functionality to preserve, adapted to new format
- The `reviewer-conformance` role (from the Review Role Content plan) — the paired role
  for this skill

**Output artifacts:**
- `.kbz/skills/review-plan/SKILL.md`
- `.kbz/skills/review-plan/references/` directory (if overflow content is needed)

**Dependencies:** None — this task can begin in parallel with Tasks 1 and 2. The spec and
design contain sufficient detail.

**Content guidance:**

*Frontmatter:*
- `name: review-plan`
- `description.expert`: Reference plan-level conformance review, feature delivery
  verification, specification approval audit, documentation currency check
- `description.natural`: Describe checking whether all the work in a plan is done,
  specs are approved, and docs are up to date
- `triggers`: At least 3 phrases — review plan completion, check plan delivery status,
  verify plan readiness
- `roles: [reviewer-conformance]`
- `stage: plan-reviewing`
- `constraint_level`: Select based on procedure style — likely `low` (exact conformance
  checks with minimal judgement required)

*Body sections (in attention-curve order):*

1. **Vocabulary** — Plan-review-specific terms: feature delivery status, specification
   approval status, documentation currency, plan completeness, conformance gap,
   delivery verification

2. **Anti-Patterns** — At least 2 entries covering plan-review failure modes:
   - Plan-level equivalent of rubber-stamp (marking plan complete without checking each
     feature)
   - Scope confusion (reviewing code quality instead of plan conformance)

3. **Procedure** — Must cover:
   - Check that all features within the plan have reached a terminal status
   - Check that all specifications are in approved status
   - Check that documentation is current (not stale, not missing)
   - Include STOP instruction for incomplete or contradictory plan state
   - The procedure is conformance-focused: verify completeness and approval status, do
     not evaluate code quality or security

4. **Output Format** — Plan review report structure: per-feature status, spec approval
   status, documentation currency assessment, overall plan verdict, list of gaps (if any)

5. **Examples** — At least 1 BAD/GOOD pair showing plan review output

6. **Evaluation Criteria** — Gradable questions covering: all features checked, all specs
   checked, documentation checked, conformance focus maintained (no scope creep into
   code quality)

7. **Questions This Skill Answers** — Retrieval anchors covering: how to review a plan
   for completion, what to check before closing a plan, how to verify all specs are approved

*Constraints:*
- Body must stay under 500 lines.
- Conformance-focused only — no code quality, security, or testing content.
- Terminology must be consistent with the vocabulary section.

---

## 3. Dependency Graph

```
Task 1: review-code/SKILL.md ──────────────┐
                                            │ (no dependencies between tasks)
Task 2: orchestrate-review/SKILL.md ───────┤
                                            │
Task 3: review-plan/SKILL.md ──────────────┘
```

**Execution order:** All 3 tasks can execute in parallel. There are no dependencies
between them — each task has sufficient context from the specification and design
document to proceed independently.

**External dependencies:**
- The reviewer role files (from the Review Role Content plan) should be completed before
  final validation of vocabulary separation in Task 1. However, the design document §3.2
  contains the canonical `review-code` example with enough detail to author the skill
  without waiting. Vocabulary separation can be verified as a post-authoring check.
- The `base.yaml` and `orchestrator.yaml` roles (from the Base and Authoring Role Content
  plan) inform Tool 2's orchestration context but are not blocking — the design document
  provides sufficient orchestrator vocabulary detail.

**Maximum parallelism:** 3 concurrent tasks.

**Recommended sequencing (optional):** If parallelism is constrained, prioritise Task 1
(`review-code`) first — it is the most complex and the most referenced by the other two
skills. Task 2 (`orchestrate-review`) and Task 3 (`review-plan`) can follow in either
order.

---

## 4. Interface Contracts

These tasks produce content files, not code. The shared interfaces are conventions,
terminology, and format requirements that must be consistent across all 3 SKILL.md files.

### 4.1 SKILL.md Frontmatter Schema Contract

Every skill file must contain YAML frontmatter with these fields:

```yaml
---
name: <skill-name>
description:
  expert: "<activates deep domain knowledge on direct invocation>"
  natural: "<triggers on casual phrasing, understandable without domain expertise>"
triggers:
  - <natural-language phrase 1>
  - <natural-language phrase 2>
  - <natural-language phrase 3>
roles: [<role-id-1>, <role-id-2>]
stage: <workflow-stage>
constraint_level: <low | medium | high>
---
```

### 4.2 Attention-Curve Section Ordering Contract

Every skill body must follow this section order (no exceptions):

1. Vocabulary (highest-attention position)
2. Anti-Patterns
3. Checklist (if applicable — required for medium/low constraint_level)
4. Procedure
5. Output Format
6. Examples
7. Evaluation Criteria
8. Questions This Skill Answers (high-attention end-of-context position)

No skill may place Procedure before Vocabulary or Anti-Patterns. No skill may place
Evaluation Criteria or Questions before Examples.

### 4.3 Anti-Pattern Structure Contract

Every anti-pattern across all skill files must have these sub-entries:
- **Detect** — observable signal (present tense)
- **BECAUSE** — explanation of why this is harmful (enables generalisation)
- **Resolve** — concrete corrective action

The BECAUSE field must never be a restatement of Detect. The Resolve field must never
be vague.

### 4.4 Vocabulary Separation Contract (Skill vs Role)

This is the critical composition model contract. Vocabulary and anti-patterns are divided
between skills and roles as follows:

| Content type | Belongs in skill | Belongs in role |
|-------------|-----------------|-----------------|
| Review methodology terms | ✓ (review-code) | |
| Orchestration terms | ✓ (orchestrate-review) | |
| Plan conformance terms | ✓ (review-plan) | |
| Security domain terms (OWASP, STRIDE, CVSS) | | ✓ (reviewer-security) |
| Quality domain terms (cyclomatic complexity) | | ✓ (reviewer-quality) |
| Testing domain terms (boundary value analysis) | | ✓ (reviewer-testing) |
| Conformance domain terms (spec mapping) | | ✓ (reviewer-conformance) |
| Foundational review terms (finding classification) | | ✓ (reviewer base) |

The `review-code` skill carries ONLY review methodology terms. If a term is specific to
security, quality, testing, or conformance, it belongs in the corresponding reviewer role.
The assembled context for any sub-agent is the union of skill vocabulary + role vocabulary.

### 4.5 Output Format Interoperability Contract

The `review-code` output format and the `orchestrate-review` collation input must be
compatible:

- `review-code` produces a structured review per review unit with per-dimension outcomes,
  classified findings, and spec anchors
- `orchestrate-review` consumes these structured reviews, deduplicates findings across
  reviewers, and produces an aggregate verdict

The output format defined in `review-code` is the contract that `orchestrate-review`
depends on for collation. The field names and structure must be consistent.

### 4.6 Terminology Consistency Contract

Each skill's vocabulary section defines the canonical terms for that skill's domain.
Within the skill's own content — anti-patterns, procedure, examples, evaluation criteria,
questions — those canonical terms must be used exclusively. No synonyms.

Examples of enforced consistency:
- If the vocabulary says "finding," never use "issue" or "problem"
- If the vocabulary says "per-dimension outcome," never use "per-dimension result" or "score"
- If the vocabulary says "review unit," never use "review chunk" or "review segment"

### 4.7 500-Line Budget Contract

Every SKILL.md body must stay under 500 lines. If a skill needs overflow content:
- Place it in the skill's `references/` subdirectory
- Link directly from SKILL.md (e.g., `See [finding classification details](references/finding-classification.md)`)
- No reference-to-reference chains (one level deep only)

---

## 5. Traceability Matrix

| Requirement | Task(s) |
|-------------|---------|
| FR-001 (Attention-curve section ordering) | 1, 2, 3 |
| FR-002 (SKILL.md frontmatter requirements) | 1, 2, 3 |
| FR-003 (review-code — frontmatter values) | 1 |
| FR-004 (review-code — vocabulary payload) | 1 |
| FR-005 (review-code — anti-patterns) | 1 |
| FR-006 (review-code — checklist) | 1 |
| FR-007 (review-code — procedure) | 1 |
| FR-008 (review-code — output format) | 1 |
| FR-009 (review-code — examples) | 1 |
| FR-010 (review-code — evaluation criteria) | 1 |
| FR-011 (review-code — retrieval anchors) | 1 |
| FR-012 (orchestrate-review — frontmatter and purpose) | 2 |
| FR-013 (orchestrate-review — adaptive composition) | 2 |
| FR-014 (orchestrate-review — anti-patterns) | 2 |
| FR-015 (review-plan — frontmatter and purpose) | 3 |
| NFR-001 (Novelty test compliance) | 1, 2, 3 |
| NFR-002 (SKILL.md body under 500 lines) | 1, 2, 3 |
| NFR-003 (Composition model integrity) | 1, 2 |
| NFR-004 (Tone and explanatory style) | 1, 2, 3 |
| NFR-005 (Uncertainty protocol inclusion) | 1, 2, 3 |
| NFR-006 (Constraint level alignment) | 1, 2, 3 |
| NFR-007 (Terminology consistency) | 1, 2, 3 |