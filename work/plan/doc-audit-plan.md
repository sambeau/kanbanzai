# Implementation Plan: `doc audit` Action

| Field    | Value                                                              |
|----------|--------------------------------------------------------------------|
| Status   | Draft                                                              |
| Created  | 2026-04-01                                                         |
| Feature  | FEAT-01KN4ZPQFQN1C (doc-audit)                                     |
| Spec     | `work/spec/doc-audit.md`                                           |
| Design   | `work/design/kanbanzai-2.5-infrastructure-hardening.md` §7        |

---

## 1. Implementation Approach

The `doc audit` action is a pure addition — it introduces new behaviour without
modifying any existing `doc` action. The work divides cleanly into three tasks:

1. **Audit service logic** — the core business logic that walks directories and
   compares against the store. This is the largest unit of work and has no
   dependency on the tool layer.
2. **MCP tool dispatch** — wiring the `audit` action into the `doc` tool handler,
   accepting parameters, and formatting the response. Depends on Task 1's
   interface.
3. **Tests** — integration tests covering the full action end-to-end. Depends on
   Tasks 1 and 2.

Tasks 1 and 2 share an interface contract (§2 below) that allows them to be
developed concurrently once the contract is agreed.

```
[Task 1: Audit service] ──────────────────────────────────┐
[Task 2: MCP dispatch]  (interface contract, can overlap) ─┤
                                                            │
[Task 3: Tests] (after Tasks 1 and 2) ─────────────────────┘
```

---

## 2. Interface Contract

The audit service exposes a single function consumed by the MCP tool handler:

```go
// AuditDocuments walks the given directories (or the default set if dirs is
// empty) and compares found .md files against the document store.
// If includeRegistered is true, the registered list is populated.
func AuditDocuments(
    ctx context.Context,
    store DocumentStore,
    dirs []string,          // empty = use defaults
    includeRegistered bool,
) (*AuditResult, error)

type AuditResult struct {
    Unregistered []UnregisteredFile
    Missing      []MissingRecord
    Registered   []RegisteredFile   // populated only if includeRegistered=true
    Summary      AuditSummary
}

type UnregisteredFile struct {
    Path         string
    InferredType string
}

type MissingRecord struct {
    Path  string
    DocID string
}

type RegisteredFile struct {
    Path  string
    DocID string
}

type AuditSummary struct {
    TotalOnDisk  int
    Registered   int
    Unregistered int
    Missing      int
}
```

This contract is agreed before Tasks 1 and 2 begin so they can proceed
independently.

---

## 3. Task Breakdown

| # | Task                      | Primary Files                                   | Spec refs         |
|---|---------------------------|-------------------------------------------------|-------------------|
| 1 | Audit service logic       | `internal/service/doc_audit.go` (new)           | REQ-01–REQ-10, REQ-19–REQ-20 |
| 2 | MCP tool dispatch         | `internal/mcp/doc_tool.go`                      | REQ-11–REQ-18     |
| 3 | Integration tests         | `internal/mcp/doc_tool_test.go`                 | AC-16–AC-20       |

---

## 4. Task Details

### Task 1: Audit Service Logic

**Objective:** Implement the `AuditDocuments` function that walks document
directories, identifies unregistered files and missing records, and returns a
structured result matching the interface contract in §2.

**Specification references:** REQ-01, REQ-02, REQ-03, REQ-04, REQ-05, REQ-06,
REQ-07, REQ-08, REQ-09, REQ-10, REQ-19, REQ-20, and the invariant that
`registered + unregistered == total_on_disk`.

**Input context:**
- `work/spec/doc-audit.md` — full requirements, default directory table (§4),
  and output structure (§5.4)
- `internal/service/batch_import.go` — the existing directory-to-type inference
  mapping to reuse (do not duplicate; call or extract the shared function)
- `internal/storage/` — how to query document records by path

**Output artefacts:**
- `internal/service/doc_audit.go` — new file implementing `AuditDocuments` and
  all associated types from the interface contract
- The directory-to-type mapping must be extracted to a shared location if it is
  currently unexported in `batch_import.go`; otherwise import it directly

**Implementation notes:**
- Walk with `filepath.WalkDir`; skip non-`.md` files
- For the "missing" check: after the walk, query the store for all records whose
  paths fall under the scanned directories; those with no matching on-disk file
  are missing
- When `dirs` is empty, use the hardcoded default set from the spec §4
- The invariant `registered + unregistered == total_on_disk` must hold; add an
  assertion or comment making this explicit
- Return empty slices (not nil) for `Unregistered` and `Missing` when there are
  no results (REQ-19, REQ-20)

**Dependencies:** None — can start immediately.

---

### Task 2: MCP Tool Dispatch

**Objective:** Add an `audit` case to the `doc` tool's action dispatcher, accept
the `path` and `include_registered` parameters, call `AuditDocuments`, and
return a JSON-serialisable response matching the spec output structure.

**Specification references:** REQ-11, REQ-12, REQ-13, REQ-14, REQ-15, REQ-16,
REQ-17, REQ-18.

**Input context:**
- `work/spec/doc-audit.md` §5 (output structure) and §5.5 (parameters)
- `internal/mcp/doc_tool.go` — existing `doc` action dispatch pattern to follow
- Interface contract in §2 of this plan

**Output artefacts:**
- `internal/mcp/doc_tool.go` — new `"audit"` case in the doc tool switch

**Parameter handling:**
- `path` (string, optional): if present, pass as a single-element `dirs` slice
  to `AuditDocuments`; if absent, pass `nil`/empty
- `include_registered` (bool, optional, default `false`): pass directly

**Response serialisation:**
- Map `AuditResult` to the JSON structure defined in spec §5.4
- When `include_registered` is false, omit the `registered` array from the
  response entirely (not present, not null, not empty array)
- `summary` is always present

**Dependencies:** Requires the interface contract (§2) to be agreed; can proceed
concurrently with Task 1 once the contract is fixed.

---

### Task 3: Integration Tests

**Objective:** Verify all acceptance criteria using a temporary fixture directory
and an in-memory or test-instance document store.

**Specification references:** AC-16, AC-17, AC-18, AC-19, AC-20.

**Input context:**
- `work/spec/doc-audit.md` — acceptance criteria
- `internal/mcp/doc_tool_test.go` — existing test patterns for doc tool actions
- `internal/testutil/` — shared test helpers (temp dirs, store setup)

**Output artefacts:**
- `internal/mcp/doc_tool_test.go` — new test cases:
  - `TestDocAudit_UnregisteredFiles`: creates fixture `.md` files, none
    registered → all appear in `unregistered` with correct `inferred_type`
  - `TestDocAudit_MissingRecords`: creates a store record pointing to a
    non-existent file → appears in `missing`
  - `TestDocAudit_PathParameter`: creates files in two directories, specifies
    one via `path` → only that directory's files returned
  - `TestDocAudit_IncludeRegistered`: creates registered files → appear in
    `registered` list when flag is true, absent when false
  - `TestDocAudit_Summary_Invariant`: verifies `registered + unregistered ==
    total_on_disk` across a mixed fixture

**Dependencies:** Tasks 1 and 2 must be complete.

---

## 5. Scope Boundaries (from spec)

Carried forward from `work/spec/doc-audit.md` §3.2:

- The `audit` action is **read-only** — it must not create, update, or delete
  any store records or files.
- No automatic registration of unregistered files.
- No deletion or archiving of missing store records.
- Default directory list is hardcoded; configurability is out of scope.
- No changes to any existing `doc` action.

---

## 6. Verification

All code tasks must pass:

```
go test ./internal/mcp/...
go test ./internal/service/...
go test -race ./...
go vet ./...
```

No regressions to existing doc tool actions are acceptable. The test suite for
`doc register`, `doc approve`, `doc import`, and `doc list` must continue to
pass without modification.