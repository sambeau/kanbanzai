# Orchestration Landscape Review: Phase 4 Planning Research

| Document | Orchestration Landscape Review |
|----------|-------------------------------|
| Status   | research note                 |
| Date     | 2026-03-25                    |
| Author   | human + AI collaborative review |
| Related  | `work/design/workflow-design-basis.md` §14, §24 |
|          | `work/design/machine-context-design.md` §7, §8, §15 |
|          | `work/plan/phase-4-scope.md` |

---

## 1. Purpose

Before committing to Phase 4 implementation, this review assesses whether kanbanzai should build its own orchestration layer or incorporate an existing tool. It surveys the current state of AI agent orchestration tools as of mid-2025, identifies what kanbanzai already has that is architecturally unique, extracts design patterns the field has validated that should inform Phase 4, and flags new scope items not present in the original design documents.

The original Phase 4 design was written when orchestration tooling was immature. This review checks whether that has changed and whether the plan remains correct.

---

## 2. Research Motivation

Concerns entering this review:

1. The existing design documents describe Phase 4 in seven words per bullet. The orchestration space has moved quickly since they were written.
2. Several open-source tools now combine task tracking, agent dispatch, and git integration. If any of them already does what kanbanzai needs, building a custom solution would waste effort.
3. A parallel search had already flagged **kagan** as a potentially relevant tool. This review examines it in detail.
4. There is a separate future interest in a **web/desktop UI** for designers and managers. kagan has this. It warrants scoping as a future track.

---

## 3. The Orchestration Landscape in 2025

### 3.1 Major Frameworks and SDKs

#### `mcp-agent` (lastmile-ai) — ⭐ 8.1k
**The most MCP-native orchestration framework.**

Built explicitly around MCP as its foundational protocol. Implements every pattern from Anthropic's "Building Effective Agents" paper: orchestrator-workers, parallel/map-reduce, router, evaluator-optimizer, swarm. Supports Temporal for durable execution (pause/resume across crashes, human-in-the-loop signals). Has a `TokenCounter` with threshold callbacks.

- **Persistent workflow state:** ✅ Dual-mode: asyncio for simple cases, Temporal for production.
- **Document/knowledge management:** ❌ None. Wire in external MCP servers.
- **Context budgeting:** ✅ `TokenCounter` with selective tool loading.
- **Git/worktrees:** ❌ None.
- **Multi-session dispatch:** ✅ Core design. `create_orchestrator()` spawns and coordinates workers.
- **Verdict:** Best choice for *building* an MCP-first system. It is a library, not a product — you compose it yourself.

#### `openai-agents-python` (OpenAI Agents SDK) — ⭐ 20.3k
**Best MCP client integration among the major SDKs.**

Provider-agnostic Python SDK. Core concepts: Agents, Handoffs, Guardrails, Sessions. Four transport modes for MCP servers. `MCPServerManager` for connection pooling. `HostedMCPTool` delegates execution to OpenAI's infrastructure.

- **Persistent workflow state:** ⚠️ Redis-backed sessions, optional. No native Temporal.
- **Document/knowledge management:** ❌ None.
- **Context budgeting:** ✅ Tool filtering and strict schema conversion.
- **Git/worktrees:** ❌ None.
- **Multi-session dispatch:** ✅ Agents-as-tools and explicit handoffs.
- **Verdict:** Clean, production-quality MCP integration. Best if your primary model is OpenAI. Provides no workflow semantics.

#### LangGraph (langchain-ai) — ⭐ 27.5k
**Best for durable stateful graph-based workflows.**

Low-level orchestration framework for long-running agents. Graph/node architecture with state checkpointing. Human-in-the-loop interruption and resume. Not MCP-native but LangChain has MCP adapters.

- **Persistent workflow state:** ✅ Core strength. Crash recovery, checkpointing.
- **Document/knowledge management:** ❌ None built-in.
- **Context budgeting:** ⚠️ Build it into your graph nodes.
- **Git/worktrees:** ❌ None.
- **Multi-session dispatch:** ✅ Subgraphs and parallel node execution.
- **Verdict:** The most mature durability story. Graph-first mental model, not MCP-first. No workflow ontology.

#### AutoGen (Microsoft) — ⭐ 56.2k
**Most enterprise-oriented multi-agent framework.**

Layered APIs (Core, AgentChat, Extensions). `McpWorkbench` in `autogen_ext.tools.mcp` connects to MCP servers. AutoGen Studio adds a no-code GUI.

- **Persistent workflow state:** ⚠️ Lives in conversation history. Complex persistence requires custom implementation.
- **Document/knowledge management:** ❌ None.
- **Context budgeting:** ❌ Not handled.
- **Git/worktrees:** ❌ None.
- **Multi-session dispatch:** ✅ `AgentTool` wraps sub-agents as callable tools.
- **Verdict:** Strongest ecosystem. Being repositioned as "Microsoft Agent Framework." Good for Azure-oriented teams. Provides no workflow semantics.

#### CrewAI — ⭐ 47.2k
**Best for role-based autonomous agent teams.**

Role-based framework with two modes: Crews (autonomous, collaborative) and Flows (event-driven, precise). Completely standalone. No native MCP interface — you write custom tool wrappers.

- **Persistent workflow state:** ✅ Flows have Pydantic state models.
- **Document/knowledge management:** ❌ None.
- **Context budgeting:** ❌ Not handled.
- **Git/worktrees:** ❌ None.
- **Multi-session dispatch:** ✅ Flows orchestrate sequential/parallel Crew execution.
- **Verdict:** Excellent for autonomous role-based teams. Poor fit for MCP-first workflows — you fight the architecture. Not relevant for kanbanzai.

### 3.2 Dedicated Orchestration MCP Servers

These are MCP *servers* that expose orchestration capabilities as MCP tools — callable directly from a Claude conversation.

#### `Agent-MCP` (rinadelph) — ⭐ 1.2k
**The closest thing to task tracking + agent dispatch + context assembly in one MCP server.**

Described as "Obsidian for your AI agents" — a living knowledge graph where agents collaborate. SQLite-backed. Exposes: `create_agent`, `assign_task`, `update_task_status`, `ask_project_rag`, `update_project_context`, `send_agent_message`.

- **Persistent workflow state:** ✅ SQLite-backed task states persist across sessions.
- **Document/knowledge management:** ✅ RAG over project docs, architectural decisions, implementation notes. OpenAI embeddings.
- **Context budgeting:** ✅ Ephemeral agent model — each worker gets minimal context, queries shared RAG for what it needs. Hard limit of 10 concurrent agents.
- **Git/worktrees:** ❌ File-level locking to prevent conflicts, but not git worktrees.
- **Multi-session dispatch:** ✅ Core design. Short-lived agents, each scoped to a single task.
- **Architectural insight:** Agents should be short-lived with minimal upfront context, querying the shared knowledge graph on demand. This is a validated pattern.
- **Caveats:** Small community (1.2k stars). AGPL license. Described as having a steep learning curve.

#### Network-AI (Jovancoding)
**Race-condition-safe multi-agent blackboard as an MCP server.**

20+ tools including: blackboard read/write, agent spawn/stop, FSM (finite state machine) transitions, budget tracking, token management, audit log query.

- **Persistent workflow state:** ✅ Blackboard = shared state. FSM = explicit state machine.
- **Context budgeting:** ✅ Budget tracking and token management are explicit features.
- **Multi-session dispatch:** ✅ Agent spawn/stop.
- **Architectural insight:** FSM + blackboard is a clean pattern. Budget tracking must be a first-class feature, not bolted on later.

#### `claude-task-master` (eyaltoledano)
PRD parsing → task breakdown → multi-provider dispatch. Selective tool loading for context optimisation. Well-adopted in the "vibe coding" community.

#### `mcp-shrimp-task-manager` (cjo4m06)
Task memory, self-reflection, dependency management for coding agents. Interesting dependency tracking model.

#### `dagger/container-use`
Docker container isolation per agent + git branch isolation. No conflicts, parallel agent experiments. The container-per-agent model is the Docker analogue of kanbanzai's worktree-per-feature model.

#### `context-rot-detection`
Context health monitoring — token utilisation, degradation signals, recovery recommendations. Validates that context quality decay is a real operational problem worth monitoring.

### 3.3 The kagan Project (Notable Comparison)

**`kagan` (kagan-sh) — ⭐ 46 stars but 67 releases, actively developed.**

kagan is the most structurally similar existing tool to what kanbanzai is building. It deserves close examination.

**What kagan is:** A terminal Kanban TUI + web dashboard that orchestrates AI coding agents. Its tagline is "orchestration layer for AI coding agents." It supports 14 coding agents (Claude Code, Codex, Gemini, OpenCode, and others) in autonomous or pair mode.

**Key capabilities:**
- SQLite-backed Kanban board (Projects → Tasks)
- **Git worktree management** — explicitly listed as a core feature; each agent works in its own worktree
- Web dashboard for progress visibility
- Agent dispatch: hand tasks to coding agents from the Kanban UI
- Supports "autonomous" mode (agent works until done) and "pair" mode (human-in-the-loop)

**The gap between kagan and kanbanzai:**

| Dimension | kagan | kanbanzai |
|---|---|---|
| Task/feature lifecycle state machines | ❌ Flat Kanban statuses | ✅ Typed, validated state machines |
| Design document graph | ❌ None | ✅ Three-layer semantic index |
| Document → entity tracing | ❌ None | ✅ Auto-injected into context |
| Tiered knowledge with confidence lifecycle | ❌ None | ✅ Wilson-score lifecycle + git staleness |
| Byte-budgeted context assembly | ❌ None | ✅ Priority-trimmed, byte-ceiling |
| Role-scoped context profiles | ❌ None | ✅ Profile inheritance |
| Semantic merge gates (workflow state checks) | ❌ Code review only | ✅ Tasks done, specs current, verification exists |
| Agent dispatch | ✅ 14 agents supported | ❌ Phase 4 |
| Web/desktop UI | ✅ TUI + web dashboard | ❌ Not yet designed |
| Git worktrees | ✅ | ✅ |

kagan solves the *dispatch* half of the problem. kanbanzai has solved everything that makes dispatch *worthwhile*: the context it can hand to a dispatched agent is richer, more accurate, and more precisely scoped than anything kagan can produce. A kagan-dispatched agent begins a task with a task description; a kanbanzai-dispatched agent begins with a complete context packet: role conventions, design fragments traced from the feature spec, relevant knowledge entries with staleness annotations, and acceptance criteria. These are not equivalent starts.

**The web/desktop UI observation:** kagan's TUI and web dashboard address a real need that kanbanzai does not currently serve: visibility for designers and managers who want to track progress and access documentation without using the CLI or MCP interface. This is noted as a future track — see §7.

### 3.4 Anthropic-Native Capabilities (2025)

Claude's API now natively accepts `mcp_servers` per-request. Agent Skills (reusable capability bundles as MCP servers) can be published to Anthropic's registry. Built-in tools (`web_search`, `code_execution`, `memory` beta, `bash`, `computer_use`) are available directly via the Messages API.

What Anthropic does **not** provide:
- No task tracking or backlog
- No multi-session agent dispatch (spawning multiple sessions)
- No git/worktree integration
- No workflow state persistence across days/weeks

The Anthropic platform provides the execution substrate. kanbanzai provides the workflow semantics on top of it.

---

## 4. Capability Matrix

```
                        Task     Agent    Context     Git/    Document   Knowledge
                      Tracking  Dispatch  Assembly  Worktrees  Graph    Lifecycle
──────────────────────────────────────────────────────────────────────────────────
kanbanzai (current)     ✅✅      ❌         ✅✅       ✅✅       ✅✅       ✅✅
kanbanzai (Phase 4)     ✅✅      ✅✅        ✅✅       ✅✅       ✅✅       ✅✅
──────────────────────────────────────────────────────────────────────────────────
mcp-agent               ❌        ✅✅       ✅✅        ❌         ❌         ❌
Agent-MCP (rinadelph)   ✅✅      ✅✅        ✅          ❌         ❌         ❌
Network-AI              ✅        ✅✅        ✅          ❌         ❌         ❌
kagan                   ✅✅      ✅✅        ⚠️         ✅✅        ❌         ❌
claude-task-master      ✅✅      ⚠️         ✅✅         ❌         ❌         ❌
LangGraph               ⚠️       ✅✅        ⚠️          ❌         ❌         ❌
OpenAI Agents SDK       ❌        ✅✅        ✅✅         ❌         ❌         ❌
AutoGen                 ❌        ✅✅        ❌           ❌         ❌         ❌
CrewAI                  ⚠️       ✅✅        ❌           ❌         ❌         ❌
Anthropic (native)      ❌        ❌          ✅✅         ❌         ❌         ❌
```

No existing tool combines task tracking, context assembly, document intelligence, knowledge lifecycle, and git worktree management. The combination is unique to kanbanzai.

---

## 5. What kanbanzai Already Has That Is Unique

This section documents the capabilities already built — the things Phase 4 must build *on top of*, and that no external orchestration tool can replicate.

### 5.1 Entity Hierarchy as a Context Scoping Instrument

kanbanzai defines a typed entity graph: **Plan → Epic → Feature → Task/Bug → Decision**. These are first-class objects with validated lifecycle state machines and typed schemas enforced at write time.

When a task is dispatched, context assembly walks up to `parent_feature`, then calls `intelligenceSvc.TraceEntity(parentFeature)` to find every design document section that mentions that feature. The entity hierarchy is the scoping key for design context retrieval — not a tag, not a label, but a structural traversal.

No existing tool has this. GitHub Issues has flat labels. Linear has hierarchy but no document graph. Jira has epic links but no semantic index. The entity hierarchy as an automatic context scoping instrument is kanbanzai's most distinctive structural feature.

### 5.2 Document Intelligence Pipeline

The document intelligence subsystem builds a three-layer index over design documents:

- **Layer 1 — Structural:** Section tree with paths, titles, levels, word counts, content hashes
- **Layer 2 — Metadata:** Entity references extracted from section text (FEAT-xxx, TASK-xxx, etc.)
- **Layer 3 — Semantic:** Agent-assigned classifications (`requirement`, `decision`, `rationale`, `constraint`, `assumption`, `risk`) with confidence scores and concept tags

This enables:
- `doc_trace(entity_id)` — all sections across all documents referencing a feature/task, ordered by document type
- `doc_find_by_concept("context budget")` — sections by semantic concept
- `doc_find_by_role("decision")` — all decisions across the corpus
- `doc_impact(section_id)` — what depends on a given design section
- `doc_gaps(feature_id)` — which of design/spec/dev-plan documents are missing

Context assembly calls `TraceEntity()` automatically and injects matching design sections into the dispatch packet without the agent needing to search. This pipeline does not exist anywhere else. codebase-memory-mcp indexes code structure; no tool indexes design documents with semantic role classification and exposes them as a queryable graph for context injection.

### 5.3 Tiered Knowledge with Lifecycle Management

Three tiers of knowledge, each with a defined lifecycle:

- **Tier 1** — Role profiles (human-authored YAML with inheritance)
- **Tier 2** — Persistent architecture knowledge (role-scoped, confidence ≥ 0.3 to surface)
- **Tier 3** — Session-derived discoveries (scoped to role/project, confidence ≥ 0.5 to surface)

Entries flow: **contribute → confirm → stale → retire**. Confidence uses the Wilson score lower bound. `knowledge_check_staleness` compares git anchors against commit history — if the files an entry was anchored to have changed since `last_confirmed`, the entry is flagged stale and the staleness annotation is included in the assembled context packet.

This is unique. No other tool has agent-facing knowledge lifecycle management with confidence scoring and git-staleness detection.

### 5.4 Byte-Budgeted, Priority-Trimmed Context Assembly

`context_assemble` implements a deterministic assembly order:

1. Role profile (never trimmed, always first)
2. Tier 2 knowledge filtered by role/project scope
3. Tier 3 knowledge filtered by role/project scope
4. Design context from document graph (traced via parent_feature)
5. Task instructions (never trimmed, always last)

When over the byte budget (default 30,720 bytes):
1. Trim Tier 3 entries, lowest confidence first
2. Trim Tier 2 entries, lowest confidence first
3. Trim design context from the tail

The model's identity is deliberately not exposed. Bytes are the universal unit — avoiding server-side tokenisation which would be inaccurate and create tokeniser maintenance burden. This is a principled, implementable design choice that no other tool has made explicitly.

### 5.5 Semantic Merge Gates

Merge readiness is checked against *workflow semantic state*, not just CI status. The `merge_readiness_check` gates verify: are all tasks `done`? Does a verification record exist? Are design documents current? Is the branch not stale? These operate at the level of the workflow ontology. GitHub's merge queue checks CI/CD; kanbanzai's gates check whether the *work itself* meets the definition of done as recorded in workflow state.

### 5.6 Behavioral Protocol for Normalization Discipline

The agent interaction protocol (§6) codifies rules enforced both by the tool and specified for agents: treat human input as intake not canonical state; normalize before commit; show normalized output before important commits; use the MCP interface for canonical changes. The workflow kernel validates schema, enforces lifecycle transitions, and rejects malformed input — so agents that violate the protocol by editing state files directly break system invariants. This is a behavioral contract, not just a data model.

---

## 6. Should kanbanzai Build Its Own Orchestration?

**Yes. The reasons are structural, not preferential.**

The context injection pipeline is the load-bearing wall. When kanbanzai dispatches a task to a fresh agent session, the packet it produces is assembled from: role profile, design fragments traced automatically through the entity hierarchy and document graph, tiered knowledge entries with staleness annotations, and task acceptance criteria. No external orchestration framework knows how to produce that packet.

An external orchestrator (mcp-agent, LangGraph, etc.) could call `context_assemble` — but then it is just a caller. kanbanzai is still doing all the work. Using an external framework as the dispatch loop would add a dependency and an abstraction layer with no corresponding capability gain. It would also introduce incompatible assumptions about state management, agent identity, and session lifecycle.

The right architecture — and the one kanbanzai is already building toward — is: **kanbanzai provides the knowledge and context; the orchestrator is Claude (or any model) calling kanbanzai's MCP tools in a loop.** The value is in what the tools do, not in the loop machinery. Phase 4 adds the tools that complete the loop.

---

## 7. Key Learnings from the Field

The following patterns have been validated across multiple production-grade tools. Phase 4 should adopt them rather than rediscover them.

### 7.1 Ephemeral Agents + Shared Knowledge Store

*Validated by: Agent-MCP, mcp-agent, dagger/container-use*

Agents should be short-lived with minimal upfront context, querying a persistent shared knowledge graph on demand. This prevents context pollution and hallucination accumulation across a long task. kanbanzai's tiered `context_assemble` + on-demand `context_get` already implements this model. Phase 4 should reinforce it: the dispatch packet is the *minimum* the agent needs to start; additional context is pulled via the existing MCP tools as needed.

### 7.2 Task Decomposition Must Gate Dispatch

*Validated by: claude-task-master, kagan, Agent-MCP*

Every mature tool has learned: never dispatch an agent to a task that is not already granular, unambiguous, and properly specified. The orchestration loop must refuse to dispatch a task that is not in `ready` status — meaning it has a summary, acceptance criteria, and resolved dependencies. This is a process constraint, not a data structure.

### 7.3 Atomic Claiming Prevents Parallel Conflicts

*Validated by: Agent-MCP (file-level locking), Network-AI (FSM transitions)*

With a shared work queue, two orchestrating agents can both read the same unassigned task and try to start it. The fix is atomic claiming: a single state transition (`ready → active`) that only one claimant can win. kanbanzai's state machine already provides this — but the Phase 4 `dispatch_task` operation must make the transition atomic and return a clear refusal if the task is already claimed. The existing `StatusTransitionHook` pattern provides the right extension point.

### 7.4 Budget Awareness Must Be Visible to the Receiving Agent

*Validated by: Network-AI (budget tracking), mcp-agent (TokenCounter), context-rot-detection*

It is not enough for the server to respect a byte budget during assembly. The receiving agent should know what was trimmed and why — so it can request missing context explicitly rather than proceeding with an incomplete picture and hallucinating the gaps. Phase 4 should add a `trimmed` field to `context_assemble` responses listing what was cut (entry IDs, priorities, sizes) so executing agents can make informed decisions about what to pull additionally.

### 7.5 Expose Orchestration as MCP Tools, Not a Framework

*Validated by: Agent-MCP, kagan*

The orchestration state machine should be exposed as MCP tools that an agent calls in a conversation loop — not as a separate daemon, not as a Python framework, not as a Temporal workflow. `dispatch_task`, `work_queue`, `complete_task` are MCP tools. The orchestrating agent is Claude calling those tools. This keeps everything in the MCP protocol layer, avoids a meta-framework abstraction, and means the orchestration behaviour is transparent and auditable. kanbanzai already follows this principle and should double down on it in Phase 4.

---

## 8. The Web/Desktop UI Opportunity

kagan has a terminal Kanban TUI and a web dashboard. This solves a real problem that kanbanzai does not currently address: **visibility for designers and managers who are not running the CLI or using an AI agent interface.**

A designer tracking feature progress, a manager reviewing whether the sprint is on track, or a stakeholder wanting to read the design documents — none of these people will use `kbz` or call MCP tools directly. They need a read-oriented interface that surfaces:

- Entity status and lifecycle progress (Epic → Feature → Task breakdown)
- Document access (design docs, specs, dev plans)
- Worktree and branch health
- Knowledge and decisions relevant to a feature
- Progress metrics (once estimation is implemented)

This is a meaningful future track. It is explicitly **not Phase 4** — orchestration comes first, and the web UI should consume a stable API surface. But it should be flagged now as **Phase 5 (or a parallel track)** so the Phase 4 API design accounts for it.

Specifically: Phase 4 MCP tools and the underlying data model should be designed with the assumption that a read-oriented web interface will eventually query them. This does not require any Phase 4 implementation work, but it does mean avoiding decisions that would make such a UI awkward (e.g., coupling query responses to agent-specific conventions that a human dashboard wouldn't need).

The kagan comparison is instructive. kagan built the UI *before* the rich context layer. kanbanzai has done the inverse: the rich context layer is built first. When the UI comes, it will have a much deeper substrate to surface than kagan does.

---

## 9. New Items Identified for the Phase 4 Plan

The following items were not present in the original Phase 4 design documents and should be added to `work/plan/phase-4-scope.md`:

| Item | Phase | Origin |
|---|---|---|
| `work_queue` MCP tool — returns tasks ready for dispatch (ready status + all dependencies met) | 4a | Core gap identified in this review |
| `dispatch_task` MCP tool — atomic claim + context assembly + dispatch record in one operation | 4a | Atomic claiming pattern (§7.3) |
| `complete_task` MCP tool — marks done, contributes knowledge, closes dispatch record | 4a | Closing the orchestration loop |
| `claimed_at` timestamp on Task — records when an agent claimed the task | 4a | Atomic claiming pattern (§7.3) |
| `dispatched_to` / `dispatched_at` / `dispatched_by` fields on Task | 4a | Delegation model (human decision log) |
| Trimming visibility in `context_assemble` — `trimmed` field listing what was cut | 4a | Budget visibility pattern (§7.4) |
| Dependency enforcement in transition validator — tasks cannot go `active` if `depends_on` incomplete | 4a | Dependency modeling (human decision log) |
| Estimation tools — implement `estimation-and-progress-design.md` | pre-4a / 4a | Ready-to-implement design, no phase assigned |
| Incident entity (first-class, heavyweight model) | 4b | Human decision log |
| RootCauseAnalysis document type (linked to Incident) | 4b | Human decision log |
| Decomposition tools — break features into tasks | 4b | Phase 4 original scope |
| Vertical slice decomposition guidance | 4b | Deferred from Phase 3 |
| Conflict domain analysis | 4b | Deferred from Phase 3 |
| Worker review against specification | 4b | Phase 4 original scope |
| Web/desktop UI for designers and managers | Phase 5 / parallel | kagan comparison (§8) |

---

## 10. Summary

Phase 4 should build its own orchestration layer. The reasons are structural: kanbanzai's context injection pipeline — entity hierarchy → document graph → knowledge lifecycle → byte-budgeted assembly with staleness annotations — cannot be replicated by any external orchestration framework. An external tool would be a caller of `context_assemble`, not a replacement for it.

The existing orchestration landscape has validated five patterns that Phase 4 should adopt: ephemeral agents, decomposition gating, atomic claiming, budget visibility, and MCP-as-orchestration-surface. Three of these (ephemeral agents, MCP-as-surface, decomposition gating) are already designed into kanbanzai. Two (atomic claiming, budget visibility) are new additions from this review.

The Phase 4 implementation surface is smaller than it might appear. The heavy lifting — context assembly, knowledge lifecycle, document intelligence, git worktrees, health checks, semantic merge gates — is done. Phase 4a adds approximately five tools and a few entity fields to complete the orchestration loop. Phase 4b builds the richer decomposition and review capabilities on top of that foundation.

A web/desktop UI for designers and managers is a genuine future need — flagged here as a Phase 5 candidate and informed by kagan's implementation. The Phase 4 API surface should be designed with this in mind.