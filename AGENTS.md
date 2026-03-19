# Agent Instructions for Kanbanzai

## Overview

Kanbanzai is a Git-native workflow system for human-AI collaborative software development.

It stores structured workflow state (epics, features, tasks, bugs, and decisions) as schema-validated YAML files tracked in Git, and exposes them through an MCP server that AI agents use as their primary interface and a CLI for humans. The core idea is that humans own intent — goals, priorities, approvals, and product direction — while AI agents own execution: decomposing work, implementing, verifying, and tracking status, all within strict guardrails. Agents normalise informal human input into clean canonical records through a clarify-before-commit pipeline, the system enforces lifecycle state machines and referential integrity, and every unit of work carries attached verification. It is designed to be simple for small projects and to scale to concurrent multi-agent teams working in isolated Git worktrees, while always staying simpler than the project it manages.

## Naming Conventions

Use these terms consistently. They refer to different things.

| Term | What it means |
|---|---|
| **Kanbanzai** (or "the Kanbanzai System") | The system/methodology, used in prose the way people say "Scrum" or "Kanban" |
| **`kanbanzai`** | The tool binary and MCP server name (used in `.mcp.json` config, `kanbanzai serve`) |
| **`kbz`** | The CLI shorthand — a symlink or invocation alias to the same binary, for human use in the terminal |
| **`.kbz/`** | The instance root directory (project-local workflow state) |

The binary is `kanbanzai`. When invoked as `kbz` or with a CLI subcommand, it runs in CLI mode. When invoked with `kanbanzai serve`, it runs as the MCP server. Both modes share the same core logic.

Examples:
- "We're adopting Kanbanzai for our workflow." (the system)
- `kanbanzai serve` (MCP server, launched by MCP client)
- `kbz status` (CLI, typed by a human)
- `.kbz/state/` (instance directory on disk)

## Project Status

This project is in the **design and planning phase**. There is no implementation code yet. The repository contains design documents, specifications, planning documents, and research that define what the system will be.

## Two Workflows

This project has two distinct workflows. Do not confuse them.

- **kbz-workflow** — the workflow process the Kanbanzai tool will implement and enforce. Described in `work/design/` and `work/spec/`. This is what we are *building*.
- **bootstrap-workflow** — the simplified process we use right now to build Kanbanzai. Described in `work/bootstrap/bootstrap-workflow.md`. This is what we *follow*.

When working on this project, follow bootstrap-workflow. When designing or implementing the tool, refer to kbz-workflow.

If you are unsure which workflow a rule or instruction belongs to, ask.

## Repository Structure

```
kanbanzai/
├── AGENTS.md              ← you are here
├── README.md              ← document map and reading guide
├── docs/                  ← user-facing and reference documentation (empty, reserved for later)
└── work/                  ← active design, spec, planning, and research documents
    ├── bootstrap/         ← bootstrap-workflow: the process we follow now
    ├── design/            ← kbz-workflow: design documents and policy documents
    ├── spec/              ← kbz-workflow: formal specifications
    ├── plan/              ← implementation plans and decision log
    └── research/          ← background analysis and review memos

When implementation begins:
├── cmd/kanbanzai/         ← binary entry point
├── internal/              ← core logic (shared by MCP server and CLI)
└── .kbz/                  ← instance root (project-local workflow state, not committed)
```

## Before Any Task

1. Run `git status` — if there are uncommitted changes from previous work, commit or stash before starting new work.
2. Read this file (`AGENTS.md`).
3. Read `work/bootstrap/bootstrap-workflow.md` — it defines the process we follow right now.
4. If the task involves understanding the system design, follow the reading order below.

## Document Reading Order

If you need to understand the project, read in this order:

1. `work/bootstrap/bootstrap-workflow.md` — how we work right now (bootstrap-workflow)
2. `work/design/workflow-design-basis.md` — consolidated design vision (kbz-workflow)
3. `work/spec/phase-1-specification.md` — Phase 1 scope and verification basis (kbz-workflow)
4. `work/design/agent-interaction-protocol.md` — agent behavior and normalization protocol
5. `work/design/quality-gates-and-review-policy.md` — review expectations and quality gates
6. `work/design/git-commit-policy.md` — commit message and commit discipline policy

Then refer to these as needed:

- `work/design/workflow-system-design.md` — earlier system design document
- `work/design/product-instance-boundary.md` — product vs. instance separation
- `work/plan/phase-1-implementation-plan.md` — concrete execution plan
- `work/plan/phase-1-decision-log.md` — architectural decisions

## Key Design Documents by Topic

| Topic | Document | Workflow |
|---|---|---|
| What we do right now | `work/bootstrap/bootstrap-workflow.md` | bootstrap |
| What the system is and why | `work/design/workflow-design-basis.md` | kbz |
| What Phase 1 must deliver | `work/spec/phase-1-specification.md` | kbz |
| How agents should behave | `work/design/agent-interaction-protocol.md` | both |
| How to review and verify work | `work/design/quality-gates-and-review-policy.md` | both |
| How to write commits | `work/design/git-commit-policy.md` | both |
| Architectural decisions made | `work/plan/phase-1-decision-log.md` | both |
| Implementation plan and work breakdown | `work/plan/phase-1-implementation-plan.md` | kbz |

## Decision-Making Rules

When making a non-trivial change to any document:

1. Identify which spec or design document owns the topic.
2. Check whether a decision in `work/plan/phase-1-decision-log.md` has already resolved the question.
3. Check whether the design basis or specification says something different from what you intend.
4. If there is a conflict or ambiguity, surface it to the human rather than guessing.

## Git Rules

- AI commits to feature/bug branches.
- AI merges to main.
- AI can push to remote when delegated by human.
- Human creates release tags.
- Use commit message format: `<type>(<object-id>): <summary>`

### Commit types

Per `work/design/git-commit-policy.md`:

- `feat` — new feature behavior
- `fix` — bug fix
- `docs` — documentation change
- `test` — test-only change
- `refactor` — behavior-preserving structural improvement
- `workflow` — workflow-state-only change
- `decision` — decision-record change
- `chore` — small maintenance change with no better category

Add `!` after the type for breaking changes: `feat(FEAT-001)!: description`

### Examples

- `feat(FEAT-152): add profile editing API and validation`
- `fix(BUG-027): prevent avatar uploads hanging on large files`
- `docs(FEAT-152): update profile editing user documentation`
- `workflow(TASK-152.3): mark upload task complete after verification`
- `decision(DEC-041): record no-client-side-cropping choice`

## Preserving Work Through Commits

### Before starting new work

Run `git status`. If there are uncommitted changes from previous work:
- If the changes are coherent and complete → commit with an appropriate message.
- If the changes are incomplete or risky → stash and inform the human.
- Never start new work on top of uncommitted changes from a different task.

### During work

- Commit at logical checkpoints: after completing a coherent change, before starting a risky edit.
- A change isn't done until it's committed.
- This applies equally to design documents, decision records, and planning changes — not just code. A drafted decision or a renamed term across multiple files is a coherent change that should be committed.

### Commit granularity for documents

During the current design/planning phase, most work produces document changes. Commit these the same way you would commit code:

- A new or updated decision record → commit when the decision is complete.
- A new document (e.g., bootstrap-workflow.md) → commit when it's coherent and reviewed.
- A cross-cutting rename or terminology change → commit as a single coherent change covering all affected files.
- Multiple unrelated document changes in one session → split into separate commits by topic.

Do not let document changes accumulate uncommitted across long sessions.

## Documentation Accuracy

- **Code is truth** — if documentation conflicts with code, fix the documentation.
- **Spec is intent** — if code conflicts with the specification, surface the conflict to the human.
- Do not silently resolve spec-vs-code conflicts in either direction without human input.

## When Implementation Begins

Once the project has implementation code, this file should be updated to include:

- Build and test commands
- Test policy and definition of done
- Code style and language conventions
- Project-specific tooling instructions