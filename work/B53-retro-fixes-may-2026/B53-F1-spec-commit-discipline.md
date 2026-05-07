# Specification: Commit Discipline for Management Actions

**Feature:** FEAT-01KR003J39A59
**Tier:** retro_fix
**Design:** FEAT-01KR003J39A59/design-b53-f1-design-commit-discipline

## Overview

Add an explicit commit step to the kanbanzai-agents skill covering non-task management workflows.

## Scope

- `.agents/skills/kanbanzai-agents/SKILL.md` only
- Documentation-only change

## Functional Requirements

- **FR-001:** The skill must include a commit step that applies when no task has been claimed (management-only sessions)
- **FR-002:** The step must cover writes to `.kbz/state/` and `.kbz/index/`
- **FR-003:** The step must not duplicate or conflict with the existing task-claimed commit checklist

## Acceptance Criteria

- [ ] AC-001: `.agents/skills/kanbanzai-agents/SKILL.md` contains a commit instruction visible without a task claim
- [ ] AC-002: The instruction specifically mentions `.kbz/state/` and `.kbz/index/` writes
- [ ] AC-003: The existing task-claimed commit checklist remains intact and unmodified
