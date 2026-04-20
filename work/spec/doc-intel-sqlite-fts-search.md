# Specification: SQLite Storage and Full-Text Search

| Field | Value |
|-------|-------|
| Feature | FEAT-01KPNNYYSQTA7 — SQLite storage and full-text search |
| Date | 2025-07-14 |
| Status | Draft |
| Design | `work/design/doc-intel-enhancement-design.md` §3, §4 |

---

## 1. Overview

This specification defines the requirements for adding SQLite-backed storage to
Kanbanzai's document intelligence system and introducing BM25 full-text search over
section content. The SQLite database replaces the flat YAML graph file for edge and
entity reference queries (eliminating linear scans) and hosts an FTS5 virtual table
that powers a new `search` action on the `doc_intel` MCP tool. Per-document YAML index
files are retained as the source of truth; SQLite serves as a derived performance index.

---

## 2. Scope

### In Scope

- SQLite database creation, initialisation, and connection lifecycle
- FTS5 virtual table for section content search
- New `search` action on the `doc_intel` MCP tool
- Graph edges table replacing `graph.yaml` for read queries
- Entity references table replacing cross-document linear scans
- Dual-write strategy (YAML remains source of truth, SQLite is derived)
- `rebuild-index` CLI command
- `.gitignore` update for the database file

### Explicitly Out of Scope

- Classification data in SQLite (classifications remain in per-document YAML)
- Concept registry in SQLite (remains in `concepts.yaml`)
- Embedding-based or semantic search
- Changes to existing `doc_intel` actions (outline, section, guide, find, etc.)
- Changes to document lifecycle management (`doc` tool)
- FTS5 tokeniser customisation beyond Porter + unicode61 defaults

---

## 3. Functional Requirements

### SQLite Database Lifecycle

**FR-001:** `IndexStore` MUST create a SQLite database at `.kbz/index/docint.db`
on first access (lazy initialisation). The database MUST NOT be created at server
startup if no document intelligence operations are invoked.
(Design §4.5)

- [ ] Database file is created at `.kbz/index/docint.db` on first `IngestDocument`
- [ ] Database file does not exist after server start with no doc_intel calls
- [ ] Calling `IngestDocument` on a fresh installation creates the database

**FR-002:** The SQLite database MUST use WAL (Write-Ahead Logging) mode for crash
safety and concurrent read support.
(Design §4.4)

- [ ] `PRAGMA journal_mode` returns `wal` after database creation
- [ ] Concurrent read operations do not block each other

**FR-003:** The SQLite connection MUST be opened once during `IndexStore` initialisation
and held for the lifetime of the service. The connection MUST be closed on server
shutdown.
(Design §13.2, question 4)

- [ ] Only one database connection exists per `IndexStore` instance
- [ ] Connection is closed when the server process exits cleanly

**FR-004:** `.kbz/index/docint.db` MUST be added to the project `.gitignore` file.
The database is a derived build artefact, not a source file.
(Design §4.5)

- [ ] `.kbz/index/docint.db` appears in `.gitignore`
- [ ] `git status` does not show the database file as untracked after creation

**FR-005:** The database schema MUST be created automatically on first connection if
the tables do not exist. The system MUST NOT require a separate migration step.
(Design §4.4)

- [ ] Tables are created on first access to a new database
- [ ] Connecting to an existing database with correct schema does not error

### FTS5 Virtual Table

**FR-006:** The database MUST contain an FTS5 virtual table `sections_fts` with the
following schema:
- `document_id` (UNINDEXED) — for joining back to document index
- `section_path` (UNINDEXED) — for joining back to section
- `title` (searchable) — section heading text
- `content` (searchable) — section body text
- Tokeniser: `porter unicode61`
(Design §3.4)

- [ ] `sections_fts` table exists with the specified columns
- [ ] `document_id` and `section_path` columns are not searchable via MATCH
- [ ] `title` and `content` columns are searchable via MATCH
- [ ] Porter stemming is active (searching "running" matches "run")

**FR-007:** During `IngestDocument`, after parsing the structural skeleton, the
pipeline MUST insert each leaf section's heading and body text into `sections_fts`.
(Design §3.5)

- [ ] After ingesting a document, `sections_fts` contains one row per section
- [ ] The `title` column contains the section heading text
- [ ] The `content` column contains the section body text (plain text, not Markdown)

**FR-008:** On re-ingest of an existing document, the pipeline MUST delete all
existing `sections_fts` rows for that document (by `document_id`) before inserting
new rows. Partial updates MUST NOT be attempted.
(Design §3.5)

- [ ] Re-ingesting a document replaces all its FTS5 rows
- [ ] The row count for a document after re-ingest matches the current section count
- [ ] FTS5 rows for other documents are not affected

### Search Action

**FR-009:** The `doc_intel` MCP tool MUST support a new `search` action with the
following parameters:
- `query` (string, required) — search query in FTS5 syntax
- `mode` (string, optional, default: `outline`) — output level: `outline`, `summary`, `full`
- `limit` (integer, optional, default: 10, max: 50) — maximum results
- `doc_type` (string, optional) — filter by document type
- `role` (string, optional) — filter by classified section role
(Design §3.3)

- [ ] `search` action is registered and callable
- [ ] Missing `query` parameter returns an actionable error message
- [ ] Default `mode` is `outline`, default `limit` is 10
- [ ] `limit` values above 50 are clamped to 50

**FR-010:** The `search` action MUST execute the query against the `sections_fts` table
using FTS5 MATCH and return results ranked by BM25 score (best match first).
(Design §3.6)

- [ ] Results are ordered by BM25 score descending
- [ ] A query matching multiple sections returns them in relevance order
- [ ] An empty result set returns an empty `results` array, not an error

**FR-011:** When `doc_type` or `role` filters are provided, the `search` action MUST
apply them as post-filters after the FTS5 query. The FTS5 query SHOULD request
`limit × 3` candidates when filters are active to ensure sufficient results survive
filtering.
(Design §3.6)

- [ ] Filtering by `doc_type: "specification"` returns only sections from specifications
- [ ] Filtering by `role: "requirement"` returns only sections classified as requirements
- [ ] Combined filters (doc_type + role) apply as AND
- [ ] Filtering does not produce fewer than `limit` results when more matches exist

**FR-012:** In `outline` mode (default), each result MUST include: `document_id`,
`document_path`, `section_path`, `section_title`, `word_count`, `role` (null if
unclassified), and `bm25_score`.
(Design §3.3)

- [ ] All specified fields are present in outline mode results
- [ ] `role` is null (not absent) for unclassified sections

**FR-013:** In `summary` mode, each result MUST include all `outline` fields plus
a `summary` field containing the agent-provided summary (if classified) or the first
paragraph of the section (if unclassified).
(Design §3.3)

- [ ] Summary mode includes all outline fields plus `summary`
- [ ] Classified sections use the agent-provided summary
- [ ] Unclassified sections use the first paragraph as summary

**FR-014:** In `full` mode, each result MUST include all `outline` fields plus the
complete section content, retrieved via the existing byte-offset mechanism from the
original file.
(Design §3.3)

- [ ] Full mode includes all outline fields plus `content`
- [ ] Content matches what `doc_intel(action: "section")` returns for the same section

**FR-015:** The `search` response MUST include metadata fields: `query` (the original
query string), `total_matches` (total FTS5 matches before filtering/limiting), and
`returned` (count of results in the response).
(Design §3.3)

- [ ] Response includes `query`, `total_matches`, and `returned` fields
- [ ] `total_matches` reflects the count before limit is applied
- [ ] `returned` matches the length of the `results` array

### Graph Edges Table

**FR-016:** The database MUST contain an `edges` table with columns: `id` (INTEGER
PRIMARY KEY AUTOINCREMENT), `from_id` (TEXT NOT NULL), `from_type` (TEXT NOT NULL),
`to_id` (TEXT NOT NULL), `to_type` (TEXT NOT NULL), `edge_type` (TEXT NOT NULL).
(Design §4.4)

- [ ] `edges` table exists with the specified schema
- [ ] Columns enforce NOT NULL constraints

**FR-017:** The `edges` table MUST have indexes on: `(from_id, from_type)`,
`(to_id, to_type)`, and `(edge_type)`.
(Design §4.4)

- [ ] All three indexes exist
- [ ] Query plans for edge lookups use indexes (EXPLAIN QUERY PLAN shows index usage)

**FR-018:** During `IngestDocument`, after building graph edges, the pipeline MUST
delete all existing edges for the document (by `from_id` prefix match) and insert
the new edges into the `edges` table. This MUST happen in the same transaction.
(Design §4.6, §4.7)

- [ ] Re-ingesting a document replaces its edges atomically
- [ ] Edges for other documents are not affected
- [ ] A failure mid-insert does not leave partial edges

**FR-019:** `IngestDocument` MUST dual-write: edges are written to both the SQLite
`edges` table and the existing `graph.yaml` file. The YAML write MUST continue to
use the existing `SaveGraph` mechanism.
(Design §4.6)

- [ ] After ingest, edges exist in both SQLite and `graph.yaml`
- [ ] Edge counts in SQLite and YAML are equal for the same document

### Entity References Table

**FR-020:** The database MUST contain an `entity_refs` table with columns: `id`
(INTEGER PRIMARY KEY AUTOINCREMENT), `entity_id` (TEXT NOT NULL), `entity_type`
(TEXT NOT NULL), `document_id` (TEXT NOT NULL), `section_path` (TEXT NOT NULL).
(Design §4.4)

- [ ] `entity_refs` table exists with the specified schema
- [ ] Columns enforce NOT NULL constraints

**FR-021:** The `entity_refs` table MUST have indexes on `(entity_id)` and
`(document_id)`.
(Design §4.4)

- [ ] Both indexes exist
- [ ] Entity lookups use the `entity_id` index

**FR-022:** During `IngestDocument`, entity references MUST be inserted into the
`entity_refs` table after deleting existing rows for the document.
(Design §4.7)

- [ ] Re-ingesting a document replaces its entity reference rows
- [ ] Entity references for other documents are not affected

### Query Migration

**FR-023:** `IntelligenceService.FindByEntity` MUST query the `entity_refs` SQLite
table instead of scanning all per-document YAML files.
(Design §4.7)

- [ ] `FindByEntity` returns the same results as before migration
- [ ] `FindByEntity` does not read any per-document YAML files for entity matching

**FR-024:** `IntelligenceService.GetImpact` MUST query the `edges` SQLite table
instead of loading and scanning `graph.yaml`.
(Design §4.7)

- [ ] `GetImpact` returns the same results as before migration
- [ ] `GetImpact` does not read `graph.yaml`

**FR-025:** `IntelligenceService.TraceEntity` MUST query the `entity_refs` SQLite
table for entity reference lookup. Document metadata lookup (for refinement chain
ordering) MAY continue to use per-document YAML files.
(Design §4.7)

- [ ] `TraceEntity` returns the same results as before migration
- [ ] Entity reference lookup uses SQLite, not YAML file scanning

### Rebuild Command

**FR-026:** A `kanbanzai rebuild-index` CLI command MUST exist that:
1. Reads all per-document YAML index files from `.kbz/index/documents/`
2. Reads the source Markdown files for FTS5 content
3. Deletes the existing SQLite database (if any)
4. Creates a fresh database and populates all tables
(Design §4.8)

- [ ] `kanbanzai rebuild-index` succeeds on a project with existing YAML indexes
- [ ] The rebuilt database contains the correct edge count
- [ ] The rebuilt database contains the correct entity reference count
- [ ] The rebuilt FTS5 index returns search results after rebuild
- [ ] Running rebuild-index twice produces the same database content

**FR-027:** `rebuild-index` MUST report progress: total documents processed, total
edges inserted, total entity references inserted, total FTS5 sections indexed.

- [ ] Output includes document count
- [ ] Output includes edge and entity reference counts

---

## 4. Non-Functional Requirements

**NFR-001:** Full-text search queries MUST complete in under 10ms for the current
corpus size (~280 documents, ~50K sections).
(Design §11.1)

- [ ] Benchmark search on 280-document corpus completes in <10ms (median)

**NFR-002:** Entity reference and graph impact queries MUST complete in under 5ms.
(Design §11.1)

- [ ] `FindByEntity` benchmark completes in <5ms (median)
- [ ] `GetImpact` benchmark completes in <5ms (median)

**NFR-003:** `rebuild-index` MUST complete in under 5 seconds for 280 documents.
(Design §11.3)

- [ ] Full rebuild on current corpus completes in <5 seconds

**NFR-004:** The SQLite database MUST be under 10 MB for the current corpus.
(Design §11.2)

- [ ] Database file size is <10 MB after full indexing

**NFR-005:** Incremental updates (single document re-ingest) MUST complete in
under 50ms.
(Design §11.3)

- [ ] Re-ingesting a single document (YAML + SQLite) completes in <50ms

**NFR-006:** Existing `doc_intel` actions (`outline`, `section`, `guide`, `find`,
`classify`, `trace`, `impact`, `pending`) MUST continue to work with identical
behaviour. The SQLite migration MUST NOT change any existing tool responses.

- [ ] All existing doc_intel action tests pass without modification
- [ ] Response format for existing actions is unchanged

---

## 5. Acceptance Criteria

### Integration-Level Criteria

- [ ] An agent can call `doc_intel(action: "search", query: "authentication")` and
  receive ranked, section-level results
- [ ] An agent can filter search by `doc_type` and `role` and receive only matching
  results
- [ ] `doc_intel(action: "find", entity_id: "FEAT-xxx")` returns the same results
  as before but completes in <5ms instead of ~200ms
- [ ] `doc_intel(action: "impact", section_id: "DOC-xxx#1.2")` returns the same
  results but completes in <5ms instead of ~100ms
- [ ] Deleting `.kbz/index/docint.db` and running `kanbanzai rebuild-index`
  restores full functionality
- [ ] `.kbz/index/docint.db` does not appear in `git status`
- [ ] All existing tests pass

---

## 6. Dependencies and Assumptions

### Dependencies

- **SQLite driver:** `modernc.org/sqlite` (pure Go, no CGo requirement).
  Design §10.1 recommends this over `mattn/go-sqlite3` for build simplicity.
- **FTS5 extension:** Must be available in the chosen SQLite driver. Both
  `modernc.org/sqlite` and `mattn/go-sqlite3` include FTS5 by default.
- **Markdown content extraction:** The FTS5 index needs plain text content
  from Markdown sections. The existing `IngestDocument` pipeline parses
  structure but does not extract plain text. A lightweight Markdown-to-text
  conversion is needed (strip formatting, keep words).

### Assumptions

- Per-document YAML index files remain the source of truth for section
  metadata, classifications, and byte offsets. SQLite is a derived index.
- The `graph.yaml` file continues to be written for backward compatibility.
  It is no longer read for queries but remains available for debugging.
- FTS5 Porter stemming is adequate for technical English documentation.
  If over-stemming of domain terms proves problematic, tokeniser tuning
  is a future enhancement (Design §13.2, question 5).
