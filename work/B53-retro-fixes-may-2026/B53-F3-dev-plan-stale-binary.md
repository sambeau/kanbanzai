# Dev-Plan: Stale Binary Detection Discoverability

**Feature:** FEAT-01KR003J3A34Z
**Tier:** retro_fix
**Spec:** FEAT-01KR003J3A34Z/spec-b53-f3-spec-stale-binary

## Overview

Two-file change: add server_info check to pre-task checklist and verification guidance.

## Task Breakdown

### T1: Update getting-started pre-task checklist

- **File:** `.agents/skills/kanbanzai-getting-started/SKILL.md`
- **Description:** Add server_info check to pre-task checklist: when MCP server may be stale, verify binary freshness.
- **Deliverable:** Updated skill file.

### T2: Update kanbanzai-agents verification guidance

- **File:** `.agents/skills/kanbanzai-agents/SKILL.md`
- **Description:** Add server_info guidance to verification section: when test results are unexpected, check binary freshness before debugging code.
- **Deliverable:** Updated skill file.

## Dependency Graph

```
T1 (no dependencies)
T2 (no dependencies)
```

T1 and T2 touch different files — dispatch in parallel.

## Interface Contracts

No interfaces changed. Pure documentation.

## Traceability Matrix

| Task | FR-001 | FR-002 | AC-001 | AC-002 |
|------|--------|--------|--------|--------|
| T1   | x      |        | x      |        |
| T2   |        | x      |        | x      |
