# Design: Test Remediation — Fix 111 Failing Tests

**Status:** Draft
**Owner:** B66-test-remediation
**Created:** 2026-05-10

---

## 1. Purpose

Three packages fail `go test`: `internal/kbzinit` (3), `internal/mcp` (3), and `internal/service` (105). Combined, 111 tests fail. "All tests passing" is a fundamental tenet of the Definition of Done — this document diagnoses the root causes and lays out a remediation plan.

---

## 2. Current State

### 2.1 Failing Package Summary

| Package | Failing Tests | Root Cause Category |
|---------|--------------|---------------------|
| `internal/kbzinit` | 3 | Embedded seed drift |
| `internal/mcp` | 3 | Regression test staleness |
| `internal/service` | 105 | Test infra type mismatch (95), nil pointer (1), cascade (9) |
| **Total** | **111** | |

### 2.2 All Other Packages Pass

29 packages pass. The failures are concentrated and do not represent systemic code quality rot.

---

## 3. Root Cause Analysis

### 3.1 RC-1: Plan→Batch Type Migration Broke Test Setup Helpers (~95 tests)

**Severity:** Critical (by volume) / Trivial (by fix complexity)

**What happened:**

The P38 plans-and-batches migration (`ca290005`, `369008a8`) renamed `model.EntityKindPlan` from `"plan"` to `"batch"`:

```go
// internal/model/entities.go
// Deprecated: use EntityKindBatch.
EntityKindPlan EntityKind = EntityKindBatch  // EntityKindBatch = "batch"
```

The production code was updated throughout — `CreateFeature`, `GetPlan`, `loadPlan`, `ListPlans`, etc. all now use `model.EntityKindPlan` (which is `"batch"`).

But **test helpers were not updated**. The critical mismatch is in `planEntityTypeFromID`:

```go
// internal/service/entities_test.go:750
func planEntityTypeFromID(id string) string {
    if len(id) > 0 && id[0] == 'B' {
        return "batch"
    }
    return "plan"  // ← STILL RETURNS "plan" for P-prefixed IDs
}
```

When `writeTestPlan` writes a test plan (`P1-dep-test`), it stores it with `Type: "plan"`. The storage layer's `entityDirectory("plan")` puts it at `plans/P1-dep-test.yaml`.

When `GetPlan` loads it, it calls `store.Load("batch", ...)` which maps to `batches/P1-dep-test.yaml` — a different directory. The file is not found → `ErrReferenceNotFound`.

**Every test that calls `CreateFeature` with a parent plan** fails at setup time, before reaching the code under test. This is pure test infrastructure — zero production code bugs.

**Why weren't tests updated?** The migration commit (`369008a8`) was titled "plan-to-batch code changes — entity model, tools, services" — it touched production code but missed the test helper that maps IDs to types. The batch-level review (B38) didn't catch it because the tests weren't run (or failed but were attributed to "expected migration churn").

### 3.2 RC-2: Nil DocumentService in Bug Gate Test (1 test, +3 cascade)

**Severity:** High (causes panic)

**What happened:**

`checkBugWorktreeHasCommit` in `prereq.go:450` calls `(*DocumentService).RepoRoot()`:

```go
func checkBugWorktreeHasCommits(entityID string, docSvc *DocumentService) (bool, string) {
    root := docSvc.RepoRoot()  // ← panics if docSvc is nil
```

`TestCheckBugTransitionGate_UngatedTransitions` doesn't set up a `DocumentService`, so `docSvc` is nil. This causes a segmentation violation panic.

This function was introduced by P56 bug lifecycle hardening (`ec2d92b6`). The 3 `VerifiedToClosed` tests likely cascade from the same issue or from RC-1 (they also create features with parent plans).

**Fix:** Add a nil guard at the call site or in `checkBugWorktreeHasCommits`.

### 3.3 RC-3: Embedded Seed Drift (3 tests)

**Severity:** Low (no functional impact, purely maintenance)

**What happened:**

The dual-write system requires that embedded seed files in `internal/kbzinit/` match their on-disk counterparts in `.agents/skills/` and `.kbz/`. Three consistency tests catch drift:

- `TestEmbeddedSkillsMatchAgentSkills` — `.agents/skills/` files were updated but not the embedded copies
- `TestEmbeddedTaskSkillsMatchProjectSkills` — `.kbz/skills/` files were updated but not the embedded copies
- `TestEmbeddedRolesMatchProjectRoles` — `.kbz/roles/` files were updated but not the embedded copies

This is a maintenance chore — the embedded files need to be synced from their source counterparts. The tests are serving their purpose by catching the drift.

### 3.4 RC-4: MCP Regression Test Staleness (3 tests)

**Severity:** Low (deliberately brittle, needs conscious update)

| Test | Failure |
|------|---------|
| `TestNextContextToMap_FieldNamesMatchCanonicalList` | New field `workflow_state_warning` in output not in canonical field list |
| `TestHandoff_ErrorResponseShapePreserved` | Error response missing `message` field |
| `TestToolDescriptions_TokenBudget` | `next` (253 tokens) and `worktree` (264 tokens) exceed 250-token budget |

These tests are **designed to be brittle** — they enforce contracts on output shape and token budgets. They're working correctly; the contracts just need updating.

---

## 4. Test Quality Assessment

### 4.1 Are tests too brittle?

**Two distinct answers:**

1. **MCP regression tests: Yes, deliberately.** These are contract tests that encode specific output shapes. Their brittleness is a feature — they force conscious decisions when output formats change. The problem is that the decisions weren't made.

2. **Service tests: No.** The tests themselves are not brittle — they test meaningful behaviors (entity CRUD, dependency hooks, cache behavior, estimation rollup, tier inference). They fail because **test setup** fails, not because the code under test changed behavior. Once setup is fixed, these tests will validate the same behaviors they always did.

### 4.2 Are there too many low-quality tests?

**No.** The test coverage is appropriate:

- Entity lifecycle tests validate state machine behavior
- Cache tests validate read-path performance optimization
- Dependency hook tests validate complex unblocking logic
- Tier inference tests validate classification rules
- Estimation rollup tests validate aggregate computation

The tests use table-driven patterns and `t.Parallel()`. They exercise edge cases (nil cache, cold cache, stale cache, corrupt files). The quality is good — the problem is infrastructure, not test design.

### 4.3 What needs to change?

**One structural improvement:** Test setup helpers are duplicated across files. Each test file has its own `writeTestPlan` variant (`writeTestPlan`, `writeTestPlanWithSeq`, `writeConflictTestPlan`, `writeDecomposeTestPlan`, `writeDispatchTestPlan`). A single shared helper would have confined the type migration fix to one location.

---

## 5. Remediation Plan

### 5.1 Phase 1: Fix Test Infrastructure (fixes ~95 tests)

**Fix:** Update `planEntityTypeFromID` to return `"batch"` for P-prefixed IDs:

```go
func planEntityTypeFromID(id string) string {
    return string(model.EntityKindPlan)  // always "batch"
}
```

Also update `planEntityTypeFromID` callers (or just replace with `model.EntityKindPlan` directly).

**Alternatively:** A single shared `writeTestEntity` helper that uses `model.EntityKindPlan` for plan/batch types, eliminating the per-file duplication.

**Risk:** Low. Tests that already work (those using `writeTestPlan` correctly) will continue to work. The `entityDirectory` function already maps `"batch"` → `"batches"`.

### 5.2 Phase 2: Fix Nil DocumentService (fixes 1 test + 3 cascade)

**Fix:** Add a nil guard in `checkBugWorktreeHasCommits`:

```go
func checkBugWorktreeHasCommits(entityID string, docSvc *DocumentService) (bool, string) {
    if docSvc == nil {
        return false, "document service not available"
    }
    root := docSvc.RepoRoot()
    // ...
}
```

Or handle it at the call site in `CheckBugTransitionGate`.

**Risk:** Low. The nil case is a legitimate state (no document service configured) that should be handled gracefully.

### 5.3 Phase 3: Sync Embedded Seeds (fixes 3 tests)

**Fix:** Copy the updated source files to their embedded counterparts:

- Copy `.agents/skills/kanbanzai-*/SKILL.md` → `internal/kbzinit/skills/*/SKILL.md`
- Copy `.kbz/skills/*/SKILL.md` → `internal/kbzinit/skills/task-execution/*/SKILL.md`
- Copy `.kbz/roles/*.yaml` → `internal/kbzinit/roles/*.yaml`

**Risk:** Low. This is a mechanical sync. The consistency tests verify correctness.

### 5.4 Phase 4: Update MCP Regression Contracts (fixes 3 tests)

**Fix:** Update the canonical field list and token budgets:

1. Add `workflow_state_warning` to `nextContextAllFields` in the regression test
2. Fix the error response shape to include `message` (or update the test to accept the new shape)
3. Trim tool descriptions to fit the 250-token budget, or raise the budget with justification

**Risk:** Low. These are deliberate contract updates. The token budget issue may require trimming prose, which needs careful wording.

---

## 6. Prevention: How Did This Happen?

### 6.1 Root Cause Chain

```
P38 plan→batch migration
  → model.EntityKindPlan renamed to "batch"
  → Production code updated ✓
  → Test helpers NOT updated ✗
  → B38 batch review didn't catch (tests not run)
  → BUG-01KQZS69DAJFG fixed 44 test failures but not this one
  → More merges (P56, P55, P57) added new tests with the same broken pattern
  → 95 tests now fail on the same root cause
```

### 6.2 Process Gaps

1. **Migration completeness:** The plan→batch migration changed the type constant but didn't have a systematic way to find all references to the old `"plan"` string literal.
2. **Test gate not enforced:** The merge gate should require `go test ./...` to pass. The B38 batch was merged with failing tests.
3. **No CI guard:** There's no CI job that runs the full test suite on merge to main (or if there is, it's not blocking).
4. **Distributed test helpers:** Each test file has its own `writeTestPlan` variant. A single shared helper would have been a single fix point.

### 6.3 Recommended Preventive Measures

1. **CI job:** Add a `go test ./...` job that blocks merges when tests fail.
2. **Shared test helpers:** Consolidate `writeTestPlan` variants into a single `internal/service/testutil` package or at least a single file.
3. **Migration checklist:** For future renames, include a step to `grep` for the old string literal across the entire codebase, including `_test.go` files.

---

## 7. Open Questions

1. **Should `planEntityTypeFromID` be removed entirely?** Since `EntityKindPlan == EntityKindBatch`, there's no distinction. The function could be replaced with `string(model.EntityKindPlan)` everywhere.

2. **Should we raise the token budget for tool descriptions?** `next` (253) and `worktree` (264) are only slightly over. The 250-token budget is arbitrary — is it still the right number?

3. **Should the nil DocumentService case be a hard error?** Currently `checkBugWorktreeHasCommits` panics. The fix adds a graceful degradation. But should bug transitions without a document service be blocked?

---

## 8. Success Criteria

1. `go test ./...` exits 0 with no failures
2. All 111 previously-failing tests pass
3. Embedded seed consistency tests pass
4. MCP regression contracts are updated and pass
5. No production behaviour changes (this is test-only remediation)
