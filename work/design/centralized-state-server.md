| Field  | Value |
|--------|-------|
| Date   | 2026-04-22T00:00:00Z |
| Status | Draft |
| Author | GPT-5.4 |

## Problem and Motivation

Kanbanzai is currently designed as a Git-native workflow system. Canonical project state lives in repository files under `.kbz/state/`, and Git acts as both transport and durability boundary for collaboration. This model has strong properties: it is transparent, portable, offline-friendly, and easy to inspect. It also aligns with the current viewer model, where read-only consumers can stay in sync simply by pulling from Git.

However, the Git-native model also imposes constraints that become more visible as the system scales from a single developer or a small number of cooperating agents toward a team-wide, multi-user workflow system.

The main pressures are:

- **Commit-bound visibility.** Shared state is not visible to others until it is committed and pushed.
- **Workflow noise in Git.** If state changes are committed frequently for safety, commit history becomes harder to use for understanding code changes.
- **Concurrency friction.** Multiple agents or humans updating the same entity or related state through separate clones can produce merge conflicts, interleaving, and delayed conflict discovery.
- **Limited real-time coordination.** A Git-native model is eventually shared through push/pull, not immediately shared through a central authority.
- **Query and analytics limits.** Cross-project, cross-user, and time-based queries are possible, but they are not the natural strength of a file-and-Git architecture.

These pressures raise a legitimate architectural question: should Kanbanzai move to, or add, a shared centralized database server that stores canonical workflow state for a team's project?

This document examines that route as a competing design direction. It addresses three questions:

1. **Should Kanbanzai support a centralized state server at all?**
2. **Could Kanbanzai support both Git-native and centralized modes?**
3. **If Kanbanzai moved toward centralized state, what would the transformation look like?**

The goal is not to assume that the Git-native model is wrong. The goal is to evaluate whether a centralized server is a better fit for some teams, and whether Kanbanzai can evolve without losing the strengths that made the Git-native model attractive in the first place.

## Design

### Overview

Introduce a **centralized state server mode** in which canonical workflow state for a project is stored in a shared database and accessed through the Kanbanzai MCP server and CLI. The existing Git-native file store remains a supported mode, at least during transition and likely long-term if dual-mode support proves tractable.

This design treats centralized state not as a small storage substitution, but as a change in system topology. In Git-native mode, the repository is the shared state medium. In centralized mode, the repository contains code and documents, while workflow state is served by a networked authority.

The design therefore has to answer four architectural questions:

1. **What becomes canonical?**
2. **How does the server relate to Git and the repository?**
3. **Can both storage models coexist without constant ambiguity?**
4. **How does Kanbanzai migrate from one model to the other?**

### Recommended centralized architecture

The recommended centralized architecture is:

- a **shared Kanbanzai state server** per team or per project
- a **relational database** as canonical storage, with PostgreSQL as the reference implementation
- the **MCP server and CLI** talking to the state server through a service layer rather than directly mutating `.kbz/state/` files
- the repository continuing to hold:
  - source code
  - human-authored documents
  - configuration needed to connect to the project state backend
- optional **Git projections or exports** of workflow state for audit, backup, or viewer compatibility

In this model, the database is the source of truth for entities, transitions, document records, knowledge entries, worktrees, checkpoints, and related workflow metadata.

### Canonical model options

There are three viable authority models.

#### Model A: Git remains canonical; database is derived

The database is an index, cache, or query accelerator built from `.kbz/state/` files.

This model improves query performance and possibly local coordination, but it does **not** solve the core centralized-state problem. Shared truth still depends on Git commits and pushes. This is therefore not a true centralized-state design.

#### Model B: Database becomes canonical; Git state becomes projection/export

The database is authoritative. `.kbz/state/` files are generated snapshots, optional exports, or compatibility artifacts.

This is the clearest centralized design. It solves real-time coordination and query problems cleanly, but it is also the biggest departure from Kanbanzai's current identity.

#### Model C: Dual canonical modes, selected per project

A project chooses one backend:

- **Git-native backend** — `.kbz/state/` files are canonical
- **Centralized backend** — database is canonical

The MCP and CLI operate against an abstract state service, and the selected backend determines persistence.

This is the most flexible model and the most realistic path if Kanbanzai wants to support both small/local and team/server-backed deployments.

The recommended approach is **Model C**, with a strong bias toward one canonical backend per project. A project should not have two simultaneous sources of truth.

### Why PostgreSQL as the reference centralized backend

A centralized backend needs:

- transactional writes
- concurrency control
- indexed queries
- append-only event history support
- multi-user access
- operational maturity

PostgreSQL is the best fit because it provides:

- strong transactional semantics
- row-level locking and optimistic concurrency support
- JSON support where useful, without forcing a document-store model
- mature tooling for backup, migration, and operations
- broad familiarity for teams likely to run a shared service

SQLite is not the right primary centralized backend. It is excellent as an embedded local database, but it is not a team-shared network authority. If Kanbanzai adopts centralized state, it should do so with a real shared database.

### Data model in centralized mode

In centralized mode, the database stores canonical workflow state. The schema would likely include:

- `projects`
- `plans`
- `features`
- `tasks`
- `bugs`
- `decisions`
- `incidents`
- `documents`
- `knowledge_entries`
- `worktrees`
- `checkpoints`
- `transition_events`
- `audit_events`
- `locks` or version columns for optimistic concurrency

The design should preserve the current conceptual model:

- entity IDs remain stable and human-friendly
- lifecycle validation rules remain the same
- document registration and approval semantics remain the same
- transition history becomes a first-class event table rather than being inferred from Git

The storage engine changes, but the workflow semantics should not.

### Repository relationship in centralized mode

A centralized state server does **not** imply that the repository stops mattering. It changes what the repository is responsible for.

In centralized mode, the repository remains responsible for:

- source code
- design/spec/plan/review documents
- project configuration
- worktree branches and code review flow

The centralized server becomes responsible for:

- canonical workflow entities and metadata
- lifecycle transitions
- task dispatch state
- knowledge entries
- checkpoints
- merge/readiness metadata
- audit history

This means Kanbanzai would no longer be purely "Git is the transport for all workflow state." Instead, Git would remain the transport for code and documents, while the state server would be the transport for workflow state.

That is a meaningful product shift and should be acknowledged explicitly rather than hidden behind implementation details.

### Could both possibilities be supported?

Yes, but only if the boundary is explicit.

The clean way to support both is:

- define a **storage backend interface** in the service layer
- implement at least two backends:
  - file-backed Git-native backend
  - database-backed centralized backend
- select the backend per project through configuration
- ensure all MCP tools and CLI commands operate through the same abstract service contract

The unsafe way to support both would be to let a single project write canonical state to both Git files and a database at the same time. That creates ambiguity, drift risk, and difficult repair semantics.

Therefore the rule should be:

- **one project, one canonical backend**
- optional projections/exports to the non-canonical representation are allowed
- dual-write is acceptable only as a migration phase, not as a steady-state architecture

### What transformation would be required?

A move toward centralized state would require a staged transformation.

#### Stage 1: isolate persistence behind interfaces

Today, much of the system assumes file-backed storage and Git-backed durability. The first transformation is architectural, not operational:

- define backend-neutral service contracts for entity, document, knowledge, checkpoint, worktree, and transition operations
- move validation and lifecycle logic above the persistence layer
- ensure MCP tools depend on services, not file layout assumptions

This stage is required even if centralized mode is never shipped, because it makes the design space explicit.

#### Stage 2: define canonical database schema and migration rules

Create a relational schema that preserves current semantics:

- entity identity and parent relationships
- lifecycle status and legal transitions
- document ownership and approval state
- knowledge confidence and status
- worktree tracking
- audit and transition history

At this stage, also define import/export rules between file state and database state.

#### Stage 3: implement centralized backend in parallel with file backend

Add a database-backed implementation of the service layer. The MCP server and CLI should be able to run against either backend based on project configuration.

This is the point where Kanbanzai becomes genuinely dual-mode.

#### Stage 4: add migration tooling

Provide explicit commands such as:

- import Git-native `.kbz/state/` into database
- export database state to `.kbz/state/` snapshot
- verify equivalence between representations
- cut over a project from file backend to database backend

Migration must be deliberate and reversible during early adoption.

#### Stage 5: adapt surrounding features

Features that currently assume Git-native state need redesign in centralized mode:

- viewer freshness model
- orphaned state commit rules
- auto-commit workflow behaviour
- branch/worktree health checks that currently infer state from Git and files together
- backup and disaster recovery guidance

#### Stage 6: decide product positioning

Once centralized mode exists, Kanbanzai must decide whether it is:

- primarily a Git-native tool with an optional enterprise/server mode, or
- primarily a workflow platform with a legacy Git-native mode

This is not just marketing language. It affects defaults, documentation, support burden, and future design decisions.

### Operational model in centralized mode

A centralized state server introduces new operational concerns that do not exist in the Git-native model:

- authentication and authorization
- database migrations
- backup and restore
- service availability and latency
- multi-tenant or per-project isolation
- audit retention
- incident response for state corruption or outage

These are not reasons to reject centralized mode, but they are real costs. The Git-native model avoids them by making Git the shared substrate. A centralized server reintroduces classic application-operations concerns.

### Failure modes and handling

A centralized design changes the failure profile.

#### Server unavailable

If the state server is down, workflow mutations stop. Depending on design, even reads may stop. This is a new class of outage absent from the Git-native model.

Mitigation:

- read replicas or degraded read-only mode
- clear local caching rules
- operational runbooks and health checks

#### Network partition or offline work

Git-native mode works naturally offline. Centralized mode does not.

Mitigation:

- define whether offline operation is unsupported, read-only, or queue-based
- avoid pretending centralized mode preserves all offline properties of Git-native mode

#### Drift between code/documents and workflow state

In Git-native mode, code and workflow state often move together in the same repository history. In centralized mode, they can diverge more easily.

Mitigation:

- record branch, commit SHA, and repository metadata in workflow operations where relevant
- add consistency checks between open worktrees/branches and centralized entity state
- optionally export state snapshots into Git for audit points

#### Migration ambiguity

If a project partially uses both backends, operators may not know which state is authoritative.

Mitigation:

- one canonical backend per project
- explicit migration states: `file`, `migrating`, `database`
- tooling that refuses ambiguous writes

## Alternatives Considered

### Alternative 1: Stay fully Git-native and improve the file model

**Description:** Keep `.kbz/state/` as canonical and address current pain through better transition logs, batching, and derived indexes.

**What it makes easier:**
- preserves current product identity
- keeps offline and portable workflows
- avoids new infrastructure
- keeps repository-local transparency

**What it makes harder:**
- real-time team coordination remains limited by Git workflows
- merge and concurrency friction remain part of the model
- some analytics and multi-user coordination problems remain awkward

**Why not chosen as the only path:** This remains the best default for many projects, but it does not answer the needs of teams that want centralized coordination and immediate shared state.

### Alternative 2: Add a centralized database as a derived cache only

**Description:** Keep Git-native files canonical and use a shared database only for indexing, dashboards, or query acceleration.

**What it makes easier:**
- better queries and reporting
- lower risk than changing canonical storage

**What it makes harder:**
- does not solve commit-bound visibility
- does not solve canonical concurrency issues
- introduces infrastructure without delivering the main benefits of centralization

**Why rejected:** This is an expensive half-step. It adds operational burden without truly changing the collaboration model.

### Alternative 3: Move entirely to centralized canonical state and remove Git-native mode

**Description:** Replace the file-backed model with a database-backed server and retire `.kbz/state/` as canonical storage.

**What it makes easier:**
- one clear architecture
- no dual-mode complexity
- strongest centralized coordination story

**What it makes harder:**
- abandons the Git-native value proposition
- breaks offline/local-first assumptions
- makes small-project adoption heavier
- turns migration into a one-way product rewrite

**Why rejected:** This is too abrupt and discards too much of what currently makes Kanbanzai distinctive. If centralized mode is added, it should be additive first.

### Alternative 4: Support both backends, selected per project

**Description:** Introduce a backend abstraction and let each project choose Git-native or centralized canonical storage.

**What it makes easier:**
- preserves current strengths for small/local projects
- enables centralized mode for teams that need it
- creates a migration path rather than a forced rewrite

**What it makes harder:**
- increases implementation and testing surface
- requires strict discipline around backend-neutral service contracts
- risks conceptual complexity in documentation and support

**Why chosen:** This is the best balance of flexibility and continuity, provided the system enforces one canonical backend per project and avoids permanent dual-write ambiguity.

## Decisions

- **Decision:** A centralized state server is a valid strategic direction for Kanbanzai, but it should be introduced as an alternative backend, not as an immediate replacement for the Git-native model.
  - **Context:** Some teams will value real-time shared state, stronger concurrency control, and richer queries enough to accept operational complexity.
  - **Rationale:** This expands Kanbanzai's applicability without forcing all users into a heavier deployment model.
  - **Consequences:** The architecture must support backend abstraction and the product must document two deployment modes clearly.

- **Decision:** PostgreSQL should be the reference centralized backend.
  - **Context:** A team-shared canonical store needs transactional integrity, concurrency control, and operational maturity.
  - **Rationale:** PostgreSQL is the most appropriate general-purpose choice for a shared workflow state service.
  - **Consequences:** Centralized mode gains operational prerequisites: database provisioning, migrations, backup, and service management.

- **Decision:** One project must have exactly one canonical backend at a time.
  - **Context:** Simultaneous canonical Git and database state would create ambiguity and drift.
  - **Rationale:** Clear authority boundaries are more important than convenience during migration.
  - **Consequences:** Dual-write is allowed only as a temporary migration mechanism, never as the steady-state model.

- **Decision:** The first implementation step is service-layer decoupling from file-backed persistence.
  - **Context:** The current codebase is optimized around Git-native files.
  - **Rationale:** Without backend-neutral service contracts, centralized mode would become a parallel system rather than an alternative backend.
  - **Consequences:** Some internal refactoring is required before centralized mode can be implemented safely.

- **Decision:** Git-native mode should remain a first-class supported option unless future product strategy explicitly retires it.
  - **Context:** Git-native storage is central to Kanbanzai's current identity and offers real advantages for small teams, portability, and transparency.
  - **Rationale:** Centralized mode should expand the product, not erase its strongest existing mode by default.
  - **Consequences:** Documentation, testing, and support burden increase, but the product remains adaptable to different team sizes and operating models.
