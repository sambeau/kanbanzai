| Field  | Value                                                      |
|--------|------------------------------------------------------------|
| Date   | 2026-04-23                                                 |
| Status | Draft                                                      |
| Author | orchestrator                                               |
| Plan   | P33-cohort-merge-checkpoints                               |

# Dev-Plan: Cohort-Based Merge Checkpoints (P33)

**Feature:** FEAT-01KPX-E8T8QWRB  
**Spec:** `work/spec/p33-cohort-merge-checkpoints.md`  
**Design:** `work/design/p33-cohort-merge-checkpoints.md`

---

## Overview

Three additive enhancements delivered across four tasks:

1. **T1** — Extend `BranchLookup` interface and `worktreeBranchLookup` with `GetBranchCreatedAt`
2. **T2** — Add feature-level conflict analysis to `internal/service/conflict.go` and wire into `internal/mcp/conflict_tool.go`
3. **T3** — Add `## Merge Schedule` section to the dev-plan template and cohort note to `write-dev-plan/SKILL.md`
4. **T4** — Add decompose merge-schedule warning to `internal/service/decompose.go` and update `orchestrate-development/SKILL.md`

T1 must complete before T2 (T2 depends on the new interface method). T3 and T4 are independent of T1/T2 and of each other — all three pairs (T1+T3, T1+T4, T3+T4) can run in parallel where the dependency graph permits.

**Execution order:**
- Batch 1: T1, T3 (parallel — disjoint file scopes)
- Batch 2: T2, T4 (parallel — T2 unblocks after T1; T4 has no dependency; disjoint file scopes)

---

## Interface Contract

**T1 → T2 contract:** T1 must add the following method to `service.BranchLookup` and implement it on `worktreeBranchLookup`:

```go
// GetBranchCreatedAt returns the UTC time at which the given branch was first
// created in the worktree store. Returns an error if no worktree record exists
// for the entity.
GetBranchCreatedAt(entityID string) (time.Time, error)
```

T2 must call `s.branchLookup.GetBranchCreatedAt(featureID)` and compute `drift_days` as
`int(time.Since(createdAt).Hours() / 24)` using UTC dates.

---

## Merge Schedule

| Cohort | Features | Gate condition |
|--------|----------|----------------|
| 1 | FEAT-01KPX-E8T8QWRB (T1, T3) | T1 and T3 reach done |
| 2 | FEAT-01KPX-E8T8QWRB (T2, T4) | T2 and T4 reach done — T2 requires T1 done |

_(Single feature; cohort structure applies to task dispatch batches within the feature.)_

---

## Task Breakdown

### T1 — Extend BranchLookup with GetBranchCreatedAt

**Objective:** Add `GetBranchCreatedAt(entityID string) (time.Time, error)` to the
`service.BranchLookup` interface and implement it on `worktreeBranchLookup` in
`internal/mcp/server.go`. The implementation reads the worktree record's `CreatedAt`
field (or equivalent timestamp) via `worktree.Store.GetByEntityID`.

**Spec references:** FR-006, NFR-002

**Input context:**
- `internal/service/conflict.go` — `BranchLookup` interface definition (lines 8–11)
- `internal/mcp/server.go` — `worktreeBranchLookup` struct and existing method implementations (lines 321–342)
- `internal/worktree/store.go` — worktree record structure; identify the field that records creation time
- `internal/worktree/` — check whether the worktree record YAML has a `created_at` or equivalent field

**Output artifacts:**
- Modified `internal/service/conflict.go` — new method on `BranchLookup` interface
- Modified `internal/mcp/server.go` — implementation of `GetBranchCreatedAt` on `worktreeBranchLookup`
- New or modified test in `internal/service/conflict_test.go` (or a new `_test.go` alongside) — mock implementation of `GetBranchCreatedAt` for use by T2's tests

**Dependencies:** None

**Verification:** `go test ./internal/service/... ./internal/mcp/...` passes. The new interface method compiles and the mock is usable by T2.

---

### T2 — Feature-level conflict analysis (service + MCP tool)

**Objective:** Add `feature_ids` mode to `ConflictService.Check` in
`internal/service/conflict.go` and wire it into the MCP tool in
`internal/mcp/conflict_tool.go`. When `feature_ids` is provided: resolve each
feature's tasks, aggregate `files_planned`, run pairwise `analyzePair` logic, compute
`drift_days` via `GetBranchCreatedAt`, and return a `FeatureConflictResult`. Supplying
both `task_ids` and `feature_ids` must return an error.

**Spec references:** FR-001, FR-002, FR-003, FR-004, FR-005, FR-006, FR-007, NFR-001, NFR-002, NFR-003, NFR-006

**Input context:**
- `internal/service/conflict.go` — full file; study `ConflictCheckInput`, `analyzePair`, `checkFileOverlap`, and helper types before writing anything
- `internal/mcp/conflict_tool.go` — full file; study the existing `conflictCheckAction` pattern
- `internal/mcp/server.go` — `worktreeBranchLookup` (updated by T1); understand how `branchLookup` is injected
- T1 output: `BranchLookup.GetBranchCreatedAt` interface method and mock
- `work/spec/p33-cohort-merge-checkpoints.md` §FR-001–007 and §AC-001–008

**New types to add in `internal/service/conflict.go`:**

```go
// FeatureConflictInfo holds the aggregated conflict data for one feature.
type FeatureConflictInfo struct {
    FeatureID   string
    FilesPlanned []string
    NoFileData   bool   // true when no tasks had files_planned populated
    DriftDays   *int   // nil when no worktree record exists
}

// FeatureConflictPairResult mirrors ConflictPairResult at the feature level.
type FeatureConflictPairResult struct {
    FeatureA       string
    FeatureB       string
    Risk           string
    Dimensions     ConflictDimensions
    Recommendation string
}

// FeatureConflictResult is returned when feature_ids mode is used.
type FeatureConflictResult struct {
    FeatureIDs  []string
    OverallRisk string
    Pairs       []FeatureConflictPairResult
    Features    []FeatureConflictInfo // per-feature metadata including drift_days and warnings
}
```

**Changes to `ConflictCheckInput`:**

```go
type ConflictCheckInput struct {
    TaskIDs    []string
    FeatureIDs []string
}
```

**Logic in `Check`:** if `len(input.FeatureIDs) > 0 && len(input.TaskIDs) > 0` → return error. If only `FeatureIDs` → call new `checkFeatures` method. Otherwise existing task path unchanged.

**`checkFeatures` method outline:**
1. For each feature ID: call `s.entitySvc.List("task")`, filter by `parent_feature == featureID`, aggregate `files_planned` into a set. If no files found, set `NoFileData: true`.
2. Compute `DriftDays` via `s.branchLookup.GetBranchCreatedAt(featureID)` — if error, leave nil.
3. Build `taskConflictInfo` structs with the aggregated file sets (re-use existing type; set `id = featureID`, `filesPlanned = aggregated`).
4. Run `s.analyzePair` for all pairs — this reuses existing logic exactly.
5. Assemble and return `FeatureConflictResult`.

**MCP tool changes in `internal/mcp/conflict_tool.go`:**
- Add `feature_ids` as an optional (non-required) array parameter alongside `task_ids`
- Make `task_ids` no longer `Required()` — both are optional at the schema level; validation happens in the handler
- In `conflictCheckAction`: extract both arrays; if both present → `inlineErr`; if `feature_ids` present → call `Check` with `FeatureIDs`; marshal `FeatureConflictResult` into the response
- Update tool description to document `feature_ids` mode and mutual exclusivity
- Response shape for feature mode: `{ "feature_ids": [...], "overall_risk": "...", "pairs": [...], "features": [...] }`

**Output artifacts:**
- Modified `internal/service/conflict.go`
- Modified `internal/mcp/conflict_tool.go`
- New tests in `internal/service/conflict_feature_test.go` covering AC-001 through AC-008

**Dependencies:** T1 (requires `GetBranchCreatedAt` on `BranchLookup`)

**Verification:** `go test ./internal/service/... ./internal/mcp/...` passes. All pre-existing conflict tests pass unmodified (NFR-001/AC-016).

---

### T3 — Dev-plan template and write-dev-plan skill update

**Objective:** Add a `## Merge Schedule` section to
`work/templates/implementation-plan-prompt-template.md` and add a cohort-authoring
note to `.kbz/skills/write-dev-plan/SKILL.md`. Pure documentation — no Go changes.

**Spec references:** FR-008, FR-009, NFR-005

**Input context:**
- `work/templates/implementation-plan-prompt-template.md` — full file; identify where the new section fits (after the task-breakdown section)
- `.kbz/skills/write-dev-plan/SKILL.md` — full file; identify the quality-checks step or checklist section where the note belongs
- `work/spec/p33-cohort-merge-checkpoints.md` §FR-008, §FR-009, §AC-009, §AC-010

**Template change:** Add the following section to `implementation-plan-prompt-template.md`
after the existing task-breakdown content:

```markdown
## Merge Schedule

_Required when the plan has more than 3 features. Omit for plans with 3 or fewer features._

Group features into cohorts of 3–5. Features within a cohort must have no file-scope
overlap (verify with `conflict(action: "check", feature_ids: [...])`). All cohort-N
features must be merged to main before cohort-N+1 work begins.

| Cohort | Features | Gate condition |
|--------|----------|----------------|
| 1      |          |                |
| 2      |          |                |
```

**Skill change:** In `.kbz/skills/write-dev-plan/SKILL.md`, locate the quality-checks
checklist or equivalent section and add:

```
- [ ] If the plan has more than 3 features: add a `## Merge Schedule` section grouping
      features into cohorts of 3–5. Use `conflict(action: "check", feature_ids: [...])`
      to verify no intra-cohort file overlap before publishing the dev-plan.
```

**Output artifacts:**
- Modified `work/templates/implementation-plan-prompt-template.md`
- Modified `.kbz/skills/write-dev-plan/SKILL.md`

**Dependencies:** None

**Verification:** Read both files after editing and confirm the new content is present at the correct location. No Go build required.

---

### T4 — Decompose warning and orchestrate-development skill update

**Objective:** Add a non-blocking merge-schedule warning to
`internal/service/decompose.go` (emitted during `DecomposeFeature` when the plan has
more than 3 features and the dev-plan has no `## Merge Schedule` heading), and add
Phase 0 + merge-checkpoint step to `.kbz/skills/orchestrate-development/SKILL.md`.

**Spec references:** FR-010, FR-011, FR-012, NFR-004, NFR-005

**Input context:**
- `internal/service/decompose.go` — full file; study `DecomposeFeature`, `Proposal.Warnings`, and the existing warning-append pattern (lines ~144–245 for the dev-plan path; lines ~708–713 for the `maxTasksPerFeature` warning pattern)
- `.kbz/skills/orchestrate-development/SKILL.md` — full file (already read); Phase 1 begins at the `### Phase 1: Read the Dev-Plan` heading; insert Phase 0 before it; Phase 6 Close-Out is the merge-checkpoint target
- `work/spec/p33-cohort-merge-checkpoints.md` §FR-010–012, §AC-011–015

**Decompose warning logic:**

In `DecomposeFeature`, after the dev-plan is resolved (or falls back) but before returning
the proposal, add:

```go
// Merge schedule check: warn when plan has >3 features and dev-plan lacks ## Merge Schedule.
if parentPlan := stringFromState(feat.State, "parent"); parentPlan != "" {
    feats, _ := s.entitySvc.List("feature")
    planFeatureCount := 0
    for _, f := range feats {
        if stringFromState(f.State, "parent") == parentPlan {
            planFeatureCount++
        }
    }
    if planFeatureCount > 3 {
        hasMergeSchedule := strings.Contains(devPlanContent, "## Merge Schedule")
        if !hasMergeSchedule {
            proposal.Warnings = append(proposal.Warnings, fmt.Sprintf(
                "plan has %d features but dev-plan has no ## Merge Schedule section; "+
                    "consider adding cohort groupings to prevent worktree drift",
                planFeatureCount,
            ))
        }
    }
}
```

Note: `devPlanContent` is the string already fetched from the dev-plan document earlier
in `DecomposeFeature`. If no dev-plan was found, skip the check (the existing
"dev-plan not found" warning already covers that case). The warning must be appended to
`proposal.Warnings` — it must not return an error or prevent the proposal from being
returned (NFR-004).

**Skill change for `orchestrate-development/SKILL.md`:**

Insert the following **before** the `### Phase 1: Read the Dev-Plan` heading:

```markdown
### Phase 0: Cohort Setup _(plans with more than 3 features only)_

Skip this phase entirely if the plan has 3 or fewer features.

1. Read the dev-plan's `## Merge Schedule` block. If a merge schedule is present, treat
   its cohort groupings as authoritative — record them and proceed to Phase 1 for
   cohort-1 features only.
2. If no merge schedule exists, call `conflict(action: "check", feature_ids: [...])` for
   all features in the plan to identify file-scope overlap.
3. Group features into cohorts based on the results: features with no file overlap may be
   parallelised (same cohort); overlapping features must be serialised (different cohorts).
   Target cohort size: 3–5 features.
4. Record the cohort plan in the session: "Cohort 1: FEAT-A, FEAT-B. Cohort 2: FEAT-C,
   FEAT-D, FEAT-E."
5. Create worktrees only for cohort-1 features. Do not create worktrees for later cohorts
   until the preceding cohort's merge checkpoint is confirmed clean.
```

In the **Phase 6: Close-Out** section, after step 6 ("Verify branch is gone"), add:

```markdown
7. **Cohort checkpoint.** If this plan has a merge schedule with multiple cohorts, check
   whether cohort-N+1 features exist. If so, return to Phase 0 for cohort N+1. Do not
   create cohort-N+1 worktrees until the cohort-N merge checkpoint is confirmed clean:
   no open feature branches from cohort N remain (`git branch | grep FEAT` returns only
   cohort-N+1 or later branches).
```

**Output artifacts:**
- Modified `internal/service/decompose.go`
- Modified `.kbz/skills/orchestrate-development/SKILL.md`

**Dependencies:** None (independent of T1 and T2)

**Verification:** `go test ./internal/service/...` passes. Read the skill file and confirm Phase 0 is present before Phase 1 and step 7 is present in Phase 6.

---

## File Ownership (conflict prevention)

| Task | Files owned — must not be touched by other tasks |
|------|--------------------------------------------------|
| T1 | `internal/service/conflict.go` (interface only), `internal/mcp/server.go` |
| T2 | `internal/service/conflict.go` (implementation), `internal/mcp/conflict_tool.go`, `internal/service/conflict_feature_test.go` |
| T3 | `work/templates/implementation-plan-prompt-template.md`, `.kbz/skills/write-dev-plan/SKILL.md` |
| T4 | `internal/service/decompose.go`, `.kbz/skills/orchestrate-development/SKILL.md` |

T1 and T2 both touch `internal/service/conflict.go` — **T1 must complete before T2 starts.** T1's scope is the interface declaration and the `worktreeBranchLookup` implementation only; T2 owns all new service logic and the MCP handler.

---

## Dependency Graph

```
T1 (BranchLookup extension)
 └── T2 (feature-level conflict service + MCP tool)

T3 (template + write-dev-plan skill)   [independent]

T4 (decompose warning + orchestrate-development skill)   [independent]
```

Optimal execution: dispatch T1 and T3 in parallel (Batch 1). When T1 is done, dispatch T2 alongside T4 (Batch 2, both parallel).

---

## Acceptance Criteria Cross-Reference

| AC | Covered by |
|----|-----------|
| AC-001 | T2 (mutual exclusivity error) |
| AC-002 | T2 (feature_ids mode returns feature result) |
| AC-003 | T2 (overlapping files → non-safe risk) |
| AC-004 | T2 (no overlap → safe_to_parallelise) |
| AC-005 | T2 (no_file_data warning) |
| AC-006 | T1 + T2 (drift_days populated from GetBranchCreatedAt) |
| AC-007 | T1 + T2 (drift_days absent when no worktree) |
| AC-008 | T2 (FeatureConflictResult structure in response) |
| AC-009 | T3 (template has Merge Schedule section) |
| AC-010 | T3 (write-dev-plan skill has cohort note) |
| AC-011 | T4 (warning emitted when >3 features, no heading) |
| AC-012 | T4 (no warning when ≤3 features) |
| AC-013 | T4 (no warning when heading present) |
| AC-014 | T4 (Phase 0 with all 5 steps in orchestrate-development) |
| AC-015 | T4 (checkpoint step in Phase 6) |
| AC-016 | T2 (existing task-level tests pass unmodified) |
| AC-017 | T2 (unit tests: aggregation, no_file_data, drift_days, mutual exclusivity) |
```

Now I have the dev-plan content. Let me save it, register it, approve it, and create the tasks: