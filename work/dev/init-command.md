# `kanbanzai init` Command: Feature Dev-Plan

| Document    | init-command dev-plan                          |
|-------------|------------------------------------------------|
| Feature     | FEAT-01KMKRQRRX3CC                             |
| Spec        | work/spec/init-command.md                      |
| Status      | Draft                                          |
| Depends on  | FEAT-01KMKRQSD1TKK (skills-content) — hard dep |

---

## 1. Overview

This dev-plan decomposes the `kanbanzai init` command into implementable tasks. The command
sets up a new or existing project by writing `.kbz/config.yaml`, installing skill files into
`.agents/skills/`, and creating `work/` placeholder directories.

**Hard dependency:** the skill file content (authored in FEAT-01KMKRQSD1TKK) must be
complete before the skill embedding task (T-04) can be implemented. The command skeleton and
config generation (T-01 through T-03) can proceed in parallel with skills authoring.

---

## 2. Task Breakdown

### T-01 — Command skeleton and flag parsing

Implement the `kanbanzai init` CLI subcommand with:
- Flag definitions: `--docs-path <path>`, `--skip-skills`, `--update-skills`,
  `--non-interactive`
- `--help` output listing all flags with descriptions and a usage example
- Mutually exclusive flag validation: `--skip-skills` and `--update-skills` cannot be used
  together (exit non-zero before creating any files)
- Top-level `--help` output listing `init` with a one-line description

**Acceptance criteria covered:** AC-18 (mutually exclusive flags), AC-09 (top-level help
partial), AC-10 (per-command help)

---

### T-02 — Git repository detection and new-project config generation

Implement:
- Git repository detection (walk up directory tree looking for `.git`; error if not found)
- New project detection: `.kbz/config.yaml` absent
- Write `.kbz/config.yaml` for a new project matching §5.2 exactly: `version: "2"`, prefix
  `P/Plan`, five document roots
- Create `work/` placeholder directories with `.gitkeep` for new projects only

**Acceptance criteria covered:** AC-01 (default new project output), AC-02 (default config
content), AC-13 (not a git repo error)

---

### T-03 — Existing project detection and interactive docs-path prompt

Implement:
- Existing project detection: `.kbz/config.yaml` present
- New-project-with-no-config behaviour: prompt user for a document root path in interactive
  mode; use `--docs-path` if provided; error in `--non-interactive` without `--docs-path`
- Existing project with config: skip `work/` directory creation; update config if needed
- Newer schema version guard: detect `version` field higher than binary supports, exit
  non-zero with download URL

**Acceptance criteria covered:** AC-05 (existing project: no work/ dirs), AC-06 (prompts for
docs-path), AC-07 (--docs-path suppresses prompt), AC-14 (newer schema version error),
AC-15 (partial/invalid config), AC-17 (--non-interactive without --docs-path)

---

### T-04 — Skill file embedding and installation

**Depends on FEAT-01KMKRQSD1TKK being complete.**

Implement:
- Embed all six skill files into the binary using `go:embed`
- Write each skill file to `.agents/skills/kanbanzai-{name}/SKILL.md`
- Include YAML frontmatter with `kanbanzai-managed:` marker and `kanbanzai-version:` set to
  the binary version
- Non-interference: do not touch `.agents/skills/` directories whose names do not start with
  `kanbanzai-`
- `--skip-skills` bypasses all skill file writes

**Acceptance criteria covered:** AC-03 (managed marker), AC-11 (--skip-skills), AC-16
(non-interference with non-kanbanzai skills)

---

### T-05 — Skill update logic

Implement:
- Version comparison: read `kanbanzai-version` from existing skill file frontmatter
- No-op if version matches current binary (do not modify file, do not change mtime)
- Overwrite if installed version is older; update version comment
- `--update-skills`: update skills only, skip config and `work/` directory steps
- Conflicting unmanaged skill: file at `kanbanzai-*` path lacks managed marker → exit
  non-zero, name the file, do not modify it

**Acceptance criteria covered:** AC-08 (no-op on version match), AC-09 (overwrite on older
version), AC-10 (skill conflict error), AC-12 (--update-skills only touches skills)

---

### T-06 — Atomicity, idempotency, and integration tests

Implement:
- Atomicity: write all files to a temp directory, then rename atomically; clean up temp
  directory on any failure path
- Idempotency: running `init` twice yields the same file state as running once; second
  invocation exits 0 without modifying any file
- Write integration tests covering all 18 acceptance criteria in the spec

**Acceptance criteria covered:** AC-04 (idempotency), and verifies all other ACs via tests

---

## 3. Implementation Order

```
T-01  command skeleton + flags          (no deps, start immediately)
T-02  git detection + config gen        (no deps, start immediately)
T-03  existing project + prompts        (after T-02)
T-04  skill embedding + install         (after skills-content FEAT-01KMKRQSD1TKK is done)
T-05  skill update logic                (after T-04)
T-06  atomicity + idempotency + tests   (after T-03, T-05)
```

T-01 and T-02 can run in parallel. T-04 is gated on the skills-content feature completing.

---

## 4. Notes

- The `.kbz/config.yaml` produced by `init` must pass the same schema validation used by all
  other kanbanzai operations; reuse the existing config write path where possible.
- Do not embed the AGENTS.md or any non-skill file content — only the six SKILL.md files.
- The `kanbanzai-version` frontmatter value must match the string returned by
  `kanbanzai --version` so that update checks are consistent.
- Partial-state recovery (`.kbz/.init-complete` sentinel) is specified in the hardening
  feature (FEAT-01KMKRQWF0FCH); do not implement it here. The init command should create the
  sentinel as its final act, but detection logic lives in hardening.