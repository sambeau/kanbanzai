# Product–Instance Boundary and Bootstrapping

- Status: design draft
- Purpose: define the boundary between the reusable workflow system, the current project instance using it, and the current project’s design/implementation materials
- Date: 2026-03-18
- Related:
  - `docs/design/workflow-design-basis.md`
  - `docs/spec/phase-1-specification.md`
  - `docs/design/agent-interaction-protocol.md`
  - `docs/design/git-commit-policy.md`

---

## 1. Purpose

This document defines the hygiene boundary required to build a reusable workflow system while simultaneously using an early version of that system to help build itself.

The key problem is that three different things exist at once:

1. the reusable workflow system as a product
2. the live workflow instance for the current project
3. the current project’s design, research, and implementation materials about the workflow system itself

If these are not kept distinct, the project will quickly become confused about:

- what is reusable
- what is project-specific
- what is current live workflow state
- what should be copied into a new project
- what is only a draft design decision
- what belongs in code, instance state, or planning documents

This document exists to prevent that confusion.

---

## 2. Core Principle

The workflow system must distinguish between:

- **product**
- **instance**
- **project design**

These are related, but they are not the same.

The product is reusable.
The instance is local to one project.
The project design documents describe how the product is being designed and built.

The system should be built so that:

- the reusable system can be packaged and reused cleanly
- the current project can use an instance of that system without polluting the reusable product
- design documents can gradually be promoted into reusable assets when they stabilize
- the workflow tool can increasingly manage its own development over time without collapsing these boundaries

---

## 3. Definitions

## 3.1 Product

The **product** is the reusable workflow system itself.

It includes the things that should eventually be portable to a new project and used there without carrying over the state of the current repository.

Examples:

- source code for the workflow tool
- MCP server implementation
- CLI implementation
- schemas
- reusable templates
- reusable default policies
- reusable default instructions/skills
- initialization logic
- migration/import logic
- reusable reference documentation

The product should be capable of being versioned and reused independently of any one project instance.

## 3.2 Instance

An **instance** is one concrete use of the workflow system inside a specific project.

It includes the state and project-specific records that exist because the workflow system is being used in that project.

Examples:

- epics, features, tasks, bugs, and decisions for this project
- project-specific specifications and plans
- project-specific documentation created through the workflow
- project-specific overrides
- project-specific knowledge
- project-specific generated reports
- local derived cache

An instance is not generic. It belongs to a single project.

## 3.3 Project design materials

**Project design materials** are the documents and planning artifacts that describe how the workflow system product is being conceived, specified, and implemented in the current repository.

Examples:

- research memos
- design documents
- design reviews
- phase specifications
- implementation plans
- bootstrap policies
- architecture discussions

These are part of the current project, but they are not automatically part of the reusable product.

## 3.4 Promotion

**Promotion** is the act of taking a project design artifact that has stabilized and turning it into a reusable product asset.

Examples:

- a draft review policy becomes a reusable default review policy
- a draft document structure becomes a reusable scaffold template
- a design rule becomes a machine-readable schema or validation rule
- a runtime-specific protocol note becomes a shipped default instruction/skill

Promotion is deliberate. It is not automatic.

---

## 4. The Three Domains

The project should be understood as having three domains.

## 4.1 Domain A: Reusable system

This is the reusable workflow product.

It should contain only assets that are intended to be used by future projects.

Examples of things that belong here:

- Go implementation code
- reusable schema definitions
- reusable policy definitions
- reusable templates
- reusable init/bootstrap code
- reusable migration/import support
- reusable default instructions/skills
- reusable reference documentation

## 4.2 Domain B: Live project instance

This is the live workflow state for the current project.

It should contain only the active records and generated/derived materials needed to operate the workflow in this repository.

Examples of things that belong here:

- active epics/features/tasks/bugs/decisions for building the workflow tool
- current project specs and plans
- generated project projections
- local cache
- project-specific workflow overrides
- project-specific knowledge entries

## 4.3 Domain C: Design and implementation planning for the product

This is the set of project documents used to design and build the workflow tool itself.

Examples of things that belong here:

- design basis
- phase specifications
- review memos
- boundary documents
- implementation plans
- bootstrap guidance
- policy drafts before productization

This domain is essential during development but should not be confused with either the reusable product or the live instance state.

---

## 5. Required Hygiene Rules

## 5.1 Product assets must not contain live instance state

Reusable product directories must not contain current project workflow state as if it were a reusable shipped asset.

Examples of what must not happen:

- current project bug records stored beside reusable bug templates
- current project state files shipped as defaults
- generated project reports treated as reusable reference artifacts
- current project-specific policy overrides copied into reusable default policy directories

## 5.2 Instance state must not be treated as reusable defaults

Current project state must not become the default scaffold for future projects merely because it exists and works here.

Examples of what must not happen:

- current project’s feature tree becoming the basis for new project initialization
- current design docs copied directly as generic defaults without promotion
- current knowledge entries treated as universal product guidance without review

## 5.3 Draft design documents are not product assets by default

A design document may inform the product, but it does not become part of the reusable system automatically.

Draft documents should remain in design/planning space until deliberately promoted.

## 5.4 Generated projections must not become canonical by accident

Generated reports, summaries, and projections must remain clearly derived unless explicitly elevated.

A projection should never silently become the source of truth.

## 5.5 The system must support manual stewardship during bootstrap

The product must not assume its own maturity too early.

Early versions must allow:

- manual review
- manual promotion decisions
- manual separation of assets
- manual operation where automation is not yet available

This is a hygiene rule, not a temporary embarrassment.

---

## 6. Product Artifacts

The following categories are intended to become reusable parts of the workflow system product.

## 6.1 Code

Examples:

- `cmd/`
- `internal/`
- `pkg/`

These contain the implementation of the workflow system.

## 6.2 Schemas

Reusable schema definitions for:

- canonical entity records
- metadata fields
- document structures
- validation rules

These are product assets because future projects should use them.

## 6.3 Templates

Reusable templates for:

- feature specs
- implementation plans
- bug reports
- decision records
- project initialization files

Templates are product assets, not live project documents.

## 6.4 Default policies

Reusable default policies such as:

- review policy
- commit policy
- interaction protocol defaults
- metadata governance defaults
- validation policy defaults

These should eventually be represented in reusable, versioned form.

## 6.5 Default instructions and skills

Reusable runtime-facing materials that help agents interact correctly with the workflow system.

Examples:

- default agent interaction guidance
- normalization behavior guidance
- review behavior guidance
- commit discipline guidance

These belong to the product if they are intended for reuse across projects.

## 6.6 Initialization and migration logic

Anything that:

- creates a new project instance
- scaffolds a new repository
- migrates an existing repository to the workflow system

is part of the reusable product.

## 6.7 Product reference documentation

Reference docs that explain how to install, initialize, operate, or extend the system generically should be product assets.

---

## 7. Instance Artifacts

The following categories belong to a specific project instance and must not be confused with product assets.

## 7.1 Canonical state for the current project

Examples:

- current epics
- current features
- current tasks
- current bugs
- current decisions

These describe the project’s workflow state, not the reusable system itself.

## 7.2 Project-specific content

Examples:

- project-specific specs
- project-specific plans
- project-specific design decisions
- project-specific documentation updates
- project-specific overrides

These are instance artifacts.

## 7.3 Project-specific memory

Examples:

- project-specific knowledge
- project-specific conventions
- project-specific heuristics
- project-specific lessons learned

Even if useful, these are not product defaults until promoted.

## 7.4 Local cache and generated instance artifacts

Examples:

- local SQLite cache
- generated reports
- generated summaries
- temporary views

These are part of the running instance, not the product.

---

## 8. Project Design Artifacts

The following categories belong to the current design-and-build effort for the workflow system itself.

Examples:

- design basis
- phase specifications
- implementation planning docs
- review memos
- product-instance boundary notes
- policy drafts
- architecture rationale

These are not the live instance state and not automatically product assets.

They are the documents by which the product is being designed.

---

## 9. Promotion Path

A clear promotion path is needed so that the project can move from draft design to reusable product assets without confusion.

## 9.1 Draft stage

A concept starts as a project design document.

Examples:

- review policy draft
- interaction protocol draft
- schema design note
- directory layout proposal

At this stage:
- it is informative
- it is not yet a reusable asset
- it may change substantially

## 9.2 Stabilization stage

The concept is used enough in the current project that it becomes stable in practice.

Examples:

- repeated use of the same review policy
- repeated use of the same spec structure
- repeated use of the same agent protocol

At this stage:
- the concept has real evidence behind it
- it may still be documented as project policy
- promotion becomes worth considering

## 9.3 Productization stage

The concept is deliberately converted into reusable product assets.

Examples:

- a policy becomes a structured policy file
- a document pattern becomes a reusable template
- a validation rule becomes code or schema
- a protocol becomes a shipped default instruction/skill

At this stage:
- the asset is made generic enough for reuse
- current-project assumptions are removed
- versioning and compatibility become relevant

## 9.4 Instance initialization stage

The promoted product asset is used by init/bootstrap logic to create a new project instance.

At this stage:
- a future project receives the reusable default
- that new project then creates its own instance-specific state
- it does not inherit this project’s live state

---

## 10. Recommended Repository Boundary Model

The repository should eventually make the boundary visible in structure.

A good long-term model is:

- reusable product assets in product-oriented directories
- current project instance state in one dedicated instance root
- design/research/planning docs in `docs/`

A likely conceptual layout is:

- product code and reusable assets
- project instance root
- `docs/research/`
- `docs/design/`
- `docs/spec/`
- `docs/plan/`

The exact implementation layout can still evolve, but the domains must remain visible.

---

## 11. Recommended Instance Root

The live workflow instance for a project should eventually live in a single dedicated instance root.

Examples of possible forms:

- `.kbz/`
- `.workflow/`
- `workflow/`

The exact name remains open, but the concept is important.

This instance root should eventually contain things like:

- canonical state
- project-specific specs/plans if the chosen architecture puts them inside the instance
- local config and policy overrides
- generated views where appropriate
- project-specific memory
- derived local cache or a reference to it

The important point is not the exact name.  
The important point is that the instance has a clear home.

---

## 12. Boundary Rules for Policies and Protocols

Policies and protocols require extra hygiene because they may exist in all three domains at once.

For example, a review policy may exist as:

- a draft design document
- a project policy used in this project
- a reusable default policy shipped by the product

These must be kept distinct.

## 12.1 Draft policy

A design-time document under `docs/design/`.

## 12.2 Active project policy

The currently adopted policy for the live project instance.

## 12.3 Product default policy

A reusable asset intended to be initialized into future projects.

The project must not assume that every draft policy is already a reusable default.

---

## 13. Boundary Rules for Instructions and Skills

Instructions and skills should follow the same pattern.

## 13.1 Current project instructions

Instructions currently used to guide agents in this repository.

These may include project-specific assumptions and bootstrap guidance.

## 13.2 Reusable default instructions

Instructions that have been generalized enough to ship as part of the product for future projects.

These should be promoted deliberately.

## 13.3 Runtime-specific wrappers

Some instructions may need runtime-specific packaging.
That packaging belongs to the product only when it has been stabilized and generalized.

---

## 14. Boundary Rules for Workflow State

The workflow tool’s own development creates a special risk: the workflow system may eventually manage its own epics, features, bugs, and decisions.

That is acceptable — but those are still **instance records**, not product defaults.

For example:

- the workflow tool’s current bug list is not a reusable asset
- the workflow tool’s current roadmap is not a future-project scaffold
- the workflow tool’s current task decomposition is not a reusable template

The system may manage itself.  
That does **not** mean its current state becomes part of the reusable product.

---

## 15. Bootstrapping Model

The workflow should be designed to bootstrap in stages.

## 15.1 Stage 1: manual stewardship with supporting documents

At the beginning:
- design and policy live mostly in documents
- humans and general-purpose agents operate manually
- the workflow tool is immature
- the product/instance boundary is maintained by discipline

## 15.2 Stage 2: kernel-assisted project management

Once the workflow kernel exists:
- some project state moves into canonical records
- the workflow tool can begin to track its own development
- agents start using the formal interface for real work
- manual oversight remains strong

## 15.3 Stage 3: partial self-management

Once the system is mature enough:
- the workflow tool manages much of its own roadmap/tasks/bugs through its own process
- reusable policies begin to replace design-draft documents
- promotion paths become part of ordinary maintenance

## 15.4 Stage 4: reusable initialization for new projects

Once product assets are stabilized:
- a new project can initialize a clean workflow instance
- receive reusable defaults
- create project-specific state
- start without carrying along this repository’s current instance state

That is the end goal.

---

## 16. Hygiene Rules for Future Initialization

The final system should support a clean “new project” initialization path.

That requires:

- reusable defaults to live separately from current instance state
- initialization logic to scaffold a new instance
- no dependence on this project’s active records
- project-specific state to be created at init time, not copied from current live state
- product defaults to be generic enough to apply elsewhere

A new project should receive:
- product code
- default templates
- default policies
- default instructions
- initialization scaffolding

A new project should **not** receive:
- this project’s active epics/features/tasks/bugs
- this project’s temporary reports
- this project’s raw design history
- this project’s local cache
- this project’s accidental assumptions

---

## 17. Rules for Current Repository Organization

Until the product and instance are fully separated in implementation, the repository should still be organized according to the conceptual boundary.

Current practical rules:

1. `docs/research/` remains the research trail
2. `docs/design/` contains active design and policy drafts
3. `docs/spec/` contains implementation-facing specifications
4. `docs/plan/` contains implementation plans
5. future reusable product assets should not be assumed to live in these directories unless explicitly promoted
6. current project workflow state should eventually move into a dedicated instance root rather than being confused with product templates or design docs

---

## 18. Promotion Criteria

An artifact should only be promoted into the reusable product if all of the following are true:

1. it has been used enough to prove it is not just a draft idea
2. it has been stripped of accidental current-project assumptions
3. it is useful outside the current repository
4. its purpose is stable enough to version
5. it fits clearly into the product rather than the current instance

If these conditions are not met, the artifact should stay in the project design domain.

---

## 19. Acceptance Criteria

This boundary design is acceptable only if it makes the following true:

1. it is always clear whether an artifact is product, instance, or project-design material
2. a future initialization path can copy reusable defaults without copying current project state
3. the workflow tool can eventually manage its own development without collapsing product and instance boundaries
4. draft design documents do not silently become shipped defaults
5. current project state does not pollute reusable assets
6. promotion from draft to reusable asset is explicit and reviewable

---

## 20. Summary

The workflow system must maintain a clear hygiene boundary between:

- the reusable product
- the current project instance
- the current project’s design and implementation planning materials

This boundary is necessary because the system is being built while also being prepared to manage itself.

The key rule is:

> Product assets are reusable.  
> Instance state is project-specific.  
> Design materials describe the product, but do not automatically become it.

The architecture should support:
- manual stewardship during bootstrap
- gradual transition toward self-management
- explicit promotion of stable design artifacts into reusable product assets
- clean initialization of future projects without inheriting this repository’s live state

This document exists so that the workflow system can eventually be reused cleanly without confusing the tool, its current instance, and the documents used to design it.