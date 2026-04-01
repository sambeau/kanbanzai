# Specification: Binding Registry Gate Integration (Kanbanzai 3.0)

**Feature:** FEAT-01KN5-8J27H83N (binding-registry-gate-integration)
**Design reference:** `work/design/kanbanzai-3.0-workflow-and-tooling-v2.md` §3.4 (override policy), §3.5 (binding registry as gate source), §16.1 Q6 (override audit resolution), §16.3 Q5 (read frequency)
**Design principle:** WP-5 — The Binding Registry Is the Decision Table
**Status:** Draft

---

## Overview

The MCP server's stage gate enforcement currently uses hardcoded prerequisite definitions in `internal/service/prereq.go`. This feature replaces the hardcoded gate source with a registry-driven approach: the transition handler reads gate prerequisites from the binding registry's `prerequisites` block per stage, evaluates them against the feature's current state, and sources the per-gate `override_policy` (agent or checkpoint) from the same registry. The binding registry file is read at startup and cached in memory with file-mtime-based invalidation so that edits take effect on the next tool call without per-call I/O cost. When the binding registry is absent or malformed, the system falls back to the existing hardcoded gate definitions, ensuring continuity during migration. This enables the skills design to evolve prerequisites — adding, removing, or modifying gates — without MCP server code changes.

---

## Scope

### In scope

- Reading gate prerequisite definitions from the binding registry (`.kbz/stage-bindings.yaml`) at transition time
- A general-purpose prerequisite evaluator that interprets the registry's `prerequisites` structure (documents, tasks) and can be extended for future prerequisite types
- File-mtime-based cache invalidation: read and parse at startup, stat on each consulting tool call, re-read only on mtime change
- Sourcing `override_policy` per stage from the binding registry's `prerequisites` block
- Implementing the `checkpoint` override policy: when a gate with `checkpoint` policy is overridden, a human checkpoint is created and the feature blocks until the human responds
- Fallback to hardcoded gate definitions when the binding registry file is absent, unparseable, or missing a `prerequisites` block for the stage being evaluated
- Coexistence: the hardcoded gate table remains as the fallback source; registry-defined gates take precedence when present
- Logging and health flagging for which gate source (registry or hardcoded) was used

### Explicitly excluded

- Defining the binding registry schema, file format, or loader (that is FEAT-01KN5-88PDPE8V, binding-registry)
- Defining or modifying the binding registry content — stage entries, roles, skills, templates (that is P16, skills redesign)
- The hardcoded gate enforcement logic itself — mandatory gates on all transitions, override mechanism, actionable error messages (that is FEAT-01KN5-8J24S2XW, mandatory-stage-gates)
- Context assembly reading from the binding registry (that is the stage-aware-context-assembly feature)
- Action pattern logging for override frequency analysis (that is the observability feature, §12)
- Promoting specific gates from `agent` to `checkpoint` policy (that is a content decision driven by observed override patterns)
- Plan lifecycle gate enforcement (plans have a separate lifecycle not governed by these gates)

---

## Functional Requirements

### Registry-driven prerequisite lookup

**FR-001:** The transition handler MUST attempt to read gate prerequisites from the binding registry before falling back to hardcoded definitions. For each feature lifecycle transition, the handler MUST look up the target stage in the binding registry's `stage_bindings` map and read its `prerequisites` block. If the registry provides prerequisites for the stage, those MUST be used instead of the hardcoded gate definition.

**Acceptance criteria:**
- A binding registry entry with `prerequisites: {documents: [{type: specification, status: approved}]}` for stage `dev-planning` causes the transition handler to require an approved specification document, matching the hardcoded behaviour but sourced from the registry
- Removing the `prerequisites` block from a stage's binding entry causes the transition to that stage to have no gate (treated as having no prerequisites)
- Adding a new prerequisite type to a stage's binding entry (e.g., a second document requirement) is enforced on the next transition without code changes

---

**FR-002:** The prerequisite evaluator MUST support the `documents` prerequisite type. A `documents` prerequisite is a list of objects, each containing `type` (string) and `status` (string). For each document prerequisite, the evaluator MUST check that at least one document of the specified type with the specified status exists for the feature, using the same three-level lookup order as the current hardcoded gates: (1) the feature's own document field reference, (2) documents owned by the feature, (3) documents owned by the parent plan.

**Acceptance criteria:**
- `prerequisites: {documents: [{type: design, status: approved}]}` is satisfied when an approved design document exists for the feature or its parent plan
- `prerequisites: {documents: [{type: design, status: approved}]}` is not satisfied when only a draft design document exists
- `prerequisites: {documents: [{type: specification, status: approved}, {type: design, status: approved}]}` requires both an approved specification and an approved design document — failing if either is missing
- A document prerequisite with an unknown document type (e.g., `type: security-review`) is evaluated against the document service without error — it simply requires a document of that type to exist with the given status

---

**FR-003:** The prerequisite evaluator MUST support the `tasks` prerequisite type. A `tasks` prerequisite is an object containing either `min_count` (integer) or `all_terminal` (boolean), but not both (per the binding registry schema). When `min_count` is specified, the evaluator MUST check that the feature has at least that many child tasks. When `all_terminal` is specified and `true`, the evaluator MUST check that all child tasks are in a terminal state (`done`, `not-planned`, or `duplicate`).

**Acceptance criteria:**
- `prerequisites: {tasks: {min_count: 1}}` is satisfied when the feature has at least one child task
- `prerequisites: {tasks: {min_count: 1}}` is not satisfied when the feature has no child tasks
- `prerequisites: {tasks: {min_count: 3}}` is not satisfied when the feature has only two child tasks
- `prerequisites: {tasks: {all_terminal: true}}` is satisfied when all child tasks are in `done`, `not-planned`, or `duplicate` status
- `prerequisites: {tasks: {all_terminal: true}}` is not satisfied when any child task is in a non-terminal state (e.g., `active`, `ready`)

---

**FR-004:** The prerequisite evaluator MUST be designed to accommodate future prerequisite types without structural changes to the evaluator framework. Each prerequisite type (documents, tasks, and any future type) MUST be handled by a distinct evaluation function dispatched by type key. An unrecognised prerequisite type key in the registry MUST cause the gate to fail with an error identifying the unknown type, rather than being silently ignored.

**Acceptance criteria:**
- A `prerequisites` block containing an unknown key (e.g., `custom_check: {threshold: 5}`) causes the gate evaluation to fail with an error message that names the unrecognised prerequisite type
- Adding a new prerequisite evaluator function for a new type key requires no changes to the dispatch logic beyond registering the new function
- The error message for an unknown prerequisite type includes the stage name and the unrecognised key

---

### Cache and hot-reload

**FR-005:** The MCP server MUST read and parse the binding registry file at startup, caching the parsed result and the file's mtime in memory. If the binding registry file does not exist at startup, the server MUST start successfully with an empty cache (falling back to hardcoded gates).

**Acceptance criteria:**
- A server started with a valid `.kbz/stage-bindings.yaml` caches the parsed bindings in memory
- A server started without a `.kbz/stage-bindings.yaml` file starts successfully and uses hardcoded gates for all transitions
- A server started with a malformed `.kbz/stage-bindings.yaml` file starts successfully, logs a warning identifying the parse error, and uses hardcoded gates

---

**FR-006:** On each tool call that consults the binding registry (gate checks during `entity(action: "transition")`), the server MUST stat the binding registry file's mtime and compare it to the cached mtime. If the mtime has changed, the server MUST re-read and re-parse the file before evaluating the gate. If the mtime is unchanged, the server MUST use the cached result without file I/O beyond the stat call.

**Acceptance criteria:**
- Editing `.kbz/stage-bindings.yaml` between two transition calls causes the second call to use the updated prerequisites
- Two consecutive transition calls with no file change result in only one file read (at startup or first access) — the second call uses the cached result
- If the file is deleted after startup, the next transition call detects the missing file, clears the cache, and falls back to hardcoded gates
- If the file is replaced with a malformed version after startup, the next transition call detects the change, fails to parse, logs a warning, and falls back to hardcoded gates (the previously cached valid version is NOT used — a broken file triggers fallback, not stale cache)

---

**FR-007:** The cache MUST be safe for concurrent access. Multiple simultaneous tool calls that trigger a cache refresh MUST NOT cause data races or partial reads.

**Acceptance criteria:**
- Concurrent transition calls during a cache refresh do not panic or produce inconsistent gate evaluations
- The Go race detector (`-race`) reports no data races during concurrent gate evaluation with cache invalidation

---

### Fallback to hardcoded gates

**FR-008:** When the binding registry is absent, unparseable, or does not contain a `prerequisites` block for the stage being evaluated, the transition handler MUST fall back to the hardcoded gate definitions currently in `internal/service/prereq.go`. The fallback MUST produce identical gate evaluation results as the current hardcoded implementation for all transitions in the existing gate prerequisite table (§3.3 of the design).

**Acceptance criteria:**
- With no binding registry file present, every gated transition in the prerequisite table behaves identically to the current hardcoded implementation
- With a binding registry that defines prerequisites for `designing` but not for `specifying`, the `designing` gate uses registry prerequisites and the `specifying` gate uses the hardcoded definition
- The fallback path produces the same `GateResult` structure (stage, satisfied, reason) as the hardcoded path

---

**FR-009:** When a fallback to hardcoded gates occurs, the gate evaluation result MUST indicate which source was used (registry or hardcoded). This source indicator MUST be included in the `GateResult` returned to the caller.

**Acceptance criteria:**
- A gate evaluated from the registry includes `source: "registry"` (or equivalent) in its result
- A gate evaluated from hardcoded definitions includes `source: "hardcoded"` (or equivalent) in its result
- The `health` tool can report which gates are still using hardcoded definitions as informational items, enabling operators to track migration progress

---

### Override policy from the binding registry

**FR-010:** The transition handler MUST read the `override_policy` field from the stage's `prerequisites` block in the binding registry. The `override_policy` field is a string enum with two valid values: `agent` and `checkpoint`. When the field is absent from the registry or when using hardcoded fallback gates, the override policy MUST default to `agent`.

**Acceptance criteria:**
- A stage with `override_policy: agent` in its prerequisites block uses the agent override behaviour
- A stage with `override_policy: checkpoint` in its prerequisites block uses the checkpoint override behaviour
- A stage with no `override_policy` field in its prerequisites block defaults to `agent`
- A stage evaluated via hardcoded fallback defaults to `agent` override policy
- An `override_policy` value that is not `agent` or `checkpoint` causes a validation error at parse time (not at transition time)

---

**FR-011:** When a gate with `override_policy: agent` is overridden (via `override: true` with a reason), the transition MUST proceed immediately. The override MUST be logged on the feature entity and flagged by the `health` tool, consistent with the existing override mechanism defined in the mandatory-stage-gates specification.

**Acceptance criteria:**
- Overriding a gate with `agent` policy proceeds without blocking
- The override is recorded on the feature entity with the from-status, to-status, reason, and timestamp
- The `health` tool flags the override as a warning

---

**FR-012:** When a gate with `override_policy: checkpoint` is overridden (via `override: true` with a reason), the transition handler MUST create a human checkpoint using the existing checkpoint system. The checkpoint question MUST identify the feature, the transition being attempted, the gate that failed, and the override reason provided by the agent. The feature MUST NOT transition until the checkpoint is responded to.

**Acceptance criteria:**
- Overriding a gate with `checkpoint` policy creates a checkpoint record with status `pending`
- The checkpoint question contains the feature ID, the from-status and to-status, the failing prerequisite description, and the agent's override reason
- The transition call returns a response indicating a checkpoint was created, including the checkpoint ID and a message instructing the agent to poll the checkpoint
- The feature's status remains unchanged until the checkpoint is responded to
- The override is logged on the feature entity, including the checkpoint ID

---

**FR-013:** When a checkpoint created by a `checkpoint` override is responded to with approval, the feature MUST be transitioned past the gate. When responded to with rejection, the feature MUST remain at its current status and the override MUST be recorded as rejected.

**Acceptance criteria:**
- A checkpoint responded to with an approving response (e.g., "approved", "yes", or any non-rejection) results in the transition completing — the feature moves to the target status
- A checkpoint responded to with a rejecting response (e.g., "rejected", "no", "denied") results in the feature remaining at its current status
- After approval, the override record on the feature includes the checkpoint ID and the human's response
- After rejection, the override record on the feature includes the checkpoint ID, the human's response, and an indication that the override was rejected
- The determination of approval vs. rejection uses a simple keyword match: responses containing "reject", "denied", or "no" (case-insensitive, as a whole word) are rejections; all other non-empty responses are approvals

---

**FR-014:** When `advance: true` is combined with `override: true` and the advance path crosses a gate with `checkpoint` override policy, the advance MUST halt at that gate and create the checkpoint. The advance MUST NOT continue past the checkpoint gate, even if subsequent gates have `agent` policy. Gates with `agent` policy earlier in the path MUST be overridden normally.

**Acceptance criteria:**
- An advance through three gates where the first two have `agent` policy and the third has `checkpoint` policy: the first two are overridden immediately, the third creates a checkpoint, and the advance halts
- The response indicates which gates were overridden via `agent` policy and which gate created a checkpoint
- After the checkpoint is resolved, the agent must issue a new advance call to continue — the original advance does not resume automatically

---

### Integration with existing transition handler

**FR-015:** The registry-driven gate evaluation MUST be invoked at the same point in the transition handler where the hardcoded `CheckFeatureGate` function is currently called. Both single-step transitions and `advance` mode transitions MUST use the same registry-driven evaluation path.

**Acceptance criteria:**
- A single-step `entity(action: "transition", status: "dev-planning")` evaluates the gate from the registry (or hardcoded fallback) before allowing the transition
- An `advance: true` call evaluates each intermediate gate from the registry (or hardcoded fallback) as it walks through states
- The evaluation point is the same for both paths — no separate gate-check logic exists for single-step vs. advance

---

**FR-016:** Gate failure error messages MUST remain actionable regardless of whether the gate source is the registry or the hardcoded fallback. Error messages MUST follow the template from the design (§3.6): what failed, why it failed, and what to do next. The error message MUST NOT expose whether the gate was sourced from the registry or from hardcoded logic — this is an internal implementation detail.

**Acceptance criteria:**
- A registry-sourced gate failure for a missing specification produces an error message containing the feature ID, the missing document type, and a recovery tool call example
- A hardcoded-fallback gate failure for the same condition produces an equivalent error message
- No gate failure message contains the words "binding registry", "hardcoded", or "fallback"

---

## Non-Functional Requirements

**NFR-001:** The mtime stat call MUST add no more than 1ms of latency to a transition call. The stat operation reads only file metadata, not file contents, and MUST NOT degrade gate evaluation performance.

**Acceptance criteria:**
- Benchmark tests confirm that the mtime stat adds negligible overhead compared to gate evaluation without the stat

---

**NFR-002:** Re-parsing the binding registry after an mtime change MUST complete within 200ms for a file containing up to 15 stage bindings (consistent with the binding registry spec NFR-001). During re-parse, concurrent transition calls MUST either use the previous cached version or wait for the re-parse to complete — they MUST NOT use partially parsed data.

**Acceptance criteria:**
- Benchmark tests with a 15-binding file confirm re-parse completes within 200ms
- Concurrent transition calls during re-parse produce consistent, valid gate evaluations

---

**NFR-003:** The registry-driven gate evaluation MUST maintain backward compatibility with the existing gate evaluation interface. The `GateResult` struct MUST remain the return type. The `CheckFeatureGate` function signature may change to accept the registry cache, but callers that currently consume `GateResult` MUST NOT require changes beyond passing the new dependency.

**Acceptance criteria:**
- Existing tests for `CheckFeatureGate` continue to pass after the change (with appropriate setup for the registry cache or fallback)
- The merge gate checker (`internal/merge/checker.go`) continues to function without changes to its gate evaluation calls

---

**NFR-004:** The fallback to hardcoded gates MUST be invisible to agents. An agent MUST NOT need to know or care whether gates are sourced from the registry or hardcoded. The tool interface (`entity(action: "transition")`) MUST NOT change.

**Acceptance criteria:**
- No new parameters are added to the `entity` tool for this feature
- An agent that previously used `entity(action: "transition")` with `override: true` and `override_reason` continues to work identically

---

**NFR-005:** The `checkpoint` override policy MUST integrate with the existing checkpoint store (`internal/checkpoint/`) and checkpoint tool. No new checkpoint infrastructure is required — the feature reuses the existing `Store.Create`, `Store.Get`, and `Store.Respond` methods.

**Acceptance criteria:**
- Checkpoints created by `checkpoint` policy overrides appear in `checkpoint(action: "list")` output
- Checkpoints created by `checkpoint` policy overrides can be responded to via `checkpoint(action: "respond")`
- No new checkpoint-related tools or tool parameters are introduced

---

## Acceptance Criteria

The following are aggregate acceptance criteria for the specification as a whole:

1. **Registry-sourced gates work:** A transition that requires an approved specification (per the binding registry) is blocked when no approved specification exists, and proceeds when one does — with the gate definition coming from the registry file, not from hardcoded logic.

2. **Hot-reload works:** Editing `.kbz/stage-bindings.yaml` to add a new document prerequisite to a stage causes the next transition to that stage to enforce the new prerequisite, without restarting the server.

3. **Fallback is seamless:** Deleting the binding registry file does not break any transitions — all gates fall back to hardcoded definitions and produce identical results.

4. **Agent override policy works:** A gate with `agent` policy can be overridden immediately. A gate with `checkpoint` policy creates a checkpoint and blocks until a human responds.

5. **Checkpoint integration works:** A checkpoint created by a `checkpoint` override appears in checkpoint listings, can be responded to, and the response determines whether the transition proceeds or is rejected.

6. **Advance mode respects policies:** An advance through multiple gates correctly handles mixed `agent` and `checkpoint` policies, halting at the first `checkpoint` gate that requires override.

7. **Extensibility is demonstrated:** Adding a new prerequisite type key to a stage's binding and registering a corresponding evaluator function results in the new prerequisite being enforced — without changes to the dispatch or transition handler logic.

8. **No regression:** All existing tests for `CheckFeatureGate`, `AdvanceFeatureStatus`, and the `entity(action: "transition")` tool continue to pass.

---

## Dependencies and Assumptions

1. **Binding registry loader (FEAT-01KN5-88PDPE8V):** This feature depends on the binding registry loader being implemented. Specifically, it requires: the `stage-bindings.yaml` file format, the stage lookup function (FR-008 of the binding registry spec), and the parsed `prerequisites` structure (FR-009 of the binding registry spec) including the `override_policy` field. If the binding registry feature is not yet complete, this feature can be developed against the schema definition with hardcoded fallback as the primary path.

2. **Mandatory stage gates (FEAT-01KN5-8J24S2XW):** This feature depends on the mandatory-stage-gates feature being implemented. Specifically, it requires: gate enforcement on all transitions (not just advance), the override mechanism (`override` and `override_reason` parameters), override logging on the feature entity, and health flagging of overrides. This feature replaces the gate *source* but does not re-implement the gate *enforcement* mechanism.

3. **Checkpoint system:** The existing checkpoint store (`internal/checkpoint/`) from Phase 4a is available and provides `Create`, `Get`, and `Respond` methods. The checkpoint MCP tool is registered and functional.

4. **Prerequisites schema:** The `prerequisites` block in the binding registry supports `documents` (list of `{type, status}` objects), `tasks` (`{min_count}` or `{all_terminal}`), and `override_policy` (string enum: `agent`, `checkpoint`). This schema is defined by the binding registry specification (FR-009) and the design document (§3.4, §16.1 Q6).

5. **Terminal task states:** Terminal task states are `done`, `not-planned`, and `duplicate`, as defined in `internal/validate/lifecycle.go`. The `all_terminal` task prerequisite uses this canonical definition.

6. **Document service:** The existing `DocumentService` with `ListDocuments` and `GetDocument` methods is available for document prerequisite evaluation. The three-level lookup order (feature field reference, feature-owned documents, parent-plan-owned documents) is preserved from the current implementation.

7. **Single file assumption:** The binding registry is a single file (`.kbz/stage-bindings.yaml`). The mtime-based cache invalidation monitors this single file. If the binding registry evolves to a directory of files, the invalidation mechanism will need to be extended (per §16.3 Q5 of the design).

8. **Default override policy:** For 3.0, all gates default to `agent` override policy. The binding registry content that ships with the initial skills redesign will set `override_policy: agent` on all stages. Promotion to `checkpoint` is a content decision driven by observed override patterns via action pattern logging — it is not part of this feature's implementation.