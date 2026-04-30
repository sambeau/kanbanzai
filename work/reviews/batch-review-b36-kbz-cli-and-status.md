# Batch Conformance Review: B36-kbz-cli-and-status

## Scope

- **Batch:** B36-kbz-cli-and-status (kbz CLI and Status Command)
- **Features:** 4 total (0 done, 0 cancelled/superseded, **4 incomplete***)
- **Review date:** 2026-04-30
- **Reviewer:** reviewer-conformance

> *All 4 features are stuck in non-terminal lifecycle states (`developing`/`dev-planning`)
> despite 19/19 tasks being done. This is a lifecycle advancement gap, not an
> implementation gap. See **Conformance Gap CG-1**.

## Feature Census

| Feature | Status | Tasks | Spec Approved | Dev-Plan Approved | Design Approved | Notes |
|---------|--------|-------|---------------|-------------------|-----------------|-------|
| F1: Binary rename kanbanzai → kbz | **dev-planning** | 5/5 done ✅ | ✅ | ✅ | ✅ | Branch already merged to main. Worktree stale. |
| F2: Status argument resolution | **developing** | 6/6 done ✅ | ✅ | ✅ | ✅ | Branch has merge conflicts with main |
| F3: Status human output | **developing** | 4/4 done ✅ | ✅ | ✅ | ✅ | No merge conflicts |
| F4: Status machine output | **developing** | 4/4 done ✅ | ✅ | ✅ | ✅ | Branch has merge conflicts with main |

## Conformance Gaps

| # | Feature | Type | Description | Severity |
|---|---------|------|-------------|----------|
| CG-1 | All 4 | lifecycle | Features are in non-terminal states (`dev-planning`/`developing`) despite all 19 child tasks being done. The health check confirms `feature_child_consistency` warnings for all 4 features. Features must be advanced through their lifecycle to `done` before the batch can be conformance-passed. | **blocking** |
| CG-2 | All 4 | worktree-cleanup | All 4 worktrees are still `active`. F1's branch is already merged to main (38 commits behind, 0 ahead — unique commits absorbed). F2 and F4 branches have merge conflicts with main. Worktrees should be cleaned up post-merge and conflicts resolved for unmerged branches. | non-blocking |
| CG-3 | B36 | doc-currency | AGENTS.md line 78 still reads `├── cmd/kanbanzai/         ← binary entry point (CLI and MCP server)` — stale reference to the old directory name after the rename. Should be `cmd/kbz/`. | non-blocking |
| CG-4 | B36 | knowledge | No knowledge entries were contributed during the batch. Retrospective synthesis returned zero signals (no `finish(retrospective: [...])` calls were made during task completion). | non-blocking |

## Documentation Currency

- **AGENTS.md:** Needs one minor update — line 78 references `cmd/kanbanzai/` instead of `cmd/kbz/`.
- **Workflow skills:** Not directly affected by B36 — pre-existing stale tool references (not B36-specific).
- **Knowledge entries:** 0 contributed. The implementation team should consider contributing knowledge about the resolution disambiguation logic, TTY detection patterns, and JSON/plain renderer architecture for future reference.
- **Scope Guard:** B36 is not listed in AGENTS.md Scope Guard — this is a pre-existing condition shared by 30+ other completed batches.

## Retrospective Summary

No retrospective signals were recorded during B36 implementation. The `retro(action: "synthesise", scope: "B36-kbz-cli-and-status")` call returned zero themes. Retrospective observations should be contributed during task completion via `finish(retrospective: [...])` in future batches.

## Implementation Verification

Beyond the lifecycle gap, the actual implementation quality is sound:

### F1: Binary rename
- `cmd/kanbanzai/` removed ✅, `cmd/kbz/` exists ✅
- `Makefile` updated: `BINARY := kbz`, `go build ./cmd/kbz` ✅
- `mcpVersion` bumped to `2` ✅
- Migration detection in `kbz init` ✅
- Protocol name `"kanbanzai"` preserved in MCP config keys ✅
- All 15 acceptance criteria verified by tasks ✅
- Test suite passes (`go test ./cmd/kbz/...` ✅, `go vet` clean ✅)

### F2: Argument resolution
- `internal/resolution/` package with lexical `Disambiguate()` implementing NFR-001 (no I/O before decision) ✅
- Four resolution kinds: `ResolvePath`, `ResolveEntity`, `ResolvePlanPrefix`, `ResolveNone` ✅
- `--format`/`-f` flag implemented ✅
- File path resolution via document service lookup ✅
- Plan prefix resolution ✅
- `kbz doc approve` path resolution ✅
- Exit code semantics: 0 (success), 1 (errors), 2 (usage) ✅
- Unit tests passing ✅

### F3: Human output
- `internal/cli/render/` package with TTY detection (`golang.org/x/term`) ✅
- Unicode symbols when TTY, ASCII fallbacks when piped ✅
- 5 views: unregistered doc, registered doc + owner, feature, plan, project overview ✅
- Test coverage on render, alignment, colour, feature, plan, project modules ✅
- Tests passing ✅

### F4: Machine output
- `internal/cli/status/` with `PlainRenderer` (key:value for 6 scope types) ✅
- `JSONRenderer` with `results` array wrapper (D-7) and distinct `scope:project` shape (D-8) ✅
- Registered/unregistered document distinction via `registered:true/false` ✅
- Go build and vet clean ✅

### Outstanding note
- No test files exist in `internal/cli/status/` — the `PlainRenderer` and `JSONRenderer` have no dedicated unit tests. The "Schema stability contract test in CI" mentioned in the summary could not be found in `.github/workflows/`.

## Batch Verdict

**FAIL** — CG-1 (blocking): Features must be advanced to terminal lifecycle states.

## Recommended Actions

1. **Required:** Advance all 4 features through their lifecycle to `done`:
   - F1: `dev-planning` → `done` (branch already merged)
   - F2: `developing` → resolve merge conflicts → merge → `done`
   - F3: `developing` → merge → `done`
   - F4: `developing` → resolve merge conflicts → merge → `done`

2. **Required:** Clean up worktrees after merges

3. **Recommended:** Fix `cmd/kbz/` reference in AGENTS.md line 78

4. **Recommended:** Add unit tests for `PlainRenderer` and `JSONRenderer` methods and a CI schema stability contract test

5. **Recommended:** Contribute knowledge entries for architecture decisions made during this batch

## Evidence

- Batch entity: `entity(action: "get", id: "B36-kbz-cli-and-status")` → 4 features, status: active
- Feature list: `entity(action: "list", type: "feature", parent: "B36-kbz-cli-and-status")` → 4 features
- Spec approvals: All 4 specs approved ✅
- Health check: `health()` → 4 `feature_child_consistency` warnings for B36 features
- Retro synthesis: `retro(action: "synthesise", scope: "B36-kbz-cli-and-status")` → 0 signals
- Build verification: `go build ./cmd/kbz/` → success; `go vet ./internal/cli/... ./internal/resolution/...` → clean
- Test verification: `go test ./internal/cli/render/... ./internal/resolution/...` → pass
