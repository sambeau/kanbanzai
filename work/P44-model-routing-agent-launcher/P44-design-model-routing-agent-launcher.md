# Design: Model Routing & Agent Launcher (Feasibility)

**Plan ID:** P44-model-routing-agent-launcher  
**Parent Plan:** [P41: OpenCode Ecosystem Features](../P41-opencode-ecosystem-features/P41-design-opencode-ecosystem-features.md)  
**Status:** Shaping — feasibility design only  
**Source:** [P41 Competitive Analysis](../P41-opencode-ecosystem-features/P41-research-competitive-analysis.md) §6.5, §7.1

## Overview

This is a **feasibility design** — evaluate whether and how Kanbanzai should own the agent dispatch loop, enabling model selection, thinking-level control, and provider fallback. Do not commit to building until P42 (hash-anchored edits) and P43 (fast-track architecture) are stable and prove the pattern of adopting ecosystem features.

The core problem: Kanbanzai's MCP server is blind to model selection. It receives tool calls and returns results — it has no visibility into or control over the client's model, temperature, thinking mode, or token budget. It can *suggest* in a prompt but cannot *enforce*. The only way to get real control is to own the dispatch loop.

This design evaluates two architectures: embedded model routing within `kbz serve`, and a separate model-routing MCP server.

## Goals and Non-Goals

**Goals (this design phase):**
- Evaluate feasibility of embedded vs. separate-server architectures
- Identify the minimum viable provider integration surface
- Map which currently-blocked features become possible (auto-compaction, thinking levels, true Ralph Loop)
- Estimate effort and risk
- Produce a decision: build, defer, or discard

**Goals (if built):**
- Dispatch tasks to different AI models based on semantic category or role
- Control thinking/reasoning depth per task (extended thinking on/off)
- Provider fallback chains (if primary is down, try next)
- Token usage tracking and cost management
- Unlock auto-compaction at threshold (can track context utilization from API metadata)
- Unlock true Ralph Loop (continuous execution with automatic compaction and resume)

**Non-Goals:**
- Not replacing the MCP-server architecture — this is additive
- Not replacing the host agent — the orchestrator still uses Kanbanzai's MCP tools for workflow decisions
- Not building a general-purpose agent platform — scoped to Kanbanzai's orchestration needs

## Design

### Architecture Options

#### Option A: Embedded (`internal/routing/` package in `kbz serve`)

```
kbz serve
├── MCP tools (entity, status, doc, handoff, ...)
├── internal/routing/
│   ├── providers/     (Anthropic, OpenAI, DeepSeek adapters)
│   ├── fallback/      (chain logic, health checks)
│   ├── categories/    (ultrabrain → model mapping)
│   └── tokens/        (usage tracking, budget enforcement)
└── dispatch loop      (spawn agent session, track tokens, return result)
```

**Pros:** Single binary, single config, tight integration with `handoff` context assembly. The orchestrator calls `dispatch_task(category: "deep", prompt: ...)` as a Kanbanzai MCP tool.

**Cons:** Kanbanzai becomes more complex — provider integrations, API key management, token tracking. Violates "always simpler than the project it manages" if the project doesn't need model routing.

#### Option B: Separate MCP Server (`kbz-route`)

```
kbz serve                     kbz-route
├── MCP tools                 ├── dispatch_task(category, prompt)
├── workflow state            ├── task_status(id)
├── entity hierarchy          ├── task_result(id)
└── document graph            ├── providers/
                              ├── fallback/
                              └── tokens/
```

**Pros:** Clean boundary — workflow management vs. model dispatch are different concerns. `kbz-route` can evolve at provider-speed without touching Kanbanzai's release cycle. Reusable by any MCP-based agent, not just Kanbanzai orchestrators. Kanbanzai stays simple.

**Cons:** Two servers to run and configure. Handoff friction — Kanbanzai's context assembly (entity hierarchy, document graph, knowledge) must serialize into `kbz-route`'s prompt format. The orchestrator bridges two servers for coupled decisions (which model depends on Kanbanzai context like role and stage).

#### Option C: Middle Ground (build together, extract later)

```
internal/routing/     ← clean package boundary within kbz serve
```

Ship as part of `kbz serve` initially. If model routing proves useful beyond Kanbanzai, extract to `kbz-route` as a packaging change, not an architecture change. The internal package boundary keeps the option open.

**Decision:** Start with Option C. The feasibility design should assume this approach while keeping the separate-server extraction path viable.

### Provider Integration Surface

Minimum viable:

| Provider | API | Models | Thinking control |
|----------|-----|--------|------------------|
| Anthropic | Messages API | Claude Opus, Sonnet, Haiku | Extended thinking budget |
| OpenAI | Chat Completions | GPT-5.4, GPT-5.4-mini | reasoning_effort parameter |
| DeepSeek | Chat Completions | DeepSeek-V3, R1 | Not yet available |

Future: Google (Gemini), MiniMax, Kimi — but start with these three.

### Category System

Categories map task intent to model preferences. Inspired by OmO but adapted to Kanbanzai's role system:

| Category | Use case | Preferred model | Fallback chain |
|----------|----------|-----------------|----------------|
| `deep-reasoning` | Architecture, spec-writing, complex debugging | Claude Opus (extended thinking) | GPT-5.4 → DeepSeek-R1 |
| `implementation` | Writing code, following patterns | Claude Sonnet | GPT-5.4 → DeepSeek-V3 |
| `quick` | Simple fixes, typos, documentation | Claude Haiku | GPT-5.4-mini |
| `review` | Code review, finding classification | GPT-5.4 (low temp) | Claude Opus |
| `audit` | Validators, compliance checks | GPT-5.4 (near-zero temp) | Claude Opus |

Categories map to roles via stage bindings, not hardcoded. A `spec-validator` role might map to `audit` category; an `implementer-go` role might map to `implementation` category.

### What Becomes Possible

Features currently blocked by MCP-server blindness, and how model routing unlocks them:

| Feature | Current blocker | How model routing enables it |
|---------|----------------|------------------------------|
| **Thinking-level control** | MCP server can't set model params | Dispatch loop controls model, temperature, thinking mode per task |
| **Auto-compaction** | Can't see context utilization | API response metadata includes `usage.input_tokens`; compute utilization and trigger compaction |
| **True Ralph Loop** | Can't auto-compact, so loop exhausts context | Compaction + resume becomes automatic within the dispatch loop |
| **Provider fallback** | Single model, client-chosen | Fallback chains try providers in order until one succeeds |
| **Cost tracking** | No visibility into token usage | Per-request token counts from API metadata; aggregate per-feature, per-batch |

### Compaction: U-Shaped Continuation Prompt

When auto-compaction triggers (at configurable threshold, e.g. 70%), the dispatch loop produces a U-shaped continuation prompt — not a summary. The prompt captures active state and positions the agent to resume:

```
You are continuing orchestration of FEAT-xxx.
Active state:
- Tasks done: T-001, T-002
- Task in flight: T-003 (dispatched, awaiting result)
- Ready frontier: T-004
- Active decisions: [decision summaries]
- Active constraints: [constraint summaries]
- Knowledge surfaced: KE-047, KE-089

Continue from Phase 2: identify the next dispatch batch.
```

See `refs/prompt-engineering-guide.md` for the structural template. A `compact-orchestration-session` skill is a promising research area once model routing makes this feasible.

## Alternatives Considered

### Single-model approach (status quo)

**Keep:** Kanbanzai stays simple. One model, client-chosen. No provider integrations.

**Reject for now:** The constraint is real — Kanbanzai can't control thinking levels, can't auto-compact, can't fall back. These are genuine gaps. But the cost of addressing them is high. The feasibility design evaluates whether the cost is justified — it doesn't assume the answer.

### MCP protocol extension

**Idea:** Extend the MCP protocol so servers can request model parameters (thinking mode, temperature) from the client. Kanbanzai would influence model selection without owning dispatch.

**Reject:** This requires changes to the MCP specification and all MCP clients. Kanbanzai can't control that timeline. Building our own dispatch loop is faster and more reliable.

### Only build the thinking-level feature, skip routing

**Idea:** Add a simple `thinking_level` parameter to `handoff` that the orchestrator sets. The MCP client would need to respect it — but it can't, because the protocol doesn't support it.

**Reject:** Same constraint as full model routing. There's no partial path — either Kanbanzai owns dispatch or it doesn't. Thinking levels, model selection, and auto-compaction are all the same architectural decision.

## Dependencies

- This is a feasibility design only — no build dependencies
- If built: requires provider API keys in `.kbz/local.yaml`, provider SDK integration
- Unlocks: auto-compaction (§6.6), thinking-level control, true Ralph Loop (§6.8)
- No dependency on P42 or P43 — can be designed in parallel, but should not be built until they are stable

## Open Questions

1. **Embedded vs. separate server:** Does the reuse argument (model routing useful outside Kanbanzai) justify the operational overhead of a separate server? Or is the middle ground (build together, extract later) sufficient?
2. **Minimum viable providers:** Is Anthropic + OpenAI sufficient for initial implementation? DeepSeek adds complexity but is strategically important for cost and independence.
3. **Token budget model:** Should Kanbanzai enforce per-feature or per-batch token budgets? Or just track and report?
4. **Category granularity:** Are 5 categories (deep-reasoning, implementation, quick, review, audit) the right granularity? OmO uses 8. Start small, add as needed.
5. **Compaction artifact format:** The U-shaped continuation prompt needs empirical validation. Does it actually produce better resumption than a summary? Test before building auto-compaction.
6. **Should validators always use a specific model?** The audit category maps validators to near-zero-temperature models. Is this worth the complexity, or do validators work fine on the default model with a different role prompt?
