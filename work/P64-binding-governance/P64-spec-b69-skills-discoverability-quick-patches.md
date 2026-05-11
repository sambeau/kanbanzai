# Specification: Skills Discoverability Quick Patches

| Field   | Value                                                |
|---------|------------------------------------------------------|
| Date    | 2026-05-11                                           |
| Batch   | B69-skills-discoverability-quick-patches             |
| Plan    | P64-binding-governance                               |
| Status | approved |
| Tier    | bug_fix (per feature)                                |
| Author  | Architect                                            |

---

## Overview

Three small, focused patches to restore agent awareness of skills before the larger
binding-governance refactor (`B70-binding-governance-impl`) lands. Diagnosed in the
P64 research report: agents — both orchestrators and casual chat sessions — currently
do not proactively discover or follow skills. This batch addresses the three observed
discovery gaps with no schema changes and no breaking behaviour. Each patch ships
independently.

## Scope

**In scope (this batch):**

1. Add dispatch and worktree-write directives to handoff-assembled prompts when the
   binding's orchestration is `orchestrator-workers`.
2. Replace the path pointer in `status()`'s orientation block with an inline skills
   index plus a context-aware suggestion based on observed work shape.
3. Reject non-canonical document paths in `doc(action: register)` with an error that
   includes the canonical path.

**Out of scope (deferred to B70):**

- Tier-aware routing (`retro_fix`, `bug_fix` pipeline binding).
- Wiring `BindingRegistry.Load` (and `ValidateBindingFile`) into server startup.
- Embedded-vs-canonical `stage-bindings.yaml` synchronisation.
- Repairing `validStages` and `workableStatuses` allowlists.
- Bug-pipeline design (separate research needed).

## Functional Requirements

**FR-1 (FEAT-01KRBB4AGXJDY): Sub-agent prompt directives**

When `internal/context/pipeline.go RenderPrompt` assembles a prompt and the resolved
`StageBinding.Orchestration == "orchestrator-workers"`:

- FR-1.1: Prepend a "Dispatch Directive" block above the existing constraints block
  that names the dispatch contract (orchestrators dispatch via `spawn_agent`,
  sub-agents execute the procedure).
- FR-1.2: When the parent feature has an active worktree (resolvable via the existing
  `WorktreeStore`), include a worktree-write directive in the constraints block:
  use `write_file(entity_id=...)` or `kanbanzai_edit_file(entity_id=...)`, never
  plain `write_file` or shell redirection.

**FR-2 (FEAT-01KRBB4AM4FQT): Status orientation skills index**

When `internal/mcp/status_tool.go synthesiseProject` builds the orientation block
returned by `status()` with no entity ID:

- FR-2.1: Replace the single-line skill-path pointer with an inline list of available
  `.agents/skills/kanbanzai-*` skills, each with a one-line summary parsed from the
  skill's frontmatter or first paragraph.
- FR-2.2: Add a context-aware suggestion line based on observed project state:
  - If no task is currently active and no recent task activity → suggest
    `kanbanzai-documents` skill.
  - If any feature is in an active stage → suggest `kanbanzai-workflow` skill.
  - Otherwise → suggest `kanbanzai-getting-started` skill.

**FR-3 (FEAT-01KRBB4HNPPWF): Doc register canonical-path validation**

When `internal/mcp/doc_tool.go` handles `action: register`:

- FR-3.1: Compute the canonical path using the same logic as `action: path` for the
  supplied `parent`, `type`, `title`, and other relevant fields.
- FR-3.2: If the supplied `path` does not match the canonical path, return an error
  that includes both the supplied and canonical paths.
- FR-3.3: If `action: path` cannot be computed (e.g. due to known parent-lookup bugs),
  fall through to the current accept-any-path behaviour with a warning rather than
  blocking the registration.

## Non-Functional Requirements

- **Backward compatibility:** No schema changes. No changes to the binding YAML, the
  skill files, the role files, or any state file format.
- **No regressions:** Existing `next`/`handoff`/`status`/`doc` callers must continue
  to work. New behaviour is additive (FR-1, FR-2) or defensive (FR-3).
- **Test coverage:** Each feature ships with at least one test that asserts the new
  behaviour and at least one that asserts the previous behaviour is preserved on the
  non-triggering path.
- **Performance:** None of the three changes introduce new I/O on the hot path.
  FR-2's skill summary parsing happens at startup, not per `status()` call.

## Acceptance Criteria

- **AC-1 (FR-1.1):** Calling `handoff(task_id=X)` where `X`'s parent feature is in
  `developing` (or any other `orchestrator-workers`-bound stage) returns a `prompt`
  string whose first non-frontmatter block is the Dispatch Directive naming both
  audiences (orchestrator and sub-agent).
- **AC-2 (FR-1.1):** Calling `handoff(task_id=X)` where `X`'s parent stage is
  `single-agent` (e.g. `designing`, `specifying`) does NOT add the Dispatch Directive.
- **AC-3 (FR-1.2):** When the feature has an active worktree, the constraints block
  includes the worktree-write directive naming `entity_id` as the required parameter.
- **AC-4 (FR-2.1):** Calling `status()` with no ID returns an `orientation` field
  whose body contains an enumerated list of `.agents/skills/kanbanzai-*` skill names
  with one-line summaries each.
- **AC-5 (FR-2.2):** Calling `status()` in a project with no active tasks returns an
  orientation suggestion for `kanbanzai-documents`. Calling `status()` in a project
  with at least one active feature returns a suggestion for `kanbanzai-workflow`.
- **AC-6 (FR-3.1, FR-3.2):** Calling `doc(action: register)` with a path that does
  not match the canonical path for the same parent/type/title returns an error whose
  message contains both the supplied path and the canonical path.
- **AC-7 (FR-3.2):** Calling `doc(action: register)` with the canonical path (the one
  returned by `doc(action: path)` for the same parameters) succeeds.
- **AC-8 (FR-3.3):** Calling `doc(action: register)` for a parent whose canonical
  path cannot be computed (e.g. strategic-plan parent lookup bug) registers
  successfully with the supplied path and emits a warning.
- **AC-9 (regression):** All existing tests in `internal/context/`, `internal/mcp/`,
  and `internal/binding/` packages continue to pass.

## Implementation Notes

- FR-1 should reuse the existing position-ordered section assembly in
  `stepAssembleSections`. The Dispatch Directive is a new section at a position
  prior to `PositionIdentity`. Suggested constant: `PositionDispatchDirective = 0`
  with all existing positions incremented.
- FR-2's skill-summary parsing should look for a YAML frontmatter `summary:` field
  first; fall back to the first paragraph below the title if missing.
- FR-3 should use the `doc.path` action's existing internal function rather than
  re-implementing path computation. The validation runs before the file write.

## Dependencies

None between the three features. They can be implemented in parallel or in any order.
Suggested order (per the discussion in P64 research):

1. FR-3 first (smallest, immediate signal),
2. FR-2 second (orients new sessions correctly),
3. FR-1 third (biggest impact on the orchestrator symptom).

## Verification

- `go test ./internal/context/... ./internal/mcp/... ./internal/binding/...` passes.
- Manual smoke test: claim any task in a `developing`-stage feature with an active
  worktree, call `handoff` against it, confirm both directives appear at the top of
  the prompt.
- Manual smoke test: call `status()` with no ID in a clean session, confirm the
  orientation block lists the skills inline.
- Manual smoke test: attempt `doc(action: register)` with a deliberately wrong path
  and confirm the error message includes the canonical path.
