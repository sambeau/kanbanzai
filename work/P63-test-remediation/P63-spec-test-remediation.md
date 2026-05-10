# P63 Specification — Test Remediation: Fix Failing Tests and Harden Definition of Done

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-10                     |
| Status | approved |
| Author | spec-author                    |

---

## Overview

This specification implements the design described in `work/P63-test-remediation/P63-design-test-remediation.md` (P63-test-remediation/design-p63-design-test-remediation, approved).

`go test ./...` on `main` reports **111 failing tests** across 3 packages: `internal/kbzinit` (3), `internal/mcp` (3), and `internal/service` (105). Twenty-nine other packages pass. The failing tests represent a Definition of Done violation — "all tests pass on main" is a non-negotiable tenet — and indicate that Kanbanzai is in a broken state unknown to its developers.

The root causes are well-understood: a plan→batch type migration missed test helpers (~95 tests), a nil pointer in a bug gate function (1 test), embedded seed drift (3 tests), and stale MCP regression contracts (3 tests). The fixes are small (~30 lines across ~5 files), but the systemic problem is larger: there are no enforcement mechanisms to detect or prevent test failures from reaching `main`.

This specification covers:
- **Immediate remediation:** fix all 111 failing tests
- **Test infrastructure hardening:** shared helpers, correct type usage, pruning of dead tests
- **Enforcement:** pre-commit hooks, merge gate test verification, `kbz doctor` diagnostics, and `health` MCP integration — all within Kanbanzai's agentic workflow model (no external CI)
- **DoD and cultural changes:** updated policy documents, AGENTS.md rules, and dashboard visibility

## Scope

This specification covers the remediation of 111 failing tests, test infrastructure hardening, enforcement mechanisms (pre-commit hooks, merge gates, kbz doctor, health MCP), and Definition of Done updates. All enforcement lives within Kanbanzai's own tooling — no external CI.

### In Scope

- Fixing all 111 failing tests across `internal/kbzinit`, `internal/mcp`, and `internal/service`
- Consolidating duplicated test helpers into shared infrastructure
- Pruning tests for removed features or duplicate coverage
- Pre-commit hook that blocks commits with failing tests
- Merge gate test verification via `merge(action: check)`
- `kbz doctor` test suite diagnostics
- `health` MCP test status dashboard
- DoD and AGENTS.md policy updates

### Out of Scope

- Adding net-new test coverage beyond what is needed to fix the failing tests
- Re-architecting the test framework or migrating to a different testing library
- Changing the Go testing runtime or test runner
- External CI/CD pipeline configuration (GitHub Actions, etc.) — enforcement lives in Kanbanzai's own tooling
- Adding test coverage for untested code paths

---

## Functional Requirements

**Phase 1 — Fixes:**

- **REQ-001:** `planEntityTypeFromID` and all equivalent test helpers must use `model.EntityKindPlan` (value `"batch"`) for all plan and batch entity types, so that `CreateFeature` can resolve parent references regardless of `P` or `B` prefix.
- **REQ-002:** `checkBugWorktreeHasCommits` in `internal/service/prereq.go` must handle a nil `*DocumentService` gracefully — returning a descriptive reason rather than panicking.
- **REQ-003:** Embedded seed files in `internal/kbzinit/skills/`, `internal/kbzinit/skills/task-execution/`, and `internal/kbzinit/roles/` must match their on-disk counterparts in `.agents/skills/` and `.kbz/` respectively, as verified by the existing consistency tests.
- **REQ-004:** The canonical field list in `TestNextContextToMap_FieldNamesMatchCanonicalList` must include `workflow_state_warning`.
- **REQ-005:** The `handoff` MCP tool error response must include a `message` field conforming to the contract validated by `TestHandoff_ErrorResponseShapePreserved`.
- **REQ-006:** The `next` and `worktree` MCP tool descriptions must each fit within the 250-token budget enforced by `TestToolDescriptions_TokenBudget`, either by trimming prose or by raising the budget with documented justification.
- **REQ-007:** After all fixes are applied, `go test ./...` must exit 0 with zero test failures.

**Phase 2 — Test Infrastructure:**

- **REQ-008:** Duplicated `writeTestPlan` helpers across test files (`entities_test.go`, `display_id_test.go`, `conflict_test.go`, `decompose_test.go`, `dispatch_test.go`, `doc_path_test.go`) must be consolidated into a single shared helper that uses `model.EntityKindPlan` for the entity type.
- **REQ-009:** The `planEntityTypeFromID` function and equivalent `if id[0] == 'B'` checks must be replaced with direct use of `model.EntityKindPlan` wherever the intent is to write a plan or batch entity.
- **REQ-010:** Tests that duplicate coverage of the same behaviour in multiple packages must be identified. Where both tests verify the same invariant, one must be removed or the coverage consolidated.
- **REQ-011:** Tests that exercise behaviour no longer present in the codebase (e.g., removed features, deprecated lifecycle states) must be identified and removed.

**Phase 3 — Enforcement:**

- **REQ-012:** A pre-commit hook must run `go test ./...` (or a scoped subset for the changed packages) and block the commit if any test fails.
- **REQ-013:** The pre-commit hook must be installable via `kbz init` and `make setup`, and must be documented in AGENTS.md.
- **REQ-014:** The pre-commit hook must be bypassable with `--no-verify` for emergency situations, emitting a prominent warning when bypassed.
- **REQ-015:** `merge(action: check)` must run `go test ./...` on the feature branch and report failures as a blocking gate.
- **REQ-016:** `kbz doctor` must include a check that runs `go test ./...` and reports the pass/fail status of each package.
- **REQ-017:** The `health` MCP tool output must include a summary of test suite status: total packages, passing packages, failing packages, and the list of specific failing test names.

**Phase 4 — Policy and Culture:**

- **REQ-018:** The Definition of Done must be updated to state: "All tests pass on `main` at all times. Any commit that introduces a test failure is a DoD violation."
- **REQ-019:** The Definition of Done must be updated to state: "Any developer who observes a failing test on `main` is responsible for reporting it immediately. 'Not my package' is not an acceptable response."
- **REQ-020:** AGENTS.md must document the "no failing tests on main" rule, the pre-commit hook, and the procedure for reporting test failures.
- **REQ-021:** The `status` dashboard must display test suite health as a top-level metric alongside entity counts and health errors/warnings.
- **REQ-022:** If a test is intentionally removed, the commit message must explain why the test was removed and reference the requirement or decision that made it obsolete.

## Non-Functional Requirements

- **REQ-NF-001:** The pre-commit hook must complete within 5 seconds for a no-op run (cached results, no changed Go files). A full `go test ./...` run from cold must complete within 90 seconds.
- **REQ-NF-002:** `kbz doctor` test check must complete within the same time as `go test ./...` — it must not add measurable overhead beyond running the test suite.
- **REQ-NF-003:** `merge(action: check)` test verification must not add more than 90 seconds to the merge gate evaluation.
- **REQ-NF-004:** The `health` MCP tool test summary must be computed from cached results where available — it must not run the full test suite on every `health` call.

---

## Constraints

- **No external CI:** Kanbanzai does not use GitHub Actions or external CI pipelines. All enforcement must use Kanbanzai's own MCP tools, CLI, and Git hooks.
- **Pre-existing test failures on main:** The current 111 failures on `main` are a known state. The pre-commit hook must not block work on this plan itself — the hook must be configurable or the plan must be implemented in a worktree where the hook can be temporarily suppressed.
- **Backward compatibility:** Fixes must not change production behaviour. This is test-only and enforcement-only work. No MCP tool contracts, API signatures, or file formats may change.
- **`go test` dependency:** All enforcement mechanisms depend on `go test` being available in the environment. `kbz doctor` must report if `go` is not found.
- **Scope boundary:** This specification does NOT cover adding new test coverage for untested code paths. It covers fixing existing failing tests, preventing regression, and enforcing test hygiene.
- **Token budgets:** The 250-token MCP description budget is a design constraint inherited from P43 fast-track architecture. Any change to the budget must be documented as an architecture decision.
- **Design constraint — no CI:** The design's Phase 3 originally referenced a GitHub Actions CI job. Per design approval caveat, this is replaced with Kanbanzai-native enforcement: pre-commit hooks, merge gate integration, `kbz doctor`, and `health` MCP dashboard.

---

## Acceptance Criteria

### Phase 1 — Fixes

- **AC-001 (REQ-001):** Given a test that calls `CreateFeature` with a P-prefixed parent plan ID, when `writeTestPlan` writes the parent with type `"batch"`, then `CreateFeature` succeeds and returns a valid feature entity.
- **AC-002 (REQ-001):** Given all 105 previously-failing `internal/service` tests, when `go test ./internal/service/...` is run, then every test passes (zero failures).
- **AC-003 (REQ-002):** Given a nil `*DocumentService`, when `checkBugWorktreeHasCommits` is called, then it returns `(false, "document service not available")` without panicking.
- **AC-004 (REQ-002):** Given `TestCheckBugTransitionGate_UngatedTransitions`, when the test runs, then it passes without a segmentation violation.
- **AC-005 (REQ-003):** Given the embedded seed files in `internal/kbzinit/`, when the consistency tests run, then `TestEmbeddedSkillsMatchAgentSkills`, `TestEmbeddedTaskSkillsMatchProjectSkills`, and `TestEmbeddedRolesMatchProjectRoles` all pass.
- **AC-006 (REQ-004):** Given the `nextContextToMap` output containing `workflow_state_warning`, when `TestNextContextToMap_FieldNamesMatchCanonicalList` runs, then the field is in the canonical list and the test passes.
- **AC-007 (REQ-005):** Given a handoff error response, when `TestHandoff_ErrorResponseShapePreserved` runs, then the response includes a `message` field and the test passes.
- **AC-008 (REQ-006):** Given the `next` and `worktree` tool descriptions, when `TestToolDescriptions_TokenBudget` runs, then both descriptions are within the 250-token budget and the test passes.
- **AC-009 (REQ-007):** Given all fixes applied, when `go test ./...` is run from the repository root, then the command exits 0 with no `FAIL` lines in output.

### Phase 2 — Test Infrastructure

- **AC-010 (REQ-008):** Given the consolidated `writeTestPlan` helper, when any test in the `internal/service` package creates a plan or batch entity, then it uses the single shared helper and the entity type is always `model.EntityKindPlan`.
- **AC-011 (REQ-009):** Given a grep for `if len(id) > 0 && id[0] == 'B'` across the codebase, when the search completes, then zero results are found — the fragile prefix check has been eliminated.
- **AC-012 (REQ-010):** Given the test suite after consolidation, when duplicate-coverage tests are identified, then each identified pair has a documented reason for keeping both or one has been removed.
- **AC-013 (REQ-011):** Given the test suite after pruning, when `go test ./...` runs, then no test fails because it references a removed feature, deprecated lifecycle state, or obsolete behaviour.

### Phase 3 — Enforcement

- **AC-014 (REQ-012):** Given a commit containing a Go file change that introduces a test failure, when `git commit` is run, then the pre-commit hook blocks the commit and prints the failing test names.
- **AC-015 (REQ-012):** Given a commit containing only documentation changes (no `.go` files), when `git commit` is run, then the pre-commit hook allows the commit without running tests.
- **AC-016 (REQ-013):** Given a fresh clone after `kbz init` or `make setup`, when `git commit` is run on a Go change, then the pre-commit hook executes `go test`.
- **AC-017 (REQ-014):** Given `git commit --no-verify` on a change with failing tests, when the command runs, then the commit succeeds and a warning message is emitted to stderr.
- **AC-018 (REQ-015):** Given a feature branch with failing tests, when `merge(action: check, entity_id: "<FEAT-xxx>")` is called, then the check reports "tests failing" as a blocking gate with the list of failed tests.
- **AC-019 (REQ-015):** Given a feature branch with all tests passing, when `merge(action: check, entity_id: "<FEAT-xxx>")` is called, then the test gate passes (non-blocking).
- **AC-020 (REQ-016):** Given a checkout with failing tests, when `kbz doctor` is run, then it reports the number of failing packages and lists the specific test names.
- **AC-021 (REQ-016):** Given a checkout with all tests passing, when `kbz doctor` is run, then it reports "test suite: all passing" or equivalent.
- **AC-022 (REQ-017):** Given a project with test failures, when `health` is called, then the response includes a `test_suite` field with `total_packages`, `passing_packages`, `failing_packages`, and `failures` (list of `{package, test_name}`).

### Phase 4 — Policy and Culture

- **AC-023 (REQ-018):** Given the updated DoD document, when read, then it contains the text "All tests pass on `main` at all times" and the commit-introducing-failure clause.
- **AC-024 (REQ-019):** Given the updated DoD document, when read, then it contains a shared-ownership clause stating that any developer observing a failing test is responsible for reporting it.
- **AC-025 (REQ-020):** Given AGENTS.md, when read, then it documents the pre-commit hook, the "no failing tests on main" rule, and the procedure for reporting test failures.
- **AC-026 (REQ-021):** Given a project with test failures, when `status` is called, then the dashboard includes a test suite health metric alongside entity counts.
- **AC-027 (REQ-022):** Given a commit that removes a test, when the commit message is inspected, then it explains why the test was removed and references the requirement or decision.

---

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Test | `go test -run TestEntityService_CreateFeature ./internal/service/...` — verify feature creation with P-prefixed parent succeeds |
| AC-002 | Test | `go test ./internal/service/...` — all 105 previously-failing tests pass |
| AC-003 | Test | Unit test: call `checkBugWorktreeHasCommits` with nil `*DocumentService`, assert no panic and correct reason |
| AC-004 | Test | `go test -run TestCheckBugTransitionGate_UngatedTransitions ./internal/service/...` |
| AC-005 | Test | `go test ./internal/kbzinit/...` — all 3 consistency tests pass |
| AC-006 | Test | `go test -run TestNextContextToMap ./internal/mcp/...` |
| AC-007 | Test | `go test -run TestHandoff_ErrorResponseShapePreserved ./internal/mcp/...` |
| AC-008 | Test | `go test -run TestToolDescriptions_TokenBudget ./internal/mcp/...` |
| AC-009 | Test | `go test ./...` from repo root — exits 0, zero failures |
| AC-010 | Inspection | Code review: verify all `writeTestPlan` calls use the shared helper with `model.EntityKindPlan` |
| AC-011 | Inspection | `grep -r "id\[0\] == 'B'"` across codebase returns zero results |
| AC-012 | Inspection | Review duplicate-coverage analysis document; verify each pair has a documented disposition |
| AC-013 | Test | `go test ./...` after pruning — no failures from obsolete behaviour |
| AC-014 | Demo | Create a Go file change with a deliberate test failure, run `git commit`, verify block |
| AC-015 | Demo | Create a docs-only change, run `git commit`, verify it succeeds |
| AC-016 | Demo | Run `make setup` on a fresh clone, make a Go change, `git commit`, verify hook runs |
| AC-017 | Demo | Run `git commit --no-verify` on a failing change, verify commit succeeds with warning |
| AC-018 | Test | Integration test: create feature branch with deliberate test failure, call `merge(action: check)`, verify blocking gate |
| AC-019 | Test | Integration test: create feature branch with all tests passing, call `merge(action: check)`, verify gate passes |
| AC-020 | Demo | Run `kbz doctor` on a checkout with known test failures, verify output lists failing packages and tests |
| AC-021 | Demo | Run `kbz doctor` on a checkout with all tests passing, verify output reports success |
| AC-022 | Test | Integration test: call `health`, verify `test_suite` field with correct counts |
| AC-023 | Inspection | Read the DoD document, verify the required text is present |
| AC-024 | Inspection | Read the DoD document, verify the shared-ownership clause is present |
| AC-025 | Inspection | Read AGENTS.md, verify pre-commit hook, no-failing-tests rule, and reporting procedure are documented |
| AC-026 | Test | Integration test: call `status`, verify test suite health metric in dashboard output |
| AC-027 | Inspection | `git log` for test-removal commits, verify commit messages explain the rationale |
| AC-NF-001 | Demo | Run pre-commit hook on a Go change with cached tests; verify completion within 5s. Run full `go test ./...` cold; verify completion within 90s. |
| AC-NF-002 | Demo | Run `kbz doctor` and `go test ./...`; verify doctor elapsed time ≤ test suite elapsed time + 1s. |
| AC-NF-003 | Test | Integration test: call `merge(action: check)` on a feature branch; verify total gate evaluation time ≤ 90s. |
| AC-NF-004 | Test | Integration test: call `health` twice in rapid succession; verify the second call uses cached test results (sub-second response). |
