# P62-F4 Dev-Plan — Install E2E Tests & `kbz doctor`

| Field  | Value |
|--------|-------|
| Date   | 2026-05-09 |
| Status | Draft |
| Author | architect |
| Feature | FEAT-01KR7BKXPXFSK (B64-F4, install-tests-doctor) |
| Batch  | B64-install-skill-quality |
| Spec   | `work/P62-install-skill-quality-remediation/P62-F4-spec-install-tests-doctor.md` |

---

## Critical Dependency

**F4 depends on F2 (install-registry, `FEAT-01KR7BKXPXFT0` or equivalent) being merged before development begins.**

Specifically, F4 consumes:

- `Manifest` — the canonical `[]Artifact` slice used by `kbz doctor` to enumerate Required artifacts and their install paths.
- `MarkerSpec` — the comparator used by `kbz doctor` for version checks (marker version ≥ binary version).
- The widened rollback in T4 iterates over all Manifest artifacts, not just `.kbz/`.

F4 is independent of F3 (runtime-surfaces). F3 artifacts will appear in the Manifest by the time F4 runs, but F4 does not need F3 to land first.

---

## Scope

This plan implements the requirements defined in
`work/P62-install-skill-quality-remediation/P62-F4-spec-install-tests-doctor.md`
(FEAT-01KR7BKXPXFSK). It covers tasks T1–T11 below.

Scope includes:

- A D7 defect fix: guarding the "previous init appears incomplete" warning behind `kbzExisted == true` (§5.9).
- Flag renames: `--skip-agents-md` → `--skip-instructions` (REQ-009) and the `--skip-mcp` / `--skip-zed` split (REQ-010), each with a deprecated alias for one release.
- A widened `runNewProject` rollback that removes all Manifest artifacts on failure (REQ-008, §5.8).
- A `kbz doctor` subcommand that validates an existing install using the F2 Manifest and MarkerSpec (REQ-006, REQ-007, §5.7).
- An end-to-end test harness in `internal/kbzinit/e2e_test.go`, gated behind the `e2e` build tag (REQ-001 through REQ-004, REQ-011, §5.6).
- All ten required e2e test cases from REQ-003, plus explicit positive-path coverage for `--skip-zed`.
- A `make test-install` Makefile target and a GitHub Actions CI job (REQ-005, REQ-011).

Out of scope: content-quality LLM evaluation; new install behaviours beyond what F1–F3 deliver; monitoring or alerting.

---

## Task Breakdown

### Task T1: Apply D7 fix — guard "incomplete init" warning

- **Description:** In `internal/kbzinit/`, locate the "previous init appears incomplete" warning and wrap it in a `kbzExisted == true` guard so the warning only fires when the `.kbz/` directory already existed before this `init` invocation. This is a 1–2 line change.
- **Deliverable:** Patched `internal/kbzinit/` source; existing unit tests still pass.
- **Depends on:** None (applies directly to the existing init code, no Manifest dependency).
- **Effort:** Small.
- **Spec requirement:** Design §5.9 defect D7 (resolved by MAJOR 2 guidance in this plan).
- **Conditional note:** If F1 (init-unblock) is already merged by the time this task is dispatched, D7 may already be addressed by that merge. In that case mark T1 `not-planned` with a comment referencing the F1 merge SHA.

---

### Task T2: Rename `--skip-agents-md` → `--skip-instructions`

- **Description:** In the `kbz init` command definition, add a new `--skip-instructions` flag that suppresses all "instruction-surface" artifacts: `AGENTS.md`, `.github/copilot-instructions.md`, `CLAUDE.md`, `OPENAI.md`, `.claude/skills/`, `.cursor/rules/`. Keep `--skip-agents-md` as a deprecated alias that prints a stderr warning and then delegates to `--skip-instructions`.
- **Deliverable:** Updated flag registration and install-path logic; unit tests for the new behaviour; deprecation warning goes to stderr only.
- **Depends on:** None.
- **Effort:** Medium.
- **Spec requirement:** REQ-009, AC-008, AC-009.

---

### Task T3: Split `--skip-mcp` and add `--skip-zed`

- **Description:** Add a new `--skip-zed` flag that suppresses `.zed/settings.json` exclusively. Narrow `--skip-mcp` to suppress only `.mcp.json`. Preserve the old combined behaviour as a one-release deprecated alias: if `--skip-mcp` is passed without `--skip-zed`, emit a stderr warning explaining the flag's narrowed scope and recommending `--skip-zed` to restore the previous suppression.
- **Deliverable:** Updated flag registration and install-path logic; unit tests for the split; deprecation warning goes to stderr only.
- **Depends on:** None. (Parallelisable with T2.)
- **Effort:** Small-medium.
- **Spec requirement:** REQ-010, AC-010, AC-011.

---

### Task T4: Widen rollback in `runNewProject`

- **Description:** Refactor `runNewProject` to maintain a tracked path list (a small `[]string` threaded through the install steps, or a `t.Cleanup`-style deferred runner). Every path created during init is appended to the list. On any error, remove all tracked paths. The sentinel write remains the single commit point — i.e. success is only declared after the sentinel is written; on any earlier failure the full tracked list is deleted. The rollback must cover: `.kbz/`, `.agents/`, `.github/`, `AGENTS.md`, `.mcp.json`, `.zed/settings.json`, `CLAUDE.md`, `OPENAI.md`, `work/`, `.claude/skills/`, `.cursor/rules/` (when installed).
- **Deliverable:** Refactored `runNewProject`; existing unit tests pass; new unit test for the tracker clean-up path.
- **Depends on:** F2 merge (for Manifest-aware path enumeration). Also benefits from T2, T3 being done so that the full set of conditionally created paths is known.
- **Effort:** Medium.
- **Spec requirement:** REQ-008, AC-007, Design §5.8 (fixes D6).

---

### Task T5: Implement `kbz doctor` subcommand

- **Description:** Register a `doctor` subcommand in the CLI. Its algorithm (using F2's `Manifest` and `MarkerSpec`):
  1. Iterate over every `Artifact` in `Manifest`.
  2. For Required artifacts: check `InstallPath` exists; parse the managed marker; report missing or unparseable as an error (accumulate; exit 1 at the end if any).
  3. For all managed artifacts: compare marker version to binary version; if marker version is older, print a warning (no exit-code change).
  4. Scan `.kbz/skills/`, `.agents/skills/`, `.kbz/roles/` for files not represented in the Manifest; print each as a "ghost file" warning (exit 0).
  - Exit 0 when all Required artifacts pass; exit 1 when any Required artifact is missing or unparseable.
  - Must complete in under 500 ms on a typical consumer install (REQ-NF-002).
- **Deliverable:** `internal/kbzdoctor/` or equivalent package; wired into CLI; unit tests using fixture directories.
- **Depends on:** F2 merge (for `Manifest` and `MarkerSpec`). Parallelisable with T4.
- **Effort:** Medium-large.
- **Spec requirement:** REQ-006, REQ-007, AC-005, AC-006, REQ-NF-002, Design §5.7.

---

### Task T6: Add `make test-install` target and e2e build-tag scaffold

- **Description:** Add the `test-install` Makefile target:
  ```
  test-install:
      go test ./internal/kbzinit -tags=e2e -race -run TestE2E_ -count=1
  ```
  Create a minimal `internal/kbzinit/e2e_test.go` with the `//go:build e2e` constraint line at the top (and `// KBZ_E2E=1 env alias is honoured in TestMain` note). Add a `TestMain` that checks either the build tag or `KBZ_E2E=1` env var; if neither is set, it skips with a clear message rather than silently passing. The build tag (`e2e`) is canonical; `KBZ_E2E=1` is the alias for CI environments that cannot pass `-tags`.
- **Deliverable:** Updated `Makefile`; skeleton `e2e_test.go` that compiles under `-tags=e2e` and skips cleanly without it.
- **Depends on:** None.
- **Effort:** Small.
- **Spec requirement:** REQ-004, REQ-011.
- **Note:** REQ-004 dual gating — `e2e` build tag is the canonical gate; `KBZ_E2E=1` is a CI convenience alias only. Both are equivalent in effect; neither supersedes the other.

---

### Task T7: Build e2e test harness helpers

- **Description:** In `e2e_test.go`, implement:
  - `buildBinary(t)` — calls `go build` once per `go test` invocation via `sync.Once`, stores the binary path in `t.TempDir()`; fails with a clear message if `git` is missing.
  - `newScratchRepo(t, withCommit bool)` — creates a `t.TempDir()`, runs `git init`, optionally adds an empty commit, returns the path.
  - `runKbz(t, dir, args...)` — invokes the binary as a subprocess; captures stdout, stderr, exit code; fails fast if the binary wasn't built.
  - No network access in any helper (REQ-NF-003).
- **Deliverable:** Harness helpers with doc comments; `buildBinary` unit-tested by a trivial `TestE2E_Harness_BinaryBuilds` smoke test.
- **Depends on:** T6 (needs the build-tag scaffold in place first).
- **Effort:** Medium.
- **Spec requirement:** REQ-001, REQ-002.

---

### Task T8: Implement core e2e test cases

- **Description:** Implement the following tests (all using the T7 harness):
  - `TestE2E_FreshInstall_AllManifestArtifactsPresent` — fresh repo, `kbz init`, assert every Manifest artifact is present on disk.
  - `TestE2E_ReInstallIsIdempotent` — run init twice, assert both exit 0 and second-run stdout does not contain "exists but is not managed by Kanbanzai".
  - `TestE2E_UpdateSkillsBumpsVersions` — run init, mutate a marker version, re-run init, assert version is restored.
  - `TestE2E_WorkDirCreatedOnFirstInit_RepoWithCommits` — repo with one prior commit, run init, assert `work/` exists (D8 regression).
  - `TestE2E_UnmanagedAgentsMD_PreservedWithWarning` — pre-create an AGENTS.md without marker, run init, assert file unchanged and warning in stdout.
  - `TestE2E_NewerMarker_NoOp` — pre-create a file with a future marker version, run init, assert file not overwritten.
  - `TestE2E_PartialInstallRecovery` — induce a write failure mid-init via a read-only directory; re-run init; verify clean recovery and no orphan files outside the Manifest.
- **Deliverable:** Passing e2e tests for all seven cases under `go test ./internal/kbzinit -tags=e2e`.
- **Depends on:** T7 (harness), T2 (instructions rename logic), T3 (flag split), T4 (widened rollback, needed for recovery test), T5 (doctor, needed for manifest assertions).
- **Effort:** Large.
- **Spec requirement:** REQ-003, AC-001, AC-002, AC-003, AC-007, REQ-NF-001 (suite under 60 s).

---

### Task T9: Implement flag-behaviour e2e tests (including `--skip-zed` positive path)

- **Description:** Implement the following tests:
  - `TestE2E_SkipInstructions_AlsoSkipsCopilot` — `kbz init --skip-instructions`, assert none of `AGENTS.md`, `.github/copilot-instructions.md`, `CLAUDE.md`, `OPENAI.md`, `.claude/skills/` exist.
  - `TestE2E_SkipMCP_DoesNotCreateMCPConfig` — `kbz init --skip-mcp`, assert `.mcp.json` absent, `.zed/settings.json` present.
  - **`TestE2E_SkipZed_DoesNotCreateZedSettings`** — `kbz init --skip-zed`, assert `.zed/settings.json` does not exist in the output directory while `.mcp.json` is present. This is the explicit positive-path coverage for REQ-010 / Design §5.6.
  - `TestE2E_SkipAgentsMD_EmitsDeprecationWarning` — `kbz init --skip-agents-md`, assert stderr contains "deprecated" and behaviour is identical to `--skip-instructions`.
  - `TestE2E_SkipMCPWithoutSkipZed_EmitsScopeWarning` — `kbz init --skip-mcp` (no `--skip-zed`), assert stderr contains the scope-narrowing warning.
- **Deliverable:** Passing e2e tests for all five flag-behaviour cases.
- **Depends on:** T7 (harness), T2 (instructions rename), T3 (flag split). Parallelisable with T8 after T7 is complete.
- **Effort:** Medium.
- **Spec requirement:** REQ-009, REQ-010, AC-008, AC-009, AC-010, AC-011, Design §5.6 (`--skip-zed` explicit test).

---

### Task T10: Implement `kbz doctor` integration tests

- **Description:** Using the T7 harness (and fixture directories for unit tests):
  - `TestE2E_Doctor_MissingRequiredArtifact` — run init, delete a Required artifact, run `kbz doctor`, assert exit 1 and output names the missing file.
  - `TestE2E_Doctor_GhostFile` — run init, drop a `.kbz/skills/legacy/SKILL.md` not in the Manifest, run `kbz doctor`, assert exit 0 and warning names the ghost file.
  - Unit test `TestDoctor_OlderMarkerVersion` — fixture with a marker at an older version; doctor prints a warning, exits 0.
- **Deliverable:** Passing doctor tests; all AC-005 / AC-006 criteria verifiable via `go test ./internal/kbzinit -tags=e2e`.
- **Depends on:** T5 (doctor implementation), T7 (harness). Parallelisable with T8 and T9 once T5 and T7 are done.
- **Effort:** Medium.
- **Spec requirement:** REQ-006, REQ-007, AC-005, AC-006.

---

### Task T11: Add GitHub Actions CI job

- **Description:** Add a new job to an appropriate workflow file (or new file) in `.github/workflows/`. The job:
  - Triggers on `pull_request` with `paths:` filter:
    ```yaml
    paths:
      - 'internal/kbzinit/**'
      - '.kbz/skills/**'
      - '.kbz/roles/**'
    ```
  - Runs `make test-install`.
  - Must be a blocking required status check (document in the PR body / repo settings note).
  - No network access inside the test binary (REQ-NF-003 — Go module cache must be warm or `GOFLAGS=-mod=vendor`).
- **Deliverable:** New or updated workflow YAML; CI job visible in GitHub Actions on a PR touching `internal/kbzinit/`.
- **Depends on:** T6 (Makefile target must exist). Parallelisable with T7, T8, T9, T10.
- **Effort:** Small.
- **Spec requirement:** REQ-005, AC-004.

---

## Dependency Graph

```
T1  (no deps — D7 guard; conditional on F1 not already merged)
T2  (no deps — --skip-instructions rename)
T3  (no deps — --skip-zed / --skip-mcp split)
T6  (no deps — Makefile + e2e scaffold)

T4  → F2-merge, T2, T3
T5  → F2-merge
T7  → T6
T11 → T6

T8  → T7, T2, T3, T4, T5
T9  → T7, T2, T3
T10 → T7, T5
```

**Parallel groups (after F2 merges):**

| Wave | Tasks | Notes |
|------|-------|-------|
| Wave 1 | T1, T2, T3, T6 | All independent; T1 may be skipped if F1 already addressed D7 |
| Wave 2 | T4, T5, T7, T11 | T4 needs T2+T3+F2; T5 needs F2; T7 needs T6; T11 needs T6 |
| Wave 3 | T8, T9, T10 | T8 needs T7+T2+T3+T4+T5; T9 needs T7+T2+T3; T10 needs T7+T5 |

**Critical path:** `T6 → T7 → T8` (three sequential steps, each medium or larger)
Secondary gate: `F2-merge → T5 → T10 → (T8 unblocked)`

The T8 set is the plan's bottleneck. Running T4, T5, T9, T10 in parallel during Wave 2/3 minimises elapsed time.

---

## Risk Assessment

### Risk: F2 merge slip

- **Probability:** Medium (F2 is a non-trivial refactor of the install registry).
- **Impact:** High — T4, T5, T8 are all blocked until F2 lands; the e2e suite can't make meaningful Manifest assertions.
- **Mitigation:** Begin T1, T2, T3, T6 immediately in parallel while F2 is in review. T7 can also start — the harness helpers have no Manifest dependency. Coordinate with F2 implementer to keep the `Manifest` API stable once landed.
- **Affected tasks:** T4, T5, T8, T10.

### Risk: D7 already addressed by F1

- **Probability:** High — F1 (init-unblock) targets the same "incomplete init" warning path.
- **Impact:** Low — T1 becomes a trivially fast no-op if already merged.
- **Mitigation:** Before dispatching T1, inspect the F1 merge commit. If the D7 guard is present, mark T1 `not-planned` with a reference to the F1 merge SHA.
- **Affected tasks:** T1.

### Risk: e2e suite exceeds 60-second budget (REQ-NF-001)

- **Probability:** Medium — each test does a real `git init` + subprocess binary invocation; ten tests × ~5 s = 50 s at the boundary.
- **Impact:** Medium — a flaky or slow suite will be skipped or disabled, defeating the purpose.
- **Mitigation:** Use `sync.Once` to build the binary once (T7 design). Tests that don't need rollback don't need to inject failures. If budget is tight, parallelise tests with `t.Parallel()` — each test uses its own `t.TempDir()` so there is no shared state.
- **Affected tasks:** T7, T8, T9, T10.

### Risk: `--skip-mcp` deprecation warning breaks existing scripted callers

- **Probability:** Low — warnings go to stderr (constrained by spec); scripted callers typically parse stdout or check exit code only.
- **Impact:** Low — stderr is not normally captured by `$()` substitution.
- **Mitigation:** Tests in T9 explicitly verify the warning is on stderr and not stdout. Document in the release notes.
- **Affected tasks:** T3, T9.

### Risk: Rollback path tracker misses a conditional artifact

- **Probability:** Medium — T4 must enumerate paths conditionally installed based on flags (e.g. `.cursor/rules/` is optional); any path created outside the tracker's scope becomes an orphan.
- **Impact:** High — the D6 regression the plan is meant to fix resurfaces.
- **Mitigation:** Derive the tracker list from the Manifest (every `InstallPath` that was written during the current run) rather than hard-coding paths. The Manifest is the single source of truth (F2 dependency).
- **Affected tasks:** T4, T8 (recovery test will catch this).

---

## Verification Approach

| Acceptance Criterion | Verification Method | Producing Task |
|---|---|---|
| AC-001 — Tagged e2e suite runs to completion | Automated: `go test ./internal/kbzinit -tags=e2e -run TestE2E_` passes | T8, T9 |
| AC-002 — Idempotent double-init | Automated: `TestE2E_ReInstallIsIdempotent` | T8 |
| AC-003 — Partial-install recovery, no orphan files | Automated: `TestE2E_PartialInstallRecovery` with injected write failure | T8 |
| AC-004 — CI `test-install` job blocks merge on failure | Manual: open a PR touching `internal/kbzinit/`; observe CI gate | T11 |
| AC-005 — `kbz doctor` exits 1 on missing Required artifact | Automated: `TestE2E_Doctor_MissingRequiredArtifact` | T10 |
| AC-006 — `kbz doctor` exits 0 + warning on ghost file | Automated: `TestE2E_Doctor_GhostFile` | T10 |
| AC-007 — Rollback removes all install paths on failure | Automated: `TestE2E_PartialInstallRecovery` (asserts no orphan files remain) | T8 |
| AC-008 — `--skip-instructions` suppresses all instruction surfaces | Automated: `TestE2E_SkipInstructions_AlsoSkipsCopilot` | T9 |
| AC-009 — `--skip-agents-md` deprecated alias emits warning | Automated: `TestE2E_SkipAgentsMD_EmitsDeprecationWarning` | T9 |
| AC-010 — `--skip-mcp` leaves Zed settings intact | Automated: `TestE2E_SkipMCP_DoesNotCreateMCPConfig` | T9 |
| AC-011 — `--skip-mcp` scope warning when `--skip-zed` absent | Automated: `TestE2E_SkipMCPWithoutSkipZed_EmitsScopeWarning` | T9 |
| REQ-010 positive path — `--skip-zed` suppresses `.zed/settings.json` | Automated: `TestE2E_SkipZed_DoesNotCreateZedSettings` (explicit positive-path test added per MAJOR 1) | T9 |
| REQ-NF-001 — Full e2e suite under 60 s | Automated: wall-clock time observed in CI job | T8, T9, T10, T11 |
| REQ-NF-002 — `kbz doctor` under 500 ms | Automated: unit benchmark in doctor package | T5 |
| REQ-NF-003 — No network access in any e2e test | Inspection: no outbound calls in harness helpers; Go module cache used | T7 |

### Implementation notes on specific requirements

**REQ-004 dual gating** — The `e2e` build tag is the canonical mechanism. `KBZ_E2E=1` is an environment-variable alias for CI environments that cannot pass `-tags e2e` (e.g. some test runners that don't forward build flags). Both are functionally equivalent; neither supersedes the other. `TestMain` checks both; `KBZ_E2E=1` without the build tag is handled via a `//go:build ignore`-style shim or a `TestMain` env check inside the file.

**REQ-005 CI path glob** — The GitHub Actions `paths:` trigger must include exactly:
```yaml
paths:
  - 'internal/kbzinit/**'
  - '.kbz/skills/**'
  - '.kbz/roles/**'
```

**REQ-008 sentinel clause** — The sentinel write is an implementation constraint (the exact ordering of the commit point in the install sequence). The testable requirement is the AC-007 rollback observable outcome: no orphan files after a mid-init failure. The sentinel's position is validated by inspection during code review of T4.
