# P37-F2: Document Type System and Filename Enforcement — Implementation Plan

| Field  | Value                                                                        |
|--------|------------------------------------------------------------------------------|
| Date   | 2026-04-27T14:11:10Z                                                         |
| Status | approved |
| Author | orchestrator                                                                 |
| Spec   | work/design/p37-f2-spec-doc-type-and-filename-enforcement.md                 |

---

## Overview

This plan covers the full implementation of F2 of Plan P37: a revised document type system
and filename/folder enforcement for `doc register`. It maps directly to the acceptance
criteria in the specification at
`work/design/p37-f2-spec-doc-type-and-filename-enforcement.md`.

The changes touch three layers of the stack:

1. **Model** (`internal/model/entities.go`) — type constants, public API functions, and
   normalisation helpers.
2. **Service** (`internal/service/documents.go`) — `SubmitDocument` wiring: type
   normalisation on input, filename and folder validation before write.
3. **Storage** (`internal/storage/document_store.go`) — `RecordToDocument` normalisation
   on load; `validateDocumentRecord` accepts legacy types without error.

Out of scope: changes to any other document action (`approve`, `validate`, `get`, `list`,
`content`, `supersede`, `import`, `chain`); changes to the document record ID format or
storage layout; any network I/O or configuration file reads in the validation path.

---

## Task Breakdown

### T1 — Update document type model

**Objective**

Introduce the four new type constants (`spec`, `review`, `retro`, `proposal`). Revise
`AllDocumentTypes()` to return exactly the eight user-facing types. Add
`NormaliseDocumentType()` for synonym resolution (`specification`→`spec`,
`retrospective`→`retro`). Ensure `ValidDocumentType()` continues to accept all types
needed by the storage layer (user-facing, internal, and legacy). Add a registration-scoped
validator that accepts user-facing types plus `policy` and `rca` but excludes `plan`.

**Specification references:** REQ-001, REQ-002, REQ-003, REQ-004, REQ-014

**Input context**

- `internal/model/entities.go` — current `DocumentType` constants, `AllDocumentTypes()`,
  `ValidDocumentType()`.

**Output artifacts**

- `internal/model/entities.go` updated with:
  - `DocumentTypeSpec`, `DocumentTypeReview`, `DocumentTypeRetro`, `DocumentTypeProposal`
    constants.
  - Legacy constants `DocumentTypeSpecification`, `DocumentTypeRetrospective`,
    `DocumentTypePlan` retained for backwards compatibility with existing call sites.
  - `AllDocumentTypes()` returns `[design, spec, dev-plan, review, report, research,
    retro, proposal]` — the eight user-facing types.
  - `NormaliseDocumentType(s string) DocumentType` — maps `specification`→`spec`,
    `retrospective`→`retro`; passes all other values through unchanged.
  - `ValidDocumentType(s string) bool` — accepts all types (user-facing + internal +
    legacy) for use by the storage layer.
  - `ValidDocumentTypeForRegistration(s string) bool` — accepts user-facing + `policy` +
    `rca`; rejects `plan`, `specification`, `retrospective` (callers normalise first).

**Dependencies:** none

**Effort:** 2 points

---

### T2 — Implement filename and folder validation helpers

**Objective**

Add pure functions that validate the filename template and folder placement rules defined
in REQ-005 through REQ-010. These helpers contain no side effects and perform no I/O;
they operate on path strings only.

Filename validation rules:

- Files under `work/{PlanID}-{plan-slug}/` must match `{PlanID}-{type}[-{slug}].{ext}` or
  `{PlanID}-F{n}-{type}[-{slug}].{ext}` (case-insensitive on the plan ID component).
- Files under `work/_project/` must match `{type}[-{slug}].{ext}`.
- Files under `work/templates/` are fully exempt from all validation.
- Files under `docs/` are exempt from folder validation only.

Folder validation rule:

- Extract the plan ID prefix from the filename (if present). The containing folder must
  be `work/{PlanID}-{any-slug}/` with a case-insensitive match on the plan ID component.
- If the filename begins with a recognised type (no plan ID prefix), the file must be in
  `work/_project/`.

Error messages must be actionable: they must name the specific expected filename pattern
or folder path, not restate the rule abstractly (REQ-008, REQ-NF-002).

**Specification references:** REQ-005, REQ-006, REQ-007, REQ-008, REQ-009, REQ-010

**Input context**

- `internal/service/documents.go` — existing `SubmitDocument` for placement context.
- `internal/model/entities.go` — `AllDocumentTypes()` for recognised type strings used in
  filename parsing (post-T1).

**Output artifacts**

- New unexported functions in `internal/service/documents.go` or a companion file
  `internal/service/doc_path_validation.go`:
  - `validateDocumentFilename(repoRoot, docPath string) error`
  - `validateDocumentFolder(repoRoot, docPath string) error`
- Both functions return `nil` for exempt paths.
- Both functions return descriptive errors naming the specific expected pattern or
  directory.

**Dependencies:** T1 (needs the final set of recognised type strings for filename parsing)

**Effort:** 3 points

---

### T3 — Wire validation into SubmitDocument

**Objective**

Update `SubmitDocument` in `internal/service/documents.go` to apply type normalisation,
use the registration-scoped type validator, and call the filename and folder validation
helpers before writing the document record.

Changes:

1. Normalise the incoming type via `NormaliseDocumentType` before validation.
2. Validate the normalised type via `ValidDocumentTypeForRegistration`; on failure, return
   an error message listing the eight user-facing types only (no `policy`, no `rca`).
3. After the file-existence check, call `validateDocumentFilename` and
   `validateDocumentFolder`; propagate any errors immediately.
4. Store the normalised type in the document record (e.g. `spec`, not `specification`).
5. Update the entity hook switch statement to match `DocumentTypeSpec` (`"spec"`) where it
   currently matches `DocumentTypeSpecification`, so lifecycle transitions continue to work
   after normalisation.

**Specification references:** REQ-001, REQ-002, REQ-003, REQ-004, REQ-005, REQ-007,
REQ-008, REQ-009, REQ-010, REQ-NF-001, REQ-NF-002

**Input context**

- `internal/service/documents.go` — full `SubmitDocument` implementation.
- T1 deliverables: `NormaliseDocumentType`, `ValidDocumentTypeForRegistration`,
  `AllDocumentTypes`.
- T2 deliverables: `validateDocumentFilename`, `validateDocumentFolder`.

**Output artifacts**

- `internal/service/documents.go` `SubmitDocument` updated as described.
- Error message for invalid type enumerates exactly: `design`, `spec`, `dev-plan`,
  `review`, `report`, `research`, `retro`, `proposal`.

**Dependencies:** T1, T2

**Effort:** 2 points

---

### T4 — Normalise legacy types on deserialisation

**Objective**

Update the storage layer so that existing document records stored with legacy type strings
are silently normalised when loaded into memory. The on-disk YAML is not modified.

Changes:

- In `RecordToDocument` (`internal/storage/document_store.go`): after reading the `type`
  field from `record.Fields`, call `model.NormaliseDocumentType` before assigning to
  `doc.Type`. This normalises `specification`→`spec` and `retrospective`→`retro`
  transparently on every load.
- In `validateDocumentRecord` (`internal/storage/document_store.go`): confirm that
  `model.ValidDocumentType` returns `true` for `plan` so legacy records load without
  error. (T1 retains the `plan` constant; verify and add a comment.)
- Audit any other code path that reads `record.Fields["type"]` directly to confirm
  normalisation is applied everywhere.

**Specification references:** REQ-011, REQ-012, REQ-013, REQ-014

**Input context**

- `internal/storage/document_store.go` — `RecordToDocument`, `validateDocumentRecord`.
- T1 deliverables: `NormaliseDocumentType`, `ValidDocumentType`.

**Output artifacts**

- `internal/storage/document_store.go` `RecordToDocument` applies `NormaliseDocumentType`
  on the type field after reading from storage.
- Confirmed (via comment): `ValidDocumentType("plan")` returns `true`.

**Dependencies:** T1

**Effort:** 1 point

---

### T5 — Tests covering all 23 acceptance criteria

**Objective**

Write unit tests and one integration test covering all 23 acceptance criteria from the
spec. Tests live in `_test.go` files alongside the packages under test, following project
conventions in `refs/testing.md`.

**Specification references:** all 23 ACs

**Input context**

- All T1–T4 deliverables.
- `internal/service/documents_test.go` — existing test patterns to follow.
- `internal/storage/document_store_test.go` — existing integration test patterns.
- Spec acceptance criteria table at
  `work/design/p37-f2-spec-doc-type-and-filename-enforcement.md §Acceptance Criteria`.

**Output artifacts**

- `internal/service/documents_test.go` (or companion `doc_path_validation_test.go`)
  updated with new test cases for AC-001 through AC-017 and AC-023.
- `internal/storage/document_store_test.go` updated with test cases for AC-018 through
  AC-021.
- If `validateDocumentFilename`/`validateDocumentFolder` are unexported, an
  `export_test.go` in the service package exposes them for testing.
- All tests pass `go test -race ./...` and `go vet ./...`.

**Dependencies:** T1, T2, T3, T4

**Effort:** 5 points

---

## Interface Contracts

The following contracts are agreed between tasks so they can proceed independently once T1
is complete.

### `model.NormaliseDocumentType` (T1 → T3, T4)

```go
// NormaliseDocumentType maps legacy and synonym type strings to their canonical
// form. Unknown values are returned unchanged.
//   "specification" → DocumentTypeSpec ("spec")
//   "retrospective" → DocumentTypeRetro ("retro")
//   all others      → unchanged
func NormaliseDocumentType(s string) DocumentType
```

### `model.ValidDocumentTypeForRegistration` (T1 → T3)

```go
// ValidDocumentTypeForRegistration returns true if s is a type that may be
// supplied to doc register. Accepts the eight user-facing types plus "policy"
// and "rca". Does NOT accept "plan", "specification", or "retrospective"
// (callers must normalise before calling).
func ValidDocumentTypeForRegistration(s string) bool
```

### `model.ValidDocumentType` (T1 → T4, storage layer)

```go
// ValidDocumentType returns true if s is any recognised document type,
// including internal types (policy, rca) and legacy types (plan,
// specification, retrospective). Used by the storage layer.
func ValidDocumentType(s string) bool
```

### `model.AllDocumentTypes` (T1 → T3 error message)

```go
// AllDocumentTypes returns the eight user-facing document types in display
// order: design, spec, dev-plan, review, report, research, retro, proposal.
// This list is used to construct validation error messages; it excludes
// internal and legacy types.
func AllDocumentTypes() []DocumentType
```

### `validateDocumentFilename` / `validateDocumentFolder` (T2 → T3)

```go
// validateDocumentFilename checks that the filename of docPath conforms to the
// canonical template for its location. Returns nil for exempt paths
// (work/templates/, any path not under work/). Returns a descriptive error
// naming the expected filename pattern on failure.
func validateDocumentFilename(repoRoot, docPath string) error

// validateDocumentFolder checks that docPath resides in the folder that
// corresponds to the plan ID in its filename. Returns nil for exempt paths
// (docs/, work/templates/). Returns a descriptive error naming the expected
// directory on failure.
func validateDocumentFolder(repoRoot, docPath string) error
```

---

## Dependency Graph

```
T1 (model)
├── T2 (validation helpers)       depends on T1
│   └── T3 (wire SubmitDocument)  depends on T1 + T2
└── T4 (storage normalisation)    depends on T1  [parallel with T2]
T5 (tests)                        depends on T1 + T2 + T3 + T4
```

T2 and T4 can be worked in parallel once T1 is complete. T3 can begin once T2 is done.
T5 is the final task and blocks on all prior tasks being complete.

---

## Risk Assessment

**R1 — Breaking `ValidDocumentType` for storage validation**

Changing `AllDocumentTypes()` to return only the 8 user-facing types would silently break
`validateDocumentRecord` in the storage layer, which calls `ValidDocumentType()`. If
`policy`, `rca`, or legacy types are no longer returned by `AllDocumentTypes()`, existing
records with those types would fail to write. Mitigation: decouple `ValidDocumentType()`
from `AllDocumentTypes()` internally; maintain a separate broad list for storage validation
that includes all types (user-facing + internal + legacy). T1 must address this explicitly.

**R2 — Entity hook switch uses old type constant**

`SubmitDocument` has a switch on `docType` for setting document references on owning
entities (`spec` field) and triggering lifecycle transitions (`specifying`). Currently it
matches `DocumentTypeSpecification`. After T1 introduces `DocumentTypeSpec` and T3
normalises input before the switch, the case must be updated to match `DocumentTypeSpec`
(`"spec"`). Failing to update this breaks the specifying lifecycle transition for documents
registered as `type: specification` (synonym) or `type: spec` (new canonical). Mitigation:
T3 explicitly covers this switch statement update.

**R3 — Normalisation applied inconsistently across deserialisation paths**

`RecordToDocument` in `internal/storage/document_store.go` is the primary deserialisation
path. However, any code path that reads `doc.Type` directly from `record.Fields` without
going through `RecordToDocument` would miss the normalisation. Mitigation: T4 audits all
callers of `RecordToDocument` and direct `record.Fields["type"]` accesses to confirm the
normalisation is applied everywhere. If a secondary path is found, it is updated in the
same task.

---

## Traceability Matrix

| Requirement | AC(s) | Task(s) |
|-------------|-------|---------|
| REQ-001 | AC-001, AC-002, AC-003 | T1, T3 |
| REQ-002 | AC-004 | T1, T3 |
| REQ-003 | AC-005 | T1, T3 |
| REQ-004 | AC-006, AC-007, AC-008 | T1, T3 |
| REQ-005 | AC-009, AC-010, AC-011 | T2, T3 |
| REQ-006 | AC-012 | T2, T3 |
| REQ-007 | AC-013, AC-014 | T2, T3 |
| REQ-008 | AC-015 | T2, T3 |
| REQ-009 | AC-016 | T2, T3 |
| REQ-010 | AC-017 | T2, T3 |
| REQ-011 | AC-018 | T4 |
| REQ-012 | AC-019 | T1, T4 |
| REQ-013 | AC-020 | T1, T4 |
| REQ-014 | AC-021 | T1, T4 |
| REQ-NF-001 | AC-022 | T3 (code review) |
| REQ-NF-002 | AC-023 | T2, T3 |

Every requirement maps to at least one task. Every task maps to at least one requirement.

### Verification Approach

All 23 acceptance criteria map to automated tests in T5, except AC-022 which is verified
by code review during the PR review stage.

The following criteria exercise the validation helpers directly (not through
`SubmitDocument`), so the helpers must be exported or tested via package-level test
functions: AC-009 through AC-014. If the helpers are unexported, use an `export_test.go`
file in the service package to expose them for testing.

Integration test AC-018 uses a real `DocumentStore` backed by a `t.TempDir()` to write a
YAML record with a non-conforming path and confirm it loads cleanly.

All tests must pass `go test -race ./...` without data races and `go vet ./...` without
warnings before the feature is considered ready for review.