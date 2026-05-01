# Design: Composite Tools for Workflow Chaining

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-01                     |
| Status | approved |
| Author | sambeau                        |

## Overview

This design proposes **composite MCP tool actions** — new actions on existing
Kanbanzai tools that collapse multi-step workflow sequences into single
server-side operations. These composites are deterministic, synchronous, and
stateless: they orchestrate existing service logic without introducing AI
calls, background execution, or new state machines.

The design covers five composite actions: `doc(action: "publish")` for
document lifecycle completion, `entity(action: "bootstrap")` for feature
lifecycle setup, `entity(action: "close-out")` for feature completion
cascading, `develop(action: "dispatch")` for orchestrator dispatch cycles,
and `batch(action: "snapshot")` for prescriptive status rollups.

## Goals and Non-Goals

### Goals

- Reduce the number of sequential MCP tool calls needed for common workflow
  operations by collapsing multi-step sequences into single calls.
- Replace prose gate-failure recovery instructions with structured,
  machine-readable next-action objects.
- Reuse existing service, gate, and lifecycle logic without modification —
  composites are wrappers, not new abstractions.
- Preserve human gates as hard stops that no composite can auto-pass.

### Non-Goals

- AI-driven orchestration inside the MCP server. No LLM calls.
- Background workflow execution or event-driven automation.
- Replacement of individual tool calls — composites are additive, not
  substitutive.
- Cross-batch or cross-project workflow coordination.

## Problem and Motivation

### Context

Kanbanzai's MCP server exposes individual, atomic tools — each does one thing:
register a document, transition an entity, check a gate, generate a handoff
prompt. This is correct architectural hygiene: each tool is testable,
composable, and independently verifiable.

But the chat agent that calls these tools must chain them into coherent
multi-step workflows. The system already knows the right sequence — it's
encoded in the stage bindings, the gate prerequisites, the forward path in
`advance.go`, and the recovery steps in `gate_errors.go`. Yet that knowledge
is only delivered as instructions the agent must read, remember, and execute
manually.

### The friction

Three concrete patterns create measurable reliability problems:

1. **Document lifecycle chain.** Registering a document for a gated type
   (design, spec, dev-plan) requires the agent to: register → receive
   classification nudge → call `doc_intel(action: "guide")` → read outline →
   call `doc_intel(action: "classify")` with per-section classifications →
   call `doc(action: "approve")`. If the agent forgets the classification
   step, approval fails with a concept-tagging gate error. The agent must
   then retrace its steps, consuming context budget and session attention.

2. **Feature bootstrap chain.** Moving a feature from `proposed` to
   `developing` requires 6-8 sequential calls across `doc`, `entity`, and
   `decompose` tools, each gated by document prerequisites. The agent must
   interpret gate failure messages, identify which document is missing, and
   take the correct recovery action. The system already knows which
   documents are missing — it computed that to produce the gate failure —
   but it delivers the answer as prose the agent must parse.

3. **Orchestrator dispatch loop.** Each dispatch cycle requires `next()`
   (inspect ready frontier) → `conflict()` (check parallelism safety) → N
   × `handoff()` (generate per-task prompts) → N × `spawn_agent()`
   (dispatch). The orchestrator skill documents this loop but the agent
   must execute it manually, tracking which tasks were dispatched and which
   remain blocked.

### What happens if nothing changes

The system works. Agents complete workflows. But:

- Context budget is wasted on repeated tool-call sequences that could be
  collapsed into single server-side operations.
- Gate failure recovery depends on the agent correctly parsing error prose
  and executing recovery steps — a brittle dependency on agent attention.
- Orchestrator sessions spend 30-40% of their tool calls on mechanical
  dispatch plumbing rather than substantive work.
- New agents (or agents in fresh sessions) must re-derive the correct
  sequence from skill instructions each time.

### Prior art within the system

The system already has two forms of multi-step composition:

- **`entity(action: "transition", advance: true)`** — walks a feature
  through multiple lifecycle states, checking gates at each step, stopping
  at human gates or checkpoints. This is the closest existing precedent.
- **`MaybeAutoAdvanceFeature`** / **`MaybeAutoAdvancePlan`** — automatic
  cascading transitions when all child entities reach terminal state.

These demonstrate that the server can and should orchestrate multi-step
sequences. This design extends that principle to the chat agent's tool
surface.

## Design

### Core concept: composite actions on existing tools

New workflow-chaining behaviour is added as new **actions** on existing
consolidated tools, consistent with the architecture established in
Kanbanzai 2.0 (Track I — consolidated `doc`, `entity`, `checkpoint` tools).

No new top-level tools are created for document and entity operations. One
new top-level tool (`develop`) is introduced for development dispatch
because no existing tool owns that concept. Each composite action is an
additional entry in the `DispatchAction` map of its parent tool, reusing the
same `ActionHandler` pattern, side-effect reporting, and auto-commit
behaviour.

### Composite Action Catalog

#### `doc(action: "publish")` — register + classify + approve

**Input:**
- `path`, `type`, `title`, `owner` — same as `doc(action: "register")`
- `classifications` — JSON array of classification objects (same schema as
  `doc_intel(action: "classify")`)
- `model_name`, `model_version` — classification metadata

**Internal sequence:**

1. Call the existing `docRegisterOne` path — registers the document, returns
   `document_id` and `content_hash`.
2. If `classifications` are provided with at least one populated
   `concepts_intro`:
   a. Call `doc_intel` classification logic internally with the provided
      classifications, `content_hash`, `model_name`, and `model_version`.
   b. Call `docApproveOne` internally.
   c. Report any entity lifecycle cascade from the approval as a side
      effect.
   d. Return: `{document, status: "approved", classified: true}`.
3. If `classifications` are missing or have no `concepts_intro`:
   a. Return the existing `classification_nudge` (same as `register` today)
      plus: `{status: "draft", next_action: "classify_then_approve"}`.
   b. The agent classifies, then calls `doc(action: "approve")` as before.

**Gate behaviour:** The concept-tagging gate is evaluated by `docApproveOne`
exactly as it is today. The composite does not bypass it — it just accepts
the classifications inline rather than requiring a separate tool call.

**Error handling:** If registration succeeds but classification or approval
fails, the document remains in `draft` status. The response includes:
- `document` — the registered document record
- `registration` — `"ok"`
- `approval` — `"failed: <reason>"`

#### `feature(action: "bootstrap")` — lifecycle setup through developing

This action operates on the `entity` tool (under `entity(action:
"bootstrap", ...)`).

**Input:**
- `feature_id` — the feature to advance (must be in `proposed`)
- `target` — optional target status (default: `developing`)

**Internal sequence:**

1. Load feature, validate it's on the forward path.
2. Walk the forward path from current status toward target (reuses
   `AdvanceFeatureStatus` internally).
3. At each stage where a gate fails:
   a. Identify the specific missing prerequisite (document type, task
      count, doc approval status).
   b. Return: `{stopped_at: <stage>, reason: <specific missing item>,
      next_action: <exact tool call to resolve>, ...}`.
4. At each stage with `human_gate: true`:
   a. Stop and return: `{stopped_at: <stage>, reason: "human_gate",
      message: "This stage requires human approval before proceeding."}`.
5. If the target is reached:
   a. Return: `{status: <target>, advanced_through: [...], ...}`.

**Next-action precision:** The `next_action` field is not prose — it's a
structured object:
```json
{
  "tool": "doc",
  "action": "approve",
  "params": {"id": "DOC-01KQJ..."},
  "description": "Approve the design document for this feature"
}
```

This eliminates the "parse gate failure prose" problem. The chat agent can
execute the next action without interpreting the error.

**Key difference from `advance`:** `entity(action: "transition", advance:
true)` today stops at gate failures with a prose `stopped_reason`. The
`bootstrap` action enhances this with structured next-action instructions.
The underlying `AdvanceFeatureStatus` function is reused without
modification — only the response format is enriched with prescriptive
follow-up actions derived from gate evaluation results.

#### `entity(action: "close-out")` — reviewing → done with cascade

**Input:**
- `feature_id` — the feature to close out (must be in `reviewing`)

**Internal sequence:**

1. Verify all tasks are terminal.
2. Check for approved review report document.
3. If report missing → return structured next-action.
4. Advance feature to `done`.
5. Check parent batch — if all features are terminal, auto-advance batch
   toward `done`.
6. Return summary of all cascading transitions.

**Cascade reporting:** Uses the existing `SideEffectStatusTransition` and
`SideEffectPlanAutoAdvanced` side-effect types. The response enumerates
every entity affected by the close-out, not just the requested feature.

#### `develop(action: "dispatch")` — one-cycle orchestrator dispatch

This action introduces a new top-level `develop` tool.

**Input:**
- `feature_id` — the feature to dispatch for
- `role` — role profile ID for context assembly (same as `handoff`)
- `instructions` — optional additional orchestrator instructions

**Internal sequence:**

1. Verify feature is in `developing`.
2. List all tasks, identify ready frontier (status = `ready`, all
   `depends_on` satisfied).
3. Run conflict analysis on ready tasks via the existing `conflict` tool
   logic.
4. For each conflict-safe task in the ready frontier:
   a. Transition `ready` → `active`.
   b. Generate handoff prompt via the existing `handoff` pipeline.
5. Return:
   - `dispatched` — array of `{task_id, handoff_prompt, conflict_domain}`
     for each dispatched task
   - `conflicting` — array of ready-but-conflicting tasks with explanation
   - `blocked` — array of non-ready tasks with what's blocking them
   - `empty_queue` — boolean, true if nothing was ready

**Safety:** Conflict analysis is applied before dispatch to prevent two
tasks that modify the same files from running in parallel. The calling agent
must still call `spawn_agent` for each dispatched task — the `dispatch`
action generates prompts, transitions state, and reports conflicts; it does
not spawn sub-agents itself.

#### `batch(action: "snapshot")` — prescriptive status rollup

**Input:**
- `batch_id` — the batch to snapshot

**Internal sequence:**

1. List all features in the batch with their current lifecycle stage.
2. For each feature not in a terminal state, evaluate what's blocking the
   next transition (document missing, tasks incomplete, human gate, review
   needed).
3. Sort features by dependency order (features that unblock others first).
4. Return structured next-action instructions for every blocked feature.

**Output format:**
```json
{
  "batch": {"id": "B43-composite-tools", "status": "designing"},
  "features": [
    {
      "feature_id": "FEAT-01KQJ...",
      "status": "designing",
      "blocked": true,
      "blocking_gate": "design→specifying",
      "missing": {"type": "document", "document_type": "design", "status_needed": "approved"},
      "next_action": {"tool": "doc", "action": "approve", "params": {"id": "DOC-01KQJ..."}}
    }
  ],
  "summary": "1 of 3 features blocked. 2 features ready to advance."
}
```

### Architectural principles

These design decisions are constrained by — and consistent with — the design
principles established in the Skills System Redesign (DP-1 through DP-11):

**DP-9 alignment (constraint levels).** Composite tools operate at the low
freedom / hard gate enforcement tier. They are deterministic — no AI
judgment is exercised inside the tool. The tool sequences are exact
reproductions of what the chat agent would do manually, collapsed into
single server-side operations. This matches DP-9: "Low freedom: exact tool
call sequences, deterministic scripts. One safe path — take it."

**No new state machines.** Composite actions reuse the existing lifecycle
state machines, gate evaluators, and transition logic. They are convenience
wrappers over existing `service` and `gate` package functions, not new
abstraction layers.

**Side-effect reporting preserved.** All composite actions use the existing
`WithSideEffects` middleware and push the same `SideEffect*` types as
individual tool calls. A `feature(action: "bootstrap")` that advances
through three stages produces the same side effects as three separate
`entity(action: "transition")` calls.

**Atomic steps, non-atomic composite.** Each internal step within a
composite is an independent atomic operation (single state transition,
single document registration). If step 3 of 5 fails, steps 1-2 are already
persisted and valid. The composite reports what succeeded and what failed.
This is identical to the existing `advance` behaviour — partial advancement
is intentional and safe.

**Human gates are hard stops.** No composite action auto-passes a human
gate. When the forward path reaches a stage with `human_gate: true`, the
composite stops and returns instructions for the human to approve before
resuming.

### What composite tools are NOT

- **Not AI-agents inside the server.** No LLM calls. All logic is
  deterministic orchestration of existing service functions.
- **Not workflow automation.** Composites don't run in the background or
  respond to events. They are synchronous request-response MCP tool calls.
- **Not replacements for individual tools.** Every underlying action
  (`entity transition`, `doc register`, `doc approve`) remains directly
  callable. Composites are convenience wrappers, not forced paths.
- **Not magic.** A composite that encounters an unexpected state returns an
  error with enough detail for the agent to fall back to individual tool
  calls.

### Implementation strategy

The design follows the existing `DispatchAction` pattern used by every
consolidated tool:

```
// In doc_tool.go:
"publish": docPublishAction(docSvc, intelligenceSvc),

// In entity_tool.go:
"bootstrap": entityBootstrapAction(entitySvc, docSvc, gateRouter, ...),
"close-out": entityCloseOutAction(entitySvc, docSvc, ...),

// New file: develop_tool.go — new top-level tool
// New logic in batch infrastructure — snapshot action
```

**New files:**
- `internal/mcp/develop_tool.go` — `develop` tool with `dispatch` action

**Modified files:**
- `internal/mcp/doc_tool.go` — add `publish` action handler
- `internal/mcp/entity_tool.go` — add `bootstrap` and `close-out` action
  handlers on the existing `entity` tool
- `internal/mcp/server.go` — register new `develop` tool

**Estimated effort:** ~800-1200 lines of new Go code across the handlers,
plus tests. Each composite action is 100-200 lines, mostly orchestrating
existing service calls.

## Alternatives Considered

### Alternative 1: Do nothing — improve skill instructions instead

Make the existing skill documents more explicit about multi-step sequences
and add "after you call X, you must call Y" nudges to tool responses.

**Trade-offs:**
- *Easier:* No code changes. No new tool surface to maintain.
- *Harder:* The fundamental problem — agent memory/attention — is not
  solved by more instructions. The agent still must read, remember, and
  execute sequences. Better instructions reduce the error rate but don't
  eliminate the failure mode.
- **Rejected because:** The cost of the wrong call sequence is wasted
  context budget and broken workflow state. At Kanbanzai's scale (40+
  batches, hundreds of tasks), the aggregate reliability loss justifies
  server-side composition.

### Alternative 2: Full workflow engine — server-driven orchestration

Build a workflow engine into the MCP server that drives entire batches
autonomously: auto-dispatch tasks, monitor completion, advance features,
merge when done. The chat agent becomes an observer rather than the driver.

**Trade-offs:**
- *Easier:* Maximum reliability — no agent attention dependency for
  workflow mechanics.
- *Harder:* Significant architectural change. Requires background job
  execution, state monitoring, and error recovery logic. Blurs the boundary
  between the MCP server (a tool provider) and an orchestration platform.
  Overlaps with human judgment at review gates.
- **Rejected for now:** This is a valid long-term direction (similar to how
  CI/CD systems evolved from manual command sequences to automated
  pipelines) but exceeds the current scope. Composite tools are a stepping
  stone — they prove the value of server-side composition without requiring
  background execution.

### Alternative 3: Client-side workflow scripts

Instead of server-side composition, provide shell scripts or Claude Code
custom commands that execute the multi-step sequences client-side (one script
calls the MCP server multiple times).

**Trade-offs:**
- *Easier:* No server changes. Scripts are independently versionable.
- *Harder:* Scripts can't access internal service state — they must parse
  MCP responses to decide next steps, reimplementing gate logic client-side.
  Two sources of truth for workflow rules (server gates + client scripts).
  Platform-specific (works in Claude Code, not in Copilot Chat or other MCP
  clients).
- **Rejected because:** The server already knows the correct sequence —
  duplicating that knowledge client-side creates a maintenance burden and
  limits portability across MCP clients.

## Decisions

### Decision 1: New actions on existing tools, not new top-level tools

- **Context:** The system already follows a consolidated-tool pattern where
  `doc`, `entity`, `checkpoint` serve multiple actions via a `DispatchAction`
  map.
- **Rationale:** Adding actions to existing tools preserves the mental model
  for chat agents ("the `doc` tool does document things") and keeps the
  tool list manageable. New top-level tools would increase discovery
  overhead without adding conceptual clarity.
- **Consequences:** The `develop` tool is introduced as a new top-level tool
  because no existing tool owns the "feature implementation dispatch"
  concept. Future development-related actions (e.g., `develop(action:
  "status")`) would also live here. The `batch(action: "snapshot")` action
  is added to the existing batch infrastructure.

### Decision 2: Structured next-action responses, not prose

- **Context:** Today's gate failure responses return prose recovery steps
  that the agent must parse.
- **Rationale:** Structured `next_action` objects eliminate the "parse
  prose" failure mode. The agent can forward them directly to the next tool
  call without interpretation. This is especially valuable for agents in
  high-utilisation sessions where prose comprehension degrades.
- **Consequences:** The structured format must be maintained as new gate
  types are added. This is a minor constraint — the format is simple
  (`{tool, action, params, description}`) and maps directly to MCP tool
  call signatures.

### Decision 3: Composites are synchronous, deterministic, and stateless

- **Context:** An alternative would be stateful workflow sessions where the
  server tracks "where we are" in a multi-step sequence across calls.
- **Rationale:** Stateless composites are simpler to implement, test, and
  reason about. Each call is independent. The agent can interleave composite
  and individual tool calls freely. Stateful workflows would introduce
  session management, timeout handling, and recovery complexity — all for a
  problem (remembering the next step) that structured next-action responses
  already solve.
- **Consequences:** The agent is still responsible for calling the next
  tool in the sequence. The composite improves reliability by making the
  next step explicit and machine-readable, not by taking over the sequence
  entirely. This is the right trade-off for the current maturity level.

### Decision 4: Classification must still be AI-driven; composite only passes it through

- **Context:** The `doc(action: "publish")` composite could theoretically
  call an LLM API to auto-classify documents server-side.
- **Rationale:** Classification requires reading and understanding document
  content — an AI task. Embedding an LLM call in the MCP server introduces
  API key management, cost unpredictability, and model selection complexity.
  The composite instead accepts classifications that the chat agent has
  already produced, collapsing the submission step without taking over the
  thinking step.
- **Consequences:** `doc(action: "publish")` without classifications still
  requires a separate `doc_intel` + `doc approve` sequence. This is the same
  as today, just with clearer next-action guidance. If server-side
  classification is pursued later, it slots into the same `publish` action
  as an optional auto-classify flag.

### Decision 5: `develop(action: "dispatch")` dispatches but does not spawn

- **Context:** The `dispatch` action could call `spawn_agent` directly,
  fully automating the orchestrator's dispatch loop.
- **Rationale:** `spawn_agent` is a chat-client operation, not an MCP server
  operation — it requires access to the client's sub-agent infrastructure.
  The MCP server generates prompts and transitions state; the chat client
  spawns agents. Keeping this boundary clean avoids coupling the server to
  a specific client's agent-spawning mechanism.
- **Consequences:** The orchestrator still calls `spawn_agent` for each
  dispatched task. However, the dispatch action eliminates the separate
  `next` → `conflict` → N × `handoff` sequence, reducing the orchestrator's
  per-cycle tool calls from 3+N to 1+N.

## Dependencies

### Internal dependencies (existing code)

- **`service/advance.go`** — `AdvanceFeatureStatus`, `AdvanceConfig`,
  `featureForwardPath`, `advanceStopStates`. The `bootstrap` and `close-out`
  actions wrap this function. No modifications required.
- **`service/entity_children.go`** — `MaybeAutoAdvanceFeature`,
  `MaybeAutoAdvancePlan`. Used by `close-out` for cascade logic.
- **`gate/` package** — `GateRouter`, `RegistryCache`, document and task
  evaluators. Gate evaluation in composites uses the same router as
  individual transitions.
- **`internal/mcp/sideeffect.go`** — `SideEffect` types and
  `WithSideEffects` middleware. All composites push the same side-effect
  types.
- **`internal/mcp/doc_tool.go`** — `docRegisterOne`, `docApproveOne`.
  `publish` calls these directly.
- **`internal/mcp/handoff_tool.go`** — handoff prompt generation pipeline.
  `dispatch` calls this for each task.

### External dependencies

- **`mcp-go` library** — used for tool registration and handler signatures.
  No version change required.
- **No new dependencies.** All composite logic uses existing `service`,
  `gate`, `binding`, and `validate` packages.

### Upstream design dependencies

- **Skills System Redesign** (`work/design/skills-system-redesign-v2.md`)
  Design Principles DP-9 (constraint levels), DP-5 (composition), and DP-10
  (only add what the model doesn't know) directly constrain this design.
- **Stage bindings** (`.kbz/stage-bindings.yaml`) — composite actions
  respect human gates and document prerequisites defined in the bindings.
  Changes to stage bindings propagate automatically via the `GateRouter`.

### Downstream consumers

- **Chat agents** using the `doc`, `entity`, `develop`, and `batch` tools.
  No migration required — existing individual actions remain available.
- **Orchestrator skills** (`orchestrate-development`, `orchestrate-review`)
  should be updated to prefer `develop(action: "dispatch")` over manual
  dispatch loops.
- **Workflow skills** (`kanbanzai-workflow`, `kanbanzai-documents`) should
  reference composite actions as the recommended path for multi-step
  operations.
