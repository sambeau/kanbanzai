# P62-F2 Specification — Install Registry &amp; Marker Unification

| Field  | Value                          |
|--------|--------------------------------|
| Feature | FEAT-01KR7BKXJGEPD (install-registry) |
| Batch  | B64-install-skill-quality |
| Plan   | P62-install-skill-quality-remediation |
| Status | draft |
| Author | spec-author |
| Design | `work/P62-install-skill-quality-remediation/P62-design-install-skill-quality.md` (approved) §5.1, §5.2, §5.5 |

## Problem Statement

The install pipeline hard-codes three separate canonical lists (skill
names in `skills.go`, task-skill names in `task_skills.go`, role files
inferred from `embed.FS` walking in `roles.go`). Each list has its own
ad-hoc managed-marker format and its own one-off comparator
(`readMarkdownManagedVersion`, `extractVersion`, `extractYAMLVersion`,
`extractStageBindingsVersion`). This causes three observable defects:

1. `extractStageBindingsVersion` rejects any non-integer value and
   returns "", so `stage-bindings.yaml` is rewritten on every re-init
   for release builds (the file stores the binary semver).
2. The "version unparseable" path silently overwrites instead of
   warning the user.
3. Drift between the install lists and the documentation (e.g. the
   AGENTS.md "Workflow Skills" table lists 6 skills but the installer
   ships 9) goes undetected.

There is no single place a future engineer can look to see "what does
`kbz init` actually deliver". F3 (runtime surfaces) is blocked on this
because it needs to add four new artifact kinds and the current shape
would force four new install functions and four new comparators.

**Scope:** Refactor the install pipeline around a single canonical
`Manifest` and a single `MarkerSpec` comparator. Add a build-time
corpus check that asserts manifest consistency.
**Out of scope:** Adding new artifact kinds (F3), e2e tests (F4),
content-quality LLM evaluation.

## Requirements

### Functional Requirements

- **REQ-001:** A new exported type `Artifact` must declare every file
  the installer ships, with fields: `Name`, `Kind`, `EmbedPath`,
  `InstallPath`, `Required` (bool), and `Marker` (`MarkerSpec`).
- **REQ-002:** A package-level `Manifest []Artifact` must enumerate
  every workflow skill, task skill, role, `AGENTS.md`,
  `copilot-instructions.md`, and `stage-bindings.yaml` currently
  installed. It must be the **only** place these lists exist.
- **REQ-003:** A single `MarkerSpec`-driven comparator
  (`compareManaged(existing []byte, spec MarkerSpec) Decision`) must
  replace `readMarkdownManagedVersion`, `extractVersion`,
  `extractYAMLVersion`, and `extractStageBindingsVersion`. It must
  return one of: `Create`, `Overwrite`, `NoOp`, `WarnSkip`.
- **REQ-004:** `MarkerSpec.VersionKind` must support `IntCounter` and
  `Semver`. Comparison rules:
  - absent file → `Create`
  - present, no marker → `WarnSkip`
  - present, marker, version unparseable → `WarnSkip` (currently silently rewrites)
  - present, marker, older → `Overwrite`
  - present, marker, equal → `NoOp`
  - present, marker, newer → `NoOp`
- **REQ-005:** `installSkills`, `installTaskSkills`, `installRoles`,
  `writeAgentsMD`, `writeCopilotInstructions`, and
  `installStageBindings` must each become a thin wrapper that filters
  the Manifest by `Kind` and delegates to a single
  `installArtifact(a Artifact)` function.
- **REQ-006:** A `TestEmbeddedCorpus` test must run as part of
  `go test ./internal/kbzinit` (no build tag) and assert:
  - Every `Manifest` entry's `EmbedPath` resolves in the embed FS.
  - Every embedded skill / role contains the marker its `MarkerSpec`
    requires and a parseable version line.
  - Every role referenced by `roles:` keys in the embedded
    `stage-bindings.yaml` is present in the Manifest as a role
    artifact.
  - Every workflow skill named in `AGENTS.md` and
    `copilot-instructions.md` content is present in the Manifest as a
    workflow-skill artifact (catches the AGENTS.md drift defect).

### Non-Functional Requirements

- **REQ-NF-001:** The refactor must not change observable install
  behaviour for any scenario already covered by existing
  `agents_md_test.go` or `init_test.go` tests, except where a defect
  fix changes a "silently rewrite" path to a "warn and skip" path
  (this is intentional — see AC-005).
- **REQ-NF-002:** Stage-bindings install on a release build must be a
  no-op when the on-disk version equals the binary version (regression
  guard for the always-rewrite defect).

## Constraints

- Must keep the existing exported function names
  (`installSkills`, `writeAgentsMD`, etc.) so external test files do
  not break — they become wrappers around `installArtifact`.
- Must not require consumers to re-run `kbz init` after upgrade — the
  on-disk file format is unchanged.
- The `MarkerSpec` for `stage-bindings.yaml` must be `Semver` so the
  comparator stops returning "" for the binary semver.

## Acceptance Criteria

- **AC-001 (REQ-001, REQ-002):** Given the codebase, when grep-ing
  for the literal strings `"kanbanzai-getting-started"`,
  `"audit-codebase"`, and `"architect.yaml"`, then each appears
  exactly once outside of test fixtures (in the Manifest declaration).
- **AC-002 (REQ-003, REQ-004):** Given a comparator unit test with a
  table of `(existing bytes, spec, expected Decision)` rows covering
  all six rules in REQ-004, when the test runs, then every row passes.
- **AC-003 (REQ-004 newer):** Given an `AGENTS.md` with marker
  `<!-- kanbanzai-managed: v999 -->`, when `kbz init` runs, then the
  file is preserved verbatim and stdout is silent about it.
- **AC-004 (REQ-004 unparseable):** Given a `.kbz/roles/architect.yaml`
  with `version: "garbage"` and a managed marker, when `kbz init`
  runs, then the file is preserved and stdout contains a warning
  naming the file.
- **AC-005 (REQ-NF-002):** Given a release binary built with
  `-ldflags '-X main.version=v9.9.9'`, when `kbz init` is run twice
  in the same scratch repo, then the second run does **not** print
  `Updated .kbz/stage-bindings.yaml`.
- **AC-006 (REQ-006):** Given a deliberately-corrupted manifest
  fixture (missing skill or marker), when `TestEmbeddedCorpus` runs,
  then the test fails with an error message naming the missing item.
- **AC-007 (REQ-006 drift):** Given an `AGENTS.md` content string
  that lists a skill not in the Manifest, when `TestEmbeddedCorpus`
  runs, then it fails with an error naming the missing skill.
- **AC-008 (REQ-006 dangling role):** Given an embedded
  `stage-bindings.yaml` referencing a role not in the Manifest, when
  `TestEmbeddedCorpus` runs, then it fails with an error naming the
  role.

## Verification Plan

| Criterion | Method | Description |
|---|---|---|
| AC-001 | Test | Automated: `TestManifestIsCanonical` greps source tree |
| AC-002 | Test | Automated: table-driven unit test for `compareManaged` |
| AC-003 | Test | Automated: e2e test pre-writes v999 AGENTS.md |
| AC-004 | Test | Automated: e2e test pre-writes garbage role version |
| AC-005 | Test | Automated: e2e test runs init twice with release binary |
| AC-006 | Test | Automated: `TestEmbeddedCorpus` with synthetic broken fixture |
| AC-007 | Test | Automated: drift fixture asserts test failure |
| AC-008 | Test | Automated: dangling-role fixture asserts test failure |
