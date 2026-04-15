# User Guide

Kanbanzai is a workflow system for human-AI collaborative software development. This guide explains what it does, how the pieces fit together, and where to find detail on each topic. It is written for designer-developers and product or design managers who know roughly what Kanbanzai is and want to understand how it works before reading the details.

After reading this guide, you will understand the workflow, orchestration, and storage model well enough to know which document to read next for any topic.

---

## What is Kanbanzai?

Kanbanzai is a **methodology** for structuring human-AI collaboration on software projects, a **workflow system** that tracks plans, features, tasks, bugs, and decisions as YAML files in your Git repository, and an **MCP server** that exposes structured tools to AI agents in any editor supporting the Model Context Protocol.

The methodology defines how work flows from an idea to a shipped feature: through design, specification, planning, implementation, and review, with human approval at each transition. The workflow system enforces that process by tracking entity state, checking prerequisites at stage gates, and preventing work from advancing until the right documents are approved. Agents interact with the system through the MCP server, which provides 22 multi-action tools covering entity management, context assembly, and merge operations.

The system is Git-native. All state lives in a `.kbz/` directory inside your repository, stored as plain YAML files that are committed alongside your code. There is no external database or cloud service, and the MCP server is not a separate process — it starts on demand when your editor launches it and communicates over stdio.

---

## Collaboration model

Humans and AI agents have distinct roles in Kanbanzai. **Humans own intent** — goals, priorities, product direction, and approvals — while **agents own execution**: decomposing work into tasks, writing code, running verification, and tracking status. Documents are the interface between the two.

This split means humans do not manage entities directly. They do not create tasks, update statuses, or wire up dependencies. Instead, they write and review design documents, make decisions in conversation, and approve the outputs that agents produce. Agents read those documents, extract the requirements, create the structured records the system needs, and do the implementation work — all within guardrails that the workflow enforces.

The result is that humans stay in control of what gets built and why, while agents handle the mechanical complexity of how. The [Workflow Overview](workflow-overview.md) covers this model and the design-to-delivery process in detail.

---

## Stage-gate workflow

Every feature in Kanbanzai passes through a defined lifecycle. Each stage produces a specific output, and a gate stands between each stage and the next.

| Stage | What happens | Output | Gate |
|-------|-------------|--------|------|
| **Proposed** | A feature is identified and described | Feature entity with summary | — |
| **Designing** | The approach is worked out | Design document | Human approves the design |
| **Specifying** | Requirements are formalised | Specification with acceptance criteria | Human approves the spec |
| **Dev-planning** | Work is broken into tasks | Development plan with task breakdown | Human approves the plan |
| **Developing** | Tasks are implemented | Code, tests, documentation | All tasks reach a terminal state |
| **Reviewing** | Implementation is evaluated against the spec | Review report | Human accepts the review |
| **Done** | Feature is complete | — | — |

The lifecycle also supports backward transitions. A feature in `reviewing` can be returned to `developing` if the review finds issues. A feature in `specifying` can go back to `designing` if the spec reveals a design gap. This means problems caught late do not require starting over — work returns to the stage where the fix belongs.

The stage-gate model is the backbone of the system. The [Workflow Overview](workflow-overview.md) covers each stage in depth, including what agents should and should not do at each point.

---

## Documents drive everything

Three document types gate the workflow: **design documents**, **specifications**, and **development plans**. Each maps to a lifecycle stage and must be approved before work advances past that stage.

A **design document** describes the approach — what problem is being solved, what the solution looks like, what alternatives were considered. It is produced during the designing stage and must be approved before specification begins.

A **specification** formalises requirements as testable assertions with acceptance criteria. It is produced during the specifying stage and must be approved before dev-planning begins. The specification becomes the contract: implementation is measured against it, and review verifies conformance to it.

A **development plan** breaks the specification into tasks with dependencies, effort estimates, and a verification approach. It is produced during the dev-planning stage and must be approved before implementation begins.

Beyond these three gated types, the system recognises several other document types — research documents, reports, policies, retrospectives, and root cause analyses — that serve supporting roles without gating the workflow. The [Workflow Overview](workflow-overview.md) explains how documents relate to each stage in detail.

---

## Approval and control

The human controls the process through document approvals. Each approval is a deliberate act that advances the workflow — approving a design allows specification to begin, approving a specification allows dev-planning, and so on. Nothing moves forward without explicit human sign-off at the gates that matter.

Approvals are recorded in the system's document records with a timestamp and the identity of the approver. They are not reversible in the casual sense — once a specification is approved, it becomes the contract for implementation. If the specification turns out to be wrong, the feature transitions backward to the specifying stage, the spec is revised, and a new approval is required.

This gives the human two levers of control: **forward approval** (letting work proceed) and **backward transition** (returning work to an earlier stage when something is wrong). Together, they mean the human never loses oversight, but also never needs to manage the mechanics of task creation, dependency resolution, or status tracking. The agents handle that within the boundaries the approved documents define.

---

## Bugs and incidents

Bugs and incidents are tracked separately from the feature workflow, each with their own lifecycle.

A **bug** tracks a code defect. Its lifecycle runs: `reported` → `triaged` → `reproduced` → `planned` → `in-progress` → `needs-review` → `verified` → `closed`. Bugs can also be marked `duplicate` or `not-planned` as terminal resolutions. A `cannot-reproduce` state sends the bug back to triage for reassessment rather than closing it. The lifecycle ensures that bugs are confirmed before work begins and verified after a fix is applied.

An **incident** tracks a production-significant failure. Incidents carry timestamps for detection, triage, mitigation, and resolution — the data needed to measure mean time to recovery. Each incident can link to affected features and to the bugs filed to fix the underlying cause. The health system flags resolved incidents that lack a linked root cause analysis, but closure is not blocked — this allows trivial or false-alarm incidents to be closed without ceremony. Severity levels — critical, high, medium, low — drive prioritisation.

Bugs and incidents surface in health checks and status reports, so they remain visible alongside feature work. The [Schema Reference](schema-reference.md) documents the full field set for both entity types.

---

## Task dispatch and context assembly

The orchestration system assembles context for agents when they claim tasks. The result is a **context packet** containing the role the agent should adopt, the skill procedure it should follow, relevant sections from the specification, knowledge entries that might help, and the file paths likely to be involved.

**Roles** define agent identity — vocabulary, anti-patterns to avoid, and tool preferences. A `doc-editor` role produces different behaviour than an `implementer-go` role, even on similar tasks, because the vocabulary and constraints differ.

**Skills** define procedures — numbered steps, checklists, evaluation criteria, and output formats. Each skill is scoped to a specific task type: writing a specification, implementing a task, reviewing code, editing documentation. Skills enforce consistency across agents and sessions.

**Task dispatch** routes ready tasks from the work queue to agents. The queue sorts tasks by priority, estimate, and age. Dependency tracking ensures a task does not become ready until everything it depends on has completed.

**Conflict awareness** checks whether tasks risk editing the same files before they are dispatched in parallel. If two tasks touch overlapping code, the system recommends serialising them or flagging the overlap for human decision.

The [Orchestration and Knowledge](orchestration-and-knowledge.md) document covers context assembly, role definitions, skill authoring, dispatch mechanics, and conflict analysis in full.

---

## Knowledge system

Knowledge entries capture lessons learned during development. When an agent completes a task, it can record observations — a pattern that worked well, a tool limitation, a design decision and its rationale. These entries persist across sessions and build up over time.

Each entry has a topic, a scope (project-wide or session-specific), a confidence score, and tags. Entries start in `contributed` status and can be promoted to `confirmed` as they prove useful, or flagged and retired if they become stale. The system tracks how often each entry is surfaced and used, so frequently used entries rank higher and unused ones drop out.

When an agent claims a new task, the context assembly system searches for knowledge entries relevant to the work and includes them in the context packet. This means an agent working on a caching feature will see previous observations about caching patterns in this codebase, even if a different agent in a different session recorded them weeks ago.

The [Orchestration and Knowledge](orchestration-and-knowledge.md) document covers the knowledge lifecycle, contribution patterns, and how entries are surfaced during context assembly.

---

## Retrospectives

The retrospective system captures workflow signals, synthesises them into patterns, and produces reports. It operates in three steps: **record**, **synthesise**, **report**.

During task completion, agents record retrospective signals — observations about what worked well, what caused friction, where tools fell short, or where specifications were ambiguous. Each signal has a category, a severity, and an optional suggestion.

The `synthesise` step clusters and ranks accumulated signals across a scope (a feature, a plan, or the whole project). It identifies recurring themes and surfaces the signals that multiple agents or sessions flagged independently.

The `report` step produces a Markdown document summarising the findings, ranked by frequency and severity. These reports feed into future planning — recurring friction points become improvement candidates, and patterns that worked well become best practices.

The [Retrospectives](retrospectives.md) document covers signal categories, severity levels, synthesis mechanics, and how to use retrospective findings in planning.

---

## Concurrency and parallel development

Kanbanzai supports multiple agents working on different features simultaneously through three mechanisms: worktrees, conflict analysis, and merge gates.

**Worktrees** provide isolated working directories. Each feature or bug gets its own Git worktree with a dedicated branch. Agents working in different worktrees cannot interfere with each other's uncommitted changes, and their commits stay on separate branches until merge.

**Conflict analysis** runs before parallel task dispatch. The system examines which files each task is likely to touch and flags overlapping scopes. The result is a per-pair risk assessment: safe to parallelise, serialise, or checkpoint required. This prevents the common failure mode where two agents edit the same file and produce merge conflicts.

**Merge gates** check readiness before a feature branch is merged into main. Gates verify that all tasks are complete, that CI passes, and that the branch is not stale relative to main. If a gate fails, the system reports what is blocking and what needs to happen before the merge can proceed.

The [Orchestration and Knowledge](orchestration-and-knowledge.md) document covers worktree setup, conflict detection mechanics, and merge gate configuration.

---

## MCP server

The MCP server is how AI agents interact with Kanbanzai — it exposes 22 structured tools covering everything from entity management to merge readiness. It communicates over stdio using the Model Context Protocol and starts automatically when your editor launches it. Any editor that supports MCP can use it: Zed, VS Code, Cursor, Claude Desktop, and others.

The 22 tools are organised into seven groups:

| Group | Tools | Purpose |
|-------|-------|---------|
| **Core** | `status`, `next`, `finish`, `handoff`, `entity`, `doc`, `health`, `server_info` | Workflow operations, entity management, status reporting |
| **Planning** | `decompose`, `estimate`, `conflict`, `retro` | Feature decomposition, sizing, conflict checks, retrospectives |
| **Knowledge** | `knowledge`, `profile` | Knowledge base management, role profiles |
| **Git** | `worktree`, `merge`, `pr`, `branch`, `cleanup` | Branch management, merge operations, PR creation |
| **Documents** | `doc_intel` | Document intelligence — structure parsing, classification, search |
| **Incidents** | `incident` | Incident lifecycle tracking |
| **Checkpoints** | `checkpoint` | Human decision checkpoints for pausing automated work |

Each tool supports multiple actions. The `entity` tool, for example, handles create, get, list, update, and transition — five distinct operations through one tool interface. This keeps the tool count manageable while covering the full workflow.

Tool groups can be enabled or disabled per project. A project that does not use incident tracking can disable the incidents group, and those tools will not appear in the agent's tool list.

The [MCP Tool Reference](mcp-tool-reference.md) documents all tools, actions, parameters, and return values.

---

## State and storage

All Kanbanzai state lives in the `.kbz/` directory at the root of your repository. This directory is committed to Git alongside your code, which means workflow state has the same versioning, branching, and collaboration properties as the code it manages.

The directory contains several areas:

- **`state/`** holds the canonical entity records — plans, features, tasks, bugs, decisions, knowledge entries, document records, and worktree tracking. Each entity is a single YAML file. These files are the system's source of truth.
- **`roles/`** holds role definitions that shape agent identity and behaviour.
- **`skills/`** holds skill definitions that provide step-by-step procedures for specific task types.
- **`config.yaml`** holds project configuration, including the plan prefix registry and tool group settings.
- **`index/`** holds derived data — document intelligence indexes and graph structures. This data is regenerated from source documents and does not need to be committed.
- **`cache/`** holds local derived data that is machine-specific and not committed.

The YAML serialisation follows strict rules: block style, deterministic field order, UTF-8 encoding, LF line endings. This ensures clean diffs when entity state changes are committed. The [Schema Reference](schema-reference.md) documents all entity types, fields, and valid values. The [Configuration Reference](configuration-reference.md) documents all configuration options.

---

## Common workflows at a glance

Most work with Kanbanzai falls into a handful of patterns. Each starts with a human decision and ends with a verified outcome.

- **Start a new feature.** Write a design document, have it approved, then let agents decompose it into a specification, dev plan, and tasks. The [Workflow Overview](workflow-overview.md) walks through every stage.
- **Review a specification or plan.** Read the document an agent produced, check it against your intent, and approve it or send it back with feedback. Approval and backward transitions are covered in the section above.
- **Investigate a bug.** File a bug entity, let agents triage and reproduce it, then track the fix through to verification. The [Schema Reference](schema-reference.md) documents the full bug lifecycle.
- **Check project health.** Use the status dashboard to see what is blocked, what is ready, and where attention is needed — across plans, features, and tasks.

These patterns compose. A typical session might start with checking status, reviewing a pending specification, and then watching agents implement the tasks from an already-approved plan.

---

## Where to go next

Your next step depends on what you want to do.

| If you want to… | Read |
|-----------------|------|
| **Try Kanbanzai** on a project | [Getting Started](getting-started.md) — install, configure your editor, run your first feature end to end |
| **Understand the workflow** in depth | [Workflow Overview](workflow-overview.md) — all stages, gates, document types, and how humans and agents collaborate |
| **Set up multi-agent orchestration** | [Orchestration and Knowledge](orchestration-and-knowledge.md) — roles, skills, context assembly, task dispatch, conflict awareness, the knowledge system |
| **Use retrospectives** to improve your process | [Retrospectives](retrospectives.md) — signal recording, synthesis, report generation |
| **Look up a tool** parameter or return value | [MCP Tool Reference](mcp-tool-reference.md) — all tools, actions, and parameters |
| **Look up an entity field** or lifecycle state | [Schema Reference](schema-reference.md) — all entity types, fields, valid values, and state machines |
| **Configure** a Kanbanzai instance | [Configuration Reference](configuration-reference.md) — `config.yaml`, `local.yaml`, prefix registry, tool groups |