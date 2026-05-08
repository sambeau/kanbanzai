| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-08T16:29:10Z           |
| Status | approved |
| Author | spec-author                    |

## Problem Statement

This specification implements Track D of the design described in
`work/P61-handoff-resilience-binding-hardening/P61-design-handoff-resilience.md`
(P61-handoff-resilience-binding-hardening/design-p61-design-handoff-resilience, approved).

The current documentation claims handoff capabilities that do not exist in the implementation:
- `internal/mcp/handoff_tool.go` header claims pipeline validation behaviour (true post-fix, but the header is imprecise)
- `internal/mcp/assembly.go` header claims it is "shared by next and handoff" — now used only by `next`
- User-facing docs claim spec sections, conflict annotations, and graph traversal in handoff — none implemented
- `AGENTS.md` and `.github/copilot-instructions.md` handoff sections describe design-intent capabilities not yet built

**Scope:** Update documentation to match current implementation. Flag the capability gap for separate planning.
**Out of scope:** Adding missing handoff capabilities (spec sections, conflict annotations, graph traversal). Those are separate features.

## Requirements

### Functional Requirements

- **REQ-001:** `internal/mcp/handoff_tool.go` header comment must accurately describe the pipeline-3.0 behaviour (unconditional pipeline, no legacy fallback).
- **REQ-002:** `internal/mcp/assembly.go` header comment must clarify it is `next`-only after legacy-removal commits.
- **REQ-003:** `AGENTS.md` handoff section must match current implementation — remove claims of spec sections, conflict annotations, and graph traversal that do not exist.
- **REQ-004:** `.github/copilot-instructions.md` handoff section must match current implementation — remove or qualify design-intent capabilities not yet built.
- **REQ-005:** A planning discussion issue or decision record must be created flagging the capability gap (spec sections, conflict annotations, graph traversal) for separate feature planning.

### Non-Functional Requirements

- **REQ-NF-001:** Documentation changes must not alter any Go source code behaviour. `go build ./...` must pass before and after.
- **REQ-NF-002:** All handoff-related documentation references across the repository must be internally consistent — no document may claim behaviour contradicted by another document.

## Constraints

- Must NOT change Go source code (except header comments).
- Must NOT add new handoff capabilities.
- Must NOT remove references to features that are planned but not yet implemented without flagging them.
- Out of scope: spec sections, conflict annotations, graph traversal in handoff.

## Acceptance Criteria

- **AC-001 (REQ-001):** Given the updated `handoff_tool.go`, when a reader inspects the header comment, then it accurately describes pipeline-3.0 behaviour with no mention of deprecated fallback.
- **AC-002 (REQ-002):** Given the updated `assembly.go`, when a reader inspects the header comment, then it states the file is `next`-only and does not claim shared use with handoff.
- **AC-003 (REQ-003):** Given the updated `AGENTS.md`, when a reader searches for "handoff", then no claims of spec sections, conflict annotations, or graph traversal appear without qualification.
- **AC-004 (REQ-004):** Given the updated `.github/copilot-instructions.md`, when a reader searches for "handoff", then no claims of spec sections, conflict annotations, or graph traversal appear without qualification.
- **AC-005 (REQ-005):** Given the completion of documentation updates, when a project maintainer reviews the capability gap, then a decision record or planning issue exists documenting the gap and proposing next steps.

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Inspection | Code review: verify `handoff_tool.go` header accuracy |
| AC-002 | Inspection | Code review: verify `assembly.go` header accuracy |
| AC-003 | Inspection | grep AGENTS.md for handoff claims; verify none are false |
| AC-004 | Inspection | grep copilot-instructions.md for handoff claims; verify none are false |
| AC-005 | Inspection | Confirm decision record or issue exists for capability gap |
