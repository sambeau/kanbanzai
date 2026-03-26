# Public Schema Interface Design

- Status: draft design
- Purpose: define the .kbz schema as a stable public interface for external consumers, including the viewer and any other tools that read Kanbanzai workflow state
- Date: 2026-05-31
- Related:
  - `work/design/kanbanzai-1.0.md` §7
  - `work/design/init-command.md` §6
  - `work/design/document-centric-interface.md`

---

## 1. Purpose

Through Phases 1–4, the `.kbz` directory structure and YAML file formats have been an implementation detail: tested, versioned, and stable, but not documented as a contract. Internal consumers — the MCP server, the CLI, the service layer — can be updated in lockstep when the format changes.

At 1.0, external consumers exist. The viewer project will read `.kbz/` directly. Other tools may follow. A format that changes without notice is no longer acceptable.

This document defines what the public schema is, what it covers, how it is versioned, and what guarantees it makes to external consumers.

---

## 2. Defining the Public Schema

### 2.1 The Simplest Possible Definition

The public schema is **committed state**.

This definition follows directly from how the viewer works: the viewer reads from its own git clone of the repository. Git only contains what has been committed and pushed. Therefore the public schema is exactly the set of files and formats that Kanbanzai commits to the repository — nothing more, nothing less.

This definition is clean, principled, and self-enforcing. If a file is never committed, it is not part of the public schema, regardless of whether it lives under `.kbz/`.

### 2.2 What Is Committed

The following are committed to the repository and are part of the public schema:

| Path | Contents |
|---|---|
| `.kbz/config.yaml` | Project configuration: schema version, prefix registry, document roots |
| `.kbz/state/plans/` | Plan entity records |
| `.kbz/state/features/` | Feature entity records |
| `.kbz/state/tasks/` | Task entity records |
| `.kbz/state/bugs/` | Bug entity records |
| `.kbz/state/decisions/` | Decision entity records |
| `.kbz/state/documents/` | Document metadata records |
| `.kbz/state/knowledge/` | Knowledge entry records |
| `.kbz/state/checkpoints/` | Human checkpoint records |
| `work/` (or configured roots) | The documents themselves (markdown files) |

### 2.3 What Is Not Committed

The following are explicitly excluded from the public schema:

| Path | Reason |
|---|---|
| `.kbz/state/worktrees/` | Local machine state — paths are filesystem-specific, meaningless to a remote reader |
| `.kbz/index/` | Derived document intelligence index — rebuilt on demand, not canonical |
| `.kbz/cache/` | Derived SQLite cache — rebuilt on demand, not canonical |
| `.kbz/local.yaml` | Per-machine settings — explicitly excluded from git by design |

Worktree records deserve specific mention. They record the relationship between an entity and a local git worktree directory, including a filesystem path. A viewer reading a remote clone would see paths that do not exist on its machine. Worktree presence is also not meaningful to the viewer's audience (designers, managers): whether a feature is being actively developed is visible from the feature's status field, not from worktree presence.

---

## 3. Schema Versioning

### 3.1 Two Version Fields

The current `config.yaml` carries a single `version` field (`version: "2"`) that has tracked internal format revisions across development phases. This field has served an internal purpose but is not suitable as a public compatibility signal: it does not follow semantic versioning, and its meaning is not documented.

At 1.0, `config.yaml` gains a dedicated `schema_version` field using semantic versioning:

```yaml
version: "2"
schema_version: "1.0.0"
prefixes:
  - prefix: P
    name: Plan
documents:
  roots:
    - path: work/design
      default_type: design
    ...
```

The `version` field is preserved for backwards compatibility with existing tooling. The `schema_version` field is the public compatibility signal. External consumers should read `schema_version`.

### 3.2 Binary Version vs. Schema Version

The schema version and the Kanbanzai binary version are independent. A binary release that adds a new CLI command, improves error messages, or changes the MCP server behaviour does not change the schema. Only changes to the files committed to the repository constitute a schema change.

This means a `.kbz/` directory at schema `1.0.0` may be written by any binary version that implements schema `1.x.x`. Consumers should depend on the schema version, not the binary version.

### 3.3 Compatibility Policy

| Version component | Meaning | Example |
|---|---|---|
| **Patch** (1.0.x) | Bug fixes, clarifications, no field changes | Correcting a documented constraint that was wrong |
| **Minor** (1.x.0) | Additive changes — backward compatible | New optional field, new entity type, new valid status value |
| **Major** (x.0.0) | Breaking changes | Removed field, renamed field, changed field semantics, changed lifecycle rules |

External consumers built against schema `1.0.0` are guaranteed to work with any `1.x.x` schema without modification. They may encounter new optional fields or new enumerated values they do not recognise and must handle these gracefully (see §3.4).

### 3.4 Unknown Values

Enumerated fields (entity status, document type, severity, etc.) can gain new valid values in minor versions. External consumers must not fail on unrecognised values. The recommended behaviour is to display or pass through the raw string value, treating it as an unknown-but-valid state.

### 3.5 Binary Behaviour at Version Boundaries

A Kanbanzai binary encountering a `.kbz/` directory will:

- **Schema version newer than binary understands**: refuse to operate and direct the user to upgrade
- **Schema version older than binary understands**: offer migration via `kanbanzai migrate` before proceeding
- **Schema version equal**: proceed normally

---

## 4. The Go Interface

### 4.1 Principle: Types and a Query Layer

External Go consumers — including the viewer — need more than raw YAML parsing. Rendering a feature page requires knowing the feature's parent plan, its tasks, its linked documents, its current status. Without a query layer, every consumer reimplements directory walking and cross-reference resolution.

The Go interface therefore provides two levels:

**Type layer**: Go structs that correspond to each entity type, with field definitions, YAML tags, and valid value constants. Sufficient for consumers that want to read and write the raw files themselves.

**Query layer**: Higher-level access patterns that traverse the directory structure, resolve cross-references, and return populated results. Consumers call `store.GetFeature(id)` or `store.ListFeaturesByPlan(planID)` rather than navigating the filesystem.

The query layer is read-oriented. It does not include write operations, lifecycle enforcement, referential integrity checking, or ID allocation. Those remain internal to the Kanbanzai binary.

The exact API shape — method names, return types, error handling conventions — is an implementation decision deferred to the specification.

### 4.2 Document Records and Document Content

Documents have two distinct representations, both accessible through the query layer:

**Document record**: the metadata stored in `.kbz/state/documents/` — path, type, title, status (draft/approved/superseded), content hash, owner, timestamps. This is what the query layer returns for listing and lookup operations.

**Document content**: the markdown file itself, committed to `work/` or the project's configured document roots. The path in the document record points to it. Reading content is a separate operation from reading the record.

The query layer provides both:

- `GetDocumentRecord(id)` → metadata including path, status, hash
- `GetDocumentContent(id)` → the markdown content, read from the path in the record
- `ListDocumentRecords(...)` → filtered list of metadata records

The query layer requires the repository root path to resolve document paths (which are relative to the repo root) to absolute filesystem paths. A consumer pointing the query layer at a git clone must provide the clone's root directory.

The content hash stored in the document record allows the query layer to detect drift — a document file that has changed since it was registered. When drift is detected, the query layer surfaces a warning rather than failing silently. This is the same behaviour the MCP server provides via `doc_record_get`.

### 4.3 Packaging

Whether the public types and query layer live in a separate Go module (e.g., `github.com/owner/kanbanzai-schema`) or in exported packages of the main module (e.g., `github.com/owner/kanbanzai/schema`) is an implementation decision. The design requirement is that external Go projects can import the types and query layer without depending on the entire Kanbanzai binary or its internal packages.

### 4.4 Stability Guarantee

The public Go interface follows the schema version. Breaking changes to the Go API are only made on major schema version bumps. New methods and types may be added in minor versions.

---

## 5. JSON Schema for Non-Go Consumers

For consumers not written in Go, a JSON Schema document is generated from the Go types as a build artefact and published alongside each release.

The JSON Schema covers:

- All entity types and their fields
- Required vs. optional fields
- Valid values for enumerated fields (as `enum` arrays)
- Field types and formats

The JSON Schema is regenerated on every release. Minor version releases may add new `enum` values to enumerated fields; consumers should not treat enum arrays as exhaustive.

The JSON Schema is a derived artefact, not a hand-maintained document. The Go types are the source of truth. Any discrepancy between the Go types and the JSON Schema is a bug in the generation, not an intentional divergence.

---

## 6. Notes for the Viewer

This section records design conclusions reached during discussion of the public schema, which bear directly on the viewer's architecture. These notes are intended to inform the viewer design document when that work begins.

### 6.1 The Viewer Has Its Own Clone

The viewer does not share a `.kbz/` directory with a working development environment. It reads from its own git clone of the repository. This resolves the concurrent access question entirely: there is no shared mutable state between the viewer and any running Kanbanzai process.

The viewer's state is as current as its most recent `git pull`. This is not a limitation — it is the correct model. In a git-native workflow, state that has not been committed and pushed is not ready to be viewed. Managers and designers do not need visibility into uncommitted work-in-progress.

### 6.2 Freshness: Webhooks and Filesystem Watching

The viewer's freshness depends on how often it pulls from remote. Two mechanisms are appropriate depending on deployment:

**Hosted or team-shared viewer**: GitHub webhooks. GitHub fires an HTTP POST to a configured endpoint on every push event. The viewer receives the webhook, performs a `git pull`, and updates. Latency is typically a few seconds after push. GitHub Apps provide a cleaner registration mechanism for organisation-wide deployments.

**Local viewer**: Filesystem watching on the local git repository. When git objects change (after a pull), the viewer detects the change and updates immediately. No network round-trip required.

Polling the GitHub API is a fallback for deployments where neither webhooks nor filesystem watching is practical. It is rate-limited and introduces unnecessary latency.

The choice of freshness mechanism is a viewer deployment concern, not a schema concern.

### 6.3 What the Viewer Can and Cannot See

The viewer can see all committed state (§2.2). In particular:

- **Checkpoints** are committed and visible. A manager viewing the project can see pending human decisions awaiting response. This is useful.
- **Worktrees** are not committed and not visible. Whether a feature is being actively developed is visible from the feature's `status` field.
- **In-progress uncommitted work** is invisible to the viewer by design. A task marked `active` in a worktree but not yet pushed appears as `active` with no further detail. This is correct behaviour.

### 6.4 Estimated Progress

Task completion percentage is derivable from committed entity state: count done tasks divided by total tasks for a given feature or plan. The viewer can compute and display this without any additional schema support.

If a team wants to commit richer progress signals (estimated percentage, narrative status updates) as first-class state, that is a natural future extension. It would be a new optional entity type or field, introduced as a minor schema version. It is explicitly deferred from 1.0.

### 6.5 Write Access

Write access through the viewer is a future consideration and may never be a feature in an agentic workflow. In an agentic model, agents handle all write operations through the MCP server; the viewer's audience (designers, managers) does not need to write entity state directly.

If write access is added to the viewer in a future version, it must go through the Kanbanzai service layer — not by writing YAML files directly — to preserve lifecycle enforcement and referential integrity. This is an architectural constraint, not a 1.0 concern.