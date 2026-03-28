# MCP Server Discoverability and Reliability

| Document | MCP Server Discoverability and Reliability Design |
|----------|---------------------------------------------------|
| Status   | Draft                                             |
| Created  | 2026-03-28T13:54:11Z                              |
| Author   | sambeau (with AI analysis)                        |
| Related  | P7 Retrospective, 2.0 Migration Audit, Hardening Spec §10 |

---

## 1. Problem Statement

The P7 implementation retrospective exposed five failure modes in the interaction between AI agents and the Kanbanzai MCP server. The root cause in every case was the same: **the MCP server does not make it easy enough for agents to do the right thing, and does nothing to prevent them doing the wrong thing.**

Specifically:

1. **Agents bypass MCP tools** — reaching for shell commands (`cat`, `grep`, `find`) to read `.kbz/state/` files instead of using `status`, `entity get`, or `doc get`. The MCP tools exist but the shell is the path of least resistance.

2. **Agents forget optional tool features** — all 12 `finish` calls in P7 omitted the `retrospective` parameter. The feature exists but nothing prompts the agent to use it.

3. **Agents skip workflow steps** — the retrospective was written without first calling `retro synthesise` or `knowledge list`. The tools exist but nothing enforces sequencing.

4. **MCP clients can't distinguish safe from dangerous operations** — every tool call requires human approval because no tool carries safety annotations. This creates a babysitting burden that makes the entire system feel heavy.

5. **Tool descriptions are functional but don't guide behaviour** — descriptions say *what* a tool does but not *when to prefer it* or *what to do instead of bypassing it*.

These problems will affect every project that uses Kanbanzai, not just this one. The fixes must be built into the MCP server binary, not into project-specific `AGENTS.md` files or skills that may not be read.

### What we are not solving here

- Agent-to-agent instruction delivery (prompts, skills, sub-agent context) — important but a separate design concern.
- Workflow enforcement at the lifecycle level (feature can't skip `reviewing`) — already implemented.
- MCP client UI improvements — outside our control.

---

## 2. Design Principles

**The server is the instruction surface, not the documentation.** Rules that live in `AGENTS.md` or skills files require agents to find and read them. Rules built into tool annotations, descriptions, and responses are unavoidable — the agent encounters them in the normal course of using the tools.

**Annotations are contracts, not suggestions.** MCP tool annotations (`readOnlyHint`, `destructiveHint`, etc.) are metadata that clients read at connection time to set their approval policy. They are the only mechanism the MCP spec provides for reducing permission prompts. Inaccurate annotations break trust; missing annotations force the client to assume the worst.

**Descriptions are the first thing an agent reads.** When a model receives a tool list, the description is its primary decision input. A description that says "Get entity details" is less useful than one that says "Get entity details — use this instead of reading .kbz/ files directly. Returns lifecycle state, derived status, and cross-references that raw YAML does not contain."

**Nudges at the point of action beat rules read in advance.** A warning in a `finish` response ("no retrospective signals recorded for this feature") is more reliable than a rule in AGENTS.md ("remember to include retrospective signals"). The agent is already reading the response; it doesn't need to remember anything.

**Conservative by default.** Any tool without explicit annotations must default to the most restrictive classification. Any new tool added without annotations must fail the canary test. Safe defaults prevent regressions.

---

## 3. MCP Spec Alignment

This design targets the **MCP specification 2025-11-25** (latest). The relevant mechanisms are:

### 3.1 Tool Annotations (spec §Tools → Data Types → Tool)

The spec defines:

```
annotations: Optional properties describing tool behavior
```

The mcp-go v0.45.0 library implements this as `ToolAnnotation`:

| Field | Type | Purpose |
|-------|------|---------|
| `Title` | `string` | Human-readable display name for the tool |
| `ReadOnlyHint` | `*bool` | If true, tool does not modify its environment |
| `DestructiveHint` | `*bool` | If true, tool may perform destructive updates |
| `IdempotentHint` | `*bool` | If true, repeated calls with same args have no additional effect |
| `OpenWorldHint` | `*bool` | If true, tool interacts with external entities |

The spec notes: "clients **MUST** consider tool annotations to be untrusted unless they come from trusted servers." For a locally-installed STDIO server like Kanbanzai, the server is trusted — the annotations can be relied upon by the client for auto-approval decisions.

Library helpers:
- `mcp.WithToolAnnotation(mcp.ToolAnnotation{...})` — set all fields at once
- `mcp.WithTitleAnnotation(title)` — set title only
- `mcp.WithReadOnlyHintAnnotation(bool)`
- `mcp.WithDestructiveHintAnnotation(bool)`
- `mcp.WithIdempotentHintAnnotation(bool)`
- `mcp.WithOpenWorldHintAnnotation(bool)`

### 3.2 Tool Descriptions (spec §Tools → Data Types → Tool)

The `description` field is defined as "Human-readable description of functionality". There is no spec constraint on length or format. Descriptions are the primary input the model uses when selecting tools.

### 3.3 Tool Title (spec §Tools → Data Types → Tool)

The `title` field is defined as "Optional human-readable name of the tool for display purposes." This is distinct from `name` (the programmatic identifier). Titles appear in client UIs and help humans identify tools in approval dialogs and tool lists.

---

## 4. Feature A: Tool Annotations

### 4.1 Overview

Restore MCP safety annotations on all 20 Kanbanzai 2.0 tools plus `server_info`. This was implemented for the 1.0 tools (hardening spec §10) but lost when the 2.0 redesign replaced all tools without carrying forward the annotations.

### 4.2 Classification Model

The hardening spec §10.3 defined a three-tier classification. The 2.0 tools are consolidated (multiple actions per tool), so each tool takes the **most permissive tier across all its actions**. We extend the model with `OpenWorldHint` which the mcp-go library now supports.

#### Tier 1 — Read-only

`readOnlyHint: true`, `destructiveHint: false`, `idempotentHint: true`, `openWorldHint: false`

These tools do not modify any state and may be auto-approved freely. For the 2.0 consolidated tools, a tool is Tier 1 only if *every action* is read-only.

| Tool | Justification |
|------|---------------|
| `status` | All actions are read-only queries |
| `health` | Read-only health check |
| `handoff` | Read-only prompt generation (does not modify task status) |
| `conflict` | Read-only conflict analysis |
| `profile` | Read-only profile queries |
| `server_info` | Read-only server metadata |

#### Tier 2 — Auto-approvable writes

`readOnlyHint: false`, `destructiveHint: false`, `openWorldHint: false`

These tools write only to `.kbz/` and do not interact with external systems. They may be auto-approved. `idempotentHint` varies by tool.

| Tool | Idempotent | Justification |
|------|------------|---------------|
| `entity` | `false` | `get`/`list` are read-only, but `create`/`update`/`transition` write to `.kbz/` |
| `doc` | `false` | `get`/`list` are read-only, but `register`/`approve`/`supersede` write to `.kbz/` |
| `doc_intel` | `false` | `classify` writes to `.kbz/index/`; all other actions are read-only |
| `next` | `false` | Queue mode promotes `queued→ready`; claim mode writes dispatch metadata |
| `finish` | `false` | Writes completion metadata, transitions status, may contribute knowledge |
| `knowledge` | `false` | `list`/`get` are read-only, but `contribute`/`confirm`/`retire` write to `.kbz/` |
| `estimate` | `false` | `query` is read-only, but `set` writes to `.kbz/` |
| `decompose` | `false` | `propose` is read-only, but `apply` creates task entities |
| `checkpoint` | `false` | `get`/`list` are read-only, but `create`/`respond` write to `.kbz/` |
| `incident` | `false` | `list` is read-only, but `create`/`update`/`link_bug` write to `.kbz/` |
| `retro` | `false` | `synthesise` is read-only, but `report` writes a file and registers a document |
| `branch` | `true` | Currently only has `status` action (read-only, could be Tier 1, but classified conservatively in case write actions are added later) |

#### Tier 3 — Requires confirmation

`destructiveHint: true` and/or `openWorldHint: true`

These tools interact with Git, GitHub, or the broader filesystem, or perform irreversible operations. MCP clients must not auto-approve these.

| Tool | Destructive | OpenWorld | Justification |
|------|-------------|-----------|---------------|
| `cleanup` | `true` | `false` | `execute` action deletes worktree directories and branches |
| `merge` | `true` | `true` | `execute` action merges branches; may push to remote |
| `pr` | `false` | `true` | `create`/`update` interact with GitHub API |
| `worktree` | `true` | `false` | `create`/`remove` operate on filesystem outside `.kbz/` |

### 4.3 Consolidated Tool Annotation Table

| Tool | readOnly | destructive | idempotent | openWorld | Tier |
|------|----------|-------------|------------|-----------|------|
| `status` | `true` | `false` | `true` | `false` | 1 |
| `health` | `true` | `false` | `true` | `false` | 1 |
| `handoff` | `true` | `false` | `true` | `false` | 1 |
| `conflict` | `true` | `false` | `true` | `false` | 1 |
| `profile` | `true` | `false` | `true` | `false` | 1 |
| `server_info` | `true` | `false` | `true` | `false` | 1 |
| `entity` | `false` | `false` | `false` | `false` | 2 |
| `doc` | `false` | `false` | `false` | `false` | 2 |
| `doc_intel` | `false` | `false` | `false` | `false` | 2 |
| `next` | `false` | `false` | `false` | `false` | 2 |
| `finish` | `false` | `false` | `false` | `false` | 2 |
| `knowledge` | `false` | `false` | `false` | `false` | 2 |
| `estimate` | `false` | `false` | `false` | `false` | 2 |
| `decompose` | `false` | `false` | `false` | `false` | 2 |
| `checkpoint` | `false` | `false` | `false` | `false` | 2 |
| `incident` | `false` | `false` | `false` | `false` | 2 |
| `retro` | `false` | `false` | `false` | `false` | 2 |
| `branch` | `false` | `false` | `true` | `false` | 2 |
| `cleanup` | `false` | `true` | `false` | `false` | 3 |
| `merge` | `false` | `true` | `false` | `true` | 3 |
| `pr` | `false` | `false` | `false` | `true` | 3 |
| `worktree` | `false` | `true` | `false` | `false` | 3 |

### 4.4 Canary Test

A test must iterate every tool registered in the MCP server and assert that all annotation fields are explicitly set (non-nil). Any tool missing an annotation fails the test. This prevents regressions when new tools are added.

The test should be structured as:

1. Instantiate the full server (or all tool groups).
2. Collect every registered `ServerTool`.
3. For each tool, assert `ReadOnlyHint != nil`, `DestructiveHint != nil`, `IdempotentHint != nil`, `OpenWorldHint != nil`.
4. Optionally assert that Tier 1 tools have `ReadOnlyHint == true` and Tier 3 tools have `DestructiveHint == true || OpenWorldHint == true`.

### 4.5 Safe Defaults

Any tool that is registered without explicit annotations must default to the most restrictive classification:

- `readOnlyHint: false`
- `destructiveHint: true`
- `idempotentHint: false`
- `openWorldHint: true`

This ensures that unclassified tools require confirmation, not that they silently auto-approve. The canary test catches this condition; the defaults are a safety net for the gap between adding a tool and running tests.

---

## 5. Feature B: Tool Titles

### 5.1 Overview

Add a human-readable `title` to every tool using `mcp.WithTitleAnnotation()`. Titles appear in MCP client UIs (tool lists, approval dialogs) and help both humans and agents identify tools at a glance.

### 5.2 Title Guidelines

- Titles should be short (3–6 words), descriptive, and action-oriented.
- Titles should distinguish the tool from similar tools. "Entity Manager" is better than "Entity Tool".
- Titles should not repeat the tool name verbatim.

### 5.3 Title Table

| Tool | Name | Title |
|------|------|-------|
| `status` | `status` | Workflow Status Dashboard |
| `entity` | `entity` | Entity Manager |
| `doc` | `doc` | Document Records |
| `doc_intel` | `doc_intel` | Document Intelligence |
| `next` | `next` | Work Queue & Dispatch |
| `handoff` | `handoff` | Sub-Agent Prompt Generator |
| `finish` | `finish` | Task Completion |
| `knowledge` | `knowledge` | Knowledge Base |
| `estimate` | `estimate` | Story Point Estimates |
| `decompose` | `decompose` | Feature Decomposition |
| `conflict` | `conflict` | Conflict Risk Analysis |
| `profile` | `profile` | Context Role Profiles |
| `worktree` | `worktree` | Git Worktree Manager |
| `merge` | `merge` | Merge Gate & Execution |
| `pr` | `pr` | Pull Request Manager |
| `branch` | `branch` | Branch Health Monitor |
| `cleanup` | `cleanup` | Worktree Cleanup |
| `incident` | `incident` | Incident Tracker |
| `checkpoint` | `checkpoint` | Human Decision Checkpoints |
| `health` | `health` | System Health Check |
| `retro` | `retro` | Retrospective Synthesis |
| `server_info` | `server_info` | Server Build Information |

---

## 6. Feature C: Improved Tool Descriptions

### 6.1 Overview

Revise tool descriptions to guide agent behaviour, not just describe functionality. The description is the first (and often only) thing an agent reads when deciding which tool to use. Current descriptions are factual but passive — they say *what* a tool does, not *when to use it* or *what not to do instead*.

### 6.2 Description Principles

1. **Lead with the primary use case**, not the internal structure.
2. **Include a behavioural nudge** where the tool replaces a common bad-habit path. For example, `status` should mention that it replaces reading `.kbz/` files directly.
3. **Name the actions** so the agent knows what's available without guessing.
4. **Keep it concise** — descriptions are included in every tool list response. Long descriptions waste context window.
5. **Don't duplicate parameter descriptions** — those belong on the parameters themselves.

### 6.3 Description Revisions

Below are the revised descriptions for tools where the current description would benefit from behavioural guidance. Tools whose descriptions are already clear and actionable are not listed.

#### `status`

Current:
> "Synthesis dashboard. Aggregates project, plan, feature, or task state into a concise view."

Revised:
> "Synthesis dashboard — the primary way to query project, plan, feature, or task state. Returns lifecycle status, attention items, progress metrics, and derived state (what's blocked, what's ready) that raw YAML files do not contain. Use this instead of reading .kbz/state/ files directly. Call with no id for project overview, plan ID for plan dashboard, FEAT-... for feature detail, TASK-... or BUG-... for task detail."

#### `entity`

Current:
> "Generic CRUD for all entity types (plan, feature, task, bug, epic, decision). Replaces create_task, create_feature, ..."

Revised:
> "Create, read, update, and transition entities (plans, features, tasks, bugs, epics, decisions). Use `action: get` or `action: list` to query entities — these return structured data with lifecycle state and cross-references. Do not read .kbz/state/ YAML files directly. Actions: create, get, list, update, transition. Entity type is inferred from ID prefix for get/update/transition. Supports batch creation via the entities array."

#### `next`

Current:
> "Claim work and get full context. Call without id to inspect the ready queue..."

Revised:
> "Inspect the work queue or claim a task with full context assembly. Call without id to see all ready tasks (sorted by priority, with optional conflict checking). Call with a task/feature/plan ID to claim the next ready task — returns spec sections, knowledge entries, file paths, and role conventions assembled for the task. This is the primary way to pick up work; prefer it over manually querying entities and assembling context yourself."

#### `finish`

Current:
> "Record task completion. Transitions the task to done (default) or needs-review..."

Revised:
> "Record task completion with optional knowledge contribution and retrospective signals. Transitions the task to done (default) or needs-review. Include the `retrospective` parameter to record observations about workflow friction, tool gaps, spec ambiguity, or things that worked well — these feed the `retro` tool for future synthesis. Accepts tasks in ready or active status. Supports batch completion via the tasks array."

#### `doc`

Current:
> "Consolidated document operations. Replaces doc_record_submit, doc_record_approve, ..."

Revised:
> "Manage document records: register, approve, query, supersede, validate, and import. Use `action: get` or `action: list` to query document status — do not read .kbz/state/documents/ files directly. Actions: register, approve, get, content, list, gaps, validate, supersede, import. Approve and supersede report entity lifecycle cascade side effects."

#### `knowledge`

Current:
> "Manage the shared knowledge base. Actions: list, get, contribute, ..."

Revised:
> "Query and manage the project knowledge base. Use `action: list` to find knowledge entries by topic, tag, or status — this surfaces information that may not be in your context window. Routine contribution happens via `finish`; use this tool for direct management: confirming entries, resolving conflicts, checking staleness, pruning. Actions: list, get, contribute, confirm, flag, retire, update, promote, compact, prune, resolve, staleness."

#### `retro`

Current:
> "Retrospective signal synthesis. Reads accumulated retrospective signals..."

Revised:
> "Synthesise retrospective signals into themed clusters. Before writing any retrospective or review document, call `action: synthesise` first — it surfaces signals from across the project that may not be in your session context. Actions: synthesise (read signals, cluster, rank), report (generate and register a markdown report document)."

---

## 7. Feature D: Response Nudges

### 7.1 Overview

Insert contextual guidance into tool responses at key decision points. Unlike descriptions (read once, when the tool list is loaded), nudges appear in the response the agent is already reading — they are impossible to miss and require no prior knowledge.

Nudges are lightweight — a single field in the response JSON, not a modal error. They do not block the operation or change the return schema. They guide the agent toward the next correct action.

### 7.2 Nudge: `finish` — No Retrospective Signals

**When:** The `finish` call completes the last task in a feature, and no tasks in that feature recorded any retrospective signals (via the `retrospective` parameter).

**What:** Include a `nudge` field in the response:

```
"nudge": "No retrospective signals were recorded for any task in this feature. If you observed workflow friction, tool gaps, spec ambiguity, or things that worked well, call finish again on any completed task with the retrospective parameter, or use knowledge(action: contribute) with tags: [\"retrospective\"]."
```

**Why:** This is the exact problem from P7 §3.4 — all 12 `finish` calls omitted retrospective signals. The nudge appears at the moment the agent completes the feature, when reflection is most natural.

### 7.3 Nudge: `finish` — Knowledge Contribution Reminder

**When:** The `finish` call includes a summary but no `knowledge` entries and no `retrospective` signals.

**What:** Include a `nudge` field:

```
"nudge": "Consider including knowledge entries (reusable facts learned during this task) or retrospective signals (process observations) in your finish call. These improve context assembly for future tasks."
```

**Why:** Many tasks produce reusable knowledge that agents don't think to record. A gentle reminder at completion time costs nothing and occasionally captures valuable information.

**Implementation note:** This nudge should be suppressible — if the agent is doing batch completion of trivial tasks, the nudge on every response would be noise. Only include it when `summary` is non-empty and both `knowledge` and `retrospective` are empty/nil.

### 7.4 Nudge Design Constraints

- Nudges are a `nudge` string field in the response JSON — not a separate tool call, not an error.
- Nudges must not change the response schema for existing fields. They are purely additive.
- Nudges should be specific and actionable — name the tool and parameters the agent should use.
- Nudges should be rare. If every response includes a nudge, agents will learn to ignore them.

---

## 8. Feature E: `doc` Refresh Action

### 8.1 Overview

Add a `refresh` action to the `doc` tool that updates a document record's stored content hash after the file has been edited, without requiring the document to be deleted and re-registered.

This was implemented on the hardening branch (commit `8ae3b7e`) but never merged to `main`. The 2.0 `doc` tool has 9 actions; `refresh` is the 10th.

### 8.2 Behaviour

- `doc(action: refresh, id: "...")` or `doc(action: refresh, path: "...")`
- Reads the current file content, computes its hash, compares with the stored hash.
- If changed: updates the stored hash. If the document was `approved`, transitions it to `draft` (the approval is void because the content changed). Returns `changed: true`, `old_hash`, `new_hash`, `status`, and any `status_transition`.
- If unchanged: returns `changed: false`. No state modification.
- If record not found or file not found: returns an error with actionable guidance.

### 8.3 Why This Matters

Without `refresh`, the only way to fix a stale hash is to delete and re-register the document, losing document intelligence data (classifications, entity references, graph relationships). This friction was cited in the P7 retro (KE-01KMT5T79D9Q1) and has recurred across multiple implementation cycles — every time a spec is approved and then its file header still says "Status: Draft", the agent or human has to work around the stale hash.

---

## 9. Feature F: `doc` Supersession Chain Action

### 9.1 Overview

Expose the existing `DocumentService.SupersessionChain()` method via the `doc` tool as `action: chain`. This method is working code with tests — it was a 1.0 tool (`doc_supersession_chain`) that was removed in the 2.0 migration without being wired to the consolidated `doc` tool.

### 9.2 Behaviour

- `doc(action: chain, id: "...")` — returns the ordered chain of document supersessions (A superseded by B superseded by C).
- Read-only. No state modification.

### 9.3 Effort

Minimal — the service method exists and is tested. The only work is wiring it into the `doc` tool's action dispatcher and adding a test for the MCP layer.

---

## 10. Out of Scope (for now)

These are related improvements that came up during analysis but are deferred from this design:

- **MCP Prompts** — server-registered slash commands (e.g. `/kbz-session-start`, `/kbz-review`) that inject structured instructions into the conversation. A promising portable replacement for AGENTS.md rules, but a separate design concern. Prompts address "how do humans kick off workflows"; this design addresses "how does the server make agent behaviour reliable."

- **MCP Resources** — exposing knowledge base entries, skill content, or documents as `kbz://` URIs. Low practical impact until MCP clients proactively consume resources.

- **Error message improvements** — the hardening branch had improved service-layer error messages with actionable guidance. Worth doing but orthogonal to discoverability. Can be a follow-up.

- **`suggest_links` automatic behaviour** — the 2.0 spec said this would be automatic in entity/doc creation. Deferred pending assessment of whether it's still desired.

---

## 11. Open Questions

### 11.1 Implementation decisions (developer resolves during spec or dev-planning)

These are technical choices with clear tradeoffs that a developer can evaluate and decide during specification or implementation. They don't affect the human interface or the agent interaction model in ways that need external input.

1. **Should `branch` be Tier 1 or Tier 2?** It currently only has a `status` action (read-only). Classifying it as Tier 1 is accurate today but would need reclassification if write actions are added. Classifying it as Tier 2 is conservative but slightly noisy for permissions. Leaning Tier 2 for safety. *The developer can assess the likelihood of future write actions and choose accordingly.*

2. **Should `retro report` auto-call `retro synthesise` internally?** The P7 retro showed agents skipping the synthesise step before writing retrospectives. The server could enforce this by having the `report` action call `synthesise` internally as a prerequisite. This changes the semantics of `report` — it would no longer just format; it would also gather. *The developer can evaluate whether composing these internally is cleaner than requiring two calls, and whether the composed version introduces unwanted coupling.*

### 11.2 Agent interface decisions (best decided by an AI agent)

These concern how agents experience the tools — what's helpful vs. noisy, what changes behaviour vs. what gets ignored. An AI agent is better positioned than a human to judge whether a nudge in a tool response will actually influence its own decision-making.

3. **Should we update `handoff` to include a retro reminder?** The `handoff` tool assembles a prompt for sub-agents. Adding "when completing this task, include retrospective signals in your finish call" costs a few tokens and nudges the sub-agent at the right time. *An agent can best judge whether this kind of inline instruction in an assembled prompt actually changes its behaviour, or whether it becomes noise that gets filtered out alongside other boilerplate.*

4. **What description phrasing most effectively steers agents toward MCP tools?** The §6.3 descriptions include phrases like "use this instead of reading .kbz/ files directly." *An agent can evaluate whether this phrasing is clear enough to override the habit of reaching for shell commands, or whether stronger/different language would be more effective.*

### 11.3 Human design decisions (human resolves before implementation)

These affect the overall system behaviour, the human experience of using Kanbanzai, or set precedents for how the server communicates with agents. They need a human designer's judgement.

5. **How aggressive should nudges be?** The design proposes nudges as a `nudge` string field — informational, not blocking. An alternative is to make `finish` on the last task in a feature return `isError: true` if no retro signals exist, forcing the agent to acknowledge the gap. This is more reliable but more intrusive. *This sets a precedent for how the Kanbanzai server communicates dissatisfaction with agent behaviour — soft guidance vs. hard enforcement. That's a product design choice about the system's personality and the human's tolerance for false positives.*

---

## 12. Implementation Approach

### 12.1 Feature ordering

Features A–C (annotations, titles, descriptions) are pure metadata changes on tool registration — no runtime logic changes, no new parameters, no schema changes. They can be implemented as a single commit touching all 22 tool files.

Feature D (nudges) requires runtime logic in `finish` to detect the "last task in feature with no retro signals" condition. Small but not trivial — needs access to sibling task state.

Features E–F (doc refresh, doc chain) are new tool actions requiring service-layer changes. Feature E is a reimplementation (reference code exists on the hardening branch); Feature F is a thin wiring of existing code.

### 12.2 Suggested task breakdown

| Task | Feature | Effort | Dependencies |
|------|---------|--------|--------------|
| 1. Add annotations to all tools | A | Small | None |
| 2. Add canary test for annotations | A | Small | Task 1 |
| 3. Add titles to all tools | B | Small | None |
| 4. Revise tool descriptions | C | Small | None |
| 5. Implement `finish` retro nudge | D | Medium | None |
| 6. Implement `doc(action: refresh)` | E | Medium | None |
| 7. Wire `doc(action: chain)` | F | Tiny | None |

Tasks 1–4 are independent and could be a single commit. Tasks 5–7 are independent of each other but each is a separate commit.

### 12.3 Verification

- `go test -race ./...` — all existing tests must pass.
- Canary test (Task 2) asserts every tool has all four annotation fields set.
- Manual verification: connect an MCP client, confirm that Tier 1/2 tools no longer prompt for approval, Tier 3 tools still do.
- Description review: read the tool list as an agent would and confirm that the descriptions guide toward correct tool selection.