---
name: kanbanzai-getting-started
description: >
  This repository is managed with Kanbanzai. Read this skill at the start of every
  session, before writing any code or running any searches. Kanbanzai provides MCP
  tools — next, entity, doc, status, and others — that replace manual grep and file
  searching for project state and work queue management. The presence of a .kbz/
  directory or kanbanzai tools in your tool list confirms you are in a
  Kanbanzai-managed project and this skill applies.
# kanbanzai-managed: true
# kanbanzai-version: dev
---

## Purpose

This skill orients you at the start of any session in a Kanbanzai-managed project.
Follow it before writing any code or making any changes.

## Preflight Check

Kanbanzai works through MCP tools. Before calling `next` or any other tool, confirm
the kanbanzai server is connected — your editor should list tools such as `next`,
`entity`, `doc`, and `status` as available.

If those tools are not available, the kanbanzai MCP server is not running. The
project's `.mcp.json` configures most editors automatically — check that the
kanbanzai binary is on your PATH and that your editor has loaded the MCP
configuration. See `docs/getting-started.md` for editor-specific setup instructions.

For Zed users: `.zed/settings.json` pre-approves all kanbanzai workflow tools so
they run without confirmation prompts. The only tools that still prompt are `merge`
(executes git merges), `pr` (creates GitHub PRs), and `cleanup` (removes worktree
directories from disk) — these have external or irreversible effects that warrant
a confirmation step. If you are seeing unexpected permission prompts for other
kanbanzai tools, check that `.zed/settings.json` contains an `agent.tool_permissions`
block; running `kbz init` will add it if missing.

If kanbanzai tool calls return unexpected errors or unknown states — transitions
failing, entity types unrecognised, responses that don't match the current codebase —
the most common cause is a **stale binary**: the running `kanbanzai serve` process was
built before the latest code changes. Run `server_info` first before investigating
further. It reports the build timestamp, git SHA, and binary path so you can confirm
whether the server matches the current source. If it is stale, rebuild and restart:
`go install ./cmd/kanbanzai/` (or the equivalent install command for your project).

Do not substitute `grep`, `find`, or direct file reading for kanbanzai tool calls.
The workflow state in `.kbz/` is structured data — the MCP tools are the correct
interface for reading and writing it.

This applies equally to writing. Do not create documents in `work/` or entities in
`.kbz/state/` by writing files directly with `edit_file` or equivalent shell commands.
Use `doc` to register and manage documents. Use `entity` to create and transition
entities. Bypassing the MCP tools skips lifecycle enforcement, document registration,
and health checks. If MCP tools are unavailable, report the issue to the human rather
than falling back to direct file writes.

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