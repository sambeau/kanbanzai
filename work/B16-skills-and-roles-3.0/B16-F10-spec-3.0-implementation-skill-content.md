# Specification: Implementation Skill Content

| Field   | Value                                                              |
|---------|--------------------------------------------------------------------|
| Status  | Draft                                                              |
| Feature | FEAT-01KN5-88PG7HA3 (implementation-skill-content)                |
| Design  | `work/design/skills-system-redesign-v2.md` §5.2, §3.2, §8, §9    |

---

## 1. Overview

This specification defines the content requirements for three implementation SKILL.md files — `implement-task`, `orchestrate-development`, and `decompose-feature` — that cover the developing and dev-planning stages of the feature lifecycle. Each skill follows the attention-curve SKILL.md format defined in §3.2 of the design, is bound to a specific feature lifecycle stage and paired role, and carries stage-appropriate vocabulary, anti-patterns, examples, and evaluation criteria. `implement-task` is deliberately lean, deferring language expertise to implementer roles and spec requirements to the task itself. `orchestrate-development` carries explicit context compaction techniques for multi-task coordination. `decompose-feature` invests disproportionately in decomposition validation as the single strongest predictor of downstream workflow success.

---

## 2. Scope

### 2.1 In Scope

- Content requirements for three SKILL.md files: `implement-task`, `orchestrate-development`, `decompose-feature`.
- Stage alignment, paired role binding, and frontmatter field values for each skill.
- Required vocabulary payload, anti-pattern set, procedure, examples, and evaluation criteria for each skill.
- Three specific context compaction techniques for `orchestrate-development`.
- Decomposition validation criteria for `decompose-feature`.
- Replacement of implicit implementation and orchestration procedures currently embedded in AGENTS.md and ad-hoc agent behaviour.

### 2.2 Explicitly Excluded

- The SKILL.md file format schema itself (structural concern defined in the skill system spec and design §3.2).
- Context assembly — how skills are combined with roles and injected into agent prompts (covered by the context assembly pipeline spec).
- The binding registry schema or `stage-bindings.yaml` structure (covered by the binding registry spec).
- Role content — the vocabulary and anti-patterns carried by paired roles such as `implementer-go`, `orchestrator`, or `architect` (covered by the role content spec).
- Tool-level validation checks for decomposition quality within the `decompose` tool itself (covered by the workflow spec §11).
- Language-specific coding conventions, test patterns, or style guides (these belong in implementer roles, not in skills).
- Implementation of the `decompose` tool or changes to its MCP interface.

---

## 3. Functional Requirements

### FR-001: Three Implementation Skills

The system MUST provide exactly three implementation skills (`implement-task`, `orchestrate-development`, `decompose-feature`), each stored at `.kbz/skills/{skill-name}/SKILL.md`.

**Acceptance criteria:**
- Each of the three SKILL.md files exists at its specified path under `.kbz/skills/`
- No other skill claims the `developing` stage for individual task execution, multi-task orchestration, or the `dev-planning` stage for feature decomposition
- Each skill is discoverable via its frontmatter `triggers` field

---

### FR-002: Stage Alignment

Each implementation skill MUST declare exactly one `stage` in its frontmatter that matches the feature lifecycle stage where that skill applies. The required bindings are:

| Skill | Stage |
|-------|-------|
| `implement-task` | `developing` |
| `orchestrate-development` | `developing` |
| `decompose-feature` | `dev-planning` |

**Acceptance criteria:**
- Each skill's frontmatter `stage` field matches the value in the table above
- The stage values correspond to valid feature lifecycle states in the existing state machine
- `implement-task` and `orchestrate-development` share the `developing` stage (one for workers, one for the coordinator)

---

### FR-003: Paired Role Declaration

Each implementation skill MUST declare its compatible roles in the frontmatter `roles` field. The required pairings are:

| Skill | Paired Role(s) |
|-------|----------------|
| `implement-task` | `implementer`, `implementer-go` (and future `implementer-*` variants) |
| `orchestrate-development` | `orchestrator` |
| `decompose-feature` | `architect` |

**Acceptance criteria:**
- Each skill's `roles` field contains the role(s) listed in the table above
- The `implement-task` skill's `roles` field includes at minimum `implementer-go` and the base `implementer` role
- Every role listed in a skill's `roles` field corresponds to a role file that exists (or will exist) at `.kbz/roles/{role-id}.yaml`

---

### FR-004: Attention-Curve Section Ordering

Each implementation skill's SKILL.md MUST follow the attention-curve section ordering defined in the design (§3.2). The sections MUST appear in this order:

1. Frontmatter (YAML)
2. `## Vocabulary`
3. `## Anti-Patterns`
4. `## Checklist` (optional — required for medium/low `constraint_level` skills)
5. `## Procedure`
6. `## Output Format`
7. `## Examples`
8. `## Evaluation Criteria`
9. `## Questions This Skill Answers`

**Acceptance criteria:**
- Every implementation SKILL.md contains sections 1–3 and 5–9 in the specified order
- No section appears out of the defined order
- Optional sections, if present, appear at their defined position

---

### FR-005: implement-task — Lean Procedure

The `implement-task` skill MUST be deliberately lean. Its procedure MUST define exactly these phases in order: (1) read the spec/acceptance criteria for the assigned task, (2) implement the required changes, (3) write or update tests, (4) verify that tests pass and acceptance criteria are met. The skill MUST NOT carry language-specific coding guidance, style rules, or framework conventions — these belong in the paired implementer role.

**Acceptance criteria:**
- The `implement-task` procedure section contains a read → implement → test → verify sequence
- The procedure does not include language-specific instructions (no Go, Python, TypeScript, etc. references)
- The procedure references the task's spec requirements and acceptance criteria as the authority for what to implement
- The skill's total content (excluding frontmatter) is shorter than either `orchestrate-development` or `decompose-feature`

---

### FR-006: implement-task — Anti-Patterns

The `implement-task` skill MUST carry 5–10 anti-patterns specific to individual task execution. The anti-patterns MUST include at minimum: (1) scope creep — implementing beyond what the task specifies, (2) untested code paths — writing code without corresponding test coverage, (3) spec deviation — diverging from specified behaviour without flagging the discrepancy. Each anti-pattern MUST include a name, Detect signal, BECAUSE clause, and Resolve step.

**Acceptance criteria:**
- The `implement-task` `## Anti-Patterns` section contains between 5 and 10 anti-patterns
- The three named anti-patterns (scope creep, untested code paths, spec deviation) are each present with all four fields
- Every anti-pattern's BECAUSE clause explains the downstream consequence, not just restates the detection signal
- Anti-patterns focus on task execution concerns, not orchestration or decomposition concerns

---

### FR-007: orchestrate-development — Coordination Procedure

The `orchestrate-development` skill MUST define a procedure that covers all of the following phases in order: (1) read the dev-plan to understand task breakdown and dependencies, (2) identify tasks that can be dispatched in parallel (no unmet dependencies), (3) dispatch implementer sub-agents for independent tasks, (4) monitor progress and handle task failures, (5) perform context compaction between sequential sub-agent completions. The procedure MUST respect dependency ordering — a task with unmet dependencies MUST NOT be dispatched.

**Acceptance criteria:**
- The procedure section contains all five phases in the specified order
- The procedure explicitly states that tasks with unmet dependencies must not be dispatched
- The procedure references the dev-plan as the authority for task breakdown and dependency ordering
- The procedure distinguishes between parallel-dispatchable tasks (independent) and sequential tasks (dependent)

---

### FR-008: orchestrate-development — Three Context Compaction Techniques

The `orchestrate-development` skill MUST include exactly three context compaction techniques in its procedure or as a dedicated sub-section. The three techniques are:

1. **Post-completion summarisation:** After each sub-agent completes, reduce the outcome to 2–3 sentences and a task ID. The full sub-agent output MUST NOT be retained in the orchestrator's conversation.
2. **Document-based offloading:** When context utilisation exceeds 60% during a multi-task orchestration, write a progress summary to a registered document and start a fresh orchestration session that reads the summary.
3. **Single-feature scoping:** Structure multi-feature plans as a sequence of single-feature contexts. Do not orchestrate all features in one session. Complete one feature's tasks, write the summary, then begin the next.

**Acceptance criteria:**
- All three compaction techniques are present and described in the skill
- The post-completion summarisation technique specifies the 2–3 sentence constraint and task ID requirement
- The document-based offloading technique specifies the 60% context utilisation threshold
- The single-feature scoping technique explicitly prohibits multi-feature orchestration in a single session
- Each technique is described with enough specificity to be followed as a procedure (not just named)

---

### FR-009: orchestrate-development — Vocabulary

The `orchestrate-development` skill MUST carry a vocabulary section containing 15–30 terms relevant to multi-agent coordination. The vocabulary MUST include terms for: parallel task dispatch, dependency-order sequencing, progress monitoring, sub-agent output handling, context compaction, and failure recovery.

**Acceptance criteria:**
- The vocabulary section contains between 15 and 30 terms (inclusive)
- At least two terms relate to parallel task dispatch (e.g., "independent task set", "parallel dispatch batch")
- At least two terms relate to dependency ordering (e.g., "dependency-order sequencing", "topological task order")
- At least two terms relate to context compaction (e.g., "completion summary", "context offloading")
- Terms pass the 15-year practitioner test: a senior engineering manager or tech lead would use them with a peer

---

### FR-010: decompose-feature — Decomposition Validation Investment

The `decompose-feature` skill MUST invest disproportionately in decomposition validation. The procedure MUST include a validation phase that checks all of the following conditions against the output of the `decompose` tool:

1. Every task has a clear, non-empty description
2. Dependencies between tasks are explicitly declared
3. Each task is sized for single-agent completion (not too large to require further decomposition)
4. No circular dependencies exist
5. Integration/test tasks are present (gap detection for missing test tasks)

The validation phase MUST be structured as a validate → fix → re-validate loop, not a single pass.

**Acceptance criteria:**
- The procedure contains a dedicated validation phase with all five checks listed
- The validation phase is structured as an explicit loop (validate → if issues found → fix → re-validate)
- The skill text characterises decomposition validation as the most important part of the procedure (disproportionate emphasis)
- Each validation check is phrased as a testable condition, not a vague guideline

---

### FR-011: decompose-feature — Anti-Patterns

The `decompose-feature` skill MUST carry 5–10 anti-patterns specific to feature decomposition. The anti-patterns MUST include at minimum: (1) over-decomposition — creating tasks so granular they add coordination overhead without value, (2) circular dependencies — tasks that depend on each other forming a cycle, (3) missing integration tasks — decomposing into implementation tasks without tasks for integration, testing, or verification. Each anti-pattern MUST include a name, Detect signal, BECAUSE clause, and Resolve step.

**Acceptance criteria:**
- The `decompose-feature` `## Anti-Patterns` section contains between 5 and 10 anti-patterns
- The three named anti-patterns (over-decomposition, circular dependencies, missing integration tasks) are each present with all four fields
- Every anti-pattern's BECAUSE clause explains the downstream consequence
- Anti-patterns focus on decomposition concerns, not implementation or orchestration concerns

---

### FR-012: decompose-feature — Vocabulary for Vertical Slicing and Sizing

The `decompose-feature` skill MUST carry a vocabulary section containing 15–30 terms relevant to feature decomposition. The vocabulary MUST include terms for: dependency analysis, vertical slicing, task sizing, and gap detection.

**Acceptance criteria:**
- The vocabulary section contains between 15 and 30 terms (inclusive)
- At least two terms relate to dependency analysis (e.g., "topological sort", "dependency graph")
- At least two terms relate to vertical slicing (e.g., "vertical slice", "end-to-end increment")
- At least two terms relate to sizing (e.g., "single-agent scope", "effort estimate")
- Terms pass the 15-year practitioner test

---

### FR-013: Frontmatter Completeness

Each implementation skill's SKILL.md MUST include a YAML frontmatter block with all required fields: `name`, `description` (with `expert` and `natural` sub-fields), `triggers` (list of at least 2 trigger phrases), `roles`, `stage`, and `constraint_level`.

**Acceptance criteria:**
- Each of the three skills has a frontmatter block containing all six required fields
- The `description.expert` field is a technical description suitable for direct invocation
- The `description.natural` field is a plain-language description suitable for casual trigger matching
- The `triggers` field contains at least 2 trigger phrases per skill
- The `constraint_level` field is one of `low`, `medium`, or `high`
- `implement-task` has `constraint_level: medium` (procedural with defined steps but implementation freedom)
- `orchestrate-development` has `constraint_level: medium` (coordination procedure with adaptation points)
- `decompose-feature` has `constraint_level: low` (high analytical freedom within validation constraints)

---

### FR-014: BAD vs GOOD Examples

Each implementation skill MUST include at least two examples in its `## Examples` section: at minimum one BAD example and one GOOD example. Each example MUST include a `WHY BAD` or `WHY GOOD` explanation. The best (GOOD) example MUST appear last in the section to exploit recency bias. Examples MUST be representative of the specific skill's domain.

**Acceptance criteria:**
- Each skill has at least one BAD and one GOOD example
- Every BAD example has a `WHY BAD` explanation
- Every GOOD example has a `WHY GOOD` explanation
- The final example in each skill's Examples section is a GOOD example
- `implement-task` examples show task execution scenarios (not orchestration or decomposition)
- `orchestrate-development` examples show multi-task coordination scenarios (not individual implementation)
- `decompose-feature` examples show feature breakdown scenarios (not implementation or orchestration)

---

### FR-015: Evaluation Criteria as Gradable Questions

Each implementation skill MUST include an `## Evaluation Criteria` section containing 4–8 gradable questions about the skill's output quality. Each question MUST have a weight designation (`required`, `high`, or `medium`). The criteria MUST be evaluable by an LLM-as-judge pass producing scores from 0.0–1.0.

**Acceptance criteria:**
- Each skill has between 4 and 8 evaluation criteria (inclusive)
- Each criterion is phrased as a yes/no or gradable question
- Each criterion has a weight of `required`, `high`, or `medium`
- At least one criterion per skill has weight `required`
- `implement-task` criteria focus on spec conformance and test coverage
- `orchestrate-development` criteria focus on dependency respect, compaction discipline, and progress tracking
- `decompose-feature` criteria focus on decomposition quality, dependency correctness, and gap coverage

---

## 4. Non-Functional Requirements

**NFR-001:** Each SKILL.md file MUST be under 500 lines. Detailed anti-pattern documentation, extended examples, and evaluation rubrics that would push a skill over this limit MUST be placed in `references/` files within the skill's directory.

**Acceptance criteria:**
- `wc -l` on each SKILL.md returns a value ≤ 500
- Any `references/` files are linked directly from the SKILL.md (one level deep — no reference-to-reference chains)

---

**NFR-002:** Vocabulary terms across an implementation skill and its paired role MUST compose additively without duplication. Where the same term appears in both the role vocabulary and the skill vocabulary, it MUST appear in only one (preferring the role for language/domain terms and the skill for procedure/methodology terms).

**Acceptance criteria:**
- The intersection of vocabulary terms between each skill and its paired role contains zero entries
- Combined vocabulary (role + skill) stays within the 15–30 term guideline per unit

---

**NFR-003:** All reference files within a skill directory MUST be linked directly from the SKILL.md. No reference file may link to another reference file (one-level-deep constraint).

**Acceptance criteria:**
- Every file in a skill's `references/` directory is referenced by a link or citation in the parent SKILL.md
- No file in `references/` contains links to other files in `references/`

---

**NFR-004:** The `implement-task` skill MUST remain lean enough that when composed with an `implementer-*` role, the combined context does not exceed 40% of a standard context window. The skill itself SHOULD be the shortest of the three implementation skills.

**Acceptance criteria:**
- The `implement-task` SKILL.md is shorter in line count than both `orchestrate-development` and `decompose-feature`
- The skill does not duplicate content that belongs in the implementer role (language conventions, style rules, tool preferences)

---

## 5. Acceptance Criteria

The requirements above include inline acceptance criteria. The following are system-level acceptance criteria for the feature as a whole:

1. **Completeness:** All three SKILL.md files exist at their specified paths, each containing all required sections in attention-curve order.
2. **Stage coverage:** The `developing` stage is covered by both `implement-task` (worker) and `orchestrate-development` (coordinator), reflecting the orchestrator-workers topology. The `dev-planning` stage is covered by `decompose-feature`.
3. **Compaction techniques:** The `orchestrate-development` skill contains all three specified compaction techniques with actionable thresholds and procedures.
4. **Decomposition validation:** The `decompose-feature` skill's validation phase covers all five validation checks in a validate → fix → re-validate loop.
5. **Lean implementation skill:** The `implement-task` skill defers language expertise to roles and spec requirements to tasks, containing no language-specific or framework-specific guidance.
6. **Quality gate:** Each skill passes the skill quality gate checklist defined in the design (§9.2): attention-curve ordering, vocabulary count 15–30, anti-patterns 5–10 with BECAUSE clauses, examples with WHY explanations, evaluation criteria as gradable questions.

---

## 6. Dependencies and Assumptions

### Dependencies

- **Skill system schema:** The SKILL.md file format (frontmatter fields, section structure) must be defined before authoring content. This is specified by the skill system feature.
- **Role content:** The paired roles (`implementer-go`, `orchestrator`, `architect`) must have their vocabulary and anti-patterns defined so that skill content complements rather than duplicates role content.
- **Binding registry:** The `stage-bindings.yaml` file must define entries for `developing` (with `orchestration_pattern: orchestrator-workers`) and `dev-planning` stages, including the role and skill pairings.
- **Feature lifecycle states:** The stages referenced by each skill (`developing`, `dev-planning`) must be valid states in the feature lifecycle state machine (assumed unchanged from current system).
- **`decompose` tool:** The `decompose-feature` skill guides the use of the existing `decompose` MCP tool. That tool's interface and output format must be stable.
- **Sub-agent dispatch mechanism:** The `orchestrate-development` skill assumes that orchestrators can dispatch sub-agents (via `handoff` or equivalent). The dispatch mechanism must support parallel execution.

### Assumptions

- The attention-curve SKILL.md format (§3.2 of the design) is stable and will not change materially before these skills are authored.
- The `developing` stage uses the `orchestrator-workers` topology as declared in the binding registry, meaning both a coordinator skill and a worker skill are needed for the same stage.
- Context utilisation percentage (referenced by the 60% compaction threshold) can be estimated by the orchestrating agent based on conversation length and known context window size; precise programmatic measurement is not required.
- The `decompose` tool performs structural validation (description present, dependencies declared); the `decompose-feature` skill provides the analytical guidance that complements tool-level checks.
- Future `implementer-*` role variants (e.g., `implementer-ts`, `implementer-python`) will be compatible with the `implement-task` skill without requiring skill modifications.