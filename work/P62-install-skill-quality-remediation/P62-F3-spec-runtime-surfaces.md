# P62-F3 Specification — Runtime Discovery Surfaces

| Field  | Value                          |
|--------|--------------------------------|
| Feature | FEAT-01KR7BKXMK3B6 (runtime-surfaces) |
| Batch  | B64-install-skill-quality |
| Plan   | P62-install-skill-quality-remediation |
| Status | draft |
| Author | spec-author |
| Design | `work/P62-install-skill-quality-remediation/P62-design-install-skill-quality.md` (approved) §5.4 |

## Problem Statement

The P59 audit found that `CLAUDE.md`, `.claude/skills/`, `OPENAI.md`,
and `.cursor/rules/kanbanzai.mdc` exist only inside the Kanbanzai
repository. None of them are embedded into the binary or written by
`kbz init`, so a freshly-initialised consumer project receives no
runtime discovery surface for Claude Code, no `OPENAI.md` redirect for
GPT-class hosts, and no Cursor rule. For Claude Code consumers in
particular this means there is no auto-discovered entry point at all —
agents fall back to whatever the host injects.

This feature delivers those surfaces through the canonical install
manifest established in F2.

**Scope:** Embed and install `CLAUDE.md`, `.claude/skills/`,
`OPENAI.md`, and conditionally `.cursor/rules/kanbanzai.mdc`.
**Out of scope:** Changing the *content* of any of those files beyond
adding managed markers; new flag taxonomy beyond `--enable-cursor`.

## Requirements

### Functional Requirements

- **REQ-001:** `kbz init` must create `CLAUDE.md` at the repo root
  for new projects. The file must carry the standard
  `<!-- kanbanzai-managed: vN -->` marker on line 1 and use the same
  comparator and Manifest mechanism as `AGENTS.md`. Content must
  redirect to `AGENTS.md` and inventory the installed
  `.claude/skills/` wrappers.
- **REQ-002:** `kbz init` must create `OPENAI.md` at the repo root.
  Content must be a one-paragraph redirect to `AGENTS.md`. Marker on
  line 1 (HTML comment, since the file is markdown).
- **REQ-003:** `kbz init` must install all
  `.claude/skills/<name>/SKILL.md` wrappers currently shipped in this
  repo (kanbanzai-getting-started, kanbanzai-workflow, write-design,
  write-spec, implement-task, orchestrate-development, review-code)
  into the consumer's `.claude/skills/` tree. Each wrapper carries a
  marker in its YAML frontmatter (using the same `# kanbanzai-managed:`
  mechanism as workflow skills).
- **REQ-004:** Wrappers installed under REQ-003 must use the
  `.claude/skills/kanbanzai-*/` namespace prefix, where the prefix is
  applied to wrappers whose canonical target lives in
  `.agents/skills/kanbanzai-*/`. Wrappers whose canonical target lives
  in `.kbz/skills/<name>/` keep the bare name (no `kanbanzai-` prefix)
  to match the canonical target's path.
- **REQ-005:** `.cursor/rules/kanbanzai.mdc` must be installed only
  when at least one of these conditions is true:
  (a) `.cursor/` directory already exists at install time, or
  (b) the user passes `--enable-cursor`.
- **REQ-006:** A new `--enable-cursor` flag must be added to
  `kbz init` (boolean, default false). When true and `.cursor/` does
  not exist, `.cursor/rules/` is created and `kanbanzai.mdc` is
  written there.
- **REQ-007:** All four artifact kinds (claudeMd, claudeWrapper,
  openaiRedirect, cursorRule) must be `Artifact` entries in the
  Manifest from F2 and must use the `MarkerSpec` comparator from F2.
  No new ad-hoc install functions.
- **REQ-008:** All artifacts under REQ-001–REQ-006 must obey the
  unmanaged-file-skip rule: if a consumer hand-authored `CLAUDE.md`
  (no managed marker), the installer must warn and preserve the
  user's file rather than overwriting it.

### Non-Functional Requirements

- **REQ-NF-001:** The total install-time increase from these new
  artifacts must be under 50 ms on commodity hardware.
- **REQ-NF-002:** Embedded content for these surfaces must add less
  than 50 KB to the binary.

## Constraints

- Must NOT silently create `.cursor/` for users who don't use Cursor.
- Must NOT install a `CLAUDE.md` that contradicts `AGENTS.md` or any
  shipped skill — the build-time corpus check from F2 must be
  extended to verify that.
- Must use the same `--skip-instructions` flag (the renamed
  `--skip-agents-md` from F4, or the legacy name in the meantime) to
  suppress all four runtime surfaces. Do not invent per-runtime skip
  flags.
- `.cursor/rules/kanbanzai.mdc` content must remain valid Cursor MDC
  format (frontmatter with `description:` field).

## Acceptance Criteria

- **AC-001 (REQ-001):** Given a fresh `git init` repo, when
  `kbz init --non-interactive ...` runs, then `CLAUDE.md` exists at
  the repo root with `<!-- kanbanzai-managed: v` on line 1 and
  references both `AGENTS.md` and `.claude/skills/`.
- **AC-002 (REQ-002):** Given the same fresh init, when it completes,
  then `OPENAI.md` exists at the repo root and references `AGENTS.md`.
- **AC-003 (REQ-003):** Given the same fresh init, when it completes,
  then `.claude/skills/<name>/SKILL.md` exists for every wrapper in
  the Manifest, and each contains a `# kanbanzai-managed:` line in
  its YAML frontmatter.
- **AC-004 (REQ-004):** Given the same fresh init, when listing
  `.claude/skills/`, then directories `kanbanzai-getting-started/`
  and `kanbanzai-workflow/` exist (workflow-skill wrappers) alongside
  bare-named directories like `write-design/` and `review-code/`
  (task-skill wrappers).
- **AC-005 (REQ-005a):** Given a fresh `git init` repo with **no**
  `.cursor/` directory and **no** `--enable-cursor` flag, when
  `kbz init` runs, then `.cursor/` is **not** created.
- **AC-006 (REQ-005b):** Given a fresh `git init` repo where
  `.cursor/` already exists, when `kbz init` runs without
  `--enable-cursor`, then `.cursor/rules/kanbanzai.mdc` is created.
- **AC-007 (REQ-006):** Given a fresh `git init` repo with no
  `.cursor/`, when `kbz init --enable-cursor` runs, then
  `.cursor/rules/` is created and `kanbanzai.mdc` is written there.
- **AC-008 (REQ-008):** Given a pre-existing user-authored `CLAUDE.md`
  (no managed marker), when `kbz init` runs, then the file is
  preserved verbatim and stdout contains a warning naming the file
  and a recovery instruction.
- **AC-009 (REQ-008 newer):** Given a pre-existing `CLAUDE.md` with
  marker `<!-- kanbanzai-managed: v999 -->`, when `kbz init` runs,
  then the file is preserved verbatim and no warning is printed.
- **AC-010 (REQ-007):** Given the F2 `TestEmbeddedCorpus`, when the
  Manifest is missing `CLAUDE.md`, `OPENAI.md`, or any wrapper, then
  the test fails naming the missing entry.

## Verification Plan

| Criterion | Method | Description |
|---|---|---|
| AC-001 | Test | Automated: e2e test asserts file presence and marker line |
| AC-002 | Test | Automated: e2e test asserts OPENAI.md content |
| AC-003 | Test | Automated: e2e test asserts every wrapper present |
| AC-004 | Test | Automated: e2e test asserts directory naming convention |
| AC-005 | Test | Automated: e2e test, no `.cursor/` precondition |
| AC-006 | Test | Automated: e2e test, pre-create `.cursor/` |
| AC-007 | Test | Automated: e2e test with `--enable-cursor` flag |
| AC-008 | Test | Automated: e2e test pre-writes unmanaged CLAUDE.md |
| AC-009 | Test | Automated: e2e test pre-writes v999 CLAUDE.md |
| AC-010 | Test | Automated: extends `TestEmbeddedCorpus` with negative fixtures |
