| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-07                     |
| Status | Draft                          |
| Author | architect                       |
| Feature | FEAT-01KR1GJGJ9BZS |

# Dev-Plan: Retrospective Pipeline Tightening

## Overview

This dev-plan decomposes the P57 specification into implementation tasks. The work is a single integrated feature because the components are tightly coupled: the `create_fix` action depends on the auto-generated document templates, which depend on the document type fix. Tasks are ordered to respect these dependencies.

## Task Breakdown

### Task 1: Document Type Registration Fix

**Summary:** Change `RetroService.Report()` to register retro reports as `type: "retro"` instead of `type: "report"`.

**Files:**
- `internal/service/retro_synthesis.go` — change `Type: "report"` to `Type: "retro"` in `Report()`
- `internal/service/retro_synthesis_test.go` — verify doc type in report registration

**Verification:** `go test ./internal/service/...` passes; `doc(action: "get")` on a retro report shows `type: "retro"`.

**Depends on:** (none — foundational, do first)

### Task 2: Auto-Generated Document Templates

**Summary:** Create template functions that generate design and specification documents from `RetroTheme` data. These produce complete markdown strings following the design document template (Component 1a) and spec template (Component 2).

**Files:**
- `internal/service/retro_templates.go` — `RenderRetroDesign(theme RetroTheme, featureID string) string` and `RenderRetroSpec(theme RetroTheme, featureID string) string`
- `internal/service/retro_templates_test.go` — verify output contains all required sections with non-empty content

**Verification:** Templates produce valid markdown with all required section headings; content is derived from theme fields.

**Depends on:** (none — can parallel with Task 1)

### Task 3: create_fix Action on retro Tool

**Summary:** Add the `create_fix` action to the `retro` MCP tool. Implements dual-mode (human-gated and auto), theme selection, parent plan auto-creation, idempotency, and the full auto-mode pipeline (design → spec → dev-plan → decompose).

**Files:**
- `internal/mcp/retro_tool.go` — add `create_fix` action handler, input parsing, mode dispatch
- `internal/service/retro_service.go` — add `CreateFix()` method with synthesis, entity creation, document generation, and lifecycle advancement
- `internal/mcp/retro_tool_test.go` — tests for both modes, idempotency, parent plan auto-creation, and error cases

**Verification:** AC-001 through AC-006, AC-014 through AC-016, AC-018.

**Depends on:** Task 1 (doc type), Task 2 (templates)

### Task 4: Retro-Adapted Definition of Done

**Summary:** Add the `dod_variant: retro-fix` support to the verifier's close-out checklist. Item 10 checks signal resolution instead of knowledge curation. The verify-closeout skill must accept a `variant` parameter.

**Files:**
- `.kbz/skills/verify-closeout/SKILL.md` — add retro-fix variant section with item 10 replacement
- `internal/service/verifier.go` or relevant verifier dispatch code — support `dod_variant` parameter
- Test: verifier dispatch with `dod_variant: retro-fix` runs the adapted checklist

**Verification:** AC-010.

**Depends on:** (none — independent of the retro tool changes)

### Task 5: Stage Binding Profile

**Summary:** Add the `retro-fixing` profile to `stage-bindings.yaml` documenting both `human-gated` and `auto` mode gate configurations.

**Files:**
- `.kbz/stage-bindings.yaml` — add `retro-fixing` profile section

**Verification:** AC-011; profile is parseable and documents correct gate values for both modes.

**Depends on:** (none — documentation-only)

### Task 6: implement-retro-fix Skill

**Summary:** Create `.kbz/skills/implement-retro-fix/SKILL.md` documenting the retro fix workflow stages, vocabulary, anti-patterns, and procedures.

**Files:**
- `.kbz/skills/implement-retro-fix/SKILL.md` — new skill file

**Verification:** AC-013; file exists with required sections (vocabulary, anti-patterns, procedure, stage guidance).

**Depends on:** (none — documentation-only, can parallel with Tasks 4 and 5)

### Task 7: Integration and End-to-End Verification

**Summary:** Wire everything together, run full end-to-end tests, and verify all 18 acceptance criteria pass. Fix any integration issues.

**Files:**
- `internal/mcp/retro_tool_test.go` — add end-to-end integration tests
- `internal/service/retro_service_test.go` — add full pipeline tests

**Verification:** All 18 acceptance criteria pass; `go test ./...` passes; `go vet ./...` clean.

**Depends on:** Tasks 1–6 (all prior tasks)

## Dependency Graph

```
Task 1 (doc type fix) ─────┐
                            ├──→ Task 3 (create_fix) ──→ Task 7 (integration)
Task 2 (templates) ────────┘
Task 4 (retro DoD) ────────────────────────────────────→ Task 7 (integration)
Task 5 (stage binding) ─────────────────────────────────→ Task 7 (integration)
Task 6 (skill) ─────────────────────────────────────────→ Task 7 (integration)
```

Tasks 1 and 2 can run in parallel. Tasks 4, 5, and 6 are independent and can all run in parallel with each other and with Tasks 1+2. Task 3 depends on Tasks 1+2. Task 7 depends on everything.

## Interface Contracts

- **Task 1 → Task 3:** `RetroService.Report()` returns documents with `Type: "retro"`. Task 3 must use this type when querying or registering retro documents.
- **Task 2 → Task 3:** `RenderRetroDesign(theme, featureID) string` and `RenderRetroSpec(theme, featureID) string` are called by `CreateFix()` in auto mode. Templates must not panic on empty suggestion or observation fields.
- **Task 3 → Task 4:** Features created by `create_fix` carry `tier: retro_fix`. The verifier's `dod_variant: retro-fix` must be triggered by this tier.
- **Task 3 → Task 5:** The `retro-fixing` profile documents the gate configuration that `create_fix` enforces at runtime.

## Traceability Matrix

| Requirement | Task(s) |
|---|---|
| REQ-001 (create_fix action) | Task 3 |
| REQ-002 (mode parameter) | Task 3 |
| REQ-003 (theme_index in human-gated) | Task 3 |
| REQ-004 (theme_count + severity_threshold) | Task 3 |
| REQ-005 (parent_plan auto-creation) | Task 3 |
| REQ-006 (idempotency) | Task 3 |
| REQ-007 (scope parameter) | Task 3 |
| REQ-008 (auto-generated design) | Task 2, Task 3 |
| REQ-009 (auto-generated spec) | Task 2, Task 3 |
| REQ-010 (spec content from theme) | Task 2 |
| REQ-011 (auto dev-plan + decompose) | Task 3 |
| REQ-012 (retro-adapted DoD) | Task 4 |
| REQ-013 (stage binding profile) | Task 5 |
| REQ-014 (doc type fix) | Task 1 |
| REQ-015 (implement-retro-fix skill) | Task 6 |
| REQ-016 (batch summary response) | Task 3 |
| REQ-017 (tags with signal IDs) | Task 3 |
| REQ-018 (auto lifecycle advancement) | Task 3 |
| REQ-NF-001 (30 tool call budget) | Task 7 |
| REQ-NF-002 (valid design doc) | Task 2, Task 7 |
| REQ-NF-003 (valid spec doc) | Task 2, Task 7 |
| REQ-NF-004 (tier config unchanged) | Task 7 |
| REQ-NF-005 (no mutation of pre-existing) | Task 3, Task 7 |
