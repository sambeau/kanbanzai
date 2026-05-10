# P62-F2 Dev-Plan — Install Registry & Marker Unification

| Field  | Value                                        |
|--------|----------------------------------------------|
| Date   | 2026-05-09                                   |
| Status | Draft                                        |
| Author | architect (senior software architect)        |
| Feature | FEAT-01KR7BKXJGEPD (B64-F2, install-registry) |
| Batch  | B64-install-skill-quality                    |
| Spec   | `work/P62-install-skill-quality-remediation/P62-F2-spec-install-registry.md` |

---

## Scope

This plan implements all requirements defined in
`work/P62-install-skill-quality-remediation/P62-F2-spec-install-registry.md`
(FEAT-01KR7BKXJGEPD). It covers Tasks T1–T9 below.

**In scope:**
- Introducing `Artifact`, `ArtifactKind`, `MarkerSpec`, `VersionKind`, and `Decision`
  types in `internal/kbzinit/registry.go` (design §5.1, §5.2).
- Building the canonical `Manifest []Artifact` that replaces the three separate
  hard-coded lists in `skills.go`, `task_skills.go`, and `roles.go`.
- Implementing a single `compareManaged(existing []byte, spec MarkerSpec) Decision`
  comparator that replaces the four ad-hoc version-extraction functions.
- Refactoring the six install wrapper functions to filter `Manifest` by `Kind` and
  delegate to a single `installArtifact(a Artifact)` function.
- Fixing the stage-bindings always-rewrite defect and the silent-rewrite-on-unparseable-version risk.
- Adding `TestEmbeddedCorpus` and `TestManifestIsCanonical`.
- Integration tests (calling Go functions directly with a temp directory) that verify
  AC-003, AC-004, and AC-005 — **not** the F4 e2e binary harness.
- Verifying that pre-existing `agents_md_test.go` and `init_test.go` pass without modification (REQ-NF-001).

**Out of scope:**
- Adding new artifact kinds (F3 scope).
- End-to-end binary harness tests (F4 scope).
- Content-quality LLM evaluation.
- `kbz doctor` / `kbz init --check` (design §5.7, deferred).

### Design notes carried forward from validation

**`Artifact` struct fields (REQ-001 + design §5.1):** The struct must declare both
`Required bool` and `Optional bool` as separate fields. The spec's REQ-001 description
omits `Optional bool`, but the design's canonical definition (§5.1) includes it. Both
fields must be present.

**`Decision` type:** Implement as a typed string constant (or typed `int` — either is
acceptable) with exactly four exported values: `Create`, `Overwrite`, `NoOp`, `WarnSkip`.
A typed constant avoids primitive obsession and makes `switch` exhaustiveness checkable.

**Integration tests vs. e2e tests (AC-003, AC-004, AC-005):** The spec labels
these as "e2e tests" in its Verification Plan table, but the Out-of-Scope section
explicitly excludes e2e tests (deferred to F4). These acceptance criteria are verified
using **integration tests that call Go install functions directly** from a `*_test.go`
file in `internal/kbzinit/`, using `t.TempDir()` as the target directory. No binary
build, no subprocess execution. Label these tests "integration tests" in all test file
comments.

---

## Task Breakdown

### Task 1: Define types and constants in `registry.go`

- **Description:** Create `internal/kbzinit/registry.go` declaring the foundational types
  that every other task depends on. No logic is implemented here — only type definitions
  and constants.
  - `ArtifactKind` — typed string constant: `WorkflowSkill`, `TaskSkill`, `Role`,
    `AgentsMd`, `CopilotInstructions`, `StageBindings`.
  - `VersionKind` — typed string constant: `IntCounter`, `Semver`.
  - `Decision` — typed string constant: `Create`, `Overwrite`, `NoOp`, `WarnSkip`.
  - `MarkerSpec` struct: `Comment string`, `VersionKind VersionKind`, `CurrentValue string`.
  - `Artifact` struct: `Name string`, `Kind ArtifactKind`, `EmbedPath string`,
    `InstallPath string`, `Required bool`, `Optional bool`, `Marker MarkerSpec`.
    (Both `Required` and `Optional` are required per design §5.1.)
- **Deliverable:** `internal/kbzinit/registry.go` with type definitions that compile cleanly.
- **Depends on:** None (independent).
- **Effort:** Small.
- **Spec requirement:** REQ-001 (Artifact struct), REQ-003 (Decision type), REQ-004 (VersionKind).

---

### Task 2: Build the canonical `Manifest` slice

- **Description:** Populate `var Manifest = []Artifact{...}` in `registry.go` (or a
  separate `manifest.go` in the same package) enumerating every artifact currently
  installed by `kbz init`:
  - All workflow skills (currently in `skills.go` / `skillNames`).
  - All task skills (currently in `task_skills.go`).
  - All role YAML files (currently inferred from embed.FS walk in `roles.go`).
  - `AGENTS.md`, `copilot-instructions.md`, `stage-bindings.yaml`.
  Each entry must have correct `EmbedPath`, `InstallPath`, `Kind`, `Required`/`Optional` flags,
  and a fully populated `MarkerSpec` (with the correct `VersionKind` per artifact type).
  `stage-bindings.yaml` **must** use `Semver` as its `VersionKind` (per spec constraint,
  to fix the always-rewrite defect). Skill and role files use `IntCounter`.
- **Deliverable:** `var Manifest []Artifact` with all current artifacts enumerated. The three
  separate hard-coded lists in `skills.go`, `task_skills.go`, and `roles.go` become dead code
  stubs pointing at the Manifest (they are removed in T4).
- **Depends on:** T1 (Artifact type required).
- **Effort:** Medium (requires careful enumeration of all embedded files).
- **Spec requirement:** REQ-002 (single canonical list).

---

### Task 3: Implement `compareManaged`

- **Description:** Implement `compareManaged(existing []byte, spec MarkerSpec) Decision`
  in a new file `internal/kbzinit/compare.go`. It must implement all six comparison
  rules from REQ-004:
  1. File absent → `Create`
  2. Present, no marker line matching `spec.Comment` → `WarnSkip`
  3. Present, marker found, version string unparseable for `spec.VersionKind` → `WarnSkip`
     (currently the code silently rewrites — this is the defect fix)
  4. Present, marker found, version older → `Overwrite`
  5. Present, marker found, version equal → `NoOp`
  6. Present, marker found, version newer → `NoOp`
  
  `VersionKind` comparison logic: for `IntCounter`, parse both values as integers; for
  `Semver`, use semantic version comparison (the `golang.org/x/mod/semver` package
  or a minimal manual comparison is acceptable since these are `vMAJOR.MINOR.PATCH` strings).
  The function must be pure (no I/O side effects) to enable easy unit testing.
- **Deliverable:** `internal/kbzinit/compare.go` with `compareManaged` implementing all
  six rules.
- **Depends on:** T1 (Decision, MarkerSpec, VersionKind types required).
- **Effort:** Small.
- **Spec requirement:** REQ-003, REQ-004.

---

### Task 4: Refactor install functions as thin Manifest wrappers

- **Description:** Implement `installArtifact(a Artifact, targetDir string)` in
  `internal/kbzinit/install.go` (or alongside the existing install logic). This function:
  1. Reads the on-disk file at `filepath.Join(targetDir, a.InstallPath)`.
  2. Calls `compareManaged(existingBytes, a.Marker)`.
  3. Acts on the returned `Decision`:
     - `Create` / `Overwrite` → write the embedded bytes to disk.
     - `NoOp` → return without writing.
     - `WarnSkip` → print a warning to stdout naming the file, return without writing.
  
  Then refactor each of the six install wrappers to filter `Manifest` by `Kind` and call
  `installArtifact` for each matching entry, removing the old hard-coded logic:
  - `installSkills` → filter `WorkflowSkill`
  - `installTaskSkills` → filter `TaskSkill`
  - `installRoles` → filter `Role`
  - `writeAgentsMD` → filter `AgentsMd`
  - `writeCopilotInstructions` → filter `CopilotInstructions`
  - `installStageBindings` → filter `StageBindings`
  
  Remove the four superseded functions: `readMarkdownManagedVersion`, `extractVersion`,
  `extractYAMLVersion`, `extractStageBindingsVersion`.
  
  **Defect fix — always-rewrite:** Because `stage-bindings.yaml` now uses `Semver`
  `VersionKind` in its `MarkerSpec`, `compareManaged` will parse the binary version
  correctly and return `NoOp` when the on-disk version equals the binary version.
  
  **Defect fix — silent rewrite:** The `WarnSkip` branch prints a warning instead of
  silently overwriting, fixing the unparseable-version path.
- **Deliverable:** Refactored `install.go` with `installArtifact` plus six thin wrappers.
  All four superseded version-extraction functions removed. `go build ./internal/kbzinit/...`
  passes.
- **Depends on:** T2 (Manifest required), T3 (compareManaged required).
- **Effort:** Large (touches multiple install functions, must preserve existing external signatures).
- **Spec requirement:** REQ-003, REQ-004, REQ-005, REQ-NF-002, Constraints (preserve exported names).

---

### Task 5: `TestManifestIsCanonical` unit test

- **Description:** Add `TestManifestIsCanonical` in `internal/kbzinit/manifest_test.go`.
  This test uses `go/packages` or simple `grep`-style source scanning to assert that each
  of the key artifact name strings (e.g., `"kanbanzai-getting-started"`, `"audit-codebase"`,
  `"architect.yaml"`) appears exactly **once** outside of test fixture files — confirming that
  the Manifest is the sole declaration point and the old hard-coded slices have been removed.
  The test must not require a binary build or subprocess execution.
- **Deliverable:** `internal/kbzinit/manifest_test.go` with `TestManifestIsCanonical` that passes
  after T4 and fails if any artifact name is re-declared outside the Manifest.
- **Depends on:** T2 (Manifest must exist with the names it asserts).
- **Effort:** Small.
- **Spec requirement:** REQ-001, REQ-002, AC-001.

---

### Task 6: `compareManaged` table-driven unit tests

- **Description:** Add `TestCompareManaged` in `internal/kbzinit/compare_test.go` as a
  table-driven test. The table must cover all six decision rules from REQ-004 with at least
  one row per rule, for both `IntCounter` and `Semver` `VersionKind`:
  - absent file bytes (`nil`) → `Create`
  - bytes with no marker → `WarnSkip`
  - bytes with marker, version unparseable → `WarnSkip`
  - bytes with marker, older version → `Overwrite`
  - bytes with marker, equal version → `NoOp`
  - bytes with marker, newer version → `NoOp`
- **Deliverable:** `internal/kbzinit/compare_test.go` with all rows passing.
- **Depends on:** T3 (compareManaged must exist).
- **Effort:** Small.
- **Spec requirement:** REQ-003, REQ-004, AC-002.

---

### Task 7: Integration tests for AC-003, AC-004, AC-005

- **Description:** Add integration tests in `internal/kbzinit/install_integration_test.go`
  (or `install_test.go`). These tests call the Go install functions **directly** with a
  `t.TempDir()` target directory. They do **not** build or invoke the `kbz` binary.
  Do **not** use the F4 e2e binary harness.

  Three tests are required:

  **AC-003 (`TestIntegration_NewerMarker_NoOp`):**
  - Pre-write an `AGENTS.md` to the temp dir containing a managed marker with version `v999`.
  - Call `writeAgentsMD(targetDir)`.
  - Assert the file content is unchanged (preserved verbatim).
  - Assert no "Updated" line is printed to stdout.

  **AC-004 (`TestIntegration_UnparseableVersion_WarnSkip`):**
  - Pre-write a `.kbz/roles/architect.yaml` to the temp dir with a managed marker and
    `version: "garbage"`.
  - Call `installRoles(targetDir)` (or `installArtifact` for that specific artifact).
  - Assert the file content is unchanged.
  - Assert stdout contains a warning that names the file.

  **AC-005 (`TestIntegration_StageBindings_NoDoubleWrite`):**
  - Call `installStageBindings(targetDir)` once (initial install).
  - Call `installStageBindings(targetDir)` a second time.
  - Assert the second call does **not** print "Updated .kbz/stage-bindings.yaml".
  - (This validates REQ-NF-002 without requiring a release binary build.)

  Capture stdout in tests using `os.Pipe()` or `bytes.Buffer` injection as appropriate.
- **Deliverable:** `internal/kbzinit/install_integration_test.go` with all three tests passing.
- **Depends on:** T4 (refactored install functions required).
- **Effort:** Medium.
- **Spec requirement:** REQ-004, REQ-NF-002, AC-003, AC-004, AC-005.

---

### Task 8: `TestEmbeddedCorpus`

- **Description:** Add `TestEmbeddedCorpus` in `internal/kbzinit/corpus_test.go`
  (no build tag — runs on every `go test ./internal/kbzinit`). The test performs four
  structural assertions against the live embedded corpus and Manifest:

  1. **Embed FS coverage:** For every `Manifest` entry, assert `a.EmbedPath` resolves
     in the embed FS without error.
  2. **Marker and version parseability:** For every embedded skill and role, assert the
     source bytes contain the marker line specified by `a.Marker.Comment` and that the
     version value is parseable for `a.Marker.VersionKind`.
  3. **AGENTS.md / copilot-instructions.md drift check:** Parse the embedded content of
     both files to extract the workflow-skill names they reference. Assert every such name
     appears in the Manifest as a `WorkflowSkill` artifact. (Catches the "AGENTS.md lists
     6 skills but installer ships 9" drift defect.)
  4. **Dangling role references:** Parse the embedded `stage-bindings.yaml` and collect
     every role name referenced under `roles:` keys. Assert each is present in the Manifest
     as a `Role` artifact.

  Also add three negative fixture sub-tests to verify the corpus check fails appropriately:
  - **AC-006 (`TestEmbeddedCorpus_MissingMarker`):** Use a synthetic in-memory fixture
    with a Manifest entry whose embedded bytes lack the expected marker. Assert the test
    fails with an error naming the item.
  - **AC-007 (`TestEmbeddedCorpus_AgentsMDDrift`):** Inject a synthetic AGENTS.md content
    string referencing a skill not in the Manifest. Assert the test fails naming the missing skill.
  - **AC-008 (`TestEmbeddedCorpus_DanglingRole`):** Inject a synthetic stage-bindings.yaml
    referencing a role not in the Manifest. Assert the test fails naming the role.

  The negative fixture sub-tests must use table-driven helpers rather than modifying the
  live embed FS.
- **Deliverable:** `internal/kbzinit/corpus_test.go` with `TestEmbeddedCorpus` and three
  negative fixture sub-tests — all passing, including the intentional failures in the
  negative cases (verified via `testing.T.Run` + `t.Fail()`/`t.Error()` assertions).
- **Depends on:** T1 (types), T2 (Manifest with embed paths).
- **Effort:** Large.
- **Spec requirement:** REQ-006, AC-006, AC-007, AC-008.

---

### Task 9: Verify pre-existing test suites pass unmodified (REQ-NF-001)

- **Description:** After the refactor in T4 is complete, run the full `go test ./internal/kbzinit`
  suite and confirm that `agents_md_test.go` and `init_test.go` pass **without any modification**.
  These tests encode the pre-refactor install behaviour contract. Any failure here indicates
  an unintended observable behaviour change introduced by the refactor.

  If a failure occurs on a "silently rewrite" path that is now intentionally a "warn and skip"
  path (i.e., the defect fix for unparseable versions), document it explicitly — this is an
  **expected** behavioural change under REQ-NF-001's exception clause and the test must be
  updated to assert the new warn-and-skip output rather than silent rewrite. All other failures
  are regressions and must be fixed before the task is marked done.

  This task has no code deliverable — it is a verification checkpoint.
- **Deliverable:** All pre-existing tests in `agents_md_test.go` and `init_test.go` pass
  (with the narrow exception above documented in a code comment if applicable). Screenshot
  or log of `go test -v ./internal/kbzinit/... | grep -E "(PASS|FAIL)"` included in the
  task completion summary.
- **Depends on:** T4 (refactored install pipeline must exist before verification).
- **Effort:** Small.
- **Spec requirement:** REQ-NF-001.

---

## Dependency Graph

```
T1: Define types and constants (no dependencies)
  └─► T2: Build Manifest (depends on T1)
  └─► T3: Implement compareManaged (depends on T1)
         └─► T4: Refactor install functions (depends on T2 + T3)
                └─► T7: Integration tests AC-003/004/005 (depends on T4)
                └─► T9: Pre-existing test verification (depends on T4)
  └─► T2
        └─► T5: TestManifestIsCanonical (depends on T2)
        └─► T8: TestEmbeddedCorpus (depends on T1 + T2)
  └─► T3
        └─► T6: compareManaged unit tests (depends on T3)
```

**Structured view:**

```
T1  (no deps)
T2  → depends on T1
T3  → depends on T1
T4  → depends on T2, T3
T5  → depends on T2
T6  → depends on T3
T7  → depends on T4
T8  → depends on T1, T2
T9  → depends on T4
```

**Parallel groups:**
- Group A: [T2, T3] — both depend on T1 only, no mutual dependency; run in parallel.
- Group B: [T5, T6, T8] — after their respective prereqs (T2, T3, T1+T2); T5 and T8 both
  need T2, so they must wait for T2 but can then run together; T6 needs T3 only.
  In practice: once T2 and T3 are complete, T5, T6, and T8 can all run in parallel.
- Group C: [T7, T9] — both depend on T4 only; run in parallel.

**Critical path:** T1 → T2 → T4 → T7 (or T9)
(T2 and T3 take equal conceptual effort so the two chains through T1 are co-critical;
T4 cannot start until both are done, making T4 the primary bottleneck.)

---

## Risk Assessment

### Risk: Enumeration gaps in the Manifest (T2)

- **Probability:** Medium — the three existing install lists were written independently
  and are likely not perfectly consistent with each other.
- **Impact:** High — a missing artifact in the Manifest means the installer silently stops
  delivering that file on fresh installs.
- **Mitigation:** T8 (`TestEmbeddedCorpus`) embeds the detection at compile time.
  During T2, cross-reference the embed FS walk output against the declared Manifest as
  a manual audit step. Run `go test ./internal/kbzinit -run TestEmbeddedCorpus` against
  a draft Manifest before proceeding to T4.
- **Affected tasks:** T2, T4, T8.

---

### Risk: `compareManaged` version-extraction edge cases (T3)

- **Probability:** Medium — semver strings vary in format (e.g., `v1.2.3`, `1.2.3`,
  pre-release suffixes) and existing on-disk files may have unexpected marker whitespace.
- **Impact:** Medium — wrong comparison returns cause silent overwrites or spurious
  WarnSkips; users may lose customisations.
- **Mitigation:** T6 provides table-driven coverage of all six rules for both VersionKind
  values. Include edge-case rows for bare integer strings, `v`-prefixed semvers, and
  whitespace variants. Implement T3 and T6 together (test-driven).
- **Affected tasks:** T3, T6, T7.

---

### Risk: Existing test breakage from warn-and-skip behaviour change (T9)

- **Probability:** Low-to-medium — `init_test.go` may have tests that expect the old
  silent-rewrite path for unparseable versions.
- **Impact:** Low — this is an expected intentional change per REQ-NF-001's exception
  clause; the fix is updating one or two test assertions, not a design change.
- **Mitigation:** Identify the affected test cases during T4 implementation (before T9).
  Update only those test assertions that specifically encode the "silently rewrite on
  unparseable version" behaviour, documenting the change with a code comment referencing
  REQ-NF-001.
- **Affected tasks:** T4, T9.

---

### Risk: `installArtifact` stdout capture complicates integration testing (T7)

- **Probability:** Medium — the current install functions may print to `os.Stdout` directly
  rather than accepting a writer, making it hard to assert warning messages in tests.
- **Impact:** Medium — AC-004 requires asserting a warning is printed; AC-005 requires
  asserting no "Updated" line is printed.
- **Mitigation:** Design `installArtifact` to accept an `io.Writer` for output (defaulting
  to `os.Stdout` in the production call sites). This is a small upfront API decision that
  eliminates a testing pain point. Alternatively, use `os.Pipe()` redirection in tests.
  Decide during T4 and implement consistently.
- **Affected tasks:** T4, T7.

---

### Risk: F3 / F4 contract breakage (design §5.4, §5.6)

- **Probability:** Low — F3 and F4 are out of scope but will build on the Manifest.
- **Impact:** Medium — if the Manifest schema or `ArtifactKind` constants are too narrow,
  F3 will require a breaking change.
- **Mitigation:** Follow design §5.1 exactly: include `Optional bool`, use the full
  `ArtifactKind` set from the design (even if some kinds have no Manifest entries yet),
  and document the extension point. Do not add F3 artifact entries in this feature.
- **Affected tasks:** T1, T2.

---

## Verification Approach

**Note on AC-003 / AC-004 / AC-005 method:** The spec's Verification Plan labels these
as "e2e tests", but the feature's Out-of-Scope section explicitly excludes e2e tests (F4).
This plan resolves the conflict by implementing these as **integration tests** that call
Go install functions directly from `*_test.go` files with a `t.TempDir()`. No binary build
or subprocess invocation is required.

| Acceptance Criterion | Verification Method | Producing Task |
|----------------------|---------------------|----------------|
| AC-001 (REQ-001, REQ-002): each key artifact name appears exactly once outside test fixtures | Unit test: `TestManifestIsCanonical` greps source tree | T5 |
| AC-002 (REQ-003, REQ-004): table of `(existing, spec, expected Decision)` rows all pass | Unit test: `TestCompareManaged` table-driven | T6 |
| AC-003 (REQ-004 newer): `AGENTS.md` with v999 marker is preserved, no stdout output | Integration test: `TestIntegration_NewerMarker_NoOp` (temp dir, Go function call) | T7 |
| AC-004 (REQ-004 unparseable): role file with `"garbage"` version preserved + warning printed | Integration test: `TestIntegration_UnparseableVersion_WarnSkip` (temp dir, Go function call) | T7 |
| AC-005 (REQ-NF-002): second `installStageBindings` call does not print "Updated …" | Integration test: `TestIntegration_StageBindings_NoDoubleWrite` (temp dir, Go function call) | T7 |
| AC-006 (REQ-006): `TestEmbeddedCorpus` fails when Manifest entry lacks expected marker | Negative fixture sub-test: `TestEmbeddedCorpus_MissingMarker` | T8 |
| AC-007 (REQ-006 drift): `TestEmbeddedCorpus` fails when AGENTS.md lists a skill not in Manifest | Negative fixture sub-test: `TestEmbeddedCorpus_AgentsMDDrift` | T8 |
| AC-008 (REQ-006 dangling role): `TestEmbeddedCorpus` fails when stage-bindings references unlisted role | Negative fixture sub-test: `TestEmbeddedCorpus_DanglingRole` | T8 |
| REQ-NF-001: no unintended behaviour change for pre-existing covered scenarios | Run `agents_md_test.go` + `init_test.go` without modification; all pass | T9 |
| REQ-NF-002: stage-bindings install is NoOp when on-disk version = binary version | Covered by AC-005 (integration test) and `compareManaged` unit test (equal version row in T6) | T6, T7 |
