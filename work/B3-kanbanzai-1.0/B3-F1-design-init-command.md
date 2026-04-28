# `kanbanzai init` Command Design

- Status: draft design
- Purpose: define the behaviour of `kanbanzai init` — the command that prepares a repository for use with Kanbanzai
- Date: 2026-03-26T10:12:57Z
- Related:
  - `work/design/kanbanzai-1.0.md` §4, §5, §6
  - `work/design/agent-interaction-protocol.md`
  - `work/design/document-centric-interface.md`

---

## 1. Purpose

`kanbanzai init` is the entry point for every new Kanbanzai project. It prepares a repository for use with the tool: creating the workflow state directory, installing agent skills, recording document roots, and creating the recommended directory layout.

It must work equally well on a brand-new empty repository and on a mature project that has been using its own conventions for years. In both cases it must not clobber, modify, or make assumptions about existing files.

---

## 2. What `init` Produces

Running `kanbanzai init` in a git repository creates or modifies exactly these things:

```
.kbz/
└── config.yaml                        ← workflow state root and configuration

.agents/skills/
├── kanbanzai-getting-started/
│   └── SKILL.md
├── kanbanzai-workflow/
│   └── SKILL.md
├── kanbanzai-planning/
│   └── SKILL.md
├── kanbanzai-design/
│   └── SKILL.md
├── kanbanzai-documents/
│   └── SKILL.md
└── kanbanzai-agents/
    └── SKILL.md

work/                                  ← new projects only; skipped for existing
├── design/
│   └── .gitkeep
├── spec/
│   └── .gitkeep
├── dev/
│   └── .gitkeep
├── research/
│   └── .gitkeep
└── reports/
    └── .gitkeep
```

Nothing else is created or modified. In particular:

- `AGENTS.md` is not created or modified
- Existing `.agents/skills/` files not named `kanbanzai-*` are not touched
- All files outside `.kbz/` and `.agents/skills/` are untouched
- The `work/` directories are only created for new projects (see §5)

---

## 3. Skills: Dual Delivery

Kanbanzai delivers agent skills through two complementary mechanisms.

### 3.1 Filesystem (`.agents/skills/`)

Skills are installed as directories under `.agents/skills/`, each containing a `SKILL.md` file following the Agent Skills specification. This is the cross-client interoperability convention: editors and agent runtimes that support skills scan `.agents/skills/` at the project level and discover them automatically, with no per-project configuration required.

The skill directories are committed to the repository. They travel with the code, so every team member and every agent working on the project sees the same skills.

### 3.2 MCP Context Assembly

An agent connected to the Kanbanzai MCP server receives skill content directly when it calls `context_assemble`. The MCP server includes relevant skill content in the assembled context packet, filtered by role and task stage.

This bypasses filesystem discovery entirely. It means skills reach agents regardless of which editor they are using, whether that editor supports skill discovery, or whether it scans `.agents/skills/` or some other location. For MCP-connected agents — which is all agents working within Kanbanzai — the filesystem skills are a human-readable reference and a fallback for non-MCP tools. The primary delivery path is the MCP server.

### 3.3 Why Both

The filesystem skills serve humans and non-MCP tools. A team member reading the repo can open `.agents/skills/kanbanzai-workflow/SKILL.md` and understand how the workflow operates. An agent in an editor that does not use Kanbanzai's MCP server can still discover and load the skills.

The MCP delivery serves all MCP-connected agents reliably, without depending on any specific editor's skill discovery implementation.

---

## 4. The Skills Directory: `.agents/skills/`

The agent skills ecosystem is currently fragmented. Different editors scan different locations:

| Tool | Project-level skills path |
|---|---|
| Gemini CLI | `.agents/skills/` |
| Cursor | `.cursor/skills/` |
| Claude Code | `.claude/skills/` |
| Windsurf | `.windsurf/skills/` |

Kanbanzai installs to `.agents/skills/` because it is the most widely adopted cross-client convention for project-level skills and is defined as the interoperability path by the Agent Skills specification. It is not a perfect universal standard — no such thing exists yet — but it is the best single choice.

Because Kanbanzai's primary delivery mechanism is MCP context assembly (§3.2), the imperfection of filesystem discovery is an acceptable limitation. Teams using editors that do not scan `.agents/skills/` will still receive skills through the MCP server.

Kanbanzai does not generate editor-specific skill locations (e.g., `.cursor/skills/`), does not create symlinks, and does not attempt to target every known path. That approach would need to be maintained indefinitely as the ecosystem evolves. Instead, editor-specific guidance is documented in the getting-started guide and updated as the ecosystem settles.

### 4.1 Skill File Format

Each skill is a directory containing a `SKILL.md` file with YAML frontmatter and a markdown body:

```
---
name: kanbanzai-workflow
description: >
  Stage gates, lifecycle, and when to stop and ask the human.
  Use when planning or implementing any work in this project.
---

[markdown body: the skill's instructions]
```

The `name` matches the directory name. The `description` is what the model sees in the skill catalog before deciding whether to load the full content. It should be specific enough that the model activates the skill when appropriate and skips it when not.

### 4.2 Managed Marker

Every `SKILL.md` installed by Kanbanzai includes a comment at the top of the YAML frontmatter:

```
---
# kanbanzai-managed: do not edit. Regenerate with: kanbanzai init --update-skills
# kanbanzai-version: 1.0.0
name: kanbanzai-workflow
...
```

This marker is how `init` detects whether a skill file was installed by Kanbanzai (and can safely be updated) or was created by the user (and must not be touched). See §7 for update behaviour.

---

## 5. New Project vs. Existing Project

### 5.1 Detecting an Existing Project

`init` considers a project "existing" if either:
- `.kbz/` already exists, or
- The repository has commits (i.e., `git log` returns at least one entry)

An empty repository with no commits and no `.kbz/` is a new project.

### 5.2 New Project Behaviour

For a new project, `init` runs non-interactively with sensible defaults:

1. Creates `.kbz/config.yaml` with the default configuration (schema version, default prefix `P`, default document roots)
2. Installs the six skills into `.agents/skills/`
3. Creates the `work/` subdirectories with `.gitkeep` files

No prompts. No questions. The user can start working immediately.

### 5.3 Existing Project Behaviour

For an existing project, `init` pauses before creating anything and describes what it will do:

```
This project already has commits. kanbanzai init will:
  - Create .kbz/config.yaml
  - Install skills into .agents/skills/kanbanzai-*/
  - NOT create work/ directories (project already has content)
  - NOT modify any existing files

Where are your workflow documents?
  (1) work/  — recommended: keeps workflow docs separate from user-facing docs
  (2) docs/  — if you prefer to keep everything in docs/
  (3) Custom — I'll enter the paths myself
  (4) Skip   — I'll configure document roots in .kbz/config.yaml manually

Choice [1]:
```

If the user selects a path that already exists and contains markdown files, `init` notes this:

```
Found 47 markdown files in docs/. These are not imported automatically.
After init, ask your AI agent: "Help me import my existing documents."
Your agent will read the files, propose types, and register the ones you confirm.
```

This is consistent with the design principle: `init` sets up infrastructure; knowledge decisions (which documents are workflow documents) are made with agent assistance.

### 5.4 Flags for Non-Interactive Use

All interactive prompts can be bypassed with flags, making `init` suitable for scripting, CI, and automated setup:

| Flag | Effect |
|---|---|
| `--docs-path <path>` | Set a single document root (can be repeated for multiple roots) |
| `--non-interactive` | Use defaults for all prompts, error instead of prompting |
| `--update-skills` | Only update managed skill files; do not create `.kbz/` or `work/` |
| `--skip-skills` | Do not install or update skills |
| `--skip-work-dirs` | Do not create `work/` subdirectories |

---

## 6. Configuration: `config.yaml`

`init` writes a `config.yaml` that reflects the project type. For a new project using the default layout:

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

For an existing project that chose `docs/` as their root, `init` writes a minimal roots list using the paths the user specified. Any paths that do not yet exist are noted but not created.

The prefix `P` is always the default. Additional prefixes can be added by editing `config.yaml` directly, or by using `kanbanzai prefix add` (a separate command, not part of `init`).

---

## 7. Updating Skills

When `kanbanzai init` is run in a project that already has Kanbanzai skills installed, or when `kanbanzai init --update-skills` is run explicitly, the following rules apply for each skill file:

| State | Action |
|---|---|
| File does not exist | Create it |
| File exists, has managed marker, version matches current binary | No action |
| File exists, has managed marker, version is older than current binary | Overwrite with current version |
| File exists, no managed marker | Error and stop; print the filename and the `--skip-skills` flag as the escape hatch |

The third case — updating an older managed file — is the common update path. The binary version and the skill version are compared; if the binary is newer, the skill is refreshed.

The fourth case protects against accidentally overwriting a user-created file that happens to share a name with a Kanbanzai skill. The error message is explicit:

```
Error: .agents/skills/kanbanzai-workflow/SKILL.md exists but is not kanbanzai-managed.
       Refusing to overwrite. To skip skill installation, use --skip-skills.
```

---

## 8. Requirements and Error Handling

### 8.1 Git Repository Required

`init` requires the working directory to be inside a git repository. If it is not:

```
Error: not a git repository.
       Run 'git init' first, or navigate to an existing git repository.
```

`init` does not run `git init` on the user's behalf. Initialising a git repository is a meaningful action with consequences (remote configuration, branching strategy, etc.) that the user should make deliberately.

### 8.2 No Write Permission

If `init` cannot write to the project directory:

```
Error: cannot write to .kbz/ — permission denied.
       Check that you have write access to this directory.
```

### 8.3 Existing `.kbz/` at Newer Schema Version

If `.kbz/config.yaml` declares a schema version newer than the binary understands:

```
Error: .kbz/config.yaml declares schema version 3, but this binary understands
       up to version 2. Upgrade kanbanzai to continue.
       Current version: 1.0.0. Download the latest at: <release URL>
```

`init` does not attempt to operate on a schema it does not understand.

### 8.4 Partial State

If `init` is interrupted partway through (process killed, disk full, etc.), the partial state is detectable on next run: `init` checks for the presence and validity of `config.yaml` before proceeding. If `config.yaml` is present but invalid or incomplete, `init` reports the issue and offers to overwrite it.

---

## 9. Out of Scope

The following are explicitly not part of the `init` command:

- **Document import**: `init` does not scan, classify, or register existing documents. That is agent-assisted work.
- **Schema migration**: upgrading an existing `.kbz/` from one schema version to another is a separate `kanbanzai migrate` command.
- **Editor-specific configuration**: `init` does not generate `.cursorrules`, `.claude/skills/`, or any other editor-specific files. Skills are installed to `.agents/skills/` only.
- **Remote configuration**: `init` does not configure GitHub, set up webhooks, or interact with any remote service.
- **Prefix management beyond the default**: additional prefixes are added separately.
- **User-level (global) skills**: `init` installs project-level skills only. Global skill management is out of scope for 1.0.