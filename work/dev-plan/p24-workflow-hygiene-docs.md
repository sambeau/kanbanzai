# Dev Plan: Workflow Hygiene Documentation

**Feature:** FEAT-01KPPG4SXY6T0
**Plan:** P24 — Retro Recommendations
**Spec:** `work/spec/p24-workflow-hygiene-docs.md`
**Design:** `work/design/p24-workflow-hygiene-docs.md`

---

## Overview

This plan decomposes the requirements in `work/spec/p24-workflow-hygiene-docs.md`
(FEAT-01KPPG4SXY6T0) into four independent documentation editing tasks. Each
task targets a single file and can execute in parallel with the others. No code
is written and no new files are created.

---

## Scope

This plan implements the requirements defined in
`work/spec/p24-workflow-hygiene-docs.md`. It covers four independent editing
tasks — one per target file — that can execute in parallel. It does not cover
code changes, schema changes, or the creation of new files.

**Target files:**
1. `.agents/skills/kanbanzai-getting-started/SKILL.md`
2. `AGENTS.md`
3. `.kbz/skills/implement-task/SKILL.md`
4. `.kbz/skills/write-research/SKILL.md` (primary) or `.kbz/roles/researcher.yaml` (fallback — see Task 4)

---

## Task Breakdown

### Task 1: Update `kanbanzai-getting-started/SKILL.md`

- **Description:** Apply three targeted edits to
  `.agents/skills/kanbanzai-getting-started/SKILL.md`:
  1. Replace the "stash incomplete work" checklist item in the Session Start
     Checklist with a hard "never stash" rule (FR-001).
  2. Replace the "Clean slate" prose section with a two-case structure that
     prohibits `git stash` and explains the worktree data-loss risk (FR-002).
  3. Insert a new "Commit workflow state" prose section immediately after
     "Clean slate", covering `.kbz/state/`, `.kbz/index/`, `.kbz/context/`
     inspection and the `git add .kbz/` commit sequence (FR-004).
  4. Add a "Shell-Querying Workflow State Files" anti-pattern entry at the end
     of the Anti-Patterns section (FR-009).
- **Deliverable:** Modified `.agents/skills/kanbanzai-getting-started/SKILL.md`
  with all four changes applied.
- **Depends on:** None.
- **Effort:** Small.
- **Spec requirements:** FR-001, FR-002, FR-004, FR-009.

**Input context:**
- `work/spec/p24-workflow-hygiene-docs.md` — AC-001 through AC-002, AC-004, AC-009
- `work/design/p24-workflow-hygiene-docs.md` — Change 1a, 1b, 2a, 4a for exact before/after text

**Acceptance criteria:**
- AC-001: Session Start Checklist contains no stash instruction; explicitly prohibits `git stash`.
- AC-002: "Clean slate" prose describes two cases (commit or stop-and-inform) and contains verbatim prohibition of `git stash`.
- AC-004: "Commit workflow state" section appears immediately after "Clean slate", names the three directories, contains the `git add .kbz/` sequence, explains the consequence.
- AC-009: Anti-Patterns section contains "Shell-Querying Workflow State Files" with all required content.

---

### Task 2: Update `AGENTS.md`

- **Description:** Apply three targeted edits to `AGENTS.md`:
  1. Strengthen the second bullet in the "Before Every Task" checklist by
     adding a parenthetical explaining why stashing is harmful (FR-003).
  2. Add a new, dedicated checklist item in the "Before Every Task" section
     for committing orphaned `.kbz/state/`, `.kbz/index/`, and
     `.kbz/context/` files (FR-005).
  3. Add a callout at the end of the "Delegating to Sub-Agents" section
     warning that worktree sub-agents cannot use `edit_file` and must use
     the `terminal` + `python3 -c` pattern (FR-008).
- **Deliverable:** Modified `AGENTS.md` with all three changes applied.
- **Depends on:** None.
- **Effort:** Small.
- **Spec requirements:** FR-003, FR-005, FR-008.

**Input context:**
- `work/spec/p24-workflow-hygiene-docs.md` — AC-003, AC-005, AC-008
- `work/design/p24-workflow-hygiene-docs.md` — Change 1c, 2b, 3c for exact before/after text

**Acceptance criteria:**
- AC-003: Second bullet in Before Every Task includes parenthetical about parallel agents and worktree-switch loss.
- AC-005: A dedicated checklist item for orphaned workflow state exists as a separate bullet from the `git status` code-change item.
- AC-008: "Delegating to Sub-Agents" section ends with a callout referencing `implement-task` skill and the `python3 -c` pattern.

---

### Task 3: Update `.kbz/skills/implement-task/SKILL.md`

- **Description:** Apply five targeted edits to
  `.kbz/skills/implement-task/SKILL.md`:
  1. Insert a new "Worktree File Editing" section between "Anti-Patterns" and
     "Checklist", containing the `edit_file` warning, `python3 -c` pattern,
     and heredoc alternative (FR-006).
  2. Add a worktree-confirmation checklist item immediately after "Claimed the
     task" (FR-007).
  3. Add a BUG-filing checklist item after the acceptance-criteria verification
     item and before the `finish` item (FR-011).
  4. Expand Phase 4 ("Verify") step 1 to cover intermittent test failures,
     including the `entity(action: "create", type: "bug", ...)` call with all
     required fields (FR-012).
  5. Add an "Unreported Flaky Test" anti-pattern entry in the Anti-Patterns
     section (FR-013).
- **Deliverable:** Modified `.kbz/skills/implement-task/SKILL.md` with all
  five changes applied.
- **Depends on:** None.
- **Effort:** Medium.
- **Spec requirements:** FR-006, FR-007, FR-011, FR-012, FR-013.

**Input context:**
- `work/spec/p24-workflow-hygiene-docs.md` — AC-006, AC-007, AC-011, AC-012, AC-013
- `work/design/p24-workflow-hygiene-docs.md` — Change 3a, 3b, 5a, 5b, 5c for exact before/after text

**Acceptance criteria:**
- AC-006: "Worktree File Editing" section appears between Anti-Patterns and Checklist; contains warning block, `python3 -c` pattern, and heredoc alternative.
- AC-007: Worktree-confirmation item is the second checklist item (immediately after "Claimed the task").
- AC-011: BUG-filing item appears after acceptance-criteria item and before `finish` item.
- AC-012: Phase 4 step 1 expansion is present with all required fields in the `entity(action: "create")` call; BUG ID recording requirement is stated.
- AC-013: "Unreported Flaky Test" anti-pattern is present with detectable behaviour, cost explanation, and resolution.

---

### Task 4: Update `write-research/SKILL.md` (or `researcher.yaml`)

- **Description:** Add two pre-writing checklist items and a "Report From
  Memory" anti-pattern to the appropriate file. The implementer MUST first
  read `.kbz/skills/write-research/SKILL.md` to determine whether it has an
  anti-patterns or checklist section. If yes, add there. If neither section
  exists, fall back to `.kbz/roles/researcher.yaml`. The change MUST NOT be
  added to both files.

  Changes to add:
  1. Two pre-writing checklist items requiring `retro(action: "synthesise")`
     and `knowledge(action: "list")` before writing any report (FR-010a).
  2. A "Report From Memory" anti-pattern entry explaining that in-session
     memory misses prior-session signals and directing the agent to treat
     synthesised output as the primary input (FR-010b).
- **Deliverable:** Modified target file with both additions applied.
- **Depends on:** None.
- **Effort:** Small.
- **Spec requirements:** FR-010.

**Input context:**
- `work/spec/p24-workflow-hygiene-docs.md` — AC-010
- `work/design/p24-workflow-hygiene-docs.md` — Change 4b for exact text
- `.kbz/skills/write-research/SKILL.md` — read first to confirm target
- `.kbz/roles/researcher.yaml` — fallback if skill file lacks required sections

**Acceptance criteria:**
- AC-010: Confirmed target file contains two pre-writing checklist items (`retro(action: "synthesise")` and `knowledge(action: "list")`) and a "Report From Memory" anti-pattern entry.

---

## Dependency Graph

```
Task 1 (no dependencies) ──┐
Task 2 (no dependencies) ──┤──► all done — feature complete
Task 3 (no dependencies) ──┤
Task 4 (no dependencies) ──┘
```

All four tasks are independent. They edit different files with no shared
sections or ordering constraints.

**Parallel groups:** [Task 1, Task 2, Task 3, Task 4]

**Critical path:** Any single task (all are approximately equal weight;
Task 3 is the largest with five changes to one file).

---

## Interface Contracts

This feature contains no code changes. No function signatures, data structures,
or inter-package contracts are defined or modified. All deliverables are
Markdown and YAML documentation files.

---

## Traceability Matrix

| Requirement | Task        |
|-------------|-------------|
| FR-001      | Task 1      |
| FR-002      | Task 1      |
| FR-003      | Task 2      |
| FR-004      | Task 1      |
| FR-005      | Task 2      |
| FR-006      | Task 3      |
| FR-007      | Task 3      |
| FR-008      | Task 2      |
| FR-009      | Task 1      |
| FR-010      | Task 4      |
| FR-011      | Task 3      |
| FR-012      | Task 3      |
| FR-013      | Task 3      |
| FR-NF-001   | Tasks 1–4   |
| FR-NF-002   | Tasks 1–4   |

---

## Risk Assessment

### Risk: REC-07b target file lacks required sections

- **Probability:** Low
- **Impact:** Low
- **Mitigation:** Task 4 explicitly requires the implementer to inspect
  `write-research/SKILL.md` first and fall back to `researcher.yaml`. The
  design and spec both document this decision point. If neither file has
  an anti-patterns or checklist section, the implementer must add one.
- **Affected tasks:** Task 4 only.

### Risk: Section headings have been renamed since the design was written

- **Probability:** Low
- **Impact:** Medium — a renamed heading means the implementer cannot locate
  the insertion point by name and must navigate by intent.
- **Mitigation:** The spec documents the fallback: "locate the equivalent
  section by intent." The design provides exact before/after text as a
  secondary anchor for locating the right place.
- **Affected tasks:** Tasks 1, 2, 3, 4.

### Risk: Markdown structure is damaged by edits

- **Probability:** Low
- **Impact:** Medium — broken fences or mismatched heading levels render
  the skill file unreadable by downstream agents.
- **Mitigation:** FR-NF-001 requires each implementer to preserve heading
  hierarchy and list structure conventions. After each edit, the implementer
  must scan the modified section for unclosed fences and orphaned bullets.
- **Affected tasks:** Tasks 1, 2, 3, 4.

---

## Verification Approach

All acceptance criteria for this feature are verified by inspection (reading
the modified file and confirming the required content is present).

| Acceptance Criterion | Verification Method | Producing Task |
|----------------------|--------------------:|----------------|
| AC-001               | Inspection          | Task 1         |
| AC-002               | Inspection          | Task 1         |
| AC-003               | Inspection          | Task 2         |
| AC-004               | Inspection          | Task 1         |
| AC-005               | Inspection          | Task 2         |
| AC-006               | Inspection          | Task 3         |
| AC-007               | Inspection          | Task 3         |
| AC-008               | Inspection          | Task 2         |
| AC-009               | Inspection          | Task 1         |
| AC-010               | Inspection          | Task 4         |
| AC-011               | Inspection          | Task 3         |
| AC-012               | Inspection          | Task 3         |
| AC-013               | Inspection          | Task 3         |