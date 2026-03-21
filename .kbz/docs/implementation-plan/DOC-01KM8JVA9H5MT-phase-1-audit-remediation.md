---
id: DOC-01KM8JVA9H5MT
type: implementation-plan
title: Phase 1 Audit Remediation
status: submitted
feature: FEAT-01KM8JTF0VP0K
created_by: human
created: 2026-03-21T16:14:58Z
updated: 2026-03-21T16:14:58Z
---
# Phase 1 Audit Remediation Plan

- Status: active
- Date: 2025-07-17
- Purpose: define the work needed to close gaps found during the Phase 1 implementation audit
- Related:
  - `work/spec/phase-1-specification.md`
  - `work/plan/phase-1-implementation-plan.md`
  - `work/plan/phase-1-decision-log.md`

---

## 1. Purpose

This document is an addendum to `phase-1-implementation-plan.md`.

After completing Tracks A–E (core model, validation, ID allocation, document lifecycle, and MCP interface), an audit of the implementation against the Phase 1 specification identified bugs, spec compliance gaps, and test coverage weaknesses.

This plan organises the remediation work into tracks that can be executed systematically. Each track groups related fixes by area. Tracks are ordered by priority: bugs and spec violations first, then quality and coverage improvements.

---

## 2. Relationship to the Implementation Plan

The original plan defines Tracks A–H. Tracks A–E are implemented. Tracks F (CLI), G (local cache), and H (bootstrap) remain.

This addendum defines Tracks R1–R6 (remediation tracks) that should be completed before resuming Track G or H. Track F (CLI) is already partially done and has no audit findings beyond minor issues noted in R5.

The remediation tracks do not change the scope of Phase 1. They fix the implementation to match the existing specification.

---

## 3. Audit Findings Summary

### 3.1 Bugs

| ID | Severity | Summary | Location |
|---|---|---|---|
| B1 | High | `Decision.Status` has `omitempty` YAML tag on a required field | `internal/model/entities.go` |
| B2 | High | `Get` constructs return paths using `core.StatePath()` instead of instance root | `internal/service/entities.go` |
| B3 | High | Document ID counter resets on restart — collision risk | `internal/document/service.go` |
| B4 | Medium | `removeIfDifferent` is a no-op — title changes leave orphaned files | `internal/document/service.go` |
| B5 | Low | `quoteString` double-escapes consecutive backslashes | `internal/storage/entity_store.go` |
| B6 | Medium | `needsQuotes` doesn't catch embedded newlines | `internal/storage/entity_store.go` |
| B7 | Low | `needsQuotes` doesn't catch YAML 1.1 boolean variants (`True`, `Yes`, `On`, etc.) | `internal/storage/entity_store.go` |
| B8 | Medium | No referential integrity enforcement at entity creation time | `internal/service/entities.go` |

### 3.2 Spec compliance gaps

| ID | Spec § | Summary |
|---|---|---|
| S1 | §14.4 | Local derived cache not implemented (required) |
| S2 | §21.2 | No general-purpose entity field update for error correction |
| S3 | §16.2 | No search/query beyond list-by-type |
| S4 | §17.2 | Link resolution support not implemented, not explicitly deferred |
| S5 | §17.3 | Duplicate detection support not implemented, not explicitly deferred |
| S6 | §15.5 | Document-to-entity extraction not implemented (agent-mediated, but no tooling support) |

### 3.3 Test coverage gaps

| ID | Summary |
|---|---|
| T1 | `internal/model` has zero tests (no YAML round-trip tests for entity structs) |
| T2 | MCP package at ~49% — only epic creation tested, 4 of 5 entity types untested through MCP |
| T3 | No negative tests for `EntityStore.Load` (corrupt file, missing file, permissions) |
| T4 | Service `Get` path bug hidden by test asserting wrong value |
| T5 | No lifecycle transition tests for Epic, Task, or Decision through the service layer |
| T6 | No tests for `ListByType`/`ListAll` silently skipping corrupt documents |

### 3.4 Code quality issues

| ID | Summary |
|---|---|
| Q1 | Duplicated `jsonResult` / `marshalResult` helpers in MCP package |
| Q2 | No sentinel errors in service layer |
| Q3 | `EntityStore.Load` requires caller to know slug — can't look up by ID alone |
| Q4 | No `context.Context` forwarding from MCP handlers to services |
| Q5 | `list_documents` MCP tool missing `doc_type` enum constraint |
| Q6 | `validate_candidate` MCP tool missing `entity_type` enum constraint |
| Q7 | No atomic file writes (write-to-temp-then-rename) |

---

## 4. Remediation Tracks

### 4.1 Track R1 — Bug fixes (high priority)

Goal: Fix all identified bugs.

Fixes B1–B8 from the audit.

Tasks:

1. **Fix B1: Remove `omitempty` from `Decision.Status` YAML tag.**
   - File: `internal/model/entities.go`
   - Change `yaml:"status,omitempty"` to `yaml:"status"`
   - Add a test in `internal/model` that verifies a Decision with empty status still serialises the `status` key (this is a prerequisite for T1)

2. **Fix B2: Use instance root in `Get` path construction.**
   - File: `internal/service/entities.go`
   - Replace `core.StatePath()` with `s.root` in the `Get` method
   - Fix the corresponding test (`TestEntityService_Get_ReturnsStoredEntity`) to assert against the temp dir root, not `core.StatePath()`

3. **Fix B3: Scan existing documents to initialise the ID counter.**
   - File: `internal/document/service.go`
   - On `NewDocService` construction (or lazily on first allocation), scan existing document files to find the maximum `DOC-NNN` value
   - Initialise the counter to `max + 1`
   - Add a test: create two `DocService` instances against the same directory, submit documents through each, verify no ID collision

4. **Fix B4: Implement `removeIfDifferent` or document limitation.**
   - File: `internal/document/service.go`
   - Implement the function: if the old path differs from the new path after a title change, remove the old file
   - Add a test that submits a document, normalises it with a different title, and verifies the old file is removed

5. **Fix B5: Fix `quoteString` backslash escaping.**
   - File: `internal/storage/entity_store.go`
   - The replacer should escape a single `\` to `\\`, not `\\` to `\\\\`
   - Add round-trip test with consecutive backslashes (`\\`, `\\\`)

6. **Fix B6: Handle embedded newlines in `needsQuotes`.**
   - File: `internal/storage/entity_store.go`
   - Add `\n` and `\r` to the set of characters that trigger quoting
   - Add round-trip test with a string containing an embedded newline

7. **Fix B7: Handle YAML 1.1 boolean variants in `needsQuotes`.**
   - File: `internal/storage/entity_store.go`
   - Add case-insensitive checks for `true`, `false`, `yes`, `no`, `on`, `off`, `null`, `~`
   - Add round-trip tests for `"True"`, `"YES"`, `"On"`, `"Off"`

8. **Fix B8: Validate referential integrity at creation time.**
   - File: `internal/service/entities.go`
   - `CreateFeature`: verify the referenced epic exists before writing
   - `CreateTask`: verify the referenced feature exists before writing
   - Add tests for both: creating a feature referencing a non-existent epic must fail, creating a task referencing a non-existent feature must fail

Outputs:
- All 8 bugs fixed with regression tests
- Deterministic YAML round-trips safe for real-world string values

### 4.2 Track R2 — Entity field update for error correction

Goal: Satisfy spec §21.2 and implementation plan §8.3.

The spec requires the ability to correct wrong field values, wrong links, and wrong normalisation results after commit. Currently the only mutation operation is `update_status`. This track adds a general-purpose entity update.

Tasks:

1. **Implement `UpdateEntity` in the service layer.**
   - File: `internal/service/entities.go`
   - Accept entity type, ID, slug, and a partial map of fields to update
   - Load the existing entity, merge the provided fields, validate the result, write it back
   - Reject changes to `id` (immutable)
   - Reject changes to `status` (must use `UpdateStatus` for lifecycle-validated transitions)
   - Validate all updated fields against the entity schema (required fields still present, enums still valid, references still well-formed)

2. **Implement `update_entity` MCP tool.**
   - File: `internal/mcp/entity_tools.go`
   - Accept `entity_type`, `id`, `slug`, and additional field arguments
   - Delegate to `EntityService.UpdateEntity`
   - Return the updated entity

3. **Add CLI `update fields` command.**
   - File: `cmd/kanbanzai/main.go`
   - `kbz update fields --type feature --id FEAT-001 --slug my-feature --epic E-002`

4. **Add tests.**
   - Correct a wrong epic reference on a feature
   - Correct a wrong `observed` field on a bug
   - Reject an attempt to change `id`
   - Reject an attempt to change `status` through this path
   - Validate that updated result still passes schema validation

Outputs:
- General-purpose field correction for all entity types
- MCP + CLI surface for error correction

### 4.3 Track R3 — Test coverage improvements

Goal: Close the test coverage gaps identified in the audit.

Tasks:

1. **T1: Add `internal/model` tests.**
   - Create `internal/model/entities_test.go`
   - YAML round-trip tests for all five entity types (marshal → unmarshal → compare)
   - Verify `Entity` interface satisfaction at compile time (type assertions)
   - Assert enum constant string values to guard against accidental changes

2. **T2: Add MCP tests for all entity creation tools.**
   - File: `internal/mcp/server_test.go`
   - Add tests for `create_feature`, `create_task`, `create_bug`, `record_decision`
   - Add negative tests: missing required arguments, invalid enum values
   - Use table-driven tests to reduce boilerplate

3. **T3: Add negative tests for `EntityStore.Load`.**
   - File: `internal/storage/entity_store_test.go`
   - Test: load a non-existent file → clear error
   - Test: load a corrupt YAML file → clear error
   - Test: load an empty file → clear error

4. **T4: Fix the `Get` path test.**
   - Covered by B2 fix in Track R1 — verify the test asserts against the actual root, not the global constant

5. **T5: Add service-layer lifecycle tests for Epic, Task, and Decision.**
   - File: `internal/service/entities_test.go`
   - Walk each entity type through a representative lifecycle chain via `UpdateStatus`
   - Verify terminal states reject further transitions

6. **T6: Test document list behaviour with corrupt files.**
   - File: `internal/document/service_test.go`
   - Write a corrupt file directly to the store directory
   - Call `ListByType` and verify it returns the valid documents without error
   - Optionally: consider whether silent skipping should become a logged warning

Outputs:
- `internal/model` coverage from 0% to meaningful
- MCP coverage from ~49% to ~75%+
- Negative test paths covered in storage and document packages
- All five entity types exercised through service-layer lifecycle transitions

### 4.4 Track R4 — Code quality improvements

Goal: Address the code quality issues that affect correctness or maintainability.

Tasks:

1. **Q1: Unify MCP JSON marshalling helpers.**
   - Remove `marshalResult` from `document_tools.go`
   - Use `jsonResult` from `entity_tools.go` (or extract to a shared unexported function in the package)

2. **Q2: Define sentinel errors in the service layer.**
   - File: `internal/service/errors.go` (new)
   - Define: `ErrNotFound`, `ErrInvalidTransition`, `ErrValidationFailed`, `ErrReferenceNotFound`
   - Update service methods to return these (wrapped with context via `fmt.Errorf("...: %w", sentinel)`)
   - Update tests to check for sentinel errors where appropriate

3. **Q3: Support entity lookup by ID without requiring slug.**
   - File: `internal/storage/entity_store.go`
   - Add a `FindByID(entityType, id string) (EntityRecord, error)` method that scans the entity directory for a file matching the ID prefix
   - This unblocks more natural `get_entity` usage where the caller may not know the slug

4. **Q4: Forward `context.Context` from MCP handlers to services.**
   - Add `context.Context` as the first parameter to service methods that may do I/O
   - Thread the context through from MCP handlers
   - This is a mechanical change — no new cancellation logic is needed yet, but the plumbing must exist

5. **Q5 + Q6: Add missing enum constraints to MCP tool definitions.**
   - Add `mcp.Enum(docTypeEnum...)` to `list_documents` `doc_type` parameter
   - Add `mcp.Enum("epic", "feature", "task", "bug", "decision")` to `validate_candidate` `entity_type` parameter

6. **Q7: Implement atomic file writes.**
   - File: `internal/storage/entity_store.go` and `internal/document/store.go`
   - Replace `os.WriteFile` with write-to-temp-then-rename pattern
   - Add a shared helper (e.g., `internal/fsutil/atomic.go`) or inline in each store

Outputs:
- Cleaner MCP layer
- Programmatic error handling via sentinel errors
- Entity lookup by ID alone
- Context propagation plumbing in place
- Atomic writes prevent corruption on crash

### 4.5 Track R5 — Spec compliance deferrals

Goal: Explicitly defer spec requirements that are out of scope for the current implementation push, as the spec requires.

The following items are specified with "should" language and explicit permission to defer. They should be recorded as deferred decisions rather than left as silent omissions.

Tasks:

1. **Record deferral of link resolution support (S4, spec §17.2).**
   - Add a decision entry `P1-DEC-017` to the decision log
   - Status: accepted
   - Decision: Link resolution from loose references is deferred to Phase 2. Phase 1 validates explicit references but does not infer likely links from free text.

2. **Record deferral of duplicate detection support (S5, spec §17.3).**
   - Add a decision entry `P1-DEC-018` to the decision log
   - Status: accepted
   - Decision: Duplicate detection for bug/feature creation is deferred to Phase 2. Phase 1 relies on health checks and human review.

3. **Record deferral of document-to-entity extraction (S6, spec §15.5).**
   - Add a decision entry `P1-DEC-019` to the decision log
   - Status: accepted
   - Decision: Automated extraction of entities from approved documents is deferred. Phase 1 provides the document lifecycle and entity creation tools; extraction is performed by agents using these tools manually. Phase 2 may add dedicated extraction tooling.

4. **Record search/query scope limitation (S3, spec §16.2).**
   - Add a decision entry `P1-DEC-020` to the decision log
   - Status: accepted
   - Decision: Phase 1 entity query is limited to list-by-type. Attribute filtering and text search are deferred to the local cache track (Track G / P1-DEC-013). The `list_entities` operation satisfies the minimum "search/query" requirement when combined with client-side filtering by agents.

Outputs:
- Four new accepted decisions in the decision log
- No silent spec omissions

### 4.6 Track R6 — Local derived cache (Track G from original plan)

Goal: Satisfy spec §14.4.

This is the original Track G, not a new track. It is listed here for completeness because the audit identified it as a hard spec requirement that is not yet implemented. The scope should be determined by resolving P1-DEC-013 (currently `proposed`).

Tasks:

1. **Resolve P1-DEC-013: decide cache scope.**
   - What queries must the cache support in Phase 1?
   - Minimum viable: entity lookup by ID (without slug), entity search by field values, health check acceleration
   - Implementation: SQLite file in `.kbz/cache/` (rebuildable, not committed)

2. **Implement cache rebuild from canonical state.**
   - Scan all entity files and document files
   - Populate SQLite tables
   - Provide a `rebuild_cache` MCP tool and `kbz cache rebuild` CLI command

3. **Use cache for query acceleration.**
   - `Get` by ID (without slug) can use the cache to find the filename
   - `List` can use the cache for faster enumeration
   - `HealthCheck` can use the cache to avoid repeated directory scans

4. **Ensure cache loss is harmless.**
   - All operations must fall back to filesystem reads if the cache is missing
   - Cache is `.gitignore`d

Outputs:
- Rebuildable SQLite cache
- Faster queries and health checks
- P1-DEC-013 resolved

---

## 5. Recommended Execution Order

The tracks have the following dependencies and priority:

```
R1 (bug fixes)          — no dependencies, highest priority
R3 (test coverage)      — depends on R1 (B2 fix changes a test)
R2 (entity update)      — independent, can run parallel with R3
R4 (code quality)       — independent, can run parallel with R2/R3
R5 (deferrals)          — independent, can run any time
R6 (local cache)        — depends on R1 and R4 (Q3 is a prerequisite for cache-backed lookup)
```

Recommended sequence:

1. **R1** — Fix all bugs first. These affect correctness.
2. **R5** — Record deferrals. This is documentation-only and removes ambiguity.
3. **R3 + R2 + R4** — These three can run in parallel or interleaved:
   - R3 (tests) is safe to do alongside R2 and R4 since it touches `*_test.go` files
   - R2 (entity update) adds new service + MCP + CLI code
   - R4 (code quality) refactors existing code
   - If parallelising, assign R2 and R4 to different files to avoid merge conflicts
4. **R6** — Local cache. This is the largest remaining track and should be last.

---

## 6. Estimated Scope

| Track | Estimated size | New files | Modified files |
|---|---|---|---|
| R1 | Small–Medium | 0 | 6–8 |
| R2 | Medium | 1 (`service/errors.go` or combined) | 4–5 |
| R3 | Medium | 1 (`model/entities_test.go`) | 4–5 test files |
| R4 | Medium | 2 (`service/errors.go`, `fsutil/atomic.go` or similar) | 6–8 |
| R5 | Small | 0 | 1 (decision log) |
| R6 | Large | 3–5 (cache package, schema, rebuild) | 4–6 |

---

## 7. Acceptance Criteria

The remediation is complete when:

1. All 8 bugs (B1–B8) are fixed with regression tests
2. A general-purpose entity field update exists via MCP and CLI
3. `internal/model` has round-trip and enum tests
4. MCP test coverage reaches ≥75% statement coverage
5. All five entity types are exercised through service-layer lifecycle transitions
6. Sentinel errors exist and are used in the service layer
7. MCP enum constraints are applied to all tools that accept entity types or document types
8. Spec deferrals are recorded as accepted decisions
9. The local derived cache is implemented (Track R6 / original Track G)
10. All tests pass, including with `-race`
11. `go vet` is clean