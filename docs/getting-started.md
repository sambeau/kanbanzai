# Getting Started with Kanbanzai

This guide takes you from installation to a completed feature. Every concept is introduced through a concrete action — by the end, you will have moved an idea through design, specification, implementation, and review using the full workflow.

> **Already familiar with the concepts?** The [User Guide](user-guide.md) gives a conceptual overview; the [Workflow Overview](workflow-overview.md) explains the stage-gate model in depth.

---

## What you will build

You will create a small feature for a hypothetical CLI tool: a `greet` subcommand that takes a name and prints a personalised greeting. The feature is deliberately simple — the point is the workflow, not the code.

Along the way you will:

- Create a **plan** and a **feature**
- Write a **design document** and have it approved
- Produce a **specification** with acceptance criteria
- Generate a **dev plan** that decomposes the feature into tasks
- **Implement** the code by claiming and completing tasks
- **Review** the work and **merge** it

This mirrors the real workflow for any feature, whether it is a one-file change or a multi-service refactor.

---

## Install Kanbanzai

### From source

Kanbanzai is a Go project. With Go 1.25 or later installed, run:

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

## Initialise a project

With Kanbanzai installed, navigate to a Git repository and run:

```sh
kanbanzai init
```

This must be run inside an existing Git repository. If you do not have one yet, run `git init` first.

`kanbanzai init` creates the following structure:

```
your-repo/
├── AGENTS.md                    # Agent instructions: workflow rules and skill pointers
├── .mcp.json                    # MCP server configuration (auto-detected by most editors)
├── .zed/
│   └── settings.json            # Zed context server configuration
├── .github/
│   └── copilot-instructions.md  # GitHub Copilot instructions
├── .kbz/
│   ├── config.yaml              # Project configuration
│   ├── state/                   # Entity storage (created when you first add an entity)
│   └── context/
│       └── roles/               # Context profiles
├── .agents/
│   └── skills/                  # Kanbanzai skill files for AI agents
└── work/
    ├── README.md                # Directory map
    ├── design/                  # Architecture decisions, vision, policies
    ├── spec/                    # Acceptance criteria and binding contracts
    ├── plan/                    # Project planning: roadmaps, scope, decision logs
    ├── dev/                     # Feature implementation plans and task breakdowns
    ├── research/                # Analysis, exploration, background reading
    ├── report/                  # Audit reports, post-mortems, general reports
    ├── review/                  # Feature and plan review reports
    └── retro/                   # Retrospective synthesis documents
```

`kanbanzai init` is idempotent — running it again updates skill files and managed configuration to the current version without overwriting your changes.

**Agent orientation.** `AGENTS.md` tells AI agents to use Kanbanzai MCP tools, follow the stage gates, and where to find the skill files. `.github/copilot-instructions.md` does the same for GitHub Copilot. Both use a managed marker so future `kbz init` runs can update them safely. Use `--skip-agents-md` to suppress writing both files.

**Existing repositories.** If you run `kanbanzai init` in a repository that already has commits but no `.kbz/` directory, it behaves as a first-time init and creates all the files shown above.

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

## Connect your editor

Kanbanzai runs as an MCP server that your editor's AI assistant calls directly. `kbz init` writes `.mcp.json` at the project root — most editors (Claude Code, VS Code with Copilot/Claude extensions, Cursor) read this file automatically and start the server when you open the project. For Zed, `kbz init` also writes `.zed/settings.json`.

If automatic configuration does not work for your setup, use the manual snippets below.

### Zed

`kbz init` writes `.zed/settings.json` automatically. To configure manually, add the following to your project-local `.zed/settings.json` or to your global Zed settings:

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

For **GitHub Copilot** (built-in MCP support), `.mcp.json` is read automatically. To configure manually, add to `.vscode/settings.json`:

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

`.mcp.json` is read automatically by Cursor. To configure manually, add a server through Settings → MCP Servers → Add:

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

## Create a plan

With the server running, you interact with Kanbanzai by asking your editor's AI assistant to call tools. Start by creating a **plan** — the top-level container that groups related features.

> Create a plan for my CLI tool improvements.

The assistant calls the `entity` tool:

```
entity(action: "create", type: "plan", prefix: "P",
       slug: "cli-improvements", name: "CLI Improvements",
       summary: "Add user-facing CLI subcommands")
```

This creates a plan with an ID like **P1-cli-improvements**. You will use this ID when creating features under it.

**What is a plan?** A plan is a container for related features — think of it as a project milestone or a themed batch of work. Plans progress through their own lifecycle (proposed → designing → active → reviewing → done) as the features within them advance. See [Workflow Overview — Plans](workflow-overview.md) for the full lifecycle.

---

## Create a feature

A plan contains features — each feature represents a distinct piece of work. Create one for the `greet` command:

> Create a feature for adding a greet subcommand.

```
entity(action: "create", type: "feature",
       parent: "P1-cli-improvements",
       slug: "greet-command",
       name: "Greet Command",
       summary: "Add a greet subcommand that takes a name argument and prints a personalised greeting")
```

The feature starts in **proposed** status. Before any code is written, it must pass through design and specification — this is the structure that prevents wasted implementation effort.

---

## Write a design document

The design phase captures *how* you intend to solve the problem before committing to detailed requirements.

> Write a design for the greet command feature.

The assistant creates `work/design/greet-command.md` covering the approach, alternatives considered, and key decisions. A simple design might look like:

```markdown
# Greet Command Design

## Overview
Add a `greet` subcommand to the CLI that accepts a `--name` flag
and prints a personalised greeting to stdout.

## Approach
- Register a new cobra command under the root command
- Accept `--name` as a required string flag
- Default to "World" when no name is provided
- Print "Hello, {name}!" to stdout

## Alternatives Considered
- Interactive prompt for the name — rejected, not composable in scripts
- Positional argument instead of flag — rejected, flags are more explicit
```

After writing the document, the assistant registers it:

```
doc(action: "register", path: "work/design/greet-command.md",
    type: "design", title: "Greet Command Design",
    owner: "FEAT-xxxxx")
```

---

## Approve the design

Read the design and decide whether the approach is sound. This is the first **stage gate** — your approval advances the feature from designing into specification.

> Approve the design.

```
doc(action: "approve", id: "FEAT-xxxxx/design-greet-command")
```

Stage gates are where human judgement shapes the project: you decide whether the approach is right before detailed requirements are written. See [Workflow Overview](workflow-overview.md) for how gates govern every transition.

---

## Write a specification

With the design approved, the next phase produces a specification — the binding contract that defines *what* the implementation must do.

> Write a spec for the greet command.

The assistant produces `work/spec/greet-command.md` with formal acceptance criteria:

```markdown
# Greet Command Specification

## Purpose
Define the acceptance criteria for the greet subcommand.

## Acceptance Criteria

**AC-01.** Running `mycli greet --name Alice` prints "Hello, Alice!" to stdout
and exits with code 0.

**AC-02.** Running `mycli greet` without --name prints "Hello, World!" and
exits with code 0.

**AC-03.** The greet subcommand appears in `mycli --help` output with a
one-line description.

**AC-04.** Running `mycli greet --name ""` (empty string) prints "Hello, World!"
— empty strings are treated as absent.
```

The assistant registers the spec, and you approve it — the same register-then-approve pattern as the design.

**Why a separate spec?** The design says *how*; the spec says *what*. Designs can be exploratory and conversational. Specifications are precise and testable — each acceptance criterion becomes a verification target during implementation and review.

---

## Decompose into tasks

The dev plan breaks the specification into implementable tasks.

> Create a dev plan for the greet command.

The assistant writes `work/dev/greet-command.md` describing the task breakdown, then uses the `decompose` tool to create task entities:

```
decompose(action: "propose", feature_id: "FEAT-xxxxx")
```

For a small feature like this, the decomposition might produce three tasks:

| Task | Summary |
|------|---------|
| **Register greet command** | Add the cobra command, wire it into the root command, add the `--name` flag |
| **Implement greet logic** | Handle the name argument, empty-string fallback, print output |
| **Add greet tests** | Unit tests covering AC-01 through AC-04 |

After reviewing the proposed breakdown:

```
decompose(action: "apply", feature_id: "FEAT-xxxxx", proposal: {...})
```

This creates the task entities with dependencies — the feature is now ready for implementation.

---

## Implement through the work queue

With tasks created, implementation happens through the **work queue**. The queue sorts tasks by estimate (smallest first), then by age (oldest first). Check what is ready:

> What's next?

```
next()
```

The work queue shows your tasks sorted by priority. The assistant claims the first ready task:

```
next(id: "TASK-xxxxx")
```

Claiming a task transitions it from **ready** to **active** and assembles context — the relevant spec sections, knowledge entries, and file paths the agent needs to do the work. The assistant then writes the code, runs the tests, and marks the task complete:

```
finish(task_id: "TASK-xxxxx",
       summary: "Registered greet cobra command with --name flag",
       verification: "go test ./cmd/... — all passing")
```

The `finish` tool transitions the task to **done** and contributes any knowledge gained during implementation. When one task completes, its dependents automatically become ready — the next `next()` call picks up the next task in the queue.

Repeat this cycle — claim, implement, verify, finish — until all tasks are done.

**Parallel work.** For larger features with independent tasks, multiple agents can work simultaneously. Kanbanzai's [conflict analysis](orchestration-and-knowledge.md) checks whether tasks risk editing the same files before you dispatch them in parallel.

---

## Review and merge

With all tasks complete, the feature moves to review.

> Review the greet command feature.

The assistant checks the implementation against the specification's acceptance criteria, verifies test coverage, and produces a review document. Reviews can flag issues at several severity levels — see [Workflow Overview](workflow-overview.md) for how the review process works.

Once the review passes:

> Merge the greet command feature.

```
merge(action: "check", entity_id: "FEAT-xxxxx")
merge(action: "execute", entity_id: "FEAT-xxxxx")
```

The merge tool checks seven gates — entity done, all tasks complete, verification exists and passed, no conflicts, health check clean, and branch not stale — then performs the merge. The feature transitions to **done**.

---

## What just happened

You moved a single idea — "add a greet command" — through a structured workflow:

1. **Plan** organised the work into a named container
2. **Design** captured the approach and key decisions
3. **Specification** defined precise, testable acceptance criteria
4. **Dev plan** decomposed the spec into implementable tasks
5. **Implementation** happened through a prioritised work queue
6. **Review** verified the implementation against the spec
7. **Merge** landed the work after passing all quality gates

Each transition required an explicit approval or gate check. The same workflow handles a three-task CLI command and a fifty-task architectural refactor — what changes is the size of the documents and the number of tasks, not the process. The AI agent handles the mechanical work (writing drafts, decomposing specs, claiming tasks); you make the judgement calls (approving designs, reviewing specs, deciding what to build).

All of this state — plans, features, tasks, documents, knowledge — lives in `.kbz/` inside your Git repository. There is no external database. Clone the repo and you have the full project history, including every decision and approval.

---

## Next steps

You have seen the end-to-end workflow. Here is where to go deeper:

- **[User Guide](user-guide.md)** — Conceptual overview of every entity type (plans, features, tasks, bugs, decisions), the document system, and the knowledge base.
- **[Workflow Overview](workflow-overview.md)** — The stage-gate model in detail: what happens at each phase, who decides what, and how approvals govern transitions.
- **[Orchestration and Knowledge](orchestration-and-knowledge.md)** — How agents receive context, work in parallel, and contribute knowledge that compounds across sessions.
- **[Retrospectives](retrospectives.md)** — How to capture process observations and synthesise them into actionable patterns.
- **[Schema Reference](schema-reference.md)** — Detailed structure of every entity type and their lifecycle states.
- **[MCP Tool Reference](mcp-tool-reference.md)** — The full list of available tools with their parameters and behaviour.
- **[Configuration Reference](configuration-reference.md)** — All settings in `config.yaml` and `local.yaml`.