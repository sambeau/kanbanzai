# Dev Plan: doc approve patches Status in source file

| Field   | Value                                       |
|---------|---------------------------------------------|
| Date    | 2025-07-18                                  |
| Author  | Architect                                   |
| Feature | FEAT-01KPPG2MYSG6A                          |
| Spec    | work/spec/p24-doc-approve-status-patch.md   |
| Status  | Draft                                       |

---

## Scope

This plan implements the requirements defined in
`work/spec/p24-doc-approve-status-patch.md` (FEAT-01KPPG2MYSG6A/specification-p24-doc-approve-status-patch).

It covers three units of work:

1. A new `PatchStatusField` utility function and its unit tests.
2. Integration of that function into `DocumentService.ApproveDocument` with best-effort
   semantics and content-hash re-synchronisation.
3. Integration tests for the full approve-then-patch round-trip.

It does **not** cover:

- Stripping Status fields from documents (the `consistent-front-matter` migration).
- Patching on supersession or refresh.
- Any MCP-layer changes.

---

## Task Breakdown

### Task 1: Implement PatchStatusField in internal/fsutil

- **Description:** Create `internal/fsutil/status_patch.go` containing the
  `PatchStatusField(path string, newStatus string) (patched bool, err error)` function.
  The function reads the file line by line, finds the first line matching any of the
  three supported Status field formats (case-insensitive on field name), replaces it
  with the normalised form, and writes the result atomically using `WriteFileAtomic`.
  Returns `(false, nil)` when no Status field is found. Returns `(false, err)` on
  read/write failure.
- **Deliverable:** `internal/fsutil/status_patch.go`
- **Depends on:** None
- **Effort:** Small
- **Spec requirements:** FR-001, FR-002, FR-003, FR-004, FR-005, FR-011, FR-012, FR-013,
  NFR-002, NFR-003

**Interface contract (consumed by Task 2):**

```
// PatchStatusField reads the file at path, replaces the first recognisable
// Status field value with newStatus, and writes the result atomically.
// Returns (false, nil) if no Status field is found — the file is not written.
// Returns (true, nil) on a successful patch.
// Returns (false, err) on any read or write error.
func PatchStatusField(path string, newStatus string) (patched bool, err error)
```

Supported formats and their normalised replacements (matched case-insensitively on field name):

| Format      | Match regex (Go)              | Replacement                   |
|-------------|-------------------------------|-------------------------------|
| Pipe-table  | `(?i)^\| *status *\|[^|]+\|`  | `\| Status \| <newStatus> \|` |
| Bullet list | `(?i)^- *status *:.*`         | `- Status: <newStatus>`       |
| Bare YAML   | `(?i)^status *:.*`            | `status: <newStatus>`         |

First match wins; scanning stops after the first replacement.

**Input context:**
- `internal/fsutil/atomic.go` — `WriteFileAtomic` signature and semantics
- FR-001 through FR-005, FR-011–FR-013 in the spec

---

### Task 2: Write unit tests for PatchStatusField

- **Description:** Create `internal/fsutil/status_patch_test.go` with table-driven unit
  tests covering all format variants, case-insensitivity, first-match-only behaviour,
  no-field-found path, and file-not-found error path.
- **Deliverable:** `internal/fsutil/status_patch_test.go`
- **Depends on:** Task 1
- **Effort:** Small
- **Spec requirements:** AC-001, AC-002, AC-003, AC-004, AC-005, AC-006, AC-007, AC-012

Test functions to implement:

| Test function                        | Acceptance criterion |
|--------------------------------------|----------------------|
| `TestPatchStatusField_PipeTable`     | AC-001               |
| `TestPatchStatusField_BulletList`    | AC-002               |
| `TestPatchStatusField_BareYAML`      | AC-003               |
| `TestPatchStatusField_CaseInsensitive` | AC-004             |
| `TestPatchStatusField_NoField`       | AC-005               |
| `TestPatchStatusField_FirstMatchOnly` | AC-006              |
| `TestPatchStatusField_FileNotFound`  | AC-007               |

AC-012 (atomic write) is verified by inspection during code review; no additional test
function is required beyond confirming the call to `WriteFileAtomic` in the source.

**Input context:**
- `internal/fsutil/status_patch.go` (Task 1 deliverable)
- `internal/fsutil/atomic.go` — for understanding write semantics
- `refs/testing.md` — project test conventions

---

### Task 3: Integrate PatchStatusField into ApproveDocument

- **Description:** Modify `internal/service/documents.go` to call
  `fsutil.PatchStatusField(fullPath, "approved")` immediately after the successful
  `s.store.Write(updatedRecord)` call in `ApproveDocument` (around line 491). If the
  patch succeeds, re-compute the content hash and perform a second store write to keep
  the record in sync. Both the patch and the hash-refresh write are best-effort: log a
  WARNING on failure and continue in all cases. Add the `internal/fsutil` import if not
  already present.
- **Deliverable:** Modified `internal/service/documents.go`
- **Depends on:** Task 1
- **Effort:** Small
- **Spec requirements:** FR-006, FR-007, FR-008, FR-009, FR-010, NFR-001, NFR-004

**Insertion point:** After the block ending at `s.store.Write(updatedRecord)` (currently
around line 491 in `documents.go`), before the `result := DocumentResult{...}` block.

**Pseudocode for the inserted block:**

```
if patched, patchErr := fsutil.PatchStatusField(fullPath, "approved"); patchErr != nil {
    log.Printf("[doc] WARNING: could not patch status field in %s: %v", doc.Path, patchErr)
} else if patched {
    if newHash, hashErr := storage.ComputeContentHash(fullPath); hashErr == nil {
        doc.ContentHash = newHash
        patchedRecord := storage.DocumentToRecord(doc, record.FileHash)
        if _, writeErr := s.store.Write(patchedRecord); writeErr != nil {
            log.Printf("[doc] WARNING: could not update content hash after status patch for %s: %v", doc.Path, writeErr)
        }
    }
}
```

**Input context:**
- `internal/service/documents.go` — `ApproveDocument` function, lines ~385–540
- `internal/fsutil/status_patch.go` (Task 1 deliverable)
- `internal/storage` — `ComputeContentHash`, `DocumentToRecord` (already imported)
- FR-006 through FR-010 in the spec

---

### Task 4: Write integration tests for ApproveDocument patch behaviour

- **Description:** Add test functions to `internal/service/documents_test.go` (or a new
  `internal/service/documents_status_patch_test.go` if the existing file is already
  large) that exercise the full approve-then-patch round-trip through the service layer.
  Each test creates a real temporary file, registers a document pointing at it, calls
  `ApproveDocument`, then asserts on file content, record status, and content hash.
- **Deliverable:** New test functions in `internal/service/documents_test.go` (or
  `documents_status_patch_test.go`)
- **Depends on:** Task 2, Task 3
- **Effort:** Medium
- **Spec requirements:** AC-008, AC-009, AC-010, AC-011

Test functions to implement:

| Test function                                          | Acceptance criterion |
|--------------------------------------------------------|----------------------|
| `TestApproveDocument_PatchesStatusField`               | AC-008               |
| `TestApproveDocument_PatchFailure_ApprovalSucceeds`    | AC-009               |
| `TestApproveDocument_NoStatusField_NoSideEffects`      | AC-010               |
| `TestApproveDocument_HashRefreshFailure_ApprovalSucceeds` | AC-011            |

For AC-009 and AC-011, use a read-only file or a stub store to simulate the failure
paths. Examine existing test patterns in `documents_test.go` and
`documents_quality_test.go` for how `NewDocumentService`, `t.TempDir()`, and stub
stores are used.

**Input context:**
- `internal/service/documents.go` (Task 3 deliverable)
- `internal/service/documents_test.go` — existing test patterns
- `internal/service/documents_quality_test.go` — stub/config patterns
- `refs/testing.md` — project test conventions

---

## Dependency Graph

```
Task 1: Implement PatchStatusField           (no dependencies)
Task 2: Unit tests for PatchStatusField      → depends on Task 1
Task 3: Integrate into ApproveDocument       → depends on Task 1
Task 4: Integration tests                    → depends on Task 2, Task 3
```

Parallel groups:
- **Group A (independent):** Task 1
- **Group B (after Task 1):** Task 2 and Task 3 can run in parallel
- **Group C (after B):** Task 4

Critical path: Task 1 → Task 3 → Task 4

---

## Risk Assessment

### Risk: Regex match too broad in bare-YAML format

- **Probability:** Low
- **Impact:** Medium — spurious replacement of a body line containing `status:` if it
  appears before the metadata Status line.
- **Mitigation:** The spec constrains matching to the first occurrence. The bare-YAML
  pattern `(?i)^status *:.*` matches only at the start of a line; in well-formed
  documents the metadata block appears at the top. Unit test AC-006 (first-match-only)
  catches regressions. ASM-003 in the spec acknowledges this assumption explicitly.
- **Affected tasks:** Task 1, Task 2

### Risk: ApproveDocument result ContentHash not updated in memory

- **Probability:** Low
- **Impact:** Low — the `result` struct is built after the patch block. If the patch
  updates `doc.ContentHash` but the `result` is already constructed, the caller receives
  a stale hash in the returned `DocumentResult`.
- **Mitigation:** Task 3 must ensure `doc.ContentHash` is updated before the
  `result := DocumentResult{...}` block is executed. Integration test AC-008 asserts
  that the record's stored hash matches the patched file.
- **Affected tasks:** Task 3, Task 4

### Risk: log.Printf not available in documents.go

- **Probability:** Low
- **Impact:** Minor — requires adding a `log` import.
- **Mitigation:** Check existing imports at the top of `documents.go`; add `"log"` if
  absent. No behaviour change.
- **Affected tasks:** Task 3

---

## Verification Approach

| Acceptance Criterion | Verification Method  | Producing Task |
|----------------------|----------------------|----------------|
| AC-001               | Unit test            | Task 2         |
| AC-002               | Unit test            | Task 2         |
| AC-003               | Unit test            | Task 2         |
| AC-004               | Unit test            | Task 2         |
| AC-005               | Unit test            | Task 2         |
| AC-006               | Unit test            | Task 2         |
| AC-007               | Unit test            | Task 2         |
| AC-008               | Integration test     | Task 4         |
| AC-009               | Integration test     | Task 4         |
| AC-010               | Integration test     | Task 4         |
| AC-011               | Integration test     | Task 4         |
| AC-012               | Code inspection      | Task 1         |