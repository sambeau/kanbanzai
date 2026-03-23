# Phase 2a Progress and Remaining Work

- Status: in progress
- Purpose: track what has been implemented, what remains, and known issues for Phase 2a
- Date: 2026-03-22
- Based on:
  - `work/spec/phase-2-specification.md` §5.1, §22.1–22.11
  - `work/plan/phase-2-scope.md`

---

## 1. Purpose

This document tracks the implementation status of Phase 2a against the Phase 2 specification. Phase 2a covers entity model evolution, document management, document intelligence, rich queries, concurrency, and migration.

---

## 2. Implementation Status Summary

| Area | Status | Notes |
|------|--------|-------|
| Entity model evolution | ✅ Done | Plan, Feature updates, DocumentRecord, tags |
| Prefix registry | ✅ Done | Config, validation, retirement, MCP tools |
| Plan lifecycle | ✅ Done | State machine, transition enforcement |
| Feature lifecycle (Phase 2) | ✅ Done | Document-driven states, backward transitions |
| Document management CRUD | ✅ Done | Submit, approve, supersede, get, list |
| Content hash & drift detection | ✅ Done | SHA-256, mtime comparison |
| MCP tools (Plan) | ✅ Done | create, get, list, update, update_status |
| MCP tools (Document records) | ✅ Done | submit, approve, supersede, get, get_content, list, list_pending, validate |
| MCP tools (Config) | ✅ Done | get_project_config, get_prefix_registry, add_prefix, retire_prefix |
| Deterministic YAML | ✅ Done | Field ordering for Plan and DocumentRecord |
| Document-driven Feature transitions | ❌ Not started | Approve/supersede should auto-transition owning Feature |
| Document intelligence (Layers 1–4) | ❌ Not started | Structural skeleton, pattern extraction, classification, graph |
| Optimistic locking | ❌ Not started | Read-hash-write-verify for .kbz/state/ writes |
| Migration command | ❌ Not started | `kbz migrate phase-2` (Epic → Plan) |
| Rich queries | ⚠️ Partial | Plan filtering done; date range, cross-entity, tag listing missing |
| Extended health checks | ❌ Not started | Phase 2 validation extensions |
| Cache schema expansion | ❌ Not started | SQLite tables for documents, tags |
| Document-to-entity linking enforcement | ❌ Not started | Bidirectional reference maintenance |

---

## 3. What Was Implemented

### 3.1 Entity model evolution (spec §6, §8)

**Plan entity** — new top-level entity replacing Epic.

- Model type with all required fields: id, slug, title, status, summary, design, tags, created, created_by, updated, supersedes, superseded_by
- ID format `{X}{n}-{slug}` with validation (`IsPlanID`, `ParsePlanID`)
- Entity type detection from ID pattern without registry lookup
- Storage in `.kbz/state/plans/` with `{id}.yaml` naming per spec §15.1

**Feature updates** — renamed and new fields for Phase 2.

- `parent` field (renamed from `epic`) referencing parent Plan ID
- `design` field for design document record reference
- `spec` field revised to reference tracked document record
- `dev_plan` field (renamed from `plan`) for dev plan document record
- `tags` field for cross-cutting metadata
- Phase 1 fields (`epic`, `plan`) preserved for backward compatibility

**DocumentRecord** — metadata-only record for tracked documents.

- All fields per spec §8.3: id, path, type, title, status, owner, approved_by, approved_at, content_hash, supersedes, superseded_by, created, created_by, updated
- Document types: design, specification, dev-plan, research, report, policy
- Document status: draft, approved, superseded
- ID format: `{owner-id}/{slug}`

**Tags** — freeform lowercase strings with optional colon-namespacing on all entity types.

Files: `internal/model/entities.go`

### 3.2 Prefix registry (spec §10)

- Config stored in `.kbz/config.yaml` with `prefixes` key
- Prefix validation: exactly one non-digit Unicode rune, case-sensitive, unique
- Default prefix `P` with label "Plan" on init
- Prefix retirement: blocks new Plans, preserves existing, cannot retire last active
- `NextPlanNumber` allocation by scanning existing Plan IDs
- Load/save with validation on both read and write

Files: `internal/config/config.go`, `internal/config/config_test.go`

### 3.3 Lifecycle (spec §9)

**Plan lifecycle** — proposed → designing → active → done; terminal: superseded, cancelled.

- All forward and terminal transitions per spec §9.1
- Transition enforcement rejects invalid transitions

**Feature lifecycle (Phase 2)** — proposed → designing → specifying → dev-planning → developing → done; terminal: superseded, cancelled.

- All forward transitions per spec §9.2
- Backward transitions for document supersession (specifying → designing, dev-planning → specifying, developing → dev-planning)
- Shortcut: proposed → specifying (skip design)
- Phase 1 states preserved for backward compatibility

**Document lifecycle** — draft → approved → superseded.

- Transitions validated in document service operations

Files: `internal/validate/lifecycle.go`

### 3.4 Document management (spec §11)

- **Submit** — registers document, creates draft record, computes SHA-256 content hash, validates file exists and type is valid
- **Approve** — validates current status is draft, verifies content hash matches current file, records approver and timestamp
- **Supersede** — validates current status is approved, verifies superseding document exists, updates bidirectional supersession links
- **Get** — retrieves record with optional content drift detection
- **Get content** — returns document file content verbatim with drift warning
- **List** — filtering by type, status, owner
- **List pending** — convenience filter for draft documents
- **Validate** — checks file existence, content hash integrity, type validity, status validity, owner reference format

**Content drift detection** — compares file mtime against record updated timestamp; if file is newer, recomputes SHA-256 and reports drift.

Files: `internal/service/documents.go`, `internal/service/documents_test.go`, `internal/storage/document_store.go`

### 3.5 Plan service (spec §7, §18.1)

- `CreatePlan` — validates prefix against registry, allocates next number, normalizes tags
- `GetPlan` — retrieves by ID with format validation
- `ListPlans` — filtering by status, prefix, tags
- `UpdatePlanStatus` — lifecycle transition enforcement
- `UpdatePlan` — update mutable fields (title, summary, design, tags)

Files: `internal/service/plans.go`, `internal/service/plans_test.go`

### 3.6 MCP tools (spec §18.1)

**Plan tools:** `create_plan`, `get_plan`, `list_plans`, `update_plan_status`, `update_plan`

**Document record tools:** `doc_record_submit`, `doc_record_approve`, `doc_record_supersede`, `doc_record_get`, `doc_record_get_content`, `doc_record_list`, `doc_record_list_pending`, `doc_record_validate`

**Config tools:** `get_project_config`, `get_prefix_registry`, `add_prefix`, `retire_prefix`

All tools registered in `NewServer` alongside Phase 1 tools. Server version updated to `phase-2a-dev`.

Files: `internal/mcp/plan_tools.go`, `internal/mcp/doc_record_tools.go`, `internal/mcp/config_tools.go`, `internal/mcp/server.go`

### 3.7 Deterministic YAML (spec §15.4)

- Plan field ordering defined in `fieldOrderForEntityType`
- DocumentRecord field ordering defined in `fieldOrderForEntityType`
- All new types follow block-style YAML, deterministic field order per P1-DEC-008

Files: `internal/storage/entity_store.go`

### 3.8 Test coverage

| Package | Test count | Coverage areas |
|---------|------------|----------------|
| `internal/config` | 12 tests | Validation, save/load, prefix operations, NextPlanNumber |
| `internal/service` (documents) | 16 tests | Submit, approve, supersede, get, drift detection, validate, list, exists |
| `internal/service` (plans) | 6 tests | ID validation, parsing, filters, tags, required fields |
| `internal/storage` | 1 test | Plan write/load round-trip with correct file naming |

---

## 4. What Remains

### 4.1 Document-driven Feature lifecycle transitions (spec §9.4, §22.3)

**Priority: High** — This is a core Phase 2a requirement.

The spec requires that document approval and supersession automatically drive Feature lifecycle transitions:

- Approving a Feature's specification → transition Feature to `dev-planning`
- Approving a Feature's dev plan → transition Feature to `developing`
- Superseding an approved design → revert Feature to `designing`
- Superseding an approved specification → revert Feature to `specifying`
- Superseding an approved dev plan → revert Feature to `dev-planning`

The `ApproveDocument` and `SupersedeDocument` services currently update document records but do not touch the owning entity's lifecycle state. The `DocumentService` needs access to the `EntityService` (or a shared abstraction) to perform these cross-entity transitions.

### 4.2 Document intelligence (spec §12, §13, §22.5, §22.6)

**Priority: High** — This is explicitly in Phase 2a scope per §5.1.

**Layer 1: Structural skeleton** — Parse Markdown into section tree with level, title, byte offset, word count, byte count. Must be deterministic and rebuilt on every document change.

**Layer 2: Pattern-based extraction** — Extract entity references (FEAT-xxx, TASK-xxx, Plan IDs), cross-document links, section classification by convention (headers containing "Decision", "Requirements", etc.), front matter parsing. Must be deterministic.

**Layer 3: AI-assisted classification** — Accept agent-provided classifications for fragment roles (requirement, decision, rationale, constraint, assumption, risk, question, definition, example, alternative, narrative). Validate against taxonomy schema. Store persistently with model/version provenance. Immutable once recorded.

**Layer 4: Document graph** — Maintain persistent graph with Document, Section, Fragment, EntityRef, Concept nodes and CONTAINS, REFERENCES, LINKS_TO, DEPENDS_ON, SUPERSEDES, INTRODUCES, USES, REFINES edges. Initial storage as flat YAML edge lists.

**Concept registry** — Corpus-wide concept deduplication, stored in `.kbz/index/concepts.yaml`.

**MCP tools needed:** `doc_classify`, `doc_outline`, `doc_section`, `doc_find_by_entity`, `doc_find_by_concept`, `doc_find_by_role`, `doc_trace`, `doc_impact`, `doc_gaps`, `doc_pending`

**Storage needed:** `.kbz/index/documents/` (one YAML per indexed document), `.kbz/index/concepts.yaml`, `.kbz/index/graph.yaml`

### 4.3 Optimistic locking (spec §16, §22.8)

**Priority: High** — Required for concurrent safety.

All writes to `.kbz/state/` files must:

1. Read the file and compute its content hash
2. Perform the intended modification
3. Before writing, verify the file's content hash has not changed
4. If changed, fail with a specific conflict error

This needs a dedicated write helper applied consistently in entity store and document store write paths. A conflict error type should be defined so callers can retry.

### 4.4 Migration command (spec §17, §22.9)

**Priority: High** — Required for Phase 1 → Phase 2 transition.

`kbz migrate phase-2` must:

- Rename Epic entities to Plans
- Rename `epic` field to `parent` on Feature entities
- Rename `plan` field to `dev_plan` on Feature entities
- Move files from `.kbz/state/epics/` to `.kbz/state/plans/`
- Re-assign `EPIC-*` IDs to `{X}{n}-{slug}` format
- Create `.kbz/state/documents/` and `.kbz/index/` directories
- Be idempotent, explicit (not automatic), and fail if prefix registry is not configured

### 4.5 Rich queries (spec §14, §22.7)

**Priority: Medium** — Partially implemented.

**Done:** Plan filtering by status, prefix, tags. Document listing by type, status, owner.

**Remaining:**

- Date range filtering (created, updated) on all entity types
- Cross-entity queries: all tasks for features in a given Plan, all entities tagged with a given tag
- List all tags in use across the project
- Filter any entity listing by one or more tags (cross-type)
- Document supersession chain query

### 4.6 Extended health checks (spec §21)

**Priority: Medium**

Phase 2 health checks must detect:

- Plan entities with undeclared prefixes
- Features with document status inconsistent with Feature lifecycle status
- Document records whose content_hash does not match the file on disk
- Document records whose path points to a nonexistent file
- Orphaned document records (owner entity does not exist)
- Documents in approved status with no approved_by or approved_at
- Index files stale relative to source documents

### 4.7 Document-to-entity linking enforcement (spec §11.3)

**Priority: Medium**

- Specification must be linked to exactly one Feature
- Dev plan must be linked to exactly one Feature
- Design may be linked to a Plan or Feature
- Bidirectional reference maintenance: when a document is submitted with an owner, the owning entity's design/spec/dev_plan field should be updated, and vice versa

### 4.8 Cache schema expansion

**Priority: Low** — Can be deferred without blocking other work.

- Add SQLite tables for documents, doc_sections, tags
- Ensure WAL mode for concurrent read access
- Rebuild support and upsert logic for document records

---

## 5. Known Issues

### 5.1 Spec deviation: prefix field name

The spec §10.2 says each prefix has a required `name` field, but the implementation uses `label`. This should be reconciled — either update the code to use `name` or update the spec to use `label`.

### 5.2 Config serialization

`Config.Save()` uses Go's default `yaml.Marshal` rather than the project's canonical YAML serializer. While `.kbz/config.yaml` is configuration (not entity state), the inconsistency with the deterministic serialization rules should be considered.

### 5.3 Plan creation test coverage

The `TestCreatePlan_WithConfig` test is skipped unless `.kbz/config.yaml` exists at the global config path. This means the full Plan create/read/list cycle through the service layer is not exercised in CI. The test should be refactored to use a test-local config path.

### 5.4 DocumentService isolation from EntityService

`DocumentService` is a separate service that cannot access `EntityService` operations. This makes it impossible to implement document-driven Feature lifecycle transitions (§4.1 above) without either merging the services, injecting an `EntityService` reference, or introducing a shared interface. This is a design decision that should be made before implementing §4.1.

---

## 6. Acceptance Criteria Status

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

### §22.3 Feature lifecycle driven by documents — ❌ Not met

- [ ] Approving a Feature's specification transitions Feature to `dev-planning`
- [ ] Approving a Feature's dev plan transitions Feature to `developing`
- [ ] Superseding an approved document reverts Feature to appropriate earlier state
- [x] Shortcut from `proposed` to `specifying` works (lifecycle states defined)

### §22.4 Document management — ⚠️ Partially met

- [x] Submit a document (creating tracked record in draft status)
- [x] Approve a document (transitioning to approved with approver and timestamp)
- [x] Supersede a document (linking to successor)
- [x] Retrieve an approved document verbatim
- [x] Detect content hash drift
- [x] List documents filtered by type, status, and owner
- [ ] Submit includes Layers 1–2 ingest and returns structural skeleton
- [ ] Query a document's supersession chain

### §22.5 Document intelligence — structural analysis — ❌ Not met

- [ ] Parse a Markdown document into a structural section tree
- [ ] Extract entity references from document text
- [ ] Extract cross-document links
- [ ] Return a document outline with section titles, levels, and sizes
- [ ] Retrieve a specific section by path

### §22.6 Document intelligence — classification — ❌ Not met

- [ ] Return structural skeleton with classification schema to agent
- [ ] Accept and validate agent-provided classifications
- [ ] Reject non-conforming classifications
- [ ] Store validated classifications persistently
- [x] List documents pending classification (ListPendingDocuments)
- [ ] Query fragments by role across corpus
- [ ] Query sections by concept

### §22.7 Rich queries — ⚠️ Partially met

- [x] Filtering entities by status, parent (Plans only), tags (Plans only)
- [ ] Filtering by date range
- [ ] Cross-entity queries (tasks for features in a Plan)
- [ ] Tag-based queries across entity types
- [x] Document listing with filtering by type, status, owner

### §22.8 Concurrency — ❌ Not met

- [ ] Optimistic locking detects conflicts and returns error

### §22.9 Migration — ❌ Not met

- [ ] Convert existing Epic entities to Plans
- [ ] Rename fields on Feature entities
- [ ] Move files to correct directories
- [ ] Idempotent
- [ ] Fail clearly if prefix registry not configured

### §22.10 Deterministic storage — ✅ Met

- [x] All new file types produce deterministic output

### §22.11 Tags — ⚠️ Partially met

- [x] Settable on any entity type
- [x] Queryable on Plans (filter by tags)
- [ ] Cross-type tag queries (list entities by tag, list all tags in use)
- [x] Freeform lowercase strings with optional colon-namespacing

---

## 7. Recommended Priority Order

1. **Document-driven Feature lifecycle transitions** — core to the Phase 2 design philosophy; requires a design decision about service coupling
2. **Document intelligence Layers 1–2** — deterministic, foundational for all intelligence features
3. **Migration command** — required for any project transitioning from Phase 1
4. **Optimistic locking** — required for correctness under concurrent use
5. **Rich queries** — date range filtering, cross-entity queries, tag listing
6. **Document intelligence Layer 3** — classification protocol
7. **Document intelligence Layer 4** — graph storage and queries
8. **Extended health checks** — validation extensions
9. **Cache schema expansion** — performance optimization