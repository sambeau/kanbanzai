# P37-F4: kbz delete Command Specification

| Field   | Value                     |
|---------|---------------------------|
| Date    | 2026-04-27T12:34:44Z      |
| Status  | Draft                     |
| Author  | sambeau                   |
| Feature | FEAT-01KQ7JDT341E8        |
| Plan    | P37-file-names-and-actions |

---

## Problem Statement

Kanbanzai work documents (specifications, designs, dev plans, reports, etc.) have both a
physical file on disk and a document record in `.kbz/state/documents/`. Deleting a file with
`git rm` or `rm` leaves an orphaned document record; deleting only the record leaves an
untracked file. Neither the Git porcelain nor the existing MCP `doc(action: "delete")` tool
(which operates by document record ID) gives users a single, safe, path-based command that:

- accepts the file path a user can see in their working tree,
- stages the deletion in Git (`git rm`, not `os.Remove`),
- removes the corresponding document record atomically,
- clears any feature entity reference to that document, and
- guards against accidental deletion of approved (canonical) documents.

This specification defines the `kbz delete <file-path> [--force]` CLI command that addresses
all of the above.

---

## Requirements

### Functional Requirements

**REQ-001 — Path argument**
The command MUST accept exactly one positional argument: a file path relative to the repository
root (e.g. `work/P37-file-names/P37-F4-spec-kbz-delete.md`).

**REQ-002 — work/ path restriction**
The command MUST reject any path that does not begin with `work/`. Paths under `docs/`,
`internal/`, `.kbz/`, or any other directory MUST produce an error and exit without modifying
any file or record.

**REQ-003 — File existence validation**
The command MUST verify that the file exists on disk before proceeding. If the file does not
exist the command MUST print an error and exit non-zero without modifying any record.

**REQ-004 — Document record lookup**
The command MUST scan `.kbz/state/documents/` to find a document record whose `path` field
matches the supplied file path. If more than one record matches, the command MUST treat this
as an error and exit non-zero without deleting anything.

**REQ-005 — Approved-document guard**
If a matching document record is found AND its `status` is `approved`, the command MUST refuse
to proceed without the `--force` flag. Without `--force` it MUST print an error message that
names the file, states it is an approved document, and instructs the user to re-run with
`--force`.

**REQ-006 — Confirmation prompt**
Unless `--force` is supplied, the command MUST print a confirmation prompt of the form:

```
Delete work/P37-file-names/P37-F4-spec-kbz-delete.md and its document record? [y/N]
```

and wait for user input. Any answer other than `y` or `Y` MUST abort the operation and exit
zero without modifying any file or record.

**REQ-007 — --force bypasses guard and prompt**
When `--force` is supplied the command MUST skip both the approved-document guard (REQ-005)
and the confirmation prompt (REQ-006) and proceed directly to deletion.

**REQ-008 — git rm staging**
The command MUST delete the file by executing `git rm <file-path>` using the existing
`runGitCmd` subprocess helper in `internal/git/`. The command MUST NOT use `os.Remove` to
delete the file.

**REQ-009 — Document record removal**
After a successful `git rm`, if a matching document record was found, the command MUST remove
that record from `.kbz/state/documents/` via `DocumentStore.Delete(id)`.

**REQ-010 — Feature entity reference clearing**
After removing the document record, if the record's `owner` field references a feature entity,
the command MUST clear the corresponding reference field (design, spec, or dev_plan) on that
feature entity. This MUST be performed via the existing `entityHook` logic already present in
`DeleteDocument` in `internal/service/documents.go`.

**REQ-011 — Success output (record found)**
On successful deletion when a document record was found, the command MUST print a single line
of the form:

```
Deleted work/P37-file-names/P37-F4-spec-kbz-delete.md (document record <owner-id>/design-p37-f4-spec removed)
```

**REQ-012 — Success output (no record found)**
When no document record is found for the supplied path, the command MUST still run `git rm`,
then print a warning of the form:

```
No document record found — file deleted but no record updated
```

The command MUST exit zero in this case.

**REQ-013 — CLI wiring**
The command MUST be reachable as `kbz delete` by adding `case "delete": return runDelete(args[1:], deps)` to the `switch` dispatcher in `cmd/kanbanzai/main.go`. The implementation MUST live in `cmd/kanbanzai/delete_cmd.go` as `func runDelete(args []string, deps dependencies) error`.

**REQ-014 — No MCP changes**
The existing MCP `doc(action: "delete")` tool MUST NOT be modified. The new CLI command is a
separate, higher-level, path-based wrapper.

### Non-Functional Requirements

**REQ-NF-001 — Atomicity on git rm failure**
If `git rm` fails (e.g. file is not tracked, merge conflict, etc.) the command MUST abort and
MUST NOT remove the document record or clear any entity reference. It MUST print the `git rm`
error output and exit non-zero.

**REQ-NF-002 — Atomicity on record-delete failure**
If `git rm` succeeds but `DocumentStore.Delete` fails, the command MUST print an error
describing the partial state and exit non-zero. It MUST NOT silently swallow the error.

**REQ-NF-003 — No external dependencies**
The implementation MUST use only the Go standard library and packages already present in the
`kanbanzai` module. No new third-party dependencies are permitted.

**REQ-NF-004 — Exit codes**
The command MUST exit zero on success and non-zero on any error condition.

---

## Constraints

- **git rm only.** The file deletion MUST be staged through Git. `os.Remove` is explicitly
  prohibited because it bypasses Git's index and leaves the deletion unstaged.
- **work/ paths only.** The command operates exclusively on files under the `work/` directory.
  This prevents accidental deletion of source code, state files, or other infrastructure.
- **Reuse existing helpers.** The implementation MUST reuse `runGitCmd` (Git subprocess),
  `DocumentStore` (record store), and `DeleteDocument`'s `entityHook` (feature-ref clearing).
  It MUST NOT duplicate logic already present in these components.
- **No MCP surface.** This feature does not add a new MCP tool. The CLI command is the only
  new public interface.
- **--force semantics.** `--force` is a single flag that bypasses both the approved guard and
  the confirmation prompt simultaneously. There is no per-guard bypass flag.

---

## Acceptance Criteria

**AC-001 (REQ-001, REQ-003):** Given the user runs `kbz delete work/P37-foo/P37-F1-design.md`
and the file exists, when the command executes, then it proceeds to the document record lookup
step without error.

**AC-002 (REQ-001, REQ-003):** Given the user runs `kbz delete work/P37-foo/missing.md` and
that file does not exist on disk, when the command executes, then it prints an error message
referencing the missing file and exits non-zero without modifying any record.

**AC-003 (REQ-002):** Given the user runs `kbz delete internal/service/documents.go`, when the
command executes, then it prints an error stating the path is outside `work/` and exits
non-zero without modifying any file or record.

**AC-004 (REQ-002):** Given the user runs `kbz delete .kbz/state/documents/some-record.yaml`,
when the command executes, then it prints an error stating the path is outside `work/` and
exits non-zero.

**AC-005 (REQ-005):** Given a file whose document record has `status: approved` and `--force`
is NOT passed, when the user runs `kbz delete work/P37-foo/P37-F1-spec.md`, then the command
prints an error naming the file, states it is an approved document, instructs the user to
re-run with `--force`, and exits non-zero without deleting the file or record.

**AC-006 (REQ-006):** Given a file with a non-approved document record and `--force` is NOT
passed, when the command executes, then it prints the prompt
`Delete work/P37-foo/P37-F1-design.md and its document record? [y/N]` and waits for input.

**AC-007 (REQ-006):** Given the confirmation prompt is displayed and the user enters `n`,
when the command processes the input, then it prints an abort message and exits zero without
deleting the file or record.

**AC-008 (REQ-006):** Given the confirmation prompt is displayed and the user enters `y`,
when the command processes the input, then it proceeds to `git rm` and subsequent steps.

**AC-009 (REQ-007):** Given a file with an approved document record and `--force` IS passed,
when the user runs `kbz delete --force work/P37-foo/P37-F1-spec.md`, then the command skips
the approved guard and the confirmation prompt and proceeds directly to deletion.

**AC-010 (REQ-008):** Given a tracked file in the repository and the user confirms deletion,
when the command executes, then it invokes `git rm work/P37-foo/P37-F1-design.md` and the file
is removed from Git's index (staged for removal) without using `os.Remove`.

**AC-011 (REQ-NF-001):** Given a file that is not tracked by Git (no `git rm` target), when
the command tries to delete it, then `git rm` fails, the command prints the git error, exits
non-zero, and the document record is NOT removed.

**AC-012 (REQ-009):** Given a successful `git rm` and a matching document record, when the
deletion completes, then the record file is removed from `.kbz/state/documents/`.

**AC-013 (REQ-010):** Given a document record whose owner is a feature entity with a `design`
reference pointing to the deleted document, when the deletion completes, then the feature
entity's `design` field is cleared.

**AC-014 (REQ-011):** Given a successful deletion with a matching document record, when the
command exits, then it prints exactly one line matching the pattern:
`Deleted <path> (document record <owner-or-id> removed)`.

**AC-015 (REQ-012):** Given a file that exists on disk but has no document record, when the
user runs `kbz delete work/P37-foo/scratch.md` and confirms, then `git rm` is executed
successfully, the command prints
`No document record found — file deleted but no record updated`, and exits zero.

**AC-016 (REQ-013):** Given a built `kbz` binary, when the user runs `kbz delete`, then the
command is dispatched to `runDelete` (not a "command not found" error).

**AC-017 (REQ-NF-004):** Given any error condition (bad path, missing file, git failure,
record-delete failure, user abort on approved doc without --force), when the command exits,
then the exit code is non-zero. Given a successful deletion or a user-aborted confirmation
prompt (`n` answer), then the exit code is zero.

---

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001: valid path proceeds | Unit test | Call `runDelete` with a path to a file that exists under `work/`; assert no early-exit error before the record-lookup step. |
| AC-002: missing file error | Unit test | Call `runDelete` with a non-existent `work/` path; assert non-zero error return and error message containing the path. |
| AC-003: non-work/ path rejected | Unit test | Call `runDelete` with `internal/service/documents.go`; assert error message mentions `work/` restriction and non-zero exit. |
| AC-004: .kbz/ path rejected | Unit test | Call `runDelete` with `.kbz/state/documents/foo.yaml`; assert rejected before any record or file access. |
| AC-005: approved guard without --force | Unit test | Inject a mock document store returning an approved record; assert command errors with message referencing `--force` and does not invoke git. |
| AC-006: confirmation prompt shown | Integration test | Run `kbz delete` against a non-approved tracked file with no `--force`; assert prompt text is written to stdout before any deletion. |
| AC-007: prompt abort on n | Integration test | Pipe `n` to stdin; assert file still exists, record still present, exit code zero. |
| AC-008: prompt proceeds on y | Integration test | Pipe `y` to stdin; assert `git rm` is called. |
| AC-009: --force skips guard and prompt | Integration test | Run with `--force` against an approved document; assert deletion completes without prompt and without approved-guard error. |
| AC-010: git rm used not os.Remove | Code review + unit test | Inspect `delete_cmd.go` for absence of `os.Remove`; mock `runGitCmd` in unit test and assert it is called with `git rm <path>`. |
| AC-011: git rm failure aborts record delete | Unit test | Mock `runGitCmd` to return error; assert `DocumentStore.Delete` is never called and command exits non-zero. |
| AC-012: record removed after git rm | Unit test | Mock successful `runGitCmd`; assert `DocumentStore.Delete(id)` is called with the correct record ID. |
| AC-013: feature entity ref cleared | Unit test | Use a mock feature store; assert the feature's design/spec/dev_plan field is cleared after deletion. |
| AC-014: success output format | Unit test | Capture stdout; assert it matches `Deleted <path> (document record ... removed)` exactly once. |
| AC-015: no-record warning output | Integration test | Delete a tracked file with no document record; assert warning text and exit zero. |
| AC-016: CLI dispatch wired | Build + smoke test | Build `kbz`; run `kbz delete --help` or `kbz delete` with no args; assert not "unknown command". |
| AC-017: exit codes | Unit + integration tests | Enumerate all error paths and assert non-zero; assert zero for success and for user `n` abort. |