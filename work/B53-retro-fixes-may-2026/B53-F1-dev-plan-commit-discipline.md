# Dev-Plan: Commit Discipline for Management Actions

**Feature:** FEAT-01KR003J39A59
**Tier:** retro_fix
**Spec:** FEAT-01KR003J39A59/spec-b53-f1-spec-commit-discipline

## Overview

Single-task change: add commit discipline guidance for non-task workflows to `.agents/skills/kanbanzai-agents/SKILL.md`.

## Task Breakdown

### T1: Add management-action commit step to kanbanzai-agents skill

- **File:** `.agents/skills/kanbanzai-agents/SKILL.md`
- **Description:** Add a subsection or note to the commit discipline section covering non-task workflows. Any tool call writing to `.kbz/state/` or `.kbz/index/` should be followed by a commit before moving to a different concern.
- **Deliverable:** Updated skill file with the new commit step.

## Dependency Graph

```
T1 (no dependencies)
```

## Interface Contracts

No interfaces changed. Pure documentation.

## Traceability Matrix

| Task | FR-001 | FR-002 | FR-003 | AC-001 | AC-002 | AC-003 |
|------|--------|--------|--------|--------|--------|--------|
| T1   | x      | x      | x      | x      | x      | x      |
