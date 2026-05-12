# Specification: Test Governance Framework

## Overview

Establish a persistent, committed test-status record that AI agents check before
starting any implementation work, and that is surfaced in every `status()` call.
The record is a single YAML file (`.kbz/state/test-status.yaml`) updated after
every test run and consulted before any new work begins. Staleness is detected
by comparing source-file modification times against the last test run — not by
a clock-based TTL.

## Scope

### In Scope

1. `.kbz/state/test-status.yaml` — record format and lifecycle
2. `test` MCP tool with actions: `run`, `verify`, `status`, `force-fail`
3. Integration: `status` project overview includes `test_health` block
4. Integration: attention item when tests are failing
5. Skill updates: `orchestrate-development`, `orchestrate-review`, `implement-task`,
   `kanbanzai-getting-started`, `verify-closeout`
6. AGENTS.md updates to the Test Discipline section
7. Pre-commit hook updates to write the record
8. Seed record creation during `kanbanzai init`

### Out of Scope

1. CI integration (GitHub Actions, etc.) — future work
2. Flaky test detection — already handled by BUG-entity convention
3. Test coverage metrics — test governance is about pass/fail, not coverage
4. Race-detector integration — agents run `go test ./...` (not `-race`) as their
   standard check; the race detector adds ~30s and is checked separately during
   review and verification stages

## Functional Requirements

### FR-001: Record Format

`.kbz/state/test-status.yaml` MUST contain the following fields:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `last_run` | ISO 8601 or null | yes | When the test run completed |
| `result` | string | yes | One of: `pass`, `fail`, `unknown` |
| `summary` | string | yes | Human-readable failure summary; empty on pass |
| `failures` | array | yes | List of failing test details; empty on pass |
| `runner` | string | no | Who/what ran the tests: `agent`, `human`, `hook`, `ci` |
| `trigger` | string | no | What triggered the run: `post-merge`, `manual`, `startup-check` |

Each entry in `failures`:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `package` | string | yes | Go package path (e.g. `./internal/service`) |
| `test` | string | yes | Test function name |
| `message` | string | yes | Failure message (first line) |

### FR-002: Record Seeding

On `kanbanzai init`, a seed record is created:

```yaml
last_run: null
result: unknown
summary: ""
failures: []
```

### FR-003: Test Runner Tool — `test(action: "run")`

- Runs `go test ./...` with the same exit-code semantics as `go test ./...`.
- Parses the output to extract failing test names, packages, and messages.
- Writes `.kbz/state/test-status.yaml` with the result.
- Returns a structured response: `{result, last_run, failure_count, summary}`.

### FR-004: Test Verify Tool — `test(action: "verify")`

- Reads `.kbz/state/test-status.yaml`.
- If file does not exist → return `unknown` with suggestion to run
  `test(action: "run")`.
- Performs a modtime staleness check:
  1. Scan all `.go` files in the repository root, excluding `.worktrees/`,
     `vendor/`, `.git/`, and any other root-level hidden directory.
  2. Find the most recent modification time among them.
  3. If `most_recent_modtime <= last_run` AND `result != fail` → return cached
     result. **No re-run.**
  4. Otherwise → run `test(action: "run")` and return the fresh result.
- Purpose: avoids unnecessary test re-runs when no source code has changed,
  while guaranteeing correctness when code has changed.

### FR-005: Test Status Tool — `test(action: "status")`

- Reads `.kbz/state/test-status.yaml` without running any tests.
- Returns the record as-is, or `result: unknown` if file does not exist.

### FR-006: Force-Fail Tool — `test(action: "force-fail")`

- Accepts `summary` (string) parameter.
- Writes `.kbz/state/test-status.yaml` with `result: fail`, `last_run: now`,
  `runner: agent`, `trigger: manual`, `summary: <provided summary>`.
- Purpose: allows a human to mark tests as failing before CI confirms, so
  agents do not start new work in the meantime.

### FR-007: Status Integration — Project Overview

`status()` (no ID) project overview response MUST include:

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

- `stale` is computed by the same modtime check as `verify`. When `true`, it
  means source files have changed since the last test run. The record is still
  meaningful (tests were passing at last_run) but should be refreshed before
  merging.
- `status` is one of: `pass`, `fail`, `unknown`.

### FR-008: Status Integration — Attention Items

When `result` is `fail`:

| Type | Severity | EntityID | Message |
|------|----------|----------|---------|
| `test_failure` | `error` | `main` | "Tests are failing on main (last checked {duration} ago). No new work should start until fixed." |

When `result` is `unknown`:

| Type | Severity | EntityID | Message |
|------|----------|----------|---------|
| `test_failure` | `warning` | `main` | "Test health unknown — no record of a test run. Run `test(action: "run")` to establish a baseline." |

### FR-009: Orchestrate-Development — Pre-Dispatch Gate

The `orchestrate-development` skill Phase 0 gets a new Step 0:

"Check test health: call `test(action: "verify")`. If result is `fail`, STOP.
Report the failure summary to the human. Do not dispatch any sub-agents or
create any worktrees. New implementation work must not start on a broken
test suite."

This is a **hard constraint (ℋ)** — violation blocks stage advance.

### FR-010: Kanbanzai-Getting-Started — Session Start Gate

The session start checklist gets a new item after "Store check":

"- [ ] **Test health check** — Call `test(action: "verify")`. If tests are
failing on `main`, report to the human and STOP. Do not start new work."

### FR-011: Implement-Task — Post-Phase-4 Update

After running `go test ./...` in Phase 4 (Verify), the implementer MUST call
`test(action: "run")` to update the cached record. This ensures the record is
fresh after any implementation work.

### FR-012: Orchestrate-Review — Post-Merge Update

After the post-merge `go test ./...` step, the orchestrator MUST call
`test(action: "run")` to update the cached record with the fresh result.

### FR-013: Verify-Closeout — DoD Item 4 Update

Item 4 (Builds and Tests Pass) MUST reference the test-status record: after
running `go test ./...`, call `test(action: "run")` to update the cached
result.

### FR-014: Pre-Commit Hook Update

After the existing `go test ./...` run in `.githooks/pre-commit`, write the
result record:

```sh
kanbanzai test record --result=$test_exit --output="$(echo "$test_output" | head -500)"
```

If the `kanbanzai` binary is not available in PATH, the hook degrades
gracefully — the record simply won't be updated, and the next explicit test
run will fix it.

### FR-015: AGENTS.md Test Discipline Update

The "No failing tests on main" section gains a new rule:

"- The `test(action: "verify")` tool MUST be checked before any new
   implementation starts. An orchestrator that begins work when tests are
   failing has violated the Definition of Done."

The "Reporting test failures" section gains:

"- After fixing the test(s), update the test-status record:
   `test(action: "run")` or `kanbanzai test record --result=0`"

### FR-016: Test-Status Record Is Always Updated After a Test Run

Every code path that runs `go test ./...` MUST also update the test-status
record via `test(action: "run")` or the equivalent CLI call. This is not
optional — the record is the single source of truth for test health.

Specific enforced points:
- Pre-commit hook (FR-014)
- Implement-task Phase 4 (FR-011)
- Orchestrate-review post-merge (FR-012)
- Orchestrate-development post-merge (FR-009)
- Verify-closeout DoD Item 4 (FR-013)

### FR-017: Merging Is Blocked When Tests Are Failing

The merge gate MUST reject a merge attempt when `test(action: "verify")`
returns `result: fail`. This is enforced at the merging stage gate.

## Non-Functional Requirements

### NFR-001: Read Performance

`test(action: "status")` MUST complete in under 10ms (single file read, no
test execution, no modtime scan).

### NFR-002: Verify Performance

`test(action: "verify")` when cached MUST complete in under 100ms (file read +
`.go` file scan). The `.go` scan is a filepath.Walk limited to a single
glob pattern.

### NFR-003: Write Safety

The `test record` subcommand MUST use atomic write (write to `.tmp`, then
`os.Rename`). This prevents partial writes from concurrent test runs.

### NFR-004: Backward Compatibility

Projects without `.kbz/state/test-status.yaml` MUST NOT error. The default is
`result: unknown`, which produces a warning attention item but does not block
any operation. This ensures existing projects upgraded to a new version of
kanbanzai are not broken.

## Acceptance Criteria

### AC-001: Record Creation
1. Run `kanbanzai init` in a fresh directory.
2. `.kbz/state/test-status.yaml` exists with `result: unknown`.

### AC-002: Test Run Recording
1. Run `go test ./...` — assume it passes.
2. Run `test(action: "run")`.
3. `.kbz/state/test-status.yaml` has `result: pass`, `last_run` populated.

### AC-003: Status Surface
1. After a passing test run, call `status()` (no ID).
2. Response includes `test_health.status` = `pass` and `stale` is computed.

### AC-004: Failure Attention
1. Set result to fail (`test(action: "force-fail", summary: "something broke")`).
2. Call `status()` (no ID).
3. Attention items includes `test_failure` with severity `error`.

### AC-005: Verify Returns Cached When Code Unchanged
1. Write record with `result: pass`, `last_run: now - 1h`.
2. Ensure no `.go` files have been modified since `last_run`.
3. Call `test(action: "verify")`.
4. Returns cached result without re-running tests.

### AC-006: Verify Re-Runs When Code Changed
1. Write record with `result: pass`, `last_run: now - 1h`.
2. Touch a `.go` file (`touch somefile.go`).
3. Call `test(action: "verify")`.
4. Re-runs tests and returns fresh result.

### AC-007: Verify Re-Runs When Last Result Was Fail
1. Write record with `result: fail`, `last_run: now`.
2. Touch no files.
3. Call `test(action: "verify")`.
4. Re-runs tests (because `result == fail` overrides the modtime check).

### AC-008: Skill Gate Blocks Dispatch
When test health is `fail`, the orchestrator must not dispatch sub-agents.
Verified by skill procedure review.

### AC-009: Merge Gate Blocks on Failure
When test health is `fail`, the merge stage gate returns a blocking error.
Verified by merge integration test.
