# Kanbanzai

Kanbanzai is a workflow system for software projects where humans and AI agents work together.

The idea is simple: humans make the decisions — what to build, what matters, what's good enough — and AI agents handle the busywork of tracking, organising, and implementing. Everything is stored as plain files in Git, so the whole project stays visible and reviewable.

---

## How it works

Most software projects end up with work scattered across chats, tickets, notes, and pull requests. When AI agents are involved, it gets worse — it becomes hard to see what was decided, what's in progress, and what an agent actually changed.

Kanbanzai keeps things organised by connecting documents to structured workflow state:

1. **A person writes a document** — a design, a specification, a plan
2. **The document gets reviewed and approved**
3. **An agent turns that approved document into structured work** — features, tasks, decisions, and the links between them
4. **Document approvals drive progress automatically** — approving a spec advances its feature to the next stage
5. **Agents implement and verify work**, keeping everything in sync
6. **Humans review the results**

The human side of the workflow stays document-based. You write, review, and approve — you don't need to manage internal records by hand. The structured side stays machine-readable, so agents can pick up where others left off.

---

## What it manages

Kanbanzai tracks work using a small set of building blocks:

| What | Purpose |
|------|---------|
| **Plan** | A body of work — a phase, a track, a project area |
| **Feature** | Something to design and build, with a document-driven lifecycle |
| **Task** | A concrete piece of implementation work |
| **Bug** | Something broken that needs fixing |
| **Decision** | A choice that was made, with rationale |
| **Document** | A design, spec, or plan with tracked approval status |
| **Knowledge entry** | A fact or convention learned during work, shared across agent sessions |

Plans coordinate. Features deliver. Documents bridge the two — a Plan's design document sets direction, and a Feature's specification drives its lifecycle forward through approval.

---

## Context and knowledge

One of the hardest problems with AI agents is context. Every new session starts from scratch — the agent doesn't know what was tried before, what conventions matter, or what the team has learned.

Kanbanzai addresses this with three layers:

- **Context profiles** — role definitions that scope what each agent should know. A backend agent gets backend conventions; a testing agent gets test strategy. Profiles are simple YAML files with inheritance, so a `backend` profile can build on a `developer` profile, which builds on a shared `base`.

- **Knowledge entries** — persistent records of things learned during work. When an agent discovers a convention or a gotcha, it can save that knowledge for future sessions. Entries earn trust over time through a confidence system — new knowledge starts uncertain and becomes authoritative through successful reuse.

- **Context assembly** — when an agent starts a task, the system assembles a targeted context packet: the relevant design fragments, applicable knowledge entries, and role conventions, all fitted within a byte budget so it works regardless of the AI model's context window size.

---

## Current status

Kanbanzai is under active development. Three implementation phases are complete:

- **Phase 1** — the workflow kernel: entities, lifecycle rules, storage, validation, and the MCP and CLI interfaces
- **Phase 2a** — document intelligence: Plans replacing Epics, document-driven Feature lifecycle, structural document analysis, and rich queries
- **Phase 2b** — context management: knowledge entries, context profiles, context assembly, agent capabilities (link resolution, duplicate detection), and batch document import

All tests pass. The system is functional and self-hosting — it manages its own project state.

What's still ahead: Git worktree management, branch tracking, and multi-agent orchestration.

---

## Getting started

Build and run with Go:

```/dev/null/sh#L1-2
go build ./cmd/kanbanzai
go run ./cmd/kanbanzai --help
```

Start the MCP server (for AI agent use):

```/dev/null/sh#L1
go run ./cmd/kanbanzai serve
```

Some useful CLI commands:

```/dev/null/sh#L1-6
kbz list features
kbz health
kbz knowledge list
kbz profile list
kbz context assemble --role backend
kbz import work/design/
```

---

## What it stores

Kanbanzai keeps project state in a `.kbz/` directory, tracked in Git:

```/dev/null/tree#L1-9
.kbz/
├── config.yaml          ← project settings and Plan prefix registry
├── state/               ← entity records (plans, features, tasks, etc.)
│   ├── knowledge/       ← knowledge entries
│   └── documents/       ← document metadata records
├── context/
│   └── roles/           ← context profile definitions
├── index/               ← document intelligence index
└── cache/               ← local derived cache (not committed)
```

Everything important is plain YAML. You can read it, diff it, and review it in a pull request.

---

## Repository layout

```/dev/null/tree#L1-7
kanbanzai/
├── cmd/kanbanzai/       ← CLI and MCP server entry point
├── internal/            ← core logic (not a library — all private packages)
├── work/                ← design, specification, and planning documents
│   ├── design/          ← design documents and policy
│   ├── spec/            ← formal specifications
│   └── plan/            ← implementation plans and progress tracking
└── .kbz/                ← project instance state
```

---

## For contributors

If you want to understand the project deeply, start with:

1. `work/bootstrap/bootstrap-workflow.md` — how we work right now
2. `work/design/workflow-design-basis.md` — the design vision
3. `work/spec/phase-1-specification.md` — Phase 1 scope
4. `work/spec/phase-2-specification.md` — Phase 2 scope
5. `work/spec/phase-2b-specification.md` — Phase 2b scope

Build and test:

```/dev/null/sh#L1-3
go build ./...
go test -race ./...
go vet ./...
```

See `AGENTS.md` for detailed contributor guidelines, code conventions, and AI agent instructions.