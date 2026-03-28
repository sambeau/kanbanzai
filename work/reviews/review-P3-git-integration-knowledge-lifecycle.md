# Review: P3 — Git Integration and Knowledge Lifecycle

| Field    | Value                                              |
|----------|----------------------------------------------------|
| Scope    | Phase 3 (all tracks A–K)                          |
| Reviewer | Claude Sonnet 4.6                                  |
| Date     | 2026-03-28T19:16:55Z                               |
| Verdict  | **Conditional Pass — 2 critical issues to fix**    |

---

## Summary

Phase 3 is a substantial and well-executed body of work. The core domain logic for worktrees, branch tracking, cleanup scheduling, knowledge lifecycle (TTL, pruning, promotion, compaction), health checks, GitHub PR integration, and the MCP/CLI surfaces is correct, idiomatic, and well-tested at the unit level. All Phase 3 packages build cleanly, `go vet` is clean, and all unit tests pass with the race detector enabled.

However, the post-implementation audit report (`work/plan/phase-3-progress.md`) contains two factual inaccuracies: items R2 and R17 are marked ✅ but the corresponding code changes were not made. These are the most significant findings in this review and must be addressed before Phase 3 can be considered fully complete.

---

## Track Status

| Track | Name                     | Spec Conformance        | Code Quality |
|-------|--------------------------|-------------------------|--------------|
| A     | Worktree management      | ✅ Met                  | Strong       |
| B     | Branch tracking          | ✅ Met                  | Strong       |
| C     | Merge gates              | ⚠️ Undocumented gate    | Good         |
| D     | GitHub PR integration    | ✅ Met                  | Good         |
| E     | Post-merge cleanup       | ✅ Met                  | Good         |
| F     | Git anchoring/staleness  | ⚠️ Divergent path       | Good         |
| G     | TTL-based pruning        | ✅ Met                  | Strong       |
| H     | Automatic promotion      | ✅ Met                  | Strong       |
| I     | Post-merge compaction    | ⚠️ Minor testability gap| Strong       |
| J     | Health check extensions  | ✅ Met                  | Strong       |
| K     | Configuration            | ⚠️ Minor gap            | Good         |

---

## Findings

Issues are ordered by severity. Items 1 and 2 are **blocking** — the progress doc claims they were fixed but the code does not reflect those fixes.

---

### 🔴 Finding 1 — Override Events Not Logged (R2 Not Implemented)

**File:** `internal/mcp/merge_tool.go` → `executeMerge`  
**Spec:** §8.4

The progress doc states: *"R2 Fixed: `executeMerge` now calls `merge.CreateOverrides` when overriding blocked gates, and includes override records in the merge response."*

This is not true. The actual code:

```go
mergeMessage := fmt.Sprintf("Merge %s: %s", entityID, entityTitle)
if override {
    mergeMessage += fmt.Sprintf("\n\nOverride reason: %s", overrideReason)
}
```

`merge.CreateOverrides()` is never called. The worktree record is never updated with an `overrides` block. Nothing about the override appears in the MCP response. The override reason only appears in the git commit message body, which is not durable workflow state and cannot be queried.

The spec §8.4 requires overrides to be stored in the worktree record:

```yaml
overrides:
  - gate: tasks_complete
    reason: "Hotfix - will backfill tests"
    overridden_by: sambeau
    overridden_at: 2025-01-27T15:00:00Z
```

The `merge.Override` type, `merge.CreateOverrides`, and `merge.FormatOverrides` all exist and are correct — they are simply never called from `executeMerge`.

**Fix:** Add an `Overrides []Override` field to `worktree.Record`, include it in `FieldOrder` and YAML serialisation, call `merge.CreateOverrides` inside `executeMerge` when `override==true`, persist the result via `worktreeStore.Update`, and include the formatted overrides in the response. Alternatively, make an explicit design decision that overrides are only captured in the commit message and update the spec accordingly — but the current state (code says one thing, spec and audit doc say another) is a defect.

---

### 🔴 Finding 2 — `FormatGateResults` Not Used (R17 Not Implemented)

**File:** `internal/mcp/merge_tool.go` → `checkMergeReadiness`  
**Progress doc R17:** *"`checkMergeReadiness` now uses `merge.FormatGateResults` instead of manual gate result construction"*

This is also not true. The function still manually constructs gate results:

```go
gates := make([]map[string]any, 0, len(gateResult.Gates))
for _, g := range gateResult.Gates {
    gate := map[string]any{
        "name":     g.Name,
        "status":   string(g.Status),
        "severity": string(g.Severity),
    }
    if g.Message != "" {
        gate["message"] = g.Message
    }
    gates = append(gates, gate)
}
resp["gates"] = gates
```

`merge.FormatGateResults` (defined in `internal/merge/format.go`) adds a `summary` sub-object with total/passed/failed/warning counts, which the spec §8.5 shows as part of the check output. The manual construction omits this. As a result, `merge.FormatGateResults` is an unreachable exported function — it exists, tests pass, but it is never called by production code.

**Fix:** Replace the manual loop with a call to `merge.FormatGateResults(gateResult)` and merge the returned map into `resp`. This is a two-line change.

---

### 🟠 Finding 3 — Default Branch Hardcoded in `executeMerge`

**File:** `internal/mcp/merge_tool.go` → `executeMerge`

R9 consolidated default-branch detection using `git.GetDefaultBranch()` across merge gates, PR creation, and elsewhere. Two sites in `executeMerge` were missed:

```go
// Site 1: conflict check
hasConflicts, err := git.HasMergeConflicts(repoPath, wt.Branch, "main")
if err != nil {
    // Try master
    hasConflicts, err = git.HasMergeConflicts(repoPath, wt.Branch, "master")
}

// Site 2: checkout before merge
if err := gitOps.CheckoutBranch("main"); err != nil {
    // Try master
    if err := gitOps.CheckoutBranch("master"); err != nil {
        return nil, fmt.Errorf("checkout base branch: %w", err)
    }
}
```

Using `err != nil` as the signal to retry with "master" is wrong for the conflict check: if "main" exists and the branch has no conflicts, `err` is `nil` and the fallback never runs — but this means a repo whose default is "master" will always report no conflicts. Both sites will also silently fail for repos whose default branch is neither "main" nor "master".

**Fix:** Call `git.GetDefaultBranch(repoPath)` once at the top of `executeMerge`, propagate errors, and use the result for both the conflict check and the checkout.

---

### 🟠 Finding 4 — `entity_done` Gate Not in Spec

**Files:** `internal/merge/gates.go`, `internal/merge/checker.go`  
**Spec:** §8.2

The spec defines 6 merge gates. The implementation has 7. `EntityDoneGate` checks that features are `"done"` and bugs are `"closed"` before merge — a sensible gate. But it is not in the spec, which means:

- The `§20.3` acceptance criterion "all defined gates are checked" is measured against 6 spec-defined gates but the implementation runs 7.
- `checker_test.go` hard-codes `len(gates) != 7`, enshrining the undocumented gate without explanation.
- Agents and humans reading §8.2 will not know this gate exists or why it might fail.

**Fix:** Add `entity_done` to §8.2 of the spec with its severity (`blocking`) and check description ("Feature status is `done`; bug status is `closed`"). Add a brief comment in `gates.go` explaining why this gate exists in addition to `tasks_complete`.

---

### 🟠 Finding 5 — Worktree Update Failure Silently Swallowed

**File:** `internal/mcp/merge_tool.go` → `executeMerge`

```go
wt.MarkMerged(mergedAt, gracePeriodDays)
if _, err := worktreeStore.Update(wt); err != nil {
    // Log but don't fail - merge succeeded
    _ = err
}
```

The comment says "Log" but `_ = err` does not log anything. If the worktree record update fails, the worktree remains in `active` status — it will never appear in the cleanup list and will accumulate indefinitely. Compare the R10 fix in `worktree_tools.go`, which correctly appends update failures as warnings in the response.

**Fix:** Replace `_ = err` with `log.Printf("warning: failed to update worktree record after merge: %v", err)` at minimum. Better: append `"worktree_update_warning"` to the response map so the caller can surface it.

---

### 🟡 Finding 6 — `checkEntryStaleness` Duplicates `git.CheckEntryStaleness`

**Files:** `internal/mcp/knowledge_tool.go` (L650–689), `internal/git/knowledge.go`

`git.CheckEntryStaleness(repoPath string, fields map[string]any)` already exists and does exactly what the private `checkEntryStaleness` helper in `knowledge_tool.go` does — except the private version adds an undocumented fallback to the `"updated"` field when `"last_confirmed"` is absent:

```go
} else if updatedStr, ok := fields["updated"].(string); ok && updatedStr != "" {
    lastConfirmed, _ = time.Parse(time.RFC3339, updatedStr)
}
```

`git.CheckEntryStaleness` uses `git.GetLastConfirmed()`, which only reads `"last_confirmed"`. This divergence means the MCP staleness tool and the health check staleness checker can produce different answers for the same entry — silent inconsistency.

**Fix:** Delete the private `checkEntryStaleness` helper and call `git.CheckEntryStaleness(repoPath, rec.Fields)` directly. If the `"updated"` fallback is intentional, add it to `git.GetLastConfirmed()` with a comment so all callers share the behaviour.

---

### 🟡 Finding 7 — `knowledgeStalenessAction` Uses Hardcoded `repoPath = "."`

**File:** `internal/mcp/knowledge_tool.go` (L597)

```go
repoPath := "."
```

Every other git-dependent MCP tool receives `repoPath` from server initialisation (wired in `server.go`). If the MCP server is started from any directory other than the repository root, `"."` is wrong and all staleness checks will silently return incorrect results or spurious errors.

**Fix:** Thread `repoPath` from server wiring into `knowledgeStalenessAction` as a closure parameter, matching the pattern used in `branch_tool.go` and `merge_tool.go`.

---

### 🟡 Finding 8 — `mergePhase3Defaults` Does Not Handle `MaxMissCount`

**File:** `internal/config/config.go` → `mergePhase3Defaults`

All other `Knowledge.Promotion` fields are covered by default-filling logic, but `MaxMissCount` is absent:

```go
if c.Knowledge.Promotion.MinUseCount == 0 {
    c.Knowledge.Promotion.MinUseCount = knowledgeDefaults.Promotion.MinUseCount
}
if c.Knowledge.Promotion.MinConfidence == 0 {
    c.Knowledge.Promotion.MinConfidence = knowledgeDefaults.Promotion.MinConfidence
}
// MaxMissCount not handled
```

The spec default for `MaxMissCount` is 0, which is also Go's zero value, so this is currently harmless. But if the spec ever changes this default, the omission will silently break promotion logic. The asymmetry also misleads readers of this function.

**Fix:** Add the missing guard: `if c.Knowledge.Promotion.MaxMissCount == 0 { c.Knowledge.Promotion.MaxMissCount = knowledgeDefaults.Promotion.MaxMissCount }`.

---

### 🟡 Finding 9 — Dead Clause in `isBranchNotFoundError`

**File:** `internal/cleanup/execute.go`

```go
return strings.Contains(errStr, "not found") ||
    strings.Contains(errStr, "branch") && strings.Contains(errStr, "not found")
```

Due to Go operator precedence (`&&` binds tighter than `||`), the second operand is `strings.Contains(errStr, "branch") && strings.Contains(errStr, "not found")`. Any string matching the second operand also matches the first (`"not found"`), making the second clause unreachable.

**Fix:** Remove the dead clause; keep only `strings.Contains(errStr, "not found")`.

---

### 🟡 Finding 10 — `MergeEntries` Uses `time.Now()` Directly

**File:** `internal/knowledge/compact.go`

```go
discarded["retired_at"] = time.Now().UTC().Format(time.RFC3339)
```

Every other time-dependent function in the knowledge package accepts a `now time.Time` parameter (`CheckPruneCondition`, `ResetTTL`, `ApplyPromotion`) to support deterministic testing. `MergeEntries` is the only outlier. Tests that exercise this function cannot assert on `retired_at`.

**Fix:** Add a `now time.Time` parameter to `MergeEntries` and pass `time.Now().UTC()` at the call site in `compactActiveEntries`.

---

### 🟢 Finding 11 — `worktreeRemoveAction` Deletes Without Status Transition

**File:** `internal/mcp/worktree_tool.go` → `worktreeRemoveAction`

The remove action calls `gitOps.RemoveWorktree` then `store.Delete`. If the git remove succeeds but the store delete fails, the record remains in state pointing to a non-existent directory — the opposite of the standard "persist state before acting" pattern used by the transition hook.

The more resilient sequence would be: update record to `StatusAbandoned` first (durable, survives restarts), then attempt git removal and record deletion.

**Fix:** This is a low-priority robustness improvement. Worth addressing in a follow-up cleanup pass rather than blocking Phase 3.

---

### 🟢 Finding 12 — `HealthCheckCleanGate` Always Passes Without Warning

**File:** `internal/merge/gates.go`

```go
// HealthCheckCleanGate checks that there are no blocking health-check errors.
// This is a placeholder implementation that always passes.
type HealthCheckCleanGate struct{}
```

This is a known deferral and is documented. The concern is that this gate is listed in spec §8.2 as **blocking**, so merges proceed even when there are blocking health errors — users have no signal that this protection is not active.

**Recommendation:** Add a `// TODO(phase-N): implement` comment cross-referencing the future phase that will wire this up. Consider removing `HealthCheckCleanGate` from `DefaultGates()` until it is implemented — a gate that always passes provides false confidence without providing protection.

---

## Test Coverage Assessment

| Package               | Unit Coverage | MCP-Level Coverage | Notes                                      |
|-----------------------|---------------|--------------------|--------------------------------------------|
| `worktree/`           | ✅ Excellent  | ⚠️ Good (5 tests)  | Minor gap: no concurrent-access test       |
| `git/`                | ✅ Excellent  | N/A                | Uses real git repos; strong                |
| `merge/`              | ✅ Good       | ❌ None            | Deferred R7; unit coverage is sufficient   |
| `github/`             | ✅ Good       | ❌ None            | Deferred R7; httptest mocks are good       |
| `cleanup/`            | ⚠️ Weak       | ❌ None            | Non-dry-run path completely untested       |
| `knowledge/`          | ✅ Excellent  | ❌ None            | 55 test funcs at unit level; very strong   |
| `health/`             | ✅ Excellent  | ⚠️ Partial         | 34 category tests; strong                  |
| `config/`             | ✅ Good       | N/A                | Defaults and validation tested             |
| `service/` (hook)     | ✅ Excellent  | N/A                | 18 tests; real git, idempotency, E2E       |

The non-dry-run path in `cleanup/execute.go` (`ExecuteCleanup` with `DryRun: false`) has zero test coverage. This exercises the git worktree removal, branch deletion, and remote branch deletion code paths. While these depend on a real git repository, the failure modes (partial cleanup, record-but-no-directory inconsistency) are exactly the cases most likely to surface in production.

---

## Documentation Assessment

| Document                               | Status                                             |
|----------------------------------------|----------------------------------------------------|
| `work/spec/phase-3-specification.md`   | ⚠️ §8.2 missing `entity_done` gate (Finding 4)    |
| `work/plan/phase-3-progress.md`        | ⚠️ R2 and R17 incorrectly marked complete          |
| `work/plan/phase-3-decision-log.md`    | ✅ Accurate and complete                           |
| `AGENTS.md`                            | ✅ Phase 3 packages and scope guard updated        |
| CLI usage text (all Phase 3 commands)  | ✅ Present and accurate                            |
| MCP tool descriptions                  | ✅ Well written; action parameters documented      |
| Package-level doc comments             | ✅ All Phase 3 packages have package docs          |

---

## Verdict

Phase 3 is **conditionally passing**. The domain logic is correct, well-structured, and well-tested. The MCP and CLI interfaces are complete and properly documented.

Two items must be resolved before the phase can be signed off as fully complete:

1. **Finding 1** — Implement override event logging to the worktree record (or formally decide against it and update the spec).
2. **Finding 2** — Replace the manual gate result construction in `checkMergeReadiness` with `merge.FormatGateResults`.

Findings 3–5 are high-priority follow-ups that should be addressed promptly (before the next phase ships code that depends on `executeMerge` correctness). Findings 6–10 are quality improvements that can be batched into a cleanup pass.

The progress document (`work/plan/phase-3-progress.md`) should be corrected to accurately reflect which items are and are not implemented, so future agents have a reliable picture of the actual state.