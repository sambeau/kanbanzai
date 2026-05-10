# P62-F4 Specification — Install E2E Tests &amp; `kbz doctor`

| Field  | Value                          |
|--------|--------------------------------|
| Feature | FEAT-01KR7BKXPXFSK (install-tests-doctor) |
| Batch  | B64-install-skill-quality |
| Plan   | P62-install-skill-quality-remediation |
| Status | draft |
| Author | spec-author |
| Design | `work/P62-install-skill-quality-remediation/P62-design-install-skill-quality.md` (approved) §5.6, §5.7, §5.8, §5.9 |

## Problem Statement

The install pipeline has unit tests for individual writers
(`TestWriteAgentsMD_*`, etc.) but no end-to-end coverage that builds
the binary and runs `kbz init` in a scratch repo. As a result, the
exact failure modes the P59 audit surfaced — re-init aborting, `work/`
not being created, `--skip-*` flags affecting unintended files,
partial installs leaking orphan files — were invisible until manually
audited.

Three follow-on UX defects from the audit are also addressed here
because they share test surface:

- D5: `--skip-agents-md` silently disables
  `.github/copilot-instructions.md`; `--skip-mcp` silently disables
  Zed config.
- D6: A failed `runNewProject` only rolls back `.kbz/`, leaving
  orphan `.agents/`, `.github/`, `AGENTS.md`, and `.mcp.json`.
- D8: `work/` is not created on first init in a repo with prior
  commits.

A consumer-facing `kbz doctor` command lets users (and our CI) check
an installed project for drift without re-running `init`.

**Scope:** End-to-end install tests, `kbz doctor` subcommand, widened
rollback, two flag renames.
**Out of scope:** Content-quality LLM evaluation; new install
behaviours beyond what F1–F3 already deliver.

## Requirements

### Functional Requirements

- **REQ-001:** A new file `internal/kbzinit/e2e_test.go` must contain
  an end-to-end test suite. The suite must build the `kbz` binary
  once per `go test` invocation (using `sync.Once` and `t.TempDir()`)
  and reuse it across tests.
- **REQ-002:** Each e2e test must create a fresh git repo in
  `t.TempDir()`, run the binary as a subprocess with controlled
  flags, and assert on the exit code, the captured stdout/stderr,
  and the resulting filesystem.
- **REQ-003:** The suite must include at minimum these tests:
  - `TestE2E_FreshInstall_AllManifestArtifactsPresent`
  - `TestE2E_ReInstallIsIdempotent` (runs init twice, asserts both
    exit 0 and second-run stdout contains no errors)
  - `TestE2E_UpdateSkillsBumpsVersions`
  - `TestE2E_SkipInstructions_AlsoSkipsCopilot`
  - `TestE2E_SkipMCP_DoesNotCreateMCPConfig`
  - `TestE2E_SkipZed_DoesNotCreateZedSettings`
  - `TestE2E_UnmanagedAgentsMD_PreservedWithWarning`
  - `TestE2E_NewerMarker_NoOp`
  - `TestE2E_PartialInstallRecovery` (induce a write failure
    mid-init, re-run, verify clean recovery)
  - `TestE2E_WorkDirCreatedOnFirstInit_RepoWithCommits`
- **REQ-004:** The e2e suite must be gated behind a build tag `e2e`
  (or env var `KBZ_E2E=1`) so `go test ./...` stays fast for normal
  development.
- **REQ-005:** A new GitHub Actions workflow job must run the e2e
  suite on every PR that modifies any file under
  `internal/kbzinit/**` or any embedded skill / role. The job runs
  `make test-install`.
- **REQ-006:** A new `kbz doctor` subcommand must validate an
  existing install by reading the F2 Manifest and checking, for
  each Required artifact:
  - file exists at `InstallPath`,
  - marker present and parseable,
  - marker version ≥ binary version (warn, do not error, on older).
  It must additionally report any "ghost" file under `.kbz/skills/`,
  `.agents/skills/`, or `.kbz/roles/` that is not in the Manifest.
- **REQ-007:** `kbz doctor` must exit 0 when all checks pass, exit 1
  when any Required artifact is missing or unparseable. Warnings
  (older versions, ghost files) do not change the exit code but are
  printed.
- **REQ-008:** `runNewProject` must track every path it creates
  (across `.kbz/`, `.agents/`, `.github/`, `AGENTS.md`, `.mcp.json`,
  `.zed/settings.json`, `CLAUDE.md`, `OPENAI.md`, `work/`,
  `.claude/skills/`, optionally `.cursor/rules/`) and remove all of
  them on failure. The sentinel write must be the only commit point.
- **REQ-009:** `--skip-agents-md` must be renamed to
  `--skip-instructions` and now suppress every "instruction surface"
  artifact (AGENTS.md, copilot-instructions.md, CLAUDE.md, OPENAI.md,
  .claude/skills/, .cursor/rules/). The old name must remain as a
  deprecated alias for one release with a stderr warning.
- **REQ-010:** A new `--skip-zed` flag must be added. The existing
  `--skip-mcp` must continue to suppress only `.mcp.json`. Zed
  settings must be controlled exclusively by `--skip-zed`. The
  existing combined behaviour of `--skip-mcp` (suppressing both)
  must be preserved as a deprecated alias for one release with a
  stderr warning explaining the change.
- **REQ-011:** A `make test-install` target must exist that runs the
  tagged e2e tests with race detector enabled.

### Non-Functional Requirements

- **REQ-NF-001:** The full e2e suite must complete in under 60
  seconds on commodity CI hardware.
- **REQ-NF-002:** `kbz doctor` must complete in under 500 ms on a
  typical consumer install.
- **REQ-NF-003:** No e2e test may depend on network access.

## Constraints

- Must not require the build cache to be primed — the suite must
  succeed on a clean checkout in CI.
- Must not skip tests silently when `git` is missing — it must fail
  with a clear message.
- Flag deprecation warnings must go to stderr, not stdout, so they
  don't pollute scripted callers.

## Acceptance Criteria

- **AC-001 (REQ-001, REQ-002):** Given the e2e build tag is enabled,
  when `go test ./internal/kbzinit -tags=e2e -run TestE2E_` is run,
  then all listed tests pass.
- **AC-002 (REQ-003 idempotency):** Given a fresh repo, when the
  binary runs `init` twice, then both runs exit 0 and the second
  run's stdout does not contain "exists but is not managed by
  Kanbanzai".
- **AC-003 (REQ-003 partial recovery):** Given an install that
  aborts partway through (induced via a read-only directory or
  failure injection), when `kbz init` is re-run, then the install
  completes successfully and no orphan files remain outside the
  Manifest.
- **AC-004 (REQ-005):** Given a PR that modifies a file under
  `internal/kbzinit/`, when CI runs, then the `test-install` job
  executes and its result blocks merge if it fails.
- **AC-005 (REQ-006, REQ-007):** Given a Kanbanzai install that is
  missing a Required artifact, when `kbz doctor` is run, then it
  exits 1 and prints a line naming the missing artifact.
- **AC-006 (REQ-006 ghost file):** Given a `.kbz/skills/legacy/SKILL.md`
  that is not in the Manifest, when `kbz doctor` is run, then it
  exits 0 but prints a warning naming the ghost file.
- **AC-007 (REQ-008):** Given a `kbz init` that fails after writing
  `AGENTS.md` and `.agents/skills/...` (induced via a synthetic
  injection in tests), when the failure occurs, then on test exit no
  files remain at any of the install paths (the rollback removed
  them all).
- **AC-008 (REQ-009):** Given `kbz init --skip-instructions`, when
  init completes, then none of `AGENTS.md`,
  `.github/copilot-instructions.md`, `CLAUDE.md`, `OPENAI.md`,
  `.claude/skills/` exists.
- **AC-009 (REQ-009 deprecation):** Given
  `kbz init --skip-agents-md`, when init runs, then it behaves as
  `--skip-instructions` and stderr contains "deprecated".
- **AC-010 (REQ-010):** Given `kbz init --skip-mcp`, when init
  completes, then `.mcp.json` does not exist but `.zed/settings.json`
  does.
- **AC-011 (REQ-010 deprecation):** Given `kbz init --skip-mcp` with
  no `--skip-zed`, when init runs in the deprecation window, then
  stderr contains a warning that the flag's scope has narrowed and
  recommends `--skip-zed` for the previous behaviour.

## Verification Plan

| Criterion | Method | Description |
|---|---|---|
| AC-001 | Test | Automated: tagged e2e suite runs to completion |
| AC-002 | Test | Automated: e2e idempotency test |
| AC-003 | Test | Automated: e2e partial-recovery test with injected failure |
| AC-004 | Manual | One-off: open PR touching `internal/kbzinit/`, observe CI |
| AC-005 | Test | Automated: doctor unit + e2e tests with broken fixtures |
| AC-006 | Test | Automated: doctor test with ghost file fixture |
| AC-007 | Test | Automated: rollback test using injected write failure |
| AC-008 | Test | Automated: e2e test asserts file absence after `--skip-instructions` |
| AC-009 | Test | Automated: e2e test captures stderr deprecation warning |
| AC-010 | Test | Automated: e2e test asserts split-flag behaviour |
| AC-011 | Test | Automated: e2e test captures stderr scope warning |
