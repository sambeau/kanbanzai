# Meta-Planning: Plans and Batches — Design

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-04-28T00:00:00Z           |
| Status | approved |
| Author | sambeau                        |

## Related Work

### Prior documents consulted

| Document | Type | Relationship |
|----------|------|-------------|
| `work/design/p37-file-names-and-actions.md` | design | Defines plan-first folder organisation, plan-scoped feature IDs (`P24-F3`), and canonical filename templates — this design extends that system with a new entity layer above plans and renames the current plan entity to "batch" |
| `work/design/p37-file-names-and-actions.md` §Open Questions #4 | design | Deferred the question of meta-planning IDs — this design resolves it |
| `work/design/workflow-design-basis.md` | design | Establishes the entity hierarchy (plan → feature → task) and lifecycle state machines — this design adds a recursive plan layer above and renames the work-grouping entity |
| `work/design/estimation-and-progress-design.md` | design | Defines the rollup model (task → feature → plan) — this design extends rollup to be recursive through the plan tree and renames the leaf rollup entity |
| `work/design/document-centric-interface.md` | design | Defines the document type taxonomy and document-driven lifecycle — this design clarifies where design documents live in the new two-layer model |
| `internal/model/entities.go` | code | Defines the current Plan struct, PlanStatus lifecycle, and entity relationships — the primary code affected by this design |
| `internal/validate/lifecycle.go` | code | Defines plan lifecycle transitions and entry states — must accommodate both plan and batch entities |
| `internal/service/estimation.go` | code | Implements `ComputePlanRollup` and `ComputeFeatureRollup` — rollup must become recursive for plans |
| `internal/mcp/status_tool.go` | code | Implements the status dashboard with plan/feature/task synthesis — must accommodate the new plan tree and batch entity |

### Decisions that constrain this design

1. **P1-DEC-006** — `.kbz/state/` owns entity state; `work/` owns human-authored content. This separation is preserved.
2. **P1-DEC-021** — TSID13 is the canonical ID format for features, tasks, bugs, and decisions. This design does not change internal ID formats.
3. **P37-D1** — Plan-first folder organisation. This design extends it: plans get folders for design documents; batches get folders for feature work documents.
4. **P37-D2** — Plan-scoped feature display IDs (`P24-F3`). This design changes the prefix from `P` to `B` for batch-scoped features: `B24-F3`.
5. **P37-D6** — No configurability for structure. The plan/batch distinction is convention, not configuration.

### How this design extends prior work

The current system has a two-level hierarchy: plan → feature → task. The plan entity serves double duty as both a strategic planning container and an operational work-grouping unit. This conflation is manageable for small projects but breaks down when:

- A project needs strategic scope decomposition (what needs to be built) that differs from execution scheduling (what we're building now)
- Work batches need to draw features from multiple areas of the planning tree
- Vague strategic directions need to coexist with concrete, actionable work containers
- The same entity must simultaneously represent an epic, a sprint, and a meta-feature

This design separates the two concerns by introducing a recursive **plan** entity for strategic scope decomposition and renaming the current plan entity to **batch** for operational work grouping. The batch retains all current plan functionality; the plan is a new, deliberately lightweight entity.

---

## Overview

This design introduces a meta-planning layer to Kanbanzai by splitting the current "plan" entity into two distinct entities: **plan** (a recursive strategic planning entity) and **batch** (the renamed current plan entity, a unit of work). Plans decompose into child plans or batches. Batches contain features and tasks, exactly as the current plan does. A project-level singleton provides persistent context (vision, architecture, constraints) that guides all plans. Together, these changes upgrade Kanbanzai from a workflow system into a project-planning system while preserving the simplicity and composability of the existing model.

## Goals and Non-Goals

**Goals:**

- Introduce a recursive planning entity that can represent strategic direction at any level of decomposition
- Rename the current plan entity to "batch" to give it a precise definition as a unit of work
- Preserve all current plan functionality in the renamed batch entity
- Provide a project-level singleton for vision, architecture, and cross-cutting context
- Support gradual refinement: vague strategic plans become concrete batches over time
- Enable cross-cutting execution: a batch can group features from multiple areas of the planning tree
- Extend the progress rollup model to work recursively through the plan tree
- Define how the new entity model interacts with the P37 file naming and folder conventions
- Maintain the ability to use batches standalone, without a parent plan, for small projects

**Non-Goals:**

- Implementing a GUI or visual timeline/roadmap view (deferred)
- Adding Gantt chart or dependency visualisation (deferred)
- Building a formal sprint/cycle time-boxing mechanism (batches are goal-boxed, not time-boxed)
- Changing the feature, task, bug, decision, or incident entity models
- Changing the internal ID system (TSID13 remains canonical for features, tasks, etc.)
- Adding custom fields or configurable metadata to plans or batches
- Implementing cross-plan dependency tracking (deferred; plan-to-plan `depends_on` is described but not required for MVP)

---

## Problem and Motivation

### The overloaded plan entity

The current plan entity (`P1-slug`) is the top-level organising unit in Kanbanzai. In practice, it has been used to represent at least four different concepts:

1. **A meta-feature** — "Build the auth system" (scope-focused, design-driven)
2. **A sprint/batch** — "This week's work across auth and frontend" (execution-focused, cross-cutting)
3. **An epic** — "Backend infrastructure overhaul" (strategic, multi-feature, long-lived)
4. **A phase** — "Phase 15: Infrastructure Hardening" (sequential, milestone-oriented)

These four uses have different lifecycles, different relationships to design documents, and different progress semantics. Conflating them into one entity works for small projects but creates friction as projects grow:

- There is no way to say "these five plans serve this strategic goal"
- There is no way to hold a vague intention that gradually becomes concrete
- Cross-cutting sprints require either duplicating features across plans or restructuring the tree to match execution order
- Design documents have no natural home when a plan is really a sprint

### The missing strategic layer

Every successful planning system, regardless of methodology, converges on three conceptual layers:

| Layer | Purpose | Linear | Shortcut | Kanbanzai (current) |
|-------|---------|--------|----------|---------------------|
| Strategic | Why are we doing this? | Initiative (nestable) | Milestone | **Missing** |
| Scoped work | What are we delivering? | Project | Epic | Plan |
| Executable | Who does what this week? | Issue | Story | Feature → Task |

Kanbanzai currently has the bottom two layers. The strategic layer is absent. Without it, there is no structured way to manage the long-term planning of a large system: no master roadmap, no milestone tracking, no way to gradually decompose a vision into actionable work.

### The cross-cutting execution problem

Consider a social media platform with plans for "Auth System", "Web Frontend", and "Mobile App". A human project manager wants to batch work that delivers end-to-end authentication across all three. Under the current model, this requires:

- Creating a new plan and moving features into it (destroying the architectural grouping), or
- Restructuring the plan tree to match execution order (losing the logical decomposition), or
- Accepting that execution cannot be tracked as a coherent unit

None of these are satisfactory. The fundamental issue is that **scope decomposition is a tree, but execution scheduling is a selection**. One entity cannot serve both purposes well.

### Research context

Analysis of Linear, Shape Up, WBS (Work Breakdown Structure), and program management methodologies confirms that separating "what to build" (scope/planning) from "what we're building now" (execution/scheduling) is a universal pattern in successful planning systems. Linear separates these as Initiatives (scope) and Projects/Cycles (execution). Shape Up separates them as shaped work (scope) and bets/cycles (execution). WBS separates them as deliverables (scope) and work packages (execution).

The key insight from this research: **the planning entity should be recursive** (to support arbitrary decomposition depth) while the **work entity should be flat or shallow** (to keep execution simple and cross-cutting).

---

## Design

### 1. The project singleton

A persistent, project-level configuration section provides context that guides all plans and batches. This is not a new entity type — it is project-level metadata.

**Location:** `.kbz/config.yaml` gains a `project` section (or a new `.kbz/project.yaml` file if config.yaml grows too large).

**Contents:**

```yaml
project:
  name: "Social Media Platform"
  vision: "work/_project/design-vision.md"
  architecture: "work/_project/design-architecture.md"
  constraints:
    - "Must support 100k concurrent users"
    - "All services deployable independently"
```

**Semantics:**

- `name` is the human-readable project name, used in dashboard headers and reports
- `vision` and `architecture` are paths to project-level documents in `work/_project/`
- `constraints` are short strings displayed as context when creating new plans or batches
- All fields are optional. A project with no singleton configured works exactly as Kanbanzai does today

**The `_project/` folder** (already defined in P37) holds project-level documents: vision, architecture, cross-cutting research, project-wide retrospectives. These documents use the existing document record system with `owner: project`.

### 2. The plan entity (new)

A plan is a recursive unit of planning. It represents *what needs to be built* — a strategic direction, a system decomposition, or a themed group of work. Plans can contain child plans, batches, or both.

**Identity:**

| Aspect | Value |
|--------|-------|
| Entity kind | `plan` |
| ID format | `P{prefix}{n}-{slug}` (e.g. `P1-social-platform`) — unchanged from current |
| Storage | `.kbz/state/plans/{id}.yaml` — unchanged from current |

The plan entity **reuses the current plan ID format**. Existing plan IDs (e.g. `P1-test-plan`) remain valid. The ID scheme does not change.

**Fields:**

```yaml
id: P1-social-platform
slug: social-platform
name: "Social Media Platform"
status: shaping          # new lifecycle, see below
summary: "End-to-end social media platform with web and mobile clients"
parent: ""               # optional: parent plan ID for nesting
design: "DOC-xxx"        # optional: design document reference
depends_on: []           # optional: other plan IDs this plan depends on (deferred)
order: 0                 # optional: sibling ordering within parent
tags: []
created: ...
created_by: ...
updated: ...
supersedes: ""
superseded_by: ""
```

**New and changed fields compared to the current plan:**

- `parent` (new) — optional reference to a parent plan ID, enabling recursive nesting. Null/empty means top-level plan.
- `depends_on` (new, deferred) — optional list of plan IDs this plan depends on. Creates a DAG of ordering constraints across the plan tree. Not required for MVP; included in the data model so it doesn't require a migration later.
- `order` (new) — optional integer for sibling ordering within a parent. Lower numbers sort first. Used by human project managers to control what gets worked on next. Defaults to 0.
- `status` — uses a new lifecycle (see §3) distinct from the current plan lifecycle, which moves to batch.

**Lifecycle:**

Plans have a planning-oriented lifecycle that reflects the maturity of the planning, not the execution of work:

```
idea → shaping → ready → active → done
```

| Status | Meaning |
|--------|---------|
| `idea` | A vague direction or aspiration. May have a rough design document or just a summary. Not yet decomposed. |
| `shaping` | Being actively refined. Design work in progress. May be partially decomposed into child plans or batches. |
| `ready` | Fully shaped. Design approved. Decomposed into batches (or child plans) that can be executed. |
| `active` | Work is in progress on one or more child batches. |
| `done` | All child batches and plans are complete. The strategic goal has been achieved. |

Terminal states: `superseded`, `cancelled` (reachable from any non-terminal state, same pattern as current plan).

**Entry state:** `idea`.

**Allowed transitions:**

- `idea` → `shaping`, `superseded`, `cancelled`
- `shaping` → `ready`, `idea` (backward: reshape), `superseded`, `cancelled`
- `ready` → `active`, `shaping` (backward: design revised), `superseded`, `cancelled`
- `active` → `done`, `shaping` (backward: scope changed), `superseded`, `cancelled`
- `done` → `superseded`, `cancelled`

The `idea` → `shaping` → `ready` progression mirrors Shape Up's "shaping" process: work starts vague, gets refined through design, and becomes concrete enough to execute.

**Document expectations by status:**

| Status | Expected documents |
|--------|--------------------|
| `idea` | None required. A summary suffices. |
| `shaping` | A design document (draft or approved). |
| `ready` | An approved design document. At least one child batch or child plan. |
| `active` | Same as ready. |
| `done` | Same as ready. Optionally a retrospective. |

These expectations are guidance, not enforced gates. Plans are human-managed; over-constraining them defeats the purpose of supporting gradual refinement.

**Nesting rules:**

- A plan can contain child plans, batches, or both
- There is no enforced depth limit (practically, 2–3 levels covers most projects)
- A plan cannot be its own ancestor (no cycles in the tree)
- A batch's parent is always a plan (or nothing); batches do not nest inside other batches

### 3. The batch entity (renamed from plan)

A batch is a unit of work. It is the current plan entity, renamed. It groups features for execution, has a lifecycle, holds documents, and coordinates AI agent work through features and tasks.

**Identity:**

| Aspect | Value |
|--------|-------|
| Entity kind | `batch` |
| ID format | `B{prefix}{n}-{slug}` (e.g. `B24-auth-system`) |
| Storage | `.kbz/state/batches/{id}.yaml` |
| Display in filenames | `B24-F3-spec-auth-flow.md` |

**The `B` prefix** replaces the current `P` prefix for what was previously a plan. This is the most visible change in the system: everywhere that currently says `P24` will say `B24` after migration.

**Fields:** Identical to the current plan entity, with one addition:

```yaml
id: B24-auth-system
slug: auth-system
name: "Auth System"
status: proposed         # current plan lifecycle, unchanged
summary: "OAuth2 and passcode authentication"
parent: "P1-social-platform"  # optional: parent plan ID
design: "DOC-xxx"        # optional: design document reference
tags: []
next_feature_seq: 4      # from P37: per-batch feature sequence counter
created: ...
created_by: ...
updated: ...
supersedes: ""
superseded_by: ""
```

- `parent` — optional reference to a plan ID. A batch with no parent is a standalone batch, exactly as standalone plans work today.

**Lifecycle:** The current plan lifecycle, unchanged:

```
proposed → designing → active → reviewing → done
```

With the existing shortcut (proposed → active) and terminal states (superseded, cancelled). All current lifecycle transitions, gates, and overrides apply.

**Document expectations:** Unchanged from the current plan. A batch can have its own design document, specification, dev-plan, review, and retrospective. If a batch has a parent plan, the parent's design document can satisfy the batch's design gate (inheriting design context upward).

**Feature relationship:** Features reference their parent batch via the `parent` field (currently pointing to a plan ID; will point to a batch ID after migration). The `Feature.parent` field type does not change — it remains a string containing the parent entity ID.

**Standalone batches:** A batch with no parent plan works identically to how a plan works today. Small projects that don't need the planning layer create batches directly, add features, and ship. There is no requirement to create a plan above a batch.

### 4. The relationship model

```
Project Singleton (.kbz/config.yaml or .kbz/project.yaml)
  │
  ├── documents: vision, architecture, constraints
  │
  └── Plans (recursive tree)
        │
        Plan: "Social Media Platform" (P1-social-platform)
        │  status: active
        │  design: platform-architecture.md
        │
        ├── Plan: "Backend Infrastructure" (P2-backend-infra)
        │     status: shaping
        │     design: backend-design.md
        │     │
        │     ├── Batch: "Auth System" (B24-auth-system)
        │     │     status: developing
        │     │     ├── Feature: B24-F1 (OAuth2)
        │     │     │     └── Tasks...
        │     │     └── Feature: B24-F2 (Passcode)
        │     │           └── Tasks...
        │     │
        │     └── Batch: "API Gateway" (B25-api-gateway)
        │           status: proposed
        │           └── Feature: B25-F1 (Rate Limiting)
        │
        └── Plan: "Web Frontend" (P3-web-frontend)
              status: idea
              (not yet decomposed)
```

**Entity ownership summary:**

| Parent | Child | Relationship |
|--------|-------|-------------|
| (none) | Plan | Top-level plan |
| Plan | Plan | Child plan (recursive nesting) |
| Plan | Batch | Executable work container within a plan |
| (none) | Batch | Standalone batch (no planning layer) |
| Batch | Feature | Feature belongs to a batch |
| Feature | Task | Task belongs to a feature |

### 5. Progress rollup

Progress rollup becomes recursive. The existing `ComputePlanRollup` and `ComputeFeatureRollup` functions extend as follows:

**Batch rollup** (formerly plan rollup, unchanged logic):
- Sum task estimates across all features in the batch
- Progress = sum of done task estimates
- Excluded: not-planned and duplicate tasks

**Plan rollup** (new):
- A plan's progress is the recursive sum of:
  - Direct child batch progress (using batch rollup)
  - Direct child plan progress (recursive plan rollup)
- A plan's total estimate is the recursive sum of child batch and child plan totals
- Progress percentage = progress / total (if total > 0)

**Status dashboard integration:**

- `status` with no ID shows the project overview, including top-level plans and their recursive progress
- `status` with a plan ID shows the plan dashboard: child plans, child batches, aggregated progress, attention items
- `status` with a batch ID shows the current plan dashboard (renamed), unchanged in functionality

### 6. Impact on P37 file naming and folders

The P37 design defines plan-first folder organisation. This design extends it:

**Batch folders** replace plan folders for work documents:

```
work/
  B24-auth-system/
    B24-design-auth-system.md
    B24-spec-auth-system.md
    B24-F1-spec-oauth-flow.md
    B24-F2-spec-passcode-auth.md
    B24-review-auth-system.md
  B25-api-gateway/
    B25-F1-spec-rate-limiting.md
```

**Plan folders** hold plan-level design documents:

```
work/
  P1-social-platform/
    P1-design-social-platform.md
    P1-research-competitive-analysis.md
  P2-backend-infra/
    P2-design-backend-infra.md
```

**Feature display IDs** change prefix from `P` to `B`:

| Before (P37) | After (this design) |
|--------------|-------------------|
| `P24-F3` | `B24-F3` |
| `P24-F3-spec-auth-flow.md` | `B24-F3-spec-auth-flow.md` |

The sequence counter (`next_feature_seq`) moves from the plan state file to the batch state file. Features belong to batches, not plans.

**Plan documents** follow the same filename template but with the plan ID:

```
{PlanID}-{type}-{slug}.md
```

For example: `P1-design-social-platform.md`, `P2-research-scaling-options.md`.

Plan documents do not have feature-scoped variants (plans don't contain features directly).

### 7. Migration strategy

The rename from "plan" to "batch" affects:

**Entity state files:**
- `.kbz/state/plans/` → `.kbz/state/batches/` (directory rename)
- Plan ID prefix `P{n}-` → `B{n}-` in all state files
- Entity kind field: `plan` → `batch`
- All `parent` references in features update from `P{n}-slug` to `B{n}-slug`

**Work document folders:**
- `work/P{n}-{slug}/` → `work/B{n}-{slug}/`
- All filenames within: `P{n}-` prefix → `B{n}-` prefix

**Code:**
- `model.Plan` struct becomes the basis for both `model.Plan` (new planning entity) and `model.Batch` (renamed current plan)
- `EntityKindPlan` constant and all references update
- `PlanStatus` type is retained for the batch lifecycle; a new `PlanPlanningStatus` (or similar) is added for the plan lifecycle
- `IsPlanID` / `ParsePlanID` functions gain `IsBatchID` / `ParseBatchID` equivalents
- All MCP tools that reference "plan" gain "batch" equivalents or are updated

**Coordination with P37:** This migration should be coordinated with the P37 file naming migration. If P37 has not yet migrated files from type-first to plan-first folders, the combined migration renames `plan` → `batch` and reorganises folders simultaneously, avoiding a double-migration.

**Backward compatibility:** During a transition period, the system should accept both `P{n}-slug` and `B{n}-slug` as batch identifiers, resolving `P{n}` to `B{n}` with a deprecation warning. This eases the transition for existing commit messages, documents, and muscle memory.

### 8. Interaction with existing features

**Worktrees:** Worktrees are created for features and bugs, not for plans or batches. No change needed.

**Document gates:** The existing gate system checks for approved documents on features and their parent batch (currently parent plan). This continues to work — the field name changes from checking `Feature.Parent` against plan IDs to checking against batch IDs, but the logic is identical.

**Design document inheritance:** When a batch has a parent plan, the plan's approved design document should satisfy the batch's design gate prerequisite. This extends the existing three-level document lookup (feature field → feature-owned docs → parent plan docs) to a four-level lookup (feature field → feature-owned docs → parent batch docs → grandparent plan docs).

**Estimation:** Batch estimation works exactly as current plan estimation. Plan estimation is a new recursive rollup (see §5).

**Decisions:** Decisions are attached to features, not to plans or batches. No change needed.

**Incidents and bugs:** These reference features, not plans or batches. No change needed.

**Knowledge system:** Knowledge entries may reference plan or batch IDs in `learned_from` fields. The migration updates these references.

---

## Alternatives Considered

### Alternative A: Recursive plan only (no separate batch entity)

Make the current plan entity self-referencing (add `parent` field) without introducing a new entity type or renaming anything.

**Trade-offs:**
- Simpler — one entity type, one lifecycle, one set of tools
- No migration of "plan" to "batch" needed
- Does not solve the cross-cutting execution problem: a sprint that pulls from multiple branches of the plan tree still requires awkward restructuring
- The plan entity remains overloaded — sometimes it's strategic, sometimes it's a sprint
- Design document ownership is ambiguous: does a "sprint plan" need a design document?

**Rejected because:** The conceptual difference between "what to build" (planning) and "what we're building now" (execution) is real and manifests in practical problems. One entity cannot serve both purposes cleanly, as evidenced by the current plan entity being used as an epic, a sprint, a meta-feature, and a phase interchangeably.

### Alternative B: Plan + Milestone (instead of Plan + Batch)

Use "milestone" as the name for the work-grouping entity.

**Trade-offs:**
- "Milestone" is a well-known term
- But it implies a point in time, not a span of work
- "A milestone of features" sounds unnatural
- "Break a plan into milestones" sounds like creating checkpoints, not work containers
- Milestones in Linear are sub-phases within a project, not the project itself

**Rejected because:** The word "milestone" carries connotations of a marker or checkpoint rather than a container of work. "Batch" better describes a collection of features grouped for execution.

### Alternative C: Plan + Sprint

Use "sprint" as the name for the work-grouping entity.

**Trade-offs:**
- Universally recognised from Scrum
- But implies time-boxing, which is not always the intent
- Calling a single-feature plan a "sprint" feels wrong
- Carries Scrum baggage that may not align with Kanbanzai's methodology

**Rejected because:** "Sprint" implies a fixed time window and a specific methodology. Kanbanzai batches are goal-boxed, not time-boxed. A batch might take a day or a month — the defining characteristic is that it's a coherent unit of work, not that it fits a two-week cycle.

### Alternative D: Keep "plan" for work, use "initiative" or "programme" above

Don't rename the current plan. Add a new entity above it called "initiative", "programme", or "goal".

**Trade-offs:**
- No rename migration needed
- Existing documents, IDs, and muscle memory preserved
- But "plan" continues to be ambiguous — is this plan a strategic direction or a work batch?
- "Initiative" and "programme" are enterprise jargon that conflicts with Kanbanzai's plain-language philosophy
- The word "plan" is the most natural name for the planning entity, and giving it to the work entity wastes it

**Rejected because:** "Plan" is the right word for the planning entity. Using it for the work entity and inventing jargon for the planning entity gets the naming backward. The rename has a one-time migration cost but permanently improves conceptual clarity.

---

## Decisions

### D1: Separate planning and execution into two entity types

**Decision:** Introduce a recursive "plan" entity for strategic planning and rename the current plan entity to "batch" for operational work grouping.

**Rationale:** The current plan entity is overloaded, serving as epic, sprint, meta-feature, and phase interchangeably. Separating planning (what to build) from execution (what we're building now) resolves ambiguity, enables cross-cutting work batches, and supports gradual refinement from vague ideas to concrete work. Research into Linear, Shape Up, WBS, and program management confirms this separation as a universal pattern in successful planning systems.

**Consequences:**
- A new entity type (plan) is added to the system
- The current plan entity is renamed to batch, with a new ID prefix (`B` instead of `P`)
- All existing plan state files, work folders, filenames, and references must be migrated
- The P37 file naming design updates from `P{n}` to `B{n}` for batch-scoped documents
- Tools, skills, and agent instructions must be updated to use the new terminology

### D2: Plans are recursive with a planning-oriented lifecycle

**Decision:** Plans can nest arbitrarily via a `parent` field. Plans use a new lifecycle (`idea` → `shaping` → `ready` → `active` → `done`) that reflects planning maturity, not execution progress.

**Rationale:** Recursive nesting allows the same entity type to represent a multi-year programme, a quarterly goal, or a focused subsystem design — determined by where it sits in the tree, not by its type. The planning lifecycle supports the gradual refinement that characterises real-world planning: ideas start vague and become concrete over time. No enforced depth limit is needed; practical use will naturally settle at 2–3 levels.

**Consequences:**
- Plan state files gain `parent`, `depends_on`, and `order` fields
- A new lifecycle state machine is needed for plans, separate from batches
- The status dashboard must render plan trees with recursive progress
- Validation must prevent cycles in the plan tree (no plan can be its own ancestor)

### D3: Batches retain all current plan functionality

**Decision:** The batch entity is the current plan entity, renamed. No functionality is removed. Batches can exist without a parent plan.

**Rationale:** The current plan entity works well as a unit of work. The rename gives it a precise definition without breaking existing workflows. Standalone batches (no parent plan) preserve the simple case: create a batch, add features, ship. The planning layer is additive, not mandatory.

**Consequences:**
- All current plan lifecycle states, gates, and overrides apply to batches
- Features reference batches via the `parent` field (previously referencing plans)
- The `next_feature_seq` counter moves to batch state files
- Standalone batches work identically to how standalone plans work today

### D4: The `B` prefix for batch IDs

**Decision:** Batch IDs use the prefix `B{n}-{slug}` (e.g. `B24-auth-system`), replacing the current `P{n}-{slug}` plan ID format for work entities.

**Rationale:** Visual distinction between plan IDs (`P1-...`) and batch IDs (`B24-...`) makes the entity type immediately apparent in conversation, filenames, and commit messages. The `B` prefix is memorable (batch) and short. Feature display IDs become `B24-F3` (batch 24, feature 3), which reads naturally.

**Consequences:**
- All existing plan IDs must be migrated from `P{n}` to `B{n}`
- P37 filename templates update: `B24-F3-spec-auth-flow.md`
- A transition period accepts both `P{n}` and `B{n}` for batch references
- Plan IDs retain the `P` prefix for the new planning entity

### D5: Project singleton for persistent context

**Decision:** A `project` section in `.kbz/config.yaml` holds project-level metadata: name, document references (vision, architecture), and constraints.

**Rationale:** Every plan and batch operates within a project context. Rather than creating a new entity type with a lifecycle, project-level context is configuration — it exists, it's referenced, it's updated by humans in documents. No state machine is needed for "the project." The `_project/` folder (P37) provides the document home.

**Consequences:**
- `.kbz/config.yaml` gains an optional `project` section
- The status dashboard can display project context in the overview
- Project-level documents use existing document records with `owner: project`
- Projects without a configured singleton work exactly as before

### D6: Design documents can live at any level

**Decision:** Both plans and batches can hold design documents. A batch can inherit design context from its parent plan. There is no enforcement of "a plan must exist above a batch if a design is present."

**Rationale:** Small projects should be able to create a batch with a design document and start working — exactly as they do today with plans. Large projects place strategic designs on plans and operational designs on batches. The document gate system already supports multi-level lookup; extending it by one level (batch → parent plan) is a natural evolution.

**Consequences:**
- Document gate evaluation gains one additional lookup level
- No new enforcement rules are needed
- The existing pattern of "check feature, then feature-owned docs, then parent docs" extends naturally

### D7: Separate prefix registries for plans and batches

**Decision:** `plan_prefixes` and `batch_prefixes` are maintained as separate registries in `.kbz/config.yaml`. They do not share a namespace.

**Rationale:** Prevents ID collisions between entity types that share short single-character prefixes. Separate registries make intent explicit and allow each entity type to be configured independently without risk of a prefix character being claimed by both types.

**Consequences:**
- `.kbz/config.yaml` schema gains two distinct registry keys: `plan_prefixes` and `batch_prefixes`
- Validation ensures a given prefix character is registered under the correct registry for its entity type
- The default plan prefix (`P`) and default batch prefix (`B`) are pre-registered in their respective registries

### D8: Independent sequence counters for plans and batches

**Decision:** Plans and batches maintain independent sequence counters. `P1` and `B1` can coexist without conflict.

**Rationale:** The prefix letter already distinguishes entity types unambiguously. A shared counter would produce unnecessarily large numbers with no benefit. Independent counters keep IDs short and human-friendly.

**Consequences:**
- Config or state tracks two separate counters (one for plans, one for batches)
- Existing plan sequence counter migrates to the batch counter; the plan counter starts from 1 after migration

---

## Open Questions

1. **Migration ordering with P37.** If P37's file migration has not yet run when this design is implemented, the two migrations should be combined into one. If P37 has already migrated to `P{n}` folders, a second migration from `P{n}` to `B{n}` is needed. The implementation plan should determine the current state and choose the appropriate path.

2. **Plan lifecycle gates.** Should plan transitions (e.g. `shaping` → `ready`) require approved documents, similar to feature gates? Recommendation: no enforcement for MVP. Plans are human-managed strategic documents; over-gating them discourages the gradual refinement this design is meant to support. Gates can be added later if needed.

3. **`depends_on` for plans.** The data model includes a `depends_on` field for plan-to-plan dependencies, but implementation is deferred. When implemented, should dependency satisfaction be a gate for plan transitions (e.g. a plan can't move to `active` if its dependency isn't `ready`)? Deferred to the dependency design.

---

## Dependencies

- **P37 File Names and Actions** (`work/design/p37-file-names-and-actions.md`) — this design extends P37's folder structure and filename templates. The `P` → `B` prefix change affects P37's feature display IDs, folder naming, and canonical filename template. Migration must be coordinated.
- **P1-DEC-006 (canonical file layout)** — the `.kbz/state/` vs `work/` separation is preserved. A new `.kbz/state/batches/` directory is created alongside the existing entity directories.
- **P1-DEC-021 (compact time-sorted IDs)** — TSID13 remains the canonical feature ID. The batch-scoped display ID (`B24-F3`) replaces the plan-scoped display ID (`P24-F3`).
- **Document gate system** (`internal/gate/`) — gate evaluation must extend to support the batch → parent plan lookup chain.
- **Estimation rollup** (`internal/service/estimation.go`) — `ComputePlanRollup` must become recursive for the new plan entity; a new `ComputeBatchRollup` (identical to the current `ComputePlanRollup`) handles batch-level rollup.
- **Status dashboard** (`internal/mcp/status_tool.go`) — must render plan trees, batch dashboards, and recursive progress aggregation.