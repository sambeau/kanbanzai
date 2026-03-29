# MCP Connection Specification

| Document | MCP Connection Specification                                        |
|----------|---------------------------------------------------------------------|
| Status   | Draft                                                               |
| Feature  | FEAT-01KMWJ3ZQ57ZY                                                  |
| Related  | `work/design/fresh-install-experience.md` §5.1, FI-D-001, FI-D-007 |
|          | `work/spec/init-command.md`                                         |
|          | `work/spec/skills-content.md`                                       |

---

## 1. Purpose

This specification defines the acceptance criteria for Feature A (MCP Connection) of the
P11 Fresh Install Experience plan. It covers three coordinated changes:

1. `kbz init` writes `.mcp.json` so that MCP-compatible editors (Claude Code, Cursor, VS Code)
   automatically connect the kanbanzai server when a contributor opens the project.
2. `kbz init` writes `.zed/settings.json` when a `.zed/` directory is already present, so Zed
   users get the context server registered without manual configuration.
3. The `kanbanzai-getting-started` skill is updated to be self-identifying and to include a
   Preflight Check section that diagnoses a disconnected server before any work begins.

These changes ensure that any contributor who clones a kanbanzai-initialised repository gets a
working MCP connection with no manual steps beyond having the binary on `PATH`.

---

## 2. Scope

### 2.1 In scope

- `.mcp.json` creation and version-aware update logic in `kbz init`
- `.zed/settings.json` creation and update logic in `kbz init` (conditional on `.zed/` presence)
- `--skip-mcp` flag on `kbz init`
- `_managed` marker format shared by both files
- `kanbanzai-getting-started` skill: `description` field and Preflight Check section

### 2.2 Out of scope

- Editor-specific configuration for editors other than the MCP standard (Claude Code, Cursor,
  VS Code) and Zed
- Automatic server restart or health-check tooling
- Changes to any other skill files
- Changes to `kbz init` behaviour not related to the above (covered by `work/spec/init-command.md`)

---

## 3. Managed Marker Format

Both `.mcp.json` and `.zed/settings.json` are JSON files that carry a top-level `"_managed"`
object. This object allows `kbz init` to distinguish files it owns from files created by the
user, and to detect when an update is required.

The `"_managed"` object has exactly two fields:

```json
"_managed": {
  "tool": "kanbanzai",
  "version": 1
}
```

- `"tool"` — always the string `"kanbanzai"`.
- `"version"` — a positive integer. The current schema version is `1`. Future changes to the
  file's structure or command arguments increment this value.

A file is considered **managed** if it contains `"_managed"` with `"tool": "kanbanzai"`.
A file without the `"_managed"` key, or with `"tool"` set to a different value, is treated as
**unmanaged**.

---

## 4. Acceptance Criteria

### 4.1 `.mcp.json`

**AC-01.** `kbz init` on a new project (no existing `.mcp.json`) creates `.mcp.json` at the
project root.

**AC-02.** The created `.mcp.json` has exactly the following structure (field order may vary;
JSON whitespace is two-space indented):

```json
{
  "_managed": {
    "tool": "kanbanzai",
    "version": 1
  },
  "mcpServers": {
    "kanbanzai": {
      "command": "kanbanzai",
      "args": ["serve"]
    }
  }
}
```

No other top-level keys are present.

**AC-03.** The `command` field is the bare string `"kanbanzai"` — PATH-relative, containing no
absolute path, username, home directory reference, or machine-specific content. The file is safe
to commit to the repository as-is.

**AC-04.** When `kbz init` runs and `.mcp.json` already exists **without** the `"_managed"` key
(or with `"_managed.tool"` set to a value other than `"kanbanzai"`), init does not modify the
file and prints a warning. The warning names the file (`.mcp.json`) and instructs the user to
add the server entry manually.

**AC-05.** When `kbz init` runs and `.mcp.json` already exists with the managed marker at a
version **less than** the current embedded version, init overwrites the file with the current
content (equivalent to AC-02).

**AC-06.** When `kbz init` runs and `.mcp.json` already exists with the managed marker at the
**current** version, init takes no action on `.mcp.json`. No file is written and no message is
printed for this file.

### 4.2 Zed support

**AC-07.** When a `.zed/` directory exists at the project root at the time `kbz init` runs,
init creates `.zed/settings.json` with a `context_servers.kanbanzai` entry.

**AC-08.** The created `.zed/settings.json` has exactly the following structure:

```json
{
  "_managed": {
    "tool": "kanbanzai",
    "version": 1
  },
  "context_servers": {
    "kanbanzai": {
      "command": {
        "path": "kanbanzai",
        "args": ["serve"]
      }
    }
  }
}
```

No other top-level keys are present.

**AC-09.** When `.zed/` does not exist at the project root, `kbz init` does not create the
`.zed/` directory and does not create `.zed/settings.json`. No warning or message is emitted
for the absent directory.

**AC-10.** When `kbz init` runs and `.zed/settings.json` already exists **without** the
`"_managed"` key (or with `"_managed.tool"` set to a value other than `"kanbanzai"`), init
does not modify the file and prints a note. The note names the file (`.zed/settings.json`) and
directs the user to `docs/getting-started.md` for the snippet to add manually.

**AC-11.** When `kbz init` runs and `.zed/settings.json` already exists with the managed marker
at a version **less than** the current embedded version, init overwrites the file with the
current content (equivalent to AC-08).

**AC-12.** When `kbz init` runs and `.zed/settings.json` already exists with the managed marker
at the **current** version, init takes no action on `.zed/settings.json`.

### 4.3 `--skip-mcp` flag

**AC-13.** `kbz init` accepts a `--skip-mcp` boolean flag. When present, init does not create
or modify `.mcp.json` regardless of whether the file already exists.

**AC-14.** When `--skip-mcp` is present, init does not create or modify `.zed/settings.json`
regardless of whether `.zed/` or `.zed/settings.json` exists.

**AC-15.** `--skip-mcp` is independent of all other flags. It may be combined with
`--skip-skills`, `--skip-roles`, and `--non-interactive` without conflict.

**AC-16.** `--skip-mcp` does not suppress the creation of any other file produced by `kbz init`
(config, skills, roles, work directories). Its effect is limited to `.mcp.json` and
`.zed/settings.json`.

### 4.4 `kanbanzai-getting-started` skill

**AC-17.** The YAML frontmatter `description` field of the `kanbanzai-getting-started` skill
starts with the text: `This repository is managed with Kanbanzai`. The description does not
contain conditional language such as "Use at the start of any agent session in a
Kanbanzai-managed project" — the fact is stated directly, not as a precondition the agent must
already know.

**AC-18.** The `kanbanzai-getting-started` skill body contains a section titled
**Preflight Check** (or `## Preflight Check` / `### Preflight Check`). This section is the
first substantive section of the skill body, appearing before any section that directs the
agent to call kanbanzai tools or perform workflow actions.

**AC-19.** The Preflight Check section explains that kanbanzai operates through MCP tools and
instructs the agent to confirm the kanbanzai MCP server is connected before taking any action.
It identifies `.mcp.json` as the file that configures the server in most editors and references
`docs/getting-started.md` for manual setup.

**AC-20.** The Preflight Check section explicitly instructs the agent **not** to substitute
`grep`, `find`, or equivalent file-system searches for kanbanzai tool calls when those tools
are unavailable. The agent must treat unavailable kanbanzai tools as a setup problem to
diagnose, not a signal to fall back to manual file search.

---

## 5. Verification

| Criterion | Method |
|-----------|--------|
| AC-01 – AC-03 | `TestInit_WritesMcpJson`: call `init` on a temp dir with no `.mcp.json`; assert file exists and content matches expected JSON. |
| AC-04 | `TestInit_UnmanagedMcpJson_Skips`: call `init` with a pre-existing `.mcp.json` that has no `_managed` key; assert file is unchanged and stderr contains `.mcp.json`. |
| AC-05 | `TestInit_ManagedMcpJson_OlderVersion_Overwrites`: pre-populate `.mcp.json` with `"version": 0`; assert overwritten after init. |
| AC-06 | `TestInit_ManagedMcpJson_CurrentVersion_NoOp`: pre-populate `.mcp.json` at current version; assert file unchanged and no write performed. |
| AC-07 – AC-08 | `TestInit_ZedDir_WritesSettingsJson`: create `.zed/` in temp dir; call `init`; assert `.zed/settings.json` exists and content matches expected JSON. |
| AC-09 | `TestInit_NoZedDir_NoSettingsJson`: call `init` on a temp dir with no `.zed/`; assert `.zed/settings.json` does not exist and `.zed/` was not created. |
| AC-10 | `TestInit_UnmanagedZedSettings_Skips`: pre-populate `.zed/settings.json` with no `_managed` key; assert file unchanged and stdout/stderr names the file. |
| AC-11 – AC-12 | Analogous to AC-05/06 tests for `.zed/settings.json`. |
| AC-13 – AC-16 | `TestInit_SkipMcp`: call `init --skip-mcp` with `.zed/` present; assert neither `.mcp.json` nor `.zed/settings.json` is created; assert other init outputs (skills, config) are unaffected. |
| AC-17 | `TestGettingStartedSkill_Description`: parse embedded skill YAML; assert `description` starts with `"This repository is managed with Kanbanzai"`. |
| AC-18 – AC-20 | `TestGettingStartedSkill_PreflightSection`: parse skill body; assert a Preflight Check section exists before any section containing a kanbanzai tool call; assert section body contains reference to `.mcp.json` and explicit instruction against using `grep` or `find` as a substitute. |

All tests must pass under `go test -race ./...`.