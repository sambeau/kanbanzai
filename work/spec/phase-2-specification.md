# Phase 2 Specification: Document Intelligence and Context Management

- Status: specification draft
- Purpose: define the required behavior, scope, constraints, and acceptance criteria for Phase 2 of the workflow system
- Date: 2026-03-22
- Based on:
  - `work/spec/phase-1-specification.md`
  - `work/design/entity-structure-and-document-pipeline.md`
  - `work/design/document-centric-interface.md`
  - `work/design/document-intelligence-design.md`
  - `work/design/machine-context-design.md`
  - `work/plan/phase-2-scope.md`
  - `work/plan/phase-1-decision-log.md`

---

## 1. Purpose

This specification defines Phase 2 of the workflow system.

Phase 1 built the workflow kernel: entities, lifecycle enforcement, ID allocation, document scaffolding, MCP interface, and health checks. Phase 2 builds the document and intelligence layer on top of that kernel.

Phase 2 is split into two sub-phases:

- **Phase 2a** delivers entity model evolution, document management with tracked lifecycle, the document intelligence backend, rich queries, and a concurrency model. It makes the system understand documents — their structure, relationships, and role in the design-to-delivery pipeline.

- **Phase 2b** delivers context management and agent capabilities. It makes the system assemble targeted context for agents and persist knowledge across sessions.

This specification defines **what Phase 2 must do**, not how it must be implemented.

---

## 2. Goals

Phase 2 must extend the workflow kernel so that:

1. The entity model reflects the design-to-delivery pipeline, with Plans coordinating bodies of work and Features tracking delivery through a document-driven lifecycle
2. Documents are first-class managed objects with tracked lifecycle metadata, not just scaffolded files
3. The system can parse, index, and query the structural content of design documents
4. AI agents can classify document fragments and the system stores those classifications persistently
5. Rich queries — by status, parent, tags, and cross-entity relationships — replace the Phase 1 list-by-type model
6. Multiple agents can work concurrently without silent data loss
7. Context can be assembled and targeted to specific agents based on their role and current task (Phase 2b)
8. Knowledge persists across agent sessions and accumulates over the life of the project (Phase 2b)

---

## 3. Non-Goals

Phase 2 does **not** aim to deliver:

- Orchestration or agent delegation (Phase 4)
- Git worktree management or branch tracking (Phase 3)
- PR integration or merge readiness checking (Phase 3)
- Cross-project knowledge sharing
- Embedding-based semantic search or vector similarity
- Concept synonym detection
- Automated re-classification on document change (classifications are explicit)
- Cross-document consistency checking beyond structural gap detection
- Entity nesting or recursive hierarchies
- Incident or root cause analysis entities
- Release management
- Visualisation of the document graph for human consumption

Phase 2 must remain focused on document understanding and context assembly.

---

## 4. Design Principles for Phase 2

Phase 2 inherits all Phase 1 design principles (§4.1–4.8 of the Phase 1 specification). The following additional principles apply.

### 4.1 Documents drive entity lifecycle

A Feature's lifecycle state is determined by the approval status of its documents and the completion of its tasks. Document approvals gate forward transitions; document supersession drives backward transitions. The system does not require explicit status commands for transitions that are implied by document events.

### 4.2 Entity types are distinguished by purpose, not size

The system has two entity types above Task — Plan and Feature — distinguished by their role in the design-to-delivery pipeline. Plans coordinate; Features deliver. There is no "big feature" or "small feature" distinction.

### 4.3 The entity hierarchy is flat

Plans do not nest within other Plans. Organisational concerns that cut across the design pipeline — phases, milestones, teams, sprints — are handled through tags on entities and views derived from canonical state, not through entity nesting.

### 4.4 The tool is a database; agents provide intelligence

The document intelligence backend is a structural index with a schema and query engine. It does not call LLMs. Agents populate it with classifications at ingest time. The tool provides structure; agents provide judgement. This separation keeps the tool deterministic and testable.

### 4.5 Graceful degradation by layer

Document intelligence operates in layers. Operations that require only structural parsing (Layers 1–2) work immediately. Operations that require AI-assisted classification (Layer 3) work once an agent has classified the document. The system is always useful at whatever level of analysis is available.

### 4.6 Non-prescriptive organisation

The system does not prescribe how Plans are organised. Different projects use Plans differently — as phases, tracks, feature groups, or deep-work areas. The system supports this diversity through user-defined ID prefixes and flexible naming rather than imposing a single vocabulary.

---

## 5. Scope

### 5.1 Phase 2a scope

Phase 2a delivers:

- Entity model evolution: Plan replaces Epic; prefix registry; Feature gains document ownership and document-driven lifecycle; document metadata records; tags for cross-cutting concerns
- Document management: submit, approve, supersede, retrieve with tracked lifecycle; document-to-entity linking; verbatim retrieval enforcement; document versioning through supersession
- Document intelligence: four-layer analysis model (structural skeleton, pattern-based extraction, AI-assisted classification, document graph); concept model and concept registry; classification protocol
- Query and infrastructure: cache schema expansion for documents and document graph; rich server-side query and filtering by status, parent, tags, date range; cross-entity queries; tag-based queries
- Concurrency: optimistic locking for Git-state writes; SQLite WAL for cache concurrency
- Migration: one-time Epic → Plan migration tool

### 5.2 Phase 2b scope

Phase 2b delivers:

- Context management: context profile definition; context assembly with byte-based budgeting; tiered retrieval composing design context and implementation context
- Knowledge lifecycle: knowledge contribution by agents; deduplication on write; confidence scoring; contribution → confirmation → staleness → retirement lifecycle; TTL-based pruning
- Agent capabilities: link resolution from free-text references; duplicate detection for entity creation; document-to-entity extraction guidance

### 5.3 Excluded from Phase 2

- Orchestration, decomposition, or agent delegation
- Git worktree management or branch tracking
- Cross-project knowledge sharing
- Embedding-based semantic similarity
- Automatic context assembly optimisation
- Automated re-classification without explicit agent intervention

---

## 6. Phase 2a Entity Model

### 6.1 Plan replaces Epic

The Epic entity type from Phase 1 is replaced by the Plan entity type. A Plan coordinates a body of work, provides direction through its design document, and organises Features.

Plans use a human-assigned ID format that is structurally distinct from all other entity types:

```
{X}{n}-{slug}
```

Where:

- `{X}` is exactly one non-digit Unicode rune (the prefix character)
- `{n}` is one or more digits (a positive integer)
- `{slug}` is a lowercase alphanumeric slug with hyphens

The system must identify entity type from the ID pattern without registry lookup:

| Pattern | Entity type |
|---------|-------------|
| `FEAT-{id}` | Feature |
| `TASK-{id}` | Task |
| `BUG-{id}` | Bug |
| `DEC-{id}` | Decision |
| `{X}{n}-{slug}` (single non-digit char + digits + hyphen) | Plan |

These patterns must never collide.

### 6.2 Feature entity changes

Feature retains its core Phase 1 role but gains:

- A `parent` field (renamed from `epic`) referencing the parent Plan ID. The field is nullable — Features can exist without a parent Plan.
- A `design` field referencing an optional design document record.
- The existing `spec` field becomes a reference to a tracked document record.
- The existing `dev_plan` field (renamed from `plan`) becomes a reference to a tracked document record.
- A `tags` field for cross-cutting organisational metadata.

Feature status is stored as a field on the entity but automatically updated when document approvals or task completions trigger lifecycle transitions.

### 6.3 Plan entity changes

Plans are a new entity type with distinct semantics from the Epic they replace:

- A Plan owns a design document (its roadmap/direction).
- A Plan organises Features — they can be added, removed, or re-parented at any time.
- A Plan does not have tasks and does not get "implemented."
- A Plan accumulates decisions made during its scope.
- A Plan's closure (`active → done`) is a manual human decision, not derived from child Feature state.

### 6.4 Document metadata records

Documents gain tracked lifecycle metadata as first-class managed records. A document record is metadata about a document file — the file content remains at its canonical path.

Document records are not workflow entities. They do not have their own MCP entity operations or lifecycle state machines. They are documents with structured metadata, owned by Plans or Features, tracked in Git, and queryable through MCP operations.

### 6.5 Tags

All entity types gain a `tags` field. Tags are freeform lowercase strings with optional namespacing via colon (e.g., `phase:2`, `team:frontend`).

The system indexes tags for querying but does not enforce a vocabulary. There is no tag registry.

---

## 7. Plan Entity Semantics

A Plan is a coordination entity that organises a body of work, provides high-level direction, and groups related Features.

A Plan:

- has one special document (type: `design`) that serves as its roadmap and direction
- may own research reports and other exploratory material
- accumulates decisions made during its scope
- organises Features — they can be added, removed, or re-parented at any time
- does not have tasks, does not get "implemented"

The Plan provides direction. The Feature is what gets designed and built.

---

## 8. Required Entity Fields

### 8.1 Plan minimum fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | yes | Format: `{X}{n}-{slug}` |
| `title` | string | yes | Human-readable title |
| `status` | enum | yes | Current lifecycle state |
| `design` | string | no | Reference to design document record |
| `tags` | list of strings | no | Freeform tags |
| `created` | timestamp | yes | Creation time |
| `created_by` | string | yes | Creator identity |
| `updated` | timestamp | yes | Last modification time |
| `supersedes` | string | no | ID of the Plan this supersedes |
| `superseded_by` | string | no | ID of the Plan that supersedes this |

### 8.2 Feature minimum fields (revised)

Feature retains all Phase 1 required fields with the following changes:

| Field | Change | Description |
|-------|--------|-------------|
| `parent` | renamed from `epic` | Reference to parent Plan ID (nullable) |
| `design` | new | Reference to optional design document record |
| `spec` | revised | Reference to tracked document record (was plain string) |
| `dev_plan` | renamed from `plan` | Reference to tracked dev plan document record |
| `tags` | new | Freeform tags |

### 8.3 Document record fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | yes | Format: `{owner-id}/{slug}` |
| `path` | string | yes | Relative path to the document file |
| `type` | enum | yes | One of: `design`, `specification`, `dev-plan`, `research`, `report`, `policy` |
| `title` | string | yes | Human-readable title |
| `status` | enum | yes | One of: `draft`, `approved`, `superseded` |
| `owner` | string | no | Parent Plan or Feature ID |
| `approved_by` | string | no | Identity of approver |
| `approved_at` | timestamp | no | Time of approval |
| `content_hash` | string | yes | SHA-256 hash of file content at last registration or approval |
| `supersedes` | string | no | ID of the document this supersedes |
| `superseded_by` | string | no | ID of the document that supersedes this |
| `created` | timestamp | yes | Creation time |
| `created_by` | string | yes | Creator identity |
| `updated` | timestamp | yes | Last modification time |

No version counter — supersession chains handle versioning.

---

## 9. Lifecycle Requirements

### 9.1 Plan lifecycle

**States:** `proposed`, `designing`, `active`, `done`

**Terminal states:** `superseded`, `cancelled`

**Forward transitions:**

| From | To | Trigger |
|------|----|---------|
| `proposed` | `designing` | Plan's design document is created (draft) |
| `designing` | `active` | Plan's design document is approved |
| `active` | `done` | Manual — human judgment |

**Terminal transitions:**

| From | To | Trigger |
|------|----|---------|
| any non-terminal | `superseded` | Another Plan supersedes this one |
| any non-terminal | `cancelled` | Manual — human judgment |

A Plan in `active` can have Features added, removed, or re-parented at any time. A Plan does not auto-transition based on Feature state. The `active → done` transition is a human decision.

### 9.2 Feature lifecycle (revised)

**States:** `proposed`, `designing`, `specifying`, `dev-planning`, `developing`, `done`

**Terminal states:** `superseded`, `cancelled`

**Forward transitions:**

| From | To | Trigger |
|------|----|---------|
| `proposed` | `designing` | Design document is created (draft) |
| `proposed` | `specifying` | Shortcut — specification is created without a design document |
| `designing` | `specifying` | Design document is approved |
| `specifying` | `dev-planning` | Specification is approved |
| `dev-planning` | `developing` | Dev plan is approved |
| `developing` | `done` | All tasks are complete |

**Backward transitions** (via document supersession):

| Condition | Feature reverts to |
|-----------|--------------------|
| Approved design is superseded | `designing` |
| Approved specification is superseded | `specifying` |
| Approved dev plan is superseded | `dev-planning` |

Feature status is stored as a field but auto-updated by document approvals and task completions. The stored field is authoritative.

### 9.3 Document lifecycle

**States:** `draft`, `approved`, `superseded`

**Transitions:**

| From | To | Trigger |
|------|----|---------|
| `draft` | `approved` | Human approval |
| `approved` | `superseded` | A newer document supersedes this one |

The `review` state is intentionally omitted. In an AI-mediated workflow, a document in `draft` is inherently in review — changes are discussed and applied conversationally, so there is no separate "waiting for review" state.

A document's approval directly drives its owning entity's lifecycle transitions.

### 9.4 Transition enforcement

The system must reject lifecycle transitions that are not in the transition tables above.

Document-driven transitions must occur automatically when the triggering event happens (e.g., approving a Feature's specification must transition the Feature from `specifying` to `dev-planning`).

The system must not allow a Feature to be in a state that contradicts its document status. If a Feature has an approved specification, it must be at `dev-planning` or later — never at `specifying` or earlier.

---

## 10. Prefix Registry Requirements

### 10.1 Registry location

The prefix registry lives in `.kbz/config.yaml` under a `prefixes` key.

### 10.2 Prefix rules

- A prefix must be exactly one non-digit Unicode rune.
- Prefixes are case-sensitive.
- Prefixes must be unique within the project.
- Each prefix has a required `name` field (human-readable) and an optional `description` field.

### 10.3 Default prefix

On `kbz init`, if no prefixes are declared, the system must create a default prefix `P` with name "Plan."

### 10.4 Prefix validation

The system must reject Plan creation with an undeclared prefix. Undeclared prefixes are an error, not a warning.

### 10.5 Prefix retirement

A prefix may be marked `retired: true`. A retired prefix:

- blocks creation of new Plans with that prefix
- does not invalidate existing Plans with that prefix
- cannot be renamed or reassigned

### 10.6 Prefix discovery

The system must expose the prefix registry through an MCP operation so that agents can discover the project's organisational vocabulary without external configuration.

---

## 11. Document Management Requirements

### 11.1 Document record storage

Document records must be stored as one YAML file per record in `.kbz/state/documents/`, tracked in Git. Document content stays at its canonical path (e.g., `work/design/foo.md`). The record is metadata only.

### 11.2 Document lifecycle operations

The system must support:

- **Submit** — register a document with the system, creating a document record in `draft` status. The system must compute and store the content hash. Submission includes Layers 1–2 ingest: the system must parse the document's structural skeleton and run pattern-based extraction, returning the skeleton to the caller for optional Layer 3 classification. Submit and ingest are a single operation — there is no reason to register a document without indexing it.
- **Approve** — transition a document from `draft` to `approved`. The system must record the approver, the approval time, and update the content hash. Approval must trigger the appropriate lifecycle transition on the owning entity.
- **Supersede** — transition a document from `approved` to `superseded`, linking to the superseding document. Supersession must trigger the appropriate backward transition on the owning entity.
- **Retrieve** — return a document record and/or its content. Approved documents must be returned verbatim — the system must not alter canonical prose.

### 11.3 Document-to-entity linking

- A specification document must be linked to exactly one Feature.
- A dev plan document must be linked to exactly one Feature.
- A design document may be linked to a Plan or a Feature.
- Research, report, and policy documents may have an owner (Plan, Feature) or may be project-level (no owner).

These links must be tracked as references on the document record (`owner` field) and as references on the entity (`design`, `spec`, `dev_plan` fields).

### 11.4 Document versioning

When a document is superseded, the previous version must be retained. The supersession chain must be queryable — given a document, the system must be able to return its predecessors and successors.

### 11.5 Document validation

The system must validate:

- Document type is a recognised enum value
- Required fields are present
- The file at the declared `path` exists
- The `content_hash` matches the file on disk (at registration and approval time)
- Referential integrity: the `owner` entity exists

### 11.6 Content hash drift detection

When the system reads a document record, it must compare the file's modification time against the record's `updated` timestamp. If the file is newer, the system must recompute the content hash and compare it to the stored hash. If the hashes differ, the document has drifted — the file was modified outside the system. The system must surface this as a warning on read and as an error in health checks. It must not silently serve drifted content as if it were the approved version.

### 11.7 Verbatim retrieval

Approved documents must be returned exactly as approved. The system must not re-render prose, alter formatting, or lose content during the store-and-retrieve cycle. The `content_hash` field must be usable to verify that the retrieved content matches the approved content.

---

## 12. Document Intelligence Requirements

### 12.1 The four-layer analysis model

The document intelligence backend must operate in four layers:

| Layer | Name | Trigger | Deterministic |
|-------|------|---------|---------------|
| 1 | Structural skeleton | Document change | Yes |
| 2 | Pattern-based extraction | Document change | Yes |
| 3 | AI-assisted classification | Agent curation | No (cached) |
| 4 | Document graph | Layers 1–3 | Yes (given inputs) |

### 12.2 Layer 1: Structural skeleton

The system must parse Markdown documents into a structural tree producing:

- **Section hierarchy** — every headed section as a node with level, title, byte offset, word count, byte count
- **Content blocks** — paragraphs, lists, tables, code blocks, and front matter identified within each section
- **Document outline** — a lightweight table of contents with section sizes

Layer 1 must be deterministic, cheap, and rebuilt on every document change.

### 12.3 Layer 2: Pattern-based extraction

On top of the structural tree, the system must apply deterministic pattern matching to extract:

- **Entity references** — workflow entity IDs (`FEAT-xxx`, `TASK-xxx`, `BUG-xxx`, `DEC-xxx`, Plan IDs) mentioned in document text. Every mention must become a typed edge in the graph.
- **Cross-document links** — markdown links and backtick-quoted paths to other documents.
- **Section classification by convention** — headers containing keywords like "Decision", "Rationale", "Requirements", "Open Questions", "Alternatives Considered", "Acceptance Criteria" must be classified by their conventional role.
- **Front matter parsing** — document metadata (type, status, date, related documents) must be extracted and indexed.

Layer 2 must be deterministic.

### 12.4 Layer 3: AI-assisted classification

The system must support agent-provided classifications for document fragments:

- **Fragment role classification** — classifying paragraphs and sections as requirement, decision, rationale, constraint, assumption, risk, question, definition, example, alternative, or narrative.
- **Concept extraction** — identifying concepts that a fragment introduces (defines) or uses (depends on).
- **Section summaries** — a one-line characterisation of each section.

Layer 3 classifications are non-deterministic but cached. The analysis runs once per document version, and results are stored as persistent metadata.

### 12.5 Classification protocol

The system must implement a classification protocol:

1. When a document is ingested, the system runs Layers 1–2 and returns the structural skeleton to the agent, along with the classification taxonomy and schema.
2. The agent classifies each section and returns the results via an MCP operation.
3. The system validates the classifications against the schema (roles drawn from the defined taxonomy, concept entries conform to schema, content hashes match).
4. If validation fails, the system must reject the classification with a specific error.
5. Validated classifications must be stored as persistent metadata and the document graph must be updated.

### 12.6 Classification stability

Classifications are immutable once recorded. Re-classification is an explicit manual operation (re-ingest), never automatic when models change or documents are re-indexed. The system must record the classification model/version on each classification for provenance.

### 12.7 Layer 4: Document graph

The system must maintain a persistent queryable graph built from Layers 1–3. The graph must include:

**Node types:**

| Node type | Source | Description |
|-----------|--------|-------------|
| Document | Layer 1 | A whole document with metadata |
| Section | Layer 1 | A headed section within a document |
| Fragment | Layer 3 | A classified piece of content within a section |
| EntityRef | Layer 2 | A mention of a workflow entity by ID |
| Concept | Layer 3 | An extracted concept or term |

**Edge types:**

| Edge type | From → To | Description |
|-----------|-----------|-------------|
| CONTAINS | Document → Section, Section → Fragment | Hierarchical containment |
| REFERENCES | Section → EntityRef | Section mentions a workflow entity |
| LINKS_TO | Section → Section | Explicit cross-document link |
| DEPENDS_ON | Fragment → Fragment | Logical dependency (AI-classified) |
| SUPERSEDES | Document → Document | Replacement relationship |
| INTRODUCES | Fragment → Concept | Where a concept is defined |
| USES | Fragment → Concept | Where a concept is used |
| REFINES | Document → Document | Refinement relationship |

### 12.8 Concept model

The system must maintain a corpus-wide concept registry:

- Concepts are named ideas or terms that appear across multiple documents.
- Concepts are extracted by agents during Layer 3 classification.
- The registry must deduplicate concepts (case-insensitive match, simple normalisation).
- Concepts do not have an explicit lifecycle — their presence and connectivity in the graph is their lifecycle. A concept with zero remaining references may be pruned automatically.

### 12.9 Unclassified documents

If a document has been indexed (Layers 1–2) but not classified (Layer 3), it must be flagged as "indexed but unclassified." Layers 1–2 queries must work normally. The system must provide an operation to list documents pending classification.

### 12.10 Index storage

Document intelligence index files must follow the same serialisation rules as entity state files (block style YAML, deterministic field order, UTF-8, LF line endings, trailing newline). Index files must be Git-tracked.

### 12.11 Graph storage

The initial graph storage model must be flat YAML edge lists. The system may migrate to SQLite if scale demands, but the YAML model is the Phase 2a requirement.

---

## 13. Fragment Role Taxonomy

The system must define and enforce a fixed taxonomy of fragment roles for Layer 3 classification.

### 13.1 Required roles

| Role | Description |
|------|-------------|
| `requirement` | Something the system must do or satisfy |
| `decision` | A design choice that was made |
| `rationale` | The reasoning behind a decision |
| `constraint` | A limitation or boundary condition |
| `assumption` | Something taken as true without proof |
| `risk` | A potential problem or concern |
| `question` | An open question or unresolved ambiguity |
| `definition` | A term or concept being defined |
| `example` | An illustrative example |
| `alternative` | A rejected or deferred alternative |
| `narrative` | Contextual prose that connects other fragments |

### 13.2 Taxonomy evolution

The taxonomy may be extended. New roles must be added to the schema. Existing classifications must not be invalidated when the taxonomy grows.

### 13.3 Confidence

Each classification must carry a confidence indicator (`high`, `medium`, `low`) set by the classifying agent. Downstream consumers may filter by confidence level.

---

## 14. Query Requirements

### 14.1 Entity queries

The system must support filtering entities by:

- Status
- Parent Plan (for Features)
- Tags (any entity with a matching tag)
- Date range (created, updated)
- Entity type

The system must support cross-entity queries:

- All tasks for features in a given Plan
- All entities tagged with a given tag
- All features in a given status across all Plans

### 14.2 Document queries

The system must support:

- List all documents with metadata (type, status, owner, classification state)
- List documents by type
- List documents by owner (Plan or Feature)
- List documents by status
- Retrieve a document's supersession chain

### 14.3 Document intelligence queries

The following operations must be supported, with graceful degradation based on available layers:

**Always available (Layers 1–2):**

| Operation | Description |
|-----------|-------------|
| `doc_outline` | Structural outline: section tree with titles, levels, word counts |
| `doc_section` | Retrieve a specific section by its path in the section tree |
| `doc_find_by_entity` | Find all sections that reference a specific workflow entity |
| `doc_gaps` | What document types are missing for a Feature? |
| `doc_pending` | List documents that are indexed but unclassified |

**Available after classification (Layer 3):**

| Operation | Description |
|-----------|-------------|
| `doc_find_by_concept` | Find all documents and sections that introduce or use a concept |
| `doc_find_by_role` | Find all fragments of a given role across the corpus |
| `doc_trace` | Trace an entity through the refinement chain |
| `doc_impact` | What depends on a given section? |

### 14.4 Tag queries

The system must support:

- List all tags in use across the project
- List all entities with a given tag
- Filter any entity listing by one or more tags

---

## 15. Storage and File Requirements

### 15.1 Plan storage

All Plan entity files must be stored in `.kbz/state/plans/`, one YAML file per Plan, regardless of prefix. The prefix is encoded in the ID and filename (e.g., `P2-basic-ui.yaml`).

### 15.2 Document record storage

All document records must be stored in `.kbz/state/documents/`, one YAML file per record.

### 15.3 Document intelligence index storage

Document intelligence index files must be stored in `.kbz/index/documents/`, one YAML file per indexed document.

The concept registry must be stored in `.kbz/index/concepts.yaml`.

The cross-document graph edges must be stored in `.kbz/index/graph.yaml`.

### 15.4 Deterministic formatting

All new file types (Plan YAML, document records, index files) must follow the same deterministic serialisation rules established in Phase 1 (P1-DEC-008): block style for mappings and sequences, double-quoted strings only when required by YAML syntax, deterministic field order, UTF-8, LF line endings, trailing newline, no YAML tags/anchors/aliases.

---

## 16. Concurrency Requirements

### 16.1 Optimistic locking for Git state

The system must implement optimistic locking for writes to Git-tracked state files:

1. Read the file and compute its content hash.
2. Perform the intended modification.
3. Before writing, verify the file's content hash has not changed since it was read.
4. If the hash has changed, fail the write with a specific conflict error. The caller may retry.

This must apply to all writes to `.kbz/state/` files (entities, document records).

### 16.2 Cache concurrency

The SQLite cache must use WAL (Write-Ahead Logging) mode for concurrent read access. Writes to the cache must be serialised.

### 16.3 No lock files

The system must not use lock files or assume a single-server deployment. The concurrency model must work with multiple MCP server instances operating on the same repository (within the constraints of optimistic locking).

---

## 17. Migration Requirements

### 17.1 Migration command

The system must provide a `kbz migrate phase-2` command that migrates Phase 1 state to Phase 2 format.

### 17.2 Migration operations

The migration must:

- Rename Epic entities to Plans
- Rename the `epic` field to `parent` on Feature entities
- Rename the `plan` field to `dev_plan` on Feature entities
- Move entity files from `.kbz/state/epics/` to `.kbz/state/plans/`
- Re-assign `EPIC-*` IDs to the `{X}{n}-{slug}` format
- Create the `.kbz/state/documents/` directory
- Create the `.kbz/index/` directory structure

### 17.3 Migration constraints

- The migration must be explicit — invoked by `kbz migrate phase-2`, not triggered automatically on startup.
- The migration must be idempotent — running it twice must produce the same result.
- The prefix registry must exist in `.kbz/config.yaml` before the migration can run. The migration must fail with a clear error if no prefixes are declared.
- Features without specification documents must become Features with null `spec` — this is valid (bottom-up features).

---

## 18. MCP Interface Requirements

### 18.1 New MCP operations for Phase 2a

Phase 2a must add MCP operations functionally equivalent to:

**Plan operations:**

- Create Plan
- Get Plan by ID
- List Plans (with filtering by status, tags, prefix)
- Update Plan status
- Update Plan fields

**Document management operations:**

- Submit document (register with metadata, compute content hash, run Layers 1–2 ingest, return structural skeleton)
- Approve document (transition to approved, trigger entity lifecycle)
- Supersede document (link to successor, trigger entity backward transition)
- Get document record
- Get document content (verbatim)
- List documents (with filtering by type, status, owner)

**Document intelligence operations:**

- Classify document (submit Layer 3 classifications from agent)
- Get document outline (structural section tree)
- Get document section (by section path)
- Find by entity reference
- Find by concept
- Find by fragment role
- List pending (unclassified) documents
- Gap analysis (missing document types for a Feature)

**Configuration operations:**

- Get project config (returns prefix registry and other project configuration)

**Query operations:**

- List entities with rich filtering (status, parent, tags, date range)
- Cross-entity queries (tasks for features in a Plan, entities by tag)

### 18.2 Changed MCP operations

- Epic operations (`create_epic`, `list_epics`, etc.) must be replaced by Plan operations.
- Feature operations must accept the renamed fields (`parent`, `dev_plan`).
- Entity listing operations must support the new filtering capabilities.

### 18.3 MCP output requirements

All new MCP operations must follow the same output conventions as Phase 1:

- Clear success/failure
- Useful error information
- Enough detail for an AI agent to interpret the outcome
- Structured machine-readable output

### 18.4 Strict validation

All new MCP operations must reject invalid writes rather than silently repairing them. This includes:

- Undeclared prefixes on Plan creation
- Invalid document types
- Invalid lifecycle transitions
- Content hash mismatches on document approval
- Classification schema violations

---

## 19. Phase 2b: Context Management Requirements

### 19.1 Context profile definition

The system must support named context profiles defined as YAML bundles with inheritance hierarchies. A context profile scopes what knowledge and document context an agent receives for a given role or task type.

### 19.2 Context assembly

The system must assemble targeted context packets for agents, composing:

- Design context (from document fragments via the document intelligence layer)
- Implementation context (from knowledge entries)
- Project conventions (from the prefix registry and project configuration)

Context assembly must support byte-based budgeting to fit within agent context windows.

### 19.3 Knowledge contribution

Agents must be able to contribute knowledge entries during work. Knowledge entries must be deduplicated on write.

### 19.4 Confidence scoring

Knowledge entries must carry confidence scores. The system must use Wilson score lower bound for reliability ranking.

### 19.5 Knowledge lifecycle

Knowledge entries must follow a lifecycle: contribution → confirmation → staleness → retirement.

The system must support:

- TTL-based pruning for stale entries
- Promotion triggers for confirmed entries
- Post-merge compaction

### 19.6 Usage reporting

The system must support usage reporting bundled with task completion.

---

## 20. Phase 2b: Agent Capability Requirements

### 20.1 Link resolution

The system must support inferring entity links from free-text references in documents and entity fields. This is a tool-assisted capability — the system provides candidates, the agent confirms.

### 20.2 Duplicate detection

The system must support detecting potential duplicate entities at creation time, using the cache and document intelligence layer to surface candidates.

### 20.3 Document-to-entity extraction guidance

The system must provide structured support (protocols or schemas) that help agents extract entities from approved documents more reliably.

---

## 21. Validation and Health Requirements

### 21.1 Extended health check coverage

Phase 2 health checks must detect, in addition to Phase 1 checks:

- Plan entities with undeclared prefixes
- Features with document status inconsistent with Feature lifecycle status
- Document records whose `content_hash` does not match the file on disk
- Document records whose `path` points to a nonexistent file
- Orphaned document records (owner entity does not exist)
- Documents in `approved` status with no `approved_by` or `approved_at`
- Index files that are stale relative to their source documents (content hash mismatch)

### 21.2 Validation timing

Validation must run:

- On every write operation (inline validation)
- On demand via health check operations
- The system must not silently accept invalid state

---

## 22. Acceptance Criteria

Phase 2 implementation is acceptable only if all of the following are true.

### 22.1 Plan creation and management

It must be possible, through the MCP interface, to:

- Create a Plan with a declared prefix
- Retrieve a Plan by ID
- List Plans with filtering by status, prefix, and tags
- Transition a Plan through its lifecycle states
- Reject Plan creation with an undeclared prefix

### 22.2 Prefix registry

The system must:

- Parse the prefix registry from `.kbz/config.yaml`
- Expose the registry through an MCP operation
- Validate Plan IDs against declared prefixes
- Support prefix retirement
- Create a default `P` prefix on `kbz init` when no prefixes are declared

### 22.3 Feature lifecycle driven by documents

Feature lifecycle transitions must be driven by document approvals:

- Approving a Feature's specification must transition the Feature to `dev-planning`
- Approving a Feature's dev plan must transition the Feature to `developing`
- Superseding an approved document must revert the Feature to the appropriate earlier state
- The shortcut from `proposed` to `specifying` (skipping design) must work

### 22.4 Document management

It must be possible to:

- Submit a document (creating a tracked record in `draft` status, running Layers 1–2 ingest, and returning the structural skeleton)
- Approve a document (transitioning to `approved` with approver and timestamp)
- Supersede a document (linking to the successor document)
- Retrieve an approved document verbatim — the content must match the stored content hash
- Detect content hash drift when a file has been modified outside the system
- List documents filtered by type, status, and owner
- Query a document's supersession chain

### 22.5 Document intelligence — structural analysis

The system must be able to:

- Parse a Markdown document into a structural section tree (Layer 1)
- Extract entity references from document text (Layer 2)
- Extract cross-document links (Layer 2)
- Return a document outline with section titles, levels, and sizes
- Retrieve a specific section by path

### 22.6 Document intelligence — classification

The system must be able to:

- Return a structural skeleton with classification schema to an agent
- Accept and validate agent-provided classifications
- Reject classifications that do not conform to the taxonomy schema
- Store validated classifications persistently
- List documents pending classification
- Query fragments by role across the corpus
- Query sections by concept

### 22.7 Rich queries

The system must support:

- Filtering entities by status, parent, tags, and date range
- Cross-entity queries (e.g., all tasks for features in a given Plan)
- Tag-based queries across entity types
- Document listing with filtering by type, status, and owner

### 22.8 Concurrency

Concurrent writes to the same entity file must not cause silent data loss. The optimistic locking mechanism must detect conflicts and return an error.

### 22.9 Migration

The `kbz migrate phase-2` command must:

- Convert existing Epic entities to Plans
- Rename fields on Feature entities
- Move files to the correct directories
- Be idempotent
- Fail clearly if the prefix registry is not configured

### 22.10 Deterministic storage

All new file types (Plan YAML, document records, index files) must produce deterministic output. Writing the same state twice without meaningful change must not produce different file output.

### 22.11 Tags

Tags must be:

- Settable on any entity type
- Queryable (list entities by tag, list all tags in use)
- Freeform lowercase strings with optional colon-namespacing

### 22.12 Context management (Phase 2b)

It must be possible to:

- Define context profiles as named YAML bundles
- Assemble context packets for agents with byte-based budgeting
- Contribute knowledge entries during work
- Query knowledge entries by relevance and confidence
- Observe knowledge lifecycle transitions (contribution → confirmation → staleness → retirement)

### 22.13 Agent capabilities (Phase 2b)

The system must support:

- Suggesting entity links from free-text references (link resolution)
- Surfacing potential duplicate entities at creation time (duplicate detection)

---

## 23. Open Questions for Planning

### 23.1 Phase 2a

1. Exact MCP operation names and request/response shapes for Plan operations, document management operations, and document intelligence operations
2. Exact YAML field order for Plan entities and document records
3. Exact ID-to-filename mapping for document records (the `{owner-id}/{slug}` format needs a filesystem-safe encoding)
4. Exact cache schema for document metadata, document graph, and rich queries

### 23.2 Phase 2b

1. Exact schema for KnowledgeEntry records
2. Context profile inheritance conflict resolution semantics
3. Minimum viable subset of the knowledge lifecycle for initial delivery
4. Bootstrap story for existing projects with documents that predate adoption

---

## 24. Summary

Phase 2 extends the workflow kernel with document understanding and context management.

Phase 2a delivers:

- Plans replace Epics — with user-defined prefixes, a coordination-focused lifecycle, and design document ownership
- Document management — tracked lifecycle records for documents, with approval driving Feature lifecycle transitions
- Document intelligence — a four-layer analysis model that parses, indexes, classifies, and queries the structural content of design documents
- Rich queries — filtering by status, parent, tags, and cross-entity relationships
- Tags — freeform cross-cutting organisational metadata on all entity types
- Concurrency — optimistic locking for safe multi-agent access
- Migration — a tested, idempotent tool for evolving Phase 1 state

Phase 2b delivers:

- Context profiles and assembly — targeted context packets for agents with byte-based budgeting
- Knowledge lifecycle — persistent knowledge that accumulates and matures over the project's life
- Agent capabilities — link resolution and duplicate detection

The design-to-delivery pipeline — design → specify → plan → implement → verify — is the structural backbone. Plans own the design space; Features own delivery. Documents bridge the two with tracked lifecycle. The entity hierarchy is flat — organisational concerns are handled through tags and views, not nesting. The document intelligence layer indexes and queries document content for both human navigation and agent context assembly.

Phase 2 builds the intelligence layer that makes the workflow system understand its own documents — not just store them.