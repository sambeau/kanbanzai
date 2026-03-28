---
name: kanbanzai-getting-started
description: >
  Use at the start of any agent session in a Kanbanzai-managed project, even
  if the task seems straightforward. Activates when the agent has just opened
  a repository, does not yet know what work to do, needs to orient itself,
  or is beginning any new session. Also activates for "where do I start?",
  "what should I work on?", or "what's the current state of this project?".
metadata:
  kanbanzai-managed: "true"
  version: "0.2.0"
---

# SKILL: Kanbanzai Getting Started

## Purpose

Orient an agent at the start of a session in a Kanbanzai-managed project.

## When to Use

- At the beginning of any new agent session
- When you don't know what work to do or where to start
- When resuming work after a break

---

## Session Start

### 1. Clean slate

Run `git status`. If there are uncommitted changes from previous work, commit
or stash them before starting anything new. Never start new work on top of
uncommitted changes from a different task.

### 2. Read the project context

Check whether an `AGENTS.md` exists in the repository root. If it does, read
it — it contains project-specific conventions, structure, and decisions. If it
does not, these Kanbanzai skills are your primary orientation.

### 3. Check the work queue

Call `next` (without an ID) to see what tasks are ready. If the queue is
empty, call `status` or `entity` action: `list` to understand the current
project state — active features, open bugs, what stage things are in.

### 4. Before starting a task

Call `next` with a task ID to claim it and get your instructions and
context. See `kanbanzai-agents` for the full dispatch-and-complete
protocol.

### 5. Understand the workflow

Kanbanzai has stage gates that require human approval at specific points.
See `kanbanzai-workflow` for the rules, the human/agent ownership boundary,
and when to stop and ask.

---

## Related

- `kanbanzai-workflow` — stage gates, lifecycle, when to stop and ask
- `kanbanzai-documents` — document types, registration, approval
- `kanbanzai-agents` — context assembly, commits, task dispatch, knowledge
- `kanbanzai-planning` — how to run a planning conversation
- `kanbanzai-design` — how to collaborate on design documents