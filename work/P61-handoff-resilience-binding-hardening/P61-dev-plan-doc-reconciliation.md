| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-08T16:29:10Z           |
| Status | approved |
| Author | architect                      |

## Overview

Track D of P61: update handoff documentation across four files to match the current pipeline-3.0 implementation and flag missing capabilities for separate planning.

## Scope

This dev-plan implements the specification `FEAT-01KR46PKHMG4J/spec-p61-spec-doc-reconciliation` (approved) covering Track D of P61: documentation reconciliation for handoff capabilities.

## Task Breakdown

| Task | Description | Deliverable | Effort |
|------|-------------|-------------|--------|
| T1 | Update `handoff_tool.go` header comment | Accurate pipeline-3.0 header | 0.5h |
| T2 | Update `assembly.go` header comment | next-only header | 0.5h |
| T3 | Update `AGENTS.md` handoff section | Corrected capability claims | 1h |
| T4 | Update `.github/copilot-instructions.md` handoff section | Corrected capability claims | 1h |
| T5 | Create gap-tracking decision record | Decision or issue for missing capabilities | 1h |

## Dependency Graph

T1, T2, T3, T4 — all independent, parallelisable. T5 depends on T3+T4 (needs to know what was corrected before flagging the gap).

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-----------|--------|------------|
| Missed handoff references in other docs | Low | Low | grep for "handoff" across all docs before closing |
| Gap decision creates scope creep | Low | Low | Decision record explicitly states out-of-scope |

## Interface Contracts

No code interfaces change. Documentation consistency contract: no document may claim handoff behaviour contradicted by another document.

## Traceability Matrix

| Task | REQ | AC |
|------|-----|----|
| T1 | REQ-001 | AC-001 |
| T2 | REQ-002 | AC-002 |
| T3 | REQ-003 | AC-003 |
| T4 | REQ-004 | AC-004 |
| T5 | REQ-005 | AC-005 |

## Verification Approach

All acceptance criteria verified by inspection (grep/code review). `go build ./...` must pass after header comment changes.
