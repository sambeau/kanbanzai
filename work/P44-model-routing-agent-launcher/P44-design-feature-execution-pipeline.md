| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-07T16:04:29Z           |
| Status | Draft                          |
| Author | sambeau                        |

## Overview

This design replaces the chat-based orchestrator dispatch model with a **server-managed feature execution pipeline**. Instead of an AI orchestrator composing prompts, dispatching sub-agents, and managing workflow transitions, the MCP server takes over the execution loop at each lifecycle stage. The orchestrator's role shrinks to: activate features, monitor progress, handle exceptions.

The pipeline is enforced in code — transition hooks fire on lifecycle state changes, run the 3.0 context assembly pipeline, dispatch role-specific sub-agents via provider APIs, and advance the feature when exit conditions are met. The orchestrator cannot skip, shortcut, or degrade the pipeline because it never touches it.

This is not a "Ralph Loop" — there is no `while true`, no continuous execution, no self-directed agent. It is a bounded pipeline with defined entry conditions, stage bodies, exit conditions, and a hard stop at Definition of Done.

## Goals and Non-Goals

**Goals:**
- Every sub-agent receives a fully assembled prompt (roles, skills, vocabulary, anti-patterns, tool hints, knowledge, procedure) — enforced in code, not by agent discipline
- The pipeline runs as a side effect of lifecycle transitions; the orchestrator transitions entities, the server does the rest
- Each pipeline stage has a defined role-specific sub-agent profile, bounded retry loops, and a clear exit condition
- The orchestrator's context is freed from dispatch mechanics — it only sees results and makes decisions
- Provider integration is the mechanism, not the feature — the design works with any provider

**Non-Goals:**
- Not continuous execution or "Ralph Loop" — the pipeline stops when DoD is achieved
- Not replacing the orchestrator entirely — it still activates features, handles exceptions, and makes strategic decisions
- Not a general-purpose agent platform — scoped to Kanbanzai's feature lifecycle
- Not modifying the 3.0 context assembly pipeline, roles, skills, or stage bindings — all remain as they are

## Problem and Motivation

### Evidence: four plans, same failure mode

The chat-based orchestrator consistently bypasses the prompt assembly pipeline. Despite explicit skill rules ("Always use `handoff`"), despite P58's hardcoded tool hints, despite a verified working pipeline — the orchestrator reads `next(id)` JSON and composes prompts by hand. Every sub-agent gets a bare prompt: no role identity, no vocabulary, no anti-patterns, no tool hints, no knowledge.

| Date | Plan | Observation |
|------|------|-------------|
| 2026-05-03 | P51 | Legacy fallback removed. Pipeline-only path established. Orchestrator could still bypass. |
| 2026-05-04 | P55/P56 | Bug lifecycle sub-agents received bare prompts — no role, no skill, no tools. |
| 2026-05-07 | P57 | P58 fix verified working via direct `handoff` call. All four P57 prompts still manually composed. |
| 2026-05-07 | P58 | Pipeline now produces all 11 sections including Available Tools. Orchestrator doesn't call it. |

### Root cause

Skills, roles, and prompts are advice. The chat-based orchestrator treats them as suggestions. Every fix that adds better advice (handoff rules, tool hints, default fallbacks) is another suggestion the orchestrator can ignore. The architecture is wrong at the structural level: **the orchestrator controls dispatch, so it controls prompt quality.**

### The fix

Move dispatch out of the orchestrator's hands entirely. Dispatch becomes a side effect of lifecycle transitions — enforced by transition hooks in the MCP server. The orchestrator transitions a feature to `developing`; the server runs the pipeline and dispatches implementers. The orchestrator never touches a prompt.

## Design

### Architecture overview

```
Feature lifecycle (orchestrator-controlled):
    designing → specifying → dev-planning → developing → reviewing → merging → verifying → done

Server-managed pipeline (within developing/reviewing/verifying):
    Transition hook fires
        │
        ▼
    Stage controller runs
        │
        ├── Assemble prompts via 3.0 pipeline (non-bypassable)
        ├── Dispatch sub-agents via provider API
        ├── Monitor completion (poll finish calls)
        ├── Handle rework loops (bounded retry)
        └── Advance feature when exit conditions met
```

### Stage controllers

Each lifecycle stage with sub-agent work gets a **stage controller** — a function invoked by the transition hook that manages the entire stage lifecycle.

#### Developing stage controller

```
Feature transitions to developing (orchestrator action)
    │
    ▼
DevelopingController(featureID):
    │
    ├── 1. Load ready tasks for feature
    ├── 2. For each task (parallel where safe):
    │       ├── Run 3.0 pipeline → assemble full prompt
    │       ├── Dispatch via provider (implementer-go role)
    │       └── Sub-agent implements, tests, calls finish()
    ├── 3. When all tasks terminal:
    │       ├── All done/not-planned/duplicate → advance to reviewing
    │       └── Any needs-rework → loop to step 2 (max 3 cycles)
    └── 4. On max cycles exceeded → checkpoint (human intervention)
```

**Sub-agent profile:** `implementer-go` role + `implement-task` skill. Full pipeline assembly: identity, role, orchestration, vocabulary, anti-patterns, available tools, procedure, output format, knowledge, evaluation criteria, retrieval anchors.

**Exit condition:** All tasks in terminal status (done/not-planned/duplicate).

**Bounded loop:** Max 3 review cycles (existing stage binding constraint). On cycle 3 failure → human checkpoint.

#### Reviewing stage controller

```
Feature transitions to reviewing (developing controller action)
    │
    ▼
ReviewingController(featureID):
    │
    ├── 1. Dispatch reviewer sub-agents in parallel:
    │       ├── reviewer-conformance + review-code skill
    │       ├── reviewer-quality + review-code skill
    │       ├── reviewer-security + review-code skill
    │       └── reviewer-testing + review-code skill
    ├── 2. Collate findings from all reviewers
    ├── 3. Classify findings:
    │       ├── blocking → create rework tasks, loop to developing
    │       └── non-blocking → record, proceed
    └── 4. No blocking findings → advance to merging
```

**Sub-agent profile:** Per-reviewer role + `review-code` skill. Full pipeline assembly with review-specific tool hints and rubrics.

**Exit condition:** No blocking findings across all reviewer dimensions.

**Bounded loop:** Same review cycle cap (max 3). Shared with developing controller.

#### Verifying stage controller

```
Feature transitions to verifying (merge hook action)
    │
    ▼
VerifyingController(featureID):
    │
    ├── 1. Dispatch verifier sub-agent:
    │       └── verifier role + verify-closeout skill
    ├── 2. Verifier runs 10-item DoD checklist
    ├── 3. Result:
    │       ├── all-pass → advance to done
    │       └── failures-found → report to orchestrator (no auto-loop)
    └── Stop.
```

**Sub-agent profile:** `verifier` role + `verify-closeout` skill. Clean context — the verifier receives only the feature ID and the checklist, no prior session context.

**Exit condition:** All 10 DoD items pass OR failures reported to orchestrator.

**No loop.** Verification is single-pass. Failures require human or orchestrator decision.

### Transition hook integration

The existing `StatusTransitionHook` interface is extended with stage controllers:

```go
// internal/service/transition_hooks.go

type StageController interface {
    // Execute runs the stage pipeline for a feature.
    // Returns the next status or an error.
    Execute(featureID string, stage string) (nextStatus string, err error)
}

type PipelineTransitionHook struct {
    developing *DevelopingController
    reviewing  *ReviewingController
    verifying  *VerifyingController
}

func (h *PipelineTransitionHook) AfterTransition(from, to string, entity model.Entity) error {
    switch to {
    case "developing":
        next, err := h.developing.Execute(entity.ID, "developing")
        if err != nil { return err }
        // Auto-transition to next stage when controller completes
        return h.entitySvc.UpdateStatus(..., next)
    case "reviewing":
        // Similar pattern
    case "verifying":
        // Similar pattern
    }
    return nil
}
```

**Key property:** The orchestrator transitions a feature to `developing`. The transition hook fires. The controller runs. The feature auto-advances through `reviewing → merging → verifying → done` without the orchestrator touching any intermediate step. The orchestrator's context never contains sub-agent prompts, tool outputs, or dispatch mechanics.

### Provider integration

The stage controllers use a `Provider` interface to dispatch sub-agents:

```go
type Provider interface {
    // Dispatch sends an assembled prompt to a model and returns the result.
    Dispatch(ctx context.Context, input DispatchInput) (DispatchResult, error)
}

type DispatchInput struct {
    Category string   // "implement", "review", "verify"
    Prompt   string   // fully assembled pipeline prompt
    Tools    []Tool   // MCP tool definitions available to the sub-agent
}

type DispatchResult struct {
    Completion string   // sub-agent's final response
    TokensUsed int      // from API metadata
    ToolCalls  int      // number of tool calls made
}
```

The provider encapsulates API key management, model selection, fallback chains, and token tracking. The stage controllers don't know which provider they're using — they just dispatch and receive results.

### What the orchestrator does

The orchestrator's job shrinks to:

1. **Activate features.** Transition features to `designing` (or later stages for retro-fix features). This is a single `entity(action: "transition")` call.
2. **Monitor progress.** Call `status(id: "FEAT-xxx")` to see which stage the feature is in, which tasks are done, and whether any blockers exist.
3. **Handle exceptions.** When a stage controller hits a checkpoint (review cycle cap, verification failure), the orchestrator decides: retry? re-scope? human escalation?
4. **Strategic decisions.** Should this feature proceed? Should the plan be re-scoped? Are there cross-feature conflicts?

The orchestrator never:
- Composes a prompt
- Calls `handoff` or `spawn_agent`
- Runs the 3.0 pipeline
- Decides when tasks are ready to dispatch
- Collates review findings manually
- Runs verification

### Human interaction points

| Point | Trigger | Human action |
|-------|---------|-------------|
| Spec approval | Feature enters specifying | Review and approve specification |
| Design approval | Feature enters designing | Review and approve design |
| Review cycle cap | 3 review cycles exhausted | Decide: override, re-scope, or cancel |
| Verification failure | DoD checklist fails | Decide: fix specific items, override, or rework |
| Exception | Provider failures, dispatch errors | Investigate and resolve |

The human can also call `status(id: "FEAT-xxx")` at any time to see progress, but they are not required to be in the loop for dispatch, review, merge, or verification.

### What this design does NOT change

- The 3.0 context assembly pipeline — same 11 sections, same role/skill resolution
- Stage bindings — same stage → role → skill mapping
- Roles and skills — same YAML files, same inheritance
- Tool hints — P58's hardcoded defaults continue to work
- `next(id)`, `handoff(task_id)`, `finish(task_id)` — all MCP tools remain; sub-agents still call `finish`
- The orchestrator skill — updated to reflect the new role but still exists for human-orchestrated workflows
- The Definition of Done — same 10-item checklist

## Alternatives Considered

### Alternative A: Keep chat-based orchestrator, add dispatch_task tool (current P44 approach)

Add a `dispatch_task(task_id, category)` MCP tool. The orchestrator calls it instead of `handoff → spawn_agent`. The tool runs the pipeline internally and dispatches via provider. The orchestrator still controls *when* to dispatch.

**Pros:** Simpler change. Less architectural risk. Builds on existing MCP tool patterns.

**Cons:** The orchestrator can still bypass it — call `spawn_agent` directly with a hand-composed prompt. Four data points show the orchestrator reliably takes shortcuts. `dispatch_task` becomes another piece of advice it can ignore.

**Verdict:** Rejected. Addresses the symptom (poor prompts) but not the root cause (orchestrator controls dispatch).

### Alternative B: Pre-generate prompts during dev-planning

Write the full handoff prompt into the dev-plan for each task. The orchestrator reads it from the document and forwards it to `spawn_agent`.

**Pros:** Zero runtime assembly. Prompts are reviewed alongside the plan.

**Cons:** Prompts become stale. If a role, skill, or knowledge entry changes between planning and implementation, the sub-agent gets outdated guidance. Adds dev-planning burden. Still relies on orchestrator choosing to use the pre-generated prompt.

**Verdict:** Rejected. Staleness risk and still gated on orchestrator discipline.

### Alternative C: One sub-agent per feature (no orchestrator at all)

When a feature reaches developing, dispatch a single sub-agent with the entire feature scope. No task decomposition needed. The sub-agent implements everything.

**Pros:** Simplest architecture. No parallelism complexity. No orchestrator needed.

**Cons:** Single sub-agent with large scope will saturate context. No parallel work. Review becomes a single-point pass/fail. Doesn't scale beyond trivial features.

**Verdict:** Rejected. Task decomposition exists for good reasons — parallel work, focused context, independent review.

## Decisions

### Decision 1: Dispatch is a side effect of lifecycle transitions, not an orchestrator action

**Context:** The orchestrator controls dispatch today. Four data points show it degrades prompts every time. Every fix that improves the pipeline (P51, P58) is bypassed.

**Rationale:** Moving dispatch into transition hooks makes it non-bypassable. The orchestrator cannot skip, shortcut, or degrade the pipeline because it never touches it. This is the only architecture where prompt quality is guaranteed.

**Consequences:**
- Positive: Every sub-agent gets a fully assembled prompt. Zero bypass paths.
- Positive: The orchestrator's context is freed from dispatch mechanics.
- Negative: Transition hooks become more complex. Failures in the hook block entity transitions.
- Negative: If the pipeline has a bug, it affects every dispatch — but this is also true today, just invisible.

### Decision 2: Stage controllers manage bounded loops with clear exit conditions

**Context:** Review cycles are already bounded (max 3). Developing has rework loops. But these are managed by the orchestrator today — inconsistently.

**Rationale:** Each controller encapsulates the loop for its stage. The orchestrator doesn't need to know about review cycle counts, rework task creation, or exit conditions. Controllers enforce bounded retry and escalate to checkpoints when limits are hit.

**Consequences:**
- Positive: Consistent loop behaviour across all features.
- Positive: Checkpoints are raised automatically when limits are hit — no orchestrator forgets.
- Negative: Less flexibility — controllers can't adapt to unusual situations. That's what checkpoints are for.

### Decision 3: The orchestrator remains but is demoted from dispatcher to supervisor

**Context:** The orchestrator is useful — it activates features, handles exceptions, makes strategic decisions. It's bad at dispatch mechanics.

**Rationale:** Keep the orchestrator for what it's good at (decisions, exceptions, strategy). Remove it from what it's bad at (prompt composition, dispatch timing, review collation). The orchestrator becomes a supervisor, not a dispatcher.

**Consequences:**
- Positive: Orchestrator context stays clean — decisions, not mechanics.
- Positive: Human can still interact with the orchestrator for strategic guidance.
- Negative: The orchestrator skill needs significant rewriting. It currently describes dispatch mechanics in detail.

### Decision 4: Provider integration is abstracted behind a single interface

**Context:** P44 originally focused on model routing — provider chains, category-based selection, thinking-level control. That's provider mechanics, not pipeline mechanics.

**Rationale:** The Provider interface is a seam. Today it dispatches to DeepSeek. Tomorrow it could dispatch to Anthropic, or to a local model, or to a mock for testing. The stage controllers don't care. Provider selection, fallback chains, and token tracking live behind the interface.

**Consequences:**
- Positive: Testable — mock providers for controller tests.
- Positive: Provider evolution (new models, new APIs) doesn't touch pipeline logic.
- Negative: The Provider interface is a new abstraction with its own design complexity.

## Dependencies

- **P51 (Handoff Pipeline Unification):** Done. Pipeline is the only assembly path.
- **P58 (Default Tool Hint Fallbacks):** Done. Pipeline produces complete prompts including Available Tools.
- **P42 (Hash-Anchored Edit Tool):** Done. `kanbanzai_edit_file` exists and is referenced in tool hints.
- **P43 (Fast-Track Architecture):** Ready, not done. Fast-track features may need a lighter controller variant — but the stage controller pattern supports this (pass `fast_track: true` to skip review cycles).

## Open Questions

1. **Should stage controllers run synchronously or asynchronously?** Synchronous means the transition hook blocks until the stage completes — potentially minutes for a developing stage with multiple tasks. Asynchronous means the transition returns immediately and the controller runs in a goroutine — but then the orchestrator sees "developing" status while dispatch is happening. Async with status polling seems right, but needs design.

2. **How does the orchestrator discover that a stage completed?** Currently, the orchestrator polls `status(id:)`. In the pipeline model, stage completion auto-advances the feature. The orchestrator would see the feature in `done` status on next poll. Is that sufficient, or do we need a notification mechanism?

3. **What happens when a sub-agent calls `finish` but tests fail?** The sub-agent's `finish` call records verification status. The stage controller reads it and decides: pass → mark task done, fail → re-dispatch with error context or create rework task. The exact re-dispatch logic needs specification.

4. **Should there be a "dry run" mode for the pipeline?** An orchestrator might want to see what would happen without actually dispatching. Useful for debugging and trust-building. Could be a `--dry-run` flag on the transition call or a separate `pipeline_preview` tool.

5. **How do we handle the transition from current orchestrator-driven workflow to pipeline-driven?** Existing features in developing/reviewing have an orchestrator managing them. Do we let them finish under the old model? Do we cut over immediately? Migration strategy needed.
