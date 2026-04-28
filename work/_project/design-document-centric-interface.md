# Document-Centric Interface Design

- Status: approved design
- Approved: 2026-03-18
- Purpose: define the document-centric human interface model for Kanbanzai
- Date: 2026-03-18
- Related:
  - `work/design/workflow-design-basis.md` §2, §3, §4, §6
  - `work/spec/phase-1-specification.md` §6, §7, §15
  - `work/design/agent-interaction-protocol.md`
  - `work/design/product-instance-boundary.md`
  - `work/design/document-intelligence-design.md` §5, §6, §7, §14

---

## 1. Purpose

This document defines the document-centric interface model for Kanbanzai.

Kanbanzai's primary purpose is the coordination of AI agent teams to efficiently turn designs into working software. It does this through an MCP server that replaces ad-hoc search, grep, and manual context-gathering with structured operations — presenting the right instructions, the right context, and the right constraints to each agent at the right time. The management and coordination of AI agents is as important as, and arguably more important than, the creation and management of design documents.

This document addresses one side of that system: the human interface. The core proposition is that the human interface should be **documents and chat**, not entity management. Humans write and read documents. Humans make decisions in conversation. The system and its agents handle the structured bookkeeping behind the scenes.

This is a design-level change to how the system presents itself to humans. It does not eliminate or downgrade the internal entity model, the agent coordination layer, or the MCP interface — it separates the human interface from the internal representation.

## 2. Problem Statement

The current design exposes the internal entity model directly to humans. Creating a decision means writing a decision record. Updating a feature means operating on a feature entity. The human workflow involves explicit entity management: allocating IDs, setting statuses, filling fields, maintaining a separate decision log.

This is slow and laborious for human designers. It requires them to think in terms of the system's data model rather than in terms of the documents they naturally produce. The decision log, in particular, imposes a ceremony that adds overhead without adding clarity — decisions made during design work are recorded a second time in a separate structured format that the designer must explicitly create and maintain.

In practice, when the need for a process and tool first arose during the development of Basil and Parsley, a decision folder and template were created but used only once. The friction is real.

The root cause is not that the system tracks decisions or maintains structured state. The root cause is that the system's human interface is the entity model rather than the documents humans naturally work with.

## 3. Core Design Decision

**The human interface is documents and chat. The internal representation is entities and structured state. The agent layer mediates between them.**

This means:

- Humans write, read, review, and approve documents.
- Humans make decisions and request changes through conversation.
- AI agents extract structured information from documents and conversations.
- AI agents create and maintain internal entities (decisions, tasks, links, status) from that extracted information.
- When a human retrieves a document, the system returns a coherent markdown document — not an entity dump.
- The system's internal model (entities, lifecycles, state machines, referential integrity) is preserved and enforced, but operated by agents on behalf of humans.

## 4. The Document Types

The system recognises the following document types. These are listed in the order they typically appear in the design-to-implementation progression.

### 4.1 Proposal

A small, informal document putting forward a broad idea and some ways it could be realised. It is an unfinished design — it has as many questions as potential answers — but it carries some decisions that will be carried forward.

Proposals are usually created by a human designer. In practice they are often pasted directly into a chat conversation. The system should be able to accept and store them, but it is also acceptable for the designer to keep them outside the system.

- Created by: human designer (usually)
- Style: informal prose, exploratory, question-heavy
- Audience: human designers, AI agents

### 4.2 Research report

A substantive piece of research, usually conducted by an AI agent (often involving online investigation), with findings recorded for use in design work. Research reports are typically consumed during the draft design stage.

- Created by: AI agent (usually)
- Style: structured findings, analysis, comparisons
- Audience: human designers, AI agents

### 4.3 Other report

General reports about the project: performance, code quality, architectural observations, recommendations for reorganisation. These are typically created by AI agents as part of ongoing project health monitoring.

- Created by: AI agent (usually)
- Style: varies — analytical, diagnostic, advisory
- Audience: human designers, AI agents

### 4.4 Draft design

A semi-formal design document that shows its working. Each feature may have multiple draft design documents. Rejected design elements and decisions are recorded in draft designs — they are the backbone of the iterative design process.

When a significant number of decisions are rejected or a major direction change occurs, a new draft is created with the rejected paths removed. Once all main decisions are resolved, a design document is created from the final draft.

In projects with UI/UX work, draft designs may include art files and images — preliminary UI designs, wireframes, logos, icons.

- Created by: human designers and AI agents collaboratively
- Style: semi-formal prose, shows reasoning, records rejected alternatives
- Audience: human designers, AI agents

### 4.5 Design

The final design document for a feature. More formal than a draft design, but not a formal specification — that is the job of the specification document.

Each feature should have one design document. A design document may go through a few iterations before settling, but it is the authoritative document from which a specification is produced.

A design document usually contains only the decisions that led to the features in the design. However, it sometimes includes selected rejected decisions to pass context to documenters — typically when the design breaks from conventions established by similar projects or products.

If too many proposals or decisions are rejected during the current design document's iteration, it is downgraded to a draft and a new design document is created from the agreed decisions.

The design document is used to generate the specification and inform the user documentation, so it must contain enough detail for both purposes while maintaining enough readability for designers to understand and verify.

Projects with UI/UX work may also have final art and design files that accompany the design.

- Created by: AI agent (usually, as a distilled summary of the final draft design), reviewed and approved by human designer
- Style: formal prose, design rationale, clear decisions
- Audience: human designers, documenters, AI agents

### 4.6 Specification

The formal specification for a feature. Used as the basis for the dev plan and for verification after implementation to confirm that everything specified was implemented and that everything implemented meets the specification.

The specification informs test planning. It may include specifications for documentation. It defines the basis for the definition of done.

In projects with UI/UX work, specifications may include art and UI designs as part of the formal specification.

- Created by: AI agent
- Style: formal, precise, unambiguous — as terse as necessary to ensure unambiguous planning and testing
- Audience: AI agents (primary), human designers (verification)

### 4.7 Dev plan

A formal plan for implementing the specification. Created by an AI agent acting as a development lead. This is a working document for AI agents — it can be as terse as necessary since only agents need to read it.

The dev plan should contain all the steps needed to create the code, tests, support scripts, definition of done, verification steps, and instructions to documentation agents on what needs to be done.

- Created by: AI agent
- Style: terse, structured, task-oriented
- Audience: AI agents

### 4.8 User documentation

The documentation delivered to end users of the product.

The scope varies by project:
- Small projects: a README is sufficient.
- Medium projects: a reference document may be necessary.
- Large projects: a multi-page manual.

Documentation content is derived from the design document and the specification. Documentation is written to the design, with reference to the specification when formality is required. Documentation is never written to the dev plan.

- Created by: AI agent (usually), reviewed by human designer
- Style: varies by audience and scope
- Audience: end users of the product

### 4.9 Custom and unrecognised types

The eight types listed above are the **recognised types** — the system knows their role in the pipeline, their formality level, their entity extraction patterns, and their review expectations.

The system is not limited to these types. Any markdown document can be added to the system regardless of its type. Projects that use ADRs, postmortems, runbooks, vendor evaluations, onboarding guides, or any other document type as standard practice can register those as recognised types with their own pipeline rules (normalisation style, entity extraction patterns, review expectations, formality level).

- **Recognised types** receive type-specific pipeline behaviour: normalisation, entity extraction, formality treatment, review gates, and a defined position in the refinement chain.
- **Unrecognised or custom-typed documents** are stored, indexed, and queryable but receive no type-specific pipeline behaviour. They are full participants in the document graph — structurally analysed, classified, connected through shared concepts and entity references — without being recognised participants in the design-to-implementation pipeline.

The document intelligence layer (`work/design/document-intelligence-design.md` §14.5) is type-agnostic — structural analysis, fragment classification, concept extraction, and entity reference detection work identically on all documents regardless of type.

## 5. The Formality Gradient

Documents move from informal to formal as they progress from early design toward implementation. This is not just a change in precision — it is a change in style, from sentences of prose toward definitions and structured lists.

| Document type | Formality | Style | Ambiguity tolerance |
|---|---|---|---|
| Proposal | Low | Conversational prose, questions, sketches | High — exploration is the point |
| Draft design | Medium-low | Semi-formal prose, reasoning visible | Medium — captures uncertainty explicitly |
| Design | Medium-high | Formal prose, rationale, clear decisions | Low — decisions should be unambiguous |
| Specification | High | Precise definitions, structured | None — precision is the point |
| Dev plan | High | Terse, structured, task-oriented | None — agents need unambiguous instructions |

This gradient matters to the system because it affects how documents are treated:

- Early-stage documents (proposals, draft designs) receive more normalisation latitude from the system. Language can be cleaned, structure improved, ambiguities flagged.
- Late-stage documents (specifications, plans) should be precise and stable. Once approved, they are returned verbatim.
- The human side of the gradient is prose. The AI side of the gradient is definitions. This reflects how humans and AI agents naturally think and work.

## 6. The Document Lifecycle

### 6.1 The design-to-implementation process

The process that documents flow through:

1. Brainstorm + research → draft design(s)
2. Draft design(s) + more research + decisions → design
3. Design → formal specification
4. Design + formal specification (with implementation as final truth) → documentation content
5. Formal specification → dev plan (including testing plan and documentation plan)
6. Dev plan → code implementation
7. Dev plan → test implementation
8. Dev plan + documentation content → documentation implementation

### 6.2 Design iteration

Design iteration occurs during stages 1 and 2.

Design iteration can also occur after testing if implementation reveals that the design was incorrect. In that case, the process returns to stage 1 or stage 2 as appropriate.

### 6.3 Where decisions are made

Decisions are made during the design stages (1 and 2) and are recorded in the design documents. The most comfortable way for decisions to be made is in a chat conversation between the AI agent and the human designer.

Implementation decisions — choices about how to realise the specification — belong to the implementors. In this system, the implementors are AI agents. If a human designer needs to make an implementation decision, there is a problem with the specification: it should be clarified rather than patched with an ad-hoc decision.

This does not mean implementation decisions are unrecorded. It means they are recorded in the dev plan, not in the design documents, and they do not require human approval.

## 7. The Human Interface Contract

### 7.1 Documents in, documents out

Humans interact with the system through documents. They put documents in (by writing or editing markdown, or by having a conversation that results in document changes). They get documents out (by requesting a document from the system and receiving coherent markdown).

It does not matter to the human how documents are stored internally. What matters is:

- A document that goes in comes back out looking like the same document.
- The system does not radically change meaning or make substantive changes to prose without approval.
- The system may clean language, tidy formatting, and normalise structure on the way in — but the human approves before the document becomes canonical.

### 7.2 Document normalisation on ingest

When a document enters the system, agents may normalise it: cleaning spelling and grammar, tightening prose, improving structure, resolving minor ambiguities. This is expected and desirable. In a team setting, normalisation helps maintain a consistent house style and level of verbosity — different contributors write differently, and the system should smooth those differences.

Normalisation may rewrite substantially, not just patch individual words. Good prose sometimes requires restructuring a paragraph rather than correcting a sentence. This is acceptable provided the meaning is preserved and the human approves the result before the document becomes canonical.

On subsequent edits, it is preferable to normalise only the changed sections. However, for coherence of prose this is not always possible — a change to one paragraph may require adjusting the flow of surrounding text. The agent should use judgement, and the human has the final say.

The normalisation style — how formal, how terse, how much rewriting is acceptable — should be configurable at the project level, so teams can tune it to their preferences.

### 7.3 The approve-before-canon workflow

1. A human writes or discusses a document (or changes to a document) in chat.
2. The agent normalises — cleans language, resolves ambiguity, structures internally — and presents the result: "I've updated the document, take a look."
3. The human reviews the rendered document. Either approves ("that looks great") or requests changes ("can you change X to Y").
4. Once approved, the document is canonical. From that point, the system returns it unchanged.

This is the standard conversational editing workflow: the human says what they want, the agent makes it so, the human verifies. The system formalises this into the approval contract.

### 7.4 Canonical documents are stable

Once a document is approved and committed to the system as canonical, the system returns it verbatim on retrieval. The system does not re-render, re-summarise, or re-normalise canonical documents on the way out. The approved form is the stored form.

This ensures multi-user consistency: every person who retrieves a canonical document sees the same document, word for word.

### 7.5 Chat as the decision-making interface

Humans make design decisions through conversation with AI agents. The agent is responsible for:

- capturing decisions from the conversation
- recording them in the appropriate document
- creating or updating internal structured records (decision entities, entity links, status changes)
- presenting the updated document for human approval

The human never has to file a decision record, allocate an ID, or set a status. They have a conversation and approve the result.

### 7.6 Two retrieval modes

The system serves two audiences with different needs:

- **Human retrieval:** returns the approved canonical markdown document, intact, readable as prose. The system may add navigational metadata (table of contents, cross-references) but does not alter the substance.
- **Agent retrieval:** may return the same markdown, or may return structured fragments, entity data, decision lists, field values — whatever the agent needs for its task. The system does not have to serve agents the same way it serves humans.

## 8. The Internal Model

### 8.1 Entities are not downgraded

The internal entity model — Epic, Feature, Task, Bug, Decision — is preserved. Entities have lifecycles, state machines, referential integrity, and structured fields. The system enforces these internally.

What changes is that humans do not directly operate the entity model. Agents mediate. The entity model is the system's internal truth; documents are the human-facing truth. Both represent the same underlying reality.

### 8.2 Documents and entities relate bidirectionally

**On the way in:** when a document is ingested, agents extract structured information from it. A design document may produce or update Feature records, Decision records, and links between them. A specification maps to the spec content on a Feature. A dev plan spawns Tasks.

**On the way out:** when a document is retrieved, the system returns the canonical stored form. For new documents that have not yet been approved, the system or agents may assemble drafts from entity data — pulling in relevant decisions, status, requirements — but once approved, the document's identity is fixed.

### 8.3 Internal fragmentation is an implementation detail

Whether the system stores documents as whole files, as fragments linked to entities, or with a layer of structured markup is an implementation detail. The human does not need to know or care.

However, some degree of internal structuring is necessary for the system to do its job. If documents are at least partially indexed or annotated internally, the system can:

- enforce cross-document consistency (a decision that appears in a design document and affects a specification can be checked for agreement)
- detect when an edit to one document invalidates another
- answer queries like "show me all decisions affecting Feature X" without full-text search
- flag inconsistencies between the design and the specification

The principle is: fragment internally for consistency, present externally as whole documents.

The mechanism for this internal structuring — a four-layer analysis model producing a queryable document graph — is defined in `work/design/document-intelligence-design.md`.

### 8.4 Context assembly for agents

Internal fragmentation serves a second purpose beyond consistency: it enables precise context assembly for AI agents.

The central problem in coordinating AI agent teams is context. Every agent has a finite context window. If each agent must ingest the full design library to understand its task, the system does not scale — contexts fill with irrelevant material, agents work slowly, and the risk of misinterpretation grows.

By structuring documents internally — indexing decisions, linking requirements to features, connecting spec sections to design rationale, linking tasks to the specific parts of a plan and spec they implement — the system can compose a targeted context for each agent. An agent implementing a specific task receives the relevant slice of the design, the relevant specification sections, the relevant decisions, and the relevant constraints — assembled from fragments across the document layers — without receiving every design document in the project.

This is the architectural payoff of the document-to-entity mapping. Humans work with whole documents because that is how humans think. The system fragments documents internally because that is how it can efficiently coordinate many agents working in parallel, each with precisely the context they need and nothing more.

The document graph defined in `work/design/document-intelligence-design.md` is the backend for this context assembly — it provides the indexed fragments that the context assembler composes into targeted packets.

### 8.5 Decision as an internal entity

Decision remains a first-class entity in the system's internal model. It has an ID, a lifecycle, links to affected entities, rationale, and the other fields defined in the current specification.

What changes is that the human workflow for creating decisions is no longer "write a decision record." It is "have a conversation or write a design document." The agent extracts the decision, creates the internal record, and ensures the decision appears in the right documents when they are retrieved.

The decision log as a human-facing artifact that the designer manually maintains — that goes away. The decision log as an internal system capability — list all decisions, filter by feature, check for conflicts, trace rationale — that stays and becomes more useful, because the system maintains it rather than the human.

## 9. Document-to-Entity Mapping

The system needs rules for how document types relate to entities.

| Document type | Primary entities produced or updated |
|---|---|
| Proposal | May create draft Feature records; may note open questions |
| Draft design | Updates Feature records; creates Decision records (including rejected alternatives); links decisions to features |
| Design | Finalises Feature design; finalises Decision records; links to specification expectations |
| Specification | Creates or updates the spec content linked from a Feature |
| Dev plan | Creates Tasks; links tasks to features; defines verification expectations |
| Research report | May inform decisions; may create or update KnowledgeEntry records (post-Phase 1) |
| User documentation | Links to features and specifications; does not create entities |

These mappings are not rigid. The agent layer interprets document content and applies judgement about what entities to create or update. The mappings above describe the typical relationship.

## 10. Effect on Bootstrap Workflow

During bootstrap (before the tool exists), this model applies through manual agent-mediated practice:

- Humans write design documents and have conversations.
- Agents extract decisions and update planning documents.
- Agents maintain cross-references and consistency.
- The decision log (`work/plan/phase-1-decision-log.md`) continues to exist as an internal tracking artifact maintained by agents, not as a human-facing workflow step.
- Humans are not required to write decision records. They write and review design documents.

When the tool exists, these practices become system-enforced.

## 11. Effect on the Current Design

This document proposes an interface layer that sits on top of the existing entity model. The following areas of the current design would need to be updated to incorporate this model.

### 11.1 Workflow design basis

- §6.3 (Intake/Canonical/Projection taxonomy): documents become a fourth category or a refinement of the taxonomy. Documents are not intake artifacts (they persist), not raw canonical records (they are prose, not YAML), and not projections (they are authored, not generated). They are the human-canonical form alongside the entity-canonical form.
- §6.4 (Normalization pipeline): the pipeline gains document-aware stages. Intake of a document produces entity updates. Retrieval of a document assembles from canonical state (for drafts) or returns the stored form (for approved documents).
- §8 (Object model): the entity model is preserved but supplemented by a document type model. Feature gains explicit document references (design, spec, plan) as distinct from embedded content fields.

### 11.2 Phase 1 specification

- §7 (Entity model): Decision remains as a first-class entity. No change to the entity set.
- §15 (Document requirements): expanded substantially to define the document types, the formality gradient, the approve-before-canon rule, and the retrieval contract.
- §16 (MCP interface): gains document-oriented operations (submit document, retrieve document, list documents by type or feature) alongside entity operations.
- Acceptance criteria: gains document round-trip criteria (a document submitted and approved must be retrievable verbatim).

### 11.3 Bootstrap workflow

- The tracking conventions change: humans work with documents, not entity records.
- The decision log convention changes from a human-maintained artifact to an agent-maintained internal record.
- The document placement rules (`work/design/`, `work/spec/`, `work/plan/`) already align well with the document type taxonomy.

## 12. Open Questions

### 12.1 Art files, images, and document bundles

Design documents and specifications may include accompanying files: wireframes, UI designs, logos, icons, diagrams. How should the system handle these?

Options include:
- Documents and their accompanying files live together in a folder (a bundle).
- The system tracks accompanying files as attachments to a document record.
- Art files are managed outside the system and referenced by path.

This question is recognised but deliberately deferred. The design should not make art file support impossible, but it should not be implemented until the need arises in practice.

### 12.2 Folder structure: by type or by feature?

The current design organises documents by type (all designs in one place, all specs in another). An alternative is to organise by feature (each feature has a folder containing its design, spec, plan, etc.).

If the system manages storage internally and presents documents on demand, the physical organisation may not matter — the system can present documents however the user requests, regardless of how they are stored.

This question should be resolved during implementation planning, not during interface design.

### 12.3 In-progress files

Should in-progress documents (not yet approved) live in the system database or stay local to the human's workspace?

Options include:
- In-progress documents stay local; only approved documents enter the system.
- In-progress documents are stored in the system with a draft status.
- The system maintains a link between the local working copy and the canonical version (linked by ID, date of last sync, date of last edit).

This affects how the system tracks work in progress and how agents can assist with draft documents. It should be resolved as part of detailed design.

### 12.4 How much belongs in the system vs project policy?

The system should define:
- the set of document types it understands
- the document-to-entity mapping rules
- the approve-before-canon contract
- the retrieval contract (verbatim for approved documents)

Project policy files should define:
- naming conventions for documents
- templates and required sections for each document type
- project-specific formality expectations

Users should be free to decide:
- how they organise in-progress work locally
- which optional document types they use
- how much ceremony they want around proposals and draft designs

The exact boundary between system, policy, and user freedom should be resolved during specification.

### 12.5 Document versioning and supersession

When a design document is updated after approval (because testing revealed a design flaw and the process returns to stage 1 or 2), what happens to the previous version?

The existing supersession model (§10 of the design basis) may apply: the old document version is marked as superseded, the new version links back to it. But the details need working out.

## 13. What This Document Is Not

- It is **not a specification**. It is a draft design that describes an interface model. A specification will follow if the design is accepted.
- It is **not a proposal to remove entities**. The internal entity model is preserved. This document changes how humans interact with the system, not what the system tracks internally.
- It is **not a proposal to turn the system into a document store**. The system maintains policy, enforces lifecycles, validates consistency, and manages structured state. Documents are the interface, not the whole system.
- It is **not a proposal to eliminate decisions**. Decisions are tracked, indexed, linked, and enforceable. They just aren't a human-facing workflow step.
- It is **not the whole system design**. This document addresses the human interface layer. The coordination and management of AI agent teams — context assembly, task delegation, progress tracking, verification — is the system's primary function and is addressed by the broader design basis and specification. This document complements that work; it does not supersede it.

## 14. Acceptance Criteria for This Design

This design is acceptable only if:

1. Humans can work entirely through documents and chat without directly managing entities.
2. The system's internal entity model is preserved with full lifecycle and integrity enforcement.
3. Approved documents are returned verbatim — the system does not silently alter canonical prose.
4. Decisions made in conversation are captured and tracked without requiring the human to file separate decision records.
5. The document type taxonomy is clear enough to guide naming, templating, and progression.
6. The formality gradient is explicit enough that the system can treat early-stage and late-stage documents appropriately.
7. The design is compatible with the Parsley aesthetic: simplicity, minimalism, completeness, composability.
8. The design does not prevent future support for art files, bundles, or multi-file documents.

## 15. Summary

The Kanbanzai system should present a document-centric interface to humans while maintaining a rich structured entity model internally.

Humans work with documents — proposals, draft designs, designs, specifications, and plans — through natural conversational editing. They make decisions in chat and review documents as markdown prose. The system and its agents handle the extraction, indexing, linking, lifecycle management, and consistency enforcement behind the scenes.

The key rules are:

> Documents are the human interface. Entities are the internal model. Agents mediate.

> Humans approve documents before they become canonical. Canonical documents are returned unchanged.

> Decisions belong in design documents, not in a separate human-facing log. The system tracks them internally.

> The formality gradient — from informal prose to precise definitions — reflects how humans think and how work progresses from design toward implementation.

This design does not reduce the system's capabilities. It changes who operates the machinery: the system and its agents, not the human designer.