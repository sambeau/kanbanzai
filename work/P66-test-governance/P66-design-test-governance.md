# Design: Test Governance Framework

## Overview

Enforce the "tests must always pass" principle through a persistent, machine-readable
record of test suite health that agents and humans can both consult. Currently, test
discipline relies on a pre-commit hook and post-merge verification — both ephemeral
checks that leave no trace for AI agents to discover on their own.

The full test suite completes in approximately 37 seconds uncached (1.5 seconds
cached by Go's built-in test cache). This speed means a time-based TTL is not
needed — the decision to re-run is driven by source-code modification, not elapsed
time.

## Goals

1. Any agent can determine the test suite health of `main` without running tests.
2. The orchestrator refuses to begin new implementation work when tests are failing.
3. A failing test is surfaced prominently in every status check until fixed.
4. The record is self-evident — a human looking at the codebase can see "tests failed
   on May 12 at 14:30" in a committed YAML file.

## Non-Goals

- Not a CI system — this does not replace GitHub Actions or other CI.
- Not a test runner — this records results produced by `go test ./...`.
- Not a flaky test detector — that's already handled by the BUG-entity convention.
- Not a coverage tracker — test governance is about pass/fail, not coverage.

## Performance Baseline

Measured on the Kanbanzai codebase itself:

| Scenario | Time |
|----------|------|
| Cached run (no code changes) | ~1.5 s |
| Uncached fresh run | ~37 s |
| With `-race` flag | ~68 s |

The 37-second uncached cost means re-running is always acceptable in a
development workflow. The 1.5-second cached cost is essentially free for
agents on every status check.

## Design

### Record Format: `.kbz/state/test-status.yaml`

A single YAML file, committed to git, updated after every test run.

```yaml
# .kbz/state/test-status.yaml
last_run: "2026-05-12T12:30:00Z"        # when the test run completed
result: pass                              # pass or fail
summary: ""                               # human-readable summary (empty on pass)
failures:                                 # list of failing tests (empty on pass)
  - package: "./internal/service"
    test: TestWorkQueue_Something
    message: "expected X, got Y"
runner: "agent"                           # who ran the tests
trigger: "post-merge"                     # what triggered the run
```

When `result` is `pass`, the `failures` list MUST be empty and `summary` MUST be
empty string. When `result` is `fail`, at least the `summary` field MUST be
populated with a short description (truncated to first 500 chars of the failure
output).

### Lifecycle

1. **Record created** when the project is initialised (`kanbanzai init`) — seeded
   with `last_run: null`, `result: unknown`.

2. **Record updated** by:
   - A new MCP tool `test(action: "run")` that runs `go test ./...`,
     parses the output, and writes the result.
   - The pre-commit hook — writes a new record after each hook run.
   - A `test(action: "verify")` tool that does a fast staleness check:
     if any `.go` source file has a modification time newer than `last_run`,
     it re-runs the tests. Otherwise it returns the cached record.

3. **Record consulted by**:
   - The `status` MCP tool — project overview includes a `test_health` block
     showing `pass`, `fail`, or `unknown` with `last_run` timestamp and a
     staleness indicator (sourced from the modtime check).
   - The `orchestrate-development` skill — Phase 0 checks the record and stops
     dispatch if `result` is `fail`.
   - The `orchestrate-review` skill — Post-Review Merge step checks the record
     before advancing.
   - The session-start checklist (`kanbanzai-getting-started`) — new step.

### Staleness Detection: Modtime, Not TTL

`test(action: "verify")` does NOT use a time-based TTL. Instead it uses a
filesystem-modification-time check:

1. Read `.kbz/state/test-status.yaml`. If missing → return `unknown`.
2. Scan all `.go` files in the repository (excluding `.worktrees/` and
   `vendor/`). Find the most recent modification time.
3. If `most_recent_modtime <= last_run` AND `result != fail` → return cached
   result. No re-run needed.
4. If `most_recent_modtime > last_run` OR `result == fail` → re-run
   `test(action: "run")` and return the fresh result.

**Rationale:** A TTL is semantically wrong — it would re-run when nobody
changed code, and skip re-running when someone did (if within the TTL window).
The modtime check is the correct semantic: "has anything changed since the
last test run?"

**Performance:** The `.go` file scan takes negligible time (a few milliseconds
on a project of this size). The 1.5-second cached test run means even full
re-runs are fast enough that agents never wait long.

### MCP Tool: `test`

```
test(action: "run")        # runs go test ./..., records result, returns summary
test(action: "verify")     # modtime staleness check; re-runs if code changed or
                           # last result was fail/unknown
test(action: "status")     # reads record without running any tests
test(action: "force-fail", summary: "...")  # manual override: mark tests as
                           # failing (for known breakage before CI confirms,
                           # so agents don't start new work)
```

### Integration with `status`

The project overview (`status()` with no ID) gains a new `test_health` block:

```json
"test_health": {
  "status": "pass",
  "last_run": "2026-05-12T12:30:00Z",
  "stale": false,
  "runner": "agent",
  "failure_count": 0,
  "summary": ""
}
```

The `stale` field indicates whether source files have been modified since
the last test run. When `stale` is `true` and the cached result is `pass`,
the record is still useful (tests were passing at last_run) but the agent
should re-run before merging.

When `result` is `fail` or `unknown`, a high-severity attention item is
appended to the attention list:

```json
{
  "type": "test_failure",
  "severity": "error",
  "entity_id": "main",
  "message": "Tests are failing on main (last checked 2h ago). No new work should start until fixed."
}
```

### Integration with Skill Procedures

#### `orchestrate-development` — Phase 0 (Session Start Audit)

Insert before Step 1:

```
0. Check test health: call test(action: "verify"). If result is fail, STOP.
   Report the failure summary to the human. Do not dispatch any sub-agents or
   create any worktrees. New implementation work must not start on a broken
   test suite.
```

#### `orchestrate-development` — Post-Merge step

The existing post-merge `go test ./...` should additionally update the
test-status record:

```
go build ./... && go test ./...   # capture exit code and output
test(action: "run")               # record the result (re-runs and updates cache)
```

#### `kanbanzai-getting-started` — Session Start Checklist

Insert after "Store check":

- [ ] **Test health check** — Call `test(action: "verify")`. If tests are
  failing on `main`, report to the human and STOP. Do not start new work.

#### `implement-task` — Phase 4 (Verify)

The existing "Run the full test suite. All tests must pass" step gains:

- [ ] After running tests, call `test(action: "run")` to update the cached
  record.

### Pre-commit Hook Update

The pre-commit hook currently runs `go test ./...` but does not write the
result record. After the test run, add:

```sh
# After running go test ./..., write the result record.
kanbanzai test record --result=$test_exit --output="$(echo "$test_output" | head -500)"
```

This keeps the record up-to-date even when CI is not configured. If the
`kanbanzai` binary is not in PATH, the hook degrades gracefully.

### Why a YAML File — Not Knowledge Entries

| Aspect | YAML Record | Knowledge Entry |
|--------|-------------|-----------------|
| Structure | Fixed schema, machine-readable | Flexible, curated text |
| Versioning | Committed to git, traceable | In knowledge store |
| Lookup | Single file read, O(1) | O(n) scan by topic |
| Modtime check | Single stat on the file | No mechanism |
| Overwrite | Single source of truth | Additive (multiple entries) |

A committed YAML file is the right fit: versioned, machine-readable, visible
to humans, and has no query overhead.

### Backward Compatibility

Projects without a `test-status.yaml` file default to `result: unknown` with a
`last_run: null`. The `status` tool shows `unknown` but does not produce a
blocking attention item — it shows a warning-level item suggesting the first
test run be executed. This ensures the feature does not break existing workflows
on upgrade.

## Edge Cases

### First Run / Fresh Clone
- No record file → `result: unknown`, `last_run: null`.
- `test(action: "verify")` returns `unknown` and suggests running
  `test(action: "run")`.
- Status shows `unknown` with a soft suggestion, not a hard block.

### Modtime Scan Scope
- The modtime check scans `.go` files recursively from the repository root.
- It MUST exclude `.worktrees/` (worktrees are independent git repos).
- It MUST exclude `vendor/` (vendored code is not under our test governance).
- It SHOULD exclude `.git/` and any other hidden directory at the root.

### Stale Pass → Still Informative
- If the record says `pass` but code has changed since, the `stale` field
  in the status response warns the agent. This is informational, not a block:
  tests were passing at last_run, which is better than unknown.

### Concurrent Test Runs
- The test-status file is a single file. Concurrent writes may race.
- **Solution:** The `test record` subcommand uses an atomic write
  (write to `.tmp`, then `os.Rename`). If two processes write simultaneously,
  one wins — the result is still a valid record. No corruption possible.

### CI Integration (Future)
- The record can be updated by a CI post-job step:
  ```yaml
  - run: kanbanzai test record --result=${{ job.status }} --output="..."
  ```
- This is explicitly **future work** and scoped out of this design.

## Alternatives Considered

1. **Time-based TTL** — Rejected. A TTL would re-run tests when no code
   changed (wasteful), and skip re-running when code did change (risky).
   The modtime check is semantically correct.

2. **GitHub Commit Status API** — Rejected. Authoritative but requires
   network access, a token, and doesn't work offline. The system must
   work without CI.

3. **Knowledge store** — Rejected. Knowledge entries are qualitative, not
   machine-readable. No modtime or staleness mechanism.

4. **Bare `go test ./...` on every status call** — Rejected as unnecessary.
   The 37-second uncached cost adds up if agents poll status frequently.
   The modtime check avoids unnecessary re-runs while ensuring correctness.
