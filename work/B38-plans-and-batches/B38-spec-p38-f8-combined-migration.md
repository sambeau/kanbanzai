# P38-F8: Combined State File and Work Tree Migration — Specification

| Field   | Value                                                                    |
|---------|--------------------------------------------------------------------------|
| Date    | 2026-04-28T16:12:29Z                                                     |
| Status | approved |
| Author  | spec-author                                                              |
| Feature | FEAT-01KQ7YQKT04M7                                                       |
| Design  | `work/design/meta-planning-plans-and-batches.md` — §7                    |

---

## Overview

This specification defines the combined migration from the legacy `P{n}` plan-centric
layout to the new `B{n}` batch-centric layout, coordinated with the P37 work tree
migration. It implements §7 (Migration strategy) of the P38 design document
`work/design/meta-planning-plans-and-batches.md`.

Per the P38 design's Open Questions #1, the P37 file migration has not yet run, so the
two migrations are combined into one atomic operation. "Combined" here means the
P{n}→B{n} state file rename and work directory rename are performed together —
eliminating the intermediate P{n} stage that Open Questions #1 identified as
avoidable. The P37-F5 type-first→plan-first work file migration runs as a separate
step after this migration. The combined migration:

1. Renames all `.kbz/state/plans/` state files from `P{n}-{slug}.yaml` to
   `B{n}-{slug}.yaml` in a new `.kbz/state/batches/` directory.
2. Rewrites all `P{n}` entity IDs, `parent` references, `design` references, and
   `display_id` fields to `B{n}` throughout `.kbz/state/`.
3. Renames `work/P{n}-{slug}/` folders to `work/B{n}-{slug}/` and renames files
   within those folders from `P{n}-*` to `B{n}-*`.
4. Updates document record paths and owner references from `P{n}` to `B{n}`.

---

## Scope

**In scope:**

- `.kbz/state/plans/` directory: create `.kbz/state/batches/`, copy all files with
  `P{n}` → `B{n}` ID rewrites, preserve original directory for backward compatibility
- Batch state files (`.kbz/state/batches/`): update `id` field from `P{n}-{slug}`
  to `B{n}-{slug}` and `design` field from `P{n}-{slug}/...` to `B{n}-{slug}/...`
- Feature state files (`.kbz/state/features/`): update `parent` field from
  `P{n}-{slug}` to `B{n}-{slug}` and `display_id` field from `P{n}-F{m}` to `B{n}-F{m}`
- Document record files (`.kbz/state/documents/`): update `owner`, `path`, and
  `plan_id` fields from `P{n}` to `B{n}`
- Work directory folders named `P{n}-{slug}`: rename to `B{n}-{slug}`
- Work files within renamed directories: rename filenames from `P{n}-*` to `B{n}-*`
- Map legacy folder patterns during triage: `specs/` → `spec`, `dev-plans/` → `dev-plan`,
  `retros/` → `retro`, `evaluation/`/`eval/` → `report`, `dev/` → `dev-plan`

**Explicitly excluded:**

- Type-first to plan-first work file migration (P37-F5 `kbz migrate` command)
- `kbz move` and `kbz delete` command implementations (P37-F3, P37-F4)
- Worktree directory migrations (`.worktrees/` — handled via git worktree prune)
- Config file schema migrations beyond prefix registries
- Document content rewriting (file contents are human-authored and use display IDs
  that may intentionally reference legacy `P{n}` IDs for historical clarity)

---

## Functional Requirements

### State File Migration

- **REQ-001:** The system MUST create a `.kbz/state/batches/` directory containing one
  YAML file per batch entity, with filenames in `B{n}-{slug}.yaml` format.

- **REQ-002:** Each batch file MUST contain the same fields as the source plan file,
  with the `id` field rewritten from `P{n}-{slug}` to `B{n}-{slug}`.

- **REQ-002a:** Each batch file's `design` field (if present) MUST be rewritten from
  `P{n}-{slug}/...` to `B{n}-{slug}/...`. The field is preserved unchanged if it does
  not contain a `P{n}` prefix.

- **REQ-003:** The migration MUST preserve the original `.kbz/state/plans/` directory
  until all cross-references are verified. Removal is a separate cleanup step.

- **REQ-004:** The migration MUST be idempotent. Running it twice MUST produce the same
  result without data loss or duplication.

### Cross-Reference Updates

- **REQ-005:** Every feature state file (`.kbz/state/features/*.yaml`) containing a
  `parent` field with a `P{n}-{slug}` value MUST be updated to `B{n}-{slug}`.

- **REQ-005a:** Every feature state file (`.kbz/state/features/*.yaml`) containing a
  `display_id` field with a `P{n}-F{m}` value MUST be updated to `B{n}-F{m}`.

- **REQ-006:** Every document record (`.kbz/state/documents/*.yaml`) containing
  `owner: P{n}-{slug}`, `plan_id: P{n}-{slug}`, or `path: work/P{n}-...` MUST be
  updated to use `B{n}`.

- **REQ-007:** The migration MUST NOT modify feature or document files that do not
  contain `P{n}` references. Unrelated files are skipped unchanged.

### Work Directory Migration

- **REQ-008:** Every `work/P{n}-{slug}/` directory MUST be renamed to
  `work/B{n}-{slug}/`.

- **REQ-008a:** Every file within a renamed work directory whose filename begins with
  `P{n}-` MUST be renamed to begin with `B{n}-`. Only the `P{n}`→`B{n}` prefix is
  changed; the remainder of the filename and the file contents are unchanged.

- **REQ-009:** The rename MUST NOT affect directories that are not plan-first folders
  (e.g., `work/_project/`, `work/templates/`, type-first folders like `work/design/`).

- **REQ-010:** Type-first work files (`work/design/*.md`, `work/spec/*.md`, etc.)
  are NOT moved by this migration. Their migration to `B{n}` plan-first folders is
  deferred to the `kbz migrate` command (P37-F5).

### Document Record Path Updates

- **REQ-011:** All document records that reference `path: work/P{n}-...` MUST be
  updated to `path: work/B{n}-...`.

- **REQ-012:** The update MUST handle both absolute paths and paths relative to the
  repository root.

---

## Non-Functional Requirements

- **REQ-NF-001:** The migration MUST complete as an atomic shell script or Go program
  that can be re-run safely.

- **REQ-NF-002:** The migration MUST NOT require external dependencies beyond POSIX
  shell utilities (sed, grep, mv, mkdir, rm).

- **REQ-NF-003:** The migration MUST log every file changed and every rename performed
  to stdout, enabling audit and rollback.

- **REQ-NF-004:** The migration MUST exit with code 0 on success and non-zero on any
  failure, with a descriptive error message identifying the failing file.

---

## Constraints

- The original `.kbz/state/plans/` directory is preserved after migration. Removal
  is a separate cleanup step to allow rollback if needed.
- **State fields vs document content:** Feature `display_id` and batch `design` fields
  in `.kbz/state/` YAML files are state metadata and are rewritten by this migration.
  Document file **contents** (`.md` files under `work/`) are NOT rewritten — they are
  human-authored and may intentionally reference legacy `P{n}` IDs for historical clarity.
  Work file **filenames** are metadata and are renamed per REQ-008a.
- The entity kind (`plan` vs `batch`) is determined by directory location
  (`.kbz/state/plans/` vs `.kbz/state/batches/`). No explicit `kind` field requires
  updating — moving files to `batches/` and rewriting `id` fields is sufficient.
- Git worktrees (`.worktrees/`) are NOT migrated by this feature. They are managed
  by git and pruned separately.
- The `prefixes` field in `.kbz/config.yaml` is NOT updated by this migration.
  Config changes to `batch_prefixes`/`plan_prefixes` are a separate step.
- This migration does NOT implement a reusable `kbz migrate` CLI command. That is
  P37-F5 scope and is deferred.

---

## Acceptance Criteria

- **AC-001 (REQ-001):** After migration, `.kbz/state/batches/B1-phase-4-orchestration.yaml`
  exists and contains `id: B1-phase-4-orchestration`.

- **AC-002 (REQ-002):** A batch file has all the same fields as the source plan file
  except the `id` field (and `design` field if present and containing `P{n}`). Field
  ordering is preserved.

- **AC-002a (REQ-002a):** Batch file `B12-agent-onboarding.yaml` (which had
  `design: P12-agent-onboarding/design-agent-onboarding`) now has
  `design: B12-agent-onboarding/design-agent-onboarding`. Batch file
  `B1-phase-4-orchestration.yaml` (which has no `design` field) is unaffected.

- **AC-003 (REQ-003):** The original `.kbz/state/plans/` directory still exists after
  migration with unchanged content.

- **AC-004 (REQ-004):** Running the migration script twice produces no errors and no
  additional changes on the second run.

- **AC-005 (REQ-005):** A feature that referenced `parent: P1-phase-4-orchestration`
  now references `parent: B1-phase-4-orchestration`. A feature that referenced a
  non-P{n} parent is unchanged.

- **AC-005a (REQ-005a):** Feature `FEAT-01KQ7YQKBDNAP` that had
  `display_id: P38-F2` now has `display_id: B38-F2`.

- **AC-005b (REQ-007):** A feature or document file that contains no `P{n}` references
  is byte-for-byte identical before and after migration.

- **AC-006 (REQ-006):** Document record `owner: P38-plans-and-batches` is now
  `owner: B38-plans-and-batches`. Document record `path: work/P38-plans-and-batches/...`
  is now `path: work/B38-plans-and-batches/...`.

- **AC-007 (REQ-008):** `work/P38-plans-and-batches/` has been renamed to
  `work/B38-plans-and-batches/` and its contents are unchanged.

- **AC-007a (REQ-008a):** File `work/B38-plans-and-batches/P38-spec-p38-f2-plan-entity-data-model-lifecycle.md`
  has been renamed to `work/B38-plans-and-batches/B38-spec-p38-f2-plan-entity-data-model-lifecycle.md`.
  Its contents are unchanged. Files within `work/B38-plans-and-batches/` that do not
  start with `P{n}-` are unaffected.

- **AC-008 (REQ-009):** `work/design/`, `work/_project/`, `work/templates/` are
  unaffected by the migration.

- **AC-009 (REQ-010):** Files in `work/design/` remain at their original paths.

- **AC-010 (REQ-011, REQ-012):** All document records referencing `work/P{n}-...` in
  their `path` field now reference `work/B{n}-...`. This holds for both relative
  paths and absolute paths rooted at the repository root.

---

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Test | Verify `B1-*.yaml` exists with correct content after migration script |
| AC-002 | Test | Diff a batch file against source plan file, verify only id (and design if P-prefixed) fields differ |
| AC-002a | Test | Verify `design` field in B12 batch file uses B-prefix; verify batch files without `design` field are unchanged |
| AC-003 | Test | Verify original plans/ directory unchanged after migration |
| AC-004 | Test | Run migration twice, assert idempotent |
| AC-005 | Test | Grep for `parent: P{n}` in features/, assert zero matches |
| AC-005a | Test | Grep for `display_id: P{n}` in features/, assert zero matches |
| AC-005b | Test | Select a feature file with no P-prefix values, verify byte-identical before and after |
| AC-006 | Test | Verify document record owner and path fields use B{n} |
| AC-007 | Test | Verify work/B{n} directories exist with unchanged contents |
| AC-007a | Test | Verify work/B{n}/B{n}-* files exist and contents match pre-migration work/P{n}/P{n}-* files |
| AC-008 | Test | Verify work/design/, work/_project/, work/templates/ still exist |
| AC-009 | Test | Verify files in work/design/ are untouched |
| AC-010 | Test | Grep document records for `work/P{n}` paths (both relative and absolute), assert zero matches |

---

## Dependencies and Assumptions

- **P38-F3 (Batch Entity Rename):** The batch entity must exist and the code
  must read from `.kbz/state/batches/` before this migration is meaningful.
- **P38 design §7:** The migration strategy defined in the design is the
  normative source for the requirements in this specification.
- **P37-F5 (Work Tree Migration):** This migration MUST run before P37-F5.
  After this migration completes, `work/B{n}-{slug}/` directories exist and
  P37-F5's `resolveMigrateTarget` function can target `B{n}` folders (instead
  of `P{n}`). The type-first→plan-first migration of individual work files
  (e.g., `work/design/foo.md` → `work/B{n}-slug/foo.md`) is P37-F5's
  responsibility and runs as a separate step after this migration completes.
- **Backward compatibility:** The code's backward-compatible `P{n}`→`B{n}`
  resolution (from P38-F3 REQ-009) means the system functions correctly even
  if some references are not migrated immediately.
