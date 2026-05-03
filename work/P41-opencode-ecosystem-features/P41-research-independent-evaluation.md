| Field  | Value                         |
|--------|-------------------------------|
| Date   | 2026-05-03T09:55:50Z          |
| Status | Draft                         |
| Author | Research Agent                |

## Research Question

This research evaluates four projects in the OpenCode plugin ecosystem — **oh-my-openagent**, **micode**, **opencode-background-agents**, and **portal** — against kanbanzai's architecture and feature set. It identifies capabilities these projects implement better, features worth exploring, candidates for adoption, architectural fit assessments, areas where kanbanzai has structural advantages, and cases where the same problem is solved via different architectural choices. This research informs kanbanzai's Phase 4+ design decisions, extending the prior landscape review (`research-orchestration-landscape-2025.md`) which did not examine the OpenCode plugin ecosystem.

## Scope and Methodology

**In scope:** The four OpenCode ecosystem projects listed above, evaluated against kanbanzai's architecture. Extended thinking on CLI and direct API extensions to kanbanzai's MCP-tool-first model.

**Out of scope:** Other orchestration frameworks (covered in prior landscape review). Implementation design for any recommended features. Evaluation of the OpenCode base platform itself (only plugins/ecosystem projects built on it).

**Methodology:** README-level documentation review for all four projects. Detailed source-structure analysis via GitHub tree API. Deep code review of one representative source file per project (oh-my-openagent orchestration guide, opencode-background-agents plugin source, micode executor agent). Comparison against kanbanzai's documented architecture and internal implementation. Synthesis of findings across projects.

**Evidence grading:**
- **Code-reviewed:** Source code directly inspected
- **Documentation-reviewed:** Official docs/guides examined
- **README-claims:** Only project README available as evidence
- **Structure-analyzed:** Repository file tree examined for architectural patterns

---

## Findings

### Finding Set 1: What do they do better?

#### 1.1 oh-my-openagent: Multi-Model Orchestration with Category-Based Delegation

**Evidence basis:** Documentation-reviewed (orchestration guide), structure-analyzed (source tree)

oh-my-openagent implements a category-based delegation system where the orchestrator (Atlas/Sisyphus) delegates to sub-agents not by model name but by **semantic category** (`visual-engineering`, `ultrabrain`, `deep`, `quick`). Each category maps to a provider-optimised fallback chain (e.g., `ultrabrain` → gpt-5.4 → gemini-3.1-pro → claude-opus-4-7 → glm-5). The model resolution is transparent to the orchestrating agent — it says *what kind of work* needs doing, and the system resolves *which model* does it.

**kanbanzai gap:** kanbanzai's `spawn_agent` delegates to a role-defined agent but does not support category-based model routing or provider fallback chains. The orchestrator agent must explicitly choose which model to use for sub-agents (via role profiles). For a system that emphasises vocabulary routing as the primary quality lever, adding category-based dispatch could further strengthen the model-to-task matching.

**What oh-my-openagent does better:** (a) Transparent multi-provider model routing via semantic categories, (b) configurable fallback chains per category, (c) model-agnostic delegation where the orchestrator specifies *intent* not *implementation*.

#### 1.2 micode: Batch-First Parallelism with 10-20 Concurrent Agents

**Evidence basis:** Code-reviewed (executor agent source), structure-analyzed

micode's executor agent implements a distinctive batch-first parallelism pattern: it groups implementation tasks into batches, then spawns 10-20 implementer agents **simultaneously** in a single message, waits for all to complete, then spawns all reviewers simultaneously. This is explicitly designed for "micro-tasks" of 2-5 minutes each, achieving high throughput on granular work.

**kanbanzai gap:** kanbanzai's `orchestrate-development` skill caps at 4 concurrent sub-agents (DeepMind saturation point for specialist panels) and focuses on per-task dispatch with `handoff`. This is appropriate for substantive development tasks but not optimised for the "micro-task" granularity that micode targets. micode's pattern of 10-20 concurrent implementers is a fundamentally different throughput model.

**What micode does better:** (a) Extreme parallelism (10-20 agents per batch) on granular micro-tasks, (b) fire-and-wait batch execution where all implementers run simultaneously, (c) structured implement→review cycles within each batch.

#### 1.3 opencode-background-agents: Async Fire-and-Forget with Disk Persistence

**Evidence basis:** Code-reviewed (full plugin source, ~59k TypeScript), structure-analyzed

opencode-background-agents implements Claude Code-style async background delegation with a sophisticated lifecycle: `registered → running → terminal`. Key features verified in source: (a) Results persisted to disk as markdown files (`~/.local/share/opencode/delegations/`), (b) Stable delegation IDs with automatic title/description generation via a small model, (c) Terminal-state protection preventing late progress events from regressing terminal status, (d) `delegation_read(id)` blocks until terminal/timeout and returns deterministic terminal info, (e) Compaction-aware context injection carrying forward running and unread completed delegation context across compaction events.

**kanbanzai gap:** kanbanzai's `spawn_agent` is synchronous — the orchestrator blocks until the sub-agent completes. There is no fire-and-forget dispatch, no background result persistence independent of session context, and no mechanism for the orchestrator to continue productive work while sub-agents run. The Phase 4 design maps to synchronous dispatch; background dispatch is identified as a potential Phase 4b extension.

**What opencode-background-agents does better:** (a) True async fire-and-forget dispatch, (b) Disk-persisted results that survive context compaction and session restarts, (c) Automatic metadata generation (title/description) via small model for retrievability, (d) Compaction-aware context injection maintaining delegation awareness across sessions.

#### 1.4 portal: Web UI for Session Visibility

**Evidence basis:** README-claims, structure-analyzed

portal provides a mobile-first web UI for interacting with OpenCode instances, including session management, real-time chat, file mentions, model selection, and git integration. It connects to a running OpenCode server and provides a browser interface accessible from mobile devices.

**kanbanzai gap:** kanbanzai has no web UI. The prior landscape review (§8) identified this as a Phase 5 need: "visibility for designers and managers who are not running the CLI or using an AI agent interface." portal validates the demand for this pattern in the OpenCode ecosystem specifically.

**What portal does better:** (a) Provides a ready-to-use web dashboard for agent sessions, (b) Mobile-responsive design for remote access, (c) Multi-instance management CLI. These are features kanbanzai does not yet have.

---

### Finding Set 2: Interesting features worth exploring

#### 2.1 oh-my-openagent: Hash-Anchored Edit Tool (Hashline)

**Evidence basis:** Documentation-reviewed, README-claims

oh-my-openagent implements a line-hashing mechanism where every line the agent reads is tagged with a content hash (`11#VK| function hello() {`). The agent edits by referencing hash tags; if the file changed since the last read, the hash won't match and the edit is rejected before corruption. Claimed improvement from 6.7% to 68.3% success rate on Grok Code Fast 1 benchmark, just from changing the edit tool.

**Why interesting:** This is a *tool-level* improvement, not an architecture-level change. It addresses the "Harness Problem" (Can Bölük) where most agent failures originate in the edit tool, not the model. kanbanzai uses `edit_file` and `write_file` MCP tools that operate on `old_text`/`new_text` pattern matching. Adding hash-anchored verification would be a tool enhancement, not an architecture change.

**Assessment:** Novel application of content-addressable editing to agent tools. Code-reviewed at the level of the hashline test suite (dedicated test directory with headless tests and multi-model benchmarks). Claims about success rate improvement are README-claims only (not independently verified), but the mechanism is sound and testable.

#### 2.2 micode: Mindmodel System

**Evidence basis:** Code-reviewed (agent and model source files), structure-analyzed

micode has a `.mindmodel/` directory structure that defines project-specific patterns and conventions in a structured format (YAML manifest + markdown files for architecture layers, components, domain concepts, stack, style, patterns, ops). A suite of mindmodel agents (anti-pattern-detector, code-clusterer, constraint-reviewer, convention-extractor, dependency-mapper, domain-extractor, pattern-discoverer, stack-detector) analyses code and enforces these patterns. Mindmodel context is injected into agent sessions via a `mindmodel-injector` hook.

**Why interesting:** This is a structured approach to *project-specific code conventions* that sits between kanbanzai's knowledge graph (general architectural knowledge) and its role profiles (behavioural rules). It's a lower-level, code-pattern-oriented knowledge layer. This could complement kanbanzai's tiered knowledge system by providing a structured format for code conventions (naming, import patterns, error handling, testing).

**Assessment:** The mindmodel concept bridges the gap between "how to behave" (roles/skills) and "how to write code here" (project conventions). It's a more structured alternative to `.cursor/rules` or `.github/copilot-instructions.md`, with the key difference that it includes agents that *verify* conformance rather than just *documenting* conventions.

#### 2.3 oh-my-openagent: Wisdom Accumulation (Notepad System)

**Evidence basis:** Documentation-reviewed (orchestration guide)

After each sub-agent task completes, Atlas extracts learnings and categorises them into: Conventions, Successes, Failures, Gotchas, Commands. These are written to `.sisyphus/notepads/{plan-name}/` as structured markdown files and passed forward to all subsequent sub-agents.

**Why interesting:** This is essentially a *session-scoped knowledge accumulation* mechanism, distinct from kanbanzai's tiered knowledge system (which is role/project-scoped). Where kanbanzai has tiered knowledge with lifecycle management (contribute → confirm → stale → retire), oh-my-openagent has a lighter-weight, plan-scoped notepad that captures immediate tactical learnings during a single orchestration session.

**Assessment:** This pattern addresses the "cumulative learning within a plan execution" problem. kanbanzai's Phase 6 close-out includes knowledge curation (confirm/flag/retire tier 2 entries), but there's no mechanism for intra-plan knowledge sharing between sub-agents. The notepad system is a lightweight complement.

#### 2.4 oh-my-openagent: Ralph Loop — Self-Referential Agent Loop

**Evidence basis:** Documentation-reviewed, structure-analyzed

The Ralph Loop (`ulw` / `ultrawork`) is a self-referential loop where the agent continues working on a task until completion is verified, without stopping to ask the user. It combines: todo enforcement (agent goes idle → system yanks it back), completion promise detection, oracle verification, and auto-continuation across context limits.

**Why interesting:** This is a different orchestration philosophy from kanbanzai's structured `orchestrator-workers` pattern. Where kanbanzai has the orchestrator explicitly manage dispatch → monitor → close-out, the Ralph Loop is an *autonomous persistence loop* where the agent self-drives. This maps to the "discipline agent" concept — the system ensures the agent doesn't stop until the task is actually done.

**Assessment:** The Ralph Loop pattern could complement kanbanzai's orchestration model for cases where a single agent needs to persist through a complex multi-step task without orchestration overhead. However, it trades structured oversight for autonomous persistence, which may not align with kanbanzai's emphasis on review gates and verification.

#### 2.5 micode: Session Continuity via Ledger System

**Evidence basis:** Documentation-reviewed, structure-analyzed

micode implements a "ledger" system: `/ledger` creates `thoughts/ledgers/CONTINUITY_{session}.md` files that compact session history into a structured continuity document. The ledger is automatically injected into new sessions via a hook. This provides structured context recovery across sessions.

**Why interesting:** kanbanzai handles context compaction via the `orchestrate-development` skill's Phase 5 (post-completion summarisation, document-based offloading at 60% utilisation, single-feature scoping). micode's ledger is a more formalised, structured approach to the same problem — it creates a dedicated continuity artifact rather than summarising into a progress document.

**Assessment:** The ledger pattern could strengthen kanbanzai's document-based offloading by providing a structured format for continuity documents, but kanbanzai's existing approach (progress document + fresh session) already addresses the same fundamental need.

---

### Finding Set 3: Candidates for kanbanzai adoption

#### 3.1 Category-Based Model Routing for Sub-Agent Dispatch

**Recommendation:** Consider adding a `category` parameter to `spawn_agent` (or a new `dispatch_task` MCP tool) that maps to provider/model fallback chains, similar to oh-my-openagent's category system. The orchestrator specifies *intent* (e.g., "deep-reasoning", "quick-fix", "visual-design"), and kanbanzai resolves the model.

**Confidence:** Medium.

**Based on:** Finding 1.1 (oh-my-openagent category system), Finding 2.1 (vocabulary routing research). Aligns with kanbanzai's existing emphasis on vocabulary as the primary quality lever — categories are a form of vocabulary routing applied to model selection.

**Architectural cost:** Low. This is a configuration layer on top of the existing `spawn_agent` / `handoff` pipeline. No entity model changes needed. Requires: (a) a category-to-model mapping in configuration, (b) optional `category` parameter on handoff/spawn, (c) model resolution logic that kanbanzai can call before dispatch.

**Priority:** Later — after Phase 4a synchronous dispatch is stable.

**Conditions:** This assumes multi-model usage is a kanbanzai use case. If kanbanzai users primarily use a single model/provider, category-based routing adds configuration complexity without corresponding value.

#### 3.2 Hash-Anchored Edit Verification

**Recommendation:** Explore adding content-hash verification to kanbanzai's `edit_file` MCP tool, inspired by oh-my-openagent's Hashline. When an agent reads a file, tag lines with hashes. When it submits an edit, verify hashes match before applying.

**Confidence:** Medium.

**Based on:** Finding 2.1 (Hashline mechanism). The "Harness Problem" is a well-documented failure mode across all agent tools, not specific to any platform. A hash-anchored edit tool is a tool-level improvement that benefits any MCP-based agent system.

**Architectural cost:** Low. This is an enhancement to the `edit_file` and `read_file` MCP tools. It does not change the entity model, workflow stages, or orchestration patterns. It may require changes to the `edit_file` tool schema to accept hash references instead of text patterns, and to `read_file` to optionally tag lines with hashes.

**Priority:** Soon — this is a tool enhancement with a clear failure mode it addresses.

**Conditions:** The implementation complexity of hash-anchored editing (generating hashes, validating on edit, handling hash collisions, backward compatibility) needs feasibility assessment. The 10x improvement claim (6.7% → 68.3%) is unverified and should not drive the decision.

#### 3.3 Async Background Dispatch with Result Persistence

**Recommendation:** Explore background agent dispatch as a Phase 4b extension, after Phase 4a delivers synchronous dispatch.

**Confidence:** Medium.

**Based on:** Finding 1.3 (opencode-background-agents implementation). The code-reviewed implementation demonstrates a mature pattern for async dispatch with lifecycle management, disk persistence, and compaction-aware context injection.

**Architectural cost:** Medium. Requires: (a) an asynchronous dispatch mechanism outside the MCP request/response cycle, (b) a result persistence layer (files or database), (c) a status-polling pattern or notification mechanism for the orchestrator, (d) the `dispatch_task` MCP tool would need an async variant. None of these require breaking the entity hierarchy or document graph.

**Priority:** Soon — after Phase 4a is stable.

**Conditions:** This assumes long-running agent tasks are a real bottleneck. If typical kanbanzai task execution is fast enough that synchronous dispatch is not a problem, background dispatch may be unnecessary complexity. The opencode-background-agents 15-minute timeout is also worth noting — background dispatch works best for bounded-duration tasks.

#### 3.4 Structured Project Convention System (Mindmodel-Inspired)

**Recommendation:** Consider adding a structured project convention layer to kanbanzai's knowledge system, inspired by micode's mindmodel. This would provide a standard format for code conventions (naming, patterns, error handling, testing) that agents can query and that verification tools can enforce.

**Confidence:** Low.

**Based on:** Finding 2.2 (micode mindmodel). The concept is architecturally interesting but the gap it fills is unclear — kanbanzai already has `AGENTS.md`, tiered knowledge, and role profiles that can encode conventions.

**Architectural cost:** Low to Medium. This would be a new knowledge tier or document type. It fits within the existing knowledge/document architecture but adds a new concern (code-level conventions) that currently sits partially in `refs/` files and partially in role profiles.

**Priority:** Watch — revisit if convention drift becomes a documented problem in kanbanzai usage.

**Conditions:** Unclear whether project conventions are better handled through existing mechanisms (rules injection, knowledge entries, AGENTS.md). The mindmodel approach of *verifying* conventions via dedicated agents is the distinguishing feature, not just *documenting* them.

#### 3.5 Web Dashboard for Entity Visibility

**Recommendation:** Flag a web UI as a Phase 5 candidate, informed by portal's implementation and the prior landscape review's §8 analysis.

**Confidence:** High.

**Based on:** Finding 1.4 (portal), prior landscape review §8. The need for non-CLI visibility into entity status, documents, and progress is validated across multiple sources.

**Architectural cost:** Medium. Phase 4 API design should avoid coupling query responses to agent-specific conventions that a human dashboard wouldn't need, as noted in the prior landscape review. The MCP tools already provide the data layer; a web UI is a presentation layer on top.

**Priority:** Later — Phase 5, after orchestration is stable.

---

### Finding Set 4: What kanbanzai does better

#### 4.1 Entity Hierarchy as Context Scoping Instrument

**Evidence basis:** Prior landscape review §5.1

kanbanzai's typed entity graph (Plan → Batch → Feature → Task/Bug → Decision) is used as a structural traversal key for automatic context assembly. When dispatching a task, `context_assemble` walks the entity hierarchy, traces document sections via `doc_intel`, and injects relevant design context automatically. This is kanbanzai's most distinctive structural feature.

**No OpenCode ecosystem project has anything comparable.** oh-my-openagent has plans (`.sisyphus/plans/`) and a boulder state tracker, but these are flat files with no typed entity graph, no lifecycle state machines, and no automatic context scoping via entity traversal. micode has artifact indexing for plans and ledgers but no entity hierarchy. opencode-background-agents and portal have no entity model at all.

**Structural advantage:** kanbanzai's entity hierarchy is a *context scoping instrument* — it automatically determines what context to inject based on entity relationships. The OpenCode projects inject context via hooks and file-based conventions (AGENTS.md injection, mindmodel injection, ledger injection), which require explicit paths and lack the automatic traversal that entity relationships provide.

#### 4.2 Document Intelligence Pipeline

**Evidence basis:** Prior landscape review §5.2

kanbanzai's three-layer document index (structural, metadata, semantic) with role classification and concept tagging enables `doc_trace(entity_id)`, `doc_find_by_concept`, `doc_find_by_role`, and `doc_impact(section_id)`. This pipeline automatically injects relevant design fragments into dispatch packets.

**No OpenCode ecosystem project has document intelligence.** oh-my-openagent has plans in markdown but no structured index over them. micode indexes artifacts (plans, ledgers) in a SQLite database with FTS, which is the closest analogue but lacks semantic role classification and concept tagging.

**Structural advantage:** kanbanzai's document graph is queryable by entity, concept, and role — it can answer "what design decisions affect this feature?" automatically. The OpenCode projects answer this through agent-driven file reading, which is less reliable and consumes context budget.

#### 4.3 Tiered Knowledge with Lifecycle Management

**Evidence basis:** Prior landscape review §5.3

kanbanzai's three-tier knowledge system with confidence scoring, git-staleness detection, and lifecycle management (contribute → confirm → stale → retire) is unique. Knowledge entries are scoped to roles and projects, surfaced in context assembly, and automatically flagged stale when their git anchors change.

**No OpenCode ecosystem project has anything comparable.** oh-my-openagent's notepad system is plan-scoped, ephemeral, and lacks lifecycle management. micode's mindmodel is project-scoped but static (no staleness detection, no confidence scoring).

**Structural advantage:** kanbanzai's knowledge lifecycle is automated — entries are contributed during task completion, confirmed during close-out, and automatically go stale when code changes. This creates a self-maintaining knowledge base that the OpenCode projects' static files cannot match.

#### 4.4 Byte-Budgeted, Priority-Trimmed Context Assembly

**Evidence basis:** Prior landscape review §5.4

`context_assemble` implements a deterministic assembly order with byte-budget trimming. Bytes are the universal unit (avoiding server-side tokenisation). Trimming proceeds: Tier 3 entries first (lowest confidence), then Tier 2, then design context from the tail.

**No OpenCode ecosystem project implements byte-budgeted context assembly.** oh-my-openagent has context window monitoring and preemptive compaction, but this is reactive (triggered by context limits) rather than proactive (budgeted at assembly time). micode has auto-compaction at 50% usage and token-aware truncation, but again these are reactive. opencode-background-agents saves context by offloading heavy research to background sessions, but the orchestrator's own context assembly is not budgeted.

**Structural advantage:** kanbanzai's approach is principled and *deterministic* — the assembly order and trimming priority are fixed, so agents know what was cut and why. The OpenCode projects' reactive approaches are heuristic and context-dependent.

#### 4.5 Semantic Merge Gates

**Evidence basis:** Prior landscape review §5.5

kanbanzai's merge readiness checks operate at the level of workflow ontology: are all tasks done? Does a verification record exist? Are design documents current? Is the branch not stale?

**No OpenCode ecosystem project has semantic merge gates.** These projects are editor plugins — they don't have merge logic at all (that's delegated to the user or to git).

#### 4.6 Role + Skill System as a Formal Protocol

**Evidence basis:** `.kbz/stage-bindings.yaml`, `.kbz/roles/`, `.kbz/skills/`

kanbanzai's role + skill system is a formal, machine-readable protocol with inheritance, vocabulary constraints, anti-pattern detection, tool scoping, and checklist-driven procedures. Stage bindings map workflow stages to specific roles, skills, and prerequisites.

**oh-my-openagent has the closest analogue** with its agent prompt system — dedicated agent definitions with model-specific prompt variants, system reminders, and tool restrictions. But these are prompt engineering artifacts (TypeScript functions that generate system prompts), not a formal protocol with inheritance, vocabulary constraints, and stage binding. micode has agent definitions but they are simpler and less structured than kanbanzai's role/skill system.

**Structural advantage:** kanbanzai's role + skill system is designed to be *readable by both humans and agents*. The YAML format with explicit inheritance and vocabulary sections is a machine-readable protocol, not just a prompt template. The OpenCode projects' agent definitions are prompt templates — they work but lack the formal structure.

---

### Finding Set 5: Same problem, different approach

#### 5.1 Task Decomposition: `decompose` tool vs. Prometheus Interview + Metis Review

**kanbanzai approach:** The `decompose` tool proposes a task breakdown from a feature's specification. The `decompose-feature` skill guides the architect through the breakdown. Decomposition is a formal stage gate with prerequisites (approved spec, approved dev-plan).

**oh-my-openagent approach:** Prometheus conducts an interactive interview with the user, consulting Metis for gap analysis. Plans are generated as markdown files. For high-accuracy mode, Momus reviews plans against clarity/verification/context/big-picture criteria.

**Trade-off analysis:** kanbanzai's approach is more formal and machine-enforceable — decomposition is a structured tool call with a typed output, and the resulting tasks have lifecycle state machines. oh-my-openagent's approach is more conversational and human-centric — the interview process surfaces ambiguities before plan generation. The interview pattern could strengthen kanbanzai's decomposition by catching ambiguities earlier, but the formal decomposition with typed outputs is more suitable for automated dispatch and dependency tracking.

#### 5.2 Parallel Agent Dispatch: `spawn_agent` + `handoff` vs. Batch-First `spawn_agent`

**kanbanzai approach:** The orchestrator identifies the ready frontier (tasks with met dependencies), checks for file conflicts, uses `handoff` to assemble scoped context packets, and dispatches sub-agents in parallel batches. Max 4 concurrent (DeepMind saturation point).

**micode approach:** The executor fires all implementers for a batch in a single message (10-20 simultaneous), waits for all to complete, then fires all reviewers. This is designed for micro-tasks of 2-5 minutes each.

**Trade-off analysis:** kanbanzai's approach is optimised for substantive development tasks where each task involves significant work and sub-agents benefit from carefully assembled context. micode's approach is optimised for throughput on granular, well-defined micro-tasks where context assembly overhead per task would dominate. These are different problem domains — kanbanzai's pattern is correct for its use case, but the micro-task pattern is worth noting for cases where features decompose into many small, independent implementation units.

#### 5.3 Context Compaction: Document-Based Offloading vs. Auto-Compaction + Ledger

**kanbanzai approach:** At 60% context utilisation, stop dispatching, write a progress document, and start a fresh session. Post-completion summarisation reduces completed task outputs to 2-3 sentences.

**micode approach:** Auto-compaction at 50% context usage, summarising the session automatically. The ledger system creates structured continuity documents for cross-session context recovery.

**Trade-off analysis:** kanbanzai's approach gives the orchestrator explicit control over when and how to compact — it's a conscious decision point. micode's approach is automatic and less disruptive to the workflow flow. The trade-off is between control (kanbanzai) and convenience (micode). The ledger pattern adds structure to continuity documents that kanbanzai's progress documents could benefit from, though the fundamental approaches are compatible.

#### 5.4 Sub-Agent Isolation: Worktrees vs. Shared Directory

**kanbanzai approach:** Each feature gets its own git worktree via `worktree(action: create)`. Parallel features have isolated working directories and branches. Conflict detection analyses file overlap before parallel dispatch.

**OpenCode ecosystem approach:** Agents share the project directory. Isolation comes from the OpenCode session model (each sub-agent gets its own session), but file-level conflicts are managed through the agent's own coordination, not structural isolation.

**Trade-off analysis:** kanbanzai's worktree isolation is stronger — parallel features cannot interfere at the filesystem level. The cost is worktree management overhead (create, merge, cleanup). The OpenCode approach is simpler but risks file conflicts that must be resolved by agents or the user. For kanbanzai's use case of parallel feature development, worktree isolation is the correct choice. For OpenCode's use case of rapid sequential development within a session, shared directory access is more natural.

---

### Finding Set 6: Architectural deltas

#### 6.1 What would it take to add category-based model routing to kanbanzai?

**Current architecture:** kanbanzai's `spawn_agent` uses role profiles to determine agent identity and behaviour. The role defines which tools the agent uses, its vocabulary, and its anti-patterns. Model selection is not part of the role — it's determined by the MCP client (Claude, etc.) that the orchestrator is running in.

**What would need to change:**
1. Add a `category` parameter to `handoff` and `spawn_agent` that maps to model/provider preferences.
2. Add a category-to-model configuration in kanbanzai's config (e.g., `.kbz/local.yaml` or a new config file).
3. If kanbanzai is dispatching agents directly to AI providers (not through an MCP client), implement model resolution and fallback chain logic.
4. This is a configuration + dispatch enhancement — it does not require entity model changes, document graph changes, or knowledge system changes.

**Assessment:** Low architectural cost. This is additive. The question is whether it's needed — kanbanzai currently runs within a single MCP client context, so model selection is the client's concern, not kanbanzai's. This becomes relevant only if kanbanzai dispatches agents directly to providers.

#### 6.2 What would it take to add async background dispatch to kanbanzai?

**Current architecture:** `spawn_agent` is synchronous — the MCP tool call blocks until the sub-agent completes and returns its result. The orchestrator's loop (dispatch → monitor → close-out) assumes synchronous completion.

**What would need to change:**
1. Add an async variant of `dispatch_task` (or a new `background_task` MCP tool) that returns immediately with a task ID.
2. Add a result persistence layer — store completed agent outputs in a file or database with structured metadata (title, description, status).
3. Add a status-polling mechanism (`task_status(id)` MCP tool) or notification pattern for the orchestrator.
4. Add a `task_result(id)` MCP tool that returns the persisted result.
5. Modify the orchestrator skill to handle the async pattern: dispatch, continue other work, check/poll for results, retrieve when ready.
6. Handle timeout and cancellation (opencode-background-agents uses 15-minute timeout with session deletion).
7. Handle compaction — running and unread completed tasks must survive context compaction.

**Assessment:** Medium architectural cost. This extends beyond the current synchronous request/response model but does not require breaking the entity hierarchy, document graph, or knowledge system. The key architectural question is: does kanbanzai need to break out of the MCP request/response cycle for dispatch? If dispatch goes through an MCP tool that returns immediately (fire-and-forget), then results come back through a different MCP tool (poll/retrieve), the dispatcher and retriever can be the same agent in the same session — no daemon required. This is architecturally consistent with kanbanzai's MCP-tool-first design.

#### 6.3 What would it take to add a web UI to kanbanzai?

**Current architecture:** kanbanzai is a CLI tool + MCP server. All interaction is through `kbz` commands or MCP tool invocations. There is no HTTP API for UI consumption (the MCP protocol is the API).

**What would need to change:**
1. Add a read-oriented HTTP API layer (or expose MCP tools over HTTP/SSE, which the MCP protocol supports).
2. Build a web frontend that consumes this API.
3. The web UI would need: entity listing/filtering, document viewing, progress dashboards, branch/worktree status, knowledge browsing.
4. The prior landscape review (§8) notes that Phase 4 API design should avoid coupling query responses to agent-specific conventions. This constraint still holds.

**Assessment:** Medium architectural cost. The data layer already exists (entity state, document graph, knowledge, health checks). The MCP protocol supports HTTP transport. This is primarily a presentation-layer addition, not a data-model change. Priority is Phase 5 per the prior landscape review.

#### 6.4 What would it take to enable kanbanzai to launch agents directly to AI providers?

**Current architecture:** kanbanzai's MCP server provides tools that an AI agent (Claude, etc.) calls in a conversation loop. kanbanzai does not directly call AI provider APIs — the agent does that through its own client.

**What would need to change:**
1. kanbanzai would need an agent runtime — the ability to create agent sessions, send prompts, and receive responses from provider APIs (Claude API, DeepSeek API, etc.).
2. This is a fundamental architectural expansion: kanbanzai goes from being an MCP tool server (called *by* agents) to being a dispatch platform (calling *out to* agents).
3. This would require: provider API integration, session management, token tracking, error handling, fallback chains, etc.
4. The orchestrator would still be an agent calling kanbanzai's MCP tools, but when it calls `dispatch_task`, kanbanzai would create a new agent session with a provider API rather than the orchestrator's MCP client spawning a sub-agent.

**Assessment:** High architectural cost. This is a major expansion of kanbanzai's scope. The prior landscape review (§6) argues that "kanbanzai provides the knowledge and context; the orchestrator is Claude (or any model) calling kanbanzai's MCP tools in a loop." Adding direct provider dispatch would change kanbanzai from a knowledge+context layer to a full orchestration runtime. This is architecturally possible but represents a significant scope expansion with corresponding maintenance burden.

---

## Trade-Off Analysis

| Criterion | OpenCode ecosystem pattern | kanbanzai current approach | Assessment |
|-----------|---------------------------|---------------------------|------------|
| **Orchestration model** | Agent-driven loop within OpenCode session (oh-my-openagent: Atlas/Sisyphus loop) or batch-fire parallelism (micode: fire all, wait all) | MCP-tool-driven orchestration where the orchestrating agent calls kanbanzai tools in a structured loop with explicit stage gates | kanbanzai's approach is more formal and auditable; OpenCode's is more fluid and conversational. Both work. kanbanzai's stage gates add safety at the cost of ceremony. |
| **Context assembly** | Hook-based injection (AGENTS.md, mindmodel, ledgers, rules) at session start or on compaction | Deterministic, priority-trimmed assembly from entity hierarchy, document graph, and knowledge tiers at dispatch time | kanbanzai's approach is more principled and budget-aware. OpenCode's approach is simpler but less controlled. |
| **Sub-agent isolation** | Session-level isolation within OpenCode (each sub-agent gets a session, shares filesystem) | Git worktree isolation (each feature gets its own working directory and branch) | kanbanzai's worktree isolation is stronger for parallel feature development. OpenCode's session isolation is adequate for sequential development within a session. |
| **Knowledge management** | Static files (AGENTS.md, mindmodel, notepads) — no lifecycle, no staleness detection | Tiered knowledge with confidence scoring, git-staleness detection, and lifecycle management (contribute → confirm → stale → retire) | kanbanzai's knowledge system is structurally superior. Static files require manual maintenance; kanbanzai's lifecycle is automated. |
| **Model routing** | Category-based semantic routing with configurable fallback chains (oh-my-openagent) | Role-based agent definition with MCP client model selection | oh-my-openagent's category system is more flexible for multi-model environments. kanbanzai's role system is simpler but assumes a single model context. |
| **Parallelism ceiling** | 10-20 concurrent agents per batch (micode) or unbounded background agents (oh-my-openagent) | 4 concurrent sub-agents (DeepMind saturation point), synchronous dispatch | micode's extreme parallelism is appropriate for micro-tasks; kanbanzai's cap is appropriate for substantive tasks. Different problem domains. |
| **Document intelligence** | None — plans and artifacts are flat markdown files | Three-layer index (structural, metadata, semantic) with role classification, concept tagging, and entity tracing | kanbanzai's document intelligence is unique. OpenCode projects have no equivalent. |
| **Merge safety** | Not applicable — editor plugins don't manage merges | Semantic merge gates that check workflow state (all tasks done? verification record exists? documents current? branch not stale?) | kanbanzai's merge gates are unique to workflow-aware systems. |
| **CLI interface** | OpenCode TUI + commands (oh-my-openagent: `ulw`, `ultrawork`, `/start-work`, `/init-deep`) | `kbz` CLI + MCP tools (entity, status, health, etc.) | OpenCode's TUI is richer for interactive development. kanbanzai's CLI + MCP tools are more structured for workflow management. Different audiences. |
| **Web UI** | portal provides mobile-first web dashboard for OpenCode sessions | None (Phase 5 candidate) | portal validates the demand for web visibility. kanbanzai should build this but not before orchestration is stable. |

---

## Recommendations

### Recommendation 1: Hash-Anchored Edit Verification

- **What:** Enhance `edit_file` and `read_file` MCP tools with content-hash verification, inspired by oh-my-openagent's Hashline.
- **Confidence:** Medium
- **Based on:** Finding 2.1, Finding Set 1 (oh-my-openagent tool quality)
- **Architectural cost:** Low — tool-level enhancement
- **Priority:** Soon

### Recommendation 2: Async Background Dispatch (Phase 4b)

- **What:** Add fire-and-forget dispatch with result persistence, inspired by opencode-background-agents.
- **Confidence:** Medium
- **Based on:** Finding 1.3, Finding 3.3
- **Architectural cost:** Medium — extends beyond synchronous request/response but stays within MCP-tool-first model
- **Priority:** Soon — after Phase 4a synchronous dispatch is stable

### Recommendation 3: Category-Based Model Routing (Phase 4b+)

- **What:** Add optional `category` parameter to sub-agent dispatch that maps to model/provider fallback chains.
- **Confidence:** Medium
- **Based on:** Finding 1.1, Finding 3.1
- **Architectural cost:** Low — configuration + dispatch enhancement
- **Priority:** Later — only if multi-model dispatch becomes a kanbanzai use case

### Recommendation 4: Web Dashboard (Phase 5)

- **What:** Build a read-oriented web UI for entity visibility, document access, and progress dashboards.
- **Confidence:** High
- **Based on:** Finding 1.4, prior landscape review §8
- **Architectural cost:** Medium — presentation layer on existing data
- **Priority:** Later — Phase 5, after orchestration is stable

### Recommendation 5: Structured Project Convention Layer

- **What:** Monitor micode's mindmodel pattern; revisit if convention drift becomes a documented problem.
- **Confidence:** Low
- **Based on:** Finding 2.2, Finding 3.4
- **Architectural cost:** Low to Medium — new document type or knowledge tier
- **Priority:** Watch

### Recommendation 6: Intra-Plan Knowledge Accumulation

- **What:** Explore a lightweight, plan-scoped knowledge sharing mechanism (inspired by oh-my-openagent's notepad system) for passing tactical learnings between sub-agents within a single orchestration session.
- **Confidence:** Low
- **Based on:** Finding 2.3
- **Architectural cost:** Low — could be implemented as structured output from `finish` with forwarding via `handoff`
- **Priority:** Watch

---

## Limitations

1. **Evidence quality varies by project.** oh-my-openagent was the most thoroughly reviewed (documentation + source structure), opencode-background-agents received full code review of its plugin source, micode received partial code review (executor agent + structure), and portal was reviewed at README + structure level only. Findings about portal should be treated as claims-based, not code-reviewed.

2. **No runtime testing.** None of these projects were installed or tested. The analysis is based on static review of documentation, source code structure, and (for one project) full plugin source code. Claims about runtime behaviour (e.g., Ralph Loop completion rates, Hashline success rate improvements, micro-task throughput) are taken from project documentation and not independently verified.

3. **OpenCode platform dependency.** All four projects are plugins for OpenCode (sst/opencode), a specific AI coding agent platform. Their architectures assume OpenCode's session model, tool system, hook system, and plugin API. Features that appear "better" may be enabled by OpenCode platform capabilities that kanbanzai would need to replicate or replace.

4. **Rapidly evolving ecosystem.** oh-my-openagent has 5,421 commits and 170 releases as of this writing. The OpenCode plugin ecosystem is moving fast. Findings may be outdated quickly. Recommendations should be treated as directional, not prescriptive.

5. **Single-assessor bias.** This research was conducted by a single analyst (Research Agent role). Findings have not been cross-validated by other reviewers or against independent sources beyond the prior landscape review.

6. **Scope limitations.** This research focused on the four specified projects. It did not re-evaluate the orchestration frameworks covered in the prior landscape review (mcp-agent, kagan, Agent-MCP, etc.). Some features identified as "kanbanzai gaps" here may be present in those frameworks and already analysed in the prior review.

7. **Assumptions underpinning the analysis:**
   - kanbanzai continues to operate as an MCP server with tools called by an AI agent in a conversation loop.
   - The entity hierarchy and document graph remain the foundation for context assembly.
   - Worktree-based isolation remains the model for parallel feature development.
   - Changes to these assumptions would require re-evaluation of all findings and recommendations.

8. **Conditions that could change conclusions:**
   - If kanbanzai adds direct AI provider dispatch, category-based model routing (Recommendation 3) becomes higher priority.
   - If kanbanzai users primarily work on small, independent micro-tasks rather than substantive features, micode's batch-first parallelism pattern becomes more relevant.
   - If the OpenCode platform becomes kanbanzai's primary agent client (rather than generic MCP clients), tighter integration with OpenCode's plugin system could unlock features currently assessed as "high architectural cost."
