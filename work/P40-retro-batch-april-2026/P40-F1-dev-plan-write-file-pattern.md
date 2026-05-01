# Dev-Plan: Document write_file as Primary Worktree Pattern (B39-F1)

| Field  | Value                         |
|--------|-------------------------------|
| Date   | 2026-04-30 (updated 2026-05-01) |
| Status | approved |
| Author | Architect                     |
| Feature | FEAT-01KQG1XWAZE8V           |
| Spec   | B39-fix-worktree-dev-experience/spec-p40-spec-b39-worktree-dev-experience |

---

## Overview

This is a single-file documentation change to `implement-task/SKILL.md`. No
code is modified. The work is to update the skill's file-writing guidance to
make `write_file(entity_id: ...)` the primary recommended pattern, demote
`python3 -c` and heredoc workarounds to fallback status, and add `write_file`
to the developing-stage default tool subset.

---

## Task Breakdown

### T1 — Update implement-task skill with write_file guidance

**Deliverable:** Edited `implement-task/SKILL.md` with updated file-writing
guidance section.

**Scope:**
- Locate the section(s) in `implement-task/SKILL.md` that describe how to write
  files in worktrees (currently recommending `python3 -c` or heredocs).
- Replace with `write_file(entity_id: "...", path: "...", content: "...")` as
  the primary recommendation, including explanation of the `entity_id` parameter.
- Mark any remaining `python3 -c` or heredoc references as fallback approaches.
- Ensure no procedure steps, anti-patterns, or vocabulary definitions are changed
  (REQ-NF-003, AC-011).

**Dependencies:** None. Standalone task.

**Verification:** Diff the file before and after; confirm `write_file(entity_id)`
appears as primary recommendation (AC-001); grep for `python3 -c` and `<<` to
confirm fallback status (AC-002); diff for non-file-writing changes (AC-011).

**Estimated effort:** 1 (single file, no code, pure text edit)

### T2 — Add write_file to developing-stage tool subset

**Deliverable:** Updated stage binding or role configuration with `write_file`
in the developing-stage tool subset.

**Scope:**
- Identify where the developing-stage tool subset is configured (stage bindings
  YAML or role configuration).
- Add `write_file` to the subset so sub-agents discover it automatically (AC-003).

**Dependencies:** None. Can run in parallel with T1.

**Verification:** Inspect the configuration to confirm `write_file` is present
in the developing-stage tool subset (AC-003).

**Estimated effort:** 0.5 (one-line config change)

**Status (2026-05-01):** ⚠️ T2 was marked `done` in the entity system but its
deliverable (`write_file` in `implementer.yaml` tool list) was never produced.
This was discovered during the B39 batch conformance review (CG-B39-1).
**Remediated 2026-05-01:** `write_file` added to `.kbz/roles/implementer.yaml`
tool list (commit pending). Since `implementer-go` inherits from `implementer`,
this satisfies AC-003 for both roles.

---

## Dependency Graph

```
T1 (update skill)
T2 (add to tool subset)
```

Both tasks are independent — no dependency edges. They touch different files
(`implement-task/SKILL.md` vs stage bindings/role config) and can run in parallel.

---

## Interface Contracts

No interface contracts — this feature touches no Go code.

---

## Traceability Matrix

| Requirement | Task(s) |
|-------------|---------|
| REQ-001 | T1 |
| REQ-002 | T1 |
| REQ-003 | T2 |
| REQ-NF-003 | T1 |
| AC-001 | T1 |
| AC-002 | T1 |
| AC-003 | T2 |
| AC-011 | T1 |
