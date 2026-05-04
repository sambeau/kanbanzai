# Review: B43 Composite Tools for Workflow Chaining

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-04                     |
| Author | orchestrator (sambeau)         |
| Review cycle | 1                       |
| Reviewers dispatched | reviewer-conformance, reviewer-quality, reviewer-testing |
| Review units | 1 (all files) |

---

## Per-Reviewer Summary

### Reviewer: reviewer-conformance
- Review unit: B43-composite-tools-all
- Verdict: **needs_remediation**
- Dimensions:
  - `spec_conformance`: **fail**
  - `implementation_quality`: concern
  - `test_adequacy`: **fail**
  - `workflow_integrity`: concern
- Findings: 5 blocking, 5 non-blocking

### Reviewer: reviewer-quality
- Review unit: B43-composite-tools-all
- Verdict: **needs_remediation**
- Dimensions:
  - Error handling chain: concern
  - Resource lifecycle: pass
  - Naming consistency: pass
  - Package cohesion: pass
  - Dead code / unused parameters: **fail**
  - Cyclomatic complexity / readability: pass_with_notes
  - Defensive coding / input validation: concern
- Findings: 2 blocking, 6 non-blocking

### Reviewer: reviewer-testing
- Review unit: B43-composite-tools-all
- Verdict: **needs_remediation**
- Dimensions:
  - `test_adequacy`: **fail**
  - `implementation_quality`: concern
- Findings: 15 blocking, 3 non-blocking

---

## Collated Findings (deduplicated)

### Blocking

**[B-1] Missing review report check in `entity(action: "close-out")`**
- Dimension: spec_conformance, workflow_integrity
- Location: `internal/mcp/entity_tool.go:L1196-1199`
- Spec ref: REQ-011, AC-009
- Description: The code has a comment `// Check for approved review report.` but the next line only extracts `batchID` and immediately proceeds to `UpdateStatus` with status `"done"`. No check for an approved review report document is performed. This means a feature can be closed out without an approved review report, directly contradicting REQ-011. The `docSvc` is passed to `entityCloseOutAction` but never used for document lookup.
- Remediation: Use `docSvc` to query for an approved review report document owned by the feature. Return a structured `nextActionForMissingDocument("report", featureID)` if absent.
- Reported by: reviewer-conformance, reviewer-quality

**[B-2] Missing conflict analysis in `develop(action: "dispatch")`**
- Dimension: spec_conformance, implementation_quality
- Location: `internal/mcp/develop_tool.go:L49-50, L162-165`
- Spec ref: REQ-014
- Description: `conflictSvc` is accepted as a parameter by `developDispatchAction` but is never referenced in the function body. The `conflicting` response field is always hardcoded as an empty array `[]any{}`. REQ-014 requires: "runs conflict analysis on those tasks using the existing conflict detection logic, and transitions conflict-safe tasks from ready to active."
- Remediation: Call `conflictSvc` to check for conflicts among ready tasks before transitioning them. Populate the `conflicting` response field with tasks that have conflicts.
- Reported by: reviewer-conformance, reviewer-quality, reviewer-testing

**[B-3] Missing dependency satisfaction check in `develop(action: "dispatch")`**
- Dimension: spec_conformance
- Location: `internal/mcp/develop_tool.go:L100-138`
- Spec ref: REQ-014
- Description: The dispatch action checks only `status == "ready"` and does not verify that all `depends_on` are satisfied before transitioning tasks to active. REQ-014 states: "identify the ready task frontier (tasks with status `ready` and all `depends_on` satisfied)."
- Remediation: For each ready task, check that all `depends_on` tasks are in terminal states before including it in the dispatch frontier.
- Reported by: reviewer-conformance

**[B-4) Missing handoff pipeline integration in `develop(action: "dispatch")`**
- Dimension: spec_conformance
- Location: `internal/mcp/develop_tool.go:L146-L151`
- Spec ref: REQ-015
- Description: Instead of calling the existing handoff pipeline, the dispatch action constructs a minimal `handoff_hint` map inline with static content: `{tool: "handoff", action: "", params: {task_id: t.ID}}`. REQ-015 states: "generate a handoff prompt for each dispatched task using the existing handoff pipeline."
- Remediation: Call the handoff pipeline (available via `handoff_tool.go`) to generate a full handoff prompt for each dispatched task. Pass the `role` and `instructions` parameters from the dispatch call through to handoff.
- Reported by: reviewer-conformance

**[B-5] `entityCloseOutAction` swallows `CountNonTerminalTasks` errors**
- Dimension: error_handling
- Location: `internal/mcp/entity_tool.go:L1209-1214`
- Spec ref: REQ-010
- Description: The error from `CountNonTerminalTasks` is checked only for nil — if `countErr != nil`, the error is silently ignored and the close-out proceeds to advance the feature to done. This could allow a feature to be closed out with unverified task state if the task-counting operation fails.
- Remediation: Return the error: `if countErr != nil { return nil, fmt.Errorf("Cannot close out: %w", countErr) }`
- Reported by: reviewer-quality

**[B-6] Zero automated tests for all composite actions**
- Dimension: test_adequacy
- Location: `internal/mcp/` — no composite action test files
- Spec ref: Verification Plan (AC-001 through AC-014, excluding AC-011)
- Description: The spec's Verification Plan designates 14 acceptance criteria as "Test" (automated). Zero of those 14 have corresponding test code. The dev plan Tasks 1-5 each state "plus unit tests in [test file]" as deliverables. No `develop_tool_test.go` exists. `doc_tool_test.go`, `entity_tool_test.go`, and `batch_test.go` have no tests for `publish`, `bootstrap`, `close-out`, or `snapshot` actions. `integration_test.go` has no composite action integration tests. Only `server_test.go` was updated (tool count 24→26), which verifies registration but not behaviour.
- Remediation: Implement unit tests per the spec's Verification Plan table for each composite action. Implement the integration test for AC-014 (side effect equivalence). Add tests for `next_action.go` constructors.
- Reported by: reviewer-conformance, reviewer-testing

### Non-Blocking

**[NB-1] `nextActionForClassification` targets `doc_intel` instead of `doc`**
- Dimension: spec_conformance
- Location: `internal/mcp/next_action.go:L85-92`
- Description: AC-002 states the next_action should be `{tool: "doc", action: "approve", ...}`. The implementation returns `{tool: "doc_intel", action: "classify", ...}` which directs the agent to classify first — a reasonable workflow improvement (classification is needed before approval), but a deviation from the literal spec text.
- Recommendation: Update the spec to match the implementation or change the implementation to match the spec.
- Reported by: reviewer-conformance

**[NB-2] Generic blocking reasons in dispatch `blocked` field**
- Dimension: spec_conformance
- Location: `internal/mcp/develop_tool.go:L125-127, L170`
- Description: All non-ready tasks get a generic "queued — dependencies not yet satisfied" reason. REQ-016 says the `blocked` field should contain "blocking reasons" implying per-task specificity. Tasks in different non-ready states (queued vs needs-rework vs unknown) could benefit from distinct reasons.
- Recommendation: Add per-status blocking reason descriptions.
- Reported by: reviewer-conformance

**[NB-3] Unused constructor parameters in `DevelopTool`**
- Dimension: implementation_quality
- Location: `internal/mcp/develop_tool.go:L25`
- Description: `DevelopTool` accepts `dispatchSvc`, `knowledgeSvc`, `intelligenceSvc`, and `docSvc` — none are used. Only `entitySvc` and the already-unused `conflictSvc` are passed through.
- Recommendation: Remove unused parameters, or implement the features that require them.
- Reported by: reviewer-quality

**[NB-4] Dead `doneTasks` variable in `developDispatchAction`**
- Dimension: implementation_quality
- Location: `internal/mcp/develop_tool.go:L84-L86`
- Description: `var doneTasks []taskInfo` is declared and populated in the status switch but never read or returned in the response.
- Recommendation: Either include done tasks in the response for completeness, or remove the dead variable.
- Reported by: reviewer-quality

**[NB-5] `readyCount` always zero in `batchSnapshotAction`**
- Dimension: implementation_quality
- Location: `internal/mcp/batch_tool.go:L76`
- Description: `analyseFeatureBlocked` returns `blocked=true` for all non-terminal states, so `isFeatureStatusReady` (which checks for "ready" or "active") is only called on done/cancelled/superseded/default features — none of which match. `readyCount` is always 0.
- Recommendation: Fix the logic so `readyCount` accurately reflects unblocked features, or remove the variable.
- Reported by: reviewer-quality

**[NB-6] Fragile string matching in `entityBootstrapAction`**
- Dimension: implementation_quality
- Location: `internal/mcp/entity_tool.go:L1130-L1141`
- Description: The bootstrap handler uses `strings.Contains` on `stopped_reason` prose text to determine the next action type. If the upstream gate's prose string changes (e.g., "specification" → "spec"), the gate will fall through to the default case producing an incorrect next_action.
- Recommendation: Carry a typed gate reason (enum or structured field) through the advance result independently of the prose message.
- Reported by: reviewer-quality

**[NB-7) Blank identifier type assertions on request arguments**
- Dimension: implementation_quality
- Location: `develop_tool.go:L58`, `doc_tool.go:L426`, `batch_tool.go:L49`, `entity_tool.go:L1084, L1166`
- Description: All composite handlers use `args, _ := req.Params.Arguments.(map[string]any)` with blank identifier. While inherited from existing code, a nil or non-map `Arguments` would produce cryptic downstream errors instead of a clear diagnostic.
- Recommendation: Extract a project-wide helper that checks the type assertion and returns a structured error. Low priority — the MCP framework guarantees `map[string]any`.
- Reported by: reviewer-quality

**[NB-8] `analyseFeatureBlocked` always returns blocked for non-terminal states**
- Dimension: implementation_quality
- Location: `internal/mcp/batch_tool.go:L127-L149`
- Description: Features in designing with approved designs, or specifying with approved specs, are still reported as blocked. REQ-018 says "whether it is blocked" suggesting actual gate evaluation, not assumed blocking.
- Recommendation: Evaluate actual gate status instead of assuming all non-terminal features are blocked.
- Reported by: reviewer-conformance

**[NB-9] No unit tests for `next_action.go` constructors**
- Dimension: test_adequacy
- Location: `internal/mcp/next_action.go`
- Description: The five `nextActionFor*` constructors are small and deterministic, but have no tests. A format regression in the `next_action` JSON shape would go undetected.
- Recommendation: Add a table-driven test verifying each constructor produces the expected JSON serialization shape.
- Reported by: reviewer-testing

**[NB-10] AC-015 pass is partial — 3 pre-existing test failures**
- Dimension: test_adequacy
- Description: Existing `doc_tool_test.go` and `entity_tool_test.go` suites pass for all B43-relevant tests. However, 3 pre-existing test failures exist (`TestEntity_Update_DependsOnRejectsInvalidID`, `TestDocIntelFind_Role_LimitCappedAt50`) that are unrelated to B43 (verified via git log — no B43 changes to those test files).
- Recommendation: These pre-existing failures should be addressed in a separate bug-fix task, not B43.
- Reported by: reviewer-testing

---

## Aggregate Verdict: **rejected**

**Rationale:** Six blocking findings indicate fundamental spec misalignment:
- B-1: Missing review report check in close-out (REQ-011 not implemented)
- B-2: Missing conflict analysis in dispatch (REQ-014 not implemented)
- B-3: Missing dependency satisfaction check in dispatch (REQ-014 not fully implemented)
- B-4: Missing handoff pipeline integration in dispatch (REQ-015 not implemented)
- B-5: Swallowed error in close-out (REQ-010 violated)
- B-6: Zero automated tests (14 spec-designated test criteria not implemented)

The `develop(action: "dispatch")` handler accounts for 3 of the 6 blocking findings — it's the least complete of the five composite actions. The `entity(action: "close-out")` handler has 2 blocking findings (missing review report check + swallowed error). The test coverage gap applies across all five composite actions.

---

## Remediation Plan

1. [B-1] Implement review report check in `entityCloseOutAction` → route to implementer
2. [B-2] Implement conflict analysis in `developDispatchAction` → route to implementer
3. [B-3] Implement dependency satisfaction check in `developDispatchAction` → route to implementer
4. [B-4] Integrate handoff pipeline in `developDispatchAction` → route to implementer
5. [B-5] Return error when `CountNonTerminalTasks` fails → route to implementer
6. [B-6] Write unit tests for all composite actions per spec Verification Plan → route to implementer

**Estimated remediation effort:** The three `developDispatchAction` fixes (B-2, B-3, B-4) plus test coverage (B-6) constitute the bulk of the work. The close-out fixes (B-1, B-5) are each a few lines of code. Recommended: one remediation task per composite action (5 tasks) plus one integration/testing task.

**Non-blocking findings** (NB-1 through NB-10) should be addressed during remediation but do not independently block merge.
