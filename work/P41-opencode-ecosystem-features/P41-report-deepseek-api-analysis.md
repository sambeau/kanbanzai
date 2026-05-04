# DeepSeek V4 API Analysis for P41/P44 Model Routing Features

**Type:** Research Report  
**Date:** 2026-07-29  
**Target Audience:** Software Engineering Team  
**Status:** Final  
**Related Documents:**
- [P41 Design: OpenCode Ecosystem Features](./P41-design-opencode-ecosystem-features.md)
- [P44 Design: Model Routing & Agent Launcher](../P44-model-routing-agent-launcher/P44-design-model-routing-agent-launcher.md)
- [Prompt Engineering Guide](../../refs/prompt-engineering-guide.md)

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Gap Analysis: P44 Design vs. DeepSeek API Reality](#2-gap-analysis-p44-design-vs-deepseek-api-reality)
3. [Feature-by-Feature DeepSeek Mapping](#3-feature-by-feature-deepseek-mapping)
4. [Thinking Mode Integration Guide](#4-thinking-mode-integration-guide)
5. [Context Caching Strategy for Kanbanzai](#5-context-caching-strategy-for-kanbanzai)
6. [Provider Adapter Design for `internal/routing/providers/`](#6-provider-adapter-design-for-internalroutingproviders)
7. [Cost Optimization Recommendations](#7-cost-optimization-recommendations)
8. [Risks and Mitigations](#8-risks-and-mitigations)
9. [Recommended Updates to P44 Design](#9-recommended-updates-to-p44-design)
10. [Quick Reference Appendix](#10-quick-reference-appendix)

---

## 1. Executive Summary

### High-Level Assessment

DeepSeek V4 (`deepseek-v4-pro` and `deepseek-v4-flash`) is **highly suitable** for Kanbanzai's model routing needs. The API provides every capability that P44's implementation phases target across all three providers — and in several dimensions it exceeds the Phase 1/2 provider choices (Anthropic, OpenAI):

| Capability | Anthropic (Phase 1) | OpenAI (Phase 2) | DeepSeek V4 | Winner |
|---|---|---|---|---|
| 1M token context window | Yes (Opus/Sonnet) | Yes (GPT-5.4) | Yes (both models) | Tie |
| Thinking/reasoning mode | Extended thinking | `reasoning_effort` | `thinking.type` + `reasoning_effort` | Tie |
| Tool calling | Native | Native | Full OpenAI-compat + `strict` mode (Beta) | DeepSeek (strict mode) |
| Context caching | Prompt caching | Prompt caching | Automatic KV cache (disk-based, best-effort) | DeepSeek (automatic) |
| Dual API format | Messages only | Chat Completions | **Both** Messages + Chat Completions | DeepSeek |
| Cost (per 1M input) | ~$15 (Opus) | ~$2.50 (GPT-5.4) | $0.14 (flash) / $1.74 ($0.435 discounted pro) | DeepSeek |
| Fallback safety | Single provider | Single provider | 2 models + dual API formats | DeepSeek |

### Key Findings

1. **DeepSeek should move from Phase 3 to Phase 1.** The P44 design defers DeepSeek to Phase 3 after "MVP validated on real features." The API analysis shows DeepSeek is richer than Anthropic in several areas (context caching visibility, dual protocol support, strict mode tool calling, automatic cache). There is no architectural reason to defer it. Starting with DeepSeek in Phase 1 (alongside Anthropic) provides a fallback path from day one and eliminates cost pressure during MVP validation.

2. **DeepSeek Flash rivals Claude Sonnet on capability while costing ~100x less on input.** At $0.14/M input tokens, flash makes cost a non-issue for routine `implementation` tasks. Even pro at the discounted rate ($0.435/M) is 6x cheaper than GPT-5.4.

3. **The automatic context caching changes the economics of Kanbanzai's orchestration pattern.** Repeated system prompts, entity state, and knowledge context across task dispatches will achieve high cache hit rates automatically — with zero code changes. This directly benefits Kanbanzai's multi-turn orchestration loop.

4. **Thinking mode `reasoning_content` management is the #1 integration risk.** The API requires `reasoning_content` to be passed back on tool-call turns but ignored on non-tool-call turns. Getting this wrong causes 400 errors. The integration guide in §4 provides exact code patterns.

5. **The dual API format (OpenAI + Anthropic) is a strategic advantage.** We can use the Anthropic Messages format for Phase 1 consistency, or the OpenAI Chat Completions format to share more code with the OpenAI Phase 2 provider. §6 provides a recommended approach.

---

## 2. Gap Analysis: P44 Design vs. DeepSeek API Reality

### 2.1 Should DeepSeek Move from Phase 3 to Phase 1 or Phase 2?

**Recommendation: Move DeepSeek to Phase 1, alongside Anthropic.**

The P44 design currently places DeepSeek in Phase 3 with rationale: "strategically important for cost and independence but adds complexity." The API analysis contradicts this rationale:

**DeepSeek adds minimal integration complexity:**
- It speaks both OpenAI Chat Completions and Anthropic Messages API — we can use whichever protocol we're already implementing for Phase 1/2.
- The API is a standard REST interface with Bearer auth — no custom SDK needed.
- Context caching is automatic (no `cache_control` markers like Anthropic) — no code changes required to benefit from it.

**DeepSeek reduces risk:**
- Having two providers from day one means fallback chains are real from MVP, not theoretical.
- If Anthropic has an outage during validation, DeepSeek provides continuity.
- The cost differential means we can run more validation iterations without budget pressure.

**The P44 phasing argument was build-around:**
- The original phasing (Anthropic → OpenAI → DeepSeek) was designed when DeepSeek-V3 was current and thinking mode wasn't available.
- DeepSeek V4 changes the calculus: it's now a first-tier provider with capabilities matching or exceeding the Phase 1/2 choices.

**Counter-argument and response:**
- *"Anthropic is the canonical provider — start with one, add others later."* This is sound engineering. But DeepSeek's compatibility with both API formats means we can treat it as a configuration change rather than an integration effort. A `deepseek` adapter that implements the same `Provider` interface as `anthropic` is ~200 lines of Go, not a new architecture.
- *"We don't know if DeepSeek quality is sufficient."* That's precisely why it should be in Phase 1 — we validate quality alongside Anthropic rather than discovering issues in Phase 3 after the architecture is locked.

**Revised phasing recommendation:**

| Phase | Original | Recommended |
|-------|----------|-------------|
| **Phase 1** | Anthropic-only | Anthropic + DeepSeek (flash for `implementation`, pro for `deep-reasoning`) |
| **Phase 2** | +OpenAI, +fallback chains | +OpenAI, +full fallback chains across all 3 |
| **Phase 3** | +DeepSeek, +auto-compaction, +Ralph Loop | +auto-compaction, +Ralph Loop (provider-agnostic) |

### 2.2 Category → DeepSeek Model Mapping

The P44 category system currently uses Anthropic-native models. Here is the recommended mapping with DeepSeek as a first-class option:

| Category | Original Preferred | DeepSeek Model | Thinking | Effort | Rationale |
|---|---|---|---|---|---|
| `deep-reasoning` | Claude Opus | `deepseek-v4-pro` | enabled | `max` | Architecture and spec-writing need deep reasoning; pro with max effort matches the cognitive profile. Flash should NOT be used here — the structural complexity of architecture tasks demands pro's full capability. |
| `implementation` | Claude Sonnet | `deepseek-v4-flash` | disabled | N/A | Writing code, following patterns. Flash is fast and cost-effective. Thinking mode off because implementation is pattern-matching, not novel reasoning. For complex multi-file refactors, escalate to pro with thinking. |
| `quick` | Claude Haiku | `deepseek-v4-flash` | disabled | N/A | Simple fixes, typos, docs. Flash with thinking off is the cheapest path. |
| `review` | GPT-5.4 (low temp) | `deepseek-v4-pro` | disabled | N/A | **Critical caveat:** thinking mode disables `temperature`. For review, we want low temperature for determinism. Use pro with thinking OFF and `temperature: 0.1`. Flash with thinking off + low temp is a cost-effective alternative for routine reviews. |
| `audit` | GPT-5.4 (near-zero) | `deepseek-v4-pro` | disabled | N/A | Same temperature constraint as review. Use pro with thinking OFF and `temperature: 0.0`. Validators need deterministic, repeatable output — thinking mode's non-determinism is counterproductive here. |

### 2.3 Updated Category → Provider/Model Mapping (All Three Providers)

This is the recommended mapping for the `internal/routing/categories/` configuration:

| Category | Primary | Secondary | Tertiary | Thinking | Temp | Orchestration |
|---|---|---|---|---|---|---|
| `deep-reasoning` | `deepseek-v4-pro` (thinking:max) | Claude Opus (extended) | GPT-5.4 (reasoning:high) | enabled | N/A | Single agent |
| `implementation` | `deepseek-v4-flash` (think:off) | Claude Sonnet | GPT-5.4 | disabled | 0.3 | Orchestrator-workers |
| `quick` | `deepseek-v4-flash` (think:off) | Claude Haiku | GPT-5.4-mini | disabled | 0.3 | Single agent |
| `review` | `deepseek-v4-pro` (think:off, temp:0.1) | GPT-5.4 (temp:0.1) | Claude Opus | disabled | 0.1 | Maker-checker |
| `audit` | `deepseek-v4-pro` (think:off, temp:0.0) | GPT-5.4 (temp:0.0) | Claude Opus | disabled | 0.0 | Single agent |

**Design rationale:** DeepSeek leads two categories (deep-reasoning, implementation) because of the cost/capability ratio. For review/audit, DeepSeek pro leads because it can run with thinking off + low temperature — matching the determinism requirement while being far cheaper than GPT-5.4. Claude Opus is retained as fallback for deep-reasoning because of its established quality on architecture tasks.

---

## 3. Feature-by-Feature DeepSeek Mapping

| P44 Feature | DeepSeek Support | Implementation Notes |
|---|---|---|
| **Thinking-level control** | `thinking.type` (enabled/disabled) + `reasoning_effort` (high/max) | Map P44 categories: `deep-reasoning` → `reasoning_effort: "max"`; `implementation`/`quick`/`review`/`audit` → `thinking.type: "disabled"`. The effort parameter is only meaningful when thinking is enabled. See §4 for integration patterns. |
| **Auto-compaction** | `usage.prompt_tokens` + context caching | `prompt_tokens` in the `usage` response gives total input tokens. Cache hit/miss breakdown (`prompt_cache_hit_tokens`, `prompt_cache_miss_tokens`) shows how much was cached. Compute utilization as `prompt_tokens / context_window` and trigger compaction at threshold (e.g., 70%). Cache hits reduce effective cost but don't change the utilization calculation — the model still processes all tokens. |
| **True Ralph Loop** | Context caching + thinking mode multi-turn | System prompts and entity state cache automatically across loop iterations. The critical requirement: `reasoning_content` MUST be passed back on tool-call turns. Multi-turn without tool calls doesn't need `reasoning_content`. See §4.4 for the exact concatenation pattern. |
| **Provider fallback** | Dynamic concurrency + error codes 429/503 | Implement retry with exponential backoff (1s, 2s, 4s, 8s, max 30s). 429 (rate limit) → retry after delay. 503 (overloaded) → retry or fail over to next provider. 401/402 → do NOT retry (auth/balance issues). See §10 for error code table. |
| **Cost tracking** | `usage` response with cache hit/miss breakdown | Per-request: `prompt_tokens`, `completion_tokens`, `prompt_cache_hit_tokens`, `prompt_cache_miss_tokens`. Aggregate per-feature and per-batch. Store in `tokens/` package. Compute cost from pricing table. Cache hit tokens cost 1/50th (flash) or 1/120th (pro) of miss tokens. |
| **Token budget communication** | `usage.prompt_tokens` + `max_tokens` | After each dispatch, update cumulative token usage for the feature. Communicate to agent at dispatch time: `Task budget: N tokens. Used so far: M tokens (X%). Remaining: R tokens.` See P44 §Agent-Facing Token Budget Communication. Compute utilization from `prompt_tokens` in the response, not the request — the response includes cached tokens in the count. |
| **Model selection** | Model parameter in request body | Simple string field: `"model": "deepseek-v4-pro"` or `"model": "deepseek-v4-flash"`. No special routing logic needed — the category system determines the model string. |
| **JSON output** | `response_format: {type: "json_object"}` | Guarantees valid JSON output. Useful for structured outputs from validators and auditors. Not needed for general task dispatch. |
| **Strict mode tool calling** | `strict: true` on function definitions + beta base URL | Forces the model to adhere strictly to JSON Schema for tool call arguments. This is a significant quality improvement for MCP tool definitions — reduces malformed tool calls. Requires `base_url: "https://api.deepseek.com/beta"`. See §6.4. |
| **Token counting (offline)** | `deepseek_tokenizer.zip` | Downloadable tokenizer for offline token counting. Use for pre-flight budget estimation without an API call. |

---

## 4. Thinking Mode Integration Guide

### 4.1 How to Toggle Thinking Mode Per Task Category

**OpenAI-compatible format (recommended for Kanbanzai):**

```json
// Deep reasoning task — thinking enabled with max effort
{
  "model": "deepseek-v4-pro",
  "messages": [...],
  "thinking": {
    "type": "enabled"
  },
  "reasoning_effort": "max"
}

// Implementation task — thinking disabled
{
  "model": "deepseek-v4-flash",
  "messages": [...],
  "thinking": {
    "type": "disabled"
  }
}
```

**Anthropic-compatible format:**

```json
{
  "model": "deepseek-v4-pro",
  "messages": [...],
  "thinking": {
    "type": "enabled"
  },
  "output_config": {
    "effort": "max"
  }
}
```

**Key detail:** `reasoning_effort` defaults to `"high"` for regular requests but the API auto-sets to `"max"` for complex agent requests (Claude Code, OpenCode). Kanbanzai should explicitly set the effort level rather than relying on auto-detection — it gives us deterministic control.

### 4.2 When to Use `reasoning_effort: "high"` vs. `"max"`

| Effort | Use in Kanbanzai | Rationale |
|---|---|---|
| `"high"` | Default for any thinking-enabled task | Good for specification authoring, design review, gap analysis. Sufficient reasoning depth without excessive token spend. |
| `"max"` | `deep-reasoning` category tasks | Architecture design, complex debugging, cross-cutting decisions. The 75% pro discount makes the extra token spend affordable during the discount window. |

**Decision rule for `internal/routing/categories/`:**

```
if category == "deep-reasoning":
    thinking = enabled, reasoning_effort = "max"
elif task_complexity == "multi-file-refactor" and model == "deepseek-v4-pro":
    thinking = enabled, reasoning_effort = "high"  // escalate implementation to thinking
else:
    thinking = disabled  // implementation, quick, review, audit
```

**Why review and audit should NOT use thinking mode:** Thinking mode disables `temperature`. Review and audit categories rely on low temperature for deterministic, repeatable output. A reviewer that produces different results on the same code is not useful. Use `thinking: disabled` + `temperature: 0.1` (review) or `0.0` (audit) instead.

### 4.3 The Critical `reasoning_content` Management Rules

This is the **#1 integration risk** for Kanbanzai's orchestration loop. The rules:

| Scenario | Rule | Consequence of getting it wrong |
|---|---|---|
| **Multi-turn WITHOUT tool calls** | `reasoning_content` from previous turns is IGNORED by the API. Do NOT pass it back. | Wasteful but not an error — extra tokens in the request but no functional impact. |
| **Multi-turn WITH tool calls** | `reasoning_content` MUST be passed back in ALL subsequent requests in the conversation. | API returns **HTTP 400** if `reasoning_content` is missing from any turn that follows a tool-call turn. |

**The simplest safe pattern (append the entire message object):**

After receiving a response, append the entire `response.choices[0].message` object to the messages array. This includes `role`, `content`, `tool_calls`, and `reasoning_content` (if present). This is safe for both tool-call and non-tool-call turns — it always includes `reasoning_content` when it exists, and the API handles it correctly (ignoring it on non-tool-call turns, requiring it on tool-call turns).

### 4.4 Code Patterns for Kanbanzai's Orchestration Loop

#### Pattern A: Initial dispatch (no prior reasoning_content)

```go
// Initial dispatch of a task — first message in the conversation
messages := []Message{
    {Role: "system", Content: systemPrompt},
    {Role: "user", Content: taskPrompt},
}

resp, err := client.ChatCompletion(ctx, ChatCompletionRequest{
    Model:    resolveModel(category),
    Messages: messages,
    Thinking: ThinkingConfig{Type: "enabled"},   // if deep-reasoning
    ReasoningEffort: "max",                       // if deep-reasoning
    Tools:    mcpToolDefinitions,
})
```

#### Pattern B: Processing the response (universal append pattern)

```go
// After receiving the response — always append the full message object
// This is safe regardless of whether the turn had tool calls or not.
assistantMsg := resp.Choices[0].Message
// assistantMsg contains: role, content, tool_calls (if any), reasoning_content (if thinking enabled)
messages = append(messages, assistantMsg)

// If the assistant made tool calls, execute them and append tool results
if len(assistantMsg.ToolCalls) > 0 {
    for _, tc := range assistantMsg.ToolCalls {
        result := executeMCPTool(tc.Function.Name, tc.Function.Arguments)
        messages = append(messages, Message{
            Role:       "tool",
            ToolCallID: tc.ID,
            Content:    result,
        })
    }
}
```

#### Pattern C: Continuation dispatch (subsequent turns in the loop)

```go
// Continue the conversation — messages already contain the full history
// including reasoning_content from previous tool-call turns
resp, err := client.ChatCompletion(ctx, ChatCompletionRequest{
    Model:    resolveModel(category),
    Messages: messages,  // full conversation history including reasoning_content
    Thinking: ThinkingConfig{Type: "enabled"},  // must match previous turns
    Tools:    mcpToolDefinitions,
})

// Again, append the full response message
messages = append(messages, resp.Choices[0].Message)
```

#### Pattern D: Compaction checkpoint (preserving reasoning_content)

When auto-compaction triggers, we produce a U-shaped continuation prompt. The `reasoning_content` from the conversation history is *not* carried forward (it would be invalid in a new prompt structure). This is correct behavior — compaction is a hard context reset, equivalent to starting a new conversation. The continuation prompt must be self-contained.

```go
// Compaction: start fresh conversation with U-shaped continuation prompt
messages := []Message{
    {Role: "system", Content: systemPrompt},
    {Role: "user", Content: continuationPrompt},  // includes trimmed_context section
}
// Note: no reasoning_content from the previous conversation is carried forward.
// This is a new conversation — the continuation prompt provides all necessary context.
```

### 4.5 Interaction with `temperature`/`top_p` in Thinking Mode

**When thinking mode is enabled:**
- `temperature`, `top_p`, `presence_penalty`, `frequency_penalty` are accepted without error but have **no effect**.
- The model's reasoning process determines output variability, not the sampling parameters.

**Implications for Kanbanzai categories:**

| Category | Thinking | Temperature behavior | Mitigation |
|---|---|---|---|
| `deep-reasoning` | enabled | `temperature` ignored | Accept inherent variability. Deep reasoning tasks benefit from exploration. |
| `implementation` | disabled | `temperature` works normally | Set `temperature: 0.3` for focused code generation. |
| `quick` | disabled | `temperature` works normally | Set `temperature: 0.3`. |
| `review` | **disabled** | `temperature` works normally | **Must disable thinking** to use `temperature: 0.1`. This is architecturally required — review needs deterministic output. |
| `audit` | **disabled** | `temperature` works normally | **Must disable thinking** to use `temperature: 0.0`. Validators must be repeatable. |

**Architectural implication:** The `review` and `audit` categories have an **irreconcilable conflict** with thinking mode. They require low temperature for determinism, but thinking mode disables temperature. This is not a DeepSeek-specific issue — it applies to any model where reasoning mode and temperature are mutually exclusive. The category system must enforce `thinking: disabled` for `review` and `audit`.

---

## 5. Context Caching Strategy for Kanbanzai

### 5.1 How Kanbanzai's Orchestration Pattern Benefits from Caching

Kanbanzai's orchestration pattern is **ideally suited** for DeepSeek's automatic context caching:

1. **Repeated system prompts** — every task dispatch within a feature uses the same system prompt (role identity, vocabulary, constraints). This is a long, stable prefix.
2. **Entity state** — feature status, task list, knowledge entries. These change slowly across dispatches.
3. **Knowledge context** — surfaced knowledge entries from the knowledge base. Stable across related tasks.

The cache works on **prefix matching**. Every request that starts with the same system prompt + entity state will hit the cache for that prefix, even if the task-specific instructions at the end differ.

**Concrete example for Kanbanzai:**

```
Request 1: [system_prompt (2K)] [entity_state (1K)] [knowledge (500)] [task_A_instructions (300)]
Request 2: [system_prompt (2K)] [entity_state (1K)] [knowledge (500)] [task_B_instructions (400)]
```

Request 2 will hit the cache for the first 3.5K tokens (system + entity + knowledge) because they match Request 1's prefix exactly. Only `task_B_instructions` (400 tokens) will be a cache miss.

### 5.2 Prompt Structure Design for Maximum Cache Hits

**The stable-prefix principle:** Structure every dispatch prompt with the same prefix ordering. Put variable content at the end.

```
[STABLE — cacheable prefix]
1. System prompt (role identity, vocabulary, constraints)
2. Entity context (feature status, task frontier, active decisions)
3. Knowledge context (surfaced knowledge entries for this feature)
4. Stage-specific instructions (from stage binding)
[VARIABLE — cache miss zone]
5. Task-specific instructions
6. Tool definitions (mostly stable, but may vary by role)
```

**Implementation in `internal/routing/`:**

```go
// PromptBuilder constructs prompts with stable prefix ordering
type PromptBuilder struct {
    systemPrompt    string  // stable: role + vocabulary
    entityContext   string  // semi-stable: changes slowly
    knowledgeCtx    string  // semi-stable: changes per batch
    stageInstr      string  // semi-stable: changes per stage
}

func (pb *PromptBuilder) Build(taskInstructions string, tools []Tool) []Message {
    return []Message{
        {Role: "system", Content: pb.systemPrompt + pb.entityContext + pb.knowledgeCtx + pb.stageInstr},
        {Role: "user",   Content: taskInstructions},
        // Tools passed separately in the API request, not in messages
    }
}
```

### 5.3 Using `user_id` for Cache Isolation

DeepSeek's cache is isolated at the `user_id` level. Different `user_id` values get separate cache namespaces.

**Strategy for Kanbanzai:**

```go
// Per-feature cache namespace — each feature gets its own cache
// This prevents cross-feature cache pollution and gives predictable hit rates
userID := fmt.Sprintf("kanbanzai:%s:%s", featureID, batchID)

req := ChatCompletionRequest{
    // ...
    User: userID,
}
```

**Why per-feature, not per-batch:** Features within a batch share entity context and knowledge, but the task-specific instructions are different enough that a per-batch cache would have lower hit rates. Per-feature caching ensures all tasks within a feature benefit from each other's cached prefixes.

**Alternative — per-batch:** If features within a batch share extensive common context (same knowledge entries, same design documents), per-batch caching could be more efficient. Start with per-feature; measure hit rates; switch to per-batch if empirical data supports it.

### 5.4 Cache Hit Monitoring

```go
// After each API call, log and track cache performance
type CacheMetrics struct {
    HitTokens  int     // prompt_cache_hit_tokens
    MissTokens int     // prompt_cache_miss_tokens
    HitRate    float64 // hit_tokens / (hit_tokens + miss_tokens)
    CostSaved  float64 // hit_tokens * (miss_price - hit_price)
}

func extractCacheMetrics(usage Usage) CacheMetrics {
    hit := usage.PromptCacheHitTokens
    miss := usage.PromptCacheMissTokens
    total := hit + miss
    if total == 0 {
        return CacheMetrics{}
    }
    return CacheMetrics{
        HitTokens:  hit,
        MissTokens: miss,
        HitRate:    float64(hit) / float64(total),
    }
}
```

Track these per-feature and per-batch. If hit rates fall below 50% for a given feature, investigate whether prompt structure has become unstable (e.g., entity context changing between every dispatch due to rapid task completion).

### 5.5 Best-Effort Caveats

**Design for cache misses as the normal case:**

1. **Cache construction takes seconds** — the first request with a new prefix will be a miss. Don't retry.
2. **Cache is automatically cleared after hours to days of inactivity** — don't rely on cache persistence across long sessions.
3. **Cache hits are NOT guaranteed** — 100% hit rate is impossible. Design cost estimates assuming 60–80% hit rate after warm-up.
4. **Partial prefix match does NOT hit** — if the system prompt changes even slightly between requests (e.g., a different knowledge entry is prepended), the cache misses for everything after that change.

**The key design rule:** Never build logic that assumes a cache hit. Cache hits are a cost optimization, not a functional requirement. If a cache miss occurs, the request still succeeds — it just costs more.

---

## 6. Provider Adapter Design for `internal/routing/providers/`

### 6.1 Recommended API Format: OpenAI Chat Completions

**Recommendation: Use the OpenAI Chat Completions format for DeepSeek integration.**

| Factor | OpenAI Format | Anthropic Format |
|---|---|---|
| **Shared code with Phase 2 (OpenAI)** | Maximum — same protocol, different base URL and auth | Minimal — different message structure, different tool format |
| **Ecosystem** | `go-openai` SDK, broader community | `anthropic-sdk-go`, smaller community |
| **DeepSeek feature support** | Full (thinking, tools, strict mode, JSON mode) | Limited (no strict mode, no `budget_tokens`, no MCP tools) |
| **Phase 1 consistency** | Different from Anthropic (adds integration work) | Same as Anthropic (reduces Phase 1 work) |

The trade-off: using Anthropic format for DeepSeek would reduce Phase 1 work (one protocol for both providers) but would limit DeepSeek features (strict mode tool calling, JSON mode) and create more work in Phase 2 (adding a new protocol for OpenAI). Using OpenAI format for DeepSeek means Phase 1 implements two protocols (Anthropic Messages + OpenAI Chat Completions) but Phase 2 adds zero new protocol work.

**Decision: Use OpenAI Chat Completions for DeepSeek.** The extra work in Phase 1 is justified by:
- Full access to DeepSeek features (strict mode tool calling)
- Zero protocol work in Phase 2
- Broader SDK ecosystem (shared with OpenAI integration)

### 6.2 Common Adapter Interface

```go
// Provider is the common interface for all model providers.
// Each provider (Anthropic, OpenAI, DeepSeek) implements this interface.
type Provider interface {
    // Name returns the provider identifier (e.g., "anthropic", "openai", "deepseek")
    Name() string

    // ChatCompletion sends a chat completion request and returns the response.
    // The Messages format is the internal canonical format — each provider
    // converts to its native wire format internally.
    ChatCompletion(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

    // Models returns the list of available models for this provider.
    Models() []string

    // SupportsThinking returns whether this provider/model supports thinking/reasoning mode.
    SupportsThinking(model string) bool

    // SupportsTemperature returns whether temperature is effective for this provider/model.
    // Thinking mode disables temperature on DeepSeek; this lets the router know.
    SupportsTemperature(model string, thinkingEnabled bool) bool

    // Health checks if the provider is currently reachable.
    Health(ctx context.Context) error
}

// ChatRequest is the canonical internal request format.
type ChatRequest struct {
    Model           string
    Messages        []Message
    Tools           []Tool
    Thinking        *ThinkingConfig   // nil = disabled
    ReasoningEffort string            // "high" or "max", only if thinking enabled
    Temperature     *float64          // nil = provider default
    MaxTokens       int
    User            string            // for cache isolation (DeepSeek)
}

// ChatResponse is the canonical internal response format.
type ChatResponse struct {
    Content          string
    ToolCalls        []ToolCall
    ReasoningContent string            // DeepSeek thinking content (empty if thinking disabled)
    Usage            Usage
}

type Usage struct {
    PromptTokens         int
    CompletionTokens     int
    PromptCacheHitTokens int  // DeepSeek-specific; 0 for providers without cache visibility
    PromptCacheMissTokens int // DeepSeek-specific
}
```

### 6.3 Provider-Specific Divergence

Each provider adapter handles protocol conversion internally. The routing layer only sees `Provider`:

```go
// deepseek/provider.go
type DeepSeekProvider struct {
    client   *openai.Client   // or raw HTTP client
    baseURL  string
    apiKey   string
}

func (p *DeepSeekProvider) ChatCompletion(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
    // Convert canonical ChatRequest → DeepSeek API request (OpenAI format)
    // Handle thinking config, reasoning_effort, user_id
    // Convert DeepSeek API response → canonical ChatResponse
    // Extract reasoning_content, cache hit/miss tokens
}

// anthropic/provider.go
type AnthropicProvider struct {
    client  *anthropic.Client
    apiKey  string
}

func (p *AnthropicProvider) ChatCompletion(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
    // Convert canonical ChatRequest → Anthropic Messages request
    // Handle extended thinking, cache_control
    // Convert Anthropic response → canonical ChatResponse
}
```

### 6.4 Strict Mode Tool Calling (Beta)

DeepSeek's `strict` mode forces the model to adhere strictly to JSON Schema for tool call arguments. This is accessed via the beta endpoint:

```go
// When strict mode is enabled, use the beta base URL
func (p *DeepSeekProvider) baseURL() string {
    if p.strictMode {
        return "https://api.deepseek.com/beta"
    }
    return "https://api.deepseek.com"
}

// Tool definitions with strict mode
func toDeepSeekTool(t Tool, strict bool) deepseekTool {
    dt := deepseekTool{
        Type: "function",
        Function: deepseekFunction{
            Name:        t.Name,
            Description: t.Description,
            Parameters:  t.Parameters,  // JSON Schema
        },
    }
    if strict {
        dt.Function.Strict = true

        // strict mode constraints: all properties must be required,
        // additionalProperties: false on all objects
        dt.Function.Parameters = enforceStrictSchema(t.Parameters)
    }
    return dt
}
```

**How strict mode benefits Kanbanzai's MCP tool definitions:**

Kanbanzai's MCP tools have well-defined JSON schemas (parameters for `entity`, `doc`, `knowledge`, etc.). Strict mode ensures the model produces valid tool calls — no hallucinated parameter names, no missing required fields. This directly reduces orchestration-loop errors.

**Trade-off:** Strict mode requires `additionalProperties: false` on all objects and all properties in `required`. This may require schema transformation for MCP tools that have optional properties. Start with non-strict mode; add strict mode as a quality improvement once the orchestration loop is stable.

---

## 7. Cost Optimization Recommendations

### 7.1 Flash vs. Pro Selection Criteria

| Criterion | Use `deepseek-v4-flash` | Use `deepseek-v4-pro` |
|---|---|---|
| **Task cognitive profile** | Pattern-matching, translation, boilerplate | Novel reasoning, architecture, multi-step planning |
| **Context size** | < 50K tokens | 50K+ tokens (pro's deeper reasoning handles long contexts better) |
| **Tool call complexity** | Simple tools, few parameters | Complex MCP tools with nested schemas, multi-tool sequences |
| **Output length** | < 4K tokens | 4K+ tokens (pro's 384K max output vs. flash's 384K — same ceiling, but pro quality degrades less at length) |
| **Error tolerance** | Retryable tasks (re-run if flash gets it wrong) | Non-retryable tasks (review findings, audit results) |
| **Cost sensitivity** | Budget-conscious (oss project) | High-value tasks where correctness > cost |
| **Recommended categories** | `implementation`, `quick` | `deep-reasoning`, `review`, `audit` |

**Decision heuristic in `internal/routing/categories/`:**

```go
func resolveModel(category string, taskComplexity TaskComplexity) string {
    switch category {
    case "deep-reasoning":
        return "deepseek-v4-pro"  // always pro for deep reasoning
    case "implementation":
        if taskComplexity.EstimatedTokens > 50000 || taskComplexity.FileCount > 3 {
            return "deepseek-v4-pro"  // escalate complex implementation to pro
        }
        return "deepseek-v4-flash"
    case "quick":
        return "deepseek-v4-flash"
    case "review", "audit":
        if taskComplexity.IsRoutine {
            return "deepseek-v4-flash"  // routine reviews can use flash
        }
        return "deepseek-v4-pro"  // architectural/security reviews need pro
    }
}
```

### 7.2 Cache Hit Economics

**Projected savings from structured prompt prefixes:**

Assume a typical Kanbanzai task dispatch:
- System prompt + entity context + knowledge: 3,500 tokens (cacheable prefix)
- Task-specific instructions: 500 tokens (variable)
- Total input: 4,000 tokens per dispatch

| Scenario | Cache hit rate | Effective input cost (flash) | Effective input cost (pro, discounted) |
|---|---|---|---|
| No caching | 0% | 4,000 × $0.14/M = $0.00056 | 4,000 × $0.435/M = $0.00174 |
| After warm-up (1st dispatch misses) | 87.5% | (500 × $0.14 + 3,500 × $0.0028)/M = $0.00008 | (500 × $0.435 + 3,500 × $0.003625)/M = $0.00023 |
| Fully cached (ideal) | 100% | 4,000 × $0.0028/M = $0.0000112 | 4,000 × $0.003625/M = $0.0000145 |

**Per-feature savings:** A feature with 20 task dispatches (~80K total input tokens):
- Without caching: $0.0112 (flash) or $0.1392 (pro)
- With caching (87.5% hit rate after first dispatch): $0.00166 (flash) or $0.01894 (pro)
- **Savings: 85–87% on input costs.**

These are small absolute numbers per feature, but they compound across the project. More importantly, caching makes cost growth sublinear with task count — the 20th dispatch costs the same as the 2nd.

### 7.3 The 75% Pro Discount Window

DeepSeek V4 Pro has a 75% discount from launch until **2026-05-31**, bringing the effective price from $1.74/M input to **$0.435/M input** and $3.48/M output to **$0.87/M output**.

**Implications for Phase 1 sequencing:**

1. **The discount window covers the entire P41/P44 development timeline** — we have ~10 months to use pro at flash-adjacent prices.
2. **Pro is cheaper during the discount than GPT-5.4 at regular pricing** ($0.435 vs. ~$2.50/M input). This means DeepSeek pro is both more capable (thinking mode) AND cheaper than the Phase 2 OpenAI option.
3. **After 2026-05-31, pro returns to $1.74/M input.** This is still 6x cheaper than GPT-5.4 and 10x cheaper than Claude Opus. The cost advantage doesn't disappear — it just narrows.
4. **Recommendation:** Use pro liberally during the discount window for `deep-reasoning`, `review`, and `audit` categories. Build the cost tracking infrastructure to monitor per-category spend. Before the discount expires, analyze usage patterns and adjust the flash/pro selection criteria based on empirical cost data.

---

## 8. Risks and Mitigations

### Risk 1: Rate Limiting (Dynamic, Non-Deterministic)

**Nature:** DeepSeek uses dynamic concurrency limiting based on server load. HTTP 429 responses are not predictable — they depend on global traffic, not just our usage.

**Impact on Kanbanzai:** A dispatch loop that hits rate limits will stall orchestration. If the orchestrator is waiting for a task result and the provider is rate-limited, the entire feature pipeline blocks.

**Mitigations:**

```go
// Exponential backoff with jitter for 429 responses
func retryWithBackoff(ctx context.Context, fn func() error, maxRetries int) error {
    base := 1 * time.Second
    max := 30 * time.Second
    for i := 0; i < maxRetries; i++ {
        err := fn()
        if err == nil {
            return nil
        }
        if !isRetryable(err) {
            return err  // 401, 402 — don't retry
        }
        backoff := time.Duration(float64(base) * math.Pow(2, float64(i)))
        if backoff > max {
            backoff = max
        }
        // Add jitter: ±25%
        jitter := time.Duration(float64(backoff) * (0.75 + rand.Float64()*0.5))
        time.Sleep(jitter)
    }
    return fmt.Errorf("max retries exceeded")
}

func isRetryable(err error) bool {
    // 429 (rate limit) and 503 (overloaded) are retryable
    // 401, 402, 400, 422 are not
    var apiErr *APIError
    if errors.As(err, &apiErr) {
        return apiErr.StatusCode == 429 || apiErr.StatusCode == 503
    }
    return false
}
```

**Additional mitigation:** Provider fallback. If DeepSeek returns 429 after 3 retries, fall back to the next provider in the chain. This is exactly the fallback pattern P44 already designs.

### Risk 2: 10-Minute Inference Start Timeout

**Nature:** DeepSeek closes the connection if inference hasn't started within 10 minutes of receiving the request.

**Impact:** Complex reasoning tasks with `reasoning_effort: "max"` could time out before the model starts generating. This is more likely during high-load periods.

**Mitigations:**
- Set client-side timeouts to >10 minutes for `deep-reasoning` category dispatches.
- Implement a pre-flight timeout: if `deep-reasoning` tasks consistently time out, fall back to a faster model or reduce `reasoning_effort` to `"high"`.
- The 10-minute limit is from request receipt to inference start, not total inference time. Once inference starts, there's no hard timeout on completion.

### Risk 3: Cache Best-Effort Nature

**Nature:** Cache hits are not guaranteed. Cache is cleared after hours/days of inactivity. Partial prefix match does not hit.

**Impact:** Cost projections based on cache hit assumptions may be optimistic. Orchestration logic must never assume a cache hit.

**Mitigation (design rule):** Treat cache hits as a cost optimization, not a functional requirement. Never build logic that branches on whether a cache hit occurred. The only cache-aware code should be in the cost tracking/metrics layer.

### Risk 4: Model Deprecation Timeline

**Nature:** `deepseek-chat` and `deepseek-reasoner` are deprecated and will be removed on **2026-07-24**. They map to `deepseek-v4-flash` (non-thinking and thinking variants, respectively).

**Impact on Kanbanzai:** This is already behind us (current date is 2026-07-29). We never used the deprecated models, so there's no migration burden.

**Mitigation:** Use only `deepseek-v4-pro` and `deepseek-v4-flash` in all code. Never reference `deepseek-chat` or `deepseek-reasoner` in configuration or documentation.

### Risk 5: Thinking Mode Disables Temperature

**Nature:** `temperature`, `top_p`, `presence_penalty`, and `frequency_penalty` are silently ignored when thinking mode is enabled.

**Impact:** The `review` and `audit` categories require low temperature for deterministic output. If thinking mode is accidentally enabled for these categories, temperature is ignored and output becomes non-deterministic.

**Mitigation:** The `SupportsTemperature` method on the `Provider` interface lets the router validate category configuration at dispatch time. If a category requires `temperature: 0.1` and the model+thinking combination doesn't support it, the router should refuse to dispatch and log an error.

```go
func (r *Router) validateDispatch(req *DispatchRequest) error {
    provider := r.resolveProvider(req.Category)
    model := r.resolveModel(req.Category)

    if req.Temperature != nil && !provider.SupportsTemperature(model, req.Thinking != nil) {
        return fmt.Errorf(
            "category %s requires temperature %.1f but model %s with thinking=%v does not support it",
            req.Category, *req.Temperature, model, req.Thinking != nil,
        )
    }
    return nil
}
```

---

## 9. Recommended Updates to P44 Design

### 9.1 Should the Category System Be DeepSeek-Native?

**No.** The category system should be **provider-agnostic**. Categories express task intent (`deep-reasoning`, `implementation`, `review`, `audit`, `quick`), not provider preferences. The mapping from category to provider+model is configuration, not architecture.

However, the P44 design's current category table lists Anthropic-native model names (Claude Opus, Sonnet, Haiku). This creates a subtle coupling. **Recommendation:** Rewrite the category table to use category-intent descriptions, not specific model names, and move the provider mapping to a separate configuration section.

### 9.2 Updated Category Configuration (Provider-Agnostic)

| Category | Cognitive profile | Thinking | Temperature | Orchestration |
|---|---|---|---|---|
| `deep-reasoning` | Novel reasoning, multi-step planning, architecture | enabled (max effort) | N/A (disabled by thinking) | Single agent |
| `implementation` | Pattern-matching, code generation, tool use | disabled | 0.3 | Orchestrator-workers |
| `quick` | Simple fixes, typos, documentation | disabled | 0.3 | Single agent |
| `review` | Evaluative, finding classification, determinism needed | disabled | 0.1 | Maker-checker |
| `audit` | Compliance, validation, repeatability critical | disabled | 0.0 | Single agent |

Provider mapping is a separate configuration layer:

```yaml
# .kbz/routing.yaml (or internal/routing/config.go defaults)
providers:
  - name: deepseek
    priority: 1
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

This decoupling means we can change provider priorities or add new providers without touching the category definitions.

### 9.3 Specific P44 Design Changes Recommended

| Section | Change | Rationale |
|---|---|---|
| **Implementation Phasing** | Move DeepSeek from Phase 3 to Phase 1 | DeepSeek V4 capabilities match/exceed Phase 1/2 providers. Dual API format minimizes integration work. Automatic caching reduces cost pressure during validation. |
| **Provider Integration Surface table** | Update DeepSeek row: thinking control is "now available" (was "not yet available"), add DeepSeek-V4-Pro and DeepSeek-V4-Flash as available models, note context caching and strict mode tool calling | The old table reflected DeepSeek-V3 capabilities. V4 changes the assessment. |
| **Category System table** | Rewrite to be provider-agnostic. Move specific model names to a provider mapping configuration section. Add thinking/temperature columns. | Decouples task intent from provider selection. Enables validation of incompatible combinations (thinking + low temperature). |
| **What Becomes Possible table** | Add "context caching" as a feature unlocked by model routing. Add "strict mode tool calling" as a quality improvement. | These are DeepSeek-specific capabilities that model routing makes available. |
| **Compaction section** | Add note about `reasoning_content` not being carried forward across compaction checkpoints. Clarify that compaction is a hard conversation reset. | Prevents a subtle bug where stale `reasoning_content` from a compacted conversation causes 400 errors. |
| **Open Questions** | Add: "Should `review` and `audit` categories support thinking mode if temperature is required?" Answer: No — they are irreconcilable. Categories that need determinism must disable thinking. | This is a design constraint discovered during DeepSeek analysis that applies to all providers. |
| **Architecture Options** | Add note that Option C is specifically validated by DeepSeek's dual-format API — a single `Provider` interface works across all three providers, supporting the "extract later" path. | Strengthens the Option C decision with concrete evidence. |

### 9.4 New Design Elements to Add

1. **Provider health check endpoint:** The `Health()` method on the `Provider` interface should be called periodically (every 60s) to detect provider outages. When a provider is unhealthy, the router skips it in fallback chains.

2. **Cache metrics dashboard:** Track per-feature cache hit rates. If hit rates drop below threshold, surface as a health warning (prompt structure may have become unstable).

3. **Thinking/temperature conflict validation:** The router must validate that categories specifying both thinking mode AND a temperature value are rejected at configuration load time, not at dispatch time.

---

## 10. Quick Reference Appendix

### 10.1 Model Selection Decision Tree

```
Task received by dispatch loop
│
├─ Category = "deep-reasoning"?
│  └─ YES → deepseek-v4-pro, thinking: enabled, reasoning_effort: "max"
│
├─ Category = "implementation"?
│  ├─ Multi-file refactor or >50K tokens?
│  │  └─ YES → deepseek-v4-pro, thinking: enabled, reasoning_effort: "high"
│  └─ NO → deepseek-v4-flash, thinking: disabled, temperature: 0.3
│
├─ Category = "quick"?
│  └─ YES → deepseek-v4-flash, thinking: disabled, temperature: 0.3
│
├─ Category = "review"?
│  └─ YES → deepseek-v4-pro, thinking: disabled, temperature: 0.1
│
└─ Category = "audit"?
   └─ YES → deepseek-v4-pro, thinking: disabled, temperature: 0.0

Fallback path (any category):
  If primary provider fails (429 × 3 retries, 503, timeout),
  try next provider in priority order:
  1. deepseek → 2. anthropic → 3. openai
```

### 10.2 API Parameter Quick-Reference

| Parameter | Type | Required | Values/Notes |
|---|---|---|---|
| `model` | string | Yes | `"deepseek-v4-pro"` or `"deepseek-v4-flash"` |
| `messages` | array | Yes | Standard OpenAI format: `[{role, content}, ...]` |
| `thinking.type` | string | No | `"enabled"` or `"disabled"`. Default: `"enabled"` |
| `reasoning_effort` | string | No | `"high"` or `"max"`. Only meaningful when thinking enabled. Default: `"high"` (auto `"max"` for agent requests) |
| `temperature` | float | No | 0.0–2.0. **Silently ignored** when thinking enabled |
| `top_p` | float | No | 0.0–1.0. **Silently ignored** when thinking enabled |
| `max_tokens` | int | No | Max output tokens. Up to 384K |
| `tools` | array | No | OpenAI function-calling format |
| `tool_choice` | string/object | No | `"auto"`, `"none"`, or specific function |
| `response_format` | object | No | `{type: "json_object"}` for guaranteed JSON output |
| `user` | string | No | Cache isolation key. Use `kanbanzai:{feature_id}:{batch_id}` |
| `stream` | bool | No | SSE streaming. Use for long-running reasoning tasks |

### 10.3 Error Code Handling Table

| HTTP Status | Meaning | Retry? | Fallback? | Action |
|---|---|---|---|---|
| 200 | Success | — | — | Process response normally |
| 400 | Invalid format | No | No | Log error, check request structure. Fix and re-dispatch. |
| 401 | Authentication failed | No | No | Check API key in `.kbz/local.yaml`. Alert human. |
| 402 | Insufficient balance | No | Yes (different provider) | Log warning, fall back to next provider. Alert human to top up. |
| 422 | Invalid parameters | No | No | Log error, check parameter combinations (thinking + temperature conflict?). Fix and re-dispatch. |
| 429 | Rate limited | Yes (3×, exponential backoff) | Yes (after retries exhausted) | Backoff 1s→2s→4s→8s with jitter. After 3 failures, fall back to next provider. |
| 500 | Server error | Yes (2×) | Yes (after retries exhausted) | Retry twice with 2s delay. Then fall back. |
| 503 | Server overloaded | Yes (2×) | Yes (after retries exhausted) | Same as 500. |

### 10.4 Pricing Comparison Table

Pricing per 1M tokens (USD). Anthropic and OpenAI pricing are approximate market rates as of mid-2026.

| Provider | Model | Input (cache miss) | Input (cache hit) | Output | Notes |
|---|---|---|---|---|---|
| **DeepSeek** | `deepseek-v4-flash` | $0.14 | $0.0028 | $0.28 | 1/50th input cost on cache hit |
| **DeepSeek** | `deepseek-v4-pro` | $1.74 ($0.435\*) | $0.0145 ($0.003625\*) | $3.48 ($0.87\*) | \*75% discount until 2026-05-31. 1/120th input cost on cache hit |
| **Anthropic** | Claude Haiku 3.5 | $0.80 | $0.08 | $4.00 | 1/10th input cost on cache hit |
| **Anthropic** | Claude Sonnet 4 | $3.00 | $0.30 | $15.00 | 1/10th input cost on cache hit |
| **Anthropic** | Claude Opus 4 | $15.00 | $1.50 | $75.00 | 1/10th input cost on cache hit |
| **OpenAI** | GPT-5.4-mini | ~$0.60 | ~$0.30 | ~$2.40 | 50% cache discount (estimated) |
| **OpenAI** | GPT-5.4 | ~$2.50 | ~$1.25 | ~$10.00 | 50% cache discount (estimated) |

**Key takeaways:**
- DeepSeek Flash is **~21× cheaper than Claude Sonnet** on input and **~54× cheaper** on output.
- Even DeepSeek Pro at full price ($1.74) is **~9× cheaper than Claude Opus** on input.
- DeepSeek's cache hit discount is **much steeper** (50:1–120:1) than Anthropic's (10:1). Structured prompts pay off faster.
- During the 75% discount window, DeepSeek Pro is **cheaper than Claude Haiku** ($0.435 vs. $0.80 input).

### 10.5 Feature Availability Summary

| Feature | `deepseek-v4-flash` | `deepseek-v4-pro` |
|---|---|---|
| Context window | 1M tokens | 1M tokens |
| Max output | 384K tokens | 384K tokens |
| Thinking mode | Yes | Yes |
| Reasoning effort control | Yes (high/max) | Yes (high/max) |
| Tool calling | Yes (OpenAI format) | Yes (OpenAI format) |
| Strict mode tool calling | Yes (Beta) | Yes (Beta) |
| JSON mode | Yes | Yes |
| Context caching | Yes (automatic) | Yes (automatic) |
| Anthropic Messages API | Yes (limited) | Yes (limited) |
| FIM completion | Yes (Beta, non-thinking) | Yes (Beta, non-thinking) |
| Logprobs | Yes | Yes |
| Streaming | Yes | Yes |
| Temperature | Works (thinking off only) | Works (thinking off only) |

---

## Document Metadata

**Author:** AI Research Agent (orchestrator role)  
**Reviewer:** Senior Software Architect (human)  
**Classification:** Public — engineering team reference  
**Supersedes:** None (new document)  
**Next steps:**
1. Review and approve this report.
2. Update P44 design document per §9 recommendations.
3. File a decision record for the DeepSeek Phase 1 inclusion.
4. Begin `internal/routing/providers/deepseek/` implementation based on §6 adapter design.
