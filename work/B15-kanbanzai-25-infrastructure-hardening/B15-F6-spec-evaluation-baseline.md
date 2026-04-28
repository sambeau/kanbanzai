# Specification: Evaluation Baseline

| Field    | Value                                                              |
|----------|--------------------------------------------------------------------|
| Status   | Draft                                                              |
| Created  | 2026-04-01                                                         |
| Updated  | 2026-04-01                                                         |
| Feature  | FEAT-01KN4ZPXXEG1F (evaluation-baseline)                          |
| Design   | `work/design/kanbanzai-2.5-infrastructure-hardening.md` §9        |

---

## 1. Purpose

This specification defines the requirements for a structured evaluation baseline
that measures current Kanbanzai agent behaviour before the V3.0 redesign. The
baseline provides the "before" data against which V3.0 improvements can be
objectively compared.

---

## 2. Goals

1. A set of 15–20 representative workflow scenarios exists in machine-readable
   form before V3.0 implementation begins.
2. Each scenario fully specifies its starting state, expected interaction
   pattern, and success criteria in a consistent structure.
3. A baseline measurement is captured for at least 5 representative scenarios
   against the current (pre-V3.0) system.
4. The scenario format and baseline data are structured so that V3.0 evaluation
   runs can be compared against the baseline without reformatting.

---

## 3. Scope

### 3.1 In Scope

- `work/eval/` directory containing scenario YAML files and a README.
- Scenario YAML schema definition (documented in the README).
- Baseline measurement records for a minimum of 5 scenarios.
- Coverage of all six scenario categories defined in the design (§9.2).

### 3.2 Out of Scope

- Automated test execution tooling or scripts that run scenarios against the
  live MCP server.
- Integration of scenario runs into CI pipelines.
- V3.0 post-implementation evaluation runs (those belong to the V3.0 phase).
- Changes to any MCP tool, service, or storage layer.
- Scenarios covering V3.0-only behaviour (stage gates, new skills, etc.).

---

## 4. Scenario File Requirements

### 4.1 Location and Format

**REQ-01.** All scenario files MUST reside in `work/eval/`.

**REQ-02.** Each scenario MUST be a single YAML file named `eval-NNN.yaml` where
`NNN` is a zero-padded three-digit sequence number (e.g., `eval-001.yaml`).

**REQ-03.** Scenario files MUST NOT be committed to `.kbz/` state — they are
documentation artefacts, not workflow entities.

### 4.2 Required Fields

**REQ-04.** Every scenario file MUST contain the following top-level fields:

| Field               | Type            | Description                                      |
|---------------------|-----------------|--------------------------------------------------|
| `id`                | string          | Unique identifier matching the filename (e.g., `eval-001`) |
| `name`              | string          | Human-readable scenario name                     |
| `description`       | string          | Prose description of what the scenario tests     |
| `category`          | string          | One of the six categories defined in §4.4        |
| `starting_state`    | mapping         | The system state at the start of the scenario    |
| `expected_pattern`  | sequence        | Ordered list of stages with expected tools and outputs |
| `success_criteria`  | sequence        | List of assertions that define a passing run     |

**REQ-05.** The `starting_state` field MUST include at minimum:

- `feature_status`: the lifecycle status of the feature under test.
- `documents`: a list of documents pre-existing at scenario start (may be empty).
- `tasks`: a list of tasks pre-existing at scenario start (may be empty).

**REQ-06.** Each entry in `expected_pattern` MUST include:

- `stage`: a label describing the workflow stage.
- `tools`: a list of MCP tool names the agent is expected to call.
- `output`: a description of the artefact or state change expected at stage end.

**REQ-07.** Each entry in `success_criteria` MUST be a testable assertion
expressed as a plain-English string (e.g., "feature reaches done status").

### 4.3 Quantity

**REQ-08.** The `work/eval/` directory MUST contain between 15 and 20 scenario
files inclusive.

### 4.4 Category Coverage

**REQ-09.** The scenario set MUST include at least one scenario from each of the
following categories:

| Category              | Minimum count | Examples                                          |
|-----------------------|---------------|---------------------------------------------------|
| Happy path            | 3             | Full lifecycle, spec-only feature, plan-level spec |
| Gate failure + recovery | 3           | Missing spec, missing tasks, unapproved design    |
| Review-rework loop    | 2             | Single rework cycle, iteration cap reached        |
| Multi-feature plan    | 2             | Parallel features, cross-feature dependencies     |
| Edge cases            | 3             | No design stage, decompose failure, doc import    |
| Tool selection        | 2             | Correct tool for stage, avoids wrong-stage tools  |

---

## 5. README Requirements

**REQ-10.** A `work/eval/README.md` file MUST exist and MUST document:

- The purpose of the `work/eval/` directory.
- The complete YAML schema for scenario files, with field descriptions and
  allowed values.
- Instructions for conducting a manual evaluation run against the scenario set.
- The format used to record baseline measurement results.

---

## 6. Baseline Measurement Requirements

**REQ-11.** A baseline measurement MUST be captured for at least 5 scenarios
from the full set before this feature is considered complete.

**REQ-12.** Each baseline measurement record MUST capture:

| Field            | Description                                               |
|------------------|-----------------------------------------------------------|
| `scenario_id`    | The `id` of the scenario that was run                     |
| `system_version` | The Kanbanzai version (git SHA or tag) used for the run   |
| `run_date`       | ISO 8601 date of the evaluation run                       |
| `tools_called`   | Ordered list of MCP tools called during the run           |
| `final_state`    | Feature status and document states at the end of the run  |
| `success`        | Boolean — whether all success criteria were met           |
| `failures`       | List of success criteria that were not met (empty if passing) |
| `notes`          | Free-text observations (optional)                         |

**REQ-13.** Baseline measurement records MUST be stored in `work/eval/` in a
format that a V3.0 evaluation run can read and compare against without
reformatting.

**REQ-14.** Baseline measurements MUST be taken against the pre-V3.0 system —
i.e., before any V3.0 stage gate, tool description, or context assembly changes
are applied.

---

## 7. Constraints and Invariants

**REQ-15.** Scenario files are static documentation. No MCP tool reads or writes
them. They are evaluated by human or agent operators who run scenarios manually
and record results.

**REQ-16.** Scenario IDs MUST be unique across the full scenario set.

**REQ-17.** The `category` field MUST contain one of exactly the six values
listed in §4.4. No other values are permitted.

---

## 8. Acceptance Criteria

**AC-26.** 15–20 scenario files exist in `work/eval/` covering all six
categories defined in §4.4 (REQ-08, REQ-09).

**AC-27.** Each scenario file contains a defined `starting_state`,
`expected_pattern`, and `success_criteria` conforming to the field requirements
in §4.2 (REQ-04 through REQ-07).

**AC-28.** A baseline measurement is captured for at least 5 representative
scenarios against the current system, with records stored in `work/eval/`
(REQ-11 through REQ-14).

**AC-29.** The baseline data is stored in a format that V3.0 can compare against
without reformatting (REQ-13).

**AC-30.** `work/eval/README.md` exists and documents the schema, run
instructions, and measurement record format (REQ-10).

---

## 9. Dependencies and Assumptions

- No MCP tool changes are required. The evaluation baseline is purely
  documentation infrastructure.
- Baseline measurement runs require a working Kanbanzai MCP server connected to
  an AI agent capable of exercising the workflow.
- The YAML scenario format defined here is a starting point; V3.0 may extend it
  with additional fields (e.g., `automated`, `timeout_s`) without breaking
  backward compatibility, provided existing required fields are preserved.
- The 5-scenario minimum for baseline capture assumes representative coverage
  across happy-path and failure modes; the specific 5 scenarios are left to the
  implementer's judgment.