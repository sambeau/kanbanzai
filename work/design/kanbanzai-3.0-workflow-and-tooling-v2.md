# Design: Kanbanzai 3.0 Workflow Engine and MCP Tooling

| Field | Value |
|-------|-------|
| Date | 2025-07-30 |
| Status | Draft |
| Author | Design Agent |
| Based on | `work/design/kanbanzai-3.0-workflow-and-tooling.md` (pre-alignment version) |
| Informed by | `work/design/orchestration-recommendations.md`, `work/research/agent-orchestration-research.md` |
| Related | `work/design/skills-system-redesign-v2.md`, `work/design/Kanbanzai-3.0-proposal.txt`, `work/design/smart-lifecycle-transitions.md` |
| Alignment | Changes from `work/reviews/3.0-design-cross-document-alignment.md` to be applied to this copy |

---

## 1. Purpose

This document defines the Kanbanzai 3.0 changes to the **workflow engine**, **MCP tool surface**, and **context assembly pipeline** — everything that is *not* the skills and roles system (which is covered by `work/design/skills-system-redesign.md`).

The skills redesign answers "what do agents know and how are they shaped?" This document answers "what does the system enforce, how do tools behave, and what context do agents receive?"

The two designs share an integration surface — the **binding registry** — which is defined in the skills redesign and consumed by the systems described here. This document does not redefine the binding registry; it specifies how the MCP server reads it and acts on it.

### 1.1 Scope Boundary with Skills Redesign

| Concern | Owner |
|---------|-------|
| Role definitions, vocabulary, anti-patterns | Skills redesign |
| Skill procedures and templates | Skills redesign |
| Binding registry schema and content | Skills redesign |
| Constraint levels (low/medium/high freedom) | Skills redesign |
| Review sub-agent topology declaration | Skills redesign |
| Context compaction guidance in orchestrator skill | Skills redesign |
| Stage gate enforcement in `entity(action: "transition")` | **This document** |
| MCP tool description quality (ACI audit) | **This document** |
| Actionable error messages across all tools | **This document** |
| Role-scoped tool subset filtering in context assembly | **This document** |
| Lifecycle-aware context assembly in `handoff` / `next` | **This document** |
| Effort budgets embedded in assembled context | **This document** |
| Structured document templates and gate checking | **This document** |
| Review-rework loop formalisation | **This document** |
| Orchestration pattern signalling per stage | **This document** |
| Action pattern logging (observability) | **This document** |
| Decomposition quality validation in `decompose` | **This document** |
| Filesystem-output convention reinforcement | **This document** |

### 1.2 Relationship to Existing Work

The **smart-lifecycle-transitions design** (`work/design/smart-lifecycle-transitions.md`) already implemented:

- The `advance` parameter for multi-step feature transitions.
- Document-based stage gate prerequisites (design → spec → dev-plan → tasks).
- Improved error messages with valid next states on transition failure.

This document extends that foundation in three ways:

1. **Stronger prerequisites** — adding task-completeness gates for `developing → reviewing` and review-report gates for `reviewing → done`, which were out of scope for the original design.
2. **Review-rework loop** — formalising the `reviewing ↔ needs-rework` cycle with an iteration cap.
3. **Binding registry integration** — reading prerequisite definitions from the binding registry rather than hardcoding them, so the skills design can evolve prerequisites without MCP server code changes.

---

## 2. Design Principles

These are specific to the workflow engine and tooling. They complement, not duplicate, the skills redesign's design principles.

### WP-1: The System Prevents Skipped Steps; Skills Prevent Bad Steps

The MCP server enforces *hard constraints* (ℋ) — prerequisites that must be met before a transition is allowed. If the constraint isn't met, the transition is rejected. Skills and roles handle *soft constraints* (𝒮) — quality expectations, thoroughness, approach. The system makes it impossible to skip specification; the skill makes the specification good.

*Source: Masters et al. (hard constraints ℋ vs soft constraints 𝒮), MetaGPT (SOPs with verification).*

### WP-2: Tools Are Interfaces for Agents, Not APIs for Developers

MCP tool descriptions are the primary interface agents use to decide what to do. They must be designed as Agent-Computer Interfaces (ACIs): leading with "when to use", including negative guidance, stating workflow position, and staying concise. Every token in a tool description is processed on every tool call — brevity is a performance characteristic.

*Source: SWE-agent (ACI design), Anthropic (40% faster task completion from optimised tool descriptions).*

### WP-3: Errors Are Instructions, Not Diagnostics

When a tool call fails, the error message is the agent's primary recovery mechanism. Every error must include what failed, why it failed, and what to do next. A diagnostic message ("invalid transition") sends the agent into a retry loop. An instructional message ("cannot transition: specification not approved — call `doc(action: 'list', owner: 'FEAT-001', pending: true)` to see pending documents") gives the agent a recovery path in one step.

*Source: SWE-agent (ACI poka-yoke principle), Anthropic ("make wrong usage hard").*

### WP-4: Context Assembly Is Stage-Aware

Different workflow stages need different context. A specification task needs the design document and a template. An implementation task needs the relevant spec section and file paths. A review task needs the implementation diff and a rubric. The assembly pipeline should vary what it includes based on the stage, not assemble the same structure for every task.

*Source: Anthropic (subagent needs objective, output format, tool guidance, task boundaries).*

### WP-5: The Binding Registry Is the Decision Table

The orchestration layer does not decide how to orchestrate a stage — it looks it up in the binding registry. The binding registry (defined in the skills redesign) maps each stage to roles, skills, tool subsets, orchestration topology, effort budgets, and prerequisites. The MCP server reads this registry and acts on it mechanically.

*Source: Masters et al. (task graph structure is the critical path), Google Research (architecture must match task structure).*

---

## 3. Stage Gate Enforcement

### 3.1 Current State

The smart-lifecycle-transitions design implemented document-based stage gates for the `advance` operation:

| Transition | Gate |
|---|---|
| `proposed → designing` | None |
| `designing → specifying` | Approved design document |
| `specifying → dev-planning` | Approved specification document |
| `dev-planning → developing` | At least one child task exists |
| `developing → reviewing` | Never skippable via advance |

These gates apply only during `advance` (multi-step transitions). Single-step transitions bypass them — an agent can call `entity(action: "transition", id: "FEAT-001", status: "dev-planning")` from `specifying` without an approved spec.

### 3.2 What Changes

Stage gates become **mandatory for all transitions**, not just during `advance`. The gate between `specifying` and `dev-planning` fires whether the transition is a single step or part of an advance sequence. This converts "agents skip steps" from a quality problem into an impossibility.

Additionally, two new gates are added for the review lifecycle:

| Transition | Gate | Rationale |
|---|---|---|
| `developing → reviewing` | All child tasks in terminal state (`done` or `not-planned`) | Cannot review incomplete work |
| `reviewing → done` | Review report document registered; no blocking findings open | Cannot close without documented review |

### 3.3 Gate Prerequisite Table (Complete)

| Transition | Prerequisites |
|---|---|
| `proposed → designing` | None |
| `designing → specifying` | Approved design document owned by feature or parent plan |
| `specifying → dev-planning` | Approved specification document owned by feature or parent plan |
| `dev-planning → developing` | Approved dev-plan document; at least one child task exists |
| `developing → reviewing` | All child tasks in terminal state |
| `reviewing → done` | Review report registered; no unresolved blocking findings |
| `reviewing → needs-rework` | None (this is a judgment call by the reviewer) |
| `needs-rework → developing` | At least one rework task exists |
| `needs-rework → reviewing` | All rework tasks in terminal state |

### 3.4 Gate Override

There must be an escape hatch. A human (or an orchestrator with explicit human delegation) can override a gate with a reason:

```
entity(action: "transition", id: "FEAT-001", status: "dev-planning",
       override: true, override_reason: "Spec exists in external system, imported later")
```

Override transitions are logged with the reason. The `health` tool should flag features that advanced via override as attention items.

### 3.5 Binding Registry as Gate Source

The gate definitions in §3.3 are the initial hardcoded set. In the medium term, the MCP server should read gate prerequisites from the binding registry's `prerequisites` block per stage. This allows the skills design to evolve prerequisites (e.g., adding a "security review required" gate for certain feature tags) without MCP server code changes.

The binding registry already declares prerequisites in this format:

```
prerequisites:
  documents:
    - type: specification
      status: approved
  tasks:
    min_count: 1
```

The MCP server's transition handler reads this structure and evaluates it against the feature's current state.

### 3.6 Error Messages for Gate Failures

Gate failures must produce actionable errors (per WP-3):

**Good:**
```
Cannot transition FEAT-001 from "specifying" to "dev-planning":
no approved specification document found.

To resolve:
1. Check pending documents: doc(action: "list", owner: "FEAT-001", pending: true)
2. If the spec exists, approve it: doc(action: "approve", id: "DOC-...")
3. If no spec exists, register one: doc(action: "register", path: "...", type: "specification", owner: "FEAT-001")
```

**Bad:**
```
invalid feature transition "specifying" → "dev-planning"
```

---

## 4. Review-Rework Loop Formalisation

### 4.1 Problem

The `reviewing` and `needs-rework` states exist but their interaction is informal. There is no iteration cap, no structured way to track how many review cycles a feature has been through, and no escalation path when rework loops repeat.

### 4.2 Review Cycle Tracking

Add a `review_cycle` counter to the feature entity. It increments each time the feature transitions into `reviewing`.

| Event | Counter Change |
|---|---|
| First entry to `reviewing` | `review_cycle: 1` |
| `needs-rework → developing → reviewing` | `review_cycle: 2` |
| Each subsequent return to `reviewing` | Increment by 1 |

The counter is stored on the feature entity and visible in `status` output.

### 4.3 Iteration Cap

The binding registry declares `max_review_cycles` per stage (currently set to 3 in the skills redesign). When a feature's `review_cycle` reaches the cap, the system blocks automatic transition back to `needs-rework` and requires human intervention.

Behaviour when the cap is reached:

1. The review verdict is recorded normally.
2. If the verdict is "fail", the system does **not** auto-transition to `needs-rework`.
3. Instead, the feature enters a `blocked` state with a reason: "Review iteration cap reached (3/3). Human decision required: accept with known issues, rework with revised scope, or cancel."
4. A human checkpoint is created automatically.

This prevents the infinite refinement loop identified as a common anti-pattern in multi-agent systems.

### 4.4 Focused Re-Review

When a feature returns to `reviewing` after rework (cycle ≥ 2), the context assembly should signal that this is a **focused re-review**, not a full review:

- Include only the rework tasks and their changes, not the full implementation.
- Include the previous review findings that triggered rework.
- Include the rework task descriptions showing what was supposed to change.

The reviewing skill handles the methodology; the context assembly handles what information is surfaced.

---

## 5. MCP Tool Description Audit (ACI Redesign)

### 5.1 Problem

MCP tool descriptions are currently written as API documentation — they explain parameters and actions. Research shows they should be designed as agent interfaces: answering "when should I use this?", "what should I use instead?", and "where does this fit in the workflow?"

Anthropic's team found that optimising tool descriptions yielded 40% faster task completion and reported spending "more time optimizing our tools than the overall prompt." Google Research identifies a tool-use bottleneck at 16+ tools — with 22+ tools in the current surface, description quality is critical.

### 5.2 ACI Description Principles

Every tool description should follow these rules:

1. **Lead with "when to use".** The first sentence answers when and why, not what.
2. **Include negative guidance.** "Use this INSTEAD OF reading .kbz/ files directly" or "Do NOT use this for X — use Y instead."
3. **State workflow position.** "Call AFTER specification is approved" or "Call BEFORE dispatching sub-agents."
4. **Make parameter relationships explicit.** "When action is 'create', type is required."
5. **Stay under 200 tokens.** Agents process all tool descriptions on every call; brevity matters.

### 5.3 Example: `entity` Tool

**Current (conceptual):**
> Create, read, update, and transition entities (plans, features, tasks, bugs, epics, decisions). Use action: get or action: list to query entities — these return structured data with lifecycle state and cross-references. Do not read .kbz/state/ YAML files directly.

**ACI-optimised:**
> The primary tool for workflow state. Use INSTEAD OF reading .kbz/state/ files. Actions: create (new entity), get (by ID), list (filtered query), update (modify fields), transition (advance lifecycle — checks prerequisites, blocks if unmet). Start here when you need the current state of any feature, task, or plan. For document records, use `doc` instead.

### 5.4 Example: `handoff` Tool

**Current (conceptual):**
> Generate a complete sub-agent prompt from a task. The output is designed to go directly into spawn_agent's message parameter.

**ACI-optimised:**
> Assemble a targeted sub-agent prompt from a task. Call this BEFORE spawn_agent — it gathers spec sections, knowledge, file paths, role conventions, and effort guidance into a ready-to-dispatch prompt. The assembled context varies by lifecycle stage. Accepts tasks in active, ready, or needs-rework status.

### 5.5 Audit Scope

Every MCP tool description should be audited and rewritten. Priority order:

1. **High-frequency tools** (entity, doc, handoff, next, finish, status) — agents use these on nearly every task.
2. **Decision-point tools** (decompose, merge, pr) — wrong usage here has high impact.
3. **Query tools** (knowledge, doc_intel, profile) — less critical but still benefit from "when to use" framing.

This is a **description-only change** — no tool logic changes. It can be done incrementally, one tool at a time.

### 5.6 Audit Methodology: Agent-Driven Testing

Following Anthropic's practice — where they "built a tool-testing agent that used tools dozens of times and rewrote descriptions to avoid failures" — the ACI audit should include an iterative agent-driven testing step:

1. Give an agent a representative workflow task (e.g., "advance this feature from specifying to dev-planning") with only the MCP tool list for guidance.
2. Observe which tools it selects, in what order, and where it gets stuck or picks the wrong tool.
3. Rewrite descriptions to address observed failure modes.
4. Repeat with a fresh agent session until the tool selection path is reliable.

This is the most direct way to surface discoverability problems that aren't visible from a human review of descriptions. Maintain a set of 5–10 representative task scenarios for this purpose.

---

## 6. Actionable Error Messages

### 6.1 Principle

Every error response from an MCP tool must include three parts:

1. **What failed** — the fact, with entity IDs and context.
2. **Why it failed** — the prerequisite, constraint, or validation rule that was violated.
3. **What to do next** — a specific recovery action, ideally formatted as a tool call the agent can copy.

### 6.2 Error Template

```
Cannot {action} {entity}: {reason}.

To resolve:
  {recovery_step_1}
  {recovery_step_2}
```

### 6.3 Examples

**Transition blocked by gate:**
```
Cannot transition FEAT-001 to "developing": dev-plan document not approved.

To resolve:
1. List pending documents: doc(action: "list", owner: "FEAT-001", pending: true)
2. Approve the dev-plan: doc(action: "approve", id: "<doc-id>")
```

**Task finish with missing verification:**
```
Cannot finish TASK-042: no verification description provided.

To resolve:
  Add a verification summary: finish(task_id: "TASK-042", summary: "...", verification: "...")
```

**Decompose with no approved spec:**
```
Cannot decompose FEAT-003: no approved specification found.

To resolve:
1. Check feature status: entity(action: "get", id: "FEAT-003")
2. The feature must be in "dev-planning" or later with an approved spec.
```

### 6.4 Audit Scope

Audit all error paths in MCP tool handlers. The transition handler (§3.6) is the highest priority because gate failures are the most common error agents will encounter after enforcement is enabled. Other priorities:

- `finish` — validation failures on missing summary/verification.
- `doc` — registration failures, approval prerequisites.
- `decompose` — input validation, prerequisite checks.
- `entity(action: "create")` — missing required fields per entity type.

---

## 7. Lifecycle-Aware Context Assembly

### 7.1 Problem

The `handoff` and `next` tools currently assemble context with the same structure regardless of workflow stage. A specification task and an implementation task receive the same treatment. But different stages need fundamentally different context.

### 7.2 Lifecycle State Validation

Before assembling context, `handoff` and `next` must validate that the feature is in an appropriate lifecycle state for the requested work. If the feature is not in the correct state, the tool should **reject the request**, not silently assemble wrong-stage context.

The research is explicit on this point: "`handoff` / `next` tools should refuse to assemble implementation context for features not in the correct lifecycle state" (Research §4.1). This is the context-assembly counterpart to stage gates on transitions — together they make it structurally impossible to skip steps.

**Behaviour:**

- `handoff` for an implementation task on a feature in `specifying` → Error: "Cannot assemble implementation context for FEAT-001: feature is in 'specifying', not 'developing'. The feature must have an approved spec and dev-plan before implementation tasks can be dispatched."
- `next` claiming a task whose parent feature is in the wrong state → Error with the same pattern, explaining what state the feature needs to reach.

This validation uses the same prerequisite model as stage gates (§3) — the binding registry declares which stage each task type belongs to, and the assembly tool checks the feature's current state against it.

### 7.3 Stage-Specific Assembly Strategy

The assembly pipeline should dispatch on the feature's lifecycle stage to vary what's included:

| Stage | Primary Context | Excluded Context | Assembly Notes |
|---|---|---|---|
| **Designing** | Related decisions, parent plan context, design template, existing designs for reference | Implementation tools, file paths, test expectations | Single-agent framing: "Do not delegate sub-tasks." |
| **Specifying** | Approved design document (full), spec template, acceptance criteria format | Implementation tools, file paths | Single-agent framing. Include full design — specs must reference it. |
| **Dev-planning** | Approved spec (full), task decomposition guidance, dependency format, sizing constraints | Implementation details, review tools | Single-agent framing. Emphasise decomposition quality. |
| **Developing** | Approved spec (relevant sections only), task description, file paths, test expectations, related knowledge entries | Planning tools, review rubrics | Multi-agent framing: "Dispatch tasks to sub-agents in parallel." |
| **Reviewing** | Spec (relevant sections), implementation summary, review rubric, verdict format, previous review findings (if re-review) | Implementation tools, planning tools | Multi-agent framing: "Dispatch specialist reviewers in parallel." |

### 7.4 Orchestration Pattern Signalling

The assembled context should explicitly state the orchestration pattern for the stage, drawn from the binding registry's `orchestration` field:

- **`single-agent`**: "This is a single-agent task. Complete it directly — do not delegate to sub-agents."
- **`orchestrator-workers`**: "This is a multi-agent task. Dispatch independent sub-tasks to sub-agents in parallel using handoff + spawn_agent."

This is a simple text insertion into the assembled prompt, positioned in the high-attention zone (near the top). It directly addresses the 3.0 proposal's observation that agents are "too keen to get to implementation" — sequential stages explicitly prohibit delegation.

### 7.5 Implementation Approach

The full assembly pipeline is defined in the skills redesign (§6.1 of `work/design/skills-system-redesign-v2.md`). This document contributes the following stage-awareness requirements to that pipeline:

- **Lifecycle state validation** (§7.2): The pipeline must validate that the feature is in the correct lifecycle state before assembling context. Reject with an actionable error if not.
- **Stage-specific inclusion/exclusion** (§7.3): The pipeline must vary what context is included based on the feature's current lifecycle stage, using the inclusion/exclusion table above.
- **Orchestration pattern signalling** (§7.4): The pipeline must insert the orchestration pattern from the binding registry into the high-attention zone of the assembled context.
- **Effort budget positioning** (§8): The pipeline must insert effort guidance in the high-attention opening zone.
- **Tool subset guidance** (§9): The pipeline must insert the stage-appropriate tool subset list into the assembled context.

The `handoff` / `next` tool interface does not change. The output format does not change. The internal assembly logic becomes stage-aware through these requirements.

---

## 8. Effort Budgets in Context Assembly

### 8.1 Problem

Agents default to producing visible output as quickly as possible. Without effort guidance, they allocate minimal effort to specification ("feels like overhead") and maximum effort to implementation. The 3.0 proposal identifies this as a core problem: "They skip steps in the workflow" and "They are too keen to get to implementation."

### 8.2 Design

The binding registry already declares `effort_budget` per stage. The context assembly pipeline reads this and embeds it prominently in the assembled prompt — in the high-attention zone, not buried in reference material.

Format in the assembled context:

```
## Effort Expectations

This is a **specification** task.
Expected effort: 5–15 tool calls.
Expected actions: Read the design document, query knowledge for related decisions,
check related specifications for consistency, draft each required section.

Do NOT skip to implementation. The specification must be complete and internally
coherent before advancing.
```

### 8.3 Effort Budget Reference

Effort budget values are defined in the binding registry (skills doc §3.3, `effort_budget` field per stage binding). The binding registry is the single source of truth for these values. This document's contribution is not the values themselves but how they are positioned and formatted in the assembled prompt (§8.2, §8.4).

For the current values, see the `effort_budget` field in each stage binding in `work/design/skills-system-redesign-v2.md` §3.3.

### 8.4 Position in Assembled Context

Effort guidance appears **above** the task description and **below** the role identity — in the high-attention opening zone. Research shows that information placement at the beginning of context has 30%+ higher retention than the middle (Liu et al., 2024). Effort expectations must be in this zone because they counteract the agent's default behaviour.

---

## 9. Role-Scoped Tool Subsets

### 9.1 Problem

Every agent session currently sees all 22+ MCP tools. Google Research identifies 16+ tools as the threshold where tool-count overhead becomes significant. Past this point, agents waste effort selecting between tools and sometimes pick the wrong one.

### 9.2 Soft Filtering (3.0 Interim Approach)

The design target is hard filtering — dynamically scoping the MCP tool list per session based on the role's `tools` field (skills doc DD-8). For the initial 3.0 release, soft filtering is accepted as a pragmatic stepping stone. The binding registry declares per-stage tool subsets, and the context assembly pipeline includes a "tools you should use" list prominently in the assembled context. All tools remain available for 3.0 — this is guidance, not yet restriction.

Format in the assembled context:

```
## Tools for This Task

Primary tools: entity, doc, knowledge, doc_intel
Do NOT use: decompose, merge, pr, worktree, finish (these are for other stages)

Use `entity(action: "get")` and `status` to check current state.
Use `doc_intel(action: "find")` to locate relevant spec sections.
Use `knowledge(action: "list")` to find related decisions and constraints.
```

### 9.3 Hard Filtering Deferral (Design Target)

The research recommends that specification agents "should have *no access* to implementation tools (terminal, file editing)" — a hard restriction (Research §3.3). Hard filtering (dynamically hiding tools from the MCP session) is the design target (skills doc DD-8) and the research-aligned approach. Every source that compares advisory constraints with enforceable ones finds the latter wins decisively (Research §2.2).

Hard filtering is deferred for the initial 3.0 release due to implementation effort. The evaluation suite (§12) should track tool selection compliance — specifically, how often agents call excluded tools during sequential stages (§12.3, "Tool subset compliance" metric). If soft filtering proves insufficient to keep agents within stage-appropriate tool usage, hard filtering implementation should be prioritised.

The skills doc retains hard-filtering language as the design intent. This section owns the implementation timeline; the skills doc (DD-8) owns the design target.

### 9.4 Tool Subsets by Stage

| Stage | Primary Tools | Excluded Tools |
|---|---|---|
| Designing | entity, doc, doc_intel, knowledge, status | decompose, merge, pr, worktree, finish |
| Specifying | entity, doc, doc_intel, knowledge, status | decompose, merge, pr, worktree, finish |
| Dev-planning | entity, doc, knowledge, decompose, estimate, status | merge, pr, worktree |
| Developing | entity, handoff, next, finish, knowledge, status, branch, worktree | decompose, doc_intel |
| Reviewing | entity, doc, doc_intel, knowledge, finish, status | decompose, merge, worktree, handoff |

---

## 10. Structured Document Templates

### 10.1 Problem

Specifications and plans vary in structure and completeness because agents have no enforced template. A skill says "write a specification" but doesn't define the mandatory sections. The result is inconsistent quality — sometimes thorough, sometimes shallow.

### 10.2 Document Templates

Template section requirements are defined per authoring skill in the skills redesign (§5.1 of `work/design/skills-system-redesign-v2.md`). See the `write-spec`, `write-design`, and `write-dev-plan` skills for the required sections per document type. The template definitions follow the n=5-beats-n=19 principle — 4–5 required sections per template, not 15.

The binding registry carries a `document_template` structure per document-producing stage that serves as the single source of truth for both template content (used by skills) and gate checking (used by the workflow engine). See the skills doc §3.3 and §5.1 for the canonical template schemas.

### 10.3 Template Delivery

Templates are delivered as part of the skill's output format section during context assembly. When a task's stage matches a document-producing stage, the assembly pipeline (skills doc §6.1) loads the corresponding authoring skill, which carries the template inline. This is not a separate assembly-pipeline concern — it is a natural consequence of skill loading.

### 10.4 Automated Structural Checks at Stage Gates

The research recommends programmatic structural checks as a core component of maker-checker automation — not a stretch goal (Research §3.4, §4.5). When a feature transitions past a document-producing stage, the system should verify:

| Check | Gate | Method |
|---|---|---|
| Required sections present | `designing → specifying`, `specifying → dev-planning`, `dev-planning → developing` | Document intelligence structural parsing — check section headings against the template |
| Cross-references valid | `specifying → dev-planning` | Spec must reference the design document |
| Acceptance criteria listed | `dev-planning → developing` | Spec must have at least one acceptance criterion |

These are **programmatic checks** — no LLM evaluation required. They use the existing document intelligence structural parser to verify section presence in the registered document.

**Behaviour on failure:**
```
Cannot transition FEAT-001 to "dev-planning": specification document DOC-042
is missing required sections: "Constraints", "Verification Plan".

To resolve:
1. Read the current spec: doc(action: "content", id: "DOC-042")
2. Add the missing sections and re-register.
```

For 3.0, these checks should be implemented as **warnings** (logged but non-blocking) initially, promoted to **hard gates** once the templates and structural parser are validated in practice. This avoids blocking workflows on parser bugs while still surfacing completeness issues.

### 10.5 LLM-as-Judge Quality Evaluation

Beyond structural checks, the research recommends an LLM-as-judge evaluation step for qualitative document quality (Research §3.4, §4.5). Following Anthropic's approach, a single LLM call scores the document on defined dimensions:

| Dimension | What It Measures | Score |
|---|---|---|
| **Completeness** | Are all required sections substantively covered? (Not just headings with placeholder text) | 0.0–1.0 |
| **Consistency** | Does the document align with its parent document? (Spec aligns with design; dev-plan aligns with spec) | 0.0–1.0 |
| **Testability** | Are acceptance criteria concrete and verifiable? (Not vague aspirations) | 0.0–1.0 |
| **Factual accuracy** | Do claims about the codebase, constraints, or dependencies match reality? | 0.0–1.0 |

A pass/fail grade is derived from the scores (e.g., all dimensions ≥ 0.6 and average ≥ 0.7 = pass). The evaluation prompt, rubric, and threshold are defined once and reused across all document evaluations.

**How this fits in the workflow:**

1. An agent produces a document (specification, design, dev-plan).
2. Before the document is approved (`doc(action: "approve")`), the system runs the LLM-as-judge evaluation.
3. If the document fails, the evaluation returns specific dimension scores and feedback, guiding the agent to improve specific sections.
4. If the document passes, approval proceeds normally.

This is a **medium-term** addition — it requires an LLM call within the MCP server, which is an architectural decision (the server currently makes no LLM calls). For 3.0, this should be designed and prototyped; full integration depends on whether the LLM-call-from-server pattern is accepted. An alternative is to implement it as a review skill that the orchestrator invokes explicitly, keeping the LLM call in the agent layer.

---

## 11. Decomposition Quality Validation

### 11.1 Problem

The research identifies decomposition as *the* critical step in multi-agent workflows — "performance gains correlate almost linearly with the quality of the induced task graph" (Masters et al.). Currently, the `decompose` tool proposes tasks but does not validate their quality.

### 11.2 Validation Checks

After decomposition, the `decompose` tool should validate:

| Check | Description | Severity |
|---|---|---|
| **Description present** | Every task has a non-empty summary | Error — blocks proposal |
| **Dependencies declared** | If tasks reference each other, `depends_on` is populated | Warning |
| **Single-agent sizing** | No task description suggests multiple independent changes | Warning |
| **Testing coverage** | At least one task mentions testing or verification | Warning |
| **No orphan tasks** | Every task is reachable from the dependency graph root | Warning |

Errors block the proposal from being applied. Warnings are included in the proposal output so the orchestrator can address them before applying.

### 11.3 Implementation

The `decompose(action: "review")` action already exists. The validation checks above should be integrated into the review step, so that `propose → review → apply` naturally catches quality issues before tasks are created.

---

## 12. Observability: Action Pattern Logging

### 12.1 Problem

We cannot currently detect *how* agents use the system — whether they follow the proactive orchestrator pattern (decompose, refine, structure) or the reactive communicator pattern (status-check, message, no-op). Without this data, we can't measure whether the 3.0 changes are working.

### 12.2 What to Log

Log MCP tool invocations per session with enough metadata to detect patterns:

| Field | Description |
|---|---|
| `timestamp` | When the call was made |
| `tool` | Tool name |
| `action` | Action parameter (if applicable) |
| `entity_id` | Entity referenced (if applicable) |
| `stage` | Lifecycle stage of the feature at call time (if applicable) |
| `success` | Whether the call succeeded |
| `error_type` | If failed: gate_failure, validation_error, not_found, etc. |

### 12.3 Stage-Level Workflow Metrics

In addition to tool-call-level logging, track higher-level workflow health metrics:

| Metric | What It Measures | Derived From |
|---|---|---|
| **Time per stage** | How long features spend in each lifecycle stage | Entity transition timestamps |
| **Revision cycle count** | How many review-rework cycles features go through | `review_cycle` counter (§4.2) |
| **Gate failure rate** | How often agents hit gate failures, by gate type | Tool call logs (error_type = gate_failure) |
| **Structural check pass rate** | How often documents pass section completeness checks on first attempt | Document evaluation results (§10.4) |
| **Tool subset compliance** | How often agents call excluded tools during a stage | Tool call logs cross-referenced with stage tool subsets (§9) |

These metrics answer different questions than tool-level logs: "Are features spending appropriate time in specification?" and "Are our templates reducing rework cycles?" versus "Which tools did the agent call?"

**Cross-reference with skills doc metrics:** The skills redesign (§10.1 of `work/design/skills-system-redesign-v2.md`) defines additional metrics that complement the workflow-level metrics above: first-attempt convention compliance, review finding specificity, stale-doc-caused errors, context assembly token utilisation, review rubber-stamp rate, sub-agent dispatch per feature, and MAST failure mode incidents. Some of these (convention compliance, finding specificity) are better measured through the skill evaluation process (skills doc §9) than through system logging. Both metric sets should be tracked; neither document's list supersedes the other.

### 12.4 What to Detect

From the combined logs and metrics, we can derive:

- **Specification depth**: Are spec-stage agents reading design documents and querying knowledge, or jumping straight to writing?
- **Gate hit rate**: How often do agents hit gate failures? (High rate = agents don't understand the workflow. Low rate = gates are working as guardrails.)
- **Tool selection accuracy**: Are agents using stage-appropriate tools, or calling implementation tools during specification?
- **Review thoroughness**: Are reviewers reading specs and checking criteria, or rubber-stamping?

### 12.5 Small-Sample Evaluation Suite

Following Anthropic's finding that "a set of about 20 test cases was enough to spot dramatic changes in early development" (Research §4.7), maintain a set of **15–20 representative workflow scenarios** for regression-testing orchestration changes.

Each scenario defines:

- A starting state (feature at a given lifecycle stage, with specific documents and tasks).
- An expected agent interaction pattern (which tools should be called, in what general order).
- Success criteria (feature reaches target state, documents have required sections, no gate overrides needed).

When a 3.0 change is deployed (e.g., new gate enforcement, revised tool descriptions), re-run the evaluation suite and compare results. This is not a CI test — it's a manual or semi-automated evaluation run that uses actual LLM agents against the updated system.

The scenarios should cover:

- Happy path through all lifecycle stages (proposed → done).
- Gate failure and recovery (missing spec, missing tasks).
- Review-rework loop (including iteration cap escalation).
- Multi-feature plan orchestration.
- Edge cases (feature with plan-level spec, feature with no design stage).

### 12.6 Implementation

Tool-level logging does not need a sophisticated analytics system. A structured log file (JSON lines) that can be grepped and aggregated with simple scripts is sufficient. The retrospective system already captures signals at task completion; action pattern logging complements this with *behavioural* data.

The log should be written to `.kbz/logs/` (local, not committed) and rotated by date or size.

Stage-level metrics can be derived from entity timestamps and tool logs — they don't require a separate data store. A periodic `kbz metrics` CLI command that aggregates from existing data would be sufficient.

---

## 13. Filesystem-Output Convention

### 13.1 Problem

When sub-agents complete work, their full output passes through the orchestrator's conversation history. For large outputs (review reports, implementation summaries), this consumes context the orchestrator needs for coordination.

### 13.2 Convention

Sub-agents should write detailed outputs to the filesystem and return lightweight references to the orchestrator:

- **Review sub-agents** write findings to registered documents (via `doc`), not to conversation.
- **Implementation sub-agents** commit code and update task status via `finish`; the orchestrator reads task status, not implementation details.
- **The orchestrator's context** contains *references* (document IDs, task IDs, status summaries) not *contents*.

### 13.3 Enforcement

For 3.0, this is reinforced through context assembly, not through MCP tool changes:

- The `handoff` assembled context for orchestrator tasks includes: "Sub-agents write outputs to documents and task records. Read their status via `entity(action: "get")` and `doc(action: "get")`. Do not retain sub-agent conversation output."
- The `finish` tool's summary field is limited to a reasonable length (500 characters) to encourage conciseness.

---

## 14. Summary of Changes

### Workflow Engine Changes

| Change | Type | Impact | Section |
|---|---|---|---|
| Mandatory stage gates on all transitions (not just advance) | Enforcement upgrade | **High** | §3 |
| Task-completeness gate for `developing → reviewing` | New gate | **High** | §3.3 |
| Review-report gate for `reviewing → done` | New gate | **High** | §3.3 |
| Gate override with reason logging | Safety mechanism | **Medium** | §3.4 |
| Binding registry as gate definition source | Architecture | **Medium** | §3.5 |
| Automated structural checks on documents at stage gates | Gate enhancement | **Medium** | §10.4 |
| Review cycle counter on feature entity | Entity model change | **Medium** | §4.2 |
| Iteration cap with human escalation | Enforcement | **Medium** | §4.3 |
| Focused re-review context for cycle ≥ 2 | Context assembly | **Low** | §4.4 |

### MCP Tool Surface Changes

| Change | Type | Impact | Section |
|---|---|---|---|
| ACI-optimised tool descriptions (with agent-driven testing) | Description-only | **High** | §5 |
| Actionable error messages across all tools | Error handling | **High** | §6 |
| Decomposition quality validation in `decompose` | Tool logic | **Medium** | §11 |

### Context Assembly Changes

| Change | Type | Impact | Section |
|---|---|---|---|
| Lifecycle state validation in `handoff` / `next` (reject wrong-state assembly) | Tool logic | **High** | §7.2 |
| Stage-specific assembly strategy in `handoff` / `next` | Assembly logic | **High** | §7.3 |
| Orchestration pattern signalling (single-agent vs multi-agent) | Context content | **High** | §7.4 |
| Effort budgets in high-attention zone | Context content | **Medium** | §8 |
| Role-scoped tool subset lists | Context content | **Medium** | §9 |
| Structured document templates in assembly | Context content | **Medium** | §10 |
| Filesystem-output convention in orchestrator context | Context content | **Low** | §13 |

### Observability Changes

| Change | Type | Impact | Section |
|---|---|---|---|
| Action pattern logging to `.kbz/logs/` | New capability | **Low** | §12 |
| Stage-level workflow metrics | New capability | **Low** | §12.3 |
| Small-sample evaluation suite (15–20 scenarios) | Evaluation methodology | **Low** | §12.5 |
| LLM-as-judge document quality evaluation | New capability (medium-term) | **Medium** | §10.5 |

---

## 15. Implementation Priority

Recommended implementation order, from highest impact to lowest:

### Phase A: Gate Enforcement (Highest Impact)

1. Make stage gates mandatory on all transitions, not just advance (§3.2).
2. Add `developing → reviewing` task-completeness gate (§3.3).
3. Add `reviewing → done` review-report gate (§3.3).
4. Implement gate override with reason logging (§3.4).
5. Implement actionable error messages for gate failures (§3.6, §6).
6. Add lifecycle state validation to `handoff` / `next` — reject wrong-state assembly (§7.2).

*Rationale: This is the single biggest quality lever. It makes it structurally impossible to skip steps — the core problem identified in the 3.0 proposal.*

### Phase B: Tool Description Audit (High Impact, Low Risk)

7. Audit and rewrite all MCP tool descriptions following ACI principles (§5).
8. Run agent-driven testing on rewritten descriptions (§5.6).
9. Audit error messages across all tool handlers (§6).

*Rationale: Description-only changes with no logic risk. Potentially the highest ROI change based on research.*

### Phase C: Context Assembly Intelligence

10. Implement stage-specific assembly in `handoff` / `next` (§7.3).
11. Add orchestration pattern signalling (§7.4).
12. Add effort budgets to assembled context (§8).
13. Add tool subset lists to assembled context (§9).
14. Add document templates to assembled context (§10).

*Rationale: These all modify the same assembly pipeline and should be done together.*

### Phase D: Review Loop and Decomposition

15. Add review cycle counter to feature entity (§4.2).
16. Implement iteration cap with human escalation (§4.3).
17. Add focused re-review context assembly (§4.4).
18. Add decomposition quality validation to `decompose` (§11).
19. Implement automated structural checks on documents at stage gates (§10.4) — initially as warnings.

### Phase E: Observability and Conventions

20. Implement action pattern logging (§12).
21. Implement stage-level workflow metrics (§12.3).
22. Create small-sample evaluation suite (§12.5).
23. Add filesystem-output convention to orchestrator context (§13).
24. Read gate definitions from binding registry instead of hardcoding (§3.5).
25. Prototype LLM-as-judge document quality evaluation (§10.5).
26. Promote structural checks from warnings to hard gates once validated.

---

## 16. Open Questions

1. **Gate granularity for the `reviewing → done` transition.** Should "no blocking findings" be checked automatically (parsing the review report), or should it be a manual assertion (the orchestrator calls a "review passed" tool)?

2. **Structural check rollout strategy.** Structural checks (§10.4) start as warnings. What criteria determine when they are promoted to hard gates? (e.g., after N features pass through with zero false positives from the parser.)

3. **LLM-as-judge architecture.** The MCP server currently makes no LLM calls. Should document quality evaluation (§10.5) be a server-side capability (requiring an LLM client in the server), or an agent-side capability (a review skill the orchestrator invokes explicitly)? The agent-side approach is simpler but less enforceable.

4. **Log retention policy.** How long should action pattern logs (§12) be retained? Should they be committed or stay local-only?

5. **Binding registry read frequency.** Should the MCP server read the binding registry once at startup, or re-read it on every tool call? The latter enables hot-reloading but adds I/O.

6. **Override audit.** Should gate overrides require a specific permission level, or should any agent with human delegation be able to override? The current design allows any override with a reason — is that sufficient?

7. **Evaluation suite maintenance.** Who maintains the 15–20 workflow scenarios (§12.5)? Should they be committed to the repository and versioned alongside the orchestration logic they test?

---

## 17. Research Traceability

| Recommendation | Research Sources | Orchestration Recommendations Section |
|---|---|---|
| Mandatory stage gates (§3) | MetaGPT, Microsoft, Masters et al., Google Research | §2.1, §3.1 |
| Review-rework iteration cap (§4) | Microsoft (maker-checker), Anthropic (LLM-as-judge) | §2.2 |
| ACI tool descriptions (§5) | SWE-agent, Anthropic (both articles) | §3.2 |
| Agent-driven tool testing (§5.6) | Anthropic (multi-agent: tool-testing agent) | §3.1 |
| Actionable error messages (§6) | SWE-agent (ACI poka-yoke), Anthropic | §3.5 |
| Lifecycle state validation in handoff/next (§7.2) | Anthropic (subagent needs), Masters et al., MetaGPT | §3.2, §4.1 |
| Stage-aware context assembly (§7.3) | Anthropic (subagent needs), Masters et al. | §3.4 |
| Orchestration pattern signalling (§7.4) | Google Research (sequential penalty), Microsoft | §2.3 |
| Effort budgets (§8) | Anthropic (effort scaling), Google (sequential penalty) | §5.1 |
| Role-scoped tool subsets (§9) | Google Research (tool-use bottleneck) | §3.3 |
| Document templates (§10) | MetaGPT, Anthropic, Microsoft | §5.2 |
| Automated structural checks (§10.4) | MetaGPT (verification gates), Microsoft (acceptance criteria) | §3.4, §4.5 |
| LLM-as-judge quality evaluation (§10.5) | Anthropic (LLM-as-judge scoring), Microsoft (maker-checker) | §3.4, §4.5 |
| Decomposition quality (§11) | Masters et al., Anthropic, Google Research | §5.5 |
| Action pattern logging (§12) | Masters et al. (proactive vs reactive), Anthropic | §5.3 |
| Stage-level metrics (§12.3) | Masters et al. (MA-Gym metrics), Anthropic | §4.7 |
| Evaluation suite (§12.5) | Anthropic (small-sample evaluation) | §4.7 |
| Filesystem-output convention (§13) | Anthropic (subagent output), Microsoft (context mgmt) | §3.6 |