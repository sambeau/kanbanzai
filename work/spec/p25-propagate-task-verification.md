# Specification: Propagate Task Verification to Feature Entity

**Feature:** FEAT-01KPQ08Y989P8
**Plan:** P25 — Agent Tooling and Pipeline Quality
**Design:** work/design/p25-propagate-task-verification.md
**Status:** Draft

---

## 1. Overview

When all tasks for a feature reach a terminal state (`done` or `wont_do`) via `finish()`, the
system MUST aggregate their per-task `verification` fields and write a summary string and a
computed status value to the feature entity. This bridges the gap between task-level verification
data recorded by `finish()` and the feature-level `verification` and `verification_status` fields
checked by the `verification_exists` and `verification_passed` merge gates, eliminating the need
for `override: true` on those gates in correctly-executed agentic pipelines.

---

## 2. Scope

### In scope

- Aggregation of task `verification` fields to a feature entity when the last task reaches a
  terminal state via `finish()`.
- Writing `verification` (string summary) and `verification_status` (`"passed"`, `"partial"`, or
  `"none"`) to the feature entity.
- Updated `VerificationPassedGate` behaviour: `"partial"` status yields a warning, not a failure.
- Aggregation in both single-item (`finish` single task) and batch (`finish` multiple tasks) modes.
- Best-effort aggregation: a write failure MUST NOT prevent task completion.
- Inclusion of the aggregation result in the `finish()` MCP response when aggregation runs.

### Out of scope

- Aggregation on every `finish()` call (only fires when all siblings are terminal).
- Modifying the `reviewing` stage lifecycle, its gates, or its role in human-reviewed pipelines.
- Changing the free-text semantics of the `verification` field (it remains a plain string).
- Adding a `pass`/`fail` parameter to `finish()`.
- Populating verification fields for features with zero tasks.
- Any changes to `finish()` parameters or existing `finish()` return fields beyond the new
  `"verification_aggregation"` key.

---

## 3. Functional Requirements

### FR-001: Aggregation trigger condition

The system MUST trigger verification aggregation at the end of `finishOne()` when ALL of the
following conditions hold:

1. The completed task has a non-empty `parent_feature` field.
2. After the task is marked terminal, all sibling tasks for that feature are in a terminal state
   (`done` or `wont_do`).

Aggregation MUST NOT fire for intermediate task completions where at least one sibling task
remains non-terminal.

**Acceptance criteria:**
- Completing the third of three tasks triggers aggregation; completing the first or second does not.
- A feature with no tasks never triggers aggregation.
- A task with an empty `parent_feature` field never triggers aggregation.

### FR-002: Aggregation in batch mode

In batch `finish()` mode, aggregation MUST be deferred until after all items in the batch are
processed, so the all-terminal check accounts for every task in the batch before sibling queries
are made.

**Acceptance criteria:**
- Finishing all remaining tasks of a feature in a single batch call produces exactly one
  aggregation write to the feature entity.
- The aggregation sees all tasks in the batch as terminal when computing the sibling check.

### FR-003: Summary string format

The aggregated `verification` string written to the feature entity MUST be a newline-separated
list of per-task lines, one line per `done` task, in the format:

```
<TASK-ID>: <verification text>
```

Tasks in `wont_do` state MUST be excluded from the summary. Tasks in `done` state with an
empty `verification` field MUST be represented by the placeholder `(no verification recorded)`.

**Acceptance criteria:**
- A feature with two `done` tasks (both with non-empty verification) produces a two-line summary.
- A feature with one `done` task (empty verification) and one `done` task (non-empty) produces a
  two-line summary where the empty-verification task's line ends with `(no verification recorded)`.
- Tasks in `wont_do` state do not appear in the summary string.

### FR-004: Verification status derivation

The aggregated `verification_status` value written to the feature entity MUST be one of three
string values, determined as follows:

| Value       | Condition |
|-------------|-----------|
| `"passed"`  | All `done` tasks have a non-empty `verification` field |
| `"partial"` | At least one `done` task has a non-empty field AND at least one `done` task has an empty field |
| `"none"`    | No `done` tasks have a non-empty `verification` field (or no `done` tasks exist) |

**Acceptance criteria:**
- All tasks done with non-empty verification → `verification_status` == `"passed"`.
- Mixed non-empty and empty verification among done tasks → `verification_status` == `"partial"`.
- All tasks done with empty verification → `verification_status` == `"none"`.
- All tasks in `wont_do` → `verification_status` == `"none"`.

### FR-005: No-write for `"none"` status

When the computed `verification_status` is `"none"`, the system MUST NOT write to the feature
entity. Both the `verification` and `verification_status` feature fields remain unchanged.

**Acceptance criteria:**
- A feature where all tasks complete with empty verification fields has no aggregation write
  applied; its `verification` and `verification_status` fields remain as they were before
  `finish()` was called.

### FR-006: Overwrite behaviour

When the computed `verification_status` is `"passed"` or `"partial"`, the system MUST write
to the feature entity unconditionally, overwriting any pre-existing value of `verification`
or `verification_status`.

**Acceptance criteria:**
- A feature with a manually pre-set `verification` field has that field overwritten when
  aggregation runs with `"passed"` or `"partial"` status.

### FR-007: Best-effort aggregation

A failure in the aggregation write MUST NOT cause `finish()` to return an error or prevent the
task from being marked terminal. The task completion MUST succeed regardless of whether the
aggregation write succeeds or fails.

**Acceptance criteria:**
- Simulating a write failure in `AggregateTaskVerification` does not cause `finish()` to fail
  or return an error; the task is still marked done.
- The failure is logged at an appropriate level for diagnostic purposes.

### FR-008: MCP response includes aggregation result

When aggregation runs (i.e. the all-terminal condition is met), the `finish()` MCP response
MUST include a `"verification_aggregation"` key containing at minimum:

- The computed status (`"passed"`, `"partial"`, or `"none"`).
- A boolean indicating whether a write to the feature entity was performed.

**Acceptance criteria:**
- When the last task in a feature is finished via `finish()`, the response JSON contains a
  `"verification_aggregation"` key.
- When a non-last task is finished, `"verification_aggregation"` is absent from the response.

### FR-009: `VerificationPassedGate` partial-status behaviour

`VerificationPassedGate.Check()` MUST return `GateStatusWarning` when
`verification_status == "partial"`. The gate MUST continue to return `GateStatusFailed` when
`verification_status` is absent or any value other than `"passed"` or `"partial"`.

**Acceptance criteria:**
- `merge(action: "check")` on a feature with `verification_status: "passed"` → gate passes.
- `merge(action: "check")` on a feature with `verification_status: "partial"` → gate returns
  warning (non-blocking); merge is not blocked by this gate alone.
- `merge(action: "check")` on a feature with `verification_status` absent → gate fails (blocking).
- `merge(action: "check")` on a feature with `verification_status: "none"` → gate fails (blocking).

### FR-010: `VerificationExistsGate` passes after aggregation

After aggregation with `"passed"` or `"partial"` status, the `verification` field of the
feature entity MUST be non-empty, causing `VerificationExistsGate` to pass without override.

**Acceptance criteria:**
- A feature where all tasks completed with non-empty verification passes `verification_exists`
  gate without `override: true`.
- A feature where tasks have mixed verification (some empty, some not) passes `verification_exists`
  gate without override (summary string is non-empty).

### FR-011: `AggregateTaskVerification` is a method on `DispatchService`

The aggregation logic MUST be implemented as a method `AggregateTaskVerification(featureID string)`
on `DispatchService`, callable from `finishOne()` with access to the existing `entitySvc`
reference already held by `DispatchService`.

**Acceptance criteria:**
- `AggregateTaskVerification` is a method on `DispatchService`.
- No new injected dependencies are added to `DispatchService` to support this method.

---

## 4. Non-Functional Requirements

### NFR-001: No additional queries beyond sibling task list

The aggregation MUST reuse the sibling task list already fetched for the all-terminal nudge
check in `finishOne()`. No additional `ListEntitiesFiltered` calls are permitted beyond what
is already made in `finishOne()` for this purpose.

### NFR-002: Aggregation does not block task completion

The total additional latency introduced by aggregation MUST be bounded by the cost of one
`UpdateEntity` call (a single YAML file write). If aggregation is not triggered (non-last task),
there is zero additional latency.

### NFR-003: Backward compatibility

The change MUST NOT alter the behaviour of `finish()` for tasks that are not the last task in
their feature, or for tasks with no `parent_feature`. Existing `finish()` callers and response
parsers that do not inspect `"verification_aggregation"` MUST continue to work without change.

---

## 5. Acceptance Criteria Summary

| ID     | Requirement | Verification |
|--------|-------------|--------------|
| AC-001 | Aggregation fires only when all siblings terminal | Unit test: partial completion does not trigger |
| AC-002 | Batch mode defers aggregation until all items processed | Unit test: batch finish with last tasks |
| AC-003 | Summary format: one line per done task, placeholder for empty | Unit test: summary string format |
| AC-004 | wont_do tasks excluded from summary | Unit test: mixed done/wont_do tasks |
| AC-005 | `verification_status` == `"passed"` when all done tasks have verification | Unit test |
| AC-006 | `verification_status` == `"partial"` when mixed | Unit test |
| AC-007 | `verification_status` == `"none"` when all empty or all wont_do | Unit test |
| AC-008 | No write when status is `"none"` | Unit test: feature entity unchanged |
| AC-009 | Unconditional overwrite for passed/partial | Unit test: pre-existing value overwritten |
| AC-010 | Task marked done even when aggregation write fails | Unit test with injected write error |
| AC-011 | MCP response contains `verification_aggregation` on last task | Integration test |
| AC-012 | `VerificationPassedGate` returns warning for `partial` | Unit test: gate check |
| AC-013 | `VerificationPassedGate` fails for absent/none status | Unit test: gate check |
| AC-014 | `VerificationExistsGate` passes after passed/partial aggregation | Integration test |
| AC-015 | No new injected dependencies on `DispatchService` | Code review / struct definition |

---

## 6. Dependencies and Assumptions

### Dependencies

| Dependency | Kind | Notes |
|------------|------|-------|
| `internal/service.DispatchService` | Code change | New `AggregateTaskVerification` method |
| `internal/mcp/finish_tool.go` | Code change | Calls `AggregateTaskVerification` in `finishOne()`; includes result in MCP response |
| `internal/merge/gates.go` — `VerificationPassedGate` | Code change | One new case: `"partial"` → `GateStatusWarning` |
| `internal/service.EntityService.UpdateEntity` | Existing | Writes `verification` and `verification_status` to feature entity |
| `internal/service.EntityService.ListEntitiesFiltered` | Existing | Queries sibling tasks; already called in `finishOne()` |

### Assumptions

- `validate.ValidateRecord` does not constrain `verification` or `verification_status` as
  enum-typed fields; no schema change is required to write these values.
- `GateSeverityBlocking` is an upper bound; returning `GateStatusWarning` from a blocking gate
  is valid and does not block the merge.
- The sibling task list returned by `ListEntitiesFiltered` for the nudge check is reusable for
  aggregation without an additional query.
- `wont_do` is a valid terminal task status recognised by the entity service.