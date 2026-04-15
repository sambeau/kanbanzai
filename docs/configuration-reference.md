# Configuration Reference

Reference for all Kanbanzai configuration files. Covers **config.yaml**
(project-wide settings), **local.yaml** (user-local overrides), **context
profiles**, and validation.

---

## Table of Contents

1. [config.yaml — Project Configuration](#1-configyaml--project-configuration)
2. [Prefix Registry](#2-prefix-registry)
3. [local.yaml — User-Local Configuration](#3-localyaml--user-local-configuration)
4. [Context Profiles](#4-context-profiles)
5. [Environment Variables](#5-environment-variables)
6. [Validation](#6-validation)
7. [Migration](#7-migration)

---

## 1. config.yaml — Project Configuration

**Location:** `.kbz/config.yaml`

The main project configuration file. It lives in version control and everyone
in the repository shares it. Kanbanzai creates it automatically when you
initialise a project; you can edit it by hand or through MCP tools.

### Top-Level Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `name` | string | `""` | Project name. |
| `schema_version` | string | `"1.0.0"` | Schema version (semver, auto-set). |
| `version` | string | `"2"` | Configuration schema version. |
| `prefixes` | PrefixEntry[] | `[{prefix: "P", name: "Plan"}]` | Plan ID prefix registry. |
| `import` | ImportConfig | *(see below)* | Batch document import settings. |
| `branch_tracking` | BranchTrackingConfig | *(see below)* | Branch staleness and drift detection. |
| `cleanup` | CleanupConfig | *(see below)* | Worktree cleanup settings. |
| `knowledge` | KnowledgeConfig | *(see below)* | Knowledge entry lifecycle. |
| `dispatch` | DispatchConfig | *(see below)* | Task dispatch settings. |
| `incidents` | IncidentsConfig | *(see below)* | Incident management settings. |
| `decomposition` | DecompositionConfig | *(see below)* | Feature decomposition settings. |
| `merge` | MergeConfig | *(see below)* | Post-merge settings. |
| `freshness` | FreshnessConfig | *(see below)* | Staleness window settings. |
| `mcp` | MCPConfig | *(see below)* | MCP tool surface configuration. |
| `quality_evaluation` | QualityEvaluationConfig | *(see below)* | Document quality settings. |
| `lifecycle` | LifecycleConfig | *(see below)* | Lifecycle timing settings. |
| `tool_hints` | map[string]string | `nil` | Per-tool agent guidance strings. |

### PrefixEntry

Each element in the `prefixes` array describes one Plan ID **prefix**.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `prefix` | string | yes | Single non-digit Unicode character (e.g. `P`, `M`, `Ω`). |
| `name` | string | yes | Human-readable name shown in listings. |
| `description` | string | no | Longer explanation of the prefix's purpose. |
| `retired` | bool | no | Whether this prefix is retired. Default: `false`. |

### ImportConfig

Controls how `doc(action: "import")` maps file paths to document types.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `type_mappings` | ImportTypeMapping[] | *(see below)* | Path-to-document-type mappings. |

Each mapping has two fields:

| Field | Type | Description |
|-------|------|-------------|
| `glob` | string | Glob pattern matched against relative paths. |
| `type` | string | Document type assigned to matches. |

Default type mappings:

| Glob | Type |
|------|------|
| `*/design/*` | design |
| `*/spec/*` | specification |
| `*/plan/*` | report |
| `*/research/*` | research |

### BranchTrackingConfig

Thresholds for the `branch_status` tool that reports staleness and drift from
the main branch.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `stale_after_days` | int | `14` | Days of inactivity before a branch is considered stale. |
| `drift_warning_commits` | int | `50` | Commits behind main that trigger a warning. |
| `drift_error_commits` | int | `100` | Commits behind main that trigger an error. |

### CleanupConfig

Settings for worktree cleanup after branches are merged or abandoned.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `grace_period_days` | int | `7` | Days to wait after merge before cleaning up. |
| `auto_delete_remote_branch` | bool | `true` | Delete the remote branch when a worktree is cleaned up. |

### KnowledgeConfig

Lifecycle settings for the knowledge base. Organised into three sub-sections.

**TTL (time-to-live)**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `ttl.tier_3_days` | int | `30` | Days before an unused tier-3 entry expires. |
| `ttl.tier_2_days` | int | `90` | Days before an unused tier-2 entry expires. |

**Promotion**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `promotion.min_use_count` | int | `5` | Number of uses required for auto-promotion. |
| `promotion.max_miss_count` | int | `0` | Maximum miss count for promotion eligibility. |
| `promotion.min_confidence` | float | `0.7` | Minimum confidence score for promotion. |

**Pruning**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `pruning.grace_period_days` | int | `7` | Grace period before expired entries are actually pruned. |

### DispatchConfig

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `stall_threshold_days` | int | `3` | Days a dispatched task can remain active before it is flagged as stalled. |

### IncidentsConfig

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `rca_link_warn_after_days` | int | `7` | Days after resolution before a warning is raised about a missing root-cause analysis link. |

### DecompositionConfig

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `max_tasks_per_feature` | int | `20` | Soft limit on the number of tasks produced by a single decomposition. |

### MergeConfig

Post-merge behaviour settings.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `post_merge_install` | *bool | `true` (nil treated as true) | Reinstall binary after merge. |

### FreshnessConfig

Staleness window settings for knowledge and document freshness.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `staleness_window_days` | int | `30` | Days before knowledge/doc freshness warnings. |

### MCPConfig

MCP tool surface configuration.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `preset` | string | `"full"` | MCP tool surface preset (full, minimal). |
| `groups` | map[string]bool | `nil` | Fine-grained tool group enables/disables. |

### QualityEvaluationConfig

Document quality evaluation settings.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `require_quality_evaluation` | bool | `false` | Require doc quality eval before approval. |
| `quality_evaluation_threshold` | float64 | `0` | Minimum quality score. |

### LifecycleConfig

Lifecycle timing settings.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `stale_reviewing_days` | int | `7` | Days in reviewing before staleness warning. |

### Minimal Example

The smallest valid configuration. Every other section inherits built-in
defaults.

```yaml
version: "2"
prefixes:
  - prefix: P
    name: Plan
```

### Full Example

All sections with explicit values (these happen to match the defaults).

```yaml
version: "2"
prefixes:
  - prefix: P
    name: Plan
  - prefix: M
    name: Maintenance
    description: "Maintenance and tech debt work"

import:
  type_mappings:
    - glob: "*/design/*"
      type: design
    - glob: "*/spec/*"
      type: specification
    - glob: "*/plan/*"
      type: report
    - glob: "*/research/*"
      type: research

branch_tracking:
  stale_after_days: 14
  drift_warning_commits: 50
  drift_error_commits: 100

cleanup:
  grace_period_days: 7
  auto_delete_remote_branch: true

knowledge:
  ttl:
    tier_3_days: 30
    tier_2_days: 90
  promotion:
    min_use_count: 5
    max_miss_count: 0
    min_confidence: 0.7
  pruning:
    grace_period_days: 7

dispatch:
  stall_threshold_days: 3

incidents:
  rca_link_warn_after_days: 7

decomposition:
  max_tasks_per_feature: 20

merge:
  post_merge_install: true

freshness:
  staleness_window_days: 30

mcp:
  preset: "full"
  groups: {}

quality_evaluation:
  require_quality_evaluation: false
  quality_evaluation_threshold: 0

lifecycle:
  stale_reviewing_days: 7

tool_hints: {}
```

---

## 2. Prefix Registry

Prefixes live in the `prefixes` array inside `config.yaml`. They control how
Kanbanzai forms Plan IDs.

### Rules

- Each prefix is a **single non-digit Unicode character** — letters, symbols,
  and emoji all work (`P`, `M`, `Ω`, etc.).
- At least one active (non-retired) prefix must exist at all times.
- Plan IDs follow the pattern `{prefix}{number}-{slug}`, where `number`
  auto-increments per prefix. For example: `P1-basic-ui`, `M3-tech-debt`.

### Managing Prefixes

| Action | Method | Notes |
|--------|--------|-------|
| Add a prefix | Edit `config.yaml` or use the `init` command | Add an entry to the `prefixes` array with a character and human-readable name. The `Config` struct has an `AddPrefix()` method internally. |
| Retire a prefix | Edit `config.yaml` | Set `retired: true` on the prefix entry. Retired prefixes cannot be used for new Plans but remain valid for Plans that already use them. The `Config` struct has a `RetirePrefix()` method internally. |

You perform these operations by editing `config.yaml` directly or through the
init command. There are no standalone MCP tools for prefix management.

---

## 3. local.yaml — User-Local Configuration

**Location:** `.kbz/local.yaml`

This file holds per-user settings — credentials and identity — that must
**not** be committed to version control. Add it to your `.gitignore`:

```gitignore
.kbz/local.yaml
```

### Fields

| Field | Type | Description |
|-------|------|-------------|
| `user.name` | string | User identity used for `created_by`, `approved_by`, and similar fields. |
| `github.token` | string | GitHub personal access token for PR and repository tools. |
| `github.owner` | string | Override the auto-detected repository owner. |
| `github.repo` | string | Override the auto-detected repository name. |
| `tool_hints` | map[string]string | Per-user tool hint overrides (merged with project-level tool_hints). |

### Identity Resolution

When a tool needs to know who you are, it checks these sources in order:

1. **Explicit parameter** — if you pass `created_by` (or equivalent) directly.
2. **local.yaml** — the `user.name` field.
3. **Git config** — the value of `git config user.name`.

If none of these resolve, the operation fails with an error.

### Example

```yaml
user:
  name: alice

github:
  token: ghp_xxxxxxxxxxxxxxxxxxxx
  owner: my-org
  repo: my-project
```

---

## 4. Context Profiles

**Location:** `.kbz/roles/*.yaml` (primary) or `.kbz/context/roles/*.yaml` (legacy fallback)

Context profiles define agent roles. The `handoff` tool reads them to build
context packets — bundles of conventions, architecture knowledge, and scoped
knowledge entries that it hands to an agent at the start of a session.

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | yes | Profile identifier. Must match the filename without the `.yaml` extension. |
| `inherits` | string | no | Parent profile ID for inheritance. |
| `description` | string | yes | Short summary of what this role does. |
| `packages` | string[] | no | Code packages or directories relevant to this role. |
| `conventions` | string[] | no | Coding and process conventions the agent should follow. |
| `architecture.summary` | string | no | Architecture overview in plain prose. |
| `architecture.key_interfaces` | string[] | no | Key interfaces and files the role cares about. |

### Inheritance

A profile with `inherits: base` receives all conventions and architecture
entries from `base`, then layers its own on top. Inheritance supports multiple
levels — a grandchild inherits from its parent, which inherits from its
grandparent, and so on. Kanbanzai detects and rejects cycles.

### Example

```yaml
id: backend
inherits: developer
description: "Backend API development"

packages:
  - internal/api/
  - internal/handlers/

conventions:
  - "All endpoints must validate input before processing"
  - "Use structured logging with slog"

architecture:
  summary: "REST API using standard library net/http"
  key_interfaces:
    - "internal/api/router.go — route registration"
```

---

## 5. Environment Variables

Kanbanzai does not use environment variables for project or user configuration.
All settings come from `config.yaml` and `local.yaml`. Standard Go environment
variables (`GOBIN`, `GOPATH`) may influence binary installation paths. The
GitHub token lives in `local.yaml` rather than in an environment variable.

---

## 6. Validation

### CLI

Run the health check from the command line:

```
kanbanzai health
```

This validates:

- Config file syntax and required fields.
- Entity YAML validity across all entity types.
- Referential integrity between entities (parent–child links, dependencies).
- Knowledge entry consistency (TTL, confidence, staleness).
- Context profile schema and inheritance resolution.
- Worktree status and branch health.

### MCP

The `health` MCP tool runs the same validation and returns a structured report.
It also checks document currency and branch health, which the CLI command does
not cover.

### Common Config Errors

| Error | Cause | Fix |
|-------|-------|-----|
| Missing prefix | `prefixes` array is empty or absent. | Add at least one prefix entry. |
| Duplicate prefix | Two entries share the same `prefix` character. | Remove or rename the duplicate. |
| No active prefixes | Every prefix has `retired: true`. | Un-retire one prefix or add a new one. |
| Invalid version | `version` is missing or not a recognised value. | Set `version: "2"`. |

---

## 7. Migration

The `version` field in `config.yaml` tracks the configuration schema version.
The current version is `"2"`.

### Automatic Defaults

When Kanbanzai loads a config file, it merges default values into any missing
keys. This means:

- Upgrading from an older configuration adds newer sections
  (`branch_tracking`, `cleanup`, `knowledge`, `dispatch`, `incidents`,
  `decomposition`) with sensible defaults.
- Kanbanzai never overwrites existing settings.
- You rarely need a manual migration.

### Phase 1 → Phase 2

Projects that used Phase 1 epics can convert them to Phase 2 plans by
re-initialising the project. Config loading handles the migration
internally — it upgrades older entity formats automatically. Running
initialisation again skips entities that have already been converted.