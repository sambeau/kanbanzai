---
name: kanbanzai-agents
description: >
  Use this skill when interacting with the Kanbanzai system as an agent: before starting
  implementation work, when assembling context, dispatching or completing tasks, writing
  commits, contributing knowledge entries, or spawning sub-agents. Activate for any task,
  not just complex ones.
# kanbanzai-managed: true
# kanbanzai-version: dev
---

## Purpose

This skill defines how agents interact with Kanbanzai: assembling context, dispatching
and completing tasks, writing commits, contributing knowledge, and spawning sub-agents.

## Context Assembly

Before starting work on any task, call `next` with the task ID to claim it and receive a
context packet:

    next(id="TASK-...")

The returned packet is byte-budgeted and prioritised — it contains the task instructions,
relevant knowledge entries, and design context trimmed to fit the budget. Read it before
doing anything else. Claiming the task transitions it from `ready` to `active`.

After completing the task, use `knowledge` to record which entries were useful or
incorrect. This drives auto-confirmation and auto-retirement:

    knowledge(action="confirm", id="KE-...")
    knowledge(action="flag", id="KE-...", reason="...")

## Task Dispatch and Completion

1. Call `next` (without an ID) to see ready tasks.
2. Call `next` with a task ID to claim it — transitions it from `ready` to `active` and
   returns the assembled context packet:

       next(id="TASK-...")

3. Do the work.
4. Call `finish` when done:

       finish(
         task_id="TASK-...",
         summary="What was done",
         files_modified=["path/to/file.go"],
         verification="What was tested or verified",
         knowledge=[...]
       )

## Commit Message Format

Every commit must follow this format: `<type>(<object-id>): <summary>`

| Type | When to use |
|---|---|
| `feat` | New feature behaviour |
| `fix` | Bug fix |
| `docs` | Documentation change only |
| `test` | Test-only change |
| `refactor` | Behaviour-preserving structural improvement |
| `workflow` | Workflow-state-only change (no code) |
| `decision` | Decision-record change |
| `chore` | Small maintenance change with no better category |

Add `!` after the type for breaking changes: `feat(FEAT-001)!: description`

Examples:
- `feat(FEAT-152): add profile editing API and validation`
- `fix(BUG-027): prevent avatar uploads hanging on large files`
- `workflow(TASK-152.3): mark upload task complete after verification`

## Knowledge Contribution

When you discover something reusable, contribute it:

    knowledge(
      action="contribute",
      topic="short-topic-slug",
      content="Concise, actionable statement",
      scope="<role or project>",
      tier=3
    )

Tier 3 is session-level (30-day TTL). The system auto-promotes entries to tier 2 based on
usage across sessions. You can also contribute via the `knowledge` argument in `finish`.

## Sub-Agent Spawning

When spawning sub-agents, include in every delegation message:

1. **Project name** — the codebase-memory-mcp project name for graph queries
2. **File scope boundaries** — which files the sub-agent should and should not modify
3. **Project conventions** — commit format, test conventions, language style rules
4. **Context propagation instruction** — if the sub-agent may itself spawn agents, tell
   it to include the same context in its delegations

`next` and `handoff` are the primary skill delivery mechanisms for MCP-connected
sub-agents. Use `handoff` with a task ID to generate a structured sub-agent prompt.
Prefer it over manually reciting conventions.

---

## Gotchas

**`next` fails when claiming a task:** Another agent already claimed the task. Call `next`
again (without an ID) to get an updated list of ready tasks.

**Tool call failures:** Read the error message before retrying. Most failures are
deterministic — retrying with the same arguments will produce the same error.

**Missing verification summary:** `finish` will succeed without `verification`, but it
degrades review quality and reduces the value of the knowledge base. Always include it.

---

## Related

- `kanbanzai-getting-started` — session orientation and work queue
- `kanbanzai-workflow` — stage gates, human/agent boundary, when to stop and ask
- `kanbanzai-documents` — document registration and approval