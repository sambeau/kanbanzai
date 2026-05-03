# Design: Reopen Status Machine Output Formats

## Context

B36-F4 (Status command machine output formats, FEAT-01KQ2VHKJB5V8) was prematurely
closed on 2026-04-30 via overrides that claimed all 8 tasks were done. In reality,
only 6 of 8 tasks were completed. The two remaining `ready` tasks are:

- **TASK-01KQFCZ3AWCSX** — JSON format renderer (3 pts)
- **TASK-01KQFCZ3AZTTQ** — Plain format renderer (3 pts)

The feature lifecycle model has no `done → developing` path, so the original feature
cannot be reopened. This batch creates a replacement feature to complete the remaining
work.

## Decision

Create a new feature under B45 that duplicates the two remaining renderer tasks.
The original design from B36-F4 applies in full — this is implementation completion,
not new design.

## Scope

| Deliverable | File | Task |
|---|---|---|
| JSON renderer | `internal/cli/status/json.go` | JSON format renderer |
| Plain renderer | `internal/cli/status/plain.go` | Plain format renderer |

## Design References

All design decisions are inherited from the original B36-F4 design and specification:

- **Design**: `B36-kbz-cli-and-status/design-design-kbz-cli-and-binary-rename`
- **Spec**: `FEAT-01KQ2VHKJB5V8/spec-b36-f4-spec-status-machine-output`
- **Dev Plan**: `FEAT-01KQ2VHKJB5V8/dev-plan-b36-f4-dev-plan-status-machine-output`

The JSON renderer wraps entity/document queries in a `results` array (D-7) and uses
a distinct `{scope: project}` shape for project overview (D-8). The plain renderer
emits `key:value` pairs per FR-3 through FR-7.

## No New Design Decisions

This document exists to satisfy the stage gate requirement. All architectural decisions
were made and approved in the original B36 design cycle.
