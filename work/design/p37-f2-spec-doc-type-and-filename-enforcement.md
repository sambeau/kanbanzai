# P37-F2: Document Type System and Filename Enforcement — Specification

| Field      | Value                                              |
|------------|----------------------------------------------------|
| Date       | 2026-04-27T12:34:44Z                               |
| Status     | Draft                                              |
| Author     | sambeau                                            |
| Feature    | FEAT-01KQ7JDSZARPC                                 |
| Plan       | P37-file-names-and-actions                         |
| Design ref | D1, D3, D4 in `work/design/p37-file-names-and-actions.md` |

---

## Problem Statement

The document type system has accumulated inconsistencies that impede consistent discovery,
reference, and automation:

1. **Type name inconsistency.** The stored types `specification` and `retrospective` are
   long-form strings that are inconsistent with the short-form prefixes used in filenames
   (`spec-`, `retro-`). No canonical short form is enforced, so both forms appear in
   practice with no normalisation.

2. **Missing types.** Formal review documents (conformance, quality, security, testing) are
   stored as type `report`, conflating structured workflow outputs with internal analysis
   documents. There is no `proposal` type for early-stage design documents that precede
   formal design work.

3. **No filename enforcement.** The `doc register` action accepts any filename. AI agents
   and humans produce inconsistent filenames (e.g., `spec-thing.md`, `SPEC_thing.md`,
   `P24-spec_thing.md`), making programmatic discovery and reference unreliable.

4. **No folder enforcement.** Documents are registered from any directory. Plan-related
   documents may be scattered across unrelated folders, undermining the plan-first
   organisation introduced in design decision D1.

These issues compound over time: each misnamed or misplaced document makes the repository
harder to navigate and requires bespoke workarounds in tooling and agent instructions.

---

## Requirements

### Functional Requirements

**REQ-001:** The system SHALL recognise exactly eight user-facing document types: `design`,
`spec`, `dev-plan`, `review`, `report`, `research`, `retro`, `proposal`.

**REQ-002:** `doc register` SHALL accept `specification` as a synonym for `spec`. When the
caller supplies `specification` as the type, the stored document type SHALL be normalised
to `spec`.

**REQ-003:** `doc register` SHALL accept `retrospective` as a synonym for `retro`. When
the caller supplies `retrospective` as the type, the stored document type SHALL be
normalised to `retro`.

**REQ-004:** `policy` and `rca` SHALL remain valid types accepted by `doc register` and
all other document actions. They SHALL be treated as internal types: they SHALL NOT appear
in user-facing validation error messages that enumerate accepted types, and they SHALL NOT
be added to the public type schema exposed to external callers.

**REQ-005:** `doc register` SHALL validate that the filename of the document being
registered matches one of the following canonical templates, depending on the file's
location:

| Location                       | Required filename template                      |
|--------------------------------|-------------------------------------------------|
| `work/{PlanID}-{plan-slug}/`   | `{PlanID}-{type}[-{slug}].{ext}`                |
| `work/{PlanID}-{plan-slug}/`   | `{PlanID}-F{n}-{type}[-{slug}].{ext}` (feature-scoped) |
| `work/_project/`               | `{type}[-{slug}].{ext}`                         |

Where:
- `{PlanID}` is the letter `P` followed by one or more digits (e.g., `P37`).
- `F{n}` is the letter `F` followed by one or more digits (e.g., `F2`).
- `{type}` is a recognised user-facing document type or accepted internal type.
- `[-{slug}]` denotes an optional suffix: a hyphen followed by one or more `[a-z0-9-]`
  characters.
- `{ext}` is a non-empty file extension (e.g., `.md`).

**REQ-006:** Plan ID prefix matching during filename validation SHALL be case-insensitive.
A filename beginning with `p37-` SHALL be accepted in the same contexts as one beginning
with `P37-`.

**REQ-007:** `doc register` SHALL validate that the document file is located in the folder
that corresponds to the plan ID extracted from its filename:
- A file whose filename begins with `P{n}-` (or `p{n}-`) MUST reside within
  `work/P{n}-{any-slug}/` (matched case-insensitively on the plan ID component).
- A file whose filename begins with a recognised type (no plan ID prefix) MUST reside
  within `work/_project/`.

**REQ-008:** When filename or folder validation fails, the returned error SHALL include:
  (a) the validation rule that was violated (filename format or folder location), and
  (b) the specific expected filename pattern or folder path for the registration context
      being attempted.

**REQ-009:** Files whose resolved path begins with `work/templates/` (relative to the
repository root) SHALL be exempt from all filename and folder validation. Registration
proceeds for such files if the file exists on disk.

**REQ-010:** Files whose resolved path begins with `docs/` (relative to the repository
root) SHALL be exempt from folder validation. These are output documents, not work
documents, and are not required to reside under `work/`.

**REQ-011:** When loading existing document records from storage, the system SHALL NOT
apply filename or folder validation. Existing records MUST be accepted as-is regardless
of the path they were registered with.

**REQ-012:** When loading an existing document record whose stored `type` field is
`specification`, the system SHALL return the normalised type `spec`. The stored record on
disk SHALL NOT be modified.

**REQ-013:** When loading an existing document record whose stored `type` field is
`retrospective`, the system SHALL return the normalised type `retro`. The stored record on
disk SHALL NOT be modified.

**REQ-014:** When loading an existing document record whose stored `type` field is `plan`
(a legacy internal type present in very old records), the system SHALL load the record
without error. The `plan` type SHALL be treated as a valid internal legacy type during
deserialisation.

### Non-Functional Requirements

**REQ-NF-001:** All type normalisation and filename/folder validation SHALL be implemented
as in-memory string operations. They SHALL NOT invoke external processes, perform network
I/O, or read from any configuration source beyond what is already available in the
`SubmitDocument` call context.

**REQ-NF-002:** Error messages produced by filename and folder validation failures SHALL be
actionable. They SHALL name the specific expected path or filename pattern rather than
restate the violated rule in abstract terms.

---

## Constraints

**C-001:** The document record ID format SHALL NOT change.

**C-002:** The storage layout for document records under `.kbz/state/documents/` SHALL NOT
change.

**C-003:** Filename and folder validation SHALL only be applied in the `doc register`
action. The actions `doc validate`, `doc refresh`, `doc approve`, `doc get`, `doc list`,
`doc content`, `doc supersede`, `doc import`, and `doc chain` SHALL NOT have filename or
folder validation added.

**C-004:** Existing registered documents with type `policy` or `rca` SHALL load and
function without modification or error after this change is implemented.

**C-005:** The internal legacy type `plan` SHALL be treated as a valid type when
deserialising document records from storage. It need not be accepted as input to
`doc register`.

---

## Acceptance Criteria

**AC-001 (REQ-001):** Given a `doc register` call with `type: review` and a valid file
path, When the call is processed, Then no error is returned and the stored document record
has `type: review`.

**AC-002 (REQ-001):** Given a `doc register` call with `type: proposal` and a valid file
path, When the call is processed, Then no error is returned and the stored document record
has `type: proposal`.

**AC-003 (REQ-001):** Given a `doc register` call with an unrecognised type such as
`type: foo`, When the error message is inspected, Then it lists exactly the eight
user-facing types (`design`, `spec`, `dev-plan`, `review`, `report`, `research`, `retro`,
`proposal`) and does not include `policy` or `rca`.

**AC-004 (REQ-002):** Given a `doc register` call with `type: specification` and a valid
file path, When the call is processed, Then no error is returned and the stored document
record has `type: spec`.

**AC-005 (REQ-003):** Given a `doc register` call with `type: retrospective` and a valid
file path, When the call is processed, Then no error is returned and the stored document
record has `type: retro`.

**AC-006 (REQ-004):** Given a `doc register` call with `type: policy` and a valid file
path, When the call is processed, Then no error is returned and the stored document record
has `type: policy`.

**AC-007 (REQ-004):** Given a `doc register` call with `type: rca` and a valid file path,
When the call is processed, Then no error is returned and the stored document record has
`type: rca`.

**AC-008 (REQ-004):** Given a `doc register` call with an unrecognised type, When the
error message is inspected, Then neither `policy` nor `rca` appear in the list of
acceptable types presented to the caller.

**AC-009 (REQ-005):** Given a file at
`work/P37-file-names-actions/P37-spec-something.md`, When `doc register` is called for it
with `type: spec`, Then filename validation passes and no filename-related error is
returned.

**AC-010 (REQ-005):** Given a file at
`work/P37-file-names-actions/spec-something.md` (missing plan ID prefix), When
`doc register` is called for it with `type: spec`, Then filename validation fails and an
error is returned.

**AC-011 (REQ-005):** Given a file at
`work/P37-file-names-actions/P37-F2-spec-enforcement.md` (feature-scoped filename), When
`doc register` is called for it with `type: spec`, Then filename validation passes and no
filename-related error is returned.

**AC-012 (REQ-006):** Given a file at
`work/p37-file-names-actions/p37-spec-something.md` (all-lowercase plan prefix), When
`doc register` is called for it with `type: spec`, Then filename validation passes
(case-insensitive match on the plan ID prefix).

**AC-013 (REQ-007):** Given a plan-37 document physically located at
`work/P25-other-plan/P37-spec-something.md` (plan ID in filename does not match the
containing folder), When `doc register` is called, Then folder validation fails and an
error is returned.

**AC-014 (REQ-007):** Given a project-level document at
`work/_project/research-ai-orchestration.md`, When `doc register` is called for it with
`type: research`, Then folder validation passes and no folder-related error is returned.

**AC-015 (REQ-008):** Given a `doc register` call that fails filename or folder
validation, When the error message is inspected, Then it contains either the specific
expected filename pattern (e.g., `P37-spec-{slug}.md`) or the specific expected directory
path (e.g., `work/P37-file-names-actions/`).

**AC-016 (REQ-009):** Given a file at `work/templates/my-template.md`, When
`doc register` is called for it, Then filename and folder validation are skipped entirely
and registration succeeds (assuming the file exists on disk).

**AC-017 (REQ-010):** Given a file at `docs/architecture/overview.md`, When
`doc register` is called for it, Then folder validation is not performed and no
folder-related error is returned.

**AC-018 (REQ-011):** Given an existing document record whose registered path does not
conform to any current filename template, When the system loads that record from storage,
Then no validation error is raised and the full record data is returned to the caller.

**AC-019 (REQ-012):** Given an existing document record stored on disk with
`type: specification`, When the record is deserialised and returned to a caller, Then the
`type` field in the response is `spec` and no error occurs.

**AC-020 (REQ-013):** Given an existing document record stored on disk with
`type: retrospective`, When the record is deserialised and returned to a caller, Then the
`type` field in the response is `retro` and no error occurs.

**AC-021 (REQ-014):** Given an existing document record stored on disk with `type: plan`,
When the record is loaded from storage, Then no error is raised and the record is
accessible to callers.

**AC-022 (REQ-NF-001):** Given the type normalisation and filename/folder validation
implementation, When the code is reviewed, Then no calls to external processes (`exec`,
`os/exec`), network operations, or configuration file reads are present within the
validation code path.

**AC-023 (REQ-NF-002):** Given a `doc register` call that fails because the file is in
the wrong folder, When the error message is inspected, Then it includes the specific
expected directory (e.g., `expected file to be in work/P37-file-names-actions/`) rather
than only a generic rule description.

---

## Verification Plan

| Criterion | Method       | Description                                                                                                                                  |
|-----------|--------------|----------------------------------------------------------------------------------------------------------------------------------------------|
| AC-001    | Unit test    | Call `SubmitDocument` with `type: "review"` and a valid path; assert no error and stored record `Type == "review"`.                         |
| AC-002    | Unit test    | Call `SubmitDocument` with `type: "proposal"` and a valid path; assert no error and stored record `Type == "proposal"`.                     |
| AC-003    | Unit test    | Call `SubmitDocument` with `type: "foo"`; assert error returned; assert error message contains all eight user-facing types and excludes `"policy"` and `"rca"`. |
| AC-004    | Unit test    | Call `SubmitDocument` with `type: "specification"` and a valid path; assert no error and stored record `Type == "spec"`.                    |
| AC-005    | Unit test    | Call `SubmitDocument` with `type: "retrospective"` and a valid path; assert no error and stored record `Type == "retro"`.                   |
| AC-006    | Unit test    | Call `SubmitDocument` with `type: "policy"` and a valid path; assert no error and stored record `Type == "policy"`.                         |
| AC-007    | Unit test    | Call `SubmitDocument` with `type: "rca"` and a valid path; assert no error and stored record `Type == "rca"`.                               |
| AC-008    | Unit test    | Call `SubmitDocument` with `type: "foo"`; assert error message string does not contain `"policy"` or `"rca"`.                               |
| AC-009    | Unit test    | Invoke filename validator with `work/P37-file-names-actions/P37-spec-something.md`; assert validation passes.                               |
| AC-010    | Unit test    | Invoke filename validator with `work/P37-file-names-actions/spec-something.md`; assert validation fails.                                    |
| AC-011    | Unit test    | Invoke filename validator with `work/P37-file-names-actions/P37-F2-spec-enforcement.md`; assert validation passes.                          |
| AC-012    | Unit test    | Invoke filename validator with `work/p37-file-names-actions/p37-spec-something.md`; assert validation passes (case-insensitive plan prefix). |
| AC-013    | Unit test    | Invoke folder validator with plan ID `P37` and path `work/P25-other-plan/P37-spec-something.md`; assert validation fails.                   |
| AC-014    | Unit test    | Invoke folder validator with path `work/_project/research-ai-orchestration.md`; assert validation passes.                                   |
| AC-015    | Unit test    | Call `SubmitDocument` with an invalid filename; assert error message contains the specific expected filename pattern or folder path.          |
| AC-016    | Unit test    | Call `SubmitDocument` with path `work/templates/my-template.md`; assert registration succeeds without triggering filename or folder checks. |
| AC-017    | Unit test    | Call `SubmitDocument` with path `docs/architecture/overview.md`; assert no folder validation error is produced.                             |
| AC-018    | Integration test | Load a document record whose registered path does not match the current filename template; assert the record loads without error and all fields are accessible. |
| AC-019    | Unit test    | Deserialise a document record YAML with `type: specification`; assert resulting struct has `Type == "spec"` and no error is returned.        |
| AC-020    | Unit test    | Deserialise a document record YAML with `type: retrospective`; assert resulting struct has `Type == "retro"` and no error is returned.       |
| AC-021    | Unit test    | Deserialise a document record YAML with `type: plan`; assert no error is raised and record fields are accessible.                            |
| AC-022    | Code review  | Review `service/documents.go` and the new validation helpers; confirm no `exec`, network, or config-file calls in the validation code path.  |
| AC-023    | Unit test    | Call `SubmitDocument` with a file in the wrong folder; assert error message contains the specific expected directory (not a generic rule description). |