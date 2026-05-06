# Dev-Plan: P50 Final Closure

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-06                     |
| Status | approved |
| Author | architect                      |

## Overview

The May 2026 retrospective batch (P50) has 5 features that were marked `done` or `reviewing`
but the code-level audit reveals gaps between the task system and reality. This dev-plan
defines the remaining work to get P50 to a clean, honest `done` state.

The original batch conformance review found 10 blocking findings. The second review for F4
claims all resolved, but the code disagrees. F2 and F3 were never formally re-reviewed after
their blocking findings. F5 was never reviewed at all.

### Feature Status Summary

| Feature | System Status | Real Status | Gap |
|---------|:-------------:|:-----------:|-----|
| F1 Error Classification | `reviewing` | Done | Transition to `done` |
| F2 Decompose Proposal Quality | `done` | Uncertain | Review rejected, BF-1 unverified |
| F3 Commit Discipline Prompts | `done` | Mostly done | BF-4 (getting-started mirror) unverified |
| F4 Merge Discipline | `done` | Not done | 3 BF open, 4 tasks ready |
| F5 Document Path Tool | `done` | Not done | 2 tasks ready, never reviewed |

## Task Breakdown

### C1: Fix F4 — needs-rework → developing transition

- **Deliverable:** Add `FeatureStatusDeveloping: true` to `FeatureValidTransitions[FeatureStatusNeedsRework]` in `internal/model/entities.go`. Update tests to assert the new transition.
- **Findings addressed:** BF-9
- **Depends on:** nothing
- **Effort:** 1 (one line + test assertion)
- **Parallelisable:** yes

### C2: Wire F4 transition enforcement into entity handler

- **Deliverable:** Call `IsValidFeatureTransition` in the entity transition path (`internal/mcp/entity_tool.go` or `internal/service/entities.go`) so invalid status transitions are rejected before state is mutated. Add tests for rejection of invalid transitions.
- **Findings addressed:** BF-5
- **Depends on:** C1 (adds the missing transition first)
- **Effort:** 2 (validation call + test cases)
- **Parallelisable:** no

### C3: Add F4 merge prompt to kanbanzai-agents skill + mirror

- **Deliverable:** Add the 5-step Post-Review Merge section to `.agents/skills/kanbanzai-agents/SKILL.md` and mirror to `internal/kbzinit/skills/agents/SKILL.md`. Content matches what already exists in `orchestrate-review/SKILL.md`.
- **Findings addressed:** BF-8
- **Depends on:** nothing
- **Effort:** 1 (documentation)
- **Parallelisable:** yes

### C4: Complete F4 remaining ready tasks

- **Deliverable:** Four tasks currently stuck in `ready`:
  - **TASK-01KQTX80GGSG8**: Implement merging stage transition logic
  - **TASK-01KQTX80GGSYS**: Implement verifying stage with build/test check
  - **TASK-01KQTX80KQX77**: Add stage bindings for merging and verifying
  - **TASK-01KQTX80GGJS2**: Add merge prompt to kanbanzai-agents skill (covered by C3)
  
  Verify that the code already implements these (the `executeMerge` path already does merging→verifying transitions, and stage bindings may already exist in `.kbz/stage-bindings.yaml`). For any not already done, implement them. Then transition all four to `done`.
- **Findings addressed:** F4 incomplete tasks
- **Depends on:** C1, C2, C3 (these may already be partially or fully done in code)
- **Effort:** 2 (audit + fill gaps + transition)
- **Parallelisable:** no

### C5: Verify F2 paired=false opt-out exists

- **Deliverable:** Check `internal/service/decompose.go` and `internal/mcp/decompose_tool.go` for the `paired_test_tasks` / `paired=false` input field. Trace through the proposal generation to confirm it produces one-task-per-AC when disabled. If missing, implement it. If present, document the evidence.
- **Findings addressed:** BF-1
- **Depends on:** nothing
- **Effort:** 1 (audit + possibly small implementation)
- **Parallelisable:** yes

### C6: Fix F3 getting-started dual-write mirror

- **Deliverable:** Compare `.agents/skills/kanbanzai-getting-started/SKILL.md` with `internal/kbzinit/skills/getting-started/SKILL.md`. Ensure the strengthened session-start git status check (commit orphaned `.kbz/` files, forbid stashing/discarding) is present in both. Mirror any missing content.
- **Findings addressed:** BF-4
- **Depends on:** nothing
- **Effort:** 1 (diff + mirror)
- **Parallelisable:** yes

### C7: Complete or defer F5 Document Path Tool

- **Deliverable:** Two tasks in `ready`:
  - **TASK-01KQTX65HWP1R**: Add prompt path support
  - **TASK-01KQTX65HWZ93**: Add register-time path warning
  
  Decision: either complete these (implement + test) or explicitly mark them `not-planned` with rationale and defer F5 to a future plan. If completed, transition F5 to `reviewing` and run a review. If deferred, update F5 status and document the decision.
- **Depends on:** nothing
- **Effort:** 3 (full completion) or 1 (deferral)
- **Parallelisable:** yes

### C8: Transition F1 to done

- **Deliverable:** Call `entity(action: "transition", id: "FEAT-01KQTNYMZRT6V", status: "done")`. F1 has all tasks done, review approved — it just needs the transition.
- **Depends on:** nothing
- **Effort:** 0 (one tool call)
- **Parallelisable:** yes

### C9: Re-review F2, F3, F4

- **Deliverable:** For each feature with previously rejected reviews, run the reviewer pipeline (conformance, quality, security, testing). Register re-review reports. If all blocking findings are resolved, mark reviews approved. If new findings emerge, fix them.
- **Depends on:** C1–C6 (all remediation must be complete first)
- **Effort:** 3 (three feature re-reviews)
- **Parallelisable:** no

### C10: Close P50

- **Deliverable:** Verify all 5 features are in terminal state (done or explicitly deferred with documentation). Verify no orphaned worktrees or branches. Transition P50 to `done`.
- **Depends on:** C7, C8, C9
- **Effort:** 1 (verification + transition)
- **Parallelisable:** no

## Dependency Graph

```
C1 ── C2 ──┐
            │
C3 ────────┤
            ├── C4 ──┐
C5 ────────┤        │
            │        ├── C9 ── C10
C6 ────────┤        │
            │        │
C7 ────────┘        │
                     │
C8 ─────────────────┘
```

**Parallel groups:**
- **Wave 1 (all independent):** C1, C3, C5, C6, C7, C8
- **Wave 2:** C2 (after C1), C4 (after C1, C2, C3)
- **Wave 3:** C9 (after C1–C6)
- **Wave 4:** C10 (after C7, C8, C9)

## Interface Contracts

- **F4 transition enforcement:** `IsValidFeatureTransition(from, to)` must be called before any feature status mutation. Invalid transitions must return a `precondition_error` with a clear message listing valid targets.
- **needs-rework → developing:** Added as a valid transition alongside existing `needs-rework → reviewing`.
- **Skill file changes:** All changes to `.agents/skills/` and `.kbz/skills/` must be mirrored to `internal/kbzinit/skills/` in the same commit.
- **F5 deferral (if chosen):** Feature status set to `not-planned` or `cancelled` with rationale recorded in the entity and a document registered explaining the deferral.

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| F4 transition enforcement breaks existing flows | Low | High | Add enforcement with clear error messages. Existing valid transitions pass through unchanged. |
| F2 opt-out never implemented | Medium | Medium | Audit first, implement only if missing. Small scope — one field + conditional branch. |
| F5 completion is de-scoped but later needed | Medium | Low | Document deferral clearly. The path action already works for design/spec/dev-plan types — only prompt path and register warning remain. |
| git status shows dirty working tree | High | Medium | Commit `.kbz/` changes between each task. Follow the commit discipline F3 itself implemented. |

## Verification Approach

| Task | Verification |
|------|-------------|
| C1 | `go test ./internal/model/ -run TestFeatureValidTransitions` — asserts `needs-rework → developing` |
| C2 | `go test ./internal/mcp/ -run TestEntityTransition` — asserts invalid transitions rejected |
| C3 | Diff inspection: merge prompt present in both agent skill files |
| C4 | `status(id: "FEAT-01KQTWFY52EG1")` shows 8/8 tasks done |
| C5 | Code audit or `go test ./internal/service/ -run TestDecompose` — paired=false produces 1 task/AC |
| C6 | Diff inspection: getting-started mirrors match |
| C7 | `status(id: "FEAT-01KQTNYN00HZA")` shows terminal state with documentation |
| C8 | `status(id: "FEAT-01KQTNYMZRT6V")` shows `done` |
| C9 | Each feature has an approved review report with no blocking findings |
| C10 | `status(id: "P50-retro-may-2026")` shows `done` |

## Traceability Matrix

| Review Finding / Requirement | Closure Task(s) |
|------------------------------|-----------------|
| BF-1: F2 missing paired=false opt-out | C5 |
| BF-4: F3 getting-started mirror missing | C6 |
| BF-5: F4 transition state machine unenforced | C2 |
| BF-8: F4 agents skill missing merge prompt | C3 |
| BF-9: F4 missing needs-rework → developing | C1 |
| F4 incomplete tasks (4 ready) | C4 |
| F5 incomplete tasks / never reviewed | C7 |
| F1 needs transition to done | C8 |
| F2, F3, F4 review not re-approved | C9 |
| P50 final closure | C10 |
