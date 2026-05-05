# Design: Infrastructure Hygiene

**Plan ID:** P53-infrastructure-hygiene
**Status:** Shaping
**Parent:** P50 (Retrospective Fixes — May 2026)

## Overview

The May 2026 retrospective and system instrumentation surfaced several infrastructure issues that degrade development velocity and trust in the tooling. These are not feature gaps — they're things that should work but don't. Fixing them removes friction from every other plan.

## Goals

1. **Stale binary detection and resolution.** The Makefile produces `kbz` but the editor MCP config expects `kanbanzai`. `server_info` detects the mismatch but doesn't fix it. The binary should align, or the MCP config should match the Makefile output.
2. **Store consistency.** Tasks showing `done` via `entity(get)` but `ready`/`active` via `entity(list)`. Cache staleness after `finish`. Untracked/modified state files at session start. The write-through cache needs to reliably reflect YAML state, or the list/queue queries need to bypass cache for status fields.
3. **Health checker false positives.** Done features flagged as "orphaned reviewing". Done plans flagged as "ready to close". Override transitions should be recognized as legitimate completion paths.
4. **Test compilation errors in internal/mcp/.** Redeclared functions (`runGit`), wrong `newServerWithConfig` signatures. These block running tests and would block CI.
5. **Plan numbering reuse.** `listAllPlanIDs` scans batch plans (`s.List("plan")`), not strategic plans. `NextPlanNumber` for prefix "P" sees no existing P-prefix IDs and returns 1. Fix: scan strategic plans (which share the same `plans/` directory) or unify the listing.
6. **`edit_file` reliability.** The tool failed repeatedly on design documents during P51/P52/P53 creation, requiring fallback to `python3`/`sed`. Needs investigation: token limit on `old_text`, fuzzy matching bug, or file-size threshold.

## Non-Goals

- Not changing the pipeline or handoff flow (P51)
- Not redesigning fast-track behavior (P52)
- Not building model routing (P44)
- Not fixing all cache staleness root causes — just the ones that produce visible state inconsistency

## Design

### 1. Stale binary

Two options:

**A: Align binary names.** Change the Makefile `BINARY` from `kbz` to `kanbanzai`, or add a symlink/copy step so `go install` updates both. The MCP config already expects `kanbanzai`.

**B: Add rebuild command.** A `kbz rebuild` or `kbz serve --rebuild` flag that recompiles and restarts the server. More complex but more robust.

Recommend: **Option A** — align the binary names. It's a one-line Makefile change.

### 2. Store consistency

The write-through pattern in `finish` calls `CacheRefresh` but the list/queue query may still return stale results. Investigation needed:

- Does `CacheRefresh` invalidate or update the specific entity's cache entry?
- Does `entity(action: list)` read from cache or filesystem?
- Are there race conditions between YAML write and cache update?

Minimum fix: Add a forced cache refresh after any status-changing operation. The `entity` tool's `get` action already bypasses cache for individual entity lookups — `list` should do the same for status fields, or accept a `bypass_cache: true` parameter.

### 3. Health checker false positives

**Orphaned reviewing:** When a feature is `done` via override, skip the "review report missing" check. The feature's status is terminal; it doesn't need a review.

**Ready to close plans:** Suppress the attention item for plans already in terminal state (`done`, `cancelled`). There is no further action.

### 4. Test compilation errors

Fix the following in `internal/mcp/`:
- `merge_tool_cleanup_test.go:20:6`: `runGit` redeclared (also in `branch_tool_test.go:236`)
- `annotations_test.go:16:52`: `newServerWithConfig` too many arguments
- `server_groups_test.go:18:52`: same
- `tool_description_budget_test.go:31:52`: same

These are merge artifacts from the `newServerWithConfig` signature change.

### 5. Plan numbering

`listAllPlanIDs` calls `s.List("plan")` → `entityDirectory("plan")` returns `"plans"` → reads both batch plans (B-prefix) and strategic plans (P-prefix) from the same directory. But `NextPlanNumber` filters by prefix, and the batch plan IDs start with "B", not "P". The strategic plans starting with "P" should be found...

**Actual investigation needed:** Why doesn't `listAllPlanIDs` find P50? The `plans/` directory contains both `P50-retro-may-2026.yaml` and batch plans. If `s.List("plan")` reads all of them, `NextPlanNumber("P", ...)` should find P50 and return 51. Something is filtering them out.

### 6. edit_file reliability

**Observed:** During P51/P52/P53 design document creation, `edit_file` failed on multiple attempts to insert text into existing sections. The `old_text` was unique and present in the file, but matching failed. Fallback to `python3` and `sed` succeeded immediately.

**Hypotheses:**
- `old_text` exceeds a token/character limit
- Fuzzy matching has a threshold that long `old_text` doesn't meet
- The file size (~8KB) triggers a different matching strategy

**Investigation:** Test with progressively shorter `old_text` on the same file. If a length threshold exists, it should be documented in the tool description.

## Files affected

| File | Change |
|------|--------|
| `Makefile` | Align binary name (`kbz` → `kanbanzai` or symlink) |
| `internal/service/entities.go` | Fix `listAllPlanIDs` to include strategic plans |
| `internal/mcp/merge_tool_cleanup_test.go` | Fix `runGit` redeclaration |
| `internal/mcp/annotations_test.go` | Fix `newServerWithConfig` signature |
| `internal/mcp/server_groups_test.go` | Fix `newServerWithConfig` signature |
| `internal/mcp/tool_description_budget_test.go` | Fix `newServerWithConfig` signature |
| `internal/health/` | Skip orphaned-review check for done features; suppress done-plan attention items |
| `internal/service/` (cache) | Investigate and fix cache staleness after `finish` |

## Dependencies

- No dependencies on P51, P52, or P44
- Test compilation fixes are prerequisites for P51's test changes

## Open Questions

1. **Binary alignment:** Should we rename the binary or add a symlink? (Recommend: rename to `kanbanzai` — it's what the MCP config expects.)
2. **Cache staleness root cause:** Is it a race condition, a missing invalidation, or a query path that doesn't check cache freshness? Needs code investigation.
3. **edit_file threshold:** What's the `old_text` length limit, and should it be documented?
4. **Plan numbering:** Why doesn't `listAllPlanIDs` find P50? Needs code investigation before fix can be designed.
