# Design: Workflow Hygiene Documentation

**Feature:** FEAT-01KPPG4SXY6T0  
**Plan:** P24 — Retro Recommendations  
**Status:** Draft  
**Author:** Architect  

---

## Overview

Five retrospective recommendations (REC-02, REC-05, REC-06, REC-07, REC-08) share a common root cause: agents follow instructions that are incomplete, missing, or buried in low-visibility locations. The result is recurring friction — stashed workflow state, missed pre-task commits, broken `edit_file` calls in worktrees, raw state-file queries, and untracked flaky tests.

This design specifies targeted edits to five existing documentation files. No new files are created, no code is changed, no schema is modified. Every change is a minimal, precise addition or replacement in an existing section.

---

## Goals and Non-Goals

### Goals

- Eliminate the implicit "stash incomplete work" instruction that conflicts with Kanbanzai store discipline (REC-02)
- Surface the "commit workflow state" prompt at session start, outside the task-claiming flow (REC-05)
- Add an explicit warning that `edit_file` does not work in worktrees, with the `python3 -c` workaround (REC-06)
- Add an anti-pattern for shell-querying `.kbz/state/` files, and require `retro synthesise` + `knowledge list` before writing reports (REC-07)
- Require a BUG entity to be filed before marking a task done when any test fails intermittently (REC-08)

### Non-Goals

- **No code changes.** This design does not touch any Go source files, test files, or build artefacts.
- **No new tools or MCP handlers.** All five changes are documentation edits only.
- **No schema changes.** No YAML models, config structs, or storage formats are modified.
- **No new documentation files.** Every change is an edit to an existing file.
- **No refactoring of existing instructions beyond the targeted change.** Only the specific sections named below are modified.

---

## Design

Each subsection covers one recommendation: which files change, exactly which section the change lands in, and the precise before/after text.

---

### REC-02 — Remove implicit "stash" instruction; replace with hard "never stash" rule

**Root cause:** `kanbanzai-getting-started/SKILL.md` tells agents to "stash incomplete work" in its session start checklist, directly contradicting the store discipline rule in `AGENTS.md`. Agents seeing the checklist first follow it without reading the deeper rule.

**Files changed:**

1. `.agents/skills/kanbanzai-getting-started/SKILL.md` — Session Start Checklist
2. `AGENTS.md` — Before Every Task checklist (strengthen the existing note)

#### Change 1a — `kanbanzai-getting-started/SKILL.md`, Session Start Checklist

**Current text (checklist item):**
```
- [ ] **Clean slate** — Run `git status`. Commit coherent changes, stash incomplete work, or proceed if clean.
```

**Replacement:**
```
- [ ] **Clean slate** — Run `git status`. If changes are coherent and complete, commit them now. If changes are incomplete or belong to a different task, inform the human and stop — do not stash. Never use `git stash` in a Kanbanzai project: stashed changes hide workflow state from other agents and are silently lost across worktree switches.
```

#### Change 1b — `kanbanzai-getting-started/SKILL.md`, "Clean slate" prose section

**Current text:**
```
### Clean slate

Run `git status`. If there are uncommitted changes from previous work, commit
or stash them before starting anything new. Never start new work on top of
uncommitted changes from a different task.
```

**Replacement:**
```
### Clean slate

Run `git status`. If there are uncommitted changes from previous work:

- Changes are coherent and complete → **commit them now**, then proceed.
- Changes belong to a different task or are incomplete → **inform the human and stop**. Do not stash, do not discard.

Never use `git stash` in a Kanbanzai project. Stashed changes hide workflow
state from parallel agents, are silently lost when switching worktrees, and
bypass the commit history that makes code review meaningful.
```

#### Change 1c — `AGENTS.md`, Before Every Task checklist

The existing checklist note already says "do not stash" but it is parenthetical. Strengthen it by removing the ambiguity in the second bullet.

**Current text (second bullet):**
```
  - Changes are incomplete or belong to a different task → inform the human, **do not stash** or discard
```

**Replacement** (no change to wording, but add a parenthetical that explains why):
```
  - Changes are incomplete or belong to a different task → inform the human, **do not stash** or discard (stashing hides state from parallel agents and is silently lost across worktree switches)
```

---

### REC-05 — Add "commit workflow state" prompt visible at session start

**Root cause:** The instruction to commit orphaned `.kbz/state/` files appears only inside the pre-task checklist, buried after the "commit coherent changes" bullet. Agents in a hurry skip to the next step and miss it. The `kanbanzai-getting-started` checklist already has a "Store check" item, but its prose is thin and it does not explain what to do step by step.

**Files changed:**

1. `.agents/skills/kanbanzai-getting-started/SKILL.md` — Store check prose section (expand)
2. `AGENTS.md` — Before Every Task checklist (elevate the store check note to a top-level bullet)

#### Change 2a — `kanbanzai-getting-started/SKILL.md`, Store check prose section

Add a new prose section immediately after "Clean slate":

**Insert after the "Clean slate" prose section:**
```
### Commit workflow state

Even when the working tree looks clean for code, run:

```
git status
```

and look specifically for untracked or modified files under `.kbz/state/`,
`.kbz/index/`, or `.kbz/context/`. These are versioned project state — entity
records, document metadata, knowledge entries — that other agents depend on.

If any appear:

1. Stage and commit them immediately before starting any new work:
   ```
   git add .kbz/
   git commit -m "workflow(<context>): commit orphaned state files"
   ```
2. Do not stash, discard, or `.gitignore` them.

MCP tools (`entity`, `doc`, `finish`, `decompose`, `merge`) auto-commit state
changes during normal operation. Orphaned files appear when a previous session
was interrupted before the auto-commit could run. They are rare but consequential:
stale state causes parallel agents to read incorrect entity status and produce
conflicting transitions.
```

#### Change 2b — `AGENTS.md`, Before Every Task checklist

The existing checklist note about orphaned `.kbz/state/` files is embedded inside the first bullet. Elevate it to a separate visible item.

**Current first bullet (with embedded note):**
```
- [ ] Run `git status`. Act on what you find:
  - Changes from previous work are coherent and complete → **commit them now**, then proceed
  - Changes are incomplete or belong to a different task → inform the human, **do not stash** or discard
  - Working tree is clean → proceed
    > Note: MCP tools (entity, doc, finish, decompose, merge) now auto-commit state changes. Orphaned `.kbz/state/` files should be rare but are still worth checking.
```

**Replacement (split into two checklist items):**
```
- [ ] Run `git status`. Act on what you find:
  - Changes from previous work are coherent and complete → **commit them now**, then proceed
  - Changes are incomplete or belong to a different task → inform the human, **do not stash** or discard (stashing hides state from parallel agents and is silently lost across worktree switches)
  - Working tree is clean → proceed
- [ ] **Commit orphaned workflow state** — if `git status` shows any modified or untracked files under `.kbz/state/`, `.kbz/index/`, or `.kbz/context/`, commit them now before starting work. MCP tools auto-commit during normal operation; orphaned files indicate an interrupted previous session. Do not stash or discard them.
```

---

### REC-06 — Warn that `edit_file` does not work in worktrees; document `python3 -c` pattern

**Root cause:** When a task runs inside a Git worktree, the editor tool (`edit_file`) operates on the main working tree, not the worktree's checked-out files. Agents discover this only after a failed edit, costing a full implementation cycle.

**Files changed:**

1. `.kbz/skills/implement-task/SKILL.md` — Checklist and Gotchas / new "Worktree file editing" section
2. `AGENTS.md` — Delegating to Sub-Agents section (one-line callout)

#### Change 3a — `.kbz/skills/implement-task/SKILL.md`, new "Worktree File Editing" section

Add a new section between "## Anti-Patterns" and "## Checklist":

```
## Worktree File Editing

> **Warning:** The `edit_file` tool does not work correctly inside Git worktrees.
> It edits files in the main working tree, not the worktree's checked-out branch.
> Using it inside a worktree produces silent incorrect edits or no-ops.

When implementing tasks assigned to a worktree, write file content using the
`python3 -c` shell pattern via the `terminal` tool:

```
terminal(
  cd: "<worktree-path>",
  command: "python3 -c \"
import pathlib
pathlib.Path('path/to/file.go').write_text('''<full file content>''')
\""
)
```

For smaller targeted edits, use a heredoc:

```
terminal(
  cd: "<worktree-path>",
  command: "cat > path/to/file.go << 'EOF'\n<content>\nEOF"
)
```

Confirm the worktree path before writing. It is available in the context
packet returned by `next(id)` under `worktree.path`.
```

#### Change 3b — `.kbz/skills/implement-task/SKILL.md`, Checklist

Add one item to the checklist immediately after the "Claimed the task" item:

**Current checklist opening:**
```
- [ ] Claimed the task with `next(id: "TASK-xxx")`
- [ ] Read the context packet — spec sections, knowledge entries, file paths
```

**Replacement:**
```
- [ ] Claimed the task with `next(id: "TASK-xxx")`
- [ ] Confirmed whether this task runs inside a worktree — if yes, use `terminal` + `python3 -c` for file writes, NOT `edit_file`
- [ ] Read the context packet — spec sections, knowledge entries, file paths
```

#### Change 3c — `AGENTS.md`, Delegating to Sub-Agents section

Add a callout at the end of the "Delegating to Sub-Agents" section:

**Add after the existing paragraph:**
```
> **Worktree sub-agents:** Sub-agents that run inside a Git worktree cannot use
> `edit_file` — it operates on the main working tree, not the worktree. Always
> instruct worktree sub-agents to write files via `terminal` using the
> `python3 -c` pattern. See the `implement-task` skill for the exact syntax.
```

---

### REC-07 — Anti-pattern for shell-querying `.kbz/state/`; require `retro synthesise` + `knowledge list` before reports

**Root cause (part A):** Agents grep or `cat` files directly from `.kbz/state/` instead of using MCP tools, producing subtly wrong data (raw YAML vs. resolved entity state). The instruction "do not read `.kbz/state/` files directly" exists in `AGENTS.md` under "Critical rules" but is not present as an explicit anti-pattern in `kanbanzai-getting-started`.

**Root cause (part B):** Agents writing retro or research reports work from in-session memory rather than calling `retro(action: "synthesise")` and `knowledge(action: "list")` first, producing incomplete reports that miss signals accumulated across sessions.

**Files changed:**

1. `.agents/skills/kanbanzai-getting-started/SKILL.md` — Anti-Patterns section (add new pattern)
2. `.kbz/roles/implementer-go.yaml` or the relevant `documenter`/`researcher` role — anti-patterns list (add directive); **if the `researcher` role or `documenter` role has an anti-patterns or checklist section, add there; otherwise add to the `write-research` skill**

> **Scoping note for implementation:** During design exploration, the `researcher` role file (`.kbz/roles/researcher.yaml`) and `write-research` skill (`.kbz/skills/write-research/SKILL.md`) were not read. The implementer must check which of those files has an anti-patterns or checklist section and add the REC-07b change there. If both exist, add to the skill file. If neither has an anti-patterns section, add a new one.

#### Change 4a — `kanbanzai-getting-started/SKILL.md`, Anti-Patterns section

Add a new anti-pattern entry at the end of the Anti-Patterns section:

```
### Shell-Querying Workflow State Files

- **Detect:** Agent runs `cat`, `grep`, `find`, or similar shell commands against `.kbz/state/`, `.kbz/index/`, or `.kbz/context/` directories to retrieve entity data.
- **BECAUSE:** Raw YAML files contain unresolved state. MCP tools apply lifecycle resolution, inheritance, computed fields, and cross-reference validation. Shell queries bypass all of this and produce subtly wrong results — wrong status, missing computed fields, stale index data — that lead to incorrect implementation decisions.
- **Resolve:** Use MCP tools exclusively for all workflow state queries:
  - Entity status → `entity(action: "get", id: "...")`
  - Project overview → `status()`
  - Knowledge entries → `knowledge(action: "list")`
  - Documents → `doc(action: "get", path: "...")`
  Never read `.kbz/state/` files with shell tools or `read_file`.
```

#### Change 4b — `write-research/SKILL.md` (or `researcher` role), pre-writing checklist

Add to the checklist or procedure section, before any writing step:

```
- [ ] Called `retro(action: "synthesise")` to surface retrospective signals from all sessions — do not rely on in-session memory alone
- [ ] Called `knowledge(action: "list")` to retrieve project-level knowledge entries relevant to the report topic
```

And add a corresponding anti-pattern:

```
### Report From Memory

- **Detect:** Agent writes a retrospective or research report without first calling `retro(action: "synthesise")` and `knowledge(action: "list")`.
- **BECAUSE:** Retrospective signals and knowledge entries accumulate across sessions. In-session memory only captures the current session. Reports written from memory systematically miss recurring patterns and prior decisions, producing incomplete analysis that cannot support reliable recommendations.
- **Resolve:** Always call `retro(action: "synthesise")` and `knowledge(action: "list")` before writing any report. Treat the synthesised output as the primary input, not a supplement.
```

---

### REC-08 — Require BUG entity before marking task done when tests fail intermittently

**Root cause:** Agents mark tasks done and move on when they observe a flaky test, treating intermittent failures as "probably fine". The failure is not recorded anywhere. The same flake surfaces again in future tasks, requiring re-investigation with no prior record.

**Files changed:**

1. `.kbz/skills/implement-task/SKILL.md` — Checklist and Phase 4: Verify

#### Change 5a — `.kbz/skills/implement-task/SKILL.md`, Checklist

Add one item at the end of the checklist, before the `finish` step:

**Current final two checklist items:**
```
- [ ] Ran the full test suite — all tests pass including regression
- [ ] Verified each acceptance criterion is met
- [ ] Completed the task with `finish` including summary and verification
```

**Replacement:**
```
- [ ] Ran the full test suite — all tests pass including regression
- [ ] Verified each acceptance criterion is met
- [ ] If any test failed intermittently (passed on retry without code change), filed a BUG entity before proceeding
- [ ] Completed the task with `finish` including summary and verification
```

#### Change 5b — `.kbz/skills/implement-task/SKILL.md`, Phase 4: Verify

Add a step to the Phase 4 procedure after "Run the full test suite":

**Current Phase 4, step 1:**
```
1. Run the full test suite. All tests must pass, including pre-existing tests (regression check).
```

**Replacement (expand step 1, add step 1a):**
```
1. Run the full test suite. All tests must pass, including pre-existing tests (regression check).
   - If any test fails intermittently — passes on retry without any code change — do not mark the task done without first filing a BUG entity:
     ```
     entity(action: "create", type: "bug", name: "<test name> fails intermittently",
            observed: "<what was seen>", expected: "test passes consistently",
            severity: "medium", priority: "medium")
     ```
     Record the BUG ID in the task completion summary. Intermittent failures are not "probably fine" — they indicate non-determinism that will compound in future tasks.
```

#### Change 5c — `.kbz/skills/implement-task/SKILL.md`, Anti-Patterns (new entry)

Add a new anti-pattern:

```
### Unreported Flaky Test

- **Detect:** Agent observes a test that fails then passes on retry (without any code change) and marks the task done without filing a BUG entity.
- **BECAUSE:** Intermittent test failures indicate non-determinism — a race condition, timing dependency, or environmental assumption. Not recording them means future agents encounter the same failure with no prior context, re-investigate from scratch, and potentially make the same "probably fine" call. The cumulative cost far exceeds the cost of filing one BUG entity.
- **Resolve:** File a BUG entity for every observed intermittent failure before calling `finish`. Include the test name, the failure message, and the conditions under which it was observed.
```

---

## Alternatives Considered

### Alternative 1: Single consolidated "agent discipline" document

Create a new file (e.g. `DISCIPLINE.md`) that consolidates all five changes into one authoritative reference, then link to it from `AGENTS.md` and the skills.

**Rejected because:** Agents already have too many documents to read at session start. A new file adds to the reading burden rather than reducing it. The five changes are most effective when they appear at the point of action — in the skill a agent is actively reading — not in a document they have to navigate to.

### Alternative 2: Enforce via MCP tool guardrails

Prevent `finish()` from succeeding if a test failure is recorded, or detect `git stash` calls via a commit hook.

**Rejected because:** This is a code change, which is outside the scope of this feature. Tooling enforcement is a valid future improvement (file a separate feature) but should not block the immediate documentation fix.

### Alternative 3: Single AGENTS.md addition covering all five

Add all five changes as a new "Known Pitfalls" section in `AGENTS.md`.

**Rejected because:** `AGENTS.md` is already long. The most effective placement for each change is at the point where the agent is most likely to encounter the situation — in the skill file they are actively executing. A central pitfalls list is too far from the context where the pitfall occurs.

### Alternative 4: Add `edit_file` worktree detection to the tool itself

Detect when `edit_file` is called from within a worktree context and return an error.

**Rejected because:** Code change, out of scope. Document the workaround now; open a separate feature for tool-level enforcement.

---

## Dependencies

- **No code dependencies.** All changes are to Markdown and YAML files tracked in the repository.
- **FEAT-01KPPG5XMJWT3** (optional GitHub PR creation) — independent; no ordering constraint.
- **REC-07b file location** — the implementer must read `.kbz/skills/write-research/SKILL.md` and `.kbz/roles/researcher.yaml` to confirm which file receives the REC-07b change. The change itself is specified above; only the target file requires confirmation during implementation.
- **Stage binding prerequisite** — this feature is in the `designing` stage. The design must be approved before tasks are created or implementation begins.