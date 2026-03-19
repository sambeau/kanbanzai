# Phase 1 Implementation Plan

- Status: plan draft
- Purpose: define the implementation plan for Phase 1 of the workflow system
- Date: 2026-03-18
- Based on:
  - `docs/design/workflow-design-basis.md`
  - `docs/spec/phase-1-specification.md`
  - `docs/design/agent-interaction-protocol.md`
  - `docs/design/quality-gates-and-review-policy.md`
  - `docs/design/git-commit-policy.md`
  - `docs/design/product-instance-boundary.md`

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
3. scaffold and validate at least the required phase 1 documents
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

`docs/spec/phase-1-specification.md` is the implementation contract for Phase 1.

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
- context packing
- worktree automation
- orchestration
- greater process self-management

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

Implement:

- document templates for required phase 1 docs
- scaffolding routines
- document validation routines
- basic schema/frontmatter/section checking

This layer should treat human-authored content as validated, not generated wholesale.

## 6.4 Layer 4: MCP service layer

Implement the formal machine interface for:

- create
- read/query
- update status
- candidate validation
- document scaffold
- document validation
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

## 7.4 Track D — Document scaffolding and validation

Goal:
Support required phase 1 documents.

Tasks:
- define templates for feature specification docs
- define templates for feature plan docs
- define template or scaffold support for bug-related docs if needed
- implement scaffold routines
- implement validation routines for required frontmatter/sections/naming/links
- ensure docs align with phase 1 scope

Outputs:
- phase 1 templates
- scaffold commands/operations
- doc validation logic

## 7.5 Track E — MCP interface

Goal:
Expose the workflow kernel through a strict MCP surface.

Tasks:
- define phase 1 operation contracts
- implement create operations
- implement query/get/search operations
- implement status update operations
- implement approve where required by the composite feature model
- implement candidate validation
- implement document scaffold/validate
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

## 8.1 Required create operations

- create epic
- create feature
- create task
- create bug
- record decision

## 8.2 Required read/query operations

- get object by identity
- search/query objects
- inspect canonical state reliably

## 8.3 Required update operations

- update status
- perform approval where needed by phase 1
- update/correct canonical records safely

## 8.4 Required validation operations

- validate candidate structured data
- validate documents
- run health checks

## 8.5 Required document operations

- scaffold required document types
- validate required document types

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

- `docs/design/agent-interaction-protocol.md`

Especially:
- normalize before commit
- clarify when ambiguity matters
- use formal operations for canonical changes
- show important normalized changes before commit

## 10.2 Quality gates and review policy

Implementation review must follow:

- `docs/design/quality-gates-and-review-policy.md`

At minimum, implementation review must consider:
- specification conformance
- implementation quality
- test adequacy
- documentation currency
- workflow integrity

## 10.3 Git commit policy

Implementation commits must follow:

- `docs/design/git-commit-policy.md`

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

- `docs/spec/phase-1-specification.md`

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
- follow `docs/design/product-instance-boundary.md`
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

The following decisions should be made explicitly during implementation planning or very early implementation:

1. exact phase 1 ID allocation strategy
2. exact canonical file layout for product and instance concerns
3. exact YAML subset/formatting constraints
4. exact minimum required field set per entity
5. exact lifecycle transition graph
6. exact document template structures
7. exact MCP operation names and request/response shapes
8. exact CLI command mapping
9. exact local cache scope and rebuild behavior
10. initial bootstrap approach for using the kernel on this project

These decisions should be recorded as decisions, not left implicit.

---

## 15. Definition of Done for the Phase 1 Implementation Plan

This plan is complete enough to begin implementation when:

1. it is consistent with `docs/spec/phase-1-specification.md`
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
3. document support
4. MCP interface
5. CLI interface
6. local cache
7. limited bootstrap use on the project itself

The implementation must stay within phase 1 scope, respect the product-instance boundary, and be verified against the phase 1 specification.

The goal is not to build the whole future system at once.

The goal is to build a trustworthy kernel that:
- solves the biggest consistency failures now
- is usable by agents through MCP
- can begin managing limited current-project work
- provides a strong base for later phases