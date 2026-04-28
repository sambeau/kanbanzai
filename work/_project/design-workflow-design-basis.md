# Workflow Design Basis

- Status: design basis
- Purpose: consolidated basis for planning a workflow system for human-and-AI collaborative software development
- Date: 2026-03-18
- Basis:
  - `workflow-system-design.md`
  - `initial-workflow-analysis.md`
  - `initial-workflow-analysis-review.md`
  - `document-centric-interface.md`
- Related:
  - `work/design/document-intelligence-design.md` §5, §6, §14.3 (extends §6.4 entity extraction, implements §8.5 internal fragmentation)
  - `work/design/machine-context-design.md` (Phase 2 context assembly)
- Notes:
  - This document should track the current consolidated design proposal closely enough to serve as the basis for planning
  - The workflow system should be built so that, over time, its own development can increasingly be managed through the workflow process it defines

---

## 1. Purpose

This document defines the current design basis for a workflow system intended to support humans and AI agents building large software projects together.

The system must support:

- brainstorming
- research
- design
- specification
- planning
- coding
- verification
- bug fixing
- approval
- shipping
- iteration at every step

It must work for:

- small projects without excessive ceremony
- large projects with many concurrent human and AI contributors
- long-running work where context, state, and decisions must persist across sessions

This document is intended to become the basis for the implementation plan.

It should also be read with a bootstrapping constraint in mind:

> the workflow tool is being built before the workflow fully exists, but it should be built so that its own ongoing design, implementation, validation, and maintenance can eventually be managed by the workflow system itself.

That means the system should support a gradual transition from manual stewardship to process-managed stewardship, without assuming that full self-management exists on day one.

---

## 2. Problem Statement

The current workflow model has strong bones but fails at consistency as projects grow.

What already works well:

- markdown is effective for human-readable documents
- staged document chains are useful:
  - design
  - specification
  - implementation plan
- IDs improve traceability
- reviewing work against specifications is effective
- separating human-owned specification from AI-owned implementation planning is valuable

What fails as scale increases:

- naming conventions drift
- document organization drifts
- markdown structure drifts
- status becomes stale
- superseded documents remain beside current ones without strong structural links
- branches are forgotten and drift too far
- backlog quality degrades
- IDs are slow to allocate and hard to find
- agents re-discover structure expensively instead of querying it directly

The root problem is not “too much documentation”.

The root problem is:

> the workflow is document-oriented, but lacks a strong canonical data model and strong enforcement.

Markdown is useful for people.
Markdown is weak as the sole system of record for a large multi-agent workflow.

---

## 3. Strategic Direction

The workflow system should be built around a strict structured core, with AI-mediated normalization of human input, and markdown as a human-facing surface.

The guiding direction is:

- canonical workflow truth lives in structured state
- humans interact primarily through normal chat and readable documents
- AI agents translate rough human intent into formal workflow actions
- strict operations happen through an MCP interface
- Git remains the durable collaboration and history layer
- local indexing and query support should make retrieval fast and reliable

This system should be:

- git-native
- MCP-first
- AI-mediated
- schema-backed
- markdown-friendly
- concurrency-aware
- simple enough to use
- strict enough to trust
- bootstrap-friendly
- capable of eventual self-hosting by the process it defines

---

## 4. Design Principles

### 4.1 The Parsley Aesthetic

All design decisions should follow the Parsley aesthetic:

- simplicity
- minimalism
- completeness
- composability

The workflow system must stay simpler than the project it manages.

It should also be designed so that the process remains usable while the tool is immature. Early versions must not require features that only later versions of the tool can provide.

### 4.2 Workflow State Is the Source of Truth, Documents and Chat Are the Interface

Canonical truth lives in structured workflow state.
Humans interact through documents and chat — not by managing entities directly.
Documents are the human-facing form of canonical truth: they go in as prose, the system extracts structure, and they come back out as the same prose.
Agents mediate between the document interface and the internal entity model.
Conversation is the interface for decisions and edits; workflow state is the source of truth.

### 4.3 Strict Core, Forgiving Interface

The system should be liberal in what it accepts, and disciplined in how it normalizes it.

Humans should be able to:

- speak naturally
- be imprecise
- refer to things loosely
- provide rough markdown
- provide incomplete descriptions

Agents should:

- interpret intent
- ask clarifying questions
- normalize input
- resolve references
- detect ambiguity
- avoid silently inventing important facts
- commit only valid, structured workflow changes

The workflow kernel itself should remain strict and deterministic.

### 4.4 One Canonical Fact in One Place

Every fact should have one authoritative location.

Examples:

- workflow status belongs in structured state, not duplicated in documents
- approvals belong in approval records or approval fields, not scattered through prose
- IDs are allocated by the system, not managed by hand
- relationships are explicit, not inferred from filenames alone

### 4.5 Codebase Is the Source of Truth for Code

The codebase is the source of truth for implementation.
Workflow state is the source of truth for planning and coordination.
Specifications define intended behavior and acceptance criteria.
Plans define how execution should proceed.

### 4.6 Humans Approve Intent; Agents Execute Scoped Work

Humans should own:

- goals
- priorities
- direction
- major tradeoffs
- approvals
- ship decisions

Agents should own:

- decomposition
- normalization
- planning support
- execution of scoped work
- verification support
- status updates
- housekeeping

### 4.7 Contexts Are Small; Links Are Strong

Each AI work unit should fit comfortably in one context window.
Context should be curated, not discovered from scratch each time.
Strong typed links between workflow objects should replace implicit shared memory.

### 4.8 Parallelism Is Earned Through Isolation, Not Hope

Concurrency requires:

- worktree isolation
- dependency awareness
- conflict-domain awareness
- merge gates
- branch hygiene

Hope is not a concurrency strategy.

### 4.9 Operational Documents Should Be Generated Where Possible

Status reports, backlog views, roadmap summaries, dashboards, and other operational views should be generated from canonical state wherever possible.

Human-authored documents such as designs and specs should be normalized and validated, not manually maintained as parallel status stores.

### 4.10 Verification Is Attached to Every Unit of Work

No task is done without verification.
No feature is done without evidence.
No bug is closed without confirming the bugfix path.
No claim of completion should exist without fresh supporting evidence.

### 4.11 Bugs Are First-Class Workflow Objects

Bugs are not mislabeled tasks.
They have their own:

- lifecycle
- metadata
- reproduction concerns
- severity concerns
- verification concerns
- escalation paths

### 4.12 Intake Artifacts Must Be Normalized Before Becoming Canonical

Human-provided material is useful, but raw input is not automatically authoritative.

The system must distinguish among:

- intake artifacts
- canonical records
- projections

and must normalize intake artifacts before formal commit.

### 4.13 Bootstrap Toward Process Self-Management

The workflow system is being created before the workflow process is fully embodied in tooling. The design must therefore support a staged transition:

- early phases may rely on humans and general-purpose agents to operate parts of the process manually
- later phases should move more of that operation into the workflow system itself
- the tool should expose enough structure, validation, and automation that its own development can eventually be tracked and governed through the same workflow model

This does not require full autonomy or self-modification. It requires that the system be capable of becoming a first-class user of its own process.

---

## 5. Lessons From Existing Systems

### 5.1 What to Preserve From the Existing Basil Workflow

The new system should preserve these strengths from the existing workflow:

- the design → spec → plan chain
- IDs everywhere for traceability
- review against specification rather than plan
- separation of “what” from “how”
- markdown as a human-readable medium
- iterative thinking before coding

### 5.2 What to Fix From the Existing Basil Workflow

The new system must fix:

- naming drift
- document organization drift
- inconsistent structure
- stale status
- weak supersession
- branch drift
- forgotten plans
- weak backlog hygiene
- slow ID management
- poor discoverability
- ad hoc metadata
- duplicated truth across prose documents

### 5.3 Useful Ideas Borrowed From External Systems

From GSD-style systems:

- context rot is real
- fresh sessions per work unit matter
- disk-backed state matters
- worktrees are useful
- dependency-aware execution waves are useful
- preloaded context is better than repeated rediscovery

From Superpowers-style systems:

- discussion before execution matters
- approval gates matter
- planning before coding matters
- skills should counter known AI failure modes

From codebase-memory-mcp:

- persistent queryable structured memory is extremely valuable
- targeted retrieval beats repeated textual exploration
- this principle should be applied not only to code, but to planning and workflow state

---

## 6. High-Level Architecture

### 6.1 Overview

The desired architecture is:

Human ←→ AI Agent ←→ MCP Server (workflow kernel) ←→ Git-tracked state files + local SQLite cache
                          ↕
                     Codebase + code-aware tooling

The human talks to the AI in natural language.
The AI uses the MCP server to perform precise workflow operations.
Canonical workflow state is stored in git-tracked structured files.
A local SQLite cache provides fast querying and indexing.

### 6.2 MCP-First Control Model

The system should be MCP-first.

That means:

- the primary formal control surface is the MCP server
- the agent uses MCP tools to read and write workflow state
- the human does not need to know commands or schemas
- a CLI may exist, but it is secondary
- the same formal interface should be usable both for ordinary project work and for work on the workflow tool itself

The CLI, if present, should be useful for:

- debugging
- repair
- CI
- scripting
- maintenance

It should not be the primary human interaction model.

#### CLI design principles

The CLI is a human interface. Its design should follow these principles:

- **Read like English.** Prefer `kbz create epic` over `kbz epic create`. Commands should form natural verb-noun phrases where possible. When English word order doesn't fit (e.g., compound operations or queries with filters), consult a human on the phrasing rather than guessing.
- **Be predictable.** Similar operations should follow the same pattern. If `create` comes first for one entity type, it comes first for all of them.
- **Be strict.** The CLI does not guess. If a required argument is missing, it fails with a clear error rather than prompting interactively or inferring a default.
- **Match the MCP semantics, not the MCP names.** The CLI wraps the same core logic as the MCP server, but its command surface is designed for human ergonomics, not machine protocol conventions.

### 6.3 Material Taxonomy

The system should explicitly distinguish four classes of material.

#### Intake artifacts

Human-provided material that has not yet been normalized or approved.

Examples:

- brainstorm notes
- rough markdown pasted into chat
- pasted bug reports
- review comments
- free-form change requests
- proposal documents before approval

#### Documents

Human-facing prose artifacts that flow through the design-to-implementation process. Documents are authored by humans and agents collaboratively, normalised by agents, and approved by humans before becoming canonical.

Once approved, a document is canonical in its own right — the system returns it verbatim on retrieval. Internally, the system indexes and fragments document content to extract structured entities (decisions, requirements, links), but the document's prose identity is preserved.

Documents are not intake artifacts (they persist after approval), not entity records (they are prose, not YAML), and not projections (they are authored, not generated). They are the human-canonical form of project truth.

Document types, listed in order of the design-to-implementation progression:

- proposal
- research report
- draft design
- design
- specification
- implementation plan
- user documentation

See `document-centric-interface.md` for the full taxonomy, formality gradient, and interface contract.

#### Canonical entity records

Validated structured workflow objects maintained by the system as the internal source of truth.

Examples:

- Epic
- Feature
- Task
- Bug
- Decision

Entity records are the system's internal representation. Humans do not manage them directly — agents extract entity data from documents and conversations, create and update entity records through formal operations, and maintain lifecycle state, referential integrity, and cross-document consistency.

#### Projections

Generated views derived from canonical state.

Examples:

- roadmap summaries
- status reports
- handoff packets
- backlog views
- dashboards
- generated markdown summaries

This taxonomy is important because it prevents the ambiguity that caused drift in the previous workflow and because it separates the human interface (documents) from the internal model (entity records) while keeping both canonical.

### 6.4 AI-Mediated Normalization Pipeline

The system should use a formal normalization pipeline. The pipeline operates in two layers: document normalization (human-facing) and entity extraction (internal).

1. intake
2. interpretation
3. clarification
4. document normalization
5. human approval
6. entity extraction
7. formal commit through MCP
8. projection

#### Intake

Input may arrive through:

- chat conversation
- markdown documents
- pasted text
- rough proposals
- draft designs

#### Interpretation

The AI identifies:

- likely workflow intent
- affected documents and entities
- explicit facts and decisions
- implied facts
- ambiguities
- missing information

#### Clarification

The AI asks focused follow-up questions where needed.

#### Document normalization

The AI prepares a normalized document:

- cleans spelling, grammar, and prose style
- tightens structure and removes redundancy
- resolves minor ambiguities
- maintains the document's appropriate level of formality (informal for proposals, formal for specifications)
- preserves meaning — the AI does not silently change intent

The normalized document is presented to the human for review.

#### Human approval

The human reviews the normalized document and either approves it or requests changes. Once approved, the document becomes canonical and the system returns it verbatim on subsequent retrieval. No further AI rewriting occurs on the way out.

#### Entity extraction

The AI extracts structured data from the approved document:

- decisions, with rationale and links to affected entities
- requirements and specification elements
- entity updates (feature records, task records, status changes)
- cross-document links and references
- metadata, state transitions, and supersession relationships

This extraction is internal — the human does not need to see or manage it.

The document intelligence design (`work/design/document-intelligence-design.md` §6, §14.3) extends this stage with structural classification — fragment role tagging, concept extraction, and section summarisation — performed by the agent during the same ingest pass.

#### Formal commit

The AI uses MCP tools to create or update canonical entity records based on the extracted data.

#### Projection

The system regenerates or updates generated views (status reports, roadmaps, dashboards) from canonical state.

### 6.5 Normalization Reliability and Review

Normalization is useful, but dangerous if it silently changes meaning.

Therefore:

- the normalization step must be visible
- changes must be reviewable before commit
- the system should show a diff or summary where possible
- meaning-changing edits should be highlighted
- the human should confirm important normalized outputs before they become canonical

This is especially important for:

- specs
- decisions
- approvals
- bug reports
- design documents

These review safeguards are especially important during the bootstrapping phase, when the process is still being established and the tool is not yet mature enough to safely absorb mistakes without close human oversight.

---

## 7. Storage and Format Design

### 7.1 Git-Native State With Local Cache

Canonical state should live in git-tracked structured text files because:

- git can diff and merge text
- humans can inspect and review text
- branches can diverge safely
- canonical state stays portable and durable

A local SQLite cache should be used for:

- fast querying
- indexing
- dependency analysis
- health checks
- search support

The cache should be:

- derived
- local
- disposable
- rebuildable

### 7.2 File Format

The structured state format should be textual, deterministic, and easy to diff.

Strong candidates:

- YAML
- TOML
- a constrained schema-backed textual format

Current working assumption: YAML, but with a strict subset and strong formatting rules.

### 7.3 Requirements for the Structured Format

The structured format should support:

- stable key ordering
- deterministic rendering
- strong schema validation
- easy human inspection
- merge-friendly diffs
- no duplicated derivable values
- normalized timestamps
- predictable section order

### 7.4 YAML Discipline Rules

If YAML is used, it should follow strict conventions:

- deterministic key order
- block style only
- no anchors or aliases
- explicit values
- ISO 8601 timestamps
- no implicit typing surprises where avoidable
- one file per entity

### 7.5 One File Per Entity

Each canonical object should live in its own structured file.

Benefits:

- lower merge conflict surface
- better Git history
- easier validation
- easier rebuild of local cache
- simpler mapping between MCP operations and filesystem state

---

## 8. Object Model

The object model has two layers: the document type model (the human interface) and the entity model (the internal representation). Both are canonical — they represent the same underlying reality through different lenses.

### 8.1 Document Type Model

The system recognises a set of document types that flow through the design-to-implementation process. These are the objects humans work with directly.

Document types, in order of the design-to-implementation progression:

- Proposal
- Research report
- Draft design
- Design
- Specification
- Implementation plan
- User documentation

Documents move from informal prose toward formal definitions as they progress. The system must understand this formality gradient because it affects how documents are normalised, stored, and retrieved.

See `document-centric-interface.md` for the full taxonomy, formality gradient, lifecycle, and interface contract.

### 8.2 Phase 1 Core Entity Types

The phase 1 system should focus on a deliberately small core set of internal entities:

- Epic
- Feature
- Task
- Bug
- Decision

These are enough to solve most of the current consistency problems without overbuilding.

Humans do not manage these entities directly. Agents extract entity data from documents and conversations, create and update entity records through formal operations, and maintain lifecycle state and referential integrity. The entity model is the system's internal truth; the document type model is the human-facing truth.

### 8.3 Deferred Entity Types

The system should explicitly recognize likely future entities, even if they are not implemented in phase 1:

- Project
- Milestone
- Approval
- Release
- Incident
- RootCauseAnalysis
- KnowledgeEntry
- TeamMemoryEntry

This is important because some of the current phase 1 simplifications are intentional, not eternal.

It also matters for self-hosting: once the workflow tool begins to manage more of its own development, entities such as Approval, Release, and Milestone will become more important for governing the tool's own roadmap and operation.

Note: `Specification`, `Plan`, `Design`, and `ResearchNote` appeared in earlier versions of this deferred list as potential first-class entities. Under the document-centric interface model, these are document types rather than entities — they are the human-facing form of project truth and flow through the document lifecycle. The system indexes and fragments their content to extract entity data, but the documents themselves are the canonical human-facing artifacts, not entity records.

### 8.4 Composite vs First-Class Modeling

One major design question remains open:

Should `Feature` remain a composite v1 object that carries:

- feature identity
- spec linkage
- plan linkage
- approval lifecycle
- implementation lifecycle

or should the system make `Specification` and `Plan` first-class entities earlier?

Under the document-centric interface model, specifications and plans are document types with their own lifecycle and formality. Feature references these documents (the design, the spec, the plan) rather than embedding their content as flat fields. This partially resolves the composite-vs-first-class question: the documents are distinct objects in the document layer, even if Feature remains the coordination entity in the internal model.

This document adopts the following position:

- for phase 1, `Feature` may remain the composite coordination entity
- specifications and plans exist as distinct documents linked from the Feature record
- the evolution path is toward richer document-to-entity mapping, not necessarily toward making Specification and Plan into separate entity types

This distinction matters for:

- approvals
- supersession
- revision history
- bug vs spec defect handling
- plan invalidation

### 8.5 Document-to-Entity Relationship

Documents and entities relate bidirectionally:

- **On the way in:** when a document is ingested, agents extract structured information. A design document may produce or update Feature records, Decision records, and links between them. A specification maps to the spec content linked from a Feature. An implementation plan spawns Tasks.
- **On the way out:** when a document is retrieved, the system returns the canonical approved form. For new documents not yet approved, agents may assemble drafts from entity data, but once approved, the document's prose identity is fixed.

Internal fragmentation of documents — indexing decisions, linking requirements to features, connecting spec sections to design rationale — enables the system to compose targeted context for each agent. An agent implementing a specific task receives the relevant slices from across the document layers, not the entire design library.

The principle is: fragment internally for consistency and context assembly, present externally as whole documents.

The mechanism for this internal fragmentation — a four-layer analysis model producing a queryable document graph — is defined in `work/design/document-intelligence-design.md`.

### 8.6 Task Hierarchy

Recommended hierarchy:

Roadmap
  └── Epic
        └── Feature
              └── Task

Alongside:

- Bug
- Decision

The roadmap may initially remain a generated view rather than a first-class stored object.

### 8.7 Example Core Entity Fields

#### Epic

Suggested fields:

- `id`
- `slug`
- `title`
- `status`
- `summary`
- `created`
- `created_by`
- `features`

#### Feature

Suggested fields:

- `id`
- `slug`
- `epic`
- `status`
- `title`
- `summary`
- `created`
- `created_by`
- `spec`
- `plan`
- `branch`
- `tasks`
- `decisions`
- `supersedes`
- `superseded_by`

#### Task

Suggested fields:

- `id`
- `feature`
- `slug`
- `summary`
- `status`
- `assignee`
- `depends_on`
- `files_planned`
- `started`
- `completed`
- `verification`

#### Bug

Suggested fields:

- `id`
- `slug`
- `title`
- `status`
- `summary`
- `severity`
- `priority`
- `type`
- `reported_by`
- `reported`
- `affects`
- `origin_feature`
- `origin_task`
- `environment`
- `observed`
- `expected`
- `reproduction`
- `duplicate_of`
- `fixed_by`
- `verified_by`
- `release_target`

#### Decision

Suggested fields:

- `id`
- `slug`
- `summary`
- `rationale`
- `decided_by`
- `date`
- `affects`
- `supersedes`
- `superseded_by`

---

## 9. Lifecycle State Machines

### 9.1 General Rule

Each canonical object type should have:

- a defined state machine
- valid transitions
- validation of transitions through formal operations

### 9.2 Epic

Possible states:

- proposed
- approved
- active
- on-hold
- done

### 9.3 Feature

If `Feature` remains composite in phase 1, the state machine should explicitly be understood as a composite lifecycle.

Possible states:

- draft
- in-review
- approved
- in-progress
- review
- needs-rework
- done
- superseded

This should later be split if `Specification` and `Plan` become first-class.

### 9.4 Task

Possible states:

- queued
- ready
- active
- blocked
- needs-review
- needs-rework
- done

### 9.5 Bug

Possible states:

- reported
- triaged
- reproduced
- planned
- in-progress
- needs-review
- verified
- closed
- duplicate
- not-planned
- cannot-reproduce

State names should be standardized consistently across the system.

### 9.6 Decision

Possible states:

- proposed
- accepted
- rejected
- superseded

---

## 10. Supersession and Revision

Supersession must be explicit and structural.

Every revisable entity should support:

- `supersedes`
- `superseded_by`

The system should be able to answer:

- what is the current approved spec for this feature?
- what replaced this document?
- which tasks were planned against stale inputs?
- which decisions are current?
- which bugs originated against a superseded spec?

Superseded objects remain in the repository for history, but are clearly marked.

---

## 11. Identity Strategy

### 11.1 Requirements

IDs must be:

- unique across contributors and branches
- short enough for humans to use
- stable once created
- merge-safe
- readable with the help of slugs
- ideally sortable

### 11.2 ID + Slug

Every entity should use both:

- a machine identifier
- a human-readable slug

Examples:

- `FEAT-152-profile-editing`
- `BUG-027-avatar-upload-timeout`

### 11.3 Open Design Question: Allocation Strategy

The current design proposal uses block allocation.
The earlier analysis also considered distributed sortable IDs.

This remains an open design question.

#### Option A: block allocation

Pros:

- friendlier IDs
- easy human scanning
- familiar model

Cons:

- coordination complexity
- reservation bookkeeping
- unused block handling
- edge cases around abandoned work

#### Option B: distributed sortable IDs

Pros:

- no central coordination
- naturally distributed-safe
- simpler concurrency model

Cons:

- slightly less friendly IDs

### 11.4 Decision for Planning Purposes

The implementation plan should treat ID allocation as a topic requiring validation and testing before final commitment.

The phase 1 implementation should avoid locking the architecture into an untested allocation model too early.

### 11.5 Edge Cases to Resolve

Any final ID strategy must answer:

- what if a reserved block runs out?
- what if a branch is abandoned?
- what if two projects share a repository?
- what if IDs must be imported from a legacy project?
- how are unused IDs handled?
- how are collisions repaired if they happen?

---

## 12. Metadata Governance

### 12.1 Metadata Is Necessary

Text search is useful, but not enough.

Text search is good for:

- narrative discovery
- exploratory lookup
- fuzzy finding

Structured metadata is needed for:

- routing
- filtering
- dashboards
- queueing
- validation
- automation
- conflict awareness
- prioritization

### 12.2 Metadata Must Be Governed

Metadata beyond the core schema should never be ad hoc.

Every metadata field should have a formal definition including:

- name
- meaning
- value type
- allowed values or format
- scope
- examples
- owner
- introduction/change process

A schema registry or metadata glossary should define these fields.

### 12.3 Example Metadata Families

Useful metadata families may include:

- priority
- risk
- domain
- subsystem
- audience
- review_type
- release_scope
- confidentiality
- verification_class
- dependency_class
- impact_area
- bug_class

---

## 13. Bugs, Incidents, and Bugfix Workflow

### 13.1 Bugs Are First-Class

Bugs must be modeled separately from tasks because bug work has special concerns:

- reproduction
- observed vs expected behavior
- severity
- impact
- regression detection
- root cause
- verification
- release targeting
- duplicate detection

### 13.2 Detailed Bug Workflow

The standard bugfix path should be:

1. report
2. triage
3. reproduce
4. plan
5. fix
6. verify
7. close

#### Report

Capture:

- rough report
- environment
- severity
- impact
- origin if known

#### Triage

Determine:

- class
- priority
- duplicate status
- likely scope
- whether this is:
  - an implementation defect
  - a specification defect
  - a design problem

#### Reproduce

Establish a reliable repro path.

Where possible, convert it into:

- a test
- a script
- a clear manual check

#### Plan

Prepare a fix plan including:

- root cause hypothesis
- affected scope
- verification expectations

#### Fix

Execute via the normal task/plan machinery.

#### Verify

Confirm:

- the repro no longer fails
- regression coverage exists where appropriate
- user-facing behavior is correct

#### Close

Record:

- release target
- verification evidence
- lessons learned if necessary

### 13.3 Bug Classification

The system should distinguish:

- implementation defect
- specification defect
- design problem

These imply different workflow paths.

If the specification is wrong, the system should support a specification supersession path rather than treating the issue as only a code defect.

### 13.4 Bug Metadata

Useful structured bug metadata includes:

- `severity`
- `impact_area`
- `bug_class`
- `introduced_by`
- `detected_in`
- `customer_visible`
- `reproducible`
- `requires_hotfix`
- `requires_backport`
- `verification_class`

### 13.5 Conversational Bug Reporting

Humans should be able to report bugs informally via chat or markdown.

Examples:

- “There’s a bug in signup”
- “The composer ate my draft again”
- “Notifications seem broken on mobile”
- “I think we broke this this week”

The AI should then:

- recognize likely bug-report intent
- interpret chat and any rough notes
- ask missing questions
- search for duplicates
- normalize links and metadata
- create a valid structured bug record
- suggest next steps

### 13.6 Incident and RootCauseAnalysis

Not every bug is an incident.

An `Incident` should be used for production-significant failures such as:

- outages
- severe degradations
- data corruption
- security failures

A `RootCauseAnalysis` should capture:

- what happened
- why it happened
- why it was not caught
- what changed
- what prevents recurrence

These may be deferred from phase 1, but they should be acknowledged in the model now.

---

## 14. Human-AI Delegation Model

### 14.1 Four-Tier Hierarchy

The conceptual hierarchy should be:

1. Humans
2. PM / orchestration agents
3. specialist team agents
4. execution agents

This same hierarchy should eventually be usable for development of the workflow tool itself, with humans setting direction, orchestration agents managing work packages, specialist agents handling domains such as schema, MCP, migration, and validation, and execution agents implementing tightly scoped units of change.

### 14.2 Humans

Humans own:

- goals
- priorities
- product direction
- major tradeoffs
- approvals
- ship decisions

### 14.3 PM / Orchestration Agents

These agents own:

- roadmap decomposition
- dependency tracking
- task decomposition
- consistency checking
- backlog hygiene
- handoff preparation
- escalation when blocked

### 14.4 Specialist Team Agents

These agents own team-scoped expertise and operational memory.

Examples:

- backend
- frontend
- infrastructure
- documentation
- QA

They sit between orchestration and execution and accumulate scoped knowledge.

### 14.5 Execution Agents

Execution agents are short-lived workers.

They should:

- implement one task
- verify one task
- report results
- stop

---

## 15. Knowledge and Memory

### 15.1 Memory Classes

The system should distinguish:

- canonical project memory
- team operational memory
- working memory
- expertise memory

### 15.2 Governance of Knowledge

The knowledge layer must be governed to avoid becoming another dumping ground.

The system should clearly distinguish among:

- `Decision`
- `KnowledgeEntry`
- `RootCauseAnalysis`
- `Specification`
- `team convention`

### 15.3 Suggested Distinction

#### Decision

A binding project choice with rationale.

#### KnowledgeEntry

A reusable operational lesson or pattern.

#### RootCauseAnalysis

A defect- or incident-linked explanation and prevention record.

#### Specification

Project-specific intended behavior.

#### Team convention

A team-scoped working rule or standard practice.

### 15.4 Knowledge Entry Format

Knowledge should be structured and queryable rather than free-form where possible.

Suggested fields:

- `id`
- `team`
- `topic`
- `tags`
- `summary`
- `detail`
- `learned_from`
- `date`

---

## 16. Concurrency and Source Control

### 16.1 Worktrees for Isolation

Each feature or bug should get its own worktree and branch so agents do not interfere with each other’s working state.

### 16.2 Conflict Domain Awareness

Parallelism should be based on conflict domains, not just team ownership.

Relevant dimensions include:

- file overlap
- dependency ordering
- architectural boundaries
- verification boundaries

### 16.3 Prefer Vertical Slices

When possible, work should be decomposed into vertical slices rather than broad horizontal layers.

Vertical slices parallelize better and reduce cross-task coordination problems.

### 16.4 Branch Hygiene

The system should track:

- branch age
- drift from main
- recent activity
- merge readiness

### 16.5 Merge Strategy

Recommended direction:

- feature/bug branches via PR
- squash merge at feature granularity
- rebase or update before merge
- cleanup after merge

### 16.6 Merge Gates

Before merge, the system should verify:

- required tasks are done
- relevant specs are current
- verification exists
- branch is not stale
- no health-check errors block merge

---

## 18. MCP Interface

### 18.1 MCP Is the Formal Control Surface

The MCP server is the primary formal interface for workflow operations.

It should provide strict typed operations for:

- querying
- creating
- updating
- linking
- validating
- superseding
- generating projections

This interface should be designed from the start as the durable control surface for the process, including the eventual case where the workflow tool’s own development is managed through the same system.

### 17.2 MCP Tool Categories

Suggested tool categories:

- identity and scaffolding
- status and lifecycle
- querying
- knowledge and memory
- documents
- git and branches
- normalization support

### 17.3 Normalization Support Tools

The interface should likely support tools such as:

- candidate validation
- required-field discovery
- duplicate detection
- likely link resolution
- preview of normalized commits

These are especially important because the AI agent is expected to mediate between rough input and formal state.

### 17.4 Tool Design Principles

MCP tools should be:

- focused
- strict
- typed
- predictable
- structured in output
- idempotent where possible
- clear in error reporting

---

## 18. Relationship to Agent Instruction Systems

### 18.1 Four-Layer Model

The instruction and control stack should be understood as four layers:

1. platform-native agent instructions
2. workflow system rules
3. generated context packets
4. workflow MCP interface

### 18.2 Platform-Native Agent Instructions

These include:

- AGENTS files
- runtime-native skill files
- coding rules
- repository instructions

These remain in place where runtimes expect them.

### 18.3 Workflow System Rules

These define:

- schemas
- naming rules
- allowed transitions
- linking rules
- approval rules
- validation rules

### 18.4 Generated Context Packets

The workflow system should generate focused handoff/context packets for specific tasks.

### 18.5 Workflow MCP Interface

The MCP interface is the formal control surface agents use to read and mutate workflow state.

### 18.6 Effect on Existing Agent Files

The system should not replace platform-native agent instruction systems.

It should:

- reduce process logic hidden in prose
- provide stronger structured context
- make workflow behavior more consistent
- let agent instructions focus on interpretation and normalization

---

## 19. Relationship to Other Tools

### 19.1 Code-Aware Memory and Retrieval Tools

The workflow system should coexist with code-oriented tooling such as code memory/query systems.

These tools solve complementary problems:

- code memory helps understand implementation structure
- workflow memory helps understand planning, state, approvals, and rationale

### 19.2 Future Tools Worth Evaluating

The system should keep visible a list of categories worth evaluating:

- fast indexed search for workflow artifacts
- schema validation tools
- git worktree lifecycle tooling
- task/dependency graph tooling
- static documentation renderers
- CI enforcement hooks
- append-only logging / event capture

---

## 20. Continuous Validation

### 20.1 Health Checks

The system should support whole-project health checks for:

- stale branches
- stalled work
- missing specs
- missing plans
- broken links
- schema violations
- naming violations
- supersession inconsistencies
- orphaned IDs

### 20.2 Document Validation

Markdown documents should be validated for:

- required frontmatter
- required sections
- naming conventions
- cross-references
- schema conformance for templated documents

### 20.3 Validation Timing

Validation should be available:

- on demand
- before merge
- on agent startup where appropriate
- optionally in CI

---

## 21. Migration, Rollback, and Error Correction

### 21.1 Migration Path

The system will need a migration story for existing projects.

A future implementation plan should define:

- what existing artifacts are imported
- what is archived
- how IDs are mapped
- how legacy documents are treated
- what the cutover process is

This need not be fully designed yet, but it must not be forgotten.

### 21.2 Rollback and Undo

The workflow layer needs a better story than “Git exists”.

The system should support a clear distinction between:

- this change was wrong and should be reverted
- this was once right but is now superseded
- this was misclassified and should be corrected

The implementation plan should include an error-correction model.

### 21.3 Wrong Normalization

If a normalization step is wrong, the system should support:

- inspection of the normalized output
- correction before commit
- clear correction paths after commit

---

## 22. GitHub Integration

The workflow kernel should remain the source of truth.
GitHub should be treated as:

- a coordination layer
- a review surface
- an integration point

not as the canonical workflow store.

Useful GitHub integrations may include:

- PR creation
- PR review
- status checks
- optional issue linkage

But the project should still work with only the workflow kernel and Git.

---

## 23. Phase 1 Scope Boundary

Phase 1 must be tightly constrained.

A realistic phase 1 should likely include:

### Entities

- Epic
- Feature
- Task
- Bug
- Decision

### Core operations

- create
- query
- update status
- validate
- scaffold docs
- validate docs
- health check

### Interfaces

- MCP server
- shared core logic
- optional strict CLI
- local cache

### Explicitly not required in phase 1

- full orchestration
- deep specialist memory systems
- incident/RCA full implementation
- complex GitHub sync
- advanced knowledge graphing
- every deferred object type
- sophisticated branch automation beyond the essentials

The point of phase 1 is to solve the major consistency failures without building the whole future system at once.

The phase 1 implementation should also be consciously bootstrap-oriented:

- enough structure to manage the workflow tool’s own tasks and bugs
- not so much machinery that building the tool depends on features the tool does not yet have
- clear manual fallbacks where the future process is not yet implemented

---

## 24. Implementation Phases

### Phase 1: Workflow Kernel

Build:

- canonical state model
- strict validation
- MCP interface
- basic doc scaffolding and validation
- ID allocation
- health checks
- basic local cache

The phase 1 kernel should be sufficient to begin tracking the workflow tool’s own development in a limited way, even if much of the process is still operated manually by humans and general-purpose agents.

### Phase 2: Retrieval and Context Packing

Build:

- context packet generation
- retrieval support
- team-scoped memory beginnings
- richer query support
- generated overviews and projections

### Phase 3: Git Integration

Build:

- worktree management
- branch tracking
- merge readiness checks
- optional PR integration
- cleanup support

### Phase 4: Orchestration

Build:

- decomposition support
- dependency-aware scheduling
- fresh-session dispatch patterns
- worker review against specification
- orchestration tooling

Orchestration should come last.

By this stage, the workflow system should begin to manage substantial parts of its own continued development and maintenance through the same process it defines.

---

## 25. Open Questions

The following questions remain open and should inform planning:

1. Should phase 1 keep `Feature` composite, or should `Specification` become first-class sooner?
2. What final ID allocation strategy should be used?
3. What exact YAML subset or alternative format should be adopted?
4. How should normalization review be presented to the human?
5. What is the first migration path for an existing project?
6. What correction/undo model should exist above raw Git history?
7. How should phase 1 scope be kept from expanding?
8. When do Incident and RootCauseAnalysis become first-class?
9. What exact metadata registry format should be used?
10. How should generated projections be stored, cached, or regenerated?
11. At what point should the workflow tool begin managing its own roadmap, bugs, and releases through this process?
12. What safeguards are needed before the process can safely govern significant changes to itself?

---

## 26. Summary

This workflow system should be built as a Git-native, schema-backed, MCP-first workflow kernel with AI-mediated normalization, markdown-friendly human surfaces, and strong validation.

Its key commitments are:

- workflow state is authoritative
- conversation is the human interface
- MCP is the machine interface
- humans can be informal
- agents normalize before commit
- markdown is both intake and view, but not automatically canonical
- one file per entity keeps state mergeable
- SQLite provides fast local querying
- bugs are first-class
- concurrency depends on isolation and conflict awareness
- phase 1 must stay small and disciplined
- the system should be built so it can eventually become a first-class user of its own process

This document is intended to serve as the basis for the implementation planning phase.