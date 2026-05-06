# Batch Conformance Review: B1-p51-exec — Handoff Pipeline Unification

## Scope
- **Plan**: P51-handoff-pipeline-unification (Handoff Pipeline Unification)
- **Batch**: B1-p51-exec
- **Features**: 4 total (4 implemented, 0 cancelled/superseded)
- **Review date**: 2026-05-06
- **Reviewer**: reviewer-conformance

## Feature Census

| Feature | ID | Status | Spec Approved | Dev-Plan Approved | Design Approved | All Tasks Done | Notes |
|---------|-----|--------|---------------|-------------------|-----------------|----------------|-------|
| Context Budget Recalibration | FEAT-01KQYZZFGBGQK | `developing` | ✅ | ✅ | ✅ | ✅ 3/3 | Uncommitted changes in worktree |
| Fix Sub-Agent Role Routing | FEAT-01KQYZZFGHM99 | `developing` | ✅ | ✅ | ✅ | ✅ 3/3 | Uncommitted changes in worktree |
| Documentation Fixes | FEAT-01KQYZZFGH6DK | `developing` | ✅ | ✅ | ✅ | ✅ 2/2 | Uncommitted changes in worktree |
| Remove Legacy Assembly Path | FEAT-01KQYZZFGJA93 | `developing` | ✅ | ✅ | ✅ | ✅ 5/5 | Uncommitted changes in worktree |

> Note: FEAT-01KQYZZFGJA93 was in `dev-planning` at the time of the impromptu review but has since been advanced to `developing`. Its status is now consistent with having all 5 tasks done.

## Conformance Gaps

| # | Feature | Type | Description | Severity |
|---|---------|------|-------------|----------|
| **CG-1** | All | lifecycle | All 4 features have uncommitted worktree changes. The branches were merged to `main` (via `git branch --merged main`) but the merge base is only the lifecycle transition commit `eda00276` — no actual code commits exist on the branches. The implementation code exists solely as dirty state in worktree working directories. | **blocking** |
| **CG-2** | All | commit | No source code commits exist on any feature branch or on main. The task completion commits on main contain only `.kbz/state/` metadata updates. The `finish()` calls recorded metadata but produced no code commits. | blocking |
| **CG-3** | All | worktree-isolation | All 4 features share overlapping modified files. The Context Budget Recalibration and Remove Legacy Assembly Path worktrees both modify `server.go`, `handoff_tool.go`, `assembly.go` — changes are interleaved rather than isolated. This means the changes cannot be cleanly separated into independent branches without manual conflict resolution. | blocking |
| **CG-4** | B1-p51-exec | entity-registration | The batch entity `B1-p51-exec` exists as a YAML state file (`.kbz/state/batches/B1-p51-exec.yaml`) but is **not registered as an entity** — `entity(action: "get", id: "B1-p51-exec")` and `status(id: "B1-p51-exec")` both fail with "entity not found". The batch is invisible to the entity system. | non-blocking (procedural) |
| **CG-5** | All | retro | No retrospective signals have been recorded for this batch. | non-blocking |

## Assessment of Implementation Quality (from worktree review)

Despite the procedural gaps, the **implementation itself is sound**:

### 1. Context Budget Recalibration ✅
- `DefaultContextWindowTokens`: `200_000` → `1_000_000` in `internal/context/pipeline.go`
- `assemblyDefaultBudget`: `30720` → `65536` (64KB) in `internal/mcp/assembly.go`
- `ContextWindowTokens` config field added to `internal/config/user.go` with validation (rejects < 100,000)
- `WindowTokens()` exported method on Pipeline struct
- `truncateTopic()` function added to trimming logic with `scope`, `tier`, `tokenEstimate` fields on `asmTrimmedEntry`
- `go build ./...` passes in the worktree

### 2. Fix Sub-Agent Role Routing ✅
- `stepResolveRole` defaults to `SubAgents.Roles[0]` before falling back to `Binding.Roles[0]`
- `stepLoadSkill` defaults to `SubAgents.Skills[0]` before falling back to `Binding.Skills[0]`
- `go build ./...` passes in the worktree

### 3. Remove Legacy Assembly Path ✅
- `HandoffTools` simplified from 9 parameters to 2 (`entitySvc`, `pipeline`)
- `tryPipeline`, `buildLegacyResponse`, `renderHandoffPrompt` removed from `handoff_tool.go`
- `handoff` calls `pipeline.Run` directly — no fallback
- `assembleContext` / `asmInput` / `assembledContext` retained in `assembly.go` (still used by `next_tool.go` — correctly kept)
- `go build ./...` passes in the worktree

### 4. Documentation Fixes ✅
- `finish` tool description updated to `"(max 500 characters)"` and 500-char validation implemented
- `.kbz/skills/orchestrate-development/SKILL.md` updated to use `handoff(task_id: "TASK-xxx", role: "implementer-go")` throughout
- Same updates in `internal/kbzinit/skills/...` (consumer install copies)
- `go build ./...` passes in the worktree

## Documentation Currency
- `.kbz/skills/orchestrate-development/SKILL.md`: **updated** (in worktree)
- `internal/kbzinit/skills/task-execution/orchestrate-development/SKILL.md`: **updated** (in worktree)
- `internal/kbzinit/skills/task-execution/implement-task/SKILL.md`: **updated** (in worktree)

## Batch Attributes
- **Parent plan**: P51-handoff-pipeline-unification (status: `active`)
- **Batch status** (from YAML): `proposed` — has never been advanced. Entity is unregistered so cannot be transitioned.
- **Features**: 4 fully implemented, all docs approved, all tasks done

## Retrospective Summary
No retrospective signals were recorded for this batch. The `retro(action: "synthesise", scope: "B1-p51-exec")` returned zero signals. This is expected when tasks are completed without recording retrospective observations via `finish(retrospective: [...])`.

## Batch Verdict
**fail** — blocking procedural/conformance issues prevent this batch from being considered delivery-complete:

1. **CG-1/CG-2 (blocking)**: No code has been committed to any feature branch. All implementation work exists only as dirty state in worktree working directories. The branches must have their code committed and be properly merged to `main` before the batch can be closed.
2. **CG-3 (blocking)**: Worktree isolation is violated — 4 features' changes are interleaved across shared worktree directories. The changes must be disentangled into per-feature commits.
3. **CG-4 (non-blocking)**: The batch entity `B1-p51-exec` exists as a YAML state file but is not registered as an entity, making it invisible to the entity system and unable to be advanced through its lifecycle.

## Recommended Next Steps

1. **Commit each worktree's changes to its respective branch** — the changes are all verified and build-passing. Each worktree's diff represents a valid, complete feature.
2. **Resolve worktree isolation** — the interleaved changes across worktrees mean that committing one worktree will include another feature's changes. Options: (a) craft per-feature commits manually, or (b) use a single branch with all changes combined as one unified merge.
3. **Register the batch entity** — `entity(action: "create", type: "batch", id: "B1-p51-exec", ...)` so it's trackable.
4. **Merge branches to main** via the standard `merge(action: "execute", entity_id: "FEAT-...")` workflow.
5. **Advance feature lifecycles** through `reviewing → done` after merge.
6. **Record retrospective signals** for this batch before closing.

## Evidence
- Batch state file: `.kbz/state/batches/B1-p51-exec.yaml` — exists but unregistered
- Feature list: `entity(action: "list", type: "feature", parent: "B1-p51-exec")` → 4 features confirmed via parent filter
- Task status: All 13 tasks across 4 features confirmed `done` (13/13)
- Spec approval: All 4 specifications approved ✅
- Dev-plan approval: All 4 dev-plans approved ✅
- Design approval: All 4 designs share a single approved design document ✅
- Health check: `health()` — no P51-specific errors; only pre-existing cross-project stale-worktree warnings
- Worktree status: `worktree(action: "list", status: "active")` → 4 P51 worktrees all active
- Branch state: All 4 branches at `eda00276` with 0 code commits beyond merge base
- Worktree changes: Verified via `git -C .worktrees/... status` — all implementations present and build-passing
- Branch merge status: `git branch --merged main` lists all 4 branches (merged at the lifecycle transition commit only, not with actual code)
- Main commits: Task completion commits on main contain only `.kbz/state/` metadata — no source code changes
