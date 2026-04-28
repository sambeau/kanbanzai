# Codebase Memory Integration Design

> Design draft — no feature entity yet
> Origin: discovered during review of FEAT-01KN83QN0VAFG (lifecycle-integrity-proactive-status)
> Status: revised draft incorporating research findings and phased recommendation
> Last updated: 2025-07-14

---

## Overview

`codebase_memory_mcp` is a graph-based code intelligence server that indexes a
codebase into a queryable knowledge graph (nodes = symbols/files, edges = call
relationships, imports, references). It is already available as an MCP tool in
Kanbanzai sessions, but it is used inconsistently and often not at all.

This design proposes making `codebase_memory_mcp` a well-integrated part of the
Kanbanzai workflow through a **phased approach**: first building a portable
role-tool-hints mechanism (complementary design: `role-tool-hints.md`), then
adding worktree-aware graph context propagation through `handoff` and `next`,
and deferring direct process-level integration until the dependency matures and
usage data justifies it.

The goal is to make graph tools the *default* navigation strategy for agents
working on feature branches — not a manual opt-in that gets skipped under time
pressure — without creating a hard dependency on an external pre-1.0 tool.

### Key design principle: prompt-layer before process-layer

Kanbanzai is an MCP server. `codebase_memory_mcp` is an MCP server. MCP
servers do not call each other — there is no standard mechanism for one MCP
server to invoke tools on another. To have Kanbanzai's Go code trigger
`index_repository` directly, it would need to either shell out to the
`codebase_memory_mcp` CLI binary (fragile — binary path discovery, version
skew) or implement an MCP client (significant complexity — JSON-RPC over
stdio, process lifecycle management).

Neither is justified today. The agents already have access to both tool sets.
The integration problem is not *capability* but *context*: agents don't know
which project to query, or that they should index the worktree first. This is
a prompt engineering problem, and the solution belongs in the prompt layer.

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

### Fast vs full indexing at Kanbanzai's scale

The `fast` vs `full` distinction matters at Linux kernel scale (28M LOC) where
fast mode drops 10.5% of nodes and skips call resolution, semantic enrichment,
test linking, and git history. **At Kanbanzai's scale (~381 Go files), the
difference is negligible.** Both modes produce essentially the same call graph.
The ~2% node difference represents unresolved type aliases and interface
implementations — relevant for exhaustive review but not for navigation or
impact analysis.

At this codebase size, always running `full` mode (18 seconds) is a reasonable
default. The 10-second difference is invisible at a task boundary. If and when
Kanbanzai grows significantly, the fast/full distinction becomes worth
parameterising.

### The multi-project architecture already exists

`codebase_memory_mcp` supports multiple simultaneously indexed projects. Every
query tool (`search_graph`, `query_graph`, `trace_call_path`, `get_code_snippet`,
`codebase_memory_mcp_search_code`, etc.) accepts an optional `project` parameter.
`list_projects` shows all indexed projects. `delete_project` removes one without
affecting others.

This means a worktree for `FEAT-X` can be indexed as a named project — e.g.
`kanbanzai-FEAT-01KN8-3QN0VAFG` — while the main branch index remains intact
and queryable in parallel. No coordination problem exists.

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

## Dependencies

`codebase_memory_mcp` is an impressive piece of engineering — a single static C
binary with zero runtime dependencies, 66 languages, 14 MCP tools. However, the
integration design must account for its maturity profile:

| Factor | Assessment |
|--------|------------|
| Version | Pre-1.0 (v0.5.7) — breaking changes expected |
| Contributors | ~3 (essentially single-developer) — bus factor risk |
| Stars | ~1,200 — meaningful adoption but not yet established |
| Known stability issues | Memory corruption on large Java projects (#189), SQLite writer crashes (#187), high CPU/stuck indexing (#195) |
| Go support quality | Excellent — one of the best-supported languages, 100% benchmark score |
| Cypher subset | Limited — no `WITH`, `COLLECT`, `OPTIONAL MATCH`, 200-row cap on `query_graph` |

**Conclusion:** The tool is solid for Go codebases and the query surface is
sufficient for Kanbanzai's needs. The stability concerns affect large polyglot
projects more than a focused Go codebase. However, creating a *hard dependency*
on a pre-1.0 single-developer project would be premature. The integration
should be purely additive — failures must never block workflow.

---

## Goals and Non-Goals

### Goals

- Make graph-based code navigation the default for all agents working on a
  feature branch, not an opt-in that competes with grep
- Ensure agents know which project to query (per-worktree project name in
  context) and are instructed to index the worktree before structural queries
- Add zero friction for agents: the project name should appear in context
  automatically, not require a lookup
- Keep index lifecycle conceptually coupled to worktree lifecycle: create
  together, clean up together
- Preserve the main branch index; never overwrite it
- Support concurrent features (multiple active worktrees) without conflict
- **No hard dependency**: if `codebase_memory_mcp` is unavailable, the
  workflow operates identically to today — no degradation

### Non-Goals

- Kanbanzai Go code calling `codebase_memory_mcp` directly (deferred; see
  Phase 3 below)
- Re-indexing automatically on every commit (polling/watching is out of scope)
- Indexing worktrees for features that do not have a worktree
- Changing `codebase_memory_mcp` itself (all changes are in Kanbanzai)
- Enforcing graph tool usage (agents are guided, not hard-blocked)
- Indexing non-Go assets differently from current behaviour
- Making `codebase_memory_mcp` a stage gate prerequisite

---

## Design

### Phase 1: Role-Tool-Hints (portable foundation)

> See `work/design/role-tool-hints.md` for full design.

Add a `tool_hints` config mechanism to `local.yaml` and `config.yaml`. Hints
are role-scoped, opaque strings injected into sub-agent prompts by `handoff`
and into context by `next`. This solves the general problem of agents not
knowing about optional tools — not just `codebase_memory_mcp` but any MCP
server available on the machine.

Example `local.yaml`:

```yaml
tool_hints:
  implementer-go: |
    Use search_graph and trace_call_path for structural code questions
    (callers, callees, symbol definitions, impact analysis). Fall back to
    grep only for string literals and error messages. Read
    .github/skills/codebase-memory-exploring/SKILL.md before your first
    graph query.
  reviewer: |
    Before starting your review, read and follow
    .github/skills/codebase-memory-tracing/SKILL.md. Use trace_call_path
    on any function whose signature changed to verify all callers are updated.
```

**Why this comes first:** It is independently valuable, creates no external
dependencies, and covers the general case. If `codebase_memory_mcp` were to
disappear tomorrow, tool hints would still improve agent behaviour with
whatever tools are available.

**Estimated cost:** ~50–80 lines of Go across config, handoff, and next.

### Phase 2: Worktree-Aware Graph Context

Add graph project awareness to the worktree system and flow it through to
agents. This is the core value: solving the "agents don't know which project
to query" problem that caused the FEAT-01KN83QN0VAFG review failure.

#### 2.1 GraphProject field on worktree records

Add a `GraphProject string` field to the `worktree.Record` struct:

```go
type Record struct {
    // ... existing fields ...
    GraphProject string     // codebase_memory_mcp project name (optional)
}
```

The field is optional. Worktrees created before this feature have an empty
`GraphProject` — all existing behaviour is preserved.

The project name follows the convention `kanbanzai-<entity-id>`, e.g.
`kanbanzai-FEAT-01KN8-3QN0VAFG`. This is short enough to be readable in
prompts and unique enough to avoid collisions (entity IDs are ULID-based).

**How the field gets populated:** The orchestrating agent (or the first
sub-agent dispatched to the worktree) runs:

```
index_repository(repo_path: <worktree_absolute_path>, mode: "full")
```

and then updates the worktree record's `GraphProject` field. This happens
in the agent layer, not in Kanbanzai's Go code — no MCP-to-MCP calls needed.

To support this, `worktree(action: update)` (or a new action) should accept a
`graph_project` parameter to set the field on an existing worktree record.

#### 2.2 Handoff and next emit graph context

When `handoff(task_id)` generates a sub-agent prompt and the worktree record
has a non-empty `GraphProject`, it includes a section:

```markdown
## Code Graph

The feature branch is indexed in codebase_memory_mcp as project:
`kanbanzai-FEAT-01KN8-3QN0VAFG`

Pass this as the `project` argument to all graph tools:
- search_graph(project: "kanbanzai-FEAT-01KN8-3QN0VAFG", ...)
- trace_call_path(project: "kanbanzai-FEAT-01KN8-3QN0VAFG", ...)
- query_graph(project: "kanbanzai-FEAT-01KN8-3QN0VAFG", ...)
- get_code_snippet(project: "kanbanzai-FEAT-01KN8-3QN0VAFG", ...)

Use graph tools in preference to grep or read_file for structural
questions (callers, callees, symbol definitions, impact analysis).

If you create new files or significantly restructure code, re-index
by running: index_repository(repo_path: "<worktree_path>", mode: "full")
```

When `GraphProject` is empty but a worktree exists, the section instead reads:

```markdown
## Code Graph

This feature's worktree has not been indexed yet. Before starting
structural code exploration, index it:

  index_repository(repo_path: "<worktree_path>", mode: "full")

This takes ~18 seconds and unlocks graph-based navigation for the session.
After indexing, use search_graph, trace_call_path, and query_graph in
preference to grep or read_file for structural questions.
```

When no worktree exists (documentation, research, planning tasks), the
section is omitted entirely. These agents fall back to the main branch index
naturally.

`next(id)` returns machine-readable context. A `graph_project` field is added
to the structured output alongside the existing `worktree` field:

```json
{
  "worktree": { "path": ".worktrees/FEAT-.../", "branch": "feat/..." },
  "graph_project": "kanbanzai-FEAT-01KN8-3QN0VAFG"
}
```

#### 2.3 Status attention item

If a worktree exists for a feature but `GraphProject` is empty (index was
never run or failed), the `status` tool emits an attention item:

```
type: missing_graph_index
severity: info
message: "Worktree exists but codebase graph is not indexed. Agents will
  fall back to grep. Run index_repository on the worktree path to enable
  graph-based navigation."
```

Severity is `info`, not `warning` — this is an optimisation opportunity, not
an error condition. It is visible to orchestrators and sub-agents without
hard-blocking the workflow.

#### 2.4 Cleanup

When `worktree(action: remove)` or `cleanup(action: execute)` removes a
worktree whose record has a non-empty `GraphProject`, the cleanup summary
should note that the agent (or user) should also delete the graph project:

```
Note: worktree had graph project "kanbanzai-FEAT-01KN8-3QN0VAFG" indexed
in codebase_memory_mcp. Run delete_project(project_name:
"kanbanzai-FEAT-01KN8-3QN0VAFG") to free the index.
```

This keeps cleanup in the agent/user layer, consistent with the prompt-layer
approach.

#### 2.5 Skill and stage binding updates

Skill files and stage bindings should be updated to reference the graph
context when it is available:

| Stage | Skill file | Addition |
|-------|-----------|----------|
| `developing` | `implement-task` | Pre-implementation checklist: "Check if `graph_project` is in context. If so, use `search_graph` or `trace_call_path` before opening files for structural questions. If not indexed, run `index_repository` on the worktree path." |
| `reviewing` | `review-code` | "Run `trace_call_path` on any function whose signature changed to verify all callers are updated. Use `detect_changes` for blast radius assessment." |
| `researching` | `write-research` | "Use `get_architecture` and `search_graph` for codebase orientation instead of reading files." |

**Estimated cost for Phase 2:** ~80–120 lines of Go (new field on Record,
handoff/next emission, status attention item, cleanup note). Plus skill/binding
documentation updates.

### Phase 3: Direct Process Integration (deferred)

> **Not recommended for initial implementation.** Revisit after Phase 1–2 are
> deployed and usage data is available.

If Phase 2 proves valuable and `codebase_memory_mcp` matures (approaches
1.0, stability issues resolved, contributor base grows), a deeper integration
becomes worth considering:

- **Kanbanzai triggers `index_repository` on worktree create** — either by
  shelling out to the `codebase_memory_mcp` CLI binary or by implementing a
  lightweight MCP client. Removes the manual indexing step from the agent.
- **Automatic re-indexing on task claim** (`next(id)`) — ensures the graph is
  always current at the start of a task. ~18s cost at task boundaries.
- **Automatic project deletion on cleanup** — Kanbanzai calls `delete_project`
  directly instead of emitting a note.
- **Stage-aware index mode** — `full` mode re-index on transition to
  `reviewing` for maximum graph completeness.

The prerequisites for Phase 3:

1. `codebase_memory_mcp` reaches 1.0 or demonstrates sustained stability
2. Phase 2 usage data confirms agents reliably use graph tools when instructed
3. The manual indexing step in Phase 2 proves to be a friction point worth
   eliminating
4. A clean mechanism for MCP-to-MCP calls exists, or the CLI binary interface
   is stable enough to shell out to

---

## Index Freshness

With the Phase 2 approach, index freshness is managed by agents:

- **Initial index:** The first agent dispatched to a worktree runs
  `index_repository` (prompted by handoff).
- **Stale index:** If an agent creates new files and then can't find them
  via `search_graph`, it re-indexes. The handoff prompt explicitly tells
  agents to do this.
- **Task boundaries:** Orchestrators can re-index between task dispatches.
  The handoff prompt for subsequent tasks on the same worktree will include
  the "re-index if you've made structural changes" instruction.

This is less automatic than the Phase 3 approach but has the advantage of
zero added complexity in Kanbanzai's Go code and zero latency impact on
`worktree(action: create)` or `next(id)`.

---

## Where the Payoff Is Greatest

The integration benefits are not uniform across workflow stages. The highest
value comes from:

### Code reviews

This is where the integration pays for itself. A reviewer agent using
`trace_call_path` on every changed function signature catches missed callers
that file-by-file review cannot. `detect_changes` provides blast radius
assessment. `get_architecture` gives structural context. These are the queries
where the 99.2% token reduction is most impactful — reviews read the *most*
code and benefit the *most* from targeted navigation.

### Impact analysis during implementation

Before editing a function, an implementer agent that runs
`trace_call_path(function_name: "MyFunc", direction: "inbound")` discovers
all callers and can assess whether a parameter change is safe. Without the
graph, this requires `grep` + `read_file` on every match — often hundreds of
tokens per caller.

### Architecture exploration during research

`get_architecture` and `search_graph` provide codebase orientation in a
single call. An agent researching "how does the config system work?" gets a
structured answer instead of navigating a file tree.

### Where it matters less

Documentation, planning, and simple single-file tasks gain little from graph
tools. These stages should not be burdened with indexing instructions.

---

## Alternatives Considered

### Alternative: Hardcode graph tool instructions in skill files

Rejected. Breaks portability — users without `codebase_memory_mcp` receive
instructions for tools that don't exist. Skill files are committed project
assets and should not encode machine-specific tool availability.

### Alternative: Require orchestrators to manually include tool context

The current approach, documented in `refs/sub-agents.md`. Rejected as the
primary solution because it is fragile — easy to forget, not enforced by the
system, and breaks down across delegation chains.

### Alternative: Deep process-level integration from the start

Rejected for Phase 1–2. Kanbanzai calling `codebase_memory_mcp` directly
requires either shelling out to the CLI binary (fragile — path discovery,
version skew) or implementing an MCP client (significant complexity). Neither
is justified while the dependency is pre-1.0 and prompt-layer integration
achieves the same outcome. Preserved as Phase 3 for future evaluation.

### Alternative: Make graph index a hard stage gate prerequisite

Rejected. Creates a hard dependency on `codebase_memory_mcp` being running
and reachable; infrastructure failures would block workflow advancement. Gate
prerequisites should reflect quality requirements, not tooling availability.
A soft `info`-level attention item achieves visibility without blocking.

---

## Relationship to role-tool-hints

The two designs are complementary, not competing:

| Concern | role-tool-hints | This design |
|---------|----------------|-------------|
| Scope | Any optional MCP tool on any machine | `codebase_memory_mcp` specifically |
| Mechanism | User-authored config strings | Structured graph project context |
| What it solves | "Agents don't know tools exist" | "Agents don't know which project to query" |
| Dependency | None | `codebase_memory_mcp` (soft, optional) |
| Persistence | Config files | Worktree records |

When both are active, the handoff prompt contains:
1. `## Available Tools` — from role-tool-hints (general guidance)
2. `## Code Graph` — from this design (specific project name and instructions)

These sections are distinct and non-overlapping. `## Code Graph` provides the
project name and per-tool examples; `## Available Tools` provides broader
guidance that may cover other tools entirely.

For users who do not have `codebase_memory_mcp`, role-tool-hints still
improves agent behaviour with whatever tools are available. For users who do
have it, this design ensures agents know the right project name without the
user needing to write it into their tool hints manually.

---

## Open Questions

**Q1. Project naming collisions**
Two features with the same short ID prefix could produce the same project
name. The `entity_id_short` should use enough characters to be unique, or
the full display ID should be used (`FEAT-01KN8-3QN0VAFG` is 18 characters —
acceptable).

**Q2. Review-stage project**
During `reviewing`, the reviewer agent may be a fresh session with no
worktree context. `handoff` for review tasks should still include the
`graph_project` name. This requires the worktree record to persist through
the reviewing stage — which it currently does (worktrees are only removed
explicitly).

**Q3. Indexing on `main` after merge**
After a feature merges, the main branch index becomes stale (it reflects
whatever was on main before the merge). A stale main index is less harmful
than a stale feature index (main is stable; features are where new code
lives), so this can be deferred. Agents or users can re-index main manually.

**Q4. Agents without worktrees**
Some tasks (documentation, research, plan-level work) never create a
worktree. These agents fall back to the main branch index, which is correct
behaviour — they are not working on new feature code. The `## Code Graph`
section is simply omitted from their handoff prompt.

**Q5. Setting GraphProject without a worktree update tool**
Phase 2 requires a way for agents to write the `GraphProject` field after
indexing. Options: extend `worktree(action: update)` to accept a
`graph_project` parameter, or add a dedicated `worktree(action: set_graph)`
action. The former is simpler and follows the existing pattern.

**Q6. What if the agent doesn't index?**
If an agent ignores the indexing instruction and proceeds with grep, the
workflow still works — just less efficiently. The `missing_graph_index`
attention item surfaces the missed opportunity on subsequent `status` calls.
This is acceptable; the design is additive, not coercive.

---

## Summary of Proposed Changes

### Phase 1 (role-tool-hints — see companion design)

| Component | Change |
|-----------|--------|
| `config.go` | Add `ToolHints map[string]string` to `Config` |
| `user.go` | Add `ToolHints map[string]string` to `LocalConfig` |
| `handoff_tool.go` / `pipeline.go` | Resolve and inject hint for active role |
| `next_tool.go` | Include resolved hint in context output |

### Phase 2 (worktree-aware graph context)

| Component | Change |
|-----------|--------|
| `worktree.go` | Add `GraphProject string` field to `Record` |
| `worktree_tool.go` | Accept `graph_project` param on update action |
| `handoff_tool.go` / `pipeline.go` | Emit `## Code Graph` section when `GraphProject` is set; emit indexing prompt when worktree exists but `GraphProject` is empty |
| `next_tool.go` | Include `graph_project` in structured context output |
| `status` tool | Emit `missing_graph_index` attention item (severity: info) |
| Cleanup tools | Note graph project for manual deletion |
| `implement-task` skill | Add graph pre-flight checklist item |
| `review-code` skill | Add graph-based caller verification step |
| `stage-bindings.yaml` | Document graph tool usage at `developing` and `reviewing` |
| `kanbanzai-agents` skill | Document `graph_project` as a standard context field |

### Phase 3 (deferred — direct process integration)

| Component | Change |
|-----------|--------|
| `worktree_tool.go` | Trigger `index_repository` on create via CLI or MCP client |
| `next_tool.go` | Re-index on task claim |
| `cleanup/execute.go` | Call `delete_project` directly |
| New: MCP client or CLI shim | Interface to `codebase_memory_mcp` binary |