| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-08T16:29:10Z           |
| Status | approved |
| Author | architect                      |

## Overview

Track A of P61: introduce schema_version: 2 to stage-bindings.yaml with version-dispatch decoder, kbz migrate command, and health() binding_loadable check.

## Scope

This dev-plan implements the specification `FEAT-01KR46PKHPVSH/spec-p61-spec-config-safety` (approved) covering Track A of P61: schema-versioned stage-bindings and health-check integration.

## Task Breakdown

| Task | Description | Deliverable | Effort |
|------|-------------|-------------|--------|
| T1 | Add `schema_version` field and v2 decoder to binding loader | `internal/binding/loader.go` with version dispatch | 3h |
| T2 | Implement `kbz migrate stage-bindings` command | Idempotent migration CLI command | 2h |
| T3 | Add `binding_loadable` check to `health()` | Health check integration | 2h |
| T4 | Add `schema_version: 2` to live `stage-bindings.yaml` | Updated config file | 0.5h |
| T5 | Write tests for version dispatch, migration, health integration | Test suite covering all ACs | 3h |

## Dependency Graph

T1 → T2 (migrate needs loader), T1 → T3 (health needs loader). T4 depends on T2. T5 depends on T1+T2+T3.

Critical path: T1 → T2 → T4 → T5 (~8.5h)

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-----------|--------|------------|
| T1 signature changes break callers | Medium | Medium | Run full test suite after T1; isolate in feature branch |
| v1→v2 migration corrupts existing config | Low | High | T2 must have dry-run mode; T4 verifies round-trip |
| health() overhead unacceptable | Low | Low | LoadBindingFile is cheap; cache if needed |

## Interface Contracts

- `LoadBindingFile(path) (*BindingFile, []error)` — signature unchanged, internal dispatch added
- `health()` — adds `binding_loadable` check; existing checks unchanged
- `kbz migrate stage-bindings` — new CLI subcommand, idempotent

## Traceability Matrix

| Task | REQ | AC |
|------|-----|----|
| T1 | REQ-001, REQ-002 | AC-001, AC-002, AC-003, AC-004 |
| T2 | REQ-003 | AC-005, AC-006 |
| T3 | REQ-004 | AC-007, AC-008 |
| T4 | REQ-001 | AC-001 |
| T5 | REQ-005, REQ-NF-001, REQ-NF-002 | AC-009 |

## Verification Approach

All 9 ACs verified by automated tests (T5). Manual inspection for AC-001 (schema_version presence in live file).
