# P37-F5: Work Tree Migration Specification

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-04-27T12:34:44Z           |
| Status | Draft                          |
| Author | sambeau                        |

## Problem Statement

This specification implements the migration strategy described in
`work/design/p37-file-names-and-actions.md` (P37-file-names-and-actions/design-p37-file-names-and-actions),
specifically §7 (Migration strategy) and the data from the migration audit.

The Kanbanzai project has 430 files spread across 18 type-first folders
(`work/design/`, `work/spec/`, `work/dev-plan/`, etc.), including 4 duplicate
folder pairs and one deprecated alias folder. This design decision D1 requires
reorganising these files into plan-first folders (`work/P{n}-{slug}/`).

This specification covers the tooling required to execute that migration safely
and the process for completing it. It explicitly depends on:

- **F2** (`FEAT-01KQ7JDSZARPC`) — document type system and filename enforcement,
  which defines the canonical filename template used to compute target paths
- **F3** (`FEAT-01KQ7JDT11MH6`) — `kbz move` command, which performs individual
  file moves with Git history preservation and document record updates
- **F4** (`FEAT-01KQ7JDT341E8`) — `kbz delete` command, which removes stale
  document records for files that no longer exist

**In scope:** the migration script, the triage report, bulk execution, and old
folder removal.

**Out of scope:** the filename template validation (F2), the `kbz move`
implementation (F3), the `kbz delete` implementation (F4), or any changes to
`.kbz/state/` entity files.

## Requirements

### Functional Requirements

- **REQ-001:** The system must provide a `kbz migrate` command (or equivalent
  script) that reads all document records from `.kbz/state/documents/` and,
  for each record, determines the target path under the canonical plan-first
  folder structure.

- **REQ-002:** For each document record, target path resolution must follow
  this chain:
  1. If `owner` is a Feature ID (`FEAT-{TSID}`): load the feature's parent plan
     from `.kbz/state/features/`, extract the plan ID and slug.
  2. If `owner` is a Plan ID (`P{n}-{slug}`): use it directly.
  3. If `owner` is `PROJECT` or absent: target folder is `work/_project/`.
  4. Construct target filename: `{PlanID}-{type}-{slug}.{ext}`, where slug is
     derived from the source filename by stripping any existing plan/feature
     prefix and type prefix, keeping the remainder as-is.

- **REQ-003:** The migration script must emit a dry-run report before executing
  any changes. The dry-run report lists, for every registered document:
  - Source path
  - Target path
  - Action (`MOVE`, `SKIP` if already in correct location, `MISSING` if source
    file does not exist on disk)

- **REQ-004:** Documents already in the correct location (source path equals
  computed target path) must be emitted as `SKIP` and not moved.

- **REQ-005:** Documents whose source file does not exist on disk (orphaned
  records) must be emitted as `MISSING` and their records must be deleted using
  `kbz delete --force <path>` (or equivalent direct record removal) rather than
  moved. The migration must not fail on missing files.

- **REQ-006:** The migration script must produce a triage report listing all
  work files that have no document record. For each unregistered file the report
  must include:
  - File path
  - Inferred type (from path segment or filename prefix, best-effort)
  - Recommended action: one of `REGISTER_AND_MOVE`, `ARCHIVE`, `DELETE`,
    or `REVIEW` (for files that cannot be automatically classified)

- **REQ-007:** The migration script must not execute any moves until the human
  has reviewed and confirmed the dry-run report and triage report. Confirmation
  is given by re-running the command with `--execute`.

- **REQ-008:** When run with `--execute`, the migration script must invoke
  `kbz move` for each `MOVE` entry in the dry-run report, accumulating all
  moves before committing. All moves must be staged in Git and committed in a
  single atomic commit with the message:
  `chore(work): migrate work tree to plan-first folder structure`

- **REQ-009:** If any individual `kbz move` invocation fails during `--execute`,
  the migration must stop immediately, report the failing file and error, and
  leave all successfully completed moves staged (not committed). The human can
  then fix the issue and re-run with `--execute --resume`, which skips files
  already staged.

- **REQ-010:** After all moves complete successfully, the migration script must
  identify folders under `work/` that are now empty (excluding `work/templates/`
  and `work/_project/`). It must list these folders in the output and, when run
  with `--execute --cleanup`, remove them with `git rm -r` and include their
  removal in a second commit:
  `chore(work): remove empty legacy type-first folders`

- **REQ-011:** The migration script must handle the following legacy folder
  duplicates by treating files in the minority folder as equivalent to the
  canonical folder for the purposes of type inference:
  - `work/specs/` → type `spec` (same as `work/spec/`)
  - `work/dev-plans/` → type `dev-plan` (same as `work/dev-plan/`)
  - `work/retros/` → type `retro` (same as `work/retro/`)
  - `work/evaluation/` and `work/eval/` → type `report`
  - `work/dev/` → type `dev-plan`

- **REQ-012:** The migration script must be idempotent: running it twice with
  `--execute` must produce `SKIP` for all files on the second run and make no
  changes.

- **REQ-013:** The `work/templates/` folder must be explicitly excluded from
  migration. Files in `work/templates/` must appear as `SKIP` in the dry-run
  report regardless of their content.

- **REQ-014:** The `docs/` directory must be explicitly excluded from migration.
  Files in `docs/` must not appear in the dry-run report or the triage report.

### Non-Functional Requirements

- **REQ-NF-001:** The dry-run report must complete in under 10 seconds for a
  repository with up to 1,000 document records and 1,000 work files.

- **REQ-NF-002:** The migration script must be implemented as a `kbz migrate`
  CLI subcommand following the existing hand-rolled dispatcher pattern
  (`cmd/kanbanzai/main.go` switch, `migrate_cmd.go` implementation file).

- **REQ-NF-003:** The migration script must produce machine-readable output
  (one line per file, tab-separated: `ACTION\tSOURCE\tTARGET`) when run with
  `--porcelain`, to allow piping and scripting.

## Constraints

- This feature must not be implemented or executed until F2, F3, and F4 are
  fully implemented and their specifications are approved.
- The migration must use `kbz move` (not `os.Rename` or direct file operations)
  to ensure Git history is preserved and document records are updated correctly.
- The migration must not change any files under `.kbz/state/` directly (other
  than document records, which `kbz move` updates as a side effect).
- The 28 currently unregistered files (as of 2026-04-27) require human triage
  before or after the bulk migration — the migration script must not block on
  their presence.
- The commit produced by `--execute` must be a single atomic commit containing
  all file moves. Partial commits are not acceptable.
- `work/templates/` contents must never be moved.
- `docs/` contents must never be moved.
- This spec does NOT cover changes to document type validation or filename
  enforcement — those are covered by F2.

## Acceptance Criteria

- **AC-001 (REQ-001, REQ-003):** Given a repository with registered documents
  in type-first folders, when `kbz migrate` is run without `--execute`, then
  a dry-run report is printed listing every registered document with its
  `ACTION`, source path, and computed target path, and no files are moved.

- **AC-002 (REQ-002):** Given a document record with owner `FEAT-01KQ7JDSVMP4E`
  whose parent plan is `P37-file-names-and-actions`, when the migration script
  computes the target path for a `design` document, then the target path is
  `work/P37-file-names-and-actions/P37-design-{slug}.md`.

- **AC-003 (REQ-002):** Given a document record with owner `P24-ac-pattern`,
  when the migration script computes the target path for a `spec` document,
  then the target path is `work/P24-ac-pattern/P24-spec-{slug}.md`.

- **AC-004 (REQ-002):** Given a document record with owner `PROJECT`, when the
  migration script computes the target path, then the target folder is
  `work/_project/` and the filename is `{type}-{slug}.md`.

- **AC-005 (REQ-004):** Given a document whose current path already equals its
  computed target path, when the dry-run report is generated, then that document
  appears as `SKIP` and `--execute` makes no change to it.

- **AC-006 (REQ-005):** Given a document record whose `path` points to a file
  that does not exist on disk, when the dry-run report is generated, then that
  document appears as `MISSING`, and when `--execute` is run, then the document
  record is deleted and no file operation is attempted.

- **AC-007 (REQ-006):** Given a repository with 28 work files that have no
  document record, when `kbz migrate` is run, then a triage report is printed
  listing each unregistered file with its inferred type and recommended action.

- **AC-008 (REQ-007):** Given a dry-run report with pending moves, when
  `kbz migrate` is run without `--execute`, then no files are moved and the
  command exits with code 0.

- **AC-009 (REQ-008):** Given a confirmed dry-run report, when `kbz migrate
  --execute` is run and all moves succeed, then all files are moved using
  `kbz move`, all document records are updated, and exactly one Git commit is
  created with the message
  `chore(work): migrate work tree to plan-first folder structure`.

- **AC-010 (REQ-009):** Given a migration in progress where the fourth `kbz
  move` fails, when `--execute` is run, then the migration halts after the
  third move, reports the error on the fourth file, leaves the first three
  moves staged, and exits with a non-zero code.

- **AC-011 (REQ-010):** Given a completed migration where legacy type-first
  folders are now empty, when `kbz migrate --execute --cleanup` is run, then
  the empty folders are removed via `git rm -r` and a second commit is created
  with the message `chore(work): remove empty legacy type-first folders`.

- **AC-012 (REQ-011):** Given a file at `work/dev-plans/feat-abc-cache.md`,
  when type is inferred for the triage or dry-run report, then the inferred
  type is `dev-plan`.

- **AC-013 (REQ-012):** Given a repository where `--execute` has already been
  run successfully, when `kbz migrate --execute` is run again, then all
  documents appear as `SKIP`, no files are moved, and no new commit is created.

- **AC-014 (REQ-013):** Given files in `work/templates/`, when the dry-run
  report is generated, then those files appear as `SKIP` and are never moved.

- **AC-015 (REQ-014):** Given files in `docs/`, when `kbz migrate` is run,
  then those files do not appear in the dry-run report or triage report.

- **AC-016 (REQ-NF-001):** Given a repository with 1,000 document records and
  1,000 work files, when `kbz migrate` is run in dry-run mode, then the report
  is produced in under 10 seconds.

- **AC-017 (REQ-NF-003):** Given any invocation of `kbz migrate` with
  `--porcelain`, when output is produced, then each line is tab-separated
  with exactly three fields: `ACTION`, `SOURCE`, `TARGET`, and the output
  contains no headers, colours, or decorative text.

## Verification Plan

| Criterion | Method      | Description |
|-----------|-------------|-------------|
| AC-001    | Test        | Integration test: create a temp repo with registered docs in `work/design/` etc.; run `kbz migrate`; assert no files moved, dry-run output present |
| AC-002    | Test        | Unit test: call target-path computation with a feature-owned design record whose feature has parent `P37-file-names-and-actions`; assert expected path |
| AC-003    | Test        | Unit test: call target-path computation with a plan-owned spec record for `P24-ac-pattern`; assert expected path |
| AC-004    | Test        | Unit test: call target-path computation with `owner: PROJECT`; assert `work/_project/{type}-{slug}.md` |
| AC-005    | Test        | Integration test: register a doc already at its canonical path; run dry-run; assert `SKIP` appears and `--execute` produces no commit |
| AC-006    | Test        | Integration test: create a document record pointing to a missing file; run dry-run; assert `MISSING`; run `--execute`; assert record deleted, no `git mv` attempted |
| AC-007    | Test        | Integration test: create unregistered files in various `work/` folders; run `kbz migrate`; assert each appears in triage report with inferred type and recommended action |
| AC-008    | Test        | Integration test: run without `--execute`; assert exit code 0, assert `git status` shows no staged changes |
| AC-009    | Test        | Integration test: set up 5 docs in old folders; run `--execute`; assert all 5 at target paths, all records updated, exactly one new commit with correct message |
| AC-010    | Test        | Integration test: make the fourth `kbz move` fail (e.g. by write-protecting the target); assert first three moves staged, error printed, non-zero exit |
| AC-011    | Test        | Integration test: run `--execute` to completion, then `--execute --cleanup`; assert empty legacy folders removed, second commit with correct message |
| AC-012    | Test        | Unit test: call type-inference function with `work/dev-plans/feat-abc-cache.md`; assert result is `dev-plan` |
| AC-013    | Test        | Integration test: run `--execute` twice; assert second run produces all `SKIP`, no new commits, exit code 0 |
| AC-014    | Test        | Integration test: place files in `work/templates/`; run dry-run; assert all appear as `SKIP` |
| AC-015    | Test        | Integration test: place files in `docs/`; run `kbz migrate`; assert none appear in output |
| AC-016    | Performance | Benchmark test: generate 1,000 mock records and 1,000 files; assert dry-run completes in under 10 seconds |
| AC-017    | Test        | Integration test: run `kbz migrate --porcelain`; parse stdout; assert each line has exactly three tab-separated fields, no headers |