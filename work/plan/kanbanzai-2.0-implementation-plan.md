# Kanbanzai 2.0 Implementation Plan

| Document | Kanbanzai 2.0 Implementation Plan                      |
|----------|---------------------------------------------------------|
| Status   | Draft                                                   |
| Created  | 2026-03-27T01:42:41Z                                   |
| Updated  | 2026-03-27T01:42:41Z                                   |
| Related  | `work/spec/kanbanzai-2.0-specification.md`              |
|          | `work/design/kanbanzai-2.0-design-vision.md`            |
|          | `work/reports/mcp-server-design-issues.md`              |
|          | `work/plan/phase-4b-implementation-plan.md`             |

---

## 1. Overview

This document defines the implementation plan for Kanbanzai 2.0: the MCP tool surface redesign.

Kanbanzai 2.0 replaces 97 entity-centric MCP tools with 20 workflow-oriented tools organised in 7 feature groups. The internal architecture — YAML-on-disk storage, lifecycle state machines, document intelligence, knowledge management, and service layers — is unchanged. What changes is how the architecture is exposed to agents.

The work is organised into 11 implementation tracks:

| Track | Name | Core deliverable |
|-------|------|-----------------|
| A | Feature Group Framework | Conditional tool registration, presets, configuration |
| B | Resource-Oriented Pattern | Action dispatch mechanism, side-effect collector |
| C | Batch Operations | Array inputs, partial failure, batch response shape |
| D | `status` | Synthesis dashboard for project/plan/feature/task |
| E | `finish` | Completion + inline knowledge + lenient lifecycle |
| F | `next` | Claim + context assembly with invisible intelligence |
| G | `handoff` | Sub-agent prompt generation |
| H | `entity` | Consolidated entity CRUD (replaces 17+ tools) |
| I | `doc` | Consolidated document operations (replaces 11+ tools) |
| J | Feature Group Tools | Consolidated remaining tools (13 tools) |
| K | 1.0 Tool Removal | Remove 97 tools, update tests, final cutover |

The gate for completion: all spec §30 acceptance criteria verified, all 1.0 tools removed, `go test -race ./...` clean, a complete `next` → `handoff` → `finish` cycle demonstrated on a real task.

---

## 2. Pre-Implementation

Before any 2.0 track begins, verify the following prerequisites.

### 2.1 Codebase health

| Check | Command | Required outcome |
|-------|---------|-----------------|
| Build | `go build ./...` | Clean |
| Tests | `go test -race ./...` | All pass |
| Vet | `go vet ./...` | Clean |
| Health | `kbz health` | 0 errors |

### 2.2 Phase 4b completion

All Phase 4b acceptance criteria must be verified. The 1.0 tool surface is the starting point — we need confidence that everything works before we start replacing it.

### 2.3 Design vision approved

`work/design/kanbanzai-2.0-design-vision.md` must be in `approved` status. The specification references it as the design basis.

### 2.4 Specification approved

`work/spec/kanbanzai-2.0-specification.md` must be in `approved` status. It is the binding contract for all implementation work.

---

## 3. Implementation Strategy

### 3.1 Dependency structure

```
Track A: Feature Group Framework
    │
    ▼
Track B: Resource-Oriented Pattern + Side-Effect Reporting
    │
    ├──────────────────────────────────────────────┐
    │                                              │
    ▼                                              ▼
Track C: Batch Operations                     Track D: status
    │                                         (read-only, no
    │                                          batching needed)
    ├──────────────────┐
    │                  │
    ▼                  ▼
Track E: finish    Track H: entity
    │                  │
    │                  ▼
    │              Track I: doc
    │
    ▼
Track F: next
    │
    ▼
Track G: handoff
    │
    └──────────────────┐
                       │
                       ▼
                  Track J: Feature Group Tools
                  (parallelisable internally)
                       │
                       ▼
                  Track K: 1.0 Tool Removal
```

Key observations:

- **A → B → C** is the infrastructure chain. Everything else depends on it.
- **D** (`status`) is read-only and depends only on A+B (no batch, no side effects). It can start as soon as B is done.
- **E** (`finish`) validates the side-effect pipeline end-to-end. It should be built before F (`next`) because it is simpler and proves the pattern.
- **F** (`next`) is the most complex track — it wires together work queue, task claiming, document intelligence, and knowledge retrieval.
- **G** (`handoff`) shares the context assembly pipeline with F, so it follows F.
- **H** and **I** can be built in parallel with E/F/G — they depend on B+C but not on the workflow tools.
- **J** is a parallelisable consolidation pass. Each tool in the feature groups is independent.
- **K** is the final cutover. Nothing is removed until everything is built.

### 3.2 Dual registration during development

During development, both 1.0 and 2.0 tools coexist. The feature group framework (Track A) should treat the existing 1.0 tools as an implicit `_legacy` group that is enabled by default. As each 2.0 tool is completed and tested, the corresponding 1.0 tools can be disabled in the `_legacy` group. The `_legacy` group is removed entirely in Track K.

This allows incremental validation: at any point during development, both old and new tools work, and the test suite passes.

### 3.3 Side-effect collector pattern

The side-effect reporting mechanism (Track B) must be implemented as a request-scoped collector:

1. Each MCP request creates a fresh `SideEffectCollector`.
2. The collector is attached to the request context (`context.Context`).
3. Service methods that produce cascades (status transitions, document approvals, dependency unblocking) push side effects onto the collector via the context.
4. At response time, the MCP handler drains the collector into the response `side_effects` field.

This avoids threading side-effect arrays through every service method signature. It integrates with the existing `StatusTransitionHook` chain — the hooks push side effects onto the collector instead of (or in addition to) logging them.

### 3.4 Context assembly shared pipeline

`next` (Track F) and `handoff` (Track G) share the same context assembly pipeline. The difference is output format: `next` returns structured data, `handoff` renders a Markdown prompt.

The pipeline should be implemented as a new `internal/assembly/` package (or extended in `internal/context/`) with a single `AssembleTaskContext` function that returns a structured intermediate representation. `next` serialises this to YAML/JSON. `handoff` renders it to Markdown.

This shared pipeline is built once in Track F and reused in Track G.

---

## 4. Track A: Feature Group Framework ✓ COMPLETE

**Goal:** Build the configuration model and conditional tool registration mechanism that controls which tools the MCP server exposes to the client.

**Spec reference:** §6

**Dependencies:** None (foundational).

**Status:** Complete. All 12 tasks implemented. All 13 spec §30.1 acceptance criteria verified by passing tests. `go test -race ./...` clean.

| Task | Description | Size | Status |
|------|-------------|------|--------|
| A.1 | Define `MCPConfig` struct in `internal/config/` with `Groups` map and `Preset` string fields | S | ✓ |
| A.2 | Add `mcp` section parsing to `config.LoadOrDefault()`; parse both `groups` and `preset` | S | ✓ |
| A.3 | Implement preset resolution: `minimal`, `orchestration`, `full` → effective group map | S | ✓ |
| A.4 | Implement preset + explicit group override merging (explicit groups override preset values) | S | ✓ |
| A.5 | Implement validation: unknown preset → startup error; unknown group → startup warning; `core: false` → warning + override to `true` | M | ✓ |
| A.6 | Implement default behaviour: no `mcp` section → `preset: full` (all groups enabled) | S | ✓ |
| A.7 | Define `ToolGroup` constants and group membership map (spec §6.4) | S | ✓ |
| A.8 | Refactor `NewServer` in `internal/mcp/server.go` to accept effective group config and conditionally call `AddTools` based on group membership | M | ✓ |
| A.9 | Implement `_legacy` group containing all 1.0 tool registrations, enabled by default during development | M | ✓ |
| A.10 | Write tests: preset resolution, override merging, default behaviour, validation warnings/errors | M | ✓ |
| A.11 | Write test: `TestServer_ListTools_GroupConfig` — verify tool count varies with group configuration | M | ✓ |
| A.12 | Verify existing `TestServer_ListTools` still passes (all 1.0 tools registered when `_legacy` enabled) | S | ✓ |

**Test inventory:**

- `internal/config/groups_test.go` — unit tests for `EffectiveGroups`: default-full, all presets, unknown preset error, explicit overrides (enable/disable), core-false warning, unknown group warning, empty preset, multiple overrides
- `internal/mcp/groups_test.go` — unit tests for `resolveServerGroups` and `GroupToolNames`: membership map completeness, non-overlapping groups, total tool count, all presets, legacy always enabled, explicit override, core cannot be disabled
- `internal/mcp/server_groups_test.go` — integration tests via `newServerWithConfig` + `MCPServer.ListTools()`: group-config table test (5 cases), core-group conditional, legacy always enabled across all presets
- `internal/config/config_test.go` — YAML parse and round-trip tests for `mcp` section (`TestConfig_MCPConfigParseFromYAML`, `TestConfig_MCPConfigRoundTrip`)

**Key implementation notes:**

- The group membership map (A.7) defines which tool names belong to which group. During Track A, the map is populated with 2.0 tool names even though the tool implementations do not yet exist. The infrastructure is ready before the tools are.
- A.8 refactored `NewServer` into a public wrapper and an internal `newServerWithConfig(entityRoot string, cfg *config.Config)`. The internal function is used by integration tests (package `mcp`) to inject test configurations without touching the filesystem.
- The `_legacy` group (A.9) wraps all existing 1.0 tool registrations. This is a development convenience — it keeps the test suite passing while 2.0 tools are being built. It is not user-facing.

**Verification (spec §30.1):** All 13 acceptance criteria have passing tests.

---

## 5. Track B: Resource-Oriented Pattern & Side-Effect Reporting ✓ COMPLETE

**Goal:** Build the action-dispatch framework and the side-effect reporting contract that all consolidated tools use.

**Spec reference:** §7, §8

**Dependencies:** Track A (tools register through the group framework).

**Status:** Complete. All 18 tasks implemented. 9 of 10 spec §30.2 acceptance criteria verified by passing tests. `go test -race ./...` clean. The remaining criterion (document-approval cascade as side effect) is scaffolded at the service layer and its end-to-end test is deferred to Track I — see B.9/B.15 note below.

| Task | Description | Size | Status |
|------|-------------|------|--------|
| B.1 | Define `SideEffect` struct: `Type`, `EntityID`, `EntityType`, `FromStatus`, `ToStatus`, `Trigger` | S | ✓ |
| B.2 | Define `SideEffectCollector` type with `Push(SideEffect)` and `Drain() []SideEffect` methods | S | ✓ |
| B.3 | Implement context key for `SideEffectCollector`; helper functions `CollectorFromContext(ctx)` and `PushSideEffect(ctx, effect)` | S | ✓ |
| B.4 | Wire collector creation into the MCP request handling path: create collector at request start, attach to context | M | ✓ |
| B.5 | Wire collector draining into MCP response path: after handler returns, drain collector and append `side_effects` to response | M | ✓ |
| B.6 | Integrate with `StatusTransitionHook` chain: hooks push `status_transition` side effects when they fire cascades | M | ✓ |
| B.7 | Integrate with `DependencyUnblockingHook`: push `task_unblocked` side effects | S | ✓ |
| B.8 | Integrate with `WorktreeTransitionHook`: push `worktree_created` side effects | S | ✓ |
| B.9 | Integrate with `EntityLifecycleHook` (document approval cascades): push `status_transition` side effects when doc approval advances a feature | M | ✓ (scaffolded; end-to-end in Track I — see note) |
| B.10 | Define `ActionDispatcher` pattern: a helper that routes `action` parameter to handler functions within a single tool | M | ✓ |
| B.11 | Implement standard error response shape: `ErrorResponse{Code, Message, Details}` | S | ✓ |
| B.12 | Implement unknown-action error: returns error listing valid actions for the tool | S | ✓ |
| B.13 | Implement irrelevant-parameter ignoring: parameters not needed by the current action are silently ignored | S | ✓ |
| B.14 | Write tests: collector lifecycle (create, push, drain, empty-after-drain) | S | ✓ |
| B.15 | Write tests: side-effect integration — document approval triggers feature transition side effect | M | ✓ (service layer; end-to-end in Track I — see note) |
| B.16 | Write tests: side-effect integration — task completion triggers dependency unblocking side effect | M | ✓ |
| B.17 | Write tests: action dispatcher routing, unknown action error, error response shape | M | ✓ |
| B.18 | Write tests: read-only operations do not include `side_effects` field | S | ✓ |

**Test inventory:**

- `internal/mcp/sideeffect.go` — core infrastructure: `SideEffect`, `SideEffectCollector`, `WithSideEffects` middleware, `DispatchAction`, `ActionError`, `buildResult`
- `internal/mcp/sideeffect_test.go` — unit tests: collector lifecycle (push, drain, empty-after-drain, concurrent push), context helpers (round-trip, nil-safe noop), `WithSideEffects` middleware (no effects, single, multiple, per-request isolation, mutation/read-only flag), `DispatchAction` (routing, unknown action, missing action, sorted listing, irrelevant params ignored), `ActionError` shape (with and without details), `buildResult` (inject into object, nil, non-object envelope, mutation empty array)
- `internal/mcp/batch.go` — batch infrastructure: `ExecuteBatch`, `IsBatchInput`, `BatchResult`, `ItemResult`
- `internal/mcp/batch_test.go` — unit tests: single item, multiple items, partial failure, all fail, limit exceeded, exactly at limit, empty batch, side effects aggregated, side effects absent when empty, input order preserved, JSON shape, `IsBatchInput` variants
- `internal/service/documents_test.go` — service-layer tests for `DocEntityTransition`: approval reports transition, no transition when already at target status, supersession reports backward transition
- `internal/service/dependency_hook_test.go` — `TestDependencyUnblockingHook_PreviousStatusRecorded`: verifies `UnblockedTask.PreviousStatus` is populated for use in `from_status` of side effects

**B.9 / B.15 deferral note:**

The `EntityLifecycleHook` interface (used by `DocumentService`) does not carry a `context.Context`, so it cannot push side effects directly onto the collector. Rather than change the interface (which would cascade through all callers), the service layer was scaffolded instead: `DocumentResult` now carries a `DocEntityTransition` field that records any entity lifecycle transition triggered by approval or supersession. Track I's `doc(action: "approve")` handler reads this field and pushes the `SideEffectStatusTransition` side effect. Tests for the service-layer plumbing (`TestApproveDocument_ReportsEntityTransition`, `TestApproveDocument_NoEntityTransition_WhenAlreadyAtTargetStatus`, `TestSupersedeDocument_ReportsEntityTransition`) are in `internal/service/documents_test.go`. The end-to-end MCP-layer test (B.15 proper) must be written in Track I once `doc(approve)` exists.

**Key implementation notes:**

- The `SideEffectCollector` must be goroutine-safe for correctness, even though the current server is single-process. Use a simple mutex-protected slice.
- B.4 and B.5 are implemented as the `WithSideEffects` middleware wrapper in `internal/mcp/sideeffect.go`. Each 2.0 tool handler is wrapped with `WithSideEffects(func(...) (any, error) { ... })`. The middleware creates a fresh collector per request, attaches it to the context, calls the inner handler, drains the collector, and injects `side_effects` into the JSON response.
- B.6–B.8 are implemented at the MCP tool layer: after a status-transition service call returns a `WorktreeResult`, the tool handler iterates `wt.UnblockedTasks` and `wt.Created` and calls `PushSideEffect`. The hooks themselves do not carry context and do not push directly — the MCP tool is the integration point.
- B.9 is scaffolded via `DocEntityTransition` on `DocumentResult` (see deferral note above).
- B.10 (`ActionDispatcher`) is implemented as `DispatchAction(ctx, req, map[string]ActionHandler{...})`. It is not a required abstraction but ensures all tools use an identical dispatch pattern.
- The `SideEffect` struct includes an `Extra map[string]string` field (not in the spec schema) to carry type-specific details such as worktree path/branch for `worktree_created` events. This is an additive extension.

**Verification (spec §30.2):** 9 of 10 acceptance criteria have passing tests. Criterion 7 (document approval cascade) is scaffolded at the service layer; end-to-end coverage is a Track I prerequisite.

---

## 6. Track C: Batch Operations ✓ COMPLETE (post-review remediation applied)

**Goal:** Build the array-accepting input pattern, partial-failure execution, and batch response shape.

**Spec reference:** §9

**Dependencies:** Track B (batch responses include side effects).

**Status:** Complete. All 12 tasks implemented. All 8 spec §30.3 acceptance criteria verified by passing tests. `go test -race ./...` clean.

**Review findings (2026-03-27):** A post-implementation review (`work/reviews/track-c-batch-operations-review.md`) identified 4 blocking findings. All 4 were remediated in the same session:

- **F1** (`batch_limit_exceeded` error code): `ExecuteBatch` now returns a typed `*BatchLimitError`; `WithSideEffects` detects it and emits `code: "batch_limit_exceeded"` instead of `"internal_error"`.
- **F2** (duplicate `side_effects` key): `buildResult` now type-checks for `*BatchResult` and returns it directly, injecting `side_effects: []` only when the field is absent (no longer injects into a response that already carries the field).
- **F3** (`finish` missing `SignalMutation`): `finish_tool.go` now calls `SignalMutation(ctx)` so `side_effects: []` is always present in single-mode responses (spec §8.4).
- **F4** (`estimate(set)` non-standard batch): `estimateSetBatch` was replaced with `ExecuteBatch`; response shape now matches §9.4 and the 100-item limit is enforced.

| Task | Description | Size | Status |
|------|-------------|------|--------|
| C.1 | Define `BatchResult` response struct: `Results []ItemResult`, `Summary {Total, Succeeded, Failed}`, `SideEffects []SideEffect` | S | ✓ |
| C.2 | Define `ItemResult` struct: `ItemID`, `Status` ("ok"/"error"), `Data`, `Error`, `SideEffects` | S | ✓ |
| C.3 | Implement input detection: determine whether the caller provided single-item parameters or a batch array parameter | M | ✓ |
| C.4 | Implement batch execution loop: iterate items, execute each independently, collect per-item results, aggregate side effects | M | ✓ |
| C.5 | Implement partial failure semantics: a failure on item N does not prevent processing of item N+1 | S | ✓ |
| C.6 | Implement batch limit enforcement: reject batches >100 items with `batch_limit_exceeded` error before processing | S | ✓ |
| C.7 | Implement response shape switching: single-item calls return the single-item shape; batch calls return the `BatchResult` shape | M | ✓ |
| C.8 | Write tests: single-item call returns single-item response (not wrapped in batch) | S | ✓ |
| C.9 | Write tests: batch call returns batch response with per-item results | M | ✓ |
| C.10 | Write tests: partial failure — one item fails, others succeed, summary counts are correct | M | ✓ |
| C.11 | Write tests: batch limit exceeded returns error before any processing | S | ✓ |
| C.12 | Write tests: aggregate side effects are union of per-item side effects | S | ✓ |

**Test inventory:**

- `internal/mcp/batch.go` — `ExecuteBatch`, `IsBatchInput`, `BatchResult`, `ItemResult`, `BatchSummary`
- `internal/mcp/batch_test.go` — unit tests: single item, multiple items, partial failure (one fails, rest continue), all fail, limit exceeded (handler not called), exactly at limit, empty batch, side effects aggregated per-item and in top-level union, side effects absent when empty, input order preserved, JSON shape (`results`/`summary`/`side_effects` fields), `IsBatchInput` variants (true, missing key, non-array value, nil args, empty array)

**Key implementation notes:**

- C.3 is implemented as `IsBatchInput(args, batchKey string) bool` in `batch.go`. Each batch-capable tool calls this to determine whether to invoke `ExecuteBatch` or the single-item path. When both single and batch parameters are provided, batch takes precedence by checking `IsBatchInput` first.
- C.4 is implemented as `ExecuteBatch(ctx, items []any, handler BatchItemHandler) (any, error)`. Each tool's batch-capable action calls this helper. Per-item side effects are captured in a sub-collector and attributed to the individual `ItemResult`; they are also aggregated into the top-level `BatchResult.SideEffects`.
- The batch infrastructure is used by: `finish` (Track E), `entity(create)` (Track H), `doc(register)` and `doc(approve)` (Track I), and `estimate(set)` (Track J). Note: `estimate(set)` originally had a custom batch loop; it was migrated to `ExecuteBatch` as part of review remediation F4.

**Verification (spec §30.3):** All 8 acceptance criteria have passing tests. Additional tests added during review remediation: `TestWithSideEffects_BatchLimitError_ProducesCorrectCode`, `TestBuildResult_BatchResult_SideEffectsNotDoubled`, `TestBuildResult_BatchResult_MutationNoEffects_SideEffectsPresent`, `TestDocTool_Approve_Batch_WithEntityTransition`.

---

## 7. Track D: `status` — Synthesis Dashboard ✓ COMPLETE (post-review remediation applied)

**Goal:** Build the `status` tool that synthesises project, plan, feature, and task state into concise dashboards.

**Spec reference:** §10

**Dependencies:** Track A (tool registration). Does not require Tracks B–C (read-only, no side effects or batching).

| Task | Description | Size |
|------|-------------|------|
| D.1 | Implement ID type inference: determine entity type from ID format (plan prefix, `FEAT-`, `TASK-`, `BUG-`) | M |
| D.2 | Implement project overview synthesis: load all plans, compute per-plan summary, aggregate health, generate `attention` items | M |
| D.3 | Implement plan dashboard synthesis: load plan features, compute per-feature task rollup, include document gaps, generate `attention` items | M |
| D.4 | Implement feature detail synthesis: load tasks with status breakdown, load documents, compute estimate progress, load worktree state | M |
| D.5 | Implement task detail synthesis: load task with parent feature context, resolve dependencies with blocking flags | M |
| D.6 | Implement `attention` item generation: heuristic rules — stalled tasks, missing docs, ready tasks, health warnings | M |
| D.7 | Implement `status` MCP tool wiring in `internal/mcp/status_tool.go`: register in core group, route by ID type | M |
| D.8 | Write tests: project overview with multiple plans, various statuses | M |
| D.9 | Write tests: plan dashboard with features in different states, document gaps | M |
| D.10 | Write tests: feature detail with task breakdown, estimate rollup | M |
| D.11 | Write tests: task detail with dependencies | S |
| D.12 | Write tests: unknown ID format returns clear error | S |
| D.13 | Write tests: entity not found returns clear error | S |

**Key implementation notes:**

- The synthesis layer (D.2–D.5) is new code — there is no 1.0 equivalent. It aggregates data from multiple service calls into a single response. Implement it as a `StatusService` in `internal/service/` that coordinates `EntityService`, `DocumentRecordService`, and health check logic.
- D.6 generates human-readable attention strings. These are curated (not exhaustive) — pick the top 3–5 most actionable items. Rules: tasks stalled >3 days, features with missing specs or dev-plans, tasks ready for dispatch, health warnings.
- `status` is the tool agents will call most often. Keep the response compact — return counts and summaries, not full entity records.

**Verification (spec §30.4):** All 10 acceptance criteria must have passing tests.

**Status:** Complete. All 13 tasks implemented. All 10 spec §30.4 acceptance criteria verified by passing tests. `go test -race ./...` clean.

Post-review remediation (applied during Track D review):
- **Bug fix:** `synthesiseTask` read `pf.State["owner"]` instead of `pf.State["parent"]` for the parent feature's plan ID, causing `parent_feature.plan_id` to always be empty. Fixed.
- **Missing §30.4 criterion 8:** Health summary (`errors`, `warnings`) was not included in project overview or plan dashboard. Added `health` field populated by `entitySvc.HealthCheck()` to both.
- **Missing §30.4 criterion 3:** Worktree info was not included in feature detail. Added `worktree` field; `StatusTools` now accepts a `*worktree.Store` parameter. Server wiring updated.
- **Missing §10.6 dispatch info:** Task detail had no `dispatch` block. Added `dispatch` field populated from `dispatched_to`, `dispatched_at`, `dispatched_by` task state fields.
- **Renamed `featureInfo.Owner` → `featureInfo.PlanID`** with JSON tag `plan_id` to match §10.6 spec naming.
- **Added tests** for all four remediation items (health in project/plan, worktree in feature, dispatch in task, plan_id correctness).

**Known response-shape divergences from spec examples** (not blocking; flagged for human decision):
The §10.3–§10.6 YAML examples show nested structures (e.g., `plans.{total,by_status,active_plans}`, `features.{total,by_status,items[]}`, `dependencies.{total,blocking,items[]}`) that the implementation flattens for compactness. Specifically:
- Project overview: `plans` is a flat array + `total` aggregate rather than `{total,by_status,active_plans}`; top-level `active_tasks`/`ready_tasks`/`blocked_tasks` are folded into `total.tasks`.
- Plan dashboard: `features` is a flat array rather than `{total,by_status,items[]}`; `doc_gaps` is a string array rather than `[{feature_id,missing:[]}]`; per-feature `estimate` and `blockers` fields are omitted.
- Feature detail: `estimate` is a scalar rather than `{total,progress,delta}`; `tasks` is a flat array rather than `{by_status,items[]}`.
- Task detail: `dependencies` is a flat array rather than `{total,blocking,items[]}`; `parent_feature` is a separate top-level field rather than nested inside `task`.
These are pragmatic simplifications. Revisit if agents need the richer structure.

---

## 8. Track E: `finish` — Completion & Inline Knowledge

**Goal:** Build the `finish` tool with inline knowledge contribution, lenient lifecycle, and side-effect reporting. This track validates the side-effect pipeline end-to-end before the more complex `next` tool.

**Spec reference:** §12

**Dependencies:** Track B (side effects), Track C (batch).

| Task | Description | Size |
|------|-------------|------|
| E.1 | Define `FinishInput` struct: `TaskID`, `Summary`, `FilesModified`, `Verification`, `Knowledge`, `ToStatus` | S |
| E.2 | Define `FinishResult` struct: `Task`, `Knowledge` (accepted/rejected), `SideEffects` | S |
| E.3 | Implement single-task completion: transition task to `done` or `needs-review`, set `completed` timestamp and `completion_summary` | M |
| E.4 | Implement lenient lifecycle: accept tasks in `ready` status, internally transition through `active` before completing | M |
| E.5 | Implement inline knowledge contribution: process `knowledge` array through existing `KnowledgeService.Contribute`, collect accepted/rejected results | M |
| E.6 | Implement `knowledge_contributed` and `knowledge_rejected` side effects for inline contributions | S |
| E.7 | Wire dependency unblocking: after completion, the existing `DependencyUnblockingHook` fires and pushes `task_unblocked` side effects via the collector | S |
| E.8 | Implement batch completion using the Track C batch infrastructure | M |
| E.9 | Implement `finish` MCP tool wiring in `internal/mcp/finish_tool.go`: register in core group, handle single and batch modes | M |
| E.10 | Write tests: single completion — task transitions to `done`, timestamp set, summary stored | M |
| E.11 | Write tests: lenient lifecycle — `ready` task is accepted, transitions through to `done` | S |
| E.12 | Write tests: `needs-review` target status | S |
| E.13 | Write tests: inline knowledge — valid entry accepted, duplicate rejected, task still completes | M |
| E.14 | Write tests: side effects — unblocked tasks appear in response | M |
| E.15 | Write tests: batch completion — partial failure, per-item results, aggregate side effects | M |
| E.16 | Write tests: error cases — task not found, task in terminal status, missing summary | S |

**Key implementation notes:**

- E.3 wraps the existing `DispatchService.CompleteTask` or calls `EntityService.UpdateStatus` directly. The existing service logic is reused — `finish` is a thin orchestration layer over it.
- E.4 (lenient lifecycle) is the key UX improvement. In 1.0, completing a task that was never formally dispatched required two calls (`dispatch_task` + `complete_task`). In 2.0, `finish` handles it in one call. The internal transition sequence is: `ready` → `active` → `done`. Both transitions fire their hooks (worktree creation on `active`, dependency unblocking on `done`).
- E.7 validates that the side-effect collector (Track B) correctly captures side effects from hooks. This is the first end-to-end test of the collector pattern.

**Verification (spec §30.6):** All 11 acceptance criteria must have passing tests.

---

## 9. Track F: `next` — Claim & Context Assembly

**Goal:** Build the `next` tool that combines work queue inspection, task claiming, and context assembly with invisible document intelligence and knowledge integration. This is the most complex track.

**Spec reference:** §11

**Dependencies:** Track B (side effects for claiming and promotion). Track E should be complete first (proves side-effect pattern).

| Task | Description | Size |
|------|-------------|------|
| F.1 | Implement queue inspection mode: reuse existing `work_queue` promotion logic, return ready queue with side effects for promoted tasks | M |
| F.2 | Implement queue sort: estimate ascending (null last), age descending, ID lexicographic | S |
| F.3 | Implement optional `role` filter in queue mode | S |
| F.4 | Implement optional `conflict_check` annotation in queue mode (reuses `ConflictService`) | S |
| F.5 | Implement claim-by-task-ID: verify `ready` status, transition to `active`, set dispatch fields | M |
| F.6 | Implement claim-by-feature-ID: find top ready task in feature, claim it | M |
| F.7 | Implement claim-by-plan-ID: find top ready task across all features in plan, claim it | M |
| F.8 | Implement "already claimed" error path (task in `active` status) matching Phase 4a `dispatch_task` behaviour | S |
| F.9 | **Build shared context assembly pipeline** in `internal/assembly/`: define `TaskContext` struct with `SpecSections`, `AcceptanceCriteria`, `Knowledge`, `FilesContext`, `Constraints`, `Trimmed` | M |
| F.10 | Implement spec section extraction: load parent feature's spec document, identify relevant sections via document intelligence Layer 3 classifications | L |
| F.11 | Implement spec section fallback: when Layer 3 classification is not available, extract all sections that reference the parent feature | M |
| F.12 | Implement acceptance criteria extraction: identify testable criteria from spec by section role or heuristic pattern matching | M |
| F.13 | Implement knowledge retrieval: query `KnowledgeService` for entries matching scope, tags, and topic; sort by confidence; fit within byte budget | M |
| F.14 | Implement file context: include `files_planned` from task; fall back to recent worktree modifications if absent | S |
| F.15 | Implement byte-budget trimming: trim in order (lowest-confidence Tier 3 → Tier 2 → design context → spec sections); report trimmed entries | M |
| F.16 | Implement automatic Layer 1–2 parse: when a spec document has no index, trigger synchronous parse before assembly | M |
| F.17 | Implement `next` MCP tool wiring in `internal/mcp/next_tool.go`: register in core group, route by input (no id → queue, id → claim) | M |
| F.18 | Write tests: queue inspection — promoted tasks appear in side effects, sort order correct | M |
| F.19 | Write tests: queue with role filter | S |
| F.20 | Write tests: claim by task ID — transitions to active, dispatch fields set, context returned | M |
| F.21 | Write tests: claim by plan ID — picks top ready task across features | M |
| F.22 | Write tests: claim by feature ID with no ready tasks returns error | S |
| F.23 | Write tests: context includes spec sections when intelligence index available | M |
| F.24 | Write tests: context falls back to document path when index not available | S |
| F.25 | Write tests: knowledge entries included, sorted by confidence, within byte budget | M |
| F.26 | Write tests: trimmed entries reported correctly | S |
| F.27 | Write tests: automatic Layer 1–2 parse triggered for unindexed spec | M |
| F.28 | Write tests: already-claimed error returns dispatch metadata | S |

**Key implementation notes:**

- F.9 is the foundation. The `TaskContext` struct is the intermediate representation shared between `next` and `handoff`. Build it carefully — it needs to be rich enough for both structured output and Markdown rendering.
- F.10–F.12 wire the existing `internal/docint/` infrastructure into the assembly pipeline. The `IntelligenceService` already supports section extraction, entity reference lookup, and role-based search. The work is in composing these existing capabilities into a coherent extraction flow, not building new intelligence.
- F.16 (automatic parsing) means the first `next` call on a feature with an unindexed spec triggers a synchronous `IntelligenceService.IndexDocument` call. This may take 1–2 seconds. Subsequent calls use the cached index. Test that this works correctly and that the parse result is persisted.
- The queue inspection mode (F.1–F.4) is largely a re-packaging of the existing `work_queue` logic from Phase 4a. The claiming mode (F.5–F.8) re-packages the existing `dispatch_task` logic. The context assembly (F.9–F.16) is the genuinely new work.

**Verification (spec §30.5):** All 16 acceptance criteria must have passing tests.

---

## 10. Track G: `handoff` — Sub-Agent Prompt Generation

**Goal:** Build the `handoff` tool that renders a complete sub-agent prompt from a task's assembled context.

**Spec reference:** §13

**Dependencies:** Track F (shares the context assembly pipeline).

| Task | Description | Size |
|------|-------------|------|
| G.1 | Implement Markdown prompt renderer: takes `TaskContext` (from F.9) and renders a formatted prompt string | M |
| G.2 | Define prompt template: task summary → spec sections → acceptance criteria → known constraints → files → conventions | M |
| G.3 | Implement `instructions` parameter: insert orchestrator-provided additional instructions into the prompt | S |
| G.4 | Implement `context_metadata` response: byte usage, sections included, trimmed entries | S |
| G.5 | Implement lenient lifecycle: accept tasks in `active`, `ready`, or `needs-rework` status; reject terminal status | S |
| G.6 | Implement `handoff` MCP tool wiring in `internal/mcp/handoff_tool.go`: register in core group | M |
| G.7 | Write tests: prompt contains all expected sections (summary, spec, criteria, knowledge, files, conventions) | M |
| G.8 | Write tests: `instructions` parameter included in prompt | S |
| G.9 | Write tests: context metadata reports byte usage and trimmed entries | S |
| G.10 | Write tests: handoff on terminal-status task returns error | S |
| G.11 | Write tests: handoff does not modify task status (read-only) | S |
| G.12 | Integration test: `next(task_id)` → `handoff(task_id)` — both use shared pipeline, context is consistent | M |

**Key implementation notes:**

- G.1–G.2 is pure rendering. The hard work of context assembly was done in Track F. `handoff` calls the same `AssembleTaskContext` pipeline and passes the result through a Markdown template.
- The prompt format (G.2) should be designed for LLM consumption: clear section headers, bullet lists for criteria, code blocks for file paths. Test with real prompt lengths to verify it fits in context windows.
- G.5 makes `handoff` useful for rework cycles: the orchestrator can re-generate a prompt for a task that was reviewed and needs rework, including updated knowledge from the first attempt.

**Verification (spec §30.7):** All 9 acceptance criteria must have passing tests.

---

## 11. Track H: `entity` — Consolidated Entity CRUD ✓ COMPLETE

**Goal:** Consolidate 17+ entity-specific tools into one resource-oriented tool with action dispatch.

**Status:** Complete. All 17 tasks implemented. All 15 spec §30.8 acceptance criteria verified by passing tests. `go test -race ./...` clean.

**Implementation:** `internal/mcp/entity_tool.go`, `internal/mcp/entity_tool_test.go`

**Spec reference:** §14

**Dependencies:** Track B (action dispatch, side effects), Track C (batch create).

| Task | Description | Size | Status |
|------|-------------|------|--------|
| H.1 | Implement `entity(action: "create")` for all entity types: task, feature, plan, bug, epic, decision | M | ✓ |
| H.2 | Implement batch create: accept `entities` array, use Track C batch infrastructure | M | ✓ |
| H.3 | Implement `entity(action: "get")` with type inference from ID prefix | M | ✓ |
| H.4 | Implement `entity(action: "list")` with filters: type, parent, status, tags, date ranges | M | ✓ |
| H.5 | Implement summary record format for `list` response (id, type, slug, status, summary — not full records) | S | ✓ |
| H.6 | Implement `entity(action: "update")` — update fields, silently ignores `id` and `status` | M | ✓ |
| H.7 | Implement `entity(action: "transition")` — lifecycle status change with side-effect reporting | M | ✓ |
| H.8 | Implement type inference from ID prefix (spec §14.8): `FEAT-` → feature, `TASK-`/`T-` → task, etc. | S | ✓ |
| H.9 | Implement `entity` MCP tool wiring in `internal/mcp/entity_tool.go`: register in core group, action dispatch | M | ✓ |
| H.10 | Wire duplicate check into `create` action (advisory, non-blocking) | S | ✓ |
| H.11 | Write tests: create single entity of each type | M | ✓ |
| H.12 | Write tests: batch create — multiple tasks in one call | M | ✓ |
| H.13 | Write tests: get with type inference from ID prefix | M | ✓ |
| H.14 | Write tests: list with various filters, verify summary record format | M | ✓ |
| H.15 | Write tests: update does not change `id` or `status` (silently ignored) | S | ✓ |
| H.16 | Write tests: transition with valid and invalid status changes, side effects reported | M | ✓ |
| H.17 | Write tests: transition error includes current status and valid transitions | S | ✓ |

**Key implementation notes:**

- H.1–H.7 each call through to the existing `EntityService` and `PlanService` methods. No service logic is duplicated — the `entity` tool is a routing layer.
- H.3 uses type inference to avoid requiring a `type` parameter for `get`, `update`, and `transition`. This is a UX improvement — agents know the ID, they shouldn't need to also specify the type.
- H.5 (summary records) is important for keeping `list` responses compact. Full entity records can be thousands of tokens. Summary records are ~50 tokens each.
- H.6: `update` silently ignores `id` and `status` fields (consistent with 1.0 `update_entity` behaviour). Use `transition` for status changes.
- H.7: transition errors are enriched with `current_status` and `valid_transitions` in the `details` field, giving agents the context needed to correct the call. A best-effort `Get` is performed on the error path to retrieve current state.
- H.7 also triggers the Track B mutation signalling fix: mutation actions (`create`, `update`, `transition`) call `SignalMutation(ctx)` so `side_effects: []` is always present in mutation responses (spec §8.4). This fix was applied to `sideeffect.go` as a Track B infrastructure improvement.
- Plan creation (`type: "plan"`) requires a valid prefix in `.kbz/config.yaml`. The corresponding test (`TestEntity_Create_Plan`) skips when no config is present, matching the pattern used throughout the service layer.

**Verification (spec §30.8):** All 15 acceptance criteria verified. See `entity_tool_test.go`.

---

## 12. Track I: `doc` — Consolidated Document Operations ✓ COMPLETE

**Goal:** Consolidate 11+ document tools into one resource-oriented tool.

**Spec reference:** §15

**Dependencies:** Track B (action dispatch, side effects), Track C (batch register/approve).

| Task | Description | Size | Status |
|------|-------------|------|--------|
| I.1 | Implement `doc(action: "register")` — single document registration | M | ✓ |
| I.2 | Implement batch register: accept `documents` array | M | ✓ |
| I.3 | Implement `doc(action: "approve")` — single and batch (with `ids` array) | M | ✓ |
| I.4 | Wire approval side-effect reporting: feature transitions from doc approval appear in `side_effects` | M | ✓ |
| I.5 | Implement `doc(action: "get")` — by ID or by path (resolve path to document record) | M | ✓ |
| I.6 | Implement `doc(action: "content")` — return file content with drift detection | S | ✓ |
| I.7 | Implement `doc(action: "list")` — with type, status, owner, and pending filters | M | ✓ |
| I.8 | Implement `doc(action: "gaps")` — missing document analysis for a feature | S | ✓ |
| I.9 | Implement `doc(action: "import")` — batch import from directory (reuses existing `batch_import_documents` logic) | S | ✓ |
| I.10 | Implement `doc(action: "validate")` — document record validation | S | ✓ |
| I.11 | Implement `doc(action: "supersede")` — mark document as superseded with side effects | S | ✓ |
| I.12 | Implement `doc` MCP tool wiring in `internal/mcp/doc_tool.go`: register in core group, action dispatch | M | ✓ |
| I.13 | Write tests: register single and batch | M | ✓ |
| I.14 | Write tests: approve with side effects (feature transition) | M | ✓ |
| I.15 | Write tests: get by ID and get by path | S | ✓ |
| I.16 | Write tests: list with filters | S | ✓ |
| I.17 | Write tests: gaps analysis | S | ✓ |
| I.18 | Write tests: import idempotency | S | ✓ |

**Key implementation notes:**

- I.4 is the canonical side-effect demonstration: approving a spec triggers a feature transition, and the response tells the agent what happened. Test this thoroughly.
- I.5 path-based lookup is a UX improvement. Agents know file paths (`work/spec/foo.md`) but often don't know the document record ID (`DOC-01JX...`). The `get` action searches for a document record with a matching `path` field.

**Status:** Complete. All 18 tasks implemented. All 13 spec §30.9 acceptance criteria have passing tests. `go test -race ./...` clean.

**Completion notes:**

- `gaps` action implements richer status-aware analysis than the 1.0 `doc_gaps` tool: it distinguishes missing, draft, and approved document types, matching the spec §15.8 response shape exactly. The `IntelligenceService.AnalyzeGaps` helper (which only reports fully missing types) is bypassed in favour of a direct `ListDocumentsByOwner` call.
- `approve` and `supersede` both push `SideEffectStatusTransition` side effects when `result.EntityTransition` is populated, completing Track B task B.9/B.15. End-to-end MCP tests are in `TestDocTool_Approve_ReportsEntityTransition` and `TestDocTool_Supersede_ReportsEntityTransition`.
- `DocTool` accepts `intelligenceSvc` in its signature (so `server.go` need not change when a future action requires it), but does not forward it to the unexported `docTool` function. No current action needs it.
- `doc_supersession_chain` is listed in spec §15.1 as a replaced tool but does not appear in the §15.2 action table and has no acceptance criterion in §30.9. It is not implemented; if needed it can be added as a `supersession_chain` action in a future track.

**Verification (spec §30.9):** All 13 acceptance criteria have passing tests.

---

## 13. Track J: Feature Group Tools

**Goal:** Consolidate the remaining 13 feature group tools from their 1.0 multi-tool forms into 2.0 action-parameter tools.

**Spec reference:** §17–§22

**Dependencies:** Track A (group registration), Track B (action dispatch). Track C for batch-capable tools.

This track is heavily parallelisable — each tool is independent.

### 13.1 Planning group

| Task | Description | Size |
|------|-------------|------|
| J.1 | Implement `decompose` tool: `propose`, `review`, `apply`, `slice` actions — wraps existing `DecomposeService` and `SliceAnalysis` | M |
| J.2 | Implement `decompose(action: "apply")` — create tasks from proposal in one call (new capability) | M |
| J.3 | Implement `estimate` tool: `set`, `query`, `add_reference`, `remove_reference` actions — wraps existing estimation tools | M |
| J.4 | Implement batch `estimate(action: "set")` with `entities` array | S |
| J.5 | Implement `conflict` tool: wraps existing `ConflictService` | S |
| J.6 | Write tests for planning group tools | M |

### 13.2 Knowledge group

| Task | Description | Size |
|------|-------------|------|
| J.7 | Implement `knowledge` tool: 12 actions wrapping existing `KnowledgeService` methods | M |
| J.8 | Implement `profile` tool: `list`, `get` actions wrapping existing `ProfileStore` | S |
| J.9 | Write tests for knowledge group tools | M |

### 13.3 Git group

| Task | Description | Size |
|------|-------------|------|
| J.10 | Implement `worktree` tool: `create`, `get`, `list`, `remove` actions | M |
| J.11 | Implement `merge` tool: `check`, `execute` actions | S |
| J.12 | Implement `pr` tool: `create`, `status`, `update` actions | S |
| J.13 | Implement `branch` tool: `status` action | S |
| J.14 | Implement `cleanup` tool: `list`, `execute` actions | S |
| J.15 | Write tests for git group tools | M |

### 13.4 Documents group

| Task | Description | Size |
|------|-------------|------|
| J.16 | Implement `doc_intel` tool: `outline`, `section`, `classify`, `find`, `trace`, `impact`, `guide`, `pending` actions | M |
| J.17 | Implement `doc_intel(action: "find")` routing: `concept`, `entity_id`, or `role` parameter determines search type | S |
| J.18 | Write tests for doc_intel tool | M |

### 13.5 Incidents group

| Task | Description | Size |
|------|-------------|------|
| J.19 | Implement `incident` tool: `create`, `update`, `list`, `link_bug` actions | M |
| J.20 | Write tests for incident tool | S |

### 13.6 Checkpoints group

| Task | Description | Size |
|------|-------------|------|
| J.21 | Implement `checkpoint` tool: `create`, `get`, `respond`, `list` actions | M |
| J.22 | Write tests for checkpoint tool | S |

**Key implementation notes:**

- Every tool in this track wraps existing service logic. No service changes are needed. The work is routing, parameter mapping, and response formatting.
- J.2 (`decompose apply`) is the only genuinely new capability in this track — it creates tasks from a proposal in one call instead of requiring N separate `create_task` calls. It iterates the proposal, creates each task, then resolves `depends_on` slugs to the newly created task IDs.
- This track can be split across multiple agents. Each group (planning, knowledge, git, documents, incidents, checkpoints) is completely independent.

**Verification (spec §30.10):** All 28 acceptance criteria must have passing tests.

---

## 14. Track K: 1.0 Tool Removal

**Goal:** Remove all 1.0 tools from the MCP server, update tests, clean up dead code.

**Spec reference:** §25

**Dependencies:** All of Tracks A–J (the 2.0 tools must exist before 1.0 tools are removed).

| Task | Description | Size |
|------|-------------|------|
| K.1 | Remove the `_legacy` group from the group configuration — all 1.0 tool registrations stop loading | M |
| K.2 | Remove 1.0 tool handler functions that are completely superseded (no shared logic with 2.0 handlers) | M |
| K.3 | Remove `EntityTools`, `PlanTools`, `DocRecordTools`, `ConfigTools`, `QueryTools`, `MigrationTools` function groups from `server.go` | M |
| K.4 | Remove `KnowledgeTools`, `ProfileTools`, `ContextTools`, `AgentCapabilityTools`, `BatchImportTools` function groups | M |
| K.5 | Remove `WorktreeTools`, `BranchTools`, `CleanupTools`, `MergeTools`, `PRTools` function groups | S |
| K.6 | Remove `QueueTools`, `EstimationTools`, `DispatchTools`, `IncidentTools`, `DecomposeTools`, `ReviewTools`, `ConflictTools` function groups | S |
| K.7 | Update `TestServer_ListTools` to validate the 2.0 tool set (7–20 tools depending on group config) | M |
| K.8 | Verify no orphaned tool handler files remain in `internal/mcp/` | S |
| K.9 | Update CLI commands to match the `kbz <resource> <action>` pattern (spec §23) | L |
| K.10 | Run `go build ./...` — verify clean compilation | S |
| K.11 | Run `go test -race ./...` — verify all tests pass | S |
| K.12 | Run `go vet ./...` — verify clean | S |
| K.13 | Integration test: `next(task_id)` → `handoff(task_id)` → `finish(task_id)` end-to-end cycle | M |
| K.14 | Update `AGENTS.md` to reflect 2.0 tool surface | M |

**Key implementation notes:**

- K.1 is the big switch. Removing the `_legacy` group means no 1.0 tools are registered. Everything must work with only 2.0 tools. Run the full test suite after this step before proceeding.
- K.2–K.6 remove the dead code. Some 1.0 tool handler functions may share utility code with 2.0 handlers (e.g., parameter parsing helpers). Identify shared code and keep it; remove only dead paths.
- K.9 (CLI update) is a Large task because many CLI commands need updating. The CLI changes are cosmetic — the underlying service calls are unchanged — but there are many commands to touch.
- K.13 is the acceptance test: a real `next` → `handoff` → `finish` cycle on a real task, demonstrating the full 2.0 workflow. This is the spec's gate for completion.

**Verification (spec §30.11):** All 5 acceptance criteria must have passing tests.

---

## 15. Dependency Graph

```
                    ┌──────────────────────────────────┐
                    │  Track A: Feature Group Framework │
                    │  (start immediately)              │
                    └──────────────┬───────────────────┘
                                   │
                    ┌──────────────▼───────────────────┐
                    │  Track B: Resource-Oriented       │
                    │  Pattern + Side-Effect Reporting  │
                    └──┬───────────┬───────────────────┘
                       │           │
          ┌────────────▼──┐   ┌───▼──────────────────┐
          │  Track C:     │   │  Track D: status      │
          │  Batch Ops    │   │  (read-only, starts   │
          └──┬──┬─────┬──┘   │   after B)            │
             │  │     │      └────────────────────────┘
    ┌────────▼┐ │  ┌──▼───────────┐
    │ Track E │ │  │ Track H:     │
    │ finish  │ │  │ entity       │
    └────┬────┘ │  └──────┬───────┘
         │      │         │
    ┌────▼────┐ │  ┌──────▼───────┐
    │ Track F │ │  │ Track I:     │
    │ next    │ │  │ doc          │
    └────┬────┘ │  └──────────────┘
         │      │
    ┌────▼────┐ │
    │ Track G │ │
    │ handoff │ │
    └────┬────┘ │
         │      │
         └──────┤
                │
    ┌───────────▼──────────────────┐
    │  Track J: Feature Group Tools │
    │  (parallelisable internally) │
    └───────────┬──────────────────┘
                │
    ┌───────────▼──────────────────┐
    │  Track K: 1.0 Tool Removal   │
    └──────────────────────────────┘
```

---

## 16. Effort Estimates

### 16.1 Size definitions

Consistent with Phase 4a and 4b:

| Size | Effort | Description |
|------|--------|-------------|
| S | 1–2 hours | Simple, well-defined; follows existing patterns |
| M | 2–4 hours | Moderate complexity; some design or edge cases |
| L | 4–8 hours | Complex; multiple moving parts or careful integration |

### 16.2 Track estimates

| Track | Tasks | S | M | L | Est. Total |
|-------|-------|---|---|---|------------|
| A: Feature Group Framework | 12 | 6 | 5 | 0 | 16–26 hrs |
| B: Resource-Oriented Pattern | 18 | 8 | 9 | 0 | 26–40 hrs |
| C: Batch Operations | 12 | 5 | 6 | 0 | 17–26 hrs |
| D: `status` | 13 | 3 | 9 | 0 | 21–33 hrs |
| E: `finish` | 16 | 5 | 10 | 0 | 25–40 hrs |
| F: `next` | 28 | 6 | 18 | 2 | 48–80 hrs |
| G: `handoff` | 12 | 5 | 6 | 0 | 17–26 hrs |
| H: `entity` | 17 | 4 | 12 | 0 | 28–44 hrs |
| I: `doc` | 18 | 6 | 11 | 0 | 28–44 hrs |
| J: Feature Group Tools | 22 | 8 | 13 | 0 | 34–54 hrs |
| K: 1.0 Tool Removal | 14 | 5 | 7 | 1 | 23–38 hrs |
| **Total** | **182** | **61** | **106** | **3** | **283–451 hrs** |

### 16.3 Phase estimate

At 6–8 productive hours per day:

- **Single agent:** 35–75 days
- **Two agents:** 20–40 days
- **Three agents (max useful parallelism):** 14–28 days

The three-agent split:
- **Agent 1 (infrastructure + core workflow):** A → B → E → F → G → K
- **Agent 2 (CRUD consolidation):** wait for B+C → H → I → K assist
- **Agent 3 (dashboard + feature groups):** wait for A+B → D → wait for B → J → K assist

### 16.4 Comparison to prior phases

| Phase | Tasks | Est. Hours | Complexity |
|-------|-------|-----------|------------|
| Phase 4a | 68 | 93–146 | New orchestration model |
| Phase 4b | 86 | 116–182 | New capabilities on existing model |
| **Kanbanzai 2.0** | **182** | **283–451** | **Tool surface redesign over existing model** |

Kanbanzai 2.0 is roughly 2× the size of Phase 4b in task count and effort. However, most tasks (Tracks H, I, J) are mechanical consolidation — routing calls from the new action-parameter surface to existing service methods. The genuinely complex work is concentrated in Tracks B (side-effect collector), F (context assembly pipeline), and D (synthesis layer).

---

## 17. Implementation Order

### 17.1 Recommended sequence (single agent)

| Step | Track | Rationale |
|------|-------|-----------|
| 1 | A (Feature Group Framework) | Foundational infrastructure — everything depends on it |
| 2 | B (Resource-Oriented Pattern) | Action dispatch and side-effect collector — used by every 2.0 tool |
| 3 | C (Batch Operations) | Batch infrastructure — used by finish, entity, doc, estimate |
| 4 | D (`status`) | Read-only, low risk — builds confidence in the synthesis pattern before tackling workflow tools |
| 5 | E (`finish`) | Validates side-effect pipeline end-to-end; simpler than `next` |
| 6 | F (`next`) | Most complex tool — benefits from patterns established in D and E |
| 7 | G (`handoff`) | Shares pipeline with F; relatively quick after F is done |
| 8 | H (`entity`) | CRUD consolidation — mechanical but thorough |
| 9 | I (`doc`) | Document consolidation — similar pattern to H |
| 10 | J (Feature Group Tools) | Parallelisable consolidation pass |
| 11 | K (1.0 Tool Removal) | Final cutover — remove old tools, update tests |

### 17.2 Parallel execution (two agents)

**Agent 1: Infrastructure + core workflow (A → B → C → E → F → G)**

| Step | Track | Notes |
|------|-------|-------|
| 1 | A (Feature Group Framework) | Both agents blocked until A is done |
| 2 | B (Resource-Oriented Pattern) | Agent 2 blocked until B is done |
| 3 | C (Batch Operations) | Agent 2 can start H after this |
| 4 | E (`finish`) | Validates side-effect pipeline |
| 5 | F (`next`) | Most complex track |
| 6 | G (`handoff`) | Quick after F |

**Agent 2: Dashboard + CRUD + feature groups (D → H → I → J)**

| Step | Track | Notes |
|------|-------|-------|
| 1 | D (`status`) | Can start after A+B; doesn't need C |
| 2 | H (`entity`) | Can start after B+C |
| 3 | I (`doc`) | After H; same pattern |
| 4 | J (Feature Group Tools) | Parallelisable consolidation |

**Integration point:** Both agents converge on Track K after all other tracks are complete.

### 17.3 Parallel execution (three agents)

**Agent 1: Infrastructure (A → B → C) then core workflow (E → F → G)**
**Agent 2: CRUD consolidation (wait for B+C → H → I)**
**Agent 3: Dashboard + feature groups (wait for A+B → D → wait for B → J)**

All three converge on Track K.

**Critical path:** A → B → C → E → F → G → K (Agent 1's path). This is the longest dependency chain and determines the minimum timeline regardless of parallelism.

---

## 18. Verification Strategy

### 18.1 Test coverage requirements

| Category | Requirement |
|----------|-------------|
| Unit tests | All action dispatch routing, parameter validation, synthesis logic |
| Integration tests | End-to-end MCP tool calls for each 2.0 tool |
| Side-effect tests | Every mutation that cascades must be verified to produce the correct side effects |
| Batch tests | Single-item, multi-item, partial-failure, and limit-exceeded for every batch-capable tool |
| Round-trip tests | No new entity schemas — existing round-trip tests must continue to pass |
| Backward compatibility | During development, existing 1.0 tests pass with `_legacy` group enabled |
| Race detector | `go test -race ./...` at every track completion checkpoint |

### 18.2 Verification checkpoints

| Checkpoint | Tracks | What to verify |
|------------|--------|----------------|
| CP0 | Prerequisites | Phase 4b complete, codebase healthy, spec approved |
| CP1 | A | Feature group config parsed; tool count varies with config; 1.0 tests still pass |
| CP2 | B | Side-effect collector works; doc approval → feature transition side effect; action dispatch routes correctly |
| CP3 | C | Batch execution works; partial failure handled; single-item returns single-item shape |
| CP4 | D | `status()` returns project overview; `status(plan_id)` returns plan dashboard; attention items generated |
| CP5 | E | `finish(task_id)` completes task; lenient lifecycle works; inline knowledge contributed; unblocked tasks in side effects |
| CP6 | F | `next()` returns queue; `next(task_id)` claims and returns context with spec sections + knowledge; auto-parse triggered |
| CP7 | G | `handoff(task_id)` returns Markdown prompt; prompt contains all sections; context consistent with `next` |
| CP8 | H+I | `entity` and `doc` tools handle all actions; batch create/approve works; side effects reported |
| CP9 | J | All feature group tools respond to action dispatch; registered only when group enabled |
| CP10 | K | No 1.0 tools registered; `TestServer_ListTools` passes; end-to-end `next → handoff → finish` cycle works |

### 18.3 Acceptance criteria coverage

Each of the ~120 acceptance criteria from spec §30 must have at least one test. Test functions should reference the criterion section: `// Verifies §30.5: next() returns promoted tasks in side_effects`.

---

## 19. Risk Mitigations

### 19.1 Side-effect collector integration complexity

**Risk:** Wiring the collector into the MCP request lifecycle requires modifying the handler wrapping pattern. The `mcp-go` library may not provide clean middleware hooks.

**Mitigation:** If the library doesn't support middleware, wrap each tool handler function with a closure that creates the collector, attaches it to the context, calls the real handler, and drains the collector into the response. This is verbose but reliable. Prototype this in Track B.1–B.5 before committing to the pattern.

### 19.2 Context assembly pipeline complexity

**Risk:** Track F's context assembly is the most complex new code. It integrates document intelligence, knowledge retrieval, byte budgeting, and trimming into a single pipeline. Bugs here affect both `next` and `handoff`.

**Mitigation:** Build the pipeline incrementally:
1. First: assemble context with no intelligence (just task summary and raw spec path). This is the graceful degradation path and must work regardless.
2. Second: add spec section extraction.
3. Third: add knowledge retrieval.
4. Fourth: add trimming.
Each step has its own test. If intelligence integration proves brittle, the degraded path (step 1) is always available.

### 19.3 Dual registration period

**Risk:** During development, both 1.0 and 2.0 tools coexist. Tool name conflicts (e.g., if a 2.0 tool accidentally uses a 1.0 name) would cause registration failures.

**Mitigation:** All 2.0 tools use new names (`status`, `next`, `finish`, `handoff`, `entity`, `doc`, `health`, `decompose`, `estimate`, `conflict`, `knowledge`, `profile`, `worktree`, `merge`, `pr`, `branch`, `cleanup`, `doc_intel`, `incident`, `checkpoint`). None of these conflict with 1.0 names (which are all longer: `create_task`, `get_entity`, `doc_record_submit`, etc.). Verify at CP1 that no name conflicts exist.

### 19.4 `status` synthesis performance

**Risk:** The `status` tool aggregates data from many service calls (plans, features, tasks, documents, health). For large projects, this could be slow.

**Mitigation:** The synthesis layer should be lazy — only load what's needed for the requested scope. `status()` loads plans but not individual tasks. `status(plan_id)` loads features but only counts tasks (not full task records). `status(feature_id)` loads tasks for that feature only. If performance becomes a concern, cache synthesis results in the derived SQLite cache in a later phase.

### 19.5 Batch partial failure semantics

**Risk:** Best-effort batch semantics mean some items succeed and some fail in a single call. This can leave the system in a partially-updated state that surprises the caller.

**Mitigation:** The batch response shape (Track C) makes partial failure explicit: per-item `status` ("ok" or "error"), a `summary` with counts, and a clear explanation for each failure. The caller can inspect the response and decide whether to retry failed items. Document this behaviour clearly in tool descriptions.

### 19.6 1.0 removal breaking CLI

**Risk:** Track K removes 1.0 tools and updates the CLI. If CLI commands depend on 1.0 tool internals (shared handler functions, shared parameter parsing), removal may break the CLI.

**Mitigation:** During Tracks H, I, and J, verify that each 2.0 tool handler is self-contained and does not import from 1.0 handler files. When a 2.0 handler needs a utility function that lives in a 1.0 handler file, extract the utility into a shared package before Track K begins.

### 19.7 Scope creep

**Risk:** The 2.0 redesign scope is clearly defined (tool surface only, no storage changes), but during implementation there may be temptation to improve internal architecture, add new capabilities, or refactor service layers.

**Mitigation:** The spec's §3.3 (explicitly excluded) and §28 (no storage changes) are the guard rails. Any internal change must be justified as necessary for the tool surface redesign. If a service layer change would be nice-to-have, defer it to a later phase.

---

## 20. Definition of Done

A track is complete when:

1. All tasks in the track are implemented
2. All unit and integration tests pass
3. `go test -race ./...` passes
4. `go vet ./...` is clean
5. `go fmt ./...` applied
6. Relevant spec §30 acceptance criteria have passing tests with criterion references in test comments
7. During dual-registration period: existing 1.0 tests still pass

Kanbanzai 2.0 is complete when:

1. All 11 tracks are complete
2. All ~120 acceptance criteria (spec §30) verified by passing tests
3. No 1.0 tools remain registered in the MCP server
4. `TestServer_ListTools` validates the 2.0 tool set
5. End-to-end `next → handoff → finish` cycle demonstrated on a real task
6. CLI commands updated to `kbz <resource> <action>` pattern
7. `AGENTS.md` updated to reflect the 2.0 tool surface
8. `go test -race ./...` clean
9. `health_check` reports 0 errors on a clean project instance

---

## 21. Open Items

### 21.1 Questions to resolve during implementation

| Question | Disposition |
|----------|-------------|
| Side-effect collector: context key or explicit parameter? | Resolve in B.3; prefer context key for cleaner service signatures, but fall back to explicit parameter if context propagation is unreliable |
| `status` synthesis: how many attention items to show? | Cap at 5 per scope; pick the most actionable; resolve during D.6 |
| `next` automatic parse: synchronous or async? | Synchronous in 2.0 (simpler); consider async in a later phase if latency is a concern |
| `handoff` prompt template: hardcoded or configurable? | Hardcoded in 2.0; configurable templates are a future enhancement if users need different prompt formats |
| `entity(action: "list")` pagination: needed? | Not in 2.0 — return all matching entities. For large projects, revisit with cursor-based pagination |
| `_legacy` group: auto-disable after all 2.0 tools exist, or manual switch? | Manual switch — a human should explicitly confirm the cutover |
| CLI restructuring scope: full rewrite or incremental? | Incremental in K.9 — add new `kbz <resource> <action>` commands alongside existing commands, then remove old commands |

### 21.2 Potential 2.0.1 scope (defer if needed)

If any track falls behind estimates, these items can be deferred without breaking the core 2.0 value:

- `doc_intel` feature group (J.16–J.18) — intelligence powers core tools invisibly; the explicit exploration tool is convenience
- `decompose(action: "apply")` (J.2) — orchestrators can still create tasks individually via `entity(action: "create")`
- CLI update (K.9) — MCP tools are the primary interface; CLI can be updated incrementally
- `conflict_check` annotation in `next()` queue mode (F.4) — a nice-to-have, not essential for the core workflow
- `status` attention items (D.6) — status dashboards work without the curated attention list
- Batch `estimate(action: "set")` (J.4) — single-item estimation is sufficient

---

## 22. Summary

Kanbanzai 2.0 is implemented across 11 tracks and approximately 182 tasks.

**Key deliverables:**

| Deliverable | Tracks | Impact |
|-------------|--------|--------|
| Feature group framework | A | Context window: 7 tools (minimal) to 20 (full), down from 97 |
| Side-effect reporting | B | Agents learn what cascaded without re-querying |
| Batch operations | C | N-entity operations in one call instead of N calls |
| `status` dashboard | D | Situational awareness in one call instead of 5+ |
| `finish` with inline knowledge | E | Task completion + knowledge contribution in one call |
| `next` with invisible intelligence | F | Claim + full context in one call instead of 3–5 |
| `handoff` prompt generation | G | Ready-to-use sub-agent prompt in one call |
| Consolidated `entity` and `doc` | H, I | 17+ tools → 1, 11+ tools → 1 |
| Feature group consolidation | J | Remaining 1.0 tools consolidated into action-parameter forms |
| 1.0 removal | K | Clean break — 97 tools removed, 20 remain |

**Estimated effort:** 283–451 hours (14–28 days with three agents)

**Critical path:** A → B → C → E → F → G → K

**Gate for completion:** All spec §30 acceptance criteria verified, no 1.0 tools remaining, `go test -race ./...` clean, end-to-end `next → handoff → finish` cycle demonstrated.

**Implementation sequence:** A (framework) → B (patterns) → C (batch) → D (status) → E (finish) → F (next) → G (handoff) → H+I (entity, doc) → J (feature groups) → K (removal).