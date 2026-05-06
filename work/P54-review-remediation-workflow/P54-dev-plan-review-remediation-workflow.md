# Dev-Plan: Review Remediation Workflow (v1 — Document-Driven)

| Field  | Value                                 |
|--------|---------------------------------------|
| Date   | 2026-05-06                            |
| Status | Draft                                 |
| Author | sambeau (architect)                   |

---

## Scope

This plan implements the document-driven v1 of the review remediation workflow as specified in [`P54-spec-review-remediation-workflow.md`](P54-spec-review-remediation-workflow.md), producing:

1. A documented remediation procedure for orchestrators.
2. A remediation dev-plan template matching the six-section structure (FR-02).
3. A re-review report template reusing the `report` type with `review_remediation` subtype (FR-07).
4. An ownership model decision tree (FR-04).
5. A walkthrough validation of the full workflow end-to-end.

**Out of scope for this plan:** Code changes, new MCP tool actions, automated finding extraction (gated on P44 Phase 1), and integration implementation with P52/P53 (those are consumed, not built here).

---

## Task Breakdown

### Task 1: Document the Remediation Workflow Procedure

- **Description:** Write the orchestrator-facing procedure document for converting a failed review into a remediation dev-plan. Covers: entry point (when to initiate), reading the review report for blocking findings, producing the remediation dev-plan, placing remediation work under the correct ownership scope, creating tasks via `decompose`, executing remediation, producing a re-review report, and closing out.
- **Deliverable:** `work/P54-review-remediation-workflow/P54-procedure-review-remediation.md`
- **Depends on:** None
- **Effort:** Medium
- **Spec requirement:** FR-01, FR-04, FR-05, FR-06

### Task 2: Define the Remediation Dev-Plan Template

- **Description:** Create a template for remediation dev-plans containing the six required sections: Scope, Task Breakdown, Dependency Graph, Risk Assessment, Verification Approach, and Traceability Matrix. Include inline guidance for the orchestrator on how to populate each section from a review report.
- **Deliverable:** `work/P54-review-remediation-workflow/P54-template-remediation-dev-plan.md`
- **Depends on:** Task 1 (template structure aligns with procedure)
- **Effort:** Small
- **Spec requirement:** FR-02, FR-03

### Task 3: Define the Re-Review Report Template

- **Description:** Create a re-review report template reusing the existing `report` document type with `review_remediation` subtype. Template includes: original review report citation, per-finding resolution status table, verification evidence per finding, and aggregate resolution verdict.
- **Deliverable:** `work/P54-review-remediation-workflow/P54-template-re-review-report.md`
- **Depends on:** Task 1 (template structure aligns with procedure)
- **Effort:** Small
- **Spec requirement:** FR-07

### Task 4: Document the Ownership Model Decision Tree

- **Description:** Create a decision tree document that guides the orchestrator through the three ownership models: single-feature, batch-level, and cross-cutting plan. Include decision criteria, entity-creation steps for each path, and examples from P50 remediation experience.
- **Deliverable:** `work/P54-review-remediation-workflow/P54-guide-ownership-model.md`
- **Depends on:** Task 1 (decision points must align with procedure)
- **Effort:** Small
- **Spec requirement:** FR-04

### Task 5: Integration Verification — Workflow Walkthrough

- **Description:** Run a real failed review through the documented workflow end-to-end using the templates and guides produced in Tasks 1–4. Verify: all six dev-plan sections are produced, every finding maps to a task or deferral, ownership model is correctly applied, re-review report cites original findings, and the audit trail is traversable in both directions.
- **Deliverable:** Walkthrough report at `work/P54-review-remediation-workflow/P54-report-walkthrough.md` plus a remediation dev-plan and re-review report produced during the walkthrough.
- **Depends on:** Task 1, Task 2, Task 3, Task 4
- **Effort:** Medium
- **Spec requirement:** AC-SPEC-01, AC-SPEC-02, AC-SPEC-03, AC-SPEC-04, AC-SPEC-07

---

## Dependency Graph

```
Task 1 (no dependencies)
Task 2 → depends on Task 1
Task 3 → depends on Task 1
Task 4 → depends on Task 1
Task 5 → depends on Task 1, Task 2, Task 3, Task 4

Parallel groups: [Task 2, Task 3, Task 4] after Task 1 completes
Critical path: Task 1 → Task 5
```

---

## Risk Assessment

### Risk: P53 Not Yet Available for Scope Inspection

- **Probability:** High (P53 hasn't shipped)
- **Impact:** Low (the spec permits manual scope inspection as fallback per C-02)
- **Mitigation:** The procedure document (Task 1) includes manual scope-check steps that do not depend on P53 tooling. When P53 ships, a follow-up task can swap in P53 tool calls.
- **Affected tasks:** Task 1, Task 5

### Risk: No Real Failed Review Available for Walkthrough

- **Probability:** Medium (depends on whether P50 or other reviews are in failed state)
- **Impact:** Medium (walkthrough quality depends on realistic findings)
- **Mitigation:** Use the P50 batch conformance review report as a realistic example if no live failed review exists. The walkthrough report notes whether a live or historical review was used.
- **Affected tasks:** Task 5

### Risk: Workflow Procedure Too Abstract Without Tool Support

- **Probability:** Low (the procedure is concrete — orchestrator reads review, produces document, creates tasks)
- **Impact:** Medium (if orchestrators find the procedure ambiguous, adoption suffers)
- **Mitigation:** Task 5 walkthrough catches ambiguity. The procedure includes concrete examples from P50 experience.
- **Affected tasks:** Task 1

---

## Verification Approach

| Acceptance Criterion | Verification Method | Producing Task |
|---|---|---|
| AC-SPEC-01: Remediation dev-plan with all six sections | Inspection of walkthrough output | Task 5 |
| AC-SPEC-02: Every blocking finding accounted for | Diff findings list against Traceability Matrix | Task 5 |
| AC-SPEC-03: Correct ownership scope | Inspection of entity hierarchy in walkthrough | Task 5 |
| AC-SPEC-04: Re-review report produced, original unchanged | Compare content hashes; inspect re-review report | Task 5 |
| AC-SPEC-06: No duplication of P52/P53 logic | Inspection of procedure document | Task 1 |
| AC-SPEC-07: Full audit trail traversable | Manual trace forward and backward | Task 5 |
| AC-SPEC-08: No new tool actions required | Inspection of procedure document for tool references | Task 1 |

Note: AC-SPEC-05 (handoff context) is verified by P51's existing test suite — no remediation-specific task needed.
