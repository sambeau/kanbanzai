# Next Sprint Proposal

**Sources:**
- `work/reports/p24-implementation-retro.md` — P24 parallel implementation pipeline retrospective
- `work/reports/retro-session-2026-04-20.md` — P23 review session retrospective

**Prepared by:** Orchestrating agent  
**Date:** 2026-04-21

---

## Executive Summary

Two retrospectives surface the same friction clusters from different angles. The P23 session retro observed these problems in a session without worktrees; the P24 implementation retro observed them again, more acutely, in a full parallel worktree pipeline. The problems are real, recurring, and well-understood. This proposal groups them into four themes and recommends a concrete set of changes, ordered by impact.

The four themes are:

1. **Worktree file editing is broken** — `edit_file` doesn't work in worktrees; the `python3 -c` workaround is brittle for source files
2. **`decompose propose` is unreliable** — hardcoded one-AC-per-task rule bypassed by two of five features; empty task names crashed `apply`
3. **The merge verification gate has no agentic write path** — every agentic merge requires `override: true`, which is semantically wrong
4. **Sub-agent context is incomplete** — `handoff` not used; graph project name lost; dual-write requirement for skills not documented

There are also several small process improvements that can be done as documentation changes with no code required.

---

## Theme 1 — Worktree File Editing

### Problem

`edit_file` operates on the main working tree, not the checked-out worktree branch. Agents working inside a worktree must use `terminal` + `python3 -c` or heredocs to write files. Both workarounds have serious reliability problems with Go source files:

- `python3 -c` with triple-quoted strings breaks on files containing embedded triple-single-quotes (Go doc comments, test string literals, generated code)
- Heredocs (`cat << 'EOF'`) are more reliable for source files but fail if the content contains a line that is exactly `EOF`, and the syntax is fragile across different shell environments
- Both approaches embed the entire file content in a shell command, consuming large amounts of context window

This was observed in P24 with Go source files. The P23 session avoided the problem entirely by not using worktrees — confirming this is a worktree-isolation issue, not a general `edit_file` reliability issue.

The `implement-task/SKILL.md` currently documents `python3 -c` as the recommended pattern. It should at minimum be updated to prefer heredocs; but the real fix is a proper tool.

### Proposed changes

**P1 — Code: Make `edit_file` worktree-aware** *(high impact, medium effort)*

When `edit_file` is called with a path that falls inside a known active worktree (i.e. a path under `.worktrees/<branch>/`), it should apply the edit to that path rather than to the main working tree. This is the cleanest fix and makes the worktree experience identical to the main-tree experience from an agent's perspective. It removes the entire category of worktree-editing friction without requiring any change to agent prompts or skill documentation.

If full worktree awareness is out of scope, an acceptable alternative is:

**P1-alt — Code: Add a `write_file` MCP tool** *(high impact, low effort)*

A new tool that accepts `path` (relative to repo root or absolute) and `content` (string) as distinct JSON parameters and writes the file atomically. No shell quoting involved. The agent's model sends the content as a JSON string value; the server writes it verbatim. This eliminates all escaping problems and context-window inflation from shell-embedded file content.

Signature sketch:

```
write_file(path: string, content: string, worktree_id?: string) → {written: bool, bytes: int}
```

**P2 — Docs: Update `implement-task/SKILL.md` worktree file editing section** *(quick win, low effort)*

Until P1 or P1-alt is shipped, update the existing worktree file editing section (added in P24) to:

- Recommend heredoc (`cat > file << 'GOEOF'`) as the primary pattern for Go source files, not `python3 -c`
- Add a note that `python3 -c` remains usable for Markdown and YAML files where escaping is not an issue
- Include a known-failure note: if heredoc fails on a file containing a bare `GOEOF` line, use a different delimiter (`GOEOF2`, `ENDOFFILE`, etc.)

---

## Theme 2 — `decompose propose` Reliability

### Problem

`decompose propose` was bypassed by two of five spec agents in P24:

- **FEAT-01KPPG4SXY6T0 (hygiene docs):** Hardcoded one-AC-per-task rule produced 13 tasks from 13 acceptance criteria, ignoring the dev plan's intended 4-task file-scoped grouping. Cannot be overridden by context.
- **FEAT-01KPPG5XMJWT3 (optional GitHub PR):** Proposal returned empty task names, causing `decompose apply` to raise an error. Tasks were created manually via `entity`.

Both failures required agents to fall back to manual `entity(action: create)` calls — work the `decompose` tool is supposed to handle. The fallback is undocumented and requires agents to understand the internal task schema and dependency wiring format.

The pattern is consistent: `decompose propose` works well for standard code features with one AC per logical unit of work. It degrades for documentation-only features, configuration-only features, and any spec where the intended task grouping cuts across AC boundaries (e.g. "one task per file modified" rather than "one task per requirement").

### Proposed changes

**P3 — Code: Fix empty task names in `decompose propose` output** *(blocking bug, low effort)*

A proposal with empty task names should never be produced. If the tool cannot derive a task name from the spec, it should either use the AC identifier as a fallback name (`"Implement AC-001"`) or fail explicitly with an error rather than producing an apply-breaking proposal. This is a correctness bug.

**P4 — Code: Add dev-plan-aware grouping to `decompose propose`** *(high impact, medium effort)*

When a dev plan document is approved for the feature, `decompose propose` should read it and use the task breakdown defined there as the authoritative grouping structure, rather than applying a heuristic. The dev plan already contains task names, scopes, and dependency relationships — the tool should extract these rather than rederiving them from the spec alone.

If the dev plan is absent or doesn't define a task breakdown, fall back to the current AC-based heuristic.

Concretely: add a step in `DecomposeFeature` that checks for an approved dev plan document, loads it, and attempts to parse a task list from it before falling back to `parseSpecStructure`. The dev plan template already has a standardised "Task Breakdown" section format that is machine-parseable.

**P5 — Docs: Add a note to `decompose-feature/SKILL.md` about manual fallback** *(quick win, low effort)*

Until P4 is implemented, document the manual fallback clearly: if `decompose propose` produces a visibly wrong proposal (wrong task count, empty names, one-task-per-AC when the dev plan specifies otherwise), use `entity(action: create)` directly. Include the minimum required fields and a dependency wiring example.

---

## Theme 3 — Merge Verification Gate

### Problem

All five P24 features failed `verification_exists` and `verification_passed` at `merge(action: check)`. These gates require a `verification` field and a `verification_status` on the feature entity. There is no MCP tool call that sets these fields. They are populated only by the formal reviewing stage workflow, which is itself a mandatory gate that cannot be auto-advanced.

In practice every agentic merge in recent sessions has required `override: true` on these two gates. `override` is semantically reserved for exceptional circumstances; using it as the normal path degrades its signal value.

The lifecycle model assumes a human or reviewer-agent stage between `developing` and `done`. In agentic-only pipelines, this stage is skipped or replaced by automated test verification recorded in task `finish()` calls. The gap is that `finish()` verification data on tasks is not propagated upward to the feature entity.

### Proposed changes

**P6 — Code: Propagate task verification to feature entity** *(high impact, medium effort)*

When all tasks for a feature are transitioned to `done` via `finish()`, aggregate the `verification` fields from those `finish()` calls and write a summary to the feature entity's `verification` field. Set `verification_status` to `"passed"` if all tasks reported passing verification, `"failed"` if any reported failure.

This allows the `verification_exists` and `verification_passed` merge gates to pass automatically in agentic pipelines where tasks were finished with `go test ./... passed` verification summaries — without requiring `override`.

If any task's `finish()` had no verification field, treat it as `"unverified"` and surface a warning (non-blocking) rather than a blocking failure.

**P7 — Code: Align the reviewing stage with agentic workflows** *(medium impact, medium effort)*

The reviewing stage should be auto-advanceable when:
- All tasks are `done`, AND
- All tasks have a non-empty verification field in their `finish()` records, AND
- `feature.config.require_human_review` is absent or `false`

This mirrors the pattern just implemented for `require_github_pr`. Projects that want a human review gate set the flag; projects running agentic-only pipelines do not, and the stage auto-advances.

---

## Theme 4 — Sub-agent Context and Handoff Quality

### Problem

In the P24 implementation pipeline, sub-agent prompts were composed manually rather than using the `handoff(task_id)` tool. This caused several compounding problems:

- The `graph_project` name (`Users-samphillips-Dev-kanbanzai`) was not included in any sub-agent prompt, so no sub-agent used `codebase-memory-mcp` graph tools
- Sub-agents navigated the codebase by `cat`-ing whole files and grepping for symbols rather than using `search_graph`, `trace_call_path`, or `get_code_snippet`
- Knowledge entries relevant to the task (e.g. the worktree file editing pattern) were not surfaced to implementing agents
- One agent (FEAT-01KPPG5XMJWT3) hit context window limits partly because it read large files in full rather than using targeted graph queries

The `handoff` tool exists specifically to solve this: it assembles a structured prompt from the task record including spec sections, knowledge entries, file paths, role conventions, and the graph project name. It was not used.

Additionally, the P23 session retro identified a dual-write requirement unique to Kanbanzai's own development: changes to `.kbz/skills/` and `AGENTS.md` must also be applied to `internal/kbzinit/skills/` (the embedded skills distributed to other projects). This was missed in P23 and not documented anywhere accessible at the point of implementation.

### Proposed changes

**P8 — Docs: Add "always use `handoff`" rule to `orchestrate-development/SKILL.md`** *(quick win, low effort)*

Add an explicit rule: when dispatching implementation sub-agents, always call `handoff(task_id: "TASK-...")` to generate the prompt. Never compose implementation prompts manually. The handoff output includes graph project name, knowledge entries, and spec context that manual prompts omit.

Add a corresponding anti-pattern: "Manual prompt composition for implementation tasks — misses graph project name, knowledge entries, and structured spec context."

**P9 — Docs: Add dual-write requirement to `AGENTS.md` and `implement-task/SKILL.md`** *(quick win, low effort)*

Add a note to the "Kanbanzai-specific" section of `AGENTS.md`:

> When modifying `.kbz/skills/`, `AGENTS.md`, or any file under `.kbz/roles/`, also check whether a corresponding file exists under `internal/kbzinit/skills/`. Changes to skills that should reach other projects must be applied to both trees. Skill files in `internal/kbzinit/` are the distributed copies; `.kbz/` is the local development copy.

Add a corresponding checklist item to `implement-task/SKILL.md` for tasks involving skill or documentation changes.

**P10 — Docs: Add one-sub-agent-per-task guidance to orchestration skill** *(quick win, low effort)*

Add a sizing guideline to `orchestrate-development/SKILL.md`:

> For features with more than three tasks where tasks involve reading or rewriting large source files (> 300 lines), dispatch one sub-agent per task rather than one per feature. Full-file rewrites via terminal embed the entire file content in tool calls, consuming context rapidly. One-per-task agents start fresh and handle one file scope each.

**P11 — Docs: Fix `entity` tool description ambiguity for `parent` field** *(quick win, low effort)*

The `parent` field in `entity(action: create)` is currently described as a "list only" filter parameter. This obscures its function as the parent ID field when creating feature entities. Update the description to make clear it is required for feature creation (to associate the feature with a plan) and also usable as a filter on list. The `parent` vs `parent_feature` distinction (features use `parent`; tasks use `parent_feature`) should be explicit.

---

## Prioritised Backlog

| ID | Change | Type | Impact | Effort | Priority |
|----|--------|------|--------|--------|----------|
| P1 | Make `edit_file` worktree-aware | Code | High | Medium | 🔴 High |
| P1-alt | Add `write_file` MCP tool | Code | High | Low | 🔴 High |
| P3 | Fix empty task names in `decompose propose` | Code (bug) | High | Low | 🔴 High |
| P6 | Propagate task `finish()` verification to feature entity | Code | High | Medium | 🔴 High |
| P4 | Dev-plan-aware grouping in `decompose propose` | Code | High | Medium | 🟡 Medium |
| P7 | Auto-advance reviewing stage in agentic workflows | Code | Medium | Medium | 🟡 Medium |
| P8 | Add "always use `handoff`" rule to orchestration skill | Docs | Medium | Low | 🟡 Medium |
| P9 | Add dual-write requirement to AGENTS.md / implement-task | Docs | Medium | Low | 🟡 Medium |
| P2 | Update worktree editing section: prefer heredoc over python3 | Docs | Medium | Low | 🟡 Medium |
| P5 | Document manual `entity` fallback for decompose failures | Docs | Low | Low | 🟢 Low |
| P10 | One-sub-agent-per-task sizing guidance | Docs | Low | Low | 🟢 Low |
| P11 | Fix `entity` tool `parent` field description | Docs | Low | Low | 🟢 Low |

---

## Recommended Sprint Scope

### Must do (blocks agentic pipeline quality)

- **P1-alt** — `write_file` MCP tool. This is the single highest-leverage change. Every agentic implementation sprint uses worktrees; every sprint hits this problem. The tool is small (a new handler that accepts path + content, writes atomically, nothing else) and removes an entire category of friction.
- **P3** — Fix empty task names in `decompose propose`. This is a correctness bug; proposals that cause `apply` to crash are never acceptable output.
- **P6** — Propagate task verification to feature entity. Every agentic merge currently requires `override: true` on two gates. This is not a niche case; it is the default pipeline. Fix it so `override` means what it's supposed to mean.

### Should do (significant recurring friction)

- **P4** — Dev-plan-aware `decompose propose`. Two of five features in P24 had to bypass the tool. The dev plan already has the task breakdown; the tool should read it.
- **P8** — "Always use `handoff`" in orchestration skill. Costs nothing to document; prevents sub-agents from being dispatched without graph context.
- **P9** — Dual-write rule in AGENTS.md. Missed in P23, not documented, will be missed again.
- **P2** — Prefer heredoc over python3 in worktree skill. Immediate improvement until P1-alt ships.

### Nice to have (lower frequency)

- **P7** — Agentic reviewing stage auto-advance
- **P5** — Manual decompose fallback docs
- **P10** — One-sub-agent-per-task sizing guidance
- **P11** — `entity` tool description fix

---

## What Is Explicitly Not Proposed

- **Removing the reviewing or verification gates entirely.** The gates are correct; the problem is that the write path for agentic workflows is missing. Fix the write path, keep the gates.
- **Changing the worktree model.** Parallel feature branches with isolated worktrees work well. The problem is tooling inside them, not the model itself.
- **Changing how `finish()` works for tasks.** The task completion flow is working correctly. The gap is the aggregation step upward to the feature entity.