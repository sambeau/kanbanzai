# Dev Plan: Phase 2 — Role Completion (G3)

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-03                     |
| Status | Draft                          |
| Author | Architect (AI)                 |

> This plan implements requirement FR-003 from
> `work/P47-bootstrapping-gap-remediation/spec-bootstrapping-gap-remediation.md`.

---

## Scope

This plan covers embedding all 16 additional role YAML files into the `kbz` binary and extending the existing role installation logic in `internal/kbzinit/roles.go` to install them. Currently, `kbz init` installs only `base.yaml` (scaffold, never overwritten) and `reviewer.yaml` (managed, version-aware update). The remaining 16 roles — `architect`, `spec-author`, `implementer`, `implementer-go`, `orchestrator`, `reviewer-conformance`, `reviewer-quality`, `reviewer-security`, `reviewer-testing`, `researcher`, `documenter`, `doc-pipeline-orchestrator`, `doc-editor`, `doc-checker`, `doc-stylist`, `doc-copyeditor` — exist in this project's `.kbz/roles/` but have no embedded source.

All 16 new roles are managed (version-aware update). Only `base` remains user-owned scaffold.

This plan does **not** cover: stage-bindings or task skills (Phase 1), AGENTS.md enrichment (Phase 3), `--update-skills` extension (Phase 4), or CI staleness checks (Phase 4).

---

## Task Breakdown

### Task 1: Create embedded roles directory and add go:embed directive

- **Description:** Create `internal/kbzinit/roles/` directory containing all 16 role YAML files copied from `.kbz/roles/` (excluding `base.yaml` and `reviewer.yaml` which are already handled by `roles.go`). Add `//go:embed roles` directive. Each role file keeps its existing content — no content changes.
- **Deliverable:** New directory `internal/kbzinit/roles/` with 16 YAML files, updated `internal/kbzinit/roles.go` with embed directive.
- **Depends on:** None
- **Effort:** Medium (16 files to copy and verify)
- **Spec requirement:** FR-003

### Task 2: Extend installRoles to iterate all embedded roles

- **Description:** Refactor `installRoles()` and related functions in `internal/kbzinit/roles.go` to iterate all embedded role files instead of hardcoding `base` and `reviewer`. Classification: roles carrying `kanbanzai-managed: "true"` in metadata are managed (version-aware update); roles without the marker are scaffold (create once, never overwrite). The existing `base` role has no `kanbanzai-managed` marker — scaffold behavior preserved. All 16 new roles must carry the managed marker. The existing `reviewer` role already carries it.
- **Deliverable:** Refactored `internal/kbzinit/roles.go` with generic iteration logic.
- **Depends on:** Task 1
- **Effort:** Medium
- **Spec requirement:** FR-003

### Task 3: Update install path to .kbz/roles/ (new location)

- **Description:** The existing `installRoles` writes to `.kbz/context/roles/` (the legacy path). Update the destination to `.kbz/roles/` (the new canonical location from the 3.0 role system). The `RoleStore` already checks `.kbz/roles/` first and falls back to `.kbz/context/roles/`, so switching the install target is backward-compatible. Ensure `base.yaml` and `reviewer.yaml` are also written to the new location. Maintain the legacy path as a deprecated fallback: if the legacy path exists and the new path does not, install to the new path without touching the legacy path.
- **Deliverable:** Modified destination paths in `roles.go`.
- **Depends on:** Task 2
- **Effort:** Small
- **Spec requirement:** FR-003

### Task 4: Ensure --skip-roles suppresses all roles

- **Description:** Verify that the existing `--skip-roles` flag suppresses installation of all roles (not just `base` and `reviewer`). The flag already gates the `installRoles` call in both `runNewProject` and `runExistingProject`. With the refactored `installRoles`, this should be automatic — but add a test to confirm.
- **Deliverable:** Test case confirming `--skip-roles` suppresses all 18 roles.
- **Depends on:** Task 2, Task 3
- **Effort:** Small
- **Spec requirement:** FR-003

### Task 5: Add managed marker to role files that lack it

- **Description:** Audit the 16 role files copied from `.kbz/roles/`. Verify each one has `kanbanzai-managed: "true"` and `version: "<version>"` in its metadata block. If any file lacks these markers, add them. The audit report confirms these files exist but the marker presence needs verification.
- **Deliverable:** Updated role YAML files with consistent managed markers.
- **Depends on:** Task 1
- **Effort:** Small
- **Spec requirement:** FR-003

---

## Dependency Graph

```
Task 1 (embed roles) ──┬── Task 5 (add managed markers)
                       │
                       ├── Task 2 (refactor installRoles)
                       │        │
                       │   Task 3 (update install path)
                       │        │
                       └────────┤
                           Task 4 (--skip-roles test)
```

- **Parallel groups:** [Task 5, Task 2] (Task 5 is content-only, Task 2 is code-only — both depend on Task 1)
- **Critical path:** Task 1 → Task 2 → Task 3 → Task 4

---

## Risk Assessment

### Risk: Role files lack `kanbanzai-managed` marker
- **Probability:** Medium
- **Impact:** Medium — unmanaged roles would be skipped on re-run, never updated
- **Mitigation:** Task 5 explicitly audits and adds markers. Also add a validation step in Task 2 that warns if an embedded role lacks a managed marker (except `base`).
- **Affected tasks:** Task 1, Task 5

### Risk: Legacy path compatibility — roles at both .kbz/roles/ and .kbz/context/roles/
- **Probability:** Medium — projects initialized with older kbz have roles at legacy path
- **Impact:** Low — `RoleStore` already handles dual locations correctly
- **Mitigation:** Task 3 writes only to new location. The `RoleStore` dual-path resolution handles the overlap transparently. No migration of existing files needed.
- **Affected tasks:** Task 3

### Risk: Role YAML content fails RoleStore validation after embedding
- **Probability:** Low
- **Impact:** High — would prevent installation
- **Mitigation:** All 18 roles already pass `RoleStore.LoadAll()` in this project. The embedded copy is byte-for-byte. Task 2 tests should validate programmatically.
- **Affected tasks:** Task 2

---

## Verification Approach

| Acceptance Criterion | Verification Method | Producing Task |
|---------------------|--------------------|----------------|
| FR-003 (all 18 roles installed) | Integration test (Phase 1 Task 5 extended) | Task 3 |
| FR-003 (version-aware update for managed roles) | Unit tests | Task 2 |
| FR-003 (base.yaml never overwritten) | Unit test | Task 2 |
| FR-003 (--skip-roles suppresses all) | Unit test | Task 4 |
| FR-003 (managed markers present) | Inspection + unit test | Task 5 |
