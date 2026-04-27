# P37-F3: kbz move Command — Implementation Plan

| Field  | Value                                                    |
|--------|----------------------------------------------------------|
| Date   | 2026-04-27T14:11:10Z                                     |
| Status | Draft                                                    |
| Author | orchestrator                                             |
| Spec   | work/design/p37-f3-spec-kbz-move.md                     |

---

## Scope

This plan implements the requirements defined in
`work/design/p37-f3-spec-kbz-move.md` for the `kbz move` CLI command
(FEAT-01KQ7JDT11MH6, Plan P37-file-names-and-actions).

It covers five tasks: adding the `GitMove` Git helper (T1), implementing Mode 1
file-move logic (T2), wiring the dispatch entry point in `main.go` (T3),
implementing Mode 2 feature re-parent logic (T4), and writing the full test
suite (T5).

It does not cover any changes to the MCP server or to the existing
`doc(action: "move")` MCP tool action (REQ-NF-003).

**Mode 2 dependency:** T4 (Mode 2 — feature re-parent) must not be started
until F1 (FEAT-01KQ7JDSVMP4E, display IDs / `next_feature_seq`) has merged into
`main`. Mode 1 (T1–T3) has no such dependency and may proceed independently.

---

## Task Breakdown

### Task 1: Add `GitMove` to `internal/git/git.go`

- **Description:** Add an exported `GitMove(repoRoot, src, dst string) error`
  function to `internal/git/git.go`. It must follow the `runGitCmd` internal
  helper pattern already present in the package (compare `GetFileLastModified`)
  by calling `exec.Command("git", "mv", src, dst)` with `cmd.Dir = repoRoot`
  and forwarding stderr on failure. This is the single authorised file-move
  primitive for the entire `kbz move` command; `os.Rename` must not be used
  for file relocation anywhere in this feature.
- **Deliverable:** `GitMove` exported function in `internal/git/git.go`;
  no other files changed in this task.
- **Depends on:** None.
- **Effort:** Small.
- **Spec requirement:** REQ-009, REQ-NF-002.

---

### Task 2: Implement Mode 1 file move in `cmd/kanbanzai/move_cmd.go`

- **Description:** Create the new file `cmd/kanbanzai/move_cmd.go` containing
  `func runMove(args []string, deps dependencies) error`. Implement the full
  Mode 1 (file-move) path:

  1. **Argument disambiguation** — detect Mode 1 when the first arg contains
     `/` or ends with `.md`/`.txt` and the second matches `P{n}` or
     `P{n}-{slug}`. Return a usage error for unrecognised argument shapes.
  2. **`work/` boundary check** — reject source paths not prefixed with
     `work/`; write error to stderr and return non-zero.
  3. **Source file validation** — `os.Stat` the source; return non-zero with
     "file not found" on stderr if absent.
  4. **Target plan validation** — verify the plan exists in
     `.kbz/state/plans/`; return non-zero with "plan not found" on stderr if
     absent.
  5. **Document-type determination** — look up the source path in the document
     store (`service.DocumentService`); if a record exists, use its `Type`;
     otherwise call `inferDocTypeFromPath` (already in `internal/service`) or
     infer from the filename stem.
  6. **Canonical target-path construction** — build
     `work/P{n}-{slug}/{PlanID}-{type}-{filestem}.{ext}` per REQ-006. Derive
     `{filestem}` from the record title-slug if available, else strip the
     known prefix pattern from the source filename stem, else use the full
     source filename stem.
  7. **Reject existing target** — if the target path exists on disk, write
     "target already exists" to stderr and return non-zero.
  8. **Create target folder** — `os.MkdirAll` the target plan folder.
  9. **Execute git mv** — call `internal/git.GitMove(repoRoot, src, dst)`.
  10. **Update document record** — if a record exists, call
      `service.MoveDocument` with the new path and update `owner` to the
      target plan ID.
  11. **Unregistered-file handling** — if no record, move succeeds; print the
      warning `No document record found — file moved but no record updated`
      to stdout and exit 0.
  12. **Print summary** — on success print `Moved <source> → <target>` to
      stdout.

  Follow the dependency-injection pattern from `doc_cmd.go`: obtain
  `repoRoot` via `core.FindRepoRoot()`, construct `service.DocumentService`
  and `service.EntityService` from `deps.newEntityService`.

- **Deliverable:** `cmd/kanbanzai/move_cmd.go` with `runMove` handling Mode 1;
  Mode 2 stub (or guarded `TODO`) present but inert.
- **Depends on:** T1.
- **Effort:** Medium.
- **Spec requirement:** REQ-001 through REQ-012, REQ-NF-001, REQ-NF-002,
  REQ-NF-005.

---

### Task 3: Wire dispatch in `cmd/kanbanzai/main.go`

- **Description:** Add `case "move": return runMove(args[1:], deps)` to the
  `switch args[0]` dispatcher in `run()` in `cmd/kanbanzai/main.go`, in the
  "Core workflow commands" group (alongside `doc`, `entity`). Add a brief
  `move` entry to `usageText`.
- **Deliverable:** Modified `cmd/kanbanzai/main.go`; `kbz move` is now
  reachable from the binary.
- **Depends on:** T2.
- **Effort:** Small.
- **Spec requirement:** REQ-NF-001.

---

### Task 4: Implement Mode 2 feature re-parent in `move_cmd.go`

- **Description:** Extend `runMove` in `cmd/kanbanzai/move_cmd.go` to handle
  Mode 2 (feature re-parent). **This task must not begin until F1
  (FEAT-01KQ7JDSVMP4E) is merged**, because it depends on the display-ID index
  and `next_feature_seq` counter introduced by F1.

  Implement the following steps inside the Mode 2 branch:

  1. **Display-ID resolution** — resolve `P{n}-F{m}` to a canonical
     `FEAT-{TSID}` via the display-ID index (from F1). Exit non-zero with
     "feature not found" on stderr if unresolvable.
  2. **Same-plan guard** — load the feature entity; if its current `parent`
     equals the target plan, exit non-zero with an explanatory message on
     stderr.
  3. **Confirmation prompt** — unless `--force` is present in args, enumerate
     all planned changes (feature entity fields + each doc path rewrite) and
     print to stdout, then prompt `Proceed? [y/N]: `. If the user does not
     enter `y`/`Y`, exit 0 with no changes.
  4. **Display-ID allocation** — read the target plan's `next_feature_seq`,
     form `P{n}-F{next}`, increment and persist `next_feature_seq`.
  5. **Feature entity update** — set `parent` → target plan ID and
     `display_id` → new display ID; persist via the entity service.
  6. **Bulk document moves** — for each document record owned by the feature
     canonical ID: construct the new canonical path (REQ-006 logic with new
     plan prefix), call `GitMove`, update record `path` and `owner`.
  7. **Output summary** — print `Moved feature P{n}-F{m} → P{n2}-F{m2}`,
     followed by one `Moved <old> → <new>` line per document.

- **Deliverable:** Mode 2 fully implemented in `cmd/kanbanzai/move_cmd.go`.
- **Depends on:** T2, F1 (FEAT-01KQ7JDSVMP4E).
- **Effort:** Large.
- **Spec requirement:** REQ-013 through REQ-020, REQ-NF-004.

---

### Task 5: Tests covering all 18 acceptance criteria

- **Description:** Write tests in `cmd/kanbanzai/move_cmd_test.go` (and a unit
  test file alongside `internal/git/git_test.go` if needed) covering all 18
  acceptance criteria from the spec. Use the temporary-git-repo helper pattern
  from `testutil` for integration tests that require a real Git repository.

  **Mode 1 unit tests** (pure logic, no filesystem):
  - `TestModeDetection_SlashInPath` (AC-001)
  - `TestModeDetection_DotMdExtension` (AC-002)
  - `TestRejectOutsideWorkDir` (AC-003)
  - `TestModeDetection_FeatureDisplayID` (AC-011)

  **Mode 1 integration tests** (real Git repo fixture):
  - `TestFileMoveSourceNotFound` (AC-004)
  - `TestFileMoveTargetPlanNotFound` (AC-005)
  - `TestFileMoveRegisteredDocument` — includes `git log --follow` history
    assertion (AC-006)
  - `TestFileMoveCreatesPlanFolder` (AC-007)
  - `TestFileMoveTargetAlreadyExists` (AC-008)
  - `TestFileMoveUnregisteredFile` (AC-009)

  **AC-010 code-review criterion:** document in test file that `os.Rename` is
  absent from `move_cmd.go` (grep assertion or comment referencing the
  constraint).

  **Mode 2 integration tests** (depend on T4; skip/build-tag guard until F1):
  - `TestReParentFeatureNotFound` (AC-012)
  - `TestReParentSamePlan` (AC-013)
  - `TestReParentConfirmationPrompt` (AC-014)
  - `TestReParentForceFlag` (AC-015)
  - `TestReParentEntityUpdate` (AC-016)
  - `TestReParentDocumentMoves` — includes `git log --follow` assertions
    (AC-017)
  - `TestReParentOutputSummary` (AC-018)

- **Deliverable:** `cmd/kanbanzai/move_cmd_test.go`; all Mode 1 tests pass
  (`go test ./cmd/kanbanzai/...`); Mode 2 tests guarded and passing after T4.
- **Depends on:** T1, T2, T3, T4.
- **Effort:** Large.
- **Spec requirement:** All 18 acceptance criteria (AC-001 – AC-018).

---

## Dependency Graph

```
T1  (no dependencies)
T2  → depends on T1
T3  → depends on T2
T4  → depends on T2, F1 (FEAT-01KQ7JDSVMP4E)
T5  → depends on T1, T2, T3, T4
```

Parallel groups:
- **Group A (now):** T1 only.
- **Group B (after T1):** T2.
- **Group C (after T2):** T3 and T4-prep (T4 blocked on F1).
- **Group D (after T3 + T4):** T5 in full; T5 Mode-1 subset can start after T3.

Critical path: T1 → T2 → T3 → T5 (Mode 1 complete)
Critical path (full): T1 → T2 → [F1 merge] → T4 → T5

---

## Risk Assessment

### Risk: Git history not preserved through `git mv`

- **Probability:** Low.
- **Impact:** High — AC-006 and AC-017 explicitly require `git log --follow`
  to show unbroken history; failure here is a visible regression for users.
- **Mitigation:** T1 implements `GitMove` via `exec.Command("git", "mv", ...)`
  with the working directory set to the repo root. Integration tests in T5
  (AC-006, AC-017) assert `git log --follow` output directly, failing the build
  if history is broken. `os.Rename` is explicitly prohibited; a grep assertion
  in the test file enforces this.
- **Affected tasks:** T1, T2, T4, T5.

### Risk: Mode 2 blocked by F1 merge timing

- **Probability:** Medium — F1 (FEAT-01KQ7JDSVMP4E) is a separate in-flight
  feature with its own timeline; delays in F1 block T4 and the Mode-2 subset
  of T5.
- **Impact:** Medium — Mode 1 is fully deliverable independently; Mode 2 is
  deferred but the plan and stub structure are in place.
- **Mitigation:** T2 and T3 ship Mode 1 as a complete, independently-releasable
  slice. T4 and Mode-2 tests carry a build-tag or compile guard so the branch
  can be merged with only Mode 1 active. T4 is explicitly marked `Depends on
  F1` in this plan.
- **Affected tasks:** T4, T5.

### Risk: Canonical path construction produces collisions or wrong stems

- **Probability:** Medium — the `{filestem}` derivation rule (REQ-006) has
  three fallback levels (record title-slug → strip prefix from filename →
  full stem) and edge cases exist when filenames do not follow the canonical
  pattern.
- **Impact:** Medium — a wrong stem produces a confusing target path; a
  collision is caught by the REQ-007 guard and surfaces as an error rather than
  silent data loss.
- **Mitigation:** T2 extracts the stem-derivation logic into a small,
  independently-testable helper function. Unit tests in T5 exercise each
  fallback branch and boundary cases (no record, non-canonical source name,
  record with title-slug). AC-008 ensures the collision guard works correctly.
- **Affected tasks:** T2, T5.

---

## Verification Approach

| Acceptance Criterion | Verification Method  | Producing Task |
|----------------------|----------------------|----------------|
| AC-001 (REQ-001)     | Unit test            | T5             |
| AC-002 (REQ-001)     | Unit test            | T5             |
| AC-003 (REQ-002)     | Unit test            | T5             |
| AC-004 (REQ-003)     | Integration test     | T5             |
| AC-005 (REQ-004)     | Integration test     | T5             |
| AC-006 (REQ-005–010, REQ-012) | Integration test (incl. `git log --follow`) | T5 |
| AC-007 (REQ-008)     | Integration test     | T5             |
| AC-008 (REQ-007)     | Integration test     | T5             |
| AC-009 (REQ-011)     | Integration test     | T5             |
| AC-010 (REQ-009, REQ-NF-002) | Code review + grep assertion in test | T5 |
| AC-011 (REQ-013)     | Unit test            | T5             |
| AC-012 (REQ-014)     | Integration test     | T5             |
| AC-013 (REQ-015)     | Integration test     | T5             |
| AC-014 (REQ-016)     | Integration test     | T5             |
| AC-015 (REQ-016)     | Integration test     | T5             |
| AC-016 (REQ-017–018) | Integration test     | T5             |
| AC-017 (REQ-019)     | Integration test (incl. `git log --follow`) | T5 |
| AC-018 (REQ-020)     | Integration test     | T5             |