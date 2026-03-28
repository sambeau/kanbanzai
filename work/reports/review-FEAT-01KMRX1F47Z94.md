# Review Report: FEAT-01KMRX1F47Z94 (review-lifecycle-states)

| Field          | Value                                                    |
|----------------|----------------------------------------------------------|
| Feature        | FEAT-01KMRX1F47Z94 (review-lifecycle-states)             |
| Status         | changes_required                                         |
| Reviewer       | Orchestrated review (2 parallel sub-agents)              |
| Date           | 2026-03-28                                               |
| Spec           | FEAT-01KMRX1F47Z94/specification-review-lifecycle-states |
| Design         | PROJECT/design-code-review-workflow                      |
| Tasks Reviewed | 4 done, 1 not-planned (housekeeping)                     |

---

## Summary Verdict

**changes_required** — 1 blocking finding in Test Adequacy. Implementation is correct and conforms to the specification. The blocking finding is a test coverage gap that leaves AC-17 (advance from `reviewing` to `done`) unverified at the smart-skip layer.

---

## Per-Dimension Verdicts

| Dimension                  | Verdict          | Notes                                                        |
|----------------------------|------------------|--------------------------------------------------------------|
| Specification Conformance  | pass             | All ACs (01–13, 16–18, 21–22) correctly implemented          |
| Implementation Quality     | pass             | Defence-in-depth on stop-state, clean Phase 1/2 separation   |
| Test Adequacy              | fail             | 1 blocking: AC-17 advance-from-reviewing path untested       |
| Documentation Currency     | pass_with_notes  | Spec still in draft status                                   |
| Workflow Integrity         | pass             | All tasks done, documents linked, summaries consistent        |

---

## Review Units

| Unit | Dimensions                                     | Files                                                                 |
|------|------------------------------------------------|-----------------------------------------------------------------------|
| A    | Specification Conformance, Implementation Quality | `entities.go`, `lifecycle.go`, `advance.go`, `gates.go`, `checker.go` |
| B    | Test Adequacy, Documentation Currency, Workflow Integrity | `entities_test.go`, `lifecycle_test.go`, `advance_test.go`, `gates_test.go`, `entity_consistency_test.go`, `entities_test.go` |

---

## Blocking Findings

### F1: AC-17 advance-from-reviewing test gap

- **Location**: `internal/service/advance_test.go` L357–384 (`TestAdvanceFeatureStatus_AdvanceToReviewing_IsTarget`)
- **Dimension**: Test Adequacy
- **Severity**: blocking
- **Description**: The test labeled AC-17 (`AdvanceToReviewing_IsTarget`) tests advancing **to** `reviewing` as an explicit target, not advancing **from** `reviewing` to `done`. AC-17 specifies: _"When advance: true is applied to a feature in reviewing status, the feature may advance to done."_ The advance-from-reviewing→done path is untested at the smart-skip layer. The underlying single-step transition `reviewing → done` is covered by `TestCanTransition` (lifecycle_test.go ~L383) and the integration test `TestEntityService_StatusUpdate_UsesLifecycleValidation` (entities_test.go L461), so the transition itself works. However, if the advance loop's gate-checking logic were accidentally changed to fire on `reviewing` exit rather than entry, no test would catch the regression.
- **Requirement violated**: Spec §4.5, AC-17
- **Suggested remediation**: Add a test `TestAdvanceFeatureStatus_AdvanceFromReviewing_ToDone` that:
  1. Writes a feature entity at status `reviewing`
  2. Calls `AdvanceFeatureStatus(feature, "done", …)`
  3. Asserts `result.FinalStatus == "done"` with empty `StoppedReason`
  4. Asserts `result.AdvancedThrough` contains `["done"]`

---

## Non-Blocking Findings

### F2: Phase 1 legacy states in ValidNextStates assertion

- **Location**: `internal/validate/lifecycle_test.go` ~L968 (`TestValidNextStates`, "feature needs-rework" case)
- **Dimension**: Test Adequacy
- **Severity**: non-blocking
- **Description**: The `ValidNextStates` test for `needs-rework` expects `["cancelled", "developing", "in-progress", "in-review", "reviewing", "superseded"]`. The spec AC-03 defines the Phase 2 transition map for `needs-rework` as `needs-rework → developing, reviewing, superseded, cancelled`. The `in-progress` and `in-review` values are Phase 1 legacy states not mentioned in the Phase 2 spec. While the AC-13 wording ("returns a set that **includes** developing and reviewing") is technically satisfied, spec AC-03 says the transition map should match "**exactly**". A clarifying comment in the test would prevent future confusion about whether these extra transitions are intentional.

### F3: Missing ValidNextStates test cases for intermediate states

- **Location**: `internal/validate/lifecycle_test.go` L890–1028 (`TestValidNextStates`)
- **Dimension**: Test Adequacy
- **Severity**: non-blocking
- **Description**: `TestValidNextStates` has explicit assertions for `proposed`, `developing`, `reviewing`, and `needs-rework` feature states. However, the intermediate Phase 2 states — `designing`, `specifying`, and `dev-planning` — have no `ValidNextStates` test cases. AC-03 specifies the exact transition sets for these states, and while `TestCanTransition` verifies individual transitions, it does not verify the **complete** set for each state (i.e., that there are no extra or missing outbound transitions). This reduces the regression safety net.

### F4: Specification document still in draft status

- **Location**: Document record `FEAT-01KMRX1F47Z94/specification-review-lifecycle-states`
- **Dimension**: Documentation Currency
- **Severity**: non-blocking
- **Description**: The specification document is still in **draft** status. The feature is in `reviewing` state and all 4 implementation tasks are `done`. Typically, the spec should be approved before the feature enters review so that reviewers can reference an approved baseline. This should be addressed before or during the `reviewing → done` transition.

### F5: Housekeeping task in not-planned status

- **Location**: Workflow metadata — `TASK-01KMRZQBKXCMB` (`verify-remediation-test`)
- **Dimension**: Workflow Integrity
- **Severity**: non-blocking
- **Description**: Task `TASK-01KMRZQBKXCMB` is in `not-planned` status under this feature. Its summary indicates it was a tool chain verification task created during the verify-tool-chain step. This is a terminal state and the task's intent is clear. Noted for completeness only.

---

## AC Coverage Matrix

| AC    | Covered? | Primary Test(s)                                                                      |
|-------|----------|--------------------------------------------------------------------------------------|
| AC-01 | ✅       | `TestEnumStringValues` (entities_test.go L577)                                        |
| AC-02 | ✅       | `TestEnumStringValues` (entities_test.go L580)                                        |
| AC-03 | ✅       | `TestCanTransition` (lifecycle_test.go L370–430)                                      |
| AC-04 | ✅       | `TestCanTransition` developing→done=false (L367) + error message test (L1085)         |
| AC-05 | ✅       | `TestCanTransition` "reviewing to done (AC-05)" (L383)                                |
| AC-06 | ✅       | `TestCanTransition` "reviewing to needs-rework (AC-06)" (L389)                        |
| AC-07 | ✅       | `TestCanTransition` "needs-rework to developing (AC-07)" (L395)                       |
| AC-08 | ✅       | `TestCanTransition` "needs-rework to reviewing quick-fix (AC-08)" (L401)              |
| AC-09 | ✅       | Pre-existing transition tests for plan/epic/task/bug/decision retained                |
| AC-10 | ✅       | `TestCanTransition` reviewing→superseded/cancelled, needs-rework→same (L407–430)      |
| AC-11 | ✅       | `TestValidNextStates` "feature developing includes reviewing (AC-11)" (L955)          |
| AC-12 | ✅       | `TestValidNextStates` "feature reviewing includes done and needs-rework (AC-12)" (L961) |
| AC-13 | ✅       | `TestValidNextStates` "feature needs-rework includes developing and reviewing" (L968) |
| AC-14 | ✅       | `TestValidateTransition_ErrorContainsValidStates` "AC-14" case (L1085)                |
| AC-15 | ✅       | `TestValidateTransition_ErrorContainsValidStates` two "AC-15" cases (L1093, L1103)    |
| AC-16 | ✅       | `TestAdvanceFeatureStatus_AdvanceToDone_StopsAtReviewing` (L192)                      |
| AC-17 | ⚠️       | **Gap**: test labeled AC-17 tests advancing TO reviewing, not FROM reviewing→done     |
| AC-18 | ✅       | `TestAdvanceFeatureStatus_NeverAutoTransitionsThroughReviewing` (L389)                |
| AC-19 | ✅       | `TestCheckFeatureChildConsistency_ReviewingFeatureNoFalseWarnings` (L318)             |
| AC-20 | ✅       | `TestIsTerminalState` reviewing=false and needs-rework=false (L98–99)                 |
| AC-21 | ✅       | `TestEntityDoneGate_Check` "feature reviewing fails" (gates_test.go L579)             |
| AC-22 | ✅       | `TestEntityDoneGate_Check` "feature needs-rework fails" (gates_test.go L586)          |
| AC-23 | ✅ᵃ      | Per TASK-01KMRXKXMFBF3 completion: `go test -race ./...` all 22 packages pass         |
| AC-24 | ✅ᵃ      | Per TASK-01KMRXKXMFBF3 completion: `go vet ./...` clean                               |
| AC-25 | ✅ᵃ      | Per TASK-01KMRXKXMFBF3 completion: no pre-existing tests broken                       |

ᵃ Accepted on basis of task completion verification claim.

---

## Transition Decision

**Verdict**: `changes_required` → transition feature to `needs-rework`

**Required remediation**: Fix blocking finding F1 (add advance-from-reviewing→done test for AC-17).

**Recommended follow-ups** (non-blocking, can be addressed during remediation):
- F2: Add clarifying comment about Phase 1 legacy states in ValidNextStates test
- F4: Approve the specification document before transitioning feature to `done`
