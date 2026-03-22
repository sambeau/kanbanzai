# Entity Structure and Document Pipeline

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
- What role the deferred entity types (`Specification`, `Plan`, `Design`) play
- How the design-space entity is identified and named
- How cross-cutting organisational concerns (phases, teams, milestones) are handled without entity nesting

The answer shapes the Phase 2 entity model, the document pipeline, context assembly, and the MCP operation surface.

---

## 2. The Problem

Phase 1 built a workflow kernel with five entity types: Epic, Feature, Task, Bug, and Decision. It deferred twelve entity types, including `Specification`, `Plan`, and `Design` as potential first-class entities.

Phase 1 also established the document-centric interface model: humans work with documents (designs, specifications, plans); agents mediate between documents and structured entities. But it left open the structural question: what exactly is the relationship between a Feature entity and the design, specification, and plan documents that define it?

Three sources gave different answers:

- The workflow design basis (§8.3) said Specification, Plan, and Design are **document types, not entities** — they are the human-facing form and flow through the document lifecycle.
- The Phase 1 specification (§7.1) listed them as **deferred entity types**.
- P1-DEC-002 hedged: Feature's optional `spec` and `plan` fields "can become foreign keys to separate entities in a future phase."

Additionally, the naming of the design-space entity (Phase 1's "Epic") proved contentious. "Epic" implies size rather than purpose, and no single English word captures the concept of "a coordinated design effort that births features." This document resolves that problem structurally rather than linguistically.

---

## 3. Design Principles

1. **Documents and entities serve different purposes at different times.** During design, documents are primary — the designer thinks in documents. During implementation, entities are primary — agents work on tasks and track status. During review, both matter.

2. **The document pipeline is the feature lifecycle, viewed from the designer's perspective.** "This feature is in design" means "we have design documents but no spec." "This feature is specified" means "we have an approved spec." The lifecycle is derivable from document state.

3. **Entity types are distinguished by purpose, not size.** The system does not need "big features" and "small features." It needs entities with genuinely different lifecycles, ownership patterns, and structural roles.

4. **The design-to-delivery boundary is the key transition.** Design work (exploration, consideration, decision-making) is qualitatively different from delivery work (specification, planning, implementation, verification). The entity model should reflect this boundary.

5. **The entity hierarchy is flat.** Organisational concerns (phases, teams, milestones, sprints) are orthogonal to the design pipeline and are handled through tags, labels, and views — not through entity nesting.

6. **The system is non-prescriptive about design-space organisation.** Different teams use the design-space entity differently — as phases, tracks, feature groups, or deep-work areas. The system supports this diversity through flexible naming rather than imposing a single vocabulary.

---

## 4. The Two-Entity-Type Model

### 4.1 Overview

The system has two entity types above Task, distinguished by purpose:

| Entity | Purpose | Phase of work | Primary owner |
|--------|---------|---------------|---------------|
| **Design-space entity** | Coordinate design work; explore a problem space; birth features when designs become specifications | Design | Human (designer) |
| **Feature** | Track delivery of a concrete, specified piece of work through planning, implementation, and verification | Delivery | Shared (human approves, agents implement) |

The design-space entity is where design happens. A Feature is what gets built.

The design-space entity has a **user-defined ID prefix** rather than a fixed type name. See §4.6 for the ID scheme and §4.7 for the prefix registry. Throughout this document, examples use prefixes like `P`, `D`, `F`, but the system does not prescribe these — each project declares its own.

### 4.2 Design-space entity

The design-space entity is a coordinated effort to explore a problem area, produce design documents, make decisions, and eventually carve out specifications that define concrete deliverables.

A design-space entity:

- owns one or more design documents
- may own research reports and other exploratory material
- accumulates decisions made during design
- births Features when designs become specifications
- may birth multiple Features over its lifetime
- does not have tasks, does not get "implemented"

A design-space entity is active for as long as design work continues in its problem space. It may be long-lived. It is not a time-boxed container. It closes when the design space is considered mature and no further features are expected from it — or it may remain open indefinitely.

### 4.3 Feature

A Feature is born when a specification is carved out of a design-space entity. A Feature is a concrete, independently deliverable piece of work with a clear scope defined by its specification.

A Feature:

- is born from a specification (the primary path)
- owns exactly one specification document
- owns an implementation plan document (when planning begins)
- owns tasks (born from the plan)
- tracks delivery lifecycle: specifying → specified → planned → in-progress → done
- links back to its parent design-space entity for design context

A Feature does not exist until it has a specification, or at minimum a specification in progress. A Feature without a spec is a design idea that hasn't crossed the design-to-delivery boundary yet — it belongs in a design-space entity.

### 4.4 Task

A Task is an implementation unit born from a Feature's plan. Tasks are not documents. They are operational entities that track bounded units of agent work.

Tasks remain as defined in Phase 1. No structural change is needed.

### 4.5 The full pipeline

```
Design-space entity (P2-basic-ui, D3-auth-redesign, etc.)
  └── design documents (exploratory, iterative)
  └── research reports
  └── decisions
  └── births Features when designs → specs
        │
Feature (delivery unit)
  ├── specification document (defines scope)
  ├── implementation plan document (decomposes work)
  └── tasks (implementation units)
        └── implementation
        └── verification
```

### 4.6 Design-space entity ID scheme

Design-space entities use a human-assigned ID format that is structurally distinct from all other entity types:

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

This format is unambiguous with respect to all other entity types. Fixed entity types use multi-character alphabetic prefixes before the first hyphen (`FEAT-`, `TASK-`, `BUG-`, `DEC-`). Design-space entity IDs always start with exactly one non-digit character followed by one or more digits followed by a hyphen. These patterns never collide.

The system identifies entity type from the ID pattern:

| Pattern | Entity type |
|---------|-------------|
| `FEAT-{id}` | Feature |
| `TASK-{id}` | Task |
| `BUG-{id}` | Bug |
| `DEC-{id}` | Decision |
| `{X}{n}-{slug}` (single non-digit char + digits + hyphen) | Design-space entity |

No registry lookup is needed for type identification. The pattern match is sufficient.

### 4.7 Prefix registry

While the system can identify design-space entities by pattern alone, each project **must declare its prefixes** in project configuration. The registry serves three purposes:

**1. Semantic display.** The system can present meaningful names in output:

```
$ kbz status
Phases:
  P2-basic-ui          active    (3 features, 1 in-progress)
  P3-advanced-ui       exploring (0 features)

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

A natural question arises: if a project has phases and tracks, should a phase be able to contain tracks? Should design-space entities nest?

No. The entity hierarchy is flat. Design-space entities are all peers, and Features are their children. There is no nesting of design-space entities within other design-space entities.

The reasoning:

**Nesting solves an organisational problem at the cost of structural complexity.** Every level of nesting doubles the complexity of lifecycle propagation, status aggregation, and querying. "When is a phase done?" becomes "when all its tracks are done, which is when all their features are done, which is when all their tasks are done." This is the Jira/Azure DevOps hierarchy trap.

**Organisational concerns are orthogonal to the design pipeline.** "Which phase is this part of?" and "which team works on this?" are properties of entities, not structural relationships. A feature might be part of Phase 2 *and* be frontend work *and* be targeted for Q3. These are cross-cutting concerns that don't map to a single hierarchy.

**Tags and labels handle cross-cutting concerns better than nesting.** See §4.9.

### 4.9 Tags for cross-cutting concerns

Organisational concerns that don't fit the design pipeline — phases, milestones, teams, sprints, priorities — are handled through tags on entities.

A design-space entity or Feature can carry tags:

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
| What are we designing? | Design-space entity (documents) | — |
| What are we delivering? | Feature (specification) | — |
| How are we building it? | Tasks (plan) | — |
| Which team works on it? | Tags on tasks or features | Entity hierarchy |
| What phase is it part of? | Tags on entities | Entity hierarchy |
| When does it ship? | Tags or milestone metadata | Entity hierarchy |
| What priority is it? | Tags or entity fields | Entity hierarchy |

---

## 5. The Document Pipeline as Feature Lifecycle

A Feature's lifecycle status is derivable from its document and task state:

| Document/task state | Feature status |
|---------------------|----------------|
| Spec in progress (draft, not yet approved) | `specifying` |
| Spec approved | `specified` |
| Plan approved | `planned` |
| Tasks created, work in progress | `in-progress` |
| All tasks completed and verified | `done` |

This means the Feature entity does not need an independently managed status field that is manually kept in sync with document state. The status is a projection of document and task reality. The system computes it.

There is minimal independent state on a Feature that is not derivable from its documents and tasks:

- `id`, `slug` — identity
- `parent` — parent design-space entity ID
- `spec` — link to specification document
- `plan` — link to plan document (when it exists)
- `created`, `created_by` — provenance
- `supersedes`, `superseded_by` — versioning
- `tags` — organisational metadata

Status, progress, and readiness are computed, not stored.

---

## 6. The Spec-Births-Feature Principle

### 6.1 The primary path

The normal flow is:

1. A designer works within a design-space entity, producing design documents.
2. When a design is sufficiently mature, the designer (or an agent, with human approval) writes a specification.
3. The act of creating a specification births a Feature. The Feature is linked to the design-space entity and owns the specification.
4. The Feature then follows the delivery pipeline: plan → tasks → implementation → verification.

### 6.2 Scoping principle: one spec, one feature

A specification should be scoped to a single independently deliverable piece of work. If a specification covers two independent things, it should be two specifications — and therefore two features.

This is a scoping principle, not a hard system constraint. The system should encourage it (through guidance, documentation, and agent behaviour) but not enforce it mechanically.

### 6.3 The secondary path: bottom-up features

Not all features originate from design-space entity work. Some arise from:

- a bug investigation that reveals the need for a significant fix
- an operational need identified during implementation
- a quick improvement spotted during other work

For these cases, a Feature may be created directly with a specification, without a parent design-space entity. This is the secondary path. It is legitimate but should be the exception rather than the norm for substantial work.

If a bottom-up feature grows complex enough to need design exploration, the system should encourage creating a design-space entity to house that exploration rather than trying to do design work within a Feature.

### 6.4 Cross-cutting documents

Some documents do not belong to any specific design-space entity or Feature:

- policies (commit policy, review policy, agent interaction protocol)
- project-level conventions
- cross-cutting design constraints

These are project-level documents. They exist outside the design-space entity → Feature hierarchy but are indexed and queryable by the document intelligence layer. They may inform context assembly for any task.

---

## 7. Effect on the Entity Model

### 7.1 Design-space entity replaces Epic

The `Epic` entity type from Phase 1 is replaced by the design-space entity with user-defined prefixes. The structural role changes:

| Aspect | Epic (Phase 1) | Design-space entity (Phase 2) |
|--------|----------------|-------------------------------|
| Purpose | Group related features | Coordinate design work, birth features |
| ID format | `EPIC-{slug}` (fixed prefix) | `{X}{n}-{slug}` (user-defined prefix) |
| Naming | Fixed ("Epic") | Project-defined via prefix registry |
| Lifecycle | Informal | Design-oriented: exploring → active → mature → closed |
| Owns | Feature references | Design documents, research, decisions |
| Relationship to features | Grouping (contains) | Generative (births) |
| Document role | None | Primary home for design documents |
| Nesting | Not addressed | Explicitly flat; cross-cutting via tags |

Migration path: existing Epic entities become design-space entities. The `epic` field on Feature entities is renamed to `parent`. The project must declare at least one prefix in the registry. See §10 for migration details.

### 7.2 Feature gains document ownership

Feature retains its core Phase 1 role but gains explicit, tracked relationships to its specification and plan documents. The `spec` and `plan` fields (optional strings in Phase 1) become references to tracked document records with lifecycle metadata.

### 7.3 Deferred entity types — resolved

The Phase 1 specification (§7.1) deferred twelve entity types. This design resolves the status of three:

| Deferred type | Resolution |
|---------------|------------|
| `Specification` | **Not a separate entity type.** A specification is a document with tracked lifecycle metadata, owned by a Feature. It does not need its own entity type — it is a document, not a workflow object. |
| `Plan` | **Not a separate entity type.** An implementation plan is a document with tracked lifecycle metadata, owned by a Feature. Same reasoning. |
| `Design` | **Not a separate entity type.** A design is a document with tracked lifecycle metadata, owned by a design-space entity. Same reasoning. |

These document types have lifecycle (draft → review → approved → superseded) and metadata (author, approval status, dates, links). But they are tracked as documents with structured metadata, not as workflow entities with their own MCP operations and lifecycle state machines. The document intelligence layer provides the indexing and querying capabilities.

The remaining nine deferred types (`Project`, `Milestone`, `Approval`, `Release`, `Incident`, `RootCauseAnalysis`, `ResearchNote`, `KnowledgeEntry`, `TeamMemoryEntry`) are unaffected by this design and remain deferred.

### 7.4 Document lifecycle

Documents owned by design-space entities and Features have their own lifecycle:

| Status | Meaning |
|--------|---------|
| `draft` | In progress, not yet submitted for review |
| `review` | Submitted for human review |
| `approved` | Human-approved; canonical; returned verbatim on retrieval |
| `superseded` | Replaced by a newer version; retained for history |

This lifecycle is tracked as metadata on the document record, not as a separate entity. A document's approval status directly affects its owning entity's computed state (an approved spec means the Feature is at least `specified`).

---

## 8. Effect on the Document-Centric Interface

The document-centric interface design (`document-centric-interface.md`) established the principle that humans work with documents and agents mediate. This design refines the model:

### 8.1 Document-to-entity mapping (revised)

| Document type | Home | Entity effect |
|---------------|------|---------------|
| Proposal | Design-space entity | May create the design-space entity itself; may surface initial design questions |
| Draft design | Design-space entity | Iterates on design thinking within the design-space entity |
| Design | Design-space entity | Finalises design thinking; may trigger spec readiness |
| Specification | Feature (owned) | Births the Feature; defines its scope |
| Implementation plan | Feature (owned) | Decomposes the Feature into Tasks |
| Research report | Design-space entity or project-level | May inform decisions; may create KnowledgeEntry records |
| User documentation | Feature (linked) | Documents the delivered feature |

### 8.2 The design-to-delivery boundary

The transition from design-space entity to Feature is the key structural boundary. It occurs when:

1. A design within a design-space entity is judged mature enough to specify.
2. A specification document is created (draft status).
3. A Feature entity is created, linked to the design-space entity and owning the specification.
4. The specification goes through review and approval.

This is an agent-mediated process: the agent recognises that design work has reached spec-readiness, proposes creating a specification and birthing a feature, and the human approves.

---

## 9. Effect on Context Assembly

The machine-context design (`machine-context-design.md`) defines how the system assembles targeted context for AI agents. The design-space entity → Feature structure affects context assembly:

- An agent working on a **Task** receives: the task definition, the relevant sections of its Feature's specification and plan, relevant decisions from the parent design-space entity, and any applicable project-level policies.
- An agent working on **design** within a design-space entity receives: the entity's design documents, related research, decisions made so far, and relevant cross-cutting constraints.
- An agent **creating a specification** receives: the mature design documents from the design-space entity, relevant decisions, and examples of existing specifications in the project.
- An agent needing **project conventions** can query the prefix registry to understand the project's organisational structure without requiring external SKILL files.

The design-space entity → Feature hierarchy provides a natural scoping mechanism for context assembly. Design context flows down from the design-space entity; implementation context stays within Feature.

---

## 10. Effect on Phase 1

### 10.1 Migration

Phase 1's entity model uses `Epic` where this design uses design-space entities with user-defined prefixes. The migration is:

- Existing Epic entities become design-space entities
- The `epic` field on Feature entities is renamed to `parent`
- The project must declare at least one prefix in `.kbz/config.yaml`
- Existing `EPIC-*` IDs must be re-assigned to the new `{X}{n}-{slug}` format
- Storage directory `.kbz/state/epics/` is renamed (see §11.4 for storage model)
- MCP operations `create_epic`, `list_epics`, etc. are replaced by design-space entity operations
- ID pattern matching logic is updated to recognise the `{X}{n}-{slug}` format

This is a breaking change relative to Phase 1 but is expected — Phase 1 explicitly anticipated entity model evolution.

### 10.2 Timing

This migration should occur at the beginning of Phase 2 implementation, before new entity types or document management features are built.

---

## 11. Open Questions

### 11.1 Design-space entity lifecycle states

The exact lifecycle states for a design-space entity need definition. A tentative model:

- `exploring` — initial design work, not yet focused
- `active` — focused design work, may be birthing features
- `mature` — design space well-understood, primarily birthing/supporting features
- `closed` — no further design work expected

The transitions and constraints need to be specified. Key questions:

- Can a design-space entity move backward (e.g., `mature` → `active` if new design work is needed)?
- Does a design-space entity auto-transition based on Feature state, or is it manually managed?
- What happens to a design-space entity's Features when the entity is closed?

### 11.2 Can a Feature change its parent?

If a Feature was born from one design-space entity but turns out to belong more naturally to another, can it be re-parented? Probably yes, as an administrative operation, but the constraints need definition.

### 11.3 Parent-less Features

The secondary path (§6.3) allows Features without a parent design-space entity. These Features simply have a null `parent` field.

### 11.4 Document metadata schema

The exact schema for tracked document metadata (lifecycle status, approval, authorship, links) needs definition. This is related to the document intelligence design and should be specified alongside it.

### 11.5 Computed vs stored Feature status

§5 proposes that Feature status is computed from document and task state rather than independently stored. This is elegant but has implications:

- Computing status requires querying document and task state, which may be slower than reading a stored field.
- The Phase 1 entity model stores status as a field. Changing to computed status is a significant architectural shift.
- Some status transitions may involve judgement (e.g., "is this feature really done?") that pure computation cannot capture.

The tradeoffs between computed and stored status need to be evaluated during implementation planning. A hybrid approach — stored status that is automatically updated when document or task state changes — may be more practical than pure computation.

### 11.6 Bug and Decision entity relationships

Phase 1 defined Bug and Decision as standalone entities. Under this design:

- **Decisions** are born during design work and naturally belong to a design-space entity. They may also affect specific Features. The `affects` field on Decision should support references to both design-space entities and Features.
- **Bugs** are born during implementation and naturally belong to a Feature (via `origin_feature`). No structural change is needed, but Bugs should be able to trigger new design work within a design-space entity if the fix requires it.

### 11.7 Storage model for design-space entities

Design-space entities need a storage directory. Options:

- A single directory for all design-space entities regardless of prefix (e.g., `.kbz/state/props/` or `.kbz/state/designs/`)
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

It also resolves the note in §8.3 that Specification, Plan, Design, and ResearchNote are "document types rather than entities." This design agrees and makes the position concrete.

The material taxonomy (§6.3) defined "projections" as generated views derived from canonical state. This design uses that concept for organisational views (phase status, team dashboards) — these are projections over tagged entities, not structural relationships.

### 12.2 Document-centric interface

This design refines the internal model described in §8 of the document-centric interface design. The principle "fragment internally for consistency, present externally as whole documents" is preserved. The design-space entity → Feature structure provides the organisational backbone that the document-centric interface assumed but did not define.

### 12.3 Machine-context design

The context assembly model in the machine-context design can use the design-space entity → Feature hierarchy as a natural scoping mechanism. Design context is scoped to design-space entities; implementation context is scoped to Features. This is consistent with the tiered retrieval model described in that design.

The prefix registry also serves as a self-describing project convention mechanism, reducing the need for external SKILL files or per-project agent instructions. Agents discover the project's organisational vocabulary through MCP operations.

### 12.4 Document intelligence design

The document intelligence design provides the mechanism for indexing and querying documents within the design-space entity → Feature structure. The four-layer analysis model operates on documents regardless of which entity owns them. The document graph connects documents across design-space entities and Features through shared concepts and entity references.

### 12.5 Phase 1 specification

This design supersedes the entity model decisions in §7, §7.1, and §8 of the Phase 1 specification for Phase 2 purposes. Phase 1's model was explicitly designed to be evolved.

### 12.6 P1-DEC-002

P1-DEC-002 anticipated this decision: "Feature's optional spec and plan fields can become foreign keys to separate entities in a future phase without breaking existing records." This design takes a slightly different path — spec and plan become references to tracked documents with lifecycle metadata, rather than to separate entity types — but the migration path P1-DEC-002 preserved remains valid.

---

## 13. Summary

The Kanbanzai entity model has two entity types above Task, distinguished by purpose:

- The **design-space entity** coordinates design work, owns design documents, accumulates decisions, and births Features when designs become specifications. It uses a human-assigned ID with a project-defined prefix (`P2-basic-ui`, `D3-auth-redesign`, etc.), allowing each project to name and categorise its design work according to its own conventions.

- The **Feature** is the delivery entity. It is born when a specification is carved out of a design-space entity. It owns a specification, a plan, and tasks. It tracks delivery from specification through verification.

The design-space entity ID format (`{X}{n}-{slug}`) is structurally distinct from all fixed entity types (`FEAT-`, `TASK-`, `BUG-`, `DEC-`) and requires no registry for type identification. However, each project must declare its prefixes in a registry that provides semantic names for display, validation against typos, and self-describing project conventions for agents.

The entity hierarchy is flat. Design-space entities do not nest within other design-space entities. Organisational concerns that cut across the design pipeline — phases, milestones, teams, sprints — are handled through tags on entities and views/projections derived from canonical state. This prevents the system from recreating the hierarchical project-management structures it is designed to replace.

The document pipeline — design → specification → plan → implementation → verification — is the bridge between design-space entities and Features. Designers work in design-space entities, producing design documents. When a design is mature enough to specify, the specification births a Feature. The Feature then follows the delivery pipeline through planning, implementation, and verification.

Documents (designs, specifications, plans) have their own tracked lifecycle (draft → review → approved → superseded) but are not workflow entities. They are documents with structured metadata, owned by design-space entities or Features, indexed by the document intelligence layer, and queryable through MCP operations.

This model preserves the design-to-implementation pipeline that produces high-quality software — design → specify → plan → implement → verify — while giving both the design phase and the delivery phase entity types that match their distinct purposes, and allowing each project to organise its design work in whatever way makes sense for its team.