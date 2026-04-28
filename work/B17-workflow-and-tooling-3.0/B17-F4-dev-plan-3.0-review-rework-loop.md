# Implementation Plan: Review-Rework Loop Formalisation

| Field          | Value                                                     |
|----------------|-----------------------------------------------------------|
| Feature        | FEAT-01KN5-8J2606B0 (review-rework-loop)                 |
| Specification  | `work/spec/3.0-review-rework-loop.md`                     |
| Status         | Draft                                                     |

---

## 1. Overview

This plan decomposes the Review-Rework Loop Formalisation specification into five implementation tasks. The specification adds a `review_cycle` counter to the feature entity, enforces an iteration cap that blocks infinite refinement loops, and provides a focused re-review signal for context assembly.

### Scope boundaries (carried forward from specification)

- **In scope:** `review_cycle` field, counter increment on transition to `reviewing`, hardcoded cap (`DefaultMaxReviewCycles = 3`), cap-reached blocking with automatic checkpoint creation, blocked state representation, re-review signal for context assembly, counter persistence rules, visibility in `status` and `entity get` output.
- **Out of scope:** Reviewing skill methodology, stage gate enforcement, binding registry runtime integration, stage-aware context assembly pipeline changes, review verdict recording.

---

## 2. Task Breakdown

### Task 1: Add `review_cycle` and `blocked_reason` fields to feature entity model and storage

**Objective:** Extend the feature entity with a `review_cycle` integer field (default 0) and a `blocked_reason` string field. Update the YAML serialisation field order so both fields are persisted correctly. Ensure backward compatibility — existing feature YAML files without these fields load successfully with zero-value defaults.

**Specification references:** FR-001, FR-007 (representation choice), FR-009, NFR-001

**Design decision — FR-007 representation:** Use approach (a) from the specification: a `blocked_reason` field on the feature entity. When non-empty, the feature is functionally blocked while remaining in `reviewing` status. This avoids adding a new lifecycle state and keeps the transition map unchanged.

**Input context:**
- `internal/model/entities.go` — `Feature` struct (L290–319), `FeatureStatus` constants
- `internal/storage/entity_store.go` — `fieldOrderForEntityType` (L220+), `MarshalCanonicalYAML`, `UnmarshalCanonicalYAML`
- `refs/go-style.md` — YAML serialisation rules, naming conventions

**Output artifacts:**
- Modified `internal/model/entities.go`:
  - Add `ReviewCycle int` field with `yaml:"review_cycle,omitempty"` tag to `Feature` struct
  - Add `BlockedReason string` field with `yaml:"blocked_reason,omitempty"` tag to `Feature` struct
- Modified `internal/storage/entity_store.go`:
  - Add `"review_cycle"` and `"blocked_reason"` to the feature field order list (after `"branch"`, before `"created"`)
- New test file `internal/storage/entity_store_review_cycle_test.go`:
  - Round-trip test: write feature with `review_cycle: 2`, read back, verify value
  - Backward compatibility test: load YAML without `review_cycle` field, verify it parses as 0 (absent from map)
  - Round-trip test: write feature with `blocked_reason` set, read back, verify value
  - Verify `blocked_reason` is omitted from YAML when empty

**Dependencies:** None — this is the foundational task.

**Interface contract:** The `review_cycle` field is stored in the entity state map as key `"review_cycle"` with an `int` value. The `blocked_reason` field is stored as key `"blocked_reason"` with a `string` value. All downstream tasks read these fields from `record.Fields["review_cycle"]` and `record.Fields["blocked_reason"]`. A missing `"review_cycle"` key or a `nil` value MUST be treated as `0` by all consumers. A missing or empty `"blocked_reason"` means the feature is not blocked.

---

### Task 2: Implement counter increment and define cap constant

**Objective:** Increment `review_cycle` by exactly 1 each time a feature transitions INTO `reviewing` status, regardless of source state. Define `DefaultMaxReviewCycles = 3` as a named constant. The increment must occur atomically with the transition — it must not be possible to enter `reviewing` without the counter incrementing. Non-reviewing transitions must not modify the counter.

**Specification references:** FR-002, FR-004, FR-009, NFR-002, NFR-003, NFR-004

**Input context:**
- `internal/service/entities.go` — `UpdateStatus` method (L588–670), the existing pattern for modifying fields during transitions (e.g., clearing `rework_reason` at L641–643)
- `internal/validate/lifecycle.go` — `CanTransition`, `ValidateTransition` functions
- `internal/model/entities.go` — `FeatureStatusReviewing` constant

**Output artifacts:**
- New file `internal/service/review_cycle.go`:
  - Named constant: `DefaultMaxReviewCycles = 3`
  - Helper function: `incrementReviewCycleIfReviewing(entityType string, fields map[string]any, nextStatus string)` — increments `review_cycle` in the fields map when `entityType == "feature"` and `nextStatus == "reviewing"`. Reads the current value from `fields["review_cycle"]`, treating missing/nil as 0, writes back the incremented value. No-ops for all other entity types and statuses.
- Modified `internal/service/entities.go`:
  - In `UpdateStatus`, after the `ValidateTransition` call succeeds and before `record.Fields["status"] = nextStatus`, call `incrementReviewCycleIfReviewing(entityType, record.Fields, nextStatus)`
- New test file `internal/service/review_cycle_test.go`:
  - Test: transition feature from `developing` → `reviewing` when `review_cycle` is 0, verify result is 1
  - Test: transition feature from `developing` → `reviewing` when `review_cycle` is 1, verify result is 2
  - Test: transition feature from `needs-rework` → `reviewing` (direct re-entry), verify increment
  - Test: transition feature from `reviewing` → `needs-rework`, verify counter unchanged
  - Test: transition feature from `needs-rework` → `developing`, verify counter unchanged
  - Test: transition feature from `reviewing` → `done`, verify counter unchanged
  - Test: `DefaultMaxReviewCycles` constant equals 3
  - Test: calling `incrementReviewCycleIfReviewing` for a task entity is a no-op

**Dependencies:** Task 1 (entity model fields must exist)

**Interface contract with Task 3:** The `incrementReviewCycleIfReviewing` function is called inside `UpdateStatus` before the write. Task 3's cap-check logic will be called after the increment (using the post-increment value), also inside `UpdateStatus`, by a separate function. The increment function must write the new value back to `fields["review_cycle"]` so the cap check can read it.

**Interface contract with Task 5:** The `DefaultMaxReviewCycles` constant is defined in `internal/service/review_cycle.go` and is importable by the context assembly code in `internal/mcp/`.

---

### Task 3: Implement cap-reached blocking and automatic checkpoint creation

**Objective:** When a feature at `review_cycle == DefaultMaxReviewCycles` receives a transition request from `reviewing` to `needs-rework`, block the transition: set a `blocked_reason` on the feature, persist it, and automatically create a human checkpoint. Features below the cap transition normally. Features at the cap transitioning to `done` (pass verdict) are unaffected.

**Specification references:** FR-005, FR-006, FR-007 (blocking behaviour)

**Input context:**
- `internal/service/entities.go` — `UpdateStatus` method
- `internal/service/review_cycle.go` — `DefaultMaxReviewCycles` constant (from Task 2)
- `internal/checkpoint/checkpoint.go` — `Store`, `Record`, `Status`, `Create` method
- `internal/mcp/entity_tool.go` — `entityTransitionAction` (L555–636), how errors/results are returned

**Output artifacts:**
- Modified `internal/service/review_cycle.go`:
  - New function: `checkReviewCycleCap(fields map[string]any, currentStatus, nextStatus string) (blocked bool, reason string)` — returns `true` and a formatted reason string when `currentStatus == "reviewing"`, `nextStatus == "needs-rework"`, and `fields["review_cycle"]` (as int) `>= DefaultMaxReviewCycles`. The reason string MUST match the pattern: `"Review iteration cap reached (N/N). Human decision required: accept with known issues, rework with revised scope, or cancel."` where N is the cycle count and the cap value.
- Modified `internal/service/entities.go`:
  - Add a `CheckpointCreator` interface field on `EntityService`:
    ```
    type CheckpointCreator interface {
        CreateCheckpoint(question, context, createdBy string) error
    }
    ```
  - Add `SetCheckpointCreator(cc CheckpointCreator)` method on `EntityService`
  - In `UpdateStatus`, after `incrementReviewCycleIfReviewing` and before writing, call `checkReviewCycleCap`. If blocked:
    1. Set `record.Fields["blocked_reason"]` to the reason string
    2. Do NOT set `record.Fields["status"] = nextStatus` (feature stays in `reviewing`)
    3. Write the record with the blocked_reason set
    4. Call `CheckpointCreator.CreateCheckpoint(...)` with appropriate question, context, and `"system"` as created_by
    5. Return a `GetResult` with the updated state and a new error type (or structured error) indicating the transition was blocked by the cap
- New adapter file `internal/service/checkpoint_adapter.go`:
  - Adapter struct that wraps `*checkpoint.Store` and implements `CheckpointCreator`
  - The `CreateCheckpoint` method creates a `checkpoint.Record` with the question, context, `StatusPending`, current time, and created_by, then calls `Store.Create`
- Modified wiring (wherever `EntityService` is constructed, likely `cmd/kanbanzai/` or server setup):
  - Wire the checkpoint adapter into the entity service via `SetCheckpointCreator`
- New test file `internal/service/review_cycle_cap_test.go`:
  - Test: feature at `review_cycle: 3` (cap=3), transition `reviewing → needs-rework` → blocked, status stays `reviewing`, `blocked_reason` set with correct format
  - Test: blocked_reason contains "Review iteration cap reached (3/3)"
  - Test: blocked_reason contains all three decision options
  - Test: feature at `review_cycle: 2`, transition `reviewing → needs-rework` → succeeds normally, no blocked_reason
  - Test: feature at `review_cycle: 3`, transition `reviewing → done` → succeeds normally (cap only applies to fail/needs-rework)
  - Test: exactly one checkpoint created when cap triggers
  - Test: checkpoint question references review iteration cap
  - Test: checkpoint context includes feature ID and cycle count
  - Test: checkpoint status is `pending`
  - Test: no checkpoint created when below cap
  - Test: transition to `needs-rework` while `blocked_reason` is set returns an error explaining human decision is required

**Dependencies:** Task 1 (entity model fields), Task 2 (counter increment logic and cap constant)

**Interface contract:** The `checkReviewCycleCap` function is a pure function that reads `fields["review_cycle"]` and returns a boolean + reason. It does not mutate state — the caller in `UpdateStatus` handles persistence and checkpoint creation. The `CheckpointCreator` interface is deliberately narrow (one method) to keep the dependency minimal and testable with a mock.

---

### Task 4: Expose `review_cycle` and `blocked_reason` in status and entity tool output

**Objective:** Surface the `review_cycle` value and `blocked_reason` in both the `status` tool's feature detail response and the `entity(action: "get")` response. The `status` tool must clearly indicate when a feature is blocked and include the blocked reason. Add an attention item when a feature is blocked by the iteration cap.

**Specification references:** FR-003, FR-007 (status display), FR-010

**Input context:**
- `internal/mcp/status_tool.go` — `featureInfo` struct (L198–205), `featureDetail` struct (L513–530), `synthesiseFeature` function (L551–676), `generateFeatureAttention` function (L933–971)
- `internal/mcp/entity_tool.go` — `entityFullRecord` function (L844–857)
- The entity tool's `entityFullRecord` already copies all `state` fields into the response map, so `review_cycle` and `blocked_reason` will appear in `entity get` output automatically once they exist in the entity state. FR-010 is largely satisfied by Task 1. This task verifies and tests that behaviour.

**Output artifacts:**
- Modified `internal/mcp/status_tool.go`:
  - Add `ReviewCycle int` field with `json:"review_cycle,omitempty"` to `featureInfo` struct
  - Add `BlockedReason string` field with `json:"blocked_reason,omitempty"` to `featureInfo` struct
  - In `synthesiseFeature`, read `review_cycle` from `feat.State["review_cycle"]` (treating missing/nil as 0) and `blocked_reason` from `feat.State["blocked_reason"]` (treating missing/nil as empty). Set on the `featureInfo` in the response.
  - In `generateFeatureAttention`, add a new attention item when `blocked_reason` is non-empty: `"Feature blocked: <blocked_reason> — respond to the pending checkpoint to unblock"`
- New test file `internal/mcp/status_review_cycle_test.go`:
  - Test: `status` with feature at `review_cycle: 0` — field omitted (or present as 0)
  - Test: `status` with feature at `review_cycle: 2` — response contains `"review_cycle": 2`
  - Test: `status` with feature with `blocked_reason` set — response contains blocked_reason and attention item
  - Test: `entity(action: "get")` for feature with `review_cycle: 2` — response contains the field (verify via `entityFullRecord` passthrough)

**Dependencies:** Task 1 (entity model fields must exist in state)

**Interface contract:** The `featureInfo` struct is used in both `featureDetail` (feature-scoped status) and `featureSummary` (plan-scoped status). Adding fields to `featureInfo` surfaces them in both contexts. The `review_cycle` field uses `omitempty` so it is absent when 0, matching the specification's FR-003 acceptance criteria.

---

### Task 5: Implement focused re-review signal in context assembly

**Objective:** When a feature is in `reviewing` status with `review_cycle ≥ 2`, signal to the context assembly system (the `handoff` tool) that this is a re-review. The handoff prompt must include guidance that narrows context to rework tasks, previous review findings, and rework task descriptions — not the full implementation.

**Specification references:** FR-008

**Input context:**
- `internal/mcp/assembly.go` — `assembledContext` struct (L91–109), `asmInput` struct (L114–123), `assembleContext` function (L131+)
- `internal/mcp/handoff_tool.go` — `renderHandoffPrompt` function (L183+), how `assembleContext` is called (L138–148 of the handler)
- `internal/service/review_cycle.go` — `DefaultMaxReviewCycles` constant (from Task 2)

**Output artifacts:**
- Modified `internal/mcp/assembly.go`:
  - Add to `assembledContext` struct:
    - `reReview bool` — true when the parent feature is in a re-review cycle
    - `reviewCycle int` — the current review cycle number
  - In `assembleContext`, after existing assembly steps, add a new step `asmCheckReReview`:
    - If `input.entitySvc` is not nil and `input.parentFeature` is not empty, load the parent feature entity
    - Read `review_cycle` from feature state (default 0) and `status`
    - If `status == "reviewing"` and `review_cycle >= 2`, set `actx.reReview = true` and `actx.reviewCycle = review_cycle`
- Modified `internal/mcp/handoff_tool.go`:
  - In `renderHandoffPrompt`, after the "Summary" section and before spec sections, add a re-review guidance block when `actx.reReview` is true:
    ```
    ### Re-Review Guidance (Cycle N)

    This is review cycle N, not a first review. Focus your review on:
    - Rework tasks and their changes — not the full implementation
    - Previous review findings that triggered the rework
    - Whether rework task descriptions match what actually changed
    - Regression in areas adjacent to rework changes
    ```
    Where N is `actx.reviewCycle`.
- New test file `internal/mcp/assembly_rereview_test.go`:
  - Test: feature at `review_cycle: 1` in `reviewing` — `reReview` is false
  - Test: feature at `review_cycle: 2` in `reviewing` — `reReview` is true, `reviewCycle` is 2
  - Test: feature at `review_cycle: 3` in `reviewing` — `reReview` is true
  - Test: feature at `review_cycle: 2` in `developing` (not reviewing) — `reReview` is false
  - Test: handoff prompt contains "Re-Review Guidance (Cycle 2)" when re-review is active
  - Test: handoff prompt does NOT contain "Re-Review Guidance" when `review_cycle` is 1

**Dependencies:** Task 1 (entity model fields), Task 2 (counter must be set on features entering reviewing)

**Interface contract:** The `reReview` and `reviewCycle` fields on `assembledContext` are internal to the assembly pipeline. The handoff tool reads them via the `assembledContext` struct — no public API change. The re-review signal is rendered as a Markdown section in the prompt, consumed by agents as natural language guidance.

---

## 3. Dependency Graph

```
Task 1: Entity model + storage
  │
  ├──→ Task 2: Counter increment + cap constant
  │      │
  │      ├──→ Task 3: Cap blocking + checkpoint creation
  │      │
  │      └──→ Task 5: Re-review signal in context assembly
  │
  └──→ Task 4: Status + entity tool output
```

**Parallel execution groups:**

| Phase | Tasks | Description |
|-------|-------|-------------|
| 1     | Task 1 | Foundation — entity model and storage changes |
| 2     | Task 2, Task 4 | Can run in parallel — counter logic and tool output are independent |
| 3     | Task 3, Task 5 | Can run in parallel — cap blocking and re-review signal are independent |

**Critical path:** Task 1 → Task 2 → Task 3

---

## 4. Traceability Matrix

| Requirement | Task(s) | Verification |
|---|---|---|
| FR-001: Review cycle counter field | Task 1 | Unit test: create feature, verify `review_cycle` default 0; load legacy YAML, verify 0 |
| FR-002: Counter increment on transition to reviewing | Task 2 | Unit test: transitions to `reviewing` from multiple source states increment by 1; non-reviewing transitions leave counter unchanged |
| FR-003: Counter visible in status output | Task 4 | Integration test: `status` tool returns `review_cycle` in feature detail |
| FR-004: Hardcoded iteration cap | Task 2 | Code inspection: `DefaultMaxReviewCycles = 3` constant exists and is referenced by cap-check logic |
| FR-005: Cap-reached blocking behaviour | Task 3 | Unit test: feature at cap with needs-rework transition is blocked; below cap transitions normally; at cap with done transition succeeds |
| FR-006: Automatic checkpoint creation at cap | Task 3 | Unit test: checkpoint created when cap triggers; checkpoint fields contain feature ID, cycle count, decision options; no checkpoint when below cap |
| FR-007: Blocked state representation | Task 1 (field), Task 3 (behaviour), Task 4 (display) | Unit test: blocked_reason set and visible in status; transition to needs-rework rejected while blocked |
| FR-008: Focused re-review signal | Task 5 | Unit test: feature at cycle ≥ 2 entering reviewing triggers re-review signal; cycle 1 does not; handoff prompt includes re-review guidance |
| FR-009: Counter survives non-review transitions | Task 1 (persistence), Task 2 (logic) | Unit test: counter unchanged after transitions to needs-rework, developing, done, cancelled, superseded |
| FR-010: Entity tool exposes review cycle | Task 4 | Integration test: `entity(action: "get")` response includes `review_cycle` |
| NFR-001: Backward compatibility | Task 1 | Unit test: feature YAML without `review_cycle` loads without error |
| NFR-002: Performance | Task 2 | By design: single integer comparison, no additional I/O |
| NFR-003: Determinism | Task 2, Task 3 | By design: increment before cap check; single code path; no concurrency |
| NFR-004: Constant extractability | Task 2 | Code inspection: constant defined once, referenced by cap-check |

All 10 functional requirements and 4 non-functional requirements are covered. Every task traces to at least one requirement. Every requirement is covered by at least one task.

---

## 5. Interface Contracts Summary

### 5.1 Entity state field conventions

| Field | Key in `record.Fields` | Type | Default | Set by |
|---|---|---|---|---|
| `review_cycle` | `"review_cycle"` | `int` | 0 (absent) | Task 2 (increment logic in `UpdateStatus`) |
| `blocked_reason` | `"blocked_reason"` | `string` | `""` (absent) | Task 3 (cap-reached handler in `UpdateStatus`) |

All consumers MUST treat a missing key or `nil` value as the zero value (0 for int, empty for string).

### 5.2 `CheckpointCreator` interface (Task 3)

```
type CheckpointCreator interface {
    CreateCheckpoint(question, context, createdBy string) error
}
```

Defined in `internal/service/`. Implemented by an adapter wrapping `*checkpoint.Store`. Injected into `EntityService` via `SetCheckpointCreator`. This follows the existing pattern of `SetStatusTransitionHook` for the worktree hook.

### 5.3 Re-review signal fields (Task 5)

Added to `assembledContext` (internal, unexported):

| Field | Type | Meaning |
|---|---|---|
| `reReview` | `bool` | True when parent feature is in reviewing at cycle ≥ 2 |
| `reviewCycle` | `int` | Current review cycle number |

Read by `renderHandoffPrompt` to conditionally emit the re-review guidance section.

---

## 6. Risks and Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| `UpdateStatus` becomes complex with multiple feature-specific hooks | Medium | Keep increment and cap-check as small, pure functions in `review_cycle.go`. The `UpdateStatus` method calls them but they contain no service logic. |
| Checkpoint store not available in all execution contexts | Low | Use the `CheckpointCreator` interface with nil-check. If no creator is set, log a warning but still block the transition (the blocked_reason on the entity is the primary signal). |
| YAML integer parsing for `review_cycle` | Low | The existing `parseScalar` in `entity_store.go` already handles integers via `strconv.Atoi`. Verified in codebase review. |
| Blocked feature cannot be unblocked | Medium (UX) | Out of scope for this feature — the checkpoint response flow and unblock mechanism are handled by the existing checkpoint system. The plan documents this as a known boundary. |