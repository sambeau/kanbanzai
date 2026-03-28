# Kanbanzai

Kanbanzai is an MCP server that gives AI agents a structured workflow to follow — so they can pick up where the last session left off, coordinate across tasks without stepping on each other, and keep you informed without overwhelming you.

You write documents and make decisions. Agents handle the rest: decomposing work, implementing, reviewing, tracking state, and handing back to you when they need a choice made.

Everything lives in plain YAML files in your Git repo. Nothing is hidden in a database.

---

## What agents can do with it

- **See the work queue** — what's ready, what's blocked, what's in progress
- **Claim and complete tasks** — with automatic unblocking of dependents
- **Assemble context before starting** — role conventions, project knowledge, and relevant spec fragments, packed to fit the model's context window
- **Review their own output** — against verification criteria before handing off
- **Remember things** — knowledge entries persist across sessions and earn confidence over time
- **Manage incidents** — create, triage, link bugs, and track resolution
- **Decompose features** — propose task breakdowns from a spec document, check for gaps and cycles
- **Work in parallel safely** — conflict domain analysis flags file overlap before tasks start
- **Create pull requests and manage branches** — Git worktrees, merge gates, GitHub PR integration

---

## Getting started

### 1. Requirements

- Go 1.25 or later
- Zed (recommended) — or any MCP client that supports stdio transport
- Git

### 2. Install

**Remote install** (easiest):

```sh
go install github.com/sambeau/kanbanzai/cmd/kanbanzai@latest
```

**From source:**

```sh
git clone https://github.com/sambeau/kanbanzai
cd kanbanzai
go install ./cmd/kanbanzai
```

Confirm it worked:

```sh
~/go/bin/kanbanzai version
# kanbanzai phase-4b
```

### 3. Initialise your project

In the root of the project you want to manage:

```sh
mkdir -p .kbz/state .kbz/context/roles
```

Create `.kbz/config.yaml`:

```yaml
version: "2"
prefixes:
  - prefix: P
    name: Plan
```

Add to `.gitignore`:

```
.kbz/cache/
.kbz/local.yaml
.kbz-tmp-*
```

Create `.kbz/local.yaml` (not committed — this is yours):

```yaml
user:
  name: Your Name
github:            # optional, only needed for PR tools
  token: ghp_...
  owner: sambeau
  repo: kanbanzai
```

If you skip `local.yaml`, identity falls back to `git config user.name` automatically.

### 4. Connect your editor

#### Zed

The setup is split across two files because Zed only allows `context_servers` in project-local settings — `agent` profiles and permissions must go in the global settings file.

**`.zed/settings.json`** (already committed to this repo — edit the binary path if your `GOPATH` differs from `~/go`):

```json
{
  "context_servers": {
    "kanbanzai": {
      "command": "/Users/you/go/bin/kanbanzai",
      "args": ["serve"]
    }
  }
}
```

**`~/Library/Application Support/Zed/settings.json`** — add an `agent` block alongside any existing entries. See `docs/getting-started.md` for the full profiles and tool permissions to paste in.

Open the project in Zed. The kanbanzai server should appear with a green dot in the Agent Panel settings. Select the **Kanbanzai Developer** or **Kanbanzai Orchestrator** profile from the profile picker to scope the tool surface for each session.

> The server resolves `.kbz/` relative to its working directory. Zed uses the workspace root as cwd for project-local servers — always open the project from its root, not a subdirectory.

#### Other MCP clients

The server speaks standard MCP over stdio. Point your client at:

```sh
kanbanzai serve
```

### 5. Create a context profile

Profiles tell agents what they need to know for a given role. Create at least a base profile at `.kbz/context/roles/base.yaml`:

```yaml
id: base
description: "Project-wide conventions for all agents"
conventions:
  - "Write your key conventions here"
  - "Error handling, test patterns, naming rules, etc."
architecture:
  summary: "One paragraph describing the project structure"
  key_interfaces:
    - "The most important files and what they do"
```

Agents call `context_assemble` with a profile ID before starting any task. The more useful information you put in profiles, the less time agents spend re-discovering your conventions.

### 6. Import your design documents

If you have existing design documents, register them:

```sh
kbz import work/design/
```

Or register a single document:

```sh
kbz create document --path work/design/my-feature.md --type design --title "My Feature Design"
```

Agents use these documents to decompose features into tasks and to review their work against specifications.

### 7. Check everything is healthy

```sh
kbz health
```

This runs a full project health check. On a fresh project it should pass cleanly. On an existing project it will flag anything that looks wrong — dangling references, stale worktrees, incidents missing an RCA, and so on.

---

## The human side of the workflow

Once the project is set up, your job is:

1. **Write documents** — designs, specifications, plans
2. **Review and approve** — approving a spec advances its feature automatically
3. **Respond to checkpoints** — when an agent needs a decision it can't make, it creates a checkpoint and waits. You answer; work resumes.
4. **Review completed work** — agents mark tasks `needs-review` after passing self-review. You make the final call.

You don't manage task lists or update statuses by hand. The agents do that. Your interface is documents and decisions.

---

## What gets stored

```
.kbz/
├── config.yaml          ← project settings
├── local.yaml           ← your machine-local settings (not committed)
├── state/
│   ├── plans/           ← plans
│   ├── features/        ← features
│   ├── tasks/           ← tasks
│   ├── bugs/            ← bugs
│   ├── decisions/       ← decisions
│   ├── documents/       ← document metadata
│   ├── knowledge/       ← knowledge entries
│   └── worktrees/       ← active worktree records
├── context/
│   └── roles/           ← context profile definitions
├── index/               ← document intelligence index (derived)
└── cache/               ← local cache (derived)
```

Everything in `state/` and `context/roles/` is plain YAML committed to Git. You can read it, diff it, and review it in a pull request like any other file.

---

## Further reading

- `docs/getting-started.md` — full setup guide, all 98 MCP tools, CLI reference, troubleshooting
- `AGENTS.md` — instructions for AI agents working on this project
- `work/design/workflow-design-basis.md` — the design vision