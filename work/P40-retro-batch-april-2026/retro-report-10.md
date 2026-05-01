# Personal Retrospective — Worktree/Branch Audit & Fix (April 2026)

**Author:** DeepSeek V4 Flash (AI agent)
**Task:** Audit all unmerged branches/worktrees, merge unreleased code, advance stuck lifecycle states, clean up stale records, and fix the root cause
**Date:** 2026-04-29

---

## What Went Well

### The system's health check told me exactly what was wrong
The `health` tool surfaced a comprehensive picture: worktree path does not exist, branch not found, critical drift, branch has merge conflicts, worktree cleanup overdue, feature has all children done but stuck in developing. Every category had a clear message. I didn't need to manually grep through files to figure out the state — the tool did it. This was the highest-value tool of the session.

### Merge tool's post-merge lifecyle auto-advance is now on main
The root cause of B34 (features marked done but never merged) was that the merge tool didn't advance the lifecycle. Writing the fix was straightforward because the existing `MaybeAutoAdvanceFeature` method already existed — the issue was just that the merge tool never called it. A two-step chain (developing → reviewing → done) in `executeMerge` solved it. This was a small, targeted fix with high impact.

### Sub-agents handled parallel merge operations cleanly
Three B34 feature branches had no merge conflicts with main. Dispatching all three merges to separate sub-agents in parallel let them rebase, push, and merge independently. They returned clear summaries of what happened (merge commit SHA, override reasons, cleanup schedule). The sub-agent pattern was well-suited to this — each branch was a fully independent write scope.

### Entity lifecycle gate override mechanism worked as intended
The `reviewing→done` gate blocked auto-advance for B38 features that lacked verification records on their tasks. Using `override: true` with an explicit `override_reason` ("State repair: all tasks done, branch already merged to main") was the right escape hatch. The forced explanation in the override_reason parameter is good design — it prevents casual skipping.

### The status dashboard scoped attention items usefully
The project-level status dashboard grouped attention items by severity and entity type. I could immediately see that B38 features with all tasks terminal were flagged as `feature_child_consistency` warnings, and that B34 branches had `branch_drift` errors. This triaged my work order correctly: urgent (unmerged code with drift/conflicts) → medium (stuck lifecycle) → cleanup (stale worktrees).

---

## What Didn't Go Well

### The merge tool's EntityDoneGate blocked merging developing features
B34 features were stuck in a catch-22: the EntityDoneGate required status `done` to merge, but they were in `developing` because they'd never been merged. The B34-F8 feature (Documentation & Skills Updates) was auto-advancing from `developing` to `reviewing` via `MaybeAutoAdvanceFeature` when the merge completed, but the gate check had already run before the auto-advance happened. The fix (accepting `reviewing` in EntityDoneGate) was trivial, but it should have been unnecessary — the merge tool should sequence as: merge → auto-advance lifecycle → check done.

### The initial auto-advance attempt failed silently
My first attempt at post-merge auto-advance tried to transition `developing → done` directly via `UpdateStatus`, which failed because the lifecycle state machine rejects that transition. I got a warning in the merge response but it was easy to miss. The fix (chaining `MaybeAutoAdvanceFeature` first, then `UpdateStatus`) required a second commit. A unit test for this path would have caught it before deployment.

### Sub-agent for the conflict-heavy branch timed out
I dispatched `FEAT-01KQ2E0RB4P8A` (the one with actual merge conflicts) to a sub-agent, but the session was canceled due to rebase complexity. I had to do it myself. The sub-agents that handled the no-conflict branches worked fine. This is expected — conflict resolution requires human oversight — but the system could better triage: flag "has conflicts" early and suggest direct intervention.

### Squash merges made re-audit misleading
The re-audit showed every squash-merged branch as "NOT MERGED" by `git merge-base --is-ancestor` because squash commits are not ancestors of the branch tip. I had to do a second pass using `git log --grep` for merge commit messages to confirm the actual state. The `branch` tool's status check also showed these branches as "critical drift" (321 commits behind main) even after the merge. The worktree records correctly show `merged_at`, but the branch-level health check doesn't account for squash merges — it sees a branch 321 commits behind and flags it.

### Remote tracking branches of already-merged features still existed
Six stale remote branches survived on origin. These were branches whose features were merged weeks ago but the remote branches were never cleaned up. The cleanup tool only handles local worktrees. A `git push origin --delete` was needed manually. This suggests the merge tool's `auto_delete_remote_branch` config option (`cfg.Cleanup.AutoDeleteRemoteBranch`) exists but either defaults to false or didn't run.

### The `worktree(action: remove)` failed for orphaned records
For the 17 worktree records whose directories had been manually deleted, `worktree(action: remove)` returned `"is not a working tree"` — a git-level error, not a tool-level one. I had to manually delete the `.kbz/state/worktrees/*.yaml` files. The worktree tool could detect this case (record exists, directory doesn't) and offer a force-remove that cleans up the state file without going through git.

---

## What to Improve

### Fix the EntityDoneGate merge ordering
The merge tool should: merge → auto-advance lifecycle → check post-merge gates. Currently it checks pre-merge gates (which include EntityDoneGate) before the merge happens. For features in `developing` or `reviewing`, the gate should pass if the feature is architecturally mergeable (tasks done, no conflicts), and the done-status check should happen after the auto-advance.

### Make squash-merged branches not appear as critically drifted
When a branch has a `merged_at` timestamp in the worktree record, the branch health check should skip the drift alert. The branch is intentionally stale — it's a completed feature. Alternatively, delete the local tracking branch on squash merge so the drift check doesn't trigger.

### Add a `worktree(action: gc)` for orphaned records
When `.kbz/state/worktrees/WT-XXX.yaml` exists but the git worktree directory and `.git/worktrees/WT-XXX/` have both been removed, the record is orphaned. A garbage-collection action could detect and remove these in bulk.

### The merge tool should clean remote branches by default
I found `cfg.Cleanup.AutoDeleteRemoteBranch` in the codebase. If this was already on and the branch deletions still needed manual action, there's a bug. If it defaults to false, it should default to true — remote branches of merged features are always clutter.

### Pre-merge conflict detection should drive dispatch decisions
Before dispatching a sub-agent to merge a branch, the conflict tool should report whether conflicts are expected. No conflicts → safe for sub-agent. Has conflicts → suggest direct orchestrator intervention. This would prevent the aborted sub-agent session I experienced.

---

## Verdict

The `health` tool did the heavy lifting — it converted a vague sense that "something is wrong with my branches" into a structured, actionable report with entity IDs, severity levels, and specific messages. That alone made this session productive.

The two code fixes (merge auto-advance + EntityDoneGate relaxation) are small (23 lines total) but close the loop on a recurring problem: the merge step is now the natural lifecycle completion point, and bypassing it requires explicit override with explanation.

The cleanup was tedious but mechanical (28 state files, 6 remote branches, 18 overdue worktrees). This will gradually improve as the merge tool's cleanup features mature — auto-deleting remote branches and better handling orphaned records would eliminate the bulk of manual work.
