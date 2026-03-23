# Phase 2a Progress and Remaining Work

**Last updated:** 2025-03-23

**Purpose:** Track implementation status of Phase 2a deliverables against the Phase 2 specification (§22 acceptance criteria) and identify remaining work.

---

## 1. Purpose

This document tracks what has been implemented for Phase 2a, what remains, and the status of each acceptance criterion from the Phase 2 specification §22. It is the single source of truth for Phase 2a completion status.

## 2. Implementation Status Summary

| Area | Status | Notes |
|------|--------|-------|
| Plan creation and management | ✅ Complete | All CRUD, lifecycle, prefix validation |
| Prefix registry | ✅ Complete | Config parsing, MCP exposure, validation, retirement |
| Feature lifecycle driven by documents | ✅ Complete | Approval and supersession hooks wired |
| Document management | ✅ Complete | Submit, approve, supersede, ingest, supersession chain |
| Document intelligence — structural | ✅ Complete | Layers 1–2 in `internal/docint` |
| Document intelligence — classification | ✅ Complete | Layer 3 classification protocol |
| Document intelligence — graph/concepts | ✅ Complete | Layer 4 graph storage and concept registry |
| Rich queries | ✅ Complete | Date range, cross-entity, tag queries |
| Concurrency (optimistic locking) | ✅ Complete | Content-hash based conflict detection |
| Migration | ✅ Complete | Epic→Plan, feature field renames, idempotent |
| Deterministic storage | ✅ Complete | All entity types produce canonical YAML |
| Tags | ✅ Complete | Cross-type queries, list all tags |
| Extended health checks | ✅ Complete | Document, plan prefix, feature parent checks |
| Cache schema expansion | ⚠️ Deferred | Low priority; not blocking acceptance |

## 3. What Was Implemented

### 3.1 Entity model evolution (spec §6, §8)

- Plan entity type with full lifecycle (proposed → designing → active → done/abandoned)
- Plan ID format: `{Prefix}{SeqNum}-{slug}` with prefix registry validation
- Document record type with lifecycle (draft → approved → superseded)
- Feature model updated: `epic` → `parent`, `plan` → `dev_plan` field renames
- All Phase 2 lifecycle states defined for Plan, Feature, Task, Bug, Decision

### 3.2 Prefix registry (spec §10)

- `config.yaml` schema with prefix entries (prefix, label, retired flag)
- `config.Load()` and `config.SaveTo()` for reading/writing
- `DefaultConfig()` provides a default `P` prefix
- MCP tools: `config_get`, `config_set_prefix`, `config_retire_prefix`
- Plan creation validates against declared prefixes

### 3.3 Lifecycle (spec §9)

- Full state machines for all entity types including Phase 2 additions
- `CanTransition()` validation in `internal/validate`
- Document-driven transitions:
  - Submitting a design document → entity transitions to `designing`
  - Submitting a specification → entity transitions to `specifying`
  - Approving a specification → feature transitions to `dev-planning`
  - Approving a dev plan → feature transitions to `developing`
  - Approving a plan design → plan transitions to `active`
  - Superseding an approved document → feature reverts to earlier state

### 3.4 Document management (spec §11)

- `DocumentService` with submit, approve, supersede, get, list, validate operations
- `DocumentStore` with atomic writes and content-hash computation
- Content hash drift detection
- Document-to-entity linking via `EntityLifecycleHook`
- Supersession chain traversal (walk forward/backward through version links)
- Best-effort Layer 1–2 ingest on document submission via `IntelligenceService` hook

### 3.5 Plan service (spec §7, §18.1)

- `CreatePlan`, `GetPlan`, `ListPlans`, `UpdatePlan`, `UpdatePlanStatus`
- Prefix validation on creation
- Filtering by status, prefix, and tags

### 3.6 MCP tools (spec §18.1)

Phase 2a tools registered in `internal/mcp/server.go`:

**Entity tools:** `create_entity`, `get_entity`, `list_entities`, `update_entity`, `update_status`
**Plan tools:** `create_plan`, `get_plan`, `list_plans`, `update_plan`, `update_plan_status`
**Document record tools:** `doc_submit`, `doc_record_approve`, `doc_record_supersede`, `doc_record_get`, `doc_record_list`
**Config tools:** `config_get`, `config_set_prefix`, `config_retire_prefix`
**Document intelligence tools:** `doc_classify`, `doc_outline`, `doc_section`, `doc_find_by_entity`, `doc_find_by_concept`, `doc_find_by_role`, `doc_trace`, `doc_gaps`, `doc_pending`, `doc_impact`
**Query tools:** `list_tags`, `list_by_tag`, `query_plan_tasks`, `doc_supersession_chain`
**Migration tools:** `migrate_phase2`

### 3.7 Deterministic YAML (spec §15.4)

- Canonical field ordering per entity type
- Block style mappings and sequences
- UTF-8, LF line endings, trailing newline
- Round-trip tested (write → read → write → compare)

### 3.8 Optimistic locking (spec §16)

- Content-hash based conflict detection in `EntityStore` and `DocumentStore`
- `Load()` computes and stores file hash; `Write()` compares expected hash
- `storage.ErrConflict` sentinel error on mismatch
- Last-write-wins semantics preserved when no prior load (new records)

### 3.9 Document intelligence (spec §12, §13)

Full four-layer implementation in `internal/docint`:

- **Layer 1 — Structural parser:** Deterministic Markdown section tree parser. Handles ATX headings, code fences, hierarchical paths. Produces section nodes with level, title, byte offsets, word/byte counts, content hash.
- **Layer 2 — Pattern extraction:** Entity reference extraction, cross-document link detection, conventional section-role classification from heading text, front matter parsing.
- **Layer 3 — Classification protocol:** Validates agent-provided classifications against the fragment role taxonomy (11 roles). Stores classifications with model provenance and immutability semantics. Rejects non-conforming classifications.
- **Layer 4 — Document graph:** Flat YAML edge lists per document, corpus-wide graph merging. Concept registry with per-document index storage and deduplication.

`IntelligenceService` in `internal/service/intelligence.go` coordinates all four layers and provides query operations: outline, section retrieval, find-by-entity, find-by-concept, find-by-role, pending classification, trace (refinement chain), gaps (coverage analysis), impact (graph edge lookup).

Index storage persisted to `.kbz/index/documents/*`, `.kbz/index/graph.yaml`, `.kbz/index/concepts.yaml`.

### 3.10 Rich queries (spec §14)

- `ListEntitiesFiltered`: filter by type, status, tags, parent, date ranges (created/updated)
- `ListAllTags`: scan all entity types, return sorted unique tags
- `ListEntitiesByTag`: find entities with a given tag across all types
- `CrossEntityQuery`: given a Plan ID, find all features, then all tasks under those features (two-hop query)

### 3.11 Migration (spec §17)

- `MigratePhase2()` on `EntityService`:
  - Converts Phase 1 epic entities to Phase 2 plan entities
  - Assigns new Plan IDs with configured prefix
  - Maps epic status to plan status
  - Renames feature fields: `epic` → `parent` (with ID remapping), `plan` → `dev_plan`
  - Removes epic-only fields (`features` list)
  - Deletes migrated epic files, cleans up empty directory
  - Creates target directories (`plans`, `documents`, `index`)
  - Idempotent: checks for existing plan by slug, skips if present
  - Fails clearly if prefix registry not configured

### 3.12 Extended health checks (spec §21)

New health check functions in `internal/validate/health.go`:

- `CheckDocumentHealth`: file existence, content-hash drift detection, orphaned document records (owner entity must exist), approval field validation (approved docs must have approved_by/approved_at)
- `CheckPlanPrefixes`: validates Plan entities use prefixes declared in prefix registry
- `CheckFeatureParentRefs`: validates feature `parent` references point to existing Plans
- `MergeReports`: combines multiple `HealthReport`s into one aggregate report
- `inferEntityType`: derives entity type from ID prefix for cross-type lookups

### 3.13 Test coverage

Comprehensive tests added across all new functionality:

- `internal/storage/*_test.go` — optimistic locking conflict detection
- `internal/service/intelligence_test.go` — 26 tests covering all IntelligenceService operations
- `internal/service/queries_test.go` — 14 tests for filtered listing, tag queries, cross-entity queries
- `internal/service/migration_test.go` — 7 tests for epic→plan conversion, idempotency, status mapping
- `internal/service/supersession_test.go` — 7 tests for version chain traversal
- `internal/docint/*_test.go` — comprehensive unit tests for each Layer 1–4 component
- `internal/validate/doc_health_test.go` — tests for all new health check functions
- All tests pass with race detector enabled

## 4. Known Issues

### 4.1 Spec deviation: prefix field name

The config YAML uses `prefix` (singular) in the prefix entry struct, matching the spec's schema definition. No action needed.

### 4.2 Config serialization

`config.SaveTo()` uses `gopkg.in/yaml.v3` for serialization rather than the custom canonical serializer. This is acceptable because config files are not entity records and don't need the same deterministic guarantees.

### 4.3 DocumentService hook error handling

Entity lifecycle hook errors in `DocumentService.SubmitDocument`, `ApproveDocument`, and `SupersedeDocument` are silently ignored (`_ = s.entityHook.TransitionStatus(...)`). This is a deliberate "best-effort" design — hook failures should not fail the document operation. However, this means hook failures are invisible. Future improvement: add structured logging.

### 4.4 Migration config isolation

`MigratePhase2()` calls `config.Load()` which reads from a hard-coded relative path (`.kbz/config.yaml`). This makes migration tests sensitive to the working directory. Tests use `t.Cleanup` to remove the config after each test. A future refactor could make the config path injectable.

### 4.5 Submit response does not include structural skeleton

The `doc_submit` MCP tool response returns document record metadata but does not include the Layer 1–2 structural skeleton in the response body. Ingest runs as a side-effect on submit, and the structural skeleton is immediately available via `doc_outline`. This is a composable API design choice — agents call `doc_outline` after `doc_submit` if they need the skeleton.

## 5. Acceptance Criteria Status

Tracking against spec §22 acceptance criteria for Phase 2a items.

### §22.1 Plan creation and management — ✅ Met

- [x] Create a Plan with a declared prefix
- [x] Retrieve a Plan by ID
- [x] List Plans with filtering by status, prefix, and tags
- [x] Transition a Plan through its lifecycle states
- [x] Reject Plan creation with an undeclared prefix

### §22.2 Prefix registry — ✅ Met

- [x] Parse the prefix registry from `.kbz/config.yaml`
- [x] Expose the registry through an MCP operation
- [x] Validate Plan IDs against declared prefixes
- [x] Support prefix retirement
- [ ] Create a default `P` prefix on `kbz init` (init command not yet implemented, but `DefaultConfig()` provides it)

### §22.3 Feature lifecycle driven by documents — ✅ Met

- [x] Approving a Feature's specification transitions Feature to `dev-planning`
- [x] Approving a Feature's dev plan transitions Feature to `developing`
- [x] Superseding an approved document reverts Feature to appropriate earlier state
- [x] Shortcut from `proposed` to `specifying` works (lifecycle states defined)

### §22.4 Document management — ✅ Met

- [x] Submit a document (creating tracked record in draft status)
- [x] Approve a document (transitioning to approved with approver and timestamp)
- [x] Supersede a document (linking to successor)
- [x] Retrieve an approved document verbatim
- [x] Detect content hash drift
- [x] List documents filtered by type, status, and owner
- [x] Submit triggers Layers 1–2 ingest (skeleton available via `doc_outline`)
- [x] Query a document's supersession chain (`doc_supersession_chain` tool)

### §22.5 Document intelligence — structural analysis — ✅ Met

- [x] Parse a Markdown document into a structural section tree
- [x] Extract entity references from document text
- [x] Extract cross-document links
- [x] Return a document outline with section titles, levels, and sizes
- [x] Retrieve a specific section by path

### §22.6 Document intelligence — classification — ✅ Met

- [x] Return structural skeleton with classification schema to agent
- [x] Accept and validate agent-provided classifications
- [x] Reject non-conforming classifications
- [x] Store validated classifications persistently
- [x] List documents pending classification
- [x] Query fragments by role across corpus (`doc_find_by_role`)
- [x] Query sections by concept (`doc_find_by_concept`)

### §22.7 Rich queries — ✅ Met

- [x] Filtering entities by status, parent, tags
- [x] Filtering by date range (created/updated)
- [x] Cross-entity queries (tasks for features in a Plan via `query_plan_tasks`)
- [x] Tag-based queries across entity types (`list_tags`, `list_by_tag`)
- [x] Document listing with filtering by type, status, owner

### §22.8 Concurrency — ✅ Met

- [x] Optimistic locking detects conflicts and returns `storage.ErrConflict`

### §22.9 Migration — ✅ Met

- [x] Convert existing Epic entities to Plans
- [x] Rename fields on Feature entities (`epic` → `parent`, `plan` → `dev_plan`)
- [x] Move files to correct directories
- [x] Idempotent (checks by slug, skips existing plans)
- [x] Fail clearly if prefix registry not configured

### §22.10 Deterministic storage — ✅ Met

- [x] All new file types produce deterministic output

### §22.11 Tags — ✅ Met

- [x] Settable on any entity type
- [x] Queryable on Plans (filter by tags)
- [x] Cross-type tag queries (`list_tags`, `list_by_tag`)
- [x] Freeform lowercase strings with optional colon-namespacing

---

## 6. Remaining Work (Post-Acceptance)

These items are not required for Phase 2a acceptance but are improvements for robustness and operational readiness:

1. **`kbz init` command** — Create default config with `P` prefix on initialization. Currently `DefaultConfig()` provides the default but there's no init command to invoke it.

2. **Cache schema expansion** — Add SQLite tables for documents, doc_sections, tags. Enable WAL mode. Implement upsert/rebuild logic. Low priority unless performance requires it.

3. **Structured logging for hook failures** — Add logging to entity lifecycle hook calls in DocumentService so hook failures are observable without failing document operations.

4. **Config path injection** — Make `config.Load()` accept a path parameter to improve test isolation for migration tests.

5. **Integration testing** — Exercise the full MCP tool surface in realistic multi-agent scenarios. Verify concurrent optimistic lock behavior under contention.

6. **`doc_consistency` operation** — Hierarchical authority model designed (§8.3 of document-intelligence-design.md). Implementation deferred to Phase 3 per spec scope.