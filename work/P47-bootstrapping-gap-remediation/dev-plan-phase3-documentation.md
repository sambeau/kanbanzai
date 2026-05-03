# Dev Plan: Phase 3 — Documentation Enrichment (G4 + G6)

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-03                     |
| Status | Draft                          |
| Author | Architect (AI)                 |

> This plan implements requirements FR-004, FR-005, and FR-009 from
> `work/P47-bootstrapping-gap-remediation/spec-bootstrapping-gap-remediation.md`.

---

## Scope

This plan covers enriching the generated `AGENTS.md` content with task-execution skill and role reference tables, enriching the generated `.github/copilot-instructions.md` with stage-bindings guidance, bumping the `agentsMDVersion` from 2 to 3 so existing managed files get the update, and reconciling the project's hand-maintained `.github/copilot-instructions.md` to not mislead readers about artifacts that don't exist after init.

This plan is independent of Phase 1 and Phase 2 — it only edits string constants in `internal/kbzinit/agents_md.go` and the project's `.github/copilot-instructions.md`. However, it references skill and role names that Phase 1 and Phase 2 install, so it should merge after both are complete to avoid referencing artifacts not yet installable.

This plan does **not** cover: embedding artifacts (Phase 1, 2), `--update-skills` extension (Phase 4), or CI checks (Phase 4).

---

## Task Breakdown

### Task 1: Add task-execution skill table to generated AGENTS.md

- **Description:** Edit `agentsMDContent` in `internal/kbzinit/agents_md.go` to add a "Task-Execution Skills" table below the existing "Skills Reference" table. The table maps each of the 17 skills referenced in `stage-bindings.yaml` to its `.kbz/skills/` path and the workflow stage(s) it's bound to. Derive the mapping from `stage-bindings.yaml` (the canonical source). Include only skills from `stage-bindings.yaml` — skills like `prompt-engineering` and `write-skill` that exist in `.kbz/skills/` but aren't bound to any stage are omitted from the table.
- **Deliverable:** Modified `agentsMDContent` constant with new table.
- **Depends on:** None (content-only)
- **Effort:** Small
- **Spec requirement:** FR-004

### Task 2: Add role table to generated AGENTS.md

- **Description:** Edit `agentsMDContent` to add a "Roles" table below the task-execution skills table. The table maps each of the 15 roles referenced in `stage-bindings.yaml` to its `.kbz/roles/` path and the workflow stage(s) it's bound to. Derive from `stage-bindings.yaml`. Add a "Stage Bindings" section noting that `.kbz/stage-bindings.yaml` is the authoritative source of truth and should be read before entering any stage.
- **Deliverable:** Modified `agentsMDContent` constant with new table and section.
- **Depends on:** Task 1 (same file)
- **Effort:** Small
- **Spec requirement:** FR-004

### Task 3: Bump agentsMDVersion to 3

- **Description:** Increment `agentsMDVersion` from 2 to 3 in `internal/kbzinit/agents_md.go`. This causes existing managed AGENTS.md and copilot-instructions.md files to be overwritten with the new content on next `kbz init` run. The version-aware logic in `writeMarkdownConfig` handles the update automatically. Verify with a unit test: write a v2 managed file, run init, assert it's updated to v3.
- **Deliverable:** One-line constant change, unit test.
- **Depends on:** Task 1, Task 2
- **Effort:** Small
- **Spec requirement:** FR-004, FR-005

### Task 4: Enrich generated copilot-instructions.md with stage-bindings guidance

- **Description:** Edit `copilotInstructionsContent` to add: "Read `.kbz/stage-bindings.yaml` before starting any workflow stage to find the correct role and skill." Ensure the file remains under 25 lines after the addition.
- **Deliverable:** Modified `copilotInstructionsContent` constant.
- **Depends on:** None (independent of Tasks 1-3, but shares the version bump)
- **Effort:** Small
- **Spec requirement:** FR-005

### Task 5: Reconcile project's copilot-instructions.md

- **Description:** Either trim `.github/copilot-instructions.md` to reference only artifacts that exist after a consumer init, OR add a prominent comment marking it as kanbanzai-project-local. The spec permits either approach. The simpler option: add `<!-- kanbanzai-project: this file contains project-local references not present in generated installs. See internal/kbzinit/agents_md.go for the canonical consumer version. -->` at the top of the file. Also verify the file does not claim that `.kbz/skills/`, `.kbz/roles/`, `.github/skills/`, `refs/`, or `work/templates/` exist after consumer init without also noting they are project-local.
- **Deliverable:** Modified `.github/copilot-instructions.md`.
- **Depends on:** None
- **Effort:** Small
- **Spec requirement:** FR-009

### Task 6: Verify generated content line counts and formatting

- **Description:** Write a unit test that verifies `agentsMDContent` does not exceed 100 lines and `copilotInstructionsContent` does not exceed 25 lines. Verify both strings start with the managed marker comment on line 1. Verify the task-execution skill table has exactly 17 entries (matching stage-bindings references). Verify the role table has exactly 15 entries.
- **Deliverable:** New test cases in `internal/kbzinit/agents_md_test.go`.
- **Depends on:** Task 1, Task 2, Task 3, Task 4
- **Effort:** Small
- **Spec requirement:** FR-004, FR-005

---

## Dependency Graph

```
Task 1 (skill table) ──┬── Task 2 (role table) ──┬── Task 3 (version bump)
                       │                          │
Task 4 (copilot) ──────┘                          │
                                                  │
Task 5 (project reconciliation) ──────────────────┤
                                                  │
                                          Task 6 (validation tests)
```

- **Parallel groups:** [Task 1, Task 4, Task 5] — all three are independent content edits
- **Critical path:** Task 1 → Task 2 → Task 3 → Task 6

---

## Risk Assessment

### Risk: Generated AGENTS.md exceeds 100-line budget with new tables
- **Probability:** Medium
- **Impact:** Low — the budget is soft; exceeding by a few lines is acceptable per spec
- **Mitigation:** Task 6 verifies line count. If the 100-line budget is tight, compress table formatting (remove verbose path prefixes, use terse column headers).
- **Affected tasks:** Task 1, Task 2, Task 6

### Risk: Skill/role counts in generated tables drift from stage-bindings.yaml
- **Probability:** Medium — if someone adds a stage binding without updating AGENTS.md content
- **Mitigation:** Task 6 verifies counts match. The Phase 4 staleness check could be extended to cover this. Document the dual-write rule in the project AGENTS.md.
- **Affected tasks:** Task 1, Task 2

### Risk: Project copilot-instructions.md becomes out of sync with reality again
- **Probability:** Medium — historically it has
- **Mitigation:** The project-local marker makes the status explicit. Any future additions to the file that reference artifacts not in the install path will be visible during review.
- **Affected tasks:** Task 5

---

## Verification Approach

| Acceptance Criterion | Verification Method | Producing Task |
|---------------------|--------------------|----------------|
| AC-004 (AGENTS.md has skill + role tables) | Inspection + unit test | Task 1, Task 2, Task 6 |
| FR-004 (Stage Bindings section present) | Inspection | Task 2 |
| FR-004 (agentsMDVersion bumped to 3) | Unit test | Task 3 |
| FR-005 (copilot has stage-bindings guidance) | Inspection + unit test | Task 4, Task 6 |
| AC-010 (project copilot marked as project-local) | Inspection | Task 5 |
| FR-004 + FR-005 (line count limits) | Unit test | Task 6 |
