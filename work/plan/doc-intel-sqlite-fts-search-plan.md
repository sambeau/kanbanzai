# Dev Plan: SQLite Storage and Full-Text Search

> Feature: FEAT-01KPNNYYSQTA7 — SQLite storage and full-text search
> Spec: work/spec/doc-intel-sqlite-fts-search.md

---

## Overview

This plan implements `work/spec/doc-intel-sqlite-fts-search.md`. It adds a SQLite
database at `.kbz/index/docint.db` with three tables (`sections_fts`, `edges`,
`entity_refs`) and migrates the read paths for `FindByEntity`, `GetImpact`, and
`TraceEntity` from linear YAML scans to indexed SQLite queries. A new `search` action
on `doc_intel` provides BM25-ranked full-text search. A `rebuild-index` CLI command
reconstructs the database from YAML source.

The `modernc.org/sqlite` driver is already a project dependency (go.mod). No CGo.

Five sequential-with-parallel-tail tasks.

---

## Task Breakdown

### Task 1: SQLite connection management and schema creation

**Description:** Add SQLite database initialisation to `IndexStore`. On first access,
create the database at `.kbz/index/docint.db` (lazy init, not at startup). Create all
three tables plus indexes. Enable WAL mode. Hold one connection for the lifetime of the
service. Add `.gitignore` entry for `docint.db`. Add `Close()` method to `IndexStore`
(called on server shutdown via service cleanup).

**Files:** `internal/docint/store.go`, `internal/docint/store_test.go`, `.gitignore`

**Deliverable:**
- `IndexStore.openDB()` (lazy) creates and migrates schema
- Tables: `sections_fts` (FTS5), `edges`, `entity_refs`
- All indexes created
- WAL mode enabled
- `IndexStore.Close()` closes the SQLite connection
- `.kbz/index/docint.db` in `.gitignore`
- Unit tests for schema creation, WAL mode

**Traceability:** FR-001, FR-002, FR-003, FR-004, FR-005, FR-016, FR-017, FR-020, FR-021

### Task 2: Dual-write: IngestDocument populates SQLite

**Description:** Modify `IntelligenceService.IngestDocument` (in
`internal/service/intelligence.go`) to write to SQLite after the existing YAML write.
For `sections_fts`: delete existing rows for the document, then insert one row per
section leaf (title + extracted plain-text content). For `edges`: delete-then-insert in
a transaction. For `entity_refs`: delete-then-insert in a transaction. The YAML
`graph.yaml` write continues (FR-019 dual-write).

Add a helper `extractSectionText(content []byte, section docint.Section) string` that
strips Markdown formatting to produce plain text for FTS5 indexing.

**Files:** `internal/service/intelligence.go`, `internal/docint/store.go`,
`internal/service/intelligence_test.go`

**Deliverable:**
- After `IngestDocument`, SQLite contains correct rows for the document
- Re-ingest replaces (not appends) rows for the same document
- YAML write is unchanged (FR-019)
- `extractSectionText` strips Markdown `#`, `*`, `` ` ``, `_`, `[...](...)`
- Tests confirm row counts match section/edge/ref counts

**Traceability:** FR-006, FR-007, FR-008, FR-018, FR-019, FR-022, NFR-005, NFR-006

### Task 3: Migrate read queries to SQLite

**Description:** Update `IntelligenceService.FindByEntity`, `GetImpact`, and
`TraceEntity` to query SQLite instead of scanning YAML files. Add corresponding
query methods to `IndexStore`. Return identical results as before migration.

**Files:** `internal/service/intelligence.go`, `internal/docint/store.go`,
`internal/service/intelligence_test.go`

**Deliverable:**
- `FindByEntity`: queries `entity_refs` table by `entity_id`
- `GetImpact`: queries `edges` table by `to_id`
- `TraceEntity`: uses `FindByEntity` (already does)
- `IndexStore.QueryEntityRefsByEntityID(entityID string) ([]docint.EntityRef, error)`
- `IndexStore.QueryEdgesByToID(toID string) ([]docint.GraphEdge, error)`
- Behaviour-equivalent tests pass
- No YAML files read for entity or edge lookups

**Traceability:** FR-023, FR-024, FR-025, NFR-002

### Task 4: search action on doc_intel tool

**Description:** Implement `docIntelSearchAction` in `internal/mcp/doc_intel_tool.go`.
Wire it into the action dispatch map. Parameters: `query` (required), `mode` (default
`outline`), `limit` (default 10, max 50), `doc_type`, `role`. Execute FTS5 MATCH query
with BM25 ranking. Apply `doc_type` and `role` post-filters. Return `query`,
`total_matches`, `returned`, and `results` array.

Add `IndexStore.SearchSections(params SearchParams) ([]SearchResult, error)` where
`SearchParams` and `SearchResult` are defined in the `docint` package.

Update the `doc_intel` tool description to mention the `search` action.

**Files:** `internal/mcp/doc_intel_tool.go`, `internal/docint/store.go`,
`internal/service/intelligence.go`, `internal/mcp/doc_intel_tool_test.go`

**Deliverable:**
- `doc_intel(action: "search", query: "...")` returns ranked results
- `outline`, `summary`, `full` modes return correct fields
- `doc_type` and `role` filters applied as AND post-filters
- `limit` clamped to 50
- Empty result set returns empty array, not error
- Tests cover basic search, filters, modes, empty results

**Traceability:** FR-009, FR-010, FR-011, FR-012, FR-013, FR-014, FR-015, NFR-001

### Task 5: rebuild-index CLI command

**Description:** Add a `rebuild-index` sub-command to the `kanbanzai` CLI in
`cmd/kanbanzai/`. The command reads all per-document YAML index files, deletes and
recreates the SQLite database, then re-populates all three tables by re-processing each
document. Progress is printed: total documents, edges, entity refs, FTS sections.

**Files:** `cmd/kanbanzai/main.go` (or a new `cmd/kanbanzai/rebuild.go`),
`internal/service/intelligence.go` (add `RebuildIndex` method)

**Deliverable:**
- `kanbanzai rebuild-index` completes without error on a project with existing YAML
- Progress output: doc count, edge count, entity ref count, FTS section count
- Rebuilt database passes the same search and query tests as dual-write path
- Running twice produces identical results

**Traceability:** FR-026, FR-027, NFR-003

---

## Dependency Graph

```
T1 → T2, T3, T4, T5   (schema must exist before any write or read)
T2 → T3, T4            (data must exist before queries and search tests)
T3, T4, T5 are independent of each other once T1 and T2 are done
```

Recommended sequence: T1 → T2 → T3 (parallel with T4) → T5

---

## Interface Contracts

### IndexStore additions (internal/docint/store.go)

```go
func (s *IndexStore) Close() error
// Lazy SQLite init called internally on first DB operation.

// Write operations (called from IngestDocument)
func (s *IndexStore) UpsertDocumentSQLite(docID string, sections []Section, content []byte, refs []EntityRef, edges []GraphEdge) error

// Read operations
func (s *IndexStore) QueryEntityRefsByEntityID(entityID string) ([]EntityRef, error)
func (s *IndexStore) QueryEdgesByToID(toID string) ([]GraphEdge, error)
func (s *IndexStore) SearchSections(params SearchParams) ([]SearchResult, error)
```

### IntelligenceService additions (internal/service/intelligence.go)

```go
func (s *IntelligenceService) Search(params SearchParams) ([]SearchResult, error)
func (s *IntelligenceService) RebuildIndex() (RebuildStats, error)
```

### New doc_intel action

`doc_intel(action: "search", query, mode, limit, doc_type, role)` — returns
`{query, total_matches, returned, results: [...]}`

---

## Traceability Matrix

| Task | Requirements |
|------|-------------|
| T1   | FR-001, FR-002, FR-003, FR-004, FR-005, FR-016, FR-017, FR-020, FR-021 |
| T2   | FR-006, FR-007, FR-008, FR-018, FR-019, FR-022, NFR-005, NFR-006 |
| T3   | FR-023, FR-024, FR-025, NFR-002 |
| T4   | FR-009 – FR-015, NFR-001 |
| T5   | FR-026, FR-027, NFR-003 |
