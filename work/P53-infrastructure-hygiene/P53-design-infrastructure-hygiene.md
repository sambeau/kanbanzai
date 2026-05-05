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
7. **Plan/batch scope inspection.** `status(P50)` showed the strategic plan but did not give a usable aggregate view of referenced batches/features. Features referenced `B49-retro-fixes-may-2026`, but batch lookup/listing failed with slug-related errors. Plan review needs a clear scope census.
8. **Dirty work attribution.** `git status` surfaced mixed prior implementation changes, workflow/index metadata, and unrelated design edits. Agents need a grouped explanation of dirty files before deciding whether to proceed or commit.
9. **Document registration side-effect visibility.** `doc(register)` creates or updates index files, but the response does not make the resulting dirty files obvious enough for commit discipline.

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

### 7. Plan/batch scope inspection

**Observed:** During P50 review, the strategic plan entity existed, but the execution scope was only discoverable by manually listing reviewing features and inspecting their `parent` fields. The features referenced `B49-retro-fixes-may-2026`, but batch entity queries failed. The user summary said five features were ready for review; entity state showed four in `reviewing` and one still `developing`.

**Design:** Extend `status()` or add an internal scope resolver so a plan dashboard can show:

- child batches and whether each referenced batch entity exists
- child features reachable through batch references
- lifecycle status, task terminality, spec/design/dev-plan approval, review report presence, and worktree status for each feature
- explicit warnings for scope references that point at missing or unregistered entities

This is infrastructure hygiene because it fixes trust in the entity/state layer rather than changing review policy.

### 8. Dirty work attribution

**Observed:** The required pre-task `git status` check produced a large mixed dirty tree. Some files were prior P50 implementation changes, some were generated `.kbz/index` document metadata, and some were unrelated P44/P51/P52 design edits. A plain file list did not answer the operational question: "what is safe to touch or commit now?"

**Design:** Add a dirty-work attribution helper used by status/health output. It should group changed files by likely source:

- workflow state/index metadata (`.kbz/state`, `.kbz/index`, `.kbz/context`)
- current plan or feature document changes
- implementation changes under known feature worktrees or source packages
- unrelated plan/design documents
- untracked document registration records

The first version can be heuristic path grouping; it does not need perfect provenance. The goal is to turn a noisy dirty tree into actionable commit-discipline guidance.

### 9. Document registration side-effect visibility

**Observed:** Registering the P50 review report and remediation dev-plan created `.kbz/index/documents/*.yaml` files. The document response included registration data, but it did not explicitly explain which dirty files were created or updated.

**Design:** Extend document tool responses for mutating actions with an explicit side-effect list of repository paths created or modified. This should include index records and graph/index updates when known. The response should be lightweight and additive; it does not need to replace `git status`, only point the agent at expected dirty files.

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
| `internal/status/` or status dashboard rendering | Add plan/batch scope inspection and missing-entity warnings |
| `internal/health/` or new git status helper | Add dirty-work attribution groups |
| `internal/doc/` / MCP doc handlers | Include mutated index paths in document registration responses |

## Dependencies

- No dependencies on P51, P52, or P44
- Test compilation fixes are prerequisites for P51's test changes

## Open Questions

1. **Binary alignment:** Should we rename the binary or add a symlink? (Recommend: rename to `kanbanzai` — it's what the MCP config expects.)
2. **Cache staleness root cause:** Is it a race condition, a missing invalidation, or a query path that doesn't check cache freshness? Needs code investigation.
3. **edit_file threshold:** What's the `old_text` length limit, and should it be documented?
4. **Plan numbering:** Why doesn't `listAllPlanIDs` find P50? Needs code investigation before fix can be designed.
5. **Scope resolver location:** Should plan/batch scope inspection live in `status()`, `health()`, or a new internal resolver shared by both? (Recommend: shared resolver, surfaced first through `status()`.)
6. **Dirty attribution ownership:** Should dirty-work grouping be a Kanbanzai MCP tool, a health-check section, or a status dashboard section? (Recommend: status dashboard for task-start use, health for periodic audit.)
