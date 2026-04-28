# B38-F9: Full Combined Migration — Specification

| Field   | Value                                                                    |
|---------|--------------------------------------------------------------------------|
| Date    | 2026-04-28T16:53:03Z                                                     |
| Status | approved |
| Author  | spec-author                                                              |
| Feature | (to be created)                                                          |
| Design  | `work/design/p37-file-names-and-actions.md` — §7                          |
|         | `work/design/meta-planning-plans-and-batches.md` — §7                    |
| Parent  | `work/B38-plans-and-batches/B38-spec-p38-f8-combined-migration.md`       |

---

## Problem Statement

This specification defines the complete migration of all work documents from the
legacy type-first folder layout to the plan-first B{n} folder layout, combined with
the P{n}→B{n} rename. It implements two design documents:

- **P37 §7 (Migration strategy):** Migrate all ~430 files from type-first folders
  (`work/design/`, `work/spec/`, etc.) to plan-first folders (`work/B{n}-slug/`)
  with canonical filenames (`B{n}-{type}-{slug}.md`).
- **P38 §7 (Migration strategy):** Rename P{n}→B{n} throughout the project and
  "reorganise folders simultaneously, avoiding a double-migration."

The existing P38-F8 specification (`B38-spec-p38-f8-combined-migration.md`) covered
the P{n}→B{n} rename for state files and the three plan-first directories that
already existed (B35, B37, B38). This specification supersedes it by adding the
type-first→plan-first file migration for the remaining ~400 files in
`work/design/`, `work/spec/`, `work/dev-plan/`, `work/dev-plans/`, `work/plan/`,
`work/research/`, `work/reports/`, `work/reviews/`, `work/retro/`, `work/retros/`,
`work/eval/`, `work/evaluation/`, `work/dev/`, and `work/specs/`.

**In scope:**
- Resolve every registered document's target B{n} plan directory
- Construct canonical filenames per the P37 filename template
- Move files from type-first directories to B{n} plan directories
- Update document record `path` fields
- Remove empty legacy directories after successful migration
- Idempotent execution (safe to re-run)
- Atomic commit with rollback via `git revert`

**Out of scope:**
- `kbz move` and `kbz delete` CLI commands (P37-F3, P37-F4)
- Unregistered file triage (separate human step before migration)
- Document content rewriting
- Worktree migrations
- Config file migrations
- Feature `display_id` allocation for pre-migration features

---

## Requirements

### Functional Requirements

#### Phase 1: Audit and Preparation

- **REQ-001:** The migration MUST produce a dry-run report listing every file move
  that will be performed, including source path, target path, and the document
  record that will be updated.

- **REQ-002:** The dry-run report MUST flag unresolvable files — documents whose
  owner cannot be resolved to a batch — as requiring human triage before the
  migration can proceed.

- **REQ-003:** The dry-run report MUST flag unregistered files — files in type-first
  directories with no document record — as requiring human triage.

- **REQ-004:** The dry-run report MUST be a machine-readable format (JSON or YAML)
  enabling automated verification before execution.

#### Phase 2: Target Resolution

- **REQ-005:** For each registered document, the target B{n} directory MUST be
  resolved via the document record's `owner` field:
  - If `owner` is a batch ID (`B{n}-slug`): target is `work/B{n}-slug/`
  - If `owner` is a feature ID (`FEAT-{TSID}`): load the feature, read its
    `parent` field, and target `work/B{n}-slug/` where B{n} is the parent batch
  - If `owner` is `PROJECT` or missing: target is `work/_project/`

- **REQ-006:** The canonical target filename MUST follow the P37 filename template:
  - Plan-owned documents: `B{n}-{type}-{slug}.md`
  - Feature-owned documents: `B{n}-F{seq}-{type}-{slug}.md`
  - Project-owned documents: `{type}-{slug}.md`
  - The `{type}` prefix is derived from the document record's `type` field:
    `design`, `spec` (for specification), `dev-plan`, `report`, `research`,
    `retro` (for retrospective)

- **REQ-007:** If the target directory does not exist, the migration MUST create it
  before moving files.

- **REQ-008:** If a file with the target name already exists at the target path,
  the migration MUST report a conflict and skip that file. No data is overwritten.

#### Phase 3: File Migration

- **REQ-009:** Each file MUST be moved from its source type-first directory to its
  resolved target B{n} directory using `git mv` to preserve Git history.

- **REQ-010:** After moving a file, the migration MUST update the corresponding
  document record's `path` field to reflect the new location.

- **REQ-011:** All file moves and document record updates in a single migration
  run MUST be committed as a single Git commit, making the migration atomic and
  reversible via `git revert`.

#### Phase 4: Cleanup

- **REQ-012:** After all files have been moved, the migration MUST identify empty
  legacy type-first directories and remove them.

- **REQ-013:** Directories that still contain files (due to conflicts or unregistered
  files) MUST NOT be removed.

#### Safety and Idempotency

- **REQ-014:** The migration MUST be idempotent. Running it twice produces no
  additional changes on the second run.

- **REQ-015:** The migration MUST NOT modify file contents. File bodies are
  byte-for-byte identical before and after migration.

- **REQ-016:** The migration MUST validate that the working tree is clean (no
  uncommitted changes) before executing. If the tree is dirty, the migration
  MUST abort with a descriptive error.

### Non-Functional Requirements

- **REQ-NF-001:** The migration MUST complete in under 60 seconds on the current
  project (~400 files).

- **REQ-NF-002:** The dry-run report MUST be generated in under 10 seconds.

- **REQ-NF-003:** The migration MUST produce a log of every action performed
  (file moved, record updated, directory removed) to stdout.

- **REQ-NF-004:** The migration MUST exit with code 0 on success and non-zero on
  any failure, with a descriptive error message identifying the failing item.

---

## Constraints

- **Git history preservation:** All file moves use `git mv` — no files are copied
  and deleted.
- **Atomicity:** All changes are committed as a single Git commit. Rollback is
  `git revert <commit>`.
- **Clean working tree required:** The migration aborts if `git status` is not clean.
- **No content changes:** File bodies are not modified. Only paths, filenames, and
  document record metadata change.
- **Human triage gating:** The migration cannot execute until all files in the
  dry-run report are resolvable (no unregistered or unresolvable files remain).
- **P38-F8 prerequisite:** The P{n}→B{n} state file and cross-reference migration
  (P38-F8) must be complete before this migration runs. Batch IDs must already
  use B{n} format.
- **Feature display IDs:** Feature `display_id` values in state files must already
  use B{n}-F{m} format (completed in P38-F8). The canonical filename template
  reads the `display_id` field to construct `B{n}-F{seq}-{type}-{slug}.md`.
- **Legacy folder mapping:** The following type-first directories are migrated:
  `design/`, `spec/`, `dev-plan/`, `dev-plans/`, `plan/`, `research/`, `reports/`,
  `reviews/`, `retro/`, `retros/`, `eval/`, `evaluation/`, `dev/`, `specs/`.
  The `_project/`, `templates/`, `bootstrap/`, and `test/` directories are NOT
  migrated.

---

## Acceptance Criteria

- **AC-001 (REQ-001):** Given a project with ~400 files in type-first directories,
  when the dry-run command executes, then a report is produced listing every
  proposed file move with source path, target path, and document record ID.

- **AC-002 (REQ-002):** Given a document whose owner cannot be resolved to a batch,
  when the dry-run executes, then that document appears in an "unresolvable"
  section of the report with the reason.

- **AC-003 (REQ-003):** Given unregistered files in type-first directories, when the
  dry-run executes, then those files appear in an "unregistered" section of the
  report.

- **AC-004 (REQ-005):** Given a feature-owned document with owner `FEAT-01KQ7YQKBDNAP`
  (whose parent is `B38-plans-and-batches`), when target resolution runs, then the
  target directory is `work/B38-plans-and-batches/`.

- **AC-005 (REQ-006):** Given a plan-owned design document for B12 with slug
  `agent-onboarding`, when the canonical filename is constructed, then the target
  filename is `B12-design-agent-onboarding.md`.

- **AC-006 (REQ-006):** Given a feature-owned specification for B38-F2 with
  display_id `B38-F2` and slug `plan-entity-data-model-lifecycle`, when the
  canonical filename is constructed, then the target filename is
  `B38-F2-spec-plan-entity-data-model-lifecycle.md`.

- **AC-007 (REQ-007):** Given a document targeting `work/B12-agent-onboarding/` and
  that directory does not exist, when the migration executes, then the directory is
  created and the file is moved into it.

- **AC-008 (REQ-008):** Given two documents resolving to the same target path, when
  the migration executes, then the first is moved successfully and the second is
  skipped with a conflict report.

- **AC-009 (REQ-009, REQ-010):** Given a file at `work/design/agent-onboarding.md`
  targeting `work/B12-agent-onboarding/B12-design-agent-onboarding.md`, when the
  migration executes, then the file is `git mv`'d and its document record's `path`
  field is updated to the new location.

- **AC-010 (REQ-011):** Given a successful migration, when `git log -1` is run, then
  a single commit contains all file moves and document record updates.

- **AC-011 (REQ-012):** Given all files have been moved from `work/design/`, when
  cleanup runs, then `work/design/` is removed.

- **AC-012 (REQ-013):** Given `work/design/` still contains an unregistered file,
  when cleanup runs, then `work/design/` is NOT removed.

- **AC-013 (REQ-014):** Given the migration has completed successfully, when it is
  run again, then no file moves or record updates are performed and the exit code
  is 0.

- **AC-014 (REQ-015):** Given a file moved by the migration, when its contents are
  diffed against the pre-migration version, then the diff is empty.

- **AC-015 (REQ-016):** Given uncommitted changes in the working tree, when the
  migration executes, then it aborts with exit code 1 and a message indicating the
  tree is dirty.

- **AC-016 (REQ-NF-001):** Given ~400 files to migrate, when the migration executes,
  then it completes in ≤ 60 seconds.

---

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Test | Run dry-run, verify report contains expected file count and correct source/target paths |
| AC-002 | Test | Create (or use existing) unresolvable document, verify it appears in unresolvable section |
| AC-003 | Test | Verify unregistered files in type-first dirs appear in unregistered section |
| AC-004 | Test | Verify feature-owned doc resolves to correct B{n} directory via parent field |
| AC-005 | Test | Verify plan-owned design doc gets correct B{n}-design-slug.md filename |
| AC-006 | Test | Verify feature-owned spec gets correct B{n}-F{seq}-spec-slug.md filename |
| AC-007 | Test | Run migration for a plan with no existing work dir, verify dir is created |
| AC-008 | Test | Create two docs with same target, verify second is skipped with conflict report |
| AC-009 | Test | Move a file, verify `git log --follow` shows history and doc record path is updated |
| AC-010 | Test | Verify single commit contains all changes; verify `git revert` restores original state |
| AC-011 | Test | Verify empty legacy directories are removed after migration |
| AC-012 | Test | Verify non-empty legacy directories are preserved |
| AC-013 | Test | Run migration twice, verify second run is a no-op |
| AC-014 | Test | Diff pre- and post-migration file contents, verify identical |
| AC-015 | Test | Create uncommitted change, run migration, verify abort |
| AC-016 | Test | Time the migration on the actual project, verify ≤ 60s |

---

## Dependencies and Assumptions

- **P38-F8 complete:** All state file P{n}→B{n} renames, cross-reference updates,
  and work directory renames from P38-F8 must be applied before this migration.
- **Feature `display_id` fields:** Must use B{n}-F{m} format (completed in P38-F8).
  The migration reads `display_id` to construct feature-owned filenames.
- **Document records current:** All ~400 documents must have valid document records
  in `.kbz/state/documents/` with correct `owner`, `type`, and `path` fields.
- **Unregistered files:** ~28 unregistered files require human triage before the
  migration can execute. The dry-run report identifies them.
- **Git availability:** The migration requires `git` for `git mv` and commit
  operations.
