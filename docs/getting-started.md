# Getting Started with Kanbanzai

This guide covers everything you need to go from a fresh clone to a working
human-AI development environment. It assumes you are working on **this
repository** (Kanbanzai managing itself), but the steps apply equally to any
project that adopts the system.

---

## Contents

1. [Prerequisites](#1-prerequisites)
2. [Install the binary](#2-install-the-binary)
3. [Project state: what is already here](#3-project-state-what-is-already-here)
4. [Local configuration](#4-local-configuration)
5. [Connect Zed to the MCP server](#5-connect-zed-to-the-mcp-server)
6. [Verify the server is running](#6-verify-the-server-is-running)
7. [How to work: the session loop](#7-how-to-work-the-session-loop)
8. [Context profiles](#8-context-profiles)
9. [Knowledge management](#9-knowledge-management)
10. [CLI quick reference](#10-cli-quick-reference)
11. [MCP tool reference](#11-mcp-tool-reference)
12. [Troubleshooting](#12-troubleshooting)

---

## 1. Prerequisites

| Requirement | Version | Notes |
|-------------|---------|-------|
| Go | 1.22 or later | `go version` |
| Zed | 0.224.0 or later | Needed for `tool_permissions` |
| Git | Any recent version | Used by worktree and branch tools |
| GitHub CLI / token | Optional | Required for `pr_create`, `pr_update`, `pr_status` |

---

## 2. Install the binary

From the repository root:

```sh
go install ./cmd/kanbanzai
```

This places `kanbanzai` at `~/go/bin/kanbanzai`. Confirm it works:

```sh
~/go/bin/kanbanzai version
# kanbanzai phase-4b
```

If you want `kbz` as a short alias on your PATH, add `~/go/bin` to your shell
profile and optionally symlink:

```sh
ln -s ~/go/bin/kanbanzai ~/go/bin/kbz
```

> **Rebuild after code changes.** The MCP server Zed launches is the installed
> binary, not a `go run` wrapper. Re-run `go install ./cmd/kanbanzai` whenever
> you change the source.

---

## 3. Project state: what is already here

Kanbanzai keeps all project state under `.kbz/` in the repository root. For
this repository it is already populated:

```
.kbz/
├── config.yaml          ← project settings and Plan prefix registry
├── local.yaml           ← per-machine settings, not committed (you create this)
├── state/
│   ├── plans/           ← Plan entities
│   ├── features/        ← Feature entities
│   ├── tasks/           ← Task entities
│   ├── bugs/            ← Bug entities
│   ├── decisions/       ← Decision entities
│   ├── documents/       ← Document record metadata
│   ├── knowledge/       ← Knowledge entries
│   └── worktrees/       ← Worktree tracking records
├── context/
│   └── roles/           ← Context profile YAML files
│       ├── base.yaml
│       └── developer.yaml
├── index/               ← Document intelligence index (derived, not committed)
└── cache/               ← Local derived cache (not committed)
```

Everything under `state/` and `context/roles/` is plain YAML committed to Git.
The `index/` and `cache/` directories are derived and gitignored.

### Starting a new project from scratch

If you are adopting Kanbanzai for a different project, you need to create the
`.kbz/` structure:

```sh
mkdir -p .kbz/state .kbz/context/roles
```

Then create a minimal `.kbz/config.yaml`:

```yaml
version: "2"
prefixes:
  - prefix: P
    name: Plan
```

Add the gitignore entries:

```
.kbz/cache/
.kbz/local.yaml
.kbz-tmp-*
```

Context profiles are optional but strongly recommended — see
[Section 8](#8-context-profiles).

---

## 4. Local configuration

`.kbz/local.yaml` holds per-machine settings that are **not committed** to Git.
You need to create it yourself.

### Minimum: user identity

```yaml
user:
  name: your-name
```

This is the identity used in `created_by`, `approved_by`, and similar fields.
If you omit `local.yaml`, the system falls back to `git config user.name`
automatically — so if your Git identity is already set, this step is optional.

### With GitHub integration

```yaml
user:
  name: your-name
github:
  token: ghp_xxxxxxxxxxxxxxxxxxxx
  owner: your-org-or-username
  repo: your-repo-name
```

The GitHub token is used by the `pr_create`, `pr_update`, and `pr_status` tools.
It requires `repo` scope. The token is never committed because `local.yaml` is
gitignored.

---

## 5. Connect Zed to the MCP server

A `.zed/settings.json` file is already committed to this repository. It
configures:

- The `kanbanzai` context server
- Auto-allow permissions for all core tools (no confirmation prompts per call)
- Two agent profiles: `kanbanzai-developer` and `kanbanzai-orchestrator`

The only value you may need to edit is the binary path if your `GOPATH` is not
`~/go`:

```json
"context_servers": {
  "kanbanzai": {
    "command": "/Users/samphillips/go/bin/kanbanzai",
    "args": ["serve"]
  }
}
```

Change `/Users/samphillips/go/bin/kanbanzai` to match your actual path
(`go env GOPATH` to check).

> **Why project-local settings?** The server uses `.kbz/state` as a path
> relative to its working directory. Zed launches MCP servers with the
> workspace root as cwd when using project-local settings — so opening
> `/path/to/your-project` in Zed automatically points the server at the right
> `.kbz/` directory. A global `~/Library/Application Support/Zed/settings.json`
> entry would work only if you always open this project as your root workspace.

---

## 6. Verify the server is running

Open the project in Zed. Go to the **Agent Panel** (the chat/AI sidebar) and
open its settings. You should see **kanbanzai** listed as a context server with
a green dot and the label "Server is active".

If the dot is red or yellow, check:

1. The binary path in `.zed/settings.json` is correct and the binary exists.
2. The binary was built for the current architecture (`go install ./cmd/kanbanzai`).
3. The Zed workspace root is the repository root (not a subdirectory).

You can also test the server manually:

```sh
cd /path/to/your-project
printf '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}\n' \
  | ~/go/bin/kanbanzai serve
# Should print: {"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05",...,"serverInfo":{"name":"kanbanzai","version":"phase-4b"}}}
```

---

## 7. How to work: the session loop

The intended workflow for any piece of implementation work follows this loop:

```
1. context_assemble   → get a targeted briefing for the task
2. work_queue         → see what is ready
3. dispatch_task      → claim a task
4. implement          → do the work (using Zed's built-in file tools)
5. review_task_output → self-review before handoff
6. complete_task      → mark done and unblock dependents
7. knowledge_contribute → save anything non-obvious for future sessions
8. human_checkpoint   → escalate if you need a decision
```

### Step 1 — Assemble context before you start

This is the most important step and the one most likely to be skipped. Do not
skip it. Call `context_assemble` with the role that matches the work and a
description of what you are about to do:

```
context_assemble({
  profile_id: "developer",
  task_description: "implement the canonical YAML serialisation for the Incident entity"
})
```

The tool returns a structured context packet containing:
- The role's architecture summary and conventions
- Knowledge entries relevant to the task (matched by topic)
- Fragments from linked design and spec documents
- A byte budget summary (so you know what was included vs trimmed)

This replaces reading `AGENTS.md` + grepping around + hoping you remember the
right conventions. The system has already curated what matters for this task.

### Step 2 — See the work queue

```
work_queue()
```

Returns tasks grouped by status: `queued` tasks that are ready (all
dependencies met), `active` tasks currently claimed, and blocked tasks with
their blockers listed. Pass `--conflict-check` (via the `conflict_check: true`
argument) to get a parallel-safety analysis for the top-N ready tasks.

### Step 3 — Claim a task

```
dispatch_task({ task_id: "TASK-01..." })
```

Marks the task `active`, records who claimed it and when, and creates a Git
worktree if one does not already exist for the parent feature. The response
includes the worktree path.

### Step 4 — Implement

Use Zed's built-in `edit_file`, `terminal`, `grep`, etc. tools to do the work.
The kanbanzai MCP tools are for workflow state; Zed's tools are for the code.

### Step 5 — Self-review

Before marking the task done, call `review_task_output` with the files you
changed and a summary of what you did:

```
review_task_output({
  task_id: "TASK-01...",
  output_files: ["internal/storage/entity_store.go"],
  output_summary: "Added incident field order to fieldOrderForEntityType"
})
```

If the review passes, the task transitions to `needs-review` automatically. If
it fails, you get structured findings and the task goes to `needs-rework`.

### Step 6 — Complete and unblock

```
complete_task({
  task_id: "TASK-01...",
  completion_summary: "Added incident field order per spec §11.2, all tests pass"
})
```

This marks the task `done`, records the completion summary, and automatically
transitions any tasks that were blocked on this one to `queued`. The response
includes `unblocked_tasks` — the IDs of tasks now ready to start.

### Step 7 — Contribute knowledge

If you discovered something non-obvious — an edge case, a constraint that
is not written down, a pattern that worked well — save it:

```
knowledge_contribute({
  topic: "canonical YAML serialisation",
  content: "Timestamps contain colons and must be double-quoted. The needsQuotes() function handles this, but raw string interpolation into YAML will silently break round-trips.",
  tier: 2,
  source: "TASK-01KMKA2D8ZK1Z"
})
```

Tier 3 is low-confidence (new observations). Tier 2 is confirmed (used
successfully). Tier 1 is authoritative (team-wide policy). New entries start at
tier 3; they earn trust through `knowledge_confirm` calls from future sessions.

### Step 8 — Create a checkpoint for human decisions

When you need a human decision — not just information, but an actual choice that
only the human can make — create a checkpoint rather than stopping and hoping
someone reads your output:

```
human_checkpoint({
  question: "The decomposition proposal has 23 tasks. This exceeds the max_tasks_per_feature limit of 20. Should I split the feature or raise the limit?",
  context: "Feature: FEAT-01...\nProposal: ...",
  urgency: "blocking"
})
```

The human sees it in the agent panel and can respond. The agent's session can
check `human_checkpoint_get` on the next run to see if a response is waiting.

---

## 8. Context profiles

Context profiles live in `.kbz/context/roles/`. They are YAML files that define
what an agent playing a given role needs to know. Profiles support inheritance
via `inherits:`.

### Existing profiles

**`base`** — Project-wide conventions for all agents. Covers error handling,
test conventions, YAML serialisation rules, commit format, import organisation,
and an architecture summary. Every other profile should inherit from `base`.

**`developer`** — General implementation conventions. Extends `base` with
package-specific architecture notes, line length guidelines, naming conventions,
interface design rules, and pointers to the key source files.

### Using a profile in context assembly

```
context_assemble({ profile_id: "developer", task_description: "..." })
```

The `profile_id` must match the `id:` field in the profile YAML file, not the
filename (though they are typically the same).

### Adding a new profile

Create a new YAML file in `.kbz/context/roles/`. Minimum structure:

```yaml
id: backend
inherits: developer
description: "Conventions for backend service implementation"
conventions:
  - "Use context.Context as the first parameter for any function that may block"
  - "Database operations go in the repository layer, not the service layer"
architecture:
  summary: "..."
  key_interfaces:
    - "..."
```

The `inherits` chain is resolved at assembly time — the assembled context
includes all conventions and architecture summaries from the full inheritance
chain, deduplicating where possible.

Verify the profile is valid:

```sh
kbz profile list
kbz profile get --id backend
```

---

## 9. Knowledge management

The knowledge system is the memory layer that persists across agent sessions.
Entries are stored in `.kbz/state/knowledge/` and committed to Git like
everything else.

### The confidence lifecycle

```
tier-3 (new)  →  tier-2 (confirmed)  →  tier-1 (authoritative)
                        ↓
                   deprecated / retired
```

An entry moves up when it is used successfully and confirmed. It moves to
`deprecated` when newer knowledge supersedes it, or `retired` when it is wrong.

### Key operations

| Tool | When to use it |
|------|---------------|
| `knowledge_contribute` | Record a new observation or convention |
| `knowledge_get` | Retrieve a specific entry by ID |
| `knowledge_list` | Browse entries by topic, tier, or status |
| `knowledge_confirm` | Confirm an entry was useful — increases confidence |
| `knowledge_flag` | Flag an entry as potentially wrong — decreases confidence |
| `knowledge_update` | Correct or expand an existing entry |
| `knowledge_retire` | Mark an entry as no longer valid |
| `context_report` | Get a health summary of the knowledge base |

### Good knowledge vs. bad knowledge

**Good to record:**
- Constraints that are not obvious from reading the code
- Patterns that were tried and failed (with why)
- Conventions enforced by tests but not explained in comments
- Edge cases in the serialisation or lifecycle logic

**Not worth recording:**
- Things already documented in `AGENTS.md` or a spec
- Transient implementation details that will change soon
- Observations so specific they apply to only one line of code

---

## 10. CLI quick reference

The CLI is a human convenience interface — the MCP server is the primary
interface for agents. Use the CLI for quick inspection and one-off operations
from the terminal.

### Entity operations

```sh
kbz list features
kbz list tasks
kbz list bugs
kbz get feature --id FEAT-01...
kbz create task --parent FEAT-01... --slug my-task --summary "Do the thing"
kbz update status --type task --id TASK-01... --status done
```

### Health and validation

```sh
kbz health
kbz validate --type feature --id FEAT-01... --slug my-feature --status draft
```

### Work queue

```sh
kbz queue                    # show ready tasks
kbz queue --conflict-check   # include parallel-safety analysis
```

### Feature decomposition and task review

```sh
kbz feature decompose FEAT-01...             # propose tasks (preview)
kbz feature decompose FEAT-01... --confirm   # propose then prompt for approval
kbz task review TASK-01...                   # run worker review
kbz task review TASK-01... \
  --files internal/storage/entity_store.go \
  --summary "Added incident field order"
```

### Incident management

```sh
kbz incident list
kbz incident list --status reported --severity high
kbz incident show INC-01...
kbz incident create \
  --slug api-outage \
  --title "API service outage" \
  --severity high \
  --summary "Service returning 503 for all requests" \
  --reported_by sambeau
```

### Knowledge

```sh
kbz knowledge list
kbz knowledge get --id KNW-01...
```

### Context profiles

```sh
kbz profile list
kbz profile get --id developer
kbz context assemble --profile developer --task "implement the storage layer"
```

### Git workflow

```sh
kbz worktree list
kbz worktree create --feature FEAT-01...
kbz branch status FEAT-01...
kbz merge check FEAT-01...
kbz merge execute FEAT-01...
kbz pr create FEAT-01...
kbz pr status FEAT-01...
kbz cleanup list
kbz cleanup execute FEAT-01...
```

---

## 11. MCP tool reference

The server exposes 98 tools. This reference groups them by workflow phase so you
can find the right tool without scanning a flat alphabetical list.

### Context and knowledge

| Tool | Purpose |
|------|---------|
| `context_assemble` | Assemble a targeted context packet for a task using a role profile |
| `context_report` | Get a health and coverage summary of the knowledge base |
| `profile_get` | Read a context profile and its resolved inheritance chain |
| `profile_list` | List all available context profiles |
| `knowledge_contribute` | Record a new knowledge entry |
| `knowledge_get` | Retrieve a specific knowledge entry |
| `knowledge_list` | List knowledge entries with optional filters |
| `knowledge_update` | Update an existing knowledge entry |
| `knowledge_confirm` | Confirm an entry was useful (increases confidence) |
| `knowledge_flag` | Flag an entry as potentially wrong |
| `knowledge_retire` | Mark an entry as no longer valid |
| `knowledge_promote` | Manually promote an entry to a higher tier |
| `knowledge_prune` | Remove expired entries past their TTL |
| `knowledge_compact` | Merge near-duplicate entries |
| `knowledge_check_staleness` | Report entries whose git anchors have gone stale |
| `knowledge_resolve_conflict` | Resolve a conflict between two similar entries |
| `suggest_links` | Suggest knowledge entries relevant to a given entity |
| `check_duplicates` | Check if a candidate knowledge entry duplicates an existing one |

### Work queue and dispatch

| Tool | Purpose |
|------|---------|
| `work_queue` | Show tasks grouped by status with optional conflict-check annotation |
| `dependency_status` | Check the dependency graph for a specific task |
| `dispatch_task` | Claim a task (sets to active, creates worktree if needed) |
| `complete_task` | Mark a task done, record completion summary, unblock dependents |
| `human_checkpoint` | Create a checkpoint requiring a human decision |
| `human_checkpoint_respond` | Record a human response to a checkpoint |
| `human_checkpoint_get` | Get a checkpoint and its current status |
| `human_checkpoint_list` | List all checkpoints with optional status filter |

### Entity operations

| Tool | Purpose |
|------|---------|
| `create_task` | Create a new task |
| `create_bug` | Create a new bug report |
| `create_feature` | Create a new feature |
| `create_epic` | Create a new epic |
| `create_plan` | Create a new plan |
| `record_decision` | Record an architectural or design decision |
| `get_entity` | Get a single entity by ID (all types) |
| `list_entities` | List entities of a given type |
| `list_entities_filtered` | List entities with tag and status filters |
| `update_status` | Transition an entity to a new lifecycle status |
| `update_entity` | Update non-status fields on an entity |
| `validate_candidate` | Validate a candidate entity without persisting it |
| `health_check` | Run a full project health check across all entity types |
| `rebuild_cache` | Rebuild the local derived cache from canonical state |

### Estimation

| Tool | Purpose |
|------|---------|
| `estimate_set` | Set a point estimate on a task or feature |
| `estimate_query` | Query estimates for a feature or set of tasks |
| `estimate_reference_add` | Add a reference task for estimation calibration |
| `estimate_reference_remove` | Remove a reference task |

### Feature decomposition and review

| Tool | Purpose |
|------|---------|
| `decompose_feature` | Propose a task decomposition for a feature (does not write tasks) |
| `decompose_review` | Review a decomposition proposal for gaps, cycles, and oversized tasks |
| `slice_analysis` | Analyse a feature's vertical slice structure for planning conversations |
| `review_task_output` | Run a first-pass review of a completed task against its verification criteria |

### Conflict analysis

| Tool | Purpose |
|------|---------|
| `conflict_domain_check` | Analyse parallel-work risk between two or more tasks |

### Incident management

| Tool | Purpose |
|------|---------|
| `incident_create` | Create a new incident in `reported` status |
| `incident_update` | Update an incident (status, severity, timestamps, affected features) |
| `incident_list` | List incidents with optional status and severity filters |
| `incident_link_bug` | Link a bug record to an incident |

### Document intelligence

| Tool | Purpose |
|------|---------|
| `doc_outline` | Get the structural outline of a document |
| `doc_section` | Extract a specific section from a document |
| `doc_trace` | Trace which entities reference a given document section |
| `doc_impact` | Analyse the impact of a proposed change to a document |
| `doc_gaps` | Find specification gaps — sections with no corresponding tasks |
| `doc_pending` | List documents awaiting review or approval |
| `doc_find_by_concept` | Find documents relevant to a concept or question |
| `doc_find_by_entity` | Find documents linked to a given entity |
| `doc_find_by_role` | Find documents relevant to a given agent role |
| `doc_classify` | Classify a document by type and infer metadata |
| `doc_extraction_guide` | Get guided extraction instructions for a document type |
| `doc_supersession_chain` | Trace the supersession history of a document |
| `doc_record_submit` | Submit a document for registration |
| `doc_record_approve` | Approve a registered document |
| `doc_record_get` | Get a document record by ID |
| `doc_record_get_content` | Get the full content of a registered document |
| `doc_record_list` | List registered documents |
| `doc_record_list_pending` | List documents awaiting approval |
| `doc_record_validate` | Validate a document record without persisting it |
| `doc_record_supersede` | Mark a document as superseded by a newer version |
| `batch_import_documents` | Import a directory of documents as registered records |

### Git integration

| Tool | Purpose |
|------|---------|
| `worktree_create` | Create a Git worktree for a feature or bug |
| `worktree_get` | Get the worktree record for a given entity |
| `worktree_list` | List all active worktrees |
| `worktree_remove` | Remove a worktree record after merge |
| `branch_status` | Check staleness and drift for a feature's branch |
| `merge_readiness_check` | Check whether a feature is ready to merge |
| `merge_execute` | Execute a merge with gate enforcement |
| `pr_create` | Create a GitHub pull request for a feature |
| `pr_update` | Update an open pull request |
| `pr_status` | Get the current status of a pull request |
| `cleanup_list` | List features with pending post-merge cleanup |
| `cleanup_execute` | Execute post-merge cleanup for a feature |

### Plans and config

| Tool | Purpose |
|------|---------|
| `get_plan` | Get a plan record |
| `list_plans` | List all plans |
| `update_plan` | Update plan fields |
| `update_plan_status` | Transition a plan to a new status |
| `query_plan_tasks` | Query all tasks belonging to a plan |
| `get_project_config` | Get the current project configuration |
| `get_prefix_registry` | List all registered Plan ID prefixes |
| `add_prefix` | Register a new Plan ID prefix |
| `retire_prefix` | Retire an existing prefix |
| `list_tags` | List all entity tags in use across the project |
| `list_by_tag` | List entities with a given tag |
| `migrate_phase2` | Run the Phase 2 entity migration (one-time) |

---

## 12. Troubleshooting

### Server shows as inactive in Zed

1. Check the binary path in `.zed/settings.json` matches `which kanbanzai` or
   `ls ~/go/bin/kanbanzai`.
2. Rebuild the binary: `go install ./cmd/kanbanzai`.
3. Check that the Zed workspace root is the repository root, not a subdirectory.
4. Restart the server from the Agent Panel settings (click the server name,
   then "Restart").

### Identity errors on entity creation

```
cannot resolve user identity: provide created_by explicitly, or set user.name
in .kbz/local.yaml, or configure git user.name
```

Either create `.kbz/local.yaml` with `user: { name: your-name }` or set
`git config user.name "Your Name"`.

### Tool calls fail with "entity not found"

The server looks in `.kbz/state/` relative to its working directory. If Zed
was opened from a different directory, the server cannot find the project state.
Close and reopen the project from the repository root.

### context_assemble returns very little content

This is expected for a fresh project. The context packet grows as you:
- Add more conventions to profiles
- Import design and spec documents
- Contribute knowledge entries over time

A newly bootstrapped project may return mostly profile conventions and an empty
knowledge section. That is correct — populate it through use.

### Knowledge entries not appearing in context_assemble

Entries at `tier-3` may be filtered out by the confidence threshold for the
assembled context. Use `knowledge_confirm` on entries you have validated, or
set a higher tier when contributing well-established facts:

```
knowledge_contribute({ ..., tier: 2 })
```

### Worktree creation fails

The worktree tools require the repository to be a real Git repository (not a
bare clone) and the working tree to be clean or allow stash. If the feature
branch already exists remotely, `worktree_create` will use it; if it does not,
it creates a new branch from the current HEAD.