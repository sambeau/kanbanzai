# Kanbanzai 2.0: MCP Server Design Vision

> **Purpose:** Establish the design direction for Kanbanzai 2.0, driven by field feedback from 1.0 and research into MCP protocol patterns.
>
> **Created:** 2026-03-26T18:18:32Z
> **Status:** Draft
> **Audience:** Design team (human + AI)
> **Prerequisite reading:** `work/reports/mcp-server-design-issues.md`

---

## 1. What Is Kanbanzai and Why Does It Exist

Kanbanzai is a Git-native workflow system for human–AI collaborative software development. Its purpose is to coordinate AI agent teams to efficiently turn designs into working software.

The system exists because of a specific problem encountered at scale: on a project with 149 specification documents (and a comparable number of designs, plans, and reports), the development process broke down. Agents burned tokens re-reading documents they'd already seen. Designs were forgotten. Search produced too many false positives to be useful. Work couldn't be parallelised because there was no coordination layer. The human lost visibility and control.

Kanbanzai was built to solve this. The core idea remains sound:

- **Humans own intent** — goals, priorities, approvals, product direction
- **AI agents own execution** — decomposing work, implementing, verifying, tracking status
- **Documents are the human interface** — humans write and review design documents, make decisions in conversation, and approve results
- **The system mediates** — extracting decisions, assembling targeted context, maintaining consistency, and enforcing workflow

Version 1.0 validated the workflow model but revealed a fundamental problem with how the system presents itself to agents. This document sets the design direction for fixing that.

---

## 2. Who Are the Users

Kanbanzai has three distinct user types. The 2.0 design must serve all three.

### 2.1 The Human (Designer / Manager)

**What they do:** Write and approve design documents. Set priorities. Make architectural decisions. Review results. Monitor progress.

**What they need:**
- A document workflow that works (this already works well in 1.0)
- Visibility into project state without asking an agent (a viewer — addressed separately)
- Confidence that the system is orchestrating efficiently (speed, token economy)
- Minimal friction — not babysitting permission prompts for routine operations

**What they don't need:** To interact with entity CRUD, lifecycle transitions, or internal tooling. The system should be invisible to the human except through documents and a viewer.

### 2.2 The Orchestrator Agent

The orchestrator reads plans and specs, decomposes work, assigns tasks to worker agents, monitors progress, and handles exceptions. In practice, this is often a single agent session that may spawn sub-agents.

**What they do:** Understand the plan. Decide what to work on next. Claim tasks. Delegate to workers. Track completion. Report status.

**What they need:**
- A situation picture — the state of a plan, feature, or project in one query
- A way to claim work and get full context for it
- A way to delegate work to a sub-agent with everything the sub-agent needs
- A way to record completion and what was learned
- Batch operations — orchestrators work on sets of entities, not individuals

**What they don't need:** To manage lifecycle transitions manually. To call separate tools for knowledge contribution, duplicate checking, or context assembly. To understand the internal entity model.

### 2.3 The Worker Agent

A worker agent is spawned by an orchestrator to perform a specific task: implement a feature, write documentation, fix a bug. It has a short lifespan and a narrow scope.

**What they do:** Receive context. Do the work. Report results.

**What they need:**
- Everything they need to know, up front, in their initial prompt
- The relevant spec sections (not the full document)
- Known constraints and conventions (from knowledge entries)
- Clear acceptance criteria
- A way to report what they did and what they learned

**What they don't need:** To query the workflow system at all. Their context should arrive pre-assembled. If a worker agent needs to call Kanbanzai tools, the handoff was incomplete.

### 2.4 The Implication

The worker agent should ideally never call a Kanbanzai tool. The orchestrator should call a small number of high-level tools. The human should interact through documents and a viewer, not tools.

This means the MCP tool surface is primarily an **orchestrator interface**. It should be designed for how orchestrators think and work — in workflows operating on sets of entities — not as a generic CRUD API.

---

## 3. What Went Wrong with 1.0

The full diagnosis is in `work/reports/mcp-server-design-issues.md`. The summary:

**97 tools, designed from the system's perspective.** The API asked "what operations exist on what entity types?" and created one tool per answer. The result was a CRUD surface that maps to the storage layer, not to agent workflows.

**No batch operations.** Orchestrators routinely touch N entities of the same kind. Each operation was a separate round-trip.

**Silent side effects.** The system performed correct cascade transitions but didn't tell the caller what it did.

**Ceremony over substance.** Mandatory parameters and strict status prerequisites created friction that agents routed around — falling back to grep and direct YAML editing.

**Intelligence and knowledge as explicit tools.** Document intelligence (9+ tools) and knowledge management (12+ tools) required voluntary invocation. Agents never called them. The capabilities are valuable at scale but the tool-per-operation exposure was wrong.

**Context window tax.** All 97 tool schemas loaded into every session, consuming thousands of tokens before any work began. Most sessions used fewer than 10 tools.

---

## 4. Design Principles for 2.0

These principles govern the redesign.

### P1: Workflow tools, not entity tools

Tools should map to what an agent wants to *do*, not what entities exist in the system. "I'm done with this task" is one thought; it should be one tool call — not "update status on entity type task with ID X to status done."

### P2: Batch by default

Every tool that accepts an entity ID should also accept an array of entity IDs. Single operations are arrays of length 1. This is the pattern used by the MCP reference Memory server and it eliminates the N-round-trip problem.

### P3: Side effects are first-class responses

Every mutation that triggers a cascade must surface what it changed. If the system advanced a feature because a spec was approved, the response says so. The rule: if the system changed something you didn't explicitly ask it to change, the response tells you.

### P4: Invisible infrastructure, not exposed tools

Document intelligence and knowledge management are infrastructure. They power the workflow tools from the inside. They do not need their own tool surface. An agent should never need to call `doc_classify` or `knowledge_contribute` directly.

### P5: Context arrives, not gets fetched

Worker agents receive their context at spawn time. The orchestrator's handoff tool assembles everything the worker needs: spec sections, knowledge entries, file references, acceptance criteria, constraints. The worker never queries the system.

### P6: Respect the context budget

Every tool schema that sits in the context window must earn its space by being used. Fewer tools with richer responses. Feature groups to load only what's needed. The target is 10–30 tools, not 97.

### P7: Scale-ready, small-project-friendly

The system must handle 149+ specs without degradation, but must not impose the overhead of that scale on a 13-spec project. Feature groups and invisible infrastructure achieve this: the intelligence layer does its work regardless of project size, but small projects don't feel the weight of it.

### P8: The internals are sound

The YAML-on-disk model, lifecycle state machines, document-driven stage gates, knowledge confidence scoring, and Git-native storage are validated by 1.0 usage. This is a tool surface redesign, not a ground-up rewrite. The implementation strategy should preserve the internal architecture and change how it's exposed.

---

## 5. The Tool Surface

### 5.1 Design Approach: Resource-Oriented Tools with Feature Groups

Based on MCP protocol research, Tools are the correct primitive for agent-facing operations. Resources (application-controlled) and Prompts (user-initiated) serve human interfaces but not agent workflows.

The tool surface uses two organising ideas:

1. **Resource-oriented tools** — one tool per domain (entity, document, knowledge) with an `action` parameter, rather than one tool per operation per entity type
2. **Feature groups** — tools organised into groups that can be enabled/disabled via configuration, following the pattern proven by Supabase's MCP server

### 5.2 Core Tools (Always Loaded)

These tools are the orchestrator's primary interface. They should cover 90% of a typical orchestration session.

#### `status`

The dashboard tool. Returns a synthesised view of project, plan, or feature state.

```
status()                          → project overview: active plans, health, what needs attention
status(plan_id)                   → plan dashboard: features by status, task rollup, blockers, what's next
status(feature_id)                → feature detail: tasks by status, documents, gaps, worktree
status(task_id)                   → task detail: dependencies, parent feature context
```

Returns synthesis, not raw records. "12 done, 3 ready, 2 blocked" — not 17 entity records.

#### `next`

Claim work and get full context. Combines work queue inspection, task claiming, and context assembly.

```
next()                            → show the ready queue (like work_queue today)
next(task_id)                     → claim this specific task, return assembled context
next(plan_id)                     → claim the top ready task in this plan
next(feature_id)                  → claim the top ready task in this feature
```

When claiming, the response includes: task summary, relevant spec sections (extracted by document intelligence), known constraints (from knowledge entries), file references, acceptance criteria, and a pre-formatted sub-agent prompt. Context assembly is automatic, not a separate call.

The `role` parameter is optional. If provided, role-specific knowledge entries are included.

#### `finish`

Record completion. Handles status transitions, completion metadata, and knowledge contribution in one call.

```
finish(task_id, summary, files_modified?, knowledge?)
finish([{task_id, summary, files_modified?, knowledge?}, ...])    → batch completion
```

Accepts tasks in `ready` or `active` status — lenient on lifecycle, strict on data. If the task was never formally claimed, `finish` transitions it through the necessary states internally.

Knowledge entries are accepted inline as an optional parameter. This is the natural moment for contribution — "here's what I learned while doing this work" — and it removes the need for a separate `knowledge_contribute` call.

#### `handoff`

Generate a sub-agent prompt from a task. This is the bridge between Kanbanzai's dispatch model and `spawn_agent`.

```
handoff(task_id, role?)           → returns a complete, ready-to-use prompt string
```

The output is designed to go directly into `spawn_agent`'s `message` parameter. It includes: task summary, relevant spec sections, acceptance criteria, existing code context, file paths, constraints, knowledge entries, and commit message format. Everything the worker needs, pre-assembled.

#### `entity`

Generic CRUD for all entity types. One tool replaces the 17+ entity-specific tools.

```
entity(action: create, type: task, parent_feature: "FEAT-...", tasks: [{slug, summary}, ...])
entity(action: get, type: feature, id: "FEAT-...")
entity(action: list, type: task, parent: "FEAT-...", status: "ready")
entity(action: update, type: task, id: "TASK-...", status: "done")
entity(action: create, type: plan, prefix: "P", slug: "...", title: "...", summary: "...")
entity(action: create, type: decision, slug: "...", summary: "...", rationale: "...")
```

The `create` action accepts arrays (batch by default). Creating 31 tasks is one call, not 31.

The `update` action with a `status` field handles lifecycle transitions, replacing the separate `update_status` tool.

#### `doc`

Document registration, approval, and querying. One tool replaces the 11+ document record tools.

```
doc(action: register, path: "work/spec/...", type: "specification", owner: "FEAT-...")
doc(action: register, documents: [{path, type, owner}, ...])      → batch registration
doc(action: approve, id: "...")
doc(action: approve, ids: ["...", "..."])                          → batch approval
doc(action: get, id: "...")
doc(action: list, type: "specification", owner: "FEAT-...")
doc(action: content, id: "...")
doc(action: gaps, feature_id: "FEAT-...")
```

Approval responses include side effects: "Feature FEAT-... transitioned from specifying to dev-planning."

Accepts paths as well as IDs where possible — agents know paths, they don't always know document record IDs.

#### `health`

Single diagnostic entry point, unchanged from 1.0.

```
health()                          → comprehensive project health check
```

### 5.3 Feature Group: Planning

Loaded when the orchestrator is decomposing features into tasks.

#### `decompose`

Feature decomposition, review, and task creation in a streamlined pipeline.

```
decompose(feature_id)                                   → propose task breakdown
decompose(feature_id, action: review, proposal: {...})  → review a proposal
decompose(feature_id, action: apply, proposal: {...})   → create tasks from proposal
decompose(feature_id, action: slice)                    → vertical slice analysis
```

The `apply` action creates all tasks from the proposal in one call, replacing the `decompose` → `review` → N × `create_task` pipeline.

Works without requiring pre-classified documents — falls back to full document content if Layer 3 classification isn't available.

#### `estimate`

Estimation for entities.

```
estimate(entity_id, points: 5)                          → set estimate
estimate(entity_id)                                     → query estimate and rollup
estimate(entities: [{id, points}, ...])                 → batch estimation
```

#### `conflict`

Conflict analysis for parallel work.

```
conflict(task_ids: ["TASK-...", "TASK-..."])             → conflict risk assessment
```

### 5.4 Feature Group: Knowledge

Loaded when explicitly managing the knowledge base. Note: routine knowledge contribution happens inline via `finish`, and routine knowledge retrieval happens automatically via `next` and `handoff`. These tools are for direct knowledge management.

#### `knowledge`

```
knowledge(action: list, scope: "project", tags: [...])
knowledge(action: get, id: "KE-...")
knowledge(action: contribute, topic: "...", content: "...", scope: "...")
knowledge(action: confirm, id: "KE-...")
knowledge(action: flag, id: "KE-...", reason: "...")
knowledge(action: retire, id: "KE-...", reason: "...")
knowledge(action: compact, scope: "...")
knowledge(action: prune)
```

#### `profile`

Context profile management.

```
profile(action: list)
profile(action: get, id: "backend")
```

### 5.5 Feature Group: Git

Loaded when working with worktrees, PRs, and merge operations.

#### `worktree`

```
worktree(action: create, entity_id: "FEAT-...")
worktree(action: get, entity_id: "FEAT-...")
worktree(action: list, status: "active")
worktree(action: remove, entity_id: "FEAT-...")
```

#### `merge`

```
merge(action: check, entity_id: "FEAT-...")             → readiness check
merge(action: execute, entity_id: "FEAT-...")            → execute merge
```

#### `pr`

```
pr(action: create, entity_id: "FEAT-...")
pr(action: status, entity_id: "FEAT-...")
pr(action: update, entity_id: "FEAT-...")
```

#### `branch`

```
branch(action: status, entity_id: "FEAT-...")            → branch health
```

#### `cleanup`

```
cleanup(action: list)
cleanup(action: execute, worktree_id: "WT-...")
```

### 5.6 Feature Group: Documents

Loaded when working with document intelligence directly. Note: document intelligence powers `next`, `handoff`, and `status` invisibly. These tools are for explicit document exploration — useful for research tasks or when onboarding to an unfamiliar project.

#### `doc_intel`

```
doc_intel(action: outline, id: "...")
doc_intel(action: section, id: "...", path: "1.2")
doc_intel(action: find, concept: "payment retry")
doc_intel(action: find, entity_id: "FEAT-...")
doc_intel(action: find, role: "requirement")
doc_intel(action: trace, entity_id: "FEAT-...")
doc_intel(action: classify, id: "...", ...)
```

### 5.7 Feature Group: Incidents

Loaded when managing incidents.

#### `incident`

```
incident(action: create, slug: "...", title: "...", severity: "...", ...)
incident(action: update, id: "...", status: "...", ...)
incident(action: list, status: "...", severity: "...")
incident(action: link_bug, incident_id: "...", bug_id: "...")
```

### 5.8 Feature Group: Checkpoints

Loaded when human decision points are needed in automated workflows.

#### `checkpoint`

```
checkpoint(action: create, question: "...", context: "...", ...)
checkpoint(action: get, id: "CHK-...")
checkpoint(action: respond, id: "CHK-...", response: "...")
checkpoint(action: list, status: "pending")
```

### 5.9 Tool Count Summary

| Group | Tools | Always loaded? |
|-------|-------|---------------|
| **Core** | `status`, `next`, `finish`, `handoff`, `entity`, `doc`, `health` | Yes |
| **Planning** | `decompose`, `estimate`, `conflict` | No |
| **Knowledge** | `knowledge`, `profile` | No |
| **Git** | `worktree`, `merge`, `pr`, `branch`, `cleanup` | No |
| **Documents** | `doc_intel` | No |
| **Incidents** | `incident` | No |
| **Checkpoints** | `checkpoint` | No |
| **Total** | **20 tools** | **7 always loaded** |

A typical orchestration session loads Core (7 tools). A session doing planning adds 3 more. A session doing implementation adds Git (5 more). The maximum is 20 — down from 97.

---

## 6. Invisible Infrastructure

### 6.1 Document Intelligence

Document intelligence continues to parse, index, and classify documents structurally. The difference in 2.0 is that no agent ever interacts with this directly in the normal workflow.

**Where it works:**

- **Inside `next`**: When claiming a task, the system identifies the parent feature's specification, extracts the relevant sections (requirements, acceptance criteria, constraints), and includes them in the response. The agent gets 400 targeted tokens instead of an 8,000-token full spec.

- **Inside `handoff`**: When preparing a sub-agent prompt, the system assembles spec sections, related design decisions, and constraints from the document corpus. At 149 specs, this is the difference between a useful prompt and an impossible manual assembly task.

- **Inside `status`**: When showing feature state, the system includes document gaps (missing spec, missing dev-plan) without the agent calling `doc_gaps` separately.

**At small scale (13 specs):** The extraction may not save much — the agent could read the full spec. But it still works, and the overhead is negligible.

**At large scale (149+ specs):** The extraction becomes essential. The orchestrator cannot read all specs. Document intelligence is what makes `handoff` possible at all.

The `doc_intel` feature group remains available for explicit exploration (research tasks, onboarding), but is not part of the normal workflow.

### 6.2 Knowledge System

Knowledge continues to use confidence scoring (Wilson score), TTL-based expiry, and compaction. The difference in 2.0 is that contribution and retrieval are built into the workflow tools.

**Contribution — inline with `finish`:**

```
finish(task_id: "TASK-...", summary: "...", knowledge: [
  {topic: "billing-api-idempotency", content: "The billing API requires idempotency keys on all POST requests", scope: "backend"}
])
```

This is the natural moment for knowledge contribution. The agent has just finished the work and knows what it learned. No separate tool call needed.

**Retrieval — automatic in `next` and `handoff`:**

When claiming a task or preparing a handoff, the system queries the knowledge store for entries relevant to the task's domain (matched by feature tags, scope, and topic similarity). Relevant entries are included in the response alongside spec sections.

At small scale, there may be few entries and the value is modest. At large scale, the accumulated knowledge from hundreds of completed tasks becomes a significant accelerator — each new task starts with the learnings of every previous task in its domain.

**Direct management — via the `knowledge` feature group:**

For explicit operations (confirming entries, resolving conflicts, pruning), the `knowledge` tool in the Knowledge feature group is available. But most agents will never need it.

---

## 7. Feature Groups and Configuration

Feature groups are configured in `.kbz/config.yaml`:

```
mcp:
  groups:
    core: true          # always enabled, cannot be disabled
    planning: false     # enabled when decomposing features
    knowledge: false    # enabled when managing knowledge directly
    git: false          # enabled when working with branches and PRs
    documents: false    # enabled when exploring documents directly
    incidents: false    # enabled when managing incidents
    checkpoints: false  # enabled when human checkpoints are needed
```

The orchestrator (or human) enables groups as needed. The system only registers tools from enabled groups with the MCP client. Disabled groups consume zero context window space.

A convenience shorthand:

```
mcp:
  preset: orchestration   # enables: core + planning + git
```

Presets:
- `minimal` — core only (7 tools)
- `orchestration` — core + planning + git (15 tools)
- `full` — all groups (20 tools)

---

## 8. What Doesn't Change

These elements of 1.0 are validated by usage and carry forward unchanged:

- **YAML on disk in Git** — Agents can always fall back to direct file operations. Nothing is locked behind the API. Everything is version-controlled.
- **Lifecycle state machines** — The status transitions are correct. They just happen inside workflow tools rather than being exposed as separate operations.
- **Document-driven stage gates** — Design → spec → dev-plan → tasks, with human approval at each gate. The document workflow is the human interface and it works.
- **The `work_queue` concept** — Now embedded in `next`, but the logic (automatic promotion, priority sorting, dependency checking) carries forward.
- **Knowledge confidence scoring** — Wilson score with use_count/miss_count. Self-tuning. Sound design.
- **Worktree isolation** — One worktree per feature/bug, isolated branches. Seamless in 1.0.
- **The internal service and storage layers** — Entity services, storage, validation, cache. These are well-tested and correct.
- **The CLI** — `kbz` as a human terminal interface. The CLI can adopt the same resource-oriented patterns.

---

## 9. Scale Considerations

The design must work at two scales:

### Small Project (10–20 specs)

- Orchestrator can hold the whole project in context
- Document intelligence extraction is nice-to-have, not essential
- Knowledge base is thin — few entries to serve
- Feature groups: `minimal` or `orchestration` preset suffices
- `handoff` may assemble context that the orchestrator already knows — still useful for sub-agents that don't

### Large Project (100–500+ specs)

- Orchestrator cannot read all specs — `handoff` and `next` become essential
- Document intelligence extraction is critical — without it, agents either burn context reading full docs or work with incomplete information
- Knowledge base is deep — accumulated learnings from hundreds of tasks significantly accelerate new work
- Search false positives make grep unreliable — structural classification (this section is a *requirement*, that section is *rationale*) becomes the difference between finding what you need and drowning in results
- Cross-feature dependencies are numerous — must be modelled in entities, not just prose
- Feature groups: `orchestration` or `full` preset
- Batch operations go from convenient to necessary

The design achieves scale-readiness through invisible infrastructure: the same tools work at both scales, but the systems behind them (document intelligence, knowledge retrieval) do proportionally more work as the project grows.

---

## 10. What This Enables

### 10.1 The Orchestration Loop

With 2.0, a typical orchestration session looks like:

```
status(plan_id)               → understand the situation (one call)
next(plan_id)                 → claim the top task, get full context (one call)
handoff(task_id)              → get a sub-agent prompt (one call)
spawn_agent(message=prompt)   → delegate to worker (not a Kanbanzai call)
finish(task_id, summary, knowledge)  → record completion (one call)
```

Five calls for a complete claim-delegate-complete cycle. In 1.0, the same workflow required 8–15 calls (work_queue + dispatch_task + context_assemble + get_entity + doc_record_get_content + spawn_agent + complete_task + knowledge_contribute + update_status).

### 10.2 Batch Orchestration

For wave-style work (claiming and completing multiple tasks):

```
status(plan_id)                           → situation picture
next(plan_id)                             → ready queue
entity(action: update, type: task, entities: [{id, status: "active"}, ...])  → claim batch
[spawn workers]
finish([{task_id, summary, knowledge}, ...])  → complete batch
```

### 10.3 The Worker Experience

A worker agent spawned via `handoff` receives:

```
## Task: Implement payment retry logic

### Spec (from work/spec/billing-spec.md §4.2)
[extracted requirements and acceptance criteria]

### Known Constraints (from knowledge base)
- The billing API requires idempotency keys on all POST requests
- Retry delays follow exponential backoff with jitter (see billing-retry-pattern.go)

### Files to Modify
- internal/billing/retry.go (create)
- internal/billing/retry_test.go (create)
- internal/billing/client.go (modify — add retry wrapper)

### Commit Format
feat(TASK-...): implement payment retry logic
```

The worker does its job using standard tools (read_file, edit_file, terminal) and never calls a Kanbanzai tool. Everything it needed arrived in the prompt.

---

## 11. The Viewer

The human interface to Kanbanzai is documents and a viewer. The viewer is a separate design concern but should be informed by the same resource-oriented model.

MCP Resources (application-controlled, read-only) are well-suited for a viewer:

- `kanbanzai://plans` → browsable plan list
- `kanbanzai://plan/{id}` → plan dashboard (same data as `status(plan_id)`)
- `kanbanzai://feature/{id}` → feature detail
- `kanbanzai://docs/{type}` → document list by type

Resources are presented by the host application (Cursor, Claude Desktop, VS Code) through their own UI — sidebars, pickers, panels. This gives the human a browsable project view without consuming agent tool slots.

The viewer design is deferred to a separate document.

---

## 12. Open Questions

These questions should be resolved during specification:

1. **Cross-feature task dependencies.** Should tasks be able to declare dependencies on features (not just other tasks)? The feedback requests it. The implementation complexity needs assessment.

2. **Document type taxonomy.** Should we add `documentation` (or `guide`) to the valid document types? The current taxonomy lacks a type for user-facing documentation.

3. **Elicitation for batch approvals.** The MCP protocol includes Elicitation — servers can request structured input from users. This could enable batch approval flows ("approve these 5 specs?") without separate tool calls. Client support needs investigation.

4. **Sampling for intelligence.** The MCP protocol includes Sampling — servers can request LLM completions through the client. This could power document classification without requiring the agent to orchestrate it. Worth exploring but adds complexity.

5. **Remote MCP server.** Notion moved from a local to a remote MCP server. Should Kanbanzai support Streamable HTTP transport for team use? Not required for 2.0 but worth designing for.

6. **How far to preserve backward compatibility.** Should 2.0 support a `legacy` feature group with the 1.0 tool names for migration? Or is a clean break acceptable?

---

## 13. Summary

Kanbanzai 2.0 is not a new system. It's the same system — same workflow model, same lifecycle enforcement, same YAML-on-disk, same document-driven stage gates — with a fundamentally different interface.

The 1.0 interface asked: "What operations exist on what entity types?" and built 97 tools.

The 2.0 interface asks: "What does an orchestrator need to do during a work session?" and builds 7 core tools (20 maximum with all feature groups enabled).

The intelligence and knowledge systems that went unused in 1.0 become invisible infrastructure that powers `next`, `handoff`, and `status` from the inside. At small scale, they're transparent. At large scale, they're essential.

The measure of success is simple: agents should prefer Kanbanzai tools over grep and direct YAML editing, because the tools are genuinely faster and more useful. If an orchestrator reaches for `grep "status:" .kbz/state/tasks/*.yaml` instead of `status(plan_id)`, the design has failed.