# Design: Document Intelligence Enhancement

| Field | Value |
|-------|-------|
| Date | 2025-07-14 |
| Status | Draft |
| Author | Research task, with human review |
| Based on | `work/research/document-retrieval-for-ai-agents.md` (Option A) |
| Extends | `work/design/document-intelligence-design.md` |
| Related | `work/design/docs-memory-mcp-design.md` (Option B — evaluate after this) |

---

## 1. Purpose

This document defines five enhancements to Kanbanzai's existing document intelligence
system that collectively close the largest capability gaps identified in the research
report (`work/research/document-retrieval-for-ai-agents.md` §3d, §5).

The enhancements are:

1. **Full-text search** via SQLite FTS5 — a new `search` action on the `doc_intel` tool
2. **SQLite-backed graph and index storage** — replacing the flat YAML graph file and
   eliminating linear scans
3. **Batch classification protocol** — a workflow for populating Layer 3 across the
   existing document corpus
4. **Knowledge ↔ document cross-queries** — wiring the two systems together
5. **Concept alias resolution** — implementing the declared-but-unused alias mechanism

Each enhancement delivers value independently. They are ordered by impact: full-text
search closes the single largest gap (no way to search document content without grep),
SQLite migration eliminates the scaling bottleneck, and the remaining three activate
capabilities that are already designed and partially implemented but have no data.

### 1.1 Scope

**In scope:**

- New `search` action on `doc_intel`
- SQLite database for graph edges, entity references, and FTS5 index
- Batch classification workflow via existing `classify` action
- New cross-system query capabilities on `doc_intel` and `knowledge` tools
- Concept alias storage and resolution in `FindByConcept`

**Out of scope:**

- New MCP tools or servers (this enhances existing tools)
- Embedding-based semantic search (evaluate after these improvements)
- Changes to the MCP tool interface beyond adding new actions/parameters
- Changes to document lifecycle management (`doc` tool)
- Paragraph-level granularity (sections remain the fundamental unit)
- Automated classification (agents still drive classification)

### 1.2 Relationship to Option B

This design is the research report's recommended first step. It activates and enhances
the existing system at lower cost and risk than building a standalone server. After
implementation, the remaining gaps should be evaluated against Option B
(`work/design/docs-memory-mcp-design.md`). If BM25 + populated concepts + indexed
graph proves sufficient for Kanbanzai's needs, Option B may be unnecessary. If not,
the SQLite schema and search logic from this work become the foundation for Option B.

---

## 2. Design Principles

This design inherits all six principles from the document intelligence design
(`work/design/document-intelligence-design.md` §4). No new principles are introduced.
The relevant ones for this work:

- **DP-1: The tool is a database; agents are the intelligence.** The SQLite migration
  and FTS5 search are database changes. No embedded AI.
- **DP-3: Graceful degradation.** Search works with Layers 1–2 only. Classification
  makes it better but is not required.
- **DP-5: Deterministic layers are always current.** FTS5 indexes are rebuilt on every
  document ingest. Classifications are eventually consistent.

---

## 3. Enhancement 1: Full-Text Search via FTS5

### 3.1 Problem

There is no way to search document content by arbitrary text within `doc_intel`. An
agent looking for "all sections about authentication timeout" must fall back to `grep`,
which returns raw line matches without section context, relevance ranking, or
progressive disclosure. This is the single largest capability gap identified in the
audit (`work/research/document-retrieval-for-ai-agents.md` §3d, weakness #1).

### 3.2 Solution

Add a `search` action to the `doc_intel` MCP tool that performs BM25-ranked full-text
search over section content using SQLite's FTS5 extension.

### 3.3 New Action: `search`

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | Yes | Search query (FTS5 syntax: terms, phrases `"..."`, AND, OR, NOT) |
| `mode` | string | No | Output level: `outline` (default), `summary`, `full` |
| `limit` | int | No | Max results (default: 10, max: 50) |
| `doc_type` | string | No | Filter by document type (design, specification, dev-plan, research, report) |
| `role` | string | No | Filter by classified role (requirement, decision, rationale, etc.) |

**Return format (outline mode — default):**

```/dev/null/search-result.yaml#L1-18
results:
  - document_id: "FEAT-01KMKRQRRX3CC/design-init-command"
    document_path: "work/design/init-command.md"
    section_path: "1.3"
    section_title: "3. Skills: Dual Delivery"
    word_count: 258
    role: "definition"               # null if unclassified
    bm25_score: 12.4
  - document_id: "FEAT-01KMKRQSD1TKK/specification-skills-content"
    document_path: "work/spec/skills-content.md"
    section_path: "2.1"
    section_title: "Skill File Structure"
    word_count: 180
    role: "requirement"
    bm25_score: 9.8
query: "skills delivery"
total_matches: 23
returned: 10
```

**Summary mode** adds the first paragraph (or agent-provided summary if classified) to
each result. **Full mode** adds the complete section content. Both modes cost more
tokens but avoid a separate `section` call.

### 3.4 FTS5 Index Design

The FTS5 virtual table indexes section headings and content:

```/dev/null/fts-schema.sql#L1-7
CREATE VIRTUAL TABLE sections_fts USING fts5(
    document_id UNINDEXED,        -- for joining back to results (not searched)
    section_path UNINDEXED,       -- for joining back to results (not searched)
    title,                        -- section heading (searchable)
    content,                      -- section body text (searchable)
    tokenize='porter unicode61'   -- Porter stemming + Unicode normalisation
);
```

**Why contentless is not used here:** Unlike the Option B design, Kanbanzai already
stores per-document indexes with byte offsets for section retrieval. The FTS5 table
needs the content for BM25 scoring but section retrieval still uses the byte-offset
mechanism from the original file. The content column is the source for FTS5 scoring;
it is not read directly for output. The FTS5 content column stores only the text
needed for tokenisation and ranking, not the full formatted Markdown.

**Tokeniser choice:** Porter stemming handles English morphology (running → run,
authentication → authent) which improves recall for natural language queries.
`unicode61` handles accented characters and Unicode normalisation. This combination
is the SQLite FTS5 default for English text and is well-suited to technical
documentation.

### 3.5 Index Population

The FTS5 index is populated during `IngestDocument` (the existing Layer 1–2 pipeline).
After parsing the structural skeleton and extracting patterns, the pipeline inserts
each section's heading and body text into the FTS5 table.

**On re-ingest:** when a document is re-indexed, its existing FTS5 rows are deleted
(by document_id) and re-inserted. This is simpler and safer than attempting incremental
updates to the FTS5 index.

### 3.6 Search Pipeline

```/dev/null/search-flow.txt#L1-10
1. PARSE the query string (passed directly to FTS5 MATCH)
2. EXECUTE: SELECT document_id, section_path, bm25(sections_fts) AS score
            FROM sections_fts WHERE sections_fts MATCH ?
            ORDER BY score LIMIT ?
3. APPLY post-filters (doc_type, role) by joining with document index metadata
4. ENRICH each result with section metadata (title, word count, role, confidence)
   from the per-document index
5. FORMAT output at the requested mode level
6. For summary/full modes, retrieve section content via existing GetSection
```

Post-filtering by doc_type and role happens after the FTS5 query because:
- doc_type requires looking up the document's front matter (stored in per-document
  index, not in the FTS5 table)
- role requires looking up section classification (per-document index)
- FTS5 does not support JOINs inside the MATCH clause

This means the FTS5 query may return more candidates than the limit, which are then
filtered down. The implementation should request `limit × 3` from FTS5 when filters
are active, to ensure enough candidates survive filtering.

---

## 4. Enhancement 2: SQLite-Backed Graph Storage

### 4.1 Problem

The document graph is stored as a single YAML file (`graph.yaml`, currently 2.3 MB,
60,537 lines, 12,107 edges). Every query that touches the graph — `impact`,
`FindByEntity`, `FindByRole` — deserialises this entire file and scans it linearly.
This is the scaling bottleneck identified in the audit.

### 4.2 Solution

Migrate the graph edge storage from `graph.yaml` to a SQLite database. Per-document
index files remain as YAML — they are small (one per document), accessed by direct
lookup (not scanned), and well-suited to YAML.

### 4.3 What Moves to SQLite

| Data | Current storage | New storage | Reason |
|------|----------------|-------------|--------|
| Graph edges | `graph.yaml` (single flat file) | SQLite `edges` table | Linear scan elimination |
| Entity references | Per-document YAML + cross-doc scan | SQLite `entity_refs` table | Indexed lookup by entity_id |
| FTS5 section index | (new) | SQLite `sections_fts` virtual table | Full-text search |
| Per-document indexes | Per-document YAML files | **No change** | Direct lookup by doc ID is already O(1) |
| Concept registry | `concepts.yaml` | **No change** (small, infrequent access) | Not a bottleneck |

### 4.4 Database Schema

```/dev/null/schema.sql#L1-26
-- Graph edges (replacing graph.yaml)
CREATE TABLE edges (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    from_id   TEXT NOT NULL,
    from_type TEXT NOT NULL,     -- 'document', 'section', 'fragment', 'entity_ref', 'concept'
    to_id     TEXT NOT NULL,
    to_type   TEXT NOT NULL,
    edge_type TEXT NOT NULL      -- 'CONTAINS', 'REFERENCES', 'LINKS_TO', 'INTRODUCES', 'USES'
);

CREATE INDEX idx_edges_from ON edges(from_id, from_type);
CREATE INDEX idx_edges_to ON edges(to_id, to_type);
CREATE INDEX idx_edges_type ON edges(edge_type);

-- Entity reference index (replacing cross-document linear scans)
CREATE TABLE entity_refs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_id   TEXT NOT NULL,      -- e.g. "FEAT-01KMKRQRRX3CC"
    entity_type TEXT NOT NULL,      -- "feature", "task", "bug", "decision", "plan"
    document_id TEXT NOT NULL,      -- which document
    section_path TEXT NOT NULL      -- which section
);

CREATE INDEX idx_entity_refs_entity ON entity_refs(entity_id);
CREATE INDEX idx_entity_refs_document ON entity_refs(document_id);
```

### 4.5 Database Location

`.kbz/index/docint.db` — alongside the existing `documents/` directory.

The database file is a build artefact, not a source file. It can be deleted and
rebuilt from the YAML index files and document source files. It **should not** be
committed to Git. Add `.kbz/index/docint.db` to `.gitignore`.

### 4.6 Migration Strategy

The migration is additive, not destructive:

1. Create the SQLite database on first access (lazy initialisation)
2. On `IngestDocument`, write to both YAML and SQLite (dual-write)
3. On read, prefer SQLite for graph queries and entity lookups
4. Retain YAML per-document indexes as the source of truth for section
   metadata, classifications, and byte offsets
5. `graph.yaml` continues to be written for backward compatibility and
   debugging, but is no longer read for queries

**Why dual-write, not migrate-and-delete:** The YAML files are Git-tracked and
human-readable. They serve as a debugging aid and a backup. The SQLite database
is a performance index over the same data, not a replacement for it.

### 4.7 Impact on Existing Code

| Component | Change |
|-----------|--------|
| `IndexStore` | Add SQLite connection management; new methods for edge/entity queries |
| `IntelligenceService.IngestDocument` | After YAML write, also insert into SQLite tables |
| `IntelligenceService.FindByEntity` | Query `entity_refs` table instead of scanning all YAML files |
| `IntelligenceService.FindByRole` | Query `edges` table (CONTAINS edges filtered by classification) or scan YAML (role data is in per-doc YAML) |
| `IntelligenceService.GetImpact` | Query `edges` table with indexed lookup instead of loading `graph.yaml` |
| `IntelligenceService.TraceEntity` | Query `entity_refs` table + join with document metadata |
| `BuildGraphEdges` | No change (still produces `[]GraphEdge`); edges written to both YAML and SQLite |
| `MergeGraphEdges` | Replaced by SQL DELETE/INSERT for the document being re-indexed |

### 4.8 Rebuild from Source

If the SQLite database is deleted or corrupted:

```/dev/null/rebuild.txt#L1-3
$ kanbanzai rebuild-index
```

This command reads all per-document YAML index files, extracts edges and entity
references, and rebuilds the SQLite database. Runtime: seconds for 280 documents.

---

## 5. Enhancement 3: Batch Classification Protocol

### 5.1 Problem

Layer 3 classification has never been run. The concept registry is empty. The
`find(concept: ...)` action has never returned a result. Role-based search relies
entirely on Layer 2 heading-keyword heuristics. This means the most valuable
search capabilities — concept search and role-based search across the corpus —
are non-functional.

### 5.2 Solution

Define a batch classification workflow that enables an agent to classify multiple
documents efficiently. This is not a new tool — it uses the existing `classify`
action — but it requires a defined protocol for efficient bulk classification.

### 5.3 Batch Classification Workflow

The workflow is agent-driven, using existing tools:

```/dev/null/batch-workflow.txt#L1-22
1. Agent calls doc_intel(action: "pending")
   → Returns list of unclassified document IDs (currently: all 280)

2. Agent selects a batch (e.g., 10 documents, prioritised by type):
   - Specifications first (most structured, highest value from classification)
   - Designs second (narrative + decisions, good concept extraction targets)
   - Dev-plans third (task-oriented, lower classification value)
   - Research/reports last

3. For each document in the batch:
   a. Agent calls doc_intel(action: "guide", id: "<doc-id>")
      → Returns outline with conventional roles, entity refs, extraction hints
   b. Agent reads sections that need classification (via get_section or read_file)
   c. Agent produces classifications:
      - Role assignment for each section
      - Concepts introduced and used
      - One-line summary per section
   d. Agent calls doc_intel(action: "classify", id: "<doc-id>",
      content_hash: "...", model_name: "...", model_version: "...",
      classifications: "[...]")
      → Server validates and stores

4. Repeat until pending list is empty or budget is exhausted
```

### 5.4 Classification on Registration

To prevent the classified corpus from going stale, add a convention:

**When a document is registered via `doc(action: "register")` and the registering
agent has the document in context, the agent should immediately classify it.**

This is a workflow convention, not an enforcement mechanism. The `doc` tool already
calls `IngestDocument` on registration (Layers 1–2). Classification (Layer 3) remains
the agent's responsibility.

To support this, add a nudge to the `doc(action: "register")` response:

```/dev/null/nudge.txt#L1-3
Document registered and indexed (Layers 1–2). Layer 3 classification is pending.
To classify: doc_intel(action: "guide", id: "DOC-xxx") then
doc_intel(action: "classify", id: "DOC-xxx", ...)
```

### 5.5 No New Tool or Action

This enhancement does not add new actions. It defines a protocol for using existing
tools efficiently at scale. The only code change is the nudge message in the `doc`
tool's register action.

---

## 6. Enhancement 4: Knowledge ↔ Document Cross-Queries

### 6.1 Problem

The knowledge system and doc_intel are completely separate. An agent cannot query
"what knowledge entries relate to this specification" or "what document sections
inform this knowledge entry." The research report identified this as weakness #4
(`work/research/document-retrieval-for-ai-agents.md` §3d).

### 6.2 Solution

Add cross-system query capabilities in two directions:

1. **doc_intel → knowledge:** When searching by entity, also return related knowledge
   entries
2. **knowledge → doc_intel:** When surfacing knowledge for a task, also surface
   relevant document section pointers

### 6.3 Direction 1: Entity Search Includes Knowledge

When `doc_intel(action: "find", entity_id: "FEAT-xxx")` is called, the response
currently lists document sections that reference the entity. Enhance this to also
include knowledge entries that:

- Have the entity ID in their `learned_from` field
- Have tags matching the entity type or ID
- Have scope matching any document that references the entity

This uses the existing entity reference scanner in `internal/knowledge/links.go`.

**Modified response format:**

```/dev/null/find-response.yaml#L1-17
document_sections:
  - document_id: "FEAT-xxx/specification-auth"
    section_path: "2.1"
    title: "Authentication Requirements"
    role: "requirement"
related_knowledge:
  - id: "KE-01KN87PQD7S3K"
    topic: "auth-timeout-handling"
    content: "Authentication timeouts should use..."
    confidence: 0.85
    status: "confirmed"
entity_id: "FEAT-xxx"
document_matches: 5
knowledge_matches: 2
```

### 6.4 Direction 2: Knowledge Surfacing Includes Document Pointers

The knowledge surfacer (`internal/knowledge/surface.go`, `MatchEntries`) currently
returns knowledge entries matched by file path prefix, role tags, and scope. Enhance
the context assembly pipeline to also include a lightweight "related documents" section
when knowledge entries reference entity IDs that appear in indexed documents.

This is a change to the context assembly pipeline (`internal/mcp/assembly.go`), not
to the knowledge tool itself. The assembly step that surfaces knowledge
(`stepSurfaceKnowledge`) already has access to the intelligence service. It can query
entity references to find related documents and include them as pointers (document path
+ section path, not full content).

**Token cost:** Minimal. Each document pointer is ~20 tokens. A typical task might
surface 3–5 related document pointers alongside 5–10 knowledge entries.

### 6.5 Implementation Boundary

The cross-system queries are **additive** — they add information to existing responses
without changing the existing fields. An agent that ignores the new fields sees the
same behaviour as before.

The `IntelligenceService` and `KnowledgeSurfacer` need read access to each other.
Currently they are independent services. The integration point is the context assembly
pipeline, which already has access to both. For the `find` action in doc_intel, the
tool handler needs a reference to the knowledge store (or a query interface).

### 6.6 What This Does Not Do

- Does not create bidirectional links between knowledge entries and documents
  (that would require modifying knowledge entry storage)
- Does not merge knowledge and document search into a single query
  (they remain separate systems with different data models)
- Does not change the knowledge surfacing algorithm (scoring, cap, matching)

---

## 7. Enhancement 5: Concept Alias Resolution

### 7.1 Problem

The `Concept` type declares an `Aliases` field that is never used. The source code
contains a TODO:

```kanbanzai/internal/docint/types.go#L109-110
	Aliases []string `yaml:"aliases,omitempty"` // Alternative forms
	// TODO: Aliases field is currently unused - implement alias resolution or remove
```

When the concept registry is populated (via batch classification), concepts with
different surface forms but the same meaning — "rate-limiting" vs. "throttling",
"context-window" vs. "context-budget" — will be treated as separate concepts.

### 7.2 Solution

Implement alias resolution in `FindByConcept` so that searching for any alias of a
concept returns results for the canonical concept.

### 7.3 Alias Declaration

Aliases are declared by agents during classification. When an agent classifies a
section and tags a concept, it may also declare aliases:

```/dev/null/classification-with-alias.yaml#L1-6
- section_path: "3.2"
  role: "definition"
  concepts_intro:
    - name: "rate-limiting"
      aliases: ["throttling", "request-throttling"]
  concepts_used: ["api-gateway"]
```

This requires a minor extension to the `Classification` struct's `ConceptsIntro`
field. Currently it is `[]string`. To support aliases, it needs to accept either a
plain string (for backward compatibility) or an object with `name` and `aliases`.

**Parsing strategy:** If the YAML value is a string, treat it as a concept name with
no aliases. If it is a map with `name` and `aliases`, extract both. This maintains
backward compatibility with existing classifications (if any are ever created).

### 7.4 Resolution in FindByConcept

The current `FindConcept` function in `internal/docint/concept.go` matches on the
normalised canonical name only. Enhance it to also check aliases:

```/dev/null/alias-resolution.txt#L1-6
For query concept Q:
  1. Normalise Q (lowercase, hyphenate)
  2. Search concept registry for Q as canonical name → match
  3. If no match: search all concepts' alias lists for Q → match on canonical
  4. If still no match: return nil
```

This is a simple linear scan of the (small) concept registry. The registry will have
at most a few hundred concepts; alias resolution does not need indexing.

### 7.5 Alias Management

Aliases accumulate. If two agents independently classify sections and declare different
aliases for the same concept, all aliases are retained. Deduplication is by normalised
form only — "Rate Limiting" and "rate-limiting" are the same alias.

There is no explicit alias removal. If an alias is wrong, the concept can be
re-classified without the incorrect alias, and the concept registry rebuild (from
current classifications) will drop it.

---

## 8. What Changes in the MCP Tool Interface

### 8.1 doc_intel Tool

| Change | Type |
|--------|------|
| New `search` action | Addition |
| `find(entity_id)` response includes `related_knowledge` field | Additive |
| `classify` accepts concept objects with aliases (alongside plain strings) | Backward-compatible extension |

No existing actions are removed or have their interfaces changed. Agents that do not
use the new capabilities see no difference.

### 8.2 knowledge Tool

No changes to the knowledge MCP tool. Cross-system integration is in the context
assembly pipeline, not the tool surface.

### 8.3 doc Tool

`doc(action: "register")` response includes a classification nudge. This is a message
change, not a schema change.

### 8.4 New CLI Command

`kanbanzai rebuild-index` — rebuilds the SQLite database from YAML index files and
document source files. Used after database deletion/corruption or after a Git checkout
that changes the YAML indexes.

---

## 9. Storage Changes

### 9.1 New File

| File | Purpose | Git-tracked |
|------|---------|-------------|
| `.kbz/index/docint.db` | SQLite database (FTS5 + edges + entity_refs) | No |

### 9.2 Modified Files

| File | Change |
|------|--------|
| `.kbz/index/graph.yaml` | Still written (dual-write) but no longer read for queries |
| `.kbz/index/concepts.yaml` | Aliases field populated when classification includes them |
| `.gitignore` | Add `.kbz/index/docint.db` |

### 9.3 No Changes

Per-document YAML index files, knowledge entries, and document records are unchanged.

---

## 10. Dependency Changes

### 10.1 New Dependency

SQLite driver for Go. Two options:

| Package | Type | Pros | Cons |
|---------|------|------|------|
| `modernc.org/sqlite` | Pure Go | No CGo, easy cross-compilation | 2–3× slower |
| `mattn/go-sqlite3` | CGo | Faster, battle-tested | Requires C compiler for builds |

**Recommendation:** `modernc.org/sqlite`. The performance difference is negligible
for our data volume (hundreds of documents, thousands of edges). Pure Go simplifies
the build pipeline and avoids CGo complications.

### 10.2 FTS5 Availability

FTS5 is compiled into both SQLite drivers by default. No special build flags or
extensions needed.

---

## 11. Performance Expectations

### 11.1 Search Latency

| Operation | Current (YAML) | After (SQLite) |
|-----------|---------------|----------------|
| Full-text search (new) | N/A (grep fallback: ~100ms) | <10ms |
| FindByEntity | ~200ms (scan 280 YAML files) | <5ms (indexed lookup) |
| GetImpact | ~100ms (deserialise 2.3MB YAML) | <5ms (indexed lookup) |
| FindByRole | ~200ms (scan 280 YAML files) | ~200ms (role data in per-doc YAML, not SQLite) |
| GetOutline | <5ms (single YAML file) | <5ms (no change) |
| GetSection | <5ms (single file read) | <5ms (no change) |

`FindByRole` does not improve because role data is in per-document YAML classifications,
not in the SQLite tables. Moving classification data to SQLite is a possible future
enhancement but is not included in this design — it would change the per-document YAML
schema and the classification protocol.

### 11.2 Storage Overhead

The SQLite database will be approximately 5–10 MB for the current corpus (280 documents,
12K edges, ~50K FTS5 entries). This is a modest addition to the 5.7 MB currently used by
the YAML index files.

### 11.3 Index Build Time

Full rebuild from YAML sources: <5 seconds for 280 documents. Incremental updates
(single document re-index): <50ms.

---

## 12. Phasing

The five enhancements can be implemented in any order, but the recommended sequence
maximises early value:

### Phase 1: Search + SQLite (Week 1–2)

1. Add SQLite database initialisation to `IndexStore`
2. Implement FTS5 table and population during `IngestDocument`
3. Implement the `search` action on `doc_intel`
4. Migrate graph edges to SQLite `edges` table
5. Migrate entity reference queries to SQLite `entity_refs` table
6. Add `rebuild-index` CLI command
7. Update `.gitignore`

**Deliverable:** Full-text search works. Graph queries are indexed. Agents can search
document content without grep.

### Phase 2: Batch Classification (Week 2–3)

8. Add classification nudge to `doc(action: "register")` response
9. Define and document the batch classification protocol
10. Run batch classification on priority documents (specs, designs)
11. Verify concept registry population and concept-based search

**Deliverable:** Concept registry populated. `find(concept: ...)` returns results.
Role-based search has Layer 3 data. Classification becomes part of the document
registration workflow.

### Phase 3: Cross-System + Aliases (Week 3–4)

12. Extend `find(entity_id)` to include related knowledge entries
13. Extend context assembly to include document pointers alongside knowledge
14. Implement concept alias storage in classification
15. Implement alias resolution in `FindByConcept`

**Deliverable:** Knowledge and documents are connected. Concept search handles synonyms.

---

## 13. Open Questions

### 13.1 Design Questions

1. **Should `FindByRole` also move to SQLite?** This would require storing
   classifications (or at least role + section_path pairs) in a SQLite table,
   duplicating data from the per-document YAML files. The benefit is query speed;
   the cost is maintaining consistency between YAML and SQLite for classification
   data. Current recommendation: defer until role queries are demonstrably slow.

2. **Should `search` support graph-boosted ranking?** The Option B design includes
   a composite score that boosts results by in-degree (structurally important sections)
   and role (requirements/decisions over narrative). This could be added to the `search`
   action here. Current recommendation: start with pure BM25 and collect real-world
   usage signals before deciding.

   **Evaluation protocol:** Rather than designing an upfront benchmark, use the existing
   retrospective and knowledge systems to accumulate evidence from real usage:

   a. **Instrument search with lightweight feedback.** When an agent calls `search` and
      then immediately calls `get_section` on one of the results, record which result
      rank was selected (1st, 3rd, 8th, etc.). When an agent calls `search` and then
      falls back to `grep` or `read_file` on a different document, record that as a
      miss. This telemetry can be captured in the context report mechanism that already
      tracks knowledge entry usage (`ContextReport(taskID, used, flagged)`).

   b. **Define a retrospective signal category.** Agents already record retrospective
      signals via `finish(retrospective: [...])`. Add a convention: when an agent
      observes that `search` returned irrelevant results, or that a structurally
      important section (a requirement, a decision) was buried below narrative, record
      a retrospective signal with category `tool-friction` and tag `doc-intel-search`.

   c. **Trigger evaluation via `retro(action: "synthesise")`.** After 2–4 weeks of
      real usage, run `retro(action: "synthesise", scope: "project")` and filter for
      `doc-intel-search` signals. If the synthesis surfaces a pattern of "BM25 ranks
      narrative above requirements" or "important sections consistently ranked low,"
      that is the evidence to add structural boosting.

   d. **Contribute a knowledge entry with the decision.** When the evaluation is done,
      record the outcome as a knowledge entry:
      `knowledge(action: "contribute", topic: "search-ranking-evaluation",
      content: "BM25 alone is [sufficient|insufficient] for doc_intel search.
      Evidence: [summary of retro signals]. Decision: [keep pure BM25 | add
      structural boosting].", scope: "project", tier: 2, tags: ["doc-intel",
      "search", "decision"])`

   This approach avoids speculative benchmarking and lets the system's own feedback
   mechanisms surface the evidence. The decision point is: **if 3+ retrospective
   signals with `tool-friction` + `doc-intel-search` accumulate, evaluate structural
   boosting. If none accumulate after a month of usage, pure BM25 is sufficient.**

3. **Should batch classification be a separate tool or action?** The current design
   uses the existing `classify` action called repeatedly. A dedicated
   `classify_batch` action could accept multiple documents in one call. Current
   recommendation: not needed. Each classification requires reading the document
   first; batching the classify call doesn't eliminate the bottleneck (agent reading
   time). Individual calls also provide clearer error handling per document.

### 13.2 Implementation Questions

4. **SQLite connection lifecycle.** Should the database connection be opened once at
   server start and held open, or opened per-query? Recommendation: open at server
   start (in `NewIntelligenceService`), hold as a field, close at shutdown. WAL mode
   supports concurrent reads from multiple goroutines on a single connection.

5. **FTS5 tokeniser tuning.** Porter stemming may over-stem domain-specific terms
   (e.g., "kanbanzai" → "kanbanzi"). Should a custom tokeniser preserve known domain
   terms? Recommendation: start with default Porter + unicode61 and evaluate on real
   queries before adding complexity.

6. **Concept alias input format.** The enhancement proposes that `concepts_intro`
   accepts either strings or objects. This complicates the YAML parsing for
   classifications. An alternative is a separate `aliases` field on the classification
   submission. Recommendation: evaluate during implementation — choose whichever is
   simpler to parse.

---

## 14. What This Design Is Not

1. **Not a replacement for doc_intel.** This enhances the existing system, not
   replaces it. All existing actions, types, and storage formats are preserved.

2. **Not a standalone tool.** These enhancements live within `kanbanzai serve`.
   A standalone tool is Option B, to be evaluated after this work.

3. **Not a vector search system.** FTS5/BM25 is keyword-based with stemming. If
   semantic search (embedding-based) proves necessary after evaluation, it would be
   a separate enhancement or the trigger for Option B.

4. **Not a specification.** The exact schemas, error messages, and test cases belong
   in the specification. This document defines what to build and why.

---

## 15. Success Criteria

After implementation, the following should be true:

1. An agent can search for "authentication timeout" and receive ranked,
   section-level results without using grep.
2. Graph queries (impact, entity search) complete in <10ms instead of ~200ms.
3. The concept registry contains entries from classified documents.
4. `find(concept: "rate-limiting")` returns results, including sections tagged
   with the alias "throttling".
5. `find(entity_id: "FEAT-xxx")` returns both document sections and related
   knowledge entries.
6. Context assembly for a task includes document section pointers alongside
   knowledge entries.
7. All existing doc_intel functionality works unchanged.
