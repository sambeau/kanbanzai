# P62-F1 Specification — Init Unblock

| Field  | Value                          |
|--------|--------------------------------|
| Feature | FEAT-01KR7BKXG3X61 (init-unblock) |
| Batch  | B64-install-skill-quality |
| Plan   | P62-install-skill-quality-remediation |
| Status | draft |
| Author | spec-author |
| Design | `work/P62-install-skill-quality-remediation/P62-design-install-skill-quality.md` (approved) §5.3, §5.9 (D1, D2, D7) |

## Problem Statement

`kbz init` aborts with a hard error on every second run and on every
`--update-skills` invocation. Root cause: the embedded source for
`internal/kbzinit/skills/task-execution/orchestrate-development/SKILL.md`
has no `# kanbanzai-managed:` frontmatter line, so the installer writes
a marker-less file on first run, then on the next run rejects its own
output as "not managed by Kanbanzai" and aborts before writing the
sentinel. The same vulnerability applies to any future skill whose
embedded source ships without the marker line.

A secondary UX defect causes `runExistingProject` to print
`Warning: previous init appears incomplete` on every brand-new repo
that has a single initial commit — confusing first-time users.

**Scope:** Smallest possible change that makes consumer install
succeed on the second run. Hot-fix candidate.
**Out of scope:** Manifest refactor (F2), runtime-surface delivery
(F3), e2e tests (F4), flag renames.

## Requirements

### Functional Requirements

- **REQ-001:** Every embedded skill source under
  `internal/kbzinit/skills/**/SKILL.md` must contain a
  `# kanbanzai-managed:` frontmatter line and a `# kanbanzai-version:`
  line.
- **REQ-002:** `installOneSkill` and `installOneTaskSkill` must
  guarantee the marker is present in the file written to disk, even
  when the embedded source omits it. They must inject the marker if
  missing rather than producing a marker-less file.
- **REQ-003:** Re-running `kbz init` against a previously-installed
  project must complete successfully (exit 0) and write the
  `.kbz/.init-complete` sentinel.
- **REQ-004:** `runExistingProject` must only print the
  "previous init appears incomplete" warning when the `.kbz/`
  directory was already present at the start of the run.

### Non-Functional Requirements

- **REQ-NF-001:** No public API or flag-surface changes.
- **REQ-NF-002:** Total diff under 200 lines (excluding tests).

## Constraints

- Must NOT introduce the registry from F2 — keep this PR minimal.
- Must NOT change install-time stdout for the success path beyond
  the warning fix.
- Must NOT delete the dead `internal/kbzinit/skills/orchestrate-review/SKILL.md`
  (separate cleanup) but must add the marker so it stops triggering
  the build-time check from F2 once that ships.

## Acceptance Criteria

- **AC-001 (REQ-001):** Given the embedded skill tree, when
  `grep -L '^# kanbanzai-managed:' internal/kbzinit/skills/**/SKILL.md`
  is run, then it returns no files.
- **AC-002 (REQ-002):** Given an embedded skill source with no
  `# kanbanzai-managed:` line, when `installOneSkill` writes it to
  disk, then the written file contains a
  `# kanbanzai-managed: do not edit. Regenerate with: kanbanzai init --update-skills`
  line and a `# kanbanzai-version: <binary version>` line in the
  frontmatter.
- **AC-003 (REQ-003):** Given a fresh `git init` repo, when
  `kbz init --non-interactive --name x --docs-path work` is run twice
  in succession, then both runs exit 0 and `.kbz/.init-complete`
  exists after each.
- **AC-004 (REQ-003 release build):** AC-003 must hold for a release
  binary built with `-ldflags '-X main.version=v9.9.9'`.
- **AC-005 (REQ-004):** Given a `git init` repo with no `.kbz/` and
  one initial commit, when `kbz init --non-interactive ...` is run,
  then the output does **not** contain "previous init appears
  incomplete".
- **AC-006 (REQ-004 negative):** Given a `git init` repo whose `.kbz/`
  exists but has no `.init-complete` sentinel (true partial install),
  when `kbz init` is run, then the warning **is** printed.

## Verification Plan

| Criterion | Method | Description |
|---|---|---|
| AC-001 | Test | Automated: `TestEmbeddedSkillsAllHaveMarker` walks the embed FS |
| AC-002 | Test | Automated: unit test of `transformSkillContent` with marker-less input |
| AC-003 | Test | Automated: e2e test in `init_test.go` builds binary, runs init twice |
| AC-004 | Test | Automated: same e2e test parameterised over dev and release versions |
| AC-005 | Test | Automated: e2e test asserts stdout does not contain warning string |
| AC-006 | Test | Automated: e2e test that pre-creates `.kbz/` without sentinel |
