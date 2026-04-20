# Specification: Workflow Hygiene Documentation

**Feature:** FEAT-01KPPG4SXY6T0
**Plan:** P24 — Retro Recommendations
**Design:** `work/design/p24-workflow-hygiene-docs.md`
**Status:** Draft

---

## Problem Statement

This specification covers the targeted documentation edits described in
`work/design/p24-workflow-hygiene-docs.md`. Five retrospective recommendations
(REC-02, REC-05, REC-06, REC-07, REC-08) identified recurring agent behaviour
failures caused by incomplete, missing, or low-visibility instructions. The
design prescribes precise additions and replacements to five existing files:

- `.agents/skills/kanbanzai-getting-started/SKILL.md`
- `AGENTS.md`
- `.kbz/skills/implement-task/SKILL.md`
- `.kbz/skills/write-research/SKILL.md` (or `.kbz/roles/researcher.yaml` — confirmed during implementation)

**In scope:** Edits to the four files listed above, at the exact sections named
in each requirement.

**Explicitly out of scope:**
- Changes to any Go source files, test files, or build artefacts
- Creation of new documentation files
- New MCP tools or handlers
- Schema, YAML model, or config struct changes
- Refactoring of existing instructions beyond the targeted sections

---

## Requirements

### Functional Requirements

#### REC-02: Remove implicit "stash" instruction; replace with hard "never stash" rule

**FR-001:** The Session Start Checklist in
`.agents/skills/kanbanzai-getting-started/SKILL.md` MUST NOT contain any
instruction to stash incomplete work. The checklist item that currently reads
`"stash incomplete work"` MUST be replaced with an instruction that:
(a) directs the agent to commit coherent and complete changes,
(b) directs the agent to inform the human and stop when changes are incomplete
or belong to a different task, and
(c) explicitly prohibits `git stash` with a brief explanation of why.

**FR-002:** The "Clean slate" prose section in
`.agents/skills/kanbanzai-getting-started/SKILL.md` MUST NOT instruct agents
to stash uncommitted changes. It MUST be replaced with a section that:
(a) covers the same two cases as FR-001 (commit or stop),
(b) states that `git stash` MUST never be used in a Kanbanzai project, and
(c) explains that stashed changes hide workflow state from parallel agents and
are silently lost when switching worktrees.

**FR-003:** The second bullet in the "Before Every Task" checklist in `AGENTS.md`
— the bullet that handles changes that are incomplete or belong to a different
task — MUST include a parenthetical explanation that stashing hides state from
parallel agents and is silently lost across worktree switches. The existing
"do not stash" directive in that bullet MUST be retained.

#### REC-05: Add "commit workflow state" prompt at session start

**FR-004:** A new prose section titled "Commit workflow state" MUST be added to
`.agents/skills/kanbanzai-getting-started/SKILL.md` immediately after the
"Clean slate" prose section. This new section MUST:
(a) instruct the agent to run `git status` and look specifically for modified
or untracked files under `.kbz/state/`, `.kbz/index/`, and `.kbz/context/`,
(b) specify the exact `git add .kbz/` and `git commit` command sequence to use
when such files are found,
(c) prohibit stashing, discarding, or adding these files to `.gitignore`,
(d) explain that MCP tools auto-commit state changes during normal operation
and that orphaned files indicate an interrupted previous session, and
(e) explain the consequence of ignoring them: stale state causes parallel
agents to read incorrect entity status and produce conflicting transitions.

**FR-005:** The "Before Every Task" checklist in `AGENTS.md` MUST contain a
dedicated checklist item for committing orphaned workflow state files. This
item MUST be separate from the `git status` item that handles code changes. It
MUST instruct the agent to commit any modified or untracked files found under
`.kbz/state/`, `.kbz/index/`, or `.kbz/context/` before starting work, and
MUST NOT instruct the agent to stash or discard them.

#### REC-06: Warn that `edit_file` does not work in worktrees; document `python3 -c` pattern

**FR-006:** A new section titled "Worktree File Editing" MUST be added to
`.kbz/skills/implement-task/SKILL.md` between the "Anti-Patterns" section and
the "Checklist" section. This section MUST:
(a) contain a visible warning that `edit_file` does not work correctly inside
Git worktrees and that it edits files in the main working tree instead,
(b) provide the exact `terminal` + `python3 -c` invocation pattern for writing
full file contents inside a worktree,
(c) provide an alternative heredoc pattern for smaller targeted edits, and
(d) instruct the agent to confirm the worktree path from the context packet
returned by `next(id)` before writing.

**FR-007:** The checklist in `.kbz/skills/implement-task/SKILL.md` MUST contain
an item immediately after "Claimed the task with `next(id: "TASK-xxx")`" that
requires the agent to confirm whether the task runs inside a worktree and, if
so, to use `terminal` + `python3 -c` for file writes instead of `edit_file`.

**FR-008:** The "Delegating to Sub-Agents" section in `AGENTS.md` MUST contain
a callout at its end stating that sub-agents running inside a Git worktree
cannot use `edit_file`, that they must write files via `terminal` using the
`python3 -c` pattern, and that the `implement-task` skill contains the exact
syntax.

#### REC-07: Anti-pattern for shell-querying `.kbz/state/`; require tool calls before reports

**FR-009:** The Anti-Patterns section in
`.agents/skills/kanbanzai-getting-started/SKILL.md` MUST contain a new
anti-pattern entry named "Shell-Querying Workflow State Files". This entry MUST:
(a) identify the detectable behaviour: running `cat`, `grep`, `find`, or
similar shell commands against `.kbz/state/`, `.kbz/index/`, or
`.kbz/context/` to retrieve entity data,
(b) explain why this is harmful: raw YAML bypasses lifecycle resolution,
computed fields, and cross-reference validation, producing subtly wrong
results,
(c) specify the correct MCP tool to use for each query type:
`entity(action: "get")` for entity status, `status()` for project overview,
`knowledge(action: "list")` for knowledge entries, `doc(action: "get")` for
documents, and
(d) conclude with a prohibition on reading `.kbz/state/` files with shell
tools or `read_file`.

**FR-010:** The `.kbz/skills/write-research/SKILL.md` file (or, if it has no
anti-patterns or checklist section, `.kbz/roles/researcher.yaml`) MUST contain:
(a) two pre-writing checklist items requiring the agent to call
`retro(action: "synthesise")` and `knowledge(action: "list")` before writing
any report, and
(b) an anti-pattern entry named "Report From Memory" that identifies writing a
retro or research report without first calling those two tools, explains that
in-session memory misses signals from prior sessions, and directs the agent to
treat the synthesised output as the primary input.

#### REC-08: Require BUG entity before marking task done when tests fail intermittently

**FR-011:** The checklist in `.kbz/skills/implement-task/SKILL.md` MUST contain
an item, positioned before the `finish` step and after the acceptance-criteria
verification item, requiring the agent to file a BUG entity before proceeding
if any test failed intermittently (passed on retry without a code change).

**FR-012:** Phase 4 ("Verify") in `.kbz/skills/implement-task/SKILL.md` MUST
expand the "Run the full test suite" step to cover intermittent failures. The
expansion MUST:
(a) define what counts as an intermittent failure: a test that fails then
passes on retry without any code change,
(b) specify the exact `entity(action: "create", type: "bug", ...)` call to use,
including the required fields: `name`, `observed`, `expected`, `severity`,
and `priority`,
(c) require recording the BUG ID in the task completion summary, and
(d) explicitly prohibit marking the task done without filing the BUG entity
first.

**FR-013:** The Anti-Patterns section in `.kbz/skills/implement-task/SKILL.md`
MUST contain a new entry named "Unreported Flaky Test". This entry MUST:
(a) identify the detectable behaviour: observing a test that fails then passes
on retry without a code change and calling `finish` without filing a BUG entity,
(b) explain why this is harmful: future agents re-investigate the same failure
with no prior context and may make the same incorrect "probably fine" call, and
(c) direct the agent to file a BUG entity for every observed intermittent
failure, including the test name, failure message, and conditions of
observation.

### Non-Functional Requirements

**FR-NF-001:** Every added or replaced block of text MUST preserve the heading
hierarchy and list structure conventions already present in the file being
edited. An agent reading the modified file MUST not encounter broken Markdown
(unclosed fences, mismatched heading levels, orphaned bullets).

**FR-NF-002:** No existing requirement, rule, or anti-pattern in any of the
target files MUST be removed or weakened, except for the explicit replacements
mandated by FR-001 and FR-002 (which replace the stash instruction with a
stricter rule).

---

## Constraints

- **No code changes.** No Go source files, test files, binary artefacts, or
  generated files are touched.
- **No new files.** All changes are edits to existing files. New files MUST NOT
  be created.
- **No new MCP tools or handlers.** The feature is documentation-only.
- **No schema changes.** No YAML models, config structs, or storage formats are
  modified.
- **Scoped placement.** Each change lands in the specific section named in its
  requirement. No other sections in the target files are modified.
- **REC-07b target file** — FR-010 targets `.kbz/skills/write-research/SKILL.md`
  as the primary file. The implementer MUST inspect that file first; if it
  lacks an anti-patterns or checklist section, the change goes to
  `.kbz/roles/researcher.yaml` instead. The implementer MUST NOT add the change
  to both files.

---

## Acceptance Criteria

**AC-001 (FR-001):** Given the Session Start Checklist in
`kanbanzai-getting-started/SKILL.md`, when an agent reads it, then the text
"stash incomplete work" (or any equivalent instruction to stash) MUST NOT
appear, the checklist MUST direct the agent to commit complete changes, MUST
direct the agent to inform the human and stop for incomplete or mismatched
changes, and MUST explicitly state that `git stash` is prohibited.

**AC-002 (FR-002):** Given the "Clean slate" prose section in
`kanbanzai-getting-started/SKILL.md`, when an agent reads it, then it MUST
describe two and only two actions for uncommitted changes (commit or stop),
MUST contain the sentence "Never use `git stash` in a Kanbanzai project" (or
equivalent verbatim prohibition), and MUST explain the worktree-switch data
loss risk.

**AC-003 (FR-003):** Given the second bullet in the "Before Every Task"
checklist in `AGENTS.md`, when an agent reads the "do not stash" instruction,
then a parenthetical MUST be present that mentions both hiding state from
parallel agents and silent loss across worktree switches.

**AC-004 (FR-004):** Given the "Commit workflow state" prose section in
`kanbanzai-getting-started/SKILL.md`, when an agent reads it, then the section
MUST appear after "Clean slate", MUST name the three directories to inspect
(`.kbz/state/`, `.kbz/index/`, `.kbz/context/`), MUST contain the exact `git
add .kbz/` commit sequence, and MUST explain the consequence of ignoring
orphaned files (conflicting transitions by parallel agents).

**AC-005 (FR-005):** Given the "Before Every Task" checklist in `AGENTS.md`,
when an agent counts the checklist items, then a dedicated item for committing
orphaned workflow state MUST exist as a separate item from the `git status`
code-change item.

**AC-006 (FR-006):** Given the "Worktree File Editing" section in
`implement-task/SKILL.md`, when an agent reads it, then it MUST appear between
"Anti-Patterns" and "Checklist", MUST contain a warning block about `edit_file`
targeting the main working tree, MUST contain the `python3 -c` pattern, and
MUST contain the heredoc alternative.

**AC-007 (FR-007):** Given the checklist in `implement-task/SKILL.md`, when an
agent reads the items in order, then a worktree-confirmation item MUST appear
immediately after the "Claimed the task" item and before the "Read the context
packet" item.

**AC-008 (FR-008):** Given the "Delegating to Sub-Agents" section in `AGENTS.md`,
when an agent reads it, then a callout referencing the `implement-task` skill
and the `python3 -c` pattern MUST appear at the end of the section.

**AC-009 (FR-009):** Given the Anti-Patterns section in
`kanbanzai-getting-started/SKILL.md`, when an agent reads it, then a
"Shell-Querying Workflow State Files" entry MUST be present that names all
three prohibited directories, names at least `cat`, `grep`, and `find` as
prohibited tools, and maps each query type to its correct MCP tool.

**AC-010 (FR-010):** Given the target file for REC-07b (confirmed during
implementation), when an agent reads it, then two pre-writing checklist items
MUST be present (`retro(action: "synthesise")` and `knowledge(action: "list")`),
and a "Report From Memory" anti-pattern entry MUST be present.

**AC-011 (FR-011):** Given the checklist in `implement-task/SKILL.md`, when an
agent reads the pre-`finish` items, then an item requiring BUG entity creation
for intermittent test failures MUST appear after the acceptance-criteria
verification item and before the `finish` item.

**AC-012 (FR-012):** Given Phase 4 in `implement-task/SKILL.md`, when an agent
reads the test-suite step, then intermittent failure handling MUST be described,
the `entity(action: "create", type: "bug", ...)` call MUST be present with the
required fields (`name`, `observed`, `expected`, `severity`, `priority`), and
the step MUST state that the BUG ID must be recorded in the task completion
summary.

**AC-013 (FR-013):** Given the Anti-Patterns section in
`implement-task/SKILL.md`, when an agent reads it, then an "Unreported Flaky
Test" entry MUST be present that defines the detectable behaviour (fail → pass
on retry without code change), explains the compounding cost of not recording
it, and directs the agent to file a BUG entity with test name, failure message,
and conditions.

---

## Verification Plan

| Criterion | Method      | Description |
|-----------|-------------|-------------|
| AC-001    | Inspection  | Read the Session Start Checklist in `kanbanzai-getting-started/SKILL.md`; confirm absence of "stash" instruction and presence of "never stash" prohibition. |
| AC-002    | Inspection  | Read the "Clean slate" prose section; confirm two-case structure and verbatim prohibition of `git stash`. |
| AC-003    | Inspection  | Read the second bullet of the Before Every Task checklist in `AGENTS.md`; confirm parenthetical explanation is present. |
| AC-004    | Inspection  | Read the "Commit workflow state" prose section; confirm placement, three directory names, commit command sequence, and consequence explanation. |
| AC-005    | Inspection  | Count checklist items in `AGENTS.md` Before Every Task section; confirm orphaned-state item exists as a separate bullet. |
| AC-006    | Inspection  | Read `implement-task/SKILL.md`; confirm new section appears between Anti-Patterns and Checklist headings, contains warning block, `python3 -c` pattern, and heredoc alternative. |
| AC-007    | Inspection  | Read the checklist in `implement-task/SKILL.md` in order; confirm worktree-confirmation item is the second item (immediately after "Claimed the task"). |
| AC-008    | Inspection  | Read the "Delegating to Sub-Agents" section in `AGENTS.md`; confirm callout is present at end of section. |
| AC-009    | Inspection  | Read Anti-Patterns in `kanbanzai-getting-started/SKILL.md`; confirm new entry is present with all required content. |
| AC-010    | Inspection  | Read the confirmed REC-07b target file; confirm two pre-writing checklist items and "Report From Memory" anti-pattern are present. |
| AC-011    | Inspection  | Read the checklist in `implement-task/SKILL.md`; confirm BUG-filing item appears in the correct position relative to acceptance-criteria and `finish` items. |
| AC-012    | Inspection  | Read Phase 4 in `implement-task/SKILL.md`; confirm intermittent-failure expansion is present with all required fields in the `entity(action: "create")` call. |
| AC-013    | Inspection  | Read Anti-Patterns in `implement-task/SKILL.md`; confirm "Unreported Flaky Test" entry is present with detectable behaviour, cost explanation, and resolution. |

---

## Dependencies and Assumptions

- **All target files exist.** The five files named in this specification
  (`.agents/skills/kanbanzai-getting-started/SKILL.md`, `AGENTS.md`,
  `.kbz/skills/implement-task/SKILL.md`, `.kbz/skills/write-research/SKILL.md`,
  `.kbz/roles/researcher.yaml`) exist in the repository. If any are absent,
  the corresponding requirement is blocked.
- **REC-07b target file confirmation.** FR-010 requires the implementer to
  inspect `.kbz/skills/write-research/SKILL.md` before writing. The choice of
  target file (skill vs. role) is an implementation-time decision, not a
  specification decision.
- **Section names are stable.** The section headings in each target file named
  above (e.g. "Anti-Patterns", "Checklist", "Phase 4: Verify", "Delegating to
  Sub-Agents") exist and have not been renamed. If a heading has been renamed,
  the implementer must locate the equivalent section by intent.
- **No concurrent changes.** No other in-flight change modifies the same
  sections of the same files. Merge conflicts are out of scope for this
  specification.
```

Now let me register and approve the specification, and simultaneously read the architect role and dev-plan skill files: