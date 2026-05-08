| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-08T16:29:10Z           |
| Status | approved |
| Author | spec-author                    |

## Problem Statement

This specification implements Track B of the design described in
`work/P61-handoff-resilience-binding-hardening/P61-design-handoff-resilience.md`
(P61-handoff-resilience-binding-hardening/design-p61-design-handoff-resilience, approved).

Two concrete weaknesses addressed:
- **T2:** MCP handlers receive `context.Context` but do not propagate it to pipeline, git, and service layers — long-running operations cannot be cancelled server-side
- **T4:** Only the handoff handler has panic recovery; all other MCP handlers can panic and produce silent timeouts

**Scope:** Propagate `context.Context` through the service and git layers; wrap all MCP handlers with panic recovery.
**Out of scope:** Re-introducing the legacy assembly path, adding new MCP handlers.

## Requirements

### Functional Requirements

- **REQ-001:** `service.EntityService.Get` must accept `context.Context` as its first parameter.
- **REQ-002:** `git.CommitStateIfDirty` must use `exec.CommandContext` for cancellable git operations.
- **REQ-003:** `kbzctx.Pipeline.Run` must accept and propagate `context.Context` to `KnowledgeSurfacer.Surface` and `RoleResolver.Resolve`.
- **REQ-004:** Every MCP handler in `internal/mcp/*_tool.go` must be wrapped with panic recovery that converts recovered panics into structured JSON-RPC errors with `internal_panic` code and diagnostic context.
- **REQ-005:** A `wrapWithRecovery` helper function must be extracted in `internal/mcp/handler.go` and wired via the existing `wrapAllTools` hook in `actionlog/hook.go`.
- **REQ-006:** Per-tool timeout budgets must be enforced at the MCP handler boundary using `context.WithTimeout` with a 5-second default budget.

### Non-Functional Requirements

- **REQ-NF-001:** Panic recovery must complete within 1 second of the panic.
- **REQ-NF-002:** Context propagation must not introduce measurable latency regression — request handling time must not increase by more than 5% at p50.
- **REQ-NF-003:** All existing tests must pass after signature changes without behavioural modification.

## Constraints

- Must NOT change the public API of `EntityService` beyond adding `context.Context` as the first parameter.
- Must NOT remove or weaken any existing error handling.
- Must NOT add recovery at the `mcp-go` worker level (third-party library boundary).
- Out of scope: async pre-computation, cache warming for handoff prompts.

## Acceptance Criteria

- **AC-001 (REQ-001):** Given a cancelled context, when `EntityService.Get` is called, then it returns a context cancellation error within the deadline.
- **AC-002 (REQ-002):** Given a cancelled context, when `CommitStateIfDirty` is called, then the git command is terminated and returns a context error.
- **AC-003 (REQ-003):** Given a cancelled context, when `Pipeline.Run` is called, then surfacing and resolution steps honour the cancellation.
- **AC-004 (REQ-004):** Given any MCP handler that panics (e.g. deliberate `panic("test")`), when the handler is invoked, then the response is a structured JSON error with code `internal_panic`, not a timeout or silent failure.
- **AC-005 (REQ-004 all-handlers):** Given the full handler set in `internal/mcp/*_tool.go`, when inspected, then every handler is wrapped with the recovery pattern.
- **AC-006 (REQ-005):** Given `internal/mcp/handler.go`, when inspected, then a `wrapWithRecovery` helper exists and is used by `wrapAllTools`.
- **AC-007 (REQ-006):** Given a handler with a deliberate 60-second sleep, when invoked via MCP, then the request returns a structured timeout error within 5 seconds.
- **AC-008 (REQ-NF-001):** Given a handler that panics, when the panic is recovered, then the error response is returned within 1 second of the panic.

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Test | Automated: cancelled context → EntityService.Get returns error |
| AC-002 | Test | Automated: cancelled context → CommitStateIfDirty terminates git |
| AC-003 | Test | Automated: cancelled context → Pipeline.Run honours cancellation |
| AC-004 | Test | Automated: deliberate panic in any handler → structured error response |
| AC-005 | Inspection | Code review: verify every handler in *_tool.go is wrapped |
| AC-006 | Inspection | Code review: verify wrapWithRecovery helper and wiring |
| AC-007 | Test | Automated: 60s sleep in handler → timeout error within 5s |
| AC-008 | Test | Automated: panic recovery → response within 1s |
