# Implementation Plan: MCP Tool Description Audit

**Feature:** FEAT-01KN5-8J257X02 (tool-description-audit)
**Specification:** `work/spec/3.0-tool-description-audit.md`
**Design reference:** `work/design/kanbanzai-3.0-workflow-and-tooling-v2.md` §5, §6.4
**Status:** Draft

---

## Overview

This plan decomposes the MCP Tool Description Audit into 8 tasks organised around the specification's three priority tiers plus cross-cutting concerns (error messages, test scenarios, and token verification). The audit rewrites `mcp.WithDescription(...)` strings, parameter `mcp.Description(...)` strings, and error message strings across all 22 MCP tool handlers — no tool logic changes.

### Scope boundaries (carried forward from specification)

- **In scope:** Tool description rewrites (ACI principles), parameter description rewrites (conditional requirements), error message rewrites (three-part template), agent-driven test scenarios, token budget verification
- **Out of scope:** Gate-failure error messages (owned by FEAT-01KN5-88R6ZYPV), tool logic changes, new tools or parameters, skill/copilot-instructions content, hard tool filtering per role

---

## Task Breakdown

### Task 1: Agent-Driven Test Scenarios

**Objective:** Create the scenario file containing 5–10 representative task scenarios for agent-driven testing. This task has no dependencies and should be completed first so that scenarios are available for validating each priority tier.

**Specification references:** FR-011, FR-012

**Input context:**
- `work/spec/3.0-tool-description-audit.md` §Agent-Driven Testing — required workflow patterns and scenario format
- `internal/mcp/groups.go` — tool groupings and tool names
- Priority tier assignments from FR-007

**Output artifacts:**
- New file `work/test/tool-description-scenarios.md` containing 5–10 scenarios, each with:
  - Natural-language task description
  - Expected tool-call sequence
  - At least one decision point where the agent must choose between plausible tools
- Scenarios must collectively cover all Priority 1 and Priority 2 tools
- The five required workflow patterns must each have at least one scenario:
  1. Advancing a feature through a lifecycle stage
  2. Claiming and completing a task (queue → claim → work → finish)
  3. Decomposing a feature into tasks (propose → review → apply)
  4. Creating and registering a document
  5. Querying project status and finding blocked work

**Dependencies:** None — can start immediately

---

### Task 2: Rewrite Priority 1 Tool Descriptions (High-Frequency Tools)

**Objective:** Rewrite the `mcp.WithDescription(...)` and parameter `mcp.Description(...)` strings for the six high-frequency tools to satisfy all five ACI description principles (FR-001 through FR-005).

**Specification references:** FR-001, FR-002, FR-003, FR-004, FR-005, FR-006, FR-007

**Input context:**
- `work/spec/3.0-tool-description-audit.md` §ACI Description Principles (FR-001 through FR-005)
- Tool files to modify:
  - `internal/mcp/entity_tool.go` — `entity` tool (most complex: action-dispatched, many conditional parameters)
  - `internal/mcp/doc_tool.go` — `doc` tool (action-dispatched, multiple parameter sets per action)
  - `internal/mcp/handoff_tool.go` — `handoff` tool (workflow predecessor to spawn_agent)
  - `internal/mcp/next_tool.go` — `next` tool (queue inspection vs claim mode)
  - `internal/mcp/finish_tool.go` — `finish` tool (single vs batch mode, optional knowledge/retro)
  - `internal/mcp/status_tool.go` — `status` tool (pure query, multiple ID-type scopes)

**Output artifacts:**
- Modified `internal/mcp/entity_tool.go` — rewritten description and parameter descriptions
- Modified `internal/mcp/doc_tool.go` — rewritten description and parameter descriptions
- Modified `internal/mcp/handoff_tool.go` — rewritten description and parameter descriptions
- Modified `internal/mcp/next_tool.go` — rewritten description and parameter descriptions
- Modified `internal/mcp/finish_tool.go` — rewritten description and parameter descriptions
- Modified `internal/mcp/status_tool.go` — rewritten description and parameter descriptions

**ACI checklist for each tool (FR-001 through FR-005):**
1. First sentence answers "when/why to use this" — no mechanical verb opener
2. At least one negative-guidance statement ("Use INSTEAD OF...", "Do NOT use for...")
3. Workflow position stated where applicable ("Call AFTER...", "Call BEFORE...")
4. Parameter relationships explicit — which params required for which actions
5. Top-level description ≤ 200 tokens (cl100k_base); parameter descriptions excluded from count

**Constraints:**
- Only modify string literals in `mcp.WithDescription(...)`, `mcp.WithTitleAnnotation(...)`, and `mcp.Description(...)` calls
- Do not change function signatures, parameter schemas, validation logic, or return structures (NFR-001)
- All existing tests must continue to pass without modification (unless they assert on exact description text)

**Dependencies:** None — can start immediately, parallel with Task 1

---

### Task 3: Rewrite Priority 2 Tool Descriptions (Decision-Point Tools)

**Objective:** Rewrite the `mcp.WithDescription(...)` and parameter `mcp.Description(...)` strings for the three decision-point tools to satisfy all five ACI description principles.

**Specification references:** FR-001, FR-002, FR-003, FR-004, FR-005, FR-006, FR-007

**Input context:**
- `work/spec/3.0-tool-description-audit.md` §ACI Description Principles (FR-001 through FR-005)
- Tool files to modify:
  - `internal/mcp/decompose_tool.go` — `decompose` tool (propose → review → apply workflow)
  - `internal/mcp/merge_tool.go` — `merge` tool (check → execute workflow)
  - `internal/mcp/pr_tool.go` — `pr` tool (create → status → update workflow)

**Output artifacts:**
- Modified `internal/mcp/decompose_tool.go` — rewritten description and parameter descriptions
- Modified `internal/mcp/merge_tool.go` — rewritten description and parameter descriptions
- Modified `internal/mcp/pr_tool.go` — rewritten description and parameter descriptions

**ACI checklist:** Same as Task 2 (FR-001 through FR-005 applied to each tool)

**Constraints:** Same as Task 2 — string literals only, no logic changes

**Dependencies:** Task 2 must complete first (FR-007: Priority 1 before Priority 2). Task 5 (Priority 1 agent testing) must also be complete before this tier begins.

---

### Task 4: Rewrite Priority 3 Tool Descriptions (Query and Support Tools)

**Objective:** Rewrite the `mcp.WithDescription(...)` and parameter `mcp.Description(...)` strings for the thirteen query and support tools to satisfy all five ACI description principles.

**Specification references:** FR-001, FR-002, FR-003, FR-004, FR-005, FR-006, FR-007

**Input context:**
- `work/spec/3.0-tool-description-audit.md` §ACI Description Principles (FR-001 through FR-005)
- Tool files to modify:
  - `internal/mcp/knowledge_tool.go` — `knowledge` tool
  - `internal/mcp/doc_intel_tool.go` — `doc_intel` tool
  - `internal/mcp/profile_tool.go` — `profile` tool
  - `internal/mcp/estimate_tool.go` — `estimate` tool
  - `internal/mcp/conflict_tool.go` — `conflict` tool
  - `internal/mcp/retro_tool.go` — `retro` tool
  - `internal/mcp/health_tool.go` — `health` tool
  - `internal/mcp/worktree_tool.go` — `worktree` tool
  - `internal/mcp/branch_tool.go` — `branch` tool
  - `internal/mcp/cleanup_tool.go` — `cleanup` tool
  - `internal/mcp/incident_tool.go` — `incident` tool
  - `internal/mcp/checkpoint_tool.go` — `checkpoint` tool
  - `internal/mcp/server_info_tool.go` — `server_info` tool

**Output artifacts:**
- All thirteen tool files modified with rewritten descriptions and parameter descriptions

**ACI checklist:** Same as Tasks 2–3 (FR-001 through FR-005 applied to each tool)

**Constraints:** Same as Tasks 2–3 — string literals only, no logic changes

**Dependencies:** Task 3 must complete first (FR-007: Priority 2 before Priority 3). Task 6 (Priority 2 agent testing) must also be complete before this tier begins.

---

### Task 5: Agent-Driven Testing — Priority 1 Tier

**Objective:** Run agent-driven testing on the rewritten Priority 1 tool descriptions. Record results, identify failures, and iterate on descriptions until the agent selects the correct tool sequence without mis-selection.

**Specification references:** FR-013, FR-014

**Input context:**
- `work/test/tool-description-scenarios.md` — scenarios from Task 1 (those covering Priority 1 tools)
- The six rewritten Priority 1 tool files from Task 2
- FR-013 testing process: present scenario → record selections → fix descriptions on failure → re-test with fresh session

**Output artifacts:**
- New file `work/test/tool-description-test-results.md` (or append if already created) with results for Priority 1 tier
- Each test record includes: scenario ID, observed tool sequence, pass/fail, and (for failures) description change made
- Any additional modifications to Priority 1 tool files in response to observed failures

**Process:**
1. For each scenario covering Priority 1 tools, present the task to an agent with only the MCP tool list for guidance
2. Record which tools the agent selects, in what order, and where it goes wrong
3. If the agent mis-selects, rewrite the relevant description and re-test with a fresh session
4. Continue until the agent selects the correct tool sequence

**Dependencies:** Task 1 (scenarios) and Task 2 (Priority 1 descriptions) must both complete first

---

### Task 6: Agent-Driven Testing — Priority 2 and Priority 3 Tiers

**Objective:** Run agent-driven testing on the rewritten Priority 2 and Priority 3 tool descriptions. Record results for both tiers, iterate descriptions as needed.

**Specification references:** FR-013, FR-014

**Input context:**
- `work/test/tool-description-scenarios.md` — scenarios covering Priority 2 and Priority 3 tools
- Rewritten tool files from Task 3 and Task 4
- `work/test/tool-description-test-results.md` — existing results from Task 5 (for format consistency)

**Output artifacts:**
- Append Priority 2 and Priority 3 test results to `work/test/tool-description-test-results.md`
- Each record: scenario ID, tier, observed tool sequence, pass/fail, description changes made
- Any additional modifications to Priority 2/3 tool files in response to failures

**Process:** Same as Task 5, applied to Priority 2 scenarios first, then Priority 3 scenarios

**Dependencies:** Task 3 (Priority 2 descriptions) must complete before Priority 2 testing. Task 4 (Priority 3 descriptions) must complete before Priority 3 testing. Task 5 (Priority 1 testing) must complete before this task begins (ensures results file format is established and Priority 1 is validated).

---

### Task 7: Error Message Audit — Priority Handlers

**Objective:** Audit and rewrite all error messages in the four priority tool handlers (`finish`, `doc`, `decompose`, `entity` create action) to follow the three-part actionable error template. Exclude gate-failure error messages in `entity(action: "transition")`.

**Specification references:** FR-008, FR-009, NFR-004

**Input context:**
- `work/spec/3.0-tool-description-audit.md` §Actionable Error Messages — three-part template format
- Tool files to audit:
  - `internal/mcp/finish_tool.go` — validation failures on missing task_id, missing summary, wrong status, auto-transition failures
  - `internal/mcp/doc_tool.go` — registration failures (missing path, type, title), approval prerequisites, batch validation
  - `internal/mcp/decompose_tool.go` — input validation, prerequisite checks
  - `internal/mcp/entity_tool.go` — `entityCreateOne` and `entityCreateAction` error paths only (not transition gate errors)
- `internal/mcp/shared.go` or equivalent — shared error helpers like `ActionError`, `DispatchAction`

**Output artifacts:**
- Modified `internal/mcp/finish_tool.go` — rewritten error messages
- Modified `internal/mcp/doc_tool.go` — rewritten error messages
- Modified `internal/mcp/decompose_tool.go` — rewritten error messages
- Modified `internal/mcp/entity_tool.go` — rewritten error messages in create action only
- Updated test files if they assert on exact error message text

**Error template (FR-008):**
```
Cannot {action} {entity}: {reason}.

To resolve:
  {recovery_step}
```

**Constraints:**
- Every `fmt.Errorf(...)` and `mcp.NewToolResultError(...)` in the target handlers must produce a message containing: what failed, why, and a recovery action
- Error messages that currently say "X is required" must include action context and recovery hint
- Error messages wrapping internal errors must lead with user-facing context before internal detail
- Must NOT expose Go package paths, internal function names, or stack traces (NFR-004)
- Gate-failure errors in `entity(action: "transition")` are excluded — owned by FEAT-01KN5-88R6ZYPV

**Dependencies:** None — can run in parallel with Tasks 1–4. Does not depend on description rewrites.

---

### Task 8: Error Message Audit — Remaining Handlers

**Objective:** Audit and rewrite all error messages in the remaining 18 tool handlers (those not covered by Task 7) to follow the three-part actionable error template.

**Specification references:** FR-008, FR-010, NFR-004

**Input context:**
- `work/spec/3.0-tool-description-audit.md` §Actionable Error Messages
- All tool files not covered by Task 7:
  - `internal/mcp/status_tool.go`
  - `internal/mcp/next_tool.go`
  - `internal/mcp/handoff_tool.go`
  - `internal/mcp/health_tool.go`
  - `internal/mcp/server_info_tool.go`
  - `internal/mcp/knowledge_tool.go`
  - `internal/mcp/profile_tool.go`
  - `internal/mcp/estimate_tool.go`
  - `internal/mcp/conflict_tool.go`
  - `internal/mcp/retro_tool.go`
  - `internal/mcp/worktree_tool.go`
  - `internal/mcp/merge_tool.go`
  - `internal/mcp/pr_tool.go`
  - `internal/mcp/branch_tool.go`
  - `internal/mcp/cleanup_tool.go`
  - `internal/mcp/doc_intel_tool.go`
  - `internal/mcp/incident_tool.go`
  - `internal/mcp/checkpoint_tool.go`
- Also audit `entity_tool.go` non-create actions (get, list, update) for error messages — but NOT transition gate-failure paths
- Shared helpers: `internal/mcp/action_dispatch.go`, `internal/mcp/batch.go` if they produce error messages

**Output artifacts:**
- All listed tool files modified with rewritten error messages
- Updated test files if they assert on exact error message text

**Error template:** Same three-part template as Task 7

**Constraints:** Same as Task 7 — three-part template, no internal details exposed, gate-failure paths excluded

**Dependencies:** Task 7 must complete first (establishes the error message pattern and any shared helpers). Also depends on Task 2 completing to avoid merge conflicts on shared files — specifically `entity_tool.go`, `doc_tool.go`, `finish_tool.go`, `next_tool.go`, `handoff_tool.go`, and `status_tool.go` are modified by both description rewrites and error message rewrites.

---

## Dependency Graph

```
Task 1 (Scenarios)─────────────────┐
                                   ├──▶ Task 5 (P1 Testing) ──▶ Task 3 (P2 Descriptions) ──┐
Task 2 (P1 Descriptions)───────────┘                                                        │
                                                                                             ├──▶ Task 6 (P2+P3 Testing)
                                                          Task 4 (P3 Descriptions) ◀────────┘         │
                                                             │                                         │
                                                             └─────────────────────────────────────────┘
                                                                         (P3 testing waits for Task 4)

Task 7 (Error Audit: Priority) ──────────────────────────────▶ Task 8 (Error Audit: Remaining)
```

**Parallel execution opportunities:**
- **Wave 1:** Tasks 1, 2, and 7 can all start immediately in parallel
- **Wave 2:** Task 5 starts after Tasks 1 + 2; Task 8 starts after Task 7 + Task 2
- **Wave 3:** Task 3 starts after Task 5
- **Wave 4:** Task 4 starts after Task 3 + Task 6 (P2 portion); Task 6 (P2 portion) starts after Task 3
- **Wave 5:** Task 6 (P3 portion) starts after Task 4

**Serialisation rationale:**
- Priority tier ordering (Tasks 2 → 3 → 4) is mandated by FR-007
- Agent-driven testing per tier (Tasks 5, 6) is mandated by FR-013 — each tier must be tested before the next begins
- Error audit Task 8 depends on Task 7 for pattern establishment and on description tasks to avoid merge conflicts on shared files

---

## Traceability Matrix

| Requirement | Task(s) | Notes |
|---|---|---|
| **FR-001** (when-to-use first sentence) | 2, 3, 4 | Applied per priority tier |
| **FR-002** (negative guidance) | 2, 3, 4 | Applied per priority tier |
| **FR-003** (workflow position) | 2, 3, 4 | Applied per priority tier |
| **FR-004** (parameter relationships) | 2, 3, 4 | Applied per priority tier |
| **FR-005** (200-token limit) | 2, 3, 4 | Verified during each description rewrite |
| **FR-006** (full tool coverage) | 2, 3, 4 | 6 + 3 + 13 = 22 tools total |
| **FR-007** (priority order) | 2, 3, 4, 5, 6 | Task dependencies enforce tier ordering |
| **FR-008** (three-part error template) | 7, 8 | Template applied across all handlers |
| **FR-009** (priority error targets) | 7 | finish, doc, decompose, entity create |
| **FR-010** (remaining error handlers) | 8 | All other tool handlers |
| **FR-011** (test scenario set) | 1 | 5–10 scenarios with required format |
| **FR-012** (workflow pattern coverage) | 1 | Five required patterns covered |
| **FR-013** (agent-driven testing per tier) | 5, 6 | Test → fix → re-test cycle per tier |
| **FR-014** (test result recording) | 5, 6 | Results in `work/test/tool-description-test-results.md` |
| **NFR-001** (no behaviour changes) | 2, 3, 4, 7, 8 | Constraint on all modification tasks |
| **NFR-002** (valid UTF-8, no control chars) | 2, 3, 4 | Constraint on all description rewrites |
| **NFR-003** (incremental delivery) | 2, 3, 4, 5, 6 | Each tier merged independently |
| **NFR-004** (no internal details in errors) | 7, 8 | Constraint on all error message rewrites |

---

## File Inventory

All files that will be created or modified, grouped by task:

**New files:**
- `work/test/tool-description-scenarios.md` (Task 1)
- `work/test/tool-description-test-results.md` (Task 5, appended by Task 6)

**Modified files (descriptions):**
- `internal/mcp/entity_tool.go` (Task 2, Task 7)
- `internal/mcp/doc_tool.go` (Task 2, Task 7)
- `internal/mcp/handoff_tool.go` (Task 2, Task 8)
- `internal/mcp/next_tool.go` (Task 2, Task 8)
- `internal/mcp/finish_tool.go` (Task 2, Task 7)
- `internal/mcp/status_tool.go` (Task 2, Task 8)
- `internal/mcp/decompose_tool.go` (Task 3, Task 7)
- `internal/mcp/merge_tool.go` (Task 3, Task 8)
- `internal/mcp/pr_tool.go` (Task 3, Task 8)
- `internal/mcp/knowledge_tool.go` (Task 4, Task 8)
- `internal/mcp/doc_intel_tool.go` (Task 4, Task 8)
- `internal/mcp/profile_tool.go` (Task 4, Task 8)
- `internal/mcp/estimate_tool.go` (Task 4, Task 8)
- `internal/mcp/conflict_tool.go` (Task 4, Task 8)
- `internal/mcp/retro_tool.go` (Task 4, Task 8)
- `internal/mcp/health_tool.go` (Task 4, Task 8)
- `internal/mcp/worktree_tool.go` (Task 4, Task 8)
- `internal/mcp/branch_tool.go` (Task 4, Task 8)
- `internal/mcp/cleanup_tool.go` (Task 4, Task 8)
- `internal/mcp/incident_tool.go` (Task 4, Task 8)
- `internal/mcp/checkpoint_tool.go` (Task 4, Task 8)
- `internal/mcp/server_info_tool.go` (Task 4, Task 8)

**Shared file conflict zones:**
- `entity_tool.go`, `doc_tool.go`, `finish_tool.go`, `decompose_tool.go` are touched by both description rewrite tasks and error message audit tasks. The dependency ordering ensures these don't run concurrently on the same file.
```

Now let me write this file and register it as a document: