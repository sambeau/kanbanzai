# Plan Review: P37 — File Names and Actions

| Field    | Value                             |
|----------|-----------------------------------|
| Plan     | P37-file-names-and-actions        |
| Reviewer | Claude Sonnet 4.6 (reviewer-conformance) |
| Date     | 2026-04-27T23:25:00Z              |
| Verdict  | **Fail** — conformance gaps block completion |

---

## Feature Census

| Feature        | Slug                              | Status     | Terminal | Notes |
|----------------|-----------------------------------|------------|----------|-------|
| FEAT-01KQ7-JDSVMP4E | plan-scoped-feature-display-ids | done    | ✅       | 6/6 tasks done |
| FEAT-01KQ7-JDSZARPC | doc-type-and-filename-enforcement | done  | ✅       | 5/5 tasks done |
| FEAT-01KQ7-JDT11MH6 | kbz-move                        | reviewing  | ❌       | **Conformance gap** — no review report registered; Mode 2 REQ-019 non-conformance |
| FEAT-01KQ7-JDT341E8 | kbz-delete                      | done       | ✅       | 3/3 tasks done; spec in draft |
| FEAT-01KQ7-JDT511BZ | work-tree-migration              | developing | ❌       | 5/5 tasks queued; 0 started; blocked on F3 merge |

**Plan is not in terminal state.** Two features (F3, F5) remain non-terminal. Per the review-plan skill, this is a conformance gap and the plan cannot be marked complete.

---

## Specification Approval

| Feature             | Spec Document                                       | Status          |
|---------------------|-----------------------------------------------------|-----------------|
| FEAT-01KQ7-JDSVMP4E | work/design/p37-f1-spec-plan-scoped-feature-display-ids.md | approved ✅ |
| FEAT-01KQ7-JDSZARPC | work/design/p37-f2-spec-doc-type-and-filename-enforcement.md | approved ✅ |
| FEAT-01KQ7-JDT11MH6 | work/design/p37-f3-spec-kbz-move.md                | approved ✅    |
| FEAT-01KQ7-JDT341E8 | work/design/p37-f4-spec-kbz-delete.md              | **draft ❌**   |
| FEAT-01KQ7-JDT511BZ | work/design/p37-f5-spec-work-tree-migration.md     | approved ✅    |

**F4 spec is unapproved.** The feature is in `done` status but its specification document was never approved. This is a documentation currency gap.

---

## Spec Conformance Detail

### F1 — Plan-scoped feature display IDs (FEAT-01KQ7-JDSVMP4E)

20 acceptance criteria. All 20 `TestDisplayID_AC0xx_*` tests pass on `main`.

| # | Criterion | Result | Notes |
|---|-----------|--------|-------|
| AC-001 | CreatePlan initialises `next_feature_seq: 1` | ✅ | `TestDisplayID_AC001_CreatePlanInitialisesSeq` |
| AC-002 | CreateFeature increments counter | ✅ | `TestDisplayID_AC002_CreateFeatureIncrementsCounter` |
| AC-003 | Feature display ID format is `P{n}-F{m}` | ✅ | `TestDisplayID_AC003_FeatureDisplayIDFormat` |
| AC-004 | Fault after plan write leaves no feature file | ✅ | `TestDisplayID_AC004_FaultAfterPlanWriteNoFeatureFile` |
| AC-005 | CreateFeature without parent returns error | ✅ | `TestDisplayID_AC005_CreateFeatureRequiresParent` |
| AC-006 | Both writes observable before return | ✅ | `TestDisplayID_AC006_FourStepSequenceObservable` |
| AC-007 | `entity get P24-F3` resolves to canonical | ✅ | `TestDisplayID_AC007_GetByDisplayID` |
| AC-008 | Case-insensitive resolution | ✅ | `TestDisplayID_AC008_GetCaseInsensitive` |
| AC-009 | `entity get` accepts display ID | ✅ | `TestDisplayID_AC009_EntityGetAcceptsDisplayID` |
| AC-010 | `entity update` accepts display ID | ✅ | `TestDisplayID_AC010_UpdateEntityAcceptsDisplayID` |
| AC-011 | `entity transition` accepts display ID | ✅ | `TestDisplayID_AC011_UpdateStatusAcceptsDisplayID` |
| AC-012 | `entity list` filter by display ID | ✅ | `TestDisplayID_AC012_ListFilterByDisplayID` |
| AC-013 | MCP response includes `display_id` field | ✅ | `TestDisplayID_AC013_FeatureFieldsIncludesDisplayID` |
| AC-014 | CLI displays `P{n}-F{m}` as primary identifier | ✅ | `TestDisplayID_AC014_IsFeatureDisplayID` |
| AC-015 | Migration assigns display IDs in creation-timestamp order | ✅ | `TestDisplayID_AC015_MigrationAssignsInCreatedOrder` |
| AC-016 | Migration sets `next_feature_seq` correctly | ✅ | `TestDisplayID_AC016_MigrationSetsPlanCounter` |
| AC-017 | Resolution performance ≤ 100 ms on 1,000-feature fixture | ✅ | `TestDisplayID_AC017_ResolutionPerformance` |
| AC-018 | Canonical TSID still resolves | ✅ | `TestDisplayID_AC018_CanonicalIDStillWorks` |
| AC-019 | Break-hyphen TSID form resolves | ✅ | `TestDisplayID_AC019_BreakHyphenStillWorks` |
| AC-020 | Migration preserves feature filenames | ✅ | `TestDisplayID_AC020_MigrationPreservesFilenames` |

**F1 verdict: Pass** — all 20 ACs verified by named tests.

---

### F2 — Document type system and filename enforcement (FEAT-01KQ7-JDSZARPC)

23 acceptance criteria. All pass on `main`.

| # | Criterion | Result | Notes |
|---|-----------|--------|-------|
| AC-001 | `type: review` accepted and stored | ✅ | `TestSubmitDocument_NewDocumentTypes/review` |
| AC-002 | `type: proposal` accepted and stored | ✅ | `TestSubmitDocument_NewDocumentTypes/proposal` |
| AC-003 | Unknown type error lists 8 user-facing types, excludes `policy`/`rca` | ✅ | `TestSubmitDocument_InvalidType` |
| AC-004 | `type: specification` normalised to `spec` | ✅ | `TestSubmitDocument_NewDocumentTypes/specification` |
| AC-005 | `type: retrospective` normalised to `retro` | ✅ | `TestSubmitDocument_NewDocumentTypes/retrospective` |
| AC-006 | `type: policy` accepted (internal-only) | ✅ | Covered by new-type test |
| AC-007 | `type: rca` accepted (internal-only) | ✅ | Covered by new-type test |
| AC-008 | Unknown type error excludes `policy`/`rca` from message | ✅ | `TestSubmitDocument_InvalidType` |
| AC-009 | Valid plan-prefixed filename passes validation | ✅ | `TestValidateDocumentFilename_PlanFolder` |
| AC-010 | Missing plan-ID prefix fails validation | ✅ | `TestValidateDocumentFilename_PlanFolder` |
| AC-011 | Feature-scoped filename (`P37-F2-spec-…`) passes | ✅ | `TestValidateDocumentFilename_PlanFolder` |
| AC-012 | Case-insensitive filename prefix match | ✅ | `TestValidateDocumentFilename_PlanFolder` |
| AC-013 | Plan-ID mismatch between folder and filename fails | ✅ | `TestValidateDocumentFolder_PlanIDMustMatchFolder` |
| AC-014 | `work/_project/` exempt from folder validation | ✅ | `TestValidateDocumentFolder_TypePrefixMustBeInProject` |
| AC-015 | Validation error contains expected path hint | ✅ | Error message validation in folder tests |
| AC-016 | `work/templates/` exempt from all validation | ✅ | `TestValidateDocumentFilename_TemplatesExempt` / `TestValidateDocumentFolder_TemplatesExempt` |
| AC-017 | `docs/` path exempt from folder validation | ✅ | `TestValidateDocumentFilename_DocsExempt` / `TestValidateDocumentFolder_DocsExempt` |
| AC-018 | Legacy paths load without error | ✅ | `TestValidateDocument_Valid` |
| AC-019 | Deserialise `type: specification` → `spec` | ✅ | `TestSubmitDocument_NewDocumentTypes/specification` |
| AC-020 | Deserialise `type: retrospective` → `retro` | ✅ | `TestSubmitDocument_NewDocumentTypes/retrospective` |
| AC-021 | Deserialise `type: plan` succeeds | ✅ | Covered by model-level type constant test |
| AC-022 | No exec/network/config calls in validation path | ✅ | Code review: validation is pure in-process string logic |
| AC-023 | Folder error message includes specific expected directory | ✅ | `TestValidateDocumentFolder_PlanIDMustMatchFolder` |

**F2 verdict: Pass** — all 23 ACs verified by named tests.

---

### F3 — kbz move command (FEAT-01KQ7-JDT11MH6)

18 acceptance criteria. All 18 tests pass. However, two ACs have a gap between the test passing and the spec requirement actually being satisfied.

#### Mode 1 (file move)

| # | Criterion | Result | Notes |
|---|-----------|--------|-------|
| AC-001 | Slash in path selects Mode 1 | ✅ | `TestModeDetection_SlashInPath` |
| AC-002 | `.md` extension selects Mode 1 | ✅ | `TestModeDetection_DotMdExtension` |
| AC-003 | Non-`work/` path rejected | ✅ | `TestRejectOutsideWorkDir` |
| AC-004 | Missing source file error | ✅ | `TestFileMoveSourceNotFound` |
| AC-005 | Unknown target plan error | ✅ | `TestFileMoveTargetPlanNotFound` |
| AC-006 | Registered doc moved; record updated; git history intact; stdout correct | ✅ | `TestFileMoveRegisteredDocument` |
| AC-007 | Plan folder created if absent | ✅ | `TestFileMoveCreatesPlanFolder` |
| AC-008 | Target already exists rejected | ✅ | `TestFileMoveTargetAlreadyExists` |
| AC-009 | Unregistered file: moves with warning | ✅ | `TestFileMoveUnregisteredFile` |
| AC-010 | `os.Rename` absent; `GitMove` is sole mechanism | ✅ | `TestFileMoveNoOsRename` (source inspection) |

#### Mode 2 (feature re-parent)

| # | Criterion | Result | Notes |
|---|-----------|--------|-------|
| AC-011 | `P{n}-F{m}` pattern selects Mode 2 | ✅ | `TestModeDetection_FeatureDisplayID` |
| AC-012 | Unknown display ID error | ✅ | `TestReParentFeatureNotFound` |
| AC-013 | Re-parent to same plan rejected | ✅ | `TestReParentSamePlan` |
| AC-014 | Confirmation prompt shown; decline aborts cleanly | ✅ | `TestReParentConfirmationPrompt` |
| AC-015 | `--force` skips prompt | ✅ | `TestReParentForceFlag` |
| AC-016 | Feature `parent` and `display_id` updated; plan `next_feature_seq` incremented | ⚠️ | `TestReParentEntityUpdate` verifies `parent` field only; `display_id` and `next_feature_seq` are **not asserted** |
| AC-017 | Documents moved via `git mv`; `path` and `owner` updated to target plan | ❌ | `TestReParentDocumentMoves` verifies file presence and `git log --follow` but **does not assert `owner` field**. Implementation sets `owner = canonicalID` (feature ID); spec REQ-019(c) requires `owner = targetPlanID` |
| AC-018 | Output contains `Moved feature P{n}-F{m} → P{n}-F{m}` and per-document lines | ✅ | `TestReParentOutputSummary` |

**F3 verdict: Fail** — AC-017 non-conformance; AC-016 test coverage gap.

---

### F4 — kbz delete command (FEAT-01KQ7-JDT341E8)

17 acceptance criteria. All 17 tests pass (verified in gate-override record). Feature is `done`.

Conformance check is not fully performed because the specification is still in `draft` status. The implementation is assumed complete based on the gate-override attestation ("all 17 AC tests pass"), but the unapproved spec is a documentation gap.

**F4 verdict: Pass with findings** — implementation attested passing; spec document must be approved.

---

### F5 — Work tree migration (FEAT-01KQ7-JDT511BZ)

Not reviewed. Feature is in `developing` state with 0/5 tasks started. Blocked on F3 merge. Will be reviewed when F5 reaches `reviewing`.

---

## Documentation Currency

| Check                                        | Result   | Notes |
|----------------------------------------------|----------|-------|
| Specs approved for all done features         | ❌       | F4 spec in `draft` |
| No review report registered for F3           | ❌       | Feature is in `reviewing` with no registered review document (attention item surfaced by `status()`) |
| AGENTS.md Scope Guard references P37         | N/A      | P37 is not yet done; no update required until completion |
| SKILL files current                          | N/A      | P37 does not add or modify SKILL files |
| Stray artifact in F3 worktree               | ❌       | `cmd/kanbanzai/move_force_fix.patch` is a committed placeholder file; not source code, not a test, no functional role |

---

## Cross-Cutting Checks

| Check | Result | Notes |
|-------|--------|-------|
| `go test -race ./...` on `main` | ⚠️ Pre-existing failures | `TestDocIntelFind_Role_*` (5 tests in `internal/mcp`) fail on both `main` and F3 branch — confirmed pre-existing before P37. All P37-relevant packages (`internal/service`, `internal/git`, `cmd/kanbanzai`) pass cleanly. |
| `go test -race ./...` on F3 branch | ⚠️ Pre-existing failures | Same `TestDocIntelFind_Role_*` failures; no new failures introduced by F3. All 18 move-command tests pass. |
| `health()` | ⚠️ Pre-existing warnings | Errors: 0. Warnings include stale worktree records, branch drift, overdue cleanup, knowledge TTL — all pre-existing. No new health issues introduced by P37. |
| `git status` clean | ✅ | Orphaned `.kbz/` index files committed at review start. Working tree clean. |
| F3 PR open on GitHub | ❌ | Branch `feature/FEAT-01KQ7JDT11MH6-kbz-move` exists but no PR has been opened. Review cannot proceed to merge without an open PR. |

---

## Conformance Gaps

| # | Severity | Category | Location | Description |
|---|----------|----------|----------|-------------|
| CG-001 | **Blocking** | feature-status | FEAT-01KQ7-JDT11MH6 | F3 is in `reviewing` — non-terminal. Plan cannot complete until F3 advances to `done`. |
| CG-002 | **Blocking** | feature-status | FEAT-01KQ7-JDT511BZ | F5 is in `developing` — non-terminal. Plan cannot complete until F5 advances to `done`. |
| CG-003 | **Blocking** | spec-conformance | `cmd/kanbanzai/move_cmd.go:241` | Mode 2 (`runMoveFeature`) calls `UpdateDocumentPathAndOwner(doc.ID, newPath, canonicalID)` — the owner argument is the feature's canonical ID, but REQ-019(c) and AC-017 require the owner to be updated to the **target plan ID**. The document record's `owner` field is not updated as specified. Note: this may indicate a spec defect — if documents should remain feature-owned after re-parent, REQ-019(c) is mis-stated. Human adjudication required. |
| CG-004 | **Blocking** | test-coverage | `cmd/kanbanzai/move_cmd_test.go` `TestReParentDocumentMoves` | AC-017 test does not assert the `owner` field of the moved document record. The passing test gives false confidence — the conformance failure in CG-003 is undetected. A passing test that fails to exercise the specified behaviour is a blocking finding. |
| CG-005 | **Blocking** | test-coverage | `cmd/kanbanzai/move_cmd_test.go` `TestReParentEntityUpdate` | AC-016 requires verification of three outcomes: (1) `parent` updated, (2) `display_id` set to new ID, (3) plan `next_feature_seq` incremented. The test only asserts (1). Outcomes (2) and (3) are unverified. |
| CG-006 | **Blocking** | documentation | `work/design/p37-f4-spec-kbz-delete.md` | F4 specification remains in `draft` status. Feature is `done` with all 17 ACs attested passing, but the spec was never approved. All feature specs must be in `approved` status before their feature is eligible for plan-level review. |
| CG-007 | **Blocking** | documentation | FEAT-01KQ7-JDT11MH6 | No review report registered for F3. The feature is in `reviewing` state and the `status()` attention item confirms no review report exists. A review report must be registered and approved before F3 can advance to `done`. |
| CG-008 | **Blocking** | documentation | FEAT-01KQ7-JDT11MH6 | No GitHub PR has been opened for F3. Branch `feature/FEAT-01KQ7JDT11MH6-kbz-move` is pushed but `pr(action: "create")` has not been called. The merge gate requires an open PR. |
| CG-009 | Advisory | artifact | `cmd/kanbanzai/move_force_fix.patch` | A placeholder file (`content: "placeholder"`) was committed to the F3 branch. It has no functional role. Should be removed before merge to keep the working tree clean. |

---

## Retrospective Observations

**What worked well:**

- F1 and F2 both delivered comprehensive, AC-named test suites (`TestDisplayID_AC0xx_*`, `TestValidateDocument*`) that map directly to spec criteria. Traceability from spec to test is excellent.
- The gate-override records for P37 are thorough and include specific test counts and rationale. Future reviewers can follow the override history without ambiguity.
- The decision to keep `kbz delete` and `kbz move` as CLI-only commands with no new MCP surface was well-scoped and enforced consistently.
- `GitMove` was correctly isolated in `internal/git/` following the `runGitCmd` pattern, satisfying REQ-NF-002 cleanly.

**What caused friction:**

- The Mode 2 owner semantics expose an ambiguity in the spec: REQ-019(c) says update owner to the target plan ID, but the system's document model uses feature IDs as owners for feature-level documents. This tension was not surfaced during the specifying stage. A design note or constraint clarifying intended ownership semantics post-re-parent would have prevented this gap.
- The `TestReParentDocumentMoves` test verifies file movement and git history (the hard part) but omits the record field verification (the straightforward part). The pattern used in F1/F2 — where each test is named after its AC and asserts the full AC — was not applied consistently to F3 Mode 2 tests.
- F4's spec was never approved despite the feature reaching `done`. The spec approval gate was bypassed with a `specifying→dev-planning` override but the approval itself was never actioned. This gap survived the full development and merge cycle undetected.

---

## Verdict

**Fail.** The plan has eight blocking conformance gaps.

P37 cannot advance to `done` in its current state. The required resolution path is:

1. **Resolve CG-003 / CG-009** — Clarify with the human whether REQ-019(c) is correct (owner → plan ID) or a spec defect (owner should remain the feature canonical ID). Update either the implementation or the spec accordingly.
2. **Fix CG-004** — Add `owner` field assertion to `TestReParentDocumentMoves`.
3. **Fix CG-005** — Add `display_id` and `next_feature_seq` assertions to `TestReParentEntityUpdate`.
4. **Fix CG-009** — Delete `cmd/kanbanzai/move_force_fix.patch` from the F3 branch.
5. **Open the F3 PR on GitHub** (CG-008) — run `pr(action: "create", entity_id: "FEAT-01KQ7JDT11MH6")`.
6. **Register and approve this review report** as a document record on F3 (CG-007) — this review document serves as the F3 review report.
7. **Approve the F4 spec** (CG-006) — `doc(action: "approve", id: "FEAT-01KQ7JDT341E8/specification-p37-f4-spec-kbz-delete")`.
8. **Merge F3** after CG-001 through CG-009 are resolved, which unblocks F5 (CG-002).
9. **Complete F5** — once F3 is merged, transition F5 tasks from queued to ready and execute the migration work.
10. **Re-run this plan review** after F3 and F5 reach `done`.

Pre-existing `TestDocIntelFind_Role_*` failures in `internal/mcp` are noted but are not P37 findings — they pre-date this plan and require a separate fix.