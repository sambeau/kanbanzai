# Orchestration Recommendations for Kanbanzai 3.0

| Field | Value |
|-------|-------|
| Date | 2025-07-29 |
| Status | Proposal |
| Author | Design Agent |
| Informed by | `work/research/agent-orchestration-research.md` |
| Related | `work/design/skills-system-redesign.md`, `work/design/Kanbanzai-3.0-proposal.txt` |

---

## Purpose

This document translates the findings from the agent orchestration research into concrete recommendations for the Kanbanzai 3.0 orchestration system — the MCP server, its tools, the workflow lifecycle, and the coordination of SKILLs and roles.

The skills-system-redesign document already covers the detailed redesign of roles, skills, vocabulary routing, anti-patterns, and context assembly. **This document focuses on what that design does not cover:** the orchestration layer that connects those skills and roles to the workflow engine, the MCP tool surface that agents interact with, and the workflow stages themselves.

The recommendations are organised into five sections:

1. **What we're already getting right** — things to preserve.
2. **Workflow changes** — lifecycle stages, gates, and state machine.
3. **MCP server and tool changes** — tool design, descriptions, and enforcement.
4. **SKILL and role integration** — how the orchestration layer connects to the skills redesign.
5. **Other recommendations** — effort budgets, observability, context compaction.

---

## Table of Contents

1. [What We're Already Getting Right](#1-what-were-already-getting-right)
2. [Workflow Recommendations](#2-workflow-recommendations)
3. [MCP Server and Tool Recommendations](#3-mcp-server-and-tool-recommendations)
4. [SKILL and Role Integration Recommendations](#4-skill-and-role-integration-recommendations)
5. [Other Recommendations](#5-other-recommendations)
6. [Summary of Changes](#6-summary-of-changes)
7. [Research Traceability](#7-research-traceability)

---

## 1. What We're Already Getting Right

The research validates several core Kanbanzai design choices. These should be preserved and reinforced, not rearchitected.

### 1.1 Centralised Orchestration with a Single Coordinator

**Research basis:** Google Research found centralised multi-agent systems contained error amplification to 4.4× versus 17.2× for independent agents. The orchestrator acts as a "validation bottleneck" — catching errors before they propagate.

**What Kanbanzai does:** The orchestrator agent coordinates all sub-agent work, dispatching tasks through `handoff`/`next` and reviewing results through `finish`. No agent operates truly independently.

**Verdict:** Keep. This is the correct architecture for software development workflows. The orchestrator-worker pattern should remain the default.

### 1.2 Structured Workflow State as the Source of Truth

**Research basis:** MetaGPT's core finding is that structured intermediate artifacts outperform freeform agent-to-agent conversation. Masters et al. found that the task graph structure is the single strongest predictor of workflow success.

**What Kanbanzai does:** All workflow state is stored as schema-validated YAML in `.kbz/`. Features travel a defined state machine. Tasks have explicit dependencies. Documents are registered with metadata. This is not a chat log — it's a structured model.

**Verdict:** Keep. The YAML-in-Git model is sound and well-validated by the research.

### 1.3 The Lifecycle State Machine

**Research basis:** Masters et al. formalise the distinction between hard constraints (must always hold, violation terminates) and soft constraints (can be violated with penalties). MetaGPT encodes SOPs as enforceable sequences.

**What Kanbanzai does:** Features travel `proposed → designing → specifying → dev-planning → developing → reviewing → done`, with the state machine enforced by the `entity(action: "transition")` tool. Transitions that violate the state machine are rejected.

**Verdict:** Keep the state machine. Strengthen enforcement at the tool level (see §3.1).

### 1.4 Context Assembly via `handoff` / `next`

**Research basis:** Anthropic found that "each subagent needs an objective, an output format, guidance on the tools and sources to use, and clear task boundaries." Without detailed task descriptions, agents "duplicate work, leave gaps, or fail to find necessary information."

**What Kanbanzai does:** The `handoff` and `next` tools assemble targeted context for each task — spec sections, knowledge entries, file paths, and role conventions. This is already the mechanism the research recommends.

**Verdict:** Keep and extend. The assembly pipeline is correct; it needs the skill/role enrichment from the skills redesign and the effort budget metadata from §5.1.

### 1.5 The Knowledge System as Long-Term Memory

**Research basis:** Guo et al. identify three memory types: short-term (conversation), long-term (external stores), and episodic (task-specific). All three are needed.

**What Kanbanzai does:** The knowledge system provides long-term project memory with confidence scoring, staleness tracking, and scoped retrieval. Knowledge entries persist across agent sessions and are surfaced during context assembly.

**Verdict:** Keep. This is a genuine differentiator. The research validates that external knowledge stores are essential for multi-session agent workflows.

### 1.6 Document Intelligence for Structured Search

**Research basis:** AutoCodeRover found that structured program representations (AST-level) outperform raw file text for context gathering, at dramatically lower token cost ($0.43 per issue versus much higher for brute-force approaches).

**What Kanbanzai does:** Document intelligence provides structural parsing, concept extraction, and graph-based search over the document store. Agents query for specific concepts and entities rather than reading entire documents.

**Verdict:** Keep. Structured search over documents is the equivalent of AST-level code search — it gives agents focused context without forcing them to read everything.

### 1.7 Maker-Checker Review Architecture

**Research basis:** Microsoft identifies maker-checker loops as a key pattern: "one agent creates, another evaluates against defined criteria." Anthropic separates generation from evaluation in their multi-agent research system.

**What Kanbanzai does:** The review stage uses separate agents (orchestrator + specialist reviewers) that evaluate implementation against the specification. The reviewing agent is distinct from the implementing agent.

**Verdict:** Keep. The skills redesign enriches this with specialist reviewer roles (conformance, quality, security, testing). The orchestration layer should support the parallel dispatch pattern described in the binding registry (§4.2).

---

## 2. Workflow Recommendations

### 2.1 Add Enforceable Stage Gates Between Lifecycle Stages

**Research basis:** MetaGPT (SOPs with intermediate verification), Microsoft (programmatic gates), Masters et al. (hard constraints ℋ), Google Research (validation bottleneck).

**Problem:** The lifecycle state machine enforces *transition ordering* — you can't jump from `proposed` to `developing`. But it doesn't enforce *prerequisites* — an agent can transition from `specifying` to `dev-planning` without an approved specification document actually existing. The gates are ordering constraints, not quality gates.

**Recommendation:** Add enforceable prerequisites to specific lifecycle transitions. These should be hard constraints (ℋ) — the system rejects the transition if they aren't met.

| Transition | Gate Prerequisites |
|---|---|
| `designing → specifying` | Design document registered and approved |
| `specifying → dev-planning` | Specification document registered and approved |
| `dev-planning → developing` | Dev-plan document registered and approved; at least one task exists |
| `developing → reviewing` | All tasks in terminal state (done or not-planned) |
| `reviewing → done` | Review report registered; no blocking findings open |

**What this changes:** The `entity(action: "transition")` tool (and the `advance` flag) must check document and task prerequisites before allowing the transition. This converts "agents skip steps" from a quality problem into an impossibility — the system literally prevents it.

**What stays the same:** The state machine itself doesn't change. The stages don't change. The transition order doesn't change. We're adding *enforcement of prerequisites that are already implicit*.

**Implementation note:** The `doc` tool's entity hook mechanism already cascades document approvals into feature transitions (e.g., approving a spec auto-advances the feature). The gate system is the inverse: blocking transitions when documents are missing. Both should use the same underlying prerequisite model.

### 2.2 Formalise the "Reviewing" Stage as Multi-Phase

**Research basis:** Microsoft (maker-checker with iteration cap), Anthropic (LLM-as-judge scoring), Masters et al. (constraint adherence as distinct from goal completion).

**Problem:** The `reviewing` stage is currently a single state. In practice, review may find issues that require rework, which loops back to development. The `needs-rework` state exists but the flow between reviewing → needs-rework → developing → reviewing is informal.

**Recommendation:** Keep the existing states but add clarity to the review loop:

1. **Review dispatch.** The orchestrator enters `reviewing` and dispatches specialist reviewers in parallel.
2. **Review verdict.** The orchestrator collates findings and produces a verdict: pass, pass-with-notes, or fail.
3. **Rework loop.** On fail, the feature transitions to `needs-rework`. Specific rework tasks are created. When rework tasks complete, the feature returns to `reviewing` for a focused re-review (not a full review).
4. **Iteration cap.** Maximum 3 review-rework cycles before escalating to human decision. This prevents infinite refinement loops (a known anti-pattern identified by Microsoft).

**What this changes:** The `needs-rework` state gets clearer semantics. The review skill (from the skills redesign) should include the iteration cap as a hard constraint. The orchestrator should track review cycle count on the feature entity.

**What stays the same:** The lifecycle states themselves don't need to change. `reviewing` and `needs-rework` already exist. This is about formalising the protocol, not changing the state machine.

### 2.3 Match Orchestration Pattern to Workflow Stage

**Research basis:** Google Research (sequential penalty: -39-70% on sequential tasks; +81% on parallelisable tasks), Microsoft (pattern selection), Anthropic ("start with the right level of complexity").

**Problem:** Currently, the same orchestration approach (orchestrator dispatches sub-agents) is used for every stage. But the research is unambiguous: specification and design are *sequential reasoning tasks* where multi-agent coordination degrades performance. Implementation is a *parallelisable task* where multi-agent coordination helps enormously.

**Recommendation:** The binding registry (from the skills redesign) already declares per-stage agent counts. The orchestration layer should enforce these patterns:

| Stage | Pattern | Agent Count | Rationale |
|---|---|---|---|
| Designing | **Single agent, no delegation** | 1 | Sequential reasoning. Fragmentation degrades quality. |
| Specifying | **Single agent, no delegation** | 1 | Sequential reasoning. The spec must be internally coherent. |
| Dev-planning | **Single agent, decomposition focus** | 1 | Sequential reasoning with structured output (task graph). |
| Developing | **Orchestrator + parallel workers** | 1 + N | Parallelisable. Tasks are independent by construction. |
| Reviewing | **Orchestrator + parallel specialists** | 1 + up to 4 | Parallelisable. Each reviewer dimension is independent. |

**What this changes:** The `handoff` tool should include the orchestration pattern in the assembled context. A specification task should explicitly say "single-agent task — do not delegate sub-tasks." An implementation orchestration task should say "dispatch independent tasks to sub-agents in parallel."

**What stays the same:** The orchestrator still coordinates all work. The difference is that for sequential stages, the orchestrator *is* the worker — it doesn't delegate.

### 2.4 No New Lifecycle Stages

**Research basis:** Anthropic ("the most successful implementations weren't using complex frameworks"), Masters et al. (the fundamental trade-off between goal completion, constraint adherence, and runtime).

**Recommendation:** Do not add new lifecycle stages. The current set (`proposed → designing → specifying → dev-planning → developing → reviewing → done`, plus `needs-rework`, `blocked`, `not-planned`, `cancelled`) is comprehensive. Adding stages increases the state machine complexity without proportional benefit.

The improvements come from better *enforcement within* the existing stages (§2.1), clearer *protocols within* the existing stages (§2.2), and matching *patterns to* the existing stages (§2.3) — not from adding more stages.

---

## 3. MCP Server and Tool Recommendations

### 3.1 Enforce Lifecycle Prerequisites in Transition Tools

**Research basis:** MetaGPT (SOPs), Microsoft (gates), Masters et al. (hard constraints ℋ).

**What changes:** The `entity(action: "transition")` handler must check prerequisites before allowing lifecycle transitions. This is the server-side implementation of the stage gates described in §2.1.

**Behaviour:**

```
entity(action: "transition", id: "FEAT-001", status: "dev-planning")
→ ERROR: Cannot transition to dev-planning: specification document
  not approved. Register and approve a specification document for
  FEAT-001 before advancing.
```

The error message must be *actionable* — it should tell the agent exactly what to do to unblock the transition, not just that it failed.

The `advance` flag should check each gate in sequence and stop at the first unmet prerequisite, reporting what's needed.

**What stays the same:** The transition mechanism itself. We're adding a prerequisite check layer, not replacing the state machine.

### 3.2 Redesign Tool Descriptions as Agent-Computer Interfaces (ACIs)

**Research basis:** SWE-agent (ACI design), Anthropic (tool description optimization yielded 40% faster task completion), Google Research (tool-use bottleneck at 16+ tools).

**Problem:** MCP tool descriptions are currently written as API documentation — they explain what each parameter does. But research shows that tool descriptions should be designed as *interfaces for agents*, answering: When should I use this? What should I use instead? What's the workflow context?

**Recommendation:** Audit and rewrite every MCP tool description following ACI principles:

1. **Lead with the "when to use" signal.** The first sentence should be when/why, not what.
2. **Include negative guidance.** "Use this INSTEAD OF reading .kbz/ files directly" or "Do NOT use this for X — use Y instead."
3. **Add workflow position.** "Call this AFTER completing specification tasks" or "Call this BEFORE spawning implementation sub-agents."
4. **Make parameter relationships explicit.** "When action is 'create', type is required. When action is 'get', id is required."
5. **Keep descriptions under 200 tokens.** Agents process tool descriptions on every call; brevity matters.

**Example — current `entity` description (conceptual):**

> "Create, read, update, and transition entities (plans, features, tasks, bugs, epics, decisions). Use action: get or action: list to query entities..."

**Example — ACI-optimised description:**

> "The primary tool for managing workflow state. Use INSTEAD OF reading .kbz/state/ files. Actions: create (new entity), get (single entity by ID), list (filtered query), update (modify fields), transition (advance lifecycle — checks prerequisites automatically). Start here when you need to know the current state of any feature, task, or plan."

**Implementation:** This is a description-only change — no tool logic changes. But based on the research, this may be one of the highest-impact changes we can make. Anthropic's team "spent more time optimizing our tools than the overall prompt."

### 3.3 Role-Scoped Tool Subsets

**Research basis:** Google Research (tool-use bottleneck grows with tool count), Anthropic ("fewer, more relevant tools reduce cognitive load"), SWE-agent (purpose-built interfaces outperform raw access).

**Problem:** Every agent session currently sees all MCP tools. At 22+ tools, this is past the point where tool-count overhead becomes significant (Google's research identifies 16+ tools as the threshold).

**Recommendation:** The skills redesign already defines per-role tool subsets in the binding registry. The MCP server should support this by allowing context assembly to *declare* which tools are relevant for a given task.

There are two implementation approaches:

**Option A: Soft filtering (recommended for 3.0).** The `handoff` / `next` output includes an explicit "tools you should use" list in the assembled context. All tools remain available, but the context directs the agent to the relevant subset. This is lower-risk and doesn't require MCP protocol changes.

**Option B: Hard filtering (future consideration).** The MCP server dynamically exposes only the tools relevant to the current role. This requires session-level tool registration, which is a more complex change.

For 3.0, Option A is sufficient — the binding registry declares the tool subset, and context assembly includes it prominently in the assembled prompt. The research suggests that clear guidance about which tools to use is nearly as effective as removing the others, with lower implementation risk.

### 3.4 Lifecycle-Aware Context Assembly in `handoff` / `next`

**Research basis:** Anthropic (subagent needs objective, output format, tool guidance, task boundaries), Masters et al. (proactive orchestrators decompose and structure; reactive communicators message and check status).

**Problem:** The `handoff` and `next` tools already assemble context, but they don't vary their assembly strategy based on the workflow stage. A specification task and an implementation task get the same structural treatment.

**Recommendation:** Context assembly should be lifecycle-aware:

| Stage | Context Assembly Strategy |
|---|---|
| Designing | Include design doc template, related decisions, parent plan context. Exclude implementation tools. |
| Specifying | Include design document (full), spec template, acceptance criteria format. Exclude implementation tools. |
| Dev-planning | Include approved spec (full), task decomposition guidance, dependency format. Exclude implementation tools. |
| Developing | Include approved spec (relevant sections only), task description, file paths, test expectations. Exclude planning/review tools. |
| Reviewing | Include spec (relevant sections), implementation diff/summary, review rubric, verdict format. Exclude implementation tools. |

**What this changes:** The assembly pipeline gains a stage-dispatch step that varies what context is included and how it's structured. The binding registry provides the data; the assembly pipeline applies it.

**What stays the same:** The `handoff` / `next` tool interface doesn't change. The output format doesn't change. The internal assembly logic becomes stage-aware.

### 3.5 Actionable Error Messages from All Tools

**Research basis:** SWE-agent (ACI poka-yoke), Anthropic ("make wrong usage hard").

**Problem:** When a tool call fails, the error message should tell the agent *what to do next*, not just what went wrong. A message like "invalid status transition" sends the agent into a recovery loop. A message like "Cannot transition FEAT-001 to developing: specification not approved. Call `doc(action: 'list', owner: 'FEAT-001', pending: true)` to see pending documents" gives the agent a clear recovery path.

**Recommendation:** Audit all error paths in MCP tools for actionability. Every error response should include:

1. **What failed** (the fact).
2. **Why it failed** (the prerequisite or constraint).
3. **What to do instead** (the recovery action, ideally as a tool call hint).

This is a poka-yoke principle: if we can't prevent the wrong action, we can make recovery from it as fast as possible.

### 3.6 Subagent Output to Filesystem, Not Conversation

**Research basis:** Anthropic multi-agent system ("direct subagent outputs can bypass the main coordinator... improving both fidelity and performance"), Microsoft (context management to prevent exceeding model limits).

**Problem:** When sub-agents complete work, their full output passes through the orchestrator's conversation history. For large outputs (review reports, implementation summaries), this consumes context that the orchestrator needs for coordination.

**Recommendation:** Sub-agents should write detailed outputs to the filesystem (documents, review reports, knowledge entries) and return lightweight references to the orchestrator. The `finish` tool already supports this pattern — sub-agents commit knowledge entries and files, and the orchestrator sees the summary.

For 3.0, reinforce this pattern in the orchestration skills:

- Review sub-agents write their findings to registered documents, not to conversation.
- Implementation sub-agents commit code and update task status; the orchestrator reads the task status, not the implementation details.
- The orchestrator's context contains *references* (document IDs, task IDs) not *contents*.

---

## 4. SKILL and Role Integration Recommendations

The skills-system-redesign covers the detailed design of roles, skills, vocabulary, anti-patterns, and the binding registry. This section covers how the *orchestration layer* connects to that design.

### 4.1 The Binding Registry Drives Orchestration Decisions

**Research basis:** Masters et al. (the task graph structure is the critical path), Google Research (architecture must match task structure).

The binding registry maps each workflow stage to roles, skills, tool subsets, and orchestration topology. The orchestration layer should treat the binding registry as its *decision table* — it doesn't decide how to orchestrate a stage; it looks it up.

**What this means for the MCP server:**

- `handoff` reads the binding registry to determine what to assemble.
- `next` reads the binding registry to determine what the orchestrator should do when claiming work (single-agent task? dispatch sub-agents? wait for human gate?).
- The `decompose` tool reads the binding registry to determine task sizing constraints for the target stage.

**What this means for SKILLs:**

- The orchestrator SKILL should reference the binding registry explicitly: "Check the binding for this stage before dispatching."
- Individual stage SKILLs should not contain orchestration logic — they should contain *task execution* logic. Orchestration lives in the binding registry and the orchestrator SKILL.

### 4.2 Orchestrator Role Gets Explicit Workflow Vocabulary

**Research basis:** Ranjan et al. (vocabulary is the primary routing mechanism), PRISM framework (domain-specific terms activate correct knowledge clusters).

The `orchestrator` role defined in the skills redesign should carry vocabulary specific to workflow coordination, not just generic project vocabulary. This vocabulary routes the model toward workflow management expertise:

Suggested vocabulary terms for the orchestrator role:

- **Workflow terms:** lifecycle gate, stage prerequisite, transition check, document approval cascade, stage binding, hard constraint, soft constraint
- **Decomposition terms:** vertical slice, task dependency graph, critical path, parallelisable vs sequential, decomposition quality
- **Delegation terms:** context assembly, effort budget, tool subset, sub-agent dispatch, lightweight reference, iteration cap
- **Review terms:** maker-checker, review verdict, rework loop, review cycle count, escalation threshold

This vocabulary should be distinct from the vocabulary of authoring roles (architect, spec-author, implementer) — the orchestrator thinks about *workflow structure*, not about *content*.

### 4.3 Constraint Levels Should Align with Stage Gates

**Research basis:** Masters et al. (hard constraints ℋ vs soft constraints 𝒮), the skills redesign DP-9 (match constraint level to task risk).

The skills redesign defines three constraint levels: low freedom (exact sequences), medium freedom (templates with flexibility), high freedom (general guidance). These should align with the stage gate enforcement:

| Constraint Level | Stage Gate Enforcement | Example |
|---|---|---|
| **Low freedom** | Hard gates — system rejects violations | Lifecycle transitions, document registration, stage prerequisites |
| **Medium freedom** | Soft gates — system warns but allows | Document section completeness, cross-reference coverage |
| **High freedom** | No gates — trust the agent | Design choices, implementation approach, research direction |

**What this means:** The MCP server enforces low-freedom constraints. SKILLs guide medium-freedom constraints (through templates and procedures). High-freedom work relies on vocabulary routing and anti-patterns, not enforcement.

### 4.4 Review Sub-Agent Topology Is Declared, Not Decided

**Research basis:** Google Research (the optimal architecture for 87% of tasks can be predicted from task properties), DeepMind (agent count saturation at 4-7 agents).

The skills redesign's binding registry declares that reviewing uses parallel sub-agents with a max of 4. The orchestration layer should *enforce* this:

- The review orchestration SKILL should include the exact dispatch pattern: "Spawn up to 4 specialist reviewers in parallel. Each reviewer gets one dimension."
- The orchestrator should not decide whether to parallelise — the binding tells it to.
- If fewer than 4 dimensions need review (e.g., a documentation-only change doesn't need security review), the orchestrator should dispatch fewer, not more.

This prevents the common anti-pattern of over-orchestration — spawning agents for work that doesn't need them.

---

## 5. Other Recommendations

### 5.1 Effort Budgets in Context Assembly

**Research basis:** Anthropic multi-agent system ("agents struggle to judge appropriate effort for different tasks, so we embedded scaling rules in the prompts").

**Problem:** Agents default to producing visible output as quickly as possible. Specification "feels like overhead" compared to implementation. Without effort guidance, agents allocate minimal effort to specification and maximum effort to implementation.

**Recommendation:** The `handoff` / `next` context assembly should include explicit effort expectations per stage:

| Stage | Expected Effort | Guidance |
|---|---|---|
| Designing | 5–15 tool calls | Read related designs, query decisions, draft structured document. Do not skip to specification. |
| Specifying | 5–15 tool calls | Read the design document, query knowledge, check related decisions, draft each required section. Do not skip to implementation. |
| Dev-planning | 5–10 tool calls | Read the spec, decompose into tasks with dependencies, estimate effort, produce the plan document. |
| Developing (per task) | 10–50 tool calls | Read spec section, implement, test, iterate. |
| Reviewing (per dimension) | 5–10 tool calls | Read artifact, check against criteria, produce structured findings. |

These are embedded in the assembled context, not in the SKILL (which may be read with lower attention). They appear near the top of the assembled prompt, in the high-attention zone.

### 5.2 Structured Output Templates for Document Stages

**Research basis:** MetaGPT (structured intermediate artifacts), Anthropic (output format requirements), Microsoft (acceptance criteria for checker agents).

**Problem:** Specifications and plans vary in structure and completeness because agents have no template. The SKILL says "write a specification" but doesn't define the sections.

**Recommendation:** Define mandatory document templates for each document type. These templates should be:

1. **Included in context assembly** — when a specification task is assembled, the template is part of the assembled context.
2. **Checked at the stage gate** — when the feature attempts to transition past the stage, the system checks that the document has the required sections.
3. **Lean** — templates should have 5–8 required sections, not 15. The n=5-beats-n=19 principle applies to template sections too.

Example specification template (5 required sections):

- **Problem Statement** — What problem does this solve? (references design document)
- **Requirements** — Functional and non-functional, each with a unique ID
- **Constraints** — What must not change, what limits apply
- **Acceptance Criteria** — Testable/verifiable conditions for "done"
- **Verification Plan** — How will each criterion be verified?

The template itself acts as a forcing function — the agent must engage with each section, which distributes effort and prevents rush-to-implementation.

### 5.3 Observability: Action Pattern Logging

**Research basis:** Masters et al. (proactive orchestrator vs reactive communicator patterns), Anthropic ("start evaluating immediately with small samples").

**Problem:** We can't currently detect *how* agents are using the system — whether they're following the proactive orchestrator pattern (decompose, refine, structure) or the reactive communicator pattern (status-check, message, no-op).

**Recommendation:** Log MCP tool invocations per session with enough metadata to detect patterns:

- **Tool call sequence** per agent session (which tools, in what order).
- **Stage context** for each call (what lifecycle stage was the feature in?).
- **Call count by category** (planning tools vs implementation tools vs status tools).

This doesn't need to be a sophisticated analytics system — a structured log that can be grepped is sufficient. The value is in being able to answer: "Are our specification agents actually reading design documents and querying knowledge, or are they skipping straight to writing?"

The retrospective system already captures signals at task completion. Action pattern logging complements this with *behavioural* data — what agents *did*, not just what they *thought*.

### 5.4 Context Compaction Between Agent Sessions

**Research basis:** Microsoft (context management), Anthropic (subagent output to filesystem), Google Research (tool-use bottleneck).

**Problem:** When an orchestrator dispatches multiple sub-agents sequentially, its context grows with each interaction. By the time it's reviewing the fourth sub-agent's output, its context window may be approaching capacity, degrading coordination quality.

**Recommendation:** The orchestrator SKILL should include explicit compaction guidance:

1. **After each sub-agent completes,** summarise the outcome in 2–3 sentences and note the document/task ID for the full output. Do not retain the full sub-agent output in conversation.
2. **Before dispatching the next sub-agent,** verify that context utilisation is below 60%. If it isn't, write a progress summary to a document and start a fresh orchestration session.
3. **For multi-feature plans,** structure orchestration as a sequence of single-feature contexts, not one massive context that covers all features.

This is primarily a SKILL-level concern (the orchestrator SKILL includes this guidance), but the MCP server can support it by:

- Including context utilisation estimates in `status` output.
- Providing a lightweight "progress checkpoint" mechanism that the orchestrator can write to and read from across sessions.

### 5.5 Decomposition Quality Investment

**Research basis:** Masters et al. ("performance gains correlate almost linearly with the quality of the induced task graph"), Anthropic (detailed task descriptions prevent duplication and gaps), Google Research (decomposability is the strongest predictive feature).

**Problem:** Decomposition is currently one step in the planning process. The research says it's *the* critical step — more important than the implementation itself.

**Recommendation:** Invest disproportionately in decomposition quality:

1. **The `decompose` tool should validate output quality.** After decomposition, check: Do tasks have clear descriptions? Are dependencies declared? Are tasks sized for single-agent completion? Are there any obvious gaps (e.g., no testing tasks)?
2. **The dev-planning SKILL should emphasise decomposition.** The skill procedure should spend more steps on decomposition validation than on any other activity.
3. **The review stage should check decomposition retroactively.** If implementation reveals missing tasks or incorrect dependencies, this should be captured as a signal for improving future decomposition.

This aligns with the research finding that "structure learning, not raw language generation, is the critical path."

---

## 6. Summary of Changes

### Workflow Changes

| Change | Type | Impact | Section |
|---|---|---|---|
| Enforceable stage gate prerequisites | New enforcement on existing stages | **High** | §2.1 |
| Formalise review-rework loop with iteration cap | Protocol clarification | **Medium** | §2.2 |
| Match orchestration pattern to stage type | New stage-aware dispatch | **Medium** | §2.3 |
| No new lifecycle stages | Non-change (preserve) | — | §2.4 |

### MCP Server and Tool Changes

| Change | Type | Impact | Section |
|---|---|---|---|
| Prerequisite checks in `entity(action: "transition")` | Tool logic change | **High** | §3.1 |
| Rewrite all tool descriptions as ACIs | Description-only change | **High** | §3.2 |
| Role-scoped tool subset guidance in context | Context assembly change | **Medium** | §3.3 |
| Stage-aware context assembly in `handoff` / `next` | Context assembly change | **Medium** | §3.4 |
| Actionable error messages across all tools | Error handling improvement | **Medium** | §3.5 |
| Reinforce filesystem-output pattern for sub-agents | SKILL + convention change | **Low** | §3.6 |

### SKILL and Role Integration Changes

| Change | Type | Impact | Section |
|---|---|---|---|
| Binding registry drives orchestration decisions | Architecture integration | **High** | §4.1 |
| Orchestrator role vocabulary for workflow coordination | Role enrichment | **Medium** | §4.2 |
| Constraint levels aligned with gate enforcement | Design consistency | **Medium** | §4.3 |
| Review topology declared in binding, not decided by agent | Enforcement | **Low** | §4.4 |

### Other Changes

| Change | Type | Impact | Section |
|---|---|---|---|
| Effort budgets in context assembly | Context enrichment | **Medium** | §5.1 |
| Structured output templates for document stages | Template + gate check | **Medium** | §5.2 |
| Action pattern logging | Observability | **Low** | §5.3 |
| Context compaction guidance in orchestrator SKILL | SKILL content | **Low** | §5.4 |
| Decomposition quality investment | SKILL + tool enrichment | **Medium** | §5.5 |

---

## 7. Research Traceability

Every recommendation traces to specific findings in `work/research/agent-orchestration-research.md`. This table maps recommendations to their research basis for auditability.

| Recommendation | Research Sources | Research Section |
|---|---|---|
| Enforceable stage gates (§2.1) | MetaGPT, Microsoft, Masters et al., Google Research | §2.2, §3.2 |
| Review-rework iteration cap (§2.2) | Microsoft (maker-checker), Anthropic (LLM-as-judge) | §4.5 |
| Stage-matched orchestration (§2.3) | Google Research (sequential penalty), Microsoft | §2.4 |
| ACI tool descriptions (§3.2) | SWE-agent, Anthropic (both articles) | §2.1, §4.2 |
| Role-scoped tool subsets (§3.3) | Google Research (tool-use bottleneck) | §2.4, §3.1 |
| Stage-aware context assembly (§3.4) | Anthropic (subagent needs), Masters et al. | §2.5 |
| Actionable error messages (§3.5) | SWE-agent (ACI poka-yoke), Anthropic | §2.1 |
| Effort budgets (§5.1) | Anthropic (effort scaling), Google (sequential penalty) | §3.3, §4.4 |
| Document templates (§5.2) | MetaGPT, Anthropic, Microsoft | §3.4, §4.3 |
| Action pattern logging (§5.3) | Masters et al. (proactive vs reactive), Anthropic | §2.5, §4.7 |
| Decomposition investment (§5.5) | Masters et al., Anthropic, Google Research | §2.3 |