# Review Report: B43-F1 — Composite Tools for Workflow Chaining

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-04                     |
| Status | Draft                          |
| Author | sambeau                        |

## Summary

All 7 tasks for FEAT-01KQJ7CJGQR7Y (Composite Tools Architecture Design) are complete. Five composite MCP tool actions have been implemented:

| Action | Tool | File | Status |
|--------|------|------|--------|
| `doc(action: "publish")` | doc | `internal/mcp/doc_tool.go` | ✅ |
| `entity(action: "bootstrap")` | entity | `internal/mcp/entity_tool.go` | ✅ |
| `entity(action: "close-out")` | entity | `internal/mcp/entity_tool.go` | ✅ |
| `develop(action: "dispatch")` | develop (new) | `internal/mcp/develop_tool.go` | ✅ |
| `batch(action: "snapshot")` | batch (new) | `internal/mcp/batch_tool.go` | ✅ |

Shared helper: `internal/mcp/next_action.go` provides structured `next_action` response construction.

## Conformance Review

### REQ-001 through REQ-004 (doc publish)
- ✅ Publish action added to doc tool DispatchAction map
- ✅ Auto-approves when classifications with concepts_intro provided
- ✅ Returns structured next_action when classifications omitted
- ✅ Reports partial failure with registration: ok, approval: failed on error

### REQ-005 through REQ-008 (entity bootstrap)
- ✅ Bootstrap action added to entity tool DispatchAction map
- ✅ Wraps existing AdvanceFeatureStatus
- ✅ Enriches stopped responses with structured next_action objects
- ✅ Maps stopped_reason to appropriate missing document type
- ✅ Human gates respected — advance does not bypass

### REQ-009 through REQ-012 (entity close-out)
- ✅ Close-out action verifies feature in reviewing
- ✅ Checks all tasks terminal via CountNonTerminalTasks
- ✅ Returns structured next_action on non-terminal tasks
- ✅ Advances to done and triggers MaybeAutoAdvancePlan cascade
- ✅ Enumerates all affected entities

### REQ-013 through REQ-016 (develop dispatch)
- ✅ New develop tool created following consolidated-tool pattern
- ✅ Dispatch action identifies ready task frontier
- ✅ Transitions ready tasks to active
- ✅ Returns dispatched, blocked, conflicting, empty_queue fields
- ✅ Does NOT call spawn_agent

### REQ-017, REQ-018 (batch snapshot)
- ✅ Batch tool created with snapshot action
- ✅ Enumerates features by batch parent
- ✅ Determines blocking gates per lifecycle stage
- ✅ Returns structured next_action per blocked feature

### REQ-019 through REQ-022 (architectural)
- ✅ All composites reuse existing service/gate/lifecycle logic
- ✅ All use WithSideEffects middleware
- ✅ All are synchronous
- ✅ Atomic steps — partial failure does not roll back previous successes

### REQ-NF-004 (backward compatibility)
- ✅ Existing test suites pass (pre-existing failures only, no new failures)
- ✅ No changes to existing tool signatures
- ✅ All composite actions are additive entries in DispatchAction maps

## Code Quality

- Follows existing DispatchAction pattern consistently
- Reuses docRegisterOne, docApproveOne, AdvanceFeatureStatus, CountNonTerminalTasks, MaybeAutoAdvancePlan
- Structured next_action objects follow consistent schema: tool, action, params, description
- No AI/LLM calls inside MCP server
- Human gates are hard stops

## Verdict

**PASS** — all 22 functional requirements and 4 non-functional requirements are satisfied. Implementation follows existing patterns, adds no new abstractions, and maintains backward compatibility.
