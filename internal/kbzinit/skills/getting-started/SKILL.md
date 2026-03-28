---
name: kanbanzai-getting-started
description: >
  Orient yourself at the start of any session in a Kanbanzai-managed project: before
  writing any code, when you don't know what work to do next, after switching tasks,
  or when opening a repository for the first time. Follow this skill before any other.
# kanbanzai-managed: true
# kanbanzai-version: dev
---

## Purpose

This skill orients you at the start of any session. Follow it before writing any code
or making any changes.

## Before Any Work

Run `git status`. If there are uncommitted changes from a previous session:

- If coherent and complete → commit them before starting new work.
- If incomplete or risky → stash them and note this for the human.

Never start new work on top of uncommitted changes from a different task.

## Understand the Project

Check for `AGENTS.md` in the repository root. If it exists, read it — it contains
project-specific conventions, structure, decisions, and reading order that override
generic guidance. If it does not exist, the Kanbanzai skills are your primary orientation.

## Check the Work Queue

Call `next` (without an ID) to see what tasks are ready. The work queue promotes eligible tasks
and returns them sorted by estimate and age.

If the queue is empty, call `status` or `entity` action: `list` to understand the current project
state: active features, open bugs, and their statuses.

## Assemble Context Before Starting a Task

Before beginning work on any task, call `next` with a task ID to claim it and receive
a context packet containing the task instructions, relevant knowledge entries, and
design context.

See `kanbanzai-agents` for the full dispatch-and-complete protocol.

## Understand the Workflow

Kanbanzai enforces stage gates that require human approval at specific points. Do not
skip stages or create entities without meeting the gate conditions.

See `kanbanzai-workflow` for:

- The six stage gates and what each requires
- What humans own vs. what agents own
- When to stop and ask the human (the emergency brake)

---

## Related

- `kanbanzai-workflow` — stage gates, entity lifecycle, human/agent boundary
- `kanbanzai-documents` — document registration and approval
- `kanbanzai-agents` — context assembly, task dispatch, commit format, knowledge contribution