# Getting Started with Kanbanzai

## Installation

### From source

Kanbanzai is a Go project. With Go 1.21 or later installed, run:

```sh
go install github.com/sambeau/kanbanzai/cmd/kanbanzai@latest
```

Or from a local clone:

```sh
go install ./cmd/kanbanzai
```

This places the `kanbanzai` binary at `~/go/bin/kanbanzai`. Make sure `~/go/bin` is in your `PATH`.

If you want a shorter command, create a symlink:

```sh
ln -s ~/go/bin/kanbanzai ~/go/bin/kbz
```

### Prebuilt binaries

At the 1.0 release, prebuilt binaries will be available from [GitHub Releases](https://github.com/sambeau/kanbanzai/releases) for macOS (arm64, amd64), Linux (arm64, amd64), and Windows (amd64). Until then, `go install` is the primary installation method.

### Verify the installation

```sh
kanbanzai version
```

If you get "command not found", check that `~/go/bin` is on your `PATH`.

### macOS GUI editors and PATH

macOS GUI applications (Zed, VS Code, Cursor) launch with a minimal environment that does not inherit your shell's `PATH`. If your editor cannot find the `kanbanzai` binary, install it to `/usr/local/bin`:

```sh
sudo ln -sf ~/go/bin/kanbanzai /usr/local/bin/kanbanzai
```

This is the most reliable way to ensure any editor can start the MCP server.

---

## What Kanbanzai is

Kanbanzai is a Git-native project workflow system that runs as an MCP (Model Context Protocol) server. MCP is an open protocol that lets AI-powered editors communicate with external tools through a structured interface. Kanbanzai exposes its entire feature set — plans, features, tasks, bugs, decisions, work queues, document tracking — as MCP tools that your editor's AI assistant can call directly.

All project state lives in a `.kbz/` directory inside your repository, stored as plain YAML files. Plans, features, tasks, knowledge entries, context profiles — they are all version-controlled alongside your code. There is no external database, no cloud service, no separate sync step. If you can clone the repo, you have the full project state.

The system bridges two worlds: human intent expressed through documents (designs, specifications, dev plans) and agent execution through structured tasks, work queues, and context assembly. You write a spec, decompose it into tasks, and an AI agent can pick up those tasks with full context about what to build and why. Kanbanzai is editor-agnostic — it works with any editor that supports MCP, including Zed, VS Code, Cursor, and Claude Desktop.

---

## Initialising a project

Navigate to a Git repository and run:

```sh
kanbanzai init
```

This must be run inside an existing Git repository. If you don't have one yet, run `git init` first.

`kanbanzai init` creates the following structure:

```
your-repo/
├── AGENTS.md                  # Agent instructions: workflow rules and skill pointers
├── .mcp.json                  # MCP server configuration (auto-detected by most editors)
├── .zed/
│   └── settings.json          # Zed context server configuration
├── .github/
│   └── copilot-instructions.md  # GitHub Copilot instructions (points to AGENTS.md)
├── .kbz/
│   ├── config.yaml            # Project configuration (schema version, prefixes, document roots)
│   ├── state/                 # Entity storage (features, tasks, bugs, etc.)
│   └── context/
│       └── roles/             # Context profiles (base.yaml scaffold, reviewer.yaml managed)
├── .agents/
│   └── skills/                # Kanbanzai skill files for AI agents
│       ├── kanbanzai-getting-started/
│       ├── kanbanzai-workflow/
│       ├── kanbanzai-design/
│       ├── kanbanzai-specification/
│       ├── kanbanzai-documents/
│       ├── kanbanzai-agents/
│       ├── kanbanzai-planning/
│       ├── kanbanzai-review/
│       └── kanbanzai-plan-review/
└── work/
    ├── README.md              # Directory map
    ├── design/                # Architecture decisions, vision, policies
    ├── spec/                  # Acceptance criteria and binding contracts
    ├── plan/                  # Project planning: roadmaps, scope, decision logs
    ├── dev/                   # Feature implementation plans and task breakdowns
    ├── research/              # Analysis, exploration, background reading
    ├── report/                # Audit reports, post-mortems, general reports
    ├── review/                # Feature and plan review reports
    └── retro/                 # Retrospective synthesis documents
```

`kanbanzai init` is idempotent. Running it again on an already-initialised project updates skill files and managed role files to the current version, and applies version-aware conflict logic to `.mcp.json`, `.zed/settings.json`, `AGENTS.md`, and `.github/copilot-instructions.md`. If `AGENTS.md` or `.github/copilot-instructions.md` already exist without a kanbanzai managed marker, they are left untouched and a warning is printed.

**Agent orientation:** `AGENTS.md` is read by most AI agent platforms and tells agents to use kanbanzai MCP tools, follow the stage gates, and where to find the skill files. `.github/copilot-instructions.md` serves the same purpose for GitHub Copilot. Both files use a `<!-- kanbanzai-managed: v1 -->` marker on line 1 so future `kbz init` runs can update them safely.

Use `--skip-agents-md` to suppress writing both files.

**Existing repositories:** If you run `kanbanzai init` in a repository that already has commits but has never had kanbanzai set up (no `.kbz/` directory), it behaves as a first-time init: all files listed above are created, including the `work/` directories. You will be prompted for a document root path — press Enter to use the standard `work/` layout, or type a custom path.

**macOS GUI editors and PATH:** If `kanbanzai` is not on the PATH that your editor can see, the MCP server will silently fail to start. See [macOS GUI editors and PATH](#macos-gui-editors-and-path) above.

### Local configuration

The init command does not create `.kbz/local.yaml` — this file is for per-machine settings that should not be committed. Create it manually when you need to set your identity or configure GitHub integration:

```yaml
user:
  name: Your Name

github:
  token: ghp_xxxxxxxxxxxxxxxxxxxx
  owner: your-org
  repo: your-repo
```

Add `.kbz/local.yaml` to your `.gitignore` — it contains credentials and machine-specific paths.

---

## Editor integration

`kbz init` writes `.mcp.json` at the project root. Most editors (Claude Code, VS Code with Copilot/Claude extensions, Cursor) read this file automatically and start the MCP server when you open the project. For Zed, `kbz init` also writes `.zed/settings.json`.

If the automatic configuration does not work for your setup, use the manual snippets below.

### Zed

`kbz init` writes `.zed/settings.json` automatically. If you need to configure it manually, add the following to your project-local `.zed/settings.json` or to your global Zed settings:

```json
{
  "context_servers": {
    "kanbanzai": {
      "command": "kanbanzai",
      "args": ["serve"]
    }
  }
}
```

If `kanbanzai` is not on the PATH that Zed can see, use the absolute path to the binary (see [macOS GUI editors and PATH](#macos-gui-editors-and-path) above).

**Verify:** Open Zed's agent panel and ask it to call the `health` tool. It should return a health report showing entity counts and system status.

### Claude Desktop

Add the following to your `claude_desktop_config.json`. On macOS, this file is at `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "kanbanzai": {
      "command": "kanbanzai",
      "args": ["serve"],
      "cwd": "/path/to/your/project"
    }
  }
}
```

Replace `/path/to/your/project` with the actual path to your project root. Claude Desktop requires a restart after changing this file.

**Verify:** Ask Claude "call the `health` tool" — it should return a health report.

### VS Code

For **GitHub Copilot** (built-in MCP support), `.mcp.json` is read automatically. If you need to configure manually, add to `.vscode/settings.json`:

```json
{
  "github.copilot.chat.mcpServers": {
    "kanbanzai": {
      "command": "kanbanzai",
      "args": ["serve"]
    }
  }
}
```

**Verify:** In Copilot Chat, ask the assistant to call `health`.

### Cursor

`.mcp.json` is read automatically by Cursor. If you need to configure manually, add a server through Settings → MCP Servers → Add:

```json
{
  "mcpServers": {
    "kanbanzai": {
      "command": "kanbanzai",
      "args": ["serve"],
      "cwd": "/path/to/your/project"
    }
  }
}
```

**Verify:** In agent mode, ask it to call `health`.

---

## Your first plan

Kanbanzai's primary interface is through MCP tools. You interact with it by asking your editor's AI assistant to call specific tools. Here is a walkthrough that creates a plan, a feature, a task, and then checks the work queue.

### Step 1: Create a plan

Ask your AI assistant to call `entity` with these arguments:

```
entity(action: "create", type: "plan", prefix: "P", slug: "my-first-project",
       title: "My First Project", summary: "Learning Kanbanzai")
```

This creates a plan with an ID like `P1-my-first-project`. Plans are the top-level organising unit — they group related features together.

### Step 2: Create a feature

```
entity(action: "create", type: "feature", parent: "P1-my-first-project",
       slug: "hello-world", summary: "Implement a hello world endpoint")
```

This returns a feature with an ID like `FEAT-01ABC...`. Note the full ID.

### Step 3: Create a task

```
entity(action: "create", type: "task", parent_feature: "FEAT-01ABC...",
       slug: "write-handler", summary: "Write the HTTP handler for /hello")
```

Tasks are the atomic units of work. Each task belongs to a feature.

### Step 4: Check the work queue

```
next()
```

Your task should appear in the ready queue. From here, you or an AI agent can claim the task with `next(id: "TASK-...")`, do the work, and mark it complete with `finish`.

---

## Next steps

- **[Workflow Overview](workflow-overview.md)** — The stage-gate model that governs how work flows from plan through feature through task to completion.
- **[Schema Reference](schema-reference.md)** — Detailed structure of every entity type: plans, features, tasks, bugs, decisions, and their lifecycle states.
- **[MCP Tool Reference](mcp-tool-reference.md)** — The full list of available tools with their parameters and behaviour.
- **[Configuration Reference](configuration-reference.md)** — All settings in `config.yaml` and `local.yaml`, including document roots, prefix registries, and branch tracking.