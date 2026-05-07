# Design: Stale Binary Detection Discoverability

**Feature:** FEAT-01KR003J3A34Z — Stale binary detection discoverability
**Tier:** retro_fix
**Retro Signal:** KE-01KMS0EE97M2P

## Overview

The `server_info` MCP tool exists and reports build timestamp, git SHA, and binary path, but agents don't know to use it proactively when verification results are unexpected. This caused significant wasted time debugging "failing" acceptance criteria that were actually passing on a stale binary.

## Design

### Change

Add a `server_info` check to the pre-task checklist and verification guidance in workflow skills.

### Locations

1. `.agents/skills/kanbanzai-getting-started/SKILL.md` — pre-task checklist
2. `.agents/skills/kanbanzai-agents/SKILL.md` — verification section

### What to add

- Pre-task checklist: "If the MCP server may be stale, run `server_info` to verify the running binary matches the latest build"
- Verification guidance: "When test results are unexpected (passes that should fail, failures that should pass), run `server_info` before debugging code"

## Goals and Non-Goals

### Goals

- Implement the change described in the Design section
- Keep scope minimal — no unrelated refactoring

### Non-Goals

- Does not modify the server_info tool itself
- Does not add automated binary freshness checks

## Alternatives Considered

### Do nothing

Leave the friction in place. Rejected: the retro signals show this wastes agent cycles and causes state drift.

### Automated enforcement

Add tooling to automatically enforce the desired behavior. Rejected: over-engineered for the current scale; documentation is sufficient.

## Dependencies

None. Pure documentation change.
