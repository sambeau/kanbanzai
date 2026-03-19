# Machine-to-Machine Context Design

- Status: draft design
- Date: 2026-03-18
- Purpose: define the context management model for AI agent teams in Kanbanzai
- Related:
  - `work/design/workflow-design-basis.md` §14, §15, §18
  - `work/design/document-centric-interface.md` §8.4
  - `work/spec/phase-1-specification.md` §20
  - `work/design/agent-interaction-protocol.md`

---

## 1. Purpose

This document defines the machine-to-machine context model for Kanbanzai — how the system manages, stores, assembles, and shares context for AI agent teams working on software development tasks.

Kanbanzai's primary purpose is the coordination of AI agent teams to efficiently turn designs into working software. The document-centric interface design defines how humans interact with the system. This document defines the other half: how the system interacts with AI agents, specifically how it manages the knowledge and context they need to work efficiently.

The central claim is: **context is the critical resource in AI-assisted software development, and managing it well is the primary leverage point for agent team productivity.**

---

## 2. Problem Statement

### 2.1 The context problem

Every AI agent has a finite context window. If each agent must ingest the full design library and full codebase knowledge to understand its task, the system does not scale:

- Contexts fill with irrelevant material
- Agents work slowly as they process unnecessary information
- The risk of misinterpretation grows with noise
- Token costs scale linearly with context size
- Attention degradation reduces output quality as context grows

### 2.2 The persistence problem

AI agents are stateless between sessions. A human developer accumulates project knowledge over months. An agent starts fresh each invocation. Without persistent, structured context, every agent session begins with an expensive rediscovery process — reading files, inferring conventions, reconstructing mental models that a previous agent session already built.

### 2.3 The sharing problem

When multiple agents work in parallel on the same project, they each independently discover the same conventions, patterns, and constraints. Without a mechanism to share knowledge between agent sessions, this work is duplicated for every task.

### 2.4 The scoping problem

Not all knowledge is relevant to all agents. A backend developer agent does not need to know CSS conventions. A test writer does not need to know deployment procedures. Loading irrelevant context wastes tokens, degrades attention, and slows work. Knowledge must be scoped to the agent's role and task.

---

## 3. Design Goals

1. **Minimise context per task** — each agent receives precisely the knowledge it needs, assembled from the right sources, without irrelevant material.
2. **Persist knowledge across sessions** — knowledge discovered during agent work is captured and available to future agents without human intervention.
3. **Scope knowledge by role** — different agent specialisms receive different context profiles, tailored to their domain.
4. **Compose design and implementation context** — the system assembles both what-to-build (from design documents) and how-to-build-it (from accumulated implementation knowledge) into a single coherent context packet.
5. **Stay simple** — no ML pipelines, no embedding databases, no knowledge graphs. Structured files in Git, queried through the MCP interface.

---

## 4. Two Types of Context

The system distinguishes two fundamentally different types of context that agents need.

### 4.1 Design context

Design context is information about **what we are trying to build**. It flows top-down through the entity hierarchy: project goals → epics → features → tasks. It originates in human-authored documents — proposals, designs, specifications, implementation plans — and is fragmented internally by the system for targeted assembly.

Design context assembly is already described in the document-centric interface design (§8.4) and the workflow design basis (§8.5, §18.4). It works by:

- Indexing decisions, linking requirements to features, connecting spec sections to design rationale
- Composing a targeted slice for each agent: the relevant design sections, specification requirements, decisions, and constraints for a specific task
- Excluding everything not relevant to the task at hand

Design context is relatively stable once approved. It changes when designs are revised, not when code is written.

### 4.2 Implementation context

Implementation context is information about **how we are building it**. It emerges bottom-up and laterally during development: code conventions, architectural patterns, tooling choices, integration gotchas, lessons learned from failed approaches.

Implementation context is not covered by the existing design. This document defines it.

Examples:

| Category | Example |
|---|---|
| Language conventions | "Error handling uses `internal/errors.Wrap`; never use bare `fmt.Errorf`" |
| Architecture patterns | "The MCP layer is a thin adapter; all business logic lives in `internal/core`" |
| Tooling | "Tests use table-driven style; mocks only at system boundaries" |
| Integration knowledge | "The YAML serializer always uses block style; field order is deterministic" |
| Negative knowledge | "We tried X approach and it failed because Y — use Z instead" |

### 4.3 Other context types

Beyond design and implementation context, agents may need:

- **Policy context** — project-wide rules about quality, review, commit discipline (partially covered by existing agent instruction files)
- **Coordination context** — what other agents are currently working on, what's blocked, what's been completed (provided by workflow state)

These are handled by existing mechanisms (AGENTS.md, workflow MCP operations) and do not require new design.

---

## 5. Context Tiers

Implementation context is further structured into three tiers, distinguished by scope and stability.

### 5.1 Tier 1: Project conventions

Relatively stable knowledge that applies across the entire project.

- Language and framework choices
- Code organisation patterns
- Naming conventions
- Error handling patterns
- Test patterns
- Build and deployment conventions

These are essentially what CLAUDE.md files and coding rules try to capture today, but structured for machine retrieval and scoped by role.

Change frequency: low (revised when major architectural decisions change).

### 5.2 Tier 2: Architecture knowledge

Moderately stable knowledge about how the codebase is structured and how its parts interact.

- Module boundaries and responsibilities
- Key abstractions and interfaces
- Data flow patterns
- Integration patterns between components
- Known constraints and gotchas

This is knowledge *about the code* that is not obvious from reading any single file but emerges from working with the codebase over time.

Change frequency: moderate (evolves as the codebase grows).

### 5.3 Tier 3: Session knowledge

Ephemeral but shareable knowledge produced during specific agent work sessions.

- "I tried approach X and it didn't work because Y"
- "This feature interacts with component W in a non-obvious way"
- "The spec says A but the code currently does B — clarified with the human that A is correct"

Change frequency: high (produced every session, may become stale quickly).

---

## 6. Context Profiles (Roles)

### 6.1 Concept

A **context profile** is a named bundle of knowledge that gets loaded into an agent's context at the start of a session. It defines what the agent needs to know for its speciality.

Context profiles are not job titles — they are scoping mechanisms. The same underlying model receives a different context profile depending on what it is doing. An agent assigned the `backend` profile and an agent assigned the `testing` profile may be the same model, but they receive different implementation context.

### 6.2 Profile contents

A context profile contains:

- **Identity and scope** — what this role covers, what packages/directories it owns
- **Conventions** — role-specific coding conventions, patterns, and anti-patterns
- **Architecture overview** — how the role's domain relates to the rest of the system
- **Tool and workflow guidance** — how to retrieve additional context, how to contribute knowledge back

### 6.3 Example profile

```yaml
id: backend
scope:
  description: "Backend development — core logic, MCP service layer, CLI"
  packages:
    - internal/core
    - internal/mcp
    - cmd/kanbanzai
conventions:
  - "Error handling: use internal/errors.Wrap, never bare fmt.Errorf"
  - "Tests: table-driven, no mocks for pure functions"
  - "Logging: structured via slog, never fmt.Println"
architecture:
  summary: "MCP layer is a thin adapter; all business logic in internal/core"
  key_interfaces:
    - "core.Store — canonical read/write interface for entities"
    - "core.Validator — schema and transition validation"
context_retrieval:
  task: "kbz context get --task {task-id}"
  role: "kbz context get --role backend"
  project: "kbz context get --scope project"
```

### 6.4 Suggested initial profiles

For a typical software project:

| Profile | Scope |
|---|---|
| `backend` | Core logic, APIs, data layer |
| `frontend` | UI, client-side logic, styling |
| `testing` | Test strategy, test implementation, coverage |
| `documentation` | User docs, API docs, internal docs |
| `infrastructure` | Build, deploy, CI/CD, configuration |

Projects define their own profiles. A small project might have only `developer` and `testing`. A large project might have a dozen.

### 6.5 Specialist context tradeoff

Narrower speciality means less breadth of context but potentially more depth. A backend developer agent does not need to know about CSS conventions, but needs to know the error handling patterns, dependency injection approach, and database layer conventions in detail. The total token count may be similar to a generalist, but the signal-to-noise ratio is much higher.

The tradeoff: more specialised agents need more coordination. Their tasks must be well-defined and scoped precisely to their domain. This is the same tradeoff seen in human teams, and it has the same resolution: a coordination layer (§7) manages the handoffs.

---

## 7. Coordination Agents

### 7.1 Purpose

The workflow design basis (§14) defines a four-tier delegation hierarchy: humans → PM/orchestration agents → specialist team agents → execution agents. This document focuses on the context needs of each tier.

### 7.2 Orchestrator context

Orchestration agents need **broad but shallow** context:

- The overall architecture and module map
- What each specialist role covers
- The current state of work (what's in progress, what's blocked, what's completed)
- Dependencies between tasks
- Enough design context to decompose features into tasks

They do *not* need deep implementation details — they need to know the module boundaries well enough to assign work to the right specialist.

### 7.3 Specialist context

Specialist agents need **narrow but deep** context:

- Their role's context profile (§6)
- Task-specific design context (assembled from the entity hierarchy)
- Relevant session knowledge from recent work in their domain
- Enough coordination context to understand handoff requirements

### 7.4 Handoff context

When one agent's work produces information that another agent needs, the orchestrator (or the system) must extract and pass **handoff context** — a small, targeted summary of what was done and what the next agent needs to know.

Handoff context is a special case of Tier 3 session knowledge that is explicitly directed at a specific next task rather than contributed to the general knowledge store.

---

## 8. Context Assembly

### 8.1 Assembly model

When an agent is assigned a task, the system assembles a **context packet** — a single coherent bundle of all the context the agent needs. The assembly draws from multiple sources in a defined order:

1. **Role profile** — the agent's speciality conventions, scope, and architecture overview
2. **Project context** — project-wide conventions and knowledge (Tier 1)
3. **Design context** — task-specific design fragments assembled from the entity hierarchy (epic goals → feature spec → task requirements → relevant decisions)
4. **Implementation context** — role-scoped Tier 2 architecture knowledge relevant to the task's code area
5. **Session context** — recent Tier 3 knowledge from related tasks
6. **Task instructions** — acceptance criteria, constraints, verification requirements

### 8.2 Assembly principle

The assembly follows a **need-to-know principle**: include only what is relevant to this specific task and role. The entity hierarchy provides natural scoping for design context. Role profiles provide scoping for implementation context. The system should err on the side of less context rather than more.

### 8.3 Context budget

Each context packet should be assembled with a token budget in mind. The system should be aware of approximate context sizes and warn (or trim) when a packet would consume too large a fraction of the model's context window. Leaving adequate room for the agent's reasoning and output is as important as providing the right input context.

### 8.4 Relationship to the four-layer instruction stack

The workflow design basis (§18) defines a four-layer instruction and control stack:

1. Platform-native agent instructions (AGENTS.md, skill files, coding rules)
2. Workflow system rules (schemas, transitions, validation)
3. Generated context packets
4. Workflow MCP interface

Context profiles and assembled context packets are **layer 3** in this stack. They sit between the static instruction files (layer 1) and the dynamic MCP interface (layer 4). The role profile concept extends but does not replace layer 1 — platform-native instructions remain in place; context profiles add structured, machine-retrievable, role-scoped knowledge.

---

## 9. Knowledge Contribution

### 9.1 Feedback loop

Context is not only consumed — agents produce valuable knowledge during their work. The system must support a feedback loop where agents contribute knowledge back for future agents to use.

### 9.2 Contribution mechanism

Agents contribute knowledge through an explicit MCP operation:

```
context_contribute(
  scope: "role" | "project",
  topic: string,
  content: string,
  learned_from: string  # task ID or session reference
)
```

The agent is instructed: "When you discover a convention, pattern, or constraint that future agents working in this area should know, use `context_contribute` to record it."

This is deliberate and opt-in. Agents contribute when they have something worth sharing, not automatically after every session.

### 9.3 Knowledge review

Contributed knowledge should be reviewed before it becomes part of the canonical context:

- **Low-risk contributions** (conventions observed in existing code) can be accepted automatically or with lightweight review
- **High-impact contributions** (new patterns, architectural claims) should be flagged for human or orchestrator review
- **Contradictory contributions** (conflicting with existing knowledge) must be surfaced as conflicts, not silently overwritten

### 9.4 Knowledge lifecycle

Knowledge entries have a lifecycle:

1. **Contributed** — proposed by an agent, not yet confirmed
2. **Confirmed** — accepted (by review or repeated observation)
3. **Stale** — flagged because the code it describes has changed significantly
4. **Retired** — explicitly marked as no longer applicable

### 9.5 Preventing context rot

Knowledge goes stale as the codebase evolves. Mechanisms to address this:

- **Timestamp tracking** — entries have creation and last-confirmed timestamps
- **Git anchoring** — entries can be tied to specific file paths or code ranges; when those files change beyond a threshold, the entry is flagged for review
- **Confirmation on use** — when an agent retrieves a knowledge entry and finds it still accurate, it can confirm it (updating the timestamp); when it finds it inaccurate, it can flag or retract it

---

## 10. Relationship to Existing Knowledge Design

The workflow design basis (§15) already defines memory classes, knowledge governance, and a knowledge entry format. This document is consistent with that design and extends it:

| Design Basis Concept | This Document |
|---|---|
| Canonical project memory | Design context + Tier 1 project conventions |
| Team operational memory | Role-scoped Tier 2 architecture knowledge |
| Working memory | Agent session state (not persisted by this system) |
| Expertise memory | Context profiles (§6) |
| KnowledgeEntry | Tier 2 and Tier 3 contributed entries (§9) |
| Team convention | Role-scoped conventions within a context profile |

The knowledge entry format suggested in the design basis (§15.4) — with fields for id, team, topic, tags, summary, detail, learned_from, date — maps directly to the contribution model described here.

---

## 11. Relationship to Existing Tools and Prior Art

### 11.1 CLAUDE.md / AGENTS.md / Cursor rules

These are static, human-maintained instruction files. They partially address Tier 1 project conventions but are:

- Flat and unstructured
- Not role-scoped
- Not machine-writable (agents cannot contribute back)
- Not composable with design context

Context profiles (§6) supersede these for role-scoped knowledge while coexisting with platform-native instruction files for platform-specific configuration.

### 11.2 Skills files

Skills (as implemented in various AI coding tools) are task-specific recipes — "how to write a database migration", "how to deploy to staging". They are useful but:

- Fill context regardless of relevance to the current task
- Not role-scoped or project-specific
- Typically read in full, wasting tokens on irrelevant sections
- Cannot be contributed to by agents

The context profile model addresses the same need (giving agents specialised knowledge) but with project-specific, role-scoped, dynamically assembled content rather than generic static recipes.

### 11.3 Codebase memory tools

Tools like codebase-memory-mcp effectively capture Tier 2 architecture knowledge using embedding-based retrieval. They solve a complementary problem and could coexist with Kanbanzai's context system. However, they lack:

- Role scoping
- Design context integration
- Feedback loops with knowledge lifecycle management
- Hierarchical context assembly tied to a work breakdown structure

### 11.4 Multi-agent orchestration frameworks

Frameworks like LangGraph provide the plumbing for multi-agent coordination (durable execution, state persistence, human-in-the-loop) but do not address context management. Agents in these frameworks share state through explicit graph edges, not through structured knowledge retrieval. They solve orchestration; Kanbanzai solves context.

### 11.5 What does not exist

No existing system combines:

1. Role-scoped, project-specific implementation knowledge
2. Hierarchical context assembly (project → team → role → task)
3. Agent-to-agent feedback loops with knowledge lifecycle management
4. Integration of implementation context with design context in a single retrieval interface

This is the gap Kanbanzai addresses.

---

## 12. MCP Operations

The following operations extend the MCP interface for context management. These are Phase 2 deliverables; Phase 1 must not preclude them.

### 12.1 Retrieval operations

```
context_get(
  scope: "task" | "role" | "project",
  id: string
) → context packet

context_list(
  scope: "role" | "project"
) → list of available context profiles or entries
```

### 12.2 Contribution operations

```
context_contribute(
  scope: "role" | "project",
  topic: string,
  content: string,
  learned_from: string
) → entry ID

context_retract(
  entry_id: string
) → confirmation
```

### 12.3 Assembly operations

```
context_assemble(
  task_id: string,
  role: string
) → complete context packet for a task + role
```

### 12.4 Confirmation operations

```
context_confirm(
  entry_id: string
) → confirmation (updates last-confirmed timestamp)

context_flag(
  entry_id: string,
  reason: string
) → confirmation (marks entry for review)
```

---

## 13. Storage Model

Context is stored alongside entity state in the Kanbanzai instance root. This is a suggested structure; the final layout is a Phase 2 decision.

```
.kbz/
├── state/           # existing entities (epics, features, tasks, etc.)
├── context/
│   ├── project/     # Tier 1: project-wide conventions and knowledge
│   │   └── conventions.yaml
│   ├── roles/       # context profiles per role
│   │   ├── backend.yaml
│   │   ├── frontend.yaml
│   │   └── testing.yaml
│   ├── knowledge/   # Tier 2: contributed architecture knowledge
│   │   ├── KE-001.yaml
│   │   └── KE-002.yaml
│   └── sessions/    # Tier 3: session knowledge (may be pruned)
│       ├── SE-001.yaml
│       └── SE-002.yaml
├── specs/           # existing document storage
├── plans/           # existing document storage
└── cache/           # existing derived cache
```

Context files follow the same principles as entity files: schema-validated YAML, deterministic serialisation, Git-tracked, tool-written.

---

## 14. Bootstrapping

### 14.1 The cold-start problem

A new project has no implementation context. How does the system become useful before agents have contributed knowledge?

### 14.2 Bootstrapping strategy

1. **Start with role profiles** — a human writes a short context profile per specialist role. This is comparable to writing a CLAUDE.md file, but scoped by role and structured for machine retrieval. This provides Tier 1 coverage from day one.

2. **Let knowledge accumulate organically** — as agents complete tasks, they contribute Tier 2 and Tier 3 knowledge through the feedback loop. The system starts sparse and becomes richer over time.

3. **Optional codebase analysis** — for projects with an existing codebase, an agent can perform a one-time analysis pass to extract initial Tier 2 knowledge (module boundaries, conventions observed in code, key interfaces). This is not required but accelerates bootstrapping.

### 14.3 Progressive value

The system should be useful at every stage:

- **With only role profiles**: agents get scoped conventions and architecture overview — better than a flat AGENTS.md
- **With accumulated Tier 2 knowledge**: agents get project-specific patterns and constraints — significantly better than starting fresh
- **With active Tier 3 sharing**: agents benefit from each other's discoveries — approaching the efficiency of a well-coordinated human team

---

## 15. Phasing

### 15.1 Phase 1 (current)

Phase 1 builds the entity kernel. For context management, Phase 1 must:

- **Not preclude** any of the context operations described here
- **Reserve namespace** in the MCP operation set for context operations
- **Design the storage layout** to accommodate context entries alongside entities

Phase 1 does not implement context profiles, assembly, or contribution.

### 15.2 Phase 2

Phase 2 implements retrieval and context packing (per the workflow design basis §24). This includes:

- Context profile definition and retrieval
- Context assembly for tasks
- Knowledge contribution and lifecycle
- Integration of design context and implementation context in assembled packets

### 15.3 Phase 3 and beyond

Later phases may add:

- Automatic context assembly optimisation (learning which context entries are most useful)
- Cross-project knowledge sharing (conventions that apply to all projects by a team)
- Orchestration-driven decomposition informed by context profiles

---

## 16. Aesthetic Constraints

Per the Parsley aesthetic, this design should remain:

- **Simple** — context profiles are named YAML files. No ML pipelines, no embedding databases, no knowledge graphs. Structured files, queried through MCP operations.
- **Minimal** — start with a small number of operations (get, contribute, assemble). Add complexity only when demonstrated need arises.
- **Complete** — the design-context + implementation-context model covers the full space of knowledge an agent needs. No obvious gaps in the model.
- **Composable** — context profiles compose with the entity hierarchy and document fragments. The same MCP interface serves orchestrators and specialists differently through scoping parameters, not separate APIs.

---

## 17. Open Questions

The following questions are deferred to Phase 2 planning:

1. **Token budget management** — how does the system measure and manage context packet size? Does it need model-specific token counting, or are rough estimates sufficient?

2. **Knowledge deduplication** — when multiple agents contribute similar knowledge, how is it deduplicated?

3. **Cross-role knowledge** — some knowledge applies to multiple roles but not all. How is this scoped?

4. **Confidence scoring** — should knowledge entries have confidence levels? How would these be used in assembly?

5. **Session knowledge pruning** — Tier 3 knowledge is high-volume. What is the retention policy?

6. **Context profile inheritance** — can roles inherit from a base profile? (e.g., all roles share project conventions, then specialise)

7. **Integration with code-aware tools** — if a project also uses codebase-memory-mcp or similar, how do the knowledge stores interact?

---

## 18. Summary

Kanbanzai manages two types of context for AI agent teams:

- **Design context** — what to build — assembled from the document-to-entity hierarchy (already designed)
- **Implementation context** — how to build it — structured in three tiers (project conventions, architecture knowledge, session knowledge) and scoped by role profiles (new in this document)

The system assembles both into targeted context packets, delivered through MCP operations, so that each agent receives precisely the knowledge it needs — no more, no less.

Agents both consume and produce context through a feedback loop, building a persistent knowledge store that makes each subsequent agent session more efficient than the last.

The result is a system where AI agent teams can approach the efficiency of well-coordinated human teams: specialists with deep domain knowledge, orchestrators with broad situational awareness, and a shared knowledge base that grows with the project.