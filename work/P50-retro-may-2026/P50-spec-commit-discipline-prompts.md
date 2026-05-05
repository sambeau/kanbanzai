# Specification: Commit Discipline Prompts

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-04                     |
| Status | approved |
| Author | spec-author                    |

## Overview

This specification implements the commit discipline prompts described in
`work/P50-retro-may-2026/P50-design-retro-may-2026.md`
(DOC-`P50-retro-may-2026/design-p50-design-retro-may-2026`), Feature 4.

MCP tool calls that modify `.kbz/state/` or `.kbz/index/` produce
uncommitted changes on disk. Agents sometimes proceed through multiple
state-mutating operations without committing, leaving orphaned state if a
session is interrupted. The current workflow relies on the pre-task
checklist's `git status` check to catch orphaned state after the fact.
Two changes close the gap: a `state_modified` flag on tool responses so
agents know when to commit, and a strengthened session-start check that
requires committing orphaned state before starting new work.

## Scope

**In scope:**
- `state_modified` flag on MCP tool responses for state-mutating operations
- Strengthened session-start `git status` check in kanbanzai-getting-started skill
- Corresponding skill file updates for kanbanzai-agents skill

**Out of scope:**
- Auto-commit behaviour — the system prompts; the agent decides when to commit
- Adding the flag to every tool response (start with high-mutation tools)
- Changes to the `finish` tool's own commit behaviour

## Functional Requirements

- **REQ-001:** When an MCP tool call modifies `.kbz/state/`, `.kbz/index/`,
  or `.kbz/context/`, the tool's response must include a `state_modified`
  field set to `true`.
- **REQ-002:** When an MCP tool call does not modify any of these
  directories, the `state_modified` field must be absent or set to `false`.
- **REQ-003:** The `state_modified` flag must be present on the
  high-mutation tools first: `entity` (transition, update, create),
  `doc` (register, approve, supersede), `knowledge` (contribute, confirm,
  retire), and `finish`. Remaining tools may be instrumented in a
  follow-up pass.
- **REQ-004:** The kanbanzai-agents skill must include a rule: "When a
  tool response contains `state_modified: true` and you are not about to
  make another state-mutating call, commit the `.kbz/` changes before
  proceeding."
- **REQ-005:** The kanbanzai-getting-started skill's pre-task checklist
  must be strengthened: if `git status` shows modified or untracked files
  under `.kbz/state/`, `.kbz/index/`, or `.kbz/context/`, the agent must
  commit them before starting any new work. The checklist must explicitly
  forbid stashing or discarding these files.
- **REQ-006:** Skill file changes must follow the dual-write rule: changes
  to `.agents/skills/kanbanzai-*/SKILL.md` must be mirrored in
  `internal/kbzinit/skills/*/SKILL.md`.

## Non-Functional Requirements

- **REQ-NF-001:** The `state_modified` flag must not add measurable
  latency to tool responses — it is a boolean set based on whether the
  handler wrote to `.kbz/`.
- **REQ-NF-002:** The flag must not leak through to MCP client error
  handling — clients that do not understand `state_modified` must not
  reject or fail on responses that include it.

## Constraints (Scope Exclusions)

- This specification does NOT introduce auto-commit behaviour. The agent
  retains control over when commits happen.
- The `state_modified` flag is a response field, not a new tool action.
- Skill file changes must not alter the tone or structure of the existing
  skill documents beyond the specific rule additions described here.
- The flag must be additive to existing response schemas — no existing
  response fields may be removed or renamed.

## Acceptance Criteria

- **AC-001 (REQ-001):** Given an `entity(action: "transition")` call that
  successfully changes an entity's status, when the response is returned,
  then `state_modified` is `true`.
- **AC-002 (REQ-002):** Given a `status` call (read-only query), when the
  response is returned, then `state_modified` is absent or `false`.
- **AC-003 (REQ-003):** Given each of the high-mutation tools (`entity`
  mutation actions, `doc` mutation actions, `knowledge` mutation actions,
  `finish`), when a mutation is performed, then the response includes
  `state_modified: true`.
- **AC-004 (REQ-004):** Given the kanbanzai-agents skill file, when
  searching for the `state_modified` rule, then the rule is present and
  instructs the agent to commit `.kbz/` changes before proceeding when the
  flag is seen.
- **AC-005 (REQ-005):** Given the kanbanzai-getting-started skill file,
  when the pre-task checklist is inspected, then it requires committing
  orphaned `.kbz/` files and explicitly forbids stashing or discarding
  them.
- **AC-006 (REQ-006):** Given a change to
  `.agents/skills/kanbanzai-agents/SKILL.md`, then the corresponding
  change exists in `internal/kbzinit/skills/agents/SKILL.md`.

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Test | Unit test: entity transition, assert response has state_modified=true |
| AC-002 | Test | Unit test: status query, assert state_modified is absent/false |
| AC-003 | Test | Table-driven test: exercise mutation actions on entity/doc/knowledge/finish, assert flag true |
| AC-004 | Inspection | Code review: verify kanbanzai-agents SKILL.md contains the state_modified rule |
| AC-005 | Inspection | Code review: verify kanbanzai-getting-started SKILL.md checklist includes the strengthened git status check |
| AC-006 | Inspection | Diff review: verify dual-write rule applied to skill file changes |
| REQ-NF-001 | Test | Benchmark entity transition with and without state_modified flag, assert no statistically significant latency increase |
| REQ-NF-002 | Test | Send tool responses with state_modified field to a vanilla MCP client, assert client does not reject or error on the unknown field |
