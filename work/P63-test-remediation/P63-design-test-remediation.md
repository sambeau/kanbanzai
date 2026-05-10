# P63 Design — Test Remediation: Fix Failing Tests and Harden Definition of Done

**Status:** Draft  
**Owner:** P63-test-remediation  
**Created:** 2026-05-10

---

## Overview

`go test ./...` reports **111 failing tests** across 3 of 32 packages: `internal/kbzinit` (3), `internal/mcp` (3), and `internal/service` (105). The remaining 29 packages pass.

This is a **Definition of Done violation**: "All tests passing" is a non-negotiable tenet of the project's DoD. Failing tests on `main` mean Kanbanzai is in a broken state — and nobody knew.

This design covers:
- Immediate remediation: fix all 111 failing tests
- Root cause analysis: why tests are brittle and why they decay silently
- Process hardening: prevent failing tests from being committed or ignored ever again

---

## Goals and Non-Goals

### Goals

1. Fix all 111 failing tests so `go test ./...` exits 0
2. Diagnose why tests became brittle enough to fail en masse from a single migration
3. Understand why test breakage goes unnoticed — why aren't tests run before merge?
4. Establish enforcement: no commit to `main` with failing tests
5. Harden the Definition of Done around test expectations
6. Assess whether test pruning (removing low-value tests) is warranted

### Non-Goals

- Re-architecting the test framework
- Rewriting all tests to be less coupled
- Adding net-new test coverage beyond what's needed to fix failures
- Changing the Go testing framework or test runner

---

## Current State

### 2.1 Failing Tests by Package

| Package | Failing | Root Cause |
|---------|---------|------------|
| `internal/service` | 105 | Plan→batch type migration broke test helpers (95); nil DocumentService panic (1); cascade (9) |
| `internal/kbzinit` | 3 | Embedded seed files out of sync with on-disk counterparts |
| `internal/mcp` | 3 | Regression contracts stale — field lists and token budgets need updating |
| **All other 29 packages** | 0 | — |

### 2.2 Historical Context

| Date | Event |
|------|-------|
| 2026-04-28 | P38 plan→batch migration (`369008a8`): `EntityKindPlan` renamed from `"plan"` to `"batch"`. Production code updated; test helpers missed. |
| 2026-05-06 | BUG-01KQZS69DAJFG filed: "44 failing tests." Fixed some but not the type-mismatch root cause. |
| 2026-05-07 | P56 bug lifecycle hardening (`ec2d92b6`): introduced `checkBugWorktreeHasCommits` without nil guard. |
| 2026-05-07–09 | Multiple merges (P51, P55, P57, B64) added new tests using the same broken helpers. |
| 2026-05-10 | **Today**: 111 tests fail. Issue surfaced by manual investigation, not by any automated guard. |

---

## Root Cause Analysis

### 3.1 RC-1: Plan→Batch Type Migration Broke Test Helpers (~95 tests)

**What:** `model.EntityKindPlan` was renamed from `"plan"` to `"batch"` in the P38 migration. All production code (`CreateFeature`, `GetPlan`, `loadPlan`, `ListPlans`) uses the new constant and writes/reads with type `"batch"`. But the test helper `planEntityTypeFromID` still returns `"plan"` for P-prefixed IDs:

```go
// internal/service/entities_test.go:750 — THE BUG
func planEntityTypeFromID(id string) string {
    if len(id) > 0 && id[0] == 'B' { return "batch" }
    return "plan"  // ← should return "batch" since EntityKindPlan == "batch"
}
```

The storage layer maps type to directory: `"plan"` → `plans/`, `"batch"` → `batches/`. Tests write to `plans/`; `GetPlan` reads from `batches/`. Entity not found → `CreateFeature` returns `ErrReferenceNotFound` at setup time. **Zero production bugs — 100% test infrastructure.**

**Why it wasn't caught:** The test helper lives in a test file and wasn't updated during the migration. No `grep` for `"plan"` string literals was run across `_test.go` files.

### 3.2 RC-2: Nil DocumentService Causes Panic (1 test + 3 cascade)

**What:** `checkBugWorktreeHasCommits` in `prereq.go:450` calls `docSvc.RepoRoot()` without checking if `docSvc` is nil. `TestCheckBugTransitionGate_UngatedTransitions` doesn't set up a `DocumentService`, causing a segmentation violation.

```go
// internal/service/prereq.go:450
func checkBugWorktreeHasCommits(entityID string, docSvc *DocumentService) (bool, string) {
    root := docSvc.RepoRoot()  // ← panics if docSvc is nil
```

This was introduced by P56 bug lifecycle hardening (`ec2d92b6`). The 3 `VerifiedToClosed` tests likely cascade from the same issue or RC-1.

### 3.3 RC-3: Embedded Seed Drift (3 tests)

**What:** The dual-write system requires embedded seed files in `internal/kbzinit/` to match their on-disk counterparts. Three consistency tests catch drift — `.agents/skills/` and `.kbz/` files were updated but the embedded copies weren't synced. The tests are working correctly (catching real drift).

### 3.4 RC-4: MCP Regression Contracts Stale (3 tests)

**What:** Three deliberately-brittle contract tests need updating:
- `TestNextContextToMap`: new field `workflow_state_warning` not in canonical list
- `TestHandoff_ErrorResponseShapePreserved`: error response missing `message` field
- `TestToolDescriptions_TokenBudget`: `next` (253 tokens) and `worktree` (264 tokens) exceed 250-token budget

These tests are **designed to fail** on any output change — they force conscious decisions. The decisions simply weren't made.

---

## Investigation: Why Tests are Brittle

### 4.1 Are tests too tightly coupled to exact output?

**Two distinct patterns exist:**

**Pattern A — Deliberately brittle regression tests (MCP, kbzinit):** These are contract tests that encode specific output shapes, field lists, and token budgets. They are *supposed* to fail when output changes. This is a valid testing strategy for external-facing contracts (MCP tool output shapes) and integrity checks (embedded seed consistency). The problem is not the coupling — it's that the contracts weren't updated when the corresponding production changes were made.

**Pattern B — Test setup fragility (service):** The 95 failing service tests are not testing output format. They fail at *setup time* because a shared constant changed and the test infrastructure wasn't updated. The tests themselves test meaningful behaviors (entity lifecycle, cache semantics, dependency resolution). The fragility is in the test infrastructure, not in what the tests assert.

**Verdict:** Tests are not fundamentally too coupled. The MCP regression tests are correctly coupled by design. The service tests broke because of an infrastructure oversight, not because they over-specify output.

### 4.2 Do we have too many low-quality tests?

**No.** The test suite has good characteristics:
- Table-driven tests with `t.Parallel()`
- Tests exercise edge cases (nil cache, cold cache, stale cache, corrupt files)
- Tests validate real behaviors (state machine transitions, cache invalidation, dependency unblocking)
- Each test name describes a specific scenario

The problem is not test quality — it's test *infrastructure*:
- `writeTestPlan` is duplicated across 5+ test files instead of being shared
- The `planEntityTypeFromID` function is a fragile `if id[0] == 'B'` check instead of using the model constant
- No shared test fixture package exists

### 4.3 Why aren't tests updated when new features break old tests?

**Root causes:**

1. **No CI enforcement.** There is no automated `go test ./...` job that blocks merges. Tests can fail silently on `main` for weeks.
2. **Merge gate doesn't run tests.** The Kanbanzai merge gate checks review approval, branch health, and task completion — but doesn't run `go test`.
3. **No pre-commit hook.** Nothing stops a developer from committing with failing tests locally.
4. **Test failures are normalized.** When tests fail for weeks, they become background noise. Developers learn to ignore them: "That's just an existing failure," "That's not my package."

### 4.4 Why are failing tests ignored?

**Cultural and systemic factors:**

1. **No single owner.** When tests fail, no specific person or role is responsible. It's everyone's problem, which means it's nobody's problem.
2. **No alerting.** Failing tests don't generate notifications, dashboard warnings, or blocked merges. They're invisible unless someone runs `go test` manually.
3. **Normalization of deviance.** When the first test failure went unnoticed, the second was easier to ignore, then the third. Now 111 failures feel like background noise rather than a crisis.
4. **The "not my package" reflex.** Developers focus on their feature's packages and ignore failures elsewhere.
5. **No health check integration.** The `health` MCP tool reports entity state issues but doesn't report test suite status.

### 4.5 Why are failing tests allowed to be committed?

**No enforcement layer exists:**

- No pre-commit hook that runs `go test ./...`
- No CI job that blocks PR merges on test failure
- No merge gate check that verifies `go test` passes
- `git push` succeeds regardless of test state

This is a process gap, not a tool gap. Git itself can't enforce test passing — that requires hooks or CI.

### 4.6 Should we be more proactive about removing tests?

**Yes, for specific categories:**

1. **Tests that duplicate coverage:** Some behaviors are tested in multiple packages (e.g., entity creation tested in both `service` and `cmd/kbz`). Consolidating to one location reduces maintenance burden.
2. **Tests that never change:** If a test has never failed (except from infrastructure changes), it may not be testing a meaningful invariant.
3. **Tests with identical setup but different assertions:** These could be table-driven.

**But blanket removal is wrong.** Most tests cover meaningful, distinct behaviors. The right approach is targeted pruning during the fix phase — if a test is hard to fix because it tests something no longer relevant, remove it. Otherwise, fix it.

---

## Remediation Plan

### Phase 1: Fix All Failing Tests

| Step | Tests Fixed | Effort |
|------|-------------|--------|
| 1a. Fix `planEntityTypeFromID` to return `"batch"` and update all callers | ~95 | 1 line |
| 1b. Add nil guard in `checkBugWorktreeHasCommits` | 1 + 3 cascade | 3 lines |
| 1c. Sync embedded seed files from source counterparts | 3 | Mechanical copy |
| 1d. Update MCP regression contracts (field list, error shape, token budgets) | 3 | ~20 lines |

**Total estimated code change:** ~30 lines across ~5 files.

### Phase 2: Test Infrastructure Cleanup

| Step | Rationale |
|------|-----------|
| 2a. Consolidate `writeTestPlan` variants into a single shared helper | Eliminates duplicated type-mapping logic |
| 2b. Replace `planEntityTypeFromID` with direct use of `model.EntityKindPlan` | Removes the fragile `id[0] == 'B'` check |
| 2c. Create `internal/service/testutil` package for shared test fixtures | Single source of truth for test entity creation |
| 2d. Prune tests that duplicate coverage or test removed features | Reduces maintenance surface |

### Phase 3: Prevent Recurrence (Enforcement)

| Step | Mechanism |
|------|-----------|
| 3a. Add pre-commit hook: block commits with failing tests | `.git/hooks/pre-commit` or `pre-commit` config |
| 3b. Add CI job: `go test ./...` must pass before merge | GitHub Actions workflow |
| 3c. Add merge gate check: verify `go test ./...` passes on the feature branch | Kanbanzai `merge(action: check)` integration |
| 3d. Integrate test status into `health` MCP tool output | Dashboard visibility |
| 3e. Add `kbz doctor` check for test suite status | CLI-accessible verification |

### Phase 4: Cultural and Process Changes

| Step | Mechanism |
|------|-----------|
| 4a. Update DoD: "All tests pass on `main` at all times" is non-negotiable | Policy document |
| 4b. Update DoD: "Any developer who sees a failing test is responsible for reporting or fixing it" | Shared ownership |
| 4c. Add "test suite status" to the `status` dashboard | Always-visible health metric |
| 4d. Document the "no failing tests on main" rule in AGENTS.md and contributor docs | Onboarding |
| 4e. Establish: if a test is intentionally removed or changed, the commit message must explain why | Audit trail |

---

## Design: Enforcement Architecture

### Pre-Commit Hook

```
git commit → pre-commit hook → go test ./... → pass? → commit proceeds
                                              → fail? → commit blocked with message
```

The hook should:
- Run only for Go code changes (skip for docs-only commits)
- Be fast: cache test results, run changed packages first
- Be overridable with `--no-verify` for emergencies (with a warning)
- Be installed automatically by `kbz init` or `make setup`

### CI Guard

A GitHub Actions workflow that:
- Triggers on PR open, push to PR branch, and merge to `main`
- Runs `go test ./...` with race detector
- Reports failures as PR checks (blocking merge)
- Caches Go modules and test results for speed

### Merge Gate Integration

The Kanbanzai `merge(action: check)` should:
- Run `go test ./...` on the feature branch
- Block merge if tests fail
- Report which tests failed in the gate check output

---

## Open Questions

1. **Pre-commit hook performance:** `go test ./...` takes ~30s. Is this acceptable for every commit? Or should we run only changed packages?
2. **Test pruning criteria:** What threshold defines "duplicate coverage"? Same assertion in two packages? Same setup with different assertions?
3. **Token budget for MCP descriptions:** Should we raise the 250-token budget, trim descriptions, or both? The overage is small (253 and 264).
4. **Flaky tests:** Do we have any tests that pass sometimes and fail other times? If so, they need to be fixed or quarantined.
5. **`kbz doctor` scope:** Should `kbz doctor` just report test status, or also attempt to diagnose common failure patterns?
6. **Strategic plan or batch?** This is currently a strategic plan (P63). Should implementation happen as a single batch under this plan, or split into multiple batches (one for fixes, one for enforcement)?

---

## Success Criteria

1. `go test ./...` exits 0 with zero failures
2. Pre-commit hook blocks commits with failing tests
3. CI job reports test status on all PRs and blocks merge on failure
4. `kbz doctor` reports test suite health
5. DoD updated with explicit test-passing requirement
6. AGENTS.md documents the "no failing tests on main" rule
7. Zero tests silently broken for more than 24 hours going forward
