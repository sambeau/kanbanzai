# Implementation Plan: Evaluation Baseline

| Field    | Value                                                              |
|----------|--------------------------------------------------------------------|
| Status   | Draft                                                              |
| Created  | 2026-04-01                                                         |
| Feature  | FEAT-01KN4ZPXXEG1F (evaluation-baseline)                          |
| Spec     | `work/spec/evaluation-baseline.md`                                 |
| Design   | `work/design/kanbanzai-2.5-infrastructure-hardening.md` §9        |

---

## 1. Implementation Approach

This feature is entirely documentation infrastructure — no MCP tool, service,
or storage changes are required. The deliverables are YAML scenario files, a
README, and baseline measurement records, all in `work/eval/`.

The work splits into three sequential tasks:

**Task 1** defines the schema and authoring conventions in `work/eval/README.md`.
It must complete first because Tasks 2 and 3 depend on the format it establishes.

**Task 2** authors all 15–20 scenario files covering the six required categories.
It depends on Task 1 (schema established) but has no external code dependencies.
The six category batches within Task 2 can be written in any order by a single
agent, or split across agents if desired.

**Task 3** captures baseline measurement records for at least 5 representative
scenarios against the current (pre-V3.0) system. It depends on Task 2 (scenarios
authored) and requires a live Kanbanzai MCP server session.

```
Task 1: README + Schema ──────────────────────────────────────────┐
                                                                   │
Task 2: Author Scenarios (all 15–20 files) ─────────────(after T1)┤
                                                                   │
Task 3: Capture Baseline Measurements ─────────────────(after T2) ┘
```

No interface contracts are required — this is documentation work with no
code interoperability concerns.

---

## 2. Task Breakdown

| # | Task                          | Output Artefacts                          | Spec Refs        |
|---|-------------------------------|-------------------------------------------|------------------|
| 1 | README and schema definition  | `work/eval/README.md`                     | REQ-10           |
| 2 | Author scenario files         | `work/eval/eval-001.yaml` … `eval-020.yaml` | REQ-01–REQ-09, REQ-15–REQ-17 |
| 3 | Capture baseline measurements | `work/eval/baseline-*.yaml` (≥5 records)  | REQ-11–REQ-14, AC-28, AC-29  |

---

## 3. Task Details

### Task 1: README and Schema Definition

**Objective:** Establish the authoritative schema for scenario files and
measurement records, and document run instructions, so that Task 2 can
author conforming files and Task 3 can capture conforming results.

**Spec refs:** REQ-10.

**Input context:**
- `work/spec/evaluation-baseline.md` §4 (scenario file schema) and §6
  (measurement record schema).
- `work/design/kanbanzai-2.5-infrastructure-hardening.md` §9.2 (example
  scenario YAML and category table).

**Output artefacts:**
- `work/eval/README.md` — must include:
  1. Purpose of `work/eval/` and its relationship to V3.0.
  2. Complete YAML schema for scenario files: all required fields with types,
     descriptions, and allowed values for `category`.
  3. Example scenario file (may be a trimmed excerpt from the design).
  4. Complete YAML schema for measurement records: all required fields.
  5. Step-by-step instructions for conducting a manual evaluation run:
     how to select a scenario, how to set up the starting state, how to
     run the agent, and how to record the result.
  6. Notes on V3.0 comparison methodology (V3.0 re-runs same scenarios and
     diffs results).

**Dependencies:** None.

**Acceptance check:** REQ-10 is satisfied when the README covers all six
points above and an agent unfamiliar with the design could author a conforming
scenario file using only the README.

---

### Task 2: Author Scenario Files

**Objective:** Write 15–20 YAML scenario files in `work/eval/`, named
`eval-001.yaml` through `eval-NNN.yaml`, covering all six required categories
with at least the minimum counts specified in the spec.

**Spec refs:** REQ-01 through REQ-09, REQ-15, REQ-16, REQ-17.

**Input context:**
- `work/eval/README.md` from Task 1 — schema is the authoritative reference.
- `work/spec/evaluation-baseline.md` §4.4 (category table and minimum counts).
- `work/design/kanbanzai-2.5-infrastructure-hardening.md` §9.2 (category
  examples and the example `eval-001` scenario).
- The current Kanbanzai feature lifecycle: proposed → designing → specifying
  → dev-planning → developing → reviewing → done. Scenarios must reference
  real lifecycle states.
- Current MCP tool names (as used in production): `entity`, `doc`, `doc_intel`,
  `knowledge`, `decompose`, `handoff`, `next`, `finish`, `checkpoint`.

**Output artefacts:**
- `work/eval/eval-001.yaml` through `work/eval/eval-NNN.yaml` (15–20 files).

**Category allocation (suggested, adjust to reach 15–20 total):**

| Category               | Min | Suggested | Example scenario names                                          |
|------------------------|-----|-----------|------------------------------------------------------------------|
| Happy path             | 3   | 3         | full-lifecycle, spec-only-feature, plan-level-feature           |
| Gate failure + recovery | 3  | 4         | missing-spec, missing-tasks, unapproved-design, wrong-doc-type  |
| Review-rework loop     | 2   | 3         | single-rework, rework-with-new-tasks, review-passes-first-try   |
| Multi-feature plan     | 2   | 3         | parallel-features, cross-feature-dep, plan-level-approval       |
| Edge cases             | 3   | 4         | no-design-stage, decompose-failure, doc-import, empty-plan      |
| Tool selection         | 2   | 3         | correct-tool-at-stage, avoids-wrong-stage, batch-vs-single      |

**Field requirements per file (from spec §4.2):**
- `id`: matches filename without extension (e.g., `eval-001`).
- `name`: concise human-readable label.
- `description`: prose description of what this scenario tests and why.
- `category`: exactly one of the six allowed values.
- `starting_state`: at minimum `feature_status`, `documents`, `tasks`.
- `expected_pattern`: ordered list of stages, each with `stage`, `tools`, `output`.
- `success_criteria`: list of testable assertion strings.

**Dependencies:** Task 1 (README must be written before authoring begins).

**Acceptance check:** REQ-08 (15–20 files), REQ-09 (all six categories met at
minimum counts), REQ-04 through REQ-07 (all required fields present and valid),
REQ-16 (unique IDs), REQ-17 (`category` values are exactly from the allowed set).

---

### Task 3: Capture Baseline Measurements

**Objective:** Run at least 5 scenarios from the authored set against the current
(pre-V3.0) Kanbanzai system and record the results as YAML measurement files in
`work/eval/`.

**Spec refs:** REQ-11 through REQ-14, AC-28, AC-29, AC-30.

**Input context:**
- `work/eval/README.md` — run instructions and measurement record schema.
- `work/eval/eval-NNN.yaml` — the scenario files authored in Task 2.
- `work/spec/evaluation-baseline.md` §6 (measurement record schema).

**Output artefacts:**
- ≥5 measurement records in `work/eval/`, named `baseline-eval-NNN-YYYYMMDD.yaml`
  (e.g., `baseline-eval-001-20260401.yaml`).

**Scenario selection for baseline:**
Prefer scenarios that exercise diverse tool paths. A reasonable baseline set:
1. One happy-path scenario (full lifecycle).
2. One gate failure + recovery scenario.
3. One review-rework loop scenario.
4. One edge-case scenario (e.g., decompose failure).
5. One tool selection scenario.

**Measurement record format (from spec REQ-12):**
```yaml
scenario_id: eval-001
system_version: <git SHA of HEAD at run time>
run_date: <ISO 8601 date>
tools_called:
  - entity
  - doc
  - decompose
  # … ordered list as called
final_state:
  feature_status: done
  documents:
    - path: work/spec/example.md
      status: approved
tasks:
  total: 3
  done: 3
success: true
failures: []
notes: >
  Optional free-text observations.
```

**Dependencies:** Task 2 (scenario files must exist before any can be run).

**Note:** This task requires human or agent execution of the scenario in a live
MCP session. It is not automated. The implementer should set up the starting
state as described in the scenario, run the agent, observe tool calls, and
record results faithfully.

**Acceptance check:** AC-28 (≥5 measurements captured), AC-29 (format is
machine-readable and V3.0-comparable), REQ-14 (measurements are against the
pre-V3.0 system — i.e., before any V3.0 gate, description, or assembly changes).

---

## 4. Scope Boundaries

Carried forward from `work/spec/evaluation-baseline.md` §3.2:

- No automated test tooling, scripts, or CI integration.
- No V3.0 post-implementation runs (belong to V3.0 Phase A).
- No changes to any MCP tool, service, or storage layer.
- No scenarios covering V3.0-only behaviour (mandatory gates, new skills, etc.).

---

## 5. Traceability

| Spec Requirement | Covered By   |
|------------------|--------------|
| REQ-01 (location) | Task 2       |
| REQ-02 (naming)   | Task 2       |
| REQ-03 (not in .kbz) | Task 2   |
| REQ-04–REQ-07 (required fields) | Task 2 |
| REQ-08 (15–20 files) | Task 2    |
| REQ-09 (category coverage) | Task 2 |
| REQ-10 (README)   | Task 1       |
| REQ-11–REQ-14 (baseline measurements) | Task 3 |
| REQ-15 (static, no MCP writes) | Task 2 |
| REQ-16 (unique IDs) | Task 2     |
| REQ-17 (category values) | Task 2 |
| AC-26             | Task 2       |
| AC-27             | Task 2       |
| AC-28             | Task 3       |
| AC-29             | Task 3       |
| AC-30             | Task 1       |