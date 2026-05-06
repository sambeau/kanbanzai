# Context Rot & Fast-Track: Strategy for Kanbanzai's Orchestration Architecture

| Field  | Value                         |
|--------|-------------------------------|
| Date   | 2026-05-06                    |
| Status | Draft                         |
| Author | P41 research synthesis        |
| Parent | P41-opencode-ecosystem-features |
| Audience | Head of Product, Head of Product Design, Senior Architects, Development Team |

---

## Executive Summary

### What happened

In May 2026, we tested the fast-track pipeline — our system for pushing small features and retro fixes straight from spec to implementation without human gates. The pipeline *infrastructure* works. But the AI orchestrator (the agent that runs the show) behaved as if it didn't know it was in fast-track mode. It stopped mid-pipeline to ask for confirmation — in a mode designed to have *zero* human gates. It wasted time implementing changes that already existed. It treated status summaries as implicit checkpoints.

We investigated. The root cause isn't a code bug. It's something deeper in how AI agents behave when their context windows fill up with instructions, tool outputs, and conversation history. The industry calls this **context rot**.

### The headline finding

**Context rot is real, but it's manageable — and the fix is already designed.** The orchestrator didn't fail because of an architectural flaw in how we build agent systems. It failed because it received the wrong instructions for the job. The full `orchestrate-development` procedure — designed for multi-feature batches with cohorts, merge schedules, and review cycles — was fed to an orchestrator running a single small feature with no review gates. The procedure said "stop at 60% context"; the feature tier said "never stop." The orchestrator followed the more recent instruction.

This is a **context engineering problem** — we gave the orchestrator a map designed for a cross-country road trip when it only needed to drive to the corner store. The map itself created confusion.

### What we recommend

**Short-term (this week):** Fix the instruction mismatch. Give fast-track features a lightweight orchestration profile that skips cohort management, merge scheduling, context offloading, and the "stop at 60%" rule. This is P52 — a behavioral profile within the existing orchestration skill, not a rewrite. Immediately fix the `handoff` tool so it correctly routes sub-agent roles (P51 — removing the legacy 2.0 pipeline fallback).

**Long-term (this quarter):** Build the model routing dispatch architecture (P44). This is the strategic fix: instead of the orchestrator manually composing prompts for sub-agents (and sometimes getting them wrong), a single `dispatch_task` tool handles everything — claiming the task, assembling the right context, calling the model, and returning results. The orchestrator never sees a raw prompt. The pipeline can't be bypassed.

**Validation track (in parallel):** P43's automated validators — which check spec completeness, plan traceability, and review quality — are already built and working. They run in fresh sessions to avoid context degradation. Continue expanding their coverage. They're the template for how all sub-agent work should work: clean session, focused context, defined rubric.

### Why this matters to Product

The fast-track pipeline is Kanbanzai's most distinctive innovation — no competitor has automated gates with risk-tiered bypass. When it works, features move from idea to implemented code without humans typing "LGTM" on mechanical checks. When the orchestrator stops mid-pipeline for confirmation, it undermines the whole value proposition: if the human has to watch anyway, why automate?

The fix is straightforward and doesn't require new infrastructure. P52 (fast-track behavioral profile) is a one-day skill documentation change. P51 (pipeline unification) is a one-day code cleanup. Together, they make fast-track actually fast.

---

## The Problem: Context Rot and the P50 Incident

### What is context rot?

AI agents — including Kanbanzai's orchestrator — operate with a finite "attention budget." Every instruction, every tool output, every piece of conversation history consumes that budget. As the budget fills up, the agent's behavior degrades. It forgets earlier instructions. It follows the most recent pattern rather than the original plan. It drifts.

Anthropic's engineering team (Sep 2025) describes this as "context rot" — "as the number of tokens in the context window increases, the model's ability to accurately recall information from that context decreases." It's not that the model can't *find* information in a large context — it can. It's that the model's *behavior* changes: it becomes less consistent, more confirmation-seeking, more likely to follow a familiar pattern than a specific instruction.

The academic foundation is the "lost in the middle" effect (Liu et al., 2024, published in TACL): language models pay most attention to information at the beginning and end of their context window. Information in the middle — even when explicitly marked as important — receives significantly less attention. Instructions given at session start (the "primacy effect") are powerful *until they're pushed far enough into the middle*. Then the most recent instructions (the "recency effect") dominate.

### What happened in P50

During the fast-track implementation test in May 2026, the orchestrator was given explicit fast-track instructions: "no human gates, no stops, no breaks." It still stopped three times:

1. After analysing remaining work (before starting implementation)
2. After implementing two tasks — asking whether to continue
3. After transitioning features to `reviewing` — producing a status table and waiting

Each stop was the orchestrator pattern-matching "summary output → wait for response." The fast-track instruction ("no stops") was present but buried in a context window that also contained the full `orchestrate-development` procedure with its "stop at 60% context" rule, cohort management phases, and merge scheduling. The procedure's structure created natural breakpoints. The orchestrator followed the structure, not the instruction.

Additionally, the orchestrator discovered mid-implementation that some tasks had already been done — "ghost work." A session-start audit could have caught this. And the `handoff` tool — which assembles prompts for sub-agents — was producing orchestrator training material (~9K tokens) when it should have been producing implementer instructions. The orchestrator recognized this mismatch and manually composed 12 custom prompts, each missing spec sections, knowledge entries, and role-grounded vocabulary.

### The critical distinction

This was **not** an architecture failure. The chat-as-orchestrator model — a single AI agent running the workflow in a conversation loop — can handle fast-track. The failure was in *what context we gave it*:

| What the orchestrator received | What it should have received |
|---|---|
| Full `orchestrate-development` skill: 6 phases, cohort setup, merge scheduling, context compaction, "stop at 60%" | Lightweight profile: session-start audit, dispatch loop, close-out only |
| `handoff` defaulting to orchestrator role/skill (~9K tokens of orchestration training) | `handoff` defaulting to `implementer-go` role/skill (~2K tokens of implementer instructions) |
| Same ~30KB of knowledge context repeated per task claim | Session-scoped context: full assembly once, task-specific only thereafter |
| "No stops" instruction at session start, buried by 40+ turns of conversation | "No stops" as a structural property of the profile — no breakpoints to stop at |

---

## Strategy: Three Horizons

### Horizon 0: Fix Fast-Track Now (This Sprint)

These are the immediate fixes that don't require new infrastructure. They address the specific failures observed in P50.

#### 1. Fast-Track Behavioral Profile (P52)

**What it is:** A lightweight alternate path within the `orchestrate-development` skill. When the feature's tier is `retro_fix` (or the batch is explicitly marked fast-track), the orchestrator follows a 3-phase flow instead of the full 6-phase procedure:

- **Phase 0: Session Start Audit.** Call `status()`, cross-reference task state against the dev-plan, identify ghost work (tasks whose changes already exist), classify dirty working-tree state, verify the running server binary is current.
- **Phase 1: Dispatch.** From the ready frontier (all `ready` tasks with satisfied dependencies), dispatch everything in parallel using `handoff(task_id, role: "implementer-go")`. Do not stop after dispatching. Poll until tasks complete. Immediately dispatch newly-unblocked tasks.
- **Phase 2: Close-Out.** When all tasks are terminal, transition features through to `done` or `reviewing`. Report completion. No status tables until all work is done.

**What's removed vs. full orchestration:** Cohort setup, merge scheduling, conflict analysis, context compaction, the "stop at 60%" rule, milestone pauses.

**Specifically added anti-patterns:**
- **Implicit Gate:** Stopping after a batch of work completes. **Only valid stop conditions:** all work done, build failure, missing dependency, or spec ambiguity.
- **Ghost Work Discovery:** Discovering mid-implementation that work is already done. **Fix:** Audit before claiming.
- **Milestone Pause:** Treating "feature transitioned" as a stop event. **Fix:** After a milestone, immediately proceed to the next action.

**Effort:** ~1 day. This is a skill documentation change — no code.

**Status:** P52 design is complete (see `work/P52-fast-track-orchestration/P52-design-fast-track-orchestration.md`).

#### 2. Handoff Pipeline Unification (P51)

**What it is:** The `handoff` tool currently has two paths: a modern 3.0 pipeline (roles, skills, vocabulary, knowledge, code graph) and a legacy 2.0 fallback. The legacy path activates silently when something is misconfigured and produces degraded prompts. P51 removes the legacy fallback entirely — misconfiguration produces a clear error instead of silently degraded output.

**The sub-agent role routing fix:** Currently, `handoff(task_id)` without an explicit `role` parameter defaults to the orchestrator's role and skill. For sub-agent dispatch (the common case), this is never correct — you never want an implementer to receive the orchestrator's training material. P51 changes the default: when the stage binding has sub-agents defined, `handoff` defaults to the sub-agent role (`implementer-go`) and skill (`implement-task`).

**Effort:** ~1 day. Removes dead code, simplifies the `handoff` function, and fixes a one-line default in the pipeline.

**Status:** P51 design is complete (see `work/P51-handoff-pipeline-unification/P51-design-handoff-pipeline-unification.md`).

#### 3. Context Budget Recalibration

**What it is:** Two stale constants are making context management more aggressive than intended:

- `assemblyDefaultBudget = 30,720` bytes — caps `handoff` and `next` responses, silently trimming knowledge entries and spec sections
- `DefaultContextWindowTokens = 200,000` — calibrates pipeline warning/refusal thresholds (40% warn, 60% refuse). Current models have 1M token windows, making these thresholds 5× more conservative than necessary.

**Fix:** Recalibrate `DefaultContextWindowTokens` to 1,000,000 and raise `assemblyDefaultBudget` (or eliminate it for `handoff` since P44 will make it irrelevant).

**Effort:** Configuration change. Minutes.

---

### Horizon 1: Build the Strategic Architecture (This Quarter)

These are the medium-term investments that address context rot at the architectural level.

#### 4. Model Routing Dispatch (P44)

**What it is:** The current sub-agent dispatch flow has three steps:

```
orchestrator calls next(id) → orchestrator calls handoff(id) → orchestrator calls spawn_agent(message=prompt)
```

Each step is a decision point where the orchestrator can diverge. In P50, it diverged at step 2: `handoff` produced orchestrator training material (~9K tokens) instead of implementer instructions, so the orchestrator discarded the output and wrote manual prompts. The manual prompts worked (all 12 tasks completed), but they lacked spec sections, knowledge entries, and code graph context that the pipeline would have included.

P44 collapses this into a single tool call:

```
orchestrator calls dispatch_task(task_id: "TASK-xxx", category: "implementation")
```

Internally, `dispatch_task`:
1. Claims the task (what `next` does)
2. Runs the 3.0 pipeline to assemble the correct role/skill/knowledge context
3. Routes to the appropriate model based on the task category
4. Calls the model API directly — no manual prompt composition possible
5. Returns structured results to the orchestrator

**The critical benefit:** The orchestrator never sees a raw prompt. The pipeline is *the* path — it can't be bypassed. Manual composition is impossible because there's no prompt to compose. The role routing problem P51 fixes at the `handoff` level is eliminated entirely at the `dispatch_task` level.

**Secondary benefit: Session-scoped context.** Currently, each `next` call returns ~30KB of context, most identical across tasks. `dispatch_task` assembles full context on the first call and only task-specific sections (spec section, file scope, acceptance criteria) on subsequent calls. For a batch of 12 tasks, this saves ~300KB of redundant context from never entering the orchestrator's conversation.

**Additional P44 features:**
- `fast_track` dispatch mode: automatically suppresses cohort management, merge scheduling, and context offloading
- Provider routing: maps task categories to specific models (deep-reasoning → Opus/DeepSeek V4 Pro, implementation → Sonnet/DeepSeek V4 Flash, review → Opus)
- Token budget communication: sub-agents know their remaining token budget and can self-regulate
- Automated compaction triggers: at 60% context utilisation (warning), 80% (hard trigger), 90% (emergency)

**Effort:** 4–6 weeks for Phase 1 (Anthropic + DeepSeek providers, `dispatch_task` tool, category system). Phase 2 adds automated compaction. Phase 3 adds evaluation and tuning.

**Status:** P44 feasibility design complete. Provider integration surface designed. Compaction artefact template designed (shared with P41). `dispatch_task` tool API designed.

#### 5. Context Compaction System

**What it is:** When the orchestrator's context fills up — even in a dispatch-loop architecture, for very large features — we compact it: extract the essential state and hand it to a fresh session. The research (P41, completed March 2026) evaluated four approaches and recommends:

- **State-based, not summary-based.** Don't write prose about "what happened." Write a structured snapshot of "where to resume from" — active tasks, active constraints, surfaced knowledge references, and a continuation anchor.
- **U-shaped ordering.** Place critical information at the beginning (identity, constraints) and end (continuation anchor) where models pay most attention. Place tabular data (task state) in the middle where attention is lower — tables survive degradation better than prose.
- **KE-ID anchoring.** Reference knowledge entries by ID instead of inlining content. Saves ~185-785 tokens per entry. The fresh session resolves them on demand.

**Effort:** Template designed. Trigger strategy designed. Implementation blocked on P44 for automated triggers; procedural triggers (orchestrator notices and acts) are unblocked today.

**Status:** Full research complete (`P41-research-context-compaction.md`). Implementation guide complete (`P41-research-context-compaction-summary.md`). Artefact template incorporated into P44 design.

---

### Horizon 2: Validate and Expand (Ongoing)

#### 6. Automated Validators (P43) — Already Built

P43's validators — spec-validator, plan-validator, review-gate-validator — are the working proof that fresh-session, rubric-based sub-agent dispatch works. Each validator:
- Runs in a clean session with only the document under review and the validator rubric
- Checks structural completeness and traceability (not creative quality)
- Produces a pass/block/needs-work verdict with specific findings

These are already in production use and have caught real issues (incomplete spec sections, untraceable requirements, missing acceptance criteria). The architecture is the template for `dispatch_task`: focused context, defined rubric, structured output.

**Next steps for P43:**
- Continue expanding validator coverage
- Add a design validator (structural completeness check for design documents)
- Tune blocking vs. non-blocking check thresholds based on observed false positive/negative rates

#### 7. Context Rot Monitoring

As we push more features through the system, we need to track whether context rot is actually occurring:

- **Goal drift detection.** Compare session-start constraints against mid-session decisions. If the orchestrator violates a constraint it stated at session start, flag it.
- **Context utilisation tracking.** Log tokens per task dispatch. If growth is superlinear, investigate the source.
- **Decision latency.** Track time and tool calls between task dispatch and completion. Increasing latency may indicate context saturation.

**Effort:** Instrumentation work. Add logging to `next`, `handoff`, and future `dispatch_task` calls.

---

## Architecture Decision: Why We're Building P44

The research (see `P41-research-context-pollution-and-rot.md` for full academic analysis) evaluated three architectural approaches:

| | Chat-as-Orchestrator + Mitigations | Model Routing Dispatch (P44) | State-Machine Orchestration |
|---|---|---|---|
| **Context rot risk** | Medium-High (growing conversation) | Low (orchestrator only sees decisions) | Very Low (no growing conversation) |
| **Flexibility** | High (handles edge cases) | High (retains decision authority) | Low (blocks on unexpected states) |
| **Implementation effort** | Small (incremental fixes) | Large (new tool, providers) | Very Large (full redesign) |
| **Pipeline enforcement** | Weak (can bypass `handoff`) | Strong (only path is `dispatch_task`) | Strong (deterministic transitions) |
| **Evidence confidence** | Medium (triangulated, not tested) | Medium-High (validated patterns) | Low (no production evidence) |
| **Migration path** | Current state | Incremental (P51 → P44 Phase 1 → 2 → 3) | Disruptive |

**The decision: Build P44.** The evidence strongly supports model routing dispatch as the correct strategic architecture. It addresses context rot at the root (the orchestrator never holds sub-agent context) while preserving the flexibility that makes chat-as-orchestrator valuable (handling edge cases, recovering from failures, making judgment calls).

**Why not state-machine orchestration?** Theoretically cleaner — deterministic transitions, no conversation growth — but practically inflexible. A state machine can't handle "the review found one ambiguous finding — should we escalate or accept?" or "task 4 and task 6 have a circular dependency, how do we break it?" These are real scenarios from Kanbanzai's own development. The hybrid approach — dispatch loop enforces pipeline use (deterministic constraint) while the orchestrator retains decision authority (flexible execution) — captures the best of both.

**Why not just keep fixing chat-as-orchestrator?** The near-term mitigations (P51, P52, budget recalibration) address the specific failures observed in P50. But they don't address the structural problems: per-claim context redundancy, the manual-prompt gap, the three-step dispatch sequence that creates three decision points where the orchestrator can diverge. Each mitigation adds instructions to an already-full context window. P44 removes the need for those instructions by making the right behavior the only possible behavior.

---

## Immediate Action Plan

### This Week

1. **Implement P52** — Add the fast-track behavioral profile to the `orchestrate-development` skill. Add anti-patterns (Implicit Gate, Ghost Work Discovery, Milestone Pause). Add Phase 0 (Session Start Audit). Remove cohort management, merge scheduling, and "stop at 60%" for fast-track features.

2. **Implement P51** — Remove the legacy 2.0 pipeline fallback from `handoff`. Fix sub-agent role routing: when sub-agents are defined, default to sub-agent role/skill instead of orchestrator role/skill. Update `orchestrate-development` to document `role: "implementer-go"` parameter.

3. **Recalibrate context budgets** — Update `DefaultContextWindowTokens` to 1,000,000. Raise `assemblyDefaultBudget` to stop silently trimming knowledge entries and spec sections.

4. **Test** — Run the P50 feature set again with P51 + P52 in place. Verify: no implicit gates, no ghost work discovered mid-implementation, sub-agents receiving implementer (not orchestrator) prompts.

### This Quarter

5. **Begin P44 Phase 1** — Build `dispatch_task` MCP tool. Integrate Anthropic + DeepSeek providers. Implement category system (deep-reasoning, implementation, review). Implement session-scoped context. Implement `fast_track` dispatch mode.

6. **Deploy procedural compaction** — Implement the U-shaped compaction artefact template. Add "context pressure check" to the orchestration procedure. Wire KE-ID resolution at session start.

### Ongoing

7. **Expand P43 validators** — Add design validator. Tune check thresholds. Track validator effectiveness metrics.

8. **Instrument context rot monitoring** — Log goal drift signals. Track context utilisation per task. Track decision latency.

---

## Risk Register

| Risk | Severity | Mitigation |
|------|----------|------------|
| P52 behavioral profile doesn't prevent all implicit gates | Medium | The profile removes the structural breakpoints that caused P50's stops. But the "summary output → wait" pattern is deep in training data. Monitor the first 5 fast-track runs for any remaining stop behavior. |
| P51 sub-agent role default breaks a use case where orchestrator role is intended | Low | The orchestrator role as default for `handoff` was never correct in practice — P50 proved it. If a rare case needs the orchestrator's context, the `role` parameter is still available. |
| P44 `dispatch_task` has no fallback if the pipeline produces wrong context | Medium | Currently, the orchestrator can bypass broken handoff output. `dispatch_task` removes that escape hatch. Mitigation: pipeline must be thoroughly tested before `dispatch_task` replaces `handoff` for production dispatch. |
| Cross-model compaction quality unknown | Low | Initial compaction uses same-model handoff (Claude → Claude). Cross-model testing deferred to Phase 3. |
| Large features still need compaction even with dispatch-loop | Low | The compaction system is already designed and can be implemented procedurally today. Automated triggers come with P44 Phase 2. |

---

## Summary: From Academic Research to Product Strategy

The research report (`P41-research-context-pollution-and-rot.md`) answers the question: "Is context rot an unavoidable failure mode in the chat-as-orchestrator architecture, or can it be managed?" The answer is clear:

**Context rot is a manageable consequence of the architecture, not an inherent limitation of AI-based orchestration.** The P50 incident was a context engineering failure — the orchestrator received contradictory instructions — not an architectural one. Three horizons of response:

- **Horizon 0 (now):** Fix the instruction mismatch. Fast-track profile (P52). Pipeline unification (P51). Budget recalibration. These are hours-to-days of work and address the P50 failures directly.

- **Horizon 1 (this quarter):** Build the strategic architecture. Model routing dispatch (P44). Compaction system. These are weeks of work and address context rot at the architectural level — the orchestrator never holds sub-agent context, the pipeline can't be bypassed, and context is assembled once per session rather than once per task.

- **Horizon 2 (ongoing):** Validate and expand. P43 validators set the pattern for fresh-session, rubric-based sub-agent work. Context rot monitoring ensures we catch degradation before it affects feature delivery.

The path forward is incremental. Each step is independently valuable, and each builds on the infrastructure of the previous step. P51 cleans up the pipeline so P44 has a single path to internalize. P52 gives fast-track features a lightweight profile now and a dispatch mode target for P44's `fast_track` flag. P43's validators prove the fresh-session pattern that `dispatch_task` generalizes.

No architecture reboot. No greenfield rewrite. Just: fix the immediate problems, then build the system that prevents them from recurring.

---

**Related documents:**
- `P41-research-context-pollution-and-rot.md` — Full academic research report with source gradings, methodology, and trade-off matrix
- `P41-research-context-compaction.md` — Compaction artefact research (20+ sources, 7 findings, 5 recommendations)
- `P41-research-context-compaction-summary.md` — Implementation guide with artefact template and trigger strategy
- `P44-design-model-routing-agent-launcher.md` — `dispatch_task` architecture, compaction system, enforcement analysis
- `P43-design-fast-track-architecture.md` — Validator architecture with fresh-session dispatch
- `P52-design-fast-track-orchestration.md` — Fast-track behavioral profile design
- `P51-design-handoff-pipeline-unification.md` — Pipeline unification and sub-agent role routing fix
