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

## Key Terms

| Term | Meaning |
|------|---------|
| **plan** | A strategic, recursive planning entity representing scope decomposition and long-term direction. Lifecycle: `idea → shaping → ready → active → done`. ID prefix: `P{n}`. Plans are optional — most work uses batches only. |
| **batch** | An execution work container that groups features for coordinated implementation. Replaces what was previously called "plan". Lifecycle: `proposed → designing → active → reviewing → done`. ID prefix: `B{n}`. Features belong to batches; batches optionally belong to plans. |
| **stage binding** | An entry in `.kbz/stage-bindings.yaml` that maps a workflow stage to its role, skill, and prerequisites |
| **role** | A YAML file in `.kbz/roles/` defining agent identity, vocabulary, anti-patterns, and tool constraints |
| **skill** | A `SKILL.md` file defining the procedure, checklist, and evaluation criteria for a specific task type |
| **lifecycle gate** | A prerequisite check that must pass before a feature can advance to the next workflow stage |
| **context packet** | The assembled bundle of role instructions, knowledge entries, vocabulary, anti-patterns, and skill procedure delivered to an agent via `handoff`; `next` additionally includes spec sections, file paths, and graph project reference |
| **Plan** | A strategic planning container for multi-batch initiatives, with `P{n}-slug` IDs (e.g., `P1-core-platform`). Owns batches and can own design documents. |
| **Batch** | An operational work container for a group of related features, with `B{n}-slug` IDs (e.g., `B1-data-model`). Owns features and their documents. |
| **entity hierarchy** | Plan → Batch → Feature → Task. Plans contain batches; batches contain features; features break down into tasks. |

## Entity Hierarchy

The Kanbanzai entity model has four levels:

```
Plan (strategic) → Batch (execution) → Feature (deliverable) → Task (work unit)
```

- **Plan** — A strategic, recursive planning entity for scope decomposition and long-term roadmap. Plans are optional. Use a plan when multiple batches serve a shared strategic goal. Plans are human-managed; their lifecycle is `idea → shaping → ready → active → done`.
- **Batch** — An execution work container that groups features for coordinated implementation. Batches are the primary grouping entity for agentic work. All features belong to batches. A batch can exist with no parent plan. Batch lifecycle: `proposed → designing → active → reviewing → done`.
- **Feature** — A single coherent piece of user-facing behaviour that can be designed, specified, and implemented independently. Features belong to batches.
- **Task** — An atomic unit of implementation work within a feature. Tasks belong to features.

**When to use a plan vs a batch:**
- A **batch alone is sufficient** for most work. Create a batch when you need to group features for delivery.
- A **plan is warranted** when multiple batches serve a shared strategic goal that requires decomposition before execution can start.
- Plans are **optional** — batches can exist with no parent plan.
- **Err towards fewer plans.** Most work is just a batch of features.

## Task-Execution Skills and Stage Bindings

When working on a task, `.kbz/stage-bindings.yaml` maps each workflow stage (designing, specifying, dev-planning, developing, reviewing, batch-reviewing) to the role and skill that apply. Read the binding for your current stage to know what role to adopt and which skill procedure to follow. Task-execution skills live in `.kbz/skills/`; system skills (how to use the Kanbanzai workflow itself) live in `.agents/skills/`.

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
│   ├── cleanup/           ← post-merge cleanup scheduling and execution
│   ├── config/            ← project configuration and prefix registry
│   ├── context/           ← context profiles, inheritance resolution, and assembly
│   ├── core/              ← instance paths and root utilities
│   ├── docint/            ← document intelligence (structural parsing, classification, graph)
│   ├── fsutil/            ← filesystem utilities (atomic write)
│   ├── git/               ← Git operations, branch tracking, staleness detection
│   ├── github/            ← GitHub API client, PR operations
│   ├── health/            ← health check categories and formatting
│   ├── id/                ← canonical ID allocation and display formatting
│   ├── knowledge/         ← deduplication, confidence scoring, TTL pruning, promotion
│   ├── checkpoint/        ← human checkpoint creation and management
│   ├── mcp/               ← MCP server and workflow-oriented tools
│   ├── merge/             ← merge gate definitions, checker, override
│   ├── model/             ← entity type definitions and ID utilities
│   ├── service/           ← entity, batch, plan, and document record service logic
│   ├── storage/           ← canonical YAML entity and document record storage
│   ├── testutil/          ← shared test helpers
│   ├── validate/          ← lifecycle state machines, health checks
│   └── worktree/          ← worktree store, git worktree operations, naming
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
    ├── local.yaml         ← per-machine settings, not committed
    ├── state/             ← canonical entity records (plans, features, tasks, etc.)
    │   ├── plans/         ← StrategicPlan entity files
    │   ├── batches/       ← Batch entity files
    ├── state/             ← canonical entity records (batches, plans, features, tasks, etc.)
    │   ├── plans/         ← Plan entity files (strategic planning entities)
    │   ├── batches/       ← Batch entity files (execution work containers)
    │   ├── documents/     ← document metadata records
    │   ├── knowledge/     ← KnowledgeEntry records
    │   ├── worktrees/     ← worktree tracking records
    │   └── ...            ← other entity type directories
    ├── context/
    │   └── roles/         ← context profile YAML files
    ├── index/             ← document intelligence index (structural, graph, concepts)
    └── cache/             ← derived local cache (not committed)
```

## Plans vs Batches

- **Plan** (`P{n}-slug`): A strategic planning container for multi-batch initiatives. Create a plan when work spans multiple batches, when there are cross-batch architectural decisions to track, or when you need strategic rollup of progress across batches.
- **Batch** (`B{n}-slug`): An operational work container for a group of related features. Create a batch for a single coherent unit of work — this is the default container for features.
- **Legacy compatibility:** Legacy P-prefix batch IDs (e.g., `P1-my-batch`) are resolved to batches at runtime. New batches should use the `B{n}-slug` format.

## Before Every Task — Required Checklist

- [ ] Run `git status`. Act on what you find:
  - Changes from previous work are coherent and complete → **commit them now**, then proceed
  - Changes are incomplete or belong to a different task → inform the human, **do not stash** or discard (stashing hides state from parallel agents and is silently lost across worktree switches)
  - Working tree is clean → proceed
- [ ] **Commit orphaned workflow state** — if `git status` shows any modified or untracked files under `.kbz/state/`, `.kbz/index/`, or `.kbz/context/`, commit them now before starting work. MCP tools auto-commit during normal operation; orphaned files indicate an interrupted previous session. Do not stash or discard them.
- [ ] Confirm you are on the correct branch for this task (or `main` if starting fresh)

One task = one clean commit history. Uncommitted changes from a previous task make commits meaningless and code review confusing.

## Documents and Decisions

For design documents, specifications, plans, and decision logs by topic, see [`refs/document-map.md`](refs/document-map.md).

When making a non-trivial change to any document or code:

1. Identify which spec or design document owns the topic.
2. Check for prior decisions — use `refs/document-map.md` to locate the relevant decision log, or query `knowledge(action: "list")` for project-level decisions. Do not reinvent or contradict existing decisions.
3. Check whether the design basis or specification says something different from what you intend.
4. If there is a conflict or ambiguity, surface it to the human rather than guessing.

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

### Dual-write rule for skill changes

**Dual-write rule for skill changes.** The kanbanzai binary embeds `.agents/skills/kanbanzai-*/SKILL.md` files under `internal/kbzinit/skills/` for distribution to newly-initialised projects via `kanbanzai init`. When you modify any file under `.agents/skills/kanbanzai-*/`, check whether a corresponding file exists under `internal/kbzinit/skills/`. If one exists, apply the same change in the same commit.

The correspondence is: `.agents/skills/kanbanzai-<name>/SKILL.md` ↔ `internal/kbzinit/skills/<name>/SKILL.md`

Task-execution skills under `.kbz/skills/` are project-local; no dual-write applies to them.

### Store discipline

`.kbz/state/` files are versioned project state, not ephemeral cache. They record entity lifecycle, document metadata, and knowledge entries that other agents depend on. Treat them as code changes:

- Commit `.kbz/state/` changes alongside the work that produced them.
- Never stash, discard, or `.gitignore` `.kbz/` files.
- If `git status` shows orphaned `.kbz/` files at the start of a task, commit them before proceeding.

## Test Discipline

### Pre-commit hook

A pre-commit hook runs `go test ./...` before every commit. If any test fails,
the commit is blocked. The hook can be bypassed with `git commit --no-verify`
for emergency situations, but bypassing emits a prominent warning and should
generally be avoided.

The hook is installed via:
```
make setup          # installs all development dependencies including the hook
```

### No failing tests on `main`

**All tests must pass on `main` at all times.** This is a hard requirement and
a Definition of Done violation if broken. Key rules:

- Any commit that introduces a test failure on `main` is a DoD violation,
  regardless of which feature or package the change was in.
- "Pre-existing" test failure is not a valid category. A failing test means
  someone did not run tests before merging.
- If tests are failing on `main`, the first task for any developer is to fix
  them. No new work should start until the test suite is green.
- Flaky or race-detected test failures must not be dismissed as "pre-existing".
  They must be filed as a BUG entity so they appear in health reports.
- If you encounter a failing test during verification, file a BUG entity and
  block the task completion (or explicitly note the bug ID in the summary line).

### Reporting test failures

Any developer who observes a failing test on `main` is responsible for
reporting it immediately:

1. **Check whether a BUG entity already exists** for the failure.
2. **If no BUG exists**, create one:
   ```
   entity(action: "create", type: "bug", slug: "<brief-description>",
          name: "<descriptive name>", severity: "<low|medium|high|critical>",
          priority: "<low|medium|high|critical>",
          summary: "Failing test(s) in <package>: <failure description>",
          observed: "<test output or error message>",
          expected: "Tests pass unconditionally")
   ```
3. **If a BUG already exists**, add a comment with your observation.
4. **For production-level impact** (multiple packages affected, or test suite
   unusable), also file an incident:
   ```
   incident(action: "create", slug: "<slug>", severity: "<medium|high|critical>",
             summary: "Test suite failures in <packages>",
             reported_by: "<your-identity>")
   ```

The key principle: **"Not my package" is not an acceptable response.**
All team members share collective responsibility for test suite health.

### Test removal convention

Any commit that removes a test must explain in the commit message **why**
the test was removed and reference the requirement or decision that made
it obsolete. Acceptable reasons include:

- The tested behaviour was removed or changed by a specification update
  (reference the spec or design document).
- The test was superseded by a more comprehensive test in a different
  location (reference the replacement test).
- The requirement that motivated the test was explicitly obsoleted by
  a decision record (reference the decision).

Unacceptable reasons:
- "Test was flaky" — file a BUG entity for the flaky test instead.
- "Test was in the way" — restructure or update it, don't delete.
- No explanation or a generic message like "cleanup".

### Test-health gate rules

The following rules govern test discipline across the project. They are
enforced by the `test(action: "verify")` tool and the `test(action: "run")`
tool:

1. ALL tests must be maintained — no test may be disabled without a decision record.
2. NO test should ever be ignored — every test failure must be investigated.
3. NO code should be merged until all tests pass — the merge gate enforces this.
4. ALL tests must pass after merge to meet the Definition of Done — DoD Item 4.
5. There is no such thing as "just a flaky test" — file a BUG entity.
6. Pre-existing failing tests must not be ignored — they appear as `test_failure` attention items at error severity in every `status()` call.
7. The orchestrator must bail before new work starts if tests are failing — `test(action: "verify")` in Phase 0 enforces this.

## Scope Guard

Phases 1–15 and Kanbanzai 2.0/2.5 are complete. For detailed delivery history, see `docs/project-timeline.md`.

Active plans: P41 (OpenCode ecosystem features) and its children — including P42 (hash-anchored edit tool, delivered), P43 (fast-track architecture), P44 (model-routing agent launcher), P45 (wisdom forwarding), P46 (elicitation checklist), and P48 (coordination server).

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

> **Terminal tool note:** Do not use heredoc (`<<EOF`) syntax in terminal commands — it fails consistently in the `sh` shell used by the terminal tool. Use `python3 -c` with escaped strings for multi-line content, or `echo` with single quotes for short strings.

## Diagnosing Tool Failures

If MCP tool calls return unexpected errors or unknown states (e.g. transitions failing, entity types unrecognised, tool responses that don't match the current codebase), the most common cause is a **stale binary** — the running `kanbanzai serve` process was built before recent code changes.

Run `server_info` first before investigating further. It reports the build timestamp, git SHA, and binary path so you can confirm whether the server matches the current source. If the binary is stale, rebuild and restart: `go install ./cmd/kanbanzai/`.

## Go Code Style and Testing

See [`refs/go-style.md`](refs/go-style.md) for full conventions: formatting, naming, error handling, interfaces, concurrency, package design, file organisation, dependencies, and YAML serialisation rules.

See [`refs/testing.md`](refs/testing.md) for test conventions, isolation rules, and what to test.

## Codebase Knowledge Graph (`codebase-memory-mcp`)

This project is indexed under **`Users-samphillips-Dev-kanbanzai`**. Use graph tools **instead of** `grep` or `find_path` for all structural questions — definitions, callers, callees, dependencies, architecture. See [`refs/knowledge-graph.md`](refs/knowledge-graph.md) for the full tool reference and fallback policy.

**One-time setup:** Set `codebase_memory.graph_project` in `.kbz/local.yaml` once per machine:

```
codebase_memory:
  graph_project: Users-samphillips-Dev-kanbanzai
```

This value is used automatically by `worktree(action: "create")` as the default `graph_project`, so sub-agents receive Code Graph context via `next` without the orchestrator needing to pass the parameter explicitly. (`handoff` renders a Markdown prompt and does not include graph project metadata.)

## Delegating to Sub-Agents

Sub-agents do **not** see this file — all context must be explicitly propagated in every `spawn_agent` call. See [`refs/sub-agents.md`](refs/sub-agents.md) for the required context template and propagation rule.

> **Worktree sub-agents:** Sub-agents that run inside a Git worktree cannot use
> `edit_file` — it operates on the main working tree, not the worktree. Always
> instruct worktree sub-agents to write files via `terminal` using the
> `python3 -c` pattern. See the `implement-task` skill for the exact syntax.
