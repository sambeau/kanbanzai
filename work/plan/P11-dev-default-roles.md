# P11 Feature C: Default Roles — Dev Plan

| Document | P11 Feature C — Default Roles Dev Plan                               |
|----------|----------------------------------------------------------------------|
| Feature  | `FEAT-01KMWJ3ZQZF5R` — default-roles                                |
| Status   | Draft                                                                |
| Related  | `work/spec/spec-default-roles.md`                                    |
|          | `work/design/fresh-install-experience.md` §5.3                       |
|          | `work/spec/init-command.md`                                          |
|          | `work/spec/reviewer-context-profile-and-skill.md`                    |

---

## 1. Approach

`kbz init` already installs skill files using a version-aware create/update/skip pattern
implemented in `internal/kbzinit/skills.go`. This feature follows the same pattern for
two role YAML files, with one important structural difference: the managed marker lives in
a YAML `metadata` field (`metadata.kanbanzai-managed: "true"`) rather than in a Markdown
frontmatter comment.

The implementation is additive — no existing logic is modified beyond two call sites in
`init.go` and one flag in `init_cmd.go`. All new logic lives in a new file `roles.go`
alongside `skills.go`.

Key behavioural rules:
- `base.yaml` is a user-owned scaffold: created once if absent, never touched again. No
  managed marker, no version field.
- `reviewer.yaml` is managed: created on new projects; on re-init, skipped with a warning
  if unmanaged, overwritten if at an older version, no-op if at the current version.
- `--skip-roles` gates `installRoles` entirely (both files).
- `--update-skills` is extended to call `updateManagedRoles`, which updates `reviewer.yaml`
  under the same managed/version rules, and never touches `base.yaml`.

---

## 2. Task Breakdown

| # | Task | Files Touched | Estimate |
|---|------|---------------|----------|
| 1 | Embed role content and implement role writer | `internal/kbzinit/roles.go` (new), `internal/kbzinit/init.go` | M |
| 2 | Add `--skip-roles` flag and extend `--update-skills` | `internal/kbzinit/init.go`, `cmd/kanbanzai/init_cmd.go` | S |
| 3 | Tests | `internal/kbzinit/init_test.go` | M |

---

## 3. Task Details

### Task 1 — Embed role content and implement role writer

**New file: `internal/kbzinit/roles.go`**

Embed the two role file bodies as Go string constants (or via `go:embed` into a sub-
directory `roles/`). The embedded content must match the canonical content defined in spec
§3.1 and §3.2 exactly.

Implement `installRoles(i *Initializer, baseDir, version string) error`:

1. Resolve the roles directory: `<baseDir>/.kbz/context/roles/`.
2. Call `writeBaseRole(rolesDir)`:
   - If `base.yaml` already exists → return nil (no-op, no warning).
   - Otherwise create the directory if needed and write the scaffold content.
3. Call `writeReviewerRole(rolesDir, version)`:
   - If `reviewer.yaml` does not exist → create it with the canonical content (version set
     to the binary's version string).
   - If it exists without `metadata.kanbanzai-managed: "true"` → print a warning to stderr
     naming the file, return nil (skip, no error).
   - If it exists with the managed marker and an older version → overwrite with canonical
     content at current version.
   - If it exists with the managed marker at the current version → return nil (no-op).

Version comparison reuses the `isNewerSchemaVersion` helper already in `init.go` (or an
equivalent semver-free string comparison — both the existing skill version logic and this
feature treat `"dev"` as a special case that always triggers an overwrite when the binary
version differs).

Implement `updateManagedRoles(i *Initializer, baseDir, version string) error`:
- Calls `writeReviewerRole` only (base.yaml is never updated).

The reviewer.yaml content to embed carries:
- `id: reviewer`, `inherits: base`
- `description` field
- `metadata.kanbanzai-managed: "true"` and `metadata.version: <version>`
- `conventions` block with exactly three sub-keys: `review_approach`, `output_format`,
  `dimensions` — content as specified in spec §3.2
- `output_format` entries reference the `kanbanzai-review` skill by name

**Changes to `internal/kbzinit/init.go`**

- In `runNewProject`: after `installSkills`, call `installRoles` (gated on
  `!opts.SkipRoles`).
- In `Run` under the `opts.UpdateSkills` branch: call `updateManagedRoles` after
  `installSkills`.
- Add `SkipRoles bool` to `Options`.

---

### Task 2 — Add `--skip-roles` flag and extend `--update-skills`

**`internal/kbzinit/init.go`**

- Add `SkipRoles bool` to `Options` struct (alongside `SkipSkills`).
- Gate `installRoles` call in `runNewProject` on `!opts.SkipRoles`.
- In the `--update-skills` path, call `updateManagedRoles` after `installSkills`.

**`cmd/kanbanzai/init_cmd.go`**

- Parse `--skip-roles` flag → set `opts.SkipRoles = true`.
- Add `--skip-roles` to the usage text block.
- Update `--update-skills` help text to note that it also updates managed role files.

---

### Task 3 — Tests

**`internal/kbzinit/init_test.go`**

Cover all spec §4 acceptance criteria with unit/integration tests using `t.TempDir()` and
a real git repo (following the `makeGitRepoNoCommits` / `makeGitRepoWithCommit` pattern
already in the file):

| Test | Verifies (spec AC) |
|------|--------------------|
| `TestInstallRoles_NewProject_BothFilesCreated` | 4.1.1, 4.2.1 — both files present after `kbz init` on a new project |
| `TestInstallRoles_BaseContent` | 4.1.2, 4.1.3, 4.1.4 — `id: base`, `description`, `conventions: []`, required comments, no managed marker |
| `TestInstallRoles_ReviewerContent` | 4.2.2, 4.2.3, 4.2.4, 4.2.5 — `id: reviewer`, `inherits: base`, managed marker, version field, three convention sub-keys, `kanbanzai-review` reference |
| `TestInstallRoles_BaseNotOverwritten` | 4.1.5 — pre-existing `base.yaml` with custom content is unchanged after re-init |
| `TestInstallRoles_ReviewerSkippedIfUnmanaged` | 4.2.6 — unmanaged `reviewer.yaml` skipped; stderr warning names the file |
| `TestInstallRoles_ReviewerOverwrittenIfOlderVersion` | 4.2.7 — managed `reviewer.yaml` at lower version overwritten |
| `TestInstallRoles_ReviewerNoOpIfCurrentVersion` | 4.2.8 — managed `reviewer.yaml` at current version unchanged |
| `TestInstallRoles_DeveloperAbsent` | 4.3.1 — `developer.yaml` not created |
| `TestInstallRoles_SkipRolesFlag` | 4.4.1, 4.4.2 — `--skip-roles` skips both files, exit status 0 |
| `TestUpdateSkills_UpdatesManagedReviewer` | 4.4.3 — `--update-skills` updates managed `reviewer.yaml` when older |
| `TestUpdateSkills_DoesNotTouchBase` | 4.4.4 — `--update-skills` leaves `base.yaml` untouched |
| `TestProfileResolution_ReviewerAfterInit` | 4.5.1 — `context.Assemble(role="reviewer")` on a freshly inited project returns a non-empty packet with all three convention keys |

The profile resolution test (`TestProfileResolution_ReviewerAfterInit`) wires the
`internal/context` assembler directly against a temp `.kbz/` directory created by the
`Initializer` (no MCP server required).

---

## 4. Dependencies

| Dependency | Status | Notes |
|------------|--------|-------|
| `internal/kbzinit/skills.go` — managed marker and version logic | Complete | `roles.go` can reuse `isNewerSchemaVersion` from `init.go`; the YAML-based managed check is new but analogous |
| `internal/context/` — profile resolution and assembly | Complete | Required only for Task 3 integration test (AC 4.5.1); no code changes needed |
| Spec §3.2 canonical `reviewer.yaml` content | Approved | Embedded content must match spec §3.2 exactly — treat it as a fixture, not free text |
| Feature B (skills consolidation) | Independent | No ordering dependency; `--skip-roles` and `--skip-skills` are independent flags |

No cross-feature blocking dependencies. Task 2 depends on Task 1 (`SkipRoles` field must
exist before the flag can be parsed). Task 3 can be written in parallel with Tasks 1–2 but
must run against the completed implementation.