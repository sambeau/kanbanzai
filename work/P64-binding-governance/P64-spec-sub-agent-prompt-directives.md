# Specification: Sub-agent prompt dispatch and worktree-write directives

## Overview

Handoff-assembled prompts currently contain no directives about the dispatch contract or worktree-write requirements. This causes orchestrators to implement directly instead of dispatching, and sub-agents to write files outside worktrees (to the repo root).

## Scope

Single-file change to `internal/context/pipeline.go` in the `RenderPrompt` function.

## Functional Requirements

- [ ] FR-1: When the binding's orchestration pattern is `orchestrator-workers`, the assembled handoff prompt SHALL include a directive block stating that orchestrators dispatch via `spawn_agent` and sub-agents execute (orchestrators must not implement directly).
- [ ] FR-2: When the entity has an active worktree, the assembled handoff prompt SHALL include a directive block instructing use of `write_file`/`kanbanzai_edit_file` with `entity_id` parameter rather than plain `write_file` or shell redirection.
- [ ] FR-3: Directive blocks SHALL be positioned after role/skill context but before task-specific content.

## Non-Functional Requirements

- [ ] NFR-1: Directive blocks SHALL NOT appear when the orchestration pattern is not `orchestrator-workers` (non-orchestrator sub-agents don't need dispatch directives).
- [ ] NFR-2: The worktree directive SHALL NOT appear when the entity has no active worktree.

## Acceptance Criteria

- [ ] AC-1: Handoff output for `orchestrator-workers` stages includes the dispatch contract directive.
- [ ] AC-2: Handoff output for entities with active worktrees includes the worktree-write directive.
- [ ] AC-3: Handoff output for non-orchestrator stages does NOT include the dispatch contract directive.
- [ ] AC-4: Existing tests continue to pass.
