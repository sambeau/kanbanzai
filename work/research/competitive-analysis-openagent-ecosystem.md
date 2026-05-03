# Competitive Analysis: OpenCode Plugin Ecosystem vs. Kanbanzai

**Type:** Research  
**Date:** 2026-07-18 (cross-validated with independent evaluation)  
**Status:** Draft  
**Author:** AI Research Agent  

## 1. Executive Summary

This report evaluates four OpenCode ecosystem projects — oh-my-openagent (OmO), micode, opencode-background-agents, and Portal — against Kanbanzai's architecture. The analysis identifies feature gaps, architectural differences, and opportunities for cross-pollination.

**Key finding:** OmO and Kanbanzai are solving overlapping problems from fundamentally different architectural starting points. This report was cross-validated against an independent evaluation — the two reports converge strongly on priorities: hash-anchored edits as the top feature to adopt, Kanbanzai's entity hierarchy and document intelligence as structural advantages, and the need for automated gate validation to reduce mechanical human-gate overhead. OmO optimizes for *autonomous throughput* (fire-and-forget, the agent figures everything out) while Kanbanzai optimizes for *structured governance* (stage gates, document prerequisites, human-owns-intent). The most valuable ideas come from OmO's runtime execution model, not its planning layer.

## 2. Project Summaries

### 2.1 oh-my-openagent (OmO)

**Scale:** 55.5k stars, 4.5k forks, 170 releases. Largest project in the ecosystem by 50x.  
**Architecture:** OpenCode plugin (TypeScript). Multi-model agent orchestration harness.  
**Core thesis:** Models have different temperaments — leverage them all. Route tasks by semantic category, not model name. Human intervention is a failure signal.

Key components:
- **Discipline agents:** Sisyphus (orchestrator), Hephaestus (autonomous worker), Prometheus (planner), Atlas (execution conductor), Oracle (architecture), Metis (gap analyzer), Momus (plan reviewer)
- **Category system:** Tasks dispatched by semantic intent (`ultrabrain`, `visual-engineering`, `deep`, `artistry`, `quick`) → auto-routed to optimal model
- **Runtime features:** Hash-anchored edits, LSP + AST-Grep tools, background agents, skill-embedded MCPs, Todo Enforcer, Ralph Loop (self-referential work loop)
- **Planning:** Prometheus interview mode + Metis gap analysis + Momus plan validation
- **Session continuity:** Boulder.json tracking, `/start-work` resume, wisdom accumulation (learnings passed forward across sub-agents)

### 2.2 micode

**Scale:** 359 stars, 23 forks.  
**Architecture:** OpenCode plugin (TypeScript). Opinionated Brainstorm → Plan → Implement workflow.  
**Core thesis:** Structure drives quality. Brainstorm first, research before implementing, TDD-enforced execution, session continuity via ledgers.

Key components:
- Structured three-phase workflow with distinct agent roles per phase
- Context compaction at 50% threshold (auto-summarizes to reduce context)
- Git worktree isolation for implementation
- Implementer→Reviewer cycle with parallel execution
- Ledger system for cross-session continuity
- Mindmodel system for project-specific pattern enforcement

### 2.3 opencode-background-agents

**Scale:** 226 stars, 16 forks.  
**Architecture:** OpenCode plugin (TypeScript). Single-purpose: async background delegation with persistence.  
**Core thesis:** Research shouldn't block coding. Results should survive context compaction.

Key components:
- Fire-and-forget delegation with disk persistence
- Each delegation auto-tagged with title + summary for discoverability
- Results survive context compaction, session restarts, crashes
- Read-only sub-agents only (write-capable agents use native `task` tool)
- Part of broader KDCO/OCX ecosystem

### 2.4 Portal

**Scale:** 621 stars, 53 forks.  
**Architecture:** Standalone web UI (React + Nitro) + CLI. Not a plugin — wraps OpenCode itself.  
**Core thesis:** Mobile-first, browser-based access to OpenCode instances via VPN.

Key components:
- Web-based chat interface for OpenCode sessions
- Session management (create, view, delete)
- File mention support (`@filename`)
- Model selection UI
- Git integration, in-browser terminal, isolated workspaces

## 3. What They Do Better

### 3.1 OmO: Runtime Agent Autonomy

OmO's execution layer is more autonomous than Kanbanzai's. Key advantages:

| Feature | OmO | Kanbanzai |
|---------|-----|-----------|
| **Self-directed continuation** | Ralph Loop / Todo Enforcer: agent keeps working until done without re-prompting | Orchestrator must re-dispatch after each task completion; each dispatch is explicit |
| **Fire-and-forget** | `ulw` / `ultrawork`: describe goal, agent figures out strategy, implementation, verification autonomously | Requires approved design → approved spec → approved dev-plan → explicit task dispatch chain |
| **Runtime error recovery** | Session Recovery: automatic recovery from context window limits, API failures, session errors | Checkpoint-based human escalation after max_review_cycles |
| **Wisdom accumulation** | Learnings from task 1 automatically passed to task 5. Conventions, successes, failures, gotchas all forwarded | Knowledge entries contributed at `finish` time, but orchestrator must explicitly query and forward — no automatic accumulation |
| **Plan validation loop** | Metis (gap analysis) + Momus (ruthless review) validate plans before execution. Momus rejects until ≥80% tasks have clear reference sources | Dev-plan review is human-gated but has no automated plan-quality validator |

### 3.2 OmO: Hash-Anchored Edit Tool

This is the single most impactful technical innovation across all four projects:

- Every line read by the agent is tagged with a content hash (`11#VK| function hello()`)
- Agent edits by referencing hash tags, not by reproducing text
- If file changed since last read, hash mismatch → edit rejected before corruption
- Claimed improvement: 6.7% → 68.3% success rate on Grok Code Fast 1

Kanbanzai has no equivalent. Our `edit_file` tool uses fuzzy matching and relies on the model reproducing content it already saw — exactly the "harness problem" OmO solves.

### 3.3 OmO: LSP + AST-Grep Integration

OmO provides IDE-precision tools to agents:
- `lsp_rename`, `lsp_goto_definition`, `lsp_find_references`, `lsp_diagnostics`
- AST-Grep for pattern-aware code search/rewriting across 25 languages

Kanbanzai delegates to sub-agents that use `edit_file`, `grep`, and `search_graph` — no LSP integration, no AST-level refactoring tools.

### 3.4 OmO: Model Category System

Semantic routing (`ultrabrain`, `visual-engineering`, `deep`) rather than explicit model selection. The agent describes *what kind of work* and the system picks the right model. This is a genuine architectural insight — it decouples task semantics from model availability.

Kanbanzai's role system is analogous at the *identity level* (architect, implementer-go, reviewer-security) but doesn't route to different AI models — it shapes the prompt for a single model.

### 3.5 micode: Mindmodel Pattern Enforcement

micode's mindmodel system enforces project-specific patterns beyond what context injection provides. What makes it interesting isn't just the convention documentation — it's that micode ships dedicated agents that *verify* conformance (anti-pattern-detector, convention-extractor, pattern-discoverer, constraint-reviewer). The system is normative (enforcing style) rather than merely descriptive (providing context). This sits in a gap between Kanbanzai's knowledge graph (general architectural knowledge) and its role profiles (behavioural rules) — a lower-level, code-pattern-oriented knowledge layer that actively checks compliance.

### 3.6 micode: Batch-First Parallelism (10-20 Concurrent Agents)

micode's executor fires 10-20 implementer agents simultaneously in a single message, waits for all to complete, then fires all reviewers simultaneously. This is designed for "micro-tasks" of 2-5 minutes each — a fundamentally different throughput model from Kanbanzai's 4-agent cap. It's not better or worse, just optimized for a different granularity. Worth noting for cases where features decompose into many small, independent implementation units.

### 3.7 micode: Auto-Compaction at Threshold

Context compaction at 50% utilization with automatic summarization. Kanbanzai has compaction guidance in the orchestrate-development skill (at 60%, write progress document, start fresh session) but it's procedural, not automated — requires the orchestrator to notice and act.

### 3.8 opencode-background-agents: Async Fire-and-Forget Dispatch

This plugin implements a sophisticated async dispatch lifecycle: `registered → running → terminal`. Key details from the source: results are persisted to disk as markdown with auto-generated title/description metadata for discoverability. Terminal-state protection prevents late progress events from regressing a completed task. `delegation_read(id)` blocks until completion or timeout, returning deterministic terminal info. Crucially, context compaction carries forward running and unread completed delegation context — so the orchestrator doesn't lose track of background work when its context window fills up.

Kanbanzai's `spawn_agent` is synchronous — the orchestrator blocks until the sub-agent completes. There's no fire-and-forget dispatch, no result persistence independent of session context, and no mechanism to continue productive work while sub-agents run. The 15-minute timeout in the plugin is also worth noting — async dispatch works best for bounded-duration tasks, not unbounded ones.

### 3.9 Portal: Multi-Interface Access

Portal's web UI + CLI combination demonstrates that Kanbanzai's chat-only interface could be extended. A mobile-accessible web UI for monitoring progress, approving documents, and intervening at checkpoints would make the human gate system more usable.

The CLI pattern is also interesting — `openportal run` starts just the server, `openportal` starts server + web UI. Kanbanzai could have `kbz ui` for a local dashboard.

## 4. What We Do Better

### 4.1 Structured Governance and Entity Hierarchy

Kanbanzai's stage gate system is more rigorous than anything in the ecosystem. But the entity hierarchy itself is also a context-scoping instrument — when dispatching a task, context assembly walks the entity graph (Plan → Batch → Feature → Task), traces document sections via `doc_intel`, and injects relevant design context automatically. No OpenCode project has anything comparable. OmO has plans as flat markdown files. micode has artifact indexing but no typed entity graph. The entity hierarchy isn't just organizational — it's what makes automatic, scoped context assembly possible.

| Concept | OmO | Kanbanzai |
|---------|-----|-----------|
| Design approval required before spec | No equivalent | Enforced via document prerequisite |
| Spec approval required before implementation | No equivalent | Enforced via document prerequisite |
| Explicit dependency graph between tasks | None — Atlas does topological sort from plan text | `depends_on` edges in entity model, conflict tool validation |
| Review verdict aggregation | Ad-hoc | Structured: conformance, quality, security, testing dimensions with formal verdicts |
| Human gate enforcement | None — "human intervention is failure signal" | Explicit human gates at key stages |
| Override audit trail | None | Override + override_reason logged on entity |
| **Entity-driven context scoping** | Hook-based injection (AGENTS.md, mindmodel files) — flat, explicit paths | Automatic traversal of entity hierarchy; context trimmed by byte budget at assembly time |
| **Merge safety** | Not applicable — editor plugins don't manage merges | Semantic merge gates: all tasks done? verification record exists? documents current? branch not stale? |
| **Role + skill system** | Prompt templates (TypeScript functions generating system prompts) | Machine-readable YAML with inheritance, vocabulary constraints, anti-pattern detection, tool scoping |

OmO's philosophy ("human intervention is a failure signal") is the polar opposite of Kanbanzai's ("human owns intent, agent owns execution"). OmO optimizes for the case where the human is lazy or unavailable; Kanbanzai optimizes for the case where the human needs to maintain control and accountability over what gets built.

### 4.2 Git-Native Architecture

Kanbanzai is fundamentally Git-native:
- Worktree isolation for parallel features
- Entity state as schema-validated YAML tracked in Git
- Branch health monitoring, staleness detection, merge gate enforcement
- Clean separation between committed state (entity records) and local state (cache, local config)

OmO is file-system native (`.sisyphus/plans/`, `.sisyphus/boulder.json`) but doesn't integrate with Git's branching model. micode uses git worktrees for implementation isolation but doesn't track workflow state in Git.

### 4.3 Document Intelligence

Kanbanzai's `doc_intel` system (structural parsing, role classification, concept extraction, full-text search with FTS5) has no equivalent in any of the four projects. OmO's plans are markdown files with checkbox lists; micode's are timestamped files in `thoughts/shared/plans/`. Neither can query document structure, trace requirements to implementation, or classify sections by role.

### 4.4 Knowledge Management

Kanbanzai's knowledge system (deduplication, confidence scoring, TTL pruning, tier promotion, staleness detection) is more sophisticated than OmO's notepad system (markdown files in `.sisyphus/notepads/`). However, OmO's wisdom *accumulation* (automatic forwarding to subsequent sub-agents) is more automated than Kanbanzai's contribution model.

### 4.5 Entity Model Coherence

Kanbanzai has a formal entity model with referential integrity, lifecycle state machines, and validation. OmO's equivalents are ad-hoc: plans are markdown files, boulder.json tracks session state, categories are runtime routing rules. There's no unified model connecting plans → tasks → verification.

## 5. What We Already Do But Differently

### 5.1 Orchestration Model

| Aspect | OmO | Kanbanzai |
|--------|-----|-----------|
| Orchestrator identity | Sisyphus (single agent, persistent) | Orchestrator role (any agent can adopt it) |
| Task dispatch mechanism | `task(category=..., prompt=...)` via OpenCode API | `handoff(task_id)` → `spawn_agent` via MCP |
| Parallel dispatch | Background agent pool, fire multiple simultaneously | `conflict` tool → check → parallel `handoff` calls |
| Sub-agent model selection | Automatic via category system | Single model (what the orchestrator uses) |
| Work tracking | Boulder.json (checkbox-based progress) | Entity lifecycle (ready → active → done) |

The fundamental difference: OmO is *agent-driven* (Sisyphus is the center of gravity, it decides everything), Kanbanzai is *system-driven* (the MCP server enforces rules, the orchestrator follows procedure).

### 5.2 Planning

| Aspect | OmO | Kanbanzai |
|--------|-----|-----------|
| Planning trigger | `@plan` command or Tab → Prometheus | designing → specifying → dev-planning stage sequence |
| Planning style | Interview mode (agent questions human) | Human writes design doc → spec author formalizes → architect decomposes |
| Plan validation | Metis + Momus (automated) | Human gate at each stage |
| Plan output | Single `.sisyphus/plans/{name}.md` | Three separate documents (design, spec, dev-plan) with formal approval |

OmO's interview mode is genuinely interesting — the agent drives requirement discovery by asking questions based on codebase exploration. Kanbanzai's model assumes the human already knows what they want and writes it down. There's a middle ground where Kanbanzai could support agent-led requirement elicitation while maintaining the stage gate structure.

### 5.3 Session Continuity

| Aspect | OmO | Kanbanzai |
|--------|-----|-----------|
| Continuity mechanism | Boulder.json + session IDs + `/start-work` resume | Phase 5 context compaction + progress document + fresh session |
| Cross-session learning | Wisdom accumulation forwarded to sub-agents | Knowledge entries contributed at finish, queried at next claim |
| Crash recovery | Automatic: session recovery from API failures, context limits | Manual: orchestrator must detect and re-dispatch |

### 5.4 Document Management

| Aspect | OmO | Kanbanzai |
|--------|-----|-----------|
| Plan storage | `.sisyphus/plans/*.md` | `work/design/`, `work/spec/`, `work/plan/` + document records |
| Approval model | None (plan is generated, accepted, or rejected) | Formal approval status with `doc(action: "approve")` |
| Document discovery | File system | `doc(action: "list")`, `doc_intel(action: "search")` |

## 6. Features Worth Exploring / Adding to Kanbanzai

### 6.1 High Priority: Hash-Anchored Edit Validation

**Source:** OmO  
**Effort:** Medium (requires changes to `edit_file` or a new edit tool)  
**Impact:** High — addresses the harness problem directly  

OmO's Hashline system validates every edit against content hashes. If the file changed since the last read, the edit is rejected before corruption. The claimed improvement (6.7% → 68.3%) is extraordinary if accurate.

Kanbanzai could implement this as:
- A new MCP tool (`hash_edit`) that returns hashed lines and accepts hash-anchored edits
- Or an enhancement to `edit_file` that returns hash-tagged content and validates edits against hashes

This is fundamentally an edit-tool problem, not a workflow problem — it benefits any agent doing any kind of code editing.

**Compatibility with codebase-memory-mcp:** Hash-anchored edits and the knowledge graph operate at different layers and don't conflict. The graph indexes filesystem state — it doesn't care *how* a file was edited, only *that* it changed. Hash-anchored validation ensures edits are correct before they land on disk, which means the graph indexes clean state rather than corrupted edits. The graph's `detect_changes` tool becomes more reliable as a result. There's also a potential synergy: `get_code_snippet` (the graph's code-reading tool) could return hash-tagged lines, giving agents stable structural understanding AND stable line identifiers for editing in a single call — no separate hashing pass needed. This is an optimization to explore during implementation, not a prerequisite.

### 6.2 High Priority: Automated Plan Quality Validation

**Source:** OmO (Metis + Momus)  
**Effort:** Medium (new `plan-validator` role + skill) — superseded by Section 11  
**Impact:** High — catches gaps before implementation  

**Note:** This feature was initially scoped as plan-only validation, but Section 11 now provides a complete fast-track architecture covering spec validation, dev-plan validation, and review gate validation with risk-tiered automation. The checks below are incorporated into the plan-validator design (Section 11.4.2). The broader fast-track system is the recommended implementation path.

Kanbanzai's dev-planning stage has a human gate, but no automated quality check. Adding a validation pass that checks:
- ≥80% of tasks have clear reference sources
- ≥90% of tasks have concrete acceptance criteria
- Zero tasks require assumptions about business logic
- File references verified

These checks (and more) are formalized in the plan-validator design in Section 11.4.2, as part of the complete fast-track architecture.

### 6.3 Medium Priority: Agent-Led Requirement Elicitation

**Source:** OmO (Prometheus interview mode)  
**Effort:** Medium (new `spec-interviewer` role + skill)  
**Impact:** Medium — improves spec quality when human doesn't know exactly what they want  

OmO's Prometheus interviews the human, asking clarifying questions based on codebase exploration. Kanbanzai could add an optional "interview mode" to the specifying stage — the spec-author role could be enhanced to proactively question the human about ambiguities rather than just formalizing what the human wrote.

This doesn't conflict with the stage gate model — the output is still an approved specification. The difference is how the specification is produced.

**When this matters vs. when it doesn't:** Kanbanzai already supports collaborative requirement discovery through natural discussion — the human and agent discuss findings, converge on what matters, and form a shared understanding. That *is* requirement elicitation, and for an engaged human who knows what they want, it works well. Prometheus-style interviewing adds two things: (a) a systematic checklist that ensures nothing is left implicit (core objective? scope boundaries? ambiguities? approach? test strategy?), and (b) codebase-aware questioning — Prometheus explores the codebase *before* asking questions, so its prompts are grounded in real patterns rather than generic templates. The primary value is for cases where the human has a fuzzy idea and needs the agent to pull details out systematically. Adopting the checklist pattern (even without the full interview mode) would strengthen Kanbanzai's current approach by preventing implicit assumptions from surviving into the spec.

### 6.4 Medium Priority: Automated Wisdom Forwarding

**Source:** OmO (wisdom accumulation)  
**Effort:** Medium (enhance `handoff` to auto-include relevant knowledge)  
**Impact:** Medium — reduces repeated mistakes across tasks  

When Task 1 discovers a convention, Task 5 should benefit automatically. Kanbanzai's `handoff` already includes knowledge entries, but the process is pull-based (orchestrator must query knowledge). OmO's model is push-based: learnings automatically flow to subsequent sub-agents.

Implementation: When a task completes via `finish`, automatically surface its knowledge entries in the next `handoff` call for tasks in the same feature. This could be opt-in per task or automatic for tier-2 knowledge.

### 6.5 High Priority: Model Routing & Thinking-Level Control (capability unlock)

**Source:** OmO (category system)  
**Effort:** Large (requires model provider integration)  
**Impact:** High — unlocks multiple other features; architectural gate for thinking-level control, auto-compaction, and true Ralph Loop  

OmO's insight that different models excel at different task types is powerful. Kanbanzai currently assumes a single model. Supporting model routing would require:

1. A model abstraction layer in Kanbanzai (provider registry, fallback chains)
2. Task categorization (mapping task types to model categories)
3. Direct API integration with model providers (Anthropic, OpenAI, DeepSeek, etc.)

This is a significant architecture change — Kanbanzai would need to become an agent launcher, not just a workflow coordinator. But it aligns with the question of what's possible if Kanbanzai could launch agents through direct API.

**Thinking-level control is the same feature.** Controlling model selection (which provider), reasoning depth (extended thinking on/off), and provider fallback are all the same architectural decision — they're all blocked by the same MCP constraint. The MCP protocol is tool-call-in, tool-result-out. The server has no visibility into or control over the client's model, temperature, thinking mode, or token budget. Kanbanzai can *suggest* in a prompt ("this task benefits from deep reasoning"), but it cannot enforce it. The only way to get real control over thinking levels is to own the dispatch — either embedded model routing within `kbz serve` or a separate routing MCP server. If you build model routing, you get thinking-level control, provider fallback, and category-based model selection as a single capability. If you don't, you get none of them.

### 6.6 Medium Priority: Auto-Compaction at Threshold

**Source:** micode  
**Effort:** Low (procedural enhancement) in theory — but **architecturally blocked** in current form  
**Impact:** Medium — prevents context degradation  

Kanbanzai's orchestrate-development skill already describes context compaction (Phase 5), but it's procedural — the orchestrator must notice and act. An automated mechanism would be more reliable. The question is whether it's possible.

**Architectural constraint:** In Kanbanzai's current MCP-server architecture, the server has no visibility into the agent's context window. It receives tool calls and returns results — it doesn't know token counts, context utilization, or whether the agent is approaching its limit. The only observable signal is tool call volume, which is a weak heuristic at best (different models have different context windows, different overhead per tool call, different response verbosity). This is also true for sub-agents launched via `spawn_agent` — Kanbanzai sees only the final output, nothing about context pressure during execution.

**When it becomes possible:** Auto-compaction requires owning the agent conversation loop. If Kanbanzai adopts model routing and launches agents directly through provider APIs (Anthropic, OpenAI, DeepSeek), it can track token usage from API response metadata (`usage.input_tokens`) and estimate context utilization after each turn. At that point, an automatic compaction trigger at a configured threshold becomes feasible. Until then, Kanbanzai's procedural approach (Phase 5: orchestrator notices, writes progress document, starts fresh session) is the right mechanism — it works within the architectural constraints and gives the orchestrator explicit control over when to compact.

**Updated assessment:** Not a standalone feature. It's a capability unlocked by model routing / direct API dispatch (Design C in Section 9). Keep the current procedural approach; revisit when the agent launcher architecture is in place.

**Compaction artifact design: U-shaped continuation prompt, not summary.** When compaction does become feasible, the artifact it produces matters. There are two approaches: (a) a summary of what happened (Kanbanzai's current Phase 5 progress document), or (b) a U-shaped continuation prompt that positions the agent to resume work without recounting history. The U-shaped approach is better suited for auto-compaction within a single orchestration session because the agent is handing off to *itself in a fresh session* — it doesn't need to remember the journey, it needs to know exactly where to continue. A U-shaped compaction prompt captures: active state (tasks done/in-flight/ready), active decisions (not historical — only decisions still constraining current work), active constraints (file ownership, dependency boundaries), and surfaced knowledge (KE-IDs for the new session to query). It explicitly discards: task completion details, historical reasoning chains, conversation structure, and failed attempts whose conclusions are already in knowledge entries. The prompt engineering guide in `refs/prompt-engineering-guide.md` provides the structural template for constructing these prompts — identity at the top, constraints in the attention peak, procedural "continue from Phase N" at the bottom. A `compact-orchestration-session` SKILL that produces U-shaped continuation prompts (rather than historical summaries) is a promising area for further research, particularly once model routing makes automatic compaction feasible.

### 6.7 Low Priority: Web UI / Dashboard

**Source:** Portal  
**Effort:** Large (new frontend + API surface)  
**Impact:** Medium — improves human experience of gates and checkpoints  

Portal demonstrates that a web UI for agent interaction is valuable, especially for mobile access. For Kanbanzai, the most valuable UI would be:
- Checkpoint response (answering questions without being in a chat session)
- Document approval (approving/rejecting docs from a dashboard)
- Progress monitoring (which features are in which stage)

This could start as a lightweight `kbz ui` command that serves a local dashboard.

**Priority is contingent on fast-track adoption:** The primary use cases for a web UI — checkpoint response, document approval — are human gates. If the fast-track architecture (Section 11) eliminates most human gates, the UI's value proposition shrinks considerably. After fast-track, the human's interaction with Kanbanzai is almost entirely the design conversation, which happens in chat anyway. Progress monitoring becomes the remaining use case, and `status` already covers that. Revisit priority after fast-track is deployed and the actual residual human touchpoints are known. If fast-track eliminates 80%+ of human gates, a web UI may not be worth the investment.

### 6.8 Low Priority: Ralph Loop / Continuous Execution

**Source:** OmO  
**Effort:** Medium (enhance orchestration procedure)  
**Impact:** Medium — reduces orchestrator idle time  

OmO's Ralph Loop keeps the agent working until done without re-prompting. Kanbanzai's orchestrator must go through Phase 2→3→4→2 cycles explicitly. A "continuous mode" where the orchestrator automatically proceeds to the next dispatch batch without re-prompting would reduce latency.

**Context exhaustion risk:** The Ralph Loop's main failure mode is silent context saturation — the agent keeps looping until its context window fills, at which point responses degrade or the session crashes. OmO mitigates this because it owns the agent runtime and can auto-compact. In Kanbanzai's current MCP-server architecture, the orchestrator has no visibility into its own context utilization (same constraint as auto-compaction — see Section 6.6). A continuous execution loop without a compaction mechanism would eventually exhaust context.

**What's feasible today:** A procedural guard rather than true autonomous looping. After every N task completions (where N is calibrated per model — perhaps 5-8 for Claude Opus, 10-15 for models with larger context windows), the orchestrator pauses, writes a progress document per Phase 5, and instructs the human to start a fresh session. This gives most of the throughput benefit without the context-exhaustion risk. It's implementable as an enhancement to the orchestrate-development skill with no architecture changes.

**What becomes possible with model routing:** True Ralph Loop behavior — continuous autonomous execution with automatic compaction and resume — if Kanbanzai owns the agent conversation loop and can track token counts from API response metadata. Like auto-compaction, this is a capability unlocked by Recommendation 5, not a standalone feature.

## 7. Architectural Implications

### 7.1 What We'd Need to Change for Model Routing

If Kanbanzai were to adopt category-based model routing (OmO's most ambitious feature):

1. **Model provider abstraction:** New `internal/model/` package with provider registry, model capability descriptions, fallback chain logic
2. **Task categorization:** New field on task entities (`category: ultrabrain | deep | quick | visual-engineering`) 
3. **Direct agent launching:** Kanbanzai would need to call AI provider APIs directly (Anthropic Messages API, OpenAI Chat Completions, etc.) rather than relying on the host agent's model
4. **Role-to-model mapping:** Stage bindings would specify not just which role but which model category to use
5. **Cost tracking:** Token usage tracking across providers for budget management

This is a fundamental expansion of scope — Kanbanzai would become both a workflow coordinator and an agent launcher, akin to what OmO does but with structured governance.

**Alternative: separate MCP server for model routing.** Rather than embedding model routing in Kanbanzai, it could be a standalone MCP server that exposes `dispatch_task(category, prompt)`, `task_status(id)`, and `task_result(id)`. The orchestrator would call both servers — Kanbanzai for workflow decisions, the routing server for model dispatch. Arguments for separation: (a) clean boundary between workflow management and model routing — different operational domains with different failure modes, (b) independent evolution — the routing server can track provider changes at provider-speed without touching Kanbanzai's release cycle, (c) reuse — model routing is useful to any MCP-based agent, not just Kanbanzai orchestrators. Arguments against: (a) two servers to run and configure, (b) handoff friction — Kanbanzai's context assembly would need to serialize into the routing server's prompt format, (c) the orchestrator has to bridge two servers for decisions that are coupled (which model for this task depends on Kanbanzai context like role and stage). The middle ground: build as a separate Go package within Kanbanzai (`internal/routing/`) with a clean internal interface, ship as part of `kbz serve` initially. If it proves useful beyond Kanbanzai, extracting to a standalone server is a packaging change, not an architecture change. This keeps the option open without the operational overhead upfront.

### 7.2 What We'd Need to Change for Beyond-Chat Interface

If Kanbanzai were to add a CLI/web UI beyond the chat interface:

1. **CLI commands:** `kbz ui` (serve dashboard), `kbz approve DOC-xxx` (approve from CLI), `kbz checkpoint respond CHK-xxx` (respond from CLI)
2. **API surface:** The MCP server already has the tools; exposing them via REST or WebSocket would enable UI
3. **Authentication:** For remote access (Portal-style), auth would be needed
4. **Real-time updates:** WebSocket or SSE for live status changes

This is additive — the MCP server remains the core, with alternative interfaces layered on top.

### 7.3 What We'd Need to Change for Hash-Anchored Edits

1. **New or enhanced MCP tool:** `hash_edit` that returns content with line hashes and accepts hash-anchored edits
2. **Hash computation:** Simple content hash per line (OmO uses 2-char hex from a larger hash)
3. **Validation logic:** Before applying edit, verify hash of target lines matches
4. **Fuzzy match integration:** Could coexist with current `edit_file` — `edit_file` for simple cases, `hash_edit` for surgical edits

This is the most contained change with the highest claimed impact.

## 8. Assessment of OmO's Claims

### 8.1 Claims That Need Validation

- **"6.7% → 68.3% success rate from hash-anchored edits":** This is a specific, testable claim. The Grok Code Fast 1 benchmark is presumably a SWE-bench variant. Would need independent reproduction.
- **"Anthropic blocked OpenCode because of us":** The README states this twice as fact. The maintainer claims Anthropic blocked OpenCode from using their API. This is unverifiable but would explain the aggressive anti-Anthropic positioning.
- **"Kimi K2.5 + GPT-5.4 already beats vanilla Claude Code":** Untestable without defined benchmarks. The claim is about orchestration benefit, not single-model capability.
- **"55.5k stars":** Extraordinary for a plugin project. Stars ≠ active users, but the community traction is real.

### 8.2 Claims That Are Architecturally Sound

- **Multi-model orchestration is the right long-term architecture:** As models specialize and get cheaper, routing by task type rather than picking one winner makes sense.
- **The harness problem is real:** Most agent failures are edit-tool failures, not reasoning failures. Hash-anchored edits are a genuine improvement.
- **Category-based routing is better than model-name routing:** Decoupling task semantics from model selection is architecturally cleaner.
- **Wisdom accumulation across tasks improves consistency:** Forwarding learnings to subsequent sub-agents reduces repeated mistakes.

## 9. Implementation Roadmap

The eight features in Section 6 are not independent. Several are architecturally gated by the same decision, and others supersede or deprioritize each other. They collapse into three design efforts, two small enhancements, and one deferred decision:

```
Design A: Hash-Anchored Edits (6.1)          ← standalone, start here
Design B: Fast-Track Architecture (6.2 + §11) ← supersedes 6.2, deprioritizes 6.7
Design C: Model Routing & Agent Launcher (6.5) ← unlocks 6.6 + full 6.8

Enhancement 1: Wisdom Forwarding (6.4)       ← small, build anytime
Enhancement 2: Elicitation Checklist (6.3)   ← adopt pattern into existing skill

Defer: Web UI (6.7)                          ← revisit after Design B deployed
```

### 9.1 Design A: Hash-Anchored Edit Tool

**Priority:** Start immediately. Highest impact, lowest risk, zero architectural dependencies.

**What it is:** A new MCP tool (`hash_edit`) or enhancement to `edit_file` that returns content with per-line content hashes and validates edits against those hashes before applying. If the file changed since the agent last read it, the edit is rejected before corruption.

**Why first:** Both reports independently identify this as the top priority. It's a tool-level change — no entity model changes, no new roles, no stage binding modifications. It benefits every agent doing any kind of code editing, regardless of workflow stage. The knowledge graph (`codebase-memory-mcp`) is compatible and there's potential synergy: `get_code_snippet` could return hash-tagged lines, giving agents structural understanding and stable edit identifiers in one call. See Section 6.1 for details.

**Deliverables:**
- Design document: hash-edit tool schema, hash format, validation logic, error modes
- Proof-of-concept implementation
- Integration with `edit_file` or standalone `hash_edit` MCP tool
- Optional: `get_code_snippet` hash-tagged output

### 9.2 Design B: Fast-Track Architecture

**Priority:** Start after Design A is stable. High impact, medium effort, no architectural dependencies.

**What it is:** The complete automated gate validation system designed in Section 11 — three new validator roles (spec-validator, plan-validator, review-gate-validator), risk-tiered automation levels (retro_fix, bug_fix, feature, critical), and an auto-approval pipeline that replaces mechanical human gates with evidence-backed automated checks.

**Why second:** This is the most distinctive Kanbanzai innovation from this research — no competitor has anything like it. It directly addresses the "humans mostly type LGTM after design" observation. It uses Kanbanzai's existing strengths (entity hierarchy, document intelligence, spawn_agent) rather than requiring new capabilities. The phased implementation (spec first, then dev-plan, then review) means each phase delivers value independently.

**Side effect:** Reduces the urgency of a web UI (6.7). If most human gates are automated, the primary use cases for a dashboard (checkpoint response, document approval) largely disappear. Revisit web UI priority after deployment.

**Deliverables:**
- Three new roles: `spec-validator`, `plan-validator`, `review-gate-validator`
- Three new skills: `validate-spec`, `validate-dev-plan`, `validate-review-gate`
- Extended structural validators (enhance `validate-spec-structure.sh`, new `validate-dev-plan-structure.sh`)
- Fast-track configuration in `.kbz/config.yaml` (risk tiers, escalation rules)
- Tier inference from entity context (retro signal → retro_fix, bug entity → bug_fix, etc.)
- Auto-approval pipeline in stage bindings (validator runs → pass → auto-approve)
- Fresh-session enforcement for validators (always `spawn_agent`, never continue)

**Phased rollout:**
1. Spec auto-validation (lowest risk) — run alongside human gates for calibration
2. Dev-plan auto-validation — once spec validator is calibrated
3. Review auto-close — once plan validator is stable
4. Full fast-track with risk tiers — once all validators are proven

### 9.3 Design C: Model Routing & Agent Launcher

**Priority:** Start with a feasibility design document. Do not commit to building until Designs A and B are stable.

**What it is:** A model provider abstraction that lets Kanbanzai launch agents directly through AI provider APIs (Anthropic, OpenAI, DeepSeek, etc.) rather than relying on the host MCP client's model. Includes category-based routing, thinking-level control, provider fallback chains, and token tracking.

**Why third:** This is the largest architectural change — Kanbanzai goes from MCP tool server to agent launcher. But it's also a capability unlock: auto-compaction (6.6), thinking-level control per task, and true Ralph Loop (6.8) all become possible as natural consequences. The separate-server alternative (a standalone model-routing MCP server that Kanbanzai calls) should be evaluated in the design document — it preserves Kanbanzai's simplicity while still delivering the capability. See Section 7.1 for the architectural analysis.

**Deliverables:**
- Feasibility design document: embedded vs. separate server, provider integration scope, API surface, token budget model
- If feasible: provider abstraction layer, fallback chain logic, thinking-level control
- Integration with `handoff` / `spawn_agent` for category-based dispatch
- Token usage tracking and cost management

### 9.4 Small Enhancements (build anytime)

**Wisdom Forwarding (6.4):** Modify `handoff` to automatically surface knowledge entries from completed sibling tasks when dispatching new tasks in the same feature. A small change to the context assembly pipeline — no new roles, no new entities. Both reports identify this gap.

**Elicitation Checklist (6.3):** Adopt the Prometheus-style systematic checklist (core objective? scope? ambiguities? approach? test strategy?) into the existing spec-author skill. Not a new role or interview mode — just a structured prompt addition that prevents implicit assumptions from surviving into specs. The interview mode itself is lower priority; the checklist pattern is the valuable part.

### 9.5 Deferred (revisit later)

**Web UI / Dashboard (6.7):** Revisit after Design B is deployed. If fast-track eliminates 80%+ of human gates, the primary use cases for a dashboard disappear.

**LSP Integration (6.9):** Valuable but requires language-specific work. Lower priority than hash-anchored edits.

**Structured Convention Layer (6.10):** Watch micode's mindmodel. Revisit if convention drift becomes a documented problem.

### 9.6 Do Not Pursue

- **"Human intervention is failure" philosophy.** Contradicts Kanbanzai's core premise that humans own intent. The stage gate model is a feature, not a bug.
- **Ad-hoc entity model.** OmO's file-based state is adequate for its scale but doesn't scale to multi-team coordination. Kanbanzai's structured entity model is superior for traceability.

## 10. Open Questions

1. Can hash-anchored edits be implemented as an enhancement to `edit_file` rather than a new tool? The fuzzy matching in `edit_file` already handles some of the line-stability problem.

2. Would model routing make Kanbanzai too complex? The project's design principle is "always simpler than the project it manages." Adding multi-provider integration might violate this. **Partial resolution in Section 7.1:** A separate MCP server for model routing is an alternative that preserves Kanbanzai's simplicity — clean boundary, independent evolution, reusable outside Kanbanzai. The middle ground (separate Go package, ship together, extract later) keeps the option open without the upfront complexity.

3. ~~Is OmO's category system genuinely better than Kanbanzai's role system?~~ **Resolved in Section 11.3:** OmO and Kanbanzai have different document structures — OmO merges spec+plan into one document, Kanbanzai separates them. Direct feature comparison requires mapping across different workflow stages. See the artifact comparison table in 11.3.

4. What would a "Kanbanzai agent launcher" look like? If Kanbanzai could call Claude/DeepSeek APIs directly, would it replace the MCP client's agent or complement it?

5. Should we invest in a deeper evaluation of OmO's hash-anchored edit system with actual benchmarks before committing to implementation?

6. **New:** What's the right validator evidence threshold for auto-approval? If validators run alongside human gates for N features and catch everything humans catch, the threshold can be lower — but what's the calibration period?

7. **New:** Should fast-track tier be per-feature, per-batch, or both? A batch might contain mixed tiers (one critical feature + three standard features). The batch-reviewing stage may need independent tiering.

## 11. Automated Gate Validation & Fast-Track Architecture

### 11.1 Motivation

Kanbanzai's current stage gate model has three human gates after design: spec approval, dev-plan approval, and review verdict. In practice, after the design is approved, human gate-keeping is often mechanical — verifying that downstream artifacts are structurally complete and trace back to the design. "LGTM" and "Approved" are the most common responses. This section designs an automated validation system that can replace or augment these gates, and a fast-track path that allows features to flow from approved design to merged implementation with zero human intervention when appropriate risk conditions are met.

A secondary motivation is the retrospective fixup use case: when a retro finding identifies a specific, evidence-backed problem, the fix design is mechanically derivable from the evidence. There is no architectural judgment required — the retro finding defines what's broken, the fix is limited to resolving that specific issue, and acceptance criteria are implicit in the bug report. For these cases, even the design gate can be automated.

### 11.2 Validator vs. Reviewer: A Critical Distinction

Before designing the system, it's essential to clarify what a "validator" is and how it differs from a reviewer. These are different roles operating at different stages on different artifacts.

| Dimension | Validator | Reviewer |
|-----------|-----------|----------|
| **What it checks** | A document (spec, dev-plan) for structural completeness and traceability | Implementation (code, tests) for conformance to specification |
| **When it runs** | Before implementation — during specifying, dev-planning stages | After implementation — during reviewing stage |
| **Evidence examined** | Parent documents + the document under validation + file system | Code + test output + specification |
| **Question it answers** | "Is this document complete and correct enough to proceed?" | "Does this code satisfy the specification?" |
| **Failure meaning** | "Fix the document before we build from it" | "Fix the code before we merge" |
| **Cost of failure** | Low — caught before code is written | High — caught after code is written |
| **Cognitive profile** | Compliance audit: checklist-driven, binary checks, low creativity, high thoroughness | Evidence evaluation: finding classification, severity judgment, spec citation |

Validators answer questions like:
- Does every requirement have an acceptance criterion?
- Does every task trace to a specification requirement?
- Are all referenced files present on disk?
- Is the dependency graph acyclic?
- Does the verification plan cover every acceptance criterion?

Reviewers answer questions like:
- Does this implementation satisfy REQ-003?
- Is the error handling chain complete?
- Are there security vulnerabilities?
- Do tests cover boundary conditions?

A plan-validator catches problems that would otherwise surface during review, when they're more expensive. This is the same logic Kanbanzai applies to specifications — catch ambiguity before code is written.

### 11.3 OmO Workflow Comparison: Apples and Oranges

It's important to understand that OmO and Kanbanzai have different document structures, which affects what validation means in each system:

| Artifact | Kanbanzai | OmO |
|----------|-----------|-----|
| **Design** | Separate design document (approved, human-gated) | Implicit in Prometheus interview — design decisions are discovered during questioning |
| **Specification** | Separate specification document with REQ-IDs, acceptance criteria, constraints, verification plan | Embedded in Prometheus plan — tasks carry their own acceptance criteria inline |
| **Dev-Plan** | Separate dev-plan document with task breakdown, dependency graph, risk assessment | Embedded in Prometheus plan — task ordering, dependencies, and verification are part of the same artifact |
| **Plan quality validation** | Not automated (human gate) | Metis (gap analyzer) + Momus (ruthless reviewer) — automated |
| **Implementation review** | 4-specialist panel: conformance, quality, security, testing | Atlas verifies completion + LSP diagnostics (ad-hoc, not structured) |

OmO collapses specification and dev-planning into a single Prometheus step, producing one plan document that serves both roles. Its Metis + Momus validation loop is therefore a combined spec+plan validator. Kanbanzai's separation into three documents (design → spec → dev-plan) means:

1. **Kanbanzai has finer-grained validation points** — we can validate the spec independently of the plan, catching requirement-level issues before task decomposition begins.
2. **Kanbanzai's validators have stronger evidence** — because the spec exists as a separate artifact with formal REQ-IDs, a plan-validator can trace tasks to specific numbered requirements, which is a stronger check than OmO can perform.
3. **OmO's approach is simpler** — one document, one validation pass. Kanbanzai's approach is more rigorous but has more gates to manage.

The automated gate design below preserves Kanbanzai's rigor while removing the mechanical human-gate overhead.

### 11.4 Automated Validator Design

#### 11.4.1 Spec Validator

**New role:** `spec-validator` (inherits from `base`, NOT from `spec-author` — different identity, different anti-patterns)

**Identity:** "Senior requirements quality auditor. You verify that specifications are complete, testable, and traceable to their parent design. You do not evaluate whether the requirements are *correct* — only whether they are *well-formed* and *complete*."

**Key anti-patterns:**

- **Assumed Traceability:** Claiming a requirement traces to the design without finding the specific design section. Every traceability claim must cite a design section heading or paragraph.
- **Content Judgment:** Evaluating whether requirements describe the *right* thing. The design gate already validated correctness. The validator checks structure and completeness only.
- **Hallucinated Completeness:** Claiming a check passed without executing it. Every "pass" verdict must describe *how* the check was performed.

**Validation checks:**

| # | Check | Method | Severity |
|---|-------|--------|----------|
| S1 | All required sections present (Overview, Scope, Functional Requirements, Non-Functional Requirements, Acceptance Criteria, Verification Plan) | Structural (extend `validate-spec-structure.sh`) | Blocking |
| S2 | Overview references the parent design document by path or DOC-ID | Structural + `doc_intel` cross-reference | Blocking |
| S3 | Every functional requirement has a unique ID (REQ-xxx format) | Regex | Blocking |
| S4 | Every requirement ID appears in the Verification Plan | Cross-reference | Blocking |
| S5 | Every acceptance criterion is a testable assertion (not vague: "works correctly", "is fast") | LLM classification | Blocking |
| S6 | Acceptance criteria use checkbox format (per stage binding template) | Structural | Non-blocking |
| S7 | No requirement is a disguised implementation instruction ("use Redis for caching" vs. "cached reads must return in <10ms") | LLM classification | Non-blocking |
| S8 | Scope section explicitly states what is in-scope AND out-of-scope | Structural | Non-blocking |
| S9 | No orphaned requirements (present in spec but not derivable from design) | `doc_intel` trace | Non-blocking (escalates to human if found) |
| S10 | Non-functional requirements have measurable thresholds (not "fast", "scalable") | LLM classification | Blocking |

**Model selection:** Same model family as reviewers (GPT-5.4 or Claude Opus), but with temperature near zero. The spec-validator role prompt explicitly instructs: "Do not invent. Do not assume. Only verify. If you cannot determine whether a check passes, report it as uncertain and escalate." The anti-pattern "Hallucinated Completeness" is the validator equivalent of the reviewer's "Rubber-Stamp Approval."

#### 11.4.2 Dev-Plan Validator

**New role:** `plan-validator` (inherits from `base`, NOT from `architect`)

**Identity:** "Senior implementation plan auditor. You verify that dev-plans are complete, well-decomposed, and fully traceable to their parent specification. You do not evaluate whether the task ordering is *optimal* — only whether it is *valid* and *complete*."

**Key anti-patterns:**

- **Phantom Traceability:** Claiming a task traces to a spec requirement without finding the specific REQ-ID in the task's spec requirement field.
- **Architectural Second-Guessing:** Evaluating whether the decomposition strategy is *best*. The architect already made that judgment. The validator checks completeness and correctness.
- **Unverified File References:** Claiming a task's file references are valid without checking the file system or knowledge graph.

**Validation checks:**

| # | Check | Method | Severity |
|---|-------|--------|----------|
| D1 | All required sections present (Scope, Task Breakdown, Dependency Graph, Risk Assessment, Verification Approach) | Structural | Blocking |
| D2 | Scope section references the parent specification by path or DOC-ID | Structural | Blocking |
| D3 | Every task has a spec requirement reference (REQ-xxx or criterion ID) | Cross-reference | Blocking |
| D4 | Every spec requirement is covered by at least one task (no gaps) | Matrix coverage via `doc_intel` | Blocking |
| D5 | No task covers requirements not in the specification (no scope drift) | Matrix coverage | Blocking |
| D6 | Dependency graph is acyclic | Graph analysis | Blocking |
| D7 | No monolithic tasks: no task touches >3 files or >1 acceptance criterion | Structural | Non-blocking |
| D8 | Tasks with no shared files or dependencies are marked parallelisable | `conflict` tool | Non-blocking |
| D9 | Verification Approach maps every acceptance criterion to a producing task | Cross-reference | Blocking |
| D10 | Risk Assessment is non-empty and contains at least one risk with probability + impact + mitigation | Content check | Non-blocking |
| D11 | All file paths referenced in task deliverables exist (or are explicitly noted as to-be-created) | Filesystem + knowledge graph | Non-blocking |
| D12 | At least one task addresses each non-functional requirement from the spec | Matrix coverage | Non-blocking |

#### 11.4.3 Review Gate Validator

**New role:** `review-gate-validator` (inherits from `reviewer`)

**Identity:** "Senior review quality auditor. You verify that a completed review is thorough, evidence-backed, and suitable for auto-approval. You do not re-review the code — you audit the review process itself."

**Key anti-patterns:**

- **Re-Reviewing:** Evaluating the code instead of the review. The specialists already reviewed the code. The gate validator audits whether the review was done properly.
- **Rubber-Stamp Acceptance:** Accepting a review as adequate without checking per-reviewer evidence.

**Validation checks:**

| # | Check | Method | Severity |
|---|-------|--------|----------|
| R1 | Every dispatched reviewer produced a structured output with per-dimension evidence | Content analysis | Blocking |
| R2 | No reviewer output is a rubber-stamp (zero findings AND no per-dimension evidence citations) | Content analysis (MAST FM-3.1) | Blocking |
| R3 | No reviewer has >40% findings classified as blocking (severity inflation check) | Ratio | Non-blocking (flags for human) |
| R4 | Every blocking finding cites a specific acceptance criterion or spec requirement | Cross-reference | Blocking |
| R5 | Aggregate verdict is consistent with per-dimension outcomes (no dimension with `fail` but aggregate `pass`) | Logic check | Blocking |
| R6 | Deduplication pass was run — no identical finding appears from multiple reviewers without collapse | Overlap analysis | Non-blocking |
| R7 | Every acceptance criterion from the spec is covered by at least one reviewer's scope | Coverage check via `doc_intel` | Blocking |
| R8 | Reviewer selection was adaptive (not all 4 reviewers dispatched when change scope didn't warrant it) | Dispatch record check | Non-blocking |

### 11.5 Fast-Track Architecture

#### 11.5.1 Risk Tiers

The fast-track is not a binary on/off. Different kinds of work carry different risk profiles, and the automation level should match.

```yaml
# Proposed .kbz/config.yaml structure
fast_track:
  enabled: true
  default_tier: feature              # which tier to use when none specified
  tiers:
    retro_fix:
      description: "Fixes derived from retrospective findings — evidence-backed, scope-bounded"
      human_gates:
        design: false                # auto-generated from retro finding
        spec: false                  # auto-validated
        dev_plan: false              # auto-validated
        review: false                # auto-validated
      validators:
        - spec-validator
        - plan-validator
        - review-gate-validator
      max_auto_cycles: 3             # max remediation cycles before human escalation
      escalate_on:
        - validator_uncertainty      # validator can't determine pass/fail
        - cycle_limit_reached
        - scope_drift_detected       # implementation exceeds retro finding scope

    bug_fix:
      description: "Bug fixes with known root cause"
      human_gates:
        design: false                # auto-generated from bug report
        spec: true                   # human confirms the fix approach
        dev_plan: false              # auto-validated
        review: false                # auto-validated
      validators:
        - plan-validator
        - review-gate-validator
      max_auto_cycles: 2
      escalate_on:
        - validator_uncertainty
        - cycle_limit_reached

    feature:
      description: "Standard features — design is the anchor human gate"
      human_gates:
        design: true                 # human approves architecture
        spec: false                  # auto-validated (derived from approved design)
        dev_plan: false              # auto-validated (derived from approved spec)
        review: false                # auto-validated (checked against spec)
      validators:
        - spec-validator
        - plan-validator
        - review-gate-validator
      max_auto_cycles: 2
      escalate_on:
        - validator_uncertainty
        - cycle_limit_reached
        - design_ambiguity           # validator finds spec requires design decision not in design doc

    critical:
      description: "Security-critical, data-loss-risk, or compliance-relevant features"
      human_gates:
        design: true
        spec: true
        dev_plan: true
        review: true
      validators: []                 # no auto-approval — validators run but findings go to human
      max_auto_cycles: 0             # always escalate
      escalate_on:
        - always
```

#### 11.5.2 Tier Inference

When a feature is created, its fast-track tier can be:

1. **Explicitly set** on the feature entity: `entity(action: "create", type: "feature", fast_track_tier: "critical")`
2. **Inferred from context:**
   - Feature created from a retro signal → `retro_fix`
   - Feature linked to a bug entity → `bug_fix`
   - Feature with tag `critical` or `security` → `critical`
   - All other features → `default_tier` (typically `feature`)
3. **Changed at any time** by the human: `entity(action: "update", id: "FEAT-xxx", fast_track_tier: "critical")`

#### 11.5.3 Validation Pipeline

When a stage completes and the next stage has `human_gates: false`, the validator runs automatically:

```
Spec author completes spec
  → doc(action: "register", type: "specification")
  → fast-track check: tier says spec gate is auto?
    → YES: spawn_agent(role: spec-validator, skill: validate-spec)
      → validator returns: pass / pass_with_notes / fail
        → pass: doc(action: "approve", id: "DOC-xxx")  ← automatic
        → pass_with_notes: doc(action: "approve") + notes attached to document
        → fail: escalate to human with findings
    → NO: present spec to human for approval (current behavior)
```

#### 11.5.4 Session and Context Management

Validators MUST run in fresh sessions to avoid context degradation:

1. **Clean context:** The validator receives only: (a) the document under validation, (b) the parent document (design for spec-validator, spec for plan-validator), and (c) the validation checklist. It does NOT receive the conversation that produced the document.
2. **Spawn, don't continue:** Validators always run via `spawn_agent` with a fresh context window, not as a continuation of the author's session.
3. **Output reduction:** Validator output reduces to: verdict (pass/pass_with_notes/fail) + N findings + evidence score. Full output is offloaded to a document record. The orchestrator receives only the summary.
4. **Cycle tracking:** Each validator run increments a counter. When `max_auto_cycles` is reached, escalation to human is mandatory regardless of validator verdict.

#### 11.5.5 Guarding Against Fast-Track Failure Modes

| Failure Mode | How It Manifests | Mitigation |
|-------------|-----------------|------------|
| **Validator deadlock** | Validator finds issue → fix → re-validate → new issue found → loop | `max_auto_cycles` cap. After N cycles, escalate to human with cycle summary |
| **Context saturation** | Validator runs in same session as author, context is degraded | Validators always run in fresh sessions via `spawn_agent` |
| **Validator sycophancy** | Validator approves because it's easier than finding issues | Validator role has "Hallucinated Completeness" / "Rubber-Stamp" anti-patterns. Every pass verdict requires evidence of how the check was performed |
| **Drift accumulation** | Each automated stage makes small allowances that compound into spec-implementation gap | Validators cross-reference upstream, not just immediate parent: plan-validator checks against spec, review-gate-validator checks against spec (not just plan) |
| **Orchestrator context bloat** | Orchestrator accumulates full validator output across multiple stages | Post-completion summarization (Phase 5). Validator output reduces to verdict + N findings + score. Full output in document record |
| **Validator misses what human would catch** | The "unknown unknown" — a problem outside the validation checklist | Mitigated by: design gate remains human-anchored. If the design is correct, downstream checks are verifying mechanical derivation. For retro_fix tier, the evidence itself defines correctness |
| **Scope drift in retro_fix** | Implementation exceeds the retro finding's scope because no human reviewed | `escalate_on: scope_drift_detected` — if implementation touches files or concepts beyond the retro finding's blast radius, escalate |

### 11.6 The Retro Fixup: Zero Human Gates

The `retro_fix` tier is the most automated path. Here's why it's safe:

1. **Design is derived from evidence, not intent.** A retro finding says "merge tool fails on worktrees with uncommitted .kbz/ changes." The fix design is: "check for uncommitted .kbz/ changes before merge and either commit or warn." This is mechanically derivable — there's no architectural judgment about *whether* to fix it, only *how*.

2. **Acceptance criteria are implicit in the finding.** "Merging with uncommitted .kbz/ changes should not fail" is the acceptance criterion. It's testable: create a worktree with uncommitted .kbz/ changes, attempt merge, verify it succeeds or produces a clear error.

3. **Scope is bounded by the retro finding.** The fix is limited to resolving the specific issue reported. The retro finding defines the blast radius.

4. **Review is fully automatable.** Does the fix resolve the finding? Do existing tests still pass? These are checkable by validators + test suites.

For a `retro_fix`, the complete automated flow:

```
Retro signal → Auto-generated design (structured write-up of finding + approach)
  → doc(action: "register", type: "design", auto_approve: true)
  → Auto-generated spec (formalized acceptance criteria from finding)
  → spec-validator runs → pass → doc(action: "approve")
  → Auto-generated dev-plan (typically 1-2 tasks)
  → plan-validator runs → pass → doc(action: "approve")
  → Implementation
  → Specialist review panel
  → review-gate-validator runs → pass → merge(action: "execute")
```

No human touched any gate. The entire flow completed from retro signal to merged fix.

### 11.7 Implementation Sequence

The fast-track system should be built incrementally, validating each stage before adding the next:

**Phase 1: Spec Auto-Validation** (lowest risk, highest immediate value)
- New role: `spec-validator`
- New skill: `validate-spec`
- Extend `validate-spec-structure.sh` with checks S1-S10
- Wire into stage binding: after spec registration, if fast-track tier allows, run validator
- Human gate remains the default until fast_track is explicitly enabled

**Phase 2: Dev-Plan Auto-Validation**
- New role: `plan-validator`
- New skill: `validate-dev-plan`
- Build matrix coverage analysis using `doc_intel` for requirement-to-task tracing
- Wire into stage binding after dev-plan registration

**Phase 3: Review Auto-Close**
- New role: `review-gate-validator`
- Enhance `orchestrate-review` with post-collation validation pass
- If validation passes and tier allows, auto-close review without human checkpoint

**Phase 4: Full Fast-Track**
- Risk tier configuration in `.kbz/config.yaml`
- Tier inference from entity context
- `max_auto_cycles` enforcement
- Escalation rules

### 11.8 Design Decisions and Rationale

**Why validators inherit from `base`, not from `reviewer` (except review-gate-validator):**

Validators and reviewers have different identity, vocabulary, and anti-patterns. A spec-validator is not a kind of reviewer — it's an auditor. Inheriting from `base` gives a clean identity. The review-gate-validator IS a kind of reviewer (it audits review quality), so it inherits from `reviewer`.

**Why validators don't evaluate correctness, only completeness:**

The design gate validates correctness — "are we building the right thing?" Validators validate completeness — "is this document sufficient to proceed?" Mixing these creates confusion about who owns what decision. The design gate is the single anchor point for human judgment about correctness.

**Why the design gate remains human even in `feature` tier:**

Architectural judgment — choosing between alternatives, assessing tradeoffs, deciding what NOT to build — cannot be reduced to a checklist. This is where human intent and product knowledge matter most. Everything downstream of design is derivation and verification, which are automatable.

**Why `retro_fix` can skip the design gate:**

A retro fix's design isn't really a design — it's a structured problem statement with a mechanically derivable solution. The retro finding already answered "what's wrong" and "what should happen instead." The fix design just formalizes that into a document. No architectural judgment is required.

**Why validators always run in fresh sessions:**

Context degradation is the primary threat to validator reliability. A validator running in the author's session sees the reasoning that produced the document and is biased by it. A fresh session sees only the document itself — exactly what a human reviewer would see when evaluating it cold.

### 11.9 Open Design Questions

1. **Should validators use a different model than authors/reviewers?** The cognitive profile is different (compliance audit vs. creative synthesis vs. evidence evaluation). If Kanbanzai ever adopts model routing, validators would be a distinct category ("thorough, low-temperature, checklist-driven"). Until then, same model with different role prompt and near-zero temperature.

2. **What's the validator evidence threshold for auto-approval?** If a validator passes at 85% confidence (some checks uncertain), should that auto-approve or escalate? This needs empirical calibration — run validators alongside human gates for N features and compare. If validators catch everything humans catch, the threshold can be lower.

3. **Should fast-track tier be per-feature or per-batch?** Per-feature gives finer control. A batch might contain one critical feature and three standard features. But the batch review stage (batch-reviewing) might need its own tier logic.

4. **How does fast-track interact with `override`?** Currently, a human can always call `entity(action: "transition", override: true)` to bypass any gate. Fast-track reduces the need for overrides by automating the common case. The escape hatch remains.

5. **Can a validator's pass/fail decision be appealed?** If a spec-validator fails a spec and the author disagrees, the human can override the validator (just like any gate). The validator's findings become advisory, not binding.


## 12. Limitations and Cross-Validation

This report was cross-validated against an independent evaluation which conducted code-level review of opencode-background-agents plugin source and micode's executor agent. Areas of strong convergence are noted throughout. Key limitations:

1. **Evidence quality varies.** OmO was reviewed via documentation and orchestration guide; opencode-background-agents and micode received partial code review in the companion evaluation; Portal was README-level only.
2. **No runtime testing.** None of these projects were installed or tested. Claims about benchmark improvements (Hashline's 6.7% to 68.3%) and throughput (micode's 10-20 concurrent agents) are taken from project documentation.
3. **OpenCode platform dependency.** All four projects are plugins for OpenCode (sst/opencode). Their architectures assume OpenCode's session model, tool system, and hook system. Features that appear superior may leverage platform capabilities Kanbanzai would need to replicate independently.
4. **Rapidly evolving ecosystem.** OmO has 5,400+ commits and 170 releases. Findings may date quickly. Recommendations should be treated as directional.
5. **Scope of automated gate validation.** Section 11 proposes a fast-track architecture with no direct analogue in the evaluated projects. The design is based on Kanbanzai's internal capabilities (doc_intel, entity model, spawn_agent) and has not been validated against external implementations. The companion evaluation does not address automated gate validation.

## 13. Plan-of-Plans: Post-Research Initiative

The findings from this report consolidate into a parent plan that owns the strategic direction, with individual designs spun out as sub-plans. This mirrors the uphill planning workflow — the parent plan captures intent and decomposition; sub-plans are implemented and closed independently without blocking the overall initiative.

### 13.1 Dependency Structure

```
Plan: "OpenCode ecosystem features" (P-xxx)
│
├── Sub-plan A: Hash-Anchored Edit Tool
│   Standalone. Zero dependencies. Ready to design now.
│   Source: Section 6.1
│
├── Sub-plan B: Fast-Track Architecture
│   Standalone. No dependencies on A or C. Can start in parallel with A.
│   Phased: spec validator → plan validator → review gate validator → risk tiers
│   Source: Section 11
│   Side effect: deprioritizes web UI (6.7) if successful
│
├── Sub-plan C: Model Routing & Agent Launcher
│   Start with feasibility design only. Do not commit to build until A and B are stable.
│   Unlocks: auto-compaction (6.6), thinking-level control, true Ralph Loop (6.8)
│   Source: Sections 6.5, 7.1
│
├── Enhancement: Wisdom Forwarding
│   Small, standalone. Modify handoff to auto-surface sibling task knowledge.
│   Source: Section 6.4
│
└── Enhancement: Elicitation Checklist
    Small, standalone. Adopt Prometheus-style checklist into spec-author skill.
    Source: Section 6.3
```

### 13.2 Sub-Plan Sequencing

| Order | Sub-plan | Can start | Depends on | Estimated effort |
|-------|----------|-----------|------------|------------------|
| 1 | A: Hash-Anchored Edits | Immediately | Nothing | Medium (new MCP tool) |
| 2 | Wisdom Forwarding | Immediately | Nothing | Small (handoff enhancement) |
| 3 | Elicitation Checklist | Immediately | Nothing | Small (skill update) |
| 4 | B: Fast-Track Architecture | Immediately (parallel with A) | Nothing | Medium-Large (3 roles, 3 skills, config, pipeline) |
| 5 | C: Feasibility Design | After A and B stable | Nothing (design-only phase) | Small (design document) |
| 6 | C: Implementation | After feasibility approved | C feasibility design | Large (provider integration, agent runtime) |

A and B can proceed in parallel — they touch different parts of the system (edit tools vs. stage gates). The two small enhancements can be done at any point. C is intentionally deferred: start the design to capture thinking while it's fresh, but don't build until A and B prove the pattern of adopting ecosystem features into Kanbanzai.

### 13.3 Deferred Items

These are intentionally NOT sub-plans. They're captured here so the intention is recorded, but they'll be revisited based on outcomes from the sub-plans above:

| Item | Deferral condition | Revisit trigger |
|------|-------------------|-----------------|
| Web UI (6.7) | Deprioritized if Design B eliminates most human gates | After fast-track deployment — measure residual human touchpoints |
| Auto-Compaction (6.6) | Requires Design C | After model routing is implemented |
| Ralph Loop — full (6.8) | Requires Design C | After model routing is implemented |
| LSP Integration (6.9) | Lower priority than edit tool improvements | After Design A is stable |
| Mindmodel Convention Layer (6.10) | Unclear gap — existing mechanisms may suffice | If convention drift becomes a documented problem |

### 13.4 Plan Lifecycle

This plan follows the uphill planning pattern:
- **Shaping:** This report serves as the shaping artifact — it identifies what to build, what depends on what, and what to defer.
- **Ready:** The plan advances to ready when the sub-plan decomposition is agreed and Design A is approved.
- **Active:** Sub-plans are spawned and closed independently. The parent plan remains active as long as any sub-plan is in flight.
- **Done:** When all sub-plans are closed or explicitly deferred.

Sub-plans that aren't starting immediately (C implementation, deferred items) remain as intended scope in the parent plan without blocking closure. The parent plan can close with deferred items recorded as decisions, not as incomplete work.

## Appendix: Quick Comparison Matrix


| Dimension | Kanbanzai | OmO | micode | bg-agents | Portal |
|-----------|-----------|-----|--------|-----------|--------|
| **Workflow governance** | ★★★★★ | ★★ | ★★★ | ★ | ★ |
| **Agent autonomy** | ★★★ | ★★★★★ | ★★★★ | ★★ | ★★ |
| **Multi-model routing** | ☆ | ★★★★★ | ★★ | ★ | ★ |
| **Edit tool quality** | ★★★ | ★★★★★ | ★★★ | ☆ | ☆ |
| **Document intelligence** | ★★★★★ | ★★ | ★★ | ☆ | ☆ |
| **Knowledge management** | ★★★★ | ★★★ | ★★ | ★ | ☆ |
| **Git integration** | ★★★★★ | ★★ | ★★★★ | ★ | ★★★ |
| **Human interface** | ★★★ (chat) | ★★★ (chat) | ★★★ (chat) | ★★ (chat) | ★★★★ (web) |
| **Session continuity** | ★★★★ | ★★★★ | ★★★★ | ★★★ | ★★ |
| **Parallel execution** | ★★★★ | ★★★★★ | ★★★★ | ★★★ | ★ |
| **Setup complexity** | Medium | Low (agent installs) | Low | Low | Low |
