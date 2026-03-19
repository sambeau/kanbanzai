# Workflow System Design

- Status: design proposal (consolidated)
- Author: Sam & Claude
- Date: 2026-03-18
- Incorporates: initial-workflow-analysis.md, initial-workflow-analysis-review.md, workflow-system-design-review.md

---

## 1. Purpose

Design a workflow system for human-AI collaborative software development. The system must work well for humans and AI agents building large projects together: planning, documenting, coding, verifying, bug fixing, and shipping.

The system is general-purpose: simple for small projects, able to grow to large multi-developer projects with multiple human and AI team members working concurrently.

---

## 2. Design Principles

### 2.1 The Parsley Aesthetic

All design decisions follow the Parsley aesthetic: simplicity, minimalism, completeness, composability.

### 2.2 Disciplined Normalisation

The system has strict internal invariants — schemas, state machines, naming rules, referential integrity. Humans communicate in natural language and write in informal markdown. AI agents normalise all human input into clean, consistent, formal representations before it enters the system.

This is a modified form of Postel's Law:

> "Be liberal in what you accept, and disciplined in how you normalize it."

The key idea is that normalisation is an **AI-driven translation step**, not a software interface concern. The chat interface itself does not need to be clever about accepting messy input — the AI agent performs the translation. This applies to both commands and content:

- **Commands:** A human says "we need profile editing" and the AI translates this into a formal `feat_create` call with the correct parameters, after asking clarifying questions.
- **Content:** A human writes a spec in markdown with inconsistent heading levels, missing frontmatter, or vague acceptance criteria. The AI normalises the document — fixing structure, filling gaps, tightening language — and presents the cleaned version for human approval before it enters the system.

This means:

- Humans are never required to learn internal syntax, remember IDs, or maintain state manually
- Humans are never required to write perfectly structured markdown — the AI cleans it up
- The AI never silently invents important facts — it infers when safe, suggests when uncertain, asks when ambiguity matters
- Every piece of human input passes through an AI normalisation step before becoming canonical state
- The normalisation step is visible — humans can review what the AI produced before it is committed
- The agent must show what it changed during normalisation, and flag any places where it changed meaning (not just structure), especially for specs and decisions where subtle meaning changes cascade

### 2.3 Workflow State Is the Source of Truth, Conversation Is the Interface

Canonical truth lives in structured state files tracked by git. Markdown documents are views over that truth, generated or validated by tooling. Humans read and write markdown; the system ensures it stays consistent with the structured state.

### 2.4 One Canonical Fact in One Place

Every fact has exactly one authoritative location. Status lives in state files, not in document headers. Specs live in spec files, not duplicated in plans. IDs are allocated by the tool, not by editing counter files.

### 2.5 Codebase Is the Source of Truth for Code

Documentation describes code; it does not prescribe it. The specification sits above both, defining intent and acceptance criteria. The implementation plan sits below both, describing how to get from spec to code.

### 2.6 Humans Approve Intent; Agents Execute Scoped Work

Humans own goals, priorities, product direction, major tradeoffs, approvals, and ship decisions. AI agents own decomposition, implementation, verification, status tracking, and housekeeping. The boundary is at the specification: humans approve it; agents plan and execute against it.

### 2.7 Contexts Are Small; Links Are Strong

Every unit of AI work fits in one context window. Agents receive precisely curated context, not the full project state. Strong typed links between artifacts replace implicit knowledge.

### 2.8 Parallelism Is Earned Through Isolation, Not Hope

Concurrent work requires conflict-domain isolation (worktrees), dependency-aware scheduling, and explicit merge gates. Hope is not a concurrency strategy.

### 2.9 Operational Documents Are Generated; Human Documents Are Validated

Status reports, roadmap summaries, backlog views, and progress dashboards are generated from structured state on demand. They are never manually maintained.

Human-authored documents (specs, designs) are normalised and validated against their schemas, not rewritten by tooling. Their content is human-authored; their structure is machine-checked.

This is not "all documents are generated" — it is a clear split between operational views (generated) and creative/decision documents (validated).

### 2.10 Verification Is Attached to Every Unit of Work

No task is done without verification. No feature merges without passing its acceptance criteria. No claim of completion without fresh evidence.

### 2.11 Bugs Are First-Class Workflow Objects

Bugs have their own lifecycle, their own schema, and their own concerns (reproduction, severity, root cause, regression). They are not mislabelled tasks.

### 2.12 The Workflow System Must Stay Simpler Than the Project It Manages

If the workflow system becomes a burden, it has failed. Every piece of ceremony must earn its place by reducing friction elsewhere.

---

## 3. Lessons From Existing Systems

### 3.1 What to Preserve From Basil

The current Basil workflow has strong foundations that the new system must preserve:

- **The document chain** (design → spec → plan) creates a decision trail useful for review
- **The Newspaper Pattern** (human-readable top, AI-dense bottom) in specifications
- **IDs everywhere** (FEAT-XXX, BUG-XXX, PLAN-XXX) for traceability
- **Specs as acceptance criteria**, not implementation detail
- **Review against the specification**, not the plan
- **Separation of spec and plan** — humans own the what; agents own the how

### 3.2 What to Fix From Basil

The recurring failures are consistency failures, not design failures:

- Naming conventions drift without enforcement
- Document organisation wanders without validation
- Markdown is too flexible to serve as a reliable system of record
- Status fields become stale without automation
- Superseded documents sit beside current ones without structural links
- Branches drift and are forgotten without tracking
- ID allocation is slow and error-prone without tooling
- The backlog becomes a dumping ground without hygiene rules

The root cause: the workflow has the right structure but lacks enforcement and automation. The solution is tooling that makes the right thing easier than the wrong thing.

### 3.3 Ideas Borrowed From External Systems

**From Superpowers:**

- Write workflow rules that anticipate and counter specific LLM rationalisation patterns
- Use hard gates at critical approval points
- Adversarial review (don't trust self-reported completion)
- Context window management as a first-class concern
- Test workflow rules against observed failure modes

**From GSD (Get Shit Done):**

- Context rot is real — fresh sessions per unit of work are the engineering solution
- The "must fit in one context window" rule for task sizing is pragmatic and effective
- File-based state on disk gives crash recovery and human editability for free
- Git worktrees provide true file-system isolation for concurrent agent work
- Squash-merge per feature gives clean, bisectable history
- Dependency-aware execution waves enable safe parallelism
- Pre-inlined context (injecting relevant state into each fresh session) avoids wasteful re-exploration

**From codebase-memory-mcp:**

- Persistent, queryable, structured memory beats repeated textual rediscovery
- Pre-computed knowledge served via targeted queries is far more token-efficient than re-exploration
- The principle extends beyond code structure to workflow state, planning decisions, and team knowledge
- A planning-oriented query layer complements (not replaces) a code-oriented one

---

## 4. Architecture

### 4.1 Overview

```
Human ←→ AI Agent ←→ MCP Server (kanbanzai) ←→ State files + SQLite cache
                 ↕
            Codebase + codebase-memory-mcp
```

The AI agent is the conversational interface. The MCP server is the agent's hands. The state files in git are the source of truth. The SQLite cache is a derived, queryable index rebuilt from those files on demand.

Humans may also use the CLI directly for quick operations, scripting, or CI integration. But the primary interface is human → agent → MCP server.

### 4.2 The kanbanzai Binary

A single Go binary with two modes:

- **`kanbanzai serve`** — MCP server (stdio transport), exposes tools to AI agents
- **`kbz <command>`** — CLI for humans, scripts, and CI

`kanbanzai` is the binary name and MCP server name. `kbz` is the CLI shorthand — a symlink or invocation alias to the same binary, used for interactive and scripted commands.

Both modes share the same core logic. The CLI is a convenience, not the primary interface.

The CLI is strict and predictable: no fuzzy matching, no interactive prompts. It supports JSON output for machine consumption alongside human-readable output. It is idempotent where possible. Error messages are comprehensive so agents can interpret failures.

### 4.3 The Material Taxonomy

All material in the system falls into one of three classes:

**Intake artifacts** — human-provided material that has not yet been normalised into canonical state. Notes, drafts, brainstorms, rough bug reports, pasted markdown, review commentary. These are accepted as useful source material but are not canonical truth. The AI agent normalises them before they enter the system.

**Canonical records** — validated structured workflow objects written through formal MCP operations. YAML state files, approved specs, implementation plans. These are the source of truth. They are created and modified only through the `kanbanzai` tool, which enforces schemas, validates transitions, and maintains referential integrity.

**Projections** — generated human-facing views derived from canonical state. Roadmap summaries, status reports, backlog views, handoff packets, dashboards. These are never manually maintained; they are regenerated from canonical records on demand.

This taxonomy prevents the ambiguity that causes documentation drift: every piece of text has a clear role, and raw input material cannot silently become an unofficial system of record.

### 4.4 Three Layers of State

```
┌─────────────────────────────────────────────┐
│  HUMAN LAYER: Markdown documents            │
│  Specs, designs, manual pages               │
│  Human-authored, AI-normalised & validated   │
├─────────────────────────────────────────────┤
│  WORKFLOW LAYER: Structured state files     │
│  YAML in git (source of truth)              │
│  SQLite cache (derived, queryable)          │
├─────────────────────────────────────────────┤
│  AGENT LAYER: Persistent knowledge          │
│  Team memory, decisions, patterns           │
│  Queryable via MCP                          │
└─────────────────────────────────────────────┘
```

**Human Layer:** Markdown documents for human consumption. Specs follow the Newspaper Pattern: human-readable summary and acceptance criteria above the fold, AI-dense technical detail below. Documents are normalised and validated against schemas (required frontmatter, correct naming, section structure) but their content is human-authored.

**Workflow Layer:** The canonical workflow state. YAML files tracked in git — one file per entity (epic, feature, task, bug, decision). Deterministic formatting, stable key ordering, append-friendly where possible. A local SQLite cache provides fast queries and is rebuilt from the YAML files on demand.

**Agent Layer:** Persistent knowledge stores for AI teams. Decisions, patterns, conventions, and learnings accumulated across sessions. Queryable via MCP tools. Structured entries with tags and categories, not free-form text.

### 4.5 Storage Design: Git-Native With Local Cache

The canonical state lives in git-tracked text files because:

- Multiple agents and humans work concurrently across branches
- SQLite is a binary blob that git cannot diff or merge
- Text files provide inspectability, diffability, and merge-friendliness

The SQLite cache is:

- Derived from the text files (rebuilt by `kbz rebuild-cache`)
- Local to each working copy
- Used for fast queries (search, dependency graphs, health checks)
- Never shared through git (listed in .gitignore)
- Disposable — losing it costs nothing; rebuilding it is fast

### 4.6 File Format: YAML

Structured state files use YAML with strict conventions:

- **Deterministic key ordering** — same data always produces the same file
- **No flow style** — always block style for readability and stable diffs
- **No anchors or aliases** — every value is explicit
- **ISO 8601 timestamps** — normalised, no timezone ambiguity
- **One file per entity** — so concurrent edits to different entities never conflict
- **Append-only where possible** — history entries, decision logs, status transitions are appended, not overwritten

The `kanbanzai` tool is the canonical formatter. All writes go through the tool, ensuring deterministic output. Humans may edit YAML directly in a pinch; `kbz lint` catches formatting drift.

---

## 5. Object Model

### 5.1 Task Hierarchy

```
Roadmap
  └── Epic (human-planned, high-level goal)
        └── Feature (FEAT-XXX, spec'd and approved)
              └── Task (context-window-sized, AI-planned)

Bug (BUG-XXX) → links to Feature or stands alone
Decision (DEC-XXX) → links to any entity
```

- **Roadmap** is the top-level view, generated from epic/feature state. Not a separate entity.
- **Epics** are human-managed. Humans decide what, why, and in what order.
- **Features** are the handoff point. Human approves the spec; AI creates the plan and tasks.
- **Tasks** are the unit of AI work. Each fits in one context window. Each gets a fresh agent session with pre-inlined context.
- **Bugs** have their own lifecycle and schema. They link to features but are not subordinate to them.
- **Decisions** record rationale and are linked to the entities they affect.

### 5.2 The Feature as Composite Entity (v1 Simplification)

In this design, Feature is intentionally a composite entity. It carries:

- The concept of the feature itself
- The specification (as a linked markdown document)
- The implementation plan (as a linked markdown document)
- Approval status (of the specification)
- Implementation lifecycle (of the tasks)

This is a deliberate v1 simplification. In a future version, Specification and Plan may become first-class entities with their own IDs, lifecycles, and supersession chains. This would be necessary if:

- Multiple specs need to coexist for the same feature (e.g. during major redesign)
- Plans need independent versioning from specs
- Approval history needs to be tracked as a separate audit trail
- The distinction between "spec defect" and "implementation defect" needs stronger structural support

For now, the composite model is simpler and sufficient. The design should be revisited when supersession and approval history become more important.

### 5.3 Entity Schemas

#### Epic

```
id: E-003
slug: user-accounts
title: User Accounts
status: active
summary: >
  User registration, authentication, profiles, and account management.
created: 2026-03-18T10:00:00Z
created_by: sam
features: [FEAT-150, FEAT-151, FEAT-152]
```

#### Feature

```
id: FEAT-152
slug: profile-editing
epic: E-003
status: in-progress
summary: >
  Allow users to edit their profile: name, bio, avatar.
  Must be mobile-responsive.
created: 2026-03-18T10:30:00Z
created_by: sam
spec: work/specs/FEAT-152-profile-editing.md
plan: work/plans/FEAT-152-profile-editing.md
branch: feat/FEAT-152-profile-editing
tasks: [FEAT-152.1, FEAT-152.2, FEAT-152.3]
decisions: [DEC-041]
```

#### Task

```
id: FEAT-152.1
feature: FEAT-152
slug: profile-model
summary: Add profile model and database migrations
status: done
assignee: agent-backend
depends_on: []
files_planned:
  - pkg/models/profile.go
  - pkg/db/migrations/004_profiles.go
started: 2026-03-18T11:00:00Z
completed: 2026-03-18T11:45:00Z
```

#### Bug

```
id: BUG-027
slug: avatar-upload-timeout
title: Avatar upload times out on files over 2MB
status: triaged
severity: medium
priority: high
type: implementation-defect
reported_by: sam
reported: 2026-03-19T09:00:00Z
affects: [FEAT-152]
origin_feature: FEAT-152
origin_task: FEAT-152.3
environment: production
observed: Upload spinner runs indefinitely, no error shown
expected: Upload completes or shows a clear error within 5 seconds
reproduction: Upload any JPEG over 2MB via the profile edit page
```

#### Decision

```
id: DEC-041
slug: no-client-side-cropping-v1
summary: Avatar uploads are upload-as-is, no client-side cropping
rationale: Simplicity for v1. Cropping can be added later as a separate feature.
decided_by: sam
date: 2026-03-18
affects: [FEAT-152]
supersedes: null
superseded_by: null
```

### 5.4 Lifecycle State Machines

Each entity type has a defined set of allowed states and transitions.

#### Epic

```
proposed → approved → active → done
                        ↓
                     on-hold
```

#### Feature

The Feature state machine is intentionally composite in v1, blending spec approval and implementation lifecycle. If Specification and Plan become first-class entities later, this state machine would split.

```
draft → in-review → approved → in-progress → review → done
                       ↓                        ↓
                   superseded              needs-rework → in-progress
```

#### Task

```
queued → ready → active → needs-review → done
                   ↓           ↓
                blocked    needs-rework → active
```

#### Bug

```
reported → triaged → reproduced → planned → in-progress → needs-review → verified → closed
              ↓                                                              ↓
          duplicate                                                     needs-rework → in-progress
              ↓
          not-planned
              ↓
          cannot-reproduce
```

#### Decision

```
proposed → accepted → superseded
              ↓
           rejected
```

### 5.5 Formal Supersession

Later documents must not merely sit beside earlier ones. Supersession is explicit and structural:

- Every revisable entity supports `supersedes` and `superseded_by` fields
- The system can answer: "What is the currently approved specification for this feature?"
- The system can detect: "Which tasks were planned against an outdated spec?"
- Superseded entities remain in the repository for history but are clearly marked

### 5.6 Bug Classification

The system distinguishes three situations that should not be conflated:

- **Implementation defect:** code is wrong relative to the approved spec. Fix the code.
- **Specification defect:** the spec itself is wrong or incomplete. Supersede the spec, then fix the code.
- **Design problem:** the design is undesirable even if implementation matches the spec. Revisit the design, then supersede the spec, then fix the code.

The bug's `type` field captures this distinction, which determines the workflow path.

### 5.7 The Standard Bugfix Path

A typical bugfix follows this workflow:

1. **Report** — create a Bug. Capture rough report, environment, severity, and impact. Humans can report bugs informally in chat ("the composer ate my draft again") and the AI normalises the report through clarifying questions before creating the canonical record.
2. **Triage** — confirm class, priority, duplicate status, and likely scope. Determine whether this is an implementation defect, spec defect, or design problem.
3. **Reproduce** — establish a reliable reproduction path. Ideally convert it into a test or repeatable script.
4. **Plan** — prepare a fix plan with root-cause hypothesis and verification expectations.
5. **Fix** — execute through normal task and plan machinery.
6. **Verify** — confirm the reproduction no longer fails. Add regression coverage where appropriate.
7. **Close** — record release target and lessons learned if needed.

### 5.8 Bug Metadata

Beyond the core schema, bugs benefit from structured metadata:

- `severity` — how bad is the impact
- `impact_area` — which part of the system
- `bug_class` — regression, edge case, data handling, UI, performance, security
- `introduced_by` — which feature or commit introduced the bug
- `detected_in` — where it was found (testing, staging, production, user report)
- `customer_visible` — whether end users are affected
- `reproducible` — always, sometimes, rarely, cannot reproduce
- `requires_hotfix` — whether it needs an out-of-cycle release
- `verification_class` — how to verify the fix (automated test, manual test, both)

### 5.9 Deferred Entity Types

The following entity types are not in the v1 object model but are expected to become necessary:

- **Specification** — if specs need independent lifecycles, versioning, or supersession chains separate from their parent feature
- **Plan** — if plans need independent versioning from specs
- **Approval** — if approval needs to become a trackable, auditable object rather than a status field
- **Release** — when the project reaches the point of shipping versioned releases
- **Incident** — for production-significant failures (outages, data corruption, security issues) that need a more rigorous workflow than a standard bugfix
- **RootCauseAnalysis** — for important bugs and incidents, capturing: what happened, why, why it wasn't caught earlier, what changed, and what will prevent recurrence

These will be designed when they are needed, not before.

---

## 6. Identity Strategy

### 6.1 Requirements

IDs must be:

- Unique across contributors and branches (distributed-safe)
- Short enough for humans to speak and remember
- Sortable (chronological ordering is useful)
- Stable once created (never renumbered)
- Merge-safe (no conflicts when two branches create entities simultaneously)

### 6.2 Format: ID + Slug

Every entity has both a machine ID and a human-readable slug:

- `FEAT-152-profile-editing`
- `BUG-027-avatar-upload-timeout`
- `E-003-user-accounts`
- `DEC-041-no-client-side-cropping-v1`

The ID is the stable reference. The slug aids human scanning. Both appear in filenames and cross-references.

### 6.3 Distributed ID Allocation

In a multi-branch, multi-agent environment, a centralised counter file is a bottleneck and merge conflict magnet. The system uses **block allocation**:

1. Before creating a feature branch, the agent allocates a block of IDs on the main branch (e.g. FEAT-152 through FEAT-156).
2. Within its branch, the agent allocates from that block without further coordination.
3. Unused IDs in a block are returned when the branch merges.
4. The `kanbanzai` tool handles this atomically.

For small teams or solo work, a simpler approach works: allocate one ID at a time on main before branching. Block allocation is available when concurrency demands it.

Task IDs are scoped to their parent feature (FEAT-152.1, FEAT-152.2) and allocated entirely within the feature branch, so they never conflict.

### 6.4 ID Strategy: Known Edge Cases

The block allocation strategy has edge cases that need handling:

- **Block exhaustion mid-feature:** If a feature needs more IDs than its reserved block, the agent must return to main to reserve another block. The tool should warn when a block is nearly exhausted.
- **Abandoned branches with unreturned IDs:** IDs in an abandoned block become permanently unused (gaps in the sequence). This is acceptable — IDs are never reused, and gaps are harmless.
- **Multi-project repositories:** Each project within a shared repo needs its own ID namespace. The tool should support a project prefix or separate counter per project.

### 6.5 Alternative: Distributed Sortable IDs

Block allocation is the current recommendation, but a distributed sortable ID format (time-based, short, collision-resistant, paired with slug) remains a viable alternative. It would eliminate coordination entirely at the cost of less human-friendly IDs (e.g. `FEAT-26H3K4-7Q` instead of `FEAT-152`).

This is an open design decision that should be tested during implementation. The system should be designed so the ID format can be changed without restructuring the rest of the architecture.

---

## 7. Directory Structure

```
work/
  state/
    epics/
      E-003-user-accounts.yaml
    features/
      FEAT-150-user-registration.yaml
      FEAT-151-authentication.yaml
      FEAT-152-profile-editing.yaml
    tasks/
      FEAT-150.1-registration-model.yaml
      FEAT-150.2-registration-api.yaml
      FEAT-152.1-profile-model.yaml
      FEAT-152.2-profile-api.yaml
    bugs/
      BUG-027-avatar-upload-timeout.yaml
    decisions/
      DEC-041-no-client-side-cropping-v1.yaml
    ids.yaml                  # ID allocation state
  specs/
    FEAT-150-user-registration.md
    FEAT-152-profile-editing.md
  plans/
    FEAT-150-user-registration.md
    FEAT-152-profile-editing.md
  designs/
    user-accounts-system.md
  knowledge/
    architecture/             # architectural decisions, patterns, tradeoffs
    backend/                  # API patterns, data model, Go idioms
    frontend/                 # UI patterns, component conventions
    testing/                  # test patterns, fixture conventions
  reports/                    # generated views (gitignored or regenerated)
    roadmap.md
    backlog.md
    status.md
```

Key properties:

- **One file per entity** in `state/` — concurrent edits to different entities never conflict
- **State is separate from content** — YAML state files in `state/`, markdown documents in `specs/`, `plans/`, `designs/`
- **Filenames include ID + slug** — both machine-sortable and human-scannable
- **Knowledge stores are team-scoped** — each subdirectory serves a specialist team
- **Generated reports are not committed** (or are regenerated before commit) — they never drift

---

## 8. MCP Server Interface

The MCP server is the primary interface for AI agents. Each tool does one thing, has strict input validation, and returns structured data the agent can interpret and present however it wants.

### 8.1 Identity & Scaffolding

| Tool | Purpose |
|------|---------|
| `id_allocate` | Allocate the next ID for a given type. Atomic. Returns the allocated ID. |
| `id_reserve` | Reserve a block of IDs for distributed work. Returns the reserved range. |
| `epic_create` | Create an epic: allocates ID, scaffolds state file. |
| `feat_create` | Create a feature: allocates ID, scaffolds state file + spec template, links to epic. |
| `bug_create` | Create a bug: allocates ID, scaffolds state file + report template. |
| `task_create` | Create a task under a feature: allocates sub-ID, scaffolds state file. |
| `decision_record` | Record a decision: allocates ID, creates state file, links to affected entities. |

### 8.2 Status & Lifecycle

| Tool | Purpose |
|------|---------|
| `status_update` | Change the status of any entity. Validates the transition is allowed. |
| `status_get` | Get current status of an entity by ID. |
| `approve` | Mark a spec or plan as approved. Records who and when. |
| `defer` | Move an entity to the backlog with a reason. |
| `supersede` | Mark an entity as superseded by another. Updates both entities. |

### 8.3 Querying

| Tool | Purpose |
|------|---------|
| `search` | Find entities by type, status, epic, assignee, text, or date range. |
| `get` | Get full details of an entity by ID. |
| `children` | List child entities (tasks of a feature, features of an epic). |
| `dependencies` | Show dependency graph for a feature or task. |
| `health_check` | Scan for stale branches, stalled tasks, missing specs, orphaned IDs, broken links. |
| `roadmap` | Return the current epic → feature → task hierarchy with statuses. |

### 8.4 Knowledge & Memory

| Tool | Purpose |
|------|---------|
| `knowledge_store` | Record a learning, pattern, or convention for a team/domain. |
| `knowledge_query` | Search the knowledge store by team, topic, or tags. |
| `decision_query` | Search decisions by feature, topic, or date. |
| `context_pack` | Generate a context packet for a task: relevant spec sections, team knowledge, decisions, file references. |

### 8.5 Documents

| Tool | Purpose |
|------|---------|
| `doc_scaffold` | Generate a document from a template (spec, plan, bug report). |
| `doc_validate` | Check a document against its schema (frontmatter, naming, cross-refs). |
| `doc_generate` | Generate a projection from current state (roadmap, status, backlog). |

### 8.6 Git & Branches

| Tool | Purpose |
|------|---------|
| `branch_create` | Create a worktree + branch for a feature or bug. Links to entity. |
| `branch_status` | Check drift from main, last activity, merge readiness. |
| `branch_list` | List active worktrees with status. |
| `branch_cleanup` | Remove merged or abandoned worktrees. |

### 8.7 Normalisation Support

These tools help the AI agent prepare and validate its normalisation work before committing to canonical state:

| Tool | Purpose |
|------|---------|
| `validate_candidate` | Check candidate entity data against the schema without creating it. Returns errors, warnings, and missing required fields. |
| `detect_duplicates` | Given a summary and type, find possible duplicate entities. Useful before creating bugs or features. |
| `resolve_links` | Given loose references (partial IDs, slugs, or descriptions), suggest likely entity matches. |
| `preview_commit` | Show what a create or update operation would produce without performing it. Returns the YAML that would be written. |

These tools make the normalisation pipeline more robust: the agent can validate its work through the MCP server before committing, rather than hoping it got everything right.

### 8.8 Tool Design Principles

- **One tool per workflow action** that a human would think of as a single step
- **Compound operations where natural** — `feat_create` allocates an ID, creates a state file, scaffolds a spec template, and links to the epic, all in one call
- **Structured return values** — JSON that the agent can interpret and present however suits the conversation
- **Clear error messages** — so the agent can diagnose failures without human help
- **Idempotent where possible** — retrying a failed call should be safe

---

## 9. Concurrency and Source Control

### 9.1 Worktrees for Isolation

Each feature or bug gets its own git worktree on a dedicated branch:

```
~/Dev/my-project/                       ← main branch (human workspace)
~/Dev/my-project-worktrees/
  ├── feat-FEAT-152-profile-editing/    ← Agent A's worktree
  ├── feat-FEAT-153-notifications/      ← Agent B's worktree
  └── bug-BUG-027-avatar-upload/        ← Agent C's worktree
```

Each agent works in complete isolation — different directory, different branch, different set of changed files. There is no chance of agents stepping on each other's uncommitted work.

### 9.2 Conflict Domain Awareness

Parallelism should be planned based on conflict domains, not just team structure. Conflict domains are defined by:

- **File overlap** — two features that modify the same files should be sequenced
- **Dependency ordering** — a feature that depends on another's output must wait
- **Architectural boundaries** — work within the same subsystem has higher conflict risk
- **Verification boundaries** — changes that affect the same test suites may interfere

The workflow engine knows (from the plan) which files each task intends to modify. When two features share a conflict domain, the engine flags this at planning time and suggests sequencing rather than parallelising. This is planning-level awareness, not git-level locking.

### 9.3 Prefer Vertical Slices

Tasks that represent a coherent end-to-end capability (a vertical slice through the architecture) tend to parallelise better than broad horizontal layers. A vertical slice touches few files across many layers; a horizontal layer touches many files in one layer. The vertical slice has a smaller conflict surface.

When decomposing features into tasks, prefer vertical slices where possible.

### 9.4 Branch Hygiene

The `kanbanzai` tool tracks every worktree:

- **Age:** warns when a branch has been active too long
- **Drift:** warns when a branch has fallen behind main by too many commits
- **Activity:** warns when a branch has had no commits recently
- **Merge readiness:** checks that tests pass, spec is current, required state transitions are complete

`health_check` surfaces all of these as warnings. The agent (or human) decides what to do.

### 9.5 Merge Strategy

- **Feature branches merge into main via PR** — this provides a review gate
- **Squash merge per feature** — gives clean, bisectable history where each merge commit represents a complete feature
- **Rebase before merge** — the `kanbanzai` tool can automate rebasing to keep branches current
- **Delete branch and worktree after merge** — `branch_cleanup` handles this

### 9.6 Merge Gates

Before a feature branch can merge, the system verifies:

- All tasks are in `done` status
- The spec is in `approved` or `done` status
- All tests pass
- The branch is not stale (recently rebased)
- No `health_check` errors exist for this feature

These are machine-checkable. The tool reports them; the human (or orchestrating agent) makes the merge decision.

---

## 10. Human-AI Delegation Model

### 10.1 Role Hierarchy

```
Human (Project Manager / Designer / Stakeholder)
  │
  ├── Defines epics, writes/approves specs, makes design decisions
  │
  ▼
Orchestrator Agent
  │
  ├── Reads workflow state, decomposes features into tasks
  ├── Assigns tasks, creates fresh sessions with curated context
  ├── Reviews task output against spec
  ├── Manages branches, tracks progress, surfaces problems
  │
  ▼
Specialist Team Agent (backend, frontend, infrastructure, QA, docs)
  │
  ├── Owns team-scoped knowledge and conventions
  ├── Provides domain expertise for planning and review
  ├── May orchestrate worker agents within its domain
  │
  ▼
Worker Agent (fresh session per task)
  │
  └── Implements one task, verifies, reports back
```

### 10.2 The Orchestrator

The orchestrator agent has a small, focused context: the current workflow state, the spec for the feature it's working on, and relevant team knowledge. It never accumulates implementation detail from worker sessions.

Its responsibilities:

- Decompose an approved feature spec into context-window-sized tasks
- Identify dependencies between tasks and sequence them appropriately
- Create a fresh worker session for each task with precisely curated context
- Review worker output against the spec's acceptance criteria
- Update workflow state as tasks complete
- Escalate to the human when blocked, ambiguous, or when decisions are needed

### 10.3 Specialist Team Agents

Specialist team agents sit between the orchestrator and workers. They own domain-specific knowledge and conventions:

- A backend team agent knows the API patterns, data model conventions, and Go idioms used in the project
- A frontend team agent knows the component conventions, styling rules, and responsive patterns
- A QA team agent knows the test patterns, fixture conventions, and coverage requirements

Each team agent has access to its scoped knowledge store (`work/knowledge/<team>/`). It can be consulted by the orchestrator during planning and by workers during implementation.

In a small project, the orchestrator and team agents may be the same session. The separation becomes important as the project grows.

### 10.4 The Worker

The worker agent gets a precisely curated context injection:

- The task description and acceptance criteria
- Relevant code files (identified via codebase-memory-mcp queries)
- Relevant team knowledge entries
- The feature spec (or the relevant section)
- Explicit boundaries: what files to touch, what not to touch

It works, commits, runs tests, and reports back. Its session ends when the task is done or blocked. It never persists beyond one task.

### 10.5 The Conversational Boundary

Humans interact with the orchestrator (or a general-purpose agent) through natural language. The agent:

1. **Interprets** — recognises intent (new feature? bug report? status question? design discussion?)
2. **Clarifies** — asks focused follow-up questions to fill gaps
3. **Validates** — checks that the information is consistent with existing state
4. **Normalises** — translates human input into clean, formal representations. This applies to both workflow commands (translating intent into MCP tool calls) and content (cleaning markdown documents, tightening acceptance criteria, fixing structure). The normalised output is presented to the human for approval before being committed.
5. **Executes** — calls the MCP server to perform formal operations
6. **Reports** — presents results in human-readable form

The normalisation step (4) is the core of the disciplined normalisation principle. It is where sloppy human input becomes clean system state. The AI does the work; the human reviews the result. The agent must show what it changed and flag any places where it altered meaning, not just structure.

The agent is proactive: when a human starts a conversation, the agent checks project health and mentions problems worth attention ("before we start, FEAT-138 has been stalled for 12 days — should I shelve it?").

---

## 11. Knowledge and Memory

### 11.1 Memory Classes

Memory is not one giant undifferentiated pile. The system distinguishes:

**Canonical project memory:** Authoritative facts — approved decisions, active specifications, architecture principles, current roadmap state. Stored in state files and specs.

**Team operational memory:** Team-scoped guidance — conventions, common pitfalls, preferred patterns, subsystem-specific norms. Stored in `work/knowledge/<team>/`.

**Working memory:** Short-lived operational context — current blockers, recent changes, active worktrees, pending reviews. Derived from current state, not separately stored.

**Expertise memory:** Accumulated heuristics — known regression hotspots, common failure modes, successful decomposition patterns, lessons learned. Stored in knowledge entries with structured tags.

### 11.2 Knowledge Entry Format

Knowledge entries are structured, not free-form:

```
id: K-backend-042
team: backend
topic: database-migrations
tags: [postgres, migrations, testing]
summary: Always run migrations in a transaction and test rollback
detail: >
  Migration files must include both up and down functions.
  The CI pipeline runs each migration forward and backward
  to verify rollback works. Migrations that cannot be rolled
  back must be flagged in the migration file header.
learned_from: FEAT-128
date: 2026-03-15
```

Structured entries with consistent metadata can be searched, filtered, and injected into context precisely. Free-form text is what design docs and reports provide; the knowledge store is for distilled, queryable facts.

### 11.3 Decision Records

Decisions are a special category of knowledge. They record not just what was decided but why, and they link to the entities they affect:

- Every decision has an ID, summary, rationale, and author
- Decisions link to features, specs, bugs, or other decisions
- Decisions support supersession (DEC-041 superseded by DEC-057)
- The `decision_query` tool lets agents find relevant decisions before starting work

When a human answers a design question during conversation, the agent records it as a decision — so future agents (and humans) can find the rationale without repeating the discussion.

### 11.4 What Goes Where

To prevent the knowledge store becoming a dumping ground, clear boundaries between record types:

| If it is... | Store it as... | Example |
|-------------|---------------|---------|
| A product or architectural choice with rationale | **Decision** | "No client-side cropping in v1 — simplicity" |
| A team convention or coding pattern | **Knowledge entry** (team operational memory) | "Always run migrations in a transaction" |
| A learned heuristic or gotcha | **Knowledge entry** (expertise memory) | "The image pipeline silently drops alpha channels" |
| A requirement or acceptance criterion | Part of the **Specification** | "Profile page must be mobile-responsive" |
| A one-off observation during implementation | Part of the **Task** completion notes | "Had to work around Go's image/jpeg not supporting progressive JPEG" |
| A production failure analysis | **Incident / RootCauseAnalysis** (deferred) | "Outage caused by unbounded query on users table" |

If it doesn't fit any of these, it probably belongs in a design document or doesn't need to be stored at all.

---

## 12. Metadata Governance

### 12.1 Principle

Metadata beyond the core schema is allowed, but must be formally defined. Ungoverned metadata is how fields like `priority` end up meaning different things to different teams.

### 12.2 Metadata Registration

Every metadata field must be defined in a central schema registry with:

- **Name** — the field identifier
- **Meaning** — what it represents
- **Value type** — string, enum, number, boolean, list
- **Allowed values** — for enums, the complete set of valid values
- **Scope** — which entity types it applies to
- **Examples** — at least two concrete examples
- **Owner** — who is responsible for the definition
- **Process** — how new values or changes are proposed

### 12.3 Standard Metadata Families

Initial metadata categories likely to be needed:

- `priority` — urgency and importance
- `severity` — impact of a bug or incident
- `risk` — likelihood and consequence of failure
- `domain` — which area of the product
- `subsystem` — which technical component
- `audience` — who is affected (end users, developers, ops)
- `verification_class` — how to verify (automated, manual, both)
- `dependency_class` — blocking, soft, informational

### 12.4 Text Search vs Structured Metadata

Both are useful for different purposes:

- **Text search** is good for narrative discovery, exploratory lookup, and fuzzy finding
- **Structured metadata** is good for filtering, queueing, routing, dashboards, validation, automation, and consistency across agents and humans

The system uses both. Text search for discovery; structured metadata for operation.

---

## 13. Continuous Validation

### 13.1 Health Checks

The `health_check` tool scans the entire project for inconsistencies:

- Stale branches (not rebased in N days, no recent commits)
- Stalled features (status is `in-progress` but no task activity)
- Missing specs (feature exists but spec file is absent)
- Missing plans (spec is approved but no plan exists)
- Orphaned IDs (ID allocated but no entity created)
- Broken links (entity references a non-existent parent, child, or decision)
- Schema violations (missing required fields, invalid status values)
- Naming violations (file doesn't match ID + slug convention)
- Supersession inconsistencies (entity superseded but successor doesn't exist)

Health checks run:

- On agent startup (proactive problem surfacing)
- Before merge (merge gate)
- On demand via CLI or MCP tool
- Optionally as a git pre-push hook

### 13.2 Document Validation

`doc_validate` checks markdown documents against their type's schema:

- Required frontmatter fields present and valid
- Filename matches the expected pattern (ID + slug)
- Cross-references point to entities that exist
- Required sections present (for templated documents)

This catches drift early without constraining the content of human-authored sections.

---

## 14. GitHub Integration

The `kanbanzai` tool is the source of truth. GitHub is a view into that truth for humans who prefer that interface, and a coordination layer for the team.

### 14.1 What Stays in kanbanzai

- ID allocation and tracking
- Specs, plans, and design docs
- Task decomposition and status
- Team knowledge and persistent memory
- Document scaffolding and validation
- Branch and worktree management

### 14.2 What Gets Pushed to GitHub

- **Pull Requests** for feature merges — familiar review interface, natural approval gate
- **PR status checks** — validation that specs exist, plans are approved, tests pass
- **Issue linking** (optional) — create a GitHub Issue for each epic or feature, keep status synced, provides a human-friendly dashboard

### 14.3 Principle

The tool pushes state to GitHub; it does not depend on GitHub as the source of truth. A project should work fully with just the `kanbanzai` tool and git. GitHub integration is an enhancement, not a requirement.

---

## 15. Relationship to Agent Instruction Systems

### 15.1 Layered Configuration

The workflow system complements, not replaces, platform-native agent instruction systems:

**Layer 1 — Platform-native agent instructions:** AGENTS.md, .github/copilot-instructions.md, skills, coding rules. These continue to exist where runtimes expect them. They help the agent carry out interpretation, clarification, and normalisation.

**Layer 2 — Workflow system rules:** Artifact schemas, allowed state transitions, task handoff structure, verification requirements, naming rules. These are defined by `kanbanzai` and enforced by its tools.

**Layer 3 — Generated context packets:** The `context_pack` tool generates compact, focused context for a specific task. This can be fed into agent prompts alongside platform-native instructions.

**Layer 4 — Workflow MCP interface:** The `kanbanzai` MCP server exposes strict typed tools for querying, validating, creating, updating, linking, superseding, and rendering workflow state. This is the formal control surface agents use when committing workflow actions.

### 15.2 What Changes

The workflow system should:

- Reduce the amount of process logic embedded in prose instruction files
- Provide stronger structured context to those systems
- Make process more consistent across sessions
- Make agents less responsible for inventing project state on the fly
- Let agent instructions focus on interpretation and normalisation while MCP tools handle formal workflow operations

It should not replace tool-native agent configuration mechanisms that runtimes expect.

---

## 16. Error Correction and Rollback

### 16.1 Principle

Mistakes happen. The system must support error correction without requiring humans to understand git internals.

### 16.2 Types of Error

**Wrong state** — a status update was incorrect, a field value was wrong, a link pointed to the wrong entity. These are corrected by updating the entity through the normal MCP tools. The previous value is visible in git history.

**Wrong decision** — a decision was recorded but turned out to be incorrect or circumstances changed. These are handled through supersession: create a new decision that supersedes the old one, with rationale explaining what changed.

**Wrong decomposition** — a feature's task breakdown was misguided. Tasks can be moved to `not-planned` status with a reason, and new tasks created. The original tasks remain in git history.

**Wrong normalisation** — the AI normalised human input incorrectly, changing meaning during cleanup. This is caught at review time (the human rejects the normalised version). If it slips through, it is corrected by updating the canonical record. The commit history shows what changed and when.

### 16.3 No Destructive Undo

The system does not support destructive undo (deleting entities, erasing history). Instead, entities move to terminal states (`not-planned`, `rejected`, `superseded`) with rationale. Git history provides the full audit trail.

---

## 17. Migration

### 17.1 Existing Projects

Projects with existing workflow documentation (like Basil, with 149 features, 130 plans, and 26 bugs) need a migration path.

### 17.2 Recommended Approach

1. **Start fresh for new work.** New features, bugs, and decisions use the `kanbanzai` system from day one.
2. **Import selectively.** Import active (non-completed) features, open bugs, and current decisions into the new state format. Completed work remains in the old format as archived history.
3. **Don't backfill exhaustively.** Importing 149 completed features provides little value and high effort. A cutover date is cleaner: "everything before FEAT-150 is in the old system; everything from FEAT-150 onward is in the new system."
4. **Build a migration tool.** A `kbz migrate` command that reads the old markdown-with-frontmatter format and produces the new YAML state files, flagging anything that needs manual review.

### 17.3 Caveats

- ID ranges may need adjustment to avoid collisions between old and new systems
- Cross-references from new entities to old entities should work (the tool should resolve IDs regardless of which system created them)
- The old documentation can remain in place alongside the new system during transition

---

## 18. Implementation Phases

### Phase 1: Workflow Kernel

The first build. Solves the highest-friction problems. Scope is deliberately constrained.

**Entity types:** Epic, Feature, Task, Bug, Decision.

**MCP tools:** `feat_create`, `bug_create`, `epic_create`, `task_create`, `decision_record`, `status_update`, `status_get`, `approve`, `search`, `get`, `health_check`, `doc_scaffold`, `doc_validate`, `validate_candidate`.

**CLI:** Same operations as MCP tools, for human and CI use.

**Not in phase 1:** Knowledge stores, worktree management, GitHub sync, orchestration, context packing, `detect_duplicates`, `resolve_links`, `preview_commit`.

This alone eliminates most of the consistency problems that plague the current workflow.

### Phase 2: Retrieval and Context Packing

The knowledge layer.

- Knowledge store (record and query learnings, patterns, conventions)
- Decision records (record and query decisions with rationale)
- Context packing (generate focused context packets for task handoff)
- Project overview generation (roadmap, status, backlog views)
- Team-scoped memory stores
- Normalisation support tools (`detect_duplicates`, `resolve_links`, `preview_commit`)

This reduces token burn and improves consistency across agent sessions.

### Phase 3: Git Integration

Branch and worktree management.

- Create worktrees linked to features/bugs
- Track branch age, drift, and activity
- Merge readiness checks
- Branch cleanup after merge
- GitHub PR creation and status sync

This enables safe concurrent work across multiple agents.

### Phase 4: Orchestration

The delegation chain.

- Task decomposition from approved specs
- Dependency-aware task scheduling
- Fresh session dispatch with curated context
- Worker output review against spec
- Automated status updates from task completion

This is the most complex piece and should come last, after the simpler tools have proven their value.

---

## 19. Future Considerations

The following areas are worth investigating but are not part of the initial design:

1. **Schema validation tooling** — standalone YAML schema validators that can run in CI, beyond what `kbz lint` provides
2. **Task graph / dependency graph tooling** — lightweight visualisation of parent-child and dependency structures
3. **CI enforcement hooks** — git hooks or CI checks that validate schema correctness, referential integrity, approval consistency, and naming conventions on every push
4. **Append-only event capture** — a log of approvals, handoffs, status transitions, and significant decisions for auditability and workflow reconstruction
5. **Static documentation renderers** — generating HTML or other formats from the structured state and markdown projections
6. **Semantic search for workflow artifacts** — vector/embedding-based search over specs, decisions, and knowledge entries (beyond text grep)
7. **Git worktree lifecycle tooling** — utilities beyond what `kanbanzai` provides for making worktree management safer and more visible

---

## 20. Open Questions

The following questions need answers before or during implementation:

1. **YAML vs alternatives:** YAML is proposed but has known pitfalls (implicit typing, indentation sensitivity). Should we evaluate TOML or a stricter YAML subset? The format must be deterministic, diffable, and human-editable.

2. **Knowledge entry granularity:** How fine-grained should knowledge entries be? One per insight? One per topic? Too fine-grained creates noise; too coarse loses precision.

3. **Worktree directory layout:** Where should worktrees live relative to the project? A sibling directory (`../project-worktrees/`)? A subdirectory? This affects agent configuration and path resolution.

4. **Approval workflow detail:** What does approval look like in practice? The human tells the agent "that looks good" and the agent calls `approve`? Or something more structured?

5. **Context packet contents:** What should `context_pack` include for a typical task? How do we keep it small enough to be useful without being so large it defeats the purpose?

6. **Report generation scope:** Which reports are generated on demand vs committed to the repo? Committing them makes them available to all agents; generating on demand keeps them fresh.

7. **Scale boundaries:** At what project size does this system's file-per-entity approach become unwieldy? Hundreds of tasks? Thousands? Is there a migration path to a different storage model?

8. **Multi-project support:** Should the tool support managing multiple projects from one installation, or is one instance per project sufficient?

9. **ID format:** Block allocation vs distributed sortable IDs — this needs testing during implementation to determine which approach works better in practice.

---

## 21. Summary

The proposed system is a Git-native, schema-backed workflow kernel with markdown projections, deterministic validation, and agent query tools.

Its core ideas:

- Structured state files in git as the single source of truth
- Three classes of material: intake artifacts, canonical records, and projections
- Markdown as a human-facing view, normalised and validated but not constraining
- A Go binary serving both CLI and MCP interfaces
- Disciplined AI-driven normalisation of all human input, with visible review before commit
- Distributed-safe ID allocation
- One file per entity for merge-safe concurrency
- Git worktrees for parallel agent isolation
- Four-tier delegation: human → orchestrator → specialist team → worker
- Persistent team knowledge queryable via MCP, with clear boundaries between record types
- Governed metadata with a central schema registry
- Operational documents generated, human documents validated
- Error correction through supersession and state updates, not destructive undo
- Phased implementation: kernel first, orchestration last

The system meets humans where they are and gives AI agents the precise tools they need. It stays simpler than the project it manages.
