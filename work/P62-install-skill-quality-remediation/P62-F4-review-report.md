# P62-F4 Review Report — Install E2E Tests & `kbz doctor`

| Field | Value |
|-------|-------|
| Feature | FEAT-01KR7BKXPXFSK (install-tests-doctor) |
| Batch | B64-install-skill-quality |
| Plan | P62-install-skill-quality-remediation |
| Reviewer | reviewer-conformance |
| Date | 2026-05-10 |
| Spec | `work/P62-install-skill-quality-remediation/P62-F4-spec-install-tests-doctor.md` |
| Dev-plan | `work/P62-install-skill-quality-remediation/P62-F4-dev-plan-install-tests-doctor.md` |

## Overall Verdict: CHANGES_REQUESTED

Three blocking conformance gaps and four non-blocking observations. The blocking gaps are
all fixable without design changes. See the Conformance Gaps table below.

---

## Checklist Verification

| # | Item | Status | Evidence |
|---|------|--------|----------|
| 1 | `--skip-agents-md` renamed to `--skip-instructions` with deprecation | ✅ | Commit `8439ed3d`; `SkipInstructions` field + `SkipAgentsMD` deprecated alias with stderr warning in `init.go` L48–58, L96–104 |
| 2 | `--skip-mcp` split, `--skip-zed` added | ✅ | Commit `d460ee1e`; `SkipZed` field, `SkipMCP` narrowed to `.mcp.json` only, deprecation warning for combined behaviour in `init.go` L40–49, L106–110 |
| 3 | Rollback widened | ✅ | Commit `12d0c84b`; `trackedPaths []string` in `runNewProject` covers `.kbz/`, `.agents/`, `.github/`, `AGENTS.md`, `.mcp.json`, `.zed/settings.json`, `work/`, roles, stage-bindings; reverse-order rollback on failure in `init.go` L234–248 |
| 4 | `kbz doctor` subcommand functional | ✅ | Commit `5e8ab483`; `internal/kbzdoctor/` package with required-file checking, marker verification, ghost-file detection, exit 0/1 logic; wired into CLI via `cmd/kbz/doctor_cmd.go` |
| 5 | `make test-install` target exists | ✅ | Commit `87f6676f`; `Makefile` L33: `go test ./internal/kbzinit -tags=e2e -race -run TestE2E_ -count=1` |
| 6 | e2e harness + core e2e tests + flag e2e tests | ✅ | Commits `8558e610` (harness), `e723f47f` (core tests), `3f692951` (flag tests); all 16 test functions in `e2e_test.go` match spec REQ-003 list |
| 7 | `kbz doctor` integration tests | ✅ | Commit `651a3126`; `TestE2E_Doctor_MissingRequiredArtifact`, `TestE2E_Doctor_GhostFile` in e2e_test.go; 7 unit tests in `doctor_test.go` including `TestDoctor_OlderMarkerVersion` |
| 8 | GitHub Actions CI job | ⚠️ | Commit `327b176a`; job exists in `.github/workflows/ci.yml` but **lacks the `paths` filter** specified in dev-plan T11 — runs on all PRs instead of only those touching `internal/kbzinit/**` |

## Conformance Gaps

| # | Severity | Category | Description | Spec Anchor |
|---|----------|----------|-------------|-------------|
| CG-1 | **blocking** | task-status | Task T1 (D7 fix — guard "incomplete init" warning) is `queued` with no commit. The dev-plan allows T1 to be marked `not-planned` if F1 already addressed D7, but the task was never transitioned. T1 has neither been implemented nor formally deferred. | Dev-plan T1; Design §5.9 D7 |
| CG-2 | **blocking** | task-status | Tasks T8, T9, T10, T11 are committed to git but remain `ready` in entity state. Tasks must be transitioned to `done` via `finish()` after implementation is committed. | Workflow: task lifecycle states |
| CG-3 | **blocking** | regression | `TestManifestIsCanonical/no_external_duplicates` fails: the widened rollback's `trackedPaths` in `init.go` introduces string literals `"AGENTS.md"`, `"copilot-instructions.md"`, `"stage-bindings.yaml"` that the canonical-manifest invariant requires to live only in `manifest.go`. This test passes on `main`; the regression is caused by T4 (commit `12d0c84b`). | REQ-008; manifest canonical invariant |
| CG-4 | non-blocking | ci-config | CI `test-install` job triggers on all `pull_request` events rather than scoping with `paths: [internal/kbzinit/**, .kbz/skills/**, .kbz/roles/**]` as specified in dev-plan T11. This is more conservative (over-catches) and does not block functionality. | REQ-005; AC-004; Dev-plan T11 |
| CG-5 | non-blocking | pre-existing-failures | `TestEmbeddedSkillsMatchAgentSkills`, `TestEmbeddedTaskSkillsMatchProjectSkills`, `TestEmbeddedRolesMatchProjectRoles` fail — but identically on `main`. These are pre-existing seed-sync issues unrelated to this feature. | N/A (not introduced by this feature) |

## Build & Test Results

### `go build ./...`
✅ Pass (clean build, no errors)

### `go test ./...`
❌ Fail — 4 failures in `internal/kbzinit`:

| Test | Status | Caused by this feature? |
|------|--------|------------------------|
| `TestManifestIsCanonical/no_external_duplicates` | FAIL | **Yes** — CG-3 |
| `TestEmbeddedSkillsMatchAgentSkills` | FAIL | No (pre-existing on main) |
| `TestEmbeddedTaskSkillsMatchProjectSkills` | FAIL | No (pre-existing on main) |
| `TestEmbeddedRolesMatchProjectRoles` | FAIL | No (pre-existing on main) |

All other packages pass.

## Git Log Summary

10 commits present (all T2–T11). T1 has no commit (see CG-1).

```
327b176a feat(TASK-01KR7DDD8VD99): add GitHub Actions CI job for install e2e tests
651a3126 feat(TASK-01KR7DDD5JDD2): implement kbz doctor integration tests
3f692951 feat(TASK-01KR7DDB3HG8H): implement flag-behaviour e2e tests
e723f47f feat(TASK-01KR7DC30AE4R): implement core e2e test cases
8558e610 feat(TASK-01KR7DC3093PX): build e2e test harness helpers
87f6676f feat(TASK-01KR7DC308AW1): add make test-install target and e2e build-tag scaffold
5e8ab483 feat(TASK-01KR7DC3063A9): implement kbz doctor subcommand
12d0c84b feat(TASK-01KR7DC3051HD): widen rollback in runNewProject
d460ee1e feat(TASK-01KR7DC304P3C): split --skip-mcp and add --skip-zed
8439ed3d feat(TASK-01KR7DC302CRA): rename --skip-agents-md to --skip-instructions
```

## Acceptance Criteria Traceability

| AC | Status | Evidence |
|----|--------|----------|
| AC-001 (e2e suite passes) | ⚠️ Not verifiable | `go test -tags=e2e` blocked by CG-3 (untagged `go test` already fails) |
| AC-002 (idempotency) | ✅ | `TestE2E_ReInstallIsIdempotent` in e2e_test.go L229 |
| AC-003 (partial recovery) | ✅ | `TestE2E_PartialInstallRecovery` in e2e_test.go L394 |
| AC-004 (CI job blocks merge) | ⚠️ | Job exists but per CG-4: paths filter missing; blocking status depends on repo settings, not verifiable in code |
| AC-005 (doctor missing artifact → exit 1) | ✅ | `TestDoctor_MissingRequiredFile` in doctor_test.go L52; `TestE2E_Doctor_MissingRequiredArtifact` in e2e_test.go L539 |
| AC-006 (doctor ghost file → exit 0 + warning) | ✅ | `TestDoctor_GhostFile` in doctor_test.go L107; `TestE2E_Doctor_GhostFile` in e2e_test.go L570 |
| AC-007 (rollback removes all) | ✅ | `TestRunNewProject_Rollback_Widened` in init_test.go |
| AC-008 (--skip-instructions suppresses all) | ✅ | `TestE2E_SkipInstructions_AlsoSkipsCopilot` in e2e_test.go L448 |
| AC-009 (--skip-agents-md deprecated) | ✅ | `TestE2E_SkipAgentsMD_EmitsDeprecationWarning` in e2e_test.go L501 |
| AC-010 (--skip-mcp narrow, .zed present) | ✅ | `TestE2E_SkipMCP_DoesNotCreateMCPConfig` in e2e_test.go L472 |
| AC-011 (--skip-mcp scope warning) | ✅ | `TestE2E_SkipMCPWithoutSkipZed_EmitsScopeWarning` in e2e_test.go L521 |

## Required Actions

1. **Fix CG-3 (blocking):** Refactor `trackedPaths` in `init.go` to avoid string-literal artifact names that duplicate the Manifest. Use constants from `manifest.go` or reference the Manifest struct fields instead of hardcoded strings.
2. **Resolve CG-1 (blocking):** Either implement T1 (the D7 guard) or transition it to `not-planned`/`cancelled` with a comment citing the F1 merge SHA that already addressed it.
3. **Fix CG-2 (blocking):** Call `finish()` for tasks T8, T9, T10, T11 to transition them from `ready` to `done`.
4. **Address CG-4 (non-blocking):** Optionally add `paths` filter to the CI job to match the dev-plan spec. Not required for conformance — the broader trigger is functionally fine.

## Evidence

- Build: `go build ./...` (worktree, clean)
- Test: `go test ./...` (worktree, 4 failures in kbzinit, 1 regression)
- Git log: 10 commits, T1 missing
- Entity status: T1=queued, T2–T7=done, T8–T11=ready
- Main baseline: `TestManifestIsCanonical` passes on main; skill/role consistency tests fail identically on main
