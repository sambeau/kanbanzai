# Dev-Plan: Orchestrator Role Hardening

## Overview

Implement Components 1–3 and Decisions 1–3 from the P55 design. Two files modified: `orchestrator.yaml` (add anti-pattern, remove tools) and `orchestrate-development/SKILL.md` (add anti-pattern, add hard constraint).

## Task Breakdown

### T1: Add anti-pattern and remove tools from orchestrator role
- Edit `.kbz/roles/orchestrator.yaml`
- Add "Pre-delegation Code Investigation" anti-pattern in alphabetical order
- Remove `grep` and `search_graph` from tools list
- Verify `read_file` remains
- Acceptance: AC-001 through AC-006, AC-009, AC-012

### T2: Add anti-pattern and hard constraint to orchestrate-development skill
- Edit `.kbz/skills/orchestrate-development/SKILL.md`
- Add "Pre-delegation Code Investigation" after "Manual Prompt Composition"
- Add "Constraint ℋ — No Code Investigation" to Phase 1
- Acceptance: AC-003, AC-007, AC-008, AC-010, AC-011

## Dependency Graph

T1 and T2 are independent (edit different files). Parallel dispatch.

## Traceability Matrix

| Task | Spec Requirements |
|------|-------------------|
| T1 | REQ-001, REQ-002, REQ-004, REQ-005, REQ-006, REQ-007, REQ-010 |
| T2 | REQ-003, REQ-008, REQ-009, REQ-011 |
