# Dev Plan: Fix Sub-Agent Role Routing

**Feature:** FEAT-01KQYZZFGHM99
**Specification:** FEAT-01KQYZZFGHM99/spec-p51-spec-fix-sub-agent-role-routing (approved)
**Date:** 2026-05-06

## Overview

Fixes pipeline sub-agent role routing: when handoff is called without an explicit role and the stage binding has sub_agents configured, default to the sub-agent role/skill instead of the orchestrator's. Updates the orchestrate-development skill to document the role parameter. Three tasks.

## Scope

Implements FR-001 through FR-006 and FR-NF-001 through FR-NF-002. Three tasks: pipeline fix, skill documentation, and tests.

## Task Breakdown

### Task 1: Fix stepResolveRole and stepLoadSkill defaults (TASK-01KQZ2ZA2AKMN)
- Change `stepResolveRole` to use `SubAgents.Roles[0]` when no caller role and SubAgents configured
- Change `stepLoadSkill` to use `SubAgents.Skills[0]` when no caller role and SubAgents configured
- Preserve existing behavior for explicit roles, non-prefix-match, and nil SubAgents
- ACs: AC-001 through AC-005

### Task 2: Update orchestrate-development skill for role param (TASK-01KQZ2ZA2G9QA)
- Update `.kbz/skills/orchestrate-development/SKILL.md` to include `role: "implementer-go"` in handoff invocation
- Apply dual-write to `internal/kbzinit/skills/orchestrate-development/SKILL.md`
- AC: AC-006

### Task 3: Tests for sub-agent role routing (TASK-01KQZ2ZA2738D)
- Unit tests verifying: sub-agent default routing, prefix-match preservation, explicit role passthrough, no-sub-agents fallback
- Dependencies: Task 1
- ACs: AC-001 through AC-005

## Dependency Graph

```
T1 (pipeline fix)
  ├── T2 (skill doc) ── independent
  └── T3 (tests) ── depends on T1
```

T1 and T2 can run in parallel. T3 requires T1.

## Risk Assessment

- **Risk:** Pipeline default change breaks non-sub-agent stages → **Low**. Guarded by `state.Binding.SubAgents != nil` check.
- **Risk:** Skill documentation update conflicts with sibling feature FEAT-01KQYZZFGH6DK → **Low**. Both features update different parts of the skill; the doc fixes feature adds the role param, the routing fix feature adds context around it.

## Verification Approach

| AC | Method | Task |
|----|--------|------|
| AC-001 | Unit test: no role + SubAgents → implementer | T1, T3 |
| AC-002 | Unit test: no role + SubAgents → implement-task | T1, T3 |
| AC-003 | Unit test: role "implementer-go" → prefix match | T1, T3 |
| AC-004 | Unit test: role "reviewer" → caller role | T1, T3 |
| AC-005 | Unit test: no SubAgents → primary fallback | T1, T3 |
| AC-006 | Inspection: skill file contains role param | T2 |

## Interface Contracts

- **stepResolveRole** — When `state.Input.Role == ""` and `state.Binding.SubAgents != nil`, resolves to `SubAgents.Roles[0]`.
- **stepLoadSkill** — When `state.Input.Role == ""` and `state.Binding.SubAgents != nil`, loads from `SubAgents.Skills[0]`.
- **orchestrate-development skill** — Updated handoff invocation includes `role: "implementer-go"`.

## Traceability Matrix

| FR | Task |
|----|------|
| FR-001 | T1 |
| FR-002 | T1 |
| FR-003 | T1 |
| FR-004 | T1 |
| FR-005 | T1 |
| FR-006 | T2 |
| FR-NF-001 | T1, T3 |
| FR-NF-002 | T1, T3 |
