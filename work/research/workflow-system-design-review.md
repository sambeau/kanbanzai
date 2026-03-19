# Workflow System Design Review

- Status: review memo
- Author: Claude
- Date: 2026-03-18
- Reviews: `workflow-system-design.md`
- Related:
  - `initial-workflow-analysis.md`

---

## 1. Purpose

This memo reviews `workflow-system-design.md` in light of the broader direction established in `initial-workflow-analysis.md`.

The goals of this review are to answer:

- Is the design proposal worth keeping?
- Is it worth merging into a single document with the analysis document?
- Which ideas are strong and should be preserved?
- Which parts need refinement?
- Are there any fundamental problems with the proposal?

---

## 2. Overall Assessment

The proposal in `workflow-system-design.md` is strong.

It is not merely a brainstorming note. It translates high-level ideas into a concrete design with:

- explicit principles
- a practical architecture
- a workable object model
- a plausible storage design
- an MCP-first interface
- a concurrency model
- a phased implementation plan

It is broadly aligned with the direction proposed in `initial-workflow-analysis.md` and should be retained.

### Verdict

- **Worth keeping:** yes
- **Worth evolving:** yes
- **Fundamentally flawed:** no
- **Ready to become the only design document:** not yet

The document is best understood as the **current primary design proposal**, while `initial-workflow-analysis.md` remains the broader rationale and research document.

---

## 3. Recommendation on Merging Documents

At this stage, the documents should **not** be merged into a single file.

They currently serve different roles:

### `initial-workflow-analysis.md`
This is the broader strategy and rationale document. It captures:

- the diagnosis of the Basil workflow
- lessons from current pain points
- evaluation of external systems
- framing principles
- broad recommendations

### `workflow-system-design.md`
This is the more concrete design proposal. It captures:

- the architectural model
- entity design
- interface design
- concurrency model
- phased implementation direction

### Recommendation

Keep both documents for now:

- `initial-workflow-analysis.md` as the research / rationale / strategic framing document
- `workflow-system-design.md` as the design proposal document

A future merge may make sense once the current round of research and brainstorming stabilizes.

---

## 4. Strong Ideas Worth Keeping

The following ideas in `workflow-system-design.md` are strong and should be preserved.

## 4.1 Workflow State Is the Source of Truth, Conversation Is the Interface

This is one of the strongest formulations in the proposal.

It cleanly captures the desired operating model:

- humans work through conversation
- agents mediate and normalize
- structured workflow state is authoritative

This matches the desired human/AI boundary very well and should remain central.

## 4.2 Disciplined Normalisation

The design principle in section 2.2 is excellent.

Especially valuable are these aspects:

- normalisation is AI-driven
- it applies to both commands and content
- humans do not need to learn internal syntax
- humans do not need to write perfectly structured markdown
- the AI should not silently invent important facts
- the normalisation step should be visible and reviewable

This principle is highly compatible with the intended workflow style and should remain foundational.

## 4.3 The MCP-First Model

The proposal correctly centers MCP as the primary formal interface for the workflow system.

This is aligned with the intended model:

- humans use chat
- agents use MCP
- strict workflow operations happen through typed tools
- CLI exists as a secondary convenience

This should be kept.

## 4.4 One File Per Entity

This is one of the strongest implementation choices in the proposal.

Benefits include:

- better git diffs
- lower merge conflict surface
- direct inspectability
- easier validation
- simpler cache rebuilding
- clean mapping between entities and state files

This is a very good fit for a Git-native workflow kernel.

## 4.5 Git-Native State with Local SQLite Cache

This is also a strong design choice.

It preserves:

- text-based canonical state
- Git friendliness
- easy inspection and diffing

while still allowing:

- fast local query
- indexing
- dependency analysis
- health checks

This hybrid model is sound and should be preserved.

## 4.6 Bugs as First-Class Workflow Objects

This is correct and necessary.

The proposal handles bugs much better than many workflow designs by giving them:

- their own schema
- their own lifecycle
- a classification model
- explicit distinction from ordinary tasks

This is a strong part of the design.

## 4.7 Concurrency and Worktree Strategy

The source control section is practical and strong.

The use of:

- worktrees
- branch hygiene
- file-level conflict prevention
- merge gates

shows good operational realism and is very much worth keeping.

## 4.8 The Conversational Boundary

The decomposition of the agent boundary into:

1. interpret
2. clarify
3. validate
4. normalise
5. execute
6. report

is especially useful.

This is not just philosophical; it gives the agent a concrete operating pattern.

That section should be preserved.

## 4.9 Phased Implementation

The phased implementation plan is sensible:

- kernel first
- retrieval/context next
- git integration next
- orchestration last

This sequencing is disciplined and reduces the risk of automating too early.

It should be kept.

---

## 5. Areas That Need Refinement

The proposal is strong, but some parts need refinement before being treated as settled.

## 5.1 The Object Model May Be Too Compressed

The current core state model includes:

- Epic
- Feature
- Task
- Bug
- Decision

This is a practical starting point, but it likely compresses too much responsibility into `Feature`.

In the current proposal, `Feature` appears to carry:

- the idea of the feature itself
- the specification handoff point
- the implementation lifecycle
- links to spec and plan documents

This works for a first draft, but it introduces a conceptual tension.

### Why this matters

The broader workflow being designed cares strongly about:

- approvals
- supersession
- revision history
- current approved spec
- distinguishing bug vs spec defect vs design problem

Those concerns are usually modeled more cleanly if at least some of these become first-class entities:

- `Feature`
- `Specification`
- `Plan`

### Review recommendation

The current simplified model is acceptable for now, but it should be treated as a likely **v1 simplification**, not necessarily the final ontology.

The design document should either:

- explicitly note that `Feature` is a composite entity in v1, or
- begin introducing `Specification` and `Plan` as first-class workflow objects

This is the most important modeling question still open.

## 5.2 The ID Strategy May Be Too Complex Too Early

The proposed block allocation strategy is thoughtful and coherent, but it may be more complex than needed for an initial implementation.

It requires:

- allocation state
- block reservation
- unused ID return
- coordination on main
- bookkeeping overhead

This may be justified later, but it increases system complexity early.

### Alternative worth evaluating

A distributed sortable ID format would avoid central reservation:

- time-based
- short
- collision-resistant
- paired with slug for readability

That would reduce coordination complexity at the cost of slightly less “nice” IDs.

### Review recommendation

Do not discard the current ID strategy, but treat it as an open design decision rather than a settled answer.

This topic likely needs explicit comparison of:

- block allocation
- distributed sortable IDs
- hybrid approaches

before implementation starts.

## 5.3 The Proposal Needs a Clearer Distinction Between Intake Artifacts and Projections

The newer clarified design direction distinguishes among:

- intake artifacts
- canonical records
- projections

The current `workflow-system-design.md` does not yet express this distinction strongly enough.

At the moment, the document describes:

- human-authored markdown
- workflow YAML state
- agent memory

That is useful, but it does not clearly separate:

### Intake artifacts
Human-authored material not yet canonical, such as:

- brainstorm notes
- rough markdown specs
- pasted bug reports
- review comments
- draft proposals

### Projections
Generated views derived from canonical state, such as:

- status reports
- roadmap summaries
- handoff docs
- dashboards

This distinction is important because the desired system behavior is:

- accept rough human markdown as input
- normalize it through AI
- commit formal structured state through MCP
- generate consistent projections from canonical state

### Review recommendation

The design doc should be updated to reflect this distinction more explicitly.

This is one of the most important refinements needed.

## 5.4 “Documents Should Be Generated, Not Maintained” Needs Slight Tightening

The underlying idea is good, but the phrasing is a little too broad.

As written, it could be interpreted as implying that all useful documents should be generated.

That is not quite the intended system.

The intended model is closer to:

- operational summaries and status views are generated
- human-authored designs and specs are normalized and validated
- raw human prose should not also become an unmanaged parallel state store

### Review recommendation

Refine the wording to distinguish:

- generated operational documents
- validated human-authored documents
- normalized intake artifacts

This is more precise and better aligned with the intended workflow.

## 5.5 The Knowledge Layer Needs Stronger Governance Rules

The knowledge store design is promising, but there is a risk of knowledge sprawl.

Potential failure modes include:

- duplicate entries
- stale entries
- overlapping decisions and knowledge
- weak links back to source entities
- “tips” being stored where more formal decisions should exist

### Review recommendation

The document should more clearly define what belongs in:

- `Decision`
- `KnowledgeEntry`
- `RootCauseAnalysis`
- `Specification`
- `Team convention`

Without this, `work/knowledge/` risks becoming another dumping ground.

This is not a fundamental flaw, but it does need governance design.

## 5.6 State Machine Design Needs Slight Clarification

Some lifecycle state machines appear to mix layers of responsibility.

For example, the `Feature` state machine appears to combine:

- spec lifecycle
- approval lifecycle
- implementation lifecycle
- review lifecycle

This is workable if `Feature` is intentionally composite, but confusing if not.

Similarly, bug lifecycle naming should be standardized consistently across documents.

### Review recommendation

Clarify whether:

- `Feature` is intentionally composite, or
- `Specification` and `Plan` are expected to separate later

and tighten lifecycle naming for consistency.

---

## 6. Are There Fundamental Problems?

There are no fatal architectural problems in the proposal.

The design is coherent and directionally correct.

However, there are two strategic tensions that should be explicitly acknowledged.

## 6.1 Tension: Composite `Feature` vs First-Class `Feature` / `Specification` / `Plan`

This is the biggest modeling issue.

The current design is simpler, but may not preserve enough structure once revisions, supersession, and approval history become more important.

This is not a flaw severe enough to invalidate the proposal, but it is a significant open modeling question.

## 6.2 Tension: Human-Friendly Sequential IDs vs Distributed-Safe IDs

The current proposal favors more human-friendly IDs with allocation support.

That is understandable, but the coordination cost may exceed the benefit in a distributed AI-heavy workflow.

Again, this is not fatal, but it is a major open design question.

---

## 7. Suggested Near-Term Direction

The proposal should remain the active design document, but it should be revised in the next iteration with the following goals.

## 7.1 Keep the Current Overall Structure

The overall document structure is good and should mostly remain:

- purpose
- principles
- lessons
- architecture
- object model
- interface
- concurrency
- delegation
- memory
- validation
- implementation phases

This is a solid backbone.

## 7.2 Revise the Architecture Section to Reflect the Intake / Canonical / Projection Split

Make the markdown model more precise by distinguishing:

- intake artifacts
- canonical workflow state
- generated projections

This will bring the design doc into stronger alignment with the clarified MCP-first normalization model.

## 7.3 Revisit the Object Model Before Implementation

Specifically:

- decide whether `Feature` remains composite in v1
- or whether `Specification` should become first-class now

This decision affects:
- lifecycle design
- approval handling
- supersession
- bug classification
- plan linking

## 7.4 Keep the MCP Interface but Expand It Around Normalisation Support

The current MCP interface is strong, but should likely add or emphasize tools that support AI-mediated normalization, such as:

- candidate validation
- duplicate detection
- required-field discovery
- likely link resolution
- preview of normalized commits

This is consistent with the intended agent role.

## 7.5 Keep the Current Phased Implementation Order

This is one of the strongest parts of the proposal and should be preserved.

Do not move orchestration earlier.

---

## 8. Recommended Merge Strategy for Documents

For now, use the documents as a two-layer set:

### Layer 1: Research and rationale
`initial-workflow-analysis.md`

### Layer 2: Design proposal
`workflow-system-design.md`

Once the open questions around:

- object model
- ID strategy
- markdown role distinctions
- memory governance

are clarified, the two documents may be merged into a single design proposal document.

That merge will be more useful once the current brainstorming phase is complete.

---

## 9. Final Verdict

`workflow-system-design.md` is a valuable and worthwhile proposal.

It should be retained and evolved, not discarded.

It is broadly aligned with the direction established in `initial-workflow-analysis.md`, and it contains many strong ideas worth carrying forward, especially around:

- disciplined normalization
- MCP-first control
- Git-native canonical state
- one-file-per-entity storage
- worktree-based concurrency
- bug-first-class treatment
- phased implementation

The proposal is not yet final. Its main areas for refinement are:

- the compression of `Feature` / `Specification` / `Plan`
- the complexity of the ID strategy
- the need to distinguish intake artifacts from generated projections
- stronger governance of knowledge entries
- tighter lifecycle semantics

### Final recommendation

- Keep both documents for now
- Treat `workflow-system-design.md` as the active design proposal
- Revise it in light of this memo and the latest MCP-first normalization model
- Consider merging documents only after the current research and brainstorming phase is complete

---