# Specification: Doc Intel Access Instrumentation

**Feature:** FEAT-01KPTHB66Y8TM  
**Status:** Draft  
**Design source:** `work/design/doc-intel-adoption-design.md` §7 (Fix 5: Access Instrumentation)  
**Date:** 2025-01-01

---

## Overview

This specification covers the lightweight access-tracking instrumentation that enables
continuous measurement of `doc_intel` and `knowledge` system adoption. It adds two new
fields (`last_accessed_at`, `recent_use_count`) to knowledge entries — incremented on
every `list` and `get` call — and two new fields (`access_count`, `last_accessed_at`) to
document and section index records — incremented whenever `outline`, `guide`, `section`,
`find`, or `search` actions are called. A `sort: "recent"` option is added to
`knowledge list`, and the `doc(action: "audit")` report is extended with a "Most Accessed
Documents" table. Counter writes are lazy and counts are approximate; accuracy under
concurrent or crash scenarios is explicitly out of scope.

---

## Actual Go File Paths

The design document (§9.3) names several files that do not exist at those exact paths in
the current codebase. The table below maps each design-document reference to the actual
file that must be modified.

| Design document reference | Actual file path |
|---|---|
| `internal/knowledge/store.go` | `internal/storage/knowledge_store.go` (raw YAML store) and `internal/service/knowledge.go` (List/Get business logic) |
| `internal/knowledge/surfacer.go` | `internal/service/knowledge.go` (KnowledgeFilters and List implementation) |
| `internal/docint/index.go` | `internal/docint/types.go` (DocumentIndex struct definition) |
| `internal/service/intelligence.go` | `internal/service/intelligence.go` (unchanged path — confirmed) |
| `internal/service/document.go` | `internal/service/doc_audit.go` (AuditDocuments, AuditResult) |

All requirements in this specification use the actual file paths above.

---

## Scope

### In Scope

- Adding `last_accessed_at` and `recent_use_count` fields to `storage.KnowledgeRecord.Fields`
  (`internal/storage/knowledge_store.go` storage schema; logic in `internal/service/knowledge.go`).
- Incrementing `last_accessed_at` and `recent_use_count` in `KnowledgeService.Get` for the
  retrieved entry.
- Incrementing `last_accessed_at` and `recent_use_count` in `KnowledgeService.List` for every
  entry included in the response.
- Surfacing `recent_use_count` alongside `use_count` in the default `knowledge list` output.
- Adding `sort: "recent"` as a valid sort option on `KnowledgeFilters` / `KnowledgeService.List`,
  ordering results by descending `recent_use_count`.
- Adding `AccessCount int` and `LastAccessedAt *time.Time` fields to `docint.DocumentIndex`
  in `internal/docint/types.go`.
- Adding a per-section access structure (new `SectionAccessInfo` type or equivalent) to
  `internal/docint/types.go`, carrying `AccessCount int` and `LastAccessedAt *time.Time` keyed
  by section path; stored within `DocumentIndex`.
- Incrementing the document-level `AccessCount` and updating `LastAccessedAt` in
  `IntelligenceService.GetOutline` (`internal/service/intelligence.go`).
- Incrementing the document-level `AccessCount` in `IntelligenceService` guide handler
  (`internal/service/intelligence.go`).
- Incrementing both the document-level counter and the section-level counter for the targeted
  section path in `IntelligenceService.GetSection` (`internal/service/intelligence.go`).
- Incrementing document-level counters for every document appearing in the result set of
  `IntelligenceService.FindByEntity`, `FindByConcept`, `FindByRole`
  (`internal/service/intelligence.go`).
- Incrementing document-level counters for every document appearing in the result set of
  `IntelligenceService.Search` (`internal/service/intelligence.go`).
- Using lazy / deferred counter writes (flush on process shutdown or every N calls); counter
  errors must not fail or slow the primary tool call.
- Extending `AuditResult` in `internal/service/doc_audit.go` with a `MostAccessed` field and
  rendering a "Most Accessed Documents" table (top 10 by 30-day access count) in
  `doc(action: "audit")` output.

### Explicitly Out of Scope

- Per-agent attribution or per-session identity tracking.
- Logging or storing query content (search terms, entity IDs queried).
- Latency measurement per call.
- Skill file changes (`.kbz/skills/`, `.agents/skills/`).
- Corpus onboarding procedures or session-start integrity checks (Fix 2).
- Classification trigger changes or `doc_intel classify` action changes (Fix 3).
- Changes to the `write-design` skill or design-stage corpus consultation mandate (Fix 1).
- Plan close-out knowledge curation (Fix 6).
- Moving counter storage to SQLite (the enhancement design's SQLite graph is a future
  dependency; counters live in the per-document YAML index until that migration occurs).
- Changes to the `doc_intel` or `knowledge` MCP tool action surface beyond what is required
  to expose `sort: "recent"` and the updated audit output.

---

## Functional Requirements

### Knowledge Base Instrumentation

**FR-001** — New `recent_use_count` field on knowledge entries  
A new field `recent_use_count` (integer, default `0`) MUST be present in
`storage.KnowledgeRecord.Fields` for every knowledge entry. Entries that lack the field
when loaded from disk MUST behave as if the field is `0` (zero-value on absent key).

**FR-002** — New `last_accessed_at` field on knowledge entries  
A new field `last_accessed_at` (RFC 3339 timestamp string, omitted when zero) MUST be
stored in `storage.KnowledgeRecord.Fields`. Entries loaded from disk without this field
MUST behave as if the field is absent / zero.

**FR-003** — `KnowledgeService.Get` updates access fields  
When `KnowledgeService.Get(id)` successfully returns an entry, the implementation MUST
increment `recent_use_count` by 1 and set `last_accessed_at` to the current UTC time for
that entry, then persist the update. The update write MUST be best-effort: a write error
MUST NOT cause `Get` to return an error to the caller.

**FR-004** — `KnowledgeService.List` updates access fields for every returned entry  
When `KnowledgeService.List(filters)` returns a non-empty result set, the implementation
MUST increment `recent_use_count` by 1 and set `last_accessed_at` to the current UTC time
for every entry included in the response, then persist the updates. Update write errors
MUST be silently skipped (best-effort, consistent with existing `ContextReport` behaviour).

**FR-005** — `recent_use_count` is a rolling 30-day window  
`recent_use_count` represents the approximate number of accesses in the trailing 30 days.
The implementation MAY use a lazy decay strategy (e.g., reset or subtract counts older than
30 days at read time). Exact precision under concurrent access or crash scenarios is not
required.

**FR-006** — `sort: "recent"` option on `KnowledgeService.List`  
`KnowledgeFilters` MUST include a `Sort string` field (or equivalent). When `Sort` is
`"recent"`, `KnowledgeService.List` MUST return entries ordered by descending
`recent_use_count`. Entries with equal `recent_use_count` MAY be ordered arbitrarily.

**FR-007** — `knowledge list` default output includes `recent_use_count`  
The MCP `knowledge(action: "list")` response MUST include `recent_use_count` in each
entry's field map alongside `use_count`. This requires no change to the tool schema beyond
ensuring the field is present in `record.Fields`.

**FR-008** — `sort: "recent"` exposed through the MCP `knowledge` tool  
The MCP `knowledge(action: "list")` tool handler MUST accept a `sort` string parameter and
pass it to `KnowledgeFilters.Sort`. When `sort: "recent"` is supplied, the response MUST
be ordered by descending `recent_use_count`.

---

### Document Intelligence Instrumentation

**FR-009** — `DocumentIndex.AccessCount` field  
`docint.DocumentIndex` (defined in `internal/docint/types.go`) MUST have an `AccessCount`
field of type `int`. Newly indexed documents and documents loaded from disk without this
field MUST have `AccessCount == 0`.

**FR-010** — `DocumentIndex.LastAccessedAt` field  
`docint.DocumentIndex` MUST have a `LastAccessedAt` field of type `*time.Time`. Newly
indexed documents and documents loaded from disk without this field MUST have
`LastAccessedAt == nil`.

**FR-011** — Per-section access data structure  
`internal/docint/types.go` MUST define a type (named `SectionAccessInfo` or equivalent)
with at minimum the following fields:
- `AccessCount int` — count of `section` calls targeting this section path.
- `LastAccessedAt *time.Time` — timestamp of the most recent `section` read.

`DocumentIndex` MUST contain a field (e.g., `SectionAccess map[string]SectionAccessInfo`)
that maps section path strings to their `SectionAccessInfo`. An absent key in the map is
equivalent to zero-value access data.

**FR-012** — `IntelligenceService.GetOutline` increments document-level counter  
After `IntelligenceService.GetOutline(docID)` successfully resolves a document index, the
implementation MUST increment `DocumentIndex.AccessCount` by 1 and update
`DocumentIndex.LastAccessedAt` to the current UTC time for that document.

**FR-013** — Guide action increments document-level counter  
The code path in `internal/service/intelligence.go` that services the `doc_intel guide`
action MUST increment `DocumentIndex.AccessCount` by 1 and update `LastAccessedAt` for the
referenced document after successfully loading the index.

**FR-014** — `IntelligenceService.GetSection` increments both counters  
After `IntelligenceService.GetSection(docID, sectionPath)` successfully resolves a section,
the implementation MUST:
1. Increment `DocumentIndex.AccessCount` by 1 and update `DocumentIndex.LastAccessedAt`.
2. Increment `SectionAccessInfo.AccessCount` by 1 and update
   `SectionAccessInfo.LastAccessedAt` for the entry keyed by `sectionPath` in
   `DocumentIndex.SectionAccess` (or equivalent map).

**FR-015** — `find` actions increment document-level counters for all result documents  
After `IntelligenceService.FindByEntity`, `FindByConcept`, and `FindByRole` return a
non-empty result set, the implementation MUST increment `DocumentIndex.AccessCount` by 1
and update `DocumentIndex.LastAccessedAt` for every distinct `DocumentID` appearing in the
results.

**FR-016** — `IntelligenceService.Search` increments document-level counters for result documents  
After `IntelligenceService.Search` returns a non-empty result set, the implementation MUST
increment `DocumentIndex.AccessCount` by 1 and update `DocumentIndex.LastAccessedAt` for
every distinct `DocumentID` appearing in the results.

**FR-017** — Counter updates are lazy  
Counter updates to `DocumentIndex` (AccessCount, LastAccessedAt, SectionAccess) MUST NOT
require a synchronous YAML write on every call. An acceptable implementation flushes
dirty indexes on process shutdown or after every N calls (where N is an implementation
choice ≥ 1). Counter update errors MUST NOT cause the primary tool-call method to return an
error.

**FR-018** — Backward-compatible YAML serialisation  
New fields added to `docint.DocumentIndex` (`AccessCount`, `LastAccessedAt`,
`SectionAccess`) MUST be tagged with `yaml:",omitempty"` (or equivalent) so that existing
index files without these fields round-trip without error and load with zero values.

---

### Audit Report Extension

**FR-019** — `AuditResult` extended with `MostAccessed`  
`AuditResult` in `internal/service/doc_audit.go` MUST gain a new field:

```go
MostAccessed []AccessedDocumentEntry `json:"most_accessed,omitempty"`
```

where `AccessedDocumentEntry` (or equivalent name) carries at minimum:
- `DocID string` — the document record ID.
- `Path string` — the repository-relative file path.
- `AccessCount int` — the document's cumulative `access_count`.
- `LastAccessedAt *time.Time` — the document's most recent access timestamp.

**FR-020** — Top-10 most-accessed documents  
When `AuditDocuments` is called, the `MostAccessed` field MUST be populated with up to 10
documents ordered by descending `AccessCount`. Documents with `AccessCount == 0` or a nil
`LastAccessedAt` MUST be excluded from this list.

**FR-021** — "Most Accessed Documents" table in `doc(action: "audit")` output  
The MCP `doc(action: "audit")` tool handler MUST render the `AuditResult.MostAccessed`
list as a Markdown table labelled "Most Accessed Documents" (or equivalent heading) in the
audit output text. The table MUST show at minimum: rank, document path, access count, and
last accessed timestamp. When `MostAccessed` is empty, the section MUST be omitted from
the output.

---

## Non-Functional Requirements

**NFR-001 — Write latency**  
Counter update writes MUST NOT block the in-process response path. A background goroutine,
deferred flush, or in-memory accumulator with periodic write-through are all acceptable
implementations.

**NFR-002 — Error isolation**  
A failure in any counter update path (read, increment, write) MUST be silently absorbed.
The primary operation (`Get`, `List`, `GetOutline`, `GetSection`, `Find`, `Search`) MUST
return its normal result unaffected by counter errors.

**NFR-003 — Backward compatibility**  
All new YAML fields on `DocumentIndex` and `KnowledgeRecord.Fields` MUST have zero values
that are equivalent to "never accessed". No migration of existing files is required.

**NFR-004 — Approximation is acceptable**  
`recent_use_count` and `AccessCount` are allowed to be approximate. Lost increments due to
process crashes, concurrent writes, or flush timing are acceptable. The system MUST NOT use
pessimistic locking or compare-and-swap on every counter write.

**NFR-005 — No new required dependencies**  
This feature MUST NOT introduce new Go module dependencies. All instrumentation uses the
standard library and existing in-process infrastructure.

---

## Acceptance Criteria

- [ ] **FR-001** A knowledge entry loaded after first creation has `recent_use_count` equal
  to `0` (absent key in `Fields` treated as zero).
- [ ] **FR-002** A knowledge entry loaded after first creation has no `last_accessed_at` key
  in `Fields` (or a zero-value equivalent).
- [ ] **FR-003** After `KnowledgeService.Get("KE-xxx")` is called successfully, the stored
  record for `KE-xxx` has `recent_use_count` incremented by 1 and `last_accessed_at` set to
  a timestamp within a few seconds of the call.
- [ ] **FR-004** After `KnowledgeService.List(filters)` returns N entries, each of those N
  entries has `recent_use_count` incremented by 1 and `last_accessed_at` updated.
- [ ] **FR-005** `recent_use_count` decays entries older than 30 days (exact mechanism is
  implementation-defined; lazy recomputation at read time is acceptable).
- [ ] **FR-006** `KnowledgeFilters` has a `Sort` field; passing `Sort: "recent"` to
  `KnowledgeService.List` returns entries in descending `recent_use_count` order.
- [ ] **FR-007** The field map returned by `KnowledgeService.List` for each entry includes
  a `recent_use_count` key alongside `use_count`.
- [ ] **FR-008** `knowledge(action: "list", sort: "recent")` MCP call returns entries sorted
  by descending `recent_use_count`.
- [ ] **FR-009** `docint.DocumentIndex` has an `AccessCount int` field; a freshly constructed
  `DocumentIndex` has `AccessCount == 0`.
- [ ] **FR-010** `docint.DocumentIndex` has a `LastAccessedAt *time.Time` field; a freshly
  constructed `DocumentIndex` has `LastAccessedAt == nil`.
- [ ] **FR-011** A type with at least `AccessCount int` and `LastAccessedAt *time.Time` exists
  in `internal/docint/types.go`; `DocumentIndex` has a map field keyed by section path using
  this type.
- [ ] **FR-012** After `IntelligenceService.GetOutline(docID)` is called, the saved index for
  `docID` has `AccessCount` incremented by 1 and `LastAccessedAt` updated.
- [ ] **FR-013** After the guide action for a document is serviced, the saved index for that
  document has `AccessCount` incremented by 1.
- [ ] **FR-014** After `IntelligenceService.GetSection(docID, "3.2")` is called, the saved
  index for `docID` has document-level `AccessCount` incremented and `SectionAccess["3.2"]`
  has its `AccessCount` incremented and `LastAccessedAt` updated.
- [ ] **FR-015** After a `find` call returns results spanning documents D1 and D2, the saved
  indexes for D1 and D2 each have `AccessCount` incremented by 1.
- [ ] **FR-016** After `IntelligenceService.Search` returns results spanning documents D1 and
  D2, the saved indexes for D1 and D2 each have `AccessCount` incremented by 1.
- [ ] **FR-017** Counter write failures (e.g., simulated I/O error) do not cause `GetOutline`,
  `GetSection`, `FindByEntity`, or `Search` to return an error.
- [ ] **FR-018** An existing `DocumentIndex` YAML file that lacks `access_count`,
  `last_accessed_at`, and `section_access` fields loads without error and produces zero
  values for those fields.
- [ ] **FR-019** `AuditResult` in `internal/service/doc_audit.go` has a `MostAccessed` field
  of slice type; each element carries at minimum `DocID`, `Path`, `AccessCount`, and
  `LastAccessedAt`.
- [ ] **FR-020** `AuditDocuments` returns at most 10 entries in `MostAccessed`; entries are
  ordered by descending `AccessCount`; documents with `AccessCount == 0` are excluded.
- [ ] **FR-021** The `doc(action: "audit")` MCP response text includes a "Most Accessed
  Documents" table when `MostAccessed` is non-empty, showing rank, path, access count, and
  last-accessed timestamp; the section is absent when `MostAccessed` is empty.

---

## Dependencies and Assumptions

### Dependencies

| Dependency | Type | Notes |
|---|---|---|
| `work/design/doc-intel-adoption-design.md` | Design (approved) | Source of truth for §7.2 and §7.3 requirements |
| `internal/docint/types.go` | Source file | `DocumentIndex` struct to be extended |
| `internal/docint/store.go` | Source file | `SaveDocumentIndex` / `LoadDocumentIndex` used for lazy flush |
| `internal/storage/knowledge_store.go` | Source file | `KnowledgeRecord.Fields` schema |
| `internal/service/knowledge.go` | Source file | `KnowledgeService.List` and `Get` to be modified |
| `internal/service/intelligence.go` | Source file | Five methods to be modified |
| `internal/service/doc_audit.go` | Source file | `AuditResult` and `AuditDocuments` to be extended |
| `internal/mcp/knowledge_tool.go` | Source file | `knowledgeListAction` to expose `sort` parameter |

### Assumptions

1. **No SQLite migration in this feature.** The doc-intel enhancement design mentions moving
   counters to SQLite; this feature defers that. Counters live in the per-document YAML index
   files (`.kbz/index/docs/<doc-id>.yaml`) until the SQLite migration ships separately.

2. **Knowledge fields are stored as `map[string]any`.** `KnowledgeRecord.Fields` is a
   schema-free map. The new fields `recent_use_count` and `last_accessed_at` are stored as
   map entries, consistent with all existing fields. No new typed struct is required.

3. **`SectionAccessInfo` is a new type.** There is no existing `SectionIndex` type in the
   codebase. The per-section access data is a new type added to `internal/docint/types.go`.

4. **The `guide` action is serviced within `internal/service/intelligence.go`.** Inspection
   of `internal/mcp/doc_intel_tool.go` confirms that `docIntelGuideAction` calls into the
   intelligence service; the counter increment belongs in the service layer, not in the MCP
   handler.

5. **`recent_use_count` decay is lazy.** The 30-day window does not require a background job.
   A recomputation or reset at read time is sufficient.

6. **Concurrent write races are acceptable.** The system is single-process; the approximation
   tolerance in FR-004 (NFR-004) means lost-update races during a flush cycle do not need to
   be resolved with locking primitives.
```

Now let me register the document with the doc system: