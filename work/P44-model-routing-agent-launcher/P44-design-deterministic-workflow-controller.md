# Design: Deterministic Workflow Controller for Kanbanzai

**Status:** Draft (replaces the original P44 model-routing design and the P44 feature-execution-pipeline design)
**Date:** 2026-05-08
**Owner:** P44-model-routing-agent-launcher
**Supersedes:**
- `P44-design-model-routing-agent-launcher.md` (multi-provider routing, demoted to Phase 3)
- `P44-design-feature-execution-pipeline.md` (incorporated and extended here)

**Companion (still active):**
- `P44-F1-design-prompt-assembly-gate.md` (the assembly gate is one component of this design and ships first)

**Source of authority:** This design implements the recommendations of `P44-research-orchestrator-architecture.md` (2026-05-08).

---

## 1. Overview

This is the canonical P44 design. It replaces the previous, scope-bloated P44 framing (multi-provider model routing as the primary deliverable) with the architecture the research report identifies as load-bearing: a **deterministic workflow controller**, owned by the MCP server, that holds workflow state, assembles prompts, dispatches sub-agents, runs deterministic gates, drives a two-layered verifier, and writes a structured audit log. Multi-provider model routing — the original P44 — is preserved as a Phase 3 optimisation behind a stable substrate.

The design is grounded in the four-plan evidence trail (P50, P55, P56, P57, P58) and the 2024–2026 industry consensus summarised in the research report: code-as-controller, scoped LLM activities, external verifiers, durable state. It explicitly rejects the failure pattern that defines the current chat-based orchestrator: prose advice (skills, role YAML, anti-patterns, DoD checklists) used as a control mechanism for procedural invariants that LLMs cannot reliably enforce.

**Thesis.** Reliable workflow execution requires a deterministic component whose only job is to enforce invariants and that **cannot be talked out of doing it**. Every mechanism that allowed prior failures (manual prompt composition, self-verification, scattered policy, invisible pipeline) is closed by code, not by additional skill text.

---

## 2. Goals and Non-Goals

### 2.1 Goals

1. **Make the dispatch path non-bypassable in tooling, not in policy.** A sub-agent can be created only via a code path that runs the assembly gate, attaches assembled context, and writes an audit record.
2. **Move stage transitions and stage execution into the server.** `developing`, `reviewing`, and `verifying` are driven by stage controllers triggered by transition hooks, not by the orchestrator composing and dispatching prompts.
3. **Two-layered Definition of Done.** Deterministic Go code runs all checks that can be deterministic (git status, ancestry, tests, document records, worktree cleanup). An LLM verifier sub-agent adjudicates only items that genuinely require natural-language judgement, with structured JSON output and schema-enforced rejection.
4. **Continuous evaluation harness.** Golden-task pipeline-assembly outputs are checked into the repo; CI fails on diff. Every controller-driven feature emits a structured audit row.
5. **Centralised policy engine.** A single Go package owns: which roles can use which tools, which transitions require which artefacts, which sub-agent role/skill is dispatched at which gate. No more rules scattered across YAML, markdown, and ad-hoc Go.
6. **Durable, replayable controller state.** Stage controllers run as bounded async workflows with explicit gates, retries, signals, and human-checkpoint primitives. No naive goroutines.
7. **Backwards-compatible migration.** Existing in-flight features finish under the legacy path. New features default to the controller. Per-entity feature flag.
8. **Demote the chat orchestrator to a supervisor.** The orchestrator no longer dispatches, composes, collates, or verifies. It activates features, monitors progress, and handles exceptions.

### 2.2 Non-Goals (this design)

- **Multi-provider model routing.** Deferred to Phase 3 (the original P44 scope). It is an optimisation that depends on the eval harness shipping first; without the harness it silently regresses quality.
- **U-shaped compaction.** The original P44 invested heavily in compaction as a coping strategy for an overloaded orchestrator. The orchestrator's role shrinks dramatically here, so compaction becomes a Phase 3 nice-to-have, not a Phase 1 deliverable.
- **Replacing the 3.0 context-assembly pipeline.** The pipeline (`internal/context/pipeline.go`) is reused unchanged. The new controller calls it; it does not replace it.
- **Replacing the existing entity / lifecycle data model.** Entity records, lifecycle states, and stage bindings remain authoritative.
- **Replacing the existing skills/roles content layer.** Skill markdown and role YAML continue to provide *content* for assembled prompts. They are no longer relied on for *control*.
- **Adopting a third-party durable execution engine (Temporal, Restate) in Phase 1.** A minimal Go durable-execution layer (~500 LOC) is built in-tree. Temporal/Restate is a Phase 3+ option if scale justifies it.

---

## 3. Problem and Motivation

### 3.1 The category error

Prose rules in `orchestrate-development.md`, `roles/*.yaml`, and the DoD checklist are being used as control mechanisms. They are advisory context. The four-plan record (P50→P58) shows every fix that adds *more advice* is bypassed, including the rule that explicitly forbids the bypass.

### 3.2 The decisive defect

The chat orchestrator is simultaneously the dispatcher, the rule-follower, and the reviewer of its own outputs. There is no external controller, no external verifier, and no enforcement boundary. Every other layer (model, prompt, context, tool affordance, workflow state) is a contributing cause; this is the load-bearing one.

### 3.3 What the research report changed

| Aspect | Original P44 | New P44 |
|---|---|---|
| Primary deliverable | Multi-provider model routing | Deterministic controller, dispatch lock-down, deterministic verifier, eval harness, policy engine |
| Pipeline assembly gate | Phase 1 | Phase 1 (unchanged — already aligned) |
| Stage controllers | Phase 1 (sync/async unresolved) | Phase 2, async, on minimal in-tree durable-execution layer |
| `spawn_agent` bypass surface | Open question | **Closed in Phase 1**: removed from the orchestrator's tool list |
| Verifier | LLM sub-agent with structured output | **Two-layered**: deterministic Go checks + LLM judgement; deterministic always runs and can independently fail the gate |
| Eval harness | Mentioned as future risk mitigation | **Phase 1 deliverable**, blocks pipeline changes that diff golden output |
| Policy engine | Implicit, scattered across YAML/md/Go | **Phase 2 Go package** (`internal/policy`) |
| Audit log | "Pipeline debug mode" only | **Phase 2 structured event log** (`internal/audit`) — append-only JSONL with correlation IDs |
| Multi-provider routing | Phase 1 (Anthropic + DeepSeek) | **Phase 3**, gated on eval harness stability and observed workload telemetry |
| U-shaped compaction | Phase 1 deliverable | Phase 3 nice-to-have; orchestrator does little long-horizon work in target architecture |
| Migration strategy | Undefined | **Per-entity feature flag**; legacy path coexists until cohort drains |

---

## 4. Architecture

### 4.1 Target shape

```
                    ┌─────────────────────────────────────┐
                    │      Human / Chat Supervisor        │
                    │  (entity transitions, exceptions,   │
                    │   strategic decisions, checkpoints) │
                    └─────────────────┬───────────────────┘
                                      │  entity(action: transition)
                                      ▼
            ┌─────────────────────────────────────────────────┐
            │            MCP Server — Controller              │
            │                                                 │
            │  Policy Engine (tool perms, gate rules) ◄──┐    │
            │  Workflow State (durable, replayable)      │    │
            │  Stage Controllers (developing, reviewing, │    │
            │   verifying) — bounded async loops         │    │
            │  Prompt Assembly Pipeline (3.0 + gate)     │    │
            │  Audit Log (every dispatch, gate, verdict) │    │
            │  Eval Harness (golden tasks, regressions)  │    │
            └────────┬───────────────────────────┬────────────┘
                     │                           │
                     ▼                           ▼
      ┌──────────────────────┐    ┌─────────────────────────────┐
      │  Provider(s)         │    │  Deterministic Verifier     │
      │  (LLM activities:    │    │  (git, tests, doc records,  │
      │   implementer,       │    │   ancestry, worktree state) │
      │   reviewer panel,    │    │                             │
      │   LLM verifier)      │    │                             │
      └──────────────────────┘    └─────────────────────────────┘
```

### 4.2 Component inventory

The new packages and their responsibilities:

| Package | Responsibility | Phase |
|---|---|---|
| `internal/context` (existing) | 3.0 prompt-assembly pipeline + assembly gate (P44-F1) | 1 |
| `internal/dispatch` (new) | `dispatch_task` MCP tool implementation; the only path to a sub-agent in Phase 1 | 1 |
| `internal/eval` (new) | Golden-task harness; checked-in expected outputs; CI hook | 1 |
| `internal/audit` (new) | Append-only structured event log (JSONL); correlation IDs; query helpers | 2 |
| `internal/controller` (new) | Stage controllers (`developing`, `reviewing`, `verifying`); bounded loops; transition-hook integration | 2 |
| `internal/durable` (new) | Minimal durable-execution layer: persistent state machine, retries, signals, replay; ~500 LOC | 2 |
| `internal/verifier` (new) | Deterministic DoD checks (git, tests, doc records, ancestry, worktree); orchestrates LLM verifier as second layer | 2 |
| `internal/policy` (new) | Single source of truth for: role→tool mappings, transition→artefact requirements, gate→sub-agent dispatch rules | 2 |
| `internal/provider` (new) | `Provider` interface; Anthropic implementation (Phase 1, single provider); routing config (Phase 3) | 1 (single provider) → 3 (multi) |
| Existing entity / lifecycle / stage-bindings | Unchanged in data model; consumed by `internal/policy` and `internal/controller` | – |

### 4.3 Data flow: a feature from `developing` to `done`

```
1. Supervisor:  entity(action: "transition", id: "FEAT-123", status: "developing")
                │
                ▼
2. Server:      transition hook fires
                policy.GateAllows("specifying" → "developing", FEAT-123)? ✓
                audit.Record("transition", FEAT-123, "developing")
                durable.StartWorkflow("developing-controller", FEAT-123)
                │
                ▼
3. Controller (developing):
                For each ready task (parallel where policy allows):
                    pipeline.Assemble(task)           → assembled prompt
                    assembly_gate.Check()             → hard-fail on missing role/skill
                    audit.Record("dispatch", task, role, skill, model, tokens)
                    provider.Dispatch(prompt, tools)  → sub-agent runs, calls finish()
                When all tasks terminal:
                    if any needs-rework AND cycle < 3 → loop
                    else: durable.Signal("transition", "reviewing")
                │
                ▼
4. Controller (reviewing):
                Dispatch parallel reviewers (conformance, quality, security, testing)
                Collate findings via deterministic policy rules
                If blocking findings AND cycle < 3:
                    create rework tasks; signal "developing"
                Else:
                    signal "merging"
                │
                ▼
5. Controller (verifying):
                verifier.RunDeterministic(FEAT-123)
                    git.StatusClean()                 → ✓/✗
                    git.MergedToMain()                → ✓/✗
                    tests.Pass()                      → ✓/✗
                    doc.Records({spec, design, dev-plan}) → ✓/✗
                    worktree.Removed()                → ✓/✗
                If any deterministic fail → audit + checkpoint to supervisor
                Else: dispatch LLM verifier (clean context, structured JSON output)
                    Schema validation → reject malformed
                    All-pass → audit.Record("verified", FEAT-123); transition "done"
                    Any fail → audit + checkpoint
```

**Key property:** the supervisor calls `entity(action: transition, status: "developing")` once. Everything else happens server-side. The supervisor's window never contains assembled prompts, sub-agent outputs, review findings, or DoD intermediate state.

---

## 5. Component Designs

### 5.1 Dispatch lock-down (Phase 1)

**Problem.** `spawn_agent` accepts arbitrary text. As long as it does, every "rule" telling the orchestrator not to compose prompts manually is advisory. The four-plan record proves this empirically.

**Solution.**

1. **Remove `spawn_agent` from the orchestrator's tool list.** Tool affordance is the lever; prose rules are not. Anthropic's published guidance is explicit: *"If you don't want the model to do X, don't give it the tool for X."*
2. Introduce **`dispatch_task`** as the only Phase 1 path to a sub-agent. Signature:

   ```
   dispatch_task(task_id: string, category?: string, role_override?: string) → DispatchResult
   ```

   Internally:
   - Loads the task and its parent feature.
   - Runs the 3.0 pipeline → assembled prompt.
   - Runs the assembly gate (P44-F1): hard-fail on missing role/skill, warn on missing tool hints / low token budget.
   - Resolves the model via the provider (Phase 1: single provider; Phase 3: routing).
   - Calls `provider.Dispatch(prompt, tools)`.
   - Writes an audit row.
   - Returns a structured result (not a raw transcript).

3. **`spawn_agent` is not removed from the codebase** — it remains available to the server for internal use (controllers call it under the hood, or use a sibling internal API). It is removed only from the *orchestrator's exposed tool list*.

4. **Bypass-attempt test suite.** A test asserts that the orchestrator role's allowed tool set excludes `spawn_agent`, and that any dispatch attempted through internal channels passes through `dispatch_task` semantics (assembly gate, audit row).

5. **Migration courtesy.** During Phase 1, `next(id)` continues to return a `handoff_prompt` field (P44-F1 Decision 4) so the supervisor can inspect what the sub-agent will see. This is a *temptation surface* by design: it should be removed in Phase 2 once controllers own dispatch, because the supervisor no longer needs it.

### 5.2 Stage controllers (Phase 2)

Each lifecycle stage with sub-agent work gets a stage controller — a function invoked by the transition hook that manages the stage end-to-end.

#### 5.2.1 Controller surface

```go
// internal/controller/controller.go
type StageController interface {
    // Run executes the stage workflow for the given entity.
    // Returns the next stage status or an error.
    // Long-running and idempotent; safe to replay from any saved checkpoint.
    Run(ctx context.Context, entityID string) (nextStatus string, err error)
}

type Hook struct {
    developing StageController
    reviewing  StageController
    verifying  StageController
    durable    durable.Runtime
    audit      audit.Sink
    policy     policy.Engine
}

func (h *Hook) AfterTransition(from, to string, e model.Entity) error {
    h.audit.Record(audit.Event{Type: "transition", Entity: e.ID, From: from, To: to})
    if !h.policy.AllowsTransition(from, to, e) { /* error */ }
    if c := h.controllerFor(to); c != nil {
        // Start as durable workflow; do not block the transition response.
        return h.durable.Start(ctx, h.workflowID(e, to), c.Run, e.ID)
    }
    return nil
}
```

#### 5.2.2 Developing controller

Identical loop to the previous design (`P44-design-feature-execution-pipeline.md` §Developing stage controller), with two changes:

- Dispatch goes through `dispatch_task` semantics — assembly gate, audit row.
- The 3-cycle cap is enforced by the controller (not the orchestrator), and exhaustion emits a `checkpoint` event consumed by the supervisor.

#### 5.2.3 Reviewing controller

Identical to the previous design with the same two changes. Findings are collated via deterministic policy rules in `internal/policy` (no LLM involved in classifying blocking vs non-blocking when the rubric is structural).

#### 5.2.4 Verifying controller

**Significantly changed** from the previous design — see §5.4.

#### 5.2.5 Sync vs async (resolved)

**Decision: async, on a minimal in-tree durable-execution layer.** Rationale:

- Synchronous controllers stall transitions for minutes-to-hours. Unacceptable.
- Naive goroutines lose state on crash, lose progress on restart, cannot be inspected, cannot be retried, and have no human-signal primitive.
- Temporal/Restate are over-engineering for a single-developer system.
- A ~500 LOC Go package (`internal/durable`) provides the minimum viable substrate: persistent state per workflow, retries with exponential backoff, signals (e.g. "supervisor approves continuation"), replay from saved checkpoint, and a debug query (`durable.Inspect(workflowID)`).

This is the kind of decision the research report explicitly flags as load-bearing (§5.3 item 4). It is now made.

### 5.3 Two-layered verifier (Phase 2)

The single most-cited correctness improvement in the research literature (Reflexion, OpenAI o-series, DeepMind, Anthropic computer-use). The previous design had only an LLM verifier; this design adds a deterministic layer that runs first and can independently fail the gate.

#### 5.3.1 Deterministic layer

`internal/verifier/deterministic.go`:

```go
type DeterministicCheck struct {
    Name     string                                   // e.g. "git_status_clean"
    Run      func(ctx context.Context, e Entity) Result
    Required bool                                     // hard-fail vs advisory
}

type Result struct {
    Pass     bool
    Evidence string   // command output or path inspected
    Error    error
}
```

**Required Phase 2 deterministic checks (initial set):**

| Check | Implementation | Required? |
|---|---|---|
| `git_status_clean` | `git status --porcelain` empty in worktree | Yes |
| `branch_merged_to_main` | `git merge-base --is-ancestor <branch> main` | Yes |
| `tests_pass` | `go test ./...` exit 0 (or per-language equivalent) | Yes |
| `doc_records_present` | `doc.list(owner: featureID)` includes `specification`, `design`, `dev-plan` (per stage binding) | Yes |
| `worktree_removed` | Worktree status = `merged` or `removed`, directory absent | Yes |
| `all_tasks_terminal` | All child tasks in `done` / `not-planned` / `duplicate` | Yes |
| `review_reports_present` | At least one review report doc per required reviewer dimension | Yes |
| `incident_links_resolved` | If `entity.affected_features` is non-empty, all linked incidents in `resolved`/`closed` | No (advisory) |

The deterministic layer always runs first. Any required-fail short-circuits the gate, writes an audit row, and emits a checkpoint event.

#### 5.3.2 LLM verifier sub-agent (second layer)

Runs only after deterministic checks pass. Inputs:

- Entity ID and feature summary.
- Approved spec, design, and dev-plan content (read by the verifier package, passed in the prompt — verifier sub-agent has **no document tools**).
- The verification rubric items that genuinely require natural-language judgement (e.g. "does the implementation match the spec's intent?", "are review findings adequately addressed?").

Contract:

- **Clean context.** The verifier sub-agent has no prior session context. It is dispatched fresh per feature.
- **Structured JSON output.** Schema:
  ```json
  {
    "schema_version": "1",
    "feature_id": "FEAT-123",
    "items": [
      {"item": "spec_intent_match", "pass": true|false, "evidence": "..."},
      ...
    ],
    "overall": "pass"|"fail",
    "notes": "..."
  }
  ```
- **Schema validation.** If output fails to parse or fails the schema, the verdict is rejected and the gate emits a checkpoint. The LLM is never given a second chance to "correct" malformed output in the same dispatch.
- **No tool list.** The LLM verifier cannot read files, cannot dispatch sub-agents, cannot call `entity`. Everything it needs is in the prompt. This bounds attack surface and reproducibility scope.

Belt-and-braces principle: **the deterministic layer always runs and can independently fail the gate; the LLM verifier is additional, never substitutive.**

### 5.4 Policy engine (Phase 2)

`internal/policy` consolidates rules currently scattered across `stage-bindings.yaml`, `roles/*.yaml`, `skills/*/SKILL.md`, and ad-hoc Go in `entityTransitionAction`.

#### 5.4.1 Policy domains

| Domain | Question it answers | Source today |
|---|---|---|
| **Role-tool permissions** | "Can role X call tool Y?" | Implicit in MCP server tool registration; not enforced per-role |
| **Transition prerequisites** | "What artefacts must exist for `specifying → dev-planning`?" | Stage bindings; partial enforcement in `CheckFeatureTransitionGate` |
| **Stage-controller dispatch** | "Which sub-agent role/skill is dispatched at the `developing` gate?" | Stage bindings (read by controller) |
| **Verifier required checks** | "Which deterministic checks are required for `verifying`?" | New |
| **Bypass policy** | "Can a transition be overridden? By whom? With what justification?" | `entity(override: true, override_reason: ...)` — currently honoured but not policy-checked |

#### 5.4.2 API sketch

```go
type Engine interface {
    AllowsTransition(from, to string, e model.Entity) Decision
    AllowedTools(role string) []string
    GateRequirements(stage string) GateSpec
    DispatchSpec(stage string) DispatchSpec  // role, skill, model category, parallelism
}

type Decision struct {
    Allowed   bool
    Reason    string
    Override  OverridePolicy   // who can override, what justification is required
}
```

#### 5.4.3 Implementation strategy

- **Phase 2.0:** Build the engine as a thin facade over today's scattered sources. No behaviour change; test that the engine returns the same decisions current code makes.
- **Phase 2.1:** Migrate transition-gate checks to read from the engine.
- **Phase 2.2:** Migrate controller dispatch decisions to read from the engine.
- **Phase 2.3:** Add per-role tool-permission enforcement — the MCP server consults `policy.AllowedTools(role)` before exposing each tool to a role-attributed session.

This avoids a big-bang policy migration.

### 5.5 Structured audit log (Phase 2)

`internal/audit` is an append-only JSON-lines event log persisted under `.kbz/audit/`. Every consequential action emits a row. Schema:

```json
{
  "ts": "2026-05-08T12:34:56Z",
  "correlation_id": "FEAT-123:cycle-1:task-7",
  "event": "dispatch" | "gate_eval" | "verifier_verdict" | "transition" | "provider_call" | "checkpoint" | "override",
  "entity_id": "FEAT-123",
  "actor": "controller:developing" | "supervisor:claude" | "verifier:deterministic" | "verifier:llm",
  "role": "implementer-go",
  "skill": "implement-task",
  "model": "claude-sonnet-4-20250514",
  "tokens_in": 8421,
  "tokens_out": 1203,
  "tool_calls": 14,
  "outcome": "ok" | "fail" | "rework_required" | "checkpoint",
  "details": { ... }
}
```

**Properties:**

- **Append-only.** No mutation. No deletion. Compaction is out-of-scope for Phase 2.
- **Correlation IDs** thread a row through the controller workflow.
- **Queryable.** A small CLI (`kbz audit query`) supports filtering by entity, event type, time range.
- **CI consumable.** The eval harness (§5.6) reads audit rows during integration tests to assert that controllers performed the expected actions in the expected order.

### 5.6 Evaluation harness (Phase 1)

The harness is the only mechanism that prevents the pipeline from becoming silently degraded. It ships in Phase 1, before the controller migration, so that controller changes can be evaluated against a stable baseline.

#### 5.6.1 Golden-task harness

- A directory `internal/eval/golden/` holds 10 curated tasks (mix of feature, bug_fix, retro_fix) with checked-in input fixtures (entity records, parent feature, stage binding) and expected pipeline-assembly outputs.
- A test (`go test ./internal/eval/...`) runs the 3.0 pipeline + assembly gate on each fixture and diffs the output against the checked-in expected file.
- The CI runs this test on every commit that touches `internal/context/`, `internal/dispatch/`, `.kbz/skills/`, `.kbz/roles/`, or `.kbz/stage-bindings.yaml`.
- Diffs fail CI. Updating an expected file requires a deliberate `go test -update-golden` invocation and a reviewer-visible commit.

#### 5.6.2 End-to-end controller harness (Phase 2)

- A test runs a fake feature through `developing → reviewing → verifying → done` with a stub `Provider` that returns canned completions.
- Assertions cover: every gate is evaluated, every artefact is created, every audit row is written, and the bypass-attempt suite (§5.6.3) passes.

#### 5.6.3 Bypass-attempt suite (Phase 1)

- A test that asserts the orchestrator role's allowed tool set excludes `spawn_agent`.
- A test that any internal dispatch path goes through assembly-gate semantics (mocked pipeline + audit hook).
- A test that the LLM verifier's allowed tool set is empty.

#### 5.6.4 Drift detection (Phase 2+)

- A weekly job samples 5 closed features per week and asserts DoD compliance against the audit log: deterministic checks recorded as passing, verifier verdict recorded, no bypass without override+reason.
- Drift findings produce a report; over a threshold, an alarm.

### 5.7 Provider abstraction (Phase 1, single provider)

```go
// internal/provider/provider.go
type Provider interface {
    Dispatch(ctx context.Context, in DispatchInput) (DispatchResult, error)
}

type DispatchInput struct {
    Category string   // "implement" | "review" | "verify" | "deep-reasoning" | "quick"
    Prompt   string
    Tools    []Tool
}

type DispatchResult struct {
    Completion string
    TokensIn   int
    TokensOut  int
    ToolCalls  int
    Model      string  // pinned model version actually used
}
```

**Phase 1:** single provider, single model per category, hardcoded mapping in `internal/provider/anthropic.go`. No routing logic. The interface is stable enough that Phase 3 swaps in a routing implementation without touching callers.

**Phase 3:** RouteLLM/HybridLLM-style routing classifier; provider fallback chains; per-category model overrides via config; provider-side prompt caching where applicable. **Gated on:** stable controller, stable verifier, stable eval harness, ≥30 days of audit-log data to anchor routing decisions in real workload distribution.

### 5.8 Orchestrator skill retirement (Phase 2)

In the controller architecture the orchestrator no longer dispatches, no longer composes prompts, no longer collates review findings, no longer runs verification. `orchestrate-development.md` shrinks to:

- **Activate.** Transition the feature into `developing`.
- **Monitor.** Call `status(id)` to see progress.
- **Decide on exceptions.** Respond to checkpoint events emitted by controllers (cycle cap exceeded, verifier failure, deterministic-check failure, ambiguous review findings).

~80% of the current text becomes irrelevant. The retirement is part of Phase 2 because it depends on the controllers actually shipping; until then, the skill remains the source of truth for what humans-in-the-loop should do.

---

## 6. Decisions

### Decision 1: `dispatch_task` is the only Phase 1 path to a sub-agent; `spawn_agent` is removed from the orchestrator's tool list

The single highest-leverage change in the design. Tool affordance enforces what prose cannot. (Research report §8.1, §10 final-answer item 1.)

### Decision 2: Stage controllers run as bounded async workflows on a minimal in-tree durable-execution layer

Sync stalls transitions; naive goroutines lose state on crash. The in-tree layer (~500 LOC) gives persistent state, retries, signals, and replay without an operational dependency on Temporal/Restate. (Research report §5.3 item 4, §8.2.)

### Decision 3: The Definition of Done is two-layered — deterministic Go checks first, LLM verifier second

The deterministic layer always runs and can independently fail the gate. The LLM verifier is additional, never substitutive. Output is structured JSON validated against a schema; malformed output is rejected. (Research report §3.4, §8.2.)

### Decision 4: The evaluation harness ships in Phase 1, not Phase 2

The pipeline becomes invisible the moment dispatch moves server-side. The harness is the only mechanism that makes regressions visible. Hours of work; non-negotiable. (Research report §5.3 item 2, §8.1.)

### Decision 5: Policy is consolidated into `internal/policy`

A single Go package owns role-tool permissions, transition prerequisites, controller dispatch rules, verifier requirements, and override policy. Phase 2 builds it as a facade over existing scattered sources, then migrates callers domain by domain. (Research report §5.3 item 6.)

### Decision 6: The structured audit log (`internal/audit`) is the system of record for every consequential action

Append-only JSONL with correlation IDs. Consumed by the eval harness, the drift-detection job, and humans debugging incidents. (Research report §5.3 item 7, §8.2.)

### Decision 7: Multi-provider routing is deferred to Phase 3

Routing without an eval harness silently regresses quality. Routing without ≥30 days of audit data is designed against an imagined workload. The original P44 routing design is preserved as a Phase 3 feasibility document and informs the `Provider` interface, but no routing logic ships in Phases 1 or 2. (Research report §3.6, §8.3.)

### Decision 8: Migration is per-entity feature-flagged

`entity.controller_managed: true|false`. New features default to true. In-flight features default to false until they close. The default flips after 20+ controller-driven features close cleanly. The legacy path remains until the cohort drains. (Research report §5.3 item 5, §9 risk row "Migration strands in-flight work".)

### Decision 9: The orchestrator skill is retired in scope (~80% removed) in Phase 2, not Phase 1

It remains the source of truth for human-in-the-loop behaviour until controllers actually ship. Premature retirement strands the supervisor without guidance during the migration cohort.

### Decision 10: U-shaped compaction is deferred to Phase 3

It was a coping strategy for an overloaded orchestrator. The orchestrator's role shrinks dramatically in this design; investing in compaction now risks building a feature that becomes irrelevant. Revisit if Phase 2 telemetry shows the supervisor still hitting context limits. (Research report §5.2.)

### Decision 11: `next(id)` continues to return `handoff_prompt` in Phase 1, removed in Phase 2

Phase 1 needs the field as a transitional inspection surface for the supervisor. Phase 2 controllers own dispatch, so the supervisor no longer needs to see assembled prompts; the field becomes a temptation to revert and is removed. (Research report §5.2.)

---

## 7. Phasing

### Phase 1 — Lock the bypass and make the pipeline visible (2–4 weeks)

Goal: stop the bleeding. No architectural rewrites yet.

| Deliverable | Package | Status |
|---|---|---|
| Prompt assembly gate (P44-F1) | `internal/context` | Already designed; ship as drafted |
| `dispatch_task` MCP tool (only path to a sub-agent for the orchestrator) | `internal/dispatch` | New |
| Remove `spawn_agent` from orchestrator's tool list | MCP server tool registration | Code change + test |
| `Provider` interface with single Anthropic implementation | `internal/provider` | New |
| Golden-task evaluation harness (10 fixtures + CI hook) | `internal/eval` | New |
| Bypass-attempt test suite | `internal/eval` | New |
| `next(id)` returns `handoff_prompt` field | Existing handler | Small change |
| P56 bug-lifecycle gate enforcement (`CheckBugTransitionGate`, `bugStopStates`, review-report requirement) | Existing entity service | Pure code fix |
| P55 procedural fixes as a *time-bounded bridge* | Skills, roles, MCP tool-list trimming | Bridge only |

**Phase 1 exit criteria:**
- `spawn_agent` is not in the orchestrator's tool set.
- All ten golden tasks pass; CI fails on diff.
- Bypass-attempt suite passes.
- At least three features have closed using `dispatch_task` end-to-end.
- P56 bug gates are enforced; no bug closes without a review-report doc.

### Phase 2 — Move dispatch into the server (1–3 months)

Goal: the orchestrator stops being a dispatcher.

| Deliverable | Package | Notes |
|---|---|---|
| `internal/durable` — minimal durable-execution layer | New | ~500 LOC; persistent state, retries, signals, replay |
| `internal/controller` — developing, reviewing, verifying controllers | New | Triggered by transition hooks; dispatch via internal `dispatch_task` semantics |
| `internal/verifier` — deterministic check layer + LLM verifier orchestration | New | Two-layered as per Decision 3 |
| `internal/audit` — append-only JSONL event log + `kbz audit query` CLI | New | Schema as in §5.5 |
| `internal/policy` — facade over existing rule sources, then migrate callers | New | Phase 2.0 → 2.3 sub-phases |
| End-to-end controller harness | `internal/eval` | Stub provider; assert gate, dispatch, verifier, audit invariants |
| `entity.controller_managed` feature flag | Existing entity model | Per-entity opt-in |
| Drift-detection job (weekly sampling) | `internal/audit` + cron | Reports DoD compliance |
| Orchestrator skill (`orchestrate-development.md`) shrunk to activate/monitor/exceptions | `.kbz/skills/orchestrate-development/` | Retire dispatch/composition/verification text |
| Remove `next(id) → handoff_prompt` field | Existing handler | Now a temptation surface |

**Phase 2 exit criteria:**
- 20+ features have closed under controller management with no manual prompt composition by the supervisor.
- Audit log shows deterministic checks running on every verification, verifier verdicts structurally validated, zero unrecorded dispatches.
- Drift-detection job has run for ≥4 weeks with no alarms.
- The orchestrator skill no longer mentions `handoff`, `spawn_agent`, or prompt composition.

### Phase 3 — Optimisations (3–9 months)

Goal: cost/quality optimisation on a stable substrate.

| Deliverable | Notes |
|---|---|
| Multi-provider routing (original P44 scope) | RouteLLM-style classifier; per-category model mapping; provider fallback; provider-side prompt caching |
| Per-stage success-rate / cycle-count / verifier-pass-rate dashboards | Reads from audit log |
| Continuous regression monitoring with provider-pin alarms | Reads from audit log + golden-task results |
| U-shaped compaction (only if telemetry shows the supervisor needs it) | Original P44 §Compaction work |

**Phase 3 entry criteria:**
- Phase 2 exit criteria all met.
- ≥30 days of audit-log data to anchor routing decisions.

---

## 8. Migration Strategy

### 8.1 Per-entity feature flag

`entity.controller_managed: bool` (defaults to false in Phase 2.0; flips to true in Phase 2.N once the cohort proves out).

- **New features (Phase 2 onwards):** created with `controller_managed: true`.
- **In-flight features:** retain `controller_managed: false`; finish under the legacy orchestrator path.
- **Cohort:** at least 20 controller-managed features must close cleanly before the default flips.
- **Rollback:** flipping a single entity back to `controller_managed: false` returns it to the legacy path mid-flight; controllers respect the flag at every transition hook.

### 8.2 Coexistence rules

- Both paths share `internal/policy` once it lands; legacy code reads it via the same facade.
- Both paths share `internal/audit`; legacy dispatches still emit audit rows so the harness covers them.
- The bypass-attempt test suite covers both paths: legacy orchestrator must use `dispatch_task` (no `spawn_agent`); controller must dispatch through the gate.

### 8.3 Rollback plan

If Phase 2 controllers cause regressions:
1. Flip the default flag back to `false`.
2. Existing controller-managed features pause at next transition; supervisor takes over via legacy path.
3. Investigate via audit log; fix; re-enable on a single canary feature; widen.

---

## 9. Alternatives Considered

### 9.1 Keep multi-provider routing as Phase 1 (original P44 scope)

Rejected. The research report is unambiguous: routing without an eval harness silently regresses quality, and the controller migration is the load-bearing reliability fix. Shipping both at once masks each other's regressions.

### 9.2 Adopt Temporal or Restate as the durable-execution layer in Phase 1

Rejected for Phase 1; preserved as a Phase 3+ option. Operational dependency and learning curve outweigh the benefit at single-developer scale. The in-tree layer is ~500 LOC and replaceable.

### 9.3 Single-layer LLM verifier (the previous P44 design)

Rejected. The research report (§3.4, §5.3 item 3) is explicit: deferring DoD enforcement to an LLM with a markdown checklist reproduces the failure mode one layer down. Two-layered verification is non-negotiable.

### 9.4 Big-bang controller migration (no feature flag)

Rejected. Strands in-flight features and concentrates risk. Per-entity flag is the standard answer.

### 9.5 Build a "prompter sub-agent" that composes prompts on the orchestrator's behalf

Rejected (also rejected in P44-F1 Alternative C). The research report §8.3 is explicit: this is a band-aid for a problem the controller architecture eliminates.

### 9.6 Keep `spawn_agent` in the orchestrator's tool list with a strong policy rule against bare prompts

Rejected. The four-plan record is the empirical refutation: every prose rule is bypassed. Tool affordance is the only mechanism the research report endorses (Anthropic tool-use guide; §3.5).

### 9.7 Embed policy in YAML rather than code

Rejected for the engine itself. YAML is a fine *source* of policy data (stage bindings, role files) but the *engine* — the code that resolves "is this transition allowed?" — must be testable, debuggable Go. The engine reads YAML; the YAML doesn't define the engine.

---

## 10. Risks and Mitigations

(Lifted from the research report §9 with additions specific to this design.)

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Pipeline becomes invisibly degraded after Phase 2 | High | High | Phase 1 eval harness + Phase 2 audit log + Phase 2 drift detection job |
| Stage controller bugs block all transitions | Medium | High | Per-entity feature flag; durable-execution replay; small first cohort |
| Sync controllers stall the supervisor | High if naively built | Medium | Decision 2: async + durable from day one |
| `dispatch_task` becomes "another rule" if `spawn_agent` remains in tool list | High | High | Decision 1: removed in Phase 1; bypass-attempt test suite enforces |
| LLM verifier silently passes failing DoD items | Medium | High | Decision 3: deterministic Go layer always runs and can independently fail |
| MCP server becomes a god-object | Medium | Medium | Hard package boundaries (`internal/controller`, `internal/policy`, `internal/audit`, `internal/dispatch`, `internal/verifier`, `internal/eval`, `internal/durable`); no cross-package shortcuts |
| In-tree durable-execution layer accumulates undocumented edge cases | Medium | Medium | Cap at ~500 LOC; if it grows, that's the signal to evaluate Temporal/Restate |
| Multi-provider routing regresses quality silently in Phase 3 | High if shipped without harness | High | Phase 3 entry criterion: ≥30 days audit data; routing decisions diff-checked against single-provider baseline |
| Migration strands in-flight work | Medium | Low | Decision 8: per-entity feature flag |
| Supervisor still violates rules in remaining surface (transitions, exceptions) | Medium | Low | Now low-impact: supervisor cannot dispatch, cannot bypass gates, cannot self-verify; policy engine enforces what's left |
| Audit log grows unboundedly | Low | Low | Out of scope for Phase 2; revisit in Phase 3 with rotation/compaction |
| Eval harness golden tasks drift from real workload | Medium | Medium | Drift-detection job samples real closed features and adds new golden fixtures over time |

---

## 11. Dependencies

| Dependency | Phase | Notes |
|---|---|---|
| 3.0 context-assembly pipeline (`internal/context/pipeline.go`) | 1 | Existing; consumed by `dispatch_task` and Phase 2 controllers |
| Stage bindings (`.kbz/stage-bindings.yaml`) | 1, 2 | Existing; consumed by `internal/policy` facade |
| Role and skill content (`.kbz/roles/*.yaml`, `.kbz/skills/*/SKILL.md`) | 1, 2 | Existing; consumed by pipeline; **no longer relied on for control** |
| Entity / lifecycle data model | 1, 2 | Existing; extended in Phase 2 with `controller_managed` flag |
| MCP tool registration | 1 | Modified to support per-role tool lists in Phase 2 (policy-driven) |
| P55 procedural fixes | 1 (bridge) | Time-bounded; supersedes itself once Phase 2 ships |
| P56 bug-lifecycle gate enforcement | 1 | Pure code fix; aligned with this design's policy direction |
| P58 hardcoded default tool hints | 1 | Already folded into P44-F1 |

---

## 12. Open Questions

These remain open and should be resolved during specification, not deferred:

1. **Where does `internal/durable` persist state?** Options: SQLite (already present in Kanbanzai for some features?), a JSONL log per workflow, or a small custom append-only format. Decision should match what the rest of the project uses for durable storage.

2. **How are deterministic-check results surfaced to the supervisor on failure?** Via a checkpoint event consumed by the chat session, an entry in the audit log, both? The supervisor needs an actionable summary, not raw stderr.

3. **How does the LLM verifier handle items where deterministic evidence partially exists?** E.g. "tests pass" is deterministic, but "tests adequately cover the spec" is judgement and depends on having read the test code. Does the verifier sub-agent receive test file contents as part of its prompt, or only summaries?

4. **What does the per-role tool-permission enforcement look like at the MCP transport layer?** Today the server exposes one tool list to all callers. Per-role enforcement requires session-attributed tool lists; the MCP protocol may not natively support that. May require wrapping the transport.

5. **Does removing `next(id) → handoff_prompt` in Phase 2 affect any existing workflow that we haven't catalogued?** Audit usages before removal.

6. **Should `spawn_agent` removal apply to all roles, or only the orchestrator role?** Recommendation: only orchestrator in Phase 1 (other roles may legitimately need it for their own internal purposes); revisit in Phase 2 once controllers own all dispatch.

---

## 13. Cross-References

- **Research report:** `work/P44-model-routing-agent-launcher/P44-research-orchestrator-architecture.md` (the source of authority for this design's recommendations)
- **Companion design (Phase 1):** `work/P44-model-routing-agent-launcher/P44-F1-design-prompt-assembly-gate.md`
- **Superseded:** `work/P44-model-routing-agent-launcher/P44-design-model-routing-agent-launcher.md` (multi-provider routing — content preserved as Phase 3 feasibility input)
- **Superseded:** `work/P44-model-routing-agent-launcher/P44-design-feature-execution-pipeline.md` (stage controllers — content incorporated into §5.2 with sync/async resolved and dispatch-via-gate semantics added)
- **Related plans:** P50 (orchestrator drift incident), P55 (orchestrator context hygiene — bridge), P56 (bug lifecycle hardening — Phase 1), P57 (retrospective pipeline tightening), P58 (hardcoded tool hints — folded into P44-F1)

---

*End of design.*
