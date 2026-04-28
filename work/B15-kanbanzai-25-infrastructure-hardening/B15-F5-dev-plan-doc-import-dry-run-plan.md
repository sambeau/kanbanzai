# Implementation Plan: `doc import` Dry-Run Mode

| Field    | Value                                                        |
|----------|--------------------------------------------------------------|
| Status   | Draft                                                        |
| Created  | 2026-04-01                                                   |
| Feature  | FEAT-01KN4ZPTQSZT5 (doc-import-dry-run)                     |
| Spec     | `work/spec/doc-import-dry-run.md`                           |
| Design   | `work/design/kanbanzai-2.5-infrastructure-hardening.md` §8  |

---

## 1. Implementation Approach

This feature adds a `dry_run` boolean parameter to the existing `doc import`
action. When `true`, the full inference pipeline runs — directory walking, type
inference, title extraction, owner inference — but no store writes occur. The
response describes what would happen rather than what did happen.

The work splits into three tasks:

**Task 1 — Service layer (dry-run path):** Extend the batch import service
to accept a dry-run flag. When set, the inference pipeline runs normally but
the store-write call is skipped. The function returns structured preview data
instead of (or in addition to) the normal import result.

**Task 2 — MCP tool layer (parameter exposure):** Add the `dry_run` parameter
to the `doc import` action handler in `doc_tool.go`. Route to the service
dry-run path when the flag is set; otherwise leave the existing code path
completely untouched.

**Task 3 — Tests:** Integration tests verifying that dry-run returns correct
preview data, creates no store records, and that the live import path is
unaffected.

Tasks 1 and 2 can be developed in parallel against the interface contract in
§2. Task 3 must follow both.

```
[Task 1: Service dry-run] ──┐
                             ├──→ [Task 3: Tests]
[Task 2: MCP parameter]  ───┘
```

---

## 2. Interface Contract

Task 2 calls into Task 1's service function. Both tasks must implement against
this contract:

**Function signature (to be added to the import service):**

```go
// ImportDryRun runs the full import inference pipeline over dir without
// writing any records to the store. It returns a preview of what would
// be registered and what would be skipped.
func (s *DocumentService) ImportDryRun(ctx context.Context, dir string) (*ImportDryRunResult, error)
```

**Result type:**

```go
type ImportDryRunResult struct {
    WouldImport []DryRunImportEntry `json:"would_import"`
    WouldSkip   []DryRunSkipEntry   `json:"would_skip"`
    Summary     DryRunSummary       `json:"summary"`
}

type DryRunImportEntry struct {
    Path  string `json:"path"`
    Type  string `json:"type"`
    Title string `json:"title"`
    Owner string `json:"owner"`
}

type DryRunSkipEntry struct {
    Path   string `json:"path"`
    Reason string `json:"reason"`
}

type DryRunSummary struct {
    WouldImport int `json:"would_import"`
    WouldSkip   int `json:"would_skip"`
}
```

The MCP tool handler serialises `*ImportDryRunResult` directly as the tool
response when `dry_run` is `true`.

---

## 3. Task Breakdown

| # | Task                              | Primary Files                           | Spec Refs     |
|---|-----------------------------------|-----------------------------------------|---------------|
| 1 | Service dry-run path              | `internal/service/batch_import.go`      | REQ-04–REQ-13 |
| 2 | MCP tool `dry_run` parameter      | `internal/mcp/doc_tool.go`              | REQ-01–REQ-03 |
| 3 | Tests                             | `internal/mcp/doc_tool_test.go`         | AC-21–AC-25   |

---

## 4. Task Details

### Task 1: Service Dry-Run Path

**Objective:** Implement `ImportDryRun` on the document service (or equivalent
import service type). The function must use the existing directory-walking, type
inference, title extraction, and owner inference logic — it must not duplicate
it. The only difference from the live import path is that the store-write step
is replaced by appending to the result lists.

**Specification references:** REQ-04, REQ-05, REQ-06, REQ-07, REQ-08, REQ-09,
REQ-10, REQ-11, REQ-12, REQ-13.

**Input context:**

- Read `work/spec/doc-import-dry-run.md` in full before starting.
- Read `internal/service/batch_import.go` to understand the existing import
  pipeline. `ImportDryRun` must share the inference steps, not copy them.
- Consult `internal/mcp/doc_tool.go` to understand how the service is currently
  called, so the new function fits the same calling pattern.
- The interface contract in §2 of this plan defines the exact return types.

**Output artefacts:**

- `internal/service/batch_import.go` — add `ImportDryRun` method.

**Behaviour details:**

1. Walk `dir` using the same logic as the live import walk.
2. For each `.md` file found:
   a. Check the store for an existing record at that path.
   b. If already registered → append to `WouldSkip` with
      `Reason: "already registered"`.
   c. If not registered → run type inference, title extraction, owner
      inference → append to `WouldImport`.
3. Populate `Summary.WouldImport` and `Summary.WouldSkip` from array lengths.
4. Return the result. Do NOT call any store-write method.

**Dependencies:** None — this task is independent of Task 2.

---

### Task 2: MCP Tool `dry_run` Parameter

**Objective:** Add the `dry_run` boolean parameter to the `doc import` action
in `doc_tool.go`. When `dry_run` is `true`, call `ImportDryRun` (from Task 1)
and return its result. When `dry_run` is `false` or absent, the existing code
path must execute without modification.

**Specification references:** REQ-01, REQ-02, REQ-03, REQ-25 (AC-25).

**Input context:**

- Read `work/spec/doc-import-dry-run.md` §4.1 (parameter requirements) before
  starting.
- Read `internal/mcp/doc_tool.go` to locate the import action handler.
- The interface contract in §2 of this plan defines the function to call and
  its return type.

**Output artefacts:**

- `internal/mcp/doc_tool.go` — add `dry_run` parameter parsing; add branch to
  call `ImportDryRun` and return its result when set.

**Behaviour details:**

1. Parse the `dry_run` boolean from the tool parameters. If absent or `false`,
   do nothing new — the existing import path runs unchanged.
2. If `dry_run` is `true`, call `ImportDryRun(ctx, path)` where `path` is the
   existing directory parameter.
3. Return the `*ImportDryRunResult` as the tool response, serialised to the
   standard tool response format.
4. Do not alter or wrap the live import response when `dry_run` is `false`.

**Dependencies:** Task 1 must be complete (the function must exist) before
Task 2 can compile. If developing in parallel, stub the signature first.

---

### Task 3: Tests

**Objective:** Write integration tests in `doc_tool_test.go` covering all five
acceptance criteria. Each test must set up a temporary directory with fixture
markdown files, pre-populate the store as needed, call the MCP tool, and assert
on the response structure.

**Specification references:** AC-21, AC-22, AC-23, AC-24, AC-25.

**Input context:**

- Read `work/spec/doc-import-dry-run.md` §5 (acceptance criteria) before
  starting.
- Read `internal/mcp/doc_tool_test.go` for existing test patterns and helper
  setup.
- Use `t.TempDir()` for all fixture directories — never write to the working
  directory.

**Output artefacts:**

- `internal/mcp/doc_tool_test.go` — new test cases.

**Test cases to write:**

| Test | Scenario | Assertion |
|------|----------|-----------|
| `TestDocImport_DryRun_ReturnsWouldImport` | Two unregistered `.md` files in temp dir | `would_import` contains both with inferred type, title, owner (AC-21, AC-23) |
| `TestDocImport_DryRun_NoStoreRecordsCreated` | Dry run followed by store query | Store contains zero new records (AC-22) |
| `TestDocImport_DryRun_AlreadyRegisteredSkipped` | One registered + one unregistered file | Registered file in `would_skip` with reason `"already registered"` (AC-24) |
| `TestDocImport_DryRunFalse_LiveBehaviourUnchanged` | `dry_run: false` on same fixture | Records are created; response is the live import format (AC-25) |
| `TestDocImport_NoDryRun_LiveBehaviourUnchanged` | `dry_run` absent | Records are created; response is the live import format (AC-25) |

**Dependencies:** Tasks 1 and 2 must both be complete before tests can run.

---

## 5. Scope Boundaries (carried from spec)

In scope:
- `dry_run` parameter on `doc import` only.
- Sharing existing inference logic — no duplication.
- All five acceptance criteria.

Out of scope:
- Dry-run modes on `doc register`, `doc approve`, `doc audit`, or any other
  action.
- Changes to type-inference or title-extraction logic.
- Persisting or caching dry-run results.
- Any new `doc import` parameters beyond `dry_run`.

---

## 6. Requirement Traceability

| Requirement | Task |
|-------------|------|
| REQ-01 (accept `dry_run`) | Task 2 |
| REQ-02 (false/omitted → unchanged) | Task 2, Task 3 |
| REQ-03 (defaults to false) | Task 2 |
| REQ-04 (run full pipeline) | Task 1 |
| REQ-05 (no store writes) | Task 1, Task 3 |
| REQ-06 (no file writes) | Task 1 |
| REQ-07 (same logic as live) | Task 1 |
| REQ-08 (`would_import` fields) | Task 1, Task 3 |
| REQ-09 (`would_skip` fields) | Task 1, Task 3 |
| REQ-10 (`summary` fields) | Task 1 |
| REQ-11 (counts match array lengths) | Task 1 |
| REQ-12 (already registered → `would_skip`) | Task 1, Task 3 |
| REQ-13 (consistency with live import) | Task 1 |
| AC-21–AC-25 | Task 3 |