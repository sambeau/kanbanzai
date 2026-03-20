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

This project is entering **Phase 1 implementation**. The repository contains design documents, specifications, planning documents, and research — and implementation code is now being written.

The binding contract for implementation is `work/spec/phase-1-specification.md`. The design basis is vision, the implementation plan is guidance, the spec is law. If code contradicts the spec, surface the conflict to the human.

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
3. `work/design/document-centric-interface.md` — document-centric human interface model (kbz-workflow)
4. `work/spec/phase-1-specification.md` — Phase 1 scope and verification basis (kbz-workflow)
5. `work/design/agent-interaction-protocol.md` — agent behavior and normalization protocol
6. `work/design/quality-gates-and-review-policy.md` — review expectations and quality gates
7. `work/design/git-commit-policy.md` — commit message and commit discipline policy

Then refer to these as needed:

- `work/design/workflow-system-design.md` — earlier system design document
- `work/design/machine-context-design.md` — machine-to-machine context model (Phase 2, but Phase 1 must not preclude it)
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
| How agents should behave | `work/design/agent-interaction-protocol.md` | both |
| How to review and verify work | `work/design/quality-gates-and-review-policy.md` | both |
| How to write commits | `work/design/git-commit-policy.md` | both |
| Architectural decisions made | `work/plan/phase-1-decision-log.md` | both |
| Implementation plan and work breakdown | `work/plan/phase-1-implementation-plan.md` | kbz |
| Machine context model (Phase 2) | `work/design/machine-context-design.md` | kbz |
| Document intelligence (Phase 2) | `work/design/document-intelligence-design.md` | kbz |

## Decision-Making Rules

When making a non-trivial change to any document or code:

1. Identify which spec or design document owns the topic.
2. Check `work/plan/phase-1-decision-log.md` — there are 12 accepted architectural decisions covering ID allocation, YAML format, lifecycle transitions, required fields, file layout, and more. Do not reinvent or contradict them.
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

## Phase 1 Scope Guard

Phase 1 is the workflow kernel. Build only what the spec requires. Do not build:

- Orchestration or agent delegation
- Context packing, context profiles, or knowledge management (Phase 2)
- Incident or RCA entities
- GitHub automation or webhook integration
- Semantic search or embedding-based retrieval
- Worktree automation
- Broad self-hosting automation

If you think something outside this scope is needed, stop and ask. Do not add it speculatively.

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

This project uses `codebase-memory-mcp` for structural code exploration. Prefer graph tools over file-by-file search for questions about code structure, dependencies, callers and callees, routes, hotspots, and architecture.

The graph auto-syncs after the initial index. If it seems stale, run `index_repository(repo_path="/absolute/path/to/kanbanzai")` to force a refresh.

### Fallback policy

- Use graph queries first for structural questions.
- Use text search (`grep`) for string literals, error messages, config values, and other non-structural content.
- Use `search_graph` to discover exact names before `trace_call_path`.
- Check `list_projects` first; if the project is missing or stale, run `index_repository`.
- When agent-native skills or instruction files are available, prefer those over duplicating long tool-usage instructions in this repository.
