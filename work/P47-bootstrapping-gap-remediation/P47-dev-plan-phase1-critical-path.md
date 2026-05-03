# Dev Plan: Phase 1 — Critical Path (G1 + G2)

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-03                     |
| Status | Draft                          |
| Author | Architect (AI)                 |

> This plan implements requirements FR-001 and FR-002 from
> `work/P47-bootstrapping-gap-remediation/spec-bootstrapping-gap-remediation.md`.

---

## Scope

This plan covers embedding `stage-bindings.yaml` and all 20 task-execution skill files into the `kbz` binary, adding install logic to `kbz init` for both artifact types, and wiring the install into `runNewProject`, `runExistingProject`, and `--update-skills`. These two gaps (G1 and G2) are co-requisites — installing `stage-bindings.yaml` without skills would cause the 3.0 pipeline to fail at step 6 ("skill not found"), and installing skills without stage-bindings means the pipeline never activates at all.

This plan does **not** cover: role installation beyond the existing 2 roles (Phase 2), AGENTS.md enrichment (Phase 3), copilot-instructions reconciliation (Phase 3), extension of `--update-skills` to non-skill artifacts (Phase 4), or CI staleness checks (Phase 4).

---

## Task Breakdown

### Task 1: Embed stage-bindings.yaml and write install logic

- **Description:** Create `internal/kbzinit/stage-bindings.yaml` (copy from `.kbz/stage-bindings.yaml`), add `//go:embed stage-bindings.yaml` directive in a new or existing file, implement `installStageBindings(kbzDir string)` with version-aware create/update/skip logic using YAML comment markers (`# kanbanzai-managed: true` and `# kanbanzai-version: N`), and add unit tests for create/update/skip/unmanaged cases.
- **Deliverable:** New file `internal/kbzinit/stage-bindings.yaml`, new or modified file with embed directive and `installStageBindings` function, new test cases in `internal/kbzinit/init_test.go`.
- **Depends on:** None
- **Effort:** Medium
- **Spec requirement:** FR-001

### Task 2: Embed task-execution skills and write install logic

- **Description:** Create `internal/kbzinit/skills/task-execution/` directory containing all 20 skill subdirectories copied from `.kbz/skills/` (excluding `CONVENTIONS.md`), create `internal/kbzinit/task_skills.go` with `//go:embed skills/task-execution` directive and `installTaskSkills(baseDir string)` function. Reuse the existing `transformSkillContent` logic from `skills.go` since task-execution skills use the same frontmatter format. Add unit tests mirroring the pattern from the workflow skills tests.
- **Deliverable:** New directory `internal/kbzinit/skills/task-execution/` with 20 skill subdirectories, new file `internal/kbzinit/task_skills.go`, new test cases.
- **Depends on:** None (can be done in parallel with Task 1)
- **Effort:** Large (20 skill files to copy and validate)
- **Spec requirement:** FR-002

### Task 3: Wire install calls into runNewProject and runExistingProject

- **Description:** Add `installStageBindings(kbzDir)` and `installTaskSkills(baseDir)` calls to both `runNewProject()` and `runExistingProject()` in `internal/kbzinit/init.go`. The calls go after config write and before/alongside existing skill install. Ensure `--skip-skills` suppresses task-execution skill installation (it already suppresses workflow skill installation). Ensure stage-bindings installation is not gated by `--skip-skills` (it's a config file, not a skill).
- **Deliverable:** Modified `internal/kbzinit/init.go` with new install calls.
- **Depends on:** Task 1, Task 2
- **Effort:** Small
- **Spec requirement:** FR-001, FR-002

### Task 4: Wire into --update-skills path

- **Description:** Add `installStageBindings(kbzDir)` and `installTaskSkills(baseDir)` calls to the `--update-skills` path in `Run()`. This is the initial wiring — Phase 4 Task 1 will extend this further for roles.
- **Deliverable:** Modified `internal/kbzinit/init.go` with new calls in `--update-skills` path.
- **Depends on:** Task 1, Task 2, Task 3
- **Effort:** Small
- **Spec requirement:** FR-001, FR-002 (partial FR-006 — full coverage in Phase 4)

### Task 5: Integration test — fresh init produces pipeline-ready project

- **Description:** Add an integration test that: creates a fresh git repo with one commit, runs `kbz init`, asserts `stage-bindings.yaml` exists and passes validation, asserts all 20 task-execution skills load via `SkillStore`, and asserts the pipeline can be constructed (stage-bindings loads). Uses the `integration` build tag.
- **Deliverable:** New file `internal/kbzinit/pipeline_readiness_test.go` (integration build tag).
- **Depends on:** Task 3
- **Effort:** Medium
- **Spec requirement:** FR-008

---

## Dependency Graph

```
Task 1 (embed stage-bindings)    Task 2 (embed task skills)
        \                               /
         \                             /
          Task 3 (wire into init flow)
                 |
          Task 4 (wire --update-skills)
                 |
          Task 5 (integration test)
```

- **Parallel groups:** [Task 1, Task 2]
- **Critical path:** Task 2 → Task 3 → Task 4 → Task 5

---

## Risk Assessment

### Risk: Task-execution skill content has validation errors
- **Probability:** Low
- **Impact:** High — would block installation
- **Mitigation:** The audit already verified all 20 skills load via `SkillStore.LoadAll()` in this project. The copy is byte-for-byte. Task 2 should validate each skill programmatically during test.
- **Affected tasks:** Task 2

### Risk: `go:embed` directive naming collision with existing `//go:embed skills`
- **Probability:** Low
- **Impact:** Medium — would require restructuring existing embed
- **Mitigation:** The existing embed targets `skills` (9 workflow skill directories). The new embed targets `skills/task-execution` (a subdirectory). Go's embed supports multiple directives targeting different paths within the same root as long as they don't overlap. Verify during Task 2.
- **Affected tasks:** Task 2

### Risk: Stage-bindings version marker scheme conflicts with YAML parsing
- **Probability:** Low
- **Impact:** Medium — `binding.LoadBindingFile()` might reject comment lines
- **Mitigation:** YAML comment lines (`# ...`) are ignored by the YAML parser. The existing skill frontmatter uses the same pattern. Verify during Task 1.
- **Affected tasks:** Task 1

---

## Verification Approach

| Acceptance Criterion | Verification Method | Producing Task |
|---------------------|--------------------|----------------|
| AC-001 (stage-bindings installed) | Integration test | Task 5 |
| AC-002 (task skills installed) | Integration test | Task 5 |
| FR-001 sub-criteria (version-aware update, skip, unmanaged) | Unit tests | Task 1 |
| FR-002 sub-criteria (version-aware update, skip, unmanaged) | Unit tests | Task 2 |
| FR-001 + FR-002 (--update-skills) | Unit test | Task 4 |
