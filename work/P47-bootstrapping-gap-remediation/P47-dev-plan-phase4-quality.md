# Dev Plan: Phase 4 — Quality Hardening (G5 + G7 + G8)

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-03                     |
| Status | Draft                          |
| Author | Architect (AI)                 |

> This plan implements requirements FR-006, FR-007, and FR-008 (the quality-focused parts) from
> `work/P47-bootstrapping-gap-remediation/spec-bootstrapping-gap-remediation.md`.

---

## Scope

This plan covers the quality hardening work: extending `--update-skills` to cover all managed artifact types (not just workflow skills and reviewer.yaml), adding a CI staleness check for embedded workflow skill seeds, and resolving the codebase-memory skills disposition (G8). This plan depends on Phases 1-3 being complete because it touches upgrade paths across all artifact types and the staleness check compares embedded seeds against project files.

This plan does **not** cover: embedding artifacts (Phase 1, 2), AGENTS.md content (Phase 3), or the core integration test (Phase 1 Task 5 already covers FR-008).

---

## Task Breakdown

### Task 1: Extend --update-skills to cover stage-bindings, task skills, and all roles

- **Description:** The current `--update-skills` path in `Run()` calls `installSkills()` (workflow skills) and `updateManagedRoles()` (reviewer.yaml only). Extend it to also call `installStageBindings(kbzDir)`, `installTaskSkills(baseDir)`, and `installAllRoles(kbzDir)` (replacing the narrow `updateManagedRoles`). All these functions already exist from Phases 1 and 2 — this task just wires them in. Add unit tests for each artifact type in the update path: verify that managed files at older versions are updated, managed files at current version are skipped, and unmanaged files are skipped with warnings.
- **Deliverable:** Modified `internal/kbzinit/init.go` `Run()` method, new test cases.
- **Depends on:** Phase 1 Task 4 (initial wiring), Phase 2 Task 2 (refactored `installRoles`)
- **Effort:** Medium
- **Spec requirement:** FR-006

### Task 2: Resolve codebase-memory skills disposition (G8)

- **Description:** Per the design decision, G8 is deferred to a separate install path. This task documents the decision and updates generated AGENTS.md to direct consumers to copy the graph tool skills from the kanbanzai repository if they use codebase-memory-mcp. The update to `agentsMDContent` is minor: change the "Optional: Code Graph Integration" section to include a pointer to the repo's `.github/skills/` directory. Also add a note in the project's AGENTS.md confirming this intentional deferral.
- **Deliverable:** Minor edit to `agentsMDContent`, annotation in project's `AGENTS.md`.
- **Depends on:** None (content-only)
- **Effort:** Small
- **Spec requirement:** n/a (G8 disposition)

### Task 3: Add CI staleness check for embedded workflow skills

- **Description:** Create `internal/kbzinit/skills_consistency_test.go` with `TestEmbeddedSkillsMatchAgentSkills`. The test iterates all 9 `skillNames`, reads the embedded file from `embeddedSkills.ReadFile("skills/" + name + "/SKILL.md")`, reads the on-disk file from `../../.agents/skills/kanbanzai-" + name + "/SKILL.md"`, normalizes both files' `# kanbanzai-version:` lines to a known value, and asserts byte identity. Run as a regular unit test (no build tag). This catches the dual-source drift documented in G7.
- **Deliverable:** New file `internal/kbzinit/skills_consistency_test.go`.
- **Depends on:** None
- **Effort:** Small
- **Spec requirement:** FR-007

### Task 4: Extend staleness check to task-execution skills and roles

- **Description:** Extend the consistency test pattern from Task 3 to cover task-execution skills (compare embedded `skills/task-execution/<name>/SKILL.md` against `.kbz/skills/<name>/SKILL.md`) and role files (compare embedded `roles/<id>.yaml` against `.kbz/roles/<id>.yaml`). Use the same normalized-comparison approach. This provides comprehensive protection against seed drift for all embedded artifact types.
- **Deliverable:** Extended `skills_consistency_test.go` (or new `roles_consistency_test.go`).
- **Depends on:** Task 3 (pattern established)
- **Effort:** Medium (20 skills + 16 roles to compare)
- **Spec requirement:** FR-007 (extended scope)

### Task 5: End-to-end --update-skills integration test

- **Description:** Add an integration test that: initializes a project with simulated old-style state (stage-bindings absent, only 2 roles, only workflow skills), runs `kbz init --update-skills` with the new binary, and asserts that all managed artifacts are brought to current versions. This is the complement to Phase 1 Task 5 (which tests fresh init). Uses the `integration` build tag.
- **Deliverable:** New integration test case.
- **Depends on:** Task 1
- **Effort:** Medium
- **Spec requirement:** FR-006, FR-008

---

## Dependency Graph

```
Task 2 (G8 disposition)     Task 3 (workflow skills staleness check)
                                       |
                                Task 4 (extend to task skills + roles)
                                       |
Task 1 (--update-skills extension) ────┤
        |
  Task 5 (--update-skills integration test)
```

- **Parallel groups:** [Task 2, Task 3] — both are independent
- **Critical path:** Task 3 → Task 4; Task 1 → Task 5 (two independent chains; Task 1 depends on Phase 2, Task 3 does not)

---

## Risk Assessment

### Risk: --update-skills integration test is fragile
- **Probability:** Medium
- **Impact:** Medium — flaky test would erode CI trust
- **Mitigation:** Use the same test patterns as the existing `init_test.go` which already test init on temp git repos. Keep the old-style state simulation minimal — just the files that differ from a fresh init.
- **Affected tasks:** Task 5

### Risk: Staleness check fails immediately due to existing drift
- **Probability:** Medium — embedded seeds may already be out of date
- **Impact:** Low — test failure is the desired outcome; it identifies drift that needs fixing
- **Mitigation:** If the test fails, update the embedded seeds to match the project files in the same PR. The test is self-healing: once seeds match, the test passes.
- **Affected tasks:** Task 3, Task 4

### Risk: Staleness check path resolution breaks in CI
- **Probability:** Low
- **Impact:** Medium — `../../.agents/skills/` is relative to the test file
- **Mitigation:** Use `runtime.Caller` to resolve the test file path programmatically rather than hardcoding `../../`. This is standard Go practice for tests that need repo-root-relative paths.
- **Affected tasks:** Task 3

---

## Verification Approach

| Acceptance Criterion | Verification Method | Producing Task |
|---------------------|--------------------|----------------|
| AC-005 (--update-skills covers all artifacts) | Integration test | Task 5 |
| FR-006 (stage-bindings updated on --update-skills) | Unit test | Task 1 |
| FR-006 (task skills updated on --update-skills) | Unit test | Task 1 |
| FR-006 (roles updated on --update-skills) | Unit test | Task 1 |
| AC-006 (embedded vs. on-disk consistency) | Unit test | Task 3, Task 4 |
| FR-007 (workflow skills staleness check) | Unit test | Task 3 |
| G8 disposition documented | Inspection | Task 2 |
