# Batch 1 Bug Fixes — Review Report

## Review Unit: Batch 1 fast_fix bugs

**Files changed:**
- `cmd/kbz/main.go` — SQLite cache attach for CLI
- `internal/service/documents.go` — false positive bypass in approval
- `internal/structural/promotion.go` — Entry() method
- `internal/service/decompose.go` — REQ-traced AC regex
- `internal/service/decompose_test.go` — REQ-traced AC test
- `internal/service/documents_test.go` — false positive bypass test
- `internal/structural/promotion_test.go` — Entry() tests

**Spec:** BUG-01KQB6TKSABJJ, BUG-01KQFB1E8WPT1, BUG-01KPVGMMP56GC
**Reviewer Role:** reviewer-conformance, reviewer-quality

## Overall: approved_with_followups

## Dimensions

### spec_conformance: pass

**Evidence:**
- BUG-01KQB6TKSABJJ: CLI cache fix matches MCP server pattern — `cache.Open` in defaultDependencies, `SetCache` + `RebuildCache` in newEntityService factory, guarded by sync.Once, best-effort fallback. (main.go L64-87)
- BUG-01KQFB1E8WPT1: False positive bypass correctly loads `PromotionState` from `s.stateRoot`, checks `FalsePositiveCount > 0` before rejecting on missing sections. (documents.go L530-544)
- BUG-01KPVGMMP56GC: Regex updated to accept optional `(REQ-NNN)` parenthetical reference and either `.` or `:` before `**`. All three capture groups unchanged. (decompose.go L413)

**Findings:** None.

### implementation_quality: pass

**Evidence:**
- CLI cache: sync.Once prevents redundant rebuild; cache is closure-scoped to share across entity service instances
- False positive bypass: correct key construction (`required_sections/<normalised-type>`); silent fall-through on LoadPromotionState error errs on side of caution (reject)
- Regex: single-line change, non-capturing group for parenthetical, preserves all downstream code
- All builds pass (go build ./cmd/kbz/)
- All existing tests pass (go test ./... in affected packages)

**Findings:** None.

### test_adequacy: pass_with_notes

**Evidence:**
- CLI cache: Existing cache tests in `internal/mcp/server_warmup_test.go` cover RebuildCache (AC-001 through AC-009). Wiring test in `cmd/kbz/` would require a controlled filesystem environment. Acceptable gap.
- False positive bypass: `TestApproveDocument_FalsePositiveBypassesMissingSections` covers the bypass scenario. `TestPromotionState_Entry_Exists` and `TestPromotionState_Entry_NotFound` cover the new `Entry()` method.
- REQ-traced AC: `TestParseSpecStructure_BoldIdent_WithREQTrace` covers `**AC-001 (REQ-001):**` format with 2 criteria.

**Findings:**
- [non-blocking] CLI cache wiring is not tested at the integration level — the factory code touches the filesystem which makes unit testing impractical without a temp-kbz scaffolding. This is acceptable given that the underlying RebuildCache/SetCache interface is well-tested.
- [non-blocking] No test verifies that a non-zero `FalsePositiveCount` with a *different* check type does NOT bypass the gate. The current test only covers the positive case. This is low-risk because the key matching is exact string comparison.

## Finding Summary

| Category | Count |
|----------|-------|
| Blocking | 0 |
| Non-blocking | 2 |
| **Total** | **2** |

## Verdict

All three bugs are fixed correctly. The implementation matches the bug report expectations, builds clean, and all tests pass. The two non-blocking findings are minor and do not affect correctness.
