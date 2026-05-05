# Dev-Plan: Decompose Proposal Quality

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-05                     |
| Status | approved |
| Author | architect                      |

## Overview

This dev-plan implements the decompose proposal quality spec:
`work/P50-retro-may-2026/P50-spec-decompose-proposal-quality.md`
(DOC-`FEAT-01KQTNYN00M4P/spec-p50-spec-decompose-proposal-quality`).

Three changes to `decompose propose`: refuse-to-propose when no ACs found, paired
implementation+test task output (default on, configurable), and dependency graph fix
(dependencies between complete tasks only).

## Task Breakdown

### T1: Implement refuse-to-propose mode
- **Deliverable:** Decompose handler updated to check whether the spec has parseable acceptance criteria before generating a proposal. Returns a clear diagnostic error when none found. Uses the P24 AC format parser as the single source of truth.
- **Depends on:** nothing
- **Effort:** 1 (conditional check + error message)
- **Parallelisable:** yes

### T2: Implement paired implementation + test task output
- **Deliverable:** Decompose proposal prompt assembly updated to produce two tasks per AC (impl + test). Test task depends_on points to impl task. Paired mode is default-on with an opt-out flag in the decompose input struct.
- **Depends on:** nothing
- **Effort:** 2 (prompt assembly change + test task generation)
- **Parallelisable:** yes

### T3: Implement testing-concern AC detection
- **Deliverable:** Logic to detect when an AC is purely a testing concern (e.g. "verify X appears") and produce a single test task instead of an impl+test pair
- **Depends on:** T2
- **Effort:** 1 (pattern matching on AC text)
- **Parallelisable:** no

### T4: Fix dependency graph to use complete tasks only
- **Deliverable:** Dependency graph generation updated to only express dependencies between complete task nodes. No partial-completion dependencies.
- **Depends on:** T2 (needs the task structure from paired output)
- **Effort:** 1 (graph generation logic)
- **Parallelisable:** no

### T5: Add decompose proposal quality tests
- **Deliverable:** Unit tests for: refuse-to-propose (AC-free spec → error), paired output (3-AC spec → 6 tasks), depends_on correctness, paired=false flag, dependency graph completeness, test-only AC detection
- **Depends on:** T1, T2, T3, T4
- **Effort:** 2 (test suite)
- **Parallelisable:** no

## Dependency Graph

```
T1 ──────────────────────┐
                          ├── T5
T2 ──┬── T3 ──┬── T4 ────┘
     │        │
     └────────┘
```

T1 and T2 are independent and can run in parallel. T3 depends on T2. T4 depends on T2. T5 gates on all.

## Interface Contracts

- **decompose propose** input struct gains optional `paired_test_tasks` bool field (default true)
- No change to `decompose review` or `decompose apply`
- AC format parser (P24) is the single source of truth for "has acceptance criteria"
- Existing decompose callers using default settings still receive valid proposals

## Traceability Matrix

| Requirement | Task(s) |
|-------------|---------|
| REQ-001 (refuse-to-propose) | T1 |
| REQ-002 (paired tasks) | T2 |
| REQ-003 (depends_on) | T2 |
| REQ-004 (configurable flag) | T2 |
| REQ-005 (dependency graph) | T4 |
| REQ-006 (test-only AC detection) | T3 |
| REQ-NF-001 (no latency impact) | T5 |
| REQ-NF-002 (backward compat) | T5 |
