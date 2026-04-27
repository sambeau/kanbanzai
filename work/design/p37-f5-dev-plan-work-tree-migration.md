# P37-F5: Work Tree Migration — Implementation Plan

| Field  | Value                                                    |
|--------|----------------------------------------------------------|
| Date   | 2026-04-27T14:11:10Z                                     |
| Status | Draft                                                    |
| Author | orchestrator                                             |
| Spec   | work/design/p37-f5-spec-work-tree-migration.md           |

## Scope

This plan implements the requirements defined in
`work/design/p37-f5-spec-work-tree-migration.md` (FEAT-01KQ7JDT511BZ). It covers
the `kbz migrate` CLI subcommand: target-path resolution, dry-run report generation,
`--execute` bulk-move mode, `--execute --cleanup` empty-folder removal, `--porcelain`
machine-readable output, `--resume` support, and the full test suite verifying all 17
acceptance criteria.

This plan does **not** cover:
- Filename validation or type enforcement (F2, `FEAT-01KQ7JDSZARPC`)
- The `kbz move` implementation (F3, `FEAT-01KQ7JDT11MH6`)
- The `kbz delete` implementation (F4, `FEAT-01KQ7JDT341E8`)
- Any changes to `.kbz/state/` entity files other than document records

**Critical dependency:** This feature MUST NOT be implemented until F2, F3, and F4
are fully merged into `main`. T3 and T4 call into `runMove` and `runDelete`, which
do not exist until those features land.

## Task Breakdown

### Task 1: Implement target-path resolver

- **Description:** Implement the pure function (or set of functions) that, given a
  document record, resolves the canonical target path under the plan-first folder
  structure. Handles three owner cases: Feature ID (load feature → parent plan),
  Plan ID (use directly), and PROJECT/absent (→ `work/_project/`). Constructs the
  target filename as `{PlanID}-{type}-{slug}.{ext}` (or `{type}-{slug}.{ext}` for
  `_project`). Also implements legacy-folder type inference for triage (maps
  `work/specs/` → `spec`, `work/dev-plans/` → `dev-plan`, `work/retros/` → `retro`,
  `work/evaluation/`/`work/eval/` → `report`, `work/dev/` → `dev-plan`).
- **Deliverable:** Functions `resolveMigrateTarget` and `inferTypeFromPath` in
  `cmd/kanbanzai/migrate_cmd.go`; they take document records and entity service
  references, return target path string and error.
- **Depends on:** None (pure logic; entity/plan service interfaces already exist).
- **Effort:** Medium
- **Spec requirement:** REQ-002, REQ-011; AC-002, AC-003, AC-004, AC-012

### Task 2: Implement dry-run report generator

- **Description:** Implement the default (no-flag) behaviour of `kbz migrate`:
  read all document records via the document service; for each, call the resolver
  from T1 and classify as `MOVE`, `SKIP`, or `MISSING`. Apply exclusions:
  `work/templates/` entries always emit `SKIP`; `docs/` entries are omitted
  entirely. Also scan `work/` for files with no document record and emit the triage
  report with inferred type and recommended action (`REGISTER_AND_MOVE`, `ARCHIVE`,
  `DELETE`, `REVIEW`). Implement `--porcelain` flag: when set, suppress all headers
  and emit tab-separated `ACTION\tSOURCE\tTARGET` lines only.
- **Deliverable:** `runMigrate` function (dry-run path) in `migrate_cmd.go`; `case
  "migrate"` entry in `cmd/kanbanzai/main.go`; human-readable and `--porcelain`
  output formatters.
- **Depends on:** T1
- **Effort:** Medium
- **Spec requirement:** REQ-001, REQ-003, REQ-004, REQ-005, REQ-006, REQ-007,
  REQ-013, REQ-014, REQ-NF-001, REQ-NF-002, REQ-NF-003;
  AC-001, AC-005, AC-006 (dry-run half), AC-007, AC-008, AC-014, AC-015, AC-016,
  AC-017

### Task 3: Implement --execute mode

- **Description:** Extend `runMigrate` to handle `--execute`: iterate the computed
  MOVE list and call `runMove` (from F3) for each entry; delete document records for
  MISSING entries by calling the delete service (from F4). Collect all staged changes
  and produce a single atomic Git commit with the message
  `chore(work): migrate work tree to plan-first folder structure`. On any `runMove`
  failure, stop immediately, print the failing file and error, leave already-staged
  moves intact, and exit non-zero. Implement `--resume`: before each move, check
  whether the source is already staged in Git and skip it if so.
- **Deliverable:** `--execute` and `--resume` branches of `runMigrate`; atomic commit
  logic using the existing `git` internal package.
- **Depends on:** T2; F2 (`FEAT-01KQ7JDSZARPC` merged), F3 (`FEAT-01KQ7JDT11MH6`
  merged), F4 (`FEAT-01KQ7JDT341E8` merged)
- **Effort:** Large
- **Spec requirement:** REQ-007, REQ-008, REQ-009, REQ-012;
  AC-006 (execute half), AC-009, AC-010, AC-013

### Task 4: Implement --execute --cleanup mode

- **Description:** After a successful `--execute` run, when `--cleanup` is also set,
  scan `work/` for directories that are now empty (excluding `work/templates/` and
  `work/_project/`). List the empty directories in output. Remove them with
  `git rm -r` and produce a second Git commit with the message
  `chore(work): remove empty legacy type-first folders`. If no directories are empty,
  skip the second commit and log accordingly.
- **Deliverable:** `--cleanup` branch of `runMigrate`; empty-directory detection and
  `git rm -r` invocation via the internal git package.
- **Depends on:** T3
- **Effort:** Small
- **Spec requirement:** REQ-010; AC-011

### Task 5: Tests — all 17 acceptance criteria

- **Description:** Write the full test suite for the migrate command. Unit tests for
  `resolveMigrateTarget` (three owner cases) and `inferTypeFromPath` (all legacy
  aliases). Integration tests using `testutil` temp-repo helpers for dry-run
  behaviour, SKIP/MISSING classification, triage report, `--execute` atomic commit,
  partial-failure halt and `--resume` recovery, `--execute --cleanup` second commit,
  idempotency, `work/templates/` and `docs/` exclusions, and `--porcelain` output
  format. Performance benchmark asserting dry-run completes in under 10 seconds for
  1,000 records and 1,000 files.
- **Deliverable:** `cmd/kanbanzai/migrate_cmd_test.go` with tests covering AC-001
  through AC-017.
- **Depends on:** T1, T2, T3, T4
- **Effort:** Large
- **Spec requirement:** All 17 acceptance criteria (AC-001–AC-017)

## Dependency Graph

```
T1  (no dependencies)
T2  → depends on T1
T3  → depends on T2, F2 (merged), F3 (merged), F4 (merged)
T4  → depends on T3
T5  → depends on T1, T2, T3, T4
```

Parallel groups: [T1] is the unblocked start; T2 follows T1; T3 is gated on T2
and three external merges; T4 and T5 follow T3/T4 respectively.

Critical path: T1 → T2 → T3 (+ F2 + F3 + F4) → T4 → T5

## Risk Assessment

### Risk: Dependency sequencing on F2, F3, and F4

- **Probability:** Medium
- **Impact:** High — T3 cannot be implemented at all without `runMove` (F3) and
  the delete service (F4). If those features slip, this entire feature is blocked.
- **Mitigation:** Plan T1 and T2 to be developed and code-reviewed in parallel with
  F2/F3/F4. Define stub interfaces or compile-time function references for `runMove`
  and `runDelete` so T3 can be drafted and syntax-checked before the real
  implementations land. Gate the merge of T3 on confirmed merge of F2, F3, and F4.
- **Affected tasks:** T3, T4, T5

### Risk: Orphaned records for missing files

- **Probability:** High (28+ unregistered files already known; orphaned records
  likely among the 430 registered docs)
- **Impact:** Medium — `--execute` must not fail or skip silently when a source
  file is absent; incorrect handling would leave the document registry inconsistent.
- **Mitigation:** T2 explicitly classifies MISSING as a first-class action and emits
  it in the dry-run report. T3 calls the delete service for MISSING entries before
  attempting file moves. Integration tests in T5 explicitly exercise this path
  (AC-006, AC-010).
- **Affected tasks:** T2, T3, T5

### Risk: Atomic commit failure partway through 430 moves

- **Probability:** Low
- **Impact:** High — a partial commit would leave the work tree in an inconsistent
  state with some records updated and some not.
- **Mitigation:** T3 stages all moves before committing (never calls `git commit`
  until all `runMove` calls succeed). On any failure, it halts with all prior moves
  staged but not committed so the user can fix the issue and `--resume`. The
  `--resume` flag detects already-staged files and skips them, enabling recovery
  without re-running successful moves. Integration test AC-010 validates this
  behaviour explicitly.
- **Affected tasks:** T3, T5

## Verification Approach

| Acceptance Criterion | Verification Method       | Producing Task |
|----------------------|---------------------------|----------------|
| AC-001 (dry-run report, no moves)          | Integration test | T2, T5 |
| AC-002 (feature-owner target path)         | Unit test        | T1, T5 |
| AC-003 (plan-owner target path)            | Unit test        | T1, T5 |
| AC-004 (PROJECT-owner target path)         | Unit test        | T1, T5 |
| AC-005 (SKIP for already-canonical path)   | Integration test | T2, T5 |
| AC-006 (MISSING: record deleted, no move)  | Integration test | T2, T3, T5 |
| AC-007 (triage report for unregistered)    | Integration test | T2, T5 |
| AC-008 (exit 0, no staged changes without --execute) | Integration test | T2, T5 |
| AC-009 (--execute atomic commit)           | Integration test | T3, T5 |
| AC-010 (--execute failure halts, moves staged) | Integration test | T3, T5 |
| AC-011 (--execute --cleanup second commit) | Integration test | T4, T5 |
| AC-012 (legacy folder type inference)      | Unit test        | T1, T5 |
| AC-013 (idempotent second run → all SKIP)  | Integration test | T3, T5 |
| AC-014 (work/templates/ always SKIP)       | Integration test | T2, T5 |
| AC-015 (docs/ excluded from all output)    | Integration test | T2, T5 |
| AC-016 (performance: 1,000 records < 10s)  | Benchmark test   | T5     |
| AC-017 (--porcelain tab-separated output)  | Integration test | T2, T5 |
```

Now I'll register the plan, then advance the feature, then decompose: