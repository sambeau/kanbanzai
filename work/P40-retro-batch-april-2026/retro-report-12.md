# Retro Report 12 — Health Check Investigation: 158 CLI Status Errors

**Date:** 2026-04-30
**Author:** Sam Phillips (sambeau)

## Task

User asked whether 158 errors shown by `kanbanzai status` (and `kanbanzai
status <id>`) indicated a serious problem or merely fallout from incomplete P37
and P38-F8 migrations.

## What went well

- **The MCP `status` and `health` tools worked correctly.** They returned
  structured, parseable output that made it easy to compare the CLI experience
  against internal state. The `status` tool gave a clean project overview with
  zero errors/warnings; the `health` tool surfaced 60+ worktree and branch
  errors but zero of the "non-existent batch" errors that the CLI was showing.

- **The MCP `entity` tool was essential for scoping.** Calling `entity(action:
  "list", type: "feature", parent: "P38-plans-and-batches")` immediately
  surfaced that F8 (State File and Work Tree Migration) was `proposed` and
  explicitly marked "ON HOLD". The description text confirmed the dependency on
  P37-F5. Without this, I would have had to grep through YAML files directly.

- **`server_info` was the right first diagnostic call.** It showed the binary
  was in-sync with current source, ruling out stale-binary as the cause (a
  common failure mode noted in AGENTS.md).

- **The investigation required no code changes.** Pure read-only analysis —
  grep, file reads, CLI invocation, and MCP tool calls — surfaced the root
  cause in under 10 tool calls. No false starts.

- **The diagnostic vs. directive instruction was helpful and appropriate.**
  The user asked "is this because X or is this more serious" — a clear
  diagnostic question. I stayed in investigation mode, presented findings, and
  stopped without writing any code.

## What didn't go well

- **The CLI and MCP tools diverged on the same error.** `kanbanzai status`
  (CLI) reported 158 "non-existent batch" errors. The MCP `status` and `health`
  tools reported zero of these. This made the initial user experience alarming
  (158 errors looks like system corruption) but the MCP tools painted a much
  cleaner picture. The discrepancy between the two paths should not exist — the
  CLI and MCP tools share a common health-check codebase.

- **The `grep` and `codebase_memory_mcp_search_code` tools failed
  consistently.** Three consecutive grep calls with different patterns all
  returned "Failed to receive tool input: tool input was not fully received."
  The `search_graph` tool returned "project not found or not indexed." I
  switched to using `terminal` for grep, which worked immediately and was
  arguably faster for this task anyway. But the error messages were opaque and
  gave no hint as to cause or remediation.

- **The worktree sprawl created noise.** 60+ health errors about orphaned
  worktrees (missing paths, missing branches, merged-but-not-cleaned-up) made
  the health output harder to scan for the real signal. These are all cleanup
  operations that should have been automated.

- **The `P{n}` ID aliasing in the entity model is a footgun during migration.**
  `EntityKindPlan` silently aliases to `EntityKindBatch`, and `IsPlanID()`
  silently delegates to `IsBatchID()`. This meant the health check was looking
  up `P1-phase-4-orchestration` as a batch — which doesn't exist in
  `.kbz/state/batches/` (it's a plan in `.kbz/state/plans/`). The silent alias
  made the root cause harder to trace because the code says "plan" but means
  "batch" but neither exists where the code looks.

## What to improve

1. **Unify CLI and MCP health-check code paths.** The two should produce
   identical results. If the MCP path filtered these errors out deliberately,
   the CLI should too. If the MCP path missed them because of a different query
   path, that's also a bug. Either way, they should agree.

2. **Make grep/codebase-memory tool failures recoverable.** Having three
   consecutive tool calls fail identically with no recourse is frustrating. The
   system should either: (a) auto-fallback to terminal grep when the graph tool
   isn't available, or (b) give a clear, actionable error message ("Graph
   project not indexed — run index_repository first").

3. **Auto-schedule worktree cleanup on merge.** The `merge(action: "execute")`
   flow should automatically schedule a `cleanup` operation for the merged
   worktree, rather than leaving it for the next human to notice 60+ stale
   records.

4. **Dual-write entity state during migration.** When a plan entity is created
   in `.kbz/state/plans/` during the P38 migration, a corresponding batch
   record should also be written to `.kbz/state/batches/` until the migration
   completes. This would prevent the 158 broken-parent errors without waiting
   for F8 to finish. The transitional period needs a compatibility shim, not
   just deprecated aliases in the model layer.
