# Kanbanzai: Guide for Viewer Agents

This document briefs AI agents building a **viewer** — a project that reads and displays Kanbanzai repository state. It covers what Kanbanzai is, how its data is structured, how to read it safely, and the pitfalls to watch for.

---

## 1. What Kanbanzai Is

Kanbanzai is a Git-native project workflow system. It tracks plans, features, tasks, bugs, decisions, documents, knowledge entries, and human checkpoints as plain YAML files inside a `.kbz/` directory at the root of a Git repository. There is no external database. All canonical state is version-controlled alongside the project's source code.

Kanbanzai supports human-AI collaborative development. Humans express intent through documents (designs, specifications, and dev plans) and decisions. AI agents execute: decomposing work, implementing tasks, and tracking status. The MCP (Model Context Protocol) server is the primary write interface. The `.kbz/` directory on disk is the primary read interface.

A viewer's job is to read committed `.kbz/` state and present it to humans — project managers, designers, and developers — who want to see project progress without using the MCP server or CLI directly.

---

## 2. Technology Stack

Kanbanzai is written in Go. There is a single binary, `kanbanzai`, with two modes:

- **`kanbanzai serve`** — runs as an MCP server (the write interface for AI agents)
- **`kanbanzai <subcommand>`** — a CLI with ~20 subcommands (`status`, `next`, `entity`, `doc`, `health`, etc.) for human use in the terminal

The getting-started guide suggests creating a symlink `ln -s ~/go/bin/kanbanzai ~/go/bin/kbz` for shorter typing, and the CLI usage text says `kbz <command>`, but there is no separate `kbz` binary — it is just a convenience alias for the same `kanbanzai` executable.

All core logic lives under `internal/` and is not importable by external Go programs. However, there is **an exported Go package specifically for external consumers**: `kbzschema`. More on this in §8.

The state format is YAML. Documents are Markdown. Timestamps are RFC 3339. IDs use **TSID13s** — 13-character time-sorted identifiers (except for Plans, which use a custom prefix+number+slug format). From 1.0 onwards, the schema uses semantic versioning via a `schema_version` field in `.kbz/config.yaml`. Pre-1.0 repositories do not have this field (see §13).

---

## 3. The `.kbz` Directory Layout

Every Kanbanzai-managed repository has this structure at the repo root:

```
.kbz/
├── config.yaml              # Project configuration and prefix registry
├── .init-complete           # Sentinel file confirming successful init
├── state/                   # All entity YAML files (the source of truth)
│   ├── plans/               # Plan entities
│   ├── features/            # Feature entities
│   ├── tasks/               # Task entities
│   ├── bugs/                # Bug entities
│   ├── decisions/           # Decision entities
│   ├── documents/           # Document metadata records
│   ├── knowledge/           # Knowledge entries
│   ├── checkpoints/         # Human checkpoint records
│   ├── incidents/           # Incident entities (created on demand)
│   ├── worktrees/           # Worktree tracking (LOCAL ONLY — see §5)
├── index/                   # Document intelligence index (derived, not canonical)
├── cache/                   # SQLite cache (derived, not canonical)
├── roles/                   # Context role profiles (YAML)
├── skills/                  # Skill definitions (SKILL.md per skill)
├── stage-bindings.yaml      # Maps workflow stages to roles and skills
├── local.yaml               # Per-machine settings (NEVER committed)
└── context/
    └── roles/               # Legacy context role profiles
```

Outside `.kbz/`, Kanbanzai also manages:

- **`.agents/skills/`** — Skill files (YAML frontmatter + Markdown body) for AI agent onboarding
- **`work/`** (or other configured roots) — the actual Markdown documents that document records point to

---

## 4. The Public Schema Boundary

This is the most important concept for a viewer. Kanbanzai defines a **public schema** — the subset of `.kbz/` that external tools can safely read. **Not everything under `.kbz/` is part of it.**

### Part of the public schema (safe and stable to read)

| Path | Contents |
|------|----------|
| `.kbz/config.yaml` | Project configuration: schema version, prefix registry |
| `.kbz/state/plans/` | Plan entity records |
| `.kbz/state/features/` | Feature entity records |
| `.kbz/state/tasks/` | Task entity records |
| `.kbz/state/bugs/` | Bug entity records |
| `.kbz/state/decisions/` | Decision entity records |
| `.kbz/state/documents/` | Document metadata records |
| `.kbz/state/knowledge/` | Knowledge entry records |
| `.kbz/state/checkpoints/` | Human checkpoint records |
| `.kbz/state/incidents/` | Incident entity records |
| `work/` (or configured roots) | The Markdown documents themselves |

### NOT part of the public schema (do not read or rely on)

| Path | Why |
|------|-----|
| `.kbz/state/worktrees/` | Local machine state — filesystem paths are meaningless to a remote reader |
| `.kbz/index/` | Derived document intelligence index — rebuilt on demand, not committed |
| `.kbz/cache/` | Derived SQLite cache — rebuilt on demand, not committed |
| `.kbz/local.yaml` | Per-machine settings — never committed to Git |

### Committed but not formally covered by the public schema spec

| Path | Notes |
|------|-------|
| `.kbz/context/roles/` | Legacy context role profiles (YAML). These ARE committed to Git and are readable, but the public schema interface specification does not formally cover them. They define agent dispatch roles (e.g. `backend`, `reviewer`) and are unlikely to be useful to a viewer, but they are safe to read if you want them. |
| `.kbz/roles/` | Role profiles (YAML). Committed to Git and readable. Define identity, vocabulary, and anti-patterns for each agent role. Not covered by the public schema spec but safe to read. |
| `.kbz/skills/` | Skill definitions (`SKILL.md` per skill directory). Committed to Git and readable. Define procedures and checklists for specific task types. Not covered by the public schema spec but safe to read. |
| `.kbz/stage-bindings.yaml` | Maps workflow stages to roles, skills, and prerequisites. Committed to Git and readable. Not covered by the public schema spec but safe to read. |

### Rule of thumb

If it is committed to Git, it is generally safe to read. If it is in `.gitignore` or derived/regenerable, it is not part of the public contract. The formal public schema covers everything in `.kbz/state/` (except `worktrees/`) plus `config.yaml`. The viewer should read from its own Git clone and should never need access to non-committed state.

---

## 5. Entity Types at a Glance

### 5.1 The Entity Hierarchy

```
Plan (top-level organising unit)
 └── Feature (deliverable unit of work, document-driven lifecycle)
      └── Task (atomic unit of work, picked up by agents)

Bug (standalone defect tracker, linked to features/tasks)
Decision (architectural/process choice, linked to affected entities)
Incident (production issue tracker, linked to features/bugs)
DocumentRecord (metadata about a Markdown file)
KnowledgeEntry (reusable project knowledge)
HumanCheckpoint (decision point requiring human input)
```

### 5.2 Entity Summary Table

| Entity | ID Format | Storage Path | Filename Pattern |
|--------|-----------|--------------|------------------|
| Plan | `{prefix}{number}-{slug}` (e.g. `P1-my-plan`) | `.kbz/state/plans/` | `{id}.yaml` |
| Feature | `FEAT-{TSID13}` | `.kbz/state/features/` | `{id}-{slug}.yaml` |
| Task | `TASK-{TSID13}` | `.kbz/state/tasks/` | `{id}-{slug}.yaml` |
| Bug | `BUG-{TSID13}` | `.kbz/state/bugs/` | `{id}-{slug}.yaml` |
| Decision | `DEC-{TSID13}` | `.kbz/state/decisions/` | `{id}-{slug}.yaml` |
| Incident | `INC-{TSID13}` | `.kbz/state/incidents/` | `{id}-{slug}.yaml` |
| DocumentRecord | `{owner}/{type}-{slug}` | `.kbz/state/documents/` | `{id-with-/→--}.yaml` |
| KnowledgeEntry | `KE-{TSID13}` | `.kbz/state/knowledge/` | `{id}.yaml` |
| HumanCheckpoint | `CHK-{TSID13}` | `.kbz/state/checkpoints/` | `{id}.yaml` |

**TSID13s** (Time-Sorted IDs) are 13-character, Crockford base32 encoded identifiers with a 48-bit millisecond timestamp (10 chars) and a 15-bit random component (3 chars). Example: `FEAT-01KMKRQRRX3CC` — the portion after the prefix-hyphen (`01KMKRQRRX3CC`) is the TSID13. They sort lexicographically in chronological order. (Note: `docs/schema-reference.md` calls these "ULIDs" — that is inaccurate; they are TSID13s, not 26-character ULIDs.)

**Plan IDs** are special: a single-character prefix from the prefix registry, a sequence number, and a slug. Example: `P3-kanbanzai-1.0`.

**Document Record IDs** contain a slash: `FEAT-01ABC/design-my-feature`. On disk, the slash becomes `--`: `FEAT-01ABC--design-my-feature.yaml`.

### 5.3 Filename Lookup

This is a practical concern for a viewer. Entities with slugs are stored as `{id}-{slug}.yaml`, but you often only have the ID. The correct lookup strategy is:

1. Scan the directory for files whose name starts with `{id}-` and ends with `.yaml`.
2. For entities without slugs (KnowledgeEntry, HumanCheckpoint), the filename is simply `{id}.yaml`.
3. For Plans, the filename is `{id}.yaml` — the slug is part of the ID itself.
4. For Document Records, replace `/` with `--` in the ID, then append `.yaml`.

The `kbzschema` Go package (§8) handles all of this for you if you are writing in Go.

---

## 6. Lifecycle State Machines

Every entity with a status field follows a defined state machine. Valid transitions are strict — the Kanbanzai server rejects anything not listed. A viewer does not enforce transitions but should understand them for display purposes (colouring, progress calculation, and filtering).

### Plan
```
proposed → designing → active → reviewing → done
```
Back-transition: reviewing → active.
From any non-terminal state: → superseded, → cancelled.
Note: `done` is non-terminal — a done plan can transition to superseded or cancelled.

### Feature (document-driven)
```
proposed → designing → specifying → dev-planning → developing → reviewing → done
```
From any non-terminal state: → superseded, → cancelled.
Additional transitions: reviewing → needs-rework, needs-rework → developing or → reviewing.

Backward transitions occur when documents are superseded (e.g. specifying → designing if the design document is superseded). This is important for the viewer: features can move backwards.

### Task
```
queued → ready → active → done
                  ├────→ needs-review → done
                  │                   → needs-rework → active
                  ├────→ blocked → active
                  └────→ ready (unclaim/crash recovery)
```
Terminal states: done, not-planned, duplicate.

Tasks auto-promote from `queued` to `ready` when all their `depends_on` tasks reach a terminal state.

### Bug
```
reported → triaged → reproduced → planned → in-progress → needs-review → verified → closed
```
Also: triaged → cannot-reproduce, needs-review → needs-rework → in-progress.
Terminal states: closed, duplicate, not-planned.

### Decision
```
proposed → accepted
```
Terminal states: rejected, superseded.

### Incident
```
reported → triaged → investigating → root-cause-identified → mitigated → resolved → closed
```
Back-transitions: root-cause-identified → investigating, mitigated → investigating.
From any non-terminal state: → closed (early-close).

### Document Record
```
draft → approved → superseded
```

### Knowledge Entry
```
contributed → confirmed  (auto: use_count ≥ 3, miss_count = 0)
           → disputed    (via flagging)
           → retired     (auto: miss_count ≥ 2, or manual, or TTL expiry)
confirmed  → stale       (staleness detection)
stale      → confirmed   (recovery — re-confirmed after review)
           → retired     (no longer relevant)
```

---

## 7. YAML Serialisation Rules

All YAML files written by Kanbanzai follow strict, deterministic conventions. A viewer reading these files can rely on:

1. **Block style only.** No flow style (`{}` or `[]`). Lists use `- item` form.
2. **Minimal quoting.** Strings are unquoted unless YAML syntax demands it. When quoting is needed, double quotes are used.
3. **Deterministic field order.** Fields always appear in the same order as defined in the struct/schema. This order is fixed.
4. **UTF-8, LF line endings, trailing newline.**
5. **No advanced YAML features.** No tags (`!!str`), no anchors (`&name`), no aliases (`*name`), and no multi-document streams.
6. **Omit empty optionals.** Optional fields with zero/nil/empty values are omitted entirely rather than written as empty.
7. **Timestamps are always double-quoted.** Example: `created: "2026-03-26T00:28:22Z"`. This prevents YAML parser date coercion. All timestamps are RFC 3339 UTC.
8. **Estimates are nullable numbers.** When present, they are from the Modified Fibonacci scale: `0, 0.5, 1, 2, 3, 5, 8, 13, 20, 40, 100`. When absent, the field is omitted entirely (not set to zero or null). In the Go `kbzschema` types, this is represented as `*float64`.

### What this means for a viewer

- You can use any standard YAML parser — there are no exotic YAML features to handle.
- Missing fields mean "not set", never "empty string". Treat absence and `null` identically.
- Timestamps are strings. Parse them as RFC 3339 if you need date arithmetic. They are always UTC.
- Enumerated string fields (status, severity, priority, and type) may contain values you don't recognise — new values are added in minor schema versions. **Do not fail on unknown values.** Display the raw string.

---

## 8. The `kbzschema` Go Package

If your viewer is written in Go, you do not need to roll your own YAML parsing. Kanbanzai exports a public Go package specifically for external consumers:

```
import "github.com/sambeau/kanbanzai/kbzschema"
```

This package provides:

### 8.1 Type Layer

Go structs for every committed entity type, with both `yaml:` and `json:` tags:

- `kbzschema.Plan`
- `kbzschema.Feature`
- `kbzschema.Task`
- `kbzschema.Bug`
- `kbzschema.Decision`
- `kbzschema.DocumentRecord`
- `kbzschema.KnowledgeEntry`
- `kbzschema.HumanCheckpoint`
- `kbzschema.ProjectConfig`
- `kbzschema.PrefixEntry`

All enumerated values have exported constants (e.g. `kbzschema.TaskStatusReady`, `kbzschema.SeverityHigh`, `kbzschema.DocTypeDesign`).

### 8.2 Query Layer (Reader)

A read-only `Reader` that handles directory walking, filename matching, and cross-reference resolution:

```go
reader, err := kbzschema.NewReader("/path/to/repo")

plan, err := reader.GetPlan("P3-kanbanzai-1.0")
features, err := reader.ListFeaturesByPlan("P3-kanbanzai-1.0")
task, err := reader.GetTask("TASK-01KMNA39KTWW4")
tasks, err := reader.ListTasksByFeature("FEAT-01KMKRQRRX3CC")
bug, err := reader.GetBug("BUG-01KMKA1KEFYX0")
bugs, err := reader.ListBugs()
doc, err := reader.GetDocumentRecord("FEAT-01KMKRQRRX3CC/design-init-command")
docs, err := reader.ListDocumentRecords()
content, driftWarning, err := reader.GetDocumentContent("FEAT-01KMKRQRRX3CC/design-init-command")
```

The Reader is safe to call concurrently — it holds no mutable state.

### 8.3 JSON Schema Generation

For non-Go consumers, `kbzschema.GenerateSchema()` produces a JSON Schema document covering all entity types. The schema version is `1.0.0` and is embedded in the output. The JSON Schema is derived from the Go types — the Go types are the source of truth.

### 8.4 Current Coverage Gaps

As of the current implementation, the `kbzschema` package does **not** yet cover:

- **Incident** — the `INC-` entity type has no struct or Reader method in `kbzschema`
- **Worktree** — deliberately excluded (local machine state, not part of public schema)
- **Plan and Feature missing `reviewing` status constant** — the `reviewing` lifecycle state is not represented in the exported status constants
- **Feature missing `needs-rework` status constant** — the `needs-rework` lifecycle state is not represented in the exported status constants
- **Missing `plan` and `retrospective` document type constants** — these document types exist but have no `kbzschema.DocType*` constants
- **Plan `Title` yaml tag should be `name`** — the Go struct's yaml tag is misaligned with the internal model (which uses `name`)
- **Bug `Title` yaml tag should be `name`** — same misalignment as Plan
- **Entity `name` field missing from Feature, Task, Decision structs** — these entity types have a `name` field in the internal model but the `kbzschema` structs do not include it

For Incidents, you would need to parse the YAML directly until `kbzschema` adds coverage (expected in a minor version). The schema-reference document (`docs/schema-reference.md`) defines the full field table.

The Reader also currently lacks methods for Decisions, KnowledgeEntries, Checkpoints, and Incidents — it covers Plans, Features, Tasks, Bugs, and DocumentRecords. For the missing types, you can use the exported structs with standard `yaml.Unmarshal` and your own directory walking.

### 8.5 If You're Not Using Go

If your viewer is written in TypeScript, Python, Rust, or anything else:

1. Use `kbzschema.GenerateSchema()` to produce a JSON Schema, or find the published schema in the GitHub Release assets.
2. Use the JSON Schema to generate types in your language, or write them manually using `docs/schema-reference.md` as the reference.
3. Implement your own directory walker following the filename patterns in §5.3.
4. Parse with any standard YAML library — the files use no exotic features.

---

## 9. Referential Integrity and Cross-References

Entities reference each other via ID strings. These are the key relationships a viewer needs to resolve:

### Parent hierarchy
- Feature → Plan: `feature.parent` is a Plan ID
- Task → Feature: `task.parent_feature` is a Feature ID
- Task → Task: `task.depends_on` is a list of Task IDs (dependency graph)

### Document linkage
- Feature → DocumentRecord: `feature.design`, `feature.spec`, `feature.dev_plan` are Document Record IDs
- Plan → DocumentRecord: `plan.design` is a Document Record ID
- DocumentRecord → file: `document_record.path` is a relative path from the repo root to the Markdown file

### Bug linkage
- Bug → Feature: `bug.origin_feature`
- Bug → Task: `bug.origin_task`
- Bug → Bug: `bug.duplicate_of`

### Incident linkage
- Incident → Feature: `incident.affected_features` (list)
- Incident → Bug: `incident.linked_bugs` (list)
- Incident → DocumentRecord: `incident.linked_rca`

### Decision linkage
- Decision → entities: `decision.affects` (list of any entity IDs)

### Supersession chains
- Plans, Features, Decisions, and Document Records form supersession chains via `supersedes` / `superseded_by` fields.

### Important: references can be broken

Kanbanzai validates references at creation time but does not cascade updates. A referenced entity may have transitioned to a terminal state or (in rare cases) may not exist. Your viewer must handle missing targets gracefully — display "unknown" or a broken-link indicator rather than crashing.

---

## 10. Deriving Progress and Metrics

A viewer can compute useful metrics purely from committed entity state:

### Task completion by feature
```
done_tasks = count tasks where parent_feature == feature.id AND status IN (done)
total_tasks = count tasks where parent_feature == feature.id AND status NOT IN (not-planned, duplicate)
completion_pct = done_tasks / total_tasks * 100
```

### Feature completion by plan
Same pattern: count features where `parent == plan.id`, compare done vs. total (excluding cancelled/superseded).

### Story point rollup
Sum `estimate` fields on tasks for a feature total. Sum feature estimates for a plan total. Not all entities have estimates — treat missing estimates as zero or "unestimated" in the UI.

### Blocked work
Tasks in `blocked` status, or tasks in `queued` status whose `depends_on` list contains non-terminal tasks.

### Active work
Tasks in `active` status. The `claimed_at` and `dispatched_to` fields tell you when and to which agent role the task was dispatched.

### Pending human decisions
Checkpoints where `status == "pending"` — these are questions awaiting human response.

---

## 11. Reading Documents (Markdown Files)

**Document Records** in `.kbz/state/documents/` are metadata. The actual content is a Markdown file elsewhere in the repository, referenced by the `path` field.

### Following the path

```yaml
# From .kbz/state/documents/FEAT-01KMKRQRRX3CC--design-init-command.yaml
id: FEAT-01KMKRQRRX3CC/design-init-command
path: work/design/init-command.md
type: design
title: kanbanzai init Command Design
status: approved
content_hash: 15e486b2966498eefcf00170e34b900f14e18a02c6be0a2b00dec731beab84fc
```

The `path` field is relative to the repository root. To read the document, resolve it against the repo root directory.

### Drift detection

The `content_hash` field is a SHA-256 hex digest of the file contents at the time the document was registered or approved. If you hash the current file and it differs from `content_hash`, the document has drifted — it was modified after registration/approval. This is worth surfacing in the viewer as a warning (e.g. "modified since approval").

An empty `content_hash` means no hash was recorded — typically for older records created before hashing was implemented.

### Markdown conventions

Kanbanzai documents are **plain Markdown**. There is no special Markdown dialect or required structure enforced by the file format.

However, **skill files** (`.agents/skills/*/SKILL.md`) use **YAML frontmatter**:

```yaml
---
name: kanbanzai-documents
description: >
  Use this skill whenever you create, edit, register, or approve any document...
metadata:
  kanbanzai-managed: "true"
  version: "0.2.0"
---

# SKILL: Kanbanzai Documents
...
```

The frontmatter is delimited by `---` on its own line. Standard YAML frontmatter parsing applies. The `metadata.kanbanzai-managed: "true"` marker identifies files managed by `kanbanzai init`.

For regular project documents (designs, specs, plans), there is **no required frontmatter**. They are plain Markdown. Some may have informal metadata at the top in a table format:

```markdown
| Document | My Feature Specification |
|----------|--------------------------|
| Status   | Draft                    |
| Created  | 2026-03-26               |
```

But this is a convention, not a schema requirement. Do not parse it as structured data — the Document Record YAML is the authoritative metadata.

### Document types and their directories

| Type | Typical Directory | Purpose |
|------|-------------------|---------|
| `design` | `work/design/` | Architecture, vision, and approach |
| `specification` | `work/spec/` | Acceptance criteria, binding contracts |
| `dev-plan` | `work/dev/` | Feature implementation plans, task breakdowns |
| `research` | `work/research/` | Analysis, exploration |
| `report` | `work/reports/` or `work/reviews/` | Audit reports, reviews |
| `policy` | `work/design/` | Process and governance documents |
| `rca` | (varies) | Root cause analysis |
| `plan` | `work/plan/` | Project planning: roadmaps, scope, and decision logs |
| `retrospective` | `work/retro/` | Retrospective synthesis documents |

The configured document roots are in `.kbz/config.yaml` under `document_roots` (if present). But the `path` field in each Document Record is the authoritative location — always use it.

---

## 12. Concurrency and Safe Reading

### Can I read while Kanbanzai is writing?

**Yes, with caveats.** Kanbanzai uses atomic writes (write to temp file, then rename) to prevent partial reads. You will never see a half-written YAML file. You will see either the old version or the new version.

However, there is no cross-file transactional consistency. If you read the features directory while Kanbanzai is creating a new task and updating its parent feature, you might see the new task but the old feature (or vice versa). For a viewer, this is acceptable — the next read will be consistent.

### The recommended model: your own Git clone

A viewer should read from **its own Git clone**, not from the same working directory as a running Kanbanzai server. This eliminates all concurrency concerns:

- The viewer's state is as current as its last `git pull`.
- No shared mutable state between the viewer and any Kanbanzai process.
- Uncommitted work-in-progress is intentionally invisible to the viewer.

### Freshness strategies

| Deployment | Strategy |
|------------|----------|
| Hosted/team viewer | GitHub webhooks — pull on push events (seconds of latency) |
| Local viewer | Filesystem watching on the `.git` directory (detects pull/fetch) |
| Fallback | Periodic `git pull` polling (rate-limited, higher latency) |

---

## 13. Schema Versioning

### The `schema_version` field

At 1.0, `config.yaml` has a `schema_version` field (semver). This is the public compatibility signal:

```yaml
version: "2"
schema_version: "1.0.0"
prefixes:
  - prefix: P
    name: Plan
```

The `version` field (`"2"`) is a legacy internal version counter from development phases. **Ignore it.** Read `schema_version` for compatibility decisions.

**Important:** Pre-1.0 repositories (including current Kanbanzai development) do not have a `schema_version` field at all. The `config.yaml` may contain only `version` and `prefixes`. Your viewer must handle this case — treat a missing `schema_version` as "pre-1.0" and read the YAML files best-effort using the current schema-reference documentation.

### Compatibility guarantees

| Version bump | What changes | Your viewer's obligation |
|--------------|--------------|--------------------------|
| Patch (1.0.x) | Bug fixes, doc clarifications | Nothing — fully compatible |
| Minor (1.x.0) | New optional fields, new entity types, new enum values | Ignore unknown fields, pass through unknown enum values |
| Major (x.0.0) | Removed/renamed fields, changed semantics | You need to update |

### The golden rule

**Never fail on unknown values.** New status values, new entity types, and new fields — display them as raw strings. A viewer built for schema `1.0.0` must work with any `1.x.x` schema without crashing.

---

## 14. What You Can Safely Assume

- Every entity directory under `.kbz/state/` contains zero or more `.yaml` files, one per entity.
- Every YAML file has an `id` field as the first field.
- Every entity with a lifecycle has a `status` field containing a lowercase, hyphenated string.
- Timestamps are always RFC 3339 UTC strings, always double-quoted in YAML.
- IDs are immutable — once created, an entity's ID never changes.
- File names encode the ID (and sometimes slug) — you can extract the ID from the filename.
- `created` and `updated` timestamps are present on most entity types and are auto-managed.
- The `state/` directory is the single source of truth. The `cache/` and `index/` directories are derived and may not exist.

---

## 15. What You Cannot Safely Assume

- **Field presence.** Optional fields may be absent. Do not assume every task has an `estimate`, every feature has a `design`, or every bug has a `reproduction`.
- **Enum exhaustiveness.** New status values, severities, priorities, bug types, and document types may appear in future minor versions.
- **Referential completeness.** A `parent_feature` ID on a task should point to a real feature — but the feature might have been superseded or cancelled. Always handle "not found" gracefully.
- **Entity counts.** A plan may have zero features. A feature may have zero tasks. Knowledge entries may not exist at all.
- **Document file existence.** A Document Record's `path` may point to a file that has been deleted or moved. Check before reading.
- **Schema version presence.** Pre-1.0 repositories may lack `schema_version` in `config.yaml`.
- **Specific field order for reading.** While Kanbanzai writes fields in deterministic order, your YAML parser should not depend on field order.
- **Worktree availability.** Worktree records are local and are not committed. Do not look for them.

---

## 16. Practical Tips

### Building a tree view

The natural display hierarchy is:

```
Plan
 ├── Feature
 │    ├── Task
 │    ├── Task (depends on above)
 │    └── Task
 ├── Feature
 │    └── ...
 └── (Orphan bugs, decisions linked via `affects`)
```

To build this:
1. Load all Plans.
2. For each Plan, find Features where `feature.parent == plan.id`.
3. For each Feature, find Tasks where `task.parent_feature == feature.id`.
4. Bugs are standalone — display them separately, optionally linked to features via `origin_feature`.
5. Decisions are standalone — display them separately, optionally linked via `affects`.

### Sorting

- TSID13s sort lexicographically in chronological order. Sorting entity IDs by string comparison gives you creation-time order.
- Plans sort by prefix + number: `P1-...`, `P2-...`, `P3-...`.
- Tasks within a feature can be sorted by ID (creation order) or by dependency graph (topological sort on `depends_on`).

### Colouring by status

Suggested groupings for colour coding:

| Colour | Statuses |
|--------|----------|
| Grey | proposed, queued, draft |
| Blue | designing, specifying, dev-planning, ready, planned, contributed |
| Yellow | active, in-progress, investigating, developing |
| Orange | blocked, needs-review, needs-rework, disputed, pending |
| Green | done, closed, verified, approved, accepted, confirmed, resolved |
| Red | cancelled, not-planned, rejected, duplicate, retired, cannot-reproduce |
| Purple | superseded |

### Document Record → Markdown rendering

The `path` field in a Document Record gives you the Markdown file. Render it with any Markdown library. There are no Kanbanzai-specific Markdown extensions to handle.

Check the `content_hash` against the file's SHA-256 to detect drift and badge the document accordingly (e.g. "approved", "approved — modified since", "draft").

### Estimation display

The Modified Fibonacci scale used for estimates: `0, 0.5, 1, 2, 3, 5, 8, 13, 20, 40, 100`. Features and plans roll up estimates from their children. You can compute these rollups yourself by summing task estimates for a feature, and feature estimates for a plan.

---

## 17. Example Entity YAML

### Plan

```yaml
id: P3-kanbanzai-1.0
slug: kanbanzai-1.0
name: Kanbanzai 1.0
status: active
summary: "Make Kanbanzai installable and usable by projects other than itself."
design: PROJECT/design-kanbanzai-10
created: "2026-03-26T00:28:22Z"
created_by: sambeau
updated: "2026-03-26T14:48:05Z"
```

### Feature

```yaml
id: FEAT-01KMKRQRRX3CC
slug: init-command
name: Init command
parent: P3-kanbanzai-1.0
status: developing
summary: "The kanbanzai init command."
design: FEAT-01KMKRQRRX3CC/design-init-command
spec: FEAT-01KMKRQRRX3CC/specification-init-command
dev_plan: FEAT-01KMKRQRRX3CC/dev-plan-init-command
created: "2026-03-26T00:29:32Z"
created_by: sambeau
```

### Task

```yaml
id: TASK-01KMNA39KTWW4
parent_feature: FEAT-01KMKRQRRX3CC
slug: init-command-skeleton
name: Init command skeleton
summary: "Implement kanbanzai init CLI subcommand skeleton with flag parsing."
status: done
files_planned:
  - cmd/kanbanzai/init_cmd.go
  - cmd/kanbanzai/main.go
completed: "2026-03-26T15:13:30Z"
claimed_at: "2026-03-26T15:03:55Z"
dispatched_to: backend
dispatched_at: "2026-03-26T15:03:55Z"
dispatched_by: agent/init-command
completion_summary: "Implemented kanbanzai init as a CLI subcommand."
verification: "go build ./..., go test -race ./internal/kbzinit/... (37 tests pass)"
```

### Document Record

```yaml
id: FEAT-01KMKRQRRX3CC/design-init-command
path: work/design/init-command.md
type: design
title: kanbanzai init Command Design
status: approved
owner: FEAT-01KMKRQRRX3CC
approved_by: sambeau
approved_at: "2026-03-26T10:18:04Z"
content_hash: 15e486b2966498eefcf00170e34b900f14e18a02c6be0a2b00dec731beab84fc
created: "2026-03-26T10:13:05Z"
created_by: sambeau
updated: "2026-03-26T10:18:04Z"
```

### Human Checkpoint

```yaml
id: CHK-01KMDEF456GHI
question: "Should we split the auth feature into separate OAuth and API-key features?"
context: "The auth feature has grown to 13 tasks spanning two distinct concerns."
orchestration_summary: "Decomposing FEAT-01KMXYZ into tasks."
created_by: agent/orchestrator
status: pending
created_at: "2026-03-27T10:30:00Z"
```

---

## 18. Key Reference Documents

If you need deeper detail, these are the authoritative sources within the Kanbanzai repository:

| Document | What it covers |
|----------|----------------|
| `docs/schema-reference.md` | Complete field tables, lifecycle state machines, ID formats, and referential integrity rules |
| `work/design/public-schema-interface.md` | Design rationale for the public schema boundary, versioning, and the Go interface |
| `work/spec/public-schema-interface.md` | Formal specification of the schema interface, compatibility policy, and acceptance criteria |
| `docs/getting-started.md` | Installation, initialisation, and first-use guide |
| `docs/configuration-reference.md` | Configuration file format details |
| `docs/workflow-overview.md` | High-level workflow concepts |

---

## 19. Quick-Start Checklist for a Viewer Project

1. **Clone the target repository** (or point at a local clone).
2. **Check for `.kbz/config.yaml`** — if it doesn't exist, this isn't a Kanbanzai-managed repo.
3. **Read `schema_version`** from `config.yaml`. If absent, treat as pre-1.0.
4. **If using Go**: import `github.com/sambeau/kanbanzai/kbzschema`, create a `Reader`, and use its methods.
5. **If not using Go**: walk `.kbz/state/` subdirectories, parse YAML files with a standard library, and use `docs/schema-reference.md` as your field reference.
6. **Build the entity hierarchy**: Plans → Features → Tasks. Bugs, Decisions, and Incidents are standalone with cross-references.
7. **Resolve document paths**: DocumentRecord `.path` → Markdown file relative to repo root.
8. **Handle missing data gracefully**: absent fields, broken references, and unknown enum values.
9. **Never write to `.kbz/`** — the viewer is read-only. All writes go through the MCP server or CLI.
10. **Pull regularly** if you need fresh data. Webhooks or filesystem watching for real-time updates.