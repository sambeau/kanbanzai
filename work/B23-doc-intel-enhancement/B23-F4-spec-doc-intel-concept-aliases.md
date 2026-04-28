# Specification: Concept Alias Resolution

**Feature:** FEAT-01KPNNYZ1ZSS6 — Concept Alias Resolution
**Design reference:** `work/design/doc-intel-enhancement-design.md` §7
**Status:** Draft

---

## Overview

This specification defines concept alias resolution for the document intelligence
subsystem. When agents classify document sections, they MAY declare aliases for
concepts they introduce (e.g. "throttling" as an alias for "rate-limiting"). The
`FindConcept` function MUST resolve queries against both canonical concept names and
their aliases, so that searching for any known surface form of a concept returns the
canonical concept entry. This completes the implementation of the `Concept.Aliases`
field that is currently declared but unused in `internal/docint/types.go`.

## Scope

### In scope

- Alias declaration during Layer 3 classification via an extended `concepts_intro`
  format
- Alias storage in the `Concept` struct and `ConceptRegistry`
- Alias resolution in `FindConcept` (canonical match first, then alias scan)
- Alias normalisation using the same rules as concept names
- Alias accumulation across independent classification submissions
- Backward compatibility with the existing plain-string `concepts_intro` format

### Out of scope

- Explicit alias removal or editing (aliases are managed through re-classification)
- Alias declaration on `concepts_used` entries (only `concepts_intro` supports aliases)
- Alias indexing or acceleration structures (linear scan is sufficient)
- UI or MCP tool changes for alias management
- Changes to `concepts_used` parsing — it remains `[]string`

## Functional Requirements

### FR-001: Extended concepts_intro format

The `concepts_intro` field in a classification submission MUST accept two value
forms for each entry:

1. **Plain string** — a concept name with no aliases (existing behaviour)
2. **Object** — a map with a required `name` field (string) and an optional
   `aliases` field (list of strings)

Both forms MAY appear in the same `concepts_intro` list.

*Design ref: §7.3*

**Acceptance criteria:**
- [ ] A classification with `concepts_intro: ["rate-limiting"]` is accepted and
  creates a concept with no aliases
- [ ] A classification with `concepts_intro: [{name: "rate-limiting", aliases: ["throttling"]}]`
  is accepted and creates a concept with the declared alias
- [ ] A classification mixing both forms in the same list is accepted

### FR-002: Backward compatibility for plain-string concepts_intro

When `concepts_intro` contains a plain string value, the system MUST process it
identically to current behaviour: the string is treated as a concept name with no
aliases. Existing classification data (if any) MUST NOT require migration.

*Design ref: §7.3*

**Acceptance criteria:**
- [ ] Existing tests for `UpdateConceptRegistry` with `[]string` concepts_intro
  continue to pass without modification
- [ ] A plain-string entry produces a `Concept` with an empty or nil `Aliases` field

### FR-003: Alias normalisation

Aliases MUST be normalised using the same rules as concept names: lowercase,
spaces and underscores replaced with hyphens, consecutive hyphens collapsed. This
is the existing `NormalizeConcept` function.

*Design ref: §7.3, §7.5*

**Acceptance criteria:**
- [ ] Alias "Rate Limiting" is stored as "rate-limiting"
- [ ] Alias "request_throttling" is stored as "request-throttling"
- [ ] Alias "  spaced  " is stored as "spaced"

### FR-004: Alias storage in ConceptRegistry

When a classification declares aliases for a concept, the aliases MUST be stored
in the concept's `Aliases` field after normalisation. The canonical concept name
MUST NOT appear in the `Aliases` list.

*Design ref: §7.3*

**Acceptance criteria:**
- [ ] After classifying with `{name: "rate-limiting", aliases: ["throttling", "request-throttling"]}`,
  the concept entry has `Aliases: ["throttling", "request-throttling"]`
- [ ] If an alias normalises to the same value as the canonical name, it is silently
  dropped rather than stored

### FR-005: Alias deduplication

Aliases MUST be deduplicated by normalised form. If the same alias (after
normalisation) is declared multiple times — whether in the same classification or
across separate classifications — it MUST appear only once in the `Aliases` list.

*Design ref: §7.5*

**Acceptance criteria:**
- [ ] Declaring alias "Throttling" when "throttling" already exists does not create
  a duplicate
- [ ] Two independent classifications each declaring alias "throttling" for the same
  concept result in a single "throttling" entry in `Aliases`

### FR-006: Alias accumulation across classifications

When multiple agents independently classify sections and declare different aliases
for the same concept, all unique aliases MUST be retained. Aliases from later
classifications MUST be merged with (not replace) existing aliases.

*Design ref: §7.5*

**Acceptance criteria:**
- [ ] Agent A classifies and declares alias "throttling" for "rate-limiting"
- [ ] Agent B later classifies and declares alias "request-cap" for "rate-limiting"
- [ ] The concept's `Aliases` list contains both "throttling" and "request-cap"

### FR-007: FindConcept canonical name resolution (existing behaviour)

`FindConcept` MUST first attempt to match the query against canonical concept names
(after normalisation). If a canonical match is found, it MUST be returned. This
preserves current behaviour.

*Design ref: §7.4*

**Acceptance criteria:**
- [ ] `FindConcept(registry, "rate-limiting")` returns the concept with canonical
  name "rate-limiting" when it exists
- [ ] Canonical match is case-insensitive via normalisation (existing behaviour)

### FR-008: FindConcept alias resolution

If no canonical name match is found, `FindConcept` MUST scan all concepts' alias
lists for a normalised match. If an alias match is found, the concept owning that
alias MUST be returned.

*Design ref: §7.4*

**Acceptance criteria:**
- [ ] Given concept "rate-limiting" with alias "throttling",
  `FindConcept(registry, "throttling")` returns the "rate-limiting" concept
- [ ] Alias matching is case-insensitive via normalisation
- [ ] `FindConcept(registry, "nonexistent")` returns nil when no canonical or alias
  match exists

### FR-009: Canonical name takes priority over aliases

If a query matches both a canonical concept name and an alias of a different concept,
the canonical match MUST be returned. Canonical names always take priority.

*Design ref: §7.4 (step ordering: canonical first, then alias scan)*

**Acceptance criteria:**
- [ ] Concept A has canonical name "throttling"; concept B has alias "throttling".
  `FindConcept(registry, "throttling")` returns concept A

### FR-010: FindConcept returns pointer to registry entry

`FindConcept` MUST continue to return a pointer to the concept within the registry
slice, allowing in-place mutation by callers. This is existing behaviour that MUST
NOT change.

*Design ref: implicit — existing API contract*

**Acceptance criteria:**
- [ ] Existing `TestFindConcept_MutatesInPlace` test continues to pass
- [ ] A concept found via alias match is also a mutable pointer into the registry

### FR-011: UpdateConceptRegistry processes aliases from extended format

`UpdateConceptRegistry` MUST extract aliases from object-form entries in
`concepts_intro` and store them on the corresponding concept in the registry.
For plain-string entries, no alias processing occurs.

*Design ref: §7.3*

**Acceptance criteria:**
- [ ] Calling `UpdateConceptRegistry` with an object-form `concepts_intro` entry
  populates the concept's `Aliases` field
- [ ] Calling `UpdateConceptRegistry` with a plain-string `concepts_intro` entry
  does not modify the concept's `Aliases` field

## Non-Functional Requirements

### NFR-001: Linear scan performance

Alias resolution in `FindConcept` MUST use a linear scan of the concept registry.
No indexing or caching structures are required. The concept registry is expected
to contain at most a few hundred entries; alias resolution MUST NOT introduce
additional data structures beyond the existing `Concept.Aliases` slice.

*Design ref: §7.4*

### NFR-002: No YAML schema migration

The change to `concepts_intro` parsing MUST NOT require migration of existing
stored classification data. The `Classification` struct's serialised form MUST
remain compatible with existing YAML files (if any exist).

*Design ref: §7.3 — backward compatibility*

## Acceptance Criteria

All requirements above include per-requirement acceptance criteria. The following
are integration-level criteria that span multiple requirements:

- [ ] **End-to-end alias flow:** An agent classifies a section with
  `concepts_intro: [{name: "rate-limiting", aliases: ["throttling"]}]`.
  A subsequent `FindConcept(registry, "throttling")` returns the "rate-limiting"
  concept with the section in its `IntroducedIn` list.
  *(FR-001, FR-004, FR-008, FR-011)*
- [ ] **Mixed format classification:** A single classification with both plain-string
  and object-form entries in `concepts_intro` correctly populates concepts and
  aliases. *(FR-001, FR-002, FR-011)*
- [ ] **Multi-agent accumulation:** Two `UpdateConceptRegistry` calls with different
  aliases for the same concept result in all aliases being present.
  *(FR-005, FR-006)*
- [ ] **All existing tests pass:** No regressions in `concepts_test.go`.
  *(FR-002, FR-007, FR-010)*

## Dependencies and Assumptions

- **Concept registry population:** This feature is most useful after the batch
  classification protocol (design §5) has been used to classify documents and
  populate the concept registry. Without classified documents, the concept registry
  is empty and alias resolution has nothing to resolve against.
- **Classification struct extension:** FR-001 requires changes to how `concepts_intro`
  is parsed. The underlying `Classification.ConceptsIntro` field type will need to
  change or a custom unmarshaller will need to be added. The serialised YAML format
  MUST remain backward-compatible (FR-002, NFR-002).
- **NormalizeConcept stability:** All alias normalisation relies on the existing
  `NormalizeConcept` function. This specification assumes its current behaviour
  (lowercase, hyphens, collapse) is stable and correct.
