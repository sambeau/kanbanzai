# Design: doc approve patches Status in source file

| Field | Value |
|-------|-------|
| Date | 2025-07-18 |
| Author | Architect |
| Feature | FEAT-01KPPG2MYSG6A |
| Plan | P24-retro-recommendations |

---

## 1. Overview

When `doc(action: "approve")` is called, the document record in `.kbz/state/documents/`
is updated from `status: draft` to `status: approved`. The Markdown source file is not
touched. Human readers and agents that read files directly continue to see `Status: Draft`
even after approval, creating a confusing and misleading gap between the file and the
authoritative store record.

This design adds a best-effort Status field patch to `ApproveDocument` in the service
layer. Immediately after writing the approved record, if the source file contains a
recognisable Status field in any of the three common formats, it is updated to `approved`.
The approval succeeds regardless of whether the patch succeeds.

### Relationship to consistent-front-matter

`work/design/consistent-front-matter.md` (§2, §4) establishes that Status is mutable
workflow state and should live only in the store — not in document files. Its migration
plan (§9 Phase 2) proposes stripping Status fields from all managed documents in a
single commit. This design is a complementary short-term measure: it corrects the Status
field in files that still carry it, bridging the gap until the strip migration is
executed. Once all Status fields are stripped, this patch becomes a no-op (no Status
field found → leave file alone) and can be removed with no behaviour change.

---

## 2. Goals and Non-Goals

### Goals

- After a successful `doc approve`, if the source file contains a Status field in any
  supported format, its value is updated to `approved`.
- Approval is not blocked if the patch fails (best-effort).
- The document record's content hash is updated after patching so the file does not
  immediately show as drifted.
- All three commonly used Status field formats are handled.

### Non-Goals

- Adding a Status field to files that do not already have one.
- Patching the Status field on supersession, refresh, or any transition other than
  approval.
- Enforcing a canonical Status format or normalising surrounding whitespace beyond the
  value cell.
- Stripping Status fields from managed documents — that is the consistent-front-matter
  migration, tracked separately.
- Patching on batch approve: the same code path applies per-document, so batch is
  covered automatically.

---

## 3. Design

### 3.1 Status field formats

Three formats appear in practice. All must be matched case-insensitively on the field
name and replaced with the value `approved` (lowercase, matching the store enum):

| Format | Example | Regex pattern |
|--------|---------|---------------|
| Pipe-table row | `\| Status \| Draft \|` | `(?i)^\| *status *\|[^|]+\|` |
| Bullet list | `- Status: Draft` | `(?i)^- *status *:.*` |
| Bare YAML | `status: draft` | `(?i)^status *:.*` |

The pipe-table format is the most common in Phase 3+ documents (specs and designs). The
bullet-list format is used in Phase 1–2 documents and some bootstrap files. Bare YAML
appears occasionally in older research and policy drafts.

For the pipe-table format, the replacement preserves the outer pipe delimiters:
`| Status | approved |`. Column widths are not re-padded — the goal is correctness,
not aesthetics.

For bullet and bare-YAML formats, the replacement normalises to `- Status: approved`
and `status: approved` respectively.

Matching is line-by-line. The first match wins; no further lines are scanned after the
first replacement. This avoids spurious replacements in body text where a table or
bullet happens to contain the word "Status".

### 3.2 Implementation location

The patch lives in the **service layer**, inside `DocumentService.ApproveDocument`
(`internal/service/documents.go`). Placing it at the MCP layer (`doc_tool.go`) would
leak file-manipulation concerns into the tool handler. The service owns the document
lifecycle and already has the full file path (`fullPath`) in scope at approval time.

The patch function itself is extracted to a new file `internal/fsutil/status_patch.go`
alongside the existing `internal/fsutil/atomic.go`. This keeps the parsing and
write logic testable in isolation and avoids coupling `documents.go` to regex
manipulation.

**New function signature:**

```kanbanzai/internal/fsutil/status_patch.go#L1-5
// PatchStatusField reads the file at path, replaces the first recognisable
// Status field value with newStatus, and writes the result atomically.
// Returns (false, nil) if no Status field is found — the file is not written.
// Returns (true, nil) on a successful patch.
func PatchStatusField(path string, newStatus string) (patched bool, err error)
```

### 3.3 Service layer call site

After `s.store.Write(updatedRecord)` succeeds and before the function returns,
`ApproveDocument` calls the patch and, if the file was changed, re-reads the content
hash and does a second store write to keep the hash in sync:

```kanbanzai/internal/service/documents.go#L1-18
// Write the record (preserve fileHash for optimistic locking)
updatedRecord := storage.DocumentToRecord(doc, record.FileHash)
recordPath, err := s.store.Write(updatedRecord)
if err != nil {
    return DocumentResult{}, fmt.Errorf("write document record: %w", err)
}

// Patch Status field in source file (best-effort). If the file is patched,
// re-compute the content hash and update the record so the file does not
// immediately appear as drifted (FR-B06 side-effect guard).
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

Both the patch and the hash-refresh store write are best-effort. If either fails, a
warning is logged and the function continues. The approval has already been persisted;
the worst outcome is that the file still shows a stale Status value — the same
situation that exists today.

### 3.4 Content hash re-sync rationale

`ApproveDocument` computes `currentHash` from the file before approval (line ~410) and
stores it in the record. If the patch changes one line of the file, that hash is
immediately stale. On the next `doc refresh` or read, the system would detect drift and
demote the document back to draft — undoing the approval. The hash re-sync step
prevents this: after patching, the record is updated with the post-patch hash so the
store and file are consistent.

### 3.5 Failure handling

| Failure scenario | Behaviour |
|-----------------|-----------|
| No Status field found in file | `PatchStatusField` returns `(false, nil)`; nothing written; no log entry |
| File unreadable or write fails | `PatchStatusField` returns `(false, err)`; WARNING logged; approval record already written — approval still stands |
| Hash re-computation fails | WARNING logged; approval record retains pre-patch hash; next `doc refresh` will re-sync it |
| Second store write fails | WARNING logged; approval record retains pre-patch hash |

Approval is never rolled back due to a patch failure. The patch is cosmetic; the
authoritative state is the store record.

### 3.6 Verify-then-fix test approach

A new test in `internal/service/documents_test.go` (or a dedicated
`documents_status_patch_test.go`) exercises the full round-trip:

1. Write a temporary Markdown file containing `| Status | Draft |` in a pipe-table
   block.
2. Register the document (pointing at that file).
3. Call `ApproveDocument`.
4. Read the file back and assert it contains `| Status | approved |`.
5. Assert the document record's `ContentHash` matches the patched file's hash.
6. Assert the document record's `Status` is `"approved"`.

A parallel test exercises the bullet-list format (`- Status: Draft`) and the bare-YAML
format (`status: Draft`), and a negative-case test confirms that a file with no Status
field is left untouched and approval still succeeds.

Unit tests for `PatchStatusField` in `internal/fsutil/` cover each format variant,
case-insensitivity, the first-match-wins rule, and the no-field-found path.

---

## 4. Alternatives Considered

### 4a. Patch in the MCP layer (`docApproveOne`)

The MCP handler `docApproveOne` already has access to `docSvc.RepoRoot()` and the
document path from the result. The patch could be applied there after the service call
returns.

**Rejected.** The MCP layer is a transport adapter; file-manipulation logic belongs
in the service layer alongside the other document lifecycle operations. Placing the
patch in the MCP layer also means it would be skipped in tests that exercise the
service directly, and would need to be duplicated if a non-MCP caller ever invokes
`ApproveDocument`.

### 4b. Lazy patch on read

Instead of patching on approval, detect the mismatch at read time and rewrite the file
silently on every `doc content` or `doc get` call that notices a Status discrepancy.

**Rejected.** Lazy patching is surprising and hard to reason about. It could silently
modify files during read-only operations, making `doc get` non-idempotent and
complicating git history. The on-approval moment is the natural and expected time to
apply the update.

### 4c. Treat Status as store-only immediately (skip the patch)

Follow the `consistent-front-matter` design now: declare that Status lives only in the
store, instruct agents never to write it to files, and let existing stale values decay
naturally.

**Rejected for this feature.** The `consistent-front-matter` migration has not run; the
corpus is full of files with `Status: Draft` that confuse human readers and agents today.
The patch is a concrete short-term improvement that takes minutes to implement and
delivers immediate value. It does not conflict with the future migration — once Status
fields are stripped, the patch becomes a no-op.

### 4d. Patch all three lifecycle transitions (draft, approved, superseded)

Extend the patch to also write `draft` when a document is demoted by `doc refresh`, and
`superseded` when `doc supersede` is called.

**Deferred.** Approval is the highest-value case (it is the most visible and most
confusing drift). Supersession and demotion can be added later. Adding them now would
require plumbing the patch call into two additional service methods and test surfaces
without clear demand.

---

## 5. Dependencies

| Dependency | Notes |
|------------|-------|
| `internal/fsutil.WriteFileAtomic` | Already exists (`internal/fsutil/atomic.go`). Used by `PatchStatusField` for crash-safe writes. |
| `internal/storage.ComputeContentHash` | Already used in `ApproveDocument`. Re-used after the patch to refresh the stored hash. |
| `internal/service.DocumentService.ApproveDocument` | Call site for the patch. No interface changes required — the patch is an internal implementation detail. |
| `internal/storage.DocumentToRecord` | Used for the second record write after hash refresh. Already in scope. |
| Go `regexp` package | Standard library. Used in `PatchStatusField` for line-level matching. |