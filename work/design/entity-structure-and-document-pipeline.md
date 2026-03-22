# Plan Entity and Document Pipeline

- Status: draft design
- Date: 2026-03-21
- Updated: 2026-03-22
- Purpose: define the structural relationship between workflow entities and design documents, resolving the deferred question of how documents and entities work together in Phase 2
- Related:
  - `work/design/workflow-design-basis.md` §8
  - `work/design/document-centric-interface.md` §8, §9
  - `work/spec/phase-1-specification.md` §7, §7.1
  - `work/design/machine-context-design.md` §15
  - `work/design/document-intelligence-design.md` §14, §18
  - `work/plan/phase-1-decision-log.md` P1-DEC-002

---

## 1. Purpose

This document defines the structural relationship between workflow entities and design documents in Kanbanzai. It resolves several questions deferred from Phase 1:

- How documents and workflow entities relate to each other
- What role the deferred types (`Specification`, `Dev Plan`, `Design`) play
- How the Plan entity is identified and named
- How cross-cutting organisational concerns (phases, teams, milestones) are handled without entity nesting

The answer shapes the Phase 2 entity model, the document pipeline, context assembly, and the MCP operation surface.

---

## 2. The Problem

Phase 1 built a workflow kernel with five entity types: Epic, Feature, Task, Bug, and Decision. It deferred twelve entity types, including `Specification`, `Dev Plan`, and `Design` as potential first-class entities.

Phase 1 also established the document-centric interface model: humans work with documents (designs, specifications, dev plans); agents mediate between documents and structured entities. But it left open the structural question: what exactly is the relationship between a Feature entity and the design, specification, and dev plan documents that define it?

Three sources gave different answers:

- The workflow design basis (§8.3) said specification, dev plan, and design are **document types, not entities** — they are the human-facing form and flow through the document lifecycle.
- The Phase 1 specification (§7.1) listed them as **deferred entity types**.
- P1-DEC-002 hedged: Feature's optional `spec` and `dev_plan` fields "can become foreign keys to separate entities in a future phase."

Additionally, naming Phase 1's "Epic" proved contentious. "Epic" implies size rather than purpose. This document resolves that problem with the **Plan** entity type — a name that captures the entity's design-coordination purpose — combined with user-defined ID prefixes that let each project categorise its Plans according to its own conventions.

---

## 3. Design Principles

1. **Documents and entities serve different purposes at different times.** During design, documents are primary — the designer thinks in documents. During implementation, entities are primary — agents work on tasks and track status. During review, both matter.

2. **The document pipeline is the feature lifecycle, viewed from the designer's perspective.** "This feature is in design" means "we have a design document in progress." "This feature is specified" means "we have an approved spec." The lifecycle is driven by document approvals.

3. **Entity types are distinguished by purpose, not size.** The system does not need "big features" and "small features." It needs entities with genuinely different lifecycles, ownership patterns, and structural roles.

4. **The design-to-delivery boundary is the key transition.** Design work (exploration, consideration, decision-making) is qualitatively different from delivery work (specification, planning, implementation, verification). The entity model should reflect this boundary.

5. **The entity hierarchy is flat.** Organisational concerns (phases, teams, milestones, sprints) are orthogonal to the design pipeline and are handled through tags, labels, and views — not through entity nesting.

6. **The system is non-prescriptive about how Plans are organised.** Different teams use Plans differently — as phases, tracks, feature groups, or deep-work areas. The system supports this diversity through user-defined ID prefixes and flexible naming rather than imposing a single vocabulary.

---

## 4. The Two-Entity-Type Model

### 4.1 Overview

The system has two entity types above Task, distinguished by purpose:

| Entity | Purpose | Phase of work | Primary owner |
|--------|---------|---------------|---------------|
| **Plan** | Coordinate a body of work; provide direction; organise Features | Direction and coordination | Human (designer) |
| **Feature** | Track delivery of a concrete piece of work through design, specification, planning, implementation, and verification | Design through delivery | Shared (human approves, agents implement) |

The Plan provides direction. The Feature is what gets designed and built.

The Plan has a **user-defined ID prefix** that lets each project name and categorise its Plans. See §4.6 for the ID scheme and §4.7 for the prefix registry. Throughout this document, examples use prefixes like `P`, `D`, `F`, but the system does not prescribe these — each project declares its own.

### 4.2 The Plan entity

The Plan is a coordination entity that organises a body of work, provides high-level direction, and groups related Features.

A Plan:

- has one special document (type: `design`) that serves as its roadmap and direction
- may own research reports and other exploratory material
- accumulates decisions made during its scope
- organises Features — they can be added, removed, or re-parented at any time
- does not have tasks, does not get "implemented"

A Plan's lifecycle is driven by its design document:

**States:** `proposed → designing → active → done`

**Transitions:**

- `proposed → designing` — Plan's design document is created (draft)
- `designing → active` — Plan's design document is approved
- `active → done` — manual, human judgment ("this body of work is complete")

**Terminal states:** `superseded` (replaced by another Plan), `cancelled` (dropped)

A Plan in `active` can have Features added, removed, or re-parented at any time. The `done` transition is manual closure — it is not derived from child Feature state. A Plan may remain `active` indefinitely.

### 4.3 Feature

A Feature is a concrete, independently deliverable piece of work. Features are born at `proposed` — potentially before any documents exist — and progress through a document-driven lifecycle.

A Feature:

- has up to three special documents: `design`, `specification`, `dev-plan`
- the design document is optional — straightforward Features can skip design
- owns tasks (born from the dev plan)
- may link to a parent Plan, or may exist without one
- tracks a document-driven lifecycle from proposal through delivery

**States:** `proposed → designing → specifying → dev-planning → developing → done`

**Transitions (forward):**

- `proposed → designing` — design document is created (draft)
- `proposed → specifying` — shortcut for straightforward Features that skip design
- `designing → specifying` — design document is approved
- `specifying → dev-planning` — specification is approved
- `dev-planning → developing` — dev plan is approved
- `developing → done` — all tasks are complete

**Terminal states:** `superseded` (replaced by another Feature), `cancelled` (dropped)

**Backward transitions** happen via document supersession, not explicit state changes:

- If an approved design is superseded → Feature reverts to `designing`
- If an approved spec is superseded → Feature reverts to `specifying`
- If an approved dev plan is superseded → Feature reverts to `dev-planning`

Feature status is stored but auto-updated. The rule: a Feature's status reflects the highest phase for which all prerequisite documents are approved (plus task completion for `done`).

### 4.4 Task

A Task is an implementation unit born from a Feature's dev plan. Tasks are not documents. They are operational entities that track bounded units of agent work.

Tasks remain as defined in Phase 1. No structural change is needed.

### 4.5 The full pipeline

```
Plan (P2-basic-ui, D3-auth-redesign, etc.)
  └── design document (roadmap / direction for the body of work)
  └── research reports
  └── decisions
  └── Features (created at proposed, assigned to Plan)
        │
Feature (design-through-delivery unit)
  ├── design document (optional — detailed design for this Feature)
  ├── specification document (defines scope)
  ├── dev plan document (decomposes work)
  └── tasks (implementation units)
        └── implementation
        └── verification
```

### 4.6 Plan ID scheme

Plans use a human-assigned ID format that is structurally distinct from all other entity types:

```
{X}{n}-{slug}
```

Where:

- `{X}` is exactly one character (any Unicode rune except a digit)
- `{n}` is one or more digits (a positive integer)
- `{slug}` is a lowercase alphanumeric slug with hyphens

Examples:

- `P2-basic-ui` — a "Phase" in a project that uses `P` for phases
- `D3-auth-redesign` — a "Design track" in a project that uses `D` for design tracks
- `F1-frontend-core` — a "Frontend track" in a project that uses `F` for frontend work
- `B5-api-layer` — a "Backend track" in a project that uses `B` for backend work
- `k12-setup-environment` — a lowercase prefix in a project that uses `k` for kikaku

This format is unambiguous with respect to all other entity types. Fixed entity types use multi-character alphabetic prefixes before the first hyphen (`FEAT-`, `TASK-`, `BUG-`, `DEC-`). Plan IDs always start with exactly one non-digit character followed by one or more digits followed by a hyphen. These patterns never collide.

The system identifies entity type from the ID pattern:

| Pattern | Entity type |
|---------|-------------|
| `FEAT-{id}` | Feature |
| `TASK-{id}` | Task |
| `BUG-{id}` | Bug |
| `DEC-{id}` | Decision |
| `{X}{n}-{slug}` (single non-digit char + digits + hyphen) | Plan |

No registry lookup is needed for type identification. The pattern match is sufficient.

### 4.7 Prefix registry

While the system can identify Plans by pattern alone, each project **must declare its prefixes** in project configuration. The registry serves three purposes:

**1. Semantic display.** The system can present meaningful names in output:

```
$ kbz status
Phases:
  P2-basic-ui          active    (3 features, 1 in-progress)
  P3-advanced-ui       designing (0 features)

Tracks:
  F1-frontend-core     active    (2 features, 2 done)
  B5-api-layer         active    (4 features, 1 blocked)
```

Without the registry, the system can only display a flat list of IDs with no grouping or semantic labels.

**2. Validation.** Undeclared prefixes are rejected, catching typos and preventing accidental proliferation:

```
Error: prefix 'Q' is not declared.
Declared prefixes: P (Phase), F (Frontend Track), B (Backend Track)
```

Adding a new prefix is a deliberate, team-visible act — a one-line config change committed to Git — not an accident.

**3. Self-describing project conventions.** Agents discover the project's organisational vocabulary through MCP operations rather than requiring external SKILL files or per-project instructions. Any agent — freshly spawned, no prior context — can query the prefix registry and understand the project's structure.

The registry is declared in project configuration:

```
# .kbz/config.yaml
prefixes:
  P:
    name: Phase
    description: "Wide slice of work, roughly milestone-scoped"
  F:
    name: Frontend Track
    description: "Deep frontend work, tightly coupled features"
  B:
    name: Backend Track
    description: "Backend and API work"
```

The registry defines the prefix character, a human-readable name, and an optional description. The system uses the name for display and reporting. The description is available to agents for context.

### 4.8 Why not recursive nesting

A natural question arises: if a project has phases and tracks, should a phase be able to contain tracks? Should Plans nest?

No. The entity hierarchy is flat. Plans are all peers, and Features are their children. Plans do not nest within other Plans.

The reasoning:

**Nesting solves an organisational problem at the cost of structural complexity.** Every level of nesting doubles the complexity of lifecycle propagation, status aggregation, and querying. "When is a phase done?" becomes "when all its tracks are done, which is when all their features are done, which is when all their tasks are done." This is the Jira/Azure DevOps hierarchy trap.

**Organisational concerns are orthogonal to the design pipeline.** "Which phase is this part of?" and "which team works on this?" are properties of entities, not structural relationships. A feature might be part of Phase 2 *and* be frontend work *and* be targeted for Q3. These are cross-cutting concerns that don't map to a single hierarchy.

**Tags and labels handle cross-cutting concerns better than nesting.** See §4.9.

### 4.9 Tags for cross-cutting concerns

Organisational concerns that don't fit the design pipeline — phases, milestones, teams, sprints, priorities — are handled through tags on entities.

A Plan or Feature can carry tags:

```
id: P2-basic-ui
tags:
  - phase-2
  - q3-2026
  - frontend
```

Tags are freeform strings. They are cheap, composable, and orthogonal. An entity can belong to multiple organisational categories simultaneously without requiring a containment hierarchy.

The system supports tag-based queries: "show me everything tagged `phase-2`" is a view, not a structural relationship. This is exactly what the projections category in the material taxonomy (`workflow-design-basis.md` §6.3) is for — generated views derived from canonical state.

Tags serve different organisational needs:

| Concern | Mechanism | Not this |
|---------|-----------|----------|
| What are we designing? | Plan (documents) | — |
| What are we delivering? | Feature (specification) | — |
| How are we building it? | Tasks (dev plan) | — |
| Which team works on it? | Tags on tasks or features | Entity hierarchy |
| What phase is it part of? | Tags on entities | Entity hierarchy |
| When does it ship? | Tags or milestone metadata | Entity hierarchy |
| What priority is it? | Tags or entity fields | Entity hierarchy |

---

## 5. The Document Pipeline as Feature Lifecycle

A Feature's lifecycle is driven by document approvals and task completion. The status reflects what work is currently happening:

| Current state | Feature status |
|---------------|----------------|
| Feature exists, no design work started | `proposed` |
| Design document exists (draft) | `designing` |
| Design approved, spec in progress | `specifying` |
| Spec approved, dev plan in progress | `dev-planning` |
| Dev plan approved, tasks in progress | `developing` |
| All tasks completed and verified | `done` |

Feature status is stored and auto-updated by document approvals. Backward transitions occur via document supersession (e.g., if an approved spec is superseded, the Feature reverts to `specifying`).

The minimal independent state on a Feature:

- `id`, `slug` — identity
- `parent` — parent Plan ID (nullable)
- `status` — current lifecycle state (auto-updated)
- `design` — link to design document (when it exists)
- `spec` — link to specification document (when it exists)
- `dev_plan` — link to dev plan document (when it exists)
- `created`, `created_by` — provenance
- `supersedes`, `superseded_by` — versioning
- `tags` — organisational metadata

---

## 6. The Document-Driven Feature Pipeline

### 6.1 The primary path

The normal flow is:

1. A Plan is created to coordinate a body of work. Its design document provides direction.
2. Features are created at `proposed` and assigned to the Plan.
3. Each Feature goes through its own document-driven lifecycle: design → specification → dev plan → implementation → verification.
4. Document approvals gate transitions between phases. The Feature's status auto-updates as documents are approved.

Design is optional per Feature. Straightforward Features can skip from `proposed` directly to `specifying`.

### 6.2 Scoping principle: one spec, one Feature

A specification should be scoped to a single independently deliverable piece of work. If a specification covers two independent things, it should be two specifications — and therefore two Features.

This is a scoping principle, not a hard system constraint. The system should encourage it (through guidance, documentation, and agent behaviour) but not enforce it mechanically.

### 6.3 The secondary path: bottom-up Features

Not all Features originate from Plan work. Some arise from:

- a bug investigation that reveals the need for a significant fix
- an operational need identified during implementation
- a quick improvement spotted during other work

For these cases, a Feature may be created at `proposed` without a parent Plan. Its `parent` field is null. It still follows the same document-driven lifecycle. It can be assigned to a Plan later if one is created to coordinate related work.

### 6.4 Plan-Feature relationship

The Plan is a coordination entity, not a strict container:

- Features can exist with or without a parent Plan.
- Features can be assigned to or moved between Plans at any time.
- A Plan's status is manually managed (for closure), not derived from child Feature state.
- Both "ideal" (Plan first, then Features) and "loose" (Features first, grouped into Plan later) patterns work with the same mechanism.

### 6.5 Cross-cutting documents

Some documents do not belong to any specific Plan or Feature:

- policies (commit policy, review policy, agent interaction protocol)
- project-level conventions
- cross-cutting design constraints

These are project-level documents. They exist outside the Plan → Feature hierarchy but are indexed and queryable by the document intelligence layer. They may inform context assembly for any task.

---

## 7. Effect on the Entity Model

### 7.1 Plan replaces Epic

The `Epic` entity type from Phase 1 is replaced by the Plan with user-defined prefixes. The structural role changes:

| Aspect | Epic (Phase 1) | Plan (Phase 2) |
|--------|----------------|-------------------------------|
| Purpose | Group related features | Coordinate a body of work, provide direction, organise Features |
| ID format | `EPIC-{slug}` (fixed prefix) | `{X}{n}-{slug}` (user-defined prefix) |
| Naming | Fixed ("Epic") | Project-defined via prefix registry |
| Lifecycle | Informal | `proposed → designing → active → done` (terminal: `superseded`, `cancelled`) |
| Owns | Feature references | Its design document (roadmap) + Feature references |
| Relationship to Features | Grouping (contains) | Coordination (organises); Features can be re-parented |
| Document role | None | Owns a design document that serves as roadmap/direction |
| Nesting | Not addressed | Explicitly flat; cross-cutting via tags |

Migration path: existing Epic entities become Plans. The `epic` field on Feature entities is renamed to `parent`. The project must declare at least one prefix in the registry. See §10 for migration details.

### 7.2 Feature gains document ownership

Feature retains its core Phase 1 role but gains explicit, tracked relationships to up to three special documents: design, specification, and dev plan. The `spec` and `dev_plan` fields (optional strings in Phase 1) become references to tracked document records with lifecycle metadata. A new `design` field is added for the optional design document.

### 7.3 Deferred entity types — resolved

The Phase 1 specification (§7.1) deferred twelve entity types. This design resolves the status of three:

| Deferred type | Resolution |
|---------------|------------|
| `Specification` | **Not a separate entity type.** A specification is a document with tracked lifecycle metadata, owned by a Feature. It does not need its own entity type — it is a document, not a workflow object. |
| `Dev Plan` | **Not a separate entity type.** A dev plan is a document with tracked lifecycle metadata, owned by a Feature. Same reasoning. (This was listed as `Plan` in the Phase 1 specification; that name now refers to the entity type that replaces Epic.) |
| `Design` | **Not a separate entity type.** A design is a document with tracked lifecycle metadata, owned by a Plan or a Feature. Same reasoning. |

These document types have lifecycle (`draft → approved → superseded`) and metadata (author, approval status, dates, links). But they are tracked as documents with structured metadata, not as workflow entities with their own MCP operations and lifecycle state machines. The document intelligence layer provides the indexing and querying capabilities.

The remaining nine deferred types (`Project`, `Milestone`, `Approval`, `Release`, `Incident`, `RootCauseAnalysis`, `ResearchNote`, `KnowledgeEntry`, `TeamMemoryEntry`) are unaffected by this design and remain deferred.

### 7.4 Document lifecycle

Documents owned by Plans and Features have their own lifecycle:

| Status | Meaning |
|--------|---------|
| `draft` | In progress; being written, discussed, and revised |
| `approved` | Human-approved; canonical; returned verbatim on retrieval |
| `superseded` | Replaced by a newer version; retained for history |

The `review` state is intentionally omitted. In an AI-mediated workflow, a document in `draft` is inherently in review — changes are discussed and applied conversationally, so there is no separate "waiting for review" state. The transition is directly from `draft` to `approved` when the human is satisfied.

This lifecycle is tracked as metadata on the document record, not as a separate entity. A document's approval status directly drives its owning entity's lifecycle transitions (see §5).

---

## 8. Effect on the Document-Centric Interface

The document-centric interface design (`document-centric-interface.md`) established the principle that humans work with documents and agents mediate. This design refines the model:

### 8.1 Document-to-entity mapping (revised)

| Document type | Home | Entity effect |
|---------------|------|---------------|
| Proposal | Plan or Feature | May create a Plan or Feature at `proposed` |
| Design (Plan level) | Plan | Roadmap/direction for the body of work; approval transitions Plan to `active` |
| Design (Feature level) | Feature | Detailed design for the Feature; approval transitions Feature to `specifying` |
| Specification | Feature (owned) | Approval transitions Feature to `dev-planning` |
| Dev plan | Feature (owned) | Approval transitions Feature to `developing`; decomposes the Feature into Tasks |
| Research report | Plan or project-level | May inform decisions; may create KnowledgeEntry records |
| User documentation | Feature (linked) | Documents the delivered feature |

### 8.2 The design-to-delivery boundary

The transition from design to delivery is a gradient within the Feature lifecycle, not a sharp structural boundary between entity types. A Feature progresses through design, specification, planning, and implementation — all within a single entity.

The Plan provides direction and coordination. The Feature owns the full design-through-delivery pipeline for its scope. Document approvals gate transitions between phases.

---

## 9. Effect on Context Assembly

The machine-context design (`machine-context-design.md`) defines how the system assembles targeted context for AI agents. The Plan → Feature structure affects context assembly:

- An agent working on a **Task** receives: the task definition, the relevant sections of its Feature's specification and dev plan, the Feature's design document (if it exists), relevant decisions from the parent Plan, and any applicable project-level policies.
- An agent working on **design** within a Feature receives: the Feature's design document, the parent Plan's design document (direction/roadmap), related research, decisions made so far, and relevant cross-cutting constraints.
- An agent **creating a specification** receives: the Feature's approved design document, the parent Plan's direction, relevant decisions, and examples of existing specifications in the project.
- An agent needing **project conventions** can query the prefix registry to understand the project's organisational structure without requiring external SKILL files.

Design context lives primarily in the Feature's own design document. The Plan's design document provides broader direction and coordination context. Implementation context stays within Feature.

---

## 10. Effect on Phase 1

### 10.1 Migration

Phase 1's entity model uses `Epic` where this design uses Plans with user-defined prefixes. The migration is:

- Existing Epic entities become Plans
- The `epic` field on Feature entities is renamed to `parent`
- The project must declare at least one prefix in `.kbz/config.yaml`
- Existing `EPIC-*` IDs must be re-assigned to the new `{X}{n}-{slug}` format
- Storage directory `.kbz/state/epics/` is renamed (see §11.7 for storage model)
- MCP operations `create_epic`, `list_epics`, etc. are replaced by Plan operations
- ID pattern matching logic is updated to recognise the `{X}{n}-{slug}` format

This is a breaking change relative to Phase 1 but is expected — Phase 1 explicitly anticipated entity model evolution.

### 10.2 Timing

This migration should occur at the beginning of Phase 2 implementation, before new entity types or document management features are built.

---

## 11. Open Questions

### 11.1 Plan lifecycle states — RESOLVED

Plan lifecycle: `proposed → designing → active → done`. Terminal states: `superseded`, `cancelled`.

- `proposed → designing` — Plan's design document is created
- `designing → active` — Plan's design document is approved
- `active → done` — manual, human judgment

A Plan does not auto-transition based on Feature state. Closure is a human decision.

### 11.2 Can a Feature change its parent? — RESOLVED

Yes. Re-parenting is allowed. A Feature can be moved between Plans at any time as an administrative operation.

### 11.3 Parent-less Features — RESOLVED

Yes. Features can exist without a parent Plan. The `parent` field is nullable. This supports the bottom-up secondary path (§6.3).

### 11.4 Document metadata schema

The exact schema for tracked document metadata (lifecycle status, approval, authorship, links) needs definition. This is related to the document intelligence design and should be specified alongside it.

### 11.5 Computed vs stored Feature status — RESOLVED

Feature status is stored and auto-updated by document approvals and task completion. This is a hybrid approach: the status field exists on the entity (avoiding recomputation on every read), but the system automatically updates it when document lifecycle events occur.

Backward transitions are handled via document supersession: if an approved document is superseded, the Feature's status reverts to the corresponding phase (e.g., superseding an approved spec reverts the Feature to `specifying`).

### 11.6 Bug and Decision entity relationships

Phase 1 defined Bug and Decision as standalone entities. Under this design:

- **Decisions** are born during design work and naturally belong to a Plan. They may also affect specific Features. The `affects` field on Decision should support references to both Plans and Features.
- **Bugs** are born during implementation and naturally belong to a Feature (via `origin_feature`). No structural change is needed, but Bugs should be able to trigger new design work within a Plan if the fix requires it.

### 11.7 Storage model for Plans

Plans need a storage directory. Options:

- A single directory for all Plans regardless of prefix (e.g., `.kbz/state/plans/`)
- Prefix-specific directories (e.g., `.kbz/state/P/`, `.kbz/state/D/`)

The single directory is simpler and avoids directory proliferation. The prefix is already encoded in the ID and filename.

### 11.8 Tag schema and conventions

Tags are freeform strings. Questions:

- Should the system enforce any tag format (e.g., lowercase, hyphenated)?
- Should there be a tag registry (like the prefix registry) or are tags truly freeform?
- Should tags be hierarchical (e.g., `team:frontend`, `phase:2`) or flat?

A lightweight approach: tags are freeform lowercase strings with optional namespacing via colon (e.g., `phase:2`, `team:frontend`). No registry required. The system indexes tags for querying but does not enforce a vocabulary.

### 11.9 Prefix registry location and format

The prefix registry is proposed to live in `.kbz/config.yaml`. Questions:

- Is this the right location? Should it be a separate file (e.g., `.kbz/prefixes.yaml`)?
- What validation rules apply to prefix characters? (Must be non-digit, single character — anything else?)
- Can prefixes be retired or renamed after entities have been created with them?

---

## 12. Relationship to Existing Designs

### 12.1 Workflow design basis

This design extends §8 (Object Model) of the workflow design basis. It resolves the open question in §8.4 about composite vs first-class modelling by taking a third path: documents are tracked with lifecycle metadata but are not workflow entities.

It also resolves the note in §8.3 that specification, dev plan, design, and research note are "document types rather than entities." This design agrees and makes the position concrete.

The material taxonomy (§6.3) defined "projections" as generated views derived from canonical state. This design uses that concept for organisational views (phase status, team dashboards) — these are projections over tagged entities, not structural relationships.

### 12.2 Document-centric interface

This design refines the internal model described in §8 of the document-centric interface design. The principle "fragment internally for consistency, present externally as whole documents" is preserved. The Plan → Feature structure provides the organisational backbone that the document-centric interface assumed but did not define.

### 12.3 Machine-context design

The context assembly model in the machine-context design can use the Plan → Feature hierarchy as a natural scoping mechanism. Design context is scoped to the Feature's own design document, with broader direction from the Plan. Implementation context stays within Feature. This is consistent with the tiered retrieval model described in that design.

The prefix registry also serves as a self-describing project convention mechanism, reducing the need for external SKILL files or per-project agent instructions. Agents discover the project's organisational vocabulary through MCP operations.

### 12.4 Document intelligence design

The document intelligence design provides the mechanism for indexing and querying documents within the Plan → Feature structure. The four-layer analysis model operates on documents regardless of which entity owns them. The document graph connects documents across Plans and Features through shared concepts and entity references.

### 12.5 Phase 1 specification

This design supersedes the entity model decisions in §7, §7.1, and §8 of the Phase 1 specification for Phase 2 purposes. Phase 1's model was explicitly designed to be evolved.

### 12.6 P1-DEC-002

P1-DEC-002 anticipated this decision: "Feature's optional spec and dev_plan fields can become foreign keys to separate entities in a future phase without breaking existing records." This design takes a slightly different path — spec and dev plan become references to tracked documents with lifecycle metadata, rather than to separate entity types — but the migration path P1-DEC-002 preserved remains valid.

---

## 13. Summary

The Kanbanzai entity model has two entity types above Task, distinguished by purpose:

- The **Plan** coordinates a body of work, provides direction through its design document, and organises Features. It uses a human-assigned ID with a project-defined prefix (`P2-basic-ui`, `D3-auth-redesign`, etc.), allowing each project to name and categorise its work according to its own conventions. Its lifecycle is `proposed → designing → active → done`, with `active → done` as a manual human decision.

- The **Feature** is the design-through-delivery entity. It is born at `proposed` and progresses through a document-driven lifecycle: `proposed → designing → specifying → dev-planning → developing → done`. Each phase transition is gated by document approval. It owns up to three special documents (design, specification, dev plan) and tasks. Design is optional — straightforward Features can skip from `proposed` to `specifying`.

Features can exist with or without a parent Plan. They can be assigned to or moved between Plans at any time. The Plan is a coordination entity, not a strict container.

The Plan ID format (`{X}{n}-{slug}`) is structurally distinct from all fixed entity types (`FEAT-`, `TASK-`, `BUG-`, `DEC-`) and requires no registry for type identification. However, each project must declare its prefixes in a registry that provides semantic names for display, validation against typos, and self-describing project conventions for agents.

The entity hierarchy is flat. Plans do not nest within other Plans. Organisational concerns that cut across the design pipeline — phases, milestones, teams, sprints — are handled through tags on entities and views/projections derived from canonical state. This prevents the system from recreating the hierarchical project-management structures it is designed to replace.

Documents (designs, specifications, dev plans) have their own tracked lifecycle (`draft → approved → superseded`) but are not workflow entities. The `review` state is omitted — in AI-mediated workflows, `draft` is inherently in review. Documents are tracked with structured metadata, owned by Plans or Features, indexed by the document intelligence layer, and queryable through MCP operations. Document approvals drive Feature lifecycle transitions; document supersession drives backward transitions.

This model preserves the design-to-implementation pipeline that produces high-quality software — design → specify → plan → implement → verify — while giving both the coordination phase (Plan) and the delivery phase (Feature) entity types that match their distinct purposes, and allowing each project to organise its work in whatever way makes sense for its team.