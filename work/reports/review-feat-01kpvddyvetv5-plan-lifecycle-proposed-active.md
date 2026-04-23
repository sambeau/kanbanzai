# Review Report: Plan Lifecycle Proposed-to-Active (FEAT-01KPVDDYVETV5)

**Verdict:** pass

## Summary

Four tasks implemented the `proposed → active` shortcut for the plan state machine:

- **T1 (027c27f):** Added `proposed → active` arc to the plan lifecycle state machine in `internal/validate/lifecycle.go`. Updated `TestValidNextStates` and added `TestCanTransition` coverage.
- **T2 (49e88d2):** Wired the precondition check into `UpdatePlanStatus` — `countPostDesigningFeaturesForPlan` queries all features for the plan and counts those in `specifying/dev-planning/developing/reviewing/done`. Rejection returns an error with the `"proposed → designing"` directive. Success appends a system-generated override record: `"proposed → active shortcut: N feature(s) in post-designing state at transition time"`.
- **T3 (3e8d485):** 10 unit tests covering AC-001–AC-010: shortcut success, rejection, override record text/prefix, and regressions for `proposed → designing` and `designing → active`.
- **T4 (e53ab71):** 3 integration tests: AC-011 latency (< 2s), AC-012 fresh-state/no-stale-cache, and end-to-end `proposed → designing → active` regression.

## Findings

### Blocking: None

### Non-Blocking
- The `internal/mcp` package has a pre-existing `TestFinish_RetroOptionalSuggestion` failure (`io: read/write on closed pipe`) that is unrelated to this feature. It is present on main before these changes.
- `designing → active` for plans has no document gate (unlike features). AC-009/AC-010 tests were adapted to verify this is the correct behaviour: `designing → active` succeeds unconditionally, whether or not a design doc is registered on the plan.

## Test Evidence

```
=== RUN   TestPlanShortcut_AC001_OneSpecifyingFeature
--- PASS: TestPlanShortcut_AC001_OneSpecifyingFeature (0.00s)
=== RUN   TestPlanShortcut_AC002_MixedQualifyingFeatures
--- PASS: TestPlanShortcut_AC002_MixedQualifyingFeatures (0.01s)
=== RUN   TestPlanShortcut_AC003_AllFeaturesDesigning
--- PASS: TestPlanShortcut_AC003_AllFeaturesDesigning (0.00s)
=== RUN   TestPlanShortcut_AC004_NoFeatures
--- PASS: TestPlanShortcut_AC004_NoFeatures (0.00s)
=== RUN   TestPlanShortcut_AC005_AllFeaturesProposed
--- PASS: TestPlanShortcut_AC005_AllFeaturesProposed (0.00s)
=== RUN   TestPlanShortcut_AC006_OverrideRecordText_TwoFeatures
--- PASS: TestPlanShortcut_AC006_OverrideRecordText_TwoFeatures (0.00s)
=== RUN   TestPlanShortcut_AC007_OverrideRecordPrefix
--- PASS: TestPlanShortcut_AC007_OverrideRecordPrefix (0.00s)
=== RUN   TestPlanShortcut_AC008_ProposedToDesigning_Regression
--- PASS: TestPlanShortcut_AC008_ProposedToDesigning_Regression (0.00s)
=== RUN   TestPlanShortcut_AC009_DesigningToActive_Regression
--- PASS: TestPlanShortcut_AC009_DesigningToActive_Regression (0.00s)
=== RUN   TestPlanShortcut_AC010_DesigningToActive_WithDesignDoc_Regression
--- PASS: TestPlanShortcut_AC010_DesigningToActive_WithDesignDoc_Regression (0.00s)
=== RUN   TestPlanShortcutIntegration_AC011_LatencyUnder2s
--- PASS: TestPlanShortcutIntegration_AC011_LatencyUnder2s (0.01s)
=== RUN   TestPlanShortcutIntegration_AC012_FreshStateNoStaleCache
--- PASS: TestPlanShortcutIntegration_AC012_FreshStateNoStaleCache (0.00s)
=== RUN   TestPlanShortcutIntegration_ProposedDesigningActive_EndToEnd
--- PASS: TestPlanShortcutIntegration_ProposedDesigningActive_EndToEnd (0.00s)
ok  github.com/sambeau/kanbanzai/internal/service  1.145s
```

## Conclusion

All 12 ACs satisfied (10 unit + 2 integration + 1 end-to-end regression). The shortcut fires with zero false positives, writes a correctly-prefixed override record, rejects cleanly when no qualifying features exist, and completes well under the 2s latency budget. Ready to merge.
