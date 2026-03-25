# Phase 3 Progress

**Last updated:** 2026-03-25

**Status:** Implementation complete — post-implementation audit identified remediation items. Remediation not yet started.

**Purpose:** Track implementation status of Phase 3 deliverables (Git integration and knowledge lifecycle) against the Phase 3 specification (§20 acceptance criteria), and record the results of the post-implementation audit.

**Related documents:**

- `work/spec/phase-3-specification.md` — binding specification
- `work/plan/phase-3-scope.md` — Phase 3 scope and planning
- `work/plan/phase-3-implementation-plan.md` — implementation plan and work breakdown
- `work/plan/phase-3-decision-log.md` — design decisions (P3-DES-001 through P3-DES-008)
- `work/plan/phase-2b-progress.md` — Phase 2b completion status (predecessor)

---

## 1. Implementation Status Summary

All 11 tracks (A–K) are implemented. All tests pass with race detector enabled (`go test -race ./...`). `go vet` is clean.

A post-implementation audit has identified 2 critical bugs, 11 correctness issues, 10 code quality issues, and 6 documentation gaps. These are catalogued in §5 below and must be addressed before Phase 3 can be considered complete.

| Track | Name | Status | Key Files |
|-------|------|--------|-----------|
| A | Worktree management | ⚠️ Implemented (audit items) | `internal/worktree/` |
| B | Branch tracking | ✅ Complete | `internal/git/branch.go` |
| C | Merge gates | ⚠️ Implemented (audit items) | `internal/merge/` |
| D | GitHub PR integration | ⚠️ Implemented (audit items) | `internal/github/` |
| E | Post-merge cleanup | ⚠️ Implemented (audit items) | `internal/cleanup/` |
| F | Git anchoring and staleness | ✅ Complete | `internal/git/anchor.go`, `internal/git/staleness.go` |
| G | TTL-based pruning | ⚠️ Implemented (minor items) | `internal/knowledge/ttl.go`, `internal/knowledge/prune.go` |
| H | Automatic promotion | ✅ Complete | `internal/knowledge/promotion.go` |
| I | Post-merge compaction | ❌ Critical bugs | `internal/knowledge/compact.go`, `internal/mcp/knowledge_tools.go` |
| J | Health check extensions | ⚠️ Implemented (minor items) | `internal/health/` |
| K | Configuration | ⚠️ Implemented (audit items) | `internal/config/` |

---

## 2. Acceptance Criteria Status

Tracking against spec §20 acceptance criteria.

### §20.1 Worktree management — ⚠️ Partially met

- [x] Worktree creation follows suggested branch naming convention
- [x] Worktree-entity relationship is tracked in state
- [x] `worktree_list` returns all worktrees with correct status
- [x] `worktree_remove` removes worktree and updates state
- [ ] Worktree is created automatically when first task in feature/bug starts — **not linked to task lifecycle; standalone tool only**
- [ ] Worktree creation failure does not block task transition — **cannot evaluate; creation not linked to transitions**

### §20.2 Branch tracking — ✅ Met

- [x] `branch_status` returns accurate metrics from Git
- [x] Stale branch detection uses configured thresholds
- [x] Merge conflict detection is accurate
- [x] Drift calculation (commits behind/ahead) is correct

### §20.3 Merge gates — ⚠️ Mostly met

- [x] All defined gates are checked by `merge_readiness_check`
- [x] Gate failures block merge by default
- [x] Override with reason allows merge despite failures
- [x] Gate check output is structured and actionable
- [ ] Override events are logged — **override structures exist but are never persisted to worktree record (R2)**

### §20.4 GitHub PR integration — ✅ Met

- [x] PR creation works with valid GitHub token
- [x] PR description is generated from entity state
- [x] PR update syncs description and labels
- [x] PR status returns CI, review, and conflict status
- [x] Missing GitHub config returns clear error
- [x] Labels are created if they don't exist

### §20.5 Post-merge cleanup — ✅ Met

- [x] Merged worktree transitions to `merged` status
- [x] Cleanup is scheduled for grace period after merge
- [x] `cleanup_list` shows pending and scheduled items
- [x] `cleanup_execute` removes worktrees and branches
- [x] Remote branch deletion works if configured
- [x] Dry-run mode works correctly

### §20.6 Git anchoring — ✅ Met

- [x] `git_anchors` field accepts list of file paths
- [x] Staleness detection compares file modification to `last_confirmed`
- [x] Stale entries are flagged in retrieval responses
- [x] Stale entries appear in health check
- [x] `knowledge_confirm` clears staleness flag
- [x] Entries without anchors are not flagged as stale

### §20.7 TTL-based pruning — ✅ Met

- [x] TTL is computed from `last_used` + `ttl_days`
- [x] TTL resets on knowledge use
- [x] Pruning respects tier-specific conditions
- [x] New entries have grace period before pruning
- [x] Pruned entries transition to `retired`
- [x] Dry-run mode works correctly

### §20.8 Automatic promotion — ✅ Met

- [x] Promotion triggers when conditions met (use_count >= 5, miss_count = 0, confidence >= 0.7)
- [x] Promotion updates tier and TTL
- [x] Promotion is logged and visible in health check
- [x] Manual promotion works via tool/command
- [x] Cannot promote directly to Tier 1

### §20.9 Post-merge compaction — ❌ Not met (critical bugs)

- [x] Exact duplicates are detected and merged (logic correct)
- [x] Near-duplicates (Jaccard > 0.65) are handled correctly (logic correct)
- [x] Contradictions are marked `disputed` (logic correct)
- [x] Tier 2 entries are flagged, not auto-modified
- [x] Tier 1 entries are never compacted
- [x] Dry-run mode works correctly
- [ ] Kept entries with merged data are persisted — **computed but never written to storage (R1a)**
- [ ] Disputed entries are persisted — **status change never saved (R1a)**
- [ ] `knowledge_resolve_conflict` resolves disputes — **`merge_content` is a no-op (R1b)**

### §20.10 Health checks — ✅ Met

- [x] New health check categories are implemented (all 6)
- [x] Health output includes all new categories
- [x] Severity levels are correct
- [x] Actionable messages are provided

### §20.11 Configuration — ⚠️ Mostly met

- [x] All configuration options are documented
- [x] Defaults are sensible
- [x] Local config is gitignored
- [ ] Invalid configuration is rejected with clear errors — **no validation for Phase 3 config fields (R5)**

### §20.12 Deterministic storage — ✅ Met

- [x] Worktree records follow deterministic YAML format
- [x] Round-trip (read-write-read) produces identical output
- [x] Field order matches specification

---

## 3. MCP Tools and CLI

### MCP Tools — ✅ All 18 tools registered

| Category | Tools | Status |
|----------|-------|--------|
| Worktree | `worktree_create`, `worktree_list`, `worktree_get`, `worktree_remove` | ✅ |
| Branch | `branch_status` | ✅ |
| Merge | `merge_readiness_check`, `merge_execute` | ✅ |
| PR | `pr_create`, `pr_update`, `pr_status` | ✅ |
| Cleanup | `cleanup_list`, `cleanup_execute` | ✅ |
| Knowledge | `knowledge_check_staleness`, `knowledge_confirm`, `knowledge_prune`, `knowledge_promote`, `knowledge_compact`, `knowledge_resolve_conflict` | ✅ |

### CLI Commands — ✅ All implemented

| Command | Subcommands | Status |
|---------|-------------|--------|
| `kbz worktree` | `list`, `create`, `show`, `remove` | ✅ |
| `kbz branch` | `status`, `list` | ✅ |
| `kbz merge` | `check`, `run` | ✅ |
| `kbz pr` | `create`, `update`, `status` | ✅ |
| `kbz cleanup` | `list`, `run` | ✅ |
| `kbz knowledge` (Phase 3 additions) | `check`, `confirm`, `prune`, `compact`, `resolve` | ✅ |

---

## 4. Test Status

All tests pass: `go test -race ./... ✅` and `go vet ./... ✅`.

### Test coverage by package

| Package | Unit Tests | MCP Tool Tests | Rating |
|---------|-----------|----------------|--------|
| `worktree/` | ✅ Excellent (23 test funcs) | ⚠️ Good (5 tests, minor gaps) | Strong |
| `git/` | ✅ Excellent (real Git repos) | N/A | Strong |
| `merge/` | ✅ Good (all gates, overrides, format) | ❌ None for `merge_tools.go` | Medium |
| `github/` | ✅ Good (httptest mocks) | ❌ None for `pr_tools.go` | Medium |
| `cleanup/` | ⚠️ Weak (only dry-run + helpers) | ❌ None for `cleanup_tools.go` | Weak |
| `knowledge/` | ✅ Excellent (55 test funcs) | ❌ None for knowledge MCP tools | Medium |
| `health/` | ✅ Excellent (34 category tests) | ⚠️ Partial | Strong |
| `config/` | ✅ Good (defaults + parse) | N/A | Good |

### Key test gaps

- No MCP tool integration tests for `merge_tools.go`, `pr_tools.go`, `cleanup_tools.go`, or Phase 3 additions to `knowledge_tools.go`
- Cleanup `execute.go` non-dry-run path is untested (mock type defined but unusable — `ExecuteCleanup` takes concrete `*worktree.Git`)
- Compaction persistence would have been caught by integration tests that verify disk state

---

## 5. Post-Implementation Audit

The audit reviewed all Phase 3 code against the specification and identified remediation items across four severity tiers. Items are numbered R1–R17 for tracking.

### Must-fix (R1–R4) — ❌ Not yet started

These items block Phase 3 completion.

**R1: Compaction persistence — computed results not written to storage**

Two related bugs in `internal/mcp/knowledge_tools.go`:

- **R1a: `knowledge_compact` tool** — Compaction computes correct results (kept entries with merged data, disputed entries) but only partially writes them. Kept entries' updated `use_count`, `merged_from`, and `git_anchors` are never saved. Disputed entries' status change is never persisted. Only `svc.Retire()` is called for retired entries, and those errors are silently swallowed.

  Fix: Add a `UpdateFields(id, fields)` method to the knowledge service (or use existing service methods) to persist kept and disputed entries. Surface retirement errors in the response.

- **R1b: `knowledge_resolve_conflict` tool** — The `merge_content: true` parameter is dead code. `MergeEntries` is called and the result is discarded. The merged use count is computed and assigned to `_`. The swap logic (reversing arguments to honor the user's `keep` choice) doesn't work because `MergeEntries` picks by confidence, not argument order.

  Fix: Either implement the write-back (requires service API for field updates) or remove the `merge_content` parameter and document the limitation.

**R2: Override events not persisted**

The spec §8.4 requires merge gate overrides to be logged in the worktree record. The `Override` struct, `CreateOverrides`, and `ValidateOverride` functions exist in `internal/merge/override.go`, but they are never called from `executeMerge` in `merge_tools.go`. The override reason is only included in the merge commit message. The entire `override.go` module is dead code from the MCP path.

Fix: Wire `CreateOverrides` into `executeMerge` and persist override records in the worktree record via store update.

**R3: Update AGENTS.md**

Three issues make `AGENTS.md` incorrect for the current state:

- Project status section does not mention Phase 3; Phase 3 spec not listed as binding contract.
- Repository structure is missing 6 new packages (`git/`, `github/`, `merge/`, `cleanup/`, `worktree/`, `health/`); existing package descriptions don't mention Phase 3.
- Scope guard lists "GitHub automation" and "Worktree automation" as things not to build — both are Phase 3 deliverables that are already implemented.

Fix: Update all three sections to reflect Phase 3 completion.

**R4: Update README.md**

- Phase 3 not mentioned in current status.
- "What's still ahead" lists worktree management and branch tracking as future work — these are implemented.
- CLI examples section has no Phase 3 commands.
- Contributor reading order doesn't list Phase 3 spec.
- `.kbz/` storage tree diagram doesn't show `state/worktrees/`.

Fix: Update all stale sections.

### Should-fix (R5–R11) — ❌ Not yet started

These items are correctness or quality issues that should be addressed soon after must-fix items.

**R5: No validation for Phase 3 configuration fields**

Spec §20.11 says "invalid configuration rejected with clear errors." The `Validate()` method in `internal/config/config.go` only validates prefix configuration. Negative values for `stale_after_days`, invalid confidence ranges, `drift_warning_commits >= drift_error_commits`, etc. are all silently accepted.

Fix: Add Phase 3 config validation (non-negative days, valid ranges, warning < error thresholds, confidence in [0,1]).

**R6: Zero-value config defaults cause spurious health warnings**

Pre-Phase 3 config files have zero values for all Phase 3 fields. A `StaleAfterDays` of 0 means every branch is immediately stale. A `DriftErrorCommits` of 0 means every branch triggers an error. `Phase3HealthChecker` reads these values directly without defaults.

Fix: Either merge defaults during `LoadFrom` for missing sections, or check for zero values in consumers and substitute defaults.

**R7: Add MCP tool integration tests**

No test coverage for `merge_tools.go`, `pr_tools.go`, `cleanup_tools.go`, or Phase 3 knowledge tools. The compaction persistence bugs (R1) would have been caught by integration tests verifying disk state.

Fix: Add integration tests for each MCP tool file, following the pattern in `worktree_tools_test.go`.

**R8: Grace period calculation inconsistency**

`ScheduleCleanup` in `cleanup/schedule.go` uses `time.Add(N * 24 * time.Hour)` while `MarkMerged` in `worktree/worktree.go` uses `time.AddDate(0, 0, N)`. These produce different results across DST boundaries.

Fix: Use `AddDate(0, 0, N)` consistently.

**R9: Consolidate main/master default branch fallback**

Three separate sites in `merge/gates.go` and `mcp/merge_tools.go` duplicate the main → master fallback logic, in addition to the existing `getDefaultBranch()` helper in `git/git.go`. The hardcoded fallback in `pr_tools.go` is also a problem — teams with `develop` or `trunk` as default branch will fail.

Fix: Use `getDefaultBranch()` (or detect via GitHub API for PR operations) consistently everywhere.

**R10: Swallowed errors in merge and cleanup**

Several locations discard errors with comments saying "log" but no logging:
- `merge_tools.go` L167: worktree update after merge
- `merge_tools.go` L137-142: conflict check fallback — if both main and master fail, merge proceeds without conflict checking
- `cleanup/execute.go` L64-70: remote branch deletion error discarded
- `pr_tools.go` L193-196 and L207-209: label operation errors discarded

Fix: At minimum log or include errors in the response. For the conflict check, fail if both branches are absent rather than silently skipping.

**R11: Cleanup `ExecuteAllReady` skips abandoned worktrees**

`cleanup_list` shows abandoned worktrees as pending, but `ExecuteAllReady` silently skips them because `IsReadyForCleanup` returns false when `CleanupAfter` is nil. An agent sees "this needs cleanup" but gets zero results from execute.

Fix: Either handle abandoned worktrees in `ExecuteAllReady`, or only show abandoned items in `cleanup_list` when they have `CleanupAfter` set.

### Nice-to-have (R12–R17) — ❌ Not yet started

These are code quality improvements.

**R12: Replace custom stdlib reimplementations**

Custom `contains`/`toLower` functions (3+ locations: `worktree_tools.go`, `cleanup/execute.go`, `git/branch_test.go`) should use `strings.Contains(strings.ToLower(...))`. The `formatCount` function in `health/format.go` (28 lines) should use `strconv.Itoa`.

**R13: Remove dead code**

- `getFieldFloat` in `knowledge/ttl.go`
- `ExtractFields` and `CollectFieldsFromRecords` in `knowledge/prune.go`
- `mockGit` in `cleanup/execute_test.go`
- Unused `gracePeriodDays` parameter in `ExecuteAllReady`

**R14: Consistency fixes in knowledge package**

- `ResetTTL` uses `time.Now()` internally (not injectable for testing) while `ApplyPromotion` accepts `now time.Time` — make consistent.
- `ResetTTL` mutates input map in-place while `ApplyPromotion` copies — make consistent.
- `GetEntryID` reads `"entry_id"` but `PruneExpiredEntries` reads `"id"` — resolve field name inconsistency.
- Duplicated `entityTypeFromID` in `mcp/worktree_tools.go` and `knowledge/links.go`.

**R15: Add `context.Context` to GitHub HTTP client**

None of the GitHub client methods accept `context.Context`. A slow or unresponsive GitHub API will block the MCP handler indefinitely with no cancellation.

**R16: Minor spec deviations in worktree naming**

- Bug branch prefix is `bugfix/` (code) vs `bug/` (spec §6.5) — confirm which is canonical.
- Slug length not capped at 40 characters per spec §6.5.

**R17: Unused `FormatGateResults`**

`merge/format.go` provides `FormatGateResults` but the MCP tool in `merge_tools.go` manually rebuilds the same structure. Either use the function or remove it.

---

## 6. Remediation Plan

### Phase 3 completion sequence

1. **Must-fix items (R1–R4)** — required before Phase 3 can be marked complete
2. **Should-fix items (R5–R11)** — address promptly after must-fix
3. **Nice-to-have items (R12–R17)** — address as time permits

### Recommended implementation order

| Order | Item | Estimated effort | Rationale |
|-------|------|-----------------|-----------|
| 1 | R1a + R1b | M | Critical correctness bug; blocks §20.9 acceptance |
| 2 | R2 | S | Dead code; straightforward wiring |
| 3 | R3 + R4 | S | Documentation only; no code risk |
| 4 | R5 + R6 | S | Config validation; low risk |
| 5 | R7 | L | Integration tests for 4 MCP tool files |
| 6 | R8 | S | One-line fix |
| 7 | R9 | S | Extract helper, replace 3 call sites |
| 8 | R10 | S | Add logging at each site |
| 9 | R11 | S | Logic fix in list or execute |
| 10 | R12–R17 | S each | Mechanical cleanup |

Size key: S = small (< 1 hour), M = medium (1–3 hours), L = large (3+ hours)

### Automatic worktree creation (§20.1 criteria 1 and 6)

The spec requires worktrees to be created automatically when the first task in a feature/bug transitions to `in_progress`. This is not implemented — `worktree_create` is a standalone MCP tool. This is the largest structural gap and may require design discussion before implementation, since it touches the task lifecycle state machine in `internal/validate/`.

This is recorded here but not assigned a remediation number because it represents a feature gap rather than a bug in existing code. It should be discussed with the human before proceeding.

---

## 7. Deferred to Phase 4

The following capabilities are explicitly out of scope for Phase 3:

- Orchestration or agent delegation
- Cross-project knowledge sharing
- Embedding-based semantic similarity for deduplication
- GitLab, Bitbucket, or other platform support
- Webhook-based real-time synchronisation
- Automated PR merging without human approval