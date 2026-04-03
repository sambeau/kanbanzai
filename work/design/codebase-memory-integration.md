# Codebase Memory — Deep Integration Design

> Design draft — no feature entity yet
> Origin: discovered during review of FEAT-01KN83QN0VAFG (lifecycle-integrity-proactive-status)
> Status: draft for discussion

---

## Overview

`codebase_memory_mcp` is a graph-based code intelligence server that indexes a
codebase into a queryable knowledge graph (nodes = symbols/files, edges = call
relationships, imports, references). It is already available as an MCP tool in
Kanbanzai sessions, but it is used inconsistently and often not at all.

This design proposes making `codebase_memory_mcp` a first-class part of the
Kanbanzai workflow: automatically indexing each feature's worktree at creation
time, recording the project name on the worktree entity, surfacing it through
`handoff` and `next`, and cleaning it up alongside the worktree. The goal is to
make the graph tools the *default* navigation strategy for agents — not a
manual opt-in that gets skipped under time pressure.

---

## Motivation

### What we observed

During the review of FEAT-01KN83QN0VAFG, the reviewer (an agent) read through
nine changed files using `read_file` and `grep` exclusively. Graph tools were
available but were never used. Post-session analysis identified two reasons:

1. **The feature branch code was not indexed.** The main project index
   (`Users-samphillips-Dev-kanbanzai`) tracks the `main` branch. New files
   added by the feature branch — `entity_children.go`, the gate sections in
   `entity_tool.go`, the status tool changes — did not exist in the index.
   Searching for `MaybeAutoAdvanceFeature` or `CountNonTerminalTasks` returned
   only YAML task records and Markdown documents, not Go source. The agent
   correctly inferred the tools would not help and fell back to `read_file`.

2. **No prompt or convention directed the agent to index first.** The
   `.github/copilot-instructions.md` says "use graph tools over grep for
   structural code questions" and references the graph skill files, but there
   is no workflow gate that ensures an index is current before an agent begins
   structural navigation. The instruction is advisory; it competes with the
   immediate availability of `grep`.

### The token efficiency case

The `codebase_memory_mcp` server was benchmarked on real codebases:

| Metric | Value |
|--------|-------|
| Linux kernel full index (28M LOC, 75K files) | 3 min |
| Linux kernel fast index | 1m 12s |
| Django full index | ~6s |
| Cypher query | <1ms |
| Trace call path (depth=5) | <10ms |
| Dead code detection (full graph scan) | ~150ms |
| **Token usage: 5 structural queries via graph** | **~3,400 tokens** |
| **Token usage: same queries via file-by-file grep** | **~412,000 tokens** |
| **Reduction** | **99.2%** |

The 99.2% token reduction is not a convenience improvement — it is an
architectural one. Context window budget spent on file navigation is budget
unavailable for reasoning. A review or implementation session that spends
100,000 tokens on `read_file` calls produces lower-quality work than one that
spends 3,000 tokens on graph queries and reserves the rest for analysis.

For Kanbanzai specifically (381 Go files at time of writing):

| Index mode | Time | Nodes | Edges |
|------------|------|-------|-------|
| `full` | ~18s | ~26,400 | ~14,650 |
| `fast` | ~8s | ~26,000 | ~14,650 |

Eight seconds to unlock the full graph for a session is not a cost — it is
an investment with a return measurable in tens of thousands of tokens per agent
dispatch.

### The multi-project architecture already exists

`codebase_memory_mcp` supports multiple simultaneously indexed projects. Every
query tool (`search_graph`, `query_graph`, `trace_call_path`, `get_code_snippet`,
`codebase_memory_mcp_search_code`, etc.) accepts an optional `project` parameter.
`list_projects` shows all indexed projects. `delete_project` removes one without
affecting others.

This means a worktree for `FEAT-X` can be indexed as a named project — e.g.
`kanbanzai-feat-x` — while the main branch index remains intact and queryable
in parallel. No coordination problem exists.

### Worktree location does not affect index quality

An experiment during the FEAT-01KN83QN0VAFG review tested whether placing a
worktree inside vs outside the main repo directory affected the index:

| Location | Nodes | Edges |
|----------|-------|-------|
| Inside repo (`.worktrees/FEAT-.../`) | 27,642 | 59,975 |
| Outside repo (`~/worktrees/kanbanzai-feat-test/`) | 27,639 | 59,971 |

The difference is noise (3 nodes, 4 edges). The inflated edge count (~60K vs
~15K for main) is intrinsic to the feature branch code graph — the new code
introduces denser call connectivity — not an artefact of the directory layout.
Separately, it was confirmed that the main index does *not* traverse into
`.worktrees/` subdirectories; the main and worktree indexes are independent.

---

## Goals

- Make graph-based code navigation the default for all agents working on a
  feature branch, not an opt-in that competes with grep
- Ensure the indexed project always reflects the current feature branch state,
  not the main branch
- Add zero friction for agents: the project name should appear in context
  automatically, not require a lookup
- Keep index lifecycle coupled to worktree lifecycle: create together,
  delete together
- Preserve the main branch index; never overwrite it
- Support concurrent features (multiple active worktrees) without conflict

## Non-Goals

- Re-indexing automatically on every commit (polling/watching is out of scope)
- Indexing worktrees for features that do not have a worktree (not all features
  require one)
- Changing `codebase_memory_mcp` itself (all changes are in Kanbanzai)
- Enforcing graph tool usage (agents are guided, not hard-blocked)
- Indexing non-Go assets (Markdown, YAML) differently from current behaviour

---

## Design

### Overview of changes

Four Kanbanzai components are touched:

1. **`worktree(action: create)`** — trigger a `fast` index after checkout,
   store the project name on the worktree record
2. **`worktree(action: remove)` / `cleanup`** — delete the project from
   `codebase_memory_mcp` as part of worktree teardown
3. **`handoff` / `next`** — include the indexed project name in generated
   agent prompts and context output
4. **Stage bindings / skill files** — document the convention so agents
   know to pass `project:` to graph tools

### 1. Worktree creation

When `worktree(action: create)` completes checkout, it runs:

```
index_repository(repo_path: <worktree_absolute_path>, mode: "fast")
```

The returned project name (derived from the path, e.g.
`Users-samphillips-Dev-kanbanzai-.worktrees-FEAT-01KN83QN0VAFG-...`) is
stored on the worktree record as a new field: `graph_project`.

The index runs synchronously before `worktree(action: create)` returns.
Eight seconds is acceptable latency for a worktree creation call. If indexing
fails (server unavailable, etc.), the worktree is still created and a warning
is returned — the graph is an enhancement, not a hard dependency.

**Project naming:** The auto-generated name is long and contains the full
path. A shorter canonical name should be derived at creation time:
`kanbanzai-<entity_id_short>` — e.g. `kanbanzai-FEAT-01KN8-3QN0VAFG`.
This name is what gets stored on the worktree record and passed to agents.

### 2. Worktree removal and cleanup

`worktree(action: remove)` reads `graph_project` from the worktree record
and calls `delete_project(project_name: <graph_project>)` before removing
the worktree directory.

`cleanup(action: execute)` does the same as part of its existing teardown
sequence — after removing the worktree directory, it deletes the associated
graph project if one is recorded.

If `graph_project` is absent (worktree was created before this feature was
introduced), removal proceeds without attempting a delete — no error.

### 3. Handoff and next

`handoff(task_id)` generates the sub-agent prompt. It currently assembles
spec sections, knowledge entries, file paths, and role conventions. It should
additionally include:

```
## Code graph

The feature branch is indexed in codebase_memory_mcp as project:
`kanbanzai-FEAT-01KN8-3QN0VAFG`

Pass this as the `project` argument to all graph tools:
- search_graph(project: "kanbanzai-FEAT-01KN8-3QN0VAFG", ...)
- trace_call_path(project: "kanbanzai-FEAT-01KN8-3QN0VAFG", ...)
- query_graph(project: "kanbanzai-FEAT-01KN8-3QN0VAFG", ...)
- codebase_memory_mcp_search_code(project: "kanbanzai-FEAT-01KN8-3QN0VAFG", ...)

Use graph tools in preference to grep or read_file for structural
questions (callers, callees, symbol definitions, impact analysis).
```

`next(id)` returns machine-readable context. A `graph_project` field should
be added to the structured output alongside the existing `worktree` field.

### 4. Skill and stage binding updates

`.kbz/stage-bindings.yaml` and the relevant skill files should be updated to
make graph tool usage explicit at the stages that benefit most:

| Stage | Graph tool benefit |
|-------|--------------------|
| `developing` | Find callers before editing a function; check impact before adding a parameter |
| `reviewing` | Verify all callers of a changed function; trace side-effect chains; find dead code introduced |
| `researching` | Explore architecture without reading hundreds of files |

The `implement-task` skill should include a pre-implementation checklist item:
"Verify `graph_project` is set in context; use `search_graph` or
`trace_call_path` before opening files for structural questions."

The `review-code` skill should include: "Run `trace_call_path` on any
function whose signature changed to verify all callers are updated."

---

## Index freshness

A known limitation: the index is a snapshot taken at worktree creation. As
implementation progresses and new files are added, the index becomes stale.

Three approaches:

**A. Re-index on task claim (`next(id)`)** — every time a sub-agent claims
a task, the worktree is re-indexed (`fast` mode, 8s). This ensures the index
reflects all commits since the last claim. Overhead is low; latency is
acceptable at task boundaries.

**B. Re-index on `finish`** — after a task completes and state is committed,
re-index so the next task's agent starts with a fresh snapshot. Adds 8s to
the `finish` call. The index then reflects work done, not work about to be done.

**C. Manual re-index only** — agents can call `index_repository` themselves
if they notice the index is stale (e.g. a function they just wrote is not
found by `search_graph`). No automatic re-indexing.

Recommendation: **option A** (re-index on task claim) for the `fast` mode.
It is transparent to the agent and ensures the graph is always current at the
start of a task. The 8-second cost is invisible at the boundary between
`next` returning and the agent beginning work.

An alternative worth considering: `fast` mode re-index on `next`, `full`
mode re-index on the transition to `reviewing` (when a reviewer needs the
most complete picture). This matches the cost to the stakes: 8s for
implementation tasks, 18s once for the review.

---

## Making it a prerequisite

The design above treats the graph as an enhancement — if indexing fails,
work continues. A stronger position: make an up-to-date index a *stage gate
prerequisite* for entering `developing`.

Arguments for:
- Removes the opt-in problem entirely
- Ensures the 99.2% token saving is realised on every feature, not just
  features where the agent happens to remember
- 8 seconds is genuinely negligible against the cost of a feature

Arguments against:
- Creates a hard dependency on `codebase_memory_mcp` being running and
  reachable; infrastructure failures block workflow advancement
- Some features are trivial (single file, well-known location) and the agent
  would never need the graph anyway
- Gate prerequisites should reflect *quality* requirements, not tooling
  availability

Recommended middle position: make the graph project a **soft prerequisite**
surfaced as an attention item. If `worktree(action: create)` was called but
`graph_project` is not set (index failed or was never attempted), the feature
detail `status` response emits an attention item of type `missing_graph_index`
with severity `warning`. This is visible to the orchestrator and sub-agents
without hard-blocking the workflow.

---

## Open questions

**Q1. Project naming collisions**
Two features with the same short ID prefix could produce the same project
name. The `entity_id_short` should use enough characters to be unique, or
the full display ID should be used (`FEAT-01KN8-3QN0VAFG` is 18 characters —
acceptable).

**Q2. Re-indexing concurrency**
If two tasks on the same feature are active simultaneously (unusual but
possible), both could trigger a re-index on `next`. The `index_repository`
call is idempotent (it overwrites), so concurrent calls produce a correct
result but waste time. A lightweight lock on the `graph_project` name would
prevent double-indexing.

**Q3. Review-stage project**
During `reviewing`, the reviewer agent may be a fresh session with no
worktree context. `handoff` for review tasks should still include the
`graph_project` name. This requires the worktree record to persist through
the reviewing stage — which it currently does (worktrees are only removed
explicitly).

**Q4. Indexing on `main` after merge**
After a feature merges, the main branch index becomes stale (it reflects
whatever was on main before the merge). The first `status` call after a merge
could trigger a re-index of main. Alternatively, `merge(action: execute)`
could trigger it. A stale main index is less harmful than a stale feature
index (main is stable; features are where new code lives), so this can be
deferred.

**Q5. Agents without worktrees**
Some tasks (documentation, research, plan-level work) never create a
worktree. These agents would fall back to the main branch index, which is
correct behaviour — they are not working on new feature code.

**Q6. `fast` vs `full` mode trade-off**
`fast` mode (8s) produces the same edges as `full` (18s) but ~2% fewer
nodes. The missing nodes are likely unresolved type aliases and interface
implementations. For implementation tasks this is fine. For review tasks,
`full` mode may be preferred to catch interface-level issues. This could be
parameterised in the stage binding.

---

## Summary of proposed changes

| Component | Change |
|-----------|--------|
| `worktree` tool | Run `fast` index on create; store `graph_project`; delete project on remove |
| `cleanup` tool | Delete `graph_project` entry during worktree teardown |
| `next` tool | Re-index (`fast`) on task claim; include `graph_project` in output |
| `handoff` tool | Include `graph_project` and graph tool usage block in generated prompt |
| `status` tool | Emit `missing_graph_index` attention item when worktree exists but index is absent |
| `implement-task` skill | Add graph pre-flight checklist item |
| `review-code` skill | Add graph-based caller verification step |
| `stage-bindings.yaml` | Document graph tool usage at `developing` and `reviewing` stages |
| `kanbanzai-agents` skill | Document `graph_project` as a standard context field |