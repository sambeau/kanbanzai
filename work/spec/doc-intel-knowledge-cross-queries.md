# Specification: Knowledge ↔ Document Cross-Queries

**Feature:** FEAT-01KPNNYYZRAEK - Knowledge-document cross-queries
**Design reference:** `work/design/doc-intel-enhancement-design.md` section 6
**Status:** Draft

---

## 1. Overview

This specification defines the requirements for cross-system query capabilities
between the document intelligence subsystem (`doc_intel`) and the knowledge base.
Two integration directions are specified: (1) enhancing the `doc_intel` `find`
action entity search to include related knowledge entries in its response, and
(2) enhancing the context assembly pipeline to surface lightweight document
section pointers alongside knowledge entries. Both directions are additive — they
extend existing response formats without modifying or removing existing fields.

---

## 2. Scope

### 2.1 In Scope

- Extending the `find(entity_id)` response in `doc_intel` to include related
  knowledge entries matched by `learned_from`, tags, and scope
- Extending the context assembly pipeline (`internal/mcp/assembly.go`) to
  include document section pointers alongside surfaced knowledge entries
- Additive response format changes to existing tool outputs
- Read-only cross-system access between the intelligence service and the
  knowledge store

### 2.2 Out of Scope

- **No bidirectional links.** This specification does not create persistent
  links between knowledge entries and document sections. Knowledge entry
  storage format is unchanged. (Design section 6.6, bullet 1)
- **No merged search.** Knowledge and document search remain separate systems
  with separate queries. There is no unified search action that queries both
  simultaneously. (Design section 6.6, bullet 2)
- **No changes to knowledge scoring.** The knowledge surfacing algorithm --
  including scoring, confidence thresholds, caps, and matching rules — is
  unchanged. (Design section 6.6, bullet 3)
- **No changes to `find(concept)` or `find(role)`.** Cross-query enhancement
  applies only to the `find(entity_id)` code path.
- **No changes to the `knowledge` MCP tool.** Cross-system integration is in
  the `doc_intel` tool handler and the context assembly pipeline, not in the
  knowledge tool surface. (Design section 8.2)

---

## 3. Functional Requirements

### Direction 1: Entity Search Includes Knowledge

**FR-001:** When `doc_intel(action: "find", entity_id: ...)` is called, the
response MUST include a `related_knowledge` field containing an array of
knowledge entries related to the queried entity. (Design section 6.3)

**Acceptance criteria:**
- [ ] Calling `find(entity_id: "FEAT-xxx")` returns a response containing a
  `related_knowledge` key
- [ ] The `related_knowledge` value is an array (possibly empty)
- [ ] Each element in the array is a knowledge entry object

---

**FR-002:** A knowledge entry MUST be included in `related_knowledge` if its
`learned_from` field contains the queried entity ID. (Design section 6.3, bullet 1)

**Acceptance criteria:**
- [ ] A knowledge entry with `learned_from: "FEAT-xxx"` appears in
  `related_knowledge` when `find(entity_id: "FEAT-xxx")` is called
- [ ] A knowledge entry with `learned_from: "TASK-yyy"` does NOT appear when
  querying `entity_id: "FEAT-xxx"`

---

**FR-003:** A knowledge entry MUST be included in `related_knowledge` if any of
its tags match the queried entity ID or entity type prefix. (Design section 6.3,
bullet 2)

**Acceptance criteria:**
- [ ] A knowledge entry tagged `["FEAT-xxx"]` appears in `related_knowledge`
  when `find(entity_id: "FEAT-xxx")` is called
- [ ] A knowledge entry tagged `["feature"]` appears in `related_knowledge`
  when `find(entity_id: "FEAT-xxx")` is called (entity type match)
- [ ] A knowledge entry tagged `["unrelated"]` does NOT appear when it has no
  other matching criteria

---

**FR-004:** A knowledge entry MUST be included in `related_knowledge` if its
scope matches any document that references the queried entity, as determined by
the `entity_refs` table. (Design section 6.3, bullet 3)

**Acceptance criteria:**
- [ ] Given document `doc-A` references entity `FEAT-xxx` (per `entity_refs`),
  a knowledge entry with `scope: "doc-A"` (or a path prefix matching `doc-A`)
  appears in `related_knowledge`
- [ ] A knowledge entry whose scope does not match any document referencing the
  entity does NOT appear (unless matched by FR-002 or FR-003)

---

**FR-005:** Knowledge entries with `status: "retired"` MUST NOT appear in
`related_knowledge`. (Consistent with existing knowledge surfacing in
`internal/knowledge/surface.go` `MatchEntries`.)

**Acceptance criteria:**
- [ ] A retired knowledge entry that would otherwise match by `learned_from` is
  excluded from `related_knowledge`

---

**FR-006:** Each entry in the `related_knowledge` array MUST include the
following fields: `id`, `topic`, `content`, `confidence`, `status`.
(Design section 6.3, response format example)

**Acceptance criteria:**
- [ ] Every element in `related_knowledge` contains all five fields
- [ ] Field values correspond to the knowledge entry's stored values

---

**FR-007:** The `related_knowledge` array MUST be deduplicated by knowledge
entry ID. A knowledge entry that matches multiple criteria (e.g. both
`learned_from` and tags) MUST appear at most once. (Consistent with
`MatchEntries` deduplication.)

**Acceptance criteria:**
- [ ] A knowledge entry matching by both `learned_from` and tag appears exactly
  once in `related_knowledge`

---

**FR-008:** The response MUST include a `knowledge_matches` field containing the
integer count of entries in `related_knowledge`. (Design section 6.3, response format)

**Acceptance criteria:**
- [ ] `knowledge_matches` equals the length of the `related_knowledge` array
- [ ] When no knowledge entries match, `knowledge_matches` is `0`

---

**FR-009:** The existing response fields (`search_type`, `entity_id`, `count`,
`matches`) MUST NOT be modified or removed. The `count` field MUST continue to
reflect only the number of document section matches. (Design section 6.5)

**Acceptance criteria:**
- [ ] The `search_type` field remains `"entity_id"`
- [ ] The `count` field equals the length of the `matches` array (document
  sections only)
- [ ] Existing consumers that ignore `related_knowledge` and
  `knowledge_matches` observe identical behaviour to the pre-enhancement
  response

---

**FR-010:** The `doc_intel` tool handler for the `find` action MUST receive
read access to the knowledge store (or a query interface) to resolve related
knowledge entries. (Design section 6.5)

**Acceptance criteria:**
- [ ] The `docIntelFindAction` function signature or closure accepts a
  knowledge query dependency
- [ ] The dependency is injectable for testing (interface or function type)

---

### Direction 2: Context Assembly Surfaces Document Pointers

**FR-011:** The context assembly pipeline MUST include a document pointers
section in the assembled context when knowledge entries reference entity IDs
that appear in indexed documents. (Design section 6.4)

**Acceptance criteria:**
- [ ] When assembled context includes knowledge entries, and those entries
  `learned_from` or tag values match entity IDs present in the `entity_refs`
  table, the assembled context includes document pointers
- [ ] When no knowledge entries reference entities with indexed documents, no
  document pointers section is emitted

---

**FR-012:** Each document pointer MUST include the document path and section
path. The pointer SHOULD include the section title when available.
(Design section 6.4)

**Acceptance criteria:**
- [ ] Each document pointer contains at minimum a document path and a section
  path
- [ ] When the intelligence service provides a section title, it is included in
  the pointer

---

**FR-013:** Document pointers MUST NOT include the full section content. They
are lightweight references that direct the agent to read the section if needed.
(Design section 6.4 — "document path + section path, not full content")

**Acceptance criteria:**
- [ ] No document pointer in the assembled context contains section body text
- [ ] The agent receives enough information to call
  `doc_intel(action: "section", ...)` to retrieve the content

---

**FR-014:** The document pointers step MUST execute after the knowledge
surfacing step (`asmLoadKnowledge`) in the assembly pipeline, since it depends
on the surfaced knowledge entries to determine which entities to query.
(Design section 6.4 — "The assembly step that surfaces knowledge already has
access to the intelligence service")

**Acceptance criteria:**
- [ ] Document pointers are derived from the knowledge entries already selected
  by `asmLoadKnowledge`
- [ ] The step uses the existing `intelligenceSvc` reference available in the
  assembly pipeline

---

**FR-015:** The context assembly pipeline MUST NOT modify the knowledge
surfacing algorithm. The set of knowledge entries surfaced MUST remain identical
to the pre-enhancement behaviour. (Design section 6.6, bullet 3)

**Acceptance criteria:**
- [ ] The output of `asmLoadKnowledge` is unchanged
- [ ] Knowledge entry filtering, scoring, and ordering logic is not modified

---

### Response Format

**FR-016:** All response format changes MUST be additive. New fields are added
to existing response maps. No existing field is renamed, removed, or
retyped. (Design section 6.5)

**Acceptance criteria:**
- [ ] The `find(entity_id)` response is a superset of the pre-enhancement
  response
- [ ] The assembled context struct gains new fields without modifying existing
  field types or names

---

## 4. Non-Functional Requirements

**NFR-001:** Each document pointer in the assembled context SHOULD consume
approximately 20 tokens. A typical task surfacing 3-5 document pointers
alongside 5-10 knowledge entries SHOULD add no more than 100 tokens to the
assembled context. (Design section 6.4)

**Acceptance criteria:**
- [ ] A document pointer rendered as text (path + section path + optional
  title) is no more than 30 tokens when measured by a standard tokeniser
- [ ] A context assembly with 5 document pointers adds no more than 150 tokens
  to the byte usage count

---

**NFR-002:** The knowledge query in `find(entity_id)` MUST NOT degrade the
latency of the existing document section lookup. The total response time for
`find(entity_id)` SHOULD remain under 50ms for a typical corpus.

**Acceptance criteria:**
- [ ] Benchmark tests confirm `find(entity_id)` completes within 50ms on a
  corpus of 280 documents and 100 knowledge entries

---

## 5. Acceptance Criteria

Summary checklist for feature completion:

### Direction 1 — Entity search includes knowledge
- [ ] `find(entity_id)` returns `related_knowledge` array (FR-001)
- [ ] Matching by `learned_from` works (FR-002)
- [ ] Matching by tags works (FR-003)
- [ ] Matching by scope-to-document works (FR-004)
- [ ] Retired entries excluded (FR-005)
- [ ] Entry format correct (FR-006)
- [ ] Deduplication by ID (FR-007)
- [ ] `knowledge_matches` count present (FR-008)
- [ ] Existing fields unchanged (FR-009)
- [ ] Knowledge store dependency injected (FR-010)

### Direction 2 — Context assembly surfaces document pointers
- [ ] Document pointers appear when knowledge references indexed entities
  (FR-011)
- [ ] Pointers include path and section path (FR-012)
- [ ] Pointers do not include full content (FR-013)
- [ ] Pointer step runs after knowledge surfacing (FR-014)
- [ ] Knowledge surfacing algorithm unchanged (FR-015)

### Response format
- [ ] All changes are additive (FR-016)

### Non-functional
- [ ] Token cost per pointer no more than 30 tokens (NFR-001)
- [ ] Response latency no more than 50ms (NFR-002)

---

## 6. Dependencies and Assumptions

### Dependencies

1. **SQLite `entity_refs` table (Enhancement 2, Phase 1).** FR-004 and FR-011
   depend on the `entity_refs` table being populated and queryable. This table
   is defined in Design section 4.4 and is part of the SQLite-backed graph storage
   enhancement. Without this table, scope-to-document matching (FR-004) and
   entity-to-document pointer resolution (FR-011) cannot be implemented
   efficiently.

2. **Intelligence service access in `doc_intel` tool handler.** The `find`
   action handler (`docIntelFindAction` in `internal/mcp/doc_intel_tool.go`)
   currently receives only `*service.IntelligenceService`. FR-010 requires it
   to also receive a knowledge query interface.

3. **Intelligence service access in context assembly.** The assembly pipeline
   (`internal/mcp/assembly.go`) already receives `intelligenceSvc` via
   `asmInput`. No new service dependency is required for Direction 2 (FR-011
   through FR-015), provided the intelligence service exposes entity reference
   queries.

### Assumptions

1. Knowledge entries are stored as YAML files under `.kbz/state/knowledge/`
   and are accessible via the existing `KnowledgeService.List` API with field
   filtering.

2. The `learned_from` field on knowledge entries is a single string value
   containing a task or entity ID (e.g. `"TASK-01KMKR..."`). It is not an
   array.

3. The entity type prefix can be derived from the entity ID format (e.g.
   `FEAT-` → `"feature"`, `TASK-` → `"task"`) for tag-based
   matching in FR-003.

4. The `entity_refs` table index on `entity_id` provides sub-millisecond
   lookup for the document set associated with a given entity.
