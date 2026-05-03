# Design: docs-memory-mcp — Document Intelligence MCP Server

| Field | Value |
|-------|-------|
| Date | 2025-07-14 |
| Status | Draft |
| Author | Research task, with human review |
| Based on | `work/research/document-retrieval-for-ai-agents.md` |
| Informed by | `work/design/document-intelligence-design.md`, codebase-memory-mcp architecture |
| Related | `work/design/document-centric-interface.md`, `work/design/machine-context-design.md` |

---

## 1. Purpose

This document defines the design for **docs-memory-mcp** — a standalone MCP server
that provides fast, token-efficient retrieval of English-language project documentation
for AI agents.

The server does for design documents, specifications, research notes, and plans what
[codebase-memory-mcp](https://github.com/DeusData/codebase-memory-mcp) does for source
code: it builds a persistent, queryable graph from a corpus of Markdown files, then
exposes structured search and retrieval tools over the Model Context Protocol.

### 1.1 Why a Standalone Server

Kanbanzai's `doc_intel` system (defined in `work/design/document-intelligence-design.md`)
provides section-level indexing and structural search. It works well for targeted queries
within the Kanbanzai workflow. But it has three structural limitations:

1. **Tight coupling.** `doc_intel` lives inside `kanbanzai serve`. Other projects cannot
   use it without adopting the full Kanbanzai workflow system.
2. **No full-text search.** Agents fall back to `grep` for any query that is not by
   entity ID, concept, or role. Grep returns raw line matches without section context,
   relevance ranking, or token-efficient output.
3. **YAML-based storage.** The graph (12K+ edges) and document indexes (280+ files) are
   stored as YAML files scanned linearly on every query. This is adequate at current
   scale but architecturally unable to support full-text search or efficient graph
   traversal.

A standalone server solves all three: it is reusable, it provides full-text search with
BM25 ranking, and it uses SQLite for indexed storage and FTS5 for text search.

### 1.2 Scope

**In scope:**

- Indexing Markdown documents into a SQLite-backed graph
- Full-text search over section content with BM25 ranking
- Section-level retrieval with progressive disclosure (outline → summary → full)
- Structural search by entity reference, concept, and fragment role
- Cross-reference traversal and impact analysis
- Classification storage (agent-provided, not agent-driven)
- A small, focused MCP tool surface (10–12 tools)

**Out of scope:**

- Workflow orchestration, lifecycle management, or entity tracking
  (these remain in Kanbanzai)
- Embedded LLM calls or embedding generation
  (the server is a database; agents provide intelligence)
- Non-Markdown document formats (PDF, DOCX, HTML)
- Real-time file watching
  (indexing is triggered explicitly or on server start)
- Kanbanzai-specific concepts (plans, features, tasks)
  (entity references are extracted by pattern, not by Kanbanzai semantics)

### 1.3 Relationship to Kanbanzai

docs-memory-mcp is a **dependency of Kanbanzai, not a component of it**. Kanbanzai's
context assembly pipeline would call docs-memory-mcp tools instead of (or in addition
to) its internal `doc_intel` tools. The integration surface is the MCP tool interface —
the same interface available to any MCP client.

Kanbanzai remains responsible for:

- Document lifecycle (registration, approval, supersession)
- Workflow-aware context assembly (which documents to surface for a task)
- Knowledge management (the `knowledge` system is orthogonal)
- Classification orchestration (deciding when to classify documents)

docs-memory-mcp is responsible for:

- Indexing Markdown files into a queryable graph
- Answering search and retrieval queries efficiently
- Storing classification metadata provided by agents
- Maintaining graph integrity across index updates

---

## 2. Design Principles

### DP-1: The Tool Is a Database; Agents Are the Intelligence

The server never calls an LLM. It never generates embeddings. It never needs API keys.
It parses structure deterministically, stores classifications provided by agents, and
answers queries from its index. This is the same principle as codebase-memory-mcp and
Kanbanzai's `doc_intel` (design principle 1 in `work/design/document-intelligence-design.md`).

*Why:* An MCP server that requires an LLM API key has deployment friction, variable
costs, and non-deterministic behaviour. A database has none of these.

### DP-2: Graph-Augmented Search, Not Pure Graph

The core retrieval pattern is: **full-text search for recall, graph structure for
precision and ranking**. This is the hybrid approach proven effective in the information
retrieval literature (BEIR benchmark, Thakur et al. 2021) and demonstrated in practice
by codebase-memory-mcp's "grep → graph enrichment → structural ranking" pipeline.

Neither pure keyword search (grep) nor pure graph traversal is sufficient alone:

- Keyword search finds mentions but cannot rank by structural importance or navigate
  relationships.
- Graph traversal answers structural queries ("what references this section?") but
  cannot answer content queries ("sections about authentication").

The combination answers both.

*Why:* The research report (`work/research/document-retrieval-for-ai-agents.md` §2c)
found that hybrid retrieval consistently outperforms either approach alone by 15–20% on
technical corpora.

### DP-3: Progressive Disclosure — Outlines Before Content

Every retrieval tool supports multiple output levels. Agents start cheap (outlines,
metadata) and drill deeper (summaries, full content) only when needed. This is the
single most important token-saving pattern.

Three levels:

| Level | What's returned | Token cost |
|-------|----------------|-----------|
| **outline** | Section paths, titles, word counts, roles | ~10–30 tokens per section |
| **summary** | Outline + first paragraph or agent-provided summary | ~50–150 tokens per section |
| **full** | Complete section content | Variable (100–2000+ tokens) |

*Why:* Anthropic's context engineering research (2025) found that 15–40% context window
utilisation is optimal. Returning full content by default wastes the budget.

### DP-4: Single Binary, Zero Dependencies

The server compiles to a single binary with SQLite statically linked. No external
database, no Docker container, no API keys, no runtime dependencies. Download, run,
done.

*Why:* codebase-memory-mcp demonstrated that this deployment model is critical for
adoption. A tool that requires `docker compose up` before first use will not be used.

### DP-5: Documents Are Somebody Else's Problem

The server indexes files on disk. It does not manage document lifecycle, versioning,
approval workflows, or naming conventions. It treats the document directory as a
read-only input and builds its index from whatever Markdown files it finds.

*Why:* Scope discipline. The Monolith Creep anti-pattern
(`work/research/document-retrieval-for-ai-agents.md` §Anti-Patterns) is the primary
risk for this project. The boundary is: this tool does retrieval, not management.

### DP-6: Deterministic Layers Are Always Current; Classifications Are Eventually Consistent

Structural parsing (headings, links, entity references) runs on every index operation
and is always correct with respect to the current file content. Agent-provided
classifications (roles, concepts, summaries) are stored metadata that may lag behind
file changes. The system tracks this via content hashes and reports staleness, but does
not block queries on stale classifications.

*Why:* Same principle as doc_intel (design principle 5). Demanding that classifications
are always current would either require embedded LLM calls (violating DP-1) or block
queries on unclassified content (violating DP-3's progressive usefulness).

---

## 3. Architecture

### 3.1 Components

```/dev/null/architecture.txt#L1-18
┌──────────────────────────────────────────────────────┐
│                 MCP Client (Agent)                    │
│          (Claude, Cursor, any MCP client)             │
└──────────────────┬───────────────────────────────────┘
                   │ JSON-RPC 2.0 over stdio
┌──────────────────▼───────────────────────────────────┐
│              docs-memory-mcp server                   │
│                                                       │
│  ┌─────────────┐  ┌──────────────┐  ┌─────────────┐ │
│  │  MCP Layer   │  │   Indexing    │  │   Query     │ │
│  │  (tool       │  │   Pipeline   │  │   Engine    │ │
│  │  dispatch)   │  │              │  │             │ │
│  └──────┬──────┘  └──────┬───────┘  └──────┬──────┘ │
│         │                │                  │         │
│  ┌──────▼──────────────────────────────────▼──────┐  │
│  │                 SQLite Store                     │  │
│  │  (graph tables + FTS5 full-text index)          │  │
│  └─────────────────────────────────────────────────┘  │
└───────────────────────────────────────────────────────┘
                   │
                   │ reads files from disk
                   ▼
┌──────────────────────────────────────────────────────┐
│              Document Directory                       │
│   work/design/*.md  work/spec/*.md  docs/*.md  ...   │
└──────────────────────────────────────────────────────┘
```

### 3.2 SQLite Store

SQLite is the storage backend, following codebase-memory-mcp's proven approach. The
database uses WAL mode for crash safety and concurrent reads.

**Database location:** `~/.cache/docs-memory-mcp/<project-hash>.db`
(project-hash derived from the indexed directory's absolute path)

**Why SQLite over a purpose-built graph database:**

- Zero deployment overhead (statically linked)
- FTS5 extension provides production-quality BM25 ranking out of the box
- Relational tables with indexes simulate a property graph efficiently for the
  query patterns this system needs (adjacency lookups, BFS traversal, filtered search)
- codebase-memory-mcp proved this works at scale (projects with 100K+ nodes)

### 3.3 Indexing Pipeline

The pipeline runs 5 sequential passes on each Markdown file:

| Pass | Name | What it produces | Deterministic? |
|------|------|-----------------|---------------|
| 1 | **Structure** | Document and Section nodes with byte offsets, word counts, content hashes | Yes |
| 2 | **Entities** | EntityRef nodes from regex patterns (e.g., `FEAT-xxx`, `FR-001`, `DEC-xxx`) | Yes |
| 3 | **Links** | LINKS_TO edges from Markdown links and backtick-path references | Yes |
| 4 | **Roles** | Conventional role assignments from heading keywords (e.g., "Requirements" → requirement) | Yes (heuristic) |
| 5 | **Front matter** | Document metadata from YAML front matter or leading table (type, status, date, author) | Yes |

Agent-provided classifications (Layer 3 in `doc_intel` terminology) are stored via the
`classify` tool, not as part of the indexing pipeline.

**Re-indexing strategy:** Each document's content hash is stored in the database. On
re-index, only files whose hash has changed are re-processed. Classifications are
preserved across re-indexes if the section's content hash has not changed; otherwise
they are marked stale.

### 3.4 When Indexing Runs

- **On server start:** if the index database does not exist, or if the `--reindex` flag
  is set, the full pipeline runs on all Markdown files in the document directory.
- **On `index_documents` tool call:** an agent explicitly triggers re-indexing. Supports
  a path filter to re-index a subset of files.
- **Not on file change.** There is no file watcher. This is a deliberate simplification
  — the agent or user triggers re-indexing when they know documents have changed.

---

## 4. Graph Model

### 4.1 Node Types

| Node type | Source | Description |
|-----------|--------|-------------|
| **Project** | Pipeline pass 1 | Root node for the indexed directory |
| **Document** | Pipeline pass 1 | A complete Markdown file with metadata (type, status, date, author) |
| **Section** | Pipeline pass 1 | A heading-delimited section with level, title, path, byte offset, word count, content hash |
| **EntityRef** | Pipeline pass 2 | A reference to an external entity by ID pattern within a section |
| **Concept** | Agent classification | A domain concept that appears across documents (introduced or used) |

Compared to `doc_intel`'s graph schema (§7 of the document intelligence design), this
model omits the **Fragment** and **Question** node types. Fragments are subsumed by
classified Sections (a Section with a role assignment *is* a fragment). Questions are a
workflow concern better handled by the consuming system.

### 4.2 Edge Types

| Edge type | From → To | Source | Description |
|-----------|-----------|--------|-------------|
| **CONTAINS** | Project → Document, Document → Section, Section → Section | Pass 1 | Hierarchical containment |
| **LINKS_TO** | Section → Section, Section → Document | Pass 3 | Explicit Markdown link or backtick-path reference |
| **MENTIONS** | Section → EntityRef | Pass 2 | Section contains an entity reference |
| **INTRODUCES** | Section → Concept | Classification | Section where a concept is first defined |
| **USES** | Section → Concept | Classification | Section that references a concept defined elsewhere |
| **NEXT_IN** | Document → Document | Metadata | Refinement chain ordering (design → spec → plan) inferred from metadata |

### 4.3 Example

```/dev/null/graph-example.txt#L1-15
Project: "kanbanzai-docs"
  ├── CONTAINS → Document: "work/design/workflow-design-basis.md" (type: design)
  │     ├── CONTAINS → Section: §6.4 "AI-Mediated Normalization Pipeline" (430 words)
  │     │     ├── MENTIONS → EntityRef: "FEAT-01KMKRQRRX3CC"
  │     │     ├── INTRODUCES → Concept: "normalization-pipeline"
  │     │     └── LINKS_TO → Section: "work/design/document-centric-interface.md" §7.1
  │     └── CONTAINS → Section: §8.5 "Internal Fragmentation" (280 words)
  │           └── USES → Concept: "normalization-pipeline"
  │
  └── CONTAINS → Document: "work/spec/phase-1-specification.md" (type: specification)
        ├── NEXT_IN ← Document: "work/design/workflow-design-basis.md"
        └── CONTAINS → Section: §6 "Document Operations" (1200 words)
              ├── MENTIONS → EntityRef: "FR-042"
              └── USES → Concept: "normalization-pipeline"
```

---

## 5. Database Schema

### 5.1 Core Tables

```/dev/null/schema.sql#L1-62
-- Nodes
CREATE TABLE documents (
    id          INTEGER PRIMARY KEY,
    path        TEXT NOT NULL UNIQUE,          -- relative file path
    title       TEXT,                          -- first heading or filename
    doc_type    TEXT,                          -- design, specification, dev-plan, research, report
    status      TEXT,                          -- draft, approved, superseded (from front matter)
    content_hash TEXT NOT NULL,                -- SHA-256 of file content
    word_count  INTEGER NOT NULL,
    indexed_at  TEXT NOT NULL,                 -- ISO 8601
    metadata    TEXT                           -- JSON blob for front matter fields
);

CREATE TABLE sections (
    id           INTEGER PRIMARY KEY,
    document_id  INTEGER NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    parent_id    INTEGER REFERENCES sections(id) ON DELETE CASCADE,
    path         TEXT NOT NULL,                -- dotted path e.g. "1.2.3"
    level        INTEGER NOT NULL,             -- heading level 1-6
    title        TEXT NOT NULL,
    byte_offset  INTEGER NOT NULL,
    byte_count   INTEGER NOT NULL,
    word_count   INTEGER NOT NULL,
    content_hash TEXT NOT NULL,
    -- Classification fields (nullable, agent-provided)
    role         TEXT,                         -- requirement, decision, rationale, etc.
    role_confidence TEXT,                      -- high, medium, low
    summary      TEXT,                         -- agent-provided summary
    classified_at TEXT,                        -- when classification was applied
    classified_by TEXT,                        -- model name + version
    UNIQUE(document_id, path)
);

CREATE TABLE entity_refs (
    id          INTEGER PRIMARY KEY,
    section_id  INTEGER NOT NULL REFERENCES sections(id) ON DELETE CASCADE,
    entity_id   TEXT NOT NULL,                -- e.g. "FEAT-01KMKRQRRX3CC", "FR-003"
    entity_type TEXT                          -- feat, task, bug, fr, nfr, dec (inferred from prefix)
);

CREATE TABLE concepts (
    id          INTEGER PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,          -- normalised: lowercase, hyphenated
    aliases     TEXT                           -- JSON array of alternative names
);

CREATE TABLE concept_refs (
    concept_id  INTEGER NOT NULL REFERENCES concepts(id) ON DELETE CASCADE,
    section_id  INTEGER NOT NULL REFERENCES sections(id) ON DELETE CASCADE,
    rel_type    TEXT NOT NULL,                 -- 'introduces' or 'uses'
    PRIMARY KEY (concept_id, section_id, rel_type)
);

-- Edges (for cross-document/cross-section links not captured by foreign keys)
CREATE TABLE edges (
    id        INTEGER PRIMARY KEY,
    from_id   INTEGER NOT NULL,
    from_type TEXT NOT NULL,                   -- 'document' or 'section'
    to_id     INTEGER NOT NULL,
    to_type   TEXT NOT NULL,
    edge_type TEXT NOT NULL,                   -- LINKS_TO, NEXT_IN
    metadata  TEXT                             -- JSON for edge properties
);
```

### 5.2 Full-Text Search Index

```/dev/null/fts.sql#L1-14
-- FTS5 virtual table over section content
CREATE VIRTUAL TABLE sections_fts USING fts5(
    title,
    content,
    content='',                               -- contentless: we store content in files, not SQLite
    tokenize='porter unicode61'               -- Porter stemming + Unicode normalisation
);

-- Populated during indexing: each section's heading + body text
-- Queried via: SELECT ... FROM sections_fts WHERE sections_fts MATCH ?
--              ORDER BY bm25(sections_fts)
-- Returns rowids that join back to the sections table
```

**Why contentless FTS5:** The actual section content lives in the Markdown files on disk.
Duplicating it into SQLite would double storage. The FTS5 index stores only the token
positions needed for BM25 ranking. Section content is retrieved from the original file
via byte offset when the agent requests `full` output.

### 5.3 Key Indexes

```/dev/null/indexes.sql#L1-10
CREATE INDEX idx_sections_document ON sections(document_id);
CREATE INDEX idx_sections_role ON sections(role) WHERE role IS NOT NULL;
CREATE INDEX idx_entity_refs_entity ON entity_refs(entity_id);
CREATE INDEX idx_entity_refs_section ON entity_refs(section_id);
CREATE INDEX idx_concept_refs_concept ON concept_refs(concept_id);
CREATE INDEX idx_concept_refs_section ON concept_refs(section_id);
CREATE INDEX idx_edges_from ON edges(from_id, from_type);
CREATE INDEX idx_edges_to ON edges(to_id, to_type);
CREATE INDEX idx_edges_type ON edges(edge_type);
```

---

## 6. MCP Tool Interface

The server exposes 11 tools over JSON-RPC 2.0 on stdio.

### 6.1 Indexing Tools

#### `index_documents`

Index or re-index Markdown files in the document directory.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | No | Subdirectory to index (default: entire document root) |
| `force` | bool | No | Re-index all files, ignoring content hash (default: false) |

Returns: count of files indexed, count unchanged, count of errors.

#### `index_status`

Check the state of the index.

Returns: total documents, total sections, total edges, last indexed timestamp,
list of stale documents (file hash differs from indexed hash).

### 6.2 Search Tools

#### `search`

Full-text search over section content with BM25 ranking.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | Yes | Search query (supports FTS5 syntax: AND, OR, NOT, phrase "...") |
| `mode` | string | No | Output level: `outline` (default), `summary`, `full` |
| `limit` | int | No | Max results (default: 10) |
| `doc_type` | string | No | Filter by document type (design, specification, etc.) |
| `role` | string | No | Filter by classified role (requirement, decision, etc.) |

Returns: ranked list of matching sections with document path, section path,
title, BM25 score, and content at the requested output level.

**This is the primary tool.** It replaces grep for document queries with
section-aware, ranked results.

#### `find`

Structural search — find sections by entity reference, concept, or role.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `entity_id` | string | * | Find sections mentioning this entity |
| `concept` | string | * | Find sections introducing or using this concept |
| `role` | string | * | Find sections classified with this role |
| `scope` | string | No | Limit search to a document path or glob |
| `mode` | string | No | Output level: `outline` (default), `summary`, `full` |

*Exactly one of `entity_id`, `concept`, or `role` is required.*

Returns: list of matching sections grouped by document.

### 6.3 Document Tools

#### `get_outline`

Return the structural skeleton of a document.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | Yes | Document file path (relative to document root) |

Returns: hierarchical section tree with paths, titles, levels, word counts,
roles (if classified), and entity reference counts per section.

**This is the recommended entry point** before reading any document. It costs
~200–800 tokens vs. 5,000–20,000 for a full read.

#### `get_section`

Return a specific section's content.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | Yes | Document file path |
| `section` | string | Yes | Section path (e.g., "2.3.1") |

Returns: section metadata + raw content read from the original file using
byte offset and byte count.

#### `get_document`

Return full document content or metadata.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | Yes | Document file path |
| `mode` | string | No | `metadata` (default), `summary` (outline + first section), `full` |

Returns: document metadata from the index, optionally with content.

### 6.4 Graph Tools

#### `trace_references`

BFS traversal of cross-references from a starting section or document.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | Yes | Starting document path |
| `section` | string | No | Starting section path (if omitted, traces from document level) |
| `depth` | int | No | Max traversal depth (default: 2, max: 5) |
| `direction` | string | No | `outbound` (default), `inbound`, `both` |

Returns: tree of connected sections/documents with edge types and depths.

#### `impact`

What would be affected if a section or document changes?

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | Yes | Document file path |
| `section` | string | No | Section path (if omitted, analyses whole document) |

Returns: list of sections/documents that reference, depend on, or use
concepts from the target, with hop distance.

#### `query_graph`

Execute a structured query over the document graph.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | Yes | Cypher-subset query |
| `limit` | int | No | Max results (default: 100) |

Supports: MATCH, WHERE (with CONTAINS, regex, comparisons), RETURN (with
COUNT, DISTINCT), ORDER BY, LIMIT.

### 6.5 Classification Tools

#### `classify`

Store agent-provided classifications for sections of a document.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | Yes | Document file path |
| `content_hash` | string | Yes | Document content hash (must match current index) |
| `model` | string | Yes | Classifying model name + version |
| `classifications` | array | Yes | Array of `{section, role, confidence, summary, concepts_introduced, concepts_used}` |

Returns: count of sections classified, count of concepts created/updated.

**Content hash requirement:** prevents classifying a stale version of the
document. If the file has changed since indexing, the agent must trigger
`index_documents` first.

#### `get_schema`

Return index statistics and configuration.

Returns: total documents, sections, concepts, edges; document type
distribution; role distribution; concept frequency list; stale document count.

---

## 7. The Search Pipeline

This section describes how the `search` tool processes a query — the core
retrieval path.

### 7.1 Query Flow

```/dev/null/search-flow.txt#L1-14
1. PARSE query string (FTS5 syntax: terms, phrases, boolean operators)
2. EXECUTE FTS5 MATCH against sections_fts
   → candidate set with BM25 scores
3. APPLY filters (doc_type, role) via JOIN to sections/documents tables
   → filtered candidate set
4. ENRICH each result with graph metadata:
   - Section role and confidence (from sections table)
   - Entity reference count (from entity_refs table)
   - Concept associations (from concept_refs table)
   - In-degree: how many other sections link to this one (from edges table)
5. RE-RANK by composite score:
   score = BM25_score × (1 + 0.1 × in_degree) × role_boost
   where role_boost = 1.5 for requirements/decisions, 1.0 otherwise
6. TRUNCATE to limit
7. FORMAT output at requested mode level (outline/summary/full)
```

### 7.2 Composite Scoring Rationale

Pure BM25 ranks by text relevance. The composite score adds two structural signals:

- **In-degree boost:** sections referenced by many other sections are structurally
  important. A 0.1× multiplier per inbound link gives a mild preference without
  overwhelming text relevance.
- **Role boost:** requirements and decisions are the most actionable section types.
  A 1.5× multiplier surfaces them above narrative and context sections of equal
  text relevance.

These weights are initial values. They should be tuned based on real query evaluation.

---

## 8. Classification Protocol

### 8.1 Workflow

Classification follows the same "AI at ingest time, not query time" principle from
Kanbanzai's document intelligence design:

1. Agent calls `get_outline` to see the document structure.
2. Agent reads sections of interest (via `get_section` or the original file).
3. Agent decides on role, confidence, summary, and concept tags for each section.
4. Agent calls `classify` with the full set of classifications.
5. Server validates content hash, validates roles against taxonomy, stores results.
6. Subsequent searches benefit from role filters, concept search, and summaries.

### 8.2 Role Taxonomy

The same 11 roles from Kanbanzai's fragment taxonomy, plus one:

| Role | Description |
|------|-------------|
| `requirement` | A testable functional or non-functional requirement |
| `decision` | An architectural or design decision |
| `rationale` | Reasoning that supports a decision |
| `constraint` | A hard limitation or boundary condition |
| `assumption` | Something taken as given without proof |
| `risk` | An identified risk or uncertainty |
| `question` | An open question or unresolved issue |
| `definition` | A term or concept definition |
| `example` | An illustrative example |
| `alternative` | A considered-but-rejected alternative |
| `narrative` | Contextual or explanatory prose |
| `overview` | High-level summary or introduction |

### 8.3 Concept Management

Concepts are normalised strings (lowercase, hyphenated). The `classify` tool creates
concept nodes and concept_ref edges automatically from the `concepts_introduced` and
`concepts_used` fields in classifications.

Concept aliases are supported: an agent can declare that "rate-limiting" and "throttling"
refer to the same concept. Alias resolution is applied during `find(concept: ...)` queries.

---

## 9. Entity Reference Extraction

### 9.1 Default Patterns

The indexing pipeline extracts entity references using configurable regex patterns. The
defaults match common project patterns:

| Pattern | Entity type | Example |
|---------|-------------|---------|
| `FEAT-[A-Z0-9]+` | feature | FEAT-01KMKRQRRX3CC |
| `TASK-[A-Z0-9]+` | task | TASK-01KN3V9C5E12 |
| `BUG-[A-Z0-9]+` | bug | BUG-01KN5X2M8P4Q |
| `DEC-[0-9]+` | decision | DEC-042 |
| `FR-[0-9]+` | functional requirement | FR-003 |
| `NFR-[0-9]+` | non-functional requirement | NFR-012 |
| `[A-Z][0-9]+-[A-Z]+-[0-9]+` | plan-scoped ID | P4-DEC-008 |

### 9.2 Custom Patterns

The server accepts a configuration file (YAML or TOML) that allows projects to define
additional entity reference patterns. This keeps the tool general-purpose while
supporting project-specific conventions.

---

## 10. Configuration

### 10.1 Server Configuration

```/dev/null/config-example.yaml#L1-23
# docs-memory-mcp.yaml (in project root or ~/.config/docs-memory-mcp/)
document_root: work/           # directory to index (relative to project root)
include:
  - "**/*.md"                  # glob patterns for files to index
exclude:
  - "**/node_modules/**"
  - "**/.git/**"

entity_patterns:               # additional entity reference patterns
  - pattern: "EPIC-[A-Z0-9]+"
    type: epic
  - pattern: "US-[0-9]+"
    type: user-story

front_matter:                  # how to extract document metadata
  type_field: type             # YAML front matter field for document type
  status_field: status
  date_field: date

cache_dir: ~/.cache/docs-memory-mcp   # where to store the SQLite database
```

### 10.2 Startup

```/dev/null/startup.txt#L1-4
$ docs-memory-mcp serve                          # start MCP server on stdio
$ docs-memory-mcp serve --config ./custom.yaml    # custom config
$ docs-memory-mcp index                           # index without starting server (CLI mode)
$ docs-memory-mcp status                          # print index statistics
```

---

## 11. Implementation Language and Dependencies

### 11.1 Language Choice: Go

Go is the recommended implementation language for the following reasons:

- Kanbanzai is written in Go; shared knowledge and tooling
- Single-binary compilation with static linking (CGo for SQLite)
- Strong concurrency primitives for parallel indexing
- Mature SQLite bindings (`modernc.org/sqlite` for pure Go, or `mattn/go-sqlite3` for CGo)
- codebase-memory-mcp is written in C; Go offers comparable performance with safer
  memory management and faster development velocity

### 11.2 Key Dependencies

| Dependency | Purpose |
|------------|---------|
| `modernc.org/sqlite` or `mattn/go-sqlite3` | SQLite with FTS5 |
| `github.com/yuin/goldmark` | Markdown parsing (headings, links, front matter) |
| Standard library | JSON-RPC 2.0, regex, file I/O |

The dependency surface should be minimal. No web frameworks, no ORM, no embedding
libraries.

---

## 12. What This Design Is Not

1. **Not a specification.** This document describes the design — the what and why. The
   specification will distill this into testable requirements.

2. **Not an implementation plan.** Technology choices are stated at the level needed to
   evaluate feasibility. Detailed task breakdowns belong in the implementation plan.

3. **Not a replacement for Kanbanzai.** This server handles document retrieval. Kanbanzai
   handles workflow, lifecycle, context assembly, and orchestration. They are complementary.

4. **Not an embedding/vector system.** If future evaluation shows that BM25 + structural
   ranking is insufficient, embedding-based retrieval can be added as a separate component.
   The current design does not preclude this but does not include it.

---

## 13. Open Questions

### 13.1 Design Questions

1. **Should the Cypher query engine be included in v1?** It is the most complex component
   and may not be needed if `search`, `find`, `trace_references`, and `impact` cover the
   common query patterns. Deferring it to v2 would reduce initial scope significantly.

2. **How should concept alias discovery work?** Manual declaration (via `classify`) is
   straightforward but requires agent effort. Automatic alias detection (via co-occurrence
   analysis or embedding similarity) is more powerful but violates DP-1 (no embedded AI).
   A middle ground: the server detects potential aliases (high co-occurrence, similar
   contexts) and surfaces them as suggestions for agent confirmation.

3. **Should the server support multiple projects in one instance?** codebase-memory-mcp
   does (one SQLite DB per project). The same pattern would work here, but adds complexity
   to the tool interface (project parameter on every call).

### 13.2 Implementation Questions

4. **Pure Go SQLite or CGo?** `modernc.org/sqlite` is pure Go (easier cross-compilation)
   but 2–3× slower than `mattn/go-sqlite3` (CGo). For index sizes under 10K documents,
   the difference is negligible.

5. **How should the FTS5 tokeniser be configured?** Porter stemming handles English
   morphology (running → run) but may over-stem domain terms. A custom tokeniser could
   preserve domain vocabulary while stemming common words.

6. **What is the re-indexing story for large corpora?** At 280 documents, full re-indexing
   takes seconds. At 10K documents, incremental re-indexing becomes important. The
   content-hash-based change detection handles this, but the FTS5 index may need special
   handling for contentless mode updates.

---

## 14. Relationship to Kanbanzai's doc_intel

### 14.1 Feature Comparison

| Capability | doc_intel (current) | docs-memory-mcp (proposed) |
|-----------|--------------------|-----------------------------|
| Section-level indexing | ✅ YAML files | ✅ SQLite tables |
| Section-level retrieval | ✅ Byte-precise | ✅ Byte-precise |
| Full-text search | ❌ | ✅ BM25 via FTS5 |
| Entity reference search | ✅ Linear scan | ✅ Indexed lookup |
| Concept search | ✅ (empty registry) | ✅ With alias support |
| Role-based search | ✅ Linear scan | ✅ Indexed lookup |
| Graph traversal | ✅ Linear scan of YAML | ✅ Indexed edge tables |
| Impact analysis | ✅ Linear scan | ✅ BFS with indexed edges |
| Refinement chain tracing | ✅ | ✅ Via NEXT_IN edges |
| Progressive disclosure | ✅ guide → section | ✅ outline/summary/full modes |
| Agent classification | ✅ (never used) | ✅ Same protocol |
| Standalone deployment | ❌ Part of Kanbanzai | ✅ Separate binary |
| Cypher queries | ❌ | ✅ (proposed, may defer) |
| Custom entity patterns | ❌ Hardcoded | ✅ Configurable |
| Incremental re-indexing | ❌ Full re-parse | ✅ Content-hash-based |

### 14.2 Migration Path

If docs-memory-mcp is built, Kanbanzai's integration path is:

1. Configure docs-memory-mcp as an MCP server in the agent's MCP config
2. Update Kanbanzai's context assembly pipeline to call docs-memory-mcp tools
   instead of internal doc_intel methods
3. Retain doc_intel for Kanbanzai-specific operations (document lifecycle
   awareness, workflow-gated queries) or deprecate it entirely
4. Migration is non-breaking — both systems can run in parallel during transition

---

## 15. Phasing

### Phase 1: Core Retrieval (MVP)

- Markdown indexing pipeline (passes 1–4: structure, entities, links, roles)
- SQLite store with FTS5
- 6 tools: `index_documents`, `index_status`, `search`, `get_outline`, `get_section`, `get_document`
- Configuration file support
- CLI mode: `index`, `status`, `serve`

**Deliverable:** A working MCP server that indexes Markdown files and provides
full-text search with section-level results. An agent can search, browse outlines,
and read targeted sections.

### Phase 2: Graph and Classification

- Classification tool and protocol
- Concept management with aliases
- `find` tool (entity, concept, role search)
- `trace_references` and `impact` tools
- Front matter extraction (pass 5)
- Incremental re-indexing

**Deliverable:** The full structural query suite. Agents can classify documents,
search by concept, trace references, and assess change impact.

### Phase 3: Advanced (If Warranted)

- Cypher-subset query engine
- Community detection (Louvain) for document clustering
- `get_schema` with frequency distributions
- Multi-project support
- Embedding-based hybrid retrieval (if BM25 proves insufficient)

**Deliverable:** Advanced query capabilities for large corpora and complex
cross-document analysis.

---

## 16. Summary

docs-memory-mcp is a focused, standalone tool for one job: helping AI agents find the
right section of the right document without reading everything. It follows proven
patterns from codebase-memory-mcp (SQLite graph, progressive disclosure, structural
ranking) adapted for English-language Markdown documents (BM25 full-text search,
heading-based chunking, classification taxonomy).

The system is designed to be useful immediately (Phase 1: search + outline + section)
and to grow incrementally (Phase 2: graph queries, Phase 3: advanced analysis) without
requiring any component that is not yet proven.

The sharpest design constraint is DP-5: **documents are somebody else's problem.** This
server indexes and retrieves. It does not manage, version, approve, or orchestrate.
That boundary is what keeps it focused, reusable, and maintainable.
