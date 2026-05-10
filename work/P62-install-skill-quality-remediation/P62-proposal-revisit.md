# P62 Revisit: Complete Unfinished Work

## Context

P62 (`Install & Skill Quality Remediation`) was marked `done` prematurely. Its execution
batch **B64** (`install-skill-quality`) is `done` but contains incomplete features.

A post-completion audit found:

| Feature | Status | Tasks |
|---------|--------|-------|
| **F1** `init-unblock` (`FEAT-01KR7BKXG3X61`) | `dev-planning` | 2 queued |
| **F2** `install-registry` (`FEAT-01KR7BKXJGEPD`) | `done` (override) | **9 queued** |
| **F3** `runtime-surfaces` (`FEAT-01KR7BKXMK3B6`) | `done` | 0 (8 orphaned in worktree cleanup) |
| **F4** `install-tests-doctor` (`FEAT-01KR7BKXPXFSK`) | `done` | 0 |

F2 was overridden from `reviewing → done` with reason *"Merged. Review approved. T4
verification was implicit"* — but 9 tasks were never started. The feature is `done` with
incomplete work.

F3 had 8 task state files destroyed by `git clean` during worktree cleanup. The override
says *"All 8 commits exist on the branch, code review approved, tests pass"* — verify this
is actually true and mark tasks as `not-planned` or `done` accordingly.

## Phase 1: Investigate What Went Wrong

Before fixing anything, investigate and summarise:

1. How did F2 end up `done` with 9 queued tasks?
2. How did F3 lose its task state?
3. Why did the batch advance to `done` with F1 still at `dev-planning`?
4. What orchestration gap allowed this to pass the DoD check?

Write a brief (~1 paragraph) root cause summary. Do not spend more than one round on this.

## Phase 2: Complete the Unfinished Work

### F1 `init-unblock` (2 queued tasks)

- TASK-01KR7D38JH4C9: Add missing frontmatter markers to embedded SKILL.md files
- TASK-01KR7D38JPJ23: Implement `transformSkillContent` helper

Advance F1 through `dev-planning → developing → reviewing → done`. Follow the standard
workflow: create a worktree, dispatch tasks via `handoff`/`spawn_agent`.

### F2 `install-registry` (9 queued tasks)

All 9 tasks are `queued`. Some implementation work may already exist on the branch.
Dispatch all 9 tasks. Do not redo work that is already done — verify each task's
acceptance criteria against what exists on the branch, then `finish` with appropriate
verification.

### F3 `runtime-surfaces` (8 orphaned tasks)

No task entities exist (state files destroyed). Verify the branch contains the expected
commits. If the work is complete: create retrospective task entities and mark them `done`
or `not-planned`. If incomplete: create proper tasks and implement them.

## Phase 3: Close Out

Complete the batch to DoD standard. F4 is already `done` — verify it still passes.

---

## Hard Rules (Non-Negotiable)

- **Sub-agents do the work. You do the coordination.** Do not write code, edit files,
  or read source files yourself. If you find yourself loading a source file to understand
  an implementation detail, stop — fix the dev-plan instead and delegate.
- **`handoff` before `spawn_agent`. Always.** No exceptions. No hand-written prompts.
- **Worktrees are mandatory.** Every feature must have its own Git worktree. Never
  implement features on `main` or in a shared branch.
- **`status()` is the source of truth.** Not the table above, not the conversation
  history. Call `status()` at session start and after each wave of completions.
- **No implicit gates.** After a task or wave completes, immediately proceed to the
  next action. The only valid stop is the final completion report.
- **45% context threshold.** If your context utilisation reaches ~45%, offload using
  the compaction artefact procedure from the fast-track profile before continuing.

---

## Definition of Done

P62 is complete when **all four features** (F1–F4) satisfy:

- [ ] All tasks terminal (`done` or `not-planned`)
- [ ] No blocking review findings outstanding
- [ ] Feature transitioned to `done`
- [ ] Branch merged to `main` and deleted
- [ ] Binary builds without error
- [ ] All tests pass
- [ ] Worktree removed
- [ ] Knowledge entries curated

Do not report completion until every feature has cleared every gate above.
