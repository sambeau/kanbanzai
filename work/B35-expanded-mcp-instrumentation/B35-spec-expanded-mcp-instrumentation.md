| Field  | Value                                      |
|--------|--------------------------------------------|
| Date   | 2026-04-28                                 |
| Status | approved |
| Author | spec-author (Claude)                       |
| Design | `work/design/p35-expanded-mcp-instrumentation.md` |

---

## Related Work

### Prior designs consulted

| Document | Section | Relationship |
|----------|---------|-------------|
| `work/design/kanbanzai-3.0-workflow-and-tooling-v2.md` | §12 | Original action log design — established the JSONL schema, `.kbz/logs/` location, and `kbz metrics` concept. This specification extends that schema. |
| `work/design/doc-intel-enhancement-design.md` | — | Established the precedent for handler-level annotation: `doc_intel(action: classify)` records `model_name` and `model_version` from within the handler. This specification generalises that pattern via the `AnnotateEntry` API. |
| `work/design/transition-log-design.md` | — | Related philosophy: append-only, event-per-action, never rewrite history. The `server_version` per-entry approach follows the same append-only constraint. |

### Constraining decisions

- **JSONL in `.kbz/logs/` is the log format.** §12.6 of the 3.0 workflow design explicitly chose structured JSONL over a persistent database. This specification must not reverse that decision.
- **Log errors are swallowed (FR-018).** Log write failures must not affect tool responses. This applies to all new fields and annotations.
- **`doc_intel(action: classify)` is the precedent.** Handler-level knowledge can be included in tool responses without breaking the outer hook. The annotation API generalises this pattern.

### Concept and entity search

Concepts searched: "actionlog", "instrumentation", "metrics", "side effect", "knowledge rejection". No directly related specifications were found in the document corpus. This is the first specification for P35.

## Overview

This specification implements the design described in
`work/design/p35-expanded-mcp-instrumentation.md`
(P35-expanded-mcp-instrumentation/design-p35-expanded-mcp-instrumentation).
It defines the requirements for extending the Kanbanzai action log system to
capture five currently invisible signals: server version correlation,
zero-result list/search calls, knowledge rejection events, tool action
distribution, document approval funnels, and task completion gaps. All changes
are additive and backward-compatible with existing JSONL log files.

## Scope

**In scope:**
- Adding `server_version` to every action log `Entry`
- Adding a sparse `extra` annotation map to `Entry` with a context-carried annotation API
- Instrumenting three list/search handlers with `result_count` annotations
- Capturing `SideEffectKnowledgeRejected` counts in the action log
- Adding three new aggregations to `ComputeMetrics`: action distribution, document approval funnel, and task completion gap

**Out of scope:**
- OpenTelemetry, Prometheus, or any push-based telemetry framework
- Persistent aggregated metric storage
- Logging query text or response content
- Latency measurement per tool call
- Annotating handlers beyond the three initial callers
- Changes to JSONL rotation, cleanup, or file partitioning
- Tool subset compliance metric (deferred to future work)
- Changes to the `kbz metrics` CLI output format

## Functional Requirements

- **FR-001:** `Entry` MUST include a `ServerVersion` field (JSON `"server_version"`) that records the server's link-time version string. The field MUST use `omitempty` so that existing log files without the field remain valid.

- **FR-002:** The `version` string from `cmd/kanbanzai/main.go` (set via ldflags) MUST be threaded through `NewServer` to `actionlog.NewHook`, and stamped onto every `Entry` written by `Hook.Wrap`.

- **FR-003:** `Entry` MUST include an `Extra` field of type `map[string]string` (JSON `"extra,omitempty"`) for sparse, handler-specific annotations.

- **FR-004:** The `actionlog` package MUST provide an `AnnotateEntry(ctx context.Context, key, value string)` function that annotates the current tool call's `Entry` with a key-value pair. When no annotation collector is present on the context, the function MUST be a no-op (no panic, no error).

- **FR-005:** Annotation key constants MUST be defined in the `actionlog` package as exported string constants: `AnnotationResultCount`, `AnnotationKBRejections`, `AnnotationEntityType`, and `AnnotationDocType`.

- **FR-006:** `Hook.Wrap` MUST place an annotation collector on the context before invoking the inner handler, drain the collector after the handler returns, and merge collected key-value pairs into `Entry.Extra` before writing the log entry.

- **FR-007:** The `entity(action: list)` handler MUST annotate the log entry with `AnnotationResultCount` set to the number of entities returned.

- **FR-008:** The `knowledge(action: list)` handler MUST annotate the log entry with `AnnotationResultCount` set to the number of knowledge entries returned.

- **FR-009:** The `doc_intel(action: search)` handler MUST annotate the log entry with `AnnotationResultCount` set to the number of search result sections returned.

- **FR-010:** `Hook.Wrap` MUST inspect the context's side-effect collector after the inner handler returns, count any `SideEffectKnowledgeRejected` events, and when the count is non-zero write it to `Entry.Extra[AnnotationKBRejections]`.

- **FR-011:** The knowledge rejection capture in FR-010 MUST NOT require changes to the `finish` handler or the knowledge service. It MUST use the string constant `"SideEffectKnowledgeRejected"` to identify rejection events, avoiding a direct import of `internal/mcp` into `internal/actionlog`.

- **FR-012:** `ComputeMetrics` MUST produce an `ActionDistribution` aggregation: a list of `{tool, action, calls, failures}` records grouped by `(tool, action)`, sorted by `calls` descending. Entries where `action` is absent (tools with no action parameter) MUST use the empty string for the action field.

- **FR-013:** `ComputeMetrics` MUST produce a `DocTypeFunnel` aggregation: a list of `{doc_type, registered, approved, rate}` records. Document type MUST be resolved by extending `StageFeatureLookup` with a `DocType(entityID string) (string, error)` method. The funnel compares `doc(action: "register")` entries against `doc(action: "approve")` entries for the same `entity_id` within the query time window.

- **FR-014:** `ComputeMetrics` MUST produce a `TaskCompletionGap` aggregation: a single `{median_hours, p90_hours, count}` record computed from paired `next` and `finish` log entries for the same `entity_id`. The gap is the time between the `next` call timestamp and the corresponding `finish` call timestamp.

## Non-Functional Requirements

- **FR-NF-001:** All new `Entry` fields MUST be backward-compatible: log files written by existing binaries (without `server_version` or `extra`) MUST remain readable by the updated `ReadEntries` function without errors or data loss.

- **FR-NF-002:** Log write failures during annotation capture MUST NOT affect the tool response returned to the MCP client. Annotation data discarded by a log write failure is silently lost — this is acceptable per the existing FR-018 constraint.

- **FR-NF-003:** `ComputeMetrics` MUST compute all three new aggregations in a single pass over the log entries within the query window. `ReadEntries` MUST NOT be called more than once per `ComputeMetrics` invocation.

## Acceptance Criteria

- **AC-001 (FR-001, FR-NF-001):** Given a log file written by an existing binary (no `server_version` field), when `ReadEntries` reads the file, then every parsed `Entry` has `ServerVersion == ""` with no parse error.

- **AC-002 (FR-002):** Given the server is built with `-ldflags "-X main.version=2.0.0"`, when any tool is called, then the resulting log entry contains `"server_version":"2.0.0"`.

- **AC-003 (FR-003, FR-NF-001):** Given a log file written by an existing binary (no `extra` field), when `ReadEntries` reads the file, then every parsed `Entry` has `Extra == nil` with no parse error.

- **AC-004 (FR-004, FR-006):** Given a handler calls `AnnotateEntry(ctx, "k", "v")` within a `Hook.Wrap` invocation, when the log entry is written, then `Extra["k"] == "v"`.

- **AC-005 (FR-004):** Given `AnnotateEntry(ctx, "k", "v")` is called on a context with no annotation collector, then the call returns without panicking and without modifying any state.

- **AC-006 (FR-005):** The constants `AnnotationResultCount`, `AnnotationKBRejections`, `AnnotationEntityType`, and `AnnotationDocType` are defined in the `actionlog` package and are of type `string`.

- **AC-007 (FR-007):** Given `entity(action: "list", type: "task", status: "ready")` returns 3 tasks, when the log entry is written, then `Extra["result_count"] == "3"`.

- **AC-008 (FR-008):** Given `knowledge(action: "list")` returns 0 entries, when the log entry is written, then `Extra["result_count"] == "0"`.

- **AC-009 (FR-009):** Given `doc_intel(action: "search", query: "foo")` returns 5 sections, when the log entry is written, then `Extra["result_count"] == "5"`.

- **AC-010 (FR-010):** Given a `finish(task_id, knowledge: [...])` call where all knowledge entries are rejected as duplicates, when the log entry is written, then `Extra["kb_rejections"]` equals the number of rejected entries.

- **AC-011 (FR-010):** Given a `finish(task_id)` call with no knowledge contributions, when the log entry is written, then the `"kb_rejections"` key is absent from `Extra`.

- **AC-012 (FR-011):** The `actionlog` package does NOT import `internal/mcp` or any sub-package of `internal/mcp`.

- **AC-013 (FR-012):** Given log entries for 10 `knowledge(action: "list")` calls (8 successful, 2 failed) and 5 `knowledge(action: "compact")` calls (all successful), when `ComputeMetrics` runs, then `ActionDistribution` contains an entry with `{tool: "knowledge", action: "list", calls: 10, failures: 2}` sorted before `{tool: "knowledge", action: "compact", calls: 5, failures: 0}`.

- **AC-014 (FR-013):** Given 4 documents were registered (2 specification, 1 design, 1 dev-plan) and 2 were later approved (1 specification, 1 design), when `ComputeMetrics` runs, then `DocTypeFunnel` contains `{doc_type: "specification", registered: 2, approved: 1, rate: 0.5}`, `{doc_type: "design", registered: 1, approved: 1, rate: 1.0}`, and `{doc_type: "dev-plan", registered: 1, approved: 0, rate: 0.0}`.

- **AC-015 (FR-014):** Given `next(id: "TASK-001")` is called at time T1 and `finish(task_id: "TASK-001")` at time T2 (2 hours later), when `ComputeMetrics` runs over a window containing both entries, then `TaskCompletionGap.Median` and `TaskCompletionGap.P90` equal 2.0 and `TaskCompletionGap.Count` is at least 1.

- **AC-016 (FR-NF-002):** Given the log writer is failing (e.g. disk full), when a handler annotated with `result_count` is called, then the tool response is returned successfully to the MCP client without error from the logging layer.

- **AC-017 (FR-NF-003):** Given a 30-day log window containing 10,000 entries, when `ComputeMetrics` runs, then `ReadEntries` is called at most once.

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Test | Unit test: parse a log line without `server_version`, assert `ServerVersion == ""` |
| AC-002 | Test | Integration test: build with ldflags version, invoke a tool, read log, assert version field |
| AC-003 | Test | Unit test: parse a log line without `extra`, assert `Extra == nil` |
| AC-004 | Test | Unit test: `Hook.Wrap` with a handler that calls `AnnotateEntry`, assert `Extra` populated |
| AC-005 | Test | Unit test: call `AnnotateEntry` on a bare `context.Background()`, assert no panic |
| AC-006 | Inspection | Code review: verify constant definitions in `actionlog` package |
| AC-007 | Test | Integration test: call `entity(action: list, ...)`, read log, assert `result_count` |
| AC-008 | Test | Integration test: call `knowledge(action: list)`, read log, assert `result_count == "0"` |
| AC-009 | Test | Integration test: call `doc_intel(action: search, ...)`, read log, assert `result_count` |
| AC-010 | Test | Unit test: `Hook.Wrap` with a handler returning `SideEffectKnowledgeRejected`, assert `kb_rejections` in `Extra` |
| AC-011 | Test | Unit test: `Hook.Wrap` with a handler returning no side effects, assert `kb_rejections` absent from `Extra` |
| AC-012 | Inspection | Code review: verify `internal/actionlog` has no import of `internal/mcp` |
| AC-013 | Test | Unit test: `ComputeMetrics` with synthetic log entries, assert `ActionDistribution` structure and sort order |
| AC-014 | Test | Unit test: `ComputeMetrics` with synthetic doc register/approve entries, assert `DocTypeFunnel` |
| AC-015 | Test | Unit test: `ComputeMetrics` with paired `next`/`finish` entries, assert `TaskCompletionGap` |
| AC-016 | Test | Unit test: `Hook.Wrap` with a failing writer, assert inner handler result returned unchanged |
| AC-017 | Test | Unit test: verify `ReadEntries` call count during `ComputeMetrics` with synthetic data |

## Dependencies and Assumptions

### Dependencies on existing code

| Package / File | Change required |
|---------------|-----------------|
| `cmd/kanbanzai/main.go` | Pass `version` string to `NewServer` |
| `internal/mcp/server.go` | Accept `version` in `NewServer` / `newServerWithConfig`, pass to `actionlog.NewHook` |
| `internal/actionlog/entry.go` | Add `ServerVersion` and `Extra` fields |
| `internal/actionlog/hook.go` | Add `version` field to `Hook`, `NewHook` signature change, annotation collector in `Wrap`, side-effect inspection |
| `internal/actionlog/metrics.go` | Add `ActionDistribution`, `DocTypeFunnel`, `TaskCompletionGap` types and aggregation logic; extend `StageFeatureLookup` with `DocType` |
| `internal/mcp/entity_tool.go` | Add `AnnotateEntry` call in list handler |
| `internal/mcp/knowledge_tool.go` | Add `AnnotateEntry` call in list handler |
| `internal/mcp/doc_intel_tool.go` | Add `AnnotateEntry` call in search handler |
| `internal/mcp/sideeffect.go` | No changes required — referenced by string constant only |

### Assumptions

- The `version` variable in `cmd/kanbanzai/main.go` is set via ldflags and contains only alphanumeric characters, dots, and hyphens — all safe in JSON without escaping.
- The `StageFeatureLookup` interface currently has one implementation (`entityStageLookup` in `server.go`). Adding `DocType` will break this implementation, requiring a corresponding update.
- Handlers are trusted code — no validation or bounding of `Extra` map size is required.
- Log volumes are on the order of hundreds of entries per day, making the single-pass `ComputeMetrics` approach (O(n) over all entries) acceptable.
- All `doc` entities carry a type that can be resolved at aggregation time from entity records.
