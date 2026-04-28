# Retrospective: P25 — Agent Tooling and Pipeline Quality

**Plan:** P25-agent-tooling-pipeline-quality
**Period:** 2026-04-21
**Scope:** Full plan delivery — 7 features, 23 tasks, merged to main
**Author:** Orchestrating agent (Claude Sonnet 4.6) + synthesised project signals

---

## Executive Summary

P25 delivered all seven planned features: the `write_file` MCP tool, two decompose reliability
fixes, task verification propagation, dev-plan-aware task grouping, agentic reviewing
auto-advance, and two sets of workflow documentation improvements. Every task reached `done` and
every branch was merged to `main` within a single session.

The implementation surface was clean — sub-agents produced correct, test-covered code on first
attempt for all three features dispatched in parallel. However, the *operational* layer of the
session had five notable failure modes: a parallel merge race condition that corrupted one
feature's audit trail, sub-agents not using the code-graph tools despite having context for them,
the `write_file` tool being unavailable to the very agents that needed it most, two rebase
conflicts from undetected shared-file overlap, and a genuine nil-pointer bug that had been silently
crashing automatic worktree creation for bug entities. All five are actionable and preventable.

---

## What Was Delivered

| Feature | Description | Addresses |
|---|---|---|
| `write_file` MCP tool | Atomic file writes scoped to entity worktrees; replaces `python3 -c` workaround | P1-alt |
| Fix empty task names in decompose | `deriveTaskName` fallback prevents apply-breaking proposals | P3 |
| Propagate task verification | `AggregateTaskVerification` on `DispatchService`; `VerificationPassedGate` partial-warning | P6 |
| Dev-plan-aware task grouping | `parseDevPlanTasks` + discovery in `DecomposeFeature`; dev plan supersedes AC heuristic | P4 |
| Agentic reviewing auto-advance | `require_human_review` config flag; eliminates override-as-normal-path | P7 |
| Implementation workflow docs | Heredoc preference, entity fallback, per-task sizing guidance in SKILL.md | P2, P5, P10 |
| Sub-agent orchestration docs | Always-use-handoff mandate, dual-write rule, entity tool parent-field clarification | P8, P9, P11 |

Additionally, a pre-existing nil pointer dereference in `ensureWorktree`
(`internal/service/status_transition_hook.go:174`) was diagnosed and fixed in the same session.

---

## Issue 1: Parallel merge race condition (Significant)

### What happened

Three features were merged in parallel: FEAT-01KPQ08Y989P8, FEAT-01KPQ08YBJ5AK, and
FEAT-01KPQ08YE4399. The `merge(action: execute)` tool performs a two-phase git operation:

1. `git merge --squash <branch>` — stages the diff but does not commit
2. A subsequent `git commit` — creates the workflow state commit

When two merge operations run concurrently, phase 1 of operation B can complete (staging its
diff) before phase 2 of operation A has committed. Operation A's commit then absorbs both its
own staged changes *and* the staged-but-uncommitted diff from operation B.

In this session, FEAT-01KPQ08YE4399's code (changes to `advance.go`, `prereq.go`,
`entity_tool.go`, and their tests) was swept into the commit labelled
`workflow(FEAT-01KPQ08YBJ5AK): mark worktree merged`. FEAT-01KPQ08YE4399's own merge then
reported `exit status 1` because the squash produced an empty delta — the content was already
on `main`.

### Consequences

- FEAT-01KPQ08YE4399's override records were not written (no `overrides` field in entity state).
- The merge commit for that feature's code is attributed to the wrong feature in `git log`.
- Manual cleanup was required: remove the orphaned worktree with `git worktree remove --force`,
  delete the branch, and patch the worktree state YAML by hand.

### Root cause

The `merge` tool's two-phase git sequence is not atomic. The git index is a shared resource
across all concurrent operations in the same repository. There is no locking between merge
operations.

### Recommendation

**Merges must be strictly serialised.** Never dispatch `merge(action: execute)` for two features
in the same parallel batch, regardless of how independent the branches appear. The `merge` tool
should ideally acquire a file lock before the squash step and release it after the workflow
commit, but until that is implemented, the orchestration discipline must compensate: merge one
feature, confirm the commit, then merge the next.

---

## Issue 2: Sub-agents did not use codebase-memory-mcp graph tools (Moderate)

### What happened

All three sub-agents dispatched in parallel were given `graph_project: "Users-samphillips-Dev-kanbanzai"`
in their `handoff()` context. None of them used `search_graph`, `trace_call_path`, or
`codebase_memory_mcp_search_code`. Their outputs described finding relevant code by reading
specific files via `terminal` — a pattern closer to grepping than to structural graph navigation.

### Why it happened

The `handoff()` context and the dispatch prompts supplied explicit file paths for every task
(sourced from the dev plan's "Input context" sections). When a sub-agent is handed a complete
list of files to read, there is no discovery problem for graph tools to solve. The sub-agents
went directly to the files and read them, which is locally rational.

### Consequences

- Sub-agents read large files in full via `terminal`, consuming more context budget than targeted
  graph queries would have.
- Structural relationships (callers, callees, interface implementors) were inferred by reading
  rather than queried directly. This is slower and less reliable on large or interconnected files.
- The graph project context was wasted — it occupied token budget in the handoff without
  contributing to the output.

### Recommendation

Two changes in tension here:

1. **For well-scoped implementation tasks** (where the dev plan specifies exact files), supplying
   file paths in the prompt is correct — graph tools add no value over direct reads, and the
   overhead of a graph query is wasted. Continue supplying file paths for these.

2. **For discovery-heavy tasks** (understanding call chains, finding all callers of a function,
   tracing interface implementations), withhold the file list and require the sub-agent to use
   graph tools. The handoff prompt should include explicit graph tool instructions:
   `"Use search_graph and trace_call_path to locate the relevant code before reading files."`

The broader signal is that sub-agents default to direct file reads when they can. Graph tools
only get used when direct reads are not viable (file too large, path unknown) or when the
orchestrator explicitly mandates them.

---

## Issue 3: `write_file` was not available during its own delivery session (Moderate)

### What happened

FEAT-01KPQ08Y47522 implemented the `write_file` MCP tool specifically to replace the
`python3 -c` workaround for writing files in git worktrees. This feature was one of seven
being delivered in P25. The three sub-agents dispatched in this session did their work using
`python3 -c` — the exact pattern `write_file` was built to eliminate — because `write_file`
was not yet merged to `main` and the MCP server had not been restarted.

### Consequences

- Sub-agent prompts required explicit `python3 -c` instructions and warnings about escaping.
- A `SyntaxWarning` about invalid escape sequences (`"\s"`) appeared during file manipulation,
  indicating the fragility of the approach.
- Any sub-agent writing Go source files with complex string literals (regex patterns, YAML
  content, struct tags) is at risk of producing subtly malformed content through escaping errors.

### What `write_file` actually provides

The `write_file` tool accepts `path` and `content` as distinct JSON parameters and writes
atomically with `entity_id` scoping to a specific worktree working directory:

```
write_file(entity_id: "FEAT-01KPQxxx", path: "internal/service/foo.go", content: "...")
```

JSON string encoding handles all special characters without shell escaping. The sub-agent
never needs to construct a `python3 -c "..."` invocation — it just passes the file content
as a string value. This eliminates the entire class of escaping errors.

### Recommendation

After any merge that adds new MCP tools, **restart the MCP server before dispatching
implementation sub-agents**. The binary is rebuilt automatically by the merge tool (the
post-merge side-effect is visible in the merge output: `"trigger": "post-merge binary rebuild"`),
but the running process does not reload. A server restart is a one-line action and should be
treated as a mandatory step after merging infrastructure features.

---

## Issue 4: Rebase conflicts from undetected shared-file overlap (Minor)

### What happened

Two end-of-session merges required manual rebase and conflict resolution:

- **FEAT-01KPQ08Y71A8V** (fix empty task names) conflicted with `internal/service/decompose.go`,
  which FEAT-01KPQ08YBJ5AK (dev-plan-aware grouping) had already modified. Both features added
  new functions to the tail of the same file (`parseDevPlanTasks` and `deriveTaskName`).
- **FEAT-01KPQ08YKHNS9** (orchestration docs) conflicted with
  `.kbz/skills/orchestrate-development/SKILL.md`, which FEAT-01KPQ08YH16WZ (impl-workflow-docs)
  had already modified. Both features added new anti-pattern sections to the same document.

Both were `keep-both` resolutions — no semantic conflict, just two independent additions to
the same region that git could not place automatically.

### Root cause

The `conflict` tool was not used before the dispatch batch was assembled. A pre-dispatch
conflict check would have flagged both overlaps. Both were flagged as `DEP-003` in the
respective spec documents ("MUST be sequenced or their branches coordinated to avoid merge
conflicts") but that dependency was not enforced at dispatch time.

### Recommendation

Before dispatching any feature for merge, run `conflict(action: check, task_ids: [...])` across
all features in the merge batch. For features flagged with shared files, establish an explicit
merge order and merge them sequentially. This is already documented in the workflow but was not
followed in this session.

---

## Issue 5: Pre-existing nil pointer dereference in `ensureWorktree` (Significant)

### What happened

`TestWorktreeTransitionHook_BugToInProgress_CreatesWorktreeForBug` and its idempotent variant
had been non-deterministically panicking with a nil pointer dereference. The tests used
`t.Parallel()`, so whichever test ran first would crash, and the other would be reported
inconsistently.

The bug was in `internal/service/status_transition_hook.go:174`:

```go
existing, err := h.store.GetByEntityID(entityID)
if err == nil && existing.ID != "" {   // panic: existing is nil
```

`GetByEntityID` returns `(*Record, nil)` — nil pointer, no error — when no record is found.
The condition `err == nil` was true, so the runtime dereferenced `existing.ID` on a nil pointer.

### Production impact

This bug was not merely a test failure. In production, any call to `OnStatusTransition` for a
bug entity transitioning to `in-progress` would panic at this line on the first call (before
a worktree record existed). The MCP server recovers from panics in individual tool calls, so the
panic would have been silently swallowed — automatic worktree creation for new bugs would fail
without any visible error to the caller.

### Fix

```go
if err == nil && existing != nil && existing.ID != "" {
```

One character change. The fix was committed as `f31b949` and the tests now pass reliably across
five consecutive parallel runs.

### Why it was undetected

The tests had `t.Parallel()`, making the failure non-deterministic: it appeared sometimes as
`CreatesWorktreeForBug` and sometimes as `IdempotentWhenWorktreeExists` depending on scheduling.
This pattern is consistent with the project's existing knowledge entry on parallel test
isolation — tests that share any mutable state (even indirectly, through a panic recovery race)
should not use `t.Parallel()`. In this case the tests do not share state, but the panic in one
goroutine affects the other's reporting.

The bug was also masked at the system level by the MCP server's panic recovery, which prevents
a panicking tool handler from crashing the server process. This is good for stability but bad
for detectability — errors that should be surfaced are silently dropped.

---

## Broader Project Signals (from synthesis)

These signals pre-date P25 but remain open and relevant to the next iteration.

### Worktree file editing friction (Significant, ongoing)

The project's documented workaround for file edits in git worktrees is `python3 -c`. This is
known to be fragile for complex file content. AGENTS.md explicitly warns against heredoc syntax
in the `sh`-based terminal tool, but the knowledge base contains a conflicting entry favouring
heredoc for multi-line Go source files. The `write_file` tool (delivered in P25) resolves this
for sub-agents, but only once the server is restarted. The documentation in
`implement-task/SKILL.md` should be updated to recommend `write_file` as the primary pattern
and retire the `python3 -c` guidance.

### Stale MCP binary causes unexplained verification failures (Moderate, ongoing)

When implementation changes are made to the MCP server and the binary is rebuilt but the
process is not restarted, tool behaviour diverges from the new code. This has caused at least
one session to spend significant time diagnosing what looked like AC failures but were actually
the old code running. The `server_info` tool (delivered in an earlier plan) addresses
diagnosability, but the workflow does not enforce a "verify binary is current" check before
running integration tests.

### Commit discipline for workflow state (Moderate, ongoing)

`.kbz/state/` changes generated by MCP tools (entity transitions, document registrations,
knowledge contributions) are not always committed promptly. Sessions that do orchestration work
without claiming a task via `next(id:...)` never see the pre-task commit checklist. Orphaned
state files accumulate and are only caught at the next session boundary via the `git status`
check. This is particularly problematic in multi-agent sessions where one agent's uncommitted
state is invisible to parallel agents reading from the same store.

---

## What Worked Well

Several patterns performed reliably and should be preserved or expanded:

- **Parallel sub-agent dispatch with disjoint file scopes** produced zero conflicts across all
  prior plans where it was correctly set up. The P25 conflicts arose specifically from
  *features* with overlapping files, not from *tasks* within a feature — the per-task conflict
  detection that was established in P16/P17 remains sound.

- **`finish()` verification recording** — all three sub-agents recorded verification in their
  `finish()` calls, which populated task-level verification strings correctly. The
  `AggregateTaskVerification` feature (delivered in P25) will make this visible at the feature
  level automatically in future sessions.

- **Sub-agent output quality** — all three dispatched sub-agents produced working,
  spec-compliant code on the first attempt with no rework required. The `handoff()` context
  assembly (spec sections, knowledge entries, dev plan guidance) is doing its job.

- **Fact-checking sub-agents in documentation pipelines** — earlier plans established that
  dedicated check-stage sub-agents reliably find genuine errors (tool counts, lifecycle
  descriptions, struct fields) that write-stage agents miss. This pattern should be applied to
  any documentation feature in future plans.

---

## Recommendations for Next Iteration

Ranked by impact:

| # | Recommendation | Category | Effort |
|---|---|---|---|
| 1 | Serialise all `merge(action: execute)` calls — never run two merges in parallel | Operational discipline | Zero — change of practice only |
| 2 | Restart MCP server after any merge that adds new MCP tools; treat as mandatory step | Operational discipline | Zero — already a one-liner |
| 3 | Add `conflict(action: check)` across all features in a merge batch before executing merges | Tooling use | Low |
| 4 | Update `implement-task/SKILL.md` to recommend `write_file` as the primary worktree write pattern and retire `python3 -c` | Documentation | Low |
| 5 | Add a "verify binary is current" step (via `server_info`) to the pre-task checklist for sessions that run integration tests | Workflow | Low |
| 6 | Audit all code paths where `GetByEntityID` or similar pointer-returning store methods are called without nil checks | Code quality | Medium |
| 7 | For discovery-heavy sub-agent tasks, withhold explicit file paths and mandate graph tool use in the handoff prompt | Sub-agent dispatch | Medium |
| 8 | Add a `.kbz/state/` commit gate to the MCP server that refuses to advance feature status if uncommitted state files exist | Tooling | High |

---

## Appendix: P25 Commit Timeline

```
8a4a834  Merge FEAT-01KPQ08Y989P8 (propagate task verification)
54a5854  workflow(FEAT-01KPQ08Y989P8): mark worktree merged
62b5ed4  Merge FEAT-01KPQ08YBJ5AK (dev-plan-aware grouping)
b539467  workflow(FEAT-01KPQ08YBJ5AK): mark worktree merged
           ↑ also contains FEAT-01KPQ08YE4399 code (parallel squash race)
32fd299  workflow(FEAT-01KPQ08YE4399): mark worktree merged [manual patch]
f119e00  Merge FEAT-01KPQ08Y47522 (write_file tool)
d4c0904  Merge FEAT-01KPQ08Y71A8V (fix empty task names) [required rebase]
5ffbcde  Merge FEAT-01KPQ08YH16WZ (impl workflow docs)
f1e8807  Merge FEAT-01KPQ08YKHNS9 (orchestration docs) [required rebase]
f31b949  fix(service): nil pointer dereference in ensureWorktree
```
