# Kanbanzai

Kanbanzai is an MCP server that gives AI agents a structured workflow to follow — so they can pick up where the last session left off, coordinate across tasks without stepping on each other, and keep you informed without overwhelming you.

You write documents and make decisions. Agents handle the rest: decomposing work, implementing, reviewing, tracking state, and handing back to you when they need you to choose.

Everything lives in plain YAML files in your Git repo. Nothing is hidden in a database.

## Why

AI agents lose context between sessions. When two agents work in parallel, they overwrite each other's files. When an agent finishes a task, nobody knows what it did or whether the output matches the spec. The human ends up as a full-time project manager, manually tracking what's done, what's blocked, and what to do next.

Kanbanzai fixes this by giving agents the same project management structure that human teams use — plans, specs, task queues, reviews, and knowledge that persists — delivered as MCP tools they can call directly.

## What agents can do

- **See the work queue** — what's ready, what's blocked, what's in progress
- **Claim and complete tasks** — automatically unblocking dependents
- **Assemble context before starting** — role conventions, project knowledge, and relevant spec fragments, packed to fit the model's context window
- **Review their own output** — against verification criteria before handing off
- **Remember things** — knowledge entries persist across sessions and earn confidence over time
- **Manage incidents** — create, triage, link bugs, and track resolution
- **Decompose features** — propose task breakdowns from a spec document, check for gaps and cycles
- **Work in parallel safely** — conflict domain analysis flags file overlap before tasks start
- **Create pull requests and manage branches** — Git worktrees, merge gates, GitHub PR integration

## What you do

Your job is documents and decisions:

1. **Write documents** — designs, specifications, plans
2. **Review and approve** — approving a spec advances its feature automatically
3. **Respond to checkpoints** — when an agent needs a decision it can't make, it creates a checkpoint and waits. You answer; work resumes.
4. **Review completed work** — agents mark tasks `needs-review` after passing self-review. You make the final call.

You don't manage task lists or update statuses by hand. The agents do that.

## Key concepts

- **Plan** — a group of related features delivered together, like a milestone or release
- **Feature** — a unit of work that progresses through a lifecycle: design → spec → dev-plan → implementation → review
- **Task** — a single implementation step within a feature, small enough for one agent to complete
- **Checkpoint** — a point where an agent pauses and asks a human for a decision
- **Knowledge entry** — a fact or convention that persists across sessions and earns confidence as agents confirm it
- **Worktree** — an isolated Git working directory for a feature, so parallel work doesn't collide

## Getting started

### Requirements

- Go 1.25 or later
- Zed (recommended) — or any MCP client that supports stdio transport
- Git

### Install

```sh
go install github.com/sambeau/kanbanzai/cmd/kanbanzai@latest
```

Or build from source:

```sh
git clone https://github.com/sambeau/kanbanzai
cd kanbanzai
go install ./cmd/kanbanzai
```

Confirm it worked:

```sh
kanbanzai version
```

### Initialise your project

In the root of the project you want to manage:

```sh
kanbanzai init
```

This creates `.kbz/config.yaml`, installs default role and skill files, sets up work directories, writes `.gitignore` entries, and generates editor configuration. Run `kanbanzai init --help` to see flags for customising what gets installed.

If you want GitHub PR integration, create `.kbz/local.yaml` (not committed):

```yaml
user:
  name: Your Name
github:
  token: ghp_...
  owner: your-org
  repo: your-repo
```

If you skip `local.yaml`, identity falls back to `git config user.name`.

### Connect your editor

#### Zed

`kanbanzai init` writes `.zed/settings.json` with the MCP server configuration. Open the project in Zed and the kanbanzai server should appear with a green dot in the Agent Panel settings.

Add agent profiles and tool permissions to your global Zed settings (`~/Library/Application Support/Zed/settings.json` on macOS). See `docs/getting-started.md` for the full profiles to paste in.

> The server resolves `.kbz/` relative to its working directory. Zed uses the workspace root as the working directory for project-local servers — always open the project from its root, not a subdirectory.

#### Other MCP clients

The server speaks standard MCP over stdio. Point your client at:

```sh
kanbanzai serve
```

### Check everything is healthy

```sh
kanbanzai health
```

On a fresh project this should pass cleanly. On an existing project it flags dangling references, stale worktrees, and incidents missing an RCA.

See `docs/getting-started.md` for the full setup guide, including importing existing design documents, customising role profiles, and troubleshooting.

## Project structure

```
.kbz/
├── config.yaml            ← project settings
├── local.yaml             ← your machine-local settings (not committed)
├── stage-bindings.yaml    ← maps workflow stages to roles and skills
├── roles/                 ← role definitions (identity, vocabulary, anti-patterns)
├── skills/                ← skill procedures (checklists, evaluation criteria)
├── state/
│   ├── plans/             ← plans
│   ├── features/          ← features
│   ├── tasks/             ← tasks
│   ├── bugs/              ← bugs
│   ├── checkpoints/       ← human decision checkpoints
│   ├── decisions/         ← architectural decisions
│   ├── documents/         ← document metadata
│   ├── knowledge/         ← knowledge entries
│   └── worktrees/         ← active worktree records
├── context/
│   └── roles/             ← context profiles for agents
├── index/                 ← document intelligence index (derived)
├── logs/                  ← server logs (not committed)
└── cache/                 ← local cache (not committed)
```

Everything in `state/`, `roles/`, and `skills/` is plain YAML committed to Git. You can read it, diff it, and review it in a pull request like any other file.

## Further reading

- `docs/getting-started.md` — full setup guide, all 22 MCP tools, CLI reference, troubleshooting
- `AGENTS.md` — instructions for AI agents working on this project
- `work/design/workflow-design-basis.md` — the design vision