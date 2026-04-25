# Implementation Plan: Decompose Paired Test Tasks

| Field   | Value                                                                      |
|---------|----------------------------------------------------------------------------|
| Date    | 2026-04-25                                                                 |
| Status  | Draft                                                                      |
| Feature | FEAT-01KQ2E0RNY261 (decompose-paired-test-tasks)                           |
| Spec    | `work/spec/feat-01kq2e0rny261-decompose-paired-test-tasks.md`              |
| Plan    | P34-agent-workflow-ergonomics                                              |

---

## 1. Scope

This plan implements the requirements defined in
`work/spec/feat-01kq2e0rny261-decompose-paired-test-tasks.md`. It covers two
tasks: replacing the global "Write tests" catch-all injection in
`internal/service/decompose.go` with per-unit paired test task generation, and
the accompanying tests. It does not cover changes to `decomposeApply`,
`decomposeReview`, or any other part of the decompose pipeline.

---

## 2. Implementation Approach

The change is contained entirely within `internal/service/decompose.go`. The
global test-task injection block (currently at the end of `generateProposal`,
lines 675–691) is removed and replaced with per-unit logic inside the task
generation loop.

After each implementation task is appended to the `tasks` slice, the loop
checks whether all of the task's `Covers` texts contain the word "test"
(case-insensitive). If not all do, a paired test task is immediately appended.

```
[Task 1: Algorithm change in decompose.go]  ──→  [Task 2: Tests]
```

Task 2 should be written alongside Task 1 (or TDD-first). Both tasks touch
the same two files and are delivered together.

---

## 3. Interface Contract

No exported function signatures change. `ProposedTask` already carries
`DependsOn []string` and `Covers []string` fields; no struct changes are
required.

The paired test task fields are derived deterministically:

| Field     | Value |
|-----------|-------|
| `Slug`    | `{impl-slug}-tests` |
| `Name`    | `Test {impl-name}` |
| `Summary` | `Write tests covering: {impl-summary}` |
| `DependsOn` | `[]string{impl-slug}` |
| `Covers`  | same `[]string` as the implementation task |

The `decomposeApply` pipeline already resolves slug-based `DependsOn` entries
to task IDs in Pass 2. No change to `decomposeApply` is needed.

---

## 4. Task Breakdown

| # | Task | Files | Spec refs |
|---|------|-------|-----------|
| 1 | Replace global catch-all with per-unit paired test tasks | `internal/service/decompose.go` | FR-001–FR-008 |
| 2 | Unit tests for paired test task generation | `internal/service/decompose_test.go` | AC-001–AC-008 |

---

## 5. Task Details

### Task 1: Replace global catch-all with per-unit paired test task generation

**Objective:** Modify `generateProposal` in `internal/service/decompose.go` so
that each implementation task is followed immediately by a paired test task
(subject to the exception rule), and the global catch-all injection is removed.

**Specification references:** FR-001, FR-002, FR-003, FR-004, FR-005, FR-006,
FR-007, FR-008.

**Input context:**

- Read `internal/service/decompose.go` in full. The relevant section is the
  `generateProposal` function. Locate the task-generation loop and the
  catch-all block that follows it (currently checks `hasTestTask` and appends a
  global "Write tests" task).
- Review the `ProposedTask` struct (top of the file) to confirm the available
  fields.
- Note that `Covers` on each generated task already contains the AC texts that
  task addresses — this is the input to the exception check.

**Output artifacts:**

- `internal/service/decompose.go`:

  1. **Remove** the `hasTestTask` variable, the loop that sets it, and the
     conditional block that appends the global "Write tests" task (the block
     currently spanning roughly lines 675–691).

  2. **Add** a helper function (unexported) for the exception check:

     ```go
     // allCoversContainTest returns true when every AC text in covers
     // contains the word "test" (case-insensitive). An empty slice returns true
     // (no paired test task is generated for a task that covers nothing).
     func allCoversContainTest(covers []string) bool {
         if len(covers) == 0 {
             return true
         }
         for _, ac := range covers {
             if !strings.Contains(strings.ToLower(ac), "test") {
                 return false
             }
         }
         return true
     }
     ```

  3. **Modify** the task-generation loop: immediately after `tasks = append(tasks,
     implTask)`, call `allCoversContainTest(implTask.Covers)`. When it returns
     `false`, append the paired test task:

     ```go
     if !allCoversContainTest(implTask.Covers) {
         tasks = append(tasks, ProposedTask{
             Slug:      implTask.Slug + "-tests",
             Name:      "Test " + implTask.Name,
             Summary:   "Write tests covering: " + implTask.Summary,
             DependsOn: []string{implTask.Slug},
             Covers:    implTask.Covers,
             Rationale: "Paired test task for " + implTask.Slug,
         })
     }
     ```

**Constraints:**

- Do not change any other part of `generateProposal`. The grouping heuristic,
  section processing, slug generation, and dependency detection logic are all
  unchanged.
- Do not modify `decomposeApply`, `decomposeReview`, or `ProposedTask`.
- The `allCoversContainTest` helper must be a case-insensitive substring check
  on the AC text strings in `covers` — not on task slugs, names, or summaries.

---

### Task 2: Unit tests for paired test task generation

**Objective:** Confirm all acceptance criteria with targeted tests. No existing
tests should regress.

**Specification references:** AC-001, AC-002, AC-003, AC-004, AC-005, AC-006,
AC-007, AC-008.

**Input context:**

- Read `internal/service/decompose_test.go` for existing test patterns,
  particularly any tests for `generateProposal` or the global test-task
  injection. The existing catch-all test cases will need updating to reflect
  the removal of the global "Write tests" task.
- The spec acceptance criteria map directly to the test cases below.

**Output artifacts:**

- `internal/service/decompose_test.go`:

  | Test case | Input | Expected result |
  |-----------|-------|-----------------|
  | AC-001 | 3 impl tasks, none with "test" in AC text | 6 tasks total; each test task follows its paired impl task |
  | AC-002 | Impl task with slug `implement-entity-cache` | Test task slug = `implement-entity-cache-tests`, name = `Test Implement entity cache`, DependsOn = `["implement-entity-cache"]` |
  | AC-003 | Same impl task | Test task `Covers` == impl task `Covers` |
  | AC-004 | Impl task whose sole AC text is `"Add a regression test for the overflow path"` | No paired test task generated |
  | AC-005 | Impl task covering 2 ACs: one contains "test", one does not | Paired test task IS generated |
  | AC-006 | Any feature spec | Output contains no task with name `"Write tests"` covering the whole feature |
  | AC-007 | Impl task whose AC is `"Implement the new resolver"` (no "test") | Paired test task IS generated |
  | AC-008 | 4 impl tasks; 1 whose only AC contains "test", 3 without | 7 tasks total (4 impl + 3 paired) |

  Add `TestAllCoversContainTest` as a table-driven unit test for the helper
  function directly, covering: empty slice → true; all contain "test" → true;
  none contain "test" → false; mixed → false; case-insensitive match → true.

  Update any existing test that asserts a global "Write tests" task is appended
  at the end of the proposal — such tests now expect the paired pattern instead.

**Constraints:**

- Tests must pass under `go test ./...` and `go test -race ./...`.
- Do not remove existing tests; update those that are invalidated by the
  removal of the global catch-all.

---

## 6. Dependency Graph

```
Task 1 (algorithm change)
Task 2 (tests) → depends on Task 1 (must pass after fix lands)

Parallel groups: none (Task 2 validates Task 1)
Critical path: Task 1 → Task 2
```

---

## 7. Risk Assessment

### Risk: Existing tests assert the global catch-all pattern
- **Probability:** high
- **Impact:** low (tests will fail at CI, not silently)
- **Mitigation:** Task 2 explicitly requires updating tests that assert the old
  global "Write tests" task. Read existing test coverage before modifying.
- **Affected tasks:** Task 2

### Risk: `Covers` is empty on some generated tasks
- **Probability:** medium
- **Impact:** low — `allCoversContainTest` returns `true` for empty slices,
  meaning no paired test task is generated. This is the conservative choice
  (avoids spurious paired tasks for tasks with no traceable ACs).
- **Mitigation:** The helper function spec above explicitly documents this
  edge case. Verify with a unit test case.
- **Affected tasks:** Task 1, Task 2

---

## 8. Acceptance Criteria Traceability

| Acceptance Criterion | Verification Method | Producing Task |
|---------------------|---------------------|----------------|
| AC-001 (6 tasks for 3 impl units) | Unit test | Task 1 (fix), Task 2 (test) |
| AC-002 (slug/name/DependsOn fields) | Unit test | Task 1 (fix), Task 2 (test) |
| AC-003 (Covers matches impl task) | Unit test | Task 1 (fix), Task 2 (test) |
| AC-004 (exception: all ACs contain "test") | Unit test | Task 1 (fix), Task 2 (test) |
| AC-005 (mixed ACs: paired task generated) | Unit test | Task 1 (fix), Task 2 (test) |
| AC-006 (no global "Write tests" task) | Unit test | Task 1 (fix), Task 2 (test) |
| AC-007 (impl AC without "test" → paired) | Unit test | Task 1 (fix), Task 2 (test) |
| AC-008 (4 impl, 1 exception → 7 tasks) | Unit test | Task 1 (fix), Task 2 (test) |

All requirements FR-001 through FR-008 are satisfied by Task 1.

---

## 9. Verification

Run after both tasks are complete:

```
go build ./...
go test ./internal/service/...
go test -race ./internal/service/...
go vet ./...
```

All must pass with zero failures and no new race conditions.