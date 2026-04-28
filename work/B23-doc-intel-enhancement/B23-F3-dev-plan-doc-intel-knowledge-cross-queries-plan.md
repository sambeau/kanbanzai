# Dev Plan: Knowledge-Document Cross-Queries

> Feature: FEAT-01KPNNYYZRAEK — Knowledge-document cross-queries
> Spec: work/spec/doc-intel-knowledge-cross-queries.md

---

## Overview

This plan implements `work/spec/doc-intel-knowledge-cross-queries.md`. Two integration
directions:

1. **Direction 1:** `doc_intel(action: "find", entity_id: ...)` includes a
   `related_knowledge` array and `knowledge_matches` count in the response.
2. **Direction 2:** The context assembly pipeline surfaces document section pointers
   alongside knowledge entries in `handoff`/`next` assembled context.

Depends on FEAT-01KPNNYYSQTA7 (SQLite entity_refs table) for efficient entity-to-
document lookups (FR-004 and Direction 2 pointer resolution).

---

## Task Breakdown

### Task 1: Inject knowledge query dependency into doc_intel find action

**Description:** Define a `KnowledgeQuerier` interface in `internal/mcp/` (or
`internal/service/`) that the `find` action can use to query knowledge entries. Update
the MCP server wiring (`internal/mcp/server.go`) to inject the knowledge service as a
`KnowledgeQuerier`. Update `docIntelTool` and `docIntelFindAction` to accept and use
this dependency.

**Files:** `internal/mcp/doc_intel_tool.go`, `internal/mcp/server.go`

**Deliverable:**
- `KnowledgeQuerier` interface with a method to list entries by learned_from, tags, scope
- `docIntelFindAction` accepts the querier as a closure argument
- Server wiring updated
- Interface injectable for tests

**Traceability:** FR-010

### Task 2: Match and return related_knowledge in find(entity_id)

**Description:** When `find(entity_id)` is called, query the knowledge base for entries
matching the entity ID by: (a) `learned_from` equals entityID, (b) tags contain
entityID or the entity type prefix, (c) scope matches a document that references the
entity (using the `entity_refs` SQLite table). Deduplicate by entry ID. Exclude retired
entries. Return `related_knowledge` array and `knowledge_matches` count.

**Files:** `internal/mcp/doc_intel_tool.go`, `internal/mcp/doc_intel_tool_test.go`

**Deliverable:**
- `find(entity_id)` response has `related_knowledge: [...]` and `knowledge_matches: N`
- Matching by learned_from, tags (entity ID + entity type), and scope-to-document
- Retired entries excluded
- Entries deduplicated by ID
- Each entry has: id, topic, content, confidence, status
- Existing response fields unchanged
- Tests covering each match criterion

**Traceability:** FR-001 – FR-009, NFR-002

### Task 3: Document pointers in context assembly

**Description:** Add a new assembly step `asmDocumentPointers` in
`internal/mcp/assembly.go` that runs after `asmLoadKnowledge`. It collects entity IDs
from the surfaced knowledge entries (learned_from and tags), queries
`IntelligenceService` for sections referencing those entities, and adds a lightweight
document pointer list to the assembled context. Each pointer: document path, section
path, optional section title.

**Files:** `internal/mcp/assembly.go`, `internal/mcp/next_tool.go` or
`internal/mcp/handoff_tool.go` (wherever assembled context is rendered)

**Deliverable:**
- When knowledge entries reference entities with indexed sections, document pointers
  appear in assembled context
- Each pointer: document path + section path (+ title when available)
- No full section content in pointers
- Pointers step runs after knowledge surfacing
- Knowledge surfacing algorithm unchanged
- Tests: pointers present when expected; absent when no indexed sections

**Traceability:** FR-011 – FR-016, NFR-001

---

## Dependency Graph

```
External: FEAT-01KPNNYYSQTA7 must be merged before T2 and T3
  (entity_refs table required for scope-to-document matching and pointer resolution)

T1 → T2   (dependency injection before use)
T3 is independent of T1 and T2 (different code path)
```

---

## Interface Contracts

### KnowledgeQuerier interface (new)

```go
type KnowledgeQuerier interface {
    QueryRelatedKnowledge(entityID string) ([]KnowledgeEntry, error)
}
```

### find(entity_id) response change (additive)

```json
{
  "search_type": "entity_id",
  "entity_id": "FEAT-xxx",
  "count": 3,
  "matches": [...],
  "related_knowledge": [
    {"id": "KE-xxx", "topic": "...", "content": "...", "confidence": 0.9, "status": "confirmed"}
  ],
  "knowledge_matches": 1
}
```

### AssembledContext change (additive)

```go
type AssembledContext struct {
  // ... existing fields ...
  DocumentPointers []DocumentPointer `json:"document_pointers,omitempty"`
}

type DocumentPointer struct {
  DocumentPath string `json:"document_path"`
  SectionPath  string `json:"section_path"`
  SectionTitle string `json:"section_title,omitempty"`
}
```

---

## Traceability Matrix

| Task | Requirements |
|------|-------------|
| T1   | FR-010 |
| T2   | FR-001 – FR-009, NFR-002 |
| T3   | FR-011 – FR-016, NFR-001 |
