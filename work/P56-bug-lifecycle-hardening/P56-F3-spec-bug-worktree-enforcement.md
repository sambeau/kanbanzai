| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-07                     |
| Author | spec-author                     |
| Status | approved |
| Plan   | P56-bug-lifecycle-hardening    |
| Feature | FEAT-01KR12RE939M7             |

# Specification: Bug Worktree Enforcement

## Related Work

- **P56-design-bug-lifecycle-hardening.md** (P56-bug-lifecycle-hardening/design-p56-design-bug-lifecycle-hardening) — Design document. This spec implements Component E.
- **internal/service/status_transition_hook.go** — Existing `WorktreeTransitionHook` that auto-creates worktrees on `in-progress` transition. This spec adds detection of bare bugs.
- **internal/health/check.go** — Existing health check infrastructure. This spec adds a bug worktree health check.
- **internal/mcp/write_file_tool.go**, **internal/mcp/edit_file_tool.go** — Write tools that this spec extends with worktree warnings.

**Constraining decisions:**
- P56 Decision 5: Worktree enforcement is a health warning, not a hard block (initially).

## Overview

The worktree auto-creation hook already creates worktrees when bugs transition to `in-progress`. However, nothing detects when a bug is `in-progress` without a worktree, and nothing warns when file mutations happen at the repo root while active bug worktrees exist. This specification adds health checks and mutation warnings to close the isolation gap.

## Scope

**In scope:**
- Health check that flags `in-progress` bugs without active worktrees
- Warning on `kanbanzai_edit_file` / `write_file` calls without `entity_id` when active bug worktrees exist
- The cleanup check at close-out is deferred to F4 (verifier sub-agent checklist)

**Out of scope:**
- Hard-blocking repo-root mutations (deferred until after data gathering per Decision 5)
- Auto-creating worktrees (already implemented in `WorktreeTransitionHook`)
- Close-out verification of worktree removal (F4)
- Feature worktree enforcement (unchanged)

## Functional Requirements

### Pillar A — Health Check for Bare In-Progress Bugs

**FR-201:** A new health check category `bug_worktree` MUST be added to `internal/health/check.go`.

**FR-202:** The health check MUST iterate over all bugs with `status: in-progress` and verify that each has an active worktree record in the worktree store.

**FR-203:** For each `in-progress` bug without an active worktree, the health check MUST report a finding with:
- Severity: `warning`
- Entity ID: the bug ID
- Message: `"bug {display_id} is in-progress but has no active worktree — changes may not be isolated"`

**FR-204:** The health check MUST NOT flag bugs in other statuses (`reported`, `triaged`, `needs-review`, `verifying`, `closed`).

**FR-205:** The health check MUST be included in the default health check run (not gated behind a flag).

**Acceptance criteria:**
- `health()` reports a warning for an `in-progress` bug with no worktree
- `health()` reports no warning for an `in-progress` bug with an active worktree
- `health()` reports no warning for bugs in `reported`, `needs-review`, or `closed` status

### Pillar B — Repo-Root Mutation Warnings

**FR-206:** The `kanbanzai_edit_file` and `write_file` tool handlers MUST check for active bug worktrees when called without an `entity_id` parameter.

**FR-207:** When a file mutation targets the repo root (no `entity_id`) AND there exists at least one bug with `status: in-progress` that has an active worktree, the tool MUST include a warning in its response:

```
"warning: bug {display_id} is in-progress with an active worktree at {worktree_path}. Consider scoping your edit with entity_id: \"{bug_id}\" to isolate changes."
```

**FR-208:** The warning MUST be informational only — the file operation MUST proceed normally. This is not a block.

**FR-209:** If multiple bugs are `in-progress` with active worktrees, the warning MUST list the first one (to avoid response bloat) with a suffix: `" (and N other active bug worktrees)"`.

**FR-210:** The warning MUST NOT fire when the caller provides an `entity_id` (the caller has explicitly scoped the edit).

**Acceptance criteria:**
- Writing a file without `entity_id` when a bug is `in-progress` with an active worktree produces a warning in the response
- The file is still written successfully (warning is non-blocking)
- Writing with an `entity_id` produces no warning
- Writing without `entity_id` when no bugs are `in-progress` produces no warning

### Pillar C — Cleanup Verification

**FR-211:** The worktree and branch cleanup requirement is specified in the verifier checklist (F4, FR-404 item 8). This feature does not add independent cleanup checks — it relies on the verifier sub-agent to confirm cleanup at close-out.

**Acceptance criteria:**
- No duplicate cleanup checks between this feature and F4

## Non-Functional Requirements

**NFR-201:** The health check MUST complete in under 200ms for up to 50 bugs.

**NFR-202:** The repo-root mutation warning MUST NOT add more than 5ms overhead to file write operations when no active bug worktrees exist.

**NFR-203:** The warning message MUST be under 300 characters to avoid response bloat.

## Acceptance Criteria (Cross-Cutting)

**AC-201:** An `in-progress` bug without a worktree triggers a health warning. After `worktree(action: "create")` for that bug, the warning disappears.

**AC-202:** Writing to the repo root while a bug is `in-progress` with a worktree produces a warning. Writing with `entity_id` set to that bug's ID produces no warning.

**AC-203:** The health check and mutation warnings work together: a bug that transitions `in-progress → needs-review` stops triggering mutation warnings (it's no longer `in-progress`), and the health check no longer flags it.

## Dependencies and Assumptions

**Dependencies:**
- `internal/worktree/Store` — Used by the health check to query active worktrees.
- `internal/service/EntityService` — Used to list bugs and check statuses.
- F4 (Bug Close-Out Verification) — The cleanup verification is delegated to the verifier's checklist.

**Assumptions:**
1. The worktree auto-creation hook fires successfully for most `in-progress` transitions. Bare bugs are the exception, not the norm.
2. The health check has access to both the entity service and the worktree store.
3. The warning message does not need to be localised.
