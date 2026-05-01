# Retrospective Report 8: BUG-01KQEWZ8YJ1FN — Filename Redundancy Fix

**Date:** 2026-04-30
**Author:** sambeau
**Scope:** Fixing redundant filename IDs and B38-compatibility review pass

---

## What went well

### Design/spec as ground truth
The P37 design document (§3, D3) and F2 specification (REQ-005) were crystal clear
about the canonical filename template. When investigating whether the issue was a
code bug or a spec ambiguity, I could point to exact sections and examples in both
documents and conclude confidently that the code (validation) was correct but the
agent guidance was wrong. No guessing, no inference — just reading the spec.

### The copilot-instructions role/skill system worked
The stage bindings and role system made the review phase straightforward. Reading
`reviewer-conformance.yaml` gave me the exact anti-patterns to check against
(Phantom Requirement was directly relevant — the fix was correct but the examples
were pre-B38 and needed updating). The review process felt structured rather than
ad-hoc.

### Git status before starting caught dirty state
The requirement to check `git status` before every task caught that the worktree
had uncommitted changes from unrelated work. This prevented accidentally mixing
the bugfix commit with other changes.

### Entity tooling for bug lifecycle
Creating BUG-01KQEWZ8YJ1FN and BUG-01KQEY5909FM7 via `entity(action: create, type: bug)`
was smooth. The structured observed/expected/severity/priority fields forced me to
think clearly about scope before writing any code.

---

## What didn't go well

### Worktree confusion with shared files
I created a worktree for the bugfix (`WT-01KQEYEPPFEMP`) but the files I needed to
edit (`.agents/skills/` and `internal/kbzinit/skills/`) are shared across all
worktrees — they live at the repo root, not under `work/`. The edits ended up on
the main working tree anyway because the worktree shares those directories. The
worktree added no isolation value and I had to force-remove it. For documentation
fixes that touch shared agent-facing files, a worktree is the wrong tool — a
simple branch on main would have sufficed.

### edit_file tool precision issues
The `edit_file` tool repeatedly failed with "old_text did not match" even when I
could verify the text existed in the file. After three attempts I fell back to
`python3 -c` via `terminal`, which worked immediately but lost the structured
edit tracking. This happened on a file that had been recently edited, suggesting
a possible stale-buffer issue.

### finish tool doesn't apply to bugs
I tried `finish(task_id: "BUG-01KQEWZ8YJ1FN")` and got an unintuitive "task not
found" error. Bugs use `entity(action: transition)` to move through their
lifecycle, not `finish`. The error message should say "bugs use entity transition,
not finish" rather than generically "task not found."

### No retro signal collection during work
The copilot instructions say to record retrospective signals via `finish` but since
this was a bug (not a task) there was no natural `finish` call point. I had to
write this report from memory after the fact. A standalone `retro` signal
collection mechanism would have helped capture observations as they happened.

### kbzinit dual-write silently out of sync
The dual-write rule (update both `.agents/skills/` and `internal/kbzinit/skills/`)
was enforced by convention only. After my initial edit, the `.agents/` copy was
updated but `internal/kbzinit/` wasn't — I caught it during review, but a CI check
or `kbz health` warning for drift between these pairs would have been better.

---

## What we should improve

1. **Worktree tool should warn when no files are isolated.** If the entity's work
   only touches shared (non-`work/`) files, the worktree provides no value and
   adds cleanup overhead. A warning or suggestion to use a plain branch would help.

2. **`finish` error message for non-task entities.** When called with a bug or
   feature ID, the error should say "finish is for tasks; use entity(action: transition)
   for bugs" rather than the generic "task not found."

3. **`edit_file` stale-buffer recovery.** When `edit_file` can't match old_text,
   it could offer to re-read the file and show a diff of what changed since the
   last read. Three attempts with manual verification between each is excessive
   for a tool that should handle this.

4. **CI check for kbzinit skill drift.** A `kbz health` check that compares
   `.agents/skills/kanbanzai-*/SKILL.md` against `internal/kbzinit/skills/*/SKILL.md`
   and warns on content divergence would catch the dual-write gap automatically.

5. **Standalone retro signal recording.** A way to record retrospective signals
   outside of `finish` (e.g., `knowledge(action: contribute, topic: "retro-...")`
   with appropriate tags) would enable signal collection during bug work or ad-hoc
   investigation where no task is being completed.
