# P37-F3: kbz move Command Specification

| Field   | Value                                         |
|---------|-----------------------------------------------|
| Date    | 2026-04-27T12:34:44Z                          |
| Status | approved |
| Author  | sambeau                                       |
| Feature | FEAT-01KQ7JDT11MH6 (kbz move command)         |
| Plan    | P37-file-names-and-actions                    |
| Design  | work/design/p37-file-names-and-actions.md §6.1, §6.3 |

---

## Overview

`kbz move` is a CLI command that safely relocates work documents and
re-parents features while keeping Git history, document records, and entity
metadata consistent. It operates in two modes: Mode 1 (single file move) and
Mode 2 (feature re-parent).

## Scope

This specification defines the requirements, constraints, and acceptance
criteria for the `kbz move` command (FEAT-01KQ7JDT11MH6) in Plan
P37-file-names-and-actions. It covers both Mode 1 (file move) and Mode 2
(feature re-parent). Mode 2 has a hard dependency on F1
(FEAT-01KQ7JDSVMP4E).

---

## Problem Statement

Work files in a Kanbanzai repository are tracked by document records stored in
`.kbz/state/documents/`. When a file is moved or renamed using plain `git mv`,
its document record is not updated, leaving an orphaned record pointing to a
non-existent path. Conversely, moving a file with `os.Rename` updates the path
but loses Git history and still leaves the record stale.

There is currently no safe way to relocate a work document — or to re-parent a
feature from one plan to another — while keeping document records, entity
metadata, display IDs, Git history, and on-disk file structure all consistent.

The `kbz move` command solves this by wrapping `git mv` with the necessary
state-update logic. It operates in two modes:

- **Mode 1 — File move:** relocates a single work file into a target plan
  folder, renames it to the canonical filename template, and updates the
  document record's `path` and `owner`.
- **Mode 2 — Feature re-parent:** moves an entire feature (and all its
  documents) from one plan to another, reallocating the feature's display ID
  in the target plan.

---

## Functional Requirements

### Mode 1: File Move (`kbz move <file-path> <plan-id>`)

**REQ-001** — Argument disambiguation (Mode 1): When the first argument
contains `/` or ends with `.md` or `.txt`, and the second argument matches
`P{n}` or `P{n}-{slug}`, the command must select Mode 1 (file move).

**REQ-002** — Restrict to `work/`: The command must reject any source path
that does not begin with `work/`, exiting with a non-zero status and a
descriptive error message.

**REQ-003** — Validate source file: The command must verify that the source
file exists on disk. If it does not, the command must exit with a non-zero
status and a descriptive error message before making any changes.

**REQ-004** — Validate target plan: The command must verify that the target
plan exists in `.kbz/state/plans/`. If it does not, the command must exit with
a non-zero status and a descriptive error message before making any changes.

**REQ-005** — Determine document type: The command must determine the document
type for the moved file. If the source file has a document record, the type is
read from that record. If no record exists, the type is inferred from the
source filename (e.g. a filename containing `-design-` implies type `design`).

**REQ-006** — Construct canonical target path: The command must construct the
target path as `work/P{n}-{slug}/{PlanID}-{type}-{filestem}.{ext}`, where:
- `P{n}-{slug}` is the target plan's folder (plan ID plus its slug);
- `{PlanID}` is the target plan ID (e.g. `P24`);
- `{type}` is the document type determined under REQ-005;
- `{filestem}` is the slug component preserved from the source filename stem
  (e.g. `foo` from `P10-design-foo.md`), or, if no slug component is
  identifiable, the document title slug from the record, or, if no record
  exists, the full source filename stem;
- `{ext}` is the original file extension.

**REQ-007** — Reject existing target: If the constructed target path already
exists on disk, the command must exit with a non-zero status and a descriptive
error message, making no changes to files or records.

**REQ-008** — Create target folder: If the target plan folder does not yet
exist on disk, the command must create it before running `git mv`.

**REQ-009** — Use `git mv`: The command must execute `git mv <source> <target>`
via a new `GitMove(repoRoot, src, dst string) error` function in `internal/git/`
that follows the `runGitCmd` pattern. `os.Rename` must not be used for file
moves anywhere in this command.

**REQ-010** — Update document record: After a successful `git mv`, the command
must update the source file's document record, setting `path` to the new target
path and `owner` to the target plan ID.

**REQ-011** — Unregistered file handling: If the source file has no document
record, the command must still move the file with `git mv`, print the warning
message `No document record found — file moved but no record updated`, and
exit with status zero.

**REQ-012** — Print summary: On success, the command must print a single
summary line to stdout: `Moved <source> → <target>`.

---

### Mode 2: Feature Re-parent (`kbz move <feature-display-id> <plan-id>`)

**REQ-013** — Argument disambiguation (Mode 2): When the first argument
matches the pattern `P{n}-F{m}` (digits only; no `/`; no `.`), and the second
argument matches `P{n}` or `P{n}-{slug}`, the command must select Mode 2
(feature re-parent).

**REQ-014** — Resolve display ID: The command must resolve the feature display
ID `P{n}-F{m}` to the canonical `FEAT-{TSID}` by reading the display-ID index.
If the display ID cannot be resolved, the command must exit with a non-zero
status and a descriptive error message.

**REQ-015** — Validate target differs from current parent: The command must
load the feature entity and verify that the target plan is different from the
feature's current parent. If they are the same, the command must exit with a
non-zero status and a descriptive error message before making any changes.

**REQ-016** — Confirm before executing: Unless `--force` is passed, the command
must print a summary of all planned changes (feature entity update and each
file move) and prompt the user for confirmation. If the user does not confirm,
the command must exit with status zero without making any changes. With
`--force`, the command must execute without prompting.

**REQ-017** — Allocate new display ID: The command must allocate the next
available display ID in the target plan by incrementing that plan's
`next_feature_seq` counter and forming `P{n}-F{next}`.

**REQ-018** — Update feature entity: The command must update the feature entity,
setting `parent` to the target plan ID and `display_id` to the newly allocated
display ID.

**REQ-019** — Move all feature documents: The command must find all document
records whose `owner` is the feature's canonical ID. For each such record, the
command must:
  (a) construct the new target path in the target plan folder using the canonical
      filename template (REQ-006 logic, applied with the new plan prefix);
  (b) execute `git mv <old-path> <new-path>` via `GitMove`;
  (c) update the document record's `path` to the new path and `owner` to the
      target plan ID.

**REQ-020** — Print re-parent summary: On success, the command must print to
stdout: one line indicating the feature move (old display ID → new display ID),
followed by one line per document moved (old path → new path).

---

## Non-Functional Requirements

**REQ-NF-001** — Implementation location: The command must be implemented as
`runMove(args []string, deps dependencies) error` in a new file
`cmd/kanbanzai/move_cmd.go`. The case `"move"` must be added to the
`switch args[0]` dispatcher in `cmd/kanbanzai/main.go`.

**REQ-NF-002** — Git helper function: All `git mv` invocations must go through
a new exported function `GitMove(repoRoot, src, dst string) error` in
`internal/git/`, implemented using the existing `runGitCmd` helper. No other
file in this command may call `exec.Command("git", ...)` directly.

**REQ-NF-003** — No modification of existing MCP tool: The existing
`doc(action: "move")` MCP tool action must not be changed. The `kbz move` CLI
command is a higher-level wrapper and does not require a new MCP equivalent.

**REQ-NF-004** — Mode 2 dependency: Mode 2 must not be implemented until the
F1 display-IDs feature is complete, as it depends on the display-ID index and
`next_feature_seq` counter introduced by F1. Mode 1 has no such dependency and
may be implemented independently.

**REQ-NF-005** — Error output: All error messages must be written to stderr.
Summary and warning output must be written to stdout.

---

## Constraints

- **`git mv` is mandatory.** File moves must use `git mv` (via `GitMove`) to
  preserve Git history. `os.Rename` is prohibited for this purpose.
- **`work/` boundary is enforced.** The command must refuse to move files
  outside the `work/` directory. Paths such as `docs/`, `internal/`, or
  `refs/` are rejected.
- **No flag-based mode selection.** Mode 1 and Mode 2 are distinguished
  entirely by the argument pattern of the first argument. No `--mode` flag or
  similar is used.
- **Mode 2 requires F1.** The feature re-parent mode depends on plan-scoped
  display IDs (`P{n}-F{m}`) and the `next_feature_seq` mechanism introduced by
  the F1 feature. Mode 2 must not be built until F1 is merged.
- **No MCP equivalent required.** The existing `doc(action: "move")` MCP tool
  (document-record-ID-based) covers the MCP use case. No new MCP tool action
  is introduced by this feature.
- **Approved documents are not specially protected in Mode 1.** The design does
  not require `--force` for moving approved documents (only for deleting them
  via `kbz delete`). Mode 1 may move approved documents without extra
  confirmation.
- **No cross-reference updates.** Cross-references between documents use entity
  IDs, not file paths. `kbz move` does not scan or rewrite document contents.

---

## Acceptance Criteria

**AC-001 (REQ-001):** Given the arguments `work/design/foo.md` and `P24`, when
the argument parser runs, then Mode 1 is selected because the first argument
contains `/`.

**AC-002 (REQ-001):** Given the arguments `notes.md` and `P24`, when the
argument parser runs, then Mode 1 is selected because the first argument ends
with `.md`.

**AC-003 (REQ-002):** Given a source path of `docs/foo.md` (outside `work/`),
when `kbz move docs/foo.md P24` is invoked, then the command exits with a
non-zero status, stderr contains a message indicating the path is not within
`work/`, and no files or records are modified.

**AC-004 (REQ-003):** Given `work/design/nonexistent.md` does not exist on
disk, when `kbz move work/design/nonexistent.md P24` is invoked, then the
command exits with a non-zero status, stderr contains a "file not found"
message, and no files or records are modified.

**AC-005 (REQ-004):** Given plan `P99` does not exist in `.kbz/state/plans/`,
when `kbz move work/design/foo.md P99` is invoked, then the command exits with
a non-zero status, stderr contains a "plan not found" message, and no files or
records are modified.

**AC-006 (REQ-005, REQ-006, REQ-009, REQ-010, REQ-012):** Given
`work/old/foo.md` is registered as a document of type `design` owned by plan
`P10`, and plan `P24` has slug `ac-pattern`, when `kbz move work/old/foo.md P24`
is invoked, then: (1) `git mv work/old/foo.md work/P24-ac-pattern/P24-design-foo.md`
is executed; (2) the document record's `path` is updated to
`work/P24-ac-pattern/P24-design-foo.md`; (3) the document record's `owner` is
updated to `P24`; (4) stdout contains
`Moved work/old/foo.md → work/P24-ac-pattern/P24-design-foo.md`; (5)
`git log --follow work/P24-ac-pattern/P24-design-foo.md` shows the full commit
history of the original file.

**AC-007 (REQ-008):** Given plan `P24` exists but its folder
`work/P24-ac-pattern/` does not yet exist on disk, when
`kbz move work/design/foo.md P24` is invoked, then `work/P24-ac-pattern/` is
created on disk before `git mv` is executed and the move succeeds.

**AC-008 (REQ-007):** Given the constructed target path
`work/P24-ac-pattern/P24-design-foo.md` already exists on disk, when
`kbz move work/old/foo.md P24` is invoked, then the command exits with a
non-zero status, stderr contains a "target already exists" message, and no
files or records are modified.

**AC-009 (REQ-011):** Given `work/scratch/notes.md` exists on disk but has no
associated document record, when `kbz move work/scratch/notes.md P24` is
invoked, then: (1) `git mv` moves the file to the canonical target path;
(2) stdout contains the warning
`No document record found — file moved but no record updated`; (3) the command
exits with status zero.

**AC-010 (REQ-009, REQ-NF-002):** Given any `kbz move` invocation that
performs a file move, when the implementation is inspected, then all file
moves are performed via `internal/git.GitMove` which calls `git mv`; `os.Rename`
is not called anywhere in `move_cmd.go` or in the code paths it invokes for
file relocation.

**AC-011 (REQ-013):** Given the arguments `P24-F3` (no `/`, no `.`) and `P25`,
when the argument parser runs, then Mode 2 is selected.

**AC-012 (REQ-014):** Given display ID `P24-F99` does not resolve to any
feature in the display-ID index, when `kbz move P24-F99 P25` is invoked, then
the command exits with a non-zero status, stderr contains a "feature not found"
message, and no files or records are modified.

**AC-013 (REQ-015):** Given feature `P24-F3` has current parent `P24`, when
`kbz move P24-F3 P24` is invoked (target equals current parent), then the
command exits with a non-zero status, stderr contains a message indicating the
feature is already in the target plan, and no files or records are modified.

**AC-014 (REQ-016):** Given a valid re-parent of `P24-F3` to `P25`, when
`kbz move P24-F3 P25` is invoked without `--force`, then: (1) stdout lists the
planned changes (feature entity update and each file to be moved) before any
changes are made; (2) the command pauses for user confirmation; (3) if the user
declines, no files, document records, or entity fields are modified and the
command exits with status zero.

**AC-015 (REQ-016):** Given a valid re-parent of `P24-F3` to `P25`, when
`kbz move P24-F3 P25 --force` is invoked, then the command executes all
changes without displaying a confirmation prompt.

**AC-016 (REQ-017, REQ-018):** Given feature `P24-F3` and plan `P25` whose
`next_feature_seq` is currently `4`, when `kbz move P24-F3 P25 --force`
completes, then: (1) the feature entity's `parent` field is `P25`; (2) the
feature entity's `display_id` is `P25-F4`; (3) plan `P25`'s `next_feature_seq`
is `5`.

**AC-017 (REQ-019):** Given feature `P24-F3` owns two document records at
`work/P24-old/P24-design-foo.md` and `work/P24-old/P24-spec-foo.md`, and plan
`P25` has slug `new-plan`, when `kbz move P24-F3 P25 --force` completes, then:
(1) `git mv` is executed for each document, producing
`work/P25-new-plan/P25-design-foo.md` and `work/P25-new-plan/P25-spec-foo.md`;
(2) both document records have `path` updated to the new paths and `owner`
updated to `P25`; (3) `git log --follow` shows unbroken history for each moved
file.

**AC-018 (REQ-020):** Given a successful re-parent of `P24-F3` to `P25-F4`
with two associated documents, when `kbz move P24-F3 P25 --force` completes,
then stdout contains: (1) a line of the form
`Moved feature P24-F3 → P25-F4`; (2) one line per document of the form
`Moved <old-path> → <new-path>`.

---

## Verification Plan

| Criterion | Method           | Description                                                                                                                           |
|-----------|------------------|---------------------------------------------------------------------------------------------------------------------------------------|
| AC-001    | Unit test        | `TestModeDetection_SlashInPath`: assert `parseArgs("work/design/foo.md", "P24")` selects Mode 1.                                      |
| AC-002    | Unit test        | `TestModeDetection_DotMdExtension`: assert `parseArgs("notes.md", "P24")` selects Mode 1.                                            |
| AC-003    | Unit test        | `TestRejectOutsideWorkDir`: assert `validateSourcePath("docs/foo.md")` returns an error containing "work/".                           |
| AC-004    | Integration test | `TestFileMoveSourceNotFound`: invoke command against a temporary git repo where the source file does not exist; assert non-zero exit and stderr "not found". |
| AC-005    | Integration test | `TestFileMoveTargetPlanNotFound`: invoke command with plan `P99` absent from `.kbz/state/plans/`; assert non-zero exit and stderr "plan not found". |
| AC-006    | Integration test | `TestFileMoveRegisteredDocument`: set up temporary git repo with registered document of type `design`; run move; assert `git log --follow` shows history, record fields updated, stdout matches expected summary line. |
| AC-007    | Integration test | `TestFileMoveCreatesPlanFolder`: invoke move targeting plan whose folder does not yet exist; assert folder is present after command, `git mv` succeeds. |
| AC-008    | Integration test | `TestFileMoveTargetAlreadyExists`: pre-create the target path; invoke move; assert non-zero exit, stderr "target already exists", source and target files unchanged. |
| AC-009    | Integration test | `TestFileMoveUnregisteredFile`: invoke move against a file with no document record; assert `git mv` executed, warning present in stdout, exit zero, no record written. |
| AC-010    | Code review      | Inspect `move_cmd.go` and all call sites: confirm `os.Rename` is absent; `internal/git.GitMove` is the sole file-move mechanism; `GitMove` calls `exec.Command("git", "mv", ...)` via `runGitCmd`. |
| AC-011    | Unit test        | `TestModeDetection_FeatureDisplayID`: assert `parseArgs("P24-F3", "P25")` selects Mode 2.                                            |
| AC-012    | Integration test | `TestReParentFeatureNotFound`: invoke with unresolvable display ID `P24-F99`; assert non-zero exit and stderr "feature not found".    |
| AC-013    | Integration test | `TestReParentSamePlan`: invoke `kbz move P24-F3 P24`; assert non-zero exit and stderr indicates feature already belongs to target plan. |
| AC-014    | Integration test | `TestReParentConfirmationPrompt`: invoke without `--force`; simulate declined confirmation; assert stdout lists planned changes before prompt, no entity or file changes after decline, exit zero. |
| AC-015    | Integration test | `TestReParentForceFlag`: invoke with `--force`; assert no prompt occurs and all changes are applied.                                  |
| AC-016    | Integration test | `TestReParentEntityUpdate`: after `--force` re-parent with `next_feature_seq=4`, load feature entity and plan record; assert `parent=P25`, `display_id=P25-F4`, plan `next_feature_seq=5`. |
| AC-017    | Integration test | `TestReParentDocumentMoves`: after `--force` re-parent with two documents, assert both files moved via `git mv`, both records have updated `path` and `owner`, `git log --follow` intact for each. |
| AC-018    | Integration test | `TestReParentOutputSummary`: capture stdout of `--force` re-parent; assert feature move line and one document move line per document are present.                      |
```

Now let me register the document: