| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-06T16:19:45Z           |
| Status | Draft                          |
| Author | spec-author                     |

# Specification: Fix Sub-Agent Role Routing

**Feature:** FEAT-01KQYZZFGHM99 (Fix Sub-Agent Role Routing)
**Parent Batch:** B1-p51-exec
**Design:** `work/P51-handoff-pipeline-unification/P51-design-handoff-pipeline-unification.md`

## Overview

This specification implements the sub-agent role routing fix described in `work/P51-handoff-pipeline-unification/P51-design-handoff-pipeline-unification.md` (design document for P51, §4). When an orchestrator calls `handoff` without a `role` parameter and the stage binding has `sub_agents` configured, the pipeline must default to the sub-agent role/skill instead of the orchestrator's. This prevents sub-agents from receiving orchestrator training material.

## Scope

**In scope:**
- Change `stepResolveRole` to use `sub_agents.roles[0]` when no caller role is provided and sub_agents are configured
- Change `stepLoadSkill` to use `sub_agents.skills[0]` when no caller role is provided and sub_agents are configured
- Update the `orchestrate-development` skill to document the `role` parameter for explicit role routing

**Out of scope:**
- Changing how explicit `role` parameter resolution works (it already works correctly when provided)
- Changing the `next` tool behavior (it does not use the pipeline for prompt assembly)

## Related Work

Concepts searched: `stepResolveRole`, `stepLoadSkill`, `sub-agent role routing`, `pipeline default role`, `orchestrate-development skill`, `handoff role parameter`.

Entity IDs searched: P50, P51, FEAT-01KQYZZFGHM99.

Prior specifications searched: none found.

**Attestation:** No directly related prior work was found in the corpus. The design document (P51-design-handoff-pipeline-unification, §4 "Sub-agent role routing fix") defines the current behavior and the required change. The P50 fast-track incident (documented in the design §4) provides empirical evidence of the default role/skill resolution causing sub-agents to receive orchestrator material instead of implementer material.

## Problem Statement

When an orchestrator calls `handoff(task_id: "TASK-xxx")` without a `role` parameter and the stage binding has `sub_agents` configured (e.g., `roles: [implementer]`, `skills: [implement-task]`), the pipeline defaults to the binding's primary role/skill (`orchestrator` / `orchestrate-development`) instead of the sub-agent role/skill. This means the sub-agent receives the orchestrator's role (with orchestrator vocabulary and anti-patterns) and the orchestrator's skill (with the 6-phase orchestration procedure and 30+ orchestration anti-patterns) — not the implementer role/skill.

During P50 fast-track, the orchestrator recognized the assembler output as orchestrator training material, discarded all handoff output, and manually composed 12 custom implementer prompts. The pipeline's sub-agent resolution logic at `stepLoadSkill` was never triggered because `state.Input.Role` was empty — the logic only checks for sub-agent skill match when a caller role is explicitly provided.

**Root cause:** Two pipeline steps have incorrect defaults when `state.Input.Role` is empty and `state.Binding.SubAgents != nil`:

1. `stepResolveRole` — falls back to `state.Binding.Roles[0]` (the orchestrator)
2. `stepLoadSkill` — falls back to `state.Binding.Skills[0]` (`orchestrate-development`)

The fix changes both steps to default to the sub-agent's role/skill when sub-agents are declared and no explicit role is provided.

## Functional Requirements

- **FR-001:** When `state.Input.Role` is empty and `state.Binding.SubAgents != nil`, `stepResolveRole` MUST resolve the role to `SubAgents.Roles[0]` instead of `Binding.Roles[0]`.
- **FR-002:** When `state.Input.Role` is empty and `state.Binding.SubAgents != nil`, `stepLoadSkill` MUST load the skill from `SubAgents.Skills[0]` instead of `Binding.Skills[0]`.
- **FR-003:** When `state.Input.Role` is explicitly provided (non-empty) and prefix-matches a sub-agent role, the existing sub-agent role routing behavior MUST be preserved unchanged. The prefix-match check in `stepLoadSkill` MUST still take precedence.
- **FR-004:** When `state.Input.Role` is explicitly provided but does NOT prefix-match any sub-agent role, `stepResolveRole` MUST resolve the caller's role directly (existing behavior preserved).
- **FR-005:** When `state.Binding.SubAgents` is nil (no sub-agents configured), both `stepResolveRole` and `stepLoadSkill` MUST fall back to the binding's primary role/skill (existing behavior preserved for non-sub-agent stages).
- **FR-006:** The `orchestrate-development` skill document MUST explicitly instruct orchestrators to pass `role: "implementer-go"` when calling `handoff` for `spawn_agent` dispatch. The instruction "Always use `handoff(task_id: "TASK-xxx")`" MUST be corrected to include the role parameter.

## Non-Functional Requirements

- **FR-NF-001:** The fix MUST NOT change the output of `handoff` when an explicit `role` parameter is provided. All existing behavior for explicit role routing MUST be preserved.
- **FR-NF-002:** The fix MUST NOT change the pipeline output for stages that have no sub-agents configured (e.g., `designing`, `specifying`, `documenting`).

## Constraints

- This specification does NOT change the `next` tool — `next` returns structured JSON and does not use the pipeline for role/skill resolution.
- The prefix-match logic in `stepLoadSkill` (`strings.HasPrefix(state.Input.Role, subRole)`) MUST NOT change.
- The skill document update is a documentation-only change — it does not affect runtime behavior.

## Acceptance Criteria

- **AC-001 (FR-001):** Given a stage binding with `roles: [orchestrator]` and `sub_agents.roles: [implementer]`, when `handoff(task_id: "TASK-xxx")` is called without a role parameter, then the resolved role is `implementer` (not `orchestrator`).
- **AC-002 (FR-002):** Given a stage binding with `skills: [orchestrate-development]` and `sub_agents.skills: [implement-task]`, when `handoff(task_id: "TASK-xxx")` is called without a role parameter, then the loaded skill is `implement-task` (not `orchestrate-development`).
- **AC-003 (FR-003):** Given a stage binding with `sub_agents.roles: [implementer]` and `sub_agents.skills: [implement-task]`, when `handoff(task_id: "TASK-xxx", role: "implementer-go")` is called, then the resolved role is `implementer-go` and the loaded skill is `implement-task` (prefix match preserves existing behavior).
- **AC-004 (FR-004):** Given a stage binding with `sub_agents.roles: [implementer]`, when `handoff(task_id: "TASK-xxx", role: "reviewer")` is called, then the resolved role is `reviewer` (caller role takes precedence when no prefix match).
- **AC-005 (FR-005):** Given a stage binding with no `sub_agents` and `roles: [spec-author]`, when `handoff(task_id: "TASK-xxx")` is called without a role parameter, then the resolved role is `spec-author` (primary binding fallback preserved).
- **AC-006 (FR-006):** Given the `orchestrate-development` skill file, when searching for `handoff`, the instruction reads "Always use `handoff(task_id: "TASK-xxx", role: "implementer-go")`" with the role parameter included.

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Test | Unit test: pipeline with sub_agents configured, no caller role → assert `state.Role.ID == "implementer"` |
| AC-002 | Test | Unit test: pipeline with sub_agents configured, no caller role → assert `state.Skill.Name == "implement-task"` |
| AC-003 | Test | Unit test: pipeline with sub_agents, caller role "implementer-go" → assert prefix match routes to sub-agent skill |
| AC-004 | Test | Unit test: pipeline with sub_agents, caller role "reviewer" → assert direct role resolution |
| AC-005 | Test | Unit test: pipeline without sub_agents → assert primary binding fallback unchanged |
| AC-006 | Inspection | Verify the `orchestrate-development` skill file includes the corrected `handoff` invocation example |
