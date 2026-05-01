# Retrospective Report 15 — P37 kbz move implementation

**Agent:** Claude Sonnet 4.6  
**Session scope:** P37-F3 (FEAT-01KQ7JDT11MH6) — `kbz move` command, Tasks T1–T5  
**Context:** Resuming a previous session that had run out of context mid-implementation

---

## What went well

### Status and orientation tools were genuinely useful
`status(id: "P37-file-names-and-actions")` gave me an accurate picture of where the plan stood in
under one tool call — tasks done, tasks active, tasks queued, feature statuses. Resuming
mid-session was much faster than it would have been without it. The active-task marker
(`status: "active"`) immediately flagged that T1 had been claimed but not completed.

### Handoff-generated prompts worked as designed
When dispatching sub-agents for T2+T3 and T4, the `handoff` context packets contained the
right things: spec sections, knowledge entries about worktree editing patterns, the role identity,
and the codebase graph project name. Both sub-agents completed their tasks correctly on the
first attempt without any back-and-forth. This is the system working as intended.

### `write_file` with `entity_id` is significantly better than heredocs
Sub-agents used `write_file(entity_id: "FEAT-...")` to write Go source files into the worktree.
This avoids the triple-escaping problem that makes `python3 -c` painful for code with embedded
quotes or braces. The knowledge entry about this pattern was surfaced in the sub-agent context
and acted on.

### Sub-agent delegation was clean and parallel
Dispatching T2+T3 to one sub-agent and T4 to another, then handling T5 myself, kept context
lean. Each sub-agent came back with exactly what was needed: commit hash, test output summary,
clear description of what was done. No context bloat from retaining full diffs.

### Worktree tool gave reliable file path
`worktree(action: "get", entity_id: "FEAT-01KQ7JDT11MH6")` immediately returned the worktree path
and branch name. I never had to guess where to look.

---

## What didn't go well

### No recovery path after context interruption
The biggest friction point of the session. The previous agent had run out of context partway
through T1, leaving uncommitted state: `internal/git/git.go` and `internal/service/documents.go`
modified but not staged. The system had no way to signal "this is where I stopped" — no
continuation note, no partial completion marker on the task, nothing.

I had to reconstruct what had been done by reading `git diff`, cross-referencing the dev plan,
and inferring intent from the code. This took more tool calls than the actual implementation.
If the interrupted session had left even a single-sentence comment in the task record ("added
GitMove, started on UpdateDocumentPathAndOwner, not committed") the recovery would have been
trivial.

**Suggestion:** When `finish` is not called before a session ends, the active task should be
automatically reset to `ready` with a timestamped note. Or: the task claim (`next`) should
auto-commit any pending `.kbz/state/` changes so at minimum the task record is clean.

### 80 orphaned .kbz/ files committed blind
There were ~80 modified knowledge and index files with no clear connection to the current work.
They had to be committed in a single catch-all commit (`chore: commit orphaned workflow state`)
because the pre-task checklist requires a clean working tree. This is a real risk: committing
files blind could include stale or incorrect state.

The system provides no tooling to inspect "which session modified this file" or "is this safe
to commit." The `health` tool doesn't surface orphaned uncommitted state as a warning.

**Suggestion:** `health` should report uncommitted `.kbz/` files as a warning. The pre-task
checklist note about committing orphaned state should include a way to identify which files
are safe vs. suspicious (e.g., only commit `.kbz/state/knowledge/` and `.kbz/index/` — avoid
committing `.kbz/state/tasks/` or `.kbz/state/features/` blindly without reading them).

### T3 auto-promotion didn't fire
After completing T2 (which T3 depends on), T3 should have auto-promoted from `queued` to `ready`.
It didn't. I had to manually call `entity(action: "transition", status: "ready")` before I could
claim or finish it. This is a workflow friction that compounds in multi-task features — if five
tasks need manual unblocking, that's five extra tool calls.

The auto-promotion hook does fire in other scenarios (side effects confirmed it for T4 and T5
when T2 was finished), so this appears to be a race or edge case where the promotion signal
was missed for T3 specifically.

**Suggestion:** Audit the auto-promotion hook for tasks that have a single dependency that was
just completed in the same `finish` call. The current T3 case may expose a gap where
batch-finish doesn't trigger per-task promotion for the tasks it doesn't process.

### The `finish` batch error was unhelpful
When I tried to batch-finish T2 and T3 together, T3 was still in `queued` status and the
batch failed for T3 with `"task is in status queued (expected ready or active)"`. This is
correct behaviour, but the error gave no hint about how to fix it — no suggestion to
transition to `ready` first, no link to the entity ID.

**Suggestion:** Batch `finish` failure messages should include a corrective action:
`"Transition to ready first: entity(action: \"transition\", id: \"TASK-...\", status: \"ready\")"`.

### T4 introduced a `--force` regression
The sub-agent implementing Mode 2 didn't notice that `--force` needed to be parsed in the
shared `runMove` dispatch function, not inside `runMoveFeature`. The task description focused
entirely on the Mode 2 logic and didn't explicitly say "also fix flag parsing in the parent
function." I caught this before writing tests, but it slipped past the sub-agent.

This is a spec-tracing gap: the spec's AC-015 says `--force` works, and the task description
mentions it, but the cross-cutting concern (flag parsing in the shared parent) wasn't called
out. The decomposition put all five implementation steps inside the `runMoveFeature` task,
but one of them — flag parsing — lives in `runMove`.

**Suggestion:** When a task description involves a flag that affects shared parent-function
dispatch, the handoff prompt or task summary should explicitly name which functions need
updating. A pattern like "this flag must be parsed in `runMove` before the mode dispatch"
is easy to miss from a spec written at the requirement level.

### Unmerged dependency not surfaced
`ParseShortPlanRef` and `ResolvePlanByNumber` were implemented in FEAT-01KQ2E0RB4P8A, a feature
that was done but living in its own worktree, not yet merged to main. The `kbz move` worktree
couldn't see these functions. I had to re-implement equivalent plan resolution logic inline in
`move_cmd.go`.

The `next`/`handoff` context assembly had no way to flag this: "you're implementing a feature
that would benefit from P34-F1, but that feature isn't merged yet." I only discovered the
gap by searching for the function names and finding them in a different worktree.

**Suggestion:** When a task's spec references functions or patterns from a feature that is
`done` but not yet merged (still in an active worktree), the handoff context should surface
a warning: "FEAT-X is done but unmerged — these symbols are not yet in main."

---

## Minor observations

- The `next` tool claims the task when called with an `id`. I occasionally wanted to inspect
  context without claiming. A `peek: true` parameter or a separate `context(task_id)` action
  would be useful for read-only inspection.

- The P37 feature display IDs (`FEAT-01KQ7-JDT11MH6`) are significantly more human-readable
  than the raw TSIDs. The display ID system clearly works and made the session easier to
  navigate. Marking this as a genuine UX win.

- The `entity(action: "transition", status: "ready")` workaround for stuck-queued tasks worked
  cleanly once found. The fact that it's a manual workaround rather than automatic is the problem,
  not the tool itself.

---

## Summary

The core workflow — `status` → `next` → `handoff` → `spawn_agent` → `finish` — worked well
for multi-task feature orchestration. The main friction was around session continuity (interrupted
sessions leave no recovery trail), auto-promotion reliability, and cross-feature dependency
visibility. The tooling for what's happening *right now* is good; the tooling for what happened
*before this session* and what's *pending in other branches* is weak.
