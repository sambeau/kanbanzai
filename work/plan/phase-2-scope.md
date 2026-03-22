# Phase 2 Scope and Planning

- Status: draft
- Date: 2026-03-21
- Purpose: define Phase 2 scope boundaries, phasing, dependencies, and open questions
- Related:
  - `work/design/entity-structure-and-document-pipeline.md`
  - `work/design/document-centric-interface.md`
  - `work/design/document-intelligence-design.md`
  - `work/design/machine-context-design.md`
  - `work/spec/phase-1-specification.md` §7.1
  - `work/plan/phase-1-decision-log.md`

---

## 1. Purpose

This document defines the scope of Phase 2 — what it delivers, what it defers, how it is split, and what design questions must be resolved before specification can begin.

Phase 1 built the workflow kernel: entities, lifecycle enforcement, ID allocation, document scaffolding, MCP interface, and health checks. Phase 2 builds the document and intelligence layer on top of that kernel.

---

## 2. Phase 2 Split

Phase 2 is split into two sub-phases to manage risk and allow incremental validation.

### 2a: Entity Evolution, Document Management, and Document Intelligence

**Theme:** the system understands documents — their structure, their relationships, and their role in the design-to-delivery pipeline.

### 2b: Context Management and Agent Capabilities

**Theme:** the system assembles targeted context for agents and knowledge persists across sessions.

Phase 2b depends on Phase 2a. Context assembly requires the document intelligence layer to provide indexed fragments.

---

## 3. Phase 2a Scope

### 3.1 Entity model evolution

Source: `work/design/entity-structure-and-document-pipeline.md`

- **Plan replaces Epic.** The Plan coordinates design work, owns design documents, accumulates decisions, and births Features when designs become specifications. It uses a human-assigned ID with a project-defined prefix (`{X}{n}-{slug}`) rather than a fixed type name, allowing each project to name and categorise its design work according to its own conventions.
- **Prefix registry.** Each project declares its prefixes in `.kbz/config.yaml`, providing semantic names for display, validation against typos, and self-describing project conventions for agents. Agents discover the project's organisational vocabulary through MCP operations rather than requiring external SKILL files.
- **Feature entity changes.** Feature is born when a specification is carved out. Feature status is derived from document and task state. The `epic` field is renamed to `parent`.
- **Flat entity hierarchy with tags.** Plans do not nest. Organisational concerns (phases, milestones, teams, sprints) are handled through tags on entities and views/projections derived from canonical state.
- **Document metadata schema.** Documents (designs, specifications, dev plans) gain tracked lifecycle metadata: draft → review → approved → superseded. Documents are not separate entity types — they are documents with structured metadata owned by Plans or Features.
- **Migration tooling.** Epic → Plan migration, field renames, ID format changes, storage directory changes, prefix registry initialisation.
- **Plan lifecycle definition.** States, transitions, and constraints.

### 3.2 Document management

Source: `work/design/document-centric-interface.md`, `work/spec/phase-1-specification.md` §15

- **Document lifecycle operations.** Submit, review, approve, supersede, retrieve. Phase 1 has scaffolding and basic document support; Phase 2a makes the lifecycle fully operational.
- **Document-to-entity linking.** Specifications own Features. Designs own Plans. Dev plans own task decompositions. These links are explicit and tracked.
- **Verbatim retrieval.** Approved documents are returned exactly as approved. This exists conceptually in Phase 1; Phase 2a enforces it with tracked approval metadata.
- **Document versioning.** When a document is superseded, the previous version is retained. The supersession chain is queryable.
- **Document validation.** Type recognition, required frontmatter, required sections, naming conventions, referential integrity.

### 3.3 Document intelligence

Source: `work/design/document-intelligence-design.md` §18.2

- **Layer 1: structural skeleton.** Markdown parsing into section hierarchy. Deterministic, runs on every document change.
- **Layer 2: pattern-based extraction.** Entity references, cross-document links, front matter parsing, conventional section detection. Deterministic.
- **Layer 3: AI-assisted classification.** Fragment role classification (requirement, decision, rationale, constraint, etc.), concept extraction, section summaries. Non-deterministic, cached, runs once per document version at ingest time via agent.
- **Layer 4: document graph.** Persistent queryable graph built from Layers 1–3. Section-to-section relationships, concept connections, entity reference links.
- **Concept model.** Corpus-wide named concepts that enable vertical slicing ("show me everything about lifecycle transitions across all documents").
- **`doc_` MCP operations.** Outline, section, find_by_concept, find_by_role, find_by_entity, trace, impact, gaps, ingest, classify, pending.

### 3.4 Query and infrastructure

Source: P1-DEC-013, P1-DEC-020

- **Cache schema expansion.** Extend the SQLite cache to support documents, document metadata, and document graph queries.
- **Rich server-side query and filtering.** Move beyond list-by-type. Support filtering by status, parent, tags, date range, and other fields. Support cross-entity queries (e.g., "all tasks for features in this Plan") and tag-based queries (e.g., "all entities tagged `phase-2`").
- **Concurrency strategy.** Define and implement a concurrency model for multi-agent access. The SQLite cache provides local concurrency (WAL mode for concurrent reads, serialised writes). The Git state layer needs optimistic locking or equivalent (check file hash before write, fail if changed).

### 3.5 Phase 2a does NOT include

- Context profiles, context assembly, or context packing
- Knowledge contribution, confidence scoring, or knowledge lifecycle
- Usage reporting or retention policies
- Orchestration or agent delegation
- Git worktree management or branch tracking
- Cross-document consistency checking (deferred to Phase 3+; see `entity-structure-and-document-pipeline.md`)
- Entity nesting or recursive hierarchies (organisational concerns handled by tags instead)
- Embedding-based semantic search
- Concept synonym detection
- Automated re-classification on document change

---

## 4. Phase 2b Scope

### 4.1 Context management

Source: `work/design/machine-context-design.md` §15.2

- **Context profile definition.** Named YAML bundles with inheritance hierarchies that scope what knowledge each agent receives.
- **Context assembly.** Tiered retrieval with byte-based budgeting that composes design context (from document fragments) and implementation context (from knowledge entries) into targeted packets per agent per task.
- **Knowledge contribution.** Agents contribute knowledge entries during work, with deduplication on write.
- **Confidence scoring.** Wilson score lower bound for knowledge entry reliability.
- **Knowledge lifecycle.** Contribution → confirmation → staleness → retirement. TTL-based pruning, promotion triggers, post-merge compaction.
- **Usage reporting.** Bundled with task completion at low token cost.
- **`KnowledgeEntry` and `TeamMemoryEntry` tracked records.** These gain structured schemas and lifecycle management.

### 4.2 Agent capabilities

Source: P1-DEC-017, P1-DEC-018, P1-DEC-019

- **Link resolution.** Infer entity links from free-text references in documents and entity fields.
- **Duplicate detection.** Fuzzy or semantic duplicate detection for entity creation, using the cache and document intelligence layer.
- **Document-to-entity extraction guidance.** Structured prompts or protocols that help agents extract entities from approved documents more reliably.

### 4.3 Phase 2b does NOT include

- Orchestration or agent delegation (Phase 4)
- Git worktree management (Phase 4)
- Cross-project knowledge sharing (Phase 3+)
- Embedding-based semantic similarity for deduplication (Phase 3+ if needed)
- Automatic context assembly optimisation (Phase 3+)

---

## 5. Updated Phase Roadmap

| Phase | Name | Scope |
|-------|------|-------|
| 1 | Workflow Kernel | Entity model, validation, MCP interface, doc scaffolding, ID allocation, health checks, local cache ✅ |
| 2a | Document Intelligence | Entity evolution (Plan/Feature), document management, document intelligence (4-layer model), rich queries, concurrency |
| 2b | Context Management | Context profiles, assembly, knowledge lifecycle, link resolution, duplicate detection |
| 3 | Git Integration | Worktree management, branch tracking, merge readiness, PR integration |
| 4 | Orchestration | Decomposition, dependency-aware scheduling, fresh-session dispatch, worker review against spec |

Phase 3 (previously Phase 2 in the workflow design basis) shifts to accommodate the 2a/2b split. Git integration is valuable but less foundational than getting documents and context right.

---

## 6. Deferred Decisions Affecting Phase 2a

Five decisions from Phase 1 were explicitly deferred to Phase 2. Their disposition under this scoping:

| Decision | Topic | Phase 2a? | Notes |
|----------|-------|-----------|-------|
| P1-DEC-013 | Cache schema expansion | **Yes** | Required for document storage, rich queries, and document graph |
| P1-DEC-017 | Link resolution | **No (2b)** | Depends on document intelligence being operational |
| P1-DEC-018 | Duplicate detection | **No (2b)** | Depends on cache expansion and document intelligence |
| P1-DEC-019 | Document-to-entity extraction | **No (2b)** | Can be improved once document intelligence provides structure |
| P1-DEC-020 | Rich server-side query/filtering | **Yes** | Required for document and entity queries beyond list-by-type |

P1-DEC-013 and P1-DEC-020 must be resolved early in Phase 2a. They define the query and storage infrastructure that everything else depends on.

---

## 7. Dependencies Within Phase 2a

```
Entity model evolution (3.1)
  │
  ├── Plan, prefix registry, lifecycle, fields, MCP ops
  ├── Tag system for cross-cutting concerns
  ├── Document metadata schema
  ├── Feature status derivation
  │
  ▼
Document management (3.2)
  │
  ├── Document lifecycle operations
  ├── Document-to-entity linking
  ├── Versioning, validation
  │
  ▼
Document intelligence (3.3)
  │
  ├── Layers 1–2 (deterministic, can start early)
  ├── Layer 3 (AI-assisted, needs document management operational)
  ├── Layer 4 (document graph, needs Layers 1–3)
  │
  ▼
Query & infrastructure (3.4)
  │
  ├── Cache expansion (supports 3.2 and 3.3, start early)
  ├── Rich queries including tag-based queries (depends on cache schema)
  ├── Concurrency (design early, implement alongside)
```

The cache expansion (P1-DEC-013) cuts across everything — it supports document storage, document graph persistence, and rich queries. It should be one of the first things resolved and implemented.

Document intelligence Layers 1–2 (markdown parsing, pattern extraction) are deterministic and have no dependency on the AI classification protocol. They can be built early and validated independently.

---

## 8. Open Design Questions for Phase 2a

These must be resolved before a Phase 2a specification can be written.

### 8.1 Plan lifecycle states and transitions

The entity-structure design (§11.1) proposed tentative states: exploring → active → mature → closed. The exact states, permitted transitions, and constraints need definition. Key questions:

- Can a Plan move backward (e.g., `mature` → `active` if new design work is needed)?
- Does a Plan auto-transition based on Feature state, or is it manually managed?
- What happens to a Plan's Features when it is closed?

### 8.2 Document metadata schema

What fields does a tracked document record have? Tentative minimum:

- document identity (path, type, title)
- lifecycle status (draft, review, approved, superseded)
- ownership (which Plan or Feature owns this document)
- approval metadata (approved_by, approved_at)
- supersession (supersedes, superseded_by)
- version or content hash

This needs to be specified precisely, including validation rules.

### 8.3 Computed vs stored Feature status

The entity-structure design (§11.5) identified this as an open question. Options:

- **Pure computation:** Feature status is always derived from document and task state. No stored status field. Clean but may be slow and can't capture human judgement.
- **Stored with auto-update:** Feature has a stored status field that is automatically updated when document or task state changes. Pragmatic but needs a synchronisation mechanism.
- **Hybrid:** Status is stored but the system warns if stored status is inconsistent with document/task state. Flexible but adds complexity.

This must be resolved before the Feature entity schema can be finalised.

### 8.4 Document storage model

Where do tracked document records live? Options:

- Alongside entity YAML files in `.kbz/state/documents/`
- As metadata sidecar files next to the documents themselves
- In the cache only (derived, not in Git)

The document intelligence design (§13) defines index storage but not document registration storage. This needs a decision.

### 8.5 Plan storage model

Where do Plan YAML files live? Options:

- A single directory for all Plans regardless of prefix (e.g., `.kbz/state/plans/`)
- Prefix-specific directories (e.g., `.kbz/state/P/`, `.kbz/state/D/`)

The single directory is simpler. The prefix is already encoded in the ID and filename.

### 8.6 Tag schema

Tags are proposed as freeform strings. Questions:

- Should the system enforce any tag format (e.g., lowercase, hyphenated)?
- Should tags support optional namespacing (e.g., `phase:2`, `team:frontend`)?
- Should there be a tag registry or are tags truly freeform?

### 8.7 Prefix registry details

The prefix registry lives in `.kbz/config.yaml`. Questions:

- What validation rules apply to prefix characters? (Must be non-digit, single character — anything else?)
- Can prefixes be retired or renamed after entities have been created with them?
- What is the MCP operation for querying the registry?

### 8.8 Concurrency model

Multi-agent access to entity and document state needs a defined strategy. The cache (SQLite with WAL mode) handles read concurrency. Write concurrency options:

- **Optimistic locking:** read file, compute hash, write with hash check, fail if changed. Simple, no lock files.
- **Advisory file locking:** lock file per entity during writes. More complex, risk of stale locks.
- **Serialised writes through the MCP server:** all writes go through a single server instance that serialises them. Simplest if only one server runs at a time.

### 8.9 Migration strategy

How does the Phase 1 → Phase 2a migration work?

- Is it a one-time migration tool, or does the system detect and migrate on startup?
- What happens to existing Epic entities? They must be re-assigned to the new `{X}{n}-{slug}` format. The project must declare at least one prefix.
- What happens to existing Feature entities that have no spec document? They become Features with a null spec — is that valid?

### 8.10 Document intelligence implementation questions

From the document intelligence design (§17.2):

- **Graph storage format:** flat YAML edge list, or something more efficient?
- **Incremental re-classification:** whole document or changed sections only?
- **Classification stability:** does the system record which model produced a classification?
- **Index bootstrapping:** batch classification or incremental for existing document corpora?

---

## 9. Open Design Questions for Phase 2b

These do not need to be resolved for Phase 2a but should be considered to avoid precluding them.

### 9.1 Knowledge entry schema

What fields does a KnowledgeEntry have? The machine-context design defines a model but it needs to be formalised as an entity schema.

### 9.2 Context profile inheritance

How are inheritance conflicts resolved? The machine-context design (§6.6) says "more specific profiles override more general ones" but the exact merge semantics need definition.

### 9.3 Knowledge lifecycle boundaries

How much of the knowledge lifecycle ships in Phase 2b? The full model (contribution → confirmation → staleness → retirement, with Wilson scoring, TTL pruning, promotion, and post-merge compaction) is sophisticated. A minimum viable subset may be appropriate for initial delivery.

### 9.4 Bootstrap story for existing projects

When adopting Kanbanzai on a project with existing documents, how does the system bootstrap? Batch classification by a dedicated agent, or incremental classification as documents are touched? This affects the Phase 2b MVP significantly.

---

## 10. Risks

### 10.1 Scope creep within Phase 2a

Phase 2a is already substantial: entity evolution, document management, document intelligence, cache expansion, rich queries, and concurrency. There is a risk of discovering additional requirements during implementation that expand scope.

Mitigation: apply the same scope discipline as Phase 1. If a capability belongs more naturally to Phase 2b or later, defer it unless a decision explicitly promotes it.

### 10.2 Document intelligence complexity

The four-layer document intelligence model is the largest single system in Phase 2a. Layers 1–2 are deterministic and well-understood. Layer 3 (AI-assisted classification) and Layer 4 (document graph) are less certain — the classification protocol, concept model, and graph query semantics need implementation experience to validate.

Mitigation: build Layers 1–2 first and validate independently. Layer 3 can be introduced incrementally — the system degrades gracefully without classification (operations that require Layer 3 simply return less information).

### 10.3 Migration risk

Replacing Epic with Plans is a breaking change. Existing `.kbz/state/` directories, entity files, ID formats, and cross-references all need updating. The ID format change (`EPIC-*` → `{X}{n}-{slug}`) is more significant than a simple rename. If the migration is incomplete or buggy, the system becomes inconsistent.

Mitigation: build migration as a tested, repeatable operation. Run it on the project's own `.kbz/` state as a validation step (self-hosting).

### 10.4 Concurrency under-design

If the concurrency model is too simple, multiple agents will hit conflicts. If it's too complex, it adds implementation burden disproportionate to Phase 2a's needs.

Mitigation: start with the simplest viable model (serialised writes through a single MCP server instance). Document the constraint. Upgrade to optimistic locking if multi-server scenarios become real.

---

## 11. Next Steps

1. **Resolve open design questions (§8)** — particularly Plan lifecycle (§8.1), document metadata schema (§8.2), and computed vs stored status (§8.3). These gate the Phase 2a specification.
2. **Write the Phase 2a specification** — following the same structure as the Phase 1 specification, with acceptance criteria for each capability.
3. **Write the Phase 2a implementation plan** — work breakdown, sequencing, and dependency ordering.
4. **Begin Phase 2a implementation** — starting with cache expansion and entity model evolution, which are foundational.

Phase 2b planning can begin in parallel once Phase 2a implementation is underway, but the Phase 2b specification should wait until Phase 2a is operational enough to validate assumptions about document intelligence.

---

## 12. Summary

Phase 2 is split into two sub-phases:

- **Phase 2a** delivers entity model evolution (Plans with flexible prefixes replace Epic), document management with tracked lifecycle, the four-layer document intelligence backend, tags for cross-cutting organisational concerns, rich queries, and a concurrency model. It makes the system understand documents — their structure, relationships, and role in the design-to-delivery pipeline.

- **Phase 2b** delivers context management (profiles, assembly, knowledge lifecycle) and agent capabilities (link resolution, duplicate detection, extraction guidance). It makes the system assemble targeted context for agents and persist knowledge across sessions.

The design-to-delivery pipeline — design → specify → plan → implement → verify — is the structural backbone. Plans own the design space; Features own delivery. Documents bridge the two with tracked lifecycle. The entity hierarchy is flat — organisational concerns are handled through tags and views, not nesting. The document intelligence layer indexes and queries document content for both human navigation and agent context assembly.