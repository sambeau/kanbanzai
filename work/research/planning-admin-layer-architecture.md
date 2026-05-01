# Planning & Admin Layer: Architecture and Product Boundary Analysis

| Field  | Value                         |
|--------|-------------------------------|
| Date   | 2026-05-01T12:02:17Z          |
| Status | Draft                         |
| Author | researcher                    |

## Research Question

Given that a planning and administration layer upstream of Kanbanzai's orchestrator is worth building (established in prior research), what
architecture should it adopt? Specifically: (1) should it be a standalone product or a layer within Kanbanzai? (2) should it use a shared data
store or maintain its own? (3) should it be served by a separate MCP server or the existing `kanbanzai serve`? (4) what agent interaction model
best supports humans forming plans? and (5) how should planning state be represented on disk?

This report extends the prior Planning and Administration Layer exploration and should be read alongside it.

## Scope and Methodology

**In scope:**
- Product boundary analysis: standalone product vs. layer within Kanbanzai vs. hybrid with server mediation
- Data store design: shared `.kbz/` store vs. separate `.plan/` sidecar vs. server-centric
- MCP server topology: one server vs. two
- Agent interaction model: chat-based vs. API-fired vs. hybrid
- On-disk representation of roadmaps, estimates, priorities, and proposals
- Multi-human collaboration constraints

**Out of scope:**
- Specific MCP tool signatures
- Visual UI design details
- Build-vs-buy for a visual planning interface
- Timeline, staffing, or resourcing decisions

**Methodology:** Architectural trade-off analysis informed by the existing Kanbanzai system architecture,
the `.kbz/` state model, MCP server design, and prior research. Evidence is drawn from internal system
analysis, prior research findings, and structured brainstorming.

## Findings

### Finding 1: Three Product Boundary Models, One Convergence Point

Three architectures for the planning layer relative to Kanbanzai were analysed:

**Model A: Wholly Separate Product.** The planning tool is a standalone application with its own
data store, its own UI, and its own agent surface. At handoff time, committed plans are exported
to Kanbanzai. Post-handoff, the plan lives in Kanbanzai's execution system; changes in execution
do not flow back, or flow back as read-only status.

**Model B: Hybrid — Shared Store, Different UIs.** The planning tool and Kanbanzai share the
`.kbz/` store. Both operate on the same entities (plans, batches, features, tasks) but through
different interfaces — a visual planning UI vs. the CLI/MCP agent surface. No sync is needed
because there is a single source of truth.

**Model C: Hybrid — Kanbanzai Server as Mediator.** The planning UI and Kanbanzai agents both
talk to the Kanbanzai server, which mediates writes to the `.kbz/` store. The server prevents
git conflicts from concurrent access and provides real-time collaboration. The filesystem remains
canonical; the server is a write mediator and read cache.

**Analysis:**

| Criterion | Model A (Separate) | Model B (Shared Store) | Model C (Server-Mediated) |
|-----------|-------------------|------------------------|---------------------------|
| UI freedom | High — unconstrained | Medium — bound by `.kbz/` schema | Medium — bound by `.kbz/` schema |
| Sync complexity | High — two data stores | None — single store | None — single store |
| Multi-human collaboration | Depends on planning tool's backend | Low — git conflicts on YAML | High — server mediates writes |
| Git-native philosophy | Planning tool abandons it | Preserved | Preserved with server augmentation |
| Implementation cost | Highest — two products | Lowest (extend `.kbz/`) | Medium (server already exists) |
| Conceptual cleanliness | High — planning ≠ execution | Medium — entities serve two masters | Medium — server is a dependency |

Model A has the strongest conceptual separation — planning and execution use different cognitive
modes (divergent vs. convergent, top-down vs. bottom-up, ambiguous vs. structured). This is a
genuine architectural argument for separation. However, the sync problem is real: plans change
during execution, execution reveals new information that changes plans, and the boundary is
porous. A one-way handoff means plans rot immediately; a two-way handoff recreates the sync
problem Model A is trying to avoid.

Model C is the likely convergence point because planning and execution are not truly separate
activities. The server mediation pattern already exists in Kanbanzai (agents use MCP tools that
write to the filesystem through `kanbanzai serve`). Extending this to a planning UI is
incremental rather than architectural.

Model A is valuable as a design prototype — build the planning UX standalone to validate
concepts without being constrained by `.kbz/` internals, then integrate later.

Source: Architectural analysis — internal system architecture (`internal/mcp/`, `.kbz/state/`),
brainstorm synthesis. Primary, current.

### Finding 2: A `.plan/` Sidecar Separates Planning from Execution State Without Adding a New Store

Rather than creating a separate database or extending `.kbz/` with planning concepts that don't
fit its execution-oriented schema, a `.plan/` directory committed to git can hold planning state:

```
my-project/
├── .plan/                       # Planning state (committed)
│   ├── roadmap.yaml             # Human-editable roadmap
│   ├── estimates.yaml           # Rough estimates (pre-Fibonacci-commitment)
│   ├── proposals/               # Draft proposals before they become features
│   ├── priorities.yaml          # Priority ordering with rationale
│   └── dependencies.yaml        # Cross-project dependency notes
├── .planconfig.yaml             # Local planning config (NOT committed)
└── .kbz/                        # Execution state
```

**What lives in `.plan/`**: Draft proposals, roadmaps, rough estimates, planning priorities,
research notes — information that is being *shaped* but has not been committed to execution.

**What lives in `.kbz/`**: Feature lifecycle state, task state, committed (Fibonacci) estimates,
approved documents, roles & skills — information that is *committed* to execution.

**The handoff is a one-way promotion**: planning state becomes execution state. When a proposal
graduates, it moves from `.plan/proposals/` to `.kbz/state/features/` as a registered feature.
After handoff, the execution system owns it. The planning system can still reference it (for
roadmap views) but doesn't own it.

**`.planconfig.yaml` is NOT committed**: It contains local view preferences, personal reminders,
cached computed state, and last-synced timestamps. This is per-human configuration, not project state.

**Advantages of this approach:**
1. Planning state stays near code — `grep`-able, versionable, diffable
2. Clean separation of concerns — `.kbz/` isn't polluted with half-formed ideas
3. No sync problems — the filesystem IS the sync mechanism via git
4. Humans can use any editor — proposals are just markdown
5. The planning assistant can still have a server cache for real-time features
6. Gradual migration path — start with files, add server cache when needed, add visual UI later

**Open questions:**
- Is `.plan/` too similar to `.kbz/`? If both are YAML state files, the separation might create
  more friction than it solves if the boundary is fuzzy.
- How do multiple humans edit `.plan/` concurrently? Git merge conflicts on YAML are painful.
  Options: (a) planning is solo with async review, (b) server mediates writes, (c) CRDT-friendly format.
- What happens to `.plan/` state after handoff? Does it persist as historical record, or get
  archived to avoid confusion? Long-term strategic plans may *never* be fully handed off — they
  span multiple Kanbanzai Plans and remain active indefinitely.

Source: Architectural brainstorming, internal system analysis. Primary, current.

### Finding 3: The Agent Interaction Model Should Be Hybrid — Chat Surface, API Engine

Two poles define the agent interaction design space:

**Pole 1: Chat-Based Agent.** The planning assistant is a conversational agent with planning-specific
skills. Humans describe intent naturally; the agent structures, questions, and refines through
conversation. This is warm, flexible, and handles ambiguity well — but struggles with batch
operations (reorder 30 roadmap items) and lacks visual/spatial reasoning.

**Pole 2: API-Fired Agent.** The planning UI sends structured commands (`analyze_dependencies`,
`estimate_batch`, `suggest_ordering`) to an agent that returns structured results. The UI renders
results visually. This is predictable, composable, and batch-capable — but loses conversational
warmth and is less flexible for ambiguous requests.

**Synthesis:** The right answer is both, in different contexts:

| Mode | Best for | Examples |
|------|----------|----------|
| **Chat** | Capture, clarification, exploration, reminders | "I've been thinking about search improvements..." |
| **API** | Dependency analysis, estimation, roadmap rendering, handoff | Batch computation of cross-plan dependencies |

The planning assistant has a **chat interface** for human interaction but invokes **structured
agent functions** behind the scenes. The human sees conversation; the system sees API calls.
This maps to Kanbanzai's existing architecture: the planning assistant is an agent with
planning-specific skills that use tools (like `decompose`, `estimate`, `entity` queries) to do
their work. The chat is the UX wrapper; the tools are the engine.

This hybrid model also addresses the "two MCP servers" question: if the planning assistant
uses the same MCP tools (extended with planning-specific ones), a single `kanbanzai serve`
process can serve both the execution agents and the planning assistant. Two separate MCP
servers would only be needed if the planning assistant became a fully separate product
(Model A).

Source: Architectural analysis, brainstorm synthesis. Primary, current.

### Finding 4: Existing Kanbanzai Entities Can Represent Most Planning Concepts

An important question is whether the `.kbz/` store needs extending for planning concepts, or whether
existing entities are flexible enough. Analysis suggests most planning concepts can be expressed in
existing entities:

| Planning Concept | Existing Entity Representation |
|------------------|-------------------------------|
| Draft proposal | Feature in `idea` status |
| Roadmap | Plan with features in priority order |
| Rough estimate | Estimate entity (pre-commitment, marked as draft) |
| Dependency | Entity dependency relationship (`depends_on`) |
| Priority ordering | Entity ordering + priority field |
| Strategic initiative | Plan (strategic, recursive) |
| Research question | Feature with `type: research` or dedicated research document |

If planning is just a different way of viewing and manipulating these entities during their
formative stages, the store problem largely disappears. The planning tool becomes a specialized
UI for interacting with Kanbanzai entities before they reach execution-ready state.

However, two gaps remain:
1. **Capture inbox** — raw, unstructured intent that hasn't been classified yet. This doesn't
   map cleanly to any existing entity type. A "proposal" entity type or a dedicated proposals
   directory in `.plan/` would fill this gap.
2. **Planning-internal documents** — research notes, brainstorming, rough proposals that aren't
   ready for the execution system. These are better kept in `.plan/` or the planning tool until
   approved, at which point they become registered Kanbanzai documents.

The boundary is **approval**: before approval, documents and state live in the planning domain.
After approval, they're registered in Kanbanzai. This gives Model A's clean separation with
Model C's shared canonical store for committed work.

Source: Internal system analysis — entity model (`internal/model/`), entity lifecycle states,
document registration flow (`doc` tool). Primary, current.

### Finding 5: Multi-Human Collaboration Is the Hardest Constraint

The strongest argument for Model C (server-mediated) is multi-human collaboration. If multiple
humans need to plan together in real time — co-editing a roadmap, discussing priorities,
reordering features — a file-based git store with merge conflicts is a non-starter for the
planning UX.

Three collaboration models were considered:

1. **Server-mediated writes**: The Kanbanzai server handles concurrent access, preventing git
   conflicts. This requires the server to be running and accessible during planning sessions.
2. **CRDT-based local state with eventual consistency**: Complex but git-friendly. Solves the
   offline-collaboration problem but at high implementation cost.
3. **Single-human planning, multi-human review**: Planning is solo work (one human drives the
   conversation with the AI assistant), review is async via git-based document approval and
   pull requests. This maps well to how humans naturally plan and avoids the concurrency problem
   entirely.

Option 3 (single-human planning) is the simplest viable approach for an MVP. It matches how
Kanbanzai already handles document approval (human gates with async review) and avoids
introducing real-time collaboration complexity into the initial implementation. Server-mediated
collaboration can be added later when the need is demonstrated.

Source: Architectural analysis, Kanbanzai document approval flow (`doc(action: approve)`),
stage gate design (`.kbz/stage-bindings.yaml`). Primary, current.

## Trade-Off Analysis

| Criterion | `.plan/` Sidecar (filesystem) | Extended `.kbz/` (shared store) | Server-Centric (DB) |
|-----------|------------------------------|--------------------------------|---------------------|
| Git-native | Full — files committed to git | Full — YAML in git | Partial — server owns state, git is backup |
| Offline planning | Full — edit files in any editor | Full — edit YAML directly | None — requires server |
| Multi-human collab | Low — git conflicts on YAML | Low — git conflicts on YAML | High — server mediates |
| Schema flexibility | High — loose YAML/Markdown | Medium — validated entity schema | Low — database schema |
| Implementation cost | Low | Low (extend existing) | High (build/maintain server DB) |
| Integration with execution | Manual (handoff) | Automatic (same entities) | Automatic (server owns both) |
| Discoverability | Low — grep files, no structured query | High — MCP tools query entities | High — server API |

The `.plan/` sidecar wins on simplicity, git-nativeness, and implementation cost. The extended
`.kbz/` wins on integration with execution. The server-centric approach wins on collaboration
but at high cost and with philosophical tension (Kanbanzai is explicitly git-native).

## Recommendations

### Recommendation 1: Start with the `.plan/` Sidecar Pattern

**Recommendation:** Adopt a `.plan/` directory committed to git for planning state (proposals,
roadmaps, rough estimates, priorities). Keep `.planconfig.yaml` uncommitted for local preferences.
Use `.kbz/` for committed execution state only. The handoff is a one-way promotion from `.plan/`
to `.kbz/` when a proposal graduates to a feature.

**Confidence:** Medium
**Based on:** Findings 1, 2, and 4
**Conditions:** This recommendation assumes planning is primarily a solo activity with async review.
If real-time multi-human planning collaboration becomes a demonstrated need, the `.plan/` sidecar
will need augmentation (server-mediated writes or CRDT). The sidecar can evolve into a
server-cached model without changing the fundamental file structure.

### Recommendation 2: Build Planning MCP Tools Within the Existing Server

**Recommendation:** Extend `kanbanzai serve` with planning-specific MCP tools rather than
creating a separate planning MCP server. The planning assistant uses the same server as execution
agents. This keeps a single server surface, avoids the dual-product complexity of Model A, and
allows planning tools to query execution state naturally (for dependency surfacing, progress
synthesis).

**Confidence:** High
**Based on:** Findings 1 and 3
**Conditions:** A separate planning MCP server would only be warranted if the planning tool
becomes a fully standalone product with its own deployment and users who don't use Kanbanzai
for execution. This is unlikely in the near term.

### Recommendation 3: Adopt the Hybrid Agent Interaction Model

**Recommendation:** The planning assistant should present a chat interface to humans but invoke
structured MCP tool functions behind the scenes. Chat mode for capture, clarification, exploration,
and reminders. API mode (structured tool calls) for dependency analysis, estimation, roadmap
computation, and handoff. A single `kanbanzai serve` process serves both modes.

**Confidence:** High
**Based on:** Finding 3
**Conditions:** If the planning UI becomes a visual tool, the "chat" vs. "API" distinction maps
to "conversation pane" vs. "visual canvas" — the hybrid model still applies.

### Recommendation 4: Defer Multi-Human Collaboration to Post-MVP

**Recommendation:** Design the MVP for single-human planning with async review via git-based
document approval. This matches Kanbanzai's existing human gate pattern. If real-time
collaboration becomes a demonstrated need, introduce server-mediated writes to `.plan/` files
(or migrate to a server-cached model) at that point.

**Confidence:** Medium
**Based on:** Finding 5
**Conditions:** This is a deliberate deferral. If early use reveals that planning is inherently
collaborative (multiple stakeholders shaping a roadmap together in real time), this constraint
would need to be revisited sooner. A small-scale test with two humans attempting to plan together
using `.plan/` + git would provide data.

### Recommendation 5: Use Existing Entities Where Possible, Add Minimal New Types

**Recommendation:** Model proposals as features in `idea` status, roadmaps as plans with ordered
features, rough estimates as estimate entities (marked draft), and dependencies as entity
relationships. Add only one new concept if needed: a lightweight "capture" or "proposal" record
for unstructured intake that hasn't been classified yet. Keep new entity types to the absolute
minimum — the `.plan/` sidecar handles planning-internal state that doesn't fit existing entities.

**Confidence:** Medium
**Based on:** Finding 4
**Conditions:** If planning concepts turn out to require significantly different lifecycle states
or relationships than existing entities support, a dedicated planning entity type may be warranted.
The "capture inbox" concept (unstructured intake before classification) is the most likely
candidate for a new entity type.

## Limitations

- **No implementation validation:** The `.plan/` sidecar pattern is an architectural hypothesis.
  It has not been tested against real planning workflows. The boundary between `.plan/` and `.kbz/`
  may prove fuzzier in practice than in theory, and the "one-way promotion" handoff may encounter
  friction when plans change during execution.

- **Git merge conflicts on YAML remain a risk:** If `.plan/` files are edited by multiple humans
  (even asynchronously), merge conflicts will occur. The single-human-planning assumption defers
  this risk but does not eliminate it — even a single human working across two machines could
  encounter conflicts.

- **The visual UI question is deferred:** This analysis focuses on data architecture and agent
  interaction. The question of whether a visual planning UI is needed and what form it should
  take is deferred to further research. The `.plan/` sidecar pattern is compatible with both
  a terminal-based planning experience and a visual UI layer.

- **Scope does not include role design:** This report does not address whether the planning
  assistant should be a named role (Admin Assistant), decomposed functional roles, or a
  tools-only surface. That question is addressed in the prior Planning and Administration
  Layer research report and remains open pending tool-shape validation.

- **Product strategy not evaluated:** The recommendation to build within Kanbanzai rather than
  as a separate product is architectural, not strategic. A separate planning product might have
  different market dynamics, user acquisition paths, or revenue models. This analysis is
  scoped to technical architecture only.
