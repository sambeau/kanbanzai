# P38-F7: Documentation and Skills Updates — Dev Plan

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-04-28T01:44:58Z           |
| Status | Draft                          |
| Author | architect                      |

---

## Scope

This plan implements the terminology-only updates defined in
`work/P38-plans-and-batches/P38-spec-p38-f7-documentation-skills-updates.md`
(FEAT-01KQ7YQKWTBRP/spec-p38-spec-p38-f7-documentation-skills-updates).

It covers tasks T1–T7 below. It updates all agent-facing materials to reflect
the plan/batch distinction: roles (`.kbz/roles/`), skills (`.kbz/skills/`),
stage bindings, `AGENTS.md`, `copilot-instructions.md`, and `refs/`.

It does **not** cover: code comments or Go documentation (handled in P38-F2–F6),
creating new documentation, or migrating existing document files (P38-F8).

---

## Task Breakdown

### Task 1: Update Role YAML Files

- **Description:** Review all 18 `.kbz/roles/*.yaml` files and update "plan" terminology
  to "batch" where it refers to the work-grouping container. Add plan/batch vocabulary
  terms to the orchestrator role. Update implementer-go feature-ownership references.
  All other roles are verified for consistency but likely need no changes — the current
  role vocabulary is entity-agnostic, referencing "features" and "tasks" directly.
- **Deliverable:** Updated `.kbz/roles/*.yaml` files with correct plan/batch terminology.
- **Depends on:** None
- **Effort:** Small
- **Spec requirement:** REQ-001, REQ-002, REQ-003, REQ-004

### Task 2: Update Skill SKILL.md Files

- **Description:** Review all 18 `.kbz/skills/**/SKILL.md` files. Update write-design to
  document plan-level vs batch-level design ownership. Update write-spec, orchestrate-development,
  orchestrate-review, and decompose-feature to reference "batch" as the parent entity for
  features. Verify remaining skills for consistency. Current skills are largely entity-agnostic;
  most changes will be targeted vocabulary additions in the five named skills.
- **Deliverable:** Updated `.kbz/skills/**/SKILL.md` files with correct plan/batch terminology.
- **Depends on:** None
- **Effort:** Medium
- **Spec requirement:** REQ-005, REQ-006, REQ-007, REQ-008, REQ-009, REQ-010

### Task 3: Update Stage Bindings

- **Description:** Update `.kbz/stage-bindings.yaml` entity type references. The
  `plan-reviewing` stage references the strategic plan entity (correct — stays as
  "plan"). The `dev-planning` and `developing` stages reference dev-plan documents
  (correct — dev-plan is a document type, not an entity). Verify all notes and
  descriptions use correct terminology. The "plan" in notes like "Architect role
  owns the plan" refers to the dev-plan document (stays). Update any entity-type
  references in document prerequisites.
- **Deliverable:** Updated `.kbz/stage-bindings.yaml`.
- **Depends on:** None
- **Effort:** Small
- **Spec requirement:** REQ-011, REQ-012

### Task 4: Update AGENTS.md

- **Description:** Update the entity hierarchy section to show: Plan → Batch → Feature → Task.
  Update the repository structure diagram: `state/plans/` → `state/batches/`, and add
  `state/plans/` as a new entry for the strategic plan entity. Update the service layer
  comment. Add guidance distinguishing when to use a plan vs a batch. Update the "Key Terms"
  table and "Naming Conventions" if needed. The `work/plan/` directory reference (for
  implementation plans) stays — it's a document directory, not an entity type.
- **Deliverable:** Updated `AGENTS.md` with correct entity hierarchy, repo structure, and usage guidance.
- **Depends on:** None
- **Effort:** Medium
- **Spec requirement:** REQ-013, REQ-014

### Task 5: Update copilot-instructions.md

- **Description:** Update `.github/copilot-instructions.md`. The skills table and roles
  table reference "plan-reviewing" (correct — strategic plan review) and "review-plan"
  (correct — skill name). The "Planning" skill entry in the workflow guides table
  references "feature vs plan decisions" — update to "feature vs batch vs plan decisions"
  or clarify the distinction. The "kanbanzai-planning" skill path stays. Verify all
  references are correct. Update the skills table description to reference "batch" where
  the work container is meant.
- **Deliverable:** Updated `.github/copilot-instructions.md`.
- **Depends on:** None
- **Effort:** Small
- **Spec requirement:** REQ-015, REQ-016

### Task 6: Update Reference Files

- **Description:** Review and update all `refs/*.md` files. The primary file needing
  changes is `refs/document-map.md` (references `work/plan/` — stays as-is since it's
  a document directory). `refs/sub-agents.md`, `refs/testing.md`, `refs/go-style.md`,
  and `refs/knowledge-graph.md` need verification for plan/batch consistency. Most
  refs files are entity-agnostic; changes will be minimal.
- **Deliverable:** Updated `refs/*.md` files with consistent plan/batch terminology.
- **Depends on:** None
- **Effort:** Small
- **Spec requirement:** REQ-017, REQ-018, REQ-019, REQ-020, REQ-021

### Task 7: Grep Audit for Consistency

- **Description:** Run a comprehensive grep across all updated files to verify:
  (a) "plan" only appears in reference to the strategic planning entity (not the
  work container), (b) "batch" is used consistently for the work-grouping entity,
  (c) no single file mixes "plan" and "batch" for the same concept. Report any
  inconsistencies found for remediation. This is the verification step for AC-009
  and AC-010.
- **Deliverable:** Clean grep report showing zero false positives, or a list of
  inconsistencies to fix.
- **Depends on:** Task 1, Task 2, Task 3, Task 4, Task 5, Task 6
- **Effort:** Small
- **Spec requirement:** AC-009, AC-010

---

## Dependency Graph

```
Task 1 (no dependencies)
Task 2 (no dependencies)
Task 3 (no dependencies)
Task 4 (no dependencies)
Task 5 (no dependencies)
Task 6 (no dependencies)
Task 7 → depends on Task 1, Task 2, Task 3, Task 4, Task 5, Task 6

Parallel groups: [Task 1, Task 2, Task 3, Task 4, Task 5, Task 6]
Critical path: Any of T1–T6 → Task 7
```

All six update tasks are independent — they touch disjoint file sets:
- T1: `.kbz/roles/*.yaml`
- T2: `.kbz/skills/**/SKILL.md`
- T3: `.kbz/stage-bindings.yaml`
- T4: `AGENTS.md`
- T5: `.github/copilot-instructions.md`
- T6: `refs/*.md`

T7 (grep audit) depends on all six completing first.

---

## Risk Assessment

### Risk: Inconsistent Terminology Across Parallel Tasks

- **Probability:** Medium
- **Impact:** High — if two tasks choose different conventions for the same term,
  the grep audit (T7) will catch it but rework will be needed.
- **Mitigation:** Establish clear conventions upfront: "batch" = operational work
  container (renamed from old "plan"), "plan" = strategic recursive planning entity.
  Each task's spec requirements define exactly which terms to use.
- **Affected tasks:** Task 1, Task 2, Task 3, Task 4, Task 5, Task 6

### Risk: File Conflicts Between Parallel Tasks

- **Probability:** Low
- **Impact:** Low — the six update tasks touch disjoint file sets. No file is
  modified by more than one task.
- **Mitigation:** File scope boundaries are naturally separated by the decomposition.
- **Affected tasks:** None (disjoint file sets)

### Risk: Missing "Plan" References in Roles/Skills

- **Probability:** Medium
- **Impact:** Low — roles and skills currently contain few or no references to
  "plan" as a work container. If a reference is missed during review, the grep
  audit (T7) will surface it.
- **Mitigation:** T7 is the safety net. Each task includes a self-review step
  before completion.
- **Affected tasks:** Task 1, Task 2

---

## Verification Approach

| Acceptance Criterion | Verification Method | Producing Task |
|---------------------|--------------------| ---------------|
| AC-001: Roles use correct terminology | Manual inspection | Task 1 |
| AC-002: Skills use correct terminology | Manual inspection | Task 2 |
| AC-003: write-design documents plan/batch ownership | Manual inspection | Task 2 |
| AC-004: Stage bindings have correct entity references | Manual inspection | Task 3 |
| AC-005: AGENTS.md entity hierarchy shows Plan→Batch→Feature→Task | Manual inspection | Task 4 |
| AC-006: AGENTS.md includes plan-vs-batch guidance | Manual inspection | Task 4 |
| AC-007: copilot-instructions.md tables use correct terminology | Manual inspection | Task 5 |
| AC-008: Reference files use consistent terminology | Manual inspection | Task 6 |
| AC-009: Grep returns only strategic-plan references for "plan" | Automated grep | Task 7 |
| AC-010: No file mixes "plan"/"batch" for same concept | Per-file scan | Task 7 |

---

## Traceability Matrix

| Spec Requirement | Task |
|-----------------|------|
| REQ-001 (roles: plan→batch) | Task 1 |
| REQ-002 (roles: entity hierarchy) | Task 1 |
| REQ-003 (orchestrator: batch+plan vocab) | Task 1 |
| REQ-004 (implementer-go: batch refs) | Task 1 |
| REQ-005 (skills: plan→batch) | Task 2 |
| REQ-006 (write-design: plan+batch ownership) | Task 2 |
| REQ-007 (write-spec: batch as parent) | Task 2 |
| REQ-008 (orchestrate-development: batch) | Task 2 |
| REQ-009 (orchestrate-review: batches) | Task 2 |
| REQ-010 (decompose-feature: batches) | Task 2 |
| REQ-011 (stage-bindings: entity refs) | Task 3 |
| REQ-012 (stage-bindings: prerequisites) | Task 3 |
| REQ-013 (AGENTS.md: entity hierarchy) | Task 4 |
| REQ-014 (AGENTS.md: plan-vs-batch guidance) | Task 4 |
| REQ-015 (copilot-instructions: tables) | Task 5 |
| REQ-016 (copilot-instructions: plan→batch) | Task 5 |
| REQ-017 (refs/go-style.md) | Task 6 |
| REQ-018 (refs/testing.md) | Task 6 |
| REQ-019 (refs/sub-agents.md) | Task 6 |
| REQ-020 (refs/document-map.md) | Task 6 |
| REQ-021 (refs/knowledge-graph.md) | Task 6 |
| AC-009 (grep audit) | Task 7 |
| AC-010 (per-file consistency) | Task 7 |
