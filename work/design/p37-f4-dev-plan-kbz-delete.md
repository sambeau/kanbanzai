# P37-F4: kbz delete Command — Implementation Plan

| Field  | Value                                                   |
|--------|---------------------------------------------------------|
| Date   | 2026-04-27T14:11:10Z                                    |
| Status | Draft                                                   |
| Author | orchestrator                                            |
| Spec   | work/design/p37-f4-spec-kbz-delete.md                  |

---

## Scope

This plan implements the requirements defined in
`work/design/p37-f4-spec-kbz-delete.md` (FEAT-01KQ7JDT341E8). It covers
the new `kbz delete <file-path> [--force]` CLI command: argument parsing,
path restriction, file existence validation, document record lookup, the
approved-document guard, interactive confirmation prompt, `git rm` staging,
record deletion, feature entity reference clearing, success/warning output,
CLI dispatch wiring, and the full unit and integration test suite.

It does not cover MCP surface changes (none are required), modifications to
`internal/git/commit.go` or `internal/service/documents.go` (both are
reused as-is), or monitoring or alerting.

---

## Task Breakdown

### Task 1: Implement `runDelete` in `cmd/kanbanzai/delete_cmd.go`

- **Description:** Create the new file `cmd/kanbanzai/delete_cmd.go` in
  `package main`. Implement `func runDelete(args []string, deps dependencies) error`
  containing the complete deletion logic:
  1. Parse `--force` flag and the single positional path argument; print
     usage and return nil on `--help` / `-h`.
  2. Reject any path that does not start with `work/` (REQ-002).
  3. Verify the file exists on disk via `os.Stat` (REQ-003).
  4. Scan `.kbz/state/documents/` using `storage.DocumentStore` to find a
     record whose `path` field matches the supplied path (REQ-004); treat
     multiple matches as an error.
  5. If a record is found with `status: approved` and `--force` is not set,
     print the guard error and return non-zero (REQ-005).
  6. Unless `--force`, print the confirmation prompt to `deps.stdout` and
     read a line from `deps.stdin`; any answer other than `y`/`Y` aborts
     and exits zero (REQ-006).
  7. Invoke `git rm <path>` as a subprocess using `exec.Command("git", "rm", path)`
     with the repo root as the working directory — the same pattern as the
     unexported `runGitCmd` helper in `internal/git/commit.go`, which is
     not directly callable from `package main` (REQ-008).
  8. On `git rm` failure, print the git error to `deps.stdout` and return
     non-zero without touching any record (REQ-NF-001).
  9. On success with a matching record, call
     `DocumentService.DeleteDocument(id, Force: true)` to clear the feature
     entity reference via the existing `entityHook` and remove the state
     record via `DocumentStore.Delete` (REQ-009, REQ-010). The `os.Remove`
     inside `DeleteDocument` will receive `ErrNotExist` (file already staged
     for removal by `git rm`) which it already ignores — no code change
     needed.
  10. On `DocumentStore.Delete` failure, print an error describing the
      partial state and return non-zero (REQ-NF-002).
  11. Print the appropriate success line (REQ-011) or no-record warning
      (REQ-012) and return nil.
  Also include a `deleteUsageText` constant for the help text.
- **Deliverable:** `cmd/kanbanzai/delete_cmd.go`
- **Depends on:** None
- **Effort:** Large
- **Spec requirements:** REQ-001–REQ-012, REQ-NF-001–REQ-NF-004

### Task 2: Wire dispatch in `cmd/kanbanzai/main.go`

- **Description:** Add `case "delete": return runDelete(args[1:], deps)` to
  the `switch` statement inside `run()` in `cmd/kanbanzai/main.go`, in the
  "Core workflow commands" section. No other changes to `main.go` are needed.
- **Deliverable:** Updated `cmd/kanbanzai/main.go` with the `delete` case
  present.
- **Depends on:** Task 1 (requires `runDelete` to exist before wiring)
- **Effort:** Small
- **Spec requirements:** REQ-013

### Task 3: Tests

- **Description:** Write the full test suite covering all 17 acceptance
  criteria:
  - **Unit tests** in `cmd/kanbanzai/delete_cmd_test.go`: mock `deps.stdin`,
    `deps.stdout`, a fake `DocumentStore` (or temp dir), and a fake git
    subprocess (swap `exec.Command` via a helper or temp `PATH` binary) to
    exercise:
    - AC-001: valid `work/` path with existing file proceeds to record lookup
    - AC-002: non-existent file returns error with path in message
    - AC-003: `internal/` path rejected with `work/` restriction message
    - AC-004: `.kbz/` path rejected before any file or record access
    - AC-005: approved record without `--force` returns error referencing `--force`
    - AC-007: confirmation prompt with `n` input exits zero, no deletions
    - AC-008: confirmation prompt with `y` input proceeds to git rm
    - AC-010: `git rm` is invoked; `os.Remove` is never called
    - AC-011: git rm failure leaves record untouched, exits non-zero
    - AC-012: successful git rm causes `DocumentStore.Delete` to be called
    - AC-013: feature entity `design`/`spec`/`dev_plan` field cleared after deletion
    - AC-014: success output matches required format
    - AC-017: all error conditions exit non-zero; success and `n`-abort exit zero
  - **Integration tests** (subprocess or temp-repo based) for prompt-driven
    flows and real git interaction:
    - AC-006: confirmation prompt text appears on stdout before any deletion
    - AC-009: `--force` against an approved document skips guard and prompt
    - AC-015: file with no record triggers warning output, exits zero
    - AC-016: build `kbz` binary; `kbz delete` (no args) returns usage, not
      "unknown command"
- **Deliverable:** `cmd/kanbanzai/delete_cmd_test.go`
- **Depends on:** Task 1, Task 2
- **Effort:** Large
- **Spec requirements:** All 17 AC (AC-001 – AC-017)

---

## Dependency Graph

```
Task 1  (no dependencies)
Task 2  → depends on Task 1
Task 3  → depends on Task 1, Task 2
```

Parallel groups: none (all tasks are on the critical path)

Critical path: Task 1 → Task 2 → Task 3

---

## Risk Assessment

### Risk: git rm failure atomicity

- **Probability:** Medium
- **Impact:** High — if the record is deleted before confirming `git rm` success,
  the document file remains on disk while its record is gone, producing an
  orphaned state that is difficult to recover.
- **Mitigation:** T1 must check the `git rm` exit code before making any call
  to `DocumentStore.Delete` or `DeleteDocument`. AC-011 is a required unit
  test that verifies the record is untouched on git failure.
- **Affected tasks:** Task 1, Task 3

### Risk: Approved document guard bypassable via flag order

- **Probability:** Low
- **Impact:** High — accidental deletion of an approved canonical document
  (specification, design) could disrupt feature traceability.
- **Mitigation:** T1 must evaluate the `--force` flag value after full flag
  parsing, not as a positional check. AC-005 and AC-009 together verify that
  the guard fires correctly without `--force` and is bypassed correctly with
  `--force`. T3 must cover both cases.
- **Affected tasks:** Task 1, Task 3

### Risk: Partial state on record-delete failure

- **Probability:** Low
- **Impact:** Medium — `git rm` stages the file deletion but the record is
  not removed, leaving a dangling record pointing to a file that no longer
  exists in the index.
- **Mitigation:** T1 must print a clear diagnostic message describing the
  partial state (file staged for removal, record not deleted) and exit
  non-zero so the operator can inspect and resolve manually. AC-017 verifies
  the non-zero exit.
- **Affected tasks:** Task 1, Task 3

---

## Verification Approach

| Acceptance Criterion | Verification Method | Producing Task |
|---|---|---|
| AC-001: valid path proceeds to record lookup | Unit test | Task 3 |
| AC-002: missing file returns error with path | Unit test | Task 3 |
| AC-003: non-work/ path rejected | Unit test | Task 3 |
| AC-004: .kbz/ path rejected before record/file access | Unit test | Task 3 |
| AC-005: approved guard fires without --force | Unit test | Task 3 |
| AC-006: confirmation prompt text written to stdout | Integration test | Task 3 |
| AC-007: n-abort exits zero, no deletions | Integration test | Task 3 |
| AC-008: y-input proceeds to git rm | Integration test | Task 3 |
| AC-009: --force skips guard and prompt | Integration test | Task 3 |
| AC-010: git rm used, os.Remove absent | Code inspection + unit test | Task 1, Task 3 |
| AC-011: git rm failure leaves record intact, exits non-zero | Unit test | Task 3 |
| AC-012: DocumentStore.Delete called after successful git rm | Unit test | Task 3 |
| AC-013: feature entity reference field cleared | Unit test | Task 3 |
| AC-014: success output matches required format | Unit test | Task 3 |
| AC-015: no-record warning output, exit zero | Integration test | Task 3 |
| AC-016: kbz delete dispatched, not unknown command | Build + smoke test | Task 2, Task 3 |
| AC-017: all error paths exit non-zero; success and n-abort exit zero | Unit + integration tests | Task 3 |
