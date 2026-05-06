# Dev Plan: Documentation Fixes for Handoff Pipeline

**Feature:** FEAT-01KQYZZFGH6DK
**Specification:** FEAT-01KQYZZFGH6DK/spec-p51-spec-documentation-fixes (approved)
**Date:** 2026-05-06

## Scope

Implements FR-001 through FR-003 and FR-NF-001 through FR-NF-002. Two independent tasks: finish tool description update and orchestrate-development skill update (dual-write).

## Task Breakdown

### Task 1: Add 500-char limit to finish tool description (TASK-01KQZ30JT87ZA)
- Update `summary` parameter description in `finish_tool.go` to include "(max 500 characters)"
- Full description: "Brief description of what was accomplished (max 500 characters)"
- Verify existing finish tests still pass
- ACs: AC-001, AC-004

### Task 2: Update orchestrate-development skill with handoff role param (TASK-01KQZ30JT8X1A)
- Update `.kbz/skills/orchestrate-development/SKILL.md` to include `role: "implementer-go"` with handoff
- Apply dual-write to `internal/kbzinit/skills/orchestrate-development/SKILL.md`
- Verify both copies match
- ACs: AC-002, AC-003, AC-005

## Dependency Graph

```
T1 (finish tool description) ── independent
T2 (skill doc update) ── independent
```

Both tasks are entirely independent and can run in parallel. No code dependencies between them.

## Risk Assessment

- **Risk:** Documentation-only changes → **None**. Neither task modifies runtime behavior. All risks are cosmetic.

## Verification Approach

| AC | Method | Task |
|----|--------|------|
| AC-001 | Inspection: read finish_tool.go summary description | T1 |
| AC-002 | Inspection: grep skill file for handoff with role param | T2 |
| AC-003 | Inspection: compare .kbz/ and internal/kbzinit/ skill copies | T2 |
| AC-004 | Test: `go test ./internal/mcp/... -run TestFinish` passes | T1 |
| AC-005 | Test: pipeline test loads skill, no regression | T2 |
