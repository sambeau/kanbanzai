# Implementation Plan: Implementation Skill Content

| Field   | Value                                                              |
|---------|--------------------------------------------------------------------|
| Status  | Draft                                                              |
| Feature | FEAT-01KN5-88PG7HA3 (implementation-skill-content)                |
| Spec    | `work/spec/3.0-implementation-skill-content.md`                    |
| Design  | `work/design/skills-system-redesign-v2.md` §5.2, §3.2, §8, §9    |

---

## 1. Overview

This plan decomposes the implementation skill content specification into assignable tasks for AI agents. The specification requires authoring three SKILL.md files (`implement-task`, `orchestrate-development`, `decompose-feature`) that cover the developing and dev-planning stages of the feature lifecycle. `implement-task` is deliberately lean, deferring language expertise to implementer roles. `orchestrate-development` carries three explicit context compaction techniques. `decompose-feature` invests disproportionately in decomposition validation as the strongest predictor of downstream workflow success.

**Scope boundaries (from specification):**

- IN: Content for three SKILL.md files, `references/` files where needed to stay under 500 lines.
- OUT: SKILL.md format schema, context assembly pipeline, binding registry schema, role content, `decompose` tool implementation, language-specific coding conventions, sub-agent dispatch mechanism.

**Key structural decision:** All three skills are independent of each other. They share the same attention-curve format conventions established by the document authoring skill content feature (or can derive them from the design §3.2 directly). No conventions task is needed here — each agent reads the design §3.2 and §9.2 directly, plus the CONVENTIONS.md produced by the document authoring skill content feature if available. All three tasks execute in parallel.

---

## 2. Task Breakdown

### Task 1: Author `implement-task/SKILL.md`

**Objective:** Create the individual task execution skill. This skill guides an agent paired with an `implementer-*` role through executing a single assigned task during the `developing` stage. The skill must be deliberately lean — it defines a read → implement → test → verify procedure and carries anti-patterns for common task execution failures. It must NOT contain language-specific coding guidance, style rules, or framework conventions (those belong in the paired implementer role). This skill must be the shortest of the three implementation skills.

**Specification references:** FR-001, FR-002 (stage: `developing`), FR-003 (roles: `implementer`, `implementer-go`), FR-004 (attention-curve ordering), FR-005 (lean procedure: read → implement → test → verify), FR-006 (anti-patterns: scope creep, untested code paths, spec deviation + 2–7 more), FR-013 (frontmatter: `constraint_level: medium`), FR-014 (BAD/GOOD examples of task execution), FR-015 (evaluation criteria: spec conformance and test coverage focus), NFR-001 (≤500 lines), NFR-002 (no vocabulary overlap with implementer roles), NFR-004 (shortest of the three skills)

**Input context:**
- `work/spec/3.0-implementation-skill-content.md` FR-005, FR-006 — lean procedure and required anti-patterns
- `work/design/skills-system-redesign-v2.md` §3.2 — attention-curve SKILL.md format
- `work/design/skills-system-redesign-v2.md` §5.2 — implementation skill definitions
- `work/design/skills-system-redesign-v2.md` §9.2 — quality gate checklist
- `.kbz/skills/CONVENTIONS.md` — shared formatting conventions (if available from document authoring feature; otherwise derive from §3.2 directly)
- `.agents/skills/kanbanzai-agents/SKILL.md` — existing agent skill for tone reference on task execution patterns

**Output artifacts:**
- `.kbz/skills/implement-task/SKILL.md` — complete skill file with frontmatter (`name: implement-task`, `stage: developing`, `roles: [implementer, implementer-go]`, `constraint_level: medium`), all attention-curve sections
- `.kbz/skills/implement-task/references/` — overflow content if needed (unlikely given lean requirement)

**Dependencies:** None — can start immediately.

**Key constraints:**
- The procedure section must contain exactly four phases: read spec/acceptance criteria → implement → write/update tests → verify
- No language-specific instructions (no Go, Python, TypeScript references)
- Must be shorter in total line count than both `orchestrate-development` and `decompose-feature`
- Vocabulary must focus on task-execution methodology terms (e.g., "acceptance criterion", "spec conformance", "test coverage"), NOT language/domain terms that belong in implementer roles

---

### Task 2: Author `orchestrate-development/SKILL.md`

**Objective:** Create the development orchestration skill. This skill guides an agent paired with the `orchestrator` role through coordinating multi-task development during the `developing` stage. The critical content is three context compaction techniques that must be described with enough specificity to follow as procedures: (1) post-completion summarisation to 2–3 sentences + task ID, (2) document-based offloading at 60% context utilisation, (3) single-feature scoping — no multi-feature orchestration in one session.

**Specification references:** FR-001, FR-002 (stage: `developing`), FR-003 (role: `orchestrator`), FR-004 (attention-curve ordering), FR-007 (coordination procedure: read plan → identify parallel tasks → dispatch → monitor → compact), FR-008 (three context compaction techniques with specific thresholds), FR-009 (vocabulary: 15–30 terms for parallel dispatch, dependency ordering, context compaction, failure recovery), FR-013 (frontmatter: `constraint_level: medium`), FR-014 (BAD/GOOD examples of multi-task coordination), FR-015 (evaluation criteria: dependency respect, compaction discipline, progress tracking), NFR-001 (≤500 lines), NFR-002 (no vocabulary overlap with orchestrator role), NFR-003 (one-level-deep references)

**Input context:**
- `work/spec/3.0-implementation-skill-content.md` FR-007, FR-008, FR-009 — procedure, compaction techniques, vocabulary requirements
- `work/design/skills-system-redesign-v2.md` §3.2 — attention-curve SKILL.md format
- `work/design/skills-system-redesign-v2.md` §5.2 — implementation skill definitions
- `work/design/skills-system-redesign-v2.md` §9.2 — quality gate checklist
- `.kbz/skills/CONVENTIONS.md` — shared formatting conventions (if available)
- `refs/sub-agents.md` — sub-agent delegation conventions for vocabulary and procedure derivation
- `.agents/skills/kanbanzai-agents/SKILL.md` — existing agent skill (covers some orchestration patterns)
- `internal/mcp/handoff_tool.go` — handoff tool interface for dispatch vocabulary

**Output artifacts:**
- `.kbz/skills/orchestrate-development/SKILL.md` — complete skill file with frontmatter (`name: orchestrate-development`, `stage: developing`, `roles: [orchestrator]`, `constraint_level: medium`), all attention-curve sections including the three compaction techniques
- `.kbz/skills/orchestrate-development/references/` — overflow content (likely needed — compaction technique examples may push past 500 lines)

**Dependencies:** None — can start immediately.

**Key constraints:**
- The procedure must contain all five phases in order: (1) read dev-plan, (2) identify parallel-dispatchable tasks, (3) dispatch sub-agents, (4) monitor + handle failures, (5) perform context compaction
- The procedure must explicitly state that tasks with unmet dependencies must NOT be dispatched
- All three compaction techniques must be present with actionable specificity:
  - Post-completion summarisation: 2–3 sentences + task ID; full sub-agent output NOT retained
  - Document-based offloading: triggered at 60% context utilisation; write progress summary to registered document, start fresh session
  - Single-feature scoping: complete one feature's tasks → write summary → begin next feature; no multi-feature sessions
- Vocabulary must include terms for parallel dispatch, dependency ordering, context compaction, and failure recovery (at least 2 terms per category)

---

### Task 3: Author `decompose-feature/SKILL.md`

**Objective:** Create the feature decomposition skill. This skill guides an agent paired with the `architect` role through decomposing a feature into tasks during the `dev-planning` stage using the `decompose` MCP tool. The skill must invest disproportionately in decomposition validation — a 5-point validate → fix → re-validate loop that checks: (1) non-empty descriptions, (2) explicit dependencies, (3) single-agent sizing, (4) no circular dependencies, (5) integration/test task presence.

**Specification references:** FR-001, FR-002 (stage: `dev-planning`), FR-003 (role: `architect`), FR-004 (attention-curve ordering), FR-010 (5-point validation loop: validate → fix → re-validate), FR-011 (anti-patterns: over-decomposition, circular dependencies, missing integration tasks + 2–7 more), FR-012 (vocabulary: 15–30 terms for dependency analysis, vertical slicing, task sizing, gap detection), FR-013 (frontmatter: `constraint_level: low`), FR-014 (BAD/GOOD examples of feature breakdown), FR-015 (evaluation criteria: decomposition quality, dependency correctness, gap coverage), NFR-001 (≤500 lines), NFR-002 (no vocabulary overlap with architect role), NFR-003 (one-level-deep references)

**Input context:**
- `work/spec/3.0-implementation-skill-content.md` FR-010, FR-011, FR-012 — validation loop, anti-patterns, vocabulary
- `work/design/skills-system-redesign-v2.md` §3.2 — attention-curve SKILL.md format
- `work/design/skills-system-redesign-v2.md` §5.2 — implementation skill definitions
- `work/design/skills-system-redesign-v2.md` §9.2 — quality gate checklist
- `.kbz/skills/CONVENTIONS.md` — shared formatting conventions (if available)
- `.agents/skills/kanbanzai-planning/SKILL.md` — existing planning skill for decomposition vocabulary reference
- `work/templates/implementation-plan-prompt-template.md` — plan template for understanding what decomposition output feeds into
- Example plans in `work/plan/` — for deriving anti-patterns and examples from real decomposition successes/failures

**Output artifacts:**
- `.kbz/skills/decompose-feature/SKILL.md` — complete skill file with frontmatter (`name: decompose-feature`, `stage: dev-planning`, `roles: [architect]`, `constraint_level: low`), all attention-curve sections with the 5-point validation loop as the centrepiece of the procedure
- `.kbz/skills/decompose-feature/references/` — overflow content (likely needed — validation loop examples and extended anti-pattern documentation)

**Dependencies:** None — can start immediately.

**Key constraints:**
- The validation phase must be structured as an explicit loop: validate → if issues → fix → re-validate (not a single pass)
- The skill text must characterise decomposition validation as the most important part of the procedure (disproportionate emphasis)
- All five validation checks must be phrased as testable conditions:
  1. Every task has a clear, non-empty description
  2. Dependencies between tasks are explicitly declared
  3. Each task is sized for single-agent completion
  4. No circular dependencies exist
  5. Integration/test tasks are present
- The three required anti-patterns (over-decomposition, circular dependencies, missing integration tasks) must each have Name, Detect, BECAUSE, Resolve
- Vocabulary must include terms for dependency analysis, vertical slicing, sizing, and gap detection (at least 2 terms per category)
- `constraint_level: low` reflects high analytical freedom within validation constraints

---

## 3. Dependency Graph

```
Task 1: implement-task      ─┐
Task 2: orchestrate-development  ├──→ All complete
Task 3: decompose-feature   ─┘
```

**Parallelism:** All three tasks are fully independent and can execute in parallel immediately. Maximum parallelism is 3 concurrent agents.

**Serial constraints:** None. There are no dependencies between tasks. Each agent reads the specification and design directly for format conventions.

**Cross-feature dependency note:** If the document authoring skill content feature (FEAT-01KN5-88PFWADY) has completed its Task 1 (shared conventions), agents here can reference `.kbz/skills/CONVENTIONS.md` for formatting consistency. This is a soft dependency — agents can derive the same conventions from the design §3.2 directly.

---

## 4. Interface Contracts

### 4.1 SKILL.md Frontmatter Schema (shared across all tasks)

Every SKILL.md produced by Tasks 1–3 must use this frontmatter structure:

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

Specific values per task:

| Field | Task 1: implement-task | Task 2: orchestrate-development | Task 3: decompose-feature |
|-------|----------------------|-------------------------------|-------------------------|
| `stage` | `developing` | `developing` | `dev-planning` |
| `roles` | `[implementer, implementer-go]` | `[orchestrator]` | `[architect]` |
| `constraint_level` | `medium` | `medium` | `low` |

### 4.2 Attention-Curve Section Order (shared across all tasks)

Every SKILL.md must contain sections in exactly this order:
1. Frontmatter (YAML)
2. `## Vocabulary` — 15–30 terms, each as `**term** — definition`
3. `## Anti-Patterns` — 5–10 entries, each with Name / Detect / BECAUSE / Resolve
4. `## Checklist` — optional; include for `constraint_level: medium` (Tasks 1, 2) and `low` (Task 3)
5. `## Procedure` — step-by-step process
6. `## Output Format` — what the skill produces
7. `## Examples` — BAD (with `WHY BAD`) then GOOD (with `WHY GOOD`); GOOD last
8. `## Evaluation Criteria` — 4–8 gradable questions with weight (`required`/`high`/`medium`)
9. `## Questions This Skill Answers` — 5–10 retrieval-anchor questions

### 4.3 Anti-Pattern Entry Format (shared across all tasks)

Each anti-pattern must follow this structure:

```
### {Anti-Pattern Name}

**Detect:** {observable signal that this anti-pattern is occurring}

**BECAUSE:** {why this is harmful — downstream consequence, not restatement of detection}

**Resolve:** {concrete corrective action}
```

### 4.4 Vocabulary Non-Overlap Contract

Each skill's vocabulary must not duplicate terms from its paired role (NFR-002). The division of responsibility:

| Skill | Skill vocabulary carries | Role vocabulary carries |
|-------|-------------------------|----------------------|
| `implement-task` | Task execution methodology terms | Language-specific terms (Go conventions, test frameworks) |
| `orchestrate-development` | Coordination and compaction terms | Orchestration infrastructure terms (tool names, dispatch mechanisms) |
| `decompose-feature` | Decomposition methodology terms | Architecture and design domain terms |

### 4.5 Stage Sharing Contract (Tasks 1 and 2)

`implement-task` and `orchestrate-development` share the `developing` stage. They are distinguished by their paired roles:
- `implement-task` + `implementer-*` = the worker agent executing individual tasks
- `orchestrate-development` + `orchestrator` = the coordinator dispatching and monitoring workers

The binding registry resolves this via the `orchestration_pattern: orchestrator-workers` declaration for the `developing` stage. Neither skill needs to reference the other, but both must be coherent with this topology. Specifically:
- `implement-task` should not contain orchestration concerns (dispatching, monitoring, aggregation)
- `orchestrate-development` should not contain implementation concerns (writing code, running tests)

---

## 5. Traceability Matrix

| Requirement | Task(s) | Notes |
|-------------|---------|-------|
| FR-001 | 1, 2, 3 | Each task creates one of the three skills |
| FR-002 | 1, 2, 3 | Task 1: `developing`; Task 2: `developing`; Task 3: `dev-planning` |
| FR-003 | 1, 2, 3 | Task 1: `implementer, implementer-go`; Task 2: `orchestrator`; Task 3: `architect` |
| FR-004 | 1, 2, 3 | All tasks follow attention-curve section ordering |
| FR-005 | 1 | Lean 4-phase procedure: read → implement → test → verify |
| FR-006 | 1 | 5–10 anti-patterns including scope creep, untested code paths, spec deviation |
| FR-007 | 2 | 5-phase coordination procedure with dependency ordering |
| FR-008 | 2 | Three compaction techniques with specific thresholds |
| FR-009 | 2 | 15–30 coordination vocabulary terms |
| FR-010 | 3 | 5-point validation loop: validate → fix → re-validate |
| FR-011 | 3 | 5–10 anti-patterns including over-decomposition, circular deps, missing integration |
| FR-012 | 3 | 15–30 decomposition vocabulary terms |
| FR-013 | 1, 2, 3 | All tasks populate complete frontmatter with specified constraint_level values |
| FR-014 | 1, 2, 3 | All tasks write BAD/GOOD examples specific to their domain |
| FR-015 | 1, 2, 3 | All tasks write 4–8 evaluation criteria as gradable questions |
| NFR-001 | 1, 2, 3 | All skills ≤500 lines; overflow to `references/` |
| NFR-002 | 1, 2, 3 | All skills avoid vocabulary overlap with paired roles |
| NFR-003 | 1, 2, 3 | All reference files linked one-level-deep from SKILL.md |
| NFR-004 | 1 | `implement-task` must be the shortest of the three skills |