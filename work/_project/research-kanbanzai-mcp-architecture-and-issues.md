# Kanbanzai MCP: What It Is, How It Works, and What's Wrong

**Date:** 2026-05-11
**Purpose:** Evidence document for architectural review — should we stay the course or start afresh?

---

## Part 1: What Is Kanbanzai MCP?

### 1.1 The elevator pitch

Kanbanzai is a **Git-native workflow system for human-AI collaborative software development**. It provides an MCP (Model Context Protocol) server — `kanbanzai serve` — that exposes 22 structured workflow tools which AI agents use to manage plans, batches, features, tasks, bugs, documents, knowledge, reviews, merges, and more.

It is not a CI/CD system, not a project management SaaS, and not a code-generation framework. It is a **workflow execution layer** that sits between the human developer and the AI coding agents, providing guardrails, state tracking, and a shared collaboration surface.

### 1.2 Core architecture

```
┌──────────────────────────────────────────────────┐
│                AI Agent (Chat)                   │
│  (Claude, GPT, DeepSeek — via MCP client)        │
└────────────────────┬─────────────────────────────┘
                     │ 22 MCP tools (JSON-RPC)
                     ▼
┌──────────────────────────────────────────────────┐
│           kanbanzai serve (MCP Server)           │
│                                                  │
│  ┌──────────┐ ┌───────────┐ ┌────────────────┐  │
│  │  Entity  │ │    Doc    │ │   Knowledge    │  │
│  │  Service │ │  Service  │ │    Service     │  │
│  └────┬─────┘ └─────┬─────┘ └───────┬────────┘  │
│       │             │               │            │
│  ┌────▼─────────────▼───────────────▼────────┐   │
│  │         .kbz/state/  (YAML files)         │   │
│  │   plans/  features/  tasks/  bugs/ ...    │   │
│  └───────────────────────────────────────────┘   │
│                                                  │
│  ┌──────────────────────────────────────────┐    │
│  │  Stage Bindings → Roles → Skills → Docs  │    │
│  │  (context assembly pipeline)             │    │
│  └──────────────────────────────────────────┘    │
└────────────────────┬─────────────────────────────┘
                     │
                     ▼
┌──────────────────────────────────────────────────┐
│              Git Repository                      │
│  .kbz/state/ (versioned) + source code + docs    │
└──────────────────────────────────────────────────┘
```

### 1.3 The 22 MCP tools

All tools use action-based dispatch (e.g., `entity(action: "create|get|list|transition")`). The consolidated tools are:

| Tool | Purpose |
|------|---------|
| `entity` | CRUD + lifecycle transitions for plans, batches, features, tasks, bugs, decisions, incidents |
| `doc` | Register, approve, query, validate document records |
| `knowledge` | Contribute, confirm, flag, retire knowledge entries |
| `next` | Claim the next ready task with assembled context |
| `finish` | Complete a task, contribute knowledge, record retro signals |
| `handoff` | Assemble a sub-agent prompt with role, skill, knowledge, and tool hints |
| `status` | Project/plan/feature dashboards with health, progress, attention items |
| `health` | Comprehensive consistency checks across entities, worktrees, knowledge |
| `decompose` | Break a feature spec into implementation tasks |
| `estimate` | Story-point sizing with Modified Fibonacci scale |
| `conflict` | Check parallel-task conflict risk |
| `retro` | Synthesise retrospective signals, generate reports, create fix features |
| `profile` | List/inspect role profiles |
| `worktree` | Create, remove, list, GC Git worktrees for feature isolation |
| `merge` | Gate-checked merge of feature branches |
| `pr` | Create/status GitHub PRs from entity metadata |
| `branch` | Check branch health, staleness, drift from main |
| `cleanup` | Reclaim merged/abandoned worktrees |
| `doc_intel` | Explore, classify, search the document knowledge graph |
| `incident` | Track production incidents through lifecycle |
| `checkpoint` | Pause for human decisions during automated orchestration |
| `server_info` | Diagnose stale-binary and version-mismatch issues |

### 1.4 Entity model

Kanbanzai manages seven entity types, each with its own lifecycle state machine enforced by `internal/validate/lifecycle.go`:

| Entity | ID Prefix | Lifecycle States |
|--------|-----------|-----------------|
| Strategic Plan | `P{n}` | idea → shaping → ready → active → done |
| Batch | `B{n}` | proposed → designing → active → reviewing → done |
| Feature | `FEAT-...` | idea → shaping → specifying → dev-planning → developing → reviewing → merging → verifying → done |
| Task | `TASK-...` | queued → ready → active → needs-review → needs-rework → done |
| Bug | `BUG-...` | reported → triaged → reproduced → planned → in-progress → needs-review → verifying → closed |
| Decision | `DEC-...` | proposed → accepted / rejected / superseded |
| Incident | `INC-...` | reported → triaged → investigating → mitigated → resolved → closed |

IDs use TSID13: a single-character type prefix followed by a 13-character time-sortable identifier (e.g., `FEAT-01KMKA278DFNV`). Plans and batches use shorter human-friendly prefixes (`P1`, `B1`).

### 1.5 How state is stored

All canonical state lives in **YAML files under `.kbz/state/`**, organized by entity type:

```
.kbz/state/
  plans/          P1-my-plan.yaml
  features/       FEAT-01KMKA278DFNV.yaml
  tasks/          TASK-01KMKA278DFNW.yaml
  bugs/           BUG-01KMKA278DFNX.yaml
  documents/      FEAT-01KMKA278DFNV/specification-foo.yaml
  knowledge/      KE-01KMKA278DFNY.yaml
  worktrees/      WT-01KMKA278DFNZ.yaml
  checkpoints/    CHK-01KMKA278DFN0.yaml
  incidents/      INC-01KMKA278DFN1.yaml
```

These files are **versioned in Git** — Git serves as both transport and durability boundary. Entity YAML follows a deterministic canonical field order to produce clean diffs.

### 1.6 Roles, skills, and stage bindings

Kanbanzai has an evidence-based **roles and skills system** that tells AI agents *who they are* and *what they're doing* at each workflow stage.

- **Roles** (`.kbz/roles/*.yaml`): Define identity, vocabulary, anti-patterns, and tool constraints. Use inheritance (e.g., `reviewer-security` inherits from `reviewer`).
- **Skills** (`.kbz/skills/*/SKILL.md`): Define procedure, vocabulary, anti-patterns, and checklist for a specific task type.
- **Stage bindings** (`.kbz/stage-bindings.yaml`): Map each workflow stage (designing, specifying, developing, reviewing, etc.) to the role and skill that apply.

There are 18 roles and ~20 skills. The system has two skill directories: `.agents/skills/` (project-local) and `.kbz/skills/` (task-execution), plus embedded copies in `internal/kbzinit/skills/` for new-project initialization.

### 1.7 Sub-agent dispatch model

The current dispatch flow is a **three-step chat-orchestrated pipeline**:

1. **`next(id: "TASK-...")`** — claims the task, returns assembled context (spec sections, knowledge entries, file paths, role conventions)
2. **`handoff(task_id: "TASK-...")`** — reads the stage binding, loads role + skill, assembles a complete sub-agent prompt (~9K tokens) with vocabulary, anti-patterns, tool hints, and knowledge
3. **`spawn_agent(message: <handoff output>)`** — the orchestrator manually calls spawn_agent with the assembled prompt

The orchestrator (a chat-based AI agent running the `orchestrate-development` skill) coordinates this flow, deciding which tasks to dispatch, to which sub-agents, and in what order.

### 1.8 Worktree isolation

Each feature and bug gets its own **Git worktree** — a separate working directory with its own branch — created via `worktree(action: "create", entity_id: "FEAT-...")`. This isolates parallel development so multiple features can be worked on simultaneously without interfering.

---

## Part 2: What Makes It Interesting and Unique

### 2.1 Git-native to the core

Unlike Jira, Linear, or other project management tools, Kanbanzai stores all workflow state *in the repository itself*. There is no external database, no SaaS backend. State is YAML files under `.kbz/state/`, versioned in Git. This means:

- **Zero infrastructure** — no database to provision, no service to deploy (beyond the MCP server binary)
- **Complete transparency** — every state change is a git commit; `git log` is the audit trail
- **Repository-local portability** — clone the repo and you have the full workflow history
- **Git-native collaboration** — pull, push, merge state changes like code

### 2.2 Deterministic lifecycle state machines

Every entity type has a directed graph of valid transitions enforced by Go code (not LLM judgement). You cannot accidentally skip a stage. The state machine catches invalid transitions at the tool level before any mutation occurs.

### 2.3 Document intelligence with 3-layer classification

Documents aren't just files — they're parsed, indexed, and classified:

- **Layer 1**: Structure parsing — extracts sections with paths, titles, word counts
- **Layer 2**: Entity linking — finds entity ID references and builds a document graph
- **Layer 3**: Classification — assigns semantic roles (requirement, decision, rationale, constraint, assumption, risk) to sections via LLM analysis

This enables queries like "find all requirements that reference FEAT-123" or "show me all decisions about caching."

### 2.4 Self-hosting validation

Kanbanzai was developed *using Kanbanzai*. The orchestration tools that dispatch tasks and manage workflow were themselves built through that same workflow. This dogfooding validated the system's real-world fitness and exposed friction points early.

### 2.5 Evidence-based agent guidance

Rather than generic system prompts, Kanbanzai gives each sub-agent a **role-specific identity** with its own vocabulary, anti-patterns, and tool permissions. A security reviewer doesn't just get "review this code" — it gets the `reviewer-security` role with security-specific anti-patterns, the `review-plan` skill with criterion-by-criterion verification, and tool hints limiting it to the tools appropriate for review work.

### 2.6 MCP as a workflow boundary

By exposing workflow operations as MCP tools rather than file-system conventions, Kanbanzai creates an API contract. Tools validate input, enforce lifecycle rules, and produce structured output. The MCP protocol provides type-safe parameter schemas and error contracts.

---

## Part 3: How It Works — The Conceptual Model

### 3.1 The plan/batch/feature/task hierarchy

```
Plan (strategic)  ──▶  Batch (execution)  ──▶  Feature  ──▶  Tasks
P1                   B1                      FEAT-001     TASK-001
                                             FEAT-002     TASK-002
                      B2                      FEAT-003     TASK-003
```

- **Plans** are strategic — scope decomposition, long-term direction
- **Batches** are execution — groups of features for implementation
- **Features** are deliverable units with specs, dev-plans, and tasks
- **Tasks** are the smallest unit of work, dispatched to sub-agents

### 3.2 The workflow lifecycle

A feature's journey through the system:

```
idea → shaping → specifying → dev-planning → developing → reviewing → merging → verifying → done
```

At each gate, the system checks prerequisites:
- **specifying → dev-planning**: Is the spec approved?
- **dev-planning → developing**: Is the dev-plan approved? Are tasks created?
- **developing → reviewing**: Are all tasks done?
- **reviewing → merging**: Is the review report approved?
- **merging → verifying**: Did the merge succeed? Are tests passing?
- **verifying → done**: Did the close-out verifier pass?

### 3.3 The orchestrator model

The orchestrator is a chat-based AI agent that:
1. Reads `status()` to understand project state
2. Reads `next()` to see the ready task queue
3. Claims tasks via `next(id)`
4. Assembles sub-agent prompts via `handoff(task_id)`
5. Dispatches sub-agents via `spawn_agent()`
6. Completes tasks via `finish(task_id)`
7. Advances features through lifecycle states

This is a **human-in-the-loop** model by default — gates like spec approval and review sign-off require human confirmation via `checkpoint`. The `fast-track` system allows certain tiers of work to bypass human gates.

### 3.4 The context assembly pipeline

When `handoff` assembles a sub-agent prompt, it follows a deterministic pipeline:

1. Read stage binding → resolve role + skill
2. Load role YAML → identity, vocabulary, anti-patterns, tool constraints
3. Load skill markdown → procedure, checklist, examples
4. Surface knowledge entries tagged for the role and task domain
5. Add code graph context (for implementation roles)
6. Inject tool hints (which tools the sub-agent should use)
7. Attach spec sections and acceptance criteria
8. Include file paths and dependency information

The output is a complete prompt (~9K tokens) that gives the sub-agent everything it needs to execute the task without additional context gathering.

---

## Part 4: The Issues — What's Wrong and Why

### 4.1 Noise in git logs caused by state changes

**What happens:** Every MCP tool call that modifies `.kbz/state/` files (entity transitions, document registrations, task completions, knowledge contributions) produces a trail of modified YAML files. These accumulate as uncommitted changes and, when committed, produce verbose git history dominated by workflow churn rather than code changes.

**Evidence:** 121 `register` commits in 8 days (KE-01KQQ5981YMN0). The P60 review found `.kbz/index/documents/*.yaml` files being modified on read-only operations (access tracking counters). The health dashboard shows 10+ modified index files and 12+ untracked index files at the start of a typical session (KE-01KQPYAGTW4KT).

**Why it's a problem:**
- `git log` becomes useless for understanding code changes
- Git blame is polluted by non-code commits
- The signal-to-noise ratio drops dramatically as the project scales
- Agents use the dirty-tree state as a sign of incomplete work, creating a negative feedback loop

### 4.2 Difficulty getting AI agents to obey rules

**What happens:** The orchestrator and sub-agents consistently bypass rules encoded in skills, roles, and AGENTS.md. Despite explicit instructions like "Always use `handoff(task_id:)`. Never compose implementation prompts manually," agents do exactly that.

**Evidence:** The P44 orchestrator architecture research documents "every plan since P50 shows the orchestrator violating this rule." In P50, the orchestrator called `handoff` (technical compliance) but the output was unusable, so it discarded it and hand-wrote 12 prompts. In P57, all four implementation prompts were composed manually with no `handoff` call at all. Sub-agents were dispatched without role identity, vocabulary, anti-patterns, or tool hints — despite the pipeline assembling all of these correctly (KE-01KQTYHDKMKMF).

**Why it's a problem:**
- **This is a category error.** Skills, roles, and "Definition of Done" instructions are being used as *control mechanisms* when they are, by their nature, *advisory context*. LLMs are stochastic compliance engines with strong recency bias and degraded mid-context attention.
- The P44 research identifies the root cause: there is no component in the system whose *only job* is to enforce workflow invariants and who *cannot be talked out of doing it*.
- The 150+ gate overrides across the project (KE-01KR8JPS4GQ5E) show the gate system doesn't match real workflow patterns.

### 4.3 Random file names and paths

**What happens:** Document files are placed with inconsistent naming conventions. Filenames like `P41-prompt-design-review.md`, `design-kanbanzai-1.0.md`, `B3-F1-design-init-command.md`, and `spec-bootstrap-specification.md` coexist with no deterministic enforcement.

**Evidence:** KE-01KQQ0RBR7YAP documents that filename consistency is "currently enforced by agent discipline alone — there's no deterministic tool that generates the correct path." KE-01KQQ0MPDMYVV notes prompt files saved to ad-hoc locations with no defined convention.

**Why it's a problem:**
- Agents cannot predict where to find documents
- Document registration and discovery are fragile
- Human readers cannot navigate the `work/` directory
- The `doc path` tool was only recently added (P50) to compute canonical paths

### 4.4 Skills not being found or read

**What happens:** Agents fail to read the stage binding, role, and skill before starting work. This is supposed to happen at the start of every task, but the mechanism to enforce it is purely advisory.

**Evidence:** KE-01KQQWC4D7E1P documents that the April 29 plan/batch migration (commit 85f3e789) rewrote `kanbanzai-agents/SKILL.md`, collapsing the detailed 11-point task lifecycle checklist into a 4-point skeleton. The old checklist included critical pre-implementation steps: "Read the assembled context," "Called knowledge(action: list)," "Confirmed the parent feature is in the correct lifecycle state." The new checklist omits all of these. KE-01KQQWCTZM20Z documents that `next()` context assembly does not include a directive to read the stage binding or skill file.

The audit of commit 85f3e789 (KE-01KQQWRY47S0X) found 2 CRITICAL regressions across kanbanzai-agents and review-plan skills, plus 3 HIGH regressions in write-design.

**Why it's a problem:**
- Skills are the mechanism for quality and consistency
- If agents don't read them, they're dead weight consuming context
- The regressions in the skill files themselves mean even agents that *do* read them get degraded guidance
- There are 46 files controlling agent behavior with zero automated structural validation (KE-01KQQXA3K20Z6)

### 4.5 Orchestrators doing implementation work

**What happens:** The orchestrator — whose role is to coordinate, not implement — performs pre-delegation investigation, reading code, running grep, and "understanding" the implementation before delegating to sub-agents.

**Evidence:** The P44 research documents orchestrators using `read_file`, `grep`, `search_graph` to "understand" code before delegating, polluting their own context with the very implementation details delegation was meant to keep out. This is a direct violation of the orchestrator role's anti-patterns.

**Why it's a problem:**
- Accumulates context that causes goal drift over long sessions
- Duplicates work that sub-agents will redo in their own sessions
- Wastes tokens on investigation that should happen in sub-agent contexts
- Masks the real problem: if the pipeline produced correct context, the orchestrator wouldn't feel the need to investigate

### 4.6 Stale MCP binary

**What happens:** The running `kanbanzai` MCP server binary is frequently older than the code in the repository. Tool calls fail with errors about unknown states or type mismatches because the server predates the code changes.

**Evidence:** KE-01KMS0EE97M2P: "The stale MCP binary issue wasted significant verification time — 3 of 9 ACs appeared to fail when the code was correct." KE-01KQQWD5JHJWT: "The MCP server binary at GOBIN/kanbanzai was 3 days stale (April 30) while the kbz binary was freshly rebuilt (May 3). Root cause: the Makefile sets BINARY := kbz, so go install produces kbz not kanbanzai." KE-01KQYZQ3F43JC: "The kbz serve MCP server has no built-in restart mechanism."

**Why it's a problem:**
- Creates false test failures and misdiagnosis
- Requires human intervention to kill and restart the server
- The `kbz` vs `kanbanzai` binary naming mismatch means `make install` doesn't update the actual MCP server binary
- Agents diagnose their own correct code as broken because the server is stale

### 4.7 Cache staleness (state consistency)

**What happens:** Entity state shown in `status()` and `entity(list)` dashboards disagrees with the canonical YAML state. Tasks and features show as `done` via `entity(get)` but as `ready`, `active`, or `reviewing` via `entity(list)`.

**Evidence:** KE-01KQHQQHDHRVG: "finish() writes task status to the entity YAML store and SQLite cache, but entity(action: list) and the task queue (next()) read from a cached index that isn't reliably invalidated after write-through." Specific examples: TASK-01KQG2WT3GC31 shows status:done via get but status:ready via list. KE-01KR8JPS4JQS4: B43 showed as "reviewing" when the batch was actually "done."

**Why it's a problem:**
- The "source of truth" is ambiguous
- Agents claim tasks that are already done
- Health dashboards show phantom issues
- Override transitions leave "done-status detritus" — features show done in YAML but stale in cache

### 4.8 Worktrees being forgotten

**What happens:** Worktrees are created for features but never cleaned up after merge. Branches are merged to main, but worktree directories and tracking records persist indefinitely.

**Evidence:** KE-01KR1RHZ1GSEV: "12 of 14 active worktrees (May 7, 2026) are stale: 6 branches already merged to main but worktrees not cleaned up, 3 closed bugs never merged, 1 done feature never merged, 1 feature's branch deleted post-merge but worktree directory remains. Only 4 of 14 active worktrees have unmerged commits for genuinely in-progress work." KE-01KQFWMRPKK9D: "When features or bugs are advanced to done/closed via override, worktrees and branches are not automatically cleaned up. This resulted in 9 stale active worktrees accumulating."

**Why it's a problem:**
- Disk space bloat from stale worktree directories
- Misleading status dashboards showing phantom "in-progress" work
- Agents may accidentally work in stale worktrees
- Task state in worktrees can be lost if worktree is cleaned before state is merged (KE-01KR8JPS4GNKF)

### 4.9 Merges not happening / Definition of Done not enforced

**What happens:** Features and bugs reach `done` or `closed` status without actually being merged to main. The Definition of Done checklist (merge + worktree cleanup + test verification) is skipped.

**Evidence:** KE-01KQZJK3GSD53: "Tests are not being run after merges to main. Multiple test failures exist on main that would have been caught if tests were run as a merge gate." KE-01KR1RHZ1GSEV: The DoD checklist is "being skipped at the merge-and-cleanup stage." KE-01KQZC158H9ZN: "finish() called for all 13 tasks across 4 features in B1-p51-exec, but none produced a code commit. Only .kbz/state/ metadata was committed." KE-01KR8JPS4PRBK: "Batch B64 is marked 'done' but contains FEAT-01KR7BKXG3X61 (status: dev-planning, 2 queued tasks)."

The P60 review found that `HealthCheckCleanGate` is registered as a blocking merge gate but always passes — it's explicitly a placeholder (C2 finding).

**Why it's a problem:**
- Code changes exist only in dirty worktrees, never merged to main
- Features marked `done` that have never been integrated
- "Pre-existing failure" culture: tests are broken on main, new failures blend in
- The system claims completion for work that never passed review, never had a spec, and was sometimes implemented directly on `main`

### 4.10 Tests broken but ignored and not reported

**What happens:** Test failures accumulate on main and are normalized as "pre-existing." Agents dismiss them and continue working without filing bugs or blocking progress.

**Evidence:** KE-01KQZJK3GSPMW: "'Pre-existing' test failure is not a valid category. A failing test is a failing test — it means someone didn't run tests before merging." KE-01KQZRA52Y9G8: "Large failing test suites must be turned into durable workflow entities immediately." The P60 review found: `go vet ./...` fails, `go test ./internal/service` does not compile, `go test ./internal/kbzinit` has multiple failures. KE-01KQVRC4WPVJF documents pre-existing test compilation errors in internal/mcp that prevented running new tests. The P63 design identified 111 failing tests.

**Why it's a problem:**
- The test suite cannot act as a regression gate
- New failures are invisible because they blend into existing failures
- Release confidence is zero — you cannot know if a change breaks anything
- The culture of dismissing test failures erodes quality discipline

### 4.11 Difficulty testing SKILLs

**What happens:** There is no automated validation that skill files are well-formed, complete, or consistent. Skills are Markdown files with no structural enforcement.

**Evidence:** KE-01KQQXA3K20Z6: "46 files control agent behavior (skills, roles, stage bindings, agent instructions) with zero automated structural validation." The 85f3e789 regressions (KE-01KQQWRY47S0X) gutted critical checklist items and vocabulary terms without any test catching it. The P60 review found embedded skill seeds have drifted from project-local sources (C4 finding).

**Why it's a problem:**
- Skill regressions are silent — agents get degraded guidance with no alert
- Cross-reference integrity (roles referenced in stage-bindings that don't exist) is unchecked
- Embedded install seeds and project-local skills diverge
- The only way to test a skill change is to run a full task with it — which is prohibitively slow

### 4.12 Difficulty sharing progress without merges to remote

**What happens:** Workflow state in `.kbz/state/` is only visible to other agents after it is committed and pushed. In-progress state lives in the local repository and is invisible to collaborators.

**Evidence:** The centralized-state-server design document identifies "commit-bound visibility" as a core constraint: "Shared state is not visible to others until it is committed and pushed." The P44 research notes that "Git-native model is eventually shared through push/pull, not immediately shared through a central authority." KE-01KQTYHDKMKMF documents the handoff-vs-compaction mismatch: there's no standardised way to pass orchestrator progress between sessions.

**Why it's a problem:**
- Multiple agents cannot see each other's in-flight state
- Session-to-session handoff requires manual progress documents
- No real-time coordination between parallel agents
- Viewers (read-only agents) see stale state

### 4.13 Context rot in the orchestrator

**What happens:** The orchestrator's chat context grows over a multi-hour session as it accumulates tool outputs, reasoning chains, and sub-agent summaries. Goal drift sets in — the orchestrator forgets the original constraints and follows whatever instruction is most recent in the context window.

**Evidence:** The P41 context pollution research provides extensive documentation. P50 incident: orchestrator drifted from fast-track purpose (no human gates) and stopped for confirmation despite explicit prohibition. The orchestrator's identity and DoD constraints sit in the *middle* of an ever-growing window of code reads, tool outputs, and prior task summaries. Drift is not a model defect — it is the *predicted behaviour* given the U-shaped attention curve documented by Liu et al. (2024).

**Why it's a problem:**
- The orchestrator's decisions become less reliable over time
- Fast-track automation is defeated because the agent self-interrupts
- Each `next()` call returns ~30KB of repeated context — 12 tasks = 360KB of redundancy
- The 60% context utilisation offload threshold was calibrated for 128K-200K windows, not the 1M windows now available (KE-01KQTXXRSCB9A)

### 4.14 Gate overrides are systemic

**What happens:** The system has accumulated ~150+ gate overrides. While many are legitimate state-repair operations, the volume and variety suggest the gate system doesn't match real workflow patterns.

**Evidence:** KE-01KR8JPS4GQ5E documents common override categories: state repair for pre-lifecycle work, batch-level review substituting for per-feature review, documentation-only features skipping gates, cache staleness forcing overrides, spec/doc ownership mismatches. KE-01KR8JPS4GHCX: "Gate overrides on reviewing→done do not validate that all child tasks are terminal."

**Why it's a problem:**
- Overrides bypass the safety mechanisms that are Kanbanzai's core value proposition
- Features can reach `done` with incomplete tasks
- The normalization of overrides erodes trust in the workflow
- If gates are routinely bypassed, they're not gates — they're suggestions

### 4.15 Main branch leakage

**What happens:** Files accumulate on main through three mechanisms: MCP server commits lifecycle transitions to main, the "commit WIP before review" anti-pattern, and direct edits on main for batches/plans that have no worktree support.

**Evidence:** KE-01KQQ5981YMN0: "121 register commits in 8 days" on main. KE-01KQZRTESNDDF: "Uncommitted production code changes on main branch are causing 44 test failures." The P47/B46 incident (KE-01KQQCAWVM89V): "implementation (5 commits) was done directly on main with no worktree, no feature entities, and no task tracking."

**Why it's a problem:**
- Main becomes unstable with uncommitted or untested changes
- Worktree isolation is undermined
- Parallel work on main causes conflicts
- The integrity of the default branch is compromised

### 4.16 edit_file reliability

**What happens:** `kanbanzai_edit_file` and `edit_file` tools report successful edits that don't persist to disk, particularly for test files in worktrees.

**Evidence:** KE-01KR48Q446EWE: "kanbanzai_edit_file reported edits_applied:1 for both loader_test.go edits but neither persisted to disk." KE-01KR66EC6D3F3: "kanbanzai_edit_file did not persist for some test files; Python replacement was needed as fallback." KE-01KQ9QCVQ58K7: "Edit_file tool diffs may not persist. Verify changes with grep after edits."

**Why it's a problem:**
- Agents believe they've made changes that don't exist
- Requires wasteful verification steps after every edit
- Forces fallback to terminal-based Python/heredoc workarounds
- The tool is the primary file-writing mechanism — unreliability undermines the whole pipeline

### 4.17 Other significant issues

- **Plan numbering reuse (P1 instead of P51/P52)**: KE-01KQVXXSKFJ1Z. Generated project IDs reuse `P1` instead of incrementing.
- **Done-status detritus**: KE-01KQHQQHDHRVG. Tasks/features show as `done` via entity(get) but as `ready`/`active`/`reviewing` via list/status.
- **No feature reopen path**: KE-01KQPYMG0ZQA7. Features cannot transition back from `done` to a working state.
- **Status dashboard misclassifies queued as ready**: KE-01KR8JPS4HNFX. Creates false "work available" signals.
- **Batch done with incomplete features**: KE-01KR8JPS4PRBK. B64 marked `done` with a feature still in `dev-planning`.
- **Bug lifecycle requires 7-9 sequential transition calls**: KE-01KQF86XCCVEG. Closing a bug takes 7 separate MCP tool calls.
- **No MCP server restart**: KE-01KQYZQ3F43JC. After building, the human must manually kill and restart.
- **Worktree creation timeouts at scale**: KE-01KPXCW22CFZ7. 34+ simultaneous worktrees cause timeouts.

---

## Part 5: Why Does It Work This Way?

### 5.1 Historical evolution

Kanbanzai evolved through distinct phases, each adding a layer to the architecture:

- **Phase 1**: Git-native entity storage with YAML files and TSID13 IDs. The core insight: "Git is the transport."
- **Phase 2**: MCP server surface with consolidated action-based tools. The 97 legacy tool names were collapsed to 22.
- **Phase 3**: Skills, roles, and stage bindings. The evidence-based guidance system.
- **Phase 4**: Orchestration — task dispatch, worktrees, merge gates, health checks.
- **Beyond**: Fast-track automation, document intelligence, plan/batch distinction, context assembly pipeline.

Each phase built on the previous without fundamentally rethinking the architecture. The chat-based orchestrator was the natural first approach — simple, flexible, and human-compatible. The file-backed state model was the natural choice for a Git-native tool.

### 5.2 Design philosophy tensions

The system has three design philosophies that are increasingly in tension:

1. **Git-native transparency**: State in files, Git as transport. Simple, portable, inspectable.
2. **LLM-driven flexibility**: The orchestrator is an AI agent that can handle edge cases dynamically.
3. **Deterministic enforcement**: Lifecycle state machines, gate checks, document approval requirements.

The tension: #2 (LLM flexibility) and #3 (deterministic enforcement) are fundamentally in conflict. You cannot have an AI agent flexibly interpret rules *and* deterministically enforce them. The system has been resolving this tension by adding more rules, more anti-patterns, more skill text — but these are all #2-style solutions that cannot achieve #3-style outcomes.

### 5.3 The architectural root cause

The P44 orchestrator architecture research identifies the central defect:

> There is no component in the system whose *only job* is to enforce workflow invariants and who *cannot be talked out of doing it*.

The orchestrator is the dispatcher *and* the rule-follower *and* the reviewer of its own outputs. Every mechanism intended to constrain it (skills, anti-patterns, checklist items) is advisory context that an LLM can and does ignore.

The fix is not more rules. It is an architectural change: move enforcement from the LLM layer (advisory context) to the server layer (deterministic code).

---

## Part 6: What Would We Do Differently From a Clean Slate?

### 6.1 What to throw away

| Component | Reason |
|-----------|--------|
| **Chat-based orchestrator as controller** | The fundamental category error: LLMs cannot enforce procedural invariants. The orchestrator should be a *supervisor*, not a *controller*. |
| **File-backed canonical state as the only mode** | Git-native is elegant but doesn't scale to team coordination. A centralized state server (PostgreSQL) should be the canonical backend, with Git-native as an optional mode. |
| **Three-step dispatch (next → handoff → spawn_agent)** | Each step is a decision point where the orchestrator can bypass the pipeline. One call should do it all. |
| **Skills as markdown prose** | Skills are content, not structure. They should be structured data (YAML/JSON) that the pipeline can validate, not prose that agents can skip. |
| **Stage bindings as a single YAML file** | Should be generated from a registry of roles and skills, not maintained manually. |
| **Manual commit discipline for state changes** | State should be auto-committed atomically with the tool call that produced it. Agents should not need to remember to commit. |
| **Access tracking in versioned YAML** | Read operations should never dirty the working tree. Access tracking belongs in a non-versioned cache or SQLite table. |
| **The spawn_agent direct path** | Should not exist for the orchestrator. Dispatch must go through a non-bypassable pipeline. |

### 6.2 What to keep

| Component | Reason |
|-----------|--------|
| **MCP as the tool surface** | Well-designed, type-safe, action-based. The 22-tool consolidation is clean. |
| **Entity lifecycle state machines** | The deterministic transition enforcement works. It just needs to be extended to the dispatch layer. |
| **TSID13 ID system** | Time-sortable, human-prefixed, collision-resistant. Simple and effective. |
| **Worktree isolation for parallel development** | Correct pattern. Needs better cleanup automation and enforcement. |
| **Document intelligence (3-layer)** | Structure parsing, entity linking, and classification are valuable. Should be more tightly integrated. |
| **Knowledge base with retro signals** | Accumulated project learning is a differentiator. Needs better auto-surfacing. |
| **Context assembly pipeline (3.0)** | The deterministic role+skill+knowledge assembly is correct. It produces good output — the problem is it's bypassable. |
| **Stage binding concept** | The mapping of stages → roles → skills → documents is the right abstraction. The content needs structured validation. |
| **Health checks and status dashboards** | Valuable for project awareness. Need cache consistency fixes. |
| **The thin-adapter MCP pattern** | MCP handlers as parameter extractors delegating to service layer. Clean separation. |
| **Sub-agent per task with clean context** | The right model. Each implementer, reviewer, verifier gets fresh context scoped to their task. |

### 6.3 Proposed target architecture

```
┌──────────────────────────────────────────────────┐
│          Human / Chat Supervisor                  │
│   (entity transitions, exceptions, strategy)     │
└────────────────────┬─────────────────────────────┘
                     │
                     ▼
┌──────────────────────────────────────────────────┐
│         MCP Server — Deterministic Controller    │
│                                                  │
│  ┌──────────┐  ┌──────────┐  ┌───────────────┐  │
│  │  Policy  │  │  Stage   │  │   Dispatch    │  │
│  │  Engine  │  │Controllers│  │   Pipeline   │  │
│  └────┬─────┘  └────┬─────┘  └───────┬───────┘  │
│       │             │               │            │
│  ┌────▼─────────────▼───────────────▼────────┐   │
│  │     PostgreSQL (canonical state)          │   │
│  │  entities, transitions, docs, knowledge   │   │
│  └───────────────────────────────────────────┘   │
│                                                  │
│  ┌──────────────────────────────────────────┐    │
│  │   Audit Log  │  Eval Harness  │  Health  │    │
│  └──────────────────────────────────────────┘    │
└────────────────────┬─────────────────────────────┘
                     │ provider API calls
                     ▼
┌──────────────────────────────────────────────────┐
│     Sub-agents (fresh sessions per task)         │
│  implementer · reviewer panel · verifier         │
└──────────────────────────────────────────────────┘
```

**Key architectural changes:**

1. **Deterministic controller replaces chat orchestrator.** The MCP server owns dispatch, gate enforcement, and DoD verification. The human/supervisor makes strategic decisions; the controller makes sure they're executed correctly.

2. **PostgreSQL canonical state with Git-native optional.** The database is the source of truth. Git-native file storage is a supported mode for small/solo projects. One canonical backend per project.

3. **Non-bypassable dispatch pipeline.** `dispatch_task(task_id, tier)` is the *only* path to a sub-agent. The orchestrator cannot compose prompts manually because `spawn_agent` is not in its tool list.

4. **Structured skills and roles.** Skills and roles are validated data (YAML with JSON Schema), not advisory prose. The pipeline hard-fails on missing or invalid role/skill definitions.

5. **Deterministic + LLM verifier.** DoD checks that can be deterministic (git status, ancestry, test exit codes, document existence) *are* deterministic. The LLM verifier handles only items requiring natural-language judgement. Both must pass.

6. **Continuous evaluation harness.** Golden tasks run on every pipeline change. Output diffs are checked in. Regressions block release.

7. **Auto-committed state.** Every MCP tool call that modifies state atomically commits its changes. Agents never see "uncommitted state changes" in git status.

8. **Real-time shared state.** With a centralized database, state changes are immediately visible to all agents. No commit-push-pull cycle for workflow coordination.

---

## Part 7: Migration Considerations

### 7.1 What we'd lose

- **Zero-infrastructure simplicity.** A PostgreSQL backend means standing up and maintaining a database. The current "clone and go" model is genuinely valuable for solo developers and small projects.
- **Git-native transparency.** With state in a database, `git log` no longer tells the full story. Audit history requires querying the database.
- **Offline operation.** A centralized server is a network dependency. Git-native mode works entirely locally.
- **The chat orchestrator's flexibility.** A deterministic controller can't handle edge cases dynamically. The human supervisor must handle exceptions.

### 7.2 Incremental migration path

The centralized-state-server design outlines a 6-stage migration that preserves backward compatibility:

1. **Isolate persistence behind interfaces** — define backend-neutral service contracts
2. **Define canonical database schema** — PostgreSQL schema preserving current semantics
3. **Implement centralized backend in parallel with file backend** — dual-mode operation
4. **Add migration tooling** — import/export between file and database backends
5. **Adapt surrounding features** — viewer model, worktree checks, backup/DR
6. **Decide product positioning** — which mode is primary, which is legacy

### 7.3 What can be done immediately

The P44 research identifies actions that don't require architectural change:

1. **Remove `spawn_agent` from the orchestrator's tool list** (or wrap it so output goes through the assembly gate). This single change eliminates the dominant failure mode.
2. **Ship the Prompt Assembly Gate (P44-F1)** — hard-fail on missing role/skill; warn on missing tool hints.
3. **Land bug lifecycle gate enforcement** — `CheckBugTransitionGate`, review-report requirement.
4. **Build a tiny evaluation harness** — 10 golden tasks, diff output on pipeline changes, fail CI on diff.
5. **Make the verifier two-layered** — deterministic Go checks plus LLM judgement for ambiguous items.
6. **Fix the `kbz` vs `kanbanzai` binary naming** — eliminate stale-binary issues.

---

## Part 8: Summary Verdict

Kanbanzai MCP is a **well-conceived but architecturally strained** system. Its core ideas — Git-native state, lifecycle state machines, evidence-based agent guidance, sub-agent dispatch, document intelligence — are sound and forward-looking.

Its problems stem from a single architectural tension: **it asks LLMs to enforce rules that only deterministic code can enforce**. The chat-based orchestrator is being asked to be both the flexible decision-maker and the rule-enforcer, and LLMs cannot do the latter reliably.

The fix is not to abandon the system but to **move enforcement from the prompt layer to the server layer**. The MCP server should own dispatch, gate enforcement, and Definition of Done verification. The human/supervisor should own strategic decisions. The LLM should own what it's good at: implementation, review, and judgement.

The P44 research is clear: **"Proceed with the MCP-server-as-orchestrator architecture. It is the correct architecture and is well-aligned with current evidence and industry consensus."**

The question is not *whether* to change the architecture but *how fast* and *whether to preserve backward compatibility* with the Git-native, chat-orchestrated model during the transition.
