# P37-F4: kbz delete Command — Implementation Plan

| Field  | Value                                                   |
|--------|---------------------------------------------------------|
| Date   | 2026-04-27T14:11:10Z                                    |
| Status | approved |
| Author | orchestrator                                            |
| Spec   | work/design/p37-f4-spec-kbz-delete.md                  |

---

## Overview

This plan implements the `kbz delete <file-path> [--force]` CLI command defined in
`work/design/p37-f4-spec-kbz-delete.md` (FEAT-01KQ7JDT341E8). It covers three tasks:
the core `runDelete` implementation in a new `cmd/kanbanzai/delete_cmd.go`, dispatch
wiring in `cmd/kanbanzai/main.go`, and the full test suite.

The command accepts a single `work/`-relative path, looks up the corresponding document
record, optionally guards against deleting approved documents, prompts for confirmation,
stages the deletion via `git rm`, removes the document record, and clears any feature
entity reference. It reuses `internal/service/documents.go` (`DeleteDocument`) and
`internal/storage/document_store.go` (`DocumentStore.Delete`) without modifying them.

Out of scope: MCP tool changes (REQ-014 explicitly prohibits them), modifications to
`internal/git/commit.go` or `internal/service/documents.go`, and any monitoring or
alerting.

---

## Task Breakdown

### T1 — Implement `runDelete` in `cmd/kanbanzai/delete_cmd.go`

**Objective**

Create `cmd/kanbanzai/delete_cmd.go` (package main) with
`func runDelete(args []string, deps dependencies) error` implementing the complete
deletion flow:

1. Parse `--force` flag and the single positional path argument. Print usage and
   return nil on `--help`/`-h` or no arguments.
2. Reject paths that do not begin with `work/`; print a clear error and return
   non-zero (REQ-002).
3. Confirm the file exists via `os.Stat`; print a "file not found" error and return
   non-zero if absent (REQ-003).
4. Scan `.kbz/state/documents/` via `storage.NewDocumentStore` to find a record whose
   `path` field matches the supplied path. Treat zero matches as a no-record case; treat
   two or more matches as an error (REQ-004).
5. If a record with `status: approved` is found and `--force` is not set, print the guard
   error (naming the file and instructing the user to re-run with `--force`) and return
   non-zero (REQ-005).
6. Unless `--force`, write the confirmation prompt to `deps.stdout` and read one line from
   `deps.stdin`. Any response other than `y`/`Y` prints an abort message and returns nil
   (REQ-006).
7. Invoke `exec.Command("git", "rm", path)` with `cmd.Dir` set to the repository root,
   capturing stdout and stderr — matching the pattern of the package-private `runGitCmd`
   in `internal/git/commit.go`, which is not directly callable from `package main`
   (REQ-008).
8. On `git rm` failure, print the git error output and return non-zero without touching
   any record (REQ-NF-001).
9. On success, if a matching record exists, call
   `service.DocumentService.DeleteDocument(id, Force: true)`. This call handles the entity
   reference clearing via the existing `entityHook` (REQ-010) and removes the record via
   `DocumentStore.Delete` (REQ-009). Because `git rm` has already staged the file for
   removal, the `os.Remove` inside `DeleteDocument` receives `ErrNotExist`, which it
   already ignores — no modification to `documents.go` is needed.
10. If `DeleteDocument` fails, print a partial-state error and return non-zero (REQ-NF-002).
11. On success with a record, print `Deleted <path> (document record <id> removed)` (REQ-011).
    On success without a record, print `No document record found — file deleted but no
    record updated` (REQ-012). Return nil in both cases.

Also include a `deleteUsageText` constant for the help output.

**Specification references:** REQ-001–REQ-012, REQ-NF-001–REQ-NF-004

**Input context**

- `cmd/kanbanzai/cleanup_cmd.go` — existing command pattern: flag parsing via
  `parseFlags`, `deps.stdout`/`deps.stdin` usage, error returns.
- `cmd/kanbanzai/main.go` — `dependencies` struct (stdout, stdin fields).
- `internal/service/documents.go` — `DocumentService`, `DeleteDocumentInput`,
  `DeleteDocument`.
- `internal/storage/document_store.go` — `DocumentStore`, `NewDocumentStore`,
  `DocumentStore.Delete`.
- `internal/git/commit.go` — `runGitCmd` pattern (reference only, not callable).

**Output artifacts**

- New file `cmd/kanbanzai/delete_cmd.go` with `runDelete` and `deleteUsageText`.

**Dependencies:** none

**Effort:** 5 points

---

### T2 — Wire dispatch in `cmd/kanbanzai/main.go`

**Objective**

Add `case "delete": return runDelete(args[1:], deps)` to the `switch` statement inside
`run()` in `cmd/kanbanzai/main.go`. Place it in the "Core workflow commands" section,
immediately after the `doc` case. No other changes to `main.go` are required.

**Specification references:** REQ-013

**Input context**

- `cmd/kanbanzai/main.go` — `run()` switch block (lines 88–161).
- T1 deliverable: `runDelete` function signature.

**Output artifacts**

- `cmd/kanbanzai/main.go` updated with the `delete` case.

**Dependencies:** T1

**Effort:** 1 point

---

### T3 — Tests

**Objective**

Write the complete test suite for `runDelete` covering all 17 acceptance criteria. Tests
live in `cmd/kanbanzai/delete_cmd_test.go` following patterns from existing `*_cmd_test.go`
files in the same package.

Unit tests (mock-based, no real git repo):

- AC-001: valid `work/` path with existing file proceeds to record lookup step without error.
- AC-002: non-existent file returns non-zero error containing the path.
- AC-003: `internal/` path rejected with `work/` restriction message, non-zero.
- AC-004: `.kbz/` path rejected before any record or file access, non-zero.
- AC-005: record with `status: approved`, no `--force` → error references `--force`, non-zero.
- AC-007: prompt with `n` input → abort message, nil return (exit zero).
- AC-008: prompt with `y` input → proceeds to git rm invocation.
- AC-010: `exec.Command("git", "rm", ...)` is called; `os.Remove` does not appear in `delete_cmd.go`.
- AC-011: git rm failure → `DocumentStore.Delete` never called, non-zero return.
- AC-012: successful git rm → `DocumentStore.Delete(id)` called with the correct ID.
- AC-013: document with `owner` pointing to a feature entity → entity `design`/`spec`/`dev_plan` field cleared.
- AC-014: success output matches `Deleted <path> (document record ... removed)` exactly.
- AC-017: all error paths return non-zero; success and `n`-abort return nil.

Integration tests (real temp git repo or subprocess):

- AC-006: confirmation prompt text appears on stdout before any deletion.
- AC-009: `--force` with an approved document skips guard and prompt, deletes successfully.
- AC-015: file with no document record → warning output, exit zero.
- AC-016: build `kbz` binary; `kbz delete` (no args) returns usage text, not "unknown command".

**Specification references:** AC-001–AC-017 (all)

**Input context**

- T1 and T2 deliverables.
- Existing `cmd/kanbanzai/*_cmd_test.go` files for test patterns.
- `refs/testing.md` for project test conventions.

**Output artifacts**

- `cmd/kanbanzai/delete_cmd_test.go`

**Dependencies:** T1, T2

**Effort:** 8 points

---

## Interface Contracts

### `runDelete` (T1 → T2)

```go
// runDelete implements the `kbz delete` command. It is called by the run()
// dispatcher in main.go with args[1:] (the arguments after "delete").
func runDelete(args []string, deps dependencies) error
```

T2 calls `runDelete` with `args[1:]` from `run()`. T1 must export this signature within
`package main` before T2 can wire the dispatch.

No other cross-task interface contracts exist; the `git rm` subprocess, `DocumentStore`,
and `DeleteDocument` are all standard library or existing internal packages consumed
directly by T1.

---

## Dependency Graph

```
T1  (no dependencies)
T2  → depends on T1
T3  → depends on T1, T2
```

Parallel groups: none — all tasks are on the critical path.

Critical path: T1 → T2 → T3

---

## Risk Assessment

**R1 — git rm failure atomicity (REQ-NF-001)**

If the document record is deleted before confirming `git rm` success, the file remains
on disk (or in a conflicted state) while its record is gone, producing an orphaned state.
Probability: medium (easy to get the order wrong). Impact: high (silent data corruption).
Mitigation: T1 must check `exec.Command.Run()` error before any call to `DeleteDocument`
or `DocumentStore.Delete`. AC-011 is a required unit test validating this ordering.

**R2 — Approved-document guard bypassed by flag-order ambiguity (REQ-005, REQ-007)**

If flag parsing is done positionally rather than via a proper flag scan, a user could
pass `--force` after the path and bypass the guard unexpectedly — or vice versa, the
guard could fire when `--force` was correctly supplied. Probability: low. Impact: high
(approved documents are canonical artefacts). Mitigation: T1 must parse all flags before
evaluating any guard logic. AC-005 (guard fires) and AC-009 (guard bypassed) together
verify correct behaviour; T3 must cover both.

**R3 — Partial state on record-delete failure (REQ-NF-002)**

`git rm` may succeed (file staged for removal in git index) while `DeleteDocument` fails,
leaving a dangling record. Probability: low. Impact: medium (recoverable but confusing).
Mitigation: T1 must print a diagnostic that names the partial state (`git rm` succeeded,
record not deleted) and return non-zero. This allows the operator to manually delete the
record. AC-017 verifies the non-zero exit for this path.

---

## Traceability Matrix

| Requirement     | AC(s)             | Task(s) |
|-----------------|-------------------|---------|
| REQ-001         | AC-001, AC-002    | T1, T3  |
| REQ-002         | AC-003, AC-004    | T1, T3  |
| REQ-003         | AC-001, AC-002    | T1, T3  |
| REQ-004         | AC-001            | T1, T3  |
| REQ-005         | AC-005            | T1, T3  |
| REQ-006         | AC-006, AC-007, AC-008 | T1, T3 |
| REQ-007         | AC-009            | T1, T3  |
| REQ-008         | AC-010            | T1, T3  |
| REQ-009         | AC-012            | T1, T3  |
| REQ-010         | AC-013            | T1, T3  |
| REQ-011         | AC-014            | T1, T3  |
| REQ-012         | AC-015            | T1, T3  |
| REQ-013         | AC-016            | T2, T3  |
| REQ-014         | (no AC — constraint) | (code review) |
| REQ-NF-001      | AC-011            | T1, T3  |
| REQ-NF-002      | AC-017            | T1, T3  |
| REQ-NF-003      | (no AC — constraint) | (code review) |
| REQ-NF-004      | AC-017            | T1, T3  |

Every functional requirement maps to at least one task. Every task maps to at least one
requirement. REQ-014 and REQ-NF-003 are verified by code review during the PR review
stage (no automated test needed).

### Verification Approach

All 17 acceptance criteria map to automated tests in T3. The unit tests use mock stdin/
stdout (`bytes.Buffer`) and a temp-directory-backed `DocumentStore`. Integration tests
for AC-006, AC-009, AC-015, and AC-016 use a real git repository in `t.TempDir()` or
build the `kbz` binary as a subprocess.

All tests must pass `go test -race ./...` without data races and `go vet ./...` without
warnings before the feature is considered ready for review.
