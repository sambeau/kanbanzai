# Dev Plan: Sub-agent Orchestration Documentation Improvements

**Feature:** FEAT-01KPQ08YKHNS9
**Plan:** P25 — Agent Tooling and Pipeline Quality
**Spec:** work/spec/p25-orchestration-docs.md

---

## Overview

Three targeted documentation changes addressing recurring orchestration failures:
1. Add "always use `handoff`" mandate + anti-pattern to `orchestrate-development/SKILL.md`
2. Add dual-write rule for embedded skill files to `AGENTS.md`
3. Fix the `entity` tool `parent` field description in `internal/mcp/entity_tool.go`

All three changes are independent and can be implemented in parallel, though the Go
source change (Task 3) requires a rebuild to take effect.

---

## Task Breakdown

### Task 1: Update orchestrate-development/SKILL.md

- **Description:** Add a bold "always use handoff" rule at the top of Phase 3 (Dispatch
  Sub-Agents) before the existing numbered steps, and add a "Manual Prompt Composition"
  anti-pattern entry to the `## Anti-Patterns` section.
- **Deliverable:** Updated `.kbz/skills/orchestrate-development/SKILL.md`
- **Depends on:** None (independent)
- **Effort:** Small
- **Spec requirements:** FR-001, FR-002

### Task 2: Update AGENTS.md with dual-write rule

- **Description:** Add a subsection to the `## Git Discipline` section of `AGENTS.md`
  documenting the dual-write requirement: when any `.agents/skills/kanbanzai-*/SKILL.md`
  file is modified, the corresponding `internal/kbzinit/skills/*/SKILL.md` must be
  updated in the same commit. Include the explanation of why (kanbanzai init embeds these
  files for distribution to new projects) and explicitly exclude `.kbz/skills/` from the
  requirement.
- **Deliverable:** Updated `AGENTS.md`
- **Depends on:** None (independent)
- **Effort:** Small
- **Spec requirements:** FR-003, FR-004

### Task 3: Fix entity_tool.go parent field description

- **Description:** Update the `mcp.Description(...)` string for the `parent` parameter
  in `internal/mcp/entity_tool.go`. The new description must:
  - Remove the `(list only)` qualifier
  - State that `parent` is the parent plan ID for features and is required on feature create
  - State that it is also used as a filter on list calls
  - Note that tasks use `parent_feature`, not `parent`
  This is a string-literal change only — no logic, routing, or schema changes.
- **Deliverable:** Updated `internal/mcp/entity_tool.go`
- **Depends on:** None (independent)
- **Effort:** Small
- **Spec requirements:** FR-005, FR-006, FR-007

### Task 4: Write tests

- **Description:** Verify that existing tests for `entity_tool.go` still pass after the
  description string change. No new tests are required for documentation-only changes
  (Tasks 1 and 2). For Task 3, confirm the existing test suite passes without modification.
- **Deliverable:** Passing test run (`go test ./internal/mcp/...`)
- **Depends on:** Task 3
- **Effort:** Small
- **Spec requirements:** AC-005

---

## Dependency Graph

Tasks 1, 2, and 3 are fully independent and can run in parallel. Task 4 must follow
Task 3.

```
Task 1 ─┐
Task 2 ─┤ (parallel)
Task 3 ─┴─► Task 4
```

---

## Interface Contracts

No shared interfaces between tasks. Each task touches a disjoint set of files:

| Task | Files Modified |
|------|---------------|
| Task 1 | `.kbz/skills/orchestrate-development/SKILL.md` |
| Task 2 | `AGENTS.md` |
| Task 3 | `internal/mcp/entity_tool.go` |
| Task 4 | No file changes — test run only |

---

## Notes

- Task 1 modifies the same file as FEAT-01KPQ08YH16WZ (impl-workflow-docs). If both
  features are being implemented concurrently, coordinate the changes to
  `orchestrate-development/SKILL.md` to avoid merge conflicts.
- Task 3 requires a Go rebuild (`go install ./cmd/kanbanzai/`) and MCP server restart
  before the updated description is visible to clients. The commit message should note this.
- No changes to `internal/kbzinit/skills/` are required by this feature itself — the
  dual-write rule added in Task 2 is about documenting the obligation, not triggering it
  for this commit.

---

## Traceability Matrix

| Spec Requirement | Task |
|-----------------|------|
| FR-001 (handoff mandate rule in Phase 3) | Task 1 |
| FR-002 (manual composition anti-pattern) | Task 1 |
| FR-003 (dual-write rule in AGENTS.md) | Task 2 |
| FR-004 (embedding relationship explanation) | Task 2 |
| FR-005 (entity_tool.go parent description) | Task 3 |
| FR-006 (no logic changes in entity_tool.go) | Task 3 |
| FR-007 (rebuild note in commit) | Task 3 |
| AC-001 (handoff mandate present in SKILL.md) | Task 1 |
| AC-002 (manual-composition anti-pattern present) | Task 1 |
| AC-003 (dual-write rule present and correct) | Task 2 |
| AC-004 (parent description accurate) | Task 3 |
| AC-005 (no regressions in entity tool tests) | Task 4 |