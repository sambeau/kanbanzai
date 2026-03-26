---
name: kanbanzai-getting-started
description: >
  Use at the start of any agent session in a Kanbanzai-managed project. Activates
  when the agent has just opened a repository, does not yet know what work to do,
  or needs to orient itself. Also activates for "where do I start?", "what should
  I work on?", or any request to begin a new session.
metadata:
  kanbanzai-managed: "true"
  version: "0.1.0"
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
or stash them before starting anything new.

### 2. Read the project context

Check whether an `AGENTS.md` exists in the repository root. If it does, read
it — it contains project-specific conventions, structure, and decisions. If it
does not, these Kanbanzai skills are your primary orientation.

### 3. Check the work queue

Call `work_queue` to see what tasks are ready for work. If the queue is empty,
call `list_entities_filtered` to understand the current project state — active
features, open bugs, what stage things are in.

### 4. Assemble context before starting a task

Before beginning work on any task, call `context_assemble` with the appropriate
role and task ID. Read the full context packet before writing any code. Do not
skip this step — it contains role instructions, relevant knowledge, and task
details that you need.

### 5. Know your role

Kanbanzai separates human concerns from agent concerns:

- **Humans own:** intent, priorities, approvals, product direction
- **Agents own:** execution, decomposition, implementation, tracking

Do not make product decisions, skip workflow stage gates, or approve your own
work.

### 6. If you are unsure — ask

It is always better to ask than to make a wrong assumption about project
structure, scope, or intent.

---

## Related

- `kanbanzai-workflow` — stage gates, lifecycle, when to stop and ask
- `kanbanzai-documents` — document types, registration, approval
- `kanbanzai-agents` — context assembly, commits, task dispatch, knowledge
- `kanbanzai-planning` — how to run a planning conversation
- `kanbanzai-design` — how to collaborate on design documents