---
id: DOC-01KM8JVA8NTPH
type: implementation-plan
title: Phase 1 Implementation Plan
status: submitted
feature: FEAT-01KM8JTF0VP0K
created_by: human
created: 2026-03-21T16:14:58Z
updated: 2026-03-21T16:14:58Z
---
# Phase 1 Implementation Plan

- Status: plan draft
- Purpose: define the implementation plan for Phase 1 of the workflow system
- Date: 2026-03-18
- Based on:
  - `work/design/workflow-design-basis.md`
  - `work/spec/phase-1-specification.md`
  - `work/design/document-centric-interface.md`
  - `work/design/machine-context-design.md`
  - `work/design/agent-interaction-protocol.md`
  - `work/design/quality-gates-and-review-policy.md`
  - `work/design/git-commit-policy.md`
  - `work/design/product-instance-boundary.md`

---

## 1. Purpose

This plan defines how Phase 1 of the workflow system should be implemented.

Phase 1 is the workflow kernel. It is intended to establish a trustworthy, minimal, MCP-first core that can:

- represent core workflow state
- validate that state
- scaffold and validate key documents
- support AI-agent-mediated use through a strict machine interface
- begin tracking the workflow tool’s own work in a limited way

This plan describes **how to implement Phase 1**, while remaining consistent with the Phase 1 specification, the design basis, and the boundary rules between:

- product
- project instance
- project design materials

---

## 2. Phase 1 Outcome

At the end of Phase 1, the project should have a working workflow kernel that can:

1. create and store canonical records for:
   - Epic
   - Feature
   - Task
   - Bug
   - Decision
2. validate records, transitions, references, and naming
3. support the document-centric interface: submission, normalisation, approval, verbatim retrieval, scaffolding, and validation for required phase 1 document types
4. expose the required formal operations through MCP
5. support a strict CLI using the same core logic
6. maintain deterministic canonical state on disk
7. rebuild and use a local derived cache for query support
8. run health checks over the project state
9. begin tracking the workflow tool’s own work in a limited way

Phase 1 is complete when those outcomes are implemented and verified against the specification.

---

## 3. Planning Principles

Implementation of Phase 1 must follow these planning principles.

### 3.1 The specification is binding

`work/spec/phase-1-specification.md` is the implementation contract for Phase 1.

Implementation should not exceed the required scope without an explicit decision.

### 3.2 Keep Phase 1 narrow

Do not build phase 2+ systems early.

Phase 1 should avoid:

- orchestration
- advanced memory systems
- incident/RCA implementation
- rich GitHub automation
- semantic search
- broad self-hosting automation

### 3.3 Build for later evolution

Phase 1 should remain minimal, but it must not block later evolution toward:

- first-class Specification and Plan entities
- richer policy query
- context packing and role-scoped context assembly (see `work/design/machine-context-design.md`)
- worktree automation
- orchestration
- greater process self-management

In particular, Phase 1 must reserve namespace in the MCP operation set and storage layout for context management operations, even though those operations are not implemented until Phase 2.

### 3.4 Build the product, not the instance

Implementation work should create reusable product assets, not entangle current project state into the product.

### 3.5 Start obeying the workflow while building it

Where practical, implementation of the workflow tool should already be tracked using the emerging workflow system.

This should be done carefully and gradually, without pretending that later capabilities already exist.

---

## 4. Deliverables

Phase 1 implementation should produce the following categories of deliverables.

## 4.1 Product deliverables

These are reusable product assets.

- Go code for the workflow kernel
- MCP server implementation
- strict CLI implementation
- phase 1 entity schemas
- validation logic
- deterministic formatting logic
- document scaffolding logic
- document validation logic
- health check logic
- local cache rebuild/query support
- default templates required for phase 1
- minimal reusable reference/config assets needed for phase 1

## 4.2 Instance deliverables for this project

These are current-project artifacts used to operate the workflow while building it.

- a dedicated project instance root or temporary equivalent
- initial canonical records for this project as needed
- phase 1 docs scaffolded/validated through the kernel where practical
- local derived cache for this project

## 4.3 Verification deliverables

- validation commands or routines
- acceptance checks against the phase 1 specification
- example or fixture records for all phase 1 entity types
- test coverage for core logic
- manual verification notes where automation is not yet sufficient

---

## 5. Product–Instance Boundary for Implementation

Implementation must respect the product/instance boundary.

## 5.1 Product code and reusable assets

Phase 1 implementation should create reusable code and assets in product-oriented directories.

Examples of product-facing categories:

- code
- schemas
- templates
- reusable defaults
- reference docs

## 5.2 Project instance state

The current project’s live workflow state should not be mixed into reusable product asset directories.

If a dedicated instance root is introduced in Phase 1, it should be used consistently.

If not, Phase 1 must still preserve the distinction conceptually and structurally as much as possible.

## 5.3 Design docs are not product assets by default

Current design/policy/specification documents remain project design materials unless deliberately promoted.

Do not copy design drafts into reusable product asset locations without explicit promotion.

---

## 6. Recommended Implementation Order

The implementation should proceed in layers.

## 6.1 Layer 1: core data model and storage

Implement:

- phase 1 entity definitions
- canonical file representation
- deterministic read/write behavior
- one-file-per-entity storage
- basic repository layout assumptions
- stable naming rules
- basic ID allocation mechanism

This layer is the foundation.

## 6.2 Layer 2: schema and validation engine

Implement:

- field validation
- entity-type validation
- lifecycle transition validation
- referential integrity validation
- supersession validation
- naming/path validation

This layer should operate independently of MCP and CLI presentation.

## 6.3 Layer 3: document support

Implement the document-centric interface model defined in `work/design/document-centric-interface.md`:

- document templates for all required phase 1 document types (proposals, draft designs, designs, specifications, implementation plans, research reports)
- scaffolding routines
- document validation routines (type recognition, frontmatter, required sections, naming, referential integrity)
- document submission and storage
- normalisation support (agents normalise; the system stores and presents for human approval)
- approve-before-canon enforcement (documents are not canonical until a human approves)
- verbatim retrieval of approved canonical documents (no re-rendering or re-summarisation on retrieval)
- document-to-entity extraction on approval (extracting decisions, entity updates, and cross-references)
- document lifecycle tracking (submitted → normalised → approved → canonical)

This layer should treat human-authored content as validated, not generated wholesale. The normalisation step is performed by agents, not by the system — the system stores the result and enforces the approval gate.

## 6.4 Layer 4: MCP service layer

Implement the formal machine interface for:

- entity operations: create, read/query, update status, candidate validation
- document operations: submit, normalise, approve, retrieve (verbatim), scaffold, validate
- extraction operations: extract entities and decisions from approved documents
- health checks

This layer should remain thin and strict.

## 6.5 Layer 5: strict CLI layer

Implement a CLI using the same shared core logic.

The CLI is secondary, but useful for:

- manual operation
- bootstrap use
- CI
- debugging
- repair

## 6.6 Layer 6: local derived cache

Implement local cache rebuild/query support.

This layer should be:

- derived
- disposable
- rebuildable from canonical state

It should not become required for correctness.

## 6.7 Layer 7: project bootstrap usage

Begin using the Phase 1 kernel to manage limited current-project workflow state.

This should happen only after the kernel is trustworthy enough for basic use.

---

## 7. Work Breakdown

The following is the recommended Phase 1 breakdown.

## 7.1 Track A — Core model and file representation

Goal:
Define and implement the canonical representation of phase 1 entities.

Tasks:
- define internal entity types for Epic, Feature, Task, Bug, Decision
- define canonical serialization model
- define deterministic field order
- define file naming conventions
- define path conventions for each entity type
- implement load/save operations
- implement normalization of timestamps and basic field formatting

Outputs:
- stable read/write core
- example files for each entity type

## 7.2 Track B — Validation engine

Goal:
Implement strict validation for canonical state.

Tasks:
- validate required fields
- validate field types
- validate status values
- validate legal transitions
- validate parent/child links
- validate supersession links
- validate references to linked docs
- validate file naming conventions
- implement health check aggregation

Outputs:
- reusable validation module
- health check report model

## 7.3 Track C — ID allocation

Goal:
Implement a safe phase 1 ID allocation system.

Tasks:
- choose and document the concrete phase 1 strategy
- implement allocation for Epic/Feature/Bug/Decision
- implement sub-ID allocation for Task
- ensure no manual user editing is required
- handle expected edge cases sufficiently for phase 1
- make the strategy replaceable later if needed

Outputs:
- working phase 1 allocator
- documented limits and assumptions

## 7.4 Track D — Document lifecycle

Goal:
Implement the document-centric interface for required phase 1 document types.

Tasks:
- define templates for all required phase 1 document types: proposals, draft designs, designs, specifications, implementation plans, research reports
- implement scaffold routines for each document type
- implement document submission and storage
- implement normalisation support (store normalised result, present for approval)
- implement approve-before-canon enforcement
- implement verbatim retrieval of approved canonical documents
- implement document-to-entity extraction on approval (decisions, entity updates, cross-references)
- implement document lifecycle state tracking (submitted → normalised → approved → canonical)
- implement validation routines for type recognition, required frontmatter, required sections, naming conventions, and referential integrity

Outputs:
- phase 1 templates for all required document types
- scaffold commands/operations
- document submission/approval pipeline
- verbatim retrieval
- entity extraction from documents
- document validation logic

## 7.5 Track E — MCP interface

Goal:
Expose the workflow kernel through a strict MCP surface.

Tasks:
- define phase 1 operation contracts
- implement entity create operations
- implement entity query/get/search operations
- implement entity status update operations
- implement entity approve where required by the composite feature model
- implement candidate validation
- implement document submit/normalise/approve/retrieve operations
- implement document scaffold/validate operations
- implement document-to-entity extraction operations
- implement health check access
- ensure structured return values and clear failures

Outputs:
- phase 1 MCP server behavior
- stable operation semantics

## 7.6 Track F — CLI interface

Goal:
Provide a strict secondary interface using the same logic.

Tasks:
- map core phase 1 operations into CLI commands
- ensure consistent output modes
- ensure errors are clear and usable
- avoid fuzzy behavior or hidden prompts
- support bootstrap/manual use

Outputs:
- strict CLI for phase 1 operations

## 7.7 Track G — Local cache

Goal:
Provide derived query support without making the cache canonical.

Tasks:
- define derived cache model
- implement rebuild from canonical state
- support enough query acceleration for search/health checks
- ensure loss of cache is harmless
- ensure rebuild is straightforward

Outputs:
- rebuildable local cache
- cache rebuild command/operation

## 7.8 Track H — Phase 1 bootstrap usage

Goal:
Use the kernel in a limited way to track this project’s own work.

Tasks:
- decide initial instance root or temporary structure
- create initial canonical records as needed
- use the kernel for at least some current epics/features/tasks/bugs/decisions
- verify that the system can already manage limited self-work without confusion

Outputs:
- proof that phase 1 can begin managing the workflow tool’s own work in a limited way

---

## 8. Phase 1 Required Operations

Implementation must support operations functionally equivalent to the following.

## 8.1 Required entity create operations

- create epic
- create feature
- create task
- create bug
- record decision

## 8.2 Required entity read/query operations

- get entity by identity
- search/query entities
- inspect canonical state reliably

## 8.3 Required entity update operations

- update status
- perform approval where needed by phase 1
- update/correct canonical records safely

## 8.4 Required validation operations

- validate candidate structured data
- validate documents
- run health checks

## 8.5 Required document lifecycle operations

- scaffold document from template (by document type)
- submit document (store as submitted, ready for normalisation)
- approve document (transition from normalised to canonical, enforce human approval gate)
- retrieve approved document verbatim (no re-rendering or modification)
- validate document (type recognition, frontmatter, sections, naming, referential integrity)

## 8.6 Required extraction operations

- extract entities and decisions from approved documents

---

## 9. Implementation Constraints

## 9.1 No silent scope expansion

If a capability belongs more naturally to Phase 2+, do not implement it in Phase 1 unless a decision explicitly promotes it.

## 9.2 No conflation of product and project state

Do not implement product features by hardcoding current project assumptions into reusable asset paths or schemas.

## 9.3 No reliance on future orchestration

The Phase 1 kernel must not depend on:
- task orchestration
- automatic decomposition
- advanced memory layers
- rich branch automation

## 9.4 No destructive workflows by default

Error correction should prefer:
- correction
- supersession
- terminal states
- Git history

not destructive deletion.

## 9.5 Keep the ID strategy replaceable

Even if a concrete phase 1 ID strategy is chosen, implementation should not make later replacement disproportionately hard.

---

## 10. Review and Quality Expectations for Implementation

Phase 1 implementation work must follow the existing policy documents.

## 10.1 Agent interaction protocol

Implementation work involving agents must follow:

- `work/design/agent-interaction-protocol.md`

Especially:
- normalize before commit
- clarify when ambiguity matters
- use formal operations for canonical changes
- show important normalized changes before commit

## 10.2 Quality gates and review policy

Implementation review must follow:

- `work/design/quality-gates-and-review-policy.md`

At minimum, implementation review must consider:
- specification conformance
- implementation quality
- test adequacy
- documentation currency
- workflow integrity

## 10.3 Git commit policy

Implementation commits must follow:

- `work/design/git-commit-policy.md`

This means:
- one coherent change per commit
- traceability to workflow objects
- structured commit messages
- no unrelated bundled changes

---

## 11. Testing and Verification Plan

Phase 1 implementation must be verified against the phase 1 specification.

## 11.1 Core verification categories

Verification should cover at least:

- canonical entity creation
- entity retrieval/query
- invalid transition rejection
- deterministic storage behavior
- reference validation
- document scaffold/validation behavior
- health check coverage
- limited bootstrap usability

## 11.2 Test types

Expected test types include:

- unit tests for core validation logic
- unit tests for serialization and deterministic formatting
- unit tests for ID allocation logic
- unit tests for lifecycle enforcement
- unit or integration tests for document validation
- integration tests for MCP operations where practical
- integration tests for CLI behavior where practical
- manual verification for bootstrap usage and workflow fit where automation is not yet available

## 11.3 Fixture strategy

The implementation should include stable fixtures or examples for:

- valid records of each phase 1 entity type
- invalid records for validation failure cases
- supersession cases
- broken reference cases
- invalid transition cases
- document validation cases

## 11.4 Verification against acceptance criteria

Implementation is not complete until it is explicitly checked against the acceptance criteria in:

- `work/spec/phase-1-specification.md`

A simple “it seems to work” is not sufficient.

---

## 12. Bootstrap Plan for Using the Kernel on Itself

The system should begin tracking limited current-project work as soon as it is safe to do so.

## 12.1 Initial bootstrap target

Once the core kernel is stable enough, use it to track at least:

- one epic for the workflow tool
- one or more features for current work
- tasks for implementing phase 1 pieces
- bugs found in the workflow tool
- decisions about architecture and scope

## 12.2 Bootstrap discipline

During bootstrap:
- keep the process simple
- use strong human oversight
- avoid pretending the tool is more mature than it is
- use manual review where necessary
- preserve the product/instance boundary

## 12.3 Success condition

Bootstrap usage is successful when the workflow kernel can manage limited current-project work without:
- corrupting state
- creating ambiguity about product vs instance
- requiring advanced future features

---

## 13. Risks and Mitigations

## 13.1 Risk: Phase 1 scope creep

Risk:
Trying to implement too much too early.

Mitigation:
- use the Phase 1 specification as a binding scope contract
- explicitly reject phase 2+ ideas unless promoted by decision
- keep implementation plan narrow

## 13.2 Risk: Composite Feature becomes confusing

Risk:
The composite `Feature` model may create ambiguity in approval/supersession/plan handling.

Mitigation:
- document clearly that it is a phase 1 simplification
- avoid overcomplicating the composite model
- leave room to split later

## 13.3 Risk: ID strategy churn

Risk:
The chosen ID strategy may prove awkward.

Mitigation:
- keep the allocation module isolated
- test expected edge cases early
- do not entangle the rest of the architecture with the exact format more than necessary

## 13.4 Risk: product/instance leakage

Risk:
Current project state pollutes reusable product assets.

Mitigation:
- follow `work/design/product-instance-boundary.md`
- keep reusable assets and current instance state conceptually separate from the start
- delay promotion until deliberate

## 13.5 Risk: overreliance on immature automation

Risk:
The project depends on workflow capabilities that do not yet exist.

Mitigation:
- maintain manual fallbacks
- keep phase 1 bootstrap expectations modest
- avoid orchestration assumptions

---

## 14. Decisions Needed Before or Early in Implementation

The following decisions are needed. Items marked ✓ have been resolved as accepted decisions in the decision log. Remaining items should be resolved during early implementation and recorded as decisions.

1. ✓ exact phase 1 ID allocation strategy (P1-DEC-007 — accepted)
2. ✓ exact canonical file layout for product and instance concerns (P1-DEC-006 — accepted)
3. ✓ exact YAML subset/formatting constraints (P1-DEC-008 — accepted)
4. ✓ exact minimum required field set per entity (P1-DEC-009 — accepted)
5. ✓ exact lifecycle transition graph (P1-DEC-010 — accepted)
6. ✓ exact document template structures (P1-DEC-014 — accepted)
7. exact MCP operation names and request/response shapes (P1-DEC-011 — implementation decision, resolve during Track E)
8. exact CLI command mapping (P1-DEC-012 — design decision requiring human input; CLI commands should read like English, see design basis §6.2)
9. exact local cache scope and rebuild behavior (P1-DEC-013 — implementation decision, resolve during Track G)
10. initial bootstrap approach for using the kernel on this project (P1-DEC-015 — process decision, resolve when kernel is stable enough for self-use)

---

## 15. Definition of Done for the Phase 1 Implementation Plan

This plan is complete enough to begin implementation when:

1. it is consistent with `work/spec/phase-1-specification.md`
2. it is consistent with the product-instance boundary
3. it keeps phase 1 scope acceptably narrow
4. it provides a clear implementation order
5. it identifies the main work tracks
6. it identifies the main risks
7. it defines how implementation will be verified

---

## 16. Summary

Phase 1 implementation should proceed by building the workflow kernel in layers:

1. core model and deterministic storage
2. validation engine
3. document lifecycle (submission, normalisation, approval, verbatim retrieval, scaffolding, validation, entity extraction)
4. MCP interface (entity operations + document lifecycle operations)
5. CLI interface
6. local cache
7. limited bootstrap use on the project itself

The implementation must stay within phase 1 scope, respect the product-instance boundary, and be verified against the phase 1 specification.

The goal is not to build the whole future system at once.

The goal is to build a trustworthy kernel that:
- solves the biggest consistency failures now
- is usable by agents through MCP
- supports the document-centric interface for human-authored design and planning materials
- can begin managing limited current-project work
- provides a strong base for later phases