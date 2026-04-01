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