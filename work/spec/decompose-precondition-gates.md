# decompose propose Precondition Gates Specification

| Document | decompose propose Precondition Gates              |
|----------|---------------------------------------------------|
| Status   | Approved                                          |
| Created  | 2026-03-28T12:04:30Z                              |
| Updated  | 2026-03-28T12:04:30Z                              |
| Feature  | FEAT-01KMT-58TV8V9C (decompose-precondition-gates) |
| Plan     | P8-decompose-reliability                          |
| Design   | `work/design/decompose-reliability.md`            |

---

## 1. Purpose

This specification defines two precondition checks added to the `decompose propose`
handler that prevent silent failure when a spec is not ready for decomposition:

1. **Spec approval gate** — the spec document must be in `approved` status before
   any parsing is attempted.
2. **Index content gate** — the document intelligence index must have non-empty
   parsed content for the spec before any task generation is attempted.

The section-header fallback path (which generated tasks named "Implement 1.
Purpose", "Implement 2. Goals", etc.) is removed entirely. There is no valid use
case for it: if the spec is approved and indexed the tool should parse it
properly; if either precondition fails the tool should stop and say so.

---

## 2. Goals

1. `decompose propose` returns a hard error if the spec document record is not
   in `approved` status.
2. `decompose propose` returns a hard error if the document intelligence index
   has no parsed content for the spec file.
3. Both error messages are specific and actionable — they tell the caller exactly
   what is wrong and what to do next.
4. The section-header fallback path is removed from the codebase.
5. Both error paths have unit tests.
6. All existing tests continue to pass.

---

## 3. Scope

### 3.1 In scope

- The `decompose propose` action handler in `internal/mcp/`.
- The internal function(s) that resolve and validate the spec document before
  parsing — approval status check and index content check added here.
- Removal of the section-header fallback code path.
- Unit tests covering both new error paths and the removal of the fallback.

### 3.2 Out of scope

- Changes to any other `decompose` actions (`review`, `apply`, `slice`).
- Changes to the document intelligence indexing pipeline or its trigger points.
- The `AGENTS.md` precondition rule — that is covered by
  FEAT-01KMT-58SKYM5C (agents-md-decompose-rule).
- The disk-read fallback (Fix 3 from the original analysis) — explicitly
  deferred; Fixes 1 and 2 together fully prevent silent failure.
- Changes to any other MCP tool handlers.

---

## 4. Acceptance Criteria

### 4.1 Spec approval gate

**AC-01.** When `decompose propose` is called and the resolved spec document
record has `status: draft` (or any status other than `approved`), the tool
returns an error. No proposal object is generated.

**AC-02.** The error message identifies the spec document ID and its current
status, and instructs the caller to approve the spec before decomposing.
Exact form:

```
spec "FEAT-.../specification-..." is in "draft" status — approve the spec before decomposing
```

**AC-03.** The approval gate fires before any attempt to read the spec content
from the document intelligence index.

**AC-04.** When the spec document record is in `approved` status, the approval
gate passes silently and execution continues to the index content check.

### 4.2 Index content gate

**AC-05.** When `decompose propose` is called, the spec document's resolved spec
finds the spec approved, and the document intelligence index returns empty or
absent parsed content for the spec file, the tool returns an error. No proposal
object is generated.

**AC-06.** The error message identifies the spec document, states that the index
has no content for it, and instructs the caller to run `index_repository` then
retry. Exact form:

```
spec content not yet indexed for "FEAT-.../specification-..." — run index_repository, then retry
```

**AC-07.** The index content gate fires after the approval gate (AC-03 ordering
is preserved).

**AC-08.** When the index has non-empty parsed content for the spec, the index
content gate passes silently and execution continues to task generation.

### 4.3 Fallback removal

**AC-09.** The section-header fallback code path — which previously generated
tasks from markdown heading text when no acceptance criteria were found — no
longer exists in `internal/mcp/`.

**AC-10.** The warning string `"No acceptance criteria found in spec; tasks
derived from section headers"` (or equivalent) is no longer present anywhere
in the codebase.

**AC-11.** No other behaviour of `decompose propose` changes: when the spec is
approved and the index has content, task generation proceeds exactly as before.

### 4.4 Testing

**AC-12.** A test covers the approval gate: calling `decompose propose` with a
spec document in `draft` status returns the expected error (AC-02 message) and
no proposal.

**AC-13.** A test covers the index content gate: calling `decompose propose`
with an approved spec document whose index entry is empty (or absent) returns
the expected error (AC-06 message) and no proposal.

**AC-14.** A test confirms that a valid input — approved spec with non-empty
index content — passes both gates and reaches the task generation stage (does
not need to produce a complete proposal; a partial stub is sufficient to confirm
the gates did not block it).

**AC-15.** All tests pass under `go test -race ./...`.

---

## 5. Error Message Reference

| Condition | Error message |
|-----------|---------------|
| Spec not approved | `spec "FEAT-.../specification-..." is in "draft" status — approve the spec before decomposing` |
| Index has no content | `spec content not yet indexed for "FEAT-.../specification-..." — run index_repository, then retry` |

Both messages must include the document ID so the caller knows which spec is
the problem without needing a separate lookup.

---

## 6. File Summary

| Path | Action |
|------|--------|
| `internal/mcp/decompose.go` (or equivalent) | Add approval gate, add index content gate, remove section-header fallback |
| `internal/mcp/decompose_test.go` (or equivalent) | Add tests for AC-12, AC-13, AC-14 |