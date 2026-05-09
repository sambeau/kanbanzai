# Delegating to Sub-Agents

Full sub-agent delegation guide. Referenced from [AGENTS.md](../AGENTS.md).

When you spawn sub-agents (via `spawn_agent`), those agents do **not** see `AGENTS.md`. They
only know what you tell them. Critical project context — tool preferences, conventions, the
knowledge graph — is lost unless you explicitly propagate it.

## Required context for every sub-agent

Include the following in every `spawn_agent` message:

1. **Codebase knowledge graph availability:**

   > This project is indexed in `codebase-memory-mcp` as project `Users-samphillips-Dev-kanbanzai`.
   > Prefer graph tools over grep/find for structural code questions:
   > - `search_graph(name_pattern="...", project="Users-samphillips-Dev-kanbanzai")` — find functions, types, classes
   > - `get_code_snippet(qualified_name="...", project="Users-samphillips-Dev-kanbanzai")` — read a specific symbol
   > - `trace_call_path(function_name="...", project="Users-samphillips-Dev-kanbanzai")` — find callers/callees
   > - `get_architecture(project="Users-samphillips-Dev-kanbanzai")` — package structure overview
   >
   > Use `grep` only for string literals, error messages, and non-structural content.

2. **File scope boundaries** — which files the agent should and should not modify (to avoid
   conflicts with parallel agents).

3. **Relevant project conventions** — commit message format, test conventions, Go style rules —
   if the agent will be committing or writing tests. Link to [`refs/go-style.md`](go-style.md)
   and [`refs/testing.md`](testing.md) rather than repeating the rules inline.

## Propagation rule

If a sub-agent may itself spawn further sub-agents, include this instruction in your message:

> When you delegate work to sub-agents, include the codebase-memory-mcp context (project name
> `Users-samphillips-Dev-kanbanzai`, tool preferences) in your delegation message. Sub-agents
> do not see project instructions automatically.

This ensures the context propagates through any depth of delegation, not just one level.

## Why this matters

Without this context, sub-agents default to `grep` and `read_file` for everything — scanning
files line by line instead of using the indexed graph. This is slower, noisier, and misses
structural relationships that the graph captures directly.

---

## Host-specific loading behaviour

Different platforms that host or proxy AI models vary in whether they automatically inject
project instruction files (`AGENTS.md`, `OPENAI.md`) and MCP tool descriptions into the
model context. The sections below document what each platform does — and what a Kanbanzai
operator must configure manually.

### DeepSeek

**Platform:** DeepSeek API (`api.deepseek.com`) — an OpenAI-compatible and
Anthropic-compatible inference API provided by DeepSeek, Inc. DeepSeek is a **model
provider**, not a coding-agent platform. It does not supply an agentic runtime analogous to
Claude Code or GitHub Copilot. Kanbanzai operators use DeepSeek by configuring a compatible
MCP client or agent framework (e.g., OpenCode) to call the DeepSeek API as its backend model.

| Field | Behaviour |
|-------|-----------|
| **Host / platform** | DeepSeek API (`api.deepseek.com`), OpenAI/Anthropic-compatible inference. Also available at `api.deepseek.com/anthropic` for Anthropic SDK callers. |
| **`AGENTS.md` injection** | **No.** The DeepSeek API is a stateless inference API with no filesystem awareness. `AGENTS.md` is never injected automatically. The system prompt is fully the caller's responsibility. |
| **Tool-description injection** | **No** (raw API). DeepSeek supports OpenAI-compatible function calling (`tools` parameter) but has no native MCP support. Tool schemas must be passed explicitly in each request. Whether the MCP client *wrapping* DeepSeek injects tool descriptions automatically is `Unknown — see REQ-010` (behaviour is client-specific, not a DeepSeek API property). |
| **Manual configuration required** | (1) Include `AGENTS.md` content in the system prompt on each invocation. (2) Provide Kanbanzai MCP tool schemas via an MCP-to-OpenAI bridge (e.g., the OpenCode built-in bridge, or a custom shim). (3) Set `base_url=https://api.deepseek.com`, API key, and model name (`deepseek-v4-pro` or `deepseek-v4-flash`) in the MCP client config. |

**Important distinction:** Because DeepSeek is a model provider rather than an agentic
runtime, most injection questions are really questions about the *client* that fronts DeepSeek
(e.g., OpenCode, a custom harness). If a client with native `AGENTS.md` or MCP support
targets DeepSeek as its backend, that client's injection behaviour applies — DeepSeek itself
adds nothing on top of the raw inference response.

**REQ-010 note:** Whether specific third-party clients that use DeepSeek as their backend
model automatically inject `AGENTS.md` or MCP tool descriptions is not verifiable from
DeepSeek's own API documentation and is therefore `Unknown — see REQ-010` for those
client-specific fields.
