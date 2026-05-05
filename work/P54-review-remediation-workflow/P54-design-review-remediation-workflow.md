# Design: Review Remediation Workflow

**Plan ID:** P54-review-remediation-workflow  
**Status:** Shaping  
**Parent:** P50 (Retrospective Fixes — May 2026)

## Overview

P54 defines the missing workflow bridge between a failed formal review and executable remediation work. It standardizes how blocking findings become a remediation dev-plan, tasks, verification evidence, and a re-review report without weakening the review gate.

## Related Work

- `work/P50-retro-may-2026/P50-report-batch-conformance-review.md` — demonstrated the current manual path from failed review to remediation planning, including BF-1 through BF-10.
- `work/P50-retro-may-2026/P50-dev-plan-review-remediation.md` — first manual remediation dev-plan produced from a failed batch conformance review.
- `work/P40-retro-batch-april-2026/P40-report-plan-review.md` — prior plan-level conformance review with blocking delivery gaps and required actions, showing the same pattern at a larger batch scale.
- `work/P51-handoff-pipeline-unification/P51-design-handoff-pipeline-unification.md` — makes sub-agent handoff context reliable, which any automated remediation task creation will depend on.
- `work/P52-fast-track-orchestration/P52-design-fast-track-orchestration.md` — defines session-start audit and no-implicit-gate behavior, but does not define the review-failure-to-remediation workflow.
- `work/P53-infrastructure-hygiene/P53-design-infrastructure-hygiene.md` — owns status/scope visibility and dirty-work attribution that this workflow should consume rather than reimplement.

Relevant decisions:

- Reviews must produce evidence-backed findings with blocking/non-blocking classification.
- Dev-plans are the artifact used to turn approved scope into executable task graphs.
- Dirty state and entity scope discovery belong to infrastructure/status tooling, not to the remediation workflow itself.

## Problem and Motivation

Kanbanzai has a strong formal review process, but a failed review currently leaves the orchestrator with a manual gap: convert blocking findings into a remediation plan, decide where the remediation work belongs, create tasks, and route the feature or batch back through review. During P50, that bridge was built by hand. The process worked, but it required manual interpretation of BF-1 through BF-10, manual grouping by feature, manual dependency planning, and a separate judgment about F5 scope ambiguity.

This is a recurring workflow, not a P50-only problem. P40 also produced a plan-level review with blocking delivery gaps and a required action list. Without a first-class remediation workflow, each failed review risks becoming an unstructured conversation instead of a traceable implementation path.

The goal is to make failed reviews actionable without weakening the review gate: every blocking finding should map to a remediation task, every remediation task should trace back to a finding, and approval should require a re-review report showing the finding is resolved.

## Goals and Non-Goals

Goals:

1. Convert failed review reports into structured remediation dev-plans.
2. Preserve traceability from each blocking finding to remediation task(s), verification, and re-review evidence.
3. Clarify where remediation work belongs: original feature, batch/plan-level remediation, or a new cross-cutting plan.
4. Keep original review reports immutable and record resolution in re-review artifacts.

Non-goals:

- Not replacing the existing review-code or review-plan skills.
- Not auto-fixing review findings.
- Not duplicating P53's status, dirty-work attribution, or scope inspection infrastructure.
- Not changing the implementation task lifecycle beyond defining how remediation work should be planned and re-reviewed.

## Design

### Remediation workflow entry point

Add a documented workflow, and eventually a tool-assisted flow, for review reports whose aggregate verdict is `fail` or `rejected`.

Inputs:

- review report document ID
- owning entity ID (feature, batch, or plan)
- list of blocking findings, each with finding ID, feature/entity scope, spec anchor, and evidence location

Outputs:

- remediation dev-plan document
- optional remediation tasks under affected features or a dedicated remediation feature
- dependency graph between remediation tasks
- re-review checklist tied back to the original findings

### Finding extraction

The first version can be document-driven rather than fully automated. The orchestrator reads the review report and creates a remediation dev-plan using a required structure:

- Scope: the failed review report and affected feature/spec documents
- Task Breakdown: one task or task group per blocking finding
- Dependency Graph: ordering for fixes that must land before tests or re-review
- Risk Assessment: risks specific to remediation, such as dirty tree mixing or lifecycle state drift
- Verification Approach: tests and re-review steps that close each finding
- Traceability Matrix: original finding ID → remediation task(s)

A later tool can parse structured review reports and propose the same mapping automatically.

### Remediation ownership model

Use the smallest owner that can resolve the findings cleanly:

1. **Single-feature findings:** create remediation tasks under the original feature and transition it to `needs-rework`.
2. **Multi-feature findings in one batch:** create a batch/plan-level remediation dev-plan and per-feature tasks for code changes.
3. **Cross-cutting workflow gaps:** create a new plan, as with this P54 design, when the fix is reusable beyond the failed review.

The workflow should not hide scope cuts inside remediation. If a feature is deferred, the remediation plan must record that as an explicit closure decision.

### Re-review closure

A remediation task is not enough to clear a finding. Closure requires:

- task terminality
- targeted verification passing or documented baseline waiver
- updated review report or re-review report that cites the original finding ID
- owner entity no longer showing review-stage prerequisite violations

The original review report remains immutable evidence. The re-review report records resolution rather than editing the original verdict.

### Integration with existing plans

- **P51:** ensures handoff prompts for remediation implementation tasks use the correct implementer role and skill.
- **P52:** consumes remediation status during session-start audit and avoids stopping after generating an intermediate remediation summary.
- **P53:** provides scope inspection and dirty-work attribution so remediation tasks start from a trustworthy state view.

## Alternatives Considered

### Keep remediation manual

**Idea:** Continue having orchestrators read failed reviews and write ad hoc remediation plans.

**Reject:** P50 showed this works but is fragile. The quality depends heavily on the current orchestrator noticing every finding, grouping correctly, and preserving traceability. A workflow should make the safe path the default.

### Auto-create tasks directly from every blocking finding

**Idea:** Skip the remediation dev-plan and create one task per blocking finding automatically.

**Reject:** Some findings share root causes and should be fixed together; others require sequencing or scope decisions before implementation. A dev-plan step prevents task spam and captures dependency reasoning.

### Put this entirely into P52 fast-track

**Idea:** Treat remediation as another fast-track close-out phase.

**Reject:** Review remediation is not limited to fast-track work. Plan-level and batch-level reviews can fail for ordinary work too. P52 should consume remediation state, but the workflow deserves its own design.

### Put this entirely into P53 infrastructure hygiene

**Idea:** Model failed-review remediation as another status/health issue.

**Reject:** P53 should provide observability primitives, but remediation planning is an orchestration workflow with documents, tasks, dependency graphs, and re-review gates. That is higher-level than infrastructure hygiene.

## Decisions

1. **Create a separate review remediation workflow.**  
   Rationale: The workflow applies beyond fast-track and beyond infrastructure hygiene. Keeping it separate avoids overloading P52 or P53 with review-specific orchestration rules.

2. **Use a remediation dev-plan as the first-class bridge from failed review to implementation.**  
   Rationale: A dev-plan captures dependency ordering and verification strategy before tasks are created. It also gives humans a reviewable artifact before remediation work starts.

3. **Require finding-to-task traceability.**  
   Rationale: A failed review is only cleared when every blocking finding has an explicit remediation path and re-review evidence. Traceability prevents silent scope reduction.

4. **Keep the original review report immutable.**  
   Rationale: The original report is evidence of the failed state. Resolution belongs in a re-review report, not by editing history.

5. **Depend on P53 for status and dirty-work visibility.**  
   Rationale: Scope inspection and dirty-file attribution are shared infrastructure needs. P54 should consume those capabilities rather than duplicating them.

## Dependencies

- P51 improves handoff reliability for remediation task implementation.
- P52 should recognize remediation plans in session-start audit, but P54 is not dependent on fast-track behavior.
- P53 should provide plan/batch scope inspection and dirty-work attribution so remediation starts from a trustworthy state.

## Open Questions

1. Should the first implementation be documentation-only, or should it add a tool action such as `review(action: "remediate")`?
2. Should remediation tasks be created automatically after dev-plan approval, or should the existing `decompose` workflow handle task creation?
3. What is the canonical lifecycle transition for a batch-level failure: affected features to `needs-rework`, batch to `active`, or both?
