# Getting Started with Kanbanzai

## Installation

### From source

Kanbanzai is a Go project. With Go 1.25 or later installed, run:

```sh
go install ./cmd/kanbanzai
```

This places the `kanbanzai` binary at `~/go/bin/kanbanzai`. Make sure `~/go/bin` is in your `PATH`.

If you want a shorter command, create an alias:

```sh
ln -s ~/go/bin/kanbanzai ~/go/bin/kbz
```

### Prebuilt binaries

The project uses GoReleaser for binary distribution. At the 1.0 release, prebuilt binaries will be available from [GitHub Releases](https://github.com/kanbanzai/kanbanzai/releases) for:

- macOS (arm64, amd64)
- Linux (arm64, amd64)
- Windows (amd64)

Until then, `go install` is the primary installation method.

### Verify the installation

Run:

```sh
kanbanzai version
```

You should see version information printed to the terminal. If you get "command not found", check that `~/go/bin` is on your `PATH`.

---

## What Kanbanzai is

Kanbanzai is a Git-native project workflow system that runs as an MCP (Model Context Protocol) server. MCP is an open protocol that lets AI-powered editors communicate with external tools through a structured interface. Kanbanzai exposes its entire feature set — plans, features, tasks, bugs, decisions, work queues, document tracking — as MCP tools that your editor's AI assistant can call directly.

All project state lives in a `.kbz/` directory inside your repository, stored as plain YAML files. Plans, features, tasks, knowledge entries, context profiles — they are all version-controlled alongside your code. There is no external database, no cloud service, no separate sync step. If you can clone the repo, you have the full project state.

The system bridges two worlds: human intent expressed through documents (designs, specifications, dev plans) and agent execution through structured tasks, work queues, and context assembly. You write a spec, decompose it into tasks, and an AI agent can pick up those tasks with full context about what to build and why. Kanbanzai is editor-agnostic — it works with any editor that supports MCP, including Zed, VS Code, Cursor, and Claude Desktop.

---

## Editor integration

Kanbanzai runs as an MCP server. You configure your editor to start it, and then your editor's AI assistant gains access to all Kanbanzai tools. Below are setup instructions for the most common editors.

### Zed

Add the following to your project-local `.zed/settings.json`, or to your global Zed settings:

```json
{
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

**Verify:** Open Zed's agent panel and ask it to call the `health_check` tool. It should return a health report showing entity counts and system status.

### Claude Desktop

Add the following to your `claude_desktop_config.json`. On macOS, this file is typically at `~/Library/Application Support/Claude/claude_desktop_config.json`:

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

**Verify:** Ask Claude "call the health_check tool" — it should return a health report.

### VS Code

Kanbanzai works with the GitHub Copilot extension (which has built-in MCP support) or the Claude extension.

For **GitHub Copilot**, add to `.vscode/settings.json`:

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

For the **Claude extension**, add to `.vscode/settings.json`:

```json
{
  "claude.mcpServers": {
    "kanbanzai": {
      "command": "kanbanzai",
      "args": ["serve"]
    }
  }
}
```

**Verify:** In Copilot Chat or the Claude panel, ask the assistant to call `health_check`.

### Cursor

Add a server through Cursor's MCP settings (Settings → MCP Servers → Add):

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

Replace `/path/to/your/project` with the actual path to your project root.

**Verify:** In Cursor's agent mode, ask it to call `health_check`.

---

## Initialising a project

Navigate to a Git repository and run:

```sh
kanbanzai init
```

This must be run inside an existing Git repository. If you don't have one yet, run `git init` first.

The command creates the following structure:

```
your-repo/
├── .kbz/
│   ├── config.yaml            # Project configuration (schema version, prefixes, document roots)
│   ├── state/                 # Entity storage (features, tasks, bugs, etc.)
│   └── context/
│       └── roles/             # Context profiles for agent roles
├── .agents/
│   └── skills/                # Kanbanzai skill files for AI agents
├── work/
│   ├── design/                # Design documents
│   ├── spec/                  # Specification documents
│   ├── dev/                   # Development plans
│   ├── research/              # Research documents
│   └── reports/               # Report documents
```

The `config.yaml` starts with a single plan prefix `P` (for "Plan") and document roots pointing to the `work/` directories.

`kanbanzai init` is idempotent. Running it again on an already-initialised project exits successfully without modifying existing files.

### Local configuration

The init command does not create `.kbz/local.yaml` — this file is for per-machine settings that should not be committed. Create it manually when you need to set your identity or configure GitHub integration:

```yaml
user:
  name: your-name

github:
  token: ghp_xxxxxxxxxxxxxxxxxxxx
  owner: your-org-or-username
  repo: your-repo-name
```

Add `.kbz/local.yaml` to your `.gitignore` — it contains credentials and machine-specific paths.

---

## Your first plan

Kanbanzai's primary interface is through MCP tools, not the CLI. You interact with it by asking your editor's AI assistant to call specific tools. Here is a walkthrough that creates a plan, a feature, a task, and then checks the work queue.

### Step 1: Create a plan

Ask your AI assistant to call `create_plan` with these arguments:

- **prefix**: `P`
- **slug**: `my-first-project`
- **title**: `My First Project`
- **summary**: `Learning Kanbanzai`

This creates a plan with the ID `P1-my-first-project`. Plans are the top-level organising unit — they group related features together.

### Step 2: Create a feature

Call `create_feature` with:

- **parent**: `P1-my-first-project`
- **slug**: `hello-world`
- **summary**: `Implement a hello world endpoint`

This returns a feature with an ID like `FEAT-01ABC...`. Note the full ID — you need it for the next step. Features represent deliverable units of work within a plan.

### Step 3: Create a task

Call `create_task` with:

- **parent_feature**: the `FEAT-...` ID from step 2
- **slug**: `write-handler`
- **summary**: `Write the HTTP handler for /hello`

Tasks are the atomic units of work that an agent (or a human) picks up and completes. Each task belongs to a feature.

### Step 4: Check the work queue

Call `work_queue` with no arguments.

Your task should appear in the ready queue. It shows as ready because it has no dependencies blocking it. In a real project, tasks with unresolved dependencies stay queued until their blockers are completed, then automatically promote to ready.

From here, you or an AI agent can dispatch the task, do the work, and mark it complete — but that workflow is covered in the documentation linked below.

---

## Next steps

Now that you have a working Kanbanzai setup, here is where to go next:

- **[Workflow Overview](workflow-overview.md)** — The stage-gate model that governs how work flows from plan through feature through task to completion.
- **[Schema Reference](schema-reference.md)** — Detailed structure of every entity type: plans, features, tasks, bugs, decisions, and their lifecycle states.
- **[MCP Tool Reference](mcp-tool-reference.md)** — The full list of available tools with their parameters and behaviour.
- **[Configuration Reference](configuration-reference.md)** — All settings in `config.yaml` and `local.yaml`, including document roots, prefix registries, and branch tracking.