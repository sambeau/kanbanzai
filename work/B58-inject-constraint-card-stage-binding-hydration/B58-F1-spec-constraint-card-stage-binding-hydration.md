| Field  | Value |
|--------|-------|
| Date   | 2026-05-08T11:06:29Z |
| Status | approved |
| Author | spec-author |
| Plan | P59 — Roles & Skills Discoverability and Enforcement Remediation |
| Batch | B58 — Inject constraint card and stage-binding hydration |
| Feature | FEAT-01KR3MDJ7AV37 — Constraint card and stage-binding hydration |
| Design | `work/P59-roles-skills-remediation/P59-design-roles-skills-remediation.md` |

## Overview

This specification implements the B1 Inject portion of the design described in `work/P59-roles-skills-remediation/P59-design-roles-skills-remediation.md` (`P59-roles-skills-remediation/design-p59-design-roles-skills-remediation`). It covers a generated constraint card and stage-binding hydration in MCP responses that deliver task or feature context.

The problem is that agents currently reach operational rules through a multi-hop markdown chain. The design requires the highest-leverage constraints, role identity, and stage-binding data to appear directly in the context responses an agent already receives, so an agent cannot miss them by skipping file reads.

## Scope

In scope:

- A typed constraint registry used by the renderer.
- A constraint card renderer that composes a fixed-shape card from role, constraint, and stage-binding inputs.
- Stage-binding hydration for task and feature context responses.
- Injection into `next` task-claim responses and `handoff` responses.
- Validation and tests that make missing role or binding data visible.

Out of scope:

- Rewriting skill markdown or role YAML prose.
- Moving the five high-violation rules into MCP tool invariants; that is covered by B59.
- Generating top-level registry tables; that is covered by B60.
- Runtime discovery wrappers such as `.claude/skills/`; that is covered by B62.
- Adding the card to purely informational responses such as `status` and entity `get`, unless those responses later become task-context carriers.

Related work checked:

- `work/P44-model-routing-agent-launcher/P44-F1-design-prompt-assembly-gate.md` defines non-bypassable prompt assembly and `dispatch_task`. This specification does not duplicate provider dispatch or P44 stage controllers.
- `work/P44-model-routing-agent-launcher/P44-design-feature-execution-pipeline.md` defines server-managed lifecycle execution. This specification only defines task-context payload hydration.
- `work/P59-roles-skills-remediation/P59-report-roles-skills-audit.md` supplies the evidence that response-level injection is needed.

## Functional Requirements

- **REQ-001:** The system must define a canonical constraint registry for the constraint card. Each registry entry must have a stable identifier, a human-readable rule statement, and explicit applicability metadata sufficient for the renderer to decide whether the rule belongs in a role and stage context.
- **REQ-002:** The constraint card renderer must compose the card from typed inputs rather than from hand-written markdown. The typed inputs are the resolved role, the stage binding, and the constraint registry entries selected for that role and stage.
- **REQ-003:** The rendered card must include the resolved role identity, the resolved stage, the stage-bound skill names, the top operational constraints for that role and stage, and the tool-routing reminder that sub-agent prompts must come from the pipeline rather than manual composition.
- **REQ-004:** When `next` claims a task and returns task context, the response must include the rendered constraint card before the detailed task context.
- **REQ-005:** When `handoff` returns a sub-agent prompt, the response must include the rendered constraint card before the detailed task prompt.
- **REQ-006:** Task or feature context responses must include a structured stage-binding payload containing the role name, skill names, effort budget, prerequisites, and sub-agent profile data when those fields are present in `.kbz/stage-bindings.yaml`.
- **REQ-007:** If a resolved role lacks required data needed by the renderer, the system must fail loudly before serving a silently degraded empty card.
- **REQ-008:** If a task or feature resolves to an unknown stage, the rendered output must include an explicit unknown-stage warning and must identify the fallback action: load `.kbz/stage-bindings.yaml` manually.
- **REQ-009:** The constraint card renderer must be covered by golden tests for at least the developing, specifying, dev-planning, and reviewing stage bindings.
- **REQ-010:** The response injection must preserve the existing machine-readable fields returned by `next` and `handoff`; existing consumers must not lose fields because the card was added.

## Non-Functional Requirements

- **REQ-NF-001:** The card must be no more than 25 non-empty lines for any resolved role and stage pair.
- **REQ-NF-002:** The rendered card must be no more than 2,500 bytes for any resolved role and stage pair.
- **REQ-NF-003:** Rendering and stage hydration must add no more than 10ms p95 latency to `next` task claim and `handoff` responses in local unit or integration tests.
- **REQ-NF-004:** Constraint registry validation must be deterministic: repeated validation of the same inputs must produce the same selected constraints in the same order.
- **REQ-NF-005:** The card must not include long examples, anti-pattern bodies, or full skill procedures; those remain in canonical role and skill files.

## Constraints

- The renderer must treat `.kbz/stage-bindings.yaml`, role YAML files, and the new constraint registry as inputs. It must not parse arbitrary prose from SKILL.md files to discover constraints.
- The renderer must be the only component that selects the top-N constraint list for context responses.
- The card must be prepended to human-readable response content, not appended, so it lands in the recency-visible part of the transcript.
- The stage-binding payload must not replace existing `next` or `handoff` fields; it is additive.
- Missing role or binding data must not be silently ignored.
- This specification does not require every MCP response to carry the card. The required injection points are task and feature context delivery responses.
- This specification does not decide or implement P44's provider dispatch mechanism.

## Acceptance Criteria

- **AC-001 (REQ-001, REQ-002):** Given a valid role, stage binding, and constraint registry, when the renderer runs, then the output card is generated entirely from typed inputs and includes no hand-written per-role card fixture.
- **AC-002 (REQ-003):** Given a developing-stage implementer context, when the card is rendered, then it names the role identity, the developing stage, the bound skill, and the selected operational constraints.
- **AC-003 (REQ-004):** Given a ready task, when `next` claims that task, then the returned task context includes the constraint card before the detailed task context.
- **AC-004 (REQ-005):** Given a task accepted by `handoff`, when `handoff` returns the rendered prompt, then the returned prompt begins with the constraint card before task-specific instructions.
- **AC-005 (REQ-006):** Given a stage binding with prerequisites and an effort budget, when task context is returned, then the structured payload contains those stage-binding fields without requiring the caller to read `.kbz/stage-bindings.yaml`.
- **AC-006 (REQ-007):** Given a role missing required renderer data, when the server validates role metadata or renders that role's card, then the operation fails with an actionable error naming the missing field.
- **AC-007 (REQ-008):** Given a task whose stage is unknown, when a context response is assembled, then the rendered card includes `UNKNOWN STAGE` and instructs the agent to load `.kbz/stage-bindings.yaml` manually.
- **AC-008 (REQ-009):** Given the golden test suite, when tests run, then at least developing, specifying, dev-planning, and reviewing expected cards match exactly.
- **AC-009 (REQ-010):** Given an existing consumer of `next` or `handoff`, when it reads the previous machine-readable fields after this change, then those fields are still present with their prior names and value shapes.
- **AC-010 (REQ-NF-001, REQ-NF-002):** Given every supported role and stage pair, when cards are rendered in tests, then no card exceeds 25 non-empty lines or 2,500 bytes.

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Test | Unit test the renderer with fixture role, binding, and constraint registry inputs and assert the card is derived from those inputs. |
| AC-002 | Test | Golden test the developing implementer card for required role, stage, skill, and constraint lines. |
| AC-003 | Integration test | Claim a fixture task with `next` and assert the response content begins with the card. |
| AC-004 | Integration test | Call `handoff` for a fixture task and assert the rendered prompt begins with the card. |
| AC-005 | Test | Assert the context payload contains role, skill, effort budget, prerequisites, and sub-agent profile fields from the stage binding. |
| AC-006 | Test | Load a fixture role missing required metadata and assert validation or rendering fails with the missing field name. |
| AC-007 | Test | Render a context with an unknown stage and assert the unknown-stage warning and manual-load instruction are present. |
| AC-008 | Test | Run renderer golden tests for developing, specifying, dev-planning, and reviewing stages. |
| AC-009 | Regression test | Compare existing `next` and `handoff` response field names before and after injection and assert no previous fields were removed. |
| AC-010 | Test | Render all role and stage combinations and assert line-count and byte-count budgets. |
