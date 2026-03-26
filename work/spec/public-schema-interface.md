# Public Schema Interface Specification

| Document | Public Schema Interface Specification        |
|----------|----------------------------------------------|
| Status   | Draft                                        |
| Created  | 2026-05-31                                   |
| Updated  | 2026-05-31                                   |
| Related  | `work/design/public-schema-interface.md`     |
|          | `work/design/kanbanzai-1.0.md` §7            |

---

## 1. Purpose

This document specifies the public schema interface for Kanbanzai 1.0. It defines what constitutes the committed, versioned state of a Kanbanzai-managed repository; how that state is versioned; and how external tools and Go programs can read that state in a stable, supported way.

The public schema interface is the contract between Kanbanzai and the broader ecosystem: CI pipelines, dashboards, reporting tools, IDE extensions, and any other software that reads Kanbanzai state without going through the MCP server or CLI.

---

## 2. Goals

1. **Define the public boundary.** Clearly distinguish committed state (public schema) from derived or machine-local state (not part of the schema).
2. **Enable stable external tooling.** External programs can read entity records and document metadata without depending on Kanbanzai internals.
3. **Version the schema independently of the binary.** A binary release that only adds CLI commands or improves error messages does not force a schema version bump.
4. **Provide a Go type layer.** Exported Go structs and constants allow external Go programs to parse entity files with correct field semantics and valid value sets.
5. **Provide a Go query layer.** A read-oriented query API hides file path resolution, YAML parsing, and drift detection behind a clean interface.
6. **Publish a JSON Schema artefact.** Each release ships a generated JSON Schema covering all entity types, enabling schema validation in non-Go tooling.
7. **Protect write operations.** All mutation — lifecycle transitions, referential integrity checks, ID allocation — remains internal to the Kanbanzai binary and is not exposed through the public interface.

---

## 3. Scope

### 3.1 In Scope

- Definition of the committed state boundary: which paths and file formats are part of the public schema.
- The `schema_version` field in `config.yaml`: format, placement, and semantics.
- The binary version boundary behaviour: how the binary responds when schema and binary versions differ.
- Compatibility policy: what changes are permitted under patch, minor, and major version increments.
- Unknown value handling rules for enumerated fields.
- The Go type layer: exported struct definitions, YAML tags, and valid value constants for all entity types.
- The Go query layer: read-oriented operations, method signatures, drift detection behaviour.
- Packaging requirements: how external Go projects import the types and query layer.
- JSON Schema generation: what the schema covers, how it is produced, and what guarantees it carries.
- Release publication: that the JSON Schema is shipped alongside each binary release.

### 3.2 Deferred

- Write operations through the public interface (explicitly excluded; may be reconsidered in a future major version).
- Streaming or watch-mode APIs for detecting state changes in real time.
- Cross-repository aggregation APIs.
- Non-Go SDK libraries (Python, TypeScript, etc.) — these may be provided by the community and are not in scope for 1.0.

### 3.3 Explicitly Excluded

- `.kbz/state/worktrees/` — local machine state; filesystem paths are meaningless to remote readers.
- `.kbz/index/` — derived document intelligence index; rebuilt on demand and not committed.
- `.kbz/cache/` — derived SQLite cache; rebuilt on demand and not committed.
- `.kbz/local.yaml` — per-machine settings; excluded from Git by design.
- All internal packages (`internal/`) — not importable by external projects; not part of the public interface.
- MCP server protocol and tool definitions — a separate interface, not governed by this specification.

---

## 4. Design Principles

**Committed state is the schema.** The public schema is defined as exactly the set of files and formats that Kanbanzai commits to the repository. If a file is never committed, it is not part of the public schema.

**Schema version is independent of binary version.** The `schema_version` field in `config.yaml` increments only when committed file formats change. Binary-only releases (new subcommands, improved diagnostics, performance improvements) do not change the schema version.

**The Go type layer is the source of truth.** JSON Schema is derived from Go types. Any discrepancy between the JSON Schema and the Go types is a bug in generation, not an intentional divergence.

**Read without writing.** The public interface exposes read operations only. External tools that need to mutate state must use the Kanbanzai binary or MCP server.

**Fail loudly on forward incompatibility.** When the binary encounters a schema version newer than it understands, it refuses to operate rather than silently misinterpreting data.

**Unknown values are not errors.** Enumerated fields can gain new valid values in minor versions. Consumers that encounter unrecognised values must pass them through unchanged rather than failing.

---

## 5. The Public Schema Boundary

### 5.1 Committed State (Part of the Public Schema)

The following paths and their contents are part of the public schema. External consumers may read and parse these files according to the format definitions in this specification.

| Path | Contents |
|------|----------|
| `.kbz/config.yaml` | Project configuration: schema version, prefix registry, document roots |
| `.kbz/state/plans/` | Plan entity records (one YAML file per plan) |
| `.kbz/state/features/` | Feature entity records (one YAML file per feature) |
| `.kbz/state/tasks/` | Task entity records (one YAML file per task) |
| `.kbz/state/bugs/` | Bug entity records (one YAML file per bug) |
| `.kbz/state/decisions/` | Decision entity records (one YAML file per decision) |
| `.kbz/state/documents/` | Document metadata records (one YAML file per registered document) |
| `.kbz/state/knowledge/` | Knowledge entry records (one YAML file per entry) |
| `.kbz/state/checkpoints/` | Human checkpoint records (one YAML file per checkpoint) |
| `work/` (or configured roots) | Document files (Markdown); paths are relative and recorded in document metadata records |

### 5.2 Non-Committed State (Not Part of the Public Schema)

The following paths are explicitly excluded from the public schema. External consumers must not attempt to read or rely on these files.

| Path | Reason for Exclusion |
|------|----------------------|
| `.kbz/state/worktrees/` | Local machine state — filesystem paths are meaningless to remote readers |
| `.kbz/index/` | Derived document intelligence index — rebuilt on demand, not committed |
| `.kbz/cache/` | Derived SQLite cache — rebuilt on demand, not committed |
| `.kbz/local.yaml` | Per-machine settings — excluded from Git by design |

---

## 6. Schema Versioning

### 6.1 The `schema_version` Field

`config.yaml` carries a dedicated `schema_version` field using semantic versioning (`MAJOR.MINOR.PATCH`). This field is in addition to the existing `version` field, which is preserved for backwards compatibility. External consumers must read `schema_version`; they must not treat `version` as a schema version indicator.

Example `config.yaml`:

```yaml
version: "2"
schema_version: "1.0.0"
prefixes:
  - prefix: P
    name: Plan
...
```

### 6.2 Binary Version Independence

The schema version and the binary release version are independent. The schema version increments only when committed file formats change:

- New required or optional fields on any entity type
- Removed or renamed fields
- Changed field semantics or valid value sets
- New entity types added to committed state
- Changes to lifecycle rules that affect stored status values

Binary-only changes — new subcommands, improved diagnostics, performance optimisations, additional MCP tools — do not increment the schema version.

### 6.3 Compatibility Policy

| Component | Meaning | Example |
|-----------|---------|---------|
| Patch (1.0.x) | Bug fixes and clarifications; no field additions, removals, or semantic changes | Correcting a documented constraint; fixing a generation bug |
| Minor (1.x.0) | Additive, backward-compatible changes | New optional field on an entity type; new entity type added to committed state; new valid status value |
| Major (x.0.0) | Breaking changes | Removed or renamed field; changed field semantics; changed lifecycle rules; removal of a committed path |

### 6.4 Unknown Value Handling

Enumerated fields — such as entity status, priority, severity, and document type — can gain new valid values in minor versions. External consumers that parse enumerated fields **must not fail** on unrecognised values. The correct behaviour is to display or pass through the raw string unchanged.

Consumers that enforce strict validation against a known-good enum set must treat the validation failure as a warning, not an error, and must not discard or reject the record.

### 6.5 Binary Behaviour at Version Boundaries

When the Kanbanzai binary reads `schema_version` from `config.yaml`, it applies the following rules:

| Condition | Binary Behaviour |
|-----------|-----------------|
| `schema_version` is absent | Treat as pre-1.0; prompt user to run `kanbanzai migrate` |
| `schema_version` newer than binary supports | Refuse to operate; display the versions and direct the user to upgrade the binary |
| `schema_version` older than binary supports | Offer migration via `kanbanzai migrate` before proceeding; do not operate on the old schema silently |
| `schema_version` matches binary expectations | Proceed normally |

The binary must never silently corrupt state by operating on a schema version it does not understand.

---

## 7. Go Type Layer

### 7.1 Overview

The Go type layer provides exported struct definitions for all entity types that are part of the public schema. It includes YAML field tags for correct serialisation, and exported constants for all valid values of enumerated fields.

The type layer is sufficient for consumers that want raw file access: they can use standard Go YAML libraries with these types to parse entity records without knowledge of Kanbanzai internals.

### 7.2 Coverage

The type layer covers the following entity types and their fields:

- **Plan** — ID, slug, title, summary, prefix, status, tags, timestamps, design reference
- **Feature** — ID, slug, summary, parent plan ID, status, tags, timestamps, design reference
- **Task** — ID, slug, summary, parent feature ID, status, estimate, dependencies, timestamps
- **Bug** — ID, slug, title, observed, expected, severity, priority, type, status, timestamps, reported by
- **Decision** — ID, slug, summary, rationale, decided by, timestamp
- **Document record** — ID, path, type, title, status, owner, content hash, timestamps
- **Knowledge entry** — ID, topic, content, scope, tier, tags, status, use/miss counts, TTL fields, timestamps
- **Human checkpoint** — ID, question, context, orchestration summary, created by, status, response, timestamps
- **Project configuration** — schema version, version, prefix registry entries, document roots

### 7.3 Valid Value Constants

The type layer exports named constants for all enumerated field values. Examples:

- Entity status values: `StatusProposed`, `StatusDesigning`, `StatusActive`, `StatusDone`, `StatusCancelled`, etc.
- Bug severity/priority: `SeverityLow`, `SeverityMedium`, `SeverityHigh`, `SeverityCritical`
- Document type: `DocTypeDesign`, `DocTypeSpecification`, `DocTypeDevPlan`, `DocTypeResearch`, `DocTypeReport`, `DocTypePolicy`
- Document status: `DocStatusDraft`, `DocStatusApproved`, `DocStatusSuperseded`
- Knowledge tier: `KnowledgeTier2`, `KnowledgeTier3`

These constants are the canonical source of valid values; the JSON Schema enum arrays are derived from them.

### 7.4 YAML Tags and Serialisation

All struct fields carry explicit `yaml:` tags matching the field names as they appear in committed YAML files. Consumers must use the tagged field names when parsing; they must not rely on Go field name capitalisation or inflection.

Structs follow the canonical serialisation rules defined in the YAML serialisation specification: block style, double-quoted strings only when required, deterministic field order, UTF-8, LF line endings, trailing newline.

---

## 8. Go Query Layer

### 8.1 Overview

The query layer provides a higher-level, read-oriented API on top of the type layer. It handles file path resolution, YAML parsing, cross-reference resolution, and drift detection, exposing a clean interface that does not require callers to understand the on-disk layout.

The query layer requires the repository root path to be provided at construction time, so it can resolve relative document paths recorded in document metadata records.

### 8.2 Method Signatures

The following operations are provided. Method signatures are indicative; exact names and parameter types are determined during implementation.

**Entity operations:**

```
store.GetPlan(id string) (*Plan, error)
store.ListPlans(filter PlanFilter) ([]*Plan, error)

store.GetFeature(id string) (*Feature, error)
store.ListFeatures(filter FeatureFilter) ([]*Feature, error)
store.ListFeaturesByPlan(planID string) ([]*Feature, error)

store.GetTask(id string) (*Task, error)
store.ListTasksByFeature(featureID string) ([]*Task, error)

store.GetBug(id string) (*Bug, error)
store.ListBugs(filter BugFilter) ([]*Bug, error)

store.GetDecision(id string) (*Decision, error)
store.ListDecisions() ([]*Decision, error)
```

**Document operations:**

```
store.GetDocumentRecord(id string) (*DocumentRecord, error)
store.ListDocumentRecords(filter DocumentFilter) ([]*DocumentRecord, error)
store.GetDocumentContent(id string) (string, error)
```

**Configuration:**

```
store.GetConfig() (*ProjectConfig, error)
```

### 8.3 Cross-Reference Resolution

`GetFeature` and similar methods return entities with resolved cross-references where appropriate. For example, a feature record includes its parent plan ID; the query layer resolves this to a usable reference without requiring callers to perform a second lookup manually. The exact resolution behaviour is defined per method in the API documentation.

### 8.4 Drift Detection

The query layer detects and surfaces drift for document records: a drift warning is issued when the document file on disk has changed since the record's content hash was recorded. The query layer does not fail silently; it returns drift information alongside the record so callers can decide how to handle it.

Drift is reported as a structured warning attached to the returned record, not as an error. Callers that do not care about drift may ignore the warning.

### 8.5 What Is NOT Included

The following operations are deliberately absent from the public query layer:

- **No write operations.** Creating, updating, or deleting entity records; transitioning lifecycle status; allocating IDs; enforcing referential integrity.
- **No MCP tool invocations.** The query layer is a direct file-reading API; it does not communicate with a running Kanbanzai server.
- **No document intelligence.** Index queries, section extraction, concept search, and classification are internal to the Kanbanzai binary and are not exposed.
- **No worktree operations.** Worktree records are local machine state and are not part of the public schema.

---

## 9. Packaging and Stability

### 9.1 Packaging Requirements

External Go projects must be able to import the type layer and query layer without depending on any `internal/` package from the Kanbanzai module. This requirement applies regardless of whether the types are published as a separate Go module (e.g., `github.com/owner/kanbanzai-schema`) or as exported packages within the main module.

The implementation must ensure:

- All exported types required to use the query layer are in importable (non-internal) packages.
- No exported API method requires a parameter or return type that is only available from an internal package.
- The module path and package layout are documented as part of the public interface.

### 9.2 API Stability

The Go API follows the same semantic versioning policy as the schema:

- **Patch releases:** No API changes. Bug fixes only.
- **Minor releases:** New exported types, new methods, and new optional parameters may be added. Existing method signatures are not changed.
- **Major releases:** Breaking changes to method signatures, type definitions, or package structure. Callers must update their imports and usages.

The stability guarantee applies to exported symbols in the public packages. Unexported symbols and `internal/` packages carry no stability guarantee.

### 9.3 Documentation

The public API is documented with Go doc comments on all exported types, methods, and constants. The documentation is sufficient for an external developer to use the API without reading Kanbanzai source code.

---

## 10. JSON Schema

### 10.1 Generation

The JSON Schema is generated from Go types as a build artefact. It is not maintained by hand. The Go types are the source of truth; any discrepancy between the JSON Schema and the Go types is a bug in generation, not an intentional divergence.

The generation process runs as part of the release build and produces a schema file that is published alongside each binary release.

### 10.2 Coverage

The generated JSON Schema covers:

- All entity types that are part of the public schema (Plan, Feature, Task, Bug, Decision, Document record, Knowledge entry, Human checkpoint).
- All fields on each entity type, with correct JSON/YAML types and formats.
- Required vs. optional field classification, matching the Go struct definitions.
- Valid values for all enumerated fields, represented as `enum` arrays.
- Field descriptions, derived from Go doc comments where present.
- The project configuration format (`config.yaml`).

### 10.3 Enum Exhaustiveness

Minor version releases may add new valid values to enumerated fields, which will be reflected in updated `enum` arrays in the JSON Schema. External consumers must not treat these arrays as exhaustive. A schema validation that fails because a field value is not in the current `enum` array should be treated as a warning, not a hard validation failure, consistent with the unknown value handling rule in §6.4.

### 10.4 Publication

The JSON Schema file is published as a release artefact alongside the binary for each Kanbanzai release. The schema version encoded in the file matches the `schema_version` of the release.

---

## 11. Acceptance Criteria

**AC-1: Committed boundary — included paths**
When a Kanbanzai repository is cloned, the paths `.kbz/config.yaml`, `.kbz/state/plans/`, `.kbz/state/features/`, `.kbz/state/tasks/`, `.kbz/state/bugs/`, `.kbz/state/decisions/`, `.kbz/state/documents/`, `.kbz/state/knowledge/`, and `.kbz/state/checkpoints/` are present in the repository and parseable using the public type layer.

**AC-2: Committed boundary — excluded paths**
The paths `.kbz/state/worktrees/`, `.kbz/index/`, `.kbz/cache/`, and `.kbz/local.yaml` are absent from the committed repository. No public API method accepts or returns data whose source is one of these paths.

**AC-3: `schema_version` field presence**
A Kanbanzai-initialised repository's `config.yaml` contains a `schema_version` field with a value in the form `MAJOR.MINOR.PATCH` (e.g., `"1.0.0"`). The existing `version` field is also present.

**AC-4: `schema_version` format validation**
The binary rejects a `config.yaml` where `schema_version` is present but does not conform to `MAJOR.MINOR.PATCH` semantic versioning format, reporting an actionable error message.

**AC-5: Binary behaviour — schema newer than binary**
When the binary reads a `config.yaml` with a `schema_version` whose major version is greater than the binary supports, the binary refuses to operate and outputs a message identifying both the schema version and the binary's supported version, directing the user to upgrade.

**AC-6: Binary behaviour — schema older than binary**
When the binary reads a `config.yaml` with a `schema_version` older than the binary's supported version, the binary offers migration via `kanbanzai migrate` and does not silently proceed to operate on the outdated schema.

**AC-7: Binary behaviour — schema absent**
When the binary reads a `config.yaml` that has no `schema_version` field, the binary treats the repository as pre-1.0 and prompts the user to run `kanbanzai migrate` before proceeding.

**AC-8: Unknown enumerated value handling**
The Go query layer returns a non-error result when an entity record contains an enumerated field value that is not in the set of defined constants. The unknown value is returned as a raw string. No record is discarded or omitted from list results because it contains an unknown value.

**AC-9: Go type layer completeness**
The public packages export Go struct types for Plan, Feature, Task, Bug, Decision, document record, knowledge entry, and human checkpoint, with YAML tags matching the on-disk field names. All enumerated field values have exported constants. An external Go program can import these types and parse entity YAML files without importing any `internal/` package.

**AC-10: Go query layer — entity operations**
The query layer provides methods to retrieve a single entity by ID and to list entities with filtering for at minimum: Plan, Feature (with `ListFeaturesByPlan`), Task (with `ListTasksByFeature`), and Bug. Each method returns a typed result or a descriptive error.

**AC-11: Go query layer — document operations**
The query layer provides `GetDocumentRecord`, `ListDocumentRecords`, and `GetDocumentContent`. `GetDocumentContent` reads the file at the path recorded in the document record, relative to the repository root. Drift between the file's current content and the recorded hash is surfaced as a structured warning on the returned record, not as an error.

**AC-12: No write operations in public interface**
No exported method in any public package performs a write, create, update, delete, or lifecycle transition on any entity record or file. Attempting to compile a program that calls such a method (if it existed) would fail; the operations simply do not exist in the public API.

**AC-13: Packaging — external import without internal dependencies**
An external Go module that imports the public type and query layer packages compiles successfully without any `internal/` package from the Kanbanzai module appearing in its dependency graph. This is verified by attempting to compile a minimal external program that uses the query layer.

**AC-14: JSON Schema generation**
The release build produces a JSON Schema file covering all entity types in the public schema, with correct field types, required/optional classification, and `enum` arrays for enumerated fields. The schema file is published as a release artefact alongside the binary.

**AC-15: JSON Schema source of truth**
Every field defined in the Go type layer is present in the generated JSON Schema with a matching type. Every enum constant for an enumerated field appears in the corresponding `enum` array. A CI check verifies this consistency and fails the build if the JSON Schema is out of sync with the Go types.

**AC-16: Schema version in JSON Schema**
The generated JSON Schema file encodes the `schema_version` of the release (e.g., in its `$id` or a custom metadata field), so consumers can confirm which schema version they are validating against.

**AC-17: API stability across patch releases**
A Go program that compiles against a `1.0.x` release of the public API also compiles without changes against any later `1.0.y` release (where `y > x`). No exported symbol is removed or changed in a patch release.
```

Now let me register this document with the document system: