# Design: Sub-agent prompt dispatch and worktree-write directives

## Overview

Add imperative directive blocks to handoff prompt assembly in `internal/context/pipeline.go` `RenderPrompt`.

## Design

Two directive blocks, conditionally included:

1. **Dispatch contract** (when `binding.Orchestration == orchestrator-workers`): States that orchestrators dispatch via `spawn_agent`, sub-agents execute — orchestrators must not implement directly.

2. **Worktree-write directive** (when entity has active worktree): Instructs sub-agents to use `write_file`/`kanbanzai_edit_file` with the `entity_id` parameter rather than plain `write_file` or shell redirection.

These directives are inserted into the handoff prompt output near the top, after the role/skill context but before task-specific content.

## Alternatives Considered

- **Agent instructions file**: Embed directives in a static instructions file. Rejected — directives are stage/context-dependent.
- **MCP tool metadata**: Expose via tool descriptions. Rejected — LLMs don't reliably read tool descriptions for workflow rules.

## Dependencies

None. Self-contained addition to the prompt assembly pipeline.
