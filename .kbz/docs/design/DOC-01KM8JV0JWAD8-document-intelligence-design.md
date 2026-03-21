---
id: DOC-01KM8JV0JWAD8
type: design
title: Document Intelligence Design
status: submitted
feature: FEAT-01KM8JT7542GZ
created_by: human
created: 2026-03-21T16:14:48Z
updated: 2026-03-21T16:14:48Z
---
# Document Intelligence Design

- Status: draft design
- Date: 2026-07-18
- Purpose: define the structural analysis backend for design documents — the mechanism that enables code-intelligence-like capabilities for the document corpus
- Related:
  - `work/design/document-centric-interface.md` §7, §8, §9
  - `work/design/machine-context-design.md` §4.1, §8, §13
  - `work/design/workflow-design-basis.md` §6.4, §8.5, §15
  - `work/spec/phase-1-specification.md`

---

## 1. Purpose

This document defines how Kanbanzai structurally analyses, indexes, and queries design documents — providing AI agents with efficient, targeted access to document content without requiring full-document reads.

The system described here is the **structural analysis backend**. It is analogous to what `codebase-memory-mcp` provides for source code: a persistent, queryable representation of the document corpus that transforms brute-force text search into intelligent retrieval.

This is the mechanism behind the "internal fragmentation" described in the workflow design basis (`work/design/workflow-design-basis.md` §8.5) and the document-centric interface design (`work/design/document-centric-interface.md` §8.3–8.4). Those documents establish that the system fragments documents internally for consistency and context assembly while presenting them externally as whole documents. This document defines **how** that fragmentation works.

---

## 2. Problem Statement

### 2.1 The scale problem

A small project has a handful of design documents totaling a few thousand words. An AI agent can read them all in seconds. At this scale, structural analysis adds overhead without proportional value.

A mature project may have 50–200 design documents totaling 300K+ words. No single agent can hold them all in context. Multiple agents working in parallel multiply the cost of reading. The cost of repeated full-document reads across many agents and many tasks becomes significant — in tokens, in latency, and in the risk of missing relevant information buried in a large corpus.

### 2.2 The precision problem

When an agent reads a 40KB document looking for the answer to a specific question, there is a non-trivial risk it misses something or misinterprets it in the noise of surrounding content. When the system serves the agent the specific 800-byte fragment containing the answer, classified and contextualised, the risk drops.

### 2.3 The search problem

Grep finds text. It cannot:

- Find all **decisions** across the corpus (they don't all look the same textually)
- Find the **rationale** for a specific choice (it's prose, not a keyword)
- Navigate conceptual dependency chains
- Tell you what sections are **about** a concept without the concept being literally mentioned
- Distinguish a section that **defines** a term from one that **uses** it
- Detect whether two documents are **consistent** with each other

Design documents need the same kind of intelligence that code navigation tools provide for source code.

### 2.4 The context assembly problem

The machine-context design (`work/design/machine-context-design.md` §8) describes context assembly — composing targeted context packets for each agent from fragments across document layers. But assembly requires an index. Without structural analysis, context assembly degrades to "concatenate whole documents and hope they fit in the budget." With it, assembly becomes precise: the right fragments, from the right documents, for the right task.

---

## 3. The Code Intelligence Analogy

`codebase-memory-mcp` exploits a fundamental property of source code: **it has a grammar**. Tree-sitter parses that grammar into an AST, and from the AST the tool extracts a graph of entities (functions, classes, methods, variables) and relationships (calls, defines, tests, imports). The graph turns brute-force text search into intelligence — call path tracing, impact analysis, architecture overview — without reading every file.

Design documents occupy a structural middle ground:

- **Source code** has a formal grammar → deterministic parsing → complete structural graph
- **Design documents** have a semi-formal grammar (markdown syntax) plus implicit semantic structure (arguments, decisions, requirements, rationale) that follows conventions rather than rules
- **Free text** (novels, emails) has almost no exploitable structural patterns

Design documents have enough structure to parse mechanically, enough convention to classify semantically, and enough regularity to model as a graph. The analysis cannot be fully deterministic the way code parsing is — but it does not need to be. It needs to be **useful and stable**.

---

## 4. Design Principles

1. **The tool is a database. Agents are the curators.** Kanbanzai does not embed AI capabilities. It provides structure — the taxonomy, the classification schema, the graph storage, the query engine. Agents provide intelligence — classification, concept extraction, relationship detection. The tool never calls an LLM. It never needs API keys.

2. **AI runs at ingest time, not at query time.** Expensive analysis happens once per document version, when an agent ingests or updates it. Results are cached as persistent metadata. Subsequent queries are served from the cache. This amortises the cost of classification across all future reads.

3. **Graceful degradation.** The system is always useful at whatever level of analysis is available. If AI-assisted classification has not run, structural queries and pattern-based extraction still work. Layers build on each other but do not require each other.

4. **Fragment internally, present externally as whole documents.** This principle is inherited from the document-centric interface design (`work/design/document-centric-interface.md` §8.3). The structural index is internal machinery. Humans retrieving documents get the canonical approved form. Agents querying the index get targeted fragments.

5. **Deterministic layers are always current. AI-assisted layers are eventually consistent.** Markdown parsing and pattern extraction run on every document change (milliseconds, deterministic). AI-assisted classification catches up when an agent next interacts with the document.

6. **The classification schema is defined by the tool, populated by agents.** The tool defines the taxonomy of fragment roles, the schema for concept entries, the structure of relationship edges. Agents fill in the values. This ensures consistency regardless of which LLM the agent uses.

---

## 5. The Four-Layer Analysis Model

The structural analysis backend operates in four layers, each building on the one below. The layers differ in what they analyse, how they are triggered, and whether they are deterministic.

| Layer | Name | Trigger | Deterministic | Analogous to |
|---|---|---|---|---|
| 1 | Structural skeleton | Document change | Yes | Tree-sitter AST |
| 2 | Pattern-based extraction | Document change | Yes | Symbol resolution |
| 3 | AI-assisted classification | Agent curation | No (cached) | Call graph construction |
| 4 | Document graph | Layers 1–3 | Yes (given inputs) | Knowledge graph queries |

### 5.1 Layer 1: Structural skeleton

Parse the markdown document into a structural tree using a standard parser. This produces:

- **Section hierarchy** — every headed section as a node with level, title, byte offset, word count, byte count
- **Content blocks** — paragraphs, lists, tables, code blocks, and front matter identified within each section
- **Document outline** — a lightweight table of contents with section sizes

This is the equivalent of tree-sitter's AST for code. It is cheap, deterministic, and rebuilt on every document change in milliseconds.

**Immediate value:** targeted section retrieval. Instead of serving an agent a 40KB document, serve the 2KB section it needs. Multiply by 8 agents working in parallel on different tasks, and the token savings are substantial. More importantly, the agent receives a focused piece of information rather than a haystack.

### 5.2 Layer 2: Pattern-based extraction

On top of the structural tree, apply deterministic pattern matching:

- **Entity references** — regex patterns for workflow entity IDs (`FEAT-xxx`, `DEC-xxx`, `TASK-xxx`, `BUG-xxx`, `DOC-xxx`, `EPIC-xxx`). Every mention becomes a typed edge in the graph.
- **Cross-document links** — markdown links and backtick-quoted paths to other documents become edges.
- **Section classification by convention** — headers containing keywords like "Decision", "Rationale", "Requirements", "Open Questions", "Alternatives Considered", "Acceptance Criteria" are classified by their conventional role. This works because our documents follow conventions we control.
- **Front matter parsing** — document metadata (type, status, date, related documents) extracted and indexed.
- **Definition detection** — patterns such as bold-colon definitions (`**Term**: definition`) or glossary-style sections.

This layer is also cheap and deterministic. It provides the first level of semantic structure — not just "this is a section" but "this is a section containing decisions" or "this section references FEAT-042 and DEC-017."

This is the equivalent of symbol resolution in code — knowing what is defined where and what references what.

### 5.3 Layer 3: AI-assisted classification

Some structural patterns cannot be extracted by regex or header heuristics. This layer uses AI agents to classify document fragments at ingest time:

- **Fragment role classification** — is this paragraph a requirement, a constraint, a rationale, an assumption, a risk, or a narrative transition? An AI can classify these reliably; a regex cannot.
- **Concept extraction** — what concepts does this section introduce, define, or depend on? Not just named entities (those are Layer 2) but conceptual dependencies.
- **Implicit cross-references** — two sections in different documents discuss the same concept using different terminology. An AI can detect this; pattern matching cannot.
- **Section summaries** — a one-line characterisation of each section, indexed for search.
- **Argument structure** — in a design document, detecting the rhetorical structure: claim, evidence, counterargument, resolution.

This layer is non-deterministic but cached. The analysis runs once per document version, and the results are stored as persistent metadata. The non-determinism is manageable: results are stable for a given document version, human-reviewable, and do not need to be perfectly reproducible — they need to be useful.

### 5.4 Layer 4: The document graph

Layers 1–3 produce nodes and edges. Layer 4 is the persistent graph itself and the query operations over it. It is described in §7 (graph schema) and §9 (query operations).

---

## 6. The AI-at-Ingest-Time Architecture

### 6.1 Core mechanism

The structural analysis backend does not embed AI. Instead, it extends the existing AI-mediated normalization pipeline (`work/design/workflow-design-basis.md` §6.4) with a classification step.

The existing pipeline already requires an agent to perform entity extraction (stage 6) after a document is approved. The structural classification is an extension of that same stage — the agent classifies fragments while it is already holding the document in context and extracting entity data.

The flow:

1. A document is created or updated.
2. The tool runs Layers 1–2 immediately: markdown parsing, section tree construction, entity reference extraction, cross-document link detection, front matter indexing. These are deterministic and complete in milliseconds.
3. The tool stores the structural index.
4. The tool returns the structural skeleton to the agent, along with the classification taxonomy and schema.
5. The agent classifies each section using its own LLM capabilities: assigns fragment roles, extracts concepts, generates summaries, identifies implicit dependencies.
6. The agent sends the classification results back to the tool via an MCP operation.
7. The tool validates the classifications against the schema, stores them as persistent metadata, and updates the graph.
8. All subsequent queries are served from the cached graph.

Steps 4–6 are the AI-at-ingest-time mechanism. The tool provides the form; the agent provides the judgement.

### 6.2 Why the tool does not call an LLM

The alternative — the tool calling out to an LLM API directly — would require:

- API keys embedded in the tool configuration
- External service dependency (network, billing, availability)
- The tool becoming an AI system rather than a database
- Non-deterministic runtime behaviour at query time
- Harder testing (mock LLM responses required)

By keeping AI in the agent layer and structure in the tool layer:

- The tool is fully deterministic and testable — given the same inputs (document + classifications), it always produces the same graph
- No external dependencies — the tool is a binary that reads and writes files
- Agent-agnostic — any LLM can do the classification; the tool does not know or care which model the agent uses
- The separation of concerns matches the rest of Kanbanzai: humans own intent, agents own intelligence, the tool owns structure

### 6.3 The classification protocol

The tool defines the taxonomy and the schema for classifications. The agent fills in the values. The tool validates and stores.

When the tool returns the structural skeleton for classification, it includes:

- The document ID and version
- The list of sections with their paths, content hashes, and word counts
- The classification taxonomy (the valid set of fragment roles)
- Null fields for the agent to populate: role, concepts introduced, concepts used, summary, dependencies

The agent returns:

- Classifications for each section: role, concepts, summary, dependencies
- Any implicit cross-references detected

The tool validates that:

- Every section is classified (or explicitly marked as not applicable)
- Roles are drawn from the defined taxonomy
- Concept entries conform to the concept schema
- Content hashes match (the document has not changed between skeleton and classification)

If validation fails, the tool rejects the classification with a specific error. The agent can retry.

### 6.4 Documents that arrive without an agent

If a human edits a markdown file directly and commits it — bypassing the agent normalization pipeline — Layers 1–2 run (deterministic, triggered by file change detection) but Layer 3 does not, because no agent was in the loop.

The system handles this through eventual consistency:

- The document is flagged as **indexed but unclassified**
- Layers 1–2 queries work normally — structural search, entity references, section retrieval
- Layer 3 catches up when:
  - An agent next touches the document (editing, reviewing, or querying it deeply)
  - An orchestrator agent performs housekeeping and processes pending classifications
  - A human explicitly asks an agent to reindex

The system degrades gracefully. It is always useful at whatever level of analysis is available. It gets more useful as curation accumulates.

---

## 7. The Document Graph Schema

### 7.1 Node types

| Node type | Source layer | Description |
|---|---|---|
| **Document** | Layer 1 | A whole document with metadata: type, status, formality level, date, author |
| **Section** | Layer 1 | A headed section within a document: level, title, path, byte offset, word count, byte count |
| **Fragment** | Layer 3 | A classified piece of content within a section: role, confidence |
| **EntityRef** | Layer 2 | A mention of a workflow entity by ID within a section |
| **Concept** | Layer 3 | An extracted concept or term that appears across documents |
| **Question** | Layer 2/3 | An open question or flagged ambiguity |

### 7.2 Edge types

| Edge type | From → To | Description |
|---|---|---|
| **CONTAINS** | Document → Section, Section → Fragment | Hierarchical containment |
| **REFERENCES** | Section → EntityRef | Section mentions a workflow entity |
| **LINKS_TO** | Section → Section | Explicit markdown link or backtick-path reference |
| **DEPENDS_ON** | Fragment → Fragment | Logical dependency (AI-classified) |
| **SUPERSEDES** | Document → Document, Fragment → Fragment | Replacement relationship |
| **INTRODUCES** | Fragment → Concept | Section where a concept is first defined or introduced |
| **USES** | Fragment → Concept | Section that uses a concept defined elsewhere |
| **REFINES** | Document → Document | Refinement relationship (spec refines design) |

### 7.3 Example graph fragment

A simplified example showing how a design document and its corresponding specification connect through the graph:

```/dev/null/example.txt#L1-14
Document: workflow-design-basis.md (type: design-basis, status: approved)
  └── Section: §6.4 "AI-Mediated Normalization Pipeline" (level: 3, words: 450)
        ├── Fragment (role: definition, concept: "normalization-pipeline")
        │     ├── INTRODUCES → Concept: "normalization-pipeline"
        │     └── REFERENCES → EntityRef: FEAT-xxx
        └── LINKS_TO → Section: document-centric-interface.md §7.1

Document: phase-1-specification.md (type: specification, status: approved)
  └── Section: §6 "Document Operations" (level: 2, words: 1200)
        ├── Fragment (role: requirement)
        │     ├── USES → Concept: "normalization-pipeline"
        │     └── DEPENDS_ON → Fragment (design-basis §6.4, role: definition)
        └── REFINES ← Document: workflow-design-basis.md
```

---

## 8. The Refinement Chain

Design documentation has a structural pattern with no direct analogue in source code: **the refinement chain**.

A proposal introduces an idea vaguely. A design document refines it with specifics. A specification pins it down with precision. An implementation plan decomposes it into tasks. User documentation explains it to end users.

Each stage refines the same concept with increasing precision and decreasing ambiguity. The document graph models this through `REFINES` edges between documents and `DEPENDS_ON` edges between corresponding fragments across documents.

### 8.1 What the refinement chain enables

- **Provenance tracing** — trace a requirement backward from the implementation plan through the spec, the design, and the original proposal. Understand *why* a requirement exists and what intent it serves.
- **Forward impact analysis** — trace a design change forward to see which specs, plans, and tasks it affects.
- **Refinement gap detection** — detect when a design introduces a concept that the specification does not cover, or when a spec contains a requirement that has no corresponding design rationale.
- **Full-chain views** — an agent can request "show me the full refinement chain for concept X" and see the progression from vague idea to precise spec to concrete task in one coherent view.

### 8.2 How refinement edges are established

- **Explicit:** documents declare their basis in front matter (the `Related:` or `Basis:` field). These become `REFINES` edges.
- **Conventional:** document types have a natural refinement order (proposal → design → specification → plan). Documents of a more formal type that reference documents of a less formal type imply refinement.
- **AI-assisted:** Layer 3 classification detects corresponding fragments across documents — sections that discuss the same concept at different levels of formality — and creates `DEPENDS_ON` edges between them.

---

## 9. Query Operations

The document graph is exposed through MCP operations. These are the queries the system can answer.

### 9.1 Document-level operations

| Operation | Description | Layers required |
|---|---|---|
| `doc_list` | List all indexed documents with metadata (type, status, date, classification state) | 1 |
| `doc_outline(doc_id)` | Structural outline: section tree with titles, levels, word counts, byte counts | 1 |
| `doc_get(doc_id)` | Retrieve the full canonical document | 1 |

### 9.2 Section-level operations

| Operation | Description | Layers required |
|---|---|---|
| `doc_section(doc_id, section_path)` | Retrieve a specific section by its path in the section tree | 1 |
| `doc_sections(doc_id, options)` | Retrieve multiple sections matching filters (level, role, size) | 1–3 |

### 9.3 Search operations

| Operation | Description | Layers required |
|---|---|---|
| `doc_search(query, doc_types?, roles?)` | Find sections or fragments matching a text query, filtered by document type and fragment role | 1–3 |
| `doc_find_by_concept(concept)` | Find all documents and sections that introduce or use a concept | 3 |
| `doc_find_by_role(role, scope?)` | Find all fragments of a given role across the corpus — all decisions, all requirements, all open questions | 3 |
| `doc_find_by_entity(entity_id)` | Find all sections that reference a specific workflow entity | 2 |

### 9.4 Analysis operations

| Operation | Description | Layers required |
|---|---|---|
| `doc_trace(entity_id)` | Trace an entity through the refinement chain: proposal → design → spec → plan | 2–3 |
| `doc_impact(section_id)` | What depends on this section? What would be affected by a change? | 3 |
| `doc_gaps(feature_id)` | What document types are missing for this feature? (No spec? No plan?) | 1–2 |
| `doc_consistency(doc_a, doc_b)` | Surface potential inconsistencies between two documents | 3 |

### 9.5 Curation operations

| Operation | Description | Layers required |
|---|---|---|
| `doc_ingest(doc_id, content)` | Ingest or update a document — runs Layers 1–2 and returns the structural skeleton for classification | 1–2 |
| `doc_classify(doc_id, classifications)` | Submit AI-assisted classifications for a document — populates Layer 3 | 3 |
| `doc_pending` | List documents that are indexed but unclassified (Layer 3 not yet run) | 1 |

### 9.6 Layered availability

Operations degrade gracefully based on which layers have run:

- **Layers 1–2 only** (always available): `doc_list`, `doc_outline`, `doc_get`, `doc_section`, `doc_find_by_entity`, `doc_gaps`, `doc_pending`
- **Layer 3 required** (after AI classification): `doc_find_by_concept`, `doc_find_by_role`, `doc_trace` (full chain), `doc_impact`, `doc_consistency`, `doc_search` (with role filters)
- **Partial Layer 3**: operations work with whatever classifications exist. Unclassified sections are included in results but without role or concept metadata.

---

## 10. Vertical and Horizontal Slicing

The document graph enables two fundamental query patterns that grep cannot provide.

### 10.1 Vertical slices

Everything about **one concept** across all documents. A vertical slice follows `INTRODUCES` and `USES` edges from a Concept node to all the fragments that discuss it, across the entire corpus.

Example: "Tell me everything about lifecycle transitions" returns fragments from:
- The design basis (where the concept is introduced)
- The specification (where the rules are defined)
- The implementation plan (where the tasks are listed)
- The decision log (where choices were made)

An agent implementing lifecycle transitions receives exactly this slice — deep and narrow. It does not need to read four complete documents to find the relevant sections.

### 10.2 Horizontal slices

Everything of **one type** across all documents. A horizontal slice filters by fragment role across the entire corpus.

Example: "Show me all open questions" returns every fragment classified as a question, from every document. A planning agent doing a gap analysis gets this slice — wide and shallow.

Example: "Show me all decisions" returns every fragment classified as a decision, regardless of which document contains it. An agent checking for contradictions gets a complete picture without reading every design document.

### 10.3 Combined slicing

Vertical and horizontal slices can be combined: "Show me all requirements related to document storage" is a vertical slice (concept: document storage) intersected with a horizontal slice (role: requirement).

---

## 11. The Fragment Role Taxonomy

Layer 3 classifies fragments using a fixed taxonomy of roles. The taxonomy is defined by the tool and enforced by schema validation.

### 11.1 Roles

| Role | Description | Example |
|---|---|---|
| **requirement** | Something the system must do or satisfy | "The system must validate all entity IDs on creation" |
| **decision** | A design choice that was made | "We chose YAML over JSON for entity storage" |
| **rationale** | The reasoning behind a decision | "YAML was chosen because it supports comments and is more human-readable" |
| **constraint** | A limitation or boundary condition | "Context packets must not exceed 30KB" |
| **assumption** | Something taken as true without proof | "We assume agents have at least a 32K context window" |
| **risk** | A potential problem or concern | "Sequential ID allocation is unsafe for concurrent access" |
| **question** | An open question or unresolved ambiguity | "Should in-progress documents live in the system or stay local?" |
| **definition** | A term or concept being defined | "A *projection* is a generated view derived from canonical state" |
| **example** | An illustrative example | Code blocks, sample YAML, worked scenarios |
| **alternative** | A rejected or deferred alternative | "We considered UUIDs but rejected them for readability" |
| **narrative** | Contextual prose that connects other fragments | Introductory paragraphs, transitions, background |

### 11.2 Taxonomy evolution

The taxonomy is intentionally small and can be extended. New roles are added to the tool's schema and become available to agents at classification time. Existing classifications are not invalidated when the taxonomy grows.

### 11.3 Confidence

Each classification carries a confidence indicator (high, medium, low) set by the classifying agent. This allows downstream consumers to decide how much to trust a classification. Low-confidence classifications are included in query results but may be filtered by consumers that need precision.

---

## 12. The Concept Model

Concepts are the connective tissue of the document graph. They are what make vertical slicing possible.

### 12.1 What a concept is

A concept is a named idea, pattern, or term that appears across multiple documents. Concepts are not workflow entities — they are not epics, features, or tasks. They are the vocabulary of the design.

Examples:

- "normalization pipeline"
- "formality gradient"
- "context budgeting"
- "refinement chain"
- "lifecycle state machine"
- "TSID13"

### 12.2 How concepts are created

Concepts are extracted by agents during Layer 3 classification. When an agent classifies a fragment, it identifies concepts that the fragment introduces (defines for the first time) or uses (depends on a definition elsewhere).

The tool maintains a concept registry — a deduplicated set of concept names with canonical forms. When an agent contributes a concept, the tool checks for near-duplicates (case-insensitive match, simple normalization) and either merges with an existing concept or creates a new one.

### 12.3 Concept scope

Concepts are corpus-wide. A concept introduced in one document and used in another creates an edge that connects those documents through the graph. This is what enables vertical slicing — following concept edges to find everything related to a single idea.

### 12.4 Extraction scope

Concept extraction is **not restricted** to a predefined project vocabulary.

New design documents introduce new concepts by definition. Implementation documents reference general engineering concepts (concurrency models, error handling patterns, API design principles) that are genuinely useful for cross-referencing. Specialised teams need specialised concepts. Restricting extraction to a predefined vocabulary assumes we know the vocabulary before the work happens — which defeats the purpose of a system that learns from its documents.

The noise filter is the **classifying agent's semantic judgement**, not a vocabulary list. The classification protocol instructs the agent to tag concepts that a section meaningfully introduces or depends on — not every word that appears. A section that mentions "authentication" in passing ("...unlike authentication, which is handled elsewhere...") should not be tagged. A section that discusses authentication design in depth should. The agent can tell the difference. That is the whole point of AI-at-ingest-time.

The **INTRODUCES vs USES** edge distinction provides an additional relevance signal at query time. A vertical slice anchored on INTRODUCES returns the definition and the sections that meaningfully depend on it, not every passing mention. This is the primary mechanism for keeping query results precise even as the concept registry grows.

If noise proves to be a problem at scale, the mitigation path is **query-time filtering** — by scope, by document type, by occurrence count, by the INTRODUCES/USES distinction — not extraction-time restriction. This is consistent with the machine-context design's approach to knowledge scoping: extract broadly, filter at assembly time through context profiles and role scoping.

This is an area where we expect to iterate based on real usage. The design deliberately avoids premature restriction. It is easier to add filtering to a broad registry than to retroactively extract concepts that were excluded by an over-narrow vocabulary.

### 12.5 Concept lifecycle

Concepts do not have an explicit lifecycle with states and transitions. They are **derived from the document graph** and stay current as documents are classified and re-classified.

- When a new document is classified that introduces a concept, the concept appears in the registry.
- When a document is superseded, the concept's introduction point may migrate to the superseding document if it re-introduces the concept. If the superseding document does not mention the concept, the introduction still points to the original document — that is provenance, not staleness.
- A concept with zero remaining references (all introducing and using documents have been removed or re-classified without it) can be pruned from the registry automatically.
- Concepts do not need manual curation, promotion, or retirement. Their presence and connectivity in the graph is their lifecycle.

This is simpler than the knowledge entry lifecycle (`work/design/machine-context-design.md` §9.4) because concepts are lightweight — a name and a set of edges — rather than content-bearing entries with confidence scores and TTLs.

### 12.6 Relationship to knowledge entries

The machine-context design (`work/design/machine-context-design.md` §9) defines knowledge entries as contributed implementation context. Concepts are different: they are extracted from design documents, not contributed by agents during implementation. However, a concept may have a corresponding knowledge entry — the concept "TSID13" in the design corpus may correspond to a knowledge entry about how TSID13 is implemented in code. Linking these is a future integration point.

---

## 13. Storage Model

### 13.1 Index storage

The document index is stored in `.kbz/index/` alongside entity state and context:

```/dev/null/storage-layout.txt#L1-14
.kbz/
├── state/           # existing entities (epics, features, tasks, etc.)
├── context/         # knowledge entries, profiles, sessions (machine-context-design)
├── index/
│   ├── documents/   # per-document index files
│   │   ├── DOC-xxxxx.yaml    # structural index + classifications for one document
│   │   └── DOC-yyyyy.yaml
│   ├── concepts.yaml          # concept registry (deduplicated, corpus-wide)
│   └── graph.yaml             # serialised edge list (cross-document relationships)
└── docs/            # canonical document files
```

### 13.2 Per-document index file

Each indexed document has a corresponding YAML file in `.kbz/index/documents/` containing the structural skeleton and classifications:

```/dev/null/example-index.yaml#L1-47
id: DOC-A1B2C3D4E5F6G
source_path: work/design/document-centric-interface.md
content_hash: sha256:abc123...
indexed_at: 2026-07-18T10:30:00Z
classified_at: 2026-07-18T10:31:00Z    # null if unclassified
classification_state: classified         # indexed | classified

metadata:
  type: design
  status: approved
  date: 2026-03-18
  related:
    - work/design/workflow-design-basis.md
    - work/spec/phase-1-specification.md

sections:
  - path: "1/Purpose"
    level: 2
    title: "Purpose"
    byte_offset: 342
    byte_count: 580
    word_count: 95
    content_hash: sha256:def456...
    role: narrative
    role_confidence: high
    summary: "Defines the purpose and scope of the document-centric interface design"
    concepts_introduced: []
    concepts_used:
      - document-centric-interface
    entity_refs: []

  - path: "7/The Human Interface Contract/7.1/Documents in, documents out"
    level: 3
    title: "Documents in, documents out"
    byte_offset: 5230
    byte_count: 1420
    word_count: 230
    content_hash: sha256:ghi789...
    role: definition
    role_confidence: high
    summary: "Defines the ingest and retrieval contract for documents"
    concepts_introduced:
      - approve-before-canon
    concepts_used:
      - normalization-pipeline
      - canonical-document
    entity_refs: []
```

### 13.3 Concept registry

```/dev/null/concepts-example.yaml#L1-18
concepts:
  - name: normalization-pipeline
    introduced_in:
      - doc: DOC-A1B2C3D4E5F6G
        section: "6.4/AI-Mediated Normalization Pipeline"
    used_in:
      - doc: DOC-B2C3D4E5F6G7H
        section: "6/Document Operations"
      - doc: DOC-C3D4E5F6G7H8I
        section: "3.1/Ingest Pipeline"
    occurrence_count: 7

  - name: formality-gradient
    introduced_in:
      - doc: DOC-A1B2C3D4E5F6G
        section: "5/The Formality Gradient"
    used_in: []
    occurrence_count: 3
```

### 13.4 Serialisation rules

All index files follow the same serialisation rules as entity state files (P1-DEC-008): block style, deterministic field order, UTF-8, LF line endings, trailing newline, no YAML tags or anchors.

Index files are Git-tracked. They change when documents change. They are tool-written and should not be hand-edited.

---

## 14. Relationship to Existing Designs

### 14.1 Document-centric interface

The document-centric interface design (`work/design/document-centric-interface.md`) establishes that:

- Documents are the human interface; entities are the internal model (§3)
- The system may internally index, annotate, and fragment documents for consistency enforcement and context assembly (§8.3)
- Externally, the system always presents whole documents (§8.3)
- Cross-document consistency enforcement and invalidation detection require an index (§8.3–8.4)

This design defines the mechanism for that internal indexing. The document graph is the index. The query operations are how the system uses it.

### 14.2 Machine-context design

The machine-context design (`work/design/machine-context-design.md`) describes context assembly — composing targeted packets for agents from design fragments and implementation knowledge (§8).

The document graph is a **source** for context assembly. When the context assembler needs design fragments for a specific task, it queries the document graph: "give me all requirements and decisions related to FEAT-042" or "give me the specification sections that cover lifecycle transitions." The document graph provides the fragments; the context assembler composes them into a packet within the byte budget.

The document graph does not replace context assembly. It is the backend that makes context assembly precise.

### 14.3 Normalization pipeline

The normalization pipeline (`work/design/workflow-design-basis.md` §6.4) already includes entity extraction (stage 6) — the agent extracts decisions, requirements, entity updates, and cross-document links from approved documents.

Layer 3 classification is an extension of that same stage. The agent is already in the loop, already holding the document in context, already performing entity extraction. Structural classification is one additional structured step during the ingest the agent is already performing — not a separate process.

### 14.4 Formality gradient

The formality gradient (`work/design/document-centric-interface.md` §5) affects structural analysis:

- **Informal documents** (proposals, draft designs): more narrative fragments, fewer requirements, more questions. Classification may be lower confidence.
- **Formal documents** (specifications, implementation plans): more requirements, constraints, and definitions. Classification is higher confidence because the language is more precise.

The system does not vary its classification approach by formality — it uses the same taxonomy everywhere — but downstream consumers may weight results from formal documents more heavily than those from informal ones.

### 14.5 Document type extensibility

The document-centric interface design (`work/design/document-centric-interface.md` §4) defines eight document types: proposal, research report, other report, draft design, design, specification, implementation plan, and user documentation. These are **recognised types** — the system knows their role in the design-to-implementation pipeline, their formality level, their entity extraction patterns, and their review expectations.

The document intelligence layer is **type-agnostic**. All four layers of structural analysis — markdown parsing, pattern-based extraction, AI-assisted classification, and graph construction — operate on any markdown document regardless of its type. The fragment role taxonomy, the concept model, and the entity reference extraction apply equally to a specification, a postmortem, a runbook, or a document type that does not yet exist.

This means:

- **Any document can be added to the system.** The system does not reject documents that do not match a predefined type.
- **Recognised types get pipeline behaviour.** Documents with a known type receive type-specific normalisation, entity extraction patterns, formality treatment, and review expectations. These are the documents that participate in the design-to-implementation pipeline and the refinement chain.
- **Unrecognised or custom-typed documents get structural analysis.** They are parsed, indexed, classified, connected through the graph, and queryable — but they receive no type-specific pipeline behaviour. They are first-class citizens of the document graph without being recognised participants in the pipeline.
- **The set of recognised types is extensible per project.** A project that uses ADRs, postmortems, runbooks, vendor evaluations, or any other document type as standard practice can register those as recognised types with their own pipeline rules. The eight default types are a starting set, not a fixed ceiling.

This design separates two concerns that are easy to conflate:

- **Document intelligence** (structural analysis, classification, graph queries) — works on everything, type-agnostic
- **Pipeline behaviour** (normalisation, entity extraction, review gates, refinement chain position) — works on recognised types, type-specific

The intelligence layer does not need to know about types to be useful. A custom-typed document ingested into the system gets the same structural analysis, the same fragment classification, the same concept extraction, and the same entity reference detection as a specification. It participates in vertical and horizontal slicing. It appears in search results. It connects to other documents through shared concepts and entity references. The only thing it does not get is type-specific pipeline behaviour — and that is added by registering the type, not by changing the intelligence layer.

---

## 15. Cost Model

### 15.1 Ingest cost (Layers 1–2)

Deterministic. Milliseconds per document. Negligible.

### 15.2 Classification cost (Layer 3)

For a 10KB document with 15 sections, the classification prompt is approximately:

- **Input:** the structural skeleton + taxonomy + section content ≈ 12K tokens
- **Output:** classifications for 15 sections ≈ 2K tokens
- **Cost:** pennies at current API prices

This runs once per document version, not once per query. For a corpus of 100 documents, the total classification cost is a few dollars — trivial compared to the token savings from targeted retrieval.

### 15.3 Query cost

Zero AI cost. All queries are served from the cached graph. Cost is CPU time for graph traversal and file I/O for section retrieval — microseconds to milliseconds.

### 15.4 Amortisation

The classification cost is amortised across all subsequent reads. If a section is queried by 8 agents across 20 tasks, the classification ran once and the retrieval ran 160 times from cache. The economics improve with scale — more agents, more tasks, more queries per document.

---

## 16. What This Document Is Not

- **Not an implementation plan.** This defines what to build and why. The implementation plan will define the work breakdown, task sequence, and verification.
- **Not a specification.** The query operation signatures, the exact schema fields, and the validation rules will be pinned down in a specification.
- **Not a vector database design.** This is a structured graph with typed nodes and edges, queried by traversal and filtering. Embedding-based semantic search could be added as an alternative query path in the future, but the graph is the primary structure.
- **Not a replacement for reading documents.** This is a way to find the right document or section to read, and to serve agents only what they need. An agent that needs to understand a full document should read the full document.
- **Not the context assembly system.** The context assembler (`work/design/machine-context-design.md` §8) is a consumer of this graph. This design defines the backend, not the consumer.

---

## 17. Open Questions

### 17.1 Design questions

1. **Cross-document consistency checking.** The `doc_consistency` operation is described but the mechanism is not defined. What does consistency checking actually compare? How are potential inconsistencies surfaced — as warnings, errors, or informational notes? This needs design work before implementation, but does not block the rest of the system.

### 17.2 Implementation questions

The following are implementation questions that should be resolved during implementation planning, not in this design document. They are recorded here for completeness.

- **Graph storage format.** Is a flat YAML edge list sufficient, or does the graph need a more efficient representation? At what corpus size does flat YAML become a bottleneck?
- **Incremental re-classification.** When a document changes, should the entire document be re-classified or only the changed sections? Content hashes per section enable incremental detection, but the classification of one section may depend on context from surrounding sections.
- **Concept deduplication boundaries.** Simple string normalization catches obvious duplicates ("Normalization Pipeline" = "normalization-pipeline") but not synonyms ("normalization pipeline" ≠ "ingest pipeline" even if they refer to the same thing). Is synonym detection worth the complexity, or can it wait until the concept registry is large enough to show the problem?
- **Classification stability across models.** Different LLMs may classify the same fragment differently. Is this a problem in practice? Should the system record which model produced a classification?
- **Index bootstrapping.** When the system is first adopted on a project with existing documents, all documents need Layer 3 classification. Batch classification by a dedicated agent, or incremental classification as documents are touched?

### 17.3 Resolved questions

1. **Scope of concept extraction.** Resolved: concept extraction is not restricted to a predefined vocabulary. The noise filter is the classifying agent's semantic judgement at ingest time, combined with the INTRODUCES/USES distinction at query time. If noise becomes a problem at scale, the mitigation is query-time filtering, not extraction-time restriction. See §12.4.

---

## 18. Phasing

### 18.1 Phase 1

Phase 1 does not build the document intelligence backend. The document storage model (§13) should be designed so that index files can be added alongside documents without structural changes. MCP operation names in the `doc_` namespace should be reserved.

### 18.2 Phase 2

Phase 2 builds the core:

- Layer 1: markdown parsing and structural skeleton
- Layer 2: pattern-based extraction (entity references, cross-document links, front matter)
- Layer 3: AI-assisted classification protocol (ingest, classify, validate, store)
- Layer 4: document graph construction and core query operations
- The concept model and concept registry
- Integration with context assembly (`work/design/machine-context-design.md` §8)
- The `doc_` MCP operations (§9)

### 18.3 Phase 3 and beyond

- Refinement chain analysis and full provenance tracing
- Cross-document consistency checking
- Argument structure detection
- Embedding-based semantic search as an alternative query path (if graph queries prove insufficient at scale)
- Concept synonym detection
- Automated re-classification on document change (without explicit agent intervention)
- Visualisation of the document graph for human consumption

---

## 19. Summary

Design documents have structure that is intermediate between source code and free text. That structure — section hierarchies, entity references, conventional section roles, refinement chains, shared concepts — can be exploited to provide code-intelligence-like capabilities for the document corpus.

The structural analysis backend operates in four layers: a deterministic structural skeleton (markdown parsing), deterministic pattern-based extraction (entity references, links, conventions), AI-assisted classification (fragment roles, concepts, summaries), and a persistent queryable graph. The first two layers run on every document change. The third runs when an agent curates the document. The fourth is always available at whatever level of analysis exists.

The system follows a principle that mirrors the rest of Kanbanzai: **the tool is a database with a schema and a query engine; agents are the curators who populate it with intelligent classifications.** The tool never calls an LLM. It provides structure. Agents provide intelligence. The economics work because classification runs once per document version and queries run from cache — the cost is amortised across all future reads by all agents.

This design is the mechanism behind the "internal fragmentation" promised by the document-centric interface design and the backend that makes context assembly precise rather than brute-force.