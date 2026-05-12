# Dev Plan: Test Governance Framework

## Overview

Implement a persistent test-status record (`.kbz/state/test-status.yaml`) that AI
agents consult before starting new work, and that is surfaced in the `status()`
project overview. This prevents agents from beginning implementation when the test
suite is broken.

## Task Breakdown

### Task 1: Test-Status Record Store

**Scope:** Create the Go package and YAML record store.

**Files:**
- `internal/teststatus/store.go` — read, write, atomic update
- `internal/teststatus/store_test.go` — tests
- `internal/teststatus/types.go` — `TestStatus` struct, failure entry type

**Acceptance Criteria:**
- `WriteRecord(root, record)` atomically writes `.kbz/state/test-status.yaml`.
- `ReadRecord(root)` parses the file and returns the struct.
- Missing file returns zero-value with `result: unknown`.
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
- `kanbanzai test verify` — cached check with TTL; re-runs if stale
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
test(action: "verify")     → cached check with TTL; re-runs if stale
test(action: "status")     → reads record without running tests
test(action: "force-fail", summary: "...") → manual override
```

**Acceptance Criteria:**
- Tool is registered in the `core` tool group.
- `test(action: "status")` returns `{result, last_run, failure_count, summary}`.
- `test(action: "verify")` re-runs if record is stale or unknown.
- `test(action: "run")` runs `go test ./...`, records result, returns summary.
- Missing file returns `result: unknown`.

### Task 4: Status Integration

**Scope:** Add `test_health` block to the status project overview.

**Files:**
- `internal/mcp/status_tool.go` — add `test_health` to `projectOverview` struct,
  populate in `synthesiseProject`, add attention item for fail/unknown.
- `internal/mcp/status_tool_test.go` — existing tests updated.

**Acceptance Criteria:**
- `status()` response includes `test_health` block with `status`, `last_run`,
  `last_run_relative`, `failure_count`, `summary`.
- When `result` is `fail`, attention list includes `test_failure` error item.
- When `result` is `unknown`, attention list includes `test_failure` warning item.
- When `result` is `pass`, no test-failure attention item.

### Task 5: Skill and Documentation Updates

**Scope:** Update all skill files and AGENTS.md to add the test-health gate.

**Files to modify:**
- `AGENTS.md` — update Test Discipline section
- `.kbz/skills/orchestrate-development/SKILL.md` — Phase 0 gate
- `.kbz/skills/orchestrate-review/SKILL.md` — Post-Merge step
- `.kbz/skills/implement-task/SKILL.md` — Phase 4 update
- `.kbz/skills/verify-closeout/SKILL.md` — DoD Item 4
- `.agents/skills/kanbanzai-getting-started/SKILL.md` — Session Start checklist

**Acceptance Criteria:**
- Every skill file mentions `test(action: "verify")` or `test(action: "run")`.
- Orchestrate-development Phase 0 includes the pre-dispatch test-health gate.
- Getting-started checklist includes the test-health check step.
- AGENTS.md rules reference the test-status record.

### Task 6: Pre-Commit Hook Update

**Scope:** Update `.githooks/pre-commit` to write the test-status record after
running tests.

**Files:**
- `.githooks/pre-commit` — add `kanbanzai test record --result=$test_exit` call

**Acceptance Criteria:**
- After pre-commit hook runs, `.kbz/state/test-status.yaml` is updated.
- Graceful degradation if `kanbanzai` binary is not in PATH.

### Task 7: Init Command Seed

**Scope:** Update `kanbanzai init` to create the seed test-status record.

**Files:**
- `internal/kbzinit/` — add seed record creation
- Test files for init

**Acceptance Criteria:**
- After `kanbanzai init`, `.kbz/state/test-status.yaml` exists with `result: unknown`.
- Existing tests for init still pass.

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
| FR-004 (Verify Tool) | 2, 3 | Test: TTL logic |
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
| NFR-001 (Read Performance) | 1, 2 | Design: single file read |
| NFR-002 (Write Safety) | 1 | Design: atomic write |
| NFR-003 (Backward Compat) | 1, 4 | Test: missing file |
| NFR-004 (Config TTL) | 2 | Test: config loading |
