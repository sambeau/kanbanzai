# Specification: Context Assembly Pipeline

| Field   | Value                                                                 |
|---------|-----------------------------------------------------------------------|
| Status  | Draft                                                                 |
| Feature | FEAT-01KN5-88PE43M6 (context-assembly-pipeline)                      |
| Design  | `work/design/skills-system-redesign-v2.md` §3.4, §6.1, §6.2, §11    |

---

## 1. Overview

This specification defines the 10-step context assembly pipeline that integrates roles, skills, stage bindings, knowledge entries, and token budget management into a single attention-curve-ordered prompt. The pipeline extends the existing `handoff` tool to produce assembled context that maximises agent effectiveness by placing content at empirically optimal attention positions, managing token budgets across progressive disclosure layers, and providing tool subset guidance. It is the single canonical path through which agent context is constructed for any task.

---

## 2. Scope

### 2.1 In Scope

- The 10-step assembly pipeline (steps 0–10) executed when `handoff` is called.
- Attention-curve output ordering of assembled context sections.
- Within-section ordering exploiting recency bias (most critical item last).
- Token budget estimation with progressive disclosure layers (Layers 1–4).
- Budget warning at 40% of context window and refusal at 60%.
- Vocabulary merging: role vocabulary combined with skill vocabulary.
- Anti-pattern merging: role anti-patterns combined with skill anti-patterns.
- Tool subset soft filtering: generating guidance text listing which tools the role should use.
- Extension of the existing `handoff` tool to use this pipeline.
- Orchestration metadata extraction (pattern, effort budget, prerequisites, max_review_cycles).
- Stage-specific inclusion/exclusion strategy application.

### 2.2 Out of Scope

- Hard tool filtering (dynamically scoping the MCP tool list per session). This is the design target but is deferred beyond this specification (per DD-8).
- The content or schema of roles, skills, or stage bindings themselves (specified in their own features).
- Knowledge auto-surfacing matching criteria and cap logic (specified in the Knowledge Auto-Surfacing feature; this spec treats step 7 as an integration point).
- Freshness tracking metadata on roles and skills (specified in the Freshness Tracking feature).
- The stage binding registry's own schema and validation.
- Changes to the feature lifecycle state machine or transition rules.
- The evaluation suite for measuring assembly quality.

---

## 3. Functional Requirements

### FR-001: Pipeline Entry Point

The `handoff` tool MUST execute the 10-step context assembly pipeline when called with a `task_id`. The tool MUST accept an optional `role` parameter for context shaping. The assembled output MUST be a structured Markdown prompt suitable for direct use as a sub-agent message.

**Acceptance criteria:**
- Calling `handoff(task_id="TASK-...")` returns a structured Markdown prompt containing all assembled sections.
- The output is non-empty and contains at minimum the identity section and the task-specific content.
- The existing `handoff` tool contract (accepting `task_id` and optional `role` and `instructions`) is preserved.

---

### FR-002: Step 0 — Lifecycle State Validation

The pipeline MUST validate that the task's parent feature is in a lifecycle state that permits work on the task. If the feature is in an invalid state, the pipeline MUST reject assembly with an error message identifying the current feature state and the required state(s).

**Acceptance criteria:**
- Calling `handoff` on a task whose parent feature is in `draft` status returns an error containing the current status and the acceptable statuses.
- The pipeline does not proceed to step 1 when validation fails.
- Calling `handoff` on a task whose parent feature is in a valid working state (e.g., `developing`) succeeds past step 0.

---

### FR-003: Step 1 — Task-to-Stage Resolution

The pipeline MUST resolve the task to its parent feature and determine the feature's current lifecycle stage. The resolved stage MUST be used as the lookup key for the stage binding in step 2.

**Acceptance criteria:**
- Given a task whose parent feature is in `developing` status, the pipeline resolves the stage as `developing`.
- If the task has no parent feature, the pipeline returns an error identifying the orphaned task.

---

### FR-004: Step 2 — Stage Binding Lookup

The pipeline MUST look up the stage binding for the resolved lifecycle stage from the binding registry. If no binding exists for the stage, the pipeline MUST return an error identifying the stage and stating that no binding is configured.

**Acceptance criteria:**
- For a feature in `developing` stage, the pipeline retrieves the `developing` binding entry including its roles, skills, orchestration pattern, and effort budget.
- For a stage with no configured binding, the pipeline returns an error containing the stage name and "no binding configured" (or equivalent).

---

### FR-005: Step 3 — Stage-Specific Inclusion/Exclusion

The pipeline MUST apply the stage binding's inclusion/exclusion strategy to vary what context is assembled. Content categories not included for the current stage MUST be omitted from the assembled output.

**Acceptance criteria:**
- A stage binding that excludes a content category (e.g., full spec sections) results in assembled context that does not contain that category.
- A stage binding that includes a content category results in that category appearing in the assembled context at its designated position.

---

### FR-006: Step 4 — Orchestration Metadata Extraction

The pipeline MUST extract orchestration metadata from the stage binding: the orchestration pattern (e.g., `single-agent`, `orchestrator-workers`), the effort budget, stage prerequisites, and `max_review_cycles` (if present). This metadata MUST appear at position 3 in the assembled output (high-attention zone).

**Acceptance criteria:**
- Assembled context for a `developing` stage binding contains the orchestration pattern and effort budget text within the first four sections of output.
- Assembled context for a `reviewing` stage binding contains `max_review_cycles` in the orchestration metadata section.
- The effort budget string from the binding appears verbatim in the assembled context.

---

### FR-007: Step 5 — Role Resolution with Inheritance

The pipeline MUST resolve the role specified by the stage binding, including inheritance. If the role declares `inherits: <parent>`, the pipeline MUST load the parent role and merge fields. Vocabulary MUST be merged by concatenating the parent's vocabulary with the child's vocabulary (parent first). Anti-patterns MUST be merged by concatenating the parent's anti-patterns with the child's anti-patterns (parent first). The child role's identity MUST override the parent's identity.

**Acceptance criteria:**
- A role `reviewer-security` that inherits from `reviewer` produces merged vocabulary containing the `reviewer` terms followed by the `reviewer-security` terms.
- A role `reviewer-security` that inherits from `reviewer` produces merged anti-patterns containing the `reviewer` anti-patterns followed by the `reviewer-security` anti-patterns.
- The identity string in the assembled context is the child role's identity, not the parent's.
- If the specified role file does not exist, the pipeline returns an error identifying the missing role.

---

### FR-008: Step 6 — Skill Loading

The pipeline MUST load the skill specified by the stage binding. The loaded skill provides: vocabulary, anti-patterns, procedure (numbered steps), output format, examples, evaluation criteria, and retrieval anchors. If the specified skill file does not exist, the pipeline MUST return an error identifying the missing skill.

**Acceptance criteria:**
- A skill loaded from `.kbz/skills/<skill-name>/SKILL.md` contributes its procedure section to the assembled context.
- If the skill file is missing, the pipeline returns an error containing the skill name and "not found" (or equivalent).

---

### FR-009: Vocabulary Merging

The pipeline MUST combine the role's vocabulary with the skill's vocabulary into a single vocabulary section. The role's vocabulary MUST appear first, followed by the skill's vocabulary appended after it. Duplicate terms across role and skill MUST be preserved (no deduplication).

**Acceptance criteria:**
- Assembled context contains a single vocabulary section.
- The role's vocabulary terms precede the skill's vocabulary terms in that section.
- A term appearing in both role and skill vocabulary appears twice in the assembled output.

---

### FR-010: Anti-Pattern Merging

The pipeline MUST combine the role's anti-patterns with the skill's anti-patterns into a single anti-pattern section. The role's anti-patterns MUST appear first, followed by the skill's anti-patterns appended after it.

**Acceptance criteria:**
- Assembled context contains a single anti-pattern section.
- The role's anti-patterns precede the skill's anti-patterns in that section.

---

### FR-011: Step 7 — Knowledge Entry Integration Point

The pipeline MUST invoke the knowledge auto-surfacing subsystem at step 7 to retrieve relevant knowledge entries for the current task context. The returned entries MUST be included at position 8 in the assembled output (after examples, before evaluation criteria). If the knowledge subsystem returns no entries, the knowledge section MUST be omitted from the assembled output.

**Acceptance criteria:**
- When knowledge entries are returned, they appear between the examples section and the evaluation criteria section in the assembled output.
- When no knowledge entries are returned, the assembled output contains no empty knowledge section or placeholder.

---

### FR-012: Step 8 — Tool Subset Guidance

The pipeline MUST read the role's `tools` field and generate a guidance text section listing which tools the role should use. This guidance MUST appear in the assembled context. The pipeline MUST NOT dynamically filter or remove tools from the MCP tool list.

**Acceptance criteria:**
- Assembled context for a role with `tools: [entity, doc, status, knowledge]` contains a guidance section listing those tool names.
- All MCP tools remain available to the agent regardless of the guidance text (no hard filtering).
- A role with no `tools` field produces no tool guidance section in the assembled output.

---

### FR-013: Step 9 — Token Budget Estimation

The pipeline MUST estimate the total token count of the assembled context by summing: role context, skill context, orchestration metadata, knowledge entries, tool guidance, and task-specific content.

**Acceptance criteria:**
- The pipeline produces a numeric token estimate for every successful assembly.
- The estimate is available in the pipeline's internal state before the final assembly step.

---

### FR-014: Token Budget Warning

If the estimated token count exceeds 40% of the context window, the pipeline MUST emit a warning in the assembled output's metadata. The warning MUST state the estimated token count and the 40% threshold value.

**Acceptance criteria:**
- Assembled context whose estimated tokens exceed 40% of the context window contains a warning with the estimated count and the threshold.
- Assembled context whose estimated tokens are at or below 40% contains no such warning.

---

### FR-015: Token Budget Refusal

If the estimated token count exceeds 60% of the context window, the pipeline MUST refuse to assemble the context. The refusal MUST return an error that states the estimated token count, the 60% threshold, and a suggestion to split the work unit.

**Acceptance criteria:**
- When estimated tokens exceed 60%, the pipeline returns an error (not assembled context) containing the estimate, the threshold, and "split" (or equivalent suggestion).
- No partial assembled context is returned on refusal.

---

### FR-016: Step 10 — Attention-Curve Output Ordering

The pipeline MUST assemble the final output in the following fixed order of sections:

| Position | Content                                | Attention Zone |
|----------|----------------------------------------|----------------|
| 1        | Project identity and hard constraints  | High           |
| 2        | Role identity                          | High           |
| 3        | Effort budget and orchestration pattern| High           |
| 4        | Combined vocabulary                    | High           |
| 5        | Combined anti-patterns                 | Medium-High    |
| 6        | Skill procedure (numbered steps)       | Medium         |
| 7        | Output format and examples             | Rising         |
| 8        | Knowledge entries                      | Rising         |
| 9        | Evaluation criteria                    | High           |
| 10       | Retrieval anchors                      | High           |

**Acceptance criteria:**
- The assembled output contains sections in the order listed above, verifiable by section headings or markers.
- No section appears out of the specified order.
- Optional sections (e.g., knowledge entries when none match) are omitted without affecting the order of remaining sections.

---

### FR-017: Within-Section Recency Bias Ordering

Within each section of the assembled output, the pipeline MUST order items so that the most critical item appears LAST. This applies to vocabulary terms, anti-patterns, and knowledge entries.

**Acceptance criteria:**
- In the vocabulary section, the most important vocabulary term (as determined by the source ordering in the role/skill file, where last = most critical) appears at the end.
- In the anti-pattern section, the most dangerous anti-pattern appears at the end.
- The ordering within sections is deterministic for the same input.

---

### FR-018: Progressive Disclosure Layers

The pipeline MUST implement progressive disclosure with four layers that control what content is loaded:

- **Layer 1 (Always loaded, ~300–500 tokens):** Base role identity, hard constraints, specific role identity, and vocabulary.
- **Layer 2 (Task-triggered, ~500–2,000 tokens):** Skill procedure, anti-patterns, output format, and examples.
- **Layer 3 (On-demand, 2,000+ tokens):** Full spec sections, reference documents, extended anti-pattern documentation. Loaded only when the skill procedure explicitly calls for it.
- **Layer 4 (Compressed, variable):** Summaries of large documents, collated findings from prior passes.

The pipeline MUST stop loading additional layers when the token budget is met.

**Acceptance criteria:**
- Layer 1 content is present in every successful assembly.
- Layer 2 content is present when a skill is bound to the stage.
- Layer 3 content is absent unless the skill procedure contains an explicit reference-loading instruction.
- Assembly stops adding layers when the next layer would push the estimate past the budget threshold.

---

## 4. Non-Functional Requirements

### NFR-001: Assembly Latency

The context assembly pipeline MUST complete within 2 seconds for typical inputs (role with inheritance, one skill, ≤10 knowledge entries). The pipeline MUST NOT make network calls; all inputs are local filesystem reads and in-process knowledge queries.

**Acceptance criteria:**
- Benchmark test assembling context with a role, inherited parent role, one skill, and 10 knowledge entries completes in under 2 seconds.

---

### NFR-002: Deterministic Output

Given identical inputs (same task, same role/skill/binding state, same knowledge base state), the pipeline MUST produce byte-identical output. Assembly MUST NOT depend on map iteration order, wall-clock time, or random values.

**Acceptance criteria:**
- Running assembly twice with the same inputs produces identical output strings.

---

### NFR-003: Backward Compatibility

The extended `handoff` tool MUST remain callable with its existing parameters (`task_id`, `role`, `instructions`). Callers that do not use roles or bindings MUST continue to receive assembled context (falling back to the pre-pipeline assembly path if no binding exists for the resolved stage).

**Acceptance criteria:**
- Existing `handoff` calls that work today continue to produce output without error after the pipeline is integrated.
- A task whose parent feature's stage has no configured binding falls back to the current assembly behaviour.

---

### NFR-004: Error Message Quality

Every error returned by the pipeline MUST identify: (a) the pipeline step that failed, (b) the specific entity that caused the failure (task ID, feature ID, role name, skill name, or stage name), and (c) a remediation hint.

**Acceptance criteria:**
- Each error path in the pipeline produces a message containing the step number or name, the failing entity identifier, and a suggestion for resolution.

---

### NFR-005: Testability

Each pipeline step MUST be independently testable. The pipeline MUST accept interfaces for role loading, skill loading, binding lookup, and knowledge querying so that tests can substitute mock implementations.

**Acceptance criteria:**
- Unit tests exist for each step in isolation using mock/stub inputs.
- Integration test exercises the full 10-step pipeline with filesystem-backed test fixtures.

---

## 5. Acceptance Criteria

| Requirement | Verification Method |
|-------------|---------------------|
| FR-001 | Integration test: `handoff(task_id=...)` returns structured Markdown prompt |
| FR-002 | Unit test: invalid feature state → error with current and required states |
| FR-003 | Unit test: task resolves to parent feature stage |
| FR-004 | Unit test: known stage returns binding; unknown stage returns error |
| FR-005 | Unit test: excluded content category absent from output |
| FR-006 | Integration test: orchestration metadata appears at position 3 in output |
| FR-007 | Unit test: inherited role merges vocabulary and anti-patterns in correct order |
| FR-008 | Unit test: missing skill → error; present skill → sections loaded |
| FR-009 | Unit test: merged vocabulary has role terms before skill terms |
| FR-010 | Unit test: merged anti-patterns has role items before skill items |
| FR-011 | Integration test: knowledge entries appear at position 8; absent when none match |
| FR-012 | Unit test: tool guidance section lists role's tools; no hard filtering occurs |
| FR-013 | Unit test: token estimate is a positive integer for any valid assembly |
| FR-014 | Unit test: assembled context includes warning when estimate > 40% threshold |
| FR-015 | Unit test: pipeline returns error (not context) when estimate > 60% threshold |
| FR-016 | Integration test: section order matches the 10-position table |
| FR-017 | Unit test: last item in vocabulary/anti-pattern sections is the most critical |
| FR-018 | Unit test: Layer 3 content absent when procedure does not request it |
| NFR-001 | Benchmark test: assembly completes within 2 seconds |
| NFR-002 | Test: two identical assemblies produce byte-identical output |
| NFR-003 | Integration test: pre-existing `handoff` calls succeed unchanged |
| NFR-004 | Test: each error path contains step name, entity ID, and remediation hint |
| NFR-005 | Code review: each step has independent unit tests with mock inputs |

---

## 6. Dependencies and Assumptions

### Dependencies

- **Role System feature:** Roles with the specified schema (`id`, `inherits`, `identity`, `vocabulary`, `anti_patterns`, `tools`) must exist in `.kbz/roles/`.
- **Skill System feature:** Skills with the specified structure (vocabulary, anti-patterns, procedure, output format, examples, evaluation criteria, retrieval anchors) must exist in `.kbz/skills/`.
- **Binding Registry feature:** Stage bindings mapping lifecycle stages to roles, skills, orchestration patterns, and effort budgets must be loadable by the pipeline.
- **Knowledge Auto-Surfacing feature:** Step 7 delegates to the knowledge auto-surfacing subsystem. That subsystem's matching and capping logic is specified separately.
- **Existing `handoff` tool:** The pipeline extends `internal/mcp/handoff.go` (or equivalent). The current tool must be functional as the integration target.

### Assumptions

- The context window size is known or configurable so that 40% and 60% thresholds can be computed. A default context window size is assumed if not configured.
- Token estimation uses an approximation (e.g., character count / 4 or a tokenizer library). Exact token counts are not required; estimates within ±10% are acceptable.
- Role inheritance is single-parent only (no multiple inheritance). A role declares at most one `inherits` value.
- The binding registry is loaded once at pipeline start and does not change during assembly.