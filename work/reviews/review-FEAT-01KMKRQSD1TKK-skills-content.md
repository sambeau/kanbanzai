# Review Report: FEAT-01KMKRQSD1TKK (skills-content)

| Field            | Value                                                    |
|------------------|----------------------------------------------------------|
| Feature          | FEAT-01KMKRQSD1TKK (skills-content)                     |
| Plan             | P3-kanbanzai-1.0                                         |
| Review Date      | 2026-03-28T01:41:39Z                                     |
| Reviewer Profile | Feature Implementation Review Profile                    |
| Review Units     | 2 (Procedural Skills, Opinionated Skills)                |
| Aggregate Verdict| **changes_required**                                     |

---

## Summary

The Skills Content feature authored six kanbanzai skill files for installation by `kanbanzai init`. All six skill files exist, are well-structured, and faithfully implement the design document's content outlines. However, two systemic issues produce blocking findings across the feature:

1. **Fabricated lifecycle states** in the workflow skill's `references/lifecycle.md` — the Feature and Bug lifecycle diagrams contain invented state names that do not match the codebase, which would cause every transition attempt to fail.
2. **Pervasive use of 1.0 MCP tool names** — all six skills (except `kanbanzai-planning`, which has no tool references) reference tools removed in the Kanbanzai 2.0 redesign. An agent following these instructions would call nonexistent tools.

Additionally, this feature has **no specification document** — only a design document. Specification conformance could only be assessed against design intent, not formal acceptance criteria. Both review units flagged this as a concern per the Missing Spec edge case.

---

## Per-Dimension Verdicts

| Dimension                  | Unit 1 (Procedural) | Unit 2 (Opinionated) | Aggregate |
|----------------------------|----------------------|----------------------|-----------|
| Specification Conformance  | concern              | concern              | **concern** |
| Implementation Quality     | fail                 | pass_with_notes      | **fail**  |
| Test Adequacy              | not_applicable       | not_applicable       | **not_applicable** |
| Documentation Currency     | fail                 | concern              | **fail**  |
| Workflow Integrity         | pass                 | pass                 | **pass**  |

---

## Blocking Findings

### B1. Feature lifecycle states are fabricated

- **Dimension:** Implementation Quality
- **Severity:** Critical
- **Location:** `.agents/skills/kanbanzai-workflow/references/lifecycle.md`, Feature section
- **Description:** The Feature lifecycle lists states `proposed → designing → spec-ready → dev-ready → active → done`. The actual states in `internal/validate/lifecycle.go` are `proposed → designing → specifying → dev-planning → developing → reviewing → done` (plus terminals `superseded`, `cancelled`). The states `spec-ready`, `dev-ready`, and `active` do not exist. An agent following this reference will attempt invalid transitions on every feature entity.
- **Requirement violated:** Design §3.2 — entity lifecycle reference must be accurate.

### B2. Bug lifecycle states are fabricated

- **Dimension:** Implementation Quality
- **Severity:** Critical
- **Location:** `.agents/skills/kanbanzai-workflow/references/lifecycle.md`, Bug section
- **Description:** The Bug lifecycle lists `reported → triaged → active → done`. The actual lifecycle is `reported → triaged → reproduced → planned → in-progress → needs-review → verified → closed` (plus `cannot-reproduce` from `triaged`). The states `active` and `done` do not exist for bugs. This is a completely wrong lifecycle.
- **Requirement violated:** Design §3.2 — entity lifecycle reference must be accurate.

### B3. Task lifecycle is incomplete

- **Dimension:** Implementation Quality
- **Severity:** Medium
- **Location:** `.agents/skills/kanbanzai-workflow/references/lifecycle.md`, Task section
- **Description:** The Task lifecycle is missing the `blocked` state (`active → blocked → active`), the `needs-rework` state, and the `duplicate` terminal state. The core flow (`queued → ready → active → done`) is correct, but incomplete.
- **Requirement violated:** Design §3.2 — entity lifecycle reference must be accurate.

### B4. getting-started skill uses removed 1.0 tool names

- **Dimension:** Documentation Currency
- **Severity:** High
- **Location:** `.agents/skills/kanbanzai-getting-started/SKILL.md`, lines 43–52
- **Description:** Three stale tool references: `work_queue` (→ `next`), `list_entities_filtered` (→ `entity` action: `list` or `status`), `context_assemble` (→ `next` with task ID or `handoff`). These tools were removed in the 2.0 redesign.
- **Requirement violated:** Kanbanzai 2.0 completion — all 1.0 tools removed.

### B5. workflow skill uses removed 1.0 tool names

- **Dimension:** Documentation Currency
- **Severity:** High
- **Location:** `.agents/skills/kanbanzai-workflow/SKILL.md`, Gotchas section
- **Description:** Two stale tool references: `update_status` (→ `entity` action: `transition`), `doc_record_approve` (→ `doc` action: `approve`).
- **Requirement violated:** Kanbanzai 2.0 completion — all 1.0 tools removed.

### B6. documents skill uses removed 1.0 tool names throughout

- **Dimension:** Documentation Currency
- **Severity:** High
- **Location:** `.agents/skills/kanbanzai-documents/SKILL.md`, multiple sections
- **Description:** Six distinct stale tool references pervade the entire skill: `doc_record_submit` (×3), `batch_import_documents` (×3), `doc_record_get` (×1), `doc_record_approve` (×1), `doc_record_refresh` (×2), `doc_record_supersede` (×1). All should use the consolidated `doc` tool with appropriate action parameters.
- **Requirement violated:** Kanbanzai 2.0 completion — all 1.0 tools removed.

### B7. agents skill uses removed 1.0 tool names throughout

- **Dimension:** Documentation Currency
- **Severity:** High
- **Location:** `.agents/skills/kanbanzai-agents/SKILL.md`, multiple sections
- **Description:** Six distinct stale tool references: `context_assemble` (→ `next`/`handoff`), `context_report` (→ removed, no 2.0 equivalent), `work_queue` (→ `next`), `dispatch_task` (→ `next` with task ID), `complete_task` (→ `finish`), `knowledge_contribute` (→ `knowledge` action: `contribute`).
- **Requirement violated:** Kanbanzai 2.0 completion — all 1.0 tools removed.

### B8. documents skill references nonexistent `doc_record_refresh` tool

- **Dimension:** Implementation Quality + Documentation Currency
- **Severity:** Medium
- **Location:** `.agents/skills/kanbanzai-documents/SKILL.md`, Drift and Refresh section
- **Description:** The skill instructs agents to call `doc_record_refresh` to update a document's content hash. This tool does not exist in the 2.0 `doc` tool surface (actions: register, approve, get, content, list, gaps, validate, supersede, import). The drift/refresh workflow needs to be rethought against the actual tool surface — possibly `doc` action: `validate` for hash checking, or re-`register` for hash updates.
- **Requirement violated:** Design §3.3 — tool references must be functional.

---

## Non-Blocking Notes

### N1. Agents skill has 4 gotchas; design specifies 3

- **Dimension:** Specification Conformance (against design)
- **Description:** Design §3.4 says "three specific failure modes." The skill has four bullets — the fourth is a cross-reference rather than a failure mode. Minor deviation; the fourth bullet is useful.

### N2. Agents skill includes Retrospective Observations section not in design

- **Dimension:** Specification Conformance (against design)
- **Description:** The agents skill includes a substantial "Retrospective Observations" section (~40 lines) documenting the `finish` tool's `retrospective` parameter. Not in design §3.4 content outline. A positive addition, but adds significant length. Consider progressive disclosure via a reference file.

### N3. `context_report` has no 2.0 equivalent

- **Dimension:** Implementation Quality
- **Description:** The agents skill's Context Assembly section instructs agents to call `context_report` to record knowledge entry usage. This tool was removed in 2.0 with no replacement. The knowledge feedback loop should be replaced with `knowledge` tool's `confirm`/`flag` actions, or the instruction should be removed.

### N4. getting-started line count is at the limit

- **Dimension:** Specification Conformance (against design)
- **Description:** The file is 67 lines, within the design constraint of "under 70 lines." If 2.0 tool name updates add lines, the constraint may be exceeded. Worth monitoring during remediation.

### N5. Plan lifecycle in reference file is correct

- **Dimension:** Implementation Quality
- **Description:** The Plan lifecycle (`proposed → designing → active → done` with `superseded`/`cancelled` terminals) matches the codebase. No change needed.

### N6. Design document itself used 1.0 tool names (root cause)

- **Dimension:** Specification Conformance (against design)
- **Description:** The design document §3.1–§3.4 references 1.0 tool names because it was written before 2.0 was complete. The implementation faithfully followed the design's tool names rather than updating them. This is the root cause of findings B4–B8. Remediation should update the skills to use 2.0 tool names.

### N7. Design skill references 3 stale tool names

- **Dimension:** Documentation Currency
- **Location:** `.agents/skills/kanbanzai-design/SKILL.md`
- **Description:** Three occurrences: `doc_record_approve` (→ `doc` action: `approve`), `doc_record_submit` (→ `doc` action: `register`), `doc_record_refresh` (→ no direct equivalent). Same root cause as B4–B8.

### N8. Missing specification document

- **Dimension:** Specification Conformance
- **Description:** This feature has no specification document — only a design. Both review units flagged this as Edge Case 1 (Missing Spec). Specification conformance was assessed against design intent only. This is a concern but not blocking since the design document provides sufficient detail to evaluate the deliverables.

### N9. Planning skill correctly avoids tool references

- **Dimension:** Implementation Quality
- **Description:** The planning skill avoids tool name references entirely, which is appropriate since planning conversations are primarily discursive. This makes it resilient to future tool surface changes.

### N10. Design quality principles are consistent across all locations

- **Dimension:** Implementation Quality
- **Description:** The six quality principles are named identically across design document, skill body, and reference file. The progressive disclosure model works well.

---

## Reviewer Unit Breakdown

### Unit 1: Procedural Skills

| Sub-Agent Scope | Files |
|-----------------|-------|
| Procedural skills | `.agents/skills/kanbanzai-getting-started/SKILL.md` |
|                   | `.agents/skills/kanbanzai-workflow/SKILL.md` |
|                   | `.agents/skills/kanbanzai-workflow/references/lifecycle.md` |
|                   | `.agents/skills/kanbanzai-documents/SKILL.md` |
|                   | `.agents/skills/kanbanzai-agents/SKILL.md` |

Spec sections checked: Design §3 (Procedural Skills) — §3.1 through §3.4

### Unit 2: Opinionated Skills

| Sub-Agent Scope | Files |
|-----------------|-------|
| Opinionated skills | `.agents/skills/kanbanzai-planning/SKILL.md` |
|                    | `.agents/skills/kanbanzai-design/SKILL.md` |
|                    | `.agents/skills/kanbanzai-design/references/design-quality.md` |

Spec sections checked: Design §4 (Opinionated Skills) — §4.1 through §4.2

---

## Recommended Remediation

The blocking findings cluster into two logical remediation groups:

1. **Fix lifecycle reference** (B1, B2, B3): Rewrite `.agents/skills/kanbanzai-workflow/references/lifecycle.md` with correct state names from `internal/validate/lifecycle.go`. Include all states, transitions, and terminal states for all four entity types.

2. **Update all tool references to 2.0** (B4, B5, B6, B7, B8): Systematically replace 1.0 tool names with 2.0 equivalents across all six skill files. Remove references to `context_report` and `doc_record_refresh` (no 2.0 equivalents). Address the design skill's stale references (N7) in the same pass.