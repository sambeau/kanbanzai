# Decision Record — Handoff Capability Gap

| Field  | Value                                        |
|--------|----------------------------------------------|
| ID     | DEC-01KR484HRQ97X                             |
| Date   | 2026-05-08                                   |
| Status | proposed                                     |
| Author | doc-reconciliation (P61 Track D, B63-F1)     |
| Ref    | P61-design-handoff-resilience.md (Track D, open question 3) |

---

## Context

During P61 Track D documentation reconciliation, inspection of the handoff toolchain confirmed that three capabilities described in user-facing documentation do not exist in the implementation:

| Claimed capability | Source document | Code reality |
|--------------------|-----------------|--------------|
| Spec section injection | `AGENTS.md` handoff section | `handoff_tool.go` does not pull spec documents |
| Conflict annotation | `.github/copilot-instructions.md` handoff section | `handoff_tool.go` does not call `conflict_domain_check` |
| Graph traversal | `.github/copilot-instructions.md` handoff section | `handoff_tool.go` does not execute graph queries |

These claims have been **removed** from `AGENTS.md` and `.github/copilot-instructions.md` as part of P61 Track D (REQ-003, REQ-004). The current handoff tool correctly routes through the pipeline-3.0 path, assembles role/skill/knowledge context, and delivers a sub-agent prompt — but does not inject spec content, annotate conflicts, or surface graph relationships.

The P61 design explicitly states this is out of scope: "Add the missing handoff capabilities (spec sections, conflict annotations, graph traversal) under this plan. Those are separate features. This plan only flags the gap (T6) so a decision can be made elsewhere."

---

## Capability Gap Description

### 1. Spec Section Injection

**What was claimed:** Handoff injects relevant sections from the approved feature spec into the context packet, giving the sub-agent direct access to requirements and acceptance criteria without a separate spec read.

**What exists:** Handoff delivers the spec *document path* as metadata (from the feature entity), but does not open the spec or extract sections. The sub-agent must call `read_file` or `doc(action: content)` independently.

**Value if implemented:** Sub-agents receive spec AC sections inline — no extra tool call, no risk of reading a stale path.

### 2. Conflict Annotation

**What was claimed:** Handoff calls the conflict-domain check and annotates the context packet with per-task conflict risk against currently active tasks.

**What exists:** No conflict check is performed. Sub-agents receive no conflict risk data. The orchestrator must manually call `conflict(action: check)` before dispatch.

**Value if implemented:** Every sub-agent prompt would include a conflict annotation, making it safe for the orchestrator to omit the explicit conflict-check step.

### 3. Graph Traversal

**What was claimed:** Handoff executes graph queries to surface related code nodes (callers, callees, related functions) for the task's files and includes them in the context packet.

**What exists:** No graph queries are executed. Sub-agents receive the `graph_project` name as a string but must call `search_graph`, `trace_path`, or `get_code_snippet` themselves.

**Value if implemented:** Sub-agents begin with structural code context pre-assembled, reducing the number of tool calls needed for orientation.

---

## Decision Options

**Option A — Implement as a future feature.**
File three features (or one combined feature) under a new plan targeting handoff context quality. Prioritise by impact: spec section injection likely highest value; graph traversal highest complexity.

**Option B — Accept the gap; update guidance only.**
Leave the capabilities unimplemented. Ensure all documentation is accurate (already done by P61 Track D). Update the `implement-task` skill to remind sub-agents to read the spec themselves, check conflicts via `conflict()`, and orient via `search_graph`.

**Option C — Hybrid.**
Implement spec section injection (highest value, lowest risk) in the near term. Defer conflict annotation and graph traversal until the broader handoff context-quality plan is ready.

---

## Recommended Next Steps

1. **File as future features.** When capacity allows, create a plan (e.g. "Handoff context quality") containing:
   - FEAT: Spec section injection in handoff context packet
   - FEAT: Conflict annotation in handoff context packet
   - FEAT: Graph traversal pre-population in handoff context packet

2. **Until implemented**, ensure the `implement-task` skill and sub-agent instructions remind implementers to:
   - Read the spec document themselves via `doc(action: content)` or `read_file`
   - Check for task conflicts via `conflict(action: check)` before starting
   - Orient in the codebase via `search_graph` before editing

3. **Track this decision.** Entity `DEC-01KR484HRQ97X` records this gap in the workflow system. Reference it when scoping the future plan.
