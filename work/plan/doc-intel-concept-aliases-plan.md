# Dev Plan: Concept Alias Resolution

> Feature: FEAT-01KPNNYZ1ZSS6 — Concept Alias Resolution
> Spec: work/spec/doc-intel-concept-aliases.md

---

## Overview

This plan implements `work/spec/doc-intel-concept-aliases.md`. It completes the
declared-but-unused `Concept.Aliases` field in `internal/docint/types.go` by:
1. Extending the `ConceptsIntro` field in `Classification` to accept both plain strings
   and objects (`{name, aliases}`).
2. Storing and accumulating aliases in `UpdateConceptRegistry`.
3. Resolving aliases in `FindConcept` (canonical match first, then alias scan).
4. Updating `BuildGraphEdges` to extract concept names from the extended format.

No public API surface changes. Fully backward-compatible with existing classification
data.

---

## Task Breakdown

### Task 1: Extend ConceptsIntro type with custom YAML unmarshalling

**Description:** Change `Classification.ConceptsIntro` in `internal/docint/types.go`
from `[]string` to a new type that accepts both plain strings and objects
`{name: string, aliases: []string}`. Implement custom `yaml.Unmarshaler` on the new
type. The serialised YAML format must remain backward-compatible.

**Files:** `internal/docint/types.go`, `internal/docint/classifier.go` (if needed),
`internal/docint/concepts_test.go`

**Deliverable:**
- `ConceptIntroEntry` type (or equivalent) with custom `UnmarshalYAML`
- Plain-string entries parse to an entry with no aliases
- Object-form entries parse to an entry with name and aliases
- Mixed lists parse correctly
- No existing tests break

**Traceability:** FR-001, FR-002, FR-003, NFR-002

### Task 2: Update UpdateConceptRegistry for alias storage

**Description:** Modify `UpdateConceptRegistry` in `internal/docint/concepts.go` to
process the new `ConceptIntroEntry` type. For each entry, extract the canonical name
and any declared aliases. Store aliases in `Concept.Aliases` after normalisation.
Deduplicate aliases. Accumulate aliases across multiple calls.

**Files:** `internal/docint/concepts.go`, `internal/docint/concepts_test.go`

**Deliverable:**
- `Concept.Aliases` populated from object-form `concepts_intro` entries
- Plain-string entries produce concepts with empty aliases
- Aliases deduplicated by normalised form
- Multiple classification calls for the same concept merge (not replace) aliases

**Traceability:** FR-004, FR-005, FR-006, FR-011

### Task 3: Update FindConcept with alias resolution

**Description:** Modify `FindConcept` in `internal/docint/concepts.go` to first
check canonical names (existing behaviour) and then scan aliases if no canonical
match is found. Canonical names always take priority over aliases.

**Files:** `internal/docint/concepts.go`, `internal/docint/concepts_test.go`

**Deliverable:**
- `FindConcept(registry, "throttling")` returns the concept with alias "throttling"
- Canonical name match takes precedence over alias match
- Returns nil when no canonical or alias match
- Existing `TestFindConcept_*` tests still pass

**Traceability:** FR-007, FR-008, FR-009, FR-010, NFR-001

### Task 4: Update BuildGraphEdges for extended ConceptsIntro

**Description:** Modify `BuildGraphEdges` in `internal/docint/graph.go` to extract
the canonical concept name from `ConceptIntroEntry` values when building INTRODUCES
edges.

**Files:** `internal/docint/graph.go`, `internal/docint/graph_test.go`

**Deliverable:**
- INTRODUCES edges built from extended format entries use the canonical name
- Existing graph edge tests still pass

**Traceability:** (structural correctness for FR-001, FR-011)

---

## Dependency Graph

```
T1 → T2, T3, T4   (type change must land before consumers)
T2, T3, T4 are independent of each other (disjoint write scopes)
```

---

## Interface Contracts

All public function signatures are unchanged:

- `FindConcept(registry *ConceptRegistry, name string) *Concept`
- `UpdateConceptRegistry(registry *ConceptRegistry, docID string, classifications []Classification)`
- `BuildGraphEdges(index *DocumentIndex) []GraphEdge`

The `Classification.ConceptsIntro` field type changes from `[]string` to a custom
type that is backward-compatible in YAML. Callers that iterate over `ConceptsIntro`
entries and call a `.Name()` method (or equivalent) will need updating — this is
limited to `concepts.go` and `graph.go`.

---

## Traceability Matrix

| Task | Requirements |
|------|-------------|
| T1   | FR-001, FR-002, FR-003, NFR-002 |
| T2   | FR-004, FR-005, FR-006, FR-011 |
| T3   | FR-007, FR-008, FR-009, FR-010, NFR-001 |
| T4   | (graph integrity for FR-001, FR-011) |
