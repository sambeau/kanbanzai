# Orchestration and Knowledge

Kanbanzai coordinates AI agents through structured context assembly, role-based identity, and persistent knowledge. This document covers the mechanics — how agents receive instructions, claim tasks, work in parallel, and contribute knowledge that compounds across sessions.

**Audience.** Agentic developers and power users who are familiar with tool calling, context windows, system prompts, and multi-agent coordination.

**Prerequisites.** Read the [User Guide](user-guide.md) for system orientation and the [Workflow Overview](workflow-overview.md) for lifecycle stages and gates.

---

## The coordination problem

Without structured orchestration, AI agents working on a codebase encounter four recurring failures:

1. **Context loss.** Each agent session starts with an empty context window. Lessons learned in previous sessions — which patterns work, which approaches failed, which files are relevant — vanish unless something preserves them.

2. **Conflicting edits.** Two agents editing the same file produce merge conflicts. Without advance warning, these conflicts surface only at merge time, after both agents have completed their work.

3. **Repeated rediscovery.** Agent A discovers that the caching layer uses a specific invalidation pattern. Agent B, working on a related feature in a different session, spends time rediscovering the same pattern because nothing carried the observation forward.

4. **Inconsistent conventions.** Without shared vocabulary and constraints, agents produce code in different styles, use different error handling patterns, and make different architectural assumptions — even when working on the same feature.

Kanbanzai addresses these failures through five mechanisms: roles constrain agent identity, skills define procedures, context assembly delivers scoped information, knowledge entries persist observations, and conflict analysis flags overlapping work before it starts.

---

## Roles and skills

Roles and skills are the two halves of agent instruction. A role defines *who the agent is* — its identity, vocabulary, and constraints. A skill defines *what the agent does* — its procedure, checklist, and evaluation criteria. Stage bindings connect them: each lifecycle stage maps to a role and a skill.

### Roles

Role definitions live in `.kbz/roles/` as YAML files. Each role contains:

| Field | Purpose |
|-------|---------|
| `id` | Role identifier (e.g. `orchestrator`, `implementer-go`) |
| `inherits` | Parent role for field inheritance (e.g. `implementer-go` inherits from `implementer`, which inherits from `base`) |
| `identity` | One-line persona the agent adopts |
| `vocabulary` | Domain terms the agent uses, each as a `"term — definition"` pair |
| `anti_patterns` | Failure modes to avoid, each with `name`, `detect`, `because`, and `resolve` fields |
| `tools` | MCP tool names the agent is permitted to call |

Roles use inheritance. The `base` role defines shared vocabulary and constraints. Language-specific roles like `implementer-go` inherit from `implementer`, which inherits from `base`. Inherited fields are merged — child vocabulary extends parent vocabulary, and child anti-patterns add to (not replace) parent anti-patterns.

The orchestrator role carries vocabulary specific to coordination: the 45% context utilisation threshold (do not load more than 45% of the context window before dispatching sub-agents), agent saturation at 4 (maximum concurrent sub-agents, based on diminishing returns beyond this point), and the cascade pattern (re-evaluate all downstream dependents when a task fails).

### Skills

Skill definitions live in `.kbz/skills/` directories, each containing a `SKILL.md` file. A skill file contains:

| Section | Purpose |
|---------|---------|
| **Frontmatter** | Metadata: `name`, `description` (expert and natural-language), `triggers` (activation phrases), `roles` (compatible role IDs), `stage`, `constraint_level` |
| **Vocabulary** | Skill-specific terms as `**term** — definition` pairs |
| **Anti-Patterns** | Failure modes in the same `detect`/`because`/`resolve` format as roles |
| **Checklist** | Copyable checkbox list for progress tracking |
| **Procedure** | Phased steps (e.g. Phase 1: Read Spec → Phase 2: Implement → Phase 3: Test → Phase 4: Verify) |
| **Examples** | BAD/GOOD pairs with explanations |
| **Evaluation Criteria** | Numbered criteria with weights (`required`, `high`, `moderate`) |

Skills enforce consistency. Two agents running the `implement-task` skill in different sessions follow the same procedure, use the same checklist, and are evaluated against the same criteria — regardless of which model powers them.

### Stage bindings

The file `.kbz/stage-bindings.yaml` maps each lifecycle stage to its role, skill, orchestration pattern, and prerequisites. When an agent enters a stage, it reads the binding to determine what role to adopt and which skill procedure to follow.

Each binding specifies:

- **Orchestration pattern** — `single-agent` (one agent does the work) or `orchestrator-workers` (an orchestrator dispatches sub-agents).
- **Roles and skills** — which role file to read and which skill procedure to follow.
- **Document type** — what kind of document this stage produces (design, specification, dev-plan, report).
- **Effort budget** — expected tool call range (e.g. "10–50 tool calls per task" for developing).
- **Prerequisites** — documents that must be approved or tasks that must exist before the stage begins.

The `developing` stage, for example, binds to the `orchestrator` role and the `orchestrate-development` skill. It uses the `orchestrator-workers` pattern with `implementer` sub-agents running the `implement-task` skill in parallel.

---

## Context assembly

When an agent claims a task or receives a handoff, the system assembles a **context packet** — a structured bundle of instructions, specification fragments, knowledge entries, and file paths scoped to the work at hand. Two tools produce context packets: `next` (which also claims the task) and `handoff` (which prepares a prompt for a sub-agent).

### What a context packet contains

| Component | Source | Purpose |
|-----------|--------|---------|
| **Role constraints** | Role profile (`.kbz/roles/`) | Identity, vocabulary, commit format conventions |
| **Stage-aware guidance** | Stage binding | Orchestration pattern, effort budget, tool subset, output convention |
| **Specification sections** | Document intelligence index | Relevant spec fragments traced to the task |
| **Acceptance criteria** | Extracted from spec sections | Testable conditions the implementation must satisfy |
| **Knowledge entries** | Knowledge store (`.kbz/state/knowledge/`) | Project-level and session-level observations relevant to the work |
| **File paths** | Task's `files_planned` field | Files the task is expected to create or modify |
| **Tool hints** | Role profile, resolved via inheritance | Which tools are primary, which are excluded |
| **Graph project** | Worktree record | Project name for code knowledge graph queries |
| **Worktree path** | Worktree record | Filesystem path for the isolated working directory |
| **Experiment nudges** | Active workflow experiments | Decisions from running experiments that affect agent behaviour |

### Budget trimming

Context packets have a byte budget (default: 30 KB). When the assembled content exceeds this budget, the system trims lower-priority entries to fit. Trimmed items are recorded with their type, topic, and size so the agent knows what was removed. Knowledge entries and specification sections are the primary candidates for trimming; role constraints and stage guidance are never trimmed.

### The handoff prompt

The `handoff` tool renders the context packet into a Markdown prompt suitable for a sub-agent. The prompt is assembled in a fixed order:

1. Conventions — role constraints and commit format
2. Stage-aware sections — orchestration, effort budget, tools, output convention
3. Task summary
4. Specification sections (from document intelligence, or a fallback path to the raw spec file)
5. Acceptance criteria
6. Known constraints from the knowledge base
7. File paths (excluded during designing and specifying stages, where files are not yet determined)
8. Active workflow experiments
9. Available tools — role-scoped tool hints
10. Code graph instructions — `search_graph`, `trace_call_path`, and `query_graph` with the project name
11. Additional instructions from the orchestrator

The `handoff` tool also commits `.kbz/state/` to disk before returning, ensuring that sub-agents spawned immediately after do not encounter uncommitted workflow state.

### Re-review injection

When a feature is in `reviewing` status with `review_cycle` ≥ 2 (meaning a previous review found issues and the feature was reworked), the handoff prompt includes focused re-review guidance directing the agent to concentrate on the areas flagged in the previous cycle.

---

## Task dispatch and orchestration

The orchestrator manages the work queue, claims tasks, and dispatches sub-agents. The process follows a claim-then-delegate pattern: the orchestrator inspects the queue, selects a task, claims it, and either performs the work directly or hands it off to a sub-agent.

### The work queue

The `next` tool operates in two modes:

**Queue inspection** (no `id` parameter) returns all ready tasks sorted by priority. The sort order is:

1. Estimate ascending — smaller tasks first (nil estimates sort last)
2. Age descending — older tasks first among equal estimates
3. Task ID lexicographic — deterministic tiebreaker

Queue inspection also triggers **promotion**: the system checks all `queued` tasks, validates their dependencies, and promotes eligible tasks to `ready`. A task is eligible for promotion when every task ID in its `depends_on` list has reached `done` status. Promoted tasks appear as side effects in the response.

When called with `conflict_check: true`, queue inspection annotates each ready task with conflict risk against all currently active tasks. This shows the orchestrator which tasks are safe to dispatch in parallel and which should be serialised.

**Task claiming** (with `id` parameter) transitions a `ready` task to `active` and sets dispatch metadata:

- `dispatched_to` — the role the agent is adopting
- `dispatched_by` — the caller's identity
- `claimed_at` — timestamp

The `id` parameter accepts a task ID (claims that specific task), a feature ID (claims the top ready task in that feature), or a plan ID (claims the top ready task across all features in that plan).

### Dependency management

Tasks declare dependencies through the `depends_on` field — an array of task IDs that must complete before this task can start. The system enforces dependencies through status transitions:

1. Tasks with unmet dependencies stay in `queued` status.
2. When a task completes, a `StatusTransitionHook` fires and checks all tasks that depend on it.
3. Tasks whose dependencies are now fully satisfied are promoted from `queued` to `ready`.
4. The work queue surfaces only `ready` tasks — agents never see tasks they cannot start.

This automatic promotion means the orchestrator does not need to manually track which tasks are unblocked. Completing a task automatically makes its dependents available.

### Dispatching sub-agents

The orchestrator dispatches work by:

1. Calling `next(id)` to claim a task and receive structured context.
2. Calling `handoff(task_id)` to render the context into a sub-agent prompt.
3. Spawning a sub-agent with the rendered prompt.
4. Monitoring the sub-agent's output and recording completion via `finish`.

The orchestrator tracks active sub-agents and their file scopes. The agent saturation limit of 4 concurrent sub-agents applies — beyond this, returns diminish and coordination overhead increases.

---

## Conflict awareness

Before dispatching tasks in parallel, the system checks whether they risk editing the same files. The `conflict` tool accepts two or more task IDs and returns a per-pair risk assessment.

### Three analysis dimensions

**File overlap** compares the `files_planned` lists between tasks. It checks three levels of proximity:

| Condition | Risk level |
|-----------|------------|
| No overlap in files or directories | `none` |
| Tasks touch files in the same directory | `low` |
| Tasks share one or more planned files | `medium` |
| Actual git-level merge conflicts detected | `high` |

When branch information is available, the system also runs `git merge-tree` to detect real merge conflicts, not just planned file overlap.

**Dependency order** checks whether one task depends on another, directly or transitively:

| Condition | Risk level |
|-----------|------------|
| Task A's `depends_on` includes Task B | `high` (direct dependency) |
| Task B is reachable from Task A via the dependency graph (BFS) | `medium` (transitive dependency) |
| No dependency path exists | `none` |

**Boundary crossing** analyses task metadata (slug, summary, feature slug, spec keywords) for semantic overlap. It tokenises the text, filters to tokens of three or more characters, and computes overlap:

| Condition | Risk level |
|-----------|------------|
| Three or more shared keywords | `low` |
| Fewer than three shared keywords | `none` |

### Risk recommendations

The system combines the three dimensions into a per-pair recommendation:

| Condition | Recommendation |
|-----------|---------------|
| Dependency risk present | `serialise` |
| Overall risk ≥ medium | `checkpoint_required` |
| Otherwise | `safe_to_parallelise` |

The queue-level conflict check (`next` with `conflict_check: true`) runs this analysis automatically for each ready task against all active tasks, annotating each queue item with `conflict_risk` and `conflict_with` fields.

---

## The knowledge system

Knowledge entries capture observations that persist across sessions. When an agent discovers a useful pattern, encounters a limitation, or makes a decision, it records the observation as a knowledge entry. Future agents working on related tasks receive these entries in their context packets.

### Entry structure

Each knowledge entry contains:

| Field | Description |
|-------|-------------|
| `id` | Unique identifier (`KE-{TSID}`) |
| `topic` | Normalised topic identifier (e.g. `worktree-file-editing-pattern`) |
| `content` | The knowledge statement — concise and actionable |
| `scope` | A role name (e.g. `implementer-go`) or `project` for project-wide knowledge |
| `tier` | `2` (project-level, 90-day TTL) or `3` (session-level, 30-day TTL) |
| `status` | Lifecycle status: `contributed`, `confirmed`, `disputed`, `stale`, `retired` |
| `confidence` | Score from 0.0 to 1.0 (starts at 0.5) |
| `use_count` | Times this entry was included in a context packet |
| `miss_count` | Times this entry was available but not included |
| `tags` | Classification tags |
| `created_by` | Who contributed the entry |
| `learned_from` | Provenance — typically a task ID |
| `ttl_days` | Time to live (30 days for tier 3, 90 days for tier 2) |

### Lifecycle

Knowledge entries move through five statuses:

```
contributed → confirmed
     ↓            ↓
  disputed → ... → retired
     ↓
   stale → retired
```

- **Contributed** — newly created, unverified. This is the default status when an entry is recorded via `finish` or `knowledge(action: "contribute")`.
- **Confirmed** — validated by usage or human review. Confirmed entries rank higher in context assembly.
- **Disputed** — flagged as potentially incorrect. Requires a reason. Disputed entries are still surfaced but with lower priority.
- **Stale** — content may be outdated. The staleness check compares git anchors against the current repository state to detect when referenced code has changed.
- **Retired** — removed from active use. Retired entries are excluded from context assembly.

### How entries are contributed

The primary contribution path is through `finish` — when an agent completes a task, it includes knowledge entries alongside the task summary:

```
finish(
  task_id: "TASK-...",
  summary: "Implemented caching layer",
  knowledge: [
    {
      topic: "cache-invalidation-pattern",
      content: "The caching layer uses TTL-based invalidation with...",
      scope: "project",
      tags: ["caching", "patterns"]
    }
  ]
)
```

Entries can also be contributed directly via `knowledge(action: "contribute")` outside a task flow. The system prevents exact-topic duplicates within the same scope, and rejects near-duplicates where content similarity (Jaccard coefficient) exceeds 0.65.

### How entries are surfaced

During context assembly, the system searches for knowledge entries relevant to the task being claimed. Matching entries are included in the context packet, subject to the byte budget. Each inclusion increments the entry's `use_count`; each skip increments `miss_count`. Over time, frequently used entries rise in ranking while unused entries become candidates for pruning.

### Promotion and pruning

**Promotion** moves a tier-3 (session-level) entry to tier-2 (project-level), extending its TTL from 30 to 90 days. The default promotion thresholds are:

| Criterion | Threshold |
|-----------|-----------|
| Minimum use count | 5 |
| Maximum miss count | 0 |
| Minimum confidence | 0.7 |

**Pruning** removes expired entries whose TTL has elapsed. A grace period of 7 days applies after expiry before the entry is removed. Pruning supports dry-run mode to preview what would be removed.

**Compaction** merges near-duplicate entries within a scope. It identifies entries with similar content (Jaccard similarity) and produces merge candidates. This keeps the knowledge base focused and prevents dilution from slightly different phrasings of the same observation.

### Conflict resolution

When two entries cover the same topic with conflicting content, the `resolve` action keeps one and retires the other. The `merge_content` option merges usage counts and git anchors from the retired entry into the kept entry, preserving usage history.

---

## Knowledge governance

The knowledge base is one of several places where project information lives. Each type of record serves a different purpose, and using the wrong one causes confusion:

| Record type | Purpose | Lifecycle |
|-------------|---------|-----------|
| **Knowledge entry** | Reusable observation learned from experience | contributed → confirmed → retired; surfaced automatically in context assembly |
| **Decision** | Architectural or process choice with rationale | Created via `entity(type: "decision")`; permanent record |
| **Root cause analysis** | Why a production incident occurred | Linked to an incident; required before incident closure |
| **Specification** | Formal requirements with acceptance criteria | Registered as a document; approved before implementation |
| **Team convention** | Shared practice encoded in a role or skill file | Lives in `.kbz/roles/` or `.kbz/skills/`; version-controlled |

The distinction matters because each type has different governance:

- Knowledge entries have TTLs and can be retired. They suit observations that may become stale — "the caching layer uses pattern X" is useful until someone changes the caching layer.
- Decisions are permanent and carry rationale. They suit choices that should not be revisited without good reason — "we chose SQLite over PostgreSQL because..."
- Conventions are codified in role and skill files. They suit practices that every agent should follow — error handling patterns, commit message format, test structure.

Contributing a decision as a knowledge entry means it will eventually expire. Recording a temporary observation as a decision means the decision log fills with transient information. The right record type preserves the right lifecycle.

---

## Merge gates

Before merging a feature branch into main, the system runs a series of gate checks. Gates verify that the work is complete, verified, and conflict-free. The `merge` tool runs these checks via `merge(action: "check")` and performs the merge via `merge(action: "execute")`.

### Gate checks

Seven gates run in order:

| Gate | Severity | What it checks |
|------|----------|----------------|
| `entity_done` | **Blocking** | Feature status is `done` (or bug status is `closed`) |
| `tasks_complete` | **Blocking** | All child tasks are `done` or `wont_do` |
| `verification_exists` | **Blocking** | The entity has a non-empty verification field |
| `verification_passed` | **Blocking** | The verification status is `passed` |
| `no_conflicts` | **Blocking** | The branch has no merge conflicts with main (checked via `git merge-tree`) |
| `health_check_clean` | **Blocking** | Project health check passes |
| `branch_not_stale` | **Warning** | The branch is not significantly behind main |

### Overall status

The overall merge status is determined by the strictest failing gate:

- Any **blocking** gate with `failed` status → overall `blocked`
- Any gate with `warning` status → overall `warnings`
- All gates passed → overall `passed`

### Merge strategies

Three merge strategies are available:

| Strategy | Behaviour |
|----------|-----------|
| `squash` (default) | Combines all branch commits into a single commit on main |
| `merge` | Creates a merge commit preserving branch history |
| `rebase` | Replays branch commits on top of main |

### Overrides

When a blocking gate fails but the merge is justified, `merge(action: "execute")` accepts `override: true` with a required `override_reason`. The override is logged permanently on the entity record. This mechanism exists for cases where a gate failure is understood and accepted — for example, merging a documentation-only change where `verification_passed` is not applicable.

### Post-merge cleanup

After a successful merge, the system:

1. Marks the worktree record as `merged` and sets `merged_at`.
2. Calculates `cleanup_after` based on a grace period (default: 7 days after merge).
3. When the grace period expires, cleanup removes the git worktree, deletes the local and remote branches, and removes the tracking record.

Abandoned worktrees (marked via `Record.MarkAbandoned`) skip the grace period and are eligible for immediate cleanup.

---

## Concurrency model

Multiple agents work on the same codebase simultaneously through three layers: worktrees for filesystem isolation, branches for version control isolation, and conflict analysis for coordination.

### Worktrees

Each feature or bug gets its own Git worktree — a separate working directory with its own branch, created via the `worktree` tool. Worktree records track:

| Field | Description |
|-------|-------------|
| `id` | Unique identifier (`WT-{ULID}`) |
| `entity_id` | The feature or bug this worktree serves |
| `branch` | Git branch name |
| `path` | Filesystem path relative to the repository root |
| `status` | `active`, `merged`, or `abandoned` |
| `graph_project` | Code knowledge graph project name for this worktree |
| `created_by` | Who created the worktree |

Worktrees provide hard isolation. Agent A working in `.worktrees/feat-caching/` and Agent B working in `.worktrees/feat-auth/` cannot interfere with each other's uncommitted changes. Their commits stay on separate branches until merge.

### Branch hygiene

The `branch` tool checks worktree branch health before merging or resuming work. It reports:

- **Staleness** — how far behind main the branch has drifted.
- **Drift** — commits on main that are not on the branch.
- **Conflicts** — whether merging the branch into main would produce conflicts.

This check runs as part of the merge gate (`branch_not_stale`) and can be called independently to assess whether a branch needs rebasing before work continues.

### Parallel execution pattern

The typical pattern for parallel work is:

1. **Decompose** — break a feature into tasks with explicit `depends_on` declarations.
2. **Check conflicts** — run `conflict(action: "check")` on task pairs that might run in parallel.
3. **Dispatch safe pairs** — tasks rated `safe_to_parallelise` are dispatched to concurrent sub-agents.
4. **Serialise risky pairs** — tasks rated `serialise` or `checkpoint_required` run sequentially, or the orchestrator creates a checkpoint for human decision.
5. **Monitor and complete** — the orchestrator monitors sub-agent outputs, records completions via `finish`, and lets automatic dependency promotion unblock the next wave of tasks.

### File editing in worktrees

Standard file editing tools (`edit_file`, `create_directory`) operate on the main project root, not on worktree directories. Agents working in a worktree use terminal commands (typically Python-based file writes) to modify files in the worktree path. This is a known limitation — the context packet includes the worktree path so agents know where to write.

---

## What to read next

- **Try it yourself** — the [Getting Started guide](getting-started.md) walks through a complete feature from installation to merge.
- **Understand retrospectives** — the [Retrospectives](retrospectives.md) document covers how the system records process signals, synthesises themes, and generates reports.
- **Look up tool parameters** — the [MCP Tool Reference](mcp-tool-reference.md) documents every tool mentioned in this document.
- **Look up entity fields** — the [Schema Reference](schema-reference.md) covers knowledge entry fields, worktree records, and task structures.
- **Return to the overview** — the [User Guide](user-guide.md) provides orientation to every part of the system.