| Field  | Value                                      |
|--------|--------------------------------------------|
| Date   | 2026-04-28                                 |
| Status | Draft                                      |
| Author | architect (Claude)                         |
| Spec   | `work/P35-expanded-mcp-instrumentation/P35-spec-expanded-mcp-instrumentation.md` |

---

## Scope

This plan implements the requirements defined in
`work/P35-expanded-mcp-instrumentation/P35-spec-expanded-mcp-instrumentation.md`
(FEAT-01KQ2MH2S2ZDH/spec-p35-spec-expanded-mcp-instrumentation). It covers the
five design components: version tagging on `Entry`, the sparse `extra`
annotation map and context-carried API, handler instrumentation, knowledge
rejection capture, and three new `ComputeMetrics` aggregations.

**Out of scope:** `kbz metrics` CLI output format changes, tool subset
compliance metric, annotation of handlers beyond the three initial callers.

## Task Breakdown

### Task 1: Entry schema — ServerVersion and Extra fields

- **Description:** Add `ServerVersion string` and `Extra map[string]string`
  fields to `actionlog.Entry`, both with `omitempty` JSON tags. Existing
  log files without these fields must remain readable.
- **Deliverable:** Modified `internal/actionlog/entry.go`.
- **Depends on:** None.
- **Effort:** Small.
- **Spec requirement:** FR-001, FR-003, FR-NF-001.

### Task 2: Version threading through server init

- **Description:** Pass the `version` string from `cmd/kanbanzai/main.go`
  through `NewServer(entityRoot, version)` to `actionlog.NewHook(writer,
  lookup, version)`. Stamp `ServerVersion` onto every `Entry` in
  `Hook.Wrap`. Update all callers of `NewServer` and `NewHook`.
- **Deliverable:** Modified `cmd/kanbanzai/main.go`, `internal/mcp/server.go`,
  `internal/actionlog/hook.go`.
- **Depends on:** Task 1 (needs `ServerVersion` field on `Entry`).
- **Effort:** Small.
- **Spec requirement:** FR-002.

### Task 3: Annotation API and Hook.Wrap integration

- **Description:** Create the context-carried annotation collector and
  `AnnotateEntry(ctx, key, value string)` function in `actionlog`. Define
  the four annotation key constants. Extend `Hook.Wrap` to place the
  collector on the context before calling the inner handler, drain it
  after, and merge key-value pairs into `Entry.Extra` before writing.
  Ensure log write failures do not affect the tool response.
- **Deliverable:** New `internal/actionlog/annotate.go` and
  `internal/actionlog/annotate_test.go`; modified
  `internal/actionlog/hook.go` and `internal/actionlog/hook_test.go`.
- **Depends on:** Task 1 (needs `Extra` field on `Entry`).
- **Effort:** Medium.
- **Spec requirement:** FR-004, FR-005, FR-006, FR-NF-002.

### Task 4: Handler instrumentation

- **Description:** Add `AnnotateEntry` calls in the list/search handlers of
  `entity`, `knowledge`, and `doc_intel` tools, annotating `result_count`
  with the number of results returned.
- **Deliverable:** Modified `internal/mcp/entity_tool.go`,
  `internal/mcp/knowledge_tool.go`, `internal/mcp/doc_intel_tool.go`.
- **Depends on:** Task 3 (needs `AnnotateEntry` function and collector on context).
- **Effort:** Small.
- **Spec requirement:** FR-007, FR-008, FR-009.

### Task 5: Knowledge rejection capture

- **Description:** Extend `Hook.Wrap` to inspect the context's side-effect
  collector after the inner handler returns, count `SideEffectKnowledgeRejected`
  events using the string constant `"SideEffectKnowledgeRejected"`, and write
  the count to `Entry.Extra[AnnotationKBRejections]` when non-zero. Ensure
  `internal/actionlog` does not import `internal/mcp`.
- **Deliverable:** Modified `internal/actionlog/hook.go` and
  `internal/actionlog/hook_test.go`.
- **Depends on:** Task 3 (needs annotation collector and `Extra` merge in `Wrap`).
- **Effort:** Small.
- **Spec requirement:** FR-010, FR-011.

### Task 6: ComputeMetrics aggregations

- **Description:** Add `ActionDistribution`, `DocTypeFunnel`, and
  `TaskCompletionGap` types to `MetricsResult`. Implement aggregation
  logic for each: group-by-tool-and-action, doc register-to-approve
  comparison, and paired `next`-to-`finish` gap calculation. Extend
  `StageFeatureLookup` with `DocType(entityID string) (string, error)`
  and update the `entityStageLookup` implementation. Ensure single-pass
  computation over log entries.
- **Deliverable:** Modified `internal/actionlog/metrics.go` and
  `internal/actionlog/metrics_test.go`; modified
  `internal/mcp/server.go` (`entityStageLookup`).
- **Depends on:** Task 1 (needs `ServerVersion` and `Extra` fields on `Entry`
  for reading — the aggregation reads entries but does not depend on
  Tasks 2–5).
- **Effort:** Medium.
- **Spec requirement:** FR-012, FR-013, FR-014, FR-NF-003.

### Task 7: End-to-end tests and verification

- **Description:** Write tests covering all 17 acceptance criteria:
  backward-compatible parsing of old log files, version stamping,
  annotation collection and draining, no-op behaviour on bare context,
  handler instrumentation correctness, knowledge rejection capture,
  import isolation for `actionlog`, all three aggregation outputs, log
  failure isolation, and single-pass computation. Fix any issues
  discovered during testing.
- **Deliverable:** New and modified test files across
  `internal/actionlog/*_test.go` and `internal/mcp/*_test.go`.
- **Depends on:** Tasks 2, 4, 5, 6 (all implementation must be complete).
- **Effort:** Medium.
- **Spec requirement:** All acceptance criteria (AC-001 through AC-017).

## Dependency Graph

```
Task 1 (no dependencies)
Task 2 → depends on Task 1
Task 3 → depends on Task 1
Task 4 → depends on Task 3
Task 5 → depends on Task 3
Task 6 → depends on Task 1
Task 7 → depends on Task 2, Task 4, Task 5, Task 6
```

```
Parallel groups: [Task 2, Task 3, Task 6] after Task 1 completes
                 [Task 4, Task 5] after Task 3 completes
Critical path: Task 1 → Task 3 → Task 4 → Task 7
```

## Risk Assessment

### Risk: Hook.Wrap side-effect inspection breaks existing handlers

- **Probability:** Low.
- **Impact:** Medium — existing tool calls could fail or produce incorrect
  log entries if the side-effect collector interaction changes.
- **Mitigation:** The inspection is additive (read-only on the collector);
  Task 3 is designed to drain annotations without modifying side effects.
  Task 7 includes tests for the full handler pipeline.
- **Affected tasks:** Task 3, Task 5.

### Risk: StageFeatureLookup.DocType breaks the entityStageLookup implementation

- **Probability:** Medium — adding a method to an interface with one
  implementation requires updating that implementation.
- **Impact:** Low — the implementation is in a single file
  (`internal/mcp/server.go`) and the fix is mechanical.
- **Mitigation:** Task 6 includes the implementation update. The change is
  colocated with the aggregation code that consumes it.
- **Affected tasks:** Task 6.

### Risk: NewServer signature change breaks test setup

- **Probability:** Medium — tests in `internal/mcp/server_test.go` or
  elsewhere may construct `NewServer` with the old single-argument form.
- **Impact:** Low — test compilation failures are immediately visible and
  the fix is mechanical (add `"test"` as the version argument).
- **Mitigation:** Task 2 includes updating all callers. Task 7 catches any
  missed callers during test execution.
- **Affected tasks:** Task 2.

### Risk: Single-pass aggregation constraint conflicts with clean separation

- **Probability:** Low — the existing `ComputeMetrics` already does a single
  pass over entries. The three new aggregations are computed from the same
  entry stream.
- **Impact:** Medium — if a second pass is needed for `DocTypeFunnel`
  (because it requires entity lookups), the non-functional requirement
  FR-NF-003 would be violated.
- **Mitigation:** Design the aggregation to collect `entity_id`s during the
  single entry pass, then batch-resolve document types via
  `StageFeatureLookup.DocType` after the pass completes. This is one pass
  over entries plus one batch lookup, which satisfies the single
  `ReadEntries` call constraint.
- **Affected tasks:** Task 6.

## Verification Approach

| Acceptance Criterion | Verification Method | Producing Task |
|---------------------|--------------------|----------------|
| AC-001: Old log without `server_version` parses with empty string | Unit test | Task 7 |
| AC-002: Log entry contains `server_version` from ldflags | Integration test | Task 7 |
| AC-003: Old log without `extra` parses with nil map | Unit test | Task 7 |
| AC-004: `AnnotateEntry` populates `Extra` via `Hook.Wrap` | Unit test | Task 7 |
| AC-005: `AnnotateEntry` is no-op on bare context | Unit test | Task 7 |
| AC-006: Annotation constants defined in `actionlog` package | Code inspection | Task 3 (self-verifying) |
| AC-007: `entity(action: list)` annotates `result_count` | Integration test | Task 7 |
| AC-008: `knowledge(action: list)` annotates zero `result_count` | Integration test | Task 7 |
| AC-009: `doc_intel(action: search)` annotates `result_count` | Integration test | Task 7 |
| AC-010: KB rejections captured in `Extra["kb_rejections"]` | Unit test | Task 7 |
| AC-011: No `kb_rejections` key when no rejections occur | Unit test | Task 7 |
| AC-012: `actionlog` does not import `internal/mcp` | Code inspection | Task 5 (self-verifying) |
| AC-013: `ActionDistribution` structure and sort order | Unit test | Task 7 |
| AC-014: `DocTypeFunnel` with mixed register/approve data | Unit test | Task 7 |
| AC-015: `TaskCompletionGap` from paired `next`/`finish` entries | Unit test | Task 7 |
| AC-016: Log write failure does not affect tool response | Unit test | Task 7 |
| AC-017: Single `ReadEntries` call in `ComputeMetrics` | Unit test | Task 7 |
