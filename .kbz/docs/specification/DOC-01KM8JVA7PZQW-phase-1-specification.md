---
id: DOC-01KM8JVA7PZQW
type: specification
title: Phase 1 Specification
status: submitted
feature: FEAT-01KM8JTF0VP0K
created_by: human
created: 2026-03-21T16:14:58Z
updated: 2026-03-21T16:14:58Z
---
# Phase 1 Specification: Workflow Kernel

- Status: specification draft
- Purpose: define the required behavior, scope, constraints, and acceptance criteria for Phase 1 of the workflow system
- Date: 2026-03-18
- Based on:
  - `workflow-design-basis.md`
  - `workflow-system-design.md`
  - `document-centric-interface.md`

---

## 1. Purpose

This specification defines Phase 1 of the workflow system.

Phase 1 is the minimum viable workflow kernel required to make project workflow state:

- structured
- queryable
- validated
- Git-friendly
- usable by AI agents through MCP
- simple enough to implement without overbuilding

Phase 1 exists to solve the most serious consistency failures in the current workflow:

- drift in naming and structure
- stale or inconsistent status
- weak referential integrity
- weak object tracking
- weak validation
- poor discoverability
- expensive rediscovery of workflow state by agents

This specification defines **what Phase 1 must do**, not how it must be implemented.

---

## 2. Goals

Phase 1 must provide a strict workflow kernel that allows humans and AI agents to create, manage, validate, and query core workflow state for software project work.

It must:

1. provide a small set of first-class workflow entities
2. store canonical workflow state in structured text files
3. expose formal workflow operations through an MCP interface
4. support deterministic validation of state and documents
5. support enough functionality to begin tracking the workflow tool’s own development in a limited way
6. avoid dependency on advanced orchestration or memory systems not yet built

---

## 3. Non-Goals

Phase 1 does **not** aim to deliver the full workflow vision.

It is explicitly **not** required to provide:

- full multi-agent orchestration
- automatic task decomposition from specifications
- specialist team memory systems
- advanced knowledge graphing
- full incident and root cause analysis workflows
- full release management
- comprehensive GitHub synchronization
- advanced worktree automation beyond what is essential
- self-managing or self-modifying system behavior
- complete migration of historical legacy workflow data
- semantic/vector search
- fully formalized approval objects as separate first-class entities

Phase 1 must remain deliberately narrow.

---

## 4. Design Principles for Phase 1

Phase 1 must conform to the following principles.

### 4.1 Workflow state is authoritative

Canonical workflow truth must live in structured workflow state.

### 4.2 Conversation is the human interface

Humans may interact informally through chat and rough documents. Formal state changes happen only through normalized, validated operations.

### 4.3 MCP is the primary machine interface

The primary formal interface for workflow operations in Phase 1 is the MCP server.

A CLI may exist, but it is secondary.

### 4.4 Strict core, forgiving interface

The workflow kernel must be strict.
Human input may be rough.
Agents are expected to normalize before commit.

### 4.5 One canonical fact in one place

Facts such as status, links, IDs, and supersession must not be duplicated across multiple conflicting locations.

### 4.6 One file per entity

Each phase 1 entity must be stored in its own canonical structured file.

### 4.7 Deterministic output

Given the same state, the system must produce the same canonical file representation.

### 4.8 Validation before trust

The system must validate schema, references, naming, and state transitions before accepting formal changes.

---

## 5. Scope

### 5.1 Included in Phase 1

Phase 1 includes:

- core workflow entities
- canonical structured state files
- deterministic formatting rules
- formal MCP operations for core workflow actions
- document scaffolding
- document validation
- health checks
- local derived query cache
- basic ID allocation
- enough structure to track the workflow system’s own tasks and bugs

### 5.2 Excluded from Phase 1

Phase 1 excludes:

- orchestration layer
- automatic planning
- automatic delegation chains
- specialist memory systems as first-class implemented subsystems
- incident and root cause analysis as implemented first-class entities
- advanced branch/worktree lifecycle management
- advanced merge automation
- comprehensive GitHub issue/PR sync
- deep migration tooling
- broad self-hosting or self-governance

---

## 6. Material Taxonomy

Phase 1 must explicitly distinguish four classes of material.

### 6.1 Intake artifacts

These are human-provided or conversational inputs that have not yet been normalised or approved.

Examples:
- brainstorm notes
- rough markdown pasted into chat
- pasted bug descriptions
- free-form requests
- review comments
- proposal documents before approval

Phase 1 does not need to fully model intake artifacts as first-class stored objects, but it must assume they exist conceptually.

### 6.2 Documents

These are human-facing prose artifacts that flow through the design-to-implementation process. Documents are authored by humans and agents collaboratively, normalised by agents, and approved by humans before becoming canonical.

Once approved, a document is canonical in its own right — the system must return it verbatim on retrieval. Internally, the system indexes and fragments document content to extract structured entities (decisions, requirements, links), but the document's prose identity is preserved.

Phase 1 must recognise the following document types:
- proposal
- research report
- draft design
- design
- specification
- implementation plan
- user documentation

Phase 1 must support document submission, approval, and verbatim retrieval. See `document-centric-interface.md` for the full taxonomy, formality gradient, and interface contract.

### 6.3 Canonical entity records

These are validated structured workflow objects maintained by the system as the internal source of truth.

Humans do not manage entity records directly. Agents extract entity data from documents and conversations, create and update records through formal operations, and maintain lifecycle state and referential integrity.

Phase 1 must implement canonical entity records.

### 6.4 Projections

These are generated or derived views based on canonical state.

Phase 1 may support only a limited set of projections, but must preserve the distinction.

---

## 7. Phase 1 Entity Model

Phase 1 must implement the following first-class entity types:

- `Epic`
- `Feature`
- `Task`
- `Bug`
- `Decision`

These are the only required first-class entities in Phase 1.

## 7.1 Deferred entity types

The following are explicitly deferred beyond Phase 1:

- `Project`
- `Milestone`
- `Specification` as a first-class entity
- `Plan` as a first-class entity
- `Approval` as a first-class entity
- `Release`
- `Incident`
- `RootCauseAnalysis`
- `ResearchNote`
- `Design`
- `KnowledgeEntry`
- `TeamMemoryEntry`

Phase 1 may refer to some of these conceptually, but must not require their implementation.

---

## 8. Entity Semantics

### 8.1 Epic

An `Epic` is a high-level unit of planned work owned primarily by humans.

It groups related features and provides a stable planning anchor.

### 8.2 Feature

A `Feature` is the handoff point between human-approved intent and AI-executed implementation.

For Phase 1, `Feature` is a **composite entity**.

It may carry:
- feature identity
- links to specification and plan documents
- approval-related status
- implementation lifecycle status

This is a deliberate simplification for Phase 1.

### 8.3 Task

A `Task` is the smallest formal execution unit in Phase 1.

A task should be suitable for a bounded unit of AI work, even if Phase 1 does not yet implement orchestration.

### 8.4 Bug

A `Bug` is a first-class defect object, not a task subtype.

It represents a defect or failure report with its own lifecycle and metadata.

### 8.5 Decision

A `Decision` records a meaningful project or architectural choice with rationale and links to affected entities.

---

## 9. Required Entity Fields

The exact implementation schema may evolve slightly, but Phase 1 must support at least the following minimum fields.

## 9.1 Epic minimum fields

- `id`
- `slug`
- `title`
- `status`
- `summary`
- `created`
- `created_by`

Optional but recommended in Phase 1:
- `features`

## 9.2 Feature minimum fields

- `id`
- `slug`
- `epic`
- `status`
- `summary`
- `created`
- `created_by`

Optional but recommended in Phase 1:
- `spec`
- `plan`
- `tasks`
- `decisions`
- `branch`
- `supersedes`
- `superseded_by`

## 9.3 Task minimum fields

- `id`
- `parent_feature`
- `slug`
- `summary`
- `status`

Optional but recommended in Phase 1:
- `assignee`
- `depends_on`
- `files_planned`
- `started`
- `completed`
- `verification`

## 9.4 Bug minimum fields

- `id`
- `slug`
- `title`
- `status`
- `severity`
- `priority`
- `type`
- `reported_by`
- `reported`
- `observed`
- `expected`

Optional but recommended in Phase 1:
- `affects`
- `origin_feature`
- `origin_task`
- `environment`
- `reproduction`
- `duplicate_of`
- `fixed_by`
- `verified_by`
- `release_target`

## 9.5 Decision minimum fields

- `id`
- `slug`
- `summary`
- `rationale`
- `decided_by`
- `date`
- `status`

Optional but recommended in Phase 1:
- `affects`
- `supersedes`
- `superseded_by`

> **Correction (2026-03-19):** `status` was added to this list. It was originally
> omitted, but §10.5 defines Decision lifecycle states (`proposed`, `accepted`,
> `rejected`, `superseded`) and P1-DEC-010 defines the entry state as `proposed`
> with explicit transition rules — all of which require a `status` field on the
> canonical record. Every other entity type already listed `status` in its §9
> minimum fields. The implementation correctly included `status` from the start;
> this edit aligns the spec with itself. The field is system-generated with the
> entry-state value `proposed`, consistent with P1-DEC-009's classification
> pattern for other entity types.

---

## 10. Lifecycle Requirements

Phase 1 must implement explicit lifecycle states and transition validation.

## 10.1 Epic lifecycle

Phase 1 Epic states must support at least:

- `proposed`
- `approved`
- `active`
- `on-hold`
- `done`

## 10.2 Feature lifecycle

Because `Feature` is composite in Phase 1, its lifecycle may combine specification and implementation state.

Phase 1 Feature states must support at least:

- `draft`
- `in-review`
- `approved`
- `in-progress`
- `review`
- `needs-rework`
- `done`
- `superseded`

## 10.3 Task lifecycle

Phase 1 Task states must support at least:

- `queued`
- `ready`
- `active`
- `blocked`
- `needs-review`
- `needs-rework`
- `done`

## 10.4 Bug lifecycle

Phase 1 Bug states must support at least:

- `reported`
- `triaged`
- `reproduced`
- `planned`
- `in-progress`
- `needs-review`
- `verified`
- `closed`
- `duplicate`
- `not-planned`
- `cannot-reproduce`

## 10.5 Decision lifecycle

Phase 1 Decision states must support at least:

- `proposed`
- `accepted`
- `rejected`
- `superseded`

## 10.6 Transition enforcement

Phase 1 must reject invalid lifecycle transitions.

The exact transition graph may be implementation-defined within reason, but the system must enforce:
- no illegal jumps
- no unknown states
- no silent transition coercion

---

## 11. Supersession Requirements

Phase 1 must support explicit supersession for revisable entities where relevant.

At minimum, the system must support supersession fields for:
- `Feature`
- `Decision`

Supersession support for other entities may be partial in Phase 1.

Phase 1 must support:
- recording that one entity supersedes another
- recording the inverse relationship
- validation that supersession targets exist
- clear detection of broken supersession links

---

## 12. Bug Workflow Requirements

Phase 1 must support bugs as first-class entities with a meaningful workflow.

## 12.1 Bug classification

Phase 1 must support classification of bugs at least along these lines:

- `implementation-defect`
- `specification-defect`
- `design-problem`

The exact values may vary, but the distinction must be preserved.

## 12.2 Standard bugfix path

Phase 1 must support a bug workflow that can represent:

1. report
2. triage
3. reproduce
4. plan
5. fix
6. verify
7. close

Phase 1 is not required to automate this path, but the state model and entity structure must support it.

## 12.3 Conversational bug intake

Phase 1 must support AI-agent-mediated creation of bug records from informal human descriptions.

That means the MCP interface must support creating valid bug records after normalization and clarification.

---

## 13. Identity Requirements

Phase 1 must allocate IDs through the system rather than by manual editing.

## 13.1 Required properties

IDs must be:
- unique
- stable
- tool-allocated
- usable with slugs
- safe for branch-based work

## 13.2 ID + slug convention

Phase 1 must support filenames and references that combine:
- machine identifier
- human-readable slug

## 13.3 Allocation strategy

Phase 1 must implement some concrete allocation strategy.

However, this specification does **not** require the final long-term global ID strategy to be fixed at this point.

What Phase 1 must guarantee:
- uniqueness
- stability
- safe operation in normal use
- no manual counter-file editing by users

The implementation may use:
- block allocation
- a simpler temporary strategy
- another safe mechanism

as long as the required behavior is satisfied.

## 13.4 ID edge cases

The implementation plan must explicitly test behavior for:
- allocation during concurrent work
- abandoned work
- legacy import scenarios
- collision handling if relevant

---

## 14. Storage and File Requirements

Phase 1 must use git-tracked structured text files as canonical storage.

## 14.1 One file per entity

Each canonical entity must be stored in its own file.

## 14.2 Deterministic formatting

The system must produce deterministic file output.

Repeated writes of unchanged data must not cause meaningless file churn.

## 14.3 YAML discipline

If YAML is used in Phase 1, it must be constrained enough to avoid ambiguous or unstable formatting.

At minimum:
- deterministic key order
- block style
- normalized timestamps
- no anchors/aliases
- explicit values where ambiguity is possible

## 14.4 Local cache

Phase 1 must support a local derived cache for query performance.

The cache:
- must be rebuildable from canonical state
- must not be canonical
- must not be required to be committed to Git

SQLite is the expected implementation direction, but this specification does not require the exact internal schema.

---

## 15. Document Requirements

Phase 1 must support the document-centric interface model defined in `document-centric-interface.md`.

## 15.1 Required document types

Phase 1 must recognise and support at least:
- proposal documents
- draft design documents
- design documents linked from features
- specification documents linked from features
- implementation plan documents linked from features
- research reports

## 15.2 Document submission and normalisation

Phase 1 must support document submission through the MCP interface. On submission, agents may normalise documents — cleaning language, tightening prose, improving structure — and must present the normalised result for human approval before the document becomes canonical.

## 15.3 Approve-before-canon

Phase 1 must enforce the rule that documents are not canonical until approved by a human. The approval workflow is:
1. Document is submitted or updated.
2. Agent normalises and presents the result.
3. Human approves or requests changes.
4. Once approved, the document is canonical.

## 15.4 Verbatim retrieval

Phase 1 must return approved canonical documents verbatim on retrieval. The system must not re-render, re-summarise, or re-normalise canonical documents on the way out. The approved form is the stored form.

This ensures multi-user consistency: every user who retrieves a canonical document sees the same document.

## 15.5 Document-to-entity extraction

Phase 1 must support extraction of structured entity data from documents. When a document is approved, the system (via agents) must be able to extract:
- decisions, with rationale and links to affected entities
- entity updates (feature records, task records, status changes)
- cross-document links and references

This extraction is internal — the human does not need to see or manage it.

## 15.6 Scaffolding

Phase 1 must be able to scaffold documents using stable templates for each recognised document type.

## 15.7 Validation

Phase 1 must validate documents for:
- document type recognition
- required frontmatter where applicable
- required sections where applicable
- naming conventions
- basic referential integrity
- schema conformity for templated documents

## 15.8 Human-authored content

Phase 1 must preserve the principle that:
- operational views are generated where possible
- human-authored design/spec content is validated and normalised, not replaced wholesale by tooling
- the human approves all substantive changes before they become canonical

---

## 16. MCP Interface Requirements

Phase 1 must expose formal workflow operations through an MCP server.

## 16.1 Required categories of MCP operations

Phase 1 must include MCP support for:

- identity/scaffolding
- status/lifecycle
- querying
- documents
- validation

Knowledge/memory and git/branch operations may be partial or deferred depending on phase 1 scope discipline.

## 16.2 Required Phase 1 operations

At minimum, Phase 1 must support operations functionally equivalent to:

- create epic
- create feature
- create task
- create bug
- record decision
- get object by identity
- search/query objects
- update status
- approve where phase-1 composite model requires it
- validate candidate data
- submit document
- retrieve document (verbatim for approved documents)
- list documents by type or by feature
- approve document
- scaffold documents
- validate documents
- run health checks

Exact names are implementation-defined.

## 16.3 MCP output requirements

Phase 1 MCP operations must return structured machine-readable output.

They must provide:
- clear success/failure
- useful error information
- enough detail for an AI agent to interpret the outcome

## 16.4 Strict validation

Phase 1 MCP operations must reject invalid writes rather than silently repairing them in unknown ways.

---

## 17. Normalization Support Requirements

Because the human interface is conversational, Phase 1 must support AI-mediated normalization safely.

## 17.1 Candidate validation

Phase 1 must support validation of candidate structured data before commit.

## 17.2 Link resolution support

Phase 1 should support resolving likely links from loose references if feasible within scope.

If not implemented in Phase 1, this must be explicitly deferred.

## 17.3 Duplicate detection support

Phase 1 should support duplicate detection for bug/feature creation if feasible within scope.

If not implemented in Phase 1, this must be explicitly deferred.

## 17.4 Preview before commit

Where practical, the system should support previewing what would be written before creating or mutating canonical state.

At minimum, the implementation plan must address how humans review normalization results before important commits.

---

## 18. Validation and Health Requirements

Phase 1 must support project health checks.

## 18.1 Health check coverage

At minimum, health checks must detect:
- broken references
- schema violations
- naming violations
- missing linked docs where required
- obvious state inconsistencies
- orphaned IDs where relevant to the chosen ID model

## 18.2 Validation timing

Phase 1 must support validation:
- on demand
- before merge or integration checks where applicable
- during normal agent use through MCP

CI integration is desirable but not strictly required in the first implementation if equivalent local validation exists.

---

## 19. Concurrency and Source Control Requirements

Phase 1 must acknowledge branch-based concurrent work, but should avoid overcommitting to advanced automation.

## 19.1 Required Phase 1 support

Phase 1 must support the data model and validation needed for concurrent work.

At minimum:
- state files must remain merge-friendly
- one-file-per-entity must reduce conflict surface
- IDs must be safe enough for ordinary concurrent use
- merge-relevant validation must exist

## 19.2 Worktree support

Advanced worktree lifecycle tooling is **not required** in Phase 1.

However, the system must not be designed in a way that blocks worktree-based isolation later.

## 19.3 Conflict-domain awareness

Phase 1 is not required to fully automate conflict-domain scheduling.

However, the architecture and planning model must preserve the principle that concurrency decisions should eventually account for:
- file overlap
- dependency ordering
- architectural boundaries
- verification boundaries

---

## 20. Relationship to Agent Instruction Systems

Phase 1 must coexist with platform-native agent instruction systems.

It must not assume replacement of those systems.

The implementation must support the conceptual stack:

1. platform-native instructions
2. workflow rules
3. generated context packets or equivalent future handoff artifacts
4. MCP workflow interface

Phase 1 is not required to implement full context packet generation, but must not preclude it.

---

## 21. Migration and Error Correction Requirements

Phase 1 must not ignore migration and correction, even if it implements them only minimally.

## 21.1 Migration

The implementation plan must include at least a basic migration strategy for:
- starting new work in the new system
- selectively importing active legacy work
- leaving historical legacy artifacts archived where appropriate

A full migration tool is not strictly required in initial implementation, but the migration path must be described.

## 21.2 Error correction

The workflow layer must support correction of:
- wrong field values
- wrong status updates
- wrong links
- wrong normalization results after commit

## 21.3 No destructive undo requirement

Phase 1 is not required to support destructive deletion or erasure of history.

It may instead rely on:
- correction
- supersession
- terminal states
- Git history

---

## 22. Phase 1 Bootstrap Requirement

Phase 1 must be usable to track limited development of the workflow tool itself.

This means Phase 1 must be sufficient to record and manage at least:
- epics for the workflow tool
- features of the workflow tool
- tasks for implementing the workflow tool
- bugs in the workflow tool
- decisions about the workflow tool

Phase 1 does **not** need to fully automate development of the workflow tool itself.

But it must be possible to begin using the workflow process on the workflow tool in a limited, manual-plus-agent-assisted way.

---

## 23. Out-of-Scope Clarifications

To avoid scope creep, the following are explicitly out of scope for Phase 1 unless later promoted by a deliberate decision:

- full self-hosting governance
- automatic roadmap generation
- automatic task decomposition from spec
- specialist memory stores as a mature subsystem
- incident/RCA implementation
- release objects
- advanced GitHub sync
- sophisticated worktree orchestration
- dependency graph visualization
- semantic search
- append-only event logging as a formal subsystem

---

## 24. Acceptance Criteria

Phase 1 implementation is acceptable only if all of the following are true.

## 24.1 Entity creation

It must be possible, through the MCP interface, to create:
- an Epic
- a Feature
- a Task
- a Bug
- a Decision

with valid canonical state written to disk.

## 24.2 Entity retrieval

It must be possible to:
- retrieve a single entity
- search entities
- inspect stored state reliably

## 24.3 State transition enforcement

Invalid lifecycle transitions must be rejected.

## 24.4 Deterministic storage

Writing the same canonical entity twice without meaningful change must not produce different file output.

## 24.5 Referential integrity

Broken references must be detectable by validation or health checks.

## 24.6 Document support

The system must support the document-centric interface:
- documents can be submitted, normalised, and approved through the MCP interface
- approved documents are returned verbatim on retrieval — the system does not alter canonical prose
- the system can scaffold and validate at least the required phase 1 document types
- entity data (decisions, requirements, links) can be extracted from approved documents

## 24.7 Health checks

The system must detect at least the required classes of inconsistency.

## 24.8 Conversational workflow support

An AI agent must be able to use the MCP interface to translate rough human intent into valid phase 1 operations for at least:
- feature creation
- bug creation
- status updates
- decision recording
- document submission and approval

## 24.9 Document round-trip integrity

A document submitted and approved must be retrievable verbatim. The system must not introduce meaning changes, re-render prose, or lose content during the store-and-retrieve cycle.

## 24.10 Bootstrap usability

The Phase 1 kernel must be sufficient to begin tracking the workflow tool's own work in a limited way without requiring Phase 2+ systems.

---

## 25. Open Questions for Planning

The implementation planning phase must explicitly address:

1. exact phase 1 ID allocation strategy
2. exact YAML subset or alternative format constraints
3. exact lifecycle transition graph
4. exact validation rule set
5. exact document templates and schemas
6. exact migration approach for early adoption
7. exact mechanism for normalization review before commit
8. which normalization-support operations are truly in phase 1 vs deferred
9. how phase 1 avoids accidental dependence on later orchestration features

---

## 26. Summary

Phase 1 is the workflow kernel.

It must provide:

- a small set of first-class workflow entities
- canonical structured workflow state
- deterministic validation
- MCP-based formal operations
- document scaffolding and validation
- health checks
- enough capability to begin managing the workflow tool’s own work in a limited way

It must **not** attempt to deliver the full workflow vision in one step.

Its purpose is to establish a trustworthy, strict, Git-native kernel that future phases can build on and that implementation can be verified against.