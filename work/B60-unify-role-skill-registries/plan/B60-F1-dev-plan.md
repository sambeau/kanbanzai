| Field  | Value |
|--------|-------|
| Date   | 2026-05-08T11:49:05Z |
| Status | Draft |
| Author | architect |
| Feature | FEAT-01KR3MEJGGMT5 — Generated Role and Skill Registry Surfaces |
| Batch | B60 — Unify role and skill registries |

## Scope

This plan implements the requirements defined in
`work/B60-unify-role-skill-registries/B60-F1-spec-generated-role-skill-registry-surfaces.md`
(DOC pending). It covers all five tasks below.

It does **not** cover:
- Generating or modifying `AGENTS.md` narrative content (deferred by spec constraint and design decision D7).
- Role/skill content cleanup or de-duplication (B61 scope).
- `.claude/skills/` runtime wrappers (B62 scope).
- Any changes to the `.kbz/stage-bindings.yaml` schema.

The implementation introduces a new `internal/registry` package and a `kbz docs` CLI command.
It touches three existing instruction files (`CLAUDE.md`, `.github/copilot-instructions.md`,
`README.md`) only to insert generated-region markers and initial generated content.

## Task Breakdown

### Task 1: Registry extraction model (`internal/registry` — extractor + model)

- **Description:** Create `internal/registry/` with the data model and extractor. The
  extractor reads `.kbz/stage-bindings.yaml` and all `.kbz/roles/*.yaml` files from a given
  root path and returns a `RegistryModel` struct. The model contains:
  - Ordered list of `StageEntry` (name, description, roles, skills, human-gate flag,
    document type, prerequisites summary, source path).
  - Map of `RoleEntry` (id, identity, inherits, source path) keyed by role ID.
  - `RoleEntry` includes only the metadata fields needed for registry tables (id, identity,
    inherits chain, source path); it does not load anti-patterns, vocabulary, or tools.
  Extractor must return a deterministic model: stages ordered by their declaration order
  in stage-bindings.yaml; roles ordered lexicographically by filename.
  Error reporting must name the file that caused any parse failure (REQ-NF-005).
- **Deliverable:** `internal/registry/model.go`, `internal/registry/extractor.go`,
  `internal/registry/extractor_test.go` (fixture-based unit tests covering: normal load,
  missing roles dir, malformed YAML, empty corpus).
- **Depends on:** None.
- **Effort:** 3 points (medium).
- **Spec requirements:** REQ-001, REQ-002, REQ-NF-003, REQ-NF-005; verifies AC-001.

---

### Task 2: Markdown region parser and writer (`internal/registry/region.go`)

- **Description:** Implement the region parser and writer that locates named generated
  regions in a Markdown file, replaces their inner content, and detects structural errors.
  Region markers use the format:
  ```
  <!-- registry-gen:begin:REGION-NAME source=... -->
  ...generated content...
  <!-- registry-gen:end:REGION-NAME -->
  ```
  The parser returns the list of named regions with their byte offsets. The writer replaces
  only the content between markers, preserving all bytes before the opening marker and after
  the closing marker byte-for-byte.
  The parser must return a clear error (including file path and marker name) for:
  - Missing begin or end marker (REQ-010).
  - Duplicated begin or end marker (REQ-010).
  - Nested regions (begin marker before previous end marker) (REQ-010).
  Check mode compares the current between-markers content with the candidate content and
  returns `(stale bool, region string, file string)`.
  Sync mode replaces the between-markers content.
  Running sync twice on unchanged input must produce identical bytes (REQ-NF-001).
- **Deliverable:** `internal/registry/region.go`, `internal/registry/region_test.go`
  (tests covering: normal replace, byte-for-byte preservation of surrounding prose,
  missing marker, duplicated marker, nested marker, idempotency).
- **Depends on:** None (pure file-string operations, no dependency on Task 1).
- **Effort:** 3 points (medium).
- **Spec requirements:** REQ-003, REQ-004, REQ-005, REQ-006, REQ-010, REQ-NF-001; verifies
  AC-002, AC-003, AC-007, AC-009.

---

### Task 3: Registry content renderer (`internal/registry/render.go`)

- **Description:** Implement the renderer that accepts a `RegistryModel` (from Task 1) and
  returns the generated Markdown string for each target region. Two regions are required:
  - **`roles-and-skills`**: A combined Markdown table with columns
    `| Stage | Description | Roles | Skills | Gate | Doc Type |` derived from
    stage-bindings entries. Stage rows reference the source path to the canonical binding
    file.
  - **`role-index`**: A Markdown table with columns `| Role | Identity | Inherits | Source |`
    derived from role entries, sorted lexicographically by role ID.
  Each generated region must open with a human-readable header and a warning comment
  identifying the canonical sources (REQ-004, REQ-009). Rendered content must not include
  full skill procedures, examples, or anti-pattern bodies (REQ-NF-004). Output must be
  deterministic: given the same model, the renderer always produces identical bytes
  (REQ-NF-003).
- **Deliverable:** `internal/registry/render.go`, `internal/registry/render_test.go`
  (snapshot/golden-file tests; tests asserting deterministic output on repeated calls with
  same input; tests asserting no procedure/example bodies in output).
- **Depends on:** Task 1 (consumes `RegistryModel`).
- **Effort:** 2 points (small).
- **Spec requirements:** REQ-001, REQ-002, REQ-004, REQ-009, REQ-NF-003, REQ-NF-004;
  verifies AC-001, AC-002, AC-006, AC-011.

---

### Task 4: `kbz docs` CLI command (`cmd/kbz/docs_cmd.go`)

- **Description:** Add the `kbz docs` subcommand with two actions:
  - `kbz docs sync [--root <path>]` — runs the extractor, renders all regions, and writes
    updated content into all in-scope target files. Exits 0 on success. Reports each file
    written.
  - `kbz docs check [--root <path>]` — runs the extractor, renders all regions, compares
    with on-disk content without writing, exits non-zero and reports the stale file(s) if
    any region is stale (REQ-006). Exits 0 if all regions are current. Must complete in
    under 2 seconds on the repository corpus (REQ-NF-002).
  Target files are hardcoded as `CLAUDE.md`, `.github/copilot-instructions.md`, and
  `README.md` relative to the project root (REQ-003, REQ-011). If `--root` is omitted,
  defaults to the working directory.
  `kbz docs sync` must never write to `AGENTS.md` (REQ-011). An explicit guard log line
  acknowledges `AGENTS.md` is intentionally excluded.
  Register the command in `cmd/kbz/main.go` under the `"docs"` case.
- **Deliverable:** `cmd/kbz/docs_cmd.go`, integration test in
  `cmd/kbz/docs_cmd_test.go` (tests covering: sync on a tmp-dir fixture with markers
  produces expected output; check exits non-zero on stale fixture; check exits 0 on
  current fixture; AGENTS.md is not modified; timing assertion for check mode).
- **Depends on:** Task 1, Task 2, Task 3.
- **Effort:** 3 points (medium).
- **Spec requirements:** REQ-003, REQ-005, REQ-006, REQ-007, REQ-011, REQ-NF-002,
  REQ-NF-005; verifies AC-003, AC-004, AC-005, AC-008, AC-010.

---

### Task 5: Marker placement, initial generated sync, CI target

- **Description:** This integration task makes the feature observable end-to-end:
  1. Insert `<!-- registry-gen:begin:... -->` / `<!-- registry-gen:end:... -->` marker
     pairs into `CLAUDE.md`, `.github/copilot-instructions.md`, and `README.md` at the
     appropriate locations (replacing the existing hand-authored registry tables for roles
     and skills). The `source=` attribute in each begin marker names the canonical source
     files.
  2. Run `kbz docs sync` against the repository root to populate the generated regions.
     Verify with a second run that no diff is produced (AC-009).
  3. Add `registry-check` to the `Makefile`:
     ```
     registry-check:
         go run ./cmd/kbz docs check
     ```
  4. Add a `registry-check` job to `.github/workflows/ci.yml` (create this file if no
     general CI workflow exists yet) that runs `make registry-check` on PRs and pushes to
     `main`. The job must fail CI when generated regions are stale (REQ-008, AC-005).
  5. Verify visually that each generated table in the three target files contains every
     configured stage and bound role with a correct source path (AC-006).
- **Deliverable:** Updated `CLAUDE.md`, `.github/copilot-instructions.md`, `README.md`
  (with markers and initial generated content); updated `Makefile`; new or updated
  `.github/workflows/ci.yml`.
- **Depends on:** Task 4.
- **Effort:** 2 points (small).
- **Spec requirements:** REQ-003, REQ-007, REQ-008; verifies AC-002, AC-005, AC-006,
  AC-008, AC-009.

## Dependency Graph

```
Task 1 (extractor + model)     — no dependencies
Task 2 (region parser/writer)  — no dependencies
Task 3 (renderer)              — depends on Task 1
Task 4 (kbz docs CLI)          — depends on Task 1, Task 2, Task 3
Task 5 (markers + CI)          — depends on Task 4
```

Parallel groups:
- **Round 1 (parallel):** Task 1, Task 2
- **Round 2 (after T1):** Task 3
- **Round 3 (after T1, T2, T3):** Task 4
- **Round 4 (after T4):** Task 5

Critical path: Task 1 → Task 3 → Task 4 → Task 5 (four hops).

No false dependencies. Tasks 1 and 2 share no types and can be implemented independently.
Task 3 consumes `RegistryModel` from Task 1 — that interface boundary is well-defined in
the model.go deliverable, so Task 3 can begin as soon as Task 1's types are committed even
if the extractor is still being tested.

## Risk Assessment

### Risk: Stage-bindings YAML ordering is non-deterministic across Go map iteration

- **Probability:** High (Go map iteration is random by definition).
- **Impact:** Medium (violates REQ-NF-003 determinism requirement; would cause check mode
  to report false positives).
- **Mitigation:** Extractor must collect stage names into a slice in declaration order by
  parsing the YAML as an ordered sequence (`yaml.Node` walk or a custom ordered-map type),
  not by decoding into a `map[string]*StageBinding`. The existing `internal/binding` package
  already uses `map[string]*StageBinding`; the registry extractor must not reuse that
  decode path if it loses order.
- **Affected tasks:** Task 1, Task 3.

### Risk: Existing instruction files have inconsistent or absent locations for marker insertion

- **Probability:** Medium (files are long and hand-maintained; the exact location for marker
  insertion requires careful placement to preserve surrounding prose).
- **Impact:** Low-medium (wrong placement would confuse readers; surrounding prose could be
  disrupted).
- **Mitigation:** Task 5 implementer should read the full current content of each target
  file before placing markers, and include the surrounding section headings in the commit
  diff for reviewer inspection. Review the diff before merging.
- **Affected tasks:** Task 5.

### Risk: No CI workflow file exists for general PRs

- **Probability:** Medium (only `release.yml` exists in `.github/workflows/`; no PR/push
  workflow was found).
- **Impact:** Low (Task 5 will create `ci.yml`; the risk is only that the new workflow needs
  initial configuration effort).
- **Mitigation:** Task 5 creates `.github/workflows/ci.yml` with the registry-check job.
  Keep the initial workflow minimal to avoid scope creep; only add the `registry-check` job.
- **Affected tasks:** Task 5.

### Risk: check-mode timing requirement (under 2 seconds) fails in slow CI environments

- **Probability:** Low (the corpus is small: ~15 stages, ~15 roles, 3 files).
- **Impact:** Low (AC-010 is a local-execution criterion per spec; CI environment timing is
  not required to meet the same bound).
- **Mitigation:** No special work required. Document the 2-second bound as a local
  performance target in the test comment; do not fail CI solely on timing.
- **Affected tasks:** Task 4.

## Verification Approach

| Acceptance Criterion | Verification Method | Producing Task |
|---------------------|---------------------|----------------|
| AC-001: All stages and roles in extracted model | Unit test (`extractor_test.go`) | Task 1 |
| AC-002: Generated-region markers identify source inputs | Inspection of target file diffs | Task 5 |
| AC-003: Surrounding prose preserved byte-for-byte | Unit test (`region_test.go`) | Task 2 |
| AC-004: Check mode exits non-zero on stale region | Integration test (`docs_cmd_test.go`) | Task 4 |
| AC-005: CI runs registry-check | CI config inspection + workflow run | Task 5 |
| AC-006: Generated paths point to existing source files | Inspection post-sync | Task 5 |
| AC-007: Missing/duplicated/nested marker returns clear error | Unit test (`region_test.go`) | Task 2 |
| AC-008: `AGENTS.md` not modified by sync mode | Regression test (`docs_cmd_test.go`) | Task 4 |
| AC-009: Sync twice produces no diff | Integration test (`docs_cmd_test.go`) | Task 4 |
| AC-010: Check mode completes under 2 seconds | Timing test (`docs_cmd_test.go`) | Task 4 |
| AC-011: No full procedures/examples in generated content | Snapshot test + inspection | Task 3 |
