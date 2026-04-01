# Specification: Stage-Aware Context Assembly (Kanbanzai 3.0)

**Feature:** FEAT-01KN5-8J25K4QD (stage-aware-context-assembly)
**Design reference:** `work/design/kanbanzai-3.0-workflow-and-tooling-v2.md` §7, §8, §9, §13
**Status:** Draft

---

## 1. Overview

Stage-aware context assembly makes the `handoff` and `next` tools lifecycle-conscious: before assembling context for a task, they validate that the parent feature is in the correct lifecycle state; they vary the content included in assembled context based on the feature's current stage; they embed orchestration pattern signals, effort budgets, and tool subset guidance in high-attention positions; and they reinforce the filesystem-output convention for orchestrator tasks. For the initial 3.0 release, effort budgets, tool subsets, and orchestration patterns are hardcoded in the assembly logic (the binding registry integration is a separate feature). The `finish` tool enforces a 500-character limit on task completion summaries to support the filesystem-output convention.

---

## 2. Scope

### In scope

- Lifecycle state validation in `handoff` and `next` before context assembly
- Stage-specific content inclusion/exclusion strategy for each feature lifecycle stage
- Orchestration pattern signalling text (`single-agent` vs `orchestrator-workers`) in assembled context
- Effort budget text embedded in the high-attention zone of assembled context
- Tool subset guidance (soft filtering) embedded in assembled context
- Filesystem-output convention text in orchestrator context
- Hardcoded stage configuration (effort budgets, tool subsets, orchestration patterns, allowed feature states) for the 3.0 release
- 500-character limit on the `finish` tool's `summary` field
- Actionable error messages for lifecycle state validation failures

### Explicitly excluded

- Hard tool filtering (dynamically hiding tools from the MCP session) — deferred post-3.0 per design §9.3
- Binding registry reading, caching, or schema (that is feature FEAT-01KN5-88PDPE8V, binding-registry)
- The content of specific roles, skills, or templates (that is plan P16, skills redesign)
- Gate enforcement on feature lifecycle transitions (that is feature FEAT-01KN5-8J24S2XW, mandatory-stage-gates)
- The 10-step assembly pipeline mechanics, attention-curve ordering, and token budget management (that is feature FEAT-01KN5-88PE43M6, context-assembly-pipeline)
- Changes to the feature lifecycle state machine or transition rules

---

## 3. Functional Requirements

### FR-001: Lifecycle State Validation on Handoff

The `handoff` tool MUST validate that the task's parent feature is in a lifecycle state that permits work on the task before assembling context. The tool MUST resolve the task's parent feature and read its current status. If the feature is not in an appropriate state, the tool MUST reject the request with an actionable error message and MUST NOT assemble or return context.

**Acceptance criteria:**
- Calling `handoff` on an implementation task whose parent feature is in `specifying` returns an error — not assembled context
- Calling `handoff` on an implementation task whose parent feature is in `developing` succeeds past validation
- The tool does not proceed to context assembly when validation fails
- A task with no parent feature returns an error identifying the orphaned task

---

### FR-002: Lifecycle State Validation on Next (Claim Mode)

The `next` tool in claim mode MUST perform the same lifecycle state validation as `handoff` (FR-001) before claiming the task and assembling context. If the feature is not in an appropriate state, the tool MUST reject the claim with an actionable error message. The task MUST NOT be transitioned to `active` when validation fails. Queue mode (`next` without an `id`) is unaffected by this requirement.

**Acceptance criteria:**
- Calling `next(id="TASK-...")` where the parent feature is in `proposed` returns an error and the task remains in `ready` status
- Calling `next(id="TASK-...")` where the parent feature is in `developing` claims the task and returns assembled context
- Queue mode (`next()` with no id) is not affected by lifecycle validation
- Calling `next(id="FEAT-...")` where the feature is in `proposed` returns an error (no task is claimed)

---

### FR-003: Stage-to-Allowed-States Mapping

The lifecycle validation (FR-001, FR-002) MUST use a hardcoded mapping from task context to allowed parent feature states. The mapping MUST be:

| Allowed Feature States | Task Context |
|---|---|
| `designing` | Tasks dispatched during design stage |
| `specifying` | Tasks dispatched during specification stage |
| `dev-planning` | Tasks dispatched during dev-planning stage |
| `developing`, `needs-rework` | Implementation tasks |
| `reviewing`, `needs-rework` | Review tasks |

For the 3.0 release, validation MUST use a simplified heuristic: the parent feature's current lifecycle state is checked against all non-terminal working states (`designing`, `specifying`, `dev-planning`, `developing`, `reviewing`, `needs-rework`). A feature in `proposed`, `done`, `superseded`, or `cancelled` MUST be rejected for any task. This simplified check ensures no task is assembled against a terminal or pre-work feature while the binding registry integration (which enables precise stage-to-task-type mapping) is pending.

**Acceptance criteria:**
- A task whose parent feature is in `developing` passes validation
- A task whose parent feature is in `designing` passes validation
- A task whose parent feature is in `needs-rework` passes validation
- A task whose parent feature is in `proposed` fails validation
- A task whose parent feature is in `done` fails validation
- A task whose parent feature is in `superseded` fails validation
- A task whose parent feature is in `cancelled` fails validation

---

### FR-004: Actionable Error Messages for Validation Failures

When lifecycle state validation rejects a request, the error message MUST follow the actionable error template from design §6.2. The message MUST include: (1) what failed — the action, task ID, and feature ID; (2) why it failed — the current feature state and the fact that it is not in a working state; (3) what to do next — a specific recovery action with tool call examples.

The error format MUST be:

```
Cannot assemble context for {task_id}: parent feature {feature_id} is in
'{current_state}', which is not an active working state.

To resolve:
1. Check feature status: entity(action: "get", id: "{feature_id}")
2. Advance the feature to a working state (designing, specifying, dev-planning,
   developing, or reviewing) before dispatching tasks.
```

**Acceptance criteria:**
- The error message contains the task ID
- The error message contains the parent feature ID
- The error message contains the current feature state in quotes
- The error message contains at least one tool call example for recovery
- The error message follows the three-part template: what failed, why, what to do next

---

### FR-005: Stage-Specific Assembly Strategy

The context assembly pipeline MUST vary what content is included and excluded based on the parent feature's current lifecycle stage. The assembly logic MUST use the following inclusion/exclusion table:

| Stage | Primary Context (included) | Excluded Context | Orchestration |
|---|---|---|---|
| **designing** | Related decisions, parent plan context, design template | Implementation tool guidance, file paths, test expectations | single-agent |
| **specifying** | Approved design document (full content), spec template, acceptance criteria format | Implementation tool guidance, file paths | single-agent |
| **dev-planning** | Approved spec (full content), decomposition guidance, dependency format, sizing constraints | Implementation details, review tool guidance | single-agent |
| **developing** | Spec (relevant sections only), task description, file paths, test expectations, related knowledge entries | Planning tool guidance, review rubrics | orchestrator-workers |
| **reviewing** | Spec (relevant sections), implementation summary, review rubric, verdict format, previous review findings (if re-review) | Implementation tool guidance, planning tool guidance | orchestrator-workers |
| **needs-rework** | Same as `developing` — spec (relevant sections), task description, file paths, plus previous review findings | Planning tool guidance, review rubrics | orchestrator-workers |

**Acceptance criteria:**
- Context assembled for a task whose parent feature is in `designing` does not contain file paths or implementation tool guidance
- Context assembled for a task whose parent feature is in `specifying` includes the full design document content (not just relevant sections)
- Context assembled for a task whose parent feature is in `developing` includes file paths and test expectations
- Context assembled for a task whose parent feature is in `developing` does not contain review rubrics
- Context assembled for a task whose parent feature is in `reviewing` includes the review rubric and verdict format
- Context assembled for a task whose parent feature is in `reviewing` does not contain implementation tool guidance
- Context assembled for a task whose parent feature is in `needs-rework` includes previous review findings when available

---

### FR-006: Orchestration Pattern Signalling

The assembled context MUST include an orchestration pattern statement positioned in the high-attention zone (above the task description, below the role identity). The statement MUST be determined by the parent feature's lifecycle stage using the mapping in FR-005. The exact text MUST be one of:

For `single-agent` stages (designing, specifying, dev-planning):

```
## Orchestration

This is a **single-agent** task. Complete it directly — do not delegate to sub-agents.
```

For `orchestrator-workers` stages (developing, reviewing, needs-rework):

```
## Orchestration

This is a **multi-agent** task. Dispatch independent sub-tasks to sub-agents
in parallel using handoff + spawn_agent.
```

**Acceptance criteria:**
- Context assembled for a feature in `specifying` contains the single-agent text verbatim, including "do not delegate to sub-agents"
- Context assembled for a feature in `developing` contains the multi-agent text verbatim, including "Dispatch independent sub-tasks"
- The orchestration section appears before the task description in the assembled output
- The orchestration section appears after the role identity in the assembled output
- No other orchestration pattern values are emitted (only `single-agent` or `orchestrator-workers`)

---

### FR-007: Effort Budget in Assembled Context

The assembled context MUST include an effort budget section positioned in the high-attention zone — above the task description and below the role identity, adjacent to the orchestration pattern (FR-006). The effort budget MUST be determined by the parent feature's lifecycle stage using the following hardcoded values:

| Stage | Effort Budget Text |
|---|---|
| designing | `5–15 tool calls. Read related designs, query decisions, draft structured document.` |
| specifying | `5–15 tool calls. Read design document, query knowledge, check related decisions, draft each required section.` |
| dev-planning | `5–10 tool calls. Read spec, decompose into tasks with dependencies, estimate effort, produce plan document.` |
| developing | `10–50 tool calls per task. Read spec section, implement, test, iterate.` |
| reviewing | `5–10 tool calls per review dimension.` |
| needs-rework | `10–50 tool calls per task. Read review findings, address issues, test fixes, iterate.` |

The format in the assembled context MUST be:

```
## Effort Expectations

This is a **{stage_name}** task.
Expected effort: {effort_budget_text}

Do NOT skip to implementation. Complete this stage's deliverables before advancing.
```

For the `developing` and `needs-rework` stages, the final line MUST instead read:

```
Do NOT skip testing. Every change must be verified before marking done.
```

**Acceptance criteria:**
- Context assembled for a feature in `specifying` contains "5–15 tool calls" and "Read design document"
- Context assembled for a feature in `developing` contains "10–50 tool calls per task"
- Context assembled for a feature in `reviewing` contains "5–10 tool calls per review dimension"
- The effort budget section contains the stage name in bold
- The effort budget section appears before the task description in the assembled output
- The "Do NOT skip" warning line is present in every stage's effort budget section
- Sequential stages (designing, specifying, dev-planning) use the "skip to implementation" warning
- Implementation stages (developing, needs-rework) use the "skip testing" warning

---

### FR-008: Tool Subset Guidance in Assembled Context

The assembled context MUST include a "Tools for This Task" section listing the primary tools and excluded tools for the current stage. All tools remain available to the agent — this is guidance, not restriction. The tool subsets MUST be determined by the parent feature's lifecycle stage using the following hardcoded values:

| Stage | Primary Tools | Excluded Tools |
|---|---|---|
| designing | entity, doc, doc_intel, knowledge, status | decompose, merge, pr, worktree, finish |
| specifying | entity, doc, doc_intel, knowledge, status | decompose, merge, pr, worktree, finish |
| dev-planning | entity, doc, knowledge, decompose, estimate, status | merge, pr, worktree |
| developing | entity, handoff, next, finish, knowledge, status, branch, worktree | decompose, doc_intel |
| reviewing | entity, doc, doc_intel, knowledge, finish, status | decompose, merge, worktree, handoff |
| needs-rework | entity, handoff, next, finish, knowledge, status, branch, worktree | decompose, doc_intel |

The format in the assembled context MUST be:

```
## Tools for This Task

Primary tools: {comma-separated list}
Do NOT use: {comma-separated list} (these are for other stages)
```

**Acceptance criteria:**
- Context assembled for a feature in `designing` lists entity, doc, doc_intel, knowledge, status as primary tools
- Context assembled for a feature in `designing` lists decompose, merge, pr, worktree, finish as excluded tools
- Context assembled for a feature in `developing` lists entity, handoff, next, finish, knowledge, status, branch, worktree as primary tools
- Context assembled for a feature in `reviewing` does not list handoff as a primary tool
- The tool subset section contains the text "Do NOT use"
- The tool subset section contains the text "these are for other stages"

---

### FR-009: Filesystem-Output Convention in Orchestrator Context

When the assembled context is for an orchestrator-workers stage (`developing`, `reviewing`, `needs-rework`), the context MUST include a filesystem-output convention section. This section reinforces that sub-agents write to documents and task records, and the orchestrator reads status not content. The exact text MUST be:

```
## Output Convention

Sub-agents write outputs to documents and task records. Read their status via
`entity(action: "get")` and `doc(action: "get")`. Do not retain sub-agent
conversation output in your context — use references (document IDs, task IDs,
status summaries) instead of contents.
```

This section MUST NOT appear in single-agent stages (designing, specifying, dev-planning).

**Acceptance criteria:**
- Context assembled for a feature in `developing` contains the filesystem-output convention text
- Context assembled for a feature in `reviewing` contains the filesystem-output convention text
- Context assembled for a feature in `needs-rework` contains the filesystem-output convention text
- Context assembled for a feature in `specifying` does not contain the filesystem-output convention text
- The text includes the `entity(action: "get")` and `doc(action: "get")` tool call examples

---

### FR-010: Finish Summary Length Limit

The `finish` tool MUST enforce a maximum length of 500 characters on the `summary` field. If the provided summary exceeds 500 characters, the tool MUST reject the call with an actionable error message. The limit applies to both single-item and batch modes.

**Acceptance criteria:**
- A `finish` call with a 500-character summary succeeds
- A `finish` call with a 501-character summary returns an error
- The error message states the 500-character limit and the actual length provided
- The error message suggests truncating the summary
- In batch mode, a single item with an oversized summary fails independently without affecting other items
- A `finish` call with an empty summary continues to be rejected by the existing "summary is required" validation (no change to that behaviour)

---

### FR-011: Hardcoded Stage Configuration

For the 3.0 release, all stage-specific data (orchestration patterns, effort budgets, tool subsets, allowed feature states) MUST be defined as hardcoded constants or lookup tables in the Go source code. The data MUST be co-located in a single source file (or a clearly bounded section of the assembly module) so that it can be replaced by binding registry lookups in a future release without scattered changes.

**Acceptance criteria:**
- All stage configuration values from FR-003, FR-005, FR-006, FR-007, and FR-008 are defined in a single Go source file
- The file contains no binding registry import or dependency
- Changing an effort budget string for a stage requires editing exactly one location
- The configuration data structure supports lookup by stage name (string key)

---

### FR-012: Handoff Prompt Rendering with Stage-Aware Sections

The `handoff` tool's Markdown prompt rendering MUST include the new stage-aware sections (orchestration pattern, effort budget, tool subset guidance, and filesystem-output convention where applicable) in the high-attention zone. The ordering in the rendered prompt MUST be:

1. Role identity / conventions (existing)
2. Orchestration pattern (FR-006)
3. Effort expectations (FR-007)
4. Tools for this task (FR-008)
5. Output convention (FR-009, orchestrator-workers stages only)
6. Task summary and description (existing)
7. Specification sections (existing)
8. Acceptance criteria (existing)
9. Knowledge constraints (existing)
10. File paths (existing, excluded for designing/specifying stages per FR-005)
11. Additional instructions (existing)

**Acceptance criteria:**
- The orchestration section appears before the task summary in the rendered prompt
- The effort expectations section appears before the task summary in the rendered prompt
- The tools section appears before the task summary in the rendered prompt
- The output convention section (when present) appears before the task summary
- Existing sections (specification, acceptance criteria, knowledge, files, instructions) continue to appear in their current relative order

---

### FR-013: Next Tool Structured Response with Stage-Aware Fields

The `next` tool's claim-mode structured response MUST include the stage-aware data as additional fields in the `context` object. The new fields MUST be:

- `orchestration_pattern`: string, either `"single-agent"` or `"orchestrator-workers"`
- `effort_budget`: object with `stage` (string), `text` (string), and `warning` (string) fields
- `tool_subset`: object with `primary` (list of strings) and `excluded` (list of strings) fields
- `output_convention`: string (present only for orchestrator-workers stages, omitted for single-agent stages)
- `feature_stage`: string, the resolved lifecycle stage used for assembly

**Acceptance criteria:**
- The `next` claim-mode response for a feature in `developing` includes `orchestration_pattern: "orchestrator-workers"`
- The `next` claim-mode response for a feature in `specifying` includes `orchestration_pattern: "single-agent"`
- The `next` claim-mode response includes `feature_stage` matching the parent feature's lifecycle state
- The `next` claim-mode response for a feature in `developing` includes `output_convention` with the convention text
- The `next` claim-mode response for a feature in `specifying` does not include `output_convention`
- The `tool_subset.primary` list for `developing` contains exactly the tools listed in FR-008

---

### FR-014: Graceful Degradation for Missing Parent Feature

When a task has no `parent_feature` field or the parent feature cannot be loaded, the assembly pipeline MUST fall back to the current (non-stage-aware) assembly behaviour. The stage-aware sections (orchestration, effort budget, tool subset, output convention) MUST be omitted. The response MUST include a metadata field `stage_aware: false` to indicate the fallback was used.

**Acceptance criteria:**
- A task with no `parent_feature` field assembles context without stage-aware sections
- A task whose parent feature ID references a non-existent entity assembles context without stage-aware sections
- The response metadata includes `stage_aware: false` when fallback is used
- The response metadata includes `stage_aware: true` when stage-aware assembly succeeds
- Existing context (spec sections, knowledge, files, constraints) continues to be assembled in fallback mode

---

## 4. Non-Functional Requirements

### NFR-001: Assembly Performance

The addition of stage-aware logic MUST NOT increase the latency of `handoff` or `next` (claim mode) by more than 50ms in the p95 case. The stage configuration lookup is a hardcoded map access (O(1)) and MUST NOT involve file I/O or network calls.

---

### NFR-002: Backward Compatibility

Existing `handoff` and `next` tool call signatures MUST NOT change. No new required parameters are introduced. Callers that do not use stage-aware features see unchanged behaviour except for the addition of new sections in the assembled context and new fields in the structured response.

---

### NFR-003: Forward Compatibility with Binding Registry

The hardcoded stage configuration (FR-011) MUST use a data structure whose shape matches the binding registry's `stage_bindings` schema (as defined in the binding registry specification). This ensures that replacing the hardcoded data with binding registry lookups requires changing only the data source, not the data consumers.

---

### NFR-004: Testability

Each stage-aware behaviour (validation, orchestration signal, effort budget, tool subset, output convention, finish limit) MUST be independently testable through unit tests that do not require a full MCP server or filesystem state. The hardcoded stage configuration MUST be accessible to tests for verification against the design document values.

---

## 5. Acceptance Criteria

This section consolidates verification approaches for all functional requirements.

| Requirement | Verification Method |
|---|---|
| FR-001 (Handoff validation) | Unit test: mock feature in terminal state, assert error returned and no context assembled |
| FR-002 (Next validation) | Unit test: mock feature in `proposed`, assert error and task not transitioned to `active` |
| FR-003 (State mapping) | Unit test: iterate all feature states, assert correct accept/reject for each |
| FR-004 (Error messages) | Unit test: assert error string contains task ID, feature ID, current state, and tool call example |
| FR-005 (Stage-specific content) | Unit test per stage: assert presence/absence of each content category |
| FR-006 (Orchestration signal) | Unit test per stage: assert correct signal text in assembled output |
| FR-007 (Effort budget) | Unit test per stage: assert correct budget text and warning line |
| FR-008 (Tool subset) | Unit test per stage: assert correct primary and excluded tool lists |
| FR-009 (Filesystem convention) | Unit test: assert presence for orchestrator-workers stages, absence for single-agent stages |
| FR-010 (Finish summary limit) | Unit test: 500-char summary succeeds, 501-char summary returns error with limit stated |
| FR-011 (Hardcoded config) | Unit test: assert all stages from the design have entries; code review: single-file co-location |
| FR-012 (Handoff rendering) | Unit test: assert section ordering in rendered Markdown prompt |
| FR-013 (Next structured response) | Unit test: assert new fields present and typed correctly in claim-mode JSON |
| FR-014 (Graceful degradation) | Unit test: task with nil parent feature assembles without stage-aware sections, metadata indicates fallback |

---

## 6. Dependencies and Assumptions

### Dependencies

- **Feature lifecycle model:** The feature entity MUST support the lifecycle states `proposed`, `designing`, `specifying`, `dev-planning`, `developing`, `reviewing`, `needs-rework`, `done`, `superseded`, `cancelled` as defined in `internal/model/entities.go`. This is the current state — no changes required.
- **Existing assembly pipeline:** The `assembleContext` function in `internal/mcp/assembly.go` and the `handoff`/`next` tool handlers in `internal/mcp/handoff_tool.go` and `internal/mcp/next_tool.go` are the integration points. Stage-aware logic extends these existing functions.
- **Entity service:** The `entitySvc.Get("feature", featureID, "")` API is used to load the parent feature and read its status. This API exists and is already called by the assembly pipeline.

### Assumptions

- The binding registry feature (FEAT-01KN5-88PDPE8V) will eventually replace the hardcoded stage configuration. The data structure used here MUST be designed for that replacement.
- The context assembly pipeline feature (FEAT-01KN5-88PE43M6) defines the overall pipeline mechanics. This feature contributes stage-specific content and validation to that pipeline. If the pipeline feature is implemented first, this feature extends it; if this feature is implemented first, the pipeline feature will refactor the placement of these sections into the full 10-step pipeline.
- The actionable error template from design §6 is the project-wide error format. This feature applies that template to lifecycle validation errors specifically.
- Tool subset guidance is advisory only for 3.0. Agents may still call excluded tools. Hard filtering is a separate post-3.0 concern.