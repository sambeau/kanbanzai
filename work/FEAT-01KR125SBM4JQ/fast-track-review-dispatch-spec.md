| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-07                     |
| Author | spec-author                     |
| Status | approved |
| Feature| FEAT-01KR125SBM4JQ              |
| Design | P55-orchestrator-context-hygiene/design-p55-design-orchestrator-context-hygiene |

# Specification: Fast-Track Review Dispatch

## Overview

The fast-track profile (`orchestrate-development` fast-track section) never dispatches review sub-agents. Phase 2 Close-Out transitions features to `done` or `reviewing` "as appropriate" but contains no step for specialist review. This means the orchestrator either skips review entirely or self-reviews — both violate the orchestrator's role and the Definition of Done. This feature adds a mandatory review sub-agent dispatch step to fast-track Phase 2.

### Problem

Bug fixes are code changes. Code changes need review. The fast-track profile was designed for speed (no human gates) but the omission of review was an oversight — not a design choice. The `orchestrate-review` skill already dispatches specialist reviewers with clean contexts and structured procedures. It just needs to be invoked from the fast-track path.

### Design References

This specification implements Decision 6 and Component 5 from the approved design:

- **Decision 6:** Dispatch review sub-agents in fast-track close-out
- **Component 5:** Fast-Track Review Dispatch

The design specifies the fix text:

> Before transitioning any feature to `done` or `reviewing`: For each feature that modified source code (not documentation-only changes), dispatch at minimum one review sub-agent. Read `orchestrate-review/SKILL.md` and follow Steps 3–6 (select reviewer, dispatch, collate, aggregate verdict). For `bug_fix` features with ≤5 files changed, a single `reviewer-conformance` sub-agent is sufficient.

### Related Specifications

- **FEAT-01KR12539CXH6 (Orchestrator role hardening)** — The anti-pattern and tool restrictions that prevent the orchestrator from self-reviewing.
- **FEAT-01KR125SBMPQT (Close-out verifier)** — The verifier that checks review happened as part of the DoD.
- **P52-fast-track-orchestration** — The fast-track behavioural profile being modified.

## Scope

### In Scope

- Modifying Phase 2 (Close-Out) of the fast-track profile in `orchestrate-development/SKILL.md` to add a mandatory review dispatch step
- The step requires: for each feature that modified source code, dispatch at minimum one review sub-agent before transitioning
- For `bug_fix` features with ≤5 files changed: dispatch one `reviewer-conformance` sub-agent
- For `retro_fix` features with source changes: dispatch `reviewer-conformance` at minimum
- For features with >5 files changed or non-trivial scope: follow `orchestrate-review` Steps 3–6 for adaptive reviewer selection
- Documentation-only features are exempt from the review requirement

### Out of Scope

- Modifying the `orchestrate-review` skill itself — it already works correctly
- Adding review dispatch to any stage other than fast-track Phase 2 Close-Out
- Runtime enforcement of review dispatch (procedural step, not tool-blocking)
- Any Go code changes — this feature modifies `orchestrate-development/SKILL.md` only

## Functional Requirements

### REQ-001: Review Dispatch Step in Fast-Track Close-Out

Phase 2 (Close-Out) of the fast-track profile in `orchestrate-development/SKILL.md` SHALL include a new step between "Verify all tasks are terminal" (current step 1) and "Transition features" (current step 2). The step SHALL be labeled "Dispatch review sub-agents" or equivalent.

### REQ-002: Minimum Review Requirement

The new step SHALL state that for each feature that modified source code, at minimum one review sub-agent MUST be dispatched before the feature is transitioned out of `developing`.

### REQ-003: Bug Fix Default

For `bug_fix` features with 5 or fewer changed files, the step SHALL specify that a single `reviewer-conformance` sub-agent is sufficient. The sub-agent SHALL be dispatched via `spawn_agent` with role `reviewer-conformance` and skill `review-code`.

### REQ-004: Retro Fix Default

For `retro_fix` features with source code changes, the step SHALL specify that `reviewer-conformance` SHALL be dispatched at minimum.

### REQ-005: Adaptive Reviewer Selection for Larger Changes

For features with more than 5 changed files or features that are not `bug_fix` or `retro_fix` tier, the step SHALL direct the orchestrator to read `orchestrate-review/SKILL.md` and follow Steps 3–6 (select specialist reviewers adaptively, dispatch, collate findings, aggregate verdict).

### REQ-006: Documentation-Only Exemption

The step SHALL state that features with documentation-only changes (no source code modifications) are exempt from the review dispatch requirement.

### REQ-007: Transition Gate

The step SHALL state that the orchestrator MUST NOT transition any feature to `reviewing` or `done` until review findings have been collated and no blocking findings remain.

### REQ-008: Clean Context Requirement

The step SHALL specify that review sub-agents are dispatched in clean contexts (fresh `spawn_agent` sessions), consistent with the `orchestrate-review` procedure.

### REQ-009: Format Consistency

The new step SHALL follow the formatting conventions of the existing fast-track Phase 2 steps — numbered list item, consistent indentation, and markdown style matching the surrounding content.

## Non-Functional Requirements

### NFR-001: No Change to Full Procedure

The full `orchestrate-development` procedure (Phases 1–6) SHALL NOT be modified. Only the fast-track profile section is changed.

### NFR-002: No Semantic Change to Existing Fast-Track Steps

Existing steps in fast-track Phase 2 (verify tasks terminal, transition features, report completion, compaction trigger) SHALL retain their current wording and ordering. The new step is inserted, not a replacement.

## Acceptance Criteria

- [ ] **AC-001:** Fast-track Phase 2 Close-Out in `orchestrate-development/SKILL.md` contains a new step requiring review sub-agent dispatch before feature transition
- [ ] **AC-002:** The step specifies one `reviewer-conformance` sub-agent for `bug_fix` features with ≤5 files
- [ ] **AC-003:** The step specifies at minimum `reviewer-conformance` for `retro_fix` features with source changes
- [ ] **AC-004:** The step directs to `orchestrate-review` Steps 3–6 for features with >5 files or non-bug_fix/retro_fix tier
- [ ] **AC-005:** The step exempts documentation-only features from the review requirement
- [ ] **AC-006:** The step states features MUST NOT be transitioned until review findings are collated and no blocking findings remain
- [ ] **AC-007:** The step specifies clean-context dispatch via `spawn_agent`
- [ ] **AC-008:** The full `orchestrate-development` procedure (Phases 1–6) is unmodified
- [ ] **AC-009:** Existing fast-track Phase 2 steps retain their current wording and ordering
- [ ] **AC-010:** `orchestrate-development/SKILL.md` is valid Markdown with no broken links or syntax

## Verification Plan

| Requirement | Verification Method | Acceptance Criterion |
|-------------|-------------------|---------------------|
| REQ-001 | Manual inspection of fast-track Phase 2 in `orchestrate-development/SKILL.md` | AC-001 |
| REQ-002 | Verify minimum-review language in the new step | AC-001 |
| REQ-003 | Verify bug_fix ≤5 files → single reviewer-conformance | AC-002 |
| REQ-004 | Verify retro_fix → reviewer-conformance minimum | AC-003 |
| REQ-005 | Verify >5 files → orchestrate-review Steps 3–6 reference | AC-004 |
| REQ-006 | Verify documentation-only exemption | AC-005 |
| REQ-007 | Verify transition gate language | AC-006 |
| REQ-008 | Verify clean-context dispatch language | AC-007 |
| NFR-001 | Diff confirms no changes to full procedure | AC-008 |
| NFR-002 | Diff confirms existing steps unchanged | AC-009 |
| REQ-009 | Markdown validity check | AC-010 |
