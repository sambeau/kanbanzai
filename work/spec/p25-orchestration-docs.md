# Specification: Sub-agent Orchestration Documentation Improvements

**Feature:** FEAT-01KPQ08YKHNS9
**Plan:** P25 — Agent Tooling and Pipeline Quality
**Design:** work/design/p25-orchestration-docs.md
**Status:** Draft

---

## 1. Overview

This specification covers three targeted documentation improvements to reduce recurring
orchestration failures caused by undocumented rules. The changes add an explicit
"always use `handoff`" mandate with corresponding anti-pattern to
`orchestrate-development/SKILL.md`, a dual-write rule for embedded skill files to
`AGENTS.md`, and a corrected description of the `parent` parameter in
`internal/mcp/entity_tool.go`.

---

## 2. Scope

### In scope

- Update `.kbz/skills/orchestrate-development/SKILL.md` to add a "always use `handoff`"
  rule before Phase 3 dispatch steps and a corresponding anti-pattern entry.
- Update `AGENTS.md` to add a dual-write rule for `.agents/skills/kanbanzai-*/SKILL.md`
  changes that must also be applied to `internal/kbzinit/skills/`.
- Update the `parent` parameter description string in `internal/mcp/entity_tool.go` to
  correctly document both its feature-create use and its list-filter use, and to
  distinguish it from `parent_feature`.

### Out of scope

- Making `handoff` mandatory at the MCP server level (server-side enforcement).
- Auto-syncing `internal/kbzinit/skills/` from `.agents/skills/` on build.
- Modifying any skill file other than `orchestrate-development/SKILL.md`.
- Modifying any other tool description strings beyond the `parent` field in `entity_tool.go`.
- Any logic changes to the `entity` tool beyond the description string update.

---

## 3. Functional Requirements

### FR-001: `orchestrate-development/SKILL.md` — `handoff` mandate rule

The file `.kbz/skills/orchestrate-development/SKILL.md` MUST contain an explicit rule
statement before the numbered dispatch steps in Phase 3 declaring that `handoff(task_id)`
MUST always be used to generate sub-agent prompts and that manual prompt composition is
prohibited.

**Acceptance criteria:**
- Phase 3 of the Procedure section begins with a bold rule statement that uses the word
  "Always" and references `handoff(task_id: "TASK-xxx")`.
- The rule statement explains that manual composition silently omits graph project context,
  knowledge entries, and spec sections.
- The rule statement precedes the existing numbered steps in Phase 3.

### FR-002: `orchestrate-development/SKILL.md` — manual composition anti-pattern

The `## Anti-Patterns` section of `.kbz/skills/orchestrate-development/SKILL.md` MUST
contain a new entry describing the manual prompt composition anti-pattern.

**Acceptance criteria:**
- The anti-patterns section contains an entry titled "Manual Prompt Composition" (or
  equivalent).
- The entry includes a detection signal (how to recognise the pattern).
- The entry includes a BECAUSE clause explaining the root cause, specifically that the
  graph project name is omitted, causing codebase-memory-mcp tools to be unavailable to
  the sub-agent.
- The entry includes a resolution directive referencing `handoff(task_id)`.
- The entry references the P24 pipeline as a concrete prior example of the failure.

### FR-003: `AGENTS.md` — dual-write rule for embedded skill files

`AGENTS.md` MUST contain a subsection documenting the dual-write requirement: when any
file under `.agents/skills/kanbanzai-*/` is modified, the corresponding file under
`internal/kbzinit/skills/` MUST be updated in the same commit.

**Acceptance criteria:**
- The rule is placed in or near the `## Git Discipline` section of `AGENTS.md`.
- The rule explicitly names the correspondence:
  `.agents/skills/kanbanzai-<name>/SKILL.md` ↔ `internal/kbzinit/skills/<name>/SKILL.md`.
- The rule states that the update must be in the same commit as the source change.
- The rule explicitly excludes task-execution skills under `.kbz/skills/` from the
  dual-write requirement.

### FR-004: `AGENTS.md` — dual-write rule explains the embedding relationship

The dual-write rule in `AGENTS.md` MUST explain why the dual-write is required — that the
kanbanzai binary embeds `internal/kbzinit/skills/` for distribution to newly-initialised
projects via `kanbanzai init`.

**Acceptance criteria:**
- The subsection describes, in one or two sentences, that `kanbanzai init` installs the
  embedded skill files into new projects as `.agents/skills/kanbanzai-<name>/SKILL.md`.
- The subsection does not need to describe the `//go:embed` mechanism — only the
  functional consequence.

### FR-005: `entity_tool.go` — corrected `parent` field description

The `parent` parameter description in `internal/mcp/entity_tool.go` MUST be updated to
reflect that the field is required on feature create (to associate the feature with its
plan) and is also used as a filter on list calls.

**Acceptance criteria:**
- The description no longer contains the text `(list only)` applied to the `parent` field.
- The description states that `parent` is the parent plan ID for features and is required
  on feature create.
- The description states that `parent` is also used as a filter on list calls.
- The description contains a note distinguishing `parent` (used for features) from
  `parent_feature` (used for tasks).

### FR-006: No logic changes in `entity_tool.go`

The update to `entity_tool.go` MUST be limited to the description string for the `parent`
parameter. No logic, validation, routing, or schema changes are permitted as part of this
feature.

**Acceptance criteria:**
- The diff for `entity_tool.go` contains only a change to the `mcp.Description(...)` string
  for the `parent` parameter.
- All existing tests for `entity_tool.go` pass without modification.

### FR-007: MCP server rebuild required after `entity_tool.go` change

After merging the `entity_tool.go` description change, documentation or commit notes MUST
indicate that a rebuild and server restart are required before the updated description is
visible to MCP clients.

**Acceptance criteria:**
- The commit message for the `entity_tool.go` change, or an accompanying note in the PR
  description, states that `go install ./cmd/kanbanzai/` and a server restart are required.

---

## 4. Non-Functional Requirements

### NFR-001: Minimal diff surface

Each of the three changes MUST be a targeted addition or update within an existing
section. No existing sections, headings, or rules in the modified files may be removed or
restructured as part of this feature.

### NFR-002: Consistent voice and style

New text added to `.kbz/skills/orchestrate-development/SKILL.md` and `AGENTS.md` MUST
match the imperative, direct style already used in those files. Anti-pattern entries MUST
follow the existing format used in the `## Anti-Patterns` section of
`orchestrate-development/SKILL.md`.

### NFR-003: No dual-write required for this feature's own changes

The files modified by this feature (`.kbz/skills/orchestrate-development/SKILL.md`,
`AGENTS.md`, `internal/mcp/entity_tool.go`) are either project-local task-execution skills
or non-skill files. No dual-write to `internal/kbzinit/skills/` is required for this
feature's own changes.

---

## 5. Acceptance Criteria

### AC-001: orchestrate-development SKILL.md contains handoff mandate

Given the updated `.kbz/skills/orchestrate-development/SKILL.md`, when an agent reads
Phase 3 of the Procedure section, the agent encounters a bold rule statement mandating
`handoff(task_id)` use before any numbered dispatch steps.

### AC-002: orchestrate-development SKILL.md contains manual-composition anti-pattern

Given the updated file, the `## Anti-Patterns` section contains an entry that:
- Is titled to convey "manual prompt composition."
- Describes the consequence: graph project name is absent, codebase-memory-mcp tools
  are unavailable to the sub-agent.
- Directs the reader to use `handoff(task_id)` instead.

### AC-003: AGENTS.md dual-write rule is present and correct

Given the updated `AGENTS.md`, when an agent reads the Git Discipline section, the agent
finds a subsection or rule that:
- Names both the source path (`.agents/skills/kanbanzai-*/SKILL.md`) and the target path
  (`internal/kbzinit/skills/*/SKILL.md`).
- States the same-commit requirement.
- Excludes `.kbz/skills/` from the requirement.

### AC-004: entity_tool.go parent description is accurate

Given the updated `internal/mcp/entity_tool.go`, when an MCP client inspects the `entity`
tool schema, the `parent` parameter description:
- Does not say `(list only)`.
- Says the field is required on feature create to associate the feature with its plan.
- Distinguishes `parent` (features) from `parent_feature` (tasks).

### AC-005: No regressions in entity tool tests

All existing tests in `internal/mcp/entity_tool_test.go` (or equivalent) pass after the
description string change.

---

## 6. Dependencies and Assumptions

### Dependencies

- **`.kbz/skills/orchestrate-development/SKILL.md`** — must have an existing `## Anti-Patterns`
  section and a Phase 3 section for the additions to be inserted into. Verified present.
- **`AGENTS.md`** — must have a `## Git Discipline` section or equivalent for the dual-write
  rule placement. Verified present.
- **`internal/mcp/entity_tool.go`** — must contain the `parent` parameter with a
  `mcp.Description(...)` string. Verified present.
- **`internal/kbzinit/skills/`** — the embedded skill directory must exist and contain
  files corresponding to `.agents/skills/kanbanzai-*/SKILL.md`. Verified present; the
  dual-write rule added by this feature documents this relationship rather than modifying
  the embedded files themselves.

### Assumptions

- The existing anti-pattern entry format in `orchestrate-development/SKILL.md` uses
  detect / BECAUSE / resolve structure. New entries follow this structure.
- A Go rebuild and MCP server restart are sufficient to make description changes visible
  to clients; no cache invalidation step is required.
- The `AGENTS.md` Git Discipline section accepts additive subsections without restructuring.