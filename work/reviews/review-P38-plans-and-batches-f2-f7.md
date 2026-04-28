# Review: P38 F2-F7 — Plans and Batches Implementation

**Date:** 2026-04-28
**Review cycle:** 1
**Reviewers dispatched:** reviewer-conformance, reviewer-quality, reviewer-testing
**Review units:** 3 (Go Implementation, Tests, Documentation)

---

## Per-Reviewer Summary

### Reviewer: reviewer-conformance
**Review unit:** p38-f2-f7-all (Go impl + tests + docs)
**Verdict:** needs_remediation
**Dimensions:**
  - spec_conformance: concern
    - F2: All 17 ACs verified — StrategicPlan struct, lifecycle, CRUD, cycle detection, deep nesting all pass
    - F3: Batch rename, B-prefix IDs, backward-compat aliases mostly pass — **but Batch struct missing `parent` field**
    - F4: B{n}-F{n} display IDs, four-level document gate verified — legacy P-prefix transitional gap noted
    - F5: ComputeBatchRollup, ComputePlanRollup verified with tests
    - F6: Entity tool, status dashboards verified — minor gaps in project overview, plan dashboard summary, estimate tool dispatch
    - F7: Documentation mostly updated — gaps in AGENTS.md, orchestrate-development skill, write-design skill, batch JSON key naming
**Findings:** 2 blocking, 10 non-blocking

### Reviewer: reviewer-quality
**Review unit:** p38-f2-f7-go-impl (all Go source)
**Verdict:** needs_remediation
**Dimensions:**
  - implementation_quality: concern
    - Error wrapping: consistent %w usage throughout
    - Naming consistency: thorough Batch/Plan renaming with deprecated aliases
    - Package cohesion: no import cycles, clean separation
    - **But:** silently discarded error in entityTransitionAction, loadBatch directory mismatch
**Findings:** 2 blocking, 4 non-blocking

### Reviewer: reviewer-testing
**Review unit:** p38-f2-f7-tests (all test files)
**Verdict:** approved_with_followups
**Dimensions:**
  - test_adequacy: pass_with_notes
    - All test functions assert specific outcomes — zero assertion-free tests
    - Strong isolation (t.Parallel() + t.TempDir() throughout)
    - Table-driven patterns used appropriately
    - Error paths broadly covered; prereq_test.go exemplary with 53 tests
    - Minor gaps: coexistence test doesn't verify list contents, missing strategic-plan status integration test, missing depth-limit test
**Findings:** 0 blocking, 8 non-blocking

---

## Collated Findings (deduplicated)

### Blocking

**[B-1] Batch struct missing `parent` field**
- Dimension: spec_conformance
- Location: `internal/model/entities.go:287-302`
- Spec ref: F3 REQ-003
- Description: The `Batch` struct lacks the required `parent` field (string, optional) for referencing a parent plan ID. Neither the internal model, nor `kbzschema.Batch` (types.go:94-106), nor service methods (CreateBatch/UpdateBatch), nor the entity tool's batch creation path support setting a batch's parent plan. This prevents batches from being placed under strategic plans, undermining the plan→batch hierarchy.
- Reported by: reviewer-conformance

**[B-2] `entitySvc.IncrementFeatureReviewCycle` error silently discarded**
- Dimension: implementation_quality
- Location: `internal/mcp/entity_tool.go:648-650`
- Description: The error return from `IncrementFeatureReviewCycle` is silently discarded in `entityTransitionAction`. If this call fails, the feature transitions to `reviewing` status but `review_cycle` is not incremented. On the next `reviewing→needs-rework→reviewing` cycle, the stale `review_cycle` value allows bypassing the review cycle cap check. The same call in `advance.go:224-229` correctly checks the error.
- Recommendation: Check the error return and return a failure response if it cannot be persisted, matching the pattern in `advance.go`.
- Reported by: reviewer-quality

**[B-3] `loadBatch` directory mismatch with `ListBatches` fallback**
- Dimension: implementation_quality
- Location: `internal/service/plans.go:148-153`, `internal/service/plans.go:262-274`, `internal/storage/entity_store.go:117-125`
- Description: `loadBatch` always loads from the "batches" directory (hardcoded), but `ListBatches` has a fallback that reads filenames from a "plans" directory when "batches" doesn't exist. The `loadBatch` fallback (`s.store.Load("plan", ...)`) also resolves to "batches" directory via `entityDirectory("plan") = "batches"`. Result: when "batches" directory does not exist, `ListBatches` silently returns empty results even though entries are visible in the "plans" directory.
- Recommendation: Either remove the "plans" directory fallback from `ListBatches` (if no legacy data uses it), or make `loadBatch` also try loading from the "plans" directory when the "batches" load fails.
- Reported by: reviewer-quality

### Non-Blocking

**[NB-1] Feature display ID transitional P-prefix for legacy batches**
- Dimension: spec_conformance
- Location: `internal/service/entities.go:216-217`
- Spec ref: F4 REQ-001
- Description: `CreateFeature` computes display ID using `model.ParsePlanID`, producing `P{n}-F{n}` for legacy P-prefix batches not yet migrated to B-prefix (F8 pending). New batches created with B-prefix work correctly. This is acceptable transitional behavior per F4 AC-005 but noted for completeness.
- Reported by: reviewer-conformance

**[NB-2] StrategicPlan.Order `omitempty` suppresses 0-value serialization**
- Dimension: spec_conformance
- Location: `internal/model/entities.go:334`
- Spec ref: F2 REQ-004
- Description: The `Order` field is tagged `yaml:"order,omitempty"`. Since the field type is `int` with default 0, the `omitempty` tag causes a value of 0 to be dropped from YAML output on round-trip. The spec says the default is 0 but doesn't prescribe omitempty behavior.
- Reported by: reviewer-conformance

**[NB-3] Status project overview doesn't list standalone batches**
- Dimension: spec_conformance
- Location: `internal/mcp/status_tool.go:320-321`
- Spec ref: F6 REQ-008
- Description: `synthesiseProject` calls `entitySvc.ListPlans` which delegates to `ListBatches`. The `Batches` field in `projectOverview` (L261-269) is declared but never populated with standalone batches. Standalone batches would appear in the `Plans` list but not in the `Batches` list.
- Reported by: reviewer-conformance

**[NB-4] Plan dashboard lacks child-entity summary string**
- Dimension: spec_conformance
- Location: `internal/mcp/status_tool.go:550-593`
- Spec ref: F6 AC-012
- Description: `synthesisePlanEntity` renders recursive progress but does not include a child-entity summary string like "3 batches, 2 plans — 65% complete". The `ComputePlanRollup` returns `BatchCount` and `PlanCount`, and the progress percentage is computed, but these are never combined into a summary string.
- Reported by: reviewer-conformance

**[NB-5] Estimate query doesn't handle "strategic-plan" entity type**
- Dimension: spec_conformance
- Location: `internal/mcp/estimate_tool.go:228-285`
- Spec ref: F5 AC-011, F6 AC-015
- Description: The `estimateQueryAction` switch handles `case "feature"` and `case "batch"` but not `case "strategic-plan"`. When `entityInferType` returns "strategic-plan" for a P-prefix ID, the entity type falls through to default. The "batch" case has fallback logic checking `prefix == "P"` to route to `ComputePlanRollup`, but this code path is unreachable because the entity type never reaches the "batch" case for strategic plans. Result: `estimate(action: "query", entity_id: "P1-test")` returns no rollup data.
- Reported by: reviewer-conformance

**[NB-6] No INFO log on legacy P{n}→B{n} resolution**
- Dimension: spec_conformance
- Location: `internal/service/plans.go:292-300`
- Spec ref: F3 REQ-011
- Description: The `loadBatch` function falls back from batches/ to plans/ directory when loading a batch, but does not emit any log line. The spec requires an INFO-level deprecation notice.
- Reported by: reviewer-conformance

**[NB-7] AGENTS.md lacks entity hierarchy and plan-vs-batch guidance**
- Dimension: spec_conformance
- Location: `AGENTS.md`
- Spec ref: F7 REQ-013, F7 REQ-014
- Description: AGENTS.md does not document the new entity hierarchy (Plan → Batch → Feature → Task) nor include guidance on when to create a plan vs a batch. The Key Terms table contains no entries for "Plan" or "Batch". The Repository Structure shows `.kbz/state/plans/` but not `.kbz/state/batches/`.
- Reported by: reviewer-conformance

**[NB-8] orchestrate-development skill uses "plan" instead of "batch"**
- Dimension: spec_conformance
- Location: `.kbz/skills/orchestrate-development/SKILL.md:119-131`
- Spec ref: F7 REQ-008
- Description: The skill says "Skip this phase entirely if the plan has 3 or fewer features" and "all features in the plan" — references to "plan" as a feature-owning work container should use "batch" terminology.
- Reported by: reviewer-conformance

**[NB-9] write-design skill doesn't document plan-level vs batch-level design ownership**
- Dimension: spec_conformance
- Location: `.kbz/skills/write-design/SKILL.md`
- Spec ref: F7 REQ-006
- Description: The write-design SKILL.md does not mention that design documents can be owned by either a plan (strategic design) or a batch (operational design).
- Reported by: reviewer-conformance

**[NB-10] Batch JSON dashboard key labeled "plan" instead of "batch"**
- Dimension: spec_conformance
- Location: `internal/mcp/status_tool.go:761`
- Spec ref: F6 REQ-007, F3 AC-014
- Description: The `batchDashboard` struct uses `Plan planHeader json:"plan"` — the JSON field name "plan" should use batch terminology to match the scope field value of "batch".
- Reported by: reviewer-conformance

**[NB-11] kbzschema.Batch missing fields**
- Dimension: spec_conformance
- Location: `kbzschema/types.go:94-106`
- Spec ref: F3 REQ-003
- Description: The exported `kbzschema.Batch` does not include `NextFeatureSeq`, and uses `Title` instead of `Name` as the field name. Also missing `CreatedBy` and `Updated`. The internal model has all fields but the exported schema is incomplete.
- Reported by: reviewer-conformance

**[NB-12] ListPlans doesn't propagate Parent filter to ListBatches**
- Dimension: implementation_quality
- Location: `internal/service/plans.go:179-183`
- Description: `ListPlans` does not propagate `BatchFilters.Parent` to `ListBatches`, despite `PlanFilters = BatchFilters` including the `Parent` field. No current callers pass `Parent` through `ListPlans`, but the contract gap could cause silent incorrect results.
- Recommendation: Propagate `filters.Parent` or document the limitation.
- Reported by: reviewer-quality

**[NB-13] entityCommitFunc errors silently swallowed without logging**
- Dimension: implementation_quality
- Location: `internal/mcp/entity_tool.go:103,187,516,540,692`
- Description: `entityCommitFunc` return values `(bool, error)` are discarded at all MCP tool call sites. A transient git failure that persists could result in many uncommitted state mutations.
- Recommendation: Log the commit error at minimum. Consider surfacing a warning in the tool response.
- Reported by: reviewer-quality

**[NB-14] entityTransitionAction uses deprecated ParseBatchID for strategic-plan**
- Dimension: implementation_quality
- Location: `internal/mcp/entity_tool.go:509`
- Description: The strategic-plan path uses `model.ParseBatchID` (deprecated alias for `ParsePlanID`). Functionally correct but reads oddly.
- Recommendation: Use `model.ParsePlanID` or add a comment noting intentional dual-use.
- Reported by: reviewer-quality

**[NB-15] Lifecycle interleaves Phase 1 and Phase 2 feature statuses**
- Dimension: implementation_quality
- Location: `internal/validate/lifecycle.go:131-198`
- Description: `allowedTransitions` for features interleaves Phase 1 statuses (Draft, InReview, Approved, InProgress) with Phase 2 statuses (Proposed, Designing, etc.), creating two parallel lifecycle entry points.
- Recommendation: Document the relationship between Phase 1 and Phase 2 feature lifecycles, or consider full deprecation of Phase 1 statuses.
- Reported by: reviewer-quality

**[NB-16] Coexistence test doesn't verify List results**
- Dimension: test_adequacy
- Location: `internal/service/strategic_plans_test.go:443-452`
- Description: `TestStrategicPlan_CoexistenceWithBatch` discards List results with `_ = batchPlans` and `_ = spPlans`. If `ListPlans` returned strategic plans or `ListStrategicPlans` returned batch plans, this test would not catch it.
- Recommendation: Assert that batchPlans contains P99-batch-plan (len ≥ 1) and spPlans contains P1-strategic-plan (len ≥ 1), and that the two lists are disjoint.
- Reported by: reviewer-testing

**[NB-17] No CreateStrategicPlan success-path-with-valid-parent test**
- Dimension: test_adequacy
- Location: `internal/service/strategic_plans_test.go:314-339`
- Description: Only the error path (nonexistent parent) is tested. No test verifies the success path where a strategic plan is created with a valid, existing parent.
- Reported by: reviewer-testing

**[NB-18] No UpdateStrategicPlan parent-change cycle detection via public API**
- Dimension: test_adequacy
- Location: `internal/service/strategic_plans_test.go:343-371`
- Description: Cycle detection is tested via the private `detectStrategicPlanCycle` directly, but the public `UpdateStrategicPlan` path that triggers it is not covered.
- Reported by: reviewer-testing

**[NB-19] No ComputePlanRollup depth-limit-exceeded test**
- Dimension: test_adequacy
- Location: `internal/service/estimation_rollup_test.go`
- Description: `maxPlanRollupDepth` is 50 but the defensive guard is untested.
- Reported by: reviewer-testing

**[NB-20] Status tool lacks strategic plan scope integration test**
- Dimension: test_adequacy
- Location: `internal/mcp/status_tool_test.go`
- Description: Tests cover feature, task, plan (batch), and project scopes, but no test validates the status tool response when given a strategic plan ID.
- Reported by: reviewer-testing

**[NB-21] SetEstimate doesn't test invalid entity type or non-existent entity**
- Dimension: test_adequacy
- Location: `internal/service/estimation_rollup_test.go:575-669`
- Description: Tests cover valid/invalid estimate values and soft-limit warnings, but not the error path where entityType is "nonexistent" or the entity ID doesn't resolve.
- Reported by: reviewer-testing

**[NB-22] GetEstimateFromFields string representation branch not unit-tested**
- Dimension: test_adequacy
- Location: `internal/service/estimation.go:135-150`
- Description: The string parsing path for estimate values is a YAML round-trip concern but has no direct test.
- Reported by: reviewer-testing

**[NB-23] EntityStore Write conflict path needs verification**
- Dimension: test_adequacy
- Location: `internal/storage/entity_store_test.go:1082-1128`
- Description: `TestEntityStore_Write_ReturnsErrConflictOnStaleFileHash` is listed in the outline but its body was not fully verified. Confirm it exists and asserts `ErrConflict` specifically.
- Reported by: reviewer-testing

---

## Aggregate Verdict: **rejected**

### Remediation Plan

1. **[B-1]** Add `parent` field to `Batch` struct, `kbzschema.Batch`, service methods, and entity tool → route to implementing agent
2. **[B-2]** Check error return from `IncrementFeatureReviewCycle` in `entityTransitionAction` → route to implementing agent
3. **[B-3]** Fix `loadBatch`/`ListBatches` directory mismatch — align fallback behavior → route to implementing agent

### Follow-up Items (non-blocking, 23 items)

See NB-1 through NB-23 above. Priority items:
- NB-5: Fix estimate tool "strategic-plan" dispatch (user-facing gap)
- NB-3/NB-4: Complete project overview and plan dashboard (user-facing gap)
- NB-7/NB-8/NB-9: Documentation gaps in AGENTS.md, skills, and roles
- NB-10: Fix JSON key naming in batch dashboard

---

## Review Unit Breakdown

| Unit | Files | Reviewers |
|------|-------|-----------|
| Go Implementation | `internal/model/entities.go`, `kbzschema/types.go`, `internal/validate/lifecycle.go`, `internal/service/plans.go`, `internal/service/strategic_plans.go`, `internal/service/entities.go`, `internal/service/estimation.go`, `internal/service/prereq.go`, `internal/mcp/entity_tool.go`, `internal/mcp/estimate_tool.go`, `internal/mcp/status_tool.go`, `internal/storage/entity_store.go`, `internal/health/entity_consistency.go` | conformance, quality |
| Tests | `internal/service/strategic_plans_test.go`, `internal/service/estimation_rollup_test.go`, `internal/service/display_id_test.go`, `internal/service/prereq_test.go`, `internal/service/plans_test.go`, `internal/mcp/status_tool_test.go`, `internal/storage/entity_store_test.go`, `internal/validate/doc_health_test.go` | conformance, testing |
| Documentation | `.kbz/roles/*.yaml`, `.kbz/skills/**/SKILL.md`, `.kbz/stage-bindings.yaml`, `AGENTS.md`, `.github/copilot-instructions.md`, `refs/*.md` | conformance |
