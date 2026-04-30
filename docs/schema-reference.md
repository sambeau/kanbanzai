# Schema Reference

This document is the authoritative reference for the `.kbz` directory structure and all YAML **entity** formats used by Kanbanzai. It covers directory layout, serialisation rules, every entity type with field tables and examples, **lifecycle** state machines, ID formats, and **referential integrity** rules.

---

## 1. Directory Layout

Every Kanbanzai project has a `.kbz` directory at the repository root. This is the complete structure:

```
.kbz/
├── config.yaml              # Project configuration and prefix registry
├── .init-complete           # Sentinel file confirming successful init
├── local.yaml               # Per-machine settings (NOT committed to git)
├── state/                   # All entity YAML files
│   ├── plans/               # Plan entities
│   ├── features/            # Feature entities
│   ├── tasks/               # Task entities
│   ├── bugs/                # Bug entities
│   ├── decisions/           # Decision entities
│   ├── epics/               # Epic entities (deprecated, Phase 1 compatibility)
│   ├── knowledge/           # Knowledge entries
│   ├── documents/           # Document records
│   ├── incidents/           # Incident entities
│   ├── checkpoints/         # Human checkpoint records
│   └── worktrees/           # Worktree tracking records
├── cache/                   # Derived cache (regenerable, not canonical)
├── index/                   # Document intelligence index (Layers 1–4)
│   └── documents/           # Per-document index files
└── context/
    └── roles/               # Context role profiles (YAML)
```

The `state/` directory is the single source of truth. Every entity is one YAML file. The `cache/` directory is derived and can be rebuilt at any time with `rebuild_cache`. The `index/` directory holds the document intelligence pipeline output and is also regenerable from the source documents.

### Files Outside `.kbz`

Two paths outside the `.kbz` directory are managed by Kanbanzai:

- **`.kbz/local.yaml`** — Per-machine settings such as user identity and GitHub credentials. This file must be listed in `.gitignore` and is never committed. It is not created by `kbz init`; users create it manually when needed.

- **`.agents/skills/`** — Skill files for AI agent onboarding. These live outside `.kbz` because they are consumed directly by editors and AI assistants. The `kbz init` command creates and updates these files.

---

## 2. YAML Serialisation Rules

All YAML files written by Kanbanzai follow these conventions:

- **Block style only.** Flow style (`{}` and `[]`) is never used. Lists use the `- item` form.
- **Minimal quoting.** Strings are unquoted unless they contain special YAML characters, are empty, or span multiple lines. When quoting is needed, double quotes are used.
- **Deterministic field order.** Fields appear in the same order as the struct definition for that entity type. This order is fixed and must not be rearranged.
- **UTF-8 encoding** with LF line endings and a trailing newline at end of file.
- **No advanced YAML features.** Tags (`!!str`), anchors (`&name`), and aliases (`*name`) are never used.
- **Omit empty optionals.** Optional fields with zero values, empty strings, empty lists, or nil timestamps are omitted entirely rather than written as empty.

These rules ensure that entity files produce clean, predictable diffs in version control.

---

## 3. Entity Types

### Plan

Plans are the top-level organising unit. They group related features into a coordinated body of work.

**Storage path:** `.kbz/state/plans/{id}.yaml`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | string | yes | Plan ID: `{prefix}{number}-{slug}` (e.g., `P1-my-project`) |
| slug | string | yes | URL-friendly identifier |
| name | string | yes | Human-readable display name |
| status | string | yes | Lifecycle status |
| summary | string | yes | Brief description of purpose and scope |
| design | string | no | Reference to a design document record ID |
| tags | string[] | no | Freeform tags for organisation |
| created | timestamp | auto | Creation timestamp (RFC 3339) |
| created_by | string | auto | Creator identity |
| updated | timestamp | auto | Last update timestamp |
| supersedes | string | no | ID of the plan this one supersedes |
| superseded_by | string | no | ID of the plan that supersedes this one |

**Valid statuses:** proposed → designing → active → reviewing → done. From any non-terminal state: → superseded, → cancelled.

**Example:**

```
id: P3-kanbanzai-1.0
slug: kanbanzai-1.0
name: Kanbanzai 1.0
status: active
summary: "Make Kanbanzai installable and usable by projects other than itself: pre-compiled binary distribution, skills-based agent onboarding, public schema interface, kbz init command, and user documentation."
design: PROJECT/design-kanbanzai-10
created: "2026-03-26T00:28:22Z"
created_by: sambeau
updated: "2026-03-26T14:48:05Z"
```

---

### Feature

Features represent deliverable units of work within a plan. In Phase 2, features follow a document-driven lifecycle where design, specification, and dev-plan documents gate progression.

**Storage path:** `.kbz/state/features/{id}-{slug}.yaml`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | string | yes | `FEAT-{TSID13}` |
| slug | string | yes | URL-friendly identifier |
| parent | string | yes | Parent Plan ID |
| status | string | yes | Lifecycle status |
| estimate | number | no | Story points (Modified Fibonacci scale) |
| summary | string | yes | Brief description |
| created | timestamp | auto | Creation timestamp |
| created_by | string | auto | Creator identity |
| updated | timestamp | auto | Last update timestamp |
| design | string | no | Document record ID for the design document |
| spec | string | no | Document record ID for the specification |
| dev_plan | string | no | Document record ID for the dev plan |
| tags | string[] | no | Freeform tags |
| name | string | yes | Human-readable display name |
| review_cycle | int | no | Review cycle counter |
| blocked_reason | string | no | Why feature is blocked |
| overrides | []OverrideRecord | no | Gate override history |
| plan | string | no | Legacy: alternative parent reference |
| tasks | string[] | no | Denormalised list of child task IDs |
| decisions | string[] | no | Related decision IDs |
| branch | string | no | Feature branch name |
| supersedes | string | no | ID of the feature this one supersedes |
| superseded_by | string | no | ID of the feature that supersedes this one |

**Valid statuses (Phase 2, document-driven):** proposed → designing → specifying → dev-planning → developing → reviewing → done. From any non-terminal state: → superseded, → cancelled.

**Example:**

```
id: FEAT-01KMKRQRRX3CC
slug: init-command
parent: P3-kanbanzai-1.0
status: developing
summary: "The kbz init command: creates .kbz/config.yaml, installs Kanbanzai-managed skill files into .agents/skills/, records document roots, handles both new and existing projects without touching pre-existing files."
design: FEAT-01KMKRQRRX3CC/design-init-command
spec: FEAT-01KMKRQRRX3CC/specification-init-command
dev_plan: FEAT-01KMKRQRRX3CC/dev-plan-init-command
created: "2026-03-26T00:29:32Z"
created_by: sambeau
```

---

### Task

Tasks are the atomic units of work. Each task belongs to a feature and can declare dependencies on other tasks. Tasks are what agents pick up from the work queue and execute.

**Storage path:** `.kbz/state/tasks/{id}-{slug}.yaml`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | string | yes | `TASK-{TSID13}` |
| parent_feature | string | yes | Parent feature ID |
| slug | string | yes | URL-friendly identifier |
| name | string | yes | Human-readable display name |
| summary | string | yes | Brief description of the work |
| status | string | yes | Lifecycle status |
| estimate | number | no | Story points |
| assignee | string | no | Assigned agent or person |
| depends_on | string[] | no | Task IDs this task depends on |
| files_planned | string[] | no | Files expected to be modified |
| started | timestamp | no | When work began |
| completed | timestamp | no | When work finished |
| claimed_at | timestamp | no | When dispatched to an agent |
| dispatched_to | string | no | Role profile the task was dispatched to |
| dispatched_at | timestamp | no | Dispatch timestamp |
| dispatched_by | string | no | Who or what dispatched the task |
| completion_summary | string | no | Summary written by `complete_task` |
| rework_reason | string | no | Reason for needs-rework transition |
| verification | string | no | Verification notes from the completing agent |
| tags | string[] | no | Freeform tags |

**Valid statuses:** queued → ready → active → done. Also: blocked (from active), needs-review (from active), needs-rework (from needs-review or active). Terminal states: done, not-planned, duplicate.

Tasks in queued status automatically promote to ready when all entries in their `depends_on` list reach a terminal state (done, not-planned, or duplicate).

**Example:**

```
id: TASK-01KMNA39KTWW4
parent_feature: FEAT-01KMKRQRRX3CC
slug: init-command-skeleton
summary: "Implement kbz init CLI subcommand skeleton with flag parsing: --docs-path, --skip-skills, --update-skills, --non-interactive."
status: done
files_planned:
  - cmd/kanbanzai/init_cmd.go
  - cmd/kanbanzai/main.go
  - internal/kbzinit/init.go
  - internal/kbzinit/git.go
  - internal/kbzinit/config_writer.go
completed: "2026-03-26T15:13:30Z"
claimed_at: "2026-03-26T15:03:55Z"
dispatched_to: backend
dispatched_at: "2026-03-26T15:03:55Z"
dispatched_by: agent/init-command
completion_summary: "Implemented kbz init as a CLI subcommand. Flag definitions: --docs-path, --skip-skills, --update-skills, --non-interactive, --skip-work-dirs. Validates mutually exclusive flags."
verification: "go build ./..., go test -race ./internal/kbzinit/... (37 tests pass), go vet ./..."
```

---

### Bug

Bugs track defects with structured severity, priority, and classification. They follow a lifecycle from report through triage, reproduction, fix, and verification.

**Storage path:** `.kbz/state/bugs/{id}-{slug}.yaml`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | string | yes | `BUG-{TSID13}` |
| slug | string | yes | URL-friendly identifier |
| name | string | yes | Bug name |
| status | string | yes | Lifecycle status |
| estimate | number | no | Story points |
| severity | string | yes | `low`, `medium`, `high`, or `critical` |
| priority | string | yes | `low`, `medium`, `high`, or `critical` |
| type | string | yes | `implementation-defect`, `specification-defect`, or `design-problem` |
| reported_by | string | yes | Who reported the bug |
| reported | timestamp | auto | When the bug was reported |
| observed | string | yes | What was observed (the actual behaviour) |
| expected | string | yes | What was expected (the correct behaviour) |
| affects | string[] | no | Affected entity IDs |
| origin_feature | string | no | Feature where the bug was discovered |
| origin_task | string | no | Task where the bug was discovered |
| environment | string | no | Environment details |
| reproduction | string | no | Steps to reproduce |
| duplicate_of | string | no | ID of the bug this duplicates |
| fixed_by | string | no | Who fixed it |
| verified_by | string | no | Who verified the fix |
| release_target | string | no | Target release |
| tags | string[] | no | Freeform tags |

**Valid statuses:** reported → triaged → reproduced → planned → in-progress → needs-review → verified → closed. Also: triaged → cannot-reproduce, needs-review → needs-rework → in-progress. Terminal states: closed, duplicate, not-planned.

**Example:**

```
id: BUG-01KMKA1KEFYX0
slug: incident-yaml-field-ordering
name: Incident YAML field ordering non-deterministic in storage
status: reported
severity: high
priority: high
type: implementation-defect
reported_by: sambeau
reported: "2026-03-25T20:12:45Z"
observed: Incident YAML files are written with fields in alphabetical order
expected: "Fields written in canonical order per spec: id, slug, title, status, severity, reported_by, detected_at, triaged_at, mitigated_at, resolved_at, affected_features, linked_bugs, linked_rca, summary, created, created_by, updated"
origin_feature: FEAT-01KMKA278DFNV
```

---

### Decision

Decisions record architectural and process choices with their rationale. They can reference the entities they affect and form supersession chains as decisions evolve.

**Storage path:** `.kbz/state/decisions/{id}-{slug}.yaml`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | string | yes | `DEC-{TSID13}` |
| slug | string | yes | URL-friendly identifier |
| name | string | yes | Human-readable display name |
| summary | string | yes | Decision summary |
| rationale | string | yes | Why this decision was made |
| decided_by | string | yes | Who made the decision |
| date | timestamp | auto | When the decision was made |
| status | string | yes | Lifecycle status |
| affects | string[] | no | Entity IDs affected by this decision |
| supersedes | string | no | ID of the decision this one supersedes |
| superseded_by | string | no | ID of the decision that supersedes this one |
| tags | string[] | no | Freeform tags |

**Valid statuses:** proposed → accepted. Terminal states: rejected, superseded.

**Example:**

```
id: DEC-01KM8JVTJ8DGS
slug: bootstrap-activation
summary: Activate bootstrap self-hosting
rationale: All prerequisites met per bootstrap spec §3
decided_by: human
date: "2026-03-21T16:15:15Z"
status: proposed
```

---

### Knowledge Entry

Knowledge entries capture reusable project knowledge discovered during task execution. They are scoped to a role profile or to the whole project, tiered by longevity, and have confidence scores that adjust automatically based on usage and flagging.

**Storage path:** `.kbz/state/knowledge/{id}.yaml`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | string | yes | `KE-{TSID13}` |
| tier | number | yes | `2` (project-level) or `3` (session-level) |
| topic | string | yes | Normalised topic (lowercase, hyphenated) |
| scope | string | yes | Profile name (e.g., `backend`) or `project` |
| content | string | yes | Concise, actionable knowledge statement |
| learned_from | string | no | Provenance — typically a task ID |
| status | string | yes | Lifecycle status |
| use_count | number | auto | Times used in context assembly |
| miss_count | number | auto | Times flagged as incorrect |
| confidence | number | auto | Wilson score confidence (0.0–1.0) |
| ttl_days | number | auto | Time-to-live in days: 30 (tier 3), 90 (tier 2), 0 (exempt) |
| git_anchors | string[] | no | File paths for staleness checking |
| tags | string[] | no | Classification tags |
| last_used | string | no | Timestamp of last use in context assembly |
| promoted_from | string | no | ID of the entry this was promoted from |
| merged_from | string[] | no | IDs of entries merged into this one |
| deprecated_reason | string | no | Reason the entry was deprecated |
| created | timestamp | auto | Creation timestamp |
| created_by | string | auto | Creator identity |
| updated | timestamp | auto | Last update timestamp |

**Valid statuses:** contributed → confirmed (automatically when `use_count` ≥ 3 and `miss_count` = 0) or → disputed (when flagged). Terminal state: retired.

Tier-3 entries expire after 30 days without use. Tier-2 entries expire after 90 days. Entries with `ttl_days: 0` are exempt from expiry.

**Example:**

```
id: KE-01KMKEZC72XAY
tier: 2
topic: tsid13-id-system
scope: project
content: "TSID13 IDs use a single character prefix (FEAT-, TASK-, BUG-, INC-) followed by 13-character TSID. Plans use multi-char prefixes based on registry (e.g. P1-, P2-). All IDs are immutable once created."
learned_from: P2-phase-1-kernel
status: contributed
use_count: 0
miss_count: 0
confidence: "0.5"
ttl_days: 90
tags:
  - id-system
  - architecture
created: "2026-03-25T21:38:55Z"
created_by: sambeau
updated: "2026-03-25T21:38:55Z"
```

---

### Document Record

Document records track the lifecycle of design documents, specifications, and dev plans. They link a Markdown file on disk to an owning entity and manage approval, content hashing for drift detection, and supersession.

**Storage path:** `.kbz/state/documents/{id-with-slashes-replaced}.yaml`

The document record ID contains a slash (e.g., `FEAT-01ABC/design-my-feature`). In the filename, the slash is replaced with `--` (e.g., `FEAT-01ABC--design-my-feature.yaml`).

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | string | yes | Composite: `{owner}/{type}-{slug}` or `PROJECT/{type}-{slug}` |
| path | string | yes | Relative file path from the repo root |
| type | string | yes | `design`, `specification`, `dev-plan`, `research`, `report`, `policy`, `rca`, `plan`, or `retrospective` |
| title | string | yes | Human-readable title |
| status | string | yes | Lifecycle status |
| owner | string | no | Parent Plan or Feature ID |
| approved_by | string | no | Who approved the document |
| approved_at | timestamp | no | When the document was approved |
| content_hash | string | auto | SHA-256 hash of file content at registration or approval |
| supersedes | string | no | ID of the document record this one supersedes |
| superseded_by | string | no | ID of the document record that supersedes this one |
| created | timestamp | auto | Creation timestamp |
| created_by | string | auto | Creator identity |
| quality_evaluation | QualityEvaluation | no | Quality assessment (overall_score, pass, evaluated_at, evaluator, dimensions) |
| updated | timestamp | auto | Last update timestamp |

**Valid statuses:** draft → approved → superseded.

Approving a document locks its `content_hash`. If the file on disk later changes, drift detection flags a mismatch. Document approval can trigger forward lifecycle transitions on the owning feature (e.g., approving a design moves the feature from designing to specifying). Document supersession can trigger backward transitions.

**Example:**

```
id: FEAT-01KMKRQRRX3CC/design-init-command
path: work/design/init-command.md
type: design
title: kbz init Command Design
status: approved
owner: FEAT-01KMKRQRRX3CC
approved_by: sambeau
approved_at: "2026-03-26T10:18:04Z"
content_hash: 15e486b2966498eefcf00170e34b900f14e18a02c6be0a2b00dec731beab84fc
created: "2026-03-26T10:13:05Z"
created_by: sambeau
updated: "2026-03-26T10:18:04Z"
```

---

### Worktree

Worktree records track Git worktrees created for feature or bug development. Each worktree provides an isolated workspace with its own branch.

**Storage path:** `.kbz/state/worktrees/{id}.yaml`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | string | yes | `WT-{TSID13}` |
| entity_id | string | yes | Associated feature or bug ID |
| branch | string | yes | Git branch name |
| path | string | yes | Worktree directory path (relative to repo root) |
| status | string | yes | `active`, `merged`, or `abandoned` |
| merged_at | timestamp | no | When the worktree branch was merged |
| cleanup_after | timestamp | no | Grace period expiry for cleanup |
| graph_project | string | no | Codebase-memory-mcp project name for graph-based navigation |
| created | timestamp | auto | Creation timestamp |
| created_by | string | auto | Creator identity |

**Example:**

```
id: WT-01KMNADCTWN72
entity_id: FEAT-01KMKRQSD1TKK
branch: feature/FEAT-01KMKRQSD1TKK-skills-content
path: .worktrees/FEAT-01KMKRQSD1TKK-skills-content
status: active
created: "2026-03-26T14:57:41Z"
created_by: sambeau
```

---

### Incident

Incidents track production issues and outages. They link to affected features and bugs, and can reference root cause analysis documents.

**Storage path:** `.kbz/state/incidents/{id}-{slug}.yaml`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | string | yes | `INC-{TSID13}` |
| slug | string | yes | URL-friendly identifier |
| name | string | yes | Incident name |
| status | string | yes | Lifecycle status |
| severity | string | yes | `critical`, `high`, `medium`, or `low` |
| reported_by | string | yes | Who reported the incident |
| detected_at | timestamp | auto | When the incident was detected |
| triaged_at | timestamp | no | When the incident was triaged |
| mitigated_at | timestamp | no | When the incident was mitigated |
| resolved_at | timestamp | no | When the incident was resolved |
| affected_features | string[] | no | Affected feature IDs |
| linked_bugs | string[] | no | Linked bug IDs |
| linked_rca | string | no | Linked RCA document record ID |
| summary | string | yes | Incident summary |
| created | timestamp | auto | Creation timestamp |
| created_by | string | auto | Creator identity |
| updated | timestamp | auto | Last update timestamp |

**Valid statuses:** reported → triaged → investigating → root-cause-identified → mitigated → resolved → closed.

**Example:**

```
id: INC-01KMABC123DEF
slug: api-timeout-spike
name: API response times spiking above 10s
status: investigating
severity: high
reported_by: monitoring
detected_at: "2026-03-27T03:15:00Z"
triaged_at: "2026-03-27T03:22:00Z"
affected_features:
  - FEAT-01KMKRQRRX3CC
summary: "API latency spiked to 10s+ across all endpoints starting at 03:15 UTC. Affects approximately 40% of requests."
created: "2026-03-27T03:15:00Z"
created_by: monitoring
updated: "2026-03-27T03:22:00Z"
```

---

### Human Checkpoint

Human checkpoints record decision points where an orchestrating agent pauses for human input. They are stored as YAML files with other entities.

**Storage path:** `.kbz/state/checkpoints/{id}.yaml`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | string | yes | `CHK-{TSID13}` |
| question | string | yes | The decision or question requiring human input |
| context | string | yes | Background information to help the human answer |
| orchestration_summary | string | yes | State of the orchestration session at checkpoint time |
| created_by | string | yes | Identity of the orchestrating agent |
| status | string | yes | `pending` or `responded` |
| response | string | no | The human's answer (populated after response) |
| created_at | timestamp | auto | Creation timestamp |
| responded_at | timestamp | no | When the human responded |

**Example:**

```
id: CHK-01KMDEF456GHI
question: "Should we split the auth feature into separate OAuth and API-key features?"
context: "The auth feature has grown to 13 tasks spanning two distinct concerns. Splitting would allow parallel development but adds coordination overhead."
orchestration_summary: "Decomposing FEAT-01KMXYZ into tasks. Found 13 candidate tasks across two domains."
created_by: agent/orchestrator
status: pending
created: "2026-03-27T10:30:00Z"
```

---

## 4. Lifecycle State Machines

Each entity type with a lifecycle follows a defined **state machine**. The system rejects unlisted transitions.

### Plan

```
proposed → designing → active → reviewing → done
```

From any non-terminal state, a plan can also transition to **superseded** or **cancelled**.

Back-transitions: reviewing → active.

> **Note:** `done` is non-terminal — plans can still transition to **superseded** or **cancelled** from `done`.

### Feature (Phase 2, document-driven)

```
proposed → designing → specifying → dev-planning → developing → reviewing → done
```

From any non-terminal state: → **superseded**, → **cancelled**.

Backward transitions are triggered by document supersession:
- specifying → designing (design document superseded)
- dev-planning → specifying (specification superseded)
- developing → dev-planning (dev plan superseded)
- reviewing → needs-rework (review fail)

### Task

```
queued → ready → active → done
                  ├────→ needs-review → done
                  │                   → needs-rework → active
                  ├────→ blocked → active
                  ├────→ ready          (unclaim / crash recovery)
                  └────→ needs-rework   (review fail from active)
```

Terminal states: **done**, **not-planned**, **duplicate**.

Tasks promote from queued to ready automatically when all dependencies reach a terminal state.

### Bug

```
reported → triaged → reproduced → planned → in-progress → needs-review → verified → closed
              │                                                │
              ├→ cannot-reproduce → triaged                    └→ needs-rework → in-progress
              ├→ planned
              ├→ not-planned
              └→ duplicate
```

Terminal states: **closed**, **duplicate**, **not-planned**.

### Decision

```
proposed → accepted
```

Terminal states: **rejected**, **superseded**.

### Incident

```
reported → triaged → investigating → root-cause-identified → mitigated → resolved → closed
                         ↑                    │                   │
                         └────────────────────┘                   │
                         ↑                                        │
                         └────────────────────────────────────────┘
```

Back-transitions: root-cause-identified → investigating, mitigated → investigating.

From any non-terminal state: → **closed** (for false alarms or early closure).

### Document Record

```
draft → approved → superseded
```

### Knowledge Entry

```
contributed → confirmed    (auto: use_count ≥ 3, miss_count = 0)
           → disputed      (via flagging)
           → retired        (auto: miss_count ≥ 2, or manual, or TTL expiry)
```

Terminal state: **retired**.

---

## 5. ID Formats

### TSID13-based IDs

All entities except Plans use a **TSID13** (Time-Sorted ID, 13 characters) as the unique portion of their ID. TSID13 uses Crockford base32 encoding with a 48-bit millisecond timestamp and a 15-bit random component. The format is `{PREFIX}-{TSID13}`, where the prefix identifies the entity type:

| Prefix | Entity Type |
|--------|-------------|
| `FEAT-` | Feature |
| `TASK-` | Task |
| `BUG-` | Bug |
| `DEC-` | Decision |
| `KE-` | Knowledge Entry |
| `WT-` | Worktree |
| `INC-` | Incident |
| `CHK-` | Human Checkpoint |

TSID13s are 13-character strings encoding a millisecond timestamp and random component, which means they sort chronologically. Example: `FEAT-01KMKRQRRX3CC`.

### Plan IDs

Plans use a different format: `{prefix}{number}-{slug}`.

- **prefix** — A single non-digit Unicode character from the prefix registry (e.g., `P`).
- **number** — An auto-incremented integer, unique per prefix.
- **slug** — A URL-friendly identifier.

Examples: `P1-my-project`, `P2-phase-two`, `P3-kanbanzai-1.0`.

### Document Record IDs

Document records use a composite format that encodes ownership and type:

- **Owned documents:** `{owner-entity-id}/{type}-{slug}` — e.g., `FEAT-01KMKRQRRX3CC/design-init-command`.
- **Unowned (project-level) documents:** `PROJECT/{type}-{slug}` — e.g., `PROJECT/design-kanbanzai-10`.

In YAML filenames, the slash is replaced with `--` to stay filesystem-safe. For example, the document record `FEAT-01KMKRQRRX3CC/design-init-command` is stored at `.kbz/state/documents/FEAT-01KMKRQRRX3CC--design-init-command.yaml`.

### YAML Filenames

Entity files follow one of two naming patterns:

- **Entities with slugs:** `{ID}-{slug}.yaml` — e.g., `FEAT-01KMKRQRRX3CC-init-command.yaml`.
- **Entities without slugs:** `{ID}.yaml` — e.g., `KE-01KMKEZC72XAY.yaml`.

---

## 6. Prefix Registry

The prefix registry lives in `.kbz/config.yaml` and controls which Plan ID prefixes are available.

```
version: "2"
prefixes:
  - prefix: P
    name: Plan
```

### Rules

- Each prefix must be a single non-digit Unicode character.
- Prefix names are human-readable labels (e.g., "Plan", "Sprint", "Release").
- At least one active prefix must remain at all times.
- Prefixes can be retired but not deleted. A retired prefix remains valid for existing Plans but cannot be used for new ones.
- The auto-incremented number is tracked per prefix, so `P1-...` and `R1-...` are independent sequences.
- New prefixes are added with `add_prefix` and retired with `retire_prefix`.

---

## 7. Referential Integrity

Kanbanzai validates references at creation time but does not enforce cascading deletes or updates. The rules below describe the expected relationships between entities.

### Hard References (Validated on Creation)

| Source | Field | Target |
|--------|-------|--------|
| Feature | `parent` | Valid Plan ID |
| Task | `parent_feature` | Valid Feature ID |
| Task | `depends_on` | Valid Task IDs |
| Document Record | `owner` | Valid Feature or Plan ID (if set) |

### Soft References (Validated When Set, Not Cascaded)

| Source | Field | Target |
|--------|-------|--------|
| Bug | `origin_feature` | Feature ID |
| Bug | `origin_task` | Task ID |
| Bug | `duplicate_of` | Bug ID |
| Incident | `linked_bugs` | Bug IDs |
| Incident | `affected_features` | Feature IDs |
| Incident | `linked_rca` | Document Record ID |
| Knowledge Entry | `learned_from` | Task ID or other reference |
| Decision | `affects` | Entity IDs |
| Feature | `design`, `spec`, `dev_plan` | Document Record IDs |

### What "Advisory" Means

If a referenced entity is later deleted or transitions to a terminal state, Kanbanzai does not automatically clear the referencing field. Tools that read these references handle missing targets gracefully. The `health_check` tool reports broken references as warnings.

---

## 8. Estimation Scale

Story point estimates use the Modified Fibonacci scale across all entity types that support estimation (tasks, features, bugs, epics, and plans):

```
0, 0.5, 1, 2, 3, 5, 8, 13, 20, 40, 100
```

Features and plans roll up estimates from their children. A feature's estimate is the sum of its task estimates. A plan's estimate is the sum of its feature estimates.

---

## 9. Timestamps

All timestamps are stored in RFC 3339 format with UTC timezone. They are always double-quoted in YAML to prevent parser ambiguity:

```
created: "2026-03-26T00:28:22Z"
```

The system sets fields marked "auto" in the entity tables automatically — do not edit them by hand. It sets `created` once at entity creation and refreshes `updated` on every write.

---

## 10. Configuration File

The project configuration file at `.kbz/config.yaml` contains:

| Field | Type | Description |
|-------|------|-------------|
| version | string | Schema version (currently `"2"`) |
| prefixes | object[] | Prefix registry entries |
| decomposition | object | Optional decomposition settings |

Each prefix entry has:

| Field | Type | Description |
|-------|------|-------------|
| prefix | string | Single non-digit Unicode character |
| name | string | Human-readable label |
| description | string | Optional description of the prefix's purpose |
| retired | boolean | Whether the prefix is retired (omitted if false) |

The configuration file is created by `kbz init` and should be committed to version control.

> **Note:** This table lists the core fields. The Configuration Reference document provides the complete listing of all supported configuration options.

---

## 11. Local Configuration

The optional file `.kbz/local.yaml` stores per-machine settings. It must be listed in `.gitignore`.

| Field | Type | Description |
|-------|------|-------------|
| user.name | string | User identity for `created_by` and `decided_by` fields |
| github.token | string | GitHub personal access token |
| github.owner | string | GitHub repository owner |
| github.repo | string | GitHub repository name |
| tool_hints | map[string]string | Per-user tool hint overrides |

If `user.name` is not set in `local.yaml`, the system falls back to `git config user.name`.

```
user:
  name: alice

github:
  token: ghp_xxxxxxxxxxxxxxxxxxxx
  owner: my-org
  repo: my-project
```
