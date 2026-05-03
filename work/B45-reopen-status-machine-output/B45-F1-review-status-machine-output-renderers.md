# Review: Status Machine Output Renderers

**Feature:** FEAT-01KQPYQJH0KS5 (status-machine-output-renderers)
**Batch:** B45-reopen-status-machine-output
**Review cycle:** 1
**Date:** 2026-05-03

## Summary Verdict

**Overall: needs_remediation** — 1 blocking finding, 9 non-blocking findings.

The single blocking finding is a spec violation in the JSON task renderer: `parent_feature_id` is typed as `string` instead of `any`, causing tasks without a parent feature to serialize as `""` instead of JSON `null` (violates FR-8.3). The fix is straightforward — the same `any` pattern is already used correctly in `jsonBug.ParentFeatureID` and `jsonFeature.PlanID`.

All 11 acceptance criteria are traceable to implementation evidence. The implementation is well-structured, the test suite covers all six scope types with contract tests verifying required keys, and the code quality is solid with only minor consistency and documentation gaps.

## Per-Dimension Verdicts

| Dimension | Reviewer | Outcome |
|-----------|----------|---------|
| spec_conformance | reviewer-conformance | concern |
| implementation_quality | reviewer-quality | pass_with_notes |
| test_adequacy | reviewer-testing | pass_with_notes |

## Review Unit Breakdown

**Unit:** `status-renderers`
**Files:**
- `internal/cli/status/json.go` — JSON format renderer
- `internal/cli/status/plain.go` — Plain format renderer
- `internal/cli/status/json_test.go` — JSON renderer tests
- `internal/cli/status/plain_test.go` — Plain renderer tests

**Spec:** FEAT-01KQ2VHKJB5V8/spec-b36-f4-spec-status-machine-output (approved, inherited from B36-F4)
**Reviewers dispatched:** reviewer-conformance, reviewer-quality, reviewer-testing

## Blocking Findings

### B1: JSON task `parent_feature_id` serializes as `""` instead of `null`

- **Severity:** blocking
- **Spec anchor:** FR-8.3 ("Missing or null references MUST be JSON null")
- **Location:** `internal/cli/status/json.go:L81-L85`
- **Description:** `jsonTask.ParentFeatureID` is typed as `string` with json tag. When a task has no parent feature (empty string), it serializes as `"parent_feature_id": ""` instead of `"parent_feature_id": null`. The bug renderer (`jsonBug`) and feature renderer (`jsonFeature`) both use the `any` type pattern correctly — the task renderer does not.
- **Fix:** Change `jsonTask.ParentFeatureID` from `string` to `any`, and set to `nil` when the input is empty (same pattern as `jsonBug` at L200-L215). Add a test for null `parent_feature_id`.

## Non-Blocking Findings

### N1: No multi-severity attention sort test for plain renderer
- **Location:** `internal/cli/status/plain_test.go`
- **Spec anchor:** FR-3.3
- The `attentionFirst` helper sorts by severity, but no test verifies it selects the highest-severity item from multiple inputs. Add a test with `[{severity: "info"}, {severity: "error"}, {severity: "warning"}]` asserting the "error" message is output.

### N2: `features.done` approximation in plain.go RenderProject
- **Location:** `internal/cli/status/plain.go:L126-L129`
- The computation `featuresTotal - featuresActive` counts non-active features as done, conflating states like "designing" or "ready" with truly done features. The `ProjectPlanInput` type lacks a `FeaturesDone` field, so this is constrained by upstream data. Document the approximation or request a `FeaturesDone` field upstream.

### N3: `RenderDocument` hardcodes `attention: "none"` in plain renderer
- **Location:** `internal/cli/status/plain.go:L117`
- The JSON renderer dynamically generates attention for unregistered documents, but the plain renderer hardcodes `"none"` regardless. Consider surfacing an unregistered warning in plain output for parity.

### N4: `jsonTaskSummary.Ready` field never populated
- **Location:** `internal/cli/status/json.go:L52, L162-L165`
- The `Ready` field is declared but never set from `in.TasksReady`. Either populate it or remove the field.

### N5: `jsonProjectPlan.Slug` populated from `DisplayID`
- **Location:** `internal/cli/status/json.go:L212-L213`
- `ProjectPlanInput` lacks a `Slug` field, so `DisplayID` is used as a substitute. If these are semantically different, this is a data mismatch. The type may need a `Slug` field.

### N6: No boundary test for task counts with all-zero values
- **Location:** `internal/cli/status/json_test.go`
- FR-9.1 says "zero is valid" for task counts, but no test explicitly verifies zero renders as `0` rather than being omitted.

### N7: No non-empty attention test for JSON task/bug renderers
- **Location:** `internal/cli/status/json_test.go:L199-L269`
- Task and bug renderers always pass `nil` for attention. Add test cases with non-empty attention to verify rendering.

### N8: No non-nil attention test for plain bug renderer
- **Location:** `internal/cli/status/plain_test.go:L131-L158`
- Both bug tests pass `nil` attention. Add a test with an attention item to verify the message renders.

### N9: No regression test for nil attention → `[]` in JSON
- **Location:** `internal/cli/status/json_test.go`
- The feature summary mentions a previous nil-attention bug fix. Add an explicit regression test with `nil` attention input asserting `[]` output.

## Acceptance Criteria Traceability

| AC | Description | Evidence | Verdict |
|----|-------------|----------|---------|
| AC-1 | Plain feature with full documents | `TestPlainRenderer_RenderFeature_Full` | pass |
| AC-2 | Plain feature with no plan | `TestPlainRenderer_RenderFeature_NoPlan` | pass |
| AC-3 | Plain unregistered document | `TestPlainRenderer_RenderDocument_Unregistered` | pass |
| AC-4 | Plain project health gate | `TestPlainRenderer_RenderProject_HealthGate` | pass |
| AC-5 | JSON feature results array | `TestJSONRenderer_RenderFeature_Full` | pass |
| AC-6 | JSON feature null plan_id | `TestJSONRenderer_RenderFeature_NullPlanID` | pass |
| AC-7 | JSON unregistered document | `TestJSONRenderer_RenderDocument_Unregistered` | pass |
| AC-8 | JSON project overview shape | `TestJSONRenderer_RenderProject` | pass |
| AC-9 | JSON empty attention | `TestJSONRenderer_EmptyAttention` | pass |
| AC-10 | Exit codes | Renderers return errors for I/O/marshal failures — exit code logic is in CLI handler | pass |
| AC-11 | Schema contract tests | `TestPlainSchemaContract_RequiredKeys`, `TestJSONSchemaContract_RequiredFields` | pass |

## Remediation Plan

### Required (blocking)

1. **Fix `jsonTask.ParentFeatureID` type** — Change from `string` to `any` in `json.go:L84`. In `RenderTask`, set `pfid` to `nil` when parentFeature is empty (following the `jsonBug` pattern). Add test: `TestJSONRenderer_RenderTask_NullParentFeatureID`.

### Recommended (non-blocking)

2. Add multi-severity attention sort test for plain renderer (N1)
3. Populate or remove `Ready` field in `jsonTaskSummary` (N4)
4. Add regression test for nil attention → `[]` (N9)
5. Add non-empty attention tests for task/bug renderers (N7, N8)
