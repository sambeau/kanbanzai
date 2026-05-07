# Design: Commit Discipline for Management Actions

**Feature:** FEAT-01KR003J39A59 — Commit discipline for management actions
**Tier:** retro_fix
**Retro Signal:** KE-01KN5ZXK486TM

## Overview

Sessions that perform management work (doc registration, entity transitions, knowledge contributions) never see the commit checklist because it's only visible when a task is claimed via `next(id:...)`. State changes accumulate silently and are only caught at the start of the next session.

## Design

### Change

Add an explicit "commit workflow state changes" step to `.agents/skills/kanbanzai-agents/SKILL.md` that covers non-task workflows.

### Location

`.agents/skills/kanbanzai-agents/SKILL.md` — the section on commit discipline, currently scoped to task-claimed workflows only.

### What to add

A subsection or note that any tool call writing to `.kbz/state/` or `.kbz/index/` should be followed by a commit before moving to a different concern. This applies regardless of whether a task was claimed.

## Goals and Non-Goals

### Goals

- Implement the change described in the Design section
- Keep scope minimal — no unrelated refactoring

### Non-Goals

- Does not change the task-claimed commit checklist
- Does not add automated commit hooks

## Alternatives Considered

### Do nothing

Leave the friction in place. Rejected: the retro signals show this wastes agent cycles and causes state drift.

### Automated enforcement

Add tooling to automatically enforce the desired behavior. Rejected: over-engineered for the current scale; documentation is sufficient.

## Dependencies

None. Pure documentation change.
