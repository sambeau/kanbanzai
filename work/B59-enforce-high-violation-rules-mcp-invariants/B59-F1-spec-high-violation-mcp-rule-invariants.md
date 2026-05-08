| Field  | Value |
|--------|-------|
| Date   | 2026-05-08T11:06:29Z |
| Status | Draft |
| Author | spec-author |
| Plan | P59 — Roles & Skills Discoverability and Enforcement Remediation |
| Batch | B59 — Enforce high-violation rules as MCP invariants |
| Feature | FEAT-01KR3MDSZKAFG — High-violation MCP rule invariants |
| Design | `work/P59-roles-skills-remediation/P59-design-roles-skills-remediation.md` |

## Problem Statement

This specification implements the B2 Enforce portion of the design described in `work/P59-roles-skills-remediation/P59-design-roles-skills-remediation.md` (`P59-roles-skills-remediation/design-p59-design-roles-skills-remediation`). It defines the prompt-layer and tool-contract requirements for moving the highest-violation workflow rules out of advisory prose and into MCP tool semantics.

The problem is that agents repeatedly violate rules that are currently written as markdown guidance: manual sub-agent prompt composition, working under unregistered entities, starting tasks with orphaned workflow state, shell-reading workflow state, and skipping artefact gates. This specification makes those rules visible as tool contracts and stable refusal behaviours while preserving the P44 and P56 ownership boundaries for implementation work.

In scope:

- A rule-invariant catalog for the five high-violation rules named in the design.
- Stable error-code and refusal-message contracts for MCP-level enforcement.
- Prompt-layer changes to role, skill, and tool descriptions that point to the invariant contracts rather than duplicating long prose.
- Removal of `spawn_agent` from orchestrator role tool availability once the `dispatch_task` path exists.
- Alignment with P44 for handoff-only dispatch, entity-existence, and commit-before-task enforcement.
- Alignment with P56 for bug gate enforcement.

Out of scope:

- Provider dispatch implementation and stage-controller execution loops owned by P44.
- Reimplementing bug lifecycle gate checks owned by P56.
- The constraint card renderer itself; that is covered by B58.
- Runtime wrapper discovery surfaces; those are covered by B62.

Related work checked:

- `work/P44-model-routing-agent-launcher/P44-F1-design-prompt-assembly-gate.md` requires a non-bypassable dispatch path and prompt assembly gate. This specification relies on that path rather than creating a competing launcher.
- `work/P44-model-routing-agent-launcher/P44-design-feature-execution-pipeline.md` moves execution dispatch into server-managed stages. P59's invariant catalog must not weaken that architecture.
- `work/P56-bug-lifecycle-hardening/P56-F1-spec-bug-lifecycle-gate-enforcement.md` defines bug gate enforcement. This specification limits itself to prompt-layer alignment and tool-description consistency for bug gates.

## Requirements

### Functional Requirements

- **REQ-001:** The system must define a stable invariant catalog containing exactly these high-violation rules for this feature: handoff-only dispatch, registered-entity requirement, commit-orphaned-workflow-state-before-task requirement, no shell reads of `.kbz/state/`, and artefact gate enforcement.
- **REQ-002:** Each invariant must have a stable error code or warning code that tests and callers can assert. The codes must be documented in the corresponding role, skill, or tool-description reference text.
- **REQ-003:** The orchestrator role must not expose direct `spawn_agent` dispatch once a `dispatch_task` tool or equivalent server-managed dispatch path is available.
- **REQ-004:** The canonical dispatch path must internally assemble prompts from the workflow context pipeline. An orchestrator must not be required or allowed to manually compose sub-agent prompts from `next` output.
- **REQ-005:** `next` and `handoff` must refuse to proceed when their requested task, feature, batch, plan, or bug identifier is not registered in Kanbanzai workflow state.
- **REQ-006:** `next` task-claim mode must refuse to claim work when `git status` reports orphaned modified or untracked files under `.kbz/state/`, `.kbz/index/`, or `.kbz/context/`.
- **REQ-007:** Workflow-state shell-read avoidance must be represented as a mandatory warning in task-context surfaces, tool descriptions, and role guidance because the MCP server cannot prevent arbitrary host filesystem reads.
- **REQ-008:** Artefact gate enforcement must remain mandatory for features and must align with P56 bug gate semantics for bugs. P59 text must not describe gates as optional advice.
- **REQ-009:** Gate-related invariants may provide `override` plus `override_reason` only where existing gate conventions allow it. Handoff-only dispatch must not provide an override path.
- **REQ-010:** Existing long-form prose copies of these invariants in orchestrator role and orchestration skill files must be reduced to short cross-references to the stable invariant codes.
- **REQ-011:** Refusal responses must include the invariant code, the refused operation, a short reason, and the next valid action the agent should take.
- **REQ-012:** Tests must cover each invariant at the tool boundary or, for the shell-read warning, at the response-description boundary.

### Non-Functional Requirements

- **REQ-NF-001:** Refusal messages must be deterministic for the same failed invariant and input class so tests can assert exact code and reason fields.
- **REQ-NF-002:** Refusal messages must be concise: no refusal body may exceed 1,200 bytes.
- **REQ-NF-003:** Removing duplicated prose must not remove any rule from all discoverable surfaces. Each invariant must remain discoverable through at least one tool description and one role or skill reference.
- **REQ-NF-004:** The invariant catalog must make P44 and P56 ownership explicit so implementation work is not duplicated across plans.
- **REQ-NF-005:** Existing successful calls that already satisfy the invariants must preserve their previous observable behaviour except for additive warning or metadata fields.

## Constraints

- P59 owns the rule catalog, prompt-layer text, tool-description text, and prose de-duplication. P44 owns dispatch implementation for handoff-only enforcement and prompt assembly. P56 owns bug lifecycle gate checks.
- The handoff-only invariant is a hard architectural invariant and must not accept override.
- Gate overrides, where available, must require a non-empty `override_reason` and must preserve existing audit logging behaviour.
- Shell-read avoidance cannot be enforced against arbitrary terminal or host filesystem tools; the required behaviour is warning and discoverability, not a false promise of prevention.
- Prose reduction must not delete the only copy of a rule. Every removed long-form copy must be replaced by a pointer to the invariant code or canonical rule catalog.
- This specification does not require P44 or P56 to be completed in the same commit as P59 prompt-layer changes, but dependent acceptance criteria cannot be marked complete until their implementation dependency is available.

## Acceptance Criteria

- **AC-001 (REQ-001, REQ-002):** Given the invariant catalog, when it is inspected, then it contains the five named invariants and each has a stable code.
- **AC-002 (REQ-003, REQ-004):** Given the orchestrator role after the dispatch path is available, when available tools are resolved, then direct `spawn_agent` dispatch is absent and the canonical dispatch path uses pipeline-assembled prompts.
- **AC-003 (REQ-005):** Given an unregistered entity identifier, when `next` or `handoff` is invoked with that identifier, then the tool refuses with the registered-entity invariant code and instructs the caller to create or choose a registered entity.
- **AC-004 (REQ-006):** Given modified or untracked files under `.kbz/state/`, `.kbz/index/`, or `.kbz/context/`, when `next` attempts to claim a task, then it refuses with the workflow-state commit invariant code and lists the affected files.
- **AC-005 (REQ-007):** Given a task-context or tool-description surface, when the content is rendered, then it includes a warning not to shell-read `.kbz/state/` and points to MCP workflow tools instead.
- **AC-006 (REQ-008):** Given a feature or bug transition that lacks required artefacts, when the transition is attempted, then the relevant gate refuses according to existing feature gate or P56 bug gate semantics.
- **AC-007 (REQ-009):** Given a handoff-only dispatch violation, when the caller attempts to bypass the canonical dispatch path, then no override parameter can make the operation succeed.
- **AC-008 (REQ-010):** Given `orchestrator.yaml` and orchestration skill text, when duplicate rule prose is reviewed, then long-form duplicated invariant prose has been replaced by short cross-references to the invariant codes.
- **AC-009 (REQ-011):** Given any tool refusal generated by these invariants, when the response is inspected, then it includes invariant code, refused operation, reason, and next valid action.
- **AC-010 (REQ-012):** Given the test suite, when invariant tests run, then each hard invariant has at least one boundary test and the shell-read warning has at least one rendered-surface test.
- **AC-011 (REQ-NF-002):** Given all invariant refusals, when response bodies are measured, then each is at or below 1,200 bytes.

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Inspection | Inspect the invariant catalog and confirm the five rule entries and stable codes. |
| AC-002 | Test | Resolve orchestrator tools after `dispatch_task` availability and assert direct `spawn_agent` is not exposed; assert dispatch uses pipeline output. |
| AC-003 | Integration test | Invoke `next` or `handoff` with an unregistered identifier and assert refusal code and guidance. |
| AC-004 | Integration test | Create controlled orphaned `.kbz/` workflow-state changes and assert `next` task claim refuses with file list. |
| AC-005 | Test | Render task-context and tool-description text and assert the workflow-state shell-read warning appears. |
| AC-006 | Integration test | Attempt feature and bug transitions without required artefacts and assert gate refusals align with feature and P56 semantics. |
| AC-007 | Test | Attempt or inspect bypass paths and assert the handoff-only invariant has no override path. |
| AC-008 | Inspection | Review role and skill files to verify duplicated long-form invariant prose was replaced by invariant-code references. |
| AC-009 | Test | Assert refusal payloads include code, operation, reason, and next action. |
| AC-010 | Test | Run invariant boundary tests and rendered-warning tests. |
| AC-011 | Test | Measure refusal response bodies and assert the 1,200-byte limit. |
