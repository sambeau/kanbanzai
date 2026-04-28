# Public Schema Interface: Feature Dev-Plan

| Document | Public Schema Interface Dev-Plan        |
|----------|-----------------------------------------|
| Feature  | FEAT-01KMKRQV025FA                      |
| Spec     | `work/spec/public-schema-interface.md`  |
| Status   | Draft                                   |
| Created  | 2026-03-26                              |

---

## Overview

This feature makes the `.kbz` schema a stable public interface: exported Go types for
external consumers, a read-only query layer, `schema_version` versioning in `config.yaml`,
and a generated JSON Schema for non-Go consumers.

The implementation is self-contained and has no hard dependencies on other Wave 1 features,
though it should be completed before binary-distribution (which needs `--version` and the
JSON Schema as a release artefact).

---

## Tasks

### T1 — Export Go type layer (`public/` package)

Create a new top-level package (e.g. `public/` or `kbzschema/`) that exports Go struct types
for every committed entity: Plan, Feature, Task, Bug, Decision, KnowledgeEntry,
DocumentRecord, and HumanCheckpoint. Each struct must carry YAML tags matching on-disk field
names. All enumerated field values must have exported constants. The package must not import
any `internal/` package from this module.

Acceptance: AC-9, AC-13.

### T2 — Add `schema_version` to config

Add the `schema_version` field (MAJOR.MINOR.PATCH format) to `config.yaml` output and
parsing. Implement:
- Format validation (reject malformed values, AC-4).
- Binary refusal when major version is newer than supported (AC-5).
- Migration prompt when schema is older than binary (AC-6).
- Pre-1.0 detection (absent field) with prompt to run `kanbanzai migrate` (AC-7).

The existing `version` field is preserved alongside `schema_version`.

Acceptance: AC-3, AC-4, AC-5, AC-6, AC-7.

### T3 — Implement Go query layer

Implement a `Reader` type (or equivalent) that opens a Kanbanzai repository root and
provides read-only methods:
- `GetPlan(id) (Plan, error)`
- `ListPlans() ([]Plan, error)`
- `GetFeature(id) (Feature, error)`
- `ListFeaturesByPlan(planID) ([]Feature, error)`
- `GetTask(id) (Task, error)`
- `ListTasksByFeature(featureID) ([]Task, error)`
- `GetBug(id) (Bug, error)`
- `ListBugs() ([]Bug, error)`
- `GetDocumentRecord(id) (DocumentRecord, error)`
- `ListDocumentRecords() ([]DocumentRecord, error)`
- `GetDocumentContent(id) (string, DriftWarning, error)`

Unknown enumerated values must be returned as raw strings, never cause a record to be
dropped (AC-8). Drift between file content and stored hash is a structured warning, not an
error (AC-11). No write methods are exposed (AC-12).

Acceptance: AC-8, AC-10, AC-11, AC-12.

### T4 — JSON Schema generation

Implement a code-generation step (e.g. `cmd/schemagen/` or a `go generate` directive) that
derives a JSON Schema from the Go type layer. The schema must:
- Cover all entity types with correct field types, required/optional classification, and
  `enum` arrays for all enumerated fields.
- Encode the `schema_version` in its `$id` or a custom metadata field (AC-16).
- Be output to a well-known path (e.g. `schema/kanbanzai.schema.json`).

Add a CI check that regenerates the schema and fails if the committed file differs from the
generated output (AC-15). Wire the schema file into the GoReleaser configuration as a
release artefact (coordinate with binary-distribution feature).

Acceptance: AC-14, AC-15, AC-16.

### T5 — Tests and external-compilation verification

Write tests covering:
- Round-trip: write entity YAML via internal types, parse via public types — fields match.
- Unknown enum value: entity with unknown status field is returned without error.
- Drift detection: modify file after recording hash, `GetDocumentContent` returns warning.
- External compilation: a minimal `_testexternal/` Go module that imports only the public
  package and the query layer compiles successfully (`go build ./...` must pass with no
  `internal/` dependency), verifying AC-13.
- Schema consistency: generated JSON Schema matches Go types (CI check described in T4 can
  serve as the test).
- API stability: reserved for manual review at patch release time.

Acceptance: AC-1, AC-2, AC-9, AC-13, AC-17 (manual).

---

## Implementation Notes

- The public package must not re-export or wrap `internal/model` types — it must define its
  own structs. This is what makes external import without `internal/` possible.
- YAML tags on the public types must exactly match the on-disk field names produced by the
  internal serialiser. Any mismatch breaks round-trip parsing.
- The `schema_version` value to embed at `init` time is `"1.0.0"` for the 1.0 release.
- The query layer is intentionally read-only. If a future phase needs mutation, it will be
  added as a separate API surface.
- T1 and T2 can be developed in parallel. T3 depends on T1. T4 depends on T1. T5 depends on
  T1–T4.

---

## Acceptance Criteria Coverage

| AC   | Task |
|------|------|
| AC-1 | T5   |
| AC-2 | T5   |
| AC-3 | T2   |
| AC-4 | T2   |
| AC-5 | T2   |
| AC-6 | T2   |
| AC-7 | T2   |
| AC-8 | T3   |
| AC-9 | T1, T5 |
| AC-10 | T3  |
| AC-11 | T3  |
| AC-12 | T3  |
| AC-13 | T5  |
| AC-14 | T4  |
| AC-15 | T4  |
| AC-16 | T4  |
| AC-17 | T5 (manual) |