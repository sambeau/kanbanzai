| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-08T16:29:10Z           |
| Status | Draft                          |
| Author | architect                      |

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

## Verification Approach

All 9 ACs verified by automated tests (T5). Manual inspection for AC-001 (schema_version presence in live file).
