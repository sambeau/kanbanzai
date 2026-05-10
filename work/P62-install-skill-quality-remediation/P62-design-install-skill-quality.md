# P62 — Design: Install Pipeline & Skill Quality Remediation

**Plan:** P62-install-skill-quality-remediation
**Author:** architect
**Status:** draft
**Parent audit:** `work/P59-roles-skills-remediation/audit-install-refactor-prompt.md`
  → audit findings (this repo's chat log) dated 2025-11-29

## 1. Context

The P59 refactor reshaped Kanbanzai's instruction corpus to improve
discoverability for AI agents. An audit (see parent audit) confirmed that
the refactor works for *this* repository but uncovered three classes of
defect that affect *consumer* projects:

1. **Install / upgrade is broken end-to-end.** A missing managed marker
   in `internal/kbzinit/skills/task-execution/orchestrate-development/SKILL.md`
   causes every second `kbz init` and every `--update-skills` run to abort
   with a hard error. The `--skip-agents-md` and `--skip-mcp` flag names
   silently affect a second file each. `work/` is not always created on
   first init. Stage-bindings is rewritten on every re-init because the
   version comparator doesn't accept the semver string the install writes.
2. **Discovery surfaces are repo-only.** `CLAUDE.md`, `.claude/skills/`,
   `OPENAI.md`, and `.cursor/rules/kanbanzai.mdc` exist only in this
   repository. None of them are embedded or installed by `kbz init`. For
   Claude Code consumers this means there is no auto-discovered entry
   point at all.
3. **No regression coverage for skills or roles.** The corpus is curated
   by hand (and increasingly by AI editors). When a skill or role
   regresses — content removed, marker stripped, contradictions
   reintroduced — nothing fails. Recent regressions (the missing
   managed marker; CLAUDE.md/getting-started "stash" contradiction)
   went unnoticed until manually audited.

This design addresses all three. It is deliberately scoped so each part
can ship independently.

## 2. Goals & non-goals

### Goals

- **G1 — Idempotent, recoverable install.** Re-running `kbz init` in any
  state must converge to the same correct end state without errors.
- **G2 — Complete consumer surface.** A fresh `kbz init` must deliver
  every runtime-discoverable file each supported agent runtime needs:
  `AGENTS.md`, `.github/copilot-instructions.md`, `CLAUDE.md`,
  `.claude/skills/`, `.cursor/rules/`. `OPENAI.md` is optional (it's a
  pure redirect to `AGENTS.md`).
- **G3 — Honest version comparison.** Managed markers must compare
  apples to apples; release builds and dev builds must both have
  predictable update behaviour.
- **G4 — Automated install verification.** A test suite must exercise
  fresh install, re-install, `--update-skills`, and skip-flag
  combinations against a real binary in a scratch repo, and run in CI.
- **G5 — Skill / role regression detection.** Structural and semantic
  invariants over `.kbz/skills/`, `.agents/skills/`, and `.kbz/roles/`
  must be checked automatically — both on this repo's CI and on every
  consumer install (the latter at install time, fail-fast).
- **G6 — Discoverable failure.** When the install can't proceed (custom
  unmanaged file, newer version present, etc.), the warning must name
  the affected file and the recovery action.

### Non-goals

- Changing what role / skill content *says*. This is purely about
  delivery and quality enforcement.
- A new flag taxonomy. We rename two flags but keep the existing
  command shape.
- Migrating consumers automatically from old layouts (P59 already did
  this).
- Building a full content-quality LLM evaluator for skills. That belongs
  in B25-class work; this design provides the hooks but only ships
  rule-based checks.

## 3. Scope

In scope:

- `internal/kbzinit/` install pipeline.
- `internal/kbzinit/skills/`, `internal/kbzinit/roles/`,
  `internal/kbzinit/stage-bindings.yaml` embedded sources.
- A new `internal/kbzinit/claude.go` (and `cursor.go`, `openai.go`)
  to deliver runtime surfaces other than AGENTS.md / Copilot.
- A new `internal/kbzregistry/` (or similar) package owning the canonical
  list of skills/roles and validating their structure.
- A new `cmd/kbz/doctor` (or `init --check`) that validates an existing
  install in place.
- `internal/kbzinit/*_test.go` for unit coverage and a new
  `tests/install/` package for end-to-end install scenarios.
- A CI job that builds the binary and runs the install test suite.

Out of scope:

- LLM-based content evaluation of skill prose.
- Multi-language support for AGENTS.md content.
- Telemetry / opt-in install reporting.

## 4. Current architecture (problem statement)

```
kbz init
 ├─ findKbzDir / isNewProject
 ├─ runNewProject  (no commits)              ──┐
 │                                              │ both call
 └─ runExistingProject  (commits or .kbz/)   ──┤  the same install steps
                                                │
   installStageBindings  ── always overwrites ──┤
   installSkills          (.agents/skills/)    │
   installTaskSkills      (.kbz/skills/)       │
   writeMCPConfig + writeZedConfig             │
   writeAgentsMD + writeCopilotInstructions    │
   installRoles           (.kbz/roles/)        │
   write sentinel ────────────────────────────┘
```

Defects rooted in this shape:

| # | Defect | Root location |
|---|---|---|
| D1 | `orchestrate-development/SKILL.md` missing `# kanbanzai-managed:` line — re-init aborts | `internal/kbzinit/skills/task-execution/orchestrate-development/SKILL.md` |
| D2 | Skill installer rejects (rather than re-marks) files it itself wrote without a marker | `internal/kbzinit/task_skills.go` `installOneTaskSkill` |
| D3 | Stage-bindings comparator returns "" for any non-integer version → always rewrites | `internal/kbzinit/stage_bindings.go` `extractStageBindingsVersion` |
| D4 | No install path for `.claude/skills/`, `CLAUDE.md`, `OPENAI.md`, `.cursor/rules/` | none — files exist only in repo |
| D5 | `--skip-agents-md` silently also disables `copilot-instructions.md`; `--skip-mcp` silently also disables Zed | `init.go` runNewProject/runExistingProject |
| D6 | Partial-install rollback only removes `.kbz/`, leaves orphan `.agents/`, `.github/`, `AGENTS.md`, `.mcp.json` | `init.go` `runNewProject` defer |
| D7 | "previous init appears incomplete" warning fires on truly fresh repos | `init.go` `runExistingProject` |
| D8 | `work/` not created on first init in a repo that already has commits | path through `runExistingProject` `kbzExisted==false` branch |
| D9 | No regression check that catches D1, missing skills, marker drift, or content gaps | nothing exists |

## 5. Proposed architecture

### 5.1 Single source of truth: a registry

Introduce `internal/kbzinit/registry.go` (or a new package
`internal/kbzregistry/`) declaring the canonical install manifest:

```go
type Artifact struct {
    Name        string      // "kanbanzai-getting-started"
    Kind        ArtifactKind // workflowSkill | taskSkill | role | claudeWrapper | cursorRule | openaiRedirect | claudeMd | …
    EmbedPath   string      // path inside the embed.FS
    InstallPath string      // path under the consumer repo root
    Required    bool        // fail install on missing
    Optional    bool
    Marker      MarkerSpec  // which managed-marker scheme applies
}

var Manifest = []Artifact{ … }
```

Every install step iterates over the manifest filtered by `Kind`. The
list of skills, roles, and runtime wrappers becomes data, not three
separate hard-coded slices in three Go files.

Benefits:

- D1/D2 — the registry is the single canonical list checked at *build
  time* (see 5.5) so missing markers can never be merged.
- D4 — adding `.claude/skills/`, `CLAUDE.md`, `OPENAI.md`, `.cursor/rules/`
  is an N-line manifest change instead of a new install function.
- D9 — the manifest itself becomes the test fixture for fresh-install
  tests; "did the install produce every Required artifact" is a single
  assertion.

### 5.2 Generalised marker scheme

Replace per-file ad-hoc parsing (`readMarkdownManagedVersion`,
`extractVersion`, `extractYAMLVersion`, `extractStageBindingsVersion`)
with one comparator that operates on:

```go
type MarkerSpec struct {
    Comment      string // "<!-- … -->" or "# …"
    VersionKind  VersionKind // intCounter | semver
    CurrentValue string      // "3" or "v9.9.9"
}
```

Comparison rules (one place, fully unit-tested):

- file absent → create
- present, no marker → warn-and-skip (preserve user customisation)
- present, marker, version older → overwrite
- present, marker, version equal → no-op
- present, marker, version newer → no-op
- present, marker, version unparseable → warn-and-skip (do **not**
  overwrite — current code silently rewrites)

This fixes D3 directly: stage-bindings can use either an integer counter
(preferred) or be tagged with the binary semver and compared against
that, without a rule that rejects everything that isn't an integer.

### 5.3 Self-healing skill installer

`installOne(artifact)` becomes:

```
read embedded source
if marker missing in embedded source:
    inject marker (do not silently produce a marker-less file)
write transformed bytes to disk
```

D1/D2 stop being possible. We also keep the build-time check (5.5) so we
notice marker omissions in PR review before they ship.

### 5.4 Runtime surfaces for every supported agent

Add manifest entries (and one small generator file each, kept tiny):

| Artifact | Kind | InstallPath | Notes |
|---|---|---|---|
| `CLAUDE.md` | claudeMd | `CLAUDE.md` | Generated content; marker on line 1; redirects to `AGENTS.md` plus a Claude-Skills inventory |
| `.claude/skills/<name>/SKILL.md` × N | claudeWrapper | mirrors `.claude/skills/` in this repo | Pre-existing wrapper text — copy verbatim with marker injected |
| `.cursor/rules/kanbanzai.mdc` | cursorRule | `.cursor/rules/kanbanzai.mdc` | Same content as the repo file; install only if `.cursor/` already exists OR if the user passes `--enable-cursor` (avoid creating Cursor state for non-Cursor users) |
| `OPENAI.md` | openaiRedirect | `OPENAI.md` | Pure redirect; safe to always install (small, content-free) |

Rules of thumb:

- **Always install** files that any agent runtime might inject *and* that
  are inert for runtimes that don't care (`AGENTS.md`, `OPENAI.md`,
  `CLAUDE.md`).
- **Conditionally install** files that imply opting into a tool the user
  may not have (`.cursor/rules/` only when `.cursor/` exists; same for
  `.zed/` settings).
- **Never silently install hidden directories** that have side effects
  on tools the user doesn't use; gate behind a flag with an obvious name.

### 5.5 Build-time corpus check

Add a `go test ./internal/kbzinit -run TestEmbeddedCorpus` that:

1. Walks every entry in `Manifest`.
2. Asserts every embedded source file exists in the embed FS.
3. Asserts every embedded skill/role contains the expected managed
   marker line(s) and a parseable version line.
4. Asserts the `Manifest` slice and the `Conventions` table in
   `AGENTS.md`/`CLAUDE.md` agree on which skills exist (i.e. catches
   the "AGENTS.md table lists 6 workflow skills but `skillNames` ships
   9" drift seen today).
5. Asserts every role referenced by the embedded `stage-bindings.yaml`
   is in the manifest as a `role` artifact.

This is the Concern-3 regression backstop the user asked for. It runs
on every PR — no skill/role can regress structurally without a red CI.

### 5.6 End-to-end install tests

A new `internal/kbzinit/e2e_test.go` (skipped by default unless
`KBZ_E2E=1` to keep `go test ./...` fast):

```go
func TestE2E_FreshInstall(t *testing.T)
func TestE2E_ReInstallIsIdempotent(t *testing.T)
func TestE2E_UpdateSkillsBumpsVersions(t *testing.T)
func TestE2E_SkipAgentsMD_DoesNotCreateCopilotInstructions(t *testing.T)
func TestE2E_SkipMCP_DoesNotCreateZed(t *testing.T)
func TestE2E_UnmanagedAgentsMD_PreservedWithWarning(t *testing.T)
func TestE2E_NewerMarker_NoOp(t *testing.T)
func TestE2E_PartialInstallRecovery(t *testing.T)
func TestE2E_AllManifestArtifactsPresent(t *testing.T)
```

Each test:

1. Builds the `kbz` binary into `t.TempDir()` (cached across tests via
   `sync.Once`).
2. Creates a scratch git repo via `git init` (and optionally an empty
   commit).
3. Runs `kbz init …` with controlled `--non-interactive` flags.
4. Asserts on the filesystem and the captured stdout.

A CI job (`make test-install` → `go test ./internal/kbzinit -tags=e2e
-run TestE2E_`) runs in GitHub Actions on every PR. This is the
Concern-2 backstop.

### 5.7 `kbz doctor` (or `kbz init --check`)

Reuses the same `Manifest` + `MarkerSpec` machinery to validate an
existing install in place:

- Every `Required` artifact present?
- Every managed file's marker version ≥ current?
- Stage-bindings file references only roles in the manifest?
- No "ghost" skill files (files in `.kbz/skills/` not in the manifest)?

This is what consumers run when they suspect drift, and what we run in
this repo's CI to catch corpus rot.

### 5.8 Rollback widening

`runNewProject` records every path it creates and, on failure, removes
all of them — not just `.kbz/`. Implementation: a small `tracker` value
threaded through the install steps (or `t.Cleanup`-style pattern). Fixes
D6.

### 5.9 Smaller fixes folded in

| Defect | Fix |
|---|---|
| D5 | Rename `--skip-agents-md` to `--skip-instructions` (alias the old name for one release with a deprecation warning). Split `--skip-mcp` into `--skip-mcp` and `--skip-zed`. |
| D7 | Guard the "previous init appears incomplete" warning behind `kbzExisted == true`. |
| D8 | Compute `workRoots` for the new-project branch *before* the branch decision, so first init on a repo with one commit still creates `work/`. |

## 6. Phasing

Phase 1 — **Unblock consumer install** (small PR, ship fast):
- D1 marker fix in `orchestrate-development/SKILL.md`.
- 5.3 self-healing installer.
- D7 warning guard.

Phase 2 — **Manifest + marker unification**:
- 5.1 registry.
- 5.2 marker spec.
- 5.5 build-time corpus check.
- D3 stage-bindings comparator fix.

Phase 3 — **Runtime surfaces**:
- 5.4 deliver `CLAUDE.md`, `.claude/skills/`, `OPENAI.md`,
  conditional `.cursor/rules/`.

Phase 4 — **Tests + doctor**:
- 5.6 e2e install tests + CI job.
- 5.7 `kbz doctor`.
- 5.8 widened rollback.
- 5.9 flag renames + remaining UX fixes.

Phases 1 and 2 must ship before the next public release. Phases 3 and 4
can ship in any order after Phase 2 lands.

## 7. Risks & mitigations

| Risk | Mitigation |
|---|---|
| Adding `CLAUDE.md` to consumers who already author one | Same managed-marker logic as `AGENTS.md`: unmanaged → warn-and-skip. |
| `.claude/skills/` collides with consumer skills | Always namespace under `.claude/skills/kanbanzai-*/` so collisions are visible. |
| `.cursor/rules/` install creates drift for non-Cursor users | Conditional install (only if `.cursor/` already exists, or behind `--enable-cursor`). |
| Manifest refactor touches every install file at once | Land registry behind the existing function shapes first; migrate one Kind at a time so the diff stays reviewable. |
| E2E tests slow CI | Tag them with `e2e`; run nightly + on `internal/kbzinit/**` PR diffs only. |
| Build-time corpus check rejects in-progress edits | Make the check structural (markers present, manifest consistent) rather than semantic. Content quality stays a manual concern. |

## 8. Open questions

- Should `CLAUDE.md` be installed unconditionally, or only when `.claude/`
  exists (parallel to the proposed Cursor rule)? Recommendation:
  unconditionally — it's just a ~30-line markdown file with no
  side-effects, and Claude Code is the one runtime where the absence
  causes silent agent under-context.
- Do we want `kbz doctor` as a separate subcommand, or `kbz init --check`?
  Recommendation: a separate subcommand. It's clearer in `--help` and
  keeps `init` semantically about installation.
- Should the build-time corpus check also assert that every skill's
  `triggers:` block is non-empty and that `roles:` reference real role
  files? Recommendation: yes — the cost is low and the historical
  regressions exactly match this shape.

## 9. Out-of-scope follow-ups

- **Skill content quality.** This design provides hooks for an LLM-based
  evaluator (run `kbz doctor --content` against a model) but does not
  ship one. That belongs in a future plan that builds on B25-class work.
- **Versioning policy.** This design uses the existing managed-marker
  versions verbatim. A separate policy doc should govern when integer
  counters get bumped vs. when binary semver is enough.
- **MCP host autodetection.** A future plan could detect which agent
  runtimes a consumer actually uses (presence of `.claude/`, `.cursor/`,
  `.github/copilot-instructions.md`) and tailor the install. Out of scope
  here.
