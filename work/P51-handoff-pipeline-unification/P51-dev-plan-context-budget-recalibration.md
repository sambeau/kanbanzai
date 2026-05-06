# Dev Plan: Context Budget Recalibration

**Feature:** FEAT-01KQYZZFGBGQK
**Specification:** FEAT-01KQYZZFGBGQK/spec-p51-spec-context-budget-recalibration (approved)
**Date:** 2026-05-06

## Overview

Updates two stale budget constants (`DefaultContextWindowTokens` → 1M, `assemblyDefaultBudget` → 64KB), makes context window configurable via `.kbz/local.yaml`, and adds topic-level detail to trimmed metadata. Three tasks: constants + config, topic metadata, and tests.

## Scope

Implements FR-001 through FR-005 and FR-NF-001 through FR-NF-003 from the spec. Three tasks covering constant updates, config loading, topic metadata, and comprehensive tests.

## Task Breakdown

### Task 1: Update budget constants and add config override (TASK-01KQZ2YJVEDYW)
- Update `DefaultContextWindowTokens` to 1,000,000
- Raise `assemblyDefaultBudget` to 65,536
- Add `context_window_tokens` config loading from `.kbz/local.yaml`
- Validate config: reject values below 100,000
- Expose active window size on Pipeline struct
- ACs: AC-001, AC-002, AC-003, AC-004, AC-005, AC-006, AC-009

### Task 2: Add topic field to trimmed knowledge metadata (TASK-01KQZ2YJVEPBS)
- Add `topic` field to trimmed entry metadata in pipeline output
- Truncate long topics with `…` (U+2026) to stay within budget
- ACs: AC-007, AC-008

### Task 3: Tests for context budget recalibration (TASK-01KQZ2YJVEGCS)
- Unit tests for all 9 acceptance criteria
- Integration test for trimmed metadata with topic field
- Dependencies: Task 1, Task 2
- ACs: AC-001 through AC-009

## Dependency Graph

```
T1 (constants + config)
  ├── T2 (topic metadata)
  └── T3 (tests) ── depends on T1, T2
```

T1 and T2 can run in parallel. T3 requires both.

## Risk Assessment

- **Risk:** Config loading changes break existing `.kbz/local.yaml` parsing → **Low**. The `context_window_tokens` key is new and optional; absence defaults to existing behavior.
- **Risk:** Budget constant changes affect response content → **Low**. Raising the budget ceiling adds content that was previously trimmed, not modifying existing content.

## Verification Approach

| AC | Method | Task |
|----|--------|------|
| AC-001 | Unit test: default window = 1M | T1, T3 |
| AC-002 | Unit test: warn/refuse thresholds | T1, T3 |
| AC-003 | Unit test: config override 500K | T1, T3 |
| AC-004 | Unit test: no config → default | T1, T3 |
| AC-005 | Unit test: reject 50K | T1, T3 |
| AC-006 | Unit test: byte_budget = 65536 | T1, T3 |
| AC-007 | Integration test: topic in trimmed | T2, T3 |
| AC-008 | Unit test: specific topic value | T2, T3 |
| AC-009 | Unit test: validation error | T1, T3 |

## Interface Contracts

- **Pipeline struct** — `WindowTokens()` method exposes active window size (from config or default). Config loading adds `ContextWindowTokens` field to `LocalConfig`.
- **Trimmed metadata** — Each entry in `trimmed` array gains a `topic` field (string, truncated with `…` when needed).
- **Config validation** — `context_window_tokens` < 100,000 returns error at load time.

## Traceability Matrix

| FR | Task |
|----|------|
| FR-001 | T1 |
| FR-002 | T1 |
| FR-003 | T1 |
| FR-004 | T2 |
| FR-005 | T1 |
| FR-NF-001 | T1 |
| FR-NF-002 | T1, T3 |
| FR-NF-003 | T2 |
