# B38-F9: Full Combined Migration — Dev Plan

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-04-28T16:55:00Z           |
| Status | Draft                          |
| Author | architect                      |
| Feature | (to be created)                |
| Spec   | `work/B38-plans-and-batches/B38-spec-p38-f9-full-combined-migration.md` |

---

## Scope

This plan implements the full combined migration defined in
`work/B38-plans-and-batches/B38-spec-p38-f9-full-combined-migration.md`
(B38-plans-and-batches/spec-b38-spec-p38-f9-full-combined-migration).

**In scope:** Dry-run audit report, target path resolution, bulk file migration
via `git mv`, document record path updates, empty directory cleanup, and
rollback validation. Tasks 1–8 below. All 16 acceptance criteria.

**Out of scope:** Unregistered file triage (human task, prerequisite),
`kbz move`/`kbz delete` CLI commands (P37-F3/F4), document content changes,
config migrations, worktree migrations.

**Prerequisites before implementation begins:**
- P38-F8 migration complete (state files, cross-references, display_ids)
- All feature `display_id` fields use B{n}-F{m} format
- Human triage of unregistered files complete
- Clean working tree on main branch

---

## Task Breakdown

### Task 1: Corpus Audit Script

- **Description:** Survey the project to catalogue every file in type-first
  directories, every document record, and every batch. Produce a machine-readable
  baseline snapshot used by the dry-run and migration scripts to detect drift.
- **Deliverable:** `work/B38-plans-and-batches/migration-audit.yaml` listing all
  type-first files, all document records, and all batch IDs.
- **Depends on:** None.
- **Effort:** Small.
- **Spec requirement:** REQ-001, REQ-002, REQ-003 (dry-run data source).

### Task 2: Dry-Run Report Generator

- **Description:** Read the audit snapshot and, for every registered document,
  resolve the target B{n} directory and canonical filename. Produce a dry-run
  report listing every proposed move. Flag unresolvable documents and
  unregistered files in separate sections.
- **Deliverable:** `work/B38-plans-and-batches/migration-dry-run.yaml` with
  structured entries: source path, target path, document record ID, status
  (resolved / unresolvable / unregistered).
- **Depends on:** Task 1.
- **Effort:** Medium.
- **Spec requirement:** REQ-001 through REQ-008.

### Task 3: Target Path Resolver

- **Description:** Implement the logic that maps a document record to its
  target B{n} directory and canonical filename. Feature-owned documents
  traverse `parent` to resolve the batch. Plan-owned and project-owned
  documents resolve directly. Construct filenames per the P37 template.
- **Deliverable:** `internal/service/migration_resolver.go` and
  `internal/service/migration_resolver_test.go`.
- **Depends on:** Task 1.
- **Effort:** Medium.
- **Spec requirement:** REQ-005, REQ-006.

### Task 4: Migration Execution Engine

- **Description:** Read the approved dry-run report and execute file moves
  via `git mv`. After each move, update the document record's `path` field.
  Handle edge cases: target directory creation (REQ-007), name conflicts
  (REQ-008), dirty tree abort (REQ-016). Commit all changes as a single
  Git commit.
- **Deliverable:** `internal/service/migration_executor.go` and
  `internal/service/migration_executor_test.go`.
- **Depends on:** Task 3.
- **Effort:** Large.
- **Spec requirement:** REQ-007 through REQ-011, REQ-014 through REQ-016.

### Task 5: Directory Cleanup

- **Description:** After migration, identify empty legacy type-first
  directories and remove them. Preserve directories that still contain
  files (unregistered, conflicts). Report which directories were removed
  and which were preserved.
- **Deliverable:** `internal/service/migration_cleanup.go` and
  `internal/service/migration_cleanup_test.go`.
- **Depends on:** Task 4.
- **Effort:** Small.
- **Spec requirement:** REQ-012, REQ-013.

### Task 6: Idempotency and Rollback Tests

- **Description:** Comprehensive tests proving idempotency (run twice, no
  changes on second run), content preservation (byte-for-byte identical
  after move), atomic commit (single git commit), and rollback (git revert
  restores original state). Includes dirty-tree abort test.
- **Deliverable:** `internal/service/migration_integration_test.go`.
- **Depends on:** Task 4, Task 5.
- **Effort:** Medium.
- **Spec requirement:** AC-013 through AC-016.

### Task 7: Logging and Error Handling

- **Description:** Add structured logging to stdout for every action
  (file moved, record updated, directory removed). Ensure exit codes
  follow REQ-NF-004 (0 on success, non-zero on failure with descriptive
  message). Add progress reporting for long-running operations.
- **Deliverable:** Modified migration executor with logging hooks, and
  `internal/service/migration_logger.go`.
- **Depends on:** Task 4.
- **Effort:** Small.
- **Spec requirement:** REQ-NF-003, REQ-NF-004.

### Task 8: Execution and Verification

- **Description:** Run the full migration against the actual project. Execute
  the dry-run, review the report, run the migration, verify all 16 acceptance
  criteria pass. This is the final validation task — no code changes, only
  execution and verification.
- **Deliverable:** Clean `git log` showing the migration commit.
  Post-migration verification report at
  `work/B38-plans-and-batches/B38-report-migration-verification.md`.
- **Depends on:** Task 6, Task 7.
- **Effort:** Medium.
- **Spec requirement:** All ACs (AC-001 through AC-016).

---

## Dependency Graph

```
Task 1 (no dependencies)
Task 2 → depends on Task 1
Task 3 → depends on Task 1
Task 4 → depends on Task 3
Task 5 → depends on Task 4
Task 6 → depends on Task 4, Task 5
Task 7 → depends on Task 4
Task 8 → depends on Task 6, Task 7

Parallel groups: [Task 2, Task 3], [Task 6, Task 7]
Critical path: Task 1 → Task 3 → Task 4 → Task 5 → Task 6 → Task 8
```

**Execution order:** 1 → (2, 3 in parallel) → 4 → (5) → (6, 7 in parallel) → 8

---

## Interface Contracts

### Contract 1: Audit Snapshot Format (Task 1 → Task 2, Task 3)

```yaml
# migration-audit.yaml
batches:
  - id: B1-phase-4-orchestration
    slug: phase-4-orchestration
type_first_files:
  - path: work/design/agent-onboarding.md
    sha256: abc123...
document_records:
  - id: P12-agent-onboarding/design-agent-onboarding
    owner: B12-agent-onboarding
    type: design
    path: work/design/agent-onboarding.md
```

### Contract 2: Dry-Run Report Format (Task 2 → Task 4)

```yaml
# migration-dry-run.yaml
entries:
  - source: work/design/agent-onboarding.md
    target: work/B12-agent-onboarding/B12-design-agent-onboarding.md
    doc_id: P12-agent-onboarding/design-agent-onboarding
    status: resolved
  - source: work/design/unknown.md
    target: null
    doc_id: null
    status: unresolvable
    reason: "owner field missing from document record"
unresolvable: [...]
unregistered: [...]
conflicts: [...]
```

### Contract 3: Migration Executor (Task 4)

- Input: approved dry-run report (Contract 2)
- Behaviour: execute moves, update records, commit
- Output: Git commit, modified `.kbz/state/documents/`, removed legacy directories
- Error: abort on dirty tree, conflict, or resolution failure

---

## Risk Assessment

### Risk: Human Triage Delays Migration
- **Probability:** Medium — unregistered files require human decisions.
- **Impact:** Medium — blocks Task 8 execution.
- **Mitigation:** Surface unregistered files early in the dry-run report (Task 2).
  Provide clear per-file recommendations (register-and-move, archive, delete).
- **Affected tasks:** Task 2, Task 8.

### Risk: Document Record Owner Resolution Failure
- **Probability:** Low — all features have parent batches and all document
  records have owners.
- **Impact:** High — unresolvable documents block migration or are left behind.
- **Mitigation:** Dry-run flagging (Task 2) catches all unresolvable cases before
  execution. No migration proceeds until the dry-run report is clean.
- **Affected tasks:** Task 2, Task 4.

### Risk: Filename Collision After Rename
- **Probability:** Low — canonical template plus unique slugs makes collisions
  unlikely.
- **Impact:** Medium — skipped file leaves document in wrong location.
- **Mitigation:** REQ-008 skips collisions and reports them. Task 2 detects
  collisions in dry-run so they can be resolved before migration.
- **Affected tasks:** Task 2, Task 4.

### Risk: Git Index Corruption During Bulk Move
- **Probability:** Low — `git mv` is a stable operation.
- **Impact:** High — corrupted index could lose file history.
- **Mitigation:** Single atomic commit (REQ-011) means any failure leaves the
  repo in a known state. Pre-migration clean-tree check (REQ-016) ensures no
  uncommitted changes are lost. Rollback via `git revert` is always available.
- **Affected tasks:** Task 4, Task 6.

### Risk: Migration Script Bug Causes Partial State
- **Probability:** Medium — the migration touches ~400 files and document records.
- **Impact:** High — partial migration leaves inconsistent state.
- **Mitigation:** Dry-run preview (Task 2), idempotency tests (Task 6), atomic
  commit (Task 4), and git revert rollback strategy. Run tests against a copy
  of the repo first.
- **Affected tasks:** Task 4, Task 6, Task 8.

### Risk: Performance Exceeds Time Budget
- **Probability:** Low — 400 files × git mv is well under 60 seconds.
- **Impact:** Low — no hard deadline; the budget is advisory.
- **Mitigation:** Task 1 establishes a baseline count. Task 2 measures
  resolution time. If resolution is slow, profile and optimise before Task 4.
- **Affected tasks:** Task 2, Task 4.

---

## Verification Approach

| Acceptance Criterion | Method | Producing Task |
|---|---|---|
| AC-001: Dry-run lists all moves | Unit test | Task 2 |
| AC-002: Unresolvable docs flagged | Unit test | Task 2 |
| AC-003: Unregistered files flagged | Unit test | Task 2 |
| AC-004: Feature-owned doc resolves to B{n} dir | Unit test | Task 3 |
| AC-005: Plan-owned doc gets canonical name | Unit test | Task 3 |
| AC-006: Feature-owned doc gets canonical name | Unit test | Task 3 |
| AC-007: Target directory auto-created | Unit test | Task 4 |
| AC-008: Name conflict skipped with report | Unit test | Task 4 |
| AC-009: git mv + doc record update | Integration test | Task 4 |
| AC-010: Single atomic commit | Integration test | Task 6 |
| AC-011: Empty dirs removed | Unit test | Task 5 |
| AC-012: Non-empty dirs preserved | Unit test | Task 5 |
| AC-013: Idempotent (run twice = no-op) | Integration test | Task 6 |
| AC-014: Content byte-identical after move | Integration test | Task 6 |
| AC-015: Abort on dirty tree | Unit test | Task 4 |
| AC-016: Completes ≤ 60s | Integration test | Task 6 |

---

## Traceability Matrix

| Spec Requirement | Task(s) |
|-----------------|---------|
| REQ-001 (dry-run report) | Task 2 |
| REQ-002 (unresolvable flag) | Task 2 |
| REQ-003 (unregistered flag) | Task 2 |
| REQ-004 (machine-readable format) | Task 2 |
| REQ-005 (target batch resolution) | Task 3 |
| REQ-006 (canonical filename) | Task 3 |
| REQ-007 (directory creation) | Task 4 |
| REQ-008 (conflict handling) | Task 4 |
| REQ-009 (git mv) | Task 4 |
| REQ-010 (doc record update) | Task 4 |
| REQ-011 (atomic commit) | Task 4 |
| REQ-012 (empty dir removal) | Task 5 |
| REQ-013 (non-empty dir preservation) | Task 5 |
| REQ-014 (idempotency) | Task 6 |
| REQ-015 (content preservation) | Task 6 |
| REQ-016 (dirty tree abort) | Task 4 |
| REQ-NF-001 (≤ 60s) | Task 6, Task 8 |
| REQ-NF-002 (dry-run ≤ 10s) | Task 2 |
| REQ-NF-003 (logging) | Task 7 |
| REQ-NF-004 (exit codes) | Task 7 |
