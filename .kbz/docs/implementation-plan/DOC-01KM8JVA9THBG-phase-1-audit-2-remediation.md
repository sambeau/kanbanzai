---
id: DOC-01KM8JVA9THBG
type: implementation-plan
title: Phase 1 Audit 2 Remediation
status: submitted
feature: FEAT-01KM8JTF0VP0K
created_by: human
created: 2026-03-21T16:14:58Z
updated: 2026-03-21T16:14:58Z
---
# Phase 1 Audit 2 Remediation Plan

- Status: complete
- Date: 2025-07-21
- Completed: 2026-03-20
- Purpose: define the work needed to close remaining gaps found during the second Phase 1 implementation audit
- Related:
  - `work/spec/phase-1-specification.md`
  - `work/plan/phase-1-implementation-plan.md`
  - `work/plan/phase-1-audit-remediation.md`
  - `work/plan/phase-1-decision-log.md`

---

## 1. Purpose

This document is a second addendum to `phase-1-implementation-plan.md`.

After completing the first audit remediation (Tracks R1–R6), a second full-scope audit of the implementation against the Phase 1 specification and implementation plan was performed. The audit assessed all seven implementation layers, the required operations list (§8), build and test health, and bootstrap readiness.

The first audit covered Tracks A–E and produced remediation tracks R1–R6, all of which are now resolved. This second audit covers the complete Phase 1 scope and identifies the remaining work needed before Phase 1 can be considered complete.

---

## 2. Audit Scope and Method

The audit assessed each of the seven implementation layers defined in `phase-1-implementation-plan.md` §6:

| Layer | Area | Verdict |
|---|---|---|
| Layer 1 | Core model and storage | ✅ Complete (one minor defect) |
| Layer 2 | Validation engine | ✅ Complete (two minor gaps) |
| Layer 3 | Document support | ⚠️ ~80% — extraction missing |
| Layer 4 | MCP service layer | ⚠️ ~90% — extraction missing |
| Layer 5 | CLI layer | ⚠️ ~60% — document and utility commands missing |
| Layer 6 | Local derived cache | ✅ Complete |
| Layer 7 | Bootstrap usage | ❌ Not started |

Build and test health:

- `go build ./...` — clean
- `go test ./...` — all pass
- `go vet ./...` — clean
- 14 test files, 5 entity fixtures, 11 of 12 packages tested

---

## 3. Audit Findings Summary

### 3.1 Bugs

| ID | Severity | Summary | Location |
|---|---|---|---|
| B9 | Medium | Decision `status` missing from canonical field order in `fieldOrderForEntityType()` | `internal/storage/entity_store.go` |

The `fieldOrderForEntityType("decision")` function does not include `"status"` in its ordered list. The field serialises via the alphabetical "extras" bucket and round-trip tests pass by coincidence (alphabetical order happens to match the fixture). This is inconsistent with the other four entity types, which all include `status` in their explicit canonical order.

### 3.2 Spec compliance gaps

| ID | Spec § | Summary |
|---|---|---|
| S7 | §8.6, §15.5 | Document-to-entity extraction not implemented — no extraction logic, no MCP tool, no service method |
| S8 | §6.5 | CLI missing all document operations (scaffold, submit, approve, retrieve, validate, list) |
| S9 | §6.5 | CLI missing health check command |
| S10 | §6.5 | CLI missing validate candidate command |
| S11 | §15.7 | Document referential integrity does not verify referenced entities exist |
| S12 | §15.7 | No slug format validation |
| S13 | §4.8, §15.7 | `ValidateRecord` does not check ID format via the allocator |

### 3.3 Layer 7 gap

| ID | Summary |
|---|---|
| L7-1 | Bootstrap usage not started — `.kbz/` is an empty placeholder with no canonical entity records |
| L7-2 | No proof the kernel can manage limited self-work without corruption |
| L7-3 | Product/instance boundary not exercised in practice |

---

## 4. Remediation Tracks

### 4.1 Track R7 — Bug fix (Decision field order)

Goal: Fix the canonical serialisation defect for Decision entities.

Tasks:

1. **Fix B9: Add `status` to Decision field order.**
   - File: `internal/storage/entity_store.go`
   - Add `"status"` after `"date"` in the `fieldOrderForEntityType("decision")` list
   - Update `testdata/entities/decision.yaml` fixture to reflect the corrected canonical field order (move `status` to its proper position after `date`, before `affects`)
   - Verify round-trip test still passes with the corrected fixture

Outputs:
- Decision entities serialise with `status` in canonical position
- Consistent with all other entity types

### 4.2 Track R8 — Document-to-entity extraction

Goal: Satisfy spec §8.6 and §15.5 — implement extraction of entities and decisions from approved documents.

This is the largest remaining functional gap. The document-centric workflow requires that after a human approves a document, the system can support extracting structured entities (features, tasks, decisions, etc.) and cross-references from it. Per P1-DEC-019 in the first audit, automated extraction was deferred. This track revisits that decision.

There are two approaches:

- **Option A (tool-assisted extraction):** Provide an `extract_from_document` MCP tool that retrieves an approved document and returns it to the calling agent for manual entity extraction using existing `create_*` and `record_decision` tools. This is minimal — it provides the tooling surface without implementing parsing logic.
- **Option B (structured extraction):** Implement parsing logic that identifies entity-like structures in approved document bodies and returns candidate entities for agent review before creation.

Recommendation: Option A. Phase 1 should provide the extraction _surface_ (an MCP tool that retrieves approved document content in a format suitable for extraction) and rely on agents to perform the actual extraction using existing entity creation tools. This matches the design principle that agents mediate between documents and structured state.

Tasks:

1. **Resolve P1-DEC-019: change from "deferred" to "Option A — agent-mediated extraction with tool support".**
   - Update the decision log entry

2. **Implement `ExtractFromDocument` service method.**
   - File: `internal/document/service.go` or `internal/service/entities.go`
   - Accept a document ID
   - Verify the document is in `approved` status (extraction only operates on approved documents)
   - Return the document content (meta + body) in a structured format suitable for an agent to parse

3. **Implement `extract_from_document` MCP tool.**
   - File: `internal/mcp/document_tools.go`
   - Accept `doc_id` parameter
   - Delegate to the service method
   - Return the document body and metadata as structured output
   - Include guidance text in the tool description explaining the agent should use the content to create entities via `create_*` / `record_decision` tools

4. **Add tests.**
   - Extraction from an approved document succeeds and returns full content
   - Extraction from a non-approved document (submitted, normalised) returns an error
   - Extraction from a non-existent document returns an error

Outputs:
- `extract_from_document` MCP tool available
- Agents can retrieve approved document content for manual entity extraction
- §8.6 requirement satisfied at the tool surface level

### 4.3 Track R9 — CLI parity for document operations

Goal: Satisfy §6.5 — the CLI should support document lifecycle operations for manual operation, bootstrap use, CI, debugging, and repair.

Tasks:

1. **Add `kbz doc scaffold` command.**
   - Accept `--type` (required)
   - Print scaffolded document content to stdout

2. **Add `kbz doc submit` command.**
   - Accept `--type`, `--title`, `--created-by` (required), `--feature` (optional)
   - Accept body from `--body` flag or stdin
   - Print the created document ID and path

3. **Add `kbz doc approve` command.**
   - Accept `--id`, `--approved-by` (required)
   - Print confirmation

4. **Add `kbz doc retrieve` command.**
   - Accept `--id` (required)
   - Print the document body to stdout (verbatim)

5. **Add `kbz doc validate` command.**
   - Accept `--id` (required)
   - Print validation results (errors and warnings)

6. **Add `kbz doc list` command.**
   - Accept `--type` (optional, lists all if omitted)
   - Print document listing

7. **Add tests.**
   - File: `cmd/kanbanzai/main_test.go`
   - Test the scaffold, submit, approve, retrieve, validate, and list subcommands

Outputs:
- Full document lifecycle available from the CLI
- Parity with MCP document tools

### 4.4 Track R10 — CLI utility commands

Goal: Add the remaining CLI utility commands that exist in MCP but not the CLI.

Tasks:

1. **Add `kbz health` command.**
   - Run health check against canonical state
   - Print errors and warnings

2. **Add `kbz validate` command.**
   - Accept `--type` and entity field flags
   - Perform candidate validation (dry run) without persistence
   - Print validation results

3. **Add tests.**
   - File: `cmd/kanbanzai/main_test.go`

Outputs:
- Health check and candidate validation available from CLI
- Useful for debugging and CI

### 4.5 Track R11 — Validation hardening

Goal: Close the minor validation gaps identified in the audit.

Tasks:

1. **S12: Add slug format validation.**
   - File: `internal/validate/entity.go`
   - Add a `ValidateSlug(slug string) error` function
   - Enforce: non-empty, lowercase, kebab-case, no path separators, no special characters beyond hyphens
   - Call from `ValidateRecord` for all entity types
   - Add tests for valid and invalid slugs

2. **S13: Add ID format checking in `ValidateRecord`.**
   - File: `internal/validate/entity.go`
   - Import `internal/id` and call `Allocator.Validate()` to verify the ID matches the expected format for the entity type
   - Add to `CheckHealth` as well so stored entities with malformed IDs are flagged
   - Add tests: a candidate with `id: "EPIC-001"` (wrong prefix for entity kind) should fail validation

3. **S11: Add document referential integrity check.**
   - File: `internal/document/validate.go` or `internal/document/service.go`
   - When a document references a feature (e.g., `feature: FEAT-003`), verify that feature exists in entity storage
   - This requires the document validator to have access to the entity store or cache
   - Add tests: document referencing a non-existent feature should produce a validation warning

Outputs:
- Slug format enforced at validation time
- ID format verified during candidate validation and health checks
- Document referential integrity checks reference actual entities

### 4.6 Track R12 — Bootstrap usage (Layer 7)

Goal: Satisfy §6.7 — begin using the Phase 1 kernel to manage limited current-project workflow state.

This is the final layer and should only be attempted after Tracks R7–R11 are complete.

Prerequisites:
- The kernel must be stable and trustworthy for basic use
- All critical bugs fixed
- CLI document commands available (needed for manual bootstrap operations)

Tasks:

1. **Create initial instance state.**
   - Use `kbz create epic` to create a Phase 1 epic (E-001)
   - Use `kbz create feature` to create representative features for the major implementation tracks
   - Use `kbz record decision` to record at least one decision via the kernel

2. **Verify the product/instance boundary.**
   - Confirm `.kbz/state/` contains only instance data
   - Confirm no instance data leaks into product directories (`internal/`, `cmd/`)
   - Confirm `.kbz/` is properly `.gitignore`d (or selectively committed per project policy)

3. **Run health check on bootstrapped state.**
   - `kbz health` should return clean results
   - `kbz cache rebuild` should succeed

4. **Verify round-trip integrity.**
   - Create entities, read them back, verify canonical YAML is deterministic
   - Update a status, verify the transition is enforced
   - Attempt an invalid transition, verify it is rejected

5. **Document the bootstrap results.**
   - Record whether the kernel is trustworthy for basic self-management
   - Note any issues encountered during bootstrap

Outputs:
- `.kbz/state/` populated with initial canonical records
- Product/instance boundary verified in practice
- Proof the kernel can manage limited self-work

---

## 5. Execution Record

R7–R11 were implemented in three commits on `main`:

- `4549887 fix(BUG-009): harden validation and decision ordering` — R7 + R11
- `4aed622 feat(P1-DEC-019): add document extraction support` — R8
- `7352a1c feat(PHASE-1): add CLI document and utility commands` — R9 + R10

R12 bootstrap verification was performed on 2026-03-20.

---

## 6. Estimated Scope

| Track | Estimated size | New files | Modified files |
|---|---|---|---|
| R7 | Small | 0 | 2 (`entity_store.go`, `testdata/entities/decision.yaml`) |
| R8 | Medium | 0–1 | 3–4 (`document/service.go`, `mcp/document_tools.go`, test files) |
| R9 | Medium–Large | 0 | 2–3 (`cmd/kanbanzai/main.go`, test file) |
| R10 | Small | 0 | 2 (`cmd/kanbanzai/main.go`, test file) |
| R11 | Medium | 0 | 4–5 (`validate/entity.go`, `document/validate.go`, `id/allocator.go`, test files) |
| R12 | Small | 0 | 1–2 (`.kbz/` state files, documentation) |

---

## 7. Relationship to First Audit Remediation

The first audit (`phase-1-audit-remediation.md`) defined Tracks R1–R6:

| Track | Status |
|---|---|
| R1 — Bug fixes (B1–B8) | ✅ Complete |
| R2 — Entity field update | ✅ Complete |
| R3 — Test coverage improvements | ✅ Complete |
| R4 — Code quality improvements | ✅ Complete |
| R5 — Spec compliance deferrals | ✅ Complete |
| R6 — Local derived cache | ✅ Complete |

This second audit found no regressions from the first remediation. All first-audit items remain resolved.

The new tracks (R7–R12) continue the numbering sequence to maintain traceability across both documents.

All tracks (R7–R12) are now complete.

---

## 8. Acceptance Criteria

The second audit remediation is complete when:

1. ✅ Decision `status` field appears in canonical position in serialised output (R7)
2. ✅ An `extract_from_document` MCP tool exists and operates on approved documents (R8)
3. ✅ All six document lifecycle operations are available from the CLI (R9)
4. ✅ `kbz health` and `kbz validate` CLI commands exist (R10)
5. ✅ Slug format validation rejects malformed slugs (R11)
6. ✅ `ValidateRecord` and `CheckHealth` verify ID format (R11)
7. ✅ Document validation checks that referenced entities exist (R11)
8. ✅ The kernel has been used to create initial project state in `.kbz/state/` (R12)
9. ✅ A health check against the bootstrapped state returns clean results (R12)
10. ✅ All tests pass, including with `-race`
11. ✅ `go vet` is clean

All acceptance criteria met as of 2026-03-20.

---

## 9. Phase 1 Completion Criteria

When all acceptance criteria in this document are met, Phase 1 is complete per the implementation plan. Specifically:

- ✅ All seven implementation layers are functional
- ✅ All required operations from §8 of the implementation plan are satisfied
- ✅ The kernel has been used on itself (Layer 7)
- ✅ No known bugs remain
- ✅ No silent spec omissions remain (all deferrals are recorded as decisions)

---

## 10. Track R12 — Bootstrap Verification Results

Verification performed: 2026-03-20

### 10.1 Instance state

Five entities created via the kernel CLI and stored in `.kbz/state/`:

| Entity | ID | Status |
|---|---|---|
| Epic: Phase 1 Completion | E-001 | done |
| Epic: Invalid Transition Check | E-002 | approved |
| Feature: Audit 2 Remediation | FEAT-001 | done |
| Feature: Bootstrap Self-Management | FEAT-002 | done |
| Decision: Phase 1 Bootstrap Scope | DEC-001 | proposed |

### 10.2 Round-trip determinism

All 5 entities loaded via CLI without error. Self-transition attempts (`--status <current>`) were correctly rejected with `self-transition "<status>" is not allowed` for all entity types. File content on disk matches CLI read-back after every mutation.

### 10.3 Lifecycle transition enforcement

| Test | From → To | Expected | Result |
|---|---|---|---|
| Invalid forward skip | `proposed` → `done` | reject | ✅ rejected: `invalid epic transition "proposed" -> "done"` |
| Valid forward | `proposed` → `approved` | accept | ✅ accepted |
| Invalid backward | `approved` → `proposed` | reject | ✅ rejected: `invalid epic transition "approved" -> "proposed"` |
| Full feature lifecycle | `draft` → ... → `done` | accept all 5 steps | ✅ all accepted |

### 10.4 Product/instance boundary

- `.kbz/state/` contains only instance data (entity YAML files)
- No instance data in `internal/`, `cmd/`, or `work/`
- `.kbz/` is `.gitignore`d except for `.gitkeep`
- `.kbz/cache/kbz.db` is the local derived cache, not committed

### 10.5 Health check

```
health check
entities: 5
errors: 0
warnings: 0
```

### 10.6 Cache rebuild

`kbz cache rebuild` succeeded: 5 entities cached to `.kbz/cache/kbz.db`.

### 10.7 Conclusion

The Phase 1 kernel can create, read, update, and validate its own project entities without corruption. Lifecycle state machines enforce valid transitions and reject invalid ones. Canonical YAML serialisation is deterministic. The product/instance boundary is clean. The kernel is suitable for limited bootstrap self-management.