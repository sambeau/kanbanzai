# Implementation Plan: Handoff Skill Assembly and Prompt Hygiene

| Field  | Value                                |
|--------|--------------------------------------|
| Date   | 2026-04-24                           |
| Status | Draft                                |
| Author | Claude Opus 4.6 (architect)          |
| Plan   | P30-handoff-skill-assembly-prompt-hygiene |

---

## Scope

This plan implements the requirements defined in `work/spec/p30-handoff-skill-assembly-prompt-hygiene.md` (DOC P30-handoff-skill-assembly-prompt-hygiene/specification-p30-handoff-skill-assembly-prompt-hygiene).

It covers two independent sub-components:

1. **Sub-agent skill routing in `stepLoadSkill`** (FR-001 through FR-006) — a Go code change in `internal/context/pipeline.go` that reads the binding's `sub_agents` block to select the correct skill when the caller's role matches a sub-agent role.
2. **`implement-task/SKILL.md` heredoc replacement** (FR-007 through FR-011) — a documentation change replacing heredoc with `write_file(entity_id: ...)` as the sole recommended file-write method for Go source files.

It does not cover changes to `stage-bindings.yaml`, the `handoff` tool's public API, the legacy 2.0 assembly path, or any skill file other than `implement-task/SKILL.md`.

## Task Breakdown

### Task 1: Sub-agent skill routing in `stepLoadSkill`

- **Description:** Modify `stepLoadSkill` in `internal/context/pipeline.go` to check whether the caller's role prefix-matches any entry in `state.Binding.SubAgents.Roles` before falling through to the primary skill. When a match is found at index `i`, load `SubAgents.Skills[i]`. Return a `pipelineError` if the matched index exceeds the skills array length. Preserve existing fallback behaviour for non-matching roles, empty roles, and nil `SubAgents`.
- **Deliverable:** Updated `stepLoadSkill` function in `internal/context/pipeline.go`.
- **Depends on:** None.
- **Effort:** Small.
- **Spec requirements:** FR-001, FR-002, FR-003, FR-004, FR-005, FR-006, NFR-001, NFR-002, NFR-003, NFR-005.

### Task 2: Unit tests for sub-agent skill routing

- **Description:** Add unit tests to `internal/context/pipeline_test.go` covering all acceptance criteria for sub-component 1: sub-agent role match loads the correct skill (AC-001), orchestrator role falls through to primary (AC-002), nil `SubAgents` loads primary without error (AC-003), out-of-bounds index returns `pipelineError` (AC-004), first prefix match wins with overlapping entries (AC-005), empty role loads primary (AC-006), and deterministic output on repeated calls (AC-012).
- **Deliverable:** New test functions in `internal/context/pipeline_test.go`.
- **Depends on:** Task 1.
- **Effort:** Small.
- **Spec requirements:** AC-001, AC-002, AC-003, AC-004, AC-005, AC-006, AC-012.

### Task 3: Replace heredoc with `write_file` in `implement-task/SKILL.md`

- **Description:** In `.kbz/skills/implement-task/SKILL.md`: replace the "Go source files — use heredoc (primary)" section with a section recommending `write_file(entity_id: ...)` as the sole method for Go source files in a worktree context. Retain `edit_file` as the method for files outside a worktree. Remove all heredoc references including the `GOEOF` delimiter convention and delimiter-collision warning. Update the checklist item to reference `write_file`, not heredoc. Remove the `python3 -c` note if `write_file` covers all worktree write scenarios.
- **Deliverable:** Updated `.kbz/skills/implement-task/SKILL.md`.
- **Depends on:** None.
- **Effort:** Small.
- **Spec requirements:** FR-007, FR-008, FR-009, FR-010, FR-011, NFR-004.

## Dependency Graph

```
Task 1 (no dependencies)
Task 3 (no dependencies)
Task 2 → depends on Task 1
```

Parallel groups: [Task 1, Task 3]
Critical path: Task 1 → Task 2

Tasks 1 and 3 are fully independent — different files, different sub-components, no shared code paths. They can be executed in parallel. Task 2 depends on Task 1 because the tests exercise the code change.

### Merge Schedule

All three tasks modify disjoint file sets with no overlap:

- Task 1: `internal/context/pipeline.go`
- Task 2: `internal/context/pipeline_test.go`
- Task 3: `.kbz/skills/implement-task/SKILL.md`

Single merge cohort — all tasks can merge together without conflict risk.

## Risk Assessment

### Risk: Prefix match order sensitivity

- **Probability:** Low.
- **Impact:** Low.
- **Mitigation:** FR-005 specifies first-match-wins semantics. The current `stage-bindings.yaml` has only one entry per `sub_agents.roles` array (`implementer`), so ordering ambiguity does not arise in practice. The unit test for AC-005 verifies the edge case explicitly.
- **Affected tasks:** Task 1, Task 2.

### Risk: Skill file cache staleness after update

- **Probability:** Low.
- **Impact:** Low.
- **Mitigation:** Skill files are read from disk at assembly time — there is no persistent cache. An MCP server restart picks up the new file content. The design document (Decision 2, Consequences) confirms this is not a persistent risk.
- **Affected tasks:** Task 3.

## Verification Approach

| Acceptance Criterion | Task | Verification Method |
|---------------------|------|---------------------|
| AC-001: `role: implementer-go` loads `implement-task` | Task 2 | Unit test with `developing` binding and sub-agent role |
| AC-002: `role: orchestrator` loads primary skill | Task 2 | Unit test with non-matching role |
| AC-003: nil `SubAgents` loads primary without error | Task 2 | Unit test with minimal binding |
| AC-004: Out-of-bounds index returns `pipelineError` | Task 2 | Unit test with mismatched array lengths |
| AC-005: First prefix match wins | Task 2 | Unit test with overlapping role entries |
| AC-006: Empty role loads primary | Task 2 | Unit test with empty `Input.Role` |
| AC-007: `write_file` section present, no heredoc heading | Task 3 | Inspection of `implement-task/SKILL.md` |
| AC-008: `GOEOF` not present in skill file | Task 3 | `grep -c GOEOF .kbz/skills/implement-task/SKILL.md` returns 0 |
| AC-009: Checklist references `write_file`, not heredoc | Task 3 | Inspection of checklist section |
| AC-010: No `handoff` API change | Task 1 | Inspection — no new fields in `HandoffParams` |
| AC-011: No `stage-bindings.yaml` change | All | `git diff` shows no changes to binding file |
| AC-012: Deterministic skill selection | Task 2 | Unit test with repeated calls |

### Build verification

All tasks must pass:

```
go build ./...
go test ./... -race -count=1
go vet ./...
```
