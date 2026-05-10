# P62-F3 Dev-Plan — Runtime Discovery Surfaces

| Field  | Value                                          |
|--------|------------------------------------------------|
| Date   | 2026-05-09                                     |
| Status | Draft                                          |
| Author | architect                                      |
| Feature | FEAT-01KR7BKXMK3B6 (B64-F3, runtime-surfaces) |
| Batch  | B64-install-skill-quality                      |

---

## ⚠️ Blocking Dependency: F2 (install-registry) must be merged first

**No tasks in this plan may begin until F2 (`FEAT-01KR7...` install-registry) is merged to
`main`.** This feature directly consumes two types that F2 delivers:

- **`Manifest`** — the canonical `[]Artifact` slice that enumerates every file the installer
  manages. F3 adds new entries to it; iterating it is the only permitted way to discover which
  `.claude/skills/` wrappers to install (see MAJOR 2 note below).
- **`MarkerSpec` comparator** — the unified comparison function (`file absent / no marker /
  older / equal / newer`) that F3 uses for CLAUDE.md, OPENAI.md, and `.cursor/rules/kanbanzai.mdc`.

If work begins before F2 is merged, tasks will conflict on `registry.go` and the comparator
interface, creating hard-to-resolve merge conflicts. The gate is not advisory — do not start.

---

## Implementation Notes

### MAJOR 1 — Newer-marker behavior for all managed surfaces (AC-009)

When the existing file carries a managed marker whose version is **greater than** the current
binary version, the installer must preserve the file verbatim and emit **no warning**. This is
the `WarnSkip → NoOp` path from the F2 `MarkerSpec` comparator:

```
present, marker, version newer → no-op (silent)
```

This applies equally to CLAUDE.md, OPENAI.md, and `.cursor/rules/kanbanzai.mdc`. It is the
same behavior already required for AGENTS.md by REQ-001 ("same comparator … as AGENTS.md").
AC-009 tests this path specifically for CLAUDE.md (marker `v999`). Task T3 must implement it
and Task T8 must cover it with an explicit test case.

### MAJOR 2 — F2 Manifest is the single authority for .claude/skills/ wrappers

REQ-003 lists seven wrapper names for illustration. **Implementers must not hard-code that
list.** The actual set of wrappers installed must exactly match the Manifest entries whose
`Kind == claudeWrapper`. At install time, iterate `Manifest`, filter by kind, and install each
entry — the spec list is illustrative, not exhaustive. If the Manifest grows or shrinks, the
installer automatically follows without a code change.

### Minor — `--skip-instructions` suppression scope

The `--skip-instructions` flag (renamed from `--skip-agents-md` in F4, or the legacy name in
the interim) must suppress **all four F3 surface kinds**: CLAUDE.md, OPENAI.md, and all
`.claude/skills/` wrappers. No per-surface skip flags are permitted. Task T6 wires this
suppression. Task T8 must include a test for this behavior.

### Minor — REQ-005/006 dual-condition idempotency

When both `--enable-cursor` is passed **and** `.cursor/` already exists, the install proceeds
normally (idempotent). No error, no duplicate-creation guard needed beyond what
`installOne()` already provides via the `MarkerSpec` comparator.

---

## Scope

This plan implements the requirements defined in
`work/P62-install-skill-quality-remediation/P62-F3-spec-runtime-surfaces.md`
(FEAT-01KR7BKXMK3B6). It covers Tasks T1–T8 below.

The plan delivers the four runtime discovery surfaces that the P59 audit found missing from
consumer installs: `CLAUDE.md`, `.claude/skills/` wrappers, `OPENAI.md`, and
`.cursor/rules/kanbanzai.mdc`. All surfaces are registered as `Artifact` entries in the F2
Manifest and use the F2 `MarkerSpec` comparator — no new ad-hoc install functions.

**Out of scope:** Changing the *content* of the installed files beyond injecting managed
markers; new flag taxonomy beyond `--enable-cursor`; `kbz doctor` / `kbz init --check` (F4);
build-time corpus check (F4); the `--skip-mcp` / `--skip-zed` split (F4).

---

## Task Breakdown

### Task T1: Embed F3 source files into the binary

- **Description:** Add `//go:embed` directives (or equivalent embed FS additions) for each F3
  source file: `CLAUDE.md`, `OPENAI.md`, `.cursor/rules/kanbanzai.mdc`, and all
  `.claude/skills/<name>/SKILL.md` wrapper files that live in this repository. The embedded
  sources become the canonical content written to consumer repos.
- **Deliverable:** Embed FS variables (or additions to the existing F2 embed FS) that make all
  F3 source files accessible at runtime without filesystem access.
- **Depends on:** None (can start as soon as F2 is merged to main, giving T1 access to the
  embed FS scaffolding established by F2).
- **Effort:** Small
- **Spec requirement:** REQ-001, REQ-002, REQ-003, REQ-007 (all four artifact kinds require
  embedded source)

### Task T2: Add F3 Artifact entries to the Manifest

- **Description:** Extend the `Manifest` slice (in `internal/kbzinit/registry.go` or the
  equivalent file established by F2) with entries for every F3 surface:
  - One entry `Kind: claudeMd` for `CLAUDE.md`
  - One entry `Kind: openaiRedirect` for `OPENAI.md`
  - N entries `Kind: claudeWrapper` — one per `.claude/skills/<name>/SKILL.md` wrapper file
    present in the embed FS (iterated from the embed FS, not hard-coded)
  - One entry `Kind: cursorRule` for `.cursor/rules/kanbanzai.mdc`

  Each entry carries the appropriate `MarkerSpec` (HTML comment for markdown files; YAML
  frontmatter comment for SKILL.md wrappers). The naming convention for claudeWrapper entries
  must implement REQ-004: wrappers whose canonical target lives under `.agents/skills/kanbanzai-*/`
  use the `kanbanzai-` prefix; wrappers whose target lives under `.kbz/skills/<name>/` keep the
  bare name.
- **Deliverable:** Updated `registry.go` (or equivalent) with all F3 `Artifact` entries
  compilable and passing `go build ./...`.
- **Depends on:** T1 (embed FS must exist before Manifest entries can reference embed paths)
- **Effort:** Medium
- **Spec requirement:** REQ-001, REQ-002, REQ-003, REQ-004, REQ-007

### Task T3: Implement install logic for CLAUDE.md and OPENAI.md

- **Description:** Implement (or extend) `installOne()` to handle `Kind: claudeMd` and
  `Kind: openaiRedirect`. The function must invoke the F2 `MarkerSpec` comparator, which
  covers all five cases:
  - File absent → create (write embedded content)
  - Present, no marker → warn-and-skip (preserve user file, print warning + recovery)
  - Present, marker, version older → overwrite
  - Present, marker, version equal → no-op
  - Present, marker, version newer → **no-op, no warning** (MAJOR 1: the `WarnSkip → NoOp` path)

  CLAUDE.md content must reference `AGENTS.md` and inventory the installed `.claude/skills/`
  wrappers; OPENAI.md is a one-paragraph redirect to `AGENTS.md`.
- **Deliverable:** `installOne()` handling for claudeMd and openaiRedirect, with all five
  comparator cases exercised.
- **Depends on:** T2 (Manifest entries must exist before install logic can reference them)
- **Effort:** Medium
- **Spec requirement:** REQ-001, REQ-002, REQ-007, REQ-008 (AC-001, AC-002, AC-008, AC-009)

### Task T4: Implement install logic for .claude/skills/ wrappers

- **Description:** Implement (or extend) `installOne()` for `Kind: claudeWrapper`. The
  installer must iterate over Manifest entries filtered by `Kind == claudeWrapper` (never a
  hard-coded list) and for each: create the target directory if absent, write the embedded
  SKILL.md wrapper with the managed marker injected into YAML frontmatter (if not already
  present). Apply the same `MarkerSpec` comparator logic as T3.

  The naming prefix rule (REQ-004) is already encoded in the Manifest entries from T2 —
  `installOne()` uses each entry's `InstallPath` directly without additional prefix logic.
- **Deliverable:** `installOne()` handling for claudeWrapper; all wrapper directories created
  under `.claude/skills/` during a fresh init.
- **Depends on:** T2
- **Effort:** Medium
- **Spec requirement:** REQ-003, REQ-004, REQ-007, REQ-008 (AC-003, AC-004)

### Task T5: Implement conditional cursorRule install and --enable-cursor flag

- **Description:** Two sub-parts, kept in one task due to their tight coupling:
  1. Add a `--enable-cursor` boolean flag (default false) to `kbz init`.
  2. Implement install guard for `Kind: cursorRule`: install `.cursor/rules/kanbanzai.mdc` only
     when `.cursor/` already exists at install time OR `--enable-cursor` is true. When neither
     condition holds, skip silently (no warning). When either condition holds, create
     `.cursor/rules/` if absent and write `kanbanzai.mdc` via `installOne()` with the standard
     `MarkerSpec` comparator. Both conditions being true simultaneously is idempotent.
- **Deliverable:** `--enable-cursor` flag wired to `kbz init`; conditional install guard for
  cursorRule; `.cursor/rules/` created only when gated.
- **Depends on:** T2
- **Effort:** Medium
- **Spec requirement:** REQ-005, REQ-006, REQ-007 (AC-005, AC-006, AC-007)

### Task T6: Wire --skip-instructions suppression for all F3 surfaces

- **Description:** Extend the `--skip-instructions` flag handler to suppress all four F3
  surface kinds: `claudeMd`, `openaiRedirect`, `claudeWrapper` (all entries), and `cursorRule`.
  The suppression should operate at the Manifest-iteration level: filter out any Artifact whose
  Kind is in the suppression set before calling `installOne()`. No per-surface skip flags.
- **Deliverable:** `--skip-instructions` skips CLAUDE.md, OPENAI.md, and all `.claude/skills/`
  wrappers (and cursorRule) in a single flag check; verified by code inspection and test T8.
- **Depends on:** T3, T4, T5 (all three install paths must exist before suppression can be
  verified end-to-end)
- **Effort:** Small
- **Spec requirement:** Constraint §"Must use the same `--skip-instructions` flag"

### Task T7: Extend TestEmbeddedCorpus for F3 artifacts

- **Description:** Extend the F2 `TestEmbeddedCorpus` test (in `internal/kbzinit/`) to assert
  that every F3 Manifest entry is present and structurally valid:
  - CLAUDE.md exists in embed FS and contains the HTML managed-marker comment
  - OPENAI.md exists in embed FS
  - Every `claudeWrapper` entry's embed path exists and contains a `# kanbanzai-managed:`
    YAML frontmatter line
  - `.cursor/rules/kanbanzai.mdc` exists and has a valid `description:` frontmatter field

  Add negative fixture tests: the corpus check fails with a descriptive error when any of the
  above entries are missing from the Manifest.
- **Deliverable:** Extended `TestEmbeddedCorpus` with F3-specific assertions; negative-fixture
  subtests that fail if Manifest is missing CLAUDE.md, OPENAI.md, or any wrapper (AC-010).
- **Depends on:** T2 (Manifest entries must exist for the corpus check to walk)
- **Effort:** Small
- **Spec requirement:** REQ-007, AC-010

### Task T8: Integration and e2e tests for AC-001 through AC-009

- **Description:** Write integration/e2e tests covering all functional acceptance criteria.
  Tests should use the pattern established by F2's e2e test suite (temp dir, `git init`,
  controlled flags). Required test cases:
  - **AC-001:** Fresh init → CLAUDE.md present at root, marker on line 1, references AGENTS.md
    and `.claude/skills/`
  - **AC-002:** Fresh init → OPENAI.md present, references AGENTS.md
  - **AC-003:** Fresh init → every `claudeWrapper` Manifest entry has a corresponding
    `.claude/skills/<name>/SKILL.md` with `# kanbanzai-managed:` line (iterate Manifest, not
    a fixed list)
  - **AC-004:** Fresh init → `kanbanzai-getting-started/` and `kanbanzai-workflow/` exist
    alongside `write-design/` and `review-code/` (prefix naming convention)
  - **AC-005:** No `.cursor/`, no `--enable-cursor` → `.cursor/` not created
  - **AC-006:** Pre-created `.cursor/` dir, no flag → `.cursor/rules/kanbanzai.mdc` written
  - **AC-007:** No `.cursor/`, `--enable-cursor` flag → `.cursor/rules/` created and
    `kanbanzai.mdc` written
  - **AC-008:** Pre-written unmanaged CLAUDE.md (no marker) → file preserved, warning on
    stdout naming the file with recovery instruction
  - **AC-009 (MAJOR 1):** Pre-written CLAUDE.md with marker `<!-- kanbanzai-managed: v999 -->`
    → file preserved verbatim, **no warning on stdout** (newer-marker no-op path)
  - **--skip-instructions suppression:** `kbz init --skip-instructions` → CLAUDE.md,
    OPENAI.md, and all `.claude/skills/` wrappers absent after init
- **Deliverable:** Test functions in `internal/kbzinit/e2e_test.go` (or equivalent), each
  asserting one AC. All pass under `KBZ_E2E=1 go test ./internal/kbzinit/...`.
- **Depends on:** T3, T4, T5, T6 (all install paths must be implemented before e2e tests can
  pass)
- **Effort:** Large
- **Spec requirement:** REQ-001 – REQ-008, AC-001 – AC-009

---

## Dependency Graph

```
T1: Embed F3 source files          (no dependencies)
T2: Manifest entries               → depends on T1
T3: Install CLAUDE.md + OPENAI.md  → depends on T2
T4: Install .claude/skills/        → depends on T2
T5: cursorRule + --enable-cursor   → depends on T2
T7: Extend TestEmbeddedCorpus      → depends on T2
T6: --skip-instructions wiring     → depends on T3, T4, T5
T8: Integration + e2e tests        → depends on T3, T4, T5, T6
```

**Parallel groups:**
- Phase 0 (pre-start): F2 must be merged — no tasks start before this
- Phase 1 (start): `[T1]` — independent
- Phase 2 (after T1): `[T2]` — single task, unblocks Phase 3
- Phase 3 (after T2): `[T3, T4, T5, T7]` — all independent of each other
- Phase 4 (after T3+T4+T5): `[T6]`
- Phase 5 (after T6): `[T8]`

**Critical path:** F2 merge → T1 → T2 → (T3 or T4 or T5, whichever is longest) → T6 → T8

T7 is off the critical path and can be completed in parallel with T3–T5.

---

## Risk Assessment

### Risk: F2 merge is delayed
- **Probability:** Medium (F2 is in review; CI or rework can delay it)
- **Impact:** High (all F3 tasks are blocked; no partial start is possible safely)
- **Mitigation:** Track F2 PR status actively. Do not start any F3 task against a local
  F2 branch — wait for the merge to main. If F2 is significantly delayed, the orchestrator
  should escalate to unblock rather than work around the dependency.
- **Affected tasks:** All (T1–T8)

### Risk: Manifest enumeration drifts from actual embed FS
- **Probability:** Low (T7 corpus check will catch this at CI time)
- **Impact:** Medium (wrappers silently missing from consumer installs)
- **Mitigation:** MAJOR 2 resolution: T4 iterates Manifest, not a hard-coded list. T7 adds
  corpus assertions. Both together prevent silent drift from reaching users.
- **Affected tasks:** T2, T4, T7

### Risk: CLAUDE.md content conflicts with AGENTS.md or installed skills
- **Probability:** Low (content is a redirect + inventory, not a duplication)
- **Impact:** Medium (contradictory guidance causes agent confusion)
- **Mitigation:** T3 generates CLAUDE.md content from the same Manifest that produced
  AGENTS.md's skill inventory — structural consistency is enforced, not asserted. T7 extends
  the corpus check to verify content alignment.
- **Affected tasks:** T3, T7

### Risk: Binary size budget exceeded (REQ-NF-002: < 50 KB)
- **Probability:** Low (.claude/skills/ wrappers are short redirect files)
- **Impact:** Low (soft requirement; no user-visible failure, only a spec gap)
- **Mitigation:** Measure binary size delta as part of T1 (compare `go build` output before
  and after embedding). If budget is at risk, compress long wrapper content before embedding.
- **Affected tasks:** T1

### Risk: .cursor/rules/ created silently for non-Cursor users (Constraint violation)
- **Probability:** Low (T5 guard logic is straightforward)
- **Impact:** High (user-visible unwanted side effect; violates an explicit constraint)
- **Mitigation:** T5 unit test exercises both the "no .cursor/, no flag → skip" path (AC-005)
  and the "flag present → create" path (AC-007) before T8 runs e2e. The guard logic is a
  single boolean condition — low surface area for bugs.
- **Affected tasks:** T5, T8

---

## Verification Approach

Every acceptance criterion is covered by an automated test. The producing task is noted.

| Acceptance Criterion | Verification Method | Producing Task |
|---|---|---|
| AC-001: CLAUDE.md present with marker on line 1, references AGENTS.md + .claude/skills/ | e2e test: fresh init, assert file content | T8 |
| AC-002: OPENAI.md present, references AGENTS.md | e2e test: fresh init, assert file presence and content | T8 |
| AC-003: Every Manifest claudeWrapper entry has a corresponding .claude/skills/<name>/SKILL.md with managed marker | e2e test: iterate Manifest, assert each path exists | T8 |
| AC-004: kanbanzai-getting-started/ and kanbanzai-workflow/ exist alongside bare-named dirs | e2e test: assert directory names after fresh init | T8 |
| AC-005: No .cursor/ when neither .cursor/ pre-exists nor --enable-cursor passed | e2e test: assert .cursor/ absent | T8 |
| AC-006: .cursor/rules/kanbanzai.mdc written when .cursor/ pre-exists | e2e test: pre-create .cursor/, assert mdc written | T8 |
| AC-007: .cursor/rules/ created and mdc written when --enable-cursor passed | e2e test: pass flag, assert .cursor/rules/ created | T8 |
| AC-008: Unmanaged CLAUDE.md preserved verbatim with warning | e2e test: pre-write CLAUDE.md without marker, assert preservation + warning | T8 |
| AC-009: CLAUDE.md with marker v999 preserved verbatim, no warning (newer-marker no-op) | e2e test: pre-write v999 CLAUDE.md, assert preservation and empty stderr | T8 |
| AC-010: TestEmbeddedCorpus fails when CLAUDE.md, OPENAI.md, or any wrapper missing from Manifest | Unit test: negative fixtures remove entries, assert test failure | T7 |
| --skip-instructions suppresses CLAUDE.md, OPENAI.md, all wrappers | e2e test: pass flag, assert none of the four kinds installed | T8 |
| REQ-NF-001: Total install-time increase < 50 ms | Benchmark or timed e2e run (informational, not a gate) | T8 |
| REQ-NF-002: Embedded content adds < 50 KB to binary | Binary size comparison before/after T1 embedding | T1 |
