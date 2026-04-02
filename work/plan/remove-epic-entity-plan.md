# Implementation Plan: Remove Epic Entity

- Feature: FEAT-01KN85HMQP2CX
- Spec: `work/spec/remove-epic-entity.md`
- Design: `work/design/remove-epic-entity.md`

---

## Overview

Pure deletion across ~15 files. No new behaviour, no data migration.
Tasks are split by package boundary so two agents can run in parallel.

---

## Task Breakdown

| # | Task | Files | Spec Refs |
|---|------|-------|-----------|
| T1 | Remove Epic from internal packages | `internal/model/entities.go`, `internal/id/allocator.go`, `internal/id/display.go`, `internal/validate/entity.go`, `internal/validate/lifecycle.go`, `internal/validate/health.go`, `internal/cache/cache.go`, `internal/storage/entity_store.go`, `internal/docint/extractor.go`, `internal/service/entities.go`, `internal/service/documents.go`, and all `_test.go` counterparts | FR-1–FR-9, FR-11, NFR-1–2 |
| T2 | Remove Epic from CLI, MCP, and testdata | `cmd/kanbanzai/entity_cmd.go`, `cmd/kanbanzai/main.go`, `cmd/kanbanzai/main_test.go`, `internal/mcp/entity_tool.go`, `internal/mcp/estimate_tool.go`, `testdata/entities/epic.yaml` (delete), `testdata/entities/feature.yaml` (update parent to a Plan ID) | FR-3–FR-5, FR-7–FR-8, FR-10–FR-11, NFR-1–2 |

T1 and T2 touch disjoint files and may run in parallel. If run sequentially, do T1 first — it removes the model types that T2's deleted code would otherwise reference.

---

## Dependency Graph

```
[T1: internal packages] ──┐
                           ├──► go build ./... && go test ./...
[T2: CLI + MCP + testdata] ┘
```

---

## Interface Contracts

None. This plan only removes interfaces; it introduces no new ones.

---

## Traceability Matrix

| Spec Ref | Task |
|----------|------|
| FR-1 (Epic struct, EpicStatus) | T1 |
| FR-2 (EntityKindEpic) | T1 |
| FR-3 (CreateEpic, entityService interface) | T1 (service), T2 (CLI interface) |
| FR-4 (CLI "epic"/"epics" cases) | T2 |
| FR-5 (EPIC- prefix in id package) | T1 |
| FR-6 (validate: lifecycle, entry/terminal, required fields) | T1 |
| FR-7 (storage, cache, service helpers, MCP descriptions) | T1 (internal), T2 (MCP) |
| FR-8 (docint EPIC- pattern) | T1 |
| FR-9 (Feature.Epic field, extractParentRefFromState fallback) | T1 |
| FR-10 (testdata) | T2 |
| FR-11 (tests) | T1 + T2 (each task cleans its own tests) |
| NFR-1 (go test ./... passes) | verified after both tasks complete |
| NFR-2 (no nolint directives) | verified after both tasks complete |