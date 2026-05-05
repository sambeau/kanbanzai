# Dev-Plan: Merge Discipline and Definition of Done

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-05                     |
| Status | approved |
| Author | architect                      |

## Overview

This dev-plan implements the merge discipline and definition-of-done spec:
`work/P50-retro-may-2026/P50-spec-merge-discipline.md`
(DOC-`FEAT-01KQTWFY52EG1/spec-p50-spec-merge-discipline`).

Adds `merging` and `verifying` lifecycle stages, a post-review merge prompt, and a fix for
worktree `merged` status verification. No new MCP tool actions — uses existing `pr`, `merge`,
and `entity` tools.

## Task Breakdown

### T1: Add merging and verifying to feature lifecycle model
- **Deliverable:** `internal/model/entities.go` updated with `FeatureStatusMerging` and `FeatureStatusVerifying` constants; lifecycle state machine updated to include `reviewing → merging → verifying → done`
- **Depends on:** nothing
- **Effort:** 1 (small model change)
- **Parallelisable:** yes

### T2: Add stage bindings for merging and verifying
- **Deliverable:** `.kbz/stage-bindings.yaml` updated with `merging` and `verifying` stage entries; `internal/kbzinit/stage-bindings.yaml` mirrored
- **Depends on:** T1 (needs the status constants)
- **Effort:** 1 (YAML config)
- **Parallelisable:** no

### T3: Implement merging stage transition logic
- **Deliverable:** Transition handler in `internal/service/entities.go` or `internal/mcp/entity_tool.go` that enforces `merging` prerequisites (all tasks terminal, approved report) and advances to `merging`
- **Depends on:** T1, T2
- **Effort:** 2 (gate logic + tests)
- **Parallelisable:** no

### T4: Implement verifying stage with build/test check
- **Deliverable:** Transition handler that runs `go build ./...` and `go test ./...` from repo root, transitions to `done` on success or `needs-rework` on failure with output attached as reason
- **Depends on:** T1, T2
- **Effort:** 3 (shell execution, output capture, error parsing)
- **Parallelisable:** no

### T5: Fix worktree merged status verification
- **Deliverable:** `internal/worktree/` or `merge` tool handler updated to run `git merge-base --is-ancestor <branch> main` after merge, only marking worktree as `merged` when verified
- **Depends on:** T1 (needs lifecycle context)
- **Effort:** 2 (git command + worktree record update)
- **Parallelisable:** yes (can run alongside T3/T4)

### T6: Add merge prompt to orchestrate-review skill
- **Deliverable:** `.kbz/skills/orchestrate-review/SKILL.md` updated with post-review merge prompt; `internal/kbzinit/skills/orchestrate-review/SKILL.md` mirrored
- **Depends on:** T2 (references the new stages)
- **Effort:** 1 (documentation)
- **Parallelisable:** yes

### T7: Add merge prompt to kanbanzai-agents skill
- **Deliverable:** `.agents/skills/kanbanzai-agents/SKILL.md` updated with merge discipline guidance; `internal/kbzinit/skills/agents/SKILL.md` mirrored
- **Depends on:** T2
- **Effort:** 1 (documentation)
- **Parallelisable:** yes

### T8: Integration test for full merge-verify-done flow
- **Deliverable:** Integration test that creates a feature, transitions through merging → verifying (mocked build/test), and asserts done/needs-rework outcomes
- **Depends on:** T3, T4, T5
- **Effort:** 2 (integration test)
- **Parallelisable:** no

## Dependency Graph

```
T1 ──┬── T2 ──┬── T3 ──┬── T8
     │        │        │
     │        ├── T4 ──┤
     │        │        │
     ├── T5 ──┘        │
     │                 │
     ├── T6 ───────────┤
     │                 │
     └── T7 ───────────┘
```

T1, T5, T6, T7 are independent and can run in parallel (4 parallel streams).
T2 depends on T1. T3 and T4 depend on T2. T8 depends on T3, T4, T5.

## Interface Contracts

- **Feature status constants:** `FeatureStatusMerging` and `FeatureStatusVerifying` added to `model` package. All existing status-switch statements must handle the new values.
- **Stage bindings:** New `merging` and `verifying` entries use existing binding model. No schema changes.
- **Build/test execution:** `verifying` stage calls `go build` and `go test` as subprocesses. No new Go APIs — uses `os/exec`.
- **Worktree verification:** `git merge-base --is-ancestor` called via existing `internal/git` package. Adds one new function.

## Traceability Matrix

| Requirement | Task(s) |
|-------------|---------|
| REQ-001 (merging stage) | T1, T3 |
| REQ-002 (verifying stage) | T1, T2 |
| REQ-003 (build check) | T4 |
| REQ-004 (test check) | T4 |
| REQ-005 (done transition) | T4 |
| REQ-006 (worktree fix) | T5 |
| REQ-007 (orchestrate-review prompt) | T6 |
| REQ-008 (kanbanzai-agents prompt) | T7 |
| REQ-009 (dual-write) | T6, T7 |
| REQ-010 (auto gate) | T2 |
| REQ-NF-001 (backward compat) | T3 |
| REQ-NF-002 (5-min timeout) | T4 |
