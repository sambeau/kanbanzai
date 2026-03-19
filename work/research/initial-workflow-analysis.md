# Initial Workflow Analysis

- Status: discussion draft
- Purpose: capture initial analysis and recommendations for designing a human-and-AI workflow system for large software projects
- Date: 2026-03-18

## Summary

The current workflow has strong foundations: explicit design thinking, staged documents, feature IDs, review against specification, and separation between human-facing design/specification and AI-facing implementation planning. The main issue is not that the process is too documented, but that the documentation is too free-form, too file-oriented, and not enforced strongly enough by tooling.

The recommended direction is not to replace everything with a central database, and not to hand the whole project lifecycle over to highly autonomous prompt-driven agent systems. Instead, the recommendation is to design a hybrid workflow system with:

- structured canonical records
- markdown as a human-facing view
- deterministic local tools for validation, rendering, orchestration, and retrieval
- explicit delegation boundaries for human and AI roles
- source-control rules designed for AI concurrency
- a local indexed cache for fast querying and memory
- an MCP-first workflow kernel controlled by AI agents
- a strict internal workflow core with AI-mediated normalization of human input

In short:

> Humans should be able to speak naturally and imperfectly through chat and documents.  
> Agents should interpret, clarify, normalize, and translate that input into valid structured workflow state.  
> The workflow kernel should be controlled through a strict MCP interface.  
> Git remains central.  
> Markdown is a view and an intake medium, not the sole source of truth.

---

## Core assessment

### What is working well already

The current document-heavy process has several strengths:

- Markdown works well for human-readable documents
- Distinct document stages have value:
  - brainstorm / research
  - design
  - specification
  - implementation plan
  - manual / support docs
- Feature IDs are useful
- Iterative review against specifications works
- Human documentation can be reviewed against design and specification
- Allowing AI agents to manage implementation plans without polluting specifications with implementation detail is a good separation

These are not things to discard. They are the foundations to preserve.

### What is missing

The current system lacks strong support for:

- high-level planning
- roadmap management
- backlog hygiene
- status accuracy
- formal approval workflows
- explicit parent/child task structure
- robust iteration management
- strong linking between documents and code
- persistent shared memory for teams of agents
- consistent cross-session retrieval for both humans and agents

### What is going wrong

The recurring failures are mostly consistency failures:

- naming conventions drift
- document organization drifts
- documentation falls out of sync with code and with itself
- markdown structure is too flexible to reliably enforce
- status fields become stale
- later documents supersede earlier ones informally rather than structurally
- branches drift and are forgotten
- planning docs are not consistently updated
- backlogs become dumping grounds
- numeric ID-only naming hurts human findability
- simply finding the right next document or next ID consumes time and tokens

### Root cause

The core problem is not “too much documentation”.

The real problem is:

> the workflow is document-oriented, but does not yet have a strong canonical data model.

Markdown is fine as a presentation and discussion format. It is poor as the sole system of record for a large, distributed, multi-agent workflow.

---

## Evaluation of existing external approaches

## Get Shit Done v1

### Strong ideas

This system gets several important things right:

- context rot is real
- decomposition is necessary
- fresh contexts improve quality
- planning artifacts help
- verification matters
- dependency-aware execution waves are valuable
- separation of vision, requirements, roadmap, plans, and summaries is good practice

### Concerns

For this workflow, it appears to lean too far toward agent autonomy and workflow encoded in prompts. It seems optimized for throughput and long autonomous runs, rather than careful governance of planning truth. It also appears comfortable with broad execution permissions, which is not a good fit for a workflow where humans retain higher-level responsibility.

### Verdict

Useful source of ideas, but not a system to adopt directly.

Borrowable ideas:

- discussion before execution
- context-sized tasks
- dependency-wave scheduling
- explicit verification
- persistent state summaries

Not recommended to copy wholesale:

- the command-heavy workflow surface
- the degree of ceremony
- the trust model
- markdown as the entire artifact universe

---

## GSD 2

### Strong ideas

This is a more serious architectural direction than v1. It moves toward deterministic orchestration rather than prompt-only orchestration. That is an important improvement.

Notable strengths:

- fresh session per unit of work
- explicit state machine
- crash recovery
- worktree isolation
- cost tracking
- timeout supervision
- stuck detection
- step mode and auto mode
- disk-backed state

### Concerns

Its center of gravity is still “a robust coding agent runtime”, while the problem here is broader: a project workflow operating system for humans and multiple classes of AI teams. Those are related, but not the same.

### Verdict

A useful source of architectural lessons, especially around deterministic orchestration and state management, but not the best model to adopt directly as the foundation.

Borrowable ideas:

- deterministic orchestration
- disk-backed state
- crash recovery
- worktree-based isolation
- fresh context per unit
- queryable workflow state
- explicit progress and cost visibility

---

## Superpowers

### Strong ideas

This is more aligned with a human-steered planning process:

- brainstorming before coding
- discussion as a first-class step
- explicit design approval
- planning before execution
- subagent use after approval, not before
- more emphasis on human alignment than raw autonomy

### Concerns

It is still heavily skill- and prompt-driven. It appears quite opinionated and may over-elaborate implementation plans into documents that become half-spec, half-code. It still relies on agents consistently following procedural prose.

### Verdict

Closer in spirit to the desired workflow than GSD v1, but still not suitable as the foundation. Better as a source of ideas than as a framework to adopt.

Borrowable ideas:

- collaborative specification refinement
- explicit approval gates
- decomposition before execution
- subagent dispatch after planning agreement

---

## codebase-memory-mcp

### Why it matters

This is the most strategically relevant external idea.

Its importance is not just code graphing, but the underlying principle:

> persistent, queryable, structured memory beats repeated textual rediscovery.

For code, this means functions, modules, call relationships, routes, architecture.

For planning and project workflow, the equivalents are:

- initiatives
- milestones
- roadmap items
- features
- decisions
- designs
- specifications
- plans
- tasks
- approvals
- code links
- release links
- team memory
- expertise memory

### Verdict

This is a very strong conceptual model for the planning system. A planning-oriented memory/query layer or MCP-style interface is likely a high-value direction.

---

## Recommended strategic direction

## Structured core, markdown surfaces

The key recommendation is:

> store workflow truth in structured canonical records, and project markdown as a human-facing view over that truth.

This allows the system to preserve human-readable documents without treating unconstrained markdown as the sole canonical state.

### Practical interpretation

- canonical truth: structured records
- human interface: natural-language chat plus markdown documents and rendered views
- agent interface: MCP tools, context packets, and deterministic workflow operations
- git: collaboration and history layer
- local cache: indexed query layer

Markdown should still exist, but as a governed format rather than an uncontrolled one.

## MCP-first control model

The preferred operating model is not a human-facing CLI with a conversational wrapper. The preferred model is:

- humans interact through chat
- AI agents control the workflow system through MCP
- the workflow kernel remains strict, typed, and deterministic
- any CLI that exists is secondary, useful for debugging, CI, repair, or maintenance

In this model, the primary formal interface is the MCP server. The human does not need to know workflow commands. The AI agent becomes the translation and control layer between informal human intent and formal workflow operations.

This gives a cleaner separation of concerns:

- humans express intent
- agents interpret and normalize
- MCP tools perform strict reads and writes
- canonical state remains validated and machine-checkable

## Strict core, forgiving interface

A key principle of the system should be:

> be liberal in what you accept, and disciplined in how you normalize it

Humans should not be required to memorize commands, rigid schemas, or exact workflow terms in order to use the system effectively. They should be able to communicate in normal language, with the AI responsible for translating rough intent into properly structured workflow objects.

The forgiving interface belongs in the agent layer, not in the workflow kernel itself. The workflow kernel should remain strict. The agent should be responsible for:

- understanding human intent
- reading loosely structured markdown and notes
- asking clarifying questions
- extracting structured meaning
- proposing normalized interpretations
- calling MCP tools with valid formal data

This implies a formal intake pipeline:

### 1. Intake

Input may arrive as:

- chat messages
- markdown notes
- brainstorm documents
- rough bug reports
- review comments
- draft designs
- free-form change requests

These should be accepted as useful source material, but not treated as canonical truth on their own.

### 2. Interpretation

The AI identifies:

- what kind of workflow action is being requested
- what objects are involved
- what facts are explicit
- what is implied
- what remains missing or ambiguous

### 3. Clarification

The AI asks focused follow-up questions where needed.

For example, on a bug report it may need to clarify:

- expected behavior
- observed behavior
- severity
- environment
- reproducibility
- affected scope

### 4. Normalization

The AI prepares candidate structured data such as:

- object type
- fields
- links
- metadata
- state transitions
- supersession relationships
- verification expectations

### 5. Formal commit through MCP

Once the required information is available, the agent should perform formal operations through the MCP interface, using deterministic tools that validate schema, required fields, links, and state transitions.

### 6. Projection

After commit, the system may regenerate or update human-facing markdown views, summaries, handoff packets, and dashboards.

This gives the system the best of both worlds:

- flexibility for humans
- consistency for the workflow
- traceability for the project
- reliable operations for agents

### Important boundary

The system should be liberal in accepting rough input, but not liberal in silently inventing truth.

It should:

- infer when safe
- suggest normalized interpretations
- ask questions when ambiguity matters
- reject invalid or incomplete formal updates

A useful short form for this principle is:

> strict core, forgiving interface

This should apply across the system:

- bug reporting
- feature requests
- approvals
- decisions
- task updates
- backlog capture
- plan corrections

## Intake artifacts, canonical records, and projections

To avoid ambiguity, the system should distinguish among three classes of material:

### Intake artifacts

Human-provided material that has not yet been normalized into canonical workflow state, such as:

- notes
- drafts
- pasted markdown
- brainstorms
- rough bug reports
- review commentary

### Canonical records

Validated structured workflow objects written through formal operations, such as:

- `Bug`
- `Feature`
- `Specification`
- `Decision`
- `Approval`
- `Task`

### Projections

Generated human-facing views derived from canonical state, such as:

- markdown summaries
- review briefs
- handoff packets
- milestone overviews
- dashboards

This distinction helps preserve flexibility for humans without allowing raw input material to drift into unofficial system-of-record status.

---

## Proposed architecture

## 1. Canonical object model

The system should define a small number of first-class workflow objects. Likely candidates:

- Project
- RoadmapItem
- Milestone
- Capability or Feature
- ResearchNote
- Design
- Specification
- ImplementationPlan
- Task
- Bug
- Incident
- RootCauseAnalysis
- Decision
- Approval
- Release
- KnowledgeEntry
- TeamMemoryEntry

Each object should have:

- stable ID
- human-readable slug or label
- type
- status
- parent links
- child links
- revision links where applicable
- related code paths
- related commits / PRs / issues
- owner team
- timestamps
- version or revision
- source references

This object model is the missing backbone.

---

## 2. Explicit lifecycle state machines

Each artifact type should have allowed states.

Examples:

### Specification

- draft
- in_review
- approved
- superseded
- rejected

### Plan

- draft
- ready
- in_progress
- blocked
- done
- superseded

### Task

- queued
- ready
- active
- needs_review
- failed_verification
- done

### Bug

- reported
- triaged
- reproducible
- planned
- in_progress
- needs_review
- needs_verification
- verified
- closed
- duplicate
- not_planned
- cannot_reproduce

### Milestone

- proposed
- approved
- active
- uat
- ready_to_ship
- shipped

This will sharply reduce stale and contradictory status information.

---

## 3. Formal supersession

Later documents must not merely sit beside earlier ones. Supersession needs to be explicit.

Each revisable object should support relationships like:

- supersedes
- superseded_by

The system should be able to answer questions like:

- what is the currently approved specification for this feature?
- what does this document replace?
- which plans were created from an outdated specification?
- which tasks need revalidation because their parent spec changed?

This is a major improvement over naming or date conventions alone.

---

## 4. IDs plus human slugs

Numeric IDs alone are too difficult for humans to scan.

Recommended pattern:

- machine ID: `SPEC-0042`
- human slug: `post-composer-draft-publishing`

Displayed together as:

- `SPEC-0042 post-composer-draft-publishing`

This gives:

- stable references
- grepability
- readable listings
- easier human navigation

---

## 5. Dual-document model

For many artifacts, maintain both:

- a structured canonical record
- a rendered markdown view

Examples:

- `specs/SPEC-0042/spec.yaml`
- `specs/SPEC-0042/spec.md`

or:

- canonical structured files in the repo
- indexed SQLite cache for queries
- generated markdown views for reading and discussion

This preserves markdown while reducing drift.

---

## 6. Local index / cache

A local embedded database is likely useful, but should not be the only source of truth.

Recommended approach:

- Git-managed structured files remain canonical
- a local SQLite database acts as a derived index or query cache
- tooling can rebuild the index from structured files
- agents can query the index rather than rediscovering project state from raw files

This preserves inspectability and diffability while gaining speed and retrieval quality.

---

## Metadata and search

## Standard metadata tags

It is worth allowing metadata beyond the core schema, but not as ad-hoc free-form fields.

Recommended rule:

- metadata tags are allowed
- each tag must be formally defined
- each tag must have:
  - name
  - meaning
  - value type
  - allowed values or format
  - scope
  - examples
  - owner
  - process for introduction
- tags should be registered in a central glossary / schema registry

This avoids uncontrolled proliferation of near-duplicate tags.

### Example metadata categories

Possible standard metadata families:

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

### Text search vs metadata

Text search alone is not enough for a system like this.

Text search is good for:

- finding narrative discussion
- exploratory lookup
- fuzzy discovery

Structured metadata is better for:

- filtering
- queueing
- routing
- dashboards
- validation
- automation
- consistency across agents and humans

Recommended position:

- use both
- text search for discovery
- structured metadata for operation

---

## File formats

## Structured files

A textual, easy-to-diff format is the best fit.

Most likely options:

- YAML
- TOML
- JSON
- Markdown with rigid front matter plus constrained sections

Current recommendation: YAML is a strong candidate if discipline is enforced.

### Requirements for the format

The format should support:

- stable field ordering
- easy human review
- simple machine parsing
- merge-friendly diffs
- strong schema validation

### To keep merges manageable

Use conventions like:

- ordered keys
- sortable IDs
- append-only time logs where possible
- no arbitrary reflow
- deterministic renderer output
- normalized timestamps
- fixed section order
- no duplicated derivable values

The system should format files deterministically so humans and agents do not introduce structural drift.

---

## IDs in a distributed, Git-native process

A distributed process without a central authority needs a robust ID strategy.

## Requirements for IDs

IDs should be:

- unique across contributors and branches
- easy to sort
- short enough for humans
- readable enough for discussion
- stable once created
- merge-safe

## Options

### Option A: simple incrementing IDs

Pros:

- readable
- compact
- familiar

Cons:

- hard in distributed workflows
- collision-prone across branches
- requires allocation logic or conflict resolution

This is probably too fragile on its own.

### Option B: time-based sortable IDs

Pros:

- distributed
- naturally sortable
- low collision risk
- no central allocator needed

Cons:

- slightly less human-friendly
- need care in formatting

This is a strong candidate.

### Option C: hybrid IDs

Recommended direction:

- short type prefix
- sortable time component
- short randomness or node suffix
- human slug alongside the ID

Examples:

- `SPEC-26H3K4-7Q post-composer-draft-publishing`
- `TASK-26H3K8-2M api-create-post`

The exact format can be refined later, but the principle is:

- distributed-safe
- sort-friendly
- readable enough
- used together with human slug

A collision-resistant sortable ID is likely better than pure increments.

---

## Human and AI roles

## Recommended hierarchy

### Humans

Humans should own:

- goals
- priorities
- product direction
- major tradeoffs
- approvals
- ship decisions

### PM / orchestration agents

These agents should manage:

- roadmap decomposition
- specification scaffolding
- dependency tracking
- context packet preparation
- consistency checking
- artifact linking
- state transitions
- backlog hygiene

They should not be the primary code writers.

### Specialist team agents

Examples:

- backend team agents
- frontend / UI team agents
- infrastructure / ops agents
- documentation agents
- QA / verification agents

Each team should have scoped memory and scoped artifacts.

### Execution agents

Short-lived worker agents should:

- implement one task
- verify one task
- report outputs
- update structured workflow state
- stop

This keeps context windows small and responsibilities clear.

---

## Bugs and bugfixing workflow

Bugs are partly implied by the existing planning model, but they should be treated as a first-class workflow stream rather than just ordinary tasks with a different label.

Bug work has concerns that feature work does not always have, including:

- reproduction
- observed versus expected behavior
- environment and version context
- severity and impact
- regression detection
- root cause
- fix verification
- release targeting
- duplicate detection
- links to originating feature, specification, or task

### Bug as a first-class object

A `Bug` record should likely include fields such as:

- `id`
- `slug`
- `status`
- `title`
- `summary`
- `severity`
- `priority`
- `type`
- `found_in`
- `environment`
- `reported_by`
- `affects`
- `origin_feature`
- `origin_spec`
- `origin_task`
- `reproduction`
- `expected_behavior`
- `observed_behavior`
- `suspected_scope`
- `duplicate_of`
- `fixed_by`
- `verified_by`
- `release_target`

### Incident and root cause analysis

Not every bug needs a broader operational process, but some do.

An `Incident` should exist for production-significant failures such as:

- outages
- data corruption
- security issues
- severe degradations

A `RootCauseAnalysis` should exist for important bugs and incidents, especially where the workflow itself can be improved. It should capture:

- what happened
- why it happened
- why it was not caught earlier
- what changed
- what will prevent recurrence

### Bug lifecycle

A bug should have an explicit lifecycle distinct from ordinary feature execution. A typical flow is:

1. report
2. triage
3. reproduce
4. plan
5. fix
6. verify
7. close

The lifecycle states listed earlier should support this flow.

### Standard bugfix path

A typical bugfix workflow should be:

1. **Report**
   - create a `Bug`
   - capture rough report, environment, severity, and impact
2. **Triage**
   - confirm class, priority, duplicate status, and likely scope
3. **Reproduce**
   - establish a reliable reproduction path
   - ideally convert it into a test or repeatable script
4. **Plan**
   - prepare a fix plan with root-cause hypothesis and verification expectations
5. **Fix**
   - execute through normal task and plan machinery
6. **Verify**
   - confirm the repro no longer fails
   - add regression coverage where appropriate
7. **Close**
   - record release target and lessons learned if needed

### Bug versus spec change

The system should help distinguish among three different situations:

- implementation defect: code is wrong relative to the approved spec
- specification defect: the spec itself is wrong or incomplete
- design problem: the design is undesirable even if implementation matches the spec

These should not be conflated.

If the spec is wrong, the system should guide the workflow toward specification supersession rather than simply logging a bugfix against code.

### Bug metadata

Bug work benefits strongly from structured metadata. Useful metadata may include:

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

This is an area where structured metadata is clearly more useful than plain text search alone.

### Bug reporting through conversational intake

Humans should be able to report bugs informally in chat or by supplying rough markdown notes, for example:

- “There’s a bug in signup”
- “The composer ate my draft again”
- “Notifications seem broken on mobile”
- “I think we broke this sometime this week”

The AI should then:

- recognize likely bug-report intent
- interpret both chat input and any supplied markdown or notes
- gather missing required facts through focused questions
- search for duplicates if appropriate
- normalize names and links
- create a consistent structured bug record through MCP
- suggest triage or next steps

This is a strong example of the strict-core, forgiving-interface principle in practice.

## Shared memory model

Memory should not be one giant undifferentiated pile.

Recommended memory classes:

### 1. Canonical project memory

Authoritative facts such as:

- approved decisions
- active specifications
- architecture principles
- current roadmap state

### 2. Team operational memory

Team-scoped guidance such as:

- conventions
- common pitfalls
- preferred patterns
- subsystem-specific norms

### 3. Working memory

Short-lived operational context such as:

- current blockers
- recent changes
- active worktrees
- pending reviews

### 4. Expertise memory

Accumulated heuristics such as:

- known regression hotspots
- common failure modes
- successful decomposition patterns
- lessons learned

These should be represented differently and retrieved differently.

---

## Source control strategy for AI-heavy concurrency

## Recommended principles

### 1. No forgotten long-lived branches

Every branch or worktree should link to:

- one milestone
- one task or task group
- one owner
- one expected lifetime

### 2. Prefer worktrees for isolation

Worktrees are better than repeated branch switching for concurrent AI execution because they reduce cross-task interference.

### 3. Parallelize by conflict domain

Parallel work should be based on:

- file overlap
- dependency ordering
- architectural boundaries
- verification boundaries

not only by human-style team structure.

### 4. Prefer vertical slices where possible

Tasks that represent a coherent end-to-end capability tend to parallelize better than broad horizontal layers.

### 5. Use explicit merge gates

Before integration, require machine-checkable gates such as:

- linked task exists
- linked spec is current
- verification is attached
- branch is not stale
- required state transitions completed
- docs / structured state updated

---

## Relationship to AGENT instructions, skills, and existing agent mechanisms

## Where should AGENT rules / instructions / skills live?

The workflow system should not replace model-specific agent instruction systems. Instead, it should complement them. In particular, agent instructions and skills should help the agent carry out interpretation, clarification, and normalization before calling the MCP workflow tools.

Recommended layering:

### Layer 1: platform-native agent instructions

Examples:

- agent instruction files
- skill definitions
- repository-level coding rules
- model- or tool-specific prompts

These should continue to exist where the runtime expects them.

### Layer 2: workflow system rules

The workflow system should define project workflow rules such as:

- artifact schema
- allowed state transitions
- task handoff structure
- verification requirements
- naming rules
- linking rules
- approval rules

These are not the same as coding-style or runtime-agent instructions.

### Layer 3: generated context packets

The workflow system should generate compact context for a task or handoff, which can then be fed into platform-native skills or agent prompts.

### Layer 4: workflow MCP interface

The workflow system should expose strict typed MCP tools for querying, validating, creating, updating, linking, superseding, and rendering workflow state. This is the main control surface agents should use when committing formal workflow actions.

### Recommended storage

A sensible layout might include:

- global workflow schema and rules
- project-specific workflow policy
- team-specific operating conventions
- generated agent context packets
- references to external skill files where needed

The key point is:

> the workflow system should become the structured source of project-process truth, while agent instruction files remain the runtime-specific method of telling agents how to behave.

## Would this replace standard AGENT files or SKILL files?

Mostly no.

It should not replace them outright.

Instead it should:

- reduce the amount of process logic embedded in prose instructions
- provide stronger structured context to those systems
- make them more consistent
- make them less responsible for inventing project state on the fly
- let them focus on interpretation and normalization while MCP tools handle formal workflow operations

In some cases, it may replace ad-hoc workflow prose currently stuffed into those files. But it should not replace tool-native agent configuration mechanisms that the runtimes expect.

---

## Relationship to codebase-memory-mcp and similar tools

## How would it work with codebase-memory-mcp?

Very well.

These systems solve different but complementary problems.

### codebase-memory-mcp

Optimized for:

- structural code understanding
- architecture retrieval
- call graphs
- code search
- impact tracing
- persistent code memory

### proposed workflow system

Optimized for:

- project planning truth
- roadmap and task structure
- approvals and decisions
- artifact linking
- workflow state
- team memory
- task handoff
- verification tracking

Together they provide:

- code memory
- workflow memory

A future agent should be able to ask both:

- “What calls this function?”
- “What approved specification governs this task?”
- “Which active decision constrains this subsystem?”
- “Which task owns this worktree?”
- “What changed since this design was approved?”

That combination is powerful.

### Integration pattern

The workflow system should likely expose its own query layer, perhaps in an MCP-style form, while coexisting with codebase-memory-mcp rather than replacing it.

---

## Tooling to build

## Phase 1: workflow kernel

The first build should likely be an MCP-first workflow kernel with registry, validator, linter, and renderer capabilities.

Core responsibilities:

- create and update structured records through strict MCP operations
- assign IDs
- enforce schemas
- enforce naming
- enforce state transitions
- enforce parent/child links
- manage supersession
- generate markdown views and other projections
- detect broken relationships and stale artifacts
- support agent-driven normalization by exposing deterministic query and mutation tools

This would likely solve a large proportion of the current pain.

### Example MCP tool categories

- create object
- approve object
- supersede object
- link parent and child
- split task
- render markdown
- validate project state
- show next work
- report stale artifacts
- preview normalization requirements
- validate candidate object data
- resolve possible links

---

## Phase 2: retrieval and context packing

The next layer should support:

- compact task packets
- project overviews
- team-focused briefings
- change summaries
- dependency views
- current approved spec lookup
- recent decision summaries

This reduces token burn and improves consistency across agent sessions.

---

## Phase 3: workflow query server

A planning-oriented MCP interface should expose things like:

- get project overview
- get milestone
- get feature
- get current approved specification
- get task context
- list blockers
- list child tasks
- list decisions in scope
- search workflow objects
- update state
- record decision
- record approval
- detect stale artifacts
- get handoff packet
- validate candidate object data
- suggest required fields for a requested action
- check possible duplicate bugs
- resolve likely links from loose references

This is where the MCP server becomes the primary control surface for agent-driven workflow operations.

---

## Other external tools worth considering

The following categories are worth investigating, even if specific tools are not selected immediately:

### 1. Fast indexed search / semantic retrieval tools

For workflow artifacts, not just code.

### 2. Schema validation tools

Useful for YAML or structured records, especially if validation can run locally and in CI.

### 3. Git worktree tooling

Anything that makes worktree lifecycle safer and more visible for multiple agents.

### 4. Task graph / dependency graph tooling

Lightweight tools that help render and validate parent-child and dependency structures.

### 5. Static documentation renderers

Useful if markdown views and dashboards are generated from structured records.

### 6. CI enforcement hooks

For validating:

- schema correctness
- referential integrity
- no stale active duplicates
- approval consistency
- render consistency between structured record and markdown projection

### 7. Logging / append-only event capture

Potentially useful for:

- approvals
- handoffs
- significant decisions
- auditability
- reconstructing workflow history

---

## Important design principles

The following principle set seems well aligned with the intended style of the system:

1. One canonical fact in one place
2. Markdown is a view, not an excuse for ambiguity
3. Strict core, forgiving interface
4. MCP is the primary formal control surface for workflow operations
5. Every artifact has a type, ID, owner, and state
6. Revisions supersede explicitly
7. Humans approve intent; agents execute scoped work
8. Contexts are small; links are strong
9. Parallelism is earned through isolation, not hope
10. Workflow state is machine-checked
11. Verification is attached to every unit of work
12. Bugs are first-class workflow objects, not just mislabeled tasks
13. Intake artifacts must be normalized before becoming canonical state
14. The workflow system must stay simpler than the project it manages

---

## Recommended next steps

### 1. Define the minimal canonical object model

Work out:

- artifact types
- required fields
- revision behavior
- links
- approval model

### 2. Define lifecycle state machines

For:

- design
- specification
- plan
- task
- milestone
- release
- decision

### 3. Define metadata governance

Work out:

- which metadata is core schema
- which metadata is registry-defined optional metadata
- how new metadata types are proposed and approved
- where metadata definitions live

### 4. Define ID strategy

Decide between:

- time-sortable distributed IDs
- hybrid IDs
- or another merge-safe Git-native approach

### 5. Define source-control operating rules

Especially:

- worktrees vs branches
- naming
- lifetime limits
- merge requirements
- stale detection

### 6. Define agent integration model

Clarify:

- what lives in runtime-native agent files
- what lives in workflow schema and policy
- what gets generated as task packets
- how human approvals are surfaced to agents
- how conversational intake maps to formal workflow operations
- how markdown intake artifacts are interpreted and normalized before commit
- how the MCP interface is structured for agent use

### 7. Define bug workflow and incident policy

Work out:

- bug object schema
- bug lifecycle states
- triage rules
- severity classes
- when a bug becomes an incident
- regression verification requirements
- when a spec change should be used instead of a bugfix path

### 8. Build the first tool

Recommended first build:

> MCP-first workflow kernel + validator/linter + markdown renderer

Not a full autonomous orchestration system yet.

That should provide the fastest value while keeping the system simple.

---

## Closing recommendation

The strongest direction is:

> Build a Git-native, schema-backed, MCP-first workflow kernel with markdown projections, deterministic validation, and agent query tools.

This should improve workflow for humans and AI agents together, preserve the readability of markdown, reduce drift, and create a foundation for later MCP-style planning memory and multi-agent orchestration.

The main caution is to avoid automating too early. If orchestration is built before the object model, state machine, and validation rules are solid, the system will simply automate inconsistency.

The right order is:

1. canonical structure
2. validation and rendering
3. retrieval and context packing
4. agent integration
5. orchestration and automation

That sequence offers the best chance of producing a workflow system that is simple, minimal, complete, composable, and durable.