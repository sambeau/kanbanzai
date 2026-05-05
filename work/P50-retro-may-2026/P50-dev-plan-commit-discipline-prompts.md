# Dev-Plan: Commit Discipline Prompts

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-05                     |
| Status | approved |
| Author | architect                      |

## Overview

This dev-plan implements the commit discipline prompts spec:
`work/P50-retro-may-2026/P50-spec-commit-discipline-prompts.md`
(DOC-`FEAT-01KQTNYN01ZF8/spec-p50-spec-commit-discipline-prompts`).

Two changes: a `state_modified` flag on MCP tool responses for state-mutating operations so
agents know when to commit, and a strengthened session-start `git status` check in the
kanbanzai-getting-started skill that requires committing orphaned `.kbz/` files.

## Task Breakdown

### T1: Add state_modified field to MCP response schema
- **Deliverable:** Response struct updated in `internal/mcp/` to include optional `state_modified` bool field. Field is present when any `.kbz/state/`, `.kbz/index/`, or `.kbz/context/` file was modified during the call.
- **Depends on:** nothing
- **Effort:** 1 (struct field + serialization)
- **Parallelisable:** yes

### T2: Set state_modified in high-mutation tools
- **Deliverable:** `entity` (transition, update, create), `doc` (register, approve, supersede), `knowledge` (contribute, confirm, retire), and `finish` handlers updated to set `state_modified: true` on their responses when state was mutated. Read-only tools leave it absent/false.
- **Depends on:** T1
- **Effort:** 2 (update 4 tool handlers)
- **Parallelisable:** no

### T3: Add state_modified rule to kanbanzai-agents skill
- **Deliverable:** `.agents/skills/kanbanzai-agents/SKILL.md` updated with rule: "When a tool response contains `state_modified: true` and you are not about to make another state-mutating call, commit the `.kbz/` changes before proceeding." `internal/kbzinit/skills/agents/SKILL.md` mirrored.
- **Depends on:** T1 (references the flag)
- **Effort:** 1 (documentation)
- **Parallelisable:** yes

### T4: Strengthen session-start git status check in getting-started skill
- **Deliverable:** `.agents/skills/kanbanzai-getting-started/SKILL.md` pre-task checklist strengthened: if `git status` shows modified/untracked files under `.kbz/state/`, `.kbz/index/`, or `.kbz/context/`, agent must commit them before starting new work. Explicitly forbids stashing or discarding. `internal/kbzinit/skills/getting-started/SKILL.md` mirrored.
- **Depends on:** nothing
- **Effort:** 1 (documentation)
- **Parallelisable:** yes

### T5: Add commit discipline tests
- **Deliverable:** Unit tests: entity transition sets state_modified, status query does not, table-driven test covering mutation actions across entity/doc/knowledge/finish. Vanilla MCP client test: state_modified field doesn't cause rejection.
- **Depends on:** T2
- **Effort:** 2 (test suite)
- **Parallelisable:** no

## Dependency Graph

```
T1 ──┬── T2 ──┬── T5
     │        │
     ├── T3 ──┤
     │        │
     └── T4 ──┘
```

T1 gates T2 and T3. T4 is independent. T5 gates on T2. T3 and T4 can run in parallel.

## Interface Contracts

- **state_modified** is a top-level boolean field in tool responses. Absent or false for read-only operations. True for state-mutating operations.
- No new tool actions — response field only
- Skill file changes must follow dual-write rule in same commit
- Flag is additive — no existing response fields removed or renamed

## Traceability Matrix

| Requirement | Task(s) |
|-------------|---------|
| REQ-001 (state_modified on mutation) | T1, T2 |
| REQ-002 (absent on read) | T2 |
| REQ-003 (high-mutation tools first) | T2 |
| REQ-004 (kanbanzai-agents rule) | T3 |
| REQ-005 (getting-started checklist) | T4 |
| REQ-006 (dual-write) | T3, T4 |
| REQ-NF-001 (no latency impact) | T5 |
| REQ-NF-002 (client compat) | T5 |
