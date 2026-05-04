# Design: Retrospective Fixes — May 2026

**Plan ID:** P50-retro-may-2026  
**Status:** Shaping  
**Source:** [Retrospective: Project — May 2026](../_project/report-2026-05-04.md)

## Overview

The May 2026 retrospective identified seven problems after filtering out artefacts of now-fixed
bugs. This design selects the highest-priority items worth fixing and proposes a batch of
features, each scoped to a single concern.

Three problems are deferred: P-4 (`needs-rework` lifecycle path) is partially addressed by
P43/B48 (fast-track validators at stage boundaries) and the remainder depends on fast-track
being live first. P-6 (worktree adoption) needs design work on plan-level worktree support
that's too large for this batch. P-7 (`entity.transition` dominance) should self-resolve as
automation matures.

## Goals and Non-Goals

**Goals:**
- Fix the error taxonomy so instrumentation is diagnostic — every failure has a meaningful `error_type`
- Improve `decompose propose` so agents stop bypassing it — fix proposal granularity and add a refuse-to-propose mode
- Add a `path` action to the `doc` tool so document placement conventions are enforced by tooling, not memory
- Add commit-discipline prompts to the workflow so management actions don't leave orphaned state on disk

**Non-Goals:**
- Not fixing `needs-rework` lifecycle path (fast-track dependency)
- Not adding plan/batch worktree support
- Not redesigning decompose from scratch — incremental improvements to proposal quality
- Not adding a full filename validator — a `path` action is sufficient for convention enforcement

## Design

### Feature 1: Error classification in MCP handlers

The `actionlog.Entry` struct already has an `ErrorType` field with five constants defined
(`gate_failure`, `validation_error`, `not_found`, `precondition_error`, `internal_error`).
The MCP handlers already return errors. The gap is purely that handlers don't classify errors
before returning them.

**Approach:** Add a thin classification layer. Each MCP handler's error path checks the
returned error against known patterns and sets `ErrorType` accordingly:

- `validation_error` — bad input (invalid entity ID format, missing required field)
- `not_found` — entity or document doesn't exist
- `gate_failure` — stage gate prerequisite not met (document not approved, tasks not terminal)
- `precondition_error` — entity in wrong state for the requested operation
- `internal_error` — everything else (nil pointer, unexpected failure)

Prioritise the high-volume tools first: `entity`, `finish`, `next`, `doc`, `decompose`.

**Files affected:** Each MCP tool handler in `internal/mcp/`. The logging hook in the MCP
server that calls `actionlog.Log()` already passes the entry — the handlers just need to
populate `ErrorType` before returning.

**Risks:** Low. This is additive — it doesn't change error behaviour, only labels what's
already happening. The constants already exist. Tests should verify that known failure
scenarios produce the expected `error_type`.

### Feature 2: Decompose proposal quality

Three changes to `decompose propose`:

**2a. Refuse-to-propose mode.** When the spec has no parseable acceptance criteria (after the
P24 AC format fixes), `decompose propose` currently produces a plausible-but-wrong heading-based
proposal. Change this to fail with a clear diagnostic: "Cannot decompose: no acceptance criteria
found in spec {spec_id}. Ensure the spec uses **AC-NN.** format and the index is current."

**2b. Implementation + test task pairs.** Instead of one task per AC, produce a pair:
an implementation task and a paired test task. The implementation task covers the production
code change; the test task covers the corresponding test. This matches how agents decompose
manually and eliminates the disconnect between `decompose` output and what agents actually need.

**2c. Dependency graph fix.** The dependency graph should not assume partial task completion.
Dependencies should be between complete tasks only. If P2 testing must happen before P3
descriptions, P2 testing must be a separate complete task, not a partial-completion state of
a combined task.

**Files affected:** `internal/mcp/decompose_tool.go`, the decompose proposal prompt assembly
in `internal/service/`.

**Risks:** Medium. The proposal prompt change (2b) could affect all existing decompose users.
Mitigation: make the paired-test-task behaviour configurable, defaulting to on. Existing
features that expect the old one-task-per-AC output can opt out.

### Feature 3: Document path tool

Add a `path` action to the `doc` MCP tool:

```
doc(action: "path", type: "design", parent: "P50-retro-may-2026")
→ work/P50-retro-may-2026/P50-design-retro-may-2026.md
```

The tool takes a document type (design, spec, dev-plan, research, report, prompt) and an
optional parent entity ID. It returns the canonical path using the convention from
`AGENTS.md` and KE-01KQSER9N0QHY: `work/{plan-slug}/{plan-id}-{type}-{topic}.md`.

For prompt files (which currently have no home): use `work/{plan-slug}/prompts/` when a parent
plan exists, or `work/_project/prompts/` for project-level prompts.

**Registration integration:** When `doc(action: "register")` is called with a path, optionally
validate that the path matches the canonical form and warn on mismatch.

**Files affected:** `internal/mcp/doc_tool.go`, plus a new path-generation function.

**Risks:** Low. This is a read-only utility tool. It doesn't change existing behaviour.

### Feature 4: Commit discipline prompts

Two changes to reduce orphaned state on disk:

**4a. Post-mutation commit prompt.** After any MCP tool call that writes to `.kbz/state/` or
`.kbz/index/` (entity transitions, doc registrations, knowledge contributions), the tool
response should include a `state_modified: true` flag. The agent's skill instructions should
treat this flag as a commit prompt: if you see `state_modified` and you're not about to make
another state-mutating call, commit.

**4b. Session-start detection.** The kanbanzai-getting-started skill's pre-task checklist
already includes a `git status` check. Strengthen it: if `git status` shows modified or
untracked files in `.kbz/`, the agent must commit them before starting any new work. This
closes the gap where state changes from a previous session were never committed.

**Files affected:** MCP tool response format (add `state_modified` field), skill files
(`.agents/skills/kanbanzai-agents/SKILL.md`, `.agents/skills/kanbanzai-getting-started/SKILL.md`).

**Risks:** Low for 4b (documentation change). Medium for 4a — adding a field to every tool
response is a cross-cutting change. Mitigation: start with the high-mutation tools (`entity`,
`doc`, `knowledge`, `finish`) and expand later.

## What's Not Included

| Problem | Reason for deferral |
|---------|-------------------|
| P-4 `needs-rework` lifecycle path | Partially addressed by P43/B48 fast-track validators. Rework-loop tracking depends on fast-track being live first. |
| P-6 Worktree adoption / main-branch drift | Requires design on plan-level worktree isolation — too large for this batch. B39+P42 already addressed the worktree editing tooling gap. |
| P-7 `entity.transition` dominance | Should self-resolve as fast-track automation reduces manual transitions. Revisit in the next retro. |
| Coordination server fallback behaviour | Discussed during this session but not in the retro report. Separate concern — defer to a dedicated plan when coordination is enabled. |

## Dependencies

- No dependency on P43/B48 (fast-track). These fixes are independent.
- F2 (decompose) depends on the P24 AC format fixes being live — they are (shipped 2026-04-21).
- F4 (commit prompts) requires skill file changes — follow the dual-write rule (`.agents/skills/` and `internal/kbzinit/skills/`).

## Alternatives Considered

### One big "fix everything" feature vs. four focused features

A single feature spanning error classification, decompose, doc paths, and commit prompts would
be ~4 concerns in one review cycle. Four focused features can be reviewed independently and
merged without cross-feature coordination.

**Decision:** Four features. Each is independently testable and shippable.

### Adding a `prompt` document type

The doc path tool needs to know where prompts go. Adding a full `prompt` document type to the
`doc` tool requires schema changes, registration support, and lifecycle integration. The
`path` action can handle prompts as a convention without a new type.

**Decision:** Defer the `prompt` document type. The `path` action handles placement; formal
registration can come later.

### Auto-commit vs. prompt-to-commit

The system could auto-commit `.kbz/state/` after every mutation. This is cleaner in theory but
risks committing broken state if a multi-step workflow fails partway through. Prompting the
agent to commit (the `state_modified` flag) keeps the agent in control of when commits happen.

**Decision:** Prompt-to-commit. Auto-commit is a future option if prompting proves insufficient.

## Open Questions

1. Should the `state_modified` flag be on every response or only on mutation responses?
   Mutation-only is simpler but requires the MCP framework to know which actions mutate.
   Every-response is safer but noisier.

2. For decompose 2b (paired test tasks), should the pairing be opt-in or default-on?
   Default-on with an opt-out flag is least disruptive to existing workflows.

3. Should the doc `path` action validate that the parent entity exists and has a slug?
   Probably yes — a path with a non-existent parent is a worse experience than no path at all.
