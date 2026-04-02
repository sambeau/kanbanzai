---
name: kanbanzai-getting-started
description: >
  Use at the start of every agent session, even if the task seems obvious and
  you think you already know what to do. Activates when the agent has just
  opened a repository, doesn't know what to do, needs to orient itself, or is
  beginning any new session. Also activates for "where do I start?", "what
  should I work on?", "what's the current state?". Skipping orientation leads
  to wasted effort and missed context.
metadata:
  kanbanzai-managed: "true"
  version: "0.3.0"
---

# SKILL: Kanbanzai Getting Started

## Purpose

Orient an agent at the start of a session in a Kanbanzai-managed project.

## When to Use

- At the beginning of any new agent session
- When you don't know what work to do or where to start
- When resuming work after a break

## Vocabulary

- **Session orientation** — The mandatory startup sequence that establishes what state the project is in, what work is available, and what conventions apply before any implementation begins.
- **Work queue** — The prioritised list of ready tasks returned by `next()`. The single source of truth for what an agent should work on.
- **Context packet** — The assembled bundle of spec sections, knowledge entries, file paths, and role conventions returned when claiming a task with `next(id)`.
- **Clean slate** — The state where `git status` shows no uncommitted changes from previous work. Required before starting any new task.
- **Task claim** — The act of calling `next(id)` to transition a task from `ready` to `active` and receive its context packet.
- **Feature lifecycle state** — The current workflow stage of a feature entity (e.g. `designing`, `specifying`, `implementing`). Determines which skills, roles, and gates apply.
- **AGENTS.md** — The project-specific conventions file in the repository root. Contains structure, build commands, Git discipline, and the pre-task checklist.
- **Stage binding** — The mapping in `.kbz/stage-bindings.yaml` that connects each workflow stage to its required role, skill, and prerequisites.
- **Project status** — The synthesised dashboard returned by `status()` showing progress, blocked items, and attention items across the project.

---

## Session Start Checklist

Copy this checklist at the beginning of every session:

- [ ] **Clean slate** — Run `git status`. Commit coherent changes, stash incomplete work, or proceed if clean.
- [ ] **Store check** — If `git status` shows uncommitted `.kbz/` files, commit them now. These are versioned project state, not ephemeral cache. Do not stash, discard, or `.gitignore` them.
- [ ] **Read project context** — Read `AGENTS.md` if you haven't this session.
- [ ] **Check the work queue** — Call `next()` to see what's ready.
- [ ] **Claim your task** — Call `next(id: "TASK-xxx")` to get full context for your chosen task.
- [ ] **Understand the workflow** — If unsure about the current stage, check the `kanbanzai-workflow` skill.

### Clean slate

Run `git status`. If there are uncommitted changes from previous work, commit
or stash them before starting anything new. Never start new work on top of
uncommitted changes from a different task.

### Read the project context

Check whether an `AGENTS.md` exists in the repository root. If it does, read
it — it contains project-specific conventions, structure, and decisions. If it
does not, these Kanbanzai skills are your primary orientation.

### Check the work queue

Call `next` (without an ID) to see what tasks are ready. If the queue is
empty, call `status` or `entity` action: `list` to understand the current
project state — active features, open bugs, what stage things are in.

### Claim your task

Call `next` with a task ID to claim it and get your instructions and
context. See `kanbanzai-agents` for the full dispatch-and-complete
protocol.

### Understand the workflow

Kanbanzai has stage gates that require human approval at specific points.
See `kanbanzai-workflow` for the rules, the human/agent ownership boundary,
and when to stop and ask.

Each workflow stage (designing, specifying, developing, reviewing, etc.) maps
to a specific **role** and **skill** via `.kbz/stage-bindings.yaml`. Read the
binding for your current stage to know which role to adopt and which skill
procedure to follow. The task-execution skills themselves live in
`.kbz/skills/` (e.g. `write-design`, `write-spec`, `review-code`,
`orchestrate-review`).

---

## Anti-Patterns

### Skipping Orientation

- **Detect:** Agent starts implementing without calling `next()` or reading AGENTS.md.
- **BECAUSE:** Without orientation, the agent misses project conventions, active decisions, and existing work — leading to duplicated effort or conflicting changes.
- **Resolve:** Always run the session start checklist before writing any code.

### Stale Context Carry-Over

- **Detect:** Agent assumes context from a previous session is still current (e.g. task status, branch state, file contents).
- **BECAUSE:** Entity states, knowledge entries, and file contents change between sessions. Stale assumptions produce incorrect implementation decisions that compound as work progresses.
- **Resolve:** Always check `git status` and call `next()` at session start. Treat every session as a fresh start.

### Store Neglect

- **Detect:** Uncommitted `.kbz/state/` files visible in `git status` at session start.
- **BECAUSE:** Store drift causes race conditions when parallel agents read stale entity state, leading to conflicting transitions and lost updates.
- **Resolve:** Commit `.kbz/` files immediately. Do not stash, discard, or gitignore them.

---

## Evaluation Criteria

1. **Did the agent run `git status` and address uncommitted changes before starting work?** — Required
2. **Did the agent call `next()` to check the work queue?** — Required
3. **Did the agent claim a specific task before beginning implementation?** — High
4. **Did the agent read AGENTS.md if it was their first action in the session?** — Medium

---

## Questions This Skill Answers

- What do I do at the start of a session?
- How do I find out what work is available?
- How do I claim a task?
- What if there are uncommitted changes from a previous session?
- Where are the project conventions?
- What's the difference between system skills and task-execution skills?
- How do I know what stage the project is in?

---

## Related

- `kanbanzai-workflow` — stage gates, lifecycle, when to stop and ask
- `kanbanzai-documents` — document types, registration, approval
- `kanbanzai-agents` — context assembly, commits, task dispatch, knowledge
- `kanbanzai-planning` — how to run a planning conversation
- `write-design` — how to collaborate on design documents