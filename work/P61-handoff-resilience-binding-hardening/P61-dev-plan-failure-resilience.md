| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-08T16:29:10Z           |
| Status | approved |
| Author | architect                      |

## Overview

Track B of P61: propagate context.Context through EntityService, git helpers, and Pipeline; wrap all MCP handlers with panic recovery producing structured JSON-RPC errors.

## Scope

This dev-plan implements the specification `FEAT-01KR46PKHPVWS/spec-p61-spec-failure-resilience` (approved) covering Track B of P61: context.Context propagation and panic recovery wrapping for all MCP handlers.

## Task Breakdown

| Task | Description | Deliverable | Effort |
|------|-------------|-------------|--------|
| T1 | Add `context.Context` to `EntityService.Get` signature and propagate to ~30 call sites | Updated service layer | 3h |
| T2 | Switch `git.CommitStateIfDirty` to `exec.CommandContext` | Cancellable git operations | 1h |
| T3 | Thread `context.Context` through `Pipeline.Run` to `Surfacer` and `Resolver` | Context-aware pipeline | 2h |
| T4 | Extract `wrapWithRecovery` helper in `handler.go` | Recovery helper | 1h |
| T5 | Wire recovery wrapping via `wrapAllTools` in `actionlog/hook.go` | All handlers wrapped | 1h |
| T6 | Add per-tool 5s timeout budgets at handler boundary | Timeout enforcement | 1h |
| T7 | Write tests: context cancellation, panic recovery, timeout enforcement | Test suite covering all ACs | 3h |

## Dependency Graph

T1, T2, T3 — independent. T4 → T5 (wire needs helper). T6 depends on T5 (timeout wraps recovery). T7 depends on T1+T2+T3+T5+T6.

Critical path: T4 → T5 → T6 → T7 (~6h). T1+T2+T3 can run in parallel with T4+T5.

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-----------|--------|------------|
| T1 signature change across ~30 call sites causes breakage | High | High | Single atomic commit; full test suite after each change |
| T4 recovery wrapping changes error contract for existing clients | Medium | Medium | Recovery only activates on panic; normal path unchanged |
| T6 5s budget too aggressive for some tools | Low | Medium | Make budget configurable; monitor in health() |

## Interface Contracts

- `EntityService.Get(ctx, id) (*Entity, error)` — ctx added as first param; ~30 call sites updated
- `git.CommitStateIfDirty(ctx, repoRoot)` — ctx added; uses exec.CommandContext
- `Pipeline.Run(ctx, input)` — ctx threaded to Surfacer and Resolver
- `wrapWithRecovery(toolName, handler) ToolHandler` — new helper in handler.go

## Traceability Matrix

| Task | REQ | AC |
|------|-----|----|
| T1 | REQ-001 | AC-001 |
| T2 | REQ-002 | AC-002 |
| T3 | REQ-003 | AC-003 |
| T4 | REQ-005 | AC-006 |
| T5 | REQ-004 | AC-004, AC-005 |
| T6 | REQ-006 | AC-007 |
| T7 | REQ-NF-001, REQ-NF-002, REQ-NF-003 | AC-008 |

## Verification Approach

All ACs verified by automated tests (T7). Manual inspection for AC-005 (all handlers wrapped) and AC-006 (helper wiring).
