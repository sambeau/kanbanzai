# Implementation Plan: Binding Registry Gate Integration

**Feature:** FEAT-01KN5-8J27H83N (binding-registry-gate-integration)
**Specification:** `work/spec/3.0-binding-registry-gate-integration.md`
**Design reference:** `work/design/kanbanzai-3.0-workflow-and-tooling-v2.md` §3.4, §3.5, §16.1 Q6, §16.3 Q5

---

## Overview

This plan decomposes the binding registry gate integration specification into seven tasks. The feature replaces the hardcoded gate source in `internal/service/prereq.go` with a registry-driven approach: gate prerequisites are read from `.kbz/stage-bindings.yaml`, evaluated by a type-dispatched evaluator framework, and augmented with per-gate override policy (agent or checkpoint). A file-mtime-based cache ensures the registry is re-read only when it changes. When the registry is absent or malformed, the system falls back to the existing hardcoded gate definitions seamlessly.

### Scope boundaries (from specification)

- **In scope:** Registry-driven prerequisite lookup, prerequisite type evaluators (documents, tasks), mtime-based cache with hot-reload, hardcoded fallback, source indicator on gate results, override policy from registry, checkpoint policy integration, advance mode behaviour with checkpoint gates.
- **Out of scope:** Defining the binding registry schema/loader (FEAT-01KN5-88PDPE8V), binding registry content (P16), hardcoded gate enforcement mechanism itself (FEAT-01KN5-8J24S2XW), context assembly, action pattern logging, plan lifecycle gates.

### Dependencies

- **Binding registry (FEAT-01KN5-88PDPE8V):** Provides `internal/binding/registry.go` with `BindingRegistry`, `Lookup`, `StageBinding`, `Prerequisites`, `DocumentPrereq`, `TaskPrereq` types. This plan codes against those types. If the binding registry feature is not yet complete, tasks can be developed against the schema definition with hardcoded fallback as the primary path.
- **Mandatory stage gates (FEAT-01KN5-8J24S2XW):** Provides `CheckTransitionGate`, override mechanism (`override`/`override_reason` parameters), `OverrideRecord` on feature entities, and `GateFailureResponse`. This feature replaces the gate *source* but not the gate *enforcement* mechanism.
- **Checkpoint system (Phase 4a):** Provides `internal/checkpoint/Store` with `Create`, `Get`, `Update`, `List` methods.

---

## Task Breakdown

### Task 1: Registry cache with mtime-based invalidation

**Objective:** Create a concurrency-safe, mtime-based cache that loads and parses the binding registry file on first access and re-reads it only when the file's mtime changes. The cache must handle missing files (empty cache, triggers fallback), malformed files (log warning, trigger fallback), and file deletion after startup (clear cache, trigger fallback). This is the foundational component that all other tasks depend on for registry access.

**Specification references:** FR-005, FR-006, FR-007, NFR-001, NFR-002

**Input context:**
- `internal/binding/registry.go` — `BindingRegistry` type with `Load`, `Lookup` methods (from binding registry feature)
- `internal/binding/model.go` — `BindingFile`, `StageBinding`, `Prerequisites` types
- `internal/core/paths.go` — `InstanceRootDir` constant (`.kbz`)
- `internal/config/config.go` — pattern for loading configuration from `.kbz/`

**Output artifacts:**
- New file `internal/gate/registry_cache.go`:
  - `RegistryCache` struct with `sync.RWMutex`, cached `*binding.BindingFile`, cached mtime, file path
  - `NewRegistryCache(path string) *RegistryCache`
  - `Get() (*binding.BindingFile, error)` — stats the file, compares mtime, re-reads if changed, returns cached result or nil (for fallback)
  - `LookupPrereqs(stage string) (*binding.Prerequisites, bool)` — convenience method that calls `Get` and looks up the stage's prerequisites block
  - Internal `refresh()` method that re-reads and re-parses the file under a write lock
  - Missing file → returns nil BindingFile (no error, triggers fallback in consumers)
  - Malformed file → logs warning, returns nil BindingFile (triggers fallback), does NOT serve stale cache
  - Concurrent callers during refresh either wait for the refresh or use the previous cached version (no partial reads)
- New file `internal/gate/registry_cache_test.go`:
  - Test: fresh cache with valid file returns parsed bindings
  - Test: fresh cache with missing file returns nil (no error)
  - Test: fresh cache with malformed file returns nil + logs warning
  - Test: mtime unchanged between calls → no re-read (verify via call count or file access tracking)
  - Test: mtime changed → re-read picks up new content
  - Test: file deleted after startup → returns nil (fallback)
  - Test: file replaced with malformed version → returns nil (not stale cache)
  - Test: concurrent access with race detector (`go test -race`)
  - Benchmark: mtime stat overhead < 1ms (NFR-001)
  - Benchmark: re-parse of 15-binding file < 200ms (NFR-002)

**Dependencies:** None — this is the foundational task. Depends on the binding registry feature's model types existing, but can use a minimal subset or define a local interface if needed.

**Interface contract (shared with Tasks 2, 3, 4, 5, 6):**

```go
// RegistryCache provides mtime-based caching of the binding registry file.
// Safe for concurrent use.
type RegistryCache struct { /* unexported fields */ }

// NewRegistryCache creates a cache for the binding registry at the given path.
// The file is not read until the first call to Get or LookupPrereqs.
func NewRegistryCache(path string) *RegistryCache

// Get returns the cached binding file, re-reading if the file's mtime has changed.
// Returns (nil, nil) if the file does not exist or is malformed (caller should fall back).
func (c *RegistryCache) Get() (*binding.BindingFile, error)

// LookupPrereqs returns the prerequisites for the given stage from the cached registry.
// Returns (nil, false) if the registry is unavailable or the stage has no prerequisites block.
// Returns (prereqs, true) if the registry provides prerequisites for the stage.
func (c *RegistryCache) LookupPrereqs(stage string) (*binding.Prerequisites, bool)

// LookupOverridePolicy returns the override policy for the given stage from the cached registry.
// Returns ("agent", false) if the registry is unavailable or the stage has no override_policy.
// Returns (policy, true) if the registry provides an override policy for the stage.
func (c *RegistryCache) LookupOverridePolicy(stage string) (string, bool)
```

---

### Task 2: Prerequisite evaluator framework with type dispatch

**Objective:** Create the evaluator framework that dispatches prerequisite evaluation by type key. Each prerequisite type (documents, tasks) is handled by a registered evaluator function. Unknown type keys cause the gate to fail with an actionable error naming the unrecognised type and stage. The framework must be extensible — adding a new prerequisite type requires only registering a new evaluator function, with no changes to dispatch logic.

**Specification references:** FR-004, NFR-003

**Input context:**
- `internal/service/prereq.go` — current `GateResult` struct, `CheckFeatureGate` pattern
- `internal/binding/model.go` — `Prerequisites` struct with `Documents` and `Tasks` fields
- `internal/model/entities.go` — `Feature` struct

**Output artifacts:**
- New file `internal/gate/evaluator.go`:
  - `PrereqEvalContext` struct: feature, docSvc, entitySvc (everything an evaluator might need)
  - `PrereqEvaluator` function type: `func(ctx PrereqEvalContext) GateResult`
  - `EvalRegistry` map of type key → evaluator function (package-level, populated at init or via registration)
  - `RegisterEvaluator(typeKey string, fn PrereqEvaluator)` — registers an evaluator for a type key
  - `EvaluatePrerequisites(prereqs *binding.Prerequisites, stage string, ctx PrereqEvalContext) []GateResult` — iterates the prerequisites struct, dispatches to registered evaluators by type key, fails with an error naming the unknown type if no evaluator is registered
  - The function inspects the `Prerequisites` struct fields (Documents, Tasks, and any future fields via reflection or explicit field iteration) and dispatches each non-nil field to its registered evaluator
- New file `internal/gate/evaluator_test.go`:
  - Test: prerequisites with documents dispatches to documents evaluator
  - Test: prerequisites with tasks dispatches to tasks evaluator
  - Test: prerequisites with both documents and tasks evaluates both
  - Test: empty prerequisites returns no gate failures
  - Test: unknown prerequisite type key returns error with stage name and type key in message
  - Test: registering a custom evaluator and verifying it is called
  - Test: nil prerequisites returns empty results

**Dependencies:** Task 1 (for `RegistryCache` types, though the evaluator framework is independent of the cache — it operates on parsed `Prerequisites` values)

**Interface contract (shared with Tasks 3, 5):**

```go
// PrereqEvalContext holds everything an evaluator needs to check prerequisites.
type PrereqEvalContext struct {
    Feature   *model.Feature
    DocSvc    *service.DocumentService
    EntitySvc *service.EntityService
}

// EvaluatePrerequisites evaluates all prerequisites in the block and returns
// a GateResult for each. An unsatisfied prerequisite produces a GateResult
// with Satisfied=false. An unknown prerequisite type produces a GateResult
// with Satisfied=false and an error-identifying Reason.
func EvaluatePrerequisites(prereqs *binding.Prerequisites, stage string, ctx PrereqEvalContext) []GateResult
```

---

### Task 3: Document and task prerequisite evaluators

**Objective:** Implement the `documents` and `tasks` prerequisite evaluators and register them with the evaluator framework. The document evaluator reuses the existing three-level lookup order (feature field reference → feature-owned documents → parent-plan-owned documents). The task evaluator supports `min_count` and `all_terminal` modes. Both produce `GateResult` values consistent with the existing hardcoded gate output.

**Specification references:** FR-002, FR-003

**Input context:**
- `internal/service/prereq.go` — `checkDocumentGate` (three-level lookup order), `checkDevelopingGate` (task count check), `featureDocRef`, `stageDocField`
- `internal/validate/lifecycle.go` — `DependencyTerminalStates()`, `IsTaskDependencySatisfied()`
- `internal/service/entities.go` — `EntityService.List` for task enumeration
- `internal/service/documents.go` — `DocumentService.GetDocument`, `DocumentService.ListDocuments`, `DocumentFilters`

**Output artifacts:**
- New file `internal/gate/eval_documents.go`:
  - `evalDocuments(prereqs []binding.DocumentPrereq, stage string, ctx PrereqEvalContext) []GateResult` — for each document prerequisite, checks the three-level lookup order for a document of the given type with the given status
  - Registers itself with `RegisterEvaluator("documents", ...)` via an `init()` function or explicit registration call
  - Handles unknown document types gracefully — evaluates against the document service without error (FR-002 AC)
  - Multiple document prerequisites in a single stage require ALL to be satisfied
- New file `internal/gate/eval_tasks.go`:
  - `evalTasks(prereq *binding.TaskPrereq, stage string, ctx PrereqEvalContext) GateResult` — dispatches to `min_count` or `all_terminal` evaluation
  - `min_count`: counts child tasks of the feature, returns satisfied if count >= min_count
  - `all_terminal`: iterates child tasks, checks each against `validate.IsTaskDependencySatisfied`, returns satisfied only if all are terminal
  - Registers itself with `RegisterEvaluator("tasks", ...)`
- New file `internal/gate/eval_documents_test.go`:
  - Test: single document prerequisite satisfied by feature field reference
  - Test: single document prerequisite satisfied by feature-owned document
  - Test: single document prerequisite satisfied by parent-plan-owned document
  - Test: document prerequisite not satisfied (draft only, no approved)
  - Test: multiple document prerequisites — all must be satisfied
  - Test: unknown document type evaluated without error
- New file `internal/gate/eval_tasks_test.go`:
  - Test: `min_count: 1` satisfied with one task
  - Test: `min_count: 1` not satisfied with zero tasks
  - Test: `min_count: 3` not satisfied with two tasks
  - Test: `all_terminal: true` satisfied when all tasks done/not-planned/duplicate
  - Test: `all_terminal: true` not satisfied with active task
  - Test: `all_terminal: true` with no tasks (vacuously true — matches current behaviour)

**Dependencies:** Task 2 (evaluator framework and registration mechanism)

**Interface contract:** The evaluators are internal to the `gate` package and registered via the framework from Task 2. No external contract beyond the `EvaluatePrerequisites` function from Task 2.

---

### Task 4: Gate source router with hardcoded fallback

**Objective:** Create the routing layer that decides whether to use registry-sourced or hardcoded gate definitions for a given transition. The router first consults the registry cache; if the registry provides prerequisites for the target stage, those are evaluated via the evaluator framework. If the registry is unavailable or has no prerequisites block for the stage, the router delegates to the existing hardcoded `CheckTransitionGate` function. The gate result includes a source indicator (registry or hardcoded) for health reporting, but this indicator is NOT exposed in error messages to agents.

**Specification references:** FR-001, FR-008, FR-009, FR-015, FR-016, NFR-003, NFR-004

**Input context:**
- `internal/service/prereq.go` — `CheckFeatureGate`, `CheckTransitionGate` (from mandatory-stage-gates feature), `GateResult` struct
- `internal/gate/registry_cache.go` — `RegistryCache.LookupPrereqs` (from Task 1)
- `internal/gate/evaluator.go` — `EvaluatePrerequisites` (from Task 2)
- `internal/service/gate_errors.go` — `GateFailureResponse` (from mandatory-stage-gates feature)

**Output artifacts:**
- New file `internal/gate/router.go`:
  - Extended `GateResult` struct (or wrapper) with a `Source` field: `"registry"` or `"hardcoded"`
  - `GateRouter` struct holding a `*RegistryCache` and references to the hardcoded gate function
  - `NewGateRouter(cache *RegistryCache) *GateRouter`
  - `CheckGate(from, to string, feature *model.Feature, docSvc *DocumentService, entitySvc *EntityService) GateResult` — the single entry point for all gate checks:
    1. Attempt registry lookup for the target stage (`to`)
    2. If registry provides prerequisites: evaluate via `EvaluatePrerequisites`, combine results, set `Source: "registry"`
    3. If registry unavailable or no prerequisites for stage: delegate to hardcoded `CheckTransitionGate`, set `Source: "hardcoded"`
    4. Gate failure messages follow the existing template (what failed, why, what to do) — no mention of registry or hardcoded source
  - `OverridePolicy(to string) string` — returns the override policy for the target stage from registry, defaulting to `"agent"` when registry is unavailable or field is absent
- New file `internal/gate/router_test.go`:
  - Test: registry provides prerequisites → evaluator used, source is "registry"
  - Test: registry unavailable → hardcoded fallback used, source is "hardcoded"
  - Test: registry present but no prerequisites for stage → hardcoded fallback used
  - Test: registry provides prerequisites for stage A but not B → A uses registry, B uses hardcoded
  - Test: gate failure messages from registry path match the actionable template (no "binding registry" or "hardcoded" text)
  - Test: gate failure messages from hardcoded path produce equivalent output
  - Test: `OverridePolicy` returns "agent" when registry is unavailable
  - Test: `OverridePolicy` returns "checkpoint" when registry specifies it
  - Test: `OverridePolicy` returns "agent" when registry has no override_policy field

**Dependencies:** Task 1 (registry cache), Task 2 (evaluator framework), Task 3 (evaluators must be registered)

**Interface contract (shared with Tasks 5, 6, 7):**

```go
// GateRouter routes gate evaluation to the registry or hardcoded fallback.
type GateRouter struct { /* unexported fields */ }

// NewGateRouter creates a router backed by the given registry cache.
// If cache is nil, all gates use the hardcoded fallback.
func NewGateRouter(cache *RegistryCache) *GateRouter

// CheckGate evaluates the gate for the given transition.
// Returns a GateResult with a Source field indicating "registry" or "hardcoded".
func (r *GateRouter) CheckGate(from, to string, feature *model.Feature, docSvc *service.DocumentService, entitySvc *service.EntityService) GateResult

// OverridePolicy returns the override policy for the target stage.
// Returns "agent" if the registry is unavailable or the field is absent.
func (r *GateRouter) OverridePolicy(to string) string
```

---

### Task 5: Checkpoint override policy integration

**Objective:** Implement the `checkpoint` override policy behaviour. When a gate with `checkpoint` override policy is overridden, the transition handler creates a human checkpoint instead of proceeding immediately. The feature blocks until the checkpoint is responded to. Approval completes the transition; rejection leaves the feature at its current status. Integrate with the existing `internal/checkpoint/Store`.

**Specification references:** FR-010, FR-011, FR-012, FR-013, NFR-005

**Input context:**
- `internal/checkpoint/checkpoint.go` — `Store`, `Record`, `StatusPending`, `StatusResponded`, `Create`, `Get`, `Update` methods
- `internal/gate/router.go` — `GateRouter.OverridePolicy` (from Task 4)
- `internal/model/entities.go` — `Feature`, `OverrideRecord` (from mandatory-stage-gates feature)

**Output artifacts:**
- New file `internal/gate/checkpoint_override.go`:
  - `CheckpointOverrideResult` struct: `CheckpointCreated bool`, `CheckpointID string`, `Message string`, `Rejected bool`
  - `HandleCheckpointOverride(params CheckpointOverrideParams) (CheckpointOverrideResult, error)` where params include: feature ID, from-status, to-status, gate result description, override reason, agent identity, checkpoint store
  - Creates a checkpoint with a question containing: feature ID, from→to transition, failing prerequisite description, agent's override reason
  - Returns a result indicating the checkpoint was created, with the checkpoint ID
  - `ResolveCheckpointResponse(response string) bool` — determines approval vs rejection using keyword matching: responses containing "reject", "denied", or "no" (case-insensitive, whole word) are rejections; all other non-empty responses are approvals
  - `CompleteCheckpointOverride(checkpointID string, feature *model.Feature, toStatus string, checkpointStore *checkpoint.Store, entitySvc *EntityService) error` — called when a checkpoint is responded to; reads the response, determines approval/rejection, transitions the feature or records rejection
- New file `internal/gate/checkpoint_override_test.go`:
  - Test: checkpoint override creates a pending checkpoint record
  - Test: checkpoint question contains feature ID, transition, prerequisite, and reason
  - Test: response "approved" → approval
  - Test: response "yes" → approval
  - Test: response "looks good to me" → approval
  - Test: response "rejected" → rejection
  - Test: response "no" → rejection
  - Test: response "denied" → rejection
  - Test: response "No, this needs more work" → rejection (contains "no" as whole word)
  - Test: response "I do not agree" → rejection (contains "no" as part of "not"? — verify whole-word semantics)
  - Test: checkpoint override result includes checkpoint ID
  - Test: approval completes the transition
  - Test: rejection leaves feature at current status
  - Test: override record on feature includes checkpoint ID

**Dependencies:** Task 4 (gate router provides override policy)

**Interface contract (shared with Task 6):**

```go
// CheckpointOverrideParams contains everything needed to create a checkpoint override.
type CheckpointOverrideParams struct {
    FeatureID       string
    FromStatus      string
    ToStatus        string
    GateDescription string
    OverrideReason  string
    AgentIdentity   string
    CheckpointStore *checkpoint.Store
}

// CheckpointOverrideResult describes the outcome of a checkpoint override attempt.
type CheckpointOverrideResult struct {
    CheckpointCreated bool
    CheckpointID      string
    Message           string
}

// HandleCheckpointOverride creates a checkpoint for a gate override with checkpoint policy.
func HandleCheckpointOverride(params CheckpointOverrideParams) (CheckpointOverrideResult, error)

// ResolveCheckpointResponse determines whether a checkpoint response is an approval or rejection.
// Returns true for approval, false for rejection.
func ResolveCheckpointResponse(response string) bool
```

---

### Task 6: Wire gate router into transition handler and advance mode

**Objective:** Replace the hardcoded `CheckTransitionGate` / `CheckFeatureGate` call sites in the MCP entity transition handler and `AdvanceFeatureStatus` with the gate router. Single-step transitions and advance mode both use the same `GateRouter.CheckGate` path. Override handling branches on the policy: `agent` proceeds immediately (existing behaviour), `checkpoint` creates a checkpoint and returns without transitioning. Advance mode halts at the first checkpoint gate. No new parameters are added to the `entity` tool.

**Specification references:** FR-014, FR-015, NFR-003, NFR-004

**Input context:**
- `internal/mcp/entity_tool.go` — `entityTransitionAction`, `entityAdvanceFeature` (current wiring)
- `internal/service/advance.go` — `AdvanceFeatureStatus` (current advance logic)
- `internal/service/prereq.go` — `CheckFeatureGate`, `CheckTransitionGate` (call sites to replace)
- `internal/gate/router.go` — `GateRouter` (from Task 4)
- `internal/gate/checkpoint_override.go` — `HandleCheckpointOverride` (from Task 5)
- `internal/mcp/server.go` — MCP server setup (where `RegistryCache` and `GateRouter` will be constructed)

**Output artifacts:**
- Modified `internal/mcp/server.go`:
  - Construct `RegistryCache` with path `.kbz/stage-bindings.yaml`
  - Construct `GateRouter` with the cache
  - Pass `GateRouter` (and checkpoint store) to `entityTransitionAction` and `entityAdvanceFeature`
- Modified `internal/mcp/entity_tool.go`:
  - `entityTransitionAction`: replace `CheckTransitionGate` call with `GateRouter.CheckGate`. After gate check, call `GateRouter.OverridePolicy` to determine override behaviour. If policy is `agent` and override is true, proceed as before. If policy is `checkpoint` and override is true, call `HandleCheckpointOverride` and return the checkpoint response without transitioning.
  - `entityAdvanceFeature`: pass `GateRouter` through to `AdvanceFeatureStatus`
- Modified `internal/service/advance.go`:
  - Update `AdvanceFeatureStatus` signature to accept a gate checking function (or the `GateRouter` directly) instead of directly calling `CheckFeatureGate`
  - At each step: use the provided gate checker. If override is true and policy is `agent`, override and continue. If override is true and policy is `checkpoint`, create checkpoint and halt — return partial result indicating which gates were overridden via `agent` and which gate created a checkpoint.
  - `AdvanceResult` gains a `CheckpointGate` field (string, stage name where checkpoint was created) and `CheckpointID` field
- Modified `internal/service/advance_test.go`:
  - Test: advance with all-agent-policy gates and override overrides all gates
  - Test: advance with mixed policies halts at first checkpoint gate
  - Test: advance response includes which gates were agent-overridden and which created a checkpoint
  - Test: after checkpoint resolved, new advance call continues from the halted point
- New/modified integration test in `internal/mcp/entity_tool_test.go` or equivalent:
  - Test: single-step transition uses registry gate when registry provides prerequisites
  - Test: single-step transition falls back to hardcoded when registry is absent
  - Test: override with agent policy proceeds immediately
  - Test: override with checkpoint policy returns checkpoint info
  - Test: no new parameters on entity tool schema (NFR-004)

**Dependencies:** Task 4 (gate router), Task 5 (checkpoint override)

**Interface contract:** The `AdvanceFeatureStatus` function signature changes to accept a gate evaluation dependency:

```go
// GateChecker is the function signature for gate evaluation, abstracting
// over the GateRouter to keep the service layer free of direct gate package imports.
type GateChecker func(from, to string, feature *model.Feature, docSvc *DocumentService, entitySvc *EntityService) GateResult

// OverridePolicyChecker returns the override policy for a target stage.
type OverridePolicyChecker func(to string) string

func AdvanceFeatureStatus(
    feature *model.Feature,
    targetStatus string,
    entitySvc *EntityService,
    docSvc *DocumentService,
    override bool,
    overrideReason string,
    checkGate GateChecker,
    overridePolicy OverridePolicyChecker,
    checkpointStore *checkpoint.Store,
) (AdvanceResult, error)
```

---

### Task 7: Health reporting and end-to-end integration tests

**Objective:** Add health check reporting for gate source indicators (which gates are using registry vs hardcoded), ensure the `health` tool surfaces override warnings including checkpoint overrides, and write end-to-end integration tests that exercise the complete pipeline from registry file through gate evaluation and checkpoint creation.

**Specification references:** FR-009 (source indicator in health), FR-011 (health flagging), FR-016 (error message consistency), acceptance criteria 1–8

**Input context:**
- `internal/mcp/health_tool.go` — existing health check implementation
- `internal/gate/router.go` — `GateRouter` with source indicators (from Task 4)
- All specification acceptance criteria (§Acceptance Criteria)

**Output artifacts:**
- Modified health check (location depends on current implementation):
  - Report which stages are using registry gates vs hardcoded gates as informational items
  - Report checkpoint overrides (pending checkpoints created by gate overrides) as warnings
- New file `internal/gate/integration_test.go`:
  - End-to-end test: registry-sourced gate blocks transition when prerequisite not met
  - End-to-end test: registry-sourced gate allows transition when prerequisite met
  - End-to-end test: hot-reload — edit registry file, next transition uses updated prerequisites
  - End-to-end test: delete registry file → fallback produces identical results to hardcoded
  - End-to-end test: agent override policy → immediate override
  - End-to-end test: checkpoint override policy → checkpoint created, feature blocks
  - End-to-end test: checkpoint approval → transition completes
  - End-to-end test: checkpoint rejection → feature stays
  - End-to-end test: advance with mixed policies halts at checkpoint gate
  - End-to-end test: extensibility — register custom evaluator, add custom prerequisite type to registry, verify it is enforced
  - End-to-end test: all existing `CheckFeatureGate` and `AdvanceFeatureStatus` tests continue to pass (regression)

**Dependencies:** Tasks 1–6 (needs the full pipeline assembled)

---

## Dependency Graph

```
Task 1: Registry cache (mtime-based)
  │
  └──► Task 2: Evaluator framework (type dispatch)
         │
         └──► Task 3: Document + task evaluators
                │
                └──► Task 4: Gate source router (registry vs hardcoded fallback)
                       │
                       ├──► Task 5: Checkpoint override policy
                       │       │
                       │       └──► Task 6: Wire into transition handler + advance
                       │               │
                       └───────────────┴──► Task 7: Health reporting + integration tests
```

**Parallelism opportunities:**

- Task 1 is the serial entry point — it must complete first.
- Tasks 2 and 3 are logically sequential (3 depends on 2's framework), but Task 2 is small enough that they could be a single unit of work.
- **Tasks 2 and 3 can be developed in parallel with Task 1** if coding against the binding model types directly (the dependency is on the `Prerequisites` struct, not the cache).
- Task 4 requires Tasks 1, 2, and 3 — it's the integration point.
- Task 5 can be developed in parallel with Task 4 since it only needs the `OverridePolicy` concept (a string), not the full router implementation. However, wiring it requires Task 4.
- **Tasks 4 and 5 can execute in parallel** — Task 5 only needs to know that the router will return a policy string.
- Task 6 requires Tasks 4 and 5 — it wires everything together.
- Task 7 requires Task 6 — it validates the full pipeline.

**Recommended execution order:** 1 → (2 → 3) → (4 ∥ 5) → 6 → 7

Alternatively, with maximum parallelism: (1 ∥ 2 → 3) → (4 ∥ 5) → 6 → 7

---

## Interface Contracts

### Contract A: Registry cache API (Task 1 → Tasks 2, 4, 6)

The `RegistryCache` type provides mtime-cached access to the binding registry file. The `LookupPrereqs` and `LookupOverridePolicy` methods are the primary consumers for the gate router. The struct definitions and method signatures in the Task 1 interface contract section are authoritative.

### Contract B: Evaluator framework (Task 2 → Tasks 3, 4)

The `EvaluatePrerequisites` function and `PrereqEvalContext` struct are the dispatch layer between the router and the individual evaluators. The function signature in the Task 2 interface contract section is authoritative.

### Contract C: Gate router API (Task 4 → Tasks 5, 6, 7)

The `GateRouter` with `CheckGate` and `OverridePolicy` methods is the primary API consumed by the transition handler and advance logic. The `GateResult` struct is extended with a `Source` field. The signatures in the Task 4 interface contract section are authoritative.

### Contract D: Checkpoint override API (Task 5 → Task 6)

The `HandleCheckpointOverride` function and `CheckpointOverrideResult` struct are the interface between the transition handler and the checkpoint system. The signatures in the Task 5 interface contract section are authoritative.

### Contract E: Gate checker function types (Task 6 → service layer)

The `GateChecker` and `OverridePolicyChecker` function types abstract the gate router for the service layer, avoiding a direct dependency from `internal/service` to `internal/gate`. These are defined in `internal/service/advance.go` and injected by the MCP layer. The signatures in the Task 6 interface contract section are authoritative.

### Cross-feature contract: GateResult

The `service.GateResult` struct (`Stage`, `Satisfied`, `Reason`) is the existing return type from `CheckFeatureGate`. The gate router's result must be compatible — it extends with a `Source` field but preserves the existing fields. The `merge.GateResult` struct (used by merge gates) is a separate type and is not modified by this feature.

### Cross-feature contract: OverrideRecord

The `model.OverrideRecord` struct (from mandatory-stage-gates) is extended with an optional `CheckpointID` field to record which checkpoint was created for a checkpoint-policy override. The field is empty for agent-policy overrides.

### Cross-feature contract: Prerequisites schema

The `binding.Prerequisites` struct (from the binding registry feature) must include an `OverridePolicy` field (`string`, optional, defaults to `"agent"`). This requires coordination with the binding registry feature. If the field is not yet on the struct, this feature adds it:

```go
// Prerequisites declares what must be true before entering the stage.
type Prerequisites struct {
    Documents      []DocumentPrereq `yaml:"documents,omitempty"`
    Tasks          *TaskPrereq      `yaml:"tasks,omitempty"`
    OverridePolicy string           `yaml:"override_policy,omitempty"` // "agent" (default) or "checkpoint"
}
```

---

## Traceability Matrix

| Requirement | Task(s) | Notes |
|-------------|---------|-------|
| FR-001 (Registry-driven lookup) | Task 1, Task 4 | Cache provides access; router dispatches to registry or fallback |
| FR-002 (Documents evaluator) | Task 3 | Three-level lookup order preserved |
| FR-003 (Tasks evaluator) | Task 3 | min_count and all_terminal modes |
| FR-004 (Extensible evaluator framework) | Task 2 | Type-dispatched registration |
| FR-005 (Startup cache load) | Task 1 | Missing file → empty cache |
| FR-006 (Mtime-based invalidation) | Task 1 | Stat on each tool call |
| FR-007 (Concurrent cache safety) | Task 1 | sync.RWMutex, race detector tests |
| FR-008 (Hardcoded fallback) | Task 4 | Router delegates to CheckTransitionGate |
| FR-009 (Source indicator) | Task 4, Task 7 | Source field on GateResult; health reporting |
| FR-010 (Override policy from registry) | Task 4, Task 5 | Router reads policy; checkpoint module handles checkpoint |
| FR-011 (Agent override behaviour) | Task 6 | Existing mechanism, no change |
| FR-012 (Checkpoint override creation) | Task 5 | Creates checkpoint, blocks feature |
| FR-013 (Checkpoint response handling) | Task 5 | Keyword-based approval/rejection |
| FR-014 (Advance with checkpoint policy) | Task 6 | Halts at first checkpoint gate |
| FR-015 (Integration with transition handler) | Task 6 | Single evaluation path for both modes |
| FR-016 (Actionable error messages) | Task 4, Task 6 | No registry/hardcoded leakage in messages |
| NFR-001 (Stat latency < 1ms) | Task 1 | Benchmark test |
| NFR-002 (Re-parse < 200ms) | Task 1 | Benchmark test |
| NFR-003 (Backward compatibility) | Task 4, Task 6 | GateResult preserved; callers updated minimally |
| NFR-004 (Invisible to agents) | Task 6 | No new entity tool parameters |
| NFR-005 (Checkpoint infrastructure reuse) | Task 5 | Uses existing Store.Create/Get/Update |