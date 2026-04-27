# P37-F2: Document Type System and Filename Enforcement — Implementation Plan

| Field  | Value                                                                        |
|--------|------------------------------------------------------------------------------|
| Date   | 2026-04-27T14:11:10Z                                                         |
| Status | Draft                                                                        |
| Author | orchestrator                                                                 |
| Spec   | work/design/p37-f2-spec-doc-type-and-filename-enforcement.md                 |

---

## Scope

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
   on load; `validateDocumentRecord` accepts legacy types.

Out of scope: changes to any other document action (`approve`, `validate`, `get`, `list`,
`content`, `supersede`, `import`, `chain`); changes to the document record ID format or
storage layout; any network I/O or configuration file reads in the validation path.

---

## Task Breakdown

### T1 — Update document type model

**Description**

Introduce the four new type constants (`spec`, `review`, `retro`, `proposal`). Revise
`AllDocumentTypes()` to return exactly the eight user-facing types. Add
`NormaliseDocumentType()` for synonym resolution (`specification`→`spec`,
`retrospective`→`retro`). Ensure `ValidDocumentType()` continues to accept all types
needed by the storage layer (user-facing, internal, and legacy). Add a registration-scoped
validator that accepts user-facing types plus `policy` and `rca` but excludes `plan`.

**Deliverables**

- `internal/model/entities.go` updated with:
  - `DocumentTypeSpec`, `DocumentTypeReview`, `DocumentTypeRetro`, `DocumentTypeProposal`
    constants.
  - Legacy constants `DocumentTypeSpecification`, `DocumentTypeRetrospective`,
    `DocumentTypePlan` retained for backwards compatibility.
  - `AllDocumentTypes()` returns `[design, spec, dev-plan, review, report, research,
    retro, proposal]` — the eight user-facing types.
  - `NormaliseDocumentType(s string) DocumentType` — maps synonyms to canonical forms;
    passes all other values through unchanged.
  - `ValidDocumentType(s string) bool` — accepts all types (user-facing + internal +
    legacy) for use by the storage layer.
  - `ValidDocumentTypeForRegistration(s string) bool` — accepts user-facing + `policy` +
    `rca`; rejects `plan`, `specification`, `retrospective` (callers must normalise first).

**Depends on:** nothing

**Effort:** 2 points

**Spec requirements:** REQ-001, REQ-002, REQ-003, REQ-004, REQ-014

---

### T2 — Implement filename and folder validation helpers

**Description**

Add pure functions to the service layer that validate the filename template and folder
placement rules defined in REQ-005 through REQ-010. These helpers contain no side effects
and perform no I/O; they operate on path strings only.

Filename validation rules:

- Files under `work/{PlanID}-{plan-slug}/` must match `{PlanID}-{type}[-{slug}].{ext}` or
  `{PlanID}-F{n}-{type}[-{slug}].{ext}` (case-insensitive on the plan ID component).
- Files under `work/_project/` must match `{type}[-{slug}].{ext}`.
- Files under `work/templates/` are fully exempt.
- Files under `docs/` are exempt from folder validation only.

Folder validation rule:

- Extract the plan ID prefix from the filename (if present). The containing folder must
  be `work/{PlanID}-{any-slug}/` with a case-insensitive match on the plan ID component.
- If the filename begins with a type (no plan ID prefix), the file must be in
  `work/_project/`.

Error messages must be actionable: they must name the specific expected filename pattern
or folder path, not restate the rule abstractly (REQ-008, REQ-NF-002).

**Deliverables**

- New (unexported) functions in `internal/service/documents.go` (or a companion file
  `internal/service/doc_path_validation.go`):
  - `validateDocumentFilename(repoRoot, docPath string) error`
  - `validateDocumentFolder(repoRoot, docPath string) error`
- Both functions return `nil` for exempt paths.
- Both functions return descriptive errors for violated rules, including the specific
  expected pattern or directory.

**Depends on:** T1 (needs the final set of recognised type strings for filename parsing)

**Effort:** 3 points

**Spec requirements:** REQ-005, REQ-006, REQ-007, REQ-008, REQ-009, REQ-010

---

### T3 — Wire validation into SubmitDocument

**Description**

Update `SubmitDocument` in `internal/service/documents.go` to:

1. Normalise the incoming type via `NormaliseDocumentType` before validation.
2. Validate the normalised type via `ValidDocumentTypeForRegistration`; on failure, return
   an error message listing the eight user-facing types only (no `policy`, no `rca`).
3. After the file-existence check, call `validateDocumentFilename` and
   `validateDocumentFolder`; propagate any errors immediately before writing the record.
4. Store the normalised type in the document record (e.g. `spec`, not `specification`).

Also update the `auto_approve` whitelist to include the new `review`, `retro`, `proposal`
types if appropriate (check against spec — if not specified, leave as-is).

Update the entity hook switch statement to recognise `DocumentTypeSpec` where it currently
recognises `DocumentTypeSpecification`, so that lifecycle transitions continue to work
after normalisation.

**Deliverables**

- `internal/service/documents.go` `SubmitDocument` updated as above.
- Error message for invalid type enumerates exactly: `design`, `spec`, `dev-plan`,
  `review`, `report`, `research`, `retro`, `proposal`.

**Depends on:** T1, T2

**Effort:** 2 points

**Spec requirements:** REQ-001, REQ-002, REQ-003, REQ-004, REQ-005, REQ-007, REQ-008,
REQ-009, REQ-010, REQ-NF-001, REQ-NF-002

---

### T4 — Normalise legacy types on deserialisation

**Description**

Update the storage layer so that existing document records stored with legacy type strings
are silently normalised when loaded into memory. The on-disk YAML is not modified.

Changes:

- In `RecordToDocument` (`internal/storage/document_store.go`): after reading the `type`
  field from `record.Fields`, apply `model.NormaliseDocumentType` before assigning to
  `doc.Type`. This normalises `specification`→`spec` and `retrospective`→`retro`
  transparently on every load.
- In `validateDocumentRecord` (`internal/storage/document_store.go`): the validation
  function is called only in `Write()`, so it never sees records with the legacy `plan`
  type from old files. However, ensure `model.ValidDocumentType` still returns `true` for
  `plan` so that any direct call path does not break. (This is already the case given T1
  retains legacy constants; confirm and add a comment.)

**Deliverables**

- `internal/storage/document_store.go` `RecordToDocument` applies normalisation on the
  type field.
- Confirmed: `ValidDocumentType("plan")` returns `true` (legacy records load without
  error).

**Depends on:** T1 (needs `NormaliseDocumentType`)

**Effort:** 1 point

**Spec requirements:** REQ-011, REQ-012, REQ-013, REQ-014

---

### T5 — Tests covering all 23 acceptance criteria

**Description**

Write unit tests and one integration test covering all 23 acceptance criteria from the
spec. Tests live in `_test.go` files alongside the packages under test, following project
conventions in `refs/testing.md`.

Test plan by acceptance criterion:

| AC    | Package under test         | Test description |
|-------|----------------------------|------------------|
| AC-001 | `service` | `SubmitDocument` with `type: review` — no error, stored `Type == "review"` |
| AC-002 | `service` | `SubmitDocument` with `type: proposal` — no error, stored `Type == "proposal"` |
| AC-003 | `service` | `SubmitDocument` with `type: foo` — error lists exactly 8 user-facing types, excludes `policy`/`rca` |
| AC-004 | `service` | `SubmitDocument` with `type: specification` — no error, stored `Type == "spec"` |
| AC-005 | `service` | `SubmitDocument` with `type: retrospective` — no error, stored `Type == "retro"` |
| AC-006 | `service` | `SubmitDocument` with `type: policy` — no error, stored `Type == "policy"` |
| AC-007 | `service` | `SubmitDocument` with `type: rca` — no error, stored `Type == "rca"` |
| AC-008 | `service` | `SubmitDocument` with `type: foo` — error string does not contain `"policy"` or `"rca"` |
| AC-009 | `service` | Filename validator: `work/P37-file-names-actions/P37-spec-something.md` — passes |
| AC-010 | `service` | Filename validator: `work/P37-file-names-actions/spec-something.md` — fails |
| AC-011 | `service` | Filename validator: `work/P37-file-names-actions/P37-F2-spec-enforcement.md` — passes |
| AC-012 | `service` | Filename validator: `work/p37-file-names-actions/p37-spec-something.md` — passes (case-insensitive) |
| AC-013 | `service` | Folder validator: `work/P25-other-plan/P37-spec-something.md` — fails |
| AC-014 | `service` | Folder validator: `work/_project/research-ai-orchestration.md` — passes |
| AC-015 | `service` | `SubmitDocument` invalid filename — error contains specific expected pattern |
| AC-016 | `service` | `SubmitDocument` `work/templates/my-template.md` — validation skipped, succeeds |
| AC-017 | `service` | `SubmitDocument` `docs/architecture/overview.md` — no folder validation error |
| AC-018 | `storage` (integration) | Load record with non-conforming path — no error, all fields accessible |
| AC-019 | `storage` | `RecordToDocument` with `type: specification` — `Type == "spec"`, no error |
| AC-020 | `storage` | `RecordToDocument` with `type: retrospective` — `Type == "retro"`, no error |
| AC-021 | `storage` | `RecordToDocument` with `type: plan` — no error, record accessible |
| AC-022 | code review | No `exec`, network, or config reads in validation code path |
| AC-023 | `service` | `SubmitDocument` wrong folder — error contains specific expected directory |

AC-022 is verified by code review during the PR review stage, not by an automated test.

**Deliverables**

- `internal/service/documents_test.go` (or companion `doc_path_validation_test.go`)
  updated with new test cases for AC-001 through AC-017 and AC-023.
- `internal/storage/document_store_test.go` updated with test cases for AC-018 through
  AC-021.

**Depends on:** T1, T2, T3, T4

**Effort:** 5 points

**Spec requirements:** all 23 ACs

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

**R2 — Entity hook switch statement uses old type constant**

`SubmitDocument` has a switch on `docType` for setting document references on owning
entities (`spec` field) and triggering lifecycle transitions (`specifying`). Currently it
matches `DocumentTypeSpecification`. After T1 introduces `DocumentTypeSpec` and T3
normalises input before the switch, the case must be updated to match `DocumentTypeSpec`
(`"spec"`). Failing to update this breaks the specifying lifecycle transition for documents
registered as `type: specification` (synonym) or `type: spec` (new canonical). Mitigation:
T3 explicitly covers this switch statement update.

**R3 — Normalisation in RecordToDocument applied inconsistently across code paths**

`RecordToDocument` in `internal/storage/document_store.go` is the primary deserialisation
path. However, any code path that reads `doc.Type` directly from `record.Fields` without
going through `RecordToDocument` would miss the normalisation. Mitigation: T4 audits all
callers of `RecordToDocument` and direct `record.Fields["type"]` accesses to confirm the
normalisation is applied everywhere. If a secondary path is found, it is updated in the
same task.

---

## Verification Approach

All 23 acceptance criteria map to automated tests in T5, except AC-022 which is verified
by code review. The mapping is stated in the T5 test plan table above.

The following criteria exercise the validation helpers directly (not through
`SubmitDocument`), so the helpers must be exported or tested via package-level test
functions: AC-009 through AC-014. If the helpers are unexported, use an `export_test.go`
file in the service package to expose them for testing.

Integration test AC-018 uses a real `DocumentStore` backed by a `t.TempDir()` to write a
YAML record with a non-conforming path and confirm it loads cleanly.

All tests must pass `go test -race ./...` without data races and `go vet ./...` without
warnings before the feature is considered ready for review.