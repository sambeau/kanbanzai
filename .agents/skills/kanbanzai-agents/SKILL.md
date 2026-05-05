---
name: kanbanzai-agents
description: >
  Use when dispatching implementation tasks to sub-agents, committing
  changes, finishing tasks, or contributing knowledge. Activates when
  there is a task to implement, a commit to make, or an entity to
  complete. Also activates for entity creation, naming, and data
  entry decisions.
metadata:
  kanbanzai-managed: "true"
  version: "0.4.0"
---

# SKILL: Kanbanzai Agents

## Purpose

The dispatch-and-complete protocol for agent-managed workflow tasks,
commit conventions, knowledge contribution, and entity naming rules.

## When to Use

- When dispatching a task to a sub-agent
- When completing a task with `finish`
- When committing code changes
- When contributing knowledge entries
- When naming an entity for the first time
- When creating or updating entity records

---

## Vocabulary

- **Task lifecycle** — `ready` → `active` → `done` (or `needs-rework`).
  Tasks transition `ready` → `active` when claimed via `next(id)`, and
  `active` → `done` when completed via `finish`.
- **Post-completion summary** — A 2-3 sentence reduction of a task's
  output, stored in the finish call. The summary replaces the full output
  in the orchestrator's context.
- **Knowledge entry** — A structured record with topic, content, scope,
  and optional tags, contributed at task completion via `finish`.
- **Retrospective signal** — An observation about workflow friction,
  tool gaps, or what worked well, contributed at task completion.
- **Task claim** — Calling `next(id)` to transition a task from `ready`
  to `active` and receive its full context packet.

---

## Task Lifecycle Checklist

- [ ] Task is `ready` → claim with `next(id)`
- [ ] Implement the task
- [ ] Commit changes before running `finish`
- [ ] Call `finish(task_id, summary, knowledge?, retrospective?, files_modified?)`

**Do not call `finish` on a task you have not claimed.**

---

## Context Assembly

When a task is claimed with `next(id)`, the system assembles a context
packet containing:

- The task's specification sections from the feature's specification document
- Knowledge entries matching the task's tags
- File paths from the dev-plan
- Role conventions from the feature's assigned role
- Graph project context from the parent entity's worktree record

Do not append additional documents or spec sections beyond what `next`
returns. Adding extra context dilutes the task-specific signal the agent
needs and wastes token budget.

---

## Task Dispatch and Completion

This section covers the lifecycle of a single task, from claim to finish.

### Picking up work

1. Call `status` to check project state and find available work.
2. Call `next()` to see the ready queue.
3. Call `next(id: "TASK-xxx")` to claim the task and receive context.
4. Implement the task. Follow the task's acceptance criteria as defined
   in the spec.
5. Commit changes before finishing.

### Finishing work

1. Ensure all changes are committed. `git status` should show a clean tree.
2. Call `finish(task_id: "TASK-xxx", summary: "...")` with:
   - A concise summary of what was accomplished (2-3 sentences)
   - Knowledge entries for anything learned that others should know
     (non-obvious gotchas, workarounds, design rationale)
   - Retrospective signals for workflow observations
   - The `files_modified` list for file-change tracking
3. Do NOT call `finish` on tasks in `ready` or `needs-rework` status.

---

## Feature Completion

When all tasks for a feature reach a terminal state (`done`, `cancelled`,
or `duplicate`), the feature is not yet complete — it must be explicitly
advanced through the remaining lifecycle stages:

1. **Advance the feature** — Call `entity(action: "transition", id: "FEAT-xxx", status: "reviewing")`.
   This transitions the feature from `developing` to `reviewing`, making it
   visible in the review queue.
2. **Merge the branch** — Call `merge(action: "check")` then `merge(action: "execute")`.
3. **Delete the branch** — Verify with `git branch | grep FEAT-xxx`. If the
   branch still exists, delete it manually.
4. **Knowledge curation** — Run `knowledge(action: "list", status: "contributed", tier: 2)`.
   Confirm or flag each entry as appropriate.

For the full close-out procedure, see `.kbz/skills/orchestrate-development/SKILL.md`
Phase 6.

---

## Commit Message Format

Commit messages follow a structured format:

```
type(scope): description
```

### Types

| Type | When to use |
|------|-------------|
| `feat` | New feature or capability |
| `fix` | Bug fix |
| `docs` | Documentation only |
| `refactor` | Code change that neither fixes nor adds |
| `test` | Adding or fixing tests |
| `chore` | Maintenance, config, CI, dependencies |
| `workflow` | State/entity changes (`.kbz/state/` files). Also used for workflow automation changes. |

### Examples

```
feat(entity): add batch display ID formatting
fix(merge): handle empty cohort edge case
docs(agents): update entity hierarchy section
workflow(state): commit orphaned knowledge entries
```

### Bad vs. Good

| Bad | Good |
|-----|------|
| `fix stuff` | `fix(merge): handle empty cohort edge case` |
| `Update` | `docs(agents): add plan-vs-batch guidance` |
| `feat: lots of changes` (no scope) | `feat(entity): add batch display ID formatting` (scope + description) |

### Commit discipline

- **One commit = one coherent change.** Do not mix unrelated changes in
  the same commit. A documentation update belongs in `docs(scope)`, a
  code change belongs in `feat(scope)` or `fix(scope)`.
- **Commit before finish.** All changes must be committed before calling
  `finish`. If `git status` shows uncommitted changes when you receive a
  finish confirmation, commit them immediately.
- **Worktree commits are local.** Changes committed in a Git worktree are
  on the feature branch, not `main`. They are isolated by design.

---

## Knowledge Contribution

Knowledge entries are structured facts or patterns that future agents on
the project should know. They are contributed at task completion via
`finish` and automatically stored in `.kbz/state/knowledge/`.

**When to contribute:**
- **Non-obvious gotchas** — something that wasted time figuring out
- **Invariants** — constraints that must hold that aren't in the docs
- **Design rationale** — why a choice was made, especially if it's
  counterintuitive
- **Tool workarounds** — patterns for dealing with MCP tool limitations

**When NOT to contribute:**
- Obvious facts from the spec or design (those are already documented)
- Temporary state (current task status, current branch)
- Personal opinions about code style

**Format:** Each knowledge entry has: topic (identifier), content (concise
statement), scope (project or profile name), and optional tags.

---

## Entity Names

Every entity (plan, batch, feature, task, bug, decision) requires a `name` field. These rules
apply to all entity types.

### Hard Rules

These are code-enforced. Violations are rejected with an error:

- **Required** — `name` must be set on every entity. Never omit it.
- **60-character maximum.**
- **No colon (`:`).**
- **No phase or version prefix** — do not start a name with a pattern like `P4`, `P8`,
  or `P11` followed by a space or dash. The phase is already encoded in the entity's
  parent field.
- Leading/trailing whitespace is stripped automatically.

### Quality Guidance

- **Target approximately four words.** If you need more than five or six, the scope is
  too broad or the name is doing the summary's job.
- **No em-dashes as separators.** Hyphens in compound technical terms are fine
  (`"Human-friendly ID display"`), but not `"P8 — decompose"`.
- **A name should not be merely the slug capitalised.** `init-command` → `"Init command"`
  adds nothing. Prefer `"Project init command"` or `"Init and skill install"`.
- **Names must be self-contained** — readable without knowing the parent entity.
  `"Update agents"` is ambiguous; `"Update AGENTS.md layout"` is not.
- **Do not include the parent batch or feature name** in the entity name (or parent
  plan name, if one exists) — the hierarchy is visible from the parent field.

### Examples

Good names:

| Entity | Name | Why |
|--------|------|-----|
| Plan | Kanbanzai 2.0 | Short, identity-oriented |
| Batch | Webhook delivery | ~3 words, describes the grouping |
| Feature | Human-friendly ID display | ~4 words, no prefix, self-contained |
| Feature | Init and skill install | ~4 words, descriptive, no prefix |
| Task | Server info tool | ~3 words, clear |
| Task | Label model and storage | ~4 words |
| Decision | Use TSID for entity IDs | Self-contained, concise |

Bad names:

| Entity | Name | Problem |
|--------|------|---------|
| Plan | P4 Kanbanzai 2.0: MCP Tool Surface Redesign | Phase prefix + colon |
| Feature | P8 — decompose propose Reliability Fixes | Phase prefix + separator dash |
| Feature | The kanbanzai init command: creates .kbz/config.yaml | Colon + far too long — this is a summary |
| Task | Update | Too vague, not self-contained |

---

## Communicating With Humans

When communicating with a human in chat:

- **Be concise.** State the outcome, the decision needed, and what happens next.
  Do not narrate reasoning or describe intermediate steps unless asked.
- **State the decision needed explicitly.** "Should I proceed with X?" not "What
  do you think about X?"
- **Use structured formats** (lists, tables) for complex information. Plain prose
  for simple messages.
- **Keep one message per topic.** Separate concerns into distinct messages so the
  human can respond to each independently.
- **No checklists in chat.** The session start checklist runs in the agent's head,
  not in the chat. Only present structured output when it serves a purpose.

---

## Sub-Agent Spawning

When spawning a sub-agent (via `spawn_agent` or similar mechanism outside
`handoff`), the sub-agent must receive:

1. **A clear task statement** — what to build, what files to modify, what
   acceptance criteria to satisfy
2. **File scope boundaries** — which files it owns and which it must not touch
3. **The parent entity context** — feature ID, batch ID, and any relevant
   spec or design context
4. **The `spawn_agent` JSON template** (see `refs/sub-agents.md` section
   "Required context template")

---

## Anti-Patterns

### Unclaimed Implementation

- **Detect:** Writing code for a task without calling `next(id)` to claim it first.
- **BECAUSE:** An unclaimed task is visible as `ready` to other agents, who may
  claim and start implementing it simultaneously — creating duplication and merge
  conflicts. The system assumes claimed tasks are being worked on; unclaimed tasks
  are assumed to be available.
- **Resolve:** Always call `next(id)` before implementing. If the task is in
  `needs-rework`, claim it to signal that you are addressing the findings.

### Empty Finish Summary

- **Detect:** `finish` is called with a one-word summary or a generic statement
  like "Done" or "Implemented the feature".
- **BECAUSE:** Orchestrators and reviewers depend on finish summaries to build
  their mental model of what changed. An empty summary leaves them without
  evidence, forcing them to re-read diffs or re-dispatch to understand what
  happened.
- **Resolve:** Write a specific 2-3 sentence summary: what was built, any
  notable decisions, verification performed.

### Forgotten Retrospective

- **Detect:** A task that encountered significant friction, tool gaps, or
  ambiguous specifications completes without any retrospective signal.
- **BECAUSE:** Retrospective signals are the primary mechanism for improving
  the workflow. Every significant friction point that goes unrecorded is a
  missed opportunity to fix systemic issues.
- **Resolve:** Always include at least one retrospective signal when the task
  had notable friction. Even a minor observation is valuable — patterns emerge
  from many minor observations.

### Uncommitted Tests

- **Detect:** A `finish` summary mentions tests but `git status` shows new or
  modified test files that are not staged or committed.
- **BECAUSE:** Uncommitted tests are invisible to the reviewing stage. Reviewers
  check test files to verify acceptance criteria coverage; if tests are not in
  the commit history, the feature appears untested regardless of what the
  summary claims.
- **Resolve:** Commit test files before calling `finish`. Verify with `git status`.

---

## Gotchas

- Do not create decision entities unless a decision was explicitly made during
  this task — retrospective observations belong in `finish` signals, not in
  decision entities.
- When contributing retrospective signals via `finish`, use the `retrospective`
  parameter, not the `knowledge` parameter. Retrospective signals are
  observational ("this took longer than expected because..."), knowledge
  entries are factual ("this API requires X because Y").
- The `files_modified` list in `finish` is for file-change tracking, not for
  changelog. List files that were added or modified, not the nature of the
  change.

---

## Retrospective Observations

Retrospective signals capture workflow friction and successes — not code
quality, but the experience of producing the code. They are the primary
mechanism for improving the developer experience over time.

### At task completion (primary mechanism)

Use the `retrospective` parameter of `finish`:

```
finish(
  task_id: "TASK-xxx",
  summary: "...",
  retrospective: [
    {
      category: "workflow-friction",  # or tool-gap, tool-friction,
                                      # spec-ambiguity, context-gap,
                                      # decomposition-issue, design-gap,
                                      # worked-well
      observation: "The handoff tool didn't include X context",
      severity: "minor"               # or moderate, significant
      suggestion: "Add X to the handoff assembly pipeline"  # optional
    }
  ]
)
```

Each signal is one observation. Do not combine multiple observations into a
single entry — they are analysed individually.

### Outside a task context

When a workflow problem is discovered that isn't tied to a specific task
(e.g. a tool always behaves unexpectedly), flag it:

1. Call `knowledge(action: "contribute", topic: "...", content: "...")` to
   register the observation as project-level knowledge
2. The knowledge entry serves as the persistent record of the issue

---

## Evaluation Criteria

1. **Was the task claimed with `next(id)` before implementation began?** (required)
2. **Were all changes committed before calling `finish`?** (required)
3. **Does the finish summary describe what was built and how it was verified?** (required)
4. **Were knowledge entries contributed for non-obvious findings?** (high)
5. **Were retrospective signals included when the task had notable friction?** (high)
6. **Are entity names descriptive, self-contained, and free of phase prefixes?** (required)

---

## Questions This Skill Answers

- How do I claim and complete a task?
- What goes in a commit message?
- How do I contribute knowledge?
- What are entity naming rules?
- How do I communicate with the human?
- How do I spawn a sub-agent?
- What is a retrospective signal and when do I contribute one?

---

## Related

- `kanbanzai-getting-started` — session orientation, work queue
- `kanbanzai-workflow` — stage gates, lifecycle, when to stop
- `kanbanzai-documents` — document management
- `orchestrate-development` — orchestrating multi-task feature work
- `refs/sub-agents.md` — required context template for spawn_agent