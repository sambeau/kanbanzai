---
id: DOC-01KM8JV0JGRZ1
type: design
title: Machine-to-Machine Context Design
status: submitted
feature: FEAT-01KM8JT7542GZ
created_by: human
created: 2026-03-21T16:14:48Z
updated: 2026-03-21T16:14:48Z
---
# Machine-to-Machine Context Design

- Status: draft design
- Date: 2026-03-18
- Purpose: define the context management model for AI agent teams in Kanbanzai
- Related:
  - `work/design/workflow-design-basis.md` §14, §15, §18
  - `work/design/document-centric-interface.md` §8.4
  - `work/design/document-intelligence-design.md` §5, §7, §12, §14.2
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

Design context assembly is described in the document-centric interface design (§8.4) and the workflow design basis (§8.5, §18.4). The concrete mechanism — a four-layer analysis model producing a queryable document graph — is defined in `work/design/document-intelligence-design.md`. It works by:

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

### 6.6 Profile inheritance

Context profiles form an inheritance hierarchy. A profile can declare a parent, inheriting all of the parent's conventions, architecture knowledge, and tool guidance, then adding or overriding with its own.

A natural hierarchy for a typical project:

```
base                  ← all agents get this (project-wide conventions)
├── developer         ← inherits base, adds code conventions
│   ├── backend       ← inherits developer, adds backend specifics
│   │   └── sql       ← inherits backend, adds database specifics
│   └── frontend      ← inherits developer, adds frontend specifics
├── testing           ← inherits base, adds test strategy
└── documentation     ← inherits base, adds doc conventions
```

In the profile YAML, this is expressed with an `inherits` field:

```yaml
id: backend
inherits: developer
scope:
  description: "Backend development — core logic, MCP service layer, CLI"
  packages:
    - internal/core
    - internal/mcp
    - cmd/kanbanzai
# ... role-specific conventions added here
```

When context is assembled for a `backend` agent, the system walks the inheritance chain (`base` → `developer` → `backend`) and layers each profile's content in order. More specific profiles override more general ones where there is a conflict.

This solves the cross-role knowledge scoping problem: knowledge that applies to all developers but not to documentation agents lives in the `developer` profile. Knowledge that applies to all agents lives in `base`. An agent contributing knowledge can specify the scope it belongs to, and a coordinator reviews whether the contribution belongs at the scope the agent suggested or should be promoted or demoted in the hierarchy.

Projects define their own inheritance trees. A small project might have only `base` and `developer`. A large project might have deep chains with multiple specialisations. The system does not enforce a specific hierarchy — it only requires that inheritance references resolve and do not form cycles.

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
3. **Design context** — task-specific design fragments from the document graph (epic goals → feature spec → task requirements → relevant decisions); the entity hierarchy provides scoping, the document graph (`work/design/document-intelligence-design.md` §7) provides the indexed fragments
4. **Implementation context** — role-scoped Tier 2 architecture knowledge relevant to the task's code area
5. **Session context** — recent Tier 3 knowledge from related tasks
6. **Task instructions** — acceptance criteria, constraints, verification requirements

### 8.2 Assembly principle

The assembly follows a **need-to-know principle**: include only what is relevant to this specific task and role. The entity hierarchy provides natural scoping for design context. Role profiles provide scoping for implementation context. The system should err on the side of less context rather than more.

### 8.3 Context budget

Each context packet should be assembled with a size budget. The system does not know which model will consume the packet — the MCP protocol does not expose the calling model's identity, context window size, or tokenizer. Models cannot reliably self-report their context limits at runtime, and there is no universal tokenizer across model families. Token counting on the server side would be inaccurate for half of users and would create a maintenance burden tracking tokenizer changes.

**Budget in bytes, not tokens.** Bytes are universal and observable. For YAML and English prose, a conservative ratio of ~1 token per 3.5 bytes provides a rough mental model, but the system expresses all limits in bytes.

**Default ceiling.** A single context packet should not exceed 30KB of text (~8–10K tokens across most tokenizers). This fits comfortably in even a 32K context window alongside system prompts, conversation history, and tool schemas. For teams using large-context models, this ceiling is configurable via a project setting.

**Tiered retrieval, not monolithic assembly.** Rather than trying to estimate the model's limit and assemble the maximum, the system provides tiered access:

1. `get_task` — compact essentials: task requirements, acceptance criteria, immediate constraints (~2–4KB)
2. `get_task_context` / `context_assemble` — full context packet: design fragments, role profile, architecture knowledge, session knowledge (~10–30KB)
3. `get_document` / `context_get` — individual documents and knowledge entries on demand

Agents request more detail when they need it. This pull-based model avoids over-stuffing context and works across models with different window sizes.

**Priority annotations.** Context entries include a priority indicator (high, normal, low). When a packet approaches the byte ceiling, low-priority entries are trimmed first. This can leverage the MCP `Annotations.priority` field so that smart clients can also participate in trimming decisions.

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

#### Deduplication on contribution

When an agent calls `context_contribute`, the system checks for existing entries that cover the same knowledge before creating a new entry:

1. **Exact topic match** — if an entry with the same `topic` key already exists in the same scope, the contribution is rejected with a pointer to the existing entry. The agent can update the existing entry instead.
2. **Near-duplicate detection** — the system computes Jaccard similarity over normalised word-sets (lowercase, stop words removed) between the new content and existing entries in the same scope. If similarity exceeds 0.65, the contribution is flagged as a potential duplicate and either rejected with a pointer or queued for merge review.

This is cheap (microseconds per comparison, no ML or embedding infrastructure) and catches the majority of same-branch duplicates. Cross-branch duplicates are handled by post-merge compaction (§9.6).

### 9.3 Knowledge review

Contributed knowledge should be reviewed before it becomes part of the canonical context:

- **Low-risk contributions** (conventions observed in existing code) can be accepted automatically or with lightweight review
- **High-impact contributions** (new patterns, architectural claims) should be flagged for human or orchestrator review
- **Contradictory contributions** (conflicting with existing knowledge) must be surfaced as conflicts, not silently overwritten

### 9.4 Knowledge lifecycle

Knowledge entries have a lifecycle:

1. **Contributed** — proposed by an agent, not yet confirmed
2. **Confirmed** — accepted (by review or repeated observation)
3. **Disputed** — contradicts another entry; both flagged for resolution
4. **Stale** — flagged because the code it describes has changed significantly
5. **Retired** — explicitly marked as no longer applicable

#### Confidence scoring

Each knowledge entry carries a confidence score derived from usage feedback. The score uses a Wilson score lower bound over `(use_count, miss_count)` — the same algorithm used by Reddit for comment ranking. This handles the key problem with raw ratios: an entry used once successfully (1/1 = 100%) should not outrank an entry used 50 times with 2 misses (48/50 = 96%). Wilson score naturally penalises low sample sizes.

New entries start with a prior confidence of 0.5 (uncertain). As they accumulate usage data, the score converges on true reliability.

Confidence is used during context assembly as a tier-dependent filter:

| Tier | Minimum confidence | Rationale |
|---|---|---|
| Tier 1 (conventions) | No filter | Human-curated, authoritative |
| Tier 2 (architecture) | 0.3 | Exclude entries that are mostly wrong |
| Tier 3 (session) | 0.5 | Higher bar — unproven entries excluded from context |

Confidence thresholds can be adjusted by context type: instructional knowledge ("how to do X") requires a higher threshold than informational knowledge ("X is structured as Y"), because wrong instructions cause cascading failures.

### 9.5 Preventing context rot

Knowledge goes stale as the codebase evolves. Mechanisms to address this:

- **Timestamp tracking** — entries have creation and last-confirmed timestamps
- **Git anchoring** — entries can be tied to specific file paths or code ranges; when those files change beyond a threshold, the entry is flagged for review
- **Confirmation on use** — when an agent retrieves a knowledge entry and finds it still accurate, it can confirm it (updating the timestamp); when it finds it inaccurate, it can flag or retract it

#### Usage reporting

When an agent completes a task, it reports which knowledge entries were used and whether they were accurate. This is bundled with task completion — not a separate API call — and kept lightweight:

- Default is "all entries were fine" (no elaboration needed)
- Only negatives require detail: "Entry KE-042 was wrong because the API endpoint changed"
- Reports update `use_count` (incremented when used and task succeeds) and `miss_count` (incremented when entry found wrong or unhelpful)

The reporting overhead is negligible (~50–100 tokens per report). A single prevented bad-knowledge incident — where a wrong entry causes a task to fail and retry — wastes 5K–50K tokens. The reporting pays for itself many times over.

#### Retention policy

Knowledge retention follows a cache-like model analogous to generational garbage collection: Tier 3 is the young generation (high churn, frequent collection), Tier 2 is the old generation (survives via promotion), Tier 1 is the permanent generation (manually managed).

| Rule | Tier 3 (session) | Tier 2 (architecture) | Tier 1 (conventions) |
|---|---|---|---|
| Default TTL | 14–30 days | 90 days | No expiry |
| TTL reset on use | Yes, reset to 30 days | Yes, reset to 90 days | N/A |
| Pruned when | TTL expires AND use_count < 3 | TTL expires AND confidence < 0.5 | Only by human |
| Promotion trigger | use_count ≥ 5 AND miss_count = 0 → Tier 2 | Human review | N/A |
| Immediate retirement | miss_count ≥ 2 OR flagged wrong | Flagged wrong by human | Human decision |

New entries receive a grace period (7 days) before pruning logic applies, avoiding premature removal of entries that haven't had the opportunity to be used.

Promotion from Tier 3 to Tier 2 creates a quality signal for human reviewers: "these session entries keep proving useful and may warrant review as architecture knowledge."

### 9.6 Post-merge compaction

In a multi-developer team using Git, parallel branches produce knowledge independently. When branches merge, duplicate or contradictory entries may appear. A post-merge compaction step resolves this.

**Detection.** After a merge, the system identifies knowledge files added or modified by the merge and compares them against existing entries:

1. **Exact duplicates** (same topic, same normalised content) — keep the entry with higher confidence; transfer usage counts from the discarded entry.
2. **Near-duplicates** (same topic or Jaccard similarity > 0.65 in the same scope) — auto-merge if both entries have confidence > 0.5 (take the union of content, sum usage counts, recompute confidence). Flag for human review if confidence scores diverge significantly.
3. **Contradictions** (same scope and topic overlap, but content diverges — Jaccard between 0.3 and 0.6) — never auto-resolve. Both entries are marked `status: disputed` until resolved by a human or coordinator agent. Both are included in context assembly with an annotation noting the conflict.
4. **Independent entries** (different topic or different scope) — no action needed.

**Scope.** Compaction applies freely to Tier 3 entries (ephemeral, agent-contributed). Tier 2 entries are flagged but not auto-modified (they've been reviewed). Tier 1 entries are never compacted — conflicts at this level are Git merge conflicts in the YAML files, handled by normal merge resolution.

**Triggering.** Compaction can be triggered automatically via a post-merge hook, manually via `kbz compact`, or by a coordinator agent after reviewing merge results. At realistic scales (hundreds of entries), pairwise comparison completes in under a second.

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

The document intelligence design (`work/design/document-intelligence-design.md` §12) introduces a separate lightweight structure — **concepts** — for tracking shared vocabulary across design documents. Concepts are graph-derived (extracted from documents by agents at ingest time, not contributed during implementation) and carry no lifecycle states, confidence scores, or TTLs. A concept may have a corresponding knowledge entry — the concept "TSID13" in the design corpus may correspond to a knowledge entry about how TSID13 is implemented in code — but the two structures are distinct. Concepts enable vertical slicing of the document corpus; knowledge entries capture implementation experience.

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
  role: string,
  max_bytes: int (optional, default 30720)
) → complete context packet for a task + role

```

The `max_bytes` parameter controls the byte ceiling for the assembled packet. When the assembled content exceeds this limit, low-priority entries are trimmed first (Tier 3 before Tier 2, lower confidence before higher). The default of 30KB (~8–10K tokens) fits comfortably in even a 32K context window.

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

### 12.5 Reporting operations

```
context_report(
  task_id: string,
  used: list of entry IDs,
  flagged: list of { entry_id: string, reason: string } (optional)
) → confirmation (updates use_count, miss_count, and confidence)
```

Bundled with task completion. The `used` list increments `use_count` for each entry. The `flagged` list increments `miss_count` and, if `miss_count` reaches the retirement threshold, marks the entry for review or retirement.

### 12.6 Compaction operations

```
context_compact(
  scope: "all" | "tier3" | "tier2" (default "tier3")
) → compaction report (duplicates merged, contradictions flagged, entries pruned)
```

Runs deduplication and retention policy enforcement across knowledge entries. Typically triggered after a merge or on a schedule. Tier 3 entries are compacted freely; Tier 2 entries are flagged but not auto-modified; Tier 1 entries are never compacted.

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
│   │   ├── base.yaml
│   │   ├── developer.yaml
│   │   ├── backend.yaml
│   │   ├── frontend.yaml
│   │   └── testing.yaml
│   ├── knowledge/   # Tier 2: contributed architecture knowledge
│   │   ├── KE-001.yaml
│   │   └── KE-002.yaml
│   └── sessions/    # Tier 3: session knowledge (may be pruned)
│       ├── SE-001.yaml
│       └── SE-002.yaml
├── index/           # document intelligence index (see document-intelligence-design.md §13)
│   ├── documents/   # per-document structural index + classifications
│   ├── concepts.yaml
│   └── graph.yaml
├── specs/           # existing document storage
├── plans/           # existing document storage
└── cache/           # existing derived cache
```

Context files follow the same principles as entity files: schema-validated YAML, deterministic serialisation, Git-tracked, tool-written.

### 13.1 Knowledge entry fields

Knowledge entries (Tier 2 and Tier 3) carry lifecycle and retention metadata:

```yaml
id: KE-042
tier: 2
topic: api-json-naming-convention
scope: backend
content: "API responses use camelCase for JSON field names. Nested objects also use camelCase."
learned_from: TASK-152.3

# Lifecycle
status: confirmed         # contributed | confirmed | disputed | stale | retired

# Retention metadata
created: 2025-01-15
last_used: 2025-01-20
use_count: 4
miss_count: 0
confidence: 0.83          # Wilson score lower bound over (use_count, miss_count)
ttl_days: 90              # tier-dependent; resets on use

# Provenance
promoted_from: SE-017     # if promoted from Tier 3
merged_from: []           # entry IDs merged during compaction
deprecated_reason: null
git_anchors:              # optional file paths for staleness detection
  - internal/api/handler.go
```

### 13.2 Role profile fields

Role profiles include an `inherits` field for the profile inheritance chain:

```yaml
id: backend
inherits: developer
scope:
  description: "Backend development — core logic, MCP service layer, CLI"
  packages:
    - internal/core
    - internal/mcp
    - cmd/kanbanzai
conventions:
  - "Error handling: use internal/errors.Wrap, never bare fmt.Errorf"
  - "Tests: table-driven, no mocks for pure functions"
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
- **Design the storage layout** to accommodate context entries alongside entities, including the `context/roles/` hierarchy and knowledge entry schema

Phase 1 does not implement context profiles, assembly, or contribution.

### 15.2 Phase 2

Phase 2 implements retrieval and context packing (per the workflow design basis §24). This includes:

- Context profile definition, inheritance resolution, and retrieval
- Context assembly for tasks with byte-based budgeting and tiered retrieval
- Knowledge contribution with deduplication on write
- Confidence scoring (Wilson score lower bound) and tier-dependent filtering
- Knowledge lifecycle management (contribution → confirmation → staleness → retirement)
- Usage reporting bundled with task completion
- Retention policy enforcement (TTL-based pruning, promotion triggers)
- Post-merge compaction for Tier 3 entries
- Integration of design context and implementation context in assembled packets
- Document intelligence backend — structural analysis, AI-assisted classification, and the document graph that provides design fragments for context assembly (see `work/design/document-intelligence-design.md` §18.2)

### 15.3 Phase 3 and beyond

Later phases may add:

- Automatic context assembly optimisation (learning which context entries are most useful)
- Cross-project knowledge sharing (conventions that apply to all projects by a team)
- Orchestration-driven decomposition informed by context profiles
- Embedding-based semantic similarity for deduplication (if algorithmic dedup proves insufficient)
- Integration with code-aware retrieval tools (e.g., codebase-memory-mcp) for hybrid knowledge retrieval
- Confidence score time decay (reducing confidence for entries that haven't been used recently)

---

## 16. Aesthetic Constraints

Per the Parsley aesthetic, this design should remain:

- **Simple** — context profiles are named YAML files. No ML pipelines, no embedding databases, no knowledge graphs. Structured files, queried through MCP operations.
- **Minimal** — start with a small number of operations (get, contribute, assemble). Add complexity only when demonstrated need arises.
- **Complete** — the design-context + implementation-context model covers the full space of knowledge an agent needs. No obvious gaps in the model.
- **Composable** — context profiles compose with the entity hierarchy and document fragments. The same MCP interface serves orchestrators and specialists differently through scoping parameters, not separate APIs.

---

## 17. Resolved Design Questions

The following questions were originally deferred as open questions. They have been resolved and their answers are incorporated into the relevant sections of this document.

1. **Token budget management** (resolved → §8.3) — the system budgets in bytes, not tokens. The MCP protocol does not expose the calling model's identity, context window, or tokenizer. There is no universal tokenizer across model families, and server-side token counting would be inaccurate. The system sets a configurable byte ceiling (default 30KB per packet) and provides tiered retrieval so agents pull more detail on demand.

2. **Knowledge deduplication** (resolved → §9.2, §9.6) — deduplication operates at two points. On contribution: exact topic match rejects, Jaccard word-set similarity > 0.65 flags near-duplicates. On merge: post-merge compaction detects exact duplicates (keep highest confidence), near-duplicates (auto-merge or flag), and contradictions (mark `disputed`, never auto-resolve). No embedding infrastructure is required — algorithmic comparison is sufficient at realistic scales.

3. **Cross-role knowledge** (resolved → §6.6) — profile inheritance solves this. Knowledge that applies to multiple roles but not all lives at the appropriate ancestor in the inheritance tree. A `developer` profile is inherited by both `backend` and `frontend`. A `base` profile is inherited by all roles. Agents contributing knowledge specify the scope; coordinators review whether the scope is appropriate.

4. **Confidence scoring** (resolved → §9.4) — yes, knowledge entries carry confidence scores. The system uses Wilson score lower bound over `(use_count, miss_count)`, which handles low sample sizes gracefully. Confidence is used as a tier-dependent filter during context assembly (Tier 1: no filter, Tier 2: > 0.3, Tier 3: > 0.5). New entries start at 0.5 (uncertain) and converge as they accumulate usage data.

5. **Session knowledge pruning** (resolved → §9.5) — retention follows a generational cache model. Tier 3 entries have a 14–30 day TTL that resets on use. Entries with low use counts are pruned on TTL expiry. Entries found wrong are immediately retired (miss_count ≥ 2). Entries that prove consistently useful are promoted to Tier 2 (use_count ≥ 5, miss_count = 0). Usage reporting is bundled with task completion at negligible token cost (~50–100 tokens per report).

6. **Context profile inheritance** (resolved → §6.6) — yes, profiles inherit via an `inherits` field. The system walks the inheritance chain and layers each profile's content in order, with more specific profiles overriding more general ones. Projects define their own inheritance trees. The system enforces only that references resolve and do not form cycles.

7. **Integration with code-aware tools** (resolved → §15.3, deferred to Phase 3+) — code-aware retrieval tools like codebase-memory-mcp are complementary, not competitive. They manage code knowledge; Kanbanzai manages workflow and project knowledge. In Phase 1 and Phase 2, integration is limited to role profiles informing agents about available tools ("you have access to codebase-memory-mcp; use it for X"). Deeper integration (hybrid retrieval combining Kanbanzai knowledge entries with embedding-based code search) is a Phase 3+ exploration.

---

## 18. Summary

Kanbanzai manages two types of context for AI agent teams:

- **Design context** — what to build — assembled from the document-to-entity hierarchy (already designed)
- **Implementation context** — how to build it — structured in three tiers (project conventions, architecture knowledge, session knowledge) and scoped by role profiles (new in this document)

The system assembles both into targeted context packets, delivered through MCP operations, so that each agent receives precisely the knowledge it needs — no more, no less.

Agents both consume and produce context through a feedback loop, building a persistent knowledge store that makes each subsequent agent session more efficient than the last.

The result is a system where AI agent teams can approach the efficiency of well-coordinated human teams: specialists with deep domain knowledge, orchestrators with broad situational awareness, and a shared knowledge base that grows with the project.