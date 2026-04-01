# Implementation Plan: Sub-Agent State Isolation

| Field    | Value                                                              |
|----------|--------------------------------------------------------------------|
| Status   | Draft                                                              |
| Created  | 2026-04-01                                                         |
| Feature  | FEAT-01KN4ZPG44T5B (sub-agent-state-isolation)                     |
| Spec     | `work/spec/sub-agent-state-isolation.md`                           |
| Design   | `work/design/kanbanzai-2.5-infrastructure-hardening.md` §4         |

---

## 1. Implementation Approach

This feature adds a pre-dispatch state commit to the `handoff` tool. Before
assembling a sub-agent prompt, `handoff` checks for uncommitted changes under
`.kbz/state/` and commits them if any exist. The commit is best-effort: failure
is logged but does not block the handoff.

The work decomposes into two sequential tasks:

**Task 1** implements the git utility function that stages and commits only
`.kbz/state/` changes. This is an isolated concern with a clear interface.

**Task 2** integrates the utility into `handoff_tool.go`, adds the failure
logging path, and writes tests. It cannot begin until Task 1's interface is
stable.

```
[Task 1: Git utility — CommitStateIfDirty] ──► [Task 2: Handoff integration + tests]
```

No parallelism is possible; the dependency is strict. Both tasks are small.
The feature can be completed in a single session by one agent working
sequentially.

---

## 2. Interface Contract

Task 2 depends on the following function exported from the git utility
(Task 1):

```
// CommitStateIfDirty stages all files under .kbz/state/ that have
// uncommitted changes and creates a commit. Returns (true, nil) if a
// commit was created, (false, nil) if the working tree was clean, and
// (false, err) if staging or committing failed.
func CommitStateIfDirty(repoRoot string) (committed bool, err error)
```

- `repoRoot` is the absolute path to the repository root (the directory
  containing `.git/`). The caller is responsible for determining this path.
- The function MUST NOT stage files outside `.kbz/state/`.
- The function MUST NOT create an empty commit.
- The commit message is fixed: `chore(kbz): persist workflow state before sub-agent dispatch`
- On success with no changes, return `(false, nil)` — not an error.

Task 2 codes against this signature. Task 1 must honour it exactly.

---

## 3. Task Breakdown

| # | Task | Primary Files | Spec Refs |
|---|------|---------------|-----------|
| 1 | Git utility: `CommitStateIfDirty` | `internal/git/commit.go` (new or extend) | REQ-01–05 |
| 2 | Handoff integration + tests | `internal/mcp/handoff_tool.go`, `internal/mcp/handoff_tool_test.go` | REQ-01–08, AC-07–11 |

---

## 4. Task Details

### Task 1: Git Utility — `CommitStateIfDirty`

**Objective:** Implement a function that detects uncommitted `.kbz/state/`
changes, stages them, and creates a commit with the fixed commit message.
Returns a clean signal when there is nothing to commit.

**Specification references:** REQ-01 (detect changes), REQ-02 (create commit),
REQ-03 (scope to `.kbz/state/` only), REQ-04 (commit message), REQ-05 (no
empty commits).

**Input context:**

- `work/spec/sub-agent-state-isolation.md` — full spec, especially §4.1.
- `work/design/git-commit-policy.md` — commit message conventions (confirm the
  `chore(kbz):` prefix is compliant).
- Existing git integration in `internal/git/` — understand what helpers already
  exist (e.g., `git.Run`, shell execution wrappers) before adding new ones.
- The function signature in §2 of this plan — implement exactly this interface.

**Output artefacts:**

- `internal/git/commit.go` — new file or extension of an existing file in
  `internal/git/`. Add `CommitStateIfDirty(repoRoot string) (bool, error)`.
- No test file required at this stage — tests are in Task 2's scope. However,
  if the package already has a test file, a simple unit test for the "nothing
  to commit" path may be added here.

**Key implementation notes:**

- Use `git status --porcelain -- .kbz/state/` (or equivalent) to detect
  changes. An empty output means nothing to commit.
- Use `git add -- .kbz/state/` to stage. Do not use `git add -A` or any form
  that would capture files outside `.kbz/state/`.
- Use `git commit -m "chore(kbz): persist workflow state before sub-agent
  dispatch"` to commit.
- All git operations should be invoked through whatever shell/exec abstraction
  the existing `internal/git/` package uses — do not add a new subprocess
  mechanism.

---

### Task 2: Handoff Integration + Tests

**Objective:** Call `CommitStateIfDirty` at the start of the `handoff` tool
handler, before context assembly begins. Log a warning if it returns an error.
Write tests covering all five acceptance criteria.

**Specification references:** REQ-01 (call before assembly), REQ-06 (log
warning on failure), REQ-07 (non-blocking), REQ-08 (assembly unchanged),
AC-07–AC-11.

**Input context:**

- `work/spec/sub-agent-state-isolation.md` — §4 and §5 (requirements and
  acceptance criteria).
- `internal/mcp/handoff_tool.go` — read the existing handler to understand
  where in the call flow to insert the pre-commit step.
- `internal/git/commit.go` — the `CommitStateIfDirty` interface from Task 1.
- Existing `internal/mcp/handoff_tool_test.go` (if present) — understand
  current test structure before adding new cases.
- `work/design/git-commit-policy.md` — warning log format conventions if any.

**Output artefacts:**

- `internal/mcp/handoff_tool.go` — add call to `CommitStateIfDirty` before
  the first context assembly step. Use the repository root derived from the
  existing config/path resolution already present in the handler.
- `internal/mcp/handoff_tool_test.go` — tests for AC-07 through AC-11:
  - AC-07: dirty `.kbz/state/` → commit created before assembly.
  - AC-08: commit contains only `.kbz/state/` files.
  - AC-09: commit message is exactly the specified string.
  - AC-10: clean state → no commit created.
  - AC-11: commit failure → warning logged, handoff returns normally.

**Testing approach:**

Use `t.TempDir()` to create a real git repository for integration-style tests.
Initialise a git repo, write files under `.kbz/state/`, call
`CommitStateIfDirty`, and assert on the git log. For the failure case (AC-11),
make the repository unwritable or pass a non-existent path to force an error,
then assert on the log output.

If a full git repo in a temp dir is impractical within existing test
infrastructure, inject `CommitStateIfDirty` as a function parameter or
interface in `handoff_tool.go` so the test can substitute a stub. Document
this injection point in a comment.

**AC cross-references:**

- AC-07 → test: dirty state, verify git log has new commit
- AC-08 → test: git show HEAD confirms only `.kbz/state/**` files
- AC-09 → test: git log --format=%s HEAD confirms message
- AC-10 → test: clean state, verify git log is unchanged
- AC-11 → test: stubbed/erroring commit function, verify warning in log
  and that the assembled prompt is returned successfully

---

## 5. Scope Boundaries

Carried forward from `work/spec/sub-agent-state-isolation.md` §3.2:

- **Not in scope:** preventing sub-agents from running git operations.
- **Not in scope:** committing non-state changes (code, documents, `.kbz/index/`,
  `.kbz/cache/`).
- **Not in scope:** modifying the `next` tool.
- **Not in scope:** skill-level anti-pattern documentation (deferred to V3.0).
- **Not in scope:** widening commit scope to other `.kbz/` subdirectories.

---

## 6. Verification Checklist

Before marking this feature done:

- [ ] `go test ./internal/git/...` passes with no regressions.
- [ ] `go test ./internal/mcp/...` passes with no regressions, including new
  AC-07 through AC-11 tests.
- [ ] `go test -race ./...` passes.
- [ ] `go vet ./...` reports no issues.
- [ ] A manual smoke test: run `handoff` against a feature with dirty
  `.kbz/state/`; confirm git log shows the pre-dispatch commit.