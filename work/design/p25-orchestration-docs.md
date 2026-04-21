# Design: Sub-agent Orchestration Documentation Improvements

_Feature: FEAT-01KPQ08YKHNS9 | Plan: P25 — Agent Tooling and Pipeline Quality_

## Overview

Three documentation gaps caused recurring orchestration failures across recent sprints.

**Gap 1 — Missing `handoff` mandate (P8).** During the P24 implementation pipeline,
sub-agent prompts were composed manually rather than using the `handoff(task_id)` tool.
`orchestrate-development/SKILL.md` Phase 3 mentions `handoff` in a procedure step but
does not state it as a rule and carries no anti-pattern for manual composition. It also
does not explain what `handoff` provides beyond what a hand-written prompt contains.
Consequently, no P24 sub-agent received the `graph_project` name, the codebase-memory-mcp
graph tools went entirely unused, and at least one agent hit context limits from full-file
reads that targeted `search_graph` calls would have avoided.

**Gap 2 — Missing dual-write rule (P9).** The kanbanzai binary embeds the
`.agents/skills/kanbanzai-*/SKILL.md` files under `internal/kbzinit/skills/` for
distribution to newly-initialised projects via `kanbanzai init`. When a skill update is
made to the live `.agents/skills/` tree, the corresponding embedded copy in
`internal/kbzinit/skills/` must be updated in the same commit — otherwise new projects
receive outdated skill instructions. This requirement is not documented anywhere. The P23
retrospective identified a missed dual-write as contributing friction, but no rule was ever
added.

**Gap 3 — `entity` tool `parent` field ambiguity (P11).** The `entity` tool's `parent`
parameter is described as `"Parent ID filter: plan ID for features, feature ID for tasks
(list only)"`. The `(list only)` qualification is incorrect: `parent` is also the required
field for associating a feature with its plan at creation time. This ambiguity caused at
least one failed batch create where an agent omitted `parent` because the description
implied it was a filter-only parameter. The `parent` vs `parent_feature` distinction
(features use `parent`; tasks use `parent_feature`) is nowhere made explicit in the
description.

## Goals and Non-Goals

**Goals:**

- Add an explicit "always use `handoff`" rule and a corresponding anti-pattern to
  `orchestrate-development/SKILL.md` so the requirement has force, not just a mention in a
  procedure step.
- Add a dual-write checklist item to `AGENTS.md` so agents know to update
  `internal/kbzinit/skills/` when changing `.agents/skills/kanbanzai-*/SKILL.md`.
- Fix the `entity` tool `parent` field description to correctly reflect its dual use
  (feature creation + list filter) and make the `parent` vs `parent_feature` distinction
  explicit.

**Non-Goals:**

- No Go code changes beyond the `entity_tool.go` string literal fix for P11 (a tool
  description update is treated as documentation for design purposes).
- Making `handoff` mandatory at the MCP server level (requires a code change to the
  entity/task lifecycle; out of scope for P25).
- Auto-syncing `internal/kbzinit/skills/` from `.agents/skills/` on build (a `go generate`
  change; out of scope for P25).
- Auto-detecting dual-write targets at dev time (a future tool feature); the rule is a
  human checklist item for now.

## Design

### P8 — Add "always use `handoff`" rule to `orchestrate-development/SKILL.md`

**File:** `.kbz/skills/orchestrate-development/SKILL.md`

**Current state.** Phase 3 (Dispatch Sub-Agents), step 1 reads: "For each task in the
dispatch batch, generate a sub-agent prompt using `handoff(task_id: "TASK-xxx")`." This is
a procedure step, not a rule. The `## Anti-Patterns` section has seven entries; none
addresses manual prompt composition. Nothing in the file explains what `handoff` includes
that a manual prompt would not.

**Change 1 — New anti-pattern entry** in `## Anti-Patterns`:

> **Manual Prompt Composition**
> - **Detect:** The orchestrator writes a sub-agent prompt by hand rather than calling
>   `handoff(task_id: "TASK-xxx")`.
> - **BECAUSE:** Manual prompts omit the graph project name (so codebase-memory-mcp tools
>   are unavailable to the sub-agent), omit knowledge entries relevant to the task, and
>   omit structured spec sections assembled by the context pipeline. This was the root
>   cause of zero graph tool usage in the P24 pipeline.
> - **Resolve:** Always call `handoff(task_id)` to generate sub-agent prompts. The handoff
>   tool assembles spec sections, knowledge constraints, file paths, role conventions, tool
>   hints, and the graph project name. Supplement the generated prompt with file scope
>   boundaries and parallel ownership constraints, but never replace it.

**Change 2 — Reinforce in Phase 3.** Prepend a bold mandate to Phase 3 before the existing
numbered steps:

> **Rule:** Always use `handoff(task_id: "TASK-xxx")` to generate sub-agent prompts. Never
> compose implementation prompts manually — manual composition silently omits graph project
> context, knowledge entries, and spec sections.

The existing Phase 3 step 0 instruction (verify `codebase_memory.graph_project` in
`local.yaml`) is correct and requires no change.

---

### P9 — Add dual-write requirement to `AGENTS.md`

**File:** `AGENTS.md` (kanbanzai project root)

**Current state.** `AGENTS.md` has a "Delegating to Sub-Agents" section that reads: "See
`refs/sub-agents.md` for the required context template and propagation rule." There is no
mention of the `internal/kbzinit/skills/` dual-write requirement.

**Background on the embedding relationship.** `internal/kbzinit/skills.go` uses
`//go:embed skills` to embed the contents of `internal/kbzinit/skills/` into the binary.
`installSkills()` installs these embedded files into new projects as
`.agents/skills/kanbanzai-<name>/SKILL.md` during `kanbanzai init`. The correspondence is
direct and 1:1:

```
internal/kbzinit/skills/<name>/SKILL.md
    ↔ .agents/skills/kanbanzai-<name>/SKILL.md  (live, in this repo)
    → .agents/skills/kanbanzai-<name>/SKILL.md  (installed in new projects)
```

Currently embedded: `agents`, `design`, `documents`, `getting-started`, `plan-review`,
`planning`, `review`, `specification`, `workflow`.

There is no auto-sync step; both files must be updated manually in the same commit. Task-
execution skills under `.kbz/skills/` are project-local and are **not** embedded; no dual-
write applies to them.

**Change — New subsection** in the `## Git Discipline` section of `AGENTS.md`:

> **Dual-write rule for skill changes.** The kanbanzai binary embeds
> `.agents/skills/kanbanzai-*/SKILL.md` files under `internal/kbzinit/skills/` for
> distribution to newly-initialised projects. When you modify any file under
> `.agents/skills/kanbanzai-*/`, check whether a corresponding file exists under
> `internal/kbzinit/skills/`. If one exists, apply the same change in the same commit.
> Task-execution skills under `.kbz/skills/` are project-local; no dual-write applies.

---

### P11 — Fix `entity` tool `parent` field description

**File:** `internal/mcp/entity_tool.go`

**Current state:**

```go
mcp.WithString("parent", mcp.Description(
    "Parent ID filter: plan ID for features, feature ID for tasks (list only)"))
```

The `(list only)` suffix is wrong. `parent` is required to associate a feature with its
plan when calling `entity(action: "create", type: "feature")`. An agent reading the current
description has no way to know this; the natural reading is "only pass this on list calls."

The `parent_feature` field used for task creation is not cross-referenced, so the
distinction between the two fields is opaque.

**Change — Update the description string:**

```go
mcp.WithString("parent", mcp.Description(
    "Parent plan ID for features (required on feature create to associate the feature "+
    "with its plan; also used as a filter on list). "+
    "Note: tasks use parent_feature, not parent."))
```

This is a string-literal update with no functional logic change. A rebuild and restart of
the MCP server is required before the updated description is visible to clients.

## Alternatives Considered

### 1. Make `handoff` mandatory at the MCP server level

Add a server-side enforcement mechanism: before a task can be dispatched, the orchestrator
must call `handoff` to claim a prompt token. Sub-agent dispatch without that token would
fail at the API boundary.

**Trade-offs:**
- Eliminates reliance on documentation compliance; the rule becomes impossible to violate.
- Requires non-trivial code changes to the MCP server and task lifecycle.
- Adds orchestration friction for legitimate lightweight dispatch patterns (e.g. simple
  documentation tasks where a full handoff is excessive).
- Scope creep for P25, which is documentation-only.

**Rejected.** Out of scope. Document the requirement now; consider mechanical enforcement
in a future sprint once the rule is well-established and edge cases are understood.

### 2. Auto-sync `internal/kbzinit/skills/` from `.agents/skills/` on build

Add a `go generate` target that copies `.agents/skills/kanbanzai-*/SKILL.md` files into
`internal/kbzinit/skills/*/SKILL.md` automatically. The embedded files would always track
the live files; no human could forget to dual-write.

**Trade-offs:**
- Eliminates the dual-write burden entirely.
- Requires `go generate` integration and build-process changes.
- The generated files would need "do not edit" markers, changing the current editing
  workflow for contributors who work directly in `internal/kbzinit/skills/`.
- Out of scope for P25.

**Rejected.** Out of scope. A documentation rule is near-zero cost and ships immediately;
the auto-sync can be added later once the boundary is better understood.

### 3. Status quo — make no changes

Leave all three gaps as-is. Rely on agents rediscovering the rules through retro signals or
accumulated context.

**Trade-offs:**
- Zero effort.
- Orchestrators continue to compose prompts manually and miss the graph project name.
- Skill updates continue to be silently missed in `internal/kbzinit/skills/`.
- Agents continue to create orphaned features by omitting `parent` on create calls.

**Rejected.** The P23 and P24 retros both surfaced these as recurring, compounding friction
points. The cost to fix is low (text edits); the cost of not fixing is measurable lost
efficiency each sprint.

## Dependencies

**P8 — no blocking dependencies.** The `orchestrate-development/SKILL.md` change can be
applied immediately. It is a local task-execution skill (`.kbz/skills/`) with no embedded
copy in `internal/kbzinit/skills/`; no dual-write is required for this change itself.

**P8 and FEAT-01KPQ08Y47522 (write_file tool) are complementary.** FEAT-01KPQ08Y47522
adds reliable file-writing capability to worktree sub-agents. Once `handoff` is always used
(P8), graph tools are always available to sub-agents because the graph project name is
always present in the prompt. Once sub-agents can also write files in their worktrees, they
can combine graph-navigation and file-editing within a single agent session — the two
improvements reinforce each other. Neither blocks the other; they can be delivered
independently.

**P9 — no blocking dependencies.** The `AGENTS.md` addition is standalone. When the change
is made, apply it to the kanbanzai project's `AGENTS.md` only (not `internal/kbzinit/`,
since `AGENTS.md` is project-specific and not a generated output of `kbzinit`).

**P11 — requires a Go rebuild.** The `entity_tool.go` string literal update is a code
change. After merging, the MCP server binary must be rebuilt (`go install ./cmd/kanbanzai/`)
and the server process restarted before the updated field description is visible to MCP
clients. This is standard for any server-side tool description update.