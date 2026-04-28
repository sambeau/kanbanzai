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
- **Feature entity changes.** Feature is born when a specification is carved out. Feature lifecycle: `proposed → designing → specifying → dev-planning → developing → done` (terminal: `superseded`, `cancelled`). Transitions are driven by document approvals; shortcut `proposed → specifying` for features that skip design. Feature status is stored as a field but automatically updated when document approvals or task completions trigger transitions. Features can change parent Plan (administrative re-parenting). The `epic` field is renamed to `parent`.
- **Flat entity hierarchy with tags.** Plans do not nest. Organisational concerns (phases, milestones, teams, sprints) are handled through tags on entities and views/projections derived from canonical state.
- **Document metadata schema.** Documents (designs, specifications, dev plans) gain tracked lifecycle metadata: `draft → approved → superseded`. The `review` state was dropped — in AI workflows, a document in `draft` is inherently in review. Documents are not separate entity types — they are documents with structured metadata owned by Plans or Features.
- **Migration tooling.** Epic → Plan migration, field renames, ID format changes, storage directory changes, prefix registry initialisation.
- **Plan lifecycle definition.** States: `proposed → designing → active → done` (terminal: `superseded`, `cancelled`). Transitions: `proposed→designing` (plan design doc created), `designing→active` (plan design doc approved), `active→done` (manual). A Plan in `active` can have Features added, removed, or re-parented at any time.

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

All Phase 2a design questions have been resolved. Decisions are recorded below.

### 8.1 Plan lifecycle states and transitions

**Resolved.** Plan lifecycle: `proposed → designing → active → done`. Terminal states: `superseded`, `cancelled`. Transitions: `proposed→designing` (plan design doc created), `designing→active` (plan design doc approved), `active→done` (manual). A Plan in `active` can have Features added, removed, or re-parented at any time. Plans do not auto-transition based on Feature state.

### 8.2 Document metadata schema

**Resolved.** Document record fields: `id` (format: `{owner-id}/{slug}`), `path` (relative path to file), `type` (enum: `design`, `specification`, `dev-plan`, `research`, `report`, `policy`), `title`, `status` (enum: `draft`, `approved`, `superseded`), `owner` (parent Plan or Feature ID), `approved_by`, `approved_at`, `content_hash` (SHA-256 of file at last registration/approval), `supersedes`, `superseded_by`, `created`, `created_by`, `updated`. No version counter — supersession chains handle versioning.

### 8.3 Computed vs stored Feature status

**Resolved.** Stored with auto-update. Feature status is stored as a field but automatically updated when document approvals or task completions trigger lifecycle transitions. Backward transitions happen when approved documents are superseded. The stored field is authoritative.

### 8.4 Document storage model

**Resolved.** `.kbz/state/documents/`, one YAML file per document record, tracked in Git. Document content stays at its real path (e.g., `work/design/foo.md`). The record is metadata only. Not sidecars (too scattered), not cache-only (must be durable and collaborative).

### 8.5 Plan storage model

**Resolved.** Single directory `.kbz/state/plans/` for all Plans regardless of prefix. The prefix is encoded in the ID and filename.

### 8.6 Tag schema

**Resolved.** Freeform lowercase strings with optional namespacing via colon (e.g., `phase:2`, `team:frontend`). No tag registry, no enforcement. The system indexes tags for querying but does not enforce a vocabulary.

### 8.7 Prefix registry details

**Resolved.** Lives in `.kbz/config.yaml` under `prefixes`. Prefix must be exactly one non-digit Unicode rune, case-sensitive, unique. Default prefix `P` (name: "Plan") is created on `kbz init` if no prefixes are declared. Retired prefixes marked `retired: true` (blocks new entity creation, existing entities remain valid). Prefixes cannot be renamed. MCP operation: `get_project_config` (returns full config including prefixes).

### 8.8 Concurrency model

**Resolved.** Optimistic locking. Read file, compute hash, write with hash check, fail-and-retry on conflict. SQLite WAL mode for cache read concurrency. No lock files, no single-server assumption.

### 8.9 Migration strategy

**Resolved.** One-time `kbz migrate phase-2` command, not auto-on-startup. Idempotent. Requires prefix registry to exist first. Renames Epic→Plan, `epic`→`parent`, `plan`→`dev_plan` on Features, moves files from `.kbz/state/epics/` to `.kbz/state/plans/`, re-assigns `EPIC-*` IDs to `{X}{n}-{slug}` format. Features without spec documents become Features with null `spec` (valid — bottom-up features).

### 8.10 Document intelligence implementation questions

**Resolved.** Flat YAML edge list for graph storage (migrate to SQLite if scale demands). Whole-document re-classification (section-level incremental is a future optimisation). Record model/version on each classification. Incremental bootstrapping (classify on register/change, not batch). Classifications are immutable once recorded — re-classification is an explicit manual operation (`re-ingest`), never automatic.

---

## 9. Open Design Questions for Phase 2b

These do not need to be resolved for Phase 2a but should be considered to avoid precluding them. All four questions have now been resolved.

### 9.1 Knowledge entry schema

**Resolved (partially).** The machine-context design (§13.1) defines a detailed draft schema. The scope decision (P2-DEC-001) confirms which fields are functional in Phase 2b vs informational-only. Formalising the schema as a spec-grade entity definition (exact required/optional fields, canonical field order, ID format) is a remaining task for the Phase 2b specification.

### 9.2 Context profile inheritance

**Resolved.** Leaf-level replace semantics, following the CSS cascade model. Child profiles completely replace parent values at each key — no deep merge, no list concatenation. If a child wants the parent's entries plus its own, it includes them explicitly. See P2-DEC-002.

### 9.3 Knowledge lifecycle boundaries

**Resolved.** Phase 2b ships a minimum viable subset: contribute/retrieve, deduplication on write, status lifecycle (contributed/confirmed/disputed/stale/retired), Wilson confidence scoring, usage reporting, and tier-dependent confidence filtering. Git anchoring, TTL-based pruning, automatic promotion triggers, and post-merge compaction are deferred to Phase 3. See P2-DEC-001.

### 9.4 Bootstrap story for existing projects

**Resolved.** Phase 2b includes a batch document import operation (`kbz import` / `batch_import_documents`) that submits existing files as document records with Layers 1–2 analysis. Classification remains incremental via `doc_pending` + `doc_classify`. Knowledge extraction from existing code is deferred to Phase 3. See P2-DEC-003. Additionally, `created_by` is auto-resolved from git config with local override to avoid placeholder attribution during bulk import. See P2-DEC-004.

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