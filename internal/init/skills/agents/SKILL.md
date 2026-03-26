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

Before starting work on any task, call `context_assemble`:

    context_assemble(role="<role>", task_id="<task_id>")

The returned packet is byte-budgeted and prioritised — it contains the task instructions,
relevant knowledge entries, and design context trimmed to fit the budget. Read it before
doing anything else.

After completing the task, call `context_report` to record which knowledge entries were
used or found incorrect. This drives auto-confirmation and auto-retirement:

    context_report(
      task_id="<task_id>",
      used=["KE-..."],
      flagged=[{"entry_id": "KE-...", "reason": "..."}]
    )

## Task Dispatch and Completion

1. Call `work_queue` to see ready tasks.
2. Call `dispatch_task` to atomically claim a task (transitions it from `ready` to `active`):

       dispatch_task(task_id="TASK-...", role="<role>", dispatched_by="agent/<name>")

3. Do the work.
4. Call `complete_task` when done:

       complete_task(
         task_id="TASK-...",
         summary="What was done",
         files_modified=["path/to/file.go"],
         verification_performed="What was tested or verified",
         knowledge_entries=[...]
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

    knowledge_contribute(
      topic="short-topic-slug",
      content="Concise, actionable statement",
      scope="<role or project>",
      tier=3
    )

Tier 3 is session-level (30-day TTL). The system auto-promotes entries to tier 2 based on
usage across sessions. You can also contribute via the `knowledge_entries` argument in
`complete_task`.

## Sub-Agent Spawning

When spawning sub-agents, include in every delegation message:

1. **Project name** — the codebase-memory-mcp project name for graph queries
2. **File scope boundaries** — which files the sub-agent should and should not modify
3. **Project conventions** — commit format, test conventions, language style rules
4. **Context propagation instruction** — if the sub-agent may itself spawn agents, tell
   it to include the same context in its delegations

`context_assemble` is the primary skill delivery mechanism for MCP-connected sub-agents.
Prefer it over manually reciting conventions.

---

## Gotchas

**`dispatch_task` fails:** Another agent already claimed the task. Call `work_queue` again
to get an updated list of ready tasks.

**Tool call failures:** Read the error message before retrying. Most failures are
deterministic — retrying with the same arguments will produce the same error.

**Missing verification summary:** `complete_task` will succeed without `verification_performed`,
but it degrades review quality and reduces the value of the knowledge base. Always include it.

---

## Related

- `kanbanzai-getting-started` — session orientation and work queue
- `kanbanzai-workflow` — stage gates, human/agent boundary, when to stop and ask
- `kanbanzai-documents` — document registration and approval