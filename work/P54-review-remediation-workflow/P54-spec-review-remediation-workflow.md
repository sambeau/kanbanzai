# Specification: Review Remediation Workflow

**Plan:** P54-review-remediation-workflow
**Parent:** P41-opencode-ecosystem-features  
**Design:** [P54 Design: Review Remediation Workflow](P54-design-review-remediation-workflow.md)  
**Status:** Draft

---

## Overview

This specification defines the review remediation workflow — the standardized bridge between a failed formal review and executable remediation work. When a review report yields an aggregate verdict of `fail`, this workflow converts blocking findings into a remediation dev-plan, remediation tasks with traceability to original findings, verification evidence, and a re-review report that records resolution without mutating the original review.

The first delivery is document-driven (orchestrator-led procedure). An automated phase using `dispatch_task` for finding extraction and dev-plan generation is deferred to a future iteration gated on P44 Phase 1.

---

## Scope

**In scope:**

1. A documented procedure for converting a failed review report into a remediation dev-plan.
2. A required remediation dev-plan structure with finding-to-task traceability.
3. An ownership model for placing remediation work (single-feature, multi-feature/batch-level, or cross-cutting plan).
4. Re-review closure criteria: task terminality, verification evidence, and a re-review report citing original finding IDs.
5. A re-review report format (reusing the existing `report` document type with a `review_remediation` subtype).
6. Integration points with P51 (handoff reliability), P52 (session-start audit), and P53 (scope inspection).

**Out of scope (deferred):**

- Automated finding extraction via `dispatch_task` (gates on P44 Phase 1).
- Automated remediation dev-plan generation (gates on P44 Phase 1).
- Tool-level enforcement of the workflow (v1 is procedural/document-driven).
- Changes to the `review-code` or `review-plan` skills themselves.
- Duplication of P53's scope inspection or dirty-work attribution.

**Explicitly excluded (non-goals):**

- Replacing existing review skills.
- Auto-fixing review findings.
- Mutating original review reports.
- Changing the implementation task lifecycle.

---

## Functional Requirements

### FR-01: Remediation Entry Point

**Requirement:** When a review report's aggregate verdict is `fail`, the orchestrator SHALL be able to initiate a remediation workflow that produces a remediation dev-plan.

**Rationale:** A failed review must produce structured, traceable follow-up work rather than an unstructured conversation.

**Reference:** Design §"Remediation workflow entry point"

**Acceptance Criteria:**

- **AC-01.1:** Given a review report with verdict `fail` and one or more blocking findings, when the orchestrator initiates remediation, then a remediation dev-plan document is registered that cites the original review report ID.
- **AC-01.2:** Given a review report with verdict `pass`, when the orchestrator attempts to initiate remediation, then the system does not require remediation — no remediation dev-plan is produced and the workflow is a no-op.

---

### FR-02: Remediation Dev-Plan Structure

**Requirement:** The remediation dev-plan SHALL include the following sections: Scope, Task Breakdown, Dependency Graph, Risk Assessment, Verification Approach, and Traceability Matrix.

**Rationale:** A standardized structure ensures completeness across different orchestrators and review types, preventing missed findings.

**Reference:** Design §"Finding extraction" and §"Remediation ownership model"

**Acceptance Criteria:**

- **AC-02.1:** Given a failed review report, when a remediation dev-plan is produced, then the dev-plan contains all six required sections.
- **AC-02.2:** Given a remediation dev-plan, when the Traceability Matrix is inspected, then every blocking finding from the original review report is listed with at least one remediation task mapped to it.
- **AC-02.3:** Given a remediation dev-plan, when the Dependency Graph is inspected, then tasks that depend on other remediation work are ordered after their dependencies.

---

### FR-03: Finding-to-Task Traceability

**Requirement:** Every blocking finding in a failed review SHALL be mapped to at least one remediation task in the dev-plan's Traceability Matrix. No blocking finding SHALL be omitted without an explicit closure decision recorded in the dev-plan.

**Rationale:** Traceability prevents silent scope reduction — if a finding is deferred, that decision must be explicit and reviewable.

**Reference:** Design §"Decisions" item 3; §"Remediation ownership model"

**Acceptance Criteria:**

- **AC-03.1:** Given a failed review with N blocking findings, when the remediation dev-plan is produced, then the Traceability Matrix contains N entries — one per finding — each with at least one associated remediation task or an explicit closure decision.
- **AC-03.2:** Given a remediation dev-plan, when a finding is marked as deferred (not to be fixed), then the closure decision includes a rationale and owner.

---

### FR-04: Remediation Ownership Model

**Requirement:** Remediation work SHALL be placed under the smallest scope entity that can resolve the findings cleanly, using one of three ownership models:

1. **Single-feature:** remediation tasks created under the original feature entity.
2. **Batch-level:** a remediation dev-plan at the batch level with per-feature tasks.
3. **Cross-cutting:** a new plan entity for findings that span multiple batches or represent a reusable workflow gap.

**Rationale:** Placing remediation at the wrong scope creates task-management confusion and lifecycle ambiguity.

**Reference:** Design §"Remediation ownership model"

**Acceptance Criteria:**

- **AC-04.1:** Given blocking findings scoped to a single feature, when the remediation dev-plan is produced, then remediation tasks are created under that feature.
- **AC-04.2:** Given blocking findings that span multiple features within one batch, when the remediation dev-plan is produced, then it is registered at the batch level and references the affected features.
- **AC-04.3:** Given blocking findings that are cross-cutting (apply to multiple batches or represent a reusable workflow gap), when the remediation dev-plan is produced, then a new plan entity is created and referenced.

---

### FR-05: Re-Review Closure Criteria

**Requirement:** A blocking finding SHALL be considered resolved only when ALL of the following are satisfied: (a) all associated remediation tasks are terminal, (b) verification evidence is documented, and (c) a re-review report cites the original finding ID and declares it resolved.

**Rationale:** Task completion alone is insufficient — verification and traceable re-review ensure the finding is actually addressed, not just marked done.

**Reference:** Design §"Re-review closure"

**Acceptance Criteria:**

- **AC-05.1:** Given a blocking finding with associated remediation tasks, when the tasks are complete but no verification evidence exists, then the finding is NOT considered resolved.
- **AC-05.2:** Given a blocking finding with completed tasks and verification evidence, when no re-review report cites the finding ID, then the finding is NOT considered resolved.
- **AC-05.3:** Given a blocking finding for which all three closure criteria are met, when the re-review report is approved, then the finding is resolved.

---

### FR-06: Original Review Immutability

**Requirement:** The original review report that produced the blocking findings SHALL remain in its original approved state. Resolution evidence SHALL be recorded in a separate re-review report, not by editing or superseding the original review.

**Rationale:** The original report is evidence of the failed state. Editing it destroys the audit trail.

**Reference:** Design §"Decisions" item 4; §"Re-review closure"

**Acceptance Criteria:**

- **AC-06.1:** Given an approved review report with verdict `fail`, when remediation is performed, then the original report status remains `approved` and its content is unchanged.
- **AC-06.2:** Given a completed remediation, when resolution is recorded, then a new re-review report exists that cites the original review report ID and the finding IDs it resolves.

---

### FR-07: Re-Review Report Format

**Requirement:** The re-review report SHALL reuse the existing `report` document type with a `review_remediation` subtype. It SHALL cite the original review report ID, list each original finding ID with its resolution status, and include verification evidence.

**Rationale:** Reusing the `report` type avoids document type proliferation while preserving the subtype distinction for filtering and auditing.

**Reference:** Design §"Open Questions" item 5; Design §"Re-review closure"

**Acceptance Criteria:**

- **AC-07.1:** Given a completed remediation, when the re-review report is registered, then its document type is `report` with subtype `review_remediation`.
- **AC-07.2:** Given a re-review report, when inspected, then it contains: the original review report ID, a listing of each original finding ID with its resolution status (resolved/deferred), and verification evidence for each resolved finding.

---

### FR-08: Integration with P51 — Handoff Reliability

**Requirement:** When remediation implementation tasks are dispatched to sub-agents via `handoff`, the sub-agent SHALL receive the correct implementer role and skill context (not the orchestrator role).

**Rationale:** P51 fixed a role-routing bug. Remediation tasks must benefit from that fix.

**Reference:** Design §"Integration with existing plans — P51"

**Acceptance Criteria:**

- **AC-08.1:** Given a remediation implementation task dispatched via `handoff(role: "implementer-go")`, when the sub-agent receives its context, then the assigned role is `implementer-go` and the assigned skill is `implement-task`.

---

### FR-09: Integration with P52 — Session-Start Audit

**Requirement:** The remediation workflow SHALL not duplicate P52's session-start audit. When P52 is active, the session-start audit SHALL detect whether a feature is in remediation state and surface that in the status output.

**Rationale:** P52 owns the session-start audit. Remediation consumes it, does not reimplement it.

**Reference:** Design §"Integration with existing plans — P52"

**Acceptance Criteria:**

- **AC-09.1:** Given a feature in remediation state (`needs-rework`), when P52's session-start audit runs, then the audit output identifies the feature as in remediation with a reference to the remediation dev-plan.

---

### FR-10: Integration with P53 — Scope Inspection

**Requirement:** The remediation workflow SHALL consume P53's scope inspection and dirty-work attribution rather than implementing its own status inspection. Before creating remediation tasks, the orchestrator SHALL verify the entity's scope and dirty-state via P53 tooling.

**Rationale:** Scope inspection is shared infrastructure. Duplicating it creates divergence.

**Reference:** Design §"Decisions" item 5; §"Integration with existing plans — P53"

**Acceptance Criteria:**

- **AC-10.1:** Given a feature selected for remediation, when the orchestrator prepares remediation, then the orchestrator queries scope and dirty-state via P53 tooling before creating tasks.
- **AC-10.2:** Given a feature with a dirty working tree, when the orchestrator initiates remediation, then the dirty-state is noted in the remediation dev-plan's Risk Assessment section.

---

## Non-Functional Requirements

### NFR-01: Procedural Clarity

**Requirement:** The documented remediation procedure SHALL be clear enough that two different orchestrators following it from the same failed review produce remediation dev-plans with identical finding-to-task coverage.

**Measurable threshold:** 100% finding coverage — every blocking finding from the original review appears in the Traceability Matrix.

---

### NFR-02: Audit Trail Integrity

**Requirement:** The remediation workflow SHALL preserve a complete audit trail from original review → remediation dev-plan → remediation tasks → verification → re-review report, without any link breaking.

**Measurable threshold:** Every resolved finding can be traced through the full chain in either direction (forward: finding → task → verification → re-review; backward: re-review → finding → original review).

---

### NFR-03: No New Tool Dependencies (v1)

**Requirement:** The document-driven v1 workflow SHALL be executable using existing Kanbanzai tools (`doc`, `entity`, `decompose`, `finish`, `status`) without requiring new MCP tool actions.

**Rationale:** v1 is procedural. Adding tools before the workflow is validated would couple the design to unvalidated automation.

---

## Acceptance Criteria

- [ ] **AC-SPEC-01:** A remediation dev-plan can be produced from a failed review report following the documented procedure, with all six required sections present. (Covers: FR-01, FR-02)
- [ ] **AC-SPEC-02:** Every blocking finding in the source review is accounted for in the Traceability Matrix — either mapped to a remediation task or explicitly deferred with rationale. (Covers: FR-03)
- [ ] **AC-SPEC-03:** Remediation work is placed under the correct ownership scope: single-feature findings create tasks under the original feature; multi-feature findings use a batch-level dev-plan; cross-cutting findings create a new plan. (Covers: FR-04)
- [ ] **AC-SPEC-04:** A re-review report is produced that cites the original review ID, lists each finding with resolution status, and includes verification evidence. Original review report is unchanged. (Covers: FR-05, FR-06, FR-07)
- [ ] **AC-SPEC-05:** When remediation tasks are dispatched via `handoff`, sub-agents receive `implementer-go` role context. (Covers: FR-08)
- [ ] **AC-SPEC-06:** The documented workflow does not duplicate P52's session-start audit or P53's scope inspection — it references them as consumed services. (Covers: FR-09, FR-10)
- [ ] **AC-SPEC-07:** The full audit trail (review → dev-plan → tasks → verification → re-review) is traversable in both directions without broken links. (Covers: NFR-02)
- [ ] **AC-SPEC-08:** The workflow is executable without new tool actions — existing `doc`, `entity`, `decompose`, `finish`, `status` tools are sufficient. (Covers: NFR-03)

---

## Constraints and Exclusions

### Constraints

1. **C-01:** The original review report must be approved before remediation can begin. A draft review has no binding findings.
2. **C-02:** P53 infrastructure hygiene must be available for scope inspection and dirty-work attribution. If P53 is not yet implemented, the orchestrator performs these checks manually.
3. **C-03:** Remediation tasks follow the standard implementation task lifecycle. No special remediation-only task states are introduced.
4. **C-04:** The `decompose` workflow (`propose` → `review` → `apply`) is used for task creation from the remediation dev-plan. No bespoke task creation path.
5. **C-05:** The automated phase (finding extraction via `dispatch_task`) is blocked until P44 Phase 1 delivers a stable `dispatch_task` tool.

### Exclusions

1. **E-01:** This specification does not define changes to the `review-code` or `review-plan` skills. The remediation workflow begins *after* a review is complete and failed.
2. **E-02:** This specification does not define automatic remediation task execution. The orchestrator reviews and approves the remediation dev-plan before tasks are created.
3. **E-03:** This specification does not define a new MCP tool action. The document-driven v1 is procedural.
4. **E-04:** This specification does not cover plan-level conformance review remediation — that is owned by the `review-plan` skill and the `batch-reviewing` stage.

---

## Verification Plan

| Acceptance Criterion | Verification Method | Evidence |
|---|---|---|
| AC-SPEC-01 (dev-plan produced) | Manual demonstration: run a failed review through the workflow | Remediation dev-plan document with all six sections |
| AC-SPEC-02 (finding coverage) | Inspection: diff the blocking findings list against the Traceability Matrix | Zero unmatched findings |
| AC-SPEC-03 (ownership model) | Inspection: verify each task/finding group is under the correct entity | Entity hierarchy matches ownership rules |
| AC-SPEC-04 (re-review report) | Manual demonstration: produce a re-review report and verify original review unchanged | Re-review report document; original review content hash unchanged |
| AC-SPEC-05 (handoff context) | Automated: dispatch a remediation task and inspect sub-agent prompt | Prompt shows `implementer-go` role and `implement-task` skill |
| AC-SPEC-06 (no duplication) | Inspection: read the workflow documentation and confirm it references but does not reproduce P52/P53 logic | No duplicated audit or scope-inspection content |
| AC-SPEC-07 (audit trail) | Manual trace: walk from each finding forward and backward | Unbroken chain for all findings |
| AC-SPEC-08 (no new tools) | Inspection: confirm workflow steps use only existing tool actions | No new tool actions referenced |

---

## Design References

- [P54 Design: Review Remediation Workflow](P54-design-review-remediation-workflow.md) — approved 2026-05-06
- [P51 Design: Handoff Pipeline Unification](../P51-handoff-pipeline-unification/P51-design-handoff-pipeline-unification.md)
- [P52 Design: Fast-Track Orchestration Profile](../P52-fast-track-orchestration/P52-design-fast-track-orchestration.md)
- [P44 Design: Model Routing & Agent Launcher](../P44-model-routing-agent-launcher/P44-design-model-routing-agent-launcher.md)
- [P41 Roadmap: Orchestration & Fast-Track Pipeline](../P41-opencode-ecosystem-features/P41-roadmap-orchestration-fast-track.md)
- [P50 Report: Batch Conformance Review](../P50-retro-may-2026/P50-report-batch-conformance-review.md)
