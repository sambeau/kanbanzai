# Plan Review: P24-retro-recommendations — Retro Recommendations

| Field    | Value                              |
|----------|------------------------------------|
| Plan     | P24-retro-recommendations          |
| Reviewer | Claude (reviewer-conformance)      |
| Date     | 2026-04-21T13:00:00Z               |
| Verdict  | Pass with findings                 |

---

## Feature Census

| Feature            | Slug                          | Status | Terminal | Notes |
|--------------------|-------------------------------|--------|----------|-------|
| FEAT-01KPPG1MF4DAT | ac-pattern-and-decompose      | done   | ✅       |       |
| FEAT-01KPPG2MYSG6A | doc-approve-status-patch      | done   | ✅       |       |
| FEAT-01KPPG3MSRRCE | standalone-bug-visibility     | done   | ✅       |       |
| FEAT-01KPPG4SXY6T0 | workflow-hygiene-docs         | done   | ✅       |       |
| FEAT-01KPPG5XMJWT3 | optional-github-pr            | done   | ✅       |       |

All 5 features are in `done` status. All 17 tasks across the plan are in terminal state.
Plan lifecycle status is `proposed` — it was never advanced despite all work completing.
This is a lifecycle hygiene item, not a delivery gap.

---

## Specification Approval

| Feature            | Spec Document                              | Status       |
|--------------------|--------------------------------------------|--------------|
| FEAT-01KPPG1MF4DAT | work/spec/p24-ac-pattern-and-decompose.md  | approved ✅  |
| FEAT-01KPPG2MYSG6A | work/spec/p24-doc-approve-status-patch.md  | approved ✅  |
| FEAT-01KPPG3MSRRCE | work/spec/p24-standalone-bug-visibility.md | approved ✅  |
| FEAT-01KPPG4SXY6T0 | work/spec/p24-workflow-hygiene-docs.md     | approved ✅  |
| FEAT-01KPPG5XMJWT3 | work/spec/p24-optional-github-pr.md        | approved ✅  |

All 5 specifications are in approved status.

---

## Spec Conformance Detail

### FEAT-01KPPG1MF4DAT — AC Pattern Recognition and Decompose Hardening

Source recommendation: REC-01. Three independent fix sites plus a diagnostic improvement.

| # | Criterion | Result | Notes |
|---|-----------|--------|-------|
| AC-001 | `extractConventionalRoles` assigns `Role: "requirement"`, `Confidence: "medium"` to a non-AC-headed section whose body contains bold-ident lines | ✅ | `TestExtractConventionalRoles_ACContentInNonACSection` in `internal/docint/extractor_test.go` |
| AC-002 | Heading-matched section not duplicated by content scan | ✅ | `TestExtractConventionalRoles_NoDuplicateRole` |
| AC-003 | `acBoldIdentLineRe` matches bare and list-item forms; rejects single-asterisk variant | ✅ | `TestAcBoldIdentLineRe_BothForms`; single-asterisk rejection confirmed by regex definition |
| AC-004 | `parseSpecStructure` extracts `- **AC-01.** text` as `"AC-01: text"` (2 criteria) | ✅ | `TestParseSpecStructure_ListItemBoldIdent` |
| AC-005 | `asmExtractCriteria` normalises `- **AC-01.** text` to `"AC-01: text"` (no raw bold markers) | ✅ | `TestAsmExtractCriteria_ListItemBoldIdent_Normalised` |
| AC-006 | Zero-criteria error when bold-idents exist outside an AC section contains section list and non-zero outside count | ✅ | `TestDecomposeFeature_RichDiagnostic_BoldOutsideSection` |
| AC-007 | Zero-criteria error when no bold-idents anywhere contains section list and zero outside count | ✅ | `TestDecomposeFeature_RichDiagnostic_NoBoldIdents` |
| AC-008 | All three named verify-then-fix tests exist and pass | ✅ | Confirmed present by name in test files |
| FR-009 | `buildZeroCriteriaDiagnostic` is an unexported helper | ✅ | Consistent with NFR-003 |
| NFR-001 | Detection is deterministic (no LLM, no external service) | ✅ | Pure regex, verified in extractor.go |

**Summary:** Fully implemented and tested. All three fix sites (extractor, parseSpecStructure, asmExtractCriteria) and the diagnostic improvement are verified by dedicated verify-then-fix tests.

---

### FEAT-01KPPG2MYSG6A — doc approve Patches Status in Source File

Source recommendation: REC-03.

| # | Criterion | Result | Notes |
|---|-----------|--------|-------|
| AC-001 | Pipe-table `\| Status \| Draft \|` → `\| Status \| approved \|`, returns `(true, nil)` | ✅ | `TestPatchStatusField_PipeTable` |
| AC-002 | Bullet `- Status: Draft` → `- Status: approved`, returns `(true, nil)` | ✅ | `TestPatchStatusField_BulletList` |
| AC-003 | Bare YAML `status: draft` → `status: approved`, returns `(true, nil)` | ✅ | `TestPatchStatusField_BareYAML` |
| AC-004 | Case-insensitive: `\| STATUS \| Draft \|` matched and normalised | ✅ | `TestPatchStatusField_CaseInsensitive` |
| AC-005 | No Status field → returns `(false, nil)`, file unchanged | ✅ | `TestPatchStatusField_NoField` |
| AC-006 | Two Status field lines → only first replaced | ✅ | `TestPatchStatusField_FirstMatchOnly` |
| AC-007 | Non-existent file → returns `(false, err)` | ✅ | `TestPatchStatusField_FileNotFound` |
| AC-008 | Full round-trip: `doc(approve)` patches file, refreshes content hash | ✅ | `TestApproveDocument_PatchesStatusField` |
| AC-009 | Unreadable file → approval succeeds, WARNING logged | ✅ | `TestApproveDocument_PatchFailure_ApprovalSucceeds` |
| AC-010 | No Status field → approval succeeds, file unchanged, no WARNING | ✅ | `TestApproveDocument_NoStatusField_NoSideEffects` |
| AC-011 | Hash refresh failure → approval succeeds, WARNING logged | ✅ | `TestApproveDocument_HashRefreshFailure_ApprovalSucceeds` |
| AC-012 | `PatchStatusField` calls `WriteFileAtomic`, not `os.WriteFile` | ✅ | Confirmed by inspection of `internal/fsutil/status_patch.go` |
| NFR-004 | Patch applied in service layer (`ApproveDocument`), not in `doc_tool.go` | ✅ | Confirmed by code location |

**Summary:** Fully implemented. All 7 unit tests and 4 integration tests are present and named exactly as specified. Best-effort semantics are correctly implemented: no patch failure can prevent the document record being approved.

---

### FEAT-01KPPG3MSRRCE — Standalone Bugs Visible in Status Health

Source recommendation: REC-04.

| # | Criterion | Result | Notes |
|---|-----------|--------|-------|
| AC-001 | Standalone `high` bug in `reported` state → `open_critical_bug` item with correct fields | ✅ | `TestSynthesiseProject_StandaloneBug_HighSeverity` |
| AC-002 | Standalone `critical` bug → item with `"Standalone critical bug: <name>"` | ✅ | `TestSynthesiseProject_StandaloneBug_CriticalSeverity` |
| AC-003 | Empty bug name → Message uses bug ID | ✅ | `TestSynthesiseProject_StandaloneBug_EmptyName` |
| AC-004 | Feature-linked `high` bug NOT in project attention | ✅ | `TestSynthesiseProject_StandaloneBug_FeatureLinked_Excluded` |
| AC-005 | `closed` standalone bug excluded | ✅ | `TestSynthesiseProject_StandaloneBug_Closed_Excluded` |
| AC-006 | `done`, `not-planned`, `duplicate`, `wont-fix` statuses all excluded | ✅ | `TestSynthesiseProject_StandaloneBug_ResolvedStatuses_Excluded` |
| AC-007 | `medium` severity excluded | ✅ | `TestSynthesiseProject_StandaloneBug_MediumSeverity_Excluded` |
| AC-008 | `low` severity excluded | ✅ | `TestSynthesiseProject_StandaloneBug_LowSeverity_Excluded` |
| AC-009 | Pre-existing attention items appear before standalone-bug items | ✅ | `TestSynthesiseProject_StandaloneBug_OrderingPreserved` |
| AC-010 | Not present in plan-scoped `status` | ✅ | Block is inside `synthesiseProject` only; confirmed by code inspection |
| AC-011 | Not present in feature-scoped `status` | ✅ | Same; `synthesiseFeature` has its own feature-linked bug path |
| AC-012 | `List("bug")` error → `synthesiseProject` succeeds, no crash | ✅ | `TestSynthesiseProject_StandaloneBug_ListError_Ignored` |
| REQ-NF-002 | `AttentionItem` struct not modified; `"open_critical_bug"` type reused | ✅ | Confirmed by inspection |

**Summary:** Fully implemented with a complete 12-test suite. The severity filter (`high` / `critical` only) and the resolved-status skip list match the existing feature-linked bug logic exactly.

---

### FEAT-01KPPG4SXY6T0 — Workflow Hygiene Documentation

Source recommendations: REC-02, REC-05, REC-06, REC-07, REC-08. Documentation-only changes across four files.

| # | Criterion | Result | Notes |
|---|-----------|--------|-------|
| AC-001 | `kanbanzai-getting-started/SKILL.md` checklist: no stash instruction; commit or stop-and-inform; `git stash` explicitly prohibited | ✅ | Verified: "Never use `git stash` in a Kanbanzai project" present |
| AC-002 | "Clean slate" section: two-action rule (commit or stop-and-inform); verbatim stash prohibition; worktree-switch loss risk | ✅ | "silently lost when switching worktrees" present |
| AC-003 | `AGENTS.md` "Before Every Task" stash parenthetical: names parallel-agent state hiding and worktree-switch loss | ✅ | "stashing hides state from parallel agents and is silently lost across worktree switches" present |
| AC-004 | "Commit workflow state" section in `kanbanzai-getting-started/SKILL.md`: after "Clean slate"; names `.kbz/state/`, `.kbz/index/`, `.kbz/context/`; `git add .kbz/` sequence; consequence for parallel agents | ✅ | Section present immediately after "Clean slate" with all required elements |
| AC-005 | `AGENTS.md` checklist has dedicated orphaned-state item separate from `git status` code item | ✅ | "Commit orphaned workflow state" is its own bullet |
| AC-006 | `implement-task/SKILL.md` "Worktree File Editing" section: between Anti-Patterns and Checklist; `edit_file` warning; `python3 -c` pattern; heredoc alternative | ✅ | Section present with all elements |
| AC-007 | `implement-task/SKILL.md` checklist: worktree-confirmation item immediately after "Claimed the task" | ✅ | Confirmed by inspection |
| AC-008 | `AGENTS.md` "Delegating to Sub-Agents": callout referencing `implement-task` skill and `python3 -c` at end of section | ✅ | "Sub-agents that run inside a Git worktree cannot use `edit_file`… use `terminal` using the `python3 -c` pattern. See the `implement-task` skill" present |
| AC-009 | `kanbanzai-getting-started/SKILL.md` Anti-Patterns: "Shell-Querying Workflow State Files" entry naming prohibited directories and tools, mapping each query type to its MCP tool | ✅ | Entry present; names `cat`, `grep`, `find`; maps to `entity`, `status`, `knowledge`, `doc` |
| AC-010 | Target file (`write-research/SKILL.md`): two pre-writing checklist items (`retro synthesise`, `knowledge list`); "Report From Memory" anti-pattern | ✅ | Both checklist items and anti-pattern present in `write-research/SKILL.md` |
| AC-011 | `implement-task/SKILL.md` checklist: BUG entity item for intermittent failures, after AC-verification item, before `finish` | ✅ | Confirmed present |
| AC-012 | `implement-task/SKILL.md` Phase 4: intermittent failure defined; `entity(action: "create", type: "bug")` with all required fields; BUG ID in summary; no `finish` without BUG | ✅ | Full expansion present in Phase 4 |
| AC-013 | `implement-task/SKILL.md` Anti-Patterns: "Unreported Flaky Test" entry with detectable behaviour, cost explanation, and resolution directive | ✅ | Entry present |
| F-01 | **`internal/kbzinit/skills/getting-started/SKILL.md` NOT updated** | ❌ | The embedded counterpart installed by `kanbanzai init` is an older version that lacks the stash prohibition, "Commit workflow state" section, and "Shell-Querying" anti-pattern. New projects receive outdated guidance. See Conformance Gaps. |

**Summary:** All 13 live documentation changes are present and correct in the `.agents/skills/` and `.kbz/skills/` files. However, the embedded `internal/kbzinit/skills/getting-started/SKILL.md` was not updated. The dual-write rule was not yet formally documented when P24 was implemented (it was documented by P25), but the gap is real and affects new projects.

---

### FEAT-01KPPG5XMJWT3 — Optional GitHub PR Creation

Source recommendation: REC-09.

| # | Criterion | Result | Notes |
|---|-----------|--------|-------|
| AC-001 | `require_github_pr: true` → `RequireGitHubPR` is non-nil pointer to `true` | ✅ | `internal/config/config.go`; test verified |
| AC-002 | Field absent → `RequireGitHubPR` is `nil` | ✅ | `omitempty` YAML tag; test verified |
| AC-003 | Nil pointer → `RequiresGitHubPR()` returns `false` | ✅ | Accessor: `m.RequireGitHubPR != nil && *m.RequireGitHubPR` |
| AC-004 | Pointer to `false` → `RequiresGitHubPR()` returns `false` | ✅ | Same nil-safe accessor |
| AC-005 | Pointer to `true` → `RequiresGitHubPR()` returns `true` | ✅ | |
| AC-006 | Flag unset + no GitHub token → merge check passes without `pr_gate` key | ✅ | Integration test confirmed |
| AC-007 | `require_github_pr: true` + no PR found → `pr_gate.status == "failed"` | ✅ | Integration test confirmed |
| AC-008 | `require_github_pr: true` + non-open PR → `pr_gate.status == "failed"` with actual state | ✅ | Integration test confirmed |
| AC-009 | `require_github_pr: true` + no open PR → execute blocked | ✅ | Integration test confirmed |
| AC-010 | Config without field → no new gate failures (regression) | ✅ | NFR-001 preserved |
| AC-011 | Updated skill files describe both tracks | ✅ | `.agents/skills/kanbanzai-workflow/SKILL.md` and `kanbanzai-agents/SKILL.md` both contain two-track PR policy |
| F-02 | **`internal/kbzinit/skills/workflow/SKILL.md` NOT updated** | ❌ | Embedded workflow skill lacks the two-track PR policy note. New projects initialised with `kanbanzai init` will not receive the guidance. |
| F-03 | **`internal/kbzinit/skills/agents/SKILL.md` NOT updated** | ❌ | Embedded agents skill lacks the two-track Feature Completion procedure. Same consequence for new projects. |

**Summary:** The Go implementation (config field, accessor, gate enforcement) is complete and correctly tested. The live `.agents/skills/` documentation is correct. However, three `internal/kbzinit/skills/` files were not updated and are substantially diverged from their `.agents/skills/` counterparts.

---

## Documentation Currency

| Check | Result | Notes |
|-------|--------|-------|
| All spec documents in approved status | ✅ | All 5 specs approved |
| Live `.agents/skills/` files reflect P24 changes | ✅ | All targeted sections verified |
| Live `.kbz/skills/implement-task/SKILL.md` reflects P24 changes | ✅ | |
| Live `.kbz/skills/write-research/SKILL.md` reflects P24 changes | ✅ | |
| `AGENTS.md` reflects P24 changes | ✅ | |
| `internal/kbzinit/skills/getting-started/SKILL.md` in sync | ❌ | Stale — missing all P24 workflow hygiene changes (F-01) |
| `internal/kbzinit/skills/workflow/SKILL.md` in sync | ❌ | Stale — missing two-track PR policy (F-02) |
| `internal/kbzinit/skills/agents/SKILL.md` in sync | ❌ | Stale — missing two-track Feature Completion procedure (F-03) |
| P24 plan lifecycle advanced | ❌ | Plan status is `proposed`; should be `done` given all features complete. Lifecycle was never advanced. |

---

## Cross-Cutting Checks

| Check | Result | Notes |
|-------|--------|-------|
| `go test -race ./...` | ⚠️ Intermittent | Same pre-existing intermittent races in `internal/mcp` as noted in P25 review (BUG-01KPQY347TABX). Not introduced by P24. All other packages pass cleanly. |
| `health()` | ✅ Clean (plan scope) | Plan dashboard reports `errors: 0, warnings: 0`. Pre-existing health noise (stale branches, doc_currency for old plans) not attributable to P24. |
| `git status` | ✅ | Working tree clean, up to date with `origin/main`. |

---

## Conformance Gaps

| # | Severity | Category | Location | Description |
|---|----------|----------|----------|-------------|
| F-01 | Moderate | documentation | `internal/kbzinit/skills/getting-started/SKILL.md` | Embedded skill installed by `kanbanzai init` is a substantially older version. It lacks the stash prohibition (REC-02), the "Commit workflow state" section (REC-05), and the "Shell-Querying Workflow State Files" anti-pattern (REC-07). New projects receive pre-P24 guidance. |
| F-02 | Moderate | documentation | `internal/kbzinit/skills/workflow/SKILL.md` | Embedded workflow skill lacks the two-track PR policy note added by FEAT-01KPPG5XMJWT3 (REC-09). |
| F-03 | Moderate | documentation | `internal/kbzinit/skills/agents/SKILL.md` | Embedded agents skill lacks the two-track Feature Completion procedure added by FEAT-01KPPG5XMJWT3 (REC-09). |
| F-04 | Advisory | lifecycle | `P24-retro-recommendations` plan entity | Plan lifecycle state is `proposed` despite all 5 features and 17 tasks being done. The plan was never advanced through its lifecycle stages. Should be transitioned to `done`. |

**Note on F-01 through F-03:** The dual-write rule for embedded skills was not formally documented until P25 (FEAT-01KPQ08YKHNS9). P24 cannot be faulted for not following a rule that did not yet exist. However, the embedded files were already substantially diverged before P24 began — the drift appears to have accumulated across multiple earlier plans. P24 made changes to the live `.agents/skills/` files without addressing the pre-existing divergence. The practical consequence is real: `kanbanzai init` installs outdated guidance to new projects.

The recommended remediation is a small focused task: copy the current live `.agents/skills/kanbanzai-{getting-started,workflow,agents}/SKILL.md` files verbatim to their `internal/kbzinit/skills/{getting-started,workflow,agents}/SKILL.md` counterparts and commit. This should be tracked as a separate task rather than reopening P24.

---

## Verdict

**Pass with findings.**

P24 delivered all five features correctly. The Go code changes (AC pattern recognition, `PatchStatusField`, standalone bug surfacing, `RequireGitHubPR` gate) are complete, well-tested, and correctly integrated. All documentation changes are present and accurate in the live skill and reference files.

The three moderate findings (F-01, F-02, F-03) relate to `internal/kbzinit/` embedded skill files that were not updated alongside the live `.agents/skills/` changes. This is a consequence of the dual-write rule not being documented until P25, combined with pre-existing drift in the embedded files. The findings do not affect the current project (which uses the live `.agents/skills/` files) but do affect newly initialised projects.

**Recommended actions before closing:**
1. File a task to sync the three diverged `internal/kbzinit/skills/` files to match their live `.agents/skills/` counterparts (addresses F-01, F-02, F-03).
2. Advance the P24 plan lifecycle from `proposed` to `done` (addresses F-04).

Neither action requires reopening the plan's features. The plan is ready to close once the lifecycle is advanced and the embedded skill sync is tracked.