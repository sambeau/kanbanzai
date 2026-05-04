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
│   ├── providers/     (DeepSeek, Anthropic, OpenAI adapters)
│   ├── fallback/      (chain logic, health checks)
│   ├── categories/    (provider-agnostic category → model mapping)
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

**Research validation:** Three independent findings from the orchestration research support Option C over Option B:
- §7.5 "Expose Orchestration as MCP Tools, Not a Framework": the orchestration state machine should be exposed as MCP tools that an agent calls in a conversation loop — not as a separate daemon. Option B (separate `kbz-route` server) would create a separate server needing its own protocol. Starting embedded keeps model routing as an MCP tool (`dispatch_task`) within the same server the orchestrator already calls.
- §6 "Should kanbanzai Build Its Own Orchestration?": "An external orchestrator could call `context_assemble` — but then it is just a caller. kanbanzai is still doing all the work." The same logic applies to model routing: the value is in integrating model selection with the context Kanbanzai already produces — tighter in one binary than across two servers.
- Anthropic (building effective agents, 2024): "The most successful implementations weren't using complex frameworks or specialized libraries. Instead, they were building with simple, composable patterns." Option C is the simple, composable choice. Start with one binary; extract when evidence warrants.
- **DeepSeek dual-format validation (July 2026):** DeepSeek's API supports both Anthropic Messages and OpenAI Chat Completions formats. A single `Provider` interface implemented by `DeepSeekProvider`, `AnthropicProvider`, and `OpenAIProvider` proves the boundary is clean: the routing layer sees only `Provider.ChatCompletion()`, while each adapter handles its own wire format internally. This is concrete evidence that the `internal/routing/` package boundary is sufficient for extraction if model routing proves useful beyond Kanbanzai.

### Provider Integration Surface

Minimum viable:

| Provider | API | Models | Thinking control | Context caching | Notes |
|----------|-----|--------|------------------|-----------------|-------|
| Anthropic | Messages API | Claude Opus, Sonnet, Haiku | Extended thinking budget | Prompt caching (`cache_control` markers) | Phase 1 provider |
| DeepSeek | Chat Completions (OpenAI format) or Messages API (Anthropic format) | deepseek-v4-pro, deepseek-v4-flash | `thinking.type` + `reasoning_effort` | Automatic KV cache (disk-based, best-effort) | Dual protocol; strict mode tool calling (Beta); 1M context; pricing ≈100× cheaper than Anthropic on input |
| OpenAI | Chat Completions | GPT-5.4, GPT-5.4-mini | reasoning_effort parameter | Prompt caching | Phase 2 provider |

Future: Google (Gemini), MiniMax, Kimi — but start with these three.

**DeepSeek protocol choice:** DeepSeek supports both OpenAI Chat Completions and Anthropic Messages API. This design uses the OpenAI Chat Completions format for DeepSeek to share the maximum code path with the OpenAI Phase 2 integration and to access DeepSeek-specific features (strict mode tool calling, automatic cache visibility). See [DeepSeek API Analysis Report](../P41-opencode-ecosystem-features/P41-report-deepseek-api-analysis.md) §6 for the full rationale.

**DeepSeek V4 replaces V3/R1:** The deprecated `deepseek-chat` (→ flash non-thinking) and `deepseek-reasoner` (→ flash thinking) were removed 2026-07-24. Use only `deepseek-v4-pro` and `deepseek-v4-flash`. See the analysis report §8 Risk 4.

### Category System

Categories express task intent — cognitive profile, not provider preference. They are **provider-agnostic**. The mapping from category to provider+model is a separate configuration layer (`internal/routing/categories/`), not part of the category definition. Inspired by OmO but adapted to Kanbanzai's role system.

#### Category Definitions (Provider-Agnostic)

| Category | Cognitive profile | Thinking | Temperature | Orchestration pattern | Agent count |
|----------|-------------------|----------|-------------|----------------------|-------------|
| `deep-reasoning` | Novel reasoning, multi-step planning, architecture, spec-writing, complex debugging | **enabled** (`max` effort) | N/A (disabled by thinking) | Single agent, no parallelism | 1 |
| `implementation` | Pattern-matching, code generation, following patterns | disabled | 0.3 | Orchestrator-workers | 1 + N workers |
| `quick` | Simple fixes, typos, documentation | disabled | 0.3 | Single agent | 1 |
| `review` | Evaluative, finding classification, determinism needed | **disabled** | 0.1 | Maker-checker or panel | 1–3 |
| `audit` | Compliance, validation, repeatability critical | **disabled** | 0.0 | Single agent, low temperature | 1 |

**Critical design constraint — thinking vs. temperature irreconcilability:** When thinking mode is enabled, `temperature`, `top_p`, `presence_penalty`, and `frequency_penalty` are silently ignored by all major providers (Anthropic extended thinking, OpenAI `reasoning_effort`, DeepSeek thinking mode). The `review` and `audit` categories require low temperature for deterministic, repeatable output. This means **thinking mode MUST be disabled for `review` and `audit` categories** — the requirements are irreconcilable. The router validates this at configuration load time: if a category specifies both `thinking: enabled` and `temperature: <value>`, the router rejects the configuration. See [DeepSeek API Analysis Report](../P41-opencode-ecosystem-features/P41-report-deepseek-api-analysis.md) §4.5.

#### Provider → Model Mapping (Configuration Layer)

This is the default mapping. It can be overridden in `.kbz/routing.yaml` without changing the category definitions above:

```yaml
# Default provider priority and model mapping (internal/routing/config.go defaults)
providers:
  - name: deepseek
    priority: 1                  # tried first for all categories
    models:
      deep-reasoning: deepseek-v4-pro
      implementation: deepseek-v4-flash
      quick: deepseek-v4-flash
      review: deepseek-v4-pro
      audit: deepseek-v4-pro
  - name: anthropic
    priority: 2
    models:
      deep-reasoning: claude-opus-4-20250514
      implementation: claude-sonnet-4-20250514
      quick: claude-haiku-3-5-20241022
      review: claude-opus-4-20250514
      audit: claude-opus-4-20250514
  - name: openai
    priority: 3
    models:
      deep-reasoning: gpt-5.4
      implementation: gpt-5.4
      quick: gpt-5.4-mini
      review: gpt-5.4
      audit: gpt-5.4
```

**Why DeepSeek leads:** DeepSeek V4 Flash is ~100× cheaper than Claude Opus on input with comparable capability on code-generation tasks. DeepSeek V4 Pro with thinking mode rivals or exceeds Claude Opus extended thinking while costing ~35× less (at regular pricing; ~175× less during the 75% discount window until 2026-05-31). Automatic context caching (no `cache_control` markers needed) provides additional cost savings with zero integration work. See the analysis report §2 and §7 for the full cost analysis.

**Model escalation within a category:** For complex `implementation` tasks (>3 files, >50K estimated tokens), the router can escalate from flash to pro (with thinking enabled). This is an adaptive routing decision, not a category change — the category remains `implementation` but the model changes based on task complexity heuristics.

Categories map to roles via stage bindings, not hardcoded. A `spec-validator` role might map to `audit` category; an `implementer-go` role might map to `implementation` category.

#### Category → Orchestration Pattern Rationale

The research (Google, "Towards a Science of Scaling Agent Systems", 2026) found that "architecture must match task structure" — applying the wrong orchestration pattern to a task type degrades performance 39–70%. The orchestration pattern is now part of each category definition above (see Category Definitions table), with explicit rationale per category:

| Category | Task structure | Why this pattern |
|----------|---------------|------------------|
| `deep-reasoning` | Sequential, low tool density | Google found "every multi-agent variant degraded performance by 39–70%" on sequential reasoning tasks. Single agent, no parallelism. |
| `implementation` | Parallelisable, high tool density | Orchestrator-workers improved throughput +81% on parallelisable tasks. |
| `quick` | Sequential, very low tool density | Single agent. Overhead of worker coordination exceeds task cost. |
| `review` | Evaluative, medium tool density | Maker-checker or panel provides independent verification paths. |
| `audit` | Evaluative, low tool density | Single agent at low temperature for consistency. Repeatability over creativity. |

**`quick` category constraint:** The `quick` category uses a weaker model for simple-looking tasks, but the research doesn't support "use a weaker model for simple tasks" as a general principle — it supports matching the model to the task's structural demands. Simple-looking tasks may have hidden sequential dependencies or dependency-tracking requirements that weaker reasoning can't handle. `quick` should only be used when the orchestrator explicitly determines the task is "low complexity, ≤1 file, no structural changes." If the orchestrator can't make that determination, default to `implementation`.

### What Becomes Possible

Features currently blocked by MCP-server blindness, and how model routing unlocks them:

| Feature | Current blocker | How model routing enables it |
|---------|----------------|------------------------------|
| **Thinking-level control** | MCP server can't set model params | Dispatch loop controls model, temperature, thinking mode per task |
| **Auto-compaction** | Can't see context utilization | API response metadata includes `usage.input_tokens`; compute utilization and trigger compaction |
| **True Ralph Loop** | Can't auto-compact, so loop exhausts context | Compaction + resume becomes automatic within the dispatch loop |
| **Provider fallback** | Single model, client-chosen | Fallback chains try providers in order until one succeeds |
| **Cost tracking** | No visibility into token usage | Per-request token counts from API metadata; aggregate per-feature, per-batch |
| **Context caching** | No control over prompt structure | Repeated system prompts + entity state form stable prefix; automatic KV cache on DeepSeek, explicit `cache_control` on Anthropic |
| **Strict mode tool calling** | Malformed tool call arguments | DeepSeek Beta `strict: true` forces JSON Schema adherence on MCP tool definitions; reduces orchestration-loop errors |

### Compaction: U-Shaped Continuation Prompt

When auto-compaction triggers, the dispatch loop produces a state-based U-shaped compaction artefact — not a prose summary. The research (`P41-research-context-compaction-summary.md`) evaluated four approaches and recommends state-based compaction with retrieval anchoring: structured machine-readable state plus KE-ID references that the fresh session resolves on demand.

**Why not summaries:** Prose summaries have three failure modes that make them unsuitable for agent handoff: (1) detail erosion — "reviewed three tasks, two passed" loses which tasks and why; (2) recency bias — the last few turns dominate, underrepresenting decisions made earlier; (3) no structured retrieval path — a KE-ID reference as text can't be passed to `knowledge(action: "get")`.

#### Compaction Artefact Template

```markdown
# Compaction Artefact: [Feature/Batch ID] — [Timestamp]

## Identity & Routing
[Brief: role, active feature, current phase — kept under 50 tokens total]
Vocabulary: [15-20 domain terms active in this session]

## Active Constraints
- NEVER [X] BECAUSE [Y]
- ALWAYS [A] BECAUSE [B]
- File ownership boundaries: [file → task mapping]
- Dependency locks: [task A blocks task B until …]

## Active State

### Done (since last compaction or session start)
| Task ID | Summary | Verdict |
|---------|---------|---------|
| TASK-042 | Implement gate validation | Passed |
| TASK-043 | Fix edge case in status tool | Passed |

### In Flight
| Task ID | Status | Current blocker |
|---------|--------|-----------------|
| TASK-044 | Active (sub-agent running) | Waiting for completion |

### Ready (next dispatch candidates)
| Task ID | Priority | Depends on |
|---------|----------|------------|
| TASK-045 | P1 | TASK-044 |
| TASK-046 | P2 | — |

## Active Decisions
[Only decisions still constraining current work. Not historical decisions whose effects are already captured in code or knowledge entries.]

- DEC-007: Use hash-anchored edit validation pattern → applies to all file modification tools
- DEC-012: Sub-agent parallelism capped at 4 for this batch → file ownership constraints

## Surfaced Knowledge
[KE-IDs to query in the fresh session. The fresh session calls knowledge(action: "get") for each.]

| KE-ID | Topic | Why relevant |
|-------|-------|-------------|
| KE-01KN5CXMBWSXE | edit_file worktree limitation | Affects TASK-045 implementation |
| KE-01KQ7TKTJ7YVB | Shell escaping in worktrees | Affects TASK-046 implementation |

## Continuation Anchor
Resume from: Phase 3 (Task Dispatch)
Next action: Check TASK-044 status → if done, dispatch TASK-045 + TASK-046 in parallel
Expected effort: 3-5 tool calls before first dispatch
```

#### Section Ordering (U-Shaped Attention)

The sections are ordered for the U-shaped attention curve (Liu et al. 2024, TACL — "Lost in the Middle"):

| Position | Attention | Section | Rationale |
|----------|-----------|---------|-----------|
| **Top** | **High** | Identity + Vocabulary | Routing signal — determines which knowledge clusters activate |
| **Near top** | **High** | Active Constraints | Hard rules must survive; peak attention ensures compliance |
| **Middle** | Lower | Active State | Tabular data survives attention degradation better than prose |
| **Near bottom** | Rising | Active Decisions + Surfaced Knowledge | Recency bias helps recall |
| **Bottom** | **High** | Continuation Anchor | End-of-context attention peak; this is what the agent acts on first |

#### What Gets Discarded

The compaction artefact explicitly **omits**:
- Task completion details (outcome captured by status + verdict)
- Historical reasoning chains (decision or knowledge entry preserved; chain is dead weight)
- Conversation structure ("You asked me to…" — scaffolding, not state)
- Raw tool outputs (once acted on, raw output is noise)
- Failed attempts whose conclusions are in knowledge entries (if KE-01KXYZ records the conclusion, the debugging turns don't need preservation)

#### Knowledge Graph Anchoring (the biggest token saver)

Instead of inlining knowledge content, the artefact inlines **KE-ID references** that the fresh session resolves on demand. This is the highest-confidence recommendation from the research:
- **Inlining:** 200-800 tokens per entry
- **Referencing:** ~15 tokens per entry (KE-ID + topic + relevance)
- **Saving:** 185-785 tokens per entry; ~1,000-8,000 tokens for a typical session

The fresh session runs `knowledge(action: "get", id: "KE-...")` for each reference at session start. This is a mandatory step — the session must resolve all referenced KE-IDs before any other work.

**`trimmed_context` section:** When the compaction artefact is assembled under a byte budget, entries that were active in the previous session but excluded from the artefact are listed in a `trimmed_context` section so the fresh session knows what it might be missing. This implements §7.4 of the landscape research: "budget awareness must be visible to the receiving agent."

#### Compaction Trigger Strategy

Graduated triggers at two levels of infrastructure maturity:

**Today (procedural — no model routing required):** The orchestrator estimates context pressure at each Phase boundary. At ~60%+ estimated utilisation, it writes the compaction artefact and instructs the human to start a fresh session. This is already partially described in the orchestration procedure.

**After model routing (P44 implementation):** The dispatch loop tracks actual token counts from API response metadata (`usage.input_tokens`):
- **60% utilisation → soft warning.** "Context pressure building. Complete current task dispatch cycle, then prepare for compaction."
- **80% utilisation → hard trigger.** Compaction happens at the next task boundary (never mid-task).
- **90%+ → emergency compaction.** Finish current tool call, compact immediately.

The graduated approach avoids single-threshold failures: compacting too early wastes capacity; compacting too late risks mid-task truncation. Used by both Claude Code and LeRiM.

**Hard cap:** The compaction artefact must be under 25% of the context window. Token budgets per section. Enforced in the template.

**`reasoning_content` and compaction (DeepSeek thinking mode):** When thinking mode is enabled, DeepSeek returns `reasoning_content` in the response. This content MUST be passed back on tool-call turns but is IGNORED on non-tool-call turns (see [analysis report](../P41-opencode-ecosystem-features/P41-report-deepseek-api-analysis.md) §4.3). Compaction is a hard conversation reset — `reasoning_content` from the old conversation is NOT carried forward into the compaction artefact. The continuation prompt starts a fresh conversation with no prior `reasoning_content`. This is correct behavior: the compaction artefact is self-contained and the fresh session has no tool-call history to reconcile.

#### Evaluation Metrics

Three metrics, tracked as a compound score:

1. **Task completion rate.** After compaction, does the fresh session complete the work that was in-flight? Binary yes/no + time-to-completion.
2. **Decision consistency.** Compare pre-compaction intent (which tasks to dispatch next) against post-compaction actions. High divergence means the fresh session is missing context or the artefact is ambiguous.
3. **Token efficiency.** `(Tasks completed post-compaction) / (Compaction artefact tokens + retrieval overhead tokens)`. Catches the failure mode where artefacts grow so large they consume more tokens than they save.

Log pre-compaction intent. After compaction, compare the fresh session's first dispatch decisions against that intent. Review the first 20 compaction events with human oversight before enabling automated triggers.

**References:** `refs/prompt-engineering-guide.md` (structural template foundation), `work/P41-opencode-ecosystem-features/P41-research-context-compaction-summary.md` (full research methodology, learnable compaction analysis, implementation checklist), `work/_project/research-orchestration-landscape-2025.md` §7.4 (budget awareness).

### Agent-Facing Token Budget Communication

Token tracking is not just a monitoring feature — it enables agent self-regulation. Research (Anthropic, 2025) found that "agents struggle to judge appropriate effort for different tasks, so we embedded scaling rules in the prompts." The `dispatch_task` tool communicates the token budget to the agent at dispatch time:

```
Task budget: 50,000 tokens
Used so far: 12,000 tokens (24%)
Remaining: 38,000 tokens

If you approach the limit, prioritize the remaining work or request continuation in a fresh session.
```

This transforms token tracking from a server-side metric into a self-regulation mechanism — the agent makes its own tradeoffs about what to invest tokens in.

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

## Implementation Phasing (MVP First)

Model routing is the largest architectural change in P41. The research repeatedly warns against building too much at once (Anthropic: "simple, composable patterns"; Microsoft: "use the lowest level of complexity that reliably meets your requirements"). The implementation must be phased.

**DeepSeek phasing rationale:** DeepSeek V4 was analysed against P44 requirements in the [DeepSeek API Analysis Report](../P41-opencode-ecosystem-features/P41-report-deepseek-api-analysis.md). Key findings: thinking mode is available (was listed as "not yet available" in the original design), context caching is automatic (no `cache_control` markers needed), and pricing is ~100× cheaper than Anthropic on input. DeepSeek is elevated from Phase 3 to Phase 1 based on this evidence. Starting with two providers from day one provides real fallback chains from MVP and eliminates cost pressure during validation.

**MVP (Phase 1):**
- 2 providers: Anthropic (Messages API) + DeepSeek (OpenAI Chat Completions format)
- 3 categories: `deep-reasoning` (deepseek-v4-pro + thinking:max, fallback Claude Opus), `implementation` (deepseek-v4-flash, fallback Claude Sonnet), `review` (deepseek-v4-pro + thinking:off + temp:0.1, fallback Claude Opus)
- Token tracking: report-only (no budget enforcement)
- Fallback chains: primary → secondary (no tertiary yet)
- No auto-compaction, no Ralph Loop — these require validated token tracking first

**Phase 2 (after MVP validated on real features):**
- Add OpenAI provider + tertiary fallback chains
- Add `audit` and `quick` categories
- Token budget enforcement (per-feature caps)
- Provider health checks (periodic `Health()` calls, auto-skip unhealthy providers)

**Phase 3 (after fast-track + Phase 2 stable):**
- Auto-compaction at threshold with U-shaped state-based compaction artefact (see Compaction section above)
- True Ralph Loop (continuous execution with automatic compaction and resume)
- `reasoning_content` lifecycle management for DeepSeek thinking-mode multi-turn conversations (see analysis report §4.3)
- Strict mode tool calling for MCP tool definitions (DeepSeek Beta endpoint)

**Phase 1 category scope change:** The original MVP had 2 categories (`deep-reasoning`, `implementation`). `review` is now included in Phase 1 because it's needed for P43's fast-track validators. Without `review`, validators can't route through a low-temperature model specific to their category.

**Compaction-specific phasing** (aligns with `P41-research-context-compaction-summary.md` implementation sequence):
- **Phase 3a (immediate, no dependencies):** Wire `entity` and `status` tool calls to auto-populate Active State section. Add KE-ID resolution step at session start (query all referenced KE-IDs). Add context pressure check at each Phase boundary in orchestration procedure. Template hard cap: ≤25% of context window.
- **Phase 3b (after model routing token tracking):** Replace manual estimation with token-count-based graduated triggers (60%/80%/90%). Automate compaction artefact generation. Begin logging evaluation metrics (task completion rate, decision consistency, token efficiency).
- **Phase 3c (evaluation-driven iteration):** Review first 20 compaction events against metrics. Tune template. A/B test U-shaped vs. flat ordering. Decide on per-model tuning.
- **Phase 4 (deferred):** Evaluate Memento/learnable compaction for agent trajectories. Requires model routing stable + sufficient training data.

## Dependencies

- This is a feasibility design only — no build dependencies
- If built: requires provider API keys in `.kbz/local.yaml`, provider SDK integration
- Unlocks: auto-compaction (§6.6), thinking-level control, true Ralph Loop (§6.8)
- No dependency on P42 or P43 — can be designed in parallel, but should not be built until they are stable
- When built, P43's `dispatch_validator` abstraction routes validators through the `audit` category automatically (see P43 Forward Compatibility)

## Open Questions

1. **Embedded vs. separate server:** Resolved by research validation — Option C (build together, extract later) is the correct initial choice. The `internal/routing/` package boundary keeps extraction viable if model routing proves useful beyond Kanbanzai. DeepSeek's dual-format API (both OpenAI and Anthropic protocols from a single provider) further validates Option C: a single `Provider` interface serves all three providers, demonstrating the boundary is clean enough to extract if needed.
2. **Minimum viable providers:** Resolved by DeepSeek V4 analysis — Phase 1 includes Anthropic + DeepSeek. DeepSeek V4 has thinking mode, 1M context, tool calling, and automatic context caching — capabilities that rival or exceed the Phase 1/2 providers. Dual protocol support (OpenAI + Anthropic formats) means DeepSeek adds ~200 lines of adapter code, not a new architecture. See [analysis report](../P41-opencode-ecosystem-features/P41-report-deepseek-api-analysis.md) §2.1 for the full rationale.
3. **Token budget model:** Report-only in MVP. Per-feature budget enforcement in Phase 2 after usage patterns are understood from real feature data.
4. **Category granularity:** 3 categories for Phase 1 (deep-reasoning, implementation, review). Add `audit` and `quick` in Phase 2.
5. **Compaction artifact format:** The U-shaped continuation prompt with `trimmed_context` section as designed. Empirical validation happens in Phase 3; the design is ready to test. `reasoning_content` is NOT carried forward across compaction checkpoints — compaction is a hard conversation reset (see analysis report §4.4 Pattern D).
6. **Should validators always use a specific model?** Yes — validators have a different cognitive profile (compliance audit) than authors (creative synthesis). Research (Masters et al., 2025) shows audit tasks value consistency over creative depth. When model routing is built, validators route through the `audit` category (near-zero temperature, consistency-optimized). Until then, P43 uses same model with different temperature and role prompt via `spawn_agent`.
7. **Thinking vs. temperature irreconcilability:** Resolved — `review` and `audit` categories MUST disable thinking mode because they require low temperature for deterministic output. When thinking is enabled, `temperature` is silently ignored by all major providers. This is a hard design constraint enforced at configuration validation time (see analysis report §4.5).
8. **DeepSeek protocol choice:** Resolved — use OpenAI Chat Completions format for DeepSeek. Rationale: maximum code sharing with Phase 2 OpenAI integration, access to DeepSeek-specific features (strict mode tool calling, JSON mode). Trade-off: Phase 1 implements two protocols (Anthropic Messages + OpenAI Chat Completions). See analysis report §6.1.
