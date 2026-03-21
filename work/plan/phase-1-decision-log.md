# Phase 1 Decision Log

- Status: active — agent-maintained internal record
- Purpose: index decisions made during design conversations and in design documents
- Date: 2026-03-18
- Related:
  - `work/design/document-centric-interface.md`
  - `work/plan/phase-1-implementation-plan.md`
  - `work/spec/phase-1-specification.md`
  - `work/design/workflow-design-basis.md`
- Note: Under the document-centric interface model, decisions are made in design documents and in conversation between human designers and AI agents. This log is maintained by agents as an internal index — the human designer does not need to write decision records directly. Decisions recorded here should also appear in the design documents they inform.

---

## 1. Purpose

This document records the important implementation decisions for Phase 1 of the workflow system.

It exists to prevent major design choices from remaining implicit during implementation.

The Phase 1 implementation plan identifies several areas that must be decided early, including:

- ID allocation strategy
- canonical file layout
- structured file format constraints
- lifecycle transition rules
- MCP operation naming and structure
- CLI command mapping
- cache scope and rebuild behavior
- bootstrap approach for using the kernel on the project itself

This document is the place to track those decisions.

---

## 2. How to Use This Log

Each decision should be recorded as one entry.

Each entry should include:

- a stable decision ID
- title
- status
- date
- scope
- decision statement
- rationale
- alternatives considered
- consequences
- links to affected documents or implementation areas

Suggested statuses:

- `proposed`
- `accepted`
- `deferred`
- `rejected`
- `superseded`

This log is for implementation decisions, not product roadmap work.

---

## 3. Decision Entry Template

## `P1-DEC-XXX: <title>`

- Status:
- Date:
- Scope:
- Related:

### Decision

### Rationale

### Alternatives Considered

### Consequences

### Follow-up Needed

---

## 4. Active Phase 1 Decision Topics

The following decision topics need to be resolved before or early in implementation.

## 4.1 ID allocation strategy

This decision must answer:

- what concrete ID allocation strategy will Phase 1 use?
- how will it support ordinary concurrent use safely?
- how replaceable is the strategy later?
- what edge cases are acceptable in Phase 1?

Options currently under consideration include:

- block allocation
- simpler sequential allocation with operational constraints
- distributed sortable IDs
- hybrid approaches

This is a critical decision because it affects:
- canonical record naming
- workflow references
- commit traceability
- migration behavior
- concurrency safety

---

## 4.2 Canonical file layout

This decision must answer:

- where canonical state files will live
- whether a dedicated instance root is introduced in Phase 1
- how product artifacts and instance artifacts are kept separate
- how the current repository avoids product/instance leakage during bootstrap

This is a critical decision because it affects:
- repository hygiene
- bootstrapping
- future initialization
- migration path
- local developer experience

---

## 4.3 YAML subset and formatting constraints

This decision must answer:

- what exact YAML subset is allowed?
- what formatting is canonical?
- how strict deterministic output will be enforced?
- what values must be normalized automatically?
- what ambiguities are explicitly forbidden?

This is important because it affects:
- diffs
- merges
- validation
- human editability
- long-term stability of canonical state

---

## 4.4 Minimum required fields per entity

This decision must answer:

- what is the exact required field set for each phase 1 entity?
- which fields are required at creation time?
- which fields are optional but recommended?
- which fields may be empty in early states?

This is important because it affects:
- normalization behavior
- MCP validation
- document scaffolding
- health checks
- user friction

---

## 4.5 Lifecycle transition graph

This decision must answer:

- what exact transitions are allowed for each entity type?
- which transitions require stronger validation?
- how composite Feature state is interpreted in Phase 1
- how invalid or partial transitions are handled

This is important because it affects:
- state correctness
- workflow integrity
- CLI/MCP behavior
- review and merge readiness

---

## 4.6 MCP operation names and shapes

This decision must answer:

- what the exact Phase 1 MCP operation names are
- what their request/response structures are
- how errors are represented
- how candidate validation is represented
- what is included in create/query/update/validate operations

This is important because it affects:
- agent interoperability
- implementation clarity
- testability
- future policy query support

---

## 4.7 CLI mapping

This decision must answer:

- what CLI commands exist in Phase 1
- how closely they mirror MCP operations
- what output formats are supported
- which commands are primarily for humans vs CI
- how strict/non-interactive the CLI is

Design constraint: the CLI is a human interface. Wherever possible, commands should read like English — `kbz create epic` rather than `kbz epic create`. This isn't always achievable (some operations don't map to natural verb-noun phrasing), so cases that don't fit should be consulted on rather than forced into the pattern.

This is important because it affects:
- bootstrap usability
- manual stewardship
- CI integration
- debugging and repair

---

## 4.8 Local cache scope and rebuild behavior

This decision must answer:

- what the local derived cache contains
- what queries depend on it
- whether health checks depend on it
- how it is rebuilt
- whether cache absence should ever block ordinary use

This is important because it affects:
- performance
- reliability
- startup behavior
- local workflow simplicity

---

## 4.9 Document template structures

This decision must answer:

- what templates are required in Phase 1
- what sections/frontmatter each template must include
- what fields are project-specific vs reusable defaults
- what validation rules apply to each document type

This is important because it affects:
- document scaffolding
- validation
- normalization of human-authored docs
- future template promotion into reusable product assets

---

## 4.10 Bootstrap approach for using the kernel on this project

This decision must answer:

- when the kernel is trusted enough to start tracking its own work
- what minimum records are created first
- whether the project starts with one epic or a broader initial structure
- what remains manual during early bootstrap
- what safeguards apply during early self-use

This is important because it affects:
- bootstrapping discipline
- confidence in the kernel
- product/instance separation
- implementation planning order

---

## 5. Decision Register

## `P1-DEC-001: Phase 1 uses a narrow core entity set`

- Status: accepted
- Date: 2026-03-18
- Scope: Phase 1 entity model
- Related:
  - `work/spec/phase-1-specification.md`
  - `work/plan/phase-1-implementation-plan.md`

### Decision

Phase 1 will implement only the following first-class entities:

- `Epic`
- `Feature`
- `Task`
- `Bug`
- `Decision`

### Rationale

This is the minimum entity set needed to solve the most serious consistency failures without overbuilding the system before the kernel exists.

### Alternatives Considered

- implement a richer first-class model immediately, including `Specification`, `Plan`, `Approval`, and `Release`
- implement a still smaller model and defer `Decision`

### Consequences

- the Phase 1 implementation can stay narrower
- `Feature` remains composite for now
- some concepts remain deferred and must not leak into implementation accidentally

### Follow-up Needed

- define exact schemas for the five entities
- define deferred-entity handling clearly

---

## `P1-DEC-002: Feature remains composite in Phase 1`

- Status: accepted
- Date: 2026-03-18
- Scope: Phase 1 object model
- Related:
  - `work/spec/phase-1-specification.md`
  - `work/design/workflow-design-basis.md`

### Decision

In Phase 1, `Feature` remains a composite entity that may carry:

- feature identity
- links to spec and plan documents
- approval-related status
- implementation lifecycle status

### Rationale

This keeps Phase 1 simpler while still allowing the system to represent the current workflow meaningfully.

### Alternatives Considered

- make `Specification` and `Plan` first-class immediately
- defer plan/spec support further and keep Feature more minimal

### Consequences

- lifecycle semantics for Feature are less pure
- approval and supersession concerns are somewhat compressed
- later phases may split Feature into richer first-class entities

### Follow-up Needed

- ~~define clear lifecycle transitions~~ — **Resolved:** P1-DEC-010 defines the exhaustive Feature lifecycle transition graph.
- ~~avoid overcomplicating the composite model~~ — **Resolved:** ongoing design constraint; the composite model has remained simple through implementation.
- ~~preserve migration path toward first-class `Specification` and `Plan`~~ — **Resolved:** this is a Phase 2 design concern. Phase 1's composite model does not preclude the split — Feature's optional `spec` and `plan` fields can become foreign keys to separate entities in a future phase without breaking existing records.

---

## `P1-DEC-003: Phase 1 is MCP-first, with CLI as secondary`

- Status: accepted
- Date: 2026-03-18
- Scope: interfaces
- Related:
  - `work/spec/phase-1-specification.md`
  - `work/design/agent-interaction-protocol.md`

### Decision

The primary formal interface for Phase 1 is MCP. A strict CLI will exist, but as a secondary interface using the same shared core logic.

### Rationale

Humans are expected to work primarily through natural-language chat with agents, while agents use the formal machine interface. A secondary CLI still supports debugging, CI, repair, and bootstrap operation.

### Alternatives Considered

- CLI-first with later MCP support
- MCP-only with no CLI
- dual-first model with equal emphasis

### Consequences

- implementation should prioritize formal machine operations first
- CLI should be thin and strict
- agent behavior and MCP semantics become central early

### Follow-up Needed

- ~~define exact MCP operation names and shapes~~ — **Resolved:** P1-DEC-011 accepted with full MCP tool surface documented.
- ~~define CLI mapping cleanly~~ — **Resolved:** P1-DEC-012 accepted with full CLI command surface documented.

---

## `P1-DEC-004: Phase 1 excludes orchestration`

- Status: accepted
- Date: 2026-03-18
- Scope: phase boundary
- Related:
  - `work/spec/phase-1-specification.md`
  - `work/plan/phase-1-implementation-plan.md`

### Decision

Phase 1 will not implement orchestration, automatic decomposition, or delegation-chain management.

### Rationale

The workflow kernel must be trustworthy before automation is layered on top of it. Building orchestration too early would increase scope and risk.

### Alternatives Considered

- implement a minimal orchestrator in Phase 1
- implement limited task decomposition in Phase 1

### Consequences

- Phase 1 remains simpler
- some operations remain manual or manually initiated by general-purpose agents
- the kernel can be tested independently of complex orchestration behavior

### Follow-up Needed

- keep implementation free of hidden orchestration assumptions
- preserve a path for later orchestration support

---

## `P1-DEC-005: Phase 1 must support limited bootstrap self-use`

- Status: accepted
- Date: 2026-03-18
- Scope: bootstrap behavior
- Related:
  - `work/design/product-instance-boundary.md`
  - `work/spec/phase-1-specification.md`

### Decision

Phase 1 must be sufficient to begin tracking limited work on the workflow tool itself, but without assuming mature self-management.

### Rationale

The system is being built before the process is fully embodied in tooling. It should become a first-class user of its own workflow gradually, not magically.

### Alternatives Considered

- forbid self-use until Phase 2 or later
- attempt broad self-management immediately

### Consequences

- some manual stewardship remains necessary
- early instance use must be cautious and explicit
- product/instance hygiene becomes even more important

### Follow-up Needed

- decide how the initial project instance is introduced
- decide what first records should be created when the kernel is ready

---

## 6. Proposed Near-Term Decision Sequence

The recommended order for resolving remaining Phase 1 decisions is:

1. `P1-DEC-009` — exact required fields by entity
2. `P1-DEC-010` — exact lifecycle transition graph
3. `P1-DEC-011` — exact MCP operation names and request/response shapes
4. `P1-DEC-012` — CLI mapping
5. `P1-DEC-013` — local cache scope
6. `P1-DEC-015` — bootstrap introduction strategy for using the kernel on this project

This order is recommended because early decisions constrain later interface and validation work.

---

## 7. Placeholder Entries for Immediate Planning

## `P1-DEC-006: Canonical file layout and instance root strategy`

- Status: accepted
- Date: 2026-03-18
- Scope: storage boundary
- Related:
  - `work/design/product-instance-boundary.md`
  - `work/plan/phase-1-implementation-plan.md`
  - `work/spec/phase-1-specification.md`

### Decision

Phase 1 will introduce a dedicated instance root at:

- `.kbz/`

The canonical project instance for this repository will live under that root.

Phase 1 canonical state and project-instance workflow materials will be placed inside `.kbz/`, while reusable product code and reusable product assets will remain outside it.

The Phase 1 layout principle is:

- reusable product assets live in product-oriented directories
- project-instance workflow state lives in `.kbz/`
- design, research, specification, and planning documents about building the workflow system remain in `docs/`

Within Phase 1, the instance root should contain at least:

- `.kbz/state/`
- `.kbz/specs/`
- `.kbz/plans/`
- `.kbz/cache/` or another clearly derived local-cache location
- additional instance-scoped directories only where clearly justified by phase-1 scope

This means the current repository will explicitly separate:

- reusable system code and reusable assets
- live workflow instance state for this project
- project design and planning documents

### Rationale

A dedicated instance root makes the product/instance boundary visible early, which is important because this project is building a reusable workflow system while also beginning to use that system on itself.

Using `.kbz/` in Phase 1 has several advantages:

- it creates a single clear home for instance state
- it reduces the risk of current project state leaking into reusable product assets
- it makes future project initialization easier to reason about
- it gives Phase 1 a practical place to begin bootstrap self-use
- it preserves the distinction between project-design docs in `docs/` and live workflow state
- it provides a path toward later richer instance behavior without changing the conceptual model

This decision favors hygiene and future reuse over short-term convenience.

### Alternatives Considered

- introduce a dedicated instance root in Phase 1
- defer the dedicated instance root while preserving the conceptual boundary
- mixed temporary approach
- use a visible non-hidden directory such as `workflow/`
- use a different hidden directory such as `.workflow/`

The deferred or mixed approaches were rejected because they would keep the conceptual boundary but delay making it operational, increasing the risk of product/instance leakage during bootstrap.

A visible directory such as `workflow/` remains viable in principle, but `.kbz/` was chosen because it clearly communicates "project instance state for the workflow system" and keeps that state distinct from reusable product directories and project-design docs.

### Consequences

- Phase 1 implementation must treat `.kbz/` as the project-instance root
- canonical state files for this project should not be placed in reusable product directories
- project-specific specs and plans that are part of the live instance should migrate toward `.kbz/`
- `docs/` remains the home for design, research, specification, and planning documents about building the workflow system itself
- local cache behavior must be designed relative to `.kbz/`
- initialization logic in later phases can target `.kbz/` consistently
- implementation planning can now make concrete assumptions about instance paths

This also means future promotion of reusable templates, schemas, and default policies must happen outside `.kbz/`, not inside it.

### Follow-up Needed

- define the exact phase-1 directory structure under `.kbz/`
- define which existing or future documents belong in `.kbz/` versus `docs/`
- define cache placement and rebuild behavior relative to `.kbz/`
- ensure implementation code does not hardcode current-project design-document paths as if they were instance defaults

---

## `P1-DEC-007: Phase 1 ID allocation strategy`

- Status: superseded by P1-DEC-021
- Date: 2026-03-18
- Scope: identity
- Related:
  - `work/spec/phase-1-specification.md`
  - `work/plan/phase-1-implementation-plan.md`
  - `work/design/workflow-design-basis.md`
  - `work/design/id-system-design.md`

### Decision

Phase 1 will use a hybrid ID strategy by object class.

Human-facing core objects will use short, human-friendly, sequential IDs:

- `Epic` → `E-001`, `E-002`, ...
- `Feature` → `FEAT-001`, `FEAT-002`, ...
- `Bug` → `BUG-001`, `BUG-002`, ...
- `Decision` → `DEC-001`, `DEC-002`, ...

`Task` IDs in Phase 1 will be feature-local:

- `FEAT-001.1`
- `FEAT-001.2`
- `FEAT-001.3`

The canonical identifier is the typed ID. The human-readable slug remains a separate part of filenames and references.

The Phase 1 system will not encode owner identity inside canonical IDs.

The Phase 1 system will not use deep hierarchical address-style IDs across multiple layers such as Epic → Feature → Task in a single chained identifier.

Hierarchy is represented by explicit links between records, not by long structural IDs.

### Rationale

Different classes of workflow objects have different needs.

High-level objects such as Epics, Features, Bugs, and Decisions are:

- few in number
- frequently referenced by humans
- commonly discussed in chat and documents
- stable enough to benefit from human-friendly IDs

Tasks are more numerous and more operational. In Phase 1, feature-local task IDs provide a good balance between readability and simplicity without requiring a fully separate high-entropy task ID system.

This strategy keeps the most human-visible identifiers pleasant to read and speak while avoiding overcomplicating Phase 1.

It also avoids embedding unstable information into IDs:

- ownership can change
- hierarchy can change
- scope can evolve

Those relationships should live in fields and links, not inside the identifier itself.

### Alternatives Considered

- block allocation for all entity types, including globally allocated task IDs
- simple sequential allocation with constraints for all entity types
- distributed sortable IDs for all entity types
- hybrid approach

A fully distributed sortable-ID scheme remains viable for later higher-volume machine-generated entities, but was rejected for Phase 1 because it would make the most human-facing IDs unnecessarily awkward.

A deeply hierarchical address-style scheme was rejected because hierarchy is not stable enough to encode in canonical IDs without creating misleading or brittle identifiers.

Embedding developer, designer, or manager identity in IDs was rejected because ownership should be metadata, not identity.

### Consequences

- Phase 1 IDs remain friendly for humans where it matters most
- task IDs remain simple and readable during early implementation
- the implementation must provide separate allocation behavior for:
  - core human-facing entities
  - feature-local tasks
- hierarchy must be modeled explicitly in canonical records
- owner identity must be stored in fields such as `created_by`, `assignee`, or similar metadata, not in IDs
- later phases may introduce different ID treatment for higher-volume machine-generated entities without breaking the Phase 1 model for core objects

### Follow-up Needed

- define the exact allocation mechanism for sequential Epic/Feature/Bug/Decision IDs
- define how task sub-IDs are allocated safely within a feature
- test the chosen strategy against normal concurrent-use scenarios
- ensure filenames and commit references consistently use `ID + slug`

---

## `P1-DEC-008: YAML subset and formatting rules`

- Status: accepted
- Date: 2026-03-18
- Scope: canonical representation
- Related:
  - `work/spec/phase-1-specification.md`
  - `work/plan/phase-1-implementation-plan.md`
  - `work/design/workflow-design-basis.md`

### Decision

Phase 1 will use YAML as the canonical on-disk format for structured workflow state, but only as a strict canonical subset.

The Phase 1 YAML rules are:

- canonical workflow files are written by the workflow tool
- block style only
- deterministic schema-defined key ordering
- no anchors
- no aliases
- no merge keys
- no custom tags
- no flow-style collections
- IDs are always stored as strings
- timestamps are always stored as normalized ISO 8601 strings
- values with ambiguous scalar interpretation must be quoted
- long prose should be minimized in YAML and kept in markdown documents where appropriate
- semantically equivalent data must always serialize identically

Humans may edit YAML directly in exceptional cases, but the tool is the canonical writer and normalizer of canonical state files.

### Rationale

Full YAML is too flexible for a Git-native canonical workflow store. It allows too many equivalent representations, too much formatting variation, and too many parser-dependent ambiguities.

A strict YAML subset preserves the main benefits of YAML:

- readability
- familiarity
- structured nesting
- inspectability

while avoiding the parts that make Git diffs, merges, validation, and agent-generated updates unstable.

The purpose of this decision is not to maximize YAML expressiveness. It is to maximize:

- determinism
- diff stability
- merge friendliness
- validator simplicity
- review clarity
- low churn under AI and human edits

This also supports the broader design principle that one canonical fact should be represented in one stable way.

### Alternatives Considered

- strict YAML subset
- more permissive YAML
- alternative textual format if necessary

A more permissive YAML approach was rejected because it would allow unnecessary variation in formatting and interpretation, which would create noisy diffs, merge friction, and inconsistent agent output.

Switching immediately to another textual format such as TOML was considered unnecessary at this stage. YAML remains acceptable so long as the project uses it as a constrained canonical serialization format rather than as a free-form authoring language.

### Consequences

- the tool must act as the canonical writer for structured state files
- validation must reject unsupported YAML constructs
- field ordering must be explicitly defined by schema, not left to incidental map ordering
- canonical records become more stable under Git
- AI agents and humans have less formatting freedom in canonical YAML, which is intentional
- long-form prose should continue to live primarily in markdown rather than being embedded extensively in YAML
- future document/template decisions should assume YAML is for structured records, not for free-form narrative

### Follow-up Needed

- define the exact schema-defined field order for each phase 1 entity type
- define quoting rules for ambiguous scalar values
- define the precise validation behavior for unsupported YAML constructs
- ensure the implementation plan treats the tool as the canonical writer for canonical state files

---

## `P1-DEC-009: Minimum required fields by entity`

- Status: accepted
- Date: 2026-03-18
- Scope: entity schemas
- Related:
  - `work/spec/phase-1-specification.md`
  - `work/design/agent-interaction-protocol.md`

### Decision

Phase 1 ratifies the minimum required fields from spec §9 for all five entity types, and classifies each field into one of three categories at creation time:

1. **System-generated** — the system provides these automatically; the caller must not supply them.
2. **Defaultable** — the system applies a sensible default; the caller may override.
3. **Caller-must-supply** — the caller must provide these; no safe default exists.

All fields listed in spec §9 remain required on the canonical record. The classification determines how they are populated at creation, not whether they appear in the stored entity.

#### Epic

| Field | Category | Default |
|---|---|---|
| `id` | system-generated | allocated via `id_allocate` |
| `status` | system-generated | `proposed` |
| `created` | system-generated | current timestamp |
| `slug` | caller-must-supply | — |
| `title` | caller-must-supply | — |
| `summary` | caller-must-supply | — |
| `created_by` | caller-must-supply | — |

Caller supplies: `slug`, `title`, `summary`, `created_by` (4 fields).

#### Feature

| Field | Category | Default |
|---|---|---|
| `id` | system-generated | allocated via `id_allocate` |
| `status` | system-generated | `draft` |
| `created` | system-generated | current timestamp |
| `slug` | caller-must-supply | — |
| `epic` | caller-must-supply | — |
| `summary` | caller-must-supply | — |
| `created_by` | caller-must-supply | — |

Caller supplies: `slug`, `epic`, `summary`, `created_by` (4 fields).

#### Task

| Field | Category | Default |
|---|---|---|
| `id` | system-generated | allocated as feature-local sub-ID |
| `status` | system-generated | `queued` |
| `slug` | caller-must-supply | — |
| `feature` | caller-must-supply | — |
| `summary` | caller-must-supply | — |

Caller supplies: `slug`, `feature`, `summary` (3 fields).

#### Bug

| Field | Category | Default |
|---|---|---|
| `id` | system-generated | allocated via `id_allocate` |
| `status` | system-generated | `reported` |
| `reported` | system-generated | current timestamp |
| `severity` | defaultable | `medium` |
| `priority` | defaultable | `medium` |
| `type` | defaultable | `implementation-defect` |
| `slug` | caller-must-supply | — |
| `title` | caller-must-supply | — |
| `reported_by` | caller-must-supply | — |
| `observed` | caller-must-supply | — |
| `expected` | caller-must-supply | — |

Caller supplies: `slug`, `title`, `reported_by`, `observed`, `expected` (5 fields).

#### Decision

| Field | Category | Default |
|---|---|---|
| `id` | system-generated | allocated via `id_allocate` |
| `status` | system-generated | `proposed` |
| `date` | system-generated | current timestamp |
| `slug` | caller-must-supply | — |
| `summary` | caller-must-supply | — |
| `rationale` | caller-must-supply | — |
| `decided_by` | caller-must-supply | — |

Caller supplies: `slug`, `summary`, `rationale`, `decided_by` (4 fields).

Note: `status` was added to this table as a correction. The original version omitted it, but Decision lifecycle states (`proposed`, `accepted`, `rejected`, `superseded`) are defined in P1-DEC-010 and require a `status` field. The omission was an oversight — every other entity type included `status` as system-generated. The implementation has always set `status: "proposed"` at creation time. See also the corresponding correction in spec §9.5.

#### Default values for Bug fields

The following defaults apply when the caller does not supply a value:

- `severity: medium` — safe middle ground; triage is expected to adjust.
- `priority: medium` — same reasoning; triage adjusts.
- `type: implementation-defect` — the most common class; spec and design defects are rarer and typically identified at triage.

These defaults signal "not yet triaged" rather than an assessed judgment. They reduce creation friction while ensuring the canonical record is always fully populated.

#### Distinction from agent inference

A system-applied default is not the same as an agent inferring a value from context. The agent interaction protocol (§6.3) prohibits agents from silently inventing severity, priority, and similar fields. That prohibition applies to agents guessing values from the content of a report. A well-known, transparent system default that marks a field as "not yet assessed" does not violate this rule. Agents should not override these defaults with inferred values without surfacing the inference to the human.

### Rationale

The spec §9 field lists are already well-considered. The question was never which fields to require on the stored record, but how many the caller must explicitly provide at creation time.

Classifying fields into three categories resolves this cleanly:

- System-generated fields (`id`, `status`, timestamps) are deterministic and should not burden the caller. Initial `status` is always the entry state of the entity's lifecycle; timestamps are always "now."
- Defaultable fields (Bug `severity`, `priority`, `type`) have a safe, well-understood baseline that triage is expected to review. This keeps bug creation lightweight (5 caller fields instead of 8) without producing incomplete records.
- Caller-must-supply fields are the substance of the entity — the things that only a human or informed agent can provide. No safe default exists for a bug's title or observed behavior.

This approach avoids the complexity of a staged required-field model (where different fields become required at different lifecycle states) while still keeping creation ergonomic. Phase 1 does not need staged requirements because the defaultable category handles the only cases where creation-time strictness would create friction.

### Alternatives Considered

- narrower minimum fields
- richer required fields at creation time
- staged required-field model by status

**Narrower minimum fields** — removing fields like `severity` or `priority` from the required set entirely. Rejected because these fields are needed for triage and prioritization. Making them optional risks records that never get them filled in. A default is better than absence.

**Richer required fields at creation time** — requiring the caller to supply all 11 Bug fields explicitly. Rejected because it creates unnecessary friction when creating bugs, especially through conversational intake. Fields like severity and priority are legitimately unknown at report time and are the responsibility of triage, not the reporter.

**Staged required-field model by status** — different lifecycle states require different field sets (e.g., `severity` only required after `triaged`). This is the most flexible option but adds significant validation complexity for Phase 1. The defaultable category achieves the same ergonomic benefit without status-dependent schema rules. A staged model could be revisited in later phases if needed.

### Consequences

- MCP create operations accept only caller-must-supply fields as required parameters, with defaultable fields as optional parameters
- the system must populate all system-generated fields automatically at creation time
- the system must apply documented defaults for defaultable fields when not supplied by the caller
- validation must ensure the canonical record always contains all spec §9 fields, regardless of how they were populated
- agents must not override defaultable fields with inferred values without surfacing the inference to the human
- triage is expected to review and adjust default values for Bug `severity`, `priority`, and `type`
- the default values are a system behavior, not an agent judgment — this distinction must be preserved in implementation and documentation
- P1-DEC-011 (MCP operation shapes) can now derive required vs. optional parameters from this classification

### Follow-up Needed

- ensure MCP create operation request shapes (P1-DEC-011) align with the caller-must-supply / defaultable distinction
- ~~define whether `created_by` / `reported_by` / `decided_by` can be inferred from the authenticated caller identity or must always be explicit~~ — resolved: these fields may be inferred from the authenticated caller identity when available, making them effectively defaultable rather than caller-must-supply in contexts where caller identity is known
- confirm the exact allowed values for `severity`, `priority`, and `type` as part of schema definition

---

## `P1-DEC-010: Lifecycle transition graph`

- Status: accepted
- Date: 2026-03-18
- Scope: validation
- Related:
  - `work/spec/phase-1-specification.md`
  - `work/design/workflow-system-design.md`

### Decision

Phase 1 ratifies the lifecycle states from spec §10 and defines the exact legal transitions for all five entity types. The system must reject any transition not listed here.

Every transition table below is exhaustive — if a `from → to` pair is not listed, it is illegal.

#### Epic

States: `proposed`, `approved`, `active`, `on-hold`, `done`

| From | To | Notes |
|---|---|---|
| `proposed` | `approved` | human approval required |
| `approved` | `active` | work begins |
| `active` | `on-hold` | paused; can resume |
| `active` | `done` | all work complete |
| `on-hold` | `active` | resumed |
| `on-hold` | `done` | closed while paused |

Entry state: `proposed`
Terminal states: `done`

#### Feature

States: `draft`, `in-review`, `approved`, `in-progress`, `review`, `needs-rework`, `done`, `superseded`

| From | To | Notes |
|---|---|---|
| `draft` | `in-review` | submitted for spec review |
| `in-review` | `approved` | spec accepted |
| `in-review` | `needs-rework` | spec rejected; needs revision |
| `approved` | `in-progress` | implementation begins |
| `approved` | `superseded` | replaced before work started |
| `in-progress` | `review` | implementation submitted for review |
| `in-progress` | `needs-rework` | problem found during implementation |
| `review` | `done` | implementation accepted |
| `review` | `needs-rework` | implementation rejected |
| `needs-rework` | `in-review` | revised spec resubmitted |
| `needs-rework` | `in-progress` | revised implementation resumed |
| `done` | `superseded` | replaced after completion |

Entry state: `draft`
Terminal states: `done`, `superseded`

Note: `needs-rework` serves double duty in the composite Feature lifecycle — it can represent either spec rework (returns to `in-review`) or implementation rework (returns to `in-progress`). The transition target disambiguates which kind of rework is happening. If Feature is decomposed in a future phase, this state would split.

#### Task

States: `queued`, `ready`, `active`, `blocked`, `needs-review`, `needs-rework`, `done`

| From | To | Notes |
|---|---|---|
| `queued` | `ready` | prerequisites met; eligible for work |
| `ready` | `active` | work begins |
| `active` | `blocked` | waiting on external dependency |
| `active` | `needs-review` | work submitted for review |
| `blocked` | `active` | blocker resolved |
| `needs-review` | `done` | review accepted |
| `needs-review` | `needs-rework` | review rejected |
| `needs-rework` | `active` | rework begins |

Entry state: `queued`
Terminal states: `done`

#### Bug

States: `reported`, `triaged`, `reproduced`, `planned`, `in-progress`, `needs-review`, `verified`, `closed`, `duplicate`, `not-planned`, `cannot-reproduce`

| From | To | Notes |
|---|---|---|
| `reported` | `triaged` | initial assessment complete |
| `reported` | `duplicate` | identified as duplicate during triage |
| `triaged` | `reproduced` | reproduction confirmed |
| `triaged` | `cannot-reproduce` | unable to reproduce |
| `triaged` | `not-planned` | acknowledged but will not fix |
| `triaged` | `duplicate` | identified as duplicate after triage |
| `triaged` | `planned` | reproduction not required; fix planned directly |
| `reproduced` | `planned` | fix approach determined |
| `reproduced` | `not-planned` | reproduced but will not fix |
| `planned` | `in-progress` | fix work begins |
| `in-progress` | `needs-review` | fix submitted for review |
| `needs-review` | `verified` | fix confirmed working |
| `needs-review` | `needs-rework` | fix rejected |
| `needs-rework` | `in-progress` | rework begins |
| `verified` | `closed` | fix shipped / accepted |
| `cannot-reproduce` | `triaged` | reopened with new information |

Entry state: `reported`
Terminal states: `closed`, `duplicate`, `not-planned`

Note: `cannot-reproduce` is near-terminal but allows reopening if new reproduction information arrives. `duplicate` and `not-planned` are fully terminal.

Note: the `triaged → planned` transition allows skipping `reproduced` for cases where the defect is obvious from code inspection or the report is sufficiently clear. This avoids forcing ceremony on trivial bugs.

#### Decision

States: `proposed`, `accepted`, `rejected`, `superseded`

| From | To | Notes |
|---|---|---|
| `proposed` | `accepted` | decision ratified |
| `proposed` | `rejected` | decision declined |
| `accepted` | `superseded` | replaced by a newer decision |

Entry state: `proposed`
Terminal states: `rejected`, `superseded`

Note: `accepted` is not terminal because decisions can be superseded. But `accepted` with no `superseded_by` link represents a current, active decision.

#### General rules

The following rules apply across all entity types:

1. **Entry states are system-enforced.** New entities always start in their entry state (as established in P1-DEC-009). The system must not allow creation in any other state.
2. **Terminal states are irreversible.** Once an entity reaches a terminal state, no further transitions are allowed. The only way to "undo" a terminal state is to create a new entity (potentially linked via `supersedes`).
3. **Self-transitions are illegal.** A transition from a state to itself is not a valid transition. If nothing changes, nothing should be recorded.
4. **Unknown states are rejected.** The system must reject any status value not in the entity's state list.

### Rationale

The design documents provide ASCII diagrams showing the primary flow paths, and the spec §10 lists the required states, but neither enumerates every legal edge. This decision fills that gap with an exhaustive transition table per entity type.

The graphs are designed to be strict but not punitive:

- Every state is reachable from the entry state through some path of legal transitions.
- Terminal states are clearly identified and irreversible, which prevents accidental reopening of completed work.
- Rework loops (`needs-rework → active/in-progress/in-review`) are explicit, avoiding the need to jump backward through intermediate states.
- The Bug lifecycle includes a small number of practical shortcuts (`triaged → planned` for obvious bugs, `reported → duplicate` for early duplicate detection) that avoid forcing unnecessary ceremony without undermining the process.

The `cannot-reproduce → triaged` reopening path is the one exception to strict terminal-state irreversibility, reflecting the reality that reproduction evidence can arrive later. This is preferable to closing and re-filing, which would lose context.

### Alternatives Considered

- strict narrow transition graph
- looser graph with validation warnings
- entity-specific rules with phase-specific exceptions

**Strict narrow transition graph** — only the linear happy-path transitions with no shortcuts or reopening. Rejected because it forces unnecessary ceremony (e.g., a clearly duplicate bug must go through triage before being marked duplicate) and doesn't reflect real workflow patterns.

**Looser graph with validation warnings** — allow any transition but warn on unusual ones. Rejected because it undermines the core design principle that the system enforces process integrity. Warnings are easily ignored, and the value of a state machine is that it constrains. If a transition is reasonable, it should be legal; if it isn't, it should be rejected, not warned about.

**Entity-specific rules with phase-specific exceptions** — allow certain "escape hatch" transitions in Phase 1 that would be removed later. Rejected because it creates a migration burden and trains users and agents to rely on transitions that will disappear. Better to get the graph right now and extend it later if needed.

### Consequences

- the system must implement per-entity-type transition validation
- MCP `status_update` operations must check the proposed transition against the entity's transition table before applying it
- health checks must detect entities in unknown states or with transition history that includes illegal jumps
- agents must understand which transitions are available from the current state, and should not propose illegal transitions
- the transition tables are exhaustive — extending them later requires a new decision or an amendment to this one
- `cannot-reproduce` occupies a special position as a near-terminal state; implementation must handle the reopening path
- the `needs-rework` state on Feature requires the system to accept two different outbound transitions depending on context, which the implementation must support cleanly

### Follow-up Needed

- ~~determine whether transition history should be stored on the entity record (e.g., a `transitions` log) or derived from Git history~~ — **Resolved:** Phase 1 derives transition history from Git commit history. No history is stored on entity records. A dedicated transition log feature is planned for post-Phase 1 (see `work/design/transition-log-design.md`).
- ~~define what metadata a transition carries (e.g., `transitioned_by`, `reason`, timestamp) — this feeds into P1-DEC-011~~ — **Resolved:** No transition metadata is stored on entities in Phase 1. Commit messages carry context. The post-Phase 1 transition log will include `from`, `to`, `at`, `by`, and optional `reason` fields per entry.
- ~~confirm whether `needs-rework` on Feature needs any disambiguation metadata to distinguish spec rework from implementation rework~~ — **Resolved:** No disambiguation metadata needed. The transition target already disambiguates: `needs-rework → in-review` indicates spec rework, `needs-rework → in-progress` indicates implementation rework. A free-text `reason` field in the future transition log will cover cases where additional context is useful.

---

## `P1-DEC-011: MCP operation names and request/response shapes`

- Status: accepted
- Date: 2026-03-18
- Accepted: 2026-07-24
- Scope: MCP interface
- Related:
  - `work/plan/phase-1-implementation-plan.md`
  - `work/spec/phase-1-specification.md` §16

### Decision

Phase 1 uses **object-specific operation naming** for entity creation (one tool per entity type) and **generic naming** for retrieval, status updates, and field updates (shared tools with an `entity_type` parameter). Document operations are namespaced under `*_document` names. The server exposes 20 MCP tools total: 12 entity tools and 8 document tools.

#### Entity tools (12)

| Tool | Required parameters | Notes |
|---|---|---|
| `create_epic` | `slug`, `title`, `summary`, `created_by` | |
| `create_feature` | `slug`, `epic` (parent epic ID), `summary`, `created_by` | |
| `create_task` | `feature` (parent feature ID), `slug`, `summary` | |
| `create_bug` | `slug`, `title`, `reported_by`, `observed`, `expected`, `severity`, `priority`, `type` | `severity`/`priority`: `low`/`medium`/`high`/`critical`. `type`: `implementation-defect`/`specification-defect`/`design-problem` |
| `record_decision` | `slug`, `summary`, `rationale`, `decided_by` | Named `record_decision` rather than `create_decision` to reflect the semantic that decisions are recorded, not created |
| `get_entity` | `entity_type`, `id` or `slug` | Returns full entity as JSON |
| `list_entities` | `entity_type` | Returns array of entity summaries |
| `update_status` | `entity_type`, `id` or `slug`, `status` | Validates transition legality per P1-DEC-010 |
| `update_entity` | `entity_type`, `id` or `slug`, plus arbitrary field key/value pairs | Cannot change `id` or `status`; for error correction (§21.2) |
| `validate_candidate` | `entity_type`, plus arbitrary field key/value pairs | Validates without persisting; returns array of validation errors |
| `health_check` | *(none)* | Returns health report across all entities |
| `rebuild_cache` | *(none)* | Rebuilds SQLite cache from canonical YAML; returns entity count |

#### Document tools (8)

The `doc_type` parameter accepts: `proposal`, `research-report`, `draft-design`, `design`, `specification`, `implementation-plan`, `user-documentation`.

| Tool | Required parameters | Optional parameters | Notes |
|---|---|---|---|
| `scaffold_document` | `doc_type`, `title` | — | Returns scaffolded markdown as plain text; does not persist |
| `submit_document` | `doc_type`, `title`, `body`, `created_by` | `feature` (required for design/specification/implementation-plan) | Creates document in `submitted` state |
| `update_document_body` | `doc_type`, `id`, `body` | — | Transitions to `normalised` state |
| `approve_document` | `doc_type`, `id`, `approved_by` | — | Transitions to `approved` state |
| `retrieve_document` | `doc_type`, `id` | — | Returns document body as plain text |
| `extract_from_document` | `doc_id` | — | Returns metadata + body for agent-mediated extraction; approved documents only |
| `list_documents` | *(none)* | `doc_type` | Omit `doc_type` to list all |
| `validate_document` | `doc_type`, `id` | — | Returns array of validation errors |

#### Response conventions

- All tools return `mcp.CallToolResult` payloads.
- Most responses are JSON-serialised structured data (entity records, arrays, health reports).
- `scaffold_document` and `retrieve_document` return plain text.
- Errors are returned as MCP error payloads, not transport-level failures — the caller always receives a result.

### Rationale

Object-specific creation tools (`create_epic`, `create_bug`, etc.) were chosen over a generic `create_entity` because each entity type has different required fields. A single `create_entity` tool would need complex conditional validation that is harder for agents to understand and use correctly. The type-specific tools make the required parameters explicit in the tool schema.

Generic tools were used for retrieval, status updates, and field updates because these operations have the same shape across entity types — only the `entity_type` parameter varies. This avoids duplicating nearly-identical tools.

Document tools are namespaced (`scaffold_document`, `submit_document`, etc.) to distinguish them from entity operations while maintaining the verb-first naming pattern.

`record_decision` was chosen over `create_decision` to reflect that decisions are *recorded* (they happened in conversation and are being captured), not *created* by the tool.

### Alternatives Considered

- **Object-specific naming for all operations** — `create_epic`, `get_epic`, `update_epic_status`, etc. Would produce ~35 tools with heavy duplication. Rejected for redundancy.
- **Generic CRUD naming for all operations** — `create_entity`, `get_entity`, `update_entity`, etc. Would require all tools to accept `entity_type` and conditionally validate fields. Rejected because creation schemas differ significantly per type, making a single tool confusing.
- **Mixed approach** (chosen) — type-specific creation, generic retrieval/update. Balances explicitness with compactness.

### Consequences

- agents must use the correct type-specific tool for creation but can use generic tools for retrieval and updates
- the tool surface is compact (20 tools) while remaining explicit about per-type creation requirements
- adding a new entity type requires adding a new `create_*` tool but no changes to `get_entity`, `list_entities`, `update_status`, or `update_entity`
- document operations are clearly separated from entity operations by naming convention

### Follow-up Needed

- ~~define exact operation set and semantics~~ — **Resolved:** documented above from implementation in `internal/mcp/`

---

## `P1-DEC-012: CLI mapping`

- Status: accepted
- Date: 2026-03-18
- Accepted: 2026-07-24
- Scope: secondary interface
- Related:
  - `work/plan/phase-1-implementation-plan.md`
  - `work/design/workflow-design-basis.md` §6.2
  - P1-DEC-011 (MCP operation names)

### Decision

The CLI follows the **verb-first, then object** pattern established during the proposal phase. The binary is `kanbanzai`; `kbz` is intended as a symlink alias for human use.

All output is **plain text** — key-value lines for single entities, tab-separated columns for lists. No JSON or YAML output from the CLI.

#### Top-level commands

| Command | Description |
|---|---|
| `help`, `-h`, `--help` | Print usage text |
| `version`, `--version` | Print version string |
| `serve` | Start MCP server (stdio transport) |
| `create <entity>` | Create a workflow entity |
| `get <entity>` | Retrieve a single entity |
| `list <collection>` | List all entities of a type |
| `update <subcommand>` | Update entity status or fields |
| `doc <subcommand>` | Document management |
| `health` | Run health check against canonical state |
| `validate` | Validate a candidate entity without persisting |
| `cache <subcommand>` | Manage local derived cache |

#### `create <entity> [flags]`

| Entity | Required flags | Optional flags |
|---|---|---|
| `epic` | `--slug`, `--title`, `--summary`, `--created_by` | — |
| `feature` | `--slug`, `--epic`, `--summary`, `--created_by` | — |
| `task` | `--slug`, `--feature`, `--summary` | — |
| `bug` | `--slug`, `--title`, `--reported_by`, `--observed`, `--expected` | `--severity`, `--priority`, `--type` |
| `decision` | `--slug`, `--summary`, `--rationale`, `--decided_by` | — |

Output: `created <type>\nid: …\nslug: …\npath: …`

#### `get <entity> [flags]`

Entities: `epic`, `feature`, `task`, `bug`, `decision`. Accepts `--id` or `--slug`.

Output: `type: …\nid: …\nslug: …\npath: …\nstatus: …`

#### `list <collection>`

Uses **plural** names: `epics`, `features`, `tasks`, `bugs`, `decisions`.

Output: header line, then one tab-separated row per entity (`id\tslug\tpath\tstatus`).

#### `update status [flags]`

Flags: `--type` (or `--entity`), `--id` or `--slug`, `--status`.

Output: `updated <type> status\nid: …\nslug: …\nold_status: …\nnew_status: …`

#### `update fields [flags]`

Flags: `--type`, `--id`, `--slug`, plus arbitrary `--<field>` flags for field updates. Cannot change `id` or `status`.

#### `doc <subcommand> [flags]`

| Subcommand | Required flags | Optional flags | Notes |
|---|---|---|---|
| `scaffold` | `--type` | `--title` | Outputs scaffolded content to stdout |
| `submit` | `--type`, `--title`, `--created_by` | `--feature`, `--body` | If `--body` omitted, reads from stdin |
| `approve` | `--type`, `--id`, `--approved_by` | — | |
| `retrieve` | `--type`, `--id` | — | Outputs raw document body |
| `validate` | `--type`, `--id` | — | |
| `list` | — | `--type` | Omit `--type` to list all |

#### `health`

No arguments. Runs integrity check on canonical state.

Output: `health check\nentities: <n>\nerrors: <n>\nwarnings: <n>`, followed by error/warning lines.

#### `validate [flags]`

Flags: `--type` (required), plus arbitrary `--<field>` flags as candidate data.

Output: `candidate is valid` or `validation errors: <n>` followed by error lines.

#### `cache rebuild`

Rebuilds local derived cache from canonical YAML files.

Output: `cache rebuilt: <n> entities cached\npath: …`

### Rationale

The verb-first pattern (`kbz create epic`, `kbz list tasks`, `kbz update status`) reads like natural English and is consistent with common CLI conventions (e.g., `git`, `kubectl`). The one exception is `doc`, which groups all document operations under a namespace (`kbz doc submit`, `kbz doc approve`) rather than using `kbz submit doc`. This was a pragmatic choice — document operations are a distinct subsystem with their own lifecycle, and grouping them under `doc` makes discoverability easier than scattering them across top-level verbs.

Plain text output was chosen over JSON/YAML because the CLI is a human interface. Structured output is available through the MCP server for agents. Tab-separated columns for lists enable easy piping to standard Unix tools (`cut`, `sort`, `grep`).

### Alternatives Considered

- **Mirror MCP closely** — expose the same tool names and JSON output. Rejected because the CLI is a human interface; JSON output and tool-style naming (`create_epic`) are less natural at a terminal.
- **Human-oriented CLI wrapper with rich formatting** — colour output, tables, interactive prompts. Rejected as over-engineering for Phase 1. Plain text is sufficient and avoids terminal compatibility issues.
- **Minimal CLI** — only `serve` and a few debugging commands. Rejected because humans need direct access to entity operations for bootstrapping, debugging, and CI.

### Consequences

- the CLI is thin: it parses flags, calls the shared service layer, and formats output as plain text
- adding new entity types requires adding `create` subcommands but generic commands (`get`, `list`, `update`) work without changes
- the `doc` namespace is the one departure from pure verb-noun; this is acceptable and may be revisited if other subsystems need similar grouping
- the `kbz` symlink is not yet created; only the `kanbanzai` binary exists

### Follow-up Needed

- ~~define command surface before implementation~~ — **Resolved:** documented above from implementation in `cmd/kanbanzai/main.go`
- ~~consult on commands that don't naturally fit verb-noun English phrasing~~ — **Resolved:** only `doc` departs from verb-noun; the namespace approach was chosen pragmatically and works well

---

## `P1-DEC-013: Local cache scope and rebuild behavior`

- Status: accepted
- Date: 2026-03-19
- Scope: cache/query support
- Related:
  - `work/spec/phase-1-specification.md` §14.4
  - `work/plan/phase-1-implementation-plan.md` §6.6, §7.7
  - `work/plan/phase-1-audit-remediation.md` Track R6

### Decision

Phase 1 implements a local derived SQLite cache for entity query acceleration. The cache is stored at `.kbz/cache/kbz.db`, is rebuildable from canonical YAML state, is not canonical, and must not be committed to Git.

**Schema.** A single `entities` table with columns: `entity_type`, `id`, `slug`, `status`, `title`, `summary`, `parent_ref`, `file_path`, `fields_json`. Primary key is `(entity_type, id)`. Indexes on `entity_type`, `id`, `(entity_type, status)`, and `(entity_type, parent_ref)`.

**Scope.** The cache supports:
- entity lookup by ID without requiring slug (`LookupByID`, `FindByID`)
- entity existence checks (`EntityExists`)
- list-by-type with cached field data (`ListByType`, `ListAll`)
- full fields retrieval from cached JSON (`GetFields`)
- count queries by type or total (`Count`)

**Rebuild.** The `Rebuild` operation clears the cache and repopulates it in a single transaction from all canonical entity files. It is exposed as:
- `rebuild_cache` MCP tool
- `kbz cache rebuild` CLI command
- `EntityService.RebuildCache()` in the service layer

**Integration.** The cache is optional. `EntityService.SetCache(c)` attaches it. When attached, create/update/status-change operations update the cache best-effort (failures are silently ignored). When the cache is nil, all operations fall back to filesystem reads with no behavior change.

**SQLite library.** `modernc.org/sqlite` (pure Go, no CGO) as specified in P1-DEC-016.

### Rationale

The spec (§14.4) requires a local derived cache for query performance. The implementation plan (§6.6) positions it as derived, disposable, and rebuildable. SQLite provides indexed queries without building a custom query engine. The single-table design is simple, covers all Phase 1 query patterns, and avoids premature normalisation.

Storing the full fields as JSON in the cache enables the cache to serve as a complete read-through layer without requiring filesystem reads for most operations. This is acceptable because the cache is derived and rebuild is cheap (scanning YAML files is O(n) where n is small for Phase 1 projects).

Best-effort cache updates on mutations keep the cache reasonably fresh without making it a correctness dependency. If the cache drifts, `rebuild_cache` restores consistency.

### Alternatives Considered

- **No cache (filesystem only)** — satisfies correctness but violates the spec requirement (§14.4) and leaves entity-by-ID lookup requiring a directory scan. Rejected.
- **In-memory cache (map-based)** — simpler but lost on process restart. Every MCP server launch would require a full scan. SQLite persists across invocations and supports indexed queries.
- **Richer cache with multiple tables** — normalised tables for references, cross-entity indexes, full-text search. Premature for Phase 1; the single-table design with JSON fields covers all current query patterns. Can be extended in Phase 2.
- **Cache as mandatory dependency** — rejected. The spec requires the cache to be derived and non-canonical. All operations must work without it.

### Consequences

- `.kbz/cache/kbz.db` is created on first MCP server start or `kbz cache rebuild`
- the cache directory is covered by the existing `.kbz/` gitignore rule
- `modernc.org/sqlite` is added as a dependency (pure Go, no CGO)
- entity lookup by ID without slug is now possible via the cache
- cache staleness is bounded by best-effort updates on mutations
- `rebuild_cache` is the recovery mechanism for any cache inconsistency
- the cache does not store documents (document queries remain filesystem-based in Phase 1)

### Follow-up Needed

- consider adding document metadata to the cache if document query performance becomes a concern
- evaluate whether Phase 2's richer query needs (attribute filtering, text search) require schema changes or a separate cache layer
- monitor whether best-effort cache updates are sufficient or whether stronger consistency guarantees are needed

---

## `P1-DEC-014: Template structures for required documents`

- Status: accepted
- Date: 2026-03-18
- Scope: document support
- Related:
  - `work/spec/phase-1-specification.md`
  - `work/design/workflow-design-basis.md`
  - `work/design/product-instance-boundary.md`

### Decision

Phase 1 will use typed markdown documents with stable templates, explicit section structure, and validation by document class.

The Phase 1 document strategy is:

- markdown remains the format for human-authored specifications and plans
- markdown is not treated as structurally free-form for canonical workflow documents
- each canonical markdown document type must have:
  - a defined purpose
  - a stable template
  - required sections
  - a required section order
  - naming rules
  - validation rules
- generated operational views remain generated projections
- intake markdown remains intake and must not be mistaken for canonical structured documents

For Phase 1, the required canonical markdown document classes are:

- feature specification documents
- feature plan documents
- bug-report-related markdown only if needed by the chosen phase-1 implementation path

The strategy for markdown fragility is:

- humans own meaning
- the workflow tool owns structural normalization and validation
- generated markdown projections are tool-owned
- canonical human-authored markdown is tool-normalized and validated
- rough markdown input is treated as intake until normalized and committed

This means the workflow tool should be the canonical writer of:

- generated projections
- scaffolded markdown templates
- normalized structural form for canonical markdown documents

It does not mean the tool owns the meaning of human-authored prose.

### Rationale

Markdown is valuable because it is readable and writable by both humans and AI agents, but it is structurally fragile if left unconstrained. The previous workflow failed partly because markdown was too permissive and too easy to let drift.

A strict document strategy preserves the benefits of markdown while reducing its risks:

- templates reduce structural drift
- validation catches missing required sections and broken links
- stable section ordering improves diff quality
- keeping long prose in markdown avoids forcing narrative into YAML
- distinguishing intake markdown from canonical markdown prevents raw notes from silently becoming authoritative
- letting the tool normalize structure keeps documents stable without requiring humans to memorize rigid formatting rules

This approach fits the broader design principle of a strict core with a forgiving interface.

### Alternatives Considered

- minimal templates
- richer structured templates
- staged template strictness

A purely minimal template approach was rejected because it would not do enough to reduce markdown drift.

A very rich or heavily generated document model was rejected for Phase 1 because it would add too much complexity too early.

The chosen approach is staged template strictness: typed markdown with required structure and validation in Phase 1, with room for richer partial rendering or stronger productization later.

### Consequences

- Phase 1 document scaffolding must be type-aware
- Phase 1 document validation must check required sections, naming, and basic referential integrity
- markdown remains important in the workflow, but only certain markdown classes are canonical
- generated views must remain clearly separate from canonical human-authored documents
- the implementation must distinguish:
  - intake markdown
  - canonical typed markdown
  - generated projections
- future promotion of stable templates into reusable product assets is supported cleanly

### Follow-up Needed

- define the exact required templates for Phase 1
- define the exact required sections and section order for each canonical markdown document type
- define naming and frontmatter requirements for each canonical markdown document type
- ensure the implementation plan treats the tool as the canonical structural writer/normalizer for canonical markdown documents

---

## `P1-DEC-015: Bootstrap introduction strategy`

- Status: accepted
- Date: 2026-03-18
- Accepted: 2026-07-24
- Scope: self-use bootstrap
- Related:
  - `work/design/product-instance-boundary.md`
  - `work/spec/phase-1-specification.md` §22
  - `work/plan/phase-1-implementation-plan.md` §12
  - P1-DEC-005 (Phase 1 must support limited bootstrap self-use)

### Decision

Bootstrap self-hosting begins now. The kernel is substantially complete (all 5 entity types, lifecycle enforcement, CRUD, documents, health checks, CLI, and MCP are implemented and tested). The remaining work (ID migration, decision-log housekeeping) does not block initial self-use.

#### What gets committed

The `.kbz/` directory is committed to the repository. The project's workflow state is version-controlled alongside its code. The workflow state is at least as important as the code — theoretically the code can be re-created from the workflow, but not the reverse.

#### Initial records

- **One Epic** representing Phase 1 itself.
- **Features** for the remaining Phase 1 work (ID migration, bootstrap, decision-log cleanup, etc.), with links to related documents.
- **Decisions** — the existing 21 decisions in `work/plan/phase-1-decision-log.md` may be migrated into the tool if practical, but the decision log document remains the authoritative record. New decisions going forward should be recorded through the tool.
- **Tasks** are a planning/implementation detail — they can be preplanned or planned just-in-time. Humans are not required to be involved in task-level planning. No backfilling of historical tasks is needed.
- **Bugs** — no backfilling of historical bugs. New bugs are recorded through the tool going forward.

#### Document backfill

All existing documents in `work/` (designs, specifications, research, plans) must be preserved and should be registered in the tool as documents. These are the project's primary intellectual record. Entity records are less critical — they are an index into the documents and the work, not the work itself.

#### Relationship to bootstrap workflow

The current bootstrap workflow (`work/bootstrap/bootstrap-workflow.md`) continues to operate in tandem with the tool during the transition period. The two systems of record overlap temporarily. The bootstrap workflow retires when the tool is demonstrably sufficient for day-to-day use — this is a gradual transition, not a hard cutover.

#### Backfill scope

| Material | Backfill? | Rationale |
|---|---|---|
| Documents (designs, specs, research, plans) | Yes — all | Primary intellectual record; must not be lost |
| Epics | Yes | Top-level work structure |
| Features | Yes, with document links | Genuinely useful for tracing design-to-implementation; links between epics, features, and documents are valuable |
| Tasks | No | Historical record of tasks completed is less important; effort outweighs value |
| Bugs | No | Same reasoning as tasks |
| Decisions | Best-effort | Migrate if practical; the decision log document remains authoritative |

### Rationale

The kernel is feature-complete enough for limited self-use. Delaying further would mean continuing to operate without the process the tool is designed to enforce, while the tool sits idle and untested on real work.

Committing `.kbz/` to the repo is the natural choice for a Git-native workflow system — the workflow state *is* the project's process record and should be versioned, reviewable, and recoverable just like code.

The selective backfill strategy (all documents, epics and features with links, skip tasks and bugs) reflects the relative value of each record type. Documents and their structural relationships are irreplaceable. Task-level history is ephemeral — knowing *what was designed* matters more than knowing *which agent worked on which task*.

### Alternatives Considered

- **Start self-use only after kernel validation (including ID migration).** Would delay bootstrapping until TSID13 is implemented, which means initial records would never need migration. Rejected because the delay is unnecessary — sequential IDs work fine for bootstrap use, and migrating a small set of initial records is trivial.
- **Create a minimal first epic/feature set early (before kernel is proven).** Would have started self-use before the tool was trustworthy. Rejected because premature self-use risks corrupted state and undermines confidence in the tool.
- **Defer self-use until after core CRUD + validation + docs support exist.** This was the original deferral condition. It is now met — all of these capabilities exist and are tested.

### Consequences

- the `.kbz/` directory appears in the repository and is committed like any other project artifact
- the bootstrap workflow and the tool coexist temporarily; the team must be clear about which system of record to consult during the transition
- document backfill is a meaningful task — all documents in `work/` need to be registered through the tool's document submission flow
- the product/instance boundary must be maintained carefully: `.kbz/` contains this project's instance state, not reusable product configuration
- initial records use sequential IDs (the current allocator); these will be migrated when TSID13 is implemented

### Follow-up Needed

- ~~choose explicit bootstrap milestone and first records~~ — **Resolved:** documented above. Bootstrap begins now with the scope described.
- create the initial Epic, Features, and document registrations
- determine the explicit retirement criteria for the bootstrap workflow

---

## `P1-DEC-016: Implementation language`

- Status: accepted
- Date: 2026-03-18
- Scope: implementation
- Related:
  - `work/design/workflow-system-design.md` §4.2
  - `work/plan/phase-1-implementation-plan.md`

### Decision

Phase 1 will be implemented in Go.

The `kanbanzai` binary will be a single statically-compiled Go binary serving both the MCP server (`kanbanzai serve`, stdio transport) and the CLI (`kbz <command>`). Both modes share the same core logic.

Dependencies will use pure-Go libraries where possible to avoid CGO and C compiler requirements:

- YAML: `gopkg.in/yaml.v3` (round-trip capable, formatting control)
- SQLite: `modernc.org/sqlite` (pure Go, no CGO)
- Git: shell out to `git` via `os/exec`
- MCP transport: JSON-RPC over stdio (standard library `encoding/json`, `bufio`)

### Rationale

The project needs a language that produces a single distributable binary with fast startup, strong standard library support for CLI tools, JSON, YAML, and file I/O, and straightforward concurrency. Go fits all of these.

**Single binary distribution.** The tool is launched by MCP clients and used by AI agents. The install path must be trivial — one binary on `$PATH`, no runtime, no dependency manager, no `node_modules`. `go build` produces this.

**Fast startup.** The CLI needs to feel instant for commands like `kbz status` and `kbz get`. Go binaries start in milliseconds.

**Stdio and JSON ergonomics.** The MCP protocol is JSON-RPC over stdio. Go's standard library (`encoding/json`, `bufio`, `os`) makes this straightforward. The transport layer is a few hundred lines, not a framework dependency.

**YAML round-tripping.** The tool is the canonical YAML writer (P1-DEC-008). `gopkg.in/yaml.v3` provides AST-level control over formatting, field ordering, and comment preservation — necessary for deterministic output and diff-stable canonical records.

**Pure-Go SQLite.** `modernc.org/sqlite` eliminates the CGO dependency, keeping the build simple and cross-compilation trivial. The cache is derived state rebuilt from YAML, so it doesn't need extreme SQLite performance.

**Git integration.** Shelling out to `git` via `os/exec` is the pragmatic approach. The design requires worktree management, branch operations, and status checking — all well-served by the `git` CLI. In-process Git libraries exist (`go-git`) but add complexity without clear benefit for Phase 1.

**Development velocity.** The project needs to bootstrap on itself quickly (P1-DEC-005). Go's compilation speed, simple toolchain, and straightforward error handling support fast iteration without fighting the language.

### Alternatives Considered

- Go
- Rust
- TypeScript / Node.js
- Python

**Rust** — would also produce a single binary with excellent performance, but development velocity would be meaningfully slower. Rust's type system creates friction with YAML round-tripping, dynamic schema validation, and the stringly-typed workflow state this project handles. Compile times are longer. The learning curve is steeper for contributors. Rust is the right choice when memory safety guarantees or extreme performance matter; this project needs neither.

**TypeScript / Node.js** — would give faster iteration on the MCP server (the MCP SDK is TypeScript-native) but creates a runtime dependency. Every developer and agent needs Node installed and `node_modules` managed. The MCP launch command becomes `node /path/to/dist/index.js` instead of `kbz`. The project explicitly avoids unnecessary complexity.

**Python** — similar runtime dependency problem, worse performance for CLI startup, and Python's type system doesn't help with schema validation and state machine enforcement. Viable for prototyping but not for a tool that needs to feel solid and start fast.

### Consequences

- the build toolchain is `go build` with no CGO
- cross-compilation is trivial (GOOS/GOARCH)
- the MCP transport layer is implemented directly against the JSON-RPC spec, not via an external MCP SDK
- the `.gitignore` already covers Go build artifacts
- contributors need Go 1.22+ installed (current stable)
- the MCP ecosystem is TypeScript-first; Go MCP libraries are less mature, but the protocol is simple enough that this is not a significant risk
- pure-Go SQLite (`modernc.org/sqlite`) may be slightly slower than CGO-based alternatives, but performance is not a concern for a derived cache

### Follow-up Needed

- confirm minimum Go version requirement
- evaluate whether any Go MCP library is worth adopting vs. implementing the thin JSON-RPC layer directly
- set up the initial Go module structure (`go mod init`)

---

## `P1-DEC-017: Link resolution support deferred to Phase 2`

- Status: accepted
- Date: 2026-03-19
- Scope: spec-compliance
- Related:
  - `work/spec/phase-1-specification.md` §17.2
  - `work/plan/phase-1-audit-remediation.md` Track R5, item S4

### Decision

Link resolution from loose references is deferred to Phase 2.

Phase 1 validates explicit references (e.g. a Feature's `epic` field must reference an existing Epic) but does not infer likely links from free text in titles, descriptions, or document content.

### Rationale

The spec (§17.2) uses "should" language and explicitly permits deferral: "If not implemented in Phase 1, this must be explicitly deferred."

Link resolution requires either text-matching heuristics or semantic search — both add complexity disproportionate to Phase 1's scope. Phase 1 focuses on the workflow kernel: entities, lifecycle, documents, and deterministic storage. Explicit references already provide referential integrity where it matters (Feature→Epic, Task→Feature).

### Alternatives Considered

- **Simple substring matching** — fragile, high false-positive rate, and would need to be replaced by something better in Phase 2 anyway. Not worth the implementation and maintenance cost.
- **Defer entirely** — chosen. Phase 2's context and knowledge management features (machine-context-design.md) are the natural home for link resolution.

### Consequences

- agents must create explicit references when creating entities; the system will not infer them
- free-text references in documents remain unresolved until Phase 2
- no additional indexing or search infrastructure is needed in Phase 1

### Follow-up Needed

- Phase 2 design should include link resolution as part of the context/knowledge layer
- evaluate whether the local cache (P1-DEC-013) could support simple link resolution as an incremental step

---

## `P1-DEC-018: Duplicate detection deferred to Phase 2`

- Status: accepted
- Date: 2026-03-19
- Scope: spec-compliance
- Related:
  - `work/spec/phase-1-specification.md` §17.3
  - `work/plan/phase-1-audit-remediation.md` Track R5, item S5

### Decision

Duplicate detection for bug and feature creation is deferred to Phase 2.

Phase 1 relies on health checks (which detect referential integrity issues) and human review to identify duplicates. Agents can use `list_entities` to manually check for existing similar entities before creating new ones.

### Rationale

The spec (§17.3) uses "should" language and explicitly permits deferral: "If not implemented in Phase 1, this must be explicitly deferred."

Meaningful duplicate detection requires either fuzzy text matching or semantic similarity — both are Phase 2 concerns that depend on the context and knowledge management layer. A naive exact-title-match would catch almost nothing useful and give false confidence.

### Alternatives Considered

- **Exact title matching** — too narrow to be useful. Duplicates rarely have identical titles.
- **Fuzzy matching (edit distance, n-grams)** — adds a dependency and tuning burden without the semantic understanding needed to do this well. Better deferred to Phase 2 where embeddings or structured search are available.
- **Defer entirely** — chosen. Agents can perform manual checks using existing list/get operations.

### Consequences

- duplicate entities may be created if agents do not check before creating
- health checks do not currently detect duplicates (this is consistent with the deferral)
- the `list_entities` operation provides a manual workaround

### Follow-up Needed

- Phase 2 should include duplicate detection as part of entity creation validation
- evaluate whether the local cache (P1-DEC-013) could support duplicate detection queries

---

## `P1-DEC-019: Document-to-entity extraction is agent-driven, not automated`

- Status: accepted
- Date: 2026-03-19
- Scope: spec-compliance
- Related:
  - `work/spec/phase-1-specification.md` §15.5
  - `work/plan/phase-1-audit-remediation.md` Track R5, item S6

### Decision

Automated extraction of structured entity data from approved documents is deferred. Phase 1 provides the document lifecycle (scaffold, submit, approve, retrieve) and entity creation tools; extraction is performed by agents using these tools manually.

When a document is approved, an agent reads the document content via `get_document`, identifies decisions, entity updates, and cross-references, and then records them using the existing entity and decision tools. The system does not perform this extraction automatically.

### Rationale

The spec (§15.5) requires that the system "must support extraction of structured entity data from documents" but qualifies that "this extraction is internal — the human does not need to see or manage it." The requirement is satisfied by agents performing extraction using existing tools — the spec does not require a dedicated extraction mechanism.

Building automated extraction tooling would require NLP or structured parsing of free-form document content — complexity that belongs in Phase 2's agent-assistance layer, not in the workflow kernel.

### Alternatives Considered

- **Template-based extraction** — parse documents for structured sections (e.g. "## Decisions") and extract entities automatically. Fragile, tightly coupled to document templates, and would need constant maintenance as templates evolve.
- **Agent-driven extraction using existing tools** — chosen. Agents already have the tools to read documents and create entities. This satisfies the spec requirement without new infrastructure.

### Consequences

- document approval does not trigger automatic entity creation or updates
- agents are responsible for extracting and recording decisions, entity updates, and references after document approval
- no additional document parsing infrastructure is needed in Phase 1

### Follow-up Needed

- Phase 2 may add dedicated extraction tooling or agent prompts to streamline this workflow
- monitor whether manual extraction by agents is reliable enough or creates a bottleneck

---

## `P1-DEC-020: Phase 1 entity query is list-by-type only`

- Status: accepted
- Date: 2026-03-19
- Scope: spec-compliance
- Related:
  - `work/spec/phase-1-specification.md` §16.2
  - `work/plan/phase-1-audit-remediation.md` Track R5, item S3
  - `P1-DEC-013` (local cache scope)

### Decision

Phase 1 entity query is limited to list-by-type via the `list_entities` operation. Attribute filtering, text search, and cross-entity queries are deferred to the local cache track (P1-DEC-013).

The `list_entities` operation returns all entities of a given type. Agents perform client-side filtering (by status, priority, assignee, etc.) on the returned results. This satisfies the spec's "search/query objects" requirement at minimum scope.

### Rationale

The spec (§16.2) requires "search/query objects" as a Phase 1 operation but does not define the query language, filter syntax, or minimum filter capabilities. List-by-type with client-side filtering is the simplest implementation that satisfies the requirement.

Server-side filtering requires either scanning all YAML files with in-memory filtering (workable but adds complexity) or the local derived cache (P1-DEC-013, currently proposed). Rather than build throwaway filtering logic, deferring rich query to the cache track keeps Phase 1 focused on the kernel.

### Alternatives Considered

- **In-memory filtering on list** — load all entities of a type, filter by requested fields, return matches. Viable for small projects but adds code that would be replaced by cache queries. Deferred rather than built twice.
- **SQLite cache with query support** — the intended long-term solution (P1-DEC-013). Deferred to Track R6 / Track G.
- **List-by-type only** — chosen. Minimal, sufficient, and does not preclude richer query later.

### Consequences

- agents receive all entities of a type and must filter client-side
- performance is acceptable for small-to-medium projects (dozens to low hundreds of entities per type)
- the `list_entities` response may be large for projects with many entities; this is acceptable for Phase 1
- richer query capability depends on resolving P1-DEC-013 and implementing the local cache

### Follow-up Needed

- resolve P1-DEC-013 (local cache scope) to enable server-side filtering and search
- if cache is deferred beyond Phase 1, consider adding simple in-memory filters to `list_entities` as a stopgap

---

## 8. Acceptance Criteria

This decision log is acceptable as a Phase 1 planning artifact if:

1. the key early implementation decisions are explicitly listed
2. accepted decisions are clearly separated from open decisions
3. unresolved decisions are framed in a way that supports planning
4. future decisions can be added without changing the structure
5. the implementation plan can reference this log instead of relying on implicit assumptions

---

## `P1-DEC-021: Compact time-sorted IDs replace sequential IDs`

- Status: accepted
- Date: 2026-03-20
- Scope: identity
- Supersedes: P1-DEC-007
- Related:
  - `work/design/id-system-design.md`
  - `work/spec/phase-1-specification.md` §13
  - `work/design/workflow-design-basis.md` §11

### Decision

The Phase 1 sequential ID strategy (P1-DEC-007) is retired. All entity and document IDs now use one of two formats:

**Epics — human-chosen slugs:**

- Format: `EPIC-{SLUG}`
- Examples: `EPIC-IDS`, `EPIC-CONCURRENCY`, `EPIC-CONTEXT-ASSEMBLY`
- Slug: uppercase alphanumeric and hyphens, 2–20 characters, URL-safe
- Uniqueness enforced at creation time; health check detects duplicates after merge

**All other entities and documents — compact time-sorted IDs (TSID13):**

- Format: `{TYPE}-{TSID13}` where TYPE is `FEAT`, `BUG`, `DEC`, `TASK`, or `DOC`
- Structure: 10 characters of Crockford base32 timestamp (48-bit millisecond) + 3 characters of Crockford base32 random (15-bit)
- Total: 13 characters after the type prefix
- Examples: `FEAT-01J3K7MXP3RT5`, `BUG-01J4AR7WHN4F2`, `TASK-01J3KZZZBB4KF`
- Time-sortable, URL-safe, case-insensitive matching
- Local collision check with retry on generation

**Tasks are independent entities** with their own TSID13 IDs. The Phase 1 hierarchical task ID format (`FEAT-001.3`) is retired. The parent feature relationship is stored as a metadata field (`parent_feature`), not embedded in the ID.

**Display conventions:**

- A break hyphen is inserted after the fifth TSID character for display: `FEAT-01J3K-7MXP3RT5`
- The break hyphen is display-only — not stored in YAML, filenames, or references
- The CLI and MCP operations accept any unique prefix: `FEAT-01J3K` resolves to the full match
- Listings show the shortest unique prefix: `FEAT-01J3K-7MX`

**Provenance is metadata, not identity.** Creator information is stored in entity fields (`created_by`) and git history, not embedded in IDs.

### Rationale

The Phase 1 sequential strategy (scan-max-increment) fails in every concurrent scenario:

- Same worktree, two agents: both scan the same max, both allocate the same ID
- Separate worktrees: both independently allocate the same sequential ID, guaranteed filename collision at merge
- Unmerged branches: IDs are not claimed until committed and merged

P1-DEC-007 explicitly acknowledged concurrency testing as open follow-up and required the strategy to remain replaceable. With the project entering parallel multi-agent work, the replacement is now due.

The TSID13 format provides collision safety without coordination. The 48-bit millisecond timestamp separates any IDs created more than 1ms apart. The 15-bit random component (32,768 values per millisecond) handles the near-impossible case of same-millisecond creation. A local collision check provides a safety net.

Full ULIDs (128-bit, 26 characters) were considered but rejected as massive overkill for a small workflow database. The system creates a handful of entities per day, not millions per second. 15 bits of randomness per millisecond is more than sufficient, and the shorter IDs (13 vs 26 characters) significantly improve readability in filenames, YAML references, and commit messages.

One format for all generated entities (features, bugs, decisions, tasks, documents) was chosen over a graduated entropy scheme for consistency — one generator, one parser, one set of tests.

Epics use human-chosen slugs because they are few (5–30 per project), always human-created, and benefit from maximum memorability.

### Alternatives Considered

- **Full ULIDs (128-bit, 26 chars):** Collision-safe but far more entropy than needed. Produces unwieldy IDs for a small database.
- **Truncated ULIDs (first N chars):** The ULID monotonic increment operates on least-significant bits, so truncation loses the monotonicity guarantee. Custom generation is cleaner.
- **Block allocation with person identifiers:** Partitions the namespace by creator identity. Rejected because it conflates identity with ownership, doesn't solve within-partition collisions, and is unnecessary when time-sorted random IDs eliminate the coordination problem entirely.
- **Hierarchical task IDs (`FEAT-xxx.Txxx`):** Preserves visual grouping but embeds the parent relationship in the ID, making tasks dependent on their parent feature. Rejected in favour of independent task IDs with parent as metadata.
- **Different ID lengths per entity type (entropy gradient):** More entropy for tasks, less for features. Rejected for consistency — 15 bits per millisecond is adequate for all entity types, and one format is simpler.

### Consequences

- The `internal/id` package is replaced with a TSID13 allocator and an epic slug allocator
- The `internal/storage` filename parser handles 13-character TSID format (legacy sequential format retained for robustness)
- The `internal/document` in-memory counter closure is replaced with TSID13 generation
- The `internal/validate` ID validation rules are updated for both formats
- Prefix resolution is added to the service layer for get/update operations
- All entity creation and document operations use the new allocators
- The local collision check adds a verify-and-retry step to ID generation
- Task entities gain a `parent_feature` metadata field to replace the hierarchical ID relationship
- Existing Phase 1 test fixtures using sequential IDs (`FEAT-001`, etc.) must be updated

### Follow-up Needed

- implement the Crockford base32 encoder for TSID13 generation
- implement prefix resolution in the service and MCP layers
- update the display layer (CLI, MCP responses) to insert break hyphens and compute shortest unique prefixes
- update P1-DEC-009 (minimum required fields) to add `parent_feature` as a required field for tasks
- update P1-DEC-006 (file layout) to reflect new filename format with 13-char TSID
- design the concurrency model for the remaining parallel-work concerns (lost updates, state machine violations) that IDs alone do not solve

---

## 9. Summary

This document records the key implementation decisions for Phase 1.

Its purpose is to make early architecture and implementation choices explicit, reviewable, and traceable.

It should be updated as decisions are made, so that:

- the implementation plan stays grounded
- the specification is interpreted consistently
- implicit assumptions are reduced
- later review can understand why important choices were made

This log is part of the discipline required to build the workflow kernel correctly.