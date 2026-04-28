# Review: Non-bypassable Merge Gate for Review Report

Feature: FEAT-01KPXGVQY3KQC — merge-gate-review-report
Review cycle: 1
Reviewers dispatched: reviewer-conformance, reviewer-quality, reviewer-security, reviewer-testing
Review units: 4

---

## Per-Reviewer Summary

  Reviewer: reviewer-conformance
  Review unit: Spec conformance — gate behaviour, Bypassable field, override rejection
  Verdict: approved
  Dimensions:
    spec-conformance: pass
    acceptance-criteria: pass
  Findings: 0 blocking, 1 non-blocking

  Reviewer: reviewer-quality
  Review unit: Implementation quality — gate.go, gates.go, checker.go, merge_tool.go
  Verdict: approved
  Dimensions:
    code-quality: pass
    maintainability: pass
    error-handling: pass
  Findings: 0 blocking, 0 non-blocking

  Reviewer: reviewer-security
  Review unit: Security — override rejection logic, fail-open behaviour
  Verdict: approved
  Dimensions:
    security: pass
    data-integrity: pass
  Findings: 0 blocking, 0 non-blocking

  Reviewer: reviewer-testing
  Review unit: Test coverage — gates_test.go, checker_test.go, merge_tool_test.go
  Verdict: approved_with_followups
  Dimensions:
    unit-coverage: pass
    integration-coverage: partial
    acceptance-criteria-coverage: partial
  Findings: 0 blocking, 1 non-blocking

---

## Collated Findings (deduplicated)

  [NB-1] (non-blocking)
  Dimension: integration-coverage
  Location: internal/mcp/merge_tool_test.go
  Spec ref: AC-002, AC-007
  Description: There is no merge_tool-level integration test covering AC-002
  (override: true is rejected when ReviewReportExistsGate fires non-bypassably)
  or AC-007 (the fail-open path through executeMerge when the document service
  returns an error). Gate-level unit tests (TestReviewReportExistsGate_DocServiceError_FailsOpen,
  TestReviewReportExistsGate_NilDocService_FailsOpen) and checker-level integration
  tests (TestCheckGates_ReviewingFeature_NoReport_Blocked,
  TestCheckGates_NilDocSvc_ReviewingGateFailsOpen) provide thorough coverage of
  the underlying logic. The gap is that no test drives executeMerge with
  override: true and a reviewing feature that has no report, which would exercise
  the NonBypassableBlockingFailures branch in merge_tool.go and confirm the error
  message wording. Similarly, no test exercises executeMerge with a doc service
  that returns an error. The code path is simple and the lower-level tests are
  convincing, so this is non-blocking; a follow-up task is recommended.
  Reported by: reviewer-conformance, reviewer-testing

---

## Spec Conformance Summary

  FR-001 (gate activates for reviewing status): ✓ — status check in ReviewReportExistsGate.Check
  FR-002 (gate skipped for non-reviewing status): ✓ — early return with Pass for status != "reviewing"
  FR-003 (passes when any report exists, draft or approved): ✓ — len(docs) > 0 check; no status filter applied
  FR-004 (blocked when no report exists): ✓ — len(docs) == 0 returns Blocked
  FR-005 (error message content): ✓ — merge_tool.go includes feature ID, reviewing status statement, numbered resolution steps, non-bypass statement
  FR-006 (Bypassable field on GateResult): ✓ — field added to GateResult struct in gate.go
  FR-007 (ReviewReportExistsGate sets Bypassable: false on block): ✓ — verified in gates.go
  FR-008 (override: true rejected for non-bypassable blocking gate): ✓ — NonBypassableBlockingFailures check in executeMerge before override logic
  FR-009 (existing override behaviour preserved): ✓ — bypassable gates fall through to original override path
  FR-010 (existing gates backfilled with Bypassable: true): ✓ — all six existing gate Check methods updated
  FR-011 (fail-open on doc service error): ✓ — both nil DocSvc and err != nil paths return Pass
  FR-012 (log warning on fail-open): ✓ — log.Printf warning emitted in both nil and error fail-open paths

  AC-001: ✓ covered by TestCheckGates_ReviewingFeature_NoReport_Blocked + TestReviewReportExistsGate_Reviewing_NoReport_Blocked
  AC-002: ~ gate-level coverage only; no merge_tool-level test (see NB-1)
  AC-003: ✓ covered by TestReviewReportExistsGate_Reviewing_WithDraftReport_Passes
  AC-004: ✓ covered by TestReviewReportExistsGate_Reviewing_WithApprovedReport_Passes
  AC-005: ✓ covered by TestReviewReportExistsGate_NotReviewing_AlwaysPasses (parameterised across multiple statuses)
  AC-006: ✓ covered by TestCheckGates_ExistingGates_AreBypassable
  AC-007: ~ gate-level fail-open coverage; no merge_tool-level test (see NB-1)

---

## Notes

- The mergeDocServiceAdapter adapter in merge_tool.go correctly bridges *service.DocumentService
  to merge.DocService without leaking service types into the merge package.
- ReviewReportExistsGate is placed first in DefaultGates(), which is appropriate: for the common
  case (non-reviewing features) it returns Pass immediately with no document service call.
- The DocService interface, DocRecord, and DocFilters types are defined in gate.go alongside
  GateResult. This keeps the merge package self-contained and avoids a circular dependency on
  the service package.
- The hardcoded error message in executeMerge (rather than using the gate's Message field) is
  acceptable given there is currently only one non-bypassable gate. If additional non-bypassable
  gates are introduced in the future, the message construction should be refactored to use the
  gate's own Message field.

---

## Aggregate Verdict: approved_with_followups

Recommended follow-up (non-blocking, can be addressed in a separate task):

  1. [NB-1] — Add merge_tool-level integration tests for AC-002 (override: true rejected
     when ReviewReportExistsGate blocks) and AC-007 (executeMerge fail-open when doc
     service errors). Suggested test names:
       TestExecuteMerge_ReviewingFeature_NoReport_OverrideRejected
       TestExecuteMerge_ReviewingFeature_DocServiceError_FailsOpen
     These tests require a stub DocumentService and a feature entity fixture in reviewing
     status. No changes to production code are needed.