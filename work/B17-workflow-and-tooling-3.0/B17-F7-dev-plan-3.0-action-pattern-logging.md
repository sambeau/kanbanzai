# Implementation Plan: Action Pattern Logging and Metrics (Kanbanzai 3.0)

**Specification:** `work/spec/3.0-action-pattern-logging.md`
**Feature:** FEAT-01KN5-8J275BWJ (action-pattern-logging)
**Design reference:** `work/design/kanbanzai-3.0-workflow-and-tooling-v2.md` §12, §16.3 Q4, Q7

---

## Overview

This plan decomposes the action pattern logging specification into assignable tasks for AI agents. The feature adds three capabilities: (1) structured JSON lines logging of every MCP tool invocation, (2) a `kbz metrics` CLI command that derives stage-level workflow health metrics from logs and entity timestamps, and (3) an evaluation suite framework with scenario schema and directory structure.

The work divides into five layers: log entry model and writer (the foundational data format and I/O), MCP server integration (the hook that captures tool calls), log file management (rotation and retention), the metrics CLI command (aggregation and reporting), and the evaluation suite (directory structure and scenario schema). The logging infrastructure is foundational — all other layers depend on the log format being stable. The evaluation suite is independent of the logging code and can be built in parallel.

### Scope boundaries (from specification)

- **In scope:** JSON lines log format/schema, log file location and naming, date-based rotation, 30-day retention with startup cleanup, stage-level workflow metrics (5 metrics), `kbz metrics` CLI command with date range and feature filters, evaluation suite directory structure, scenario YAML schema, evaluation README
- **Out of scope:** Analytics dashboards, real-time alerting, CI integration for evaluation, `review_cycle` counter implementation, structural check implementation, hard tool filtering, log shipping/remote storage, metric persistence, scenario content

---

## Task Breakdown

### Task 1: Log entry model and writer

**Objective:** Define the Go struct for the 7-field log entry schema (FR-002), the error type classification constants (FR-003), and a writer that appends JSON lines atomically to a file. The writer must handle date-based file selection (FR-006), directory creation, and graceful failure (FR-018). This is the foundational data layer — every other task depends on its types and interfaces.

**Specification references:** FR-002, FR-003, FR-006 (file naming/location), FR-017 (synchronous, post-completion), FR-018 (log failures must not break tool calls), NFR-001 (< 5ms overhead), NFR-002 (standard Unix tool compatibility), NFR-004 (compatible with `.kbz/` structure)

**Input context:**
- `internal/core/paths.go` — `InstanceRootDir` constant (`.kbz`), path conventions
- `refs/go-style.md` — naming, error handling, package design conventions
- Spec §FR-002 for the 7-field schema: `timestamp`, `tool`, `action`, `entity_id`, `stage`, `success`, `error_type`
- Spec §FR-003 for error type enum: `gate_failure`, `validation_error`, `not_found`, `precondition_error`, `internal_error`
- Spec §FR-006 for file naming: `.kbz/logs/actions-YYYY-MM-DD.jsonl`

**Output artifacts:**
- New file `internal/actionlog/entry.go` containing `Entry` struct with JSON tags, `ErrorType` constants, and `NewEntry` constructor
- New file `internal/actionlog/writer.go` containing `Writer` struct with `Log(Entry) error` method, buffered I/O, date-based file selection, atomic line writes, and directory auto-creation
- New file `internal/actionlog/entry_test.go` with tests: JSON serialisation round-trip, all fields present (including nulls), RFC 3339 timestamp format, each error type constant value
- New file `internal/actionlog/writer_test.go` with tests: writes valid JSON lines, date rotation creates new file, directory auto-creation, write failure does not panic, concurrent writes produce valid lines

**Dependencies:** None — this is the foundation task.

**Interface contract (shared with Tasks 2, 3, 4):**

```go
package actionlog

import "time"

// Entry represents a single MCP tool invocation log record.
// All fields are always present in JSON output (nulls are not omitted).
type Entry struct {
    Timestamp string  `json:"timestamp"`            // RFC 3339
    Tool      string  `json:"tool"`                 // MCP tool name
    Action    *string `json:"action"`               // action parameter or null
    EntityID  *string `json:"entity_id"`            // referenced entity or null
    Stage     *string `json:"stage"`                // parent feature lifecycle stage or null
    Success   bool    `json:"success"`              // true if tool call succeeded
    ErrorType *string `json:"error_type"`           // error classification or null
}

// Error type constants (FR-003).
const (
    ErrorGateFailure      = "gate_failure"
    ErrorValidationError  = "validation_error"
    ErrorNotFound         = "not_found"
    ErrorPreconditionError = "precondition_error"
    ErrorInternalError    = "internal_error"
)

// Writer appends log entries to date-partitioned JSONL files.
type Writer struct { /* ... */ }

// NewWriter creates a writer that writes to the given logs directory.
// The directory is created on first write if it does not exist.
func NewWriter(logsDir string) *Writer

// Log appends an entry to the current day's log file.
// Returns an error if the write fails, but callers (the MCP hook)
// must not propagate this error to the tool call response (FR-018).
func (w *Writer) Log(e Entry) error

// LogsDir returns the canonical logs directory path.
func LogsDir() string  // returns filepath.Join(core.InstanceRootDir, "logs")
```

---

### Task 2: Entity ID extraction and stage resolution

**Objective:** Implement the logic that extracts the `entity_id` from tool call parameters (FR-004) and resolves the `stage` field by looking up the parent feature's lifecycle stage (FR-005). This is a pure function layer with no I/O dependencies beyond entity lookup — it translates MCP request parameters into the log entry's `entity_id` and `stage` fields.

**Specification references:** FR-004 (entity ID extraction from `id`, `entity_id`, `task_id` parameters), FR-005 (stage resolution via parent feature lookup)

**Input context:**
- `internal/mcp/sideeffect.go` — `DispatchAction`, `WithSideEffects`, `ActionHandler` — shows how tool call parameters are accessed via `req.Params.Arguments`
- `internal/mcp/entity_tool.go` — example of parameter extraction patterns
- `internal/model/entities.go` — `Feature` struct with `Status` field, `Task` struct with `ParentFeature` field
- `internal/service/entity_service.go` — `Get` method for loading entities
- Spec §FR-004 for parameter priority: `id` > `entity_id` > `task_id`
- Spec §FR-005 for stage resolution: feature → its own stage; task → parent feature's stage; no entity → null

**Output artifacts:**
- New file `internal/actionlog/extract.go` containing `ExtractEntityID(args map[string]any) *string` and `ResolveStage(entityID string, lookup StageLookup) *string`
- New file `internal/actionlog/extract_test.go` with table-driven tests: `id` parameter used, `entity_id` parameter used, `task_id` parameter used, no entity parameter returns nil, feature entity returns its own stage, task entity returns parent feature stage, unknown entity returns nil, no entity ID returns nil stage

**Dependencies:** Task 1 (uses `Entry` type and `*string` conventions).

**Interface contract (shared with Task 3):**

```go
// StageLookup abstracts entity loading for stage resolution.
// Defined at the consumer (actionlog package), not the provider.
type StageLookup interface {
    // GetEntityKindAndParent returns the entity kind ("feature", "task", etc.),
    // the parent feature ID (empty if none), and an error.
    GetEntityKindAndParent(entityID string) (kind string, parentFeatureID string, err error)
    // GetFeatureStage returns the current lifecycle stage of a feature.
    GetFeatureStage(featureID string) (string, error)
}

// ExtractEntityID extracts the entity ID from tool call parameters.
// Checks "id", then "entity_id", then "task_id". Returns nil if none found.
func ExtractEntityID(args map[string]any) *string

// ResolveStage determines the lifecycle stage for a log entry.
// If entityID is nil, returns nil. If the entity is a feature, returns its stage.
// If the entity is a task, returns its parent feature's stage.
// Returns nil on any lookup failure (does not propagate errors).
func ResolveStage(entityID *string, lookup StageLookup) *string
```

---

### Task 3: MCP server logging hook

**Objective:** Integrate the action log writer into the MCP server's tool dispatch path so that every tool invocation produces exactly one log entry (FR-001). The hook must execute synchronously after the tool call completes and before the response is returned (FR-017). Log write failures must not affect tool call results (FR-018). The hook must add less than 5ms overhead (NFR-001).

**Specification references:** FR-001 (log every invocation), FR-004 (entity ID extraction, delegated to Task 2), FR-005 (stage resolution, delegated to Task 2), FR-017 (synchronous, post-completion logging), FR-018 (log failures do not break tool calls), NFR-001 (< 5ms overhead)

**Input context:**
- `internal/mcp/server.go` — `NewServer`, `newServerWithConfig`, `Serve` — where the MCP server is constructed and tools are registered; the logging hook needs to wrap tool handlers or be injected into the dispatch path
- `internal/mcp/sideeffect.go` — `WithSideEffects` wrapper pattern — the logging hook should wrap around or compose with this existing wrapper
- `internal/service/entity_service.go` — provides entity loading for stage resolution (Task 2's `StageLookup` interface)
- Spec dependency §7: "The MCP server has a centralised tool dispatch path where logging can be injected without modifying each individual tool handler"

**Output artifacts:**
- New file `internal/actionlog/hook.go` containing the logging middleware that wraps tool handlers, captures tool name, parameters, success/error, and delegates to the writer
- New file `internal/actionlog/classify.go` containing `ClassifyError(err error) string` that maps errors to the FR-003 error type constants
- Modified `internal/mcp/server.go` — construct `actionlog.Writer` in `newServerWithConfig`, wire it into the tool dispatch path
- New file `internal/actionlog/hook_test.go` with tests: successful call logged with correct fields, failed call logged with error_type, log write failure does not affect tool result, action parameter captured for action-based tools, action is null for non-action tools
- New file `internal/actionlog/classify_test.go` with tests: gate failure errors classified correctly, validation errors, not-found errors, precondition errors, unknown errors default to internal_error

**Dependencies:** Task 1 (writer), Task 2 (extraction and stage resolution).

**Interface contract (shared with Task 1, Task 2):**

```go
// Hook wraps an MCP tool handler to log every invocation.
type Hook struct {
    writer *Writer
    lookup StageLookup // may be nil if entity service unavailable
}

// NewHook creates a logging hook.
func NewHook(writer *Writer, lookup StageLookup) *Hook

// Wrap returns a wrapped handler that logs the tool invocation after
// the inner handler completes. The tool name is provided at wrap time.
func (h *Hook) Wrap(toolName string, inner func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error)) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error)

// ClassifyError maps an error to one of the FR-003 error type constants.
// Returns ErrorInternalError if the error does not match any specific category.
func ClassifyError(err error) string
```

---

### Task 4: Log file cleanup at startup

**Objective:** Implement automatic cleanup of log files older than 30 days at MCP server startup (FR-007). Cleanup must only delete files matching the `actions-YYYY-MM-DD.jsonl` naming pattern, must tolerate missing directories and individual file deletion failures, and must not prevent server startup.

**Specification references:** FR-007 (30-day retention, startup cleanup, pattern matching, error tolerance), FR-008 (`.kbz/logs/` is local-only, covered by existing `.kbz/` gitignore)

**Input context:**
- `internal/actionlog/writer.go` (from Task 1) — `LogsDir()` function for directory path
- `internal/mcp/server.go` — `Serve()` function where startup cleanup should be triggered
- `.gitignore` — verify `.kbz/logs/` is covered by existing `.kbz/cache/` pattern or add `.kbz/logs/` explicitly

**Output artifacts:**
- New file `internal/actionlog/cleanup.go` containing `Cleanup(logsDir string, now time.Time) error` that scans the logs directory, identifies files matching the `actions-YYYY-MM-DD.jsonl` pattern, deletes those older than 30 days, and logs (to stderr) any per-file deletion failures without returning an error
- New file `internal/actionlog/cleanup_test.go` with tests: files older than 30 days deleted, files exactly 30 days old retained, non-matching filenames not deleted, missing directory does not error, individual deletion failure logged but does not abort
- Modified `internal/mcp/server.go` — call `actionlog.Cleanup` in `Serve()` before starting the server
- Modified `.gitignore` — add `.kbz/logs/` if not already covered (verify FR-008)

**Dependencies:** Task 1 (uses `LogsDir()` and file naming conventions).

---

### Task 5: Metrics aggregation service

**Objective:** Implement the metrics computation engine that reads log files and entity data to produce the five metrics defined in FR-010. This is a pure computation layer — it reads log entries from files and entity data from the entity service, computes aggregates, and returns structured results. It does not handle CLI flags or output formatting.

**Specification references:** FR-010 (five metrics: time per stage, revision cycle count, gate failure rate, structural check pass rate, tool subset compliance), FR-012 (time per stage from entity timestamps, not logs)

**Input context:**
- `internal/actionlog/entry.go` (from Task 1) — `Entry` struct for log parsing
- `internal/model/entities.go` — `Feature` struct (status, created, updated fields), `FeatureStatus` constants for lifecycle stages
- `internal/service/entity_service.go` — `List` for loading features in a date range
- Spec §FR-010 for metric definitions: median/p90 durations, cycle counts, failure rates, compliance rates
- Spec §FR-012 for time-per-stage: derived from entity transition timestamps, not action logs
- Spec dependency §2: `review_cycle` counter may not exist yet — report no data
- Spec dependency §3: structural check results may not exist yet — report no data
- Spec dependency §4: binding registry tool subsets may not exist yet — report no data

**Output artifacts:**
- New file `internal/actionlog/reader.go` containing `ReadEntries(logsDir string, since, until time.Time) ([]Entry, error)` that reads and parses JSONL files in the date range
- New file `internal/actionlog/metrics.go` containing `ComputeMetrics(input MetricsInput) (*MetricsResult, error)` with the five metric computations
- New file `internal/actionlog/reader_test.go` with tests: reads entries from multiple date files, skips files outside range, handles malformed lines gracefully, empty directory returns empty slice
- New file `internal/actionlog/metrics_test.go` with tests: time per stage median/p90, gate failure rate calculation, empty data returns no-data indicators, feature filter limits scope

**Dependencies:** Task 1 (entry model and file naming conventions).

**Interface contract (shared with Task 6):**

```go
// MetricsInput configures the metrics computation.
type MetricsInput struct {
    LogsDir   string
    Since     time.Time
    Until     time.Time
    FeatureID string // optional filter; empty means all features
}

// MetricsResult contains all computed metrics.
type MetricsResult struct {
    TimePerStage          []StageDuration      `json:"time_per_stage"`
    RevisionCycleCounts   []FeatureCycleCount  `json:"revision_cycle_counts"`
    GateFailureRate       GateFailureMetric    `json:"gate_failure_rate"`
    StructuralCheckRate   *PassRateMetric      `json:"structural_check_pass_rate"` // nil = no data
    ToolSubsetCompliance  *ComplianceMetric    `json:"tool_subset_compliance"`     // nil = no data
}

// StageFeatureLookup abstracts entity loading for metrics computation.
type StageFeatureLookup interface {
    // ListFeaturesInRange returns features with activity in the date range.
    ListFeaturesInRange(since, until time.Time, featureID string) ([]FeatureMetricsData, error)
}

// FeatureMetricsData is the subset of feature data needed for metrics.
type FeatureMetricsData struct {
    ID             string
    Status         string
    Created        time.Time
    Updated        time.Time
    ReviewCycles   int  // 0 if counter not available
    // Stage transition timestamps (ordered list of status + timestamp pairs)
    Transitions    []StatusTransition
}

type StatusTransition struct {
    Status    string
    EnteredAt time.Time
}

// StageDuration reports time spent in a single lifecycle stage.
type StageDuration struct {
    Stage    string        `json:"stage"`
    Median   time.Duration `json:"median_seconds"`
    P90      time.Duration `json:"p90_seconds"`
    Count    int           `json:"feature_count"`
}

// GateFailureMetric reports gate failure counts and rates.
type GateFailureMetric struct {
    Count      int     `json:"count"`
    TotalCalls int     `json:"total_calls"`
    Rate       float64 `json:"rate"`
    HasData    bool    `json:"has_data"`
}

// ReadEntries reads and parses log entries from JSONL files in the date range.
func ReadEntries(logsDir string, since, until time.Time) ([]Entry, error)

// ComputeMetrics computes all five metrics from logs and entity data.
func ComputeMetrics(input MetricsInput, lookup StageFeatureLookup) (*MetricsResult, error)
```

---

### Task 6: `kbz metrics` CLI command

**Objective:** Add the `kbz metrics` CLI command that accepts `--since`, `--until`, `--feature`, and `--json` flags (FR-009, FR-011), invokes the metrics aggregation service (Task 5), and formats output as either human-readable text or JSON. The command must complete within 10 seconds for 100,000 log entries (NFR-003).

**Specification references:** FR-009 (CLI command with date range and feature filters), FR-011 (human-readable default, `--json` flag), NFR-003 (10-second performance target)

**Input context:**
- `cmd/kanbanzai/main.go` — `run` function switch statement (add `case "metrics":`), `usageText` constant (add metrics to help text), `dependencies` struct pattern
- `cmd/kanbanzai/workflow_cmd.go` — existing CLI command pattern for reference
- `internal/actionlog/metrics.go` (from Task 5) — `ComputeMetrics` function and result types
- `internal/core/paths.go` — `InstanceRootDir` for constructing logs directory path

**Output artifacts:**
- New file `cmd/kanbanzai/metrics_cmd.go` containing `runMetrics(args []string, deps dependencies) error` with flag parsing, service invocation, and both text and JSON formatters
- Modified `cmd/kanbanzai/main.go` — add `case "metrics":` to `run` switch, add metrics to `usageText`
- New file `cmd/kanbanzai/metrics_cmd_test.go` with tests: default 30-day range, custom date range, feature filter, JSON output is valid JSON, text output is human-readable, no data returns informative message with non-zero exit, combined flags

**Dependencies:** Task 5 (metrics aggregation service).

---

### Task 7: Evaluation suite framework

**Objective:** Create the evaluation suite directory structure, scenario YAML schema validation, and documentation (FR-013, FR-014, FR-015, FR-016). This task produces the framework and 15–20 scenario files covering all five categories. The scenarios define structure and expected patterns — they do not contain implementation logic.

**Specification references:** FR-013 (directory structure and README), FR-014 (scenario YAML schema with 6 required fields and 5 category values), FR-015 (15–20 scenarios covering all 5 categories), FR-016 (maintenance discipline documented)

**Input context:**
- Spec §FR-014 for scenario schema: `name`, `description`, `category`, `starting_state`, `expected_pattern`, `success_criteria`
- Spec §FR-014 for category enum: `happy-path`, `gate-failure-recovery`, `review-rework-loop`, `multi-feature-orchestration`, `edge-case`
- Spec §FR-015 for coverage: 15–20 scenarios, all 5 categories represented
- Spec §FR-016 for maintenance discipline: scenarios updated with workflow changes
- Spec §NFR-005: no external dependencies beyond kanbanzai binary and LLM agent

**Output artifacts:**
- New directory `work/evaluation/`
- New file `work/evaluation/README.md` documenting purpose, invocation method, result interpretation, maintenance discipline (FR-016), and self-contained requirement (NFR-005)
- New directory `work/evaluation/scenarios/`
- 15–20 new `.yaml` files in `work/evaluation/scenarios/` covering all 5 categories (at least 3 `happy-path`, 3 `gate-failure-recovery`, 3 `review-rework-loop`, 2 `multi-feature-orchestration`, 2 `edge-case`)
- New file `internal/actionlog/scenario.go` containing `Scenario` struct with YAML tags and `LoadScenario(path string) (*Scenario, error)` with field and category validation
- New file `internal/actionlog/scenario_test.go` with tests: valid scenario loads, missing required field rejected with named error, invalid category rejected, all 5 categories accepted

**Dependencies:** None — independent of logging code. Can run in parallel with Tasks 1–4.

---

## Dependency Graph

```
Task 1: Log entry model + writer     Task 7: Evaluation suite framework
   │                                      (independent, parallel)
   ├──► Task 2: Entity ID extraction + stage resolution
   │       │
   │       └──► Task 3: MCP server logging hook
   │
   ├──► Task 4: Log file cleanup at startup
   │
   └──► Task 5: Metrics aggregation service
            │
            └──► Task 6: `kbz metrics` CLI command
```

**Parallel execution opportunities:**
- Task 7 has no dependencies and can execute in parallel with all other tasks
- Tasks 2, 4, and 5 depend only on Task 1 and can execute in parallel once Task 1 completes
- Task 3 requires Tasks 1 + 2
- Task 6 requires Task 5

**Minimum serial path:** Task 1 → Task 2 → Task 3 (logging complete); Task 1 → Task 5 → Task 6 (metrics complete)

**Recommended execution order:**

| Phase | Tasks | Rationale |
|-------|-------|-----------|
| Phase 1 | Task 1, Task 7 | Foundation model + independent evaluation suite |
| Phase 2 | Task 2, Task 4, Task 5 | All depend only on Task 1; run in parallel |
| Phase 3 | Task 3, Task 6 | Hook needs Task 2; CLI needs Task 5 |

---

## Traceability Matrix

| Requirement | Task(s) | Notes |
|-------------|---------|-------|
| FR-001 | Task 3 | Every tool invocation logged via MCP hook |
| FR-002 | Task 1 | Entry struct with 7 fields |
| FR-003 | Task 1, Task 3 | Constants in Task 1; classification logic in Task 3 |
| FR-004 | Task 2 | `ExtractEntityID` from tool parameters |
| FR-005 | Task 2 | `ResolveStage` via parent feature lookup |
| FR-006 | Task 1 | Writer handles `.kbz/logs/actions-YYYY-MM-DD.jsonl` |
| FR-007 | Task 4 | Startup cleanup of files > 30 days |
| FR-008 | Task 4 | Verify/update `.gitignore` coverage |
| FR-009 | Task 6 | `kbz metrics` command with flags |
| FR-010 | Task 5 | Five metric computations |
| FR-011 | Task 6 | Text and JSON output formats |
| FR-012 | Task 5 | Time per stage from entity timestamps |
| FR-013 | Task 7 | `work/evaluation/` directory and README |
| FR-014 | Task 7 | Scenario YAML schema and validation |
| FR-015 | Task 7 | 15–20 scenarios across 5 categories |
| FR-016 | Task 7 | Maintenance discipline in README |
| FR-017 | Task 3 | Synchronous post-completion logging |
| FR-018 | Task 1, Task 3 | Writer returns error; hook swallows it |
| NFR-001 | Task 1, Task 3 | Buffered I/O in writer; benchmark in hook test |
| NFR-002 | Task 1 | Plain JSON lines, no compression |
| NFR-003 | Task 5, Task 6 | Linear scan; performance test in Task 6 |
| NFR-004 | Task 1 | `.kbz/logs/` does not conflict with existing dirs |
| NFR-005 | Task 7 | README documents self-contained invocation |