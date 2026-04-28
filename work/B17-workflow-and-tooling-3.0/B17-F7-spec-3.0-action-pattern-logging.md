# Specification: Action Pattern Logging and Metrics (Kanbanzai 3.0)

**Feature:** FEAT-01KN5-8J275BWJ (action-pattern-logging)
**Design reference:** `work/design/kanbanzai-3.0-workflow-and-tooling-v2.md` §12, §16.3 Q4, Q7
**Status:** Draft

---

## Overview

This specification defines the action pattern logging infrastructure, stage-level workflow metrics, and evaluation suite framework for Kanbanzai 3.0. The system logs MCP tool invocations as structured JSON lines to local files, derives higher-level workflow health metrics from the logs and entity timestamps, provides a CLI command for metric aggregation, and establishes a framework for representative workflow evaluation scenarios. Together, these capabilities enable measurement of whether agents follow the intended orchestration patterns and whether the 3.0 workflow changes are achieving their goals.

---

## Scope

### In scope

- JSON lines log format and schema for MCP tool invocations
- Log file location, naming convention, date-based rotation, and 30-day retention
- Automatic cleanup of expired log files at server startup
- Stage-level workflow metrics derived from logs and entity timestamps
- CLI command for metric aggregation and reporting
- Evaluation suite directory structure, scenario schema, and maintenance conventions
- Evaluation suite invocation documentation

### Explicitly excluded

- Analytics dashboards or visualization UIs
- Real-time alerting or notification systems
- CI integration for evaluation scenarios (evaluation requires live LLM agents)
- The `review_cycle` counter implementation (owned by the review-rework-loop feature; this feature reads that counter as a data source)
- Structural check implementation (owned by the document-structural-checks feature; this feature logs and derives metrics from structural check results)
- Hard tool filtering at runtime (owned by the role-scoped-tool-subsets feature; this feature measures compliance against declared subsets)
- Log shipping, remote storage, or centralised log aggregation
- Metric persistence beyond what is derived on demand from logs and entity timestamps
- Scenario content (specific evaluation scenarios are an implementation concern; this spec defines their structure)

---

## Functional Requirements

### Log Infrastructure

**FR-001:** The MCP server MUST log every MCP tool invocation to a structured log file. Each log entry MUST be a single JSON object written as one line (JSON lines format).

**Acceptance criteria:**
- Every MCP tool call handled by the server produces exactly one log entry
- Each log entry is a valid JSON object occupying exactly one line in the log file
- Log entries are appended atomically — a partial write or server crash does not corrupt previously written entries
- Log entries appear in chronological order within each file

---

**FR-002:** Each log entry MUST contain the following fields:

| Field | Type | Required | Description |
|---|---|---|---|
| `timestamp` | string (RFC 3339) | always | When the tool call was received |
| `tool` | string | always | MCP tool name (e.g., `entity`, `doc`, `status`) |
| `action` | string \| null | always | The `action` parameter value if the tool accepts one; `null` otherwise |
| `entity_id` | string \| null | always | The entity ID referenced by the call if applicable; `null` otherwise |
| `stage` | string \| null | always | The lifecycle stage of the referenced entity's parent feature at call time; `null` if no entity is referenced or the entity has no parent feature |
| `success` | boolean | always | `true` if the tool call completed without error; `false` otherwise |
| `error_type` | string \| null | always | If `success` is `false`: a classification of the error; `null` if `success` is `true` |

No additional fields are required. The schema MAY be extended with additional fields in future versions, but consumers MUST tolerate unknown fields.

**Acceptance criteria:**
- A successful `entity(action: "get", id: "FEAT-001")` call produces a log entry with `tool: "entity"`, `action: "get"`, `entity_id: "FEAT-001"`, `success: true`, `error_type: null`
- The `stage` field reflects the feature's current lifecycle stage at the time of the call, not after any transition the call may perform
- A tool that does not accept an `action` parameter (e.g., `status`) logs `action: null`
- A tool call that references no entity (e.g., `knowledge(action: "list")`) logs `entity_id: null`
- A failed call logs `success: false` with a non-null `error_type`
- All fields are present in every log entry — none are omitted, even when their value is `null`

---

**FR-003:** The `error_type` field MUST classify failures into one of the following categories: `gate_failure`, `validation_error`, `not_found`, `precondition_error`, `internal_error`. If a failure does not fit any specific category, `internal_error` MUST be used.

**Acceptance criteria:**
- A gate prerequisite failure (e.g., transitioning a feature without required documents) logs `error_type: "gate_failure"`
- An invalid parameter (e.g., malformed entity ID) logs `error_type: "validation_error"`
- A reference to a non-existent entity logs `error_type: "not_found"`
- A call that fails because a precondition is not met (e.g., finishing a task whose parent feature is in the wrong status) logs `error_type: "precondition_error"`
- An unexpected server-side error logs `error_type: "internal_error"`
- No other `error_type` values are produced

---

**FR-004:** The `entity_id` field MUST be extracted from the tool call parameters. For tools that accept an `id` parameter, use that value. For tools that accept an `entity_id` parameter, use that value. For tools that accept a `task_id` parameter, use that value. If the tool call references no entity, the field MUST be `null`.

**Acceptance criteria:**
- `entity(action: "get", id: "FEAT-001")` logs `entity_id: "FEAT-001"`
- `finish(task_id: "TASK-042")` logs `entity_id: "TASK-042"`
- `knowledge(action: "list")` logs `entity_id: null`
- `status(id: "FEAT-001")` logs `entity_id: "FEAT-001"`
- `status()` (no id) logs `entity_id: null`

---

**FR-005:** The `stage` field MUST be resolved by looking up the parent feature of the referenced entity and reading its current lifecycle stage. If the referenced entity is itself a feature, use that feature's stage. If the referenced entity is a task, use the parent feature's stage. If no entity is referenced or the entity has no parent feature, the field MUST be `null`.

**Acceptance criteria:**
- A call referencing `FEAT-001` which is in `specifying` logs `stage: "specifying"`
- A call referencing `TASK-042` whose parent feature `FEAT-001` is in `developing` logs `stage: "developing"`
- A call referencing no entity logs `stage: null`
- The stage reflects the state before any transition the current call may perform

---

### Log File Management

**FR-006:** Log files MUST be written to the `.kbz/logs/` directory. Each log file MUST be named `actions-YYYY-MM-DD.jsonl` where `YYYY-MM-DD` is the UTC date of the log entries. A new file MUST be created when the UTC date changes.

**Acceptance criteria:**
- Log entries generated on 2026-07-15 UTC are written to `.kbz/logs/actions-2026-07-15.jsonl`
- A log entry generated at 2026-07-15T23:59:59Z is written to the `2026-07-15` file; an entry generated at 2026-07-16T00:00:00Z is written to the `2026-07-16` file
- The `.kbz/logs/` directory is created automatically if it does not exist when the first log entry is written
- Log files are plain text files readable with standard tools (`cat`, `grep`, `jq`)

---

**FR-007:** The MCP server MUST perform automatic cleanup of log files older than 30 days at server startup. Cleanup MUST delete any file in `.kbz/logs/` matching the `actions-YYYY-MM-DD.jsonl` pattern whose date is more than 30 days before the current UTC date. Files that do not match the naming pattern MUST NOT be deleted.

**Acceptance criteria:**
- On startup on 2026-08-15, a file named `actions-2026-07-15.jsonl` (31 days old) is deleted
- On startup on 2026-08-15, a file named `actions-2026-07-16.jsonl` (30 days old) is retained
- A file named `notes.txt` in `.kbz/logs/` is not deleted regardless of age
- If `.kbz/logs/` does not exist, cleanup completes without error
- Cleanup failures on individual files (e.g., permission errors) are logged but do not prevent server startup

---

**FR-008:** The `.kbz/logs/` directory MUST be local-only and MUST NOT be committed to version control. The project's `.gitignore` (or equivalent `.kbz/` exclusion) MUST cover this directory.

**Acceptance criteria:**
- `.kbz/logs/` is covered by the existing `.kbz/` gitignore exclusion
- No log files appear in `git status` output

---

### Stage-Level Workflow Metrics

**FR-009:** The system MUST provide a CLI command (`kbz metrics`) that computes and displays stage-level workflow metrics. The command MUST accept an optional date range filter (defaulting to the last 30 days) and an optional feature ID filter.

**Acceptance criteria:**
- `kbz metrics` produces output covering all features with activity in the last 30 days
- `kbz metrics --since 2026-07-01 --until 2026-07-31` limits output to the specified date range
- `kbz metrics --feature FEAT-001` limits output to a single feature
- Date range and feature filters may be combined
- The command exits with a non-zero status if the logs directory does not exist or contains no data in the requested range, with a message explaining the situation

---

**FR-010:** The `kbz metrics` command MUST compute and report the following metrics:

1. **Time per stage:** For each lifecycle stage, the median and p90 duration that features spend in that stage, computed from entity transition timestamps.
2. **Revision cycle count:** For each completed feature, the number of review-rework cycles, read from the feature's `review_cycle` counter.
3. **Gate failure rate:** The number and rate of gate failures by error type, computed from tool call log entries where `error_type` is `gate_failure`.
4. **Structural check pass rate:** The proportion of document structural checks that pass on first attempt, computed from document evaluation results recorded in tool call logs.
5. **Tool subset compliance:** The number and rate of tool calls that invoke tools excluded for the current stage, computed by cross-referencing tool call log entries (the `tool` and `stage` fields) against the declared tool subsets in the binding registry.

**Acceptance criteria:**
- Time per stage is reported for every stage that has at least one feature transition in the date range
- Time per stage reports both median and 90th percentile durations
- Revision cycle count is reported per feature with at least one review cycle
- Gate failure rate is reported as both an absolute count and a percentage of total tool calls
- Structural check pass rate is reported as a percentage
- Tool subset compliance is reported as both a count of violations and a percentage of total calls within each stage
- If a metric has no data in the requested range, it is reported as having no data rather than being omitted

---

**FR-011:** The `kbz metrics` command MUST output results in a human-readable text format by default. The command MUST also accept a `--json` flag that outputs the same metrics as a single JSON object.

**Acceptance criteria:**
- Default output is formatted text readable in a terminal
- `kbz metrics --json` outputs a valid JSON object containing all computed metrics
- Both output formats contain the same metric data
- JSON output can be piped to `jq` for further processing

---

**FR-012:** Time-per-stage metrics MUST be computed from entity transition timestamps already recorded on feature entities, not from the action log. The metrics command MUST read feature entity data to derive these durations.

**Acceptance criteria:**
- Time per stage is accurate even if action log files have been cleaned up, provided the feature entities still exist
- The duration for a stage is measured from the timestamp of the transition into that stage to the timestamp of the transition out of it
- Features currently in a stage (no exit transition yet) are excluded from the completed-duration statistics but MAY be reported separately as "in progress"

---

### Evaluation Suite Framework

**FR-013:** The project MUST contain an evaluation suite directory at `work/evaluation/`. This directory MUST contain a `README.md` documenting the purpose of the suite, how to invoke it, and how to interpret results.

**Acceptance criteria:**
- The directory `work/evaluation/` exists and is committed to version control
- `work/evaluation/README.md` exists and documents invocation method and result interpretation
- The README documents that the suite requires live LLM agents and is not CI-gated

---

**FR-014:** Each evaluation scenario MUST be a YAML file in `work/evaluation/scenarios/`. Each scenario file MUST define the following fields:

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | yes | Human-readable scenario name |
| `description` | string | yes | What this scenario tests |
| `category` | string | yes | One of: `happy-path`, `gate-failure-recovery`, `review-rework-loop`, `multi-feature-orchestration`, `edge-case` |
| `starting_state` | object | yes | The initial state: feature lifecycle stage, existing documents, existing tasks |
| `expected_pattern` | object | yes | Expected tool call sequence: which tools should be called, in what general order |
| `success_criteria` | list | yes | Verifiable conditions that define scenario success |

**Acceptance criteria:**
- A scenario file containing all required fields with valid types loads without error
- A scenario file missing any required field is rejected with an error identifying the missing field
- The `category` field is validated against the five permitted values
- Scenario files use the `.yaml` extension

---

**FR-015:** The evaluation suite MUST contain between 15 and 20 scenarios. The scenarios MUST collectively cover all five categories defined in FR-014.

**Acceptance criteria:**
- The `work/evaluation/scenarios/` directory contains between 15 and 20 `.yaml` files
- At least one scenario exists for each of the five categories: `happy-path`, `gate-failure-recovery`, `review-rework-loop`, `multi-feature-orchestration`, `edge-case`

---

**FR-016:** Evaluation scenarios MUST be maintained alongside the workflow logic they test. When a workflow change is made (e.g., a gate is added, a tool description changes, a stage binding is modified), any affected scenarios MUST be updated in the same commit.

**Acceptance criteria:**
- The `work/evaluation/README.md` documents the maintenance discipline: scenarios are updated with the workflow changes they cover
- Scenario version history is visible in the Git log alongside related workflow changes

---

### Logging Behaviour

**FR-017:** Logging MUST be performed synchronously as part of the tool call handling path, after the tool call completes and the result is determined, but before the response is returned to the caller. The log entry MUST reflect the actual outcome of the call.

**Acceptance criteria:**
- The `success` and `error_type` fields accurately reflect the outcome of the tool call
- If the server crashes after a tool call completes but before logging, the log entry for that call MAY be missing (crash consistency is not required beyond atomic line writes)
- The log write occurs after the tool call result is known, not before

---

**FR-018:** If log writing fails (e.g., disk full, permission error), the tool call MUST still succeed and return its result to the caller. Log write failures MUST NOT cause tool call failures. Log write failures SHOULD be reported via server-side error logging (stderr or equivalent).

**Acceptance criteria:**
- A tool call succeeds and returns the correct result even if the log file cannot be written
- A log write failure is reported to server-side error output
- Repeated log write failures do not degrade tool call performance (no retry loops or backoff on log writes)

---

## Non-Functional Requirements

**NFR-001:** Logging MUST NOT add perceptible latency to MCP tool calls. The logging overhead MUST be less than 5 milliseconds per tool call under normal operating conditions (local filesystem, non-degraded disk).

**Acceptance criteria:**
- Benchmark tool calls with and without logging enabled; the difference is less than 5ms at the 99th percentile
- Log writes use buffered I/O or equivalent to minimise system call overhead

---

**NFR-002:** Log files MUST be usable with standard Unix text-processing tools (`grep`, `cat`, `head`, `tail`, `wc`, `jq`). No proprietary format or special tooling is required to read or query log files.

**Acceptance criteria:**
- `cat .kbz/logs/actions-2026-07-15.jsonl | jq .tool` extracts all tool names from a day's log
- `grep gate_failure .kbz/logs/actions-*.jsonl | wc -l` counts total gate failures across all log files
- Log files do not use compression, binary encoding, or multi-line JSON entries

---

**NFR-003:** The `kbz metrics` command MUST complete within 10 seconds for a 30-day window containing up to 100,000 log entries across all log files.

**Acceptance criteria:**
- The command completes within 10 seconds when processing 100,000 log entries
- Performance scales linearly (or better) with log volume within the expected range

---

**NFR-004:** The logging infrastructure MUST be compatible with the existing `.kbz/` directory structure and MUST NOT conflict with other `.kbz/` subsystems (state, config, etc.).

**Acceptance criteria:**
- The `.kbz/logs/` directory does not conflict with any existing `.kbz/` subdirectory
- Existing server functionality is unaffected when the logs directory is present or absent

---

**NFR-005:** The evaluation suite MUST be usable without any external dependencies beyond the Kanbanzai MCP server and a compatible LLM agent. No additional test frameworks, databases, or services are required to run scenarios.

**Acceptance criteria:**
- The `work/evaluation/README.md` documents a self-contained invocation method
- Running the suite requires only the kanbanzai binary and an LLM agent session

---

## Dependencies and Assumptions

1. **Entity transition timestamps:** Time-per-stage metrics (FR-010, FR-012) depend on feature entities recording timestamps for each lifecycle transition. This is assumed to be an existing capability of the entity system.

2. **Review cycle counter:** The revision cycle count metric (FR-010) depends on the `review_cycle` counter on feature entities. This counter is implemented by the review-rework-loop feature (design §4.2). If that feature is not yet implemented, the revision cycle count metric reports no data.

3. **Structural check results:** The structural check pass rate metric (FR-010) depends on document structural checks being performed and their results being observable in tool call logs (as success/failure outcomes of transition calls that trigger checks). This is implemented by the document-structural-checks feature (design §10.4). If that feature is not yet implemented, this metric reports no data.

4. **Binding registry tool subsets:** The tool subset compliance metric (FR-010) depends on the binding registry declaring per-stage tool subsets (design §9.4). The metric cross-references tool call logs against these declared subsets. If the binding registry does not yet declare tool subsets, this metric reports no data.

5. **`.kbz/` directory:** The `.kbz/` directory exists and is writable. The `.kbz/logs/` subdirectory is created on demand (FR-006).

6. **`.kbz/` gitignore exclusion:** The `.kbz/` directory (or `.kbz/logs/` specifically) is excluded from version control by an existing gitignore rule (FR-008).

7. **MCP server architecture:** The MCP server has a centralised tool dispatch path where logging can be injected without modifying each individual tool handler (FR-001, FR-017, FR-018).