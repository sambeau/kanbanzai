| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-08T16:29:10Z           |
| Status | approved |
| Author | spec-author                    |

## Problem Statement

This specification implements Track A of the design described in
`work/P61-handoff-resilience-binding-hardening/P61-design-handoff-resilience.md`
(P61-handoff-resilience-binding-hardening/design-p61-design-handoff-resilience, approved).

The handoff panic incident (BUG-01KR45KJWB2KY) was caused by a silent stage-bindings load failure cascading into a nil-pointer panic. Two concrete weaknesses are addressed here:
- **T1:** `stage-bindings.yaml` has no schema versioning — adding fields silently breaks decoding
- **T5:** `health()` does not surface binding-load failures — operators cannot detect configuration errors

**Scope:** Schema-versioned stage-bindings with forward-compatible upgrade path and health-check integration.
**Out of scope:** Re-platforming the storage engine, adding new binding fields beyond schema_version.

## Requirements

### Functional Requirements

- **REQ-001:** `stage-bindings.yaml` must include a `schema_version` field (integer) at the top level. The current file is version 2.
- **REQ-002:** The binding loader must inspect `schema_version` first and dispatch to the appropriate decoder. Unsupported versions must produce a clear, structured error.
- **REQ-003:** A `kbz migrate stage-bindings` command must add the `schema_version: 2` key idempotently without altering other content.
- **REQ-004:** `health()` must include a `binding_loadable` check that calls `LoadBindingFile` and reports errors as warnings.
- **REQ-005:** Older binaries (without schema_version support) that encounter a v2 file must refuse with a clear message, not silently mis-decode.

### Non-Functional Requirements

- **REQ-NF-001:** The version dispatch must add less than 1 ms to binding load time.
- **REQ-NF-002:** The `kbz migrate` command must be idempotent — running it twice produces the same result.

## Constraints

- Must NOT change the structure or semantics of existing stage-binding fields.
- Must NOT require operators to manually edit `stage-bindings.yaml` after migration.
- Must preserve `KnownFields(true)` strict decoding for typo detection.
- Out of scope: moving stage-bindings to SQLite or another store.

## Acceptance Criteria

- **AC-001 (REQ-001):** Given `stage-bindings.yaml`, when inspected, then `schema_version: 2` is present at the top level.
- **AC-002 (REQ-002a):** Given a valid v2 file, when `LoadBindingFile` is called, then it successfully decodes using the v2 decoder.
- **AC-003 (REQ-002b):** Given a file with `schema_version: 99`, when `LoadBindingFile` is called, then it returns a structured error containing "unsupported schema_version" and the binary's supported versions.
- **AC-004 (REQ-002c):** Given a v2 file with an unknown field, when `LoadBindingFile` is called, then strict decoding rejects it with a clear field-name error.
- **AC-005 (REQ-003):** Given a v1 file without `schema_version`, when `kbz migrate stage-bindings` runs, then the file gains `schema_version: 2` and all other content is preserved.
- **AC-006 (REQ-003 idempotent):** Given a file with `schema_version: 2`, when `kbz migrate stage-bindings` runs again, then the file is unchanged.
- **AC-007 (REQ-004):** Given a malformed `stage-bindings.yaml`, when `health()` is called, then the response includes a `binding_loadable` warning with the error detail.
- **AC-008 (REQ-004 healthy):** Given a valid `stage-bindings.yaml`, when `health()` is called, then the `binding_loadable` check reports `ok`.
- **AC-009 (REQ-005):** Given an older binary that does not understand v2, when it loads a v2 `stage-bindings.yaml`, then it produces a clear version-mismatch error instead of a cryptic YAML decode failure.

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Test | Automated: verify schema_version present in test fixture |
| AC-002 | Test | Automated: LoadBindingFile with valid v2 fixture |
| AC-003 | Test | Automated: LoadBindingFile with unsupported version fixture |
| AC-004 | Test | Automated: strict decode rejects unknown field in v2 fixture |
| AC-005 | Test | Automated: migrate command on v1 fixture produces v2 output |
| AC-006 | Test | Automated: migrate command is idempotent |
| AC-007 | Test | Automated: health() with malformed bindings produces warning |
| AC-008 | Test | Automated: health() with valid bindings reports ok |
| AC-009 | Test | Automated: old binary with v2 file produces clear error |
