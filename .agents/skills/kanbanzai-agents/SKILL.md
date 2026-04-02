---
name: kanbanzai-agents
description: >
  Use when implementing tasks, dispatching work, completing tasks, writing
  commits, contributing knowledge entries, assembling context, or spawning
  sub-agents. Activates for agent interaction protocol, commit message
  format, knowledge management, context packets, or hand-off between
  agents. Use this skill for every task — it defines how agents interact
  with the workflow system.
metadata:
  kanbanzai-managed: "true"
  version: "0.2.0"
---

# SKILL: Kanbanzai Agents

## Purpose

How agents interact with the Kanbanzai system: assembling context,
dispatching and completing tasks, writing commits, contributing knowledge,
and spawning sub-agents.

## When to Use

- Before starting implementation work on any task
- When dispatching or completing tasks
- When writing commit messages
- When contributing knowledge entries after completing work
- When spawning sub-agents for parallel or delegated work

---

## Task Lifecycle Checklist

Copy this checklist when starting any task:

- [ ] Claimed the task with `next(id: "TASK-xxx")` — context assembled
- [ ] Read the assembled context (spec sections, knowledge entries, file paths)
- [ ] Confirmed the parent feature is in the correct lifecycle state
- [ ] Implemented the changes
- [ ] Ran tests (`go test ./...`) and they pass
- [ ] Committed with a properly formatted message (see Commit Message Format below)
- [ ] Completed the task with `finish(task_id: "TASK-xxx", summary: "...", verification: "...")`
- [ ] Included retrospective observations if any friction was encountered

---

## Context Assembly

Before beginning any task, assemble context by claiming it:

1. Call `next` with a task ID to claim the task and receive a context packet
   containing role instructions, relevant knowledge entries, design context,
   and task details.
2. The packet is byte-budgeted and prioritised — it contains what matters
   most for this task and role.
3. To generate a prompt for a sub-agent instead, call `handoff` with the
   task ID — it produces a self-contained Markdown prompt ready for
   delegation.

After completing the task, maintain the knowledge base:

- Call `knowledge` action: `confirm` on entries that were accurate and
  useful — their use count increments; frequently-confirmed entries gain
  confidence.
- Call `knowledge` action: `flag` on entries that were incorrect or
  misleading — their miss count increments; repeatedly-flagged entries are
  auto-retired.

This feedback loop keeps the knowledge base accurate over time.

---

## Task Dispatch and Completion

### Picking up work

1. Call `next` (without an ID) to see ready tasks, sorted by priority.
2. Call `next` with a task ID to claim it. This moves it from `ready` to
   `active` and returns a context packet with role instructions, knowledge
   entries, and task details.
3. Do the work.

### Finishing work

Call `finish` with:

- `task_id` — the task being completed
- `summary` — brief description of what was accomplished
- `files_modified` — files created or changed
- `verification` — what testing or verification was done
- `knowledge` — any reusable knowledge learned during the task

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

### Bad vs. Good Commit Messages

**Bad:** `fix: update code`
- No scope, no description of what changed or why

**Good:** `fix(validate): reject task finish when parent feature is in draft status`
- Scoped to the package, describes the behavior change

**Bad:** `feat: add new feature`
- Meaningless — every feature commit "adds a new feature"

**Good:** `feat(mcp): add context budget estimation to handoff tool`
- Scoped, specific, describes the actual capability added

### Commit discipline

- Commit at logical checkpoints — after completing a coherent change, before
  starting a risky edit.
- Do not commit directly to `main`. Work on feature or bug branches.
- Document changes follow the same commit discipline as code changes.
- **`.kbz/state/` files are versioned project state, not ephemeral cache.**
  They record entity lifecycle, document metadata, and knowledge entries that
  other agents depend on. Treat `.kbz/state/` changes as code changes —
  commit them alongside the work that produced them. Never leave `.kbz/`
  files uncommitted at the end of a task.

---

## Knowledge Contribution

When you learn something useful during a task that is not already in the
knowledge base, contribute it:

```
knowledge(
  action="contribute",
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

## Communicating With Humans

Documents are the human interface to the system. Decision records and their
IDs are internal tracking mechanisms — important for system integrity and
useful for agents, but not how humans navigate the project.

When talking with humans:

- Reference **documents** by name: "the ID system design", "the Phase 1
  spec §10"
- Use **prose descriptions** of decisions: "the decision about cache-based
  locking"
- Do **not** lead with decision IDs: ~~"P1-DEC-021 defines the ID format"~~

Decision IDs don't carry enough context for a human to act on without
querying the system. A document name or a prose summary is immediately
meaningful. Save decision IDs for commit messages, entity cross-references,
and agent-to-agent communication.

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
3. **MCP delivery** — `next` and `handoff` automatically include relevant
   skill content in the context packet for sub-agents running through MCP.
   This is the primary skill delivery mechanism for sub-agents.

---

## Anti-Patterns

**Starting implementation without claiming the task.** Jumping straight to
coding without calling `next(id: ...)` means you miss assembled context —
spec sections, relevant knowledge entries, and file paths that the system
curated for this specific task. You'll waste time re-discovering what the
system already knows.

**Empty or vague finish summaries.** `finish(summary: "done")` tells the
next agent nothing. The summary is institutional memory — it should say what
was accomplished, what approach was taken, and any decisions made. Future
agents and humans read these summaries to understand what happened.

**Forgetting to include retrospective signals.** When you encounter
friction — a confusing spec, a missing test utility, an ambiguous design
decision — recording it via `finish(retrospective: [...])` feeds the retro
system. Without these signals, the same friction repeats across every future
task.

**Committing without running tests.** Always run `go test ./...` before
committing. A commit that breaks tests blocks every subsequent task and
requires rework. The cost of running tests is always less than the cost of
fixing a broken build.

---

## Gotchas

- If `next` fails when claiming a task, another agent likely claimed it.
  Call `next` again (without an ID) to pick a different one. Do not retry
  the same task.
- If a Kanbanzai tool call returns an error, read the message — it usually
  tells you exactly what went wrong and what the valid options are. Do not
  retry with the same arguments.
- Completing a task without `verification_performed` will succeed but
  degrades review quality. Always include what you tested, even if brief.
- If you are unsure about the workflow rules (stage gates, human vs. agent
  ownership, when to stop and ask), see `kanbanzai-workflow` — it is the
  canonical reference.

---

## Retrospective Observations

When completing a task, reflect briefly on the process — not just the output.
If you noticed any of the following, include a `retrospective` signal in your
`finish` call:

- The spec was ambiguous or contradictory on a point that mattered
- Information you needed was not in the context packet
- A tool was missing, or an existing tool was awkward or returned unhelpful output
- The task was too large, too small, or had undeclared dependencies
- A workflow step felt unnecessary or was confusing
- Something worked particularly well and should be preserved

**Not every task will have observations. That's fine — don't force it.**
When you do have something to note, be specific: name the section, the tool,
or the step that caused friction. "Things were confusing" is not useful.
"Spec §3.2 didn't define the error format for async callbacks" is.

### At task completion (primary mechanism)

Pass a `retrospective` array to `finish`:

```
finish(
  task_id: "TASK-...",
  summary: "Implemented the billing webhook handler",
  retrospective: [
    {
      category: "spec-ambiguity",
      observation: "Spec did not define retry behaviour for failed webhook deliveries",
      suggestion: "Add retry policy section to webhook specs",
      severity: "moderate"
    },
    {
      category: "worked-well",
      observation: "Context packet included the billing API idempotency entry, which saved a round of debugging",
      severity: "minor"
    }
  ]
)
```

**Valid categories:** `workflow-friction`, `tool-gap`, `tool-friction`,
`spec-ambiguity`, `context-gap`, `decomposition-issue`, `design-gap`,
`worked-well`

**Valid severities:** `minor` (slight friction), `moderate` (required
workaround), `significant` (materially slowed the work)

The `suggestion` field is optional. The `category`, `observation`, and
`severity` fields are required per signal. Invalid signals are rejected
individually — they do not block task completion or other signals.

### Outside a task context (secondary mechanism)

Observations that arise during planning, design review, or general usage
can be contributed directly:

```
knowledge(
  action="contribute",
  topic="retro-planning-session-2026-03-27",
  content="[moderate] workflow-friction: Observation here. Suggestion: ...",
  scope="project",
  tier=3,
  tags=["retrospective", "workflow-friction"]
)
```

---

## Related

- `kanbanzai-getting-started` — session orientation (what to do first)
- `kanbanzai-workflow` — stage gates and when to stop and ask the human
- `kanbanzai-documents` — document registration and approval