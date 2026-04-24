# Review: FEAT-01KPXGW5BCGY4 — orphaned-reviewing-dashboard-warning

Feature: FEAT-01KPXGW5BCGY4 — orphaned-reviewing-dashboard-warning
Review cycle: 1
Reviewers dispatched: reviewer-conformance, reviewer-quality, reviewer-testing
Review units: 2

---

## Per-Reviewer Summary

  Reviewer: reviewer-conformance
  Review unit: status_tool.go — generateOrphanedReviewingAttention + wiring (synthesiseProject, synthesisePlan, synthesiseFeature)
  Verdict: approved
  Dimensions:
    spec-conformance: pass
    acceptance-criteria: pass
  Findings: 0 blocking, 0 non-blocking

  Reviewer: reviewer-quality
  Review unit: status_tool.go — reviewingCandidate type, helper function, scope wiring
  Verdict: approved
  Dimensions:
    code-quality: pass
    maintainability: pass
    error-handling: pass
  Findings: 0 blocking, 1 non-blocking

  Reviewer: reviewer-testing
  Review unit: status_tool_test.go — TestGenerateOrphanedReviewingAttention_* suite
  Verdict: approved
  Dimensions:
    test-coverage: pass
    test-quality: pass
  Findings: 0 blocking, 0 non-blocking

---

## Collated Findings (deduplicated)

  [NB-1] (non-blocking)
  Dimension: code-quality / maintainability
  Location: internal/mcp/status_tool.go — reviewingCandidate type declaration
  Description: The `reviewingCandidate` type is declared after the `generateOrphanedReviewingAttention`
    function that uses it, separated by the `hasDocType` function comment banner. Convention in this file
    is to declare supporting types near their first point of use or in a logical grouping block. Moving the
    type declaration to just above the function would improve readability, though this is a style nit with no
    correctness impact.
  Reported by: reviewer-quality

---

## Spec Conformance Trace

| Requirement | Status | Evidence |
|---|---|---|
| FR-001 — check runs at project scope | ✅ pass | `synthesiseProject` appends `generateOrphanedReviewingAttention` results |
| FR-002 — check runs at plan scope | ✅ pass | `synthesisePlan` appends `generateOrphanedReviewingAttention` results |
| FR-003 — check runs at feature scope | ✅ pass | `synthesiseFeature` appends `generateOrphanedReviewingAttention` results |
| FR-004 — condition: reviewing + no report | ✅ pass | `generateOrphanedReviewingAttention` calls `ListDocuments(owner, type=report)` and emits item on empty result |
| FR-005 — severity: warning | ✅ pass | `Severity: "warning"` set on every emitted item |
| FR-006 — item references entity_id | ✅ pass | `EntityID: c.ID` in each item |
| FR-007 — message format | ✅ pass | `fmt.Sprintf("Feature %s (%s) is in 'reviewing' status with no registered review report", c.DisplayID, c.Slug)` matches spec exactly |
| FR-008 — early exit when no reviewing features | ✅ pass | `if len(candidates) == 0 \|\| docSvc == nil { return nil }` |
| FR-009 — fail-open on doc service unavailability | ✅ pass | nil docSvc check in early return; per-feature `continue` on error |
| FR-010 — pre-existing orphans surface on every call | ✅ pass | check runs unconditionally on each status() invocation |
| FR-011 — one query per reviewing feature | ✅ pass | loop calls `ListDocuments` exactly once per candidate |
| NFR-001 — no blocking behaviour | ✅ pass | no synchronisation primitives, no result gating |
| NFR-002 — graceful degradation per feature | ✅ pass | per-feature `continue` on error leaves other items unaffected |
| NFR-003 — scope isolation | ✅ pass | project scope iterates `allFeatures`, plan scope iterates `features` (plan-scoped slice), feature scope uses single candidate |

## Acceptance Criteria Trace

| AC | Status | Test |
|---|---|---|
| AC-001 — project scope, orphaned features produce warnings | ✅ pass | `TestGenerateOrphanedReviewingAttention_NoReports_EmitsWarning` + project wiring confirmed by code review |
| AC-002 — project scope, features with report produce no items | ✅ pass | `TestGenerateOrphanedReviewingAttention_WithReports_NoItems` |
| AC-003 — plan scope, orphaned feature in plan produces warning | ✅ pass | plan-scope wiring iterates plan-scoped `features` slice; same helper tested |
| AC-004 — plan scope, feature in other plan not surfaced | ✅ pass | `synthesisePlan` passes only plan-filtered features as candidates |
| AC-005 — feature scope, orphaned feature produces warning | ✅ pass | feature-scope wiring confirmed by code review |
| AC-006 — feature scope, feature with report produces no items | ✅ pass | same helper; `TestGenerateOrphanedReviewingAttention_WithReports_NoItems` |
| AC-007 — no reviewing features → no items, no queries | ✅ pass | `TestGenerateOrphanedReviewingAttention_EmptyCandidates`; early-exit before any ListDocuments call |
| AC-008 — doc service unavailable → no error, no items from failure | ✅ pass | `TestGenerateOrphanedReviewingAttention_NilDocSvc` |
| AC-009 — severity is warning | ✅ pass | `TestGenerateOrphanedReviewingAttention_NoReports_EmitsWarning` asserts `item.Severity == "warning"` |
| AC-010 — message pattern matches spec | ✅ pass | `TestGenerateOrphanedReviewingAttention_NoReports_EmitsWarning` asserts exact message |

---

## Aggregate Verdict: approved

All functional requirements and acceptance criteria are satisfied. Test coverage is thorough for the
`generateOrphanedReviewingAttention` helper across all defined scenarios. The single non-blocking
finding (NB-1) is a minor code organisation nit with no correctness impact and does not warrant
remediation before merge.