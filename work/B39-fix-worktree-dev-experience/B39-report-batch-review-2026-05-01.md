# Batch Conformance Review: B39-fix-worktree-dev-experience

## Scope
- Batch: B39-fix-worktree-dev-experience
- Features: 2 total (2 done, 0 cancelled/superseded, 0 incomplete)
- Review date: 2026-05-01
- Reviewer: reviewer-conformance

## Feature Census
| Feature | Status | Spec Approved | Dev-Plan | Tasks | Merged to Main | Notes |
|---------|--------|---------------|----------|-------|----------------|-------|
| B39-F1 — Document write_file as Primary Worktree Pattern | done | yes (batch-level) | yes | 2/2 done | yes | Skill file updated on main; T2 deliverable not found (see CG-1) |
| B39-F2 — Make edit_file Worktree-Aware | done | yes (batch-level) | yes | 3/3 done | **no** | Implementation on feature branch only; not merged (see CG-2) |

Both features share the batch-level specification (`B39-fix-worktree-dev-experience/spec-p40-spec-b39-worktree-dev-experience`, approved). Per-feature specs were bypassed via batch-level override on `specifying → dev-planning`. The design document (`P40-retro-batch-april-2026/design-p40-design-retro-batch-improvements`) is approved.

## Spec Conformance Matrix

| Criterion | Requirement | B39-F1 | B39-F2 | Evidence |
|-----------|-----------|--------|--------|----------|
| AC-001 | write_file documented as primary pattern in implement-task/SKILL.md | ✅ PASS | — | `.kbz/skills/implement-task/SKILL.md` L85-107: `write_file(entity_id: ...)` is primary recommendation in "Worktree File Editing" section |
| AC-002 | No python3 -c or heredoc references as primary | ✅ PASS | — | `grep` for `python3\|heredoc\|<<` in implement-task/SKILL.md on main returns no matches |
| AC-003 | write_file in developing-stage tool subset | ❌ GAP | — | `write_file` not found in `.kbz/roles/implementer.yaml`, `.kbz/roles/implementer-go.yaml`, `.kbz/roles/base.yaml`, or `.kbz/stage-bindings.yaml` |
| AC-004 | edit_file accepts entity_id, resolves to worktree | — | ✅ PASS | Verified on feature branch: `edit_file` handler resolves `entity_id` via `worktreeStore.GetByEntityID()`, joins path to worktree root |
| AC-005 | edit_file without entity_id unchanged | — | ✅ PASS | Handler defaults to `repoRoot` when `entity_id` is empty string |
| AC-006 | Clear error for non-existent entity_id | — | ✅ PASS | Handler returns `inlineErr("worktree_not_found", "no worktree found for entity "+entityID)` |
| AC-007 | Multi-edit with entity_id works | — | ✅ PASS | Handler processes edits sequentially after worktree path resolution; same code path for single and multi-edit |
| AC-008 | Same resolution mechanism as write_file | — | ✅ PASS | Both use `worktreeStore.GetByEntityID(entityID)` — confirmed by code inspection of `edit_file` handler and `write_file` handler |
| AC-009 | Existing edit_file tests pass unchanged | — | ✅ PASS | Feature branch commit message: "go test passed on feature branch"; additive change with optional parameter |
| AC-010 | O(1) worktree resolution | — | ✅ PASS | Single `worktreeStore.GetByEntityID()` call — no filesystem scan or loop |
| AC-011 | Only file-writing guidance changed in SKILL.md | ✅ PASS | — | `grep` shows "Worktree File Editing" section (L83-114) contains only write_file/edit_file guidance; no procedure, anti-pattern, or vocabulary changes in other sections |

## Conformance Gaps

| # | Feature | Type | Description | Severity |
|---|---------|------|-------------|----------|
| CG-1 | B39-F1 | spec-gap | **REQ-003 / AC-003 unmet:** `write_file` was not added to the developing-stage tool subset. It does not appear in the `implementer` or `implementer-go` role tool lists, nor in `stage-bindings.yaml`. T2's deliverable is absent from main. Sub-agents spawned for the developing stage may not discover `write_file` without explicit instruction. | **blocking** |
| CG-2 | B39-F2 | merge-status | **Branch not merged:** `feature/FEAT-01KQG1XWE9ABP-edit-file-worktree-aware` exists with 3 commits (verified: `ca639dea`, `89acc1ad`, `8209b0ae`) but is not merged to main. The feature entity is marked `done` via override, but the code has not landed on main. The `edit_file` tool on main does not have the `entity_id` parameter. | **blocking** |

## Documentation Currency

### implement-task/SKILL.md
- **Status:** Current for F1 scope. The "Worktree File Editing" section (L83-114) documents `write_file(entity_id: ...)` as the primary pattern. No `python3 -c` or heredoc references remain.
- **Gap:** The note at L85 still says "The `edit_file` tool only operates on the main working tree. Do not use it for files inside a Git worktree." This will need updating once F2 is merged (the tool will support worktrees via `entity_id`). This is a **documentation lag** — not a conformance gap for the current review, but a known follow-up.

### AGENTS.md Scope Guard
- **Status:** Stale. `health()` reports that B39-fix-worktree-dev-experience (and many other done batches) are not mentioned in the AGENTS.md Scope Guard. This is a pre-existing project-wide issue, not specific to B39.

### Knowledge entries
- No knowledge entries contributed during B39's execution. Not applicable.

### Other project documentation
- The `implementer` and `implementer-go` roles do not list `write_file` in their tool sets. This is tracked as CG-1.

## Cross-Cutting Checks

### Health Report (B39-specific findings)

| Issue | Severity | Detail |
|-------|----------|--------|
| Worktree branch merged but still active (F1) | Warning | `FEAT-01KQG1XWAZE8V` worktree is active but branch `feature/FEAT-01KQG1XWAZE8V-doc-write-file-worktree-pattern` is already merged to main. Should be cleaned up. |
| Branch drift (F1) | Warning | F1 branch is 57 commits behind main (threshold: 50). No impact since already merged. |
| Branch drift (F2) | Warning | F2 branch is 55 commits behind main (threshold: 50). Needs rebase before merge. |

### Worktree Status

| Feature | Worktree | Branch | Status | Action |
|---------|----------|--------|--------|--------|
| B39-F1 | WT-01KQG4FKYE8G7 | feature/FEAT-01KQG1XWAZE8V-doc-write-file-worktree-pattern | active, merged to main | Cleanup needed |
| B39-F2 | WT-01KQG4PDY0PDN | feature/FEAT-01KQG1XWE9ABP-edit-file-worktree-aware | active, not merged | Merge to main, then cleanup |

### Orphaned worktrees
None specific to B39.

## Retrospective Summary

`retro(action: "synthesise", scope: "B39-fix-worktree-dev-experience")` returned **0 signals**. This is expected — both features completed in a single session with minimal friction. No task-completion retrospective signals were contributed during the batch's execution. The batch itself was small (2 features, 5 tasks) and didn't generate workflow observations.

## Batch Verdict

**fail** — 2 blocking conformance gaps:

1. **CG-1:** `write_file` was not added to the developing-stage tool subset (REQ-003 / AC-003). The `implementer` and `implementer-go` role tool lists do not include `write_file`. Sub-agents in the developing stage will not discover the tool automatically.

2. **CG-2:** B39-F2's code (3 commits on `feature/FEAT-01KQG1XWE9ABP-edit-file-worktree-aware`) is not merged to main. The `edit_file` tool on main does not have the `entity_id` parameter. The feature is marked `done` via gate override, but the deliverable has not landed.

**Recommended actions:**
- For CG-1: Add `write_file` to the `implementer` role's tool list in `.kbz/roles/implementer.yaml` (where it will be inherited by `implementer-go`).
- For CG-2: Rebase `feature/FEAT-01KQG1XWE9ABP-edit-file-worktree-aware` onto main, run `go test ./...`, open a PR, merge to main.
- After both are resolved: clean up the F1 worktree (branch already merged), update the `implement-task/SKILL.md` note about `edit_file` not working in worktrees (it will after F2 merges).

## Evidence

- Batch entity: `entity(action: "get", id: "B39-fix-worktree-dev-experience")` → status: done, 2 features
- Feature list: `entity(action: "list", type: "feature", parent: "B39-fix-worktree-dev-experience")` → 2 features
- Feature details: `entity(action: "get", id: "FEAT-01KQG1XWAZE8V")` → done
- Feature details: `entity(action: "get", id: "FEAT-01KQG1XWE9ABP")` → done
- Task list F1: 2 tasks, both done
- Task list F2: 3 tasks, all done
- Spec: `doc(action: "list", owner: "B39-fix-worktree-dev-experience")` → 1 spec, approved
- Dev-plans: both approved
- Branch merge state: `git branch --merged main` → only F1 branch merged
- F2 branch commits: `git log feature/FEAT-01KQG1XWE9ABP-edit-file-worktree-aware` → 3 commits
- F2 implementation: Verified `entity_id` parameter in `edit_file` handler, worktree resolution via `worktreeStore.GetByEntityID()`
- Skill file: `.kbz/skills/implement-task/SKILL.md` L83-114 — write_file documented as primary, no python3/heredoc references
- Role tools: `grep` for `write_file` in `.kbz/roles/*.yaml` and `.kbz/stage-bindings.yaml` → not found
- Health: `health()` → B39-related warnings for worktree cleanup and branch drift
- Retro: `retro(action: "synthesise", scope: "B39-fix-worktree-dev-experience")` → 0 signals
