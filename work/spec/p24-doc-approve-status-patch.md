# Specification: doc approve patches Status in source file

| Field       | Value                                    |
|-------------|------------------------------------------|
| Date        | 2025-07-18                               |
| Author      | Spec Author                              |
| Feature     | FEAT-01KPPG2MYSG6A                       |
| Design      | work/design/p24-doc-approve-status-patch.md |
| Status      | Draft                                    |

---

## Problem Statement

This specification implements the design described in
`work/design/p24-doc-approve-status-patch.md`.

When `doc(action: "approve")` is called, the document record in `.kbz/state/documents/`
is updated to `status: approved`, but the Markdown source file is left unchanged. Files
that carry a `Status` field continue to display `Status: Draft` after approval, creating
a visible and confusing inconsistency between the authoritative store record and the
file content.

This specification covers:

- A `PatchStatusField` utility function in `internal/fsutil/status_patch.go` that
  rewrites the first recognisable Status field in a file to a given value.
- Integration of that function into `DocumentService.ApproveDocument` in
  `internal/service/documents.go`, with best-effort semantics and content-hash
  re-synchronisation after a successful patch.

This specification does **not** cover:

- Adding a Status field to files that do not already contain one.
- Patching the Status field on supersession, refresh, or any lifecycle transition
  other than approval.
- Normalising whitespace or column widths beyond the Status value cell.
- Stripping Status fields from managed documents (that is the `consistent-front-matter`
  migration, tracked separately).
- Patching on individual documents within a batch approve (covered automatically via
  per-document code path; no separate work required).

---

## Requirements

### Functional Requirements

- **FR-001:** `PatchStatusField(path string, newStatus string) (patched bool, err error)`
  MUST read the file at `path` line by line and replace the **first** line that matches
  any of the three supported Status field formats with a normalised replacement line
  using `newStatus` as the value. No further lines are scanned after the first
  replacement.

- **FR-002:** `PatchStatusField` MUST match Status field lines **case-insensitively**
  on the field name. The following three formats MUST be recognised:

  | Format      | Match pattern (case-insensitive on field name)  | Normalised replacement               |
  |-------------|--------------------------------------------------|---------------------------------------|
  | Pipe-table  | `^\| *status *\|[^|]+\|`                         | `\| Status \| <newStatus> \|`         |
  | Bullet list | `^- *status *:.*`                                | `- Status: <newStatus>`               |
  | Bare YAML   | `^status *:.*`                                   | `status: <newStatus>`                 |

- **FR-003:** When `PatchStatusField` finds no matching Status field line, it MUST
  return `(false, nil)` and MUST NOT write the file.

- **FR-004:** When `PatchStatusField` successfully patches a line, it MUST write the
  complete modified file content atomically (using `internal/fsutil.WriteFileAtomic`)
  and MUST return `(true, nil)`.

- **FR-005:** When `PatchStatusField` encounters a file-read or file-write error, it
  MUST return `(false, err)` describing the failure. The file MUST NOT be partially
  written.

- **FR-006:** `DocumentService.ApproveDocument` MUST call `PatchStatusField` with
  `newStatus = "approved"` immediately after successfully writing the approved document
  record to the store.

- **FR-007:** If `PatchStatusField` returns an error, `ApproveDocument` MUST log a
  WARNING (format: `[doc] WARNING: could not patch status field in <path>: <err>`) and
  MUST continue — the approval MUST NOT be rolled back.

- **FR-008:** If `PatchStatusField` returns `(true, nil)` (file was patched),
  `ApproveDocument` MUST re-compute the content hash of the patched file and perform a
  second store write to update the document record's `ContentHash` to the post-patch
  value.

- **FR-009:** If the hash re-computation or the second store write fails,
  `ApproveDocument` MUST log a WARNING and MUST continue. The approval MUST NOT be
  rolled back. The record retains the pre-patch content hash.

- **FR-010:** If `PatchStatusField` returns `(false, nil)` (no Status field found),
  `ApproveDocument` MUST NOT attempt a hash re-computation or second store write, and
  MUST NOT emit any log entry related to the patch.

- **FR-011:** The pipe-table replacement MUST preserve the outer pipe delimiters and
  normalise to `| Status | <newStatus> |`. Column widths MUST NOT be re-padded.

- **FR-012:** The bullet-list replacement MUST normalise to `- Status: <newStatus>`.

- **FR-013:** The bare-YAML replacement MUST normalise to `status: <newStatus>`.

### Non-Functional Requirements

- **NFR-001:** The patch operation MUST be best-effort with respect to approval
  success. Under no circumstances MUST a patch failure prevent the approved document
  record from being persisted.

- **NFR-002:** `PatchStatusField` MUST use `internal/fsutil.WriteFileAtomic` for all
  file writes, ensuring crash-safe, all-or-nothing replacement of the file content.

- **NFR-003:** `PatchStatusField` MUST be independently testable without invoking the
  service layer. It MUST have no dependency on `internal/service` or
  `internal/storage`.

---

## Constraints

- `PatchStatusField` MUST be placed in a new file `internal/fsutil/status_patch.go`,
  not inlined into `internal/service/documents.go`.
- The patch MUST be applied in the **service layer** (`DocumentService.ApproveDocument`),
  not in the MCP tool handler (`doc_tool.go`).
- The existing `DocumentService.ApproveDocument` function signature and its public
  contract MUST NOT change. The patch is an internal implementation detail.
- The patch applies only to the `approve` action. It MUST NOT be added to `supersede`,
  `refresh`, or any other document lifecycle method.
- The Go standard library `regexp` package MUST be used for line matching. No external
  regex dependency is permitted.
- This specification does NOT cover adding a Status field to files that lack one.
- This specification does NOT cover enforcing a canonical Status format across the
  corpus or normalising whitespace beyond the value cell.

---

## Acceptance Criteria

- **AC-001 (FR-001, FR-002):** Given a file containing `| Status | Draft |` (pipe-table
  format), when `PatchStatusField(path, "approved")` is called, then the function
  returns `(true, nil)` and the file contains `| Status | approved |`.

- **AC-002 (FR-001, FR-002):** Given a file containing `- Status: Draft` (bullet-list
  format), when `PatchStatusField(path, "approved")` is called, then the function
  returns `(true, nil)` and the file contains `- Status: approved`.

- **AC-003 (FR-001, FR-002):** Given a file containing `status: draft` (bare-YAML
  format), when `PatchStatusField(path, "approved")` is called, then the function
  returns `(true, nil)` and the file contains `status: approved`.

- **AC-004 (FR-002):** Given a file containing `| STATUS | Draft |` (upper-case field
  name), when `PatchStatusField(path, "approved")` is called, then the function
  returns `(true, nil)` and the file contains `| Status | approved |`.

- **AC-005 (FR-001, FR-003):** Given a file containing no Status field, when
  `PatchStatusField(path, "approved")` is called, then the function returns
  `(false, nil)` and the file is unchanged.

- **AC-006 (FR-001):** Given a file containing two Status field lines (e.g. a pipe-table
  row followed by a bare-YAML line), when `PatchStatusField(path, "approved")` is
  called, then only the **first** matching line is replaced and the second is left
  untouched.

- **AC-007 (FR-005):** Given a path pointing to a non-existent file, when
  `PatchStatusField(path, "approved")` is called, then the function returns
  `(false, err)` where `err` is non-nil.

- **AC-008 (FR-006, FR-007):** Given a registered document whose source file contains
  `| Status | Draft |`, when `doc(action: "approve")` is called, then:
  - The document record's `Status` is `"approved"`.
  - The source file contains `| Status | approved |`.
  - The document record's `ContentHash` matches the SHA of the patched file.

- **AC-009 (FR-007):** Given a registered document whose source file is unreadable,
  when `doc(action: "approve")` is called, then:
  - The document record's `Status` is `"approved"` (approval succeeds).
  - A WARNING is logged containing `could not patch status field`.

- **AC-010 (FR-010):** Given a registered document whose source file contains no
  Status field, when `doc(action: "approve")` is called, then:
  - The document record's `Status` is `"approved"`.
  - The source file is unchanged.
  - No patch-related WARNING is logged.

- **AC-011 (FR-008, FR-009):** Given a registered document whose source file contains
  `| Status | Draft |` and whose store write of the refreshed hash fails, when
  `doc(action: "approve")` is called, then:
  - The document record's `Status` is `"approved"`.
  - The source file contains `| Status | approved |`.
  - A WARNING is logged; the function returns without error.

- **AC-012 (NFR-002):** `PatchStatusField` writes using `WriteFileAtomic`. A test
  confirms that a simulated mid-write failure leaves the original file intact.

---

## Verification Plan

| Criterion | Method      | Description                                                                                  |
|-----------|-------------|----------------------------------------------------------------------------------------------|
| AC-001    | Unit test   | `TestPatchStatusField_PipeTable` in `internal/fsutil/status_patch_test.go`                  |
| AC-002    | Unit test   | `TestPatchStatusField_BulletList` in `internal/fsutil/status_patch_test.go`                 |
| AC-003    | Unit test   | `TestPatchStatusField_BareYAML` in `internal/fsutil/status_patch_test.go`                   |
| AC-004    | Unit test   | `TestPatchStatusField_CaseInsensitive` in `internal/fsutil/status_patch_test.go`            |
| AC-005    | Unit test   | `TestPatchStatusField_NoField` in `internal/fsutil/status_patch_test.go`                    |
| AC-006    | Unit test   | `TestPatchStatusField_FirstMatchOnly` in `internal/fsutil/status_patch_test.go`             |
| AC-007    | Unit test   | `TestPatchStatusField_FileNotFound` in `internal/fsutil/status_patch_test.go`               |
| AC-008    | Integration test | `TestApproveDocument_PatchesStatusField` in `internal/service/documents_test.go`       |
| AC-009    | Integration test | `TestApproveDocument_PatchFailure_ApprovalSucceeds` in `internal/service/documents_test.go` |
| AC-010    | Integration test | `TestApproveDocument_NoStatusField_NoSideEffects` in `internal/service/documents_test.go`  |
| AC-011    | Integration test | `TestApproveDocument_HashRefreshFailure_ApprovalSucceeds` in `internal/service/documents_test.go` |
| AC-012    | Inspection  | Code review confirms `PatchStatusField` calls `WriteFileAtomic`, not `os.WriteFile`         |

---

## Dependencies and Assumptions

- **DEP-001:** `internal/fsutil.WriteFileAtomic` exists in `internal/fsutil/atomic.go`
  and is available for use by `PatchStatusField`.
- **DEP-002:** `internal/storage.ComputeContentHash(path string) (string, error)` exists
  and is already called within `ApproveDocument`; it will be re-used for the post-patch
  hash refresh.
- **DEP-003:** `internal/storage.DocumentToRecord` is already in scope within
  `ApproveDocument` and will be used for the second store write.
- **ASM-001:** The Go standard library `regexp` package provides sufficient performance
  for line-level matching of document files at the sizes encountered in this project.
- **ASM-002:** Document source files use UTF-8 encoding; no special handling for other
  encodings is required.
- **ASM-003:** The first Status field line in a file is always the metadata Status
  declaration. Body text may contain the word "status" but will not match the supported
  formats as the first occurrence in well-formed documents.