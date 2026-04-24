# Specification: Handoff Skill Assembly and Prompt Hygiene

| Field  | Value                                |
|--------|--------------------------------------|
| Date   | 2026-04-24                           |
| Status | Draft                                |
| Author | Claude Opus 4.6 (spec-author)        |
| Plan   | P30-handoff-skill-assembly-prompt-hygiene |

---

## Related Work

### Prior specifications consulted

| Document | Relationship |
|----------|-------------|
| `work/spec/3.0-stage-aware-context-assembly.md` (P16) | Established the 3.0 pipeline assembly model including `PipelineInput.Role` and the `stepLoadSkill` step. FR-008 specifies Step 6 — Skill Loading, which this specification extends with sub-agent routing. |
| `work/spec/p25-impl-workflow-docs.md` (P25) | FR-001 through FR-004 established heredoc as the primary worktree file-write pattern in `implement-task/SKILL.md`. This specification supersedes those requirements with `write_file` as the sole recommended method. |

### Decisions from prior documents constraining this specification

| Decision | Source | Constraint |
|----------|--------|------------|
| WP-5: Binding Registry Is the Decision Table | P17 design §2.5 | The binding registry must remain the authoritative source for role-to-skill mappings. The pipeline must look up decisions from the registry, not encode them in Go logic. |
| Role identity ≠ skill content | P16 spec (FR-007, FR-008) | `stepResolveRole` and `stepLoadSkill` are separate pipeline steps. This separation must be preserved. |

### Deliberate divergence from prior work

This specification replaces the heredoc recommendation established by P25 (FR-001 through FR-004). The P25 spec documented heredoc as primary based on the assumption that delimiter collision was an edge case. Empirical evidence from P25, P27, P28, and P30 design research shows that the failure mode is common-case for Go files containing embedded double-quoted strings. The P30 design (Decision 2) documents the rationale for removal rather than deprecation.

---

## Problem Statement

This specification implements the design described in `work/design/p30-handoff-skill-assembly-prompt-hygiene.md` (DOC P30-handoff-skill-assembly-prompt-hygiene/design-p30-handoff-skill-assembly-prompt-hygiene).

Two defects degrade sub-agent dispatch quality:

1. **Wrong skill loaded for implementer sub-agents.** The `handoff` tool's `stepLoadSkill` (Step 6) always loads the binding's primary skill (`Skills[0]`), regardless of the caller's role. When an orchestrator calls `handoff(task_id: T, role: implementer-go)` for a feature in `developing` status, the sub-agent receives `orchestrate-development` (~3,300 words of orchestrator procedures) instead of `implement-task` (~2,800 words of implementation procedures). The `sub_agents.skills` field in `stage-bindings.yaml` already contains the correct mapping but is never read.

2. **Heredoc recommended as primary in `implement-task/SKILL.md`.** The skill file recommends heredoc as the primary method for writing Go source files. This method fails on any Go file containing embedded double-quoted strings — virtually all Go files. `write_file(entity_id: ...)` is the correct tool for this purpose.

### Scope

**In scope:**
- Modifying `stepLoadSkill` in `internal/context/pipeline.go` to route skill selection through the binding's `sub_agents` block when the caller's role matches a sub-agent role.
- Replacing the heredoc recommendation in `.kbz/skills/implement-task/SKILL.md` with `write_file(entity_id: ...)`.

**Out of scope:**
- Changes to the `handoff` tool's public API (parameters, response format).
- Changes to `stage-bindings.yaml` content or schema.
- Changes to any skill file other than `implement-task/SKILL.md`.
- State persistence performance (P29).
- Lifecycle gate enforcement (P31).

---

## Requirements

### Functional Requirements

#### Sub-component 1: Sub-agent skill routing in `stepLoadSkill`

- **FR-001: Sub-agent role matching.** When `state.Input.Role` is non-empty, `stepLoadSkill` MUST check whether the caller's role matches any entry in `state.Binding.SubAgents.Roles` before falling through to the primary skill. The match uses prefix logic: `strings.HasPrefix(callerRole, subAgentRole)` where `callerRole` is `state.Input.Role` and `subAgentRole` is the binding entry. This is consistent with how the rest of the system handles role variants (e.g., `implementer-go` matching `implementer`).

- **FR-002: Parallel array skill selection.** When a sub-agent role match is found at index `i` in `SubAgents.Roles`, `stepLoadSkill` MUST load the skill at index `i` in `SubAgents.Skills`. The two arrays are parallel: the Nth role maps to the Nth skill.

- **FR-003: Fallback to primary skill.** If no sub-agent role match is found, or if `state.Input.Role` is empty, or if `state.Binding.SubAgents` is nil, `stepLoadSkill` MUST fall through to loading `state.Binding.Skills[0]` as it does today. Existing behaviour for orchestrator callers and for bindings without a `sub_agents` block MUST be preserved.

- **FR-004: Out-of-bounds index error.** If a sub-agent role match is found at index `i` but `i >= len(SubAgents.Skills)`, `stepLoadSkill` MUST return a `pipelineError` at step 6 ("skill-loading") with a message identifying the misconfigured binding and the index mismatch. The error MUST NOT be silently skipped.

- **FR-005: First match wins.** If the caller's role matches multiple entries in `SubAgents.Roles` (e.g., if both `implementer` and `implementer-go` appeared in the list), the first match by array index MUST be used.

- **FR-006: Binding struct already parsed.** The `SubAgents` struct in `internal/binding/model.go` already exposes `Roles []string` and `Skills []string` as parsed fields. No struct changes are required.

#### Sub-component 2: `implement-task/SKILL.md` heredoc replacement

- **FR-007: `write_file` as sole recommended method for Go files.** The "Go source files — use heredoc (primary)" section in `implement-task/SKILL.md` MUST be replaced with a section recommending `write_file(entity_id: "FEAT-xxx", path: "...", content: "...")` as the only method for Go source files in a worktree context.

- **FR-008: `edit_file` for non-worktree files.** The replacement section MUST name `edit_file` as the method for files outside a worktree context. This is existing guidance and is unchanged.

- **FR-009: Heredoc removal.** All heredoc recommendations for Go source files MUST be removed from `implement-task/SKILL.md`. This includes the heredoc example, the `GOEOF` delimiter convention, and the delimiter-collision warning. The removal is complete, not a deprecation with a warning.

- **FR-010: Checklist update.** The checklist item at approximately line 137 that reads "use `terminal` + heredoc for Go files, `python3 -c` for Markdown/YAML, NOT `edit_file`" MUST be replaced with guidance that reads: confirm whether this task runs inside a worktree — if yes, use `write_file(entity_id: ...)` for Go files and Markdown; do NOT use `terminal` + heredoc.

- **FR-011: Retain `python3 -c` only if applicable.** The `python3 -c` note for YAML files MUST be retained only if `write_file` does not cover the use case. If `write_file` covers all remaining worktree write scenarios, the `python3 -c` note MUST be removed.

### Non-Functional Requirements

- **NFR-001: No `handoff` API change.** The `handoff` tool's parameter set and response format MUST remain unchanged. No new parameters are introduced.

- **NFR-002: No `stage-bindings.yaml` change.** The stage bindings file content and schema MUST remain unchanged. The existing `sub_agents.skills: [implement-task]` field is read by the updated pipeline without any YAML modification.

- **NFR-003: Deterministic skill selection.** Given the same `Input.Role` and `Binding`, `stepLoadSkill` MUST always select the same skill. There is no non-determinism in prefix matching or array traversal.

- **NFR-004: No other skill files modified.** Only `implement-task/SKILL.md` is modified. No changes to `orchestrate-development`, `orchestrate-review`, or any other skill file.

- **NFR-005: Backward compatibility.** Existing callers that do not pass a `role` parameter, or that pass a role matching the binding's primary roles (e.g., `orchestrator`), MUST continue to receive the primary skill (`Skills[0]`).

---

## Constraints

- The `Binding` struct in `internal/binding/model.go` already parses `SubAgents.Roles` and `SubAgents.Skills`. No struct extension is needed.
- The `PipelineInput.Role` field already carries the caller's role into the pipeline. No input changes are needed.
- This specification does NOT cover the legacy 2.0 assembly path in `handoff_tool.go` (`buildLegacyResponse`). That path uses the `role` parameter differently and is not the source of this defect.
- This specification does NOT cover changes to orchestration skills or their dispatch instructions. Orchestrators continue to call `handoff(task_id: T, role: implementer-go)` unchanged.

---

## Acceptance Criteria

- **AC-001 (FR-001, FR-002):** Given a feature in `developing` status and a `handoff` call with `role: implementer-go`, when `stepLoadSkill` executes, then the loaded skill is `implement-task` (not `orchestrate-development`), because `implementer-go` prefix-matches `implementer` in `SubAgents.Roles[0]`, selecting `SubAgents.Skills[0]` which is `implement-task`.

- **AC-002 (FR-003):** Given a feature in `developing` status and a `handoff` call with `role: orchestrator` (or no role), when `stepLoadSkill` executes, then the loaded skill is `orchestrate-development` (the primary skill `Skills[0]`), because `orchestrator` does not match any entry in `SubAgents.Roles`.

- **AC-003 (FR-003):** Given a stage binding with no `sub_agents` block (e.g., `specifying`), when `stepLoadSkill` executes with any role, then the primary skill is loaded. The absence of `SubAgents` does not cause an error.

- **AC-004 (FR-004):** Given a binding where `SubAgents.Roles` has 2 entries but `SubAgents.Skills` has 1 entry, when `stepLoadSkill` matches the caller's role at index 1, then a `pipelineError` at step 6 is returned mentioning the index mismatch.

- **AC-005 (FR-005):** Given a binding where `SubAgents.Roles` contains both `implementer` at index 0 and `implementer-go` at index 1, when `stepLoadSkill` is called with `role: implementer-go`, then the skill at index 0 is selected (first prefix match wins).

- **AC-006 (FR-001, NFR-005):** Given a `handoff` call with an empty `role` parameter, when `stepLoadSkill` executes, then the primary skill `Skills[0]` is loaded. Empty role does not trigger sub-agent matching.

- **AC-007 (FR-007, FR-009):** The section heading "Go source files — use heredoc (primary)" no longer exists in `implement-task/SKILL.md`. In its place, a section recommends `write_file(entity_id: ...)` as the sole method for Go source files in a worktree context.

- **AC-008 (FR-009):** The string `GOEOF` does not appear in `implement-task/SKILL.md`. The heredoc example and delimiter-collision warning have been removed.

- **AC-009 (FR-010):** The checklist in `implement-task/SKILL.md` contains an item that mentions `write_file(entity_id: ...)` for Go files and does NOT mention heredoc.

- **AC-010 (NFR-001):** The `handoff` tool's MCP parameter schema is unchanged. No new parameters exist.

- **AC-011 (NFR-002):** The file `.kbz/stage-bindings.yaml` is not modified by this plan.

- **AC-012 (NFR-003):** Calling `stepLoadSkill` twice with the same `PipelineState` produces the same skill selection both times.

---

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Test | Unit test: `stepLoadSkill` with `role: implementer-go` and a `developing` binding loads `implement-task` |
| AC-002 | Test | Unit test: `stepLoadSkill` with `role: orchestrator` loads primary skill `orchestrate-development` |
| AC-003 | Test | Unit test: `stepLoadSkill` with nil `SubAgents` loads primary skill without error |
| AC-004 | Test | Unit test: `stepLoadSkill` returns `pipelineError` step 6 when matched index exceeds `SubAgents.Skills` length |
| AC-005 | Test | Unit test: `stepLoadSkill` with overlapping prefix entries selects the first match |
| AC-006 | Test | Unit test: `stepLoadSkill` with empty role loads primary skill |
| AC-007 | Inspection | Verify `implement-task/SKILL.md` contains "write_file" section and no "heredoc (primary)" heading |
| AC-008 | Inspection | `grep -c GOEOF .kbz/skills/implement-task/SKILL.md` returns 0 |
| AC-009 | Inspection | Checklist item references `write_file(entity_id: ...)` and does not reference heredoc |
| AC-010 | Inspection | `handoff` tool parameter struct has no new fields compared to pre-P30 |
| AC-011 | Inspection | `git diff` shows no changes to `.kbz/stage-bindings.yaml` |
| AC-012 | Test | Unit test: two consecutive calls to `stepLoadSkill` with identical state produce the same skill |