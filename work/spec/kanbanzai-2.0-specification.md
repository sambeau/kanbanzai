# Kanbanzai 2.0 Specification: MCP Tool Surface Redesign

| Document | Kanbanzai 2.0 Specification |
|----------|------------------------------|
| Status   | Draft                        |
| Created  | 2026-03-27T01:42:41Z         |
| Updated  | 2026-03-27T01:42:41Z         |
| Related  | `work/design/kanbanzai-2.0-design-vision.md`          |
|          | `work/reports/mcp-server-design-issues.md`             |
|          | `work/spec/phase-4b-specification.md`                  |
|          | `work/spec/phase-4a-specification.md`                  |
|          | `work/design/workflow-design-basis.md`                 |

---

## 1. Purpose

This specification defines the requirements for Kanbanzai 2.0: a redesign of the MCP tool surface from 97 entity-centric tools to a small set of workflow-oriented tools organised by feature groups.

Kanbanzai 2.0 is not a new system. The internal architecture — YAML-on-disk storage, lifecycle state machines, document-driven stage gates, knowledge confidence scoring, Git-native worktree isolation, and the service/storage layers — is validated by Phases 1–4b and carries forward unchanged. What changes is how this architecture is exposed to agents.

The 1.0 tool surface was designed from the system's perspective: "what operations exist on what entity types?" This produced 97 tools that consumed thousands of context-window tokens, required many round-trips for common workflows, and was routinely bypassed by agents who found grep and direct YAML editing faster.

The 2.0 tool surface is designed from the orchestrator's perspective: "what does an agent need to do during a work session?" It replaces 97 tools with 7 core tools (20 maximum with all feature groups enabled), adds batch operations, surfaces side effects, and makes document intelligence and knowledge management invisible infrastructure that powers the workflow tools from inside.

The measure of success: agents prefer Kanbanzai tools over grep and direct YAML editing because the tools are genuinely faster and more useful.

---

## 2. Goals

1. **Reduce tool count from 97 to 20.** Seven core tools always loaded; 13 additional tools available through optional feature groups. Every tool schema that occupies the context window earns its space by being used.

2. **Reduce round-trips per workflow.** A complete claim-delegate-complete cycle requires 4–5 Kanbanzai calls, down from 8–15 in 1.0.

3. **Surface side effects.** Every mutation that triggers a cascade (document approval → feature transition, task completion → dependency unblocking) reports what it changed in the response.

4. **Batch by default.** Every tool that accepts an entity ID also accepts an array. Single operations are arrays of length 1.

5. **Make intelligence invisible.** Document intelligence and knowledge management power `next`, `handoff`, and `status` automatically. Agents never call `doc_classify` or `knowledge_contribute` directly in the normal workflow.

6. **Eliminate worker tool calls.** Worker agents receive complete context at spawn time via `handoff`. A worker that needs to call Kanbanzai tools indicates an incomplete handoff, not a missing tool.

7. **Respect scale.** The same tools work for 13-spec and 149-spec projects. At small scale, intelligence is transparent. At large scale, it is essential. Feature groups prevent small projects from paying the context-window cost of capabilities they don't use.

---

## 3. Scope

### 3.1 In scope for Kanbanzai 2.0

- **Feature group framework:** Configuration model, conditional tool registration, presets
- **Resource-oriented tool pattern:** Action-parameter dispatch for consolidated tools
- **Side-effect reporting:** Structured `side_effects` field on all mutation responses
- **Batch operations:** Array-accepting inputs with partial-failure semantics
- **Core tools:** `status`, `next`, `finish`, `handoff`, `entity`, `doc`, `health`
- **Feature group tools:** `decompose`, `estimate`, `conflict`, `knowledge`, `profile`, `worktree`, `merge`, `pr`, `branch`, `cleanup`, `doc_intel`, `incident`, `checkpoint`
- **Invisible infrastructure wiring:** Document intelligence and knowledge retrieval integrated into `next`, `handoff`, and `status`
- **1.0 tool removal:** Deprecation and removal of the 97-tool surface
- **CLI alignment:** CLI commands updated to match the resource-oriented pattern

### 3.2 Deferred beyond 2.0

- MCP Resources for a human viewer (separate design document)
- MCP Prompts for user-initiated templates
- Elicitation-based batch approval flows (pending client support investigation)
- Sampling-based document classification (pending complexity assessment)
- Remote MCP server / Streamable HTTP transport
- Cross-project knowledge sharing
- Semantic search or embedding-based retrieval

### 3.3 Explicitly excluded

- Changes to the internal storage model (YAML on disk, entity schemas, lifecycle state machines)
- Changes to the document-driven stage gate model
- Changes to knowledge confidence scoring or TTL rules
- GitLab, Bitbucket, or other platform support
- Web or desktop UI

---

## 4. Design Principles

### 4.1 Workflow tools, not entity tools

Tools map to what an agent wants to *do*, not what entities exist in the system. "I'm done with this task" is one thought and one tool call — not "update status on entity type task with ID X to status done."

### 4.2 Batch by default

Every tool that accepts an entity ID also accepts an array of entity IDs. Single operations are arrays of length 1. This eliminates the N-round-trip problem for orchestrators working on sets of entities.

### 4.3 Side effects are first-class responses

Every mutation that triggers a cascade must surface what it changed. If the system advanced a feature because a spec was approved, the response says so. Rule: if the system changed something you didn't explicitly ask it to change, the response tells you.

### 4.4 Invisible infrastructure, not exposed tools

Document intelligence and knowledge management are infrastructure. They power the workflow tools from inside. An agent should never need to call `doc_classify` or `knowledge_contribute` directly in the normal workflow.

### 4.5 Context arrives, not gets fetched

Worker agents receive their context at spawn time. The orchestrator's `handoff` tool assembles everything the worker needs. If a worker agent needs to call Kanbanzai tools, the handoff was incomplete.

### 4.6 Respect the context budget

Every tool schema that sits in the context window must earn its space by being used. Fewer tools with richer responses. Feature groups to load only what's needed.

### 4.7 The internals are sound

The YAML-on-disk model, lifecycle state machines, document-driven stage gates, knowledge confidence scoring, and Git-native storage are validated by 1.0 usage. This is a tool surface redesign, not a ground-up rewrite.

---

## 5. Approved Design Decisions

Design decisions for 2.0 are recorded in `work/plan/kanbanzai-2.0-decision-log.md`.

Kanbanzai 2.0 also inherits all accepted decisions from Phases 1–4b that pertain to the internal model. The tool surface changes; the internal model does not.

---

## 6. Feature Group Framework

### 6.1 Purpose

Feature groups control which tools the MCP server registers with the client. Disabled groups consume zero context-window space. This replaces the 1.0 model where all 97 tools were always registered.

### 6.2 Configuration

Feature groups are configured in `.kbz/config.yaml` under a new `mcp` section:

```yaml
mcp:
  groups:
    core: true          # always enabled, cannot be disabled
    planning: false
    knowledge: false
    git: false
    documents: false
    incidents: false
    checkpoints: false
```

Setting a group to `true` enables all tools in that group. The `core` group is always enabled regardless of configuration and cannot be set to `false`.

### 6.3 Presets

A `preset` key provides shorthand for common configurations:

```yaml
mcp:
  preset: orchestration
```

| Preset | Groups enabled | Tool count |
|--------|---------------|------------|
| `minimal` | core | 7 |
| `orchestration` | core, planning, git | 15 |
| `full` | all groups | 20 |

When both `preset` and explicit `groups` are specified, the explicit `groups` override the preset. This allows `preset: orchestration` with `checkpoints: true` to produce orchestration + checkpoints.

### 6.4 Group membership

| Group | Tools |
|-------|-------|
| **core** | `status`, `next`, `finish`, `handoff`, `entity`, `doc`, `health` |
| **planning** | `decompose`, `estimate`, `conflict` |
| **knowledge** | `knowledge`, `profile` |
| **git** | `worktree`, `merge`, `pr`, `branch`, `cleanup` |
| **documents** | `doc_intel` |
| **incidents** | `incident` |
| **checkpoints** | `checkpoint` |

### 6.5 Tool registration behaviour

On server startup, the MCP server reads `.kbz/config.yaml`, resolves the effective group configuration (preset + overrides), and registers only the tools from enabled groups. The `tools/list` MCP response contains only the enabled tools.

If `.kbz/config.yaml` does not contain an `mcp` section, the default is `preset: full` (all groups enabled, all 20 tools registered). This provides backward-compatible behaviour for projects that have not yet configured groups.

### 6.6 Validation

- Unrecognised group names in the `groups` map produce a startup warning (not an error) and are ignored.
- Unrecognised preset names produce a startup error.
- Setting `core: false` produces a startup warning and is silently overridden to `true`.

---

## 7. Resource-Oriented Tool Pattern

### 7.1 Purpose

The resource-oriented pattern replaces per-operation-per-entity-type tools with a single tool per domain that accepts an `action` parameter. This is the mechanism by which 97 tools become 20.

### 7.2 Action dispatch

Every consolidated tool (entity, doc, knowledge, worktree, merge, pr, branch, cleanup, incident, checkpoint, decompose, estimate, doc_intel) accepts an `action` parameter as its first required field. The action determines which operation is performed and which additional parameters are required.

```
entity(action: "create", type: "task", ...)
entity(action: "list", type: "feature", ...)
doc(action: "approve", id: "DOC-...", ...)
```

### 7.3 Parameter validation

Each action within a tool has its own set of required and optional parameters. Parameters that are required for one action but irrelevant to another are ignored (not rejected) when the irrelevant action is called. This avoids forcing agents to know the full parameter set of every action.

Example: `entity(action: "create", type: "task", parent_feature: "FEAT-...", slug: "my-task", summary: "...")` — the `status` parameter is irrelevant to `create` and is ignored if provided.

Invalid or unrecognised `action` values return an error listing valid actions for that tool.

### 7.4 Error response shape

All tools use a consistent error response structure:

```yaml
error:
  code: "invalid_action"          # machine-readable error code
  message: "Unknown action 'foo'. Valid actions: create, get, list, update"
  details: {}                     # optional additional context
```

Error codes are stable identifiers that agents can match on. Messages are human-readable and may change between versions.

---

## 8. Side-Effect Reporting

### 8.1 Purpose

In 1.0, the system performed correct cascade transitions (e.g., document approval advancing a feature's lifecycle) but did not report them. The caller had no way to know what changed without re-querying. In 2.0, every mutation response includes a structured `side_effects` field.

### 8.2 Response contract

Every mutation (any tool call that changes state) includes a `side_effects` field in its response:

```yaml
side_effects:
  - type: "status_transition"
    entity_id: "FEAT-01JX..."
    entity_type: "feature"
    from_status: "specifying"
    to_status: "dev-planning"
    trigger: "Specification document DOC-01JX... approved"
  - type: "task_unblocked"
    entity_id: "TASK-01JY..."
    entity_type: "task"
    from_status: "blocked"
    to_status: "ready"
    trigger: "All dependencies of TASK-01JY... are now in terminal state"
```

### 8.3 Side-effect types

| Type | Trigger | Description |
|------|---------|-------------|
| `status_transition` | Document approval, task completion | An entity's lifecycle status changed as a cascade |
| `task_unblocked` | Dependency completion | A blocked task became ready |
| `knowledge_contributed` | `finish` with inline knowledge | A knowledge entry was created |
| `knowledge_rejected` | `finish` with duplicate knowledge | A knowledge entry was rejected (duplicate) |
| `worktree_created` | Task dispatch, feature activation | A worktree was automatically created |

### 8.4 Empty side effects

When a mutation produces no side effects, the field is present as an empty array:

```yaml
side_effects: []
```

The field is never omitted. This allows callers to unconditionally read `side_effects` without null-checking.

### 8.5 Read-only operations

Read-only operations (`status`, `entity(action: get)`, `entity(action: list)`, etc.) do not include a `side_effects` field. The `work_queue` mode of `next` (queue inspection without claiming) is an exception: it performs write-through promotion (`queued` → `ready`) and therefore includes `side_effects`.

---

## 9. Batch Operations

### 9.1 Purpose

Orchestrators routinely operate on sets of entities. In 1.0, each operation was a separate round-trip. In 2.0, every tool that accepts a single entity identifier also accepts an array.

### 9.2 Input semantics

Batch-capable parameters accept either a single value or an array:

```
# Single operation
finish(task_id: "TASK-01JX...", summary: "Done")

# Batch operation
finish(tasks: [{task_id: "TASK-01JX...", summary: "Done"}, {task_id: "TASK-01JY...", summary: "Done"}])
```

For tools where the single and batch parameter names differ, both are documented in the tool schema. The tool detects which form is provided.

### 9.3 Execution semantics

Batch operations use **best-effort execution**. Each item in the batch is processed independently. A failure on one item does not prevent processing of subsequent items.

### 9.4 Response shape

Batch responses include per-item results and a summary:

```yaml
results:
  - item_id: "TASK-01JX..."
    status: "ok"
    data: { ... }         # item-specific response data
    side_effects: [...]
  - item_id: "TASK-01JY..."
    status: "error"
    error:
      code: "invalid_status"
      message: "Task TASK-01JY... is in status 'done', expected 'active'"
summary:
  total: 2
  succeeded: 1
  failed: 1
side_effects: [...]       # aggregate of all side effects from all succeeded items
```

When all items succeed, the top-level `side_effects` is the union of all per-item side effects. When called with a single item (not using the batch parameter), the response shape is the single-item form — not a batch response with one result.

### 9.5 Batch limits

Batch operations accept a maximum of 100 items per call. Requests exceeding this limit are rejected with error code `batch_limit_exceeded` before any processing occurs.

---

## 10. Core Tool: `status`

### 10.1 Purpose

The situational awareness tool. Returns a synthesised view of project, plan, feature, or task state. Replaces the pattern of calling `list_entities` + `get_entity` + `doc_record_list` + `health_check` separately.

### 10.2 Input parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | string | no | Plan ID, Feature ID, or Task ID. If omitted, returns project overview. |

The entity type is inferred from the ID format (plan prefix+number, FEAT-, TASK-, BUG-).

### 10.3 Project overview (no `id`)

Returns a synthesis of the entire project:

```yaml
project:
  plans:
    total: 3
    by_status: {active: 2, done: 1}
    active_plans:
      - id: "P1-basic-ui"
        title: "Basic UI"
        feature_count: 5
        task_progress: "12/18 done"
        blockers: 1
      - id: "P2-billing"
        title: "Billing Integration"
        feature_count: 3
        task_progress: "0/9 done"
        blockers: 0
  health:
    errors: 0
    warnings: 2
    categories:
      - name: "stalled_dispatches"
        severity: "warning"
        count: 1
      - name: "estimation_coverage"
        severity: "warning"
        count: 1
  attention:
    - "2 tasks stalled for >3 days"
    - "Feature FEAT-01JX... has no estimated tasks"
  active_tasks: 4
  ready_tasks: 3
  blocked_tasks: 2
```

The `attention` field contains human-readable strings highlighting what needs action. It is a curated summary, not an exhaustive list.

### 10.4 Plan dashboard (`id` = plan ID)

Returns a synthesis of a single plan:

```yaml
plan:
  id: "P1-basic-ui"
  title: "Basic UI"
  status: "active"
  features:
    total: 5
    by_status: {developing: 3, specifying: 1, done: 1}
    items:
      - id: "FEAT-01JA..."
        slug: "user-auth"
        status: "developing"
        tasks: {total: 6, done: 4, active: 1, ready: 1, blocked: 0}
        estimate: {total: 21, progress: 14}
        blockers: 0
      # ... remaining features
  documents:
    specs: {total: 4, approved: 3, draft: 1}
    designs: {total: 2, approved: 2}
    dev_plans: {total: 3, approved: 2, draft: 1}
    gaps:
      - feature_id: "FEAT-01JC..."
        missing: ["dev-plan"]
  health:
    errors: 0
    warnings: 1
  attention:
    - "Feature FEAT-01JC... is missing a dev-plan"
    - "3 tasks ready for dispatch"
```

### 10.5 Feature detail (`id` = feature ID)

Returns a synthesis of a single feature:

```yaml
feature:
  id: "FEAT-01JA..."
  slug: "user-auth"
  status: "developing"
  parent_plan: "P1-basic-ui"
  summary: "User authentication with JWT"
  tasks:
    total: 6
    by_status: {done: 4, active: 1, ready: 1}
    items:
      - id: "TASK-01JX..."
        slug: "jwt-middleware"
        status: "done"
        estimate: 5
      - id: "TASK-01JY..."
        slug: "login-endpoint"
        status: "active"
        estimate: 3
        dispatched_to: "backend"
        dispatched_at: "2026-03-27T10:00:00Z"
      # ... remaining tasks
  documents:
    - id: "DOC-01JA..."
      type: "specification"
      status: "approved"
      path: "work/spec/user-auth-spec.md"
    - id: "DOC-01JB..."
      type: "dev-plan"
      status: "draft"
      path: "work/plan/user-auth-plan.md"
  estimate: {total: 21, progress: 14, delta: -1}
  worktree:
    status: "active"
    branch: "feat/user-auth"
    path: ".kbz/worktrees/feat-user-auth"
  attention:
    - "1 task ready for dispatch"
    - "Dev-plan is still in draft"
```

### 10.6 Task detail (`id` = task ID)

Returns a synthesis of a single task with parent context:

```yaml
task:
  id: "TASK-01JY..."
  slug: "login-endpoint"
  status: "active"
  summary: "Implement login endpoint with JWT token generation"
  estimate: 3
  parent_feature:
    id: "FEAT-01JA..."
    slug: "user-auth"
    status: "developing"
    plan_id: "P1-basic-ui"
  dependencies:
    total: 2
    blocking: 0
    items:
      - task_id: "TASK-01JW..."
        slug: "jwt-middleware"
        status: "done"
        blocking: false
      - task_id: "TASK-01JV..."
        slug: "user-model"
        status: "done"
        blocking: false
  dispatch:
    dispatched_to: "backend"
    dispatched_at: "2026-03-27T10:00:00Z"
    dispatched_by: "orchestrator-session-abc"
  files_planned: ["internal/auth/login.go", "internal/auth/login_test.go"]
```

### 10.7 Errors

- Unknown `id` format: `"Cannot determine entity type from ID '{id}'. Expected a Plan ID (e.g., P1-basic-ui), Feature ID (FEAT-...), Task ID (TASK-...), or Bug ID (BUG-...)."`
- Entity not found: `"Entity {id} not found"`

---

## 11. Core Tool: `next`

### 11.1 Purpose

Claim work and get full context. Combines work queue inspection, task claiming, and context assembly into a single tool. This is the primary tool that replaces the 1.0 pattern of `work_queue` → `dispatch_task` → `context_assemble`.

### 11.2 Input parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | string | no | Task ID, Feature ID, or Plan ID. If omitted, returns the ready queue. |
| `role` | string | no | Role profile ID for context assembly (e.g., "backend"). If omitted, context is assembled without role-specific filtering. |

### 11.3 Queue inspection mode (no `id`)

When called without an `id`, `next` returns the current ready queue — the same data as `work_queue` in 1.0, including write-through promotion of eligible `queued` → `ready` tasks.

```yaml
queue:
  - task_id: "TASK-01JX..."
    slug: "jwt-middleware"
    summary: "Implement JWT authentication middleware"
    parent_feature: "FEAT-01JA..."
    feature_slug: "user-auth"
    estimate: 5
    age_days: 2
  - task_id: "TASK-01JY..."
    slug: "add-rate-limiting"
    summary: "Add rate limiting to public API endpoints"
    parent_feature: "FEAT-01JB..."
    feature_slug: "api-stability"
    estimate: 3
    age_days: 1
promoted_count: 1
total_queued: 5
side_effects:
  - type: "task_unblocked"
    entity_id: "TASK-01JX..."
    from_status: "queued"
    to_status: "ready"
    trigger: "All dependencies resolved"
```

Sorting follows the Phase 4a rule: estimate ascending (null last), then age descending, then task ID lexicographic.

When `role` is provided in queue inspection mode, results are filtered to tasks whose parent feature matches the role profile.

### 11.4 Claim mode (with `id`)

When called with an `id`, `next` claims the identified task (or the top ready task in the identified plan/feature) and returns full assembled context.

**Task ID:** Claims that specific task.

**Feature ID:** Finds the top ready task in the feature (by the same sort order as the queue) and claims it. If no tasks are ready, returns an error.

**Plan ID:** Finds the top ready task across all features in the plan and claims it. If no tasks are ready, returns an error.

**Claim behaviour:**

1. Transition the task from `ready` to `active`.
2. Set dispatch fields: `claimed_at`, `dispatched_to` (from `role` or "unspecified"), `dispatched_at`, `dispatched_by` (from caller identity or "mcp-session").
3. Assemble the context packet for the task.

**Response:**

```yaml
task:
  id: "TASK-01JX..."
  slug: "jwt-middleware"
  summary: "Implement JWT authentication middleware with RS256 signature verification"
  status: "active"
  estimate: 5
  parent_feature:
    id: "FEAT-01JA..."
    slug: "user-auth"
    plan_id: "P1-basic-ui"
context:
  spec_sections:
    - document: "work/spec/user-auth-spec.md"
      section: "§4.2 Authentication Middleware"
      content: |
        The middleware MUST verify RS256 JWT signatures...
        [extracted relevant content]
    - document: "work/spec/user-auth-spec.md"
      section: "§4.2.1 Token Validation Rules"
      content: |
        Tokens MUST be rejected if...
  acceptance_criteria:
    - "JWT signatures are verified using RS256"
    - "Expired tokens return 401 with error body"
    - "Missing Authorization header returns 401"
  knowledge:
    - topic: "jwt-rs256-key-rotation"
      content: "Key rotation requires graceful handling of the previous key for 24 hours after rotation"
      scope: "backend"
      confidence: 0.85
    - topic: "auth-middleware-pattern"
      content: "Use http.Handler middleware wrapping, not per-route checks"
      scope: "backend"
      confidence: 0.92
  files_context:
    - path: "internal/auth/middleware.go"
      note: "Existing auth package — add JWT verification here"
    - path: "internal/auth/middleware_test.go"
      note: "Existing test file"
  constraints:
    - "Commit format: feat(TASK-01JX...): ..."
    - "Run go test -race ./... before completing"
  role_profile: "backend"
  byte_usage: 4200
  byte_budget: 30720
  trimmed: []
side_effects:
  - type: "status_transition"
    entity_id: "TASK-01JX..."
    entity_type: "task"
    from_status: "ready"
    to_status: "active"
    trigger: "Claimed via next"
```

### 11.5 Context assembly

Context assembly in `next` follows the same logic as Phase 4a `context_assemble` + `dispatch_task`, with the following enhancements:

1. **Spec section extraction:** The system identifies the parent feature's specification document and extracts sections relevant to the task. Relevance is determined by: (a) document intelligence Layer 3 classifications matching the task's domain, (b) entity references to the parent feature in section content, (c) section roles of `requirement`, `constraint`, or `definition`. If Layer 3 classification is not available, the system falls back to extracting all sections from the spec that reference the parent feature.

2. **Acceptance criteria extraction:** If the spec document contains testable criteria for the parent feature (identified by section role `requirement` or by heuristic pattern matching for "MUST", "SHALL", "acceptance criteria" headings), these are extracted into a separate `acceptance_criteria` list.

3. **Knowledge retrieval:** The system queries the knowledge store for entries relevant to the task. Relevance is determined by: (a) scope matching the `role` parameter, (b) tag overlap with the parent feature's tags, (c) topic similarity to the task summary. Entries are sorted by confidence (descending) and included up to the byte budget.

4. **File context:** If the task has `files_planned`, those paths are included. If not, the system uses the parent feature's worktree (if any) to identify recently modified files.

5. **Trimming:** When the assembled context exceeds the byte budget (default 30,720 bytes), entries are trimmed in the order defined by Phase 4a §9.1: lowest-confidence Tier 3 knowledge first, then Tier 2 knowledge, then design context, then spec sections. Profile and task instructions are never trimmed. Trimmed entries are reported in the `trimmed` field.

### 11.6 Lenient lifecycle

If the task is in `active` status (already claimed), `next` returns the "already claimed" error with the existing dispatch metadata, matching Phase 4a `dispatch_task` behaviour. If the task is in any status other than `ready` or `active`, `next` returns an error naming the current status.

### 11.7 Errors

- No ready tasks: `"No ready tasks in {plan/feature} {id}"`
- Task not ready: `"Task {id} is in status '{status}', expected 'ready'"`
- Task already claimed: `"Task {id} is already claimed (dispatched to '{role}' at {timestamp} by {identity})"`
- Entity not found: `"Entity {id} not found"`

---

## 12. Core Tool: `finish`

### 12.1 Purpose

Record task completion. Handles status transitions, completion metadata, side-effect reporting, and inline knowledge contribution in one call. Replaces the 1.0 pattern of `complete_task` + `knowledge_contribute` (+ `update_status`).

### 12.2 Input parameters (single)

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `task_id` | string | yes (single mode) | Task ID to complete |
| `summary` | string | yes | Brief description of what was accomplished |
| `files_modified` | string[] | no | Files created or modified |
| `verification` | string | no | Testing or verification performed |
| `knowledge` | object[] | no | Knowledge entries to contribute (see §12.5) |
| `to_status` | string | no | Target status: `done` (default) or `needs-review` |

### 12.3 Input parameters (batch)

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `tasks` | object[] | yes (batch mode) | Array of task completion objects, each containing `task_id`, `summary`, and optional fields |

### 12.4 Behaviour

1. **Status transition:** Transition the task to `done` (default) or `needs-review`.
2. **Lenient lifecycle:** Accept tasks in `ready` or `active` status. If the task is in `ready` status, internally transition through `active` before completing. This accommodates workflows where the orchestrator and worker are the same agent and formal claiming is unnecessary ceremony.
3. **Completion metadata:** Set `completed` timestamp and `completion_summary` on the task.
4. **Knowledge contribution:** If `knowledge` entries are provided, process each through the existing knowledge contribution pipeline. Duplicates are rejected per-entry without blocking the overall completion.
5. **Dependency unblocking:** After completing the task, fire the dependency unblocking hook (Phase 4b). Any tasks unblocked are reported in `side_effects`.

### 12.5 Inline knowledge format

```yaml
knowledge:
  - topic: "billing-api-idempotency"
    content: "The billing API requires idempotency keys on all POST requests"
    scope: "backend"
    tags: ["billing", "api"]
  - topic: "retry-backoff"
    content: "Retry delays follow exponential backoff with jitter"
    scope: "project"
```

Each knowledge entry follows the same schema as `knowledge_contribute` in 1.0 (topic, content, scope required; tags, tier optional). Tier defaults to 3. The `learned_from` field is automatically set to the completing task's ID.

### 12.6 Response (single)

```yaml
task:
  id: "TASK-01JX..."
  status: "done"
  completed: "2026-03-27T14:30:00Z"
  completion_summary: "Implemented JWT middleware with RS256 verification and full test coverage"
knowledge:
  accepted:
    - entry_id: "KE-01JZ..."
      topic: "jwt-rs256-key-rotation"
  rejected:
    - topic: "golang-error-handling"
      reason: "Duplicate: similar entry KE-01JA... already exists with confidence 0.92"
  total_attempted: 2
  total_accepted: 1
side_effects:
  - type: "task_unblocked"
    entity_id: "TASK-01JY..."
    entity_type: "task"
    from_status: "blocked"
    to_status: "ready"
    trigger: "All dependencies of TASK-01JY... now in terminal state"
  - type: "knowledge_contributed"
    entity_id: "KE-01JZ..."
    topic: "jwt-rs256-key-rotation"
```

### 12.7 Errors

- Task not in completable status: `"Task {id} is in status '{status}', expected 'ready' or 'active'"`
- Task not found: `"Task {id} not found"`
- Missing summary: `"summary is required"`

---

## 13. Core Tool: `handoff`

### 13.1 Purpose

Generate a complete sub-agent prompt from a task. The output is designed to go directly into `spawn_agent`'s `message` parameter. This is the bridge between Kanbanzai's structured dispatch model and the pattern of orchestrators spawning worker agents.

The key difference from `next`: `next` claims a task and returns structured context for the orchestrator. `handoff` takes an already-claimed task and renders a ready-to-use prompt string for a worker agent.

### 13.2 Input parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `task_id` | string | yes | Task ID (should be in `active` status, but see §13.4) |
| `role` | string | no | Role profile ID for context shaping |
| `instructions` | string | no | Additional orchestrator instructions to include in the prompt |

### 13.3 Behaviour

1. Load the task and its parent feature context.
2. Assemble context using the same pipeline as `next` (§11.5): spec section extraction, acceptance criteria, knowledge retrieval, file context.
3. Render the assembled context as a formatted prompt string suitable for `spawn_agent`.

### 13.4 Lenient lifecycle

`handoff` accepts tasks in `active`, `ready`, or `needs-rework` status. It does not modify task status — it is a read operation that assembles context. If the orchestrator wants to claim the task first, they use `next`. If they want to generate a prompt for an already-claimed task (e.g., for a rework cycle), they use `handoff` directly.

### 13.5 Response

```yaml
task_id: "TASK-01JX..."
prompt: |
  ## Task: Implement JWT authentication middleware

  ### Summary
  Implement JWT authentication middleware with RS256 signature verification.

  ### Specification (from work/spec/user-auth-spec.md §4.2)
  The middleware MUST verify RS256 JWT signatures using the public key
  from the configured JWKS endpoint. Expired tokens MUST return 401
  with a JSON error body. Missing Authorization headers MUST return 401.

  ### Acceptance Criteria
  - JWT signatures are verified using RS256
  - Expired tokens return 401 with error body
  - Missing Authorization header returns 401
  - Unit tests cover valid token, expired token, and missing header cases

  ### Known Constraints (from knowledge base)
  - Key rotation requires graceful handling of the previous key for 24 hours
  - Use http.Handler middleware wrapping, not per-route checks

  ### Files
  - internal/auth/middleware.go (create)
  - internal/auth/middleware_test.go (create)
  - internal/auth/keys.go (modify — add JWKS client)

  ### Conventions
  - Commit format: feat(TASK-01JX...): implement JWT authentication middleware
  - Run `go test -race ./...` before completing
  - Do not modify files outside the listed paths without checking with the orchestrator

context_metadata:
  spec_sections_included: 2
  knowledge_entries_included: 2
  byte_usage: 3800
  byte_budget: 30720
  trimmed: []
```

The `prompt` field is a single string, pre-formatted in Markdown. The orchestrator passes it directly to `spawn_agent(message=prompt)`.

The `context_metadata` field is for the orchestrator's information — it is not included in the prompt itself.

### 13.6 Errors

- Task not found: `"Task {id} not found"`
- Task in terminal status: `"Task {id} is in status '{status}' (terminal). Handoff is only meaningful for active or ready tasks."`

---

## 14. Core Tool: `entity`

### 14.1 Purpose

Generic CRUD for all entity types. One tool replaces the 17+ entity-specific tools in 1.0 (`create_task`, `create_feature`, `create_plan`, `create_bug`, `create_epic`, `record_decision`, `get_entity`, `list_entities`, `list_entities_filtered`, `update_entity`, `update_status`, `update_plan`, `update_plan_status`, etc.).

### 14.2 Actions

| Action | Description |
|--------|-------------|
| `create` | Create one or more entities |
| `get` | Get a single entity by ID |
| `list` | List entities with optional filters |
| `update` | Update fields on an entity |
| `transition` | Change an entity's lifecycle status |

### 14.3 `create`

**Input:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `action` | string | yes | `"create"` |
| `type` | string | yes | Entity type: `plan`, `feature`, `task`, `bug`, `epic`, `decision` |
| `entities` | object[] | no | Array of entity creation objects (batch mode) |
| (inline) | varies | no | Single entity fields at top level (single mode) |

Entity-specific required fields follow the existing creation rules from Phases 1–4b. For example:

- **task:** `parent_feature`, `slug`, `summary`
- **feature:** `slug`, `parent` (plan ID), `summary`
- **plan:** `prefix`, `slug`, `title`, `summary`
- **bug:** `slug`, `title`, `reported_by`, `observed`, `expected`, `severity`, `priority`, `type`
- **decision:** `slug`, `summary`, `rationale`
- **epic:** `slug`, `title`, `summary`

**Single mode:**

```
entity(action: "create", type: "task", parent_feature: "FEAT-01JA...", slug: "jwt-middleware", summary: "Implement JWT middleware")
```

**Batch mode:**

```
entity(action: "create", type: "task", entities: [
  {parent_feature: "FEAT-01JA...", slug: "jwt-middleware", summary: "Implement JWT middleware"},
  {parent_feature: "FEAT-01JA...", slug: "login-endpoint", summary: "Implement login endpoint"},
  {parent_feature: "FEAT-01JA...", slug: "token-refresh", summary: "Implement token refresh"}
])
```

**Response (single):**

```yaml
entity:
  id: "TASK-01JX..."
  type: "task"
  slug: "jwt-middleware"
  status: "proposed"
side_effects: []
```

**Response (batch):** Uses the batch response shape from §9.4.

### 14.4 `get`

**Input:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `action` | string | yes | `"get"` |
| `id` | string | yes | Entity ID (type is inferred from prefix) |

**Response:** The full entity record as stored, matching the current `get_entity` output.

### 14.5 `list`

**Input:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `action` | string | yes | `"list"` |
| `type` | string | yes | Entity type to list |
| `parent` | string | no | Filter by parent ID (plan for features, feature for tasks) |
| `status` | string | no | Filter by lifecycle status |
| `tags` | string[] | no | Filter by tags (entities must have at least one) |
| `created_after` | string | no | RFC3339 timestamp filter |
| `created_before` | string | no | RFC3339 timestamp filter |

**Response:**

```yaml
entities:
  - id: "FEAT-01JA..."
    type: "feature"
    slug: "user-auth"
    status: "developing"
    summary: "User authentication with JWT"
  # ... remaining entities
total: 5
```

The list response returns summary records (id, type, slug, status, summary), not full entity records. Use `get` to retrieve full details.

### 14.6 `update`

**Input:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `action` | string | yes | `"update"` |
| `id` | string | yes | Entity ID |
| (fields) | varies | no | Fields to update (entity-type-specific) |

Cannot change `id` or `status` (use `transition` for status changes). Updates follow existing entity update rules from 1.0.

**Response:**

```yaml
entity:
  id: "FEAT-01JA..."
  type: "feature"
  # ... updated fields
side_effects: []
```

### 14.7 `transition`

**Input:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `action` | string | yes | `"transition"` |
| `id` | string | yes | Entity ID |
| `status` | string | yes | Target lifecycle status |

Lifecycle transition rules are unchanged from 1.0. Invalid transitions return an error naming the current status, the requested status, and the valid transitions from the current state.

**Response:**

```yaml
entity:
  id: "FEAT-01JA..."
  type: "feature"
  status: "developing"
side_effects:
  - type: "worktree_created"
    entity_id: "FEAT-01JA..."
    trigger: "Feature transitioned to developing"
```

### 14.8 Type inference from ID

The `get`, `update`, and `transition` actions infer entity type from the ID prefix:

| Prefix | Type |
|--------|------|
| `FEAT-` | feature |
| `TASK-` or `T-` | task |
| `BUG-` | bug |
| `EPIC-` | epic |
| `DEC-` | decision |
| `INC-` | incident |
| Plan prefix (e.g., `P1-`) | plan |

The `type` parameter is optional for these actions when the ID is unambiguous. It is required for `create` and `list`.

---

## 15. Core Tool: `doc`

### 15.1 Purpose

Document registration, approval, and querying. One tool replaces the 11+ document record tools in 1.0 (`doc_record_submit`, `doc_record_approve`, `doc_record_get`, `doc_record_get_content`, `doc_record_list`, `doc_record_list_pending`, `doc_record_validate`, `doc_record_supersede`, `doc_supersession_chain`, `doc_gaps`, `batch_import_documents`).

### 15.2 Actions

| Action | Description |
|--------|-------------|
| `register` | Register a document (single or batch) |
| `approve` | Approve a document (single or batch) |
| `get` | Get a document record by ID or path |
| `content` | Get document file content |
| `list` | List documents with filters |
| `gaps` | Analyse missing documents for a feature |
| `validate` | Validate a document record |
| `supersede` | Mark a document as superseded |
| `import` | Batch import documents from a directory |

### 15.3 `register`

**Single input:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `action` | string | yes | `"register"` |
| `path` | string | yes | Relative path to document file |
| `type` | string | yes | Document type: `design`, `specification`, `dev-plan`, `research`, `report`, `policy` |
| `title` | string | yes | Human-readable title |
| `owner` | string | no | Parent Plan or Feature ID |

**Batch input:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `action` | string | yes | `"register"` |
| `documents` | object[] | yes | Array of `{path, type, title, owner?}` objects |

**Response (single):**

```yaml
document:
  id: "DOC-01JX..."
  path: "work/spec/user-auth-spec.md"
  type: "specification"
  title: "User Authentication Specification"
  status: "draft"
  owner: "FEAT-01JA..."
side_effects: []
```

### 15.4 `approve`

**Input:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `action` | string | yes | `"approve"` |
| `id` | string | yes (single) | Document record ID |
| `ids` | string[] | yes (batch) | Array of document record IDs |

**Response (single):**

```yaml
document:
  id: "DOC-01JX..."
  status: "approved"
  approved_at: "2026-03-27T15:00:00Z"
side_effects:
  - type: "status_transition"
    entity_id: "FEAT-01JA..."
    entity_type: "feature"
    from_status: "specifying"
    to_status: "dev-planning"
    trigger: "Specification document DOC-01JX... approved"
```

This is the canonical example of side-effect reporting: the agent approves a document and learns that the approval triggered a feature transition.

### 15.5 `get`

**Input:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `action` | string | yes | `"get"` |
| `id` | string | no | Document record ID |
| `path` | string | no | Document file path (resolved to ID) |

Either `id` or `path` must be provided. When `path` is provided, the system finds the document record with that path and returns it. This accommodates agents who know the file path but not the record ID.

**Response:** The full document record as stored, plus drift detection (whether the file content has changed since the record was created/approved).

### 15.6 `content`

**Input:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `action` | string | yes | `"content"` |
| `id` | string | yes | Document record ID |

**Response:** The document file content, with a drift warning if content has changed since approval.

### 15.7 `list`

**Input:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `action` | string | yes | `"list"` |
| `type` | string | no | Filter by document type |
| `status` | string | no | Filter by status: `draft`, `approved`, `superseded` |
| `owner` | string | no | Filter by owner entity ID |
| `pending` | boolean | no | If true, list only draft documents awaiting approval (shorthand for `status: "draft"`) |

**Response:**

```yaml
documents:
  - id: "DOC-01JA..."
    type: "specification"
    status: "approved"
    path: "work/spec/user-auth-spec.md"
    title: "User Authentication Specification"
    owner: "FEAT-01JA..."
  # ... remaining documents
total: 12
```

### 15.8 `gaps`

**Input:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `action` | string | yes | `"gaps"` |
| `feature_id` | string | yes | Feature ID to analyse |

**Response:**

```yaml
feature_id: "FEAT-01JA..."
gaps:
  - type: "dev-plan"
    status: "missing"
  - type: "specification"
    status: "draft"      # exists but not approved
present:
  - type: "design"
    status: "approved"
    id: "DOC-01JA..."
```

### 15.9 `import`

**Input:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `action` | string | yes | `"import"` |
| `path` | string | yes | Directory to scan |
| `glob` | string | no | File pattern filter |
| `owner` | string | no | Parent entity ID for imported documents |
| `default_type` | string | no | Fallback document type |

Behaviour matches the existing `batch_import_documents` tool. Idempotent: already-imported files are skipped.

### 15.10 `validate`

**Input:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `action` | string | yes | `"validate"` |
| `id` | string | yes | Document record ID |

**Response:** List of validation issues found (matching existing `doc_record_validate` output).

### 15.11 `supersede`

**Input:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `action` | string | yes | `"supersede"` |
| `id` | string | yes | Document record ID being superseded |
| `superseded_by` | string | yes | Document record ID of the replacement |

**Response:** Confirmation with side effects (if supersession triggers backward lifecycle transitions).

---

## 16. Core Tool: `health`

### 16.1 Purpose

Single diagnostic entry point. Unchanged in behaviour from 1.0.

### 16.2 Input parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| (none) | | | Returns comprehensive project health check |

### 16.3 Response

The same structured health check output as the existing `health_check` tool, including all Phase 3 and Phase 4a/b health check categories.

---

## 17. Feature Group: Planning

### 17.1 `decompose`

Consolidates `decompose_feature`, `decompose_review`, and `slice_analysis` into a single tool with an `action` parameter.

**Actions:**

| Action | Description |
|--------|-------------|
| `propose` | Propose task breakdown from a feature's spec (replaces `decompose_feature`) |
| `review` | Review a proposal against the spec (replaces `decompose_review`) |
| `apply` | Create tasks from a confirmed proposal |
| `slice` | Vertical slice analysis (replaces `slice_analysis`) |

#### `propose`

Input and behaviour match Phase 4b `decompose_feature` (§6.2 of the Phase 4b spec).

#### `review`

Input and behaviour match Phase 4b `decompose_review` (§6.3 of the Phase 4b spec).

#### `apply`

**Input:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `action` | string | yes | `"apply"` |
| `feature_id` | string | yes | Feature ID |
| `proposal` | object | yes | The proposal object from `propose` output |

**Behaviour:**

1. For each task in the proposal, create a task entity via the existing `create_task` logic.
2. Set `depends_on` from the proposal's dependency declarations (resolving proposed slugs to the newly created task IDs).
3. Return all created tasks.

This replaces the 1.0 pattern of manually calling `create_task` N times after reviewing a proposal.

**Response:**

```yaml
feature_id: "FEAT-01JA..."
tasks_created:
  - id: "TASK-01JX..."
    slug: "jwt-middleware"
    status: "proposed"
  - id: "TASK-01JY..."
    slug: "login-endpoint"
    status: "proposed"
    depends_on: ["TASK-01JX..."]
total_created: 2
side_effects: []
```

#### `slice`

Input and behaviour match Phase 4b `slice_analysis` (§10.2 of the Phase 4b spec).

### 17.2 `estimate`

Consolidates `estimate_set`, `estimate_query`, `estimate_reference_add`, and `estimate_reference_remove` into a single tool.

**Actions:**

| Action | Description |
|--------|-------------|
| `set` | Set an estimate on an entity (single or batch) |
| `query` | Query estimate and rollup |
| `add_reference` | Add a calibration reference |
| `remove_reference` | Remove a calibration reference |

#### `set`

**Single input:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `action` | string | yes | `"set"` |
| `entity_id` | string | yes | Entity ID |
| `points` | number | yes | Story points (Modified Fibonacci scale) |

**Batch input:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `action` | string | yes | `"set"` |
| `entities` | object[] | yes | Array of `{entity_id, points}` objects |

Behaviour and response match Phase 4a `estimate_set`, including soft limit warnings and calibration reference inclusion.

#### `query`

Input and behaviour match Phase 4a `estimate_query`.

#### `add_reference` / `remove_reference`

Input and behaviour match Phase 4a `estimate_reference_add` / `estimate_reference_remove`.

### 17.3 `conflict`

Consolidates `conflict_domain_check`.

**Input:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `task_ids` | string[] | yes | Two or more task IDs to check for conflict risk |

Behaviour and response match Phase 4b `conflict_domain_check` (§9.2 of the Phase 4b spec).

---

## 18. Feature Group: Knowledge

### 18.1 `knowledge`

Consolidates `knowledge_contribute`, `knowledge_list`, `knowledge_get`, `knowledge_confirm`, `knowledge_flag`, `knowledge_retire`, `knowledge_compact`, `knowledge_prune`, `knowledge_promote`, `knowledge_update`, `knowledge_resolve_conflict`, and `knowledge_check_staleness`.

**Actions:**

| Action | Description |
|--------|-------------|
| `list` | List knowledge entries with filters |
| `get` | Get a knowledge entry by ID |
| `contribute` | Create a new knowledge entry |
| `confirm` | Confirm a contributed entry |
| `flag` | Flag an entry as incorrect |
| `retire` | Retire an entry |
| `update` | Update entry content |
| `promote` | Promote Tier 3 to Tier 2 |
| `compact` | Merge near-duplicates |
| `prune` | Remove expired entries |
| `resolve` | Resolve a conflict between two entries |
| `staleness` | Check staleness of anchored entries |

Each action's input and behaviour match the corresponding 1.0 tool. The consolidation is purely structural — the knowledge logic is unchanged.

Note: routine knowledge contribution happens inline via `finish` (§12.5). Routine knowledge retrieval happens automatically via `next` (§11.5) and `handoff` (§13.3). The `knowledge` tool is for direct knowledge management — confirming entries, resolving conflicts, pruning — which is uncommon in normal orchestration sessions.

### 18.2 `profile`

Consolidates `profile_list` and `profile_get`.

**Actions:**

| Action | Description |
|--------|-------------|
| `list` | List all context profiles |
| `get` | Get a profile by ID (resolved or raw) |

Each action's input and behaviour match the corresponding 1.0 tool.

---

## 19. Feature Group: Git

### 19.1 `worktree`

Consolidates `worktree_create`, `worktree_get`, `worktree_list`, and `worktree_remove`.

**Actions:**

| Action | Description |
|--------|-------------|
| `create` | Create a worktree for an entity |
| `get` | Get worktree record for an entity |
| `list` | List worktrees with optional filters |
| `remove` | Remove a worktree |

Each action's input and behaviour match the corresponding 1.0 tool.

### 19.2 `merge`

Consolidates `merge_readiness_check` and `merge_execute`.

**Actions:**

| Action | Description |
|--------|-------------|
| `check` | Check merge readiness (gates) |
| `execute` | Execute merge after gate verification |

Each action's input and behaviour match the corresponding 1.0 tool.

### 19.3 `pr`

Consolidates `pr_create`, `pr_status`, and `pr_update`.

**Actions:**

| Action | Description |
|--------|-------------|
| `create` | Create a PR for an entity |
| `status` | Get PR status |
| `update` | Update PR description and labels |

Each action's input and behaviour match the corresponding 1.0 tool.

### 19.4 `branch`

Consolidates `branch_status`.

**Actions:**

| Action | Description |
|--------|-------------|
| `status` | Get branch health metrics |

Input and behaviour match the existing `branch_status` tool.

### 19.5 `cleanup`

Consolidates `cleanup_list` and `cleanup_execute`.

**Actions:**

| Action | Description |
|--------|-------------|
| `list` | List worktrees pending cleanup |
| `execute` | Execute cleanup on a worktree |

Each action's input and behaviour match the corresponding 1.0 tool.

---

## 20. Feature Group: Documents

### 20.1 `doc_intel`

Consolidates `doc_outline`, `doc_section`, `doc_classify`, `doc_find_by_concept`, `doc_find_by_entity`, `doc_find_by_role`, `doc_trace`, `doc_impact`, `doc_extraction_guide`, and `doc_pending`.

**Actions:**

| Action | Description |
|--------|-------------|
| `outline` | Get structural outline of a document |
| `section` | Get a specific section's content |
| `classify` | Submit agent classifications for a document |
| `find` | Find sections by concept, entity, or role |
| `trace` | Trace an entity through the refinement chain |
| `impact` | Find what depends on a section |
| `guide` | Get extraction guide for a document |
| `pending` | List documents awaiting classification |

The `find` action accepts one of `concept`, `entity_id`, or `role` to determine the search type:

```
doc_intel(action: "find", concept: "payment retry")
doc_intel(action: "find", entity_id: "FEAT-01JA...")
doc_intel(action: "find", role: "requirement")
```

Each action's input and behaviour match the corresponding 1.0 tool. The consolidation is purely structural.

Note: document intelligence powers `next`, `handoff`, and `status` invisibly. The `doc_intel` feature group is for explicit document exploration — useful for research tasks or onboarding to an unfamiliar project. It is not part of the normal orchestration workflow.

---

## 21. Feature Group: Incidents

### 21.1 `incident`

Consolidates `incident_create`, `incident_update`, `incident_list`, and `incident_link_bug`.

**Actions:**

| Action | Description |
|--------|-------------|
| `create` | Create a new incident |
| `update` | Update an incident |
| `list` | List incidents with filters |
| `link_bug` | Link a bug to an incident |

Each action's input and behaviour match the corresponding 1.0 tool from Phase 4b.

---

## 22. Feature Group: Checkpoints

### 22.1 `checkpoint`

Consolidates `human_checkpoint`, `human_checkpoint_get`, `human_checkpoint_respond`, and `human_checkpoint_list`.

**Actions:**

| Action | Description |
|--------|-------------|
| `create` | Create a new checkpoint |
| `get` | Get checkpoint state |
| `respond` | Record a human response |
| `list` | List checkpoints with optional status filter |

Each action's input and behaviour match the corresponding 1.0 tool from Phase 4a.

---

## 23. CLI Alignment

### 23.1 Principle

The CLI adopts the same resource-oriented pattern as the MCP tools. Commands follow the structure `kbz <resource> <action> [args]`, matching the tool's `action` parameter.

### 23.2 Core commands

```
kbz status                          Project overview
kbz status <id>                     Plan, feature, or task dashboard
kbz next                            Show the ready queue
kbz next <id>                       Claim a task and print context
kbz finish <task-id> -m "summary"   Complete a task
kbz handoff <task-id>               Print a sub-agent prompt
kbz entity create <type> [fields]   Create an entity
kbz entity get <id>                 Get an entity
kbz entity list <type> [filters]    List entities
kbz entity transition <id> <status> Transition status
kbz doc register <path> [options]   Register a document
kbz doc approve <id>                Approve a document
kbz doc list [filters]              List documents
kbz health                          Run health check
```

### 23.3 Feature group commands

```
kbz decompose <feature-id>                  Propose decomposition
kbz decompose <feature-id> --review         Review a proposal
kbz decompose <feature-id> --apply          Apply a proposal (creates tasks)
kbz decompose <feature-id> --slice          Vertical slice analysis
kbz estimate <entity-id> <points>           Set estimate
kbz estimate <entity-id>                    Query estimate
kbz conflict <task-id> <task-id> [...]      Conflict analysis
kbz knowledge list [filters]                List entries
kbz knowledge get <id>                      Get entry
kbz worktree create <entity-id>             Create worktree
kbz worktree list                           List worktrees
kbz merge check <entity-id>                 Check merge readiness
kbz merge <entity-id>                       Execute merge
kbz pr create <entity-id>                   Create PR
kbz pr status <entity-id>                   PR status
kbz incident create [fields]                Create incident
kbz incident list [filters]                 List incidents
kbz checkpoint create [fields]              Create checkpoint
kbz checkpoint respond <id> "response"      Respond to checkpoint
```

### 23.4 CLI availability

All CLI commands are available regardless of feature group configuration. Feature groups control MCP tool registration (context window), not CLI availability. The CLI is for humans, who are not subject to context-window constraints.

---

## 24. Invisible Infrastructure Integration

### 24.1 Purpose

Document intelligence and knowledge management are existing subsystems that work correctly but were exposed as separate tools in 1.0, leading to non-use. In 2.0, they are wired into the core tools as invisible infrastructure.

### 24.2 Integration points

| Core tool | Document intelligence | Knowledge system |
|-----------|----------------------|-----------------|
| `status` | Document gaps included in attention items | — |
| `next` (claim) | Spec section extraction, acceptance criteria | Relevant entries by scope/tag/topic |
| `handoff` | Spec section extraction, acceptance criteria, file references | Relevant entries by scope/tag/topic |
| `finish` | — | Inline contribution pipeline |
| `doc(approve)` | — (triggers re-index if needed) | — |

### 24.3 Graceful degradation

If document intelligence has not been run on a project (no Layer 1–3 index), the core tools still work:

- `next` and `handoff` include the full spec document path instead of extracted sections. The agent receives "read this document" instead of "here are the relevant sections."
- `status` omits document gap analysis.
- `finish` knowledge contribution works regardless of intelligence state.

This ensures small projects that haven't invested in document classification still get value from the workflow tools.

### 24.4 Automatic indexing

When `next` or `handoff` encounters a spec document that has not been indexed (no Layer 1 outline exists), it triggers a synchronous Layer 1–2 parse before assembly. Layer 3 classification is not triggered automatically — it requires agent involvement and is deferred to the `doc_intel` feature group.

This means the first `next` call on a feature with an unindexed spec may be slightly slower (document parse), but subsequent calls are fast (cached index).

---

## 25. 1.0 Tool Removal

### 25.1 Removal strategy

All 1.0 tools are removed when 2.0 is deployed. There is no `legacy` feature group or backward compatibility shim.

**Rationale:** The 1.0 and 2.0 tool surfaces have fundamentally different interaction patterns (per-entity-per-operation vs. resource-oriented with actions). A compatibility layer would need to translate between these patterns, adding complexity without providing a meaningful migration path. The internal model is unchanged, so any workflow that worked in 1.0 works in 2.0 — just with different tool calls.

### 25.2 Removed tools

The following 1.0 tools are removed and replaced by their 2.0 equivalents:

**Entity tools (replaced by `entity`):**
`create_task`, `create_feature`, `create_plan`, `create_bug`, `create_epic`, `record_decision`, `get_entity`, `list_entities`, `list_entities_filtered`, `update_entity`, `update_status`, `update_plan`, `update_plan_status`, `validate_candidate`, `check_duplicates`, `list_by_tag`, `kanbanzai_list_tags`, `suggest_links`

**Document tools (replaced by `doc`):**
`doc_record_submit`, `doc_record_approve`, `doc_record_get`, `doc_record_get_content`, `doc_record_list`, `doc_record_list_pending`, `doc_record_validate`, `doc_record_supersede`, `doc_supersession_chain`, `doc_gaps`, `batch_import_documents`

**Dispatch/completion tools (replaced by `next` and `finish`):**
`dispatch_task`, `complete_task`, `context_assemble`, `context_report`, `work_queue`

**Knowledge tools (replaced by `knowledge` in knowledge group, and inline in `finish`):**
`knowledge_contribute`, `knowledge_list`, `knowledge_get`, `knowledge_confirm`, `knowledge_flag`, `knowledge_retire`, `knowledge_update`, `knowledge_promote`, `knowledge_compact`, `knowledge_prune`, `knowledge_resolve_conflict`, `knowledge_check_staleness`

**Profile tools (replaced by `profile`):**
`profile_list`, `profile_get`

**Document intelligence tools (replaced by `doc_intel`):**
`doc_outline`, `doc_section`, `doc_classify`, `doc_find_by_concept`, `doc_find_by_entity`, `doc_find_by_role`, `doc_trace`, `doc_impact`, `doc_extraction_guide`, `doc_pending`

**Worktree tools (replaced by `worktree`):**
`worktree_create`, `worktree_get`, `worktree_list`, `worktree_remove`

**Merge tools (replaced by `merge`):**
`merge_readiness_check`, `merge_execute`

**PR tools (replaced by `pr`):**
`pr_create`, `pr_status`, `pr_update`

**Branch tools (replaced by `branch`):**
`branch_status`

**Cleanup tools (replaced by `cleanup`):**
`cleanup_list`, `cleanup_execute`

**Decomposition tools (replaced by `decompose`):**
`decompose_feature`, `decompose_review`, `slice_analysis`

**Estimation tools (replaced by `estimate`):**
`estimate_set`, `estimate_query`, `estimate_reference_add`, `estimate_reference_remove`

**Conflict tools (replaced by `conflict`):**
`conflict_domain_check`

**Review tools (absorbed into `finish` and removed as standalone):**
`review_task_output`

**Incident tools (replaced by `incident`):**
`incident_create`, `incident_update`, `incident_list`, `incident_link_bug`

**Checkpoint tools (replaced by `checkpoint`):**
`human_checkpoint`, `human_checkpoint_get`, `human_checkpoint_respond`, `human_checkpoint_list`

**Health tools (replaced by `health` — name unchanged, tool unchanged):**
`health_check`

**Miscellaneous tools absorbed or removed:**
`add_prefix`, `get_prefix_registry`, `get_project_config`, `get_plan`, `list_plans`, `query_plan_tasks`, `rebuild_cache`, `migrate_phase2`, `dependency_status`

### 25.3 Relocated functionality

Some 1.0 tools have functionality that moves rather than being removed:

| 1.0 Tool | 2.0 Location |
|----------|-------------|
| `dispatch_task` | `next` (claim mode) |
| `complete_task` | `finish` |
| `context_assemble` | Inside `next` and `handoff` (invisible) |
| `context_report` | Inside `finish` (automatic) |
| `work_queue` | `next` (queue inspection mode) |
| `review_task_output` | `finish` with `to_status: "needs-review"`, or retained as an action on `entity` if standalone review is needed |
| `check_duplicates` | Inside `entity(action: create)` (automatic, advisory) |
| `suggest_links` | Inside `entity(action: create)` and `doc(action: register)` (automatic) |
| `get_prefix_registry` | Inside `entity(action: create, type: plan)` error messages |
| `get_project_config` | `status()` project overview includes relevant config |
| `dependency_status` | `status(task_id)` includes dependency information |
| `query_plan_tasks` | `status(plan_id)` includes task rollup |

### 25.4 Test updates

`TestServer_ListTools` in `internal/mcp/server_test.go` must be updated to validate the 2.0 tool list (7–20 tools depending on configuration) instead of the 1.0 list.

---

## 26. Open Questions Resolved During Specification

### 26.1 Cross-feature task dependencies

**Question:** Should tasks be able to declare dependencies on features (not just other tasks)?

**Decision:** No. Task-to-task dependencies are sufficient. If a task depends on the completion of another feature, the correct model is to depend on a specific terminal task within that feature (e.g., the integration-test or merge task). Feature-level dependencies are too coarse — a feature being "done" encompasses many tasks, and depending on all of them creates unnecessary serialisation.

**Rationale:** Cross-feature dependencies add validation complexity (cycle detection across features, cascading unblock across feature boundaries) without clear benefit over explicit task-to-task dependencies. The orchestrator can express "wait for feature X's final task" directly.

### 26.2 Document type taxonomy

**Question:** Should `documentation` or `guide` be added as a document type?

**Decision:** Deferred. The current taxonomy (`design`, `specification`, `dev-plan`, `research`, `report`, `policy`) covers the workflow documents. User-facing documentation is not managed by the workflow system in Phases 1–4b. If a `documentation` type is needed, it can be added without schema changes (the type field is a string, not an enum in storage).

### 26.3 Backward compatibility

**Question:** Should 2.0 provide a `legacy` feature group mapping 1.0 tool names?

**Decision:** No. Clean break. See §25.1 for rationale.

### 26.4 `review_task_output` disposition

**Question:** Phase 4b's `review_task_output` is a standalone tool. Where does it live in 2.0?

**Decision:** The review capability is retained as an implicit part of `finish`. When `finish` is called with `to_status: "needs-review"`, it performs the same review logic. For explicit pre-completion review (review without changing status), the orchestrator uses `entity(action: "get")` to inspect the task's current state and files, then decides whether to `finish` with `done` or `needs-review`. The dedicated `review_task_output` tool is removed.

**Rationale:** Standalone review was used in exactly one workflow pattern: "review then complete." Folding it into `finish` with `to_status: "needs-review"` eliminates a tool without losing the capability. The review findings are included in the `finish` response when `needs-review` is the target.

### 26.5 `work_queue` conflict annotation

**Question:** The 1.0 `work_queue` has an optional `conflict_check` flag. Where does this live in 2.0?

**Decision:** `next()` (queue inspection mode) accepts an optional `conflict_check: true` parameter that annotates each ready task with conflict risk against currently active tasks. This matches the Phase 4b `work_queue --conflict-check` behaviour.

---

## 27. Configuration

### 27.1 New configuration section

The following is added to `.kbz/config.yaml`:

```yaml
mcp:
  groups:
    core: true
    planning: false
    knowledge: false
    git: false
    documents: false
    incidents: false
    checkpoints: false
  # OR:
  preset: orchestration
```

### 27.2 Existing configuration

All existing configuration sections (`dispatch`, `estimation`, `incidents`, `decomposition`) are unchanged. They continue to govern the internal behaviour of the corresponding subsystems.

---

## 28. Storage Model

### 28.1 No storage changes

Kanbanzai 2.0 does not change the storage model. Entity schemas, lifecycle state machines, YAML serialisation rules, and file layouts are unchanged from Phases 1–4b.

The tool surface is a presentation layer over the existing storage and service layers. The internal `EntityService`, `PlanService`, `DocumentRecordService`, `KnowledgeStore`, `WorktreeStore`, and other service types continue to work as they do today.

### 28.2 New configuration fields

The only storage-adjacent change is the new `mcp` section in `.kbz/config.yaml` (§27.1). This is configuration, not entity state.

---

## 29. Implementation Features

The following features define the implementation work. They are listed in dependency order.

### Feature 1: Feature Group Framework & Tool Registration

**Summary:** Build the configuration model and conditional tool registration mechanism.

**Scope:**
- Parse `mcp.groups` and `mcp.preset` from `.kbz/config.yaml`
- Resolve presets with explicit overrides
- Conditional tool registration in the MCP server startup
- Validation (unknown groups, unknown presets, core override)
- Default behaviour when no `mcp` section exists (`preset: full`)

**Dependencies:** None (foundational).

**Acceptance criteria:** §30.1

### Feature 2: Resource-Oriented Tool Pattern & Side-Effect Reporting

**Summary:** Build the action-dispatch framework and the side-effect reporting contract that all consolidated tools use.

**Scope:**
- Action parameter dispatch mechanism (shared by all consolidated tools)
- Parameter validation per action
- Error response shape (§7.4)
- Side-effect collector: a request-scoped mechanism that accumulates side effects from cascading operations
- Side-effect response shape (§8.2)
- Integration with existing `StatusTransitionHook` and document approval cascades

**Dependencies:** Feature 1 (tools register through the group framework).

**Acceptance criteria:** §30.2

### Feature 3: Batch Operations

**Summary:** Build the array-accepting input pattern, partial-failure execution, and batch response shape.

**Scope:**
- Single-or-array input detection
- Best-effort execution with per-item error isolation
- Batch response shape (§9.4)
- Batch limit enforcement (100 items)
- Integration with side-effect reporting (aggregate side effects)

**Dependencies:** Feature 2 (batch responses include side effects).

**Acceptance criteria:** §30.3

### Feature 4: `status` — Synthesis Dashboard

**Summary:** Build the `status` tool that synthesises project, plan, feature, and task state.

**Scope:**
- Entity type inference from ID format
- Project overview synthesis (§10.3)
- Plan dashboard synthesis (§10.4)
- Feature detail synthesis (§10.5)
- Task detail synthesis (§10.6)
- Health summary integration
- Document gap integration
- Attention items generation

**Dependencies:** Feature 1 (tool registration). Does not require Features 2–3 (status is read-only, no side effects or batching needed).

**Acceptance criteria:** §30.4

### Feature 5: `next` — Claim & Context Assembly

**Summary:** Build the `next` tool that combines work queue, task claiming, and context assembly with invisible intelligence integration.

**Scope:**
- Queue inspection mode (write-through promotion, sorting, optional conflict check)
- Claim mode with plan/feature/task targeting
- Context assembly pipeline: spec section extraction, acceptance criteria, knowledge retrieval, file context
- Document intelligence integration (graceful degradation without index)
- Automatic Layer 1–2 parse on first encounter
- Trimming logic with reporting
- Lenient lifecycle (already-claimed error)

**Dependencies:** Feature 2 (side-effect reporting for promotion and claiming).

**Acceptance criteria:** §30.5

### Feature 6: `finish` — Completion & Inline Knowledge

**Summary:** Build the `finish` tool with inline knowledge contribution and lenient lifecycle.

**Scope:**
- Single and batch completion
- Lenient lifecycle (accept `ready` or `active`)
- Inline knowledge contribution pipeline
- Dependency unblocking hook integration
- Side-effect reporting for unblocked tasks and contributed knowledge

**Dependencies:** Feature 2 (side effects), Feature 3 (batch).

**Acceptance criteria:** §30.6

### Feature 7: `handoff` — Sub-Agent Prompt Generation

**Summary:** Build the `handoff` tool that renders a complete sub-agent prompt.

**Scope:**
- Context assembly (shared pipeline with `next`)
- Prompt rendering in Markdown format
- Context metadata reporting
- Lenient lifecycle (active, ready, needs-rework)
- `instructions` parameter for orchestrator additions

**Dependencies:** Feature 5 (shares the context assembly pipeline).

**Acceptance criteria:** §30.7

### Feature 8: `entity` — Consolidated Entity CRUD

**Summary:** Consolidate 17+ entity-specific tools into one resource-oriented tool.

**Scope:**
- Action dispatch: create, get, list, update, transition
- Batch create
- Type inference from ID prefix
- Side-effect reporting for transitions (worktree creation, etc.)
- Duplicate check integration (advisory, inside `create`)
- Summary records for `list` (not full entities)

**Dependencies:** Feature 2 (action dispatch, side effects), Feature 3 (batch create).

**Acceptance criteria:** §30.8

### Feature 9: `doc` — Consolidated Document Operations

**Summary:** Consolidate 11+ document tools into one resource-oriented tool.

**Scope:**
- Action dispatch: register, approve, get, content, list, gaps, validate, supersede, import
- Batch register and approve
- Path-based lookup (resolve path to ID)
- Side-effect reporting for approval cascades
- Drift detection in `get` and `content`

**Dependencies:** Feature 2 (action dispatch, side effects), Feature 3 (batch).

**Acceptance criteria:** §30.9

### Feature 10: Consolidated Feature Group Tools

**Summary:** Consolidate the remaining feature group tools into their resource-oriented forms.

**Scope:**
- `decompose` (planning group): propose, review, apply, slice
- `estimate` (planning group): set, query, add_reference, remove_reference
- `conflict` (planning group): single action
- `knowledge` (knowledge group): 12 actions
- `profile` (knowledge group): list, get
- `worktree` (git group): create, get, list, remove
- `merge` (git group): check, execute
- `pr` (git group): create, status, update
- `branch` (git group): status
- `cleanup` (git group): list, execute
- `doc_intel` (documents group): 8 actions
- `incident` (incidents group): create, update, list, link_bug
- `checkpoint` (checkpoints group): create, get, respond, list

Each tool's internal logic is unchanged — this is a re-packaging of existing tools into the action-parameter pattern.

**Dependencies:** Feature 1 (group registration), Feature 2 (action dispatch). Feature 3 for tools with batch support (`estimate(set)`, `knowledge(contribute)`).

**Acceptance criteria:** §30.10

### Feature 11: 1.0 Tool Removal & Migration

**Summary:** Remove all 1.0 tools and update tests.

**Scope:**
- Remove all 1.0 tool registrations from the MCP server
- Update `TestServer_ListTools` to validate the 2.0 tool set
- Remove any dead code from tool handlers that are fully replaced
- Update CLI commands to match the resource-oriented pattern
- Verify `go test -race ./...` passes with no 1.0 tools

**Dependencies:** All of Features 1–10 (the 2.0 tools must exist before 1.0 tools are removed).

**Acceptance criteria:** §30.11

---

## 30. Acceptance Criteria

### 30.1 Feature group framework

- [ ] `.kbz/config.yaml` `mcp.groups` section is parsed on server startup
- [ ] Setting a group to `true` registers all tools in that group
- [ ] Setting a group to `false` registers no tools from that group
- [ ] The `core` group is always registered regardless of configuration
- [ ] Setting `core: false` produces a startup warning and is overridden to `true`
- [ ] `mcp.preset: minimal` registers exactly the 7 core tools
- [ ] `mcp.preset: orchestration` registers core + planning + git (15 tools)
- [ ] `mcp.preset: full` registers all 20 tools
- [ ] Explicit `groups` override preset values
- [ ] Unrecognised group names produce a startup warning and are ignored
- [ ] Unrecognised preset names produce a startup error
- [ ] When no `mcp` section exists, the default behaviour is `preset: full`
- [ ] The MCP `tools/list` response contains only enabled tools

### 30.2 Resource-oriented tool pattern & side-effect reporting

- [ ] All consolidated tools accept an `action` parameter as the first required field
- [ ] Invalid `action` values return an error listing valid actions for that tool
- [ ] Parameters irrelevant to the current action are ignored (not rejected)
- [ ] Error responses use the consistent structure: `code`, `message`, optional `details`
- [ ] Every mutation response includes a `side_effects` field
- [ ] `side_effects` is an empty array when no cascades occur
- [ ] Document approval cascading a feature transition is reported as a side effect
- [ ] Task completion unblocking a dependent task is reported as a side effect
- [ ] Worktree auto-creation on entity transition is reported as a side effect
- [ ] Read-only operations do not include a `side_effects` field (except `next` queue mode)

### 30.3 Batch operations

- [ ] Batch-capable tools accept an array of items
- [ ] Single-item calls return the single-item response shape (not a batch wrapper)
- [ ] Batch calls return the batch response shape with per-item results and summary
- [ ] A failure on one batch item does not prevent processing of subsequent items
- [ ] Batch responses include per-item `status` ("ok" or "error")
- [ ] Batch responses include aggregate `side_effects` (union of all per-item effects)
- [ ] Batches exceeding 100 items are rejected with `batch_limit_exceeded` before processing
- [ ] `summary.total`, `summary.succeeded`, and `summary.failed` are correct

### 30.4 `status` tool

- [ ] `status()` returns project overview with plans, health summary, and attention items
- [ ] `status(plan_id)` returns plan dashboard with features, documents, and attention items
- [ ] `status(feature_id)` returns feature detail with tasks, documents, estimate, and worktree
- [ ] `status(task_id)` returns task detail with parent context and dependencies
- [ ] Entity type is correctly inferred from the ID format
- [ ] Unknown ID formats return a clear error
- [ ] Entity not found returns a clear error
- [ ] Health summary is included in project and plan views
- [ ] Document gaps are included in plan and feature views
- [ ] `attention` items highlight actionable issues (stalled tasks, missing docs, ready tasks)

### 30.5 `next` tool

- [ ] `next()` returns the ready queue with write-through promotion
- [ ] `next()` returns promoted tasks in `side_effects`
- [ ] `next()` queue is sorted: estimate ascending (null last), age descending, ID lexicographic
- [ ] `next(role: "backend")` filters queue to matching features
- [ ] `next(task_id)` claims the task and transitions it to `active`
- [ ] `next(task_id)` returns assembled context with spec sections, knowledge, and acceptance criteria
- [ ] `next(plan_id)` claims the top ready task in the plan
- [ ] `next(feature_id)` claims the top ready task in the feature
- [ ] `next(plan_id)` returns an error when no tasks are ready
- [ ] Context includes spec sections extracted by document intelligence when available
- [ ] Context falls back to full document path when intelligence index is not available
- [ ] Knowledge entries are included, sorted by confidence, within byte budget
- [ ] Trimmed entries are reported in the `trimmed` field
- [ ] Already-claimed tasks return the "already claimed" error with dispatch metadata
- [ ] Automatic Layer 1–2 parse is triggered for unindexed spec documents
- [ ] `next(conflict_check: true)` annotates queue items with conflict risk

### 30.6 `finish` tool

- [ ] `finish(task_id, summary)` completes a task in `active` status
- [ ] `finish` accepts tasks in `ready` status (lenient lifecycle)
- [ ] `finish` with `to_status: "needs-review"` transitions to `needs-review`
- [ ] `finish` defaults to `to_status: "done"`
- [ ] `completion_summary` and `completed` timestamp are set on the task
- [ ] Inline `knowledge` entries are processed through the contribution pipeline
- [ ] Duplicate knowledge entries are rejected per-entry without blocking completion
- [ ] Knowledge contributions are reported in the response
- [ ] Unblocked tasks are reported in `side_effects`
- [ ] Batch `finish` processes each task independently (best-effort)
- [ ] Batch `finish` returns per-item results with aggregate side effects

### 30.7 `handoff` tool

- [ ] `handoff(task_id)` returns a complete prompt string in Markdown
- [ ] Prompt includes task summary, spec sections, acceptance criteria, knowledge, file paths, and conventions
- [ ] Prompt is designed for direct use in `spawn_agent(message=...)`
- [ ] `context_metadata` reports byte usage, sections included, and trimmed entries
- [ ] `handoff` accepts tasks in `active`, `ready`, or `needs-rework` status
- [ ] `handoff` does not modify task status (read-only assembly)
- [ ] `handoff(role: "backend")` shapes context with role-specific knowledge
- [ ] `handoff(instructions: "...")` includes orchestrator instructions in the prompt
- [ ] `handoff` on a terminal-status task returns an error

### 30.8 `entity` tool

- [ ] `entity(action: "create", type: "task", ...)` creates a task
- [ ] `entity(action: "create", type: "feature", ...)` creates a feature
- [ ] `entity(action: "create", type: "plan", ...)` creates a plan
- [ ] `entity(action: "create", type: "bug", ...)` creates a bug
- [ ] `entity(action: "create", type: "decision", ...)` creates a decision
- [ ] Batch create with `entities` array creates multiple entities in one call
- [ ] `entity(action: "get", id: "FEAT-...")` returns the full entity record
- [ ] `entity(action: "get")` infers entity type from ID prefix
- [ ] `entity(action: "list", type: "task", parent: "FEAT-...")` returns filtered summary records
- [ ] `entity(action: "list")` returns summary records (id, type, slug, status, summary)
- [ ] `entity(action: "update", id: "FEAT-...", ...)` updates entity fields
- [ ] `entity(action: "update")` cannot change `id` or `status`
- [ ] `entity(action: "transition", id: "FEAT-...", status: "developing")` transitions status
- [ ] Invalid transitions return an error with current status and valid transitions
- [ ] Side effects from transitions (worktree creation, etc.) are reported

### 30.9 `doc` tool

- [ ] `doc(action: "register", path: "...", type: "...", title: "...")` creates a document record
- [ ] Batch register with `documents` array registers multiple documents
- [ ] `doc(action: "approve", id: "...")` approves a document
- [ ] Approval side effects (feature transitions) are reported
- [ ] Batch approve with `ids` array approves multiple documents
- [ ] `doc(action: "get", id: "...")` returns the full document record
- [ ] `doc(action: "get", path: "...")` resolves path to ID and returns the record
- [ ] `doc(action: "content", id: "...")` returns document file content
- [ ] `doc(action: "list")` with filters returns matching documents
- [ ] `doc(action: "gaps", feature_id: "...")` returns missing document analysis
- [ ] `doc(action: "import", path: "...")` batch imports documents (idempotent)
- [ ] `doc(action: "validate", id: "...")` returns validation issues
- [ ] `doc(action: "supersede", id: "...", superseded_by: "...")` supersedes a document

### 30.10 Consolidated feature group tools

- [ ] `decompose(action: "propose", feature_id: "...")` returns a task proposal
- [ ] `decompose(action: "review", feature_id: "...", proposal: {...})` returns review findings
- [ ] `decompose(action: "apply", feature_id: "...", proposal: {...})` creates tasks from a proposal
- [ ] `decompose(action: "slice", feature_id: "...")` returns vertical slice analysis
- [ ] `estimate(action: "set", entity_id: "...", points: 5)` sets an estimate
- [ ] `estimate(action: "set", entities: [...])` batch sets estimates
- [ ] `estimate(action: "query", entity_id: "...")` returns estimate and rollup
- [ ] `conflict(task_ids: [...])` returns conflict risk assessment
- [ ] `knowledge(action: "list", ...)` lists knowledge entries
- [ ] `knowledge(action: "contribute", ...)` creates a knowledge entry
- [ ] `profile(action: "list")` lists context profiles
- [ ] `profile(action: "get", id: "...")` returns a profile
- [ ] `worktree(action: "create", entity_id: "...")` creates a worktree
- [ ] `worktree(action: "list")` lists worktrees
- [ ] `merge(action: "check", entity_id: "...")` checks merge readiness
- [ ] `merge(action: "execute", entity_id: "...")` executes merge
- [ ] `pr(action: "create", entity_id: "...")` creates a PR
- [ ] `pr(action: "status", entity_id: "...")` returns PR status
- [ ] `doc_intel(action: "outline", id: "...")` returns document outline
- [ ] `doc_intel(action: "find", concept: "...")` finds sections by concept
- [ ] `doc_intel(action: "find", entity_id: "...")` finds sections by entity reference
- [ ] `doc_intel(action: "trace", entity_id: "...")` traces entity through refinement chain
- [ ] `incident(action: "create", ...)` creates an incident
- [ ] `incident(action: "list", ...)` lists incidents
- [ ] `checkpoint(action: "create", ...)` creates a checkpoint
- [ ] `checkpoint(action: "respond", id: "...", response: "...")` records a response
- [ ] All feature group tools are registered only when their group is enabled
- [ ] All feature group tools use the resource-oriented action dispatch pattern

### 30.11 1.0 tool removal

- [ ] No 1.0 tools are registered by the MCP server
- [ ] `TestServer_ListTools` validates the 2.0 tool set (7–20 tools depending on config)
- [ ] CLI commands follow the `kbz <resource> <action>` pattern
- [ ] `go test -race ./...` passes with all 1.0 tools removed
- [ ] No dead code remains from 1.0 tool handlers that are fully replaced

---

## 31. Implementation Notes

### 31.1 Implementation sequence

Features 1–3 are infrastructure. They should be implemented first as they provide the framework every other feature depends on.

Features 4–7 are the high-value core tools. They represent the MVP — the tools that fundamentally change the orchestrator experience. Feature 5 (`next`) is the most complex and should be approached last in this group, after `status` (Feature 4) proves the synthesis pattern and `finish` (Feature 6) proves the side-effect pipeline.

Feature 8 (`entity`) and Feature 9 (`doc`) are lower-risk consolidations of existing functionality.

Feature 10 is a parallelisable cleanup pass — each tool in the feature groups is independent.

Feature 11 (removal) is the cutover and must come last.

Suggested order:
1. Feature 1 (group framework)
2. Feature 2 (action dispatch + side effects)
3. Feature 3 (batch operations)
4. Feature 4 (`status`)
5. Feature 6 (`finish`) — validates side-effect pipeline
6. Feature 5 (`next`) — the most complex tool
7. Feature 7 (`handoff`) — shares pipeline with `next`
8. Features 8–9 (`entity`, `doc`) — can be parallelised
9. Feature 10 (feature group tools) — heavily parallelisable
10. Feature 11 (removal) — final cutover

### 31.2 Side-effect collector pattern

The side-effect reporting mechanism should be implemented as a request-scoped collector. Each service method that produces a cascade pushes a side effect onto the collector. At response time, the MCP handler drains the collector into the response.

This avoids threading side-effect arrays through every service method signature. The collector is created at the start of each MCP request and attached to the request context.

### 31.3 Context assembly shared pipeline

`next` and `handoff` share the same context assembly pipeline. The difference is in output: `next` returns structured data, `handoff` renders a Markdown prompt. The pipeline should be implemented once in `internal/context/` (or a new package) and called from both tool handlers.

### 31.4 Dual registration during development

During development, both 1.0 and 2.0 tools may coexist temporarily. The feature group framework should support this by treating the existing 1.0 tools as an implicit `legacy` group that is enabled by default and can be disabled. This is a development convenience, not a user-facing feature — the `legacy` group is removed before 2.0 ships.

### 31.5 Testing strategy

Each consolidated tool needs:
1. **Action dispatch tests:** Verify that each valid action routes correctly and that invalid actions produce the expected error.
2. **Behaviour tests:** Verify that each action's behaviour matches the 1.0 tool it replaces (the internal logic is unchanged; the test is that the new routing produces the same results).
3. **Side-effect tests:** Verify that cascading operations produce the correct side-effect entries.
4. **Batch tests:** For batch-capable tools, verify single-item, multi-item, partial-failure, and limit-exceeded cases.
5. **Integration tests:** End-to-end tests that exercise the `next` → `handoff` → `finish` workflow cycle.

### 31.6 Preserving internal architecture

The 2.0 tool handlers are a thin layer over the existing service types (`EntityService`, `PlanService`, `DocumentRecordService`, etc.). The handlers translate between the MCP tool input/output format and the service method signatures. No service logic should be duplicated in the handlers.

When a 2.0 tool consolidates multiple 1.0 tools that called different service methods, the 2.0 handler's action dispatch calls the same service methods the 1.0 handlers did.

---

## 32. Summary

Kanbanzai 2.0 redesigns the MCP tool surface from 97 entity-centric tools to 20 workflow-oriented tools organised in 7 feature groups.

| Group | Tools | Count | Always loaded? |
|-------|-------|-------|---------------|
| **Core** | `status`, `next`, `finish`, `handoff`, `entity`, `doc`, `health` | 7 | Yes |
| **Planning** | `decompose`, `estimate`, `conflict` | 3 | No |
| **Knowledge** | `knowledge`, `profile` | 2 | No |
| **Git** | `worktree`, `merge`, `pr`, `branch`, `cleanup` | 5 | No |
| **Documents** | `doc_intel` | 1 | No |
| **Incidents** | `incident` | 1 | No |
| **Checkpoints** | `checkpoint` | 1 | No |
| **Total** | | **20** | **7** |

The specification defines 11 implementation features:

| Feature | Name | Core deliverable |
|---------|------|-----------------|
| 1 | Feature Group Framework | Conditional tool registration, presets, configuration |
| 2 | Resource-Oriented Pattern | Action dispatch, side-effect reporting |
| 3 | Batch Operations | Array inputs, partial failure, batch responses |
| 4 | `status` | Synthesis dashboard for project/plan/feature/task |
| 5 | `next` | Claim + context assembly with invisible intelligence |
| 6 | `finish` | Completion + inline knowledge + lenient lifecycle |
| 7 | `handoff` | Sub-agent prompt generation |
| 8 | `entity` | Consolidated entity CRUD (replaces 17+ tools) |
| 9 | `doc` | Consolidated document operations (replaces 11+ tools) |
| 10 | Feature Group Tools | Consolidated remaining tools (13 tools) |
| 11 | 1.0 Removal | Remove 97 tools, update tests |

The internal architecture is unchanged. YAML-on-disk storage, lifecycle state machines, document-driven stage gates, knowledge confidence scoring, Git-native worktree isolation, and the service/storage layers carry forward from Phases 1–4b.

**Gate for completion:** All 30.x acceptance criteria verified, all 1.0 tools removed, `go test -race ./...` clean, a complete `next` → `handoff` → `finish` cycle demonstrated on a real task.