# Specification: MCP Tool Description Audit (Kanbanzai 3.0)

**Feature:** FEAT-01KN5-8J257X02 (tool-description-audit)
**Design reference:** `work/design/kanbanzai-3.0-workflow-and-tooling-v2.md` §5, §6.4
**Status:** Draft

---

## Overview

This specification defines the requirements for auditing and rewriting all MCP tool descriptions to follow Agent-Computer Interface (ACI) design principles, and for auditing all non-gate error messages to follow a three-part actionable error template. The audit is description-and-error-message-only — no tool logic changes are in scope. The goal is to make tool descriptions answer "when should I use this?" and "what should I use instead?", and to make error messages provide concrete recovery actions, so that agents select the right tool on the first attempt and recover from errors without retry loops.

---

## Scope

### In scope

- Rewriting the `mcp.WithDescription(...)` string for every MCP tool to follow ACI principles
- Rewriting `mcp.Description(...)` strings for tool parameters where the current text is ambiguous about when/whether a parameter is required
- Auditing and rewriting error messages in MCP tool handlers (excluding gate-failure errors) to follow the three-part actionable error template
- Defining and maintaining a set of representative task scenarios for agent-driven testing of rewritten descriptions
- A prioritised audit order across all tools
- Token-budget compliance checking for each tool description

### Explicitly excluded

- **Gate-failure error messages** — these are covered by the mandatory-stage-gates feature (FEAT-01KN5-88R6ZYPV). The `entity(action: "transition")` gate-rejection messages in §3.6 of the design are out of scope for this feature
- **Tool logic changes** — no handler behaviour, parameter validation rules, or return value structures change. Only the human-readable description strings and error message strings change
- **Hard filtering of tools per role** — covered by stage-aware-context-assembly and deferred to post-3.0 (design §9.3)
- **New tools or new parameters** — the tool surface area is fixed; this audit rewrites existing text only
- **Skill content or copilot-instructions.md changes** — tool descriptions are the MCP-level interface; skill-level guidance is a separate concern

---

## Functional Requirements

### ACI Description Principles

**FR-001:** Every MCP tool description MUST lead with a "when to use" sentence. The first sentence MUST answer when and why an agent should reach for this tool, not describe what the tool does mechanically.

**Acceptance criteria:**
- For every tool in the tool surface, the first sentence of the description contains language indicating when or why to use the tool (e.g., "Use when...", "The primary tool for...", "Start here when...")
- No tool's first sentence begins with a mechanical action verb describing API behaviour (e.g., "Create, read, update, and delete..." or "Manage..." as a standalone opener without context on when)

---

**FR-002:** Every MCP tool description MUST include negative guidance — at least one statement identifying what NOT to use the tool for, or what to use INSTEAD of an alternative approach.

**Acceptance criteria:**
- For every tool, the description contains at least one of: "Use this INSTEAD OF...", "Do NOT use this for... — use Y instead", "For X, use Y instead", or equivalent negative-guidance phrasing
- The negative guidance references a specific alternative (another tool name, or a specific anti-pattern like "reading .kbz/ files directly")

---

**FR-003:** Every MCP tool description MUST state the tool's workflow position where applicable — what should be called before or after this tool in a typical workflow sequence.

**Acceptance criteria:**
- For tools that have a natural predecessor or successor in a workflow (e.g., `handoff` before `spawn_agent`, `next` before beginning work, `decompose` after specification approval), the description contains sequencing language ("Call AFTER...", "Call BEFORE...", "Use ... → ... as the standard workflow")
- Tools that are pure queries with no workflow ordering (e.g., `status`, `knowledge` with `action: list`) MAY omit sequencing if no natural predecessor/successor exists, but MUST still satisfy FR-001 and FR-002

---

**FR-004:** Every MCP tool description MUST make parameter relationships explicit where conditional requirements exist. When one parameter value makes another parameter required or changes its interpretation, the description or the relevant parameter descriptions MUST state this.

**Acceptance criteria:**
- For every tool that uses an `action` parameter to dispatch to sub-operations, the description or parameter descriptions state which parameters are required for which actions
- For `entity`: the description states that `type` is required for `create` and `list`, and that `id` is required for `get`, `update`, and `transition`
- For `doc`: the description states which parameters are required per action
- For `finish`: the description states that `task_id` and `summary` are required in single-item mode, and `tasks` is used for batch mode

---

**FR-005:** Every MCP tool description MUST be 200 tokens or fewer, measured by the `cl100k_base` tokeniser (GPT-4 / Claude tokeniser family). Parameter descriptions are counted separately and are not subject to this limit.

**Acceptance criteria:**
- A verification script or test counts the tokens in each tool's top-level `WithDescription(...)` string using `cl100k_base` encoding and confirms each is ≤ 200 tokens
- No tool description exceeds the 200-token budget
- Parameter-level `Description(...)` strings are excluded from this count

---

### Audit Scope and Priority Order

**FR-006:** The audit MUST cover every MCP tool registered in the server. The complete tool surface as of the audit baseline is: `status`, `next`, `finish`, `handoff`, `entity`, `doc`, `health`, `server_info`, `decompose`, `estimate`, `conflict`, `retro`, `knowledge`, `profile`, `worktree`, `merge`, `pr`, `branch`, `cleanup`, `doc_intel`, `incident`, `checkpoint`.

**Acceptance criteria:**
- Every tool listed above has a rewritten description that satisfies FR-001 through FR-005
- No tool in the registered tool surface is skipped
- If new tools are added to the server before the audit is complete, they are added to the audit scope

---

**FR-007:** The audit MUST be executed in the following priority order. All tools in a priority tier MUST be completed before moving to the next tier.

- **Priority 1 — High-frequency tools:** `entity`, `doc`, `handoff`, `next`, `finish`, `status`
- **Priority 2 — Decision-point tools:** `decompose`, `merge`, `pr`
- **Priority 3 — Query and support tools:** `knowledge`, `doc_intel`, `profile`, `estimate`, `conflict`, `retro`, `health`, `worktree`, `branch`, `cleanup`, `incident`, `checkpoint`, `server_info`

**Acceptance criteria:**
- Implementation plan tasks are ordered to match this priority sequence
- Priority 1 tools are rewritten and agent-tested before Priority 2 work begins
- Priority 2 tools are rewritten and agent-tested before Priority 3 work begins

---

### Actionable Error Messages

**FR-008:** Every error message returned by an MCP tool handler (excluding gate-failure errors in `entity(action: "transition")`) MUST follow the three-part template:

1. **What failed** — the action and entity, with IDs where available
2. **Why it failed** — the specific prerequisite, constraint, or validation rule that was violated
3. **What to do next** — a concrete recovery action, ideally formatted as a tool call example the agent can adapt

The template format is:

```
Cannot {action} {entity}: {reason}.

To resolve:
  {recovery_step}
```

**Acceptance criteria:**
- Every `fmt.Errorf(...)` and `mcp.NewToolResultError(...)` return in the tool handlers (excluding gate-failure paths) produces a message containing all three parts: what failed, why, and a recovery action
- Error messages that currently say only "X is required" are rewritten to include the action context and a recovery hint (e.g., `Cannot register document: path is required. Provide the path parameter: doc(action: "register", path: "...", type: "...", title: "...")`)
- Error messages that wrap internal errors (e.g., `fmt.Errorf("resolve path: %w", err)`) are rewritten to lead with the user-facing action context before the internal detail

---

**FR-009:** The error message audit MUST cover the following tool handlers as priority targets, based on design §6.4:

- `finish` — validation failures on missing `task_id`, missing `summary`, wrong task status, auto-transition failures
- `doc` — registration failures (missing `path`, `type`, `title`), approval prerequisites, batch document validation
- `decompose` — input validation, prerequisite checks (e.g., no approved specification)
- `entity(action: "create")` — missing required fields per entity type (e.g., missing `type`, missing `parent_feature` for tasks)

**Acceptance criteria:**
- Every error path in the `finish`, `doc`, `decompose`, and `entity` (create action) handlers is rewritten to follow FR-008
- Each rewritten error message includes a tool-call example in its recovery section that an agent could adapt to resolve the error

---

**FR-010:** Error messages in all remaining tool handlers (those not listed in FR-009) MUST also be audited and rewritten to follow the FR-008 template.

**Acceptance criteria:**
- Every error path in every tool handler across the full tool surface follows the three-part template
- A code review confirms no error return site was missed

---

### Agent-Driven Testing

**FR-011:** A set of representative task scenarios MUST be maintained for agent-driven testing of tool descriptions. The set MUST contain between 5 and 10 scenarios. Each scenario MUST specify:

1. A natural-language task description (what the agent is asked to do)
2. The expected tool-call sequence (which tools, in what order)
3. At least one decision point where the agent must choose between two or more plausible tools

**Acceptance criteria:**
- A scenario file exists (at a documented location in the repository) containing 5–10 scenarios in the specified format
- Each scenario has a task description, an expected tool sequence, and at least one documented decision point
- The scenarios collectively cover all Priority 1 and Priority 2 tools from FR-007

---

**FR-012:** The scenarios MUST cover the following representative workflow patterns at minimum:

1. Advancing a feature through a lifecycle stage (e.g., `specifying` → `dev-planning`)
2. Claiming and completing a task (queue inspection → claim → work → finish)
3. Decomposing a feature into tasks (propose → review → apply)
4. Creating and registering a document
5. Querying project status and finding blocked work

**Acceptance criteria:**
- At least one scenario exists for each of the five workflow patterns listed
- No two scenarios test the identical tool sequence

---

**FR-013:** Agent-driven testing MUST be performed on the rewritten descriptions for each priority tier before that tier is considered complete. The testing process for each tier is:

1. Present an agent with a scenario task and only the MCP tool list (with rewritten descriptions) for guidance
2. Record which tools the agent selects, in what order, and where it selects the wrong tool or gets stuck
3. If the agent selects the wrong tool or fails to find the right tool, rewrite the relevant descriptions to address the observed failure mode
4. Repeat with a fresh agent session (no conversation history from previous attempts) until the agent selects the correct tool sequence without mis-selection

**Acceptance criteria:**
- For each priority tier, there is a documented record of at least one agent-driven test session per scenario covering that tier's tools
- Any description rewrites made in response to observed failures are documented with the failure mode they address
- The final descriptions for each tier pass agent-driven testing: an agent given the scenario selects the correct tools without being guided by conversation history

---

**FR-014:** Agent-driven test results MUST be recorded in a lightweight format that captures: the scenario used, the tool sequence the agent attempted, any wrong-tool selections or failures observed, and the description changes made in response.

**Acceptance criteria:**
- Test results are recorded in a file at a documented repository location
- Each test record includes the scenario ID, the observed tool sequence, a pass/fail indicator, and (for failures) the description change made
- Results are retained for the duration of the feature's implementation (they need not be permanent project artefacts)

---

## Non-Functional Requirements

**NFR-001:** Tool description changes MUST NOT alter any tool's runtime behaviour. The changes are restricted to `mcp.WithDescription(...)`, `mcp.WithTitleAnnotation(...)`, `mcp.Description(...)` (parameter descriptions), and error message strings in `fmt.Errorf(...)` / `mcp.NewToolResultError(...)` calls. No function signatures, parameter schemas, validation logic, or return structures may change.

**Acceptance criteria:**
- All existing tool tests continue to pass without modification (except tests that assert on specific error message text, which may be updated to match new messages)
- A diff of the changes shows modifications only to string literals in description and error-message positions

---

**NFR-002:** Rewritten descriptions MUST be compatible with the MCP protocol's tool description field. They MUST be valid UTF-8 strings with no control characters other than newlines.

**Acceptance criteria:**
- All rewritten descriptions are valid UTF-8
- The MCP server starts and registers all tools without error after the changes

---

**NFR-003:** The audit MUST be incremental — each priority tier can be completed and merged independently. Partial completion of the audit (e.g., Priority 1 done, Priority 2 in progress) MUST leave the system in a fully functional state.

**Acceptance criteria:**
- After completing each priority tier, all tests pass and the server operates normally
- Tools whose descriptions have not yet been rewritten continue to function with their existing descriptions

---

**NFR-004:** Error messages MUST NOT expose internal implementation details (Go package paths, internal function names, raw stack traces) to the agent. The "why it failed" section uses domain-language explanations, not code-level diagnostics.

**Acceptance criteria:**
- No rewritten error message contains Go package paths (e.g., `internal/service/...`), function names not visible to the agent, or stack trace fragments
- Error messages reference domain concepts (entity IDs, status names, document types) rather than implementation details

---

## Acceptance Criteria

The following criteria determine when this feature is complete:

1. **All tools audited:** Every tool listed in FR-006 has a rewritten description satisfying FR-001 through FR-005
2. **Priority order respected:** The implementation followed the priority order in FR-007, with agent-driven testing per tier
3. **All error messages audited:** Every error path in every tool handler follows the three-part template (FR-008), with priority handlers (FR-009) completed first
4. **Token budgets verified:** A verification mechanism confirms every tool description is ≤ 200 tokens (FR-005)
5. **Agent-driven testing complete:** Each priority tier's descriptions have been validated through agent-driven testing with documented results (FR-013, FR-014)
6. **Test scenarios maintained:** 5–10 scenarios exist covering the required workflow patterns (FR-011, FR-012)
7. **No behaviour changes:** All existing tests pass, confirming no tool logic was altered (NFR-001)
8. **Incremental delivery:** Each priority tier was merged independently and left the system functional (NFR-003)

---

## Dependencies and Assumptions

### Dependencies

- **mandatory-stage-gates feature (FEAT-01KN5-88R6ZYPV):** Gate-failure error messages are owned by that feature. This audit explicitly excludes gate-failure paths in `entity(action: "transition")` to avoid conflicting changes. If both features are in progress simultaneously, the boundary is: this feature owns all non-gate error messages; the stage-gates feature owns gate-rejection messages
- **Stable tool surface:** The audit targets the tool surface as registered in `internal/mcp/server.go` and enumerated in `internal/mcp/groups.go`. If new tools are added during the audit, they must be included in the audit scope (FR-006)

### Assumptions

- The current tool surface (22 tools across 7 groups) is the baseline. The tool list in FR-006 is accurate as of specification time
- The `cl100k_base` tokeniser is a reasonable proxy for token counting across the model families that consume these descriptions. An approximation (e.g., word-count heuristic validated against a tokeniser sample) is acceptable for the verification mechanism if an exact tokeniser is not readily available in the test environment
- Agent-driven testing uses the project's own MCP server with real tool descriptions — it does not require a separate test harness or mock MCP server. The testing can be performed by giving an agent access to the kanbanzai MCP tools and observing its behaviour on the scenario tasks
- Error messages that wrap errors from lower layers (e.g., filesystem errors, YAML parse errors) may retain the wrapped error as supplementary detail after the three-part template, but the template's three parts must come first