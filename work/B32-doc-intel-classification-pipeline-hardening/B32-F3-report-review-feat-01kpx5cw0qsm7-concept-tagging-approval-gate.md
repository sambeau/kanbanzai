# Review Report: Concept Tagging Approval Gate (FEAT-01KPX5CW0QSM7)

**Feature:** FEAT-01KPX5CW0QSM7  
**Plan:** P32 — Doc-Intel Classification Pipeline Hardening  
**Reviewer:** orchestrator  
**Date:** 2026-04-24  
**Verdict:** pass

---

## Summary

This feature adds a server-side hard gate to `doc approve` that blocks approval of
`specification`, `design`, and `dev-plan` documents until at least one classified section
has `concepts_intro` populated. It escalates concept tagging from a soft advisory nudge
(routinely ignored) to an enforced prerequisite at approval time.

The gate returns a structured soft-error response (`"error": "concept_tagging_required"`)
containing `content_hash` so callers can proceed directly to classify without an extra
`doc_intel guide` call.

---

## Review Rounds

### Round 1 — Findings

| # | Severity | Finding |
|---|----------|---------|
| F-1 | Major | Gate-block response omitted `content_hash`; callers could not proceed to classify without a separate `doc_intel guide` round-trip, defeating the 2-call workflow goal |
| F-2 | Major | AC-006 test gutted — the test existed but no longer asserted the `content_hash` field, providing false coverage confidence |

Both findings sent the feature to `needs-rework`.

### Round 2 — Findings

| # | Severity | Finding |
|---|----------|---------|
| F-3 | Major | `content_hash` restored in gate-block payload but AC-006 test still not asserting its value correctly |
| F-4 | Major | `concepts_intro` present check used wrong comparison; gate could fire even when tagging was complete |

Both findings sent the feature to `needs-rework`.

### Round 3 — Findings

None. Both issues confirmed fixed:

- Gate-block response now returns full structured payload including `content_hash` (head `062a472f`).
- AC-006 test (`TestApproveGate_BlockedResponseIncludesContentHash`) asserts the hash value correctly.
- `concepts_intro` check logic correct; gate fires only when no section has a non-empty `concepts_intro`.

**Verdict: pass.**

---

## Tasks Reviewed

13 tasks completed. 12 `not-planned` tasks are stubs from two superseded decomposition
runs and do not represent missing work — all acceptance criteria are covered by the 13
completed tasks.

---

## Findings

### Blocking

None (all resolved across review rounds).

### Non-Blocking

None.

---

## Test Evidence

```
go test ./internal/mcp/ -run TestApproveGate -v -count=1

--- PASS: TestApproveGate_SkipsOutOfScopeType (0.05s)
--- PASS: TestApproveGate_SkipsUnclassifiedDoc (0.05s)
--- PASS: TestApproveGate_PassesWhenConceptsIntroPresent (0.05s)
--- PASS: TestApproveGate_BlockedForSpecWithNoConceptsIntro (0.06s)
--- PASS: TestApproveGate_BlockedForDesignWithNoConceptsIntro (0.06s)
--- PASS: TestApproveGate_BlockedForDevPlanWithNoConceptsIntro (0.05s)
--- PASS: TestApproveGate_BlockedResponseIncludesContentHash (0.06s)
--- PASS: TestApproveGate_SkipsNilIntelligenceService (0.05s)
--- PASS: TestApproveGate_AllowsReportType (0.04s)
--- PASS: TestApproveGate_AllowsPolicyType (0.04s)
--- PASS: TestApproveGate_AllowsResearchType (0.04s)
PASS
ok  github.com/sambeau/kanbanzai/internal/mcp  0.312s
```

---

## Spec Traceability

| Requirement | AC | Covered By | Result |
|-------------|-----|------------|--------|
| REQ-001 (gate skips out-of-scope types) | AC-001 | TestApproveGate_SkipsOutOfScopeType | ✅ |
| REQ-002 (gate skips unclassified docs) | AC-002 | TestApproveGate_SkipsUnclassifiedDoc | ✅ |
| REQ-003 (gate passes when concepts_intro present) | AC-003 | TestApproveGate_PassesWhenConceptsIntroPresent | ✅ |
| REQ-004 (gate blocks spec with no concepts_intro) | AC-004 | TestApproveGate_BlockedForSpecWithNoConceptsIntro | ✅ |
| REQ-005 (gate blocks dev-plan with no concepts_intro) | AC-005 | TestApproveGate_BlockedForDevPlanWithNoConceptsIntro | ✅ |
| REQ-006 (blocked response includes content_hash) | AC-006 | TestApproveGate_BlockedResponseIncludesContentHash | ✅ |
| REQ-007 (gate skips nil intelligence service) | AC-007 | TestApproveGate_SkipsNilIntelligenceService | ✅ |
| REQ-008 (report/policy/research types not blocked) | AC-008 | TestApproveGate_AllowsReportType, _AllowsPolicyType, _AllowsResearchType | ✅ |

---

## Conclusion

All 8 acceptance criteria satisfied after three review rounds. The two major findings
(missing `content_hash` in gate payload; AC-006 false coverage) were both resolved by
head commit `062a472f`. The 12 `not-planned` stub tasks are confirmed non-work. Feature
is ready to merge.