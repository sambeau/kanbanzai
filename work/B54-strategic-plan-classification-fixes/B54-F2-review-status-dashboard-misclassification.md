# Review: FEAT-01KR03G3TTF5H — Fix status dashboard plan-to-batch misclassification

## Summary

**Feature:** Fix status dashboard plan-to-batch misclassification (BUG-01KQR5XA77Y49)
**Batch:** B54-strategic-plan-classification-fixes
**Reviewers:** reviewer-conformance, reviewer-quality, reviewer-testing
**Date:** 2026-05-07

## Findings

### F1: Cascade through synthesiseProject → generateProjectAttention (PASS)
- **Severity:** ✅ Pass
- **Detail:** Data flow from `synthesiseProject` ➝ `ListPlans` ➝ `isStrategicPlanRecord` ➝ `generateProjectAttention` confirmed correct. The `plan_ready_to_close` check is properly shielded.

### F2: No other callers of `ListPlans` expose stale P-prefix data (PASS)
- **Severity:** ✅ Pass
- **Detail:** All 7 callers of `ListPlans` were audited. None are affected.

### F3: No conflicting `strings.HasPrefix` usage in `status_tool.go` (PASS)
- **Severity:** Non-blocking

### F4: Shared fix at data layer is appropriate
- **Severity:** Non-blocking
- **Detail:** Fix at the data layer protects `listPlanIDs` and `HealthCheck` from the same issue.

### F5: Performance concern — redundant YAML reads
- **Severity:** Non-blocking
- **Detail:** Negligible impact at current scale.

## Verdict

**Pass.** No blocking issues. All cascade paths to the status dashboard are correctly shielded.
