# Specification: Status command machine output formats

| Field  | Value                    |
|--------|--------------------------|
| Date   | 2026-04-30               |
| Status | Draft                    |
| Author | spec-author              |

---

## Related Work

- **Design document:** `work/_project/design-kbz-cli-and-binary-rename.md`
  - §5.4 — `plain` and `json` format definitions
  - §6.2 — AI agents in shell-only contexts
  - §6.3 — Shell scripts and pre-commit hooks
  - §6.4 — CI and reporting
  - **Decision D-7** — JSON output for entity/document queries is always array-wrapped (`results`), even for a single target
  - **Decision D-8** — Project overview JSON contains summary counts only, not full feature lists, to keep output small and fast
- **B36-F2** — Argument resolution and routing (dependency: this feature requires B36-F2 to resolve which target to query and dispatch to the correct output path)
- **B36-F3** — Human output format (`--format human`/default). This spec covers **only** `--format plain` and `--format json`. Human-readable output is out of scope here.

---

## Overview

The `kbz status` command must support two machine-readable output formats selectable via `--format <plain|json>`. These formats are designed for programmatic consumption: shell scripts, pre-commit hooks, CI pipelines, and AI agents operating in shell-only contexts. Both formats must be stable within a major version — existing keys and fields must not be renamed or removed.

---

## Scope

**In scope:**
- `--format plain` output for all target types: feature, plan, task, bug, document, and project overview (no target)
- `--format json` output for all target types above
- Key schema and field schema definitions for each target type and scope
- Schema stability guarantee as a versioned contract
- Boundary and edge cases: null plan, missing documents, empty attention, unregistered documents

**Out of scope:**
- `--format human` (default) output — covered by B36-F3
- The argument resolution and routing logic — covered by B36-F2
- Go implementation details, JSON library choice, or serialisation mechanism
- Any format other than `plain` and `json`

---

## Functional Requirements

### FR-1: Format flag

**FR-1.1** The `kbz status` command MUST accept a `--format` flag with at least the values `plain` and `json`. The value `human` is also accepted (B36-F3); it is the default when `--format` is omitted.

**FR-1.2** When `--format plain` or `--format json` is specified, the command MUST suppress all colour, ANSI escape codes, alignment padding, decorative symbols, and progress indicators.

**FR-1.3** Machine-format output MUST be written to stdout. Errors (invalid flag, target not found) MUST be written to stderr and MUST NOT appear in stdout.

---

### FR-2: `--format plain` — general rules

**FR-2.1** Plain output MUST consist of `key: value` pairs, one per line, with no leading or trailing whitespace on either the key or the value.

**FR-2.2** Keys MUST use only lowercase ASCII letters, digits, hyphens, and dots (`.` used as namespace separator). No spaces in keys.

**FR-2.3** Values MUST be single-line strings. Multi-word values MUST NOT be quoted. Boolean values MUST be rendered as `true` or `false` (lowercase).

**FR-2.4** Missing or null values MUST be rendered as the literal string `missing`. The key MUST still be present.

**FR-2.5** The `scope` key MUST be the first key in every plain output block. Its value identifies the target type and MUST be one of: `feature`, `plan`, `task`, `bug`, `document`, `project`.

**FR-2.6** Plain output for machine consumption MUST be greppable. Each key-value pair on its own line enables `grep '^registered: false'` and similar patterns.

---

### FR-3: `--format plain` — feature target

When the resolved target is a feature, the plain output MUST contain exactly the following keys in the order listed. Keys MUST be present even when the value is `missing`.

```
scope: feature
id: <entity-id>           # e.g. FEAT-042
slug: <slug>
status: <lifecycle-status>
plan: <plan-id>           # value is "missing" when no parent plan
doc.design: <path>        # file path; "missing" when no document registered
doc.design.status: <document-status>   # "missing" when no document
doc.spec: <path>
doc.spec.status: <document-status>
doc.dev-plan: <path>
doc.dev-plan.status: <document-status>
tasks.active: <integer>
tasks.ready: <integer>
tasks.done: <integer>
tasks.total: <integer>
attention: <free-text>    # first attention item message; "none" when empty
```

**FR-3.1** When a feature has no parent plan, the value of `plan` MUST be the literal string `missing`.

**FR-3.2** When a document type is not registered for the feature, the corresponding `doc.<type>` key MUST have value `missing` and the `doc.<type>.status` key MUST also have value `missing`.

**FR-3.3** When there are multiple attention items, only the highest-severity item MUST appear in the `attention` key. For full attention output the JSON format SHOULD be used.

**FR-3.4** When the attention list is empty, the `attention` key MUST have value `none`.

---

### FR-4: `--format plain` — plan target

When the resolved target is a plan:

```
scope: plan
id: <plan-id>             # e.g. P1-my-plan
slug: <slug>
status: <lifecycle-status>
features.active: <integer>
features.done: <integer>
features.total: <integer>
attention: <free-text or "none">
```

---

### FR-5: `--format plain` — task and bug targets

When the resolved target is a task:

```
scope: task
id: <task-id>
slug: <slug>
status: <lifecycle-status>
parent_feature: <feature-id>
attention: <free-text or "none">
```

When the resolved target is a bug:

```
scope: bug
id: <bug-id>
slug: <slug>
status: <lifecycle-status>
severity: <severity>
parent_feature: <feature-id or "missing">
attention: <free-text or "none">
```

---

### FR-6: `--format plain` — document target

When the resolved target is a document:

```
scope: document
id: <doc-id>
path: <file-path>
type: <document-type>
status: <document-status>
registered: <true|false>
owner: <entity-id or "missing">
attention: <free-text or "none">
```

**FR-6.1** The `registered` key MUST be present in every document plain output.

**FR-6.2** When a document is not registered in the Kanbanzai document store, `registered` MUST be `false`. Pre-commit hooks MUST be able to detect unregistered documents by grepping for `^registered: false`.

**FR-6.3** When a document is registered, `registered` MUST be `true`.

**FR-6.4** When a document has no owner (e.g. it was registered standalone), the `owner` key MUST be present with value `missing`.

---

### FR-7: `--format plain` — project overview (no target)

When `kbz status` is invoked with no target argument:

```
scope: project
plans.total: <integer>
features.active: <integer>
features.done: <integer>
features.total: <integer>
health.errors: <integer>
health.warnings: <integer>
attention: <free-text or "none">
```

**FR-7.1** `health.errors` MUST reflect the count of error-severity attention items across all entities in the project. This value MUST be accurate enough to drive a binary CI pass/fail gate (i.e. `grep '^health.errors: 0'`).

---

### FR-8: `--format json` — general rules

**FR-8.1** JSON output MUST be valid RFC 8259 JSON.

**FR-8.2** JSON output MUST be emitted as a single JSON object on stdout. Streaming (newline-delimited) JSON is not required.

**FR-8.3** Object keys MUST use `snake_case`. Values that are natural strings (IDs, slugs, paths, statuses) MUST be JSON strings. Counts MUST be JSON numbers (integers). Missing or null references MUST be JSON `null`.

**FR-8.4** Boolean values MUST be JSON `true` or `false`.

**FR-8.5** Empty arrays MUST be represented as `[]`, not `null`.

---

### FR-9: `--format json` — entity/document queries (feature, plan, task, bug, document)

All entity and document target queries MUST produce a top-level `results` array (Decision D-7). A single-target query produces an array of one element.

#### FR-9.1 Feature result object

```json
{
  "results": [
    {
      "scope": "feature",
      "feature": {
        "id": "FEAT-042",
        "display_id": "F-042",
        "slug": "my-feature",
        "status": "developing",
        "plan_id": "P1-my-plan"
      },
      "documents": {
        "design":   { "id": "DOC-0019", "path": "work/design/my-feature.md", "status": "approved" },
        "spec":     { "id": "DOC-0023", "path": "work/spec/my-feature-spec.md", "status": "approved" },
        "dev-plan": null
      },
      "tasks": { "active": 1, "ready": 3, "done": 7, "total": 11 },
      "attention": [
        { "severity": "warning", "message": "No dev-plan document registered — agents cannot begin planning" }
      ]
    }
  ]
}
```

- `feature.plan_id` MUST be `null` (JSON null) when the feature has no parent plan.
- `documents` MUST contain keys for all registered document types the system tracks. A document type not registered for this feature MUST appear as the key mapped to `null` (not absent).
- `tasks` counts MUST be integers; zero is valid.
- `attention` MUST be an array. When there are no attention items the value MUST be `[]`.

#### FR-9.2 Plan result object

```json
{
  "results": [
    {
      "scope": "plan",
      "plan": {
        "id": "P1-my-plan",
        "slug": "my-plan",
        "status": "active"
      },
      "features": { "active": 2, "done": 3, "total": 5 },
      "attention": []
    }
  ]
}
```

#### FR-9.3 Task result object

```json
{
  "results": [
    {
      "scope": "task",
      "task": {
        "id": "TASK-0099",
        "slug": "implement-output-flag",
        "status": "active",
        "parent_feature_id": "FEAT-042"
      },
      "attention": []
    }
  ]
}
```

#### FR-9.4 Bug result object

```json
{
  "results": [
    {
      "scope": "bug",
      "bug": {
        "id": "BUG-0017",
        "slug": "crash-on-empty-project",
        "status": "active",
        "severity": "high",
        "parent_feature_id": null
      },
      "attention": []
    }
  ]
}
```

- `parent_feature_id` MUST be `null` when the bug has no parent feature.

#### FR-9.5 Document result object

```json
{
  "results": [
    {
      "scope": "document",
      "document": {
        "id": "DOC-0023",
        "path": "work/spec/my-feature-spec.md",
        "type": "specification",
        "status": "approved",
        "registered": true,
        "owner_id": "FEAT-042"
      },
      "attention": []
    }
  ]
}
```

- `registered` MUST be a JSON boolean.
- When a document is not registered, `registered` MUST be `false`. The `id` field MUST be `null` when unregistered (no ID has been assigned).
- `owner_id` MUST be `null` when the document has no owner.

#### FR-9.6 Unregistered document result object

When `kbz status <path>` is invoked for a file that is not registered:

```json
{
  "results": [
    {
      "scope": "document",
      "document": {
        "id": null,
        "path": "work/spec/unregistered-spec.md",
        "type": null,
        "status": null,
        "registered": false,
        "owner_id": null
      },
      "attention": [
        { "severity": "warning", "message": "Document is not registered in the Kanbanzai document store" }
      ]
    }
  ]
}
```

---

### FR-10: `--format json` — project overview (no target)

When invoked with no target, the output MUST use a distinct top-level shape (Decision D-8). It MUST NOT be wrapped in a `results` array.

```json
{
  "scope": "project",
  "plans": [
    {
      "id": "P1-main-plan",
      "slug": "main-plan",
      "status": "active",
      "features": { "active": 2, "done": 3, "total": 5 }
    }
  ],
  "health": { "errors": 0, "warnings": 2 },
  "attention": [
    { "severity": "warning", "entity_id": "FEAT-042", "message": "No dev-plan document registered" }
  ]
}
```

**FR-10.1** Each object in `plans` MUST include feature counts (`active`, `done`, `total`) but MUST NOT include the full feature list (Decision D-8).

**FR-10.2** `health.errors` MUST be the total count of error-severity attention items across the project. `health.warnings` MUST be the total count of warning-severity items.

**FR-10.3** `attention` at the project level MUST include `entity_id` on each item. `entity_id` MUST be `null` for project-wide items not attributable to a single entity.

**FR-10.4** When there are no plans, `plans` MUST be `[]`.

**FR-10.5** When there are no attention items, `attention` MUST be `[]`.

---

### FR-11: Exit codes

**FR-11.1** `kbz status --format plain` and `kbz status --format json` MUST exit with code `0` on success, regardless of the content of health or attention output.

**FR-11.2** A non-zero exit code MUST only be used for invocation errors (unknown target, invalid flag value, I/O error). It MUST NOT be used to signal health status (callers can inspect `health.errors` or `registered: false` themselves).

---

## Non-Functional Requirements

### NFR-1: Schema stability

**NFR-1.1** Within a given major version of `kbz`, the plain-format key names defined in FR-3 through FR-7 MUST NOT be renamed or removed.

**NFR-1.2** Within a given major version, the JSON field names defined in FR-9 and FR-10 MUST NOT be renamed or removed.

**NFR-1.3** New keys (plain) and new fields (JSON) MAY be added in minor or patch releases without breaking compatibility.

**NFR-1.4** If a breaking schema change is required, it MUST be gated behind a major version increment and MUST be documented in CHANGELOG as a breaking change.

**NFR-1.5 (Testable):** An automated contract test MUST enumerate all required plain keys (FR-3–FR-7) and required JSON fields (FR-9–FR-10) and assert their presence in the output of each command variant. This test MUST run in CI.

### NFR-2: Performance

**NFR-2.1** `kbz status --format json` with no target MUST complete in under 2 seconds for a project with up to 200 features.

### NFR-3: Parsability

**NFR-3.1** JSON output MUST be parseable by `jq` without flags, by Python's `json.loads`, and by any RFC 8259-compliant parser.

**NFR-3.2** Plain output MUST be parseable by POSIX `grep`, `awk`, and `cut` without preprocessing.

---

## Acceptance Criteria

### AC-1: Plain format — feature with full documents

Given a feature `FEAT-042` with:
- slug `my-feature`, status `developing`, plan `P1-my-plan`
- design doc registered (approved), spec doc registered (approved), no dev-plan registered
- 1 active task, 3 ready tasks, 7 done tasks (total 11)
- 1 warning attention item: "No dev-plan document registered"

Running `kbz status FEAT-042 --format plain` MUST produce output that:
- Starts with `scope: feature`
- Contains `id: FEAT-042`
- Contains `plan: P1-my-plan`
- Contains `doc.design: work/design/my-feature.md`
- Contains `doc.design.status: approved`
- Contains `doc.dev-plan: missing`
- Contains `doc.dev-plan.status: missing`
- Contains `tasks.total: 11`
- Contains `attention:` with a non-empty message

### AC-2: Plain format — feature with no plan

Running `kbz status FEAT-099 --format plain` for a feature with no parent plan MUST contain `plan: missing`.

### AC-3: Plain format — unregistered document

Running `kbz status work/spec/unregistered.md --format plain` MUST produce output that contains `registered: false`.

Running `kbz status work/spec/unregistered.md --format plain | grep '^registered: false'` MUST produce a match.

### AC-4: Plain format — project overview health gate

Running `kbz status --format plain | grep '^health.errors: 0'` on a healthy project MUST produce a match. On a project with at least one error-severity item it MUST NOT match.

### AC-5: JSON format — feature results array

Running `kbz status FEAT-042 --format json` MUST produce JSON where:
- Top-level key is `results` (array)
- `results[0].scope == "feature"`
- `results[0].feature.id == "FEAT-042"`
- `results[0].documents["dev-plan"] == null`
- `results[0].attention` is a non-empty array
- `results[0].tasks.total == 11`

### AC-6: JSON format — feature with null plan_id

For a feature with no parent plan: `results[0].feature.plan_id` MUST be JSON `null`.

### AC-7: JSON format — unregistered document

Running `kbz status work/spec/unregistered.md --format json` MUST produce JSON where:
- `results[0].scope == "document"`
- `results[0].document.registered == false`
- `results[0].document.id == null`
- `results[0].attention` is a non-empty array

### AC-8: JSON format — project overview shape

Running `kbz status --format json` with no target MUST produce JSON where:
- Top-level key is `scope` with value `"project"` (NOT a `results` array)
- `plans` is an array
- Each plan object contains `features.total` (integer) but NOT a `features` array of full feature objects
- `health` contains integer keys `errors` and `warnings`
- `attention` is an array (possibly empty)

### AC-9: JSON format — empty attention

For a fully healthy feature: `results[0].attention` MUST be `[]` (empty array), not `null` and not absent.

### AC-10: Exit codes

All successful invocations of `kbz status --format plain` and `kbz status --format json` MUST exit with code `0`, including when `health.errors > 0` or `registered: false`.

### AC-11: Schema contract test

A contract test MUST exist that asserts the presence of every key defined in FR-3–FR-7 (plain) and every field defined in FR-9–FR-10 (JSON) across all scope types, and MUST run in CI.

---

## Verification Plan

| ID | Method | Description |
|----|--------|-------------|
| V-1 | Integration test | AC-1 through AC-11 above, run against a seeded test project |
| V-2 | Contract test (NFR-1.5) | Enumerate required keys/fields; assert presence for each scope type |
| V-3 | Regression test | After any schema change, re-run contract test; enforce fail-on-removal in CI |
| V-4 | Manual | Run `kbz status FEAT-042 --format json | jq .` and confirm no parse errors |
| V-5 | Manual | Run `kbz status --format plain | grep '^health.errors:'` in a CI-like environment |
| V-6 | Performance test (NFR-2.1) | Seed 200 features; measure `kbz status --format json` wall time |

---

## Dependencies and Assumptions

### Dependencies

- **B36-F2 (Argument resolution):** This feature depends on B36-F2 to resolve the target argument and route to the correct status data path. The `--format` flag handling specified here assumes the target has already been resolved.
- **Document store:** The `registered` boolean and document ID/status fields depend on the document store being queryable from the `status` command path.
- **Entity state files:** Feature, plan, task, and bug field values are assumed to be available via the MCP server tools, not by direct `.kbz/state/` file reads.

### Assumptions

- **A-1:** The set of document types tracked per feature is fixed at `design`, `spec`, and `dev-plan`. If additional types are added in future, the plain and JSON schemas will be extended additively (NFR-1.3).
- **A-2:** The `display_id` field in the feature JSON object (e.g. `F-042`) is generated from the entity ID by stripping the `FEAT-` prefix and reformatting. This generation rule is implementation detail; the field must be present and non-null.
- **A-3:** Attention item severity levels are `error` and `warning`. No other severity values are defined for the v1 schema.
- **A-4:** The `health.errors` and `health.warnings` counts in the project overview are computed at query time and reflect the current state. Stale counts are not acceptable for CI gate use.
- **A-5:** `kbz status <path>` for a file that does not exist on disk is an invocation error (exit non-zero, message to stderr); it is distinct from a file that exists but is not registered (exit zero, `registered: false` in output).
