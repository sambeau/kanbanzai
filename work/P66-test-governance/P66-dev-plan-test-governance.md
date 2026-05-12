# Dev Plan: Test Governance Framework

## Overview

Implement a persistent test-status record (`.kbz/state/test-status.yaml`) that AI
agents consult before starting new work, and that is surfaced in the `status()`
project overview. This prevents agents from beginning implementation when the test
suite is broken. Staleness is detected by `.go` source-file modification times,
not by a time-based TTL.

## Task Breakdown

### Task 1: Test-Status Record Store

**Scope:** Create the Go package and YAML record store.

**Files:**
- `internal/teststatus/store.go` — read, write, atomic update, modtime staleness check
- `internal/teststatus/store_test.go` — tests
- `internal/teststatus/types.go` — `TestStatus` struct, failure entry type

**Key logic:**
- `WriteRecord(root, record)` atomically writes `.kbz/state/test-status.yaml`.
- `ReadRecord(root)` parses the file and returns the struct.
- `IsStale(root, record)` — scans `.go` files (excluding `.worktrees/`, `vendor/`,
  hidden dirs) and returns true if any file's modtime > record.last_run.
- Missing file returns zero-value with `result: unknown`.

**Acceptance Criteria:**
- `WriteRecord` + `ReadRecord` round-trip correctly.
- `IsStale` returns true after touching a `.go` file.
- `IsStale` returns false when no `.go` files have changed.
- Missing file returns `result: unknown`.
- `go test ./internal/teststatus/` passes.

### Task 2: CLI Subcommand `kanbanzai test`

**Scope:** Wire up the `teststatus` package to a `kanbanzai test` CLI subcommand.

**Files:**
- `cmd/kbz/test_cmd.go` — `runTest` function with subcommands
- `cmd/kbz/test_cmd_test.go` — tests for CLI parsing

**Subcommands:**
- `kanbanzai test status` — prints current record (JSON)
- `kanbanzai test record --result=<exit-code> --output=<test-output>` — writes record
- `kanbanzai test run` — runs `go test ./...`, captures output, writes record
- `kanbanzai test verify` — modtime staleness check; re-runs if stale or fail/unknown
- `kanbanzai test force-fail --summary="reason"` — manual override

**Acceptance Criteria:**
- `kanbanzai test status` prints JSON.
- `kanbanzai test record --result=0` writes `result: pass`.
- `kanbanzai test record --result=1 --output="FAIL TestFoo"` writes `result: fail`.
- `kanbanzai test run` executes `go test ./...` and records result.
- All commands handle missing file gracefully (returning `unknown`).

### Task 3: MCP Tool `test`

**Scope:** Expose the test-status record as an MCP tool.

**Files:**
- `internal/mcp/test_tool.go` — MCP tool registration and handler
- `internal/mcp/test_tool_test.go` — MCP tool tests
- `internal/mcp/server.go` — register tool in core group

**Tool schema:**
```
test(action: "run")        → runs tests, records result
test(action: "verify")     → modtime staleness check; re-runs if stale or
                             last result was fail/unknown
test(action: "status")     → reads record without running tests
test(action: "force-fail", summary: "...") → manual override
```

The handler delegates to the `teststatus` store package for all read/write
operations. The `verify` action runs `IsStale` and, if stale or `result == fail`,
runs `test(action: "run")` (which itself runs `go test ./...`).

**Acceptance Criteria:**
- Tool is registered in the `core` tool group.
- `test(action: "status")` returns `{result, last_run, failure_count, summary}`.
- `test(action: "verify")` returns cached result when code is unchanged and
  result was pass.
- `test(action: "verify")` re-runs when code has changed.
- `test(action: "verify")` re-runs when last result was fail.
- `test(action: "run")` runs `go test ./...`, records result, returns summary.
- Missing file returns `result: unknown`.

### Task 4: Status Integration

**Scope:** Add `test_health` block to the status project overview.

**Files:**
- `internal/mcp/status_tool.go` — add `test_health` to `projectOverview` struct,
  populate in `synthesiseProject`, add attention item for fail/unknown.
- `internal/mcp/status_tool_test.go` — existing tests updated.

The `test_health` block is populated by reading `.kbz/state/test-status.yaml`
and running the same modtime scan for the `stale` field. It does NOT trigger a
test re-run — `status` is a read-only dashboard. If the agent needs a fresh
result, they call `test(action: "verify")` separately.

**Acceptance Criteria:**
- `status()` response includes `test_health` block with `status`, `last_run`,
  `stale`, `runner`, `failure_count`, `summary`.
- When `result` is `fail`, attention list includes `test_failure` error item.
- When `result` is `unknown`, attention list includes `test_failure` warning item.
- When `result` is `pass`, no test-failure attention item.

### Task 5: Skill and Documentation Updates

**Scope:** Update all skill files and AGENTS.md to add the test-health gate.
Modify 6 files (5 skill docs + AGENTS.md).

**Files to modify:**
- `AGENTS.md` — update Test Discipline section (add verify-before-work rule,
  post-fix record update, number the rules explicitly)
- `.kbz/skills/orchestrate-development/SKILL.md` — Phase 0 gate + post-merge update
- `.kbz/skills/orchestrate-review/SKILL.md` — Post-Merge step uses
  `test(action: "run")` after `go test ./...`
- `.kbz/skills/implement-task/SKILL.md` — Phase 4 gains `test(action: "run")`
- `.kbz/skills/verify-closeout/SKILL.md` — DoD Item 4 references the record
- `.agents/skills/kanbanzai-getting-started/SKILL.md` — Session Start checklist
  gains test health check step

**All seven rules (codified as AGENTS.md additions):**
1. ALL tests must be maintained — no test may be disabled without a decision record.
2. NO test should ever be ignored — every test failure must be investigated.
3. NO code should be merged until all tests pass — the merge gate enforces this.
4. ALL tests must pass after merge to meet the Definition of Done — DoD Item 4.
5. There is no such thing as "just a flaky test" — file a BUG entity.
6. Pre-existing failing tests must not be ignored — they appear as `test_failure`
   attention items at error severity in every `status()` call.
7. The orchestrator must bail before new work starts if tests are failing —
   `test(action: "verify")` in Phase 0 enforces this.

**Acceptance Criteria:**
- Every skill file mentions `test(action: "verify")` or `test(action: "run")`.
- Orchestrate-development Phase 0 includes the pre-dispatch test-health gate.
- Getting-started checklist includes the test-health check step.
- AGENTS.md contains the 7 explicit rules as a numbered list.

### Task 6: Pre-Commit Hook Update

**Scope:** Update `.githooks/pre-commit` to write the test-status record after
running tests.

**Files:**
- `.githooks/pre-commit` — after the `go test ./...` block, add:
  ```sh
  kanbanzai test record --result=$test_exit --output="$(echo "$test_output" | head -500)"
  ```

**Acceptance Criteria:**
- After pre-commit hook runs, `.kbz/state/test-status.yaml` is updated.
- Graceful degradation if `kanbanzai` binary is not in PATH.

### Task 7: Init Command Seed

**Scope:** Update `kanbanzai init` to create the seed test-status record.

**Files:**
- `internal/kbzinit/` — add seed record creation alongside other init artefacts
- Test files for init

**Acceptance Criteria:**
- After `kanbanzai init`, `.kbz/state/test-status.yaml` exists with
  `result: unknown`, `last_run: null`.
- Existing tests for init still pass.

## Interface Contracts

### `internal/teststatus` package (Task 1)

```go
package teststatus

type Result string

const (
    ResultPass    Result = "pass"
    ResultFail    Result = "fail"
    ResultUnknown Result = "unknown"
)

type Failure struct {
    Package string `yaml:"package"`
    Test    string `yaml:"test"`
    Message string `yaml:"message"`
}

type Record struct {
    LastRun  *time.Time `yaml:"last_run"`           // nil when unknown
    Result   Result     `yaml:"result"`
    Summary  string     `yaml:"summary"`
    Failures []Failure  `yaml:"failures"`
    Runner   string     `yaml:"runner,omitempty"`   // agent | human | hook | ci
    Trigger  string     `yaml:"trigger,omitempty"`  // post-merge | manual | startup-check
}

// ReadRecord parses .kbz/state/test-status.yaml. Returns a zero-value record
// with Result=ResultUnknown when the file does not exist.
func ReadRecord(repoRoot string) (Record, error)

// WriteRecord atomically writes the record (write to .tmp, then os.Rename).
func WriteRecord(repoRoot string, rec Record) error

// IsStale returns true if any .go file under repoRoot has a modtime newer than
// rec.LastRun, or if rec.LastRun is nil. Excludes .worktrees/, vendor/, .git/,
// and any other top-level hidden directory.
func IsStale(repoRoot string, rec Record) (bool, error)
```

### `kanbanzai test` CLI (Task 2)

```
kanbanzai test status
  → prints the current record as JSON to stdout.
  → exit 0 always (status is informational).

kanbanzai test record --result=<exit-code> [--output=<test-output>]
  → writes the record. exit-code 0 → result: pass; non-zero → result: fail.
  → --output is parsed for failing test names/packages when provided.
  → exit 0 on successful write.

kanbanzai test run
  → executes `go test ./...`, captures stdout+stderr, writes the record.
  → exit code mirrors `go test`.

kanbanzai test verify
  → modtime staleness check. Re-runs `kanbanzai test run` if stale or
    last result was fail/unknown; otherwise returns the cached record.
  → exit 0 on cached pass; mirrors `go test` exit on a re-run.

kanbanzai test force-fail --summary="reason"
  → writes result: fail, runner: agent, trigger: manual.
  → exit 0 on successful write.
```

### MCP `test` tool (Task 3)

```
test(action: "run")
  → { result: "pass"|"fail", last_run: ISO8601, failure_count: int, summary: string }

test(action: "verify")
  → { result, last_run, failure_count, summary, was_rerun: bool }

test(action: "status")
  → { result, last_run, stale: bool, runner, failure_count, summary }
  → does NOT run tests; pure read.

test(action: "force-fail", summary: string)
  → { result: "fail", last_run: ISO8601 }
```

### `status` tool extension (Task 4)

```
status()  // no ID
  → response.test_health = {
      status: "pass"|"fail"|"unknown",
      last_run: ISO8601 | null,
      stale: bool,
      runner: string,
      failure_count: int,
      summary: string,
    }
  → response.attention may include {
      type: "test_failure",
      severity: "error"|"warning",
      entity_id: "main",
      message: string,
    }
```

## Dependency Graph

```
Task 1 (Store) ──→ Task 2 (CLI) ──→ Task 3 (MCP Tool)
                   Task 1 ──→ Task 4 (Status Integration)
                   Task 3 ──→ Task 5 (Skill Updates)
                   Task 2 ──→ Task 6 (Pre-Commit Hook)
                   Task 1 ──→ Task 7 (Init Seed)
```

Task 1 is the foundational dependency. Tasks 2 and 7 depend on it directly.
Tasks 3 and 4 depend on the store via the MCP tool. Tasks 5 and 6 are
documentation/script changes that depend on the tool being available.

Tasks 5 and 6 can be parallelised after Task 2 is complete.

## Traceability Matrix

| Requirement | Task | Verification |
|-------------|------|-------------|
| FR-001 (Record Format) | 1 | Test: store round-trip |
| FR-002 (Record Seeding) | 7 | Test: init produces file |
| FR-003 (Test Run Tool) | 2, 3 | Test: MCP tool handler |
| FR-004 (Verify Tool) | 2, 3 | Test: modtime logic |
| FR-005 (Status Tool) | 3 | Test: read without run |
| FR-006 (Force Fail) | 2, 3 | Test: manual override |
| FR-007 (Status Integration) | 4 | Test: status output |
| FR-008 (Attention Items) | 4 | Test: attention list |
| FR-009 (Gate: Orchestrate) | 5 | Review: skill doc |
| FR-010 (Gate: Getting Started) | 5 | Review: skill doc |
| FR-011 (Update: Implement) | 5 | Review: skill doc |
| FR-012 (Update: Review) | 5 | Review: skill doc |
| FR-013 (Update: Verify-Closeout) | 5 | Review: skill doc |
| FR-014 (Pre-Commit Hook) | 6 | Manual: hook runs |
| FR-015 (AGENTS.md Update) | 5 | Review: doc |
| FR-016 (Always Update Record) | 5, 6 | Review: all test-run paths covered |
| FR-017 (Merge Gate) | 3 | Integration: merge tool checks |
| NFR-001 (Read Performance) | 1, 2 | Design: single file read |
| NFR-002 (Verify Performance) | 1 | Design: modtime scan < 100ms |
| NFR-003 (Write Safety) | 1 | Design: atomic write |
| NFR-004 (Backward Compat) | 1, 4 | Test: missing file |
