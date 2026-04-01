# work/eval — Evaluation Baseline

This directory contains the structured evaluation baseline for Kanbanzai workflow
agent behaviour. The baseline provides a "before" snapshot against which the V3.0
redesign can be objectively compared.

---

## Purpose

Before V3.0 implementation begins, a representative set of workflow scenarios is
captured in machine-readable form. Each scenario defines:

- The system state at the start of the scenario.
- The expected sequence of MCP tool calls the agent should make.
- The success criteria that determine whether a run passes.

After V3.0 ships, the same scenarios are re-run and the results are diffed against
the baseline measurements recorded here. Improvements, regressions, and unchanged
behaviour are all surfaced by this comparison.

---

## Directory Contents

| File / Pattern              | Description                                        |
|-----------------------------|----------------------------------------------------|
| `README.md`                 | This file — schema, run instructions, methodology  |
| `eval-NNN.yaml`             | Scenario definition files (NNN = 001 … 020)        |
| `baseline-eval-NNN-YYYYMMDD.yaml` | Baseline measurement records                 |

---

## Scenario File Schema

Each scenario file is named `eval-NNN.yaml` (zero-padded three-digit sequence
number) and contains the following top-level fields.

### Required Fields

```yaml
# Unique identifier — must match the filename without extension.
id: eval-001

# Human-readable scenario name.
name: "Full lifecycle happy path"

# Prose description of what the scenario tests and why it is representative.
description: >
  Tests the complete feature lifecycle from proposed through done, with a
  spec document, dev plan, tasks, and a review cycle. Exercises the most
  common agent workflow path.

# Scenario category — must be exactly one of the six allowed values:
#   happy-path | gate-failure-and-recovery | review-rework-loop |
#   multi-feature-plan | edge-case | tool-selection
category: happy-path

# The system state at the start of the scenario.
starting_state:
  # Lifecycle status of the feature under test at scenario start.
  feature_status: proposed

  # Documents pre-existing at scenario start (empty list is valid).
  documents:
    - path: work/design/example-feature.md
      type: design
      status: approved

  # Tasks pre-existing at scenario start (empty list is valid).
  tasks:
    - id: TASK-001
      status: queued
      summary: "Implement the core handler"

# Ordered list of workflow stages the agent is expected to traverse.
expected_pattern:
  - stage: "Advance to specifying"
    tools:
      - entity
      - doc
    output: "Feature transitions to specifying status; spec document registered"

  - stage: "Write and approve specification"
    tools:
      - doc
    output: "Specification document approved"

  - stage: "Generate tasks via decompose"
    tools:
      - decompose
      - entity
    output: "3-5 tasks created in queued status"

# Testable assertions — each is a plain-English string that can be evaluated
# as true/false against the observed final state.
success_criteria:
  - "Feature reaches done status"
  - "All tasks are in done status"
  - "At least one specification document is approved"
  - "No task summary matches a bare section heading"
```

### Field Descriptions

| Field                          | Type     | Required | Description                                                       |
|--------------------------------|----------|----------|-------------------------------------------------------------------|
| `id`                           | string   | yes      | Unique ID matching filename without `.yaml` extension             |
| `name`                         | string   | yes      | Concise human-readable label                                      |
| `description`                  | string   | yes      | Prose description of what the scenario tests                      |
| `category`                     | string   | yes      | One of the six allowed category values (see below)                |
| `starting_state`               | mapping  | yes      | System state at scenario start                                    |
| `starting_state.feature_status`| string   | yes      | Lifecycle status of the feature: proposed, designing, specifying, etc. |
| `starting_state.documents`     | sequence | yes      | Pre-existing documents (may be empty list)                        |
| `starting_state.tasks`         | sequence | yes      | Pre-existing tasks (may be empty list)                            |
| `expected_pattern`             | sequence | yes      | Ordered stages with expected tools and outputs                    |
| `expected_pattern[].stage`     | string   | yes      | Label describing the workflow stage                               |
| `expected_pattern[].tools`     | sequence | yes      | MCP tool names the agent is expected to call at this stage        |
| `expected_pattern[].output`    | string   | yes      | Description of the artefact or state change expected at stage end |
| `success_criteria`             | sequence | yes      | Testable assertion strings defining a passing run                 |

### Allowed Category Values

| Category                  | Description                                                |
|---------------------------|------------------------------------------------------------|
| `happy-path`              | All preconditions met; expected to complete successfully   |
| `gate-failure-and-recovery` | A lifecycle gate blocks progress; agent must recover     |
| `review-rework-loop`      | Implementation goes through at least one review cycle      |
| `multi-feature-plan`      | Multiple features interact or run in parallel              |
| `edge-case`               | Unusual or boundary conditions                             |
| `tool-selection`          | Focuses on the agent choosing the correct tool at a stage  |

### Valid Feature Lifecycle Statuses

`proposed` → `designing` → `specifying` → `dev-planning` → `developing` →
`reviewing` → `done`

Also valid: `blocked`, `not-planned`

### Known MCP Tool Names

`entity`, `doc`, `doc_intel`, `knowledge`, `decompose`, `handoff`, `next`,
`finish`, `checkpoint`, `status`, `estimate`, `conflict`, `retro`, `worktree`,
`merge`, `pr`, `branch`, `cleanup`, `health`

---

## Measurement Record Schema

Baseline measurement records are named `baseline-eval-NNN-YYYYMMDD.yaml` and
contain the following fields.

### Required Fields

```yaml
scenario_id: eval-001

# Git SHA or tag of the Kanbanzai binary used for the run.
system_version: "abc1234"

# ISO 8601 date of the evaluation run.
run_date: "2026-04-01"

# Ordered list of MCP tools called during the run (in call order).
tools_called:
  - next
  - entity
  - doc
  - decompose
  - handoff
  - finish

# Final system state after the run completes.
final_state:
  feature_status: done
  documents:
    - path: work/spec/example.md
      status: approved
  tasks:
    total: 3
    done: 3

# Whether all success_criteria from the scenario were met.
success: true

# List of success_criteria that were NOT met (empty list when success is true).
failures: []

# Optional free-text observations about the run.
notes: >
  Agent required one clarifying prompt to advance past the specifying gate.
  All tasks were generated correctly from bold-identifier AC lines.
```

### Field Descriptions

| Field            | Type     | Required | Description                                               |
|------------------|----------|----------|-----------------------------------------------------------|
| `scenario_id`    | string   | yes      | The `id` of the scenario that was run                     |
| `system_version` | string   | yes      | Git SHA or tag of the Kanbanzai version used              |
| `run_date`       | string   | yes      | ISO 8601 date (YYYY-MM-DD)                                |
| `tools_called`   | sequence | yes      | Ordered list of MCP tool names called during the run      |
| `final_state`    | mapping  | yes      | Feature status and document/task states at end of run     |
| `success`        | boolean  | yes      | `true` if all success_criteria were met, otherwise `false`|
| `failures`       | sequence | yes      | Success criteria that were not met (empty list if passing)|
| `notes`          | string   | no       | Free-text observations (optional)                         |

---

## How to Conduct a Manual Evaluation Run

### Prerequisites

1. A working Kanbanzai MCP server connected to an AI agent (e.g., Claude).
2. A clean repository with the pre-V3.0 Kanbanzai binary.
3. Note the current git SHA: `git rev-parse --short HEAD`

### Step 1 — Select a Scenario

Open `eval-NNN.yaml` for the scenario you want to run. Read the `description`,
`starting_state`, `expected_pattern`, and `success_criteria` in full before
starting.

### Step 2 — Set Up the Starting State

Using the MCP tools or direct YAML writes, create the system state described in
`starting_state`:

- Create a feature in the specified `feature_status`.
- Register any pre-existing documents listed in `starting_state.documents`.
  If `status: approved`, approve them too.
- Create any pre-existing tasks listed in `starting_state.tasks` and advance
  them to the specified status.

### Step 3 — Run the Agent

Prompt the AI agent with a task that triggers the scenario. For example:

> "Please implement feature FEAT-XXX following the Kanbanzai workflow."

Observe the agent's tool calls. Do not intervene unless the scenario description
explicitly requires a human interaction point.

### Step 4 — Record Results

After the agent completes (or stalls), create a measurement record:

1. Copy the measurement record schema above as a starting template.
2. Fill in `scenario_id`, `system_version` (git SHA), `run_date`.
3. List every MCP tool called, in order, in `tools_called`.
4. Record the final feature status and document/task states in `final_state`.
5. Evaluate each entry in `success_criteria` against the observed final state.
   Set `success: true` if all pass; add failing criteria to `failures`.
6. Add any notable observations to `notes`.
7. Save the file as `baseline-eval-NNN-YYYYMMDD.yaml` in this directory.

### Step 5 — Commit the Record

```
git add work/eval/baseline-eval-NNN-YYYYMMDD.yaml
git commit -m "eval: capture baseline measurement for eval-NNN"
```

---

## V3.0 Comparison Methodology

When V3.0 evaluation runs are conducted:

1. Run each scenario from the full scenario set against the V3.0 system.
2. Save results as `v3-eval-NNN-YYYYMMDD.yaml` (same schema as baseline records).
3. For each scenario, compare:
   - `success` (pass/fail change)
   - `tools_called` count (efficiency: did V3.0 use fewer calls?)
   - `failures` list (which specific criteria improved or regressed?)
   - `notes` (qualitative observations)
4. A scenario is considered **improved** if `success` changed `false → true`
   or if `tools_called` count decreased with no new failures.
5. A scenario is considered **regressed** if `success` changed `true → false`
   or if new entries appear in `failures`.

The baseline and V3.0 result files share the same schema, so comparison can be
done by a script or by manual inspection side-by-side.

---

## Category Coverage Requirements

The scenario set must cover all six categories with the following minimum counts:

| Category                    | Minimum | File Range     |
|-----------------------------|---------|----------------|
| `happy-path`                | 3       | eval-001–003   |
| `gate-failure-and-recovery` | 3       | eval-004–007   |
| `review-rework-loop`        | 2       | eval-008–010   |
| `multi-feature-plan`        | 2       | eval-011–013   |
| `edge-case`                 | 3       | eval-014–017   |
| `tool-selection`            | 2       | eval-018–020   |