# `kanbanzai init` Command Specification

| Document | `kanbanzai init` Command Specification        |
|----------|-----------------------------------------------|
| Status   | Draft                                         |
| Created  | 2025-07-14                                    |
| Updated  | 2025-07-14                                    |
| Related  | `work/design/init-command.md`                 |
|          | `work/design/kanbanzai-1.0.md` §4, §5, §6    |
|          | `work/spec/hardening.md`                      |

---

## 1. Purpose

This document specifies the behaviour of the `kanbanzai init` command — the entry point for
setting up a repository to use the Kanbanzai workflow system. It defines exactly what the
command creates, how it detects project state, how it handles existing files, and what errors
it must produce.

`kanbanzai init` is the only command that writes to `.agents/skills/` and the only command
that creates the initial `config.yaml`. All other Kanbanzai operations assume an already-
initialised project.

---

## 2. Goals

- Provide a single, repeatable command that bootstraps a repository for Kanbanzai use.
- Install agent skill files that work across all MCP-compatible clients via the `.agents/skills/`
  convention and via `context_assemble` at runtime.
- Generate a valid, minimal `config.yaml` that satisfies the schema and is ready for immediate
  use without manual editing.
- Distinguish clearly between new projects (no prior state) and existing projects (pre-existing
  Kanbanzai config or git history), applying appropriate defaults for each case.
- Never silently overwrite or destroy files that Kanbanzai did not create.
- Be safe to re-run: running `init` again on an already-initialised project must be idempotent
  unless a version upgrade is needed.

---

## 3. Scope

### 3.1 In scope

- Creation of `.kbz/config.yaml` with a default prefix registry and document roots.
- Installation of six managed skill files under `.agents/skills/kanbanzai-*/SKILL.md`.
- Creation of `work/` subdirectories (new projects only) with `.gitkeep` placeholders.
- Detection of new vs. existing project state and branching behaviour accordingly.
- Interactive prompt for document root path on existing projects (unless `--non-interactive`
  or `--docs-path` is supplied).
- `--docs-path`, `--non-interactive`, `--update-skills`, `--skip-skills`, and
  `--skip-work-dirs` flags.
- Skill update logic: version-aware overwrite of managed files, error on unmanaged conflicts.
- All named error cases: not a git repository, no write permission, schema version mismatch,
  partial/invalid config, conflicting unmanaged skill file.

### 3.2 Deferred

- Document import (agent-assisted work, separate workflow step).
- Schema migration (handled by a separate `kanbanzai migrate` command).
- Remote configuration (GitHub integration, webhooks).
- Prefix management beyond the single default `P` prefix.

### 3.3 Explicitly excluded

- `AGENTS.md` — _superseded by `work/spec/agent-onboarding.md` §3–4: `init` now creates and manages `AGENTS.md` and `.github/copilot-instructions.md`. The exclusion below no longer applies._
- Editor-specific config files (`.cursorrules`, `.claude/skills/`, `.cursor/skills/`, etc.).
- Global or user-level skill installation.
- Running `git init` on behalf of the user.
- Importing or indexing existing markdown documents.

---

## 4. Design Principles

**Non-destructive by default.** `init` must never silently overwrite a file it did not
create. The managed-marker mechanism in skill files is the sole mechanism for distinguishing
Kanbanzai-owned files from user-owned files. Any file without a managed marker is treated as
user-owned and must not be modified.

**Idempotency.** Running `kanbanzai init` multiple times on the same project must produce the
same result as running it once, assuming the binary version has not changed. The second run
should be a no-op for up-to-date managed files and should not re-prompt the user for
information already captured in `config.yaml`.

**Minimal output surface.** `init` writes only the files listed in §5. It does not generate
`.gitignore` entries, CI configuration, README content, or any file not explicitly listed.

**New vs. existing branching.** The command behaves differently depending on whether a project
is new (no prior state) or existing (prior Kanbanzai config or git commits). New projects run
non-interactively with defaults. Existing projects pause, describe their actions, and prompt
for missing information unless flags suppress this.

**Escape hatches over silent failure.** When `init` encounters a condition that prevents it
from proceeding safely, it reports the exact file or condition causing the problem and tells
the user which flag or manual step resolves it. It does not silently skip or partially apply
changes.

---

## 5. Output File Specification

### 5.1 Complete output path list

`kanbanzai init` creates or modifies **only** the following paths:

```
.kbz/
└── config.yaml

.agents/skills/
├── kanbanzai-getting-started/SKILL.md
├── kanbanzai-workflow/SKILL.md
├── kanbanzai-planning/SKILL.md
├── kanbanzai-design/SKILL.md
├── kanbanzai-documents/SKILL.md
└── kanbanzai-agents/SKILL.md

work/                          ← new projects only
├── design/.gitkeep
├── spec/.gitkeep
├── dev/.gitkeep
├── research/.gitkeep
└── reports/.gitkeep
```

No other paths are created, modified, or deleted.

### 5.2 `.kbz/config.yaml`

`init` writes a `config.yaml` conforming to the current config schema. For a new project with
the default document layout, the canonical output is:

```yaml
version: "2"
prefixes:
  - prefix: P
    name: Plan
documents:
  roots:
    - path: work/design
      default_type: design
    - path: work/spec
      default_type: specification
    - path: work/dev
      default_type: dev-plan
    - path: work/research
      default_type: research
    - path: work/reports
      default_type: report
```

When `--docs-path` is supplied on an existing project, the `documents.roots` list is replaced
with entries derived from the provided paths. Each supplied path maps to a single root entry;
`default_type` for user-supplied paths defaults to `design` unless the path segment matches a
known type keyword (`spec`, `dev`, `research`, `reports`).

If `config.yaml` already exists and is valid, `init` does not overwrite it unless the schema
version is incompatible (see §10.3).

### 5.3 Skill files

Each skill is a directory under `.agents/skills/` containing a single `SKILL.md` file. The
file has a YAML frontmatter block followed by a markdown body.

Frontmatter format:

```yaml
---
# kanbanzai-managed: do not edit. Regenerate with: kanbanzai init --update-skills
# kanbanzai-version: <semver of binary that wrote this file>
name: kanbanzai-{name}
description: >
  [imperative description of when to activate the skill]
---
```

The comment `# kanbanzai-managed:` on its own line within the YAML frontmatter is the managed
marker. Its presence identifies the file as installed by Kanbanzai. Its absence means the file
is user-owned.

The six skill directories and their purpose:

| Directory                           | Purpose                                                  |
|-------------------------------------|----------------------------------------------------------|
| `kanbanzai-getting-started`         | Orientation: what Kanbanzai is and how to start          |
| `kanbanzai-workflow`                | Core workflow: lifecycle states, transitions, guardrails |
| `kanbanzai-planning`                | Planning stage: plans, features, decomposition           |
| `kanbanzai-design`                  | Design stage: documents, approvals, design decisions     |
| `kanbanzai-documents`               | Document registration, classification, and intelligence  |
| `kanbanzai-agents`                  | Agent delegation, context assembly, sub-agent protocol   |

### 5.4 `work/` placeholder directories

For new projects only, `init` creates the following directories, each containing an empty
`.gitkeep` file to make the directory trackable in Git:

- `work/design/`
- `work/spec/`
- `work/dev/`
- `work/research/`
- `work/reports/`

If `--skip-work-dirs` is passed, this step is skipped entirely. If any of these directories
already exist, `init` does not modify them or their contents.

### 5.5 Non-interference guarantees

- Files under `.agents/skills/` that do not match `kanbanzai-*` are never read, modified, or
  deleted.
- `AGENTS.md` at any path is never created or modified.
- Any existing markdown files in the chosen document root paths are noted in the command
  output but are not imported, indexed, or modified.

---

## 6. Project Detection and Behaviour

### 6.1 New project definition

A project is **new** if both of the following are true:

1. `.kbz/` does not exist in the current directory or any parent up to the repository root.
2. The git repository has no commits (i.e., `git log` returns an empty result).

If either condition is false, the project is treated as **existing**.

### 6.2 New project behaviour

When the project is new, `init` runs without user interaction:

1. Creates `.kbz/config.yaml` with the default layout (§5.2).
2. Creates `work/` subdirectories with `.gitkeep` files (unless `--skip-work-dirs`).
3. Installs all six skill files (unless `--skip-skills`).
4. Prints a brief summary of what was created.
5. Exits with code 0.

No prompts are shown. No questions are asked.

### 6.3 Existing project behaviour

When the project is existing, `init`:

1. Prints a description of what it intends to do (update skills, check config).
2. If `config.yaml` does not yet exist and `--docs-path` has not been supplied, prompts the
   user to specify where their workflow documents live (an interactive path selection). If
   `--non-interactive` is set and `--docs-path` is absent, `init` errors with a clear message
   instead of prompting.
3. If `config.yaml` already exists, skips the document root prompt entirely.
4. Installs or updates skill files according to the rules in §7 (unless `--skip-skills`).
5. Does **not** create `work/` subdirectories.
6. Notes any existing markdown files found in the chosen document root paths (informational
   only — no import, no modification).
7. Prints a summary of actions taken.
8. Exits with code 0.

---

## 7. Skill Update Rules

When `init` runs (or when `--update-skills` is passed), it applies the following rules to
each of the six managed skill files:

| State of `.agents/skills/kanbanzai-{name}/SKILL.md` | Action |
|------------------------------------------------------|--------|
| File does not exist | Create it with current content and managed marker. |
| File exists; has managed marker; `kanbanzai-version` matches current binary version | No action. Leave file untouched. |
| File exists; has managed marker; `kanbanzai-version` is older than current binary version | Overwrite with current version and updated `kanbanzai-version`. |
| File exists; no managed marker present | Error and stop. Print the filename and instruct the user to pass `--skip-skills` to bypass. Do not overwrite or modify. |

"Has managed marker" means the string `# kanbanzai-managed:` appears on its own line within
the YAML frontmatter block of the file.

"Version is older" means the semver in the `kanbanzai-version` comment is strictly less than
the version of the running binary, as determined by standard semver precedence rules.

The `--update-skills` flag causes `init` to perform only the skill update step, skipping
config and `work/` directory creation. All four rules above still apply.

---

## 8. Configuration Output

### 8.1 Schema version

`config.yaml` is written with `version: "2"`. This is the schema version the binary
understands. Future schema versions may add fields; the schema version in the file must never
exceed the version the running binary understands.

### 8.2 Prefix registry

The default config includes a single prefix entry:

```yaml
prefixes:
  - prefix: P
    name: Plan
```

Additional prefixes are added by the user or by other commands after `init`. The `init`
command does not add prefixes beyond this default.

### 8.3 Document roots

Document roots are set once, during `init`, and represent the directories Kanbanzai monitors
for workflow documents. The default roots map to the `work/` layout created for new projects.
For existing projects using `--docs-path`, each supplied path becomes one root entry.

`init` does not modify document roots in an existing `config.yaml`. Document root changes
after initial setup are a manual edit operation.

### 8.4 Fields not written

`init` does not write optional config fields (such as `github`, `identity`, or `worktrees`
settings). These are added by the user or by other commands when needed.

---

## 9. CLI Flags

| Flag | Type | Effect |
|------|------|--------|
| `--docs-path <path>` | String, repeatable | Set one or more document root paths. Suppresses the interactive prompt on existing projects. Each value becomes a `documents.roots` entry. |
| `--non-interactive` | Boolean | Use defaults and error instead of prompting. Requires `--docs-path` if an existing project has no `config.yaml`. |
| `--update-skills` | Boolean | Perform only the skill update step. Skip config writing and `work/` directory creation. Applies the full §7 rule table. |
| `--skip-skills` | Boolean | Do not install or update any skill files. Takes precedence over `--update-skills`. |
| `--skip-work-dirs` | Boolean | Do not create `work/` subdirectories. Has no effect on existing projects (which never receive `work/` directories). |

### 9.1 Flag interactions

- `--update-skills` and `--skip-skills` are mutually exclusive. If both are passed, `init`
  errors immediately with a clear message before taking any action.
- `--non-interactive` without `--docs-path` on an existing project that has no `config.yaml`
  is an error. `init` must not silently use an arbitrary default path.
- `--docs-path` may be repeated to supply multiple document roots. Each invocation appends
  one root. Order is preserved.

---

## 10. Error Handling

All errors must:

- Print a human-readable message explaining what went wrong.
- Identify the specific file, path, or condition causing the problem where applicable.
- Name the flag or manual step that resolves the issue where one exists.
- Exit with a non-zero exit code.
- Leave no partial state behind (i.e., if `init` fails mid-run, any files it partially wrote
  must be cleaned up before exit).

### 10.1 Not a git repository

**Condition:** The current directory (and its parents up to the filesystem root) contains no
`.git` directory.

**Message:** Explains that `kanbanzai init` must be run inside a git repository and instructs
the user to run `git init` first. `init` does not run `git init` itself.

### 10.2 No write permission

**Condition:** `init` cannot create or write a required file or directory due to OS-level
permission denial.

**Message:** Identifies the path that could not be written and suggests checking directory
permissions.

### 10.3 Existing config at newer schema version

**Condition:** `.kbz/config.yaml` exists and contains a `version` field with a value strictly
greater than the schema version the running binary understands.

**Message:** States that the config was written by a newer version of Kanbanzai, names the
config's schema version, names the binary's supported schema version, and provides the
download URL for the latest Kanbanzai release.

### 10.4 Partial or invalid config

**Condition:** `.kbz/config.yaml` exists but fails schema validation or is not valid YAML
(indicating an interrupted previous `init` or manual corruption).

**Message:** Identifies the file, describes the validation failure, and offers the user the
choice to overwrite with a fresh default config (interactive) or pass `--non-interactive` to
accept an overwrite without prompting.

### 10.5 Conflicting unmanaged skill file

**Condition:** A file at `.agents/skills/kanbanzai-{name}/SKILL.md` exists but does not
contain the managed marker `# kanbanzai-managed:` in its YAML frontmatter.

**Message:** Names the exact file path, explains that the file appears to have been created
outside of Kanbanzai and will not be overwritten, and instructs the user to pass `--skip-skills`
to skip skill installation entirely, or to manually remove or rename the conflicting file.

`init` stops after reporting the first conflicting file. It does not proceed with any other
skill installation for that run.

### 10.6 Mutually exclusive flags

**Condition:** Both `--update-skills` and `--skip-skills` are passed.

**Message:** States that these flags are mutually exclusive and cannot be combined.

### 10.7 Non-interactive without docs-path on existing project

**Condition:** `--non-interactive` is passed, the project is existing, and no `config.yaml`
exists and no `--docs-path` is supplied.

**Message:** States that `--non-interactive` requires `--docs-path` when no `config.yaml`
exists and the document root cannot be determined automatically.

---

## 11. Acceptance Criteria

The following criteria are independently testable. Each must pass for the feature to be
considered complete.

**AC-01 — Default new project output**
Given an empty git repository with no commits and no `.kbz/` directory, running
`kanbanzai init` with no flags creates `.kbz/config.yaml`, six skill files under
`.agents/skills/kanbanzai-*/SKILL.md`, and five `work/` subdirectories each containing
`.gitkeep`. No other files are created.

**AC-02 — Default config content**
The `config.yaml` produced by `init` on a new project exactly matches the canonical YAML
in §5.2: `version: "2"`, one `P` prefix, and five document roots with correct paths and
`default_type` values.

**AC-03 — Skill managed marker present**
Each installed `SKILL.md` file contains the string `# kanbanzai-managed:` on its own line
within the YAML frontmatter, and a `# kanbanzai-version:` comment set to the binary's
version.

**AC-04 — Idempotency**
Running `kanbanzai init` twice on a new project produces the same file state as running it
once. The second invocation exits with code 0 and does not modify any file.

**AC-05 — Existing project: no work/ directories**
Given a repository with existing git commits and an existing `.kbz/config.yaml`, running
`kanbanzai init` does not create any `work/` directories or `.gitkeep` files.

**AC-06 — Existing project: prompts for docs-path when config absent**
Given a repository with git commits but no `.kbz/config.yaml`, running `kanbanzai init`
without `--non-interactive` prompts the user for a document root path before writing
`config.yaml`.

**AC-07 — --docs-path suppresses prompt**
Given a repository with git commits and no `config.yaml`, running
`kanbanzai init --docs-path work/docs` creates `config.yaml` with a single document root at
`work/docs` without prompting, even when stdin is not a terminal.

**AC-08 — Skill update: no-op on version match**
Given a project where all six skill files are already installed at the current binary version,
running `kanbanzai init` does not modify any skill file (file mtimes are unchanged).

**AC-09 — Skill update: overwrite on older version**
Given a project where a skill file has a `kanbanzai-version` older than the current binary,
running `kanbanzai init` overwrites that file with the current content and updates the version
comment. Files at the current version are not touched.

**AC-10 — Skill conflict: error on unmanaged file**
Given a file at `.agents/skills/kanbanzai-workflow/SKILL.md` that does not contain the
managed marker, running `kanbanzai init` exits with a non-zero code, prints the file path,
and does not modify or overwrite the file.

**AC-11 — --skip-skills bypasses skill installation**
Running `kanbanzai init --skip-skills` on a new project creates `config.yaml` and `work/`
directories but does not create or modify any file under `.agents/skills/`.

**AC-12 — --update-skills only touches skills**
Running `kanbanzai init --update-skills` on an existing project updates managed skill files
according to §7 but does not modify `config.yaml` and does not create `work/` directories.

**AC-13 — Not a git repository error**
Running `kanbanzai init` in a directory that is not inside a git repository exits with a
non-zero code and prints a message directing the user to run `git init` first.

**AC-14 — Newer schema version error**
Given a `.kbz/config.yaml` with a `version` field higher than the binary supports, running
`kanbanzai init` exits with a non-zero code, identifies the version mismatch, and includes a
download URL for the latest release.

**AC-15 — Partial config error**
Given a `.kbz/config.yaml` that is syntactically invalid YAML, running `kanbanzai init` exits
with a non-zero code and reports the file as invalid. In interactive mode it offers the user
the option to overwrite. In `--non-interactive` mode it overwrites without prompting.

**AC-16 — Non-interference with non-kanbanzai skill files**
Given files under `.agents/skills/` whose directory names do not begin with `kanbanzai-`,
running `kanbanzai init` does not read, modify, or delete any of those files.

**AC-17 — --non-interactive + no --docs-path on existing project without config**
Running `kanbanzai init --non-interactive` on a repository with commits but no `config.yaml`
and no `--docs-path` exits with a non-zero code and a message explaining that `--docs-path`
is required in non-interactive mode.

**AC-18 — Mutually exclusive flag error**
Running `kanbanzai init --update-skills --skip-skills` exits with a non-zero code before
creating or modifying any file, with a message stating the flags are mutually exclusive.
```

Now let me register the document: