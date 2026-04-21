# Specification: Implementation Workflow Documentation Improvements

**Feature:** FEAT-01KPQ08YH16WZ
**Plan:** P25 — Agent Tooling and Pipeline Quality
**Design:** work/design/p25-impl-workflow-docs.md
**Status:** Draft

---

## Overview

This specification defines the documentation changes required to close three
agent workflow gaps identified in P24 post-mortems: (1) incorrect primary
recommendation for file writes inside Git worktrees in `implement-task/SKILL.md`,
(2) missing manual fallback guidance when `decompose propose` fails in
`decompose-feature/SKILL.md`, and (3) missing per-task sub-agent sizing guidance
for large-file features in `orchestrate-development/SKILL.md`. All changes are
documentation-only; no MCP server code or tool signatures are modified.

---

## Scope

### In Scope

- Targeted edits to `.kbz/skills/implement-task/SKILL.md` to swap heredoc and
  `python3 -c` recommendation order for worktree file writes.
- Targeted additions to `.kbz/skills/decompose-feature/SKILL.md` to document
  the manual `entity(action: "create")` fallback when `decompose propose` fails.
- Targeted additions to `.kbz/skills/orchestrate-development/SKILL.md` to add
  one-sub-agent-per-task sizing guidance for features involving large source files.

### Out of Scope

- MCP server code changes of any kind.
- New tools or modifications to existing tool signatures.
- Changes to `.agents/skills/` (kanbanzai system skills). These files have a
  dual-write requirement handled by FEAT-01KPQ08YKHNS9.
- Changes to `internal/kbzinit/skills/` (the three files edited here are
  task-execution skills under `.kbz/skills/`, not embedded kbzinit skills).
- Rewrites or restructuring of the three skill files beyond what is specified here.
- Changes to any other skill files.

---

## Functional Requirements

### FR-001: Heredoc as primary worktree file-write pattern

`implement-task/SKILL.md` MUST present heredoc (`cat > file << 'GOEOF'`) as the
**primary** recommended pattern for writing Go source files inside Git worktrees.
`python3 -c` with triple-quoted strings MUST be demoted to a **secondary** pattern,
scoped explicitly to Markdown and YAML files where Python string escaping is not
a problem.

**Acceptance criteria:**
- The section that formerly listed `python3 -c` first now lists heredoc first.
- The `python3 -c` example remains present and is scoped to non-Go files.
- A note is present explaining why `python3 -c` is not recommended for Go source
  files (triple-quote collision).

### FR-002: Heredoc delimiter collision warning

`implement-task/SKILL.md` MUST include a known-failure note in the heredoc section:
if the file content contains a line that is exactly the delimiter (`GOEOF`), the
heredoc will silently truncate at that line. The note MUST state that the fix is
to choose a different delimiter (e.g. `GOEOF2`, `ENDOFFILE`).

**Acceptance criteria:**
- A visible warning or note about delimiter collision is present near the heredoc
  code example.
- The note names `GOEOF` as the standard delimiter and gives at least one
  alternative.

### FR-003: Standard heredoc delimiter is `GOEOF`

`implement-task/SKILL.md` MUST use `GOEOF` as the standard heredoc delimiter in
code examples for Go source files. The skill MUST state that any unique uppercase
string may substitute if a collision is encountered.

**Acceptance criteria:**
- All heredoc code examples in the worktree file-editing section use `GOEOF` as
  the delimiter.
- The skill text acknowledges that the delimiter can be changed to avoid collisions.

### FR-004: Checklist item updated

Any checklist item in `implement-task/SKILL.md` that previously referenced
`python3 -c` as the default file-write method for Go files MUST be updated to
reference heredoc as the default for Go files, with `python3 -c` for Markdown/YAML.

**Acceptance criteria:**
- No checklist item in the file instructs agents to use `python3 -c` for Go source
  file writes as the primary recommendation.

### FR-005: Decompose propose failure fallback — Phase 2 note

`decompose-feature/SKILL.md` MUST include a note at the end of the Phase 2
(Generate Initial Decomposition) procedure stating that if `decompose propose`
returns a proposal with empty task names, wrong task count, or a structure that
cannot be corrected by adjusting input context, the agent MUST NOT call
`decompose apply` and MUST proceed to the manual fallback.

**Acceptance criteria:**
- The note is present in or immediately after the Phase 2 steps.
- The note names the triggering conditions: empty task names, wrong task count,
  structurally uncorrectable proposal.
- The note directs the agent to the manual fallback section.

### FR-006: Manual entity fallback — Phase 4 addition

`decompose-feature/SKILL.md` MUST include a "Manual Fallback" subsection in Phase
4 (Apply the Decomposition) describing how to create tasks directly via
`entity(action: "create", type: "task")` when `decompose propose` has failed.

**Acceptance criteria:**
- The subsection is present in Phase 4, after the existing apply step.
- The subsection specifies the required fields: `name`, `summary`,
  `parent_feature`.
- The subsection specifies the recommended optional field: `depends_on` (array of
  TASK-... IDs for dependency wiring).
- The subsection states that tasks should be created in dependency order so IDs
  are available before they are needed in `depends_on` references.
- A minimal wiring example showing the `depends_on` field format is included.
- The fallback is presented as an escape hatch, not the default path.

### FR-007: Verification step after manual task creation

The manual fallback subsection in `decompose-feature/SKILL.md` MUST instruct
agents to verify the created tasks with `status(id: "FEAT-xxx")` before
proceeding.

**Acceptance criteria:**
- The verification instruction is present in the manual fallback subsection.

### FR-008: Per-task sizing rule for large-file features

`orchestrate-development/SKILL.md` MUST include a sizing rule in Phase 3
(Dispatch Sub-Agents), before the `handoff` invocation step, stating that for
features with more than three tasks where tasks involve reading or rewriting
source files longer than approximately 300 lines, one sub-agent MUST be
dispatched per task rather than assigning multiple tasks to a single agent.

**Acceptance criteria:**
- The sizing rule is present in Phase 3 before the dispatch steps.
- The rule states the threshold conditions: more than three tasks AND tasks involve
  large source files (~300 lines).
- The rule states the required dispatch pattern: one sub-agent per task.
- The rule explains the rationale: full-file rewrites saturate context windows
  when multiple tasks are assigned to one agent.

### FR-009: Batch dispatch remains appropriate for small-file or doc-only features

The sizing rule added by FR-008 MUST explicitly state that batch dispatch remains
appropriate for features with small files or documentation-only tasks.

**Acceptance criteria:**
- The sizing rule text includes a qualifier indicating that per-task isolation is
  not required for small-file or documentation-only features.

### FR-010: Anti-pattern entry for over-sized dispatch

`orchestrate-development/SKILL.md` MUST include an anti-pattern entry in the
`## Anti-Patterns` section for assigning multiple large-file tasks to a single
sub-agent.

**Acceptance criteria:**
- An anti-pattern entry is present in the Anti-Patterns section describing the
  failure mode of assigning multiple large-file tasks to one sub-agent.
- The anti-pattern entry is consistent with the sizing rule added by FR-008.

---

## Non-Functional Requirements

### NFR-001: Structure preservation

The overall section structure, headings, and ordering of each of the three skill
files MUST be preserved. Changes are additions or targeted substitutions within
existing sections; the files are not restructured.

### NFR-002: No cross-file consolidation

The three changes MUST be delivered as independent edits to three separate files.
No new composite document or aggregated skill file is created.

### NFR-003: No invented patterns

All guidance added by these changes MUST document existing, proven workarounds
observed in production pipeline runs. No new patterns are invented or introduced
speculatively.

---

## Acceptance Criteria (Summary)

| ID | Criterion |
|----|-----------|
| AC-01 | `implement-task/SKILL.md`: heredoc is listed before `python3 -c` in the worktree file-editing section |
| AC-02 | `implement-task/SKILL.md`: `python3 -c` is scoped to Markdown and YAML files |
| AC-03 | `implement-task/SKILL.md`: delimiter collision note present, naming `GOEOF` and giving alternative |
| AC-04 | `implement-task/SKILL.md`: checklist updated to reflect heredoc as primary for Go files |
| AC-05 | `decompose-feature/SKILL.md`: Phase 2 note present directing agent to fallback on broken proposal |
| AC-06 | `decompose-feature/SKILL.md`: Phase 4 manual fallback subsection present with required fields |
| AC-07 | `decompose-feature/SKILL.md`: fallback includes `depends_on` wiring example |
| AC-08 | `decompose-feature/SKILL.md`: fallback instructs verification with `status()` |
| AC-09 | `orchestrate-development/SKILL.md`: sizing rule present in Phase 3 with threshold and rationale |
| AC-10 | `orchestrate-development/SKILL.md`: sizing rule explicitly exempts small-file/doc-only features |
| AC-11 | `orchestrate-development/SKILL.md`: Anti-Patterns section includes over-sized dispatch entry |
| AC-12 | No MCP server code files are modified |
| AC-13 | No `.agents/skills/` or `internal/kbzinit/skills/` files are modified |

---

## Dependencies and Assumptions

### Dependencies

- **FEAT-01KPQ08Y47522 (`write_file` MCP tool):** The heredoc guidance added by
  FR-001–FR-004 is a workaround for the absence of a worktree-safe file-writing
  tool. When `write_file` ships, the `## Worktree File Editing` section in
  `implement-task/SKILL.md` should be superseded to make `write_file` the
  primary recommendation. This feature does not block and is not blocked by
  FEAT-01KPQ08Y47522; they can be implemented independently.

- **FEAT-01KPQ08YKHNS9 (orchestration docs):** Also modifies
  `orchestrate-development/SKILL.md`. Both features must be implemented in
  separate commits or their changes to that file must be coordinated to avoid
  conflicts.

### Assumptions

- The three target skill files exist at the paths specified:
  - `.kbz/skills/implement-task/SKILL.md`
  - `.kbz/skills/decompose-feature/SKILL.md`
  - `.kbz/skills/orchestrate-development/SKILL.md`
- These files are task-execution skills (`.kbz/skills/`) and are **not** embedded
  in `internal/kbzinit/skills/`; no dual-write is required for any change in this
  feature.
- The `## Worktree File Editing` section and the checklist exist in
  `implement-task/SKILL.md`.
- Phase 2, Phase 4, and the `## Anti-Patterns` section exist in the respective
  target skill files.