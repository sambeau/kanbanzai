# Specification: Fast-Track Architecture

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-04                     |
| Status | Draft                          |
| Author | AI spec-author                 |

## Problem Statement

This specification implements the fast-track architecture described in
`work/P43-fast-track-architecture/P43-design-fast-track-architecture.md` (DOC-`P43-fast-track-architecture/design-p43-design-fast-track-architecture`). The design introduces automated gate validation with risk-tiered approval levels, replacing mechanical human gates with evidence-backed validator roles.

Today, after a design is approved, downstream stage approvals (spec, dev-plan, review) require a human to type "LGTM" — a mechanical gate that adds latency without adding value beyond what an automated structural check can provide. This specification defines three validator roles, risk tiers, enforceable stage transitions, and a validation pipeline that catches document quality issues at the stage where they are cheapest to fix.

**In scope:** Three validator roles (spec-validator, plan-validator, review-gate-validator) with specific validation checks and severity classifications. Four risk tiers (retro_fix, bug_fix, feature, critical) with per-gate automation levels. Enforceable stage transition hooks. Tier inference rules. Fresh-session validation with cycle tracking. The `dispatch_validator` abstraction for forward compatibility with model routing. Human override as an escape hatch.

**Out of scope:** Replacing the design gate (architectural judgment remains human). Evaluating correctness of requirements (validators check structure and traceability, not correctness). Replacing the specialist review panel (review-gate-validator audits the review process, not the code). The `dispatch_validator` implementation routing through P44 model routing (forward-compatible abstraction only — the actual P44 integration is a separate specification).

## Requirements

### Functional Requirements

#### Spec Validator

- **REQ-SPEC-001:** The system must provide a `spec-validator` role that inherits from `base` and validates specification documents for structural completeness and traceability.
- **REQ-SPEC-002:** The spec-validator must execute 10 validation checks: S1 (all required sections present), S2 (overview references parent design), S3 (every requirement has unique REQ-ID), S4 (every REQ-ID appears in Verification Plan), S5 (every acceptance criterion is testable), S6 (acceptance criteria use checkbox format), S7 (no requirement is a disguised implementation instruction), S8 (scope states in-scope AND out-of-scope), S9 (no orphaned requirements not in design), S10 (non-functional requirements have measurable thresholds).
- **REQ-SPEC-003:** Checks S1, S2, S3, S4, S5, and S10 must be classified as blocking. The remaining checks (S6, S7, S8, S9) must be classified as non-blocking.
- **REQ-SPEC-004:** The spec-validator identity must state: "Senior requirements quality auditor. Verify that specifications are complete, testable, and traceable to their parent design. Do not evaluate whether requirements are *correct* — only whether they are *well-formed* and *complete*."

#### Plan Validator

- **REQ-PLAN-001:** The system must provide a `plan-validator` role that inherits from `base` and validates dev-plan documents for completeness, decomposition quality, and traceability.
- **REQ-PLAN-002:** The plan-validator must execute 13 validation checks: D1 (all required sections present), D2 (scope references parent specification), D3 (every task references a spec requirement), D4 (every spec requirement covered by ≥1 task), D5 (no scope drift — tasks not in spec), D6 (dependency graph is acyclic), D7 (no monolithic tasks >3 files or >1 AC), D8 (independent tasks marked parallelisable), D9 (verification maps every AC to producing task), D10 (risk assessment is non-empty), D11 (file paths in deliverables exist or noted as new), D12 (non-functional requirements addressed by tasks), D13 (every task has an actionable description ≥50 words, states what it produces, what inputs it requires, what "done" means beyond the AC).
- **REQ-PLAN-003:** Checks D1, D2, D3, D4, D5, D6, D9, and D13 must be classified as blocking. The remaining checks (D7, D8, D10, D11, D12) must be classified as non-blocking.
- **REQ-PLAN-004:** The plan-validator identity must state: "Senior implementation plan auditor. Verify that dev-plans are complete, well-decomposed, and fully traceable to their parent specification. Do not evaluate whether task ordering is *optimal* — only whether it is *valid* and *complete*."

#### Review Gate Validator

- **REQ-RVW-001:** The system must provide a `review-gate-validator` role that inherits from `reviewer` and audits the review process rather than the code itself.
- **REQ-RVW-002:** The review-gate-validator must execute 8 validation checks: R1 (every reviewer produced structured output with evidence), R2 (no rubber-stamp reviews — zero findings, no evidence), R3 (no severity inflation >40% blocking), R4 (every blocking finding cites a spec requirement), R5 (aggregate verdict consistent with per-dimension outcomes), R6 (deduplication pass was run), R7 (every acceptance criterion covered by ≥1 reviewer), R8 (reviewer selection was adaptive to change scope).
- **REQ-RVW-003:** Checks R1, R2, R4, R5, and R7 must be classified as blocking. The remaining checks (R3, R6, R8) must be classified as non-blocking.
- **REQ-RVW-004:** The review-gate-validator identity must state: "Senior review quality auditor. Verify that a completed review is thorough, evidence-backed, and suitable for auto-approval. Do not re-review the code — audit the review process itself."

#### Validator Rubrics

- **REQ-RUB-001:** Validation checks that rely on LLM classification (S5, S7, D7, D10, D13, R2, R3) must each have a rubric containing: a clear definition of "pass" with 2–3 positive examples, a clear definition of "fail" with 2–3 negative examples, and a "borderline → escalate" pattern for ambiguous cases.
- **REQ-RUB-002:** Validator rubrics must be stored as files in `work/P43-fast-track-architecture/validator-rubrics/`, with one file per validator role containing all rubric definitions for that role's LLM-classification checks.
- **REQ-RUB-003:** Rubrics must be tested on 15–20 real Kanbanzai documents before the validation pipeline is enabled for automated gating.

#### Enforceable Stage Transitions

- **REQ-TRANS-001:** Before a feature transitions from `specifying` to `dev-planning`, the spec-validator must be executed. If any blocking check (S1, S2, S3, S4, S5, S10) fails, the transition must be rejected with the validator findings.
- **REQ-TRANS-002:** Before a feature transitions from `dev-planning` to `developing`, the plan-validator must be executed. If any blocking check (D1, D2, D3, D4, D5, D6, D9, D13) fails, the transition must be rejected with the validator findings.
- **REQ-TRANS-003:** Before a feature transitions from `reviewing` to `done`, the review-gate-validator must be executed. If any blocking check (R1, R2, R4, R5, R7) fails, the transition must be rejected with the validator findings.
- **REQ-TRANS-004:** Transition validator hooks must be defined in the stage bindings configuration, not hardcoded in entity transition logic.
- **REQ-TRANS-005:** Non-blocking check failures must not prevent stage advancement. They must attach findings to the document record and be visible in `status` dashboards.
- **REQ-TRANS-006:** Accumulation of non-blocking findings across multiple features must trigger a quality review signal visible in the project health dashboard.
- **REQ-TRANS-007:** Human override must be available for all validator-enforced transitions via `entity(action: "transition", override: true, override_reason: "...")`. Override usage must be tracked as a metric — if a particular check is consistently overridden, that check's rubric needs refinement.

#### Risk Tiers

- **REQ-TIER-001:** The system must support four risk tiers: `retro_fix`, `bug_fix`, `feature`, and `critical`.
- **REQ-TIER-002:** Each risk tier must define which stage gates are automated vs. human, per the following matrix: `retro_fix` (design: auto, spec: auto, dev-plan: auto, review: conditional), `bug_fix` (design: auto, spec: human, dev-plan: auto, review: auto), `feature` (design: human, spec: auto, dev-plan: auto, review: auto), `critical` (design: human, spec: human, dev-plan: human, review: human).
- **REQ-TIER-003:** Each risk tier must define a maximum auto-cycle count: `retro_fix` = 3, `bug_fix` = 2, `feature` = 2, `critical` = 0.
- **REQ-TIER-004:** The `retro_fix` review gate must be conditional on change type: implementation changes (files outside `work/`, `docs/`, `refs/`) trigger a specialist review panel + review-gate-validator audit; documentation-only changes skip the review gate entirely with an explicit check that no implementation files were modified.
- **REQ-TIER-005:** Risk tiers must be configurable via a `fast_track` configuration block in `.kbz/config.yaml`.
- **REQ-TIER-006:** A `default_tier` setting must be defined in the configuration. Features created without an explicit tier must use the default tier.

#### Tier Inference

- **REQ-INFER-001:** When a feature is created, its tier must be either explicitly set on the entity or inferred from context.
- **REQ-INFER-002:** The tier inference rules must be: (a) if the feature has a retro signal → `retro_fix`, (b) if the feature is a bug entity type → `bug_fix`, (c) if the feature has a `critical` or `security` tag → `critical`, (d) otherwise → `default_tier`.
- **REQ-INFER-003:** An explicitly set tier on the entity must override any inferred tier.

#### Validation Pipeline

- **REQ-PIPE-001:** When a document is registered after authoring (spec or dev-plan), the system must check whether the feature's risk tier allows automated validation for that stage.
- **REQ-PIPE-002:** When automated validation is allowed, the system must dispatch the appropriate validator via `dispatch_validator(role, skill, context)`.
- **REQ-PIPE-003:** On pass: the validator's full report must be registered as a document and the document under validation must be automatically approved.
- **REQ-PIPE-004:** On pass_with_notes: the document must be approved, non-blocking findings must be attached to the document record, and the findings must be surfaced in `status` output.
- **REQ-PIPE-005:** On fail: the document must NOT be approved. The validator findings must be presented to the human for resolution. The feature must remain in its current stage.
- **REQ-PIPE-006:** The pipeline must track the number of auto-validation cycles for each feature. When the cycle count reaches the tier's `max_auto_cycles` cap, the system must escalate to human and must not attempt further automated validation until the human responds.
- **REQ-PIPE-007:** Automated validation must not apply to the design stage regardless of risk tier. Architectural judgment remains human-gated.

#### Session Management

- **REQ-SESS-001:** Validators must always run in fresh sessions. The validator must receive only the document under validation, the parent document, and the validation checklist with rubrics. The validator must not receive the conversation that produced the document.
- **REQ-SESS-002:** Validators must produce two output artifacts: a summary (verdict, blocking/non-blocking finding counts, evidence score, reference to full report) for the orchestrator, and a full report (detailed per-check analysis, evidence citations, uncertain findings) written to the document store. The orchestrator must not hold the full validator output in context.
- **REQ-SESS-003:** The summary artifact must fit within a single tool response (no context bloat for the orchestrator). The full report is retrievable via `doc(action: "content")` for human audit.
- **REQ-SESS-004:** The `dispatch_validator(role, skill, context)` abstraction must use `spawn_agent` at implementation time and must be designed so that future integration with model routing (P44) requires a configuration change, not a code change.

### Non-Functional Requirements

- **REQ-NF-001:** A validator execution for a specification document must complete within 5 tool calls (including document reading, validation, and report writing).
- **REQ-NF-002:** Transition validator hooks must not add more than 2 seconds of latency to `entity(action: "transition")` calls when validators are not required (e.g., transitions where no validator is configured, or when the tier says the gate is human).
- **REQ-NF-003:** When a validator is required and fails, the error message returned to the caller must include: which checks failed, whether each was blocking or non-blocking, and a reference to the full validator report document ID.

## Constraints

- The design stage must remain human-gated regardless of risk tier. Fast-track automation applies to spec, dev-plan, and review stages only.
- The `spec-validator` and `plan-validator` roles must inherit from `base`, not from `spec-author` or `architect`. The `review-gate-validator` must inherit from `reviewer`.
- Validator rubrics must exist as committed files before the validation pipeline is enabled for automated gating. Rubrics are a prerequisite for deployment, not for specification.
- The `dispatch_validator` abstraction must not hardcode `spawn_agent`. It must be designed so that the dispatch mechanism can be replaced with model routing (P44) without modifying validator code.
- Human override is always available as an escape hatch. No validator can permanently block a transition if a human explicitly overrides it.
- Validators must not evaluate correctness. Spec-validator checks whether requirements are well-formed and traceable, not whether they are the right requirements for the design. Plan-validator checks whether tasks are well-decomposed and traceable, not whether the task ordering is optimal. Review-gate-validator checks whether the review process was thorough, not whether the code is correct.
- This specification does NOT cover: the P44 model routing integration itself (separate specification), changes to the design stage gate, or replacement of the specialist review panel.

## Acceptance Criteria

#### Spec Validator

- **AC-SPEC-001 (REQ-SPEC-001, REQ-SPEC-004):** Given a specification document registered in the system, when the spec-validator is dispatched, then it returns a structured verdict containing a pass/pass_with_notes/fail decision and per-check findings.
- **AC-SPEC-002 (REQ-SPEC-002, REQ-SPEC-003):** Given a specification missing the Verification Plan section, when the spec-validator runs, then check S1 fails as blocking and the overall verdict is fail.
- **AC-SPEC-003 (REQ-SPEC-002, REQ-SPEC-003):** Given a specification where every blocking check passes but S7 (implementation instruction detection) fires, then the verdict is pass_with_notes and S7 is reported as a non-blocking finding.
- **AC-SPEC-004 (REQ-SPEC-002):** Given a specification where S5 finds an acceptance criterion that is not testable (subjective language with no observable outcome), then S5 fails as blocking.

#### Plan Validator

- **AC-PLAN-001 (REQ-PLAN-001, REQ-PLAN-004):** Given a dev-plan document registered in the system, when the plan-validator is dispatched, then it returns a structured verdict with per-check findings.
- **AC-PLAN-002 (REQ-PLAN-002, REQ-PLAN-003):** Given a dev-plan with a cyclic dependency graph (task A depends on B, B depends on A), when the plan-validator runs, then D6 fails as blocking and the verdict is fail.
- **AC-PLAN-003 (REQ-PLAN-002, REQ-PLAN-003):** Given a dev-plan where a task description is 30 words and states only "Fix the bug" without input requirements or done criteria, when the plan-validator runs, then D13 fails as blocking.
- **AC-PLAN-004 (REQ-PLAN-002):** Given a dev-plan where every blocking check passes but D7 fires (a task touches 5 files), then the verdict is pass_with_notes and D7 is reported as non-blocking.

#### Review Gate Validator

- **AC-RVW-001 (REQ-RVW-001, REQ-RVW-004):** Given a completed review with structured output from all reviewers, when the review-gate-validator is dispatched, then it returns a structured verdict auditing the review process.
- **AC-RVW-002 (REQ-RVW-002, REQ-RVW-003):** Given a review where one reviewer produced a single line "LGTM" with no evidence, when the review-gate-validator runs, then R2 fails as blocking (rubber-stamp detection).
- **AC-RVW-003 (REQ-RVW-002, REQ-RVW-003):** Given a review where a blocking finding cites no spec requirement, when the review-gate-validator runs, then R4 fails as blocking.

#### Validator Rubrics

- **AC-RUB-001 (REQ-RUB-001, REQ-RUB-002):** Given the `validator-rubrics/` directory, when opened, then it contains one rubric file per validator role, and each LLM-classification check has pass/fail/borderline definitions with examples.
- **AC-RUB-002 (REQ-RUB-003):** Given the rubric files exist, when they are reviewed, then each has been tested against the corpus of Kanbanzai documents and the test results are documented.

#### Enforceable Transitions

- **AC-TRANS-001 (REQ-TRANS-001):** Given a feature in `specifying` stage with a spec that fails S1 (missing required section), when `entity(action: "transition", status: "dev-planning")` is called, then the transition is rejected with an error containing the S1 failure details.
- **AC-TRANS-002 (REQ-TRANS-001):** Given a feature in `specifying` stage with a spec that passes all blocking checks, when the transition to `dev-planning` is attempted, then it succeeds.
- **AC-TRANS-003 (REQ-TRANS-002):** Given a feature in `dev-planning` stage with a dev-plan that fails D13 (vague task description), when the transition to `developing` is attempted, then it is rejected.
- **AC-TRANS-004 (REQ-TRANS-005):** Given a feature in `specifying` stage with a spec that passes all blocking checks but fails S7 (non-blocking), when the transition to `dev-planning` succeeds, then the S7 finding is attached to the document record.
- **AC-TRANS-005 (REQ-TRANS-007):** Given a feature whose transition is blocked by a validator, when `entity(action: "transition", override: true, override_reason: "validator false positive on S7")` is called, then the transition succeeds and the override is recorded.

#### Risk Tiers

- **AC-TIER-001 (REQ-TIER-001, REQ-TIER-002):** Given a `critical` tier feature with a completed spec, when attempting to transition to `dev-planning`, then the transition requires human approval (not auto-validated).
- **AC-TIER-002 (REQ-TIER-002, REQ-TIER-004):** Given a `retro_fix` tier feature with only documentation changes (files under `docs/`), when the feature reaches the review stage, then the review gate is skipped with an explicit "documentation-only" annotation.
- **AC-TIER-003 (REQ-TIER-002, REQ-TIER-004):** Given a `retro_fix` tier feature with implementation changes, when the feature reaches the review stage, then a specialist review panel runs and the review-gate-validator audits it.
- **AC-TIER-004 (REQ-TIER-005, REQ-TIER-006):** Given the `fast_track` configuration block in `.kbz/config.yaml`, when a new feature is created without an explicit tier and no retro/bug/critical signals, then it receives the `default_tier` value from the configuration.

#### Tier Inference

- **AC-INFER-001 (REQ-INFER-002):** Given a new bug entity is created, when its tier is not explicitly set, then it is inferred as `bug_fix`.
- **AC-INFER-002 (REQ-INFER-002):** Given a new feature tagged `security`, when its tier is not explicitly set, then it is inferred as `critical`.
- **AC-INFER-003 (REQ-INFER-003):** Given a new bug entity with an explicitly set tier of `critical`, when tier inference runs, then the explicit `critical` tier is used (not the `bug_fix` inference).

#### Validation Pipeline

- **AC-PIPE-001 (REQ-PIPE-001, REQ-PIPE-002):** Given a `feature` tier feature and a registered spec, when the spec is registered, then the spec-validator is automatically dispatched because the `feature` tier allows auto-spec validation.
- **AC-PIPE-002 (REQ-PIPE-003):** Given a spec-validator returns pass, when the pipeline processes the verdict, then the spec document is automatically approved.
- **AC-PIPE-003 (REQ-PIPE-005):** Given a spec-validator returns fail, when the pipeline processes the verdict, then the spec remains unapproved and the findings are surfaced to the human.
- **AC-PIPE-004 (REQ-PIPE-006):** Given a `feature` tier feature (max_auto_cycles = 2) that has completed 2 fix-validate cycles without passing, when the third cycle would begin, then the system escalates to human and does not dispatch the validator again.
- **AC-PIPE-005 (REQ-PIPE-007):** Given a feature in the designing stage, when the design document is registered, then no validator is dispatched and no auto-approval occurs. The design gate remains human.

#### Session Management

- **AC-SESS-001 (REQ-SESS-001):** Given a spec-validator is dispatched, when it runs, then it reads the specification document, the parent design document, and the validation checklist — and it does not read the conversation log that produced the specification.
- **AC-SESS-002 (REQ-SESS-002):** Given a validator completes its run, when the orchestrator receives the result, then the orchestrator receives only a summary (verdict + counts + document reference), not the full per-check analysis.
- **AC-SESS-003 (REQ-SESS-002):** Given a validator completes its run, when a human inspects the document store, then the full per-check analysis with evidence citations is available via `doc(action: "content")`.
- **AC-SESS-004 (REQ-SESS-004):** Given the `dispatch_validator` implementation, when a reviewer reads the validator code, then the dispatch mechanism (spawn_agent vs. model routing) is abstracted behind a single interface and can be swapped by changing configuration, not validator logic.

#### Non-Functional

- **AC-NF-001 (REQ-NF-001):** Given a specification document of typical size (under 200 lines), when the spec-validator is dispatched, then it completes its full run (read doc, validate, write report) within 5 tool calls.
- **AC-NF-002 (REQ-NF-002):** Given a feature in `specifying` stage with a `critical` tier (human-gated spec), when `entity(action: "transition")` is called, then the check for whether a validator must run completes with no perceptible delay (the tier check is a config lookup, not a validator dispatch).
- **AC-NF-003 (REQ-NF-003):** Given a validator fails a transition, when the error is returned, then the error message contains: the failing check IDs, blocking/non-blocking classification per check, and the validator report document ID.

## Verification Plan

| Requirement(s) | Method | Description |
|----------------|--------|-------------|
| REQ-SPEC-001, REQ-SPEC-004 | Inspection | Verify `spec-validator.yaml` role file exists, inherits from `base`, contains the specified identity text |
| REQ-SPEC-002, REQ-SPEC-003 | Inspection + Test | Verify the spec-validator skill file lists all 10 checks with correct blocking/non-blocking classifications. Unit test: feed a spec with missing Verification Plan → verify S1 blocks |
| REQ-PLAN-001, REQ-PLAN-004 | Inspection | Verify `plan-validator.yaml` role file exists, inherits from `base`, contains the specified identity text |
| REQ-PLAN-002, REQ-PLAN-003 | Inspection + Test | Verify the plan-validator skill file lists all 13 checks. Unit test: feed a dev-plan with cyclic deps → verify D6 blocks |
| REQ-RVW-001, REQ-RVW-004 | Inspection | Verify `review-gate-validator.yaml` role file exists, inherits from `reviewer`, contains the specified identity text |
| REQ-RVW-002, REQ-RVW-003 | Inspection + Test | Verify the review-gate-validator skill file lists all 8 checks. Unit test: feed a rubber-stamp review → verify R2 blocks |
| REQ-RUB-001, REQ-RUB-002 | Inspection | Verify rubric files exist in `work/P43-fast-track-architecture/validator-rubrics/` with pass/fail/borderline definitions for each LLM-classification check |
| REQ-RUB-003 | Inspection | Verify rubric test documentation references 15–20 real Kanbanzai documents |
| REQ-TRANS-001 | Test | Integration test: feature in specifying with failing spec → attempt transition to dev-planning → verify rejection with S1 failure details |
| REQ-TRANS-002 | Test | Integration test: feature in dev-planning with failing dev-plan → attempt transition to developing → verify rejection with D13 failure details |
| REQ-TRANS-003 | Test | Integration test: feature in reviewing with inadequate review → attempt transition to done → verify rejection with R2 failure details |
| REQ-TRANS-004 | Inspection | Verify stage bindings contain `transition_validator` hooks, not hardcoded in entity transition code |
| REQ-TRANS-005 | Test | Unit test: non-blocking finding on a clean spec → transition succeeds, finding attached to document record |
| REQ-TRANS-006 | Test | Integration test: accumulate non-blocking findings across 3 features → verify quality review signal appears in `status` |
| REQ-TRANS-007 | Test | Integration test: blocked transition → override → verify transition succeeds and override is recorded |
| REQ-TIER-001 | Inspection + Test | Verify `fast_track` config schema defines four tiers. Unit test: create feature at each tier → verify tier value is stored |
| REQ-TIER-002 | Test | Unit test: create `critical` tier feature with completed spec → transition to dev-planning requires human (not auto). Unit test: `bug_fix` tier → spec is human-gated, dev-plan auto |
| REQ-TIER-003 | Test | Unit test: verify max_auto_cycles values match design: retro_fix=3, bug_fix=2, feature=2, critical=0 |
| REQ-TIER-004 | Test | Unit test: retro_fix + docs-only changes → review gate skipped. Unit test: retro_fix + code change → review panel + validator |
| REQ-TIER-005, REQ-TIER-006 | Inspection + Test | Verify config block in `.kbz/config.yaml`. Unit test: create feature without tier → verify `default_tier` applied |
| REQ-INFER-001 | Test | Unit test: create feature without explicit tier → verify inference runs |
| REQ-INFER-002 | Test | Unit test: bug entity → `bug_fix`; tagged security → `critical`; no signals → `default_tier` |
| REQ-INFER-003 | Test | Unit test: bug entity with explicit `critical` tier → verify explicit tier used (not inferred `bug_fix`) |
| REQ-PIPE-001 | Test | Integration test: register spec on `feature` tier → verify auto-validation check runs |
| REQ-PIPE-002 | Test | Integration test: auto-validation allowed → verify `dispatch_validator` called with correct role and skill |
| REQ-PIPE-003 | Test | Integration test: validator returns pass → verify spec auto-approved |
| REQ-PIPE-004 | Test | Integration test: validator returns pass_with_notes → verify spec approved, non-blocking findings attached |
| REQ-PIPE-005 | Test | Integration test: validator returns fail → verify spec NOT approved, findings surfaced |
| REQ-PIPE-006 | Test | Integration test: force 2 cycles on feature tier → verify 3rd attempt escalates to human |
| REQ-PIPE-007 | Test | Integration test: register design document → verify no validator dispatched |
| REQ-SESS-001 | Inspection | Review validator skill files: verify the procedure reads only the document, parent document, and checklist |
| REQ-SESS-002, REQ-SESS-003 | Test | Integration test: run validator → verify orchestrator receives summary only; full report accessible via `doc(action: "content")` |
| REQ-SESS-004 | Inspection | Review `dispatch_validator` code: verify it is an interface/abstraction, not a direct `spawn_agent` call |
| REQ-NF-001 | Test | Performance test: run spec-validator on a 150-line spec → verify tool call count ≤ 5 |
| REQ-NF-002 | Test | Performance test: transition call on critical tier feature → verify no validator dispatch latency |
| REQ-NF-003 | Test | Integration test: blocked transition → verify error message format contains check IDs, severity, and report reference |
