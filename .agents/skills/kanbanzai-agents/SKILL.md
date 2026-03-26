---
name: kanbanzai-agents
description: >
  Use when implementing tasks, dispatching work, completing tasks, writing
  commits, contributing knowledge entries, assembling context, or spawning
  sub-agents. Also activates for questions about the agent interaction
  protocol, commit message format, knowledge management, context packets,
  or how to hand off work between agents.
metadata:
  kanbanzai-managed: "true"
  version: "0.1.0"
---

# SKILL: Kanbanzai Agents

## Purpose

Define how agents interact with the Kanbanzai system: assembling context,
dispatching and completing tasks, writing commits, contributing knowledge,
and spawning sub-agents.

## When to Use

- Before starting implementation work on any task
- When dispatching or completing tasks
- When writing commit messages
- When contributing knowledge entries after completing work
- When spawning sub-agents for parallel or delegated work

---

## Context Assembly

Before beginning any task, assemble context:

1. Call `context_assemble(role="<role>", task_id="<task_id>")` to get a
   context packet containing role instructions, relevant knowledge entries,
   design context, and task details.
2. Read the full context packet before starting. Do not skip it.
3. The packet is byte-budgeted and prioritised — it contains what matters
   most for this task and role.

After completing the task, call `context_report` to record which knowledge
entries were used and which were incorrect:

- **Used entries** — their use count increments; frequently-used entries are
  auto-confirmed.
- **Flagged entries** — their miss count increments; repeatedly-flagged
  entries are auto-retired.

This feedback loop keeps the knowledge base accurate over time.

---

## Task Dispatch and Completion

### Picking up work

1. Call `work_queue` to see ready tasks, sorted by priority.
2. Call `dispatch_task(task_id, role, dispatched_by)` to atomically claim a
   task. This moves it from `ready` to `active` and returns a context packet.
3. Do the work.

### Finishing work

Call `complete_task` with:

- `task_id` — the task being completed
- `summary` — brief description of what was accomplished
- `files_modified` — files created or changed
- `verification_performed` — what testing or verification was done
- `knowledge_entries` — any reusable knowledge learned during the task

Do not mark a task complete without providing a verification summary. The
summary does not need to be long, but it must exist.

---

## Commit Message Format

Every commit follows this format:

```
<type>(<object-id>): <summary>
```

### Types

| Type | When to use |
|------|-------------|
| `feat` | New feature behaviour |
| `fix` | Bug fix |
| `docs` | Documentation change |
| `test` | Test-only change |
| `refactor` | Behaviour-preserving structural improvement |
| `workflow` | Workflow-state-only change |
| `decision` | Decision-record change |
| `chore` | Small maintenance, no better category |

Add `!` after the type for breaking changes: `feat(FEAT-001)!: description`

### Examples

- `feat(FEAT-152): add profile editing API and validation`
- `fix(BUG-027): prevent avatar uploads hanging on large files`
- `docs(FEAT-152): update profile editing user documentation`
- `workflow(TASK-152.3): mark upload task complete after verification`
- `decision(DEC-041): record no-client-side-cropping choice`

### Commit discipline

- Commit at logical checkpoints — after completing a coherent change, before
  starting a risky edit.
- A change is not done until it is committed.
- Do not commit directly to `main`. Work on feature or bug branches.
- Document changes follow the same commit discipline as code changes.

---

## Knowledge Contribution

When you learn something useful during a task that is not already in the
knowledge base, contribute it:

```
knowledge_contribute(
  topic="<topic>",
  content="<concise actionable statement>",
  scope="<role or 'project'>",
  tier=3,
  learned_from="<task-id>"
)
```

- **Tier 3** — session-level knowledge; expires after 30 days without use.
- **Tier 2** — project-level knowledge; the system promotes tier-3 entries
  automatically when they are used repeatedly.

Good knowledge entries are concise, actionable, and specific. "The API
returns 404 for deleted users, not 410" is a good entry. "Be careful with
the API" is not.

---

## Sub-Agent Spawning

When delegating work to a sub-agent:

1. **Include context** — sub-agents do not see your conversation history or
   project skills automatically. Include the project name for any codebase
   knowledge graph, relevant file paths, and any conventions the sub-agent
   needs to follow.
2. **Scope boundaries** — tell each sub-agent which files or directories it
   owns. If spawning multiple agents in parallel, ensure they do not write to
   the same files.
3. **MCP delivery** — `context_assemble` automatically includes relevant
   skill content in the context packet for sub-agents running through MCP.
   This is the primary skill delivery mechanism for sub-agents.

---

## What Agents Must Not Do

- Commit directly to `main`
- Skip `context_assemble` before implementation work
- Complete tasks without providing a verification summary
- Ignore errors — never use `_` to discard an error return
- Make product decisions — priority, scope, and direction belong to the human
- Approve their own work

---

## Related

- `kanbanzai-getting-started` — session orientation (what to do first)
- `kanbanzai-workflow` — stage gates and when to stop and ask the human
- `kanbanzai-documents` — document registration and approval