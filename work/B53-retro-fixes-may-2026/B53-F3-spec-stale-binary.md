# Specification: Stale Binary Detection Discoverability

**Feature:** FEAT-01KR003J3A34Z
**Tier:** retro_fix
**Design:** FEAT-01KR003J3A34Z/design-b53-f3-design-stale-binary

## Overview

Make server_info tool usage proactive in workflow guides and pre-task checklists.

## Scope

- `.agents/skills/kanbanzai-getting-started/SKILL.md`
- `.agents/skills/kanbanzai-agents/SKILL.md`
- Documentation-only change

## Functional Requirements

- **FR-001:** Pre-task checklist must include instruction to verify MCP binary freshness via server_info when stale binary is suspected
- **FR-002:** Verification guidance must include server_info check when test results are unexpected

## Acceptance Criteria

- [ ] AC-001: `kanbanzai-getting-started/SKILL.md` pre-task checklist mentions server_info for binary freshness
- [ ] AC-002: `kanbanzai-agents/SKILL.md` verification section mentions server_info for unexpected test results
