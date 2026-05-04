# Prompt: DeepSeek V4 API Analysis for P41 Model Routing Features

**Target audience:** Senior Software Architect  
**Output:** A report suitable for the software engineering team  
**Purpose:** Analyse the DeepSeek V4 API documentation against P41/P44 requirements and produce recommendations

---

## Context

Kanbanzai is planning Model Routing & Agent Launcher capabilities (P41 Sub-plan C / P44). The first version of these features will primarily use the DeepSeek V4 models:

- `deepseek-v4-pro` — full-capability model with thinking mode support
- `deepseek-v4-flash` — faster/cheaper model; thinking mode default-on (replaces the deprecated `deepseek-chat` and `deepseek-reasoner`)

The DeepSeek API is OpenAI-compatible (Chat Completions) and Anthropic-compatible (Messages API). Both protocol surfaces are available.

## Reference Material

### Our Design Documents

1. **P41 Plan (OpenCode Ecosystem Features):** `/Users/samphillips/Dev/kanbanzai/work/P41-opencode-ecosystem-features/P41-design-opencode-ecosystem-features.md`
   - Model routing is Sub-plan C. Deferred until Sub-plans A and B are stable.
   - Unlocks: auto-compaction, thinking-level control, true Ralph Loop.
   - Architecture decision: Option C — embedded `internal/routing/` package in `kbz serve`, with extraction path to separate server.

2. **P44 Design (Model Routing & Agent Launcher Feasibility):** `/Users/samphillips/Dev/kanbanzai/work/P44-model-routing-agent-launcher/P44-design-model-routing-agent-launcher.md`
   - **Architecture:** Option C (build together, extract later). `internal/routing/` package within `kbz serve` with clean boundaries.
   - **Category System (current design):**
     | Category | Preferred model | Fallback |
     |----------|----------------|----------|
     | `deep-reasoning` | Claude Opus | GPT-5.4 → DeepSeek-R1 |
     | `implementation` | Claude Sonnet | GPT-5.4 → DeepSeek-V3 |
     | `quick` | Claude Haiku | GPT-5.4-mini |
     | `review` | GPT-5.4 (low temp) | Claude Opus |
     | `audit` | GPT-5.4 (near-zero) | Claude Opus |
   - **Category → orchestration pattern mapping** exists (single agent vs. orchestrator-workers vs. maker-checker).
   - **Phasing:** Phase 1: Anthropic-only MVP. Phase 2: +OpenAI. Phase 3: +DeepSeek.
   - **Compaction:** U-shaped continuation prompt with `trimmed_context` section.
   - **Token budget:** Agent-facing communication at dispatch time.
   - **Provider Integration Surface:** Anthropic Messages API, OpenAI Chat Completions, DeepSeek Chat Completions.

### DeepSeek API Documentation (fully reviewed)

All documentation has been read. Here is a structured summary of everything relevant:

#### 1. API Surface
- **Base URL (OpenAI format):** `https://api.deepseek.com`
- **Base URL (Anthropic format):** `https://api.deepseek.com/anthropic`
- **Beta features URL:** `https://api.deepseek.com/beta`
- **Endpoint:** `POST /chat/completions`
- **Auth:** Bearer token (`Authorization: Bearer ${DEEPSEEK_API_KEY}`)
- OpenAPI spec available at `/api/deepseek-api`

#### 2. Models
| Model | Context | Max Output | Thinking Mode | Notes |
|-------|---------|------------|---------------|-------|
| `deepseek-v4-pro` | 1M tokens | 384K tokens | Supported (default: enabled) | Full capability |
| `deepseek-v4-flash` | 1M tokens | 384K tokens | Supported (default: enabled) | Faster/cheaper |
| `deepseek-chat` | — | — | — | Deprecated 2026-07-24; maps to `deepseek-v4-flash` non-thinking |
| `deepseek-reasoner` | — | — | — | Deprecated 2026-07-24; maps to `deepseek-v4-flash` thinking |

#### 3. Pricing (per 1M tokens)
| Model | Input (cache miss) | Input (cache hit) | Output |
|-------|-------------------|-------------------|--------|
| `deepseek-v4-flash` | $0.14 | $0.0028 | $0.28 |
| `deepseek-v4-pro` | $1.74 ($0.435 with current 75% discount until 2026-05-31) | $0.0145 ($0.003625 discounted) | $3.48 ($0.87 discounted) |

Cache hit is 1/50th the cost of cache miss for flash, and 1/120th for pro.

#### 4. Thinking Mode (`/guides/thinking_mode`)
- **Toggle:** `{"thinking": {"type": "enabled/disabled"}}` (OpenAI format). In Anthropic format, use `thinking` param (but `budget_tokens` is ignored).
- **Effort control:** `reasoning_effort: "high"` or `"max"` (OpenAI); `output_config: {effort: "high/max"}` (Anthropic). Default is `high` for regular requests; auto-set to `max` for complex agent requests (Claude Code, OpenCode).
- **Parameters disabled in thinking mode:** `temperature`, `top_p`, `presence_penalty`, `frequency_penalty` — set without error but have no effect.
- **Reasoning content:** Returned via `reasoning_content` field at same level as `content`.
- **Multi-turn without tool calls:** `reasoning_content` from previous turns is ignored by the API — do not need to pass back.
- **Multi-turn with tool calls:** `reasoning_content` MUST be passed back to the API in all subsequent requests. If missing, API returns 400 error. Simplest pattern: append entire `response.choices[0].message` object.

#### 5. Context Caching (`/guides/kv_cache`)
- **Enabled by default** for all users. No code changes needed.
- **How it works:** Disk-based cache. Each request creates cache prefix units. Subsequent requests with matching prefixes get cache hits.
- **Cache persistence patterns:**
  1. At request boundaries (end of user input, end of model output).
  2. Common prefix detection across multiple requests.
  3. At fixed token intervals for long inputs/outputs.
- **Cache hit requirement:** Full match of a cache prefix unit. Partial prefix match does NOT hit.
- **Practical example:** Multi-turn conversations — second request fully reuses the first request's prefix (system + user messages) → cache hit.
- **Practical example for Kanbanzai:** Repeated system prompts + knowledge context + entity state would cache well across multiple task dispatches within the same feature.
- **Usage response fields:** `prompt_cache_hit_tokens`, `prompt_cache_miss_tokens` in `usage`.
- **Best-effort** — not guaranteed 100% hit rate. Cache construction takes seconds. Auto-cleared after hours to days of inactivity.
- **Cache isolation** at `user_id` level. Different `user_id` values get separate cache namespaces.

#### 6. Tool Calls (`/guides/tool_calls`)
- Full OpenAI-compatible function calling.
- **`strict` mode (Beta):** Forces model to adhere strictly to JSON Schema. Requires `base_url="https://api.deepseek.com/beta"` and `strict: true` on each function definition.
- **Strict mode supported JSON Schema types:** `object`, `string`, `number`, `integer`, `boolean`, `array`, `enum`, `anyOf`, `$ref`/`$def`.
- **Strict mode constraints:** All object properties must be `required`, `additionalProperties: false` required on all objects.
- Thinking mode + tool calls supported since DeepSeek-V3.2.

#### 7. Anthropic API Compatibility (`/guides/anthropic_api`)
- Full Messages API support with some limitations:
  - **Supported:** `model`, `max_tokens`, `stop_sequences`, `stream`, `system`, `temperature` (0.0–2.0), `tools` (name, input_schema, description), `tool_choice`, message content (text, tool_use, tool_result, thinking).
  - **Ignored:** `cache_control`, `citations`, `top_k`, `metadata`, `mcp_servers`, `container`, `service_tier`, `disable_parallel_tool_use`, `is_error`.
  - **Not supported:** Image content, document content, search_result, redacted_thinking, server_tool_use, MCP tool types, container_upload.
  - **Thinking:** `thinking` param accepted but `budget_tokens` ignored. Use `output_config.effort` instead.
  - **Auto model mapping:** Unsupported model names auto-map to `deepseek-v4-flash`.

#### 8. Rate Limits & Error Codes (`/quick_start/rate_limit`, `/quick_start/error_codes`)
- **Dynamic concurrency limiting** based on server load. HTTP 429 when exceeded.
- **Keep-alive:** Non-streaming returns empty lines; streaming returns SSE `: keep-alive` comments.
- **Timeout:** 10 minutes before inference starts → connection closed.
- **Error codes:** 400 (invalid format), 401 (auth), 402 (insufficient balance), 422 (invalid params), 429 (rate limit), 500 (server error), 503 (overloaded).

#### 9. Token Usage (`/quick_start/token_usage`)
- ~1 English char ≈ 0.3 token; ~1 Chinese char ≈ 0.6 token.
- Tokenizer available for offline calculation: `deepseek_tokenizer.zip`.
- Actual token counts returned in `usage` response field.

#### 10. Other Features
- **JSON Output:** `response_format: {type: "json_object"}` — guarantees valid JSON.
- **Chat Prefix Completion (Beta):** Forces model to continue from a given assistant prefix.
- **FIM Completion (Beta):** Fill-in-the-middle for code. Non-thinking mode only.
- **Logprobs:** Supported (`logprobs: true`, `top_logprobs` up to 20).

---

## Your Task

Produce a comprehensive how-to guide and analysis report that the engineering team can use as their primary reference for implementing the DeepSeek integration in P41/P44 Model Routing features. The report should eliminate the need for developers to repeatedly consult the DeepSeek website.

### Required Sections

#### 1. Executive Summary
- High-level assessment of DeepSeek V4 suitability for Kanbanzai's model routing needs.
- Key strengths and gaps vs. the P44 design requirements.

#### 2. Gap Analysis: P44 Design vs. DeepSeek API Reality
- The current P44 design defers DeepSeek to Phase 3 and positions it as a fallback for Anthropic/OpenAI categories.
- The DeepSeek V4 models have thinking mode, 1M context, tool calling, and context caching — capabilities that rival or exceed the Phase 1/2 provider choices.
- **Analyse:** Should DeepSeek be elevated from Phase 3 to Phase 1 or Phase 2? Why or why not?
- **Analyse:** For each P44 category (`deep-reasoning`, `implementation`, `review`, `audit`, `quick`), which DeepSeek model (pro vs. flash) and configuration (thinking on/off, effort level) is appropriate?
- **Recommend:** Updated category → provider/model mapping that includes DeepSeek as a first-class option, not just a fallback.

#### 3. Feature-by-Feature DeepSeek Mapping
For each P44 capability that model routing unlocks, describe exactly how DeepSeek supports it:

| P44 Feature | DeepSeek Support | Implementation Notes |
|-------------|-----------------|---------------------|
| Thinking-level control | `thinking.type` + `reasoning_effort` | Map P44 categories to effort levels |
| Auto-compaction | `usage.prompt_tokens` + context caching | Cache hit tokens count; use for utilization calculation |
| True Ralph Loop | Context caching + thinking mode multi-turn | Cache system prompts; handle reasoning_content correctly across tool-call turns |
| Provider fallback | Dynamic concurrency + error codes 429/503 | Implement retry with backoff |
| Cost tracking | `usage` response with cache hit/miss breakdown | Per-request; aggregate per-feature/batch |
| Token budget communication | `usage.prompt_tokens` + `max_tokens` | Compute utilization; communicate to agent |

#### 4. Thinking Mode Integration Guide
- How to toggle thinking mode per task category via the API.
- When to use `reasoning_effort: "high"` vs. `"max"` in Kanbanzai's context.
- The critical `reasoning_content` management rules (tool-call vs. non-tool-call turns).
- Code examples showing correct message concatenation patterns for Kanbanzai's orchestration loop (which involves tool calls).
- Interaction with `temperature`/`top_p` (disabled in thinking mode — what this means for `review` and `audit` categories that rely on low temperature).

#### 5. Context Caching Strategy for Kanbanzai
- How to maximize cache hits in Kanbanzai's orchestration pattern (repeated system prompts, entity state, knowledge context).
- Design pattern: structure prompts with stable prefixes (system prompt + entity context) followed by variable suffixes (task-specific instructions).
- The `user_id` parameter for cache isolation — how to use it for per-feature or per-batch cache namespaces.
- Cache hit monitoring via `prompt_cache_hit_tokens` / `prompt_cache_miss_tokens`.
- Best-effort caveats and what to design for when cache misses occur.

#### 6. Provider Adapter Design for `internal/routing/providers/`
- Given the DeepSeek API supports both OpenAI and Anthropic protocol formats, recommend which format to use.
- Trade-offs: OpenAI SDK (broader ecosystem) vs. Anthropic SDK (already planned for Phase 1).
- Common adapter interface that works across Anthropic, OpenAI, and DeepSeek with minimal per-provider divergence.
- How `strict` mode tool calling (Beta) could benefit Kanbanzai's MCP tool definitions.

#### 7. Cost Optimization Recommendations
- Flash vs. Pro selection criteria based on task category.
- Cache hit economics: projected savings from structured prompt prefixes.
- The 75% Pro discount (until 2026-05-31) and implications for Phase 1 vs. Phase 3 sequencing.

#### 8. Risks and Mitigations
- Rate limiting (dynamic, non-deterministic) — how to handle in a dispatch loop.
- 10-minute inference start timeout — implications for complex reasoning tasks.
- Cache best-effort nature — don't build logic that assumes cache hits.
- Model deprecation timeline (`deepseek-chat`/`deepseek-reasoner` → 2026-07-24).
- Thinking mode disables temperature — impact on `review`/`audit` categories that need determinism.

#### 9. Recommended Updates to P44 Design
- Concrete recommendations for changes to the P44 design document.
- Should the category system be DeepSeek-native rather than Anthropic-native?
- Should DeepSeek move from Phase 3 to Phase 1 or Phase 2?
- New categories or category refinements based on flash vs. pro capabilities.

#### 10. Quick Reference Appendix
- Model selection decision tree (flowchart in text form).
- API parameter quick-reference table.
- Error code handling table.
- Pricing comparison table (Anthropic vs. OpenAI vs. DeepSeek — research current pricing for the comparison).

### Tone and Format

- Write for a software engineering team. Be precise, concrete, and actionable.
- Include code snippets where they illustrate API usage patterns.
- Use tables for comparisons and mappings.
- Call out design implications explicitly — don't just describe the API, explain what it means for our architecture.
- Where the current P44 design has gaps, recommend specific changes.
- Where DeepSeek has limitations, be honest about them and propose mitigations.

### Important Constraints

- The P44 design currently sequences DeepSeek in Phase 3 (after Anthropic-only MVP and OpenAI Phase 2). Challenge this if the evidence supports it.
- The architecture is Option C: embedded `internal/routing/` package with clean boundaries for potential extraction.
- Kanbanzai's orchestration involves tool calls (it's an MCP server). The thinking mode tool-call `reasoning_content` management rules are critical.
- Cost matters — this is an open-source project. Optimize for cost-effectiveness without sacrificing capability.

---

## Output

Produce the final report as a markdown document. Store it at:

`/Users/samphillips/Dev/kanbanzai/work/P41-opencode-ecosystem-features/P41-report-deepseek-api-analysis.md`

The report should be self-contained — a developer should be able to read it without consulting the DeepSeek website or the P41/P44 design documents (though it should reference them).
