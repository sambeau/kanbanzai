# Review: Handoff Skill Assembly Fix

| Field     | Value                                      |
|-----------|--------------------------------------------|
| Feature   | FEAT-01KQ049NBSQAR — handoff-skill-assembly-fix |
| Plan      | P30-handoff-skill-assembly-prompt-hygiene   |
| Date      | 2026-04-25                                 |
| Reviewer  | Claude Opus 4.6 (orchestrator)             |
| Verdict   | **approved**                               |

---

## Summary

P30 delivers two independent fixes: (1) sub-agent skill routing in `stepLoadSkill` so implementer sub-agents receive the correct skill (`implement-task`) instead of the orchestrator skill (`orchestrate-development`), and (2) replacement of the heredoc recommendation in `implement-task/SKILL.md` with `write_file(entity_id: ...)` as the sole worktree file-write method.

All 12 acceptance criteria pass. No blocking or non-blocking findings were identified. The implementation is minimal, correctly scoped, and fully tested.

---

## Reviewers Dispatched

| Reviewer | Scope | Verdict |
|----------|-------|---------|
| reviewer-conformance | All files — spec conformance | pass |
| reviewer-quality | `internal/context/pipeline.go` — implementation quality | pass |
| reviewer-testing | `internal/context/pipeline_test.go` — test adequacy | pass |
| reviewer-security | All files — security | pass_with_notes |

---

## Review Units

### Unit 1: Sub-agent skill routing (`internal/context/pipeline.go`)

**Files:** `internal/context/pipeline.go`

The `stepLoadSkill` function now checks `state.Binding.SubAgents` when `state.Input.Role` is non-empty. It iterates `SubAgents.Roles` using `strings.HasPrefix(callerRole, subAgentRole)` for prefix matching. On match at index `i`, it loads `SubAgents.Skills[i]`. Out-of-bounds indices return a step-6 `pipelineError`. Fallback to `Skills[0]` is preserved for non-matching roles, empty roles, and nil `SubAgents`.

### Unit 2: Skill routing tests (`internal/context/pipeline_test.go`)

**Files:** `internal/context/pipeline_test.go`

Seven new `TestStepLoadSkill_*` tests cover the full routing matrix:

| Test | AC |
|------|----|
| `TestStepLoadSkill_SubAgentRoleMatch` | AC-001 |
| `TestStepLoadSkill_OrchestratorFallsThrough` | AC-002 |
| `TestStepLoadSkill_NilSubAgents` | AC-003 |
| `TestStepLoadSkill_SubAgentOutOfBounds` | AC-004 |
| `TestStepLoadSkill_FirstPrefixMatchWins` | AC-005 |
| `TestStepLoadSkill_EmptyRoleLoadsPrimary` | AC-006 |
| `TestStepLoadSkill_Deterministic` | AC-012 |

All 10 `TestStepLoadSkill_*` tests pass (3 pre-existing + 7 new).

### Unit 3: Skill file update (`.kbz/skills/implement-task/SKILL.md`)

**Files:** `.kbz/skills/implement-task/SKILL.md`

The "Worktree File Editing" section now recommends `write_file(entity_id: ...)` as the sole method for all worktree file types. `edit_file` is retained for non-worktree files. All heredoc references (`GOEOF`, delimiter-collision warning, heredoc examples) and `python3 -c` guidance have been removed. The checklist item references `write_file(entity_id: ...)` and does not mention heredoc.

---

## Per-Dimension Verdicts

### Spec Conformance
**Outcome:** pass

Evidence:
- AC-001: `TestStepLoadSkill_SubAgentRoleMatch` — `role: implementer-go` prefix-matches `implementer` in `SubAgents.Roles[0]`, loads `implement-task-go` from `SubAgents.Skills[0]`
- AC-002: `TestStepLoadSkill_OrchestratorFallsThrough` — `role: orchestrator` does not match `implementer`, falls through to primary skill `implement-task`
- AC-003: `TestStepLoadSkill_NilSubAgents` — nil `SubAgents` loads primary skill without error
- AC-004: `TestStepLoadSkill_SubAgentOutOfBounds` — empty `SubAgents.Skills` with matching role returns step-6 `pipelineError` mentioning "out of range"
- AC-005: `TestStepLoadSkill_FirstPrefixMatchWins` — `implementer-go` matches `impl` at index 0 before `implementer` at index 1, selects `skill-first`
- AC-006: `TestStepLoadSkill_EmptyRoleLoadsPrimary` — empty role skips sub-agent routing, loads primary skill
- AC-007: `implement-task/SKILL.md` section heading is "Worktree files — use `write_file` (all file types)", no heredoc heading exists
- AC-008: `grep -c GOEOF .kbz/skills/implement-task/SKILL.md` returns 0
- AC-009: Checklist item reads "use `write_file(entity_id: ...)` for all file types, NOT `edit_file`"
- AC-010: No new parameters added to `handoff` tool — verified by inspection of unchanged `HandoffParams`
- AC-011: `git diff` shows no changes to `.kbz/stage-bindings.yaml`
- AC-012: `TestStepLoadSkill_Deterministic` — 5 consecutive calls with identical state produce the same skill

### Implementation Quality
**Outcome:** pass

Evidence:
- The routing logic is 15 lines added to `stepLoadSkill`, localized and minimal
- Fallback path is cleanly preserved via the existing `skillName == ""` check
- Error handling for the only new misconfiguration case (out-of-bounds index) is explicit and informative
- No unnecessary abstractions, helpers, or refactoring introduced
- Zero diagnostics (warnings or errors) on `pipeline.go`

### Test Adequacy
**Outcome:** pass

Evidence:
- All 7 new tests are parallel, isolated, and use dedicated fixtures
- Each test targets a single AC with clear assertion
- Error-path tests (`SubAgentOutOfBounds`) verify both error presence and error message content
- Edge cases covered: empty role, nil sub-agents, overlapping prefixes, repeated deterministic calls
- `go test ./internal/context` passes; zero diagnostics on test file

### Security
**Outcome:** pass_with_notes

Evidence:
- No new trust boundaries, authentication flows, external I/O, or secret handling introduced
- Change is internal skill-selection logic and documentation guidance
- No security findings

---

## Collated Findings

### Blocking Findings
None.

### Non-Blocking Findings
None.

---

## Aggregate Verdict

**approved** — All dimensions pass with zero findings. The feature is ready to advance to done.

---

## Build Verification

```
go test ./internal/context  → ok (cached)
diagnostics pipeline.go     → no errors or warnings
diagnostics pipeline_test.go → no errors or warnings
diagnostics SKILL.md        → no errors or warnings
```
