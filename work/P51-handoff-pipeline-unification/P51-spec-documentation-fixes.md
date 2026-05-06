| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-06T16:19:45Z           |
| Status | Draft                          |
| Author | spec-author                     |

# Specification: Documentation Fixes for Handoff Pipeline

**Feature:** FEAT-01KQYZZFGH6DK (Documentation Fixes for Handoff Pipeline)
**Parent Batch:** B1-p51-exec
**Design:** `work/P51-handoff-pipeline-unification/P51-design-handoff-pipeline-unification.md`

## Overview

This specification implements the documentation fixes described in `work/P51-handoff-pipeline-unification/P51-design-handoff-pipeline-unification.md` (design document for P51, §1.6.3 and §4). It documents the `finish` tool's 500-character summary limit in the tool description and updates the `orchestrate-development` skill to instruct orchestrators to pass `role: "implementer-go"` when calling `handoff`.

## Scope

**In scope:**
- Add "(max 500 characters)" to the `finish` tool's `summary` parameter description
- Update the `orchestrate-development` skill to document `handoff(task_id: "TASK-xxx", role: "implementer-go")`

**Out of scope:**
- Changing the 500-character limit itself
- Changing any other tool descriptions
- Changing the pipeline default role/skill resolution (that is FEAT-01KQYZZFGHM99)

## Related Work

Concepts searched: `finish summary limit`, `tool description`, `MCP parameter description`, `max 500 characters`, `orchestrate-development skill`, `handoff role parameter`.

Entity IDs searched: P50, P51, FEAT-01KQYZZFGH6DK, FEAT-01KQYZZFGHM99 (sibling feature that also updates the orchestrate-development skill).

Prior specifications searched: none found.

**Attestation:** No directly related prior work was found in the corpus. The design document (P51-design-handoff-pipeline-unification, §4 "Sub-agent role routing fix" and §1.6.3 "finish summary limit") identifies both documentation gaps. The sibling feature FEAT-01KQYZZFGHM99 (Fix Sub-Agent Role Routing) also touches the orchestrate-development skill — this spec's skill documentation changes are coordinated with that feature but independent in implementation.

## Problem Statement

Two documentation gaps were discovered during P50 fast-track:

1. **`finish` summary limit is undocumented in the tool description.** The `finish` tool enforces a 500-character limit on the `summary` parameter. The limit is surfaced only in the runtime error message — the tool description says "Brief description of what was accomplished" with no mention of the character limit. During P50 close-out, the orchestrator hit this limit twice and had to learn it by failing. The fix is a one-line change: add "(max 500 characters)" to the summary parameter's MCP description.

2. **`orchestrate-development` skill has incorrect `handoff` invocation.** The skill instructs orchestrators to call `handoff(task_id: "TASK-xxx")` without a `role` parameter. The sibling feature FEAT-01KQYZZFGHM99 fixes the pipeline default so sub-agents get the correct role even without an explicit `role` parameter, but the skill should still document the correct invocation with the role parameter. This documents the explicit path as the recommended practice and serves as a fallback discovery mechanism.

## Functional Requirements

- **FR-001:** The `finish` tool's `summary` parameter description MUST include the text "(max 500 characters)". The full description MUST read: "Brief description of what was accomplished (max 500 characters)".
- **FR-002:** The `orchestrate-development` skill file (`.kbz/skills/orchestrate-development/SKILL.md`) MUST include the instruction to pass `role: "implementer-go"` when calling `handoff` for `spawn_agent` dispatch. The corrected instruction MUST read: "Always use `handoff(task_id: "TASK-xxx", role: "implementer-go")` to generate sub-agent prompts."
- **FR-003:** The `orchestrate-development` skill update MUST follow the dual-write rule: the corresponding file at `internal/kbzinit/skills/orchestrate-development/SKILL.md` MUST receive the same change in the same commit.

## Non-Functional Requirements

- **FR-NF-001:** The `finish` tool description change MUST NOT alter tool behavior — it is a documentation-only change. All existing `finish` tests MUST continue to pass.
- **FR-NF-002:** The skill documentation changes MUST NOT alter runtime behavior. Skill files are read as Markdown content by the pipeline; the updated instruction is prose, not code.

## Constraints

- The 500-character limit itself MUST NOT change — only the documentation of it.
- The dual-write rule (AGENTS.md §"Dual-write rule for skill changes") applies: `.agents/skills/kanbanzai-*/SKILL.md` ↔ `internal/kbzinit/skills/*/SKILL.md`. The orchestrate-development skill is a task-execution skill under `.kbz/skills/`, not under `.agents/skills/kanbanzai-*/`. Verify the skill's location before applying the dual-write rule.
- The `orchestrate-development` skill update MUST be coordinated with FEAT-01KQYZZFGHM99 (Fix Sub-Agent Role Routing) but implemented independently — no code dependency between the two features.

## Acceptance Criteria

- **AC-001 (FR-001):** Given the `finish` tool definition, when the `summary` parameter description is inspected, then it reads "Brief description of what was accomplished (max 500 characters)".
- **AC-002 (FR-002):** Given the `orchestrate-development` skill file, when searching for `handoff`, then the instruction includes `role: "implementer-go"` as a parameter.
- **AC-003 (FR-003):** Given the dual-write skill file at `internal/kbzinit/skills/`, when the orchestrate-development skill is checked, then it matches the `.kbz/skills/` version in the handoff instruction.
- **AC-004 (FR-NF-001):** Given the updated `finish` tool description, when running `go test ./internal/mcp/... -run TestFinish`, then all tests pass.
- **AC-005 (FR-NF-002):** Given the updated skill file, when the pipeline loads the `orchestrate-development` skill, then the output is semantically unchanged (the prose instruction is the only difference).

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Inspection | Read `internal/mcp/finish_tool.go`, verify `summary` parameter description |
| AC-002 | Inspection | grep `orchestrate-development/SKILL.md` for `handoff` → verify role parameter present |
| AC-003 | Inspection | Compare `.kbz/skills/` and `internal/kbzinit/skills/` versions of the skill |
| AC-004 | Test | Run `go test ./internal/mcp/... -run TestFinish` — all pass |
| AC-005 | Test | Run pipeline test that loads orchestrate-development skill, verify no regression |
