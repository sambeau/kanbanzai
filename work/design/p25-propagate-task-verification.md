# Design: Propagate Task Verification to Feature Entity

**Feature:** FEAT-01KPQ08Y989P8
**Plan:** P25 — Agent Tooling and Pipeline Quality
**Status:** Draft

---

## Overview

Every agentic `finish()` call records a verification string on the task record
(`taskRecord.Fields["verification"]` in `CompleteTask`, `internal/service/dispatch.go`).
Two merge gates — `verification_exists` and `verification_passed` — check the *feature
entity's* `verification` and `verification_status` fields respectively, not the task-level
fields. These feature-level fields are never populated by `finish()`, nor by any other MCP
tool call outside the formal reviewing stage workflow.

The consequence is that every agentic merge requires `override: true` on both gates. This
is not a theoretical gap: all five P24 features failed `verification_exists` and
`verification_passed` and required an override. Using `override` as the normal code path
degrades its signal value — it should indicate an exceptional circumstance, not be the
standard outcome of a correctly-executed agentic pipeline.

The formal reviewing stage that normally populates verification fields is itself a mandatory
gate that cannot be auto-advanced in agentic-only pipelines. The lifecycle model assumes a
human or reviewer-agent sits between `developing` and `done`; in agentic pipelines that
stage is replaced by automated test runs recorded in task `finish()` calls. There is no
bridge from that task-level data to the feature entity. This design adds that bridge.

---

## Goals and Non-Goals

### Goals

- When all tasks for a feature transition to a terminal state (`done` or `wont_do`) via
  `finish()`, aggregate their `verification` fields and write a summary to the feature
  entity's `verification` field and a computed value to `verification_status`.
- After the change, `merge(action: check)` must pass `verification_exists` and
  `verification_passed` without `override` when all tasks were completed with a non-empty
  verification string.
- Partial verification (some tasks have no `verification` field) must be surfaced as a
  non-blocking warning at the `verification_passed` gate, not a blocking failure.
- The aggregation must be best-effort: a failure to write to the feature entity must not
  prevent the task from being marked done.

### Non-Goals

- Implementing the reviewing-stage auto-advance logic (proposal P7). That is a separate
  feature (FEAT-01KPQ08YE4399) with its own design.
- Changing the free-text semantics of the `verification` field — it remains a string, not a
  structured object.
- Adding a pass/fail boolean to `finish()` parameters. Verification status is derived from
  the presence or absence of a string, not from an explicit flag.
- Populating verification fields for features that have no tasks. Zero-task features
  continue to require a manual write if the gates must pass.
- Modifying the reviewing stage lifecycle, its gate prerequisites, or its role in
  human-reviewed pipelines.

---

## Design

### Trigger Point

Aggregation fires at the end of `finishOne()`, after `CompleteTask()` has succeeded and the
finished task is in its terminal state. The condition mirrors the existing nudge-1 check
that is already present in `finishOne()`:

1. The finished task has a non-empty `parent_feature` field.
2. All sibling tasks for that feature are now in a terminal state (`done` or `wont_do`).

This means aggregation runs exactly once per feature — when the last task completes — and
produces a single, deterministic state transition on the feature entity. Intermediate task
completions do not trigger it. The sibling task list returned by `ListEntitiesFiltered` for
the nudge check is reused directly; no additional query is needed.

In batch mode (`finishBatch()`), aggregation is deferred to after all items in the batch are
processed, so the all-terminal check accounts for every task in the batch before querying
siblings. This mirrors the existing deferred auto-commit pattern.

### Fields Written

Two fields are written to the feature entity via `entitySvc.UpdateEntity()`:

**`verification` (string)** — A newline-separated summary of per-task verification strings,
one line per `done` task. Tasks in `wont_do` state are excluded. Tasks in `done` state with
an empty `verification` field are represented by the placeholder `(no verification
recorded)`:

```
TASK-01Kxxx: go test ./... passed, 0 failures
TASK-01Kyyy: lint clean, all unit tests green
TASK-01Kzzz: (no verification recorded)
```

**`verification_status` (string)** — One of three values:

| Value       | Condition                                                                  |
|-------------|----------------------------------------------------------------------------|
| `"passed"`  | All `done` tasks have a non-empty `verification` field                     |
| `"partial"` | ≥1 `done` task has a non-empty field, but ≥1 `done` task has an empty one |
| `"none"`    | No `done` tasks have a verification field (or only `wont_do` tasks exist)  |

When `verification_status` is `"none"`, aggregation writes nothing to the feature entity and
returns silently. This preserves the current gate behaviour (both gates fail; merge requires
override) rather than writing a misleading or empty summary.

### Merge Gate Behaviour After the Change

`VerificationExistsGate` checks `entity["verification"]` for non-empty content. After
aggregation, the summary string is non-empty for `"passed"` and `"partial"` statuses (it
includes at least one task line). This gate passes for both without any gate-level code
change.

`VerificationPassedGate` checks `entity["verification_status"] == "passed"`. A status of
`"partial"` does not equal `"passed"`, so without a gate change the gate would still block.
To implement the non-blocking-partial requirement, the gate logic is updated with one
additional case:

- `verification_status == "passed"` → `GateStatusPassed` (unchanged)
- `verification_status == "partial"` → `GateStatusWarning` (new; was implicitly `GateStatusFailed`)
- `verification_status` absent or any other value → `GateStatusFailed` (unchanged)

The gate's declared `Severity()` remains `GateSeverityBlocking`; severity is an upper bound.
The change is localised to `VerificationPassedGate.Check()` in `internal/merge/gates.go`.

### Implementation Boundaries

| Component | Change |
|---|---|
| `internal/service/dispatch.go` | New method `AggregateTaskVerification(featureID string) (*VerificationAggregationResult, error)` on `DispatchService`. Queries sibling tasks, builds summary string, computes status, writes to feature entity via `entitySvc.UpdateEntity()`. |
| `internal/mcp/finish_tool.go` | In `finishOne()`: after the existing all-terminal check, call `dispatchSvc.AggregateTaskVerification(parentFeatureID)`. Best-effort: log and continue on error. Include result under `"verification_aggregation"` in the MCP response when it runs. |
| `internal/merge/gates.go` | `VerificationPassedGate.Check()`: add case for `verification_status == "partial"` → `GateStatusWarning`. |

`AggregateTaskVerification` returns a result type that carries the status computed and
whether a write was performed, so `finishOne()` can include it in the MCP response for
agent visibility.

### Data Flow

```
finishOne()
  └─ dispatchSvc.CompleteTask(...)
       writes task.Fields["verification"]
       returns CompleteResult
  └─ [all siblings terminal?]
       └─ dispatchSvc.AggregateTaskVerification(parentFeatureID)
            └─ entitySvc.ListEntitiesFiltered(Type:"task", Parent:featureID)
            └─ for each done task: read task.State["verification"]
            └─ build summary string, compute "passed" | "partial" | "none"
            └─ if status != "none":
                 entitySvc.UpdateEntity(Type:"feature", ID:featureID, Fields:{
                   "verification":        "<summary>",
                   "verification_status": "<passed|partial>",
                 })
            └─ return VerificationAggregationResult{Status, Written}
  └─ include result in MCP response under "verification_aggregation"
```

### Open Questions

1. **Overwrite guard**: If the feature entity already has a manually-set `verification` field
   (e.g. written by a reviewer agent), should aggregation overwrite it or skip? The current
   design overwrites unconditionally. A guard (skip if `verification_status` is already
   `"passed"`) would prevent regression but adds complexity. The specification should settle
   this before implementation.

2. **`wont_do`-only features**: If all tasks are `wont_do` (none are `done`), aggregation
   produces no tasks to summarise and hits the `"none"` path, writing nothing. This preserves
   existing gate behaviour. Whether this is the right outcome for a feature where all work was
   deliberately skipped is an open design question; it is left for the reviewer.

---

## Alternatives Considered

### 1. Aggregate on Every `finish()` Call (Not Just the Last)

**Approach:** On every `finish()` call, regardless of whether all siblings are done,
recompute the feature-level verification summary from tasks completed so far and write it to
the feature entity.

**What it makes easier:** The feature entity has a live, incrementally-updated verification
snapshot after each task completion. Tooling that reads the feature entity mid-flight (e.g.
`entity(action: "get")`) would show partial progress.

**What it makes harder:** The feature entity is written on every `finish()` call, not just
the last, producing multiple YAML mutations during development. The `verification_status`
would cycle through `"none"` → `"partial"` → `"passed"` as tasks complete, creating noisy
intermediate states. If `merge(action: check)` is called mid-flight, it might see a
misleading partial state. There is no correctness benefit over single-write-on-completion:
the final state after all tasks are done is identical in both approaches.

**Why rejected:** Single-write-on-completion is simpler and produces one deterministic state
transition per feature. Incremental writes add noise without a correctness advantage.

### 2. Dedicated `record-verification` MCP Tool

**Approach:** Add a new tool `record_verification(entity_id, summary, status)` that
explicitly writes `verification` and `verification_status` on a feature entity. Agents would
call this tool after all tasks are done, before running `merge(action: check)`.

**What it makes easier:** The contract is explicit. Agents control the summary text and
status value. No automatic inference; no surprising side effects from `finish()`.

**What it makes harder:** This is effectively the current status quo with a renamed surface.
Agents already have `entity(action: "update")` which can write arbitrary fields. The problem
is not the absence of a tool; it is that the step is consistently omitted in agentic
pipelines. A new tool merely renames the obligation without eliminating it. Every pipeline
still needs an extra call that currently gets forgotten, and the gap persists.

**Why rejected:** A new tool does not solve the root cause: the step is reliably skipped.
Automatic aggregation on the last `finish()` removes the obligation entirely — no explicit
call is needed, no forgetting is possible.

### 3. Remove the Verification Gates Entirely

**Approach:** Delete `VerificationExistsGate` and `VerificationPassedGate` from
`DefaultGates()`. Features would merge without any verification signal requirement.

**What it makes easier:** No gate failures, no overrides required. The merge path becomes
simpler for all pipeline types.

**What it makes harder:** The gates exist because the lifecycle model expects some
verification evidence to be recorded before a merge is considered ready. Removing them
eliminates all protection, including for human-reviewed pipelines where a reviewer explicitly
sets verification status. The `HealthCheckCleanGate` is already a placeholder that always
passes; removing two more substantive gates would leave the gate suite largely decorative.

**Why rejected:** The problem is that the gates are unreachable via agentic task completion,
not that the gates are wrong. Removing them conflates "hard to reach in agentic context" with
"not needed". The correct fix is to make the reachable code path populate the fields the
gates check.

---

## Dependencies

| Dependency | Kind | Notes |
|---|---|---|
| `internal/service.EntityService.UpdateEntity` | Internal (existing) | Writes `verification` and `verification_status` to the feature entity. `ValidateRecord` does not validate these fields as enum-constrained, so no schema change is required. |
| `internal/service.EntityService.ListEntitiesFiltered` | Internal (existing) | Queries sibling tasks by `parent_feature`. Already called from `finishOne()` for the nudge logic; the aggregation reuses this result. |
| `internal/service.DispatchService` | Internal (change required) | New `AggregateTaskVerification` method. `DispatchService` already holds `entitySvc`, so no new injected dependency is needed. |
| `internal/mcp/finish_tool.go` | Internal (change required) | Calls `AggregateTaskVerification` after the all-terminal check. Result included in MCP response. |
| `internal/merge/gates.go` `VerificationPassedGate` | Internal (change required) | One new case: `verification_status == "partial"` → `GateStatusWarning`. |