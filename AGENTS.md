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
| **`kbz`** | The CLI shorthand — a symlink or invocation alias to the same binary, for human use in the terminal |
| **`.kbz/`** | The instance root directory (project-local workflow state) |

The binary is `kanbanzai`. When invoked as `kbz` or with a CLI subcommand, it runs in CLI mode. When invoked with `kanbanzai serve`, it runs as the MCP server. Both modes share the same core logic.

Examples:
- "We're adopting Kanbanzai for our workflow." (the system)
- `kanbanzai serve` (MCP server, launched by MCP client)
- `kbz status` (CLI, typed by a human)
- `.kbz/state/` (instance directory on disk)

## Project Status

**Phase 1 is complete. Phase 2a is complete. Phase 2b is complete. Phase 3 is complete. Phase 4a is complete. Phase 4b is complete.** The repository contains design documents, specifications, planning documents, research, and working implementation code. All Phase 2a acceptance criteria are met, all audit bugs (B1–B8) are fixed, and all tests pass with race detector enabled. All Phase 2b acceptance criteria are met. All Phase 3 acceptance criteria (§20.1–§20.12) are met, all audit remediation items (R1–R17) are resolved, automatic worktree creation on task/bug transition is implemented, and all tests pass with race detector enabled. All Phase 4a acceptance criteria are met: estimation tools, work queue, dispatch/complete task, human checkpoints, and orchestration health dashboard are implemented, and all tests pass with race detector enabled. All Phase 4b acceptance criteria (§16.1–§16.7) are met: feature decomposition and review, automatic dependency unblocking, worker review with rework lifecycle, conflict domain analysis with work queue integration, vertical slice guidance, incidents and RCA, and Phase 1 document store removal are implemented, and all tests pass with race detector enabled.

The binding contracts for implementation are `work/spec/phase-1-specification.md` (Phase 1), `work/spec/phase-2-specification.md` (Phase 2), `work/spec/phase-2b-specification.md` (Phase 2b), `work/spec/phase-3-specification.md` (Phase 3), `work/spec/phase-4a-specification.md` (Phase 4a), and `work/spec/phase-4b-specification.md` (Phase 4b). The design basis is vision, the implementation plan is guidance, the spec is law. If code contradicts the spec, surface the conflict to the human.

Current Phase 2a status is tracked in `work/plan/phase-2a-progress.md`. Phase 2b status is tracked in `work/plan/phase-2b-progress.md`. Phase 3 status is tracked in `work/plan/phase-3-progress.md`.

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
│   ├── mcp/               ← MCP server and tools (Phase 1 + Phase 2a + Phase 2b + Phase 3 + Phase 4a)
│   ├── merge/             ← merge gate definitions, checker, override (Phase 3)
│   ├── model/             ← entity type definitions and ID utilities
│   ├── service/           ← entity, plan, and document record service logic
│   ├── storage/           ← canonical YAML entity and document record storage
│   ├── testutil/          ← shared test helpers
│   ├── validate/          ← lifecycle state machines, health checks
│   └── worktree/          ← worktree store, git worktree operations, naming (Phase 3)
├── docs/                  ← user-facing and reference documentation (reserved for later)
├── work/                  ← active design, spec, planning, and research documents
│   ├── bootstrap/         ← bootstrap-workflow: the process we follow now
│   ├── design/            ← kbz-workflow: design documents and policy documents
│   ├── spec/              ← kbz-workflow: formal specifications
│   ├── plan/              ← implementation plans, decision log, and progress tracking
│   └── research/          ← background analysis and review memos
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

## Before Any Task

1. Run `git status` — if there are uncommitted changes from previous work, commit or stash before starting new work.
2. Read this file (`AGENTS.md`).
3. Read `work/bootstrap/bootstrap-workflow.md` — it defines the process we follow right now.
4. If the task involves understanding the system design, follow the reading order below.

## Document Reading Order

If you need to understand the project, read in this order:

1. `work/bootstrap/bootstrap-workflow.md` — how we work right now (bootstrap-workflow)
2. `work/design/workflow-design-basis.md` — consolidated design vision (kbz-workflow)
3. `work/design/document-centric-interface.md` — document-centric human interface model (kbz-workflow)
4. `work/spec/phase-1-specification.md` — Phase 1 scope and verification basis (kbz-workflow)
5. `work/spec/phase-2-specification.md` — Phase 2 scope and verification basis (kbz-workflow)
6. `work/design/agent-interaction-protocol.md` — agent behavior and normalization protocol
7. `work/design/quality-gates-and-review-policy.md` — review expectations and quality gates
8. `work/design/git-commit-policy.md` — commit message and commit discipline policy

Then refer to these as needed:

- `work/spec/phase-4a-specification.md` — Phase 4a scope and verification basis (kbz-workflow)
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

| Topic | Document | Workflow |
|---|---|---|
| What we do right now | `work/bootstrap/bootstrap-workflow.md` | bootstrap |
| What the system is and why | `work/design/workflow-design-basis.md` | kbz |
| How humans interact with the system | `work/design/document-centric-interface.md` | kbz |
| What Phase 1 must deliver | `work/spec/phase-1-specification.md` | kbz |
| What Phase 2 must deliver | `work/spec/phase-2-specification.md` | kbz |
| Phase 2a implementation status | `work/plan/phase-2a-progress.md` | both |
| Phase 2 scope and planning | `work/plan/phase-2-scope.md` | kbz |
| How agents should behave | `work/design/agent-interaction-protocol.md` | both |
| How to review and verify work | `work/design/quality-gates-and-review-policy.md` | both |
| How to write commits | `work/design/git-commit-policy.md` | both |
| Architectural decisions made | `work/plan/phase-1-decision-log.md` | both |
| Implementation plan and work breakdown | `work/plan/phase-1-implementation-plan.md` | kbz |
| Machine context model (Phase 2) | `work/design/machine-context-design.md` | kbz |
| Document intelligence (Phase 2) | `work/design/document-intelligence-design.md` | kbz |
| Phase 2b specification | `work/spec/phase-2b-specification.md` | kbz |
| Phase 2b implementation plan | `work/plan/phase-2b-implementation-plan.md` | kbz |
| Phase 2 decisions | `work/plan/phase-2-decision-log.md` | both |
| Phase 3 spec and status | `work/spec/phase-3-specification.md`, `work/plan/phase-3-progress.md` | kbz |
| Phase 4a specification | `work/spec/phase-4a-specification.md` | kbz |
| Phase 4b specification | `work/spec/phase-4b-specification.md` | kbz |
| Phase 4b implementation plan | `work/plan/phase-4b-implementation-plan.md` | kbz |
| Phase 4 decisions | `work/plan/phase-4-decision-log.md` | both |

## Communicating With Humans

Documents are the human interface to the system. Decision records and their IDs are internal tracking mechanisms — important for system integrity and useful for agents, but not how humans navigate the project.

When talking with humans:

- Reference **documents** by name: "the ID system design", "the Phase 1 spec §10"
- Use **prose descriptions** of decisions: "the decision about cache-based locking"
- Do **not** lead with decision IDs: ~~"P1-DEC-021 defines the ID format"~~

Decision IDs don't carry enough context for a human to act on without querying the system. A document name or a prose summary is immediately meaningful. Save decision IDs for commit messages, entity cross-references, and agent-to-agent communication.

This rule is also codified in the agent interaction protocol (`work/design/agent-interaction-protocol.md` §6.11).

## Document Creation Workflow

When you create a new document in the `work/` directory, you must register it with the kanbanzai system. Documents are the human interface, but the system needs metadata records to track status, ownership, and lifecycle.

**Follow the `document-creation` SKILL:** `.skills/document-creation.md`

The SKILL provides step-by-step procedures for:
- Registering single documents with `doc_record_submit`
- Batch importing multiple documents with `batch_import_documents`
- Document type mapping (directory → type)
- Safety checks and verification
- Troubleshooting common issues

**Key principle:** Always register documents immediately after creating them. Unregistered documents are invisible to document intelligence, entity extraction, approval workflow, and health checks.

**Quick reference:**

```
# Single document
doc_record_submit(path="work/design/my-doc.md", type="design", title="...")

# Batch import (idempotent, safe to repeat)
batch_import_documents(path="work")
```

## Decision-Making Rules
</text>


When making a non-trivial change to any document or code:

1. Identify which spec or design document owns the topic.
2. Check `work/plan/phase-1-decision-log.md` — there are 12 accepted architectural decisions covering ID allocation, YAML format, lifecycle transitions, required fields, file layout, and more. Do not reinvent or contradict them.
3. Check `work/plan/phase-2-decision-log.md` — there are Phase 2 architectural decisions (P2-DEC-001 through P2-DEC-004) covering context profiles, knowledge management, and related topics. Do not reinvent or contradict them.
4. Check `work/plan/phase-3-decision-log.md` — there are Phase 3 design decisions (P3-DES-001 through P3-DES-008) covering worktree lifecycle, branch naming, merge gates, PR scope, and cleanup behavior. Do not reinvent or contradict them.
5. Check `work/plan/phase-4-decision-log.md` — there are Phase 4 design decisions (P4-DES-001 through P4-DES-007) covering phase split, estimation, self-management thresholds, dependency modelling, agent delegation, incidents/RCA, and document store deprecation. Do not reinvent or contradict them.
6. Check whether the design basis or specification says something different from what you intend.
7. If there is a conflict or ambiguity, surface it to the human rather than guessing.

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

## Scope Guard

Phase 1 (workflow kernel), Phase 2a (entity model evolution, document intelligence, migration), Phase 2b (context profiles, knowledge management, user identity), Phase 3 (Git integration, knowledge lifecycle), Phase 4a (estimation, work queue, dispatch, human checkpoints, orchestration health), and Phase 4b (feature decomposition, automatic unblocking, worker review, conflict analysis, vertical slice guidance, incidents/RCA, document store removal) are complete. Current work should focus on Phase 5 planning and implementation as directed by the human.

Do not build beyond the current phase without explicit direction:

- Cross-project knowledge sharing
- GitLab, Bitbucket, or other platform support (beyond GitHub)
- Webhook-based real-time synchronisation
- Semantic search or embedding-based retrieval
- Broad self-hosting automation

If you think something outside current scope is needed, stop and ask. Do not add it speculatively.

The implementation plan (`work/plan/phase-1-implementation-plan.md` §9) defines additional constraints: no silent scope expansion, no conflation of product and project state, no reliance on future orchestration, no destructive workflows by default.

## YAML Serialisation Rules

Entity state and documents are stored as YAML. Deterministic, canonical serialisation is a core requirement — not a nice-to-have. The accepted decision P1-DEC-008 in the decision log defines the exact rules:

- Block style for mappings and sequences (no flow style)
- Double-quoted strings only when required by YAML syntax
- Deterministic field order (defined per entity type)
- UTF-8, LF line endings, trailing newline
- No YAML tags, anchors, or aliases
- No multi-document streams

Do not rely on Go's default YAML marshaller to produce correct output. The serialisation must be explicit and tested with round-trip tests (write → read → write → compare).

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

## Go Code Style

### Formatting
- Write idiomatic Go
- Run `go fmt` before committing
- Use `goimports` for import organisation
- Maximum line length: 100 characters (soft limit)

### Naming
- Use camelCase for unexported identifiers
- Use PascalCase for exported identifiers
- Acronyms should be consistent case: `URL`, `HTTP`, `ID` (not `Url`, `Http`, `Id`)
- Package names: lowercase, single word, no underscores

### Error Handling
- Always check errors; never use `_` to ignore them
- Wrap errors with context: `fmt.Errorf("doing X: %w", err)`
- Return errors, don't panic (except for truly unrecoverable situations)
- Define sentinel errors with `errors.New` for errors that callers need to check

### Comments
- Exported functions must have doc comments
- Doc comments start with the function name: `// FunctionName does...`
- Use `// TODO:` for planned improvements
- Use `// FIXME:` for known issues

### Interfaces
- Accept interfaces, return structs
- Define interfaces at the consumer, not the provider
- Keep interfaces small — one or two methods is ideal
- Do not define interfaces preemptively; extract them when a second implementation or a test double is needed

### Concurrency
- Do not use goroutines unless there is a demonstrated need
- Phase 1 is a request-response system — no concurrent workflows
- If goroutines are needed later, pass `context.Context` and use it for cancellation

### Package Design
- Keep packages small and focused on a single responsibility
- No circular imports — if two packages need each other, extract shared types into a third
- The `internal/` directory is not importable from outside this module
- No `init()` functions — they create hidden coupling and make testing harder

## File Organisation
```
cmd/kanbanzai/    # binary entry point
internal/         # all private packages (core logic, MCP server, CLI)
```

This is not a library. There is no `pkg/` directory.

## Dependencies
- Prefer the standard library when reasonable
- Run `go mod tidy` after adding/removing dependencies
- Commit `go.sum` with `go.mod`

## Testing

### Conventions
- Test files: `*_test.go` in the same package
- Test functions: `TestFunctionName_Scenario`
- Use table-driven tests for multiple cases
- Aim for meaningful coverage, not 100%

### Test isolation
- Tests must not depend on external services or network calls
- Use `t.TempDir()` for filesystem tests — never write to the working directory
- Test fixtures live in `testdata/` directories alongside the test files
- Test helpers must call `t.Helper()` so failures report the caller's line number

### What to test
- Core validation logic (field validation, lifecycle transitions, referential integrity)
- Serialisation and deterministic formatting (round-trip: write → read → compare)
- ID allocation edge cases
- Document validation (valid and invalid cases)
- MCP operations (integration tests where practical)
- CLI behaviour (integration tests where practical)

### What not to test
- Do not test the standard library
- Do not write tests that only assert that a mock was called — test behaviour, not wiring
- Do not test unexported functions directly unless they contain complex logic worth isolating

## Codebase Knowledge Graph (`codebase-memory-mcp`)

This project is indexed in `codebase-memory-mcp` under the project name **`Users-samphillips-Dev-kanbanzai`** with root path `/Users/samphillips/Dev/kanbanzai`.

The graph is the preferred way to explore code structure. Use it **instead of** `grep` or `find_path` whenever you need to understand definitions, relationships, callers, callees, dependencies, or architecture.

### When to use graph tools (preferred)

| Question | Tool | Example |
|----------|------|---------|
| What does a function/type look like? | `get_code_snippet` | `get_code_snippet(qualified_name="EntityService.Get")` |
| Who calls this function? | `trace_call_path` | `trace_call_path(function_name="ResolvePrefix", direction="inbound")` |
| What does this function call? | `trace_call_path` | `trace_call_path(function_name="Get", direction="outbound")` |
| Find a function/class/type by name | `search_graph` | `search_graph(name_pattern="Allocat")` |
| Understand package structure | `get_architecture` | `get_architecture(project="Users-samphillips-Dev-kanbanzai")` |
| Complex cross-package queries | `query_graph` | Cypher queries for multi-hop analysis |

### When to use text search (fallback)

Use `grep` only for content that is not structural:

- String literals and error messages
- Config values and magic constants
- YAML field names in test fixtures
- Comments and documentation text
- Broad "does this string appear anywhere?" sweeps

Use `find_path` only when searching by filename pattern, not by code content.

### Keeping the graph current

The graph auto-syncs after the initial index. If results seem stale or the project is missing from `list_projects`, force a refresh:

```
index_repository(repo_path="/Users/samphillips/Dev/kanbanzai")
```

### Fallback policy

1. Use graph queries first for structural questions.
2. Use `search_graph` to discover exact qualified names before `trace_call_path` or `get_code_snippet`.
3. Fall back to `grep` only for non-structural content searches.
4. Fall back to `read_file` only when you need to see exact file content that the graph doesn't cover (e.g., full test bodies, YAML fixtures).

---

## Delegating to Sub-Agents

When you spawn sub-agents (via `spawn_agent`), those agents do **not** see this file. They only know what you tell them. This means critical project context — tool preferences, conventions, the knowledge graph — is lost unless you explicitly propagate it.

### Required context for every sub-agent

Include the following in every `spawn_agent` message:

1. **Codebase knowledge graph availability:**

   > This project is indexed in `codebase-memory-mcp` as project `Users-samphillips-Dev-kanbanzai`. Prefer graph tools over grep/find for structural code questions:
   > - `search_graph(name_pattern="...", project="Users-samphillips-Dev-kanbanzai")` to find functions, types, classes
   > - `get_code_snippet(qualified_name="...", project="Users-samphillips-Dev-kanbanzai")` to read a specific symbol
   > - `trace_call_path(function_name="...", project="Users-samphillips-Dev-kanbanzai")` to find callers/callees
   > - `get_architecture(project="Users-samphillips-Dev-kanbanzai")` for package structure
   > Use `grep` only for string literals, error messages, and non-structural content.

2. **File scope boundaries** — which files the agent should and should not modify (to avoid conflicts with parallel agents).

3. **Any relevant project conventions** — e.g., commit message format, test conventions, Go style rules — if the agent will be committing or writing tests.

### Propagation rule

If a sub-agent may itself spawn further sub-agents, include this instruction:

> When you delegate work to sub-agents, include the codebase-memory-mcp context (project name, tool preferences) in your delegation message. Sub-agents do not see project instructions automatically.

This ensures the context propagates through any depth of delegation, not just one level.

### Why this matters

Without this context, sub-agents will default to `grep` and `read_file` for everything — scanning files line by line instead of using the indexed graph. This is slower, noisier, and misses structural relationships that the graph captures directly.
