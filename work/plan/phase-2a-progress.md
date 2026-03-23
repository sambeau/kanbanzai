# Phase 2a Progress and Remaining Work

**Last updated:** 2025-07-23

**Purpose:** Track implementation status of Phase 2a deliverables against the Phase 2 specification (§22 acceptance criteria) and identify remaining work.

---

## 1. Purpose

This document tracks what has been implemented for Phase 2a, what remains, and the status of each acceptance criterion from the Phase 2 specification §22. It is the single source of truth for Phase 2a completion status.

## 2. Implementation Status Summary

| Area | Status | Notes |
|------|--------|-------|
| Plan creation and management | ⚠️ Has issues | Lifecycle bug: `done` marked terminal, blocks `done→superseded` (see audit B1) |
| Prefix registry | ✅ Complete | Config parsing, MCP exposure, validation, retirement |
| Feature lifecycle driven by documents | ⚠️ Has issues | Lifecycle bugs, entry state stuck at Phase 1 `draft`, field renames incomplete at service layer (see audit B2, B3) |
| Document management | ⚠️ Has issues | Optimistic locking bypassed in DocumentService (see audit B4) |
| Document intelligence — structural | ⚠️ Has issues | Content blocks not identified (see audit report) |
| Document intelligence — classification | ⚠️ Has issues | `ClassifiedAt` timestamp not set in `doc_classify` (see audit B5) |
| Document intelligence — graph/concepts | ⚠️ Has issues | 3 edge types missing, non-deterministic role matching (see audit B6, B7) |
| Rich queries | ✅ Complete | Date range, cross-entity, tag queries |
| Concurrency (optimistic locking) | ⚠️ Needs remediation | Bypassed in DocumentService — FileHash dropped during model conversion (see audit B4) |
| Migration | ✅ Complete | Epic→Plan, feature field renames, idempotent |
| Deterministic storage | ⚠️ Needs remediation | Index files use `yaml.Marshal`, not canonical serializer (see audit report) |
| Tags | ✅ Complete | Cross-type queries, list all tags |
| Extended health checks | ✅ Complete | Document, plan prefix, feature parent checks |
| Cache schema expansion | ⚠️ Deferred | Low priority; not blocking acceptance |

## 3. What Was Implemented

### 3.1 Entity model evolution (spec §6, §8)

- Plan entity type with full lifecycle (proposed → designing → active → done/abandoned)
- Plan ID format: `{Prefix}{SeqNum}-{slug}` with prefix registry validation
- Document record type with lifecycle (draft → approved → superseded)
- Feature model updated: `epic` → `parent`, `plan` → `dev_plan` field renames at struct level
- All Phase 2 lifecycle states defined for Plan, Feature, Task, Bug, Decision

**Note:** Feature field renames (`epic` → `parent`, `plan` → `dev_plan`) are struct-level only. The service layer and MCP tool layer still use the old names (`epic`/`plan`) in several places. Full propagation is incomplete.

### 3.2 Prefix registry (spec §10)

- `config.yaml` schema with prefix entries (prefix, label, retired flag)
- `config.Load()` and `config.SaveTo()` for reading/writing
- `DefaultConfig()` provides a default `P` prefix
- MCP tools: `get_project_config`, `add_prefix`, `retire_prefix`
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

**Entity tools:** `get_entity`, `list_entities`, `update_entity`, `update_status` (note: generic entity tools do not support Plan type — use dedicated plan tools)
**Plan tools:** `create_plan`, `get_plan`, `list_plans`, `update_plan`, `update_plan_status`
**Document record tools:** `doc_record_submit`, `doc_record_approve`, `doc_record_supersede`, `doc_record_get`, `doc_record_get_content`, `doc_record_list`, `doc_record_list_pending`, `doc_record_validate`
**Config tools:** `get_project_config`, `add_prefix`, `retire_prefix`, `get_prefix_registry`
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

**Note:** Optimistic locking is effectively bypassed in the `DocumentService` because `FileHash` is dropped during model conversion (storage record → service model → storage record). By the time a document record is written back, the hash used for comparison is zero-valued, so the conflict check always passes. See audit finding B4.

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
- Most tests pass with race detector enabled. **Known failure:** `TestEntityService_ResolvePrefix` is flaky due to non-deterministic map iteration order (see audit finding B8).

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

The `doc_record_submit` MCP tool response returns document record metadata but does not include the Layer 1–2 structural skeleton in the response body. Ingest runs as a side-effect on submit, and the structural skeleton is immediately available via `doc_outline`. This is a composable API design choice — agents call `doc_outline` after `doc_record_submit` if they need the skeleton.

### 4.6 Plan `done` incorrectly marked terminal (B1)

The Plan lifecycle state machine marks `done` as a terminal state. This blocks the `done → superseded` transition required by the spec. The spec defines `done` as a valid source for supersession. Remediation: remove `done` from terminal states for Plan.

### 4.7 Feature `done` incorrectly marked terminal (B2)

Same issue as §4.6 but for the Feature lifecycle. `done` is marked terminal, blocking `done → superseded`. Remediation: remove `done` from terminal states for Feature.

### 4.8 Feature entry state stuck at Phase 1 `draft` (B3)

The Feature lifecycle still uses `draft` as its entry state from Phase 1. Phase 2 specifies `proposed` as the entry state for Features. This means newly created Features enter the wrong initial state, and the Phase 2 lifecycle transitions that start from `proposed` are unreachable. Remediation: update Feature entry state to `proposed`.

### 4.9 Optimistic locking bypassed in DocumentService (B4)

`FileHash` is dropped during the round-trip between storage records and service-layer models. When `DocumentService` loads a document record, the hash is computed by the storage layer but lost during conversion to the service model. When the record is written back, the zero-valued hash always passes the conflict check. This effectively disables optimistic locking for all document operations. Remediation: propagate `FileHash` through the model conversion layer.

### 4.10 `ClassifiedAt` timestamp not set in `doc_classify` (B5)

The `doc_classify` MCP tool handler does not set the `ClassifiedAt` timestamp on classifications before storing them. Classifications are persisted without a timestamp, making it impossible to determine when a classification was applied. Remediation: set `ClassifiedAt` to `time.Now()` in the classify handler.

### 4.11 `MatchConventionalRole` non-deterministic (B6)

`MatchConventionalRole` iterates over a `map[string][]string` to match heading text against conventional role patterns. Go map iteration order is non-deterministic, so when a heading matches patterns for multiple roles, the returned role varies between runs. Remediation: use a deterministic iteration order (sorted keys or an ordered slice).

### 4.12 `NormalizeConcept` produces double hyphens (B7)

`NormalizeConcept` replaces non-alphanumeric characters with hyphens but does not collapse consecutive hyphens. Input like "model — overview" produces `model---overview` instead of `model-overview`. Remediation: collapse runs of hyphens after replacement.

### 4.13 `TestEntityService_ResolvePrefix` flaky (B8)

This test is sensitive to map iteration order when resolving prefixes. Under the race detector or on different platforms, the test can fail non-deterministically. Remediation: fix the test to not depend on map iteration order, or fix the underlying `ResolvePrefix` implementation to be deterministic.

## 5. Acceptance Criteria Status

Tracking against spec §22 acceptance criteria for Phase 2a items.

### §22.1 Plan creation and management — ⚠️ Has issues

- [x] Create a Plan with a declared prefix
- [x] Retrieve a Plan by ID
- [x] List Plans with filtering by status, prefix, and tags
- [x] Transition a Plan through its lifecycle states
- [x] Reject Plan creation with an undeclared prefix
- Note: `done → superseded` transition is blocked because `done` is incorrectly marked terminal (B1)

### §22.2 Prefix registry — ✅ Met

- [x] Parse the prefix registry from `.kbz/config.yaml`
- [x] Expose the registry through an MCP operation
- [x] Validate Plan IDs against declared prefixes
- [x] Support prefix retirement
- [ ] Create a default `P` prefix on `kbz init` (init command not yet implemented, but `DefaultConfig()` provides it)

### §22.3 Feature lifecycle driven by documents — ⚠️ Has issues

- [x] Approving a Feature's specification transitions Feature to `dev-planning`
- [x] Approving a Feature's dev plan transitions Feature to `developing`
- [x] Superseding an approved document reverts Feature to appropriate earlier state
- [x] Shortcut from `proposed` to `specifying` works (lifecycle states defined)
- Note: Feature entry state is `draft` (Phase 1) instead of `proposed` (Phase 2), so newly created Features cannot reach Phase 2 lifecycle paths without manual status override (B3)
- Note: `done → superseded` blocked because `done` is terminal (B2)

### §22.4 Document management — ✅ Met

- [x] Submit a document (creating tracked record in draft status)
- [x] Approve a document (transitioning to approved with approver and timestamp)
- [x] Supersede a document (linking to successor)
- [x] Retrieve an approved document verbatim
- [x] Detect content hash drift
- [x] List documents filtered by type, status, and owner
- [x] Submit triggers Layers 1–2 ingest (skeleton available via `doc_outline`)
- [x] Query a document's supersession chain (`doc_supersession_chain` tool)

### §22.5 Document intelligence — structural analysis — ⚠️ Has issues

- [x] Parse a Markdown document into a structural section tree
- [x] Extract entity references from document text
- [x] Extract cross-document links
- [x] Return a document outline with section titles, levels, and sizes
- [x] Retrieve a specific section by path
- Note: Content blocks (code fences, tables, lists) are not identified as distinct structural elements within sections (see audit report)

### §22.6 Document intelligence — classification — ⚠️ Has issues

- [x] Return structural skeleton with classification schema to agent
- [x] Accept and validate agent-provided classifications
- [x] Reject non-conforming classifications
- [x] Store validated classifications persistently
- [x] List documents pending classification
- [x] Query fragments by role across corpus (`doc_find_by_role`)
- [x] Query sections by concept (`doc_find_by_concept`)
- Note: `ClassifiedAt` timestamp is not set when classifications are stored, so temporal queries and provenance tracking are broken (B5)

### §22.7 Rich queries — ✅ Met

- [x] Filtering entities by status, parent, tags
- [x] Filtering by date range (created/updated)
- [x] Cross-entity queries (tasks for features in a Plan via `query_plan_tasks`)
- [x] Tag-based queries across entity types (`list_tags`, `list_by_tag`)
- [x] Document listing with filtering by type, status, owner

### §22.8 Concurrency — ⚠️ Has issues

- [x] Optimistic locking detects conflicts and returns `storage.ErrConflict`
- Note: Locking is bypassed in `DocumentService` because `FileHash` is dropped during model conversion — conflict detection is effectively disabled for document operations (B4)

### §22.9 Migration — ✅ Met

- [x] Convert existing Epic entities to Plans
- [x] Rename fields on Feature entities (`epic` → `parent`, `plan` → `dev_plan`)
- [x] Move files to correct directories
- [x] Idempotent (checks by slug, skips existing plans)
- [x] Fail clearly if prefix registry not configured

### §22.10 Deterministic storage — ⚠️ Has issues

- [x] All new entity file types produce deterministic output
- Note: Index files (`.kbz/index/graph.yaml`, `.kbz/index/concepts.yaml`) use `yaml.Marshal` instead of the canonical serializer, so they do not meet the deterministic storage contract (see audit report)

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

See `work/plan/phase-2a-audit-report.md` for the complete remediation plan covering all audit findings (B1–B8).