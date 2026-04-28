| Field  | Value                                      |
|--------|--------------------------------------------|
| Date   | 2026-04-25                                 |
| Status | approved |
| Author | architect (Claude Sonnet 4.6)              |
| Plan   | P35-expanded-mcp-instrumentation           |

---

## Overview

The Kanbanzai MCP server already records every tool invocation to a structured
JSONL action log (`internal/actionlog`). This log drives the `kbz metrics`
command, which today computes gate failure rates, time-per-lifecycle-stage, and
revision cycle counts. The instrumentation was designed alongside the 3.0
workflow system (§12 of `kanbanzai-3.0-workflow-and-tooling-v2.md`) and has
proved its value: it confirmed that gate failures cluster in specific stages and
that review cycles correlate with specification quality.

Three blind spots remain that prevent data-driven iteration on the server's
features:

1. **No version correlation.** Log entries carry no record of which server
   version produced them. When a tool description or behaviour changes, there is
   no way to ask "did call success rates improve after v2.1?" without joining
   against Git history and deployment timestamps.

2. **Payload quality is invisible.** The outer logging hook sees whether a call
   succeeded or failed, but not what it returned. A `knowledge(action: list)`
   that returns zero entries looks identical in the log to one that returns
   twenty. Zero-result calls are a strong quality signal — they reveal filter
   misuse, stale state, or broken prompting — but they are currently
   undetectable.

3. **Aggregations are underpowered.** `ComputeMetrics` does not surface tool
   action distribution (which of the 13 `knowledge` actions are actually called?)
   or document approval funnels (how many registered documents are never
   approved?). Both questions are answerable from data that is already being
   collected; the aggregation layer has simply not been built.

This design extends the action log schema and the `ComputeMetrics` aggregator to
close these three blind spots without changing the fundamental JSONL-per-day
architecture.

---

## Goals and Non-Goals

**Goals**

- Add `server_version` to every log entry, threaded from the binary's
  link-time version variable through to `actionlog.Hook`.
- Add a sparse `extra` annotation map to `Entry`, with a context-carried
  annotation API that allows individual handlers to attach payload-level signals
  (e.g. result count) without coupling the outer hook to handler internals.
- Define a small set of typed annotation key constants to prevent key drift
  across callers.
- Instrument the three highest-signal handlers with `extra` annotations:
  `entity(action: list)`, `knowledge(action: list)`, and
  `doc_intel(action: search)` — each annotated with `result_count`.
- Capture `SideEffectKnowledgeRejected` events in the action log by inspecting
  drained side effects inside `Hook.Wrap` before writing the entry.
- Add three new aggregations to `ComputeMetrics`: action distribution (call and
  failure count per tool/action pair), document approval funnel (registered vs
  approved per document type), and task completion gap (time from
  `next`→active claim to `finish`→done).
- Keep all changes additive and backward-compatible: existing log files without
  `server_version` or `extra` remain valid.

**Non-Goals**

- OpenTelemetry, Prometheus, or any push-based telemetry framework.
- Persistent aggregated metric storage (metrics are always computed on demand
  from raw JSONL).
- Logging query text or response content (data sensitivity; not analytically
  useful).
- Latency measurement per tool call (dominated by LLM roundtrips; not
  observable from within the server handler).
- Annotating every handler — only handlers where the annotation answers a
  specific identified question are in scope.
- Changes to the JSONL rotation, cleanup, or file partitioning strategy.

---

## Related Work

### Prior documents consulted

| Document | Type | Relevance |
|----------|------|-----------|
| `work/design/kanbanzai-3.0-workflow-and-tooling-v2.md` §12 | Design | Original design of the action pattern logging system; established the JSONL schema, `.kbz/logs/` location, and `kbz metrics` concept |
| `work/design/doc-intel-enhancement-design.md` | Design | Established the precedent for payload-level annotation: `doc_intel(action: classify)` records `model_name` and `model_version` from within the handler |
| `work/design/transition-log-design.md` | Design | Related philosophy: append-only, event-per-action, never rewrite history; also relevant to the `status_transition` side effects captured in the log |

### Constraining decisions

- **JSONL in `.kbz/logs/` is the log format.** §12.6 of
  `kanbanzai-3.0-workflow-and-tooling-v2.md` explicitly chose structured JSONL
  over a persistent database: "A structured log file (JSON lines) that can be
  grepped and aggregated with simple scripts is sufficient." This design must not
  reverse that decision.
- **The outer hook wraps every tool.** `wrapAllTools` in `server.go` applies the
  hook uniformly. The hook must not require per-tool coupling to emit standard
  fields. Payload signals that require handler knowledge must use a
  context-carried mechanism, not hook-level introspection.
- **Log errors are swallowed.** Per FR-018 (captured in `hook.go`): log failures
  must not affect tool responses. This constraint applies to any new fields.
- **`doc_intel(action: classify)` is the precedent.** It demonstrates that
  handler-level knowledge (model name, classification count) can be included in
  tool responses without breaking the outer hook. The annotation API proposed
  here generalises this pattern.

### Open questions from prior work

§12 of `kanbanzai-3.0-workflow-and-tooling-v2.md` listed "tool subset
compliance" (whether agents call excluded tools during a stage) as a metric to
track. This was not implemented and remains out of scope for this design — it
requires cross-referencing log entries against stage-binding tool subsets, which
is a query-layer concern that can be added to `ComputeMetrics` later without
schema changes.

---

## Problem and Motivation

### 1. Version correlation is absent

The `version` variable in `cmd/kanbanzai/main.go` is set at link time via
`-ldflags "-X main.version=<semver>"`. It is used only by the `kbz version`
command. It is not passed to `NewServer`, not stored on `actionlog.Hook`, and
not written to any log entry.

This means that when a change to a tool description, gate behaviour, or skill
produces a measurable change in agent behaviour — fewer gate failures, shorter
time-in-stage, higher doc approval rates — the log data cannot be queried
against the version boundary. The analyst must manually cross-reference
deployment timestamps against Git tags.

This is not hypothetical. The P32 classification pipeline hardening shipped
multiple iterative improvements to `doc_intel`. The action log recorded the
effect, but attributing specific improvements to specific releases required
manual timestamp matching — error-prone for anyone who is not the original
developer.

### 2. Zero-result list calls are a known-unknown

When `entity(action: list, type: task, status: ready)` returns an empty array,
the log records: `{tool: "entity", action: "list", success: true}`. When it
returns fifty tasks, the log records the same thing. The two cases are
analytically indistinguishable.

Empty list results matter because they reveal one of two problems:

- **Filter misuse:** The agent is calling with parameter combinations that
  cannot match any data (e.g. filtering on a status value that no entity
  currently holds). This indicates a prompting or tool-description defect.
- **Broken state:** The system state is empty when the agent expected it not to
  be (e.g. no ready tasks exist because auto-promotion failed). This indicates a
  workflow correctness defect.

Neither defect is currently detectable through the log. Both are high-value
signals for iterating on tool descriptions and workflow correctness.

The same gap applies to `knowledge(action: list)` and
`doc_intel(action: search)`. A search that consistently returns zero results
across many sessions is a strong indicator that the search terms don't match the
indexed content — a corpus classification problem.

### 3. Knowledge rejection is unlogged

`SideEffectKnowledgeRejected` is emitted as a side effect when an inline
knowledge contribution from `finish` is rejected as a duplicate or fails
validation. Side effects are returned to the client in the tool response, but
they are not written to the action log.

This means the following question cannot be answered from log data: *What
fraction of inline knowledge contributions are being rejected, and is that rate
increasing over time?*

A rising rejection rate is a leading indicator that the knowledge base is
accumulating near-duplicate entries — the symptom identified in the P35
knowledge system effectiveness design as a cause of poor delivery quality.

### 4. Action distribution within multi-action tools is invisible

The `knowledge` tool has 13 actions. The log records `{tool: "knowledge",
action: "list"}`, `{tool: "knowledge", action: "compact"}`, etc. This data
exists. `ComputeMetrics` does not aggregate it.

Without an action distribution aggregation, the following questions require
manual log parsing:

- Which `knowledge` actions are never called? (Candidates for removal or
  improved prompting.)
- Which actions have the highest failure rates? (Candidates for better error
  messages or tool-description clarification.)
- Is `knowledge(action: confirm)` ever called, or do entries stay in
  `contributed` status indefinitely?

The same applies to `doc` (8 actions), `entity` (5 actions), and
`doc_intel` (9 actions).

### 5. Document approval funnels are invisible

`doc(action: register)` and `doc(action: approve)` both appear in the log with
the same `entity_id`. The log therefore contains the data needed to compute:
*For each document type, what fraction of registered documents reach `approved`
status?*

This funnel is a workflow health signal. A low approval rate for `specification`
documents means features are being specified but the specs are not being reviewed
and approved — a gate compliance problem. A low rate for `design` documents
means design work is happening but not being formalised.

`ComputeMetrics` does not compute this.

---

## Design

### Component 1: Version tagging in `Entry`

Add `ServerVersion string` to `Entry` with JSON tag `"server_version,omitempty"`.
The `omitempty` tag ensures existing log files (which have no version field)
remain valid — the zero value simply produces no field in serialised output.

Thread the version string from the link-time variable through the server
initialisation path:

```
main.go: version (ldflags) → NewServer(entityRoot, version) → actionlog.NewHook(writer, lookup, version)
```

`NewServer` gains a `version string` parameter. `actionlog.Hook` stores it as a
field and stamps it onto every `Entry` before writing.

The `kbz metrics` command gains no new flags for this change — version is stored
in the data, not the query. Aggregations can group or filter by version as
needed.

**Backward compatibility:** Log entries written by binaries without this change
have `server_version: ""` (omitted from JSON). The reader treats a missing field
as the empty string, which is a valid "unknown version" sentinel.

### Component 2: Sparse `extra` annotation map

Add `Extra map[string]string` to `Entry` with JSON tag `"extra,omitempty"`.
Introduce a context-carried annotation API in the `actionlog` package:

```go
// AnnotateEntry adds a key-value pair to the current entry's extra map.
// It is a no-op if the context carries no annotation collector.
func AnnotateEntry(ctx context.Context, key, value string)
```

This mirrors the `PushSideEffect` / `CollectorFromContext` pattern already used
for side effects. A lightweight `entryAnnotator` struct is stored on the context
by `Hook.Wrap` before calling the inner handler. The hook drains the annotator
after the handler returns, merging the key-value pairs into the `Entry.Extra`
map before writing.

Annotation key names are defined as typed constants in the `actionlog` package
to prevent key drift:

```go
const (
    AnnotationResultCount  = "result_count"   // int, serialised as string
    AnnotationKBRejections = "kb_rejections"  // int, serialised as string
    AnnotationEntityType   = "entity_type"    // e.g. "task", "feature"
    AnnotationDocType      = "doc_type"       // e.g. "specification", "design"
)
```

**Why not add typed fields to `Entry` directly?** The existing `Entry` struct
has seven fields, all of which are populated for every call by the outer hook. A
typed field for `result_count` would be set for three handlers and zero for the
remaining thirty-plus. The `extra` map preserves the lean schema for the common
case (most entries have no payload annotation) while allowing sparse, typed
annotation where it adds value.

The `extra` map is bounded in practice: only handler code (trusted, not user input) writes
to it, and the set of keys is limited to the constants defined in this package. No size cap
or validation is needed at the `Hook.Wrap` level.

### Component 3: Initial annotation callers

Three handlers are instrumented in this release, selected because each produces
a zero-result scenario with clear diagnostic value:

| Handler | Annotation key | Value |
|---------|---------------|-------|
| `entity(action: list)` | `result_count` | count of entities returned |
| `knowledge(action: list)` | `result_count` | count of entries returned |
| `doc_intel(action: search)` | `result_count` | count of sections returned |

Each handler calls `actionlog.AnnotateEntry(ctx, actionlog.AnnotationResultCount,
strconv.Itoa(n))` after computing its result, before returning. The annotation
is fire-and-forget — no return value, no error handling required.

### Component 4: Knowledge rejection capture

`Hook.Wrap` already drains side effects indirectly (via the `SideEffectCollector`
on the context). Extend `Hook.Wrap` to inspect the drained side effects for
`SideEffectKnowledgeRejected` entries, count them, and write the count into
`Extra[AnnotationKBRejections]` when non-zero.

This requires no changes to the `finish` handler or the knowledge service. The
hook sees the side effects after the handler returns but before writing the entry.

**Scope:** Only `SideEffectKnowledgeRejected` is captured this way. Other side
effect types remain response-only. The log is not a side-effect mirror.

**Failure mode — annotation loss on log write failure:** If the log writer fails (disk full,
permission error), the `Entry` including any drained annotations is discarded per FR-018.
Annotations are fire-and-forget; this silent loss is acceptable because log writes are
best-effort and the annotation data has no correctness impact on tool responses.

**Version string safety:** The `version` variable is set at link time via
. Semver strings contain only alphanumerics, dots, and
hyphens — all safe in JSON without escaping. No validation is required.

### Component 5: New `ComputeMetrics` aggregations

Three new aggregations are added to `MetricsResult`. All three are computed
from existing log data — no schema changes unlock them; they simply have not
been written yet.

**5a. Action distribution**

```go
type ActionDistribution struct {
    Tool    string `json:"tool"`
    Action  string `json:"action"` // empty string if tool has no action param
    Calls   int    `json:"calls"`
    Failures int   `json:"failures"`
}
```

Computed by grouping log entries by `(tool, action)` and counting total entries
and entries where `success == false`. Sorted by `calls` descending. This
surfaces dead actions (zero calls), high-failure actions (candidates for tool
description improvement), and the relative frequency of every tool invocation
pattern.

**5b. Document approval funnel**

```go
type DocTypeFunnel struct {
    DocType    string  `json:"doc_type"`   // resolved via StageFeatureLookup.DocType
    Registered int     `json:"registered"`
    Approved   int     `json:"approved"`
    Rate       float64 `json:"rate"`
}
```

Computed by finding all `doc` entries with `action == "register"` and all with
`action == "approve"`, grouping by `entity_id`. For each registered document,
check whether an approval entry exists for the same `entity_id` within the time
window. The `doc_type` field is not in the log today; it is inferred from the
it is resolved at aggregation time by extending the existing `StageFeatureLookup`
interface with a `DocType(entityID string) (string, error)` method. This keeps
document type resolution behind the same lookup abstraction already used for
stage resolution, avoiding a second mechanism.

**5c. Task completion gap**

```go
type TaskCompletionGap struct {
    Median float64 `json:"median_hours"`
    P90    float64 `json:"p90_hours"`
    Count  int     `json:"count"`
}
```

Computed by finding paired `next` entries (where the task transitions to active)
and `finish` entries for the same `entity_id`. The gap is the time between the
`next` call timestamp and the `finish` call timestamp. This does not require any
new log data — both events already produce log entries with the task's
`entity_id`.

---

## Alternatives Considered

### A. OpenTelemetry or Prometheus

Both frameworks are designed for distributed systems with multiple services,
high cardinality dimensions, and external metric consumers (dashboards, alerting
pipelines). Kanbanzai is a local, single-user MCP server. The operational
overhead of running a metrics collector sidecar or exporting to an OTLP endpoint
is not justified. The JSONL-per-day approach already provides the analytical
capability needed at a fraction of the complexity.

**Rejected:** Wrong problem domain. Revisit only if Kanbanzai becomes a
multi-tenant hosted service.

### B. Add typed fields to `Entry` for every signal

Instead of a sparse `extra` map, add `ResultCount *int`, `EntityType *string`,
`DocType *string`, `KBRejections *int` directly to `Entry`. This is strongly
typed and avoids string serialisation of integers.

The cost is schema bloat: most entries would carry four additional `null` fields.
The `Entry` struct is written for every tool call — adding nullable fields that
are populated for 3 of 35+ handlers imposes a permanent documentation and
maintenance burden (every new engineer asks why these fields exist on most
entries). The `extra` map keeps the common case clean and the sparse case
explicit.

**Rejected:** Schema hygiene. The `extra` map is the right abstraction for
sparse, handler-specific annotations.

### C. Write version to a session-start record only

Instead of stamping `server_version` on every entry, write a single "session
start" sentinel record when the server initialises. Version-to-timestamp mapping
can then be joined against other entries.

This is cheaper per-entry (one write at startup rather than a field on every
entry). The cost is query complexity: to find all entries produced by v2.1, the
analyst must first find the session-start record for v2.1, extract its timestamp,
then find all entries after that timestamp and before the next session-start with
a different version. Restarts, upgrades, and concurrent sessions make this join
fragile.

Per-entry version is ten bytes of JSON overhead per call. Given that log files
are date-partitioned and the server handles on the order of hundreds of calls per
day, this is not a storage concern.

**Rejected:** Query complexity is not justified by the marginal storage saving.

### D. Extend `ComputeMetrics` only — no schema changes

Compute action distribution and doc funnels from the existing log schema, and
defer version tagging and payload annotation to a later design.

This delivers the low-hanging fruit (aggregations over existing data) immediately
and avoids any risk from schema extension. The cost is that the most valuable
signal — zero-result detection — cannot be added without the `extra` map, and
version correlation cannot be added without `server_version`. These are not
hypothetical future needs; they are identified gaps with specific analytical
questions attached.

**Rejected:** Delivers less than half the value for most of the planning effort.
The schema changes are small and additive; there is no good reason to stage them.

### E. Status quo

The current system already records gate failures, stage durations, and revision
cycles. These have proved useful. The argument for doing nothing is that the
existing metrics may be sufficient to drive improvement decisions without the
additional signals.

The counter-argument is concrete: we cannot currently tell whether the
`knowledge` tool's 13 actions are all being used, or whether zero-result
`entity(action: list)` calls are occurring. These are specific questions with
identified answers, not speculative future needs. The implementation cost is
low relative to the analytical value.

**Rejected:** The identified blind spots have specific, answerable questions
attached. The cost of remaining blind is higher than the cost of the change.

---

## Decisions

**Decision 1: `extra map[string]string` over typed fields**
- **Context:** Payload-level annotations are sparse — relevant for 3 of 35+
  handlers today, and potentially a handful more in the future.
- **Rationale:** Typed fields on `Entry` impose a null-field maintenance burden
  on the common case. The `extra` map keeps the schema stable as new annotations
  are added, and the constant-defined key names prevent the map from becoming a
  free-form grab-bag.
- **Consequences:** Integer values (result count, rejection count) are serialised
  as strings. Aggregation code must parse them. This is a minor cost; the
  alternative (a typed map or a union type) adds generics complexity to a package
  that currently has none.

**Decision 2: Context-carried annotation, not handler return values**
- **Context:** The `Hook.Wrap` outer handler signature is `(ctx, req) →
  (*CallToolResult, error)`. It cannot see inside the returned result without
  coupling to each handler's response format.
- **Rationale:** The `PushSideEffect` / context pattern is already established
  in this codebase for exactly this problem — communicating from inside a handler
  to the outer middleware without changing the handler's return type. Reusing the
  same pattern keeps the annotation API consistent with existing middleware
  conventions.
- **Consequences:** Handlers that want to annotate must have access to the
  context. All existing handlers already receive a `context.Context`. No
  interface changes are required.

**Decision 3: Annotation keys as package-level constants**
- **Context:** If annotation keys are ad-hoc strings, different handlers will use
  different key names for the same concept (`"count"`, `"result_count"`,
  `"num_results"`), making aggregation brittle.
- **Rationale:** A small set of typed constants in `actionlog` gives the compiler
  a chance to catch typos and makes the intended key vocabulary explicit to future
  implementers.
- **Consequences:** Adding a new annotation key requires a constant in `actionlog`.
  This is a minor friction cost that pays for itself by preventing silent key drift.

**Decision 4: Side-effect inspection for KB rejections, not a separate log call**
- **Context:** Knowledge rejections are already communicated as
  `SideEffectKnowledgeRejected` via the side-effect collector on the context.
  Adding a separate `actionlog.AnnotateEntry` call in the `finish` handler would
  duplicate the mechanism.
- **Rationale:** The hook already has access to the side-effect collector (it is
  on the same context). Counting `SideEffectKnowledgeRejected` entries in
  `Hook.Wrap` before writing the log entry avoids any change to the `finish`
  handler.
- **Consequences:** The hook's `Wrap` method needs the count of
  `SideEffectKnowledgeRejected` events. The design intent is to count them
  via a type-assertion against the side-effect interface from the context
  collector, using only the string constant `"SideEffectKnowledgeRejected"`
  (already public as a side-effect type name). This avoids importing
  `internal/mcp` into `internal/actionlog`

**Decision 5: Task completion gap computed from existing log data**
- **Context:** `next(id)` and `finish(task_id)` both produce log entries with
  the same `entity_id`. The timestamps are present. No new instrumentation is
  needed.
- **Rationale:** Computing the gap in `ComputeMetrics` at query time is cheaper
  than storing it at write time, and it allows historical data to be included
  retrospectively without log rewriting.
- **Consequences:** The aggregation requires finding paired entries across the
  full log window, which is O(n) in the number of entries. For the expected log
  volumes (hundreds of entries per day, 30-day default window), this is
  negligible.

---

## Dependencies

- `cmd/kanbanzai/main.go` — must pass the `version` string to `NewServer`.
  `NewServer`'s signature changes: `NewServer(entityRoot, version string)`.
  The CLI's `runServe` and `runMetrics` paths must be updated accordingly.
- `internal/mcp/server.go` — `newServerWithConfig` receives the version and
  passes it to `actionlog.NewHook`.
- `internal/actionlog` — `Hook`, `Entry`, and writer are modified. `NewHook`
  gains a `version string` parameter.
- `internal/mcp/entity_tool.go`, `internal/mcp/knowledge_tool.go`,
  `internal/mcp/doc_intel_tool.go` — the three initial annotation callers.
  Each adds a single `AnnotateEntry` call in the list/search action handler.
- `internal/mcp/sideeffect.go` and `internal/actionlog/hook.go` — the
  KB-rejection capture requires the hook to read
  `SideEffectKnowledgeRejected` events. Coupling boundary to be resolved in
  the specification.
- No external dependencies are introduced. No new packages are required.