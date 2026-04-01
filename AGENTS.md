# Agent Instructions for Kanbanzai

## Overview

Kanbanzai is a Git-native workflow system for human-AI collaborative software development.

Its primary purpose is the coordination of AI agent teams to efficiently turn designs into working software. It does this through an MCP server that replaces ad-hoc search, grep, and manual context-gathering with structured operations — presenting the right instructions, the right context, and the right constraints to each agent at the right time. Internally, it maintains structured workflow state (epics, features, tasks, bugs, and decisions) as schema-validated YAML files tracked in Git, enforces lifecycle state machines and referential integrity, and attaches verification to every unit of work.

Humans interact with the system through documents and chat, not by managing entities directly. They write and review design documents, make decisions in conversation, and approve results. AI agents mediate between this document interface and the structured internal model — extracting decisions, creating entity records, assembling targeted context for other agents, and maintaining consistency across the project. The core idea is that humans own intent — goals, priorities, approvals, and product direction — while AI agents own execution: decomposing work, implementing, verifying, and tracking status, all within strict guardrails.

It is designed to be simple for small projects and to scale to concurrent multi-agent teams working in isolated Git worktrees, while always staying simpler than the project it manages.

## Naming Conventions

Use these terms consistently. They refer to different things.

| Term | What it means |
|---|---|
| **Kanbanzai** (or "the Kanbanzai System") | The system/methodology, used in prose the way people say "Scrum" or "Kanban" |
| **`kanbanzai`** | The tool binary and MCP server name (used in `.mcp.json` config, `kanbanzai serve`) |
| **`.kbz/`** | The instance root directory (project-local workflow state) |

The binary is `kanbanzai`. When invoked with `kanbanzai serve`, it runs as the MCP server. Both modes share the same core logic.

Examples:
- "We're adopting Kanbanzai for our workflow." (the system)
- `kanbanzai serve` (MCP server, launched by MCP client)
- `.kbz/state/` (instance directory on disk)

## Self-Managed Development

Kanbanzai manages its own development — this project uses the tool it is building. The workflow rules, stage gates, and lifecycle states are defined in the `.agents/skills/kanbanzai-*/SKILL.md` files (the product interface) and enforced by the MCP tools. This file (`AGENTS.md`) contains only project-specific instructions for developing the kanbanzai server itself.

## Repository Structure

```
kanbanzai/
├── AGENTS.md              ← you are here
├── README.md              ← document map and reading guide
├── cmd/kanbanzai/         ← binary entry point (CLI and MCP server)
├── internal/              ← core logic (shared by MCP server and CLI)
│   ├── cache/             ← local derived SQLite cache
│   ├── cleanup/           ← post-merge cleanup scheduling and execution (Phase 3)
│   ├── config/            ← project configuration and prefix registry (Phase 2a + Phase 2b user identity)
│   ├── context/           ← context profiles, inheritance resolution, and assembly (Phase 2b)
│   ├── core/              ← instance paths and root utilities
│   ├── docint/            ← document intelligence (structural parsing, classification, graph)
│   ├── fsutil/            ← filesystem utilities (atomic write)
│   ├── git/               ← Git operations, branch tracking, staleness detection (Phase 3)
│   ├── github/            ← GitHub API client, PR operations (Phase 3)
│   ├── health/            ← health check categories and formatting (Phase 3)
│   ├── id/                ← canonical ID allocation and display formatting
│   ├── knowledge/         ← deduplication, confidence scoring, link resolution (Phase 2b), TTL pruning, promotion, compaction (Phase 3)
│   ├── checkpoint/        ← human checkpoint creation and management (Phase 4a)
│   ├── mcp/               ← MCP server and 22 workflow-oriented 2.0 tools across 7 feature groups (Kanbanzai 2.0)
│   ├── merge/             ← merge gate definitions, checker, override (Phase 3)
│   ├── model/             ← entity type definitions and ID utilities
│   ├── service/           ← entity, plan, and document record service logic
│   ├── storage/           ← canonical YAML entity and document record storage
│   ├── testutil/          ← shared test helpers
│   ├── validate/          ← lifecycle state machines, health checks
│   └── worktree/          ← worktree store, git worktree operations, naming (Phase 3)
├── docs/                  ← user-facing and reference documentation (reserved for later)
├── work/                  ← active design, spec, planning, and research documents
│   ├── bootstrap/         ← historical process documents
│   ├── design/            ← design documents and policy documents
│   ├── spec/              ← formal specifications
│   ├── plan/              ← implementation plans, decision log, and progress tracking
│   ├── research/          ← background analysis and review memos
│   └── reviews/           ← feature and bug review reports from the reviewing lifecycle gate
└── .kbz/                  ← instance root (project-local workflow state, not committed)
    ├── config.yaml        ← project configuration including prefix registry
    ├── local.yaml            ← per-machine settings, not committed (Phase 2b)
    ├── state/             ← canonical entity records (plans, features, tasks, etc.)
    │   ├── plans/         ← Plan entity files (Phase 2a)
    │   ├── documents/     ← document metadata records (Phase 2a)
    │   ├── knowledge/     ← KnowledgeEntry records (Phase 2b)
    │   ├── worktrees/     ← worktree tracking records (Phase 3)
    │   └── ...            ← other entity type directories
    ├── context/
    │   └── roles/            ← context profile YAML files (Phase 2b)
    ├── index/             ← document intelligence index (structural, graph, concepts)
    └── cache/             ← derived local cache (not committed)
```

## Before Every Task — Required Checklist

**Copy this checklist. Complete every item before writing any code or documents.**

- [ ] Run `git status`. Act on what you find:
  - Changes from previous work are coherent and complete → **commit them now**, then proceed
  - Changes are incomplete or belong to a different task → **stash them** and inform the human
  - Working tree is clean → proceed
- [ ] Confirm you are on the correct branch for this task (or `main` if starting fresh)
- [ ] Read this file (`AGENTS.md`) if you haven't already this session
- [ ] If the task involves design decisions, read the relevant spec or design document before touching any file

**Why this matters:** Uncommitted changes from a previous task bleed into your new work. This makes commits meaningless (one commit covers two unrelated tasks), makes `git bisect` impossible, and makes code review confusing. One task = one clean commit history. This step is not optional.

## Document Reading Order

If you need to understand the project, read in this order:

1. `work/design/workflow-design-basis.md` — consolidated design vision
2. `work/design/document-centric-interface.md` — document-centric human interface model
3. `work/spec/phase-1-specification.md` — Phase 1 scope and verification basis
4. `work/spec/phase-2-specification.md` — Phase 2 scope and verification basis
5. `work/design/agent-interaction-protocol.md` — agent behavior and normalization protocol
6. `work/design/quality-gates-and-review-policy.md` — review expectations and quality gates
7. `work/design/git-commit-policy.md` — commit message and commit discipline policy

Then refer to these as needed:

- `work/spec/phase-4a-specification.md` — Phase 4a scope and verification basis
- `work/plan/phase-2a-progress.md` — Phase 2a implementation status and remaining work
- `work/plan/phase-2-scope.md` — Phase 2 scope and planning
- `work/spec/phase-2b-specification.md` — Phase 2b scope and verification basis
- `work/plan/phase-2b-implementation-plan.md` — Phase 2b implementation plan and audit remediation
- `work/plan/phase-2-decision-log.md` — Phase 2 architectural decisions
- `work/design/workflow-system-design.md` — earlier system design document
- `work/design/machine-context-design.md` — machine-to-machine context model (implemented in Phase 2b)
- `work/design/document-intelligence-design.md` — structural analysis backend for design documents (Phase 2)
- `work/design/product-instance-boundary.md` — product vs. instance separation
- `work/plan/phase-1-implementation-plan.md` — concrete execution plan
- `work/plan/phase-1-decision-log.md` — architectural decisions

## Key Design Documents by Topic

| Topic | Document |
|---|---|
| Historical process documents | `work/bootstrap/bootstrap-workflow.md` |
| What the system is and why | `work/design/workflow-design-basis.md` |
| How humans interact with the system | `work/design/document-centric-interface.md` |
| What Phase 1 must deliver | `work/spec/phase-1-specification.md` |
| What Phase 2 must deliver | `work/spec/phase-2-specification.md` |
| Phase 2a implementation status | `work/plan/phase-2a-progress.md` |
| Phase 2 scope and planning | `work/plan/phase-2-scope.md` |
| How agents should behave | `work/design/agent-interaction-protocol.md` |
| How to review and verify work | `work/design/quality-gates-and-review-policy.md` |
| Code review SKILL (procedure + orchestration) | `.skills/code-review.md` |
| Plan review SKILL (procedure + checklist) | `.skills/plan-review.md` |
| How to write commits | `work/design/git-commit-policy.md` |
| Architectural decisions made | `work/plan/phase-1-decision-log.md` |
| Implementation plan and work breakdown | `work/plan/phase-1-implementation-plan.md` |
| Machine context model (Phase 2) | `work/design/machine-context-design.md` |
| Document intelligence (Phase 2) | `work/design/document-intelligence-design.md` |
| Phase 2b specification | `work/spec/phase-2b-specification.md` |
| Phase 2b implementation plan | `work/plan/phase-2b-implementation-plan.md` |
| Phase 2 decisions | `work/plan/phase-2-decision-log.md` |
| Phase 3 spec and status | `work/spec/phase-3-specification.md`, `work/plan/phase-3-progress.md` |
| Phase 4a specification | `work/spec/phase-4a-specification.md` |
| Phase 4b specification | `work/spec/phase-4b-specification.md` |
| Phase 4b implementation plan | `work/plan/phase-4b-implementation-plan.md` |
| Phase 4 decisions | `work/plan/phase-4-decision-log.md` |



## Decision-Making Rules

When making a non-trivial change to any document or code:

1. Identify which spec or design document owns the topic.
2. Check `work/plan/phase-1-decision-log.md` — there are 12 accepted architectural decisions covering ID allocation, YAML format, lifecycle transitions, required fields, file layout, and more. Do not reinvent or contradict them.
3. Check `work/plan/phase-2-decision-log.md` — there are Phase 2 architectural decisions (P2-DEC-001 through P2-DEC-004) covering context profiles, knowledge management, and related topics. Do not reinvent or contradict them.
4. Check `work/plan/phase-3-decision-log.md` — there are Phase 3 design decisions (P3-DES-001 through P3-DES-008) covering worktree lifecycle, branch naming, merge gates, PR scope, and cleanup behavior. Do not reinvent or contradict them.
5. Check `work/plan/phase-4-decision-log.md` — there are Phase 4 design decisions (P4-DES-001 through P4-DES-007) covering phase split, estimation, self-management thresholds, dependency modelling, agent delegation, incidents/RCA, and document store deprecation. Do not reinvent or contradict them.
6. Check whether the design basis or specification says something different from what you intend.
7. If there is a conflict or ambiguity, surface it to the human rather than guessing.

## Git Discipline

### Branch and merge roles

- AI commits to feature/bug branches.
- AI merges to main.
- AI can push to remote when delegated by human.
- Human creates release tags.

For commit message format, types, and examples, see the `kanbanzai-agents` skill.

### During work

- Commit at logical checkpoints: after completing a coherent change, before starting a risky edit.
- A change isn't done until it's committed.
- This applies equally to design documents, decision records, and planning changes — not just code. A drafted decision or a renamed term across multiple files is a coherent change that should be committed.

### Commit granularity for documents

- A new or updated decision record → commit when the decision is complete.
- A new document → commit when it's coherent and reviewed.
- A cross-cutting rename or terminology change → commit as a single coherent change covering all affected files.
- Multiple unrelated document changes in one session → split into separate commits by topic.

Do not let document changes accumulate uncommitted across long sessions.

## Scope Guard

Phase 1 (workflow kernel), Phase 2a (entity model evolution, document intelligence, migration), Phase 2b (context profiles, knowledge management, user identity), Phase 3 (Git integration, knowledge lifecycle), Phase 4a (estimation, work queue, dispatch, human checkpoints, orchestration health), Phase 4b (feature decomposition, automatic unblocking, worker review, conflict analysis, vertical slice guidance, incidents/RCA, document store removal), Kanbanzai 2.0 (MCP tool surface redesign — all 11 tracks A–K complete), P6 (workflow quality and code review), P7 (developer experience), P8 (decompose reliability), P9 (MCP discoverability and reliability), P10 (review workflow and documentation currency), P11 (fresh install experience — MCP server connection, embedded review skills, default context roles, standard document layout), and P12 (agent onboarding and skill discovery — AGENTS.md generation, copilot instructions, specification skill, MCP orientation breadcrumbs) are all complete. There is no current in-progress phase. For detailed delivery history, see `docs/project-timeline.md`.

Do not build beyond the current phase without explicit direction:

- Cross-project knowledge sharing
- GitLab, Bitbucket, or other platform support (beyond GitHub)
- Webhook-based real-time synchronisation
- Semantic search or embedding-based retrieval
- Broad self-hosting automation

If you think something outside current scope is needed, stop and ask. Do not add it speculatively.

The implementation plan (`work/plan/phase-1-implementation-plan.md` §9) defines additional constraints: no silent scope expansion, no conflation of product and project state, no reliance on future orchestration, no destructive workflows by default.

## Build and Test Commands

```
go build ./...          # build everything
go test ./...           # run all tests
go test -race ./...     # run tests with race detector
go vet ./...            # static analysis
go fmt ./...            # format all code
goimports -w .          # organise imports
go mod tidy             # clean up dependencies
```

## Go Code Style and Testing

See [`refs/go-style.md`](refs/go-style.md) for full conventions: formatting, naming, error handling, interfaces, concurrency, package design, file organisation, dependencies, and YAML serialisation rules.

See [`refs/testing.md`](refs/testing.md) for test conventions, isolation rules, and what to test.

## Codebase Knowledge Graph (`codebase-memory-mcp`)

This project is indexed under **`Users-samphillips-Dev-kanbanzai`**. Use graph tools **instead of** `grep` or `find_path` for all structural questions — definitions, callers, callees, dependencies, architecture. See [`refs/knowledge-graph.md`](refs/knowledge-graph.md) for the full tool reference and fallback policy.

## Delegating to Sub-Agents

Sub-agents do **not** see this file — all context must be explicitly propagated in every `spawn_agent` call. See [`refs/sub-agents.md`](refs/sub-agents.md) for the required context template and propagation rule.
